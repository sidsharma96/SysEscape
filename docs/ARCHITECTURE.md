# Architecture — Systems Escape Rooms

> **Version:** v1.0 (aligned with TDD v0.9)
> **Last updated:** 2026-02-24

## 1. System Overview

Systems Escape Rooms is a web-first learning platform that teaches distributed systems
and security through replayable, production-like incident simulations. Users solve "rooms"
by diagnosing signals (metrics, logs, traces, topologies) and applying fixes.

### Two Gameplay Engines

- **Engine A (Simulator):** Grafana-like panels + topology map + action controls.
  Fast, deterministic, replayable. State is reconstructable from `(seed + action_log)`.
- **Engine B (Workspace + Judge):** Ephemeral sandbox workspace + server-side judge
  with hidden tests. Players write/fix real code; judge grades submissions.

### Architecture Diagram (text)

```
┌──────────────────────────────────────────────────────────────────┐
│                         EDGE (Cloudflare)                        │
│                   DNS + TLS + CDN + Rate Limiting                │
└──────────────────────────┬───────────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│                     PLATFORM (k3s / host)                        │
│                                                                  │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │  Web UI  │  │ GraphQL BFF  │  │  Engine A    │               │
│  │(Vite SPA)│  │ (control     │  │  Service     │               │
│  │ :3000    │  │  plane)      │  │ (sim + WS)   │               │
│  └────┬─────┘  └──────┬───────┘  └──────┬───────┘               │
│       │               │                  │                        │
│       │ GraphQL (:8080)│   WS: /ws/engineA/{runId}               │
│       └───────────────▶│◀────────────────┘                        │
│  ┌────────────────────▼──────────────────▼──────┐               │
│  │              Postgres (system of record)       │               │
│  │  rooms, versions, runs, progress, audit_log   │               │
│  └───────────────────────────────────────────────┘               │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  Engine B    │  │    Judge     │  │    Kafka     │           │
│  │ Orchestrator │─▶│  Dispatcher  │◀─│ (event bus)  │           │
│  │ (pods + WS)  │  │ (K8s Jobs)  │  │              │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │ Bundle Proxy │  │Artifact Proxy│  │    Redis     │           │
│  │ (hash→blob)  │  │(upload/dl)   │  │  (cache/     │           │
│  │              │  │              │  │   sessions)  │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└──────────────────────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────────┐
│                    SANDBOX (untrusted zone)                       │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐                              │
│  │  Workspace   │  │  Judge Job   │  Default-deny egress.        │
│  │  Pod         │  │  (K8s Job)   │  No S3 creds.               │
│  │              │  │              │  No platform secrets.        │
│  └──────────────┘  └──────────────┘                              │
└──────────────────────────────────────────────────────────────────┘
```

## 2. Trust Zones

| Zone | Assumption | Controls |
|------|-----------|----------|
| **Edge** (Cloudflare) | Internet-facing; hostile traffic possible | TLS, CDN caching, edge rate limit, WAF (optional) |
| **Platform** (k3s + host) | Trusted code we deploy; strong authz required | JWT/session auth, RBAC, audit logs, secrets mgmt |
| **Sandbox** (workspace + judge) | Untrusted user-controlled code | Default-deny egress, non-root, seccomp/AppArmor, resource limits, no S3 creds |

## 3. Key Invariants (non-negotiable)

These are system-wide rules that must never be violated. CI enforces them.

1. **Immutable room versions:** Published room content is immutable. Any change requires a new `room_version` with new bundle hash(es).
2. **Sandbox isolation:** Sandbox workloads cannot directly access object storage credentials. They use bundle-proxy and artifact-proxy with scoped, short-lived tokens.
3. **No hidden test leakage:** Workspace bundle and judge bundle are separate. Workspace never receives hidden tests. `roomctl validate` enforces this.
4. **Idempotent mutations:** All player-facing mutations use `clientRequestId`. Retries with the same ID return the stored response; different fingerprint returns CONFLICT.
5. **Engine A determinism:** Runs are replayable. Pinned to `room_version_id`. Engine A uses deterministic seed + action log. `(seed + action_log)` → identical outcome.

## 4. Layering Rules

Code follows a strict layered architecture. Dependencies flow downward only.

