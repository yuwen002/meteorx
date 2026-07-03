import { get, post, put, del, type ApiResult } from '@/api/request'

export interface UserItem {
  id: string
  tenant_id?: string
  username: string
  nickname?: string
  email?: string
  status?: number
  is_master?: boolean
  roles?: string[]
  role_ids?: string[]
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
  return put<UserItem>(`/users/${id}`, data)
}

// 删除用户
export function deleteUser(id: string) {
  return del(`/users/${id}`)
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