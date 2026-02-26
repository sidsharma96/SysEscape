# Agent Guidance: Web UI (`web/`)

> **Stack:** Vite · React 18+ · React Router · TypeScript · Tailwind CSS
> **Role:** Room browser, Engine A panels, Engine B workspace terminal, run explorer, auth flow.
> **This file is a living document.** Update it when an agent makes a UI-related mistake.

## Directory Structure

```
web/
├── src/
│   ├── main.tsx                    # Vite entrypoint (renders <App />)
│   ├── App.tsx                     # React Router setup + providers (auth, theme)
│   ├── routes/                     # Route components (one file per page)
│   │   ├── Layout.tsx              # Root layout (nav shell, theme, auth guard wrapper)
│   │   ├── CatalogPage.tsx         # Landing / room catalog
│   │   ├── RoomDetailPage.tsx      # Room detail + "Start Run" CTA
│   │   ├── LoginCallbackPage.tsx   # GitHub OAuth callback
│   │   ├── EngineAPage.tsx         # Engine A gameplay (panels + topology + actions)
│   │   ├── EngineBPage.tsx         # Engine B gameplay (terminal + submit + results)
│   │   ├── RunsPage.tsx            # Run history / progress dashboard
│   │   └── AdminPublishPage.tsx    # Room publishing (ADMIN only)
│   │
│   ├── components/
│   │   ├── ui/                     # Generic primitives (Button, Card, Modal, Toast, etc.)
│   │   ├── layout/                 # Shell, Nav, Sidebar, Footer
│   │   ├── catalog/                # RoomCard, RoomGrid, DifficultyBadge, DistrictFilter
│   │   ├── engine-a/               # Panels, TopologyMap, ActionBar, TimerBar, WinOverlay
│   │   ├── engine-b/               # Terminal, SubmitButton, JudgeStatus, ArtifactViewer
│   │   ├── run-explorer/           # RunList, RunDetail, TraceViewer, LogViewer
│   │   └── auth/                   # LoginButton, UserMenu, AuthGuard
│   │
│   ├── hooks/                      # Custom React hooks
│   │   ├── use-ws.ts               # WebSocket connection + reconnect + resume
│   │   ├── use-engine-a.ts         # Engine A state machine (snapshot/delta/actions)
│   │   ├── use-engine-b.ts         # Engine B terminal + judge status
│   │   ├── use-graphql.ts          # GraphQL client wrapper
│   │   └── use-auth.ts             # Session state + login/logout
│   │
│   ├── lib/
│   │   ├── graphql/                # GraphQL client, queries, mutations, types
│   │   │   ├── client.ts           # urql client config
│   │   │   ├── queries.ts          # Catalog, runs, progress queries
│   │   │   ├── mutations.ts        # startRun, submitProof, submitToJudge, publish
│   │   │   └── types.ts            # Generated types (from GraphQL schema)
│   │   ├── ws/
│   │   │   ├── protocol.ts         # WS message types, envelope, hello/ack/delta/snapshot
│   │   │   ├── client.ts           # WS connect, reconnect, resume-by-seq logic
│   │   │   └── heartbeat.ts        # Ping/pong handler (25s interval, 75s timeout)
│   │   ├── tokens.ts               # runToken storage (memory only, NOT localStorage)
│   │   └── idempotency.ts          # clientRequestId generation (UUID v4)
│   │
│   ├── styles/                     # Tailwind config, global styles, Grafana-like theme tokens
│   └── __tests__/                  # Test files (mirroring src/ structure)
│
├── public/                         # Static assets
├── index.html                      # Vite HTML entrypoint
├── vite.config.ts                  # Vite config (proxy for /graphql and /ws in dev)
├── tailwind.config.ts
├── tsconfig.json
├── vitest.config.ts                # Test config (Vitest + React Testing Library)
└── package.json
```

## Five UI Surfaces

The UI has five distinct surfaces. Each has different data patterns and state complexity.
**Do not mix state between surfaces** — they are separate feature domains.

