import { storeToRefs } from 'pinia'
import { useThemeStore, type ThemeMode } from '@/stores/theme'

export type { ThemeMode }

// 兼容封装：保留 useTheme() 原有 API，底层主题状态已迁移至 Pinia theme store。
// 主题需通过 useThemeStore() 在应用安装 Pinia 后访问，首屏无闪烁由 main.ts 引导保证。
export function useTheme() {
  const store = useThemeStore()
  const { theme } = storeToRefs(store)
  return { theme, setTheme: store.setTheme, toggleTheme: store.toggleTheme, isDark: store.isDark }
}
