// Package handler 提供认证模块 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/middleware"
	auditmodel "meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/service"
	userdto "meteorx/internal/modules/user/dto"
	userModel "meteorx/internal/modules/user/model"
)

// AuthService 认证服务接口（handler 依赖的最小业务面，便于测试注入桩）
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterUserReq) (*userModel.User, error)
	Login(ctx context.Context, req dto.LoginReq) (*userModel.User, []string, []string, string, error)
	Logout(ctx context.Context, tokenString string) (userID, username, tenantID string, err error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	SendEmailVerification(ctx context.Context, email string, currentUserID string) error
	VerifyEmail(ctx context.Context, token string) error
	Setup2FA(ctx context.Context, userID string) (*dto.Setup2FAResp, error)
	Enable2FA(ctx context.Context, userID, code string) error
	Disable2FA(ctx context.Context, userID, code string) error
}

// AuthHandler 认证处理器
type AuthHandler struct {
	svc AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(svc AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register 用户注册（支持租户自主注册）
// POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterUserReq

	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "用户注册失败: "+err.Error())
		return
	}

	converter := userdto.UserConverter{}
	response.Success(w, converter.ToResponse(user))
}

// Login 用户登录，返回 JWT token、用户信息和权限列表
// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginReq

	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	start := time.Now()

	user, _, permCodes, token, err := h.svc.Login(r.Context(), req)
	if err != nil {
		// 记录登录失败审计日志
		loginReq := middleware.NewAuditLogReq(r)
		loginReq.Module = "auth"
		loginReq.Action = auditmodel.ActionTypeLogin
		loginReq.Resource = r.URL.Path
		loginReq.StatusCode = http.StatusUnauthorized
		loginReq.Result = auditmodel.ResultFailure
		loginReq.ErrorMessage = err.Error()
		loginReq.RiskLevel = auditmodel.RiskHigh
		loginReq.Duration = time.Since(start).Milliseconds()
		middleware.RecordAuditLog(r.Context(), &loginReq)

		if loginErr, ok := err.(*service.LoginError); ok {
			errorResp := dto.LoginErrorResp{
				Message:           loginErr.Message,
				RemainingAttempts: loginErr.RemainingAttempts,
				Locked:            loginErr.Locked,
				LockoutDuration:   loginErr.LockoutDuration,
				Require2FA:        loginErr.Require2FA,
			}
			response.FailWithData(w, http.StatusUnauthorized, loginErr.Message, errorResp)
			return
		}
		response.Fail(w, http.StatusUnauthorized, err.Error())
		return
	}

	// 记录登录成功审计日志
	loginReq := middleware.NewAuditLogReq(r)
	loginReq.UserID = user.ID
	loginReq.Username = user.Username
	loginReq.TenantID = user.TenantID
	loginReq.Module = "auth"
	loginReq.Action = auditmodel.ActionTypeLogin
	loginReq.Resource = r.URL.Path
	loginReq.StatusCode = http.StatusOK
	loginReq.Result = auditmodel.ResultSuccess
	loginReq.RiskLevel = auditmodel.RiskMedium
	loginReq.Duration = time.Since(start).Milliseconds()
	middleware.RecordAuditLog(r.Context(), &loginReq)

	converter := userdto.UserConverter{}
	loginResp := dto.LoginResp{
		Token:       token,
		User:        converter.ToResponse(user),
		Permissions: permCodes,
	}

	response.Success(w, loginResp)
}

// Logout 用户登出，将当前 token 加入黑名单
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权，请先登录")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		response.Fail(w, http.StatusUnauthorized, "无效的 Token 格式")
		return
	}

	tokenString := parts[1]
	start := time.Now()

	userID, username, tenantID, err := h.svc.Logout(r.Context(), tokenString)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "登出失败: "+err.Error())
		return
	}

	// 记录登出审计日志
	logoutReq := middleware.NewAuditLogReq(r)
	logoutReq.UserID = userID
	logoutReq.Username = username
	logoutReq.TenantID = tenantID
	logoutReq.Module = "auth"
	logoutReq.Action = auditmodel.ActionTypeLogout
	logoutReq.Resource = r.URL.Path
	logoutReq.StatusCode = http.StatusOK
	logoutReq.Result = auditmodel.ResultSuccess
	logoutReq.RiskLevel = auditmodel.RiskLow
	logoutReq.Duration = time.Since(start).Milliseconds()
	middleware.RecordAuditLog(r.Context(), &logoutReq)

	response.Success(w, nil)
}

// ForgotPassword 发送密码重置邮件
// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.ForgotPassword(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, service.ErrEmailNotConfigured) {
			response.Fail(w, http.StatusInternalServerError, "邮件服务未配置")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "发送重置邮件失败")
		return
	}

	response.Success(w, dto.ForgotPasswordResp{
		Message: "如果该邮箱已注册，重置链接已发送至您的邮箱",
	})
}

