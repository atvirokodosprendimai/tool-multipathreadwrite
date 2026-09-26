# ADR-078: An error reaches its reader in a form it can use

**Status:** Accepted
**Accepted:** 2026-09-26 by M — approved the plan to clear the open backlog (*"build a plan to address them at once, no dangling pieces, no dead code, no mockery, no features only in tests, all is wired, all is exercised"*), whose ADR-078 is this record
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-011, ADR-014, ADR-015, ADR-032, ADR-067, ADR-069, ADR-074, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/usage078_test.go`, `cmd/mrw/cmdquote_windows_test.go`, `internal/mcp/mcp.go`, `internal/mcp/tools.go`, `internal/mcp/protocol078_test.go`, `internal/read/read.go`, `internal/read/walk.go`, `internal/read/stop078_test.go`, `internal/plan/plan.go`, `internal/plan/parse078_test.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `cmd/mrw/usage078_test.go::TestAUsageErrorWritesNothingToStdout`
**Invalidates:** none
**Served-path change:** a usage error writes nothing to stdout and names the help to read, exit 2; `mrw mcp` refuses arguments; `-C` on a read with no `/pattern/` is refused, exit 2; a header that does not parse is one error, its body not reported as stray text, and a tab after `@@` is named; on Windows a padded-path refusal also names the cmd.exe form; over MCP an id that is neither a string nor an integer is refused `-32600`, an argument that is not valid UTF-8 is refused by name, an `exclude` glob that can never match is refused as the CLI refuses it, a named read of many specs stops once it has overflowed and says "more than N bytes", and a line that alone encodes past the ceiling is named with the ranges around it that still serve.

## Context

**What was observed** (v1.25.1 adversarial round, BACKLOG "Smaller, recorded as found", the review
of #232, and probes on v1.26.0, 2026-09-26). Each is an answer the reader cannot use:

- `mrw mcp --bogus-flag` printed its help on STDOUT — the protocol stream a host reads JSON-RPC
  from. urfave/cli prints help to `Root().Writer` on a usage error.
- Over MCP an id of `{}` or `1.5` was dispatched and echoed back; invalid UTF-8 in an argument was
  decoded to U+FFFD, so a spec sent as `\xff.go` reached the engine as a path nobody sent;
  `exclude: ["["]` was ignored while the CLI refuses `--exclude '['`.
- A named MCP read of 100,000 specs held the server past two minutes, reading every spec to the end
  to produce a refusal it knew after a few thousand.
- A line that alone encodes past the MCP ceiling was advised a range that held it; the next call
  said no narrower range could help, though one after the line did (the waiver on #232).
- An overflowing read of many specs whose last served header had no numbered line under it was told
  "One line of this file renders to more than the whole limit", a line that did not exist. Found by
  this record's own mutation run.
- `@@ a.go 3 replac` was followed by "text before the first @@ header" for each of its body lines; a
  tab after `@@` read as prose; `mrw read -C 1 a.go:2` ignored `-C` in silence.
- A padded-path refusal suggested `mrw read -- 'x '`, which cmd.exe passes with the quotes.

## Existing Primitives Audit

- `cli.Exit` and `exitUsage` are how every refusal reaches main, which prints to stderr.
- `pathExcluded` (ast-grep) and `walker.excluded` were the same matcher, written twice.
- `servedLineNumber` and `encodedTextLen` already read a served line's number and its wire size.
- `capped` already knows when the answer overflowed (`over`).

## Decision

1. Every command answers a flag the parser rejects with `OnUsageError`: the error and the help to
   read, returned to main for stderr; nothing on stdout. `mrw mcp` refuses arguments.
2. `-C` on a read with no single `/pattern/` range is refused — it widens a pattern match and
   nothing else — as `--exclude` without `--grep` is (ADR-007).
3. A header that does not parse marks the lines under it as its body until the next good header; a
   top-level line beginning `@@` and a tab names the tab.
4. On Windows a padded-path refusal adds the cmd.exe form (double quotes) after the POSIX one.
5. Over MCP: `validRequestID` (a string, or an integer literal); `nonUTF8Arg` names the first
   argument that is not valid UTF-8, as a tool error; `read.CheckExclude` refuses a malformed glob or
   one starting with `/` on both surfaces, and `walker.excluded` calls `pathExcluded`.
6. `read.Options.Stop` is asked before each spec; a named MCP read of several specs stops once it has
   overflowed, and the refusal says "more than N bytes".
7. `tooLongLine` finds the served line that alone encodes past the ceiling; both refusals name it,
   the CLI read for it, and the ranges before and after it that still serve.
8. `overflowMessage` says one line of a file cannot fit only when that file's header opened within
   the first quarter of the sample, so the unterminated line holds most of it; otherwise the refusal
   advises narrower ranges or fewer files.

## Alternatives Considered

- **`HideHelp` on `mcp`.** Rejected: it makes `mrw mcp --help` an unknown flag, and a root-flag typo
  still printed the root help to stdout.
- **Refuse a decoded U+FFFD.** Rejected: a real filename may contain one; the raw bytes are the
  evidence.
- **Apply `-C` to line ranges.** Rejected: it would change what a range serves and what the ledger
  records for every caller who passes `-C` with mixed specs.

## Component / Boundary Impact

`cmd/mrw`, `internal/mcp`, `internal/read` (`Options.Stop`, `CheckExclude`), `internal/plan` (parse
messages), `internal/guide`. The engine packages `apply`, `check`, `state`, `lines`, `iter`, `seen`,
`subproc` and `rooted` are unchanged.

## Wiring & Contract Changes

New exit-2 refusals (usage errors now without the help dump, `mcp` arguments, `-C` without a
pattern); new MCP refusals (`-32600` for an id, tool errors for UTF-8 and `exclude`). Contract
§155–§159.

## Inter-task Contracts

T2 produces `read.CheckExclude`, which T1's CLI path consumes. T3 produces `read.Options.Stop`.

## Implementation

See `tasks/`.

## Consequences

- A host config that passes stray arguments to `mrw mcp` stops working, and says why.
- `mrw read -C N` with no pattern, which did nothing, now exits 2.
- A usage error no longer prints the whole help; it names `--help`.

## Out of Scope

- A `\\udc80` escape in MCP arguments (permanent: fact: it is valid JSON text that decodes to U+FFFD, indistinguishable from a real U+FFFD in a name)
- A `@@` followed by a tab inside an uncounted body (permanent: fact: a body line is content, and a counted body is the documented escape)
- An attached `-C/path` rewritten by MSYS (permanent: fact: MSYS rewrites argv before mrw starts; AGENTS.md names both switches that stop it)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A script parses the help text a usage error used to print | Low | Low | `--help` prints it; the error names `--help` |
| The early stop changes a multi-spec refusal's byte estimate | Certain | Low | it says "more than N", which is true |

## Rollback

Revert the three tasks. No state format changes.

## Follow-ups

- [ ] Release with ADR-076, ADR-077, ADR-079 and ADR-080 as v1.27.0.
