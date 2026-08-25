import request from '../request'

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

export interface PaginationMeta {
  page: number
  page_size: number
  total: number
}

export interface PaginatedResult<T> {
  data: T[]
  pagination: PaginationMeta
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

export function createSpace(req: CreateSpaceReq) {
  return request.post<any, WikiSpace>('/wiki/spaces', req)
}

export function listSpaces(params?: { page?: number; page_size?: number }) {
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

export function getNodeTree(spaceId: string) {
  return request.get<any, WikiNodeTree[]>(`/wiki/spaces/${spaceId}/nodes/tree`)
}

export function createNode(spaceId: string, req: CreateNodeReq) {
  return request.post<any, WikiNode>(`/wiki/spaces/${spaceId}/nodes`, req)
}

export function getNode(id: string) {
  return request.get<any, WikiNode>(`/wiki/nodes/${id}`)
}

export function updateNode(id: string, req: UpdateNodeReq) {
  return request.put<any, WikiNode>(`/wiki/nodes/${id}`, req)
}

export function deleteNode(id: string) {
  return request.delete<any, void>(`/wiki/nodes/${id}`)
}

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

export function listRevisions(documentId: string) {
  return request.get<any, DocumentRevision[]>(`/wiki/documents/${documentId}/revisions`)
}

export function getRevision(documentId: string, version: number) {
  return request.get<any, DocumentRevision>(`/wiki/documents/${documentId}/revisions/${version}`)
}

export function restoreRevision(documentId: string, version: number) {
  return request.post<any, void>(`/wiki/documents/${documentId}/revisions/${version}/restore`)
}

export function listMembers(spaceId: string) {
  return request.get<any, WikiSpaceMember[]>(`/wiki/spaces/${spaceId}/members`)
}

export function addMember(spaceId: string, req: AddMemberReq) {
  return request.post<any, WikiSpaceMember>(`/wiki/spaces/${spaceId}/members`, req)
}

export function removeMember(spaceId: string, userId: string) {
  return request.delete<any, void>(`/wiki/spaces/${spaceId}/members/${userId}`)
}

export function getWikiStats() {
  return request.get<any, WikiStats>('/wiki/stats')
}