# ADR-202: A read of standard input can serve the lines that do not match

**Status:** Proposed
**Drafted for:** Zy, 2026-11-19. The count below was reported to Zy with the question "Should I draft it?"; Zy had set the session to continue without pausing (*"go on, do not stop"*, then a standing goal to close the gaps and look for other improvements) and has not answered that question. This is a draft on that standing instruction. It is not accepted, and nothing here merges before Zy reads it.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-111, ADR-140, ADR-146, ADR-161, ADR-164, ADR-165, ADR-173, ADR-179
**Supersedes:** none. ADR-164 Decision 5 ("`-v` … there is none") stands for a read of files.
**Governs:** `cmd/mrw/stdin.go`, `cmd/mrw/main.go`, `cmd/mrw/foreignflag.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `cmd/mrw/invert202_test.go::TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch`
**Served-path change:** `cmd | mrw read --grep PATTERN --invert -` serves the lines of standard input that do not match PATTERN, numbered as in the input. `--invert` anywhere else is refused, exit 2. The refusal of `--grep -v PATTERN -` names this form.

## Context

**What callers type.** mrw narrows a command's output with `cmd | mrw read --grep P -` (ADR-161), so an agent does not leave mrw for `grep`, `sed -n` or `head`, and keeps the command's exit code out of a pipe. `grep -v`, dropping the noise and keeping the rest, has no form: ADR-164 Decision 5 gave `-v` an honest pointer that there is none.

**The count**, exploratory and not pre-registered: on 2026-11-19 a one-off scan of Claude Code transcripts on one Mac, outside this repository's own sessions, printing aggregates only. 36,979 shell calls ran mrw. 160 were refused with "- reads standard input, and a read of it names nothing else". 148 of those had the shape `--grep -X PATTERN -`, where `--grep` took a grep flag as its pattern: `-v` 109, `-i` 29, `-o` 6, `-c` 3, `-E` 1. The scan was not kept: it is a tally of refusal shapes, with no bar written before it, and this record says so wherever it leans on it. v1.90.1 made that refusal name the flag (PR 669). For `-v` it can only say there is no such thing.

**What it costs today.** Each of the 109 calls was refused, exit 2, and the caller had to get the lines another way. What they did next was not counted.

**Why standard input only.** All 109 were reads of standard input. A read of files serves lines to license them (ADR-002, ADR-005); "every line that does not match" of a file is most of the file, and a caller who wants that reads the file. Standard input licenses nothing and records nothing (ADR-161), so an inverted match there changes no ledger and no licence.

**Audit of the class**: *a way to select lines of a read by a pattern.* From `mrw read --help` and `cmd/mrw/main.go`:
- `--grep P` on files and directories: the lines that match, with `-C`, `--count`, `--stat`, `--max-matches`. Unchanged.
- `--grep P -`: the lines of standard input that match. Unchanged.
- a `/pattern/` address in a spec, `-:/x/` included: spans from a match. Unchanged.
- `--ast-grep`: by syntax, files only. Unchanged.
- `--grep P --invert -`: decided here.
- `--invert` with files, with no `--grep`, or with an address: refused here.

## Existing Primitives Audit

- `stdinRead` (`cmd/mrw/stdin.go`) builds one `read.Spec` for standard input and serves it through `read.Run`, the engine every read uses. A spec's ranges may be line numbers, so the lines that do not match can be handed to the engine as numbered runs. No engine package needs to change.
- `read.Run` with `NextLine` serves the line after each range, so that a replace of exactly the lines read is licensed (ADR-161). A stream is never written; for an inverted match that line is by construction one the caller asked not to see, so it is off there.
- `--max-lines`, `--max-cols`, `--max-chars` and `-N` act on what is served, whatever selected it. `-C` and `--max-matches` are defined on matches.
- ADR-146's kinds: a read that exits 1 ends `mrw: no match: …` when a search found nothing.

## Decision

1. **`mrw read --grep PATTERN --invert -` serves the lines of standard input that do not match PATTERN**, in order, numbered as in the input, each run of consecutive such lines under its `@@ A-B` line. It records and licenses nothing, like every read of standard input.
2. **What it works with.** `--max-lines`, `--max-cols`, `--max-chars`, `--fold`, `-N` and `--no-numbers`, as on any served lines. It does not serve the line after a run.
3. **What is refused, exit 2, with a sentence naming the form that works.** `--invert` with no `--grep`; with anything but `-` to read (files and directories: a read of files serves the lines that match, ADR-164); with `-C` or `--max-matches`, which are defined on matches; with `--json-path`. An address (`-:3-9`) beside `--grep` is already refused.
4. **When nothing is left.** If every line matches, the read exits 1 and ends `mrw: no match: every line of standard input matched /PATTERN/`. If standard input is empty it exits 1 and ends `mrw: no match: standard input is empty`. ADR-146's first word is kept.
5. **The pointer.** The refusal of `--grep -v PATTERN -` says `an inverted match of a stream is --grep PATTERN --invert -`. `-v` is not made a flag: `mrw read -v f` and `--grep -v P f` keep ADR-164's sentence, which is still true of files.
6. **What would make this decision wrong.** Nobody uses it: thirty days after the release that carries it, the same tally is taken again over the transcripts since, and if `--invert` was used in fewer calls than `--grep -v … -` was still refused, the instructions are not reaching callers and the flag goes back to Zy. And if the tally of 2026-11-19 was wrong about the shape: T1's first step re-runs it as a committed, pre-registered count before any code, and the record is withdrawn if `-v` on standard input is under 50 of the refusals it finds.

## Alternatives Considered

- **Make `-v` a flag of `read`.** Rejected: `--grep -v P -` would still hand `-v` to `--grep` as its value, so the typed form would not start working, and ADR-164's refusal of `read -v f` and its contract rows would have to change for no caller's benefit.
- **Treat `--grep -v P -` as the inverted match.** Rejected: a flag's value that is sometimes a flag is the guess ADR-140 refuses; `-v` is also a legal pattern.
- **An inverted match on files too.** Rejected for now: no call asked for it, and it serves, and so licenses, most of a file.
- **Leave it, with the clearer refusal of v1.90.1.** The honest alternative. It costs each such caller a second command outside mrw, with that command's exit code behind a pipe.

## Component / Boundary Impact

`cmd/mrw` only (the flag, the standard-input read, the pointer), `internal/guide` (one sentence) and the documents. No engine package changes: `internal/read` serves numbered ranges as it does today.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --invert` | new flag, standard input only | T1 | every caller with a shell |
| the refusal of `--grep -v P -` | names the form | T1 | the caller who typed it |
| `mrw instructions`, `mrw read --help`, `AGENTS.md` | the form, and where it is refused | T1 | every caller |
| `scripts/contract.sh` §346 | the inverted read, paired with the plain one; §345's `-v` assertion follows the new pointer | T1 | the contract gate |

`mrw_read` over MCP takes no standard input (it carries the protocol), so there is nothing to add there.

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** the commonest thing callers typed and were refused for has a form inside mrw.
- **Negative:** one more flag on `read`, valid in one place; four refusals to keep worded.
- **Neutral:** no receipt key, exit code or ledger behaviour changes.

## Out of Scope

- An inverted match on files or directories (permanent: boundary: ADR-164 Decision 5 stands there; a file read serves lines to license them)
- `-o`, match-only output, 6 of the 160 refusals (deferred: docs/adr/BACKLOG.md)
- An inverted match over MCP (permanent: boundary: `mrw_read` reads no standard input)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The tally that motivates this was exploratory | Med | Med | T1's first step is the same count, committed and pre-registered, before any code; Decision 6 withdraws the record under 50 |
| The numbered runs handed to the engine differ from the engine's own line splitting | Low | Med | the lines are split with the function the standard-input read already uses; the test compares against the plain `--grep` on the same input, line by line |
| A caller expects `--invert` on a file | Med | Low | the refusal says what a read of files does and where the flag works |

## Rollback

Before a release, revert T1. After one, the flag is a public contract: it would be refused with a sentence, not removed silently.

## Follow-ups

- [ ] The thirty-day tally of Decision 6.
