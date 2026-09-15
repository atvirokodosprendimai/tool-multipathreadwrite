# Task ADR-058-T3: a hanging ast-grep times out

**Depends-on:** T1
**Covers:** none — amendment Decision 5; not in the four-leftovers spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** 2 s `CommandContext` around `ast-grep`; §111
**Consumes:** `read.AstGrep` + CLI `--ast-grep` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a hanging ast-grep returns within 2 s`, `timeout is exit 2 and names timed out`, `timeout is not the missing-binary path`

## Goal

A present `ast-grep` that never returns is killed at 2 s. The process exits 2. The reason names `ast-grep` and `timed out`. It is not `ErrAstGrepMissing` and not zero hits.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/astgrep.go` | edit | `CommandContext` with a 2 s deadline; timeout error distinct from missing binary. |
| `cmd/mrw/astgrep_timeout_test.go` | create | Red: fake sleeps 30 s; 3 s test deadline. |
| `scripts/contract.sh` | edit | **§111**, under perl alarm. |

## Ordered Steps

1. [S1] Confirm the failing test exists. [proof: mutation]
2. [S2] Bound the subprocess. S1 GREEN. [proof: mutation]
3. [S3] §111. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v \
    -run '^TestAHangingAstGrepTimesOut$' 2>&1 | tee /tmp/adr058-t3.out \
  && grep -q '^--- PASS: TestAHangingAstGrepTimesOut' /tmp/adr058-t3.out \
  && grep -q '^# 111\. ' scripts/contract.sh \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr058-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/read/ ./cmd/mrw/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAHangingAstGrepTimesOut` | `cmd/mrw/astgrep_timeout_test.go` | hang returns within 2 s, exit 2, names timed out, not missing, not zero hits | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the named test |
| 2 — something selects it | `CommandContext` in `AstGrep`; deleting the deadline leaves S1 killed-by-test |
| 3 — the caller can discover it | the refusal; §111 |
| 4 — it is used | a hostile PATH binary; ADR-009 refuses telemetry |

## Mutation Log
- 2026-09-15 · bfbd2cc* · mutant killed · exit 1 · `internal/read/astgrep.go` · without the 2 s deadline a hanging ast-grep outlives the bound and the test kills it · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · covers:a hanging ast-grep returns within 2 s
- 2026-09-15 · bfbd2cc* · mutant killed · exit 1 · `internal/read/astgrep.go` · inverting the deadline check turns a hang into empty JSON / zero hits instead of timed out · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · covers:timeout is exit 2 and names timed out
- 2026-09-15 · bfbd2cc* · mutant killed · exit 1 · `internal/read/astgrep.go` · returning the missing-binary sentinel on a hang is the path the test forbids · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · covers:timeout is not the missing-binary path

## Invariants

- Missing binary stays `ErrAstGrepMissing` / exit 2 / names ast-grep.
- Zero hits stay exit 1 and name the pattern.
- apply/plan/seen/check/state stay byte-identical vs merge-base.
- `--grep` is still unbounded (it is in-process).

## Risks

- A legitimate ast-grep scan slower than 2 s is refused. Same bound as the hook; revisit with evidence.

## Stop Condition

Stop if the bound must be a flag or must change `--grep`.

## Out of Scope

- Bounding `--grep` (permanent: boundary: in-process).
- Bundling ast-grep (permanent: boundary: ADR-058 Decision 2).

## Verification Log
- 2026-09-15 · bfbd2cc* · exit 1 · `set -o pipefail …` · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · ms:3478 · test-lock-sha256:834b46e542e20cbaad8b853b34aa54b7d83c8b2e64634760652e7072f958ccc9 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF90aW1lb3V0X3Rlc3QuZ28JVGVzdEFIYW5naW5nQXN0R3JlcFRpbWVzT3V0CTdlNzQxZDljNjJjYTJmNzY0ZjFiYzM1OGM4MzE2ZmI5MTAzZWYyNTZhYTkyMTkwODRiYzg1ZTNhMmRmM2RkODY
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAHangingAstGrepTimesOut
      astgrep_timeout_test.go:40: ast-grep still running at 3 s; the 2 s bound never fired
  --- FAIL: TestAHangingAstGrepTimesOut (3.08s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	3.254s
  FAIL
  ```
- 2026-09-15 · bfbd2cc* · exit 0 · `set -o pipefail …` · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · ms:3305
- 2026-09-15 · bfbd2cc* · exit 0 · `set -o pipefail …` · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · ms:2716
- 2026-09-15 · bfbd2cc* · exit 0 · `set -o pipefail …` · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · ms:2605
- 2026-09-15 · bfbd2cc* · exit 0 · `set -o pipefail …` · acceptance-sha256:43b02241b66bd8b8bb6bb1160bab54124784f9780722df7047554a4664483a1c · ms:2599
