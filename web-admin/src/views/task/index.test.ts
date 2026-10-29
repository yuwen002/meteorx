// 任务看板视图（board mode）测试
//
// 目标：断言看板三列按状态分组渲染、卡片拖拽到不同列即调用批量改状态接口、
// 以及结果数超过已加载数时展示截断提示。
// vitest 未启用 unplugin 自动导入，故 el-* 组件、v-loading 指令均以桩件透传渲染。
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

// i18n：t 直接回显 key（忽略占位参数），便于断言文案键
vi.mock('@/composables/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

// 路由：任务页读取 route.query.task_id，给空查询即可
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ push: vi.fn() }),
}))

// element-plus：拦截器/提示以桩件替代，避免真实弹窗与重量级导入
vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
  ElMessageBox: { confirm: vi.fn().mockResolvedValue(true) },
}))

// 任务接口：仅看板链路关心 getTaskList / getTaskStats / batchUpdateTaskStatus
vi.mock('@/api/modules/task', () => ({
  getTaskList: vi.fn(),
  getTask: vi.fn(),
  createTask: vi.fn(),
  updateTask: vi.fn(),
  completeTask: vi.fn(),
  reopenTask: vi.fn(),
  deleteTask: vi.fn(),
  getTaskStats: vi.fn().mockResolvedValue({ pending: 0, in_progress: 0, completed: 0, overdue: 0, total: 0, completion_rate: 0, avg_handle_seconds: 0 }),
  getTaskTrash: vi.fn(),
  restoreTask: vi.fn(),
  permanentDeleteTask: vi.fn(),
  batchCompleteTasks: vi.fn(),
  batchDeleteTasks: vi.fn(),
  batchUpdateTaskStatus: vi.fn().mockResolvedValue({ affected: 1 }),
  batchAssignTasks: vi.fn(),
}))

vi.mock('@/api/modules/task-ai', () => ({
  getAICapabilities: vi.fn().mockResolvedValue({ enabled: false, model: '' }),
  parseTaskByNL: vi.fn(),
  suggestTaskFields: vi.fn(),
  expandTaskDescription: vi.fn(),
  summarizeTasks: vi.fn(),
}))

import TaskView from './index.vue'
import { getTaskList, batchUpdateTaskStatus } from '@/api/modules/task'

// 透传桩：渲染默认插槽使卡片文本可断言
const passThrough = { template: '<div><slot name="header" /><slot /></div>' }
// 空桩：表格/抽屉/对话框内含 #default="{ row }" 作用域插槽，透传会提前以空 row 渲染报错；
// 看板区域本身用原生 div 不依赖它们，测试中直接置空。
const empty = { template: '<div />' }
const stubs = {
  'el-row': passThrough,
  'el-col': passThrough,
  'el-card': passThrough,
  'el-icon': passThrough,
  'el-tag': passThrough,
  'el-button': passThrough,
  'el-table': empty,
  'el-table-column': empty,
  'el-empty': passThrough,
  'el-divider': passThrough,
  'el-radio-group': passThrough,
  'el-radio-button': passThrough,
  'el-select': passThrough,
  'el-option': passThrough,
  'el-tooltip': passThrough,
  'el-input': passThrough,
  'el-date-picker': passThrough,
  'el-dialog': empty,
  'el-dropdown': passThrough,
  'el-dropdown-menu': passThrough,
  'el-dropdown-item': passThrough,
  'el-pagination': passThrough,
  'el-drawer': empty,
  'el-form': passThrough,
  'el-form-item': passThrough,
  'router-link': { template: '<a><slot /></a>' },
}

function makeTask(id: string, status: 'pending' | 'in_progress' | 'completed') {
  return {
    id,
    tenant_id: 't1',
    creator_id: 'u1',
    assignee_id: 'u1',
    title: `任务-${id}`,
    description: '',
    status,
    priority: 'normal',
    tags: [],
    visibility: 'personal',
    created_at: '2026-01-01 00:00:00',
    updated_at: '2026-01-01 00:00:00',
  }
}

async function mountBoard() {
  localStorage.setItem('meteorx.task.viewMode', 'board')
  const wrapper = mount(TaskView, {
    global: { stubs, directives: { loading: {} } },
  })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  vi.mocked(batchUpdateTaskStatus).mockResolvedValue({ affected: 1 })
})

describe('任务看板视图', () => {
  it('看板模式按状态渲染三列并分列计数', async () => {
    vi.mocked(getTaskList).mockResolvedValue({
      list: [makeTask('a', 'pending'), makeTask('b', 'in_progress'), makeTask('c', 'completed')],
      total: 3,
    })
    const wrapper = await mountBoard()
    const cols = wrapper.findAll('.board-col')
    expect(cols).toHaveLength(3)
    // 每列各 1 张卡片
    expect(cols[0].findAll('.board-card')).toHaveLength(1)
    expect(cols[1].findAll('.board-card')).toHaveLength(1)
    expect(cols[2].findAll('.board-card')).toHaveLength(1)
    // 卡片标题渲染
    expect(wrapper.text()).toContain('任务-a')
  })

  it('卡片拖拽到不同状态列触发批量改状态接口', async () => {
    vi.mocked(getTaskList).mockResolvedValue({
      list: [makeTask('a', 'pending'), makeTask('c', 'completed')],
      total: 2,
    })
    const wrapper = await mountBoard()
    const cols = wrapper.findAll('.board-col')
    // 拖起 pending 列首卡，投放到 completed 列
    await cols[0].find('.board-card').trigger('dragstart')
    await cols[2].trigger('drop')
    await flushPromises()
    expect(batchUpdateTaskStatus).toHaveBeenCalledTimes(1)
    expect(batchUpdateTaskStatus).toHaveBeenCalledWith({ ids: ['a'], status: 'completed' })
  })

  it('投放到同状态列不调用接口', async () => {
    vi.mocked(getTaskList).mockResolvedValue({
      list: [makeTask('a', 'pending')],
      total: 1,
    })
    const wrapper = await mountBoard()
    const cols = wrapper.findAll('.board-col')
    await cols[0].find('.board-card').trigger('dragstart')
    await cols[0].trigger('drop')
    await flushPromises()
    expect(batchUpdateTaskStatus).not.toHaveBeenCalled()
  })

  it('结果数超过已加载数时展示截断提示', async () => {
    vi.mocked(getTaskList).mockResolvedValue({
      list: [makeTask('a', 'pending')],
      total: 137,
    })
    const wrapper = await mountBoard()
    expect(wrapper.find('.board-truncated').exists()).toBe(true)
    expect(wrapper.find('.board-truncated').text()).toContain('taskPage.board.truncated')
  })

  it('空列展示占位文案', async () => {
    vi.mocked(getTaskList).mockResolvedValue({
      list: [makeTask('a', 'pending')],
      total: 1,
    })
    const wrapper = await mountBoard()
    const cols = wrapper.findAll('.board-col')
    // completed 列无卡片，展示空占位
    expect(cols[2].find('.board-col__empty').exists()).toBe(true)
    expect(cols[2].find('.board-col__empty').text()).toContain('taskPage.board.emptyCol')
  })
})
