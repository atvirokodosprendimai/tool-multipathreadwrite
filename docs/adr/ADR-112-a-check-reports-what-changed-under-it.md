# ADR-112: A check reports what changed under it

**Status:** Accepted
**Accepted:** 2026-10-01 by Zy — "work on the deffered ones", on the ADR-108 deferrals in `docs/adr/BACKLOG.md` "From ADR-108", of which B5 is this record's scope. The record's text was drafted after that instruction and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-054, ADR-075, ADR-092, ADR-109, ADR-111
**Governs:** `internal/writer/drift.go`, `cmd/mrw/main.go`, `docs/receipts.txt`, `AGENTS.md`, `scripts/contract.sh`
**Enforced-by:** `internal/writer/drift112_test.go::TestDriftNamesAFileChangedAfterTheWrite`
**Served-path change:** A CLI `write` whose check ran names each file the write touched that changed while the check ran — `drift: <path> changed while the check ran`, and `drift` in the `--json` receipt — so a verdict about a tree that moved under it is not read as a verdict about the write. Advisory: exit codes keep their meanings.

## Context

A write's check runs after the write lock is released (ADR-075; `internal/writer/writer.go:73`), so another writer
— or the check itself: `mrw check` is not read-only, and a code generator or formatter rewrites files — can change
a file the write just landed while the check runs. The check's verdict is then about a tree that is no longer the
write's, and nothing said so (B5 in BACKLOG "From ADR-108"; the 2026-10-01 Codex design review).

**Audit of the class** — *a verdict mrw reports about files after something else could have run*: `mrw read --grep
'check\.Run\(' --exclude '*_test.go' cmd internal` — **2** calls: the write's check (`cmd/mrw/main.go:1455`) and
`mrw check` on its own (`:2078`). Only the first has a set of files whose content mrw knows: the write's receipt
carries each written file's `sha_after`. `mrw check` alone wrote nothing to compare against, and MCP never runs a
check; `--then` steps run after the check and are out of scope here.

## Existing Primitives Audit

- **`FileResult.SHAAfter`** (`internal/apply/apply.go:142`) — the sha256 of the bytes each written file was given.
- **`seen.SHA`** (`internal/seen/seen.go:505`) — the same hash of a file's bytes.
- **`regular.Open`** (ADR-109) — reads a file without blocking on one swapped for a FIFO.

## Decision

1. `writer.Drift(root, res)` returns, sorted, each file the write touched — written, not removed, with a
   `sha_after` — whose bytes no longer hash to it, or that is gone or no longer a regular file. It looks at the
   rename's destination and at a symlink's target, where the bytes went.
2. The CLI write calls it once the check has run, and reports each path: a `drift:` line after the check's verdict,
   and `drift` in the `--json` receipt (absent when empty). The wording is "changed while the check ran", not
   "another writer": the check may have changed it itself.
3. It is advisory: the exit code is the check's, and nothing is undone.
4. `write drift` is appended to `docs/receipts.txt` — the first key added under ADR-111.

## Alternatives Considered

- **Fail the write (exit 3) on drift** — rejected: a formatter in the check rewrites files on purpose, and the check
  passed; turning that red would make a working setup fail.
- **Watch the files during the check** — rejected: a before-and-after hash answers the question at the cost of one
  read per written file; a watcher is a subsystem.

## Component / Boundary Impact

`internal/writer` (T1) and `cmd/mrw` (T2), neither an engine package. Every engine package stays byte-identical;
`go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `writer.Drift` | the files that changed since the write | T1 | T2 |
| CLI `write` with a check | `drift:` lines and `drift` in `--json` | T2 | callers |
| `docs/receipts.txt` | `write drift` | T2 | ADR-111's tests |
| `AGENTS.md` | says so | T2 | readers |
| `scripts/contract.sh` | §211 (T2) | T2 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `writer.Drift` | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2.

## Consequences

- **Positive:** a green check over a tree that moved under it says so.
- **Negative:** one read and hash per written file after each check.
- **Neutral:** exit codes and every other receipt key are unchanged.

## Out of Scope

- B1 and B2 (deferred: `docs/adr/BACKLOG.md` "From ADR-108")
- `mrw check` on its own, MCP, and `--then` steps (permanent: boundary: `mrw check` wrote nothing to compare against, MCP runs no check, and steps run after the check this reports on)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a check that formats files reports drift every time | Medium | Low | advisory only; the line says what changed, which is what such a setup wants to know |

## Rollback

Revert T1–T2. The `drift` key disappears with it, which ADR-111 would call a removal: revert before a release ships it.

## Follow-ups

- None — the record carries no open follow-up.
