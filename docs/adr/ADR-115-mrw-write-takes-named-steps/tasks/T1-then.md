# Task ADR-115-T1: mrw_write runs declared steps after its check

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `mrw_write` `then`; `then` in its receipt
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `mrw_write runs declared steps after its check`

## Goal

`mrw_write` takes `then` (declared step names), refuses at the depth limit before parsing, runs the steps through the shared sequence after a passing check, returns `then` in its receipt with the isError, elision and floor rules of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `writeArgs.Then`, the depth refusal, `Request.Steps`, `writeReceipt.Then`, the ledger-failure path, isError, step text, elision, `stepPhrase`, `floorAt` |
| `internal/mcp/mcp.go` | edit | the `then` input property; the description |
| `internal/mcp/schema.go` | edit | `then` and `then.*` descriptions; `elided` |
| `internal/mcp/steps115_test.go` | add | `TestAnMCPWriteRunsItsSteps` |
| `internal/mcp/undeclared093_test.go` | edit | `then` is declared now: the rows use `then_sh` |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated |
| `docs/receipts.txt` | edit | `mcp_write then…` |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | say so; close "Steps over MCP" |
| `cmd/opencode/mrw-plugin/src/index.ts`, its smoke test | edit | `then` |
| `scripts/contract.sh` | edit | §215 |
| `internal/writer/flow.go`, `cmd/mrw/main.go` | edit | `Request.StepFlag`: a refused step name is reported under the surface's own name for it |

## Ordered Steps

1. [S1] Write `TestAnMCPWriteRunsItsSteps`: a declared step runs after a passing check (`then.steps[0].status` pass); a failing step is not isError and counts `failed_check`; an undeclared name writes nothing; a step that cannot start is isError with the receipt; a failed check leaves the steps not_run; `MRW_STEP_DEPTH=8` refuses before anything is written; the smallest ceiling keeps the step phrase. Confirm RED. [proof: mutation]
2. [S2] The change. Mutants: steps not passed to the request; the depth refusal dropped; a step that could not start not isError; the floor without the step phrase. [proof: mutation]
3. [S3] Docs, receipts, schema, golden, plugin, contract §215 (a declared step passes; the pair: an undeclared name is refused and the file unchanged). [proof: acceptance]
4. [S4] What breaking it found: an undeclared name over MCP was refused as `--then nosuch: …`, the CLI's flag, which a caller with no shell cannot pass. `Request.StepFlag` names it per surface — `then` over MCP, `--then` on the CLI, whose words and golden are unchanged. Mutant: MCP passes no StepFlag. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 600s -run 'TestAnMCPWriteRunsItsSteps' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnMCPWriteRunsItsSteps \(' "$out" \
  && go test ./internal/mcp/ -count=1 -timeout 600s \
  && grep -q '^# 215\. ' scripts/contract.sh \
  && grep -q '^mcp_write then\.steps\[\]\.status$' docs/receipts.txt \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnMCPWriteRunsItsSteps` | `internal/mcp/steps115_test.go` | a declared step runs and is reported; a failing step is data and counted failed_check; an undeclared name writes nothing; could-not-start is isError with the receipt; a failed check leaves steps not_run; the depth limit refuses first; the smallest ceiling names a stopped step | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every `mrw_write` with `then` |
| 3 — the caller can discover it | the schema's `then`, the description, the receipt |
| 4 — it is used | the gap list of 2026-10-02, item 2 |

## Mutation Log
- 2026-10-02 · 2818b24* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the steps not passed to the request · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf
- 2026-10-02 · 2818b24* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the depth refusal dropped · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf
- 2026-10-02 · 2818b24* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: a step that could not start is not isError · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf
- 2026-10-02 · 2818b24* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the floor without the step phrase · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf
- 2026-10-02 · 2818b24* · mutant killed · exit 1 · `internal/mcp/tools.go` · S4: MCP passes no StepFlag, so the refusal names --then · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf

## Invariants

- A write with no `then` answers exactly as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change beyond the `then` rows of `undeclared093_test.go`.

## Out of Scope

- `then_sh` over MCP (permanent: boundary: the record's Decision 1)

## Verification Log
- 2026-10-02 · 8307f87* · exit 1 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:295 · test-lock-sha256:a2e571044c3ca8a779d6a385fab0cb1a14dba118c000540401ec048f5eff693d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JVGVzdEFuTUNQV3JpdGVSdW5zSXRzU3RlcHMJNjEzNDNkOGVmYThkNjhiZjE4ODY5MGExNDgyN2RkM2FmMjEzODFmN2YzNWFmODk5MWVjOTY1MjU4ZTdiMTVmYQpib2R5CWludGVybmFsL21jcC9zdGVwczExNV90ZXN0LmdvCWEgZGVjbGFyZWQgc3RlcCBydW5zIGFmdGVyIGEgcGFzc2luZyBjaGVjawk3YTc5Y2Q2YWI2NWExZTRiYzM5ZTA2ZDI3ZWRjNDVjNjM1YzFmMzhhNGNhMzJmYjgxZWQ3YzFiMzFhOGI2OTMxCmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JYSBmYWlsZWQgY2hlY2sgbGVhdmVzIHRoZSBzdGVwcyBub3QgcnVuCTZmYzcyNzlkNDg2NThhNmY4MDE4ZjcxMGVhNWYwYmJiYmQ0ODcxMDhlNjk3YzY0YzAwYjQxOTk5NGU0NGY5YTUKYm9keQlpbnRlcm5hbC9tY3Avc3RlcHMxMTVfdGVzdC5nbwlhIGZhaWxpbmcgc3RlcCBpcyBkYXRhIGFuZCBjb3VudGVkIGZhaWxlZF9jaGVjawlhN2UxYTkzNGQ2OWFmYWFlNTg2NWNkNDZmZGQ0ZjUyZTU5NzAwMTllNmQ2YTc2MjVjZjlmZWZhMzQ3MmUxZmQ3CmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JYSBzdGVwIHRoYXQgY2Fubm90IHN0YXJ0IGlzIGlzRXJyb3Igd2l0aCB0aGUgcmVjZWlwdAkwN2YxNTAxYjIwZDM3NzczOWM0YzQzNzgwZTRlYjcwNjUwMzA3OTExMTVmOGNmNWNiYzY5YzI2YmVkM2Y4MjgxCmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JYW4gdW5kZWNsYXJlZCBuYW1lIHdyaXRlcyBub3RoaW5nCTEzYTg3YjgwYmU3YjE3ZTM0MjU1MTJhODk1MWNhZWE2MDYxZjI0YzMzZTVhODJlNTFmMjM5MjNlODkwNmNlNzkKYm9keQlpbnRlcm5hbC9tY3Avc3RlcHMxMTVfdGVzdC5nbwlldmVyeSBzdGVwIHBocmFzZSBmaXRzIHRoZSB3cml0ZSBmbG9vcgk0ODkzOWM0Mjg0MzgzMTBkOTIxZjMyNmZiOTA3Y2U5NzNmZmIzMWMzNzgxNTlmMzFlNjc4NzdlODdhNDgwNDQ0CmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JdGhlIGRlcHRoIGxpbWl0IHJlZnVzZXMgYmVmb3JlIGFueXRoaW5nIGlzIHdyaXR0ZW4JYjZjNTIyZjJhM2NkYmViMTU4OTgxZDM3NGFiYmU4NzNmYWUwMDA0M2Y5N2QyYTc3MWExYmVjNzRiMTliYzZmNw
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/steps115_test.go:96:114: undefined: stepPhrase
  internal/mcp/steps115_test.go:97:64: undefined: stepPhrase
  internal/mcp/steps115_test.go:98:74: undefined: stepPhrase
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-10-02 · 8307f87* · exit 1 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:22437
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_declared_step_runs_after_a_passing_check (0.10s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failing_step_is_data_and_counted_failed_check (0.07s)
      --- PASS: TestAnMCPWriteRunsItsSteps/an_undeclared_name_writes_nothing (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_step_that_cannot_start_is_isError_with_the_receipt (0.03s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failed_check_leaves_the_steps_not_run (0.04s)
      --- PASS: TestAnMCPWriteRunsItsSteps/the_depth_limit_refuses_before_anything_is_written (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/every_step_phrase_fits_the_write_floor (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.507s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	20.704s
  ```
