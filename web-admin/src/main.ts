import { createApp } from 'vue'
import { createPinia } from 'pinia'

// Element Plus 函数式 API（Message / MessageBox / Notification）由 JS 动态挂载到 body，
// 不在模板中直接出现，无法被按需解析覆盖，其样式需要在此一次性引入
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import 'element-plus/es/components/notification/style/css'

// Element Plus 暗色主题 CSS 变量（配合 <html class="dark"> 生效）
import 'element-plus/theme-chalk/dark/css-vars.css'

// 首屏无闪烁：挂载前用持久化值同步 <html> 的主题/品牌/语言。
// 以下均为纯 DOM 引导函数，不依赖 Pinia；store 在组件 setup 中实例化时
// 会读取同一份 localStorage，与引导结果保持一致。
import { applyThemeToDOM, readStoredTheme } from '@/stores/theme'
import { applyBrandToDOM, readStoredBrand } from '@/stores/brand'
import { applyLocaleToDOM, readStoredLocale } from '@/stores/i18n'

import App from './App.vue'
import router from './router'

applyThemeToDOM(readStoredTheme())
applyBrandToDOM(readStoredBrand())
applyLocaleToDOM(readStoredLocale())

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')