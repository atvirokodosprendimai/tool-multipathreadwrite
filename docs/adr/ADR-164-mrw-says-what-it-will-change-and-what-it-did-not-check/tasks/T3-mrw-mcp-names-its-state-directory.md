# Task ADR-164-T3: the state-directory hint reaches every surface

**Depends-on:** T2
**Blocked-on:** PR #460 merged: `git merge-base --is-ancestor $(gh pr view 460 --json mergeCommit -q .mergeCommit.oid) origin/main`
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `internal/statehint` (`statehint.Hint`); `read.BuildError.Err`; MCP routes wrapped with `%w` and passed through `statehint.Hint`
**Consumes:** `internal/mcp/tools.go` as T2 leaves it (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the package judges by type`, `a plan maker keeps the error`, `every MCP route is passed through it`, `the CLI still hints`, `no answer changes shape`, `the binary answers with it`

## Goal

Decision 3: the CLI's plan makers and the MCP surface say what the CLI's read and write say when they cannot use the state directory.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/statehint/statehint.go` | add | `Hint`, `about`, `within`, moved from `cmd/mrw/statehint.go` |
| `internal/statehint/statehint_test.go` | add | TestAStateFailureIsNamedByItsType: a new name for #460's type cases, moved |
| `cmd/mrw/statehint.go` | edit | keeps `exitWith`, `exitSaying` and `exitError` (gaining an exit code, since `exitOf` exits 1 or 2); `main` calls `statehint.Hint` |
| `cmd/mrw/sub.go` | edit | `exitOf` keeps the `BuildError` and its exit code |
| `cmd/mrw/statehint_test.go` | edit | the `--sub` case #460 left out, back in |
| `internal/read/plansource.go` | edit | `BuildError.Err` and `Unwrap`; `RangeFreePaths` and the walk error keep the error they were built from, words and `Usage` unchanged |
| `internal/mcp/tools.go`, `internal/mcp/ack.go`, `internal/mcp/sub161.go` | edit | each route from S1 wraps with `%w` and passes `statehint.Hint` |
| `internal/mcp/statehint164_test.go` | add | the MCP test |
| `scripts/contract.sh` | edit | §275 |

## Ordered Steps

1. [S1] Enumerate the routes and record here the command, the count and the list: `mrw read --grep 'seen\.(Record|Snapshot|Drop|Load)\(|state\.(Hold|Dir|Path|WriteSynced)\(|iter\.Load\(|\.Land\(|land\.Err|LedgerErr|promote\(|hold\(' --exclude '*_test.go' internal/mcp`, keeping call sites and dropping helper definitions (`func hold`, `func promote`, `pendingPath`, `readPending`). Say for `firstPage`'s dropped `hold` error (`tools.go`, `return callToolResult{}, false`) whether it is routed or stays dropped, and why. Then write the failing tests and see them red: TestTheMCPSurfaceNamesItsStateDirectory (with `XDG_STATE_HOME` a file: `mrw_read` of a file, `mrw_read` with `grep`+`sub` and an explicit path, `mrw_write` with a plan, and `mrw_write` with `grep`+`sub` and an explicit path each answer naming `mrw's state directory`, with the same `isError` or JSON-RPC shape and code as today; an MCP error not about the state directory is unchanged), and the `--sub` case in `cmd/mrw/statehint_test.go`. [proof: acceptance]
2. [S2] `BuildError.Err`; `exitOf` keeps it. [proof: mutation]
3. [S3] Move the judgement into `internal/statehint`; `main` calls it. [proof: mutation]
4. [S4] Wrap and route each site from S1. [proof: mutation]
5. [S5] Contract §275: `mrw mcp` driven over stdin with `XDG_STATE_HOME` a file names the state directory; `mrw write --grep a --sub b x` and `mrw read --grep a --sub b x` likewise (the builders are shared); the pair: a writable one serves. [proof: human: the executor runs ./scripts/contract.sh once and reads the §275 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestTheMCPSurfaceNamesItsStateDirectory$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheMCPSurfaceNamesItsStateDirectory \(' "$out" \
  && go test ./internal/statehint/ ./cmd/mrw/ ./internal/read/ -count=1 -timeout 300s -run 'TestAStateFailureIsNamedByItsType|TestAStateDirectoryThatCannotBeMadeNamesTheVariableThatMovesIt|TestAWriteThatCannotUseTheStateDirectoryNamesIt|TestALandingThatCannotTakeTheWriteLockNamesTheStateDirectory|TestNoCommandExitsWithAnErrorThroughCliExit' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAStateFailureIsNamedByItsType \(' "$out" \
  && grep -qE '^--- PASS: TestAStateDirectoryThatCannotBeMadeNamesTheVariableThatMovesIt \(' "$out" \
  && grep -qE '^--- PASS: TestAWriteThatCannotUseTheStateDirectoryNamesIt \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep '^# 275\. ' scripts/contract.sh > /tmp/adr164-t3-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWriteToolReturnsTheSameResultAsTheCLI` | `internal/mcp/tools_test.go` | the two surfaces agree (ADR-010), which this task extends to the state error | none | — |

Planned, named without backticks until they exist on `main`: TestTheMCPSurfaceNamesItsStateDirectory in `internal/mcp/statehint164_test.go` (S1, S4), TestAStateFailureIsNamedByItsType in `internal/statehint/statehint_test.go` (S3), and #460's TestAStateDirectoryThatCannotBeMadeNamesTheVariableThatMovesIt, TestAWriteThatCannotUseTheStateDirectoryNamesIt (S1, S2: its `--sub` case) and TestNoCommandExitsWithAnErrorThroughCliExit in `cmd/mrw/statehint_test.go`.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `statehint.Hint`, `BuildError.Err` |
| 2 — something selects it | `main`, `exitOf` and each MCP route from S1; contract §275 drives the binary |
| 3 — the caller can discover it | the message itself |
| 4 — it is used | nothing measures this; the field report was a CLI session, the MCP half is parity |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- No answer changes shape: an `isError` stays one, a JSON-RPC error stays one, with the same code; every CLI exit code and `BuildError` message is unchanged.
- `cmd/mrw`'s behaviour from #460 is unchanged.

## Risks

- A route missed by S1's command keeps the bare message; S1 records the command and the list so it can be re-run.

## Out of Scope

- Changing which MCP failures are `isError` and which are JSON-RPC errors (permanent: boundary: ADR-067, ADR-072 own that split)

## Stop Condition

Stop and bring it to Zy if routing a site through `statehint` would change whether its answer is an `isError` result or a JSON-RPC error, or its code.
