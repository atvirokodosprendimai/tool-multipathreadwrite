# Task ADR-107-T3: the recheck sees the leaf, and a failed ID load refuses

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the recheck sees the leaf, and a failed ID load refuses
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a leaf swapped for a link is refused`

## Goal

`changedSince` lstats the target through the tree first, and a link where validation saw a regular file is a change; a failed eager ID load at validation refuses the file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `changedSince` takes the tree and lstats the leaf; the ID load is checked |
| `internal/apply/leaf107_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestTheRecheckRefusesALeafSwappedForALink`; confirm RED. [proof: mutation]
2. [S2] The leaf check and the ID refusal. Mutants: the leaf check removed. The ID refusal is Windows-only and declared uncovered in the record. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestTheRecheckRefusesALeafSwappedForALink' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheRecheckRefusesALeafSwappedForALink \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRecheckRefusesALeafSwappedForALink` | `internal/apply/leaf107_test.go` | with `a.txt` moved to `b.txt` and `a.txt` made a link to it after it stages, the plan fails naming `a.txt`, `a.txt` is still the link and `b.txt` keeps its bytes; skipped where symlinks cannot be made | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every write, or every MCP acknowledgement |
| 3 — the caller can discover it | the refusal names the file and the reason |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 final design review |

## Mutation Log
- 2026-09-30 · 433ed20* · mutant killed · exit 1 · `internal/apply/apply.go` · the leaf check removed · acceptance-sha256:8c64db54171b669ff6bd250d40c4221473f0d2c259759a0d01503a0094535bd4 · covers:a leaf swapped for a link is refused

## Invariants

- Exit codes keep their meanings; plans under the limits are unaffected.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-107 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 433ed20* · exit 1 · `set -o pipefail …` · acceptance-sha256:8c64db54171b669ff6bd250d40c4221473f0d2c259759a0d01503a0094535bd4 · ms:127 · test-lock-sha256:feac82e1c81a139491b2abc9102b6304d43bc8bee400160b201d70b510eeb5c0 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2xlYWYxMDdfdGVzdC5nbwlUZXN0VGhlUmVjaGVja1JlZnVzZXNBTGVhZlN3YXBwZWRGb3JBTGluawllYTJjMjY5MzI4MzEzM2U4YzkyZmZmYzIyN2RhOWU4NzRhNmQ2NTcxMWVhNTdhMzVlNTM2YjcxNzI5Y2Y0NTY1
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply.test]
  internal/apply/load107_test.go:16:9: undefined: maxLoadBytes
  internal/apply/load107_test.go:17:21: undefined: maxLoadBytes
  internal/apply/load107_test.go:18:2: undefined: maxLoadBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-09-30 · 433ed20* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c64db54171b669ff6bd250d40c4221473f0d2c259759a0d01503a0094535bd4 · ms:274
- 2026-09-30 · 433ed20* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c64db54171b669ff6bd250d40c4221473f0d2c259759a0d01503a0094535bd4 · ms:293
- 2026-10-06 · d493df7* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:8c64db54171b669ff6bd250d40c4221473f0d2c259759a0d01503a0094535bd4 · ms:0 · test-lock-sha256:ccfdee41c0b2aff67553bdba01189abb5cbd82e0488c1d52a4fed62343828439 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVhZjEwN190ZXN0LmdvCVRlc3RUaGVSZWNoZWNrUmVmdXNlc0FMZWFmU3dhcHBlZEZvckFMaW5rCTk0Mzg0MGFmN2Q4NjlmNmZjNGYxZDA4NTFlODFiNTVmZDRkNjc0YjAzNTI0NWQxYjFmY2M5ZWU0NjVjNzMyOGM · test-lock-kind:replace
