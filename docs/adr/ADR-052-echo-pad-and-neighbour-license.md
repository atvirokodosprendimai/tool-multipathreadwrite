# ADR-052: Echo pad is opt-in; a multi-line replace needs a served line after End

**Status:** Accepted
**Accepted:** 2026-09-13 by M — *"echo, license"*
**Date:** 2026-09-13
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-035, ADR-048, docs/adr/BACKLOG.md
**Governs:** `internal/apply/apply.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/schema.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/apply_test.go::TestAMultiLineReplaceWithoutAServedLineAfterEndWritesNothing`
**Invalidates:** ADR-048 — the clause of its Decision reading "Wrap-tail stays teaching" for the neighbour-license half only. Echo does not invalidate that clause: a pad is visibility, not a checker.
**Served-path change:** `mrw write --echo-pad N` (MCP `mrw_write.echo_pad`, default 0) prints N lines after an applied body so a surviving closer is visible; the hunk stays `ok`. A multi-line `replace` whose ledger does not already cover End+1 fails, siblings `skip`, nothing is written (ADR-001).
**Notes:** M said *"echo, license"* after the safety ranking. That arms both halves in one record. 4096 stays. 019 A stands. No parser. No default echo. No AST. #166 shipped first as contract §86; this record's rows are §87 (license) and §88 (echo).

## Context

**The class this record governs.** Every write surface that applies a plan hunk or reports a write receipt, plus the ledger check that licenses a multi-line replace. Enumerated 2026-09-13 with

```
git ls-files internal/apply/apply.go cmd/mrw/main.go internal/mcp/tools.go internal/mcp/mcp.go internal/mcp/schema.go scripts/contract.sh
```

Six tracked files (1+1+1+1+1+1). Members left out: `insert-after` / `insert-before` / `delete` / `create` (the license is replace-only; insert neighbour falls out), Shared() (4096; teach on `write --help`), a target-syntax parser (ADR-048 still refuses that).

ADR-048 recorded wrap-tail as teaching: mrw puts the lines you gave where you said, and a surviving closer sits below the body by construction. BACKLOG (2026-09-06 field reports) left a padded write echo as an open cost question: opt-in, not a default, not a checker. M named both.

The license narrows "stays teaching" for one ledger fact: a multi-line replace without a served line after End is the wrap-tail miss as an engine refuse. The echo does not: it prints lines and understands none of them.

## Existing Primitives Audit

- **`seen.Observation.Covers`.** Reused. The neighbour is one more line span, not a new ledger.
- **`apply.Options.Seen` / `Force`.** Reused. A nil ledger still disables every ledger check (engine tests construct file and hunks together). Force still waives unread *hunk* lines. Force does **not** waive the neighbour when an observation exists.
- **ADR-035 `anchor=`.** Unchanged. The neighbour is a different miss (the closer below, not the first line of the range).
- **`mrw read -C` / `A,+N`.** Already serve extra lines. The license asks that End+1 was served, however it was served.
- **A syntax checker.** Audited and rejected (ADR-048). Echo is not one.

## Decision

Two arms, one record.

**1. Opt-in padded write echo.** `--echo-pad N` on `mrw write`, and `echo_pad` on the existing `mrw_write` tool. Default 0. Negative is usage (exit 2). After an applied replace or insert, the receipt carries N lines after the new body, numbered as they sit in the written file. A pad that shows a closer does not fail the hunk — visibility, not refuse. Quiet still hides `ok` lines and their pad.

