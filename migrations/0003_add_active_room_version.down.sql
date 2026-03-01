-- migrations/0003_add_active_room_version.down.sql
ALTER TABLE rooms
    DROP COLUMN active_room_version_id;
