<template>
  <div class="app-container dark">
    <template v-if="auth.isLoggedIn">
      <Sidebar @collapse-change="onSidebarCollapse" />
      <div class="main-wrapper" :style="{ marginLeft: sidebarWidth + 'px' }">
        <header class="app-header">
          <div class="ws-status" :class="device.online ? 'online' : 'offline'">
            <span class="ws-dot"></span>
            <span>{{ device.online ? '已连接' : '未连接' }}</span>
          </div>
          <el-dropdown trigger="click" @command="handleCommand">
            <span style="color: #bbb; cursor: pointer; display: flex; align-items: center; gap: 8px">
              <el-avatar :size="28" style="background: #409eff">{{ auth.username?.charAt(0)?.toUpperCase() }}</el-avatar>
              <span>{{ auth.username }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>
                  <el-tag size="small" :type="auth.role === 'admin' ? 'danger' : 'info'">{{ auth.role === 'admin' ? '管理员' : '用户' }}</el-tag>
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </header>
        <main class="app-main">
          <router-view />
        </main>
      </div>
    </template>
    <router-view v-else />
    <!-- 手机 UA 被强制桌面模式时，显示回切按钮 -->
    <div v-if="showMobileSwitch" class="mobile-switch-hint" @click="switchToMobile">
      切换到手机版
    </div>
  </div>
</template>

<script setup>
import { onMounted, computed, onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth.js'
import { useDeviceStore } from './stores/device.js'
import Sidebar from './components/Sidebar.vue'
import { ArrowDown } from '@element-plus/icons-vue'
import { checkIsMobile } from './utils/isMobile.js'

const router = useRouter()
const auth = useAuthStore()
const device = useDeviceStore()
const sidebarWidth = ref(64)

function onSidebarCollapse(collapsed) {
  sidebarWidth.value = collapsed ? 64 : 200
}

// 检测：手机 UA 但被 force_platform=desktop 强制到桌面版
const isMobileUA = /Android|iPhone|iPad|iPod|webOS|BlackBerry|IEMobile/i.test(navigator.userAgent)
const showMobileSwitch = computed(() => isMobileUA && localStorage.getItem('force_platform') === 'desktop')

function switchToMobile() {
  localStorage.removeItem('force_platform')
  window.location.href = '/m'
}

function handleCommand(cmd) {
  if (cmd === 'logout') {
    device.disconnect()
    auth.logout()
    router.push('/login')
  }
}

// 监听设备类型变化：PC↔手机切换时自动刷新
let lastMobile = false

function checkAndReload() {
  const nowMobile = checkIsMobile()
  if (nowMobile !== lastMobile) {
    window.location.reload()
  }
}

// resize：窗口大小变化（如DevTools切换设备模拟）
function onResize() {
  checkAndReload()
}

// storage：其他标签页修改 force_platform
function onStorage(e) {
  if (e.key === 'force_platform') {
    checkAndReload()
  }
}

let uaPollTimer = null

onMounted(() => {
  if (auth.isLoggedIn) {
    device.connect()
  }
  lastMobile = checkIsMobile()
  window.addEventListener('resize', onResize)
  window.addEventListener('storage', onStorage)
  // 轮询检测UA变化（DevTools切换设备模拟时UA会变但不触发事件）
  uaPollTimer = setInterval(checkAndReload, 1000)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  window.removeEventListener('storage', onStorage)
  if (uaPollTimer) clearInterval(uaPollTimer)
})
</script>

<style>
html, body {
  margin: 0;
  padding: 0;
  background: #0a0a0a;
  color: #e0e0e0;
}
html.dark {
  color-scheme: dark;
}
.app-container {
  min-height: 100vh;
}
.main-wrapper {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  transition: margin-left 0.2s;
}
.app-header {
  background: #141414;
  border-bottom: 1px solid #2a2a2a;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 50px;
  padding: 0 20px;
  position: sticky;
  top: 0;
  z-index: 100;
  flex-shrink: 0;
}
.app-main {
  flex: 1;
  padding: 0;
  background: #0a0a0a;
}
.ws-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #999;
}
.ws-status.online { color: #67c23a; }
.ws-status.offline { color: #f56c6c; }
.ws-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #999;
}
.ws-status.online .ws-dot { background: #67c23a; }
.ws-status.offline .ws-dot { background: #f56c6c; }
.mobile-switch-hint {
  position: fixed;
  bottom: 20px;
  right: 20px;
  background: #409eff;
  color: #fff;
  padding: 10px 18px;
  border-radius: 24px;
  font-size: 14px;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(64, 158, 255, 0.4);
  z-index: 99999;
}
.mobile-switch-hint:active {
  transform: scale(0.95);
}
</style>
