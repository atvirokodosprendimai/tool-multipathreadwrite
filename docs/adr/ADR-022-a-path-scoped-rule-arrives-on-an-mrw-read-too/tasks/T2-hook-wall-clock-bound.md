# Task ADR-022-T2: the hook returns within 2 s and still exits 0

**Depends-on:** none
**Covers:** F-7, UC4-S1, UC4-S2
**Estimated scope:** S
**Owner:** unassigned
**Produces:** 2 s SIGALRM in `main`; §110
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a hanging run() returns within 2 s`, `timeout still exits 0`

## Goal

`.claude/hooks/rules-on-read.py` `main` arms a 2 s SIGALRM (where the platform has one) and `_exit(0)` if it fires, so a pathological matcher cannot outlive the turn. Exit 0 stays unconditional.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `.claude/hooks/rules-on-read.py` | edit | `signal.alarm(2)` before `run`; handler `_exit(0)`. |
| `cmd/mrw/ruleshook_test.go` | edit | Red: hang injected in `run()`, 3 s test deadline. |
| `scripts/contract.sh` | edit | **§110**, under perl alarm. |

## Ordered Steps

1. [S1] Confirm the failing tests for `Covers:` IDs exist. [proof: mutation]
2. [S2] Arm the alarm in `main`. S1 GREEN. [proof: mutation]
3. [S3] §110. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 110\. ' scripts/contract.sh \
  && grep -q 'signal.alarm(2)' .claude/hooks/rules-on-read.py \
  && go test ./cmd/mrw/ -count=1 -v \
    -run 'TestTheRulesHookReturnsWithinTheWallClockBound|TestTheRulesHookStillExitsZeroAfterTimeout' 2>&1 | tee /tmp/adr022-t2.out \
  && grep -q '^--- PASS: TestTheRulesHookReturnsWithinTheWallClockBound' /tmp/adr022-t2.out \
  && grep -q '^--- PASS: TestTheRulesHookStillExitsZeroAfterTimeout' /tmp/adr022-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr022-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRulesHookReturnsWithinTheWallClockBound` | `cmd/mrw/ruleshook_test.go` | hang in `run()` returns within 2 s | F-7, UC4-S1 | S1, S2 |
| `TestTheRulesHookStillExitsZeroAfterTimeout` | `cmd/mrw/ruleshook_test.go` | timeout still exits 0 | UC4-S2 | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | `main` arms SIGALRM before `run`; deleting it leaves S1 killed-by-test |
| 3 — the caller can discover it | module comment; §110 |
| 4 — it is used | this checkout's PostToolUse; ADR-009 refuses telemetry |

## Mutation Log
- 2026-09-15 · 9ff8e35* · mutant killed · exit 1 · `.claude/hooks/rules-on-read.py` · without the 2 s alarm a hanging run() outlives the bound and the test kills it · acceptance-sha256:629093fec424fb0d5575713998c9fcf5b836029e7b2847af3826896f926624d3 · covers:a hanging run() returns within 2 s

## Invariants

- Exit 0 is still unconditional.
- `seg_match` stays.
- §55 still wraps every hook child in perl alarm.

## Risks

- No SIGALRM on Windows. Tests skip without python3; contract is Linux.

## Stop Condition

Stop if the bound must take the turn down (non-zero exit).

## Out of Scope

- Changing the four-tool matcher.
- Windows SIGALRM (already deferred: BACKLOG rules-hook-on-Windows).

## Verification Log
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:629093fec424fb0d5575713998c9fcf5b836029e7b2847af3826896f926624d3 · ms:4688
- 2026-09-15 · 9ff8e35* · exit 1 · `set -o pipefail …` · acceptance-sha256:629093fec424fb0d5575713998c9fcf5b836029e7b2847af3826896f926624d3 · ms:26 · test-lock-sha256:145c0f2662f713ac041008751878320bf044346a009899213988271c1809147a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcnVsZXNob29rX3Rlc3QuZ28JVGVzdFRoZVBhdGhTY29wZWRSdWxlc0hvb2tEZWxpdmVyc09uQW5NcndSZWFkCTgyMWI0OGE5OWIxZWVmN2UzNWM1Nzk0YTExM2ViZjFlZWQ0MzkyNzVlYzM0ZGIzMzRjZGM3Njk3OGU3YzYxNDgKYm9keQljbWQvbXJ3L3J1bGVzaG9va190ZXN0LmdvCVRlc3RUaGVSdWxlc0hvb2tSZXR1cm5zV2l0aGluVGhlV2FsbENsb2NrQm91bmQJMzVhMzg5MDc5YTJhNDM0MDA1MjMxMjVlNzIyYmViN2Q3MDRkZGJlMThkZTI4MzI0NWRjNDAyYjg0ZDcxZTdlYwpib2R5CWNtZC9tcncvcnVsZXNob29rX3Rlc3QuZ28JVGVzdFRoZVJ1bGVzSG9va1N0aWxsRXhpdHNaZXJvQWZ0ZXJUaW1lb3V0CTQ3YjIyY2IxOGRlMGVmZGE0Njg5YjlmZTgyZGZlZDkwYzIwOWNiOTllNjliODFhMjE4OWQ0ODc5N2FiMmI1YWU
  ```
  ```
