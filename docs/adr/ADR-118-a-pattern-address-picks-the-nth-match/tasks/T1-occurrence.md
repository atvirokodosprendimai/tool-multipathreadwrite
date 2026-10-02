# Task ADR-118-T1: occurrence=N picks the Nth start match, every earlier match served

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `plan.Hunk.Occurrence`, `apply.Input.Occurrence`, `refusal.OccurrenceAddress`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `occurrence=N picks the Nth start match, every earlier match served`

## Goal

A plan hunk with a start pattern and `occurrence=N` resolves to the Nth matching line of the original file, refused when N is past the count, when any earlier match was not served, or when the address is not a pattern.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | parse `occurrence=`; refuse it off a pattern; the unknown-option text |
| `internal/refusal/refusal.go` | edit | `OccurrenceAddress` |
| `internal/apply/apply.go` | edit | `Input`/`hunk` field; the engine mirror; pick `at[N-1]`; the earlier-matches ledger rule; the several-matches remedy |
| `internal/seen/seen.go`, `internal/seen/seen_test.go` | edit | `Written`/`Shown`, `ServedLine`, merge, the `w` record, v4 |
| `internal/writer/writer.go` | edit | a write records `Written` |
| `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/curve/score.go`, `internal/ingest/applypatch_test.go`, `internal/mcp/tools_test.go` | edit | carry the field |
| `internal/apply/occurrence118_test.go`, `internal/plan/occurrence118_test.go` | add | the tests |
| `internal/guide/guide.go`, `internal/mcp/mcp.go`, `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | say so; close the ADR-013 deferral |
| `scripts/contract.sh` | edit | §218 |

## Ordered Steps

1. [S1] Write the tests: `TestAnOccurrencePicksTheNthStartMatch` (the second of three matches changes alone), `TestAnOccurrencePastTheMatchCountIsRefused`, `TestAnOccurrenceWhoseEarlierMatchesWereNotServedIsRefused` (and its pair: after a read of the earlier matches it applies), `TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine`, `TestTheSeveralMatchesRefusalNamesOccurrence`, `TestEveryBuilderCarriesOccurrence`. Confirm RED. [proof: mutation]
2. [S2] The parser, the engine, the builders. Mutants: `at[0]` for `at[N-1]`; the earlier-matches rule dropped; the parser refusal dropped (the engine mirror must still refuse); a builder that drops the field. [proof: mutation]
3. [S3] Docs and contract §218 (three `^func X` lines: `occurrence=2` after a read applies and changes only the second; the pairs: `occurrence=4` exits 1 with the file unchanged, and `occurrence=` on a line address exits 1). [proof: acceptance]
4. [S4] What the review of #320 found: right after mrw wrote a file, the earlier-matches rule was skipped — a write records the file whole, and the rule exempted a whole observation — so `occurrence=3` applied with none of the matches ever served. Zy chose "strict: always served": `seen.Observation` records whether a whole came from a write (`Written`) and the lines read since (`Shown`), merged on each read and kept by a write of the same version; the rule asks `ServedLine`; the ledger moves to v4. Also: the refusal named "line(s) lines"; the receipt and dry run did not show which occurrence; the engine mirror ran after other refusals; the occurrence refusal offered MCP's ack remedy where an ack could not help; `occurrence=+2` parsed. Mutants: a written whole counted as served; the merge dropping lines shown after a write. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ ./internal/plan/ -count=1 -timeout 600s -run 'Occurrence' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnOccurrencePicksTheNthStartMatch \(' "$out" \
  && grep -qE '^--- PASS: TestAnOccurrenceWhoseEarlierMatchesWereNotServedIsRefused \(' "$out" \
  && grep -qE '^--- PASS: TestEveryBuilderCarriesOccurrence \(' "$out" \
  && grep -qE '^--- PASS: TestAnOccurrenceAfterAWriteNeedsTheMatchesRead \(' "$out" \
  && go test ./internal/apply/ ./internal/plan/ ./internal/seen/ ./internal/writer/ ./internal/mcp/ ./cmd/mrw/ ./internal/curve/ -count=1 -timeout 900s \
  && grep -q '^# 218\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnOccurrencePicksTheNthStartMatch` | `internal/apply/occurrence118_test.go` | occurrence=2 of three matches changes the second alone, with the end pattern as the first match at or after it | none | S1, S2 |
