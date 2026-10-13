import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ThemeMode = 'light' | 'dark'

const THEME_KEY = 'meteorx_theme'

// 纯 DOM 应用逻辑，不依赖 Pinia，供首屏引导与 store 变更复用。
export function applyThemeToDOM(mode: ThemeMode) {
  document.documentElement.classList.toggle('dark', mode === 'dark')
  document.documentElement.setAttribute('data-theme', mode)
}

// 从 localStorage 读取持久化主题，回退浅色（无 Pinia 依赖，供挂载前引导调用）。
export function readStoredTheme(): ThemeMode {
  return (localStorage.getItem(THEME_KEY) as ThemeMode) || 'light'
}

export const useThemeStore = defineStore('theme', () => {
  // 全局唯一主题源，跨组件共享。
  const theme = ref<ThemeMode>(readStoredTheme())

  function setTheme(mode: ThemeMode) {
    theme.value = mode
    localStorage.setItem(THEME_KEY, mode)
    applyThemeToDOM(mode)
  }

  function toggleTheme() {
    setTheme(theme.value === 'dark' ? 'light' : 'dark')
  }

  function isDark() {
    return theme.value === 'dark'
  }

  return { theme, setTheme, toggleTheme, isDark }
})
