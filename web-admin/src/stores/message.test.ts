import { describe, it, expect, beforeEach, vi } from 'vitest'
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

  it('reset 清空退出登录后的残留角标', () => {
    const s = useMessageStore()
    s.setUnreadTotal(9)
    s.reset()
    expect(s.total).toBe(0)
    expect(s.hasUnread).toBe(false)
  })
})
