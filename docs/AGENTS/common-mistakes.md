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

1. _[Template]_ Agent tried to `curl` an external URL during build → Network egress is
   denied by default. Use `make` targets for all fetches.

2. _[Template]_ Agent put WS connection setup in a component body → Move to
   `hooks/use-ws.ts`. Components consume state via hooks, not raw sockets.

3. _[Template]_ Agent hand-wrote GraphQL response types → Run `make ui-codegen` to
   regenerate from schema. Delete hand-written types.
