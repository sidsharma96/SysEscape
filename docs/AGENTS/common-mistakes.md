# Common Mistakes — Systems Escape Rooms

> **Update this file after every agent failure.** Format: what went wrong → how to fix it.
> Each entry prevents a specific past failure from recurring.

## How to Use This File

When an agent makes a mistake:
1. Add an entry below describing what happened and the fix.
2. Ask: "Can this be caught by a linter, test, or hook instead?"
   - If yes → create the check AND add the entry here.
   - If no → the entry here is the only guardrail. Make it clear.
3. If the mistake is module-specific, also add it to the relevant `docs/AGENTS/<module>.md`.

---

## Entries

1. AGENTS.md references module docs that don't exist yet. Agents should not create them opportunistically; only create when doing substantial first-time work on that module.

2. Agent added whole path of file in the scope of change in PR description. Agents should only add the path relative to the project like 'SysEscape/internal/auth/service/auth_service.go' instead of starting with 'Users/siddharthsharma/...'

3. Roomctl publish payload drifted from GraphQL schema. Always verify the live schema input fields before wiring mutation variables; do not send removed/unsupported fields such as `metadata`.

4. Local GraphQL-BFF route mismatch caused smoke failures. Use `http://localhost:8080/graphql` as the default publish endpoint unless explicitly overridden.

5. Engine A WS ordering regression: treating `connWriter` mutex safety as sufficient caused protocol-order bugs (`delta` arriving before `action_accepted` / `win_update`). For WS handlers, enforce message-order contracts explicitly:
   - `apply_action` response order must be deterministic: `action_accepted` -> immediate action `delta` -> optional `win_update`.
   - Do not allow runtime tick emissions to interleave into that response sequence.
   - Keep strict integration assertions for this ordering; do not relax tests to hide interleaving.
