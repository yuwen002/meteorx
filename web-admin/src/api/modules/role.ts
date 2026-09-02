import { get, post, put, del, type ApiResult } from '@/api/request'
import type { PermissionItem } from './permission'

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
  deleted_at?: string
}

// 角色下拉选项（简化版，用于选择器）
export interface RoleOption {
  id: string
  name: string
  code?: string
}

export interface RoleCreateParams {
  name: string
  code: string
  scope?: string
  description?: string
  status?: number
}

export interface RoleUpdateParams {
  name?: string
  code?: string
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

// 获取角色下拉列表（不分页，用于选择）
// scope: system/tenant/all，默认为 system
export function getRoleListForSelect(scope?: string) {
  return get<RoleItem[]>('/rbac/roles/select', { scope })
}

// 获取系统管理员角色列表（用于创建系统管理员时选择角色）
// 只返回 IsSystem=true 且 scope 为 system 或 all 的角色
export function getSystemAdminRoles() {
  return get<RoleItem[]>('/rbac/roles/system-admin')
}

// 获取角色下拉列表（不分页，用于选择）
// scope: system-系统级角色, tenant-租户级角色, all-所有角色
export function getRolesForSelect(scope: 'system' | 'tenant' | 'all' = 'system') {
  return get<RoleItem[]>(`/rbac/roles/select?scope=${scope}`)
}

export function getRoleDetail(id: string) {
  return get<RoleItem>(`/rbac/roles/${id}/detail`)
}

export function createRole(data: RoleCreateParams) {
  return post<RoleItem>('/rbac/roles', data)
}

export function updateRole(id: string, data: RoleUpdateParams) {
  return put<RoleItem>(`/rbac/roles/${id}/update`, data)
}

export function updateRoleStatus(id: string, status: number) {
  return put(`/rbac/roles/${id}/status`, { status })
}

export function deleteRole(id: string) {
  return del(`/rbac/roles/${id}/delete`)
}

export function batchDeleteRoles(ids: string[]) {
  return del('/rbac/roles/batch/delete', { data: { ids } })
}

export function batchUpdateRoleStatus(ids: string[], status: number) {
  return put('/rbac/roles/batch/status', { ids, status })
}

// 角色绑定权限
export function bindRolePermissions(roleId: string, permissionIds: string[]) {
  return put(`/rbac/roles/${roleId}/permissions`, { permission_ids: permissionIds })
}

// 获取角色已绑定的权限列表
export function getRolePermissions(roleId: string) {
  return get<PermissionItem[]>(`/rbac/roles/${roleId}/permissions`)
}

// 解绑角色的单个权限
export function unbindRolePermission(roleId: string, permissionId: string) {
  return del(`/rbac/roles/${roleId}/permissions`, { data: { permission_id: permissionId } })
}

// 批量解绑角色的权限
export function unbindRolePermissions(roleId: string, permissionIds: string[]) {
  return del(`/rbac/roles/${roleId}/permissions/batch`, { data: { permission_ids: permissionIds } })
}

// 获取 RBAC 统计信息
export function getRBACStats() {
  return get<{ role_count: number; permission_count: number; my_permission: number }>('/rbac/stats')
}

// 获取用户的角色列表
export function getUserRoles(userId: string) {
  return get<RoleItem[]>(`/rbac/user-roles/${userId}/roles`)
}

// 分配角色给用户
export function assignUserRoles(userId: string, roleIds: string[]) {
  return post(`/rbac/user-roles/${userId}/roles`, { role_ids: roleIds })
}

// 移除用户的单个角色
export function removeUserRole(userId: string, roleId: string) {
  return del(`/rbac/user-roles/${userId}/roles/${roleId}`)
}

// 移除用户的所有角色
export function removeAllUserRoles(userId: string) {
  return del(`/rbac/user-roles/${userId}/roles`)
}

// 获取已删除的角色列表（回收站）
export function getDeletedRoleList(params: RoleListParams) {
  return get<{ list: RoleItem[]; total: number }>('/rbac/roles/deleted', params)
}

// 恢复已删除的角色
export function restoreRole(id: string) {
  return put(`/rbac/roles/${id}/restore`)
}

// 永久删除角色（从回收站彻底删除）
export function permanentDeleteRole(id: string) {
  return del(`/rbac/roles/${id}/permanent`)
}

// 批量永久删除角色
export function batchPermanentDeleteRoles(ids: string[]) {
  return del('/rbac/roles/batch/permanent', { data: { ids } })
}