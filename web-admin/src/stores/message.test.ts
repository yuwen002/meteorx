import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// 隔离网络层：store 只依赖未读数接口
vi.mock('@/api/modules/message', () => ({ getUnreadCount: vi.fn() }))

import { getUnreadCount } from '@/api/modules/message'
import { useMessageStore } from './message'

describe('message store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    // 用例里可能开了假时钟，结束必须复位，否则后续用例的定时器全部卡住
    vi.useRealTimers()
  })

  it('refreshUnread 以服务端未读数覆盖本地状态', async () => {
    vi.mocked(getUnreadCount).mockResolvedValue({ total: 4, by_category: { task: 3, plan: 1 } })
    const s = useMessageStore()
    await s.refreshUnread()
    expect(s.total).toBe(4)
    expect(s.hasUnread).toBe(true)
    expect(s.categoryUnread('task')).toBe(3)
    expect(s.categoryUnread('system')).toBe(0)
  })

  it('refreshUnread 失败时保留上一次读数', async () => {
    vi.mocked(getUnreadCount).mockResolvedValueOnce({ total: 2, by_category: { task: 2 } })
    const s = useMessageStore()
    await s.refreshUnread()

    vi.mocked(getUnreadCount).mockRejectedValueOnce(new Error('network'))
    await s.refreshUnread()
    expect(s.total).toBe(2)
    expect(s.loading).toBe(false)
  })

  it('setUnreadTotal 覆盖总数并忽略非法值', () => {
    const s = useMessageStore()
    s.setUnreadTotal(7)
    expect(s.total).toBe(7)

    s.setUnreadTotal(Number.NaN)
    expect(s.total).toBe(7)

    s.setUnreadTotal(-3)
    expect(s.total).toBe(0)
  })

  it('decrementUnread 扣减总数与对应分类，且不会变负', () => {
    const s = useMessageStore()
    s.setUnreadTotal(1)
    vi.mocked(getUnreadCount).mockResolvedValue({ total: 1, by_category: { task: 1 } })

    return s.refreshUnread().then(() => {
      s.decrementUnread('task')
      expect(s.total).toBe(0)
      expect(s.categoryUnread('task')).toBe(0)

      s.decrementUnread('task')
      expect(s.total).toBe(0)
    })
  })

  it('clearUnread 传分类只清该分类，不传清空全部', () => {
    const s = useMessageStore()
    s.byCategory = { task: 2, plan: 1 }
    s.setUnreadTotal(3)

    s.clearUnread('task')
    expect(s.total).toBe(1)
    expect(s.categoryUnread('task')).toBe(0)
    expect(s.categoryUnread('plan')).toBe(1)

    s.clearUnread()
    expect(s.total).toBe(0)
    expect(s.byCategory).toEqual({})
  })

  it('clearUnread 传服务端 updated 时按它扣减，覆盖过期的分类明细', () => {
    const s = useMessageStore()
    // WS 只推总数，分类明细停留旧值：task 实际已涨到 3 条仍未同步
    s.byCategory = { task: 1, plan: 1 }
    s.setUnreadTotal(5)

    s.clearUnread('task', 3)
    expect(s.total).toBe(2)
    expect(s.categoryUnread('task')).toBe(0)
    expect(s.categoryUnread('plan')).toBe(1)
  })

  it('receiveUnreadPush 先覆盖总数，随后补拉分类明细', async () => {
    vi.useFakeTimers()
    vi.mocked(getUnreadCount).mockResolvedValue({ total: 5, by_category: { task: 5 } })
    const s = useMessageStore()

    s.receiveUnreadPush(5)
    expect(s.total).toBe(5)
    expect(s.categoryUnread('task')).toBe(0)
    expect(getUnreadCount).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(300)
    expect(getUnreadCount).toHaveBeenCalledTimes(1)
    expect(s.categoryUnread('task')).toBe(5)
  })

  it('receiveUnreadPush 忽略非法值，不触发补拉', async () => {
    vi.useFakeTimers()
    const s = useMessageStore()

    s.receiveUnreadPush(Number.NaN)
    await vi.advanceTimersByTimeAsync(300)
    expect(s.total).toBe(0)
    expect(getUnreadCount).not.toHaveBeenCalled()
  })

  it('refreshUnreadSoon 把短时间内的多次校正合并为一次请求', async () => {
    vi.useFakeTimers()
    vi.mocked(getUnreadCount).mockResolvedValue({ total: 0, by_category: {} })
    const s = useMessageStore()

    s.refreshUnreadSoon()
    s.refreshUnreadSoon()
    s.receiveUnreadPush(2)
    s.refreshUnreadSoon()

    await vi.advanceTimersByTimeAsync(300)
    expect(getUnreadCount).toHaveBeenCalledTimes(1)
  })

  it('reset 取消排队中的校正请求', async () => {
    vi.useFakeTimers()
    const s = useMessageStore()

    s.refreshUnreadSoon()
    s.reset()

    await vi.advanceTimersByTimeAsync(1000)
    expect(getUnreadCount).not.toHaveBeenCalled()
  })

  it('reset 清空退出登录后的残留角标', () => {
    const s = useMessageStore()
    s.setUnreadTotal(9)
    s.byCategory = { task: 9 }
    s.reset()
    expect(s.total).toBe(0)
    expect(s.byCategory).toEqual({})
    expect(s.hasUnread).toBe(false)
  })
})
