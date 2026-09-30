import { get, post } from '@/api/request'

export interface OAuthRedirectResponse {
  url: string
  state: string
}

export interface OAuthLoginResponse {
  token: string
  refresh_token?: string
  user: any
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
  return post<any>('/auth/oauth/unbind', { provider })
}

// 绑定新的第三方账号
export function bindOAuth(data: { provider: string; code: string; state: string }) {
  return post<any>('/auth/oauth/bind', data)
}