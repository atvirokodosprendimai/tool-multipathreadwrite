# ADR-130: a walk does not enter a nested repository

**Status:** Accepted
**Accepted:** 2026-10-06 by Zy — "Nested repo like git" among the items chosen for this round, then "go next, properly, build a spec/adr and execute. i want complete stability and completeness". The record's text was drafted after those answers.
**Date:** 2026-10-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-111, ADR-116, ADR-122
**Invalidates:** None — it amends ADR-116 Decision 1's nested-repository clause for a repository found inside a checkout; a root that is no checkout keeps that clause
**Governs:** `internal/read/walk.go`, `internal/mcp/mcp.go`, `internal/guide/guide.go`, `cmd/mrw/main.go`, `docs/receipts.txt`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/read/nested130_test.go::TestANestedRepositoryIsNotEntered`
**Served-path change:** inside a git checkout, `--grep`, `grep`, `--ast-grep` and `ast_grep` no longer enter a directory below the walk's start that holds its own `.git`, as git does not; the `-- skipped:` line and `mrw_read`'s `skipped` count it as `nested`. A path the caller names inside one is walked, by that repository's rules; `--no-ignore` and `no_ignore` walk them all.

## Context

ADR-116 Decision 1 applies a nested repository's own `.gitignore` inside it (`noteNested`, `internal/read/walk.go:383`) and walks it. git does not descend into an untracked nested repository: `git status` lists `nested/` and `git grep` searches none of its files. A tree with vendored clones, a `go mod vendor` replaced by checkouts, or worktrees under the root therefore had mrw searching files git treats as another project's. BACKLOG "From the Windows peers" carries it, and the 2026-10-06 plan chose it ("Nested repo like git").

A root that is no checkout has no outer repository to compare against; there each repository found is its own, as ADR-116 says, and a repository nested inside one of those is that one's nested repository.

**Audit of the class** — *a finder that walks or judges discovered paths*: `mrw read --grep 'noteNested\(' internal/read/` names the walk (`walkDir`, two calls) and the ast-grep judge (`hitJudge.skip`, ADR-122); `read.Walk` serves `--grep` on the CLI and `grep` on `mrw_read`, and `hitJudge` serves `--ast-grep` and `ast_grep`. Four surfaces, two code paths.

## Existing Primitives Audit

- **`noteNested` / `ignored`** (ADR-116) — find a `.git` and keep a nested checkout's rules; the same test decides "nested" here.
- **`WalkSkipped` / `SkipNote` / `skipCounts`** (ADR-116) — the counts and the `-- skipped:` line; `nested` joins them, counted as a pruned directory is: not when a named path entered it.
- **`underGit`** (ADR-122) — the `.git` directory itself, pruned whatever the flags; unchanged.

## Decision

1. **A directory the walk meets that holds a `.git` (a directory or a gitfile), below a checkout, is not entered**: the walk prunes it and counts it `nested`. "Below a checkout" means the root is inside one, or the directory lies inside a repository the walk already found below a root that is none.
2. **A path the caller names is walked**, inside a nested repository included, by that repository's own rules (ADR-116's clause, kept for named paths), and a nested repository below it is again not entered.
3. **`--no-ignore` / `no_ignore` walk every repository**, as they walk every ignored file.
4. **`--ast-grep` / `ast_grep` drop a hit inside a nested repository** the walk would not have entered, and count it the same way (ADR-122).
5. **The count is a receipt key**: `skipped.nested` on `mrw_read` (ADR-111: appended to `docs/receipts.txt`), "N nested repositor(ies), not entered" on the `-- skipped:` line, and the `no_ignore` description says a walk does not enter one. `mrw_read` carries no output schema (ADR-023), so its keys are described in that input description and the `-- skipped:` line, not in `schema.go`.

## Alternatives Considered

- **Keep walking with the nested repository's rules** (ADR-116 as written) — rejected by Zy, 2026-10-06: the files are another project's, and git agrees.
- **Read `.gitmodules` and walk submodules** — rejected: `git grep` does not recurse into submodules by default either, and a submodule's working tree holds a gitfile, which Decision 1 already treats as a repository.

## Component / Boundary Impact

`internal/read` (engine change owned by this record: the walk's pruning and the judge), `internal/mcp` (schema text). `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `--grep`, `grep`, `--ast-grep`, `ast_grep` | a nested repository inside a checkout is not entered | T1 | CLI and MCP callers |
| `-- skipped:` line | "N nested repositor(ies), not entered" | T1 | CLI and MCP readers |
| `mrw_read` receipt | `skipped.nested` — a new key (ADR-111) | T1 | MCP callers |
| `docs/receipts.txt`, `internal/mcp/mcp.go`, `internal/mcp/testdata/legacy_golden.jsonl` | the key; the `no_ignore` description; the golden that pins it | T1 | ADR-111's test; the era test |
| `scripts/contract.sh` | §233 | T1 | CI Linux |
| `AGENTS.md`, `README.md` | the walk paragraph | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a search inside a checkout covers what git covers; vendored clones and worktrees below the root stop answering.
- **Negative:** a caller who relied on mrw searching a nested clone must name it, or pass `--no-ignore`.
- **Neutral:** a root that is no checkout walks each repository it holds, as before.

## Out of Scope

- `core.ignorecase` (permanent: boundary: mrw parses no git config, ADR-116; BACKLOG "From the Windows peers" carries it)
- A nested repository a caller names as the start (permanent: boundary: Decision 2 — naming a path is how a caller says "this one")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a `.git` file that is not a gitfile prunes a directory | Low | its files go unsearched, counted `nested` | the count names it on every read; `--no-ignore` walks it |

## Rollback

Revert T1: nested repositories are walked again. No persistent state.

## Follow-ups

- None — the record carries no open follow-up.
