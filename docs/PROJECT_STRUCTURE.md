# Recommended Project Structure — Systems Escape Rooms

> **Philosophy:** Structure follows trust boundaries, not feature grouping.
> Backend services share types via `pkg/` but communicate only via network (GraphQL, WS, Kafka, HTTP).
> Frontend is a fully separate workspace that speaks to the backend exclusively via GraphQL and WS.

## Full Tree (annotated)

```
systems-escape-rooms/
│
├── cmd/                              # ── SERVICE ENTRYPOINTS ──
│   │                                 # Each dir is a main.go that wires dependencies.
│   │                                 # No business logic here — just config parsing,
│   │                                 # dependency injection, and server startup.
│   │
│   ├── graphql-bff/                  # GraphQL BFF — control plane API
│   │   └── main.go                   # Starts HTTP server (:8080), mounts GraphQL handler
│   │
│   ├── engine-a/                     # Engine A — simulator runtime
│   │   └── main.go                   # Starts HTTP + WS server (:8081)
│   │
│   ├── engine-b-orchestrator/        # Engine B — workspace pods + judge dispatch
│   │   └── main.go                   # Starts HTTP + WS server (:8082), Kafka producer
│   │
│   ├── judge-dispatcher/             # Judge — Kafka consumer, K8s Job spawner
│   │   └── main.go                   # Consumes judge.commands, creates K8s Jobs
│   │
│   ├── bundle-proxy/                 # Bundle Proxy — serves bundles by hash
│   │   └── main.go                   # HTTP server (:8083), verifies bundleToken + sha256
│   │
│   ├── artifact-proxy/               # Artifact Proxy — upload/download with scoped tokens
│   │   └── main.go                   # HTTP server (:8084), verifies artifactToken
│   │
│   ├── roomctl/                      # CLI tool for room validation + publishing
│   │   └── main.go                   # `roomctl validate`, `roomctl build`
│   │
│   └── migrate/                      # DB migration runner
│       └── main.go                   # `migrate up`, `migrate down`, `migrate create`
│
├── internal/                         # ── PRIVATE GO PACKAGES ──
│   │                                 # The bulk of the codebase. Not importable outside this module.
│   │                                 # Organized by domain (auth, engine, etc.), each with
│   │                                 # three sub-layers: transport → service → repo.
│   │
│   ├── auth/                         # Authentication + authorization
│   │   ├── transport/                # GitHub OAuth callback handler, session middleware
│   │   ├── service/                  # OAuth flow orchestration, RBAC checks
│   │   └── repo/                     # Session store (Redis), user store (Postgres)
│   │
│   ├── catalog/                      # Room catalog (read-only queries)
│   │   ├── transport/                # GraphQL resolvers for rooms, roomBySlug, versions
│   │   ├── service/                  # Catalog business logic (filtering, pagination)
│   │   └── repo/                     # Postgres queries (rooms, room_versions)
│   │
│   ├── engine/
│   │   ├── a/                        # Engine A simulator
│   │   │   ├── transport/            # WS handler (snapshot/delta/action), GraphQL resolvers
│   │   │   ├── service/              # Sim runtime, tick loop, win checks, replay
│   │   │   │   ├── sim.go            # Core simulation engine (deterministic: seed + actions → state)
│   │   │   │   ├── action.go         # Action application (idempotent via clientRequestId)
│   │   │   │   └── wincheck.go       # Win condition evaluator (per room definition)
│   │   │   └── repo/                 # Run store, action log (Postgres)
│   │   │
│   │   └── b/                        # Engine B orchestrator
│   │       ├── transport/            # WS handler (terminal bridge), GraphQL resolvers
│   │       ├── service/              # Workspace lifecycle, judge submission, Kafka producer
│   │       └── repo/                 # Submission store, workspace tracking (Postgres)
│   │
│   ├── judge/                        # Judge dispatcher (Kafka consumer side)
│   │   ├── dispatcher.go             # Consumes judge.commands, spawns K8s Jobs
│   │   ├── result.go                 # Processes judge.results, persists verdicts
│   │   └── repo/                     # Verdict + artifact metadata store (Postgres)
│   │
│   ├── bundle/                       # Bundle proxy logic
│   │   ├── handler.go                # HTTP handler: GET /bundles/{sha256}
│   │   ├── cache.go                  # Disk LRU cache keyed by sha256
│   │   └── verify.go                 # Token verification + hash integrity check
│   │
│   ├── artifact/                     # Artifact proxy logic
│   │   ├── handler.go                # HTTP handler: PUT/GET /artifacts/{artifactId}
│   │   └── verify.go                 # artifactToken verification + scope check
│   │
│   ├── publish/                      # Room publishing pipeline
│   │   ├── transport/                # GraphQL mutation: publishRoomVersion
│   │   ├── service/                  # Validation, bundle hashing, S3 upload orchestration
│   │   └── repo/                     # room_versions insert (Postgres)
│   │
│   ├── ws/                           # Shared WebSocket infrastructure
│   │   ├── server.go                 # WS upgrade, connection registry, routing by runId
│   │   ├── protocol.go              # Message envelope (protocolVersion, type, runId, payload)
│   │   ├── heartbeat.go             # Ping/pong (25s / 75s timeout)
│   │   └── resume.go                # Resume-by-seq: ring buffer, snapshot fallback
│   │
│   ├── idempotency/                  # clientRequestId deduplication
│   │   ├── middleware.go             # HTTP/GraphQL middleware: check before handler, store after
│   │   └── repo/                     # Idempotency key store (Postgres, TTL cleanup)
│   │
│   ├── token/                        # JWT minting + verification
│   │   ├── mint.go                   # Mint runToken, bundleToken, artifactToken
│   │   ├── verify.go                 # Verify + extract claims
│   │   └── claims.go                 # Typed claim structs (scope, purpose, expiry)
│   │
│   └── platform/                     # Cross-cutting concerns (imported by everything)
│       ├── log/                      # Structured JSON logging via slog
│       │   └── logger.go            # Logger factory: requestId, runId context fields
│       ├── trace/                    # OpenTelemetry tracing setup
│       │   └── provider.go          # Trace provider config (Tempo exporter)
│       ├── metrics/                  # Prometheus metrics setup
│       │   └── registry.go          # Custom metrics: active_runs, ws_connections, etc.
│       ├── config/                   # Config loading (env vars, .env.local)
│       │   └── config.go            # Struct-based config with validation
│       ├── errors/                   # Error types and wrapping
│       │   └── errors.go            # Domain errors: NotFound, Conflict, Forbidden, etc.
│       └── health/                   # Health check endpoint (/healthz, /readyz)
│           └── handler.go
│
├── pkg/                              # ── PUBLIC GO PACKAGES ──
│   │                                 # Shared types that define the vocabulary of the system.
│   │                                 # No imports from internal/. Kept deliberately small.
│   │
│   ├── models/                       # Domain types (the nouns of the system)
│   │   ├── room.go                   # Room, RoomVersion, RoomEngine, RoomDifficulty
│   │   ├── run.go                    # Run, RunStatus, RunResult, ActionLogEntry
│   │   ├── submission.go             # Submission, SubmissionStatus, JudgeVerdict
│   │   ├── artifact.go               # Artifact metadata (artifactId, type, size, hash)
│   │   ├── user.go                   # User, Role (USER, ADMIN)
│   │   └── events.go                 # Kafka event envelope (eventType, schemaVersion, payload)
│   │
│   └── api/                          # Generated types (from GraphQL schema)
│       └── generated.go              # Auto-generated by gqlgen — DO NOT EDIT
│
├── migrations/                       # ── POSTGRES MIGRATIONS ──
│   │                                 # Sequential, numbered. Never modify applied migrations.
│   │                                 # Each file: NNNN_description.up.sql + .down.sql
│   │
│   ├── 0001_init_users_sessions.up.sql
│   ├── 0001_init_users_sessions.down.sql
│   ├── 0002_rooms_versions.up.sql
│   ├── 0002_rooms_versions.down.sql
│   ├── 0003_runs_actions.up.sql
│   ├── 0003_runs_actions.down.sql
│   ├── 0004_submissions_artifacts.up.sql
│   ├── 0004_submissions_artifacts.down.sql
│   └── 0005_idempotency_keys.up.sql
│   └── 0005_idempotency_keys.down.sql
│
├── rooms/                            # ── ROOM CONTENT ──
│   │                                 # One directory per room slug. Published as immutable,
│   │                                 # content-addressed bundles. Validated by `roomctl`.
│   │
│   └── <room-slug>/                  # e.g., rooms/cache-thundering-herd/
│       ├── metadata.yaml             # Title, district, difficulty, engine type, description
│       ├── debrief.yaml              # Post-win educational content + links
│       │
│       ├── engineA/                  # (present if Engine A room)
│       │   ├── scenario.yaml         # Initial sim state, seed, tick config
│       │   ├── actions.yaml          # Available player actions + effects
│       │   ├── signals.yaml          # Panel definitions (metrics, logs, topology nodes)
│       │   └── win_checks.yaml       # Win condition rules
│       │
│       └── engineB/                  # (present if Engine B room)
│           ├── workspace/
│           │   ├── Dockerfile         # Workspace container image
│           │   └── starter/           # Starter code given to player
│           └── judge/
│               ├── Dockerfile         # Judge container image
│               ├── hidden_tests/      # Tests player never sees (grading criteria)
│               └── harness/           # Test runner scaffolding
│
├── web/                              # ── FRONTEND (Next.js) ──
│   │                                 # Fully separate TypeScript project. Communicates with
│   │                                 # backend ONLY via GraphQL (:8080) and WebSocket (:8081/:8082).
│   │                                 # Never imports from Go packages.
│   │
│   ├── src/
│   │   ├── app/                      # Next.js App Router (pages + layouts)
│   │   │   ├── layout.tsx            # Root layout: auth provider, theme, nav shell
│   │   │   ├── page.tsx              # Landing → room catalog
│   │   │   ├── login/
│   │   │   │   └── page.tsx          # GitHub OAuth callback
│   │   │   ├── rooms/
│   │   │   │   └── [slug]/
│   │   │   │       └── page.tsx      # Room detail + "Start Run" CTA
│   │   │   ├── play/
│   │   │   │   └── [runId]/
│   │   │   │       ├── page.tsx      # Run router: redirect to engine-a or engine-b
│   │   │   │       ├── engine-a/
│   │   │   │       │   └── page.tsx  # Engine A gameplay (panels + topology + actions)
│   │   │   │       └── engine-b/
│   │   │   │           └── page.tsx  # Engine B gameplay (terminal + submit + judge results)
│   │   │   ├── runs/
│   │   │   │   └── page.tsx          # Run history + progress dashboard
│   │   │   └── admin/
│   │   │       └── publish/
│   │   │           └── page.tsx      # Room publishing (ADMIN only)
│   │   │
│   │   ├── components/               # UI components (organized by surface)
│   │   │   ├── ui/                   # Generic primitives: Button, Card, Modal, Toast, Badge
│   │   │   ├── layout/              # Shell, Nav, Sidebar, Footer
│   │   │   ├── catalog/             # RoomCard, RoomGrid, DifficultyBadge, DistrictFilter
│   │   │   ├── engine-a/            # MetricsPanel, LogsPanel, TopologyMap, ActionBar, TimerBar, WinOverlay
│   │   │   ├── engine-b/            # Terminal, SubmitButton, JudgeStatus, ArtifactViewer, VerdictCard
│   │   │   ├── run-explorer/        # RunList, RunDetail, AtlasCard, TraceViewer
│   │   │   └── auth/                # LoginButton, UserMenu, AuthGuard
│   │   │
│   │   ├── hooks/                    # Custom React hooks
│   │   │   ├── use-ws.ts            # WS connect + reconnect + resume-by-seq + heartbeat
│   │   │   ├── use-engine-a.ts      # Engine A state machine (snapshot → delta → derived panels)
│   │   │   ├── use-engine-b.ts      # Engine B terminal bridge + judge status
│   │   │   ├── use-graphql.ts       # GraphQL client wrapper (urql)
│   │   │   └── use-auth.ts          # Session state, login/logout, role checks
│   │   │
│   │   ├── lib/                      # Pure logic (no React dependencies)
│   │   │   ├── graphql/
│   │   │   │   ├── client.ts        # urql client config (cookie auth, CSRF header)
│   │   │   │   ├── queries.ts       # Catalog, runs, progress queries
│   │   │   │   ├── mutations.ts     # startRun, submitProof, submitToJudge, publishRoomVersion
│   │   │   │   └── types.ts         # ⚡ GENERATED — run `make ui-codegen` — DO NOT HAND-EDIT
│   │   │   ├── ws/
│   │   │   │   ├── protocol.ts      # WS message types: hello, hello_ack, snapshot, delta, action, ping/pong
│   │   │   │   ├── client.ts        # WS connect, reconnect w/ backoff, resume-from-seq logic
│   │   │   │   └── heartbeat.ts     # Ping/pong handler (25s send, 75s receive timeout)
│   │   │   ├── tokens.ts            # runToken in-memory storage (NOT localStorage)
│   │   │   └── idempotency.ts       # clientRequestId generation (UUID v4)
│   │   │
│   │   ├── styles/
│   │   │   └── globals.css          # Tailwind directives + Grafana-dark theme tokens
│   │   │
│   │   └── __tests__/               # Test files (mirrors src/ structure)
│   │       ├── hooks/
│   │       │   ├── use-ws.test.ts
│   │       │   └── use-engine-a.test.ts
│   │       ├── lib/
│   │       │   ├── ws/protocol.test.ts
│   │       │   └── graphql/mutations.test.ts
│   │       └── components/
│   │           ├── engine-a/action-bar.test.tsx
│   │           └── catalog/room-card.test.tsx
│   │
│   ├── public/                       # Static assets (favicons, images)
│   ├── schema.graphql                # 🔗 Copied/synced from backend for codegen
│   ├── codegen.ts                    # GraphQL Codegen config
│   ├── next.config.ts                # Next.js config (rewrites for WS proxy in dev)
│   ├── tailwind.config.ts            # Tailwind theme (Grafana-dark tokens)
│   ├── tsconfig.json                 # TypeScript strict mode
│   ├── vitest.config.ts              # Vitest + React Testing Library
│   ├── .eslintrc.json                # ESLint config
│   ├── .env.local                    # UI env vars (NEXT_PUBLIC_GRAPHQL_URL, NEXT_PUBLIC_WS_HOST)
│   ├── pnpm-lock.yaml
│   └── package.json
│
├── infra/                            # ── INFRASTRUCTURE ──
│   │
│   ├── docker-compose.yaml           # Local dev: Postgres, Kafka (KRaft), Redis, MinIO
│   │
│   ├── k8s/                          # Kubernetes manifests (k3s)
│   │   ├── base/                     # Kustomize base
│   │   │   ├── platform/             # Deployments: graphql-bff, engine-a, engine-b, judge, proxies
│   │   │   │   ├── graphql-bff.yaml
│   │   │   │   ├── engine-a.yaml
│   │   │   │   ├── engine-b.yaml
│   │   │   │   ├── judge-dispatcher.yaml
│   │   │   │   ├── bundle-proxy.yaml
│   │   │   │   └── artifact-proxy.yaml
│   │   │   ├── sandbox/              # Workspace pod template, judge job template
│   │   │   │   ├── workspace-pod.yaml
│   │   │   │   └── judge-job.yaml
│   │   │   └── network/              # NetworkPolicies
│   │   │       ├── default-deny.yaml       # Sandbox namespace: deny all egress
│   │   │       ├── workspace-allow.yaml    # Workspace: allow DNS + bundle-proxy + orchestrator
│   │   │       └── judge-allow.yaml        # Judge: allow DNS + bundle-proxy + artifact-proxy
│   │   │
│   │   └── overlays/                 # Environment-specific overrides
│   │       ├── local/                # Patches for local k3s dev
│   │       └── production/           # Patches for prod (resource limits, replicas, sealed-secrets)
│   │
│   ├── envoy/                        # Envoy Gateway config (WS routing, TLS)
│   │   └── gateway.yaml              # Routes: /graphql → BFF, /ws/engineA → Engine A, /ws/engineB → Engine B
│   │
│   ├── observability/                # LGTM stack config
│   │   ├── grafana/
│   │   │   └── dashboards/           # Pre-built dashboards (active runs, WS connections, judge latency)
│   │   ├── prometheus/
│   │   │   └── prometheus.yml        # Scrape config for all services
│   │   ├── loki/
│   │   │   └── loki-config.yaml      # 7-day retention
│   │   └── tempo/
│   │       └── tempo-config.yaml     # 3-7 day retention
│   │
│   └── scripts/                      # Operational scripts
│       ├── init-minio.sh             # Create S3 bucket in MinIO for local dev
│       ├── sealed-secret.sh          # Helper for creating sealed-secrets
│       └── db-backup.sh              # Nightly Postgres dump (v1 manual backup)
│
├── docs/                             # ── DOCUMENTATION ──
│   │
│   ├── ARCHITECTURE.md               # System overview, trust zones, data flows, module map
│   ├── DECISIONS.md                  # ADR log (ADR-001 through ADR-012+)
│   ├── DEV.md                        # Local dev setup, make targets, troubleshooting
│   │
│   └── AGENTS/                       # Per-module agent guidance (living docs)
│       ├── auth.md
│       ├── engine-a.md
│       ├── engine-b.md
│       ├── proxy.md                  # Bundle + artifact proxies
│       ├── graphql.md
│       ├── rooms.md
│       ├── infra.md
│       └── ui.md                     # Frontend agent guidance (Next.js, WS, components)
│
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md      # Evidence Block + checklists
│   └── workflows/
│       ├── pr.yaml                   # PR gate: lint + unit + contract + roomctl validate
│       ├── main.yaml                 # Main: integration tests + build images + push to GHCR
│       └── nightly.yaml              # Nightly: E2E + containment + load-lite
│
├── testdata/                         # ── SHARED TEST FIXTURES ──
│   │                                 # Golden files, recorded scenarios, fake room content.
│   │                                 # Referenced by both unit and integration tests.
│   │
│   ├── golden/                       # Golden scenario recordings
│   │   ├── engine-a/
│   │   │   └── cache-thundering-herd.json  # Recorded: seed + actions + expected state snapshots
│   │   └── engine-b/
│   │       └── dns-misconfiguration.json   # Recorded: submission + expected verdict
│   │
│   ├── fixtures/
│   │   ├── rooms/                    # Minimal valid room for testing
│   │   │   └── test-room-a/
│   │   │       ├── metadata.yaml
│   │   │       └── engineA/
│   │   ├── ws/                       # Sample WS message sequences for protocol tests
│   │   │   ├── hello-handshake.json
│   │   │   ├── resume-from-seq.json
│   │   │   └── action-accept-delta.json
│   │   └── graphql/                  # Sample GraphQL responses for contract tests
│   │       ├── rooms-query.json
│   │       └── start-run-mutation.json
│   │
│   └── schemas/                      # Schema snapshots for contract testing
│       ├── graphql.snapshot.graphql   # Checked-in schema snapshot (diff = breaking change)
│       ├── ws-envelope.json          # JSON Schema for WS messages
│       └── kafka-envelope.json       # JSON Schema for Kafka events
│
├── Makefile                          # ── COMMAND SURFACE ──
│                                     # Deterministic targets for humans, agents, and CI.
│                                     # If a command isn't here, it shouldn't be run.
│
├── Procfile                          # goreman/overmind: run all Go services at once
├── go.mod                            # Go module definition
├── go.sum
├── .env.local                        # Local dev env vars (safe defaults, committed)
├── .gitignore
├── .editorconfig
├── AGENTS.md                         # ── ROOT AGENT GUIDANCE ──
│                                     # Living document: project structure, layering rules,
│                                     # do/don'ts, per-module pointers, common mistakes.
│
└── README.md                         # Project overview, quick start, links to docs/
```

