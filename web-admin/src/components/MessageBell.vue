<template>
  <div class="message-bell">
    <el-popover v-model:visible="visible" placement="bottom-end" :width="380" trigger="click" @show="handleShow">
      <template #reference>
        <el-badge :value="messageStore.total" :hidden="!messageStore.hasUnread" :max="99" class="bell-badge">
          <el-tooltip :content="t('messageBell.tooltip')" placement="bottom">
            <el-button :icon="Message" circle :type="messageStore.hasUnread ? 'primary' : 'default'" :aria-label="t('messageBell.tooltip')" />
          </el-tooltip>
        </el-badge>
      </template>

      <div class="panel">
        <div class="panel-header">
          <h3>{{ t('messageBell.heading') }}</h3>
          <el-button v-if="items.length > 0" link type="primary" size="small" @click="handleMarkAll">
            {{ t('messageBell.markAll') }}
          </el-button>
        </div>

        <div v-loading="loading" class="list">
          <div
            v-for="item in items"
            :key="item.id"
            class="list-item"
            :class="{ unread: !item.read_at, [item.category]: true }"
            @click="handleOpen(item)"
          >
            <span class="category-dot" :style="{ backgroundColor: categoryColor(item.category) }" />
            <div class="item-content">
              <div class="item-title">{{ item.title }}</div>
              <div class="item-desc">{{ item.content }}</div>
              <div class="item-time">{{ formatTime(item.created_at) }}</div>
            </div>
            <div v-if="!item.read_at" class="unread-dot" />
          </div>
        </div>

        <el-empty v-if="!loading && items.length === 0" :description="t('messageBell.empty')" :image-size="60" />

        <div class="panel-footer">
          <el-button link type="primary" size="small" @click="goToInbox">{{ t('messageBell.viewAll') }}</el-button>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Message } from '@element-plus/icons-vue'
import { listMessages, markAllMessagesRead, markMessageRead, type MessageItem } from '@/api/modules/message'
import { useMessageStore } from '@/stores/message'
import { useI18n } from '@/composables/useI18n'
import { toPageResult } from '@/types/pagination'
import { clearMessageCategory } from '@/utils/messageCategory'

const router = useRouter()
const messageStore = useMessageStore()
const { t, locale } = useI18n()

const visible = ref(false)
const loading = ref(false)
const items = ref<MessageItem[]>([])

/** 分类 → 标识色，用于左侧色点快速区分消息来源 */
const CATEGORY_COLORS: Record<string, string> = {
  announcement: '#409eff',
  task: '#e6a23c',
  plan: '#67c23a',
  audit: '#f56c6c',
  system: '#909399'
}

function categoryColor(category: string): string {
  return CATEGORY_COLORS[category] || CATEGORY_COLORS.system
}

function formatTime(value: string | null | undefined): string {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const diff = Date.now() - d.getTime()
  if (diff < 60000) return t('time.justNow')
  if (diff < 3600000) return t('time.minutesAgo', Math.floor(diff / 60000))
  if (diff < 86400000) return t('time.hoursAgo', Math.floor(diff / 3600000))
  if (diff < 604800000) return t('time.daysAgo', Math.floor(diff / 86400000))
  return d.toLocaleDateString(locale.value)
}

async function loadPreview() {
  loading.value = true
  try {
    const raw = await listMessages({ page: 1, page_size: 6 })
    items.value = toPageResult<MessageItem>(raw).list
  } catch {
    // 请求失败提示由 axios 拦截器统一弹出，这里保留已有列表
  } finally {
    loading.value = false
  }
}

function handleShow() {
  loadPreview()
  messageStore.refreshUnread()
}

async function handleOpen(item: MessageItem) {
  visible.value = false
  if (!item.read_at) {
    try {
      await markMessageRead(item.id)
      item.read_at = new Date().toISOString()
      messageStore.decrementUnread(item.category)
      // 别处可能已标记过同一条（另一标签页/设备），合并拉一次服务端读数对账
      messageStore.refreshUnreadSoon()
    } catch {
      // 标记失败不阻断跳转：消息本身仍可读
    }
  }
  // link 由后端按业务写入（如 /task、/system/plan），缺失时回落到收件箱
  router.push(item.link || '/messages')
}

async function handleMarkAll() {
  try {
    const res = await markAllMessagesRead()
    items.value = items.value.map(i => ({ ...i, read_at: i.read_at || new Date().toISOString() }))
    messageStore.clearUnread(undefined, res?.updated)
    messageStore.refreshUnreadSoon()
  } catch {
    // 失败提示由拦截器统一处理
  }
}

function goToInbox() {
  visible.value = false
  // 收件箱会记住上次的分类页签；「查看全部消息」如果不先清掉，落地页仍是被筛过的列表
  clearMessageCategory()
  router.push('/messages')
}

onMounted(() => {
  messageStore.refreshUnread()
})
</script>

<style scoped>
.message-bell {
  display: inline-flex;
  align-items: center;
}
.bell-badge :deep(.el-badge__content) {
  background-color: #f56c6c;
}
.panel {
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
  font-size: 15px;
  font-weight: 600;
}
.list {
  max-height: 320px;
  min-height: 60px;
  overflow-y: auto;
  margin-top: 8px;
}
.list-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 8px;
  border-radius: 6px;
  cursor: pointer;
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
.category-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 7px;
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
.unread-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #409eff;
  flex-shrink: 0;
  margin-top: 6px;
}
.panel-footer {
  text-align: center;
  padding-top: 8px;
  margin-top: 4px;
  border-top: 1px solid #f0f0f0;
}
</style>
