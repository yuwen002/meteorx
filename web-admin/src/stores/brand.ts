import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getTenantSettings } from '@/api/modules/tenant'

export interface BrandConfig {
  name: string // display name (browser title, sidebar logo text)
  logo: string // logo image URL, falls back to the first letter of the name when empty
  primaryColor: string // brand primary color (overrides Element Plus main color)
}

// 服务端下发的租户品牌配置（对齐 TenantSettingsResp 的相关字段）
export interface ServerBrandInput {
  tenant_name?: string
  logo?: string
  primary_color?: string
}

const BRAND_KEY = 'meteorx_brand'

export const DEFAULT_BRAND: BrandConfig = {
  name: 'MeteorX',
  logo: '',
  primaryColor: '#409eff'
}

// 纯 DOM 应用逻辑，不依赖 Pinia，供首屏引导与 store 变更复用。
export function applyBrandToDOM(b: BrandConfig) {
  document.title = b.name || DEFAULT_BRAND.name
  const root = document.documentElement
  if (b.primaryColor) {
    // 覆盖 Element Plus 主色及其状态色，令按钮/链接/高亮统一为品牌色
    root.style.setProperty('--el-color-primary', b.primaryColor)
    root.style.setProperty('--mx-brand-primary', b.primaryColor)
  } else {
    root.style.removeProperty('--el-color-primary')
    root.style.removeProperty('--mx-brand-primary')
  }
}

// 从 localStorage 读取持久化品牌，回退默认品牌（无 Pinia 依赖，供挂载前引导调用）。
export function readStoredBrand(): BrandConfig {
  try {
    const raw = localStorage.getItem(BRAND_KEY)
    if (raw) {
      return { ...DEFAULT_BRAND, ...(JSON.parse(raw) as Partial<BrandConfig>) }
    }
  } catch {
    // 忽略损坏的持久化数据，回退默认品牌
  }
  return { ...DEFAULT_BRAND }
}

export const useBrandStore = defineStore('brand', () => {
  // 全局唯一品牌源，跨组件共享。
  const brand = ref<BrandConfig>(readStoredBrand())

  function applyBrand() {
    applyBrandToDOM(brand.value)
  }

  function setBrand(patch: Partial<BrandConfig>) {
    brand.value = { ...brand.value, ...patch }
    localStorage.setItem(BRAND_KEY, JSON.stringify(brand.value))
    applyBrand()
  }

  function resetBrand() {
    brand.value = { ...DEFAULT_BRAND }
    localStorage.setItem(BRAND_KEY, JSON.stringify(brand.value))
    applyBrand()
  }

  // 将服务端下发的品牌配置应用到全局 UI（服务端优先，空字段回退默认值），
  // 并持久化到 localStorage 作为下次首屏/离线时的缓存。
  function applyServerBrand(settings: ServerBrandInput) {
    brand.value = {
      name: settings.tenant_name?.trim() || DEFAULT_BRAND.name,
      logo: settings.logo?.trim() || '',
      primaryColor: settings.primary_color?.trim() || DEFAULT_BRAND.primaryColor
    }
    localStorage.setItem(BRAND_KEY, JSON.stringify(brand.value))
    applyBrand()
  }

  // 登录后从服务端拉取品牌配置并应用；失败（如平台超管无租户上下文、网络异常）
  // 则保留本地持久化/默认品牌作为回退，不阻断主流程。
  async function syncBrandFromServer() {
    try {
      const res = await getTenantSettings()
      applyServerBrand(res)
    } catch {
      // 回退：维持当前（本地缓存或默认）品牌
    }
  }

  return { brand, applyBrand, setBrand, resetBrand, applyServerBrand, syncBrandFromServer }
})
