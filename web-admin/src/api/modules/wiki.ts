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

export function createSpace(req: CreateSpaceReq) {
  return request.post<any, WikiSpace>('/wiki/spaces', req)
}

export function listSpaces(params?: { page?: number; page_size?: number; keyword?: string }) {
  return request.get<any, PaginatedResult<WikiSpace>>('/wiki/spaces', { params })
}

export function getSpace(id: string) {
  return request.get<any, WikiSpace>(`/wiki/spaces/${id}`)
}

export function updateSpace(id: string, req: UpdateSpaceReq) {
  return request.put<any, WikiSpace>(`/wiki/spaces/${id}`, req)
}

export function deleteSpace(id: string) {
  return request.delete<any, void>(`/wiki/spaces/${id}`)
}

// ============================================================
// 节点（节点 CRUD 均挂在 /spaces/{spaceId}/nodes 之下）
// ============================================================

export function getNodeTree(spaceId: string) {
  return request.get<any, WikiNodeTree[]>(`/wiki/spaces/${spaceId}/nodes/tree`)
}

export function createNode(spaceId: string, req: CreateNodeReq) {
  return request.post<any, WikiNode>(`/wiki/spaces/${spaceId}/nodes`, req)
}

export function getNode(spaceId: string, id: string) {
  return request.get<any, WikiNode>(`/wiki/spaces/${spaceId}/nodes/${id}`)
}

export function updateNode(spaceId: string, id: string, req: UpdateNodeReq) {
  return request.put<any, WikiNode>(`/wiki/spaces/${spaceId}/nodes/${id}`, req)
}

export function deleteNode(spaceId: string, id: string) {
  return request.delete<any, void>(`/wiki/spaces/${spaceId}/nodes/${id}`)
}

// ============================================================
// 文档 & Markdown 预览
// ============================================================

export function createDocument(nodeId: string, req: CreateDocumentReq) {
  return request.post<any, WikiDocument>(`/wiki/documents/nodes/${nodeId}`, req)
}

export function getDocument(nodeId: string) {
  return request.get<any, WikiDocument>(`/wiki/documents/nodes/${nodeId}`)
}

export function updateDocument(id: string, req: UpdateDocumentReq) {
  return request.put<any, WikiDocument>(`/wiki/documents/${id}`, req)
}

export function deleteDocument(id: string) {
  return request.delete<any, void>(`/wiki/documents/${id}`)
}

export function previewMarkdown(content: string, format = 'markdown') {
  return request.post<any, { content_html: string }>('/wiki/documents/preview', { content, format })
}

// ============================================================
// 历史版本
// ============================================================

export function listRevisions(documentId: string) {
  return request.get<any, DocumentRevision[]>(`/wiki/documents/${documentId}/revisions`)
}

export function getRevision(documentId: string, version: number) {
  return request.get<any, DocumentRevision>(`/wiki/documents/${documentId}/revisions/${version}`)
}

export function restoreRevision(documentId: string, version: number) {
  return request.post<any, void>(`/wiki/documents/${documentId}/revisions/${version}/restore`)
}

// ============================================================
// 成员管理
// ============================================================

export function listMembers(spaceId: string) {
  return request.get<any, WikiSpaceMember[]>(`/wiki/spaces/${spaceId}/members`)
}

export function addMember(spaceId: string, req: AddMemberReq) {
  return request.post<any, WikiSpaceMember>(`/wiki/spaces/${spaceId}/members`, req)
}

export function removeMember(spaceId: string, userId: string) {
  return request.delete<any, void>(`/wiki/spaces/${spaceId}/members/${userId}`)
}

// ============================================================
// 节点权限
// ============================================================

export function listNodePermissions(spaceId: string, nodeId: string) {
  return request.get<any, NodePermission[]>(`/wiki/spaces/${spaceId}/nodes/${nodeId}/permissions`)
}

export function setNodePermission(spaceId: string, nodeId: string, data: { user_id: string; permission: string }) {
  return request.post<any, NodePermission>(`/wiki/spaces/${spaceId}/nodes/${nodeId}/permissions`, data)
}

export function removeNodePermission(spaceId: string, nodeId: string, userId: string, permission: string) {
  return request.delete<any, void>(
    `/wiki/spaces/${spaceId}/nodes/${nodeId}/permissions/${userId}/${permission}`
  )
}

// ============================================================
// 回收站
// ============================================================

export function listTrash(params?: { page?: number; page_size?: number; space_id?: string; item_type?: string }) {
  return request.get<any, { items: WikiTrashItem[]; total: number; page: number; page_size: number }>(
    '/wiki/trash',
    { params }
  )
}

export function restoreTrashItem(id: string) {
  return request.post<any, void>(`/wiki/trash/${id}/restore`)
}

export function permanentDeleteTrashItem(id: string) {
  return request.delete<any, void>(`/wiki/trash/${id}`)
}

// ============================================================
// 全局搜索
// ============================================================

