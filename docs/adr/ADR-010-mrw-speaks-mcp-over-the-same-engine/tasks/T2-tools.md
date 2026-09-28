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
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Two clauses ARE the go/no-go conditions rather than descriptions of them: exactly one requirement in
`go.mod`, and a clean working tree across the six engine directories. Neither consults a remote ref.
An earlier draft used ONLY `git merge-base HEAD origin/main`, which is right in intent and
incomplete in practice — it exits 128 in a clone that lacks the ref, and CI checks out at
`fetch-depth: 1`.

`git status --porcelain --untracked-files=all` rather than `git diff` because the diff form does not
see an UNTRACKED file: `internal/read/new.go` added by this task passes a diff and is exactly how an
engine grows a second answer. Verified 2026-09-03 — the diff form exits 0 with such a file present.

**Both forms are here because each has the blind spot the other closes**, measured 2026-09-03 by a
reviewer and reproduced: commit a one-line change to `internal/seen/seen.go` and the porcelain
clause returns EMPTY and passes, while the merge-base diff returns 1 and catches it; add an
untracked `internal/read/new.go` and the merge-base diff exits 0 while the porcelain clause catches
it. A working-tree gate cannot see a committed change and a commit-range gate cannot see an
untracked file, so "no engine change" needs both sentences or it is not the claim the ADR makes.
The merge-base clause is the one that fails in a shallow checkout; when it does, it fails LOUDLY
with 128 rather than passing, which is the right direction for a go/no-go to break in.

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
- 2026-09-03 · 71eef3e · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: drop the verdict from the envelope, so only one transport says what happened · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6
- 2026-09-03 · fbc5c0d · mutant killed · exit 1 · `internal/mcp/mcp.go` · rung 2 re-minted against the dual-clause fence: unroute tools/call so the handlers are unreachable · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6
- 2026-09-03 · dae1c61 · mutant killed · exit 1 · `internal/mcp/tools.go` · S4 re-minted: stop sharing the ledger, so an MCP read no longer licenses a CLI write and ADR-002 would hold per transport instead of once · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6
- 2026-09-28 · 86c576d* · mutant killed · exit 1 · `internal/mcp/ack.go` · an acknowledged MCP read is recorded under another root, so the CLI ledger never sees it · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · covers:the shared ledger
- 2026-09-28 · 86c576d* · mutant survived · exit 0 · `internal/mcp/tools.go` · drop the in-process gate around a tool call · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · covers:the in-process serialization
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-28 · 86c576d* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP receipt stops carrying the engine result the CLI reports · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · covers:the identical apply.Result

