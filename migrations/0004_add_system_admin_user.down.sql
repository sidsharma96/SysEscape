-- migrations/0004_add_system_admin_user.down.sql
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000000';
