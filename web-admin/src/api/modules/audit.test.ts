import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getAlertStats, getDetailedStats, getAnomalyLogs, getAuditLogList, getAuditLogDetail, getAuditStats, getAuditDashboard, cleanupAuditLogs, exportAuditLogs, getUserTimeline, verifyAuditIntegrity, rehashAuditIntegrity, type AlertStatsResult, type DetailedStats, type AnomalyLogItem } from './audit'

vi.mock('../request', () => ({
  get: vi.fn(),
  del: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

import { get, del, post } from '../request'

describe('Audit API - Alert Stats', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('getAlertStats should call /audit/alerts/stats with days param', async () => {
    const mockData: AlertStatsResult = {
      total_alerts: 100,
      today_alerts: 12,
      notified_count: 80,
      pending_count: 20,
      risk_level_stats: { low: 30, medium: 40, high: 20, critical: 10 },
      rule_stats: { 'ar-001': 60, 'ar-002': 40 },
      trend: [
        { date: '2026-09-28', count: 8, success: 6, failure: 2 },
        { date: '2026-09-29', count: 12, success: 10, failure: 2 },
      ],
      top_rules: [
        { rule_id: 'ar-001', rule_name: 'High Risk Alert', count: 60 },
      ],
    }
    vi.mocked(get).mockResolvedValue(mockData)

    const result = await getAlertStats(7)
    expect(get).toHaveBeenCalledWith('/audit/alerts/stats', { days: 7 })
    expect(result.total_alerts).toBe(100)
    expect(result.today_alerts).toBe(12)
    expect(result.risk_level_stats.critical).toBe(10)
    expect(result.trend).toHaveLength(2)
    expect(result.top_rules[0].rule_name).toBe('High Risk Alert')
  })

  it('getAlertStats should default to 7 days', async () => {
    vi.mocked(get).mockResolvedValue({} as AlertStatsResult)

    await getAlertStats()
    expect(get).toHaveBeenCalledWith('/audit/alerts/stats', { days: 7 })
  })

  // 回归：Go 的 nil 切片序列化成 null，trend 为 null 时页面 trendMax 里的 .map 会抛 TypeError
  it('getAlertStats should normalize null trend/top_rules to empty arrays', async () => {
    vi.mocked(get).mockResolvedValue({
      total_alerts: 0,
      today_alerts: 0,
      notified_count: 0,
      pending_count: 0,
      risk_level_stats: null,
      rule_stats: null,
      trend: null,
      top_rules: null,
    } as unknown as AlertStatsResult)

    const result = await getAlertStats(7)
    expect(result.trend).toEqual([])
    expect(result.top_rules).toEqual([])
    expect(result.risk_level_stats).toEqual({})
    expect(result.rule_stats).toEqual({})
    // 页面直接对这些字段做 .map / 索引运算，崩不崩取决于这里
    expect(() => result.trend.map(p => p.count)).not.toThrow()
  })
})

describe('Audit API - Detailed Stats', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('getDetailedStats should call /audit/detailed-stats with days param', async () => {
    const mockData: DetailedStats = {
      total_count: 500,
      today_count: 50,
      action_stats: { login: 200, create: 150, delete: 50 },
      module_stats: { auth: 200, wiki: 150, file: 100 },
      result_stats: { success: 450, failure: 50 },
      risk_level_stats: { low: 300, medium: 100, high: 80, critical: 20 },
      hourly_stats: { '0': 10, '1': 5, '12': 30 },
      user_activity: [
        { user_id: 'u-001', username: 'alice', count: 100, failures: 5 },
      ],
      trend: [
        { date: '2026-09-29', count: 50, success: 45, failure: 5 },
      ],
    }
    vi.mocked(get).mockResolvedValue(mockData)

    const result = await getDetailedStats(14)
    expect(get).toHaveBeenCalledWith('/audit/detailed-stats', { days: 14 })
    expect(result.total_count).toBe(500)
    expect(result.risk_level_stats.critical).toBe(20)
    expect(result.user_activity[0].username).toBe('alice')
  })
})

describe('Audit API - Anomaly Logs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('getAnomalyLogs should call /audit/anomalies with params', async () => {
    const mockData: AnomalyLogItem[] = [
      {
        user_id: 'u-001',
        username: 'alice',
        anomaly_type: 'brute_force',
        anomaly_label: '暴力破解',
        failure_count: 15,
        total_count: 20,
        window_minutes: 30,
        first_seen: '2026-09-29T10:00:00Z',
        last_seen: '2026-09-29T10:30:00Z',
        risk_level: 'critical',
        details: '30分钟内15次登录失败',
      },
    ]
    vi.mocked(get).mockResolvedValue(mockData)

    const result = await getAnomalyLogs({ threshold: 10, window_minutes: 30 })
    expect(get).toHaveBeenCalledWith('/audit/anomalies', { threshold: 10, window_minutes: 30 })
    expect(result).toHaveLength(1)
    expect(result[0].anomaly_type).toBe('brute_force')
    expect(result[0].risk_level).toBe('critical')
  })

  it('getAnomalyLogs should work without params', async () => {
    vi.mocked(get).mockResolvedValue([])

    const result = await getAnomalyLogs()
    expect(get).toHaveBeenCalledWith('/audit/anomalies', undefined)
    expect(result).toHaveLength(0)
  })
})

