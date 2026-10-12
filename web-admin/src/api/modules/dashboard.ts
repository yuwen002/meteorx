import { get } from '@/api/request'

// 平台运营数据看板：获取总览统计（仅平台超级管理员）
export function getDashboardOverview() {
  return get<DashboardOverview>('/admin/dashboard/overview')
}

// 看板响应数据结构
export interface DashboardOverview {
  tenant_stats: {
    total: number
    enabled: number
    disabled: number
    today_new: number
    week_new: number
    month_new: number
  }
  user_stats: {
    total: number
    today_new: number
    week_new: number
    month_new: number
  }
  subscription_stats: {
    total: number
    active: number
    expired: number
    cancelled: number
    active_tenant: number
  }
  audit_stats: {
    total: number
    today: number
    success: number
    failure: number
    action_stats: Record<string, number>
    module_stats: Record<string, number>
  }
}

// ---------- 租户健康度看板（仅平台超级管理员）----------

/** 单个租户健康度条目（与后端 dto.TenantHealthItemResp 对齐） */
export interface TenantHealthItem {
  tenant_id: string
  tenant_name: string
  status: number
  plan_name: string
  subscription_status: number
  subscription_expires_at?: string | null
  days_remaining: number
  user_count: number
  active_user_count: number
  task_count: number
  task_completed_count: number
  wiki_doc_count: number
  storage_used: number
  audit_count_7d: number
  last_activity_at?: string | null
  health_score: number
  health_level: string
}

/** 租户健康度列表响应 */
export interface TenantHealthList {
  items: TenantHealthItem[]
  total: number
  page: number
  page_size: number
  healthy: number
  moderate: number
  at_risk: number
}

/** 租户健康度列表查询参数 */
export interface TenantHealthParams {
  page?: number
  page_size?: number
  keyword?: string
}

// 分页获取租户健康度列表
export function getTenantHealthList(params: TenantHealthParams) {
  return get<TenantHealthList>('/admin/dashboard/tenants-health', params)
}

// 获取单个租户健康度详情
export function getTenantHealthDetail(tenantId: string) {
  return get<TenantHealthItem>(`/admin/dashboard/tenants-health/${tenantId}`)
}
