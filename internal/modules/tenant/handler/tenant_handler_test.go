package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"meteorx/internal/common/contextx"
	planDto "meteorx/internal/modules/plan/dto"
	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/handler"
	tenantModel "meteorx/internal/modules/tenant/model"
	"meteorx/internal/modules/tenant/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ================= 租户桩 =================

// tenantStub 桩：嵌入具体服务（为保持真实方法签名集合），覆写被测方法。
type tenantStub struct {
	handler.TenantService

	err    error
	calls  []string
	params map[string][]string
}

func newTenantStub() *tenantStub {
	return &tenantStub{params: map[string][]string{}}
}

func (s *tenantStub) rec(method string, args ...string) {
	s.calls = append(s.calls, method)
	s.params[method] = args
}

func (s *tenantStub) failWith(err error) *tenantStub {
	s.err = err
	return s
}

func (s *tenantStub) assertCalled(t *testing.T, method string) {
	t.Helper()
	assert.Contains(t, s.calls, method, "stub 应收到调用 %s", method)
}

func sampleTenant(id, name, domain string) *tenantModel.Tenant {
	return &tenantModel.Tenant{ID: id, Name: name, Domain: domain, Status: 1}
}

// ---------- 覆写 handler 所需的方法 ----------

func (s *tenantStub) Register(_ context.Context, req dto.RegisterTenantReq) (*tenantModel.Tenant, error) {
	s.rec("Register", req.Name, req.Domain, req.AdminUser.Username)
	if s.err != nil {
		return nil, s.err
	}
	return sampleTenant("t-new", req.Name, req.Domain), nil
}

func (s *tenantStub) AdminCreate(_ context.Context, req dto.AdminCreateTenantReq) (*tenantModel.Tenant, error) {
	s.rec("AdminCreate")
	if s.err != nil {
		return nil, s.err
	}
	return sampleTenant("t-new", req.Name, req.Domain), nil
}

func (s *tenantStub) UpdateTenantStatus(_ context.Context, id string, status int) error {
	s.rec("UpdateTenantStatus", id, strconv.Itoa(status))
	return s.err
}

func (s *tenantStub) QueryTenantList(_ context.Context, page, pageSize int, name string, status *int) ([]*tenantModel.Tenant, int64, error) {
	statusVal := -1
	if status != nil {
		statusVal = *status
	}
	s.rec("QueryTenantList", strconv.Itoa(page), strconv.Itoa(pageSize), name, strconv.Itoa(statusVal))
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*tenantModel.Tenant{sampleTenant("t-1", name, "acme.example.com")}, 1, nil
}

func (s *tenantStub) GetTenantPlanBriefs(_ context.Context, _ []string) (map[string]*planDto.TenantPlanBrief, error) {
	s.rec("GetTenantPlanBriefs")
	return map[string]*planDto.TenantPlanBrief{"t-1": {PlanName: "专业版"}}, nil
}

func (s *tenantStub) AdminDetail(_ context.Context, id string) (*tenantModel.Tenant, error) {
	s.rec("AdminDetail", id)
	if s.err != nil {
		return nil, s.err
	}
	return sampleTenant(id, "深度租户", "deep.example.com"), nil
}

func (s *tenantStub) AdminUpdate(_ context.Context, id string, _ dto.AdminUpdateTenantReq) error {
	s.rec("AdminUpdate", id)
	return s.err
}

func (s *tenantStub) AdminDelete(_ context.Context, id string) error {
	s.rec("AdminDelete", id)
	return s.err
}

func (s *tenantStub) AdminHardDelete(_ context.Context, id string) error {
	s.rec("AdminHardDelete", id)
	return s.err
}

func (s *tenantStub) AdminUpdatePlan(_ context.Context, id string, _ planDto.AssignPlanReq) error {
	s.rec("AdminUpdatePlan", id)
	return s.err
}

func (s *tenantStub) BatchUpdateStatus(_ context.Context, ids []string, status int) (int64, []string, error) {
	s.rec("BatchUpdateStatus", strconv.Itoa(status))
	if s.err != nil {
		return 0, nil, s.err
	}
	return 2, []string{"t-x"}, nil
}

func (s *tenantStub) BatchDelete(_ context.Context, ids []string) (int64, []string, error) {
	s.rec("BatchDelete")
	if s.err != nil {
		return 0, nil, s.err
	}
	return int64(len(ids)), nil, nil
}

func (s *tenantStub) FindDeleted(_ context.Context, page, pageSize int, name string) ([]*tenantModel.Tenant, int64, error) {
	s.rec("FindDeleted", strconv.Itoa(page), strconv.Itoa(pageSize), name)
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*tenantModel.Tenant{sampleTenant("t-del", name, "del.example.com")}, 1, nil
}

