<template>
  <div class="notification-center">
    <el-popover
      v-model:visible="popoverVisible"
      placement="bottom-end"
      :width="420"
      trigger="click"
      @show="handleShow"
    >
      <template #reference>
        <el-badge :value="notificationStore.totalUnread" :hidden="!notificationStore.hasUnread" class="header-badge">
          <el-tooltip content="通知中心" placement="bottom">
            <el-button :icon="Bell" circle :type="notificationStore.hasUnread ? 'danger' : 'default'" />
          </el-tooltip>
        </el-badge>
      </template>

      <div class="panel">
        <!-- 面板头部 -->
        <div class="panel-header">
          <h3>通知中心</h3>
          <div class="header-actions">
            <el-button
              v-if="notificationStore.hasUnread"
              link
              type="primary"
              size="small"
              @click="notificationStore.markAllAsRead()"
            >
              全部标为已读
            </el-button>
          </div>
        </div>

        <!-- 标签切换 -->
        <el-tabs v-model="activeTab" class="panel-tabs">
          <el-tab-pane label="公告" name="announcement" />
          <el-tab-pane label="告警" name="alert" />
        </el-tabs>

        <!-- 公告列表 -->
        <div v-show="activeTab === 'announcement'" class="list-container">
          <div v-loading="notificationStore.loading" class="list">
            <div
              v-for="item in notificationStore.announcements"
              :key="item.id"
              class="list-item"
              :class="{ unread: !item.is_read }"
              @click="handleAnnouncementClick(item)"
            >
              <div class="item-icon">
                <el-icon :size="18" color="#409eff"><Bell /></el-icon>
              </div>
              <div class="item-content">
                <div class="item-title">{{ item.title }}</div>
                <div class="item-desc">{{ item.content }}</div>
                <div class="item-time">{{ formatTime(item.published_at) }}</div>
              </div>
              <div v-if="!item.is_read" class="item-dot" />
            </div>
          </div>

          <el-empty v-if="!notificationStore.loading && notificationStore.announcements.length === 0" description="暂无公告" />

          <div class="panel-footer">
            <el-button link type="primary" size="small" @click="goToAnnouncement">
              查看全部公告
            </el-button>
          </div>
        </div>

        <!-- 告警列表 -->
        <div v-show="activeTab === 'alert'" class="list-container">
          <div class="list">
            <div
              v-for="item in notificationStore.alerts.slice(0, 10)"
              :key="item.id"
              class="list-item"
              @click="handleAlertClick(item)"
            >
              <div class="item-icon">
                <el-icon :size="18" :color="getRiskColor(item.risk_level)">
                  <Warning />
                </el-icon>
              </div>
              <div class="item-content">
                <div class="item-title">{{ item.rule_name }}</div>
                <div class="item-desc">{{ item.message }}</div>
                <div class="item-time">{{ formatTime(item.time) }}</div>
              </div>
            </div>
          </div>

          <el-empty v-if="notificationStore.alerts.length === 0" description="暂无告警" />

          <div class="panel-footer">
            <el-button link type="primary" size="small" @click="goToAlert">
              查看全部告警
            </el-button>
          </div>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Bell, Warning } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { useNotificationStore } from '@/stores/notification'

const router = useRouter()
const notificationStore = useNotificationStore()

const popoverVisible = ref(false)
const activeTab = ref('announcement')

function handleShow() {
  if (notificationStore.announcements.length === 0) {
    notificationStore.loadAnnouncements()
  }
}

function formatTime(time: string | number | undefined | null): string {
  if (!time) return ''
  const d = typeof time === 'number' ? new Date(time * 1000) : new Date(time)
  if (isNaN(d.getTime())) return ''
  const now = Date.now()
  const diff = now - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)} 天前`
  return d.toLocaleDateString('zh-CN')
}

function getRiskColor(level: string): string {
  const map: Record<string, string> = {
    critical: '#f56c6c',
    high: '#e6a23c',
    medium: '#409eff',
    low: '#909399'
  }
  return map[level] || '#909399'
}

function handleAnnouncementClick(item: any) {
  notificationStore.markAnnouncementAsRead(item.id)
  popoverVisible.value = false
  router.push('/announcement')
}

function handleAlertClick(item: any) {
  notificationStore.markAllAlertsAsRead()
  popoverVisible.value = false
  router.push('/system/audit/alert')
}

function goToAnnouncement() {
  popoverVisible.value = false
  router.push('/announcement')
}

function goToAlert() {
  notificationStore.markAllAlertsAsRead()
  popoverVisible.value = false
  router.push('/system/audit/alert')
}
</script>

<style scoped>
.notification-center {
  display: inline-flex;
  align-items: center;
}

.header-badge {
  margin-right: 4px;
}
.header-badge :deep(.el-badge__content) {
  background-color: #f56c6c;
}

.panel {
  max-height: 480px;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px;
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.panel-tabs {
  margin-top: -8px;
}

.panel-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.panel-tabs :deep(.el-tabs__nav-wrap::after) {
  height: 1px;
}

.list-container {
  max-height: 340px;
  overflow-y: auto;
}

.list {
  min-height: 60px;
}

.list-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 8px;
  cursor: pointer;
  border-radius: 6px;
  transition: background-color 0.2s;
  position: relative;
}

.list-item:hover {
  background-color: #f5f7fa;
}

.list-item.unread {
  background-color: #f0f7ff;
}

.list-item.unread:hover {
  background-color: #e6f0ff;
}

.item-icon {
  flex-shrink: 0;
  margin-top: 2px;
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-title {
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
  margin-bottom: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-desc {
  font-size: 12px;
  color: #6b7280;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin-bottom: 2px;
}

.item-time {
  font-size: 11px;
  color: #9ca3af;
}

.item-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #409eff;
  flex-shrink: 0;
  margin-top: 6px;
}

.panel-footer {
  text-align: center;
  padding: 8px 0 0;
  border-top: 1px solid #f0f0f0;
  margin-top: 4px;
}
</style>