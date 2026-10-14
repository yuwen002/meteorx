import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useI18n } from './useI18n'

describe('useI18n', () => {
  beforeEach(() => {
    // useI18n 已迁移至 Pinia store，测试需先激活一个干净的 pinia 实例
    setActivePinia(createPinia())
    const { setLocale } = useI18n()
    setLocale('zh-CN')
  })

  it('should return Chinese text by default', () => {
    const { t } = useI18n()
    expect(t('common.confirm')).toBe('确定')
    expect(t('common.cancel')).toBe('取消')
    expect(t('nav.dashboard')).toBe('首页')
  })

  it('should switch to English', () => {
    const { t, setLocale } = useI18n()
    setLocale('en-US')
    expect(t('common.confirm')).toBe('Confirm')
    expect(t('common.cancel')).toBe('Cancel')
    expect(t('nav.dashboard')).toBe('Dashboard')
  })

  it('should return key when translation missing', () => {
    const { t } = useI18n()
    expect(t('nonexistent.key')).toBe('nonexistent.key')
  })

  it('should support interpolation', () => {
    const { t } = useI18n()
    expect(t('common.total', 42)).toBe('共 42 条')
    expect(t('profile.oauth.unbindConfirm', 'google')).toBe('确定要解绑 google 账号吗？')
  })

  it('should support English interpolation', () => {
    const { t, setLocale } = useI18n()
    setLocale('en-US')
    expect(t('common.total', 42)).toBe('Total 42')
  })

  it('should list available locales', () => {
    const { availableLocales } = useI18n()
    expect(availableLocales).toContain('zh-CN')
    expect(availableLocales).toContain('en-US')
  })

  it('should report current locale', () => {
    const { locale, setLocale } = useI18n()
    expect(locale.value).toBe('zh-CN')
    setLocale('en-US')
    expect(locale.value).toBe('en-US')
  })

  it('should not change locale for invalid code', () => {
    const { locale, setLocale } = useI18n()
    setLocale('fr-FR')
    expect(locale.value).toBe('zh-CN')
  })

  it('should translate risk levels', () => {
    const { t } = useI18n()
    expect(t('risk.low')).toBe('低')
    expect(t('risk.medium')).toBe('中')
    expect(t('risk.high')).toBe('高')
    expect(t('risk.critical')).toBe('严重')
  })

  it('should translate notification channels', () => {
    const { t } = useI18n()
    expect(t('channel.email')).toBe('邮件')
    expect(t('channel.dingtalk')).toBe('钉钉')
    expect(t('channel.wechat')).toBe('企业微信')
    expect(t('channel.webhook')).toBe('Webhook')
  })

  it('should translate a11y labels', () => {
    const { t } = useI18n()
    expect(t('a11y.skipToContent')).toBe('跳到主内容')
    expect(t('a11y.mainNav')).toBe('主导航')
    expect(t('a11y.header')).toBe('顶栏')
    expect(t('a11y.toggleSidebar')).toBe('折叠或展开侧边栏')
    expect(t('a11y.switchLanguage')).toBe('切换语言')
    expect(t('a11y.userMenu')).toBe('用户菜单')
  })

  it('should translate a11y labels in English', () => {
    const { t, setLocale } = useI18n()
    setLocale('en-US')
    expect(t('a11y.skipToContent')).toBe('Skip to main content')
    expect(t('a11y.mainNav')).toBe('Main navigation')
    expect(t('a11y.header')).toBe('Top bar')
    expect(t('a11y.toggleSidebar')).toBe('Collapse or expand sidebar')
    expect(t('a11y.switchLanguage')).toBe('Switch language')
    expect(t('a11y.userMenu')).toBe('User menu')
  })
})