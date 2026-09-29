package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/invitation/dto"
	"meteorx/internal/modules/invitation/model"
	"meteorx/internal/modules/invitation/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

type InvitationHandler struct {
	svc *service.InvitationService
}

func NewInvitationHandler(svc *service.InvitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
}

func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req dto.CreateInvitationReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	inv, err := h.svc.Create(r.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyInvited) {
			response.Fail(w, http.StatusConflict, "该邮箱已有待处理的邀请")
			return
		}
		if errors.Is(err, service.ErrEmailAlreadyMember) {
			response.Fail(w, http.StatusConflict, "该邮箱已是本租户成员")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "创建邀请失败: "+err.Error())
		return
	}

	response.Success(w, toResp(inv))
}

func (h *InvitationHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	keyword := r.URL.Query().Get("keyword")
	status := r.URL.Query().Get("status")

	invs, total, err := h.svc.List(r.Context(), tenantID, pg.Page, pg.PageSize, keyword, status)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取邀请列表失败")
		return
	}

	var items []*dto.InvitationResp
	for _, inv := range invs {
		items = append(items, toResp(inv))
	}
	result := pagination.NewPaginatedResult(items, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

func (h *InvitationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	invitationID := chi.URLParam(r, "id")
	if tenantID == "" || invitationID == "" {
		response.Fail(w, http.StatusBadRequest, "参数不完整")
		return
	}

	if err := h.svc.Cancel(r.Context(), tenantID, invitationID); err != nil {
		if errors.Is(err, service.ErrInvitationNotFound) {
			response.Fail(w, http.StatusNotFound, "邀请不存在")
			return
		}
		if errors.Is(err, service.ErrInvitationAlreadyUsed) {
			response.Fail(w, http.StatusBadRequest, "邀请已处理，无法取消")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "取消邀请失败")
		return
	}

	response.Success(w, nil)
}

func (h *InvitationHandler) Resend(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	invitationID := chi.URLParam(r, "id")
	if tenantID == "" || invitationID == "" {
		response.Fail(w, http.StatusBadRequest, "参数不完整")
		return
	}

	inv, err := h.svc.Resend(r.Context(), tenantID, invitationID)
	if err != nil {
		if errors.Is(err, service.ErrInvitationNotFound) {
			response.Fail(w, http.StatusNotFound, "邀请不存在")
			return
		}
		if errors.Is(err, service.ErrEmailNotConfigured) {
			response.Fail(w, http.StatusInternalServerError, "邮件服务未配置")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "重发邀请失败")
		return
	}

	response.Success(w, toResp(inv))
}

func (h *InvitationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	invitationID := chi.URLParam(r, "id")
	if tenantID == "" || invitationID == "" {
		response.Fail(w, http.StatusBadRequest, "参数不完整")
		return
	}

	if err := h.svc.Delete(r.Context(), tenantID, invitationID); err != nil {
		if errors.Is(err, service.ErrInvitationNotFound) {
			response.Fail(w, http.StatusNotFound, "邀请不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "删除邀请失败")
		return
	}

	response.Success(w, nil)
}

func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	var req dto.AcceptInvitationReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.Accept(r.Context(), req); err != nil {
		if errors.Is(err, service.ErrInvitationNotFound) {
			response.Fail(w, http.StatusNotFound, "邀请不存在")
			return
		}
		if errors.Is(err, service.ErrInvitationExpired) {
			response.Fail(w, http.StatusBadRequest, "邀请已过期")
			return
		}
		if errors.Is(err, service.ErrInvitationAlreadyUsed) {
			response.Fail(w, http.StatusBadRequest, "邀请已被使用")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "接受邀请失败: "+err.Error())
		return
	}

	response.Success(w, dto.AcceptInvitationResp{
		Message: "邀请接受成功，请使用新账号登录",
	})
}

func (h *InvitationHandler) GetByToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		response.Fail(w, http.StatusBadRequest, "缺少邀请令牌")
		return
	}

	inv, err := h.svc.GetByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrInvitationNotFound) {
			response.Fail(w, http.StatusNotFound, "邀请不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "查询邀请失败")
		return
	}

	response.Success(w, toResp(inv))
}

func toResp(inv *model.Invitation) *dto.InvitationResp {
	if inv == nil {
		return nil
	}
	var roleIDs []string
	_ = json.Unmarshal([]byte(inv.RoleIDs), &roleIDs)

	resp := &dto.InvitationResp{
		ID:        inv.ID,
		TenantID:  inv.TenantID,
		Email:     inv.Email,
		RoleIDs:   roleIDs,
		Status:    inv.Status,
		InvitedBy: inv.InvitedBy,
		ExpiresAt: inv.ExpiresAt.Format("2006-01-02 15:04:05"),
		CreatedAt: inv.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if inv.AcceptedAt != nil {
		resp.AcceptedAt = inv.AcceptedAt.Format("2006-01-02 15:04:05")
	}
	return resp
}