package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// containerAPIPort 计算容器内部安卓 API 端口（非桥接模式）
// 复用 upload_proxy.go 中的同名函数
func (c *WSClient) containerAPIPort(containerName string) int {
	return containerAPIPort(c.hub, containerName)
}

// handleProxyAction 处理容器 S5 代理操作
func (c *WSClient) handleProxyAction(req WSRequest) {
	log.Printf("[DEBUG] handleProxyAction: action=%s id=%s", req.Action, req.ID)
	name := getStr(req.Data, "name")
	if name == "" {
		c.sendResponse(req.ID, false, "缺少容器名称", nil)
		return
	}

	// 坑位权限检查
	if msg := c.checkContainerSlotAccess(name); msg != "" {
		c.sendResponse(req.ID, false, msg, nil)
		return
	}

	port := c.containerAPIPort(name)
	if port == 0 {
		c.sendResponse(req.ID, false, "找不到容器或无法计算端口", nil)
		return
	}

	switch req.Action {
	case "proxy:status":
		c.proxyRequest(req, port, "GET", fmt.Sprintf("/proxy?cmd=1"), nil)

	case "proxy:set":
		addr := getStr(req.Data, "addr")
		s5Port := getStr(req.Data, "port")
		usr := getStr(req.Data, "usr")
		pwd := getStr(req.Data, "pwd")
		s5Type := getStr(req.Data, "type")
		if addr == "" || s5Port == "" {
			c.sendResponse(req.ID, false, "IP 和端口不能为空", nil)
			return
		}
		if s5Type == "" {
			s5Type = "1"
		}
		q := url.Values{}
		q.Set("cmd", "2")
		q.Set("type", s5Type)
		q.Set("ip", addr)
		q.Set("port", s5Port)
		q.Set("usr", usr)
		q.Set("pwd", pwd)
		c.proxyRequest(req, port, "GET", "/proxy?"+q.Encode(), nil)

	case "proxy:stop":
		c.proxyRequest(req, port, "GET", "/proxy?cmd=3", nil)

	case "clipboard:get":
		c.proxyRequest(req, port, "GET", "/clipboard?cmd=1", nil)

	case "clipboard:set":
		text := getStr(req.Data, "text")
		// POST + JSON UTF-8 body 传中文，比 URL 编码更可靠
		c.proxyRequest(req, port, "POST", "/clipboard", map[string]interface{}{
			"cmd":  "2",
			"text": text,
		})

	case "clipboard:paste":
		// 设置安卓剪贴板后自动触发粘贴（绕过 WebRTC KEYTEXT 中文乱码）
		text := getStr(req.Data, "text")
		if text == "" {
			c.sendResponse(req.ID, false, "文本为空", nil)
			return
		}
		// 1) 通过容器 HTTP API 设置剪贴板（GET + URL 参数，容器端只支持此方式）
		clipboardURL := fmt.Sprintf("/clipboard?cmd=2&text=%s", url.QueryEscape(text))
		c.proxyRequest(req, port, "GET", clipboardURL, nil)
		// 2) 延迟后通过 exec 触发粘贴
		//    方案 A: input keyevent 279 (KEYCODE_PASTE，Android 7+)
		//    方案 B: am broadcast (备用，部分容器支持)
		pasteCmd := "input keyevent 279"
		execBody, _ := json.Marshal(map[string]interface{}{
			"name":    name,
			"command": []string{"sd", "-c", pasteCmd},
		})
		execURL := fmt.Sprintf("http://%s/android/exec", c.hub.deviceAddr)
		go func() {
			time.Sleep(500 * time.Millisecond) // 等剪贴板设置完成
			client := &http.Client{Timeout: 10 * time.Second}
			httpReq, err := http.NewRequest("POST", execURL, strings.NewReader(string(execBody)))
			if err != nil {
				return
			}
			httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
			resp, err := client.Do(httpReq)
			if err != nil {
				log.Printf("[ClipboardPaste] 容器 %s 触发粘贴失败: %v", name, err)
				return
			}
			resp.Body.Close()
			log.Printf("[ClipboardPaste] 容器 %s 触发粘贴 (HTTP %d)", name, resp.StatusCode)
		}()

	case "android:shake":
		c.proxyRequest(req, port, "GET", "/modifydev?cmd=17&shake=1", nil)

	case "android:ping":
		// 轻量 ping：请求容器 API 测延迟
		start := time.Now()
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/proxy?cmd=1", port))
		latency := time.Since(start).Milliseconds()
		if err != nil {
			c.sendResponse(req.ID, true, "ok", map[string]interface{}{"latency": -1})
			return
		}
		resp.Body.Close()
		c.sendResponse(req.ID, true, "ok", map[string]interface{}{"latency": latency})

	case "android:sms":
		address := getStr(req.Data, "address")
		body := getStr(req.Data, "body")
		if address == "" || body == "" {
			c.sendResponse(req.ID, false, "发送号码和内容不能为空", nil)
			return
		}
		// POST /sms?cmd=4 + JSON body {"address":"xxx","body":"xxx"}
		smsBody := map[string]string{"address": address, "body": body}
		c.proxyRequest(req, port, "POST", "/sms?cmd=4", smsBody)

	case "android:orientation":
		rotation := getStr(req.Data, "rotation")
		if rotation == "" {
			rotation = "0"
		}
		// 通过 exec 执行 settings 命令旋转设备
		execBody, _ := json.Marshal(map[string]interface{}{
			"name":    name,
			"command": []string{"sd", "-c", fmt.Sprintf("settings put system accelerometer_rotation 0; settings put system user_rotation %s", rotation)},
		})
		execURL := fmt.Sprintf("http://%s/android/exec", c.hub.deviceAddr)
		client := &http.Client{Timeout: 10 * time.Second}
		httpReq, err := http.NewRequest("POST", execURL, strings.NewReader(string(execBody)))
		if err != nil {
			c.sendResponse(req.ID, false, err.Error(), nil)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := client.Do(httpReq)
		if err != nil {
			c.sendResponse(req.ID, false, "旋转设备失败: "+err.Error(), nil)
			return
		}
		defer resp.Body.Close()
		io.ReadAll(resp.Body)
		log.Printf("[Orientation] 容器 %s 旋转 → %s (HTTP %d)", name, rotation, resp.StatusCode)
		c.sendResponse(req.ID, true, "ok", nil)
	}
}

// proxyRequest 向容器内部 API 发送 HTTP 请求
func (c *WSClient) proxyRequest(req WSRequest, port int, method, path string, body interface{}) {
	reqURL := fmt.Sprintf("http://%s:%d%s", "127.0.0.1", port, path)

	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(data))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		c.sendResponse(req.ID, false, err.Error(), nil)
		return
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		c.sendResponse(req.ID, false, "容器未响应: "+err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[Proxy] %s %s → HTTP %d", method, reqURL, resp.StatusCode)

	if resp.StatusCode >= 400 {
		c.sendResponse(req.ID, false, fmt.Sprintf("容器返回错误 (HTTP %d)", resp.StatusCode), nil)
		return
	}
	c.sendResponse(req.ID, true, "ok", json.RawMessage(respBody))
}
