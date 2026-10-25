<template>
  <div class="page">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <span>消息中心</span>
            <el-tag v-if="messageStore.total > 0" type="danger" size="small" round>
              {{ messageStore.total }} 条未读
            </el-tag>
          </div>
          <div class="header-actions">
            <el-button @click="openPreferences">偏好设置</el-button>
            <el-button :disabled="unreadOf(activeCategory) === 0" @click="handleMarkAll">
              {{ activeCategory ? '本分类全部已读' : '全部标为已读' }}
            </el-button>
            <el-button :loading="loading" @click="reload">刷新</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeCategory" class="category-tabs" @tab-change="handleCategoryChange">
        <el-tab-pane v-for="tab in categoryTabs" :key="tab.value" :name="tab.value">
          <template #label>
            <span class="tab-label">
              {{ tab.label }}
              <el-badge
                v-if="unreadOf(tab.value) > 0"
                :value="unreadOf(tab.value)"
                :max="99"
                class="tab-badge"
              />
            </span>
          </template>
        </el-tab-pane>
      </el-tabs>

      <div class="filter-bar">
        <el-input
          v-model="keyword"
          placeholder="搜索标题或内容"
          clearable
          style="width: 260px; margin-right: 12px;"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-checkbox v-model="unreadOnly" style="margin-right: 12px;" @change="handleSearch">仅看未读</el-checkbox>
        <el-checkbox
          :model-value="allSelected"
          :indeterminate="someSelected"
          :disabled="list.length === 0"
          style="margin-right: 12px;"
          @change="toggleSelectAll"
        >
          全选本页
        </el-checkbox>
        <el-button type="primary" @click="handleSearch">查询</el-button>
      </div>

      <div v-loading="loading" class="message-list">
        <div v-if="selected.length > 0" class="batch-bar">
          <span class="batch-count">已选 {{ selected.length }} 条</span>
          <el-button size="small" :loading="batchLoading" @click="handleBatchRead">批量已读</el-button>
          <el-button size="small" type="danger" plain :loading="batchLoading" @click="handleBatchDelete">
            批量删除
          </el-button>
          <el-button size="small" link @click="clearSelection">取消选择</el-button>
        </div>

        <el-checkbox-group v-model="selected">
          <div
            v-for="item in list"
            :key="item.id"
            class="message-item"
            :class="{ unread: !item.read_at }"
          >
            <el-checkbox :value="item.id" class="item-check" :disabled="batchLoading" />
            <span class="category-dot" :style="{ backgroundColor: categoryColor(item.category) }" />
            <div class="item-main">
              <div class="item-head">
                <span class="item-title" :class="{ clickable: !!item.link }" @click="openLink(item)">
                  {{ item.title }}
                </span>
                <el-tag size="small" :type="priorityTagType(item.priority)" effect="plain">
                  {{ categoryLabel(item.category) }}
                </el-tag>
              </div>
              <div class="item-content">{{ item.content }}</div>
              <div class="item-meta">
                <span>{{ formatTime(item.created_at) }}</span>
                <span v-if="item.link" class="item-link" @click="openLink(item)">查看详情</span>
              </div>
            </div>
            <div class="item-actions">
              <el-button v-if="!item.read_at" link type="primary" size="small" @click="handleRead(item)">
                标为已读
              </el-button>
              <el-button link type="danger" size="small" @click="handleDelete(item)">删除</el-button>
            </div>
          </div>
        </el-checkbox-group>

        <el-empty
          v-if="!loading && list.length === 0"
          :description="unreadOnly ? '没有未读消息' : '暂无消息'"
        />
      </div>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-dialog v-model="prefVisible" title="消息偏好设置" width="480px">
      <div v-loading="prefLoading" class="pref-list">
        <div v-for="item in preferences" :key="item.category" class="pref-row">
          <div class="pref-text">
            <span class="pref-label">{{ categoryLabel(item.category) }}</span>
            <span class="pref-desc">
              {{ item.mandatory ? '安全告警强制接收，不可关闭' : '关闭后该类新消息不再进入收件箱' }}
            </span>
          </div>
          <el-switch
            :model-value="!item.muted"
            :disabled="item.mandatory || prefSaving"
            @change="handleToggle(item, $event)"
          />
        </div>
        <el-empty v-if="!prefLoading && preferences.length === 0" description="暂无偏好数据" />
      </div>
      <template #footer>
        <el-button @click="prefVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteMessage,
  deleteMessages,
  getMessagePreferences,
  listMessages,
  markAllMessagesRead,
  markMessageRead,
  markMessagesRead,
  updateMessagePreference,
  type MessageItem,
  type MessagePreferenceItem
} from '@/api/modules/message'
import { useMessageStore } from '@/stores/message'
import { toPageResult } from '@/types/pagination'
import { readMessageCategory, saveMessageCategory } from '@/utils/messageCategory'

const router = useRouter()
const messageStore = useMessageStore()

/** 分类页签：value 为空表示不过滤（后端以空 category 表示全部） */
const categoryTabs = [
  { label: '全部', value: '' },
  { label: '系统', value: 'system' },
  { label: '公告', value: 'announcement' },
  { label: '任务', value: 'task' },
  { label: '套餐', value: 'plan' },
  { label: '告警', value: 'audit' }
] as const

