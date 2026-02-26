# Systems Escape Rooms (SysEscape)

Systems Escape Rooms is a web-first platform for learning distributed systems and security through replayable incident simulations.

Players solve "rooms" using two gameplay engines:
- Engine A: deterministic simulator with telemetry-style panels and live state updates.
- Engine B: sandboxed coding workspace graded by hidden tests.

## Repository Overview

- `cmd/`: Go service entrypoints
- `internal/`: backend domain logic
- `pkg/`: shared public types and API models
- `web/`: Vite + React frontend
- `migrations/`: Postgres migrations
- `rooms/`: room content and assets
- `infra/`: local/docker and deployment manifests
- `docs/`: architecture, decisions, and development guides

## Core Services

- `graphql-bff`: control-plane API
- `engine-a`: simulator runtime + WS streaming
- `engine-b-orchestrator`: workspace lifecycle + submission flow
- `judge-dispatcher`: judge job orchestration
- `bundle-proxy`: bundle delivery and integrity checks
- `artifact-proxy`: artifact upload/download
- `roomctl`: room validation/build tooling

## Quick Start

```bash
# 1) Start local dependencies
make dev-up
make migrate-up

# 2) Verify backend
make build
make test-unit

# 3) Start backend services
make run-all

# 4) Start frontend (new terminal)
make ui-install
make ui-dev
```

## Useful Commands

- `make ci`: full gate (lint + tests + build)
- `make lint`: backend lint checks
- `make test-integration`: integration tests (requires local infra)
- `make roomctl-validate`: validate room content
- `make ui-lint`: frontend lint + type-check
- `make ui-test`: frontend tests
- `make ui-build`: frontend production build

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Development Guide](docs/DEV.md)
- [Decisions](docs/DECISIONS.md)
- [Agent Guidance](AGENTS.md)