**The in-process gate is left unbound, on purpose (2026-09-28, C3).** Removing `gate.Lock()` around a
tool call SURVIVES this fence: `TestConcurrentToolCallsDoNotLoseALedgerEntry` stays green because
ADR-038 (#154) made the ledger write one writer across processes, so the file lock under
`seen.Record` does the job the gate did. The name stays in Rests-on so the lint keeps saying nothing
kills it; a mutant chosen only to satisfy that counter is the failure this pipeline refuses. The gate
still serializes `callModern`/`callReserve`, which only `go test -race` could catch, and the fence
does not run it.

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
- 2026-09-03 · 517c285 · exit 1 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:12310
  ```
  --- last 10 line(s) of stdout (of 348 after folding 348 raw)
    PASS  mrw mcp applies the same plan over a real pipe
    PASS  every stdout line is a valid MCP message
    PASS  and nothing was written to stderr
    PASS  the MCP receipt equals the CLI receipt for the same plan
    PASS  and the file really changed
    PASS  a live server answers a read over a real pipe
    PASS  the server is killed mid-session
    PASS  a NEW server completes a write the killed one licensed
    PASS  and a CLI write after the killed server is licensed too
  1 assertion(s) FAILED
  ```
- 2026-09-03 · 71eef3e · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:10977
- 2026-09-03 · fbc5c0d · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:11733
- 2026-09-03 · a679d3b · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:11081
- 2026-09-03 · dae1c61 · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:8758
- 2026-09-03 · d2540df · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:8352
- 2026-09-28 · 86c576d* · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:108230 · test-lock-sha256:d7353c3ebf0815b1dcba1468d8a381135a44d51c5f41974392ab3b540c22fde8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBUGFnZUlzS25vd25CeUl0c1NlcnZlZFRleHQJMGM3NWY4NGI2MTExZWU5NTI1ODYyMWNhOTU1NDdiYWE2ZTRiNDU1OTZmM2Y5MDUwYTY3ZjI0Zjc2NmZkZmQyOQpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBUGFnZUlzTWVhc3VyZWRBZnRlckl0c01hcmtlcnNBbmRGb290ZXIJZWQ2MzBiNjg4YTkyMWNlMTk0ZTgyNWY3NTJlYzlmZGQ4ZWVkZDg0MWQxODBmNTczOTY1YjNmMDllNTNkMTVhOApib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBUGFnZUxpY2Vuc2VzT25seVdoYXRJdFNlcnZlZAk0YWQ1N2FlYjhmNjU1OGQ4Mjc4M2JkNDJiNDA0MmU1ZWJmZDliZjE1MjRhNGQ4MDAxMzFkOGM1NTE1Y2VkNTM3CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFQYWdlVGhhdENhbm5vdEZpdElzTm90QVBhZ2UJNjU3Nzg3OTNmMWY1Y2IyYzlmOTE4NGM3NzFjNzA1NjQzNjRjZDc5ODllNGQ1YjA2OGRlNzhjYTM1NTZkZDRmNApib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBUGFnZWRGb290ZXJDYXJyaWVzVGhlT25lUnVsZQlhNGQxZWQxNDQzYTBmMmJhZTNlNDU0MTZjNjI5NGNhMjliNzdhNGVlYzgyNmMxYzg3MDhkN2FhOGE2MDViNmY2CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFQYWdlZFJlYWRSZWFzc2VtYmxlc1RoZVdob2xlRmlsZQkzMzUzZjg1MzQ2YWMzM2IwNzE0ZWNmOGI4ZGIzMWY4MDMwZjViYjgwZGQzN2U5YjIxYzZkMTEyMzQxODQwNDQ3CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFSZWFkT3ZlclRoZUxpbWl0SXNSZWZ1c2VkTm90VHJ1bmNhdGVkCTAwZmNiZWNkOGRiY2JmZjAxMjdlYmYzNjQwYTQ0M2Q5ZTdlM2Q3Y2ExMmFjNjcwOTgyNDJmMTIyNzc3M2RlZjYKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVJlYWRUaGF0U2VydmVkTm90aGluZ0lzQW5FcnJvcgk2YTdhNDQzNDU4ZTBmNjVkMjVlMzBiNDkxNTY4ZWVkOTI0ZGE1OTE1ZjIzOGViMGQ5MWM5ZTZjY2NjZmQ3MWI5CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFSZWFkVW5kZXJUaGVMaW1pdElzVW5jaGFuZ2VkCTcxNDczYTViOTE3MzFkZmUxOTA5NTJhZTQyOWU1NWU0MWRlMWJhODQxM2JlYmFmYmY2YjM2YmY3NWVjZDRlODcKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QVdhbGtQcm9ibGVtSXNSZXBvcnRlZEFuZE5vdFN3YWxsb3dlZAkzZTZlNjI5MDhiZTZhYjg5ZjZiOWFiOTE1MDEwODM0ZWU5NDQxM2Q1MmRiMTIwYjQ3NzRlZjhkYjU1MjU5ODE1CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFXYWxrUHJvYmxlbVN1cnZpdmVzQVZhbGlkU2libGluZwliZWU3MDg4ZTVjYzIyMWU3NzUyNjk1NDUyNDA1NWM1ODlmOGYwMjBlNjI5OGM1MGY4MWNhYzk3NWM0YWI5MWJiCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFXcml0ZVRvQW5VbnJlYWRGaWxlSXNSZWZ1c2VkT3Zlck1DUAliYjJhNGUxZmMyNDE0YTNjYzUwNjMxNzVhMDM0Yzk3MGNlZDkwNWE3NTdlMGI2ZGEyZGYwN2VjYTliOWU5OWFjCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFja09uQVJlYWRQcm9tb3Rlc1Rvbwk3N2I4NzFjYzNkMzIwOWE4NzU0YzI5MTY1ZWY0ZGFhOTBlN2RmNTIxZDZhZDA1NGUwODMyNjY4MGI1ZWQ3NTA4CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFmdGVyV2l0aG91dEdyZXBJc1JlZnVzZWQJOTE3NTJiMDdiZGY2NjcwMzhiNWFjYjhmZTgyODIzZmM0MmMyYjcxZjg1NjFiZmM0M2IwNWQzYmUwMWE1Y2IyOApib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBbkluZGV4VG9vTGFyZ2VUb1NlcnZlUGFnZXNCeUZpbGUJNmI2MjQxNGVjYTlkMzgxMmNhNTY4NGQxMTQ2OWZlNjM1NTYwZGZlZGZjNGI3YzBhMTgxZjcxYWMwNzYxMmI3Ygpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RBbk1DUFJlYWRMaWNlbnNlc0FDTElXcml0ZQk5MzZlMWRiOGQ3ZmU2NTZiZTcxNTI0NTM5MTM1MTc1ZWVjMmI5MDI5MmJkMzkyMTYxNzYzZDc5N2FhZDZjZmQxCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEFuT3ZlcnNpemVkR3JlcFJldHVybnNUaGVJbmRleEFuZE5vdEFEZWFkRW5kCTczZDUyZDcwNWIyODc1YTIxYzNjNDM3YmUxZjVhN2M2NTE1NDczN2MzMWM0MjVmYmQ5MDQwZDk2YWMyY2VkNTEKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0QW5PdmVyc2l6ZWRSZWFkU3RpbGxSZWFkc0FzSW5jb21wbGV0ZQkzOGYyNTRiNjVmM2RiYzAxZjhkMmZmMWU2OGM0ODVkMzk4NjE1NThjOTU2ZGY0ZTY5ZjA1MzAwODI3Mzg4OWMxCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdEJvdGhUb29sc0FkdmVydGlzZUFjawk5YTM2MTNlMzBhZjYxNWVmNjA4NGI0ZjMwZjRmNGYwZWQxZjIxMzZmMzY2MTM2Y2ZmNDBkOWU5ZWIxNTM0Y2I5CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdENvbmN1cnJlbnRUb29sQ2FsbHNEb05vdExvc2VBTGVkZ2VyRW50cnkJNzI3NWRlMWM3ZDUxNTE3ZWUxOTc5YjBlZjY5YzExNzU2ODJlNTQ1YTEzMGQ3ZDdmMjZjMDhmNWNjN2I5ZWMwMgpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RHcmVwUmVmdXNlc0FSYW5nZWRTcGVjCTg0NjE5YjhhNGUyYzNiOTkyNjJjNzQ1NTJlZWE0MjA3N2I0ZjMwZmIzZTgwOGRkODM5OGVhMzE3ODE2ZjkyMzQKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0R3JlcFNlcnZlc1doYXRJdEZpbmRzQW5kUmVjb3Jkc0l0CTUyOTE3MTNiODBjZmM4OGYyMzg5MmRlZGY4MDM0MGU2MmNhZjg2MTY2MWRhNmYyMTRmYjFiNDYwYzExZDY2ZjIKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0Tm9HcmVwQW5zd2VyRXhjZWVkc1RoZURlY2xhcmVkQ2FwCTBlMzJmMzdjMjUwMzQ0ZTdkN2FkNTYyNDgyZjYzZGZiM2ZiYmFhMTAwZDg0NDE4ZDhmZTU2OTQxOGU4N2I2YTMKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0VGhlQ0xJUmVhZElzVW5hZmZlY3RlZEJ5VGhlTUNQTGltaXQJNGU1ODBlYjE4ZWU2MTQwZTYwZWRmZjMzNDc2OTdlNmYwZWY2NTM1MjI4Y2UzNGQzZDg5NTI3NDk5MWUxZGZmNQpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RUaGVDYXBwZWRXcml0ZXJSZXRhaW5zTm9Nb3JlVGhhbkl0c0xpbWl0CTU3NzdjMTI1N2I0YWVhZTg3Nzg1NDk5NDY3NDM1MmIwYTY3ZTNiNTYxYmZhMzg2MjU2OGE1OGJjMWY3Zjk4MTEKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0VGhlSW5kZXhTdXJ2aXZlc0FQYXR0ZXJuVGhhdExvb2tzTGlrZUFSYW5nZQliMjYwODIxZDk1ZTU1ZTQ3ZGE4YjdjZjM3MjhjNzRiZTYxMGQ1Mjk4MDIwOGQ0MzQ3MTBkZTEyZWQ3NTA0N2UxCmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdFRoZVJlYWRUb29sT2JzZXJ2ZXNXaGF0VGhlQ0xJV291bGRPYnNlcnZlCWJhNDZiYmJlMjE5OWI5MDM5OTFhMDk0NjFhMTM0NDgzZTIzYTdkNWYyMTc4NTM0ZTZiNjhmYzkwNDY3YmEwNGYKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0VGhlUmVmdXNhbERvZXNOb3RJbnZlbnRBbkludmFsaWRTcGVjCTFiZjVmZWFkZGFkYTgyMmQyZWQ0YzU2MzY4YzU2NzQ3ODdlNzVmY2I4ODEzNzgxMjZmY2FkMGY4ZWFjMTgyMDEKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0VGhlUmVmdXNhbE5hbWVzVGhlTGltaXRBbmRBUmFuZ2VUb1JldHJ5CTM4MTZmYTAxMjk3Mjk2NDU4NDk2NzFlOTM1MzYwMjFjNGRmN2M5OTE1NzA5ZDM5OWVmYmYzNDJhNzQzMTRkMjgKYm9keQlpbnRlcm5hbC9tY3AvdG9vbHNfdGVzdC5nbwlUZXN0VGhlVG9vbFJlc3VsdENhcnJpZXNDb250ZW50QW5kU3RydWN0dXJlZENvbnRlbnQJMGFkMmIzN2ZmOTBkODNmNWI1MjgxN2YxOWJjMTZmYTkwZjIwZTcxZDIyNTg4NWM4NGY2ZmI0NDliOGY5YjFhNwpib2R5CWludGVybmFsL21jcC90b29sc190ZXN0LmdvCVRlc3RUaGVVbnNlcnZhYmxlTGluZUlzRGlhZ25vc2VkUGVyRmlsZQk1M2M2YjE0ZWVmZGIzYzlhMWU1ODM3MmE2ZGJmODU3ODk4NWM3YzJmMTc0YjRhN2RkOGEzMDQ2MTU3YzU3Yjg5CmJvZHkJaW50ZXJuYWwvbWNwL3Rvb2xzX3Rlc3QuZ28JVGVzdFRoZVdyaXRlVG9vbFJldHVybnNUaGVTYW1lUmVzdWx0QXNUaGVDTEkJYWEyNzIwZTcwM2Q4ZDFhY2MzZTI4ZjIwMTI0Y2U0ODM2YzM2M2EyYjE5N2UwNmU4M2Q4NDRkMDQwNDU4OGQzOA
- 2026-09-28 · 86c576d* · exit 1 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:185525
  ```
  --- last 10 line(s) of stdout (of 1734 after folding 1734 raw)
    PASS  a write whose check passes exits 0
    PASS  the ag child's background process is gone when mrw returns
    PASS  the ck child's background process is gone when mrw returns
    PASS  a write whose check times out exits 3
    PASS  and the timed-out check names its log, which is there
    PASS  a check log older than a week is pruned
    PASS  a young one is kept
    PASS  and the receipt says what it removed
    PASS  no process of this run survives it
  2 assertion(s) FAILED
  --- last 4 line(s) of stderr
  ./scripts/contract.sh: line 2938: 65564 Alarm clock: 14         ( perl -e 'alarm shift; exec @ARGV' 1 bash -c "$(declare -f mk55 hook55); R55='$R55'; HOOK='$HOOK'; hook55 s9n Bash '{\"command\":\"env $many55 -C docs cat adr/x.md\"}'" > "$R55/many.out" )
  Terminated: 15
  ./scripts/contract.sh: line 6169: 88538 Alarm clock: 14         PATH="$(dirname "$MRW"):$PATH" perl -e 'alarm shift; exec @ARGV' 5 bash -c "$1" < "$WORK/117.fifo" > "$WORK/117.out" 2>&1
  ./scripts/contract.sh: line 6760: 89517 Killed: 9               "$MRW" -C "$R" write "$R/p143.mrw" > "$R/out143" 2>&1
  ```
- 2026-09-28 · 86c576d* · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:107283
- 2026-09-28 · 86c576d* · exit 0 · `set -o pipefail …` · acceptance-sha256:d425240fd07a542d44e74114d4f395d158c62b38624ef73798e40e7f40ae57d6 · ms:117481
