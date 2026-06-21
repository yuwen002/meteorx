// internal/modules/rbac/handler/rbac_handler.go
package handler

import (
	"net/http"
	"strconv"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/rbac/dto"
	"meteorx/internal/modules/rbac/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

type RBACHandler struct {
	svc *service.RBACService
}

func NewRBACHandler(svc *service.RBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

// CreateRole POST /api/v1/rbac/roles
func (h *RBACHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRoleReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	role, err := h.svc.CreateRole(r.Context(), tenantID, req)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(w, role)
}

// GetRole GET /api/v1/rbac/roles/{id}
func (h *RBACHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := h.svc.GetRole(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "角色不存在")
		return
	}
	response.Success(w, role)
}

// ListRoles GET /api/v1/rbac/roles
func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	keyword := r.URL.Query().Get("keyword")

	roles, total, err := h.svc.ListRoles(r.Context(), tenantID, pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取角色列表失败")
		return
	}

	result := pagination.NewPaginatedResult(roles, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// UpdateRole PUT /api/v1/rbac/roles/{id}
func (h *RBACHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateRoleReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.UpdateRole(r.Context(), id, req); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// DeleteRole DELETE /api/v1/rbac/roles/{id}
func (h *RBACHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteRole(r.Context(), id); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// BindRolePermissions PUT /api/v1/rbac/roles/{id}/permissions
func (h *RBACHandler) BindRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	var req dto.BindRolePermissionsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.BindRolePermissions(r.Context(), roleID, req); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// GetRolePermissions GET /api/v1/rbac/roles/{id}/permissions
func (h *RBACHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	permissions, err := h.svc.GetRolePermissions(r.Context(), roleID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取角色权限失败")
		return
	}
	response.Success(w, permissions)
}

// CreatePermission POST /api/v1/rbac/permissions
func (h *RBACHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePermissionReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	permission, err := h.svc.CreatePermission(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(w, permission)
}

// GetPermission GET /api/v1/rbac/permissions/{id}
func (h *RBACHandler) GetPermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	permission, err := h.svc.GetPermission(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "权限不存在")
		return
	}
	response.Success(w, permission)
}

// ListPermissions GET /api/v1/rbac/permissions
func (h *RBACHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	resource := r.URL.Query().Get("resource")
	keyword := r.URL.Query().Get("keyword")

	permissions, total, err := h.svc.ListPermissions(r.Context(), pg.Page, pg.PageSize, resource, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取权限列表失败")
		return
	}

	result := pagination.NewPaginatedResult(permissions, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// UpdatePermission PUT /api/v1/rbac/permissions/{id}
func (h *RBACHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdatePermissionReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.UpdatePermission(r.Context(), id, req); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// DeletePermission DELETE /api/v1/rbac/permissions/{id}
func (h *RBACHandler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeletePermission(r.Context(), id); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}
