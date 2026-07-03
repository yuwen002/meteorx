package repository

import (
	"context"
	"meteorx/internal/modules/rbac/model"
)

type RoleRepository interface {
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id string) (*model.Role, error)
	GetByCode(ctx context.Context, tenantID, code string) (*model.Role, error)
	List(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*model.Role, int64, error)
	ListByScope(ctx context.Context, scope string) ([]*model.Role, error)
	// ListSystemAdminRoles 查询系统管理员角色（IsSystem=true 且 scope 为 system 或 all）
	ListSystemAdminRoles(ctx context.Context) ([]*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	UpdateStatus(ctx context.Context, id string, status int) error
	BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, error)
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) (int64, error)
	FindDeleted(ctx context.Context, page, pageSize int, keyword string) ([]*model.Role, int64, error)
	Restore(ctx context.Context, id string) error
	Count(ctx context.Context, tenantID string) (int64, error)
}

type PermissionRepository interface {
	Create(ctx context.Context, permission *model.Permission) error
	GetByID(ctx context.Context, id string) (*model.Permission, error)
	GetByCode(ctx context.Context, code string) (*model.Permission, error)
	List(ctx context.Context, page, pageSize int, resource, keyword string) ([]*model.Permission, int64, error)
	Update(ctx context.Context, permission *model.Permission) error
	UpdateStatus(ctx context.Context, id string, status int) error
	BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, error)
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) (int64, error)
	Count(ctx context.Context) (int64, error)
}

type RolePermissionRepository interface {
	BindPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*model.Permission, error)
	GetPermissionsByRoleIDWithResource(ctx context.Context, roleID string, resource string) ([]*model.Permission, error)
	GetPermissionCodesByRoleID(ctx context.Context, roleID string) ([]string, error)
	UnbindPermission(ctx context.Context, roleID, permissionID string) error
	BatchBindPermissions(ctx context.Context, roleIDs []string, permissionIDs []string) (int64, error)
	BatchUnbindPermissions(ctx context.Context, roleIDs []string, permissionIDs []string) (int64, error)
	List(ctx context.Context, page, pageSize int, roleID, permissionID string) ([]*model.RolePermission, int64, error)
	CountByRoleID(ctx context.Context, roleID string) (int64, error)
	CountByPermissionID(ctx context.Context, permissionID string) (int64, error)
}

type UserRoleRepository interface {
	AssignRoles(ctx context.Context, userID string, roleIDs []string) error
	GetRoleIDsByUserID(ctx context.Context, userID string) ([]string, error)
	GetRoleCodesByUserID(ctx context.Context, userID string) ([]string, error)
	BatchGetRoleIDsByUserIDs(ctx context.Context, userIDs []string) (map[string][]string, error)
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteByUserIDAndRoleID(ctx context.Context, userID, roleID string) error
	GetUserIDsByRoleID(ctx context.Context, roleID string) ([]string, error)
	CountByRoleID(ctx context.Context, roleID string) (int64, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	CheckUserExists(ctx context.Context, userID string) error
	ListUserRoles(ctx context.Context, page, pageSize int, userID, roleID string) ([]*model.UserRole, int64, error)
}