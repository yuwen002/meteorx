import { get, post, put, del } from '@/api/request'

// 第三方登录提供商类型（与后端 model.ProviderKindXxx 对齐）
export type OAuthProviderKind =
  | 'google'
  | 'github'
  | 'wechat'
  | 'feishu'
  | 'dingtalk'
  | 'qq'
  | 'microsoft'
  | 'oidc'

export interface OAuthProviderItem {
  code: string
  kind: OAuthProviderKind
  display_name: string
  enabled: boolean
  client_id: string
  client_secret_mask: string
  has_secret: boolean
  redirect_url: string
  tenant?: string
  auth_url?: string
  token_url?: string
  userinfo_url?: string
  scopes?: string[]
  editable_endpoint: boolean
  created_at: string
  updated_at: string
}

export interface OAuthProviderSaveParams {
  kind: OAuthProviderKind
  code: string
  display_name?: string
  enabled: boolean
  client_id: string
  client_secret?: string
  redirect_url: string
  tenant?: string
  auth_url?: string
  token_url?: string
  userinfo_url?: string
  scopes?: string[]
}

// 类型选项（供新建/编辑对话框下拉）
export const providerKindOptions: Array<{ value: OAuthProviderKind; label: string; fixed: boolean }> = [
  { value: 'google', label: 'Google', fixed: true },
  { value: 'github', label: 'GitHub', fixed: true },
  { value: 'wechat', label: '微信', fixed: true },
  { value: 'feishu', label: '飞书', fixed: true },
  { value: 'dingtalk', label: '钉钉', fixed: true },
  { value: 'qq', label: 'QQ', fixed: true },
  { value: 'microsoft', label: 'Microsoft Entra ID', fixed: true },
  { value: 'oidc', label: '通用 OIDC（企业自建）', fixed: false }
]

// 查询全部第三方登录配置
export function listOAuthProviderConfigs() {
  return get<{ providers: OAuthProviderItem[] }>('/admin/oauth-providers')
}

// 查询单个第三方登录配置
export function getOAuthProviderConfig(code: string) {
  return get<OAuthProviderItem>(`/admin/oauth-providers/${code}`)
}

// 新建第三方登录配置
export function createOAuthProviderConfig(data: OAuthProviderSaveParams) {
  return post<void>('/admin/oauth-providers', data)
}

// 更新第三方登录配置
export function updateOAuthProviderConfig(code: string, data: OAuthProviderSaveParams) {
  return put<void>(`/admin/oauth-providers/${code}`, data)
}

// 删除第三方登录配置
export function deleteOAuthProviderConfig(code: string) {
  return del<void>(`/admin/oauth-providers/${code}`)
}

// 切换启用状态
export function setOAuthProviderStatus(code: string, enabled: boolean) {
  return put<void>(`/admin/oauth-providers/${code}/status`, { enabled })
}
