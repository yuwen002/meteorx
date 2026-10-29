import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listSystemSettings,
  updateSystemSettings,
  getPublicSettings,
  type SettingItem,
} from './settings'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  put: vi.fn(),
}))

import { get, put } from '@/api/request'

const sampleSetting: SettingItem = {
  key: 'registration_enabled',
  value: 'true',
  value_type: 'bool',
  default_value: 'true',
  is_public: true,
  source: 'db',
  description: '是否开放用户自助注册',
  updated_at: '2026-01-01 09:00:00',
}

describe('系统配置 API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('listSystemSettings 应 GET /admin/settings 并透传列表', async () => {
    vi.mocked(get).mockResolvedValue([sampleSetting])
    const res = await listSystemSettings()
    expect(get).toHaveBeenCalledWith('/admin/settings')
    expect(res[0].key).toBe('registration_enabled')
  })

  it('updateSystemSettings 应 PUT /admin/settings 且请求体按后端契约包裹 settings 数组', async () => {
    const changes = [
      { key: 'registration_enabled', value: 'false' },
      { key: 'maintenance_message', value: '今晚升级' },
    ]
    vi.mocked(put).mockResolvedValue([sampleSetting])
    const res = await updateSystemSettings(changes)
    expect(put).toHaveBeenCalledWith('/admin/settings', { settings: changes })
    expect(res).toHaveLength(1)
  })

  it('getPublicSettings 应 GET 公开端点并返回类型化配置', async () => {
    vi.mocked(get).mockResolvedValue({ registration_enabled: true, maintenance_mode: false })
    const res = await getPublicSettings()
    expect(get).toHaveBeenCalledWith('/settings/public')
    // 布尔保持原生 JSON 类型（后端 parseTyped 还原）
    expect(res.registration_enabled).toBe(true)
    expect(res.maintenance_mode).toBe(false)
  })
})
