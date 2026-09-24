# ADR-063: `mrw instructions` teaches the read side

**Status:** Accepted
**Accepted:** 2026-09-24 by M — *"approve"*, after *"since this search capabilities seems not advertised properly"*
**Date:** 2026-09-23
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-017, ADR-026, ADR-036, ADR-037, ADR-058, ADR-062, docs/adr/BACKLOG.md
**Governs:** `internal/guide/guide.go`, `cmd/mrw/main.go`, `.claude/skills/mrw/SKILL.md`, `scripts/contract.sh`
**Enforced-by:** `internal/guide/guide_test.go::TestCLITeachesTheReadSide`
**Served-path change:** `mrw instructions` gains a read section naming the address forms a read and a plan take and every `read` flag; `mrw --help` says mrw finds as well as reads and writes, and describes `read` as serving patterns and `--grep` hits, not only line ranges; `mrw read --help` shows `--ast-grep PATTERN` instead of `--ast-grep ast-grep`. The MCP handshake does not change.

## Context

**What was observed.** 2026-09-23, M asked whether mrw can really read many files, by line and by symbol. A live run showed that it can — `path:/^func Apply/,+3` resolved to `internal/apply/apply.go:283-286`, `--grep 'func Shared'` walked `internal/` and served `internal/guide/guide.go:11-13` — and then asked why none of it is in the instructions a model gets. It is not:

| Surface | Address forms | `--grep` / `--ast-grep` / `--exclude` / `--files-from` |
|---|---|---|
| `mrw instructions` (`guide.CLI()`) | no — "regex address" appears only inside the MSYS trap | no |
| `mrw --help` | NAME line is "read and write many file ranges in one call"; `read` is "print line ranges from one or more files" | no |
| `.claude/skills/mrw/SKILL.md` description | "reads many file ranges" | no |
| MCP handshake (`internal/mcp/instructions.go`) | yes | yes |
| `mrw read --help`, AGENTS.md "Using mrw" | yes | yes |

ADR-062 made `guide.CLI()` "the effective-use document" for a caller with only the binary, and its first sentence tells that caller to plan "one read of every site". It then teaches every write op and none of the read addressing that makes one read of every site possible. A PATH caller learns the finding half only if it thinks to run `mrw read --help`, and nothing it is shown points there.

**The class this record governs.** Every agent-facing summary of mrw that a caller reads BEFORE `mrw read --help`. Enumerated 2026-09-23 with

```
git grep -nE 'Usage: +"|func CLI\(\)|^description:' -- 'cmd/mrw/main.go' 'internal/guide/*.go' '.claude/skills/mrw/SKILL.md'
```

That lists `guide.CLI()`, the skill `description:`, and every command and flag `Usage` in `cmd/mrw/main.go`. Five members describe reading and undersell it: `guide.CLI()`, the root command's `Usage` (the NAME line of `mrw --help`), the `read` command's `Usage`, the `--ast-grep` flag's `Usage` (whose first backticked word is `ast-grep`, so urfave/cli prints `--ast-grep ast-grep` as its placeholder), and the skill `description:`. The other `Usage` strings are sibling commands and flags that do not summarise reading. Left out, because they already teach the read side (checked with `git grep -lE 'grep|files-from'` on the same day): `internal/mcp/instructions.go`, `internal/mcp/mcp.go`, `AGENTS.md`, `README.md`, and `mrw read --help`'s Description and remaining flag usages, which are the source this record copies from.

## Existing Primitives Audit

- **`guide.CLI()`.** Reused. The read section goes after the write cookbook ADR-062 added; Shared() is untouched and stays five sentences (§75, §85).
- **`readCmd()` in `cmd/mrw/main.go`.** Reused as the authority: its `Flags` are what the new cmd test iterates, so a flag added or renamed there without the instructions following turns the test red. The other direction — the instructions still naming a flag `read` no longer has — is contract §115's, which checks every taught `--flag` against `read --help`. The instructions teach what the binary accepts, not a list kept beside it.
- **`instructionsText()` / `maxInstructionsChars`.** Unchanged. The handshake already teaches the read forms, at 3,890 of 4,096 bytes (measured 2026-09-23 through `mrw mcp` `initialize` on v1.22.1; the 4,095 in BACKLOG.md is dated 2026-09-08 and stale). Nothing here moves it.
- **`TestInstructionsCommandPrintsCLI`.** Reused: it already proves `mrw instructions` prints `guide.CLI()` byte for byte, so the new text needs no new wiring.

## Decision

**`guide.CLI()` teaches the read side, after the write cookbook: the address forms a read takes, the subset a plan takes, what finds files you cannot name, and where a write reads an address differently.** The text, as it will be printed:

```
Read every site in one call: mrw read a.go:40-60 'b.go:/func Start/,+12' c.go:$
A spec is a bare path (the whole file) or PATH:RANGE[,RANGE...]. A RANGE is N, N-M, N- (to the end), -M (from the start), A,+N (A plus the N lines after it), $ (the last line), /regexp/ (every matching line; -C N, or --context N, adds lines either side) or /from/,/to/ (to the first match of to at or after from). Quote a spec that contains a space.
A write plan takes N, N-M, N-, $, A,+N, /regexp/ and /from/,/to/, but not -M or a comma list; its start pattern must match exactly once, and it refuses a relative end past the last line where a read clamps.
To find files you cannot name: --grep PATTERN walks the paths given, or the root when none are, and serves each match as /regexp/ would. --exclude GLOB drops files the walk finds, by root-relative path or basename, and repeats. --grep does not read .gitignore; a .git directory the walk meets is skipped, but one you name is walked.
--ast-grep PATTERN is structural search run by the ast-grep binary, which must be on PATH: a missing one exits 2 naming it, and one that hangs is killed at 2 s.
--files-from FILE takes one spec per line, - for stdin: rg -l X . | sed 's|$|:/X/|' | mrw read --files-from - (name rg's path: with none, rg reads a piped stdin and waits)
--stat prints only length, size and sha; --max-lines N caps each spec; --no-numbers drops the numbers a plan addresses by. A read exits 1 when a range cannot be served (a pattern with no match, a start past the end, lines --max-lines withheld) and still prints the rest; an end past the last line is clamped, not an error.
```

