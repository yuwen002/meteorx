import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  openDirectConversation,
  listConversations,
  sendChatMessage,
  listChatMessages,
  markConversationRead,
  type ConversationItem,
  type ChatMessageItem
} from '@/api/modules/chat'
import { toPageResult } from '@/types/pagination'

/**
 * 协作即时消息（1:1 私聊）状态。
 *
 * 集中管理会话列表、当前会话历史与 WS 实时增量：
 * - 收到 MsgTypeChat 推送时，若为当前会话则即时追加，否则刷新会话列表并累加未读；
 * - 打开会话拉取历史后标记已读，清零该会话未读。
 */
export const useChatStore = defineStore('chat', () => {
  const conversations = ref<ConversationItem[]>([])
  const activeId = ref<string>('')
  const messages = ref<ChatMessageItem[]>([])
  const loadingConversations = ref(false)
  const loadingMessages = ref(false)
  const sending = ref(false)

  /** 全部会话未读总数（用于侧边栏/菜单角标） */
  const totalUnread = computed(() => conversations.value.reduce((s, c) => s + (c.unread_count || 0), 0))

  /** 当前会话（未选中时为 null） */
  const activeConversation = computed(() => conversations.value.find((c) => c.id === activeId.value) || null)

  /** 拉取会话列表 */
  async function loadConversations() {
    loadingConversations.value = true
    try {
      conversations.value = (await listConversations()) || []
    } finally {
      loadingConversations.value = false
    }
  }

  /** 发起/复用与 peerId 的私聊会话，返回会话对象 */
  async function openWith(peerId: string): Promise<ConversationItem> {
    const conv = await openDirectConversation(peerId)
    // 若列表已有该会话则原位替换，否则前置插入
    const idx = conversations.value.findIndex((c) => c.id === conv.id)
    if (idx >= 0) conversations.value[idx] = conv
    else conversations.value = [conv, ...conversations.value]
    return conv
  }

  /** 选中会话并加载历史、标记已读 */
  async function selectConversation(id: string) {
    activeId.value = id
    messages.value = []
    loadingMessages.value = true
    try {
      const res = await listChatMessages(id, { page: 1, page_size: 50 })
      messages.value = toPageResult<ChatMessageItem>(res).list
      // 拉到历史即视为已读，清零本地未读并推进服务端水位
      const conv = conversations.value.find((c) => c.id === id)
      if (conv) conv.unread_count = 0
      void markConversationRead(id).catch(() => {})
    } finally {
      loadingMessages.value = false
    }
  }

  /** 发送消息，成功后追加到当前会话并更新会话摘要 */
  async function send(content: string): Promise<boolean> {
    const text = content.trim()
    if (!activeId.value || !text) return false
    sending.value = true
    try {
      const msg = await sendChatMessage(activeId.value, text)
      messages.value = [...messages.value, msg]
      const conv = conversations.value.find((c) => c.id === activeId.value)
      if (conv) {
        conv.last_message = msg.content
        conv.last_at = msg.created_at
      }
      return true
    } finally {
      sending.value = false
    }
  }

  /** 处理 WS 推送来的私聊新消息 */
  function receiveChatPush(msg: ChatMessageItem) {
    if (msg.conversation_id === activeId.value) {
      // 当前会话：直接追加并顺带标记已读（用户正看着这个会话）
      messages.value = [...messages.value, msg]
      const conv = conversations.value.find((c) => c.id === activeId.value)
      if (conv) {
        conv.last_message = msg.content
        conv.last_at = msg.created_at
      }
      void markConversationRead(msg.conversation_id).catch(() => {})
      return
    }
    // 非当前会话：累加未读并更新摘要；会话不在列表中（首次）时回源刷新
    const conv = conversations.value.find((c) => c.id === msg.conversation_id)
    if (conv) {
      conv.unread_count = (conv.unread_count || 0) + 1
      conv.last_message = msg.content
      conv.last_at = msg.created_at
    } else {
      void loadConversations()
    }
  }

  /** 退出登录时重置，避免跨账号泄露会话与消息 */
  function reset() {
    conversations.value = []
    messages.value = []
    activeId.value = ''
  }

  return {
    conversations,
    activeId,
    messages,
    loadingConversations,
    loadingMessages,
    sending,
    totalUnread,
    activeConversation,
    loadConversations,
    openWith,
    selectConversation,
    send,
    receiveChatPush,
    reset
  }
})
