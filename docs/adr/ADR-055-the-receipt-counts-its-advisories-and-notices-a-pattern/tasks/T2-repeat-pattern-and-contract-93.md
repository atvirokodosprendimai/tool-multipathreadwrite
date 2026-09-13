# Task ADR-055-T2: Recent-window ring and the pattern line; contract §93

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `authoring.Recent` + pattern line (T2)
**Consumes:** `apply.Result.Advisories` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `ring holds last 10`, `no paths or plan text`, `line at 3 of 10`, `fails open`, `stats shows the window`

## Goal

Every write `Record` site also appends `<unix-seconds> write <advisories>` to `recent` beside the tally, keeping the last 10. After a CLI write, when ≥3 of the last 10 carried an advisory, `report` prints `pattern: K of your last N writes carried a balance advisory — read past the range before the next one`. `mrw stats` prints the same line under the same condition and `recent: N write(s) in the window` always. Thresholds are named constants in `internal/authoring`. Unreadable or absent `recent` is empty, never an error.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/authoring/authoring.go` | edit | `RecordRecent`, `Recent`, `Pattern` and the two constants. Vocabulary untouched. |
| `internal/authoring/authoring_test.go` | edit | Red: ring keeps 10; no path survives a write; `Pattern` at 3 of 10; fail-open on garbage. |
| `cmd/mrw/main.go` | edit | Call `RecordRecent` beside every write `Record`; `report` pattern line; `stats` window lines. The selector. |
| `internal/mcp/tools.go` | edit | `RecordRecent` beside its `Record` (feeds the ring; prints nothing). |
| `cmd/mrw/advisory_test.go` | edit | Red: three advisory writes then a fourth prints the line; two do not; `stats` shows the window. |
| `scripts/contract.sh` | edit | **§93** — pair: three `{`-only replaces then the receipt prints `pattern:` / two do not / `stats` prints `recent:`; `strings recent` shows no path. |

## Ordered Steps

1. [S1] Write `TestRecentKeepsTheLastTenWritesAndNoPaths` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestPatternFiresAtThreeOfTenAndNotAtTwo` and `TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory` — RED. [proof: mutation]
3. [S3] Implement the ring, `Pattern`, and the two call sites. Confirm GREEN. Deleting the CLI `RecordRecent` call must fail S2; lowering the threshold to 2 must fail the not-at-two half. [proof: mutation]
4. [S4] Write §93 RED then GREEN, including the `strings` check that no path reached disk. [proof: mutation]
5. [S5] Scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 93\. ' scripts/contract.sh \
  && go test ./internal/authoring/ ./cmd/mrw/ -count=1 -v \
    -run 'TestRecentKeepsTheLastTenWritesAndNoPaths|TestPatternFiresAtThreeOfTenAndNotAtTwo|TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory' 2>&1 | tee /tmp/adr055-t2.out \
  && grep -q '^--- PASS: TestRecentKeepsTheLastTenWritesAndNoPaths' /tmp/adr055-t2.out \
  && grep -q '^--- PASS: TestPatternFiresAtThreeOfTenAndNotAtTwo' /tmp/adr055-t2.out \
  && grep -q '^--- PASS: TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory' /tmp/adr055-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr055-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/authoring/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestRecentKeepsTheLastTenWritesAndNoPaths` | `internal/authoring/authoring_test.go` | ring keeps the last 10; three fields per line; nothing path-shaped | — | S1, S3 |
| `TestPatternFiresAtThreeOfTenAndNotAtTwo` | `internal/authoring/authoring_test.go` | 2 of 4 does not fire; 3 of 5 fires; garbage reads as empty | — | S2, S3 |
| `TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory` | `cmd/mrw/advisory_test.go` | third advisory write prints `pattern:`; second does not; stats shows window | — | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §93 |
| 2 — something selects it | every write `Record` site calls `RecordRecent`; deleting the CLI call fails S2 and §93 |
| 3 — the caller can discover it | the line arrives unasked on the receipt; `stats` shows the window; T4 |
| 4 — it is used | the field run's fourth repeat is the case; ADR-009 refused telemetry |

## Mutation Log
(empty until execute)
- 2026-09-13 · b02be55* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never feeds the ring: three advisory writes leave it empty, no pattern line prints, and TestTheReceiptPrintsThePatternLineOnTheThirdAdvisory must go red · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729
- 2026-09-13 · b02be55* · mutant killed · exit 1 · `internal/authoring/authoring.go` · the threshold drops to two: the second advisory write prints the line, and the not-at-two half of TestPatternFiresAtThreeOfTenAndNotAtTwo plus the receipt test must go red · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729
- 2026-09-13 · b02be55* · mutant survived · exit 0 · `internal/authoring/authoring.go` · the ring never trims: thirteen writes leave thirteen lines and TestRecentKeepsTheLastTenWritesAndNoPaths must go red · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-13 · b02be55* · mutant killed · exit 1 · `internal/authoring/authoring.go` · the writer never trims: thirteen writes leave thirteen lines on disk and the file-length assertion in TestRecentKeepsTheLastTenWritesAndNoPaths must go red · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729

## Invariants

- ADR-009: counts and a timestamp only; no path, plan text or address in `recent`. Fails open. Never fails a write.
- Five authoring names unchanged; no sixth Outcome.
- MCP feeds the ring and prints nothing.

## Risks

- Two writers appending concurrently (CLI + MCP) can interleave lines; the ring is advisory input and a torn line is skipped by the reader, never an error.
- The reader trims to the window as well as the writer, which masked a mutant on the writer's trim until the test asserted the FILE's line count — a ring that trims only on read grows without bound on disk.
- Only LANDED writes join the ring (`res.Applied && !res.DryRun`); a refused plan wrote nothing and is not a write the pattern is about.

## Stop Condition

If the only way to go green is to record a path or a plan line, stop — ADR-009's boundary.

## Out of Scope

- The advisory count itself (T1)
- `--strict-balance` (T3)
- Pattern line on the MCP receipt (BACKLOG)

## Verification Log
(empty until execute)
- 2026-09-13 · b02be55* · exit 1 · `set -o pipefail …` · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729 · ms:95 · test-lock-sha256:af094822a0df175a9e7ea3e33de557eb86a78089f5174a23f031021ec80f3c5a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYWR2aXNvcnlfdGVzdC5nbwlUZXN0UXVpZXRBbmRKU09OQ2FycnlUaGVBZHZpc29yeUNvdW50CTVjODliNjNjYWMzM2E4YWYwNzczZGViMGUyNDIxNTY4NmNjOGI1MDI3MGNmZmEwMDcyNDhmYjI3NGI3MGI3MTAKYm9keQljbWQvbXJ3L2Fkdmlzb3J5X3Rlc3QuZ28JVGVzdFRoZVJlY2VpcHRQcmludHNUaGVQYXR0ZXJuTGluZU9uVGhlVGhpcmRBZHZpc29yeQljYWU3Zjg3N2NhNjU0MmU5NzRlYTlhOTZiOWM3NWQ1NjU2OGRjMzk5ODZkNjhmOTMzMzYxYzdiYzFjYjcxY2FiCmJvZHkJY21kL21ydy9hZHZpc29yeV90ZXN0LmdvCVRlc3RUaGVTdW1tYXJ5TGluZUNvdW50c0Fkdmlzb3JpZXMJZTEwODk2MDBmNDVmOGI5Zjg2NDFjOWY1ZjZhYzFlODZkMjA2YTE1ZDEwNDVhODhiMWEyOTBhOGY2ZDk1MjFhMwpib2R5CWludGVybmFsL2F1dGhvcmluZy9hdXRob3JpbmdfdGVzdC5nbwlUZXN0QW5VbnJlYWRhYmxlVGFsbHlGYWlsc09wZW4JYTk2YjE4NjRjYjRlN2VmNTU2N2FhNzc0YmE5NDFkNTFhNTZmYzIxMWYyZWJhY2M1MWFlYjFmNDI5MTlkZGRhZApib2R5CWludGVybmFsL2F1dGhvcmluZy9hdXRob3JpbmdfdGVzdC5nbwlUZXN0UGF0dGVybkZpcmVzQXRUaHJlZU9mVGVuQW5kTm90QXRUd28JZGE4NDM3NmFiOTgzZDI3MGQwYTkzYjg4NWIxODljYWZlYWIwZTg2NzY2Y2YxNjFiMGMwYWU3YTczYzkxMTcyYgpib2R5CWludGVybmFsL2F1dGhvcmluZy9hdXRob3JpbmdfdGVzdC5nbwlUZXN0UmVjZW50S2VlcHNUaGVMYXN0VGVuV3JpdGVzQW5kTm9QYXRocwlmM2JiYjM3MDQ4NGVjNzE5YjFkODVmYjUyMmJmODU3MjI5NjU4YTE5YmViMDYzNzA3OTdjMjBkZTFmMTlkZDFhCmJvZHkJaW50ZXJuYWwvYXV0aG9yaW5nL2F1dGhvcmluZ190ZXN0LmdvCVRlc3RSZWNvcmROZXZlckZhaWxzQVdyaXRlCTJhMWIzMDRjY2E1NmQzODU5NDRkMjE4OWMwOTQzMTNlY2JlNTdkNjVjMTNkY2U2OWEzNTBlOWMzYWZkYzAyMDUKYm9keQlpbnRlcm5hbC9hdXRob3JpbmcvYXV0aG9yaW5nX3Rlc3QuZ28JVGVzdFRhbGx5Q291bnRzRWFjaE91dGNvbWVTZXBhcmF0ZWx5CWQ4MmM1YTQxM2ViYWYzOTIyOWEzY2Q5NjhhMDI5YWZiYWJjNjdhYjkyYjM2YTBiNWY3YzVhNmMyZjZlOWRhMWUKYm9keQlpbnRlcm5hbC9hdXRob3JpbmcvYXV0aG9yaW5nX3Rlc3QuZ28JVGVzdFRhbGx5Um91bmRUcmlwc1Rocm91Z2hMb2FkCTU4OGU3MzYxOTFmNjY3OTkwNjdlNzJjM2RiMTZjMmM0MDE5M2VmZDEyMTkzOWI4YjgwZTQyYWJjMjAyYWMxM2EKYm9keQlpbnRlcm5hbC9hdXRob3JpbmcvYXV0aG9yaW5nX3Rlc3QuZ28JVGVzdFRoZVRhbGx5TmV2ZXJSZWNvcmRzUGxhbkNvbnRlbnRPclBhdGhzCTBjY2ZjMGE0NzA3NmY3N2Y2MWY0Y2VhMjI2MGZlYmIzYzFlOTU0ZTUyMGVjNWQwZDk0MjY1ZmIwZDEyMjFlN2Y
  ```
  ```
- 2026-09-13 · b02be55* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729 · ms:982
- 2026-09-13 · b02be55* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729 · ms:873
- 2026-09-13 · b02be55* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729 · ms:721
- 2026-09-13 · b02be55* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7c0f2683fa398b01884a687d2108ae6cb0d7d248926a34facae7f9123131729 · ms:781
