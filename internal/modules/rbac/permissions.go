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
// 用户模块权限码（/api/v1/users 相关，租户内用户管理）
// ============================================================
const (
	UserList             = "user:list"              // 查询租户用户列表
	UserCreate           = "user:create"            // 创建租户用户
	UserRead             = "user:read"              // 查询租户用户详情
	UserUpdate           = "user:update"            // 更新租户用户
	UserDelete           = "user:delete"            // 删除租户用户（软删除）
	UserResetPassword    = "user:reset_password"    // 重置租户用户密码
	UserListDeleted      = "user:list_deleted"      // 查询已删除租户用户（回收站）
	UserRestore          = "user:restore"           // 恢复已删除租户用户
	UserPermanentDelete  = "user:permanent_delete"  // 永久删除租户用户（回收站）
	UserBatchStatus      = "user:batch_status"      // 批量启用/禁用租户用户
	UserBatchDelete      = "user:batch_delete"      // 批量删除租户用户
)

// ============================================================
// 角色管理权限码（/api/v1/rbac/roles 相关）
// ============================================================
const (
	RoleList            = "rbac:role:list"              // 查询角色列表
	RoleListSelect      = "rbac:role:list_select"       // 查询角色下拉列表（不分页）
	RoleListSystemAdmin = "rbac:role:list_system_admin" // 查询系统管理员角色列表
	RoleCreate          = "rbac:role:create"            // 创建角色
	RoleRead            = "rbac:role:read"              // 查询角色详情
	RoleUpdate          = "rbac:role:update"            // 更新角色
	RoleDelete          = "rbac:role:delete"            // 删除角色
	RoleStatus          = "rbac:role:status"            // 切换角色状态
	RoleBatchStatus     = "rbac:role:batch_status"      // 批量切换角色状态
	RoleBatchDelete     = "rbac:role:batch_delete"      // 批量删除角色
	RoleListDeleted     = "rbac:role:list_deleted"      // 查询已删除角色（回收站）
	RoleRestore         = "rbac:role:restore"           // 恢复已删除角色
	RoleBindPerm        = "rbac:role:bind_perm"         // 角色绑定权限
	RoleUnbindPerm      = "rbac:role:unbind_perm"       // 角色解绑单个权限
	RoleBatchUnbindPerm = "rbac:role:batch_unbind_perm" // 角色批量解绑权限
	RoleGetPerms        = "rbac:role:get_perms"         // 查询角色的权限列表
	RolesBatchBind      = "rbac:role:batch_bind"        // 批量为多个角色绑定权限
	RolesBatchUnbind    = "rbac:role:batch_unbind"      // 批量为多个角色解绑权限
)

// ============================================================
// 权限管理权限码（/api/v1/rbac/permissions 相关）
// ============================================================
const (
	PermissionList        = "rbac:perm:list"             // 查询权限列表
	PermissionCreate      = "rbac:perm:create"           // 创建权限
	PermissionRead        = "rbac:perm:read"             // 查询权限详情
	PermissionUpdate      = "rbac:perm:update"           // 更新权限
	PermissionDelete      = "rbac:perm:delete"           // 删除权限
	PermissionStatus      = "rbac:perm:status"           // 切换权限状态
	PermissionBatchStatus = "rbac:perm:batch_status"     // 批量切换权限状态
	PermissionBatchDelete = "rbac:perm:batch_delete"     // 批量删除权限
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
	UserRoleList        = "rbac:user_role:list"          // 查询用户角色关系列表
	UserRoleBatchAssign = "rbac:user_role:batch_assign"  // 批量为多个用户分配角色
	UserRoleAssign      = "rbac:user_role:assign"        // 为用户分配角色
	UserRoleGetRoles    = "rbac:user_role:get_roles"     // 查询用户拥有的角色
	UserRoleRemoveOne   = "rbac:user_role:remove_one"    // 删除用户的单个角色
	UserRoleRemoveAll   = "rbac:user_role:remove_all"    // 删除用户的所有角色
	UserRoleGetUsers    = "rbac:user_role:get_users"     // 查询拥有某角色的用户ID列表
)

