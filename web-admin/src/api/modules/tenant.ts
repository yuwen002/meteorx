import { get, post, put, del } from '@/api/request'
import type { PaginatedResult } from '@/types/pagination'

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
  plan_name?: string
  plan_expired?: boolean
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
  return get<PaginatedResult<TenantItem>>('/admin/tenants', params)
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
  return get<PaginatedResult<TenantItem>>('/admin/tenants/deleted', params)
}

// 恢复已删除的租户
export function restoreTenant(id: string) {
  return put(`/admin/tenants/${id}/restore`)
}

// ==================== 注销申请审批接口（平台管理员）====================

export interface CancelRequestItem {
  id: string
  tenant_id: string
  tenant_name: string
  reason: string
  status: number
  status_text: string
  approver_id: string
  review_remark: string
  effective_at: string
  applied_at: string
  approved_at: string
  completed_at: string
  created_at: string
  updated_at: string
}

export interface CancelRequestListParams {
  page?: number
  page_size?: number
  status?: number
  keyword?: string
}

// 获取注销申请列表
export function getCancelRequestList(params: CancelRequestListParams) {
  return get<PaginatedResult<CancelRequestItem>>('/admin/cancel-requests', params)
}

// 通过注销申请（effective_days 为生效天数，0 表示立即执行）
export function approveCancelRequest(id: string, data: { review_remark?: string; effective_days: number }) {
  return put<CancelRequestItem>(`/admin/cancel-requests/${id}/approve`, data)
}

// 驳回注销申请
export function rejectCancelRequest(id: string, data: { review_remark?: string }) {
  return put<CancelRequestItem>(`/admin/cancel-requests/${id}/reject`, data)
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

// ==================== 租户设置接口 ====================

export interface TenantSettings {
  id: string
  tenant_id: string
  logo: string
  favicon: string
  primary_color: string
  theme: string
  language: string
  timezone: string
  description: string
  welcome_text: string
  contact_name: string
  contact_email: string
  contact_phone: string
  address: string
  extra: string
  created_at: string
  updated_at: string
}

export interface UpdateTenantSettingsReq {
  logo?: string
  favicon?: string
  primary_color?: string
  theme?: string
  language?: string
  timezone?: string
  description?: string
  welcome_text?: string
  contact_name?: string
  contact_email?: string
  contact_phone?: string
  address?: string
  extra?: string
}

// 获取租户设置
export function getTenantSettings() {
  return get<TenantSettings>('/tenant-settings')
}

// 更新租户设置
export function updateTenantSettings(data: UpdateTenantSettingsReq) {
  return put<TenantSettings>('/tenant-settings', data)
}

