// Package handler 提供 RBAC 角色权限管理 HTTP 处理器
package handler

import (
	"meteorx/internal/modules/rbac/model"
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

// RBACHandler 角色权限管理处理器
type RBACHandler struct {
	svc *service.RBACService
}

// NewRBACHandler 创建角色权限管理处理器
func NewRBACHandler(svc *service.RBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

// CreateRole 创建角色
// POST /api/v1/rbac/roles
func (h *RBACHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRoleReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 使用请求体中的 tenant_id，为空时由 service 层默认为 SYSTEM_ROOT
	role, err := h.svc.CreateRole(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(w, dto.ToRoleResp(role))
}

// GetRole 获取单个角色详情
// GET /api/v1/rbac/roles/{id}/detail
func (h *RBACHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id") // 从 URL 路径中提取角色 ID
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}
	// 调用 service 根据 ID 查询角色
	role, err := h.svc.GetRole(r.Context(), id)
	if err != nil {
		// 角色不存在或查询失败，返回 404 状态码
		response.Fail(w, http.StatusNotFound, "角色不存在")
		return
	}
	// 返回角色详情
	response.Success(w, dto.ToRoleResp(role))
}

// ListRoles 获取角色列表（带分页和关键字搜索）
// GET /api/v1/rbac/roles
func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	// 从 URL 查询参数中读取分页参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	keyword := r.URL.Query().Get("keyword")
	tenantID := r.URL.Query().Get("tenant_id") // 可选：按租户过滤，为空则查全部

	roles, total, err := h.svc.ListRoles(r.Context(), tenantID, pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取角色列表失败")
		return
	}

	// 转换为 RoleResp
	resp := make([]*dto.RoleResp, len(roles))
	for i, role := range roles {
		resp[i] = dto.ToRoleResp(role)
	}

	result := pagination.NewPaginatedResult(resp, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// ListRolesForSelect 获取角色下拉列表（不分页，用于选择）
// GET /api/v1/rbac/roles/select
func (h *RBACHandler) ListRolesForSelect(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "system" // 默认返回系统级角色
	}

	roles, err := h.svc.ListRolesByScope(r.Context(), scope)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取角色列表失败")
		return
	}

	// 转换为 RoleResp
	resp := make([]*dto.RoleResp, len(roles))
	for i, role := range roles {
		resp[i] = dto.ToRoleResp(role)
	}

	response.Success(w, resp)
}

// ListSystemAdminRoles 获取系统管理员角色列表
// GET /api/v1/rbac/roles/system-admin
func (h *RBACHandler) ListSystemAdminRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListSystemAdminRoles(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取系统管理员角色列表失败")
		return
	}

	// 转换为 RoleResp
	resp := make([]*dto.RoleResp, len(roles))
	for i, role := range roles {
		resp[i] = dto.ToRoleResp(role)
	}

	response.Success(w, resp)
}

// UpdateRole 更新角色信息
// PUT /api/v1/rbac/roles/{id}/update
func (h *RBACHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id") // 获取要更新的角色 ID
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	var req dto.UpdateRoleReq // 角色更新请求结构体
	// 解析并校验请求体，校验失败则返回
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 调用 service 执行更新操作
	role, err := h.svc.UpdateRole(r.Context(), id, req)
	if err != nil {
		// 更新失败，返回 400 状态码
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// 更新成功，返回更新后的角色信息
	response.Success(w, dto.ToRoleResp(role))
}

// DeleteRole 删除角色
// DELETE /api/v1/rbac/roles/{id}/delete
func (h *RBACHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id") // 获取要删除的角色 ID
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	// 调用 service 执行删除操作
	if err := h.svc.DeleteRole(r.Context(), id); err != nil {
		// 删除失败，返回 400 状态码
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// 删除成功，返回空数据
	response.Success(w, nil)
}

// PermanentDeleteRole 永久删除角色（从回收站彻底删除）
// DELETE /api/v1/rbac/roles/{id}/permanent
func (h *RBACHandler) PermanentDeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	if err := h.svc.PermanentDeleteRole(r.Context(), id); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// BatchPermanentDeleteRoles 批量永久删除角色
// DELETE /api/v1/rbac/roles/batch/permanent
func (h *RBACHandler) BatchPermanentDeleteRoles(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchDeleteRolesReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	deleted, err := h.svc.BatchPermanentDeleteRoles(r.Context(), req.IDs)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"deleted": deleted})
}

