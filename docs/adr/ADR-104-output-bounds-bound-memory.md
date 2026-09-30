# ADR-104: Output bounds bound memory

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — approved the plan "resolve the seven findings of the 2026-09-30 Codex design review" (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`), whose ADR-104 is this record's scope, and set the goal "deliver open tasks end to end". The record's text was drafted after that approval and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-032, ADR-067, ADR-074, ADR-080
**Governs:** `internal/check/check.go`, `internal/mcp/mcp.go`, `internal/read/read.go`, `internal/read/walk.go`, `internal/read/astgrep.go`, `internal/subproc/subproc.go`, `scripts/contract.sh`
**Enforced-by:** `internal/check/tail104_test.go::TestTheCheckTailReadsALogInBoundedMemory`
**Invalidates:** none — checked. ADR-032 and ADR-067 bound what a result SAYS; this record bounds what mrw READS to produce it. ADR-074's `--files-from` line cap (8 MiB) is the precedent for a named input cap.
**Served-path change:** (1) A check's tail line longer than 4 KiB is shown as its first 4 KiB and ` … [N more bytes]`; the tail and `truncated_lines` are otherwise unchanged. (2) `mrw mcp` answers a request line longer than 64 MiB with JSON-RPC -32600 (id null) naming the limit, and keeps serving; it read such a line whole before. (3) `mrw read` refuses a file over 1 GiB by name, UNREADABLE with its size and the limit, and `--grep` reports one it skipped for the same reason. (4) An ast-grep answer over 256 MiB is refused, naming its size and the limit. Exit codes keep their meanings.

## Context

**What was observed** (2026-09-30, the Codex design review of `f1d5996`, confirmed in source):

1. `check.lastLines` (`internal/check/check.go:746-759`) reads the whole check log with `os.ReadFile`, converts
   it to a string and splits it, to keep the last `tail_lines` (30) lines: a noisy check of G bytes holds about
   3G in memory for a 30-line answer.
2. `mcp.Serve` (`internal/mcp/mcp.go:203-208`) reads each request with `ReadString('\n')`, which grows without
   limit: one line with no newline is read whole. The comment there chose it over `bufio.Scanner`'s 64 KB cap
   because a write plan is a large message; nothing bounds it.
3. `read.go:451` and `walk.go:252` read every served or searched file whole with `os.ReadFile`, with no size
   check; `subproc.Output` (`internal/subproc/subproc.go:94-111`) reads ast-grep's whole answer with
   `io.ReadAll`. The result ceilings (ADR-032, ADR-067) bound what is sent, not what is read.

**Audit of the class.** The class is *an input mrw reads whole with no bound of its own*. Enumerated 2026-09-30
by `mrw read --grep 'os\.ReadFile|io\.ReadAll|ReadString\(' --exclude '*_test.go' internal/ cmd/` and reading each
site: the check log, the MCP request line, a served file, a searched file, the ast-grep answer — **5** — and a sixth
the first audit missed, found by the review of #295: ast-grep's CR-only probe reads each hit file whole
(`astgrep.go:180`). All six in scope.
**Left out:** the state files (`seen`, `iteration`, the tally), which mrw writes itself and bounds by what it
records; `apply`'s load of a file to edit, which a write reaches only after a read licensed the file (a file over
the cap is never served, so never licensed, `--force` aside); the plan file, which a caller writes and a CLI
caller controls; `.quality-harness.json`, a small config file.

## Existing Primitives Audit

- **The ADR-074 `--files-from` cap** (`cmd/mrw/main.go:2720`) — the model: a named limit and a refusal that says so.
- **`installFakeAstGrepJSON`** (`internal/read/leftovers_stress_test.go:474`) — reused for T4's test.
- **`errorResponse(null, codeInvalidRequest, …)`** (`internal/mcp/mcp.go:266`) — reused for T2's answer.

## Decision

1. `lastLines` streams the log with a ring of `tail_lines` lines and keeps at most 4 KiB of each; a longer line
   ends ` … [N more bytes]`. Line splitting keeps its meaning (on `\n`, a trailing `\r` kept), and
   `truncated_lines` still counts every earlier line. A passing check whose tail cut a line keeps its log, as one
   that withheld earlier lines does, and the marker counts a split rune's stray bytes; the ring grows as lines
   arrive, and a read that fails part-way answers no tail, as the whole-file read did (the review of #295).
2. `Serve` reads a request line up to 64 MiB; past that it answers -32600 with a null id naming the limit,
   discards the rest of the line, and reads the next one.
3. `read` refuses a file larger than 1 GiB before reading it, naming its size and the limit; the `--grep` walk
   reports it as a problem it skipped, and ast-grep's CR-only probe reports the hit once. Each reads through
   `readCapped`: a size check, then a read bounded at the limit plus one byte, because a file can grow after
   its size was taken (the review of #295).
4. `subproc.Output` takes a limit and refuses an answer larger than it — measured, then read through a bound —
   and `read.AstGrep` passes 256 MiB and names the size in its refusal, advising a narrower pattern or fewer paths.

Each limit is a package variable so its test can set a small one; no flag or environment variable exposes it.

## Alternatives Considered

- **Stream `read` instead of capping it** — rejected for now: every range address, pattern and balance check
  works on the whole file's lines, and a streaming rewrite is a new engine; a file over 1 GiB fails today by
  memory, and a named refusal replaces that.
- **Flags for the limits** — rejected: no caller has asked, and a knob is a contract (YAGNI).
- **A `bufio.Scanner` with a raised buffer for MCP** — rejected: it fails the session on an overlong token
  instead of answering it and reading on.

## Component / Boundary Impact

Engine packages owned: `internal/check` (T1), `internal/read` (T3, T4). `internal/mcp` and `internal/subproc`
are not engine packages. `internal/apply`, `internal/plan`, `internal/seen`, `internal/state`, `internal/lines`,
`internal/iter`, `internal/rooted` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| check receipts (`tail`) | a line over 4 KiB is capped with a marker | T1 | CLI, JSON callers |
| `mrw mcp` | -32600 for a request over 64 MiB; the session continues | T2 | MCP hosts |
| `mrw read`, `--grep` | a file over 1 GiB refused / skipped by name | T3 | callers |
| `mrw read --ast-grep` | an answer over 256 MiB refused | T4 | callers |
| `scripts/contract.sh` | §199 (T2), §200 (T1) | T1, T2 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| none | — | — | the four tasks are independent |

## Implementation

See `tasks/README.md`: T1 (the check tail), T2 (the MCP request), T3 (the file cap), T4 (the ast-grep cap).

## Consequences

- **Positive:** mrw's memory tracks its limits, not its inputs, on the five paths that read without a bound.
- **Negative:** a file over 1 GiB cannot be read or searched; a check line over 4 KiB is cut in the tail (the log
  file keeps it).
- **Neutral:** answers under the limits are unchanged.

## Out of Scope

- A contract row for the file and ast-grep caps (permanent: boundary: they need a file over 1 GiB or an answer over 256 MiB; each task's unit test sets the package variable small)
- Streaming `read` (permanent: boundary: every address form works on the whole file's lines; a streaming engine is a different design)
- `apply`'s load and the state files (permanent: boundary: a file over the cap is never licensed, and state files are written by mrw; qualified by ADR-107 on 2026-09-30: apply read the file whole before its licence was checked, so its load is now bounded at the same limit — the state-file half stands)
- A test for a check log that fails part-way, and for an ast-grep answer that grows after it was measured (permanent: boundary: no fixture reaches either — an I/O error mid-read, and a descendant writing after Run returned, which Windows allows for any grandchild and Unix for one that called setsid and left the group; both branches are named here as uncovered)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a legitimate plan over 64 MiB over MCP | Low | Med | the refusal names the limit and the CLI, which has none; the 2026-09-02 scale campaign's largest plan is far below it |
| the streamed tail differs from the old split for some input | Low | Med | T1's test compares the old and new answers on edge inputs (no newline, trailing newline, empty lines, CRLF) |

## Rollback

Revert T1–T4. No state or exit code changes.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `internal/check/tail104_test.go::TestTheCheckTailReadsALogInBoundedMemory` in T1's commit.
