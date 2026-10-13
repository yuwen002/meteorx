import { storeToRefs } from 'pinia'
import {
  useBrandStore,
  DEFAULT_BRAND,
  type BrandConfig,
  type ServerBrandInput,
} from '@/stores/brand'

export type { BrandConfig, ServerBrandInput }
export { DEFAULT_BRAND }

// 兼容封装：保留 useBrand() 原有 API，底层品牌状态已迁移至 Pinia brand store。
// 服务端白标下发（applyServerBrand/syncBrandFromServer）一并由 store 承载。
export function useBrand() {
  const store = useBrandStore()
  const { brand } = storeToRefs(store)
  return {
    brand,
    setBrand: store.setBrand,
    resetBrand: store.resetBrand,
    applyBrand: store.applyBrand,
    applyServerBrand: store.applyServerBrand,
    syncBrandFromServer: store.syncBrandFromServer,
    DEFAULT_BRAND,
  }
}
