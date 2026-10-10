# ADR-143: Writes into `.git` are refused

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — "Refuse writes into .git/ (Recommended)", the answer to the open question from the Windows retest of v1.60.0 (a plan edited `.git/config` and nothing objected)
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-071, ADR-077, ADR-081, docs/adr/BACKLOG.md
**Invalidates:** None — no accepted record let a write into `.git`; it was an unstated gap
**Governs:** `internal/rooted/gitdir.go`, `internal/apply/apply.go`, `internal/apply/tree.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/rooted/gitdir143_test.go::TestAPathInsideADotGitIsRefused`
**Served-path change:** a plan hunk, a `--create`, an `apply_patch` or `search_replace` edit or an `mrw_write` whose path has a `.git` component, or whose real location does, is refused as a failed hunk: exit 1, nothing written, the siblings skip. Reads of `.git` are unchanged.

## Context

mrw writes anything under `--root`, and `.git` is under the root of every checkout. A plan that edits `.git/config`, drops a file in `.git/hooks/` or rewrites `.git/HEAD` changes what git does next without git being asked: a hook runs on the next commit, a `url` rewrites where a push goes. The Windows retest of v1.60.0 (2026-10-10) found a plan editing `.git/config` applied at exit 0, and the owner chose to refuse it (2026-10-10).

