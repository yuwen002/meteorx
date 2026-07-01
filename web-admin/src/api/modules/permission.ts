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

export function getPermissionList(params?: { page?: number; page_size?: number; keyword?: string; resource?: string }) {
  return get<{ list: PermissionItem[]; total: number }>('/rbac/permissions', params)
}

export function getPermissionDetail(id: string) {
  return get<PermissionItem>(`/rbac/permissions/${id}/detail`)
}

export function createPermission(data: PermissionCreateParams) {
  return post<PermissionItem>('/rbac/permissions', data)
}

export function updatePermission(id: string, data: Partial<PermissionCreateParams>) {
  return put<PermissionItem>(`/rbac/permissions/${id}/update`, data)
}

export function deletePermission(id: string) {
  return del(`/rbac/permissions/${id}/delete`)
}

export function updatePermissionStatus(id: string, status: number) {
  return put(`/rbac/permissions/${id}/status`, { status })
}

export function batchUpdatePermissionStatus(ids: string[], status: number) {
  return put('/rbac/permissions/batch/status', { ids, status })
}

export function batchDeletePermissions(ids: string[]) {
  return del('/rbac/permissions/batch/delete', { data: { ids } })
}