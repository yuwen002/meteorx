import { get, post, put, del } from '@/api/request'
import type { PageResult } from '@/types/pagination'

export interface InvitationItem {
  id: string
  tenant_id: string
  tenant_name?: string
  email: string
  role_ids: string[]
  status: string
  invited_by: string
  expires_at: string
  accepted_at?: string
  created_at: string
}

export interface CreateInvitationParams {
  email: string
  role_ids: string[]
}

export interface AcceptInvitationParams {
  token: string
  username: string
  password: string
  nickname: string
}

export interface InvitationListParams {
  page?: number
  page_size?: number
  keyword?: string
  status?: string
}

export function getInvitationList(params: InvitationListParams) {
  return get<PageResult<InvitationItem>>('/invitations', params)
}

export function createInvitation(data: CreateInvitationParams) {
  return post<InvitationItem>('/invitations', data)
}

export function cancelInvitation(id: string) {
  return put(`/invitations/${id}/cancel`)
}

export function resendInvitation(id: string) {
  return put<InvitationItem>(`/invitations/${id}/resend`)
}

export function deleteInvitation(id: string) {
  return del(`/invitations/${id}/delete`)
}

export function acceptInvitation(data: AcceptInvitationParams) {
  return post<{ message: string }>('/invitations/accept', data)
}

export function getInvitationInfo(token: string) {
  return get<InvitationItem>('/invitations/info', { token })
}
