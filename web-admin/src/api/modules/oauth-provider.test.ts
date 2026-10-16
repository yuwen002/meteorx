import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listOAuthProviderConfigs,
  getOAuthProviderConfig,
  createOAuthProviderConfig,
  updateOAuthProviderConfig,
  deleteOAuthProviderConfig,
  setOAuthProviderStatus,
  providerKindOptions,
  type OAuthProviderItem,
  type OAuthProviderSaveParams,
} from './oauth-provider'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
  default: { get: vi.fn() },
}))

import { get, post, put, del } from '@/api/request'

const sampleProvider: OAuthProviderItem = {
  code: 'github',
  kind: 'github',
  display_name: 'GitHub',
  enabled: true,
  client_id: 'cid-1',
  client_secret_mask: 'enc:v1:****abcd',
  has_secret: true,
  redirect_url: 'https://app.example.com/oauth/callback',
  editable_endpoint: false,
  created_at: '2026-01-01 09:00:00',
  updated_at: '2026-01-01 09:00:00',
}

describe('OAuth Provider API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('listOAuthProviderConfigs 应 GET /admin/oauth-providers 并透传列表', async () => {
    vi.mocked(get).mockResolvedValue({ providers: [sampleProvider] })
    const res = await listOAuthProviderConfigs()
    expect(get).toHaveBeenCalledWith('/admin/oauth-providers')
    expect(res.providers[0].code).toBe('github')
  })

  it('getOAuthProviderConfig 应 GET /admin/oauth-providers/{code}', async () => {
    vi.mocked(get).mockResolvedValue(sampleProvider)
    const res = await getOAuthProviderConfig('github')
    expect(get).toHaveBeenCalledWith('/admin/oauth-providers/github')
    expect(res.enabled).toBe(true)
  })

  it('createOAuthProviderConfig 应 POST /admin/oauth-providers 并透传请求体', async () => {
    const payload: OAuthProviderSaveParams = {
      kind: 'oidc',
      code: 'oidc_corp',
      display_name: '企业登录',
      enabled: true,
      client_id: 'cid-2',
      client_secret: 'plain-secret',
      redirect_url: 'https://app.example.com/oauth/callback',
      auth_url: 'https://idp.example.com/authorize',
      token_url: 'https://idp.example.com/token',
      userinfo_url: 'https://idp.example.com/userinfo',
      scopes: ['openid', 'email'],
    }
    vi.mocked(post).mockResolvedValue(undefined)
    await createOAuthProviderConfig(payload)
    expect(post).toHaveBeenCalledWith('/admin/oauth-providers', payload)
  })

  it('updateOAuthProviderConfig 应 PUT /admin/oauth-providers/{code}', async () => {
    const payload: OAuthProviderSaveParams = {
      kind: 'github',
      code: 'github',
      enabled: false,
      client_id: 'cid-1',
      redirect_url: 'https://app.example.com/oauth/callback',
    }
    vi.mocked(put).mockResolvedValue(undefined)
    await updateOAuthProviderConfig('github', payload)
    expect(put).toHaveBeenCalledWith('/admin/oauth-providers/github', payload)
  })

  it('deleteOAuthProviderConfig 应 DELETE /admin/oauth-providers/{code}', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await deleteOAuthProviderConfig('github')
    expect(del).toHaveBeenCalledWith('/admin/oauth-providers/github')
  })

  it('setOAuthProviderStatus 应 PUT 状态端点并携带 enabled', async () => {
    vi.mocked(put).mockResolvedValue(undefined)
    await setOAuthProviderStatus('github', false)
    expect(put).toHaveBeenCalledWith('/admin/oauth-providers/github/status', { enabled: false })
  })

  it('providerKindOptions 应包含 8 类 provider，且仅通用 OIDC 非固定', () => {
    expect(providerKindOptions).toHaveLength(8)
    expect(providerKindOptions.find(o => o.value === 'oidc')?.fixed).toBe(false)
    expect(providerKindOptions.filter(o => !o.fixed)).toHaveLength(1)
  })
})
