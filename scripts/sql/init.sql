-- ============================================================
-- MeteorX 初始化建表脚本
-- 说明：MySQL 8.0+ / utf8mb4，按依赖顺序依次建表
-- 使用：mysql -u your_user -p your_db < scripts/sql/init.sql
-- ============================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ------------------------------------------------------------
-- 1. 租户表（tenants）
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `tenants`;
CREATE TABLE `tenants` (
    `id`           VARCHAR(26)  NOT NULL         COMMENT '租户唯一标识',
    `name`         VARCHAR(100) DEFAULT NULL      COMMENT '租户名称',
    `domain`       VARCHAR(255) NOT NULL         COMMENT '租户域名，全局唯一',
    `status`       INT          DEFAULT 1        COMMENT '状态：1-启用 0-禁用',
    `description`  VARCHAR(255) DEFAULT NULL      COMMENT '租户简述',
    `contact_email` VARCHAR(100) DEFAULT NULL     COMMENT '联系邮箱',
    `region`       VARCHAR(50)  DEFAULT NULL      COMMENT '地区/数据中心',
    `logo`         VARCHAR(500) DEFAULT NULL      COMMENT '租户 Logo URL',
    `extra`        TEXT         DEFAULT NULL      COMMENT '扩展字段（JSON 格式）',
    `created_at`   DATETIME     DEFAULT NULL      COMMENT '创建时间',
    `updated_at`   DATETIME     DEFAULT NULL      COMMENT '更新时间',
    `deleted_at`   DATETIME     DEFAULT NULL      COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenants_domain` (`domain`),
    KEY `idx_tenants_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户表';

-- ------------------------------------------------------------
-- 2. 用户表（users）
--   注意：users 表没有 role 字段。
--        角色通过 user_roles 关联表管理（多对多）。
--        超级管理员（is_master = 1）通过 contextx 判定。
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
    `id`         VARCHAR(26)  NOT NULL                  COMMENT '用户ID',
    `tenant_id`  VARCHAR(26)  NOT NULL                  COMMENT '租户ID',
    `username`   VARCHAR(50)  NOT NULL                  COMMENT '用户名',
    `password`   VARCHAR(255) NOT NULL                  COMMENT '密码（BCrypt 哈希）',
    `nickname`   VARCHAR(50)  DEFAULT NULL              COMMENT '昵称',
    `email`      VARCHAR(100) DEFAULT NULL              COMMENT '邮箱',
    `status`     INT          DEFAULT 1                 COMMENT '状态：1-启用 0-禁用',
    `is_master`  TINYINT(1)   DEFAULT 0                 COMMENT '是否为主管理员（1=是，0=否）',
    `created_at` DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME     DEFAULT NULL              COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_tenant_username` (`tenant_id`, `username`),
    KEY `idx_users_tenant_id` (`tenant_id`),
    KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户表';

-- ------------------------------------------------------------
-- 3. 角色表（roles）
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `roles`;
CREATE TABLE `roles` (
    `id`          VARCHAR(26)  NOT NULL                  COMMENT '角色ID',
    `name`        VARCHAR(50)  NOT NULL                  COMMENT '角色名称',
    `code`        VARCHAR(50)  NOT NULL                  COMMENT '角色编码',
    `description` VARCHAR(255) DEFAULT NULL              COMMENT '角色描述',
    `tenant_id`   VARCHAR(26)  DEFAULT NULL              COMMENT '租户ID（系统级角色=SYSTEM_ROOT）',
    `is_system`   TINYINT(1)   DEFAULT 0                 COMMENT '是否系统内置（1=是，0=否）',
    `scope`       VARCHAR(20)  DEFAULT 'tenant'          COMMENT '作用域：system/tenant/all',
    `status`      INT          DEFAULT 1                 COMMENT '状态：1-启用 0-禁用',
    `created_at`  DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at`  DATETIME     DEFAULT NULL              COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    KEY `idx_roles_code` (`code`),
    KEY `idx_roles_tenant_id` (`tenant_id`),
    KEY `idx_roles_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色表';

-- ------------------------------------------------------------
-- 4. 权限表（permissions）
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `permissions`;
CREATE TABLE `permissions` (
    `id`          VARCHAR(26)  NOT NULL                  COMMENT '权限ID',
    `name`        VARCHAR(50)  NOT NULL                  COMMENT '权限名称',
    `code`        VARCHAR(100) NOT NULL                  COMMENT '权限编码（唯一，用于中间件校验）',
    `description` VARCHAR(255) DEFAULT NULL              COMMENT '权限描述',
    `resource`    VARCHAR(50)  NOT NULL                  COMMENT '资源类型（user/role/perm/tenant 等）',
    `action`      VARCHAR(50)  NOT NULL                  COMMENT '操作类型（list/create/read/update/delete 等）',
    `status`      INT          DEFAULT 1                 COMMENT '状态：1-启用 0-禁用',
    `created_at`  DATETIME     DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_permissions_code` (`code`),
    KEY `idx_permissions_resource` (`resource`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='权限表';

-- ------------------------------------------------------------
-- 5. 角色-权限关联表（role_permissions）—— 多对多
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `role_permissions`;
CREATE TABLE `role_permissions` (
    `role_id`       VARCHAR(26) NOT NULL COMMENT '角色ID',
    `permission_id` VARCHAR(26) NOT NULL COMMENT '权限ID',
    `created_at`    DATETIME    DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`role_id`, `permission_id`),
    KEY `idx_role_permissions_permission_id` (`permission_id`),
    CONSTRAINT `fk_role_permissions_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_role_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色-权限关联表';

-- ------------------------------------------------------------
-- 6. 用户-角色关联表（user_roles）—— 多对多
-- ------------------------------------------------------------
DROP TABLE IF EXISTS `user_roles`;
CREATE TABLE `user_roles` (
    `user_id`    VARCHAR(26) NOT NULL COMMENT '用户ID',
    `role_id`    VARCHAR(26) NOT NULL COMMENT '角色ID',
    `created_at` DATETIME    DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`user_id`, `role_id`),
    KEY `idx_user_roles_role_id` (`role_id`),
    CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户-角色关联表';

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================================
-- 初始化完成。
--
-- 下一步（首次部署时执行，顺序不能错）：
--
--   ① 跑 scripts/sql/seed.sql
--        → 插入 admin 用户 + 3 个初始角色（superadmin/tenant_admin/user）
--
--   ② 启动 Go 服务：go run main.go
--        → rbac/module.go:SeedPermissions()
--              → 将 permissions.go 中所有权限插入 permissions 表
--        → rbac/module.go:SeedRolePermissions()
--              → 将 permissions 表中所有权限绑定到 superadmin 角色
--        → rbac/module.go:SeedUserRoles()
--              → 将 admin 用户绑定到 superadmin 角色
--
--   ③ 用 admin / 123456 登录（或登录请求中指定）
--        → 此时 admin 通过 user_roles → superadmin → role_permissions
--          拥有所有接口权限
--
-- 说明：以上第②步全部幂等，重复启动不会重复插入数据。
--       新增权限只需在 permissions.go 中补一条，重启服务即可自动入库。
-- ============================================================