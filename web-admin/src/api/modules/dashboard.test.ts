import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getDashboardOverview,
  getTenantHealthList,
  getTenantHealthDetail,
  getSlowQueries,
  resetSlowQueries,
  type DashboardOverview,
  type TenantHealthList,
  type TenantHealthItem,
  type SlowQuerySnapshot,
} from './dashboard'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
}))

import { get, post } from '@/api/request'

const sampleHealth: TenantHealthItem = {
  tenant_id: 'tn-1',
  tenant_name: '测试租户',
  status: 1,
  plan_name: '专业版',
  subscription_status: 1,
  days_remaining: 30,
  user_count: 12,
  active_user_count: 5,
  task_count: 40,
  task_completed_count: 25,
  wiki_doc_count: 8,
  storage_used: 2048,
  audit_count_7d: 66,
  health_score: 82,
  health_level: 'healthy',
}

const sampleOverview: DashboardOverview = {
  tenant_stats: { total: 10, enabled: 9, disabled: 1, today_new: 2, week_new: 5, month_new: 8 },
  user_stats: { total: 100, today_new: 3, week_new: 20, month_new: 60 },
  subscription_stats: { total: 10, active: 8, expired: 1, cancelled: 1, active_tenant: 8 },
  audit_stats: { total: 999, today: 12, success: 980, failure: 19, action_stats: { login: 500 }, module_stats: { auth: 500 } },
}

describe('Dashboard API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getDashboardOverview 应 GET /admin/dashboard/overview 并透传统计结构', async () => {
    vi.mocked(get).mockResolvedValue(sampleOverview)
    const res = await getDashboardOverview()
    expect(get).toHaveBeenCalledWith('/admin/dashboard/overview')
    expect(res.tenant_stats.total).toBe(10)
    expect(res.audit_stats.action_stats.login).toBe(500)
  })

  it('getTenantHealthList 应携带分页参数请求 tenants-health', async () => {
    const listResp: TenantHealthList = {
      items: [sampleHealth], total: 1, page: 1, page_size: 20,
      healthy: 1, moderate: 0, at_risk: 0,
    }
    vi.mocked(get).mockResolvedValue(listResp)
    const res = await getTenantHealthList({ page: 1, page_size: 20, keyword: '测试' })
    expect(get).toHaveBeenCalledWith('/admin/dashboard/tenants-health', { page: 1, page_size: 20, keyword: '测试' })
    expect(res.items[0].health_level).toBe('healthy')
    expect(res.healthy).toBe(1)
  })

  it('getTenantHealthList 无关键字时应透传 undefined 参数', async () => {
    vi.mocked(get).mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, healthy: 0, moderate: 0, at_risk: 0 })
    await getTenantHealthList({ page: 1, page_size: 20 })
    expect(get).toHaveBeenCalledWith('/admin/dashboard/tenants-health', { page: 1, page_size: 20 })
  })

  it('getSlowQueries 应 GET slow-queries 并返回统计与明细', async () => {
    const snap: SlowQuerySnapshot = {
      stats: { threshold_ms: 500, total_queries: 120, slow_queries: 3, max_duration_ms: 1200, avg_slow_ms: 800.5, capacity: 200 },
      entries: [{ sql: 'SELECT * FROM t', duration_ms: 1200, rows: 10, time: '2026-10-13T00:00:00Z' }],
    }
    vi.mocked(get).mockResolvedValue(snap)
    const res = await getSlowQueries()
    expect(get).toHaveBeenCalledWith('/admin/dashboard/slow-queries')
    expect(res.stats.slow_queries).toBe(3)
    expect(res.entries[0].duration_ms).toBe(1200)
  })

  it('resetSlowQueries 应 POST slow-queries/reset', async () => {
    vi.mocked(post).mockResolvedValue({ message: '慢查询统计已重置' })
    const res = await resetSlowQueries()
    expect(post).toHaveBeenCalledWith('/admin/dashboard/slow-queries/reset')
    expect(res.message).toContain('重置')
  })

  it('getTenantHealthDetail 应 GET tenants-health/{tenantId}', async () => {
    vi.mocked(get).mockResolvedValue(sampleHealth)
    const res = await getTenantHealthDetail('tn-1')
    expect(get).toHaveBeenCalledWith('/admin/dashboard/tenants-health/tn-1')
    expect(res.health_score).toBe(82)
    expect(res.audit_count_7d).toBe(66)
  })
})
