<template>
  <div class="notification-center">
    <el-popover
      v-model:visible="popoverVisible"
      placement="bottom-end"
      :width="400"
      trigger="click"
    >
      <template #reference>
        <el-badge :value="unreadCount" :hidden="unreadCount === 0" class="notification-badge">
          <el-button :icon="Bell" circle />
        </el-badge>
      </template>

      <div class="notification-panel">
        <div class="panel-header">
          <h3>通知中心</h3>
          <el-button link type="primary" @click="markAllAsRead">全部标为已读</el-button>
        </div>

        <el-divider style="margin: 8px 0" />

        <div class="notification-list">
          <div
            v-for="notification in notifications"
            :key="notification.id"
            class="notification-item"
            :class="{ unread: !notification.is_read }"
            @click="handleNotificationClick(notification)"
          >
            <div class="notification-icon">
              <el-icon :size="20" :color="getNotificationColor(notification.type)">
                <component :is="getNotificationIcon(notification.type)" />
              </el-icon>
            </div>
            <div class="notification-content">
              <div class="notification-title">{{ notification.title }}</div>
              <div class="notification-desc">{{ notification.content }}</div>
              <div class="notification-time">{{ formatTime(notification.created_at) }}</div>
            </div>
          </div>

          <el-empty v-if="notifications.length === 0" description="暂无通知" />
        </div>

        <div v-if="total > pageSize" class="panel-footer">
          <el-pagination
            v-model:current-page="page"
            :page-size="pageSize"
            :total="total"
            layout="prev, pager, next"
            small
            @current-change="loadNotifications"
          />
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Bell, Edit, ChatDotRound, Share, InfoFilled } from '@element-plus/icons-vue'
import {
  listNotifications,
  markNotificationAsRead,
  markAllNotificationsAsRead,
  getUnreadNotificationCount,
  type Notification
} from '@/api/modules/wiki'
import { ElMessage } from 'element-plus'

const popoverVisible = ref(false)
const notifications = ref<Notification[]>([])
const unreadCount = ref(0)
const page = ref(1)
const pageSize = 10
const total = ref(0)

onMounted(async () => {
  await loadNotifications()
  await loadUnreadCount()
})

async function loadNotifications() {
  try {
    const res = await listNotifications(page.value, pageSize)
    notifications.value = res.data ?? []
    total.value = res.pagination?.total ?? 0
  } catch (error) {
    ElMessage.error('加载通知失败')
  }
}

async function loadUnreadCount() {
  try {
    const res = await getUnreadNotificationCount()
    unreadCount.value = res.count
  } catch (error) {
    console.error('加载未读数量失败:', error)
  }
}

async function handleNotificationClick(notification: Notification) {
  if (!notification.is_read) {
    try {
      await markNotificationAsRead(notification.id)
      notification.is_read = true
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    } catch (error) {
      ElMessage.error('标记已读失败')
    }
  }
}

async function markAllAsRead() {
  try {
    await markAllNotificationsAsRead()
    notifications.value.forEach((n) => (n.is_read = true))
    unreadCount.value = 0
    ElMessage.success('已全部标为已读')
  } catch (error) {
    ElMessage.error('操作失败')
  }
}

function getNotificationIcon(type: string) {
  const icons: Record<string, any> = {
    edit: Edit,
    comment: ChatDotRound,
    share: Share,
    info: InfoFilled
  }
  return icons[type] || Bell
}

function getNotificationColor(type: string) {
  const colors: Record<string, string> = {
    edit: '#409eff',
    comment: '#67c23a',
    share: '#e6a23c',
    info: '#909399'
  }
  return colors[type] || '#409eff'
}

function formatTime(time: string) {
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  return date.toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.notification-center {
  display: inline-block;
}

.notification-badge {
  cursor: pointer;
}

.notification-panel {
  max-height: 500px;
  overflow-y: auto;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.notification-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.notification-item {
  display: flex;
  gap: 12px;
  padding: 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.notification-item:hover {
  background-color: #f5f7fa;
}

.notification-item.unread {
  background-color: #ecf5ff;
}

.notification-icon {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background-color: #f5f7fa;
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 4px;
}

.notification-desc {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-time {
  font-size: 12px;
  color: #9ca3af;
}

.panel-footer {
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
  display: flex;
  justify-content: center;
}
</style>