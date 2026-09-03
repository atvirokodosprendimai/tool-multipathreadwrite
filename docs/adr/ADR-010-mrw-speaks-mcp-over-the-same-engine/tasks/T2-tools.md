# Task ADR-010-T2: Two tools over the unchanged engine, one answer

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** MCP tools `mrw_read` and `mrw_write`
**Consumes:** `mcp.Serve`, the JSON-RPC frame (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the shared ledger`, `the identical apply.Result`, `the in-process serialization`

## Goal

`mrw_read` and `mrw_write` call the same engine functions `cmd/mrw` calls, share the same ledger, and
return the same per-hunk verdict — so the two transports cannot disagree about what happened.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | add | the two handlers, each a thin adapter over `read.Run` / `apply.Apply` |
| `internal/mcp/tools_test.go` | add | its tests, including this ADR's `Enforced-by` |
| `internal/mcp/mcp.go` | edit | route `tools/call` to the handlers T1 declared in `tools/list` |
| `scripts/contract.sh` | edit | §38 — drive the server through a REAL PIPE, read then write, and assert the receipt matches the CLI's |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a `mrw_write` over MCP returns the same
   `apply.Result` the CLI produces for the same plan; a write to a file not read over EITHER
   transport is refused; two concurrent `tools/call` requests do not lose a ledger entry.
2. [S2] Implement `mrw_read` as an adapter over `read.Run` — parse the spec strings the CLI already
   parses, call the same function, return what it observed. No new spec syntax; the format is
   ADR-001's and this task does not own it.
3. [S3] Implement `mrw_write` as an adapter over `apply.Apply`, returning the SAME `apply.Result`
   the `--json` receipt carries, serialized once. Two serializations of one result is how the
   transports start to drift. [proof: mutation]
4. [S4] Share the ledger. A file read over MCP licenses a CLI write and the reverse — one guarantee,
   not one per transport. This is what makes ADR-002 hold across both. [proof: mutation]
5. [S5] Serialize `tools/call` in-process with a mutex, so the README's "one call at a time" applies
   to the CLI path only. One server is one writer; this is the concurrency gap closing for free,
   and it is three lines rather than a ledger redesign. [proof: acceptance]
6. [S6] Add contract §38: start `mrw mcp` as a real subprocess, speak framed JSON-RPC down its
   stdin, and compare the write receipt against the one `mrw write --json` produces for the same
   plan. Kill it mid-session and assert the ledger still licenses the next CLI write — that is
   ADR-001's original objection, tested. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr010-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr010-t2.out \
  && grep -q '^# 38\.' scripts/contract.sh \
  && [ "$(grep -c '^	' go.mod)" = "1" ] \
  && git diff --quiet -- internal/read internal/apply internal/plan internal/seen \
  && go test ./... \
  && ./scripts/contract.sh
```

Two clauses ARE the go/no-go conditions rather than descriptions of them: the `go.mod` count, and
`git diff --quiet` over the engine directories. If this task changed the engine, the fence goes red
— which is the ADR's "one engine, one answer" made mechanical instead of aspirational.

Confirm `# 38.` is unused before relying on that clause. ADR-007-T3's fence named `# 15.`, which
already existed, and so passed on an untouched `contract.sh` from the day it was written.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWriteToolReturnsTheSameResultAsTheCLI` | `internal/mcp/tools_test.go` | **the ADR's Enforced-by** — one engine, one answer | — | S1, S3 |
| `TestTheReadToolObservesWhatTheCLIWouldObserve` | `internal/mcp/tools_test.go` | S2 — the ledger entry is identical | — | S1, S2 |
| `TestAnMCPReadLicensesACLIWrite` | `internal/mcp/tools_test.go` | S4 — one guarantee across both transports | — | S1, S4 |
| `TestAWriteToAnUnreadFileIsRefusedOverMCP` | `internal/mcp/tools_test.go` | ADR-002 holds on the new path | — | S1, S4 |
| `TestConcurrentToolCallsDoNotLoseALedgerEntry` | `internal/mcp/tools_test.go` | S5 — the concurrency gap is actually closed | — | S1, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests above |
| 2 — something selects it | `tools/call` routing in `mcp.go` (S3); removing the `mrw_write` case makes `TestTheWriteToolReturnsTheSameResultAsTheCLI` and §38 red, both inside the fence |
| 3 — the caller can discover it | `tools/list` advertises both schemas — T1 declared them, and a host reads that list rather than the source |
| 4 — it is used | `mrw stats` counts the plan outcomes recorded by BOTH transports, since the tally is written by the shared engine path — the first thing in this repository that measures a transport at all |

## Mutation Log

## Invariants

- `internal/read`, `internal/apply`, `internal/plan` and `internal/seen` are byte-identical after
  this task.
- A verdict returned over MCP is the same `apply.Result` value the CLI would return, serialized by
  the same code.
- The ledger is one file, shared, and neither transport can license an edit the other would refuse.

## Risks

- A mutex around `tools/call` serializes an agent's parallel calls, which is a throughput cost. It
  is the right trade while the ledger is a whole-file rewrite, and the alternative — per-file
  locking — is a ledger redesign this ADR refuses.
- An adapter is where "just one small difference" enters. The `git diff --quiet` clause is what
  makes that visible rather than arguable.

## Stop Condition

Stop if either tool needs to compute a verdict, parse a spec, or decide a refusal that `cmd/mrw`
does not already compute. The tools are adapters. The moment one holds logic, there are two answers
to "did this apply?" and this ADR has the wrong shape — say so and withdraw `--grep`-style rather
than shipping a second engine behind a protocol.

## Out of Scope

- Exposing `check`, `iter`, `seen` or `stats` as tools (deferred: docs/adr/BACKLOG.md)
- MCP resources or prompts (permanent: boundary: stated in the parent ADR)

## Verification Log
