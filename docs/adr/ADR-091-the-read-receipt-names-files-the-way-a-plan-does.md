# ADR-091: The read receipt names files the way a plan does

**Status:** Accepted
**Accepted:** 2026-09-28 by Zy — "windows - fix", then "Accept" on the record as drafted
**Date:** 2026-09-28
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-010, ADR-023, ADR-039, ADR-071, ADR-090
**Governs:** `internal/mcp/tools.go`, `internal/mcp/wirepath.go`
**Enforced-by:** `internal/mcp/wirepath_test.go::TestAReceiptKeyIsSpelledWithSlashes`
**Served-path change:** on Windows, `mrw_read`'s receipt keys `observed` as `internal/store/store.go` where it said `internal\store\store.go`. Unchanged on every other platform, where the separator is already `/`.

## Context

`mrw_read`'s receipt (content[1], ADR-023) maps each served file to what was observed of it. The map
comes straight from `read.Run`, which keys a file by `filepath.Clean(path)`
(`internal/read/read.go:357`) — the same key the ledger and `apply` use, so "a.go" read and
"./a.go" written are one file. On Windows that key is backslashed.

Every other path mrw hands a caller is slash-spelled: a plan names files with `/`, `mrw_write`'s
`hunks.path` and `files.path` echo the plan, and a grep index is `filepath.ToSlash`-ed
(`internal/read/walk.go:268`). So a Windows caller that reads a file and looks its observation up by
the path it asked for, or by the path it will put in a plan, finds nothing. ADR-090's read test did
exactly that and failed on windows-shard 1 (#261); the Codex review of #261 found it from source.
The published description says only "keyed by path".

## Existing Primitives Audit

- The three receipt builders that serialize `observed`: `tools.go` `readResult` calls (the served
  and the marked answer) and `pagedResult`. Nothing else puts a file key on the wire.
- The ledger, the ack store and `markServed` key by the OS path internally; none of them is a wire
  field, and changing their key would move every Windows ledger entry.

## Decision

1. `observed` is keyed on the wire by the root-relative path with `/` separators, as a plan names it.
2. The conversion happens where the receipt is built, in `internal/mcp`, by one helper
   `slashKeys(m, sep)` called with `filepath.Separator`. The engine, the ledger and the ack store
   keep their keys.
3. The helper takes the separator as a parameter so a test drives it with `\` on any platform
   (ADR-071's `throughLinks` precedent: a boundary proved on every OS, not only the one that has it).
4. `observed`'s description says the keys are slash-spelled root-relative paths.
5. ADR-090's read test compares the receipt's keys as they come, so windows-shard 1 proves the
   wiring.

## Alternatives Considered

- **Key the engine by slash.** Rejected: `read.Run`'s key is the ledger's and `apply`'s; changing it
  moves every Windows ledger entry and touches three engine packages for a wire spelling.
- **Document the OS spelling.** Rejected: every other path on the wire is slash-spelled, so the read
  receipt would stay the one field a plan author cannot use as written.

## Component / Boundary Impact

`internal/mcp` only. No engine package changes.

## Wiring & Contract Changes

On Windows the `observed` keys change spelling; the value shape, the ledger and every other field
are unchanged.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- A Windows caller can use a key from `observed` in a plan, and look a served file up by the path it
  asked for.
- A caller that parsed backslashed keys sees slashes; no caller on another platform sees a change.

## Out of Scope

- The CLI's served text and `==>` headers (permanent: boundary: human-read output; the CLI read has no JSON receipt)
- A contract row (permanent: boundary: `scripts/contract.sh` runs on Linux, where the separator is already `/`; the Windows CI shard runs the Go test instead)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A Windows caller parsed backslashed keys | Low | Low | slashes are what every other field uses; noted in the release |
| A key is converted where the ledger reads it | Low | High | the helper is called only on the map handed to `readResult`/`pagedResult`; the ledger is written from `read.Run`'s map |

## Rollback

Revert the task.

## Follow-ups

None.
