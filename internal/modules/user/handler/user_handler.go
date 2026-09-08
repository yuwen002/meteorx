// Package handler 提供用户管理 HTTP 处理器
package handler

import (
	"context"
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
	"gorm.io/gorm"
)

// UserService 用户服务接口（handler 依赖的最小业务面，便于测试注入桩；
// *service.UserService 完整实现该接口，生产装配零改动）
type UserService interface {
	// 租户内用户管理
	ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword string, status *int) ([]*dto.UserResp, int64, error)
	GetByID(ctx context.Context, userID string) (*dto.UserResp, error)
	BelongsToTenant(ctx context.Context, userID, tenantID string) bool
	Create(ctx context.Context, tenantID string, req dto.CreateUserReq) (*dto.UserResp, error)
	Update(ctx context.Context, userID string, req dto.UpdateUserReq) (*dto.UserResp, error)
	Delete(ctx context.Context, userID string) error
	CountByTenant(ctx context.Context, tenantID string) (int64, error)
	CountAllUsers(ctx context.Context) (int64, error)
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	ResetPassword(ctx context.Context, userID, newPassword string) error
	ListDeletedUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error)
	RestoreUser(ctx context.Context, tenantID, userID string) error
	PermanentDeleteUser(ctx context.Context, tenantID, userID string) error
	// 系统管理员管理
	ListMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error)
	GetMasterAdmin(ctx context.Context, userID string) (*dto.UserResp, error)
	CreateMasterAdmin(ctx context.Context, req dto.CreateMasterAdminReq) (*dto.UserResp, error)
	UpdateMasterAdmin(ctx context.Context, userID string, req dto.UpdateMasterAdminReq) (*dto.UserResp, error)
	DeleteMasterAdmin(ctx context.Context, userID string) error
	UpdateMasterAdminStatus(ctx context.Context, userID string, status int) error
	ListDeletedMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error)
	RestoreMasterAdmin(ctx context.Context, userID string) error
	PermanentDeleteMasterAdmin(ctx context.Context, userID string) error
	BatchUpdateMasterAdminStatus(ctx context.Context, ids []string, status int) (int64, error)
	BatchDeleteMasterAdmins(ctx context.Context, ids []string) (int64, error)
	// 系统管理员跨租户用户管理
	AdminCreateTenantUser(ctx context.Context, req dto.AdminCreateTenantUserReq) (*dto.UserResp, error)
	AdminListTenantUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error)
	AdminListAllTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error)
	AdminUpdateTenantUser(ctx context.Context, tenantID, userID string, req dto.UpdateUserReq) (*dto.UserResp, error)
	AdminDeleteTenantUser(ctx context.Context, tenantID, userID string) error
	AdminListDeletedTenantUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error)
	AdminListAllDeletedTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error)
	AdminRestoreTenantUser(ctx context.Context, tenantID, userID string) error
	AdminPermanentDeleteTenantUser(ctx context.Context, tenantID, userID string) error
	AdminUpdateTenantUserStatus(ctx context.Context, tenantID, userID string, status int) error
	AdminBatchUpdateTenantUserStatus(ctx context.Context, tenantID string, ids []string, status int) (int64, error)
	AdminBatchDeleteTenantUsers(ctx context.Context, tenantID string, ids []string) (int64, error)
	AdminResetTenantUserPassword(ctx context.Context, tenantID, userID, newPassword string) error
}

// UserHandler 用户管理处理器
type UserHandler struct {
	svc UserService
}

