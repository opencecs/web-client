import axios from 'axios'
import { useAuthStore } from '../stores/auth.js'
import router from '../router/index.js'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000
})

api.interceptors.request.use(config => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

api.interceptors.response.use(
  response => response,
  error => {
    // Don't intercept login/logout errors
    if (error.config?.url?.includes('/auth/login') || error.config?.url?.includes('/auth/logout')) {
      return Promise.reject(error)
    }
    if (error.response?.status === 401) {
      // 401 = 未登录/token过期，需要重新登录
      const auth = useAuthStore()
      auth.logout()
      router.push('/login')
    }
    // 403 = 无权限，不退出登录，由具体页面自行处理提示
    return Promise.reject(error)
  }
)

export default api
