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

// handleContainerAction 处理容器相关操作
func (c *WSClient) handleContainerAction(req WSRequest) {
	name := getStr(req.Data, "name")

	// 非 admin 用户检查坑位权限
	if name != "" {
		if msg := c.checkContainerSlotAccess(name); msg != "" {
			c.sendResponse(req.ID, false, msg, nil)
			return
		}
	}

	switch req.Action {
	case "container:start":
		c.containerSimple(req, "POST", "/android/start")
	case "container:stop":
		c.containerSimple(req, "POST", "/android/stop")
	case "container:restart":
		c.containerSimple(req, "POST", "/android/restart")
	case "container:reset":
		name := getStr(req.Data, "name")
		body, status, err := c.hub.deviceRequest("PUT", "/android", map[string]interface{}{"name": name, "start": true})
		c.handleDeviceResponse(req.ID, body, status, err)
		c.clearSlotScreenshot(name)
	case "container:delete":
		name := getStr(req.Data, "name")
		c.clearSlotScreenshot(name)
		_, status, err := c.hub.deviceRequest("DELETE", "/android?name="+url.QueryEscape(name), nil)
		if err != nil {
			c.sendResponse(req.ID, false, err.Error(), nil)
		} else if status >= 400 {
			c.sendResponse(req.ID, false, "删除失败", nil)
		} else {
			c.sendResponse(req.ID, true, "删除成功", nil)
		}
		c.hub.TriggerRefresh()
	case "container:rename":
		name := getStr(req.Data, "name")
		newName := getStr(req.Data, "newName")
		body, status, err := c.hub.deviceRequest("POST", "/android/rename", map[string]interface{}{"name": name, "newName": newName})
		c.handleDeviceResponse(req.ID, body, status, err)
	case "container:copy":
		name := getStr(req.Data, "name")
		indexNum := getNum(req.Data, "indexNum")
		count := getNum(req.Data, "count")
		if count < 1 {
			count = 1
		}
		// 复制是在同一坑位创建多个容器，只检查该坑位权限
		if !c.isAdmin && c.permissions != nil {
			// 未指定坑位（0=自动分配）时按源容器所在坑位检查
			if indexNum <= 0 {
				indexNum = c.getContainerSlot(name)
			}
			if !c.canAccessSlot(indexNum) {
				c.sendResponse(req.ID, false, fmt.Sprintf("无权操作坑位 %d", indexNum), nil)
				return
			}
		}
		path := fmt.Sprintf("/android/copy?name=%s&indexNum=%d&count=%d", url.QueryEscape(name), indexNum, count)
		added, errMsg := c.streamDeviceCopy(path, count)
		if added >= count {
			c.sendResponse(req.ID, true, fmt.Sprintf("复制成功（%d 台）", added), nil)
		} else if added > 0 {
			c.sendResponse(req.ID, false, fmt.Sprintf("部分成功：%d/%d 台，%s", added, count, errMsg), nil)
		} else {
			c.sendResponse(req.ID, false, errMsg, nil)
		}
		c.hub.TriggerRefresh()
	}
}

// handleAliasAction 处理别名操作
func (c *WSClient) handleAliasAction(req WSRequest) {
	// alias:set 和 alias:delete 需要检查坑位权限
	if req.Action != "alias:list" {
		name := getStr(req.Data, "name")
		if name != "" {
			if msg := c.checkContainerSlotAccess(name); msg != "" {
				c.sendResponse(req.ID, false, msg, nil)
				return
			}
		}
	}

	switch req.Action {
	case "alias:list":
		aliases := c.hub.buildAliasMap()
		c.sendResponse(req.ID, true, "ok", map[string]interface{}{"aliases": aliases})
	case "alias:set":
		name := getStr(req.Data, "name")
		alias := getStr(req.Data, "alias")
		if name == "" {
			c.sendResponse(req.ID, false, "容器名不能为空", nil)
			return
		}
		if err := c.hub.alias.SetAlias(name, alias); err != nil {
			c.sendResponse(req.ID, false, "设置别名失败: "+err.Error(), nil)
		} else {
			c.sendResponse(req.ID, true, "ok", nil)
			c.hub.Broadcast("aliases:list", c.hub.buildAliasMap())
		}
	case "alias:delete":
		name := getStr(req.Data, "name")
		if name == "" {
			c.sendResponse(req.ID, false, "容器名不能为空", nil)
			return
		}
		if err := c.hub.alias.DeleteAlias(name); err != nil {
			c.sendResponse(req.ID, false, "删除别名失败: "+err.Error(), nil)
		} else {
			c.sendResponse(req.ID, true, "ok", nil)
			c.hub.Broadcast("aliases:list", c.hub.buildAliasMap())
		}
	}
}

