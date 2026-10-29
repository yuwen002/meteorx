import { get, put } from '@/api/request'

// 平台运行时配置项（与后端 dto.SettingView 对齐）
export interface SettingItem {
  key: string
  value: string
  value_type: string
  default_value: string
  is_public: boolean
  source: 'db' | 'default'
  description: string
  updated_at: string
}

// 批量更新单项（值统一为字符串，类型校验由后端按 value_type 归一化完成）
export interface SettingUpdateItem {
  key: string
  value: string
}

// 公开配置（登录/注册页消费，值已按类型还原为 JSON 原生形态）
export interface PublicSettings {
  registration_enabled?: boolean
  maintenance_mode?: boolean
  maintenance_message?: string
  [key: string]: unknown
}

// 管理视角列出全部配置（含默认值与来源）
export function listSystemSettings() {
  return get<SettingItem[]>('/admin/settings')
}

// 批量更新配置：后端整批校验后落库并失效缓存，返回最新全量列表
export function updateSystemSettings(settings: SettingUpdateItem[]) {
  return put<SettingItem[]>('/admin/settings', { settings })
}

// 公开配置（免鉴权）
export function getPublicSettings() {
  return get<PublicSettings>('/settings/public')
}
