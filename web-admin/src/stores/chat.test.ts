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
  getConversationReadStatus: vi.fn(),
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
  listConversationMembers,
  listChatMessages,
  markConversationRead,
  getConversationReadStatus,
  getPresence,
  sendChatMessage
} from '@/api/modules/chat'
import type { ChatMessageItem, ConversationItem, MemberItem } from '@/api/modules/chat'
import { useUserStore } from '@/stores/user'
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

// ---- 历史消息分页加载 ----

function msg(id: string, convId = 'g1'): ChatMessageItem {
  return {
    id,
    conversation_id: convId,
    sender_id: 'u1',
    sender_name: '甲',
    content: `m-${id}`,
    type: 'text',
    created_at: '2026-01-01 10:00:00'
  }
}

/** 构造后端分页信封（页内升序 + pagination.total） */
function pageRes(ids: string[], page: number, total: number) {
  return {
    data: ids.map((id) => msg(id)),
    pagination: { page, page_size: 50, total }
  }
}

describe('chat store 历史消息分页', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    vi.mocked(markConversationRead).mockResolvedValue({ ok: true })
    vi.mocked(getPresence).mockResolvedValue({ online_user_ids: [] })
  })

  it('selectConversation 载入首页并置分页状态', async () => {
    const ids = Array.from({ length: 50 }, (_, i) => `p1-${i}`)
    vi.mocked(listChatMessages).mockResolvedValue(pageRes(ids, 1, 120) as never)
    const s = useChatStore()
    await s.selectConversation('g1')
    expect(listChatMessages).toHaveBeenCalledWith('g1', { page: 1, page_size: 50 })
    expect(s.messages).toHaveLength(50)
    expect(s.messagePage).toBe(1)
    expect(s.hasMoreMessages).toBe(true)
  })

  it('首页未取满一页判定无更多', async () => {
    vi.mocked(listChatMessages).mockResolvedValue(pageRes(['a', 'b'], 1, 2) as never)
    const s = useChatStore()
    await s.selectConversation('g1')
    expect(s.hasMoreMessages).toBe(false)
  })

  it('loadOlderMessages 前插更早页并推进页码', async () => {
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(Array.from({ length: 50 }, (_, i) => `p1-${i}`), 1, 120) as never)
    const s = useChatStore()
    await s.selectConversation('g1')
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(Array.from({ length: 50 }, (_, i) => `p2-${i}`), 2, 120) as never)
    const added = await s.loadOlderMessages()
    expect(added).toBe(50)
    expect(s.messagePage).toBe(2)
    expect(s.messages).toHaveLength(100)
    expect(s.messages[0].id).toBe('p2-0') // 更早页在前
    expect(s.hasMoreMessages).toBe(true)
  })

  it('loadOlderMessages 按 id 去重防偏移漂移重复', async () => {
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(Array.from({ length: 50 }, (_, i) => `p1-${i}`), 1, 120) as never)
    const s = useChatStore()
    await s.selectConversation('g1')
    // 第二页含 1 条与首页重叠的消息（新消息插入导致后端偏移漂移）
    const overlap = ['p1-49', ...Array.from({ length: 49 }, (_, i) => `p2-${i}`)]
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(overlap, 2, 120) as never)
    const added = await s.loadOlderMessages()
    expect(added).toBe(49)
    expect(s.messages).toHaveLength(99)
    expect(s.messages.filter((m) => m.id === 'p1-49')).toHaveLength(1)
  })

  it('末页加载后 hasMoreMessages 归位且不再请求', async () => {
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(Array.from({ length: 50 }, (_, i) => `p1-${i}`), 1, 60) as never)
    const s = useChatStore()
    await s.selectConversation('g1')
    expect(s.hasMoreMessages).toBe(true)
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(Array.from({ length: 10 }, (_, i) => `p2-${i}`), 2, 60) as never)
    await s.loadOlderMessages()
    expect(s.hasMoreMessages).toBe(false)
    // 无更多时再次调用不发起请求
    vi.mocked(listChatMessages).mockClear()
    expect(await s.loadOlderMessages()).toBe(0)
    expect(listChatMessages).not.toHaveBeenCalled()
  })

  it('加载失败不推进页码，可重试', async () => {
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(Array.from({ length: 50 }, (_, i) => `p1-${i}`), 1, 120) as never)
    const s = useChatStore()
    await s.selectConversation('g1')
    vi.mocked(listChatMessages).mockRejectedValueOnce(new Error('network'))
    expect(await s.loadOlderMessages()).toBe(0)
    expect(s.messagePage).toBe(1)
    expect(s.loadingOlder).toBe(false)
    // 重试成功
    vi.mocked(listChatMessages).mockResolvedValueOnce(pageRes(Array.from({ length: 50 }, (_, i) => `p2-${i}`), 2, 120) as never)
    expect(await s.loadOlderMessages()).toBe(50)
    expect(s.messagePage).toBe(2)
  })

  it('未选中会话时 loadOlderMessages 直接返回 0', async () => {
    const s = useChatStore()
    expect(await s.loadOlderMessages()).toBe(0)
    expect(listChatMessages).not.toHaveBeenCalled()
  })
})

