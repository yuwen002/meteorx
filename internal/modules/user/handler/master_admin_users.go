package handler

import (
	"errors"
	"net/http"
	"strconv"

	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/service"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/pkg/logger"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// ============ 系统管理员管理接口 ============

// ListMasterAdmins 获取系统管理员列表（分页+搜索）
// GET /api/v1/admin/users
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

// GetMasterAdmin 获取系统管理员详情
// GET /api/v1/admin/users/{id}/detail
func (h *UserHandler) GetMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	user, err := h.svc.GetMasterAdmin(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotSystemAdmin) {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "获取系统管理员详情失败")
		}
		return
	}
	response.Success(w, user)
}

// CreateMasterAdmin 创建系统管理员
// POST /api/v1/admin/users
func (h *UserHandler) CreateMasterAdmin(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMasterAdminReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.CreateMasterAdmin(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			response.Fail(w, http.StatusConflict, "用户名已被使用")
		} else {
			response.Fail(w, http.StatusInternalServerError, "创建系统管理员失败")
		}
		return
	}
	response.Success(w, user)
}

// UpdateMasterAdmin 更新系统管理员信息
// PUT /api/v1/admin/users/{id}/update
func (h *UserHandler) UpdateMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	var req dto.UpdateMasterAdminReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.UpdateMasterAdmin(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrUserNotSystemAdmin) {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "更新系统管理员失败")
		}
		return
	}
	response.Success(w, user)
}

// DeleteMasterAdmin 删除系统管理员
// DELETE /api/v1/admin/users/{id}/delete
func (h *UserHandler) DeleteMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	err := h.svc.DeleteMasterAdmin(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotSystemAdmin) {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	response.Success(w, nil)
}

// UpdateMasterAdminStatus 更新系统管理员状态
// PUT /api/v1/admin/users/{id}/status
func (h *UserHandler) UpdateMasterAdminStatus(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	var req dto.UpdateUserStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.UpdateMasterAdminStatus(r.Context(), userID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrUserNotSystemAdmin) {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "更新系统管理员状态失败")
		}
		return
	}
	response.Success(w, nil)
}

// ListDeletedMasterAdmins GET /api/v1/admin/users/deleted - 获取已删除的系统管理员列表
func (h *UserHandler) ListDeletedMasterAdmins(w http.ResponseWriter, r *http.Request) {
	// 解析分页参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	// 解析搜索关键词
	keyword := r.URL.Query().Get("keyword")

	// 调用服务层查询
	users, total, err := h.svc.ListDeletedMasterAdmins(r.Context(), pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取已删除系统管理员列表失败")
		return
	}

	// 使用分页包返回结果
	result := pagination.NewPaginatedResult(users, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// RestoreMasterAdmin PUT /api/v1/admin/users/{id}/restore - 恢复已删除的系统管理员
func (h *UserHandler) RestoreMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	err := h.svc.RestoreMasterAdmin(r.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在或未删除")
		} else {
			response.Fail(w, http.StatusInternalServerError, "恢复系统管理员失败")
		}
		return
	}
	response.Success(w, nil)
}

// PermanentDeleteMasterAdmin DELETE /api/v1/admin/users/{id}/permanent - 永久删除系统管理员
func (h *UserHandler) PermanentDeleteMasterAdmin(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	err := h.svc.PermanentDeleteMasterAdmin(r.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "系统管理员不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "永久删除系统管理员失败")
		}
		return
	}
	response.Success(w, nil)
}

// BatchUpdateMasterAdminStatus PUT /api/v1/admin/users/batch/status - 批量更新系统管理员状态
func (h *UserHandler) BatchUpdateMasterAdminStatus(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchUpdateUserStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	updated, err := h.svc.BatchUpdateMasterAdminStatus(r.Context(), req.IDs, req.Status)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"updated": updated})
}

