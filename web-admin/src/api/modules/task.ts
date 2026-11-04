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
/** 重复周期（空表示不重复） */
export type TaskRecurrence = 'daily' | 'weekly' | 'monthly'

/** 任务依赖引用（阻塞关系轻量视图）：id/标题/状态，非 completed 即视为"未完成/阻塞中" */
export interface TaskDependencyRef {
  id: string
  title: string
  status: TaskStatus
}

/** 任务记录 */
export interface TaskItem {
  id: string
  tenant_id: string
  creator_id: string
  creator_name?: string
  assignee_id: string
  assignee_name?: string
  /** 多负责人集合（含主负责人，主负责人排首位） */
  assignees?: string[]
  /** 负责人展示名集合（与 assignees 同序） */
  assignee_names?: string[]
  /** 父任务 ID，为空表示根任务（仅支持单层） */
  parent_id?: string
  /** 前置任务集合（本任务被其阻塞）：含 id/title/status，供列表/看板阻塞徽标与解锁展示 */
  blocked_by?: TaskDependencyRef[]
  /** 后继任务集合（依赖本任务）：blocked_by 的反向视图 */
  blocks?: TaskDependencyRef[]
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  due_date?: string
  remind_before?: string
  tags: string[]
  visibility: TaskVisibility
  recurrence?: string
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
  remind_before?: string
  tags?: string[]
  visibility?: TaskVisibility
  assignee_id?: string
  /** 多负责人集合：给出时以集合为准（主负责人=首位），否则回落 assignee_id；个人任务强制创建人 */
  assignees?: string[]
  recurrence?: TaskRecurrence | ''
  /** 父任务 ID：创建子任务时指定，仅允许挂在根任务下 */
  parent_id?: string
  /** 前置任务 ID 集合：创建时一并建锁（任一不存在/跨租户/不可见/自引用/成环都会被后端拒绝） */
  blocked_by?: string[]
}

/** 更新任务参数（仅提交需要修改的字段） */
export interface UpdateTaskParams {
  title?: string
  description?: string
  status?: TaskStatus
  priority?: TaskPriority
  due_date?: string
  remind_before?: string
  tags?: string[]
  visibility?: TaskVisibility
  assignee_id?: string
  /** 多负责人集合：非空时以集合为准重置负责人（主负责人=首位） */
  assignees?: string[]
  recurrence?: TaskRecurrence | ''
  /** 父任务 ID：传入设为子任务，传空字符串解除父子关系回退为根任务 */
  parent_id?: string
  /** 前置任务 ID 集合：给出（非 null）时以其为准重置本任务前置集合，空数组=清空；不传则不改动 */
  blocked_by?: string[]
}

/** 任务列表查询参数 */
export interface TaskListParams {
  page?: number
  page_size?: number
  visibility?: TaskVisibility
  /** 状态筛选；'overdue' 为"仅逾期"伪状态（后端映射为逾期过滤），用于逾期卡片下钻 */
  status?: TaskStatus | 'overdue'
  priority?: TaskPriority
  assignee_id?: string
  keyword?: string
  /** 按标签精确筛选 */
  tag?: string
  /** 截止时间区间下界（含），支持 YYYY-MM-DD 或 RFC3339 */
  due_from?: string
  /** 截止时间区间上界（含），仅传日期时后端自动补足到当天 23:59:59 */
  due_to?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  /** 按父任务 ID 过滤，拉取某任务的直接子任务 */
  parent_id?: string
  /** 仅返回根任务（parent_id 为空），列表默认视图用于隐藏子任务 */
  root_only?: boolean
}

/** 任务统计 */
export interface TaskStats {
  pending: number
  in_progress: number
  completed: number
  overdue: number
  total: number
  /** 完成率（completed/total），0~1 */
  completion_rate: number
  /** 已完成任务平均处理耗时（秒） */
  avg_handle_seconds: number
}

/** 获取任务列表 */
export function getTaskList(params: TaskListParams) {
  return get('/tasks', params)
}

/** 看板查询参数：与列表共享过滤条件，但不分页、忽略 status（看板按状态分列需含各状态任务） */
export interface TaskBoardParams {
  visibility?: TaskVisibility
  priority?: TaskPriority
  assignee_id?: string
  keyword?: string
  tag?: string
  due_from?: string
  due_to?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  /** 看板默认仅展示根任务，保持分列清爽 */
  root_only?: boolean
}

/** 看板全量加载响应：当前筛选范围内（上限后端 BoardMax=500）的任务、真实总数、是否被截断 */
export interface TaskBoardResult {
  items: TaskItem[]
  total: number
  /** truncated 为真表示匹配总数超过上限，看板仅展示了其中一部分 */
  truncated: boolean
}

/** 获取看板全量任务（一次性拉足当前筛选范围内的任务用于分列） */
export function getTaskBoard(params?: TaskBoardParams) {
  return get<TaskBoardResult>('/tasks/board', params)
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

/** 批量修改状态参数 */
export interface BatchStatusParams {
  ids: string[]
  status: TaskStatus
}

/** 批量指派参数：assignees 非空时以集合为准；否则回落 assignee_id 单个 */
export interface BatchAssignParams {
  ids: string[]
  assignee_id?: string
  assignees?: string[]
}

/** 可指派成员简讯（供负责人选择器，来自 /tasks/assignable，仅需登录） */
export interface AssignableMember {
  id: string
  name: string
}

/** 获取当前租户可指派的存活成员列表（无需 user:list 权限） */
export function getAssignableMembers() {
  return get<AssignableMember[]>('/tasks/assignable')
}

/** 批量修改任务状态 */
export function batchUpdateTaskStatus(data: BatchStatusParams) {
  return post<BatchResult>('/tasks/batch/status', data)
}

/** 批量指派负责人 */
export function batchAssignTasks(data: BatchAssignParams) {
  return post<BatchResult>('/tasks/batch/assign', data)
}
