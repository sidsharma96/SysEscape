# AGENTS.md — Systems Escape Rooms

> **This file is a living document.** Update it every time an agent makes a mistake.
> Each entry should correspond to a specific past failure that is now prevented.

## Quick Reference

| Command | Purpose |
|---------|---------|
| `make ci` | Run full gate: lint + test-unit + test-integration + build |
| `make lint` | Go vet + staticcheck + custom arch lint |
| `make test-unit` | Fast unit tests only (<2 min target) |
| `make test-integration` | Requires docker-compose up (Postgres/Kafka/Redis) |
| `make build` | Build all service binaries |
| `make fmt` | gofumpt + goimports (auto-format) |
| `make migrate-up` | Apply pending Postgres migrations |
| `make roomctl-validate` | Validate room content (schema + leak check) |
| `make ui-dev` | Start Next.js dev server on :3000 |
| `make ui-lint` | ESLint + TypeScript type-check for `web/` |
| `make ui-test` | Vitest unit + component tests |
| `make ui-build` | Next.js production build |
| `make ui-codegen` | Generate TypeScript types from GraphQL schema |

## Project Structure

```
systems-escape-rooms/
├── cmd/                    # Service entrypoints (main.go per service)
│   ├── graphql-bff/
│   ├── engine-a/
│   ├── engine-b-orchestrator/
│   ├── judge-dispatcher/
│   ├── bundle-proxy/
│   ├── artifact-proxy/
│   └── roomctl/            # CLI tool for room publishing
├── internal/               # Private Go packages (the bulk of the code)
│   ├── auth/               # GitHub OAuth, session, RBAC
│   ├── catalog/            # Room/version queries
│   ├── engine/
│   │   ├── a/              # Engine A simulator runtime
│   │   └── b/              # Engine B orchestrator + judge dispatch
│   ├── bundle/             # Bundle proxy logic, token verification
│   ├── artifact/           # Artifact proxy logic
│   ├── publish/            # Room publishing pipeline
│   ├── ws/                 # WebSocket server, reconnect/resume
│   ├── idempotency/        # clientRequestId dedup
│   ├── token/              # JWT minting/verification (run/bundle/artifact tokens)
│   └── platform/           # Shared: logging, tracing, config, errors
├── pkg/                    # Public Go packages (shared types/interfaces)
│   ├── models/             # Domain types: Room, Run, Submission, etc.
│   └── api/                # GraphQL schema types (generated)
├── migrations/             # Postgres migrations (sequential, numbered)
├── rooms/                  # Room content (one dir per room slug)
│   └── <room-slug>/
│       ├── metadata.yaml
│       ├── engineA/        # or engineB/
│       └── ...
├── infra/                  # Docker, k8s manifests, network policies
│   ├── docker-compose.yaml
│   ├── k8s/
│   └── scripts/
├── web/                    # Next.js web UI (TypeScript / React)
│   ├── src/
│   │   ├── app/            # App Router pages (catalog, play, runs, admin)
│   │   ├── components/     # UI components (by surface: catalog/, engine-a/, engine-b/, etc.)
│   │   ├── hooks/          # Custom hooks (use-ws.ts, use-engine-a.ts, use-auth.ts)
│   │   └── lib/            # GraphQL client, WS protocol, idempotency, tokens
│   ├── public/
│   ├── next.config.ts
│   ├── tailwind.config.ts
│   ├── vitest.config.ts
│   └── package.json
├── docs/                   # Architecture, decisions, dev guide, agent guidance
│   ├── ARCHITECTURE.md
│   ├── DECISIONS.md
│   ├── DEV.md
│   └── AGENTS/             # Per-module agent guidance (see below)
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── workflows/
├── Makefile
├── go.mod
├── go.sum
└── AGENTS.md               # This file
```

## Layering Rules (STRICT — linter-enforced)

```
pkg/models  →  (no internal imports)
pkg/api     →  pkg/models only
internal/*  →  pkg/models, pkg/api, internal/platform
cmd/*       →  internal/*, pkg/*
```

**Violations are CI failures.** If you need a type from another internal package,
it likely belongs in `pkg/models` or needs an interface in `pkg/api`.

### Frontend Layering (`web/`)

```
web/src/app/       →  components/, hooks/ only (pages compose — no raw fetch/WS)
web/src/components →  hooks/, lib/ (components consume state, not raw sockets)
web/src/hooks      →  lib/ only (hooks wrap lib clients into React state)
web/src/lib        →  standalone (GraphQL client, WS protocol, tokens, idempotency)
```

**Cross-boundary rule:** `web/` never imports from `internal/`, `pkg/`, or `cmd/`.
The frontend communicates with the backend **only** via GraphQL and WebSocket.
Shared types come from GraphQL codegen (`web/src/lib/graphql/types.ts`), not from Go.

