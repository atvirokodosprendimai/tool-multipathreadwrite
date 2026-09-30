# ADR-102 Tasks

Implementation tasks for ADR-102: a write that landed is reported and counted as landed. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1 |
| 4 | T4 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `writer.MutationOf` and the sixth outcome, `partially_applied`; contract §196 | done | — | `docs/adr/ADR-102-a-write-that-landed-is-reported-and-counted-as-landed/tasks/T1-mutation-and-partially-applied.md` fence |
| T2 | the ledger records a partial commit | done | — | `docs/adr/ADR-102-a-write-that-landed-is-reported-and-counted-as-landed/tasks/T2-ledger-after-a-partial-commit.md` fence |
| T3 | `mrw_write` sends the receipt on a ledger failure; contract §197 | done | — | `docs/adr/ADR-102-a-write-that-landed-is-reported-and-counted-as-landed/tasks/T3-mcp-receipt-on-a-ledger-failure.md` fence |
| T4 | the unreportable message says not to re-run | done | — | `docs/adr/ADR-102-a-write-that-landed-is-reported-and-counted-as-landed/tasks/T4-do-not-re-run.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `writer.MutationOf(res)` (T1) | T2, T3 | T1 first |

## Notes

- Engine go/no-go: ADR-102 owns no engine package; `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted` stay byte-identical.
- Contract sections §196 and §197: §195 is the highest (ADR-101), found with the `sort -k2,2n` recipe on 2026-09-30.
- A partial commit is forced without a seam: content renames commit before path ops, and an unlink in a read-only directory passes validation (ADR-066 deferral) and fails at commit.
