package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ============ 租户用户列表 ============

func TestListUsers_NoTenantCtx_Returns401(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := doAnon(t, router, http.MethodGet, "/users", "")

	assertFailCode(t, w, http.StatusUnauthorized, "未获取到租户信息")
}

func TestListUsers_Success_ForwardsQueryAndPagination(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("u1"), sampleUserResp("u2")}, Total: 2}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/users?page=2&page_size=50&keyword=alice&status=1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, 2, stub.GotPage)
	assert.Equal(t, 50, stub.GotPageSize)
	assert.Equal(t, "alice", stub.GotKeyword)
	require.NotNil(t, stub.GotStatus)
	assert.Equal(t, 1, *stub.GotStatus)

	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	data, _ := out["data"].(map[string]any)
	pagination, _ := data["pagination"].(map[string]any)
	assert.Equal(t, float64(2), pagination["total"])
}

func TestListUsers_InvalidStatusFilter_DefaultsToDisabled(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	// handler 对非法 status 保守降级为 status=0（禁用）过滤，避免越权看到启用账户
	w := do(t, router, http.MethodGet, "/users?status=abc", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotStatus)
	assert.Equal(t, 0, *stub.GotStatus)
}

func TestListUsers_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("db down")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/users", "")

	assertFailCode(t, w, http.StatusInternalServerError, "获取用户列表失败")
}

// ============ 用户详情 ============

func TestGetUser_MissingID_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	// 不经过 {id} 路由：chi 上下文无 id
	w := do(t, router, http.MethodGet, "/users//detail", "")

	assertFailCode(t, w, http.StatusBadRequest, "用户ID不能为空")
}

func TestGetUser_NotBelongToTenant_Returns403(t *testing.T) {
	stub := &stubUserService{Belongs: false}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/users/u-other/detail", "")

	assertFailCode(t, w, http.StatusForbidden, "无权查看该用户")
	assert.Equal(t, "u-other", stub.GotUserID)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
}

func TestGetUser_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Belongs: true, Err: errors.New("load failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/users/u1/detail", "")

	assertFailCode(t, w, http.StatusInternalServerError, "获取用户详情失败")
}

func TestGetUser_Success(t *testing.T) {
	stub := &stubUserService{Belongs: true, Resp: sampleUserResp("u1")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/users/u1/detail", "")

	var out struct {
		Data userRespBody `json:"data"`
	}
	requireOK(t, w, &out)
	assert.Equal(t, "u1", out.Data.ID)
	assert.Equal(t, "alice", out.Data.Username)
}

// ============ 创建用户 ============

func TestCreateUser_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	// 缺 role_ids（必填）
	w := do(t, router, http.MethodPost, "/users", `{"username":"bobby","password":"123456","nickname":"bob"}`)

	assertFailCode(t, w, http.StatusBadRequest, "role_ids")
	assert.Empty(t, stub.GotCreateReq, "参数非法不应触达服务层")
}

func TestCreateUser_InvalidEmail_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/users", `{"username":"bobby","password":"123456","nickname":"bob","role_ids":["r1"],"email":"bad"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotCreateReq)
}

func TestCreateUser_UsernameExists_Returns409(t *testing.T) {
	stub := &stubUserService{Err: service.ErrUsernameExists}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/users", `{"username":"bobby","password":"123456","nickname":"bob","role_ids":["r1"]}`)

	assertFailCode(t, w, http.StatusConflict, "用户名已被使用")
}

func TestCreateUser_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("write failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/users", `{"username":"bobby","password":"123456","nickname":"bob","role_ids":["r1"]}`)

	assertFailCode(t, w, http.StatusInternalServerError, "创建用户失败")
}

func TestCreateUser_Success_ForwardsTenant(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("u-new")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPost, "/users", `{"username":"bobby","password":"123456","nickname":"bob","email":"b@x.com","role_ids":["r1","r2"]}`)

	var out struct {
		Data userRespBody `json:"data"`
	}
	requireOK(t, w, &out)
	assert.Equal(t, "u-new", out.Data.ID)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	require.NotNil(t, stub.GotCreateReq)
	assert.Equal(t, []string{"r1", "r2"}, stub.GotCreateReq.RoleIDs)
}

// ============ 更新 / 删除用户 ============

func TestUpdateUser_MissingID_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users//update", `{"nickname":"x"}`)

	assertFailCode(t, w, http.StatusBadRequest, "用户ID不能为空")
}

