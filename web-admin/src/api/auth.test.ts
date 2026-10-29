import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listOAuthAccounts,
  unbindOAuth,
  createAPIToken,
  listAPITokens,
  revokeAPIToken,
  login,
  logout,
  forgotPassword,
  resetPassword,
  getOAuthRedirectURL,
  register,
  registerUser,
  getOAuthTenants,
  oauthLogin,
  bindOAuth,
  refreshOAuthToken,
  sendEmailVerification,
  verifyEmail,
  setup2FA,
  enable2FA,
  disable2FA,
  getRecoveryCodeStatus,
  regenerateRecoveryCodes,
  type LoginParams,
  type RegisterParams,
  type OAuthAccountItem,
  type APITokenItem,
  type CreateAPITokenResult,
} from './auth'

vi.mock('./request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
  put: vi.fn(),
}))

import { get, post } from './request'

describe('OAuth Account API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('listOAuthAccounts should call /auth/oauth/accounts', async () => {
    const mockAccounts: OAuthAccountItem[] = [
      { id: 'oa-001', provider: 'google', email: 'alice@gmail.com', created_at: '2026-09-29T10:00:00Z' },
      { id: 'oa-002', provider: 'github', email: 'alice@github.com', created_at: '2026-09-29T11:00:00Z' },
    ]
    vi.mocked(get).mockResolvedValue({ accounts: mockAccounts })

    const result = await listOAuthAccounts()
    expect(get).toHaveBeenCalledWith('/auth/oauth/accounts')
    expect(result.accounts).toHaveLength(2)
    expect(result.accounts[0].provider).toBe('google')
    expect(result.accounts[1].provider).toBe('github')
  })

  it('unbindOAuth should call /auth/oauth/unbind with provider', async () => {
    vi.mocked(post).mockResolvedValue({})

    await unbindOAuth('google')
    expect(post).toHaveBeenCalledWith('/auth/oauth/unbind', { provider: 'google' })
  })

  it('unbindOAuth should work for github provider', async () => {
    vi.mocked(post).mockResolvedValue({})

    await unbindOAuth('github')
    expect(post).toHaveBeenCalledWith('/auth/oauth/unbind', { provider: 'github' })
  })
})

describe('API Token API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('createAPIToken should call /auth/tokens with params', async () => {
    const mockResult: CreateAPITokenResult = {
      id: 'at-001',
      name: 'CI/CD Token',
      token: 'mx_at_abc123xyz',
      allowed_paths: ['/users', '/files'],
      expires_at: '2026-12-29T00:00:00Z',
      created_at: '2026-09-29T12:00:00Z',
    }
    vi.mocked(post).mockResolvedValue(mockResult)

    const result = await createAPIToken({
      name: 'CI/CD Token',
      expires_in: '2160h',
      allowed_paths: ['/users', '/files'],
    })
    expect(post).toHaveBeenCalledWith('/auth/tokens', {
      name: 'CI/CD Token',
      expires_in: '2160h',
      allowed_paths: ['/users', '/files'],
    })
    expect(result.token).toBe('mx_at_abc123xyz')
    expect(result.id).toBe('at-001')
  })

  it('createAPIToken should work without optional params', async () => {
    const mockResult: CreateAPITokenResult = {
      id: 'at-002',
      name: 'Simple Token',
      token: 'mx_at_simple123',
      created_at: '2026-09-29T12:00:00Z',
    }
    vi.mocked(post).mockResolvedValue(mockResult)

    const result = await createAPIToken({ name: 'Simple Token' })
    expect(post).toHaveBeenCalledWith('/auth/tokens', { name: 'Simple Token' })
    expect(result.token).toBe('mx_at_simple123')
  })

  it('listAPITokens should call /auth/tokens', async () => {
    const mockTokens: APITokenItem[] = [
      {
        id: 'at-001',
        name: 'CI/CD Token',
        allowed_paths: ['/users'],
        last_used_at: '2026-09-28T12:00:00Z',
        expires_at: '2026-12-29T00:00:00Z',
        created_at: '2026-09-29T12:00:00Z',
        revoked: false,
      },
      {
        id: 'at-002',
        name: 'Old Token',
        created_at: '2026-06-01T00:00:00Z',
        revoked: true,
      },
    ]
    vi.mocked(get).mockResolvedValue({ tokens: mockTokens })

    const result = await listAPITokens()
    expect(get).toHaveBeenCalledWith('/auth/tokens')
    expect(result.tokens).toHaveLength(2)
    expect(result.tokens[0].revoked).toBe(false)
    expect(result.tokens[1].revoked).toBe(true)
  })

  it('revokeAPIToken should call /auth/tokens/revoke with id', async () => {
    vi.mocked(post).mockResolvedValue({})

    await revokeAPIToken({ id: 'at-001' })
    expect(post).toHaveBeenCalledWith('/auth/tokens/revoke', { id: 'at-001' })
  })
})