### 1. Room Catalog (`/`, `/rooms/[slug]`)

| Aspect | Detail |
|--------|--------|
| Data source | GraphQL queries (`rooms`, `roomBySlug`) |
| State | Server-fetched; minimal client state (filters, search) |
| Auth | Public for browsing; auth required for "Start Run" |
| Key components | `RoomCard`, `RoomGrid`, `DifficultyBadge`, `DistrictFilter` |
| Agent notes | Fetches room list on mount via GraphQL. Consider caching with urql's document cache. Simple data-in → render-out. |

### 2. Engine A Gameplay (`/play/[runId]/engine-a`)

| Aspect | Detail |
|--------|--------|
| Data source | WebSocket `wss://<domain>/ws/engineA/{runId}` |
| State | Complex: snapshot + delta stream → derived panel state |
| Auth | `runToken` (JWT, from `startRun` mutation) sent in WS `hello` |
| Key components | `MetricsPanel`, `LogsPanel`, `TopologyMap`, `ActionBar`, `TimerBar`, `WinOverlay` |
| Agent notes | This is the most complex surface. See "Engine A State Machine" below. Must handle reconnect/resume. All panels are **read-only views of server-streamed state** — the only user action is `apply_action`. |

### 3. Engine B Gameplay (`/play/[runId]/engine-b`)

| Aspect | Detail |
|--------|--------|
| Data source | WebSocket `wss://<domain>/ws/engineB/{runId}` (terminal stream) |
| State | Terminal buffer + judge submission status (async) |
| Auth | `runToken` in WS `hello` |
| Key components | `Terminal` (xterm.js), `SubmitButton`, `JudgeStatus`, `ArtifactViewer`, `VerdictCard` |
| Agent notes | Terminal uses xterm.js with the WS bridge. Judge results arrive async via WS after submission. Artifact download uses short-lived `artifactToken`. |

### 4. Run Explorer / Progress (`/runs`)

| Aspect | Detail |
|--------|--------|
| Data source | GraphQL queries (`myRuns`, `runById`) |
| State | Server-fetched; optional trace/log drill-down |
| Auth | Authenticated (user sees only own runs) |
| Key components | `RunList`, `RunDetail`, `AtlasCard`, `TraceViewer` (links to Grafana) |
| Agent notes | Simple CRUD views. Fetch on mount, minimal client state. Good candidate for route-level data loading. |

### 5. Admin: Room Publishing (`/admin/publish`)

| Aspect | Detail |
|--------|--------|
| Data source | GraphQL mutation (`publishRoomVersion`) |
| State | Form state + upload progress |
| Auth | ADMIN role required |
| Key components | `PublishForm`, `VersionList`, `ActivateToggle` |
| Agent notes | ADMIN-only. Guard with `AuthGuard` role check. |

## Engine A State Machine (critical section)

Engine A is the hardest UI surface. The server streams sim state over WS as snapshots + deltas.
The client must maintain a local state object that reflects the simulation.

```
┌───────────┐   hello(runToken,     ┌───────────┐   hello_ack    ┌──────────────┐
│DISCONNECTED│──resumeFromSeq?)───▶│ CONNECTING │──────────────▶│  CONNECTED    │
└───────────┘                      └───────────┘               └──────┬───────┘
      ▲                                                               │
      │ WS close / error                              snapshot or delta│
      │ (auto-reconnect                                               ▼
      │  with backoff)                                         ┌──────────────┐
      └────────────────────────────────────────────────────────│   STREAMING  │
                                                               │ (panels live)│
                                                               └──────────────┘
```

### State update rules

1. **On `snapshot` message:** replace entire local state with snapshot payload. Set `lastSeq = snapshot.seq`.
2. **On `delta` message:** apply `ops` array to local state. Delta `seq` must equal `lastSeq + 1`. If seq gap detected → request full snapshot (reconnect with no `resumeFromSeq`).
3. **On `action_accepted`:** optimistic UI is allowed but must be reconciled with the next delta. If action is rejected, revert optimistic state.
4. **On `win_update`:** show win overlay. Run is complete.

