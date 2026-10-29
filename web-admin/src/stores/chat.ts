import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import {
  openDirectConversation,
  createGroupConversation,
  listConversations,
  sendChatMessage,
  listChatMessages,
  markConversationRead,
  getConversationReadStatus,
  recallChatMessage,
  sendTyping,
  getPresence,
  listConversationMembers,
  addConversationMembers,
  removeConversationMember,
  leaveGroup as apiLeaveGroup,
  renameGroup as apiRenameGroup,
  type ConversationItem,
  type ChatMessageItem,
  type MemberItem,
  type ReadStateItem
} from '@/api/modules/chat'
import { toPageResult, asArray } from '@/types/pagination'
import { useUserStore } from '@/stores/user'

// 输入中提示自动消退时长（无停止事件时多久后不再展示）
const TYPING_TTL_MS = 4000
// 历史消息单页条数（首屏与「加载更早」共用）
const MESSAGE_PAGE_SIZE = 50

/**
 * 协作即时消息（1:1 私聊）状态。
 *
 * 集中管理会话列表、当前会话历史与 WS 实时增量：
 * - 收到 MsgTypeChat 推送时，若为当前会话则即时追加，否则刷新会话列表并累加未读；
 * - 打开会话拉取历史后标记已读，清零该会话未读。
 */
