import { get, post, put, del, type ApiResult } from '@/api/request'

export interface UserRoleInfo {
  id: string
  name: string
  code: string
}

export interface UserItem {
  id: string
  tenant_id?: string
  username: string
  nickname?: string
  email?: string
  status?: number
  is_master?: boolean
  roles?: string[]        // 角色编码列表（兼容旧版）
  role_ids?: string[]     // 角色ID列表
  role_list?: UserRoleInfo[]  // 角色详细信息列表
  deleted_at?: string
  created_at?: string
  updated_at?: string
}

export interface UserCreateParams {
  tenant_id?: string
  username: string
  password: string
  nickname?: string
  email?: string
  role_ids?: string[]
}

export interface UserUpdateParams {
  nickname?: string
  email?: string
  status?: number
  password?: string
  role_ids?: string[]
}

// 系统管理员创建参数（单角色）
export interface MasterAdminCreateParams {
  username: string
  password: string
  nickname?: string
  email?: string
  role_id?: string  // 单个角色ID
}

// 系统管理员更新参数（单角色）
export interface MasterAdminUpdateParams {
  nickname?: string
  email?: string
  status?: number
  password?: string
  role_id?: string  // 单个角色ID
}

export interface UserListParams {
  page?: number
  page_size?: number
  keyword?: string
  status?: number
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// ==================== 普通用户接口 ====================

// 用户列表（租户内用户或管理员接口）
export function getUserList(params: UserListParams) {
  return get<PageResult<UserItem>>('/users', params)
}

// 用户详情
export function getUserDetail(id: string) {
  return get<UserItem>(`/users/${id}`)
}

// 创建用户
export function createUser(data: UserCreateParams) {
  return post<UserItem>('/users', data)
}

// 更新用户
export function updateUser(id: string, data: UserUpdateParams) {
  return put<UserItem>(`/users/${id}/update`, data)
}

// 重置用户密码（管理员使用，不需要原密码）
export function resetUserPassword(id: string, data: { new_password: string; confirm_password: string }) {
  return put(`/users/${id}/reset-password`, data)
}

// 删除用户
export function deleteUser(id: string) {
  return del(`/users/${id}`)
}

// ==================== 个人中心接口 ====================

// 获取当前用户个人信息
export function getProfile() {
  return get<UserItem>('/profile')
}

// 更新当前用户个人信息
export function updateProfile(data: { nickname?: string; email?: string }) {
  return put<UserItem>('/profile', data)
}

// 修改当前用户密码
export function changePassword(data: { old_password: string; new_password: string }) {
  return put('/profile/password', data)
}

// 获取用户统计信息（当前租户）
export function getUserStats() {
  return get<{ user_count: number }>('/profile/stats')
}

// 获取所有用户统计信息（跨租户，管理员用）
export function getAllUserStats() {
  return get<{ user_count: number }>('/admin/stats')
}

// ==================== 系统管理员接口 ====================

// 系统管理员列表
export function getMasterAdminList(params: UserListParams) {
  return get<PageResult<UserItem>>('/admin/users', params)
}

// 系统管理员详情
export function getMasterAdminDetail(id: string) {
  return get<UserItem>(`/admin/users/${id}/detail`)
}

// 创建系统管理员
export function createMasterAdmin(data: MasterAdminCreateParams) {
  return post<UserItem>('/admin/users', data)
}

// 更新系统管理员
export function updateMasterAdmin(id: string, data: MasterAdminUpdateParams) {
  return put<UserItem>(`/admin/users/${id}/update`, data)
}

// 删除系统管理员
export function deleteMasterAdmin(id: string) {
  return del(`/admin/users/${id}/delete`)
}

// 更新系统管理员状态
export function updateMasterAdminStatus(id: string, status: number) {
  return put(`/admin/users/${id}/status`, { status })
}

// 批量更新系统管理员状态
export function batchUpdateMasterAdminStatus(ids: string[], status: number) {
  return put('/admin/users/batch/status', { ids, status })
}

// 批量删除系统管理员
export function batchDeleteMasterAdmins(ids: string[]) {
  return del('/admin/users/batch/delete', { data: { ids } })
}

// ==================== 系统管理员回收站接口 ====================

// 获取已删除的系统管理员列表（回收站）
export function getDeletedMasterAdminList(params: UserListParams) {
  return get<PageResult<UserItem>>('/admin/users/deleted', params)
}

// 恢复已删除的系统管理员
export function restoreMasterAdmin(id: string) {
  return put(`/admin/users/${id}/restore`)
}

// 永久删除系统管理员（从回收站彻底删除）
export function permanentDeleteMasterAdmin(id: string) {
  return del(`/admin/users/${id}/permanent`)
}

// ==================== 租户用户回收站接口（系统管理员使用） ====================

// 获取所有租户的已删除用户列表（回收站）
export function getAllDeletedTenantUsers(params: UserListParams) {
  return get<PageResult<UserItem>>('/admin/tenant-users/deleted/all', params)
}

// 恢复已删除的租户用户
export function restoreTenantUser(tenantId: string, userId: string) {
  return put(`/admin/tenant-users/${tenantId}/${userId}/restore`)
}

// 永久删除租户用户（从回收站彻底删除）
export function permanentDeleteTenantUser(tenantId: string, userId: string) {
  return del(`/admin/tenant-users/${tenantId}/${userId}/permanent`)
}

// ==================== 系统管理员跨租户用户管理接口 ====================

// 获取所有租户用户列表（系统管理员使用）
export function getAllTenantUsers(params: UserListParams) {
  return get<PageResult<UserItem>>('/admin/tenant-users/all', params)
}

// 获取指定租户的用户列表（系统管理员使用）
export function getTenantUsers(tenantId: string, params: UserListParams) {
  return get<PageResult<UserItem>>(`/admin/tenant-users/${tenantId}/list`, params)
}

// 为指定租户创建用户（系统管理员使用）
export function createTenantUser(tenantId: string, data: UserCreateParams) {
  return post<UserItem>('/admin/tenant-users', { ...data, tenant_id: tenantId })
}

// 更新指定租户的用户（系统管理员使用）
export function updateTenantUser(tenantId: string, userId: string, data: UserUpdateParams) {
  return put<UserItem>(`/admin/tenant-users/${tenantId}/${userId}/update`, data)
}

// 删除指定租户的用户（系统管理员使用）
export function deleteTenantUser(tenantId: string, userId: string) {
  return del(`/admin/tenant-users/${tenantId}/${userId}/delete`)
}

// 更新指定租户的用户状态（系统管理员使用）
export function updateTenantUserStatus(tenantId: string, userId: string, status: number) {
  return put(`/admin/tenant-users/${tenantId}/${userId}/status`, { status })
}

// 重置指定租户用户的密码（系统管理员使用）
export function resetTenantUserPassword(tenantId: string, userId: string, data: { new_password: string; confirm_password: string }) {
  return put(`/admin/tenant-users/${tenantId}/${userId}/reset-password`, data)
}

// 批量删除指定租户的用户（系统管理员使用）
export function batchDeleteTenantUsers(tenantId: string, ids: string[]) {
  return del(`/admin/tenant-users/${tenantId}/batch/delete`, { data: { ids } })
}

// 批量更新指定租户的用户状态（系统管理员使用）
export function batchUpdateTenantUserStatus(tenantId: string, ids: string[], status: number) {
  return put(`/admin/tenant-users/${tenantId}/batch/status`, { ids, status })
}

// ==================== 用户角色管理接口 ====================

// 解除用户角色绑定
export function unbindUserRole(userId: string, roleId: string) {
  return del(`/rbac/user-roles/${userId}/roles/${roleId}`)
}

// 获取用户已绑定的角色列表
export function getUserRoles(userId: string) {
  return get<{ id: string; name: string; code: string }[]>(`/rbac/user-roles/${userId}/roles`)
}

// ==================== 租户用户回收站接口 ====================

// 获取当前租户的已删除用户列表（回收站）
export function getDeletedUserList(params: UserListParams) {
  return get<PageResult<UserItem>>('/users/deleted', params)
}

// 恢复已删除的用户
export function restoreUser(id: string) {
  return put(`/users/${id}/restore`)
}

// 永久删除用户（从回收站彻底删除）
export function permanentDeleteUser(id: string) {
  return del(`/users/${id}/permanent`)
}