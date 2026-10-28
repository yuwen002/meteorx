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
  tenant_name?: string        // tenant name (filled only on query)
  email: string               // invitee email
  role_ids: string[]          // assigned role ID list
  status: string              // status: pending / accepted / cancelled / expired
  invited_by: string          // inviter user ID
  expires_at: string          // expiration time
  accepted_at?: string        // accepted time (only set when accepted)
  created_at: string          // created time
}

/** 创建邀请参数 */
export interface CreateInvitationParams {
  email: string               // invitee email
  role_ids: string[]          // assigned role ID list
}

/** 批量创建邀请参数：多个邮箱共享同一组角色（上限 50） */
export interface BatchCreateInvitationParams {
  emails: string[]            // invitee email list
  role_ids: string[]          // assigned role ID list
}

/** 批量邀请中被跳过或失败的单个邮箱 */
export interface SkippedEmail {
  email: string
  reason: string
}

/** 批量创建邀请结果 */
export interface BatchCreateInvitationResult {
  created: number             // number successfully created
  skipped: SkippedEmail[]     // skipped/failed emails with reasons
}

/** 接受邀请参数 */
export interface AcceptInvitationParams {
  token: string               // invitation token (carried in the email link)
  username: string            // login username to set
  password: string            // password to set
  nickname: string            // nickname to set
}

/** 可分配角色选项 */
export interface RoleOption {
  id: string
  name: string
  code: string
}

/** 邀请列表查询参数 */
export interface InvitationListParams {
  page?: number               // page number
  page_size?: number          // page size
  keyword?: string            // search by email
  status?: string             // filter by status
}

/** 获取当前租户的邀请列表 */
export function getInvitationList(params: InvitationListParams) {
  return get<PageResult<InvitationItem>>('/invitations', params)
}

/** 创建邀请（邀请新成员加入租户） */
export function createInvitation(data: CreateInvitationParams) {
  return post<InvitationItem>('/invitations', data)
}

/** 批量创建邀请（多个邮箱共享同一组角色） */
export function createInvitationsBatch(data: BatchCreateInvitationParams) {
  return post<BatchCreateInvitationResult>('/invitations/batch', data)
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

/** 获取可分配角色列表（不分页，供邀请下拉使用） */
export function getAssignableRoles() {
  return get<RoleOption[]>('/invitations/roles')
}