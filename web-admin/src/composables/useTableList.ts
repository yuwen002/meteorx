/**
 * useTableList —— 列表页通用状态与加载逻辑。
 *
 * 封装 loading / list / total / page / pageSize / 查询条件，
 * 并暴露 search / reset / 分页切换等常用操作，减少列表页重复代码。
 *
 * 用法：
 * ```ts
 * const { list, total, page, pageSize, loading, query,
 *         load, search, reset, handlePageChange, handleSizeChange } = useTableList({
 *   fetchList: async (params) => toPageResult<UserItem>(await getUserList(params)),
 *   initialQuery: { keyword: '', status: undefined },
 * })
 * ```
 */
import { reactive, ref } from 'vue'
import type { PageQuery, PageResult } from '@/types/pagination'

export interface UseTableListOptions<T, Q extends object> {
  /** 列表请求函数：入参为 page/page_size 与过滤条件，返回前端统一分页结构 */
  fetchList: (params: PageQuery & Q) => Promise<PageResult<T>> | PageResult<T>
  /** 初始化过滤条件（reset 后回到该值） */
  initialQuery?: Q
  /** 默认每页条数，默认 10 */
  defaultPageSize?: number
  /** 是否立即加载，默认 true */
  immediate?: boolean
}

export function useTableList<T, Q extends object = Record<string, unknown>>(
  options: UseTableListOptions<T, Q>
) {
  const list = ref<T[]>([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(options.defaultPageSize ?? 10)
  const loading = ref(false)
  const query = reactive<Q>({ ...(options.initialQuery ?? ({} as Q)) })

  async function load(): Promise<void> {
    loading.value = true
    try {
      const params = { page: page.value, page_size: pageSize.value, ...query } as PageQuery & Q
      const res = await options.fetchList(params)
      list.value = res?.list ?? []
      total.value = res?.total ?? 0
    } catch {
      list.value = []
      total.value = 0
    } finally {
      loading.value = false
    }
  }

  /** 触发搜索：回到第一页并加载 */
  function search(): void {
    page.value = 1
    void load()
  }

  /** 重置查询条件到初始值并重新加载 */
  function reset(): void {
    const initial = options.initialQuery
    if (initial) {
      Object.assign(query, initial)
    }
    page.value = 1
    void load()
  }

  /** el-pagination current-change */
  function handlePageChange(p: number): void {
    page.value = p
    void load()
  }

  /** el-pagination size-change */
  function handleSizeChange(size: number): void {
    pageSize.value = size
    page.value = 1
    void load()
  }

  /** 重新加载（通常操作完成后刷新） */
  function reload(): void {
    void load()
  }

  if (options.immediate !== false) {
    void load()
  }

  return {
    list,
    total,
    page,
    pageSize,
    loading,
    query,
    load,
    search,
    reset,
    handlePageChange,
    handleSizeChange,
    reload
  }
}
