# ADR-107: The last reads are bounded, and the recheck sees the leaf

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — within the goal "deliver open tasks end to end" and the approved plan to resolve the seven findings of the 2026-09-30 Codex design review (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`): the plan's last step re-ran that review on v1.37.0, which found finding 6 only partly closed and two defects in ADR-106. The record's text was drafted after that approval and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-104, ADR-106
**Governs:** `internal/apply/apply.go`, `internal/mcp/ack.go`
**Enforced-by:** `internal/apply/leaf107_test.go::TestTheRecheckRefusesALeafSwappedForALink`
**Invalidates:** ADR-104 Out of Scope *"`apply`'s load and the state files (permanent: boundary: a file over the cap is never licensed …)"*, for `apply`'s load: the final design review found that `apply` reads the whole file BEFORE its licence is checked, so the rationale does not hold; the state-file half stands.
**Served-path change:** (1) A plan that edits a file larger than 1 GiB is refused on that file's hunk, naming its size and the limit (exit 1, nothing written); `apply` used to read it whole first. (2) A file a plan edits whose name another process swapped for a link after validation is refused before its commit rename, naming the file, where the rename replaced the new link. (3) On Windows, a file whose identity cannot be read at validation is refused, naming it. MCP acknowledgement hashes a file without holding it in memory. Exit codes keep their meanings.

## Context

The final Codex design review of v1.37.0 (`b54f78e`, 2026-09-30, `codex-design2`) closed six of the seven findings
of the first and left three items open:

1. **Finding 6, partly closed.** `apply` calls `readLines` (`apply.go:514`, `:1738`), which reads the whole file with
   `os.ReadFile`, before the licence and sha gate (`:1004`): an unread file of any size reaches that allocation.
   MCP acknowledgement promotion hashes each acknowledged file with `os.ReadFile` (`ack.go:379`, `:455`).
2. **The identity recheck follows a final link** (`changedSince`, `apply.go:2016`, `os.Stat`): after staging `a`,
   another process moves the file to `b` and makes `a` a link to `b`; both rechecks see the original file and pass,
   and the commit replaces the link with a regular file, leaving `b` unchanged — against the preserved-link
   behaviour `stageFile` documents.
3. **On Windows a failed eager ID load is ignored** (`apply.go:477`): `os.SameFile(info, info)` returns false and
   leaves the ID to be loaded lazily later, which is the blindness ADR-106 loaded it eagerly to avoid.

**Audit of the class.** *A read of a whole file in the write path or the MCP acknowledgement, with no bound of its
own.* Enumerated 2026-09-30 by `mrw read --grep 'os\.ReadFile|io\.ReadAll' --exclude '*_test.go' internal/apply/
internal/mcp/ internal/writer/ cmd/mrw/`: `apply.readLines`, `ack.currentSHA`, and reads of the plan file, the
harness file and state files already dispositioned by ADR-104 — **2** in scope.

## Existing Primitives Audit

- **`read.readCapped`** (`internal/read/read.go:62`) — the size check then bounded read that `apply` copies.
- **`tree.lstat`** (`internal/apply/tree.go`) — the root-confined lstat the recheck uses for the leaf.

## Decision

1. `apply` refuses, at validation, a file a plan edits whose size is over `maxLoadBytes` (1 GiB, the same limit as
   `read`), on the file's hunk; `readLines` reads through a bound of the limit plus one byte, for a file that grows
   after its size was taken.
2. `currentSHA` streams the file into the hash (`io.Copy`), so an acknowledgement holds no file in memory.
3. `changedSince` first lstats the target through the tree: a link where validation saw a regular file is a change.
   A failed eager ID load at validation refuses the file.

## Alternatives Considered

- **Export `read`'s limit and share it** — rejected: `apply` importing `read` for one constant couples two engine
  packages; the limit is duplicated with a comment naming its twin.
- **Cap acknowledgement hashing instead of streaming** — rejected: streaming bounds memory with no new refusal.

## Component / Boundary Impact

Engine package owned: `internal/apply`. `internal/mcp` is not an engine package. Every other engine package stays
byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write`, `mrw_write` | a file over 1 GiB refused on its hunk | T1 | callers |
| `mrw_read` acknowledgement | hashing streams | T2 | MCP hosts |
| `mrw write`, `mrw_write` | a leaf swapped for a link refused before its rename | T3 | callers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| none | — | — | the three tasks are independent |

## Implementation

See `tasks/README.md`: T1 (the load limit), T2 (streamed acknowledgement hashing), T3 (the leaf and the ID).

## Consequences

- **Positive:** the write path and acknowledgement read no file without a bound; a link swapped in is not overwritten.
- **Negative:** a file over 1 GiB cannot be edited.
- **Neutral:** plans under the limits are unchanged.

## Out of Scope

- A contract row for the limit and the leaf swap (permanent: boundary: one needs a file over 1 GiB, the other a swap between two steps of one call; each task's test drives it through a small limit or the staging seam)
- A test of a failed ID load (permanent: boundary: no fixture makes Windows refuse a file's ID; the branch is named here as uncovered, and on darwin and Linux os.SameFile of a FileInfo with itself cannot fail)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a legitimate edit of a file over 1 GiB | Low | Low | the refusal names the limit; `read` already refuses such a file (ADR-104) |

## Rollback

Revert T1–T3. No state or receipt changes.

## Follow-ups

- None — the record carries no open follow-up.
