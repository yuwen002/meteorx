package handler

import (
	"encoding/json"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/service"
	"net/http"
)

// RegisterRequest 注册租户的请求载体
type RegisterRequest struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type TenantHandler struct {
	svc *service.TenantService
}

func NewTenantHandler(svc *service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterTenantReq

	// 1. 解析
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	// 2. 逻辑调用
	tenant, err := h.svc.Register(r.Context(), req.Name, req.Domain)
	if err != nil {
		// 甚至可以根据 err 类型判断返回什么错误
		response.Fail(w, 10001, err.Error())
		return
	}

	// 3. 返回成功
	response.Success(w, NewTenantResp(tenant))
}
