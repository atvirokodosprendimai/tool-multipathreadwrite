# Task ADR-072-T1: The harness is read before anything is written

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the early `check.Load` in `write`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the harness is read before apply`, `no-check and dry-run skip it`, `a contract row drives the binary`, `the engine packages are unchanged`

## Goal

A malformed `.quality-harness.json` applied the write and then exited 2 with only the JSON error: `check.Load` ran after `apply.Apply` committed. Read it first; refuse before anything is written.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `check.Load` before `apply.Apply`; the late load removed |
| `cmd/mrw/harness_before_apply_test.go` | new | the refusal, prose included, and the two flags that skip it |
| `docs/adr/ADR-059-honour-fencetimeout.md` | edit | its Out of Scope line marked invalidated |
| `scripts/contract.sh` | edit | §142 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN; the existing write and check tests stay green. [proof: mutation]
   Mutants: the load moved back after apply; the dry-run guard dropped; the no-check guard dropped.
3. [S3] The contract row drives the built binary with the good case and the one that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 180s -run 'TestAMalformedHarness|TestAMarkdownOnlyPlanIsAlsoRefused|TestNoCheckSkipsTheHarness|TestDryRunDoesNotLoadTheHarness' -v 2>&1 | tee /tmp/adr072-T1.out \
  && missing=$(for t in TestAMalformedHarnessRefusesTheWriteBeforeAnythingIsWritten TestAMarkdownOnlyPlanIsAlsoRefusedByABadHarness TestNoCheckSkipsTheHarnessEntirely TestDryRunDoesNotLoadTheHarness; do grep -qE "^--- PASS: $t \(" /tmp/adr072-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 142\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMalformedHarnessRefusesTheWriteBeforeAnythingIsWritten` | `cmd/mrw/harness_before_apply_test.go` | exit 2, the harness named, `a.go` unchanged | — | S1, S2 |
| `TestAMarkdownOnlyPlanIsAlsoRefusedByABadHarness` | `cmd/mrw/harness_before_apply_test.go` | a prose plan is refused the same way | — | S1, S2 |
| `TestNoCheckSkipsTheHarnessEntirely` | `cmd/mrw/harness_before_apply_test.go` | the pair: `--no-check` applies | — | S1, S2 |
| `TestDryRunDoesNotLoadTheHarness` | `cmd/mrw/harness_before_apply_test.go` | the pair: `--dry-run` validates | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code and its tests |
| 2 — something selects it | every `mrw write`; `mrw check` for T4 |
| 3 — the caller can discover it | the receipt, the exit code and the refusal text |
| 4 — it is used | the v1.25.1 round reproduced each defect with the shipped binary |

## Verification Log
(empty until execute)
- 2026-09-25 · e3f7978* · exit 1 · `set -o pipefail …` · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · ms:1137 · test-lock-sha256:c168472451566b12bce1671b4de270471507db0aabe05e01c1a87b63a1920fd7 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvaGFybmVzc19iZWZvcmVfYXBwbHlfdGVzdC5nbwlUZXN0QU1hbGZvcm1lZEhhcm5lc3NSZWZ1c2VzVGhlV3JpdGVCZWZvcmVBbnl0aGluZ0lzV3JpdHRlbgk2M2QzMjJkNTg0MTZhZmQ4NDYzNDFmMTZiMjhjZTdmYWEzODAwMGFhNjhkZmRlMWYxMTVlMWEzNTNjODExOTgzCmJvZHkJY21kL21ydy9oYXJuZXNzX2JlZm9yZV9hcHBseV90ZXN0LmdvCVRlc3RBTWFya2Rvd25Pbmx5UGxhbklzQWxzb1JlZnVzZWRCeUFCYWRIYXJuZXNzCTQ1OTkzNjAzNjRiZTMyNzY4OGZmNmM4YmUwNmU3NzQ4MTg2ZDQyN2NkNDYzODA5MmYyYWMxZDkyYTdkMjU3ZjcKYm9keQljbWQvbXJ3L2hhcm5lc3NfYmVmb3JlX2FwcGx5X3Rlc3QuZ28JVGVzdERyeVJ1bkRvZXNOb3RMb2FkVGhlSGFybmVzcwk0NzY2MDExYTc4MTVmNWQ2ZTMzNWM2YWU5MTY0YWI1NzY2NTc2NWE3MjRkNDE2ODhjYzE5MDE4OTU3MDFjYTYwCmJvZHkJY21kL21ydy9oYXJuZXNzX2JlZm9yZV9hcHBseV90ZXN0LmdvCVRlc3ROb0NoZWNrU2tpcHNUaGVIYXJuZXNzRW50aXJlbHkJZTE4ODM1MDkxN2JhNWMzZTcwMGVjZjQ5YzcyN2FhMjdiMmZmODYxNzVhYzVlNDhiZTgwODAyMWJmOTFkNGZjMg
  ```
  --- last 10 line(s) of stdout (of 64 after folding 64 raw)
            "pattern": {
              "advisory_writes": 0,
              "window": 0,
              "fires": false
            }
          }
  --- FAIL: TestAMalformedHarnessIsAJSONDocumentUnderJSON (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.275s
  FAIL
  ```
- 2026-09-25 · e3f7978* · exit 1 · `set -o pipefail …` · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · ms:458
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  === RUN   TestNoCheckSkipsTheHarnessEntirely
  --- PASS: TestNoCheckSkipsTheHarnessEntirely (0.00s)
  === RUN   TestDryRunDoesNotLoadTheHarness
  --- PASS: TestDryRunDoesNotLoadTheHarness (0.00s)
  === RUN   TestAMalformedHarnessIsAJSONDocumentUnderJSON
      json_refusal_test.go:59: --json printed something that is not one JSON document: unexpected end of JSON input
  --- FAIL: TestAMalformedHarnessIsAJSONDocumentUnderJSON (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.109s
  FAIL
  ```
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · ms:634
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · ms:348
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · ms:328
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · ms:418

## Mutation Log
(empty until execute)
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · the harness is never read before apply, so a malformed one lets the write land · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · covers:the harness is read before apply
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · --dry-run reads the harness and a malformed one refuses a dry run · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · covers:no-check and dry-run skip it
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · --no-check reads the harness and a malformed one refuses the write · acceptance-sha256:01001c2816d767067181c26d24e79776efa8c6c2662b58625f15983bb6893db3 · covers:no-check and dry-run skip it

## Invariants

- Exit codes keep their meaning.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-072 Out of Scope)

## Stop Condition

Stop if the change needs an engine package other than `internal/check`.