// BindRolePermissions 为角色绑定权限
// PUT /api/v1/rbac/roles/{id}/permissions
func (h *RBACHandler) BindRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id") // 获取角色 ID
	if roleID == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}
	var req dto.BindRolePermissionsReq // 权限绑定请求（包含权限 ID 列表）
	// 解析并校验请求体
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 调用 service 将权限绑定到角色
	if err := h.svc.BindRolePermissions(r.Context(), roleID, req); err != nil {
		// 绑定失败，返回 400 状态码
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// 绑定成功，返回空数据
	response.Success(w, nil)
}

// GetRolePermissions 获取角色拥有的权限列表
// GET /api/v1/rbac/roles/{id}/permissions
func (h *RBACHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id") // 获取角色 ID
	if roleID == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}
	resource := r.URL.Query().Get("resource") // 可选：按资源过滤

	var permissions []*model.Permission
	var err error

	// 如果指定了 resource 参数，使用带资源筛选的方法
	if resource != "" {
		permissions, err = h.svc.GetRolePermissionsWithResource(r.Context(), roleID, resource)
	} else {
		permissions, err = h.svc.GetRolePermissions(r.Context(), roleID)
	}

	if err != nil {
		// 查询失败，返回 500 状态码
		response.Fail(w, http.StatusInternalServerError, "获取角色权限失败")
		return
	}
	// 转换为 DTO 返回
	resp := make([]*dto.PermissionResp, len(permissions))
	for i, p := range permissions {
		resp[i] = dto.ToPermissionResp(p)
	}
	response.Success(w, resp)
}

// CreatePermission 创建权限
// POST /api/v1/rbac/permissions
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

	response.Success(w, dto.ToPermissionResp(permission))
}

// GetPermission 获取单个权限详情
// GET /api/v1/rbac/permissions/{id}/detail
func (h *RBACHandler) GetPermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "权限 ID 不能为空")
		return
	}
	permission, err := h.svc.GetPermission(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "权限不存在")
		return
	}
	response.Success(w, dto.ToPermissionResp(permission))
}

// ListPermissions 获取权限列表（带分页、按资源和关键字筛选）
// GET /api/v1/rbac/permissions
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

	resp := make([]*dto.PermissionResp, len(permissions))
	for i, p := range permissions {
		resp[i] = dto.ToPermissionResp(p)
	}

	result := pagination.NewPaginatedResult(resp, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// UpdatePermission 更新权限信息
// PUT /api/v1/rbac/permissions/{id}/update
func (h *RBACHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "权限 ID 不能为空")
		return
	}
	var req dto.UpdatePermissionReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	permission, err := h.svc.UpdatePermission(r.Context(), id, req)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, dto.ToPermissionResp(permission))
}

// UpdatePermissionStatus 更改权限状态
// PUT /api/v1/rbac/permissions/{id}/status
func (h *RBACHandler) UpdatePermissionStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "权限 ID 不能为空")
		return
	}
	var req dto.UpdatePermissionStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.UpdatePermissionStatus(r.Context(), id, req.Status); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// BatchUpdatePermissionStatus 批量更改权限状态
// PUT /api/v1/rbac/permissions/batch/status
func (h *RBACHandler) BatchUpdatePermissionStatus(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchUpdatePermissionStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	updated, err := h.svc.BatchUpdatePermissionStatus(r.Context(), req.IDs, req.Status)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"updated": updated})
}

