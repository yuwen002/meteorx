/**
 * 租户工作台统计装载 —— 首页（非超管视角）各卡片的数据来源
 *
 * 复用任务/Wiki/成员/站内信/通知各自的租户级接口，Promise.allSettled 并发拉取：
 * 任一接口失败仅对应卡片降级为零值，不阻断整屏（避免一个模块故障拖垮首页）。
 * 聊天未读不在这里拉取，直接派生自 chat store 的现值（WS 实时维护）。
 */
import { computed, reactive, ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { getTaskStats, type TaskStats } from '@/api/modules/task'
import { getWikiStats, type WikiStats } from '@/api/modules/wiki'
import { getUserStats } from '@/api/modules/user'
import { getUnreadCount } from '@/api/modules/message'
import { getUnreadNotificationCount } from '@/api/modules/wiki'

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

export function useWorkspaceStats() {
  const chatStore = useChatStore()

  // reactive 收纳：模板里 ws.task.total 等可直接读取（内部 ref 自动解包）
  const state = reactive({
    task: ref<TaskStats>({ ...EMPTY_TASK }),
    wiki: ref<WikiStats>({ ...EMPTY_WIKI }),
    memberCount: ref(0),
    unreadMessages: ref(0),
    unreadNotifications: ref(0),
    // 聊天未读与 @我 徽标：chat store 由 WS 与 loadConversations 维护，这里只读
    unreadChats: computed(() => chatStore.totalUnread),
    mentionChats: computed(
      () => chatStore.conversations.filter((c) => c.mentioned && c.unread_count > 0).length
    )
  })

  /** 并发装载五路统计；settled 逐项取值，失败项保持零值 */
  function load() {
    return Promise.allSettled([
      getTaskStats().then((v) => {
        if (v) state.task = v
      }),
      getWikiStats().then((v) => {
        if (v) state.wiki = v
      }),
      getUserStats().then((v) => {
        if (v) state.memberCount = v.user_count ?? 0
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
