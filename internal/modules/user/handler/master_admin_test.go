package handler_test

import (
	"errors"
	"net/http"
	"testing"

	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/handler"
	"meteorx/internal/modules/user/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func rawHandler(stub *stubUserService) *handler.UserHandler {
	return handler.NewUserHandler(stub)
}

// ============ 系统管理员 CRUD ============

func TestListMasterAdmins_Success(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("m1")}, Total: 1}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/users?page=1&page_size=10&keyword=root", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, stub.GotPage)
	assert.Equal(t, 10, stub.GotPageSize)
	assert.Equal(t, "root", stub.GotKeyword)
}

func TestListMasterAdmins_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("db down")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/users", "")

	assertFailCode(t, w, http.StatusInternalServerError, "获取系统管理员列表失败")
}

func TestGetMasterAdmin_MissingID_Returns400(t *testing.T) {
	stub := &stubUserService{}
	h := rawHandler(stub)

	w := doRaw(h.GetMasterAdmin)

	assertFailCode(t, w, http.StatusBadRequest, "用户ID不能为空")
}

func TestGetMasterAdmin_NotSystemAdmin_Returns404(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUserNotSystemAdmin}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/users/u1/detail", "")

	assertFailCode(t, w, http.StatusNotFound, "系统管理员不存在")
}

func TestGetMasterAdmin_Success(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("u1")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/users/u1/detail", "")

	var out struct {
		Data userRespBody `json:"data"`
	}
	requireOK(t, w, &out)
	assert.Equal(t, "u1", out.Data.ID)
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestCreateMasterAdmin_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/admin/users", `{"username":"root2","password":"123456"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotCreateAdmin)
}

func TestCreateMasterAdmin_UsernameExists_Returns409(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUsernameExists}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/admin/users", `{"username":"root2","password":"123456","nickname":"root2"}`)

	assertFailCode(t, w, http.StatusConflict, "用户名已被使用")
}

func TestCreateMasterAdmin_Success_ForwardsRoleID(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("m-new")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/admin/users", `{"username":"root2","password":"123456","nickname":"root2","role_id":"role-super"}`)

	var out struct {
		Data userRespBody `json:"data"`
	}
	requireOK(t, w, &out)
	assert.Equal(t, "m-new", out.Data.ID)
	require.NotNil(t, stub.GotCreateAdmin)
	assert.Equal(t, "role-super", stub.GotCreateAdmin.RoleID)
}

func TestUpdateMasterAdmin_NotSystemAdmin_Returns404(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUserNotSystemAdmin}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/u1/update", `{"nickname":"x"}`)

	assertFailCode(t, w, http.StatusNotFound, "系统管理员不存在")
}

func TestUpdateMasterAdmin_Success_ForwardsFields(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("u1")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/u1/update", `{"nickname":"new","role_id":"role-super","status":1}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotUpdateAdmin)
	assert.Equal(t, "new", stub.GotUpdateAdmin.Nickname)
	assert.Equal(t, "role-super", stub.GotUpdateAdmin.RoleID)
	stat := 1
	assert.Equal(t, &stat, stub.GotUpdateAdmin.Status)
}

func TestDeleteMasterAdmin_NotSystemAdmin_Returns404(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUserNotSystemAdmin}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/users/u1/delete", "")

	assertFailCode(t, w, http.StatusNotFound, "系统管理员不存在")
}

func TestDeleteMasterAdmin_BusinessError_Returns400(t *testing.T) {
	stub := &stubUserService{Err: errors.New("初始管理员不允许删除")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/users/admin-id-000001/delete", "")

	assertFailCode(t, w, http.StatusBadRequest, "不允许删除")
}

func TestDeleteMasterAdmin_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/users/u2/delete", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u2", stub.GotUserID)
}

func TestUpdateMasterAdminStatus_Success_ForwardsStatus(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/u1/status", `{"status":0}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
	assert.Equal(t, 0, stub.GotStatusN)
}

