/**
 * 数据导出模块 API
 *
 * 提供异步 CSV 导出任务的创建、查询、列表、下载与删除：
 * - 创建导出任务（指定资源类型与筛选条件），后端 worker 异步生成 CSV
 * - 分页列出当前用户的导出任务及其状态
 * - 下载已完成任务的 CSV 文件
 * - 删除导出任务记录及其文件
 */
import { get, post, del } from '@/api/request'
import request from '@/api/request'
import type { PageResult } from '@/types/pagination'

/** 导出任务状态：pending / running / completed / failed */
export type ExportStatus = 'pending' | 'running' | 'completed' | 'failed'

/** 可导出资源类型 */
export type ExportResource = 'user' | 'task' | 'audit_log' | 'wiki_document'

/** 导出文件格式 */
export type ExportFormat = 'csv' | 'xlsx' | 'json' | 'pdf'

/** 导出任务记录（与后端 dto.ExportJobResp 对齐） */
export interface ExportJobItem {
  id: string
  tenant_id: string
  user_id: string
  resource: string
  /** 导出格式：csv / xlsx / json / pdf，空表示 csv */
  format?: string
  status: string
  rows: number
  error_msg?: string
  created_at: string
  started_at?: string
  completed_at?: string
  expires_at: string
}

/** 创建导出任务参数 */
export interface CreateExportParams {
  resource: ExportResource
  /** 导出格式，不传默认 csv */
  format?: ExportFormat
  /** 目标租户：仅系统管理员生效，为其它租户创建任务时显式声明归属 */
  tenant_id?: string
  /** 筛选条件；导出所选用户时传 { ids: [...] }（后端限 200 条） */
  filters?: Record<string, unknown>
}

/** 导出列表查询参数 */
export interface ExportListParams {
  page?: number
  page_size?: number
  status?: string
  resource?: string
}

/** 分页获取当前用户的导出任务列表 */
export function getExportList(params: ExportListParams) {
  return get<PageResult<ExportJobItem>>('/exports', params)
}

/** 创建导出任务 */
export function createExport(data: CreateExportParams) {
  return post<ExportJobItem>('/exports', data)
}

/** 获取单个导出任务详情 */
export function getExportDetail(id: string) {
  return get<ExportJobItem>(`/exports/${id}`)
}

/** 下载已完成导出的 CSV 文件（blob） */
export function downloadExport(id: string) {
  return request.get<unknown, Blob>(`/exports/${id}/download`, {
    responseType: 'blob'
  })
}

/** 删除导出任务（同时删除文件） */
export function deleteExport(id: string) {
  return del(`/exports/${id}`)
}
