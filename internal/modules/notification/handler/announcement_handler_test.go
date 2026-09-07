package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/notification/dto"
	"meteorx/internal/modules/notification/handler"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAnnouncementService 桩实现：记录入参并按预设返回
type stubAnnouncementService struct {
	err         error
	resp        *dto.AnnouncementResp
	listResp    *dto.AnnouncementListResp
	publisher   string
	createID    string
	updateID    string
	statusID    string
	status      int
	getID       string
	deleteID    string
	query       *dto.ListAnnouncementsQuery
	tenantID    string
	page        int
	pageSize    int
	createCalls int
}

func (s *stubAnnouncementService) Create(_ context.Context, publisherID string, _ dto.CreateAnnouncementReq) (*dto.AnnouncementResp, error) {
	s.createCalls++
	s.publisher = publisherID
	return s.resp, s.err
}
func (s *stubAnnouncementService) Update(_ context.Context, id string, _ dto.UpdateAnnouncementReq) (*dto.AnnouncementResp, error) {
	s.updateID = id
	return s.resp, s.err
}
func (s *stubAnnouncementService) UpdateStatus(_ context.Context, id string, status int) (*dto.AnnouncementResp, error) {
	s.statusID = id
	s.status = status
	return s.resp, s.err
}
func (s *stubAnnouncementService) GetByID(_ context.Context, id string) (*dto.AnnouncementResp, error) {
	s.getID = id
	return s.resp, s.err
}
func (s *stubAnnouncementService) Delete(_ context.Context, id string) error {
	s.deleteID = id
	return s.err
}
func (s *stubAnnouncementService) List(_ context.Context, query *dto.ListAnnouncementsQuery) (*dto.AnnouncementListResp, error) {
	s.query = query
	return s.listResp, s.err
}
func (s *stubAnnouncementService) ListForTenant(_ context.Context, tenantID string, page, pageSize int) (*dto.AnnouncementListResp, error) {
	s.tenantID = tenantID
	s.page = page
	s.pageSize = pageSize
	return s.listResp, s.err
}

func defaultResp(id string) *dto.AnnouncementResp {
	return &dto.AnnouncementResp{ID: id, Title: "标题", Content: "内容", Scope: "all", Status: 1, StatusText: "已发布"}
}

func newTestRouter(stub *stubAnnouncementService) http.Handler {
	h := handler.NewAnnouncementHandler(stub)
	r := chi.NewRouter()
	// 与管理端/租户端路由一致的挂载方式（省略权限中间件，其行为由 middleware 测试覆盖）
	r.Route("/admin/announcements", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Put("/{id}/status", h.UpdateStatus)
		r.Delete("/{id}", h.Delete)
	})
	r.Route("/announcements", func(r chi.Router) {
		r.Get("/", h.ListForTenant)
	})
	return r
}

