# ADR-091: A receipt names files the way a plan does

**Status:** Accepted
**Accepted:** 2026-09-28 by Zy — "windows - fix", then "Accept" on the record as drafted; amended the same day to the write receipt on "Extend to write receipt (Recommended)", after the Codex review of #262
**Date:** 2026-09-28
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-010, ADR-023, ADR-039, ADR-071, ADR-076, ADR-090
**Governs:** `internal/mcp/tools.go`, `internal/mcp/wirepath.go`, `internal/mcp/schema.go`
**Enforced-by:** `internal/mcp/wirepath_test.go::TestAReceiptKeyIsSpelledWithSlashes`
**Served-path change:** on Windows only, every root-relative path an MCP receipt carries is spelled with `/`: `mrw_read`'s `observed` keys, and `mrw_write`'s `hunks[].path`, `files[].path`, `files[].renamed_to`, `files[].target` and `dirs_created`, in the structured receipt and the text report alike — where they carried `\`. Unchanged on every other platform, where the separator is already `/`. `root` stays an OS path, and `next_read` echoes the caller's spec.

## Context

`mrw_read`'s receipt (content[1], ADR-023) maps each served file to what was observed of it. The map
comes straight from `read.Run`, which keys a file by `filepath.Clean(path)`
(`internal/read/read.go:357`) — the same key the ledger and `apply` use, so "a.go" read and
"./a.go" written are one file. On Windows that key is backslashed.

`mrw_write`'s receipt has the same property: `boundedReceipt` serializes the engine's `apply.Result`
(`internal/mcp/tools.go:781`), whose paths the engine cleaned (`internal/apply/apply.go:337`), so
`hunks[].path`, `files[].path`, `renamed_to`, `target` and `dirs_created` are backslashed on Windows
too. A plan names files with `/`, and a grep index is `filepath.ToSlash`-ed
(`internal/read/walk.go:268`), so a Windows caller cannot match a receipt's path to the plan line
or the spec that produced it. ADR-090's read test did exactly that and failed on windows-shard 1
(#261), found from source too by the Codex review of #261; the Codex review of #262 found that the
first draft of this record claimed the write receipt was already slash-spelled, which it was not.
The published descriptions say only "keyed by path" and "relative to root".

## Existing Primitives Audit

- The three read-receipt builders that serialize `observed`: `tools.go`'s `readResult` calls (the
  served and the marked answer) and `pagedResult`. The index and no-match answers carry an empty map.
- `boundedReceipt`, the one path by which an `apply.Result` reaches the wire (`tools.go:709`), for the
  whole receipt and the elided one.
- The ledger, the ack store and `markServed` key by the OS path internally; none of them is a wire
  field, and changing their key would move every Windows ledger entry.

## Decision

1. Every root-relative path in an MCP receipt is spelled with `/`, as a plan names it: `mrw_read`'s
   `observed` keys, and `mrw_write`'s `hunks[].path`, `files[].path`, `files[].renamed_to`,
   `files[].target` and `dirs_created`.
2. The conversion happens where a receipt is built, in `internal/mcp`: `slashKeys(m, sep)` on the
   map handed to a read receipt, and `slashResult(res, sep)` at the top of `boundedReceipt`, which
   copies the slices so the engine's result is not changed. The engine, the ledger and the ack store
   keep their keys.
3. Both helpers take the separator as a parameter so a test drives them with `\` on any platform
   (ADR-071's `throughLinks` precedent: a boundary proved on every OS, not only the one that has it).
4. `root` stays an OS path: it is absolute, and a caller hands it to its own filesystem. `next_read`
   echoes the spec the caller sent, so following it verbatim works on any spelling.
5. The descriptions of `observed`, `hunks.path`, `files.path`, `files.renamed_to`, `files.target`
   and `dirs_created` say the paths are slash-spelled.
6. ADR-090's read test compares the receipt's keys as they come, and a write test compares the
   write receipt's paths as they come, so windows-shard 1 proves the wiring.

## Alternatives Considered

- **Key the engine by slash.** Rejected: `read.Run`'s key is the ledger's and `apply`'s; changing it
  moves every Windows ledger entry and touches three engine packages for a wire spelling.
- **Document the OS spelling.** Rejected: a plan and a grep index are slash-spelled, so the receipts
  would stay the fields a plan author cannot use as written.
- **Fix `observed` only.** The first accepted draft; rejected on review of #262, because the write
  receipt carries the same defect and "the receipt names files the way a plan does" is one promise.

## Component / Boundary Impact

`internal/mcp` only. No engine package changes.

## Wiring & Contract Changes

On Windows the listed fields change spelling; value shapes, the ledger, `root`, `next_read` and every
other field are unchanged. Six descriptions in `tools/list` change, so `legacy_golden.jsonl` is
regenerated.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- A Windows caller can match a receipt's paths to its plan and its specs, and use an `observed` key
  in a plan as written.
- A caller that parsed backslashed paths sees slashes; no caller on another platform sees a change.

## Out of Scope

- The CLI's served text, `==>` headers and write report (permanent: boundary: the CLI receipts are separate surfaces; this record owns the MCP wire)
- A contract row (permanent: boundary: `scripts/contract.sh` runs on Linux, where the separator is already `/`; the Windows CI shard runs the Go tests instead)
- A reason string that quotes a path (permanent: boundary: prose written by the engine, not a path field a caller matches)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A Windows caller parsed backslashed paths | Low | Low | slashes are what a plan uses; noted in the release |
| A converted path reaches the ledger | Low | High | both helpers act on copies handed to a receipt; the ledger is written from `read.Run`'s map and the engine's own result |

## Rollback

Revert the tasks.

## Follow-ups

None.
