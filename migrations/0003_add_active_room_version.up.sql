-- migrations/0003_add_active_room_version.up.sql
ALTER TABLE rooms
    ADD COLUMN active_room_version_id UUID
    REFERENCES room_versions(id);
