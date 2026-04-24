# MYT Panel

<p align="center">
  <strong>魔云互联云手机管理面板</strong><br>
  基于 Go + Vue 3 的轻量级云手机 Web 管理系统
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.4+-4FC08D?style=flat-square&logo=vue.js" alt="Vue">
  <img src="https://img.shields.io/badge/SQLite-embedded-003B57?style=flat-square&logo=sqlite" alt="SQLite">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Platform-ARM64-orange?style=flat-square" alt="Platform">
</p>

---

## 功能概览

### 🖥️ 容器管理

- **多坑位管理** — 可视化网格展示所有云手机容器，实时状态监控
- **生命周期控制** — 启动、停止、重启、重置、删除、复制容器
- **容器创建** — 选择镜像、机型、国家码一键创建新容器
- **批量刷机** — 批量切换容器镜像，支持流式进度推送
- **容器别名** — 为容器设置自定义别名，方便识别管理
- **实时截图** — 每 2 秒自动抓取运行中容器的屏幕截图，按坑位权限过滤推送

### 📱 投屏控制

- **Web 投屏** — 浏览器内实时投屏，基于 WebRTC (Pion) 低延迟传输
- **UDP 媒体流** — TCP/UDP 共用端口号，UDP Mux 复用器转发媒体数据
- **投屏 Token** — 短期 JWT Token 绑定用户+容器，断线重连安全复用
- **连接预热** — 提前建立 WebSocket 连接池，加速首次投屏

### 🔧 终端与文件

- **SSH 终端** — 浏览器内 xterm.js 终端，WebSocket 代理到设备 SSH
- **容器终端** — 通过 SDK exec 接口直接进入容器 Shell
- **文件上传** — 拖拽上传文件到容器，流式转发不占内存
- **证书上传** — 支持向容器上传 SSL 证书
- **APK 自动安装** — 上传 APK 后自动执行 `pm install` 安装

### 🌐 网络管理

- **虚拟内置网卡** — 创建、修改、删除 Bridge 网络
- **VPC 分组** — VPC 组管理，支持 SOCKS5 节点添加与延迟测试
- **VPC 规则** — 容器级规则管理，支持批量添加/删除规则
- **域名直连** — 容器级域名直连配置
- **域名过滤** — 容器级 + 全局域名过滤规则
- **DNS 白名单** — 一键切换白名单 DNS 模式
- **S5 代理** — 容器内 SOCKS5/HTTP 代理的设置、查看、停止
- **网络测速** — 容器连接延迟测试

### 👥 用户与权限

- **多用户体系** — admin / 普通用户两种角色
- **细粒度权限** — 14 项独立权限开关，按需分配
- **坑位隔离** — 普通用户仅可见和操作被授权的坑位
- **账户过期** — 支持设置用户过期时间，过期自动禁用并踢下线
- **密码修改踢人** — 修改密码后自动使旧 Token 失效，强制重新登录

### 📊 设备监控

- **实时状态** — CPU 温度、内存使用、存储占用、运行时间
- **设备信息** — 型号、SDK 版本、IP/MAC 地址、网络速度
- **硬盘监控** — 型号、温度、读写速度
- **设备重启** — 远程重启设备
- **系统升级** — 流式推送设备系统升级进度
- **磁盘清理** — 一键清理磁盘空间

### 📦 镜像与备份

- **镜像管理** — 查看、拉取、删除、清理镜像，流式拉取进度
- **在线镜像源** — 从魔云镜像源获取可用镜像列表
- **备份管理** — 容器备份和机型备份的查看与删除

### ☁️ 魔云平台对接

- **账号绑定** — 登录魔云平台，手机验证码绑定/解绑设备
- **自动同步** — 开启/关闭自动同步坑位状态
- **坑位状态** — 实时查看魔云平台坑位在线状态

### 📱 移动端适配

- **响应式布局** — 自动识别 PC/手机端，切换对应 UI
- **移动端专属** — 完整的移动端视图，触控优化操作体验

### 🔄 系统维护

- **在线更新** — 面板内置在线更新，SHA256 校验 + 自动重启
- **手动更新** — 支持手动替换二进制文件更新
- **完整性校验** — 启动时自动校验二进制 SHA256，防止篡改
- **日志轮转** — 自动日志轮转（50MB/文件，5 份备份，30 天保留）
- **系统设置** — 可配置公网 UDP 端口等参数

