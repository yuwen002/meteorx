package handler

import (
	"context"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/audit/dto"
	"net/http"
)

// IntegrityVerifyService 完整性校验服务接口（handler 依赖的最小业务面；*service.IntegrityService 实现）
type IntegrityVerifyService interface {
	Verify(ctx context.Context) (*dto.IntegrityVerifyResp, error)
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
// 只读重放、不改任何数据；verdict 为 verified / tampered / not_configured。
func (h *IntegrityHandler) VerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Verify(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "完整性校验失败")
		return
	}
	response.Success(w, result)
}
