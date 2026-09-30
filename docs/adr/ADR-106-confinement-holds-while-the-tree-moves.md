# ADR-106: Confinement holds while the tree moves

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — approved the plan "resolve the seven findings of the 2026-09-30 Codex design review" (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`), whose ADR-106 is this record's scope, and set the goal "deliver open tasks end to end". The record's text was drafted after that approval and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-066, ADR-071, ADR-075, ADR-086, ADR-105
**Governs:** `internal/apply/tree.go`, `internal/apply/apply.go`, `internal/apply/pathop.go`, `README.md`, `docs/adr/ADR-075-one-writer-per-checkout.md`
**Enforced-by:** `internal/apply/swap106_test.go::TestAWriteThroughASwappedParentStaysInTheRoot`
**Invalidates:** ADR-071 Alternatives, *"`os.Root` for every file access … Rejected for this record"*, for the WRITE path only: this record adopts it there. The read path keeps ADR-071's decision (Out of Scope below). ADR-075's *"the sha guard still catches those after the fact"* is qualified by T4: a change before validation is refused by the guard; a change between validation and commit is what T3's recheck narrows.
**Served-path change:** (1) A write whose target's parent directory is swapped for a link out of the root after validation is refused, naming the path, and writes nothing outside the root; before, an unlink through such a parent removed the outside file, and a rename made its destination directory outside. (2) A write whose existing target was replaced by another file after validation is refused before its commit rename, naming the file, where it used to rename over the newcomer and lose it. Exit codes keep their meanings: both are filesystem failures at commit, reported like any other (ADR-066).
**Served-path change:** (1) A write whose target's parent directory is swapped for a link out of the root after validation is refused, naming the path, and writes nothing outside the root; before, an unlink through such a parent removed the outside file, and a rename made its destination directory outside. (2) A write whose existing target was replaced or rewritten after validation is refused, naming the file: found before the first commit rename, nothing is written; found at a later file's own recheck, the commit stops there with earlier files written (ADR-066). Before, the rename replaced the newcomer, which was lost. Exit codes keep their meanings: both are filesystem failures reported like any other (ADR-066).
## Context

**What was observed** (2026-09-30, the Codex design review of `f1d5996`, finding 3, confirmed in source at `321e066`):

`rooted.Resolve` validates a path and returns it; staging, commit and the path operations then reopen it BY
NAME. `stageFile` resolves through links again at staging (`apply.go:1840`), `unlinkOne` creates its placeholder
in `filepath.Dir(w.full)` and renames `w.full` into it (`pathop.go:203-216`), `renameOne` makes the destination's
directory with `os.MkdirAll` (`pathop.go:228`), and each commit rename is `os.Rename` on the names. A directory
component swapped for a link between validation and that operation redirects it: the unlink's rename then moves
the file the link leads to into an aside beside it and removes it, and the rename's `MkdirAll` builds its
directories wherever the link points. The sha guard cannot see it, because it compares file content, and the
file that moves is not the one the guard read.

The second half is a lost update inside the root: validation reads a file, and another process replaces it with
a new file before the commit rename; the rename then replaces the newcomer, which is lost without a word.

**Audit of the class.** The class is *a filesystem operation that changes the tree after validation*. Enumerated
2026-09-30 by `mrw read --grep 'os\.(Rename|Remove|MkdirAll|CreateTemp|OpenFile|Chmod)\(|commitRenameFn\(|removeFn\('
--exclude '*_test.go' internal/apply/` and reading each site: staging's `MkdirAll`, `CreateTemp` and `Chmod`
(`apply.go:1847-1871`), the rename destination's `MkdirAll` at staging (`:792`), the probe's create and remove
(`:1714-1719`), the commit rename (`:827`), the cleanup removal (`:1774`), and in `pathop.go` the placeholder's
create and removal (`:203-212`), the rename into the aside (`:216`), the destination's `MkdirAll` and the
rename (`:228-234`), the two undo renames (`:171`, `:185`), and `keepAttributes` (`apply.go:1877`,
`attrs_windows.go:31`), which set Windows attributes on the staged file by name after it was closed (found by the
review of the record) — **16** sites, all in scope. The post-validation reads are dispositioned too: the ones
that decide a change — the rename destination's `Lstat` (`:799`, `pathop.go:231`), the "appeared before commit"
`Stat` (`:821`), staging's mode `Stat` (`:1858`) and `noteIfLeft`'s `Lstat` and `ReadDir` — go through the root;
the ones that only report or plan an abort — `missingDirs` and `nameDirs` (`:645`, `:1906`) — stay by name,
since their paths are resolved and any removal they lead to goes through the root.
**Left out:** the reads before validation completes (`os.Stat`, `os.ReadFile`, `apply.go:468`, `:1682`), which
are the read path; `internal/state`, which writes outside the tree.

## Existing Primitives Audit

- **`os.Root`** (Go 1.25+: `OpenFile`, `MkdirAll`, `Rename`, `Remove`, `Lstat`, `Chmod`; `go.mod` is 1.26) —
  every operation resolves each component beneath the root and refuses one that would leave it, a link
  included. It follows a RELATIVE link that stays inside and refuses an absolute one, so it is handed resolved
  paths (below).
- **`rooted.RealAsFarAsItExists`** (`internal/rooted/rooted.go:233`) — the resolution staging already does; it
  follows in-root links and Windows junctions (ADR-071), so the path it returns holds no link.
- **The seams** `stageFileFn`, `commitRenameFn`, `removeFn`, `probeNameFn`, `lstatFn` (`apply.go`) — the
  failure-injection suite; about 30 overrides in four test files.
- **`os.Root` in `internal/state/prune.go:193`** — the precedent for opening a root once and working beneath it.

## Decision

1. **A `tree` holds the checkout open as an `os.Root`** for the length of one `Apply`, opened on the canonical
   root (`rooted.Abs`). Every one of the 15 sites goes through it: its methods take the absolute paths the code
   already carries, convert each to root-relative, refuse one that is not beneath the root, and call the
   `os.Root` operation. A path is handed over RESOLVED — staging's `RealAsFarAsItExists` for a content write,
   and the resolved parent joined with the literal last component for an unlink, a rename and its destination
   — so no link is left in it; a link `os.Root` meets was swapped in after resolution, and one that leaves the
   root is refused by the OS. An in-root relative or absolute symlink and an in-root Windows junction stay
   writable, because resolution removed them before the root saw the path. The temp name is made with
   `OpenFile(O_CREATE|O_EXCL)` and a random suffix (`os.Root` has no `CreateTemp`); its mode and, on Windows, its
   Hidden and System attributes are set through the open handle before it is closed, the attributes read from
   the target through the root (the review of the record).
2. **The seams take the tree as their first parameter** (`commitRenameFn = (*tree).rename`, and so on), so a
   test that injects a failure and passes the rest through still exercises the root. Moved test locks are
   relocked in T2's commit, each with a note naming this change.
3. **Each existing target's identity is checked twice** against what validation stat'ed, its file ID loaded
   then, since on Windows `os.SameFile` otherwise reads it lazily at comparison. First, for every target before
   the first commit rename: a change there stops the plan with nothing written. Then for each target immediately
   before its own rename: a change there stops the commit at that file — earlier files stay written, that file
   fails, later ones skip, ADR-066's partial application, since content renames are never undone. A different
   file, a different size and a different modification time each count, and the refusal names the file. This
   narrows a lost update; it is not confinement. A change after the second check and before the rename, and an
   in-place rewrite that keeps both size and modification time, still get through (the review of the record).
4. **Docs.** ADR-075's sentence on the sha guard is qualified; the README says what confinement holds and when;
   a caller sharing a checkout pins each file with `sha=` from the read header.

## Alternatives Considered

- **Re-check containment by name before each operation** — rejected: it narrows the window without closing it;
  the check and the operation are still two lookups of one name.
- **A global "current tree" the seams read** — rejected: mutable package state keyed per call breaks under any
  concurrent caller.
- **Nil-default seams whose overrides pass through with `os.Rename`** — rejected: about 30 failure-injection
  tests would stop exercising the root while staying green.
- **`os.Root` for the read path too** — rejected for now (Out of Scope): the walk, `--grep` and ast-grep name
  files the root cannot hand them, and a read changes nothing on disk.

## Component / Boundary Impact

Engine package owned: `internal/apply` (T1–T3). `internal/read`, `internal/plan`, `internal/seen`,
`internal/state`, `internal/lines`, `internal/iter`, `internal/rooted`, `internal/check`, `internal/links` stay
byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write`, `mrw_write` | a swapped parent is refused at commit, nothing written outside | T2 | callers |
| `mrw write`, `mrw_write` | a target replaced after validation is refused before its rename | T3 | callers |
| `README.md`, ADR-075 | what confinement holds; `sha=` pinning for shared checkouts | T4 | callers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the swap fixtures | T1 | T2, T3 | no |
| `tree` | T2 | T3 | no |

## Implementation

See `tasks/README.md`: T1 (the swap escapes today), T2 (the write path through `os.Root`), T3 (the identity
recheck), T4 (docs).

## Consequences

- **Positive:** no write reaches outside the root through a link swapped in after validation; a file replaced
  under a plan is not silently overwritten.
- **Negative:** one more open handle per write; a target touched between validation and commit — an editor
  saving it — now refuses the write where it used to win.
- **Neutral:** receipts and exit codes are unchanged; paths still read the same.

## Out of Scope

- The read path through `os.Root` (permanent: boundary: the walk, `--grep` and ast-grep name files the root cannot hand them, and a read changes nothing on disk; ADR-071 keeps governing it)
- A contract row for the swap (permanent: boundary: no fixture swaps a directory between two steps of one call from outside the process; T1 drives it through `stageFileFn`)
- A temp file another process moved away before cleanup (permanent: boundary: `left_behind` names what mrw made where it made it; a temp renamed away with its directory is at a path mrw never made, and the README's staging row says so; T2's `TestATempMovedByAnotherProcessIsNotClaimed` pins it)
- A change after the second identity check and before the rename, or an in-place rewrite keeping size and modification time (permanent: boundary: no filesystem mrw runs on offers a compare-and-rename; the window is the time between two system calls, and a same-size same-time rewrite is indistinguishable by metadata)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| `os.Root` refuses a path Windows reaches through a junction | Med | High | resolved paths hold no junction; T2's in-root link test and the Windows CI shards run before T3 |
| an open root handle blocks a rename beneath it on Windows | Low | High | the handle omits delete sharing only for the root directory itself, which mrw never renames; the Windows shards run every rename test beneath it |
| an editor's save between validation and commit now refuses the write | Low | Low | the refusal names the file; read and send again |

## Rollback

Revert T1–T4. No state or receipt changes.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `internal/apply/swap106_test.go::TestAWriteThroughASwappedParentStaysInTheRoot` in T1's commit.
