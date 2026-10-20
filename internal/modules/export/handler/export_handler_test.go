package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/export/handler"
	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// newExportRouter 复刻 modules/export routes.go 的路由形状，保证 {id} URLParam 注入真实生效
func newExportRouter(h *handler.ExportHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/api/v1/exports", h.List)
	r.Post("/api/v1/exports", h.Create)
	r.Get("/api/v1/exports/{id}", h.Get)
	r.Get("/api/v1/exports/{id}/download", h.Download)
	r.Delete("/api/v1/exports/{id}", h.Delete)
	return r
}

// newRequest 构造请求；tenantID 非空时注入 contextx 租户/用户上下文
func newRequest(method, target, body, tenantID, userID string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if tenantID != "" {
		r = r.WithContext(contextx.SetVars(r.Context(), tenantID, userID, nil))
	}
	return r
}

// completedJob 构造一个已完成的导出任务（文件路径可指定）
func completedJob(id, filePath string) *model.ExportJob {
	now := time.Now()
	return &model.ExportJob{
		ID:        id,
		TenantID:  "t1",
		UserID:    "u1",
		Resource:  model.ResourceUser,
		Status:    model.ExportStatusCompleted,
		FilePath:  filePath,
		CreatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
}

// ---------- Create ----------

func TestHandler_Create_Success(t *testing.T) {
	svc := &stubExportService{createJob: completedJob("exp1", "")}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodPost, "/api/v1/exports", `{"resource":"user","filters":{"status":"1"}}`, "t1", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "t1", svc.gotTenant)
	require.Equal(t, "u1", svc.gotUser)
	require.Equal(t, model.ResourceUser, svc.gotCreate.Resource)

	var body struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "exp1", body.Data.ID)
}

func TestHandler_Create_MissingIdentity(t *testing.T) {
	h := handler.NewExportHandler(&stubExportService{})

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodPost, "/api/v1/exports", `{"resource":"user"}`, "", ""))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Create_BadResourceRejectedByValidator(t *testing.T) {
	svc := &stubExportService{}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodPost, "/api/v1/exports", `{"resource":"unknown"}`, "t1", "u1"))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Empty(t, svc.gotCreate.Resource) // 校验失败不应触达服务层
}

func TestHandler_Create_ErrorMapping(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
	}{
		{"queue_full_429", service.ErrExportQueueFull, http.StatusTooManyRequests},
		{"invalid_resource_400", service.ErrInvalidResource, http.StatusBadRequest},
		{"internal_500", errBoom, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubExportService{createErr: tc.err}
			h := handler.NewExportHandler(svc)
			w := httptest.NewRecorder()
			newExportRouter(h).ServeHTTP(w, newRequest(http.MethodPost, "/api/v1/exports", `{"resource":"user"}`, "t1", "u1"))
			require.Equal(t, tc.status, w.Code)
		})
	}
}

var errBoom = errors.New("boom")

// ---------- List ----------

func TestHandler_List_PassesQuery(t *testing.T) {
	svc := &stubExportService{
		listItems: []*model.ExportJob{completedJob("exp1", ""), completedJob("exp2", "")},
		listTotal: 2,
	}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports?page=2&page_size=50&status=completed&resource=user", "", "t1", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 2, svc.gotList.Page)
	require.Equal(t, 50, svc.gotList.PageSize)
	require.Equal(t, "completed", svc.gotList.Status)
	require.Equal(t, "user", svc.gotList.Resource)

	var body struct {
		Data struct {
			Data       []map[string]any `json:"data"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Data, 2)
	require.Equal(t, 2, body.Data.Pagination.Total)
}

func TestHandler_List_MissingIdentity(t *testing.T) {
	h := handler.NewExportHandler(&stubExportService{})
	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports", "", "", ""))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------- Get ----------

func TestHandler_Get_Found(t *testing.T) {
	svc := &stubExportService{getJob: completedJob("exp1", "")}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports/exp1", "", "t1", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "exp1", svc.gotID)
	require.False(t, svc.gotMaster) // 普通租户非 master
}

func TestHandler_Get_MasterIdentity(t *testing.T) {
	svc := &stubExportService{getJob: completedJob("exp1", "")}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports/exp1", "", contextx.SystemTenantID, "root"))

	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, svc.gotMaster) // SYSTEM_ROOT 虚拟租户判定为 master
}

func TestHandler_Get_ErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{service.ErrExportNotFound, http.StatusNotFound},
		{service.ErrExportForbidden, http.StatusForbidden},
	}
	for _, tc := range cases {
		svc := &stubExportService{getErr: tc.err}
		h := handler.NewExportHandler(svc)
		w := httptest.NewRecorder()
		newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports/exp1", "", "t1", "u1"))
		require.Equal(t, tc.status, w.Code)
	}
}

// ---------- Download ----------

func TestHandler_Download_NotCompleted_409(t *testing.T) {
	job := completedJob("exp1", "")
	job.Status = model.ExportStatusPending
	svc := &stubExportService{getJob: job}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports/exp1/download", "", "t1", "u1"))
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Download_FileMissing_404(t *testing.T) {
	svc := &stubExportService{getJob: completedJob("exp1", filepath.Join(t.TempDir(), "gone.csv"))}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports/exp1/download", "", "t1", "u1"))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Download_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exp1.csv")
	content := "id,name\n1,alice\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	svc := &stubExportService{getJob: completedJob("exp1", path)}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodGet, "/api/v1/exports/exp1/download", "", "t1", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, w.Header().Get("Content-Disposition"), `attachment; filename="exp1.csv"`)
	require.Equal(t, content, w.Body.String())
}

// ---------- Delete ----------

func TestHandler_Delete_OK(t *testing.T) {
	svc := &stubExportService{delJob: completedJob("exp1", filepath.Join(t.TempDir(), "exp1.csv"))}
	h := handler.NewExportHandler(svc)

	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodDelete, "/api/v1/exports/exp1", "", "t1", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "exp1", svc.gotID)
	require.Equal(t, "t1", svc.gotTenant)
	require.Equal(t, "u1", svc.gotUser)
	// 文件删除已下沉到 service.Delete（先文件后记录），具体行为由 service 层用例覆盖；
	// handler 不得再自行 os.Remove，否则错误被丢弃且记录已先删，无法重试。
}

// TestHandler_Delete_FileInUseMapping 文件被占用时返回 409，供前端提示重试。
func TestHandler_Delete_FileInUseMapping(t *testing.T) {
	svc := &stubExportService{delErr: service.ErrExportFileInUse}
	h := handler.NewExportHandler(svc)
	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodDelete, "/api/v1/exports/exp1", "", "t1", "u1"))
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Delete_ErrorMapping(t *testing.T) {
	svc := &stubExportService{delErr: service.ErrExportNotFound}
	h := handler.NewExportHandler(svc)
	w := httptest.NewRecorder()
	newExportRouter(h).ServeHTTP(w, newRequest(http.MethodDelete, "/api/v1/exports/exp1", "", "t1", "u1"))
	require.Equal(t, http.StatusNotFound, w.Code)
}
