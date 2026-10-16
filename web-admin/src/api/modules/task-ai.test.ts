import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getAICapabilities,
  parseTaskByNL,
  suggestTaskFields,
  expandTaskDescription,
  summarizeTasks,
  type AICapability,
  type TaskDraft,
  type TaskSuggestion,
  type TaskExpansion,
  type TaskSummary,
} from './task-ai'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
  default: { get: vi.fn() },
}))

import { get, post } from '@/api/request'

describe('Task AI API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('getAICapabilities 应 GET /tasks/ai/capabilities', async () => {
    const cap: AICapability = { enabled: true, model: 'gpt-4o-mini' }
    vi.mocked(get).mockResolvedValue(cap)
    const res = await getAICapabilities()
    expect(get).toHaveBeenCalledWith('/tasks/ai/capabilities')
    expect(res.enabled).toBe(true)
    expect(res.model).toBe('gpt-4o-mini')
  })

  it('parseTaskByNL 应 POST /tasks/ai/parse 并携带 text', async () => {
    const draft: TaskDraft = { title: '写周报', priority: 'high', tags: ['工作'] }
    vi.mocked(post).mockResolvedValue(draft)
    const res = await parseTaskByNL('下周五前高优先级写周报')
    expect(post).toHaveBeenCalledWith('/tasks/ai/parse', { text: '下周五前高优先级写周报' })
    expect(res.title).toBe('写周报')
  })

  it('suggestTaskFields 应 POST /tasks/ai/suggest，缺省描述转为空串', async () => {
    const s: TaskSuggestion = { priority: 'urgent', tags: ['线上'], reason: '线上故障' }
    vi.mocked(post).mockResolvedValue(s)
    const res = await suggestTaskFields('修复登录报错')
    expect(post).toHaveBeenCalledWith('/tasks/ai/suggest', { title: '修复登录报错', description: '' })
    expect(res.priority).toBe('urgent')
  })

  it('expandTaskDescription 应 POST /tasks/ai/expand', async () => {
    const e: TaskExpansion = { description: '完善并同步', steps: ['收集', '撰写'] }
    vi.mocked(post).mockResolvedValue(e)
    const res = await expandTaskDescription('写周报', '初稿')
    expect(post).toHaveBeenCalledWith('/tasks/ai/expand', { title: '写周报', description: '初稿' })
    expect(res.steps).toHaveLength(2)
  })

  it('summarizeTasks 默认 scope=all', async () => {
    const sum: TaskSummary = { text: '共 3 条', highlights: [], suggested_focus: [], total: 3 }
    vi.mocked(post).mockResolvedValue(sum)
    const res = await summarizeTasks()
    expect(post).toHaveBeenCalledWith('/tasks/ai/summary', { scope: 'all' })
    expect(res.total).toBe(3)
  })

  it('summarizeTasks 可指定 scope=overdue', async () => {
    vi.mocked(post).mockResolvedValue({ text: '', highlights: [], suggested_focus: [], total: 0 })
    await summarizeTasks('overdue')
    expect(post).toHaveBeenCalledWith('/tasks/ai/summary', { scope: 'overdue' })
  })
})
