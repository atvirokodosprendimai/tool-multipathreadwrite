# Task ADR-145-T1: The install check migrates nothing

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `startsWithoutState`, the condition in `main`
**Consumes:** `state.Migrate` (existing)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `version, -v, --version and instructions as the first argument skip the legacy state migration and every other command still runs it`

## Goal

Decisions 1 to 3 of the record, with a test that fails before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/startup.go` | add | `startsWithoutState` |
| `cmd/mrw/main.go` | edit | `main` skips the migration for it |
| `cmd/mrw/startup145_test.go` | add | the predicate's table |
| `scripts/contract.sh` | edit | §249 |

## Ordered Steps

1. [S1] Write `TestVersionAndInstructionsMigrateNothing`. Confirm RED.
2. [S2] `startsWithoutState` and `migrateLegacyState`, which `main` calls. Mutants: the predicate answers false for every argument list (killed by the table test); the wrapper ignores the predicate (killed by `TestTheLegacyMigrationRunsForOtherCommandsOnly`, which the package step of the fence runs). [proof: mutation]
3. [S3] Contract §249 drives the built binary on a legacy `.mrw/seen`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestVersionAndInstructionsMigrateNothing' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestVersionAndInstructionsMigrateNothing \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && grep -q '^# 249\. ' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestVersionAndInstructionsMigrateNothing` | `cmd/mrw/startup145_test.go` | `version`, `-v`, `--version` and `instructions` skip the migration; `read`, `write`, `stats`, `mcp`, a flag before the verb and no argument do not | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `startsWithoutState` |
| 2 — something selects it | `main`, which every start passes |
| 3 — the caller can discover it | `mrw version` prints no "moved" line and writes nothing |
| 4 — it is used | contract §249 drives the built binary; no telemetry (ADR-009) |

## Invariants

- Every other command migrates exactly as before.
- Exit codes and output of the four commands are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a command in the four needs the migrated state.

## Out of Scope

- A flag before the verb (permanent: boundary: the migration runs before the parse)

## Mutation Log

## Verification Log
- 2026-10-10 · f9f5453* · exit 1 · `set -o pipefail …` · acceptance-sha256:00e8f33910611613108ad097aa7332da8f5d2256ebeab1cbeeca4e7cc97eb976 · ms:505 · test-lock-sha256:614a5efaf3d37da26120db3841032e4daca1ac9d1b49cb524331acf81db1a381 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdGFydHVwMTQ1X3Rlc3QuZ28JVGVzdFZlcnNpb25BbmRJbnN0cnVjdGlvbnNNaWdyYXRlTm90aGluZwkzNGJjYzE2NzY2MzA5YjE3ZTgxNDI3YWUyNTkwNzI5ZGVkYWFhNzM0MmE4YmE0ZmQ3ODY5NmE0YTUyYzhkYTA1
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test]
  cmd/mrw/startup145_test.go:26:13: undefined: startsWithoutState
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [build failed]
  FAIL
  ```
