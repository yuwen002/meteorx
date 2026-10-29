// 租户工作台首页（非超管视角）视图测试
//
// 目标：断言工作台四类卡片渲染、团队任务概况卡消费 tenant-overview、逾期>0 标红、
// 存储格式化边界（1024 → "1.0 KB"）正确，且不会误渲染超管运营看板分支。
// vitest 配置未启用 unplugin 自动导入，故 el-* 组件与 router-link 以透传桩件渲染插槽内容。
import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// i18n：t 直接回显 key，便于断言各卡片文案键
vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

// 工作台统计：以固定对象驱动模板渲染，隔离真实网络装载
vi.mock('@/composables/useWorkspaceStats', () => ({
  useWorkspaceStats: vi.fn(() => ({
    task: {
      pending: 1,
      in_progress: 1,
      completed: 1,
      overdue: 0,
      total: 3,
      completion_rate: 0.33,
      avg_handle_seconds: 0
    },
    tenantTask: { pending: 5, in_progress: 3, completed: 10, overdue: 2, total: 18, completion_rate: 0.55 },
    wiki: { total_spaces: 2, total_nodes: 0, total_documents: 5, total_views: 40 },
    memberCount: 7,
    storageUsed: 1024,
    unreadMessages: 4,
    unreadNotifications: 2,
    unreadChats: 1,
    mentionChats: 1,
    load: vi.fn().mockResolvedValue(undefined)
  }))
}))

// 非超管身份：走工作台分支
vi.mock('@/stores/user', () => ({
  useUserStore: () => ({ userInfo: { is_master: false, nickname: 'Tom', username: 'tom' } })
}))

// 路由与超管看板接口：工作台分支不触发，桩接避免真实依赖
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('@/api/modules/role', () => ({ getRBACStats: vi.fn() }))
vi.mock('@/api/modules/dashboard', () => ({ getDashboardOverview: vi.fn(), getGrowthTrend: vi.fn() }))

import Dashboard from './index.vue'

// 透传桩：渲染 header 与默认插槽，使卡片文本可被断言
const passThrough = { template: '<div><slot name="header" /><slot /></div>' }
const stubs = {
  'el-row': passThrough,
  'el-col': passThrough,
  'el-card': passThrough,
  'el-icon': passThrough,
  'el-progress': passThrough,
  'el-badge': passThrough,
  'el-tag': passThrough,
  'el-button': passThrough,
  'el-table': passThrough,
  'el-table-column': passThrough,
  'el-empty': passThrough,
  'el-divider': passThrough,
  'el-radio-group': passThrough,
  'el-radio-button': passThrough,
  'el-select': passThrough,
  'el-option': passThrough,
  'router-link': { template: '<a><slot /></a>' }
}

async function mountWorkspace() {
  const wrapper = mount(Dashboard, { global: { stubs } })
  await flushPromises()
  return wrapper
}

describe('dashboard 租户工作台视图', () => {
  it('渲染四类个人卡片标题', async () => {
    const text = (await mountWorkspace()).text()
    expect(text).toContain('dashboardPage.ws.taskTitle')
    expect(text).toContain('dashboardPage.ws.wikiTitle')
    expect(text).toContain('dashboardPage.ws.memberTitle')
    expect(text).toContain('dashboardPage.ws.completion')
  })

  it('团队任务概况卡消费 tenant-overview 的租户级任务分布', async () => {
    const text = (await mountWorkspace()).text()
    expect(text).toContain('dashboardPage.ws.teamTaskTitle')
    // 团队总任务数 18 仅来自 tenantTask（个人任务 total=3，不冲突）
    expect(text).toContain('18')
    expect(text).toContain('dashboardPage.ws.taskDone')
    expect(text).toContain('dashboardPage.ws.taskTotal')
  })

  it('逾期>0 标红：个人任务逾期 0 不标红，仅团队任务逾期标红一处', async () => {
    const wrapper = await mountWorkspace()
    expect(wrapper.findAll('.overdue-red')).toHaveLength(1)
  })

  it('成员卡展示存储并修正 1024 的格式化边界为 "1.0 KB"', async () => {
    const text = (await mountWorkspace()).text()
    expect(text).toContain('7') // memberCount
    expect(text).toContain('1.0 KB') // formatBytes(1024) 边界
    expect(text).not.toContain('1024 B')
  })

  it('非超管不渲染平台运营看板分支', async () => {
    const text = (await mountWorkspace()).text()
    expect(text).not.toContain('dashboardPage.stat.tenantTotal')
    expect(text).not.toContain('dashboardPage.trend.title')
  })
})
