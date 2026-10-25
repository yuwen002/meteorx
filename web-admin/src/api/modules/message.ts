/**
 * 站内消息（收件箱）模块 API
 *
 * 提供当前登录用户的收件箱读写：
 * - 分页查询消息列表（可按分类/未读/关键字过滤）
 * - 查询未读总数与分类明细（顶栏铃铛角标）
 * - 标记单条 / 全部已读，删除单条消息
 *
 * 消息按「收件人」隔离，接口不挂 RBAC 权限码，任何登录用户只能看到自己的消息。
 */
import { get, put, del } from '@/api/request'
import type { PageResult } from '@/types/pagination'

/** 消息分类：系统 / 公告 / 任务 / 套餐 / 审计告警 */
export type MessageCategory = 'system' | 'announcement' | 'task' | 'plan' | 'audit'

/** 收件箱消息（与后端 dto.MessageResp 对齐） */
export interface MessageItem {
  id: string
  category: string
  title: string
  content: string
  /** 业务跳转链接，如 /task；空表示无跳转 */
  link: string
  /** 0=低 1=普通 2=高 3=紧急 */
  priority: number
  source_type: string
  source_id: string
  /** 为空表示未读 */
  read_at: string | null
  created_at: string
}

/** 未读数响应（与后端 dto.UnreadCountResp 对齐） */
export interface UnreadCountResult {
  total: number
  by_category: Record<string, number>
}

/** 收件箱列表查询参数 */
export interface MessageListParams {
  page?: number
  page_size?: number
  category?: string
  /** true 仅看未读 */
  unread?: boolean
  keyword?: string
}

/** 分页获取当前用户的收件箱 */
export function listMessages(params: MessageListParams) {
  return get<PageResult<MessageItem>>('/messages', params)
}

/** 获取未读总数与分类明细 */
export function getUnreadCount() {
  return get<UnreadCountResult>('/messages/unread-count')
}

/** 标记单条消息已读（幂等） */
export function markMessageRead(id: string) {
  return put<{ id: string }>(`/messages/${id}/read`)
}

/** 标记全部已读，可按分类；不传分类表示全部 */
export function markAllMessagesRead(category?: string) {
  return put<{ updated: number }>('/messages/read-all', category ? { category } : {})
}

/** 删除单条消息 */
export function deleteMessage(id: string) {
  return del<{ id: string }>(`/messages/${id}`)
}

/** 单个分类的消息偏好（与后端 dto.PreferenceItem 对齐） */
export interface MessagePreferenceItem {
  category: string
  /** true 表示关闭接收：该类消息不再落库 */
  muted: boolean
  /** true 表示强制接收（安全告警），前端应禁用开关 */
  mandatory: boolean
}

/** 消息偏好响应 */
export interface MessagePreferencesResult {
  items: MessagePreferenceItem[]
}

/** 获取当前用户的消息偏好（未设置过的分类按已接收返回） */
export function getMessagePreferences() {
  return get<MessagePreferencesResult>('/messages/preferences')
}

/** 修改单个分类的接收开关，返回修改后的完整偏好列表 */
export function updateMessagePreference(category: string, muted: boolean) {
  return put<MessagePreferencesResult>('/messages/preferences', { category, muted })
}
