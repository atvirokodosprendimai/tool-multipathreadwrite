# ADR-166: A read that cannot record still serves, and says nothing was recorded

**Status:** Accepted
**Accepted:** 2026-10-21 by Zy — chose "Serve, exit 0, note (Recommended)" when asked what a read whose ledger cannot be written should do, after a Codex agent reported that a read-only run of v1.70.0 "still returns exit 2 because of seen.lock, though it delivers the content" (pasted by Zy, in Lithuanian); then "accepted" for the record as drafted, including Decision 2 for `mrw_read`.
**Date:** 2026-10-21
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-033, ADR-079, ADR-108, ADR-133, ADR-146, ADR-148, ADR-164
**Governs:** `cmd/mrw/main.go`, `internal/mcp/tools.go`
**Enforced-by:** None — two surfaces; each task's test and contract row (§284, §285) hold its own decision
**Invalidates:** ADR-164 — T3's assertion that a content read with an unusable state directory exits 2: `cmd/mrw/statehint_test.go`'s `{"read", "a.txt"}` row, contract §272's "a read whose state directory cannot be made exits 2", and `internal/mcp/statehint164_test.go`'s two `mrw_read` content rows ("a read holds its checkpoints", "a paged read holds its checkpoints"); T1 and T2 update them. ADR-164's plan-read, quiet, ack and write rows, and its hint, stand. Also superseded, from no record: contract §244 and §245 (the bounded fixes #377 and #381, BACKLOG-archive), which kept exit 2 for a read that had served its lines when the ledger could not be updated (a Windows holder) or opened; their read rows now expect the read's own exit, their message and licence checks kept. ADR-133 is extended, not changed.
**Served-path change:** `mrw read` with an unwritable or uncreatable state directory serves its answer, exits as the read itself would (0, or 1 when a range was not served), and ends standard error with one line saying nothing was recorded and why; `mrw_read` in the same state serves the answer without checkpoints and says so, where it used to fail with JSON-RPC error -32603.

## Context

**A field report**, 2026-10-21, pasted by Zy from a Codex agent that runs mrw in a sandbox with a read-only home: on v1.70.0 a read "still returns exit 2 because of seen.lock, though it delivers the content". Reproduced the same day on v1.70.0 (`~/.local/bin/mrw`), three shapes:

1. the state directory exists but is read-only: `mrw read f.txt:1` serves the lines, then exits 2 with `open …/seen.lock: permission denied — mrw's state directory, …: set XDG_STATE_HOME to a writable absolute directory` (`cmd/mrw/main.go`, `seen.Record` after the flush);
2. the state directory cannot be created: `mrw read f.txt:1` serves nothing and exits 2 with `mkdir …: permission denied — …` — the read fails before serving, in `iter.Load` (`internal/iter/iter.go`, `load` → `ReadPath`), which it calls to resolve working-set references even when the caller named none;
3. over MCP, `mrw_read` of `f.txt` fails outright: JSON-RPC error -32603 `holding checkpoints: mkdir …: permission denied — …` (`internal/mcp/tools.go`, the `hold` call after the answer is built), so nothing is served.

A write in the same state exits 2, `mkdir …: permission denied`, and must: it cannot check a licence it cannot read, and it cannot record what it wrote.

ADR-133 already settled the neighbouring case: a read whose answer goes to the null device served nobody, records nothing, says so on stderr, and keeps the read's own exit code. A read whose ledger cannot be written is the mirror image: the answer reached the caller and nothing could be recorded. Today it exits 2, which a caller reads as "the read failed" and retries, though the answer arrived.

