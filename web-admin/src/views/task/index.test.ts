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

// 任务接口：看板链路关心 getTaskBoard / getTaskStats / batchUpdateTaskStatus
vi.mock('@/api/modules/task', () => ({
  getTaskList: vi.fn(),
  getTaskBoard: vi.fn(),
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
  getAssignableMembers: vi.fn().mockResolvedValue([{ id: 'u1', name: '张三' }, { id: 'u2', name: '李四' }]),
}))

vi.mock('@/api/modules/task-ai', () => ({
  getAICapabilities: vi.fn().mockResolvedValue({ enabled: false, model: '' }),
  parseTaskByNL: vi.fn(),
  suggestTaskFields: vi.fn(),
  expandTaskDescription: vi.fn(),
  summarizeTasks: vi.fn(),
}))

import TaskView from './index.vue'
import { getTaskList, getTaskBoard, batchUpdateTaskStatus, getAssignableMembers, type TaskItem } from '@/api/modules/task'

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

function makeTask(id: string, status: TaskItem['status'], extra: Partial<TaskItem> = {}): TaskItem {
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
    ...extra,
  }
}

async function mountBoard(groupBy?: 'status' | 'priority' | 'assignee' | 'tag') {
  localStorage.setItem('meteorx.task.viewMode', 'board')
  if (groupBy) localStorage.setItem('meteorx.task.groupBy', groupBy)
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

// el-table 专用桩：具名以便测试通过 $emit('expand-change') 驱动行首展开懒加载逻辑。
const ElTableStub = {
  name: 'ElTable',
  emits: ['expand-change', 'selection-change'],
  template: '<div class="el-table-stub"><slot /></div>',
}

async function mountList(tableRows: TaskItem[]) {
  localStorage.setItem('meteorx.task.viewMode', 'list')
  vi.mocked(getTaskList).mockResolvedValue({ list: tableRows, total: tableRows.length } as never)
  const wrapper = mount(TaskView, {
    global: { stubs: { ...stubs, 'el-table': ElTableStub }, directives: { loading: {} } },
  })
  await flushPromises()
  return wrapper
}

describe('任务看板视图', () => {
  it('看板模式按状态渲染三列并分列计数', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [makeTask('a', 'pending'), makeTask('b', 'in_progress'), makeTask('c', 'completed')],
      total: 3,
      truncated: false,
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
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [makeTask('a', 'pending'), makeTask('c', 'completed')],
      total: 2,
      truncated: false,
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
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [makeTask('a', 'pending')],
      total: 1,
      truncated: false,
    })
    const wrapper = await mountBoard()
    const cols = wrapper.findAll('.board-col')
    await cols[0].find('.board-card').trigger('dragstart')
    await cols[0].trigger('drop')
    await flushPromises()
    expect(batchUpdateTaskStatus).not.toHaveBeenCalled()
  })

  it('后端标记截断（总数超上限）时展示截断提示', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [makeTask('a', 'pending')],
      total: 600,
      truncated: true,
    })
    const wrapper = await mountBoard()
    expect(wrapper.find('.board-truncated').exists()).toBe(true)
    expect(wrapper.find('.board-truncated').text()).toContain('taskPage.board.truncated')
  })

  it('未截断时即使总数大于当前列卡也不提示', async () => {
    // 全量加载后 items 已含当前筛选全部（例：看板仅 1 列有卡但总数为全部任务）
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [makeTask('a', 'pending'), makeTask('b', 'completed')],
      total: 2,
      truncated: false,
    })
    const wrapper = await mountBoard()
    expect(wrapper.find('.board-truncated').exists()).toBe(false)
  })

  it('空列展示占位文案', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [makeTask('a', 'pending')],
      total: 1,
      truncated: false,
    })
    const wrapper = await mountBoard()
    const cols = wrapper.findAll('.board-col')
    // completed 列无卡片，展示空占位
    expect(cols[2].find('.board-col__empty').exists()).toBe(true)
    expect(cols[2].find('.board-col__empty').text()).toContain('taskPage.board.emptyCol')
  })

  it('按优先级分组时渲染固定四列并按 priority 分桶', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [
        makeTask('u', 'pending', { priority: 'urgent' }),
        makeTask('h', 'pending', { priority: 'high' }),
        makeTask('n', 'in_progress', { priority: 'normal' }),
        makeTask('l', 'completed', { priority: 'low' }),
      ],
      total: 4,
      truncated: false,
    })
    const wrapper = await mountBoard('priority')
    const cols = wrapper.findAll('.board-col')
    // urgent/high/normal/low 固定四列
    expect(cols).toHaveLength(4)
    expect(cols[0].findAll('.board-card')).toHaveLength(1)
    expect(cols[3].findAll('.board-card')).toHaveLength(1)
  })

  it('按负责人分组时按 assignee_id 动态分列', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [
        makeTask('a', 'pending', { assignee_id: 'u1', assignee_name: '张三' }),
        makeTask('b', 'pending', { assignee_id: 'u1', assignee_name: '张三' }),
        makeTask('c', 'completed', { assignee_id: 'u2', assignee_name: '李四' }),
      ],
      total: 3,
      truncated: false,
    })
    const wrapper = await mountBoard('assignee')
    const cols = wrapper.findAll('.board-col')
    expect(cols).toHaveLength(2)
    // 按卡片数降序，张三列在前且 2 张卡
    expect(cols[0].findAll('.board-card')).toHaveLength(2)
    expect(cols[1].findAll('.board-card')).toHaveLength(1)
  })

  it('按标签分组时多标签任务出现在每个标签列并附未分类列', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [
        makeTask('a', 'pending', { tags: ['x', 'y'] }),
        makeTask('b', 'pending', { tags: ['x'] }),
        makeTask('c', 'pending', { tags: [] }),
      ],
      total: 3,
      truncated: false,
    })
    const wrapper = await mountBoard('tag')
    const cols = wrapper.findAll('.board-col')
    // x、y 两列 + 未分类列 = 3 列
    expect(cols).toHaveLength(3)
    // 最后一列为未分类，包含无标签任务 c
    const lastCol = cols[cols.length - 1]
    expect(lastCol.text()).toContain('taskPage.board.untagged')
    expect(lastCol.findAll('.board-card')).toHaveLength(1)
  })

  it('非状态分组时卡片不可拖拽', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({
      items: [makeTask('a', 'pending', { priority: 'urgent' })],
      total: 1,
      truncated: false,
    })
    const wrapper = await mountBoard('priority')
    const card = wrapper.find('.board-card')
    expect(card.attributes('draggable')).toBe('false')
    // 拖放也不应触发改状态接口
    await card.trigger('dragstart')
    await wrapper.findAll('.board-col')[1].trigger('drop')
    await flushPromises()
    expect(batchUpdateTaskStatus).not.toHaveBeenCalled()
  })

  it('看板请求默认携带 root_only 仅含根任务', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({ items: [], total: 0, truncated: false })
    await mountBoard()
    expect(vi.mocked(getTaskBoard).mock.calls[0][0]).toMatchObject({ root_only: true })
  })

  it('打开新建弹窗时加载可指派成员（负责人多选数据源）', async () => {
    vi.mocked(getTaskBoard).mockResolvedValue({ items: [], total: 0, truncated: false })
    const wrapper = await mountBoard()
    expect(getAssignableMembers).not.toHaveBeenCalled()
    // 点击工具栏“新建任务”按钮触发 openDialog → loadAssignableMembers。
    // el-button 桩渲染为 div（非原生 button），取文本命中的最深元素即按钮本体。
    const btns = wrapper.findAll('*').filter((el) => el.text().includes('taskPage.create'))
    const createBtn = btns[btns.length - 1]
    expect(createBtn).toBeTruthy()
    await createBtn.trigger('click')
    await flushPromises()
    expect(getAssignableMembers).toHaveBeenCalledTimes(1)
  })
})

