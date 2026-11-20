# ADR-213: A no-replace rename is one system call where the OS has one

**Status:** Proposed
**Drafted for:** Zy, 2026-11-20. Zy accepted drafting it (*"accept all"*, relayed by the session that commissioned it); acceptance of the new module requirement follows the Codex xhigh challenge, so this record is Proposed and nothing here merges before Zy reads it.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-034, ADR-038, ADR-057, ADR-066, ADR-071, ADR-088, ADR-106, ADR-129, #726, #728, #730, #731, #744
**Supersedes:** in part, on Zy's acceptance: the standing invariant that `go.mod` declares exactly one requirement (ADR-034's tasks, ADR-038's audit "`golang.org/x/sys` was audited and **not** taken", ADR-010's fences and the ~100 records whose fences repeat the clause), and `CONTRIBUTING.md`'s "One module dependency today"; #744's statement, in the comment on `renameNew`, that "no portable no-replace rename … closes" the last window, for Linux and macOS.
**Governs:** `internal/apply/tree.go`, `go.mod`, `go.sum`, `CONTRIBUTING.md`, `scripts/fence-prose.py`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/noreplace213_test.go::TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne`
**Served-path change:** none a caller can trigger on purpose: a move that must not replace (an undo, a create, a path op's rename, #744) is, on Linux and macOS, one system call that the OS refuses when the destination name is taken, instead of a hard link and a removal. The window #744 left — a destination swapped in after mrw's last look and before it removes the source, which can cost the source its last name — closes there. Receipts, refusals and exit codes are unchanged. Windows and the other systems keep #744's path.

## Context

**What #744 did.** `tree.renameNew` (`internal/apply/tree.go:190-254`, merged in #744 at `a4040443`) moves a file without replacing one another process put at the destination: it hard-links the source under the new name (the OS refuses a taken name, atomically), looks at the destination again, removes the source, and takes the new name back on any failure (`finishLink`, `:256-287`; `stillLinked`, `:289-314`). Where a hard link is impossible (`noHardLinks`: exFAT or FAT, `protected_hardlinks`, another device) or the source is a directory, it asks whether the name is free and renames (`checkThenRename`, `:316-331`). Every no-replace move takes this path: `tree.noReplace()` callers at `internal/apply/apply.go:1353` and `internal/apply/pathop.go:409`, `:430`, `:497`.

**The window that remains**, in the comment on `renameNew`: "A destination swapped in after THAT look, in the instant before the removal, still loses the source's last name: no portable no-replace rename (os.Root has none) closes it." `checkThenRename` has the same window between its look and its rename. The test seam `beforeRemoveSource` (`tree.go:173-176`) is where a test swaps the destination in.

**What the OS offers.** Linux `renameat2(2)` with `RENAME_NOREPLACE` (kernel 3.15+, and the filesystem must support it, else `EINVAL`) and macOS `renameatx_np(2)` with `RENAME_EXCL` (macOS 10.12+, `ENOTSUP` where the filesystem cannot) rename and refuse a taken destination in one call. The standard library exposes neither: `syscall` has no flagged `renameat2` on most architectures and no `renameatx_np`; `internal/syscall/unix` has a `renameat2` but is not importable; `os.Root` has `Rename` (which replaces) and `Link`, nothing else. `golang.org/x/sys/unix` has both, `Renameat2` and `RenameatxNp` (checked in the module cache at `v0.49.0`, `unix/zsyscall_linux.go:1468`, `unix/syscall_darwin.go:414`), with `RENAME_NOREPLACE` and `RENAME_EXCL`.

**What a second requirement costs here.** `CONTRIBUTING.md` (Prerequisites) says a new module needs a record saying why the standard library and the existing dependency cannot serve, and `govulncheck` stays clean. The corpus has treated one requirement as a standing invariant: ADR-038 audited `golang.org/x/sys` and did not take it, citing ADR-034, and 205 task fences across 101 records assert `[ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]`. Those are done tasks; their receipts were true when written. `scripts/fence-prose.py` reruns only plain `grep` clauses over tracked files, so of all those it checks two, ADR-010 T1's and T2's `grep -q '^require github.com/urfave/cli/v3 ' go.mod`, which a `require ( … )` block no longer matches.

**Audit of the class**: *the moves mrw makes that must not replace.* A content rename onto a target (replaces on purpose; not in the class); a create's rename of its temp (#744: must not replace a name that appeared); a path op's rename (#744); the undo of a rename and the restore of an aside (#744, ADR-066). All four of the class go through `renameNew`.

## Existing Primitives Audit

- `tree` holds the checkout's `os.Root`; `Root.Open` of a directory gives an `*os.File` whose descriptor is a directory inside the root, the way every tree operation stays confined (ADR-071, ADR-106).
- `respelling` (`internal/apply/pathop.go:132`) already decides a case-only rename on a folding filesystem (ADR-129), which a no-replace call refuses as a taken name.
- `extraNameError`, `noHardLinks` and the seams `linkFn`, `fallbackLstatFn`, `removeNameFn` stay for the fallback.

## Decision

1. **On Linux and macOS, `renameNew` first tries one no-replace rename.** It opens the source's and the destination's parent directories through the tree's `os.Root` (`Root.Open`), and calls `unix.Renameat2(srcdir, srcleaf, dstdir, dstleaf, unix.RENAME_NOREPLACE)` on Linux or `unix.RenameatxNp(…, unix.RENAME_EXCL)` on macOS, with each leaf a single path component, then closes the descriptors. The `*at` calls do not follow a link at the leaf, and the directories are the root's, so the move stays confined as every tree operation is.
2. **What its answers mean.** Success: the move is done and whole. `EEXIST` (and `ENOTEMPTY` for a directory onto one): the destination is taken; the move fails with an error satisfying `fs.ErrExist`, as #744's does, except for a respelling on a folding filesystem (ADR-129), which is renamed as today. `EINVAL`, `ENOSYS`, `ENOTSUP` or `EOPNOTSUPP` (a kernel or filesystem without the flag): today's path, unchanged — the hard link and removal, or `checkThenRename` — with its window. Any other error is the move's.
3. **Directories** take the same call, so a directory's no-replace move is atomic too where it is available; where it is not, `checkThenRename` as today.
4. **Windows and every other system are unchanged** (#744's path). `os.Root.Rename` on Windows replaces, and its flags are Go's; the BSDs other than macOS have no such call in `x/sys`.
5. **The requirement.** `golang.org/x/sys` is added at the newest release at least two weeks old when T1 lands (`v0.49.0` was checked), used from one build-tagged file per OS in `internal/apply`; `govulncheck` stays clean (`scripts/static.sh`). `CONTRIBUTING.md`'s Prerequisites row names two dependencies and this record.
6. **The corpus's one-requirement clause.** The 205 done fences are not rewritten: their receipts stand as of when they ran, and a re-run of one fails on that clause by this record, which `adr-verify` reports. `scripts/fence-prose.py`'s `EXEMPT` gains the two plain-grep clauses of ADR-010 T1 and T2, each naming ADR-213, as ADR-108 and ADR-113 freed clauses before.

## Alternatives Considered

- **Raw `syscall.Syscall6` for `renameat2`.** Rejected: the syscall number differs per architecture and is missing from `syscall` on most; on macOS raw system calls are unsupported, and `renameatx_np` is a libSystem call reached only through `x/sys`'s trampolines.
- **Copy `x/sys`'s two wrappers into the tree.** Rejected: the macOS one needs `x/sys`'s assembly trampolines and `//go:linkname` to libSystem, which is the module.
- **Keep #744's window.** Rejected by the critique that #744 answered: the window is narrow but loses a file's last name when hit.
- **Linux only.** Rejected: macOS is where most of this repository's sessions run, and the call exists there.

