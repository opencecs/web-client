<template>
  <div class="login-container">
    <!-- 背景装饰 -->
    <div class="bg-decoration">
      <div class="bg-circle bg-circle-1"></div>
      <div class="bg-circle bg-circle-2"></div>
      <div class="bg-circle bg-circle-3"></div>
    </div>

    <div class="login-wrapper">
      <!-- Logo 区域 -->
      <div class="login-header">
        <div class="logo-icon">
          <img src="/favicon.ico" alt="魔云互联" class="logo-img" />
        </div>
        <h1 class="login-title">魔云互联</h1>
        <p class="login-subtitle">云手机管理平台</p>
      </div>

      <!-- 登录卡片 -->
      <el-card class="login-card" :body-style="{ padding: '32px 32px 24px' }">
        <el-form @submit.prevent="handleLogin" :model="form">
          <el-form-item>
            <el-input v-model="form.username" placeholder="请输入用户名" prefix-icon="User" size="large"
              :class="{ 'is-error': errorMsg }" @input="errorMsg = ''" />
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.password" placeholder="请输入密码" type="password" prefix-icon="Lock"
              size="large" show-password :class="{ 'is-error': errorMsg }"
              @keyup.enter="handleLogin" @input="errorMsg = ''" />
          </el-form-item>
          <div v-if="errorMsg" class="login-error">{{ errorMsg }}</div>
          <el-form-item style="margin-bottom: 8px">
            <div style="display: flex; align-items: center; width: 100%">
              <el-checkbox v-model="rememberMe" label="记住账号密码" />
            </div>
          </el-form-item>
          <el-form-item style="margin-bottom: 12px">
            <el-button type="primary" size="large" class="login-btn" :loading="loading" @click="handleLogin">
              {{ loading ? '登录中...' : '登 录' }}
            </el-button>
          </el-form-item>
        </el-form>
        <div class="login-hint">
          <el-icon :size="14" style="margin-right: 4px; vertical-align: -2px"><InfoFilled /></el-icon>
          默认管理账号：<span class="hint-code">myt</span> / <span class="hint-code">myt</span>
        </div>
      </el-card>

      <!-- 启动参数说明 -->
      <div class="startup-info">
        <div class="info-toggle" @click="showInfo = !showInfo">
          {{ showInfo ? '收起' : '使用说明' }}
        </div>
        <div v-if="showInfo" class="info-content">
          <div class="info-title">适配机型</div>
          <div class="info-note">目前仅适配 <b>C1 / Q1 / R1S</b> 最新固件</div>
          <div class="info-title" style="margin-top: 10px">自定义端口</div>
          <code class="info-code">./myt-panel -port 9090</code>
          <div class="info-note" style="margin-top: 6px">默认端口 <b>8181</b>，TCP/UDP 共用同一端口</div>
        </div>
      </div>

      <!-- 底部版本 -->
      <div class="login-footer">v{{ panelVersion }}</div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'
import { useDeviceStore } from '../stores/device.js'
import { InfoFilled } from '@element-plus/icons-vue'

const router = useRouter()
const auth = useAuthStore()
const device = useDeviceStore()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const rememberMe = ref(false)
const showInfo = ref(false)
const panelVersion = ref('...')
const errorMsg = ref('')

// 加载记住的用户名密码 + 获取版本号
onMounted(async () => {
  // 获取面板版本号（不需要登录）
  try {
    const resp = await fetch('/api/version')
    const data = await resp.json()
    panelVersion.value = data.version || 'dev'
  } catch {
    panelVersion.value = 'dev'
  }
  // 迁移旧格式（清除明文密码）
  const oldSaved = localStorage.getItem('saved_credentials')
  if (oldSaved) {
    try {
      const { username, password } = JSON.parse(oldSaved)
      if (username) localStorage.setItem('saved_username', username)
      if (password) localStorage.setItem('saved_password', btoa(password))
    } catch {}
    localStorage.removeItem('saved_credentials')
  }
  const savedUser = localStorage.getItem('saved_username')
  const savedPass = localStorage.getItem('saved_password')
  if (savedUser) {
    form.username = savedUser
    rememberMe.value = true
  }
  if (savedPass) {
    try { form.password = atob(savedPass) } catch {}
  }
})

