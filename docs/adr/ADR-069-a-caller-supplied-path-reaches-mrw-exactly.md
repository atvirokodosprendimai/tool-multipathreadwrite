# ADR-069: A caller-supplied path reaches mrw exactly as typed, or is refused

**Status:** Accepted
**Accepted:** 2026-09-25 by M — *"Accepted"*
**Date:** 2026-09-25
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-051, ADR-057, ADR-068, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `internal/iter/iter.go`, `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/ingest/applypatch.go`, `internal/ingest/searchreplace.go`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/paddedarg_test.go::TestAPaddedPositionalWithoutDashDashIsRefused`
**Invalidates:** none — checked
**Served-path change:** A path with a trailing space now reaches mrw as typed on every surface that takes one: `--files-from` lines, the working set (`mrw iter`), a rename destination, and the paths in `--format=apply_patch` and `--format=search_replace`. Today each of these silently retargets `x ` to `x`. A padded positional given WITHOUT `--` (`mrw read 'x '`) is now refused, exit 2, and the refusal names `--`. Today it silently reads `x`. `mrw read -- 'x '` is unchanged.

## Context

**What was observed.** ADR-068 (v1.24.1) stopped the read ledger turning `x ` into `x`. The Codex
review of #214 and a read-only survey on 2026-09-25 found the same trim at every other place a
caller hands mrw a path. None of these sites licenses a file that was not read: ADR-068's ledger
still refuses the write. But each one acts on a file the caller did not name, and says so only in
a header or receipt that names `x`.

**The class this record governs:** a path the caller supplied, trimmed by mrw or by its argument
parser before use. Enumerated 2026-09-25 with `git grep -n TrimSpace -- '*.go' ':!*_test.go'`
plus urfave/cli v3.11.0's `command_parse.go`:

| Site | What is trimmed | Grammar needs it? |
|---|---|---|
| urfave/cli `command_parse.go:81`, `:117` | every positional before `--` (read `:621`, write `:921`, iter `:1281`, check `:1378`) | no |
| `cmd/mrw/main.go:1706` | each `--files-from` line | only the blank/`#` test |
| `internal/iter/iter.go:60`, `:109`, `:125` | working-set lines; `iter add` / `rm` arguments | only the blank/`#` test |
| `internal/apply/apply.go:387`, `:453`; `pathop.go:76` | a rename's one-line destination | no |
| `internal/ingest/applypatch.go:76`, `:91`, `:103`, `:110`, `:187` | the path after `*** Delete/Move to/Update/Add File:` | ONE separator space |
| `internal/ingest/searchreplace.go:40`, `:83` | the path after `<<<<<<< SEARCH`; the filename line | ONE separator space; the blank test |

Not members (not paths): `applypatch.go:363-365` (`trimStar`), `plan.go:257`, `:270`; emptiness
guards `pathop.go:48`, `plan.go:749`, `applypatch.go:184` stay.

**Measured 2026-09-25** in a scratch module against urfave/cli v3.11.0: for `read 'x '`,
`cmd.Args()` is `["x"]` and `cmd.Root().Args()` is `["read", "x "]`. A padded flag value
(`--grep ' y'`) is not trimmed. `--` passes everything after it untouched.

**Decided with M, 2026-09-25:** a padded positional without `--` is refused, not recovered.
Recovering it means re-implementing urfave's flag arity and short-flag grouping to line tokens up.

## Existing Primitives Audit

- **urfave/cli's `--`**: already delivers an exact positional. The refusal points at it and adds
  nothing new.
- **`cmd.Root().Args()`**: the root hands its raw tail to the subcommand. This is the one place
  the untrimmed token survives, in tests (`rootCommand().Run`) and in the binary alike.
- **The plan header's quoting** (`plan.go` `splitHeader`, `quotePlanPath` in `applypatch.go`)
  already keeps a spaced path exactly. ADR-051's compilers emit it, so once the ingest side stops
  trimming, the compiled plan carries the exact name with no new syntax.
- **`exitUsage`** (`main.go:62`) with `cli.Exit` is the existing usage-refusal idiom.

## Decision

