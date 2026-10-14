# ADR-157: A notice says what happened, and only advice that can help

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — picked "Accept (Recommended)", with all eight leads taken as drafted.
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-077, ADR-136, ADR-142, ADR-144, ADR-150, ADR-155
**Governs:** `internal/writer/flow.go`, `cmd/mrw/main.go`, `internal/seen/seen.go`, `internal/mcp/ack.go`, `internal/mcp/tools.go`, `internal/rooted/rooted.go`, `internal/rooted/hardlink.go`, `internal/read/read.go`, `internal/apply/apply.go`, `internal/check/check.go`, `internal/ingest/create.go`, `AGENTS.md`, `README.md`
**Enforced-by:** `cmd/mrw/newer157_test.go::TestANewerLedgersRefusalLeadsWithItsCause`
**Invalidates:** ADR-150's Neutral consequence "the MCP read path prints no notice of its own" (the read's answer text can carry it; only stderr is missing there); ADR-150 Decision 3's wording (the sentence no longer trails "has not been read" but replaces it); ADR-150's "every command that reads the ledger says so once" is kept but qualified for `write`, which says it in a refusal when one carries it and on stderr otherwise; ADR-136 Decision 2's sentence (it claimed a discard on commands that make none) and Decision 3's skip list (help and a bare `mrw` join it) — checked: ADR-144's damage rule, ADR-077's refusal itself and ADR-142's CRLF notes are unchanged
**Served-path change:** a write refused under a newer ledger leads with that cause and drops advice that cannot help, and the CLI says it once; `mrw_read` under a newer ledger says no ack will license; `mrw seen` names the ledger's format and writer; the stale notice claims no discard and is not told by help; the own-state refusal drops the `--root` advice; a create names a second byte order mark; AGENTS.md says how each Windows shell quotes `$["k"]`

## Context

The v1.66.0 Windows battery (2026-10-14; six sessions on Windows 11 build 26200, pwsh 7.6.6, Windows PowerShell 5.1.26100, cmd, Git Bash and MCP; memoryd open-thread `cdebb4c2`) found every expected behaviour held, and eight places where what mrw says is wrong or does not help. None changes what a write does.

1. `mrw seen` prints the state directory and the entries, never the ledger's first line, so ADR-150's writer cannot be checked through mrw: all six sessions read the file with their shell.
2. Under a newer ledger a refused write leads with "t.json has not been read … Run `mrw read t.json` first, or pass --force" (CLI) or "Read t.json with mrw_read first" (MCP); the cause comes after a semicolon (`internal/writer/flow.go:222`). A read cannot help, so an agent that acts on the first clause loops (the MCP session said so).
3. On the CLI the newer-ledger sentence prints twice for one refused write: once from the root `Before` (`cmd/mrw/main.go:177`) and once inside the FAIL reason (five sessions).
4. A read or plan that names mrw's own state directory is refused (ADR-077, by design), then told "read it with --root pointed where you mean" (`internal/read/read.go:527`) or "a plan may only change files under the directory mrw was pointed at" (`internal/apply/apply.go:2505`), when `--root` already points there and no `--root` lifts the refusal (pwsh and cmd sessions).
5. `mrw_read` under a newer ledger hands out checkpoint ids and "This serve licenses NOTHING until you acknowledge it" (`internal/mcp/ack.go:331`), and nothing says an ack will license nothing either; the agent learns it at write time (MCP session). ADR-150 left the MCP read path without a notice because the server has no stderr; the answer text is not stderr.
6. `mrw help` with a v3 ledger prints "the read ledger … has been discarded" and leaves the file as it was. Reproduced on macOS with v1.66.0: the header stayed `#mrw-seen v3`. The root `Before` consults the ledger for every verb but `version`, `instructions` and `stats` (`cmd/mrw/main.go:170-173`), so help, a bare `mrw`, `check`, `iter` and `seen` all print it, and only a read or a write that saves replaces the file.
7. JSON-path addresses (ADR-155) need per-shell quoting the docs do not give. Measured: pwsh 7 passes `'t.json:$["k"]'`; Windows PowerShell 5.1 strips the inner quotes from it and needs `'t.json:$[\"k\"]'` (`--%` does not help); cmd needs `"t.json:$[\"k\"]"`, since without backslashes the C runtime strips them; Git Bash takes single quotes; in pwsh, a double-quoted `\"` reaches mrw with the backslashes. An unquoted `[0]` is a glob that works only while no file matches it.
8. Windows PowerShell 5.1 under code page 65001 with a `$OutputEncoding` that has a preamble pipes TWO byte order marks; `--create` keeps both and its note says "begins with a UTF-8 byte order mark" (`internal/ingest/create.go:46`).