**Audit of the class** — *a place where mrw chooses the file a write lands on*: `mrw read --grep 'resolve\(' --exclude '*_test.go' internal/apply` names five call sites in `internal/apply` (a hunk's path, two rename destinations in `Apply`, the destination in `planPathOp`, and `existsUnder`, which only asks whether a name is on disk). Every other call to `rooted.Resolve` serves a read (`files_from`, a line count, the ingest formats reading the original they patch). `mrw read --grep 'os\.(WriteFile|OpenFile|Create|Remove|Rename|MkdirAll)|WriteSynced' --exclude '*_test.go' internal cmd` finds the rest, and all of them write mrw's own state or temporary files. The tree is written only through `apply`, so one function decides.

## Existing Primitives Audit

- **`apply.resolve`** — already the one door every hunk path and rename destination goes through, and already refuses what must not be written (a path outside the root, a name Windows will not keep). The new refusal is one more line of it.
- **`rooted.inState` (ADR-077)** — refuses a read or a write by where the path really lands, links followed. This record judges by the same two views, spelled and real, and does not reuse the state test: a different place, a different message, and reads stay allowed.
- **`rooted.RealAsFarAsItExists`** — the real location of a path whose leaf may not exist yet; used as it is.
- **`RefusedByName`** — not extended: it marks refusals that read paths also make, and this one is made on the write side only.

## Decision

1. **A write whose path lands in a `.git` is refused before anything is staged.** `rooted.GitDir(root, path)` returns an error when a component of the path as spelled (cleaned), of the root as spelled, of the path's real location (links followed as far as it exists), or of the real directory holding its leaf entry (a link inside `.git` is judged where it sits, since unlink and rename act on the entry and not on what it leads to) is `.git`. `apply.resolve` calls it after `rooted.Resolve`, so every op (create, replace, insert, delete, unlink, rename source, rename destination) on every surface (CLI plan, `--create`, `--format`, `mrw_write`) fails the hunk that names it: exit 1, the siblings skip, nothing written (ADR-001).
2. **The match is the name, folded.** `.git` is matched ignoring case on every platform, since git on a case-insensitive disk reads `.GIT` as `.git`; the code points HFS+ ignores (U+200C to U+200F, U+202A to U+202E, U+206A to U+206F, U+FEFF) are dropped first, as git's own protection does; on a Windows build `git~1` is matched too, the 8.3 name NTFS gives `.git` and the one spelling git's own protection matches. `.github`, `.gitignore`, `x.git`, `git` and `git~12` are not `.git`.
3. **Reads stay allowed.** `mrw read .git/config` and a `--grep` over `.git` are unchanged (ADR-116 skips `.git` in a walk; a path you name is served).
4. **There is no override.** `--force` lifts the read-before-modify guards and does not lift this; no flag does. The message says what to use instead.
5. **The message names the cause once:** `<path> is inside a .git directory; mrw does not write there, since a hook, a config or a ref changed behind git's back changes what git does next — use git for it (mrw read still reads it)`. A path that reaches `.git` through a link says the same with the real location.
6. **A path reopened by name is judged again, by where it lands now.** Staging, rename and removal reopen their paths by name after validation, so a directory swapped for a link to `.git` in between would carry the write there (reproduced with ADR-106's seams: a hook was made in `.git` and `.git/config` removed). `tree.rel`, through which every one of them passes, refuses a path with a `.git` component (T2), or whose parent now resolves into one, the leaf as written (T3): a rename or an unlink acts on the entry, not on what a link at its end leads to, and a RELATIVE link swapped in after staging leaves the spelling clean. A refusal while staging is a failed hunk, exit 1 (T3); one while committing is the commit's own failure (ADR-066). What stays is the instant between that check and the syscall that follows it.

## Alternatives Considered

- **Allow it, warn on the receipt** — rejected: a warning after a hook is written is the failure; the owner chose refuse.
- **Refuse only `.git/hooks` and `.git/config`** — rejected: `HEAD`, `index`, `objects` and `refs` corrupt a repository as surely, and a list of names is the list that misses one.
- **Judge by identity (the same file as one under `.git`)** — rejected for now: needs a walk of `.git` on every write; the real-location check already catches links and junctions, and a hard link is no hole: an edit is a temp file renamed over the name, so the other name keeps its content (pinned by `TestAHardLinkToAFileUnderDotGitIsNotAWayIn`).
- **A `--allow-git` flag** — rejected: the first caller to need it would pass it by habit, which is the failure again; `git` itself edits `.git`.

## Component / Boundary Impact

`internal/rooted` (one function, one file), `internal/apply` (one call in `resolve`). No other engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `rooted.GitDir` | new function | T1 | `apply.resolve` |
| any write whose path lands in `.git` | refused: exit 1, nothing written | T1 | CLI and MCP callers |
| `scripts/contract.sh` | §246 | T1 | CI Linux |
| `AGENTS.md` | one sentence in section 4 | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `inGit` (T1), exported as `rooted.HasGitComponent` | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, T2, T3.

## Consequences

- **Positive:** a plan cannot plant a hook or rewrite a ref; the refusal names git as the tool for it.
- **Negative:** a caller who really means to edit `.git` (a hand-fixed config, a test fixture that builds a repository by writing its files) must use git or the shell; that is the decision.
- **Neutral:** a root that sits inside a `.git` directory refuses every write, which is the same decision applied to the root.

## Out of Scope

- A hard link in the tree to a file under `.git` (permanent: fact: an edit is a temp file renamed over the name, so the other name keeps its content; citation: file `internal/apply/gitdir143_test.go:68`)
- A repository whose git directory is elsewhere and not named `.git` (a bare repository, `--separate-git-dir`) (permanent: boundary: mrw cannot know which directory is a repository by its name)
- What a check or a `--then-sh` step does to `.git` (permanent: boundary: they run the project's own commands, not mrw's writes)
- A directory swapped for a link to `.git` in the instant between `tree.rel`'s check and the syscall that follows it (permanent: boundary: the check runs when each staging, rename and removal opens its path, the finest grain mrw has; a process that can swap a directory in that instant can write `.git` itself)
- Windows 8.3 names of `.git` other than `git~1`, such as a hashed short name (permanent: fact: Windows path resolution turns an 8.3 alias of an existing directory into its long name, so whatever short name a volume made for `.git` reaches `GitDir` as `.git`; measured on the CI Windows runner for its alias `GIT~1`; citation: file `internal/rooted/gitdir143_windows_test.go:69`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a legitimate directory named `.git` that is not a repository is unwritable | Low | the caller uses the shell | the message names the cause; the owner chose it |
| an extra real-path lookup on every written path | Low | one `EvalSymlinks` per hunk path | the write already stats and stages each path |

## Rollback

Revert T1: writes into `.git` apply as before. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the deferred items above.
