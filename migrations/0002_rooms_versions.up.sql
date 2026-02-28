-- migrations/0002_rooms_versions.up.sql
CREATE TABLE rooms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT UNIQUE NOT NULL,
    title       TEXT NOT NULL,
    district    TEXT NOT NULL,
    engine      TEXT NOT NULL CHECK (engine IN ('A', 'B')),
    difficulty  TEXT NOT NULL CHECK (difficulty IN ('L0','L1','L2','L3')),
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE room_versions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id          UUID NOT NULL REFERENCES rooms(id),
    version_number   INT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'PUBLISHED'
                     CHECK (status IN ('PUBLISHED','DEPRECATED','DISABLED')),
    bundle_hash      TEXT,          -- sha256 of published bundle (nullable until M2)
    changelog        TEXT NOT NULL DEFAULT '',
    published_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (room_id, version_number)
);

CREATE INDEX idx_room_versions_room_id ON room_versions(room_id);
CREATE INDEX idx_room_versions_status ON room_versions(status);
