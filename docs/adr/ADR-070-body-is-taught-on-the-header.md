# ADR-070: `body=` is taught on the header, and a body line that is a `body=` is refused

**Status:** Accepted
**Accepted:** 2026-09-25 by M — *"Accepted"*
**Date:** 2026-09-25
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-015, ADR-060, ADR-062, ADR-063, docs/blind/blind-04-result.md, docs/adr/BACKLOG.md
**Governs:** `internal/plan/plan.go`, `internal/ingest/applypatch.go`, `internal/guide/guide.go`, `internal/mcp/instructions.go`, `scripts/blind-score.py`, `scripts/contract.sh`
**Enforced-by:** `internal/plan/bodyline_test.go::TestABodyLineThatIsABodyCountIsRefused`
**Invalidates:** none — checked
**Served-path change:** `mrw instructions` and the MCP handshake show `body=` on a worked `@@` header. A hunk with NO `body=` count whose first body line begins `body=` is refused, exit 2 as a plan that does not parse, nothing written. Today that line is written into the file as content. A hunk that counts its body (`body=<n>` on the header) still writes such a line literally, and the apply_patch and search_replace compilers now count any body whose first line begins `body=`.

## Context

**What was observed.** Blind reading 04 (`docs/blind/blind-04-result.md:59-78`, 2026-09-24) ran
Haiku and Sonnet with nothing but `mrw instructions` in context. Four of seven trials put `body=` on
a plan line of its own, under the header: h2 7 times, h4 4, s2 3, s3 2. Three of the four wrote
`body=N`; h4 wrote `body=` and content on one line (`body=status: x`, 3 times; `body=status: done`,
once). mrw read that line as
body and wrote it; s2 committed `body=1` into `docs/meta.yaml`. Its task 8 took 10 Bash calls, and
six of them (seven mrw invocations) went to noticing and undoing the line. That cost is part of why
both models FAILED the 20-call budget, with every answer within the accuracy bar.

**Why.** `mrw instructions` says what `body=` means (`internal/guide/guide.go:46-51`) but never shows
it on a header; its only header-shaped mention is `empty is body=0` in prose. The MCP handshake's
worked plan (`internal/mcp/instructions.go:39-46`) carries no `body=`. The one surface that shows it
inline, `mrw write --help` (`cmd/mrw/main.go:825`), is banned in the blind prompt. And no refusal
fires: a hunk without a count takes every non-header line as body (`plan.go:265-268`).

