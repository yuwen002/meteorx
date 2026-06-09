package handler

import (
	"errors"
	"net/http"
	"strconv"

	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/service"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// ListUsers GET /api/v1/users - 获取租户下的用户列表
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	users, err := h.svc.ListByTenant(r.Context(), tenantID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}
	response.Success(w, users)
}

// GetUser GET /api/v1/users/{id} - 获取用户详情
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 检查用户是否属于当前租户
	if !h.svc.BelongsToTenant(r.Context(), userID, tenantID) {
		response.Fail(w, http.StatusForbidden, "无权查看该用户")
		return
	}

	user, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户详情失败")
		return
	}
	response.Success(w, user)
}

// CreateUser POST /api/v1/users - 创建新用户
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	var req dto.CreateUserReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.Create(r.Context(), tenantID, req)
	if err != nil {
		if errors.Is(err, errors.New("用户名已被使用")) {
			response.Fail(w, http.StatusConflict, "用户名已被使用")
		} else {
			response.Fail(w, http.StatusInternalServerError, "创建用户失败")
		}
		return
	}
	response.Success(w, user)
}

// UpdateUser PUT /api/v1/users/{id} - 更新用户信息
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 检查用户是否属于当前租户
	if !h.svc.BelongsToTenant(r.Context(), userID, tenantID) {
		response.Fail(w, http.StatusForbidden, "无权操作该用户")
		return
	}

	var req dto.UpdateUserReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.Update(r.Context(), userID, req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "更新用户失败")
		return
	}
	response.Success(w, user)
}

// DeleteUser DELETE /api/v1/users/{id} - 删除用户
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 检查用户是否属于当前租户
	if !h.svc.BelongsToTenant(r.Context(), userID, tenantID) {
		response.Fail(w, http.StatusForbidden, "无权操作该用户")
		return
	}

	if err := h.svc.Delete(r.Context(), userID); err != nil {
		response.Fail(w, http.StatusInternalServerError, "删除用户失败")
		return
	}
	response.Success(w, nil)
}

// ============ 系统管理员管理接口 ============

// ListMasterAdmins GET /api/v1/admin/users?page=1&page_size=10&keyword=xxx - 获取系统管理员列表
func (h *UserHandler) ListMasterAdmins(w http.ResponseWriter, r *http.Request) {
	// 1. 解析分页参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	// 2. 解析搜索关键词
	keyword := r.URL.Query().Get("keyword")

	// 3. 调用服务层查询
	list, total, err := h.svc.ListMasterAdmins(r.Context(), pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取系统管理员列表失败")
		return
	}

	// 4. 使用分页包返回结果
	result := pagination.NewPaginatedResult(list, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// GetMasterAdmin GET /api/v1/admin/users/{id} - 获取系统管理员详情
func (h *UserHandler) GetMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	user, err := h.svc.GetMasterAdmin(r.Context(), userID)
	if err != nil {
		if err.Error() == "用户不是系统管理员" {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "获取系统管理员详情失败")
		}
		return
	}
	response.Success(w, user)
}

// CreateMasterAdmin POST /api/v1/admin/users - 创建系统管理员
func (h *UserHandler) CreateMasterAdmin(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMasterAdminReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.CreateMasterAdmin(r.Context(), req)
	if err != nil {
		if err.Error() == "用户名已被使用" {
			response.Fail(w, http.StatusConflict, "用户名已被使用")
		} else {
			response.Fail(w, http.StatusInternalServerError, "创建系统管理员失败")
		}
		return
	}
	response.Success(w, user)
}

// UpdateMasterAdmin PUT /api/v1/admin/users/{id} - 更新系统管理员信息
func (h *UserHandler) UpdateMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	var req dto.UpdateUserReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.UpdateMasterAdmin(r.Context(), userID, req)
	if err != nil {
		if err.Error() == "用户不是系统管理员" {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "更新系统管理员失败")
		}
		return
	}
	response.Success(w, user)
}

// DeleteMasterAdmin DELETE /api/v1/admin/users/{id} - 删除系统管理员
func (h *UserHandler) DeleteMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	err := h.svc.DeleteMasterAdmin(r.Context(), userID)
	if err != nil {
		if err.Error() == "用户不是系统管理员" {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "删除系统管理员失败")
		}
		return
	}
	response.Success(w, nil)
}

// AdminCreateTenantUser POST /api/v1/admin/tenant-users - 系统管理员为指定租户创建用户
func (h *UserHandler) AdminCreateTenantUser(w http.ResponseWriter, r *http.Request) {
	var req dto.AdminCreateTenantUserReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.AdminCreateTenantUser(r.Context(), req)
	if err != nil {
		if err.Error() == "用户名已被使用" {
			response.Fail(w, http.StatusConflict, "用户名已被使用")
		} else {
			response.Fail(w, http.StatusInternalServerError, "创建用户失败")
		}
		return
	}
	response.Success(w, user)
}

// AdminListTenantUsers GET /api/v1/admin/tenant-users/{tenantID} - 系统管理员获取指定租户的用户列表
func (h *UserHandler) AdminListTenantUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	if tenantID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}

	users, err := h.svc.AdminListTenantUsers(r.Context(), tenantID, page, pageSize)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}
	response.Success(w, users)
}

// AdminUpdateTenantUser PUT /api/v1/admin/tenant-users/{tenantID}/{userID} - 系统管理员更新指定租户的用户
func (h *UserHandler) AdminUpdateTenantUser(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	userID := chi.URLParam(r, "userID")
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID和用户ID不能为空")
		return
	}

	var req dto.UpdateUserReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.AdminUpdateTenantUser(r.Context(), tenantID, userID, req)
	if err != nil {
		if err.Error() == "用户不属于指定租户" {
			response.Fail(w, http.StatusBadRequest, "用户不属于指定租户")
		} else {
			response.Fail(w, http.StatusInternalServerError, "更新用户失败")
		}
		return
	}
	response.Success(w, user)
}

// AdminDeleteTenantUser DELETE /api/v1/admin/tenant-users/{tenantID}/{userID} - 系统管理员删除指定租户的用户
func (h *UserHandler) AdminDeleteTenantUser(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	userID := chi.URLParam(r, "userID")
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID和用户ID不能为空")
		return
	}

	if err := h.svc.AdminDeleteTenantUser(r.Context(), tenantID, userID); err != nil {
		if err.Error() == "用户不属于指定租户" {
			response.Fail(w, http.StatusBadRequest, "用户不属于指定租户")
		} else {
			response.Fail(w, http.StatusInternalServerError, "删除用户失败")
		}
		return
	}
	response.Success(w, nil)
}