describe('Auth core API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('login should POST /auth/login with params', async () => {
    const params: LoginParams = { username: 'alice', password: 'secret' }
    vi.mocked(post).mockResolvedValue({ token: 't', user: { id: 'u1', username: 'alice' } })

    const result = await login(params)
    expect(post).toHaveBeenCalledWith('/auth/login', params)
    expect(result.token).toBe('t')
  })

  it('logout should POST /auth/logout', async () => {
    vi.mocked(post).mockResolvedValue({})
    await logout()
    expect(post).toHaveBeenCalledWith('/auth/logout')
  })

  it('forgotPassword should POST /auth/forgot-password with email', async () => {
    vi.mocked(post).mockResolvedValue({ message: 'ok' })
    await forgotPassword('a@b.com')
    expect(post).toHaveBeenCalledWith('/auth/forgot-password', { email: 'a@b.com' })
  })

  it('resetPassword should POST /auth/reset-password with token and new password', async () => {
    vi.mocked(post).mockResolvedValue({ message: 'ok' })
    await resetPassword('tok', 'newpw')
    expect(post).toHaveBeenCalledWith('/auth/reset-password', { token: 'tok', new_password: 'newpw' })
  })

  it('getOAuthRedirectURL should GET provider redirect', async () => {
    vi.mocked(get).mockResolvedValue({ url: 'https://oauth.example' })
    const result = await getOAuthRedirectURL('google')
    expect(get).toHaveBeenCalledWith('/auth/oauth/google/redirect')
    expect(result.url).toBe('https://oauth.example')
  })

  it('register should POST /tenants/register', async () => {
    const data: RegisterParams = {
      name: 'Acme',
      domain: 'acme',
      admin_user: { username: 'admin', password: 'pw', nickname: 'Admin', email: 'a@b.com' },
    }
    vi.mocked(post).mockResolvedValue({ id: 't1', name: 'Acme', domain: 'acme', admin_user: data.admin_user })
    await register(data)
    expect(post).toHaveBeenCalledWith('/tenants/register', data)
  })

  it('registerUser should POST /auth/register', async () => {
    const data = { tenant_id: 't1', username: 'bob', password: 'pw', nickname: 'Bob', email: 'b@b.com' }
    vi.mocked(post).mockResolvedValue({ id: 'u2', ...data, status: 1, created_at: '' })
    await registerUser(data)
    expect(post).toHaveBeenCalledWith('/auth/register', data)
  })

  it('getOAuthTenants should GET /auth/oauth/tenants', async () => {
    vi.mocked(get).mockResolvedValue({ tenants: [{ id: 't1', name: 'Acme' }] })
    const result = await getOAuthTenants()
    expect(get).toHaveBeenCalledWith('/auth/oauth/tenants')
    expect(result.tenants).toHaveLength(1)
  })

  it('oauthLogin should POST /auth/oauth/callback', async () => {
    vi.mocked(post).mockResolvedValue({ token: 't', user: { id: 'u1' }, permissions: [], is_new_user: true })
    await oauthLogin('google', 'code123', 't1', 'state1')
    expect(post).toHaveBeenCalledWith('/auth/oauth/callback', { provider: 'google', code: 'code123', tenant_id: 't1', state: 'state1' })
  })

  it('bindOAuth should POST /auth/oauth/bind', async () => {
    vi.mocked(post).mockResolvedValue({})
    await bindOAuth('github', 'code', 'state')
    expect(post).toHaveBeenCalledWith('/auth/oauth/bind', { provider: 'github', code: 'code', state: 'state' })
  })

  it('refreshOAuthToken should POST /auth/oauth/token/refresh', async () => {
    vi.mocked(post).mockResolvedValue({ token: 't2', refresh_token: 'r2' })
    const result = await refreshOAuthToken('r1')
    expect(post).toHaveBeenCalledWith('/auth/oauth/token/refresh', { refresh_token: 'r1' })
    expect(result.token).toBe('t2')
  })

  it('sendEmailVerification should POST /auth/email/send-verification', async () => {
    vi.mocked(post).mockResolvedValue({ message: 'sent' })
    await sendEmailVerification('a@b.com')
    expect(post).toHaveBeenCalledWith('/auth/email/send-verification', { email: 'a@b.com' })
  })

  it('verifyEmail should POST /auth/email/verify', async () => {
    vi.mocked(post).mockResolvedValue({ message: 'verified' })
    await verifyEmail('tok')
    expect(post).toHaveBeenCalledWith('/auth/email/verify', { token: 'tok' })
  })

  it('setup2FA should POST /auth/2fa/setup', async () => {
    vi.mocked(post).mockResolvedValue({ secret: 'sec', otpauth_url: 'otpauth://x' })
    const result = await setup2FA()
    expect(post).toHaveBeenCalledWith('/auth/2fa/setup', {})
    expect(result.secret).toBe('sec')
  })

  it('enable2FA should POST /auth/2fa/enable with code', async () => {
    vi.mocked(post).mockResolvedValue({ message: 'enabled', recovery_codes: ['AAAAA-BBBBB'] })
    const result = await enable2FA('123456')
    expect(post).toHaveBeenCalledWith('/auth/2fa/enable', { code: '123456' })
    expect(result.recovery_codes).toEqual(['AAAAA-BBBBB'])
  })

  it('disable2FA should POST /auth/2fa/disable with code', async () => {
    vi.mocked(post).mockResolvedValue({ message: 'disabled' })
    await disable2FA('123456')
    expect(post).toHaveBeenCalledWith('/auth/2fa/disable', { code: '123456' })
  })

  it('getRecoveryCodeStatus should GET /auth/2fa/recovery-codes', async () => {
    vi.mocked(get).mockResolvedValue({ enabled: true, remaining: 6 })
    const result = await getRecoveryCodeStatus()
    expect(get).toHaveBeenCalledWith('/auth/2fa/recovery-codes')
    expect(result.remaining).toBe(6)
  })

  it('regenerateRecoveryCodes should POST /auth/2fa/recovery-codes/regenerate with code', async () => {
    vi.mocked(post).mockResolvedValue({ recovery_codes: ['CCCCC-DDDDD'] })
    const result = await regenerateRecoveryCodes('123456')
    expect(post).toHaveBeenCalledWith('/auth/2fa/recovery-codes/regenerate', { code: '123456' })
    expect(result.recovery_codes).toEqual(['CCCCC-DDDDD'])
  })
})
