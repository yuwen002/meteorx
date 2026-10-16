package invitation

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/invitation/handler"
	"meteorx/internal/modules/invitation/model"
	"meteorx/internal/modules/invitation/repository"
	"meteorx/internal/modules/invitation/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userRepo "meteorx/internal/modules/user/repository"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newInviteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))
	require.NoError(t, userRepo.AutoMigrate(db))
	require.NoError(t, tenantRepo.AutoMigrate(db))
	require.NoError(t, rbacRepo.AutoMigrate(db))
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// newInviteApp 用真实仓储装配邀请服务，并挂载公开接受/查询路由（无需登录）。
func newInviteApp(t *testing.T, db *gorm.DB) *chi.Mux {
	t.Helper()
	invRepo := repository.NewInvitationRepository(db)
	uRepo := userRepo.NewUserRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)

	svc := service.NewInvitationService(
		invRepo, uRepo, tRepo, rRepo, urRepo,
		config.EmailConfig{},
		config.ClientConfig{BaseURL: "http://localhost:3000"},
		config.SecurityConfig{PasswordPolicy: config.PasswordPolicyConfig{MinLength: 6}},
	)
	h := handler.NewInvitationHandler(svc)

	r := chi.NewRouter()
	RegisterPublicRoutes(r, h) // POST /invitations/accept, GET /invitations/info
	return r
}

func insertInvitation(t *testing.T, db *gorm.DB, token, email, tenantID string, expires time.Time) {
	t.Helper()
	inv := &model.Invitation{
		ID:        "inv-" + token,
		TenantID:  tenantID,
		Email:     email,
		Token:     token,
		RoleIDs:   "[]",
		Status:    model.InvitationStatusPending,
		InvitedBy: "inviter-1",
		ExpiresAt: expires,
	}
	require.NoError(t, repository.NewInvitationRepository(db).Create(context.Background(), inv))
}

func doJSON(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestInteg_InvitationAccept(t *testing.T) {
	db := newInviteDB(t)
	r := newInviteApp(t, db)

	const (
		token    = "tok-valid-123"
		email    = "invitee@example.com"
		tenantID = "t-1"
	)

	// 1) 合法接受：成功创建账户并标记 accepted
	insertInvitation(t, db, token, email, tenantID, time.Now().Add(24*time.Hour))
	w := doJSON(t, r, http.MethodPost, "/invitations/accept", `{
		"token":"tok-valid-123","username":"invitee1","password":"secret123","nickname":"Invitee"
	}`)
	require.Equal(t, http.StatusOK, w.Code, "accept: %s", w.Body.String())

	// 用户已落库且邮箱已验证
	uRepo := userRepo.NewUserRepository(db)
	user, err := uRepo.GetByEmail(context.Background(), email)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, user.EmailVerified, "通过邀请注册应视为邮箱已验证")
	assert.Equal(t, tenantID, user.TenantID)

	// 邀请状态已置为 accepted
	inv, err := repository.NewInvitationRepository(db).GetByToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, model.InvitationStatusAccepted, inv.Status)

	// 2) 重放同一令牌：应拒绝（已使用）
	w = doJSON(t, r, http.MethodPost, "/invitations/accept", `{
		"token":"tok-valid-123","username":"invitee2","password":"secret123","nickname":"X"
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "replay should be rejected")

	// 3) 查询接口：已接受的邀请仍可查到且状态正确
	w = doJSON(t, r, http.MethodGet, "/invitations/info?token=tok-valid-123", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var info map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &info))
	data := info["data"].(map[string]any)
	assert.Equal(t, model.InvitationStatusAccepted, data["status"])

	// 4) 过期邀请：接受应被拒
	insertInvitation(t, db, "tok-expired", "exp@example.com", tenantID, time.Now().Add(-time.Hour))
	w = doJSON(t, r, http.MethodPost, "/invitations/accept", `{
		"token":"tok-expired","username":"exp1","password":"secret123","nickname":"E"
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "expired should be rejected")

	// 5) 不存在的令牌：404
	w = doJSON(t, r, http.MethodPost, "/invitations/accept", `{
		"token":"tok-nope","username":"unknown1","password":"secret123","nickname":"X"
	}`)
	assert.Equal(t, http.StatusNotFound, w.Code, "unknown token should 404")

	// 6) 弱密码（校验前置）：400
	w = doJSON(t, r, http.MethodPost, "/invitations/accept", `{
		"token":"tok-weak","username":"weak1","password":"123","nickname":"W"
	}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "weak password should be rejected by validator")
}
