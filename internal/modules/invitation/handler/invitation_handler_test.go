package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/invitation/dto"
	"meteorx/internal/modules/invitation/handler"
	"meteorx/internal/modules/invitation/model"
	"meteorx/internal/modules/invitation/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type stubInvService struct {
	createFn      func(ctx context.Context, tenantID, invitedBy string, req dto.CreateInvitationReq) (*model.Invitation, error)
	batchCreateFn func(ctx context.Context, tenantID, invitedBy string, req dto.BatchCreateInvitationReq) (*dto.BatchCreateInvitationResp, error)
	listFn        func(ctx context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error)
	cancelFn      func(ctx context.Context, tenantID, invitationID string) error
	resendFn      func(ctx context.Context, tenantID, invitationID string) (*model.Invitation, error)
	deleteFn      func(ctx context.Context, tenantID, invitationID string) error
	acceptFn      func(ctx context.Context, req dto.AcceptInvitationReq) error
	getByTokenFn  func(ctx context.Context, token string) (*model.Invitation, error)
}

func (s *stubInvService) Create(ctx context.Context, tenantID, invitedBy string, req dto.CreateInvitationReq) (*model.Invitation, error) {
	return s.createFn(ctx, tenantID, invitedBy, req)
}
func (s *stubInvService) BatchCreate(ctx context.Context, tenantID, invitedBy string, req dto.BatchCreateInvitationReq) (*dto.BatchCreateInvitationResp, error) {
	return s.batchCreateFn(ctx, tenantID, invitedBy, req)
}
func (s *stubInvService) List(ctx context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error) {
	return s.listFn(ctx, tenantID, page, pageSize, keyword, status)
}
func (s *stubInvService) Cancel(ctx context.Context, tenantID, invitationID string) error {
	return s.cancelFn(ctx, tenantID, invitationID)
}
func (s *stubInvService) Resend(ctx context.Context, tenantID, invitationID string) (*model.Invitation, error) {
	return s.resendFn(ctx, tenantID, invitationID)
}
func (s *stubInvService) Delete(ctx context.Context, tenantID, invitationID string) error {
	return s.deleteFn(ctx, tenantID, invitationID)
}
func (s *stubInvService) Accept(ctx context.Context, req dto.AcceptInvitationReq) error {
	return s.acceptFn(ctx, req)
}
func (s *stubInvService) GetByToken(ctx context.Context, token string) (*model.Invitation, error) {
	return s.getByTokenFn(ctx, token)
}
func (s *stubInvService) ListAssignableRoles(_ context.Context) ([]dto.RoleOption, error) {
	return []dto.RoleOption{{ID: "role-1", Name: "租户管理员", Code: "tenant_admin"}}, nil
}

func newInvRouter(stub *stubInvService) http.Handler {
	h := handler.NewInvitationHandler(stub)
	r := chi.NewRouter()
	r.Route("/invitations", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Post("/batch", h.BatchCreate)
		r.Get("/", h.List)
		r.Put("/{id}/cancel", h.Cancel)
		r.Put("/{id}/resend", h.Resend)
		r.Delete("/{id}/delete", h.Delete)
		r.Post("/accept", h.Accept)
		r.Get("/info", h.GetByToken)
	})
	return r
}

func withTenantContext(r *http.Request, tenantID, userID string) *http.Request {
	ctx := contextx.SetVars(r.Context(), tenantID, userID, nil)
	return r.WithContext(ctx)
}

