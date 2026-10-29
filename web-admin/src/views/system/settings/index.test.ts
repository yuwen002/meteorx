// 平台运行时配置管理页视图测试
//
// 目标：断言装载后表格获得全量配置、无改动时保存禁用、草稿变更后仅提交差量、
// 保存成功以返回值重建基线、失败保留草稿、放弃修改可还原。
// vitest 未启用 unplugin 自动导入，el-* 组件以桩件渲染。
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// i18n：t 回显 key（附参数）便于断言文案键
vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (key: string, ...args: unknown[]) => (args.length ? `${key}:${args.join(',')}` : key) }),
}))

vi.mock('element-plus/es/components/message/index', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))

vi.mock('@/api/modules/settings', () => ({
  listSystemSettings: vi.fn(),
  updateSystemSettings: vi.fn(),
}))

import SettingsView from './index.vue'
import { listSystemSettings, updateSystemSettings } from '@/api/modules/settings'
import { ElMessage } from 'element-plus/es/components/message/index'
import type { SettingItem } from '@/api/modules/settings'

const passThrough = { template: '<div><slot name="header" /><slot /></div>' }
// 表格桩：只计数渲染行，行内编辑器不在本测试关注面
const tableStub = { props: ['data'], template: '<div class="mx-table">{{ data ? data.length : 0 }}</div>' }
// 按钮桩：保留 disabled 与 click 语义，使差量提交链路可被驱动
const buttonStub = {
  props: ['disabled', 'loading'],
  emits: ['click'],
  template: '<button :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
}
const stubs = {
  'el-alert': passThrough,
  'el-card': passThrough,
  'el-table': tableStub,
  'el-table-column': { template: '<div />' },
  'el-tag': passThrough,
  'el-button': buttonStub,
  'el-switch': { template: '<div />' },
  'el-input-number': { template: '<div />' },
  'el-input': { template: '<div />' },
  'el-empty': passThrough,
}

function makeSetting(key: string, value: string, valueType = 'bool'): SettingItem {
  return {
    key,
    value,
    value_type: valueType,
    default_value: value,
    is_public: true,
    source: 'db',
    description: `desc-${key}`,
    updated_at: '2026-01-01 09:00:00',
  }
}

const threeSettings = [
  makeSetting('registration_enabled', 'true'),
  makeSetting('maintenance_mode', 'false'),
  makeSetting('maintenance_message', '系统维护中', 'string'),
]

async function mountView() {
  const wrapper = mount(SettingsView, { global: { stubs, directives: { loading: {} } } })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(listSystemSettings).mockResolvedValue(threeSettings)
})

describe('系统配置管理页', () => {
  it('装载后表格获得全量三行配置', async () => {
    const wrapper = await mountView()
    expect(listSystemSettings).toHaveBeenCalledTimes(1)
    expect(wrapper.find('.mx-table').text()).toBe('3')
  })

  it('无改动时保存按钮禁用且不显示脏提示', async () => {
    const wrapper = await mountView()
    const save = wrapper.findAll('button').find((b) => b.text().includes('settingsPage.save'))
    expect(save?.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).not.toContain('settingsPage.dirtyHint')
  })

  it('修改草稿后仅提交差量，成功后以返回值重建基线', async () => {
    const wrapper = await mountView()
    const vm = wrapper.vm as unknown as { draft: Record<string, string> }
    vm.draft.registration_enabled = 'false'
    await flushPromises()

    expect(wrapper.text()).toContain('settingsPage.dirtyHint:1')
    const saved = wrapper.find('settingsPage.savedSuccess')
    expect(saved.exists()).toBe(false)

    const updated = [makeSetting('registration_enabled', 'false')]
    vi.mocked(updateSystemSettings).mockResolvedValue(updated)
    const save = wrapper.findAll('button').find((b) => b.text().includes('settingsPage.save'))!
    await save.trigger('click')
    await flushPromises()

    expect(updateSystemSettings).toHaveBeenCalledWith([{ key: 'registration_enabled', value: 'false' }])
    // 保存成功提示 + 基线重建后按钮回到禁用
    expect(ElMessage.success).toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('settingsPage.dirtyHint')
  })

  it('保存失败保留草稿供修正', async () => {
    const wrapper = await mountView()
    const vm = wrapper.vm as unknown as { draft: Record<string, string> }
    vm.draft.maintenance_message = '今晚 22:00 升级'
    await flushPromises()

    vi.mocked(updateSystemSettings).mockRejectedValue(new Error('bad value'))
    const save = wrapper.findAll('button').find((b) => b.text().includes('settingsPage.save'))!
    await save.trigger('click')
    await flushPromises()

    expect(ElMessage.error).toHaveBeenCalled()
    // 草稿未被重建，仍处脏状态
    expect(vm.draft.maintenance_message).toBe('今晚 22:00 升级')
    expect(wrapper.text()).toContain('settingsPage.dirtyHint:1')
  })

  it('放弃修改将草稿还原为服务端值', async () => {
    const wrapper = await mountView()
    const vm = wrapper.vm as unknown as { draft: Record<string, string> }
    vm.draft.registration_enabled = 'false'
    await flushPromises()

    const reset = wrapper.findAll('button').find((b) => b.text().includes('settingsPage.reset'))!
    await reset.trigger('click')
    await flushPromises()

    expect(vm.draft.registration_enabled).toBe('true')
    expect(wrapper.text()).not.toContain('settingsPage.dirtyHint')
  })
})
