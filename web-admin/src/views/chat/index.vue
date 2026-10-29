<template>
  <div class="chat-page">
    <!-- 左：会话列表 -->
    <aside class="conv-panel">
      <div class="conv-header">
        <span class="conv-title">{{ t('chat.conversations') }}</span>
        <el-button type="primary" size="small" :icon="Plus" @click="openMemberDialog">
          {{ t('chat.newChat') }}
        </el-button>
      </div>
      <div class="conv-list" v-loading="chatStore.loadingConversations">
        <el-empty
          v-if="!chatStore.conversations.length"
          :description="t('chat.noConversations')"
          :image-size="80"
        />
        <div
          v-for="conv in chatStore.conversations"
          :key="conv.id"
          class="conv-item"
          :class="{ active: conv.id === chatStore.activeId }"
          @click="chatStore.selectConversation(conv.id)"
        >
          <el-avatar :size="38" class="conv-avatar">{{ (conv.peer_name || '?').charAt(0) }}</el-avatar>
          <div class="conv-main">
            <div class="conv-line1">
              <span class="conv-name">
                {{ conv.peer_name || conv.peer_user_id }}
                <el-tag v-if="conv.type === 'group'" size="small" type="info" class="conv-tag">{{ t('chat.groupTag') }}</el-tag>
              </span>
              <span class="conv-time">{{ formatTime(conv.last_at) }}</span>
            </div>
            <div class="conv-line2">
              <span class="conv-preview">{{ conv.last_message }}</span>
              <el-badge v-if="conv.unread_count > 0" :value="conv.unread_count" :max="99" class="conv-unread" />
            </div>
          </div>
        </div>
      </div>
    </aside>

    <!-- 右：聊天面板 -->
    <section class="msg-panel">
      <template v-if="chatStore.activeConversation">
        <header class="msg-header">
          <el-avatar :size="32">{{ (chatStore.activeConversation.peer_name || '?').charAt(0) }}</el-avatar>
          <span class="msg-peer">{{ chatStore.activeConversation.peer_name || chatStore.activeConversation.peer_user_id }}</span>

          <!-- 群聊：成员数 + 成员管理/改名(群主)/退群 -->
          <template v-if="isActiveGroup">
            <el-tag size="small" type="info" class="head-tag">{{ t('chat.memberCount', chatStore.activeConversation.member_count ?? 0) }}</el-tag>
            <el-button link size="small" :icon="UserFilled" @click="openMemberDrawer">{{ t('chat.manageMembers') }}</el-button>
            <el-button v-if="isOwner" link size="small" :icon="EditPen" @click="doRename">{{ t('chat.rename') }}</el-button>
            <el-button link size="small" type="danger" @click="doLeave">{{ t('chat.leave') }}</el-button>
          </template>

          <!-- 私聊：对方在线状态点 -->
          <template v-else>
            <span v-if="peerOnline" class="presence-dot online" :title="t('chat.online')"></span>
            <span v-else class="presence-dot offline" :title="t('chat.offline')"></span>
          </template>

          <span v-if="activeTypingNames" class="typing-hint">{{ activeTypingNames }} {{ t('chat.typing') }}</span>
        </header>

        <div ref="scrollRef" class="msg-body" v-loading="chatStore.loadingMessages" @scroll="onBodyScroll">
          <el-empty v-if="!chatStore.messages.length" :description="t('chat.emptyMessages')" :image-size="80" />
          <!-- 滚动到顶加载更早历史的反馈：加载中 / 已到底 -->
          <div v-if="chatStore.loadingOlder" class="msg-older-tip">{{ t('chat.loadingOlder') }}</div>
          <div v-else-if="chatStore.messagePage > 1 && !chatStore.hasMoreMessages" class="msg-older-tip">
            {{ t('chat.noMoreMessages') }}
          </div>
          <div
            v-for="msg in chatStore.messages"
            :key="msg.id"
            class="msg-row"
            :class="{ mine: isMine(msg) }"
          >
            <div class="msg-bubble">
              <div class="msg-meta">{{ displayName(msg) }} · {{ formatClock(msg.created_at) }}</div>
              <div v-if="msg.recalled" class="msg-recalled">{{ t('chat.recalled') }}</div>
              <div v-else class="msg-text">{{ msg.content }}</div>
              <div v-if="canRecall(msg)" class="msg-actions">
                <el-button link size="small" :class="isMine(msg) ? 'recall-mine' : 'recall-theirs'" @click="doRecall(msg)">
                  {{ t('chat.recall') }}
                </el-button>
              </div>
            </div>
          </div>
        </div>

        <footer class="msg-input">
          <el-input
            v-model="draft"
            type="textarea"
            :rows="2"
            resize="none"
            :placeholder="t('chat.inputPlaceholder')"
            @keydown="onKeydown"
          />
          <div class="msg-input-actions">
            <el-button
              type="primary"
              :loading="chatStore.sending"
              :disabled="!draft.trim()"
              @click="doSend"
            >
              {{ t('chat.send') }}
            </el-button>
          </div>
        </footer>
      </template>

      <el-empty v-else :description="t('chat.selectHint')" class="msg-placeholder" />
    </section>

    <!-- 发起会话：选择同租户成员（私聊单选 / 群聊多选） -->
    <el-dialog v-model="memberDialogVisible" :title="t('chat.newChat')" width="460px">
      <el-radio-group v-model="startMode" class="start-mode">
        <el-radio-button value="direct">{{ t('chat.chatTypeDirect') }}</el-radio-button>
        <el-radio-button value="group">{{ t('chat.chatTypeGroup') }}</el-radio-button>
      </el-radio-group>

      <el-input
        v-if="startMode === 'group'"
        v-model="groupName"
        :placeholder="t('chat.groupNamePlaceholder')"
        maxlength="50"
        class="start-field"
      />

      <el-select
        v-if="startMode === 'direct'"
        v-model="pickedUserId"
        filterable
        remote
        reserve-keyword
        :remote-method="searchMembers"
        :loading="memberLoading"
        :placeholder="t('chat.memberSearchPlaceholder')"
        style="width: 100%"
      >
        <el-option
          v-for="m in memberOptions"
          :key="m.id"
          :label="m.nickname || m.username"
          :value="m.id"
        />
      </el-select>
      <el-select
        v-else
        v-model="pickedMemberIds"
        multiple
        filterable
        remote
        reserve-keyword
        :remote-method="searchMembers"
        :loading="memberLoading"
        :placeholder="t('chat.pickMembers')"
        style="width: 100%"
      >
        <el-option
          v-for="m in memberOptions"
          :key="m.id"
          :label="m.nickname || m.username"
          :value="m.id"
        />
      </el-select>

      <template #footer>
        <el-button @click="memberDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!canStart" :loading="starting" @click="doStart">
          {{ t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 群成员管理抽屉 -->
    <el-drawer v-model="memberDrawerVisible" :title="t('chat.membersTitle')" size="380px">
      <div class="member-toolbar">
        <el-select
          v-model="addPickedIds"
          multiple
          filterable
          remote
          reserve-keyword
          collapse-tags
          :remote-method="searchMembers"
          :loading="memberLoading"
          :placeholder="t('chat.addMembers')"
          class="member-add-select"
        >
          <el-option
            v-for="m in memberOptions"
            :key="m.id"
            :label="m.nickname || m.username"
            :value="m.id"
          />
        </el-select>
        <el-button type="primary" size="small" :disabled="!addPickedIds.length" :loading="addingMember" @click="doAddMembers">
          {{ t('chat.addMembers') }}
        </el-button>
      </div>
      <div class="member-list" v-loading="loadingMembers">
        <div v-for="m in activeMembers" :key="m.user_id" class="member-row">
          <el-avatar :size="28">{{ (m.name || m.user_id).charAt(0) }}</el-avatar>
          <span class="member-name">{{ m.name || m.user_id }}</span>
          <el-tag v-if="m.role === 'owner'" size="small" type="warning">{{ t('chat.roleOwner') }}</el-tag>
          <el-button
            v-if="canRemove(m)"
            link
            size="small"
            type="danger"
            class="member-remove"
            @click="doRemoveMember(m)"
          >
            {{ t('chat.removeMember') }}
          </el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { Plus, UserFilled, EditPen } from '@element-plus/icons-vue'
import { useI18n } from '@/composables/useI18n'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { getUserList, type UserItem } from '@/api/modules/user'
import { toPageResult } from '@/types/pagination'
import type { ChatMessageItem, MemberItem } from '@/api/modules/chat'

const { t } = useI18n()
const chatStore = useChatStore()
const userStore = useUserStore()

const draft = ref('')
const scrollRef = ref<HTMLElement | null>(null)

// 当前登录用户 ID，用于判定消息是否本人发送
const myId = () => userStore.userInfo?.id || ''
function isMine(msg: ChatMessageItem): boolean {
  return msg.sender_id === myId()
}
function displayName(msg: ChatMessageItem): string {
  return isMine(msg) ? t('chat.me') : msg.sender_name || msg.sender_id
}

// ---- 在线状态 / 输入中 ----
const peerOnline = computed(() => {
  const peer = chatStore.activeConversation?.peer_user_id
  return !!peer && chatStore.isOnline(peer)
})

// 当前会话是否为群聊
const isActiveGroup = computed(() => chatStore.activeConversation?.type === 'group')
// 当前用户是否为该群群主
const isOwner = computed(() => chatStore.activeConversation?.my_role === 'owner')

// 群成员缓存（userId -> 展示名），供输入中提示与成员抽屉复用
const activeMembers = ref<MemberItem[]>([])
const loadingMembers = ref(false)
const memberNameMap = computed<Record<string, string>>(() => {
  const map: Record<string, string> = {}
  for (const m of activeMembers.value) map[m.user_id] = m.name || m.user_id
  return map
})

// 当前会话内正在输入的成员展示名（逗号分隔）
const activeTypingNames = computed(() => {
  const ids = chatStore.typingUsers[chatStore.activeId] || []
  if (!ids.length) return ''
  return ids
    .map((id) => {
      if (!isActiveGroup.value) {
        const conv = chatStore.activeConversation
        return (conv?.peer_user_id === id ? conv?.peer_name : undefined) || id
      }
      return memberNameMap.value[id] || id
    })
    .join(', ')
})

// 选中群会话时拉取成员（同时为输入中提示提供名字）
watch(
  () => chatStore.activeId,
  (id) => {
    if (id && isActiveGroup.value) void refreshMembers()
    else activeMembers.value = []
  }
)

// ---- 消息撤回 ----
// 与后端 service.RecallWindow 保持一致：发送后 5 分钟内可撤回
const RECALL_WINDOW_MS = 5 * 60 * 1000

function canRecall(msg: ChatMessageItem): boolean {
  if (msg.recalled || msg.sender_id !== myId()) return false
  const d = toDate(msg.created_at)
  if (!d) return false
  return Date.now() - d.getTime() < RECALL_WINDOW_MS
}

async function doRecall(msg: ChatMessageItem) {
  try {
    await ElMessageBox.confirm(t('chat.recallConfirm'), t('common.tip'), { type: 'warning' })
  } catch {
    return // cancelled
  }
  const ok = await chatStore.recall(msg.id)
  if (ok) ElMessage.success(t('chat.recallSuccess'))
  else ElMessage.error(t('chat.recallFailed'))
}

// ---- 时间格式化 ----
function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n)
}
function toDate(raw?: string): Date | null {
  if (!raw) return null
  const d = new Date(raw.replace(' ', 'T'))
  return Number.isNaN(d.getTime()) ? null : d
}
function formatClock(raw?: string): string {
  const d = toDate(raw)
  if (!d) return ''
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}
function formatTime(raw?: string): string {
  const d = toDate(raw)
  if (!d) return ''
  const today = new Date()
  if (d.toDateString() === today.toDateString()) return formatClock(raw)
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

// ---- 发送 ----
async function doSend() {
  const text = draft.value.trim()
  if (!text) return
  const ok = await chatStore.send(text)
  if (ok) {
    draft.value = ''
    stopTyping()
    scrollToBottom()
  }
}
function onKeydown(e: KeyboardEvent) {
  // Enter 发送，Shift+Enter 换行
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    void doSend()
    return
  }
  notifyTypingOnce()
}

