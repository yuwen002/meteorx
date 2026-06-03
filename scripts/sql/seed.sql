-- Admin user with password: 123456
INSERT INTO users (id, tenant_id, username, password, nickname, email, role, status, is_master, created_at, updated_at) VALUES
    ('admin-id-000001', 'SYSTEM_ROOT', 'admin', '$2a$10$4cBedBBsEeKToxEw1Jh7iucFnuIStSm2eku7XuBhrZfr13w34x/nO', 'Administrator', 'admin@example.com', 'superadmin', 1, true, '2026-05-20 10:40:48', '2026-05-20 10:40:48');