## Key Structural Decisions (with rationale)

### 1. Monorepo with Go + Next.js colocated

The Go backend and Next.js frontend live in the same repository but are completely independent build systems. Go uses `go.mod` at the root; Next.js uses `pnpm` in `web/`. They share no code — the boundary between them is the network (GraphQL + WS).

**Why monorepo:** You're a solo builder with agents. Having everything in one repo means every agent can see the full picture. Context packs can include both backend and frontend files. CI runs against the whole system. The alternative (polyrepo) would add coordination overhead between repos with no benefit at this scale.

**Why separate build systems (not a JS monorepo tool):** Go's toolchain is self-contained and fast. Mixing it into a turborepo/nx setup would add complexity with no gain. `make` unifies the command surface — `make ci` runs both Go and TypeScript checks.

### 2. `internal/` sub-layering: transport → service → repo

Every domain in `internal/` follows the same three-layer pattern:

```
internal/<domain>/
├── transport/    # HTTP handlers, GraphQL resolvers, WS handlers
│                 # Input validation + deserialization happens HERE
├── service/      # Business logic, orchestration, domain rules
│                 # Pure Go — no HTTP, no SQL — just interfaces + logic
└── repo/         # Data access: Postgres queries, Kafka producers, Redis calls
                  # Returns domain types from pkg/models
```

