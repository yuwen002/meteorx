package service

import (
	"context"
	"time"

	"meteorx/internal/middleware"
	auditModel "meteorx/internal/modules/audit/model"
	auditRepo "meteorx/internal/modules/audit/repository"
	"meteorx/pkg/idgen"
)

// PermissionAuditStorageImpl 权限审计日志存储实现
// 将权限变更日志写入审计日志模块的数据库存储
type PermissionAuditStorageImpl struct {
	auditRepo auditRepo.AuditLogRepository
}

// NewPermissionAuditStorage 创建权限审计日志存储
func NewPermissionAuditStorage(repo auditRepo.AuditLogRepository) *PermissionAuditStorageImpl {
	return &PermissionAuditStorageImpl{
		auditRepo: repo,
	}
}

// SavePermissionAuditLog 保存权限审计日志
func (s *PermissionAuditStorageImpl) SavePermissionAuditLog(ctx context.Context, log *middleware.PermissionAuditLog) error {
	auditLog := &auditModel.AuditLog{
		ID:         idgen.New(),
		UserID:     log.UserID,
		Username:   log.ResourceName,
		TenantID:   "",
		Module:     "rbac",
		Action:     mapAction(log.Action),
		Resource:   log.ResourceType,
		ResourceID: log.ResourceID,
		IPAddress:  log.IPAddress,
		UserAgent:  log.UserAgent,
		RiskLevel:  auditModel.RiskMedium,
		Result:     auditModel.ResultSuccess,
		CreatedAt:  log.CreatedAt,
	}

	if auditLog.CreatedAt.IsZero() {
		auditLog.CreatedAt = time.Now()
	}

	return s.auditRepo.Create(ctx, auditLog)
}

// mapAction 将权限操作映射为审计日志操作类型
func mapAction(action string) string {
	switch action {
	case "assign", "grant":
		return auditModel.ActionTypeCreate
	case "remove", "revoke":
		return auditModel.ActionTypeDelete
	default:
		return auditModel.ActionTypeOther
	}
}

// Ensure interface compliance
var _ middleware.PermissionAuditStorage = (*PermissionAuditStorageImpl)(nil)