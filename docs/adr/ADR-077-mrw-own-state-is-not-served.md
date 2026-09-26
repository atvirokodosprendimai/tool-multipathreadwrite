# ADR-077: mrw's own state is never served as the caller's file

**Status:** Accepted
**Accepted:** 2026-09-26 by M — approved the plan to clear the open backlog (*"build a plan to address them at once, no dangling pieces, no dead code, no mockery, no features only in tests, all is wired, all is exercised"*), whose ADR-077 is this record
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-006, ADR-007, ADR-031, ADR-034, ADR-076, docs/adr/BACKLOG.md
**Governs:** `internal/rooted/rooted.go`, `internal/rooted/state077_test.go`, `internal/state/state.go`, `internal/state/prune.go`, `internal/state/base077_test.go`, `internal/read/astgrep.go`, `internal/read/state077_test.go`, `internal/mcp/state077_test.go`, `cmd/mrw/state077_test.go`, `scripts/contract.sh`, `AGENTS.md`, `docs/adr/ADR-007-mrw-finds-the-files-it-serves.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/rooted/state077_test.go::TestAPathInsideMrwsStateIsRefused`
**Invalidates:** none (amends ADR-007 rule 6)
**Served-path change:** a path inside mrw's state base (`$XDG_STATE_HOME/mrw`, or `~/.local/state/mrw`) is refused by `rooted.Resolve` on every surface — a read, a plan, a `check` path, a working-set entry — naming mrw's own state; a `--grep` drops every file inside it as it drops any path the boundary refuses, and an `--ast-grep` hit inside it is dropped the same way. Nothing changes for a root that does not hold the state base.

## Context

**What was observed** (2026-09-26, while ranking the backlog): with `XDG_STATE_HOME` inside the
root, a `--grep` walked mrw's state directory and served the ledger, and an explicit spec served
`pending.json`. Over MCP that file holds the checkpoint ids an `ack` names, so a caller could ack
lines it had never received — the thing ADR-031's ack exists to prevent — and `mrw_write` could edit
the ledger that licenses every write. `--root "$HOME"`, which ADR-018 serves by design, holds
`~/.local/state/mrw` by default, so this is not only an exotic configuration.

**Why a walk skip alone would not do.** The walk already sends every discovered file through
`rooted.Resolve` and drops a refusal in silence (ADR-007 rule 3, `walk.go`). The boundary is where
the rule belongs; a skip beside `.git` would close only the walk and leave the named read and the
plan open.

## Existing Primitives Audit

- `rooted.Resolve` is the one boundary (ADR-006) every surface calls.
- `state.stateHome` names the state home; `Dir` and `openBase` each joined `mrw` onto it.
- `rooted.Contains` compares a path with a root by separator.

## Decision

1. `state.Base` names `<state home>/mrw` without making it; `Dir` and `openBase` use it.
2. `rooted.InState` reports whether a path is inside the base, both sides resolved as far as they
   exist (macOS `/var` is `/private/var`; the base may not exist yet).
3. `rooted.Resolve` refuses a path `InState`, after the root check: "is inside mrw's own state
   directory".
4. An `--ast-grep` hit `InState` is dropped unless the caller named it, as the walk drops every
   discovered path the boundary refuses.

## Alternatives Considered

- **Skip the directory in the walk.** Rejected as the fix: the named read, the plan and the MCP ack
  store stay open, and the walk already defers to the boundary.
- **Refuse a root that contains the state base.** Rejected: `--root "$HOME"` is served by design.

## Component / Boundary Impact

`internal/rooted` gains an edge to `internal/state`, which imports nothing internal. `internal/read`'s
ast-grep path consults `rooted.InState`. The engine packages `apply`, `plan`, `check`, `lines`,
`iter`, `seen` and `subproc` are unchanged.

## Wiring & Contract Changes

A refusal on every surface for a path inside the state base. Contract §154.

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- Every `Resolve` names the state base once more; `EvalSymlinks` of it is the cost.
- A caller who kept files of their own inside `…/mrw/` under the state home cannot serve them
  through mrw; the directory is mrw's by name.

## Out of Scope

- A state directory the caller owns a file inside of under the same name (permanent: boundary: `<state home>/mrw` is mrw's by name; a file of the caller's belongs anywhere else)
- Skipping the directory before walking into it (permanent: fact: the walk refuses each file inside at the boundary; a skip would be a second rule for the same fact)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A test or tool that reads mrw's state through mrw | Low | Low | the state is read through `internal/state`, never through `Resolve` |
| The extra resolution per `Resolve` on a large walk | Low | Low | `Resolve` runs once per discovered file, and `InState` resolves the base and the path again and, where their strings differ, stats each ancestor; measured by the review of #238 at about 0.1 s per 5,000 files |

## Rollback

Revert the task. No state format changes.

## Follow-ups

- [x] Release with ADR-076 and ADR-078 to ADR-080 as v1.27.0 — tagged at `7058767` (#241), Status #242.
