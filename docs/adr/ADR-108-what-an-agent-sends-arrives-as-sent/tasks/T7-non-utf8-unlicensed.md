# Task ADR-108-T7: non-UTF-8 content is served unlicensed over MCP

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** non-UTF-8 content is served unlicensed over MCP
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a host never acknowledges text JSON changed`

## Goal

`interleave` issues no checkpoint for a slice that is not valid UTF-8 and prints a line saying why, so a host never acknowledges text JSON changed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/ack.go` | edit | `interleave` |
| `internal/mcp/utf108_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test(s) `TestNonUTF8ContentIsServedUnlicensedOverMCP`; confirm RED. [proof: mutation]
2. [S2] No checkpoint for invalid UTF-8; a named line instead. Mutant: the UTF-8 check removed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestNonUTF8ContentIsServedUnlicensedOverMCP' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestNonUTF8ContentIsServedUnlicensedOverMCP \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/iter internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestNonUTF8ContentIsServedUnlicensedOverMCP` | `internal/mcp/utf108_test.go` | through the real `mrw_read` handler, a Latin-1 file is served with no checkpoint, a line naming why, and nothing held; a file holding a literal U+FFFD is served with its checkpoint as before | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write or acknowledgement that reaches it |
| 3 — the caller can discover it | the refusal or the receipt names it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review |

## Mutation Log
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/mcp/ack.go` · the UTF-8 check removed: a Latin-1 read is checkpointed and licensed · acceptance-sha256:d2ab22b9291d6f430340e9e564adb7bcc1f3f08caed807dc678e855fb2b3f351

## Invariants

- Exit codes keep their meanings; inputs the defect did not touch behave as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 98feab5* · exit 1 · `set -o pipefail …` · acceptance-sha256:d2ab22b9291d6f430340e9e564adb7bcc1f3f08caed807dc678e855fb2b3f351 · ms:406 · test-lock-sha256:02b37cf8bace5bba15b2a9ffb7fba3ad2d8e7e042d06f354a9f67b4f5df4e157 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC91dGYxMDhfdGVzdC5nbwlUZXN0Tm9uVVRGOENvbnRlbnRJc1NlcnZlZFVubGljZW5zZWRPdmVyTUNQCTQyNDA4NDYyYzI1YmQ4ZWU0NjA2Y2E5MDg4OThkYjI1MjIyMDE5MjllNWY1NzE3M2JhMmNiMzg2YWU0MjNkOGQ
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
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:d2ab22b9291d6f430340e9e564adb7bcc1f3f08caed807dc678e855fb2b3f351 · ms:410
- 2026-10-01 · 98feab5* · exit 0 · `adr-verify --relock` · acceptance-sha256:d2ab22b9291d6f430340e9e564adb7bcc1f3f08caed807dc678e855fb2b3f351 · ms:0 · test-lock-sha256:02b37cf8bace5bba15b2a9ffb7fba3ad2d8e7e042d06f354a9f67b4f5df4e157 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC91dGYxMDhfdGVzdC5nbwlUZXN0Tm9uVVRGOENvbnRlbnRJc1NlcnZlZFVubGljZW5zZWRPdmVyTUNQCTQyNDA4NDYyYzI1YmQ4ZWU0NjA2Y2E5MDg4OThkYjI1MjIyMDE5MjllNWY1NzE3M2JhMmNiMzg2YWU0MjNkOGQ · test-lock-kind:relock
