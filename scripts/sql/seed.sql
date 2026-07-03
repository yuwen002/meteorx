-- ============================================
-- 系统初始化数据
-- 执行时机：数据库首次创建时
-- ============================================

-- --------------------------------------------
-- 1. 初始化系统管理员用户
-- 密码: 123456
-- --------------------------------------------
INSERT INTO users (id, tenant_id, username, password, nickname, email, status, is_master, created_at, updated_at) VALUES
    ('admin-id-000001', 'SYSTEM_ROOT', 'admin', '$2a$10$4cBedBBsEeKToxEw1Jh7iucFnuIStSm2eku7XuBhrZfr13w34x/nO', 'Administrator', 'admin@example.com', 1, true, '2026-05-20 10:40:48', '2026-05-20 10:40:48');

-- --------------------------------------------
-- 2. 初始化角色
-- --------------------------------------------
-- 系统级角色（用于系统管理员）
INSERT INTO roles (id, name, code, description, tenant_id, is_system, scope, status, created_at, updated_at) VALUES
    ('role-superadmin-001', '超级管理员', 'superadmin', '平台超级管理员，拥有所有权限', 'SYSTEM_ROOT', true, 'system', 1, '2026-05-20 10:40:48', '2026-05-20 10:40:48'),
    ('role-system-admin-001', '系统管理员', 'system_admin', '系统管理员，管理系统级资源', 'SYSTEM_ROOT', true, 'system', 1, '2026-05-20 10:40:48', '2026-05-20 10:40:48');

-- 租户级角色（用于普通租户用户）
INSERT INTO roles (id, name, code, description, tenant_id, is_system, scope, status, created_at, updated_at) VALUES
    ('role-tenant-admin-001', '租户管理员', 'tenant_admin', '租户级管理员，管理本租户资源', 'SYSTEM_ROOT', true, 'tenant', 1, '2026-05-20 10:40:48', '2026-05-20 10:40:48'),
    ('role-user-000000001', '普通用户', 'user', '普通用户，拥有基本操作权限', 'SYSTEM_ROOT', true, 'tenant', 1, '2026-05-20 10:40:48', '2026-05-20 10:40:48');

-- --------------------------------------------
-- 3. 初始化用户-角色关联
-- --------------------------------------------
-- 将 admin 用户关联到 superadmin 角色
INSERT INTO user_roles (user_id, role_id, created_at) VALUES
    ('admin-id-000001', 'role-superadmin-001', '2026-05-20 10:40:48');