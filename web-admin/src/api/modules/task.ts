/**
 * 任务管理 API
 *
 * 提供个人待办 + 租户团队协作任务的完整接口：
 * - 创建、列表查询（分页/筛选/排序）、详情、更新、完成、重开、删除、统计
 * 所有接口均需登录，后端以数据归属（个人/租户）为准控制可见与可操作范围。
 */
import { get, post, put, del } from '@/api/request'

/** 任务状态 */
export type TaskStatus = 'pending' | 'in_progress' | 'completed'
/** 任务优先级 */
export type TaskPriority = 'low' | 'normal' | 'high' | 'urgent'
/** 可见范围 */
export type TaskVisibility = 'personal' | 'tenant'

/** 任务记录 */
export interface TaskItem {
  id: string
  tenant_id: string
  creator_id: string
  creator_name?: string
  assignee_id: string
  assignee_name?: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  due_date?: string
  tags: string[]
  visibility: TaskVisibility
  started_at?: string
  completed_at?: string
  deleted_at?: string
  created_at: string
  updated_at: string
}

/** 创建任务参数 */
export interface CreateTaskParams {
  title: string
  description?: string
  status?: TaskStatus
  priority?: TaskPriority
  due_date?: string
  tags?: string[]
  visibility?: TaskVisibility
  assignee_id?: string
}

/** 更新任务参数（仅提交需要修改的字段） */
export interface UpdateTaskParams {
  title?: string
  description?: string
  status?: TaskStatus
  priority?: TaskPriority
  due_date?: string
  tags?: string[]
  visibility?: TaskVisibility
  assignee_id?: string
}

/** 任务列表查询参数 */
export interface TaskListParams {
  page?: number
  page_size?: number
  visibility?: TaskVisibility
  status?: TaskStatus
  priority?: TaskPriority
  assignee_id?: string
  keyword?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/** 任务统计 */
export interface TaskStats {
  pending: number
  in_progress: number
  completed: number
  overdue: number
  total: number
}

/** 获取任务列表 */
export function getTaskList(params: TaskListParams) {
  return get('/tasks', params)
}

/** 获取任务详情 */
export function getTask(id: string) {
  return get<TaskItem>(`/tasks/${id}`)
}

/** 创建任务 */
export function createTask(data: CreateTaskParams) {
  return post<TaskItem>('/tasks', data)
}

/** 更新任务 */
export function updateTask(id: string, data: UpdateTaskParams) {
  return put<TaskItem>(`/tasks/${id}`, data)
}

/** 完成任务 */
export function completeTask(id: string) {
  return put<TaskItem>(`/tasks/${id}/complete`)
}

/** 重新打开任务（可指定回到 pending 或 in_progress） */
export function reopenTask(id: string, status?: TaskStatus) {
  return put<TaskItem>(`/tasks/${id}/reopen`, status ? { status } : undefined)
}

/** 删除任务（软删除） */
export function deleteTask(id: string) {
  return del(`/tasks/${id}`)
}

/** 获取任务统计 */
export function getTaskStats(params?: { visibility?: TaskVisibility }) {
  return get<TaskStats>('/tasks/stats', params)
}

/** 回收站列表查询参数 */
export interface TaskTrashParams {
  page?: number
  page_size?: number
  keyword?: string
}

/** 批量操作结果 */
export interface BatchResult {
  affected: number
}

/** 获取回收站（已删除）任务列表 */
export function getTaskTrash(params?: TaskTrashParams) {
  return get('/tasks/deleted', params)
}

/** 恢复已删除任务 */
export function restoreTask(id: string) {
  return put<TaskItem>(`/tasks/${id}/restore`)
}

/** 永久删除任务（不可恢复） */
export function permanentDeleteTask(id: string) {
  return del(`/tasks/${id}/permanent`)
}

/** 批量完成任务 */
export function batchCompleteTasks(ids: string[]) {
  return post<BatchResult>('/tasks/batch/complete', { ids })
}

/** 批量删除任务 */
export function batchDeleteTasks(ids: string[]) {
  return post<BatchResult>('/tasks/batch/delete', { ids })
}
