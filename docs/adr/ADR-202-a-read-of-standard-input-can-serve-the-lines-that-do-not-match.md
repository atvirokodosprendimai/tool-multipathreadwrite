# ADR-202: A read of standard input can serve the lines that do not match

**Status:** Proposed
**Drafted for:** Zy, 2026-11-19. The tally below was reported to Zy with the question "Should I draft it?". Zy has not answered that question; the session is under Zy's standing instruction to continue without pausing and to look for improvements once the gaps with Claude Code's edit tools were closed. This draft, its count and code on its branch are done on that instruction. The record is not accepted, and nothing here merges before Zy reads it.
**Date:** 2026-11-19
**Amended:** 2026-11-19, before any scanner or code, from an adr-challenger pass and a Codex xhigh challenge (gpt-6.2-sol), each finding checked against the code: the engine does change, by one option, since it prints a note about a replace that means nothing for a stream; the refusals are a complete list that speaks before any dispatch; what stdout holds when nothing is left is stated; the test compares against the lines that match and not against the plain read, which also serves the line after each match; every way to type `-v` is pointed at the form; the count is called what it is, a replication of a tally already seen, and its rule names both wordings of the refusal.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-111, ADR-140, ADR-141, ADR-146, ADR-161, ADR-164, ADR-165, ADR-173, ADR-179
**Supersedes:** none. It amends ADR-164 Decision 5's sentence for `-v` by adding where an inverted match exists, and it is an exception to ADR-161 Decision 10 (the line after each range) for an inverted read.
**Governs:** `cmd/mrw/stdin.go`, `cmd/mrw/main.go`, `cmd/mrw/foreignflag.go`, `internal/read/read.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `cmd/mrw/invert202_test.go::TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch`
**Served-path change:** `cmd | mrw read --grep PATTERN --invert -` serves the lines of standard input that do not match PATTERN, numbered as in the input. `--invert` anywhere else is refused, exit 2. Every refusal that today says there is no inverted match also says where one is. `--grep --invert P` is refused by ADR-140's sentence for a flag taken as the pattern.

## Context

**What callers type.** mrw narrows a command's output with `cmd | mrw read --grep P -` (ADR-161): the lines keep the numbers they had in the output, and the caps (`--max-lines`, `--max-cols`, `--max-chars`) say what they withheld. `grep -v`, dropping the noise and keeping the rest, has no form: ADR-164 Decision 5 gave `-v` an honest pointer that there is none.

**The tally**, exploratory and not pre-registered: on 2026-11-19 a one-off scan of the Claude Code transcripts on one account's Mac, every entry up to the moment it ran, outside this repository's own sessions, printing aggregates only. 36,979 shell calls ran mrw. 160 were refused with "- reads standard input, and a read of it names nothing else". 148 of those had the shape `--grep -X PATTERN -`, where `--grep` took a grep flag as its pattern: `-v` 109, `-i` 29, `-o` 6, `-c` 3, `-E` 1. These are calls, not callers. The scan was not kept and had no bar written first. v1.90.1 made that refusal name the flag (PR 669); for `-v` it can only say there is no such thing. It is the commonest refusal of a read of standard input in that tally, not of mrw: the commonest overall is a multi-line replace with no `anchor=`, which is ADR-035 working.

**What it costs today.** Each of the 109 calls was refused, exit 2. What the caller did next was not counted.

**Why standard input only.** All 109 were reads of standard input. A read of files serves lines to license them (ADR-002, ADR-005); "every line that does not match" of a file is most of the file. Standard input licenses nothing (ADR-161), so an inverted match there changes no ledger and no licence.

**Audit of the class**: *the flags of `mrw read` beside which a line is selected.* `mrw read 'cmd/mrw/main.go:885-1000' -N | mrw read --grep 'Name: +"' -` lists 22 named flags, and `--json-path` is declared just above them: 23, at `3a21d8c3`. Beside `--invert`:

| Disposition | Flags |
|---|---|
| required | `--grep` |
| allowed, acting on the lines served | `--max-lines`, `--max-cols`, `--max-chars`, `--fold`, `--no-numbers` (`-N`) |
| refused by this record | `--context` (`-C`), `--max-matches`, `--no-comments` |
| already refused on standard input by `stdinRead` | `--sub`, `--quiet`, `--stat`, `--count`, `--files-from`, `--ast-grep`, `--exclude`, `--no-ignore`, `--json`, `--if-changed`, `--outline`, `--if-served` |
| already refused beside `--grep` | `--json-path`, `--rewrite` (which needs `--ast-grep`) |

Other ways a line is selected by a pattern, unchanged: a `/pattern/` address in a spec; `--ast-grep`; `mrw write --grep P --sub R` and `mrw read --sub`, which select the lines a plan changes.

## Existing Primitives Audit

- `stdinRead` (`cmd/mrw/stdin.go`) builds one `read.Spec` for standard input and serves it through `read.Run`. A spec may carry many numbered ranges; `Run` sorts them and merges touching ones, so the runs of lines that do not match come out as one `@@ A-B` each with the input's own numbers.
- With `NextLine` off, `Run` prints `-- note: a multi-line replace of A-B needs a served line after B` once, for the first run of two or more lines that ends before the input does (`internal/read/read.go`). It is about licensing a write. A stream is never written, so the note is false there, and the inverted read must be able to turn it off.
- A spec with no ranges means the whole input. "Nothing is left" therefore has to be decided before `Run` is called.
- `lines.Split` is what the standard-input read already uses to ask whether any line matches, on the bytes left after ADR-161's trim of PowerShell's trailing CRLF. The engine splits the same bytes the same way.
- `readFlagPatternRefusal` (ADR-140) refuses a `--grep` value that is exactly a flag the command defines. Once `--invert` is a flag, `--grep --invert P` is such a value.
- ADR-146's kinds: a read that exits 1 ends `mrw: no match: …` when a search found nothing, `mrw: incomplete: …` when something asked for was not served; a caller's own cap exits 0 ending `mrw: capped: …` (ADR-173).

## Decision

1. **`mrw read --grep PATTERN --invert -` serves the lines of standard input that do not match PATTERN**, in order, numbered as in the input, each run of consecutive such lines under its `@@ A-B` line. Nothing of the stream is recorded or licensed, like every read of standard input. It does not serve the line after a run (an exception to ADR-161 Decision 10: that line is by construction one the caller asked not to see), and it prints no note about a replace.
2. **How the allowed flags act**, as they do on any numbered lines. `--max-lines` is one budget across the runs: it can shorten a run and name the rest as withheld. `--fold` folds runs of like lines among those served. `--max-cols` cuts a long line from its first column, there being no match to centre on. `--max-chars` bounds the answer. `-N` drops the line numbers and keeps the `@@` lines.
3. **`--invert` is refused, exit 2, before anything is read or dispatched**, in this order, each with one sentence that names `--grep PATTERN --invert -`:
   - with no `--grep`;
   - with anything to read but the one argument `-`: a file, a directory, no argument, `./-`, an address (`-:3-9`), a second argument. For a file the sentence says that a read of files serves the lines that match;
   - beside any flag other than the allowed five and `--grep`, naming the flag: `-C` and `--max-matches` are defined on matches, `--no-comments` on Go files, and the rest are the flags a read of standard input already refuses.
   `--invert=false` is the flag not given.
4. **When nothing is left.** Standard output gets the stream's header line and one `!!` line, and no numbered line. If every line matches: `!! every line matches /PATTERN/`, exit 1, standard error ending `mrw: no match: every line of standard input matched /PATTERN/`. If standard input is empty: `!! standard input is empty`, exit 1, ending `mrw: no match: standard input is empty`. When lines were left and a cap withheld some, the read ends as any capped read does: exit 0 `mrw: capped: …` for `--max-lines`, exit 1 `mrw: incomplete: …; --max-chars N reached` for `--max-chars`.
5. **Every way to type `-v` is pointed at the form.** The sentence for `-v` (ADR-164 Decision 5) becomes `there is no inverted match: --grep serves the lines that match; an inverted match of a stream is --grep PATTERN --invert -`. It is what `mrw read -v …`, `--grep P -v …` and `--grep -v P f` end in, and what the refusal of `--grep -v P -` carries. `-v` is not made a flag.
6. **`--grep --invert P` is a flag taken as the pattern.** ADR-140's refusal answers it on files and on standard input: `--grep took "--invert" as its pattern; put the pattern right after --grep …`. The literal is written `--grep="(?:--invert)"`.
7. **What would make this decision wrong.** The tally of 2026-11-19 being wrong about the shape: T1's first step replicates it with a committed scanner and a stated rule, and the record is withdrawn if `-v` on standard input is under 50 of the refusals it finds. And nobody using the flag: the tally of BACKLOG.md ("Pre-registered for ADR-202's use"), taken thirty days after the release that carries it.

## Alternatives Considered

- **Make `-v` a flag of `read`.** Rejected: `--grep -v P -` would still hand `-v` to `--grep` as its value, so the typed form would not start working.
- **Treat `--grep -v P -` as the inverted match.** Rejected: a flag's value that is sometimes a flag is the guess ADR-140 refuses; `-v` is also a legal pattern.
- **An inverted match on files too.** Not built: no call asked for it, and it serves, and so licenses, most of a file.
- **No engine change, and accept the note.** Rejected: a line of the answer that says something false about a stream is what this tool exists not to print.
- **Leave it, with the clearer refusal of v1.90.1.** The honest alternative. The caller then pipes through `grep -v`, which loses the output's own line numbers and says nothing about what a cap withheld. The command's exit code is behind a pipe either way; `mrw run` is what keeps that.

## Component / Boundary Impact

One engine package, owned here for this change: `internal/read`, one option that turns the replace note off. `cmd/mrw` (the flag, the standard-input read, the refusals, the `-v` sentence), `internal/guide` (one sentence in the full instructions, none in the core) and the documents. `internal/plan`, `internal/apply`, `internal/seen`, `internal/check` and `internal/state` are not changed.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --invert` | new flag, standard input only | T1 | every caller with a shell |
| `read.Options` | an option that drops the replace note | T1 | `stdinRead` |
| the `-v` sentence | says where an inverted match is | T1 | whoever types `-v` |
| `mrw instructions`, `mrw read --help`, `AGENTS.md`, `README.md` | the form, and where it is refused | T1 | every caller |
| `scripts/contract.sh` §346; §345 and §277 keep passing | the inverted read and its pair | T1 | the contract gate |