// DeletePermission 删除权限
// DELETE /api/v1/rbac/permissions/{id}/delete
func (h *RBACHandler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "权限 ID 不能为空")
		return
	}
	if err := h.svc.DeletePermission(r.Context(), id); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// BatchDeletePermissions 批量删除权限
// DELETE /api/v1/rbac/permissions/batch/delete
func (h *RBACHandler) BatchDeletePermissions(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchDeletePermissionsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	deleted, err := h.svc.BatchDeletePermissions(r.Context(), req.IDs)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"deleted": deleted})
}

// ListDeletedRoles 获取已软删除的角色列表
// GET /api/v1/rbac/roles/deleted
func (h *RBACHandler) ListDeletedRoles(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	keyword := r.URL.Query().Get("keyword")

	roles, total, err := h.svc.ListDeletedRoles(r.Context(), pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取已删除角色列表失败")
		return
	}

	resp := make([]*dto.RoleResp, len(roles))
	for i, role := range roles {
		resp[i] = dto.ToRoleResp(role)
	}

	result := pagination.NewPaginatedResult(resp, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// RestoreRole 恢复已软删除的角色
// POST /api/v1/rbac/roles/{id}/restore
func (h *RBACHandler) RestoreRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	if err := h.svc.RestoreRole(r.Context(), id); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// UpdateRoleStatus 更改角色状态（启用/禁用）
// PUT /api/v1/rbac/roles/{id}/status
func (h *RBACHandler) UpdateRoleStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	var req dto.UpdateRoleStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.UpdateRoleStatus(r.Context(), id, req.Status); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// BatchUpdateRoleStatus 批量更改角色状态
// PUT /api/v1/rbac/roles/batch/status
func (h *RBACHandler) BatchUpdateRoleStatus(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchUpdateRoleStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	updated, err := h.svc.BatchUpdateRoleStatus(r.Context(), req.IDs, req.Status)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"updated": updated})
}

// BatchDeleteRoles 批量删除角色
// DELETE /api/v1/rbac/roles/batch/delete
func (h *RBACHandler) BatchDeleteRoles(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchDeleteRolesReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	deleted, err := h.svc.BatchDeleteRoles(r.Context(), req.IDs)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"deleted": deleted})
}

// UnbindRolePermission 解绑角色单个权限
// DELETE /api/v1/rbac/roles/{id}/permissions
func (h *RBACHandler) UnbindRolePermission(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")

	if roleID == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	var req dto.UnbindRolePermissionReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.UnbindRolePermission(r.Context(), roleID, req.PermissionID); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// UnbindRolePermissions 解绑角色多个权限
// DELETE /api/v1/rbac/roles/{id}/permissions/batch
func (h *RBACHandler) UnbindRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	if roleID == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	var req dto.UnbindRolePermissionsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	unbound, err := h.svc.UnbindRolePermissions(r.Context(), roleID, req.PermissionIDs)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"unbound": unbound})
}

// ListRolePermissions 获取角色权限关系列表（分页，支持按 role_id / permission_id 过滤）
// GET /api/v1/rbac/role-permissions
func (h *RBACHandler) ListRolePermissions(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	roleID := r.URL.Query().Get("role_id")
	permissionID := r.URL.Query().Get("permission_id")

	list, total, err := h.svc.ListRolePermissions(r.Context(), pg.Page, pg.PageSize, roleID, permissionID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取角色权限关系列表失败")
		return
	}

	resp := make([]*dto.RolePermissionResp, len(list))
	for i, rp := range list {
		resp[i] = dto.ToRolePermissionResp(rp)
	}

	result := pagination.NewPaginatedResult(resp, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// BatchBindRolesPermissions 批量为多个角色绑定权限
// PUT /api/v1/rbac/roles/batch/permissions
func (h *RBACHandler) BatchBindRolesPermissions(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchBindRolesPermissionsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	bound, err := h.svc.BatchBindRolesPermissions(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"bound": bound})
}

// BatchUnbindRolesPermissions 批量解绑多个角色的权限
// DELETE /api/v1/rbac/roles/batch/permissions
func (h *RBACHandler) BatchUnbindRolesPermissions(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchUnbindRolesPermissionsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	unbound, err := h.svc.BatchUnbindRolesPermissions(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"unbound": unbound})
}

