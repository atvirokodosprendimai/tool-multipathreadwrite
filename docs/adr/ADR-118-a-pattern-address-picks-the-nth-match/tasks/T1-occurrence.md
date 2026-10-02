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
| `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/curve/score.go`, `internal/ingest/applypatch_test.go`, `internal/mcp/tools_test.go` | edit | carry the field |
| `internal/apply/occurrence118_test.go`, `internal/plan/occurrence118_test.go` | add | the tests |
| `internal/guide/guide.go`, `internal/mcp/mcp.go`, `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | say so; close the ADR-013 deferral |
| `scripts/contract.sh` | edit | §218 |

## Ordered Steps

1. [S1] Write the tests: `TestAnOccurrencePicksTheNthStartMatch` (the second of three matches changes alone), `TestAnOccurrencePastTheMatchCountIsRefused`, `TestAnOccurrenceWhoseEarlierMatchesWereNotServedIsRefused` (and its pair: after a read of the earlier matches it applies), `TestAnOccurrenceOnALineAddressIsRefusedByParserAndEngine`, `TestTheSeveralMatchesRefusalNamesOccurrence`, `TestEveryBuilderCarriesOccurrence`. Confirm RED. [proof: mutation]
2. [S2] The parser, the engine, the builders. Mutants: `at[0]` for `at[N-1]`; the earlier-matches rule dropped; the parser refusal dropped (the engine mirror must still refuse); a builder that drops the field. [proof: mutation]
3. [S3] Docs and contract §218 (three `^func X` lines: `occurrence=2` after a read applies and changes only the second; the pairs: `occurrence=4` exits 1 with the file unchanged, and `occurrence=` on a line address exits 1). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ ./internal/plan/ -count=1 -timeout 600s -run 'Occurrence' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnOccurrencePicksTheNthStartMatch \(' "$out" \
  && grep -qE '^--- PASS: TestAnOccurrenceWhoseEarlierMatchesWereNotServedIsRefused \(' "$out" \
  && grep -qE '^--- PASS: TestEveryBuilderCarriesOccurrence \(' "$out" \
  && go test ./internal/apply/ ./internal/plan/ ./internal/mcp/ ./cmd/mrw/ ./internal/curve/ -count=1 -timeout 900s \
  && grep -q '^# 218\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state internal/lines internal/rooted \
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
