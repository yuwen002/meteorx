import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listOAuthAccounts,
  unbindOAuth,
  createAPIToken,
  listAPITokens,
  revokeAPIToken,
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
