package handler

import (
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/service"
	userdto "meteorx/internal/modules/user/dto" // 引用 user 的 DTO 进行转换
	"net/http"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register 处理用户注册请求
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterUserReq

	// 1. 解析并验证请求参数 (使用你之前写的 validator)
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用 Service 执行注册逻辑
	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		// 这里可以根据错误类型返回不同的状态码，目前先统一返回 500
		response.Fail(w, 500, "用户注册失败: "+err.Error())
		return
	}

	// 3. 使用转换器将 Model 转为 Response DTO
	converter := userdto.UserConverter{}

	// 4. 返回成功响应
	response.Success(w, converter.ToResponse(user))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginReq

	// 1. 验证输入
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用登录服务
	user, token, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.Fail(w, 401, err.Error())
		return
	}

	// 3. 组装响应数据
	converter := userdto.UserConverter{}
	loginResp := dto.LoginResp{
		Token: token,
		User:  converter.ToResponse(user),
	}

	response.Success(w, loginResp)
}
