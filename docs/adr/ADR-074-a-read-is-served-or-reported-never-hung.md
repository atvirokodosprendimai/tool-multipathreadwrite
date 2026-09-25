# ADR-074: A read is served or reported, never hung

**Status:** Accepted
**Accepted:** 2026-09-25 by M — *"plan to address these, properly, no looping on small details"*; the plan that groups the v1.25.1 adversarial round into five records was approved the same day
**Date:** 2026-09-25
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-014, ADR-023, ADR-024, ADR-031, ADR-032, ADR-058, ADR-065, ADR-072, ADR-073, docs/adr/BACKLOG.md
**Governs:** `internal/read/read.go`, `internal/read/walk.go`, `internal/read/astgrep.go`, `internal/read/fifo_unix_test.go`, `internal/read/msys_hint_test.go`, `internal/subproc/subproc.go`, `internal/subproc/interrupt_unix_test.go`, `internal/check/check.go`, `internal/mcp/tools.go`, `internal/mcp/escaped_page_test.go`, `cmd/mrw/main.go`, `cmd/mrw/astgrep_grandchild_unix_test.go`, `cmd/mrw/filesfrom_long_test.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `internal/guide/guide.go`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/mcp/escaped_page_test.go::TestAMarkupFilePagesByItsEncodedSize`
**Invalidates:** none — checked: ADR-058's 2 s bound stands (this makes it hold); ADR-014's first page and ADR-032's encoded ceiling stand (this sizes the page by the measure the ceiling uses); ADR-072's interrupt handling stands and moves into `internal/subproc` unchanged
**Served-path change:** `mrw read` and `mrw_read` report a FIFO, socket or device spec by name instead of blocking on it. A hanging `--ast-grep` returns at its 2 s bound even when a grandchild holds its stdout, on unix the grandchild is killed, and an interrupt sent to mrw stops ast-grep. Over MCP a file whose JSON-escaped text overflows the ceiling (markup: `<`, `>`, `&` cost six bytes each) is served as a first page with `next_read` instead of refused whole, and a closed range that overflows is refused with the encoding named. `--files-from` names the line when one exceeds its 8 MiB limit, and the MSYS hint names both variables that stop the rewriting.

## Context

**What was observed.** The v1.25.1 adversarial round (2026-09-25; `docs/adr/BACKLOG.md`, "From the
v1.25.1 adversarial round", and the read-side addendum in team memory) found four ways a read hangs
or refuses what it could serve.

1. **A FIFO hangs `read`**, and `--stat`, and a symlink to one: `read.Run` calls `os.ReadFile` on
   the resolved spec (`internal/read/read.go:416`) before looking at what it is. The walk has
   refused a non-regular candidate since ADR-007 (`internal/read/walk.go:121-128`); a named spec
   never did.
2. **The 2 s `--ast-grep` kill fails when a grandchild holds stdout.** `internal/read/astgrep.go:76-80`
   runs `exec.CommandContext` and `Output()`, which waits for every pipe to close: a wrapper script
   running `sleep 30` made the read take 30 s, and the sleeper outlived mrw.
3. **The MCP page budget ignores JSON escaping.** `firstPage` sizes a page from the raw bytes
   (`internal/mcp/tools.go:998`) while the ceiling is checked on the encoded result (`:458`,
   `:1080`). A 153,600-byte TSX file was refused whole with no `next_read`, and the refusal blamed
   the per-file receipt, while a 275,200-byte plain file paged.
4. **Smaller:** `--files-from` died on a line over 8 MiB with "token too long" and no line number
   (`cmd/mrw/main.go:2064-2077`); the MSYS hint names `MSYS2_ARG_CONV_EXCL` and not
   `MSYS_NO_PATHCONV`, which works too, and the docs say the rewrite fires on an example the round
   measured it does not fire on.

**A consequence found while designing item 2.** `internal/subproc` puts a child in a process group
of its own, so the terminal's ^C no longer reaches it. ADR-072 handled that for the check alone, in
`internal/check/check.go:200-212` and `:621-636`. Run through `subproc` without the same handling,
a ^C during a hung ast-grep would end mrw and orphan ast-grep, where today the terminal's group
signal takes both.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `lines.NotRegular` | `internal/lines/lines.go` (ADR-073) | The sentence, already shared by apply and the `--format` compilers; the walk and a named spec use it too. |
| `subproc.Command` | `internal/subproc/subproc.go:27` (ADR-072) | Process group, group kill on cancel, bounded wait for held pipes. |
| the check's signal handling | `internal/check/check.go:200-212`, `:621-636` (ADR-072) | Moves into `subproc` so every child mrw starts has it. |
| `encodedSize` | `internal/mcp/tools.go:1474` | The ceiling's own measure: `json.Marshal`. |
| `suggestLines` | `internal/mcp/tools.go:923` | Keeps its arithmetic; it is fed the encoded length. |
| `msysHint` | `internal/read/read.go:110-117` | The hint to extend. |

## Decision

1. **A named spec that is not a regular file is reported, not opened**: `==> <spec>  UNREADABLE
   not a regular file: mrw would block on a pipe or stream a device without end`, one problem,
   exit 1. A directory keeps its own "is a directory" line, and a stat that fails falls through to
   the error it always gave.
2. **`--ast-grep` runs through `internal/subproc`**: its group is killed at the 2 s bound and the
   wait for held pipes is bounded. The listening ADR-072 gave the check moves into `subproc` as
   `Interruptible`, and both children run under it: an interrupt, terminate or hangup sent to mrw
   cancels the child's context and kills its group, and a signal mrw was started with ignored
   stays ignored. On Windows only the wait bound applies.
3. **The MCP page is sized by its encoded length**: the per-line cost comes from the JSON-encoded
   text, the measure the ceiling checks. When a read's text renders inside the limit and its
   encoded answer does not, the refusal names the cause that fits the remedy: the encoding (ask for
   a narrower range) when the text alone encodes past the limit, the per-file receipt otherwise.
4. **The one-liners**: `--files-from` names the line that exceeds 8 MiB; the MSYS hint names both
   variables, and AGENTS.md, README and the served guide say when the rewrite fires.

## Alternatives Considered

- **Estimate the escape cost by counting `<>&`.** Rejected: the encoder's table (control bytes,
  U+2028, invalid UTF-8) is the truth, and a second table drifts from it.
