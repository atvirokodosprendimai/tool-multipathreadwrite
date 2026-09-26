# ADR-076: A path means what it says, and the receipt names every path a write touched

**Status:** Accepted
**Accepted:** 2026-09-26 by M — approved the plan to clear the open backlog (*"build a plan to address them at once, no dangling pieces, no dead code, no mockery, no features only in tests, all is wired, all is exercised"*), and answered *"Refuse (Recommended)"* for read-only files
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-005, ADR-006, ADR-008, ADR-021, ADR-066, ADR-071, ADR-073, docs/adr/BACKLOG.md
**Governs:** `internal/rooted/**`, `internal/read/read.go`, `internal/read/trailing076_test.go`, `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/apply/attrs_windows.go`, `internal/apply/attrs_other.go`, `internal/apply/paths076_test.go`, `cmd/mrw/main.go`, `cmd/mrw/receipt076_test.go`, `cmd/mrw/device_windows_test.go`, `cmd/mrw/attrs_windows_test.go`, `internal/mcp/schema.go`, `internal/mcp/era_test.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `internal/mcp/receipt076_test.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/apply/paths076_test.go::TestAHunkPathEndingInASeparatorIsRefused`
**Invalidates:** none
**Served-path change:** a spec, a plan path or a rename destination that ends in a separator and does not name a directory is refused (read: `UNREADABLE`; write: the hunk fails, exit 1, nothing written); a root that does not exist is named as missing, and a create under it no longer makes it; on Windows a name the OS opens as a device (`NUL`, `CON`, `COM1`, …) is refused; a line edit, unlink or rename of a read-only file is refused, naming `chmod u+w` (`attrib -r` on Windows); a write keeps a Windows file's Hidden and System attributes; an insert into an empty file ends with a newline; the receipt gains `files[].target` (the file a write through an in-root symlink changed) and `dirs_created` (the directories the plan made), and the human receipt prints them and a removed file's former sha instead of `sha ` and nothing.

## Context

**What was observed.** The v1.25.1 adversarial round (BACKLOG, "Smaller, recorded as found", and
the review of #228) listed paths that do not mean what they say, and receipts that do not say what
changed. Each was reproduced on v1.26.0 on 2026-09-26:

- `mrw read a.txt/` served a.txt; `@@ a.txt/ 1 replace` edited a.txt; a rename to `d/` made a FILE
  named `d`. `filepath.Clean` drops a trailing separator before anything looks at it
  (`apply.go` grouping, `pathop.go` destination, `rooted.Resolve`'s `filepath.Join`), while the OS
  refuses `open("a.txt/")` with ENOTDIR.
- `mrw -C nope read a.txt` said "a.txt resolves to <parent>, which is outside the root <nope>":
  `rooted.Abs` fell back to the spelling when the root did not exist, and a create under it made the
  root.
- On Windows `NUL`, `CON`, `COM1`, `LPT1` and, before Windows 11, the same names with an extension
  open a DEVICE. `mrw read NUL` served an empty file, and a plan could write to a device at exit 0.
  ADR-071 refused a trailing dot, a trailing space and a `:`; device names are the same class.
- A read-only file was replaced (a new file renamed over it keeps mode 0444) and unlinked at exit 0:
  neither needs the file to be writable, only its directory. On Windows a write stripped Hidden.
- A write through an in-root symlink followed it (ADR-005 §2, kept) and the receipt named only the
  link; the directories a create or a rename made (`missingDirs`, ADR-066) were not named; a removed
  file's line printed `sha ` and nothing.
- An insert into an empty file wrote no final newline: `readLines` reports `final=false` for a file
  with no lines, and the write kept it.

**What is kept, with its record.** `./a.txt` and `sub/../a.txt` are cleaned on purpose (ADR-005 §1;
`TestTwoSpellingsOfOnePathAreOneFile`, `TestAPathThatClimbsAndReturnsIsStillInsideTheRoot`). A
quoted header path is the quoting feature (`TestQuotedFieldsSurvive`). A write through an in-root
symlink still follows it (`TestEditingThroughASymlinkKeepsTheSymlink`). Only a TRAILING separator is
a different name, because it names a different kind of thing.

## Existing Primitives Audit

- `rooted.Resolve` is the one boundary (ADR-006); a read spec reaches it unclean, a plan path does not.
- `win32Alias` (`links.go`) is ADR-071's pure Windows-name check, called only when `followLinks`.
- `refuseFile` (`apply.go`) is the one-refusal-per-file shape ADR-021 and ADR-073 use.
- `missingDirs` already records, before `MkdirAll`, the directories staging makes (ADR-066).
- `stageFile` already resolves a link to its target and copies the mode bits.

## Decision

1. **A trailing separator names a directory.** `rooted.EndsInSeparator` is the predicate. `Resolve`
   refuses a path that ends in one and names an existing non-directory, wrapping
   `rooted.ErrNotADirectory`; `read` reports it `UNREADABLE`, keeping the separator through its
   absolute-path branch. A plan names files, so apply refuses a hunk path spelled with one whether or
   not it exists, and `planPathOp` refuses a rename destination spelled with one, naming the file to
   write instead (`d/a.txt`). It is not a move into a directory.
2. **A root that does not exist says so.** `rooted.Abs` returns "the root X does not exist" (and "is
   not a directory"), so every surface names the root, and a create cannot make it.
3. **A Windows device name is refused.** `win32Device` picks a candidate by Go's own rule
   (`internal/filepathlite` `isReservedName`: the name before its first `.` or `:`, trailing spaces
   dropped), and `opensDevice` asks the OS — `GetFullPathName` answers `\\.\NUL` for a device —
   because Windows 11 opens `nul.txt` as a file (the Windows CI job writes `nul.bin`). Only the last
   component is a candidate: a device name mid-path cannot be made as a directory, and fails loudly.
4. **A read-only file is refused** for every op that would change it (M, 2026-09-26). A line edit
   judges the file it reaches; an unlink or rename judges the entry (`Lstat`), so a link to a
   read-only file can still be removed. The mode bits decide, not EACCES, so it holds under uid 0.
5. **A write keeps Hidden and System** on Windows (`keepAttributes`, before the commit rename).
6. **An empty file ends its lines with a newline**, as a created file does: it had no last line whose
   terminator could be kept (ADR-005 §3 still keeps a real file's missing newline).
7. **The receipt names every path a write touched**: `files[].target`, `dirs_created` (parents first;
   none on a dry run), and a removed file's `was sha` on the human receipt. `hunks.path` is described
   as cleaned, which it always was.

## Alternatives Considered

- **Treat a rename to `d/` as a move into `d`** (as `mv` does). Rejected: a new meaning for an old
  spelling; naming the file is one more word and says exactly what happens.
- **Refuse every device-shaped name by list.** Rejected: `aux.c` and `con.h` are files on Windows 11,
  and the CI runner writes `nul.bin`; the OS is the authority.
- **Refuse on EACCES rather than the mode bits.** Rejected: uid 0 and Windows backup semantics make
  EACCES vary; the mark is what the owner set.

## Component / Boundary Impact

`internal/rooted` (Resolve, Abs, the Windows name checks), `internal/apply` (validation, staging,
receipt fields), `internal/read` (the spec path), `cmd/mrw` (the human receipt), `internal/mcp`
(schema descriptions). The engine packages `plan`, `check`, `state`, `lines`, `iter`, `seen` and
`subproc` are unchanged.

## Wiring & Contract Changes

New refusals: exit 1 for a plan (the hunk fails, nothing written), `UNREADABLE` and exit 1 for a read.
New receipt keys `files[].target` and `dirs_created`, both omitted when empty. Contract §151–§153.

## Inter-task Contracts

T1 produces `rooted.EndsInSeparator` and `rooted.ErrNotADirectory`, which T1's apply refusal and
`planPathOp` consume. T2 and T3 are independent of each other.

## Implementation

See `tasks/`.

## Consequences

- A caller that relied on `a.txt/` reaching a.txt, or on a rename to `d/` making `d`, is refused and
  told the spelling that works.
- A write to a read-only file needs the mark cleared first; the refusal names the command.
- `--json` consumers see two new optional keys.

## Out of Scope

- A trailing separator in an apply_patch or search_replace document (permanent: fact: those formats compile to a native plan, so the apply refusal covers them)
- Reading a Windows device on purpose (permanent: boundary: mrw reads and writes files)
- A device name in the middle of a path (permanent: fact: it cannot be made as a directory, so the write fails loudly)
- Windows job objects and other process matters (permanent: boundary: not a path question; ADR-072)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A read-only file a caller meant mrw to edit | Low | Low | the refusal names the command that clears the mark |
| `MoveFileEx` refuses to replace a System file | Low | Medium | the Windows CI test writes a Hidden+System file |
| A caller parses the human receipt line for a removed file | Low | Low | the line keeps `removed <path>`; `--json` is the stable surface |

## Rollback

Revert the three tasks. No state format changes.

## Follow-ups

- [ ] Release with ADR-077 to ADR-080 as v1.27.0.
