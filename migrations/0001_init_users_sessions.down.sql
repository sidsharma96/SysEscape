-- 0001_init_users_sessions.down.sql
-- Reverse of 0001_init_users_sessions.up.sql (drop in reverse order).

DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