// BatchDeleteMasterAdmins DELETE /api/v1/admin/users/batch/delete - 批量删除系统管理员
func (h *UserHandler) BatchDeleteMasterAdmins(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchDeleteUsersReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	deleted, err := h.svc.BatchDeleteMasterAdmins(r.Context(), req.IDs)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"deleted": deleted})
}

// AdminCreateTenantUser POST /api/v1/admin/tenant-users - 系统管理员为指定租户创建用户
func (h *UserHandler) AdminCreateTenantUser(w http.ResponseWriter, r *http.Request) {
	var req dto.AdminCreateTenantUserReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.AdminCreateTenantUser(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			response.Fail(w, http.StatusConflict, "用户名已被使用")
		} else {
			response.Fail(w, http.StatusInternalServerError, "创建用户失败")
		}
		return
	}
	response.Success(w, user)
}

// AdminListTenantUsers GET /api/v1/admin/tenant-users/{tenantID}/list?page=1&page_size=10&keyword=xxx - 系统管理员获取指定租户的用户列表
func (h *UserHandler) AdminListTenantUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	if tenantID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	// 使用 pagination 包解析分页参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	// 解析搜索关键词
	keyword := r.URL.Query().Get("keyword")

	// 调用服务层查询
	users, total, err := h.svc.AdminListTenantUsers(r.Context(), tenantID, pg.Page, pg.PageSize, keyword)
	if err != nil {
		logger.Errorf("[UserHandler] AdminListTenantUsers failed: %v", err)
		response.Fail(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}

	// 使用分页包返回结果
	result := pagination.NewPaginatedResult(users, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// AdminListAllTenantUsers GET /api/v1/admin/tenant-users/all - 系统管理员获取所有租户用户列表（不包括系统管理员）
func (h *UserHandler) AdminListAllTenantUsers(w http.ResponseWriter, r *http.Request) {
	// 解析分页参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	// 解析搜索关键词
	keyword := r.URL.Query().Get("keyword")

	// 调用服务层查询
	users, total, err := h.svc.AdminListAllTenantUsers(r.Context(), pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}

	// 使用分页包返回结果
	result := pagination.NewPaginatedResult(users, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// AdminUpdateTenantUser PUT /api/v1/admin/tenant-users/{tenantID}/{userID}/update - 系统管理员更新指定租户的用户
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
		if errors.Is(err, service.ErrUserNotInTenant) {
			response.Fail(w, http.StatusBadRequest, "用户不属于指定租户")
		} else {
			response.Fail(w, http.StatusInternalServerError, "更新用户失败")
		}
		return
	}
	response.Success(w, user)
}

// AdminDeleteTenantUser DELETE /api/v1/admin/tenant-users/{tenantID}/{userID}/delete - 系统管理员删除指定租户的用户
func (h *UserHandler) AdminDeleteTenantUser(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	userID := chi.URLParam(r, "userID")
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID和用户ID不能为空")
		return
	}

	if err := h.svc.AdminDeleteTenantUser(r.Context(), tenantID, userID); err != nil {
		if errors.Is(err, service.ErrUserNotInTenant) {
			response.Fail(w, http.StatusBadRequest, "用户不属于指定租户")
		} else {
			response.Fail(w, http.StatusInternalServerError, "删除用户失败")
		}
		return
	}
	response.Success(w, nil)
}

// AdminListDeletedTenantUsers GET /api/v1/admin/tenant-users/{tenantID}/deleted - 系统管理员获取指定租户的已删除用户列表（回收站）
func (h *UserHandler) AdminListDeletedTenantUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	if tenantID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	keyword := r.URL.Query().Get("keyword")

	users, total, err := h.svc.AdminListDeletedTenantUsers(r.Context(), tenantID, pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取已删除用户列表失败")
		return
	}

	result := pagination.NewPaginatedResult(users, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// AdminListAllDeletedTenantUsers GET /api/v1/admin/tenant-users/deleted/all - 系统管理员获取所有租户的已删除用户列表（回收站）
func (h *UserHandler) AdminListAllDeletedTenantUsers(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	keyword := r.URL.Query().Get("keyword")

	users, total, err := h.svc.AdminListAllDeletedTenantUsers(r.Context(), pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取已删除用户列表失败")
		return
	}

	result := pagination.NewPaginatedResult(users, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// AdminRestoreTenantUser PUT /api/v1/admin/tenant-users/{tenantID}/{userID}/restore - 系统管理员恢复已删除的租户用户
func (h *UserHandler) AdminRestoreTenantUser(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	userID := chi.URLParam(r, "userID")
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID和用户ID不能为空")
		return
	}

	err := h.svc.AdminRestoreTenantUser(r.Context(), tenantID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "用户不存在或未删除")
		} else {
			response.Fail(w, http.StatusInternalServerError, "恢复用户失败")
		}
		return
	}
	response.Success(w, nil)
}

// AdminPermanentDeleteTenantUser DELETE /api/v1/admin/tenant-users/{tenantID}/{userID}/permanent - 系统管理员永久删除租户用户
func (h *UserHandler) AdminPermanentDeleteTenantUser(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	userID := chi.URLParam(r, "userID")
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID和用户ID不能为空")
		return
	}

	err := h.svc.AdminPermanentDeleteTenantUser(r.Context(), tenantID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "用户不存在或未删除")
		} else {
			response.Fail(w, http.StatusInternalServerError, "永久删除用户失败")
		}
		return
	}
	response.Success(w, nil)
}

