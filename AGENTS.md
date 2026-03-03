# AGENTS.md — Systems Escape Rooms

> Update this file when an agent makes a mistake that no linter or test caught.
> Every entry should prevent a specific past failure from recurring.

## Commands

| Command                 | Purpose                                                            |
| ----------------------- | ------------------------------------------------------------------ |
| `make ci`               | **Run this before declaring done.** Full gate: lint + test + build |
| `make lint`             | Go vet + staticcheck + arch lint                                   |
| `make test-unit`        | Fast unit tests (<2 min)                                           |
| `make test-integration` | Requires `make dev-up` (Postgres/Kafka/Redis)                      |
| `make build`            | Compile all service binaries                                       |
| `make fmt`              | Auto-format (gofumpt + goimports)                                  |
| `make migrate-up`       | Apply pending Postgres migrations                                  |
| `make ui-dev`           | Vite dev server on :3000                                           |
| `make ui-lint`          | ESLint + TypeScript type-check                                     |
| `make ui-test`          | Vitest tests                                                       |
| `make ui-build`         | Vite production build                                              |
| `make ui-codegen`       | Generate TS types from GraphQL schema                              |
| `make smoke-m3`         | E2E smoke for Engine A WS (requires local stack)                   |

## Stop Conditions

- If `make ci` fails twice on the same issue: **STOP.** Describe the failure. Do not loop.
- If you need to modify files outside your stated task scope: **STOP** and ask.
- If a test is flaky (passes sometimes, fails sometimes): **STOP** and report it.
  Do not add retries or `time.Sleep` hacks.
- Maximum 2 CI retry cycles per PR. After that, the harness needs fixing, not the code.

## Core Invariants (linter-enforced)

```
pkg/models  →  no internal/ imports
pkg/api     →  pkg/models only
internal/*  →  pkg/models, pkg/api, internal/platform
cmd/*       →  internal/*, pkg/*  (cmd/ packages are leaves, not libraries)
web/        →  network boundary only: GraphQL + WebSocket. Never imports Go packages.
```

Shared types between frontend and backend come from GraphQL codegen, not from Go.

## Key Rules

- Never add `Co-Authored-By` or similar attribution trailers to commit messages.
- Write failing tests FIRST. Confirm they fail. Then implement.
- All external inputs validated at the boundary (transport layer).
- `clientRequestId` (UUID v4) on every mutation — idempotency is a system invariant.
- Structured JSON logging via `slog`. Include `requestId` and `runId` where available.
- Never modify a migration file that has already been applied. Create a new one.
- Business logic lives in `internal/*/service/`. Not in `cmd/` (wiring only) or transport (HTTP/GraphQL handlers).
- Database access goes through repo interfaces that accept/return `pkg/models` types. Use `pgx/v5`, not `database/sql`.

## Frontend Key Rules

- WS logic in `hooks/` only. Components consume state via hooks, not raw sockets.
- `React.lazy()` for Engine A/B pages. Don't bundle gameplay with the catalog.
- Tokens in memory only. Never `localStorage`/`sessionStorage`. Session cookie handles auth.
- Generate GraphQL types with `make ui-codegen`. Never hand-write them.
- Read from `VITE_*` env vars. Never hardcode URLs.

## Before Working on a Module

**IMPORTANT:** Read the relevant guidance file before starting. These contain module-specific
patterns, pitfalls, and architectural decisions that aren't in this file.

| Module                       | Read first                |
| ---------------------------- | ------------------------- |
| Auth (OAuth, sessions, RBAC) | `docs/AGENTS/auth.md`     |
| Room catalog                 | `docs/AGENTS/catalog.md`  |
| Engine A (simulator)         | `docs/AGENTS/engine-a.md` |
| Engine B (sandbox + judge)   | `docs/AGENTS/engine-b.md` |
| GraphQL BFF                  | `docs/AGENTS/graphql.md`  |
| Frontend (web/)              | `docs/AGENTS/ui.md`       |
| Bundle/artifact proxies      | `docs/AGENTS/proxy.md`    |
| Infrastructure               | `docs/AGENTS/infra.md`    |
| Room content authoring       | `docs/AGENTS/rooms.md`    |

If a guidance file doesn't exist yet, create it when you first work on that module.

For detailed coding conventions (Do/Don't lists, testing patterns, PR requirements):
see `docs/AGENTS/conventions.md`.

## Common Mistakes

Track at: `docs/AGENTS/common-mistakes.md`. Add an entry after every agent failure.

## PR Evidence Block

Use this exact block in every PR description:

```md
## Evidence
- Goal: <PR goal from Section 4>
- Scope (modules/files): <list from scope fence>
- Commands run:
  - make lint: PASS/FAIL (runtime: ___)
  - make test-unit: PASS/FAIL (runtime: ___)
  - make test-integration: PASS/FAIL (runtime: ___)
  - make ci: PASS/FAIL (runtime: ___)
- Proof artifacts: <test output snippet>
- Risk notes: <migrations / compat / security>
- Rollback: <how to revert>
```
