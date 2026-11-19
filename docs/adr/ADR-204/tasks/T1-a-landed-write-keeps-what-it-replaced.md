# Task ADR-204-T1: a landed write keeps what it replaced

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** package `internal/undo` (the entry: `Clear`, `Publish`, `Load`); the entry format; the receipt line and key when nothing was kept; the refusal when a stale entry cannot be removed; contract §350
**Consumes:** `applyCounted`'s critical section (`internal/writer/writer.go`), apply's validation read and the kinds of the entries it changes, the checkout's state directory
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a landed write publishes one complete entry of what it replaced`, `the previous entry is removed before the tree is touched, or the write is refused`, `a write over the size limit, a partial landing, an unverified store and a failed publish keep nothing and say so`, `a refused plan and a dry run remove nothing`, `the other engine packages are untouched`

## Goal

ADR-204 Decisions 2 to 4. Each row starts from a fresh root; "the entry" is the one kept entry of the checkout:

| Call | The entry afterwards |
|------|----------------------|
| a plan editing `a.txt` (mode 0755) and `b.txt` | both paths, each a regular file with its bytes and permission bits before, and the sha it was left with |
| `mrw write --create d/c.txt` | `d/c.txt` absent before, a regular file after; `d` among the created directories |
| a plan `@@ a.txt - unlink` | `a.txt` a regular file before with its bytes, absent after |
| a plan that unlinks `c` and renames `b` onto `c` with an edit | one state per entry: `b` before and absent after, `c` before and what it holds after |
| an edit through an in-root symlink | the file the link led to; an unlink of the link itself records the link's text |
| `mrw write --grep X --sub Y .`, a `search_replace` plan, `mrw_write` over MCP, `mrw skill --dir D` | one entry, as a native plan's (for the last, in the state of `D`'s checkout) |
| a second landed write | only the second write's entry |
| a dry run, a refused plan, after a landed write | the first write's entry, untouched |
| before-images exactly at the limit; one byte over; several files over in total (the limit lowered through a test seam) | kept; nothing kept and the receipt says `undo: not kept` with the reason; the same |
| a partial landing (a commit step made to fail through apply's own seam, tested in `internal/apply`) | nothing kept |
| the entry cannot be published (the store made unwritable after the clear) | the write lands with the exit it had, and the receipt says nothing was kept and why |
| the previous entry cannot be removed | the write is refused, exit 2, nothing written, the refusal naming `mrw undo --forget` |
| a state directory that is group-writable, or inside the checkout | nothing kept, the receipt says why |
| a write from `sub/` of a checkout, and one with `--root sub` | the entry is in the state directory of the checkout's top, where the ledger is, and records the root, with paths relative to it |

Bytes round-trip exactly: CRLF, no final newline, a NUL, 0755. The entry's files are 0600 in a 0700 directory. Nothing appears in the tree.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/undo/undo.go`, `internal/undo/undo_test.go` | new | the entry |
| `internal/writer/writer.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go` | edit | clear, apply, publish inside the write lock; the not-kept line and key |
| `internal/apply/apply.go`, `internal/apply/pathop.go` | edit | hand over the validation read and the entry kinds |
| `internal/apply/keep204_test.go`, `internal/writer/undo204_test.go`, `cmd/mrw/keep204_test.go` | new | the partial landing; the table through the writer; through the binary |
| `scripts/contract.sh` | edit | §350 |
| `docs/receipts.txt` | edit | the not-kept key |

## Ordered Steps

1. [S1] The tests, RED. [proof: acceptance]
2. [S2] The entry and the critical section. Mutants, each killed: the previous entry is removed after the apply; a stale entry that cannot be removed lets the write land; the before-image is read again after the write; the permission bits are not kept; a removed file's bytes are not kept; a dry run clears the entry; a refused plan clears the entry; an over-limit write keeps a truncated entry; a partial landing publishes; the entry is published before it is complete; a group-writable store is used; the entry is kept in the state directory of the root and not of the checkout's top; the entry is published after the write lock is released; a failed publish fails the write; the files are created 0644 (asserted under a zero umask). [proof: mutation]
3. [S3] Contract §350 drives the built binary: a landed edit publishes an entry outside the tree whose before-image is the file's earlier bytes; paired with a dry run leaving the earlier entry as it was. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/undo ./internal/apply ./internal/writer ./cmd/mrw -run '^(TestAnEntryHoldsExactlyWhatAWriteReplaced|TestAPartialLandingHandsOverNothingToKeep|TestAWriteClearsThenAppliesThenPublishes|TestTheBinaryKeepsOneUndoEntryOutsideTheTree)$' > "${TMPDIR:-/tmp}/adr204t1-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr204t1-fence.out")" -eq 4 ] \
  && grep -q '^# 350\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnEntryHoldsExactlyWhatAWriteReplaced` | `internal/undo/undo_test.go` | the entry round-trips bytes, kinds, modes and link text exactly; publish is all or nothing; permissions under a zero umask; the store checks | none | S1, S2 |
| `TestAPartialLandingHandsOverNothingToKeep` | `internal/apply/keep204_test.go` | apply's hand-over for a clean landing and for a commit that fails part way | none | S1, S2 |
| `TestAWriteClearsThenAppliesThenPublishes` | `internal/writer/undo204_test.go` | every row of the Goal's table through the writer, the order of the three steps included | none | S1, S2 |
| `TestTheBinaryKeepsOneUndoEntryOutsideTheTree` | `cmd/mrw/keep204_test.go` | the first, second, MCP, second-write, dry-run and stale-entry rows through the built binary | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §350 |
| 2 — something selects it | `applyCounted`, which every write reaches |
| 3 — the caller can discover it | T2's command and instructions; this task alone is not discoverable and is not released without T2 |
| 4 — it is used | T2 |

## Mutation Log

## Verification Log

## Invariants

- A write's exit code and ledger effect are unchanged, but for the one refusal of Decision 2.
- A failure to publish does not fail the write.

## Risks

- The fence's `# 350.` clause is a marker: §350 is proved by running `scripts/contract.sh`, a gate.
- This task merges only together with T2, in one pull request.
- A test seam for the size limit and for store failures must live in `_test.go` or be unexported and unreachable from production paths, or `deadcode` flags it.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- The command (deferred: ADR-204 T2)
