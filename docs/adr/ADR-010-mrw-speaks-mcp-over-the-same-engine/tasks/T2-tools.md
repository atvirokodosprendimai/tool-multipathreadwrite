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
3. [S3] Implement `mrw_write` as an adapter over `apply.Apply`, returning a `CallToolResult` — the
   protocol's envelope, not a bare `apply.Result`. `content` carries the receipt rendered as text
   for hosts that display it; `structuredContent` carries the `apply.Result` itself, serialized by
   the same code that writes the `--json` receipt. Two serializations of one result is how the
   transports start to drift, and a bare result is a tool a host may reject outright. Source:
   https://modelcontextprotocol.io/specification/2025-06-18/server/tools. [proof: mutation]
4. [S4] Share the ledger. A file read over MCP licenses a CLI write and the reverse — one guarantee,
   not one per transport. This is what makes ADR-002 hold across both. [proof: mutation]
5. [S5] Serialize `tools/call` in-process with a mutex, so calls made THROUGH the server no longer
   race. This does not make the README's "one call at a time" obsolete: a CLI process running beside
   the server is still a second writer of the same ledger file, and that is still the CLI
   limitation. One server is one writer; this is three lines rather than a ledger redesign.
   [proof: acceptance]
6. [S6] Add contract §38: start `mrw mcp` as a real subprocess, speak newline-delimited JSON-RPC
   down its stdin, and compare the write receipt against the one `mrw write --json` produces for the
   same plan. Assert too that every line the server wrote to stdout parses as JSON — the spec's
   stdout rule, tested. Then kill it mid-session, START A NEW SERVER, and complete a new `mrw_write`
   over MCP as well as a CLI write: ADR-001's objection was that a server is unrecoverable
   mid-session, and a test that only proves the CLI still works has answered a smaller question than
   the one that was asked. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr010-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr010-t2.out \
  && [ "$(grep -cE '^--- PASS: (TestTheWriteToolReturnsTheSameResultAsTheCLI|TestTheReadToolObservesWhatTheCLIWouldObserve|TestAnMCPReadLicensesACLIWrite|TestAWriteToAnUnreadFileIsRefusedOverMCP|TestConcurrentToolCallsDoNotLoseALedgerEntry|TestTheToolResultCarriesContentAndStructuredContent)\b' /tmp/adr010-t2.out)" = "6" ] \
  && grep -q '^# 38\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && grep -q '^require github.com/urfave/cli/v3 ' go.mod \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Two clauses ARE the go/no-go conditions rather than descriptions of them: exactly one requirement in
`go.mod`, and a clean working tree across the six engine directories. Neither consults a remote ref.
An earlier draft diffed both against `git merge-base HEAD origin/main`, which is right in intent and
wrong in practice — it exits 128 in a clone that lacks the ref, and CI checks out at `fetch-depth: 1`.

`git status --porcelain --untracked-files=all` rather than `git diff` because the diff form does not
see an UNTRACKED file: `internal/read/new.go` added by this task passes a diff and is exactly how an
engine grows a second answer. Verified 2026-09-03 — the diff form exits 0 with such a file present.

What neither clause proves is that a change already COMMITTED did not touch the engine; this is a
working-tree gate, run before the commit it gates. The review diff and the SHA in the verification
log are what cover the rest, and saying so here is better than a clause that looks stronger than it is.

The named-test count is not `-run`: a `-run` regex silently drops names it does not match, and
running the package without naming the tests lets T1's own tests satisfy this clause. Counting
`--- PASS:` lines for the six T2 tests by name is what makes this fence about T2.