**Why this matters for agents:** This layering is linter-enforced. An agent can't accidentally put SQL in a handler or HTTP parsing in a service. Each layer has a predictable role, so an agent working on "add a new GraphQL mutation" knows exactly which files to touch: resolver in `transport/`, use-case in `service/`, query in `repo/`.

### 3. `pkg/models` is the shared vocabulary

`pkg/models` contains the domain types that appear everywhere: `Room`, `Run`, `Submission`, `Artifact`, `User`. It has zero imports from `internal/` — it's the foundation layer.

`pkg/api` contains generated types (from gqlgen). It imports `pkg/models` only.

**Why separate from `internal/`:** Multiple services need the same types. `pkg/models` is the single source of truth. If an agent defines a new struct, the decision is simple: if more than one service needs it → `pkg/models`. If it's service-specific → keep it in `internal/<domain>/`.

### 4. `internal/ws/` as shared WebSocket infrastructure

Both Engine A and Engine B use WebSocket with the same protocol envelope, handshake, heartbeat, and resume-by-seq semantics. Rather than duplicating this, `internal/ws/` provides the reusable WS infrastructure. Engine A and Engine B each have their own `transport/` that uses `internal/ws/` for the connection lifecycle but defines engine-specific message handling.

### 5. `rooms/` is content, not code

