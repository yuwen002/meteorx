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
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/auth/service"
	userModel "meteorx/internal/modules/user/model"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAuthService 桩实现：记录入参并按预设返回
type stubAuthService struct {
	registerReq  dto.RegisterUserReq
	registerUser *userModel.User
	registerErr  error

	loginReq  dto.LoginReq
	loginUser *userModel.User
	permCodes []string
	token     string
	loginErr  error

	logoutToken string
	logoutErr   error

	forgotEmail string
	forgotErr   error

	resetToken   string
	resetNewPass string
	resetErr     error

	sendEmailVerifyErr error
	verifyEmailErr    error

	setup2FAResp *dto.Setup2FAResp
	setup2FAErr  error
	enable2FAErr error
	disable2FAErr error
}

func (s *stubAuthService) Register(_ context.Context, req dto.RegisterUserReq) (*userModel.User, error) {
	s.registerReq = req
	return s.registerUser, s.registerErr
}

func (s *stubAuthService) Login(_ context.Context, req dto.LoginReq) (*userModel.User, []string, []string, string, error) {
	s.loginReq = req
	return s.loginUser, nil, s.permCodes, s.token, s.loginErr
}

func (s *stubAuthService) Logout(_ context.Context, tokenString string) (string, string, string, error) {
	s.logoutToken = tokenString
	return "user-id", "admin", "tenant-id", s.logoutErr
}

func (s *stubAuthService) ForgotPassword(_ context.Context, email string) error {
	s.forgotEmail = email
	return s.forgotErr
}

func (s *stubAuthService) ResetPassword(_ context.Context, token, newPassword string) error {
	s.resetToken = token
	s.resetNewPass = newPassword
	return s.resetErr
}

func (s *stubAuthService) SendEmailVerification(_ context.Context, email string, _ string) error {
	return s.sendEmailVerifyErr
}

func (s *stubAuthService) VerifyEmail(_ context.Context, token string) error {
	return s.verifyEmailErr
}

func (s *stubAuthService) Setup2FA(_ context.Context, _ string) (*dto.Setup2FAResp, error) {
	return s.setup2FAResp, s.setup2FAErr
}

func (s *stubAuthService) Enable2FA(_ context.Context, _, _ string) error {
	return s.enable2FAErr
}

func (s *stubAuthService) Disable2FA(_ context.Context, _, _ string) error {
	return s.disable2FAErr
}

func newAuthRouter(stub *stubAuthService) http.Handler {
	h := handler.NewAuthHandler(stub)
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	return r
}

func doAuth(t *testing.T, router http.Handler, path, body string, header func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(http.MethodPost, path, nil)
	} else {
		req = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	}
	if header != nil {
		header(req)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func sampleUser(id, username string) *userModel.User {
	return &userModel.User{ID: id, Username: username, Nickname: "昵称", Email: "u@x.com", Status: 1}
}

func TestRegister_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubAuthService{}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/register", `{"username":"短","email":"bad"}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "用户名")
}

func TestRegister_Success(t *testing.T) {
	stub := &stubAuthService{registerUser: sampleUser("u-1", "alice")}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/register", `{"tenant_id":"t-1","username":"alice123","password":"pass1234","nickname":"爱丽丝","email":"a@x.com"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"username":"alice"`)
	assert.Equal(t, "alice123", stub.registerReq.Username)
}

func TestRegister_ServiceError_Returns500(t *testing.T) {
	stub := &stubAuthService{registerErr: errors.New("user exists")}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/register", `{"tenant_id":"t","username":"alice123","password":"pass1234","nickname":"爱丽丝","email":"a@x.com"}`, nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "用户注册失败")
}

func TestLogin_Success_ReturnsTokenAndPermissions(t *testing.T) {
	stub := &stubAuthService{loginUser: sampleUser("u-1", "alice"), token: "jwt-token", permCodes: []string{"wiki:create", "wiki:read"}}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/login", `{"tenant_id":"t","username":"alice","password":"pass1234"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token":"jwt-token"`)
	assert.Contains(t, w.Body.String(), `"wiki:create"`)
}

func TestLogin_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubAuthService{}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/login", `{"username":"","password":""}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "密码")
}

func TestLogin_LockoutError_Returns401WithRemaining(t *testing.T) {
	stub := &stubAuthService{loginErr: &service.LoginError{Message: "失败次数过多", RemainingAttempts: 2, Locked: true, LockoutDuration: 300}}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/login", `{"username":"alice","password":"wrong"}`, nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "失败次数过多")
	assert.Contains(t, w.Body.String(), `"remaining_attempts":2`)
	assert.Contains(t, w.Body.String(), `"locked":true`)
}

func TestLogin_GenericError_Returns401(t *testing.T) {
	stub := &stubAuthService{loginErr: errors.New("用户名或密码错误")}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/login", `{"username":"alice","password":"x"}`, nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "用户名或密码错误")
}

func TestLogout_NoAuthorization_Returns401(t *testing.T) {
	stub := &stubAuthService{}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/logout", "", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未授权，请先登录")
	assert.Empty(t, stub.logoutToken)
}

