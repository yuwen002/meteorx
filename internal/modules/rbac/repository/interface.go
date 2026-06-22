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
	Update(ctx context.Context, role *model.Role) error
	UpdateStatus(ctx context.Context, id string, status int) error
	BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, error)
	Delete(ctx context.Context, id string) error
	FindDeleted(ctx context.Context, page, pageSize int, keyword string) ([]*model.Role, int64, error)
	Restore(ctx context.Context, id string) error
}

type PermissionRepository interface {
	Create(ctx context.Context, permission *model.Permission) error
	GetByID(ctx context.Context, id string) (*model.Permission, error)
	GetByCode(ctx context.Context, code string) (*model.Permission, error)
	List(ctx context.Context, page, pageSize int, resource, keyword string) ([]*model.Permission, int64, error)
	Update(ctx context.Context, permission *model.Permission) error
	Delete(ctx context.Context, id string) error
}

type RolePermissionRepository interface {
	BindPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*model.Permission, error)
	GetPermissionCodesByRoleID(ctx context.Context, roleID string) ([]string, error)
	UnbindPermission(ctx context.Context, roleID, permissionID string) error
}
