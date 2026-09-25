# ADR-071: A path reaches the file it names on disk, or is refused

**Status:** Accepted
**Accepted:** 2026-09-25 by M — *"plan to address these, properly, no looping on small details"*; the plan that groups the v1.25.1 adversarial round into five records was approved the same day
**Date:** 2026-09-25
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-004, ADR-005, ADR-006, ADR-021, ADR-029, ADR-069, docs/adr/BACKLOG.md
**Governs:** `internal/rooted/**`, `internal/apply/apply.go`, `internal/read/read.go`, `internal/read/walk.go`, `internal/apply/create_identity_test.go`, `internal/check/*_test.go`, `cmd/mrw/paddedflag_test.go`, `cmd/mrw/junction_windows_test.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `internal/guide/guide.go`, `internal/guide/guide_test.go`
**Enforced-by:** `internal/apply/create_identity_test.go::TestTwoCreatesThatCouldBeOneFileAreRefused`
**Invalidates:** ADR-021 Decision 4 ("Only existing files are checked") — two creates are now checked by name; ADR-021 Decision 5 ("Nothing folds case") now holds for EXISTING files only
**Served-path change:** On Windows a junction under `--root` is followed and judged like a symlink, so a read, replace, create, rename or unlink through a junction that leaves the root is refused; a path component ending in `.` or a space, or holding a `:`, is refused by name on Windows. On every platform a plan that creates one path twice, or leaves two paths on disk that differ only by case (Unicode simple case folding) where at least one does not exist yet, is refused, exit 1, nothing written; a create whose name the filesystem folds into one an earlier create made (ß and ss, NFC and NFD) stops the commit there, PARTIALLY APPLIED, exit 2, instead of renaming over it.

## Context

**What was observed.** The v1.25.1 adversarial round (2026-09-25, twelve peer sessions; findings in
`docs/adr/BACKLOG.md`, "From the v1.25.1 adversarial round") found four things about paths.

1. **A junction inside `--root` escapes it (Windows).** Three Windows sessions confirmed it with the
   CLI, the release asset and MCP: read, replace, create, rename into it and unlink through it all
   landed outside the root at exit 0, while `..` and POSIX symlinks were refused. A junction needs
   no privilege. The mechanism, read from the Go source: since Go 1.23 (`winsymlink=1`, the default
   for this module's `go 1.26.6`), `os.Lstat` gives a junction `ModeIrregular`, not `ModeSymlink`,
   and `filepath.EvalSymlinks` no longer evaluates mount points. `internal/rooted/rooted.go:53`
   then fails with ENOTDIR at the junction, treats the error as a missing leaf, walks up to an
   ancestor that resolves lexically, and `Contains` answers "inside".
2. **Two creates are never compared.** Two `create` hunks for one path both reported ok and the
   bodies were concatenated (each becomes an insert at `apply.go:959`, and an insert never advances
   the overlap cursor). Two spellings of a NEW file on a case-insensitive filesystem (`n.txt` +
   `N.TXT` on APFS and NTFS; `n.txt` + `n.txt.` on NTFS) both reported "created"; one file remained
   and the first body was lost, exit 0. ADR-021's same-file check stats files, and a create has
   nothing to stat (`apply.go:373-374` says so). Reproduced on macOS with v1.25.1.
3. **Win32 name aliasing (Windows).** `b.txt.`, `sp.txt ` and `b.txt::$DATA` reach `b.txt`, so a
   write or an unlink lands through a name that does not exist on disk and the receipt names the
   alias. `-C 'dir '` and `-C 'dir.'` were accepted.
4. **The Go suite on Windows.** Eleven `internal/check` tests FAIL rather than skip when `sh` is not
   on PATH (PowerShell), `TestTailAnnouncesWhatItLeftOut` panics indexing an empty tail, and all
   thirteen padded-path tests in `cmd/mrw` SKIP on NTFS because their fixture needs a file named
   `x `, which Win32 cannot hold beside `x`. Two Windows sessions reported it.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `rooted.Resolve`, `rooted.Abs` | `internal/rooted/rooted.go:35-75` | The one boundary (ADR-006 Decision 2). Resolves with `EvalSymlinks`, which since Go 1.23 does not follow a junction. |
| `os.Readlink` | Go `os/file_windows.go:475-498` | Follows `IO_REPARSE_TAG_SYMLINK` and `IO_REPARSE_TAG_MOUNT_POINT`; any other tag returns `ENOENT`, "not a symlink or junction but another type of reparse point". |
| `ModeIrregular` | Go `os/types_windows.go:205-228` | Set on every reparse point except symlink, AF_UNIX and dedup: junctions, and also OneDrive placeholders, which redirect nothing. |
| `sameFileAs`, `groupedFile` | `internal/apply/apply.go:392-428`, `:1776-1793` | ADR-021: one inode, one spelling. Existing files only. |
| `produced`, `destCount` | `internal/apply/apply.go:375-391` | String-keyed: what the plan leaves on disk, and each rename destination. |
| `paddedTree` | `cmd/mrw/paddedflag_test.go:10-24` | Creates `x` and `x `; skips where the filesystem folds them. |

## Decision

1. **A junction is followed like a symlink, in the one boundary.** On Windows, `Resolve` and `Abs`
   first walk the path component by component; a component whose `Lstat` mode is `ModeSymlink` or
   `ModeIrregular` is replaced by its `os.Readlink` target, and the walk restarts on the result.
   A `Readlink` that fails with Go's "another type of reparse point" `ENOENT` leaves an irregular
   component as it is: it redirects nothing (a OneDrive placeholder). Any other failure, a symlink
   that cannot be read, or a component `Lstat` cannot examine, refuses the path; only a component
   that does not exist ends the walk. The walk is platform-independent code driven through an
   injected `Lstat`/`Readlink`; only the choice to run it is Windows-only, so the POSIX path is
   byte-for-byte what it was. `rooted.Real` resolves an absolute argument the same way, and
   `read` and the `--grep` walk use it, so a root reached through a junction and a path inside it
   are compared in one spelling.
2. **A create is the only hunk of its file.** A second `create` of one path fails:
   `<path> is created twice in this plan (plan lines A and B); one create per file`.
3. **Names that differ only by case are one file, whatever the filesystem.** Every name the plan
   will leave on disk (a path with a non-path op, and every rename destination) is compared under
   Unicode simple case folding, the equality `strings.EqualFold` uses (`s` and `ſ`, `σ` and
   `ς`). Two different names with one folded form, where at least one does not exist yet, are
   refused on the later one: `<b> may name the same file as <a> (plan line N) on a
   case-insensitive filesystem; one file, one spelling per plan`. Two EXISTING files are left to
   ADR-021's `os.SameFile`, so `a.txt` and `A.txt` on ext4 still both apply. A rename's own source
   is not a name the plan leaves, so a case-only rename is untouched. Trailing dots, spaces and
   `:` are NOT folded here: on POSIX `10:00.log` and `10:01.log` are two files, and on Windows
   Decision 4 refuses those spellings before any comparison.

   **And the commit catches the folds a name cannot show.** APFS also folds full case forms (`ß`
   and `ss`, `ﬁ` and `fi`) and ignores Unicode normalization (NFC and NFD), and comparing
   those needs tables the standard library does not carry. So a create whose target exists by the
   time it is committed stops the commit there: PARTIALLY APPLIED, exit 2, naming what landed,
   instead of renaming over the earlier file at exit 0. It also catches another process creating
   the file between validation and commit.
4. **Win32 aliases are refused on Windows.** A component of a caller's path, or of the root, that
   ends in `.` or a space (other than `.` and `..`), or that holds a `:`, is refused by name:
   Windows drops the trailing characters, cannot create such a name, and reads `:` as a stream.
5. **The Windows suite reaches its branches.** Tests that need `sh` skip without it; the tail test
   stops before indexing; the padded-path refusals, which fire before any I/O, get a fixture that
   does not need a file named `x `.

**What would make this decision fail:** a Windows reparse point that redirects a path but is neither
a symlink nor a junction, which `Readlink` cannot follow and the walk then treats as ordinary.

## Alternatives Considered

- **`os.Root` for every file access.** It confines at the OS, junctions included. Rejected for this
  record: it rewrites the open, stage and rename paths of `apply`, `read`, `walk` and `pathop`, four
  engine packages and the whole commit sequence, to close an escape one boundary function owns.
- **`godebug winsymlink=0` in `go.mod`.** One line, and `EvalSymlinks` follows mount points again.
  Rejected: it pins every `Lstat` in the binary, `os.Root` in `internal/state` included, to a
  compatibility mode Go keeps for a limited time.
- **Refuse every junction.** Rejected: a junction that stays inside the root is ordinary on Windows
  (pnpm's `node_modules`), and refusing it would refuse the tree.
- **Probe the filesystem's case sensitivity during validation.** Rejected: a probe is a write
  (ADR-004), and a plan's validity would then depend on the directory it happened to name.

## Component / Boundary Impact

`internal/apply` is an engine package and this record owns its create checks and the commit guard
(Decisions 2 and 3). `internal/rooted` is the boundary package (ADR-006). `internal/read` changes in
two lines, the absolute-path pre-screens of `read.go` and `walk.go`, which now call
`rooted.Real`. `internal/check` changes in its test files only. Byte-identical: `internal/plan`,
`internal/seen`, `internal/state`, `internal/lines`, `internal/iter`, the rest of
`internal/read`, and `internal/check` outside its test files.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `rooted.Resolve`, `rooted.Abs` (Windows) | follow a junction; refuse a Win32 alias | T1, T3 | read, write, check, iter, `body=@`, MCP |
| plan validation | a second create of one path; two names with one fold key | T2 | CLI `write`, MCP `mrw_write` |
| contract §141 | T2 | T2 | CI, `adr-verify` |

## Inter-task Contracts

None: T1 and T3 share `internal/rooted` and T3 lands after T1; T2 is `internal/apply`; T4 is tests;
T5 is prose.

## Implementation

See `docs/adr/ADR-071-a-path-reaches-the-file-it-names/tasks/README.md`.

## Consequences

- **Positive:** the only confinement escape the round found is closed, and no create loses a body
  at exit 0: a case pair is refused before anything is written, and a fold only the filesystem
  knows stops the commit, PARTIALLY APPLIED, exit 2.
- **Negative:** on a case-sensitive filesystem a plan that creates `n.txt` and `N.TXT` together is
  refused; the message says why, and two plans do it.
- **Neutral:** no exit code changes meaning.

## Out of Scope

- An in-root symlink followed on write with its target absent from the receipt (deferred: docs/adr/BACKLOG.md)
- Plan-header paths cleaned rather than refused (`a.txt/`, `./a.txt`) (deferred: docs/adr/BACKLOG.md)
- 8.3 short names and case aliases of EXISTING files (permanent: fact: the OS resolves them to one file and the ledger's `os.SameFile` lookup matches it; citation: file `internal/apply/apply.go:1752`)
- A grandchild of a check or ast-grep outliving mrw on Windows (deferred: docs/adr/BACKLOG.md "From the v1.25.1 adversarial round", the killed-check entry; ADR-072 owns the subprocess bound)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A non-junction name-surrogate reparse point (WCI, a WSL symlink) redirects a path | Low | Med | Win32 cannot traverse a WSL symlink; a WCI link lives inside containers; the Windows junction test pins the common case |
| The walk runs per component on every Resolve on Windows | Med | Low | one `Lstat` per component; POSIX is untouched |
| The case comparison refuses a legitimate case-sensitive plan | Low | Low | the refusal names both paths; split the plan |

## Rollback

Revert the walk, the two create checks and the alias check. Nothing persistent moves.

## Follow-ups

- [ ] A Windows peer re-runs the junction repro (team memory, `incidents`) against the v1.26.0 release asset.
- [ ] Release with ADR-072 to ADR-075 as v1.26.0.
