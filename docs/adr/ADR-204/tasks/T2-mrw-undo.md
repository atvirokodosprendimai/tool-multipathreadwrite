# Task ADR-204-T2: mrw undo

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw undo` with `--dry-run`, `--list`, `--forget`, `--json`; `undo.Restore`; the `undo` receipt in `docs/receipts.txt`; contract §351; the instructions, `AGENTS.md`, `README.md`, `docs/agents.md` §7
**Consumes:** T1's entries; apply's staged commit and ADR-066's reporting; the write lock; the ledger
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the newest kept write is restored exactly`, `nothing moves unless every file is as the write left it`, `a second undo walks back and a limit ends the walk`, `the restored files are recorded as written`, `the flags and the receipt do what Decision 8 says`

## Goal

ADR-204 Decisions 1 to 3 and 6 to 9. Each row starts from a fresh root and a landed write made by this binary:

| Then | Outcome |
|------|---------|
| `mrw undo` after an edit of `a.txt` and `b.txt` | exit 0; both files hold their earlier bytes and mode; the receipt names both |
| after `--create c.txt` in a new directory | `c.txt` and the directory the write made are gone |
| after `@@ a.txt - unlink` | `a.txt` is back, bytes and mode |
| after a rename with an edit | the source is back as it was; the destination is gone |
| after `write --grep X --sub Y .` over three files | all three restored |
| `mrw undo` twice after two writes | the second undoes the earlier write |
| `mrw undo` with nothing kept | exit 1, `mrw: nothing to undo`, nothing changed |
| one of the two files edited by another tool after the write | exit 1; neither file changes; the differing file named with the sha expected and found |
| a created file removed by hand; an unlinked path recreated; a rename's source recreated | exit 1, nothing changed, the path named |
| `mrw undo --dry-run` | exit 0, lists what it would restore, changes nothing, keeps the entry |
| `mrw undo --list` | newest first; `--forget` then empties it and `mrw undo` says nothing to undo |
| `mrw undo --json` | one document with the keys listed in `docs/receipts.txt` |
| after an undo, `@@ a.txt 2 replace` by number | refused by ADR-203: the restored file is one mrw wrote |
| a write that kept no entry (over the limit) between two that did | the walk back stops at it, exit 1, saying why |
| a file of the write held open without delete sharing (Windows) | exit 1, nothing changed, as a write refuses it |
| `mrw undo extra`, `mrw undo --list --forget` | exit 2, usage |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/undo/restore.go`, `internal/undo/restore_test.go` | new | verification and restore |
| `cmd/mrw/undo204.go`, `cmd/mrw/main.go` | new, edit | the subcommand |
| `cmd/mrw/undo204_test.go` | new | the Goal's table through the built binary |
| `docs/receipts.txt` | edit | the `undo` receipt |
| `internal/guide/guide.go`, `.claude/skills/mrw/SKILL.md`, `internal/guide/undo204_test.go` | edit, new | the instructions |
| `scripts/contract.sh` | edit | §351 |
| `AGENTS.md`, `README.md`, `docs/agents.md` | edit | §5 "The other subcommands"; what undo covers beside `/rewind` |

## Ordered Steps

1. [S1] The tests, RED. [proof: acceptance]
2. [S2] Verification and restore. Mutants, each killed: an edited file is not verified; a created file's absence is not noticed; an unlinked path that exists again is overwritten; the rename's source is not checked; one file is restored before the other is verified; the mode is not restored; the entry survives a successful undo; the entry is removed by a refused undo; the restored files are not recorded in the ledger; a not-kept write is skipped over. [proof: mutation]
3. [S3] The flags and the receipt. Mutants, each killed: `--dry-run` restores; `--list` is oldest first; `--forget` leaves an entry; nothing to undo exits 0. [proof: mutation]
4. [S4] Contract §351 drives the built binary: write, undo, the file byte-identical to before; paired with the same after another tool changed the file, refused with the tree untouched. The instructions and documents. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/undo ./cmd/mrw ./internal/guide -run '^(TestRestoreVerifiesEveryFileBeforeAnyMoves|TestUndoRestoresTheLastWriteWholeOrNotAtAll|TestTheInstructionsSayWhatUndoCovers)$' > "${TMPDIR:-/tmp}/adr204t2-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr204t2-fence.out")" -eq 3 ] \
  && grep -q '^# 351\. ' scripts/contract.sh \
  && grep -q '^undo ' docs/receipts.txt \
  && grep -q 'mrw undo' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestRestoreVerifiesEveryFileBeforeAnyMoves` | `internal/undo/restore_test.go` | each arm of Decision 3 and of Decision 6, on entries a real landing wrote | none | S1, S2 |
| `TestUndoRestoresTheLastWriteWholeOrNotAtAll` | `cmd/mrw/undo204_test.go` | every row of the Goal's table through the built binary, with exit codes and bytes | none | S1, S2, S3 |
| `TestTheInstructionsSayWhatUndoCovers` | `internal/guide/undo204_test.go` | the full instructions name `mrw undo`, that it covers mrw's own writes only, and that copies are kept until 16 newer writes or `--forget`; the core is unchanged | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §351 |
| 2 — something selects it | `mrw undo` in the root command |
| 3 — the caller can discover it | `mrw --help`, `mrw instructions`, `AGENTS.md` §5, `docs/agents.md` §7 beside `/rewind` |
| 4 — it is used | telemetry is refused by ADR-009; the reading of BACKLOG.md "Pre-registered for ADR-204" |

## Mutation Log

## Verification Log

## Invariants

- `mrw undo` changes no file that was not in the write it undoes.
- A refused undo changes nothing: not the tree, not the store, not the ledger.

## Risks

- The Windows row needs a held file; where the test cannot hold one it is skipped by name, not silently.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- A redo (deferred: docs/adr/BACKLOG.md)
- An undo over MCP (deferred: docs/adr/BACKLOG.md)