func TestUpdateUser_Forbidden_Returns403(t *testing.T) {
	stub := &stubUserService{Belongs: false}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u-other/update", `{"nickname":"x"}`)

	assertFailCode(t, w, http.StatusForbidden, "无权操作该用户")
	assert.False(t, stub.GotUpdateReq != nil, "跨租户用户不应触达更新")
}

func TestUpdateUser_InvalidStatus_Returns400(t *testing.T) {
	stub := &stubUserService{Belongs: true}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/update", `{"status":9}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotUpdateReq)
}

func TestUpdateUser_Success_ForwardsFields(t *testing.T) {
	stub := &stubUserService{Belongs: true, Resp: sampleUserResp("u1")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/update", `{"nickname":"new-name","email":"a@x.com","role_ids":["r2"],"status":0}`)

	var out struct {
		Data userRespBody `json:"data"`
	}
	requireOK(t, w, &out)
	assert.Equal(t, "u1", stub.GotUserID)
	require.NotNil(t, stub.GotUpdateReq)
	assert.Equal(t, "new-name", stub.GotUpdateReq.Nickname)
	stat := 0
	assert.Equal(t, &stat, stub.GotUpdateReq.Status)
}

func TestUpdateUser_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Belongs: true, Err: errors.New("update failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/update", `{"nickname":"x"}`)

	assertFailCode(t, w, http.StatusInternalServerError, "更新用户失败")
}

func TestDeleteUser_Success(t *testing.T) {
	stub := &stubUserService{Belongs: true}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/users/u1/delete", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestDeleteUser_Forbidden_Returns403(t *testing.T) {
	stub := &stubUserService{Belongs: false}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/users/u-other/delete", "")

	assertFailCode(t, w, http.StatusForbidden, "无权操作该用户")
}

func TestDeleteUser_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Belongs: true, Err: errors.New("del failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/users/u1/delete", "")

	assertFailCode(t, w, http.StatusInternalServerError, "删除用户失败")
}

// ============ 租户管理员重置密码 ============

func TestResetPassword_NoTenantCtx_Returns401(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := doAnon(t, router, http.MethodPut, "/users/u1/reset-password", `{"new_password":"abcdef","confirm_password":"abcdef"}`)

	assertFailCode(t, w, http.StatusUnauthorized, "未获取到租户信息")
}

func TestResetPassword_NotInTenant_Returns403(t *testing.T) {
	stub := &stubUserService{Belongs: false}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u-other/reset-password", `{"new_password":"abcdef","confirm_password":"abcdef"}`)

	assertFailCode(t, w, http.StatusForbidden, "无权操作该用户")
}

func TestResetPassword_ConfirmMismatch_Returns400(t *testing.T) {
	stub := &stubUserService{Belongs: true}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/reset-password", `{"new_password":"abcdef","confirm_password":"different"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotNewPass, "校验失败不应触达服务层")
}

func TestResetPassword_Success(t *testing.T) {
	stub := &stubUserService{Belongs: true}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/reset-password", `{"new_password":"abcdef","confirm_password":"abcdef"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
	assert.Equal(t, "abcdef", stub.GotNewPass)
}

func TestResetPassword_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Belongs: true, Err: errors.New("update failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/reset-password", `{"new_password":"abcdef","confirm_password":"abcdef"}`)

	assertFailCode(t, w, http.StatusInternalServerError, "重置密码失败")
}

// ============ 回收站 ============

func TestListDeletedUsers_NoTenant_Returns401(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := doAnon(t, router, http.MethodGet, "/users/deleted", "")

	assertFailCode(t, w, http.StatusUnauthorized, "tenant not found")
}

func TestListDeletedUsers_Success_ForwardsQuery(t *testing.T) {
	stub := &stubUserService{List: []*dto.UserResp{sampleUserResp("u-d1")}, Total: 1}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/users/deleted?page=1&page_size=20&keyword=old", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, 1, stub.GotPage)
	assert.Equal(t, 20, stub.GotPageSize)
	assert.Equal(t, "old", stub.GotKeyword)
}

func TestListDeletedUsers_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("scan failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/users/deleted", "")

	assertFailCode(t, w, http.StatusInternalServerError, "get deleted users failed")
}

func TestRestoreUser_RecordNotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u-gone/restore", "")

	assertFailCode(t, w, http.StatusNotFound, "user not found")
}