func TestUpdateMasterAdminStatus_InvalidStatus_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/u1/status", `{"status":5}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateMasterAdminStatus_NotSystemAdmin_Returns404(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUserNotSystemAdmin}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/u1/status", `{"status":1}`)

	assertFailCode(t, w, http.StatusNotFound, "系统管理员不存在")
}

// ============ 系统管理员回收站 ============

func TestListDeletedMasterAdmins_Success(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("m-d")}, Total: 1}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/users/deleted", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "m-d", stub.List[0].ID)
}

func TestRestoreMasterAdmin_RecordNotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/u-gone/restore", "")

	assertFailCode(t, w, http.StatusNotFound, "系统管理员不存在或未删除")
}

func TestRestoreMasterAdmin_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/u1/restore", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestPermanentDeleteMasterAdmin_RecordNotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/users/u-gone/permanent", "")

	assertFailCode(t, w, http.StatusNotFound, "系统管理员不存在")
}

func TestPermanentDeleteMasterAdmin_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/users/u1/permanent", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// ============ 批量操作 ============

func TestBatchUpdateMasterAdminStatus_Success(t *testing.T) {
	stub := &stubUserService{Affected: 2}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/batch/status", `{"ids":["u1","u2"],"status":0}`)

	var out map[string]any
	requireOK(t, w, &out)
	data, _ := out["data"].(map[string]any)
	assert.Equal(t, float64(2), data["updated"])
	assert.Equal(t, []string{"u1", "u2"}, stub.GotIDs)
	assert.Equal(t, 0, stub.GotStatusN)
}

func TestBatchUpdateMasterAdminStatus_ServiceError_Returns400(t *testing.T) {
	stub := &stubUserService{Err: errors.New("ids empty")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/users/batch/status", `{"ids":["u1"],"status":1}`)

	assertFailCode(t, w, http.StatusBadRequest, "ids empty")
}

func TestBatchDeleteMasterAdmins_Validation_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/users/batch/delete", `{"ids":[]}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBatchDeleteMasterAdmins_Success(t *testing.T) {
	stub := &stubUserService{Affected: 3}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/users/batch/delete", `{"ids":["u1","u2","u3"]}`)

	var out map[string]any
	requireOK(t, w, &out)
	data, _ := out["data"].(map[string]any)
	assert.Equal(t, float64(3), data["deleted"])
}

// ============ 跨租户用户管理 ============

func TestAdminCreateTenantUser_Validation_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/admin/tenant-users", `{"tenant_id":"","username":"bobby"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotAdminCreateTenant)
}

func TestAdminCreateTenantUser_UsernameExists_Returns409(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUsernameExists}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/admin/tenant-users", `{"tenant_id":"t1","username":"bobby","password":"123456","nickname":"bob","role_ids":["r1"]}`)

	assertFailCode(t, w, http.StatusConflict, "用户名已被使用")
}

func TestAdminCreateTenantUser_Success(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("u-x")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/admin/tenant-users", `{"tenant_id":"t1","username":"bobby","password":"123456","nickname":"bob","role_ids":["r1"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotAdminCreateTenant)
	assert.Equal(t, "t1", stub.GotAdminCreateTenant.TenantID)
}

func TestAdminListTenantUsers_MissingTenantID_Returns400(t *testing.T) {
	stub := &stubUserService{}
	h := rawHandler(stub)

	w := doRaw(h.AdminListTenantUsers)

	assertFailCode(t, w, http.StatusBadRequest, "租户ID不能为空")
}

func TestAdminListTenantUsers_Success(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("u1")}, Total: 1}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/tenant-users/t1/list?page=1&page_size=20&keyword=k", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t1", stub.GotTenantID)
	assert.Equal(t, 20, stub.GotPageSize)
}

func TestAdminListAllTenantUsers_Success(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("u1")}, Total: 1}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/tenant-users/all", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminUpdateTenantUser_NotInTenant_Returns400(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUserNotInTenant}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u9/update", `{"nickname":"x"}`)

	assertFailCode(t, w, http.StatusBadRequest, "用户不属于指定租户")
}

