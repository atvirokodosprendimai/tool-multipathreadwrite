# Task ADR-164-T4: a check that places some files runs those and names the rest

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `check.Result.OutsideScope`; `check.Run`'s whole-check flag, set by `writer.Verify` for a removed or renamed-away `.go` or module file; per-path judgement in `packages()` and `command()` in `internal/check/check.go`; the `outside the scoped check:` line on both surfaces; `internal/mcp/tools.go`, `internal/mcp/schema.go` and `internal/mcp/testdata/legacy_golden.jsonl` as T2 consumes them
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a file outside is named and its package runs`, `an unplaceable .go runs everything`, `a removed package runs everything`, `a module file runs everything`, `a directory runs everything`, `files and dirs templates report nothing outside`, `the CLI prints it`, `the MCP text keeps the count`, `the receipt keys are listed`

## Goal

Decision 4, with the tests that pin today's all-or-nothing fallback for non-Go FILES rewritten and every other fallback test unchanged.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `packages()` judges per path (Decision 4's list); `command()` scopes over placed packages for a `{packages}`-only template; `Result.OutsideScope` |
| `internal/check/partial164_test.go` | add | the engine test |
| `internal/check/check_test.go` | edit | `TestScopeDerivation`'s "non-Go falls back" and "a directory and a non-Go" cases, which hold a non-Go FILE beside a mapped path, now expect the scoped command and `OutsideScope`; "prose beside a package" (a directory) is unchanged |
| `internal/check/adr061_stress_test.go` | edit | the oracle: `{a.go, a.rs}` leaves `unmappedPool`, and a {packages}-only template over it scopes with `a.rs` outside; mixed templates over it scope with nothing outside |
| `internal/writer/flow.go` | edit | `Verify` passes `check.Run` a whole-check flag when the write removed or renamed away a `.go` or module file; `CheckPaths` is unchanged, so no missing name reaches `confine` (ADR-042) (Codex, re-checks of the record) |
| `cmd/mrw/main.go` | edit | `reportCheck` prints the line |
| `cmd/mrw/partial164_test.go` | add | the CLI line and JSON key |
| `internal/mcp/tools.go` | edit | `checkReport` prints the line; the terminal phrase under the ceiling keeps the count (ADR-032) |
| `internal/mcp/schema.go` | edit | a description for `check.outside_scope` (`TestEveryOutputSchemaPropertyIsDescribed`); the `elided` description's order |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated for the new description, as ADR-152 did for `check.hint` |
| `internal/mcp/partial164_test.go` | add | the MCP line, and the count under a small ceiling |
| `docs/receipts.txt` | edit | `check outside_scope`, `mcp_write check.outside_scope`, `write check.outside_scope` |
| `scripts/contract.sh` | edit | §276 |
| `README.md`, `AGENTS.md` | edit | the sentence on `{packages}` falling back |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest (a module with packages `a` and `b`, `b` imported by `a`; `scoped_check` `go test {packages}`: `a/a.go` + `run.sh` runs `go test ./a` with `OutsideScope` `[run.sh]`; `run.sh` alone runs the whole check; `a/a.go` + `go.mod` runs the whole check; `a/a.go` + a directory named `z.go` (a `.go` path `packages()` cannot place; a present `testdata/x.go` maps today and is not one) runs the whole check; `a/a.go` + the unlink of `go.sum` runs the whole check through the writer (`writer.Verify`), not by handing `go.sum` to `Run`; `a/a.go` + the rename of `b/b.go` to `b/c.go` likewise; `mrw check go.sum` for a missing `go.sum` is still refused (ADR-042); `a/a.go` + `b` as a directory holding no package (what `CheckPaths` hands for an unlinked `b/b.go`) runs the whole check; `go test {packages} && lint {files}` over `a/a.go` + `run.sh` scopes with nothing outside; `{files}` alone is unchanged; called through `command()` and through `Run`, the missing-path refusal of ADR-042 kept as its own case), TestTheCLIPrintsTheFilesOutsideTheScopedCheck (the line and `check.outside_scope`; also `mrw check a/a.go run.sh`), TestTheMCPTextKeepsTheCountOutsideTheScopedCheck (the line; under a ceiling small enough to elide the list, the terminal phrase still says `1 file(s) were outside it`). [proof: acceptance]
2. [S2] `packages()` per path, Decision 4's list. [proof: mutation]
3. [S3] `command()` scopes over placed packages for a `{packages}`-only template; `Run` sets `OutsideScope`. [proof: mutation]
4. [S4] Rewrite the two `TestScopeDerivation` cases and the stress oracle; leave every other fallback test as it is and run them. [proof: acceptance]
5. [S5] `reportCheck` and `checkReport` print the line; the ceiling's terminal phrase; schema descriptions and the golden; `docs/receipts.txt`; README and AGENTS. [proof: mutation]
6. [S6] Contract §276: through the built binary in a scratch module, a `.go` + `.sh` write's receipt shows the scoped command and `outside the scoped check: run.sh`; the pair: a `.sh`-only write shows the whole command; §15, §86 and §113 still pass. [proof: human: the executor runs ./scripts/contract.sh once and reads the §276, §15, §86 and §113 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ -count=1 -timeout 300s -run 'TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest \(' "$out" \
  && go test ./internal/check/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestTheCLIPrintsTheFilesOutsideTheScopedCheck|TestTheMCPTextKeepsTheCountOutsideTheScopedCheck|TestScopeDerivation|TestRandomisedCommandMatchesTheFilesScopeOracle|TestPackagesOnlyNonGoStillFallsBack|TestMixedPlaceholdersFallBackWhenUnmapped|TestFilesOnlyWithNoPathsStillFallsBack|TestAPresentUnplaceableInRootScopeStillFallsBack|TestAScopeOutsideTheRootIsRefusedNotFallenBackTo|TestASlowUnscopedCheckSaysHowToScopeIt|TestNoShippedReceiptFieldDisappears|TestNoShippedMCPReceiptFieldDisappears|TestEveryOutputSchemaPropertyIsDescribed|TestALegacyResultIsUnchangedByTheModernPath' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheCLIPrintsTheFilesOutsideTheScopedCheck \(' "$out" \
  && grep -qE '^--- PASS: TestTheMCPTextKeepsTheCountOutsideTheScopedCheck \(' "$out" \
  && grep -qE '^--- PASS: TestScopeDerivation \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep -x 'write check.outside_scope' docs/receipts.txt > /tmp/adr164-t4-key.txt \
  && grep '^# 276\. ' scripts/contract.sh > /tmp/adr164-t4-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestScopeDerivation` | `internal/check/check_test.go` | two cases rewritten, the rest unchanged | none | S4 |
