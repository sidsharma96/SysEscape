# M3 Engine A Vertical Slice — Research

> Generated 2026-03-01. Research only — no implementation plan.

---

## 1. What Exists Today

### 1.1 Current DB Tables

Four migrations exist (0001–0004):

| Migration | Tables / Changes |
|-----------|-----------------|
| `0001_init_users_sessions` | `users`, `sessions`, `idempotency_keys` |
| `0002_rooms_versions` | `rooms`, `room_versions` |
| `0003_add_active_room_version` | `ALTER TABLE rooms ADD COLUMN active_room_version_id` |
| `0004_add_system_admin_user` | Inserts system-roomctl admin user (UUID zero) |

**No `runs` or `run_actions` tables exist.** These are entirely new for M3.
The PROJECT_STRUCTURE.md mentions `0003_runs_actions` in its recommended layout,
but that numbering is aspirational — 0003 and 0004 are already taken by M2 work.
The next migration will be `0005_runs_actions`.

### 1.2 Current GraphQL Schema

**Queries:**
- `viewer` — returns authenticated user info (userId, role, githubUsername)
- `rooms(engine, difficulty, district)` — catalog listing with filters
- `roomBySlug(slug)` — single room lookup

**Mutations:**
- `publishRoomVersion(input: PublishRoomVersionInput!)` — admin-only publishing

**Types:** `Room`, `RoomVersion`, `Viewer`, enums (`RoomEngine`, `RoomDifficulty`, `RoomVersionStatus`)

**Not yet in schema (needed for M3):**
- `startRun` mutation (returns `runId` + `runToken`)
- `Run` type, `RunStatus` enum
- Any run-related queries

### 1.3 Current Middleware Chain & Auth Flow

HTTP mux in `cmd/graphql-bff/main.go`:
```
POST /graphql → AdminAPIKeyMiddleware → SessionMiddleware → gqlgen handler
GET  /auth/github/login    → HandleGitHubLogin
GET  /auth/github/callback → HandleGitHubCallback
POST /auth/logout          → HandleLogout
GET  /healthz              → 200 OK
```

**Auth flow:**
1. `AdminAPIKeyMiddleware` — checks `Authorization: Bearer <key>`, injects synthetic ADMIN user
2. `SessionMiddleware` — extracts session cookie → validates → puts `*models.User` in context
3. Resolvers call `UserFromContext(ctx)` to get authenticated user
4. ADMIN checks are resolver-level (check `user.Role == "ADMIN"`)

**No token minting or verification exists.** No JWT library in go.mod.
The `internal/token/` package from PROJECT_STRUCTURE.md does not exist yet.

### 1.4 Resolver & Wiring Pattern

- **Resolver struct** (`internal/graphql/resolvers/resolver.go`): holds `CatalogRepo` and `PublishService`
- **Resolver methods** are thin: extract args → call service/repo → map `pkg/models` types to `generated.*` types
- **Mapping functions** (`mapRoom`, `mapRoomVersion`) convert between domain and GraphQL types
- `gqlgen.yml`: no autobind — all types generated into `internal/graphql/generated/models_gen.go`
- Adding new mutations: edit `schema.graphql` → `make gqlgen` → implement generated stubs

### 1.5 What's in the cache-stampede Room Bundle

The published room at `rooms/cache-stampede/` contains:

**`metadata.yaml`:**
- slug: `cache-stampede`, title: "Cache Stampede", engine: A, difficulty: L1
- district: `distributed-systems`, estimated 15 minutes
- Tags: caching, thundering-herd, database, performance

**`engineA/scenario.yaml`:**
- Topology: 3 nodes — `app-server` (service), `cache` (redis), `database` (postgres)
- Initial metrics: `request_latency_ms: 45`, `db_connections: 20`, `cache_hit_rate: 0.97`, `error_rate: 0.001`
- Config: `cache_ttl_seconds: 60`

**`engineA/actions.yaml`:** 4 actions:
1. `enable_singleflight` — Coalesce concurrent cache misses per key
2. `add_jitter_to_ttl` — Spread expirations to avoid synchronized misses
3. `enable_stale_while_revalidate` — Serve stale cache while refreshing in background
4. `increase_cache_ttl` — Extend cache TTL to reduce database pressure

**`engineA/signals.yaml`:**
- Metrics: `request_latency`, `db_connections`, `cache_hit_rate`, `error_rate`
- Log patterns: "cache miss burst", "db pool exhausted"
- Trace spans: "GET /api/items", "cache.lookup", "db.query"

**`engineA/win_checks.yaml`:** 3 checks (all must pass):
- `cache_hit_rate > 0.95`
- `db_connections < 80`
- `error_rate < 0.01`

**Assessment:** The room YAML is a real definition, not a dummy. It describes topology,
initial state, actions, observable signals, and win conditions. It does NOT yet describe:
- **State transition rules:** How each action changes the metrics over time
- **Tick/simulation model:** How metrics evolve between actions
- **Failure scenario trigger:** When/how the stampede starts

