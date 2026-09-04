/**
 * 统一分页类型。
 *
 * 后端不同接口存在三种分页返回包裹（历史原因）：
 *   1. { data: T[], pagination: { page, page_size, total } }
 *   2. { list: T[], total }
 *   3. { items: T[], page, page_size, total }
 * 本文件统一对外类型，并通过 toPageResult 在列表层归一化，供 useTableList 消费。
 */

/** 分页请求参数（后端统一 page / page_size） */
export interface PageQuery {
  page: number
  page_size: number
  /** 扩展过滤字段，由具体列表接口自行扩展 */
  [key: string]: unknown
}

/** 后端嵌套分页元数据 */
export interface PaginationMeta {
  page: number
  page_size: number
  total: number
  total_pages?: number
}

/** 后端嵌套分页响应：{ data, pagination } */
export interface PaginatedResult<T> {
  data: T[]
  pagination: PaginationMeta
}

/** 前端统一列表形态：{ list, total }（useTableList 消费此结构） */
export interface PageResult<T> {
  list: T[]
  total: number
}

/**
 * 将任意后端分页包裹归一化为 PageResult<T>。
 * 兼容 { data, pagination: { total } } / { list, total } / { items, ... } 三种形态。
 */
export function toPageResult<T>(raw: unknown): PageResult<T> {
  if (raw == null) return { list: [], total: 0 }
  const r = raw as Record<string, unknown>
  const rawList = r.data ?? r.list ?? r.items
  const list = Array.isArray(rawList) ? (rawList as T[]) : []
  const pagination = (r.pagination ?? {}) as Partial<PaginationMeta>
  const total = Number(pagination.total ?? r.total ?? list.length) || 0
  return { list, total }
}
