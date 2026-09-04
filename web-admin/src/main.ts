import { createApp } from 'vue'
import { createPinia } from 'pinia'

// Element Plus 函数式 API（Message / MessageBox / Notification）由 JS 动态挂载到 body，
// 不在模板中直接出现，无法被按需解析覆盖，其样式需要在此一次性引入
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import 'element-plus/es/components/notification/style/css'

import App from './App.vue'
import router from './router'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')