| `TestAnOccurrencePastTheMatchCountIsRefused` | `internal/apply/occurrence118_test.go` | N past the count is refused, naming the matches | none | S1, S2 |
| `TestAnOccurrenceWhoseEarlierMatchesWereNotServedIsRefused` | `internal/apply/occurrence118_test.go` | an unread earlier match refuses the hunk, naming it; served, it applies | none | S1, S2 |
| `TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine` | `internal/plan/occurrence118_test.go` | the parser and the engine both refuse occurrence= off a pattern, with one kind | none | S1, S2 |
| `TestTheSeveralMatchesRefusalNamesOccurrence` | `internal/apply/occurrence118_test.go` | the refusal names occurrence=N as a remedy | none | S1, S2 |
| `TestEveryBuilderCarriesOccurrence` | `internal/plan/occurrence118_test.go` | the CLI, MCP and curve builders carry the field (source scan) | none | S1, S2 |
| `TestAnOccurrenceAfterAWriteNeedsTheMatchesRead` | `internal/apply/occurrence118_test.go` | right after a write the earlier matches still need a read; read, it applies | none | S4 |
| `TestAWrittenFileIsLicensedButNotShown` | `internal/seen/seen_test.go` | a write licenses every line and shows none; reads add shown lines; a whole read shows all; the `w` record round-trips | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `apply.Input.Occurrence` |
| 2 — something selects it | `resolve` picks `at[N-1]` |
| 3 — the caller can discover it | the several-matches refusal and the guide name it |
| 4 — it is used | the gap list of 2026-10-02, item 4b; the assessment of 2026-10-02 |

## Mutation Log
- 2026-10-02 · 269dbc2* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: at[0] for at[N-1] · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39
- 2026-10-02 · 269dbc2* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the earlier-matches rule dropped · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39
- 2026-10-02 · 269dbc2* · mutant killed · exit 1 · `internal/plan/plan.go` · S2: the parser refusal dropped · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39
- 2026-10-02 · 269dbc2* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the CLI builder drops the field · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39
- 2026-10-02 · 3ede136* · mutant killed · exit 1 · `internal/seen/seen.go` · S4: a written whole counted as served · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1
- 2026-10-02 · 3ede136* · mutant survived · exit 0 · `internal/seen/seen.go` · S4: the merge drops lines shown after a write · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-02 · 3ede136* · mutant survived · exit 0 · `internal/seen/seen.go` · S4: the merge drops lines shown after a write · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-02 · 3ede136* · mutant killed · exit 1 · `internal/seen/seen.go` · S4: a written whole counted as served · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1
- 2026-10-02 · 3ede136* · mutant killed · exit 1 · `internal/seen/seen.go` · S4: the merge drops lines shown after a write · acceptance-sha256:8292a2850f8167c98bd8270df265f934ed7ba3428306c048a2095952b7a5429a
- 2026-10-02 · 3ede136* · mutant killed · exit 1 · `internal/seen/seen.go` · S4: a written whole counted as served · acceptance-sha256:8292a2850f8167c98bd8270df265f934ed7ba3428306c048a2095952b7a5429a

## Invariants

- A start pattern without `occurrence=` still matches exactly once or is refused.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the earlier-matches rule cannot be checked with the ledger the engine already holds.

## Out of Scope

- `occurrence=` on a read address (permanent: boundary: a read already serves every match)

