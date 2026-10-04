import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getTaskList,
  getTask,
  createTask,
  updateTask,
  completeTask,
  reopenTask,
  deleteTask,
  getTaskStats,
  getTaskTrash,
  restoreTask,
  permanentDeleteTask,
  batchCompleteTasks,
  batchDeleteTasks,
  batchUpdateTaskStatus,
  batchAssignTasks,
  type TaskItem,
  type CreateTaskParams,
} from './task'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

import { get, post, put, del } from '@/api/request'

const sampleTask: TaskItem = {
  id: 't-1',
  tenant_id: 'tn-1',
  creator_id: 'u-1',
  creator_name: '张三',
  assignee_id: 'u-1',
  assignee_name: '张三',
  title: '写周报',
  description: '',
  status: 'pending',
  priority: 'normal',
  tags: ['工作'],
  visibility: 'personal',
  created_at: '2026-10-01 09:00:00',
  updated_at: '2026-10-01 09:00:00',
}

describe('Task API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getTaskList 应携带查询参数请求 /tasks', async () => {
    vi.mocked(get).mockResolvedValue({ data: [sampleTask], pagination: { total: 1 } })
    await getTaskList({ page: 1, page_size: 20, status: 'pending' })
    expect(get).toHaveBeenCalledWith('/tasks', { page: 1, page_size: 20, status: 'pending' })
  })

  it('getTask 应 GET 单个任务', async () => {
    vi.mocked(get).mockResolvedValue(sampleTask)
    const result = await getTask('t-1')
    expect(get).toHaveBeenCalledWith('/tasks/t-1')
    expect(result.id).toBe('t-1')
  })

  it('createTask 应 POST /tasks', async () => {
    const payload: CreateTaskParams = { title: '写周报', priority: 'high' }
    vi.mocked(post).mockResolvedValue(sampleTask)
    await createTask(payload)
    expect(post).toHaveBeenCalledWith('/tasks', payload)
  })

  it('createTask 携带 remind_before 时应透传该字段', async () => {
    const payload: CreateTaskParams = { title: '带提前量', due_date: '2026-12-31 10:00:00', remind_before: '2h' }
    vi.mocked(post).mockResolvedValue({ ...sampleTask, remind_before: '2h0m0s' })
    const res = await createTask(payload)
    expect(post).toHaveBeenCalledWith('/tasks', payload)
    expect(res.remind_before).toBe('2h0m0s')
  })

  it('createTask 携带 recurrence 时应透传重复周期', async () => {
    const payload: CreateTaskParams = { title: '周报', due_date: '2026-12-31 10:00:00', recurrence: 'weekly' }
    vi.mocked(post).mockResolvedValue({ ...sampleTask, recurrence: 'weekly' })
    const res = await createTask(payload)
    expect(post).toHaveBeenCalledWith('/tasks', payload)
    expect(res.recurrence).toBe('weekly')
  })

  it('updateTask 应 PUT /tasks/{id}', async () => {
    vi.mocked(put).mockResolvedValue(sampleTask)
    await updateTask('t-1', { title: '改名' })
    expect(put).toHaveBeenCalledWith('/tasks/t-1', { title: '改名' })
  })

  it('completeTask 应 PUT 完成接口', async () => {
    vi.mocked(put).mockResolvedValue(sampleTask)
    await completeTask('t-1')
    expect(put).toHaveBeenCalledWith('/tasks/t-1/complete')
  })

  it('reopenTask 携带状态时应发送 status 请求体', async () => {
    vi.mocked(put).mockResolvedValue(sampleTask)
    await reopenTask('t-1', 'in_progress')
    expect(put).toHaveBeenCalledWith('/tasks/t-1/reopen', { status: 'in_progress' })
  })

  it('reopenTask 不携带状态时请求体为 undefined', async () => {
    vi.mocked(put).mockResolvedValue(sampleTask)
    await reopenTask('t-1')
    expect(put).toHaveBeenCalledWith('/tasks/t-1/reopen', undefined)
  })

  it('deleteTask 应 DELETE /tasks/{id}', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await deleteTask('t-1')
    expect(del).toHaveBeenCalledWith('/tasks/t-1')
  })

  it('getTaskStats 无参数时应以 undefined 请求 /tasks/stats', async () => {
    vi.mocked(get).mockResolvedValue({ pending: 2, in_progress: 1, completed: 5, overdue: 0, total: 8, completion_rate: 0.625, avg_handle_seconds: 7200 })
    const stats = await getTaskStats()
    expect(get).toHaveBeenCalledWith('/tasks/stats', undefined)
    expect(stats.pending).toBe(2)
    expect(stats.completion_rate).toBe(0.625)
    expect(stats.avg_handle_seconds).toBe(7200)
  })

  it('getTaskStats 指定 visibility 时应透传参数', async () => {
    vi.mocked(get).mockResolvedValue({ pending: 0, in_progress: 0, completed: 0, overdue: 0, total: 0, completion_rate: 0, avg_handle_seconds: 0 })
    await getTaskStats({ visibility: 'tenant' })
    expect(get).toHaveBeenCalledWith('/tasks/stats', { visibility: 'tenant' })
  })

  it('getTaskTrash 应 GET /tasks/deleted 并透传分页', async () => {
    vi.mocked(get).mockResolvedValue({ data: [sampleTask], pagination: { total: 1 } })
    await getTaskTrash({ page: 1, page_size: 10 })
    expect(get).toHaveBeenCalledWith('/tasks/deleted', { page: 1, page_size: 10 })
  })

  it('getTaskTrash 无参数时应以 undefined 请求', async () => {
    vi.mocked(get).mockResolvedValue({ data: [], pagination: { total: 0 } })
    await getTaskTrash()
    expect(get).toHaveBeenCalledWith('/tasks/deleted', undefined)
  })

  it('restoreTask 应 PUT 恢复接口', async () => {
    vi.mocked(put).mockResolvedValue(sampleTask)
    await restoreTask('t-1')
    expect(put).toHaveBeenCalledWith('/tasks/t-1/restore')
  })

  it('permanentDeleteTask 应 DELETE 永久删除接口', async () => {
    vi.mocked(del).mockResolvedValue(undefined)
    await permanentDeleteTask('t-1')
    expect(del).toHaveBeenCalledWith('/tasks/t-1/permanent')
  })

  it('batchCompleteTasks 应 POST 批量完成接口', async () => {
    vi.mocked(post).mockResolvedValue({ affected: 2 })
    const res = await batchCompleteTasks(['a', 'b'])
    expect(post).toHaveBeenCalledWith('/tasks/batch/complete', { ids: ['a', 'b'] })
    expect(res.affected).toBe(2)
  })

  it('batchDeleteTasks 应 POST 批量删除接口', async () => {
    vi.mocked(post).mockResolvedValue({ affected: 1 })
    await batchDeleteTasks(['a'])
    expect(post).toHaveBeenCalledWith('/tasks/batch/delete', { ids: ['a'] })
  })

  it('batchUpdateTaskStatus 应 POST 批量改状态接口', async () => {
    vi.mocked(post).mockResolvedValue({ affected: 2 })
    const res = await batchUpdateTaskStatus({ ids: ['a', 'b'], status: 'in_progress' })
    expect(post).toHaveBeenCalledWith('/tasks/batch/status', { ids: ['a', 'b'], status: 'in_progress' })
    expect(res.affected).toBe(2)
  })

  it('batchAssignTasks 应 POST 批量指派接口', async () => {
    vi.mocked(post).mockResolvedValue({ affected: 1 })
    await batchAssignTasks({ ids: ['a'], assignee_id: 'u-2' })
    expect(post).toHaveBeenCalledWith('/tasks/batch/assign', { ids: ['a'], assignee_id: 'u-2' })
  })
})