describe('任务子任务（B3）', () => {
  it('列表视图默认只请求根任务 root_only', async () => {
    await mountList([makeTask('r1', 'pending')])
    const first = vi.mocked(getTaskList).mock.calls[0][0]
    expect(first).toMatchObject({ root_only: true })
  })

  it('展开行时按 parent_id 懒加载子任务', async () => {
    const root = makeTask('r1', 'pending')
    const wrapper = await mountList([root])
    // 首次 mount 已调用一次 loadList，展开前应无 parent_id 查询
    expect(vi.mocked(getTaskList).mock.calls.every((c) => !(c[0] as { parent_id?: string })?.parent_id)).toBe(true)
    // 触发展开：emit expand-change(row, 当前展开行数组含 row)
    wrapper.findComponent(ElTableStub).vm.$emit('expand-change', root, [root])
    await flushPromises()
    const calls = vi.mocked(getTaskList).mock.calls.map((c) => c[0] as { parent_id?: string })
    expect(calls.some((p) => p?.parent_id === 'r1')).toBe(true)
  })

  it('收起行不触发子任务查询', async () => {
    const root = makeTask('r1', 'pending')
    const wrapper = await mountList([root])
    const before = vi.mocked(getTaskList).mock.calls.length
    // expanded 为空数组（收起），不应新增 getTaskList 调用
    wrapper.findComponent(ElTableStub).vm.$emit('expand-change', root, [])
    await flushPromises()
    expect(vi.mocked(getTaskList).mock.calls.length).toBe(before)
  })
})
