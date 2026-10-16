/**
 * 任务 AI 辅助 API
 *
 * 可选叠加层：自然语言建任务、优先级/标签建议、描述扩写、任务摘要，以及能力探测。
 * 均挂在受保护的 /tasks/ai/* 下（需登录）。后端未启用/失败时分别返回 503/502，
 * 前端应先经 getAICapabilities 门控按钮，收到非 2xx 时以提示降级，不阻断手动填写。
 */
import { get, post } from '@/api/request'
import type { TaskPriority, TaskRecurrence } from '@/api/modules/task'

/** 自然语言解析产出的任务草稿（对齐创建表单，用户确认后调用 createTask） */
export interface TaskDraft {
  title: string
  priority?: TaskPriority
  due_date?: string
  tags?: string[]
  recurrence?: TaskRecurrence | ''
}

/** 优先级/标签建议 */
export interface TaskSuggestion {
  priority: TaskPriority
  tags: string[]
  reason: string
}

/** 描述扩写/拆解 */
export interface TaskExpansion {
  description: string
  steps: string[]
}

/** 摘要范围 */
export type TaskSummaryScope = 'today' | 'week' | 'overdue' | 'all'

/** 任务摘要 */
export interface TaskSummary {
  text: string
  highlights: string[]
  suggested_focus: string[]
  total: number
}

/** AI 能力探测 */
export interface AICapability {
  enabled: boolean
  model: string
}

/** 探测 AI 能力是否可用（门控前端相关按钮） */
export function getAICapabilities() {
  return get<AICapability>('/tasks/ai/capabilities')
}

/** 自然语言解析为任务草稿 */
export function parseTaskByNL(text: string) {
  return post<TaskDraft>('/tasks/ai/parse', { text })
}

/** 优先级/标签建议 */
export function suggestTaskFields(title: string, description?: string) {
  return post<TaskSuggestion>('/tasks/ai/suggest', { title, description: description ?? '' })
}

/** 描述扩写/拆解 */
export function expandTaskDescription(title: string, description?: string) {
  return post<TaskExpansion>('/tasks/ai/expand', { title, description: description ?? '' })
}

/** 生成任务摘要/周报 */
export function summarizeTasks(scope: TaskSummaryScope = 'all') {
  return post<TaskSummary>('/tasks/ai/summary', { scope })
}
