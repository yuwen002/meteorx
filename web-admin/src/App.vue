<template>
  <!-- Element Plus 按需引入后组件不再全局注册，用 ConfigProvider 统一注入中文语言包 -->
  <el-config-provider :locale="zhCn">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { useWebSocket, type WSMessage } from '@/composables/useWebSocket'

const userStore = useUserStore()
const notificationStore = useNotificationStore()
const { connected, connect, disconnect, onMessage } = useWebSocket()

// 用户登录后自动连接 WebSocket
watch(() => userStore.token, (newToken) => {
  if (newToken) {
    connect()
  } else {
    disconnect()
  }
})

// 监听 WebSocket 消息
onMessage((msg: WSMessage) => {
  switch (msg.type) {
    case 'alert':
      // 收到告警通知
      notificationStore.addAlert({
        id: msg.payload.id,
        rule_name: msg.payload.rule_name,
        risk_level: msg.payload.risk_level,
        action: msg.payload.action,
        message: msg.payload.message,
        username: msg.payload.username,
        time: msg.payload.time
      })
      break

    case 'announcement':
      // 公告通知 - 可触发弹窗或页面刷新
      break

    case 'unread_count':
      // 未读数量更新
      break

    case 'pong':
      // 心跳响应，无需处理
      break
  }
})

// 监听 WebSocket 连接状态
watch(connected, (val) => {
  notificationStore.setWSConnected(val)
})

onMounted(() => {
  // 如果已登录，自动连接 WebSocket
  if (userStore.isLoggedIn) {
    connect()
  }
})

onUnmounted(() => {
  disconnect()
})
</script>

<style>
html, body, #app {
  height: 100%;
  margin: 0;
  padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
* {
  box-sizing: border-box;
}
</style>