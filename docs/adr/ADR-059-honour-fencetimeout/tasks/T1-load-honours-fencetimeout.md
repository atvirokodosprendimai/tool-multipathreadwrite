# Task ADR-059-T1: `Load` honours `fenceTimeout`; contract §97

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `fenceTimeout` alias of `timeout_seconds`; disagree-refuse; §97
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `alias fills TimeoutSeconds`, `timeout_seconds alone still applies`, `equal keys accepted`, `disagreeing keys refuse`, `Zeus-shaped Load then Run times out`

## Goal

quality-harness writes `fenceTimeout`. mrw dropped it. Honour it as `timeout_seconds`. Both keys set to different values refuse at `Load`. Drive a Zeus-shaped file through `Load` then `Run`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `FenceTimeout` JSON field; resolve after unmarshal. |
| `internal/check/timeout_alias_test.go` | create | Red: Zeus JSON → 1800; timeout_seconds alone; equal keys; disagree refuse; Load+Run times out. |
| `scripts/contract.sh` | edit | **§97** — next free after §96. |

## Ordered Steps

1. [S1] Write the five tests — RED. [proof: mutation]
2. [S2] Implement the alias and the disagree refusal. S1 GREEN. [proof: mutation]
3. [S3] Write §97 RED then GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 97\. ' scripts/contract.sh \
  && go test ./internal/check/ -count=1 -v \
    -run 'TestLoadHonoursFenceTimeout|TestLoadTimeoutSecondsAloneStillApplies|TestLoadEqualTimeoutKeysAreAccepted|TestLoadRefusesWhenTimeoutKeysDisagree|TestAZeusShapedHarnessTimesOutUsingFenceTimeout' 2>&1 | tee /tmp/adr059-t1.out \
  && grep -q '^--- PASS: TestLoadHonoursFenceTimeout' /tmp/adr059-t1.out \
  && grep -q '^--- PASS: TestLoadTimeoutSecondsAloneStillApplies' /tmp/adr059-t1.out \
  && grep -q '^--- PASS: TestLoadEqualTimeoutKeysAreAccepted' /tmp/adr059-t1.out \
  && grep -q '^--- PASS: TestLoadRefusesWhenTimeoutKeysDisagree' /tmp/adr059-t1.out \
  && grep -q '^--- PASS: TestAZeusShapedHarnessTimesOutUsingFenceTimeout' /tmp/adr059-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr059-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/check/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestLoadHonoursFenceTimeout` | `internal/check/timeout_alias_test.go` | Zeus-shaped `{check, fenceTimeout:1800}` → `TimeoutSeconds==1800` | — | S1, S2 |
| `TestLoadTimeoutSecondsAloneStillApplies` | `internal/check/timeout_alias_test.go` | `{timeout_seconds:42}` → 42 | — | S1, S2 |
| `TestLoadEqualTimeoutKeysAreAccepted` | `internal/check/timeout_alias_test.go` | both 30 → 30, no error | — | S1, S2 |
| `TestLoadRefusesWhenTimeoutKeysDisagree` | `internal/check/timeout_alias_test.go` | 10 vs 1800 is an error naming both | — | S1, S2 |
| `TestAZeusShapedHarnessTimesOutUsingFenceTimeout` | `internal/check/timeout_alias_test.go` | Load then Run of `sleep 5` with `fenceTimeout:1` reports timed out | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §97 |
| 2 — something selects it | `Load` is the only unmarshal; deleting the copy leaves TimeoutSeconds 0 and S1 red |
| 3 — the caller can discover it | §97 drives the binary; README/AGENTS name the alias in T-teach of ADR-057 if needed |
| 4 — it is used | Zeus already writes the key; ADR-009 refuses telemetry |

## Mutation Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/check/check.go` · the alias is never copied: TimeoutSeconds stays 0 and TestLoadHonoursFenceTimeout must go red · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/check/check.go` · disagreeing keys are accepted: TestLoadRefusesWhenTimeoutKeysDisagree must go red · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876

## Invariants

- `timeout_seconds` alone is unchanged.
- `Run` still reads only `TimeoutSeconds`.
- A missing file still infers `go test` when `go.mod` exists.

## Risks

| Risk | Mitigation |
|------|------------|
| The 1s timeout test flakes on a slow runner | `sleep 5` against a 1s bound; the existing `TestTimeoutIsReportedAsAFailureNotAPass` already uses this shape |

## Stop Condition

Stop if honouring `fenceTimeout` requires changing `Run`'s clamp or `defaultTimeout`: that is a different bound, not an alias.

## Out of Scope

- Per-extension check skip (the record).
- Teaching AGENTS.md (a one-line note if a later teach task exists; this task is Load).

