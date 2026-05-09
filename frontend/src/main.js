import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { isMobile } from './utils/isMobile.js'

async function bootstrap() {
  if (isMobile) {
    // 移动端：加载 Vant + 暗色主题
    const [{ default: App }, { default: router }] = await Promise.all([
      import('./AppMobile.vue'),
      import('./router/index.js'),
    ])
    // vant CSS (移动端按需导入, 桌面端跳过)
    // await import('vant/lib/index.css')
    await import('./mobile-theme.css')

    const app = createApp(App)
    app.use(createPinia())
    app.use(router)
    app.mount('#app')
  } else {
    // 桌面端：加载 Element Plus + 全局主题
    const [{ default: App }, { default: router }] = await Promise.all([
      import('./App.vue'),
      import('./router/index.js'),
    ])
    await import('element-plus/theme-chalk/dark/css-vars.css')
    await import('./theme.css')
    // 命令式组件样式（ElMessage/ElMessageBox/ElDialog/ElNotification 不在模板中使用，按需导入无法自动检测）
    await import('element-plus/theme-chalk/el-message.css')
    await import('element-plus/theme-chalk/el-message-box.css')
    await import('element-plus/theme-chalk/el-dialog.css')
    await import('element-plus/theme-chalk/el-overlay.css')

    const app = createApp(App)
    app.use(createPinia())
    app.use(router)
    app.mount('#app')
  }
}

bootstrap()
