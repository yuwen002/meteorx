/**
 * 邀请模块 API
 *
 * 提供租户成员邀请的完整接口：
 * - 管理端：创建邀请、列表查询、取消、重发、删除
 * - 公开端：查询邀请信息、接受邀请并注册
 */
import { get, post, put, del } from '@/api/request'
import type { PageResult } from '@/types/pagination'

/** 邀请记录 */
export interface InvitationItem {
  id: string
  tenant_id: string
  tenant_name?: string        // 租户名称（仅查询时填充）
  email: string               // 被邀请人邮箱
  role_ids: string[]          // 分配的角色 ID 列表
  status: string              // 状态：pending / accepted / cancelled / expired
  invited_by: string          // 邀请人用户 ID
  expires_at: string          // 过期时间
  accepted_at?: string        // 接受时间（仅已接受时有值）
  created_at: string          // 创建时间
}

/** 创建邀请参数 */
export interface CreateInvitationParams {
  email: string               // 被邀请人邮箱
  role_ids: string[]          // 分配的角色 ID 列表
}

/** 接受邀请参数 */
export interface AcceptInvitationParams {
  token: string               // 邀请令牌（邮件链接中携带）
  username: string            // 设置的登录用户名
  password: string            // 设置的密码
  nickname: string            // 设置的昵称
}

/** 邀请列表查询参数 */
export interface InvitationListParams {
  page?: number               // 页码
  page_size?: number          // 每页条数
  keyword?: string            // 按邮箱搜索
  status?: string             // 按状态筛选
}

/** 获取当前租户的邀请列表 */
export function getInvitationList(params: InvitationListParams) {
  return get<PageResult<InvitationItem>>('/invitations', params)
}

/** 创建邀请（邀请新成员加入租户） */
export function createInvitation(data: CreateInvitationParams) {
  return post<InvitationItem>('/invitations', data)
}

/** 取消邀请（仅 pending 状态可取消） */
export function cancelInvitation(id: string) {
  return put(`/invitations/${id}/cancel`)
}

/** 重发邀请邮件（仅 pending 状态可重发） */
export function resendInvitation(id: string) {
  return put<InvitationItem>(`/invitations/${id}/resend`)
}

/** 删除邀请记录（物理删除，不可恢复） */
export function deleteInvitation(id: string) {
  return del(`/invitations/${id}/delete`)
}

/** 接受邀请并注册（公开接口，无需登录） */
export function acceptInvitation(data: AcceptInvitationParams) {
  return post<{ message: string }>('/invitations/accept', data)
}

/** 通过令牌查询邀请信息（公开接口，用于接受邀请页面） */
export function getInvitationInfo(token: string) {
  return get<InvitationItem>('/invitations/info', { token })
}