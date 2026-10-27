package handler

import (
	"context"
	"errors"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/service"
	"net/http"
)

// IntegrityVerifyService 完整性用例接口（handler 依赖的最小业务面；*audit.IntegrityCoordinator 实现）
type IntegrityVerifyService interface {
	Verify(ctx context.Context) (*dto.IntegrityVerifyResp, error)
	Rehash(ctx context.Context) (*dto.IntegrityRehashResp, error)
}

// IntegrityHandler 审计完整性校验处理器
type IntegrityHandler struct {
	svc IntegrityVerifyService
}

// NewIntegrityHandler 创建完整性校验处理器
func NewIntegrityHandler(svc IntegrityVerifyService) *IntegrityHandler {
	return &IntegrityHandler{svc: svc}
}

// VerifyIntegrity 校验审计日志完整性（行摘要 + 锚点链），仅超级管理员可触发（整组已在 RequiresMasterAdmin 下）。
// POST /api/v1/audit/integrity/verify
//
// 只读重放、不改任何数据；verdict 为 verified / tampered / pending_rehash / not_configured。
func (h *IntegrityHandler) VerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Verify(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "完整性校验失败")
		return
	}
	response.Success(w, result)
}

// RehashIntegrity 密钥轮换迁移：把仅匹配旧 pepper 的历史行重算到当前 pepper 并重建锚点链，仅超管可触发。
// POST /api/v1/audit/integrity/rehash
//
// 未启用或未配旧 pepper（非轮换期）时返回 400；其余异常 500。成功返回重算行数与重建锚点数。
func (h *IntegrityHandler) RehashIntegrity(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Rehash(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIntegrityNotEnabled), errors.Is(err, service.ErrIntegrityNoPreviousKey):
			response.Fail(w, http.StatusBadRequest, err.Error())
		default:
			response.Fail(w, http.StatusInternalServerError, "密钥重算失败")
		}
		return
	}
	response.Success(w, result)
}
