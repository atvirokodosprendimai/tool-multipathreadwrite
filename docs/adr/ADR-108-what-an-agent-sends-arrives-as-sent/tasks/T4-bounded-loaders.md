# Task ADR-108-T4: foreign formats and body files are bounded

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** bounded `targetBytes` and `LoadBodyFiles`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `no entry point reads a file without a bound`

## Goal

`targetBytes` (both foreign compilers) and `LoadBodyFiles` refuse a file over 1 GiB by its size, naming it and
the limit, and read through a bound of the limit plus one byte, for a file that grows after its size was taken.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/applypatch.go` | edit | `targetBytes`; `maxTargetBytes` |
| `internal/plan/plan.go` | edit | `LoadBodyFiles`; `maxBodyBytes` |
| `internal/ingest/bound108_test.go` | add | the first test below |
| `internal/plan/bound108_test.go` | add | the second test below |

## Ordered Steps

1. [S1] Write the failing tests `TestForeignFormatsAreBounded` and `TestABodyFileIsBounded`; confirm RED. [proof: mutation]
2. [S2] The size refusals and bounded reads. [proof: mutation] Mutants: `targetBytes` reads unbounded; `LoadBodyFiles` reads unbounded.

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/ingest/ ./internal/plan/ -count=1 -timeout 300s -run 'TestForeignFormatsAreBounded|TestABodyFileIsBounded' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestForeignFormatsAreBounded \(' "$out" \
  && grep -qE '^--- PASS: TestABodyFileIsBounded \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/iter internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestForeignFormatsAreBounded` | `internal/ingest/bound108_test.go` | with `maxTargetBytes` set small, `targetBytes` refuses a larger file naming size and limit, and a 4 MB file past the limit costs under 256 KB; a file under the limit reads whole | — | S1, S2 |
| `TestABodyFileIsBounded` | `internal/plan/bound108_test.go` | with `maxBodyBytes` set small, `LoadBodyFiles` refuses a larger body file naming size and limit; one under the limit loads | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every foreign-format write and every `body=@` |
| 3 — the caller can discover it | the refusal names size and limit |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review |

## Mutation Log
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · targetBytes no longer refuses by size before reading: the refusal stops naming the file size · acceptance-sha256:7d92151ea3524553595730e7eab996a5593f9e8c828a5e1f5b3f6b60b2d52835
- 2026-10-01 · 98feab5* · mutant killed · exit 1 · `internal/plan/plan.go` · LoadBodyFiles no longer refuses by size before reading: the refusal stops naming the file size · acceptance-sha256:7d92151ea3524553595730e7eab996a5593f9e8c828a5e1f5b3f6b60b2d52835

## Invariants

- Exit codes keep their meanings; files under the limit behave as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 98feab5* · exit 1 · `set -o pipefail …` · acceptance-sha256:7d92151ea3524553595730e7eab996a5593f9e8c828a5e1f5b3f6b60b2d52835 · ms:204 · test-lock-sha256:2dacd9a1d483e4a7af53655fa141bf461be4de7c53941ea307dbb4bfb55b519e · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2luZ2VzdC9ib3VuZDEwOF90ZXN0LmdvCVRlc3RGb3JlaWduRm9ybWF0c0FyZUJvdW5kZWQJNmQ0ZTk1MGIyOTBiMzNkYzBhZjcxZWY2OWNjMzRlYTYxOTkyZTBmMWVkOTVmODQ1YTk4ODg1ZDhkMjQxNDJhZQpib2R5CWludGVybmFsL3BsYW4vYm91bmQxMDhfdGVzdC5nbwlUZXN0QUJvZHlGaWxlSXNCb3VuZGVkCTA3Y2Y3OTI3YjZhN2ZhODAzNjc5MzM0YWUzNzA2YjQ0MjIzODEyZGFmNWQ0NDk3NmQwN2ZhZGU0NGU0Nzk2ZmU
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
  internal/plan/bound108_test.go:14:9: undefined: maxBodyBytes
  internal/plan/bound108_test.go:15:21: undefined: maxBodyBytes
  internal/plan/bound108_test.go:16:2: undefined: maxBodyBytes
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest.test]
  internal/ingest/bound108_test.go:17:9: undefined: maxTargetBytes
  internal/ingest/bound108_test.go:18:21: undefined: maxTargetBytes
  internal/ingest/bound108_test.go:19:2: undefined: maxTargetBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest [build failed]
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan [build failed]
  FAIL
  ```
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d92151ea3524553595730e7eab996a5593f9e8c828a5e1f5b3f6b60b2d52835 · ms:356
- 2026-10-01 · 98feab5* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d92151ea3524553595730e7eab996a5593f9e8c828a5e1f5b3f6b60b2d52835 · ms:517
- 2026-10-01 · 98feab5* · exit 0 · `adr-verify --relock` · acceptance-sha256:7d92151ea3524553595730e7eab996a5593f9e8c828a5e1f5b3f6b60b2d52835 · ms:0 · test-lock-sha256:2dacd9a1d483e4a7af53655fa141bf461be4de7c53941ea307dbb4bfb55b519e · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2luZ2VzdC9ib3VuZDEwOF90ZXN0LmdvCVRlc3RGb3JlaWduRm9ybWF0c0FyZUJvdW5kZWQJNmQ0ZTk1MGIyOTBiMzNkYzBhZjcxZWY2OWNjMzRlYTYxOTkyZTBmMWVkOTVmODQ1YTk4ODg1ZDhkMjQxNDJhZQpib2R5CWludGVybmFsL3BsYW4vYm91bmQxMDhfdGVzdC5nbwlUZXN0QUJvZHlGaWxlSXNCb3VuZGVkCTA3Y2Y3OTI3YjZhN2ZhODAzNjc5MzM0YWUzNzA2YjQ0MjIzODEyZGFmNWQ0NDk3NmQwN2ZhZGU0NGU0Nzk2ZmU · test-lock-kind:relock
