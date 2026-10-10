# ADR-140: a read flag from grep is answered with mrw's spelling

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — "/loop continue delivering, end to end, no dead code", on the survey's flag-friction item ("`--grep -i` shifts the pattern into the path slot", "`--plan` guessed", 4 of 14 sessions as the author's 2026-10-09 synthesis recorded the replies)
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-078, ADR-097, docs/adr/BACKLOG.md
**Invalidates:** None — it appends a sentence to an error that is already an error; exit codes and the refusal itself are unchanged
**Governs:** `cmd/mrw/foreignflag.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `cmd/mrw/foreignflag140_test.go::TestAForeignReadFlagIsAnsweredWithMrwsSpelling`
**Served-path change:** `mrw read` given a flag it does not have but grep, rg or ack do (`-i`, `-n`, `-r`, `-l`, `-A`, `-e`, `--include` and the others in the table) still exits 2 with the same message, and the message now ends with what mrw spells instead, for example `-i`: put `(?i)` at the start of the pattern. The commoner trap, `--grep -i PATTERN`, is not an unknown flag at all: `-i` is taken as the pattern and `PATTERN` as a path, and the read ends `no file matched /-i/` (exit 1, as before); that message now carries the same hint when the pattern is exactly a dash and one table letter. Any other unknown flag, any other pattern, and every other command, is worded as before.

## Context

An agent that knows grep reaches for `-i`, `-n`, `-r`, `-e` or `-A 2` on `mrw read --grep`. mrw has none of them; the parser answers `flag provided but not defined: -i (see: mrw read --help)`, which names the help and not the answer. The survey's sessions met this 4 times and the author's own session met it five times in one day (`-i`, `-e`, a pattern starting with `--`, `-n`, `-A`), each costing a turn to read `--help` and retype. Every such flag has an exact mrw spelling, or a plain reason there is none, and the usage-error handler (ADR-078, ADR-097) already has the flag name in hand.

**Audit of the class** — *a flag a caller types that mrw's `read` does not have and a searcher they know does*: the table in `cmd/mrw/foreignflag.go` is the enumeration, built from grep and ripgrep's most-used options and checked one by one against the built binary on 2026-10-10 (each equivalent below was run, not read from a doc). Flags that have no mrw equivalent and no honest pointer (`-v`, `-o`, `-z`) are left out and get the old message.

## Existing Primitives Audit

- **`usageError` and `subcommandForFlag`** (ADR-078, ADR-097) — already receive every parser rejection and read the flag name from urfave/cli's message prefix; the hint is a second lookup beside the first, with the same fall-through if the message is ever reworded.
- **`read`'s own flags** — `--grep`, `--context`/`-C`, `--no-numbers`/`-N`, `--stat`, `--max-lines`, `--exclude`, `--files-from`; the hints point only at these.
- **Go's `regexp` (RE2)** — `(?i)`, `\Q…\E` and `\b` are its own syntax, run against the binary before they were written here.

## Decision

1. **A table maps a flag name to a sentence.** `mrw read` (only) given one of `i n r R l c count A B e E P F w H include m` appends `; ` and its sentence to the existing message: `flag provided but not defined: -i (see: mrw read --help); case-insensitive: start the pattern with (?i), as in --grep '(?i)PATTERN'`.
2. **Nothing else changes**: the exit code (2), the prefix of the message, every other command, every flag not in the table, and a flag spelled like a subcommand (ADR-097) are as before.
3. **Each sentence names a spelling that works, with the limit it has**: `-i` → `(?i)`; `-n` → numbers are on, `-N` drops them; `-r`/`-R` → a named directory is walked; `-l` → `--grep P --stat`; `-c`/`--count` → no count, `--stat` lists matching files; `-A`/`-B` → `-C N`; `-e` → the value of `--grep`, and `--grep=-PATTERN` for one that starts with a dash; `-E`/`-P` → patterns are Go RE2; `-F` → `\Q…\E`, or escape each metacharacter for a literal that holds `\E`; `-w` → `\b…\b`, whose boundaries are ASCII-only in Go; `-H` → every file has a `==>` header; `--include` → name the directories or pipe a list to `--files-from`, `--exclude` drops; `-m` → `--max-lines N`.
4. **`--grep -X` is answered at the no-match message.** urfave/cli takes the token after `--grep` as its value, so `mrw read --grep -i needle f` searches for the pattern `-i` in the path `needle` and `f` (measured 2026-10-10: `==> needle  REFUSED …` then `no file matched /-i/`, exit 1). When the pattern is exactly `-` and one letter that is a key of the table, the `no file matched` line ends `— <sentence> (mrw took the value after --grep as the pattern)`. A longer pattern, or a letter not in the table, is worded as before.

## Alternatives Considered

- **Accept the flags** (`-i` as an alias for `(?i)`) — rejected: it grows the public grammar for flags whose meaning differs subtly (`-n`, `-r`, `-l`), and every alias is a promise to keep.
- **Print the whole help on an unknown flag** — rejected by ADR-078: help on stdout corrupts `mrw mcp`'s protocol stream, and a page of help is the cost this removes.
- **Fuzzy "did you mean"** — rejected: the names are not near each other (`-i` and `(?i)`), the table is exact.
- **Leave it (the message names the help)** — rejected: five turns in one day on one session, and the survey's four.

## Component / Boundary Impact

`cmd/mrw` only (one new file, one call in `usageError`). No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` usage errors | a spelling hint on a table flag | T1 | CLI callers |
| `scripts/contract.sh` | §241 | T1 | CI Linux |
| `AGENTS.md` | the read section names the table | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** the wrong flag costs one call, not two; the hint is exactly the equivalent a grep user needs.
- **Negative:** the table is a list to keep true; a change to `read`'s flags must change a sentence (a test runs each one's claimed equivalent).
- **Neutral:** stdout, exit codes and receipts are unchanged; a script that matches the old message by its prefix still matches.

## Out of Scope

- Hints for `write`, `check` and the other commands (deferred: docs/adr/BACKLOG.md — no evidence yet; the survey named `--plan` for `write -`, which is a different mistake)
- A flag with no mrw equivalent and no honest pointer, `-v` `-o` `-z` (permanent: boundary: a hint would be a guess)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a hint names a spelling that does not work, or works only with a limit it omits | Low | a caller is sent wrong | `TestEveryForeignFlagHintIsATrueSpelling` runs the runnable claims (`(?i)`, `\Q…\E`, `\b…\b`) against a fixture; `TestTheWordAndLiteralHintsNameTheirLimits` runs the two limits the review found (ASCII-only `\b`, a literal holding `\E`); the rest name flags `read --help` lists |
| urfave/cli rewords its message | Low | the hint silently disappears | the lookup falls through to the old message; `TestAForeignReadFlagIsAnsweredWithMrwsSpelling` fails on the reword |

## Rollback

Revert T1: the message loses its second half. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the deferred item above.
