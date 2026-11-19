# Task ADR-204-T1: a simple write keeps what it replaced, and mrw undo puts it back

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** unassigned
**Produces:** package `internal/undo`; apply's pre-commit hook and hand-over; the publish in `applyCounted`; `mrw undo` with `--dry-run`, `--forget`, `--json`; the `undo` receipt and the `undo_not_kept` key; contract §350; the instructions, `AGENTS.md`, `README.md`, `docs/agents.md` §7
**Consumes:** `applyCounted` (`internal/writer/writer.go`), `apply.Apply`'s validation read and commit, `seen.Dir`, `seen.Drop`, `seen.Record`, the write lock, `os.Root`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a landed write of edits and creates of regular files publishes one complete entry`, `every other landed write keeps nothing and says which rule`, `the previous entry goes at the pre-commit hook, or the write is refused`, `undo restores exactly or refuses before anything moves`, `an interrupted or failed restore leaves no eligible entry once a file moved`, `a damaged or foreign entry writes nothing`, `the ledger after an undo is replaced, not merged`, `the markers exist`, `the other engine packages are untouched`

## Goal

ADR-204 Decisions 1 to 10, on unix. Rows start from a fresh root.

**Keeping.**

| Write | Afterwards |
|-------|------------|
| a plan editing `a.txt` (0755) and `b.txt`; `mrw write --create d/c.txt`; `write --grep X --sub Y .`; a `search_replace` plan; `mrw_write` over MCP | one entry holding each file's bytes and permission bits before (or that it did not exist) and its sha and bits after |
| a plan with an `unlink`; with a `rename`; an edit through an in-root symlink; an edit of a file with two hard links; an edit of a setuid file | the write lands; nothing is kept; standard error says `mrw: undo: not kept:` and which rule; `--json` and `mrw_write` carry `undo_not_kept` |
| a second landed write | only its entry |
| a dry run; a plan refused at validation; a plan refused after staging (a target made read-only), each after a landed write | the first write's entry, untouched |
| a landing whose commit fails after the first file (apply's own seam, driven from `internal/writer`) | no entry; the not-kept line |
| before-images exactly at the limit; one byte over (the limit lowered through a test seam) | kept; not kept, with the reason |
| the entry cannot be published (the state directory made unwritable after the hook) | the write lands with the exit it had; not kept, with the reason |
| the previous entry cannot be removed (its directory made unwritable before the write) | the write is refused, exit 2, nothing written, naming the path and `undo --forget` |
| a state directory with group access; one inside the checkout | not kept, with the reason |
| a write from `sub/` of a git checkout; one with `--root sub` | the entry is in the state directory of the checkout's top and records the root |
| a build for Windows | `go vet` clean; nothing is kept and the reason says so (asserted on the reason function, since CI's Windows shards run it) |

**Undoing.**

| Then | Outcome |
|------|---------|
| `mrw undo` after the first row's edit | exit 0; both files hold their earlier bytes and bits; the receipt names both and when the write was made; no entry remains |
| after `--create d/c.txt` | `c.txt` is gone, `d` stays |
| `mrw undo` again; and with nothing ever kept | exit 1, `nothing to undo` |
| one of two files changed by another tool, in each order; then repaired and undone again | exit 1, neither changes, the difference named, the entry kept (a second `--dry-run` names the same files); then exit 0 |
| a file's permission bits changed; a created file removed; a file replaced by a symlink to itself elsewhere; a second hard link added; two of these at once | exit 1, nothing changed, each named |
| a restore step made to fail after the first file moved (a seam in `internal/undo`) | exit 2, `PARTIALLY APPLIED` naming restored and unrestored files; no eligible entry |
| a failure before any file moved | exit 2, nothing changed, the entry eligible again |
| the entry damaged by hand, one field at a time: a path with `..`; an absolute path; a `.git` component; a path through a symlinked directory; a duplicate path; another checkout; another root; a mode with setuid; a blob shorter than recorded; a wrong digest; an unknown version; the entry replaced by a symlink | exit 2 and nothing written (exit 1 naming the root for "another root"); the entry left in place |
| a write at the top touching `x/a` then `mrw --root x undo` | exit 1, naming the root to run it from |
| `mrw undo --dry-run` on a restorable and on a conflicting entry | exit 0 and exit 1; nothing changed |
| `mrw undo --forget` with an entry, with an abandoned temporary, with nothing | exit 0, says what it removed |
| `mrw undo --json` for a restore, a refusal, nothing to undo | one document each; the receipt test finds the type and every key in `docs/receipts.txt` |
| after an undo of an edit of a file the caller had read whole: `@@ a.txt 2 replace` by number; then a pattern edit | refused with kind `stale-number`; the pattern edit lands with no read |
| a pending `mrw_read` checkpoint taken before the write, acked after the undo | not promoted |
| a check running while `mrw undo` restores | its receipt counts one other write; `mrw stats` counts no landed write |
| `mrw undo extra`, `mrw undo --check`, `mrw undo --dry-run --forget`; at `MRW_STEP_DEPTH=8` | exit 2, usage; and it runs |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/undo/undo.go`, `restore.go`, `*_test.go` | new | the entry, its validation, the restore |
| `internal/apply/apply.go`, `internal/apply/keep204_test.go` | edit, new | the pre-commit hook; the hand-over of validated bytes, bits and link count |
| `internal/writer/writer.go`, `internal/writer/undo204_test.go` | edit, new | hook and publish inside the write lock |
| `cmd/mrw/undo204.go`, `cmd/mrw/main.go`, `cmd/mrw/skill168.go`, `cmd/mrw/undo204_test.go`, `cmd/mrw/receipts111_test.go` | new, edit | the subcommand; the not-kept line; the receipt registered |
| `internal/mcp/tools.go`, `internal/mcp/schema.go`, `internal/mcp/undo204_test.go` | edit, new | `undo_not_kept` on `mcp_write` |
| `docs/receipts.txt` | edit | the `undo` receipt and the key |
| `internal/guide/guide.go`, `.claude/skills/mrw/SKILL.md`, `internal/guide/undo204_test.go` | edit, new | the instructions (full text only) |
| `scripts/contract.sh` | edit | §350 |
| `AGENTS.md`, `README.md`, `docs/agents.md`, `docs/adr/ADR-127-*.md` | edit | §5; what undo covers beside `/rewind`; restores in `drift_writers` |

## Ordered Steps

1. [S1] The tests, RED. [proof: acceptance]
2. [S2] Keeping. Mutants, each killed: the hook is called before the last pre-commit refusal; the hook is called after the first rename; a stale entry that cannot be removed lets the write land; the entry is published after the lock is released; a partial landing publishes; a write with an `unlink` publishes; a write through a link publishes; a file with two links publishes; the bytes are read again after the write; the permission bits are not kept; an over-limit write keeps a truncated entry; a store with group access is used; the entry is kept under the root's state and not the checkout's; a failed publish fails the write. [proof: mutation]
3. [S3] Validation and verification. Mutants, each killed, one per damaged field of the Goal's row; and: an edited file is not verified; a created file's absence is not noticed; permission bits are not compared; a link in place of the file is followed; a second link is not noticed; one file is restored before the others are verified; a refused undo removes the entry. [proof: mutation]
4. [S4] The restore and the ledger. Mutants, each killed: the entry stays eligible during the restore; a file is not rechecked at its own move; the permission bits are not restored; a created file is left; the entry survives a successful undo; a failure before any move loses the entry; the ledger record is merged and not replaced; pending acknowledgements survive; the undo is counted as a landed write. [proof: mutation]
5. [S5] The command and the receipts. Mutants, each killed: `--dry-run` restores; `--forget` leaves the entry; nothing to undo exits 0; a refusal under `--json` prints no document; the not-kept line is not printed under `--quiet`. [proof: mutation]
6. [S6] Contract §350 drives the built binary: write, undo, the file byte-identical to before, its mode too; paired with the same after another tool changed the file, refused with the tree and the entry untouched; a write with a rename lands and says nothing was kept; a stale entry that cannot be removed refuses the write. The instructions and documents. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/undo ./internal/apply ./internal/writer ./internal/mcp ./cmd/mrw ./internal/guide -run '^(TestAnEntryHoldsExactlyWhatAWriteReplaced|TestATamperedEntryWritesNothing|TestRestoreChecksEveryFileBeforeAnyMoves|TestApplyCallsTheHookAfterItsLastRefusal|TestAWriteClearsAtTheHookAndPublishesWhenComplete|TestMrwWriteSaysWhenNoUndoWasKept|TestUndoPutsBackTheLastSimpleWriteOrRefusesBeforeAnythingMoves|TestNoShippedReceiptFieldDisappears|TestTheInstructionsSayWhatUndoCovers)$' > "${TMPDIR:-/tmp}/adr204-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr204-fence.out")" -eq 9 ] \
  && grep -q '^# 350\. ' scripts/contract.sh \
  && grep -q '^undo ' docs/receipts.txt \
  && grep -q 'mrw undo' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnEntryHoldsExactlyWhatAWriteReplaced` | `internal/undo/undo_test.go` | the entry round-trips bytes and bits exactly (CRLF, no final newline, a NUL in a created file, 0755); publish is all or nothing; the store checks; permissions under a zero umask | none | S1, S2 |
| `TestATamperedEntryWritesNothing` | `internal/undo/restore_test.go` | each damaged field of the Goal's row | none | S1, S3 |
| `TestRestoreChecksEveryFileBeforeAnyMoves` | `internal/undo/restore_test.go` | Decision 6 (c) to (f) and Decision 7, the two seamed failures included, on entries a real landing wrote | none | S1, S3, S4 |
| `TestApplyCallsTheHookAfterItsLastRefusal` | `internal/apply/keep204_test.go` | the hook's place for a clean plan, a plan refused at validation, one refused after staging, and a commit that fails part way; the hand-over | none | S1, S2 |
| `TestAWriteClearsAtTheHookAndPublishesWhenComplete` | `internal/writer/undo204_test.go` | every "Keeping" row through the writer, the order of hook, apply and publish against the lock included | none | S1, S2 |
| `TestMrwWriteSaysWhenNoUndoWasKept` | `internal/mcp/undo204_test.go` | `undo_not_kept` on `mrw_write`, its schema and description | none | S1, S5 |
| `TestUndoPutsBackTheLastSimpleWriteOrRefusesBeforeAnythingMoves` | `cmd/mrw/undo204_test.go` | every "Undoing" row and the CLI "Keeping" rows through the built binary | none | S1, S2, S3, S4, S5 |
| `TestNoShippedReceiptFieldDisappears` | `cmd/mrw/receipts111_test.go` | exists today; gains the `undo` type, which is what goes red | none | S1, S5 |
| `TestTheInstructionsSayWhatUndoCovers` | `internal/guide/undo204_test.go` | the full instructions name `mrw undo`, which writes keep a copy and which do not, what is not put back, how long the copy stays, and the limits of Decisions 3, 4 and 8; the core is unchanged | none | S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the nine tests and §350 |
| 2 — something selects it | `applyCounted` for the keeping; `mrw undo` in the root command |
| 3 — the caller can discover it | `mrw --help`, `mrw instructions`, `AGENTS.md` §5, `docs/agents.md` §7 beside `/rewind`, and the not-kept line on a write |
| 4 — it is used | telemetry is refused by ADR-009; the reading of BACKLOG.md "Pre-registered for ADR-204" |

## Mutation Log

## Verification Log

## Invariants

- A write's exit code, receipt and ledger effect are unchanged, but for the not-kept line and key and the one refusal of Decision 3.
- A refused undo changes nothing: not the tree, not the entry, not the ledger.
- `mrw undo` changes no file that was not in the write it undoes.

## Risks

- The fence's `# 350.`, `^undo ` and `mrw undo` clauses are markers; the named tests and the contract gate are the proof.
- Test seams for the size limit, the store and the restore's failures are unexported and reached from production code paths or live in `_test.go`, or `deadcode` flags them.
- Platform skips are inside the named tests, so the count of nine holds on every platform.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes together with CI's Python suites and `scripts/measure.sh`, and the record is Accepted by Zy.

## Out of Scope

- Undo of a write that unlinks, renames, goes through a link or changes a hard-linked name (deferred: docs/adr/BACKLOG.md)
- Undo on Windows (deferred: docs/adr/BACKLOG.md)
