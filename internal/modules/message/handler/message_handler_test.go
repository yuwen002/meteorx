package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/message/dto"
	"meteorx/internal/modules/message/handler"
	"meteorx/internal/modules/message/model"
	messagerepo "meteorx/internal/modules/message/repository"
	"meteorx/internal/modules/message/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// stubMessageService 记录入参并返回预设结果，用于隔离 handler 的 HTTP 语义。
type stubMessageService struct {
	items      []*dto.MessageResp
	total      int64
	listErr    error
	gotFilter  messagerepo.ListFilter
	unread     *dto.UnreadCountResp
	unreadErr  error
	gotUserID  string
	gotID      string
	gotCat     string
	markErr    error
	markAllN   int64
	markAllErr error
	deleteErr  error
}

func (s *stubMessageService) List(_ context.Context, f messagerepo.ListFilter) ([]*dto.MessageResp, int64, error) {
	s.gotFilter = f
	return s.items, s.total, s.listErr
}

func (s *stubMessageService) UnreadCount(_ context.Context, userID string) (*dto.UnreadCountResp, error) {
	s.gotUserID = userID
	return s.unread, s.unreadErr
}

func (s *stubMessageService) MarkRead(_ context.Context, userID, id string) error {
	s.gotUserID, s.gotID = userID, id
	return s.markErr
}

func (s *stubMessageService) MarkAllRead(_ context.Context, userID, category string) (int64, error) {
	s.gotUserID, s.gotCat = userID, category
	return s.markAllN, s.markAllErr
}

func (s *stubMessageService) Delete(_ context.Context, userID, id string) error {
	s.gotUserID, s.gotID = userID, id
	return s.deleteErr
}

var errBoom = errors.New("boom")

// newMessageRouter 复刻 module.go 的路由形状，保证 {id} URLParam 与静态段优先级真实生效。
func newMessageRouter(h *handler.MessageHandler) chi.Router {
	r := chi.NewRouter()
	r.Route("/api/v1/messages", func(r chi.Router) {
		r.Get("/unread-count", h.UnreadCount)
		r.Put("/read-all", h.MarkAllRead)
		r.Get("/", h.List)
		r.Put("/{id}/read", h.MarkRead)
		r.Delete("/{id}", h.Delete)
	})
	return r
}

func newMessageRequest(method, target, body, userID string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		r = r.WithContext(contextx.SetVars(r.Context(), "t1", userID, nil))
	}
	return r
}

func TestHandler_List_PassesQuery(t *testing.T) {
	svc := &stubMessageService{
		items: []*dto.MessageResp{{ID: "m1", Category: model.CategoryTask, Title: "t"}},
		total: 1,
	}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodGet,
		"/api/v1/messages/?page=2&page_size=50&category=task&unread=true&keyword=hello", "", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "u1", svc.gotFilter.UserID)
	require.Equal(t, model.CategoryTask, svc.gotFilter.Category)
	require.True(t, svc.gotFilter.UnreadOnly)
	require.Equal(t, "hello", svc.gotFilter.Keyword)
	require.Equal(t, 2, svc.gotFilter.Page)
	require.Equal(t, 50, svc.gotFilter.PageSize)

	var body struct {
		Data struct {
			Data       []map[string]any `json:"data"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Data, 1)
	require.Equal(t, 1, body.Data.Pagination.Total)
}

// TestHandler_List_InvalidCategory 分类拼错必须报 400，不能静默返回全部消息。
func TestHandler_List_InvalidCategory(t *testing.T) {
	svc := &stubMessageService{}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodGet, "/api/v1/messages/?category=nope", "", "u1"))

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Empty(t, svc.gotFilter.UserID, "校验失败不应触达服务层")
}

func TestHandler_List_MissingIdentity(t *testing.T) {
	h := handler.NewMessageHandler(&stubMessageService{})
	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodGet, "/api/v1/messages/", "", ""))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_List_ServiceError(t *testing.T) {
	h := handler.NewMessageHandler(&stubMessageService{listErr: errBoom})
	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodGet, "/api/v1/messages/", "", "u1"))
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_UnreadCount(t *testing.T) {
	svc := &stubMessageService{unread: &dto.UnreadCountResp{Total: 3, ByCategory: map[string]int64{"task": 3}}}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodGet, "/api/v1/messages/unread-count", "", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "u1", svc.gotUserID)

	var body struct {
		Data dto.UnreadCountResp `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(3), body.Data.Total)
}

func TestHandler_MarkRead(t *testing.T) {
	svc := &stubMessageService{}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodPut, "/api/v1/messages/m1/read", "", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "m1", svc.gotID)
	require.Equal(t, "u1", svc.gotUserID)
}

// TestHandler_MarkRead_NotFound 非本人/不存在的消息统一映射 404。
func TestHandler_MarkRead_NotFound(t *testing.T) {
	svc := &stubMessageService{markErr: service.ErrNotFound}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodPut, "/api/v1/messages/m1/read", "", "u2"))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_MarkRead_InternalError(t *testing.T) {
	svc := &stubMessageService{markErr: errBoom}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodPut, "/api/v1/messages/m1/read", "", "u1"))
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestHandler_MarkAllRead_EmptyBodyAllowed 全部已读是默认语义，允许空 body。
func TestHandler_MarkAllRead_EmptyBodyAllowed(t *testing.T) {
	svc := &stubMessageService{markAllN: 4}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodPut, "/api/v1/messages/read-all", "", "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, svc.gotCat)

	var body struct {
		Data struct {
			Updated int64 `json:"updated"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(4), body.Data.Updated)
}

func TestHandler_MarkAllRead_WithCategory(t *testing.T) {
	svc := &stubMessageService{markAllN: 2}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodPut, "/api/v1/messages/read-all", `{"category":"task"}`, "u1"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, model.CategoryTask, svc.gotCat)
}

func TestHandler_MarkAllRead_InvalidCategory(t *testing.T) {
	svc := &stubMessageService{}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodPut, "/api/v1/messages/read-all", `{"category":"nope"}`, "u1"))

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Empty(t, svc.gotCat)
}

func TestHandler_Delete(t *testing.T) {
	svc := &stubMessageService{}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodDelete, "/api/v1/messages/m1", "", "u1"))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "m1", svc.gotID)

	w = httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodDelete, "/api/v1/messages/m2", "", "u1"))
	require.Equal(t, http.StatusOK, w.Code)

	svc.deleteErr = service.ErrNotFound
	w = httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodDelete, "/api/v1/messages/m3", "", "u1"))
	require.Equal(t, http.StatusNotFound, w.Code)

	svc.deleteErr = errBoom
	w = httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodDelete, "/api/v1/messages/m4", "", "u1"))
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestHandler_StaticSegmentNotShadowed /unread-count 与 /read-all 不得被 /{id} 吞掉。
func TestHandler_StaticSegmentNotShadowed(t *testing.T) {
	svc := &stubMessageService{
		unread:   &dto.UnreadCountResp{},
		markAllN: 0,
	}
	h := handler.NewMessageHandler(svc)

	w := httptest.NewRecorder()
	newMessageRouter(h).ServeHTTP(w, newMessageRequest(http.MethodGet, "/api/v1/messages/unread-count", "", "u1"))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "u1", svc.gotUserID)
	require.Empty(t, svc.gotID, "静态段不应命中 MarkRead")
}
