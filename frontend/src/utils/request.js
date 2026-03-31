import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { createSystemLog } from '@/api/log'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  withCredentials: true
})

const refreshClient = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  withCredentials: true
})

const LOGIN_URL = '/auth/login'
const REFRESH_URL = '/auth/refresh'
const LOGOUT_URL = '/auth/logout'
const SYSTEM_LOG_URL = '/system/logs'

let refreshPromise = null
let authRedirecting = false

// 请求拦截器
request.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  response => {
    return response.data
  },
  async error => {
    const { response, config = {} } = error

    if (response) {
      switch (response.status) {
        case 401:
          if (config.url === LOGIN_URL) {
            ElMessage.error(response.data?.message || '用户名或密码错误')
            break
          }
          if (config.url === SYSTEM_LOG_URL) {
            break
          }
          if (config.url === LOGOUT_URL) {
            clearLocalAuthState()
            break
          }
          if (config.url === REFRESH_URL || config._retry) {
            clearLocalAuthState()
            redirectToLogin(response.data?.message || '登录已过期，请重新登录')
            break
          }

          try {
            await ensureRefreshed()
            config._retry = true
            config.headers = config.headers || {}
            const token = localStorage.getItem('token')
            if (token) {
              config.headers.Authorization = `Bearer ${token}`
            }
            return request(config)
          } catch (refreshError) {
            clearLocalAuthState()
            redirectToLogin(refreshError.response?.data?.message || '登录已过期，请重新登录')
            return Promise.reject(refreshError)
          }
        case 403:
          ElMessage.error('权限不足')
          if (!config.silent) {
            createSystemLog({
              level: 'warn',
              message: `API Forbidden [403]: ${response.data?.message || response.statusText}`,
              url: config.url,
              stack: JSON.stringify(config.data || config.params)
            })
          }
          break
        case 404:
          ElMessage.error('资源不存在')
          if (!config.silent) {
            createSystemLog({
              level: 'warn',
              message: `API Not Found [404]: ${response.data?.message || response.statusText}`,
              url: config.url,
              stack: JSON.stringify(config.data || config.params)
            })
          }
          break
        case 500:
        case 502:
        case 503:
        case 504:
          ElMessage.error(response.data?.message || '服务器错误')
          // 记录服务器 5xx 错误
          if (!config.silent) {
            createSystemLog({
              level: 'error',
              message: `API Server Error [${response.status}]: ${response.data?.message || response.statusText}`,
              url: config.url,
              stack: JSON.stringify(config.data || config.params)
            })
          }
          break
        default:
          ElMessage.error(response.data?.message || '请求失败')
          if (!config.silent) {
            createSystemLog({
              level: 'warn',
              message: `API Error [${response.status}]: ${response.data?.message || response.statusText}`,
              url: config.url,
              stack: JSON.stringify(config.data || config.params)
            })
          }
      }
    } else {
      ElMessage.error('网络连接失败')
      // 记录完全断网或无响应的错误
      if (!config.silent) {
        createSystemLog({
          level: 'error',
          message: `Network Error or Timeout: ${error.message}`,
          url: config.url,
          stack: error.stack || ''
        })
      }
    }

    return Promise.reject(error)
  }
)

function ensureRefreshed() {
  if (!refreshPromise) {
    refreshPromise = refreshClient.post(REFRESH_URL)
      .then(response => {
        const payload = response.data?.data
        if (!payload?.token) {
          throw new Error('refresh response missing token')
        }

        localStorage.setItem('token', payload.token)
        if (payload.token_expires_at) {
          localStorage.setItem('tokenExpiresAt', payload.token_expires_at)
        } else {
          localStorage.removeItem('tokenExpiresAt')
        }

        if (payload.user) {
          localStorage.setItem('user', JSON.stringify(payload.user))
        }

        window.dispatchEvent(new CustomEvent('auth:updated'))
        return payload
      })
      .finally(() => {
        refreshPromise = null
      })
  }

  return refreshPromise
}

function clearLocalAuthState() {
  localStorage.removeItem('token')
  localStorage.removeItem('tokenExpiresAt')
  localStorage.removeItem('user')
  localStorage.removeItem('currentShopId')
  window.dispatchEvent(new Event('auth:cleared'))
}

function redirectToLogin(message) {
  if (authRedirecting) {
    return
  }

  authRedirecting = true
  if (router.currentRoute.value.path !== '/login') {
    router.push('/login').finally(() => {
      authRedirecting = false
    })
  } else {
    authRedirecting = false
  }
  ElMessage.error(message)
}

export default request
