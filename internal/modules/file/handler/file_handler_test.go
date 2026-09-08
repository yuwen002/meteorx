package handler_test

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/config"
	"meteorx/internal/modules/file/dto"
	"meteorx/internal/modules/file/handler"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFileRouter 注册 FileHandler 全部路由，cfg 可自定义白名单/上限
func newFileRouter(stub *stubFileService, cfg config.FileConfig) http.Handler {
	h := handler.NewFileHandler(stub, cfg)
	r := chi.NewRouter()
	r.Post("/files/upload", h.Upload)
	r.Get("/files/{id}", h.GetByID)
	r.Get("/files/mine", h.ListByUser)
	r.Get("/files", h.ListByTenant)
	r.Get("/files/deleted", h.GetDeletedList)
	r.Put("/files/{id}", h.Update)
	r.Delete("/files/{id}", h.Delete)
	r.Post("/files/{id}/restore", h.Restore)
	r.Delete("/files/{id}/permanent", h.PermanentDelete)
	r.Post("/files/batch/delete", h.BatchDelete)
	r.Get("/files/{id}/download", h.Download)
	return r
}

// doFile 发送带身份上下文的 JSON 请求
func doFile(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
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

// uploadReq 构造 multipart 上传请求（可指定文件名与 Content-Type）
func uploadReq(t *testing.T, router http.Handler, filename, contentType string, withCtx bool) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := mw.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte("fake file content"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/files/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if withCtx {
		req = req.WithContext(contextx.SetVars(req.Context(), "tenant-1", "user-1", []string{"admin"}))
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ============ 上传 ============

func TestUpload_Success_ForwardsHeaderAndCtx(t *testing.T) {
	stub := &stubFileService{UploadResp: &dto.UploadFileResp{ID: "f1", FileName: "a.png", OriginalName: "a.png", FileType: "image"}}
	router := newFileRouter(stub, config.FileConfig{})

	w := uploadReq(t, router, "a.png", "image/png", true)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotHeader)
	assert.Equal(t, "image/png", stub.GotHeader.Header.Get("Content-Type"))
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, "user-1", stub.GotUserID)
	assert.Contains(t, w.Body.String(), `"f1"`)
}

func TestUpload_UnsupportedType_Returns400(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	w := uploadReq(t, router, "evil.exe", "application/x-executable", true)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "不支持的文件类型")
	assert.Empty(t, stub.UploadResp)
}

func TestUpload_CustomAllowedTypes_Respected(t *testing.T) {
	// 只允许 pdf，png 应被拒绝
	cfg := config.FileConfig{AllowedTypes: []string{"application/pdf"}}
	stub := &stubFileService{UploadResp: &dto.UploadFileResp{ID: "f1"}}
	router := newFileRouter(stub, cfg)

	w := uploadReq(t, router, "a.png", "image/png", true)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "不支持的文件类型")
}

func TestUpload_MissingFileField_Returns400(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	// 空 multipart 表单，无 file 字段
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/files/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = req.WithContext(contextx.SetVars(req.Context(), "tenant-1", "user-1", []string{"admin"}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "未找到上传文件")
}

func TestUpload_FileTooLarge_Returns400(t *testing.T) {
	stub := &stubFileService{}
	// 上限 4 字节，任何真实文件都会超限
	cfg := config.FileConfig{MaxFileSize: 4}
	router := newFileRouter(stub, cfg)

	w := uploadReq(t, router, "a.png", "image/png", true)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "文件过大或解析失败")
}

func TestUpload_NoAuth_Returns401(t *testing.T) {
	stub := &stubFileService{UploadResp: &dto.UploadFileResp{ID: "f1"}}
	router := newFileRouter(stub, config.FileConfig{})

	w := uploadReq(t, router, "a.png", "image/png", false)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未授权")
}

func TestUpload_ServiceError_Returns500(t *testing.T) {
	stub := &stubFileService{Err: errors.New("磁盘已满")}
	router := newFileRouter(stub, config.FileConfig{})

	w := uploadReq(t, router, "a.png", "image/png", true)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "文件上传失败")
	assert.Contains(t, w.Body.String(), "磁盘已满")
}

// ============ 详情 / 列表 ============

func TestGetByID_MissingID_Returns400(t *testing.T) {
	stub := &stubFileService{}
	h := handler.NewFileHandler(stub, config.FileConfig{})

	// 无 {id} 路径参数（如路由不匹配时兜底调用）
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.GetByID(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "文件ID不能为空")
}

func TestGetByID_Success_ScopesByTenant(t *testing.T) {
	stub := &stubFileService{File: &dto.FileResp{ID: "f1", FileName: "a.png", OriginalName: "a.png", TenantID: "tenant-1"}}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files/f1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "f1", stub.GotID)
	assert.Equal(t, "tenant-1", stub.GotTenantID, "详情必须按租户范围隔离")
	assert.Contains(t, w.Body.String(), `"f1"`)
}

