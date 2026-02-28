# Auth Module — Agent Guidance

## Architecture

- Auth is server-side only. The frontend never touches tokens directly.
- Session = HTTP-only Secure SameSite=Lax cookie. No exceptions.
- OAuth secrets come from config structs, never hardcoded or read from env inline.

## Layering
```
transport/  → AuthService (interface or struct)
service/    → UserRepo, SessionRepo, GitHubClient (all interfaces)
repo/       → pgxpool (concrete Postgres)
```

Transport calls service. Service calls repo + GitHubClient.
Transport never calls repo directly.

## Key Decisions

- GitHubClient is an interface defined in the service package (where it's consumed),
  not in transport. Real implementation and mock both satisfy it.
- Session expiry is checked on every ValidateSession call, not just at creation.
- SessionMiddleware does NOT block unauthenticated users. It sets user in context
  if valid, continues without if not. RequireAuth is a separate middleware applied per-route.
- Context key for the user is an unexported type to prevent collisions.

## Security Invariants

- Cookie: HttpOnly=true, Secure=true (in prod), SameSite=Lax, Path=/
- Session TTL: 7 days from creation. No sliding window.
- OAuth state parameter: include if CSRF protection is needed (acceptable to skip in M1,
  must add before production).
- Never log access tokens or OAuth secrets.