const CATEGORY_LABELS: Record<string, string> = {
  system: '系统',
  announcement: '公告',
  task: '任务',
  plan: '套餐',
  audit: '告警'
}

const CATEGORY_COLORS: Record<string, string> = {
  announcement: '#409eff',
  task: '#e6a23c',
  plan: '#67c23a',
  audit: '#f56c6c',
  system: '#909399'
}

function categoryLabel(category: string): string {
  return CATEGORY_LABELS[category] || category
}

function categoryColor(category: string): string {
  return CATEGORY_COLORS[category] || CATEGORY_COLORS.system
}

/** 优先级：0 低 / 1 普通 / 2 高 / 3 紧急，仅用于标签配色 */
function priorityTagType(priority: number): 'info' | 'primary' | 'warning' | 'danger' {
  if (priority >= 3) return 'danger'
  if (priority === 2) return 'warning'
  if (priority === 1) return 'primary'
  return 'info'
}

function formatTime(value: string | null | undefined): string {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString('zh-CN')
}

const loading = ref(false)
const list = ref<MessageItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

/**
 * 分类页签跨会话记住：用户常按「任务」「告警」定点看消息，刷新一次就回到「全部」等于每次重选。
 * 只存页签值，关键词与「仅看未读」属于一次性筛选，不跨会话保留。
 */
const activeCategory = ref<string>(readMessageCategory(categoryTabs.map(tab => tab.value)))
const unreadOnly = ref(false)
const keyword = ref('')

/** 批量操作：只保存当前页选中的 ID，切换分页/筛选时清空 */
const selected = ref<string[]>([])
const batchLoading = ref(false)

const allSelected = computed(() => list.value.length > 0 && selected.value.length === list.value.length)
const someSelected = computed(
  () => selected.value.length > 0 && selected.value.length < list.value.length
)

function toggleSelectAll(checked: boolean | string | number) {
  selected.value = checked === true ? list.value.map(item => item.id) : []
}

function clearSelection() {
  selected.value = []
}

function unreadOf(category: string): number {
  return category === '' ? messageStore.total : messageStore.categoryUnread(category)
}

async function loadList() {
  loading.value = true
  try {
    const raw = await listMessages({
      page: page.value,
      page_size: pageSize.value,
      category: activeCategory.value || undefined,
      unread: unreadOnly.value || undefined,
      keyword: keyword.value.trim() || undefined
    })
    const result = toPageResult<MessageItem>(raw)
    list.value = result.list
    total.value = result.total
    // 列表内容变了，旧选中项可能已不在当前页：留着会让批量操作带上看不见的消息
    selected.value = []
  } catch {
    // 失败提示由 axios 拦截器统一弹出
  } finally {
    loading.value = false
  }
}

/** 刷新列表与角标：本地乐观更新可能已与服务端不一致，回到列表页以服务端为准 */
function reload() {
  loadList()
  messageStore.refreshUnread()
}

function handleSearch() {
  page.value = 1
  loadList()
}

function handleCategoryChange() {
  page.value = 1
  saveMessageCategory(activeCategory.value)
  loadList()
}

function openLink(item: MessageItem) {
  if (!item.link) return
  // 点开即视为已读：跳走再回来时这条不应还挂在未读里
  if (!item.read_at) handleRead(item)
  router.push(item.link)
}

async function handleRead(item: MessageItem) {
  try {
    await markMessageRead(item.id)
    item.read_at = new Date().toISOString()
    messageStore.decrementUnread(item.category)
    // 别处可能已标记过同一条（另一标签页/设备），合并拉一次服务端读数对账
    messageStore.refreshUnreadSoon()
    // 「仅看未读」下列表只收未读：标记后必须立刻从当前页移除，
    // 否则筛选条件与展示自相矛盾，看起来像没生效
    if (unreadOnly.value) {
      list.value = list.value.filter(m => m.id !== item.id)
      total.value = Math.max(total.value - 1, 0)
      if (list.value.length === 0 && page.value > 1) {
        page.value -= 1
        loadList()
      }
    }
  } catch {
    // 失败提示由拦截器统一处理
  }
}

async function handleMarkAll() {
  const category = activeCategory.value || undefined
  const unread = unreadOf(activeCategory.value)
  // 已读不可撤回，一次清空整类前先把范围与条数说清楚
  const scope = category ? `「${categoryLabel(category)}」分类` : '全部'
  try {
    await ElMessageBox.confirm(
      `确定把${scope}的 ${unread} 条未读消息全部标为已读？`,
      '标记已读',
      { type: 'info' }
    )
  } catch {
    return // 用户取消
  }

  try {
    const res = await markAllMessagesRead(category)
    // 按服务端返回的 updated 扣减：分类明细可能已被 WS 总数覆盖而过期
    messageStore.clearUnread(category, res?.updated)
    messageStore.refreshUnreadSoon()
    ElMessage.success(res?.updated ? `已标记 ${res.updated} 条为已读` : '当前没有未读消息')
    loadList()
  } catch {
    // 失败提示由拦截器统一处理
  }
}

