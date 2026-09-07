# Task ADR-031-T1: Checkpoints and the pending store

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (a marker format, a store outside the tree, and the interleaving)
**Owner:** Zy
**Produces:** the checkpoint format and the pending store (T2, T3)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a checkpoint covering exactly the span it follows`, `a pending record reaching no ledger until it is promoted`

## Goal

Put unguessable checkpoints through a served page, and hold each one's span outside the ledger until
somebody proves they received it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/ack.go` | create | New: the marker format, the interleave over already-served text, and the pending store. It is its own file because `tools.go` is already the largest file in the package and this is a distinct mechanism, not a variation on paging. |
| `internal/mcp/ack_test.go` | create | `TestOnlyAckedSegmentsAreRecorded` and the interleave's own tests. |

## Ordered Steps

1. [S1] Write `TestACheckpointCoversTheSpanItFollows` and confirm it is RED: interleaving over a 500-line served text at N=200 yields checkpoints for 1-200, 201-400, 401-500, each covering exactly its own span and no other. [proof: mutation]
2. [S2] Implement the marker: `-- ck <8 hex>` on its own line, the hex from `crypto/rand`. ⚠ Not a hash of the content and not a counter — either can be derived by a caller that received nothing, which is the property the whole record rests on. [proof: mutation]
3. [S3] Implement the interleave over the text `read.Run` already produced, tracking served line numbers from the `NNN|` prefix so a span is what was SERVED rather than what was requested. [proof: mutation]
4. [S4] Implement the pending store under `internal/state`'s directory (ADR-004: nothing in the working tree), keyed by checkpoint, holding path, sha and span. Cap the entries per root and drop the oldest beyond it. [proof: mutation]
5. [S5] Assert a pending record reaches NO ledger: write a page, run `seen.Load`, and find nothing. Without this the store could be a second name for the same bug. [proof: mutation]
6. [S6] Run the package, `gofmt`, `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -v \
  -run 'TestACheckpointCoversTheSpanItFollows|TestAPendingRecordReachesNoLedger' 2>&1 | tee /tmp/adr031-t1.out \
  && grep -q '^--- PASS: TestACheckpointCoversTheSpanItFollows' /tmp/adr031-t1.out \
  && grep -q '^--- PASS: TestAPendingRecordReachesNoLedger' /tmp/adr031-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr031-t1.out \
  && go test ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACheckpointCoversTheSpanItFollows` | `internal/mcp/ack_test.go` | Interleaving yields one checkpoint per N served lines, each covering exactly the span since the previous marker, taken from the served line numbers rather than the request | — | S1, S2, S3 |
| `TestAPendingRecordReachesNoLedger` | `internal/mcp/ack_test.go` | After a page is served, `seen.Load` holds nothing for that path | — | S4, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/mcp/ack.go` |
| 2 — something selects it | T2 wires it into both tools; until then it is unreached, which is why T2 is not optional |
| 3 — the caller can discover it | The markers are in the served text and T3 documents them |
| 4 — it is used | Contract §68 (T3) drives the built server; no telemetry, per ADR-009 |

## Mutation Log

⚠ **S1's RED run was not observed: the implementation was written first and the test after.** That
inverts `.claude/rules/lifecycle.md`'s order and it is recorded rather than glossed, because a test
written against code that already passes is exactly the shape that proves nothing. The mutants below
are what stands in for it — each breaks one mechanism and the fence goes red — and they are the only
evidence this task has that its tests bind. A reader should weigh them accordingly.

## Invariants
- The checkpoint is RANDOM. Not a hash of the served text, not a counter, not derived from the request — a caller that received nothing must be unable to produce it.
- A pending record is not a ledger record. Nothing in `internal/seen` changes and nothing is written there until T2 promotes it.
- Spans come from the SERVED line numbers, so a page that served 1-2727 of 3619 yields checkpoints inside 1-2727 and none beyond.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- A checkpoint derived from the content would be computable by a caller that never received it, which would make the whole record ceremony. S2's mutant replaces `crypto/rand` with a content hash and the fence must go red.
- The store lives outside the tree (ADR-004) and could grow. S4 caps it.

## Stop Condition

Stop and ask if interleaving cannot be done over the text `read.Run` already produced — reaching into
`internal/read` to emit markers would put an MCP concern in the engine, which the engine go/no-go
forbids and which would put markers in CLI output.

## Out of Scope

- The `ack` field and promotion — T2
- The contract row and the instructions — T3

## Verification Log
