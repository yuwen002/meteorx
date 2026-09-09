import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { listTenantAnnouncements } from '@/api/modules/announcement'

export interface AlertNotification {
  id: string
  rule_name: string
  risk_level: string
  action: string
  message: string
  username: string
  time: number
}

export interface AnnouncementNotification {
  id: string
  title: string
  content: string
  published_at: string
  is_read: boolean
}

export const useNotificationStore = defineStore('notification', () => {
  // === 告警通知 ===
  const alerts = ref<AlertNotification[]>([])
  const unreadAlertCount = ref(0)
  const wsConnected = ref(false)

  const hasUnreadAlerts = computed(() => unreadAlertCount.value > 0)

  // === 公告通知 ===
  const announcements = ref<AnnouncementNotification[]>([])
  const unreadAnnouncementCount = ref(0)
  const loading = ref(false)

  const totalUnread = computed(() => unreadAlertCount.value + unreadAnnouncementCount.value)
  const hasUnread = computed(() => totalUnread.value > 0)

  // === 告警方法 ===
  function addAlert(alert: AlertNotification) {
    alerts.value.unshift(alert)
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

  // === 公告方法 ===
  async function loadAnnouncements() {
    loading.value = true
    try {
      const res = await listTenantAnnouncements({ page: 1, page_size: 5 })
      const items = res.data ?? []
      announcements.value = items.map((item: any) => ({
        id: item.id,
        title: item.title,
        content: item.content?.replace(/<[^>]*>/g, '').substring(0, 100) || '',
        published_at: item.publish_at || item.created_at,
        is_read: false
      }))
      unreadAnnouncementCount.value = announcements.value.length
    } catch {
      // 静默失败
    } finally {
      loading.value = false
    }
  }

  function markAnnouncementAsRead(id: string) {
    const item = announcements.value.find(a => a.id === id)
    if (item && !item.is_read) {
      item.is_read = true
      unreadAnnouncementCount.value = Math.max(0, unreadAnnouncementCount.value - 1)
    }
  }

  function markAllAnnouncementsAsRead() {
    announcements.value.forEach(a => { a.is_read = true })
    unreadAnnouncementCount.value = 0
  }

  // === 全局方法 ===
  function markAllAsRead() {
    markAllAlertsAsRead()
    markAllAnnouncementsAsRead()
  }

  function clearAll() {
    alerts.value = []
    announcements.value = []
    unreadAlertCount.value = 0
    unreadAnnouncementCount.value = 0
  }

  return {
    alerts,
    unreadAlertCount,
    wsConnected,
    hasUnreadAlerts,
    announcements,
    unreadAnnouncementCount,
    loading,
    totalUnread,
    hasUnread,
    addAlert,
    markAllAlertsAsRead,
    setWSConnected,
    loadAnnouncements,
    markAnnouncementAsRead,
    markAllAnnouncementsAsRead,
    markAllAsRead,
    clearAll
  }
})