package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
)

const authTestSecret = "auth-test-secret-0123456789abcdef"

const bypassToken = "123456789"

func newAuthTestHandler(t *testing.T, appMode string, allowTestBypass bool, capture *string) http.Handler {
	t.Helper()
	helper := jwt.NewTokenHelper(config.JWTConfig{
		Secret:     authTestSecret,
		Expiration: "1h",
		Issuer:     "meteorx-test",
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			*capture, _ = r.Context().Value(contextx.UserIDKey).(string)
		}
		w.WriteHeader(http.StatusOK)
	})
	return Auth(helper, nil, appMode, allowTestBypass)(next)
}

func authTestRequest(h http.Handler, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/probe", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func generateValidToken(t *testing.T) string {
	t.Helper()
	helper := jwt.NewTokenHelper(config.JWTConfig{
		Secret:     authTestSecret,
		Expiration: "1h",
		Issuer:     "meteorx-test",
	})
	token, err := helper.GenerateToken("user-1", "tenant-1", []string{"member"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

// TestAuthTestBypassGate 验证调试后门 Token 只有在「配置显式开启 + 非 release 模式」才放行
func TestAuthTestBypassGate(t *testing.T) {
	valid := generateValidToken(t)

	tests := []struct {
		name    string
		mode    string
		allow   bool
		token   string
		want    int
	}{
		{"缺失 Token 一律拒绝", "debug", true, "", http.StatusUnauthorized},
		{"默认关闭：debug 下后门 Token 拒绝", "debug", false, bypassToken, http.StatusUnauthorized},
		{"release 模式即使开启也拒绝", "release", true, bypassToken, http.StatusUnauthorized},
		{"显式开启 + debug：后门 Token 放行", "debug", true, bypassToken, http.StatusOK},
		{"合法 JWT 正常放行", "debug", false, valid, http.StatusOK},
		{"非法 JWT 拒绝", "debug", true, "abc.def.ghi", http.StatusUnauthorized},
		{"release 下合法 JWT 正常放行", "release", false, valid, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newAuthTestHandler(t, tc.mode, tc.allow, nil)
			rr := authTestRequest(h, tc.token)
			if rr.Code != tc.want {
				t.Fatalf("expected status %d, got %d", tc.want, rr.Code)
			}
		})
	}
}

// TestAuthTestBypassIdentity 放行时注入固定的超级管理员身份
func TestAuthTestBypassIdentity(t *testing.T) {
	var captured string
	h := newAuthTestHandler(t, "debug", true, &captured)
	rr := authTestRequest(h, bypassToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if captured != "admin-id-001" {
		t.Fatalf("expected bypass user id admin-id-001, got %q", captured)
	}
}

// TestAuthInjectsContext 正常 JWT 校验后注入用户/租户上下文
func TestAuthInjectsContext(t *testing.T) {
	var userID, tenantID string
	helper := jwt.NewTokenHelper(config.JWTConfig{
		Secret:     authTestSecret,
		Expiration: "1h",
		Issuer:     "meteorx-test",
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, _ = ctx.Value(contextx.UserIDKey).(string)
		tenantID, _ = ctx.Value(contextx.TenantIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})
	h := Auth(helper, nil, "debug", false)(next)

	token, err := helper.GenerateToken("user-7", "tenant-9", []string{"editor"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	rr := authTestRequest(h, token)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if userID != "user-7" || tenantID != "tenant-9" {
		t.Fatalf("context injection mismatch: user=%q tenant=%q", userID, tenantID)
	}
}
