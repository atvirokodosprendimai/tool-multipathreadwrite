# ADR-151: a guessed write flag and a digit root are named

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — approved the plan `glimmering-frolicking-harp` ("ADR-151 — write-side flag hints") and set the goal "implement this plan end to end", on the survey's flag-friction item that ADR-140 left open ("`-C` before and after the verb, `--plan` guessed for `write -`")
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-069, ADR-078, ADR-097, ADR-140, docs/adr/BACKLOG.md
**Invalidates:** None — it appends a sentence to an error that is already an error, and adds one stderr line to a call whose result is unchanged
**Governs:** `cmd/mrw/foreignflag.go`, `cmd/mrw/shortroot.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`, `internal/guide/guide.go`
**Enforced-by:** `cmd/mrw/flag151_test.go::TestAPlanFlagOnWriteIsAnsweredWithTheArgument`, `cmd/mrw/flag151_test.go::TestAnAllDigitShortRootBeforeTheVerbIsNamed`
**Served-path change:** `mrw write --plan x` (and `--file`, `--input`, `--stdin`) still exits 2 with the same message, which now ends with mrw's spelling: the plan is an argument, `mrw write PLAN`, or `mrw write -` for standard input. And `mrw -C 3 read …` — a short `-C` before the subcommand whose value is all digits — prints one stderr line saying that `-C` there is `--root`, so `3` is the root directory, and that context lines take `-C` after `read`; what the call does and its exit code are unchanged. `--root 3`, a `-C` whose value has a non-digit, and `-C N` after `read` print no such line.

## Context

The 14-session survey (2026-10-09, `docs/adr/BACKLOG.md` "Flag friction") named two mistakes ADR-140 did not take. Two sessions guessed `--plan` for the plan argument of `mrw write`, and got `flag provided but not defined: -plan (see: mrw write --help)`, which names the help and not the answer. Four sessions were caught by `-C`: before the subcommand it is the short form of `--root`, after `read` it is `--context`. Probed on v1.65.0 (2026-10-14): `mrw -C 3 read go.mod:1` takes `3` as the root and ends `REFUSED the root 3 does not exist`, exit 1; and where a directory named `3` exists the same call reads from it with no word at all, which is the worse case because nothing looks wrong.

**Audit of the class** — *a flag a caller types on `write` to name the plan*: `--plan` (named by the survey), `--file`, `--input` and `--stdin` (the same mistake spelled the way other tools name their input), enumerated from `writeCmd`'s own flags (`mrw read --grep 'Name:' cmd/mrw/main.go`, 2026-10-14) so that none of them is a real `write` flag. Left out on purpose: `-f` (read as `--force` by `rm`, `git push` and `cp`; a plan hint there could send a caller who wanted force the wrong way) and `-p` (`patch -p1` strips path components). *A root flag spelled so it reads as context*: the short alias `-C` with an all-digit value, in every form the parser accepts (`-C 3`, `-C=3`, `--C 3`, `--C=3`, all probed 2026-10-14); `-C3` is already refused by the parser.

## Existing Primitives Audit

- **ADR-140's table and `usageError`** — `foreignFlagHint` already receives every parser rejection with the command's full name; a per-command table extends it without a second mechanism. `foreignFlags` stays the `read` table, because ADR-140's locked test ranges over it.
- **`refusePaddedFlagValues`, `flagRole` and `flagKinds`** (ADR-069) — `main` already walks the whole argv before the parser, because a root flag never reaches a subcommand's raw tail; the `-C` check walks it the same way, with the same classifier.
- **urfave/cli v3.11.0** — exposes the flag's value and whether it was set, not which alias was typed (`Command` keeps its parsed args unexported, `command.go:155`), so `--root 3` and `-C 3` cannot be told apart after parsing. That is why the check reads argv.

## Decision

1. **A per-command table.** `mrw write` given `--plan`, `--file`, `--input` or `--stdin` appends `; ` and one sentence to the existing message: `the plan is an argument, not a flag: mrw write PLAN, or mrw write - to read it from standard input`. `mrw read`'s table is ADR-140's, unchanged; every other command and flag is worded as before; a flag spelled like a subcommand (ADR-097) is answered first, as before.
2. **An all-digit short root is named.** Before the parser runs, `main` walks the tokens before the subcommand. A short `-C` (one dash or two) whose value — attached after `=` or the next token — is one or more ASCII digits prints one line on stderr: `mrw: -C before the subcommand is --root, so "3" is taken as the root directory; for lines of context, put -C after read: mrw read -C 3 …`. The line is printed whether or not a directory of that name exists, because the silent case is the one that exists.
3. **Nothing else changes**: exit codes, stdout, the root mrw uses, and every message already printed. `--root 3` (spelled long), a `-C` whose value has any other character, and `-C` after the subcommand print no line.

## Alternatives Considered

- **Accept `--plan PATH` as an alias for the argument** — rejected: it grows the public grammar for one guess, and every alias is a promise to keep (ADR-140's reason).
- **Refuse an all-digit `-C` before the subcommand** — rejected: a directory named `3` is a legal root and a caller who meant it would be refused; a note costs nothing and changes nothing.
- **Make `-C` a separate hidden flag so the parser says which was typed** — rejected: `--root` is read in many places as one value, and splitting it would touch every one of them for a stderr line.
- **Leave it** — rejected: two and four survey sessions, and the silent read of a digit directory.

## Component / Boundary Impact

`cmd/mrw` only: one new file, one table, one call in `main`. No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` usage errors | a sentence for a plan-naming flag | T1 | CLI callers |
| stderr before any command | one line for an all-digit short root | T1 | CLI callers |
| `scripts/contract.sh` | §255 | T1 | CI Linux |
| `AGENTS.md`, `mrw instructions` | one sentence each | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** the guessed flag costs one call, not two; a digit root that was read silently is now named.
- **Negative:** one more table to keep true; a script that parses stderr line by line sees one more line on an all-digit short root.
- **Neutral:** stdout, exit codes and receipts are unchanged; MCP is untouched (`mrw mcp` has no plan flag, and its root is given once at launch).

## Out of Scope

- A hint for `-f` and `-p` on `write` (permanent: boundary: each reads as another tool's flag — `--force`, `patch -p` — so the plan sentence would be a guess)
- `-C` misuse after the subcommand on a command other than `read` (permanent: boundary: no other command has `-C`, and its unknown-flag error already names the help)
- Hints for `check`, `iter`, `seen` and `stats` (deferred: docs/adr/BACKLOG.md — no evidence yet)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the argv walk disagrees with the parser about where the subcommand starts | Low | a missed or spurious note | it uses ADR-069's `flagRole` and `flagKinds`, the classifier `refusePaddedFlagValues` already relies on; `TestAnAllDigitShortRootBeforeTheVerbIsNamed` drives every accepted spelling and the after-the-verb case through the built binary |
| urfave/cli rewords its unknown-flag message | Low | the write hint silently disappears | the lookup falls through to the old message; `TestAPlanFlagOnWriteIsAnsweredWithTheArgument` fails on the reword |

## Rollback

Revert T1: the message loses its second half and the note goes. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the deferred item above.