// ---- 已读回执 ----

function readPage(ids: string[], total: number) {
  return { data: ids.map((id) => msg(id)), pagination: { page: 1, page_size: 50, total } }
}

describe('chat store 已读回执', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    vi.mocked(markConversationRead).mockResolvedValue({ ok: true })
    vi.mocked(getPresence).mockResolvedValue({ online_user_ids: [] })
    vi.mocked(listChatMessages).mockResolvedValue(readPage([], 0) as never)
  })

  /** 等待 selectConversation 内部 fire-and-forget 的 read-status 回拉完成 */
  async function flush() {
    await Promise.resolve()
    await Promise.resolve()
  }

  it('选中会话后拉取其他成员水位并按水位计数', async () => {
    vi.mocked(getConversationReadStatus).mockResolvedValue([
      { user_id: 'u2', name: '乙', last_read_at: '2026-01-01 10:05:00' },
      { user_id: 'u3', name: '丙' } // 从未读
    ])
    const s = useChatStore()
    await s.selectConversation('g1')
    await flush()
    expect(getConversationReadStatus).toHaveBeenCalledWith('g1')
    expect(s.readCountAt('2026-01-01 10:00:00')).toBe(1) // 晚于消息时间的水位计入已读
    expect(s.readCountAt('2026-01-01 10:06:00')).toBe(0) // 水位早于消息则未读
  })

  it('read-status 空列表信封对象经 asArray 归一化不报错', async () => {
    // 后端无其他成员已读时可能返回 { data: null } 回退信封
    vi.mocked(getConversationReadStatus).mockResolvedValue({ data: null } as never)
    const s = useChatStore()
    await s.selectConversation('g1')
    await flush()
    expect(s.readCountAt('2026-01-01 10:00:00')).toBe(0)
  })

  it('receiveReadPush 仅更新当前会话且水位只前进不后退', async () => {
    vi.mocked(getConversationReadStatus).mockResolvedValue([
      { user_id: 'u2', name: '乙', last_read_at: '2026-01-01 10:05:00' }
    ])
    const s = useChatStore()
    await s.selectConversation('g1')
    await flush()
    // 更新为更晚水位 → 生效
    s.receiveReadPush({ conversation_id: 'g1', user_id: 'u2', read_at: '2026-01-01 10:08:00' })
    expect(s.readCountAt('2026-01-01 10:07:00')).toBe(1)
    // 更旧水位的乱序推送 → 丢弃
    s.receiveReadPush({ conversation_id: 'g1', user_id: 'u2', read_at: '2026-01-01 10:01:00' })
    expect(s.readStates.u2).toBe('2026-01-01 10:08:00')
    // 非当前会话的推送 → 忽略
    s.receiveReadPush({ conversation_id: 'g9', user_id: 'u5', read_at: '2026-01-01 11:00:00' })
    expect(s.readStates.u5).toBeUndefined()
  })

  it('切换会话与登出重置水位', async () => {
    vi.mocked(getConversationReadStatus).mockResolvedValue([
      { user_id: 'u2', name: '乙', last_read_at: '2026-01-01 10:05:00' }
    ])
    const s = useChatStore()
    await s.selectConversation('g1')
    await flush()
    expect(s.readCountAt('2026-01-01 10:00:00')).toBe(1)
    // 切到新会话：旧水位清空
    vi.mocked(getConversationReadStatus).mockResolvedValue([])
    await s.selectConversation('g2')
    expect(s.readStates.u2).toBeUndefined()
    // 登出重置
    s.reset()
    expect(Object.keys(s.readStates)).toHaveLength(0)
  })
})

// ---- @提及 ----

