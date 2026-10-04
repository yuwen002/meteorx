import request from '../request'
import type { PaginatedResult } from '@/types/pagination'

// ============================================================
// 类型定义（与后端 dto 对齐）
// ============================================================

export interface WikiSpace {
  id: string
  tenant_id: string
  name: string
  description: string
  icon: string
  visibility: number
  member_count: number
  node_count: number
  created_by: string
  created_at: string
  updated_at: string
  my_role: string
}

export interface WikiNode {
  id: string
  space_id: string
  parent_id: string
  type: 'folder' | 'document'
  title: string
  icon: string
  sort: number
  status: number
  owner_id: string
  child_count: number
  created_at: string
  updated_at: string
}

export interface WikiNodeTree extends WikiNode {
  children?: WikiNodeTree[]
}

export interface WikiDocument {
  id: string
  node_id: string
  title: string
  content: string
  content_html: string
  format: string
  current_ver: number
  view_count: number
  last_edited_by: string
  last_edited_at: string
  publish_status: string
  published_at?: string
  reviewed_by?: string
  reviewed_at?: string
  created_at: string
  updated_at: string
}

export interface DocumentRevision {
  id: string
  document_id: string
  version: number
  content: string
  content_html: string
  summary: string
  edited_by: string
  created_at: string
}

export interface WikiSpaceMember {
  id: string
  space_id: string
  user_id: string
  user_name: string
  user_email: string
  role: string
  created_at: string
}

export interface WikiStats {
  total_spaces: number
  total_nodes: number
  total_documents: number
  total_views: number
}

export interface WikiTrashItem {
  id: string
  item_type: string
  item_id: string
  space_id: string
  title: string
  deleted_by: string
  deleted_at: string
  expires_at: string
}

export interface WikiSearchItem {
  id: string
  type: string
  title: string
  space_id: string
  node_id: string
  snippet: string
  highlight: string
  updated_at: string
  score: number
}

export interface WikiAttachment {
  id: string
  document_id: string
  file_id?: string
  file_name: string
  file_size: number
  mime_type: string
  file_url: string
  uploaded_by: string
  created_at: string
}

export interface NodePermission {
  id: string
  node_id: string
  user_id: string
  user_name?: string
  permission: string
  created_at: string
  updated_at: string
}

export interface CreateSpaceReq {
  name: string
  description?: string
  icon?: string
  visibility?: number
}

export interface UpdateSpaceReq {
  name?: string
  description?: string
  icon?: string
  visibility?: number
}

export interface CreateNodeReq {
  space_id?: string
  parent_id?: string
  type: 'folder' | 'document'
  title: string
  icon?: string
  sort?: number
  content?: string
}

export interface UpdateNodeReq {
  title?: string
  icon?: string
  parent_id?: string
  sort?: number
  status?: number
}

export interface CreateDocumentReq {
  node_id: string
  content?: string
  format?: string
  summary?: string
}

export interface UpdateDocumentReq {
  content?: string
  format?: string
  summary?: string
}

export interface AddMemberReq {
  user_id: string
  role: 'owner' | 'admin' | 'editor' | 'viewer'
}

// ============================================================
// 空间
// ============================================================

// 创建知识空间
export function createSpace(req: CreateSpaceReq) {
  return request.post<unknown, WikiSpace>('/wiki/spaces', req)
}

// 获取知识空间列表（分页）
export function listSpaces(params?: { page?: number; page_size?: number; keyword?: string }) {
  return request.get<unknown, PaginatedResult<WikiSpace>>('/wiki/spaces', { params })
}

// 获取知识空间详情
export function getSpace(id: string) {
  return request.get<unknown, WikiSpace>(`/wiki/spaces/${id}`)
}

// 更新知识空间
export function updateSpace(id: string, req: UpdateSpaceReq) {
  return request.put<unknown, WikiSpace>(`/wiki/spaces/${id}`, req)
}