Confirm `# 38.` is unused before relying on that clause. ADR-007-T3's fence named `# 15.`, which
already existed, and so passed on an untouched `contract.sh` from the day it was written. Note what
that grep does and does not prove: it establishes the section exists, never that it asserts
anything. Its non-vacuity is proved by the `adr-verify --mutant` entry this task must record — break
the `tools/call` routing and §38 must go red.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWriteToolReturnsTheSameResultAsTheCLI` | `internal/mcp/tools_test.go` | **the ADR's Enforced-by** — the DECODED `structuredContent` equals the CLI's `--json` receipt | — | S1, S3 |
| `TestTheToolResultCarriesContentAndStructuredContent` | `internal/mcp/tools_test.go` | S3 — the reply is a `CallToolResult`, so a host does not reject the tool | — | S1, S3 |
| `TestTheReadToolObservesWhatTheCLIWouldObserve` | `internal/mcp/tools_test.go` | S2 — the ledger entry is identical | — | S1, S2 |
| `TestAnMCPReadLicensesACLIWrite` | `internal/mcp/tools_test.go` | S4 — one guarantee across both transports | — | S1, S4 |
| `TestAWriteToAnUnreadFileIsRefusedOverMCP` | `internal/mcp/tools_test.go` | ADR-002 holds on the new path | — | S1, S4 |
| `TestConcurrentToolCallsDoNotLoseALedgerEntry` | `internal/mcp/tools_test.go` | S5 — the concurrency gap is actually closed | — | S1, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the six tests above |
| 2 — something selects it | `tools/call` routing in `mcp.go` (S3); removing the `mrw_write` case makes `TestTheWriteToolReturnsTheSameResultAsTheCLI` and §38 red, both inside the fence |
| 3 — the caller can discover it | `tools/list` advertises both schemas — T1 declared them, and a host reads that list rather than the source |
| 4 — it is used | `mrw stats` counts the plan outcomes recorded by BOTH transports, since the tally is written by the shared engine path — the first thing in this repository that measures a transport at all |

## Mutation Log

- 2026-09-03 · 1eaffaa · mutant killed · exit 1 · `internal/mcp/mcp.go` · rung 2: unroute tools/call so the handlers are unreachable — the Enforced-by test and contract §38 must both go red · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141
- 2026-09-03 · 86dab85 · mutant killed · exit 1 · `internal/mcp/tools.go` · S4: stop sharing the ledger — an MCP read no longer licenses a CLI write, so ADR-002 would hold per transport instead of once · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141
- 2026-09-03 · a10d3a7 · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: drop the verdict from the envelope — the transports would still both "work" while only one of them says what happened · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141

## Invariants

- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and
  `internal/state` carry no change and no new file when this task's fence runs.
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
- 2026-09-03 · 916188b* · exit 1 · `set -o pipefail …` · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141 · ms:639
  ```
  --- last 10 line(s) of stdout (of 35 after folding 35 raw)
  --- FAIL: TestAnMCPReadLicensesACLIWrite (0.00s)
  === RUN   TestAWriteToAnUnreadFileIsRefusedOverMCP
      tools_test.go:200: tools/call returned a JSON-RPC error: map[code:-32601 message:method not found: tools/call]
  --- FAIL: TestAWriteToAnUnreadFileIsRefusedOverMCP (0.00s)
  === RUN   TestConcurrentToolCallsDoNotLoseALedgerEntry
      tools_test.go:261: 12 of 12 reads left no ledger entry: [fa.txt fb.txt fc.txt fd.txt fe.txt ff.txt fg.txt fh.txt fi.txt fj.txt fk.txt fl.txt]
  --- FAIL: TestConcurrentToolCallsDoNotLoseALedgerEntry (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.377s
  FAIL
  ```
- 2026-09-03 · 21323d9* · exit 0 · `set -o pipefail …` · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141 · ms:10107
- 2026-09-03 · 1eaffaa · exit 0 · `set -o pipefail …` · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141 · ms:9401
- 2026-09-03 · 86dab85 · exit 0 · `set -o pipefail …` · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141 · ms:8072
- 2026-09-03 · a10d3a7 · exit 0 · `set -o pipefail …` · acceptance-sha256:ef6aefdb5fb626df7d515fc315fc58bf58c63ea7e07fe346e6b1f8f6454d8141 · ms:8033
