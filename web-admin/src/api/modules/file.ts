import request from '../request'
import type { PaginatedResult } from '@/types/pagination'

// ============================================================
// 类型定义（与后端 dto 对齐）
// ============================================================

/** 文件信息 */
export interface FileItem {
  id: string
  tenant_id: string
  user_id: string
  file_name: string
  original_name: string
  file_size: number
  mime_type: string
  /** image | document | video | audio | other */
  file_type: string
  url: string
  md5?: string
  status: number
  created_at: string
  updated_at: string
  /** 软删除时间（仅回收站有值） */
  deleted_at?: string
}

/** 上传文件响应 */
export interface UploadFileResp {
  id: string
  file_name: string
  original_name: string
  file_size: number
  mime_type: string
  file_type: string
  url: string
}

/** 文件列表请求参数 */
export interface FileListParams {
  page?: number
  page_size?: number
  /** image | document | video | audio | other */
  file_type?: string
  /** 文件名关键词搜索 */
  keyword?: string
}

/** 文件更新请求参数 */
export interface FileUpdateParams {
  file_name: string
}

/** 批量删除请求参数 */
export interface BatchDeleteParams {
  ids: string[]
}

/** 批量删除响应 */
export interface BatchDeleteResp {
  success_count: number
  failed_count: number
  failed_ids?: string[]
}

// ============================================================
// API 接口
// ============================================================

/**
 * 获取租户文件列表（所有文件）
 * GET /api/v1/files
 */
export function getFileList(params: FileListParams) {
  return request.get<any, PaginatedResult<FileItem>>('/files', { params })
}

/**
 * 获取当前用户文件列表
 * GET /api/v1/files/my
 */
export function getMyFileList(params: FileListParams) {
  return request.get<any, PaginatedResult<FileItem>>('/files/my', { params })
}

/**
 * 获取文件详情
 * GET /api/v1/files/:id
 */
export function getFileDetail(id: string) {
  return request.get<any, FileItem>(`/files/${id}`)
}

/**
 * 上传文件（支持 multipart/form-data）
 * POST /api/v1/files/upload
 */
export function uploadFile(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return request.post<any, UploadFileResp>('/files/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

/**
 * 更新文件信息（重命名）
 * PUT /api/v1/files/:id
 */
export function updateFile(id: string, data: FileUpdateParams) {
  return request.put<any, null>(`/files/${id}`, data)
}

/**
 * 删除文件（软删除，进入回收站）
 * DELETE /api/v1/files/:id
 */
export function deleteFile(id: string) {
  return request.delete<any, null>(`/files/${id}`)
}

/**
 * 批量删除文件
 * POST /api/v1/files/batch/delete
 */
export function batchDeleteFile(data: BatchDeleteParams) {
  return request.post<any, BatchDeleteResp>('/files/batch/delete', data)
}

/**
 * 获取已删除文件列表（回收站）
 * GET /api/v1/files/deleted
 */
export function getDeletedFileList(params: FileListParams) {
  return request.get<any, PaginatedResult<FileItem>>('/files/deleted', { params })
}

/**
 * 恢复已删除文件（从回收站恢复）
 * PUT /api/v1/files/:id/restore
 */
export function restoreFile(id: string) {
  return request.put<any, null>(`/files/${id}/restore`)
}

/**
 * 永久删除文件（物理删除，不可恢复）
 * DELETE /api/v1/files/:id/permanent
 */
export function permanentDeleteFile(id: string) {
  return request.delete<any, null>(`/files/${id}/permanent`)
}

/**
 * 下载文件
 * GET /api/v1/files/:id/download
 */
export function downloadFile(id: string) {
  return request.get<any, Blob>(`/files/${id}/download`, {
    responseType: 'blob'
  })
}