import { get, post } from '@/api/request'

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

// ---------- 增长趋势看板（仅平台超级管理员）----------

/** 单日增长趋势数据点（与后端 dto.TrendDayResp 对齐） */
export interface TrendDay {
  date: string
  new_tenants: number
  new_users: number
  audit_logs: number
}

/** 增长趋势响应（与后端 dto.GrowthTrendResp 对齐） */
export interface GrowthTrend {
  days: number
  list: TrendDay[]
}

// 获取最近 N 天的增长趋势时间序列（默认 30，上限 365）
export function getGrowthTrend(days?: number) {
  return get<GrowthTrend>('/admin/dashboard/trend', days ? { days } : {})
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

// ---------- 慢查询看板（仅平台超级管理员）----------

/** 单条慢查询记录（与后端 dbmonitor.Entry 对齐） */
export interface SlowQueryEntry {
  sql: string
  duration_ms: number
  rows: number
  error?: string
  trace_id?: string
  time: string
}

/** 慢查询整体统计（与后端 dbmonitor.Stats 对齐） */
export interface SlowQueryStats {
  threshold_ms: number
  total_queries: number
  slow_queries: number
  max_duration_ms: number
  avg_slow_ms: number
  capacity: number
}

/** 慢查询看板快照 */
export interface SlowQuerySnapshot {
  stats: SlowQueryStats
  entries: SlowQueryEntry[]
}

// 获取慢查询看板快照
export function getSlowQueries() {
  return get<SlowQuerySnapshot>('/admin/dashboard/slow-queries')
}

// 重置慢查询统计缓冲区
export function resetSlowQueries() {
  return post<{ message: string }>('/admin/dashboard/slow-queries/reset')
}