func (s *tenantStub) Restore(_ context.Context, id string) error {
	s.rec("Restore", id)
	return s.err
}

func (s *tenantStub) GetCurrentTenant(_ context.Context, tenantID string) (*tenantModel.Tenant, error) {
	s.rec("GetCurrentTenant", tenantID)
	if s.err != nil {
		return nil, s.err
	}
	return sampleTenant(tenantID, "当前租户", "me.example.com"), nil
}

func (s *tenantStub) UpdateCurrentTenant(_ context.Context, tenantID string, _ dto.UpdateCurrentTenantReq) error {
	s.rec("UpdateCurrentTenant", tenantID)
	return s.err
}

func (s *tenantStub) GetInitStatus(_ context.Context, tenantID string) (*dto.GetInitStatusResp, error) {
	s.rec("GetInitStatus", tenantID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.GetInitStatusResp{Status: "completed", Progress: 100, Initialized: true}, nil
}

func (s *tenantStub) ApplyCancellation(_ context.Context, tenantID string, req dto.ApplyCancellationReq) (*dto.ApplyCancellationResp, error) {
	s.rec("ApplyCancellation", tenantID, req.Reason)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.ApplyCancellationResp{AppliedAt: "2026-01-01", Status: "pending", EstimatedDay: 7}, nil
}

func (s *tenantStub) ListCancelRequests(_ context.Context, page, pageSize, status int, keyword string) (*dto.CancelRequestListResp, error) {
	s.rec("ListCancelRequests", strconv.Itoa(page), strconv.Itoa(pageSize), strconv.Itoa(status), keyword)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.CancelRequestListResp{Items: []*dto.CancelRequestResp{{ID: "cr-1", TenantName: "Acme", Status: 1}}, Total: 1}, nil
}

func (s *tenantStub) ApproveCancellation(_ context.Context, requestID, approverID string, _ dto.AdminApproveCancelReq) (*dto.CancelRequestResp, error) {
	s.rec("ApproveCancellation", requestID, approverID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.CancelRequestResp{ID: requestID, Status: 2, StatusText: "已通过"}, nil
}

func (s *tenantStub) RejectCancellation(_ context.Context, requestID, approverID string, _ dto.AdminRejectCancelReq) (*dto.CancelRequestResp, error) {
	s.rec("RejectCancellation", requestID, approverID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.CancelRequestResp{ID: requestID, Status: 3, StatusText: "已驳回"}, nil
}

// ================= 路由与辅助 =================

const (
	tTenantID = "t-1"
	tUserID   = "u-admin"
)

func withTenantIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := contextx.SetVars(r.Context(), tTenantID, tUserID, []string{"admin"})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// newTenantRouter 构造租户主路由（identity=true 时注入租户身份）
func newTenantRouter(stub *tenantStub, identity bool) http.Handler {
	h := handler.NewTenantHandler(stub)
	r := chi.NewRouter()
	if identity {
		r.Use(withTenantIdentity)
	}
	r.Post("/tenants/register", h.Register)
	r.Post("/admin/tenants", h.AdminCreate)
	r.Put("/admin/tenants/{id}/status", h.AdminUpdateStatus)
	r.Get("/admin/tenants", h.List)
	r.Get("/admin/tenants/{id}/detail", h.AdminDetail)
	r.Put("/admin/tenants/{id}/update", h.AdminUpdate)
	r.Delete("/admin/tenants/{id}/delete", h.AdminDelete)
	r.Delete("/admin/tenants/{id}/hard", h.AdminHardDelete)
	r.Put("/admin/tenants/{id}/plan", h.AdminUpdatePlan)
	r.Put("/admin/tenants/batch/status", h.AdminBatchUpdateStatus)
	r.Delete("/admin/tenants/batch", h.AdminBatchDelete)
	r.Get("/admin/tenants/deleted", h.AdminDeletedList)
	r.Put("/admin/tenants/{id}/restore", h.AdminRestore)
	r.Get("/tenants/current", h.GetCurrentTenant)
	r.Put("/tenants/current", h.UpdateCurrentTenant)
	r.Get("/tenants/current/status", h.GetInitStatus)
	r.Post("/tenants/current/cancel", h.ApplyCancellation)
	r.Get("/admin/cancel-requests", h.AdminListCancelRequests)
	r.Put("/admin/cancel-requests/{id}/approve", h.AdminApproveCancel)
	r.Put("/admin/cancel-requests/{id}/reject", h.AdminRejectCancel)
	return r
}

func doTenant(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// assertTenantData 断言 200 且 data 内包含关键字
func assertTenantData(t *testing.T, w *httptest.ResponseRecorder, keyword string) {
	t.Helper()
	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), keyword)
}

// ================= 注册 / 后台创建 =================

func TestTenantRegister_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	body := `{"name":"Acme 科技","domain":"acme","contact_email":"a@x.com","admin_user":{"username":"boss01","password":"pass1234","nickname":"管理员"}}`
	w := doTenant(t, router, http.MethodPost, "/tenants/register", body)

	assertTenantData(t, w, `"domain":"acme"`)
	assert.Equal(t, []string{"Acme 科技", "acme", "boss01"}, stub.params["Register"])
}

func TestTenantRegister_InvalidBody_Returns400(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPost, "/tenants/register", `{"name":"Acme","domain":"acme"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestTenantRegister_DomainConflict_Returns409(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrDomainConflict)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPost, "/tenants/register", `{"name":"Acme 科技","domain":"acme","admin_user":{"username":"boss01","password":"pass1234","nickname":"管理员"}}`)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "该域名已被其他租户占用")
}

func TestTenantRegister_UsernameConflict_Returns409(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrUsernameConflict)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPost, "/tenants/register", `{"name":"Acme 科技","domain":"acme2","admin_user":{"username":"boss01","password":"pass1234","nickname":"管理员"}}`)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "该用户名已被使用")
}

func TestTenantRegister_UnexpectedError_Returns500(t *testing.T) {
	stub := newTenantStub().failWith(assert.AnError)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPost, "/tenants/register", `{"name":"Acme 科技","domain":"acme","admin_user":{"username":"boss01","password":"pass1234","nickname":"管理员"}}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "服务器开小差了")
}

