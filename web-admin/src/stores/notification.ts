import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface AlertNotification {
  id: string
  rule_name: string
  risk_level: string
  action: string
  message: string
  username: string
  time: number
}

export const useNotificationStore = defineStore('notification', () => {
  const alerts = ref<AlertNotification[]>([])
  const unreadAlertCount = ref(0)
  const wsConnected = ref(false)

  const hasUnreadAlerts = computed(() => unreadAlertCount.value > 0)

  function addAlert(alert: AlertNotification) {
    alerts.value.unshift(alert)
    // 最多保留 100 条
    if (alerts.value.length > 100) {
      alerts.value = alerts.value.slice(0, 100)
    }
    unreadAlertCount.value++
  }

  function markAllAlertsAsRead() {
    unreadAlertCount.value = 0
  }

  function setWSConnected(connected: boolean) {
    wsConnected.value = connected
  }

  function clearAlerts() {
    alerts.value = []
    unreadAlertCount.value = 0
  }

  return {
    alerts,
    unreadAlertCount,
    wsConnected,
    hasUnreadAlerts,
    addAlert,
    markAllAlertsAsRead,
    setWSConnected,
    clearAlerts
  }
})