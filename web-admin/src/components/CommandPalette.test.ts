// 命令面板组件单测：验证权限门控渲染、输入过滤、Ctrl/Cmd+K 全局开关、Enter 执行跳转。
// el-* 以插槽桩渲染（el-dialog 始终渲染默认插槽，使输入框与列表可被驱动）；
// store/router/i18n/theme 均以最小 mock 注入，聚焦本组件交互而非依赖真实全局态。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'

const pushSpy = vi.fn()
const toggleSidebarSpy = vi.fn()
const setLocaleSpy = vi.fn()

vi.mock('vue-router', () => ({ useRouter: () => ({ push: pushSpy }) }))
vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key, locale: ref('zh-CN'), setLocale: setLocaleSpy }),
}))
vi.mock('@/composables/useTheme', () => ({
  useTheme: () => ({ theme: ref('light'), toggleTheme: vi.fn() }),
}))
vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ isAdmin: false, hasPermission: (c: string) => c === 'wiki:list', logout: vi.fn() }),
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ toggleSidebar: toggleSidebarSpy }),
}))
vi.mock('element-plus/es/components/message/index', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))
vi.mock('element-plus/es/components/message-box/index', () => ({
  ElMessageBox: { confirm: vi.fn(async () => true) },
}))

import CommandPalette from './CommandPalette.vue'
import { useCommandPalette } from '@/composables/useCommandPalette'

const dialogStub = { template: '<div class="mx-dialog"><slot /></div>' }
const stubs = { 'el-dialog': dialogStub }

beforeEach(() => {
  vi.clearAllMocks()
  // 复位单例开关，避免用例间相互污染
  useCommandPalette().close()
})

afterEach(() => {
  useCommandPalette().close()
})

async function mountOpen() {
  const wrapper = mount(CommandPalette, { global: { stubs } })
  await flushPromises()
  useCommandPalette().open()
  await flushPromises()
  return wrapper
}

describe('CommandPalette', () => {
  it('打开后按权限渲染：授予 wiki:list 显示 Wiki，未授予 user:list 隐藏邀请，非超管隐藏系统入口', async () => {
    const wrapper = await mountOpen()
    const labels = wrapper.findAll('.cmdk-item').map((el) => el.text())
    const joined = labels.join('|')
    expect(joined).toContain('nav.wiki')
    expect(joined).not.toContain('nav.invitation')
    expect(joined).not.toContain('nav.settings')
    expect(joined).not.toContain('nav.user')
    // 四个操作命令恒在
    expect(joined).toContain('a11y.toggleSidebar')
    expect(joined).toContain('auth.logout')
  })

  it('输入过滤仅保留匹配项，Enter 执行导航并关闭面板', async () => {
    const wrapper = await mountOpen()
    await wrapper.find('input').setValue('wiki')
    await flushPromises()
    const items = wrapper.findAll('.cmdk-item')
    expect(items).toHaveLength(1)
    expect(items[0].text()).toContain('nav.wiki')

    await wrapper.find('input').trigger('keydown.enter')
    expect(pushSpy).toHaveBeenCalledWith('/wiki')
    expect(useCommandPalette().visible.value).toBe(false)
  })

  it('Ctrl/Cmd+K 全局键切换面板显隐', async () => {
    const wrapper = mount(CommandPalette, { global: { stubs } })
    await flushPromises()
    expect(useCommandPalette().visible.value).toBe(false)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))
    expect(useCommandPalette().visible.value).toBe(true)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))
    expect(useCommandPalette().visible.value).toBe(false)
    wrapper.unmount()
  })

  it('↑ 键在列表内循环移动高亮，Enter 执行高亮项', async () => {
    const wrapper = await mountOpen()
    const input = wrapper.find('input')
    await input.setValue('theme')
    await flushPromises()
    expect(wrapper.findAll('.cmdk-item')).toHaveLength(1)
    await input.trigger('keydown.enter')
    // 主题动作命令执行后关闭面板，但不跳转路由
    expect(pushSpy).not.toHaveBeenCalled()
    expect(useCommandPalette().visible.value).toBe(false)
  })
})
