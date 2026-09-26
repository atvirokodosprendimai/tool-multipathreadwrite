# ADR-078: An error reaches its reader in a form it can use

**Status:** Accepted
**Accepted:** 2026-09-26 by M — approved the plan to clear the open backlog (*"build a plan to address them at once, no dangling pieces, no dead code, no mockery, no features only in tests, all is wired, all is exercised"*), whose ADR-078 is this record
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-011, ADR-014, ADR-015, ADR-032, ADR-067, ADR-069, ADR-074, docs/adr/BACKLOG.md
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/usage078_test.go`, `cmd/mrw/cmdquote_windows_test.go`, `internal/mcp/mcp.go`, `internal/mcp/tools.go`, `internal/mcp/protocol078_test.go`, `internal/mcp/tools_test.go`, `internal/read/read.go`, `internal/read/walk.go`, `internal/read/stop078_test.go`, `internal/plan/plan.go`, `internal/plan/parse078_test.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `cmd/mrw/usage078_test.go::TestAUsageErrorWritesNothingToStdout`
**Invalidates:** none
**Served-path change:** a usage error writes nothing to stdout and names the help to read, exit 2; `mrw mcp` refuses arguments; `-C` on a read with no `/pattern/` is refused, exit 2, and by name with `--ast-grep`; a header that does not parse is one error, its body not reported as stray text, and a tab after `@@` is named, under a broken header too; a padded-path refusal quotes a name's own apostrophe, and on Windows also names the cmd.exe form; over MCP an id that is neither a string nor a number with an integer value is refused `-32600`, an argument that is not valid UTF-8 is refused by name, an `exclude` glob that is malformed or spelled so it can never match (rooted, `./`, a trailing `/`) is refused on both surfaces, a named read of many specs stops once it has overflowed and says "more than N bytes", a line past the ceiling is named with the ranges around it, and a file that got only the room earlier files left is named and sent to be read alone.

## Context

**What was observed** (v1.25.1 adversarial round, BACKLOG "Smaller, recorded as found", the review
of #232, and probes on v1.26.0, 2026-09-26). Each is an answer the reader cannot use:

- `mrw mcp --bogus-flag` printed its help on STDOUT — the protocol stream a host reads JSON-RPC
  from. urfave/cli prints help to `Root().Writer` on a usage error.
- Over MCP an id of `{}` or `1.5` was dispatched and echoed back; invalid UTF-8 in an argument was
  decoded to U+FFFD, so a spec sent as `\xff.go` reached the engine as a path nobody sent;
  `exclude: ["["]` was ignored while the CLI refuses `--exclude '['`.
- A named MCP read of 100,000 specs held the server past 120 s in that round, reading every spec to
  the end to produce a refusal it knew after a few thousand (3.9 s on the reviewer's machine for
  #239: the work is per spec, so it is the machine that varies).
- A line that alone encodes past the MCP ceiling was advised a range that held it; the next call
  said no narrower range could help, though one after the line did (the waiver on #232).
- An overflowing read of many specs whose last served header had no numbered line under it was told
  "One line of this file renders to more than the whole limit", a line that did not exist. Found by
  this record's own mutation run.
- The reviews of #239 (Codex and in-process) found the fix incomplete: a plain line past the ceiling
  still took "no narrower range"; "still serve here" was said of a prefix range that was then
  refused; a file after another was still told its line could not fit; an id spelled `1.0` or `1e3`
  was refused; an argument named `""` passed the UTF-8 check; `-C` with `--ast-grep` was told to name
  a pattern; a noted working set printed its note before the `-C` refusal; a tab header under a
  broken one was swallowed; `vendor/` and `./vendor` matched nothing in silence; and the cmd.exe fix
  rewrote a name's own apostrophe.
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
   nothing else — as `--exclude` without `--grep` is (ADR-007); with `--ast-grep` it is refused by
   that name, before the finder runs. A noted working set's note is printed only past the refusals.
3. A header that does not parse marks the lines under it as its body until the next good header; a
   top-level line beginning `@@` and a tab names the tab, under a broken header too.
4. A padded-path refusal quotes the name for a POSIX shell (`posixQuote`), and on Windows adds the
   cmd.exe form (double quotes) after it, built from the name, not from the POSIX form.
5. Over MCP: `validRequestID` (a string, or a number whose value is an integer, judged from its
   digits and echoed as sent); `nonUTF8Arg` names the first argument that is not valid UTF-8, as a
   tool error, and an empty name as "the argument with an empty name"; `read.CheckExclude` refuses a
   malformed glob or one no root-relative path can match — rooted, `./`-prefixed, or ending in `/` —
   on both surfaces, and `walker.excluded` calls `pathExcluded`.
6. `read.Options.Stop` is asked before each spec; a named MCP read of several specs stops once it has
   overflowed, and the refusal says "more than N bytes".
7. `tooLongLine` finds the served line that alone encodes past the ceiling, and `overflowMessage`
   takes a plain one from the cut-off tail's gutter; both refusals name it, the CLI read for it, and
   the ranges around it — the open one after it "reads on", the one before it "holds the lines
   before it", neither promised.
8. `overflowMessage` says a line of a file cannot fit only when that file is the only one in the
   sample; a file after others got only the room they left, so it is named and sent to be read
   alone, which settles it.

## Alternatives Considered

- **`HideHelp` on `mcp`.** Rejected: it makes `mrw mcp --help` an unknown flag, and a root-flag typo
  still printed the root help to stdout.
- **Refuse a decoded U+FFFD.** Rejected: a real filename may contain one; the raw bytes are the
  evidence.
- **Apply `-C` to line ranges.** Rejected: it would change what a range serves and what the ledger
  records for every caller who passes `-C` with mixed specs.
- **An integer literal only, for an id.** Shipped in the first cut and rejected in the review of
  #239: MCP's RequestId is a JSON Schema integer, a value, so `1.0` and `1e3` are integers.
- **Measure the cut-off line from disk before diagnosing it.** Rejected: the refusal would read a
  file the caller did not get, and "read it alone" reaches a certain answer in one more call.

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

- [x] Release with ADR-076, ADR-077, ADR-079 and ADR-080 as v1.27.0 — tagged at `7058767` (#241), Status #242.
