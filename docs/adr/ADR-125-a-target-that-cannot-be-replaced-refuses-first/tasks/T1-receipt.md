# Task ADR-125-T1: a target mrw cannot open is refused on its hunk, naming why

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `apply.causeOf`, the load and identity refusals on the hunk
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a target mrw cannot open is refused on its hunk, naming why`

## Goal

A load error whose cause mrw can name (permission, held open, invalid name), and a file identity that cannot be read, fail the file's hunk with a reason naming the cause; the siblings skip, nothing is written, exit 1, and the receipt is printed. Any other load error stays exit 2.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the load error and the identity refusal through `refuseFile`; `sameFileFn` |
| `internal/apply/replace.go` | add | `causeOf` |
| `internal/apply/replace_windows.go`, `internal/apply/replace_other.go` | add | the platform causes (sharing violation, invalid name) |
| `internal/apply/replace125_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §223 |
| `AGENTS.md`, `README.md` | edit | a refusal names why mrw could not open the file |

## Ordered Steps

1. [S1] Write `TestAnUnreadableTargetGetsAReceipt` and `TestTheIdentityRefusalNamesItsCause`. Confirm RED. [proof: mutation]
2. [S2] `causeOf`, the load refusal, the identity refusal. Mutants: the load error returned bare again; the identity reason without its cause. [proof: mutation]
3. [S3] Contract §223 and the docs. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAnUnreadableTargetGetsAReceipt|TestTheIdentityRefusalNamesItsCause' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnUnreadableTargetGetsAReceipt \(' "$out" \
  && grep -qE '^--- PASS: TestTheIdentityRefusalNamesItsCause \(' "$out" \
  && go test ./internal/apply/ ./internal/writer/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 900s \
  && GOOS=windows go vet ./internal/apply/ \
  && grep -q '^# 223\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnUnreadableTargetGetsAReceipt` | `internal/apply/replace125_test.go` | a permission-denied load fails its hunk naming "permission denied"; the sibling skips and is unchanged; no error is returned bare | none | S1, S2 |
| `TestTheIdentityRefusalNamesItsCause` | `internal/apply/replace125_test.go` | an unreadable identity names the open's error, and "send the plan again" only when nothing explains it | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `causeOf` |
| 2 — something selects it | `Apply`'s validation loop, on every write |
| 3 — the caller can discover it | the receipt's reason; AGENTS.md and README |
| 4 — it is used | the Windows peers met it on 2026-10-02; no telemetry (ADR-009) |

## Invariants

- A file mrw can open is validated exactly as before.
- Nothing is written when a target cannot be opened.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a load error cannot be made a hunk refusal without changing a receipt key.

## Out of Scope

- The Windows probe and rename (deferred: T2-replace.md)

## Mutation Log
- 2026-10-06 · cc32ee4* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the load error returned bare again · acceptance-sha256:6772b608c68ca4c9f4e23228360ff1deaef712cacf34ca1a05d5d2c2ddf3de8a
- 2026-10-06 · cc32ee4* · mutant killed · exit 1 · `internal/apply/replace.go` · S2: the identity reason without its cause · acceptance-sha256:6772b608c68ca4c9f4e23228360ff1deaef712cacf34ca1a05d5d2c2ddf3de8a

## Verification Log
- 2026-10-06 · cc32ee4* · exit 1 · `set -o pipefail …` · acceptance-sha256:6772b608c68ca4c9f4e23228360ff1deaef712cacf34ca1a05d5d2c2ddf3de8a · ms:496 · test-lock-sha256:6798b76601cec51519b786d4cb6960f9788cb441fc621c8da7a45475d3f61504 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvcmVwbGFjZTEyNV90ZXN0LmdvCVRlc3RBblVucmVhZGFibGVUYXJnZXRHZXRzQVJlY2VpcHQJODhjMDQyM2YzMzEyODU5YzdjY2E1ZTU0YjU5YWU5MjA2MjUxNmIwY2RjZmY0MjU5NzRlMjJmZmQ3NGNiZjg3Ywpib2R5CWludGVybmFsL2FwcGx5L3JlcGxhY2UxMjVfdGVzdC5nbwlUZXN0VGhlSWRlbnRpdHlSZWZ1c2FsTmFtZXNJdHNDYXVzZQliNDE4MmZkYjM1MGFkZTAyN2EzYmYzOWEzOTE5NjI2NzljYzUxNjMzOTZmNjExNGI0ZmE3NWY5NjUwNDljN2E2
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply.test]
  internal/apply/replace125_test.go:64:10: undefined: sameFileFn
  internal/apply/replace125_test.go:65:21: undefined: sameFileFn
  internal/apply/replace125_test.go:66:2: undefined: sameFileFn
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-10-06 · cc32ee4* · exit 1 · `set -o pipefail …` · acceptance-sha256:6772b608c68ca4c9f4e23228360ff1deaef712cacf34ca1a05d5d2c2ddf3de8a · ms:49870
  ```
  --- last 10 line(s) of stdout (of 67 after folding 67 raw)
  1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied
  @1   sub/a.go
  1 entr(ies), 1 file(s)
  --- FAIL: TestAPlanRefusedAfterItParsedIsOneRefusal (0.01s)
      tally083_test.go:46: a plan naming a directory: exit 1, want 2
          FAIL d 1 replace (plan line 1): d cannot be opened: read /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAPlanRefusedAfterItParsedIsOneRefusal4063711364/002/d: is a directory
          1 hunk(s), 1 file(s), 1 failed, 0 advisories — NOTHING WRITTEN
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	46.744s
  FAIL
  ```
- 2026-10-06 · cc32ee4* · exit 1 · `set -o pipefail …` · acceptance-sha256:6772b608c68ca4c9f4e23228360ff1deaef712cacf34ca1a05d5d2c2ddf3de8a · ms:50181
  ```
  --- last 10 line(s) of stdout (of 67 after folding 67 raw)
  1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied
  @1   sub/a.go
  1 entr(ies), 1 file(s)
  --- FAIL: TestAPlanRefusedAfterItParsedIsOneRefusal (0.01s)
      tally083_test.go:46: a plan naming a directory: exit 1, want 2
          FAIL d 1 replace (plan line 1): d cannot be opened: read /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAPlanRefusedAfterItParsedIsOneRefusal2051479216/002/d: is a directory
          1 hunk(s), 1 file(s), 1 failed, 0 advisories — NOTHING WRITTEN
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	46.622s
  FAIL
  ```
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:6772b608c68ca4c9f4e23228360ff1deaef712cacf34ca1a05d5d2c2ddf3de8a · ms:37287
- 2026-10-06 · cc32ee4* · exit 0 · `set -o pipefail …` · acceptance-sha256:6772b608c68ca4c9f4e23228360ff1deaef712cacf34ca1a05d5d2c2ddf3de8a · ms:37294
