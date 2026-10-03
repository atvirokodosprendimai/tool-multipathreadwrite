# ADR-124: {dirs} is the portable scope

**Status:** Accepted
**Accepted:** 2026-10-03 by Zy — "Document {files}, add {dirs} (Recommended)", on the gap list's "check maps packages for Go only" and the plan amendment of 2026-10-02. The record's text was drafted after that answer.
**Date:** 2026-10-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-054, ADR-061, ADR-094
**Invalidates:** None — {packages} and {files} keep their meanings
**Governs:** `internal/check/check.go`, `AGENTS.md`, `README.md`, `internal/guide/guide.go`, `scripts/contract.sh`
**Enforced-by:** `internal/check/check_test.go::TestDirsPlaceholderNamesEachEditedDirectoryOnce`
**Served-path change:** `scoped_check` takes `{dirs}`: the directory of each edited file, or a named directory, as `./dir` (`.` for the root), each once, sorted. A template of `{files}` or `{dirs}` without `{packages}` runs scoped for any language; a step command holding `{dirs}` is refused as one holding the other two is.

## Context

`{packages}` maps a `.go` file to its package and anything else makes the scoped check fall back to the full one; `{files}` already runs scoped for any language (ADR-061), but a runner that takes directories — pytest, jest, a cargo crate — had to be given files or the whole tree. The plan's amendment named "check maps packages for Go only"; Zy chose documentation plus one language-neutral token over per-ecosystem rules.

**Audit of the class** — *a token `scoped_check` substitutes*: `mrw read --grep 'placeholders|NewReplacer\("\{' internal/check/check.go` — `placeholders` (what `Placeholder` refuses in a step) and `command`'s two replacers.

## Existing Primitives Audit

- **`command` / `packages` / `shellArgs`** — the expansion and its quoting, reused.
- **`placeholders` / `Placeholder`** (ADR-094) — the list a step command is refused by; `{dirs}` joins it.

## Decision

1. **`{dirs}`** expands to the edited files' directories (a named directory to itself), `./dir` or `.`, once each, sorted, quoted as `{files}` is.
2. **A `{files}` or `{dirs}` template without `{packages}` runs scoped** for any language; with `{packages}` and a Go map, all three are substituted; otherwise the full check runs, as before.
3. **A step holding `{dirs}` is refused** (ADR-094), and every surface that teaches the step rule names all three tokens.
4. **The docs say `{packages}` is Go's and `{files}`/`{dirs}` are portable.**

## Alternatives Considered

- **Map Rust and Node in `{packages}`** — rejected by Zy's answer: per-ecosystem rules mrw would have to keep right.
- **Docs only** — rejected: a directory-taking runner had no scope at all.

## Component / Boundary Impact

Owns `internal/check` (engine): one token and one helper. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `scoped_check` | `{dirs}` | T1 | project configs |
| the step rule on every surface | names `{dirs}` | T1 | callers |
| `scripts/contract.sh` | §222 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a Python, JavaScript or Rust project gets a scoped check from one line of config.
- **Negative:** a step command that held the literal text `{dirs}` is now refused.
- **Neutral:** Go projects see no change.

## Out of Scope

- Per-ecosystem package mapping (permanent: boundary: Zy chose the neutral token)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a directory scope omits a test elsewhere that the edit breaks | Medium | Medium | the same trade `{files}` makes; `mrw check --full` and the project's own `check` stay available |

## Rollback

Revert the task; `{dirs}` stays literal and a `{files}`-less template falls back.

## Follow-ups

- None — the record carries no open follow-up.