---

## 快速开始

### 安装

```bash
# 1. 上传文件到设备
scp myt-panel myt-panel.sha256 user@DEVICE_IP:/home/user/
scp -r deploy user@DEVICE_IP:/home/user/

# 2. SSH 登录设备
ssh user@DEVICE_IP

# 3. 赋予执行权限
chmod +x /home/user/myt-panel

# 4. 安装为系统服务
cd /home/user/deploy

# Alpine Linux
sudo ./install-alpine.sh

# Debian / Ubuntu
sudo ./install-debian.sh
```

### 默认访问

| 项目 | 值 |
|------|-----|
| URL | `http://DEVICE_IP:8081` |
| 用户名 | `myt` |
| 密码 | `myt` |

> **首次登录后请立即修改默认密码！**

详细安装说明请参阅 [INSTALL.md](INSTALL.md)

---

## 从源码构建

### 环境要求

- Go 1.26+
- Node.js 18+
- npm 9+

### 构建

```bash
# Windows
set DEVICE=r1s
set VERSION=1.0.0
build.bat

# Linux / macOS
DEVICE=r1s VERSION=1.0.0 bash scripts/build.sh

# 构建所有设备型号
DEVICE=all VERSION=1.0.0 bash scripts/build.sh
```

### 支持的设备型号

| 代号 | 设备 |
|------|------|
| r1s | R1S |
| r1q | R1Q |
| r1z | R1Z |
| c1 | C1 |
| q1 | Q1 |
| q1n | Q1N |
| p1 | P1 |

### 可选构建优化

