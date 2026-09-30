# Task ADR-102-T3: `mrw_write` sends the receipt on a ledger failure

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the receipt on a `LedgerError`; `writeReceipt.Error`; contract §197
**Consumes:** `writer.MutationOf(res)` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a landed write is answered with its receipt`, `the receipt names the error`, `a contract row drives the binary`, `no engine package changes`

## Goal

When `writer.Apply` returns `LedgerError`, `mrw_write` counts the write `applied`, feeds the recent window, and
answers with `boundedReceipt` — `isError` true, `applied` true, the written files, and `error` naming the ledger
failure. `writeReceipt` carries `error` whenever the write returned one.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | the `LedgerError` branch (`:702-709`); `writeReceipt.Error`; `boundedReceipt` fills it |
| `internal/mcp/testdata/legacy_golden.jsonl` | regen | the output schema gains `error`; logged in `era_test.go` |
| `internal/mcp/landed102_test.go` | edit | the test below |
| `scripts/contract.sh` | edit | §197 |

## Ordered Steps

1. [S1] Write the failing test `TestALedgerFailureStillSendsTheMCPReceipt`; confirm RED. [proof: mutation]
2. [S2] Answer with the receipt. [proof: mutation] Mutants: the branch returns the bare RPC error again; `error` left empty.
3. [S3] Contract §197, driving `$MRW mcp`: with the ledger read-only, a licensed `mrw_write` changes the file and answers a result (no JSON-RPC error) whose structured value says `applied` true and carries `error`; the pair, the ledger writable, answers without `error`. [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §197's rows printed]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestALedgerFailureStillSendsTheMCPReceipt' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestALedgerFailureStillSendsTheMCPReceipt \(' "$out" \
  && grep -q '^# 197\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestALedgerFailureStillSendsTheMCPReceipt` | `internal/mcp/landed102_test.go` | with the ledger read-only, `mrw_write` of a licensed plan answers a result (no RPC error): `isError` true, structured `applied` true, the file record written, a non-empty `error`; the file changed on disk; the tally counts `applied` 1 | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every `mrw write` and `mrw_write` goes through `writer.Apply` and the tallies; the tests drive `rootCommand` or `Serve` |
| 3 — the caller can discover it | the receipt, `stats`, and the ledger's licence on the next write |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/mcp/tools.go` · the ledger branch answers a bare RPC error again · acceptance-sha256:b5d2d7f3684705d81ad3d49562e88c44bd9a637e8efcb8423ebf13ced3951f28 · covers:a landed write is answered with its receipt
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/mcp/tools.go` · the receipt drops the error · acceptance-sha256:b5d2d7f3684705d81ad3d49562e88c44bd9a637e8efcb8423ebf13ced3951f28 · covers:the receipt names the error
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:b5d2d7f3684705d81ad3d49562e88c44bd9a637e8efcb8423ebf13ced3951f28 · covers:no engine package changes

## Invariants

- Exit codes and hunk statuses are unchanged.
- A write that landed whole is counted and recorded exactly as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-102 tasks, each in its own file.

## Verification Log
- 2026-09-30 · f1d5996* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5d2d7f3684705d81ad3d49562e88c44bd9a637e8efcb8423ebf13ced3951f28 · ms:318 · test-lock-sha256:743944ced0e868f8735c2dc8d1b2e2ec003de6c6d7cb3aa7a70e1df8eef070be · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QUxlZGdlckZhaWx1cmVTdGlsbFNlbmRzVGhlTUNQUmVjZWlwdAk3Njc5NTA5Mzg5ZmNhMDE5NDAyYzZhM2RkM2NkMzFlOTJhMWZmZmFjMzlkOTYyMmUzY2FkZmZkYmJjODY5YjczCmJvZHkJaW50ZXJuYWwvbWNwL2xhbmRlZDEwMl90ZXN0LmdvCVRlc3RBbk1DUFBhcnRpYWxDb21taXRJc0NvdW50ZWRBc1BhcnRpYWxseUFwcGxpZWQJZWI4OTY4ZmMyZGNlZDk3ZTBjZDFhZDg3N2JjNDUyOTBkMjg3MmYxNWMwNTFhM2NjZGE1OTYxN2RmYmZmNzU0ZQpib2R5CWludGVybmFsL21jcC9sYW5kZWQxMDJfdGVzdC5nbwlUZXN0QW5VbnJlcG9ydGFibGVXcml0ZVNheXNOb3RUb1JlcnVuCTllOWQ2NDM0ZjgyMzgzM2M1NWY3ZTdhYjY0YjRjODJmZTY1ZjU3OWQxZmUwNTIxM2FkYWE5YWIxYzI3ZmUwNzU
  ```
  --- last 6 line(s) of stdout
  === RUN   TestALedgerFailureStillSendsTheMCPReceipt
      landed102_test.go:78: tools/call returned a JSON-RPC error: map[code:-32603 message:open /var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestALedgerFailureStillSendsTheMCPReceipt4151028642/002/mrw/c296dc72d612c5a3/seen: permission denied]
  --- FAIL: TestALedgerFailureStillSendsTheMCPReceipt (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.132s
  FAIL
  ```
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5d2d7f3684705d81ad3d49562e88c44bd9a637e8efcb8423ebf13ced3951f28 · ms:321
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5d2d7f3684705d81ad3d49562e88c44bd9a637e8efcb8423ebf13ced3951f28 · ms:577
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5d2d7f3684705d81ad3d49562e88c44bd9a637e8efcb8423ebf13ced3951f28 · ms:359
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped on this branch (base f1d5996), exit 0, with §197 printed: through the built binary's mcp, a licensed mrw_write with the ledger read-only changes a.go and answers a result (no JSON-RPC error) with isError, structured applied true and a non-empty error; the pair with the ledger writable answers applied with no error key
