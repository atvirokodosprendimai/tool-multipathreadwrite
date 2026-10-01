# Task ADR-108-T1: a checkpoint covers only consecutive served lines

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** a checkpoint covers only consecutive served lines
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a checkpoint licenses only what was served`

## Goal

`interleave` ends a checkpoint when the next served line is not the next number, so acknowledging a sparse read licenses exactly the served lines.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/ack.go` | edit | `interleave` |
| `internal/mcp/gap108_test.go` | add | the test below |
| `scripts/contract.sh` | edit | §202 |

## Ordered Steps

1. [S1] Write the failing test(s) `TestACheckpointCoversOnlyConsecutiveServedLines`; confirm RED. [proof: mutation]
2. [S2] Break at a gap; contract §202 through `$MRW mcp`. Mutant: the gap break removed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestACheckpointCoversOnlyConsecutiveServedLines' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestACheckpointCoversOnlyConsecutiveServedLines \(' "$out" \
  && grep -q '^# 202\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/iter internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACheckpointCoversOnlyConsecutiveServedLines` | `internal/mcp/gap108_test.go` | lines 1 and 100 of one file served together give two checkpoints, [1,1] and [100,100]; a run of 3 consecutive lines gives one; across random served sets, the union of checkpoint spans equals the served numbers | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write or acknowledgement that reaches it |
| 3 — the caller can discover it | the refusal or the receipt names it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review |

## Mutation Log
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/mcp/ack.go` · the gap break removed: lines 1 and 100 share one checkpoint spanning 2-99 · acceptance-sha256:f6d6dd4b5ae42d2a5f3bca0de9c12764459d69b780cba4a59794e9ff7c8feefa

## Invariants

- Exit codes keep their meanings; inputs the defect did not touch behave as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 98feab5* · exit 1 · `set -o pipefail …` · acceptance-sha256:f6d6dd4b5ae42d2a5f3bca0de9c12764459d69b780cba4a59794e9ff7c8feefa · ms:1068 · test-lock-sha256:eb218df2731ff62d7b63447f0ae0f6c9d54ac4f6b549c6b850073ba7e32d8653 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9nYXAxMDhfdGVzdC5nbwlUZXN0QUNoZWNrcG9pbnRDb3ZlcnNPbmx5Q29uc2VjdXRpdmVTZXJ2ZWRMaW5lcwliNGFmODAzOTU3ZDZmOWUwMDNiZmI0OWRjOWFmOGJiYTQ3NDQ5MmIzN2Y2MjllMjhkZjgzNjRmNWY3NTBkMTNk
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
  internal/mcp/landed102_test.go:62:34: cannot use root (variable of type map[int]bool) as string value in argument to authoring.Recent
  internal/mcp/landed102_test.go:62:64: cannot use root (variable of type map[int]bool) as string value in argument to authoring.LoadPricing
  internal/mcp/landed102_test.go:73:22: too many arguments in call to licensed
  	have (*testing.T, map[string]string)
  	want (map[string][2]int)
  internal/mcp/landed102_test.go:74:31: cannot use root (variable of type map[int]bool) as string value in argument to seen.ReadPath
  internal/mcp/landed102_test.go:82:17: cannot use root (variable of type map[int]bool) as string value in argument to call
  internal/mcp/landed102_test.go:82:17: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:f6d6dd4b5ae42d2a5f3bca0de9c12764459d69b780cba4a59794e9ff7c8feefa · ms:551
- 2026-10-01 · 98feab5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f6d6dd4b5ae42d2a5f3bca0de9c12764459d69b780cba4a59794e9ff7c8feefa · ms:0 · test-lock-sha256:5bcb2fc8531da9e6d710665ea74c3b8274f5ecc2bc369686fb8b3e8e3b9139cc · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9nYXAxMDhfdGVzdC5nbwlUZXN0QUNoZWNrcG9pbnRDb3ZlcnNPbmx5Q29uc2VjdXRpdmVTZXJ2ZWRMaW5lcwlhMWM4M2I0NzlmZjI0ODM4NGJlYzk2MjZhZGJhMzhiMTlmZjU4ZTFmNmVkYzZlZmM4MzA3YmJkN2Y0MmEyYWE3 · test-lock-kind:replace
- 2026-10-01 · human-observed · Zy's session observed the relock: after red the helper licensed was renamed licensedLines, since landed102_test.go declares licensed; the assertions are unchanged