describe('Audit API - Logs & Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('getAuditLogList should map paginated response to list result', async () => {
    vi.mocked(get).mockResolvedValue({
      data: [{ id: 'l1', username: 'alice' }],
      pagination: { page: 1, page_size: 20, total: 1 },
    })

    const result = await getAuditLogList({ page: 1, username: 'alice' })
    expect(get).toHaveBeenCalledWith('/audit/logs', { page: 1, username: 'alice' })
    expect(result.items).toHaveLength(1)
    expect(result.page).toBe(1)
    expect(result.page_size).toBe(20)
    expect(result.total).toBe(1)
  })

  it('getAuditLogDetail should GET the log by id', async () => {
    vi.mocked(get).mockResolvedValue({ id: 'l1' })
    const result = await getAuditLogDetail('l1')
    expect(get).toHaveBeenCalledWith('/audit/logs/l1')
    expect(result.id).toBe('l1')
  })

  it('getAuditStats should GET /audit/stats', async () => {
    vi.mocked(get).mockResolvedValue({ total_count: 10 })
    const result = await getAuditStats()
    expect(get).toHaveBeenCalledWith('/audit/stats')
    expect(result.total_count).toBe(10)
  })

  it('getAuditDashboard should pass days or default to 7', async () => {
    vi.mocked(get).mockResolvedValue({})
    await getAuditDashboard(30)
    expect(get).toHaveBeenCalledWith('/audit/dashboard', { days: 30 })

    await getAuditDashboard()
    expect(get).toHaveBeenCalledWith('/audit/dashboard', { days: 7 })
  })

  it('cleanupAuditLogs should DELETE with days param', async () => {
    vi.mocked(del).mockResolvedValue({ deleted_count: 5 })
    const result = await cleanupAuditLogs(30)
    expect(del).toHaveBeenCalledWith('/audit/logs/cleanup', { params: { days: 30 } })
    expect(result.deleted_count).toBe(5)
  })

  it('exportAuditLogs should build query string from provided fields', () => {
    const url = exportAuditLogs({ format: 'csv', module: 'auth', keyword: 'login' })
    expect(url).toBe('/api/v1/audit/logs/export?format=csv&module=auth&keyword=login')
  })

  it('exportAuditLogs should return empty query when no fields', () => {
    const url = exportAuditLogs({})
    expect(url).toBe('/api/v1/audit/logs/export?')
  })

  it('exportAuditLogs should include all optional filters', () => {
    const url = exportAuditLogs({
      format: 'csv',
      module: 'auth',
      action: 'login',
      result: 'success',
      start_time: '2026-01-01',
      end_time: '2026-01-02',
      keyword: 'k',
    })
    for (const frag of ['format=csv', 'module=auth', 'action=login', 'result=success', 'start_time=', 'end_time=', 'keyword=k']) {
      expect(url).toContain(frag)
    }
  })

  it('getUserTimeline should GET /audit/user-timeline with params', async () => {
    vi.mocked(get).mockResolvedValue({ items: [], total: 0, pages: 0 })
    const result = await getUserTimeline({ user_id: 'u1', page: 1 })
    expect(get).toHaveBeenCalledWith('/audit/user-timeline', { user_id: 'u1', page: 1 })
    expect(result.items).toHaveLength(0)
  })
})

