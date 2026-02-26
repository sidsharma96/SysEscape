## Summary

<!-- One sentence: what does this PR do and why? -->


## Related Issue

<!-- Link to the GitHub issue this addresses. -->
Closes #


## Type of Change

<!-- Check the one that applies. -->
- [ ] Feature (new functionality)
- [ ] Bug fix
- [ ] Refactor (no behavior change)
- [ ] Infrastructure / CI
- [ ] Room content
- [ ] Documentation
- [ ] Harness improvement (tests/docs/error messages that make agents more effective)


## Evidence Block

<!-- REQUIRED. Fill in all fields. If a gate doesn't apply, write N/A with reason. -->

```
## Evidence
- Goal:
- Scope (modules/files changed):
- Commands run:
  - make lint:             PASS / FAIL  (runtime: ___)
  - make test-unit:        PASS / FAIL  (runtime: ___)
  - make test-integration: PASS / FAIL  (runtime: ___)
  - make ui-lint:          PASS / FAIL / N/A  (runtime: ___)
  - make ui-test:          PASS / FAIL / N/A  (runtime: ___)
  - make ci:               PASS / FAIL  (runtime: ___)
- Proof artifacts (if applicable):
  - UI screenshots:
  - Logs/snippets:
  - Golden scenario recorded: yes / no / N/A
- Risk notes:
  - Migrations:      yes / no
  - Compat impact:   yes / no
  - Security review:  needed / done / N/A
  - Perf impact:     yes / no
- Rollback:
  - <how to revert if this breaks something>
```


## Checklist

<!-- All boxes must be checked before merge. -->
- [ ] `make ci` passes locally
- [ ] PR is under 400 LOC diff (or split plan documented below)
- [ ] Tests added/updated for new behavior
- [ ] Failing tests written FIRST, confirmed they fail, then implementation added
- [ ] No API contract changes (or: contract change documented + versioned)
- [ ] No layering rule violations (see AGENTS.md)
- [ ] `docs/` updated if behavior or interfaces changed
- [ ] `AGENTS.md` or `docs/AGENTS/*.md` updated if agent guidance affected
- [ ] `docs/DECISIONS.md` updated if a new architectural decision was made
- [ ] If touching `web/`: no `any` types, no `localStorage` for tokens, WS logic in hooks only
- [ ] If GraphQL schema changed: `make ui-codegen` run and generated types committed


## Split Plan (if >400 LOC)

<!-- If this PR exceeds 400 LOC, explain why it can't be split, or list the follow-up PRs. -->


## Agent Session Metadata (if agent-generated)

<!-- Fill this in if a coding agent produced this PR. Helps with observability. -->

| Field | Value |
|-------|-------|
| Agent role | implementer / scout / critic / verifier / room-author |
| Model used | |
| Session duration | |
| CI retries | 0 / 1 / 2 |
| Handoff from previous session? | yes / no |
