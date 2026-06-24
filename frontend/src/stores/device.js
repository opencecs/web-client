import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from './auth.js'

// AES-GCM 解密（配合后端 aesEncrypt）
async function aesDecrypt(sessionKeyBase64, encryptedBase64) {
  const keyBytes = Uint8Array.from(atob(sessionKeyBase64), c => c.charCodeAt(0))
  const key = await crypto.subtle.importKey('raw', keyBytes, 'AES-GCM', false, ['decrypt'])
  const data = Uint8Array.from(atob(encryptedBase64), c => c.charCodeAt(0))
  const nonce = data.slice(0, 12)
  const ciphertext = data.slice(12)
  const plaintext = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: nonce }, key, ciphertext)
  return new TextDecoder().decode(plaintext)
}

export const useDeviceStore = defineStore('device', () => {
  const status = ref(null)
  const online = ref(false)
  const containers = ref([])
  const containerAliases = ref({})
  const screenshots = ref({})
  let ws = null
  let _reqId = 0
  let _kicked = false
  let _reconnectDelay = 1000

  // 设备鉴权 401 触发回调
  const _auth401Listeners = new Set()

  const _ws = { get value() { return ws } }

  const pendingRequests = new Map()
  const eventListeners = new Set()

  function connect() {
    const auth = useAuthStore()
    if (!auth.token) return
    _kicked = false

    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${location.host}/ws?token=${auth.token}`

    ws = new WebSocket(url)

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)

        if (msg.type === 'encrypted' && msg.data && auth.sessionKey) {
          aesDecrypt(auth.sessionKey, msg.data).then(plaintext => {
            try { handleMessage(JSON.parse(plaintext)) } catch {}
          }).catch(() => {})
          return
        }

        handleMessage(msg)
      } catch (e) {}
    }

    function handleMessage(msg) {
      if (msg.type === 'event') {
        if (msg.event === 'device:status') {
          status.value = msg.data?.data || msg.data
          online.value = msg.data?.online ?? true
        }
        if (msg.event === 'device:auth401') {
          triggerAuth401()
        }
        if (msg.event === 'containers:list') {
          const raw = msg.data
          const list = raw?.list || raw?.data?.list || []
          containers.value = Array.isArray(list) ? list.sort((a, b) => a.indexNum - b.indexNum) : []
        }
        if (msg.event === 'aliases:list') {
          containerAliases.value = msg.data || {}
        }
        if (msg.event === 'user:kicked') {
          const kickedUser = msg.data?.username
          const currentUser = auth.user?.username
          if (kickedUser && currentUser && kickedUser !== currentUser) return
          _kicked = true
          const reason = msg.data?.reason
          if (reason === 'password_changed') alert('密码已被修改，请重新登录')
          else if (reason === 'expired') alert('账号已到期，请联系管理员续期')
          else if (reason === 'disabled') alert('账号已被禁用')
          else if (reason !== 'logout') alert('账号已被强制下线')
          auth.logout()
          window.location.href = '/login'
          return
        }
        if (msg.event === 'user:permissions') auth.setPermissions(msg.data)
        if (msg.event === 'screenshots') screenshots.value = msg.data || {}
        if (msg.event === 'token:refresh' && msg.data?.token) {
          auth.token = msg.data.token
          localStorage.setItem('token', msg.data.token)
        }

        for (const listener of eventListeners) { try { listener(msg) } catch {} }
      }

      if (msg.type === 'response' && msg.id) {
        const pending = pendingRequests.get(msg.id)
        if (pending) {
          clearTimeout(pending.timer)
          pendingRequests.delete(msg.id)
          if (msg.ok) {
            pending.resolve(msg)
          } else {
            const err = new Error(msg.message || '操作失败')
            // 检测设备鉴权失败
            if (msg.message?.includes('HTTP 401') || msg.message?.includes('设备需要鉴权密码')
                || msg.message?.includes('Authentication Failed')) {
              err.isAuth401 = true
              // 触发外部 401 回调
              _auth401Listeners.forEach(fn => { try { fn() } catch {} })
            }
            pending.reject(err)
          }
        }
      }
    }

    ws.onopen = () => { _reconnectDelay = 1000 }

    ws.onclose = () => {
      online.value = false
      if (_kicked) return
      setTimeout(() => connect(), Math.min(_reconnectDelay * 2, 30000))
      _reconnectDelay = Math.min(_reconnectDelay * 2, 30000)
    }

    ws.onerror = () => ws.close()
  }

  function disconnect() {
    for (const [id, pending] of pendingRequests) {
      clearTimeout(pending.timer)
      pending.reject(new Error('连接已断开'))
    }
    pendingRequests.clear()
    if (ws) { ws.close(); ws = null }
  }

  function request(action, data = {}, timeout = 30000) {
    return new Promise((resolve, reject) => {
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket 未连接'))
        return
      }
      const id = String(++_reqId)
      const timer = setTimeout(() => {
        pendingRequests.delete(id)
        reject(new Error('请求超时'))
      }, timeout)
      pendingRequests.set(id, { resolve, reject, timer })
      ws.send(JSON.stringify({ action, id, data }))
    })
  }

  function refreshContainers() {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ action: 'containers:refresh', id: String(++_reqId) }))
    }
  }

  // 注册设备鉴权 401 回调
  function onAuth401(fn) { _auth401Listeners.add(fn) }
  function offAuth401(fn) { _auth401Listeners.delete(fn) }

  function triggerAuth401() {
    _auth401Listeners.forEach(fn => { try { fn() } catch {} })
  }

  // 等待设备鉴权密码确认（返回 Promise，密码保存后 resolve）
  let _auth401Resolve = null
  function waitDeviceAuth() {
    return new Promise(resolve => {
      _auth401Resolve = resolve
      triggerAuth401()
    })
  }
  function resolveDeviceAuth() {
    if (_auth401Resolve) { _auth401Resolve(); _auth401Resolve = null }
  }

  function displayName(name) {
    if (!name) return ''
    return containerAliases.value[name] || name
  }

  async function setAlias(name, alias) {
    await request('alias:set', { name, alias })
    containerAliases.value = { ...containerAliases.value, [name]: alias }
  }

  async function removeAlias(name) {
    await request('alias:delete', { name })
    const copy = { ...containerAliases.value }
    delete copy[name]
    containerAliases.value = copy
  }

  function onEvent(listener) { eventListeners.add(listener) }
  function offEvent(listener) { eventListeners.delete(listener) }

  async function requestProjectionToken(containerName) {
    const auth = useAuthStore()
    const resp = await request('projection:token', { container_name: containerName })
    const data = resp.data
    if (data?.encrypted && data?.data) {
      if (auth.sessionKey) {
        try {
          const plaintext = await aesDecrypt(auth.sessionKey, data.data)
          const parsed = JSON.parse(plaintext)
          return { token: parsed.token, udpPort: parsed.udpPort || '' }
        } catch { console.warn('[投屏] sessionKey 解密失败') }
      }
      console.warn('[投屏] 无 sessionKey，无法解密投屏 token')
      return null
    }
    return { token: data?.token, udpPort: data?.udpPort || '' }
  }

  return {
    status, online, containers, containerAliases, screenshots,
    connect, disconnect, request, refreshContainers, requestProjectionToken,
    displayName, setAlias, removeAlias, onEvent, offEvent,
    onAuth401, offAuth401, waitDeviceAuth, resolveDeviceAuth,
    get _ws() { return ws },
  }
})