| 工具 | 用途 | 安装 |
|------|------|------|
| [garble](https://github.com/BurrowShark/garble) | 代码混淆 | `go install mvdan.cc/garble@latest` |
| [upx](https://upx.github.io/) | 二进制压缩 | `apt install upx` / `brew install upx` |

---

## 技术栈

### 后端

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.26+ |
| HTTP 框架 | [chi/v5](https://github.com/go-chi/chi) |
| WebSocket | [gorilla/websocket](https://github.com/gorilla/websocket) |
| 数据库 | [SQLite](https://gitlab.com/cznic/sqlite)（纯 Go，无需 CGO） |
| 认证 | [golang-jwt](https://github.com/golang-jwt/jwt) + bcrypt |
| 投屏 | [pion/webrtc](https://github.com/pion/webrtc) |
| 日志 | [lumberjack](https://github.com/natefinch/lumberjack) |

### 前端

| 组件 | 技术 |
|------|------|
| 框架 | Vue 3 |
| 构建 | Vite 5 |
| PC 端 UI | [Element Plus](https://element-plus.org/) |
| 移动端 UI | [Vant 4](https://vant-ui.github.io/vant/) |
| 状态管理 | Pinia |
| 终端 | [xterm.js](https://xtermjs.org/) |
| HTTP | Axios |

### 架构特点

- **单二进制部署** — 前端 embed 到 Go 二进制，零依赖运行
- **SQLite WAL 模式** — 并发读写无阻塞，无需外部数据库
- **AES-GCM 加密** — WebSocket 敏感数据加密传输
- **JWT 双 Token** — 登录 Token + 投屏专用 Token，独立过期策略
- **UDP Mux** — 投屏 UDP 流与 TCP 共用端口，NAT 友好
- **流式转发** — 大文件上传、镜像拉取、系统升级均流式处理

---

## 项目结构

```
myt-panel/
├── main.go                  # 入口：服务初始化、路由注册
├── auth.go                  # 认证：JWT、登录、用户 CRUD、权限
├── auth_sync.go             # 魔云平台账号同步
├── ws.go                    # WebSocket Hub：连接管理、消息分发
├── ws_container.go          # WS 容器操作：启停、重置、删除、复制
├── ws_device.go             # WS 设备操作：信息、重启、升级、清理
├── ws_user.go               # WS 用户管理：CRUD、权限
├── ws_permission.go         # WS 权限检查：角色+坑位权限
├── ws_myt.go                # WS 魔云平台：登录、绑定、同步
├── ws_sdk.go                # WS SDK 代理：镜像、网络、备份
├── ws_proxy.go              # WS 容器代理：S5、剪贴板、短信
├── ws_screenshot.go         # WS 截图轮询：并发抓取、缓存推送
├── ws_settings.go           # WS 系统设置
├── device.go                # 设备服务：SDK 代理、SSH、状态轮询
├── projection_proxy.go      # 投屏代理：WebRTC 信令、连接池
├── udp_mux.go               # UDP 复用器：投屏媒体流转发
├── upload_proxy.go          # 文件上传代理：流式转发、APK 自动安装
├── container_alias.go       # 容器别名服务
├── mirror_list.go           # 在线镜像源
├── updater.go               # 在线更新：下载、校验、替换、重启
├── integrity.go             # 完整性校验：SHA256 自校验
├── android_proxy.go         # 安卓代理控制
├── build.bat                # Windows 构建脚本
├── scripts/build.sh         # Linux/macOS 构建脚本
├── deploy/                  # 部署配置
│   ├── alpine-openrc        # Alpine 服务文件
│   ├── debian-systemd.service # Debian 服务文件
│   ├── install-alpine.sh    # Alpine 安装脚本
│   └── install-debian.sh    # Debian 安装脚本
├── frontend/                # 前端 Vue 3 项目
│   └── src/
│       ├── views/           # 页面视图
│       │   ├── Login.vue
│       │   ├── Dashboard.vue
│       │   ├── AndroidManage.vue
│       │   ├── DeviceManage.vue
│       │   ├── NetworkManage.vue
│       │   ├── UserManagement.vue
│       │   └── mobile/     # 移动端视图
│       └── components/     # 组件
│           ├── Sidebar.vue
│           ├── DeviceStatusCard.vue
│           └── android/    # 容器管理组件
├── webplayer/               # 投屏播放器静态资源
└── tools/                   # 测试工具
```

---

## API 文档

### 认证方式

所有需要认证的接口均使用 JWT Token，支持两种传递方式：

1. **HTTP Header**：`Authorization: Bearer <token>`
2. **Query 参数**：`?token=<token>`（用于 WebSocket 连接）

---

### HTTP REST API

#### 登录

```
POST /api/auth/login
```

请求体：

```json
{
  "username": "myt",
  "password": "myt"
}
```

响应：

```json
{
  "token": "eyJhbGciOi...",
  "role": "admin",
  "username": "myt",
  "session_key": "base64-encoded-32-bytes"
}
```

> `session_key` 用于后续 WebSocket 通信的 AES-GCM 加密，客户端需保存。

---

#### 登出

```
POST /api/auth/logout
```

Header：`Authorization: Bearer <token>`

响应：

```json
{ "ok": true }
```

> 登出后该用户所有已签发的 Token 立即失效，WebSocket 连接被踢出。

---

#### 获取当前用户

```
GET /api/auth/me
```

Header：`Authorization: Bearer <token>`

响应：

```json
{
  "id": 1,
  "username": "myt",
  "role": "admin"
}
```

普通用户额外返回 `permissions` 字段。

---

#### 获取面板版本

```
GET /api/version
```

无需认证。

响应：

```json
{ "version": "0.3.6" }
```

---

#### 容器文件上传

```
POST /api/container/{name}/upload
```

Header：`Authorization: Bearer <token>`

Body：`multipart/form-data`，字段名为 `file`

> 上传 `.apk` 文件后自动执行 `pm install` 安装。

---

#### 容器证书上传

```
POST /api/container/{name}/cert
```

Header：`Authorization: Bearer <token>`

Body：`multipart/form-data`，字段名为 `file`

---

#### 投屏代理

```
GET /lgcloud?token=<projection_token>
```

投屏专用 Token（通过 WebSocket `projection:token` 动作获取），24 小时有效。

---

#### SDK HTTP 代理

```
ANY /api/sdk/*
```

仅 admin 可用。直接代理到设备 SDK HTTP 接口，路径透传。

---

### WebSocket API

连接地址：`ws://HOST:8081/ws?token=<jwt_token>`

所有业务操作通过 WebSocket 消息完成，采用 **请求-响应** 模式。

#### 消息格式

**请求**（客户端 → 服务端）：

```json
{
  "action": "containers:refresh",
  "id": "unique-request-id",
  "data": {}
}
```

**响应**（服务端 → 客户端）：

```json
{
  "type": "response",
  "id": "unique-request-id",
  "ok": true,
  "message": "",
  "data": {}
}
```

**事件推送**（服务端 → 客户端）：

```json
{
  "type": "event",
  "event": "containers:list",
  "data": {}
}
```

**加密消息**（含 `session_key` 时敏感数据加密传输）：

```json
{
  "type": "encrypted",
  "data": "base64(aes-gcm-encrypted-payload)"
}
```

---

#### 容器操作

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `containers:refresh` | 刷新容器列表 | 所有 | — |
| `container:start` | 启动容器 | `container_start` + 坑位 | `name` |
| `container:stop` | 停止容器 | `container_start` + 坑位 | `name` |
| `container:restart` | 重启容器 | `container_restart` + 坑位 | `name` |
| `container:reset` | 重置容器 | `container_reset` + 坑位 | `name` |
| `container:delete` | 删除容器 | `container_delete` + 坑位 | `name` |
| `container:rename` | 重命名容器 | `container_rename` + 坑位 | `name`, `newName` |
| `container:copy` | 复制容器 | `container_copy` + 坑位 | `name`, `indexNum`, `count` |

---

#### 别名管理

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `alias:list` | 获取别名列表 | `alias_manage` | — |
| `alias:set` | 设置别名 | `alias_manage` + 坑位 | `name`, `alias` |
| `alias:delete` | 删除别名 | `alias_manage` + 坑位 | `name` |

---

#### 镜像管理

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `sdk:listImages` | 列出镜像 | `image_view` | `imageName`(可选) |
| `sdk:pullImage` | 拉取镜像（流式） | `image_view` | `imageUrl` |
| `sdk:deleteImage` | 删除镜像 | `image_view` | `image` |
| `sdk:pruneImages` | 清理未用镜像 | `image_view` | — |
| `sdk:getPhoneModels` | 获取机型列表 | `container_create` | — |
| `sdk:getCountryCodes` | 获取国家码 | `container_create` | — |
| `sdk:batchChangeImage` | 批量切换镜像 | `backup_manage` | 见下方 |

`sdk:batchChangeImage` 参数：

```json
{
  "names": ["container-1", "container-2"],
  "image": "镜像名"
}
```

---

#### 容器创建

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `sdk:createContainer` | 创建容器 | `container_create` | 见下方 |

`sdk:createContainer` 参数：

```json
{
  "indexNum": 1,
  "image": "镜像名",
  "phoneModel": "机型",
  "countryCode": "国家码",
  "alias": "别名(可选)"
}
```

---

#### 备份管理

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `sdk:listBackups` | 列出容器备份 | `backup_manage` | — |
| `sdk:deleteBackup` | 删除容器备份 | `backup_manage` | `name` |
| `sdk:listModelBackups` | 列出机型备份 | `backup_manage` | — |
| `sdk:deleteModelBackup` | 删除机型备份 | `backup_manage` | `name` |

---

#### 网络管理

**Bridge 网卡**

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `sdk:listBridges` | 列出 Bridge | `network_bridge` | — |
| `sdk:createBridge` | 创建 Bridge | `network_bridge` | Bridge 配置 |
| `sdk:updateBridge` | 更新 Bridge | `network_bridge` | Bridge 配置 |
| `sdk:deleteBridge` | 删除 Bridge | `network_bridge` | `name` |

**VPC 分组**

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `sdk:listVpcGroups` | 列出 VPC 组 | `vpc_manage` | `alias`(可选) |
| `sdk:createVpcGroup` | 创建 VPC 组 | `vpc_manage` | 组配置 |
| `sdk:deleteVpcGroup` | 删除 VPC 组 | `vpc_manage` | `id` |
| `sdk:renameVpcGroup` | 重命名 VPC 组 | `vpc_manage` | 组配置 |
| `sdk:refreshVpcGroup` | 刷新 VPC 组 | `vpc_manage` | 组配置 |
| `sdk:deleteVpcNode` | 删除 VPC 节点 | `vpc_manage` | `vpcID` |
| `sdk:addVpcSocks` | 添加 SOCKS5 节点 | `vpc_manage` | 节点配置 |
| `sdk:testVpcNode` | 测试节点延迟 | `vpc_manage` | `address` |

**VPC 规则**

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `sdk:listContainerRules` | 列出容器规则 | `vpc_manage` | — |
| `sdk:addVpcRule` | 添加规则 | `vpc_manage` | 规则配置 |
| `sdk:removeVpcRule` | 删除规则 | `vpc_manage` | 规则配置 |
| `sdk:addVpcRuleBatch` | 批量添加规则 | `vpc_manage` | 规则列表 |
| `sdk:removeVpcRuleBatch` | 批量删除规则 | `vpc_manage` | 规则列表 |

**域名管理**

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `sdk:getDomainDirect` | 获取域名直连 | `vpc_manage` | `containerID` |
| `sdk:setDomainDirect` | 设置域名直连 | `vpc_manage` | 直连配置 |
| `sdk:deleteDomainDirect` | 删除域名直连 | `vpc_manage` | `containerID` |
| `sdk:getDomainFilter` | 获取域名过滤 | `vpc_manage` | `containerID` |
| `sdk:setDomainFilter` | 设置域名过滤 | `vpc_manage` | 过滤配置 |
| `sdk:deleteDomainFilter` | 删除域名过滤 | `vpc_manage` | `containerID` |
| `sdk:getGlobalDomainFilter` | 获取全局域名过滤 | `vpc_manage` | — |
| `sdk:setGlobalDomainFilter` | 设置全局域名过滤 | `vpc_manage` | 过滤配置 |
| `sdk:deleteGlobalDomainFilter` | 删除全局域名过滤 | `vpc_manage` | — |
| `sdk:toggleWhiteListDns` | 切换 DNS 白名单 | `vpc_manage` | 配置 |

---

#### 设备管理

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `device:info` | 设备信息 | 所有 | — |
| `device:version` | SDK 版本 | admin | — |
| `device:mirrors` | 在线镜像源 | 所有 | — |
| `device:reboot` | 重启设备 | admin | — |
| `device:upgrade` | 系统升级（流式） | admin | — |
| `device:cleanDisk` | 清理磁盘（流式） | admin | — |

---

#### 用户管理

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `user:list` | 用户列表 | admin | — |
| `user:create` | 创建用户 | admin | `username`, `password`, `role`, `expiresAt` |
| `user:update` | 更新用户 | admin | `id`, `password`/`role`/`expiresAt`/`enabled` |
| `user:delete` | 删除用户 | admin | `id` |
| `user:getPermissions` | 获取权限 | admin | `id` |
| `user:setPermissions` | 设置权限 | admin | 见下方 |

`user:setPermissions` 参数：

```json
{
  "id": 2,
  "slots": [1, 2, 3],
  "container_start": true,
  "container_restart": false,
  "container_reset": false,
  "container_delete": false,
  "container_rename": false,
  "container_copy": false,
  "container_create": false,
  "alias_manage": false,
  "backup_manage": false,
  "image_view": false,
  "projection": true,
  "terminal": false,
  "network_bridge": false,
  "vpc_manage": false
}
```

---

#### 投屏

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `projection:token` | 获取投屏 Token | `projection` + 坑位 | `container_name` |

响应：

```json
{
  "ok": true,
  "data": {
    "token": "projection-jwt-token",
    "udpPort": "8081"
  }
}
```

---

#### 容器代理

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `proxy:status` | 查看 S5 代理状态 | `container_start` + 坑位 | `name` |
| `proxy:set` | 设置 S5 代理 | `container_start` + 坑位 | `name`, `addr`, `port`, `usr`, `pwd`, `type` |
| `proxy:stop` | 停止 S5 代理 | `container_start` + 坑位 | `name` |
| `clipboard:get` | 获取剪贴板 | `container_start` + 坑位 | `name` |
| `clipboard:set` | 设置剪贴板 | `container_start` + 坑位 | `name`, `text` |
| `android:shake` | 模拟摇一摇 | `container_start` + 坑位 | `name` |
| `android:sms` | 发送短信 | `container_start` + 坑位 | `name`, `address`, `body` |
| `android:ping` | 容器延迟测试 | `container_start` + 坑位 | `name` |

---

#### 魔云平台

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `myt:status` | 登录状态 | admin | — |
| `myt:slotStates` | 坑位在线状态 | 所有 | — |
| `myt:login` | 平台登录 | admin | `username`, `password` |
| `myt:logout` | 平台登出 | admin | — |
| `myt:sync` | 同步坑位 | admin | — |
| `myt:autoToggle` | 自动同步开关 | admin | `autoSync` |
| `myt:bindStatus` | 绑定状态 | admin | — |
| `myt:bind` | 绑定设备 | admin | — |
| `myt:vcode` | 获取验证码 | admin | `phone` |
| `myt:unbind` | 解绑设备 | admin | `vcode`, `vkey` |

---

#### 面板更新

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `panel:version` | 当前版本 | 所有 | — |
| `panel:checkUpdate` | 检查更新 | admin | — |
| `panel:doUpdate` | 执行更新 | admin | — |

---

#### 系统设置

| Action | 说明 | 权限 | 参数 |
|--------|------|------|------|
| `settings:get` | 获取所有设置 | admin | — |
| `settings:set` | 保存设置 | admin | `key`, `value` |

---

### 服务端事件推送

客户端连接 WebSocket 后，服务端会主动推送以下事件：

| 事件 | 说明 | 数据格式 |
|------|------|----------|
| `containers:list` | 容器列表（按坑位过滤） | SDK 容器 JSON |
| `aliases:list` | 别名映射 | `{ "container-name": "别名" }` |
| `screenshots` | 容器截图 | `{ "1": "data:image/jpeg;base64,..." }` |
| `user:permissions` | 用户权限（非 admin） | 权限对象 |
| `user:kicked` | 用户被踢出 | `{ "username": "xxx", "reason": "expired" }` |
| `token:refresh` | Token 自动刷新 | `{ "token": "new-jwt-token" }` |
| `task:progress` | 任务进度（升级/拉取镜像等） | `{ "action": "pullImage", "chunk": "...", "done": false }` |

---

### 权限体系

#### 角色

| 角色 | 说明 |
|------|------|
| `admin` | 全部权限，不可被删除，可管理用户和系统设置 |
| `user` | 受权限控制，只能访问被授权的坑位和功能 |

#### 权限项

| 权限 | 对应操作 |
|------|----------|
| `slots` | 允许访问的坑位列表，如 `[1, 2, 3]` |
| `container_start` | 启动/停止容器、S5 代理、剪贴板、安卓控制 |
| `container_restart` | 重启容器 |
| `container_reset` | 重置容器 |
| `container_delete` | 删除容器 |
| `container_rename` | 重命名容器 |
| `container_copy` | 复制容器 |
| `container_create` | 创建容器、查看机型/国家码 |
| `alias_manage` | 别名管理 |
| `backup_manage` | 备份管理、批量刷机 |
| `image_view` | 镜像管理 |
| `projection` | 投屏 |
| `terminal` | 终端 |
| `network_bridge` | Bridge 网卡管理 |
| `vpc_manage` | VPC 分组/规则/域名管理 |

> **坑位权限**：普通用户只能看到和操作 `slots` 列表中包含的坑位对应的容器，容器列表、截图、投屏均按坑位过滤。

---

## 安全机制

| 机制 | 说明 |
|------|------|
| JWT 认证 | 24 小时有效，登出即失效 |
| AES-GCM 加密 | WebSocket 敏感消息（如 Token 刷新）加密传输 |
| 投屏 Token | 独立短期 Token，绑定用户+容器，24 小时有效 |
| 密码哈希 | bcrypt 加密存储 |
| 二进制校验 | 启动时 SHA256 自校验，防止篡改 |
| 坑位隔离 | 普通用户只能操作被授权的坑位 |
| 账户过期 | 过期自动禁用，踢出所有连接 |
| CORS | 动态 Origin，允许局域网访问 |

---

## 端口说明

| 端口 | 协议 | 用途 |
|------|------|------|
| 8081 | TCP | 面板 HTTP/WebSocket（可自定义） |
| 8081 | UDP | 投屏媒体流（与 TCP 共用） |
| 8000 | TCP | 设备 SDK 服务（本地回环） |
| 30000+ | TCP | 容器内部 API 端口（`30000 + (indexNum-1)*100 + 1`） |

---

## 许可证

[MIT License](LICENSE)