| `TestRandomisedCommandMatchesTheFilesScopeOracle` | `internal/check/adr061_stress_test.go` | the oracle under the new rule | none | S4 |
| `TestMixedPlaceholdersFallBackWhenUnmapped` | `internal/check/check_test.go` | an empty map still falls back (ADR-061) | none | — |
| `TestAPresentUnplaceableInRootScopeStillFallsBack` | `internal/check/check_test.go` | a directory alone still falls back (ADR-042) | none | — |
| `TestASlowUnscopedCheckSaysHowToScopeIt` | `internal/check/slow152_test.go` | ADR-152's hint unchanged | none | — |

Planned, named without backticks until they exist: TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest in `internal/check/partial164_test.go` (S1, S2, S3), TestTheCLIPrintsTheFilesOutsideTheScopedCheck in `cmd/mrw/partial164_test.go` (S1, S5), TestTheMCPTextKeepsTheCountOutsideTheScopedCheck in `internal/mcp/partial164_test.go` (S1, S5).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Result.OutsideScope`, the per-path judgement |
| 2 — something selects it | `check.Run`, which every write's check and `mrw check PATH` call; contract §276 drives the built binary |
| 3 — the caller can discover it | the line on every receipt; the schema description; README and AGENTS |
| 4 — it is used | nothing measures this; met by this session (2026-10-20: 300 s, exit 3) |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- An unplaceable `.go`, a directory that holds no package, a module file, a write whose paths place nothing, `--full`, and a template with `{files}` or `{dirs}` run exactly what they run today, apart from a mixed template over a partial map, which now scopes with every path in `{files}`.
- ADR-042's refusal of a missing path is unchanged.
- No receipt key is renamed or removed (ADR-111).

## Risks

- A Go test that reads an edited non-Go file is skipped; the file is named (the record's Risks).

## Out of Scope

- Mapping `.sh` or testdata to the Go packages that read them (permanent: boundary: the record's Out of Scope)

## Stop Condition

Stop and bring it to Zy if a contract row other than the ones this task names goes red, or if keeping `TestAPresentUnplaceableInRootScopeStillFallsBack` green needs a rule Decision 4 does not state.
