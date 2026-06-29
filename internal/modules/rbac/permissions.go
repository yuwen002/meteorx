package rbac

import (
	"context"
	"meteorx/internal/modules/rbac/model"
	"meteorx/internal/modules/rbac/service"
)

// PermissionDef 预定义的权限信息，用于服务启动时自动注册到数据库
// code 作为唯一键做幂等判断：已存在则跳过，不存在则插入
type PermissionDef struct {
	Name        string // 权限名称（中文，便于后台显示）
	Code        string // 权限编码（唯一标识，中间件权限校验时用）
	Description string // 权限描述
	Resource    string // 资源类型（如 user/role/permission）
	Action      string // 操作类型（如 list/create/read/update/delete）
}

// ============================================================
// 用户模块权限码（/api/v1/users 相关）
// ============================================================
const (
	UserList   = "user:list"   // 查询用户列表
	UserCreate = "user:create" // 创建用户
	UserRead   = "user:read"   // 查询用户详情
	UserUpdate = "user:update" // 更新用户信息
	UserDelete = "user:delete" // 删除用户
)

// ============================================================
// 角色管理权限码（/api/v1/rbac/roles 相关）
// ============================================================
const (
	RoleList         = "rbac:role:list"           // 查询角色列表
	RoleCreate       = "rbac:role:create"         // 创建角色
	RoleRead         = "rbac:role:read"           // 查询角色详情
	RoleUpdate       = "rbac:role:update"         // 更新角色
	RoleDelete       = "rbac:role:delete"         // 删除角色
	RoleStatus       = "rbac:role:status"         // 切换角色状态
	RoleBatchStatus  = "rbac:role:batch:status"   // 批量切换角色状态
	RoleBatchDelete  = "rbac:role:batch:delete"   // 批量删除角色
	RoleListDeleted  = "rbac:role:list_deleted"   // 查询已删除角色（回收站）
	RoleRestore      = "rbac:role:restore"        // 恢复已删除角色
	RoleBindPerm     = "rbac:role:bind_perm"      // 角色绑定权限
	RoleUnbindPerm   = "rbac:role:unbind_perm"    // 角色解绑单个权限
	RoleUnbindPerms  = "rbac:role:unbind_perms"   // 角色批量解绑权限
	RoleGetPerms     = "rbac:role:get_perms"      // 查询角色的权限列表
	RolesBatchBind   = "rbac:roles:batch:bind"    // 批量为多个角色绑定权限
	RolesBatchUnbind = "rbac:roles:batch:unbind"  // 批量为多个角色解绑权限
)

// ============================================================
// 权限管理权限码（/api/v1/rbac/permissions 相关）
// ============================================================
const (
	PermissionList        = "rbac:perm:list"            // 查询权限列表
	PermissionCreate      = "rbac:perm:create"          // 创建权限
	PermissionRead        = "rbac:perm:read"            // 查询权限详情
	PermissionUpdate      = "rbac:perm:update"          // 更新权限
	PermissionDelete      = "rbac:perm:delete"          // 删除权限
	PermissionStatus      = "rbac:perm:status"          // 切换权限状态
	PermissionBatchStatus = "rbac:perm:batch:status"    // 批量切换权限状态
	PermissionBatchDelete = "rbac:perm:batch:delete"    // 批量删除权限
)

// ============================================================
// 角色-权限关系列表权限码
// ============================================================
const (
	RolePermissionList = "rbac:role_perm:list" // 查询角色权限关系列表
)

// ============================================================
// 用户-角色分配权限码
// ============================================================
const (
	UserRoleList         = "rbac:user_role:list"         // 查询用户角色关系列表
	UserRoleAssign       = "rbac:user_role:assign"       // 为用户分配角色
	UserRoleGetRoles     = "rbac:user_role:get_roles"    // 查询用户拥有的角色
	UserRoleRemoveOne    = "rbac:user_role:remove_one"   // 删除用户的单个角色
	UserRoleRemoveAll    = "rbac:user_role:remove_all"   // 删除用户的所有角色
	UserRoleGetUsers     = "rbac:user_role:get_users"    // 查询拥有某角色的用户ID列表
	UserRoleBatchAssign  = "rbac:user_role:batch:assign" // 批量为多个用户分配角色
)

