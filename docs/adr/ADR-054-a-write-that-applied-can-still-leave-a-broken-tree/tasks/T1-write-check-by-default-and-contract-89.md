# Task ADR-054-T1: CLI write runs the check by default on non-prose; `--no-check`; contract §89

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** default check / `--no-check` (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `check exists then default run`, `prose skip`, `no command then no exit 2`, `explicit --check still exit 2`, `--no-check opts out`, `explicit check still runs on prose`

## Goal

After a successful apply, CLI `write` runs `check.Load`+`Run` when a command exists, `--no-check` is off, and at least one written path is not prose (`.md`, `.markdown`, `.txt`, `.rst`, `.adoc`). A markdown-only plan in a harnessed tree is Applied, exit 0, does not spawn the check, and records `applied`. A tree with no harness and no `go.mod` still exits 0. Explicit `--check` with no command stays exit 2. Explicit `--check` on prose still runs. `--check` and `--no-check` together is usage. `--dry-run` implies no check; `--check --dry-run` stays usage. MCP unchanged. `command` / `packages` byte-identical.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `writeCmd`: `--no-check`; default run when Load has a command and a written path is not prose. The selector. |
| `cmd/mrw/writecheck_test.go` | create | Red tests for default-on (code), prose skip, opt-out, bare directory, both-flags usage, explicit `--check` on prose. |
| `internal/apply/prose.go` | create | `IsProse` — the one closed extension list both arm 1 and arm 2 read. New file; `apply.go` stays T2's. |
| `scripts/contract.sh` | edit | **§89** — next free after §88. Pair default-check-red on a `.go` file (exit 3, tree kept) with a `.md` apply that does not run the check (exit 0), `--no-check` (exit 0), and a no-go.mod apply (exit 0). |

## Ordered Steps

1. [S1] Write `TestWriteRunsTheCheckByDefault` and confirm it is RED (a Go fixture write without `--check` must run the inferred/declared check). [proof: mutation]
2. [S2] Write `TestWriteOfProseDoesNotRunTheDefaultCheck`, `TestExplicitCheckStillRunsOnProse`, `TestNoCheckOptsOut`, `TestWriteWithoutACheckCommandStillApplies`, and `TestCheckAndNoCheckTogetherIsUsage` — RED until the flags and the prose skip exist. [proof: mutation]
3. [S3] Implement default check / `--no-check` / prose skip in `writeCmd`. Both flags = usage. Dry-run does not run a check unless `--check` was also passed (then usage, as today). Un-strike the Tests rows when the funcs exist. Confirm S1–S2 GREEN. Deleting the default-run call must fail S1. Deleting the prose skip must fail `TestWriteOfProseDoesNotRunTheDefaultCheck`. [proof: mutation]
4. [S4] Write §89 against a binary that still opts in `--check` and confirm it is RED, then rebuild and confirm GREEN. The `.md` half must fail if the binary runs the check on prose. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 89\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v \
    -run 'TestWriteRunsTheCheckByDefault|TestWriteOfProseDoesNotRunTheDefaultCheck|TestExplicitCheckStillRunsOnProse|TestNoCheckOptsOut|TestWriteWithoutACheckCommandStillApplies|TestCheckAndNoCheckTogetherIsUsage' 2>&1 | tee /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestWriteRunsTheCheckByDefault' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestWriteOfProseDoesNotRunTheDefaultCheck' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestExplicitCheckStillRunsOnProse' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestNoCheckOptsOut' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestWriteWithoutACheckCommandStillApplies' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestCheckAndNoCheckTogetherIsUsage' /tmp/adr054-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr054-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| ~~`TestWriteRunsTheCheckByDefault`~~ | `cmd/mrw/writecheck_test.go` | not yet written — Proposed; T1 S1 writes it | — | S1, S3 |
| ~~`TestWriteOfProseDoesNotRunTheDefaultCheck`~~ | `cmd/mrw/writecheck_test.go` | not yet written — Proposed; T1 S2 writes it | — | S2, S3 |
| ~~`TestExplicitCheckStillRunsOnProse`~~ | `cmd/mrw/writecheck_test.go` | not yet written — Proposed; T1 S2 writes it | — | S2, S3 |
| ~~`TestNoCheckOptsOut`~~ | `cmd/mrw/writecheck_test.go` | not yet written — Proposed; T1 S2 writes it | — | S2, S3 |
| ~~`TestWriteWithoutACheckCommandStillApplies`~~ | `cmd/mrw/writecheck_test.go` | not yet written — Proposed; T1 S2 writes it | — | S2, S3 |
| ~~`TestCheckAndNoCheckTogetherIsUsage`~~ | `cmd/mrw/writecheck_test.go` | not yet written — Proposed; T1 S2 writes it | — | S2, S3 |
| `§89` | `scripts/contract.sh` | Built binary: default check on `.go`, skip on `.md`, `--no-check`, no-command apply | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the six tests and §89 |
| 2 — something selects it | `writeCmd` after apply; deleting the default-run call fails S1 and §89; deleting the prose skip fails `TestWriteOfProseDoesNotRunTheDefaultCheck` |
| 3 — the caller can discover it | T4 teaches `--no-check` and the prose skip on `write --help` |
| 4 — it is used | T3's `failed_check` column; ADR-009 refused telemetry |

## Mutation Log

(empty until execute)

## Invariants

- ADR-003: exit 3 does not revert; missing demanded check is still exit 2.
- Explicit `--check --dry-run` stays usage.
- MCP write does not grow a check flag.
- `internal/check.command` and `packages` stay byte-identical.
- Prose skip records `applied`, not `check_not_run`.
- The prose list is the Decision's five extensions; do not add `.json` / empty-ext / `.rs` here.

## Risks

- A Zeus `.rs` write pays the whole-project `Check` because `packages()` cannot map it: full workspace clippy plus 3096 tests, measured 2026-09-13 at 58–107 seconds for the nextest half alone. Named in the parent Consequences; `--no-check` is the escape, not a Rust mapper.
- Inferred `go test ./...` on a dirty sibling package can fail a write that did not touch it. That is today's `--check` behaviour; do not special-case it here.

## Stop Condition

If the only way to go green is to run a check when `Load` has no command (Desktop exit 2), stop — that fork was rejected.
If the only way to go green is to spawn the check on a markdown-only plan, stop — that is the blocking review finding.

## Out of Scope

- Balance delta (T2)
- stats rendering (T3)
- Teaching (T4)
- MCP check
- A harness `covers` glob / a non-Go `packages()`

## Verification Log

(empty until execute)