- **Halve a page that still does not fit, up to three times.** Rejected for now: sizing by the
  encoded length fits the measured fixture on the first page, and a retry loop is code no fixture
  exercises. `firstPage` still declines a page that does not fit (ADR-031).
- **Stat every spec on MCP only.** Rejected: the CLI hung the same way.
- **Give ast-grep its own signal handling.** Rejected: that is ADR-072's list of signals a second
  time, and the two would drift. One function in `subproc` serves both children.

## Component / Boundary Impact

`internal/read` and `internal/mcp` are engine packages and this record owns the changes named above;
`internal/check/check.go` changes only where its signal handling moves into `internal/subproc`.
Byte-identical: `internal/apply`, `internal/plan`, `internal/seen`, `internal/state`,
`internal/lines`, `internal/iter`, `internal/rooted`, and the rest of `internal/read`,
`internal/check` and `internal/subproc`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --ast-grep` | the bound holds past a held pipe; an interrupt stops ast-grep | T1 | CLI, MCP `mrw_read` |
| `mrw write` check | listens through `subproc.Interruptible`, behaviour unchanged | T1 | CLI, MCP `mrw_write` |
| `mrw read` | a FIFO, socket or device spec is `UNREADABLE`, never opened | T2 | CLI, MCP `mrw_read` |
| `mrw_read` | a page sized by encoded length; a refusal naming the encoding | T3 | MCP |
| `mrw read --files-from` | the over-long line is named | T4 | CLI |
| contract §147–§149 | T1, T2, T3 | T1–T3 | CI, `adr-verify` |

## Inter-task Contracts

None: the four tasks touch disjoint code.

## Implementation

See `docs/adr/ADR-074-a-read-is-served-or-reported-never-hung/tasks/README.md`.

## Consequences

- **Positive:** no read blocks on a special file, a wrapper cannot defeat the ast-grep bound, a ^C
  stops ast-grep as it stops a check, and markup files page over MCP.
- **Negative:** a markup page carries fewer lines than a plain one, so reading such a file takes
  more calls.
- **Neutral:** no exit code changes meaning.

## Out of Scope

- A file literally named `c:1-2` (permanent: fact: everything after the last colon is the range and a path holding a colon is unsupported by design; citation: file `internal/read/read.go:119`)
- `a.go:$-1` exiting 1 where `a.go:5-3` exits 2 (permanent: fact: a plain reversed range is wrong for every file and refused at parse, one written with `$` is judged per file at resolve; citation: file `internal/read/read.go:327`)
- `--files-from` naming a FIFO (permanent: fact: a list may be a pipe, and stdin is one; citation: file `cmd/mrw/main.go:2053`)
- A `--grep` pattern MSYS rewrote into a false no-match getting a hint of its own (permanent: fact: the rewritten pattern is a valid regex and "no match" is a true answer to it; the docs name the trigger; citation: file `AGENTS.md:183`)
- A grandchild of ast-grep outliving mrw on Windows (deferred: docs/adr/BACKLOG.md "From the v1.25.1 adversarial round", the killed-check entry)
- The MCP items the round found (ids of any JSON type, U+FFFD for invalid UTF-8, usage on stdout, 100,000 specs, `exclude: ["["]`) (deferred: docs/adr/BACKLOG.md "From the v1.25.1 adversarial round")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A first page sized by the encoded average still does not fit, because a file's heavily escaped lines sit at its start | Low | Low | `firstPage` declines as before (ADR-031) and the refusal names the encoding and a narrower range |
| A signal arrives in the moment before `Interruptible` installs its handler | Low | Low | as ADR-072 records for the check: mrw dies by the default action, and the child has not started |
| Under `mrw mcp`, a terminate that arrives while a check or an ast-grep runs stops the child and leaves the server serving, where before it ended the server | Low | Low | a host stops a stdio server by closing its stdin first, and the server exits at the end of that call; a second signal meets the default action. ADR-072's check has the same property |

## Rollback

Revert the four tasks. Nothing persistent moves.

## Follow-ups

- [ ] Release with ADR-071, ADR-072, ADR-073 and ADR-075 as v1.26.0.