func TestCreate_Success(t *testing.T) {
	now := time.Now()
	stub := &stubInvService{
		createFn: func(_ context.Context, _, _ string, _ dto.CreateInvitationReq) (*model.Invitation, error) {
			return &model.Invitation{
				ID: "inv-1", TenantID: "t-1", Email: "test@example.com",
				Status: model.InvitationStatusPending, ExpiresAt: now.Add(7 * 24 * time.Hour),
				CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.CreateInvitationReq{
		Email:   "test@example.com",
		RoleIDs: []string{"role-1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCreate_DuplicateEmail(t *testing.T) {
	stub := &stubInvService{
		createFn: func(_ context.Context, _, _ string, _ dto.CreateInvitationReq) (*model.Invitation, error) {
			return nil, service.ErrEmailAlreadyInvited
		},
	}

	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.CreateInvitationReq{
		Email:   "dup@example.com",
		RoleIDs: []string{"role-1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreate_Unauthorized(t *testing.T) {
	stub := &stubInvService{
		createFn: func(_ context.Context, _, _ string, _ dto.CreateInvitationReq) (*model.Invitation, error) {
			return nil, nil
		},
	}

	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.CreateInvitationReq{
		Email:   "test@example.com",
		RoleIDs: []string{"role-1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestBatchCreate_Success(t *testing.T) {
	stub := &stubInvService{
		batchCreateFn: func(_ context.Context, _, _ string, req dto.BatchCreateInvitationReq) (*dto.BatchCreateInvitationResp, error) {
			return &dto.BatchCreateInvitationResp{
				Created: len(req.Emails),
				Skipped: []dto.SkippedEmail{{Email: "dup@example.com", Reason: "邮箱重复"}},
			}, nil
		},
	}

	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.BatchCreateInvitationReq{
		Emails:  []string{"a@example.com", "b@example.com"},
		RoleIDs: []string{"role-1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Created int                `json:"created"`
			Skipped []dto.SkippedEmail `json:"skipped"`
		} `json:"data"`
	}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Data.Created)
	assert.Len(t, resp.Data.Skipped, 1)
}

func TestBatchCreate_Unauthorized(t *testing.T) {
	stub := &stubInvService{
		batchCreateFn: func(_ context.Context, _, _ string, _ dto.BatchCreateInvitationReq) (*dto.BatchCreateInvitationResp, error) {
			return &dto.BatchCreateInvitationResp{}, nil
		},
	}
	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.BatchCreateInvitationReq{Emails: []string{"a@example.com"}, RoleIDs: []string{"role-1"}})
	req := httptest.NewRequest(http.MethodPost, "/invitations/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// 未设置租户上下文
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestList_Success(t *testing.T) {
	now := time.Now()
	stub := &stubInvService{
		listFn: func(_ context.Context, _ string, _, _ int, _, _ string) ([]*model.Invitation, int64, error) {
			return []*model.Invitation{
				{ID: "inv-1", TenantID: "t-1", Email: "a@example.com", Status: model.InvitationStatusPending, ExpiresAt: now, CreatedAt: now, UpdatedAt: now},
			}, 1, nil
		},
	}

	router := newInvRouter(stub)
	req := httptest.NewRequest(http.MethodGet, "/invitations?page=1&page_size=10", nil)
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCancel_Success(t *testing.T) {
	stub := &stubInvService{
		cancelFn: func(_ context.Context, _, _ string) error { return nil },
	}

	router := newInvRouter(stub)
	req := httptest.NewRequest(http.MethodPut, "/invitations/inv-1/cancel", nil)
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCancel_NotFound(t *testing.T) {
	stub := &stubInvService{
		cancelFn: func(_ context.Context, _, _ string) error { return service.ErrInvitationNotFound },
	}

	router := newInvRouter(stub)
	req := httptest.NewRequest(http.MethodPut, "/invitations/nonexistent/cancel", nil)
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDelete_Success(t *testing.T) {
	stub := &stubInvService{
		deleteFn: func(_ context.Context, _, _ string) error { return nil },
	}

	router := newInvRouter(stub)
	req := httptest.NewRequest(http.MethodDelete, "/invitations/inv-1/delete", nil)
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDelete_NotFound(t *testing.T) {
	stub := &stubInvService{
		deleteFn: func(_ context.Context, _, _ string) error { return service.ErrInvitationNotFound },
	}

	router := newInvRouter(stub)
	req := httptest.NewRequest(http.MethodDelete, "/invitations/nonexistent/delete", nil)
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAccept_Success(t *testing.T) {
	stub := &stubInvService{
		acceptFn: func(_ context.Context, _ dto.AcceptInvitationReq) error { return nil },
	}

	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.AcceptInvitationReq{
		Token:    "token-123",
		Username: "newuser",
		Password: "password123",
		Nickname: "New User",
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations/accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAccept_Expired(t *testing.T) {
	stub := &stubInvService{
		acceptFn: func(_ context.Context, _ dto.AcceptInvitationReq) error { return service.ErrInvitationExpired },
	}

	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.AcceptInvitationReq{
		Token:    "token-exp",
		Username: "newuser",
		Password: "password123",
		Nickname: "New User",
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations/accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccept_NotFound(t *testing.T) {
	stub := &stubInvService{
		acceptFn: func(_ context.Context, _ dto.AcceptInvitationReq) error {
			return service.ErrInvitationNotFound
		},
	}

	router := newInvRouter(stub)
	body, _ := json.Marshal(dto.AcceptInvitationReq{
		Token:    "invalid",
		Username: "user",
		Password: "pass123",
		Nickname: "Nick",
	})
	req := httptest.NewRequest(http.MethodPost, "/invitations/accept", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetByToken_Success(t *testing.T) {
	now := time.Now()
	stub := &stubInvService{
		getByTokenFn: func(_ context.Context, _ string) (*model.Invitation, error) {
			return &model.Invitation{
				ID: "inv-1", TenantID: "t-1", Email: "info@example.com",
				Status: model.InvitationStatusPending, ExpiresAt: now, CreatedAt: now, UpdatedAt: now,
			}, nil
		},
	}

	router := newInvRouter(stub)
	req := httptest.NewRequest(http.MethodGet, "/invitations/info?token=token-123", nil)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetByToken_MissingToken(t *testing.T) {
	stub := &stubInvService{}
	router := newInvRouter(stub)

	req := httptest.NewRequest(http.MethodGet, "/invitations/info", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestResend_EmailNotConfigured(t *testing.T) {
	stub := &stubInvService{
		resendFn: func(_ context.Context, _, _ string) (*model.Invitation, error) {
			return nil, service.ErrEmailNotConfigured
		},
	}

	router := newInvRouter(stub)
	req := httptest.NewRequest(http.MethodPut, "/invitations/inv-1/resend", nil)
	req = withTenantContext(req, "t-1", "u-1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

var _ service.InvitationServiceInterface = (*stubInvService)(nil)