- 2026-10-02 · 8307f87* · exit 1 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:21840
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_declared_step_runs_after_a_passing_check (0.08s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failing_step_is_data_and_counted_failed_check (0.06s)
      --- PASS: TestAnMCPWriteRunsItsSteps/an_undeclared_name_writes_nothing (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_step_that_cannot_start_is_isError_with_the_receipt (0.04s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failed_check_leaves_the_steps_not_run (0.04s)
      --- PASS: TestAnMCPWriteRunsItsSteps/the_depth_limit_refuses_before_anything_is_written (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/every_step_phrase_fits_the_write_floor (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.412s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	20.284s
  ```
- 2026-10-02 · 8307f87* · exit 1 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:22805
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_declared_step_runs_after_a_passing_check (0.10s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failing_step_is_data_and_counted_failed_check (0.07s)
      --- PASS: TestAnMCPWriteRunsItsSteps/an_undeclared_name_writes_nothing (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_step_that_cannot_start_is_isError_with_the_receipt (0.03s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failed_check_leaves_the_steps_not_run (0.04s)
      --- PASS: TestAnMCPWriteRunsItsSteps/the_depth_limit_refuses_before_anything_is_written (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/every_step_phrase_fits_the_write_floor (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.469s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	21.087s
  ```
- 2026-10-02 · 8307f87* · exit 1 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:22155
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_declared_step_runs_after_a_passing_check (0.12s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failing_step_is_data_and_counted_failed_check (0.07s)
      --- PASS: TestAnMCPWriteRunsItsSteps/an_undeclared_name_writes_nothing (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_step_that_cannot_start_is_isError_with_the_receipt (0.04s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failed_check_leaves_the_steps_not_run (0.04s)
      --- PASS: TestAnMCPWriteRunsItsSteps/the_depth_limit_refuses_before_anything_is_written (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/every_step_phrase_fits_the_write_floor (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.474s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	20.682s
  ```
- 2026-10-02 · 8307f87* · exit 1 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:21518
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_declared_step_runs_after_a_passing_check (0.07s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failing_step_is_data_and_counted_failed_check (0.06s)
      --- PASS: TestAnMCPWriteRunsItsSteps/an_undeclared_name_writes_nothing (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_step_that_cannot_start_is_isError_with_the_receipt (0.03s)
      --- PASS: TestAnMCPWriteRunsItsSteps/a_failed_check_leaves_the_steps_not_run (0.04s)
      --- PASS: TestAnMCPWriteRunsItsSteps/the_depth_limit_refuses_before_anything_is_written (0.01s)
      --- PASS: TestAnMCPWriteRunsItsSteps/every_step_phrase_fits_the_write_floor (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.291s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	20.731s
  ```
- 2026-10-02 · 2818b24* · exit 0 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:20917
- 2026-10-02 · 2818b24* · exit 0 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:20885
- 2026-10-02 · 2818b24* · exit 0 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:23304
- 2026-10-02 · 2818b24* · exit 0 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:22517
- 2026-10-02 · 2818b24* · exit 0 · `set -o pipefail …` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:20835
- 2026-10-02 · 2818b24* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:db031f6bc9394cc7750fee28beed683f56e4e2b0b2a788db68c322cb69bd0adf · ms:0 · test-lock-sha256:da794dfaa16cfa718ddfe1073a98939a0714c38913be46cc81e26b1aad172623 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JVGVzdEFuTUNQV3JpdGVSdW5zSXRzU3RlcHMJMDI2YzU3YzJhOTQzMDgyOTUwMWY4ZWMzYjVlMTA3YTg4MTM2M2JmYzE0NDI1NDlmMmFlZTUwNDI0ZTE1ZTBiZApib2R5CWludGVybmFsL21jcC9zdGVwczExNV90ZXN0LmdvCWEgZGVjbGFyZWQgc3RlcCBydW5zIGFmdGVyIGEgcGFzc2luZyBjaGVjawk3YTc5Y2Q2YWI2NWExZTRiYzM5ZTA2ZDI3ZWRjNDVjNjM1YzFmMzhhNGNhMzJmYjgxZWQ3YzFiMzFhOGI2OTMxCmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JYSBmYWlsZWQgY2hlY2sgbGVhdmVzIHRoZSBzdGVwcyBub3QgcnVuCTZmYzcyNzlkNDg2NThhNmY4MDE4ZjcxMGVhNWYwYmJiYmQ0ODcxMDhlNjk3YzY0YzAwYjQxOTk5NGU0NGY5YTUKYm9keQlpbnRlcm5hbC9tY3Avc3RlcHMxMTVfdGVzdC5nbwlhIGZhaWxpbmcgc3RlcCBpcyBkYXRhIGFuZCBjb3VudGVkIGZhaWxlZF9jaGVjawlhN2UxYTkzNGQ2OWFmYWFlNTg2NWNkNDZmZGQ0ZjUyZTU5NzAwMTllNmQ2YTc2MjVjZjlmZWZhMzQ3MmUxZmQ3CmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JYSBzdGVwIHRoYXQgY2Fubm90IHN0YXJ0IGlzIGlzRXJyb3Igd2l0aCB0aGUgcmVjZWlwdAkwN2YxNTAxYjIwZDM3NzczOWM0YzQzNzgwZTRlYjcwNjUwMzA3OTExMTVmOGNmNWNiYzY5YzI2YmVkM2Y4MjgxCmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JYW4gdW5kZWNsYXJlZCBuYW1lIHdyaXRlcyBub3RoaW5nCTc0ODE0ZDZiMjc2YzkxN2VhNmE2YTJhMzA1YjY5YjRmNTZhMTVkOTk2NjAzN2U1ODZiYjkzZGI4YzQ1ZTg2N2YKYm9keQlpbnRlcm5hbC9tY3Avc3RlcHMxMTVfdGVzdC5nbwlldmVyeSBzdGVwIHBocmFzZSBmaXRzIHRoZSB3cml0ZSBmbG9vcgk0ODkzOWM0Mjg0MzgzMTBkOTIxZjMyNmZiOTA3Y2U5NzNmZmIzMWMzNzgxNTlmMzFlNjc4NzdlODdhNDgwNDQ0CmJvZHkJaW50ZXJuYWwvbWNwL3N0ZXBzMTE1X3Rlc3QuZ28JdGhlIGRlcHRoIGxpbWl0IHJlZnVzZXMgYmVmb3JlIGFueXRoaW5nIGlzIHdyaXR0ZW4JYjZjNTIyZjJhM2NkYmViMTU4OTgxZDM3NGFiYmU4NzNmYWUwMDA0M2Y5N2QyYTc3MWExYmVjNzRiMTliYzZmNw · test-lock-kind:replace