**Audit of the class**: *a sentence mrw prints whose advice does not fit the case it is printed in*. `mrw read --grep 'pointed where you mean|may only change files under|has not been read|has been discarded|byte order mark' --exclude '*_test.go' cmd/ internal/` (run 2026-10-14 on be76e27) names these production sites:
- `read.go:495`, `walk.go:229` and `check.go:533`: a path outside the root; the advice fits; kept.
- `read.go:527`, `apply.go:2505`, `check.go:540`: changed here when the error is the own-state refusal.
- `apply.go:1220`, `1223`, `1259`: the generic not-read refusals; they fit while the ledger is current, and `flow.go` replaces them only under a newer one.
- `seen.go:342`: the stale notice; reworded.
- `seen.go:422`: the damage notice's discard, which `DamageNotice` tells only for read, write and seen; kept (Out of Scope).
- `create.go:46`: the BOM note; counted.
- The ack footers `ack.go:331` and `tools.go:1710` gain the newer-ledger line.
- `tools.go:2332` (`nameTheAck`, found by the cold review) appends "an unacknowledged serve licenses nothing" plus the ack rule to a not-read refusal over MCP; under a newer ledger an ack licenses nothing either, so it is skipped there.
- `ack.go:138` (bytes that are not valid UTF-8: "read and edit the file with the CLI") and `apply.go:1217` (a create over an existing file: "read it and use replace or delete"), both found by the cold review: kept, because the newer-ledger line beside them (Decisions 2 and 3) says why no read licenses.
- The own-state refusal itself is built at `rooted.go:185`, `rooted.go:528`, `hardlink.go:23` and `hardlink.go:30`; all four wrap one sentinel.

## Existing Primitives Audit

- `rooted.RefusedByName` already makes read, apply and check skip the `--root` advice for a refusal no `--root` lifts (ADR-081); the own-state refusal is the same kind and joins it through a sentinel, `rooted.ErrOwnState`.
- `seen.Newer` and `seen.NewerNotice` (ADR-150) say whether the ledger is newer and give the sentence; the write path, the MCP read footers and `Before` reuse them.
- `tellsLedgerDamage` (`cmd/mrw/damage.go`, ADR-144) is the list of verbs that use the ledger; the skip list for help follows the same idea.

## Decision

1. **A refusal under a newer ledger leads with its cause.** When the ledger is newer, a not-read refusal's reason is replaced, not extended: `<path> cannot be edited by this mrw: the read ledger was written by a newer mrw (<writer>); this one is <ours> and leaves it untouched, so nothing it reads licenses a write. Use the newer mrw; mrw version --check-latest names the latest.` It names neither `mrw read` nor `--force`, and its kind stays `refusal.NotRead`. Both surfaces (`internal/writer/flow.go`).
2. **The CLI says it once per hunk, and the root adds nothing to it.** The root `Before` does not print the newer-ledger line for `write`. After the write, the CLI prints it on stderr once only when no hunk's reason carried it — a success, a dry run, a create refused because the file exists, a forced write that failed validation. Every other command that consults the ledger prints it from `Before` as now. Over MCP, `nameTheAck` (`internal/mcp/tools.go:2332`) adds no ack advice while the ledger is newer.
3. **`mrw_read` under a newer ledger says no ack licenses.** Its answer, beside the ack footer and the page footer, carries one line: `-- the read ledger was written by a newer mrw (<writer>): an ack here licenses nothing. Use the newer mrw.`
4. **`mrw seen` names the ledger.** After the state-directory count it prints one comment line:
   - `# ledger format v<N>, written by <writer>` for a current ledger (`unknown` when a v4 or a writerless v5 names none);
   - `# ledger written by a newer mrw (<writer>), format v<N>: left untouched, licenses nothing`;
   - `# ledger in an older format: not used; mrw replaces it the next time it records a read or a write`.

   Nothing is printed when there is no ledger. `seen.Header(root)` returns the format, its number and the writer.
5. **The stale notice claims no discard, and help does not consult the ledger.** `seen.StaleNotice` becomes `mrw: the read ledger was written by an older mrw, or its line endings were changed; it is not used, and the next recorded read or write replaces it. Read the files you mean to edit again.`, 37 words within ADR-136's bound of 40 (`seen.Record` saves only when a call recorded something, `internal/seen/seen.go:592`). `help`, `h` and a bare `mrw` join `version`, `instructions` and `stats` in skipping the ledger.
6. **No `--root` advice on the own-state refusal.** The four sites that build it wrap `rooted.ErrOwnState`; read, apply and check return it without their `--root` sentence, as they do for `RefusedByName`.
7. **A create names every leading byte order mark.** Two or more are named with their count: `the content begins with 2 UTF-8 byte order marks, kept as content (…; Windows PowerShell 5.1 adds both when $OutputEncoding has a preamble and [Console]::InputEncoding does too)`. One keeps today's sentence.
8. **AGENTS.md and README say how to quote a JSON path** in pwsh 7, Windows PowerShell 5.1, cmd and Git Bash, and that an unquoted `[0]` is a glob.