```
┌─────────────────────────────────────┐
│  cmd/          (entrypoints)        │  ← wires dependencies; no business logic
├─────────────────────────────────────┤
│  internal/*    (business logic)     │  ← services, repos, domain logic
│    transport/  (HTTP/WS/GraphQL)    │  ← input validation happens HERE
│    service/    (business rules)     │  ← orchestration, use cases
│    repo/       (data access)        │  ← Postgres queries, Kafka producers
├─────────────────────────────────────┤
│  pkg/models    (domain types)       │  ← shared types; no imports from internal/
│  pkg/api       (generated schemas)  │  ← GraphQL types; imports models only
└─────────────────────────────────────┘
```

**Rules:**
- `transport/` → `service/` → `repo/` — no skipping layers.
- All external inputs validated/parsed at transport boundary. No implicit parsing deeper.
- Network calls go through adapter interfaces (mockable in tests).
- `internal/platform/` provides shared infra: logging, tracing, config, errors.

## 5. Service Responsibilities

| Service | Responsibilities | State | Failure Behavior |
|---------|-----------------|-------|-----------------|
| **GraphQL BFF** | Auth, catalog, run lifecycle, token minting, admin publishing | Postgres | Degrades to read-only if DB read-only; retries safe via idempotency |
| **Engine A** | Sim runtime, apply actions, stream snapshot/delta over WS, win checks | Postgres (runs/actions) + in-memory sim | WS reconnect resume; can rebuild state from action log |
| **Engine B Orchestrator** | Create workspace pods, bridge terminal/exec WS, enqueue judge submissions | Postgres + Kafka | Backpressure via queue; cap concurrent workspaces/judges |
| **Judge Dispatcher** | Consume judge commands; start K8s Jobs; persist verdict + artifacts | Kafka + Postgres | At-least-once; idempotent per submissionId |
| **Bundle Proxy** | Serve bundles by hash with verification + disk LRU cache | Disk cache | Cache miss fetches from S3; verification failure blocks serving |
| **Artifact Proxy** | Upload/download artifacts with scoped tokens | Disk staging (optional) | Reject invalid tokens; retryable uploads |

## 6. Data Flow

### Engine A (happy path)
```
User → GraphQL: startRun(roomSlug, clientRequestId)
     → Backend mints runToken (JWT, scoped to runId)
     → User opens WS: /ws/engineA/{runId} with runToken
     → Engine A streams snapshot + deltas
     → User applies action (via WS)
     → Engine A updates sim state, checks win conditions
     → On win: submitProof → generates Atlas card
```

### Engine B (happy path)
```
User → GraphQL: startRun(roomSlug, clientRequestId)
     → Orchestrator creates workspace pod (downloads workspace bundle via bundle-proxy)
     → User opens WS: /ws/engineB/{runId} — terminal session
     → User codes fix, runs: submit
     → Orchestrator: submitToJudge → Kafka judge.commands
     → Judge Dispatcher: creates K8s Job
     → Judge: downloads judge bundle, runs hidden tests
     → Judge: uploads artifacts via artifact-proxy
     → Judge: publishes verdict to Kafka judge.results
     → Backend: persists verdict, streams result to user via WS
```

### Room Publishing (admin)
```
Admin → roomctl validate (schema + leak check)
      → roomctl build (creates content-addressed bundles)
      → GraphQL: publishRoomVersion(clientRequestId, bundleRefs, hashes)
      → Backend: inserts room_version row, uploads bundles to S3
      → Room becomes available in catalog
```

## 7. Frontend Architecture (`web/`)

The web UI is a **Vite + React** single-page application (TypeScript / React Router / Tailwind CSS).
It communicates with the backend exclusively via GraphQL (control plane) and WebSocket (real-time streaming).
It produces static files deployable behind any CDN or simple HTTP server — no Node.js runtime required in production.

### Five UI Surfaces

| Surface | Route | Data Source | Rendering |
|---------|-------|-------------|-----------|
| **Room Catalog** | `/`, `/rooms/:slug` | GraphQL queries | Fetch on mount, cached |
| **Engine A Gameplay** | `/play/:runId/engine-a` | WS: `wss://.../ws/engineA/{runId}` | WS snapshot + delta stream |
| **Engine B Gameplay** | `/play/:runId/engine-b` | WS: `wss://.../ws/engineB/{runId}` | xterm.js terminal + judge results |
| **Run Explorer** | `/runs` | GraphQL queries | Fetch on mount |
| **Admin Publishing** | `/admin/publish` | GraphQL mutations | ADMIN-gated form |

### Frontend-to-Backend Communication

