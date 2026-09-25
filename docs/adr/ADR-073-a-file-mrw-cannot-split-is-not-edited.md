# ADR-073: A file mrw cannot split into lines is not edited

**Status:** Accepted
**Accepted:** 2026-09-25 by M — *"plan to address these, properly, no looping on small details"*; the plan that groups the v1.25.1 adversarial round into five records was approved the same day
**Date:** 2026-09-25
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-007, ADR-065, docs/adr/BACKLOG.md
**Governs:** `internal/lines/lines.go`, `internal/lines/lines_test.go`, `internal/apply/apply.go`, `internal/apply/encoding_test.go`, `internal/read/read.go`, `internal/read/encoding_test.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/encoding_test.go::TestAUTF16FileIsRefusedNotRewrittenAsMixedEncodings`
**Invalidates:** none — checked: ADR-007 rules out encoding detection for DISCOVERED walk candidates, whose failure mode is silent omission; this refuses a named file by name and leaves the walk alone
**Served-path change:** A line edit to an existing file that begins with a UTF-16 or UTF-32 byte-order mark, or holds a NUL byte in its first 8 KiB, is refused per hunk, exit 1, nothing written, naming the encoding; so is any edit of a path that is not a regular file (a FIFO no longer blocks the write). `create`, `unlink` and `rename` of an encoded file are unaffected. `mrw read` serves such a file as before, with one `-- note:` line under its header.

## Context

**What was observed.** The v1.25.1 adversarial round (2026-09-25; `docs/adr/BACKLOG.md`, "From the
v1.25.1 adversarial round") wrote to a UTF-16LE file (BOM `FF FE`): it was served as byte-split
lines, and a replace rewrote it at exit 0 with the BOM gone and two encodings mixed. Nothing refuses
a write to a file mrw cannot split into lines. The splitter, `internal/lines.Split`
(`lines.go:17-27`), splits bytes by `\n`; UTF-16 puts a NUL beside every ASCII byte, so a "line"
is half a character pair and a replace body in UTF-8 lands between them.

The same round found that a FIFO named in `mrw read` blocked forever. The write path has the same
shape: `readLines` (`apply.go:1518`) calls `os.ReadFile` on whatever the plan names.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `lines.Split` | `internal/lines/lines.go:17` | The one splitter (ADR-065); no encoding step. |
| `readLines` | `internal/apply/apply.go:1518` | Reads the whole file, then splits. The bytes are in hand. |
| The ADR-021 stat | `internal/apply/apply.go:408` | Already stats every addressed file before reading it. |
| walk's regular-file rule | `internal/read/walk.go:124-128` | "not a regular file: mrw would block on a pipe or stream a device without end" — the wording to reuse. |

## Decision

1. **`lines.Unsplittable(b)` names what makes bytes unsplittable**: a UTF-32 BOM (`FF FE 00 00`,
   `00 00 FE FF`, checked first because `FF FE` is its prefix), a UTF-16 BOM (`FF FE`, `FE FF`), or
   a NUL byte in the first 8 KiB; otherwise nothing.
2. **A line edit to such a file is refused**, one refusal per file on its first hunk, siblings
   skipped: `<path> is UTF-16 (BOM FF FE): mrw edits UTF-8 text by line and would mix encodings`.
   `--force` does not bypass it: it overrides the read ledger, not the file's encoding. `unlink` and
   `rename` do not split lines and are unaffected; a `create` has no existing bytes.
3. **A path that is not a regular file is refused before it is read**, with walk.go's reason.
4. **`mrw read` notes it and serves as before**: `-- note: <path> is UTF-16 (BOM FF FE): served as
   bytes; a write to it is refused`. No exit-code change: the read still served what was asked.

## Alternatives Considered

- **Transcode UTF-16 to UTF-8 and back.** Rejected: mrw edits bytes by line (ADR-005), and a
  round trip would rewrite every line the plan did not name.
- **Refuse on read too.** Rejected: the read served the bytes faithfully in the round, and a caller
  may need to see the file to decide what to do with it.

## Component / Boundary Impact

`internal/apply`, `internal/lines` and `internal/read` are engine packages and this record owns the
changes named above. Byte-identical: `internal/plan`, `internal/seen`, `internal/check`,
`internal/state`, `internal/iter`, `internal/rooted`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` | an encoded or non-regular file is refused per hunk | T1 | CLI, MCP `mrw_write` |
| `mrw read` | a `-- note:` line under an encoded file's header | T2 | CLI, MCP `mrw_read` |
| contract §146 | T1, T2 | T1 | CI, `adr-verify` |

## Inter-task Contracts

T2 consumes T1's `lines.Unsplittable`.

## Implementation

See `docs/adr/ADR-073-a-file-mrw-cannot-split-is-not-edited/tasks/README.md`.

## Consequences

- **Positive:** a UTF-16 file can no longer be corrupted by a line edit at exit 0.
- **Negative:** a binary file with a NUL in its first 8 KiB cannot be line-edited; it never could be
  safely.
- **Neutral:** no exit code changes meaning.

## Out of Scope

- Editing UTF-16 or UTF-32 text (permanent: boundary: mrw edits bytes by line; ADR-005)
- A NUL after the first 8 KiB (permanent: fact: the scan is bounded so a large file costs one prefix read; citation: file `internal/lines/lines.go:49`)
- Encoding detection in the walk (permanent: boundary: ADR-007 keeps discovery free of content heuristics)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A UTF-16LE file whose first character is U+0000 reads as UTF-32 | Low | Low | the refusal names the wrong width; the file is still refused |

## Rollback

Revert the guard and the note. Nothing persistent moves.

## Follow-ups

- [ ] Release with ADR-071, ADR-072, ADR-074 and ADR-075 as v1.26.0.
