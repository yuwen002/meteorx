import { get, post } from '@/api/request'
import type { LoginUserInfo } from '@/api/auth'

export interface OAuthRedirectResponse {
  url: string
  state: string
}

export interface OAuthLoginResponse {
  token: string
  refresh_token?: string
  user: LoginUserInfo
  permissions: string[]
  is_new_user: boolean
}

export interface TenantOption {
  id: string
  name: string
}

export interface TenantListResponse {
  tenants: TenantOption[]
}

export interface OAuthAccountItem {
  id: string
  provider: string
  email: string
  created_at: string
}

export interface RefreshTokenResponse {
  token: string
  refresh_token: string
}

// ProviderMeta 已启用第三方登录提供商的展示元信息
export interface ProviderMeta {
  id: string
  label: string
}

// 获取已启用的第三方登录提供商列表
export function listOAuthProviders() {
  return get<{ providers: ProviderMeta[] }>('/auth/oauth/providers')
}

// 第三方登录后补全邮箱（需携带登录签发的临时令牌）
export function completeOAuthEmail(token: string, email: string) {
  return post<RefreshTokenResponse>(
    '/auth/oauth/complete-email',
    { email },
    { headers: { Authorization: `Bearer ${token}` } }
  )
}

// 获取第三方授权跳转地址
export function getOAuthRedirect(provider: string) {
  return get<OAuthRedirectResponse>(`/auth/oauth/${provider}/redirect`)
}

// 第三方授权回调，完成登录或注册
export function oauthCallback(data: {
  provider: string
  code: string
  state: string
  tenant_id: string
}) {
  return post<OAuthLoginResponse>('/auth/oauth/callback', data)
}

// 回调后查询该 OAuth 账号关联的租户列表
export function getOAuthTenants() {
  return get<TenantListResponse>('/auth/oauth/tenants')
}

// 刷新 OAuth 临时 Token
export function refreshOAuthToken(refresh_token: string) {
  return post<RefreshTokenResponse>('/auth/oauth/token/refresh', { refresh_token })
}

// 查看当前用户已绑定的第三方账号列表
export function listOAuthAccounts() {
  return get<{ accounts: OAuthAccountItem[] }>('/auth/oauth/accounts')
}

// 解绑指定第三方账号
export function unbindOAuth(provider: string) {
  return post<void>('/auth/oauth/unbind', { provider })
}

// 绑定新的第三方账号
export function bindOAuth(data: { provider: string; code: string; state: string }) {
  return post<void>('/auth/oauth/bind', data)
}