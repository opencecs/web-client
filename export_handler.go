package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ContainerExportHandler 容器导出下载处理器
type ContainerExportHandler struct {
	auth *AuthService
	hub  *WSHub
}

// HandleExport GET /api/container/export?name=xxx&token=xxx
// 1. 调用 SDK POST /android/export 触发导出，获取 exportName
// 2. 从设备下载导出的文件，流式转发给浏览器
func (h *ContainerExportHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	containerName := r.URL.Query().Get("name")
	if containerName == "" {
		jsonError(w, "缺少容器名称", 400)
		return
	}

	// 权限检查（兼容 query token 和 JWT）
	if !h.checkExportAccess(r) {
		jsonError(w, "无权限", 403)
		return
	}

	log.Printf("[Export] 开始导出容器: %s", containerName)

	// 1. 调用 SDK POST /android/export 获取 exportName
	exportName, err := h.triggerExport(containerName)
	if err != nil {
		log.Printf("[Export] 导出失败: %v", err)
		// 设备认证失败时返回 401，前端检测并弹出密码输入
		if strings.Contains(err.Error(), "HTTP 401") {
			jsonError(w, "设备需要鉴权密码", 401)
		} else {
			jsonError(w, "导出失败: "+err.Error(), 500)
		}
		return
	}

	log.Printf("[Export] 导出完成，文件名: %s，开始流式下载", exportName)

	// 2. 从设备下载文件，流式转发给浏览器
	if err := h.streamDownload(w, exportName, containerName); err != nil {
		log.Printf("[Export] 下载失败: %v", err)
		// 此时可能已经写了部分响应头，无法再返回 JSON 错误
		// 只记录日志
	}
}

