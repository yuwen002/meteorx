// Package handler 提供 Wiki 模块 HTTP 处理器
package handler

import (
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/service"
	"meteorx/pkg/pagination"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// WikiHandler Wiki 模块 HTTP 处理器
type WikiHandler struct {
	svc service.WikiService
}

// NewWikiHandler 创建 Wiki 模块处理器
func NewWikiHandler(svc service.WikiService) *WikiHandler {
	return &WikiHandler{svc: svc}
}

// CreateSpace 创建 Wiki Space，并将创建者自动添加为 Owner
// POST /api/v1/wiki/spaces
func (h *WikiHandler) CreateSpace(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateWikiSpaceReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())

	space, err := h.svc.CreateSpace(r.Context(), tenantID, userID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, space)
}

// GetSpace 获取 Space 详情
// GET /api/v1/wiki/spaces/{id}
func (h *WikiHandler) GetSpace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "space ID is required")
		return
	}

	userID := contextx.GetUserID(r.Context())
	space, err := h.svc.GetSpace(r.Context(), id, userID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, space)
}

// ListSpaces 列出用户可访问的 Spaces（分页）
// GET /api/v1/wiki/spaces
func (h *WikiHandler) ListSpaces(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())

	spaces, total, err := h.svc.ListSpaces(r.Context(), tenantID, userID, pg.Page, pg.PageSize)
	if err != nil {
		response.FailError(w, err)
		return
	}

	response.SuccessWithPagination(w, spaces, pg.Page, pg.PageSize, total)
}

// UpdateSpace 更新 Space 信息
// PUT /api/v1/wiki/spaces/{id}
func (h *WikiHandler) UpdateSpace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "space ID is required")
		return
	}

	var req dto.UpdateWikiSpaceReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	space, err := h.svc.UpdateSpace(r.Context(), id, tenantID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, space)
}

// DeleteSpace 删除 Space（进入回收站）
// DELETE /api/v1/wiki/spaces/{id}
func (h *WikiHandler) DeleteSpace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "space ID is required")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if err := h.svc.DeleteSpace(r.Context(), id, tenantID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// CreateNode 创建节点（文件夹或文档）
// POST /api/v1/wiki/spaces/{spaceId}/nodes
func (h *WikiHandler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateWikiNodeReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	spaceID := chi.URLParam(r, "spaceId")
	userID := contextx.GetUserID(r.Context())

	node, err := h.svc.CreateNode(r.Context(), spaceID, userID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, node)
}

// GetNode 获取节点详情
// GET /api/v1/wiki/spaces/{spaceId}/nodes/{id}
func (h *WikiHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "node ID is required")
		return
	}

	node, err := h.svc.GetNode(r.Context(), id)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, node)
}

// GetNodeTree 获取 Space 下的完整节点树（已排序）
// GET /api/v1/wiki/spaces/{spaceId}/nodes/tree
func (h *WikiHandler) GetNodeTree(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceId")
	if spaceID == "" {
		response.BadRequest(w, "space ID is required")
		return
	}

	tree, err := h.svc.GetNodeTree(r.Context(), spaceID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, tree)
}

// UpdateNode 更新节点（支持移动、重命名、排序）
// PUT /api/v1/wiki/spaces/{spaceId}/nodes/{id}
func (h *WikiHandler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "node ID is required")
		return
	}

	var req dto.UpdateWikiNodeReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	node, err := h.svc.UpdateNode(r.Context(), id, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, node)
}

