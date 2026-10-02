# ADR-116: A walk honours .gitignore and skips binaries

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — "Default on, flag to disable (Recommended)" to "should --grep honour .gitignore and skip binaries?" (2026-10-01), then "Native matcher" to "how should the walk know what is ignored?" (2026-10-02), and the plan he approved the same day. The record's text was drafted after those answers and not shown to Zy before execution: this line is the session's reading of them, stated so it can be checked
**Date:** 2026-10-02
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-007, ADR-073, ADR-077, ADR-096, ADR-111
**Invalidates:** ADR-007 — its rule 6 ("`.gitignore` is not read"), its Out of Scope entries making ignore files and binary detection permanent, and the "mrw has no git dependency" premise only as far as reading `.gitignore` files: mrw still runs no git and needs none
**Governs:** `internal/read/ignore.go`, `internal/read/walk.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/guide/guide.go`, `docs/receipts.txt`, `AGENTS.md`, `README.md`, `scripts/contract.sh`
**Enforced-by:** `internal/read/ignore116_test.go::TestAWalkSkipsWhatGitIgnores`
**Served-path change:** `mrw read --grep` and `mrw_read`'s `grep` no longer serve files a checkout's `.gitignore` ignores, nor files that are binary (a UTF-16/32 byte-order mark or a NUL in the first 8 KiB, ADR-073's test), and say how many they skipped. `--no-ignore` (CLI) and `no_ignore: true` (MCP) walk as before. A path the caller names is always served.

## Context

A walk served everything that is a regular file inside the root (ADR-007 rule 6), so `--grep` over a checkout read `node_modules`, build output and binaries — `cmd/opencode/mrw-plugin/node_modules` in this repository, which `.gitignore` names, was served by every `--grep` over `cmd/`. The gap list of 2026-10-01 named it; Zy chose default-on with a flag, and a native matcher rather than a git subprocess, so mrw still runs no git.

ADR-007 rejected both on stated premises: honouring `.gitignore` "needs either a git subprocess or a reimplementation" (Zy chose the reimplementation), and binary detection "silently skips" a candidate (this record does not: every skip is counted and said, with the flag that walks it).

**Audit of the class** — *a finder that discovers files by walking*: `mrw read --grep 'read\.Walk\(|AstGrep\(' --exclude '*_test.go' cmd internal` — the CLI's `--grep` (`cmd/mrw/main.go:818`), MCP's `grep` (`internal/mcp/tools.go:1666`), and `--ast-grep`/`ast_grep`, which runs ast-grep's own walker. The first two get this record. ast-grep is left to its own ignore rules (Out of Scope).

## Existing Primitives Audit

- **`read.Walk`, `walkDir`, `offer`** (`internal/read/walk.go`) — the walk; `.git` is already pruned.
- **`lines.Unsplittable`** (ADR-073) — the binary test, on bytes `offer` already reads.
- **`pathExcluded`** — `--exclude`, unchanged and applied as before.

## Decision

1. **A native gitignore matcher** in `internal/read/ignore.go`, following gitignore(5): blank lines and `#` skipped, `\#` and `\!` escapes, trailing unescaped spaces trimmed, `!` negation (a path under an ignored directory cannot be re-included), a trailing `/` for directories only, a pattern with a `/` at its start or middle anchored to its `.gitignore`'s directory and one without matching the name at any depth, `*`, `?`, `[...]`, and `**` as `**/`, `/**` and `/**/`; the last matching rule wins.
2. **Which rules apply.** Only inside a checkout — a `.git` directory or file at or above the walk root — as git applies them: every `.gitignore` from the checkout's top down to each directory the walk enters, and `info/exclude` in the checkout's git directory (where `.git` is a file — a worktree or a submodule — the gitdir it names, through its `commondir`). `core.excludesFile` is not read. An ignored directory is pruned, not entered. Matching folds case where the checkout's filesystem does, the probe `git init` makes to set `core.ignorecase`; an explicit `core.ignorecase` is not read, since mrw parses no git config. An ignore file is read as a served file is: a FIFO or device gives no rules rather than a wait (ADR-109), and one over the read bound gives none (ADR-104).
3. **Binaries.** A discovered file that `lines.Unsplittable` refuses is skipped, inside a checkout or not.
4. **Nothing is skipped silently.** The CLI read ends with a `-- skipped:` line counting files and directories the ignore rules skipped and binary files, naming `--no-ignore`; MCP's read receipt gains `skipped` with the same counts. Nothing skipped, no line and no key.
5. **`--no-ignore` / `no_ignore: true`** walk as ADR-007 did: every regular file. A path the caller names is served whatever the rules say, and an ignored directory the caller names is walked, as `--exclude` prunes only below where a walk starts; the rules still apply inside it.

## Alternatives Considered

- **A `git ls-files` subprocess** — rejected by Zy's answer: a git dependency, and behaviour that changes with git's presence.
- **Off by default, opt in** — rejected by Zy's answer of 2026-10-01.
- **Skip binaries without counting them** — rejected: ADR-007's objection to a silent omission stands.

## Component / Boundary Impact

Owns `internal/read` (an engine package). `cmd/mrw`, `internal/mcp`, `internal/guide` are not engine. Every other engine package stays byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `read.WalkOptions` | `NoIgnore`, `Skipped *WalkSkipped` | T1 | CLI, MCP |
| CLI `read` | `--no-ignore`; the `-- skipped:` line | T1 | callers |
| `mrw_read` | `no_ignore`; receipt `skipped` | T1 | MCP callers |
| `docs/receipts.txt` | `mcp_read skipped…` | T1 | ADR-111's tests |
| AGENTS.md, README, guide, instructions | say so | T1 | readers |
| `scripts/contract.sh` | a row | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a `--grep` over a checkout reads what its owner tracks, and binaries never reach a caller's context.
- **Negative:** a caller who meant an ignored file must name it or pass `--no-ignore`; the counts say so.
- **Neutral:** `--exclude` is unchanged.

## Out of Scope

- `--ast-grep` / `ast_grep` (deferred: `docs/adr/BACKLOG.md` "From ADR-116")
- `core.excludesFile`, a user's global ignore file (permanent: boundary: per-user configuration; a walk answers the same on every machine)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the matcher disagrees with git | Medium | Medium | `TestTheIgnoreMatcherAgreesWithGit` compares it with `git check-ignore` over a fixture tree where git is installed |
| a caller misses an ignored file they meant | Low | Low | the skipped line names `--no-ignore`; a named path is always served |
| an explicit `core.ignorecase` disagrees with the filesystem | Low | Low | the walk serves more or fewer by case alone; the count says how many were skipped |

## Rollback

Revert T1. The `skipped` receipt key disappears with it, which ADR-111 calls a removal: revert before a release ships it.

## Follow-ups

- None — the record carries no open follow-up.
