package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/modules/rbac/handler"
	"meteorx/internal/modules/rbac/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignUserRoles_Success_ForwardsReq(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/user-roles/u1/roles", `{"role_ids":["r1","r2"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
	require.NotNil(t, stub.GotAssign)
	assert.Equal(t, []string{"r1", "r2"}, stub.GotAssign.RoleIDs)
}

func TestAssignUserRoles_MissingUserID_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	h := handler.NewRBACHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h.AssignUserRoles(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "用户ID不能为空")
}

func TestAssignUserRoles_MissingRoleIDs_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	// 缺 role_ids（nil）触发 required 校验
	w := doRBAC(t, router, http.MethodPost, "/user-roles/u1/roles", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotAssign)
}

func TestAssignUserRoles_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("角色已停用")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/user-roles/u1/roles", `{"role_ids":["r1"]}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "角色已停用")
}

func TestGetUserRoles_Success_ForwardsUserID(t *testing.T) {
	stub := &stubRBACService{Roles: []*model.Role{sampleRole("r1", "管理员", "admin")}}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/user-roles/u1/roles", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
	assert.Contains(t, w.Body.String(), `"管理员"`)
}

func TestGetUserRoles_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/user-roles/u1/roles", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取用户角色失败")
}

func TestRemoveUserRole_Success_ForwardsBoth(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/user-roles/u1/roles/r1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
	assert.Equal(t, "r1", stub.GotRoleID)
}

func TestRemoveUserRole_MissingRoleID_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	h := handler.NewRBACHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	h.RemoveUserRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "用户ID不能为空")
}

func TestRemoveAllUserRoles_Success_ForwardsUserID(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/user-roles/u1/roles", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestGetRoleUsers_Success_ReturnsUserIDs(t *testing.T) {
	stub := &stubRBACService{UserIDList: []string{"u1", "u2"}}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/user-roles/roles/r1/users", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotRoleID)
	assert.Contains(t, w.Body.String(), `"u1"`)
	assert.Contains(t, w.Body.String(), `"u2"`)
}

func TestGetRoleUsers_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/user-roles/roles/r1/users", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取角色用户列表失败")
}

func TestListUserRoles_Success_ForwardsFilters(t *testing.T) {
	stub := &stubRBACService{UserRoles: []*model.UserRole{{
		UserID: "u1", RoleID: "r1",
		User: &model.RoleUserInfo{ID: "u1", Username: "bobby", Nickname: "波比"},
		Role: &model.Role{ID: "r1", Name: "管理员", Code: "admin"},
	}}, Affected: 1}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/user-roles?page=1&page_size=10&user_id=u1&role_id=r1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "u1", stub.GotUserID)
	assert.Equal(t, "r1", stub.GotRoleID)
	assert.Contains(t, w.Body.String(), `"bobby"`)
	assert.Contains(t, w.Body.String(), `"管理员"`)
}

func TestListUserRoles_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/user-roles", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取用户角色关系列表失败")
}

func TestBatchAssignUserRoles_Success_ForwardsReq(t *testing.T) {
	stub := &stubRBACService{Affected: 1}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/user-roles/batch/assign",
		`{"assignments":[{"user_id":"u1","role_ids":["r1"]},{"user_id":"u2","role_ids":["r2"]}]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotBatchAsg)
	assert.Len(t, stub.GotBatchAsg.UserRoleAssignments, 2)
	assert.Contains(t, w.Body.String(), `"assigned":1`)
}

func TestBatchAssignUserRoles_InvalidBody_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	// 缺 assignments（nil）触发 required 校验
	w := doRBAC(t, router, http.MethodPost, "/user-roles/batch/assign", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotBatchAsg)
}

func TestGetStats_Success_ReturnsThreeCounts(t *testing.T) {
	stub := &stubRBACService{RoleCount: 5, PermCount: 20, MyPerms: 3}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/stats", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, "user-1", stub.GotUserID)
	assert.Contains(t, w.Body.String(), `"role_count":5`)
	assert.Contains(t, w.Body.String(), `"permission_count":20`)
	assert.Contains(t, w.Body.String(), `"my_permission":3`)
}

func TestGetStats_CountRolesError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("count failed")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/stats", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取角色统计失败")
}
