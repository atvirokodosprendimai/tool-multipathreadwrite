# Task ADR-109-T3: the compilers, body files and MCP ask the descriptor

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the descriptor decides for `targetBytes`, `LoadBodyFiles` and the MCP sites
**Consumes:** `regular.Open` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the compilers, body files and MCP ask the descriptor`

## Goal

`ingest.targetBytes` and `plan.LoadBodyFiles` let the descriptor decide whether the file is regular, through `regular.Open`, keeping their refusal texts; `mcp.currentSHA` and `mcp.countFileLines` (ADR-108) open through the same function instead of their own copies.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/applypatch.go` | edit | `targetBytes` |
| `internal/plan/plan.go` | edit | `LoadBodyFiles`, `readBounded` |
| `internal/mcp/ack.go` | edit | `currentSHA` |
| `internal/mcp/tools.go` | edit | `countFileLines` |
| `internal/ingest/fifo109_unix_test.go` | add | the compiler's test |
| `internal/plan/fifo109_unix_test.go` | add | the body file's test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestForeignTargetFIFOIsRefusedByItsDescriptor`, `TestABodyFileFIFOIsRefusedByItsDescriptor`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: the descriptor check in `regular.Open` removed, which `targetBytes`' test must see (its package no longer imports `os`); `readBounded` back to `os.Open`. The MCP sites keep ADR-108's tests (T6, T8), which go red without the descriptor check. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/ingest/ ./internal/plan/ -count=1 -timeout 300s -run 'TestForeignTargetFIFOIsRefusedByItsDescriptor|TestABodyFileFIFOIsRefusedByItsDescriptor' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestForeignTargetFIFOIsRefusedByItsDescriptor \(' "$out" \
  && grep -qE '^--- PASS: TestABodyFileFIFOIsRefusedByItsDescriptor \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/state internal/lines internal/rooted internal/seen \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestForeignTargetFIFOIsRefusedByItsDescriptor` | `internal/ingest/fifo109_unix_test.go` | `targetBytes` of a FIFO returns `regular.ErrNotRegular` within 5 s, its text unchanged; unix only, where FIFOs exist | — | S1, S2 |
| `TestABodyFileFIFOIsRefusedByItsDescriptor` | `internal/plan/fifo109_unix_test.go` | `LoadBodyFiles` with a FIFO body file refuses within 5 s with `is not a regular file`, wrapping `regular.ErrNotRegular`; unix only, where FIFOs exist | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write, compile or check that opens the file |
| 3 — the caller can discover it | the refusal names the file and why |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex re-review of #304 and the 2026-10-01 measurement |

## Mutation Log
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/regular/regular.go` · the descriptor check removed: targetBytes reads a FIFO as an empty target · acceptance-sha256:30e477ffd538194e708c73fc05e1480eb394e6c0c3d751380a1bc480dbe876b2
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/plan/plan.go` · readBounded back to os.Open: a FIFO body file blocks the write · acceptance-sha256:30e477ffd538194e708c73fc05e1480eb394e6c0c3d751380a1bc480dbe876b2

## Invariants

- A regular file opens and reads as before; refusal texts and exit codes are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-109 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 92450b2* · exit 1 · `set -o pipefail …` · acceptance-sha256:30e477ffd538194e708c73fc05e1480eb394e6c0c3d751380a1bc480dbe876b2 · ms:203 · test-lock-sha256:73f5b763d2613efc0b4464128b5db2315b788a5741746f25973f10c3bdf04be0 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvaW5nZXN0L2ZpZm8xMDlfdW5peF90ZXN0LmdvCVRlc3RGb3JlaWduVGFyZ2V0RklGT0lzUmVmdXNlZEJ5SXRzRGVzY3JpcHRvcgk4MTgxOWEzZmM3NzA3NmE5YzE5NGYxNTczN2EzMTI5ZTJiNDk4ZGNmNGI1YmJmNjViYjc2NWFjYjViNzU1ZGJjCmJvZHkJaW50ZXJuYWwvcGxhbi9maWZvMTA5X3VuaXhfdGVzdC5nbwlUZXN0QUJvZHlGaWxlRklGT0lzUmVmdXNlZEJ5SXRzRGVzY3JpcHRvcglhOTdiOTM1Y2JlMWExNWUwMGFjMmQwMDhkYWNkNzIyZDdiZTRmYjU3NjBmMjc1Mjc5ODViNWU0ZTRkMTE1YzU3
  ```
  --- last 4 line(s) of stdout
  github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular: no non-test Go files in ~/GolandProjects/wt-109/internal/regular
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest [build failed]
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan [build failed]
  FAIL
  ```
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:30e477ffd538194e708c73fc05e1480eb394e6c0c3d751380a1bc480dbe876b2 · ms:677
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:30e477ffd538194e708c73fc05e1480eb394e6c0c3d751380a1bc480dbe876b2 · ms:647