func doJSON(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	// 模拟已通过 Auth 中间件的登录态
	req = req.WithContext(contextx.SetVars(req.Context(), "tenant-1", "user-1", []string{"admin"}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeData(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, "unexpected status, body=%s", w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
}

func TestCreate_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubAnnouncementService{}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodPost, "/admin/announcements", `{"title":"","scope":"bad"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "标题不能为空")
	assert.Zero(t, stub.createCalls, "参数非法时不应触达服务层")
}

func TestCreate_Success_ReadsPublisherFromCtx(t *testing.T) {
	stub := &stubAnnouncementService{resp: defaultResp("ann-1")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodPost, "/admin/announcements", `{"title":"t","content":"c","scope":"all","status":1}`)

	var out struct {
		Data dto.AnnouncementResp `json:"data"`
	}
	decodeData(t, w, &out)
	assert.Equal(t, "ann-1", out.Data.ID)
	assert.Equal(t, 1, stub.createCalls)
	assert.Equal(t, "user-1", stub.publisher)
}

func TestCreate_ServiceError_Returns500(t *testing.T) {
	stub := &stubAnnouncementService{err: errors.New("db down")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodPost, "/admin/announcements", `{"title":"t","content":"c","scope":"all"}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "创建公告失败")
}

func TestGet_Success(t *testing.T) {
	stub := &stubAnnouncementService{resp: defaultResp("ann-1")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodGet, "/admin/announcements/ann-1", "")

	var out struct {
		Data dto.AnnouncementResp `json:"data"`
	}
	decodeData(t, w, &out)
	assert.Equal(t, "ann-1", out.Data.ID)
	assert.Equal(t, "ann-1", stub.getID)
}

func TestGet_NotFound_Returns404(t *testing.T) {
	stub := &stubAnnouncementService{err: errors.New("not found")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodGet, "/admin/announcements/missing", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "公告不存在")
}

func TestGet_EmptyID_BadRequest(t *testing.T) {
	stub := &stubAnnouncementService{}
	h := handler.NewAnnouncementHandler(stub)

	// 绕过路由直接调用：chi 路由上下文无 {id} 参数时 URLParam 为空
	w := httptest.NewRecorder()
	h.Get(w, httptest.NewRequest(http.MethodGet, "/admin/announcements/", nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateStatus_InvalidStatus_Returns400(t *testing.T) {
	stub := &stubAnnouncementService{}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodPut, "/admin/announcements/ann-1/status", `{"status":9}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "状态")
	assert.Empty(t, stub.statusID, "非法状态不应触达服务层")
}

func TestUpdateStatus_Success_ForwardsStatus(t *testing.T) {
	stub := &stubAnnouncementService{resp: defaultResp("ann-1")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodPut, "/admin/announcements/ann-1/status", `{"status":2}`)

	var out struct {
		Data dto.AnnouncementResp `json:"data"`
	}
	decodeData(t, w, &out)
	assert.Equal(t, "ann-1", out.Data.ID)
	assert.Equal(t, "ann-1", stub.statusID)
	assert.Equal(t, 2, stub.status)
}

func TestUpdate_Success(t *testing.T) {
	stub := &stubAnnouncementService{resp: defaultResp("ann-1")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodPut, "/admin/announcements/ann-1", `{"title":"新","content":"新内容","scope":"tenant"}`)

	var out struct {
		Data dto.AnnouncementResp `json:"data"`
	}
	decodeData(t, w, &out)
	assert.Equal(t, "ann-1", out.Data.ID)
	assert.Equal(t, "ann-1", stub.updateID)
}

func TestUpdate_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubAnnouncementService{}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodPut, "/admin/announcements/ann-1", `{"title":"","scope":"bad"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "标题不能为空")
	assert.Empty(t, stub.updateID)
}

func TestList_ServiceError_Returns500(t *testing.T) {
	stub := &stubAnnouncementService{err: errors.New("list broken")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodGet, "/admin/announcements", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取公告列表失败")
}

func TestDelete_Success(t *testing.T) {
	stub := &stubAnnouncementService{}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodDelete, "/admin/announcements/ann-1", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"ann-1"`)
	assert.Equal(t, "ann-1", stub.deleteID)
}

func TestDelete_ServiceError_Returns500(t *testing.T) {
	stub := &stubAnnouncementService{err: errors.New("delete failed")}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodDelete, "/admin/announcements/ann-1", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "删除公告失败")
}

func TestList_ForwardsParsedQueryParams(t *testing.T) {
	stub := &stubAnnouncementService{listResp: &dto.AnnouncementListResp{
		Items: []*dto.AnnouncementResp{defaultResp("a"), defaultResp("b")},
		Total: 2,
	}}
	router := newTestRouter(stub)

	// status 未传 → 默认按全部(-1)处理
	w := doJSON(t, router, http.MethodGet, "/admin/announcements?page=2&page_size=50&keyword=升级&scope=all", "")

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, stub.query)
	assert.Equal(t, 2, stub.query.Page)
	assert.Equal(t, 50, stub.query.PageSize)
	assert.Equal(t, "升级", stub.query.Keyword)
	assert.Equal(t, -1, stub.query.Status)
	assert.Equal(t, "all", stub.query.Scope)
	// 分页响应结构：outer.data = PaginatedResult{data: items, pagination:{total}}
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	data, _ := out["data"].(map[string]any)
	pagination, _ := data["pagination"].(map[string]any)
	assert.Equal(t, float64(2), pagination["total"])
	assert.Equal(t, float64(2), pagination["page"])
}

func TestList_StatusFilter_PassedThrough(t *testing.T) {
	stub := &stubAnnouncementService{listResp: &dto.AnnouncementListResp{}}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodGet, "/admin/announcements?status=1", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, stub.query.Status)
}

func TestListForTenant_MissingTenantCtx_Returns401(t *testing.T) {
	stub := &stubAnnouncementService{}
	router := newTestRouter(stub)

	req := httptest.NewRequest(http.MethodGet, "/announcements", nil) // 无身份上下文
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "租户信息缺失")
	assert.Empty(t, stub.tenantID)
}

func TestListForTenant_Success_ForwardsTenantAndDefaults(t *testing.T) {
	stub := &stubAnnouncementService{listResp: &dto.AnnouncementListResp{}}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodGet, "/announcements", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "tenant-1", stub.tenantID)
	assert.Equal(t, 1, stub.page, "未传 page 应回退默认值")
	assert.Equal(t, 20, stub.pageSize, "未传 page_size 应回退默认值")
}

func TestWrongMethod_Returns405(t *testing.T) {
	stub := &stubAnnouncementService{}
	router := newTestRouter(stub)

	w := doJSON(t, router, http.MethodDelete, "/admin/announcements", "")

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