**Audit of the class** — *a read path that touches mrw's state directory before or after serving*. Enumerated 2026-10-21 by `grep -n 'seen.Record\|iter.Load\|hold(root\|promote\|recordQuiet\|seen.Record(' cmd/mrw/*.go internal/mcp/*.go internal/read/plansource.go`, and widened by the Codex challenger of this record, which found the routes the first grep missed:
- **Content reads**, which this record changes: the CLI read's two `iter.Load` sites (with paths after `--grep`/`--ast-grep`, and the default branch with arguments or `--files-from`) and its `seen.Record` after the flush; `mrw_read`'s two `hold` sites (a whole answer, `internal/mcp/tools.go` near 690, and a page, near 1880).
- **Licence reads**, which keep refusing, as Decision 3 says: the CLI's `--quiet` and plan reads (`--sub`, `--rewrite`, `cmd/mrw/sub.go` through `internal/read/plansource.go`); `mrw_read`'s `quiet` and its `sub`/`rewrite` plans (`internal/mcp/sub161.go`); `mrw_read` with `ack`, whose promotion runs before serving (`internal/mcp/ack.go`).
- **Already tolerant**: the `Before` hook's ledger checks (`seen.Newer`, `seen.IsStale`, `seen.DamageNotice`), the legacy migration, ADR-148's "still licensed" lookup and MCP's newer-ledger notice ignore their errors. `mrw read -` records nothing (ADR-161).
- **A boundary kept**: a hard link into the state directory is refused when the state cannot be inspected (`internal/rooted/hardlink.go`, ADR-134); that refusal is not a state *write* and is unchanged.
Writes, `mrw iter`, `mrw seen` and `mrw stats` are left out: each needs the state to do its job.

## Existing Primitives Audit

- `statehint.About(err)` (ADR-164 T3, `internal/statehint`) recognises a filesystem failure in mrw's own state directory by its type, and `statehint.Hint(err)` names it with `XDG_STATE_HOME`; both decisions use them, so no new classification exists.
- ADR-133's null-device branch in the CLI read action is the precedent for "the answer reached the caller and nothing was recorded": a stderr line, the read's own exit code.
- `mrw_read` already serves lines it cannot checkpoint, without a checkpoint and with a line saying so (ADR-108, invalid UTF-8); Decision 2 uses the same shape for a whole answer.

## Decision

1. **The CLI content read serves, records nothing new, and says so (T1).** In `mrw read`'s action, a state-directory failure (`statehint.About`) from `seen.Record` no longer exits 2 for a content read: the answer has been flushed, and the read ends with one stderr line, `mrw: nothing was recorded, so a write to these lines will be refused until a read can record them: <statehint.Hint(err)>`, and the read's own exit code (0, or 1 `incomplete:` / `no match:` as before, ADR-146). A state-directory failure from `iter.Load` is skipped when the effective specs — after `--files-from` is expanded — hold no working-set pointer (`iter.IsPointer`) and the read names at least one spec, since the set is then not needed; at both `iter.Load` sites. A read that needs the set (no specs, or a pointer) still exits 2 when loading it fails, because it cannot know what to serve. Any other error is unchanged.
2. **`mrw_read` serves a content read without checkpoints, and says so (T2).** When `hold` fails with a state-directory failure, `mrw_read` returns the answer as built before the checkpoint markers were added (`served`, whose receipt holds observations and no checkpoint id), not an error, with one line at the top of `content[0]`: `-- no checkpoint was held, so nothing here can be acknowledged and a write to it is refused: <statehint.Hint(err)>`. A page is rebuilt the same way from that page's own unmarked text, its range, observations and `next_read` kept; the outer `served` is not that page. Any other `hold` failure is unchanged.
3. **A read whose purpose is the licence keeps refusing.** `--quiet` and plan reads (`--sub`, `--rewrite`) on the CLI, and `mrw_read`'s `quiet`, `sub`/`rewrite` and `ack`, exist to record a licence or to feed a write; without a writable ledger they cannot do their job, and a plan piped into `mrw write` would fail there anyway. They exit 2 / answer the error as today (ADR-164 T3).

**What would falsify it:** a content read with an unwritable state directory exiting 2 when every range was served; a new licence recorded by it (a later write to lines it served applying, when no earlier read had licensed them — an earlier licence survives, as ADR-133 says); a read that needs the working set serving something; `mrw_read` in that state answering a content read with an error, or with a checkpoint marker an ack could name; a `--quiet` or plan read, or an `ack`, succeeding without a ledger.

## Alternatives Considered

- **Keep exit 2, serve in both cases:** the caller cannot miss the missing licence. Rejected by Zy (2026-10-21): an exit 2 reads as a failed read and invites a retry of an answer that arrived; ADR-133 already says the read's own exit plus a stderr line is how mrw reports "nothing recorded".
- **Fall back to a temporary state directory:** the read would record somewhere. Rejected: a licence in a directory the next process cannot find licenses nothing, and pretending it does is ADR-002 inverted.
- **Leave it as it is:** rejected by Zy.

## Component / Boundary Impact

None — no module added or moved; two call sites in `cmd/mrw` and two in `internal/mcp` change what they do with an error `statehint` already classifies.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` exit code | 0 (or the read's own 1) instead of 2 when only the ledger could not be written | `cmd/mrw/main.go` | shell callers, sandboxed agents |
| `mrw read` stderr | one `nothing was recorded` line naming the state directory | `cmd/mrw/main.go` | the same |
| `mrw_read` | an answer without checkpoints and a top line, instead of JSON-RPC -32603 | `internal/mcp/tools.go` | MCP hosts |
| README §Exit status, AGENTS.md | one sentence each | T1, T2 | callers |
| `scripts/contract.sh` §284, §285 | one row per decision, each with its pair | T1, T2 | CI |

## Inter-task Contracts

None — T1 and T2 change separate surfaces and share only `statehint`, which neither changes.

## Implementation

`tasks/README.md`; `tasks/T1-the-cli-read-serves-and-says-nothing-was-recorded.md`, `tasks/T2-mrw-read-serves-without-checkpoints.md`. Independent; each a red test, the change, a killed mutant, a contract row, the gates, and a Codex review.

## Consequences

- **Positive:** a sandboxed agent reads at exit 0 and is told in one line why a later write will be refused, and by what fix.
- **Negative:** a caller who treated exit 2 as "the ledger is broken" must read stderr instead.
- **Neutral:** writes and every other state-dependent command are unchanged.

## Out of Scope

- Writing without a ledger (permanent: boundary: a write must check a licence and record what it wrote, ADR-002 and ADR-005)
- A temporary or in-memory ledger for a read-only run (permanent: boundary: a licence the next process cannot find licenses nothing)
- `mrw iter`, `mrw seen`, `mrw stats` in a read-only state (permanent: boundary: each exists to read or change the state)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A real ledger fault (not permissions) is hidden behind exit 0 | Low | Med | only `statehint.About` errors take the new path; every other error keeps exit 2; the stderr line names the cause either way |
| A caller ignores stderr and is surprised by a refused write | Med | Low | the write's refusal names the unread lines and the state hint, as today |

## Rollback

Revert the two call-site changes; no state or receipt key changes.