// 删除知识空间（进入回收站）
export function deleteSpace(id: string) {
  return request.delete<unknown, void>(`/wiki/spaces/${id}`)
}

// ============================================================
// 节点（节点 CRUD 均挂在 /spaces/{spaceId}/nodes 之下）
// ============================================================

// 获取空间下的节点树（已排序）
export function getNodeTree(spaceId: string) {
  return request.get<unknown, WikiNodeTree[]>(`/wiki/spaces/${spaceId}/nodes/tree`)
}

// 创建节点（文件夹或文档）
export function createNode(spaceId: string, req: CreateNodeReq) {
  return request.post<unknown, WikiNode>(`/wiki/spaces/${spaceId}/nodes`, req)
}

// 获取节点详情
export function getNode(spaceId: string, id: string) {
  return request.get<unknown, WikiNode>(`/wiki/spaces/${spaceId}/nodes/${id}`)
}

// 更新节点（支持移动、重命名、排序）
export function updateNode(spaceId: string, id: string, req: UpdateNodeReq) {
  return request.put<unknown, WikiNode>(`/wiki/spaces/${spaceId}/nodes/${id}`, req)
}

// 删除节点（进入回收站）
export function deleteNode(spaceId: string, id: string) {
  return request.delete<unknown, void>(`/wiki/spaces/${spaceId}/nodes/${id}`)
}

// ============================================================
// 文档 & Markdown 预览
// ============================================================

// 创建文档（关联到指定节点）
export function createDocument(nodeId: string, req: CreateDocumentReq) {
  return request.post<unknown, WikiDocument>(`/wiki/documents/nodes/${nodeId}`, req)
}

// 获取文档详情（通过节点 ID 查询）
export function getDocument(nodeId: string) {
  return request.get<unknown, WikiDocument>(`/wiki/documents/nodes/${nodeId}`)
}

// 更新文档内容
export function updateDocument(id: string, req: UpdateDocumentReq) {
  return request.put<unknown, WikiDocument>(`/wiki/documents/${id}`, req)
}

// 删除文档
export function deleteDocument(id: string) {
  return request.delete<unknown, void>(`/wiki/documents/${id}`)
}

// Markdown 内容预览（返回渲染后的 HTML）
export function previewMarkdown(content: string, format = 'markdown') {
  return request.post<unknown, { content_html: string }>('/wiki/documents/preview', { content, format })
}

// ============================================================
// 历史版本
// ============================================================

// 获取文档历史版本列表
export function listRevisions(documentId: string) {
  return request.get<unknown, DocumentRevision[]>(`/wiki/documents/${documentId}/revisions`)
}

// 获取指定版本详情
export function getRevision(documentId: string, version: number) {
  return request.get<unknown, DocumentRevision>(`/wiki/documents/${documentId}/revisions/${version}`)
}

// 恢复到指定历史版本
export function restoreRevision(documentId: string, version: number) {
  return request.post<unknown, void>(`/wiki/documents/${documentId}/revisions/${version}/restore`)
}

// ============================================================
// 成员管理
// ============================================================

// 获取空间成员列表
export function listMembers(spaceId: string) {
  return request.get<unknown, WikiSpaceMember[]>(`/wiki/spaces/${spaceId}/members`)
}

// 添加空间成员
export function addMember(spaceId: string, req: AddMemberReq) {
  return request.post<unknown, WikiSpaceMember>(`/wiki/spaces/${spaceId}/members`, req)
}

// 移除空间成员
export function removeMember(spaceId: string, userId: string) {
  return request.delete<unknown, void>(`/wiki/spaces/${spaceId}/members/${userId}`)
}

// ============================================================
// 节点权限
// ============================================================

// 获取节点权限列表
export function listNodePermissions(spaceId: string, nodeId: string) {
  return request.get<unknown, NodePermission[]>(`/wiki/spaces/${spaceId}/nodes/${nodeId}/permissions`)
}

