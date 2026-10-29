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
              <el-tag v-if="conv.mentioned" size="small" type="danger" class="conv-mention">{{ t('chat.mentionMe') }}</el-tag>
              <span class="conv-preview">{{ previewText(conv) }}</span>
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
              <!-- 图片消息：可预览则缩略图点击新窗口查看，否则降级为文件名卡片 -->
              <div v-else-if="msg.type === 'image'" class="msg-media">
                <a v-if="msg.url" :href="msg.url" target="_blank" rel="noopener">
                  <img :src="msg.url" :alt="mediaInfo(msg)?.name || ''" class="msg-image" loading="lazy" />
                </a>
                <div v-else class="msg-file-card">
                  <el-icon class="file-icon"><Picture /></el-icon>
                  <span class="file-name">{{ t('chat.imageMsg') }} {{ mediaInfo(msg)?.name || '' }}</span>
                </div>
              </div>
              <!-- 文件消息：名称 + 大小卡片，可下载时包链接 -->
              <div v-else-if="msg.type === 'file'" class="msg-file-card">
                <el-icon class="file-icon"><Document /></el-icon>
                <component :is="msg.url ? 'a' : 'span'" :href="msg.url" target="_blank" rel="noopener" class="msg-file-link">
                  <span class="file-name">{{ mediaInfo(msg)?.name || t('chat.fileMsg') }}</span>
                  <span v-if="mediaInfo(msg)?.size" class="file-size">{{ formatSize(mediaInfo(msg)!.size) }}</span>
                </component>
              </div>
              <div v-else class="msg-text">
                <span v-for="(seg, si) in renderSegments(msg)" :key="si" :class="{ mention: seg.hi }">{{ seg.text }}</span>
              </div>
              <div v-if="canRecall(msg)" class="msg-actions">
                <el-button link size="small" :class="isMine(msg) ? 'recall-mine' : 'recall-theirs'" @click="doRecall(msg)">
                  {{ t('chat.recall') }}
                </el-button>
              </div>
              <!-- 已读回执：仅在我发送的最后一条消息上展示 -->
              <div v-if="isMine(msg) && !msg.recalled && msg.id === lastMyMessageId" class="msg-read">
                {{ readLabel(msg) }}
              </div>
            </div>
          </div>
        </div>

        <footer class="msg-input">
          <div v-if="mentionMenuOpen" class="mention-menu">
            <div
              v-for="m in mentionCandidates"
              :key="m.user_id"
              class="mention-item"
              @mousedown.prevent="pickMention(m)"
            >
              <el-avatar :size="22">{{ (m.name || m.user_id).charAt(0) }}</el-avatar>
              <span>{{ m.name || m.user_id }}</span>
            </div>
            <div v-if="!mentionCandidates.length" class="mention-empty">{{ t('chat.mentionNoMatch') }}</div>
          </div>
          <el-input
            v-model="draft"
            type="textarea"
            :rows="2"
            resize="none"
            :placeholder="t('chat.inputPlaceholder')"
            @keydown="onKeydown"
          />
          <div class="msg-input-actions">
            <div class="msg-input-tools">
              <el-button link :icon="Picture" :title="t('chat.pickImage')" :disabled="uploading" @click="triggerPick('image')" />
              <el-button link :icon="Paperclip" :title="t('chat.pickFile')" :disabled="uploading" @click="triggerPick('file')" />
              <span v-if="uploading" class="uploading-tip">{{ t('chat.uploading') }}</span>
            </div>
            <el-button
              type="primary"
              :loading="chatStore.sending"
              :disabled="!draft.trim()"
              @click="doSend"
            >
              {{ t('chat.send') }}
            </el-button>
          </div>
          <input ref="imageInputRef" type="file" accept="image/*" class="hidden-file-input" @change="onFilePicked($event, 'image')" />
          <input ref="fileInputRef" type="file" class="hidden-file-input" @change="onFilePicked($event, 'file')" />
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
import { Plus, UserFilled, EditPen, Picture, Paperclip, Document } from '@element-plus/icons-vue'
import { useI18n } from '@/composables/useI18n'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { getUserList, type UserItem } from '@/api/modules/user'
import { uploadFile } from '@/api/modules/file'
import { toPageResult } from '@/types/pagination'
import type { ChatMessageItem, ConversationItem, MemberItem } from '@/api/modules/chat'

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

