# Task ADR-113-T2: an MCP write runs the check and reports it

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `mrw_write` `check` argument; `check` and `drift` in its receipt
**Consumes:** `writer.Prepare` (T1), `Prepared.Land` (T1), `Landed.Verify` (T1), `Landed.Settle` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an MCP write runs the check and reports it`

## Goal

`mrw_write` goes through `Prepare`, `Land`, `Verify` and `Settle`, runs the check by ADR-054's rule unless `check: false`, and returns `check` and `drift` in its receipt, with the verdict leading the text, never elided, and named in the terminal sentence the write floor is measured against; a check that ran is never `isError`, one that could not run is, with the receipt.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `writeArgs.Check`, the phases, `writeReceipt.Check`/`Drift`, the elision order, the terminal sentence, the floor, the hint |
| `internal/mcp/mcp.go` | edit | the `check` property in `mrw_write`'s input schema; the description |
| `internal/mcp/schema.go` | edit | the output schema's descriptions of `check`, its keys, and `drift`; `elided` says the check's tail goes first |
| `internal/mcp/instructions.go` | edit | the handshake's WHICH SURFACE line |
| `internal/mcp/check113_test.go` | add | `TestAnMCPWriteRunsTheCheck` |
| `cmd/mrw/surfaces113_test.go` | add | `TestTheTwoSurfacesCountAWriteAlike`: both surfaces in-process — the CLI through its command, MCP through `mcp.Serve` |
| `internal/mcp/says098_test.go` | edit | the M11 rows that required "this tool runs none" |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated |
| `docs/receipts.txt` | edit | `mcp_write check` and every key under it, `mcp_write drift` |
| `AGENTS.md`, `README.md` | edit | an MCP write is checked; inspect before re-sending after a lost receipt |
| `scripts/contract.sh` | edit | §213 |

## Ordered Steps

1. [S1] Write `TestAnMCPWriteRunsTheCheck` and `TestTheTwoSurfacesCountAWriteAlike` (the same plans through `mrw write --json` and `mrw_write`: the same tally, pricing, hunk verdicts and check verdict); confirm RED on the check-due rows. [proof: mutation]
2. [S2] `mrw_write` on the phases, `check` in the schema and the receipt; a failed check not `isError`, one that could not start `isError` with the receipt; the tail elided first and the verdict never; the terminal sentence and the floor carry the verdict. Mutants: `Verify` not called; `check: false` ignored; the verdict elided. [proof: mutation]
3. [S3] The prose and receipts: description, handshake (re-measure its byte budget), AGENTS.md, README, receipts.txt, the golden, the BACKLOG entries; contract §213 through `$MRW mcp` — a write that breaks the check reports it, `check: false` runs none. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestAnMCPWriteRunsTheCheck|TestTheTwoSurfacesCountAWriteAlike' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnMCPWriteRunsTheCheck \(' "$out" \
  && grep -qE '^--- PASS: TestTheTwoSurfacesCountAWriteAlike \(' "$out" \
  && go test ./internal/mcp/ -count=1 -timeout 600s \
  && grep -q '^# 213\. ' scripts/contract.sh \
  && grep -q '^mcp_write check\.exit_code$' docs/receipts.txt \
  && grep -q '^mcp_write drift$' docs/receipts.txt \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnMCPWriteRunsTheCheck` | `internal/mcp/check113_test.go` | a code write whose check fails returns `check.ran` true and a non-zero exit, not `isError`, and counts `failed_check`; `check: false` runs none and counts `applied`; a prose write runs none; an oversized receipt keeps the verdict | none | S1, S2 |
| `TestTheTwoSurfacesCountAWriteAlike` | `cmd/mrw/surfaces113_test.go` | one plan per outcome — prose, no harness, a check passed, failed, timed out, unable to start, unable to run, drift, opted out, an unread line, a dry run, a malformed harness, a ledger failure with a check due, would-refuse pricing — through both surfaces gives the same tally, pricing, hunk statuses, check verdict and drift | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every `mrw_write` call |
| 3 — the caller can discover it | the `check` property in the schema, the description, and the receipt's `check` |
| 4 — it is used | the gap list of 2026-10-01, item 1; telemetry is refused (ADR-009) |

