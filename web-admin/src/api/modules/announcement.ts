import { get, post, put, del } from '@/api/request'

// 公告响应类型
export interface AnnouncementItem {
  id: string
  title: string
  content: string
  scope: string
  target_tenant_id: string
  status: number
  status_text: string
  publisher_id: string
  publish_at: string
  expire_at: string
  created_at: string
  updated_at: string
}

// 获取公告列表
export function getAnnouncementList(params: any) {
  return get('/admin/announcements', { params })
}

// 获取公告详情
export function getAnnouncementDetail(id: string) {
  return get(`/admin/announcements/${id}`)
}

// 创建公告
export function createAnnouncement(data: any) {
  return post('/admin/announcements', data)
}

// 更新公告
export function updateAnnouncement(id: string, data: any) {
  return put(`/admin/announcements/${id}`, data)
}

// 发布/下架公告
export function updateAnnouncementStatus(id: string, status: number) {
  return put(`/admin/announcements/${id}/status`, { status })
}

// 删除公告
export function deleteAnnouncement(id: string) {
  return del(`/admin/announcements/${id}`)
}
