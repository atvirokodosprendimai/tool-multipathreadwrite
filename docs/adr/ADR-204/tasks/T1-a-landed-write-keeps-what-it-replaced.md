# Task ADR-204-T1: a landed write keeps what it replaced

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** package `internal/undo` (the store: `Keep`, `Entries`, `Forget`); the entry format; the receipt line when nothing was kept; contract §350
**Consumes:** `writer.Land` (`internal/writer/flow.go`), apply's per-file result and the bytes it validated against, the checkout's state directory
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a landed write leaves one entry holding the bytes and mode of every file it changed or removed`, `the newest 16 are kept`, `a write over the size limit, a partial landing and a dry run keep none and say so`, `the store is in the checkout's state, never in the tree`, `the other engine packages are untouched`

## Goal

ADR-204 Decisions 4 and 5. After each of these, the checkout's `undo/` holds what the table says:

| Call | Entry |
|------|-------|
| a plan editing `a.txt` and `b.txt` | one entry: both paths, each with its bytes and mode before, `sha_before`, `sha_after` |
| `mrw write --create c.txt` | one entry: `c.txt` created, no before-image, the directories the write made |
| a plan `@@ a.txt - unlink` | one entry: `a.txt` removed, its bytes and mode |
| a plan renaming `a.txt` to `d/e.txt` with an edit | one entry: the source's bytes before, the destination, `sha_after` |
| `mrw write --grep X --sub Y .` | one entry, as a native plan's |
| `mrw_write` over MCP | one entry, as the CLI's |
| a write through an in-root link | the entry holds the file the link led to |
| a dry run; a refused plan | no entry |
| a landing whose before-images exceed the limit (the limit lowered through a test seam) | no entry; the receipt says `undo: not kept` with the byte count; the walk back ends there |
| a partial landing (a commit step made to fail through the existing seam) | no entry, and the walk back ends there |
| 17 landed writes | 16 entries, the oldest gone |

The directory is `undo/` inside the checkout's state directory, 0700, its files 0600; no file appears in the tree; a tree whose state directory is inside it (ADR-077) serves and edits nothing under `undo/`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/undo/undo.go`, `internal/undo/undo_test.go` | new | the store |
| `internal/writer/flow.go` | edit | `Land` keeps the entry under the write lock |
| `internal/apply/apply.go` | edit | hands over the bytes and mode it validated against |
| `internal/writer/undo204_test.go`, `cmd/mrw/keep204_test.go` | new | the table through `Land` and through the binary |
| `scripts/contract.sh` | edit | §350 |
| `docs/receipts.txt` | edit | only if the not-kept line is a receipt key |

## Ordered Steps

1. [S1] The tests, RED. [proof: acceptance]
2. [S2] The store and `Land`. Mutants, each killed: the before-image is taken after the write; the mode is not kept; a removed file's bytes are not kept; a dry run keeps an entry; the seventeenth write keeps seventeen; an over-limit write keeps a truncated entry; a partial landing keeps an entry; the files are created 0644. [proof: mutation]
3. [S3] Contract §350 drives the built binary: a landed edit leaves an entry outside the tree whose before-image is the file's earlier bytes; paired with a dry run leaving none. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/undo ./internal/writer ./cmd/mrw -run '^(TestAWriteKeepsWhatItReplaced|TestLandKeepsOneEntryPerLandedWrite|TestTheBinaryKeepsAnUndoEntryOutsideTheTree)$' > "${TMPDIR:-/tmp}/adr204t1-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr204t1-fence.out")" -eq 3 ] \
  && grep -q '^# 350\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteKeepsWhatItReplaced` | `internal/undo/undo_test.go` | the store: an entry round-trips bytes and modes exactly, the 16 kept, the size limit, permissions | none | S1, S2 |
| `TestLandKeepsOneEntryPerLandedWrite` | `internal/writer/undo204_test.go` | every row of the Goal's table through `Land` | none | S1, S2 |
| `TestTheBinaryKeepsAnUndoEntryOutsideTheTree` | `cmd/mrw/keep204_test.go` | the first, second, MCP and dry-run rows through the built binary; nothing in the tree | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §350 |
| 2 — something selects it | `writer.Land`, which every write reaches |
| 3 — the caller can discover it | T2's command and instructions; this task alone is not discoverable, and says so |
| 4 — it is used | T2 |

## Mutation Log

## Verification Log

## Invariants

- A write's receipt, exit code and ledger effect are unchanged but for the not-kept line.
- A failure to keep an entry (a full disk, an unwritable state directory) does not fail the write: the write landed; the receipt says the undo was not kept and why.

## Risks

- Keeping copies makes every write read its originals once more or hold them longer: the size limit bounds it, and a test times nothing.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- The command (deferred: docs/adr/BACKLOG.md)
