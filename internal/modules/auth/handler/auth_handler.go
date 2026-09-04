// Package handler 提供认证模块 HTTP 处理器
package handler

import (
	"errors"
	"net/http"
	"strings"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/service"
	userdto "meteorx/internal/modules/user/dto"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
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

	user, _, permCodes, token, err := h.svc.Login(r.Context(), req)
	if err != nil {
		if loginErr, ok := err.(*service.LoginError); ok {
			errorResp := dto.LoginErrorResp{
				Message:           loginErr.Message,
				RemainingAttempts: loginErr.RemainingAttempts,
				Locked:            loginErr.Locked,
				LockoutDuration:   loginErr.LockoutDuration,
			}
			response.FailWithData(w, http.StatusUnauthorized, loginErr.Message, errorResp)
			return
		}
		response.Fail(w, http.StatusUnauthorized, err.Error())
		return
	}

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
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		response.Fail(w, http.StatusUnauthorized, "无效的 Token 格式")
		return
	}

	tokenString := parts[1]

	if err := h.svc.Logout(r.Context(), tokenString); err != nil {
		response.Fail(w, http.StatusInternalServerError, "登出失败: "+err.Error())
		return
	}

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
