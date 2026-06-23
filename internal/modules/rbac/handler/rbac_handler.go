// internal/modules/rbac/handler/rbac_handler.go
// RBAC 角色权限管理模块的 HTTP 处理器层
// 负责接收和解析 HTTP 请求，调用业务逻辑层（service）处理，
// 并将处理结果以统一的响应格式返回给客户端。
package handler

import (
	"net/http"
	"strconv"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/rbac/dto"
	"meteorx/internal/modules/rbac/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

// RBACHandler 角色权限管理处理器结构体
// 封装了 RBACService 服务引用，用于处理所有角色和权限相关的 HTTP 请求
type RBACHandler struct {
	svc *service.RBACService // RBAC 业务逻辑服务实例
}

// NewRBACHandler 创建并返回一个新的 RBACHandler 实例
// 采用依赖注入的方式接收 RBACService，便于单元测试和模块解耦
// svc: 已初始化的 RBACService 指针
// 返回: 初始化后的 RBACHandler 指针
func NewRBACHandler(svc *service.RBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

// CreateRole 创建角色
// POST /api/v1/rbac/roles
// 从请求体中解析角色创建参数，支持指定 tenant_id（为空则创建系统级角色）
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
// 根据 URL 路径参数中的角色 ID 查询并返回角色详细信息
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
// 支持通过查询参数：page、page_size、keyword、tenant_id 进行分页、模糊搜索和租户过滤
// tenant_id 为空时返回所有角色，指定时仅返回该租户下的角色
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

// UpdateRole 更新角色信息
// PUT /api/v1/rbac/roles/{id}/update
// 根据角色 ID 和请求体参数更新角色的属性
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
// 根据角色 ID 删除角色（逻辑删除或物理删除，取决于 service 层实现）
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

// BindRolePermissions 为角色绑定权限
// PUT /api/v1/rbac/roles/{id}/permissions
// 将一组权限 ID 关联到指定角色，实现角色与权限的多对多关系
func (h *RBACHandler) BindRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")    // 获取角色 ID
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
// 根据角色 ID 查询该角色被授予的所有权限
func (h *RBACHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id") // 获取角色 ID
	// 调用 service 获取角色关联的权限列表
	permissions, err := h.svc.GetRolePermissions(r.Context(), roleID)
	if err != nil {
		// 查询失败，返回 500 状态码
		response.Fail(w, http.StatusInternalServerError, "获取角色权限失败")
		return
	}
	// 返回权限列表
	response.Success(w, permissions)
}

// CreatePermission 创建权限
// POST /api/v1/rbac/permissions
// 创建新的权限条目，权限通常描述对某个资源的特定操作
func (h *RBACHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePermissionReq // 权限创建请求结构体
	// 解析并校验 JSON 请求体
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 调用 service 创建权限
	permission, err := h.svc.CreatePermission(r.Context(), req)
	if err != nil {
		// 创建失败，返回 400 状态码
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	// 创建成功，返回权限数据
	response.Success(w, permission)
}

// GetPermission 获取单个权限详情
// GET /api/v1/rbac/permissions/{id}
// 根据权限 ID 查询权限的详细信息
func (h *RBACHandler) GetPermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id") // 从 URL 获取权限 ID
	// 调用 service 根据 ID 查询权限
	permission, err := h.svc.GetPermission(r.Context(), id)
	if err != nil {
		// 权限不存在，返回 404 状态码
		response.Fail(w, http.StatusNotFound, "权限不存在")
		return
	}
	// 返回权限详情
	response.Success(w, permission)
}

// ListPermissions 获取权限列表（带分页、按资源和关键字筛选）
// GET /api/v1/rbac/permissions
// 支持查询参数：page、page_size、resource（按资源过滤）、keyword（关键字搜索）
func (h *RBACHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	// 读取分页参数并转换为整数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize) // 初始化分页对象

	resource := r.URL.Query().Get("resource") // 按资源类型过滤权限
	keyword := r.URL.Query().Get("keyword")   // 按关键字模糊搜索权限

	// 调用 service 获取权限列表，返回权限数据、总数和错误
	permissions, total, err := h.svc.ListPermissions(r.Context(), pg.Page, pg.PageSize, resource, keyword)
	if err != nil {
		// 查询失败，返回 500 状态码
		response.Fail(w, http.StatusInternalServerError, "获取权限列表失败")
		return
	}

	// 封装分页结果并返回给客户端
	result := pagination.NewPaginatedResult(permissions, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// UpdatePermission 更新权限信息
// PUT /api/v1/rbac/permissions/{id}
// 根据权限 ID 和请求体参数更新权限的属性
func (h *RBACHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")     // 获取要更新的权限 ID
	var req dto.UpdatePermissionReq // 权限更新请求结构体
	// 解析并校验请求体
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 调用 service 执行更新操作
	if err := h.svc.UpdatePermission(r.Context(), id, req); err != nil {
		// 更新失败，返回 400 状态码
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// 更新成功，返回空数据
	response.Success(w, nil)
}

// DeletePermission 删除权限
// DELETE /api/v1/rbac/permissions/{id}
// 根据权限 ID 删除权限条目
func (h *RBACHandler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id") // 获取要删除的权限 ID
	// 调用 service 执行删除操作
	if err := h.svc.DeletePermission(r.Context(), id); err != nil {
		// 删除失败，返回 400 状态码
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	// 删除成功，返回空数据
	response.Success(w, nil)
}

// ListDeletedRoles 获取已软删除的角色列表
// GET /api/v1/rbac/roles/deleted
// 支持查询参数：page、page_size、keyword
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