// DeleteNode 删除节点及其子树（进入回收站 + 级联删除）
// DELETE /api/v1/wiki/spaces/{spaceId}/nodes/{id}
func (h *WikiHandler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "node ID is required")
		return
	}

	if err := h.svc.DeleteNode(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// MoveNode 移动节点到新父节点（含安全校验）
// POST /api/v1/wiki/spaces/{spaceId}/nodes/{id}/move
func (h *WikiHandler) MoveNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "node ID is required")
		return
	}

	userID := contextx.GetUserID(r.Context())
	var req struct {
		NewParentID string `json:"new_parent_id"`
	}
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.MoveNode(r.Context(), id, req.NewParentID, userID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// SortNode 设置节点排序值
// PUT /api/v1/wiki/spaces/{spaceId}/nodes/{id}/sort
func (h *WikiHandler) SortNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "node ID is required")
		return
	}

	userID := contextx.GetUserID(r.Context())
	var req struct {
		Sort int `json:"sort"`
	}
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.SortNode(r.Context(), id, req.Sort, userID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// CreateDocument 创建文档（支持 Markdown 渲染）
// POST /api/v1/wiki/documents/nodes/{nodeId}
func (h *WikiHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeId")
	userID := contextx.GetUserID(r.Context())

	var req dto.CreateDocumentReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	req.NodeID = nodeID

	doc, err := h.svc.CreateDocument(r.Context(), nodeID, userID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, doc)
}

// GetDocument 获取文档内容
// GET /api/v1/wiki/documents/nodes/{nodeId}
func (h *WikiHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeId")
	userID := contextx.GetUserID(r.Context())

	doc, err := h.svc.GetDocument(r.Context(), nodeID, userID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, doc)
}

// UpdateDocument 更新文档（事务保证 + Revision 自动创建）
// PUT /api/v1/wiki/documents/{id}
func (h *WikiHandler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := contextx.GetUserID(r.Context())

	var req dto.UpdateDocumentReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	doc, err := h.svc.UpdateDocument(r.Context(), id, userID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, doc)
}

// DeleteDocument 删除文档（进入回收站 + 级联删除附件）
// DELETE /api/v1/wiki/documents/{id}
func (h *WikiHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteDocument(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// ListRevisions 列出文档的所有历史版本
// GET /api/v1/wiki/documents/{documentId}/revisions
func (h *WikiHandler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")

	revisions, err := h.svc.ListRevisions(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, revisions)
}

// GetRevision 获取指定版本的历史内容
// GET /api/v1/wiki/documents/{documentId}/revisions/{version}
func (h *WikiHandler) GetRevision(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")
	versionStr := chi.URLParam(r, "version")
	version, _ := strconv.Atoi(versionStr)

	revision, err := h.svc.GetRevision(r.Context(), documentID, version)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, revision)
}

// RestoreRevision 恢复到指定版本（自动保存当前版本为新 Revision）
// POST /api/v1/wiki/documents/{documentId}/revisions/{version}/restore
func (h *WikiHandler) RestoreRevision(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")
	versionStr := chi.URLParam(r, "version")
	version, _ := strconv.Atoi(versionStr)
	userID := contextx.GetUserID(r.Context())

	doc, err := h.svc.RestoreRevision(r.Context(), documentID, version, userID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, doc)
}

// AddMember 添加 Space 成员
// POST /api/v1/wiki/spaces/{spaceId}/members
func (h *WikiHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceId")

	var req dto.WikiSpaceMemberReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	member, err := h.svc.AddMember(r.Context(), spaceID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, member)
}

// RemoveMember 移除成员
// DELETE /api/v1/wiki/spaces/{spaceId}/members/{userId}
func (h *WikiHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceId")
	userID := chi.URLParam(r, "userId")

	if err := h.svc.RemoveMember(r.Context(), spaceID, userID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// ListMembers 列出 Space 成员列表
// GET /api/v1/wiki/spaces/{spaceId}/members
func (h *WikiHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceId")

	members, err := h.svc.ListMembers(r.Context(), spaceID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, members)
}

// GetStats 获取 Wiki 统计数据
// GET /api/v1/wiki/stats
func (h *WikiHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())

	stats, err := h.svc.GetStats(r.Context(), tenantID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, stats)
}

// SetNodePermission 设置节点级别权限
// POST /api/v1/wiki/spaces/{spaceId}/nodes/{id}/permissions
func (h *WikiHandler) SetNodePermission(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	if nodeID == "" {
		response.BadRequest(w, "node ID is required")
		return
	}

	var req dto.SetNodePermissionReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	perm, err := h.svc.SetNodePermission(r.Context(), nodeID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, perm)
}

// GetNodePermissions 获取节点的所有权限配置
// GET /api/v1/wiki/spaces/{spaceId}/nodes/{id}/permissions
func (h *WikiHandler) GetNodePermissions(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	if nodeID == "" {
		response.BadRequest(w, "node ID is required")
		return
	}

	perms, err := h.svc.GetNodePermissions(r.Context(), nodeID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, perms)
}

// RemoveNodePermission 移除节点级别权限
// DELETE /api/v1/wiki/spaces/{spaceId}/nodes/{id}/permissions/{userId}/{permission}
func (h *WikiHandler) RemoveNodePermission(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	permission := chi.URLParam(r, "permission")

	if nodeID == "" || userID == "" || permission == "" {
		response.BadRequest(w, "node ID, user ID and permission are required")
		return
	}

	if err := h.svc.RemoveNodePermission(r.Context(), nodeID, userID, permission); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// ListTrashItems 列出回收站项目（分页）
// GET /api/v1/wiki/trash
func (h *WikiHandler) ListTrashItems(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	spaceID := r.URL.Query().Get("space_id")
	itemType := r.URL.Query().Get("item_type")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	items, total, err := h.svc.ListTrashItems(r.Context(), tenantID, spaceID, itemType, page, pageSize)
	if err != nil {
		response.FailError(w, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// RestoreTrashItem 从回收站恢复项目
// POST /api/v1/wiki/trash/{id}/restore
func (h *WikiHandler) RestoreTrashItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "trash item ID is required")
		return
	}

	userID := contextx.GetUserID(r.Context())
	if err := h.svc.RestoreTrashItem(r.Context(), id, userID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// PermanentDeleteTrashItem 永久删除回收站项目
// DELETE /api/v1/wiki/trash/{id}
func (h *WikiHandler) PermanentDeleteTrashItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "trash item ID is required")
		return
	}

	userID := contextx.GetUserID(r.Context())
	if err := h.svc.PermanentDeleteTrashItem(r.Context(), id, userID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// Search 搜索 Wiki（标题 + 内容，自动过滤无权限内容）
// GET /api/v1/wiki/search
func (h *WikiHandler) Search(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())
	query := r.URL.Query().Get("q")
	spaceID := r.URL.Query().Get("space_id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	results, total, err := h.svc.Search(r.Context(), tenantID, userID, query, spaceID, page, pageSize)
	if err != nil {
		response.FailError(w, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"results":   results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"query":     query,
	})
}

// CreateAttachment 创建文档附件
// POST /api/v1/wiki/documents/attachments
func (h *WikiHandler) CreateAttachment(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())

	var req dto.CreateAttachmentReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	attachment, err := h.svc.CreateAttachment(r.Context(), userID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, attachment)
}

// ListAttachments 列出文档的所有附件
// GET /api/v1/wiki/documents/{documentId}/attachments
func (h *WikiHandler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")
	if documentID == "" {
		response.BadRequest(w, "document ID is required")
		return
	}

	userID := contextx.GetUserID(r.Context())
	attachments, err := h.svc.ListAttachments(r.Context(), documentID, userID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, attachments)
}

// DeleteAttachment 删除附件
// DELETE /api/v1/wiki/documents/attachments/{id}
func (h *WikiHandler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "attachment ID is required")
		return
	}

	userID := contextx.GetUserID(r.Context())
	if err := h.svc.DeleteAttachment(r.Context(), id, userID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}