export const useChatStore = defineStore('chat', () => {
  const userStore = useUserStore()
  const conversations = ref<ConversationItem[]>([])
  const activeId = ref<string>('')
  const messages = ref<ChatMessageItem[]>([])
  const loadingConversations = ref(false)
  const loadingMessages = ref(false)
  const sending = ref(false)

  // ---- 历史消息分页状态（page 基于后端 created_at DESC 分页，加载更早即取更大页码后前插） ----
  /** 已加载到的页码（0 表示尚未加载） */
  const messagePage = ref(0)
  /** 是否还有更早的历史消息可加载 */
  const hasMoreMessages = ref(false)
  /** 正在加载更早的历史消息 */
  const loadingOlder = ref(false)

  /** 当前在线（持有 WS 连接）的用户 ID 集合 */
  const onlineUserIds = ref<string[]>([])
  /**
   * 当前会话内除本人外各成员的已读水位：userId -> last_read_at。
   * 时间格式为后端统一的 YYYY-MM-DD HH:mm:ss（秒级），字典序即时间序，可直接字符串比较。
   */
  const readStates = ref<Record<string, string>>({})
  /** conversationID -> 正在输入的用户 ID 列表 */
  const typingUsers = ref<Record<string, string[]>>({})
  /** `${convId}:${userId}` -> 消退定时器 */
  const typingTimers: Record<string, ReturnType<typeof setTimeout>> = {}

  /** 全部会话未读总数（用于侧边栏/菜单角标） */
  const totalUnread = computed(() => conversations.value.reduce((s, c) => s + (c.unread_count || 0), 0))

  /** 当前会话（未选中时为 null） */
  const activeConversation = computed(() => conversations.value.find((c) => c.id === activeId.value) || null)

  /** 判定用户是否在线 */
  function isOnline(userId: string): boolean {
    return onlineUserIds.value.includes(userId)
  }

  /** 当前登录用户 ID（用于 @提及判定），未登录为空串 */
  const myId = () => userStore.userInfo?.id || ''

  /** 判定一条消息是否 @ 了当前用户 */
  function mentionsMe(msg: { mention_ids?: string }): boolean {
    const me = myId()
    if (!me || !msg.mention_ids) return false
    return msg.mention_ids.split(',').some((id) => id.trim() === me)
  }

  /** 拉取在线用户集合（拉取式，选中会话或收到 chat 事件时刷新，不做轮询） */
  async function loadPresence() {
    try {
      const res = await getPresence()
      onlineUserIds.value = res?.online_user_ids || []
    } catch {
      /* 忽略：在线状态为弱信息，拉取失败不影响聊天主流程 */
    }
  }

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
    upsertConversation(conv)
    return conv
  }

  /** 创建群聊会话（memberIds 不含创建人），返回列表富化后的会话 */
  async function createGroup(name: string, memberIds: string[]): Promise<ConversationItem> {
    const conv = await createGroupConversation(name, memberIds)
    upsertConversation(conv)
    return conv
  }

  /** 将会话原位替换或前置插入列表 */
  function upsertConversation(conv: ConversationItem) {
    const idx = conversations.value.findIndex((c) => c.id === conv.id)
    if (idx >= 0) conversations.value[idx] = conv
    else conversations.value = [conv, ...conversations.value]
  }

  /** 选中会话并加载首页历史、标记已读 */
  async function selectConversation(id: string) {
    activeId.value = id
    messages.value = []
    messagePage.value = 0
    hasMoreMessages.value = false
    readStates.value = {}
    loadingMessages.value = true
    try {
      const res = await listChatMessages(id, { page: 1, page_size: MESSAGE_PAGE_SIZE })
      const page = toPageResult<ChatMessageItem>(res)
      messages.value = page.list
      messagePage.value = 1
      // 后端返回 total；total 缺失时以「取满一页」推断还有更早消息
      hasMoreMessages.value = page.total > 0 ? messages.value.length < page.total : page.list.length === MESSAGE_PAGE_SIZE
      // 拉到历史即视为已读，清零本地未读与 @我 标记并推进服务端水位
      const conv = conversations.value.find((c) => c.id === id)
      if (conv) {
        conv.unread_count = 0
        conv.mentioned = false
      }
      void markConversationRead(id).catch(() => {})
      void loadPresence()
      void loadReadStates(id)
    } finally {
      loadingMessages.value = false
    }
  }

  /** 拉取当前会话其他成员的已读水位（弱信息，失败静默） */
  async function loadReadStates(conversationId: string) {
    try {
      const res = await getConversationReadStatus(conversationId)
      const map: Record<string, string> = {}
      // 空列表时后端序列化为 { data: null }，须经 asArray 归一化
      for (const s of asArray<ReadStateItem>(res)) {
        if (s.last_read_at) map[s.user_id] = s.last_read_at
      }
      // 回拉期间会话已切换则丢弃，避免跨会话串位
      if (activeId.value === conversationId) readStates.value = map
    } catch {
      /* 忽略：已读回执为弱信息，不影响聊天主流程 */
    }
  }

  /** 统计 createdAt 时刻发送的消息已被多少成员阅读（水位 >= 消息时间即已读） */
  function readCountAt(createdAt: string): number {
    let n = 0
    for (const at of Object.values(readStates.value)) {
      if (at >= createdAt) n++
    }
    return n
  }

  /**
   * 加载更早的历史消息并前插到当前列表。
   * 返回本次新增条数（0 表示无更多或正在加载），供视图据此保持滚动位置。
   */
  async function loadOlderMessages(): Promise<number> {
    if (!activeId.value || !hasMoreMessages.value || loadingOlder.value || messagePage.value < 1) return 0
    loadingOlder.value = true
    try {
      const nextPage = messagePage.value + 1
      const res = await listChatMessages(activeId.value, { page: nextPage, page_size: MESSAGE_PAGE_SIZE })
      const page = toPageResult<ChatMessageItem>(res)
      // 期间有新消息插入会使后端偏移漂移，按 id 去重防重复前插
      const held = new Set(messages.value.map((m) => m.id))
      const older = page.list.filter((m) => !held.has(m.id))
      messages.value = [...older, ...messages.value]
      messagePage.value = nextPage
      hasMoreMessages.value = page.total > 0
        ? messages.value.length < page.total
        : page.list.length === MESSAGE_PAGE_SIZE
      return older.length
    } catch {
      // 加载失败不改变分页状态，用户可再次滚动重试
      return 0
    } finally {
      loadingOlder.value = false
    }
  }

  /** 发送消息，成功后追加到当前会话并更新会话摘要；mentionIds 为逗号分隔的 @提及用户 ID */
  async function send(content: string, mentionIds = ''): Promise<boolean> {
    const text = content.trim()
    if (!activeId.value || !text) return false
    sending.value = true
    try {
      const msg = await sendChatMessage(activeId.value, text, mentionIds)
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
      // 被 @ 时在会话上打标，供侧栏“@我”徽标提醒
      if (mentionsMe(msg)) conv.mentioned = true
    } else {
      void loadConversations()
    }
  }

  /** 处理 WS 推送来的撤回事件：原位把该消息标记为已撤回 */
  function receiveRecallPush(payload: { conversation_id: string; message_id: string }) {
    if (!payload?.message_id) return
    if (payload.conversation_id === activeId.value) {
      const idx = messages.value.findIndex((m) => m.id === payload.message_id)
      if (idx >= 0) {
        messages.value[idx] = { ...messages.value[idx], recalled: true, content: '' }
      }
    }
    // 若被撤回的正是会话摘要那条，摘要已不可靠，交由下次拉列表自然纠正，这里不主动改写
  }

  /** 处理 WS 推送来的已读回执：仅更新当前会话内对应成员的水位（取更大值） */
  function receiveReadPush(payload: { conversation_id: string; user_id: string; read_at: string }) {
    if (!payload?.conversation_id || !payload?.user_id || payload.conversation_id !== activeId.value) return
    const cur = readStates.value[payload.user_id]
    if (!cur || payload.read_at > cur) {
      readStates.value = { ...readStates.value, [payload.user_id]: payload.read_at }
    }
  }

  /** 处理 WS 推送来的输入中事件 */
  function receiveTypingPush(payload: { conversation_id: string; user_id: string; typing: boolean }) {
    if (!payload?.conversation_id || !payload?.user_id) return
    setTyping(payload.conversation_id, payload.user_id, payload.typing)
  }

  /** 处理 WS 推送来的会话元信息变更（群名/成员数等）：原位刷新或直接回拉列表 */
  function receiveConversationPush(conv: ConversationItem) {
    if (!conv?.id) return
    const idx = conversations.value.findIndex((c) => c.id === conv.id)
    if (idx >= 0) {
      // 保留本地已清零的未读（正看着该会话时不应被推送重新点亮）
      conversations.value[idx] = { ...conv, unread_count: conversations.value[idx].unread_count }
    } else {
      void loadConversations()
    }
  }

  // ---- 群成员管理 ----

  /** 拉取指定会话成员列表 */
  async function loadMembers(conversationId: string): Promise<MemberItem[]> {
    return (await listConversationMembers(conversationId)) || []
  }

  /** 向群添加成员，成功后刷新会话列表以更新成员数 */
  async function addMembers(conversationId: string, userIds: string[]) {
    await addConversationMembers(conversationId, userIds)
    await loadConversations()
  }

  /** 群主移除成员，成功后刷新会话列表 */
  async function removeMember(conversationId: string, userId: string) {
    await removeConversationMember(conversationId, userId)
    await loadConversations()
  }

  /** 退出群聊；若退出的是当前会话则清空选中，列表移除该会话（空群后端已软删） */
  async function leaveConversation(conversationId: string) {
    await apiLeaveGroup(conversationId)
    conversations.value = conversations.value.filter((c) => c.id !== conversationId)
    if (activeId.value === conversationId) {
      activeId.value = ''
      messages.value = []
    }
    await loadConversations()
  }

  /** 群改名，成功后同步本地会话展示名 */
  async function rename(conversationId: string, name: string) {
    await apiRenameGroup(conversationId, name)
    const conv = conversations.value.find((c) => c.id === conversationId)
    if (conv) conv.peer_name = name
  }

  /** 维护某会话内某成员的输入中状态，带 TTL 自动消退 */
  function setTyping(convId: string, userId: string, typing: boolean) {
    const key = `${convId}:${userId}`
    if (!typing) {
      clearTypingUser(convId, userId)
      if (typingTimers[key]) {
        clearTimeout(typingTimers[key])
        delete typingTimers[key]
      }
      return
    }
    const arr = typingUsers.value[convId] || []
    if (!arr.includes(userId)) typingUsers.value[convId] = [...arr, userId]
    if (typingTimers[key]) clearTimeout(typingTimers[key])
    typingTimers[key] = setTimeout(() => {
      clearTypingUser(convId, userId)
      delete typingTimers[key]
    }, TYPING_TTL_MS)
  }

  function clearTypingUser(convId: string, userId: string) {
    const arr = typingUsers.value[convId]
    if (!arr) return
    const next = arr.filter((u) => u !== userId)
    if (next.length) typingUsers.value[convId] = next
    else delete typingUsers.value[convId]
  }

  /** 上报本端输入中状态（频率控制由调用方承担；失败静默，属弱信息） */
  function notifyTyping(typing: boolean) {
    if (!activeId.value) return
    void sendTyping(activeId.value, typing).catch(() => {})
  }

  /** 撤回自己发送的消息，成功后原位标记 */
  async function recall(messageId: string): Promise<boolean> {
    if (!activeId.value) return false
    try {
      await recallChatMessage(activeId.value, messageId)
      const idx = messages.value.findIndex((m) => m.id === messageId)
      if (idx >= 0) {
        messages.value[idx] = { ...messages.value[idx], recalled: true, content: '' }
      }
      return true
    } catch {
      return false
    }
  }

  /** 退出登录时重置，避免跨账号泄露会话与消息 */
  function reset() {
    conversations.value = []
    messages.value = []
    activeId.value = ''
    messagePage.value = 0
    hasMoreMessages.value = false
    readStates.value = {}
    onlineUserIds.value = []
    typingUsers.value = {}
    Object.values(typingTimers).forEach((t) => clearTimeout(t))
  }

  return {
    conversations,
    activeId,
    messages,
    loadingConversations,
    loadingMessages,
    loadingOlder,
    hasMoreMessages,
    messagePage,
    readStates,
    sending,
    onlineUserIds,
    typingUsers,
    totalUnread,
    activeConversation,
    isOnline,
    mentionsMe,
    loadPresence,
    loadConversations,
    openWith,
    createGroup,
    selectConversation,
    loadOlderMessages,
    readCountAt,
    receiveReadPush,
    send,
    recall,
    notifyTyping,
    receiveChatPush,
    receiveRecallPush,
    receiveTypingPush,
    receiveConversationPush,
    loadMembers,
    addMembers,
    removeMember,
    leaveConversation,
    rename,
    reset
  }
})
