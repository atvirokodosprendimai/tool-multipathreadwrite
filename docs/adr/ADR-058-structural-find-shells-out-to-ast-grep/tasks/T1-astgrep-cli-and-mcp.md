# Task ADR-058-T1: `--ast-grep` / `ast_grep` share one primitive

**Depends-on:** none
**Covers:** F-5, F-11, F-12, F-13, F-15, UC2-S1, UC2-S2, UC2-S3
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `read.AstGrep`; CLI `--ast-grep`; MCP `ast_grep`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `missing binary is exit 2 and names ast-grep`, `two finders are two sources`, `hits serve through read.Run`, `zero hits name the pattern`, `MCP is the same primitive`

## Goal

CLI `--ast-grep` and MCP `ast_grep` call `read.AstGrep`, serve through `read.Run`, and refuse the same way.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/astgrep.go` | add | Shell out; map JSON hits to specs. Selected by CLI and MCP. |
| `cmd/mrw/main.go` | edit | Flag, two-sources, missing-binary, empty-hits. |
| `internal/mcp/tools.go` | edit | `ast_grep` field; `astGrepSpecs`. |
| `internal/mcp/mcp.go` | edit | Schema property. |
| `cmd/mrw/astgrep_test.go` | edit | CLI red tests. |
| `internal/mcp/astgrep_test.go` | edit | MCP red tests. |

## Ordered Steps

1. [S1] Confirm the failing tests for `Covers:` IDs exist. [proof: mutation]
2. [S2] Implement `AstGrep` and wire CLI + MCP. S1 GREEN. [proof: mutation]
3. [S3] Scoped tests, `gofmt`, `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ ./internal/mcp/ -count=1 -v \
  -run 'TestMissingAstGrepIsUsageAndNamesTheBinary|TestGrepAndAstGrepTogetherAreUsage|TestAstGrepServesRangesThroughRead|TestAPresentAstGrepWithZeroHitsIsNotTheMissingBinaryPath|TestAstGrepObservesOnlyServedLines|TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli|TestGrepAndAstGrepTogetherOnMcpAreUsage' 2>&1 | tee /tmp/adr058-t1.out \
  && grep -q '^--- PASS: TestMissingAstGrepIsUsageAndNamesTheBinary' /tmp/adr058-t1.out \
  && grep -q '^--- PASS: TestGrepAndAstGrepTogetherAreUsage' /tmp/adr058-t1.out \
  && grep -q '^--- PASS: TestAstGrepServesRangesThroughRead' /tmp/adr058-t1.out \
  && grep -q '^--- PASS: TestAPresentAstGrepWithZeroHitsIsNotTheMissingBinaryPath' /tmp/adr058-t1.out \
  && grep -q '^--- PASS: TestAstGrepObservesOnlyServedLines' /tmp/adr058-t1.out \
  && grep -q '^--- PASS: TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli' /tmp/adr058-t1.out \
  && grep -q '^--- PASS: TestGrepAndAstGrepTogetherOnMcpAreUsage' /tmp/adr058-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr058-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/read/ ./cmd/mrw/ ./internal/mcp/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMissingAstGrepIsUsageAndNamesTheBinary` | `cmd/mrw/astgrep_test.go` | exit 2, names `ast-grep`, not unknown-flag | F-12, UC2-S2 | S1, S2 |
| `TestGrepAndAstGrepTogetherAreUsage` | `cmd/mrw/astgrep_test.go` | two sources | F-13, UC2-S3 | S1, S2 |
| `TestAstGrepServesRangesThroughRead` | `cmd/mrw/astgrep_test.go` | hits serve through Run | F-5, UC2-S1 | S1, S2 |
| `TestAPresentAstGrepWithZeroHitsIsNotTheMissingBinaryPath` | `cmd/mrw/astgrep_test.go` | zero hits name the pattern | F-12 | S1, S2 |
| `TestAstGrepObservesOnlyServedLines` | `cmd/mrw/astgrep_test.go` | license is served lines | F-15 | S1, S2 |
| `TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli` | `internal/mcp/astgrep_test.go` | MCP names ast-grep, not ignored | F-11 | S1, S2 |
| `TestGrepAndAstGrepTogetherOnMcpAreUsage` | `internal/mcp/astgrep_test.go` | MCP two sources | F-13 | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the seven tests |
| 2 — something selects it | `cmd.IsSet("ast-grep")` and `a.AstGrep`; deleting either leaves CLI or MCP red |
| 3 — the caller can discover it | flag Usage; MCP schema `ast_grep` |
| 4 — it is used | nothing measures this yet |

## Mutation Log
- 2026-09-15 · 9ff8e35* · mutant killed · exit 1 · `internal/read/astgrep.go` · missing binary must name ast-grep so the caller installs the right CLI · acceptance-sha256:2909b17434a830fca148c07a7b190b94c3b067bd55766aeee971eb459481b989 · covers:missing binary is exit 2 and names ast-grep

## Invariants

- `--grep` stays regex Walk.
- apply/plan/seen/check/state stay byte-identical vs merge-base.
- Walk observes nothing; Run observes served lines.

## Risks

- Fake JSON ≠ real ast-grep. Mapper is the documented slice.

## Stop Condition

Stop if the implementation needs a write-time parser or a fifth Walk rule.

## Out of Scope

- Contract rows (T2).
- Bundling the binary.

## Verification Log
- 2026-09-15 · 9ff8e35* · exit 1 · `set -o pipefail …` · acceptance-sha256:4c97614b5451dd5f5b1003fa4751147a7a0be7134fad9a44ac60a7ad04652241 · ms:2143 · test-lock-sha256:be13a0487374acb131de2e1aefeeb23707fdb9c19c32c345e4a0c560687c3e50 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcF90ZXN0LmdvCVRlc3RBUHJlc2VudEFzdEdyZXBXaXRoWmVyb0hpdHNJc05vdFRoZU1pc3NpbmdCaW5hcnlQYXRoCWYzYWQzY2Y1ZmEwNTQzMDEyZjRhYjlkNjA1MDhmZDAzODJmZDc0NTQyMDU3OTJmY2FlNTdlYTY3NWQ1NDQ5ZjkKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0QXN0R3JlcE9ic2VydmVzT25seVNlcnZlZExpbmVzCWE2NWNiM2Q1MmE5ZjgyNmE3NjJmNmFmZmM2ODFhZTNmNTQyNDAxNDllYTNjN2VjNWUxNGMyM2NkN2RjZmVlYTkKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0QXN0R3JlcFNlcnZlc1Jhbmdlc1Rocm91Z2hSZWFkCTNlMzMwNjcyNjQwMDJkM2EyYzEyN2M2MTk0ZDgyNDU2MjYzMzU4MWQ4NzFmNmUzODlkMzhlYTJhZTVkMDlmMTUKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0R3JlcEFuZEFzdEdyZXBUb2dldGhlckFyZVVzYWdlCTU1MjI0MWZiYmMwNzlhODQ1YWMzNzZiNzRhOTIyM2JmZThkMWExZGQwZWYzNDE2YTc1MTRjMzBjOTIxZWVlYzgKYm9keQljbWQvbXJ3L2FzdGdyZXBfdGVzdC5nbwlUZXN0TWlzc2luZ0FzdEdyZXBJc1VzYWdlQW5kTmFtZXNUaGVCaW5hcnkJM2M2ZTUwZjQ3YmQ1NzBkMTE4ZWM1MGY1M2ZmZTM0ZmFlMWMwYTVjOGU3YmMzZjNmZTNlZDUxNmM1MzAwYTAyYQpib2R5CWludGVybmFsL21jcC9hc3RncmVwX3Rlc3QuZ28JVGVzdEFzdEdyZXBPbk1jcElzVGhlU2FtZVByaW1pdGl2ZUFzVGhlQ2xpCWZmZTQzZmY3Nzc2OTY4YmZiMGVkYzYxNDUyODY1ZTJjYWMyNmU4ZDA1OGUwOTk4NmJhYjVmMzZmNGUzMDkxNjkKYm9keQlpbnRlcm5hbC9tY3AvYXN0Z3JlcF90ZXN0LmdvCVRlc3RHcmVwQW5kQXN0R3JlcFRvZ2V0aGVyT25NY3BBcmVVc2FnZQk4YWU2OWYwMDkxMTc3ZDMyN2MwNzgxMTAzZGMyZjJkNTBiYWQyZGY5ZDBhYTY4MzFlZDYyNGRhMjQyMzdjZGU1
  ```
  --- last 10 line(s) of stdout (of 21 after folding 21 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	1.779s
  === RUN   TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli
  --- PASS: TestAstGrepOnMcpIsTheSamePrimitiveAsTheCli (0.00s)
  === RUN   TestGrepAndAstGrepTogetherOnMcpAreUsage
  --- PASS: TestGrepAndAstGrepTogetherOnMcpAreUsage (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.726s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.435s [no tests to run]
  ```
- 2026-09-15 · 9ff8e35* · exit 0 · `set -o pipefail …` · acceptance-sha256:2909b17434a830fca148c07a7b190b94c3b067bd55766aeee971eb459481b989 · ms:2353