// ============================================================
// 套餐管理权限码（/api/v1/admin/plans 相关）
// ============================================================
const (
	PlanList   = "admin:plan:list"   // 查询套餐列表
	PlanCreate = "admin:plan:create" // 创建套餐
	PlanRead   = "admin:plan:read"   // 查询套餐详情
	PlanUpdate = "admin:plan:update" // 编辑套餐
	PlanDelete = "admin:plan:delete" // 删除套餐
	PlanAssign = "admin:plan:assign" // 为租户分配套餐
	PlanSelect = "admin:plan:select" // 查询启用套餐下拉列表
)

// ============================================================
// 审计日志权限码（/api/v1/audit 相关）
// ============================================================
const (
	AuditLogList     = "audit:log:list"     // 查询审计日志列表
	AuditLogExport   = "audit:log:export"   // 导出审计日志
	AuditLogCreate   = "audit:log:create"   // 创建审计日志（内部使用）
	AuditLogRead     = "audit:log:read"     // 查询审计日志详情
	AuditLogCleanup  = "audit:log:cleanup"  // 清理审计日志
)

// ============================================================
// 租户管理权限码（/api/v1/admin/tenants 相关，平台后台）
// ============================================================
const (
	AdminTenantList            = "admin:tenant:list"              // 查询租户列表
	AdminTenantCreate          = "admin:tenant:create"            // 后台新建租户
	AdminTenantRead            = "admin:tenant:read"              // 查询租户详情
	AdminTenantUpdate          = "admin:tenant:update"            // 更新租户
	AdminTenantDelete          = "admin:tenant:delete"            // 删除租户（软删除）
	AdminTenantStatus          = "admin:tenant:status"           // 启用/禁用租户
	AdminTenantListDeleted     = "admin:tenant:list_deleted"      // 查询已删除租户（回收站）
	AdminTenantRestore         = "admin:tenant:restore"          // 恢复已删除租户
	AdminTenantBatchStatus     = "admin:tenant:batch_status"    // 批量启用/禁用租户
	AdminTenantBatchDelete     = "admin:tenant:batch_delete"    // 批量删除租户
)

// ============================================================
// 系统管理员权限码（/api/v1/admin/users 相关，平台后台）
// ============================================================
const (
	AdminMasterList             = "admin:master:list"             // 查询系统管理员列表
	AdminMasterCreate           = "admin:master:create"           // 创建系统管理员
	AdminMasterRead             = "admin:master:read"            // 查询系统管理员详情
	AdminMasterUpdate           = "admin:master:update"          // 更新系统管理员
	AdminMasterStatus           = "admin:master:status"          // 启用/禁用系统管理员
	AdminMasterDelete           = "admin:master:delete"          // 删除系统管理员（软删除）
	AdminMasterListDeleted      = "admin:master:list_deleted"    // 查询已删除系统管理员（回收站）
	AdminMasterRestore          = "admin:master:restore"        // 恢复已删除系统管理员
	AdminMasterPermanentDelete = "admin:master:permanent_delete" // 永久删除系统管理员
	AdminMasterBatchStatus      = "admin:master:batch_status"   // 批量启用/禁用系统管理员
	AdminMasterBatchDelete      = "admin:master:batch_delete"   // 批量删除系统管理员
)

