package handler_test

import (
	"context"

	"meteorx/internal/modules/user/dto"
)

// stubUserService 桩实现 handler.UserService：预设返回并捕获最近入参。
type stubUserService struct {
	// 预设返回
	Err      error
	Resp     *dto.UserResp
	List     []*dto.UserResp
	Total    int64
	Belongs  bool
	Affected int64

	// 捕获入参
	GotTenantID string
	GotUserID   string
	GotKeyword  string
	GotPage     int
	GotPageSize int
	GotStatus   *int
	GotIDs      []string
	GotStatusN  int
	GotOldPass  string
	GotNewPass  string
	GotCreateReq   *dto.CreateUserReq
	GotUpdateReq   *dto.UpdateUserReq
	GotCreateAdmin *dto.CreateMasterAdminReq
	GotUpdateAdmin *dto.UpdateMasterAdminReq
	GotAdminCreateTenant *dto.AdminCreateTenantUserReq
}

// 租户内用户管理

func (s *stubUserService) ListByTenant(_ context.Context, tenantID string, page, pageSize int, keyword string, status *int) ([]*dto.UserResp, int64, error) {
	s.GotTenantID, s.GotPage, s.GotPageSize, s.GotKeyword, s.GotStatus = tenantID, page, pageSize, keyword, status
	return s.List, s.Total, s.Err
}

func (s *stubUserService) GetByID(_ context.Context, userID string) (*dto.UserResp, error) {
	s.GotUserID = userID
	return s.Resp, s.Err
}

func (s *stubUserService) BelongsToTenant(_ context.Context, userID, tenantID string) bool {
	s.GotUserID, s.GotTenantID = userID, tenantID
	return s.Belongs
}

func (s *stubUserService) Create(_ context.Context, tenantID string, req dto.CreateUserReq) (*dto.UserResp, error) {
	s.GotTenantID, s.GotCreateReq = tenantID, &req
	return s.Resp, s.Err
}

func (s *stubUserService) Update(_ context.Context, userID string, req dto.UpdateUserReq) (*dto.UserResp, error) {
	s.GotUserID, s.GotUpdateReq = userID, &req
	return s.Resp, s.Err
}

func (s *stubUserService) Delete(_ context.Context, userID string) error {
	s.GotUserID = userID
	return s.Err
}

func (s *stubUserService) CountByTenant(_ context.Context, tenantID string) (int64, error) {
	s.GotTenantID = tenantID
	return s.Total, s.Err
}

func (s *stubUserService) CountAllUsers(_ context.Context) (int64, error) {
	return s.Total, s.Err
}

func (s *stubUserService) ChangePassword(_ context.Context, userID, oldPassword, newPassword string) error {
	s.GotUserID, s.GotOldPass, s.GotNewPass = userID, oldPassword, newPassword
	return s.Err
}

func (s *stubUserService) ResetPassword(_ context.Context, userID, newPassword string) error {
	s.GotUserID, s.GotNewPass = userID, newPassword
	return s.Err
}

func (s *stubUserService) ListDeletedUsers(_ context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	s.GotTenantID, s.GotPage, s.GotPageSize, s.GotKeyword = tenantID, page, pageSize, keyword
	return s.List, s.Total, s.Err
}

func (s *stubUserService) RestoreUser(_ context.Context, tenantID, userID string) error {
	s.GotTenantID, s.GotUserID = tenantID, userID
	return s.Err
}

func (s *stubUserService) PermanentDeleteUser(_ context.Context, tenantID, userID string) error {
	s.GotTenantID, s.GotUserID = tenantID, userID
	return s.Err
}

// 系统管理员管理

func (s *stubUserService) ListMasterAdmins(_ context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	s.GotPage, s.GotPageSize, s.GotKeyword = page, pageSize, keyword
	return s.List, s.Total, s.Err
}

func (s *stubUserService) GetMasterAdmin(_ context.Context, userID string) (*dto.UserResp, error) {
	s.GotUserID = userID
	return s.Resp, s.Err
}

func (s *stubUserService) CreateMasterAdmin(_ context.Context, req dto.CreateMasterAdminReq) (*dto.UserResp, error) {
	s.GotCreateAdmin = &req
	return s.Resp, s.Err
}

func (s *stubUserService) UpdateMasterAdmin(_ context.Context, userID string, req dto.UpdateMasterAdminReq) (*dto.UserResp, error) {
	s.GotUserID, s.GotUpdateAdmin = userID, &req
	return s.Resp, s.Err
}

