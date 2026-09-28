package handler

import (
	"errors"
	"net/http"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/service"
)

type APITokenService interface {
	Create(ctx context.Context, userID, tenantID string, req dto.CreateAPITokenReq) (*dto.CreateAPITokenResp, error)
	ListByUserID(ctx context.Context, userID string) ([]*dto.APITokenResp, error)
	Revoke(ctx context.Context, userID, tokenID string) error
}

type APITokenHandler struct {
	svc APITokenService
}

func NewAPITokenHandler(svc APITokenService) *APITokenHandler {
	return &APITokenHandler{svc: svc}
}

func (h *APITokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	tenantID := contextx.GetTenantID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "未授权")
		return
	}

	var req dto.CreateAPITokenReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	result, err := h.svc.Create(r.Context(), userID, tenantID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAPITokenMaxExceeded):
			response.Fail(w, http.StatusConflict, "已达到 API 令牌数量上限")
		case errors.Is(err, service.ErrAPITokenNameExists):
			response.Fail(w, http.StatusConflict, "同名令牌已存在")
		default:
			response.Fail(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	response.Success(w, result)
}

func (h *APITokenHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "未授权")
		return
	}

	tokens, err := h.svc.ListByUserID(r.Context(), userID)
	if err != nil {
		response.InternalError(w, "获取令牌列表失败")
		return
	}

	response.Success(w, map[string]interface{}{"tokens": tokens})
}

func (h *APITokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "未授权")
		return
	}

	var req dto.RevokeAPITokenReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.Revoke(r.Context(), userID, req.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAPITokenNotFound):
			response.NotFound(w, "令牌不存在")
		case errors.Is(err, service.ErrAPITokenRevoked):
			response.Fail(w, http.StatusBadRequest, "令牌已被撤销")
		default:
			response.InternalError(w, "撤销令牌失败")
		}
		return
	}

	response.Success(w, nil)
}