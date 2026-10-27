import { get, post, del } from '../request'

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

// 获取审计日志列表（分页，支持多维度筛选）
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
  return get<PaginatedResponse<AuditLogItem>>('/audit/logs', params).then((res) => {
    return {
      items: res.data,
      page: res.pagination.page,
      page_size: res.pagination.page_size,
      total: res.pagination.total
    }
  })
}

// 获取审计日志详情
export function getAuditLogDetail(id: string): Promise<AuditLogItem> {
  return get(`/audit/logs/${id}`)
}

// 获取审计统计概览
export function getAuditStats(): Promise<AuditLogStats> {
  return get('/audit/stats')
}

// 获取审计看板数据（含趋势和模块分布）
export function getAuditDashboard(days?: number): Promise<AuditDashboardData> {
  return get('/audit/dashboard', { days: days || 7 })
}

// 清理指定天数之前的审计日志
export function cleanupAuditLogs(days: number): Promise<{ deleted_count: number }> {
  return del('/audit/logs/cleanup', { params: { days } })
}

// 导出审计日志（返回下载 URL）
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

// ---------------------- 扩展 API ----------------------

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

// 获取用户操作时间线（按日期分组）
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

// 获取详细统计数据（含风险等级、小时分布、用户活跃度）
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

// 获取异常行为日志（可设置阈值和时间窗口）
export function getAnomalyLogs(params?: {
  threshold?: number
  window_minutes?: number
}): Promise<AnomalyLogItem[]> {
  return get('/audit/anomalies', params)
}

// 告警统计
export interface AlertRuleCount {
  rule_id: string
  rule_name: string
  count: number
}

export interface AlertStatsResult {
  total_alerts: number
  today_alerts: number
  notified_count: number
  pending_count: number
  risk_level_stats: Record<string, number>
  rule_stats: Record<string, number>
  trend: TrendPoint[]
  top_rules: AlertRuleCount[]
}

/**
 * 零数据兜底。后端 Go 的 nil 切片会序列化成 null（trend / top_rules 曾因此把告警页整页打崩），
 * 在 API 边界统一收敛成空数组/空对象，页面里的 .map 就不会再拿到 null。
 */
function normalizeAlertStats(raw: Partial<AlertStatsResult> | null | undefined): AlertStatsResult {
  return {
    total_alerts: raw?.total_alerts ?? 0,
    today_alerts: raw?.today_alerts ?? 0,
    notified_count: raw?.notified_count ?? 0,
    pending_count: raw?.pending_count ?? 0,
    risk_level_stats: raw?.risk_level_stats ?? {},
    rule_stats: raw?.rule_stats ?? {},
    trend: Array.isArray(raw?.trend) ? raw.trend : [],
    top_rules: Array.isArray(raw?.top_rules) ? raw.top_rules : []
  }
}

// 获取告警统计数据（含风险等级分布和规则命中排名）
export function getAlertStats(days?: number): Promise<AlertStatsResult> {
  return get<Partial<AlertStatsResult>>('/audit/alerts/stats', { days: days || 7 }).then(normalizeAlertStats)
}

// ---------------------- 审计完整性校验 ----------------------

export type IntegrityVerdict = 'verified' | 'tampered' | 'pending_rehash' | 'not_configured'

export interface IntegrityVerifyResult {
  enabled: boolean
  verdict: IntegrityVerdict | string
  checked_rows: number
  legacy_rows: number
  tampered_rows: number
  tampered_samples: string[]
  pending_rehash_rows: number
  pending_rehash_samples: string[]
  anchors_checked: number
  anomaly_anchor_ids: string[]
  anomalies: string[]
  covered_from: string
  covered_to: string
  message: string
}

// 密钥轮换 rehash 结果：重算到新密钥的行数、重建的锚点数
export interface IntegrityRehashResult {
  rehashed_rows: number
  anchors_rebuilt: number
  message: string
}

/**
 * 零值兜底。后端 Go 的 nil 切片（tampered_samples / anomaly_anchor_ids / anomalies）
 * 会序列化成 null，页面直接对这些字段 v-for / .length 会报错，在 API 边界统一收敛成空数组。
 */
function normalizeIntegrityResult(raw: Partial<IntegrityVerifyResult> | null | undefined): IntegrityVerifyResult {
  return {
    enabled: raw?.enabled ?? false,
    verdict: raw?.verdict ?? 'not_configured',
    checked_rows: raw?.checked_rows ?? 0,
    legacy_rows: raw?.legacy_rows ?? 0,
    tampered_rows: raw?.tampered_rows ?? 0,
    tampered_samples: Array.isArray(raw?.tampered_samples) ? raw.tampered_samples : [],
    pending_rehash_rows: raw?.pending_rehash_rows ?? 0,
    pending_rehash_samples: Array.isArray(raw?.pending_rehash_samples) ? raw.pending_rehash_samples : [],
    anchors_checked: raw?.anchors_checked ?? 0,
    anomaly_anchor_ids: Array.isArray(raw?.anomaly_anchor_ids) ? raw.anomaly_anchor_ids : [],
    anomalies: Array.isArray(raw?.anomalies) ? raw.anomalies : [],
    covered_from: raw?.covered_from ?? '',
    covered_to: raw?.covered_to ?? '',
    message: raw?.message ?? ''
  }
}

// 触发审计日志防篡改完整性校验（行级重算 + 锚点链），仅平台超管可调
export function verifyAuditIntegrity(): Promise<IntegrityVerifyResult> {
  return post<Partial<IntegrityVerifyResult>>('/audit/integrity/verify').then(normalizeIntegrityResult)
}

/**
 * 密钥轮换 rehash：把仅匹配旧 pepper 的历史行重算到新 pepper 并重建锚点链，仅平台超管可调。
 * 后端返回 nil 数值在边界兜底为 0。
 */
function normalizeRehashResult(raw: Partial<IntegrityRehashResult> | null | undefined): IntegrityRehashResult {
  return {
    rehashed_rows: raw?.rehashed_rows ?? 0,
    anchors_rebuilt: raw?.anchors_rebuilt ?? 0,
    message: raw?.message ?? ''
  }
}

export function rehashAuditIntegrity(): Promise<IntegrityRehashResult> {
  return post<Partial<IntegrityRehashResult>>('/audit/integrity/rehash').then(normalizeRehashResult)
}