func (s *stubUserService) DeleteMasterAdmin(_ context.Context, userID string) error {
	s.GotUserID = userID
	return s.Err
}

func (s *stubUserService) UpdateMasterAdminStatus(_ context.Context, userID string, status int) error {
	s.GotUserID, s.GotStatusN = userID, status
	return s.Err
}

func (s *stubUserService) ListDeletedMasterAdmins(_ context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	s.GotPage, s.GotPageSize, s.GotKeyword = page, pageSize, keyword
	return s.List, s.Total, s.Err
}

func (s *stubUserService) RestoreMasterAdmin(_ context.Context, userID string) error {
	s.GotUserID = userID
	return s.Err
}

func (s *stubUserService) PermanentDeleteMasterAdmin(_ context.Context, userID string) error {
	s.GotUserID = userID
	return s.Err
}

func (s *stubUserService) BatchUpdateMasterAdminStatus(_ context.Context, ids []string, status int) (int64, error) {
	s.GotIDs, s.GotStatusN = ids, status
	return s.Affected, s.Err
}

func (s *stubUserService) BatchDeleteMasterAdmins(_ context.Context, ids []string) (int64, error) {
	s.GotIDs = ids
	return s.Affected, s.Err
}

// 系统管理员跨租户用户管理

func (s *stubUserService) AdminCreateTenantUser(_ context.Context, req dto.AdminCreateTenantUserReq) (*dto.UserResp, error) {
	s.GotAdminCreateTenant = &req
	return s.Resp, s.Err
}

func (s *stubUserService) AdminListTenantUsers(_ context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	s.GotTenantID, s.GotPage, s.GotPageSize, s.GotKeyword = tenantID, page, pageSize, keyword
	return s.List, s.Total, s.Err
}

func (s *stubUserService) AdminListAllTenantUsers(_ context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	s.GotPage, s.GotPageSize, s.GotKeyword = page, pageSize, keyword
	return s.List, s.Total, s.Err
}

func (s *stubUserService) AdminUpdateTenantUser(_ context.Context, tenantID, userID string, req dto.UpdateUserReq) (*dto.UserResp, error) {
	s.GotTenantID, s.GotUserID, s.GotUpdateReq = tenantID, userID, &req
	return s.Resp, s.Err
}

func (s *stubUserService) AdminDeleteTenantUser(_ context.Context, tenantID, userID string) error {
	s.GotTenantID, s.GotUserID = tenantID, userID
	return s.Err
}

func (s *stubUserService) AdminListDeletedTenantUsers(_ context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	s.GotTenantID, s.GotPage, s.GotPageSize, s.GotKeyword = tenantID, page, pageSize, keyword
	return s.List, s.Total, s.Err
}

func (s *stubUserService) AdminListAllDeletedTenantUsers(_ context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	s.GotPage, s.GotPageSize, s.GotKeyword = page, pageSize, keyword
	return s.List, s.Total, s.Err
}

func (s *stubUserService) AdminRestoreTenantUser(_ context.Context, tenantID, userID string) error {
	s.GotTenantID, s.GotUserID = tenantID, userID
	return s.Err
}

func (s *stubUserService) AdminPermanentDeleteTenantUser(_ context.Context, tenantID, userID string) error {
	s.GotTenantID, s.GotUserID = tenantID, userID
	return s.Err
}

func (s *stubUserService) AdminUpdateTenantUserStatus(_ context.Context, tenantID, userID string, status int) error {
	s.GotTenantID, s.GotUserID, s.GotStatusN = tenantID, userID, status
	return s.Err
}

func (s *stubUserService) AdminBatchUpdateTenantUserStatus(_ context.Context, tenantID string, ids []string, status int) (int64, error) {
	s.GotTenantID, s.GotIDs, s.GotStatusN = tenantID, ids, status
	return s.Affected, s.Err
}

func (s *stubUserService) AdminBatchDeleteTenantUsers(_ context.Context, tenantID string, ids []string) (int64, error) {
	s.GotTenantID, s.GotIDs = tenantID, ids
	return s.Affected, s.Err
}

func (s *stubUserService) AdminResetTenantUserPassword(_ context.Context, tenantID, userID, newPassword string) error {
	s.GotTenantID, s.GotUserID, s.GotNewPass = tenantID, userID, newPassword
	return s.Err
}
