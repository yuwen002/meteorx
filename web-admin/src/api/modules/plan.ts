import { get, post, put, del } from '@/api/request'

export interface PlanItem {
  id: string
  name: string
  code: string
  description?: string
  user_limit: number
  price: number
  status: number
  subscriber_cnt?: number
  created_at?: string
  updated_at?: string
}

export interface CurrentPlan {
  tenant_id: string
  plan_id: string
  plan_name: string
  plan_code: string
  user_limit: number
  current_users: number
  started_at?: string
  expires_at?: string
  status: number
  effective_days?: number
}

export interface CreatePlanParams {
  name: string
  code: string
  description?: string
  user_limit: number
  price: number
  status: number
}

export interface UpdatePlanParams {
  name?: string
  description?: string
  user_limit?: number
  price?: number
  status?: number
}

export interface AssignPlanParams {
  plan_id: string
  expires_at?: string
}

// ==================== 套餐管理接口（系统管理员使用）====================

// 获取套餐列表
export function getPlanList(params?: {
  page?: number
  page_size?: number
  keyword?: string
  status?: number
}) {
  return get<PageResult<PlanItem>>('/admin/plans', params)
}

// 获取启用套餐下拉列表
export function getPlanSelect() {
  return get<PlanItem[]>('/admin/plans/select')
}

// 创建套餐
export function createPlan(data: CreatePlanParams) {
  return post<PlanItem>('/admin/plans', data)
}

// 更新套餐
export function updatePlan(id: string, data: UpdatePlanParams) {
  return put<PlanItem>(`/admin/plans/${id}/update`, data)
}

// 删除套餐
export function deletePlan(id: string) {
  return del(`/admin/plans/${id}/delete`)
}

// 查询租户当前套餐
export function getTenantPlan(tenantId: string) {
  return get<CurrentPlan>(`/admin/tenants-plan/${tenantId}`)
}

// 为租户分配/变更套餐
export function assignTenantPlan(tenantId: string, data: AssignPlanParams) {
  return put(`/admin/tenants-plan/${tenantId}`, data)
}

// 当前租户（自己）的套餐与用量
export function getMyPlan() {
  return get<CurrentPlan>('/tenant/current/plan')
}

interface PageResult<T> {
  data?: T[]
  list?: T[]
  pagination: {
    page: number
    page_size: number
    total: number
  }
}