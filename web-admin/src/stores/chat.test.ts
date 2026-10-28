import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// 隔离网络层：mock chat API 全部导出，store 逻辑聚焦状态流转
vi.mock('@/api/modules/chat', () => ({
  openDirectConversation: vi.fn(),
  createGroupConversation: vi.fn(),
  listConversations: vi.fn(),
  sendChatMessage: vi.fn(),
  listChatMessages: vi.fn(),
  markConversationRead: vi.fn(),
  recallChatMessage: vi.fn(),
  sendTyping: vi.fn(),
  getPresence: vi.fn(),
  listConversationMembers: vi.fn(),
  addConversationMembers: vi.fn(),
  removeConversationMember: vi.fn(),
  leaveGroup: vi.fn(),
  renameGroup: vi.fn()
}))

import {
  createGroupConversation,
  listConversations,
  addConversationMembers,
  removeConversationMember,
  leaveGroup,
  renameGroup,
  listConversationMembers
} from '@/api/modules/chat'
import type { ConversationItem, MemberItem } from '@/api/modules/chat'
import { useChatStore } from './chat'

function groupConv(over: Partial<ConversationItem> = {}): ConversationItem {
  return {
    id: 'g1',
    type: 'group',
    peer_user_id: '',
    peer_name: '研发群',
    last_message: '',
    unread_count: 0,
    member_count: 3,
    my_role: 'owner',
    created_at: '2026-01-01 10:00:00',
    ...over
  }
}

describe('chat store 群聊', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    // resetAllMocks 会清掉实现：mock 工厂未预置实现，每个用例按需 mockResolvedValue
    vi.resetAllMocks()
  })

  it('createGroup 成功后前置插入会话列表', async () => {
    vi.mocked(createGroupConversation).mockResolvedValue(groupConv())
    const s = useChatStore()
    const conv = await s.createGroup('研发群', ['u2', 'u3'])
    expect(createGroupConversation).toHaveBeenCalledWith('研发群', ['u2', 'u3'])
    expect(conv.id).toBe('g1')
    expect(s.conversations[0].id).toBe('g1')
    expect(s.conversations[0].type).toBe('group')
  })

  it('createGroup 对已存在会话原位替换而非重复插入', async () => {
    const s = useChatStore()
    s.conversations = [groupConv({ peer_name: '旧名' })]
    vi.mocked(createGroupConversation).mockResolvedValue(groupConv({ peer_name: '新名' }))
    await s.createGroup('新名', ['u2'])
    expect(s.conversations).toHaveLength(1)
    expect(s.conversations[0].peer_name).toBe('新名')
  })

  it('addMembers 调用后端后刷新会话列表以更新成员数', async () => {
    vi.mocked(addConversationMembers).mockResolvedValue({ ok: true })
    vi.mocked(listConversations).mockResolvedValue([groupConv({ member_count: 5 })])
    const s = useChatStore()
    await s.addMembers('g1', ['u4', 'u5'])
    expect(addConversationMembers).toHaveBeenCalledWith('g1', ['u4', 'u5'])
    expect(listConversations).toHaveBeenCalledTimes(1)
    expect(s.conversations[0].member_count).toBe(5)
  })

  it('removeMember 调用后端后刷新会话列表', async () => {
    vi.mocked(removeConversationMember).mockResolvedValue({ ok: true })
    vi.mocked(listConversations).mockResolvedValue([groupConv({ member_count: 2 })])
    const s = useChatStore()
    await s.removeMember('g1', 'u3')
    expect(removeConversationMember).toHaveBeenCalledWith('g1', 'u3')
    expect(s.conversations[0].member_count).toBe(2)
  })

  it('leaveConversation 移出列表并清空当前选中', async () => {
    vi.mocked(leaveGroup).mockResolvedValue({ ok: true })
    vi.mocked(listConversations).mockResolvedValue([])
    const s = useChatStore()
    s.conversations = [groupConv()]
    s.activeId = 'g1'
    await s.leaveConversation('g1')
    expect(leaveGroup).toHaveBeenCalledWith('g1')
    expect(s.activeId).toBe('')
    expect(s.conversations).toHaveLength(0)
  })

  it('rename 成功后同步本地会话展示名', async () => {
    vi.mocked(renameGroup).mockResolvedValue({ ok: true })
    const s = useChatStore()
    s.conversations = [groupConv({ peer_name: '旧名' })]
    await s.rename('g1', '新名')
    expect(renameGroup).toHaveBeenCalledWith('g1', '新名')
    expect(s.conversations[0].peer_name).toBe('新名')
  })

  it('loadMembers 返回成员列表', async () => {
    const members: MemberItem[] = [
      { user_id: 'u1', name: '甲', role: 'owner', joined_at: '2026-01-01 10:00:00' },
      { user_id: 'u2', name: '乙', role: 'member', joined_at: '2026-01-01 10:00:00' }
    ]
    vi.mocked(listConversationMembers).mockResolvedValue(members)
    const s = useChatStore()
    const got = await s.loadMembers('g1')
    expect(got).toHaveLength(2)
    expect(got[0].role).toBe('owner')
  })

  it('receiveConversationPush 原位刷新元信息但保留本地已清零的未读', () => {
    const s = useChatStore()
    s.conversations = [groupConv({ member_count: 3, unread_count: 0 })]
    s.receiveConversationPush(groupConv({ member_count: 6, unread_count: 9 }))
    expect(s.conversations[0].member_count).toBe(6)
    // 正看着该会话时不应被推送重新点亮未读
    expect(s.conversations[0].unread_count).toBe(0)
  })

  it('receiveConversationPush 会话不在列表时回源刷新', async () => {
    vi.mocked(listConversations).mockResolvedValue([groupConv()])
    const s = useChatStore()
    s.receiveConversationPush(groupConv({ id: 'g9' }))
    // 触发一次列表回拉（异步）
    await Promise.resolve()
    expect(listConversations).toHaveBeenCalledTimes(1)
  })
})
