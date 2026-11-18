# Task ADR-201-T1: the seven state reads open without blocking

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** unassigned
**Produces:** `regular.ReadFile`, `regular.OpenIn`; non-blocking reads in `internal/state/state.go`, `internal/state/prune.go`, `internal/state/lock.go`, `internal/writer/writes.go`, `internal/mcp/ack.go`, `internal/check/check.go`, `cmd/mrw/run177.go`; a lock error that carries its path; contract §344
**Consumes:** `regular.Open` (ADR-109); `internal/apply/tree.go`'s open through an `*os.Root` and `internal/authoring`'s opener variable (#663) as patterns
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a FIFO at a state file returns at once with its decided outcome`, `a counter that could not be read is not overwritten`, `a lock error names its file`, `a regular file reads as before on every platform`, `the built binary is not held by a FIFO at a marker`, `the engine packages this record does not own are untouched`

## Goal

ADR-201 Decisions 1 to 5. With a FIFO where each file is expected, the function that reads it returns within its test's deadline and does this:

| Site | With a FIFO there |
|------|-------------------|
| `state.Dir`, the `root` marker | returns the directory; the marker is then a regular file holding the root |
| `state` `describe` (through `Entries` and `Prune`), another directory's `root` | the entry is not identified and is kept; a dry run and a real prune both return |
| `state` `holderOf`, a lock's `.holder`, read after the wait ran out | `""`; `HoldFileWithin` with a zero wait returns its timeout error with the "another process" wording |
| `writer.Writes`, the write counter | zero; `bumpWrites` then leaves a regular file holding 1 |
| `mcp` `loadPending` | an empty store and no error; `ForgetPending` returns no error; a later save leaves a regular file |
| `check` `lastLines` | no lines, zero earlier, not cut |
| `cmd/mrw` `scanLog` | zero lines, size 0, no last line |

And:
- **a counter that could not be read is not overwritten.** With the opener answering `EWOULDBLOCK`, and with the file made unreadable by mode where the test's user is bound by modes, `bumpWrites` returns zero and a counter holding 7 still holds 7; once it reads again the next bump is 8. A counter holding text that does not parse is still read as zero and bumped to 1.
- **the pending store keeps returning an error for a read that failed otherwise** (the opener answering `EWOULDBLOCK`), and the store on disk is untouched.
- **a lock error names its file.** Taking a lock whose file is a FIFO either takes it (and a second taker with a zero wait is refused, so it is a lock) or returns an error that is an `*fs.PathError` naming the lock file. The test says which the platform gave.
- **a regular file reads as before on every platform**: `regular.ReadFile` returns a regular file's bytes, `fs.ErrNotExist` for a missing one, and an error for a directory; none of these tests is unix-only.

`regular.ReadFile` and `regular.OpenIn` return path-bearing errors (`*fs.PathError`), so `statehint.About` still recognises a state failure.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/regular/readfile201_test.go` | new | the red test for the helper, every platform |
| `internal/regular/readfile201_unix_test.go` | new | the red test for a FIFO |
| `internal/regular/regular.go` | edit | `ReadFile`, `OpenIn` |
| `internal/state/fifo201_unix_test.go` | new | the red tests for the marker, the holder, the lock and the prune read |
| `internal/state/state.go`, `internal/state/prune.go`, `internal/state/lock.go` | edit | read without blocking; the lock error's path |
| `internal/writer/writes201_test.go`, `internal/writer/writes201_unix_test.go` | new | the red tests for the counter |
| `internal/writer/writes.go` | edit | read without blocking; no save over a failed read |
| `internal/mcp/pending201_unix_test.go` | new | the red test for the pending store |
| `internal/mcp/ack.go` | edit | read without blocking; a non-regular store is empty |
| `internal/check/lastlines201_unix_test.go` | new | the red test for the check log |
| `internal/check/check.go` | edit | open without blocking |
| `cmd/mrw/scanlog201_unix_test.go` | new | the red test for the run log |
| `cmd/mrw/run177.go` | edit | open without blocking |
| `scripts/contract.sh` | edit | §344 |
| `docs/adr/ADR-109-a-loader-opens-only-a-regular-file.md` | edit | a `Superseded in part` pointer to this record |

## Ordered Steps

1. [S1] Write the nine tests and see them RED: the helper's tests do not compile, each FIFO test reaches its deadline on the blocking read, the counter and lock tests fail on today's outcomes. Every FIFO test bounds its call with a deadline and releases a blocked reader in its cleanup by opening the FIFO's write end without blocking. [proof: acceptance]
2. [S2] `regular.ReadFile` and `regular.OpenIn`, and the seven sites read through them or `regular.Open`. Mutants, each killed by its site's test: the marker is read with `os.ReadFile`; the prune read opens blocking; the holder; the counter; the pending store; `lastLines` opens with `os.Open`; `scanLog`; a non-regular pending store is an error. [proof: mutation]
3. [S3] `bumpWrites` writes nothing after a read that failed otherwise. Mutants, each killed: it writes 1 over a busy counter; a counter that does not parse is treated as a failed read. [proof: mutation]
4. [S4] `Hold` and `HoldFileWithin` wrap an open or lock error with the lock's path. Mutant, killed: the bare error is returned. [proof: mutation]
5. [S5] Contract §344 drives the built binary, each command through the script's `bounded` helper: in a scratch root with its own state home, `mrw read f.txt` is served; the state directory is taken from `mrw seen`; with a FIFO at its `root`, `mrw read f.txt` exits 0, serves the file, and the marker is then a regular file holding the root; with a FIFO at the `root` of a second directory under the same state home, `mrw seen --prune --dry-run` exits 0 and removes nothing. On the binary before this task both bounded commands are killed at the limit. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/regular ./internal/state ./internal/writer ./internal/mcp ./internal/check ./cmd/mrw -run '^(TestReadFileReadsARegularFileAndNamesWhatItCannot|TestReadFileRefusesANonRegularFileAtOnce|TestAFifoAtAStateFileDoesNotHoldMrw|TestAFifoMarkerInAnotherEntryDoesNotHoldPrune|TestAFifoAtTheWriteCounterReadsAsZero|TestAWriteCounterThatCouldNotBeReadIsNotOverwritten|TestAFifoAtThePendingStoreIsAnEmptyStore|TestAFifoAtTheCheckLogReadsAsNoLines|TestAFifoAtARunLogReadsAsEmpty)$' > "${TMPDIR:-/tmp}/adr201-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr201-fence.out")" -eq 9 ] \
  && grep -q '^# 344\. ' scripts/contract.sh \
  && grep -q 'func ReadFile' internal/regular/regular.go \
  && grep -q 'func OpenIn' internal/regular/regular.go \
  && [ "$(cat internal/state/state.go internal/state/lock.go internal/writer/writes.go internal/mcp/ack.go internal/check/check.go cmd/mrw/run177.go internal/authoring/authoring.go | grep -c 'os\.ReadFile(\|os\.Open(')" -eq 0 ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestReadFileReadsARegularFileAndNamesWhatItCannot` | `internal/regular/readfile201_test.go` | a regular file's bytes; `fs.ErrNotExist` for a missing one; an error for a directory; each error an `*fs.PathError` naming the path; `OpenIn` the same through an `*os.Root`. Runs on every platform | none | S1, S2 |
| `TestReadFileRefusesANonRegularFileAtOnce` | `internal/regular/readfile201_unix_test.go` | a FIFO is `ErrNotRegular` within the deadline, by `ReadFile` and by `OpenIn` | none | S1, S2 |
| `TestAFifoAtAStateFileDoesNotHoldMrw` | `internal/state/fifo201_unix_test.go` | the marker, the holder after a zero wait, and the lock file, as the Goal's table and bullets say | none | S1, S2, S4 |
| `TestAFifoMarkerInAnotherEntryDoesNotHoldPrune` | `internal/state/fifo201_unix_test.go` | `Entries` and `Prune`, dry run and real, return; the entry is unidentified and kept | none | S1, S2 |
| `TestAFifoAtTheWriteCounterReadsAsZero` | `internal/writer/writes201_unix_test.go` | `Writes` is zero within the deadline; `bumpWrites` leaves a regular file holding 1 | none | S1, S2 |
| `TestAWriteCounterThatCouldNotBeReadIsNotOverwritten` | `internal/writer/writes201_test.go` | 7 stays 7 under a busy opener and, where modes bind, an unreadable file; then 8; text that does not parse is bumped to 1 | none | S1, S3 |
| `TestAFifoAtThePendingStoreIsAnEmptyStore` | `internal/mcp/pending201_unix_test.go` | `loadPending` and `ForgetPending` as the table says; a busy opener is still an error and leaves the file | none | S1, S2 |
| `TestAFifoAtTheCheckLogReadsAsNoLines` | `internal/check/lastlines201_unix_test.go` | `lastLines` returns nothing within the deadline; a regular log reads as before | none | S1, S2 |
| `TestAFifoAtARunLogReadsAsEmpty` | `cmd/mrw/scanlog201_unix_test.go` | `scanLog` returns zeros within the deadline; a regular log reads as before | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the nine tests and §344 |
| 2 — something selects it | `state.Dir` is on every command's path; the others are on a prune, a refused lock, a landing, an MCP call or `--forget`, a check and a run |
| 3 — the caller can discover it | nothing to discover: a command that hung now answers; a lock error names its file |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is three hangs reproduced on v1.89.0 and the four files #662 and #663 already fixed |

## Mutation Log

## Verification Log

## Invariants

- No receipt key or exit code is added, and a state file that is a regular file is read exactly as before.
- `internal/seen`, `internal/plan`, `internal/read` and `internal/apply` are byte-identical to main.
- Nothing opens a file mrw keeps blocking.

## Risks

- The busy case is driven through a variable on the opener, as #663 did; a real lease is not taken in a test.
- The FIFO tests are unix-only: Windows has no FIFO in a directory. The controls for a regular file, a missing file and a directory run on Windows.
- `flock` on a FIFO differs by platform; the lock case accepts either outcome the Goal names and reports which.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- The measurement tools' reads (permanent: boundary: they are not the served binary and read files their own run wrote)
- Waiting on a lock another process holds (permanent: boundary: ADR-110 bounds the write lock's wait; a wait is not a content read)
