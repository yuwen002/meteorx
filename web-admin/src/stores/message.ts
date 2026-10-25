import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getUnreadCount } from '@/api/modules/message'

/**
 * 站内消息未读状态（顶栏铃铛角标 + 侧边栏菜单角标共用）。
 *
 * 服务端才是未读数的唯一真相：本地只做「乐观扣减」把交互做顺，
 * 下一次 refreshUnread（进入收件箱页、收到 WS 推送、重新登录）以服务端为准覆盖。
 */
export const useMessageStore = defineStore('message', () => {
  const total = ref(0)
  const byCategory = ref<Record<string, number>>({})
  const loading = ref(false)

  /** 排队中的合并校正定时器；null 表示当前无待执行请求 */
  let reconcileTimer: ReturnType<typeof setTimeout> | null = null

  const hasUnread = computed(() => total.value > 0)

  /** 取某个分类的未读数，未出现过的分类为 0 */
  function categoryUnread(category: string): number {
    return byCategory.value[category] ?? 0
  }

  /** 拉取服务端未读数（失败静默：角标缺失不应打断用户操作） */
  async function refreshUnread() {
    loading.value = true
    try {
      const res = await getUnreadCount()
      total.value = Math.max(0, res?.total ?? 0)
      byCategory.value = res?.by_category ?? {}
    } catch {
      // 静默失败，保留上一次读数
    } finally {
      loading.value = false
    }
  }

  /** WS 推送的角标直接覆盖总数（payload 不含分类明细） */
  function setUnreadTotal(count: number) {
    if (!Number.isFinite(count)) return
    total.value = Math.max(0, count)
  }

  /**
   * 稍后校正未读数，窗口内多次调用合并为一次请求。
   * 两类场景：本地乐观更新后与服务端对账；WS 只推总数时补齐分类明细。
   */
  function refreshUnreadSoon(delay = 300) {
    if (reconcileTimer !== null) return
    reconcileTimer = setTimeout(() => {
      reconcileTimer = null
      void refreshUnread()
    }, delay)
  }

  /** 收到 WS 未读数推送：先覆盖总数保证即时，再合并补拉一次分类明细 */
  function receiveUnreadPush(count: number) {
    if (!Number.isFinite(count)) return
    setUnreadTotal(count)
    refreshUnreadSoon()
  }

  /** 标记单条已读后的乐观扣减 */
  function decrementUnread(category: string) {
    total.value = Math.max(0, total.value - 1)
    if (category && (byCategory.value[category] ?? 0) > 0) {
      byCategory.value = { ...byCategory.value, [category]: (byCategory.value[category] ?? 0) - 1 }
    }
  }

  /**
   * 标记全部已读后的角标更新；传分类只清该分类。
   *
   * cleared 传服务端返回的 updated：WS 只推总数，分类明细可能已被留在旧值，
   * 用服务端的结果扣减比本地 byCategory 更准；剩余偏差由 refreshUnreadSoon 兜底。
   */
  function clearUnread(category?: string, cleared?: number) {
    if (!category) {
      total.value = 0
      byCategory.value = {}
      return
    }
    const removed = Math.max(0, cleared ?? byCategory.value[category] ?? 0)
    total.value = Math.max(0, total.value - removed)
    byCategory.value = { ...byCategory.value, [category]: 0 }
  }

  /** 退出登录时重置，避免下一个账号看到上一个账号的角标 */
  function reset() {
    total.value = 0
    byCategory.value = {}
    // 退出后不能留下排队中的校正：它会在新账号登录后带着旧会话的意图打请求
    if (reconcileTimer !== null) {
      clearTimeout(reconcileTimer)
      reconcileTimer = null
    }
  }

  return {
    total,
    byCategory,
    loading,
    hasUnread,
    categoryUnread,
    refreshUnread,
    refreshUnreadSoon,
    receiveUnreadPush,
    setUnreadTotal,
    decrementUnread,
    clearUnread,
    reset
  }
})