export function searchWiki(params: { q: string; space_id?: string; page?: number; page_size?: number }) {
  return request.get<any, { results: WikiSearchItem[]; total: number; page: number; page_size: number }>(
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

export function listAttachments(documentId: string) {
  return request.get<any, WikiAttachment[]>(`/wiki/documents/${documentId}/attachments`)
}

export function createAttachment(data: CreateAttachmentReq) {
  return request.post<any, WikiAttachment>('/wiki/documents/attachments', data)
}

export function deleteAttachment(id: string) {
  return request.delete<any, void>(`/wiki/documents/attachments/${id}`)
}

export function getWikiStats() {
  return request.get<any, WikiStats>('/wiki/stats')
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

export function createTag(req: CreateTagReq) {
  return request.post<any, Tag>('/wiki/spaces/tags', req)
}

export function listTags() {
  return request.get<any, Tag[]>('/wiki/spaces/tags')
}

export function deleteTag(id: string) {
  return request.delete<any, void>(`/wiki/spaces/tags/${id}`)
}

export function addDocumentTag(documentId: string, tagId: string) {
  return request.post<any, void>(`/wiki/documents/${documentId}/tags/${tagId}`)
}

export function removeDocumentTag(documentId: string, tagId: string) {
  return request.delete<any, void>(`/wiki/documents/${documentId}/tags/${tagId}`)
}

export function listDocumentTags(documentId: string) {
  return request.get<any, DocumentTag[]>(`/wiki/documents/${documentId}/tags`)
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

export function createComment(req: CreateCommentReq) {
  return request.post<any, Comment>(`/wiki/documents/${req.document_id}/comments`, {
    document_id: req.document_id,
    parent_id: req.parent_id,
    content: req.content,
    mention_ids: req.mention_ids
  })
}

export function listComments(documentId: string) {
  return request.get<any, Comment[]>(`/wiki/documents/${documentId}/comments`)
}

export function updateComment(id: string, content: string) {
  return request.put<any, void>(`/wiki/documents/comments/${id}`, { content })
}

export function deleteComment(id: string) {
  return request.delete<any, void>(`/wiki/documents/comments/${id}`)
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
  return request.post<any, ShareLink>(`/wiki/documents/${req.document_id}/share`, body)
}

export function listShareLinks(documentId: string) {
  return request.get<any, ShareLink[]>(`/wiki/documents/${documentId}/shares`)
}

export function deleteShareLink(id: string) {
  return request.delete<any, void>(`/wiki/documents/shares/${id}`)
}

export function getShareLink(token: string, password?: string) {
  return request.get<any, ShareLink>(`/wiki/share/${token}`, { params: { password } })
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

export function createTemplate(req: CreateTemplateReq) {
  return request.post<any, DocumentTemplate>('/wiki/spaces/templates', req)
}

export function listTemplates(category?: string) {
  return request.get<any, DocumentTemplate[]>('/wiki/spaces/templates', { params: { category } })
}

export function getTemplate(id: string) {
  return request.get<any, DocumentTemplate>(`/wiki/spaces/templates/${id}`)
}

export function updateTemplate(id: string, req: Partial<CreateTemplateReq>) {
  return request.put<any, DocumentTemplate>(`/wiki/spaces/templates/${id}`, req)
}

export function deleteTemplate(id: string) {
  return request.delete<any, void>(`/wiki/spaces/templates/${id}`)
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

export function getDocumentStats(documentId: string) {
  return request.get<any, DocumentStats>(`/wiki/documents/${documentId}/stats`)
}

export function listAccessLogs(documentId: string, page = 1, pageSize = 20) {
  return request.get<any, PaginatedResult<DocumentAccessLog>>(
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

export function subscribeDocument(documentId: string) {
  return request.post<any, void>(`/wiki/documents/${documentId}/subscribe`)
}

export function unsubscribeDocument(documentId: string) {
  return request.delete<any, void>(`/wiki/documents/${documentId}/subscribe`)
}

/** 当前用户在空间内订阅的全部文档（后端 GET /wiki/spaces/subscriptions） */
export function listUserSubscriptions() {
  return request.get<any, DocumentSubscription[]>('/wiki/spaces/subscriptions')
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

export function listNotifications(page = 1, pageSize = 20) {
  return request.get<any, PaginatedResult<Notification>>('/wiki/spaces/notifications', {
    params: { page, page_size: pageSize }
  })
}

export function markNotificationAsRead(id: string) {
  return request.put<any, void>(`/wiki/spaces/notifications/${id}/read`)
}

export function markAllNotificationsAsRead() {
  return request.put<any, void>('/wiki/spaces/notifications/read-all')
}

export function getUnreadNotificationCount() {
  return request.get<any, { count: number }>('/wiki/spaces/notifications/unread-count')
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

export function acquireEditLock(documentId: string) {
  return request.post<any, EditLock>(`/wiki/documents/${documentId}/edit-lock`)
}

export function releaseEditLock(documentId: string) {
  return request.delete<any, void>(`/wiki/documents/${documentId}/edit-lock`)
}

export function refreshEditLock(documentId: string) {
  return request.put<any, void>(`/wiki/documents/${documentId}/edit-lock`)
}

export function getEditLock(documentId: string) {
  return request.get<any, EditLock>(`/wiki/documents/${documentId}/edit-lock`)
}

// 批量操作（统一 POST /wiki/spaces/nodes/batch，通过 action 区分）
export interface BatchDeleteReq {
  node_ids: string[]
}

export interface BatchMoveReq {
  node_ids: string[]
  new_parent_id: string
}

export function batchDeleteNodes(req: BatchDeleteReq) {
  return request.post<any, void>('/wiki/spaces/nodes/batch', {
    action: 'delete',
    node_ids: req.node_ids
  })
}

export function batchMoveNodes(req: BatchMoveReq) {
  return request.post<any, void>('/wiki/spaces/nodes/batch', {
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

export function compareRevisions(documentId: string, version1: number, version2: number) {
  return request.get<any, RevisionDiff>(
    `/wiki/documents/${documentId}/revisions/compare`,
    { params: { version1, version2 } }
  )
}

// 导入导出
export function exportDocument(documentId: string, format = 'markdown') {
  // 后端直接回写文件流，调用方需自行触发浏览器下载
  return request.post<Blob>(`/wiki/documents/${documentId}/export`, { format }, { responseType: 'blob' })
}

export function importDocument(documentId: string, file: File) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('format', 'markdown')
  return request.post<any, any>(`/wiki/documents/${documentId}/import`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}