### Key invariant
The UI **never mutates sim state locally**. All state comes from the server via snapshots/deltas.
The only client→server message is `apply_action` (with `clientRequestId` for idempotency).

## WebSocket Reconnection Protocol

This applies to both Engine A and Engine B.

```typescript
// Pseudo-code for the reconnection logic in use-ws.ts
const RECONNECT_DELAYS = [500, 1000, 2000, 4000, 8000]; // exponential backoff, cap at 8s

function connect(runId: string, runToken: string, resumeFromSeq?: number) {
  const ws = new WebSocket(`wss://${host}/ws/engineA/${runId}`);

  ws.onopen = () => {
    ws.send(JSON.stringify({
      protocolVersion: 1,
      type: "hello",
      runId,
      payload: { runToken, resumeFromSeq }
    }));
  };

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    switch (msg.type) {
      case "hello_ack":
        // If snapshotRequired, wait for snapshot message
        // Else, expect deltas starting at resumeFromSeq + 1
        break;
      case "snapshot":
        replaceState(msg.payload);
        break;
      case "delta":
        applyDelta(msg.payload);
        break;
      case "ping":
        ws.send(JSON.stringify({ type: "pong" }));
        break;
      // ... other message types
    }
  };

  ws.onclose = () => {
    // Auto-reconnect with backoff, passing lastSeq as resumeFromSeq
    scheduleReconnect(runId, runToken, lastSeq);
  };
}
```

### Rules
- **Always pass `resumeFromSeq`** on reconnect so the server can resume from where the client left off.
- **If the server responds with `snapshotRequired: true`**, the client's seq was too old — accept the full snapshot.
- **Heartbeat:** respond to server `ping` with `pong` within a few seconds. Server disconnects after ~75s of silence.
- **Backoff:** use exponential backoff (500ms → 1s → 2s → 4s → 8s cap). Reset on successful `hello_ack`.
- **Token refresh:** `runToken` has 10–30 min TTL. Before it expires, call the GraphQL mutation to refresh it. If expired during reconnect, fetch a new token first.

## GraphQL Usage

### Client setup
Use `urql` (lightweight) or Apollo Client. Configure with:
- **Session cookie auth** (credentials: "include" for same-origin).
- **CSRF header** on mutations: `X-CSRF-Token` (value from cookie or meta tag).

### Idempotency
All mutations that create resources (`startRun`, `submitProof`, `submitToJudge`, `publishRoomVersion`) must include `clientRequestId` (UUID v4). Generate a fresh UUID per user action. Store the UUID so retries reuse the same one.

```typescript
// lib/idempotency.ts
import { v4 as uuidv4 } from "uuid";

// Generate once per user intent; reuse on retry
export function newRequestId(): string {
  return uuidv4();
}
```

### Error handling
GraphQL errors include `extensions.code`:
- `UNAUTHENTICATED` → redirect to login
- `FORBIDDEN` → show "not authorized" (role check failed)
- `CONFLICT` → idempotency violation (different payload, same clientRequestId)
- `VALIDATION_FAILED` → show field-level errors

## Styling and Theming

### Grafana-inspired design
Engine A panels should feel like a simplified Grafana. Use a **dark theme** with:
- Dark background (`#1a1a2e` or similar)
- Panel cards with subtle borders
- Green/amber/red signal colors for metrics
- Monospace font for logs and terminal

### Tailwind CSS
Use Tailwind with a custom theme extending the Grafana-inspired tokens:

```typescript
// tailwind.config.ts (theme extension snippet)
theme: {
  extend: {
    colors: {
      panel: { bg: '#1e1e3f', border: '#2a2a5a', hover: '#2e2e6a' },
      signal: { ok: '#73bf69', warn: '#ff9830', crit: '#f2495c' },
      surface: { dark: '#0d0d1a', mid: '#1a1a2e', light: '#2a2a4a' },
    },
    fontFamily: {
      mono: ['"JetBrains Mono"', '"Fira Code"', 'monospace'],
    },
  },
}
```