func TestAdminUpdateTenantUser_Success(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("u1")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u1/update", `{"nickname":"x"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t1", stub.GotTenantID)
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestAdminDeleteTenantUser_NotInTenant_Returns400(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUserNotInTenant}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/tenant-users/t1/u9/delete", "")

	assertFailCode(t, w, http.StatusBadRequest, "用户不属于指定租户")
}

func TestAdminDeleteTenantUser_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/tenant-users/t1/u1/delete", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminListDeletedTenantUsers_MissingTenant_Returns400(t *testing.T) {
	stub := &stubUserService{}
	h := rawHandler(stub)

	w := doRaw(h.AdminListDeletedTenantUsers)

	assertFailCode(t, w, http.StatusBadRequest, "租户ID不能为空")
}

func TestAdminListDeletedTenantUsers_Success(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("u-d")}, Total: 1}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/tenant-users/t1/deleted", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminListAllDeletedTenantUsers_Success(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("u-d")}, Total: 1}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/tenant-users/deleted/all", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminRestoreTenantUser_RecordNotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u-gone/restore", "")

	assertFailCode(t, w, http.StatusNotFound, "用户不存在或未删除")
}

func TestAdminRestoreTenantUser_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u1/restore", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestAdminPermanentDeleteTenantUser_RecordNotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/tenant-users/t1/u-gone/permanent", "")

	assertFailCode(t, w, http.StatusNotFound, "用户不存在或未删除")
}

func TestAdminPermanentDeleteTenantUser_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/tenant-users/t1/u1/permanent", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestAdminUpdateTenantUserStatus_NotInTenant_Returns400(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUserNotInTenant}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u9/status", `{"status":0}`)

	assertFailCode(t, w, http.StatusBadRequest, "用户不属于指定租户")
}

func TestAdminUpdateTenantUserStatus_RecordNotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u-gone/status", `{"status":0}`)

	assertFailCode(t, w, http.StatusNotFound, "用户不存在")
}

func TestAdminUpdateTenantUserStatus_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u1/status", `{"status":0}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 0, stub.GotStatusN)
}

func TestAdminBatchUpdateTenantUserStatus_MissingTenant_Returns400(t *testing.T) {
	stub := &stubUserService{}
	h := rawHandler(stub)

	w := doRaw(h.AdminBatchUpdateTenantUserStatus)

	assertFailCode(t, w, http.StatusBadRequest, "租户ID不能为空")
}

func TestAdminBatchUpdateTenantUserStatus_Success(t *testing.T) {
	stub := &stubUserService{Affected: 2}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/batch/status", `{"ids":["u1","u2"],"status":1}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t1", stub.GotTenantID)
	assert.Equal(t, []string{"u1", "u2"}, stub.GotIDs)
}

func TestAdminBatchUpdateTenantUserStatus_ServiceError_Returns400(t *testing.T) {
	stub := &stubUserService{Err: errors.New("not found")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/batch/status", `{"ids":["u9"],"status":1}`)

	assertFailCode(t, w, http.StatusBadRequest, "not found")
}

func TestAdminBatchDeleteTenantUsers_Success(t *testing.T) {
	stub := &stubUserService{Affected: 2}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/admin/tenant-users/t1/batch/delete", `{"ids":["u1","u2"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t1", stub.GotTenantID)
}

func TestAdminResetTenantUserPassword_ConfirmMismatch_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u1/reset-password", `{"new_password":"abcdef","confirm_password":"zzzzzz"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotNewPass)
}

func TestAdminResetTenantUserPassword_RecordNotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u-gone/reset-password", `{"new_password":"abcdef","confirm_password":"abcdef"}`)

	assertFailCode(t, w, http.StatusNotFound, "用户不存在")
}

func TestAdminResetTenantUserPassword_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/admin/tenant-users/t1/u1/reset-password", `{"new_password":"abcdef","confirm_password":"abcdef"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t1", stub.GotTenantID)
	assert.Equal(t, "u1", stub.GotUserID)
	assert.Equal(t, "abcdef", stub.GotNewPass)
}