// AdminUpdateTenantUserStatus PUT /api/v1/admin/tenant-users/{tenantID}/{userID}/status - 系统管理员更新指定租户的用户状态
func (h *UserHandler) AdminUpdateTenantUserStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	userID := chi.URLParam(r, "userID")
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID和用户ID不能为空")
		return
	}

	var req dto.UpdateUserStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.AdminUpdateTenantUserStatus(r.Context(), tenantID, userID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrUserNotInTenant) {
			response.Fail(w, http.StatusBadRequest, "用户不属于指定租户")
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "用户不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "更新用户状态失败")
		}
		return
	}
	response.Success(w, nil)
}

// AdminBatchUpdateTenantUserStatus PUT /api/v1/admin/tenant-users/{tenantID}/batch/status - 系统管理员批量更新指定租户的用户状态
func (h *UserHandler) AdminBatchUpdateTenantUserStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	if tenantID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	var req dto.BatchUpdateUserStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	updated, err := h.svc.AdminBatchUpdateTenantUserStatus(r.Context(), tenantID, req.IDs, req.Status)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"updated": updated})
}

// AdminBatchDeleteTenantUsers DELETE /api/v1/admin/tenant-users/{tenantID}/batch/delete - 系统管理员批量删除指定租户的用户
func (h *UserHandler) AdminBatchDeleteTenantUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	if tenantID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	var req dto.BatchDeleteUsersReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	deleted, err := h.svc.AdminBatchDeleteTenantUsers(r.Context(), tenantID, req.IDs)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(w, map[string]interface{}{"deleted": deleted})
}

// AdminResetTenantUserPassword PUT /api/v1/admin/tenant-users/{tenantID}/{userID}/reset-password - 系统管理员重置指定租户用户的密码
func (h *UserHandler) AdminResetTenantUserPassword(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	userID := chi.URLParam(r, "userID")
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID和用户ID不能为空")
		return
	}

	var req dto.ResetPasswordReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		response.Fail(w, http.StatusBadRequest, "新密码与确认密码不一致")
		return
	}

	if err := h.svc.AdminResetTenantUserPassword(r.Context(), tenantID, userID, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrUserNotInTenant) {
			response.Fail(w, http.StatusBadRequest, "用户不属于指定租户")
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "用户不存在")
		} else {
			response.Fail(w, http.StatusInternalServerError, "重置密码失败")
		}
		return
	}
	response.Success(w, nil)
}
