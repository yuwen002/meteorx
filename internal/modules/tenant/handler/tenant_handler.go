package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/service"
	"meteorx/pkg/pagination"
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
		case errors.Is(err, service.ErrUsernameConflict):
			response.Fail(w, 409, "该用户名已被使用")
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

	// 1. 用现有的 validator 组件解析并验证 JSON 传参
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用服务层逻辑
	tenant, err := h.svc.AdminCreate(r.Context(), req)
	fmt.Println(tenant)
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

// AdminUpdateStatus PUT /api/v1/admin/tenants/:id/status
func (h *TenantHandler) AdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL 获取 ID
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	// 2. 从请求体获取 Status
	var req dto.AdminUpdateTenantStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 3. 调用服务层更新状态
	err := h.svc.UpdateTenantStatus(r.Context(), id, *req.Status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		default:
			response.Fail(w, http.StatusInternalServerError, "更新租户状态失败")
		}
		return
	}

	// 4. 返回成功响应
	response.Success(w, nil)
}

// List GET /api/v1/admin/tenants?page=1&page_size=10&name=极客
// List 是一个处理 HTTP 请求的方法，用于获取租户列表
// 它接收一个 http.ResponseWriter 和 http.Request 作为参数，并返回租户列表的分页数据
func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL query 获取参数
	// 从请求的 URL 查询参数中获取分页页码、每页大小和租户名称
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	name := r.URL.Query().Get("name")

	// 2. 解析分页参数
	// 将获取到的页码和每页大小从字符串转换为整数
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	// 3. 调用服务层查询
	// 调用服务层的 QueryTenantList 方法查询租户列表，传入上下文、页码、每页大小和租户名称
	tenants, total, err := h.svc.QueryTenantList(r.Context(), pg.Page, pg.PageSize, name)
	if err != nil {
		// 如果查询失败，返回错误响应
		response.Fail(w, http.StatusInternalServerError, "查询租户列表失败")
		return
	}

	// 4. 转换为 DTO
	// 将查询到的租户数据转换为前端需要的响应格式
	var list []dto.AdminTenantResp
	for _, tenant := range tenants {
		list = append(list, dto.AdminTenantResp{
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
		})
	}

	// 5. 使用分页包返回结果
	// 使用分页工具包装结果，并返回成功响应
	result := pagination.NewPaginatedResult(list, page, pageSize, int(total))
	response.Success(w, result)
}

// AdminDetail GET /api/v1/admin/tenants/:id
func (h *TenantHandler) AdminDetail(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL 获取 ID
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	// 2. 调用服务层查询
	tenant, err := h.svc.AdminDetail(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		default:
			response.Fail(w, http.StatusInternalServerError, "查询租户详情失败")
		}
		return
	}

	// 3. 转换为 DTO
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

// AdminUpdate PUT /api/v1/admin/tenants/:id
func (h *TenantHandler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL 获取 ID
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	// 2. 从请求体获取更新数据
	var req dto.AdminUpdateTenantReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 3. 调用服务层更新
	err := h.svc.AdminUpdate(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		case errors.Is(err, service.ErrDomainConflict):
			response.Fail(w, http.StatusConflict, "该域名已被其他租户占用")
		case errors.Is(err, service.ErrNameConflict):
			response.Fail(w, http.StatusConflict, "该租户名称已被使用")
		default:
			response.Fail(w, http.StatusInternalServerError, "更新租户信息失败")
		}
		return
	}

	// 4. 返回成功响应
	response.Success(w, nil)
}

// AdminDelete DELETE /api/v1/admin/tenants/:id
func (h *TenantHandler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL 获取 ID
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	// 2. 调用服务层删除
	err := h.svc.AdminDelete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		default:
			response.Fail(w, http.StatusInternalServerError, "删除租户失败")
		}
		return
	}

	// 3. 返回成功响应
	response.Success(w, nil)
}

// AdminBatchUpdateStatus PUT /api/v1/admin/tenants/batch/status
func (h *TenantHandler) AdminBatchUpdateStatus(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求体
	var req dto.AdminBatchUpdateStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用服务层批量更新
	affected, failedIDs, err := h.svc.BatchUpdateStatus(r.Context(), req.IDs, *req.Status)
	if err != nil {
		fmt.Println(err)
		response.Fail(w, http.StatusInternalServerError, "批量更新租户状态失败")
		return
	}

	// 3. 组装结果
	resp := dto.AdminBatchOperationResp{
		Total:     len(req.IDs),
		Succeeded: int(affected),
		Failed:    len(failedIDs),
		FailedIDs: failedIDs,
	}

	// 4. 返回成功响应
	response.Success(w, resp)
}