async function handleLogin() {
  if (!form.username || !form.password) {
    errorMsg.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  errorMsg.value = ''
  try {
    await auth.login(form.username, form.password)
    // 记住/清除账号密码
    if (rememberMe.value) {
      localStorage.setItem('saved_username', form.username)
      localStorage.setItem('saved_password', btoa(form.password))
    } else {
      localStorage.removeItem('saved_username')
      localStorage.removeItem('saved_password')
    }
    device.connect()
    router.push('/')
  } catch (e) {
    const msg = e.response?.data?.error
    if (msg === 'invalid credentials') {
      errorMsg.value = '用户名或密码错误'
    } else if (msg === 'account disabled') {
      errorMsg.value = '账号已禁用'
    } else if (msg === 'account expired') {
      errorMsg.value = '账号已过期'
    } else {
      errorMsg.value = '登录失败'
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-default);
  position: relative;
  overflow: hidden;
}

/* 背景装饰圆 */
.bg-decoration {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}
.bg-circle {
  position: absolute;
  border-radius: 50%;
  filter: blur(100px);
  opacity: 0.12;
}
.bg-circle-1 {
  width: 500px; height: 500px;
  background: var(--accent);
  top: -150px; left: -100px;
  animation: float 8s ease-in-out infinite;
}
.bg-circle-2 {
  width: 350px; height: 350px;
  background: var(--success);
  bottom: -120px; right: -100px;
  animation: float 10s ease-in-out infinite reverse;
}
.bg-circle-3 {
  width: 200px; height: 200px;
  background: var(--warning);
  top: 50%; left: 65%;
  animation: float 12s ease-in-out infinite 2s;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-30px); }
}

.login-wrapper {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

/* Logo 头部 */
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
.logo-icon {
  margin-bottom: 14px;
}
.logo-img {
  width: 68px;
  height: 68px;
  object-fit: contain;
}
.login-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 6px;
  letter-spacing: 3px;
}
.login-subtitle {
  font-size: 13px;
  color: var(--text-tertiary);
  margin: 0;
  letter-spacing: 1px;
}

/* 卡片 */
.login-card {
  width: 380px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  position: relative;
  overflow: hidden;
}
.login-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--accent), transparent);
}

:deep(.el-input__wrapper) {
  background: var(--bg-elevated) !important;
  box-shadow: 0 0 0 1px var(--border-color) inset !important;
  border-radius: var(--radius-sm) !important;
  transition: box-shadow var(--transition-fast);
}
:deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--border-light) inset !important;
}
:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px var(--accent) inset !important;
}
:deep(.el-input__inner) {
  color: var(--text-primary) !important;
}
:deep(.el-input__prefix-inner .el-icon) {
  color: var(--text-tertiary);
}

.login-btn {
  width: 100%;
  font-size: 15px;
  letter-spacing: 4px;
  border-radius: var(--radius-sm);
  height: 42px;
  transition: all var(--transition-fast);
}
.login-btn:not(.is-loading):hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 16px rgba(64, 158, 255, 0.3);
}

.login-error {
  color: var(--danger);
  font-size: 13px;
  text-align: center;
  margin-bottom: 12px;
}

:deep(.is-error .el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--danger) inset !important;
}

/* 默认账号提示 */
.login-hint {
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
  padding: 12px 0 0;
  border-top: 1px solid var(--border-color);
}
.hint-code {
  display: inline-block;
  background: var(--bg-hover);
  color: var(--accent);
  padding: 1px 8px;
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 13px;
  font-weight: 600;
}

/* 启动参数说明 */
.startup-info {
  width: 380px;
  margin-top: 16px;
}
.info-toggle {
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary);
  cursor: pointer;
  padding: 4px 0;
  transition: color var(--transition-fast);
}
.info-toggle:hover { color: var(--accent); }
.info-content {
  margin-top: 8px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 14px 16px;
  font-size: 12px;
  color: var(--text-secondary);
}
.info-title {
  color: var(--text-primary);
  font-weight: 600;
  margin-bottom: 6px;
  font-size: 12px;
}
.info-code {
  display: block;
  background: var(--bg-elevated);
  color: var(--success);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: 12px;
  word-break: break-all;
}
.info-note {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.6;
}
.info-note b {
  color: var(--warning);
}

/* 底部版本号 */
.login-footer {
  margin-top: 24px;
  font-size: 11px;
  color: #777;
}
</style>
