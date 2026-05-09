<template>
  <el-dialog :modelValue="modelValue" @update:modelValue="$emit('update:modelValue', $event)"
    title="容器终端" width="850px" :close-on-click-modal="false" @close="cleanup" destroy-on-close
    style="--el-dialog-border-radius: 8px">
    <div v-if="container" style="margin-bottom: 8px">
      <el-tag size="small">{{ device.displayName(container.name) }}</el-tag>
    </div>
    <div ref="termRef" style="height: 450px; background: #0c0c0c; border-radius: 4px"></div>
  </el-dialog>
</template>

<script setup>
import { ref, watch, nextTick, onBeforeUnmount } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from '@xterm/addon-fit'
import 'xterm/css/xterm.css'
import { useDeviceStore } from '../../stores/device.js'

const device = useDeviceStore()

const props = defineProps({
  modelValue: Boolean,
  container: Object
})
const emit = defineEmits(['update:modelValue'])

const termRef = ref(null)
let term = null
let socket = null
let fitAddon = null
let heartbeat = null

watch(() => props.modelValue, async (val) => {
  if (val && props.container) {
    await nextTick()
    initTerminal()
  }
})

function initTerminal() {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    theme: { background: '#0c0c0c' },
    allowProposedApi: true
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(termRef.value)
  fitAddon.fit()
  term.focus()
  term.write('\r\n\x1b[32m正在连接容器终端...\x1b[0m\r\n')

  // 剪贴板辅助函数（兼容 HTTP 环境）
  function clipboardCopy(text) {
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(text).catch(() => fallbackCopy(text))
    } else {
      fallbackCopy(text)
    }
  }
  function fallbackCopy(text) {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }

  // 粘贴辅助函数（HTTPS环境用Clipboard API）
  function clipboardPaste() {
    if (navigator.clipboard?.readText) {
      navigator.clipboard.readText().then(text => {
        if (text && socket?.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'stdin', data: text }))
        }
      }).catch(() => {})
    }
  }

  // 原生 paste 事件
  termRef.value.addEventListener('paste', (e) => {
    const text = e.clipboardData?.getData('text')
    if (text && socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'stdin', data: text }))
    }
  })

  // 右键：有选中→复制；无选中→尝试读取剪贴板粘贴，失败则不拦截让浏览器菜单处理
  termRef.value.addEventListener('contextmenu', (e) => {
    const selection = term.getSelection()
    if (selection) {
      e.preventDefault()
      clipboardCopy(selection)
      return
    }
    // 无选中：尝试Clipboard API直接粘贴（HTTPS环境）
    if (navigator.clipboard?.readText) {
      e.preventDefault()
      navigator.clipboard.readText().then(text => {
        if (text && socket?.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'stdin', data: text }))
        }
      }).catch(() => {
        // HTTP环境无权限，释放右键让浏览器菜单弹出，用户点"粘贴"即可
      })
    }
    // HTTP环境不preventDefault，浏览器右键菜单含"粘贴"
  })

  // WebSocket 连接到后端 SDK 代理
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('token')
  socket = new WebSocket(`${proto}//${location.host}/api/sdk/link/exec?token=${token}`)
  socket.binaryType = 'arraybuffer'

  socket.onopen = () => {
    // 发送登录指令
    socket.send(JSON.stringify({
      type: 'login',
      container_id: props.container.name,
      shell: '/bin/sd'
    }))
    // 发送终端大小
    const dims = fitAddon.proposeDimensions()
    if (dims) {
      socket.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
    }
    // 心跳
    heartbeat = setInterval(() => {
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'heartbeat' }))
      }
    }, 30000)
  }

  socket.onmessage = (event) => {
    if (event.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(event.data))
    } else {
      term.write(event.data)
    }
  }

  socket.onclose = () => {
    term?.write('\r\n\x1b[31m连接已断开\x1b[0m\r\n')
  }

  socket.onerror = () => {
    term?.write('\r\n\x1b[31m连接错误\x1b[0m\r\n')
  }

  // 用户输入
  term.onData((data) => {
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'stdin', data }))
    }
  })

  // 窗口 resize
  window._containerTermResize = () => {
    if (fitAddon && term) {
      fitAddon.fit()
      const dims = fitAddon.proposeDimensions()
      if (dims && socket?.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
      }
    }
  }
  window.addEventListener('resize', window._containerTermResize)
}

function cleanup() {
  if (heartbeat) { clearInterval(heartbeat); heartbeat = null }
  if (socket) { socket.close(); socket = null }
  if (term) { term.dispose(); term = null }
  if (window._containerTermResize) {
    window.removeEventListener('resize', window._containerTermResize)
    window._containerTermResize = null
  }
  fitAddon = null
}

onBeforeUnmount(() => cleanup())
</script>