// 设置节点权限（如已存在则更新）
export function setNodePermission(spaceId: string, nodeId: string, data: { user_id: string; permission: string }) {
  return request.post<unknown, NodePermission>(`/wiki/spaces/${spaceId}/nodes/${nodeId}/permissions`, data)
}

// 移除节点权限
export function removeNodePermission(spaceId: string, nodeId: string, userId: string, permission: string) {
  return request.delete<unknown, void>(
    `/wiki/spaces/${spaceId}/nodes/${nodeId}/permissions/${userId}/${permission}`
  )
}

// ============================================================
// 回收站
// ============================================================

// 获取回收站列表（分页）
export function listTrash(params?: { page?: number; page_size?: number; space_id?: string; item_type?: string }) {
  return request.get<unknown, { items: WikiTrashItem[]; total: number; page: number; page_size: number }>(
    '/wiki/trash',
    { params }
  )
}

// 恢复回收站项目
export function restoreTrashItem(id: string) {
  return request.post<unknown, void>(`/wiki/trash/${id}/restore`)
}

// 永久删除回收站项目（不可恢复）
export function permanentDeleteTrashItem(id: string) {
  return request.delete<unknown, void>(`/wiki/trash/${id}`)
}

// ============================================================
// 全局搜索
// ============================================================

// 全局搜索 Wiki 内容
export function searchWiki(params: { q: string; space_id?: string; page?: number; page_size?: number }) {
  return request.get<unknown, { results: WikiSearchItem[]; total: number; page: number; page_size: number }>(
    '/wiki/search',
    { params }
  )
}

// ============================================================
// 附件
// ============================================================

export interface CreateAttachmentReq {
  document_id: string
  file_id?: string
  file_name: string
  file_size?: number
  mime_type?: string
  file_url: string
}

// 获取文档附件列表
export function listAttachments(documentId: string) {
  return request.get<unknown, WikiAttachment[]>(`/wiki/documents/${documentId}/attachments`)
}

// 创建附件（关联文件到文档）
export function createAttachment(data: CreateAttachmentReq) {
  return request.post<unknown, WikiAttachment>('/wiki/documents/attachments', data)
}

// 删除附件
export function deleteAttachment(id: string) {
  return request.delete<unknown, void>(`/wiki/documents/attachments/${id}`)
}

// 获取 Wiki 统计信息
export function getWikiStats() {
  return request.get<unknown, WikiStats>('/wiki/stats')
}

// ============================================================
// 扩展功能 API
// ============================================================

// 标签系统
export interface Tag {
  id: string
  tenant_id: string
  name: string
  color: string
  created_by: string
  created_at: string
}

export interface DocumentTag {
  id: string
  document_id: string
  /** 后端以嵌套 tag 对象返回，无 tag_id 顶层字段 */
  tag: Tag
  created_at: string
}

export interface CreateTagReq {
  name: string
  color: string
}

// 创建标签
export function createTag(req: CreateTagReq) {
  return request.post<unknown, Tag>('/wiki/spaces/tags', req)
}

// 获取标签列表
export function listTags() {
  return request.get<unknown, Tag[]>('/wiki/spaces/tags')
}

// 删除标签
export function deleteTag(id: string) {
  return request.delete<unknown, void>(`/wiki/spaces/tags/${id}`)
}

// 为文档添加标签
export function addDocumentTag(documentId: string, tagId: string) {
  return request.post<unknown, void>(`/wiki/documents/${documentId}/tags/${tagId}`)
}

// 移除文档标签
export function removeDocumentTag(documentId: string, tagId: string) {
  return request.delete<unknown, void>(`/wiki/documents/${documentId}/tags/${tagId}`)
}

// 获取文档标签列表
export function listDocumentTags(documentId: string) {
  return request.get<unknown, DocumentTag[]>(`/wiki/documents/${documentId}/tags`)
}

// 评论系统
export interface Comment {
  id: string
  document_id: string
  node_id?: string
  parent_id?: string
  content: string
  created_by: string
  user_name?: string
  mention_ids?: string
  status?: number
  replies?: Comment[]
  created_at: string
  updated_at: string
}

