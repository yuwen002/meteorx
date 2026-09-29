import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getAlertStats, getDetailedStats, getAnomalyLogs, type AlertStatsResult, type DetailedStats, type AnomalyLogItem } from './audit'

vi.mock('../request', () => ({
  get: vi.fn(),
  del: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

import { get } from '../request'

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