func TestTenantAdminCreate_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	body := `{"name":"Acme","domain":"acme","status":1,"admin_user":{"username":"boss01","password":"pass1234","nickname":"管理员"}}`
	w := doTenant(t, router, http.MethodPost, "/admin/tenants", body)

	assertTenantData(t, w, `"name":"Acme"`)
	stub.assertCalled(t, "AdminCreate")
}

func TestTenantAdminCreate_DomainConflict_Returns409(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrDomainConflict)
	router := newTenantRouter(stub, false)

	body := `{"name":"Acme","domain":"acme","admin_user":{"username":"boss01","password":"pass1234","nickname":"管理员"}}`
	w := doTenant(t, router, http.MethodPost, "/admin/tenants", body)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestTenantAdminCreate_MissingAdminUser_Returns400(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPost, "/admin/tenants", `{"name":"Acme","domain":"acme"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

// ================= 后台租户管理 =================

func TestTenantAdminUpdateStatus_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/t-1/status", `{"status":0}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"t-1", "0"}, stub.params["UpdateTenantStatus"])
}

func TestTenantAdminUpdateStatus_MissingStatus_Returns400(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/t-1/status", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestTenantAdminUpdateStatus_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/ghost/status", `{"status":0}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "租户不存在")
}

func TestTenantList_QueryAndPlanEnrichment(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodGet, "/admin/tenants?name=Acme&status=1&page=2&page_size=5", "")

	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"plan_name":"专业版"`)
	assert.Contains(t, w.Body.String(), `"total":1`)
	assert.Equal(t, []string{"2", "5", "Acme", "1"}, stub.params["QueryTenantList"])
	stub.assertCalled(t, "GetTenantPlanBriefs")
}

func TestTenantAdminDetail_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodGet, "/admin/tenants/t-9/detail", "")

	assertTenantData(t, w, `"name":"深度租户"`)
	assert.Equal(t, []string{"t-9"}, stub.params["AdminDetail"])
}

func TestTenantAdminDetail_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodGet, "/admin/tenants/ghost/detail", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenantAdminUpdate_Conflict_Returns409(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrNameConflict)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/t-1/update", `{"name":"Acme"}`)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "该租户名称已被使用")
}

func TestTenantAdminUpdate_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/ghost/update", `{"name":"Acme"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenantAdminUpdate_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/t-1/update", `{"name":"Acme 新"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"t-1"}, stub.params["AdminUpdate"])
}

func TestTenantAdminDelete_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodDelete, "/admin/tenants/t-1/delete", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"t-1"}, stub.params["AdminDelete"])
}

func TestTenantAdminDelete_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodDelete, "/admin/tenants/ghost/delete", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenantAdminHardDelete_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodDelete, "/admin/tenants/ghost/hard", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, []string{"ghost"}, stub.params["AdminHardDelete"])
}

func TestTenantAdminUpdatePlan_ServiceError_Returns400(t *testing.T) {
	stub := newTenantStub().failWith(assert.AnError)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/t-1/plan", `{"plan_id":"p-1","effective_from":"2026-01-01"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, []string{"t-1"}, stub.params["AdminUpdatePlan"])
}