export interface CreateCommentReq {
  document_id: string
  parent_id?: string
  content: string
  mention_ids?: string
}

// 创建评论（支持 @提及）
export function createComment(req: CreateCommentReq) {
  return request.post<unknown, Comment>(`/wiki/documents/${req.document_id}/comments`, {
    document_id: req.document_id,
    parent_id: req.parent_id,
    content: req.content,
    mention_ids: req.mention_ids
  })
}

// 获取文档评论列表（含嵌套回复）
export function listComments(documentId: string) {
  return request.get<unknown, Comment[]>(`/wiki/documents/${documentId}/comments`)
}

// 更新评论内容
export function updateComment(id: string, content: string) {
  return request.put<unknown, void>(`/wiki/documents/comments/${id}`, { content })
}

// 删除评论
export function deleteComment(id: string) {
  return request.delete<unknown, void>(`/wiki/documents/comments/${id}`)
}

// 分享链接
export interface ShareLink {
  id: string
  document_id: string
  node_id?: string
  token: string
  password?: string
  /** 过期时间（后端 DTO 字段为 expire_at） */
  expire_at?: string
  max_views?: number
  view_count: number
  allow_download?: boolean
  created_by?: string
  created_at: string
  /** 相对分享地址，例如 /wiki/share/{token} */
  share_url?: string
}

export interface CreateShareLinkReq {
  document_id: string
  password?: string
  /** 过期时间，支持任意可被 Date 解析的格式（如 "2026-01-01 12:00:00" 或 RFC3339） */
  expires_at?: string
  max_views?: number
  allow_download?: boolean
}

// 创建分享链接（支持密码、过期时间、访问次数限制）
export function createShareLink(req: CreateShareLinkReq) {
  const body: Record<string, unknown> = {
    document_id: req.document_id,
    password: req.password ?? '',
    max_views: req.max_views ?? 0,
    allow_download: req.allow_download ?? false
  }
  // 后端 DTO 字段为 expire_at，且期望可解析的时间值（RFC3339 / 时间戳）
  if (req.expires_at) {
    const ts = Date.parse(req.expires_at)
    body.expire_at = isNaN(ts) ? req.expires_at : new Date(ts).toISOString()
  }
  return request.post<unknown, ShareLink>(`/wiki/documents/${req.document_id}/share`, body)
}

// 获取文档分享链接列表
export function listShareLinks(documentId: string) {
  return request.get<unknown, ShareLink[]>(`/wiki/documents/${documentId}/shares`)
}

// 删除分享链接
export function deleteShareLink(id: string) {
  return request.delete<unknown, void>(`/wiki/documents/shares/${id}`)
}

// 获取分享链接详情（公开接口，通过 token 访问）
export function getShareLink(token: string, password?: string) {
  return request.get<unknown, ShareLink>(`/wiki/share/${token}`, { params: { password } })
}

// 文档模板（挂在 /wiki/spaces/templates 下）
export interface DocumentTemplate {
  id: string
  tenant_id: string
  name: string
  description: string
  format: string
  category: string
  content: string
  is_public: boolean
  created_by: string
  created_at: string
  updated_at: string
}

export interface CreateTemplateReq {
  name: string
  description?: string
  format?: string
  category: string
  content: string
  is_public?: boolean
}

// 创建文档模板
export function createTemplate(req: CreateTemplateReq) {
  return request.post<unknown, DocumentTemplate>('/wiki/spaces/templates', req)
}

// 获取模板列表（可按分类筛选）
export function listTemplates(category?: string) {
  return request.get<unknown, DocumentTemplate[]>('/wiki/spaces/templates', { params: { category } })
}

// 获取模板详情
export function getTemplate(id: string) {
  return request.get<unknown, DocumentTemplate>(`/wiki/spaces/templates/${id}`)
}

