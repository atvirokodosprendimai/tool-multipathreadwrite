# ADR-122: ast-grep serves what the walk serves

**Status:** Accepted
**Accepted:** 2026-10-03 by Zy — the plan amendment of 2026-10-02 ("ADR-122 ast-grep walks what mrw's walk found — one finder walk, ignore rules shared; arms BACKLOG 'From ADR-116'"), and on 2026-10-03, with ast-grep 0.45.3's flags in hand, "ast-grep walks, mrw filters (Recommended)". The record's text was drafted after those answers and not shown to Zy before execution.
**Date:** 2026-10-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-074, ADR-096, ADR-116
**Invalidates:** BACKLOG "From ADR-116"'s note that `--no-ignore` does not reach `--ast-grep`
**Governs:** `internal/read/astgrep.go`, `internal/read/walk.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `AGENTS.md`, `scripts/contract.sh`
**Enforced-by:** `internal/read/astgrep122_test.go::TestAnAstGrepHitTheWalkWouldSkipIsDroppedAndCounted`
**Served-path change:** `--ast-grep` and `ast_grep` serve the files `--grep` would: a hit in a file the walk's rules skip — an ignored file, a file in an ignored directory or a nested checkout's ignored path, a binary file, anything under `.git` — is dropped unless its file was named, and what mrw drops is counted in the same `-- skipped:` line and `skipped` key; what ast-grep's own `.gitignore` handling pruned never reaches mrw and is not counted. A hidden file ast-grep used to skip is served. `--no-ignore` / `no_ignore` now reach `--ast-grep` and turn every ignore source off.

## Context

Two finders served different files. ADR-116 taught `--grep`'s walk to skip what `.gitignore` and `.git/info/exclude` ignore and to skip binaries, natively; ast-grep walks with its own rules: measured with ast-grep 0.45.3 (`npx -p @ast-grep/cli ast-grep run --help`, 2026-10-03), it also skips hidden files and reads `.ignore` files and `core.excludesFile`, none of which mrw reads, and `--no-ignore` was refused beside `--ast-grep`. BACKLOG "From ADR-116" recorded the disagreement; the plan's amendment armed it.

**Audit of the class** — *a finder that discovers files to serve*: `mrw read --grep 'read\.(Walk|AstGrep)\(' --exclude '*_test.go' cmd/ internal/` — `read.Walk` (CLI `--grep`, MCP `grep`) and `read.AstGrep` (CLI `--ast-grep`, MCP `ast_grep`), one call each per surface.

## Existing Primitives Audit

- **The walk's judgement** (`walker.ignored`, `noteNested`, `skipCounts`, ADR-116) — reused through a `hitJudge` over the same walker, not reimplemented.
- **`SkipNote` / `WalkSkipped`** — the footer and the `skipped` key `--grep` already reports.
- **ast-grep's `--no-ignore <source>`** — `hidden`, `dot`, `exclude`, `global`, `parent`, `vcs` (0.45.3).

## Decision

1. **ast-grep keeps the rules mrw also reads, and drops the rest.** It is run with `--no-ignore hidden --no-ignore dot --no-ignore global`, so it sees hidden files and ignores neither `.ignore` files nor `core.excludesFile`, while still pruning what `.gitignore`, its parents and `info/exclude` ignore — it never opens what both skip, which keeps it inside its 2 s bound on a tree with `node_modules`.
2. **Every hit passes the walk's judgement.** A hit is dropped when the walk would not have served its file — ignored by the checkout it is in (a nested checkout's own rules below it), inside an ignored directory, binary, or under a `.git` the walk would not enter — unless the file was named; a named directory is walked and its rules still apply, as `--grep` does. What is dropped is counted, once per path, in the same `-- skipped:` line and `skipped` key.
3. **`--no-ignore` / `no_ignore` reach `--ast-grep` / `ast_grep`**: all six ast-grep ignore sources are off and mrw drops nothing.
4. **What is left is named.** A file only ast-grep's own `.gitignore` matcher ignores, which mrw's would serve, is still missed: ast-grep never reports it. And what ast-grep prunes itself is not counted: `-- skipped:` and `skipped` count what mrw's rules dropped from ast-grep's hits, so under `--ast-grep` they can read lower than `--grep`'s on the same tree (the review of #329). The docs say so.

## Alternatives Considered

- **mrw walks, ast-grep reads the list** — rejected by Zy's answer: exact by construction, but every file is enumerated and passed in chunks under Windows' command-line limit, and the 2 s bound would cover several invocations.

## Component / Boundary Impact

Owns `internal/read` (engine): `astgrep.go` and a `hitJudge` in `walk.go`. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `read.AstGrep` | takes `AstGrepOptions{NoIgnore, Skipped}` | T1 | CLI, MCP |
| CLI `--ast-grep` | `--no-ignore` accepted; the `-- skipped:` line | T1 | callers |
| MCP `ast_grep` | `no_ignore` accepted; `skipped` | T1 | callers |
| `scripts/contract.sh` | §221 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** the two finders serve the same files, and say what they skipped the same way.
- **Negative:** a hidden file ast-grep used to skip is now served, as `--grep` serves it.
- **Neutral:** a caller who never relied on the difference sees no change.

## Out of Scope

- A file only ast-grep's `.gitignore` matcher ignores (permanent: boundary: ast-grep keeps its ignore rules for speed, Zy's choice; arm a list-driven run if a disagreement is reported)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a future ast-grep renames a `--no-ignore` source | Low | Medium | ast-grep refuses an unknown value loudly; the CI fake records the flags |
| a hit's ancestry probe is slow on a deep tree | Low | Low | the judge loads each `.gitignore` once, as the walk does |

## Rollback

Revert the task; ast-grep goes back to its own rules. No receipt key changes: `skipped` already exists.

## Follow-ups

- None — the record carries no open follow-up.
