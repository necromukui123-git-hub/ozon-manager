import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

import App from './App.vue'
import router from './router'
import { initTheme } from './utils/theme'
import { createSystemLog } from '@/api/log'

import './styles/main.scss'

const app = createApp(App)

// 注册Element Plus图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn })

initTheme()

// ==========================================
// 全局错误捕获机制
// ==========================================
// 1. 捕获 Vue 渲染与组件生命周期内的运行错误
app.config.errorHandler = (err, instance, info) => {
  console.error('Vue Error Captured:', err, info)

  createSystemLog({
    level: 'error',
    message: `[Vue Error] ${err.name}: ${err.message}`,
    url: window.location.href,
    stack: `Info: ${info}\nStack: ${err.stack}`
  })
}

// 2. 捕获未处理的 Promise 拒绝 (例如未 catch 的异步抛错)
window.addEventListener('unhandledrejection', event => {
  console.error('Unhandled Promise Rejection:', event.reason)

  let msg = 'Unknown Promise Error'
  let stack = ''

  if (event.reason instanceof Error) {
    msg = event.reason.message
    stack = event.reason.stack
  } else if (typeof event.reason === 'string') {
    msg = event.reason
  } else {
    msg = JSON.stringify(event.reason)
  }

  createSystemLog({
    level: 'error',
    message: `[Unhandled Rejection] ${msg}`,
    url: window.location.href,
    stack: stack
  })
})

app.mount('#app')
