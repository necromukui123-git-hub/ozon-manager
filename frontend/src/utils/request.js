import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'
import { createSystemLog } from '@/api/log'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
})

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
  error => {
    if (response) {
      switch (response.status) {
        case 401:
          localStorage.removeItem('token')
          localStorage.removeItem('user')
          router.push('/login')
          ElMessage.error('登录已过期，请重新登录')
          break
        case 403:
          ElMessage.error('权限不足')
          break
        case 404:
          ElMessage.error('资源不存在')
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

export default request
