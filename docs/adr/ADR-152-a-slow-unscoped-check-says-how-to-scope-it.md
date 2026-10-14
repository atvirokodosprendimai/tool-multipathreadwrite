# ADR-152: A slow check that ran the whole project says how to scope it

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — approved the plan `glimmering-frolicking-harp` ("ADR-152 — a slow whole-project check says how to scope it") and set the goal "implement this plan end to end".
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-054, ADR-061, ADR-087, ADR-111, ADR-113, ADR-124
**Governs:** `internal/check/check.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `docs/receipts.txt`, `README.md`, `AGENTS.md`
**Enforced-by:** `internal/check/slow152_test.go::TestASlowUnscopedCheckSaysHowToScopeIt`
**Invalidates:** none — checked: the check's command, verdict, exit code and duration are unchanged; ADR-111's receipts gain one key, appended
**Served-path change:** a check that ran the whole project's command for a write or a `mrw check` that named paths, and took 60 s or more, adds one `hint:` line naming `scoped_check` (CLI receipt and `mrw_write` text) and a `hint` key in the check's JSON

## Context

The 14-session survey (2026-10-09, `docs/adr/BACKLOG.md`, "The default check after a write takes minutes in several repos, so `--no-check` goes on every call"; four sessions) found callers turning the default check off rather than narrowing it. A write's check runs the project's `scoped_check` when one applies to the edited paths (ADR-054, ADR-061, ADR-124) and the whole `check` otherwise. The callers who reached for `--no-check` were paying for the whole command, and nothing told them a narrower one could be declared: `check.Run` computed whether the scoped form applied and dropped it (`_ = scoped`, `internal/check/check.go`).

**Audit of the class** — *where a check's verdict is shown to a caller*: `mrw read --grep 'reportCheck\(|checkReport\(' --exclude '*_test.go' cmd/ internal/` names `reportCheck` (the CLI write receipt and `mrw check`, `cmd/mrw/main.go`) and `checkReport` (the `mrw_write` text, `internal/mcp/tools.go`). Both print the line. The JSON receipts of `write`, `check` and `mrw_write` carry `check` as `check.Result`, so one field reaches all three.

## Existing Primitives Audit

- `command()` in `internal/check/check.go` already returns whether the scoped form applied; `Result.DurationMS` already holds the time.
- `docs/receipts.txt` and `cmd/mrw/receipts111_test.go` hold every receipt key (ADR-111).

## Decision

1. **The hint.** `check.Run` sets `Result.Hint` when all of these hold:
   - the check ran (a process exited, `Ran`);
   - the command was the whole project's: no scoped form applied to the paths;
   - the caller named paths — a write's edited files, or `mrw check PATH…`. `mrw check --full` asks for the whole command and gets no hint;
   - it took `check.SlowAfter` or longer, 60 s.
2. **The words** name the cause and the remedy:
   - no `scoped_check` declared: `the check took <duration> running the whole project's command; declare "scoped_check" in .quality-harness.json to check only the files a write touched, e.g. "pytest {dirs}" ({files} and {dirs} work in any language, {packages} in Go)`;
   - a `scoped_check` that could not name these paths: `the check took <duration> running the whole project's command, because scoped_check "<template>" could not name these paths ({packages} names Go packages only); a template with {files} or {dirs} runs scoped for any language`.
3. **Where it shows.** One `hint: …` line under the CLI check verdict (`reportCheck`, so `write` and `mrw check` both), one in the `mrw_write` text after the check's tail, and `hint` in the check's JSON (receipt key `check hint`, `write check.hint`, `mcp_write check.hint`, appended to `docs/receipts.txt`). Not counted in the write receipt's `hints`: that count is ADR-119's shape hints about hunks, and a check hint is about the project's configuration.
4. **60 s** is where the survey's complaint begins ("takes minutes"); a check under a minute is cheaper to wait for than to configure. `SlowAfter` is a package variable so tests need not wait a minute; nothing outside tests sets it, and the binary takes no flag or variable for it.

It fails if a slow whole-project run for named paths prints no hint, or if a scoped run, a fast run, a `--full` run or a run that never started prints one.

## Alternatives Considered

- **Count it in `hints`.** Rejected: ADR-119's `hints` counts hunks whose shape looks wrong, and its summary line ("1 hint") would then say a hunk was suspect when none was.
- **Hint on every unscoped check, whatever its time.** Rejected: a project whose whole check takes seconds has no reason to configure anything, and the line would be noise on every write.
- **A configurable threshold.** Rejected for now: no caller asked for one, and a flag is a contract.

## Component / Boundary Impact

`internal/check` gains one field and one variable; `cmd/mrw` and `internal/mcp` print it.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `check.Result.Hint`, JSON `hint` | new, omitted when empty | T1 | `write --json`, `check --json`, `mrw_write` |
| CLI receipt | one `hint:` line | T1 | the caller |
| `mrw_write` text | one `hint:` line | T1 | the MCP host |
| `docs/receipts.txt` | three keys appended | T1 | ADR-111's test |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** a caller paying minutes per write learns the one setting that removes the cost.
- **Negative:** one more line on a slow receipt.
- **Neutral:** nothing changes for a project with a working `scoped_check`.

## Out of Scope

- Splitting `mrw stats` refused-at-apply by cause (permanent: boundary: ADR-087 types only the parser-mirrored refusals and keeps the rest untyped by decision, so a split would read "not-read" against "unknown"; citation in ADR-087's Out of Scope)
- A threshold the caller can set (permanent: boundary: Decision 4; no caller asked for one)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The hint fires on a scoped run | Low | Low | the test runs a scoped config past the threshold and requires no hint |

## Rollback

Revert T1: one field, one variable, two print sites, three receipt keys.

## Follow-ups

None.
