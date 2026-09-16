# ADR-061: `{files}` still scopes when `packages()` cannot map

**Status:** Accepted
**Accepted:** 2026-09-16 by M — *"so work on 054"*
**Date:** 2026-09-16
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-054, docs/adr/BACKLOG.md
**Governs:** `internal/check/check.go`, `scripts/contract.sh`
**Enforced-by:** `internal/check/check_test.go::TestFilesPlaceholderDoesNotNeedPackages`
**Invalidates:** ADR-054 — the clause of its Decision reading "When `packages()` cannot map (non-Go paths), the whole-project `Check` runs"
**Served-path change:** a declared `scoped_check` whose template contains `{files}` and not `{packages}` runs on a `.rs` (or other non-Go) path instead of falling back to `check`.
**Notes:** Zeus field report 2026-09-15, filed under From ADR-054: a `.rs` edit with both `scoped_check` and `check` declared still ran `echo FULL`, because `command()` scopes only when `packages()` maps every path and `packages()` is Go-only. BACKLOG named the lever and said it needs a new record. M, 2026-09-16: *"so work on 054"* after quality-harness v2.99.5. Execute on this Accept. No Rust `packages()`. No harness `covers` glob. `{packages}`-only templates still fall back. Mixed `{packages}`+`{files}` with an empty map still falls back (`go test` with no args is a silent PASS).

## Context

**The class this record governs.** Every `command()` choice between `ScopedCheck` and `Check`. Enumerated 2026-09-16 with

```
git ls-files internal/check/check.go
```

One tracked file. Members left out: `packages()` itself (still Go-only), `Load`, `Run`'s timeout, MCP (ADR-044: no check).

**Why this is a record.** ADR-003 abandoned the scoped form whenever any changed path was not a Go file, because `{packages}` cannot place a `.rs` and a scoped run that quietly omits a changed file is worse than a slow complete one. ADR-054 restated that fallback as the remaining Zeus cost. `{files}` names every path the caller wrote — it cannot omit — so the omission rationale does not apply to a `{files}`-only template. Leaving the fallback in place trains a Rust checkout to declare `scoped_check` and still pay workspace clippy+nextest.

## Existing Primitives Audit

- **`command()` / `packages()`.** Reused. `command()` currently returns ScopedCheck only when `len(packages()) > 0`. `{files}` is already substituted in that arm (`TestFilesPlaceholder`). This record adds the arm that fires when the map is empty.
- **`shellArgs`.** Reused. Paths with spaces / `;` / `$(…)` stay quoted (ADR-003).
- **A Rust `packages()` / a `covers` glob.** Audited and rejected (ADR-054 Out of Scope, permanent: boundary).

## Decision

**When `packages()` returns nothing and the path list is non-empty, `command()` still returns `ScopedCheck` if that template contains `{files}` and does not contain `{packages}`.** `{files}` expands to those paths, quoted. The hunk is scoped.

A `{packages}`-only template still falls back to `Check`. A mixed template (`{packages}` and `{files}`) still falls back when the map is empty: substituting an empty `{packages}` into `go test {packages}` runs `go test` with no args, which tests the current package and can PASS while omitting the file the caller named — the silent omission ADR-003 exists to prevent. An empty path list (`mrw check --full`, empty working set) still falls back: scoping `{files}` with nothing named would run a command that covers no file.

`packages()` stays Go-only. No `covers` glob. `--no-check` stays the escape when the project declared only `check`.

## Alternatives Considered

- **Invent a Rust `packages()`.** Rejected: ADR-054 permanent boundary; a language table is a parser's cousin.
- **A harness `covers` glob.** Rejected: same.
- **Always substitute `{files}` even when `{packages}` is also in the template and the map is empty.** Rejected: empty `{packages}` is a silent PASS.
- **Leave the fallback; teach Zeus `--no-check`.** Rejected: they already declared `scoped_check`; the tool dropped it.

## Component / Boundary Impact

`internal/check.command` gains one arm. `packages()` is unchanged. No new JSON key. CLI flags unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `command()` `{files}`-only + empty map | return scoped substitution | T1 | `Run`; CLI write default check; `mrw check` |
| contract §113 | next free after §112 | T1 | `adr-verify`, CI |

Exit codes unchanged.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `{files}`-only scopes when unmapped (T1) | T1 | — | Yes — check command selection |

## Implementation

See `docs/adr/ADR-061-files-scope-does-not-need-packages/tasks/README.md`.

## Consequences

- **Positive:** a Zeus-shaped `scoped_check: "cargo clippy -- {files}"` (no `{packages}`) runs on a `.rs` write instead of the workspace `check`. `{files}` cannot omit a named path.
- **Negative:** a `{packages}`-only `scoped_check` in a Rust tree still falls back. Callers who copied a Go template keep paying `check`. Mixed templates keep falling back.
- **Neutral:** Go `{files}` with a non-empty map is unchanged. MCP still does not run a check.

## Out of Scope

- A non-Go `packages()` (permanent: boundary: a language table is a parser's cousin)
- A harness `covers` glob (permanent: boundary: ADR-054 closed this)
- Emptying `{packages}` in a mixed template (permanent: boundary: go test with no args is a silent PASS)
- MCP `mrw_write` running a check (permanent: fact: ADR-044 stays two tools; citation: file `docs/adr/ADR-044-mcp-cargo-stays-two-tools.md:7`)
- Growing the prose list so `.rs` skips the check (permanent: boundary: ADR-054 closed the list)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A mixed template silently PASSes if we emptied `{packages}` | High if mixed were scoped | High | Mixed + empty map still falls back |
| Zeus `scoped_check` is `{packages}`-shaped and still falls back | Med | Med | Named in Consequences; they must write `{files}` |
| `{files}` with a phantom path | Low | Med | `packages()` / `confine` already refuse a miss; `{files}` uses the caller's paths after confine |

## Rollback

Revert the branch. Non-Go paths fall back to `Check` again.

## Stress suite

Measured 2026-09-16 against this Decision, not `command()`. New files only — T1's leftover lock hashed every `Test*` in `check_test.go`.

- Layer 1: `internal/check/adr061_stress_test.go` — `refShellArg` walks bytes against an inert charset string; `shellArg` uses `ContainsFunc` + a switch. Named examples plus `FuzzShellArg` (2.3M execs / 8s, 45 new interesting, no disagreement).
- Layer 2: same file, `TestRandomisedCommandMatchesTheFilesScopeOracle` — 400 iters × 12 seeds (`MRW_SEED` 1,2,3,7,11,13,17,19,29,41,61,99). Template × path class (empty / allGo / unmapped). Mapped class is a root-level `.go` fixture so the oracle substitutes `.` and never calls `packages()`. Left out: `{files_extra}`; nested package trees; Zeus's real JSON.
- Layer 3: `internal/adversarial/adr061_test.go` — 120 iters × 4 seeds through the built binary (`mrw check` + `--json` / `--full`). `TestAFilesOnlyWriteOnRustRunsTheScopedCheck` is the Zeus-shaped write: `{files}`-only on `a.rs` prints `echo SCOPED a.rs`, not `echo FULL`. Template, not their checkout file.
- Found: none.
- Hand mutants against that suite, 4 of 4 killed: files-only arm deleted; `!{packages}` guard dropped; `len(paths)>0` dropped; arm fires on any non-empty path list. Baseline green, each compiled, restore from a pre-sweep copy (not `git checkout --`: uncommitted stress files).

## Follow-ups

- Relock ADR-054 T1–T4 with quality-harness 2.99.5 so Go test bodies hash; `§89` in that Tests table stays UNPROVEN and `done` stays refused until the section row leaves the table.
