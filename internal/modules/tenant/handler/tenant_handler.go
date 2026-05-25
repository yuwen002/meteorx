package handler

import (
	"errors"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/service"
	"net/http"
)

type TenantHandler struct {
	svc *service.TenantService
}

func NewTenantHandler(svc *service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

// Register POST /api/v1/tenants/register
func (h *TenantHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterTenantReq

	// 1. 解析并验证请求
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 逻辑调用
	tenant, err := h.svc.Register(r.Context(), req)
	if err != nil {
		// 分门别类处理错误
		switch {
		case errors.Is(err, service.ErrDomainConflict):
			response.Fail(w, 409, "该域名已被其他租户占用")
		default:
			// 记录日志并在响应中隐藏细节
			// log.Printf("Register Error: %v", err)
			response.Fail(w, 500, "服务器开小差了，请稍后再试")
		}
		return
	}

	// 3. 返回成功
	converter := dto.TenantConverter{}
	response.Success(w, converter.ToTenantResponse(tenant))
}

// AdminCreate POST /api/v1/admin/tenants
// 运营后台管理员手动创建租户接口，需要超级管理员权限
func (h *TenantHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.AdminCreateTenantReq

	// 1. 利用你现有的 validator 组件解析并验证 JSON 传参
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用服务层逻辑
	tenant, err := h.svc.AdminCreate(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDomainConflict):
			response.Fail(w, http.StatusConflict, "该租户域名已被占用")
		default:
			// 可在此处调用你的日志组件记录底层错误细节，如：log.Printf("%v", err)
			response.Fail(w, http.StatusInternalServerError, "创建租户失败，服务器内部错误")
		}
		return
	}

	// 3. 组装返回给后台的视图数据
	respData := dto.AdminTenantResp{
		ID:           tenant.ID,
		Name:         tenant.Name,
		Domain:       tenant.Domain,
		Status:       tenant.Status,
		Description:  tenant.Description,
		ContactEmail: tenant.ContactEmail,
		Region:       tenant.Region,
		Logo:         tenant.Logo,
		Extra:        tenant.Extra,
		CreatedAt:    tenant.CreatedAt,
		UpdatedAt:    tenant.UpdatedAt,
	}

	// 4. 返回成功响应
	response.Success(w, respData)
}

// UpdateStatus POST/PUT /api/v1/admin/tenants/:id/status
func (h *TenantHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL 或 Body 拿到 ID 和 Status
	// 2. 调用 h.svc.UpdateTenantStatus(ctx, id, req.Status)
	// 3. response.Success(w, nil)
}

// List GET /api/v1/admin/tenants?page=1&page_size=10&name=极客
func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	// 从 URL query 拿到 page, page_size, name
	// tenants, total, err := h.svc.QueryTenantList(...)
	// 转换成 DTO 后，response.Success(w, map[string]interface{}{"list": list, "total": total})
}
