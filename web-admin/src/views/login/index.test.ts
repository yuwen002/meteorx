// 登录页公开配置消费测试
//
// 目标：断言维护模式公告按 maintenance_mode 渲染、公告描述取 maintenance_message（空则回落默认键）、
// registration_enabled=false 时隐藏注册入口、接口失败时 fail-open 保持默认展示（与后端口径一致）。
// vitest 未启用 unplugin 自动导入，el-* 组件与图标均以桩件渲染。
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (key: string, ...args: unknown[]) => (args.length ? `${key}:${args.join(',')}` : key) }),
}))

const push = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push }),
}))

vi.mock('element-plus/es/components/message/index', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))

// 图标组件桩：避免真实 SVG 导入
vi.mock('@element-plus/icons-vue', () => {
  const stub = (name: string) => ({ name, render: () => null })
  return {
    InfoFilled: stub('InfoFilled'),
    Warning: stub('Warning'),
    Avatar: stub('Avatar'),
    OfficeBuilding: stub('OfficeBuilding'),
    ArrowDown: stub('ArrowDown'),
    Key: stub('Key'),
    User: stub('User'),
    Lock: stub('Lock'),
  }
})

// 登录页挂载期依赖的接口：OAuth 提供商与公开配置
vi.mock('@/api/modules/oauth', () => ({
  listOAuthProviders: vi.fn().mockResolvedValue({ providers: [] }),
}))

vi.mock('@/api/auth', () => ({
  getOAuthRedirectURL: vi.fn(),
  getOAuthTenants: vi.fn(),
}))

vi.mock('@/api/modules/settings', () => ({
  getPublicSettings: vi.fn(),
}))

import LoginView from './index.vue'
import { getPublicSettings } from '@/api/modules/settings'

const passThrough = { template: '<div><slot name="header" /><slot /></div>' }
const empty = { template: '<div />' }
const stubs = {
  'el-alert': { name: 'ElAlertStub', props: ['type', 'title', 'description', 'closable', 'showIcon'], template: '<div class="mx-alert">{{ title }}|{{ description }}</div>' },
  'el-form': passThrough,
  'el-form-item': passThrough,
  'el-input': empty,
  'el-button': passThrough,
  'el-icon': passThrough,
  'el-link': passThrough,
  'el-dropdown': passThrough,
  'el-dropdown-menu': passThrough,
  'el-dropdown-item': passThrough,
  'el-dialog': empty,
  'el-select': empty,
  'el-option': empty,
}

async function mountLogin() {
  const wrapper = mount(LoginView, { global: { stubs, directives: { loading: {} } } })
  await flushPromises()
  return wrapper
}

function publicSettings(overrides: Record<string, unknown> = {}) {
  return {
    registration_enabled: true,
    maintenance_mode: false,
    maintenance_message: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
  vi.mocked(getPublicSettings).mockResolvedValue(publicSettings() as never)
})

describe('登录页消费公开配置', () => {
  it('默认状态：不渲染维护公告且注册入口可见', async () => {
    const wrapper = await mountLogin()
    expect(wrapper.find('.mx-alert').exists()).toBe(false)
    expect(wrapper.text()).toContain('loginPage.registerNow')
  })

  it('maintenance_mode=true 时渲染公告并携带自定义文案', async () => {
    vi.mocked(getPublicSettings).mockResolvedValue(
      publicSettings({ maintenance_mode: true, maintenance_message: '今晚 22:00 升级' }) as never,
    )
    const alert = (await mountLogin()).find('.mx-alert')
    expect(alert.exists()).toBe(true)
    expect(alert.text()).toBe('loginPage.maintenanceTitle|今晚 22:00 升级')
  })

  it('公告文案为空时回落默认文案键', async () => {
    vi.mocked(getPublicSettings).mockResolvedValue(
      publicSettings({ maintenance_mode: true, maintenance_message: '' }) as never,
    )
    const alert = (await mountLogin()).find('.mx-alert')
    expect(alert.text()).toBe('loginPage.maintenanceTitle|loginPage.maintenanceDefault')
  })

  it('registration_enabled=false 时隐藏注册入口', async () => {
    vi.mocked(getPublicSettings).mockResolvedValue(
      publicSettings({ registration_enabled: false }) as never,
    )
    const wrapper = await mountLogin()
    expect(wrapper.text()).not.toContain('loginPage.registerNow')
  })

  it('公开配置接口失败时保持默认展示（不阻塞登录）', async () => {
    vi.mocked(getPublicSettings).mockRejectedValue(new Error('network down'))
    const wrapper = await mountLogin()
    expect(wrapper.find('.mx-alert').exists()).toBe(false)
    expect(wrapper.text()).toContain('loginPage.registerNow')
  })
})