These gaps are addressed by the new `simulation.yaml` file (see §3.3 for the resolved format).
The `simulation.yaml` adds the tick model, event timeline, and action effects — all as
lerp-toward-target with per-tick rates. This file will be created as part of PR4.

### 1.6 WebSocket-Related Code & Dependencies

**Go backend:**
- `gorilla/websocket v1.5.0` is in go.mod as an **indirect dependency** (pulled in by gqlgen)
- No WS handler code exists anywhere in `internal/`
- `internal/ws/` package (from PROJECT_STRUCTURE.md) does not exist
- `cmd/engine-a/main.go` is a bare stub: prints "engine-a starting..." and exits

**Frontend:**
- Vite config already proxies `/ws` to `ws://localhost:8081` — ready for Engine A
- `EngineAPage.tsx` is a placeholder (shows runId from URL params)
- `web/src/hooks/use-ws.ts` and `web/src/lib/ws/` do not exist
- No xterm.js, no WS client library in package.json
- `uuid` package already installed (for clientRequestId)

### 1.7 Existing Platform Infrastructure

**Docker Compose services available:**
- Postgres 16 (port 5432)
- Kafka 3.7 KRaft mode (port 9092)
- Redis 7 (port 6379)
- MinIO (ports 9000/9001)

**Internal platform packages:**
- `internal/platform/log` — structured JSON logger via slog
- `internal/platform/db` — `DBTX` interface (pgxpool.Pool or pgx.Tx)
- `internal/platform/storage` — `BundleStore` interface + S3 implementation (Upload, Download, Exists)
- `internal/platform/publish` — publish service with idempotency

---

## 2. Key Constraints (from ARCHITECTURE.md & DECISIONS.md)

### 2.1 WS Protocol Envelope

From ARCHITECTURE.md §7 and docs/AGENTS/ui.md:

```
Client → Server:
  { protocolVersion: 1, type: "hello", runId, payload: { runToken, resumeFromSeq? } }
  { protocolVersion: 1, type: "apply_action", runId, payload: { actionKey, clientRequestId, expectedSeq } }
  { type: "pong" }

Server → Client:
  { type: "hello_ack", payload: { snapshotRequired: bool } }
  { type: "snapshot", seq: N, payload: { ...fullState } }
  { type: "delta", seq: N, payload: { ops: [...] } }
  { type: "action_accepted", payload: { actionKey, seq } }
  { type: "win_update", payload: { ...winState } }
  { type: "ping" }
```

- Heartbeat: 25s ping interval, 75s timeout
- Reconnect: exponential backoff 500ms → 1s → 2s → 4s → 8s cap
- Resume: client sends `resumeFromSeq`, server either resumes delta stream or sends full snapshot

### 2.2 Token Types — runToken

