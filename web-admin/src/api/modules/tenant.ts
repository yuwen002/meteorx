import { get, post, put, del, type ApiResult } from '@/api/request'

export interface TenantItem {
  id: string
  name: string
  domain: string
  status: number
  description?: string
  contact_email?: string
  region?: string
  logo?: string
  extra?: string
  created_at?: string
  updated_at?: string
  deleted_at?: string
}

export interface TenantListParams {
  page?: number
  page_size?: number
  name?: string
}

export interface CreateTenantParams {
  name: string
  domain: string
  description?: string
  contact_email?: string
  region?: string
  logo?: string
  status?: number
  extra?: string
  admin_user: {
    username: string
    password: string
    nickname: string
    email?: string
  }
}

export interface UpdateTenantParams {
  name?: string
  domain?: string
  description?: string
  contact_email?: string
  region?: string
  logo?: string
  extra?: string
}

export interface BatchUpdateStatusParams {
  ids: string[]
  status: number
}

export interface BatchDeleteParams {
  ids: string[]
}

// ==================== 租户管理接口（系统管理员使用）====================

// 获取租户列表
export function getTenantList(params: TenantListParams) {
  return get<PageResult<TenantItem>>('/admin/tenants', params)
}

// 创建租户
export function createTenant(data: CreateTenantParams) {
  return post<TenantItem>('/admin/tenants', data)
}

// 获取租户详情
export function getTenantDetail(id: string) {
  return get<TenantItem>(`/admin/tenants/${id}/detail`)
}

// 更新租户
export function updateTenant(id: string, data: UpdateTenantParams) {
  return put<TenantItem>(`/admin/tenants/${id}/update`, data)
}

// 删除租户（软删除）
export function deleteTenant(id: string) {
  return del(`/admin/tenants/${id}/delete`)
}

// 更新租户状态
export function updateTenantStatus(id: string, status: number) {
  return put(`/admin/tenants/${id}/status`, { status })
}

// 批量更新租户状态
export function batchUpdateTenantStatus(data: BatchUpdateStatusParams) {
  return put('/admin/tenants/batch/status', data)
}

// 批量删除租户
export function batchDeleteTenants(data: BatchDeleteParams) {
  return del('/admin/tenants/batch', { data })
}

// ==================== 租户回收站接口 ====================

// 获取已删除的租户列表
export function getDeletedTenantList(params: TenantListParams) {
  return get<PageResult<TenantItem>>('/admin/tenants/deleted', params)
}

// 恢复已删除的租户
export function restoreTenant(id: string) {
  return put(`/admin/tenants/${id}/restore`)
}

// ==================== 当前租户接口（普通用户）====================

// 获取当前租户详情
export function getCurrentTenant() {
  return get<TenantItem>('/tenants/current')
}

// 更新当前租户信息
export function updateCurrentTenant(data: UpdateTenantParams) {
  return put<TenantItem>('/tenants/current', data)
}

// 类型定义辅助
interface PageResult<T> {
  data: T[]
  pagination: {
    page: number
    page_size: number
    total: number
  }
}