// NewUserHandler 创建用户管理处理器
func NewUserHandler(svc UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// ListUsers 获取租户下的用户列表（分页+搜索）
// GET /api/v1/users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 解析分页参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	// 解析搜索关键字
	keyword := r.URL.Query().Get("keyword")

	// 解析状态筛选
	statusStr := r.URL.Query().Get("status")
	var status *int
	if statusStr != "" {
		s, _ := strconv.Atoi(statusStr)
		status = &s
	}

	users, total, err := h.svc.ListByTenant(r.Context(), tenantID, pg.Page, pg.PageSize, keyword, status)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户列表失败")
		return
	}

	// 使用分页包返回结果
	result := pagination.NewPaginatedResult(users, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// GetUser 获取用户详情
// GET /api/v1/users/{id}/detail
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

// CreateUser 创建新用户
// POST /api/v1/users
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
		if errors.Is(err, service.ErrUsernameExists) {
			response.Fail(w, http.StatusConflict, "用户名已被使用")
		} else {
			response.Fail(w, http.StatusInternalServerError, "创建用户失败")
		}
		return
	}
	response.Success(w, user)
}

// UpdateUser 更新用户信息
// PUT /api/v1/users/{id}/update
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

// DeleteUser 删除用户
// DELETE /api/v1/users/{id}/delete
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

// GetStats 获取当前租户用户总数
// GET /api/v1/profile/stats
func (h *UserHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	count, err := h.svc.CountByTenant(r.Context(), tenantID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户统计失败")
		return
	}
	response.Success(w, map[string]interface{}{"user_count": count})
}

// GetAllStats 获取所有用户总数（跨租户，仅限管理员）
// GET /api/v1/admin/stats
func (h *UserHandler) GetAllStats(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.CountAllUsers(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户统计失败")
		return
	}
	response.Success(w, map[string]interface{}{"user_count": count})
}

// GetProfile GET /api/v1/profile - 获取当前用户个人信息
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	user, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户信息失败")
		return
	}
	response.Success(w, user)
}

// UpdateProfile PUT /api/v1/profile - 更新当前用户个人信息（仅限昵称、邮箱等非敏感信息）
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req dto.UpdateUserReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 用户只能修改自己的非敏感信息（昵称、邮箱），不能修改角色和状态
	req.RoleIDs = nil
	req.Status = nil

	user, err := h.svc.Update(r.Context(), userID, req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "更新用户信息失败")
		return
	}
	response.Success(w, user)
}

// ChangePassword PUT /api/v1/profile/password - 修改当前用户密码
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req dto.ChangePasswordReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		response.Fail(w, http.StatusBadRequest, "新密码与确认密码不一致")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrWrongOldPassword) {
			response.Fail(w, http.StatusBadRequest, "原密码错误")
		} else {
			response.Fail(w, http.StatusInternalServerError, "修改密码失败")
		}
		return
	}
	response.Success(w, nil)
}

// ResetPassword PUT /api/v1/users/{id}/reset-password - 管理员重置用户密码
func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
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

	var req dto.ResetPasswordReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		response.Fail(w, http.StatusBadRequest, "新密码与确认密码不一致")
		return
	}

	if err := h.svc.ResetPassword(r.Context(), userID, req.NewPassword); err != nil {
		response.Fail(w, http.StatusInternalServerError, "重置密码失败")
		return
	}
	response.Success(w, nil)
}

// ListDeletedUsers GET /api/v1/users/deleted
func (h *UserHandler) ListDeletedUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	keyword := r.URL.Query().Get("keyword")

	users, total, err := h.svc.ListDeletedUsers(r.Context(), tenantID, pg.Page, pg.PageSize, keyword)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "get deleted users failed")
		return
	}

	result := pagination.NewPaginatedResult(users, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// RestoreUser PUT /api/v1/users/{id}/restore
func (h *UserHandler) RestoreUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "user id required")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	err := h.svc.RestoreUser(r.Context(), tenantID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "user not found")
		} else {
			response.Fail(w, http.StatusInternalServerError, "restore user failed")
		}
		return
	}
	response.Success(w, nil)
}

// PermanentDeleteUser DELETE /api/v1/users/{id}/permanent
func (h *UserHandler) PermanentDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "user id required")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "tenant not found")
		return
	}

	err := h.svc.PermanentDeleteUser(r.Context(), tenantID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(w, http.StatusNotFound, "user not found")
		} else {
			response.Fail(w, http.StatusInternalServerError, "permanent delete user failed")
		}
		return
	}
	response.Success(w, nil)
}
