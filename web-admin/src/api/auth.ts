import { post } from '@/api/request'

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