// 租户开户页注册开关守卫测试
//
// 目标：断言公开配置 registration_enabled=false 时，直达 URL 也会提示并跳回登录页；
// 开关打开或配置接口失败时不拦截（提交仍由后端 403 兜底）。
// vitest 未启用 unplugin 自动导入，el-* 组件以桩件渲染。
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

const push = vi.fn()
const replace = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push, replace }),
}))

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
  ElMessageBox: { confirm: vi.fn() },
}))

vi.mock('@/api/auth', () => ({
  register: vi.fn(),
}))

vi.mock('@/api/modules/settings', () => ({
  getPublicSettings: vi.fn(),
}))

import RegisterView from './index.vue'
import { getPublicSettings } from '@/api/modules/settings'
import { ElMessage } from 'element-plus'

const passThrough = { template: '<div><slot name="header" /><slot /></div>' }
const empty = { template: '<div />' }
const stubs = {
  'el-form': passThrough,
  'el-form-item': passThrough,
  'el-input': empty,
  'el-divider': passThrough,
  'el-button': passThrough,
  'el-link': passThrough,
}

async function mountRegister() {
  const wrapper = mount(RegisterView, { global: { stubs, directives: { loading: {} } } })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('租户开户页注册开关守卫', () => {
  it('注册关闭时提示并跳回登录页', async () => {
    vi.mocked(getPublicSettings).mockResolvedValue({ registration_enabled: false } as never)
    await mountRegister()
    expect(ElMessage.warning).toHaveBeenCalledWith('registration.closed')
    expect(replace).toHaveBeenCalledWith('/login')
  })

  it('注册开启时停留在开户页', async () => {
    vi.mocked(getPublicSettings).mockResolvedValue({ registration_enabled: true } as never)
    const wrapper = await mountRegister()
    expect(replace).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('register.tenantTitle')
  })

  it('配置接口失败时不拦截（后端 403 兜底）', async () => {
    vi.mocked(getPublicSettings).mockRejectedValue(new Error('network down'))
    await mountRegister()
    expect(replace).not.toHaveBeenCalled()
    expect(ElMessage.warning).not.toHaveBeenCalled()
  })
})