// 输入中上报：状态翻转必发，持续输入每 3s 最多一次，频率控制在前端
const TYPING_PING_INTERVAL_MS = 3000
let typingActive = false
let lastTypingPing = 0
let typingIdleTimer: ReturnType<typeof setTimeout> | null = null

function notifyTypingOnce() {
  const now = Date.now()
  if (!typingActive) {
    typingActive = true
    lastTypingPing = now
    chatStore.notifyTyping(true)
  } else if (now - lastTypingPing >= TYPING_PING_INTERVAL_MS) {
    lastTypingPing = now
    chatStore.notifyTyping(true)
  }
  if (typingIdleTimer) clearTimeout(typingIdleTimer)
  typingIdleTimer = setTimeout(stopTyping, TYPING_PING_INTERVAL_MS + 500)
}

function stopTyping() {
  if (typingIdleTimer) {
    clearTimeout(typingIdleTimer)
    typingIdleTimer = null
  }
  if (!typingActive) return
  typingActive = false
  chatStore.notifyTyping(false)
}

function scrollToBottom() {
  nextTick(() => {
    const el = scrollRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

// 滚动接近顶部时加载更早的历史消息；前插后回补滚动高度，保持视口停留在原消息上
const OLDER_SCROLL_THRESHOLD_PX = 60
let loadingOlderGuard = false
async function onBodyScroll() {
  const el = scrollRef.value
  if (!el || loadingOlderGuard) return
  if (el.scrollTop > OLDER_SCROLL_THRESHOLD_PX) return
  if (!chatStore.hasMoreMessages || chatStore.loadingOlder || chatStore.loadingMessages) return
  loadingOlderGuard = true
  try {
    const prevHeight = el.scrollHeight
    const prevTop = el.scrollTop
    const added = await chatStore.loadOlderMessages()
    if (added > 0) {
      await nextTick()
      // 前插内容抬高 scrollHeight，按差值补偿scrollTop避免视口跳转
      el.scrollTop = el.scrollHeight - prevHeight + prevTop
    }
  } finally {
    loadingOlderGuard = false
  }
}

// ---- 发起会话：成员选择（私聊/群聊） ----
const memberDialogVisible = ref(false)
const memberLoading = ref(false)
const memberOptions = ref<UserItem[]>([])
const startMode = ref<'direct' | 'group'>('direct')
const pickedUserId = ref('')
const pickedMemberIds = ref<string[]>([])
const groupName = ref('')
const starting = ref(false)

// 发起按钮可用条件：私聊需选中一人；群聊需群名 + 至少一名成员
const canStart = computed(() =>
  startMode.value === 'direct'
    ? !!pickedUserId.value
    : !!groupName.value.trim() && pickedMemberIds.value.length > 0
)

function openMemberDialog() {
  startMode.value = 'direct'
  pickedUserId.value = ''
  pickedMemberIds.value = []
  groupName.value = ''
  memberOptions.value = []
  memberDialogVisible.value = true
  void searchMembers('')
}

async function searchMembers(keyword: string) {
  memberLoading.value = true
  try {
    const res = await getUserList({ page: 1, page_size: 50, keyword: keyword || undefined })
    const self = myId()
    // getUserList 返回后端原生分页包裹 { data, pagination }，须经 toPageResult 归一化出 list
    memberOptions.value = toPageResult<UserItem>(res).list.filter((u) => u.id !== self)
  } catch {
    memberOptions.value = []
  } finally {
    memberLoading.value = false
  }
}

async function doStart() {
  if (!canStart.value) return
  starting.value = true
  try {
    const conv =
      startMode.value === 'direct'
        ? await chatStore.openWith(pickedUserId.value)
        : await chatStore.createGroup(groupName.value.trim(), pickedMemberIds.value)
    memberDialogVisible.value = false
    await chatStore.selectConversation(conv.id)
    scrollToBottom()
  } catch {
    ElMessage.error(startMode.value === 'direct' ? t('chat.createFailed') : t('chat.createGroupFailed'))
  } finally {
    starting.value = false
  }
}

// ---- 群成员管理 ----
const memberDrawerVisible = ref(false)
const addPickedIds = ref<string[]>([])
const addingMember = ref(false)

// 拉取当前群成员列表（选中群会话与增删后刷新）
async function refreshMembers() {
  const id = chatStore.activeId
  if (!id) return
  loadingMembers.value = true
  try {
    activeMembers.value = await chatStore.loadMembers(id)
  } catch {
    /* 忽略：成员列表为辅助信息，拉取失败不打断聊天主流程 */
  } finally {
    loadingMembers.value = false
  }
}

function openMemberDrawer() {
  addPickedIds.value = []
  memberDrawerVisible.value = true
  void refreshMembers()
}

// 可移除判定：仅群主，且不能移除 owner（含创建人自己）
function canRemove(m: MemberItem): boolean {
  return isOwner.value && m.role !== 'owner'
}

async function doAddMembers() {
  if (!addPickedIds.value.length) return
  addingMember.value = true
  try {
    await chatStore.addMembers(chatStore.activeId, addPickedIds.value)
    addPickedIds.value = []
    ElMessage.success(t('chat.addSuccess'))
    await refreshMembers()
  } catch {
    ElMessage.error(t('chat.addFailed'))
  } finally {
    addingMember.value = false
  }
}

async function doRemoveMember(m: MemberItem) {
  try {
    await ElMessageBox.confirm(t('chat.removeConfirm'), t('common.tip'), { type: 'warning' })
  } catch {
    return // cancelled
  }
  try {
    await chatStore.removeMember(chatStore.activeId, m.user_id)
    ElMessage.success(t('chat.removeSuccess'))
    await refreshMembers()
  } catch {
    ElMessage.error(t('chat.removeFailed'))
  }
}

async function doLeave() {
  const id = chatStore.activeId
  if (!id) return
  try {
    await ElMessageBox.confirm(t('chat.leaveConfirm'), t('common.tip'), { type: 'warning' })
  } catch {
    return // cancelled
  }
  try {
    await chatStore.leaveConversation(id)
    ElMessage.success(t('chat.leaveSuccess'))
  } catch {
    ElMessage.error(t('chat.leaveFailed'))
  }
}

async function doRename() {
  const id = chatStore.activeId
  if (!id) return
  let value = ''
  try {
    const r = await ElMessageBox.prompt(t('chat.renameTitle'), t('chat.rename'), {
      inputValue: chatStore.activeConversation?.peer_name || '',
      inputValidator: (v: string) => (v && v.trim() ? true : t('chat.groupNameRequired'))
    })
    value = ((r as { value: string }).value || '').trim()
  } catch {
    return // cancelled
  }
  if (!value) return
  try {
    await chatStore.rename(id, value)
    ElMessage.success(t('chat.renameSuccess'))
  } catch {
    ElMessage.error(t('chat.renameFailed'))
  }
}

onMounted(() => {
  void chatStore.loadConversations()
})

// 离开页面时收起输入中状态，避免对方一直看到「正在输入…」
onUnmounted(() => {
  stopTyping()
})
</script>

<style scoped>
.chat-page {
  display: flex;
  height: calc(100vh - 120px);
  background: var(--mx-header-bg);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}
.conv-panel {
  width: 300px;
  min-width: 240px;
  border-right: 1px solid var(--mx-header-border);
  display: flex;
  flex-direction: column;
}
.conv-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--mx-header-border);
}
.conv-title {
  font-weight: 600;
  color: var(--mx-text-strong);
}
.conv-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
}
.conv-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border-radius: 8px;
  cursor: pointer;
}
.conv-item:hover {
  background: var(--mx-main-bg);
}
.conv-item.active {
  background: var(--el-color-primary-light-9);
}
.conv-main {
  flex: 1;
  min-width: 0;
}
.conv-line1 {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.conv-name {
  font-weight: 600;
  color: var(--mx-text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.conv-time {
  font-size: 12px;
  color: var(--mx-text-muted);
  flex-shrink: 0;
}
.conv-line2 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 2px;
}
.conv-preview {
  font-size: 12px;
  color: var(--mx-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
.msg-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.msg-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--mx-header-border);
}
.msg-peer {
  font-weight: 600;
  color: var(--mx-text-strong);
}
.presence-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
.presence-dot.online {
  background: var(--el-color-success);
}
.presence-dot.offline {
  background: var(--mx-text-muted);
}
.typing-hint {
  font-size: 12px;
  color: var(--mx-text-muted);
  margin-left: auto;
}
.msg-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: var(--mx-main-bg);
}
.msg-older-tip {
  text-align: center;
  font-size: 12px;
  color: var(--mx-text-muted);
  padding: 4px 0 12px;
}
.msg-row {
  display: flex;
  margin-bottom: 14px;
}
.msg-row.mine {
  justify-content: flex-end;
}
.msg-bubble {
  max-width: 68%;
  padding: 8px 12px;
  border-radius: 10px;
  background: var(--mx-header-bg);
  border: 1px solid var(--mx-header-border);
}
.msg-row.mine .msg-bubble {
  background: var(--el-color-primary);
  color: #fff;
  border-color: var(--el-color-primary);
}
.msg-meta {
  font-size: 11px;
  opacity: 0.7;
  margin-bottom: 4px;
}
.msg-text {
  white-space: pre-wrap;
  word-break: break-word;
}
.msg-recalled {
  font-size: 12px;
  font-style: italic;
  opacity: 0.7;
}
.msg-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 4px;
}
.recall-theirs {
  color: var(--mx-text-muted);
}
.recall-mine {
  color: rgba(255, 255, 255, 0.85);
}
.recall-mine:hover {
  color: #fff;
}
.msg-input {
  border-top: 1px solid var(--mx-header-border);
  padding: 10px 14px;
}
.msg-input-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}
.msg-placeholder {
  margin: auto;
}
.conv-tag {
  margin-left: 6px;
  vertical-align: middle;
}
.head-tag {
  margin-left: 4px;
}
.start-mode {
  margin-bottom: 12px;
}
.start-field {
  margin-bottom: 12px;
}
.member-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 14px;
}
.member-add-select {
  flex: 1;
}
.member-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.member-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 4px;
  border-bottom: 1px solid var(--mx-header-border);
}
.member-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--mx-text-strong);
}
.member-remove {
  flex-shrink: 0;
}
</style>