1. **The CLI refuses a positional its parser trimmed.** One helper, `refusePaddedArgs(cmd)`, runs
   first in the `read`, `write`, `iter` and `check` Actions. It walks `cmd.Root().Args()` after the
   subcommand name, up to the first `--`. A token that differs from its own trim, whose trim is one
   of `cmd.Args()`, is refused with exit 2:
   `mrw: 'x ' has edge whitespace the argument parser strips; put -- before the path: mrw read -- 'x '`.
   `mrw iter note` is exempt: its argument is free text, not a path, and Load trims a note anyway
   (`iter.go:66`).
2. **The line formats test for blank and `#` with a trimmed copy, and keep the line.** This covers
   `--files-from` and the working set (Load, Add, Remove).
3. **A rename destination is its body line as written.** The plan scanner has already removed the
   line's terminator.
4. **The foreign formats strip exactly one separator space** after their marker (`TrimPrefix`),
   and a search/replace filename line is kept as written. Whether a path is PRESENT is still decided
   on a trimmed copy, so a marker followed only by whitespace means "no path", as today (a
   `<<<<<<< SEARCH` line with trailing blanks falls back to the previous filename).
5. **Amended 2026-09-25 after the Codex review of v1.25.0 (T5).** The first helper stopped at
   every `--`, including one consumed as a flag's value, so `read --grep -- 'x '` still reached `x`;
   and urfave trims an ATTACHED value with its token, so `--files-from='list '` opened `list` and
   `--root='dir '` named `dir`. Now a flag's own value is skipped (the parser keeps a separate value
   as given), so only a `--` in argument position ends the guard; an attached value that ends in
   whitespace is refused, exit 2, naming the separate spelling; `main` checks the whole argv first,
   because a root flag never reaches a subcommand's tail. The iter refusal keeps its verb
   (`mrw iter add -- 'x '`). Skipping flag values also retires the false refusal this record listed
   under Risks: a padded flag value beside an equal positional (`--exclude ' x' x`) is accepted.
6. **Amended 2026-09-25 after the Codex review of PR #222 (T6).** Three gaps in item 5. The guard
   looked a flag name up as typed, so a padded boolean name (`'--no-numbers '`, which the parser
   trims to the flag) read as value-taking and the padded path after it was skipped: `read
   '--no-numbers ' 'x '` served `x`. The whole-argv check read every `-` token as a flag, so a
   separate value that looks like one (`--files-from '--list= '`) was refused though the parser
   keeps it, and it stopped at any `--`, including one a root flag consumed, so
   `--root -- --root='dir '` reached `dir`. And `padAttached` checked for space and tab while the
   parser's trim is `strings.TrimSpace`, so `--files-from=$'list\n'` opened `list`. Now a flag name
   is classified trimmed, the whole-argv check reads argv as the parser does (root flags and their
   values, the subcommand, its flags and their values; only a bare `--` ends it), and any trailing
   whitespace on an attached value is refused. Item 5's invariant holds again: a separate value is
   never refused.
7. **Amended 2026-09-25 after the second Codex review of PR #222 (T7).** Two gaps in item 6. A
   parent's persistent flag is accepted by a subcommand that has no flag of the same name
   (`command_parse.go:43-57`): `write` and `iter` take `--root` after the verb, `read`, whose `-C`
   is context, does not. Neither guard read the inherited flag, so `write --root -- --root='dir '`
   ended the walk at the `--` the root flag consumed and reached `dir`, and `iter --root ' x' add x`
   was falsely refused. And a single dash before a non-letter is where the parser stops and keeps
   every remaining token as given (`command_parse.go:134-138`), so a file named ` -1= ` was served
   by v1.25.0 and refused by item 6's walker as an attached value. Now both guards read the flags
   the parser accepts for the command, ancestors' persistent ones included, and stop where the
   parser stops.
