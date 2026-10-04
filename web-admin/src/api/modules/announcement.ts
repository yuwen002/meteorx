import request from '../request'
import type { PaginatedResult } from '@/types/pagination'

export interface AnnouncementItem {
  id: string
  title: string
  content: string
  scope: string
  target_tenant_id: string
  status: number
  status_text?: string
  publisher_id: string
  publish_at: string | null
  expire_at: string | null
  created_at: string
  updated_at: string
}

// 获取公告列表（管理端，分页）
export function getAnnouncementList(params?: {
  page?: number
  page_size?: number
  keyword?: string
  status?: number
  scope?: string
}) {
  return request.get<unknown, PaginatedResult<AnnouncementItem>>('/admin/announcements', { params })
}

// 获取公告详情
export function getAnnouncementDetail(id: string) {
  return request.get<unknown, AnnouncementItem>(`/admin/announcements/${id}`)
}

// 创建公告
export function createAnnouncement(data: {
  title: string
  content: string
  scope: string
  target_tenant_id?: string
  status: number
  publish_at?: string
  expire_at?: string
}) {
  return request.post<unknown, AnnouncementItem>('/admin/announcements', data)
}

// 更新公告
export function updateAnnouncement(id: string, data: {
  title: string
  content: string
  scope: string
  target_tenant_id?: string
  status: number
  publish_at?: string
  expire_at?: string
}) {
  return request.put<unknown, AnnouncementItem>(`/admin/announcements/${id}`, data)
}

// 更新公告状态（发布/下架）
export function updateAnnouncementStatus(id: string, status: number) {
  return request.put<unknown, AnnouncementItem>(`/admin/announcements/${id}/status`, { status })
}

// 删除公告
export function deleteAnnouncement(id: string) {
  return request.delete<unknown, { id: string }>(`/admin/announcements/${id}`)
}

// 获取当前租户的公告列表（租户端，分页）
export function listTenantAnnouncements(params?: { page?: number; page_size?: number }) {
  return request.get<unknown, PaginatedResult<AnnouncementItem>>('/announcements', { params })
}