## Verification Log
- 2026-10-02 · 269dbc2* · exit 1 · `set -o pipefail …` · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39 · ms:731 · test-lock-sha256:56e02b637d347b66e40f8c0f7bd1694537b72e2170b3fe520a2878c3c4faa32a · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvb2NjdXJyZW5jZTExOF90ZXN0LmdvCVRlc3RBbk9jY3VycmVuY2VQYXN0VGhlTWF0Y2hDb3VudElzUmVmdXNlZAkwZDJkMzlkM2IxMzVjMzQwYWM0ODRhM2JhMjY5NTc3NDM0ZWRjMjQ1MDExYTk5YzEwYzBkMTI0MmQ2OWQwY2ZlCmJvZHkJaW50ZXJuYWwvYXBwbHkvb2NjdXJyZW5jZTExOF90ZXN0LmdvCVRlc3RBbk9jY3VycmVuY2VQaWNrc1RoZU50aFN0YXJ0TWF0Y2gJODAyODVhNDY4ODg3NGI0YTViNGNiODYzYWViMzRmYjI5YzNhOWY1MDZkNmY5OGE3YmE1ZDQwOThjZmI5Yjk2Nwpib2R5CWludGVybmFsL2FwcGx5L29jY3VycmVuY2UxMThfdGVzdC5nbwlUZXN0QW5PY2N1cnJlbmNlV2hvc2VFYXJsaWVyTWF0Y2hlc1dlcmVOb3RTZXJ2ZWRJc1JlZnVzZWQJM2JjMTA1MGY0OTI1MGI5NTZlYWVkMjcyNmRmZDE3YjNhNzg0MDExNmM1MDA0NWIzYWRkZjJhYmM1ZjZmMmRhYQpib2R5CWludGVybmFsL2FwcGx5L29jY3VycmVuY2UxMThfdGVzdC5nbwlUZXN0VGhlU2V2ZXJhbE1hdGNoZXNSZWZ1c2FsTmFtZXNPY2N1cnJlbmNlCWZmZDY5N2ZmZGZlOGMzNzJmOGRiMzBkYjUzYTMwYzJmN2MxMmU5MTMzZjFhODlkODgxMGI3MGFiMTQwOThjZWIKYm9keQlpbnRlcm5hbC9wbGFuL29jY3VycmVuY2UxMThfdGVzdC5nbwlUZXN0QW5PY2N1cnJlbmNlT25BTGluZUFkZHJlc3NJc1JlZnVzZWRCeVBhcnNlckFuZEVuZ2luZQk0YjIyMTVkYzM4ZmRmMmQ1OWQyOTIyMmJhN2VkY2Q0OWJlYzk4NjIxN2U3Y2U3MWEyY2JmMzBmMGI0MzRmMTBmCmJvZHkJaW50ZXJuYWwvcGxhbi9vY2N1cnJlbmNlMTE4X3Rlc3QuZ28JVGVzdEV2ZXJ5QnVpbGRlckNhcnJpZXNPY2N1cnJlbmNlCTY1ZTRjYmU2ZmY5Y2IxNDBmYjE4OTE2MThiOTU5MjEyNDZkMDVhYTBkZjNhNjc1MzI2MjFlYjcxOWM2MmE5MDg
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan_test [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan.test]
  internal/plan/occurrence118_test.go:27:75: undefined: refusal.OccurrenceAddress
  internal/plan/occurrence118_test.go:28:78: undefined: refusal.OccurrenceAddress
  internal/plan/occurrence118_test.go:32:39: h[0].Occurrence undefined (type plan.Hunk has no field or method Occurrence)
  internal/plan/occurrence118_test.go:44:126: unknown field Occurrence in struct literal of type apply.Input
  internal/plan/occurrence118_test.go:48:53: undefined: refusal.OccurrenceAddress
  internal/plan/occurrence118_test.go:49:89: undefined: refusal.OccurrenceAddress
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan [build failed]
  FAIL
  ```
- 2026-10-02 · 269dbc2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39 · ms:39483
- 2026-10-02 · 269dbc2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39 · ms:39041
- 2026-10-02 · 269dbc2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39 · ms:40067
- 2026-10-02 · 269dbc2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5f62f6fc579bcb76718adcc15e73f3f35bf5ef121c2ef23e0fc437e274194d39 · ms:39186
- 2026-10-02 · 3ede136* · exit 2 · `set -o pipefail …` · acceptance-sha256:f2a25d9eb5b96560d4d2ded06dc079cadfcc5ada7993b3e035b4ffa1da90ecb4 · ms:1252
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
  === RUN   TestAnOccurrenceUnderForceAndTwiceInOnePlan
  --- PASS: TestAnOccurrenceUnderForceAndTwiceInOnePlan (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.189s
  === RUN   TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine
  --- PASS: TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine (0.00s)
  === RUN   TestEveryBuilderCarriesOccurrence
  --- PASS: TestEveryBuilderCarriesOccurrence (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.184s
  --- last 3 line(s) of stderr
  grep: \: No such file or directory
  bash: -c: line 7: syntax error near unexpected token `&&'
  bash: -c: line 7: `  && go test ./internal/apply/ ./internal/plan/ ./internal/mcp/ ./cmd/mrw/ ./internal/curve/ -count=1 -timeout 900s \'
  ```