// triggerExport 调用设备 SDK POST /android/export，返回导出文件名。
// 设备端返回 SSE 流（data: 行）推送打包进度，最终事件携带 exportName；
// 同时兼容旧的纯 JSON 同步响应（{"code":0,"data":{"exportName":"..."}}）。
func (h *ContainerExportHandler) triggerExport(name string) (string, error) {
	reqURL := fmt.Sprintf("http://%s/android/export", h.hub.deviceAddr)
	data, _ := json.Marshal(map[string]string{"name": name})

	// 打包耗时较长：无超时 streamClient + ctx 兜底（30 分钟）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if h.hub.deviceAuth != "" {
		httpReq.Header.Set("Authorization", h.hub.deviceAuth)
	}
	resp, err := h.hub.streamClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("设备连接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		if resp.StatusCode == 401 || isAuthFailedBody(respBody) {
			return "", fmt.Errorf("导出失败 (HTTP 401)")
		}
		return "", fmt.Errorf("导出失败 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	// 逐行解析：SSE 事件流提取 exportName / 错误信息；非 SSE 行按旧 JSON 收集
	reader := bufio.NewReader(resp.Body)
	var (
		events     []string
		exportName string
		errMsg     string
		jsonBody   strings.Builder
	)
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			trimmed := strings.TrimRight(line, "\r\n")
			switch {
			case trimmed == "":
				// 空行（SSE 事件分隔符）
			case strings.HasPrefix(trimmed, "event:"):
				if len(events) < 200 {
					events = append(events, trimmed)
				}
				if strings.TrimSpace(strings.TrimPrefix(trimmed, "event:")) == "error" && errMsg == "" {
					errMsg = "设备端返回错误"
				}
			case strings.HasPrefix(trimmed, "data:"):
				payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				if len(events) < 200 {
					events = append(events, "data: "+truncateStr(payload, 300))
				}
				if n, m := parseExportEvent(payload); n != "" {
					exportName = n
				} else if m != "" && errMsg == "" {
					errMsg = m
				}
			default:
				// 非 SSE 格式行：按旧的纯 JSON 响应收集
				jsonBody.WriteString(trimmed)
			}
		}
		if exportName != "" || errMsg != "" || readErr != nil {
			break // 拿到文件名 / 出错 / 流结束
		}
	}
	log.Printf("[Export] SSE 流结束 name=%s exportName=%q errMsg=%q 事件流:\n%s", name, exportName, errMsg, strings.Join(events, "\n"))

	if exportName != "" {
		return exportName, nil
	}
	// 兼容旧纯 JSON 响应
	if jsonBody.Len() > 0 {
		var exportResp struct {
			Code int    `json:"code"`
			Data struct {
				ExportName string `json:"exportName"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(jsonBody.String()), &exportResp); err == nil && exportResp.Data.ExportName != "" {
			return exportResp.Data.ExportName, nil
		}
	}
	if errMsg != "" {
		return "", fmt.Errorf("%s", errMsg)
	}
	return "", fmt.Errorf("响应缺少文件名")
}

// parseExportEvent 解析导出 SSE 事件，返回 (exportName, 错误信息)
func parseExportEvent(payload string) (string, string) {
	var ev map[string]interface{}
	if json.Unmarshal([]byte(payload), &ev) != nil {
		return "", ""
	}
	if s, _ := ev["status"].(string); s == "error" {
		if m, _ := ev["message"].(string); m != "" {
			return "", m
		}
		return "", "设备端导出失败"
	}
	// exportName 可能位于顶层或 data 对象内
	if v, _ := ev["exportName"].(string); v != "" {
		return v, ""
	}
	if d, ok := ev["data"].(map[string]interface{}); ok {
		if v, _ := d["exportName"].(string); v != "" {
			return v, ""
		}
	}
	return "", ""
}

// streamDownload 从设备下载导出文件，流式转发给浏览器
func (h *ContainerExportHandler) streamDownload(w http.ResponseWriter, exportName, containerName string) error {
	// 尝试多个下载路径
	downloadURLs := []string{
		fmt.Sprintf("http://%s/backup/download?name=%s", h.hub.deviceAddr, url.QueryEscape(exportName)),
		fmt.Sprintf("http://%s/android/imageTar/download?name=%s", h.hub.deviceAddr, url.QueryEscape(exportName)),
		fmt.Sprintf("http://%s/android/export/download?name=%s", h.hub.deviceAddr, url.QueryEscape(exportName)),
	}

	client := h.hub.streamClient
	var resp *http.Response
	var successURL string

	deviceAuth := h.hub.deviceAuth
	for _, downloadURL := range downloadURLs {
		httpReq, _ := http.NewRequest("GET", downloadURL, nil)
		if deviceAuth != "" {
			httpReq.Header.Set("Authorization", deviceAuth)
		}
		r, err := client.Do(httpReq)
		if err != nil {
			log.Printf("[Export] 下载路径失败: %s, %v", downloadURL, err)
			continue
		}
		if r.StatusCode >= 400 {
			r.Body.Close()
			log.Printf("[Export] 下载路径 HTTP %d: %s", r.StatusCode, downloadURL)
			continue
		}
		resp = r
		successURL = downloadURL
		break
	}

	if resp == nil {
		return fmt.Errorf("所有下载路径均失败")
	}
	defer resp.Body.Close()

	log.Printf("[Export] 下载成功: %s", successURL)

	// 设置浏览器下载响应头
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/json" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exportName))
	if resp.ContentLength > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.ContentLength))
	}
	w.WriteHeader(resp.StatusCode)

	written, err := io.Copy(w, resp.Body)
	log.Printf("[Export] 容器 %s 下载完成: %s (%d 字节)", containerName, exportName, written)
	return err
}

// checkExportAccess 检查导出权限
func (h *ContainerExportHandler) checkExportAccess(r *http.Request) bool {
	// 优先检查 JWT（已通过中间件）
	claims, ok := r.Context().Value(userContextKey).(*Claims)
	if ok && claims != nil {
		if claims.Role == "admin" {
			return true
		}
		perms := h.auth.GetUserPermissions(claims.UserID)
		return perms != nil && perms.BackupManage
	}

	// 回退到 query token
	token := r.URL.Query().Get("token")
	if token == "" {
		return false
	}
	claims, err := h.auth.parseToken(token)
	if err != nil {
		return false
	}
	if claims.Role == "admin" {
		return true
	}
	perms := h.auth.GetUserPermissions(claims.UserID)
	return perms != nil && perms.BackupManage
}
