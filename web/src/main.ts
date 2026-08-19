import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerSW } from 'virtual:pwa-register'
import App from './App.vue'
import router from './router'
import { initTheme } from './theme'
import './style.css'

initTheme()

// PWA 自动更新:插件检测到新版本后自动激活新 Service Worker 并刷新页面,
// 避免用户一直使用缓存的旧版前端资源。
registerSW({
  immediate: true,
  onRegisteredSW: (_swUrl, registration) => {
    if (!registration) return
    // 定期检查更新,让长时间停留的页面也能发现新版本并自动刷新
    setInterval(() => registration.update(), 5 * 60 * 1000)
  },
})

createApp(App).use(createPinia()).use(router).mount('#app')