// ---- 已读回执 ----
// 仅在我发送的最后一条未撤回消息上展示已读状态，避免逐条刷屏
const lastMyMessageId = computed(() => {
  for (let i = chatStore.messages.length - 1; i >= 0; i--) {
    const m = chatStore.messages[i]
    if (m.sender_id === myId() && !m.recalled) return m.id
  }
  return ''
})

function readLabel(msg: ChatMessageItem): string {
  const n = chatStore.readCountAt(msg.created_at)
  // 私聊：对方已读/未读二态；群聊：仅在有成员已读时显示人数，未读不占位
  if (isActiveGroup.value) return n > 0 ? t('chat.readStatusCount', n) : ''
  return n > 0 ? t('chat.readStatusRead') : t('chat.readStatusUnread')
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

// ---- @提及 ----
// 待发送消息累积的 @提及用户 ID（发送后清空）
const pendingMentions = ref<string[]>([])
// @ 弹层开关与当前过滤词（@ 后紧跟的不含空白的片段）
const mentionMenuOpen = ref(false)
const mentionQuery = ref('')

// 候选成员：按当前群成员过滤掉自己，再按 mentionQuery 前缀/子串匹配展示名
const mentionCandidates = computed(() => {
  const q = mentionQuery.value.trim().toLowerCase()
  const self = myId()
  return activeMembers.value
    .filter((m) => m.user_id !== self)
    .filter((m) => !q || (m.name || '').toLowerCase().includes(q))
    .slice(0, 8)
})

// 输入变化时探测末尾是否正处于 @token 中（@ 前需为行首或空白）
watch(draft, (val) => {
  if (!isActiveGroup.value) {
    mentionMenuOpen.value = false
    return
  }
  const m = /(?:^|\s)@([^\s@]*)$/.exec(val)
  if (m) {
    mentionQuery.value = m[1]
    mentionMenuOpen.value = true
  } else {
    mentionMenuOpen.value = false
  }
})

// 选中候选：把末尾的 @query 替换为 @展示名 ，并记录其 ID
function pickMention(m: MemberItem) {
  const name = m.name || m.user_id
  draft.value = draft.value.replace(/@([^\s@]*)$/, '@' + name + ' ')
  if (!pendingMentions.value.includes(m.user_id)) pendingMentions.value.push(m.user_id)
  mentionMenuOpen.value = false
}

interface Seg {
  text: string
  hi: boolean
}

// 将消息正文按 @提及成员展示名切分为普通段与高亮段（无 @ 时单段）
function renderSegments(msg: ChatMessageItem): Seg[] {
  const content = msg.content || ''
  if (!msg.mention_ids) return [{ text: content, hi: false }]
  const names = msg.mention_ids
    .split(',')
    .map((id) => memberNameMap.value[id.trim()])
    .filter((n): n is string => !!n)
  if (!names.length) return [{ text: content, hi: false }]
  const marks: Array<[number, number]> = []
  for (const name of names) {
    const token = '@' + name
    let from = 0
    let idx = content.indexOf(token, from)
    while (idx >= 0) {
      marks.push([idx, idx + token.length])
      from = idx + token.length
      idx = content.indexOf(token, from)
    }
  }
  if (!marks.length) return [{ text: content, hi: false }]
  marks.sort((a, b) => a[0] - b[0])
  const segs: Seg[] = []
  let pos = 0
  for (const [s, e] of marks) {
    if (s < pos) continue // 跳过重叠区间
    if (s > pos) segs.push({ text: content.slice(pos, s), hi: false })
    segs.push({ text: content.slice(s, e), hi: true })
    pos = e
  }
  if (pos < content.length) segs.push({ text: content.slice(pos), hi: false })
  return segs
}

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

// ---- 图片/文件消息 ----
// 上传中状态与两个隐藏文件选择器（图片限定 accept）
const uploading = ref(false)
const imageInputRef = ref<HTMLInputElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

function triggerPick(kind: 'image' | 'file') {
  ;(kind === 'image' ? imageInputRef : fileInputRef).value?.click()
}

/** 选完文件：先传文件模块拿存储文件名，再以媒体 JSON 正文发送 image/file 消息 */
async function onFilePicked(e: Event, kind: 'image' | 'file') {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // 允许连续选同一文件
  if (!file || !chatStore.activeId) return
  uploading.value = true
  try {
    const resp = await uploadFile(file)
    const content = JSON.stringify({
      file: resp.file_name,
      name: resp.original_name || file.name,
      size: typeof resp.file_size === 'number' ? resp.file_size : file.size
    })
    const ok = await chatStore.send(content, '', kind)
    if (ok) scrollToBottom()
  } catch {
    ElMessage.error(t('chat.uploadFailed'))
  } finally {
    uploading.value = false
  }
}

/** 解析媒体消息正文元信息；非法/非媒体返回 null（渲染降级） */
function mediaInfo(msg: ChatMessageItem): { name: string; size: number } | null {
  if (msg.type !== 'image' && msg.type !== 'file') return null
  try {
    const p = JSON.parse(msg.content || '')
    if (typeof p?.file === 'string' && p.file) {
      return { name: typeof p.name === 'string' && p.name ? p.name : p.file, size: Number(p.size) || 0 }
    }
  } catch {
    /* 正文非法 JSON：按不可预览处理 */
  }
  return null
}

/** 字节数可读化（B/KB/MB/GB，保留一位小数） */
function formatSize(bytes: number): string {
  if (!bytes || bytes < 0) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let v = bytes
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${i === 0 ? v : v.toFixed(1)} ${units[i]}`
}

/** 会话列表摘要：媒体消息按类型展示占位标签 + 文件名，文本消息原样 */
function previewText(conv: ConversationItem): string {
  if (!conv.last_message) return ''
  if (conv.last_message_type === 'image' || conv.last_message_type === 'file') {
    const label = conv.last_message_type === 'image' ? t('chat.imageMsg') : t('chat.fileMsg')
    const info = mediaInfo({ type: conv.last_message_type, content: conv.last_message } as ChatMessageItem)
    return info?.name ? `${label} ${info.name}` : label
  }
  return conv.last_message
}

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
  const ok = await chatStore.send(text, pendingMentions.value.join(','))
  if (ok) {
    draft.value = ''
    pendingMentions.value = []
    mentionMenuOpen.value = false
    stopTyping()
    scrollToBottom()
  }
}
function onKeydown(e: KeyboardEvent) {
  // @ 弹层展开时，Enter 选中首个候选而非发送
  if (e.key === 'Enter' && !e.shiftKey && mentionMenuOpen.value && mentionCandidates.value.length) {
    e.preventDefault()
    pickMention(mentionCandidates.value[0])
    return
  }
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
  let value: string
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
.msg-read {
  font-size: 11px;
  opacity: 0.7;
  text-align: right;
  margin-top: 2px;
}
/* 图片消息缩略图 */
.msg-image {
  display: block;
  max-width: 240px;
  max-height: 240px;
  border-radius: 8px;
  cursor: zoom-in;
}
/* 文件/降级消息卡片 */
.msg-file-card {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 180px;
  max-width: 280px;
  padding: 8px 10px;
  border: 1px solid var(--mx-header-border);
  border-radius: 8px;
  background: var(--mx-main-bg);
}
.msg-row.mine .msg-file-card {
  background: rgba(255, 255, 255, 0.16);
  border-color: rgba(255, 255, 255, 0.35);
}
.msg-file-link {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: inherit;
  text-decoration: none;
}
a.msg-file-link:hover .file-name {
  text-decoration: underline;
}
.file-icon {
  flex-shrink: 0;
}
.file-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-size {
  flex-shrink: 0;
  font-size: 12px;
  opacity: 0.7;
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
  position: relative;
  border-top: 1px solid var(--mx-header-border);
  padding: 10px 14px;
}
/* @提及候选菜单：锚定在输入区上方弹出 */
.mention-menu {
  position: absolute;
  left: 14px;
  bottom: 100%;
  z-index: 10;
  min-width: 200px;
  max-height: 220px;
  overflow-y: auto;
  padding: 4px;
  margin-bottom: 4px;
  background: var(--mx-header-bg);
  border: 1px solid var(--mx-header-border);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}
.mention-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--mx-text-strong);
}
.mention-item:hover {
  background: var(--el-color-primary-light-9);
}
.mention-empty {
  padding: 8px;
  font-size: 12px;
  color: var(--mx-text-muted);
  text-align: center;
}
/* 消息正文中的 @提及高亮 */
.msg-text .mention {
  color: var(--el-color-primary);
  font-weight: 600;
}
.msg-row.mine .msg-text .mention {
  color: #ffe58f;
}
.conv-mention {
  flex-shrink: 0;
  margin-left: 6px;
}
.msg-input-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}
.msg-input-tools {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-right: auto;
}
.uploading-tip {
  font-size: 12px;
  color: var(--mx-text-muted);
}
.hidden-file-input {
  display: none;
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
