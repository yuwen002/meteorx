import { get, post, put, del, type ApiResult } from '@/api/request'

export interface UserItem {
  id: string
  tenant_id?: string
  username: string
  nickname?: string
  email?: string
  status?: number
  is_master?: boolean
  created_at?: string
  updated_at?: string
}

export interface UserCreateParams {
  tenant_id?: string
  username: string
  password: string
  nickname?: string
  email?: string
}

export interface UserUpdateParams {
  nickname?: string
  email?: string
  status?: number
  password?: string
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

// 获取用户统计信息
export function getUserStats() {
  return get<ApiResult<{ user_count: number }>>('/profile/stats')
}