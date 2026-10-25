import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listMessages,
  getUnreadCount,
  markMessageRead,
  markAllMessagesRead,
  deleteMessage,
  getMessagePreferences,
  updateMessagePreference,
  type MessageItem,
  type UnreadCountResult,
  type MessagePreferencesResult,
} from './message'

vi.mock('@/api/request', () => ({
  get: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

import { get, put, del } from '@/api/request'

const sampleMessage: MessageItem = {
  id: 'msg-1',
  category: 'task',
  title: '任务即将到期',
  content: '请尽快处理',
  link: '/task',
  priority: 1,
  source_type: 'task',
  source_id: 'tk-1',
  read_at: null,
  created_at: '2026-10-23 10:00:00',
}

describe('Message API', () => {
  beforeEach(() => vi.clearAllMocks())

  it('listMessages 应携带分页与过滤参数请求 /messages', async () => {
    vi.mocked(get).mockResolvedValue({ list: [sampleMessage], total: 1 })
    const res = await listMessages({ page: 2, page_size: 20, category: 'task', unread: true, keyword: '到期' })
    expect(get).toHaveBeenCalledWith('/messages', {
      page: 2,
      page_size: 20,
      category: 'task',
      unread: true,
      keyword: '到期',
    })
    expect(res.list[0].id).toBe('msg-1')
  })

  it('getUnreadCount 应 GET /messages/unread-count', async () => {
    const payload: UnreadCountResult = { total: 3, by_category: { task: 2, system: 1 } }
    vi.mocked(get).mockResolvedValue(payload)
    const res = await getUnreadCount()
    expect(get).toHaveBeenCalledWith('/messages/unread-count')
    expect(res.total).toBe(3)
    expect(res.by_category.task).toBe(2)
  })

  it('markMessageRead 应 PUT /messages/{id}/read', async () => {
    vi.mocked(put).mockResolvedValue({ id: 'msg-1' })
    await markMessageRead('msg-1')
    expect(put).toHaveBeenCalledWith('/messages/msg-1/read')
  })

  it('markAllMessagesRead 无分类时应发送空对象', async () => {
    vi.mocked(put).mockResolvedValue({ updated: 5 })
    const res = await markAllMessagesRead()
    expect(put).toHaveBeenCalledWith('/messages/read-all', {})
    expect(res.updated).toBe(5)
  })

  it('markAllMessagesRead 传分类时应只提交分类', async () => {
    vi.mocked(put).mockResolvedValue({ updated: 2 })
    await markAllMessagesRead('task')
    expect(put).toHaveBeenCalledWith('/messages/read-all', { category: 'task' })
  })

  it('deleteMessage 应 DELETE /messages/{id}', async () => {
    vi.mocked(del).mockResolvedValue({ id: 'msg-1' })
    await deleteMessage('msg-1')
    expect(del).toHaveBeenCalledWith('/messages/msg-1')
  })

  it('getMessagePreferences 应 GET /messages/preferences', async () => {
    const payload: MessagePreferencesResult = {
      items: [
        { category: 'task', muted: true, mandatory: false },
        { category: 'audit', muted: false, mandatory: true },
      ],
    }
    vi.mocked(get).mockResolvedValue(payload)
    const res = await getMessagePreferences()
    expect(get).toHaveBeenCalledWith('/messages/preferences')
    expect(res.items).toHaveLength(2)
    expect(res.items[1].mandatory).toBe(true)
  })

  it('updateMessagePreference 应 PUT 分类与开关值', async () => {
    vi.mocked(put).mockResolvedValue({ items: [] })
    await updateMessagePreference('task', true)
    expect(put).toHaveBeenCalledWith('/messages/preferences', { category: 'task', muted: true })
  })
})
