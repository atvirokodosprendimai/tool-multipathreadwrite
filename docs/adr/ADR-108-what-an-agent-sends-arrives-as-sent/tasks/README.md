# ADR-108 Tasks

Implementation tasks for ADR-108: what an agent sends arrives as sent. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Notes |
|------|-------|-------|
| 1 | T1, T2, T3, T4, T5, T6, T7, T8, T9, T10 | independent; one defect each |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | a checkpoint covers only consecutive served lines | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T1-checkpoint-gaps.md` fence |
| T2 | a non-ASCII rune never splits a path | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T2-rune-separators.md` fence |
| T3 | a body file is split like every other text | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T3-body-line-endings.md` fence |
| T4 | foreign formats and body files are bounded | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T4-bounded-loaders.md` fence |
| T5 | the ledger reads back every record it saves | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T5-ledger-record-bound.md` fence |
| T6 | acknowledging a non-regular file returns at once | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T6-ack-non-regular.md` fence |
| T7 | non-UTF-8 content is served unlicensed over MCP | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T7-non-utf8-unlicensed.md` fence |
| T8 | a page counts only what read would serve | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T8-page-count.md` fence |
| T9 | the working set reads back every record it saves | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T9-working-set-record.md` fence |
| T10 | permissions issued under the old checkpoint rules are discarded | done | — | `docs/adr/ADR-108-what-an-agent-sends-arrives-as-sent/tasks/T10-old-permissions.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| none | — | — | independent |

## Notes

- Engine go/no-go: ADR-108 owns `internal/plan` and `internal/seen`; `internal/iter` (T9) is not on the go/no-go list; `internal/read`, `internal/apply`, `internal/state`, `internal/lines`, `internal/rooted`, `internal/check` stay byte-identical.
- Contract sections §202 (T1), §203 (T3), §204 (T8), §205 (T9) and §206 (T10).