// ============================================================
// 跨租户用户管理权限码（/api/v1/admin/tenant-users 相关，平台后台）
// ============================================================
const (
	AdminTenantUserList             = "admin:tenant_user:list"              // 查询全量租户用户列表
	AdminTenantUserListAll          = "admin:tenant_user:list_all"         // 查询全量租户用户（排除系统管理员）
	AdminTenantUserCreate           = "admin:tenant_user:create"           // 为指定租户创建用户
	AdminTenantUserRead             = "admin:tenant_user:read"            // 查询租户用户详情
	AdminTenantUserUpdate           = "admin:tenant_user:update"          // 更新租户用户
	AdminTenantUserStatus           = "admin:tenant_user:status"          // 启用/禁用租户用户
	AdminTenantUserResetPassword    = "admin:tenant_user:reset_password"  // 重置租户用户密码
	AdminTenantUserDelete           = "admin:tenant_user:delete"          // 删除租户用户（软删除）
	AdminTenantUserListDeletedAll   = "admin:tenant_user:list_deleted_all" // 查询全量已删除租户用户
	AdminTenantUserListDeleted      = "admin:tenant_user:list_deleted"    // 查询指定租户已删除用户
	AdminTenantUserRestore          = "admin:tenant_user:restore"         // 恢复已删除租户用户
	AdminTenantUserPermanentDelete  = "admin:tenant_user:permanent_delete" // 永久删除租户用户
	AdminTenantUserBatchStatus      = "admin:tenant_user:batch_status"    // 批量启用/禁用租户用户
	AdminTenantUserBatchDelete      = "admin:tenant_user:batch_delete"    // 批量删除租户用户
)

// ============================================================
// 租户侧套餐权限码（/api/v1/tenant/current/plan 相关，租户私有）
// ============================================================
const (
	TenantPlanCurrent = "tenant:plan:current" // 查询当前租户套餐与用量
)

