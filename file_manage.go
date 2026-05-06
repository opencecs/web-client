package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileManageHandler 文件管理处理器（上传到程序运行目录下的 mmc/）
type FileManageHandler struct {
	auth *AuthService
}

// fileInfo 文件信息
type fileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	IsDir   bool   `json:"isDir"`
}

// uploadDir 返回上传目录的绝对路径（mmc/upload/）
func uploadDir() string {
	exePath, _ := os.Executable()
	base := filepath.Dir(exePath)
	// 优先使用工作目录
	if wd, err := os.Getwd(); err == nil {
		base = wd
	}
	return filepath.Join(base, "mmc", "upload")
}

// HandleUpload 上传文件到 mmc 目录（流式写入，支持大文件）
func (h *FileManageHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 权限检查：admin 或 backup_manage 权限
	claims := r.Context().Value(userContextKey).(*Claims)
	if claims.Role != "admin" {
		perms := h.auth.GetUserPermissions(claims.UserID)
		if perms == nil || !perms.BackupManage {
			jsonError(w, "无权限", 403)
			return
		}
	}

	// 流式读取 multipart，不缓存整个文件到内存
	reader, err := r.MultipartReader()
	if err != nil {
		jsonError(w, "文件解析失败: "+err.Error(), 400)
		return
	}

	// 找到 file 字段
	var part *multipart.Part
	for {
		p, err := reader.NextPart()
		if err != nil {
			jsonError(w, "缺少文件", 400)
			return
		}
		if p.FormName() == "file" {
			part = p
			break
		}
		p.Close()
	}
	defer part.Close()

	filename := part.FileName()
	if filename == "" {
		jsonError(w, "文件名为空", 400)
		return
	}
	// 安全检查：禁止路径穿越
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		jsonError(w, "文件名不合法", 400)
		return
	}

	dir := uploadDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		jsonError(w, "创建目录失败: "+err.Error(), 500)
		return
	}

	dst := filepath.Join(dir, filename)
	f, err := os.Create(dst)
	if err != nil {
		jsonError(w, "创建文件失败: "+err.Error(), 500)
		return
	}
	defer f.Close()

	written, err := io.Copy(f, part)
	if err != nil {
		f.Close()
		os.Remove(dst)
		jsonError(w, "写入文件失败: "+err.Error(), 500)
		return
	}

	log.Printf("[FileManage] 上传文件: %s (%d bytes)", filename, written)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "上传成功",
		"data": map[string]interface{}{
			"name": filename,
			"size": written,
		},
	})
}

// HandleList 列出 mmc 目录下的文件
func (h *FileManageHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(userContextKey).(*Claims)
	if claims.Role != "admin" {
		perms := h.auth.GetUserPermissions(claims.UserID)
		if perms == nil || !perms.BackupManage {
			jsonError(w, "无权限", 403)
			return
		}
	}

	dir := uploadDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": []interface{}{}})
			return
		}
		jsonError(w, "读取目录失败: "+err.Error(), 500)
		return
	}

	var files []fileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   entry.IsDir(),
		})
	}

	// 按修改时间倒序
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime > files[j].ModTime
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    files,
	})
}

// HandleDelete 删除 mmc 目录下的文件
func (h *FileManageHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(userContextKey).(*Claims)
	if claims.Role != "admin" {
		perms := h.auth.GetUserPermissions(claims.UserID)
		if perms == nil || !perms.BackupManage {
			jsonError(w, "无权限", 403)
			return
		}
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		jsonError(w, "缺少文件名", 400)
		return
	}
	// 安全检查
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		jsonError(w, "文件名不合法", 400)
		return
	}

	dir := uploadDir()
	path := filepath.Join(dir, name)

	// 确保文件在 mmc 目录内
	absPath, _ := filepath.Abs(path)
	absDir, _ := filepath.Abs(dir)
	if !strings.HasPrefix(absPath, absDir) {
		jsonError(w, "非法路径", 400)
		return
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			jsonError(w, "文件不存在", 404)
		} else {
			jsonError(w, "删除失败: "+err.Error(), 500)
		}
		return
	}

	log.Printf("[FileManage] 删除文件: %s", name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "删除成功",
	})
}

// HandleDownload 下载 mmc 目录下的文件
func (h *FileManageHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(userContextKey).(*Claims)
	if claims.Role != "admin" {
		perms := h.auth.GetUserPermissions(claims.UserID)
		if perms == nil || !perms.BackupManage {
			jsonError(w, "无权限", 403)
			return
		}
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		jsonError(w, "缺少文件名", 400)
		return
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		jsonError(w, "文件名不合法", 400)
		return
	}

	dir := uploadDir()
	path := filepath.Join(dir, name)

	absPath, _ := filepath.Abs(path)
	absDir, _ := filepath.Abs(dir)
	if !strings.HasPrefix(absPath, absDir) {
		jsonError(w, "非法路径", 400)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			jsonError(w, "文件不存在", 404)
		} else {
			jsonError(w, "打开文件失败", 500)
		}
		return
	}
	defer f.Close()

	stat, _ := f.Stat()
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	http.ServeContent(w, r, name, stat.ModTime(), f)
}

// formatFileSize 格式化文件大小
func formatFileSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// 避免未使用导入
var _ = time.Now
