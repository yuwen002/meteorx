/**
 * 协作即时消息（Chat）API —— 同租户内 1:1 私聊与群聊
 *
 * - 发起/复用与某成员的私聊会话；创建群聊会话
 * - 列出我的会话（含展示名、最后消息摘要、未读数；群聊额外含成员数与我的角色）
 * - 发送消息（后端落库并经 WS 实时投递给其他成员）
 * - 分页拉取会话历史消息、标记已读
 * - 群成员管理：列表/添加/移除/退群/改名
 *
 * 接口均只需登录，后端以会话成员归属为准做访问控制（踢人/改名限群主），不挂 RBAC 权限码。
 */
import { del, get, post, put } from '@/api/request'
import type { PageResult } from '@/types/pagination'

/** 会话类型 */
export type ConversationType = 'direct' | 'group'

/** 会话成员角色 */
export type MemberRole = 'owner' | 'member'

/** 会话（与后端 dto.ConversationResp 对齐） */
export interface ConversationItem {
  id: string
  type: ConversationType
  peer_user_id: string
  peer_name: string
  last_message: string
  last_at?: string
  unread_count: number
  member_count?: number
  my_role?: MemberRole
  created_at: string
}

/** 会话成员（与后端 dto.MemberResp 对齐） */
export interface MemberItem {
  user_id: string
  name: string
  role: MemberRole
  joined_at: string
}

/** 聊天消息（与后端 dto.ChatMessageResp 对齐） */
export interface ChatMessageItem {
  id: string
  conversation_id: string
  sender_id: string
  sender_name?: string
  content: string
  type: string
  /** 是否已被发送人撤回（后端软撤回，正文为空） */
  recalled?: boolean
  created_at: string
}

/** 历史消息分页查询参数 */
export interface MessageListParams {
  page?: number
  page_size?: number
}

/** 发起/复用与 peerId 的 1:1 私聊会话 */
export function openDirectConversation(peerId: string) {
  return post<ConversationItem>('/chat/conversations/direct', { participant_id: peerId })
}

/** 创建群聊会话（memberIds 不含创建人，由后端自动置为 owner） */
export function createGroupConversation(name: string, memberIds: string[]) {
  return post<ConversationItem>('/chat/conversations/group', { name, member_ids: memberIds })
}

/** 列出我的全部会话 */
export function listConversations() {
  return get<ConversationItem[]>('/chat/conversations')
}

/** 发送一条消息，返回落库后的消息（含服务端生成的 id/时间/发送人姓名） */
export function sendChatMessage(conversationId: string, content: string) {
  return post<ChatMessageItem>(`/chat/conversations/${conversationId}/messages`, { content })
}

/** 分页拉取会话历史消息（后端页内按时间升序返回） */
export function listChatMessages(conversationId: string, params?: MessageListParams) {
  return get<PageResult<ChatMessageItem>>(`/chat/conversations/${conversationId}/messages`, params)
}

/** 标记会话已读（幂等，推进服务端已读水位） */
export function markConversationRead(conversationId: string) {
  return post<{ ok: boolean }>(`/chat/conversations/${conversationId}/read`)
}

/** 撤回自己发送的消息（仅发送人本人、限时 5 分钟） */
export function recallChatMessage(conversationId: string, messageId: string) {
  return del<{ ok: boolean }>(`/chat/conversations/${conversationId}/messages/${messageId}`)
}

/** 上报输入中状态（后端无状态转发给其他成员，频率控制由调用方承担） */
export function sendTyping(conversationId: string, typing: boolean) {
  return post<{ ok: boolean }>(`/chat/conversations/${conversationId}/typing`, { typing })
}

/** 在线状态：当前持有 WS 连接的用户 ID 集合 */
export function getPresence() {
  return get<{ online_user_ids: string[] }>('/chat/presence')
}

/** 列出会话成员（含角色） */
export function listConversationMembers(conversationId: string) {
  return get<MemberItem[]>(`/chat/conversations/${conversationId}/members`)
}

/** 向群添加成员（任意成员可邀请） */
export function addConversationMembers(conversationId: string, userIds: string[]) {
  return post<{ ok: boolean }>(`/chat/conversations/${conversationId}/members`, { user_ids: userIds })
}

/** 群主移除成员 */
export function removeConversationMember(conversationId: string, userId: string) {
  return del<{ ok: boolean }>(`/chat/conversations/${conversationId}/members/${userId}`)
}

/** 退出群聊 */
export function leaveGroup(conversationId: string) {
  return post<{ ok: boolean }>(`/chat/conversations/${conversationId}/leave`)
}

/** 群改名（仅群主） */
export function renameGroup(conversationId: string, name: string) {
  return put<{ ok: boolean }>(`/chat/conversations/${conversationId}`, { name })
}
