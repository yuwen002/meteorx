package handler_test

import (
	"errors"
	"net/http"
	"testing"

	"meteorx/internal/modules/rbac/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func samplePerm(id, code string) *model.Permission {
	return &model.Permission{ID: id, Name: "查看用户", Code: code, Resource: "user", Action: "list"}
}

// ============ 权限 ============

func TestCreatePermission_Success_ForwardsReq(t *testing.T) {
	stub := &stubRBACService{Perm: samplePerm("p1", "user:list")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/permissions",
		`{"name":"查看用户","code":"user:list","resource":"user","action":"list","description":"查看用户列表"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotCreateP)
	assert.Equal(t, "user:list", stub.GotCreateP.Code)
	assert.Equal(t, "user", stub.GotCreateP.Resource)
	assert.Contains(t, w.Body.String(), `"p1"`)
}

func TestCreatePermission_InvalidBody_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	// resource/action/code 必填
	w := doRBAC(t, router, http.MethodPost, "/permissions", `{"name":"x"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotCreateP)
}

func TestCreatePermission_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("权限码已存在")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/permissions",
		`{"name":"查看用户","code":"user:list","resource":"user","action":"list"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "权限码已存在")
}

func TestGetPermission_NotFound_Returns404(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("not found")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/permissions/p-x/detail", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "权限不存在")
}

func TestGetPermission_Success_ForwardsID(t *testing.T) {
	stub := &stubRBACService{Perm: samplePerm("p1", "user:list")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/permissions/p1/detail", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", stub.GotID)
	assert.Contains(t, w.Body.String(), `"user:list"`)
}

func TestListPermissions_Success_ForwardsFilters(t *testing.T) {
	stub := &stubRBACService{Perms: []*model.Permission{samplePerm("p1", "user:list")}, Affected: 1}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/permissions?page=1&page_size=20&resource=user&keyword=list", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "user", stub.GotResource)
	assert.Equal(t, "list", stub.GotKeyword)
	assert.Equal(t, 20, stub.GotPageSize)
	assert.Contains(t, w.Body.String(), `"p1"`)
}

func TestListPermissions_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/permissions", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取权限列表失败")
}

func TestUpdatePermission_Success_ForwardsIDAndReq(t *testing.T) {
	stub := &stubRBACService{Perm: samplePerm("p1", "user:list")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/permissions/p1/update", `{"name":"改后名称"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", stub.GotID)
	require.NotNil(t, stub.GotUpdateP)
	assert.Equal(t, "改后名称", stub.GotUpdateP.Name)
}

func TestUpdatePermission_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("更新失败")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/permissions/p1/update", `{"name":"x"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdatePermissionStatus_Success_ForwardsStatus(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/permissions/p1/status", `{"status":0}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", stub.GotID)
	assert.Equal(t, 0, stub.GotStatus)
}

func TestUpdatePermissionStatus_InvalidStatus_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/permissions/p1/status", `{"status":5}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBatchUpdatePermissionStatus_Success(t *testing.T) {
	stub := &stubRBACService{Affected: 2}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/permissions/batch/status", `{"ids":["p1","p2"],"status":0}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []string{"p1", "p2"}, stub.GotIDs)
	assert.Equal(t, 0, stub.GotStatus)
	assert.Contains(t, w.Body.String(), `"updated":2`)
}

func TestDeletePermission_Success(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/permissions/p1/delete", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", stub.GotID)
}

func TestBatchDeletePermissions_Success(t *testing.T) {
	stub := &stubRBACService{Affected: 3}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/permissions/batch/delete", `{"ids":["p1","p2","p3"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"deleted":3`)
}

func TestBatchDeletePermissions_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("含系统权限")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/permissions/batch/delete", `{"ids":["p1"]}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "含系统权限")
}

// ============ 角色-权限绑定 ============

func TestBindRolePermissions_Success_ForwardsReq(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/r1/permissions", `{"permission_ids":["p1","p2"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotRoleID)
	require.NotNil(t, stub.GotBind)
	assert.Equal(t, []string{"p1", "p2"}, stub.GotBind.PermissionIDs)
}

func TestBindRolePermissions_MissingPermissionIDs_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	// 缺 permission_ids（nil）触发 required 校验
	w := doRBAC(t, router, http.MethodPut, "/roles/r1/permissions", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotBind)
}

func TestBindRolePermissions_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("绑定失败")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/r1/permissions", `{"permission_ids":["p1"]}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "绑定失败")
}

func TestGetRolePermissions_Success_WithoutResource(t *testing.T) {
	stub := &stubRBACService{Perms: []*model.Permission{samplePerm("p1", "user:list")}}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/r1/permissions", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotRoleID)
	assert.Empty(t, stub.GotResource)
	assert.Contains(t, w.Body.String(), `"p1"`)
}

func TestGetRolePermissions_WithResource_ForwardsResource(t *testing.T) {
	stub := &stubRBACService{Perms: []*model.Permission{samplePerm("p2", "file:download")}}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/r1/permissions?resource=file", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "file", stub.GotResource)
}

func TestGetRolePermissions_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/r1/permissions", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取角色权限失败")
}

func TestUnbindRolePermission_Success_ForwardsBothIDs(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/r1/permissions", `{"permission_id":"p1"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotRoleID)
	assert.Equal(t, "p1", stub.GotPermID)
}

func TestUnbindRolePermissions_Success(t *testing.T) {
	stub := &stubRBACService{Affected: 2}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/r1/permissions/batch", `{"permission_ids":["p1","p2"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotRoleID)
	assert.Equal(t, []string{"p1", "p2"}, stub.GotIDs)
	assert.Contains(t, w.Body.String(), `"unbound":2`)
}

func TestBatchBindRolesPermissions_Success_ForwardsReq(t *testing.T) {
	stub := &stubRBACService{Affected: 4}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/batch/permissions",
		`{"role_ids":["r1","r2"],"permission_ids":["p1","p2"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotBatchBind)
	assert.Equal(t, []string{"r1", "r2"}, stub.GotBatchBind.RoleIDs)
	assert.Contains(t, w.Body.String(), `"bound":4`)
}

func TestBatchUnbindRolesPermissions_Success(t *testing.T) {
	stub := &stubRBACService{Affected: 2}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/batch/permissions",
		`{"role_ids":["r1"],"permission_ids":["p1","p2"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotBatchUnb)
	assert.Equal(t, []string{"p1", "p2"}, stub.GotBatchUnb.PermissionIDs)
	assert.Contains(t, w.Body.String(), `"unbound":2`)
}

func TestListRolePermissions_Success_ForwardsFilters(t *testing.T) {
	stub := &stubRBACService{RolePerms: []*model.RolePermission{{
		RoleID: "r1", PermissionID: "p1",
		Role:       &model.Role{ID: "r1", Name: "管理员", Code: "admin"},
		Permission: samplePerm("p1", "user:list"),
	}}, Affected: 1}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/role-permissions?role_id=r1&permission_id=p1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotRoleID)
	assert.Equal(t, "p1", stub.GotPermID)
	assert.Contains(t, w.Body.String(), `"管理员"`)
	assert.Contains(t, w.Body.String(), `"user:list"`)
}

func TestListRolePermissions_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/role-permissions", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取角色权限关系列表失败")
}
