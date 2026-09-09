import { get, del } from '../request'

export interface AuditLogItem {
  id: string
  user_id: string
  username: string
  tenant_id: string
  module: string
  action: string
  resource: string
  resource_id: string
  method: string
  path: string
  request_body: string
  response_body: string
  status_code: number
  result: string
  error_message: string
  client_ip: string
  ip_location: string
  user_agent: string
  device_info: string
  duration: number
  session_id: string
  request_id: string
  trace_id: string
  referer: string
  risk_level: string
  tags: string
  created_at: string
}

export interface AuditLogListResult {
  items: AuditLogItem[]
  page: number
  page_size: number
  total: number
}

export interface AuditLogStats {
  total_count: number
  today_count: number
  action_stats: Record<string, number>
  module_stats: Record<string, number>
  result_stats: Record<string, number>
}

export interface TrendPoint {
  date: string
  count: number
  success: number
  failure: number
}

export interface ModuleCount {
  module: string
  count: number
}

export interface AuditDashboardData {
  total_count: number
  today_count: number
  action_stats: Record<string, number>
  module_stats: Record<string, number>
  result_stats: Record<string, number>
  trend: TrendPoint[]
  top_modules: ModuleCount[]
}

export interface PaginatedResponse<T> {
  data: T[]
  pagination: {
    page: number
    page_size: number
    total: number
  }
}

export function getAuditLogList(params: {
  page?: number
  page_size?: number
  user_id?: string
  username?: string
  tenant_id?: string
  module?: string
  action?: string
  result?: string
  risk_level?: string
  start_time?: string
  end_time?: string
  keyword?: string
}): Promise<AuditLogListResult> {
  return get('/audit/logs', params).then((res: PaginatedResponse<AuditLogItem>) => {
    return {
      items: res.data,
      page: res.pagination.page,
      page_size: res.pagination.page_size,
      total: res.pagination.total
    }
  })
}

export function getAuditLogDetail(id: string): Promise<AuditLogItem> {
  return get(`/audit/logs/${id}`)
}

export function getAuditStats(): Promise<AuditLogStats> {
  return get('/audit/stats')
}

export function getAuditDashboard(days?: number): Promise<AuditDashboardData> {
  return get('/audit/dashboard', { days: days || 7 })
}

export function cleanupAuditLogs(days: number): Promise<{ deleted_count: number }> {
  return del('/audit/logs/cleanup', { params: { days } })
}

export function exportAuditLogs(params: {
  format?: 'csv'
  module?: string
  action?: string
  result?: string
  start_time?: string
  end_time?: string
  keyword?: string
}): string {
  const query = new URLSearchParams()
  if (params.format) query.append('format', params.format)
  if (params.module) query.append('module', params.module)
  if (params.action) query.append('action', params.action)
  if (params.result) query.append('result', params.result)
  if (params.start_time) query.append('start_time', params.start_time)
  if (params.end_time) query.append('end_time', params.end_time)
  if (params.keyword) query.append('keyword', params.keyword)
  
  return `/api/v1/audit/logs/export?${query.toString()}`
}

// ---------------------- 新增 API ----------------------

// 用户时间线
export interface TimelineItem {
  date: string
  count: number
  success: number
  failure: number
  logs: AuditLogItem[]
}

export interface UserTimelineResult {
  items: TimelineItem[]
  total: number
  pages: number
}

export function getUserTimeline(params: {
  user_id: string
  page?: number
  page_size?: number
  start_time?: string
  end_time?: string
}): Promise<UserTimelineResult> {
  return get('/audit/user-timeline', params)
}

// 详细统计
export interface UserActivityStat {
  user_id: string
  username: string
  count: number
  failures: number
}

export interface DetailedStats {
  total_count: number
  today_count: number
  action_stats: Record<string, number>
  module_stats: Record<string, number>
  result_stats: Record<string, number>
  risk_level_stats: Record<string, number>
  hourly_stats: Record<string, number>
  user_activity: UserActivityStat[]
  trend: TrendPoint[]
}

export function getDetailedStats(days?: number): Promise<DetailedStats> {
  return get('/audit/detailed-stats', { days: days || 7 })
}

// 异常检测
export interface AnomalyLogItem {
  user_id: string
  username: string
  anomaly_type: string
  anomaly_label: string
  failure_count: number
  total_count: number
  window_minutes: number
  first_seen: string
  last_seen: string
  risk_level: string
  details: string
}

export function getAnomalyLogs(params?: {
  threshold?: number
  window_minutes?: number
}): Promise<AnomalyLogItem[]> {
  return get('/audit/anomalies', params)
}