# Task ADR-108-T6: acknowledging a non-regular file returns at once

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** acknowledging a non-regular file returns at once
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `acknowledging a non-regular file returns at once`

## Goal

`currentSHA` opens without blocking and refuses a descriptor that is not a regular file, so acknowledgement of a file swapped for a FIFO or a device returns at once and licenses nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/ack.go` | edit | `currentSHA` |
| `internal/mcp/fifo108_unix_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAcknowledgingANonRegularFileReturnsAtOnce`; confirm RED. [proof: mutation]
2. [S2] Non-blocking open and a regular-file check on the descriptor. Mutant: the regular-file check removed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAcknowledgingANonRegularFileReturnsAtOnce' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAcknowledgingANonRegularFileReturnsAtOnce \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/iter internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAcknowledgingANonRegularFileReturnsAtOnce` | `internal/mcp/fifo108_unix_test.go` | `currentSHA` of a FIFO returns an error within 5 s with no writer, and of a regular file its digest; unix only, where FIFOs exist | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write or acknowledgement that reaches it |
| 3 — the caller can discover it | the refusal or the receipt names it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review |

## Mutation Log
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/mcp/ack.go` · the regular-file check removed: a FIFO is digested as empty · acceptance-sha256:5529d9fd6a0afdda5d904ab32968f09ea80afb7b58506d1178f6317bef2fe75f

## Invariants

- Exit codes keep their meanings; inputs the defect did not touch behave as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 98feab5* · exit 1 · `set -o pipefail …` · acceptance-sha256:5529d9fd6a0afdda5d904ab32968f09ea80afb7b58506d1178f6317bef2fe75f · ms:332 · test-lock-sha256:1fc7a2fcd992ff2d3490af18685302aca2ee30538f29ff8c583c688aa92ab3c4 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9maWZvMTA4X3VuaXhfdGVzdC5nbwlUZXN0QWNrbm93bGVkZ2luZ0FOb25SZWd1bGFyRmlsZVJldHVybnNBdE9uY2UJZWY3MmZjYWQ3YWVmZmIzOGIxZjU5MGYyZDYyMjhjYTEyMjNhNjlhYmYyODI1NDI5ZGJlOGEzYzNhOTQwNDE5OQ
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
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:5529d9fd6a0afdda5d904ab32968f09ea80afb7b58506d1178f6317bef2fe75f · ms:374
- 2026-10-01 · 98feab5* · exit 0 · `adr-verify --relock` · acceptance-sha256:5529d9fd6a0afdda5d904ab32968f09ea80afb7b58506d1178f6317bef2fe75f · ms:0 · test-lock-sha256:1fc7a2fcd992ff2d3490af18685302aca2ee30538f29ff8c583c688aa92ab3c4 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9maWZvMTA4X3VuaXhfdGVzdC5nbwlUZXN0QWNrbm93bGVkZ2luZ0FOb25SZWd1bGFyRmlsZVJldHVybnNBdE9uY2UJZWY3MmZjYWQ3YWVmZmIzOGIxZjU5MGYyZDYyMjhjYTEyMjNhNjlhYmYyODI1NDI5ZGJlOGEzYzNhOTQwNDE5OQ · test-lock-kind:relock