## Component / Boundary Impact

Engine package owned here: `internal/apply` (`tree.renameNew`, new build-tagged files for the call). `go.mod` and `go.sum` gain a requirement. `CONTRIBUTING.md` and `scripts/fence-prose.py` change words and exemptions. No other engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a no-replace move on Linux and macOS | one system call; the remaining window of #744 closed | T1 | every write with a create, a path op or an undo |
| `go.mod`, `go.sum` | `golang.org/x/sys` | T1 | builds, `govulncheck` |
| `CONTRIBUTING.md` Prerequisites | two dependencies, and this record | T1 | contributors |
| `scripts/fence-prose.py` `EXEMPT` | ADR-010 T1 and T2's `^require github.com/urfave/cli/v3 ` clauses, freed by ADR-213 | T1 | the static gate |
| `scripts/contract.sh` §375 | on Linux under `strace` where present, a create and a rename show `renameat2(…, RENAME_NOREPLACE)` and no `link`/`linkat`; paired with a destination made beforehand, which refuses the move with the source unchanged | T1 | the contract gate |

§375 is free on `origin/main` and on every remote branch at `a4040443` (checked under bash: sections reach §371, and §374 is reserved in prose by ADR-208 T2).

No receipt key is added or renamed (ADR-111).

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** on Linux and macOS a move that must not replace is atomic; the source keeps a name whatever another process does.
- **Negative:** a second module requirement, `golang.org/x/sys`, with its updates and `govulncheck` findings; the corpus's one-requirement clause no longer holds, and re-running an old fence fails on it.
- **Neutral:** the fallback, Windows and the other systems behave as #744 left them; one more `open` and `close` of two directories per move.

## Out of Scope

- Windows' no-replace rename (`MoveFileEx` without `MOVEFILE_REPLACE_EXISTING` through a root handle) (permanent: boundary: `os.Root.Rename`'s Windows path is Go's handle-relative call and its flags are not exposed; #744's link-then-remove stays there)
- The other BSDs (permanent: boundary: `x/sys/unix` exposes no flagged no-replace rename for them, so they keep #744's path)
- Rewriting the 205 done fences (permanent: boundary: a done task's receipt records what held when it ran; Decision 6)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A filesystem answers the flag with an error not in Decision 2's list | Low | Med | the move fails and says so; T1 adds the error to the fallback list only with a reproduction |
| `x/sys` raises the minimum OS version | Low | Med | T1 checks its `go.mod` and build constraints against the release matrix |
| A `govulncheck` finding in `x/sys` blocks a release | Low | Low | updated like any requirement; the static gate names it |

## Rollback

Revert T1: `go.mod` loses the requirement, and #744's path is the only one again.

## History

- 2026-11-20: drafted as Proposed; the Codex xhigh challenge follows.

## Follow-ups

- [ ] None beyond T1.