// ResetPassword 通过重置链接设置新密码
// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, service.ErrInvalidResetToken) {
			response.Fail(w, http.StatusBadRequest, "重置链接已失效，请重新请求")
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			response.Fail(w, http.StatusNotFound, "用户不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "重置密码失败")
		return
	}

	response.Success(w, dto.ResetPasswordResp{
		Message: "密码重置成功，请使用新密码登录",
	})
}

// SendEmailVerification 发送邮箱验证链接
// POST /api/v1/auth/email/send-verification
// 需登录，且只能验证当前登录用户自己的邮箱。
func (h *AuthHandler) SendEmailVerification(w http.ResponseWriter, r *http.Request) {
	var req dto.SendEmailVerificationReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	currentUserID := contextx.GetUserID(r.Context())
	if currentUserID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	err := h.svc.SendEmailVerification(r.Context(), req.Email, currentUserID)
	if err != nil {
		if errors.Is(err, service.ErrEmailNotConfigured) {
			response.Fail(w, http.StatusInternalServerError, "邮件服务未配置")
			return
		}
		if errors.Is(err, service.ErrEmailAlreadyVerified) {
			response.Fail(w, http.StatusBadRequest, "邮箱已验证，无需重复验证")
			return
		}
		if errors.Is(err, service.ErrEmailNotOwned) {
			response.Fail(w, http.StatusForbidden, "只能验证自己的邮箱")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "发送验证邮件失败")
		return
	}

	response.Success(w, dto.SendEmailVerificationResp{
		Message: "验证链接已发送至您的邮箱",
	})
}

// VerifyEmail 验证邮箱
// POST /api/v1/auth/email/verify
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyEmailReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.VerifyEmail(r.Context(), req.Token)
	if err != nil {
		if errors.Is(err, service.ErrInvalidResetToken) {
			response.Fail(w, http.StatusBadRequest, "验证链接已失效，请重新请求")
			return
		}
		if errors.Is(err, service.ErrUserNotFound) {
			response.Fail(w, http.StatusNotFound, "用户不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "邮箱验证失败")
		return
	}

	response.Success(w, dto.VerifyEmailResp{
		Message: "邮箱验证成功",
	})
}

// Setup2FA 开启双因素认证第一步：生成密钥与二维码 URI（需登录）
// POST /api/v1/auth/2fa/setup
func (h *AuthHandler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	resp, err := h.svc.Setup2FA(r.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(w, http.StatusNotFound, "用户不存在")
		case errors.Is(err, service.Err2FAAlreadyEnabled):
			response.Fail(w, http.StatusBadRequest, "双因素认证已启用，无需重复开启")
		case errors.Is(err, service.Err2FAStorageUnavailable):
			response.Fail(w, http.StatusInternalServerError, "验证服务不可用，请稍后重试")
		default:
			response.Fail(w, http.StatusInternalServerError, "初始化双因素认证失败")
		}
		return
	}

	response.Success(w, resp)
}

// Enable2FA 确认绑定双因素认证：提交验证器口令以激活（需登录）
// POST /api/v1/auth/2fa/enable
func (h *AuthHandler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req dto.Enable2FAReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.Enable2FA(r.Context(), userID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.Err2FASetupExpired):
			response.Fail(w, http.StatusBadRequest, "绑定已过期，请重新发起开启流程")
		case errors.Is(err, service.Err2FAInvalidCode):
			response.Fail(w, http.StatusBadRequest, "验证码错误，请重试")
		case errors.Is(err, service.Err2FAStorageUnavailable):
			response.Fail(w, http.StatusInternalServerError, "验证服务不可用，请稍后重试")
		default:
			response.Fail(w, http.StatusInternalServerError, "启用双因素认证失败")
		}
		return
	}

	response.Success(w, map[string]string{"message": "双因素认证已启用"})
}

// Disable2FA 关闭双因素认证：需当前有效验证码（需登录）
// POST /api/v1/auth/2fa/disable
func (h *AuthHandler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req dto.Disable2FAReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.Disable2FA(r.Context(), userID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(w, http.StatusNotFound, "用户不存在")
		case errors.Is(err, service.Err2FANotEnabled):
			response.Fail(w, http.StatusBadRequest, "双因素认证未启用")
		case errors.Is(err, service.Err2FAInvalidCode):
			response.Fail(w, http.StatusBadRequest, "验证码错误，请重试")
		default:
			response.Fail(w, http.StatusInternalServerError, "关闭双因素认证失败")
		}
		return
	}

	response.Success(w, map[string]string{"message": "双因素认证已关闭"})
}