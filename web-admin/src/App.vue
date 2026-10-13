<template>
  <!-- Element Plus 按需引入后组件不再全局注册，用 ConfigProvider 统一注入当前语言包 -->
  <el-config-provider :locale="elLocale">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import enUs from 'element-plus/es/locale/lang/en'
import { ElNotification } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { useWebSocket, type WSMessage } from '@/composables/useWebSocket'
import { useI18n } from '@/composables/useI18n'
import { useTheme } from '@/composables/useTheme'
import { useBrand } from '@/composables/useBrand'

const { locale } = useI18n()
const elLocale = computed(() => (locale.value === 'en-US' ? enUs : zhCn))

// 确保主题与白标品牌在挂载时应用到位
useTheme()
const { applyBrand, syncBrandFromServer } = useBrand()
applyBrand()

const userStore = useUserStore()
const notificationStore = useNotificationStore()
const { connected, connect, disconnect, onMessage } = useWebSocket()

// 用户登录后自动连接 WebSocket，并同步服务端下发的租户品牌（多租户白标）
watch(() => userStore.token, (newToken) => {
  if (newToken) {
    connect()
    syncBrandFromServer()
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
      // 任务提醒（指派/到期/逾期）弹出轻量通知，不刷新公告；
      // 后端 WS 站内信统一以 announcement 消息下发，靠 payload.type 区分
      if (msg.payload?.type === 'task_assigned' || msg.payload?.type === 'task_due') {
        ElNotification({
          title: msg.payload.title || '任务提醒',
          message: msg.payload.content || '',
          type: msg.payload.type === 'task_due' ? 'warning' : 'info',
          duration: 6000
        })
      } else {
        // 收到新公告通知，刷新公告列表
        notificationStore.loadAnnouncements()
      }
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
  // 如果已登录，自动连接 WebSocket 并应用服务端品牌配置
  if (userStore.isLoggedIn) {
    connect()
    syncBrandFromServer()
  }
})

onUnmounted(() => {
  disconnect()
})
</script>

<style>
:root {
  --mx-header-bg: #ffffff;
  --mx-header-border: #e5e7eb;
  --mx-main-bg: #f3f4f6;
  --mx-text: #374151;
  --mx-text-strong: #111827;
  --mx-text-muted: #9ca3af;
  --mx-brand-primary: #409eff;
}
html.dark {
  --mx-header-bg: #1d1e1f;
  --mx-header-border: #303030;
  --mx-main-bg: #141414;
  --mx-text: #e5e7eb;
  --mx-text-strong: #f3f4f6;
  --mx-text-muted: #9ca3af;
}
html, body, #app {
  height: 100%;
  margin: 0;
  padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}
body {
  background: var(--mx-main-bg);
  color: var(--mx-text);
}
* {
  box-sizing: border-box;
}
</style>