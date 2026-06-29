import { get, post, put, del } from '@/api/request'

export interface RoleItem {
  id: string
  name: string
  code: string
  description?: string
  tenant_id?: string
  is_system?: boolean
  scope?: string
  status?: number
  created_at?: string
  updated_at?: string
}

export interface RoleCreateParams {
  name: string
  code: string
  description?: string
  status?: number
}

export interface RoleUpdateParams {
  name?: string
  description?: string
  status?: number
}

export interface RoleListParams {
  page?: number
  page_size?: number
  keyword?: string
  status?: number
}

export function getRoleList(params: RoleListParams) {
  return get<{ list: RoleItem[]; total: number }>('/rbac/roles', params)
}

export function getRoleDetail(id: string) {
  return get<RoleItem>(`/rbac/roles/${id}`)
}

export function createRole(data: RoleCreateParams) {
  return post<RoleItem>('/rbac/roles', data)
}

export function updateRole(id: string, data: RoleUpdateParams) {
  return put<RoleItem>(`/rbac/roles/${id}`, data)
}

export function deleteRole(id: string) {
  return del(`/rbac/roles/${id}`)
}

// 角色绑定权限
export function bindRolePermissions(roleId: string, permissionIds: string[]) {
  return put(`/rbac/roles/${roleId}/permissions`, { permission_ids: permissionIds })
}

// 获取角色已绑定的权限 ID 列表
export function getRolePermissionIds(roleId: string) {
  return get<{ ids?: string[]; permission_ids?: string[]; list?: { id: string }[] }>(
    `/rbac/roles/${roleId}/permissions`
  )
}