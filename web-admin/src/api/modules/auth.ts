import { get, post } from '@/api/request'

export interface LoginParams {
  tenant_id?: string
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: any
  permissions: string[]
}

export interface RegisterParams {
  tenant_id: string
  username: string
  password: string
  nickname: string
  email: string
}

export interface LoginErrorResponse {
  message: string
  remaining_attempts: number
  locked: boolean
  lockout_duration: number
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

export interface CreateAPITokenResponse {
  id: string
  name: string
  token: string
  allowed_paths?: string[]
  expires_at?: string
  created_at: string
}

// 用户登录
export function login(data: LoginParams) {
  return post<LoginResponse>('/auth/login', data)
}

// 用户注册
export function register(data: RegisterParams) {
  return post<any>('/auth/register', data)
}

// 用户登出
export function logout() {
  return post<any>('/auth/logout')
}

// 忘记密码（发送重置链接到邮箱）
export function forgotPassword(email: string) {
  return post<{ message: string }>('/auth/forgot-password', { email })
}

// 重置密码（通过令牌设置新密码）
export function resetPassword(token: string, new_password: string) {
  return post<{ message: string }>('/auth/reset-password', { token, new_password })
}

// 发送邮箱验证链接
export function sendEmailVerification(email: string) {
  return post<{ message: string }>('/auth/email/send-verification', { email })
}

// 验证邮箱（通过令牌）
export function verifyEmail(token: string) {
  return post<{ message: string }>('/auth/email/verify', { token })
}

// 获取 API Token 列表
export function listAPITokens() {
  return get<APITokenItem[]>('/auth/tokens')
}

// 创建 API Token
export function createAPIToken(data: {
  name: string
  expires_in?: string
  allowed_paths?: string[]
}) {
  return post<CreateAPITokenResponse>('/auth/tokens', data)
}

// 撤销 API Token
export function revokeAPIToken(id: string) {
  return post<any>('/auth/tokens/revoke', { id })
}