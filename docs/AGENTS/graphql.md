# GraphQL BFF — Agent Guidance

## Role

Single entry point for all frontend requests. Serves at :8080.
Handles both GraphQL queries and auth HTTP routes.

## Schema Rules

- Schema is additive only. Never remove or rename a field once merged.
- No mutations in M1. Only queries: viewer, rooms, roomBySlug.
- No GraphQL subscriptions. Real-time uses WebSocket (separate service).

## Wiring

cmd/graphql-bff/main.go is wiring only:
- Create pgxpool from DATABASE_URL
- Create repo instances (PostgresUserRepo, PostgresSessionRepo, PostgresRoomRepo)
- Create AuthService (with RealGitHubClient)
- Set up HTTP mux:
  - GET  /auth/github/login    → HandleGitHubLogin
  - GET  /auth/github/callback → HandleGitHubCallback
  - POST /auth/logout          → HandleLogout
  - POST /graphql              → gqlgen handler (wrapped in SessionMiddleware)
  - GET  /healthz              → 200 OK
- Read all config from env vars, never hardcode

## Resolver Patterns

- Resolvers are thin: extract args, call repo/service, map result.
- viewer resolver returns nil (not error) for unauthenticated requests.
- rooms resolver returns empty slice (not null) when no rooms match.
- All repo dependencies injected via resolver struct, not globals.
