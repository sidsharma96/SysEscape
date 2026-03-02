-- migrations/0004_add_system_admin_user.up.sql
INSERT INTO users (id, github_id, github_username, display_name, role)
VALUES (
    '00000000-0000-0000-0000-000000000000',
    0,
    'system-roomctl',
    'System (roomctl)',
    'ADMIN'
)
ON CONFLICT (id) DO NOTHING;