## Mutation Log
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write never runs the check · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/mcp/tools.go` · check: false is ignored · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/mcp/tools.go` · the elision drops the verdict with the tail · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/mcp/tools.go` · the write floor does not carry the verdict phrase · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write never settles: a failed check stays applied · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664

## Invariants

- A write that lands is reported as landed whatever its check did (ADR-102); no answer after a landing is a bare RPC error.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass, beyond the M11 rows this task names.

## Out of Scope

- Steps over MCP (deferred: `docs/adr/BACKLOG.md` "From ADR-113")

## Verification Log
- 2026-10-01 · 95f5dab* · exit 1 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:1905 · test-lock-sha256:739cfff6a5022235e968f33b2228c91b680a574160fe24470e8f2b4a58057809 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdXJmYWNlczExM190ZXN0LmdvCVRlc3RUaGVUd29TdXJmYWNlc0NvdW50QVdyaXRlQWxpa2UJZDE5ODdmZWU2M2Q5YWRlMjFkZDIxODA3YjJlZTFjYmY2Mjg5YzI3Mjk3MzAzZjdiZWUwNjM2ODM4YTRmMjQ0Mgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCVRlc3RBbk1DUFdyaXRlUnVuc1RoZUNoZWNrCTNlMzY5YzIxZmE2MzU2ZjRiYzRiYmRlZWU4MmQxMDYxNWI5MDg1ODZmODQwMzA0ZWM4ZDI0NDdmNmNhNWI0MzkKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNoZWNrIHRoYXQgY2Fubm90IHJ1biBpcyBpc0Vycm9yIHdpdGggdGhlIHJlY2VpcHQJMjExNWIyNzFlMmMzNDhjNjBhOWI3OTcxZjE1NGRlYzY0OWJmMzRmMGY2ZDJjY2UzMDQ1ZWFhYTQzNzJiN2RmMgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgY2hlY2sgdGhhdCBjYW5ub3Qgc3RhcnQgaXMgaXNFcnJvciB3aXRoIHRoZSByZWNlaXB0CWIwMDFkYThlYmIyODQxNTQ5OTk3NWM4MTVlMTlhNGU1MzQ1MmE0NGJlZjc2OWZlODQ2ZmFiMmYxYzM5NmUwMmMKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNvZGUgd3JpdGUgcnVucyB0aGUgY2hlY2sgYW5kIHJlcG9ydHMgaXRzIGZhaWx1cmUJNDIyZjE0Njg3MzYzMDkxNWZiZjZlNTVjYjhkMGY5ODUyOGJmZGM5YWIzZmE0ZWM1YmMzM2M1NjEzZDFjY2NlYgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgbWFsZm9ybWVkIGhhcm5lc3MgcmVmdXNlcyB0aGUgd3JpdGUgYmVmb3JlIGFueXRoaW5nIGxhbmRzCTdlMzRmZjMzNTJlNmU1ODI2NDlmZGM5MTU1ODUxZjE0OTVkOTI4NTAzMzNjOWY1ZDAyNmFmNDI4NDQzYjc4ZGEKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHBhc3NpbmcgY2hlY2sgaXMgcmVwb3J0ZWQgYW5kIHByaWNlZCBhcyBoZWxkCWE3NzdjNTA0YWI2NGE1NWJiYzA2MDRjMTBhZWYyZjA4MGMzMGMxMzY5MzA4MDlmZGFmNDMzNmM4OTE4YmI5YjYKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHByb3NlIHdyaXRlIHJ1bnMgbm9uZQk5MGEzZWRlOTRjYjc0YWRkNWQxZDBmODFlYTkyMDgxMzcwYzMyMGUzZjlkMTU4ZWZkMDhkYjdiYTBmMjFkZDUyCmJvZHkJaW50ZXJuYWwvbWNwL2NoZWNrMTEzX3Rlc3QuZ28JY2hlY2sgZmFsc2UgcnVucyBub25lCWZhMzM0NTMyM2MzODY3NGNjNzlhNmE5ODUwZTEyNTczNmVhNDBhMWU2ZDI4ZDNkMTNkYzUxYzc2NTRiMGQ3MDM
  ```
  --- last 10 line(s) of stdout (of 67 after folding 67 raw)
      surfaces113_test.go:62: ledger_failed_check_due: the surfaces disagree:
           cli: counts applied=0 check_not_run=1 failed_check=0 partially_applied=0 refused_apply=0 refused_parse=0 pricing strict_candidates=1 strict_would_refuse=0 strict_would_refuse_broke=0 strict_would_refuse_held=0 strict_would_refuse_unchecked=0 hunks [ok] check none drift false
           mcp: counts applied=1 check_not_run=0 failed_check=0 partially_applied=0 refused_apply=0 refused_parse=0 pricing strict_candidates=1 strict_would_refuse=0 strict_would_refuse_broke=0 strict_would_refuse_held=0 strict_would_refuse_unchecked=0 hunks [ok] check none drift false
      surfaces113_test.go:62: pricing_would_refuse_broke: the surfaces disagree:
           cli: counts applied=0 check_not_run=0 failed_check=1 partially_applied=0 refused_apply=0 refused_parse=0 pricing strict_candidates=1 strict_would_refuse=1 strict_would_refuse_broke=1 strict_would_refuse_held=0 strict_would_refuse_unchecked=0 hunks [ok] check ran=true exit=3 drift false
           mcp: counts applied=1 check_not_run=0 failed_check=0 partially_applied=0 refused_apply=0 refused_parse=0 pricing strict_candidates=1 strict_would_refuse=1 strict_would_refuse_broke=0 strict_would_refuse_held=0 strict_would_refuse_unchecked=1 hunks [ok] check none drift false
  --- FAIL: TestTheTwoSurfacesCountAWriteAlike (1.37s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	1.482s
  FAIL
  ```
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:21146
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:21166
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:22329
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:22053
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:26347
- 2026-10-01 · 95f5dab* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:0 · test-lock-sha256:b9769c71960b67ae15640674ea180607688e67b2bcbe7bf527fd60461f5f61cb · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdXJmYWNlczExM190ZXN0LmdvCVRlc3RUaGVUd29TdXJmYWNlc0NvdW50QVdyaXRlQWxpa2UJZDE5ODdmZWU2M2Q5YWRlMjFkZDIxODA3YjJlZTFjYmY2Mjg5YzI3Mjk3MzAzZjdiZWUwNjM2ODM4YTRmMjQ0Mgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCVRlc3RBbk1DUFdyaXRlUnVuc1RoZUNoZWNrCWQxMmQwOWU2NmI0MzllOGFmMGE0NGQ5ZDYwMWFhMzBiMDMzYjJkNGU1MWQyMDdmY2MwNzQzMWU1N2ZlMmIwOWYKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNoZWNrIHRoYXQgY2Fubm90IHJ1biBpcyBpc0Vycm9yIHdpdGggdGhlIHJlY2VpcHQJMjExNWIyNzFlMmMzNDhjNjBhOWI3OTcxZjE1NGRlYzY0OWJmMzRmMGY2ZDJjY2UzMDQ1ZWFhYTQzNzJiN2RmMgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgY2hlY2sgdGhhdCBjYW5ub3Qgc3RhcnQgaXMgaXNFcnJvciB3aXRoIHRoZSByZWNlaXB0CWIwMDFkYThlYmIyODQxNTQ5OTk3NWM4MTVlMTlhNGU1MzQ1MmE0NGJlZjc2OWZlODQ2ZmFiMmYxYzM5NmUwMmMKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNvZGUgd3JpdGUgcnVucyB0aGUgY2hlY2sgYW5kIHJlcG9ydHMgaXRzIGZhaWx1cmUJNDIyZjE0Njg3MzYzMDkxNWZiZjZlNTVjYjhkMGY5ODUyOGJmZGM5YWIzZmE0ZWM1YmMzM2M1NjEzZDFjY2NlYgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgbWFsZm9ybWVkIGhhcm5lc3MgcmVmdXNlcyB0aGUgd3JpdGUgYmVmb3JlIGFueXRoaW5nIGxhbmRzCTdlMzRmZjMzNTJlNmU1ODI2NDlmZGM5MTU1ODUxZjE0OTVkOTI4NTAzMzNjOWY1ZDAyNmFmNDI4NDQzYjc4ZGEKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHBhc3NpbmcgY2hlY2sgaXMgcmVwb3J0ZWQgYW5kIHByaWNlZCBhcyBoZWxkCWE3NzdjNTA0YWI2NGE1NWJiYzA2MDRjMTBhZWYyZjA4MGMzMGMxMzY5MzA4MDlmZGFmNDMzNmM4OTE4YmI5YjYKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHByb3NlIHdyaXRlIHJ1bnMgbm9uZQk5MGEzZWRlOTRjYjc0YWRkNWQxZDBmODFlYTkyMDgxMzcwYzMyMGUzZjlkMTU4ZWZkMDhkYjdiYTBmMjFkZDUyCmJvZHkJaW50ZXJuYWwvbWNwL2NoZWNrMTEzX3Rlc3QuZ28JY2hlY2sgZmFsc2UgcnVucyBub25lCWZhMzM0NTMyM2MzODY3NGNjNzlhNmE5ODUwZTEyNTczNmVhNDBhMWU2ZDI4ZDNkMTNkYzUxYzc2NTRiMGQ3MDMKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlldmVyeSB2ZXJkaWN0IHBocmFzZSBmaXRzIHRoZSB3cml0ZSBmbG9vcgk1MDA3ZGY5NTg3MzZjNWZiZmZmYTE3YTJhMjZlZTVmMDdmOGM1MDQ3ODZiZTg5ZWM4YjJkMDNmMDZiODA1YTAz · test-lock-kind:replace
- 2026-10-01 · 0ee1f95* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:0 · test-lock-sha256:fdf3881d7b36da1811382c19abb8552d41c28b2d75a75e41b6b9581657e50dbe · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdXJmYWNlczExM190ZXN0LmdvCVRlc3RUaGVUd29TdXJmYWNlc0NvdW50QVdyaXRlQWxpa2UJZDE5ODdmZWU2M2Q5YWRlMjFkZDIxODA3YjJlZTFjYmY2Mjg5YzI3Mjk3MzAzZjdiZWUwNjM2ODM4YTRmMjQ0Mgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCVRlc3RBbk1DUFdyaXRlUnVuc1RoZUNoZWNrCTRjYTE4NzVlYzVkMmZhNjQ4MWViMGEwYWMzNWQ4MzM3ZTRiMzRkM2U0YjI4NzkyYjJkZmJhMTg5NmY2YWZiZGIKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNoZWNrIHRoYXQgY2Fubm90IHJ1biBpcyBpc0Vycm9yIHdpdGggdGhlIHJlY2VpcHQJMjExNWIyNzFlMmMzNDhjNjBhOWI3OTcxZjE1NGRlYzY0OWJmMzRmMGY2ZDJjY2UzMDQ1ZWFhYTQzNzJiN2RmMgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgY2hlY2sgdGhhdCBjYW5ub3Qgc3RhcnQgaXMgaXNFcnJvciB3aXRoIHRoZSByZWNlaXB0CWIwMDFkYThlYmIyODQxNTQ5OTk3NWM4MTVlMTlhNGU1MzQ1MmE0NGJlZjc2OWZlODQ2ZmFiMmYxYzM5NmUwMmMKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNvZGUgd3JpdGUgcnVucyB0aGUgY2hlY2sgYW5kIHJlcG9ydHMgaXRzIGZhaWx1cmUJNDIyZjE0Njg3MzYzMDkxNWZiZjZlNTVjYjhkMGY5ODUyOGJmZGM5YWIzZmE0ZWM1YmMzM2M1NjEzZDFjY2NlYgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgbWFsZm9ybWVkIGhhcm5lc3MgcmVmdXNlcyB0aGUgd3JpdGUgYmVmb3JlIGFueXRoaW5nIGxhbmRzCTdlMzRmZjMzNTJlNmU1ODI2NDlmZGM5MTU1ODUxZjE0OTVkOTI4NTAzMzNjOWY1ZDAyNmFmNDI4NDQzYjc4ZGEKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHBhc3NpbmcgY2hlY2sgaXMgcmVwb3J0ZWQgYW5kIHByaWNlZCBhcyBoZWxkCWE3NzdjNTA0YWI2NGE1NWJiYzA2MDRjMTBhZWYyZjA4MGMzMGMxMzY5MzA4MDlmZGFmNDMzNmM4OTE4YmI5YjYKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHByb3NlIHdyaXRlIHJ1bnMgbm9uZQk5MGEzZWRlOTRjYjc0YWRkNWQxZDBmODFlYTkyMDgxMzcwYzMyMGUzZjlkMTU4ZWZkMDhkYjdiYTBmMjFkZDUyCmJvZHkJaW50ZXJuYWwvbWNwL2NoZWNrMTEzX3Rlc3QuZ28JY2hlY2sgZmFsc2UgcnVucyBub25lCWZhMzM0NTMyM2MzODY3NGNjNzlhNmE5ODUwZTEyNTczNmVhNDBhMWU2ZDI4ZDNkMTNkYzUxYzc2NTRiMGQ3MDMKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlldmVyeSB2ZXJkaWN0IHBocmFzZSBmaXRzIHRoZSB3cml0ZSBmbG9vcgk1MDA3ZGY5NTg3MzZjNWZiZmZmYTE3YTJhMjZlZTVmMDdmOGM1MDQ3ODZiZTg5ZWM4YjJkMDNmMDZiODA1YTAz · test-lock-kind:replace
- 2026-10-01 · 0ee1f95* · exit 0 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:20347
- 2026-10-02 · 6967a22* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:0 · test-lock-sha256:b1ffed5dc1db218eac95edc73ab11ba974aabc8f944e5116bf2e0d00e0cf6813 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdXJmYWNlczExM190ZXN0LmdvCVRlc3RUaGVUd29TdXJmYWNlc0NvdW50QVdyaXRlQWxpa2UJZDE5ODdmZWU2M2Q5YWRlMjFkZDIxODA3YjJlZTFjYmY2Mjg5YzI3Mjk3MzAzZjdiZWUwNjM2ODM4YTRmMjQ0Mgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCVRlc3RBbk1DUFdyaXRlUnVuc1RoZUNoZWNrCTg0N2JhYmMxZmQyN2U4ZmIxYzBhNWM0NTcyMjU3NzE3ZWE5MDliYmNmNGY5ZTRjNjhkYjE0ZTRmNmQzMDc4MDYKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNoZWNrIHRoYXQgY2Fubm90IHJ1biBpcyBpc0Vycm9yIHdpdGggdGhlIHJlY2VpcHQJMjExNWIyNzFlMmMzNDhjNjBhOWI3OTcxZjE1NGRlYzY0OWJmMzRmMGY2ZDJjY2UzMDQ1ZWFhYTQzNzJiN2RmMgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgY2hlY2sgdGhhdCBjYW5ub3Qgc3RhcnQgaXMgaXNFcnJvciB3aXRoIHRoZSByZWNlaXB0CWIwMDFkYThlYmIyODQxNTQ5OTk3NWM4MTVlMTlhNGU1MzQ1MmE0NGJlZjc2OWZlODQ2ZmFiMmYxYzM5NmUwMmMKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIGNvZGUgd3JpdGUgcnVucyB0aGUgY2hlY2sgYW5kIHJlcG9ydHMgaXRzIGZhaWx1cmUJNDIyZjE0Njg3MzYzMDkxNWZiZjZlNTVjYjhkMGY5ODUyOGJmZGM5YWIzZmE0ZWM1YmMzM2M1NjEzZDFjY2NlYgpib2R5CWludGVybmFsL21jcC9jaGVjazExM190ZXN0LmdvCWEgbWFsZm9ybWVkIGhhcm5lc3MgcmVmdXNlcyB0aGUgd3JpdGUgYmVmb3JlIGFueXRoaW5nIGxhbmRzCTdlMzRmZjMzNTJlNmU1ODI2NDlmZGM5MTU1ODUxZjE0OTVkOTI4NTAzMzNjOWY1ZDAyNmFmNDI4NDQzYjc4ZGEKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHBhc3NpbmcgY2hlY2sgaXMgcmVwb3J0ZWQgYW5kIHByaWNlZCBhcyBoZWxkCWE3NzdjNTA0YWI2NGE1NWJiYzA2MDRjMTBhZWYyZjA4MGMzMGMxMzY5MzA4MDlmZGFmNDMzNmM4OTE4YmI5YjYKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlhIHByb3NlIHdyaXRlIHJ1bnMgbm9uZQk5MGEzZWRlOTRjYjc0YWRkNWQxZDBmODFlYTkyMDgxMzcwYzMyMGUzZjlkMTU4ZWZkMDhkYjdiYTBmMjFkZDUyCmJvZHkJaW50ZXJuYWwvbWNwL2NoZWNrMTEzX3Rlc3QuZ28JY2hlY2sgZmFsc2UgcnVucyBub25lCWZhMzM0NTMyM2MzODY3NGNjNzlhNmE5ODUwZTEyNTczNmVhNDBhMWU2ZDI4ZDNkMTNkYzUxYzc2NTRiMGQ3MDMKYm9keQlpbnRlcm5hbC9tY3AvY2hlY2sxMTNfdGVzdC5nbwlldmVyeSB2ZXJkaWN0IHBocmFzZSBmaXRzIHRoZSB3cml0ZSBmbG9vcgk1MDA3ZGY5NTg3MzZjNWZiZmZmYTE3YTJhMjZlZTVmMDdmOGM1MDQ3ODZiZTg5ZWM4YjJkMDNmMDZiODA1YTAz · test-lock-kind:replace
- 2026-10-02 · 6967a22* · exit 0 · `set -o pipefail …` · acceptance-sha256:e4330044fb178d57eee8c55627fdccc9a25902b1558a114f058f98ce917fa664 · ms:26944