- 2026-10-02 · 3ede136* · exit 2 · `set -o pipefail …` · acceptance-sha256:f2a25d9eb5b96560d4d2ded06dc079cadfcc5ada7993b3e035b4ffa1da90ecb4 · ms:886
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
  === RUN   TestAnOccurrenceUnderForceAndTwiceInOnePlan
  --- PASS: TestAnOccurrenceUnderForceAndTwiceInOnePlan (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.189s
  === RUN   TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine
  --- PASS: TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine (0.00s)
  === RUN   TestEveryBuilderCarriesOccurrence
  --- PASS: TestEveryBuilderCarriesOccurrence (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.178s
  --- last 3 line(s) of stderr
  grep: \: No such file or directory
  bash: -c: line 7: syntax error near unexpected token `&&'
  bash: -c: line 7: `  && go test ./internal/apply/ ./internal/plan/ ./internal/mcp/ ./cmd/mrw/ ./internal/curve/ -count=1 -timeout 900s \'
  ```
- 2026-10-02 · 3ede136* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1 · ms:43614
- 2026-10-02 · 3ede136* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1 · ms:41032
- 2026-10-02 · 3ede136* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1 · ms:44325
- 2026-10-02 · 3ede136* · exit 0 · `set -o pipefail …` · acceptance-sha256:fb5fe1b64f1b4f951833ea5763845816087beaaebeb3dd4802de194be8f638e1 · ms:43094
- 2026-10-02 · 3ede136* · exit 0 · `set -o pipefail …` · acceptance-sha256:8292a2850f8167c98bd8270df265f934ed7ba3428306c048a2095952b7a5429a · ms:42009
- 2026-10-02 · 3ede136* · exit 0 · `set -o pipefail …` · acceptance-sha256:8292a2850f8167c98bd8270df265f934ed7ba3428306c048a2095952b7a5429a · ms:47981
- 2026-10-02 · 3ede136* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:8292a2850f8167c98bd8270df265f934ed7ba3428306c048a2095952b7a5429a · ms:0 · test-lock-sha256:d532d24e38ea6402c9d6a7f36121f1b7b9ca13406feb770c1d59c66f3808768a · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvb2NjdXJyZW5jZTExOF90ZXN0LmdvCVRlc3RBbk9jY3VycmVuY2VBZnRlckFXcml0ZU5lZWRzVGhlTWF0Y2hlc1JlYWQJMjc1MGE1ZGI3MjM5OTBkMjkwMmE5YjVlMjgzZjE4YmUzYWU4MzIyNmRiYzJhMzRiYjUwNjc4YWU2N2RjNWExNgpib2R5CWludGVybmFsL2FwcGx5L29jY3VycmVuY2UxMThfdGVzdC5nbwlUZXN0QW5PY2N1cnJlbmNlUGFzdFRoZU1hdGNoQ291bnRJc1JlZnVzZWQJMGQyZDM5ZDNiMTM1YzM0MGFjNDg0YTNiYTI2OTU3NzQzNGVkYzI0NTAxMWE5OWMxMGMwZDEyNDJkNjlkMGNmZQpib2R5CWludGVybmFsL2FwcGx5L29jY3VycmVuY2UxMThfdGVzdC5nbwlUZXN0QW5PY2N1cnJlbmNlUGlja3NUaGVOdGhTdGFydE1hdGNoCTgwMjg1YTQ2ODg4NzRiNGE1YjRjYjg2M2FlYjM0ZmIyOWMzYTlmNTA2ZDZmOThhN2JhNWQ0MDk4Y2ZiOWI5NjcKYm9keQlpbnRlcm5hbC9hcHBseS9vY2N1cnJlbmNlMTE4X3Rlc3QuZ28JVGVzdEFuT2NjdXJyZW5jZVVuZGVyRm9yY2VBbmRUd2ljZUluT25lUGxhbgkxOWFhMmQzZTgxZGVjYTgzNzU4Zjk0NGFhNjZhYzA2NWM2NDk2Y2FjYTQyZGYxMzg2Y2EwM2NkMWUxMjBkZjE0CmJvZHkJaW50ZXJuYWwvYXBwbHkvb2NjdXJyZW5jZTExOF90ZXN0LmdvCVRlc3RBbk9jY3VycmVuY2VXaG9zZUVhcmxpZXJNYXRjaGVzV2VyZU5vdFNlcnZlZElzUmVmdXNlZAkzYmMxMDUwZjQ5MjUwYjk1NmVhZWQyNzI2ZGZkMTdiM2E3ODQwMTE2YzUwMDQ1YjNhZGRmMmFiYzVmNmYyZGFhCmJvZHkJaW50ZXJuYWwvYXBwbHkvb2NjdXJyZW5jZTExOF90ZXN0LmdvCVRlc3RUaGVTZXZlcmFsTWF0Y2hlc1JlZnVzYWxOYW1lc09jY3VycmVuY2UJZmZkNjk3ZmZkZmU4YzM3MmY4ZGIzMGRiNTNhMzBjMmY3YzEyZTkxMzNmMWE4OWQ4ODEwYjcwYWIxNDA5OGNlYgpib2R5CWludGVybmFsL3BsYW4vb2NjdXJyZW5jZTExOF90ZXN0LmdvCVRlc3RBbk9jY3VycmVuY2VPbkFMaW5lQWRkcmVzc0lzUmVmdXNlZEJ5UGFyc2VyQW5kRW5naW5lCTI5ZTRlNjFjNzg1YjM5OWM4MDM4MWFmNDhkMDMwZjlmZTA3NDNiNGMwNWFiMWFjMzQ2YTc1NmEyYWMyZDU3YjQKYm9keQlpbnRlcm5hbC9wbGFuL29jY3VycmVuY2UxMThfdGVzdC5nbwlUZXN0RXZlcnlCdWlsZGVyQ2Fycmllc09jY3VycmVuY2UJNjVlNGNiZTZmZjljYjE0MGZiMTg5MTYxOGI5NTkyMTI0NmQwNWFhMGRmM2E2NzUzMjYyMWViNzE5YzYyYTkwOApib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBTGVkZ2VyVGhpc0J1aWxkV3JvdGVMb2Fkc0JhY2sJZDFkNmI4OTRlNTFlZmZhZjAwNjI0NWQ3ZjdjZWExNTY4YTIzNmMyZTRhNWJhYjVjOTc2NDNiM2M2MTY3YTAwMApib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBTGVnYWN5UGF0aENvbnRhaW5pbmdBRG91YmxlU3BhY2VJc09uZVBhdGgJY2I3YjVjY2Q0NGMzMTk0YWY4OTVjMzhlMDlkNjJiODA0M2RmZjExZDcyZTBiNTIyNzU1OTg2MmFiYmEyYWYxNQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBUHJlVjJJblRyZWVMZWRnZXJJc0Rpc2NhcmRlZE5vdFRydXN0ZWQJOTYwY2YyOWViZjZmYTJkMmI5OGViYWZhZTZiMzU0ODc2ODNjMDM3Njc1YzU0ZjQ4YjExOGJmNzc0OTBlYWZlZQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBVjRMZWRnZXJJc0FjY2VwdGVkQW5kVGhlQnVtcElzRGVsaWJlcmF0ZQk4YjhlMzNmOTI4MDFkNmJiYzY0MWEwOThkMmIxMWY4NjkyOGUxMjdmMTM2NTNjMTM3MTQwMDkwYWVkMDZjNDZlCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdEFXcml0dGVuRmlsZUlzTGljZW5zZWRCdXROb3RTaG93bgk3OGY3NjkzYTMwNjQyNTk3YjY3MmIzMTIyNDNmYThkZDQ2NDgxYWIyNjY0ZTE1Y2EyMDJlODJjZjcwYzczYTI4CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdENvbmN1cnJlbnRSZWNvcmRzS2VlcEV2ZXJ5UGF0aAllYjlhNmM5NTAxMmVjNjM1YTRlNzUxOWI3NWM3NjEzNGNiOTdjMDA0Y2VkMGYyMTJmN2I4OWEyM2I0ODY5NTNhCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdExlZGdlcklzTm90V3JpdHRlbkludG9UaGVSb290CTQ0MWU5Zjc4YTRhOTMyYmY2MmI5NmIyMjA2MjI3N2EwMjdlODdiN2RjMjc4ZWVhOGU0ZDhjMjg2YThhMTg0YTYKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TGVnYWN5SW5UcmVlTGVkZ2VySXNTdGlsbFJlYWQJYzRkYThjNzQzMmRhM2IzNTM0N2ZjOGUzMDI4YTNiMTY4ODQ5ZmM1MjM5Zjk5NGE1MzBiOWU4MTgzODU4YzhjNQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RNYWluCTM4M2EzNzQwZWI1YjEzNzI0MDQzZTZmN2RhMWI4YWJiYmU3MTVjYThjYTRjNTIxYzNjNzY1ZDkwMWM5ODJmZDQKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TWlzc2luZ0xlZGdlcklzRW1wdHlOb3RBbkVycm9yCTU4NTczZDg0ZmNhZTVlNTBjYTEwMjMzMDY4ZDM2MjZkYTNhNmUyMWYzOWM1ZjljMTcyMjZkMjVkOTNlYzA1ZTgKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0UmVjb3JkTWVyZ2VzUmF0aGVyVGhhblJlcGxhY2VzCTIzN2I4MjA1ODNmZjliYmI4NmQ4YmYzYjQ1Zjc5NWNmNjVjMmU0YzQzMjcxNGIwYjliZDhhM2NhODYwMTIxYWMKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0UmVjb3JkT3ZlcndyaXRlc1RoZVNhbWVQYXRoCTViYzYwMDY0Y2RmMzE2ZTYxMmQ3Mjg4MDk0MTc5NzRlNGUwNTkzNDc0YzYwYmZiNWIwMmU2MDYzZWMxMmY5YTcKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0Um91bmRUcmlwCTFlMTgyOGYwMmUyMzhlY2ZjMDhmMGY1MmE4NmI2ZGQwZDhlYjgwODY2NjU5YmE0MDBhYmZlZTNmNjU3ODAyMmYKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0U0hBSXNTdGFibGUJODk4ZjljNmMwNTYwMTczZGM0MWI2ZGM1MzJhOTY3MDU2MWNhMjJkMDExN2Y4Y2IwMTUzNWRmNWI3OTQxZWEwZA · test-lock-kind:replace
