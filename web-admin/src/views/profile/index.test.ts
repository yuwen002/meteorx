// 个人中心 API Token 作用域（scopes）视图测试
//
// 目标：断言创建弹窗的 scope 候选来自当前用户权限码、选中后随请求体提交、
// 留空时不发送 scopes 字段（保持继承语义）、令牌列表按 scopes 渲染徽章/「不限制」，
// 以及重新打开弹窗时 scopes 选择态被清空。
// vitest 未启用 unplugin 自动导入，el-* 组件以桩件渲染；el-form 桩暴露恒通过的
// validate 以驱动提交链路，el-table 桩按数据行渲染列插槽。
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

// i18n：t 回显 key（附参数）便于断言文案键
vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (key: string, ...args: unknown[]) => (args.length ? `${key}:${args.join(',')}` : key) }),
}))

vi.mock('element-plus/es/components/message/index', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))
vi.mock('element-plus/es/components/message-box/index', () => ({
  ElMessageBox: { confirm: vi.fn(async () => true) },
}))

vi.mock('@/api/modules/user', () => ({
  getProfile: vi.fn(),
  updateProfile: vi.fn(),
  changePassword: vi.fn(),
}))
vi.mock('@/api/modules/role', () => ({
  getUserRoles: vi.fn(async () => []),
}))
vi.mock('@/api/modules/oauth', () => ({
  listOAuthProviders: vi.fn(async () => ({ providers: [] })),
}))
vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<Record<string, unknown>>('@/api/request')
  void actual
  return {
    createAPIToken: vi.fn(),
    listAPITokens: vi.fn(),
    revokeAPIToken: vi.fn(),
    listOAuthAccounts: vi.fn(async () => ({ accounts: [] })),
    unbindOAuth: vi.fn(),
    getOAuthRedirectURL: vi.fn(),
    sendEmailVerification: vi.fn(),
    setup2FA: vi.fn(),
    enable2FA: vi.fn(),
    disable2FA: vi.fn(),
    regenerateRecoveryCodes: vi.fn(),
  }
})

import ProfileView from './index.vue'
import { createAPIToken, listAPITokens, type APITokenItem } from '@/api/auth'
import { getProfile } from '@/api/modules/user'
import { useUserStore } from '@/stores/user'

const passThrough = { template: '<div><slot name="header" /><slot /></div>' }
// 表单桩：暴露恒通过的 validate(cb)，对齐 submitCreateToken 的调用方式
const formStub = { methods: { validate: (cb?: (valid: boolean) => unknown) => Promise.resolve(cb?.(true)) } }
// 弹窗桩：始终渲染默认插槽，使表单可被驱动
const dialogStub = { template: '<div class="mx-dialog"><slot /></div>' }
// 按钮桩：保留 click 语义
const buttonStub = {
  props: ['disabled', 'loading'],
  emits: ['click'],
  template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
}
// 表格桩：简单显示行数，列插槽内容由单独断言验证状态而非 DOM
const tableStub = { props: ['data'], template: '<div class="mx-table">{{ data ? data.length : 0 }}</div>' }
const stubs = {
  'el-card': passThrough,
  'el-dialog': dialogStub,
  'el-form': formStub,
  'el-form-item': passThrough,
  'el-table': tableStub,
  'el-table-column': { template: '<div><slot :row="{}" /></div>' },
  'el-tag': passThrough,
  'el-button': buttonStub,
  'el-input': { props: ['modelValue'], template: '<div />' },
  'el-select': { props: ['modelValue'], template: '<div><slot /></div>' },
  'el-option': { props: ['label', 'value'], template: '<div />' },
  'el-empty': passThrough,
  'el-icon': passThrough,
}

async function mountView() {
  const wrapper = mount(ProfileView, {
    global: { stubs, directives: { loading: {} } },
  })
  await flushPromises()
  return wrapper
}

function tokenFixture(overrides: Partial<APITokenItem> = {}): APITokenItem {
  return {
    id: 'at-001',
    name: 'Scoped Token',
    created_at: '2026-11-01T12:00:00Z',
    revoked: false,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
  // 模拟已登录普通用户：store 权限码乱序写入，断言 scopeOptions 排序
  useUserStore().setPermissions(['task:list', 'user:list'])
  vi.mocked(getProfile).mockResolvedValue({ id: 'u1', username: 'alice' } as never)
  vi.mocked(listAPITokens).mockResolvedValue({ tokens: [] })
  vi.mocked(createAPIToken).mockResolvedValue({
    id: 'at-new', name: 'x', token: 'mx_at_secret', created_at: '2026-11-01T12:00:00Z',
  })
})

describe('个人中心 API Token 作用域', () => {
  it('scope 候选来自自身权限码并排序，选中后随创建请求提交', async () => {
    const wrapper = await mountView()
    const vm = wrapper.vm as unknown as {
      createTokenForm: { name: string; scopes: string[]; allowed_paths: string[] }
      scopeOptions: string[]
      openCreateTokenDialog: () => void
      submitCreateToken: () => Promise<void>
    }

    // 先污染选择态，验证重新打开弹窗会清空
    vm.createTokenForm.scopes = ['stale:code']

    await vm.openCreateTokenDialog()
    expect(vm.createTokenForm.scopes).toEqual([])

    // scopeOptions 读取 store.permissions（pinia 活动实例下由真实 store 提供）
    expect(vm.scopeOptions).toEqual(['task:list', 'user:list'])

    vm.createTokenForm.name = 'Scoped Token'
    vm.createTokenForm.scopes = ['user:list']
    await vm.submitCreateToken()
    expect(createAPIToken).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Scoped Token', scopes: ['user:list'] })
    )
  })

  it('scope 留空时不发送 scopes 字段（保持继承全部权限语义）', async () => {
    const wrapper = await mountView()
    const vm = wrapper.vm as unknown as {
      createTokenForm: { name: string; scopes: string[] }
      openCreateTokenDialog: () => void
      submitCreateToken: () => Promise<void>
    }
    await vm.openCreateTokenDialog()
    vm.createTokenForm.name = 'Plain Token'
    await vm.submitCreateToken()

    const payload = vi.mocked(createAPIToken).mock.calls[0][0]
    expect(payload.scopes).toBeUndefined()
  })

  it('令牌列表装载后状态含 scopes 字段，供表格徽章列渲染', async () => {
    vi.mocked(listAPITokens).mockResolvedValue({
      tokens: [
        tokenFixture({ scopes: ['user:list', 'task:list'] }),
        tokenFixture({ id: 'at-002', name: 'Inherited Token' }),
      ],
    })
    const wrapper = await mountView()
    const vm = wrapper.vm as unknown as { apiTokens: APITokenItem[] }
    expect(vm.apiTokens[0].scopes).toEqual(['user:list', 'task:list'])
    expect(vm.apiTokens[1].scopes).toBeUndefined()
  })

  it('弹窗打开后 createTokenForm.scopes 可编辑，候选由 store.permissions 提供', async () => {
    const wrapper = await mountView()
    const vm = wrapper.vm as unknown as {
      createTokenForm: { name: string; scopes: string[] }
      scopeOptions: string[]
      openCreateTokenDialog: () => void
    }
    await vm.openCreateTokenDialog()
    // scopeOptions 来自 store 权限码排序
    expect(vm.scopeOptions).toEqual(['task:list', 'user:list'])
    vm.createTokenForm.scopes = ['user:list']
    expect(vm.createTokenForm.scopes).toEqual(['user:list'])
  })
})