**Every flag `mrw read` defines is named in `guide.CLI()` as `--<name>`.** Enforced from the binary's own flag list, so the instructions cannot fall behind the command.

**The help summaries name the read side.** The root `Usage` becomes `find, read and write many file ranges in one call`. `read`'s `Usage` becomes `print line ranges, pattern matches or --grep hits from one or more files`. The `--ast-grep` flag's `Usage` puts `` `PATTERN` `` first, so its placeholder is `PATTERN`.

**The repo skill's `description:` names the address forms and `--grep`** in one clause, staying under the 1,024-byte description cap (measured 722 bytes as parsed YAML on 2026-09-23).

**The MCP handshake does not change.** Examples and flags stay off `initialize`, as ADR-062 decided.

## Alternatives Considered

- **Point at `mrw read --help` from `mrw instructions` instead of teaching.** Rejected: one more call before the first useful read, and a pointer is exactly the sentence a skimming model drops. The whole read section is seven lines.
- **Generate the read section from `readCmd()` flag usages.** Rejected: flag `Usage` strings are written for a human scanning `--help`, not for a caller planning one read; the test that iterates `readCmd().Flags` gives the same drift protection without making prose a build product (ADR-045's reasoning).
- **Put the read section on the MCP handshake too.** Rejected: already there in its own words, and the handshake is paid by every session; ADR-062 keeps examples and the cookbook off it (ADR-037's bound).
- **Change only `guide.CLI()`.** Rejected: `mrw --help` is the first thing a caller with the binary runs, and "line ranges" is precisely the misreading M had.

## Component / Boundary Impact

`internal/guide` owns the text. `cmd/mrw` changes three `Usage` strings and gains a test reading its own `rootCommand()` and `readCmd()`. The skill file is a doc. Engine packages (`internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`) and `internal/mcp` stay byte-identical.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `guide.CLI()` | read section after the write cookbook | T1 | `mrw instructions` stdout |
| root, `read` and `--ast-grep` `Usage` | name finding; placeholder is `PATTERN` | T1 | `mrw --help`, `mrw read --help` |
| `.claude/skills/mrw/SKILL.md` `description:` | names address forms and `--grep` | T1 | Claude Code skill listing |
| contract §115 | next free after §114 | T1 | `adr-verify`, CI |

Exit codes unchanged. `mrw instructions` with an extra argument stays exit 2 (§75).

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| read section on CLI(); every read flag named | T1 | — | No — additive teaching text |

## Implementation

See `docs/adr/ADR-063-instructions-teach-the-read-side/tasks/README.md`.

## Consequences

- **Positive:** a PATH caller that reads `mrw instructions` can do the one read of every site it is told to plan, including sites it cannot name. `mrw --help` no longer undersells `read`.
- **Negative:** `mrw instructions` grows by seven lines, paid once by a caller that asks for it.
- **Neutral:** handshake bytes unchanged. Shared() unchanged. AGENTS.md and README.md change only their `rg -l` example, which now names `.` (the chaos finding under Risks).

## Out of Scope

- The MCP handshake and tool descriptions (permanent: boundary: already teach the read side; 4096 is ADR-037's bound)
- Raising `maxInstructionsChars` (permanent: boundary: the handshake is paid once per session)
- Generating CLI() from flag usages (permanent: boundary: ADR-045's reasoning — prose is authored)
- Engine packages `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` (permanent: boundary: teaching, not serving)
- Installing `ast-grep` on any machine (permanent: boundary: ADR-058 shells out to what is present)
- `@N` working-set pointers in the read section (permanent: boundary: they belong to `mrw iter`, which teaches the working set)
- Updating the centralised `mrw` skill description in the palace (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A read flag is added later and the instructions fall behind | Med | Med | cmd test iterates `readCmd().Flags` |
| The taught examples are wrong | Low | Med | §115 runs the quoted example spec and the `--files-from -` form against `$MRW`, and drives the two write refusals the text teaches |
| Someone copies the section onto the handshake | Low | High | §85 already bounds 4096; Out of Scope names it |
| The taught `rg -l X \| sed \| mrw read --files-from -` pipeline hangs: with no path, rg searches a piped stdin (found 2026-09-24 by the chaos pass, stdin an open pipe, exit 124 at a 5 s timeout; with `.` it served three files) | Med | Med | the example names `.` and says why; `TestCLITeachesTheReadSide` pins `rg -l X . \|`; the AGENTS.md and README.md copies of the pipeline get the same `.` |

## Rollback

Revert the read section in `guide.CLI()`, the three `Usage` strings, the skill description clause, the three tests and §115. Nothing persistent moves.

## Follow-ups

- [ ] Centralised `mrw` skill description (agentsmemory) — receipt in BACKLOG; not this binary