async function handleDelete(item: MessageItem) {
  try {
    await ElMessageBox.confirm('确定删除这条消息？', '删除确认', { type: 'warning' })
    await deleteMessage(item.id)
    if (!item.read_at) {
      messageStore.decrementUnread(item.category)
      messageStore.refreshUnreadSoon()
    }
    ElMessage.success('已删除')
    // 删除后当前页可能已空，回退一页避免看到空列表
    if (list.value.length === 1 && page.value > 1) {
      page.value -= 1
    }
    loadList()
  } catch {
    // 用户取消，或请求失败（提示由拦截器统一弹出）
  }
}

/**
 * 批量标记已读。
 * 不按本地选中数做乐观扣减：选中项里可能有别的终端已读过的，
 * 实际清除条数以服务端 `updated` 为准，角标直接重拉一次完整读数。
 */
async function handleBatchRead() {
  const ids = [...selected.value]
  if (ids.length === 0) return
  batchLoading.value = true
  try {
    const res = await markMessagesRead(ids)
    messageStore.refreshUnread()
    ElMessage.success(res?.updated ? `已标记 ${res.updated} 条为已读` : '选中消息已是已读状态')
    loadList()
  } catch {
    // 失败提示由拦截器统一处理
  } finally {
    batchLoading.value = false
  }
}

async function handleBatchDelete() {
  const ids = [...selected.value]
  if (ids.length === 0) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${ids.length} 条消息？`, '批量删除确认', { type: 'warning' })
  } catch {
    return // 用户取消
  }

  batchLoading.value = true
  try {
    const res = await deleteMessages(ids)
    // 删掉的可能包含未读，条数与分类都无法从前端推算，直接以服务端读数为准
    messageStore.refreshUnread()
    ElMessage.success(res?.deleted ? `已删除 ${res.deleted} 条` : '选中消息已不存在')
    // 整页被删空时回退一页，避免停在空列表
    if (list.value.length === ids.length && page.value > 1) {
      page.value -= 1
    }
    loadList()
  } catch {
    // 失败提示由拦截器统一处理
  } finally {
    batchLoading.value = false
  }
}

const prefVisible = ref(false)
const prefLoading = ref(false)
const prefSaving = ref(false)
const preferences = ref<MessagePreferenceItem[]>([])

/** 打开偏好弹窗时拉一次服务端状态：弹窗里的开关必须以服务端为准 */
async function openPreferences() {
  prefVisible.value = true
  prefLoading.value = true
  try {
    const res = await getMessagePreferences()
    preferences.value = res?.items ?? []
  } catch {
    // 失败提示由 axios 拦截器统一弹出
  } finally {
    prefLoading.value = false
  }
}

/**
 * 切换分类接收开关。
 * 不做本地乐观切换：关闭接收会让后续消息直接不落库，
 * 必须等服务端确认；失败时开关保持原状态。
 */
async function handleToggle(item: MessagePreferenceItem, value: string | number | boolean) {
  if (item.mandatory) return
  const receive = value === true
  prefSaving.value = true
  try {
    const res = await updateMessagePreference(item.category, !receive)
    preferences.value = res?.items ?? preferences.value
    const label = categoryLabel(item.category)
    ElMessage.success(receive ? `已开启「${label}」接收` : `已关闭「${label}」接收`)
    messageStore.refreshUnread()
  } catch {
    // 失败保持原状态（后端拒绝时前端不能显示已生效）
  } finally {
    prefSaving.value = false
  }
}

onMounted(() => {
  loadList()
  messageStore.refreshUnread()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.category-tabs :deep(.el-tabs__header) {
  margin-bottom: 8px;
}
.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.tab-badge :deep(.el-badge__content) {
  position: static;
  transform: none;
  background-color: #f56c6c;
}
.filter-bar {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.message-list {
  min-height: 160px;
}
.batch-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  margin-bottom: 4px;
  background-color: #f5f7fa;
  border-radius: 4px;
}
.batch-count {
  font-size: 13px;
  color: #606266;
}
.message-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 12px;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.2s;
}
.item-check {
  height: auto;
  margin-top: 3px;
}
.message-item:hover {
  background-color: #fafafa;
}
.message-item.unread {
  background-color: #f7fbff;
}
.category-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 7px;
}
.item-main {
  flex: 1;
  min-width: 0;
}
.item-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}
.item-title {
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
}
.item-title.clickable {
  cursor: pointer;
}
.item-title.clickable:hover {
  color: #409eff;
}
.item-content {
  font-size: 13px;
  color: #6b7280;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}
.item-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 6px;
  font-size: 12px;
  color: #9ca3af;
}
.item-link {
  color: #409eff;
  cursor: pointer;
}
.item-actions {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.pref-list {
  min-height: 120px;
}
.pref-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid #f0f0f0;
}
.pref-row:last-child {
  border-bottom: none;
}
.pref-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.pref-label {
  font-size: 14px;
  color: #1f2937;
}
.pref-desc {
  font-size: 12px;
  color: #9ca3af;
}
</style>