`mrw_read` over MCP reads no standard input, so there is nothing to add there, and no receipt key changes (ADR-111): a `--json` read of standard input stays refused (ADR-165).

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** the commonest refused read of standard input has a form inside mrw that keeps the output's line numbers.
- **Negative:** one more flag on `read`, valid in one place, with three kinds of refusal to keep worded; one more option in the read engine.
- **Neutral:** no receipt key, exit code or ledger behaviour changes.

## Out of Scope

- An inverted match on files or directories (deferred: docs/adr/BACKLOG.md)
- `-o`, match-only output (deferred: docs/adr/BACKLOG.md)
- An inverted match over MCP (permanent: boundary: `mrw_read` reads no standard input, which carries the protocol)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The tally that motivates this was exploratory | Med | Med | T1's first step replicates it with a committed scanner; Decision 7 withdraws the record under 50 |
| The numbered runs differ from the engine's own line splitting | Low | Med | one function splits for both; byte-level cases for CRLF, a CR-only stream, a final line with no newline and an empty line |
| "Nothing left" serves the whole input | Med | High | decided before `Run`; a mutant removes that and must be killed |
| A caller expects `--invert` on a file | Med | Low | the refusal says what a read of files does and where the flag works |

## Rollback

Before a release, revert T1. After one, the flag is a public contract: it would be refused with a sentence, not removed silently.

## Follow-ups

- [ ] The tally of BACKLOG.md, "Pre-registered for ADR-202's use".
