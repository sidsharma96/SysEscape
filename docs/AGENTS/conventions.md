# Coding Conventions — Systems Escape Rooms

These conventions apply to all code in the repository. Violations should be caught by
linters or tests where possible; this file covers patterns that require human/agent judgment.

## Go Backend

### Do
- Use table-driven tests: `func TestX(t *testing.T) { tests := []struct{...} }`.
- Keep PRs under 400 lines of diff. If bigger, propose a split plan before implementing.
- Use `internal/platform/log` for all logging. Add `requestId`, `runId` fields where available.
- Make all network calls through adapter interfaces (must be mockable in tests).
- Use `pgx/v5` (pgxpool) for Postgres. Repo interfaces accept/return `pkg/models` types only —
  no pgx types in the interface signature.
- Repo tests run against real Postgres (require `make dev-up`). Use transaction rollback
  per test to keep tests isolated.

### Don't
- Add `Co-Authored-By` lines to commit messages.
- Invent new make targets without updating `AGENTS.md` and the `Makefile`.
- Use `fmt.Println` for logging. Use `slog` via the platform logger.
- Put business logic in `cmd/` main files. They only wire dependencies and start servers.
- Access Postgres from transport/handler code. Go through the repo layer.
- Store secrets in code or env vars directly. Use the secrets broker (see `docs/ARCHITECTURE.md`).
- Modify migration files that have already been applied. Always create a new numbered migration.

## Frontend (web/)

### Do
- Use React Router's `React.lazy()` for Engine A/B pages (heavy; don't bundle with catalog).
- Keep route components thin — they compose hooks + layout, no business logic.
- Keep WS logic in `hooks/use-ws.ts`. Components consume state via hooks, not raw sockets.
- Generate GraphQL types from the backend schema (`make ui-codegen`). Never hand-write them.
- Use `clientRequestId` (UUID v4) on every GraphQL mutation. Generate once per user action.
- Handle WS reconnection gracefully — show "Reconnecting…" toast, not a crash.
- Use `React.memo` on Engine A panel components (delta stream causes frequent re-renders).
- Put Engine A components in `components/engine-a/` and Engine B in `components/engine-b/`.

### Don't
- Use `localStorage` or `sessionStorage` for tokens. Session cookie handles auth.
  Keep `runToken` in React state / in-memory only.
- Put WS connection setup directly in component bodies. Use `hooks/use-ws.ts`.
- Poll GraphQL for real-time data. Use WS for Engine A/B; GraphQL is request/response only.
- Import from Go backend packages. Frontend ↔ backend boundary is network only.
- Hardcode WS or GraphQL URLs. Read from `VITE_WS_HOST` / `VITE_GRAPHQL_URL`.
- Use `any` in TypeScript. Define types via codegen or in `lib/ws/protocol.ts`.

### Frontend Layering

```
routes/      → components/, hooks/ only (routes compose, no raw fetch/WS)
components/  → hooks/, lib/ (components consume state, not raw sockets)
hooks/       → lib/ only (hooks wrap lib clients into React state)
lib/         → standalone (GraphQL client, WS protocol, tokens, idempotency)
```

## Testing Patterns

- Write failing tests FIRST. Confirm they fail. Then implement.
- Backend repo tests: real Postgres, transaction rollback per test.
- Backend service tests: mock repos (implement the interface with test doubles).
- Backend transport tests: `httptest` with mocked services.
- Frontend component tests: `@testing-library/react` + mock urql provider.
- Don't test framework behavior (React Router navigation, gqlgen wiring).

## PR Requirements

- Every PR must include an Evidence Block (see `.github/PULL_REQUEST_TEMPLATE.md`).
- Run `make ci` before declaring done. No exceptions.
- PRs should touch one module boundary. If you're editing both `internal/auth/`
  and `internal/engine/`, split into two PRs.