// ============================================================
// 预定义权限列表（服务启动时自动注册到 permissions 表）
// ============================================================
func GetPermissionDefs() []PermissionDef {
	return []PermissionDef{
		// 用户模块
		{Name: "查询用户列表", Code: UserList, Description: "查询租户下用户列表", Resource: "user", Action: "list"},
		{Name: "创建用户", Code: UserCreate, Description: "创建新用户", Resource: "user", Action: "create"},
		{Name: "查询用户详情", Code: UserRead, Description: "查看单个用户详细信息", Resource: "user", Action: "read"},
		{Name: "更新用户", Code: UserUpdate, Description: "修改用户信息", Resource: "user", Action: "update"},
		{Name: "删除用户", Code: UserDelete, Description: "删除用户", Resource: "user", Action: "delete"},

		// 角色管理
		{Name: "查询角色列表", Code: RoleList, Description: "查询所有角色", Resource: "role", Action: "list"},
		{Name: "创建角色", Code: RoleCreate, Description: "创建新角色", Resource: "role", Action: "create"},
		{Name: "查询角色详情", Code: RoleRead, Description: "查看角色详情", Resource: "role", Action: "read"},
		{Name: "更新角色", Code: RoleUpdate, Description: "修改角色信息", Resource: "role", Action: "update"},
		{Name: "删除角色", Code: RoleDelete, Description: "删除角色", Resource: "role", Action: "delete"},
		{Name: "切换角色状态", Code: RoleStatus, Description: "启用/禁用单个角色", Resource: "role", Action: "status"},
		{Name: "批量切换角色状态", Code: RoleBatchStatus, Description: "批量启用/禁用角色", Resource: "role", Action: "batch_status"},
		{Name: "批量删除角色", Code: RoleBatchDelete, Description: "批量删除角色", Resource: "role", Action: "batch_delete"},
		{Name: "查询已删除角色", Code: RoleListDeleted, Description: "角色回收站列表", Resource: "role", Action: "list_deleted"},
		{Name: "恢复已删除角色", Code: RoleRestore, Description: "从回收站恢复角色", Resource: "role", Action: "restore"},
		{Name: "角色绑定权限", Code: RoleBindPerm, Description: "为角色分配权限", Resource: "role", Action: "bind_perm"},
		{Name: "角色解绑权限", Code: RoleUnbindPerm, Description: "移除角色的单个权限", Resource: "role", Action: "unbind_perm"},
		{Name: "角色批量解绑权限", Code: RoleUnbindPerms, Description: "移除角色的多个权限", Resource: "role", Action: "batch_unbind_perm"},
		{Name: "查询角色权限", Code: RoleGetPerms, Description: "查询角色拥有的权限列表", Resource: "role", Action: "get_perms"},
		{Name: "多角色批量绑定权限", Code: RolesBatchBind, Description: "批量为多个角色绑定权限", Resource: "role", Action: "batch_bind"},
		{Name: "多角色批量解绑权限", Code: RolesBatchUnbind, Description: "批量为多个角色解绑权限", Resource: "role", Action: "batch_unbind"},

		// 权限管理
		{Name: "查询权限列表", Code: PermissionList, Description: "查询所有权限", Resource: "permission", Action: "list"},
		{Name: "创建权限", Code: PermissionCreate, Description: "创建新权限", Resource: "permission", Action: "create"},
		{Name: "查询权限详情", Code: PermissionRead, Description: "查看权限详情", Resource: "permission", Action: "read"},
		{Name: "更新权限", Code: PermissionUpdate, Description: "修改权限信息", Resource: "permission", Action: "update"},
		{Name: "删除权限", Code: PermissionDelete, Description: "删除权限", Resource: "permission", Action: "delete"},
		{Name: "切换权限状态", Code: PermissionStatus, Description: "启用/禁用单个权限", Resource: "permission", Action: "status"},
		{Name: "批量切换权限状态", Code: PermissionBatchStatus, Description: "批量启用/禁用权限", Resource: "permission", Action: "batch_status"},
		{Name: "批量删除权限", Code: PermissionBatchDelete, Description: "批量删除权限", Resource: "permission", Action: "batch_delete"},

		// 角色-权限关系列表
		{Name: "查询角色权限关系", Code: RolePermissionList, Description: "查询角色与权限的关联关系列表", Resource: "role_permission", Action: "list"},

		// 用户-角色分配
		{Name: "查询用户角色关系", Code: UserRoleList, Description: "查询用户与角色的关联关系列表", Resource: "user_role", Action: "list"},
		{Name: "为用户分配角色", Code: UserRoleAssign, Description: "为用户指定角色（覆盖）", Resource: "user_role", Action: "assign"},
		{Name: "查询用户的角色", Code: UserRoleGetRoles, Description: "查询用户拥有的角色列表", Resource: "user_role", Action: "get_roles"},
		{Name: "删除用户单个角色", Code: UserRoleRemoveOne, Description: "移除用户的单个角色", Resource: "user_role", Action: "remove_one"},
		{Name: "删除用户全部角色", Code: UserRoleRemoveAll, Description: "清空用户的所有角色", Resource: "user_role", Action: "remove_all"},
		{Name: "查询角色下的用户", Code: UserRoleGetUsers, Description: "查询拥有某角色的用户ID列表", Resource: "user_role", Action: "get_users"},
		{Name: "批量为用户分配角色", Code: UserRoleBatchAssign, Description: "批量为多个用户分配角色", Resource: "user_role", Action: "batch_assign"},
	}
}

// SeedPermissions 将预定义权限注册到数据库
// 幂等：基于 code 唯一键判断，已存在则跳过，不存在则插入
// 返回 (新增数量, 总数量, error)
func SeedPermissions(ctx context.Context, svc *service.RBACService) (int, int, error) {
	defs := GetPermissionDefs()
	return svc.SeedPermissions(ctx, toModelDefs(defs))
}

// toModelDefs 将 []PermissionDef 转换为 service 层需要的结构
func toModelDefs(defs []PermissionDef) []*model.Permission {
	result := make([]*model.Permission, len(defs))
	for i, d := range defs {
		result[i] = &model.Permission{
			Name:        d.Name,
			Code:        d.Code,
			Description: d.Description,
			Resource:    d.Resource,
			Action:      d.Action,
			Status:      1, // 默认启用
		}
	}
	return result
}