describe('Audit API - Integrity Verify', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('verifyAuditIntegrity should POST /audit/integrity/verify and pass through result', async () => {
    vi.mocked(post).mockResolvedValue({
      enabled: true,
      verdict: 'tampered',
      checked_rows: 120,
      legacy_rows: 5,
      tampered_rows: 2,
      tampered_samples: ['l-001', 'l-002'],
      anchors_checked: 3,
      anomaly_anchor_ids: ['an-001'],
      anomalies: ['窗口摘要与锚点不符'],
      covered_from: '2026-01-01T00:00:00Z',
      covered_to: '2026-01-02T00:00:00Z',
      message: '检测到 2 行被篡改',
    })

    const result = await verifyAuditIntegrity()
    expect(post).toHaveBeenCalledWith('/audit/integrity/verify')
    expect(result.enabled).toBe(true)
    expect(result.verdict).toBe('tampered')
    expect(result.tampered_rows).toBe(2)
    expect(result.tampered_samples).toHaveLength(2)
    expect(result.anomaly_anchor_ids).toEqual(['an-001'])
  })

  it('verifyAuditIntegrity should normalize null arrays to empty (Go nil slice)', async () => {
    vi.mocked(post).mockResolvedValue({
      enabled: false,
      verdict: 'not_configured',
      checked_rows: null,
      tampered_samples: null,
      anomaly_anchor_ids: null,
      anomalies: null,
    })

    const result = await verifyAuditIntegrity()
    expect(result.enabled).toBe(false)
    expect(result.verdict).toBe('not_configured')
    expect(result.checked_rows).toBe(0)
    expect(result.tampered_samples).toEqual([])
    expect(result.anomaly_anchor_ids).toEqual([])
    expect(result.anomalies).toEqual([])
    // 页面直接对这些数组做 .length / v-for，崩不崩取决于这里
    expect(() => result.anomalies.map(a => a)).not.toThrow()
  })

  it('verifyAuditIntegrity should pass through pending_rehash rows/samples (key rotation)', async () => {
    vi.mocked(post).mockResolvedValue({
      enabled: true,
      verdict: 'pending_rehash',
      checked_rows: 50,
      pending_rehash_rows: 9,
      pending_rehash_samples: ['l-100', 'l-101'],
      tampered_rows: 0,
      message: '检测到 9 行待密钥重算',
    })

    const result = await verifyAuditIntegrity()
    expect(result.verdict).toBe('pending_rehash')
    expect(result.pending_rehash_rows).toBe(9)
    expect(result.pending_rehash_samples).toEqual(['l-100', 'l-101'])
  })

  it('verifyAuditIntegrity should normalize null pending_rehash_samples to empty', async () => {
    vi.mocked(post).mockResolvedValue({
      enabled: true,
      verdict: 'verified',
      pending_rehash_rows: null,
      pending_rehash_samples: null,
    })

    const result = await verifyAuditIntegrity()
    expect(result.pending_rehash_rows).toBe(0)
    expect(result.pending_rehash_samples).toEqual([])
    expect(() => result.pending_rehash_samples.map(s => s)).not.toThrow()
  })
})

describe('Audit API - Integrity Rehash', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('rehashAuditIntegrity should POST /audit/integrity/rehash and pass through result', async () => {
    vi.mocked(post).mockResolvedValue({
      rehashed_rows: 9,
      anchors_rebuilt: 3,
      message: '已重算 9 行并重建 3 个锚点',
    })

    const result = await rehashAuditIntegrity()
    expect(post).toHaveBeenCalledWith('/audit/integrity/rehash')
    expect(result.rehashed_rows).toBe(9)
    expect(result.anchors_rebuilt).toBe(3)
    expect(result.message).toBe('已重算 9 行并重建 3 个锚点')
  })

  it('rehashAuditIntegrity should normalize null numbers to 0 (Go nil)', async () => {
    vi.mocked(post).mockResolvedValue({
      rehashed_rows: null,
      anchors_rebuilt: null,
      message: null,
    })

    const result = await rehashAuditIntegrity()
    expect(result.rehashed_rows).toBe(0)
    expect(result.anchors_rebuilt).toBe(0)
    expect(result.message).toBe('')
  })
})