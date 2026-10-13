import { storeToRefs } from 'pinia'
import { useI18nStore } from '@/stores/i18n'

// 兼容封装：保留 useI18n() 原有 API（t/setLocale/locale/availableLocales），
// 底层语言状态与消息表已迁移至 Pinia i18n store。
export function useI18n() {
  const store = useI18nStore()
  const { locale } = storeToRefs(store)
  return {
    t: store.t,
    setLocale: store.setLocale,
    locale,
    availableLocales: store.availableLocales,
  }
}
