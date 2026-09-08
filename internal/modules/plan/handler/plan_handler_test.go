package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/plan/dto"
	"meteorx/internal/modules/plan/handler"
	"meteorx/internal/modules/plan/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPlanRouter(stub *stubPlanService) http.Handler {
	h := handler.NewPlanHandler(stub)
	r := chi.NewRouter()
	r.Get("/admin/plans", h.ListPlans)
	r.Post("/admin/plans", h.CreatePlan)
	r.Get("/admin/plans/select", h.ListAllEnabledPlans)
	r.Put("/admin/plans/{id}/update", h.UpdatePlan)
	r.Delete("/admin/plans/{id}/delete", h.DeletePlan)
	r.Put("/admin/tenants/{id}/plan", h.AssignPlan)
	r.Get("/admin/tenants/{id}/plan", h.CheckTenantPlan)
	r.Get("/tenant/current/plan", h.GetCurrentPlan)
	return r
}

func doPlan(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(contextx.SetVars(req.Context(), "tenant-1", "user-1", []string{"admin"}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func samplePlanResp() *dto.PlanResp {
	return &dto.PlanResp{ID: "p1", Name: "基础版", Code: "basic", UserLimit: 50, Price: 99, Status: 1}
}

func TestCreatePlan_Success(t *testing.T) {
	stub := &stubPlanService{Resp: samplePlanResp()}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPost, "/admin/plans",
		`{"name":"基础版","code":"basic","description":"d","user_limit":50,"price":99,"status":1}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotCreate)
	assert.Equal(t, "basic", stub.GotCreate.Code)
	assert.Equal(t, 50, stub.GotCreate.UserLimit)
}

func TestCreatePlan_CodeConflict_Returns409(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrPlanCodeConflict}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPost, "/admin/plans",
		`{"name":"基础版","code":"basic","user_limit":50,"price":99,"status":1}`)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "套餐编码已存在")
}

func TestCreatePlan_InvalidBody_Returns400(t *testing.T) {
	stub := &stubPlanService{}
	router := newPlanRouter(stub)

	// name 少于 2 字符 & user_limit=-5 非法
	w := doPlan(t, router, http.MethodPost, "/admin/plans",
		`{"name":"x","code":"basic","user_limit":-5,"price":-1,"status":3}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotCreate, "非法参数不应触达服务层")
}

func TestCreatePlan_ServiceError_Returns500(t *testing.T) {
	stub := &stubPlanService{Err: errors.New("write failed")}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPost, "/admin/plans",
		`{"name":"基础版","code":"basic","user_limit":50,"price":99,"status":1}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "创建套餐失败")
}

func TestUpdatePlan_MissingID_Returns400(t *testing.T) {
	stub := &stubPlanService{}
	h := handler.NewPlanHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	h.UpdatePlan(w, req) // 无 chi {id}

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdatePlan_NotFound_Returns404(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrPlanNotFound}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/plans/p-x/update", `{"name":"新版"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "套餐不存在")
}

func TestUpdatePlan_Success_ForwardsID(t *testing.T) {
	stub := &stubPlanService{Resp: samplePlanResp()}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/plans/p1/update", `{"name":"基础版","user_limit":100}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", stub.GotID)
	require.NotNil(t, stub.GotUpdate)
	assert.Equal(t, 100, stub.GotUpdate.UserLimit)
}

func TestUpdatePlan_ServiceError_Returns500(t *testing.T) {
	stub := &stubPlanService{Err: errors.New("update failed")}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/plans/p1/update", `{"name":"新版"}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListPlans_Success_ForwardsQuery(t *testing.T) {
	stub := &stubPlanService{List: []*dto.PlanResp{samplePlanResp()}, Total: 1}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/plans?page=1&page_size=10&keyword=基础&status=1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, stub.GotPage)
	assert.Equal(t, 10, stub.GotPageSz)
	assert.Equal(t, "基础", stub.GotKeyword)
	require.NotNil(t, stub.GotStatus)
	assert.Equal(t, 1, *stub.GotStatus)
}

func TestListPlans_InvalidStatus_Ignored(t *testing.T) {
	stub := &stubPlanService{}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/plans?status=9", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Nil(t, stub.GotStatus)
}

