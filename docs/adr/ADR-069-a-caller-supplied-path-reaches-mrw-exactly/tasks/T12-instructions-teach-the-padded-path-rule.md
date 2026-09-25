# Task ADR-069-T12: `mrw instructions` teaches the padded-path rule; contract §140

**Depends-on:** T11
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** one line in `guide.CLI`
**Consumes:** T1's refusal and T5's attached-value refusal
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `instructions teach the rule`, `the binary prints it`, `no engine file changes`

## Goal

The v1.25.1 field test of the downloaded Windows asset found that `mrw instructions` said nothing about the padded-path refusal or the `--` escape, so a caller with only the binary learned the rule at the refusal. ADR-063 promised the read side's traps there. The fix: one line in `guide.CLI` — a padded path goes after `--`; an attached value ending in whitespace is passed as its own argument.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | the line |
| `internal/guide/guide_test.go` | edit | the test |
| `scripts/contract.sh` | edit | §140 |

## Ordered Steps

1. [S1] Write the test; confirm RED. [proof: mutation]
2. [S2] Add the line; GREEN. [proof: mutation]
   Mutant: the line teaches the trim as the rule.
3. [S3] §140 through the binary. [proof: mutation]

## Acceptance

```bash
set -o pipefail
grep -q '^# 140\. ' scripts/contract.sh \
  && go test ./internal/guide/ -count=1 -v -run 'TestInstructionsTeachThePaddedPathRule' 2>&1 | tee /tmp/adr069-T12.out \
  && grep -q '^--- PASS: TestInstructionsTeachThePaddedPathRule ' /tmp/adr069-T12.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T12.out \
  && ./scripts/contract.sh > /tmp/adr069-T12-contract.out 2>&1 \
  && grep -q '^  PASS  instructions teach that a padded path goes after --' /tmp/adr069-T12-contract.out \
  && grep -q '^  PASS  and they do not teach the trim as the rule' /tmp/adr069-T12-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/guide)" ] \
  && go vet ./internal/guide/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInstructionsTeachThePaddedPathRule` | `internal/guide/guide_test.go` | `CLI()` carries the `--` rule and the attached-value rule | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §140 |
| 2 — something selects it | `mrw instructions` prints `CLI()` verbatim |
| 3 — the caller can discover it | it is the discovery path |
| 4 — it is used | a Windows field tester read `instructions` looking for exactly this and did not find it |

## Verification Log
(empty until execute)
- 2026-09-25 · 5cf522c* · exit 1 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:351 · test-lock-sha256:41fc4078025abb0d152786584f3ef8d78cbfc352e64740f1a5d41532e6e6e152 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdENMSUNvbnRhaW5zU2hhcmVkQW5kVGhlT3BlcmF0b3JUcmFwcwk2NTZiMWY2MzU0ZDRjYzY0NmEzNjAxZDBkYWUzY2Y4MmNkNzlhMGNhMGY2MTM0NzQ0ZTJkODU1ZjQwY2FlNzU1CmJvZHkJaW50ZXJuYWwvZ3VpZGUvZ3VpZGVfdGVzdC5nbwlUZXN0Q0xJVGVhY2hlc0Fsd2F5c0FuZEFQbGFuCTg3NmQ4ZDVlZWU2MGIwODk5ZjI0ODhmYzdhOGU3YzM4YzQ5Mzk1MWY0ZTRmODcxNDcxZGIzMGM1ZjBmN2U3MTEKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RDTElUZWFjaGVzVGhlUmVhZFNpZGUJMDFmYzAzYTE1MGMyMTdjMTExZDNlZjE2MDAyMDliMjNlYmY2YTBlZTY1NzA4ZjRiMGNiMmIxNzAzN2QxZjlkMwpib2R5CWludGVybmFsL2d1aWRlL2d1aWRlX3Rlc3QuZ28JVGVzdEV2ZXJ5U3VyZmFjZUNvbnRhaW5zVGhlU2hhcmVkU2VudGVuY2VzCWM0ZTMxNjkzMWUzNWYxNWJlODljZmU0ZWEzYTRmNWQ1ODZiNzI3NDE0OTE5ODYwYjNkYTMxMzY3MTUzY2Y3NzAKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RJbnN0cnVjdGlvbnNUZWFjaFRoZVBhZGRlZFBhdGhSdWxlCTg2NTgyMDMwNjU2ZTg2ZTdkN2U3MGZiY2Y3NGU1ZDNhMTRiNDQxN2I3NGE2YzY1NjA3NTVjY2I4ZTc5NTI4YTcKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3ROb1NlcnZlZFRleHRQcm9taXNlc05vdGhpbmdXcml0dGVuT25BbnlGYWlsdXJlCTBjMzUzN2I1OTc0ODZmZmY2OTM5NTEwZDI2NjgzOTc1NDIzYmI2MzgwZjEzOTJjNGVmZjRjNGFhYTY3MGI5ZGQKYm9keQlpbnRlcm5hbC9ndWlkZS9ndWlkZV90ZXN0LmdvCVRlc3RTaGFyZWRTYXlzQUZhaWxlZENvbW1pdElzUmVwb3J0ZWRQYXJ0aWFsCWRmOTFhNzIyZTU1ODExMDcyNzFhMDkxZDFhZDEyMzYwMWJlZDlhMjBhNzA5NWJmM2ZmODc1M2Q1OGJiNDE4ZGM
  ```
  --- last 8 line(s) of stdout
  === RUN   TestInstructionsTeachThePaddedPathRule
      guide_test.go:150: mrw instructions lacks "goes after --"
      guide_test.go:150: mrw instructions lacks "mrw read -- 'x '"
      guide_test.go:150: mrw instructions lacks "--files-from 'list '"
  --- FAIL: TestInstructionsTeachThePaddedPathRule (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide	0.166s
  FAIL
  ```
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:66978
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:47094
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:33570
- 2026-09-25 · 5cf522c* · exit 0 · `set -o pipefail …` · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · ms:34992

## Mutation Log
(empty until execute)
- 2026-09-25 · 5cf522c* · mutant killed · exit 1 · `internal/guide/guide.go` · the line teaches the trim as the rule: the test and §140 go red · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · covers:instructions teach the rule
- 2026-09-25 · 5cf522c* · mutant killed · exit 1 · `internal/guide/guide.go` · the attached-value half is dropped: the test and §140 go red through the binary · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · covers:the binary prints it
- 2026-09-25 · 5cf522c* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:ae19b52fe51bf780264464bf10d65e2164a177747684f36c680459915bef550f · covers:no engine file changes

## Invariants

- Every earlier `guide` test stays green; `mrw instructions` still exits 0 with no flags.

## Risks

- None known.

## Out of Scope

- Teaching the rule in the MCP handshake: the MCP tools take specs as JSON and never pass through the argument parser (permanent: boundary: ADR-068 keeps a spec exactly on that surface; citation: `internal/mcp/tools.go`)

## Stop Condition

Stop if the line needs a format change or an exit-code change.