## Verification Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · exit 1 · `set -o pipefail …` · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876 · ms:5985 · test-lock-sha256:4d47b7c9ae963963e01347386e0b87d02638ae28f98f2150e0baf6ed48fa15e1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL3RpbWVvdXRfYWxpYXNfdGVzdC5nbwlUZXN0QVpldXNTaGFwZWRIYXJuZXNzVGltZXNPdXRVc2luZ0ZlbmNlVGltZW91dAliMWEwODE3NzdmYWE3NWQ3OTE3MDZlYjFmYmExOTczNjI5YmIzNDE3MWRhOTkyNWIwMjYyYjRmY2JhMjFhZTI0CmJvZHkJaW50ZXJuYWwvY2hlY2svdGltZW91dF9hbGlhc190ZXN0LmdvCVRlc3RMb2FkRXF1YWxUaW1lb3V0S2V5c0FyZUFjY2VwdGVkCTQ0ZjQxYmQ3MTNkM2YwOWZmZDM3NzM3MWQ1YzVkOGUwNGVkMTIwODQ3MTIxNDFiNDM0Y2ZhZjk5NjIzNGRhMmYKYm9keQlpbnRlcm5hbC9jaGVjay90aW1lb3V0X2FsaWFzX3Rlc3QuZ28JVGVzdExvYWRIb25vdXJzRmVuY2VUaW1lb3V0CTk1M2FkZmM1NDA2ZjQ1OTcyNzljY2EzMTIxZTRjZWNjMTk3MjMzOGE5M2I5MTlhYzBiZGYxNTgxMGFhYWQzNDkKYm9keQlpbnRlcm5hbC9jaGVjay90aW1lb3V0X2FsaWFzX3Rlc3QuZ28JVGVzdExvYWRSZWZ1c2VzV2hlblRpbWVvdXRLZXlzRGlzYWdyZWUJODZkOGRjZDJjMWJkYTg1NzYyM2EwZjc0ZDQ2MGYwYzNiNGJkZDBkYWU5OTQ3YTljNmU1NDAyM2E3M2NmMmE4ZApib2R5CWludGVybmFsL2NoZWNrL3RpbWVvdXRfYWxpYXNfdGVzdC5nbwlUZXN0TG9hZFRpbWVvdXRTZWNvbmRzQWxvbmVTdGlsbEFwcGxpZXMJMTA0ZjMxZTUzMGUxODdmODgxYzMyZGQ2NDAwOTcyOTQ5MDQ2MGE4ZDg1YjdhODU2ODU5NDI3ODcwZjllM2Y5NA
  ```
  --- last 10 line(s) of stdout (of 17 after folding 17 raw)
  === RUN   TestLoadRefusesWhenTimeoutKeysDisagree
      timeout_alias_test.go:67: disagreeing timeout keys loaded
  --- FAIL: TestLoadRefusesWhenTimeoutKeysDisagree (0.00s)
  === RUN   TestAZeusShapedHarnessTimesOutUsingFenceTimeout
      timeout_alias_test.go:90: a fenceTimeout-bounded sleep reported OK
      timeout_alias_test.go:93: Skipped = ""
  --- FAIL: TestAZeusShapedHarnessTimesOutUsingFenceTimeout (5.02s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	5.202s
  FAIL
  ```
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876 · ms:2410
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876 · ms:4049
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876 · ms:2334
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876 · ms:0 · test-lock-sha256:743aa1210b6ee9a7fb6caa9360f92ee252e2d5935b0d014b24cdaa28c1548e00 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL3RpbWVvdXRfYWxpYXNfdGVzdC5nbwlUZXN0QVpldXNTaGFwZWRIYXJuZXNzVGltZXNPdXRVc2luZ0ZlbmNlVGltZW91dAliNzY5NGQ4NmMwNjcxZDgyYThlYTM5N2MxMTQyODVjMWNmZmNhOThkYzc1ODM4YTM1YjIxNWMwZTJlMDI4ZGJkCmJvZHkJaW50ZXJuYWwvY2hlY2svdGltZW91dF9hbGlhc190ZXN0LmdvCVRlc3RMb2FkRXF1YWxUaW1lb3V0S2V5c0FyZUFjY2VwdGVkCTQ0ZjQxYmQ3MTNkM2YwOWZmZDM3NzM3MWQ1YzVkOGUwNGVkMTIwODQ3MTIxNDFiNDM0Y2ZhZjk5NjIzNGRhMmYKYm9keQlpbnRlcm5hbC9jaGVjay90aW1lb3V0X2FsaWFzX3Rlc3QuZ28JVGVzdExvYWRIb25vdXJzRmVuY2VUaW1lb3V0CTk1M2FkZmM1NDA2ZjQ1OTcyNzljY2EzMTIxZTRjZWNjMTk3MjMzOGE5M2I5MTlhYzBiZGYxNTgxMGFhYWQzNDkKYm9keQlpbnRlcm5hbC9jaGVjay90aW1lb3V0X2FsaWFzX3Rlc3QuZ28JVGVzdExvYWRSZWZ1c2VzV2hlblRpbWVvdXRLZXlzRGlzYWdyZWUJODZkOGRjZDJjMWJkYTg1NzYyM2EwZjc0ZDQ2MGYwYzNiNGJkZDBkYWU5OTQ3YTljNmU1NDAyM2E3M2NmMmE4ZApib2R5CWludGVybmFsL2NoZWNrL3RpbWVvdXRfYWxpYXNfdGVzdC5nbwlUZXN0TG9hZFRpbWVvdXRTZWNvbmRzQWxvbmVTdGlsbEFwcGxpZXMJMTA0ZjMxZTUzMGUxODdmODgxYzMyZGQ2NDAwOTcyOTQ5MDQ2MGE4ZDg1YjdhODU2ODU5NDI3ODcwZjllM2Y5NA · test-lock-kind:replace
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:71931591a0b05f129ae4a42c832630be633f596e03699c2783f2230561a3f876 · ms:1773
