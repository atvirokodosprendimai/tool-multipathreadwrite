# Task ADR-108-T2: a non-ASCII rune never splits a path

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** a non-ASCII rune never splits a path
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a non-ASCII rune never splits a path`

## Goal

`components` treats a rune as a separator only when it is ASCII and the OS calls it one.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/links/links.go` | edit | `components` |
| `internal/links/rune108_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAUnicodeRuneNeverSplitsAPath`; confirm RED. [proof: mutation]
2. [S2] Compare runes, not narrowed bytes. Mutant: the narrowing restored. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/links/ -count=1 -timeout 300s -run 'TestAUnicodeRuneNeverSplitsAPath' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAUnicodeRuneNeverSplitsAPath \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAUnicodeRuneNeverSplitsAPath` | `internal/links/rune108_test.go` | `components` keeps `aЯb.txt` and `aŜb.txt` whole and every rune above 0x7F whose low byte is 0x2F or 0x5C inside its component, and still splits on `/` (and `\\` on Windows) | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write or acknowledgement that reaches it |
| 3 — the caller can discover it | the refusal or the receipt names it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review |

## Mutation Log
- 2026-10-01 · 98feab5* · mutant inconclusive · exit 1 · `internal/links/links.go` · the narrowing restored: U+042F splits a path at its low byte 0x2F · acceptance-sha256:ced98c14f957e315c01daf917f5551622b27ad22652e6ce2e9b243843543eedb
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/links/links.go` · the narrowing restored: U+042F splits a path at its low byte 0x2F · acceptance-sha256:ced98c14f957e315c01daf917f5551622b27ad22652e6ce2e9b243843543eedb
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/links/links.go` · the narrowing restored: U+042F splits a path at its low byte 0x2F · acceptance-sha256:8916065c4be054ff4cecacdf5af1ae40ee85918be7841bf2eae977531ce65006

## Invariants

- Exit codes keep their meanings; inputs the defect did not touch behave as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 98feab5* · exit 1 · `set -o pipefail …` · acceptance-sha256:ced98c14f957e315c01daf917f5551622b27ad22652e6ce2e9b243843543eedb · ms:654 · test-lock-sha256:5b85e4561e0def36d7b30197ebf51701c7e197075ddb68c87dc539c9516f48a8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2xpbmtzL3J1bmUxMDhfdGVzdC5nbwlUZXN0QVVuaWNvZGVSdW5lTmV2ZXJTcGxpdHNBUGF0aAlhYzQzNzI4ZDQyNzEwZTJmZGU3NTNmMmY0MzI5Y2M4ZmU5Y2E0MWZkY2I5ZTIxNzJjMTQ4ODk0Mzc2ZTcxN2Nj
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAUnicodeRuneNeverSplitsAPath
      rune108_test.go:16: components("aЯb.txt") = ["a" "b.txt"], want it whole
      rune108_test.go:25: U+012F split "xįy" into ["x" "y"]
  --- FAIL: TestAUnicodeRuneNeverSplitsAPath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/links	0.181s
  FAIL
  ```
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:ced98c14f957e315c01daf917f5551622b27ad22652e6ce2e9b243843543eedb · ms:423
- 2026-10-01 · 98feab5* · exit 0 · `adr-verify --relock` · acceptance-sha256:ced98c14f957e315c01daf917f5551622b27ad22652e6ce2e9b243843543eedb · ms:0 · test-lock-sha256:5b85e4561e0def36d7b30197ebf51701c7e197075ddb68c87dc539c9516f48a8 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2xpbmtzL3J1bmUxMDhfdGVzdC5nbwlUZXN0QVVuaWNvZGVSdW5lTmV2ZXJTcGxpdHNBUGF0aAlhYzQzNzI4ZDQyNzEwZTJmZGU3NTNmMmY0MzI5Y2M4ZmU5Y2E0MWZkY2I5ZTIxNzJjMTQ4ODk0Mzc2ZTcxN2Nj · test-lock-kind:relock
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:ced98c14f957e315c01daf917f5551622b27ad22652e6ce2e9b243843543eedb · ms:326
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:8916065c4be054ff4cecacdf5af1ae40ee85918be7841bf2eae977531ce65006 · ms:569
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAUnicodeRuneNeverSplitsAPath
  --- PASS: TestAUnicodeRuneNeverSplitsAPath (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/links	0.172s
  ```
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:8916065c4be054ff4cecacdf5af1ae40ee85918be7841bf2eae977531ce65006 · ms:344