## Testing Strategy

### What to test

| Layer | Tool | What to test |
|-------|------|-------------|
| Component rendering | Vitest + React Testing Library | Panel renders with snapshot data; action bar emits correct messages |
| WS protocol logic | Vitest (unit) | `use-ws` hook: reconnect backoff, seq tracking, snapshot/delta application |
| Engine A state | Vitest (unit) | Delta application, seq gap detection, optimistic action rollback |
| GraphQL queries | MSW (Mock Service Worker) | Correct query variables, error handling, loading states |
| E2E | Playwright | Login → start room → apply action → see panel update (against real backend) |

### How to run

```bash
# From repo root
make ui-test          # Vitest (unit + component tests)
make ui-lint          # ESLint + TypeScript check
make ui-build         # Vite production build
make ui-dev           # Vite dev server on :3000
```

### What NOT to test
- Don't test Tailwind class names (they change with design iteration).
- Don't test React Router navigation (framework's job).
- Don't snapshot-test entire pages (too brittle with agents generating code).

## Do / Don't

### Do:
- Use React Router's lazy loading (`React.lazy`) for gameplay pages (Engine A/B are heavy; don't load them for catalog visitors).
- Keep route components thin — they compose hooks + layout components, no business logic.
- Keep WS logic in `hooks/use-ws.ts` — components consume state, not raw sockets.
- Generate GraphQL types from the backend schema (codegen). Don't hand-write them.
- Use `clientRequestId` on every mutation. Generate UUID once per user action.
- Handle WS reconnection gracefully — user should see a brief "reconnecting..." toast, not a crash.
- Put Engine A panel components in `components/engine-a/` and Engine B in `components/engine-b/`. Never mix them.
- Use `React.memo` on panel components (Engine A re-renders frequently from delta stream).

### Don't:
- Don't use `localStorage` or `sessionStorage` for tokens. Keep `runToken` in React state / in-memory only. Session cookie handles auth.
- Put business logic in route components. Routes compose hooks + components — no raw fetch/WS logic.
- Don't poll GraphQL for real-time data. Use WS for Engine A/B; GraphQL is for request/response only.
- Don't put business logic in page components. Pages compose hooks + components — no raw fetch/WS logic.
- Don't import from `internal/` (Go backend). The frontend communicates only via GraphQL and WS.
- Don't hardcode WS URLs. Read from environment variable (`VITE_WS_HOST`).
- Don't use `any` in TypeScript. If the type is complex, define it in `lib/ws/protocol.ts` or `lib/graphql/types.ts`.

## Common Mistakes (update after every agent failure)

<!-- Add entries as agents make mistakes. Format: what went wrong → fix. -->

1. _[Template]_ Agent put WS connection setup in a component body instead of a hook → Move to `hooks/use-ws.ts`. Components consume state, not raw sockets.

## Dependencies (recommended)

| Package | Purpose | Notes |
|---------|---------|-------|
| `vite` | Build tool + dev server | Near-instant HMR, fast builds |
| `react`, `react-dom` | UI library | 18+ |
| `react-router-dom` | Client-side routing | v6+, lazy loading for code splitting |
| `typescript` | Type safety | Strict mode enabled |
| `tailwindcss` | Styling | Dark theme with custom tokens |
| `urql` or `@apollo/client` | GraphQL client | urql preferred (lighter) |
| `graphql` + `@graphql-codegen/*` | Type generation | Codegen from backend schema |
| `xterm` + `@xterm/addon-fit` + `@xterm/addon-web-links` | Terminal emulator | Engine B workspace terminal |
| `uuid` | clientRequestId | v4 UUIDs |
| `vitest` + `@testing-library/react` | Unit/component tests | Fast, Vite-based |
| `msw` | API mocking | Mock GraphQL + WS in tests |
| `playwright` | E2E tests | Browser-level testing |