// AssignUserRoles 为用户分配角色
// POST /api/v1/rbac/user-roles/{user_id}/roles
func (h *RBACHandler) AssignUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	var req dto.AssignUserRolesReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.AssignUserRoles(r.Context(), userID, req); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// GetUserRoles 获取用户的角色列表
// GET /api/v1/rbac/user-roles/{user_id}/roles
func (h *RBACHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	roles, err := h.svc.GetUserRoles(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户角色失败")
		return
	}

	resp := make([]*dto.RoleResp, len(roles))
	for i, role := range roles {
		resp[i] = dto.ToRoleResp(role)
	}
	response.Success(w, resp)
}

// RemoveUserRole 删除用户的单个角色
// DELETE /api/v1/rbac/user-roles/{user_id}/roles/{role_id}
func (h *RBACHandler) RemoveUserRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	roleID := chi.URLParam(r, "role_id")

	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}
	if roleID == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	if err := h.svc.RemoveUserRole(r.Context(), userID, roleID); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// RemoveAllUserRoles 删除用户的所有角色
// DELETE /api/v1/rbac/user-roles/{user_id}/roles
func (h *RBACHandler) RemoveAllUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	if err := h.svc.RemoveAllUserRoles(r.Context(), userID); err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, nil)
}

// GetRoleUsers 获取拥有某角色的用户列表
// GET /api/v1/rbac/user-roles/roles/{role_id}/users
func (h *RBACHandler) GetRoleUsers(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "role_id")
	if roleID == "" {
		response.Fail(w, http.StatusBadRequest, "角色ID不能为空")
		return
	}

	userIDs, err := h.svc.GetRoleUsers(r.Context(), roleID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取角色用户列表失败")
		return
	}
	response.Success(w, map[string]interface{}{"user_ids": userIDs})
}

// ListUserRoles 获取用户-角色关系列表（分页，支持按 user_id / role_id 过滤）
// GET /api/v1/rbac/user-roles
func (h *RBACHandler) ListUserRoles(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	userID := r.URL.Query().Get("user_id")
	roleID := r.URL.Query().Get("role_id")

	list, total, err := h.svc.ListUserRoles(r.Context(), pg.Page, pg.PageSize, userID, roleID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户角色关系列表失败")
		return
	}

	resp := make([]*dto.UserRoleResp, len(list))
	for i, ur := range list {
	resp[i] = dto.ToUserRoleResp(ur)
	}

	result := pagination.NewPaginatedResult(resp, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// BatchAssignUserRoles 批量为用户分配角色
// POST /api/v1/rbac/user-roles/batch/assign
func (h *RBACHandler) BatchAssignUserRoles(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchAssignUserRolesReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	assigned, err := h.svc.BatchAssignUserRoles(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"assigned": assigned})
}

// GetStats GET /api/v1/rbac/stats - 获取 RBAC 统计（角色总数、权限总数、我的权限数）
func (h *RBACHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := contextx.GetTenantID(ctx)
	userID := contextx.GetUserID(ctx)

	roleCount, err := h.svc.CountRoles(ctx, tenantID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取角色统计失败")
		return
	}

	permCount, err := h.svc.CountPermissions(ctx)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取权限统计失败")
		return
	}

	myPermCount, err := h.svc.CountUserPermissions(ctx, userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户权限统计失败")
		return
	}

	response.Success(w, map[string]interface{}{
		"role_count":       roleCount,
		"permission_count": permCount,
		"my_permission":    myPermCount,
	})
}