**The scorer.** Reading 04 ran `scripts/blind-score.py` unchanged on purpose. BACKLOG (`:301-319`)
re-deferred its known misreads to "before reading 05": `\`-continued newlines, heredoc bodies,
`command cat`, `$(cat …)`, `env mrw`, a quoted banned word after `;`, a non-object final JSON fence,
a wrongly typed answer, and no minimum mrw-call count.

## Existing Primitives Audit

- **The overcount guard** (`plan.go:239-245`, ADR-015 family): refuses a line inside a COUNTED body
  that is a valid header, and names `raw=true`. The new guard is its mirror for an UNCOUNTED body.
- **`raw=true`** is refused without `body=` (`plan.go:421-424`), so it cannot be the escape here. The
  escape is the count itself: a counted body is the caller saying exactly what the content is.
- **The `mrw instructions` test** (`internal/guide/guide_test.go:42`) already requires the token
  `body=`; it does not require a header carrying it.

## Decision

1. **Teach it on the header.**
   - `guide.go`: one worked line, `@@ a.go 12-14 replace anchor="func A" body=3`, beside the
     line-count sentence.
   - `examplePlan` in `internal/mcp/instructions.go`: `body=4` on its first hunk.
   - AGENTS.md §2: the example gains `body=`.
2. **Refuse the misplaced count.** In `plan.Parse`, when a hunk has no `body=` count and its first
   body line matches `^\s*body=` (both reading-04 shapes: `body=1` and `body=status: x`), record
   an error:
   `line N: "body=3" sits under the @@ header; body= belongs ON the header line: @@ <path> <addr> <op> body=<n>. To write this line as content, count the body on the header.`
   The two foreign-format compilers (`internal/ingest`, `emit`) count any body whose first line
   begins `body=`, so a plan mrw generates never hits the guard.
3. **Fix the scorer** for each BACKLOG shape, each pinned by a synthetic transcript. Pre-register
   in BACKLOG, before reading 05's first trial, the one criterion change: at least one mrw call per
   non-void run.
4. **Amended 2026-09-25 after the Codex review of v1.25.0 (T4).** The T3 scorer counted
   `command -v mrw` (a lookup) as a call and voided `command -v cat`; it read `env -u VAR mrw` as
   running `VAR`; it dropped a wrapper's `--help`; and it discarded every heredoc body, hiding a
   `$(cat f)` that an UNQUOTED heredoc runs. Now `command -v`/`-V` runs nothing, a wrapper's option
   operands are skipped, `--help` is checked on the unstripped segment, and the `$(…)` and backticks
   of an unquoted heredoc body are scanned. Re-scoring readings 03 and 04 moved nothing.

**What would make this decision fail:** a file whose first line really begins `body=` (an `.env`,
an `.ini`), edited by a hand-written hunk with no count. It is refused, and the message names the
escape (count the body).

## Alternatives Considered

- **Teaching only.** Rejected with M, 2026-09-25: the failure writes into the caller's file and
  reports `ok`. A silent write is the thing mrw exists to prevent, so it earns a guard.
- **Accept `body=N` on its own line as a header option.** Rejected: two spellings of one option,
  and a plan that means "write this line" could no longer say so.
- **Refuse `body=` anywhere in an uncounted body.** Rejected: only the first line is where the
  observed mistake lands; deeper lines are ordinary content far more often.

## Component / Boundary Impact

`internal/plan` is an engine package, and this record owns the guard. `internal/ingest` (the
compilers' `emit`), `internal/guide`, `internal/mcp` and `scripts/blind-score.py` are not engine
code. Byte-identical: `internal/read`, `internal/apply`, `internal/seen`, `internal/check`,
`internal/state`, `internal/lines`, `internal/iter`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw instructions`, MCP handshake | a worked header with `body=` | T1 | every caller; blind reading 05 |
| plan parse | an uncounted hunk whose first body line begins `body=` is refused, exit 2 (the plan does not parse) | T2 | CLI `write`, MCP `mrw_write` |
| `--format=apply_patch`, `search_replace` | a compiled body whose first line begins `body=` is counted | T2 | CLI + MCP `mrw_write` |
| `scripts/blind-score.py` | the BACKLOG misreads fixed; minimum one mrw call | T3 | blind reading 05 |
| contract §132, §133 | T1, T2 | T1, T2 | CI, `adr-verify` |

## Inter-task Contracts

None. T1 and T2 touch disjoint files, and T3 is the bench tooling.

## Implementation

See `docs/adr/ADR-070-body-is-taught-on-the-header/tasks/README.md`.

## Consequences

- **Positive:** the reading-04 mistake is refused with a message that shows the right line, instead
  of landing in a file.
- **Negative:** a hunk without a count cannot start its content with a `body=` line; it has to count
  its body.
- **Neutral:** no exit code changes meaning; the plan grammar is unchanged.

## Out of Scope

- Blind reading 05 itself: six or more headless `claude -p` runs, launched on M's go after this record ships (deferred: docs/adr/BACKLOG.md)
- `anchor=`, `lines=` or `sha=` on a line of their own (permanent: boundary: not observed in readings 03 or 04; a guard needs the input that breaks the tool)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A real file starts with `body=<n>` and an uncounted hunk edits it | Low | Low | refused, exit 2 as a plan that does not parse, nothing written; the message names the escape |
| The worked header teaches a model to copy `body=3` blindly | Med | Low | an overcount is refused (ADR-015), and an undercount leaves stray text that is refused (ADR-060) |
| A scorer change moves a past verdict | Low | Med | re-score readings 03 and 04 with the fixed scorer and record any change in their result files |

## Rollback

Revert the guard, the two teaching lines and the scorer. Nothing persistent moves.

## Follow-ups

- [ ] Freeze `docs/blind/blind-05-plan.md` and run reading 05 on the build that ships this record, on M's go.
- [ ] Release with ADR-069 as v1.25.0; `am_update_skill("mrw")`.
