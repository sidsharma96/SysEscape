# Local Development Guide — Systems Escape Rooms

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl/) |
| Docker + Docker Compose | 24+ / v2 | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Node.js + pnpm | 20+ / 9+ | [nodejs.org](https://nodejs.org/), `corepack enable && corepack prepare pnpm@latest --activate` |
| kubectl | 1.28+ | For k8s interaction (optional for local dev) |
| GitHub CLI (`gh`) | 2.40+ | For PR operations |
| gofumpt | latest | `go install mvdan.cc/gofumpt@latest` |
| staticcheck | latest | `go install honnef.co/go/tools/cmd/staticcheck@latest` |

## One-Command Setup

```bash
# Clone and start everything
git clone <repo-url> && cd systems-escape-rooms
make dev-up        # starts Postgres, Kafka, Redis, MinIO via docker-compose
make migrate-up    # applies Postgres migrations
make build         # builds all Go service binaries
make test-unit     # verifies backend works

# Web UI setup
make ui-install    # pnpm install in web/
make ui-codegen    # generate TypeScript types from GraphQL schema
make ui-build      # verify frontend builds
make ui-test       # verify frontend tests pass
```

## Docker Compose Services (local)

```bash
make dev-up        # start all infra services
make dev-down      # stop all infra services
make dev-logs      # tail logs from all services
```

Starts:
- **Postgres** on `localhost:5432` (user: `ser`, password: `ser`, db: `ser`)
- **Kafka** (KRaft mode) on `localhost:9092`
- **Redis** on `localhost:6379`
- **MinIO** (S3-compatible) on `localhost:9000` (console: `localhost:9001`)

All data is stored in Docker volumes. To reset: `make dev-reset` (destroys volumes).

## Running Services Locally

```bash
# Run individual services (each in a separate terminal)
make run-graphql       # GraphQL BFF on :8080
make run-engine-a      # Engine A on :8081
make run-engine-b      # Engine B Orchestrator on :8082
make run-bundle-proxy  # Bundle Proxy on :8083
make run-artifact-proxy # Artifact Proxy on :8084

# Or run all at once (uses goreman/overmind if available)
make run-all
```

## Environment Variables

Services read config from environment variables with sensible local defaults.
A `.env.local` file is loaded automatically by `make run-*` targets.

```bash
# .env.local (committed to repo with local-only values)
DATABASE_URL=postgres://ser:ser@localhost:5432/ser?sslmode=disable
KAFKA_BROKERS=localhost:9092
REDIS_URL=redis://localhost:6379
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=ser-bundles
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
GITHUB_CLIENT_ID=<your-dev-oauth-app-id>
GITHUB_CLIENT_SECRET=<your-dev-oauth-app-secret>
LOG_LEVEL=debug
LOG_FORMAT=text
```

> **Never commit real secrets.** The `.env.local` values are for local dev only.
> Production secrets use sealed-secrets in k8s (see ARCHITECTURE.md §10).

## Make Targets Reference

### Core (used by CI and agents)

| Target | What it does | When to use |
|--------|-------------|-------------|
| `make ci` | `lint` + `test-unit` + `test-integration` + `build` | Before every PR. Non-negotiable. |
| `make lint` | `go vet` + `staticcheck` + `gofumpt -d` + arch linter | Catches style + architecture violations |
| `make test-unit` | `go test ./... -short` (no external deps) | Fast feedback (<2 min) |
| `make test-integration` | Tests requiring Postgres/Kafka/Redis | After schema or eventing changes |
| `make build` | `go build ./cmd/...` | Verifies compilation |
| `make fmt` | `gofumpt -w .` + `goimports -w .` | Auto-fix formatting |

### Infrastructure

| Target | What it does |
|--------|-------------|
| `make dev-up` | `docker-compose up -d` (Postgres, Kafka, Redis, MinIO) |
| `make dev-down` | `docker-compose down` |
| `make dev-reset` | `docker-compose down -v` (destroys data volumes) |
| `make dev-logs` | `docker-compose logs -f` |
| `make migrate-up` | Apply pending migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-create NAME=<name>` | Create new migration files |

### Room Content

| Target | What it does |
|--------|-------------|
| `make roomctl-validate` | Validate all rooms (schema + leak check) |
| `make roomctl-build` | Build room bundles |
| `make roomctl-validate ROOM=<slug>` | Validate a single room |

### Web UI (`web/`)

| Target | What it does | When to use |
|--------|-------------|-------------|
| `make ui-install` | `pnpm install` in `web/` | After clone or dependency changes |
| `make ui-dev` | Start Next.js dev server on `:3000` | Local UI development |
| `make ui-lint` | ESLint + TypeScript type-check | Catches type errors + style violations |
| `make ui-test` | Vitest unit + component tests | Before every PR touching `web/` |
| `make ui-build` | Next.js production build | Verifies build; runs in CI |
| `make ui-codegen` | Generate TS types from GraphQL schema | After any GraphQL schema change |
| `make ui-fmt` | Prettier + ESLint auto-fix | Auto-fix formatting |

## Web UI Development

The web UI lives in `web/` and is a separate pnpm workspace (not part of `go.mod`).

```bash
# Start the UI dev server (proxies API to backend on :8080)
make ui-dev        # Next.js dev server on http://localhost:3000

# Run the full backend alongside it (separate terminal)
make run-all       # All Go services
```

### GraphQL Codegen

TypeScript types are generated from the backend's GraphQL schema. Never hand-write response types.

```bash
# After any change to the GraphQL schema:
make ui-codegen

# Generated files land in:
#   web/src/lib/graphql/types.ts
```

### Environment Variables (UI)

UI env vars use the `NEXT_PUBLIC_` prefix for client-side values:

```bash
# web/.env.local (local dev only, not committed)
NEXT_PUBLIC_GRAPHQL_URL=http://localhost:8080/graphql
NEXT_PUBLIC_WS_HOST=localhost:8081
NEXT_PUBLIC_SITE_URL=http://localhost:3000
```

## Database Migrations

Migrations live in `migrations/` and are numbered sequentially.

```bash
# Create a new migration
make migrate-create NAME=add_submissions_table

# Apply all pending migrations
make migrate-up

# Roll back the last migration
make migrate-down
```

**Rules:**
- Never modify a migration that has already been applied (on any environment).
- Always create a new migration for schema changes.
- Test both up and down migrations locally before committing.
- For large tables: prefer additive changes first (new columns), then backfill, then switch reads.

## Troubleshooting

### `make dev-up` fails with port conflicts
Something else is using 5432/9092/6379. Check with `lsof -i :5432` and stop the conflicting process, or change the port in `docker-compose.yaml`.

### `make test-integration` fails with connection refused
Docker Compose services aren't ready. Run `make dev-up` and wait 5–10 seconds, then retry. Kafka in KRaft mode can be slow to start on first boot.

### `make lint` fails on import order
Run `make fmt` to auto-fix. If it still fails, check for a circular import (layering violation — see ARCHITECTURE.md §4).

### Migrations fail with "already applied"
You probably modified an existing migration file. Create a new one instead. If you need to reset: `make dev-reset && make migrate-up`.

### Agent session: "command not found"
Agents must use `make` targets exclusively. If a command is missing, add it to the Makefile — don't bypass with raw `go` commands.

### `make ui-dev` fails with "Module not found"
Run `make ui-install` first. If still failing, delete `web/node_modules` and retry.

### `make ui-codegen` produces empty types
The GraphQL schema file doesn't exist yet or the backend isn't serving the introspection endpoint. Ensure `make run-graphql` is running, or provide the schema file at `web/schema.graphql`.

### WS connection fails in dev with CORS error
The Next.js dev server (:3000) is trying to connect to the WS server (:8081) cross-origin. Either configure the Next.js `rewrites` in `next.config.ts` to proxy `/ws/` paths, or set the CORS allow-origin header on the Go WS handler for local dev.

### TypeScript errors after pulling new changes
Run `make ui-codegen` to regenerate types from the latest GraphQL schema, then `make ui-lint` to verify.