// containerSimple 简单容器操作（start/stop/restart）
func (c *WSClient) containerSimple(req WSRequest, method, path string) {
	name := getStr(req.Data, "name")
	body, status, err := c.hub.deviceRequest(method, path, map[string]interface{}{"name": name})
	c.handleDeviceResponse(req.ID, body, status, err)
	// 非 start 操作清除截图缓存
	if req.Action != "container:start" {
		c.clearSlotScreenshot(name)
	}
	time.AfterFunc(1*time.Second, func() { c.hub.TriggerRefresh() })
}

// handleDeviceResponse 处理设备响应
func (c *WSClient) handleDeviceResponse(reqID string, body []byte, status int, err error) {
	if err != nil {
		c.sendResponse(reqID, false, "设备连接失败: "+err.Error(), nil)
		return
	}
	if status >= 400 {
		var errResp map[string]interface{}
		if json.Unmarshal(body, &errResp) == nil {
			if msg, ok := errResp["message"].(string); ok {
				c.sendResponse(reqID, false, msg, nil)
				return
			}
			if msg, ok := errResp["error"].(string); ok {
				c.sendResponse(reqID, false, msg, nil)
				return
			}
		}
		c.sendResponse(reqID, false, fmt.Sprintf("操作失败 (HTTP %d)", status), nil)
		return
	}
	c.sendResponse(reqID, true, "操作成功", nil)
}

// clearSlotScreenshot 清除容器对应坑位的截图缓存并推送更新
func (c *WSClient) clearSlotScreenshot(containerName string) {
	if c.hub.ssCache == nil {
		return
	}
	slot := c.getContainerSlot(containerName)
	if slot <= 0 {
		return
	}
	if c.hub.ssCache.ClearSlot(slot) {
		c.hub.pushScreenshots(c.hub.ssCache)
	}
}