Room content is declarative YAML. It's validated by `roomctl validate` (schema + leak check) and built into content-addressed bundles by `roomctl build`. The `rooms/` directory is designed for room authors (which may be a separate agent role in the playbook's M7 phase), not for service developers.

The critical invariant: `rooms/<slug>/engineB/workspace/` (what the player sees) must never contain anything from `rooms/<slug>/engineB/judge/hidden_tests/` (what grades them). `roomctl validate` enforces this at the file system level.

### 6. `testdata/` for golden scenarios and contract snapshots

Golden scenarios are recorded play-throughs (seed + action sequence + expected state) that serve as regression tests. They live in `testdata/golden/` and are referenced by integration tests. Contract snapshots (GraphQL schema snapshot, WS/Kafka JSON Schemas) live in `testdata/schemas/` and are diffed on every PR — any change means you're modifying a public interface, which requires a version bump.

**Why not test fixtures inside each `internal/<domain>/` package?** Some fixtures are cross-cutting (a golden scenario exercises auth → engine → judge). Centralizing them in `testdata/` prevents duplication and makes them discoverable by agents.

### 7. `web/src/components/` organized by UI surface, not by component type

Components are grouped by the surface they belong to: `catalog/`, `engine-a/`, `engine-b/`, `run-explorer/`, `auth/`. Generic primitives (`Button`, `Card`, `Modal`) go in `ui/`.

