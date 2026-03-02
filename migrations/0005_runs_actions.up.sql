-- migrations/0005_runs_actions.up.sql
CREATE TABLE runs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id),
    room_version_id UUID        NOT NULL REFERENCES room_versions(id),
    seed            BIGINT      NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'ACTIVE'
                    CHECK (status IN ('ACTIVE', 'COMPLETED', 'ABANDONED')),
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_runs_user_id_started_at ON runs(user_id, started_at);

CREATE TABLE run_actions (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id            UUID        NOT NULL REFERENCES runs(id),
    seq               INT         NOT NULL,
    action_type       TEXT        NOT NULL DEFAULT 'player'
                      CHECK (action_type IN ('player', 'tick')),
    action_key        TEXT,
    client_request_id TEXT,
    applied_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (run_id, seq),
    CHECK (
        (action_type = 'player' AND action_key IS NOT NULL AND client_request_id IS NOT NULL) OR
        (action_type = 'tick' AND action_key IS NULL AND client_request_id IS NULL)
    )
);

CREATE UNIQUE INDEX idx_run_actions_dedup
    ON run_actions(run_id, client_request_id)
    WHERE client_request_id IS NOT NULL;

CREATE INDEX idx_run_actions_run_id ON run_actions(run_id);