// ============================================================
// 预定义权限列表（服务启动时自动注册到 permissions 表）
// ============================================================
func GetPermissionDefs() []PermissionDef {
	return []PermissionDef{
		// 用户模块（租户内用户管理）
		{Name: "查询租户用户列表", Code: UserList, Description: "查询租户下用户列表", Resource: "user", Action: "list"},
		{Name: "创建租户用户", Code: UserCreate, Description: "创建新用户", Resource: "user", Action: "create"},
		{Name: "查询租户用户详情", Code: UserRead, Description: "查看单个用户详细信息", Resource: "user", Action: "read"},
		{Name: "更新租户用户", Code: UserUpdate, Description: "修改用户信息", Resource: "user", Action: "update"},
		{Name: "删除租户用户", Code: UserDelete, Description: "删除用户（软删除）", Resource: "user", Action: "delete"},
		{Name: "重置租户用户密码", Code: UserResetPassword, Description: "管理员重置用户密码（不需要原密码）", Resource: "user", Action: "reset_password"},
		{Name: "查询已删除租户用户", Code: UserListDeleted, Description: "租户用户回收站列表", Resource: "user", Action: "list_deleted"},
		{Name: "恢复已删除租户用户", Code: UserRestore, Description: "从回收站恢复租户用户", Resource: "user", Action: "restore"},
		{Name: "永久删除租户用户", Code: UserPermanentDelete, Description: "从回收站永久删除租户用户", Resource: "user", Action: "permanent_delete"},
		{Name: "批量启用/禁用租户用户", Code: UserBatchStatus, Description: "批量启用/禁用租户用户", Resource: "user", Action: "batch_status"},
		{Name: "批量删除租户用户", Code: UserBatchDelete, Description: "批量删除租户用户", Resource: "user", Action: "batch_delete"},

		// 角色管理
		{Name: "查询角色列表", Code: RoleList, Description: "查询所有角色", Resource: "role", Action: "list"},
		{Name: "查询角色下拉列表", Code: RoleListSelect, Description: "查询角色下拉列表（不分页）", Resource: "role", Action: "list_select"},
		{Name: "查询系统管理员角色列表", Code: RoleListSystemAdmin, Description: "查询系统管理员角色列表（用于创建系统管理员时选择角色）", Resource: "role", Action: "list_system_admin"},
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
		{Name: "角色批量解绑权限", Code: RoleBatchUnbindPerm, Description: "移除角色的多个权限", Resource: "role", Action: "batch_unbind_perm"},
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

		// 审计日志
		{Name: "查询审计日志列表", Code: AuditLogList, Description: "查询系统审计日志列表", Resource: "audit_log", Action: "list"},
		{Name: "导出审计日志", Code: AuditLogExport, Description: "导出审计日志为 CSV 文件", Resource: "audit_log", Action: "export"},
		{Name: "创建审计日志", Code: AuditLogCreate, Description: "创建审计日志记录（内部使用）", Resource: "audit_log", Action: "create"},
		{Name: "查询审计日志详情", Code: AuditLogRead, Description: "查看单个审计日志详情", Resource: "audit_log", Action: "read"},
		{Name: "清理审计日志", Code: AuditLogCleanup, Description: "清理过期审计日志", Resource: "audit_log", Action: "cleanup"},

		// 套餐管理
		{Name: "查询套餐列表", Code: PlanList, Description: "查询套餐列表", Resource: "plan", Action: "list"},
		{Name: "创建套餐", Code: PlanCreate, Description: "创建新套餐", Resource: "plan", Action: "create"},
		{Name: "查询套餐详情", Code: PlanRead, Description: "查询套餐详情", Resource: "plan", Action: "read"},
		{Name: "编辑套餐", Code: PlanUpdate, Description: "编辑套餐信息", Resource: "plan", Action: "update"},
		{Name: "删除套餐", Code: PlanDelete, Description: "删除套餐", Resource: "plan", Action: "delete"},
		{Name: "分配套餐", Code: PlanAssign, Description: "为租户分配/变更套餐", Resource: "plan", Action: "assign"},
		{Name: "查询套餐下拉", Code: PlanSelect, Description: "查询启用套餐下拉列表（不分页）", Resource: "plan", Action: "select"},

		// 租户侧套餐
		{Name: "查询当前租户套餐", Code: TenantPlanCurrent, Description: "查询当前租户套餐与用量", Resource: "plan", Action: "current"},

		// 租户管理（平台后台）
		{Name: "查询租户列表", Code: AdminTenantList, Description: "平台后台查询全量租户列表", Resource: "tenant", Action: "list"},
		{Name: "后台新建租户", Code: AdminTenantCreate, Description: "平台后台手动创建租户", Resource: "tenant", Action: "create"},
		{Name: "查询租户详情", Code: AdminTenantRead, Description: "平台后台查询租户详情", Resource: "tenant", Action: "read"},
		{Name: "更新租户", Code: AdminTenantUpdate, Description: "平台后台更新租户信息", Resource: "tenant", Action: "update"},
		{Name: "删除租户", Code: AdminTenantDelete, Description: "平台后台删除租户（软删除）", Resource: "tenant", Action: "delete"},
		{Name: "启用/禁用租户", Code: AdminTenantStatus, Description: "平台后台启用/禁用租户", Resource: "tenant", Action: "status"},
		{Name: "查询已删除租户", Code: AdminTenantListDeleted, Description: "平台后台租户回收站列表", Resource: "tenant", Action: "list_deleted"},
		{Name: "恢复已删除租户", Code: AdminTenantRestore, Description: "平台后台恢复已删除租户", Resource: "tenant", Action: "restore"},
		{Name: "批量启用/禁用租户", Code: AdminTenantBatchStatus, Description: "平台后台批量启用/禁用租户", Resource: "tenant", Action: "batch_status"},
		{Name: "批量删除租户", Code: AdminTenantBatchDelete, Description: "平台后台批量删除租户", Resource: "tenant", Action: "batch_delete"},

		// 系统管理员管理（平台后台）
		{Name: "查询系统管理员列表", Code: AdminMasterList, Description: "平台后台查询系统管理员列表", Resource: "master_admin", Action: "list"},
		{Name: "创建系统管理员", Code: AdminMasterCreate, Description: "平台后台创建系统管理员", Resource: "master_admin", Action: "create"},
		{Name: "查询系统管理员详情", Code: AdminMasterRead, Description: "平台后台查询系统管理员详情", Resource: "master_admin", Action: "read"},
		{Name: "更新系统管理员", Code: AdminMasterUpdate, Description: "平台后台更新系统管理员", Resource: "master_admin", Action: "update"},
		{Name: "启用/禁用系统管理员", Code: AdminMasterStatus, Description: "平台后台启用/禁用系统管理员", Resource: "master_admin", Action: "status"},
		{Name: "删除系统管理员", Code: AdminMasterDelete, Description: "平台后台删除系统管理员（软删除）", Resource: "master_admin", Action: "delete"},
		{Name: "查询已删除系统管理员", Code: AdminMasterListDeleted, Description: "平台后台系统管理员回收站列表", Resource: "master_admin", Action: "list_deleted"},
		{Name: "恢复已删除系统管理员", Code: AdminMasterRestore, Description: "平台后台恢复已删除系统管理员", Resource: "master_admin", Action: "restore"},
		{Name: "永久删除系统管理员", Code: AdminMasterPermanentDelete, Description: "平台后台永久删除系统管理员", Resource: "master_admin", Action: "permanent_delete"},
		{Name: "批量启用/禁用系统管理员", Code: AdminMasterBatchStatus, Description: "平台后台批量启用/禁用系统管理员", Resource: "master_admin", Action: "batch_status"},
		{Name: "批量删除系统管理员", Code: AdminMasterBatchDelete, Description: "平台后台批量删除系统管理员", Resource: "master_admin", Action: "batch_delete"},

		// 跨租户用户管理（平台后台）
		{Name: "查询全量租户用户列表", Code: AdminTenantUserListAll, Description: "平台后台查询全量租户用户", Resource: "tenant_user", Action: "list_all"},
		{Name: "查询指定租户用户列表", Code: AdminTenantUserList, Description: "平台后台查询指定租户用户", Resource: "tenant_user", Action: "list"},
		{Name: "为指定租户创建用户", Code: AdminTenantUserCreate, Description: "平台后台为指定租户创建用户", Resource: "tenant_user", Action: "create"},
		{Name: "查询租户用户详情", Code: AdminTenantUserRead, Description: "平台后台查询租户用户详情", Resource: "tenant_user", Action: "read"},
		{Name: "更新租户用户", Code: AdminTenantUserUpdate, Description: "平台后台更新租户用户", Resource: "tenant_user", Action: "update"},
		{Name: "启用/禁用租户用户", Code: AdminTenantUserStatus, Description: "平台后台启用/禁用租户用户", Resource: "tenant_user", Action: "status"},
		{Name: "重置租户用户密码", Code: AdminTenantUserResetPassword, Description: "平台后台重置租户用户密码", Resource: "tenant_user", Action: "reset_password"},
		{Name: "删除租户用户", Code: AdminTenantUserDelete, Description: "平台后台删除租户用户（软删除）", Resource: "tenant_user", Action: "delete"},
		{Name: "查询全量已删除租户用户", Code: AdminTenantUserListDeletedAll, Description: "平台后台查询全量已删除租户用户", Resource: "tenant_user", Action: "list_deleted_all"},
		{Name: "查询指定租户已删除用户", Code: AdminTenantUserListDeleted, Description: "平台后台查询指定租户已删除用户", Resource: "tenant_user", Action: "list_deleted"},
		{Name: "恢复已删除租户用户", Code: AdminTenantUserRestore, Description: "平台后台恢复已删除租户用户", Resource: "tenant_user", Action: "restore"},
		{Name: "永久删除租户用户", Code: AdminTenantUserPermanentDelete, Description: "平台后台永久删除租户用户", Resource: "tenant_user", Action: "permanent_delete"},
		{Name: "批量启用/禁用租户用户", Code: AdminTenantUserBatchStatus, Description: "平台后台批量启用/禁用租户用户", Resource: "tenant_user", Action: "batch_status"},
		{Name: "批量删除租户用户", Code: AdminTenantUserBatchDelete, Description: "平台后台批量删除租户用户", Resource: "tenant_user", Action: "batch_delete"},
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