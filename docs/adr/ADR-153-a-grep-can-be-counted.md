# ADR-153: A grep can be counted without being served

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — approved the plan `glimmering-frolicking-harp` ("ADR-153 — `mrw read --count`") and set the goal "implement this plan end to end".
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-033, ADR-111, ADR-117, ADR-146
**Governs:** `internal/read/read.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `AGENTS.md`, `README.md`
**Enforced-by:** `cmd/mrw/count153_test.go::TestAGrepCountSaysHowManyLinesMatchAndLicensesNothing`
**Invalidates:** ADR-140 Decision 3's `-c`/`--count` sentence ("no count, `--stat` lists matching files"): `--count` is now a flag of `read`, so it leaves ADR-140's table, and `-c` points at it. Checked otherwise: `--stat` keeps its header and its empty observation; a `--grep` read without `--count` serves as before; ADR-146's `no match:` and `incomplete:` lines are reused, not changed
**Served-path change:** `mrw read --grep P --count` and `mrw_read` with `count: true` print each matching file's header with its count of matching lines and a total, serve no line and license none; the MCP receipt gains `counts`

## Context

The 14-session survey (2026-10-09, `docs/adr/BACKLOG.md`, "Look mode, what is left"; five sessions on counts) asked for a count: how many lines match, in which files, before deciding what to read. Today the answer costs a full serve of every match (`--grep P`) and counting the output, or `--grep P --stat`, which names the files and not how many lines in each. Callers reach for `grep -c`, which reads the files outside mrw and gives no sha.

The same survey asked for `--limit N` "that keeps mrw's exit code (`| head` loses it)". That flag exists: `--max-lines N` caps the lines served per spec, reports what it withheld, and exits 1 `incomplete:` (`internal/read/read.go`, `Options.MaxLines`; ADR-033). The sessions did not know it, which is a teaching gap, not a missing flag.

**Audit of the class** — *a read that answers a question about matches without serving them*: `mrw read --grep 'opt\.Stat|a\.Stat' internal cmd` names the one existing member, `--stat` (CLI) and `stat` (MCP), each answering "which files" and recording an empty observation. `--count` joins it on both surfaces; nothing else in the tree answers about matches without serving them.

## Existing Primitives Audit

- `read.Run`'s `--stat` branch already prints a file's header and records an empty span set (hashed, none shown). Counting is that branch plus one pass over the lines with the spec's pattern.
- `read.Walk` hands each matching file to `Run` as `path:/PATTERN/`, so the pattern to count is the spec's single range.
- ADR-146's `no match:` line already ends a grep that matched no file; a count that matched nothing takes the same path.

## Decision

1. **`mrw read --grep P --count`** prints, for each file the walk found, the header `--stat` prints with `  N matching line(s)` after it, then one total line `-- T matching line(s) in F file(s)`. A line counts once however many times it matches. No line is served; each file is recorded as `--stat` records it (its sha, no spans), so it licenses no write.
2. **Exit codes are a grep's**: 0 when something matched; 1 with ADR-146's `mrw: no match:` when no file did; 1 `incomplete:` when a path was refused.
3. **`--count` needs `--grep`.** Without it — a named range, `--files-from`, the working set — it is a usage error, exit 2, naming `--grep`. With `--ast-grep` it is refused too: an ast-grep hit is a span of lines, not a line, and a count of hits beside a count of lines in one flag would mean two things.
4. **Flags that act on served lines are refused beside it**, by name, exit 2: `--stat` (the count's header already carries it), `-C`, `--max-lines`, `--max-cols`, `-N`.
5. **`mrw_read` takes `count`** with `grep`, under the same refusals (`max_lines`, `stat`, `ast_grep`, no `grep`). Its receipt gains `counts`, the count per file keyed as `observed` is; an answer too large for one result is refused as a capped read is (`cappedOverflow`), never turned into an index, because an index carries no counts.
6. **No `--limit`.** `mrw instructions`, AGENTS.md and README teach `--max-lines N` as the replacement for `| head`: it keeps mrw's exit code and says what it withheld.

It fails if a count is wrong, if a counted line becomes writable, or if a count with nothing to count exits 0.

## Alternatives Considered

- **Count occurrences, not lines.** Rejected: every other answer of a grep is line-shaped (a span is lines, a licence is lines), and `grep -c` counts lines, which is what the sessions compared it to.
- **A `--limit N` that caps the whole answer.** Rejected for now: it is the cross-file budget BACKLOG defers ("A cross-file `--max-lines` budget"), and the ask it came from is answered by `--max-lines`.
- **Count `--ast-grep` hits.** Rejected: one flag would mean "lines" under one finder and "hits" under the other.

## Component / Boundary Impact

`internal/read` gains `Options.Count` and `Options.Counts` (an engine package; this record owns the change). `cmd/mrw` and `internal/mcp` select it.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --count` | new flag, with `--grep` only | T1 | the caller |
| `mrw_read` `count` | new argument; receipt key `counts` | T1 | the MCP host |
| `docs/receipts.txt` | `mcp_read counts` appended (ADR-111) | T1 | receipt readers |
| `mrw instructions`, AGENTS.md, README.md | `--count`; `--max-lines` in place of `\| head` | T1 | agents, people |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** a caller learns where and how much matches for the price of a header per file, inside mrw, with a sha it can compare later.
- **Negative:** one more flag on `read`, and five refusals beside it.
- **Neutral:** `--max-lines` is taught where `| head` was used; its behaviour does not change.

## Out of Scope

- A cross-file budget for a read (deferred: docs/adr/BACKLOG.md "A cross-file `--max-lines` budget")
- A JSON path address and `max_cols` on `mrw_read` (deferred: docs/adr/BACKLOG.md "Look mode, what is left" — planned as ADR-155 and ADR-154)
- Counting `--ast-grep` hits (permanent: boundary: a hit is a span, and one flag counts one unit)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A counted file becomes writable | Low | High | the test attempts a write to a counted line and requires the refusal |
| The count disagrees with what `--grep` serves | Low | Medium | the test compares the count with the lines `--grep` serves for the same pattern |

## Rollback

Revert T1: one option in `internal/read`, the flag and its refusals, the MCP argument, and the doc lines. The `counts` receipt key would then be a removal, which ADR-111 allows only at a major version; a rollback before release removes nothing shipped.

## Follow-ups

None.