// streamDeviceCopy 消费设备端 /android/copy 的 SSE 流式响应。
// 设备端异步复制并通过事件推送进度（event:/data: 行），流可能提前关闭，
// 因此结束后轮询容器列表确认最终新增数。返回新增容器数和错误信息。
func (c *WSClient) streamDeviceCopy(path string, count int) (int, string) {
	// 记录复制前的容器名集合，用于识别新增
	before := c.hub.containerNameSet()

	// 总超时按台数估算（每台 90 秒 + 2 分钟基础），下限 3 分钟
	timeout := time.Duration(2+count) * 90 * time.Second
	if timeout < 3*time.Minute {
		timeout = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	reqURL := fmt.Sprintf("http://%s%s", c.hub.deviceAddr, path)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.hub.deviceAuth != "" {
		req.Header.Set("Authorization", c.hub.deviceAuth)
	}
	// 必须用无超时的 streamClient：共享 httpClient 的 10 秒 Timeout 会掐断 SSE 流
	resp, err := c.hub.streamClient.Do(req)
	if err != nil {
		return 0, "设备连接失败: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		if resp.StatusCode == 401 || isAuthFailedBody(body) {
			return 0, "设备需要鉴权密码"
		}
		msg := parseDeviceErrorMessage(body, resp.StatusCode)
		return 0, msg
	}

	// 逐行解析 SSE 流：转发进度事件给前端、捕获错误事件（记录完整事件流便于排查）
	// 设备端事件语义：status=copying 进行中 / status=success 单台成功（message 为容器标识）/ status=done 全部完成 / status=error 或 event:error 失败
	reader := bufio.NewReader(resp.Body)
	var errMsg string
	var streamDone bool
	var events []string
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			line = strings.TrimRight(line, "\r\n")
			switch {
			case strings.HasPrefix(line, "event:"):
				if len(events) < 200 {
					events = append(events, line)
				}
				if strings.TrimSpace(strings.TrimPrefix(line, "event:")) == "error" {
					errMsg = "设备端返回错误"
				}
			case strings.HasPrefix(line, "data:"):
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if len(events) < 200 {
					events = append(events, "data: "+truncateStr(data, 300))
				}
				var prog struct {
					Name    string   `json:"name"`
					Current int      `json:"current"`
					Total   int      `json:"total"`
					Status  string   `json:"status"`
					Message string   `json:"message"`
					Success []string `json:"success"`
				}
				if json.Unmarshal([]byte(data), &prog) == nil {
					switch prog.Status {
					case "error":
						// 设备端明确的失败事件
						if prog.Message != "" {
							errMsg = prog.Message
						} else if errMsg == "" {
							errMsg = "设备端复制失败"
						}
					case "done":
						// 全部完成（success 数组为成功的容器名列表）
						streamDone = true
					}
					// 转发进度（success 的 message 是容器标识 hash，对用户无意义，换成完成提示）
					msg := prog.Message
					if prog.Status == "success" {
						msg = fmt.Sprintf("已完成 %d/%d 台", prog.Current, prog.Total)
					}
					c.pushCopyProgress(prog.Current, prog.Total, msg)
				}
			}
		}
		if readErr != nil {
			break // 流结束（EOF / 超时 / 连接断开）
		}
	}
	log.Printf("[Copy] SSE 流结束 count=%d done=%v errMsg=%q 事件流:\n%s", count, streamDone, errMsg, strings.Join(events, "\n"))

	// 流结束后轮询容器列表确认最终新增数（流可能提前关闭，复制仍在后台进行）
	// 每轮拉取都会广播，前端容器列表实时更新
	pollDeadline := time.Now().Add(time.Duration(count)*90*time.Second + time.Minute)
	if streamDone || errMsg != "" {
		// 设备端已确认完成/失败，只需短暂等待列表同步
		pollDeadline = time.Now().Add(30 * time.Second)
	}
	log.Printf("[Copy] 进入轮询确认 count=%d 窗口=%v", count, time.Until(pollDeadline).Round(time.Second))
	for time.Now().Before(pollDeadline) {
		c.hub.fetchAndBroadcastContainers()
		added := c.hub.countNewContainers(before)
		if added >= count {
			log.Printf("[Copy] 轮询确认完成 新增 %d/%d 台", added, count)
			return added, ""
		}
		c.pushCopyProgress(added, count, fmt.Sprintf("已复制 %d/%d 台", added, count))
		time.Sleep(3 * time.Second)
	}

	added := c.hub.countNewContainers(before)
	log.Printf("[Copy] 轮询结束 最终新增 %d/%d 台 errMsg=%q", added, count, errMsg)
	if added > 0 {
		return added, errMsg
	}
	if errMsg == "" {
		errMsg = "复制超时：未检测到新增容器"
	}
	return 0, errMsg
}

// pushCopyProgress 向当前客户端推送复制进度事件
func (c *WSClient) pushCopyProgress(current, total int, message string) {
	if total <= 0 {
		return
	}
	c.SendJSON(map[string]interface{}{
		"type":  "event",
		"event": "copy:progress",
		"data": map[string]interface{}{
			"current": current,
			"total":   total,
			"message": message,
		},
	})
}

// truncateStr 截断字符串到指定长度
func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// parseDeviceErrorMessage 从设备错误响应中提取可读信息
func parseDeviceErrorMessage(body []byte, status int) string {
	var errResp map[string]interface{}
	if json.Unmarshal(body, &errResp) == nil {
		if msg, ok := errResp["message"].(string); ok {
			return msg
		}
		if msg, ok := errResp["error"].(string); ok {
			return msg
		}
	}
	return fmt.Sprintf("操作失败 (HTTP %d)", status)
}

// containerNameSet 返回当前所有容器名的集合
func (h *WSHub) containerNameSet() map[string]bool {
	h.containerMu.RLock()
	defer h.containerMu.RUnlock()
	names := make(map[string]bool, len(h.parsedContainers))
	for _, pc := range h.parsedContainers {
		names[pc.Name] = true
	}
	return names
}

// countNewContainers 统计不在旧集合中的容器数
func (h *WSHub) countNewContainers(old map[string]bool) int {
	h.containerMu.RLock()
	defer h.containerMu.RUnlock()
	n := 0
	for _, pc := range h.parsedContainers {
		if !old[pc.Name] {
			n++
		}
	}
	return n
}
