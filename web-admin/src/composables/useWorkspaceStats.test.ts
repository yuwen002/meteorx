import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// 隔离网络层：工作台各统计源 + 会话列表全部 mock，聚焦装载与降级/兜底逻辑
vi.mock('@/api/modules/task', () => ({ getTaskStats: vi.fn() }))
vi.mock('@/api/modules/wiki', () => ({ getWikiStats: vi.fn(), getUnreadNotificationCount: vi.fn() }))
vi.mock('@/api/modules/user', () => ({ getUserStats: vi.fn() }))
vi.mock('@/api/modules/message', () => ({ getUnreadCount: vi.fn() }))
vi.mock('@/api/modules/dashboard', () => ({ getTenantOverview: vi.fn() }))
vi.mock('@/api/modules/chat', () => ({ listConversations: vi.fn() }))

import { getTaskStats } from '@/api/modules/task'
import { getWikiStats, getUnreadNotificationCount } from '@/api/modules/wiki'
import { getUserStats } from '@/api/modules/user'
import { getUnreadCount } from '@/api/modules/message'
import { getTenantOverview } from '@/api/modules/dashboard'
import { listConversations } from '@/api/modules/chat'
import type { ConversationItem } from '@/api/modules/chat'
import { useChatStore } from '@/stores/chat'
import { useWorkspaceStats } from './useWorkspaceStats'

function conv(over: Partial<ConversationItem> = {}): ConversationItem {
  return {
    id: 'g1',
    type: 'group',
    peer_user_id: '',
    peer_name: '研发群',
    last_message: '',
    unread_count: 0,
    created_at: '2026-01-01 10:00:00',
    ...over
  }
}

describe('useWorkspaceStats 租户工作台装载', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    vi.mocked(getTaskStats).mockResolvedValue({
      pending: 3,
      in_progress: 2,
      completed: 10,
      overdue: 1,
      total: 15,
      completion_rate: 0.66,
      avg_handle_seconds: 7200
    })
    vi.mocked(getTenantOverview).mockResolvedValue({
      member: { total: 9, active: 7 },
      task: { pending: 3, in_progress: 2, completed: 10, overdue: 1, total: 15, completion_rate: 0.66 },
      wiki: { spaces: 4, documents: 36, views: 500 },
      storage_used: 12345
    })
    vi.mocked(getWikiStats).mockResolvedValue({
      total_spaces: 4,
      total_nodes: 20,
      total_documents: 36,
      total_views: 500
    })
    vi.mocked(getUserStats).mockResolvedValue({ user_count: 9 })
    vi.mocked(getUnreadCount).mockResolvedValue({ total: 5, by_category: {} })
    vi.mocked(getUnreadNotificationCount).mockResolvedValue({ count: 2 })
    vi.mocked(listConversations).mockResolvedValue([])
  })

  it('load 走聚合接口取成员/Wiki/存储，任务与未读各自独立', async () => {
    const ws = useWorkspaceStats()
    await ws.load()
    expect(ws.task.pending).toBe(3)
    expect(ws.wiki.total_documents).toBe(36)
    expect(ws.wiki.total_spaces).toBe(4)
    expect(ws.memberCount).toBe(9)
    expect(ws.storageUsed).toBe(12345)
    expect(ws.unreadMessages).toBe(5)
    expect(ws.unreadNotifications).toBe(2)
    // 聚合成功时不应再各自回退到独立接口
    expect(getUserStats).not.toHaveBeenCalled()
    expect(getWikiStats).not.toHaveBeenCalled()
  })

  it('聚合接口失败时回退独立接口取成员/Wiki', async () => {
    vi.mocked(getTenantOverview).mockRejectedValue(new Error('overview down'))
    vi.mocked(getUserStats).mockResolvedValue({ user_count: 11 })
    vi.mocked(getWikiStats).mockResolvedValue({
      total_spaces: 6,
      total_nodes: 30,
      total_documents: 42,
      total_views: 800
    })
    const ws = useWorkspaceStats()
    await ws.load()
    expect(getUserStats).toHaveBeenCalledTimes(1)
    expect(getWikiStats).toHaveBeenCalledTimes(1)
    expect(ws.memberCount).toBe(11)
    expect(ws.wiki.total_documents).toBe(42)
    // 兜底路径不含存储维度，保持零值
    expect(ws.storageUsed).toBe(0)
  })

  it('单路失败仅该卡降级为零值，其余照常', async () => {
    vi.mocked(getUnreadCount).mockRejectedValue(new Error('msg down'))
    const ws = useWorkspaceStats()
    await ws.load()
    expect(ws.unreadMessages).toBe(0)
    expect(ws.task.total).toBe(15)
    expect(ws.memberCount).toBe(9)
  })

  it('会话列表为空时补拉；已有会话则不重复请求', async () => {
    const ws = useWorkspaceStats()
    await ws.load()
    expect(listConversations).toHaveBeenCalledTimes(1)

    const s = useChatStore()
    s.conversations = [conv()]
    const ws2 = useWorkspaceStats()
    await ws2.load()
    expect(listConversations).toHaveBeenCalledTimes(1)
  })

  it('聊天补拉失败静默降级，不阻断 load 其余结果', async () => {
    vi.mocked(listConversations).mockRejectedValue(new Error('chat down'))
    const ws = useWorkspaceStats()
    await expect(ws.load()).resolves.toBeUndefined()
    expect(ws.task.total).toBe(15)
    expect(ws.unreadChats).toBe(0)
  })

  it('unreadChats 合计与会话未读联动；mentionChats 仅统计未读且 @我 的会话', async () => {
    vi.mocked(listConversations).mockResolvedValue([
      conv({ id: 'a', unread_count: 4, mentioned: true }),
      conv({ id: 'b', unread_count: 0, mentioned: true }),
      conv({ id: 'c', unread_count: 2 })
    ])
    const ws = useWorkspaceStats()
    await ws.load()
    expect(ws.unreadChats).toBe(6)
    expect(ws.mentionChats).toBe(1)
  })
})