// 更新模板
export function updateTemplate(id: string, req: Partial<CreateTemplateReq>) {
  return request.put<unknown, DocumentTemplate>(`/wiki/spaces/templates/${id}`, req)
}

// 删除模板
export function deleteTemplate(id: string) {
  return request.delete<unknown, void>(`/wiki/spaces/templates/${id}`)
}

// 访问统计
export interface DocumentStats {
  document_id?: string
  total_views: number
  total_edits: number
  total_downloads: number
  total_shares: number
  unique_viewers: number
  last_viewed_at: string
}

export interface DocumentAccessLog {
  id: string
  document_id: string
  user_id: string
  user_name?: string
  action: string
  ip_address: string
  created_at: string
}

// 获取文档访问统计
export function getDocumentStats(documentId: string) {
  return request.get<unknown, DocumentStats>(`/wiki/documents/${documentId}/stats`)
}

// 获取文档访问日志（分页）
export function listAccessLogs(documentId: string, page = 1, pageSize = 20) {
  return request.get<unknown, PaginatedResult<DocumentAccessLog>>(
    `/wiki/documents/${documentId}/access-logs`,
    { params: { page, page_size: pageSize } }
  )
}

// 订阅管理
export interface DocumentSubscription {
  id: string
  document_id: string
  node_id: string
  user_id: string
  user_name?: string
  notify_type: string
  created_at: string
}

// 订阅文档变更通知
export function subscribeDocument(documentId: string) {
  return request.post<unknown, void>(`/wiki/documents/${documentId}/subscribe`)
}

// 取消订阅文档
export function unsubscribeDocument(documentId: string) {
  return request.delete<unknown, void>(`/wiki/documents/${documentId}/subscribe`)
}

// 当前用户在空间内订阅的全部文档
export function listUserSubscriptions() {
  return request.get<unknown, DocumentSubscription[]>('/wiki/spaces/subscriptions')
}

// 通知系统（挂在 /wiki/spaces/notifications 下）
export interface Notification {
  id: string
  user_id: string
  title: string
  content: string
  type: string
  is_read: boolean
  related_id?: string
  related_type?: string
  created_at: string
}

// 获取通知列表（分页）
export function listNotifications(page = 1, pageSize = 20) {
  return request.get<unknown, PaginatedResult<Notification>>('/wiki/spaces/notifications', {
    params: { page, page_size: pageSize }
  })
}

// 标记通知为已读
export function markNotificationAsRead(id: string) {
  return request.put<unknown, void>(`/wiki/spaces/notifications/${id}/read`)
}

// 标记所有通知为已读
export function markAllNotificationsAsRead() {
  return request.put<unknown, void>('/wiki/spaces/notifications/read-all')
}

// 获取未读通知数量
export function getUnreadNotificationCount() {
  return request.get<unknown, { count: number }>('/wiki/spaces/notifications/unread-count')
}

// 编辑锁（挂在 /wiki/documents/{id}/edit-lock 下）
export interface EditLock {
  document_id: string
  user_id: string
  user_name?: string
  locked_at: string
  expires_at: string
  can_edit: boolean
}

// 获取编辑锁（若不存在则自动获取）
export function acquireEditLock(documentId: string) {
  return request.post<unknown, EditLock>(`/wiki/documents/${documentId}/edit-lock`)
}

// 释放编辑锁
export function releaseEditLock(documentId: string) {
  return request.delete<unknown, void>(`/wiki/documents/${documentId}/edit-lock`)
}

// 刷新编辑锁（延长过期时间）
export function refreshEditLock(documentId: string) {
  return request.put<unknown, void>(`/wiki/documents/${documentId}/edit-lock`)
}

// 获取编辑锁状态
export function getEditLock(documentId: string) {
  return request.get<unknown, EditLock>(`/wiki/documents/${documentId}/edit-lock`)
}

// 批量操作（统一 POST /wiki/spaces/nodes/batch，通过 action 区分）
export interface BatchDeleteReq {
  node_ids: string[]
}