describe('chat store @提及', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    vi.mocked(markConversationRead).mockResolvedValue({ ok: true })
    vi.mocked(getPresence).mockResolvedValue({ online_user_ids: [] })
    vi.mocked(listChatMessages).mockResolvedValue(readPage([], 0) as never)
    vi.mocked(getConversationReadStatus).mockResolvedValue([])
    // 当前登录用户 u1：myId 取自 user store
    const us = useUserStore()
    us.userInfo = { id: 'u1', username: 'u1' }
  })

  it('mentionsMe 仅当 mention_ids 含当前用户时为真', () => {
    const s = useChatStore()
    expect(s.mentionsMe({ mention_ids: 'u2,u1' })).toBe(true)
    expect(s.mentionsMe({ mention_ids: 'u1' })).toBe(true)
    expect(s.mentionsMe({ mention_ids: 'u2, u3' })).toBe(false) // 带空格分隔也不会误判
    expect(s.mentionsMe({ mention_ids: '' })).toBe(false)
    expect(s.mentionsMe({})).toBe(false)
  })

  it('未登录时 mentionsMe 恒为假', () => {
    useUserStore().userInfo = null
    const s = useChatStore()
    expect(s.mentionsMe({ mention_ids: 'u1' })).toBe(false)
  })

  it('send 将 mentionIds 透传给发送接口', async () => {
    const s = useChatStore()
    s.activeId = 'g1'
    vi.mocked(sendChatMessage).mockResolvedValue(msg('m1'))
    const ok = await s.send('hi @乙', 'u2')
    expect(ok).toBe(true)
    expect(sendChatMessage).toHaveBeenCalledWith('g1', 'hi @乙', 'u2', 'text')
  })

  it('receiveChatPush 非当前会话被 @ 时打 mentioned 标', () => {
    const s = useChatStore()
    s.activeId = 'g2'
    s.conversations = [groupConv({ id: 'g1' })]
    s.receiveChatPush({ ...msg('m1', 'g1'), mention_ids: 'u1' })
    expect(s.conversations[0].mentioned).toBe(true)
    expect(s.conversations[0].unread_count).toBe(1)
  })

  it('receiveChatPush 未被 @ 时不打标', () => {
    const s = useChatStore()
    s.activeId = 'g2'
    s.conversations = [groupConv({ id: 'g1' })]
    s.receiveChatPush({ ...msg('m1', 'g1'), mention_ids: 'u9' })
    expect(s.conversations[0].mentioned).toBeFalsy()
  })

  it('selectConversation 选中后清除会话的 mentioned 标', async () => {
    const s = useChatStore()
    s.conversations = [groupConv({ id: 'g1', mentioned: true })]
    await s.selectConversation('g1')
    expect(s.conversations[0].mentioned).toBe(false)
  })
})

// ---- 图片/文件消息 ----

describe('chat store 媒体消息', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    vi.mocked(markConversationRead).mockResolvedValue({ ok: true })
  })

  const mediaContent = '{"file":"a.png","name":"相册.png","size":9}'

  it('send 透传 msgType 并同步会话摘要类型', async () => {
    const s = useChatStore()
    s.activeId = 'g1'
    s.conversations = [groupConv()]
    vi.mocked(sendChatMessage).mockResolvedValue({
      ...msg('m9'),
      type: 'image',
      content: mediaContent
    })
    const ok = await s.send(mediaContent, '', 'image')
    expect(ok).toBe(true)
    expect(sendChatMessage).toHaveBeenCalledWith('g1', mediaContent, '', 'image')
    expect(s.conversations[0].last_message).toBe(mediaContent)
    expect(s.conversations[0].last_message_type).toBe('image')
  })

  it('receiveChatPush 当前与非当前会话均同步 last_message_type', () => {
    const s = useChatStore()
    s.conversations = [groupConv({ id: 'g1' }), groupConv({ id: 'g2', peer_name: '另一群' })]
    // 当前会话
    s.activeId = 'g1'
    s.receiveChatPush({ ...msg('m1', 'g1'), type: 'file', content: mediaContent })
    expect(s.conversations[0].last_message_type).toBe('file')
    // 非当前会话（累加未读分支）
    s.activeId = 'g1'
    s.receiveChatPush({ ...msg('m2', 'g2'), type: 'image', content: mediaContent })
    expect(s.conversations[1].last_message_type).toBe('image')
    expect(s.conversations[1].unread_count).toBe(1)
  })
})
