import { get, post, put, del } from '@/api/request'

export interface PermissionItem {
  id: string
  name: string
  code: string
  description?: string
  resource: string
  action: string
  status?: number
  created_at?: string
}

export interface PermissionCreateParams {
  name: string
  code: string
  resource: string
  action: string
  description?: string
}

// 获取权限列表（分页）
export function getPermissionList(params?: { page?: number; page_size?: number; keyword?: string; resource?: string }) {
  return get<{ list: PermissionItem[]; total: number }>('/rbac/permissions', params)
}

// 获取权限详情
export function getPermissionDetail(id: string) {
  return get<PermissionItem>(`/rbac/permissions/${id}/detail`)
}

// 创建权限
export function createPermission(data: PermissionCreateParams) {
  return post<PermissionItem>('/rbac/permissions', data)
}

// 更新权限
export function updatePermission(id: string, data: Partial<PermissionCreateParams>) {
  return put<PermissionItem>(`/rbac/permissions/${id}/update`, data)
}

// 删除权限
export function deletePermission(id: string) {
  return del(`/rbac/permissions/${id}/delete`)
}

// 更新权限状态
export function updatePermissionStatus(id: string, status: number) {
  return put(`/rbac/permissions/${id}/status`, { status })
}

// 批量更新权限状态
export function batchUpdatePermissionStatus(ids: string[], status: number) {
  return put('/rbac/permissions/batch/status', { ids, status })
}

// 批量删除权限
export function batchDeletePermissions(ids: string[]) {
  return del('/rbac/permissions/batch/delete', { data: { ids } })
}