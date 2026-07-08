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
  user_agent: string
  duration: number
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

export function getAuditLogList(params: {
  page?: number
  page_size?: number
  user_id?: string
  username?: string
  tenant_id?: string
  module?: string
  action?: string
  result?: string
  start_time?: string
  end_time?: string
  keyword?: string
}): Promise<AuditLogListResult> {
  return get('/audit/logs', params)
}

export function getAuditLogDetail(id: string): Promise<AuditLogItem> {
  return get(`/audit/logs/${id}`)
}

export function getAuditStats(): Promise<AuditLogStats> {
  return get('/audit/stats')
}

export function cleanupAuditLogs(days: number): Promise<{ deleted_count: number }> {
  return del('/audit/logs/cleanup', { days })
}