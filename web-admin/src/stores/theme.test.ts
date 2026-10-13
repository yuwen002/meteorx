import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useThemeStore } from './theme'

describe('theme store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.removeItem('meteorx_theme')
    document.documentElement.classList.remove('dark')
  })

  it('defaults to light when no persisted value', () => {
    const s = useThemeStore()
    expect(s.theme).toBe('light')
  })

  it('setTheme persists to localStorage and applies DOM class', () => {
    const s = useThemeStore()
    s.setTheme('dark')
    expect(s.theme).toBe('dark')
    expect(localStorage.getItem('meteorx_theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('toggleTheme flips between modes', () => {
    const s = useThemeStore()
    s.toggleTheme()
    expect(s.theme).toBe('dark')
    s.toggleTheme()
    expect(s.theme).toBe('light')
  })
})
