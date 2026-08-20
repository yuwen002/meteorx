import { get } from '@/api/request'

// 平台运营数据看板：获取总览统计（仅平台超级管理员）
export function getDashboardOverview() {
  return get('/admin/dashboard/overview')
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
