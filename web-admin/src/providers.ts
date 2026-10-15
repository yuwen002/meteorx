// 各 OAuth 提供商的展示元信息（品牌色），key 为后端 provider 标识。
// 展示用 label 由后端返回，这里仅补充前端无法从后端获取的品牌色。
export const providerBrand: Record<string, { color: string }> = {
  google: { color: '#ea4335' },
  github: { color: '#24292f' },
  wechat: { color: '#07c160' },
  feishu: { color: '#3370ff' },
  dingtalk: { color: '#1677ff' },
  qq: { color: '#12b7f5' },
  microsoft: { color: '#5e5e5e' }
}

// providerColor 返回 provider 的品牌色，未知（如自定义 OIDC）使用默认中性色。
export function providerColor(id: string): string {
  return providerBrand[id]?.color || '#6366f1'
}
