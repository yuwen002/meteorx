/**
 * 协作即时消息（Chat）API —— 首版仅同租户内 1:1 私聊
 *
 * - 发起/复用与某成员的私聊会话
 * - 列出我的会话（含对方、最后消息摘要、未读数）
 * - 发送消息（后端落库并经 WS 实时投递给对方）
 * - 分页拉取会话历史消息
 * - 标记会话已读
 *
 * 接口均只需登录，后端以会话成员归属为准做访问控制，不挂 RBAC 权限码。
 */
import { del, get, post } from '@/api/request'
import type { PageResult } from '@/types/pagination'

/** 会话类型（首版仅 direct） */
export type ConversationType = 'direct'

/** 会话（与后端 dto.ConversationResp 对齐） */
export interface ConversationItem {
  id: string
  type: ConversationType
  peer_user_id: string
  peer_name: string
  last_message: string
  last_at?: string
  unread_count: number
  created_at: string
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
