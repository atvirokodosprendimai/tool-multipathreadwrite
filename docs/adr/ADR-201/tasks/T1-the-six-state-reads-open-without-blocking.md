# Task ADR-201-T1: the six state reads open without blocking

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `regular.ReadFile`; non-blocking reads in `internal/state/state.go`, `internal/state/lock.go`, `internal/writer/writes.go`, `internal/mcp/ack.go`, `internal/check/check.go`, `cmd/mrw/run177.go`; contract §344
**Consumes:** `regular.Open` (ADR-109); the busy handling of `internal/authoring` (#663) as a pattern
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a FIFO at a state file returns at once`, `the write counter is not saved over a busy file`, `the built binary is not held by a FIFO at the marker`, `the engine packages this record does not own are untouched`

## Goal

ADR-201 Decisions 1 to 3. With a FIFO at each of the six files, the function that reads it returns within its test's deadline, and does what it does for a file it cannot read:

| Site | With a FIFO there |
|------|-------------------|
| `state.Dir`, the `root` marker | returns the directory; the marker is replaced by a regular file holding the root |
| `state` `holderOf`, a lock's `.holder` | the holder is unknown (`""`) |
| `writer.Writes`, the write counter | zero; `bumpWrites` then writes 1 over it, as for a missing counter |
| `mcp` `loadPending` | an error that is `regular.ErrNotRegular`; nothing is licensed |
| `check` `lastLines` | no lines |
| `cmd/mrw` `scanLog` | zero lines, size 0, no last line |

And for the write counter held busy (the opener answers `EWOULDBLOCK` until the retry gives up): `bumpWrites` returns zero and the file is left as it was.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/regular/readfile201_unix_test.go` | new | the red test for the helper |
| `internal/regular/regular.go` | edit | `ReadFile` |
| `internal/state/fifo201_unix_test.go` | new | the red test for the marker and the holder |
| `internal/state/state.go`, `internal/state/lock.go` | edit | read without blocking |
| `internal/writer/writes201_unix_test.go` | new | the red tests for the counter: a FIFO, and a busy file |
| `internal/writer/writes.go` | edit | read without blocking; no save over a busy file |
| `internal/mcp/pending201_unix_test.go` | new | the red test for the pending store |
| `internal/mcp/ack.go` | edit | read without blocking |
| `internal/check/lastlines201_unix_test.go` | new | the red test for the check log |
| `internal/check/check.go` | edit | open without blocking |
| `cmd/mrw/scanlog201_unix_test.go` | new | the red test for the run log |
| `cmd/mrw/run177.go` | edit | open without blocking |
| `scripts/contract.sh` | edit | §344 |

## Ordered Steps

1. [S1] Write the six tests and see them RED: each times out at its deadline on the blocking read, and the helper's does not compile. [proof: acceptance]
2. [S2] `regular.ReadFile`, and the six sites read through `regular.Open` or it. Mutants, each killed by its site's test: the marker is read with `os.ReadFile`; the holder; the counter; the pending store; `lastLines` opens with `os.Open`; `scanLog`. [proof: mutation]
3. [S3] `bumpWrites` does not write when the read was busy. Mutant, killed: it writes 1 over a busy counter. [proof: mutation]
4. [S4] Contract §344 drives the built binary: with a FIFO at the `root` marker of a scratch root's state, `mrw read f.txt` under a ten-second limit exits 0 and serves the file; paired with the same read before the FIFO is made. The row finds the state directory with `mrw seen`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
[ "$(go test -count=1 -v ./internal/regular ./internal/state ./internal/writer ./internal/mcp ./internal/check ./cmd/mrw -run '^(TestReadFileRefusesANonRegularFileAtOnce|TestAFifoAtAStateFileDoesNotHoldMrw|TestAFifoAtTheWriteCounterReadsAsZero|TestABusyWriteCounterIsNotOverwritten|TestAFifoAtThePendingStoreIsRefusedAtOnce|TestAFifoAtTheCheckLogReadsAsNoLines|TestAFifoAtARunLogReadsAsEmpty)$' | grep -c '^--- PASS: ')" -eq 7 ] \
  && grep -q '^# 344\. ' scripts/contract.sh \
  && grep -q 'func ReadFile' internal/regular/regular.go \
  && ! grep -n 'os\.ReadFile(\|os\.Open(' internal/state/state.go internal/state/lock.go internal/writer/writes.go internal/mcp/ack.go cmd/mrw/run177.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestReadFileRefusesANonRegularFileAtOnce` | `internal/regular/readfile201_unix_test.go` | a regular file is read whole; a FIFO is `ErrNotRegular` within the deadline; a missing file is `fs.ErrNotExist` | none | S1, S2 |
| `TestAFifoAtAStateFileDoesNotHoldMrw` | `internal/state/fifo201_unix_test.go` | `Dir` with a FIFO at the marker returns and leaves a regular marker holding the root; `holderOf` with a FIFO is `""`; taking a lock whose file is a FIFO returns | none | S1, S2 |
| `TestAFifoAtTheWriteCounterReadsAsZero` | `internal/writer/writes201_unix_test.go` | `Writes` returns zero within the deadline and `bumpWrites` then leaves a regular file holding 1 | none | S1, S2 |
| `TestABusyWriteCounterIsNotOverwritten` | `internal/writer/writes201_unix_test.go` | with the opener answering `EWOULDBLOCK`, `bumpWrites` returns zero and a counter holding 7 still holds 7; with the opener restored the next bump is 8 | none | S1, S3 |
| `TestAFifoAtThePendingStoreIsRefusedAtOnce` | `internal/mcp/pending201_unix_test.go` | `loadPending` returns an error that is `regular.ErrNotRegular` within the deadline | none | S1, S2 |
| `TestAFifoAtTheCheckLogReadsAsNoLines` | `internal/check/lastlines201_unix_test.go` | `lastLines` returns no lines within the deadline | none | S1, S2 |
| `TestAFifoAtARunLogReadsAsEmpty` | `cmd/mrw/scanlog201_unix_test.go` | `scanLog` returns zeros within the deadline | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the seven tests and §344 |
| 2 — something selects it | `state.Dir` is on every command's path; the other five are on a landing, an MCP call, a check and a run |
| 3 — the caller can discover it | nothing to discover: a command that hung now answers |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is the hang reproduced on v1.89.0 and the four files #663 already fixed |

## Mutation Log

## Verification Log

## Invariants

- No receipt key, exit code or refusal text is added or changed for a state file that is a regular file.
- `internal/seen`, `internal/plan`, `internal/read` and `internal/apply` are byte-identical to main.
- Nothing opens a state file blocking.

## Risks

- The busy case is driven through a variable on the opener, as #663 did; a real lease is not taken in a test.
- The tests are unix-only: Windows has no FIFO in a directory. The Windows jobs run every existing state test over the changed reads.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- The measurement tools' reads (permanent: boundary: they are not the served binary and read files their own run wrote)