From ARCHITECTURE.md §8:
- **Issued by:** Backend (BFF's `startRun` mutation)
- **Audience:** Engine A WS handler
- **TTL:** 10–30 minutes
- **Scope:** Single `runId` + `userId` + engine type
- **Format:** JWT

Claims structure (inferred):
```json
{
  "sub": "<userId>",
  "runId": "<runId>",
  "engine": "A",
  "exp": <unix_timestamp>,
  "iat": <unix_timestamp>
}
```

### 2.3 Engine A Determinism Requirements

From ARCHITECTURE.md §3.5 and §6:
- "Engine A determinism: Runs are replayable. Pinned to `room_version_id`. Engine A uses deterministic seed + action log. `(seed + action_log)` → identical outcome."
- State is reconstructable from `(seed + action_log)`
- WS reconnect rebuilds state from the action log, not from in-memory caches

### 2.4 Action Log: Ordered, Monotonic Seq

From ARCHITECTURE.md §6 (Engine A data flow):
- Actions are applied sequentially and produce deltas with monotonic `seq` numbers
- If a seq gap is detected by the client, it must request a full snapshot
- The action log is the single source of truth for replay

### 2.5 Idempotent Action Application

From ADR-009 and ARCHITECTURE.md §3.4:
- All player-facing mutations use `clientRequestId` (UUID v4)
- `apply_action` WS messages include `clientRequestId`
- Same action with same `clientRequestId` must be a no-op (not double-applied)
- WS action dedup for player actions should be enforced in `run_actions` with a partial
  unique index on `(run_id, client_request_id)` where `client_request_id IS NOT NULL`.
- Virtual tick entries intentionally have `client_request_id = NULL` and are excluded from dedup.

### 2.6 Resume Model: Ring Buffer of Recent Deltas

From docs/AGENTS/ui.md and PROJECT_STRUCTURE.md (`internal/ws/resume.go`):
- Server keeps a ring buffer of recent deltas per run
- On reconnect with `resumeFromSeq`, server replays deltas from the buffer
- If the requested seq is too old (fallen off the ring buffer), send a full snapshot instead

---

## 3. Design Decisions (Resolved)

All open questions have been answered by the human lead. These are binding decisions.

### 3.1 Engine A: Separate Binary (Option A)

Engine A runs as its own binary (`cmd/engine-a/`) on :8081.
- Matches ARCHITECTURE.md diagram, Vite proxy config, Makefile targets, and existing stub.
- Requires shared token verification with BFF via `RUN_TOKEN_SECRET` env var.
- Engine A has its own Postgres connection (reads `runs`, writes `run_actions`)
  and its own S3/BundleStore access (downloads room bundles directly).

### 3.2 WS Library: gorilla/websocket

Use `gorilla/websocket` (already in go.mod as indirect dependency via gqlgen).
- Battle-tested, good ping/pong support, extensive documentation.
- Archived but stable — no breaking changes expected.
- Promotes it from indirect to direct dependency; no new dep added to go.mod.

### 3.3 Simulation Definition: Direct Metric Effects + Tick Model

**Format:** Option A (direct metric effects) with timed events for the failure scenario.

A new file `engineA/simulation.yaml` defines the tick model, failure events, and action effects.
The simulation engine uses **lerp-toward-target** for all metric changes: each tick, for
each active effect (from events + applied actions), lerp each metric toward its target at
the given rate. Effects from later actions override earlier event effects on the same metric.

```yaml
# engineA/simulation.yaml
simulation:
  tick_interval_ms: 1000    # sim advances every 1s
  duration_ticks: 300       # 5 minute room

  # The "problem" that unfolds without intervention
  events:
    - at_tick: 0
      description: "Cache TTLs begin expiring simultaneously"
      effects:
        - metric: cache_hit_rate
          target: 0.30
          rate: 0.05          # per tick, lerp toward target
        - metric: db_connections
          target: 180
          rate: 3
        - metric: error_rate
          target: 0.15
          rate: 0.008

  # What each player action does
  action_effects:
    enable_singleflight:
      effects:
        - metric: db_connections
          target: 40
          rate: 2
        - metric: cache_hit_rate
          target: 0.85
          rate: 0.03
    add_jitter_to_ttl:
      effects:
        - metric: cache_hit_rate
          target: 0.93
          rate: 0.02
    enable_stale_while_revalidate:
      effects:
        - metric: error_rate
          target: 0.005
          rate: 0.01
        - metric: request_latency_ms
          target: 50
          rate: 1.5
    increase_cache_ttl:
      effects:
        - metric: cache_hit_rate
          target: 0.97
          rate: 0.01
        - metric: db_connections
          target: 25
          rate: 1
```

**Key design properties:**
- State at any tick is deterministic given `(seed + action_log + tick_count)`.
- Ticks are virtual (sim-tick counter), not wall-clock — preserves replay invariant.
- The `events` array creates the problem the player must diagnose and fix.
- Action effects override event effects on the same metric (later source wins).
- Trivially authorable by M7 room-authoring agents — every room is just numbers in YAML.
- `simulation.yaml` is additive — it doesn't replace `scenario.yaml`, `actions.yaml`,
  `signals.yaml`, or `win_checks.yaml`. It adds the "how" to the existing "what".

### 3.4 startRun: Lives in the BFF

`startRun` is a GraphQL mutation in the BFF. The BFF creates the `runs` row in Postgres,
generates a deterministic seed, mints a `runToken` JWT, and returns both `runId` and
`runToken` to the client.

Engine A has its own Postgres connection:
- Verifies `runToken` → extracts `runId`
- Reads the `runs` row to get `room_version_id` and `seed`
- Reads the `room_versions` row to get `bundle_hash`
- Downloads the bundle from S3 via BundleStore
- Parses simulation YAML → starts sim loop
- Writes to `run_actions` as actions are applied

### 3.5 Token Signing: Shared HMAC (HS256)

Shared secret via `RUN_TOKEN_SECRET` env var, configured in both BFF and Engine A.
- BFF mints with `jwt.SigningMethodHS256`
- Engine A verifies with the same key
- Adequate for single-VPS v1 deployment
- Simple key management: one env var, no PKI

### 3.6 Frontend Scope: Full Panel Set for M3

All Engine A UI components are in scope for M3 (they are not covered by any later milestone).
Split across two frontend PRs:

**PR7a (core gameplay):**
- WS connection hook (`use-ws.ts`) with reconnect + heartbeat
- Engine A state hook (`use-engine-a.ts`) with snapshot/delta handling
- MetricsPanel (renders metric values with signal colors)
- ActionBar (buttons for available actions)
- WinOverlay (shows win state when conditions are met)
- `startRun` mutation call from RoomDetailPage "Start" button

**PR7b (full panel set):**
- TopologyMap (renders topology nodes + connections)
- LogsPanel (renders log stream from simulation)
- TimerBar (shows elapsed time / remaining ticks)
- Reconnecting toast UI

---

## 4. Likely Footguns & Integration Points

### 4.1 BFF ↔ Engine A Token Handoff

- New `internal/token/` package: JWT mint (BFF) + verify (Engine A) with shared HMAC key.
- **No JWT library in go.mod yet** — add `github.com/golang-jwt/jwt/v5`.
- Both services read `RUN_TOKEN_SECRET` from env. If they mismatch, WS `hello` silently fails.
- **Dev ergonomics:** Follow the "zero flags, zero env vars" pattern from M2 (same as S3
  credentials defaulting to `minioadmin`). Set `RUN_TOKEN_SECRET=dev-run-token-secret-not-for-production`
  in `.env.local` during PR1. Both services should validate on startup with a clear error
  message if the var is missing (production) but work out of the box for local dev.

### 4.2 Engine A Bundle Loading

Engine A downloads room bundles directly from S3 via `BundleStore` (platform trust zone).
- Engine A gets `bundle_hash` from the `runs` → `room_versions` join → calls `BundleStore.Download(hash)`
- Re-uses existing `internal/platform/storage` package — same S3 config env vars as BFF.
- **Footgun:** Bundle is a tar archive. Need to extract and parse 5 YAML files
  (`metadata.yaml`, `scenario.yaml`, `actions.yaml`, `signals.yaml`, `win_checks.yaml`,
  `simulation.yaml`). The tar extraction + YAML parsing should be a shared utility
  (future rooms will need the same logic).
- **Caching:** Consider an in-memory LRU cache keyed by `bundle_hash` so repeated runs
  of the same room version don't re-download from S3 on every WS connect.

### 4.3 Simulation Engine: Tick + Lerp Model

The simulation engine is a tight loop with well-defined semantics:

1. **Initialize:** Parse `scenario.yaml` → set initial metric values. Parse `simulation.yaml`
   → load events and action effect definitions.
2. **Each tick:** For each metric, collect all active effects (events whose `at_tick` has passed
   + actions the player has applied). Later effects override earlier ones on the same metric.
   Lerp each metric toward its target at the given rate.
3. **Action apply:** Record in action log, activate the action's effects, generate delta.
4. **Win check:** After each tick, evaluate `win_checks.yaml`. If all pass → run complete.
5. **Delta generation:** Diff previous and current metric values → emit delta to WS.

**Determinism invariant:** `(seed + action_log + tick_count) → identical state`.
- Ticks are sim-tick counter, NOT wall-clock time.
- **Tick advancement model (real-time play vs fast-forward replay):**
  During live play, the server advances ticks on a real timer (every `tick_interval_ms`).
  Each tick is recorded in the action log as a virtual `tick` entry (not a player action,
  but part of the log). On reconnect/replay, ticks are replayed from the log without
  real-time delays — the engine fast-forwards through all recorded ticks to rebuild state,
  then resumes real-time ticking from the current tick number.
  This is the standard game server pattern: real-time during play, fast-forward during replay.
  The PR4 sim engine must support both modes (real-time tick with timer vs batch replay from log).
  The PR5 WS handler calls the replay path on `hello` (to rebuild state for the connecting
  client) and then switches to the real-time tick loop for ongoing play.
- **Footgun:** Floating-point arithmetic is deterministic on a single platform but may
  differ cross-platform. For M3 (single-VPS), this is fine. If multi-node replay is
  needed later, consider fixed-point math or rounding to N decimal places per tick.

### 4.4 WS Upgrade Through Reverse Proxy / Gateway

- In local dev, Vite proxy handles `/ws` → `ws://localhost:8081` (already configured).
- In production, Envoy Gateway routes `/ws/engineA → Engine A`.
- **For M3 (local dev):** Not a concern. Vite proxy works out of the box.

### 4.5 Seq Monotonicity & Connection Exclusivity

- Single-player rooms (v1) mean one WS connection per run — no true concurrency.
- But reconnect can cause brief overlap: old connection hasn't closed, new one connects.
- **Solution:** One active WS connection per runId. When a new `hello` arrives for a runId
  that already has an active connection, close the old connection with a "replaced" close code.
- A mutex per run (or per-run goroutine with channel) serializes tick advancement + action application.

### 4.6 Room Definition: simulation.yaml is Additive

The new `simulation.yaml` file is additive — it does not replace `scenario.yaml`, `actions.yaml`,
`signals.yaml`, or `win_checks.yaml`. Each existing file retains its purpose:
- `scenario.yaml` — topology + initial metric values
- `actions.yaml` — available actions with descriptions (for UI display)
- `signals.yaml` — panel definitions (which metrics/logs/traces to show)
- `win_checks.yaml` — win conditions
- `simulation.yaml` (new) — tick model + event timeline + action effects

**Footgun:** Action keys in `simulation.yaml` `action_effects` must exactly match keys
in `actions.yaml`. A typo means the action has no simulation effect. `roomctl validate`
should cross-reference these.

### 4.7 `pkg/models` Types Needed

Currently `pkg/models/` has: `User`, `Session`, `Room`, `RoomVersion`, `RoomWithLatestVersion`.

For M3, we need at minimum:
```go
// pkg/models/run.go
type Run struct {
    ID              uuid.UUID
    UserID          uuid.UUID
    RoomVersionID   uuid.UUID
    Seed            int64
    Status          RunStatus   // "ACTIVE", "COMPLETED", "ABANDONED"
    StartedAt       time.Time
    CompletedAt     *time.Time
}

type RunStatus string
const (
    RunStatusActive    RunStatus = "ACTIVE"
    RunStatusCompleted RunStatus = "COMPLETED"
    RunStatusAbandoned RunStatus = "ABANDONED"
)

type RunAction struct {
    ID              uuid.UUID
    RunID           uuid.UUID
    Seq             int
    ActionType      string     // "player" or "tick"
    ActionKey       *string    // nil for ticks
    ClientRequestID *string    // nil for ticks
    AppliedAt       time.Time
}
```

---

## 5. Estimated Scope

### 5.1 New DB Migrations

**`0005_runs_actions.up.sql`:**
```sql
CREATE TABLE runs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id),
    room_version_id   UUID NOT NULL REFERENCES room_versions(id),
    seed              BIGINT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'ACTIVE'
                      CHECK (status IN ('ACTIVE','COMPLETED','ABANDONED')),
    started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at      TIMESTAMPTZ
);

CREATE INDEX idx_runs_user_id_started_at ON runs(user_id, started_at);

CREATE TABLE run_actions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id            UUID NOT NULL REFERENCES runs(id),
    seq               INT NOT NULL,
    action_type       TEXT NOT NULL DEFAULT 'player'
                      CHECK (action_type IN ('player','tick')),
    action_key        TEXT,
    client_request_id TEXT,
    applied_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (run_id, seq),
    CHECK (
      (action_type = 'player' AND action_key IS NOT NULL AND client_request_id IS NOT NULL) OR
      (action_type = 'tick'   AND action_key IS NULL     AND client_request_id IS NULL)
    )
);

CREATE UNIQUE INDEX idx_run_actions_dedup
  ON run_actions(run_id, client_request_id)
  WHERE client_request_id IS NOT NULL;

CREATE INDEX idx_run_actions_run_id ON run_actions(run_id);
```

**Index rationale:**
- `idx_runs_user_id_started_at` — covers "show my recent runs" queries (run history page).
  Composite `(user_id, started_at)` per TDD §6.2. Add now while the table is empty.
- `idx_run_actions_run_id` — covers "replay all actions for this run" query that Engine A
  hits on every WS connect (reconnect rebuilds state from action log). The UNIQUE constraint
  on `(run_id, seq)` and partial dedup index on `(run_id, client_request_id)` (non-null only)
  help point lookups but not a
  bare `WHERE run_id = $1 ORDER BY seq` scan.

### 5.2 New or Modified Modules

| Module | Status | Work |
|--------|--------|------|
| `pkg/models/run.go` | **New** | `Run`, `RunStatus`, `RunAction` types |
| `internal/token/` | **New** | JWT mint + verify for runToken (HS256 HMAC, claims types) |
| `internal/engine/a/repo/` | **New** | RunRepo (create run, append action, get by ID, list actions) |
| `internal/engine/a/service/` | **New** | Sim engine (tick+lerp), action applicator, win checker, bundle parser |
| `internal/engine/a/transport/` | **New** | WS handler (gorilla/websocket: upgrade, hello, action, delta stream) |
| `internal/ws/` | **New** | Shared WS infra: protocol envelope, heartbeat, resume ring buffer |
| `cmd/engine-a/main.go` | **Modify** | Real wiring: HTTP+WS server on :8081, DB pool, S3, config |
| `internal/graphql/schema.graphql` | **Modify** | Add `startRun` mutation, `Run` type, `RunStatus` enum |
| `internal/graphql/resolvers/` | **Modify** | `startRun` resolver (creates run, mints token, returns both) |
| `internal/graphql/resolvers/resolver.go` | **Modify** | Add `RunService` + `TokenMinter` to Resolver struct |
| `cmd/graphql-bff/main.go` | **Modify** | Wire RunService, token minter, `RUN_TOKEN_SECRET` config |
| `rooms/cache-stampede/engineA/simulation.yaml` | **New** | Tick model + event timeline + action effects |
| `web/src/lib/ws/` | **New** | WS protocol types, client, heartbeat |
| `web/src/hooks/use-ws.ts` | **New** | WS connection hook with reconnect + resume |
| `web/src/hooks/use-engine-a.ts` | **New** | Engine A state machine (snapshot/delta) |
| `web/src/lib/graphql/mutations.ts` | **New** | `startRun` mutation |
| `web/src/components/engine-a/` | **New** | MetricsPanel, ActionBar, WinOverlay, TopologyMap, LogsPanel, TimerBar |
| `web/src/routes/EngineAPage.tsx` | **Modify** | Replace placeholder with real gameplay layout |
| `web/src/routes/RoomDetailPage.tsx` | **Modify** | Add "Start Run" button calling `startRun` mutation |

### 5.3 Approximate LOC & Suggested PR Split

**Total estimated new code:** ~3,000–4,000 LOC (Go + TypeScript + SQL + YAML)

```
PR1 (token) ──────────┐
                       ├── PR3 (startRun) ─── PR6 (wiring) ─── PR7a ─── PR7b
PR2 (run DB + models) ─┤         ╱
                       └── PR4 (sim engine) ─── PR5 (WS infra)
```

| PR | Scope | Est. LOC | Dependencies |
|----|-------|----------|-------------|
| **PR1: Token package** | `internal/token/` (mint, verify, claims), go.mod update (add `golang-jwt/jwt/v5`), unit tests | ~250 | None |
| **PR2: Run DB + models** | Migration 0005, `pkg/models/run.go`, `internal/engine/a/repo/` (create/get/append), repo tests | ~350 | None |
| **PR3: startRun mutation** | Schema update, gqlgen regen, `startRun` resolver, run service (create + mint token), BFF wiring, `RUN_TOKEN_SECRET` in `.env.local` | ~400 | PR1, PR2 |
| **PR4: Sim engine + room definition** | `internal/engine/a/service/` (sim.go, action.go, wincheck.go), bundle parser (tar extract + YAML parse), `rooms/cache-stampede/engineA/simulation.yaml`, unit tests with golden scenarios | ~700 | PR2 |
| **PR5: WS infrastructure + Engine A transport** | `internal/ws/` (protocol, heartbeat, resume ring buffer), `internal/engine/a/transport/` (gorilla/websocket handler), promote gorilla to direct dep | ~500 | PR1, PR4 |
| **PR6: Engine A service wiring** | `cmd/engine-a/main.go` (real HTTP+WS server), config parsing, S3 bundle loading + LRU cache, integration test (WS connect + apply action + verify delta) | ~350 | PR3, PR4, PR5 |
| **PR7a: Frontend core gameplay** | `lib/ws/` (protocol.ts, client.ts, heartbeat.ts), `hooks/use-ws.ts`, `hooks/use-engine-a.ts`, `components/engine-a/` (MetricsPanel, ActionBar, WinOverlay), `EngineAPage.tsx`, `RoomDetailPage.tsx` (Start Run button), `lib/graphql/mutations.ts` | ~600 | PR3, PR6 |
| **PR7b: Frontend full panel set** | `components/engine-a/` (TopologyMap, LogsPanel, TimerBar), reconnecting toast UI | ~400 | PR7a |

**Notes on ordering:**
- **PR1 and PR2 are independent** — start both simultaneously.
  PR1 defines token types; PR2 defines run models + migration. Neither references the other.
- PR3 is the first PR that needs both (resolver mints a token and creates a run row).
- PR4 (sim engine + room def) depends only on PR2 (uses `pkg/models` run types).
  Can start as soon as PR2 lands, in parallel with PR3.
- PR5 (WS infra) depends on PR1 (token verify in hello handler) and PR4 (sim engine to call into).
- PR6 wires everything together — depends on backend PRs 3, 4, 5.
- PR7a (frontend core) can start WS client code in parallel with PR5/PR6,
  but needs PR6 merged to integration-test against a real WS endpoint.
- PR7b (full panels) is a follow-on that only needs PR7a.

### 5.4 New Dependencies

| Dependency | Purpose | Notes |
|------------|---------|-------|
| `github.com/golang-jwt/jwt/v5` | JWT mint/verify for runToken (HS256) | Standard JWT library for Go |
| `github.com/gorilla/websocket` | WebSocket server for Engine A | Already indirect dep via gqlgen; promote to direct |

### 5.5 Config / Env Vars Needed

| Var | Service | Purpose | Default |
|-----|---------|---------|---------|
| `RUN_TOKEN_SECRET` | BFF + Engine A | Shared HMAC signing key for runToken JWTs | `dev-run-token-secret-not-for-production` (in `.env.local`) |
| `RUN_TOKEN_TTL` | BFF | TTL for minted runTokens | `30m` |
| `ENGINE_A_PORT` | Engine A | HTTP/WS listen port | `8081` |
| `DATABASE_URL` | Engine A | Postgres connection for runs/actions | (same as BFF) |
| `S3_ENDPOINT` | Engine A | S3-compatible endpoint for bundle download | `http://localhost:9000` |
| `S3_BUCKET` | Engine A | Bundle storage bucket | `ser-bundles` |
| `S3_ACCESS_KEY` | Engine A | S3 access key | `minioadmin` |
| `S3_SECRET_KEY` | Engine A | S3 secret key | `minioadmin` |
| `S3_REGION` | Engine A | S3 region | `us-east-1` |
| `S3_FORCE_PATH_STYLE` | Engine A | Path-style S3 addressing (MinIO) | `true` |

---

## 6. Summary of Key Findings

1. **No run infrastructure exists.** Tables, models, token package, and WS code are all greenfield.
2. **Room content is real but incomplete.** cache-stampede has topology, actions, signals, and
   win conditions. The missing piece — simulation tick model + event timeline + action effects —
   will be added as `simulation.yaml` (additive, not replacing existing files).
3. **Simulation engine is simple.** Lerp-toward-target per tick, with event effects creating the
   problem and action effects providing the fix. Deterministic: `(seed + action_log + tick_count) → state`.
4. **gorilla/websocket** is the WS library. Already an indirect dep; promote to direct.
5. **Engine A is a separate binary** on :8081. Has its own Postgres + S3 access.
6. **Shared HMAC (HS256)** for runToken. `RUN_TOKEN_SECRET` env var in both BFF and Engine A.
7. **Frontend shell is ready** — routing, lazy loading, Vite proxy all configured.
8. **Full panel set in M3** — MetricsPanel, ActionBar, WinOverlay, TopologyMap, LogsPanel, TimerBar.
9. **8 PRs, ~3,000–4,000 LOC total.** Critical path: PR1→PR2→PR3→PR6→PR7a→PR7b.
   PR4 and PR5 can parallelize with PR3.

---

## 7. PR8 Research: TopologyMap Click-to-Filter + Design System Polish

> Generated 2026-03-03. Research only — no implementation plan.

PR8 has two sub-goals:
- **(A)** Clickable topology nodes that filter MetricsPanel + LogsPanel to signals relevant to the selected node
- **(B)** JetBrains Mono font loading, signal color thresholds on metric values

### 7.1 Current State (Post-PR7b)

#### signals.yaml Schema — Flat, No Cross-references

`rooms/cache-stampede/engineA/signals.yaml`:
```yaml
metrics:
  - request_latency
  - db_connections
  - cache_hit_rate
  - error_rate
logPatterns:
  - "cache miss burst"
  - "db pool exhausted"
traceSpans:
  - "GET /api/items"
  - "cache.lookup"
  - "db.query"
```

Go structs — both `roomctl/schema.go:29-33` and `bundle_loader.go:37-41`:
```go
type BundleSignals struct {
    Metrics     []string `yaml:"metrics"`
    LogPatterns []string `yaml:"logPatterns"`
    TraceSpans  []string `yaml:"traceSpans"`
}
```

No node associations, no thresholds, no structure beyond flat string lists.

#### Topology — In scenario.yaml, Untyped

`rooms/cache-stampede/engineA/scenario.yaml:1-6`:
```yaml
topology:
  - name: app-server
    type: service
  - name: cache
    type: redis
  - name: database
    type: postgres
```

Go struct is `[]map[string]any` (`bundle_loader.go:23`, `roomctl/schema.go:15`) — completely untyped.

#### Engine A Component Tree (Post-PR7b)

```
EngineAPage.tsx:11-46            (route, composes everything)
├── TimerBar.tsx:8-35            (progress bar, signal-color thresholds on timer %)
├── TopologyMap.tsx:8-24         (horizontal badge list with → arrows, NO click, NO connections)
├── MetricsPanel.tsx:12-28       (plain key/value table, NO signal colors on values)
├── LogsPanel.tsx:8-47           (auto-scrolling log list with tick prefix)
├── ActionBar.tsx:9-31           (action buttons, applied state)
├── WinOverlay.tsx:7-17          (full-screen overlay on win)
└── ReconnectingToast.tsx:8-18   (fixed toast for reconnecting/disconnected)
```

All use `React.memo`. Layout: TimerBar → TopologyMap → (MetricsPanel | LogsPanel) 2-col grid → ActionBar → overlays.

#### Tailwind Theme Tokens — All Match ui.md

`web/src/styles/globals.css` uses Tailwind v4 `@theme` directives:

| Token | Value | Matches ui.md? |
|-------|-------|---------------|
| `--color-panel-bg` | `#1e1e3f` | Yes |
| `--color-panel-border` | `#2a2a5a` | Yes |
| `--color-panel-hover` | `#2e2e6a` | Yes |
| `--color-signal-ok` | `#73bf69` | Yes |
| `--color-signal-warn` | `#ff9830` | Yes |
| `--color-signal-crit` | `#f2495c` | Yes |
| `--color-surface-dark` | `#0d0d1a` | Yes |
| `--color-surface-mid` | `#1a1a2e` | Yes |
| `--color-surface-light` | `#2a2a4a` | Yes |
| `--font-mono` | `"JetBrains Mono", "Fira Code", monospace` | Yes |

No `tailwind.config.ts` exists — correct for Tailwind v4 CSS-first approach.

#### Font Loading — JetBrains Mono NOT Loaded

The `--font-mono` CSS variable references `"JetBrains Mono"` but:
- No `@font-face` declarations in `globals.css`
- No font files in `web/public/` (empty)
- No Google Fonts `<link>` in `index.html`
- No fontsource import

Browser falls through to `"Fira Code"` → system `monospace`.

#### Go Snapshot Struct — Missing Topology/Signals

`internal/engine/a/service/types.go:54-59`:
```go
type Snapshot struct {
    Tick    int                `json:"tick"`
    Won     bool               `json:"won"`
    Metrics map[string]float64 `json:"metrics"`
    Actions []string           `json:"actions,omitempty"`
}
```

No `topology`, `logs`, `totalTicks`, or `signals`. The frontend `SnapshotPayload` (`protocol.ts:41-50`) expects these fields optionally, but the backend never sends them.

#### Runtime Data Path Gap

`cmd/engine-a/runtime.go`:

- `Connect()` (line 71): `payload, _ := json.Marshal(s.engine.Snapshot())` — serializes only `{tick, won, metrics, actions}`
- `runState` struct (line 36): stores engine, lastSeq, tick timing — NOT topology or signals
- `state()` (line 220): loads bundle via `LoadEngineABundleFromTar(raw)` — `bundle.Scenario.Topology` and `bundle.Signals` are available but **discarded** after engine creation
- `tickLoop()` (line 184) and `ApplyAction()` (line 104): also serialize only `Engine.Snapshot()` for deltas

### 7.2 Key Constraints

| Source | Constraint |
|--------|-----------|
| ADR-005 | Room content immutable per version. Topology is static room metadata. |
| ARCHITECTURE.md §3.5 | Engine A determinism: `(seed + action_log)` → identical outcome. Topology unaffected by actions. |
| ARCHITECTURE.md §7 | "Map click filters signals" — non-negotiable for Engine A. |
| ADR-012 | Tailwind v4 CSS-first. Use `@theme` in globals.css, not tailwind.config.ts. |
| PROJECT_STRUCTURE.md | `signals.yaml` intended to define "Panel definitions (metrics, logs, topology nodes)". |
| AGENTS/ui.md | TopologyMap listed as key component. Click-to-filter is the architectural intent. |
| AGENTS/ui.md | "Don't test Tailwind class names" — test behavior, not styling. |

### 7.3 Footguns, Edge Cases, Integration Points

#### 7.3a. Topology Node ↔ Signal Cross-referencing

Current signals.yaml has **no concept of "which metric belongs to which node."** For click-to-filter, metrics/logPatterns need a `node` field. The Go `BundleSignals` struct and `roomctl validate` need updates.

#### 7.3b. Topology Data: WS Snapshot vs Static Load

Two delivery options:
1. **Via WS initial snapshot** (current frontend expectation): Runtime includes topology + signals in the connect snapshot payload. Simple, frontend already expects it.
2. **Separate GraphQL endpoint**: Cleaner separation but requires extra fetch + hook rework.

Frontend `use-engine-a.ts:60` already reads `payload.topology`. Option 1 avoids rework.

#### 7.3c. Font Loading

| Approach | Pros | Cons |
|----------|------|------|
| **Fontsource npm** (`@fontsource/jetbrains-mono`) | Self-hosted via bundler, no CDN dep | Extra npm package |
| **Self-hosted woff2** in `web/public/fonts/` | Full control | Manual file management |
| **Google Fonts CDN** | Zero config | External dep, GDPR |

Fontsource aligns with "no external CDN" philosophy (ADR-011).

#### 7.3d. Signal Color Threshold Semantics

Only TimerBar currently uses signal colors (`TimerBar.tsx:18`). MetricsPanel renders all values in `text-gray-100`.

For per-metric thresholds, direction is inferred from comparing `ok` and `warn`:
- `ok < warn` → lower is better (latency, connections, error_rate): `<=ok` green, `ok..warn` amber, `>warn` red
- `ok > warn` → higher is better (hit_rate): `>=ok` green, `warn..ok` amber, `<warn` red

No explicit `direction` field needed.

#### 7.3e. roomctl validate Impact

`roomctl/schema.go` + `validate.go` parse signals.yaml. Schema changes require:
- Updated Go structs in `roomctl/schema.go` and `bundle_loader.go`
- Updated test fixtures in `validate_test.go` and `bundle_loader_test.go`

#### 7.3f. Go Runtime Enrichment Strategy

The Go `Engine.Snapshot()` method should NOT change (stays dynamic-only). Instead:
- `runState` stores `topology` and `signals` from the bundle at init
- `Connect()` builds an enriched payload merging `Engine.Snapshot()` + static topology + signals
- `tickLoop()` and `ApplyAction()` continue serializing plain `Engine.Snapshot()` (no topology/signals in deltas)

#### 7.3g. Log Filtering by Node

Logs in the WS stream are `{tick, message}` — no explicit node field. Filtering requires matching log messages against `logPatterns` associated with the selected node via the `signals.logPatterns[].node` field.

### 7.4 Open Questions (Answered)

| Question | Answer |
|----------|--------|
| Interactive SVG vs clickable badges for M3? | **(b) Clickable badges** — defer SVG to M8 |
| Signal color threshold strategy? | **(a) Per-metric thresholds** in signals.yaml (ok/warn, crit implicit) |
| Topology node ↔ signal cross-referencing? | **(a) Add `node` field** to each metric/logPattern in signals.yaml |
| Font loading approach? | **(a) Fontsource** npm package (@fontsource/jetbrains-mono) |
| Topology data delivery? | **(c) Initial WS snapshot only**, not in deltas |
| Test room? | **cache-stampede** (3 nodes, 4 metrics sufficient) |