**2. Context-license.** When a ledger observation exists, a `replace` whose resolved `End > Start` and whose `End` is not the last line of the file is refused unless that observation already covers End+1. Single-line replace is unchanged. Insert neighbour falls out. Line spans only. A nil ledger skips the check (same as today's unread check). `--force` does not waive a partial observation.

**EOF rule (pinned):** when End is the last line, skip the neighbour license. There is no End+1, and requiring Start-1 would be a different miss (orphaned lines above). Do not invent that rule here.

The licence is the engine half of wrap-tail. Echo is not.

## Alternatives Considered

- **Default echo on every write.** Rejected: BACKLOG already rejected that on cost; M's quote armed an opt-in flag, not a default.
- **Require Start-1 at EOF.** Rejected: that is the short-address orphan above, not wrap-tail. Skip at EOF instead.
- **License insert neighbours.** Rejected: insert does not consume a range whose closer can survive below a replaced body.
- **`--force` waives the neighbour.** Rejected: Force already means "edit lines you claim to know". The neighbour is a line you are not editing; waiving it recreates teaching-only wrap-tail.
- **A parser / AST / Blade-HTML-YAML checker.** Rejected: ADR-048. 4096 stays. No AST.

## Component / Boundary Impact

`internal/apply` owns the licence and attaches the pad to `HunkResult`. `cmd/mrw` selects `--echo-pad`. `internal/mcp` adds `echo_pad` on the existing write tool — no third tool (ADR-044). No architecture doc exists; no new bounded context.

C4: same write container. The ledger is the same trust boundary ADR-002 already named.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `apply` neighbour check | multi-line replace without End+1 fails when a ledger observation exists | `internal/apply/apply.go` | CLI; MCP; contract §87 |
| `mrw write --echo-pad` | opt-in pad; default 0; negative is usage | `cmd/mrw/main.go` `writeCmd` | CLI callers; contract §88 |
| `mrw_write.echo_pad` | same number on the existing tool | `internal/mcp/tools.go` / `mcp.go` | MCP hosts; contract §88 |
| `HunkResult.Echo` | numbered lines after the body, omitempty | T2 | `report`; MCP receipt; `hunks.echo` description |
| contract §87 | unread-neighbour FAIL+skip writes nothing; paired with served End+1 and single-line | `scripts/contract.sh` | `adr-verify`, CI |
| contract §88 | `--echo-pad 1` shows the line after the body and stays `ok`; default 0 prints no pad | `scripts/contract.sh` | `adr-verify`, CI |

No exit-code change: neighbour refuse is 1 (hunk failed, nothing written). Negative pad is 2.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| neighbour license in `Apply` (T1) | T1 | T2, T3 | No — T2 adds a field beside it |
| `HunkResult.Echo` / `--echo-pad` (T2) | T2 | T3 | No — teaching names the flag |

## Implementation

See `docs/adr/ADR-052-echo-pad-and-neighbour-license/tasks/README.md`.

## Consequences

- **Positive:** wrap-tail's common miss is a refuse, and a caller who wants the closer visible can ask for it without making every write pay.
- **Negative:** a ranged read of exactly Start-End no longer licenses a multi-line replace. The caller must serve End+1 (or the whole file).
- **Neutral:** Shared() still teaches "read on past the range". The engine now also refuses the exact-range case. Echo is not a checker.

## Out of Scope

- A target-syntax parser / AST / per-language closer checker (permanent: boundary: ADR-048 still holds; echo understands no tokens)
- Default echo (permanent: boundary: opt-in; cost was why BACKLOG rejected a default)
- Requiring Start-1, including at EOF (permanent: boundary: that is a different miss; EOF skips)
- Licensing insert / delete / create by a neighbour line (permanent: boundary: replace-only)
- `--force` waiving a partial observation's neighbour (permanent: boundary: Force is for hunk lines, not the closer)
- Cargo MCP tools / a third write tool (permanent: fact: ADR-044 stays two tools; citation: file `docs/adr/ADR-044-mcp-cargo-stays-two-tools.md:7`)
- Raising 4096 / rewriting Shared() (permanent: boundary: teach on `write --help`)
- Reopening ADR-019 B/C (permanent: fact: pick A stands; citation: file `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md:106`)
- Merging #166 / ADR-042 on this branch (permanent: fact: #166 shipped §86 independently; this record's rows are §87–§88; citation: file `scripts/contract.sh:5238`)
- Indent-reparent with no token after the body (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Existing CLI tests that read exactly Start-End for a multi-line replace go red | Med | Med | Licence fires only when a ledger observation exists; engine tests keep a nil ledger |
| Echo is mistaken for a checker | Med | High | Decision and receipt stay `ok`; contract §88 asserts `ok` with a closer in the pad |
| Shared() rewrite blows 4096 | Low | High | Do not touch Shared(); teach on help |
| EOF skip looks like a hole | Med | Med | Pinned in Decision; T1 names the last-line case |

## Rollback

Revert the branch. Drop `--echo-pad` / `echo_pad`. The neighbour refuse is the behaviour change; callers who already read past the range keep working.

## Follow-ups

- [ ] Indent-reparent with no token after the body stays BACKLOG (YAML `when:` class).