**Why not by type (e.g., `atoms/`, `molecules/`):** The five surfaces are independent feature domains with different data sources and state patterns. An agent working on Engine A should never need to look at Engine B components. Surface-based grouping makes scope boundaries obvious and prevents accidental coupling.

### 8. `web/src/lib/` has no React dependency

`lib/` contains pure TypeScript logic: GraphQL client setup, WS protocol types, connection management, idempotency helpers. It's testable without React Testing Library — just plain Vitest unit tests. Hooks in `hooks/` bridge `lib/` into React state.

This separation means an agent can work on WS reconnection logic (a complex piece) without touching any React code, and test it in isolation.

### 9. `infra/k8s/` uses Kustomize (base + overlays)

K8s manifests use Kustomize with a `base/` for shared definitions and `overlays/` for environment-specific patches (local vs production). The sandbox namespace has its own subdirectory with `NetworkPolicy` manifests that enforce default-deny egress.

**Why Kustomize over Helm:** For a solo builder, Kustomize is simpler — plain YAML with patches. Helm's templating adds a layer of abstraction that makes agent-generated manifests harder to verify. You can always migrate to Helm later if needed.

### 10. Flat `migrations/` with sequential numbering

Migrations are numbered `NNNN_description.{up,down}.sql`. No subdirectories, no framework-specific formats. Sequential numbering means merge conflicts are immediately visible (two PRs can't both add migration `0006` without conflict).

## File Count Estimate

| Directory | Estimated Files | Growth Pattern |
|-----------|----------------|----------------|
| `cmd/` | ~8 (one main.go per service) | Rarely changes after M0 |
| `internal/` | ~60-80 (transport + service + repo per domain) | Grows steadily M1–M5, stabilizes M6+ |
| `pkg/` | ~8-10 (models + generated) | Grows slowly; mostly additive |
| `migrations/` | ~10-15 | One per schema change |
| `rooms/` | ~30-50 (10+ rooms × 3-5 files each) | Big growth at M7 (content buildout) |
| `web/src/` | ~50-70 (pages + components + hooks + lib) | Grows M0–M5, then stabilizes |
| `infra/` | ~20-25 (compose + k8s + observability) | Mostly set by M0, minor growth |
| `docs/` | ~12-15 | Living docs, updated continuously |
| `testdata/` | ~15-20 | Grows with each room + contract change |
| **Total** | **~220-290 files** | |

## Milestone-to-Structure Mapping

Which parts of the tree get built when:

| Milestone | New Directories | Key Files Created |
|-----------|----------------|-------------------|
| **M0** (dev env) | `cmd/migrate/`, `infra/`, `web/` (shell), `docs/`, root files | docker-compose, Makefile, AGENTS.md, layout.tsx, go.mod |
| **M1** (auth + catalog) | `internal/auth/`, `internal/catalog/`, `pkg/models/`, `migrations/0001-0002` | OAuth flow, catalog resolvers, room/user models, basic catalog page |
| **M2** (publishing) | `internal/publish/`, `cmd/roomctl/`, `rooms/test-room-a/` | roomctl validate/build, publishRoomVersion mutation, first test room |
| **M3** (Engine A) | `internal/engine/a/`, `internal/ws/`, `internal/token/`, `web/src/components/engine-a/`, `web/src/hooks/use-ws.ts` | Sim runtime, WS server, delta streaming, Engine A gameplay page |
| **M4** (Engine B) | `internal/engine/b/`, `internal/judge/`, `internal/bundle/`, `internal/artifact/`, `web/src/components/engine-b/` | Workspace lifecycle, judge dispatch, terminal UI, xterm.js integration |
| **M5** (observability) | `infra/observability/`, `web/src/components/run-explorer/` | Grafana dashboards, trace/log integration, run explorer page |
| **M6** (sandbox hardening) | `infra/k8s/base/sandbox/`, `infra/k8s/base/network/` | NetworkPolicies, resource limits, containment tests |
| **M7** (10+ rooms) | `rooms/<slug>/` × 10 | Room YAML content, golden scenarios per room |
| **M8-M10** (polish, deploy, docs) | `infra/k8s/overlays/production/`, `.github/workflows/` | CI pipelines, prod manifests, README |
