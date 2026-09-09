package middleware

import (
	"context"
	"time"

	"meteorx/pkg/logger"
)

// PermissionAuditLog 权限审计日志记录
type PermissionAuditLog struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	OperatorID   string    `json:"operator_id"`
	Action       string    `json:"action"`        // assign/remove
	ResourceType string    `json:"resource_type"` // role/permission
	ResourceID   string    `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
}

// PermissionAuditLogger 权限审计日志记录器
type PermissionAuditLogger struct {
	storage PermissionAuditStorage
}

// PermissionAuditStorage 权限审计日志存储接口
type PermissionAuditStorage interface {
	SavePermissionAuditLog(ctx context.Context, log *PermissionAuditLog) error
}

// NewPermissionAuditLogger 创建权限审计日志记录器
func NewPermissionAuditLogger(storage PermissionAuditStorage) *PermissionAuditLogger {
	return &PermissionAuditLogger{
		storage: storage,
	}
}

// LogPermissionChange 记录权限变更
func (pal *PermissionAuditLogger) LogPermissionChange(
	ctx context.Context,
	userID, operatorID, action, resourceType, resourceID, resourceName string,
) {
	log := &PermissionAuditLog{
		UserID:       userID,
		OperatorID:   operatorID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		IPAddress:    logger.GetRequestID(ctx), // 简化处理，实际应从 context 获取
		CreatedAt:    time.Now(),
	}

	// 异步保存日志（不阻塞主流程）
	go func() {
		if err := pal.storage.SavePermissionAuditLog(context.Background(), log); err != nil {
			logger.Error("Failed to save permission audit log",
				"error", err,
				"user_id", userID,
				"action", action,
			)
		}
	}()

	// 同时记录到应用日志
	logger.Ctx(ctx).Info("Permission change audited",
		"user_id", userID,
		"operator_id", operatorID,
		"action", action,
		"resource_type", resourceType,
		"resource_id", resourceID,
		"resource_name", resourceName,
	)
}

// LogRoleAssignment 记录角色分配
func (pal *PermissionAuditLogger) LogRoleAssignment(ctx context.Context, userID, operatorID, roleID, roleName string) {
	pal.LogPermissionChange(ctx, userID, operatorID, "assign", "role", roleID, roleName)
}

// LogRoleRemoval 记录角色移除
func (pal *PermissionAuditLogger) LogRoleRemoval(ctx context.Context, userID, operatorID, roleID, roleName string) {
	pal.LogPermissionChange(ctx, userID, operatorID, "remove", "role", roleID, roleName)
}

// LogPermissionGrant 记录权限授予
func (pal *PermissionAuditLogger) LogPermissionGrant(ctx context.Context, userID, operatorID, permCode, permName string) {
	pal.LogPermissionChange(ctx, userID, operatorID, "grant", "permission", permCode, permName)
}

// LogPermissionRevoke 记录权限撤销
func (pal *PermissionAuditLogger) LogPermissionRevoke(ctx context.Context, userID, operatorID, permCode, permName string) {
	pal.LogPermissionChange(ctx, userID, operatorID, "revoke", "permission", permCode, permName)
}