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

type WikiHandler struct {
	svc service.WikiService
}

func NewWikiHandler(svc service.WikiService) *WikiHandler {
	return &WikiHandler{svc: svc}
}

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

func (h *WikiHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteDocument(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "documentId")

	revisions, err := h.svc.ListRevisions(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, revisions)
}

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

func (h *WikiHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceId")
	userID := chi.URLParam(r, "userId")

	if err := h.svc.RemoveMember(r.Context(), spaceID, userID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceId")

	members, err := h.svc.ListMembers(r.Context(), spaceID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, members)
}

func (h *WikiHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())

	stats, err := h.svc.GetStats(r.Context(), tenantID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, stats)
}

// SetNodePermission 设置节点权限
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

// GetNodePermissions 获取节点的所有权限
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

// RemoveNodePermission 移除节点权限
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