// AdminBatchDelete DELETE /api/v1/admin/tenants/batch
func (h *TenantHandler) AdminBatchDelete(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求体
	var req dto.AdminBatchDeleteReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 2. 调用服务层批量删除
	affected, failedIDs, err := h.svc.BatchDelete(r.Context(), req.IDs)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "批量删除租户失败")
		return
	}

	// 3. 组装结果
	resp := dto.AdminBatchOperationResp{
		Total:     len(req.IDs),
		Succeeded: int(affected),
		Failed:    len(failedIDs),
		FailedIDs: failedIDs,
	}

	// 4. 返回成功响应
	response.Success(w, resp)
}

// AdminDeletedList GET /api/v1/admin/tenants/deleted?page=1&page_size=10&name=极客
func (h *TenantHandler) AdminDeletedList(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL query 获取参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	name := r.URL.Query().Get("name")

	// 2. 解析分页参数
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	// 3. 调用服务层查询已删除的租户
	tenants, total, err := h.svc.FindDeleted(r.Context(), pg.Page, pg.PageSize, name)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "查询回收站租户列表失败")
		return
	}

	// 4. 转换为 DTO
	var list []dto.AdminDeletedTenantResp
	for _, tenant := range tenants {
		deletedAt := time.Time{}
		if tenant.DeletedAt != nil {
			deletedAt = *tenant.DeletedAt
		}
		list = append(list, dto.AdminDeletedTenantResp{
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
			DeletedAt:    deletedAt,
		})
	}

	// 5. 使用分页包返回结果
	result := pagination.NewPaginatedResult(list, page, pageSize, int(total))
	response.Success(w, result)
}

// AdminRestore PUT /api/v1/admin/tenants/{id}/restore
func (h *TenantHandler) AdminRestore(w http.ResponseWriter, r *http.Request) {
	// 1. 从 URL 获取 ID
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	// 2. 调用服务层恢复租户
	err := h.svc.Restore(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotDeleted):
			response.Fail(w, http.StatusNotFound, "租户不存在或未处于已删除状态")
		default:
			response.Fail(w, http.StatusInternalServerError, "恢复租户失败")
		}
		return
	}

	// 3. 返回成功响应
	response.Success(w, nil)
}

// GetCurrentTenant GET /api/v1/tenants/current
// GetCurrentTenant 获取当前租户信息的处理函数
// 该函数从请求上下文中获取租户ID，查询租户信息，并将其转换为DTO返回
// @param w HTTP响应写入器
// @param r HTTP请求指针
func (h *TenantHandler) GetCurrentTenant(w http.ResponseWriter, r *http.Request) {
	// 1. 从上下文获取当前租户ID
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 2. 调用服务层查询
	tenant, err := h.svc.GetCurrentTenant(r.Context(), tenantID)
	if err != nil {
		// 根据错误类型返回不同的响应
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		default:
			response.Fail(w, http.StatusInternalServerError, "查询租户详情失败")
		}
		return
	}

	// 3. 转换为 DTO
	converter := dto.TenantConverter{}
	response.Success(w, converter.ToTenantResponse(tenant))
}

// UpdateCurrentTenant PUT /api/v1/tenants/current
func (h *TenantHandler) UpdateCurrentTenant(w http.ResponseWriter, r *http.Request) {
	// 1. 从上下文获取当前租户ID
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 2. 解析请求体
	var req dto.UpdateCurrentTenantReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 3. 调用服务层更新
	err := h.svc.UpdateCurrentTenant(r.Context(), tenantID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		case errors.Is(err, service.ErrNameConflict):
			response.Fail(w, http.StatusConflict, "该租户名称已被使用")
		default:
			response.Fail(w, http.StatusInternalServerError, "更新租户信息失败")
		}
		return
	}

	// 4. 返回成功响应
	response.Success(w, nil)
}

// GetInitStatus GET /api/v1/tenants/current/status
func (h *TenantHandler) GetInitStatus(w http.ResponseWriter, r *http.Request) {
	// 1. 从上下文获取当前租户ID
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 2. 调用服务层查询
	status, err := h.svc.GetInitStatus(r.Context(), tenantID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		default:
			response.Fail(w, http.StatusInternalServerError, "查询租户初始化状态失败")
		}
		return
	}

	// 3. 返回成功响应
	response.Success(w, status)
}

// ApplyCancellation POST /api/v1/tenants/current/cancel
func (h *TenantHandler) ApplyCancellation(w http.ResponseWriter, r *http.Request) {
	// 1. 从上下文获取当前租户ID
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	// 2. 解析请求体
	var req dto.ApplyCancellationReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	// 3. 调用服务层申请注销
	result, err := h.svc.ApplyCancellation(r.Context(), tenantID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			response.Fail(w, http.StatusNotFound, "租户不存在")
		default:
			response.Fail(w, http.StatusInternalServerError, "申请注销失败")
		}
		return
	}

	// 4. 返回成功响应
	response.Success(w, result)
}