export interface BatchMoveReq {
  node_ids: string[]
  new_parent_id: string
}

// 批量删除节点
export function batchDeleteNodes(req: BatchDeleteReq) {
  return request.post<unknown, void>('/wiki/spaces/nodes/batch', {
    action: 'delete',
    node_ids: req.node_ids
  })
}

// 批量移动节点到新父节点
export function batchMoveNodes(req: BatchMoveReq) {
  return request.post<unknown, void>('/wiki/spaces/nodes/batch', {
    action: 'move',
    node_ids: req.node_ids,
    target: req.new_parent_id
  })
}

// 版本对比
export interface RevisionDiffLine {
  type: 'added' | 'removed' | 'unchanged'
  line_num: number
  content: string
  old_line?: number
  new_line?: number
}

export interface RevisionDiff {
  old_version: number
  new_version: number
  diffs: RevisionDiffLine[]
}

// 对比两个版本的差异
export function compareRevisions(documentId: string, version1: number, version2: number) {
  return request.get<unknown, RevisionDiff>(
    `/wiki/documents/${documentId}/revisions/compare`,
    { params: { version1, version2 } }
  )
}

// 导出文档（后端返回文件流，调用方需自行触发浏览器下载）
export function exportDocument(documentId: string, format = 'markdown') {
  return request.post<Blob>(`/wiki/documents/${documentId}/export`, { format }, { responseType: 'blob' })
}

// 导入文档（支持 Markdown 文件上传）
export function importDocument(documentId: string, file: File) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('format', 'markdown')
  return request.post<unknown, { document_id?: string; title?: string; format?: string }>(`/wiki/documents/${documentId}/import`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// ============================================================
// 发布工作流
// ============================================================

export type PublishStatus = 'draft' | 'pending_review' | 'published' | 'archived' | 'rejected'

export interface DocumentPublishStatus {
  document_id: string
  node_id: string
  title: string
  publish_status: PublishStatus
  published_at?: string
  reviewed_by?: string
  reviewed_at?: string
}

export interface ReviewComment {
  id: string
  document_id: string
  node_id: string
  action: string
  content: string
  reviewer_id: string
  reviewer_name: string
  created_at: string
}

export interface PendingReviewItem {
  id: string
  document_id: string
  node_id: string
  title: string
  space_id: string
  space_name: string
  publish_status: PublishStatus
  submitted_by: string
  submitted_at?: string
}

// 提交文档审核
export function submitForReview(documentId: string, comment?: string) {
  return request.post<unknown, DocumentPublishStatus>(`/wiki/documents/${documentId}/submit-review`, { comment })
}

// 审核通过
export function approveDocument(documentId: string, comment?: string) {
  return request.post<unknown, DocumentPublishStatus>(`/wiki/documents/${documentId}/approve`, { comment })
}

// 审核驳回
export function rejectDocument(documentId: string, comment?: string) {
  return request.post<unknown, DocumentPublishStatus>(`/wiki/documents/${documentId}/reject`, { comment })
}

// 发布文档
export function publishDocument(documentId: string) {
  return request.post<unknown, DocumentPublishStatus>(`/wiki/documents/${documentId}/publish`)
}

// 取消发布
export function unpublishDocument(documentId: string) {
  return request.post<unknown, DocumentPublishStatus>(`/wiki/documents/${documentId}/unpublish`)
}

// 归档文档
export function archiveDocument(documentId: string) {
  return request.post<unknown, DocumentPublishStatus>(`/wiki/documents/${documentId}/archive`)
}

// 获取审核评论列表
export function listReviewComments(documentId: string) {
  return request.get<unknown, ReviewComment[]>(`/wiki/documents/${documentId}/review-comments`)
}

// 获取待审核列表（分页）
export function listPendingReviews(page = 1, pageSize = 20) {
  return request.get<unknown, { items: PendingReviewItem[]; total: number; page: number; page_size: number }>(
    '/wiki/pending-reviews',
    { params: { page, page_size: pageSize } }
  )
}