func TestLogout_MalformedHeader_Returns401(t *testing.T) {
	stub := &stubAuthService{}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/logout", "", func(r *http.Request) { r.Header.Set("Authorization", "Basic abc") })

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "无效的 Token 格式")
	assert.Empty(t, stub.logoutToken)
}

func TestLogout_Success_BlacklistsToken(t *testing.T) {
	stub := &stubAuthService{}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/logout", "", func(r *http.Request) { r.Header.Set("Authorization", "Bearer token-123") })

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "token-123", stub.logoutToken)
}

func TestLogout_ServiceError_Returns500(t *testing.T) {
	stub := &stubAuthService{logoutErr: errors.New("redis down")}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/logout", "", func(r *http.Request) { r.Header.Set("Authorization", "Bearer t") })

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "登出失败")
}

func TestForgotPassword_Success(t *testing.T) {
	stub := &stubAuthService{}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/forgot-password", `{"email":"a@x.com"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "a@x.com", stub.forgotEmail)
	assert.Contains(t, w.Body.String(), "重置链接已发送")
}

func TestForgotPassword_EmailNotConfigured_Returns500(t *testing.T) {
	stub := &stubAuthService{forgotErr: service.ErrEmailNotConfigured}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/forgot-password", `{"email":"a@x.com"}`, nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "邮件服务未配置")
}

func TestResetPassword_InvalidToken_Returns400(t *testing.T) {
	stub := &stubAuthService{resetErr: service.ErrInvalidResetToken}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/reset-password", `{"token":"expired","new_password":"newpass123"}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "重置链接已失效")
}

func TestResetPassword_UserNotFound_Returns404(t *testing.T) {
	stub := &stubAuthService{resetErr: service.ErrUserNotFound}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/reset-password", `{"token":"t","new_password":"newpass123"}`, nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "用户不存在")
}

func TestResetPassword_Success(t *testing.T) {
	stub := &stubAuthService{}
	router := newAuthRouter(stub)

	w := doAuth(t, router, "/reset-password", `{"token":"rt-1","new_password":"newpass123"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rt-1", stub.resetToken)
	assert.Equal(t, "newpass123", stub.resetNewPass)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
}

// ---------- 双因素认证 (2FA) handler ----------

// newTwoFactorRouter 构建带用户上下文注入的 2FA 路由；通过 X-Test-User 头控制当前用户ID（为空则未登录）。
func newTwoFactorRouter(stub *stubAuthService) http.Handler {
	h := handler.NewAuthHandler(stub)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if uid := req.Header.Get("X-Test-User"); uid != "" {
				ctx := contextx.SetVars(req.Context(), "t-1", uid, nil)
				req = req.WithContext(ctx)
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Post("/2fa/setup", h.Setup2FA)
	r.Post("/2fa/enable", h.Enable2FA)
	r.Post("/2fa/disable", h.Disable2FA)
	return r
}

func withUser(uid string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("X-Test-User", uid) }
}

func TestSetup2FA_Unauthorized_Returns401(t *testing.T) {
	router := newTwoFactorRouter(&stubAuthService{})
	w := doAuth(t, router, "/2fa/setup", "", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未获取到用户信息")
}

func TestSetup2FA_Success(t *testing.T) {
	stub := &stubAuthService{setup2FAResp: &dto.Setup2FAResp{Secret: "SECRET123", OTPAuthURL: "otpauth://totp/MeteorX:a@x.com?secret=SECRET123"}}
	router := newTwoFactorRouter(stub)
	w := doAuth(t, router, "/2fa/setup", "", withUser("u-1"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SECRET123")
}

func TestSetup2FA_AlreadyEnabled_Returns400(t *testing.T) {
	stub := &stubAuthService{setup2FAErr: service.Err2FAAlreadyEnabled}
	router := newTwoFactorRouter(stub)
	w := doAuth(t, router, "/2fa/setup", "", withUser("u-1"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "已启用")
}

func TestEnable2FA_ValidationFailure_Returns400(t *testing.T) {
	router := newTwoFactorRouter(&stubAuthService{})
	w := doAuth(t, router, "/2fa/enable", `{"code":""}`, withUser("u-1"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEnable2FA_InvalidCode_Returns400(t *testing.T) {
	stub := &stubAuthService{enable2FAErr: service.Err2FAInvalidCode}
	router := newTwoFactorRouter(stub)
	w := doAuth(t, router, "/2fa/enable", `{"code":"123456"}`, withUser("u-1"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "验证码错误")
}

func TestEnable2FA_Success(t *testing.T) {
	router := newTwoFactorRouter(&stubAuthService{})
	w := doAuth(t, router, "/2fa/enable", `{"code":"123456"}`, withUser("u-1"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "已启用")
}

func TestDisable2FA_NotEnabled_Returns400(t *testing.T) {
	stub := &stubAuthService{disable2FAErr: service.Err2FANotEnabled}
	router := newTwoFactorRouter(stub)
	w := doAuth(t, router, "/2fa/disable", `{"code":"123456"}`, withUser("u-1"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "未启用")
}

func TestDisable2FA_Success(t *testing.T) {
	router := newTwoFactorRouter(&stubAuthService{})
	w := doAuth(t, router, "/2fa/disable", `{"code":"123456"}`, withUser("u-1"))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "已关闭")
}