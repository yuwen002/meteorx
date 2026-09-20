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

// OAuth2 获取租户列表
export function getOAuthTenants() {
  return get<{ tenants: Array<{ id: string; name: string }> }>('/auth/oauth/tenants')
}

// OAuth2 登录回调
export function oauthLogin(provider: string, code: string, tenant_id: string, state?: string) {
  return post<OAuthLoginResult>('/auth/oauth/callback', { provider, code, tenant_id, state })
}