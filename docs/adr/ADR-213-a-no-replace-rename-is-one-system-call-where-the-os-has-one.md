# ADR-213: A no-replace rename is one system call where the OS has one

**Status:** Proposed
**Drafted for:** Zy, 2026-11-20. Zy accepted drafting it (*"accept all"*, relayed by the session that commissioned it); acceptance of the new module requirement follows the Codex xhigh challenge, so this record is Proposed and nothing here merges before Zy reads it.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-034, ADR-038, ADR-057, ADR-066, ADR-071, ADR-088, ADR-106, ADR-129, #726, #728, #730, #731, #744
**Supersedes:** in part, on Zy's acceptance: the standing invariant that `go.mod` declares exactly one requirement (ADR-034's tasks, ADR-038's audit "`golang.org/x/sys` was audited and **not** taken", ADR-010's fences and the other done fences that repeat the clause), and `CONTRIBUTING.md`'s "One module dependency today"; #744's statement, in the comment on `renameNew`, that "no portable no-replace rename … closes" the last window, for Linux and macOS on filesystems that support the flag; #744's contract §370 and its Go twin, on Darwin, where a refused atomic rename makes no second name to report.
**Governs:** `internal/apply/tree.go`, `go.mod`, `go.sum`, `CONTRIBUTING.md`, `scripts/fence-prose.py`, `scripts/contract.sh`, `.github/workflows/ci.yml`
**Enforced-by:** `internal/apply/noreplace213_test.go::TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne`
**Served-path change:** on Linux and macOS, where the filesystem supports the flag, a move that must not replace (a create's temp, a path op's rename, an undo, #744) is one system call that the OS refuses when the destination name is taken, instead of a hard link and a removal; the window #744 left closes for those moves. Refusals and exit codes are unchanged. One receipt changes on those systems: when the rename itself is refused (an ACL that denies the delete, #744's §370), no second name was made, so `left_behind` no longer names one. Windows, the other systems, and filesystems without the flag keep #744's path.

## Context

**What #744 did.** `tree.renameNew` (`internal/apply/tree.go:190-254`, merged in #744 at `a4040443`) moves a file without replacing one another process put at the destination: it hard-links the source under the new name (the OS refuses a taken name, atomically), looks at the destination again, removes the source, and takes the new name back on any failure (`finishLink`, `:256-287`; `stillLinked`, `:289-314`). Where a hard link is impossible (`noHardLinks`) or the source is a directory, it asks whether the name is free and renames (`checkThenRename`, `:316-331`). Every no-replace move takes this path: `tree.noReplace()` callers at `internal/apply/apply.go:1353` and `internal/apply/pathop.go:409`, `:430`, `:497`.

**The windows that remain.** The comment on `renameNew`: "A destination swapped in after THAT look, in the instant before the removal, still loses the source's last name". `checkThenRename` has its own window between its look and its rename. The seam `beforeRemoveSource` (`tree.go:173-176`) runs before the final look, so #744's own tests survive an injection there; the window is after that look.

**What the OS offers.** Linux `renameat2(2)` with `RENAME_NOREPLACE` (kernel 3.15+, filesystem support required, else `EINVAL`) and macOS `renameatx_np(2)` with `RENAME_EXCL` (`ENOTSUP` where the filesystem cannot) rename and refuse a taken destination in one call, for files and directories, between different parents on one mount; across mounts they fail with `EXDEV` as any rename does. On a case-folding macOS volume `RENAME_EXCL` permits a genuine case-only rename (Apple's `tests/rename_excl.c`); on Linux two spellings that are one entry can answer `EEXIST`.

**Why `golang.org/x/sys`** (the challenge's findings, checked against Go 1.27.4's source). `os.Root.Rename` takes no flags. `syscall`'s Linux `Renameat` passes zero flags even where it uses the `renameat2` number, and `internal/syscall/unix`'s `renameat2` is not importable. A Linux-only implementation could call `syscall.Syscall6` with a per-architecture syscall number. On macOS the supported route is libSystem: cgo, which the release policy forbids (`CONTRIBUTING.md`, Prerequisites: "no cgo"), or dynamic imports and assembly trampolines kept in this tree. `golang.org/x/sys/unix` has both, `Renameat2` and `RenameatxNp` (module cache `v0.49.0`: `unix/zsyscall_linux.go:1468`, `unix/syscall_darwin.go:414`), with `RENAME_NOREPLACE` and `RENAME_EXCL`; its `go.mod` declares `go 1.26.0` and no requirements of its own. Its licence is BSD-3-Clause, whose notice must travel with a binary.

**What a second requirement costs here.** `CONTRIBUTING.md` (Prerequisites) says a new module needs a record saying why the standard library and the existing dependency cannot serve, and `govulncheck` stays clean. The corpus has treated one requirement as a standing invariant: ADR-038 audited `golang.org/x/sys` and did not take it, citing ADR-034. Measured on the done task fences (the drafter's count, confirmed by the challenge with an in-memory `go mod edit -print`):

| Clause in a done fence | How many | With `x/sys` added |
|------------------------|----------|--------------------|
| `[ "$(grep -cE '^require\|^[[:space:]]' go.mod)" = "1" ]` | 205 fences, 101 records | the count is 3 (the `require (` line and two module lines): a fresh run fails on it |
| `grep -q '^require github.com/urfave/cli/v3 ' go.mod` (a plain grep `scripts/fence-prose.py` reruns) | 2, ADR-010 T1 and T2 | fails: the requirement moves into a block |
| older counters (`grep -c '^require ' go.mod` and similar, counted by the challenge) | 17 | still 1: passes, though the invariant it meant no longer holds |
| `go.mod`/`go.sum` baseline comparisons (counted by the challenge) | 25 | pass on `main` after the merge; fail on a branch cut before it and compared after |

**Audit of the class**: *the moves mrw makes that must not replace.* A create's rename of its temp (#744); a path op's rename (#744); the undo of a rename and the restore of an aside (#744, ADR-066). A content rename onto its target replaces on purpose and is not in the class. All three go through `renameNew`.

## Existing Primitives Audit

- `tree` holds the checkout's `os.Root`; `internal/rooted/fdlimit_other.go:36` already opens a directory through a root with `O_RDONLY|O_DIRECTORY|O_NONBLOCK`, which a FIFO fails at once, where a plain `Root.Open` would block.
- `respelling` (`internal/apply/pathop.go:132`) decides a case-only rename on a folding filesystem (ADR-129) by the exact spelling and identity.
- `extraNameError`, `noHardLinks` and the seams `linkFn`, `fallbackLstatFn`, `removeNameFn` stay for the fallback.

## Decision

1. **On Linux and macOS, `renameNew` first tries one no-replace rename.** It opens the source's and the destination's parent directories through the tree's `os.Root` with `O_RDONLY|O_DIRECTORY|O_NONBLOCK` (as `rooted` does), calls `unix.Renameat2(…, unix.RENAME_NOREPLACE)` on Linux or `unix.RenameatxNp(…, unix.RENAME_EXCL)` on macOS with each leaf a single path component, and closes the descriptors. The `*at` calls do not follow a link at the leaf; a link in a parent is resolved by the root at the open. A parent directory moved outside the checkout after the open is followed by its descriptor, as Go's own `Root.Rename` does on Unix; that is the confinement model's existing limit, stated.
2. **What the call's answers mean.** The fallback list applies only to an error from the rename call itself; an error opening a parent is the move's.
   - Success: the move is done, atomic.
   - `EEXIST`, or `ENOTEMPTY` for a directory onto one: the name is taken; the move fails with an error satisfying `fs.ErrExist`, as #744's does.
   - `EINVAL`, `ENOSYS`, `ENOTSUP`, `EOPNOTSUPP`: #744's path, unchanged. Falling back on `EINVAL` is a compatibility choice: it also answers an invalid move (a directory into its own descendant), which #744's path then refuses on its own terms.
   - `EXDEV`: the move's, as for any rename across mounts.
3. **Respelling stays ADR-129's, outside the atomic guarantee.** A case-only rename is first judged by `respelling`'s exact-spelling and identity checks, as today. On macOS `RENAME_EXCL` performs a genuine case-only rename itself; on Linux an `EEXIST` for two spellings of one entry falls to the plain-rename branch ADR-129 already has. That branch is a compatibility path with #744's window, not part of this record's guarantee.
4. **What is guaranteed, and what is not.** Where the call succeeds or refuses, the source entry present when it runs keeps a name: the move happens whole or not at all. The guarantee does not reach the fallback paths (they keep #744's windows), a move across mounts, or Windows and the other systems (#744's path, unchanged).
5. **The requirement.** `golang.org/x/sys` at the newest release at least two weeks old when T1 lands (`v0.49.0` checked), used from one build-tagged file per OS in `internal/apply`. T1 checks the parsed module requirements (`go mod edit -json`): exactly `github.com/urfave/cli/v3` and `golang.org/x/sys`. The release archives carry `x/sys`'s BSD-3-Clause notice beside mrw's `LICENSE`. `govulncheck` runs for `GOOS=darwin` as well as the host configuration. `CONTRIBUTING.md`'s Prerequisites row names two dependencies and this record.
6. **The done fences.** Their receipts stand as of when they ran. A fresh run of one fails on its one-requirement clause, by this record (the table in Context says which clause does what); `adr-verify` prints the failing clause, and the reader takes it as this record's change, not a regression. `scripts/fence-prose.py`'s `EXEMPT` gains the two plain-grep clauses of ADR-010 T1 and T2, naming ADR-213.
7. **#744's receipt on Darwin.** §370 (an ACL denies the delete, so #744's path leaves two names and `left_behind` names one) becomes: the atomic rename is refused, no second name exists, `left_behind` is empty, and the move fails as before (exit 2). Its Go twin changes with it; both keep the old expectation by forcing the fallback path.

## Alternatives Considered

- **Raw `syscall.Syscall6` on Linux, nothing on macOS.** Rejected: per-architecture numbers kept by hand for half the guarantee; macOS is where most of this repository's sessions run.
- **Copy `x/sys`'s wrappers and trampolines into the tree.** Rejected: the macOS one needs `x/sys`'s dynamic imports and assembly, which is the module kept by hand.
- **Keep #744's window.** Rejected by the critique #744 answered: the window is narrow, and loses a file's last name when hit.

## Component / Boundary Impact

Engine package owned here: `internal/apply` (`tree.renameNew`, new build-tagged files for the call). `go.mod` and `go.sum` gain a requirement. `CONTRIBUTING.md`, `scripts/fence-prose.py`, the release job (`.github/workflows/ci.yml`) and `scripts/static.sh`'s `govulncheck` step change. No other engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a no-replace move on Linux and macOS | one system call where the filesystem supports it | T1 | every write with a create, a path op or an undo |
| `left_behind` after a refused rename on Darwin (§370) | no second name to report | T1 | callers reading the receipt |
| `go.mod`, `go.sum` | `golang.org/x/sys` | T1 | builds |
| release archives | `x/sys`'s licence notice | T1 | users of a release |
| `scripts/static.sh` | `govulncheck` also for `GOOS=darwin` | T1 | the static gate |
| `CONTRIBUTING.md` Prerequisites | two dependencies, and this record | T1 | contributors |
| `scripts/fence-prose.py` `EXEMPT` | ADR-010 T1 and T2's `^require github.com/urfave/cli/v3 ` clauses, freed by ADR-213 | T1 | the static gate |
| `scripts/contract.sh` §375 | on the designated Linux runner, required: a create and a rename traced with `strace`, matching the fixture paths, show `renameat2(…, RENAME_NOREPLACE) = 0` and no `link`/`linkat`; paired with a destination the trace shows answered `EEXIST`, the source unchanged | T1 | the contract gate |
| `scripts/contract.sh` §370 | Decision 7 | T1 | the contract gate |

§375 is free on `origin/main` and on every remote branch at `a4040443` (checked under bash: sections reach §371, and §374 is reserved in prose by ADR-208 T2).

No receipt key is added or renamed (ADR-111).

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** on Linux and macOS, where the filesystem supports the flag, a move that must not replace is whole or not at all.
- **Negative:** a second module requirement, with its updates, its licence notice and a second `govulncheck` configuration; the corpus's one-requirement clause no longer holds, and fresh runs of the fences in Context's table read as that table says.
- **Neutral:** the fallback, Windows and the other systems behave as #744 left them; one more `open` and `close` of two directories per move.

## Out of Scope

- Windows' no-replace rename (permanent: boundary: `os.Root.Rename`'s Windows path is Go's handle-relative call and its flags are not exposed; #744's link-then-remove stays there)
- The other BSDs (permanent: boundary: `x/sys/unix` exposes no flagged no-replace rename for them, so they keep #744's path)
- Rewriting the done fences (permanent: boundary: a done task's receipt records what held when it ran; Decision 6)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A filesystem answers the flag with an error not in Decision 2's list | Low | Med | the move fails and says so; T1 adds the error to the fallback list only with a reproduction |
| `x/sys` raises the minimum OS version | Low | Med | T1 checks its build constraints against the release matrix |
| A `govulncheck` finding in `x/sys` blocks a release | Low | Low | updated like any requirement; the static gate names it |

## Rollback

Revert T1: `go.mod` loses the requirement, and #744's path is the only one again.

## History

- 2026-11-20: drafted as Proposed.
- 2026-11-20, Codex xhigh challenge at `bda4dda6`: AMEND, five points, each checked and applied. (1) The guarantee is narrowed to the source entry present at the call, with the fallback, `EXDEV` and the other systems outside it; `EINVAL` is a compatibility fallback, and the list applies to the rename call's errors only. (2) Parents open with `O_DIRECTORY|O_NONBLOCK` (`rooted/fdlimit_other.go:36`); a parent moved out after the open is followed, as `Root.Rename` does; respelling stays ADR-129's and outside the guarantee. (3) The reasons for `x/sys` are the challenge's, checked: Linux could use raw numbers, macOS needs libSystem; `x/sys` v0.49.0 declares Go 1.26 and requires nothing (its `go.mod`). (4) The dependency's cost is tabled (205 fences, 2 plain greps, 17 older counters and 25 baseline comparisons, the last two the challenge's counts); T1 checks the parsed requirements, ships the licence notice and runs `govulncheck` for darwin. (5) T1's seam delegates to the real call, §375 matches paths and results and is required on a Linux runner, a cross-directory case and a native macOS run are added, #744's tests are forced onto the fallback, and §370's Darwin expectation and the receipt promise are amended (the Served-path change no longer says receipts are unchanged).

## Follow-ups

- [ ] None beyond T1.
