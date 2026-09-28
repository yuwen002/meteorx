import { post, get } from '@/api/request'

export interface LoginParams {
  username: string
  password: string
  tenant_id?: string
}

export interface LoginUserInfo {
  id: string
  tenant_id?: string
  username: string
  nickname?: string
  email?: string
  status?: number
  is_master?: boolean
  created_at?: string
}

export interface LoginResult {
  token: string
  user: LoginUserInfo
  permissions?: string[]
}

export interface LoginErrorData {
  message: string
  remaining_attempts: number
  locked: boolean
  lockout_duration: number
}

export interface ForgotPasswordReq {
  email: string
}

export interface ResetPasswordReq {
  token: string
  new_password: string
}

export interface OAuthRedirectResult {
  url: string
}

export interface OAuthLoginResult {
  token: string
  user: LoginUserInfo
  permissions: string[]
  is_new_user: boolean
}

// 登录
export function login(params: LoginParams) {
  return post<LoginResult>('/auth/login', params)
}

// 登出
export function logout() {
  return post('/auth/logout')
}

// 忘记密码 - 发送重置邮件
export function forgotPassword(email: string) {
  return post<{ message: string }>('/auth/forgot-password', { email })
}

// 重置密码
export function resetPassword(token: string, newPassword: string) {
  return post<{ message: string }>('/auth/reset-password', { token, new_password: newPassword })
}

// OAuth2 获取跳转链接
export function getOAuthRedirectURL(provider: string) {
  return get<OAuthRedirectResult>(`/auth/oauth/${provider}/redirect`)
}

export interface RegisterParams {
  name: string
  domain: string
  description?: string
  contact_email?: string
  admin_user: {
    username: string
    password: string
    nickname: string
    email: string
  }
}

export interface RegisterResult {
  id: string
  name: string
  domain: string
  admin_user: {
    id: string
    username: string
    nickname: string
    email: string
  }
}

// 用户注册（租户自助注册）
export function register(data: RegisterParams) {
  return post<RegisterResult>('/tenants/register', data)
}

export interface RegisterUserParams {
  tenant_id: string
  username: string
  password: string
  nickname: string
  email: string
}

export interface RegisterUserResult {
  id: string
  tenant_id: string
  username: string
  nickname: string
  email: string
  status: number
  created_at: string
}

// 普通用户注册（加入已有租户）
export function registerUser(data: RegisterUserParams) {
  return post<RegisterUserResult>('/auth/register', data)
}

// OAuth2 获取租户列表
export function getOAuthTenants() {
  return get<{ tenants: Array<{ id: string; name: string }> }>('/auth/oauth/tenants')
}

// OAuth2 登录回调
export function oauthLogin(provider: string, code: string, tenant_id: string, state?: string) {
  return post<OAuthLoginResult>('/auth/oauth/callback', { provider, code, tenant_id, state })
}

// ========== OAuth 账号管理 ==========

export interface OAuthAccountItem {
  id: string
  provider: string
  email: string
  created_at: string
}

// 查看已绑定的第三方账号列表
export function listOAuthAccounts() {
  return get<{ accounts: OAuthAccountItem[] }>('/auth/oauth/accounts')
}

// 绑定第三方账号
export function bindOAuth(provider: string, code: string, state: string) {
  return post('/auth/oauth/bind', { provider, code, state })
}

// 解绑第三方账号
export function unbindOAuth(provider: string) {
  return post('/auth/oauth/unbind', { provider })
}

// 刷新 OAuth Token
export function refreshOAuthToken(refresh_token: string) {
  return post<{ token: string; refresh_token: string }>('/auth/oauth/token/refresh', { refresh_token })
}

// ========== API Token 管理 ==========

export interface CreateAPITokenParams {
  name: string
  expires_in?: string
  allowed_paths?: string[]
}

export interface CreateAPITokenResult {
  id: string
  name: string
  token: string
  allowed_paths?: string[]
  expires_at?: string
  created_at: string
}

export interface APITokenItem {
  id: string
  name: string
  allowed_paths?: string[]
  last_used_at?: string
  expires_at?: string
  created_at: string
  revoked: boolean
}

export interface RevokeAPITokenParams {
  id: string
}

// 创建 API Token
export function createAPIToken(data: CreateAPITokenParams) {
  return post<CreateAPITokenResult>('/auth/tokens', data)
}

// 查询 API Token 列表
export function listAPITokens() {
  return get<{ tokens: APITokenItem[] }>('/auth/tokens')
}

// 撤销 API Token
export function revokeAPIToken(data: RevokeAPITokenParams) {
  return post('/auth/tokens/revoke', data)
}