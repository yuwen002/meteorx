import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// 隔离网络层：brand store 的服务端同步依赖 getTenantSettings
vi.mock('@/api/modules/tenant', () => ({ getTenantSettings: vi.fn() }))

import { getTenantSettings } from '@/api/modules/tenant'
import { useBrandStore } from './brand'

describe('brand store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.removeItem('meteorx_brand')
    vi.clearAllMocks()
  })

  it('applyServerBrand uses server values, applies DOM title and caches to storage', () => {
    const s = useBrandStore()
    s.applyServerBrand({ tenant_name: 'Acme', logo: 'http://logo.png', primary_color: '#123456' })
    expect(s.brand.name).toBe('Acme')
    expect(s.brand.logo).toBe('http://logo.png')
    expect(s.brand.primaryColor).toBe('#123456')
    expect(document.title).toBe('Acme')
    expect(JSON.parse(localStorage.getItem('meteorx_brand') as string).name).toBe('Acme')
  })

  it('applyServerBrand falls back to defaults for empty server fields', () => {
    const s = useBrandStore()
    s.applyServerBrand({ tenant_name: '  ', logo: '', primary_color: '' })
    expect(s.brand.name).toBe('MeteorX')
    expect(s.brand.primaryColor).toBe('#409eff')
  })

  it('syncBrandFromServer applies server brand on success', async () => {
    vi.mocked(getTenantSettings).mockResolvedValue({
      tenant_name: 'TenantX',
      logo: '',
      primary_color: '#abcdef'
    } as never)
    const s = useBrandStore()
    await s.syncBrandFromServer()
    expect(s.brand.name).toBe('TenantX')
    expect(s.brand.primaryColor).toBe('#abcdef')
  })

  it('syncBrandFromServer keeps current brand on failure (superadmin/no-tenant fallback)', async () => {
    const s = useBrandStore()
    s.setBrand({ name: 'LocalOnly' })
    vi.mocked(getTenantSettings).mockRejectedValue(new Error('401 unauthorized'))
    await s.syncBrandFromServer()
    expect(s.brand.name).toBe('LocalOnly')
  })
})
