/**
 * 租户工作台统计装载 —— 首页（非超管视角）各卡片的数据来源
 *
 * 慢变的租户级事实（成员/Wiki/存储）统一走 tenant-overview 聚合接口一次拉取，
 * 聚合失败时回退到各自独立接口（getWikiStats/getUserStats）作兜底；
 * "我的任务"是随用户与可见范围而变的个人视图，仍走 /tasks/stats（含平均处理时长）；
 * 未读站内信/通知按用户实时维度独立调用。聊天未读派生自 chat store 现值（WS 实时维护）。
 * 任一维度失败仅相关卡片降级为零值，不阻断整屏。
 */
import { computed, reactive, ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { getTaskStats, type TaskStats } from '@/api/modules/task'
import { getWikiStats, type WikiStats, getUnreadNotificationCount } from '@/api/modules/wiki'
import { getUserStats } from '@/api/modules/user'
import { getUnreadCount } from '@/api/modules/message'
import { getTenantOverview, type TenantOverview } from '@/api/modules/dashboard'

/** 单卡失败时的零值兜底（字段与 TaskStats/WikiStats 对齐） */
const EMPTY_TASK: TaskStats = {
  pending: 0,
  in_progress: 0,
  completed: 0,
  overdue: 0,
  total: 0,
  completion_rate: 0,
  avg_handle_seconds: 0
}
const EMPTY_WIKI: WikiStats = { total_spaces: 0, total_nodes: 0, total_documents: 0, total_views: 0 }

/** 团队级任务分布零值兜底（字段与 TenantOverview['task'] 对齐） */
type TenantTask = TenantOverview['task']
const EMPTY_TENANT_TASK: TenantTask = {
  pending: 0,
  in_progress: 0,
  completed: 0,
  overdue: 0,
  total: 0,
  completion_rate: 0
}

export function useWorkspaceStats() {
  const chatStore = useChatStore()

  // reactive 收纳：模板里 ws.task.total 等可直接读取（内部 ref 自动解包）
  const state = reactive({
    task: ref<TaskStats>({ ...EMPTY_TASK }),
    // 团队级任务分布：随 tenant-overview 一并返回，供“团队任务概况”卡消费
    tenantTask: ref<TenantTask>({ ...EMPTY_TENANT_TASK }),
    wiki: ref<WikiStats>({ ...EMPTY_WIKI }),
    memberCount: ref(0),
    storageUsed: ref(0),
    unreadMessages: ref(0),
    unreadNotifications: ref(0),
    // 聊天未读与 @我 徽标：chat store 由 WS 与 loadConversations 维护，这里只读
    unreadChats: computed(() => chatStore.totalUnread),
    mentionChats: computed(
      () => chatStore.conversations.filter((c) => c.mentioned && c.unread_count > 0).length
    )
  })

  /** 成员/Wiki/存储：优先聚合接口，失败回退独立接口 */
  async function loadTenantFacts() {
    try {
      const ov = await getTenantOverview()
      if (ov) {
        state.memberCount = ov.member?.total ?? 0
        state.wiki = {
          total_spaces: ov.wiki?.spaces ?? 0,
          total_nodes: 0,
          total_documents: ov.wiki?.documents ?? 0,
          total_views: ov.wiki?.views ?? 0
        }
        state.storageUsed = ov.storage_used ?? 0
        state.tenantTask = ov.task ?? { ...EMPTY_TENANT_TASK }
        return
      }
    } catch {
      /* 聚合接口不可用，落到独立接口兜底 */
    }
    await Promise.allSettled([
      getUserStats().then((v) => {
        if (v) state.memberCount = v.user_count ?? 0
      }),
      getWikiStats().then((v) => {
        if (v) state.wiki = v
      })
    ])
  }

  /** 并发装载各维度；settled 逐项取值，失败项保持零值 */
  function load() {
    return Promise.allSettled([
      loadTenantFacts(),
      getTaskStats().then((v) => {
        if (v) state.task = v
      }),
      getUnreadCount().then((v) => {
        if (v) state.unreadMessages = v.total ?? 0
      }),
      getUnreadNotificationCount().then((v) => {
        if (v) state.unreadNotifications = v.count ?? 0
      })
    ]).then(async () => {
      // 会话列表若尚未加载则补拉（聊天未读依赖它）；失败静默，卡片降级为 0
      if (!chatStore.conversations.length) {
        try {
          await chatStore.loadConversations()
        } catch {
          /* 聊天不可用不影响工作台其余卡片 */
        }
      }
    })
  }

  // 直接返回 reactive 本身（而非展开快照），保留响应式解包；附带 load 方法
  return Object.assign(state, { load })
}