func TestRestoreUser_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/restore", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestRestoreUser_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("restore failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/users/u1/restore", "")

	assertFailCode(t, w, http.StatusInternalServerError, "restore user failed")
}

func TestPermanentDeleteUser_NotFound_Returns404(t *testing.T) {
	stub := &stubUserService{Err: gorm.ErrRecordNotFound}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/users/u-gone/permanent", "")

	assertFailCode(t, w, http.StatusNotFound, "user not found")
}

func TestPermanentDeleteUser_Success(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/users/u1/permanent", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestPermanentDeleteUser_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("permanent failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodDelete, "/users/u1/permanent", "")

	assertFailCode(t, w, http.StatusInternalServerError, "permanent delete user failed")
}

// ============ Profile / 统计 ============

func TestGetProfile_NoUserCtx_Returns401(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := doAnon(t, router, http.MethodGet, "/profile/", "")

	assertFailCode(t, w, http.StatusUnauthorized, "未获取到用户信息")
}

func TestGetProfile_Success_ReadsUserFromCtx(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("user-1")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/profile/", "")

	var out struct {
		Data userRespBody `json:"data"`
	}
	requireOK(t, w, &out)
	assert.Equal(t, "user-1", stub.GotUserID)
}

func TestUpdateProfile_Success_SanitizesSensitiveFields(t *testing.T) {
	stub := &stubUserService{Resp: sampleUserResp("user-1")}
	router := newUserRouter(stub)

	// 用户自改不能携带角色/状态字段
	w := do(t, router, http.MethodPut, "/profile/", `{"nickname":"new","email":"p@x.com","role_ids":["r1"],"status":0}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotUpdateReq)
	assert.Equal(t, "user-1", stub.GotUserID)
	assert.Equal(t, "new", stub.GotUpdateReq.Nickname)
	assert.Nil(t, stub.GotUpdateReq.RoleIDs, "自改 profile 应剥离 role_ids")
	assert.Nil(t, stub.GotUpdateReq.Status, "自改 profile 应剥离 status")
}

func TestChangePassword_NoUserCtx_Returns401(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := doAnon(t, router, http.MethodPut, "/profile/password", `{"old_password":"111111","new_password":"222222","confirm_password":"222222"}`)

	assertFailCode(t, w, http.StatusUnauthorized, "未获取到用户信息")
}

func TestChangePassword_ConfirmMismatch_Returns400(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/profile/password", `{"old_password":"111111","new_password":"222222","confirm_password":"333333"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotNewPass)
}

func TestChangePassword_WrongOldPassword_Returns400(t *testing.T) {
	stub := &stubUserService{Err: service.ErrWrongOldPassword}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/profile/password", `{"old_password":"wrong1","new_password":"222222","confirm_password":"222222"}`)

	assertFailCode(t, w, http.StatusBadRequest, "原密码错误")
}

func TestChangePassword_Success_ForwardsPasswords(t *testing.T) {
	stub := &stubUserService{}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodPut, "/profile/password", `{"old_password":"111111","new_password":"222222","confirm_password":"222222"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "user-1", stub.GotUserID)
	assert.Equal(t, "111111", stub.GotOldPass)
	assert.Equal(t, "222222", stub.GotNewPass)
}

func TestGetStats_Success(t *testing.T) {
	stub := &stubUserService{Total: 12}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/profile/stats", "")

	var out map[string]any
	requireOK(t, w, &out)
	data, _ := out["data"].(map[string]any)
	assert.Equal(t, float64(12), data["user_count"])
	assert.Equal(t, "tenant-1", stub.GotTenantID)
}

func TestGetStats_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("count failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/profile/stats", "")

	assertFailCode(t, w, http.StatusInternalServerError, "获取用户统计失败")
}

func TestAdminStats_Success(t *testing.T) {
	stub := &stubUserService{Total: 120}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/stats", "")

	var out map[string]any
	requireOK(t, w, &out)
	data, _ := out["data"].(map[string]any)
	assert.Equal(t, float64(120), data["user_count"])
}

func TestAdminStats_ServiceError_Returns500(t *testing.T) {
	stub := &stubUserService{Err: errors.New("count failed")}
	router := newUserRouter(stub)

	w := do(t, router, http.MethodGet, "/admin/stats", "")

	assertFailCode(t, w, http.StatusInternalServerError, "获取用户统计失败")
}

// userRespBody 仅解码 data 中关心的字段
type userRespBody struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}