func TestGetByID_NotFound_Returns404(t *testing.T) {
	stub := &stubFileService{Err: errors.New("not found")}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files/f-x", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "文件不存在")
}

func TestListByTenant_Success_ForwardsFilters(t *testing.T) {
	stub := &stubFileService{Files: []*dto.FileResp{{ID: "f1", FileName: "a.png"}}, Total: 1}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files?page=1&page_size=10&file_type=image&keyword=a", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	require.NotNil(t, stub.GotListReq)
	assert.Equal(t, "image", stub.GotListReq.FileType)
	assert.Equal(t, "a", stub.GotListReq.Keyword)
	assert.Equal(t, 10, stub.GotListReq.PageSize)
	assert.Contains(t, w.Body.String(), `"f1"`)
}

func TestListByTenant_NoAuth_Returns401(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	req := httptest.NewRequest(http.MethodGet, "/files", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListByTenant_ServiceError_Returns500(t *testing.T) {
	stub := &stubFileService{Err: errors.New("query failed")}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取文件列表失败")
}

func TestListByUser_Success_ForwardsUserScope(t *testing.T) {
	stub := &stubFileService{Files: []*dto.FileResp{{ID: "f2", UserID: "user-1"}}, Total: 1}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files/mine", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, "user-1", stub.GotUserID)
	assert.Contains(t, w.Body.String(), `"f2"`)
}

func TestListByUser_ServiceError_Returns500(t *testing.T) {
	stub := &stubFileService{Err: errors.New("boom")}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files/mine", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetDeletedList_Success_ForwardsPage(t *testing.T) {
	stub := &stubFileService{Files: []*dto.FileResp{{ID: "f9", Status: 0}}, Total: 1}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files/deleted?page=2&page_size=20", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 2, stub.GotPage)
	assert.Equal(t, 20, stub.GotPageSize)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Contains(t, w.Body.String(), `"f9"`)
}

// ============ 更新 / 删除 / 回收站 ============

func TestUpdate_Success_ForwardsReq(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodPut, "/files/f1", `{"file_name":"改名.txt"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "f1", stub.GotID)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	require.NotNil(t, stub.GotUpdateReq)
	assert.Equal(t, "改名.txt", stub.GotUpdateReq.FileName)
}

func TestUpdate_InvalidJSON_Returns400(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodPut, "/files/f1", `{"file_name":`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "参数错误")
	assert.Empty(t, stub.GotUpdateReq)
}

func TestUpdate_ServiceError_Returns500(t *testing.T) {
	stub := &stubFileService{Err: errors.New("update failed")}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodPut, "/files/f1", `{"file_name":"x"}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDelete_MissingID_Returns400(t *testing.T) {
	stub := &stubFileService{}
	h := handler.NewFileHandler(stub, config.FileConfig{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	h.Delete(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "文件ID不能为空")
}

func TestDelete_Success_ScopedByTenant(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodDelete, "/files/f1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "f1", stub.GotID)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
}

func TestBatchDelete_Success_ForwardsReq(t *testing.T) {
	stub := &stubFileService{BatchResp: &dto.BatchDeleteResp{SuccessCount: 2, FailedIDs: []string{"f-x"}}}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodPost, "/files/batch/delete", `{"ids":["f1","f2","f-x"]}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotBatchReq)
	assert.Equal(t, []string{"f1", "f2", "f-x"}, stub.GotBatchReq.IDs)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Contains(t, w.Body.String(), `"success_count":2`)
	assert.Contains(t, w.Body.String(), `"f-x"`)
}

func TestBatchDelete_InvalidJSON_Returns400(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodPost, "/files/batch/delete", `{"ids":`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.GotBatchReq)
}

func TestRestore_Success_ForwardsTenant(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodPost, "/files/f1/restore", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "f1", stub.GotID)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
}

func TestPermanentDelete_Success(t *testing.T) {
	stub := &stubFileService{}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodDelete, "/files/f1/permanent", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "f1", stub.GotID)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
}

func TestPermanentDelete_ServiceError_Returns500(t *testing.T) {
	stub := &stubFileService{Err: errors.New("delete failed")}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodDelete, "/files/f1/permanent", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============ 下载 ============

func TestDownload_Success_StreamsContent(t *testing.T) {
	stub := &stubFileService{Reader: io.NopCloser(strings.NewReader("hello-file")), Filename: "a.txt"}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files/f1/download", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, "attachment; filename=\"a.txt\"", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "hello-file", w.Body.String())
}

func TestDownload_NotFound_Returns404(t *testing.T) {
	stub := &stubFileService{Err: errors.New("not found")}
	router := newFileRouter(stub, config.FileConfig{})

	w := doFile(t, router, http.MethodGet, "/files/f-x/download", "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "文件不存在或下载失败")
}