8. **Amended 2026-09-25 after the third Codex review of PR #222 (T8).** Two gaps in item 7. A `--`
   before the subcommand ends the ROOT's options only: the parser still dispatches the subcommand,
   which parses its own flags (`command_run.go:282-315`). The whole-argv guard ended its walk there
   and the iter guard exempted a note entirely, so `-- iter note --root='dir ' x` and
   `-- stats --root='dir '` reached `dir`. And a lone `-` ends the parse: the parser keeps it as a
   positional and drops every token after it (`command_parse.go:123-125`), so the guard, reading on,
   refused `write - '--format=plan '`, which v1.25.0 accepted. Now the whole-argv walk continues
   below the subcommand after a root `--`, the note exemption covers the note's words and not the
   flags beside them, and both guards stop at a lone `-`.

**What would make this decision fail:** a caller who relies on the trim, e.g. a generated
`--files-from` list with trailing spaces after every path. That caller now gets a read of a
missing file, `x ` UNREADABLE, exit 1. That is loud, and it names the path mrw looked for.

## Alternatives Considered

- **Recover the raw positional** by aligning `cmd.Root().Args()` with `cmd.Args()`. Rejected with
  M: it needs urfave's flag arity (`-C 2`, `--grep x`) and short-flag grouping, and a misalignment
  would silently pick the wrong token, which is the defect this record removes.
- **`SkipFlagParsing` on every subcommand and a hand parser.** Rejected: it rewrites every
  subcommand's flags to fix one edge case.
- **Leave the trims and document `--`.** Rejected: `--files-from`, the working set, rename and the
  foreign formats have no `--` to reach for.

## Component / Boundary Impact

`cmd/mrw`, `internal/iter`, `internal/apply` (rename destination only) and `internal/ingest`.
`internal/apply` and `internal/iter` are engine packages, and this record owns those lines.
Byte-identical: `internal/read`, `internal/plan`, `internal/seen`, `internal/check`,
`internal/state`, `internal/lines`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| CLI positionals | a padded positional without `--` is exit 2, naming `--` | T1 | every CLI caller |
| `--files-from`, `mrw iter` | a line keeps its trailing space | T2 | CLI callers |
| plan `rename` | the destination keeps its trailing space | T3 | CLI + MCP `mrw_write` |
| `--format=apply_patch`, `search_replace` | a path keeps its trailing space | T4 | CLI + MCP `mrw_write` |
| contract §128–§131 | one section per task | T1–T4 | CI, `adr-verify` |

## Inter-task Contracts

None — the four tasks touch disjoint sites and can land in any order; they are listed in the order
a caller meets them.

## Implementation

See `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/README.md`.

## Consequences

- **Positive:** mrw acts on the file the caller named, or refuses. It never quietly acts on its
  trimmed sibling.
- **Negative:** a list or plan that carried stray trailing spaces after ordinary names now fails
  to find those files. The failure names the padded path.
- **Neutral:** no exit code changes meaning; no format change; `--` is unchanged.

## Out of Scope

- A path whose first non-blank character is `#`, or that is only whitespace, in `--files-from` or the working set; a LEADING space is kept, like a trailing one (permanent: boundary: one spec per line with no quoting, and the documented skip rule reads such a line as a comment or blank)
- A path ending in `\r` in any line format (permanent: boundary: the scanners strip a `\r` before `\n` before any of this code runs; only a quoted plan header carries one, ADR-068)
- The MCP surface, whose specs and plans are JSON strings and are not trimmed (permanent: boundary: nothing to fix there; `mrw_read` of `x ` is served since #214)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A padded flag VALUE equal to a positional's trim (`--exclude ' x' x`) | — | — | retired by T5: flag values are skipped, not compared |
| A caller's generated list carries trailing spaces | Low | Med | the read reports the padded path UNREADABLE, exit 1 |
| Windows cannot hold `x` and `x ` apart | High on Windows | Low | the space fixtures skip there, as ADR-068's do; the Go logic is platform-free |

## Rollback

Revert the helper, `padAttached`, `refusePaddedFlagValues`, the four trims, the tests and §128–§131 and §134. Nothing persistent moves: a
working-set file written with a trailing space is read back trimmed by the old binary.

## Follow-ups

- [x] AGENTS.md §1: a path with edge whitespace goes after `--` (landed with T1); `am_update_skill("mrw")` at the release.
- [ ] Release with ADR-070 as v1.25.0.