Do NOT:
- Import `internal/engine/a` from `internal/engine/b` or vice versa.
- Import `internal/auth` from `internal/engine/*` — use the token interfaces.
- Import `cmd/*` from `internal/*` — command packages are leaves, not libraries.

## Do / Don't

### Do:
- Write failing tests FIRST, confirm they fail, then implement.
- Use `make ci` before opening a PR. No exceptions.
- Keep PRs under 400 lines of diff. If bigger, propose a split plan.
- Use `clientRequestId` for all mutations (idempotency is a system invariant).
- Use structured JSON logging via `internal/platform/log`.
- Add `requestId` and `runId` fields to all log entries where available.
- Validate all external inputs at boundary (transport layer), not deep in business logic.
- Use table-driven tests (`func TestX(t *testing.T) { tests := []struct{...} }`).

### Don't:
- Invent new make targets without updating this file and the Makefile.
- Use `fmt.Println` for logging. Use `slog` via the platform logger.
- Put business logic in `cmd/` main files. They only wire dependencies.
- Access Postgres directly from transport/handler code — go through the repo layer.
- Store secrets in environment variables. Use the secrets broker (see docs/ARCHITECTURE.md).
- Make network calls without going through adapter interfaces (must be mockable).
- Modify `migrations/` files that have already been applied. Create a new migration.

### Frontend Do:
- Use Server Components for catalog and run-explorer pages (data-fetching at the server).
- Use Client Components (`"use client"`) for Engine A/B gameplay (WS + complex state).
- Keep WS logic in `hooks/use-ws.ts` — components consume state, not raw sockets.
- Generate GraphQL types from the backend schema (`make ui-codegen`). Never hand-write them.
- Use `clientRequestId` (UUID v4) on every GraphQL mutation. Generate once per user action.
- Handle WS reconnection gracefully — show "Reconnecting…" toast, not a crash.
- Use `React.memo` on Engine A panel components (delta stream causes frequent re-renders).
- Put Engine A components in `components/engine-a/` and Engine B in `components/engine-b/`. Never mix them.

### Frontend Don't:
- Use `localStorage` or `sessionStorage` for tokens. Keep `runToken` in React state / in-memory. Session cookie handles auth.
- Make WS connections from Server Components. WS is client-side only (`"use client"`).
- Poll GraphQL for real-time data. Use WS for Engine A/B; GraphQL is request/response only.
- Put business logic in page components. Pages compose hooks + components.
- Import from Go backend packages (`internal/`, `pkg/`). Frontend ↔ backend boundary is network only.
- Hardcode WS URLs. Read from `NEXT_PUBLIC_WS_HOST` environment variable.
- Use `any` in TypeScript. Define types in `lib/ws/protocol.ts` or `lib/graphql/types.ts`.

## Per-Module Guidance

Detailed agent instructions for each module live in `docs/AGENTS/`:

| Module | Guidance File | Key Constraints |
|--------|---------------|-----------------|
| auth | `docs/AGENTS/auth.md` | GitHub OAuth flow; session cookie; RBAC (USER/ADMIN) |
| engine-a | `docs/AGENTS/engine-a.md` | Must be deterministic (seed + action log = same outcome) |
| engine-b | `docs/AGENTS/engine-b.md` | Workspace + judge isolation; no hidden test leakage |
| bundle/artifact proxy | `docs/AGENTS/proxy.md` | Token verification; sha256 integrity; no S3 creds in sandbox |
| graphql | `docs/AGENTS/graphql.md` | Schema additive only; idempotency on all mutations |
| rooms | `docs/AGENTS/rooms.md` | roomctl validate must pass; golden scenario required |
| ui (web/) | `docs/AGENTS/ui.md` | Next.js App Router; WS reconnect/resume; GraphQL codegen; no localStorage for tokens |
| infra | `docs/AGENTS/infra.md` | NetworkPolicy default-deny; sealed-secrets only |

> **If a guidance file doesn't exist yet, create it when you first work on that module.**

## Common Mistakes (update this section after every agent failure)

<!-- Add entries here as agents make mistakes. Format: what went wrong → fix. -->

1. _[Template]_ Agent tried to `curl` an external URL during build → Add to AGENTS.md: network egress is denied by default. Use `make` targets for all fetches.
2. _[Template]_ Agent created WS connection in a Server Component → Move to Client Component with `"use client"` directive. WS is browser-only.
3. _[Template]_ Agent hand-wrote GraphQL response types → Run `make ui-codegen` to regenerate from schema. Delete hand-written types.

## Evidence Block Reminder

Every PR must include an Evidence Block. See `.github/PULL_REQUEST_TEMPLATE.md`.
