# Decision Log — Systems Escape Rooms

> **Format:** Lightweight ADRs. One entry per decision.
> **Rule:** If you change a decision, don't delete the old entry — mark it `Superseded` and add a new one.

---

## ADR-001: GraphQL control plane + WebSocket streaming

**Status:** Accepted
**Date:** 2026-02-24

**Context:** We need a control-plane API (catalog, run lifecycle, publishing) and a real-time streaming surface (dashboards, terminal I/O). These have very different performance characteristics.

**Decision:** GraphQL for CRUD/control-plane operations. WebSocket for high-frequency interactive streams (Engine A sim deltas, Engine B terminal). No GraphQL subscriptions in v1.

**Consequences:** Two protocol surfaces to maintain. WS routing must be sticky by `runId`. GraphQL schema is the contract test boundary for the control plane.

---

## ADR-002: Kafka as event backbone

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Judge submission processing is async and must survive pod restarts. We also want replay semantics for future analytics.

**Decision:** Kafka for judge command queue, judge results, and platform events. Topics are treated as versioned APIs with stable `eventType` + `schemaVersion` envelopes.

**Consequences:** Kafka adds operational complexity (ZooKeeper / KRaft). Worth it for replay semantics + decoupling. Consumer groups enable at-least-once delivery; handlers must be idempotent.

---

## ADR-003: k3s single-node + stateful services on host

**Status:** Accepted
**Date:** 2026-02-24

**Context:** v1 targets a single VPS. Running Postgres/Kafka/Redis inside k3s complicates storage and debugging.

**Decision:** Run Postgres, Kafka, and Redis directly on the VPS host. k3s manages platform services and sandbox pods. Stateful services are reachable via the node-private interface.

**Consequences:** Simpler ops for v1. Not horizontally scalable without rearchitecting data services. Acceptable for Season 1 (single VPS, low user count).

---

## ADR-004: No S3 credentials in sandbox — use proxies

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Sandbox pods run untrusted user code (Engine B). Giving them S3 credentials is a data exfiltration risk.

**Decision:** Sandbox pods access bundles and artifacts exclusively through bundle-proxy and artifact-proxy, using scoped, short-lived JWT tokens (`bundleToken`, `artifactToken`).

**Consequences:** Two extra services to build and operate (proxies). But the security boundary is clean: sandbox never holds long-lived credentials. Token scoping prevents cross-user access.

---

## ADR-005: Immutable room versions + content-addressed bundles

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Runs must be reproducible. If room content changes between a user's start and finish, results are unreliable.

**Decision:** Room versions are immutable once published. Bundles are content-addressed by SHA-256. Runs pin to `room_version_id`. Rollback is a pointer switch to a previous version, not mutation of content.

**Consequences:** Any content fix requires a new version + new bundles. This is deliberate: it guarantees reproducibility and simplifies debugging.

---

## ADR-006: Go as primary backend language

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Need strong concurrency (WS connections, sim ticks), fast startup (for judge jobs), and good K8s ecosystem support.

**Decision:** Go for all backend services. TypeScript for web UI (React). Room content is YAML (declarative).

**Consequences:** Single-binary deploys for each service. Strong stdlib for HTTP/WS. Go's type system is simpler than generics-heavy languages — good for agent-generated code (less ambiguity).

---

## ADR-007: Monorepo with Go modules per service

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Multiple services share types and interfaces. Polyrepo adds friction for a solo builder.

**Decision:** Single monorepo. Each service has its own `cmd/` entrypoint. Shared code in `internal/` and `pkg/`. Single `go.mod` at repo root.

**Consequences:** Simpler dependency management. Agents can see the full codebase. CI must be smart about what to rebuild (path-based triggers). Merge conflicts are the main risk — mitigated by module boundaries and small PRs.

---

## ADR-008: GitHub OAuth for auth (v1)

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Need login for progress tracking. Target audience is developers who already have GitHub accounts.

**Decision:** GitHub OAuth with server-side session cookie. RBAC: USER (play rooms, view own progress) and ADMIN (publish rooms, manage versions). CSRF header required for mutations.

**Consequences:** No email/password management. Dependent on GitHub as identity provider. Acceptable for v1 developer audience.

---

## ADR-009: Idempotent mutations via clientRequestId

**Status:** Accepted
**Date:** 2026-02-24

**Context:** WebSocket reconnections and network retries mean mutations may arrive more than once. Duplicate side effects (double-starting a run, double-submitting to judge) would corrupt state.

**Decision:** All player-facing mutations accept `clientRequestId` (UUID). Server persists idempotency keys. Same ID + same fingerprint = stored response. Same ID + different fingerprint = CONFLICT.

**Consequences:** Every mutation handler must check idempotency table before executing. Small storage overhead (idempotency keys with TTL). But eliminates an entire class of bugs.

---

## ADR-010: LGTM observability stack

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Need logs, metrics, traces for both platform debugging and gameplay signal inspection.

**Decision:** LGTM stack: Loki (logs), Grafana (dashboards), Tempo (traces), Prometheus (metrics). Deployed via docker-compose locally, Helm/k3s in production.

**Consequences:** Familiar Grafana UX for the target audience (people learning distributed systems). Disk-heavy (especially Loki/Tempo) — requires retention limits from day 1.

---

_Add new decisions below this line. Use the next sequential ADR number._

---

## ADR-011: Next.js App Router for web UI

**Status:** Accepted
**Date:** 2026-02-24

**Context:** The web UI has five distinct surfaces with different rendering needs: server-rendered catalog pages (SEO, fast initial load), client-rendered gameplay pages (WebSocket streaming, complex state), and admin forms. We need a React framework that supports both patterns cleanly.

**Decision:** Next.js 14+ with App Router. Server Components for catalog and run explorer (SSR/ISR). Client Components (`"use client"`) for Engine A/B gameplay (WS + xterm.js). TypeScript in strict mode. urql as the GraphQL client (lighter than Apollo). GraphQL types generated from the backend schema via `@graphql-codegen`.

**Consequences:** Server Components reduce JS bundle for read-heavy pages. Client Components provide the full React lifecycle needed for WS connections. Two rendering modes means agents must understand the Server/Client Component boundary — `"use client"` is required for any file that uses hooks, browser APIs, or WS. `web/` is a separate pnpm workspace; it communicates with the backend only via GraphQL and WS, never via Go imports.

---

## ADR-012: Tailwind CSS with Grafana-inspired dark theme

**Status:** Accepted
**Date:** 2026-02-24

**Context:** Engine A panels are Grafana-like dashboards (metrics, logs, topology). The UI should feel like a simplified Grafana to the target audience (developers who already know Grafana). We need a styling system that agents can use without design ambiguity.

**Decision:** Tailwind CSS with a custom theme extending Grafana-inspired tokens: dark surfaces (`#0d0d1a`, `#1a1a2e`), signal colors (green/amber/red for metrics), panel card components with subtle borders, monospace font (JetBrains Mono) for logs and terminal. No component library (e.g., no MUI, no Chakra) — build from Tailwind primitives for full control.

**Consequences:** Agents have an explicit token vocabulary (`panel-bg`, `signal-ok`, `signal-warn`, `signal-crit`, `surface-dark`, etc.) rather than subjective design choices. No external design system dependency. Trade-off: more effort to build accessible primitives (focus rings, ARIA) since we don't inherit them from a component library.
