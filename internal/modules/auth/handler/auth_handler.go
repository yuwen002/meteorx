package handler

import (
	"net/http"
	"strings"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/service"
	userdto "meteorx/internal/modules/user/dto" // 引用 user 的 DTO 进行转换
)

// AuthHandler 认证处理器
// 处理用户注册、登录、登出等认证相关请求
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler 创建认证处理器实例
// svc: 认证服务实例
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register 处理用户注册请求
// POST /api/v1/auth/register
// 支持租户自主注册新用户
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterUserReq

	// 1. 解析并验证请求参数 (使用 validator 进行参数校验)
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用 Service 执行注册逻辑
	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		// 根据错误类型返回不同的状态码，目前先统一返回 500
		response.Fail(w, 500, "用户注册失败: "+err.Error())
		return
	}

	// 3. 使用转换器将 Model 转为 Response DTO
	converter := userdto.UserConverter{}

	// 4. 返回成功响应
	response.Success(w, converter.ToResponse(user))
}

// Login 处理用户登录请求
// POST /api/v1/auth/login
// 返回 JWT token、用户信息和权限列表
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginReq

	// 1. 验证输入参数
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用登录服务
	user, _, permCodes, token, err := h.svc.Login(r.Context(), req)
	if err != nil {
		// 检查是否为登录错误（包含安全信息如剩余尝试次数）
		if loginErr, ok := err.(*service.LoginError); ok {
			errorResp := dto.LoginErrorResp{
				Message:           loginErr.Message,
				RemainingAttempts: loginErr.RemainingAttempts,
				Locked:            loginErr.Locked,
				LockoutDuration:   loginErr.LockoutDuration,
			}
			response.FailWithData(w, 401, loginErr.Message, errorResp)
			return
		}
		response.Fail(w, 401, err.Error())
		return
	}

	// 3. 组装响应数据（包含用户信息 + 权限码列表）
	converter := userdto.UserConverter{}
	loginResp := dto.LoginResp{
		Token:       token,
		User:        converter.ToResponse(user),
		Permissions: permCodes,
	}

	response.Success(w, loginResp)
}

// Logout 处理用户登出请求
// POST /api/v1/auth/logout
// 将当前 token 加入黑名单，使其失效
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 1. 从 Header 获取 token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		response.Fail(w, 401, "未授权，请先登录")
		return
	}

	// 解析 Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		response.Fail(w, 401, "无效的 Token 格式")
		return
	}

	tokenString := parts[1]

	// 2. 调用登出服务，将 token 加入黑名单
	if err := h.svc.Logout(r.Context(), tokenString); err != nil {
		response.Fail(w, 500, "登出失败: "+err.Error())
		return
	}

	// 3. 返回成功响应
	response.Success(w, nil)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.ForgotPassword(r.Context(), req.Email)
	if err != nil {
		if err.Error() == "email service not configured" {
			response.Fail(w, 500, "邮件服务未配置")
			return
		}
		response.Fail(w, 500, "发送重置邮件失败")
		return
	}

	response.Success(w, dto.ForgotPasswordResp{
		Message: "如果该邮箱已注册，重置链接已发送至您的邮箱",
	})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword)
	if err != nil {
		if err.Error() == "invalid or expired token" {
			response.Fail(w, 400, "重置链接已失效，请重新请求")
			return
		}
		if err.Error() == "user not found" {
			response.Fail(w, 404, "用户不存在")
			return
		}
		response.Fail(w, 500, "重置密码失败")
		return
	}

	response.Success(w, dto.ResetPasswordResp{
		Message: "密码重置成功，请使用新密码登录",
	})
}