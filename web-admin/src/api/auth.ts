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

// 登录
export function login(params: LoginParams) {
  return post<LoginResult>('/auth/login', params)
}

// 登出（如果后端需要调用接口）
export function logout() {
  return post('/auth/logout')
}