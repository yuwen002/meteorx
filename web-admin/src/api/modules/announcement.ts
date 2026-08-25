import request from '../request'

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

export interface PaginationMeta {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface PaginatedResult<T> {
  data: T[]
  pagination: PaginationMeta
}

export function getAnnouncementList(params?: {
  page?: number
  page_size?: number
  keyword?: string
  status?: number
  scope?: string
}) {
  return request.get<any, PaginatedResult<AnnouncementItem>>('/admin/announcements', { params })
}

export function getAnnouncementDetail(id: string) {
  return request.get<any, AnnouncementItem>(`/admin/announcements/${id}`)
}

export function createAnnouncement(data: {
  title: string
  content: string
  scope: string
  target_tenant_id?: string
  status: number
  publish_at?: string
  expire_at?: string
}) {
  return request.post<any, AnnouncementItem>('/admin/announcements', data)
}

export function updateAnnouncement(id: string, data: {
  title: string
  content: string
  scope: string
  target_tenant_id?: string
  status: number
  publish_at?: string
  expire_at?: string
}) {
  return request.put<any, AnnouncementItem>(`/admin/announcements/${id}`, data)
}

export function updateAnnouncementStatus(id: string, status: number) {
  return request.put<any, AnnouncementItem>(`/admin/announcements/${id}/status`, { status })
}

export function deleteAnnouncement(id: string) {
  return request.delete<any, { id: string }>(`/admin/announcements/${id}`)
}

export function listTenantAnnouncements(params?: { page?: number; page_size?: number }) {
  return request.get<any, PaginatedResult<AnnouncementItem>>('/announcements', { params })
}