```
 Vite SPA (web/)
 ┌──────────────────────────────┐
 │  Route components            │──GraphQL fetch──▶ GraphQL BFF (:8080)
 │  (catalog, run explorer)     │                   (session cookie auth)
 │                              │
 │  Gameplay components         │──WebSocket──────▶ Engine A (:8081)
 │  (Engine A panels,           │  hello(runToken,   /ws/engineA/{runId}
 │   Engine B terminal)         │   resumeFromSeq)
 │                              │
 │                              │──WebSocket──────▶ Engine B (:8082)
 │                              │  hello(runToken)   /ws/engineB/{runId}
 │                              │
 │  GraphQL mutations           │──POST + CSRF────▶ GraphQL BFF (:8080)
 │  (startRun, submit, publish) │  clientRequestId   (idempotent)
 └──────────────────────────────┘
```

### WS Reconnection Model

Both Engine A and Engine B WS connections support resume-by-seq:

1. Client sends `hello` with `runToken` and optional `resumeFromSeq`
2. Server responds with `hello_ack` including `snapshotRequired` flag
3. If snapshot required: server sends full state snapshot (client's seq was too stale)
4. Otherwise: server resumes delta stream from `resumeFromSeq + 1`
5. Client handles `ping` / `pong` heartbeat (25s interval, 75s timeout)
6. On disconnect: exponential backoff reconnect (500ms → 1s → 2s → 4s → 8s cap)

### Engine A Client State Rule

The UI **never mutates simulation state locally**. All state arrives from the server as
snapshots and deltas. The only client→server message is `apply_action` (with `clientRequestId`
for idempotency and `expectedSeq` for optimistic concurrency).

### Styling

Dark theme (Grafana-inspired) with Tailwind CSS custom tokens. Engine A panels use monospace
fonts, signal colors (green/amber/red), and panel-card layouts.

## 8. Token Types

| Token | Issued By | Audience | Typical TTL | Scope |
|-------|-----------|----------|-------------|-------|
| Session cookie | Auth service | Backend | Hours-days | User session |
| runToken (JWT) | Backend | Engine A/B WS | 10–30 min | Single runId + userId + engine |
| bundleToken (JWT) | Orchestrator | Bundle proxy | 5–15 min | Single bundle hash + purpose (workspace/judge/engineA) |
| artifactToken (JWT) | Orchestrator/Judge | Artifact proxy | 5–15 min | Single artifactId; upload or download |

## 9. Eventing (Kafka)

| Topic | Purpose | Partition Key | Retention |
|-------|---------|--------------|-----------|
| `judge.commands` | Queue submissions to judge dispatcher | submissionId | 7–14d |
| `judge.results` | Publish verdicts back to platform | submissionId | 7–30d |
| `platform.events` | Room published, run completed, etc. | entityId | 7–14d |

## 10. Allowed Dependencies (Module Boundary Map)

```
graphql-bff    → auth, catalog, publish, engine/a (start/proof), engine/b (start/submit), token, idempotency
engine-a       → token (verify runToken), models, platform
engine-b       → token (verify runToken, mint bundleToken), models, platform, Kafka producer
judge          → token (verify bundleToken), models, platform, Kafka consumer, artifact (client)
bundle-proxy   → token (verify bundleToken), platform
artifact-proxy → token (verify artifactToken), platform
roomctl        → publish (validate/build logic), models
web/ (Vite SPA) → GraphQL BFF (via HTTP), Engine A/B (via WS) — NO Go imports
```

No service imports another service's internal packages directly. Communication is via:
- GraphQL (client → BFF)
- WebSocket (client → Engine A/B)
- Kafka (Engine B → Judge)
- HTTP (sandbox → proxies, with scoped tokens)

The `web/` directory is a separate TypeScript project. It shares types with the backend
only through GraphQL codegen (`make ui-codegen`), never through Go package imports.

## 11. Deployment (v1)

- **Single VPS** with k3s (control plane + sandbox).
- **Stateful services on host:** Postgres, Kafka, Redis (not in k3s).
- **Container images:** GitHub Container Registry (GHCR).
- **Room bundles + artifacts:** S3 (content-addressed keys).
- **Secrets:** Kubernetes Secrets + sealed-secrets (git-safe). Sandbox namespace does NOT mount platform secrets.
- **Web UI:** Vite produces static files (`web/dist/`). In dev: `make ui-dev` on :3000 with proxy to backend :8080. In production: serve from Cloudflare CDN or an nginx container on the VPS. No Node.js runtime needed.
- **Network:** Only 80/443 exposed to internet. Postgres/Kafka/Redis listen on node-private interface only.
