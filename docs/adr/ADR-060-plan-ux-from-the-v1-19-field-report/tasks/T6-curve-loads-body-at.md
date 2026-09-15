# Task ADR-060-T6: the curve scorer loads `body=@` after parse

**Depends-on:** T4
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `curve.ScoreTrial` calls `plan.LoadBodyFiles`
**Consumes:** T4 `LoadBodyFiles`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `curve calls LoadBodyFiles`, `missing body=@ is RefusedParse`

## Goal

`ScoreTrial` loads `body=@` after parse and after copying the fixture, before Apply — the same call CLI and MCP already make. A load error is `RefusedParse`, not a harness error. A `create body=@path` whose file is missing or rooted no longer applies an empty body.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/curve/score.go` | edit | `LoadBodyFiles(scratch, hunks)` after `copyTree`; load error → `RefusedParse`. Comment matches the call. |
| `internal/curve/score_bodyfile_test.go` | create | Red: missing `body=@` is `RefusedParse`; present create is not. Dedicated file so the first-red lock does not bind sibling scorer tests. |
| `docs/adr/ADR-060-plan-ux-from-the-v1-19-field-report.md` | edit | Governs `internal/curve/score.go`; Decision 4 names the scorer. |

## Ordered Steps

1. [S1] Write the failing test. [proof: mutation]
2. [S2] Call `LoadBodyFiles` on the apply tree. S1 GREEN. [proof: mutation]
3. [S3] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/curve/ -count=1 -v \
  -run 'TestScoreTrialLoadsABodyAtPath' 2>&1 | tee /tmp/adr060-t6.out \
  && grep -q '^--- PASS: TestScoreTrialLoadsABodyAtPath' /tmp/adr060-t6.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr060-t6.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/curve/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/read \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestScoreTrialLoadsABodyAtPath` | `internal/curve/score_bodyfile_test.go` | `create body=@missing` is `RefusedParse` (skipping the load applies an empty create and scores Miss); a present `body=@src.txt` is Miss because it wrote elsewhere, not a parse refusal | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `ScoreTrial` → `plan.LoadBodyFiles`. Deleting the call leaves S1 red |
| 3 — the caller can discover it | `curve score` drives `ScoreTrial`; no new flag |
| 4 — it is used | T4 left curve unwired on purpose; this is the remaining production Parse-then-Apply caller. No telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-14 · fef9763* · mutant killed · exit 1 · `internal/curve/score.go` · LoadBodyFiles is a no-op so TestScoreTrialLoadsABodyAtPath/missing must go red · acceptance-sha256:9567f81ac51c974df9562d4f98e0ca9a675270b2ee1ef6db2fc9c4201ae1f47a · covers:curve calls LoadBodyFiles
- 2026-09-14 · fef9763* · mutant killed · exit 1 · `internal/curve/score.go` · LoadBodyFiles error is swallowed so TestScoreTrialLoadsABodyAtPath/missing must go red · acceptance-sha256:9567f81ac51c974df9562d4f98e0ca9a675270b2ee1ef6db2fc9c4201ae1f47a · covers:missing body=@ is RefusedParse

## Invariants

- CLI and MCP wiring from T4 is unchanged.
- Integer `body=N` plans still score.
- A load error is an Outcome (`RefusedParse`), never `Score{}` harness breakage — a client's path is data (same as Parse).
- `rooted.IsRooted` lives in `LoadBodyFiles`, not here.
- Engine go/no-go: `internal/plan`, `internal/apply`, `internal/seen`, `internal/check`, `internal/state`, `internal/read` identical against the merge-base.

## Risks

- Loading against the trial directory instead of the copied fixture tree looks for `body=@` next to the manifest, not next to the target file.

## Stop Condition

Stop if scoring `body=@` needs a second root besides the tree Apply uses.

## Out of Scope

- A contract.sh row (permanent: fact: curve is not `$MRW`; citation: file `cmd/curve/main.go`).
- Teaching AGENTS.md (permanent: boundary: T5 already taught `body=@path`; the scorer is not a write surface).
- Letting `replace` / `insert-*` parse with an empty Body because `BodyFile` is set (deferred: T7).

## Verification Log
_(tool-written)_
- 2026-09-14 · fef9763* · exit 1 · `set -o pipefail …` · acceptance-sha256:9567f81ac51c974df9562d4f98e0ca9a675270b2ee1ef6db2fc9c4201ae1f47a · ms:591 · test-lock-sha256:e83ef9d88f3856428cb703f21d40a8835872cbaf713fe514cb1cb074d3d4e540 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2N1cnZlL3Njb3JlX2JvZHlmaWxlX3Rlc3QuZ28JVGVzdFNjb3JlVHJpYWxMb2Fkc0FCb2R5QXRQYXRoCTFiMGZmNTdlNzdmM2E5ODgzN2EzZmE0YjIyNjFhMGYyN2FiMmIyMjZlNTc0ZjQxZWY4ZDVmNjMxM2JhMjBjMzYKYm9keQlpbnRlcm5hbC9jdXJ2ZS9zY29yZV9ib2R5ZmlsZV90ZXN0LmdvCW1pc3NpbmcJOTU2YjRmODEwNGFhNGQ2MjI5NTUxZjk5NjE4ZDM1MWQ0ODA3NmFhZDdmYmU1YjcyZDIyNjg5YzI3MzkzMGU0Ywpib2R5CWludGVybmFsL2N1cnZlL3Njb3JlX2JvZHlmaWxlX3Rlc3QuZ28JcHJlc2VudAlhOTg2NDY1ODc5YzMyZDk0YTlmNmNjMGQxNmUyZTQwNzVjZGVmZjYyZDE3NjVlMmZhYTQ5MTg0YTg4YzRlNWQz
  ```
  --- last 10 line(s) of stdout
  === RUN   TestScoreTrialLoadsABodyAtPath
  === RUN   TestScoreTrialLoadsABodyAtPath/missing
      score_bodyfile_test.go:25: create body=@missing.txt scored miss, want refused_parse — skipping LoadBodyFiles applies an empty create
  === RUN   TestScoreTrialLoadsABodyAtPath/present
  --- FAIL: TestScoreTrialLoadsABodyAtPath (0.01s)
      --- FAIL: TestScoreTrialLoadsABodyAtPath/missing (0.00s)
      --- PASS: TestScoreTrialLoadsABodyAtPath/present (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/curve	0.238s
  FAIL
  ```
- 2026-09-14 · fef9763* · exit 0 · `set -o pipefail …` · acceptance-sha256:9567f81ac51c974df9562d4f98e0ca9a675270b2ee1ef6db2fc9c4201ae1f47a · ms:1538
- 2026-09-14 · fef9763* · exit 0 · `set -o pipefail …` · acceptance-sha256:9567f81ac51c974df9562d4f98e0ca9a675270b2ee1ef6db2fc9c4201ae1f47a · ms:510
