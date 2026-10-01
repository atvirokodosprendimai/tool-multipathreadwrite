# Task ADR-109-T4: a FIFO config is refused

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `.quality-harness.json` is opened through `regular.Open`
**Consumes:** `regular.Open` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a FIFO config is refused`

## Goal

`check.Load` read `.quality-harness.json` with `os.ReadFile`, so a FIFO there hung `mrw check` and every write whose check was due (measured on v1.37.2: killed at 5 s). It opens through `regular.Open` and refuses one, exit 2.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `Load` |
| `internal/check/fifo109_unix_test.go` | add | the test |
| `scripts/contract.sh` | edit | §207 |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAFIFOConfigIsRefusedAtOnce`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal; contract §207 through `$MRW`. Mutant: `Load` back to `os.ReadFile`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ -count=1 -timeout 300s -run 'TestAFIFOConfigIsRefusedAtOnce' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAFIFOConfigIsRefusedAtOnce \(' "$out" \
  && grep -q '^# 207\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFIFOConfigIsRefusedAtOnce` | `internal/check/fifo109_unix_test.go` | `Load` of a root whose `.quality-harness.json` is a FIFO returns an error naming it and the not-a-regular-file refusal within 5 s; a regular config still loads; unix only, where FIFOs exist | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write, compile or check that opens the file |
| 3 — the caller can discover it | the refusal names the file and why |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex re-review of #304 and the 2026-10-01 measurement |

## Mutation Log
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/check/check.go` · readConfig back to a blocking open: a FIFO config hangs the check · acceptance-sha256:88093b26bb0cd8e5d523d66d8bea1ef7ef59f3cadb9c5bcdb5cc83764adfca0e
- 2026-10-01 · 1fb4932* · mutant killed · exit 1 · `internal/check/check.go` · readConfig back to a blocking open: a FIFO config hangs the check · acceptance-sha256:8ca26262138487915a45483a536a737aa52e0bc6c2e0fa74fbba8af4d7eee9e1

## Invariants

- A regular file opens and reads as before; refusal texts and exit codes are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-109 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 92450b2* · exit 1 · `set -o pipefail …` · acceptance-sha256:88093b26bb0cd8e5d523d66d8bea1ef7ef59f3cadb9c5bcdb5cc83764adfca0e · ms:5445 · test-lock-sha256:fee2113dbb9d9e1d05f5014c041fa53786b79d426954f5975a69c550d98db044 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvY2hlY2svZmlmbzEwOV91bml4X3Rlc3QuZ28JVGVzdEFGSUZPQ29uZmlnSXNSZWZ1c2VkQXRPbmNlCTRjOGJiMDA1MWU5YmVjMGJkY2YwMmI3NTYxYTIyOTJmMTNmNTI1Mjc1MDhlOTc1MjU1NWY1YTFhMDRlZDQyM2Q
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAFIFOConfigIsRefusedAtOnce
      fifo109_unix_test.go:39: Load blocked on a FIFO config
  --- FAIL: TestAFIFOConfigIsRefusedAtOnce (5.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	5.178s
  FAIL
  ```
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:88093b26bb0cd8e5d523d66d8bea1ef7ef59f3cadb9c5bcdb5cc83764adfca0e · ms:601
- 2026-10-01 · 1fb4932* · exit 0 · `set -o pipefail …` · acceptance-sha256:8ca26262138487915a45483a536a737aa52e0bc6c2e0fa74fbba8af4d7eee9e1 · ms:742
