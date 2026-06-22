-- Admin user with password: 123456
INSERT INTO users (id, tenant_id, username, password, nickname, email, role, status, is_master, created_at, updated_at) VALUES
    ('admin-id-000001', 'SYSTEM_ROOT', 'admin', '$2a$10$4cBedBBsEeKToxEw1Jh7iucFnuIStSm2eku7XuBhrZfr13w34x/nO', 'Administrator', 'admin@example.com', 'superadmin', 1, true, '2026-05-20 10:40:48', '2026-05-20 10:40:48');

-- Default admin role (matches the superadmin role code used in auth middleware)
INSERT INTO roles (id, name, code, description, tenant_id, is_system, status, created_at, updated_at) VALUES
    ('role-superadmin-001', '超级管理员', 'superadmin', '平台超级管理员，拥有所有权限', 'SYSTEM_ROOT', true, 1, '2026-05-20 10:40:48', '2026-05-20 10:40:48'),
    ('role-tenant-admin-001', '租户管理员', 'tenant_admin', '租户级管理员，管理本租户资源', 'SYSTEM_ROOT', true, 1, '2026-05-20 10:40:48', '2026-05-20 10:40:48');