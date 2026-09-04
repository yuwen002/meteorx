package rbac

import (
	"context"
	"meteorx/internal/modules/rbac/handler"
	"meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/rbac/service"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/pkg/logger"
)

// InitModule 初始化 RBAC 模块，按顺序做以下 3 步：
//
// 1. SeedPermissions：把 permissions.go 中预定义的权限插入 permissions 表（幂等）
// 2. SeedRolePermissions：给 superadmin 角色绑定所有权限（幂等）
// 3. SeedUserRoles：把 admin 用户绑定到 superadmin 角色（幂等）
// 4. 注册所有角色/权限/用户-角色相关的 HTTP 路由
func InitModule(r chi.Router, db *gorm.DB) {
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	rolePermRepo := repository.NewRolePermissionRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)

	svc := service.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo)

	// ====== Step 1: 注册预定义权限（基于 code 做幂等） ======
	permInserted, permTotal, err := SeedPermissions(context.Background(), svc)
	if err != nil {
		logger.Errorf("[RBAC] 步骤1-权限初始化失败: %v", err)
	} else {
		logger.Infof("[RBAC] 步骤1-权限初始化完成: 新增 %d / 共 %d 个权限", permInserted, permTotal)
	}

	// ====== Step 2: 给 superadmin 角色绑定所有权限（幂等） ======
	permBound, err := SeedRolePermissions(context.Background(), db)
	if err != nil {
		logger.Errorf("[RBAC] 步骤2-角色权限绑定失败: %v", err)
	} else if permBound > 0 {
		logger.Infof("[RBAC] 步骤2-角色权限绑定完成: 为 superadmin 新增 %d 个权限绑定", permBound)
	}

	// ====== Step 3: 给 admin 用户绑定 superadmin 角色（幂等） ======
	roleBound, err := SeedUserRoles(context.Background(), db)
	if err != nil {
		logger.Errorf("[RBAC] 步骤3-用户角色绑定失败: %v", err)
	} else if roleBound > 0 {
		logger.Infof("[RBAC] 步骤3-用户角色绑定完成: 为 admin 新增 %d 个角色绑定", roleBound)
	}

	// ====== Step 4: 注册路由 ======
	h := handler.NewRBACHandler(svc)
	RegisterRoutes(r, h, svc)
}

// SeedRolePermissions 给 superadmin 角色绑定所有权限（幂等：已存在则跳过）
// 返回新绑定的权限数量。
// 约定：
//   - 通过 roles.code = 'superadmin' 找到超级管理员角色
//   - 从 permissions 表读取所有权限的 id
//   - 逐个检查 role_permissions 表是否已有绑定，没有才插入
func SeedRolePermissions(ctx context.Context, db *gorm.DB) (int, error) {
	// 1. 找 superadmin 角色
	var roleID string
	err := db.WithContext(ctx).Table("roles").
		Where("code = ?", "superadmin").
		Select("id").
		Scan(&roleID).Error
	if err != nil {
		return 0, err
	}
	if roleID == "" {
		// roles 表中还没有 superadmin 角色，跳过（seed.sql 可能还没跑）
		logger.Warnf("[RBAC] 步骤2-跳过：roles 表中不存在 code='superadmin' 的角色（seed.sql 可能还未执行）")
		return 0, nil
	}

	// 2. 获取所有权限的 id
	var allPermIDs []string
	err = db.WithContext(ctx).Table("permissions").
		Where("status = 1").
		Select("id").
		Pluck("id", &allPermIDs).Error
	if err != nil {
		return 0, err
	}
	if len(allPermIDs) == 0 {
		return 0, nil
	}

	// 3. 找出已绑定的权限
	var boundPermIDs []string
	err = db.WithContext(ctx).Table("role_permissions").
		Where("role_id = ?", roleID).
		Select("permission_id").
		Pluck("permission_id", &boundPermIDs).Error
	if err != nil {
		return 0, err
	}

	boundSet := make(map[string]bool, len(boundPermIDs))
	for _, pid := range boundPermIDs {
		boundSet[pid] = true
	}

	// 4. 找出还没绑定的权限并插入
	now := time.Now()
	inserted := 0
	for _, pid := range allPermIDs {
		if boundSet[pid] {
			continue
		}
		err = db.WithContext(ctx).Table("role_permissions").
			Create(map[string]interface{}{
				"role_id":       roleID,
				"permission_id": pid,
				"created_at":    now,
			}).Error
		if err != nil {
			return inserted, err
		}
		inserted++
	}

	return inserted, nil
}

// SeedUserRoles 给 admin 用户绑定 superadmin 角色（幂等：已存在则跳过）
// 返回新绑定的角色数量。
// 约定：
//   - 通过 users.username = 'admin' AND users.is_master = true 找到超级管理员用户
//   - 通过 roles.code = 'superadmin' 找到超级管理员角色
//   - 检查 user_roles 表是否已有关联，没有才插入
func SeedUserRoles(ctx context.Context, db *gorm.DB) (int, error) {
	// 1. 找 admin 用户
	var userID string
	err := db.WithContext(ctx).Table("users").
		Where("username = ? AND is_master = ?", "admin", true).
		Select("id").
		Scan(&userID).Error
	if err != nil {
		return 0, err
	}
	if userID == "" {
		logger.Warnf("[RBAC] 步骤3-跳过：users 表中不存在 username='admin' 且 is_master=1 的用户（seed.sql 可能还未执行）")
		return 0, nil
	}

	// 2. 找 superadmin 角色
	var roleID string
	err = db.WithContext(ctx).Table("roles").
		Where("code = ?", "superadmin").
		Select("id").
		Scan(&roleID).Error
	if err != nil {
		return 0, err
	}
	if roleID == "" {
		logger.Warnf("[RBAC] 步骤3-跳过：roles 表中不存在 code='superadmin' 的角色（seed.sql 可能还未执行）")
		return 0, nil
	}

	// 3. 检查是否已经绑定
	var count int64
	err = db.WithContext(ctx).Table("user_roles").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	if count > 0 {
		return 0, nil // 已存在，跳过
	}

	// 4. 插入关联
	err = db.WithContext(ctx).Table("user_roles").
		Create(map[string]interface{}{
			"user_id":    userID,
			"role_id":    roleID,
			"created_at": time.Now(),
		}).Error
	if err != nil {
		return 0, err
	}

	return 1, nil
}