It fails in any of these cases:
- a refusal under a newer ledger names `mrw read` or `--force`;
- the CLI prints the sentence more than once for one refused hunk, or not at all for a write under a newer ledger;
- `mrw_write` under a newer ledger advises an ack;
- `mrw_read` under a newer ledger says nothing of it;
- `mrw seen` omits the writer;
- `mrw help` prints the stale notice;
- the own-state refusal advises `--root`;
- two byte order marks are named as one.

## Alternatives Considered

- **Keep the stderr line on `write` and drop the sentence from the CLI's FAIL reason.** Rejected: the reason is shared by both surfaces, and the FAIL line is the one a caller reads; MCP must keep it.
- **Stop every non-saving command (check, iter, seen) from telling a stale ledger.** Rejected: ADR-136 kept them on purpose, so a caller who runs `seen` after an upgrade learns why it is empty; the reworded sentence is true for all of them.
- **Strip extra byte order marks.** Rejected: ADR-142 keeps a BOM as content because mrw cannot tell PowerShell's from the file's; the note names what came.
- **Print the ledger header as `mrw seen`'s first line.** Rejected: `mrw seen | head -1` is the documented way to find the state directory (contract §71).

## Component / Boundary Impact

Engine packages `internal/seen`, `internal/read`, `internal/apply` and `internal/check` change wording and one predicate; this record owns those changes. `internal/rooted` gains a sentinel. `internal/mcp` adds one footer line. No new flag, receipt key or exit code.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| write refusal, CLI and `mrw_write` | the newer-ledger reason replaces "has not been read" | T1 | the caller, the MCP host |
| CLI stderr on `write` | the newer-ledger line printed after the write, once, only when no refusal carried it | T1 | the caller |
| `mrw_read` answer text | one line under a newer ledger | T1 | the MCP host |
| `mrw_write` refusal | no ack advice while the ledger is newer | T1 | the MCP host |
| `mrw seen` stdout | one `#` line naming the ledger | T2 | the caller |
| CLI stderr, stale notice | new words; not printed for help or a bare `mrw` | T2 | the caller |
| `seen.Header` | new exported function | T2 | `cmd/mrw` |
| `rooted.ErrOwnState` | new sentinel | T3 | read, apply, check |
| `--create` stderr note | counts byte order marks | T3 | the caller |
| AGENTS.md, README.md | the JSON-path quoting paragraph | T3 | agents, people |

## Inter-task Contracts

T2 edits the root `Before` after T1 has, so T2 depends on T1 and the two edits to `cmd/mrw/main.go:170-184` land in order. T3 is independent.

## Implementation

See `tasks/`.

## Consequences

- **Positive:**
  - An agent under a newer ledger is told the one thing that helps, once, on both surfaces, and before it acks.
  - `mrw seen` answers "which mrw wrote this".
  - help is silent.
  - No refusal sends the caller to a `--root` that cannot help.
  - A doubled BOM is visible.
- **Negative:**
  - A plan refused in two files under a newer ledger prints the sentence twice, once per refused hunk: each FAIL line must stand alone.
  - ADR-150 T1's locked test (`cmd/mrw/writer150_test.go`) changes with the refusal's words, and that task is re-locked.
- **Neutral:** the stale notice still prints on check, iter and seen, with words that are now true there.

## Out of Scope

- The damage notice's "so it has been discarded" for a line past the record bound (permanent: boundary: `DamageNotice` is told only by read, write and seen; a read or write that records saves the ledger, and seen's case is ADR-144's and was not reported)
- A Git Bash run without `MSYS_NO_PATHCONV` (permanent: boundary: AGENTS.md already names that trap; the battery ran with it set)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The replaced reason loses the not-read kind, so `stats` stops counting it | Low | Medium | `Kind` stays `refusal.NotRead`, and the T1 test asserts it |
| `help` skipping the ledger hides a real stale ledger | Low | Low | read, write and seen still tell it; help never used the ledger |
| A caller parses `mrw seen` lines and trips on the new `#` line | Low | Low | `seen` already prints `#` lines; contract §71 reads only the first line |

## Rollback

Revert the three tasks. Nothing persistent changes: the ledger format and every receipt key are untouched.

## Follow-ups

None.
