# Task ADR-204-T2: mrw undo

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw undo` with `--dry-run`, `--forget`, `--json`; `undo.Restore`; the `undo` receipt in `docs/receipts.txt`; contract §351; the instructions, `AGENTS.md`, `README.md`, `docs/agents.md` §7
**Consumes:** T1's entry; apply's staged commit and ADR-066's reporting; the write lock; the ledger; `rooted.Resolve`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the contract section, the receipt type and the AGENTS.md mention exist`, `the last write is put back exactly as far as Decision 7 goes`, `nothing moves unless every directory entry is as the write left it`, `a failure during the restore is reported as a commit failure is`, `a damaged or foreign entry writes nothing`, `the ledger after an undo is what Decision 9 says`, `the flags, exit codes and receipt are what Decision 10 says`

## Goal

ADR-204 Decisions 1 and 5 to 10. Each row starts from a fresh root and a landed write made by this binary:

| Then | Outcome |
|------|---------|
| `mrw undo` after an edit of `a.txt` (0755) and `b.txt` | exit 0; both hold their earlier bytes and permission bits; the receipt names both; the entry is gone |
| after `--create d/c.txt` | `d/c.txt` and `d` are gone; with another file put in `d` meanwhile, `c.txt` goes and `d` stays |
| after `@@ a.txt - unlink` | `a.txt` is back |
| after a plan that unlinks `c` and renames `b` onto `c` with an edit | `b` and `c` are as before |
| after a case-only rename on a folding filesystem (skipped by name elsewhere) | the earlier spelling is back |
| after an edit through an in-root symlink | the file the link led to is restored; the link is untouched |
| after `write --grep X --sub Y .`, a `search_replace` plan and an `mrw_write` landing | each restored |
| `mrw undo` a second time; and with nothing ever kept | exit 1, `nothing to undo`, nothing changed |
| one of two files changed by another tool after the write; then repaired and `mrw undo` again | exit 1, neither file changes, the differing file named with what was expected and found, the entry kept; then exit 0 |
| a created file changed; removed; an unlinked path recreated; a rename's source recreated; a file's permission bits changed; two of these at once | exit 1, nothing changed, every differing entry named |
| a write that changed one name of a hard-linked file | exit 1, the sentence saying a hard link cannot be put back |
| a restore step made to fail after the first file moved (apply's seam, in `internal/undo`) | exit 2, `PARTIALLY APPLIED` naming what was and was not restored; the entry removed |
| the entry edited by hand: a path with `..`, a path under `.git`, an absolute path, another checkout's name, a mode with setuid, a blob shorter than recorded, an unknown version | exit 2 (exit 1 for a path outside the invocation root), nothing written, the entry left in place |
| a write made at the checkout top touching `x/` and `y/`, then `mrw --root x undo` | exit 1, naming the root to run it from; nothing changed |
| `mrw undo --dry-run` on a restorable and on a conflicting entry | exit 0 and exit 1 with the same lines; nothing changed; the entry kept |
| `mrw undo --forget` with and without an entry | exit 0, says what it removed; `mrw undo` then says nothing to undo |
| `mrw undo --json` for a restore, a refusal and nothing to undo | one document each, with the keys in `docs/receipts.txt`; the restore names each file and when the write was made |
| a check running while `mrw undo` restores | the check's receipt counts one other write (ADR-127); `mrw stats` counts no landed write for the undo |
| after an undo of an edit, `@@ a.txt 2 replace` by number; after an undo of an unlink, a pattern edit of `a.txt` | refused by ADR-203; refused as not read, and landing after `mrw read a.txt` |
| a file of the write held without delete sharing (Windows; skipped by name elsewhere) | exit 1, nothing changed |
| `mrw undo extra`, `mrw undo --check`, `mrw undo --dry-run --forget`; `mrw undo` at `MRW_STEP_DEPTH=8` | exit 2, usage; and it runs |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/undo/restore.go`, `internal/undo/restore_test.go` | new | validation of the entry, verification, restore |
| `cmd/mrw/undo204.go`, `cmd/mrw/main.go` | new, edit | the subcommand |
| `cmd/mrw/undo204_test.go`, `cmd/mrw/receipts111_test.go` | new, edit | the Goal's table through the built binary; the receipt registered |
| `docs/receipts.txt` | edit | the `undo` receipt |
| `internal/guide/guide.go`, `.claude/skills/mrw/SKILL.md`, `internal/guide/undo204_test.go` | edit, new | the instructions |
| `scripts/contract.sh` | edit | §351 |
| `AGENTS.md`, `README.md`, `docs/agents.md` | edit | §5 "The other subcommands"; what undo covers beside `/rewind` |

## Ordered Steps

1. [S1] The tests, RED. [proof: acceptance]
2. [S2] Validation of the entry and verification. Mutants, each killed: a path is not resolved against the root; a `.git` path is accepted; another checkout's entry is accepted; a blob's digest is not checked; an edited file is not verified; a created file's absence is not noticed; an unlinked path that exists again is overwritten; the rename's source is not checked; permission bits are not compared; one entry is restored before the others are verified; a refused undo removes the entry (asserted by a second `mrw undo --dry-run` naming the same files); the tampered file is checked in both orders of two. [proof: mutation]
3. [S3] The restore and the ledger. Mutants, each killed: the permission bits are not restored; a directory holding another file is removed; the entry survives a successful undo; a partial restore keeps the entry; a restored file is recorded as shown (asserted by the refusal's kind, `stale-number`, and by a pattern edit landing with no read); a file back from an unlink is recorded as written; the hard-link refusal is not made. [proof: mutation]
4. [S4] The flags and the receipt. Mutants, each killed: `--dry-run` restores; `--forget` leaves the entry; nothing to undo exits 0; a refusal under `--json` prints no document. [proof: mutation]
5. [S5] Contract §351 drives the built binary: write, undo, the file byte-identical to before; paired with the same after another tool changed the file, refused with the tree and the entry untouched. The instructions and documents. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/undo ./cmd/mrw ./internal/guide -run '^(TestRestoreChecksEveryEntryBeforeAnyMoves|TestATamperedEntryWritesNothing|TestUndoPutsBackTheLastWriteOrRefusesBeforeAnythingMoves|TestTheInstructionsSayWhatUndoCovers)$' > "${TMPDIR:-/tmp}/adr204t2-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr204t2-fence.out")" -eq 4 ] \
  && grep -q '^# 351\. ' scripts/contract.sh \
  && grep -q '^undo ' docs/receipts.txt \
  && grep -q 'mrw undo' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestRestoreChecksEveryEntryBeforeAnyMoves` | `internal/undo/restore_test.go` | each arm of Decisions 5 to 7 on entries a real landing wrote, the failure after the first move included | none | S1, S2, S3 |
| `TestATamperedEntryWritesNothing` | `internal/undo/restore_test.go` | each field of Decision 8, damaged by hand | none | S1, S2 |
| `TestUndoPutsBackTheLastWriteOrRefusesBeforeAnythingMoves` | `cmd/mrw/undo204_test.go` | every row of the Goal's table through the built binary, with exit codes, bytes, modes and the ledger | none | S1, S2, S3, S4 |
| `TestTheInstructionsSayWhatUndoCovers` | `internal/guide/undo204_test.go` | the full instructions name `mrw undo`, that it covers the last mrw write only, what it does not restore, that the copy is kept until the next write or `--forget`, and the limit of the guarantee; the core is unchanged | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §351 |
| 2 — something selects it | `mrw undo` in the root command |
| 3 — the caller can discover it | `mrw --help`, `mrw instructions`, `AGENTS.md` §5, `docs/agents.md` §7 beside `/rewind`, and the write receipt's line when nothing was kept |
| 4 — it is used | telemetry is refused by ADR-009; the reading of BACKLOG.md "Pre-registered for ADR-204" |

## Mutation Log

## Verification Log

## Invariants

- `mrw undo` changes no directory entry that was not in the write it undoes.
- A refused undo changes nothing: not the tree, not the entry, not the ledger.

## Risks

- The fence's `# 351.`, `^undo ` and `mrw undo` clauses are markers; the gates and the named tests are the proof.
- This task merges together with T1, in one pull request.
- A process that changes a file between the last check and its rename is not caught; the instructions say so.
- The Windows and case-folding rows run only where the filesystem allows and are skipped by name elsewhere.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- More than one kept write, and a redo (deferred: docs/adr/BACKLOG.md)
- An undo over MCP (deferred: docs/adr/BACKLOG.md)
