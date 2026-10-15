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
	return Auth(helper, nil, appMode, allowTestBypass, nil)(next)
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
		name  string
		mode  string
		allow bool
		token string
		want  int
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

// authRequestWithPath 以指定路径发起带 Bearer 令牌的请求，返回响应记录器。
func authRequestWithPath(h http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// TestAuthEmailPendingGate 验证未补全邮箱的临时令牌只能访问白名单接口，其余一律 403。
func TestAuthEmailPendingGate(t *testing.T) {
	helper := jwt.NewTokenHelper(config.JWTConfig{
		Secret:     authTestSecret,
		Expiration: "1h",
		Issuer:     "meteorx-test",
	})
	// emailPending=true 的临时令牌
	pendingToken, err := helper.GenerateTokenEx("user-1", "tenant-1", []string{"member"}, true)
	if err != nil {
		t.Fatalf("generate pending token: %v", err)
	}
	// 普通令牌（pending=false）
	normalToken, err := helper.GenerateTokenEx("user-1", "tenant-1", []string{"member"}, false)
	if err != nil {
		t.Fatalf("generate normal token: %v", err)
	}

	h := newAuthTestHandler(t, "release", false, nil)

	tests := []struct {
		name  string
		token string
		path  string
		want  int
	}{
		{"pending 令牌访问业务接口被拒", pendingToken, "/api/v1/probe", http.StatusForbidden},
		{"pending 令牌放行补全邮箱", pendingToken, "/api/v1/auth/oauth/complete-email", http.StatusOK},
		{"pending 令牌放行刷新令牌", pendingToken, "/api/v1/auth/oauth/token/refresh", http.StatusOK},
		{"普通令牌访问业务接口放行", normalToken, "/api/v1/probe", http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := authRequestWithPath(h, tc.path, tc.token)
			if rr.Code != tc.want {
				t.Fatalf("path %s: expected status %d, got %d", tc.path, tc.want, rr.Code)
			}
		})
	}

	// 被拒时携带 X-Error-Code: EMAIL_PENDING 供前端识别跳转
	rr := authRequestWithPath(h, "/api/v1/probe", pendingToken)
	if got := rr.Header().Get("X-Error-Code"); got != "EMAIL_PENDING" {
		t.Fatalf("expected X-Error-Code EMAIL_PENDING, got %q", got)
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
	h := Auth(helper, nil, "debug", false, nil)(next)

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
