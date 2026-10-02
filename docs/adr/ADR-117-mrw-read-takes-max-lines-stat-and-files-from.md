# ADR-117: mrw_read takes max_lines, stat and files_from

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — "max_lines (Recommended), stat (Recommended), files_from", the answer to "Item 3b — which of the CLI's read flags should mrw_read gain?", on the refreshed gap list of 2026-10-02 and the plan he approved the same day. The record's text was drafted after that answer and not shown to Zy before execution: this line is the answer, not a review of these words.
**Date:** 2026-10-02
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-033, ADR-039, ADR-077, ADR-093, ADR-104, ADR-109
**Invalidates:** ADR-093 — only its Out of Scope routing of `--max-lines`, `--stat` and `--files-from` to the CLI; and AGENTS.md's "`--files-from` has no MCP equivalent and does not need one"
**Governs:** `internal/mcp/tools.go`, `internal/mcp/ack.go`, `internal/mcp/mcp.go`, `internal/mcp/instructions.go`, `internal/speclist/speclist.go`, `cmd/mrw/main.go`, `AGENTS.md`, `README.md`, `cmd/opencode/mrw-plugin/src/index.ts`, `scripts/contract.sh`
**Enforced-by:** `internal/mcp/readargs117_test.go::TestMrwReadTakesMaxLinesStatAndFilesFrom`
**Served-path change:** `mrw_read` takes `max_lines` (a cap per spec, as `--max-lines`), `stat` (headers only, as `--stat`) and `files_from` (a root-relative file of specs, as `--files-from FILE`). Each licenses exactly what it serves: a capped read licenses the lines it served, a stat licenses nothing.

## Context

A caller with no shell had `specs`, `grep`, `ast_grep` and `exclude`, and was routed to the CLI for the rest. The refreshed gap list of 2026-10-02 named three that matter on a host with no shell: a cap, so a large file can be sampled without paging it whole; a stat, so "did this change, how long is it" costs no content; and a list of specs kept in a file, since a long list sent inline is a large argument the host must carry. `read.Options` already has `Stat` and `MaxLines` (ADR-033: a nil cap is no cap, zero is headers only); MCP calls `read.Run` with neither.

**Audit of the class** — *a read flag the CLI has and MCP routes away*: `mrw read --grep 'cliRoute|"--files-from"|max_lines|"stat"' internal/mcp cmd/mrw` names `cliRoute["mrw_read"]` (`--files-from`, `--max-lines`, `--stat`), the `mrw_read` description, the instructions line, and the tests that use those three as undeclared examples (`undeclared093_test.go` rows 70, 71, 156, 186; `mcp_test.go` 512). `--context` and `--no-numbers` stay CLI-only: the routing names them now.

## Existing Primitives Audit

- **`read.Options.Stat`, `.MaxLines`** (ADR-033) — the cap and the stat, with WITHHELD reported as a problem.
- **`specList`** (`cmd/mrw/main.go`) — the list parser: blank lines and `#` comments skipped, a line kept as written, 8 MiB a line. Its scanning moves to `internal/speclist` so both surfaces read a list one way.
- **`rooted.Resolve`, `rooted.InState`, `regular.Open`** (ADR-071, ADR-077, ADR-109) — the boundary, mrw's own state, and an open that never waits on a FIFO.

## Decision

1. **`max_lines`** is an integer cap per spec, as `--max-lines`: absent is no cap, `0` serves headers only, a negative value is refused. A capped read is checkpointed exactly for the numbered lines it served; WITHHELD lines are reported and license nothing. When a named capped read is still too large for one answer it is refused with the reason, never paged past the cap the caller set. A `grep` or `ast_grep` too large to serve answers with its INDEX, capped or not: an index serves no line and licenses nothing, and the caller sends the files it wants back with the same cap.
2. **`stat`** serves each file's header — lines, bytes, sha — and no content, as `--stat`. It holds no checkpoint, so it licenses nothing (ADR-002).
3. **`files_from`** names a file of specs, resolved inside the root, read the way `--files-from FILE` reads one: blank lines and `#` comments skipped, each line a spec as written, 8 MiB a line — one parser, `internal/speclist`, for both. It is refused beside `specs`, `grep` or `ast_grep` ("two sources of specs; use one"), when empty, or when it holds no spec, as the CLI's is; and, because a server takes it from a caller rather than from its own shell, also as `-` (stdin is the protocol), outside the root, inside mrw's own state directory, when not a regular file (a FIFO is refused at once), and over 64 MiB, a list that grows past that while read included. The CLI's `--files-from` keeps its rules: its file is the invoking shell's argument.
4. **The routing moves.** `mrw_read` no longer sends a caller to the CLI for these; the CLI's remaining read-only extras, `--context` and `--no-numbers`, are what the routing names.

## Alternatives Considered

- **`files_from` left out** — rejected by Zy's answer: a list kept in the checkout is how a long, re-used read is sent without carrying it in every call.
- **A capped read that pages past its cap** — rejected: the caller asked for at most N lines a spec, and a next_read beyond them answers a question nobody asked.

## Component / Boundary Impact

`internal/mcp`, a new non-engine `internal/speclist`, `cmd/mrw` (its parser moves), the opencode plugin. No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_read` input schema | `max_lines`, `stat`, `files_from` | T1 | MCP callers |
| `internal/speclist.Parse` | the list parser, shared | T1 | CLI, MCP |
| descriptions, instructions, routing, AGENTS.md, README, plugin | say so | T1 | readers |
| `scripts/contract.sh` | §217 | T1 | CI Linux |

No receipt key is added: a stat answer and a capped answer carry the keys a served read carries.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a caller with no shell samples a file, asks for its length and sha, or sends a stored list of specs.
- **Negative:** three more arguments to describe and keep in step with the CLI.
- **Neutral:** the CLI's behaviour is unchanged; its list parser moved.

## Out of Scope

- `--context` and `--no-numbers` over MCP (deferred: `docs/adr/BACKLOG.md` "From ADR-117")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a capped read licenses a line it did not serve | Low | High | checkpoints wrap numbered lines only; the test writes past the cap and is refused |
| a files_from path escapes the root or blocks the server | Low | High | `rooted.Resolve`, `InState`, `regular.Open`; tested with `..`, a link out, and a FIFO |
| two versions of a file served in one read share an 8-hex sha prefix, so a replaced version's spans are held | Low | High | the prefix is what the header prints; a chance collision is 1 in 2^32 per pair, and a writer able to forge one can already change the file — carrying the full sha beside the text is in BACKLOG "From ADR-117" |

## Rollback

Revert T1. No receipt key is added, so nothing ADR-111 holds is removed.

## Follow-ups

- None — the record carries no open follow-up.