func TestListPlans_ServiceError_Returns500(t *testing.T) {
	stub := &stubPlanService{Err: errors.New("scan failed")}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/plans", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListAllEnabledPlans_Success(t *testing.T) {
	stub := &stubPlanService{List: []*dto.PlanResp{samplePlanResp()}}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/plans/select", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var out struct {
		Data []*dto.PlanResp `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Len(t, out.Data, 1)
}

func TestListAllEnabledPlans_ServiceError_Returns500(t *testing.T) {
	stub := &stubPlanService{Err: errors.New("db down")}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/plans/select", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeletePlan_NotFound_Returns404(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrPlanNotFound}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodDelete, "/admin/plans/p-x/delete", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePlan_InUse_Returns409(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrPlanInUse}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodDelete, "/admin/plans/p1/delete", "")

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "无法删除")
}

func TestDeletePlan_Success(t *testing.T) {
	stub := &stubPlanService{}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodDelete, "/admin/plans/p1/delete", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "p1", stub.GotID)
}

func TestAssignPlan_MissingTenant_Returns400(t *testing.T) {
	stub := &stubPlanService{}
	h := handler.NewPlanHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	h.AssignPlan(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAssignPlan_Validation_Returns400(t *testing.T) {
	stub := &stubPlanService{}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/tenants/t1/plan", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotAssign)
}

func TestAssignPlan_ExpiresInPast_Returns400(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrExpiredAtInPast}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/tenants/t1/plan", `{"plan_id":"p1"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "到期时间不能早于当前时间")
}

func TestAssignPlan_Success_ForwardsReq(t *testing.T) {
	stub := &stubPlanService{}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/tenants/t1/plan", `{"plan_id":"p1","expires_at":"2099-01-01 00:00:00"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t1", stub.GotTenant)
	require.NotNil(t, stub.GotAssign)
	assert.Equal(t, "p1", stub.GotAssign.PlanID)
}

func TestAssignPlan_BusinessError_Returns400(t *testing.T) {
	stub := &stubPlanService{Err: errors.New("套餐已停用，无法分配")}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/tenants/t1/plan", `{"plan_id":"p1"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "套餐已停用")
}

func TestAssignPlan_PlanNotFound_Returns404(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrPlanNotFound}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodPut, "/admin/tenants/t1/plan", `{"plan_id":"p-x"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCurrentPlan_NoTenantCtx_Returns401(t *testing.T) {
	stub := &stubPlanService{}
	router := newPlanRouter(stub)

	req := httptest.NewRequest(http.MethodGet, "/tenant/current/plan", nil) // 无身份上下文
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCurrentPlan_NoSubscription_Returns404(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrSubscriptionEmpty}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/tenant/current/plan", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "尚未开通套餐")
}

func TestGetCurrentPlan_Success(t *testing.T) {
	stub := &stubPlanService{Current: &dto.CurrentPlanResp{
		TenantID: "tenant-1", PlanID: "p1", PlanName: "基础版", UserLimit: 50, CurrentUsers: 3, Status: 1,
	}}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/tenant/current/plan", "")

	var out struct {
		Data dto.CurrentPlanResp `json:"data"`
	}
	requireOKPlan(t, w, &out)
	assert.Equal(t, "tenant-1", out.Data.TenantID)
	assert.Equal(t, "p1", out.Data.PlanID)
	assert.Equal(t, "tenant-1", stub.GotTenant)
}

func TestCheckTenantPlan_NoSubscription_Returns200Null(t *testing.T) {
	stub := &stubPlanService{Err: service.ErrSubscriptionEmpty}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/tenants/t1/plan", "")

	// 无订阅时返回 200 与空响应体（前端按无套餐处理），而非 404 阻断租户详情
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), "失败")
}

func TestCheckTenantPlan_Success(t *testing.T) {
	stub := &stubPlanService{Current: &dto.CurrentPlanResp{TenantID: "t1", PlanID: "p1"}}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/tenants/t1/plan", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t1", stub.GotTenant)
}

func TestCheckTenantPlan_ServiceError_Returns500(t *testing.T) {
	stub := &stubPlanService{Err: errors.New("boom")}
	router := newPlanRouter(stub)

	w := doPlan(t, router, http.MethodGet, "/admin/tenants/t1/plan", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func requireOKPlan(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, "unexpected status, body=%s", w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
}