func TestTenantAdminUpdatePlan_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/ghost/plan", `{"plan_id":"p-1"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ================= 批量操作 =================

func TestTenantBatchUpdateStatus_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/batch/status", `{"ids":["t-1","t-2","t-3"],"status":0}`)

	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var out struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Contains(t, string(out.Data["total"]), "3")
	assert.Contains(t, string(out.Data["succeeded"]), "2")
	assert.Contains(t, string(out.Data["failed"]), "1")
	assert.Contains(t, string(out.Data["failed_ids"]), "t-x")
	assert.Equal(t, []string{"0"}, stub.params["BatchUpdateStatus"])
}

func TestTenantBatchUpdateStatus_MissingIDs_Returns400(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/batch/status", `{"status":0}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestTenantBatchDelete_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodDelete, "/admin/tenants/batch", `{"ids":["t-1","t-2"]}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"succeeded":2`)
	stub.assertCalled(t, "BatchDelete")
}

// ================= 回收站 =================

func TestTenantDeletedList_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodGet, "/admin/tenants/deleted?name=Acme", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":1`)
	assert.Equal(t, []string{"1", "20", "Acme"}, stub.params["FindDeleted"])
}

func TestTenantRestore_NotDeleted_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotDeleted)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/t-1/restore", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "租户不存在或未处于已删除状态")
}

func TestTenantRestore_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/tenants/t-1/restore", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"t-1"}, stub.params["Restore"])
}

// ================= 当前租户自助操作 =================

func TestTenantGetCurrent_MissingTenant_Returns401(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false) // 无身份

	w := doTenant(t, router, http.MethodGet, "/tenants/current", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, stub.calls)
}

func TestTenantGetCurrent_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodGet, "/tenants/current", "")

	assertTenantData(t, w, `"name":"当前租户"`)
	assert.Equal(t, []string{tTenantID}, stub.params["GetCurrentTenant"])
}

func TestTenantGetCurrent_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodGet, "/tenants/current", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenantUpdateCurrent_NameConflict_Returns409(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrNameConflict)
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodPut, "/tenants/current", `{"name":"Acme 新"}`)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestTenantUpdateCurrent_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodPut, "/tenants/current", `{"name":"Acme 新"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{tTenantID}, stub.params["UpdateCurrentTenant"])
}

func TestTenantGetInitStatus_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodGet, "/tenants/current/status", "")

	assertTenantData(t, w, `"initialized":true`)
	assert.Equal(t, []string{tTenantID}, stub.params["GetInitStatus"])
}

func TestTenantGetInitStatus_MissingTenant_Returns401(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodGet, "/tenants/current/status", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ================= 注销流程 =================

func TestTenantApplyCancellation_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodPost, "/tenants/current/cancel", `{"reason":"业务调整"}`)

	assertTenantData(t, w, `"status":"pending"`)
	assert.Equal(t, []string{tTenantID, "业务调整"}, stub.params["ApplyCancellation"])
}

func TestTenantApplyCancellation_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodPost, "/tenants/current/cancel", `{"reason":"业务调整"}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenantListCancelRequests_Defaults(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodGet, "/admin/cancel-requests", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":1`)
	assert.Equal(t, []string{"1", "20", "0", ""}, stub.params["ListCancelRequests"])
}

func TestTenantApproveCancellation_WithApproverFromContext(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodPut, "/admin/cancel-requests/cr-1/approve", `{"effective_days":3}`)

	assertTenantData(t, w, `"status_text":"已通过"`)
	assert.Equal(t, []string{"cr-1", tUserID}, stub.params["ApproveCancellation"])
}

func TestTenantApproveCancellation_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/cancel-requests/ghost/approve", `{"effective_days":0}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenantApproveCancellation_OtherError_Returns400(t *testing.T) {
	stub := newTenantStub().failWith(assert.AnError)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/cancel-requests/cr-1/approve", `{"effective_days":0}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantRejectCancellation_Success(t *testing.T) {
	stub := newTenantStub()
	router := newTenantRouter(stub, true)

	w := doTenant(t, router, http.MethodPut, "/admin/cancel-requests/cr-1/reject", `{"review_remark":"信息不全"}`)

	assertTenantData(t, w, `"status_text":"已驳回"`)
	assert.Equal(t, []string{"cr-1", tUserID}, stub.params["RejectCancellation"])
}

func TestTenantRejectCancellation_NotFound_Returns404(t *testing.T) {
	stub := newTenantStub().failWith(service.ErrTenantNotFound)
	router := newTenantRouter(stub, false)

	w := doTenant(t, router, http.MethodPut, "/admin/cancel-requests/ghost/reject", `{}`)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
