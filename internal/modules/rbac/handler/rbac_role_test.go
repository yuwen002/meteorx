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

func sampleRole(id, name, code string) *model.Role {
	return &model.Role{ID: id, Name: name, Code: code, Scope: "tenant", Status: 1}
}

func TestCreateRole_Success_ForwardsReq(t *testing.T) {
	stub := &stubRBACService{Role: sampleRole("r1", "租户管理员", "tenant_admin")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/roles",
		`{"name":"租户管理员","code":"tenant_admin","description":"d","scope":"tenant","status":1}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotCreateR)
	assert.Equal(t, "tenant_admin", stub.GotCreateR.Code)
	assert.Contains(t, w.Body.String(), `"r1"`)
}

func TestCreateRole_InvalidBody_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	// name/code 必填
	w := doRBAC(t, router, http.MethodPost, "/roles", `{"description":"d"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotCreateR, "非法参数不应触达服务层")
}

func TestCreateRole_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("编码已存在")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/roles",
		`{"name":"管理员","code":"admin"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "编码已存在")
}

func TestGetRole_MissingID_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	h := handler.NewRBACHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.GetRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "角色ID不能为空")
}

func TestGetRole_NotFound_Returns404(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("not found")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/r-x/detail", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "角色不存在")
}

func TestGetRole_Success_ForwardsID(t *testing.T) {
	stub := &stubRBACService{Role: sampleRole("r1", "管理员", "admin")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/r1/detail", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotID)
	assert.Contains(t, w.Body.String(), `"管理员"`)
}

func TestListRoles_Success_ForwardsQuery(t *testing.T) {
	stub := &stubRBACService{Roles: []*model.Role{sampleRole("r1", "管理员", "admin")}, Affected: 1}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles?page=1&page_size=10&keyword=管理&tenant_id=t-9", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t-9", stub.GotTenantID, "tenant_id 应透传给服务层")
	assert.Equal(t, "管理", stub.GotKeyword)
	assert.Equal(t, 10, stub.GotPageSize)
	assert.Contains(t, w.Body.String(), `"r1"`)
}

func TestListRoles_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("scan failed")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取角色列表失败")
}

func TestListRolesForSelect_DefaultScopeSystem(t *testing.T) {
	stub := &stubRBACService{Roles: []*model.Role{sampleRole("r1", "系统管理员", "sys_admin")}}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/select", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "system", stub.GotScope, "缺省作用域应为 system")
}

func TestListRolesForSelect_ForwardsScope(t *testing.T) {
	stub := &stubRBACService{Roles: []*model.Role{sampleRole("r2", "租户角色", "t_role")}}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/select?scope=tenant", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant", stub.GotScope)
	assert.Contains(t, w.Body.String(), `"r2"`)
}

func TestListRolesForSelect_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/select", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListSystemAdminRoles_Success(t *testing.T) {
	stub := &stubRBACService{Roles: []*model.Role{sampleRole("r1", "平台管理员", "platform_admin")}}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/system-admin", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"platform_admin"`)
}

func TestUpdateRole_Success_ForwardsIDAndReq(t *testing.T) {
	stub := &stubRBACService{Role: sampleRole("r1", "改名角色", "admin")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/r1/update", `{"name":"改名角色"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotID)
	require.NotNil(t, stub.GotUpdateR)
	assert.Equal(t, "改名角色", stub.GotUpdateR.Name)
}

func TestUpdateRole_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("更新失败")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/r1/update", `{"name":"x"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "更新失败")
}

func TestDeleteRole_MissingID_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	h := handler.NewRBACHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	h.DeleteRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteRole_Success_ForwardsID(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/r1/delete", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotID)
}

func TestDeleteRole_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("内置角色不可删除")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/r1/delete", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "内置角色不可删除")
}

func TestPermanentDeleteRole_Success(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/r1/permanent", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotID)
}

func TestRestoreRole_Success_ForwardsID(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPost, "/roles/r1/restore", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotID)
}

func TestUpdateRoleStatus_Success_ForwardsStatus(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/r1/status", `{"status":0}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "r1", stub.GotID)
	assert.Equal(t, 0, stub.GotStatus)
}

func TestUpdateRoleStatus_InvalidStatus_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/r1/status", `{"status":9}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotStatus, "非法状态不应触达服务层")
}

func TestBatchUpdateRoleStatus_Success_ForwardsIDsAndStatus(t *testing.T) {
	stub := &stubRBACService{Affected: 2}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodPut, "/roles/batch/status", `{"ids":["r1","r2"],"status":1}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []string{"r1", "r2"}, stub.GotIDs)
	assert.Equal(t, 1, stub.GotStatus)
	assert.Contains(t, w.Body.String(), `"updated":2`)
}

func TestBatchUpdateRoleStatus_MissingIDs_Returns400(t *testing.T) {
	stub := &stubRBACService{}
	router := newRBACRouter(stub)

	// 缺 ids（nil）触发 required 校验
	w := doRBAC(t, router, http.MethodPut, "/roles/batch/status", `{"status":1}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotIDs)
}

func TestBatchPermanentDeleteRoles_Success(t *testing.T) {
	stub := &stubRBACService{Affected: 2}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/batch/permanent", `{"ids":["r1","r2"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []string{"r1", "r2"}, stub.GotIDs)
	assert.Contains(t, w.Body.String(), `"deleted":2`)
}

func TestBatchDeleteRoles_ServiceError_Returns400(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("含内置角色")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodDelete, "/roles/batch/delete", `{"ids":["r1"]}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "含内置角色")
}

func TestListDeletedRoles_Success_ForwardsPage(t *testing.T) {
	stub := &stubRBACService{Roles: []*model.Role{sampleRole("r9", "已删角色", "gone")}, Affected: 1}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/deleted?page=1&page_size=10&keyword=已删", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, stub.GotPage)
	assert.Equal(t, "已删", stub.GotKeyword)
	assert.Contains(t, w.Body.String(), `"r9"`)
}

func TestListDeletedRoles_ServiceError_Returns500(t *testing.T) {
	stub := &stubRBACService{Err: errors.New("boom")}
	router := newRBACRouter(stub)

	w := doRBAC(t, router, http.MethodGet, "/roles/deleted", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
