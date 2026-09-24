# ADR-066: A rename is checked before anything is written, a failed path-op commit is undone as a unit, and a partial commit says what it wrote

**Status:** Accepted
**Accepted:** 2026-09-24 by M — *"accepted"*
**Date:** 2026-09-24
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-057, docs/adr/BACKLOG.md
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `cmd/mrw/main.go` (`report`), `scripts/contract.sh`, `scripts/break-campaign.sh`
**Enforced-by:** `internal/apply/pathop_commit_test.go::TestAFailedRenameAfterAReplacingRenameLosesNoFile`
**Served-path change:**
- A rename whose destination directory cannot be created now writes nothing and exits 2, with the rename `failed` and the rest `skipped`. It used to write the plan's other files, report every hunk `ok` and `0 failed — applied`, and exit 2.
- A rename whose destination name the filesystem rejects is refused at validation, exit 1, nothing written.
- A failure while committing unlinks and renames undoes every unlink and rename of the plan, instead of restoring only the unlinked files over files already renamed into place, which destroyed data.
- The receipt reports each hunk by what reached disk, and the text summary reads `PARTIALLY APPLIED` when some file was written and a hunk failed.

## Context

**What was observed.** A randomized chaos pass on 2026-09-24 (`main` `c3f5661`, v1.22.3; about 12,600 mrw runs checked against an independent model) generated a plan with a rename and a line edit to another file, where the rename's destination held a byte macOS APFS refuses in a filename. The run:
- wrote the other file;
- printed both hunks `ok` and `2 hunk(s), 1 file(s), 0 failed, 0 advisories — applied`;
- exited 2 with `mrw: notes.txt: mkdir …: illegal byte sequence (ALREADY WRITTEN: x y.txt)`.

A destination directory name of 300 bytes reproduces it on every OS. `--json` then says `"applied": false`, `"failed": 0`, and the rename hunk `"status": "ok"`.

**A cold review of this record's first draft found data loss in the same code.** It was reproduced on the installed v1.22.3 (`63729bd`) the same day:

```
printf 'OLD-C\n' > c.txt; printf 'B-CONTENT\n' > b.txt; printf 'D\n' > d.txt; mkdir ok
mrw read c.txt b.txt d.txt
printf '@@ c.txt - unlink\n@@ b.txt - rename\nc.txt\n@@ d.txt - rename\nok/<300 x>\n' | mrw write -
# every hunk "ok", "3 hunk(s), 3 file(s), 0 failed, 0 advisories — applied", exit 2
# (ALREADY WRITTEN: c.txt, b.txt, c.txt); afterwards c.txt holds OLD-C and B-CONTENT exists nowhere
```

**Why each happens.**
- `apply()` stages only content files (`internal/apply/apply.go:563-595`), renames their temps into place (`:596-603`), and only then runs `commitPathOps` (`:604`, `internal/apply/pathop.go:128`).
- `renameOne` creates the destination directory at commit (`pathop.go:161`), after the other files are on disk.
- Validation checks the destination with `os.Lstat` but acts only on `err == nil` (`pathop.go:112`). ENAMETOOLONG, and any other error that is not "does not exist", passes validation and fails at the commit rename (`:167`).
- On a commit failure, `restore()` (`pathop.go:130-134`) renames each unlinked file's aside back over its original path. It ignores every rename that already completed, including one that moved another file ONTO that path. The moved file is overwritten and lost. Restore errors are ignored too.
- No hunk's status changes on either commit-failure return (`apply.go:599`, `pathop.go:198`). Every hunk stays `ok` from the verdict loop (`apply.go:477-487`), and `report()` (`cmd/mrw/main.go:1539-1545`) prints `applied` whenever `Failed` is 0.
- `writtenSoFar` (`apply.go:616-628`) lists every record in `res.Files`, including the unlink records `restore()` put back, so it named `c.txt` twice.
- ADR-057's stress table records the mutant "`commitPathOps` restore() no-op" as SURVIVED (`docs/adr/ADR-057-unlink-and-rename.md:127`): no test reached a commit-phase failure.

**What the records say.**
- ADR-001 rule 2 put filesystem failures in staging, so an abort writes nothing. Its 2026-09-02 amendment gave that abort honest verdicts.
- ADR-001 accepts that "a failing rename can leave the tree genuinely partial" and names what was written.
- Rule 3 forbids "ok but not written".
- ADR-057 §4 says "Any sibling failure restores the aside". Nothing records a decision to lose a renamed file.

**The class this record governs:** a step after the first commit rename that can fail, and what a failure there leaves behind and reports. Enumerated 2026-09-24 with

```
grep -nE 'os\.(Rename|MkdirAll|Remove|CreateTemp|Lstat)\(' internal/apply/apply.go internal/apply/pathop.go
```

Members on the commit path:
- `apply.go:597` (content rename);
- `pathop.go:137-149` (aside placeholder and move);
- `:161` (destination mkdir), which is stageable;
- `:164` (appeared-before-commit check);
- `:167` (rename);
- `:132` (restore);
- `:213` (final aside removal).

Also `pathop.go:112` (destination check at validation), which is incomplete.

Not governed here, with reasons:
- the placeholder at `:137-147` can fail only on an unwritable or full directory, and T2's unit undo makes that failure non-destructive (deferred);
- a failed final aside removal at `:213` leaves a `.mrw-aside-*` file after a plan that applied, which is ADR-004 hygiene rather than a false receipt (deferred).

## Existing Primitives Audit

- **`missingDirs` / `staged.dirs` / `discard`** (`apply.go:1532-1546`, `:1522-1526`, `:552-562`). Reused to create rename destination directories during staging and to take them back, deepest first, refusing a non-empty directory.
- **The staging-abort verdict block** (`apply.go:575-591`). Extracted and shared.
- **`reportAddressed`** (`apply.go:346-360`). Reused for aborts. The partial path keeps written records rather than calling it, because it zeroes `Written`.
- **`writtenSoFar`**. Kept on the error, and given only records whose file is written.
- **`stageFileFn`** (`apply.go:1465`). The seam pattern copied for `commitRenameFn`.

## Decision

**1. What can be checked before the first write is checked there.**
- A rename's destination directories are created during staging, recorded in a dirs-only `staged` entry after the content entries, and taken back on abort.
- A destination whose `Lstat` fails for any reason other than "does not exist" fails validation, with the error as the reason.
- Both aborts write nothing and use ADR-001's staging-abort verdicts.

**2. A failure while committing unlinks and renames undoes all of them.**
- Every completed rename is reversed, newest first, and only then is each aside restored.
- No undo step overwrites a file. An aside whose path is still occupied by a rename whose undo failed stays where it is, as a recovery file, and the error names it.
- Each undo step is checked, and a step that cannot be undone keeps its file reported as written.

**3. Every commit failure leaves a truthful receipt.**
- Hunks whose file reached disk are `ok`; the hunk whose commit failed is `failed` with its error; every other hunk is `skipped`.
- `Echo` and `Balance` are cleared on every hunk that is not `ok`, and `Advisories` is recounted.
- `Failed` is at least 1 and `Applied` false.
- `files[]` holds the written records plus every other addressed file with `written: false`.
- `writtenSoFar` lists written files only.
- The CLI summary reads `PARTIALLY APPLIED` when a file was written and a hunk failed; that case is tested before `NOTHING WRITTEN`.
- **`skipped` now means "this hunk's file was not written"**, and ADR-001 rule 3 is amended.

## Alternatives Considered

- **Stage renames like content files (copy, then rename).** Rejected: a rename moves the inode; copying changes ADR-057's semantics and doubles I/O.
- **Undo content renames too.** Rejected: the originals are already replaced; keeping backups of every file changes ADR-001's staging design for a failure staging now makes rare.
- **Restore only the asides whose path is free.** Rejected: it keeps the file safe but leaves the plan's renames half done with the unlink undone, a state no hunk describes.
- **A `partial` status or a new JSON field.** Rejected: `files[].written` records it, and a new enum value breaks every caller switching on `ok|failed|skipped`.

## Component / Boundary Impact

- `internal/apply` owns all three decisions.
- `cmd/mrw` changes `report()`'s summary word.
- `internal/mcp` changes only descriptions: the receipt's `failed`, `files.written` and `hunks.status` fields and `mrw_write`'s all-or-nothing sentence now allow a partial commit (Codex review of #207). `writeReport` already prints each hunk's status and elision keeps written files (`internal/mcp/tools.go:660-680`, `:753-777`); a test pins both.
- Byte-identical: `internal/read`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| validation | rename destination `Lstat` errors refuse the hunk | T1 | `mrw write`, MCP `mrw_write` |
| staging | rename destination directories made and taken back | T1 | same |
| path-op commit | a failure undoes every unlink and rename | T2 | same |
| receipts | ok / failed / skipped by what reached disk; Echo, Balance and Advisories consistent | T3 | text, `--json`, MCP |
| CLI summary word | `PARTIALLY APPLIED` | T3 | callers reading the last line |
| contract §118 | long directory → exit 2, nothing written; long leaf → exit 1; short → applies | T1 | CI, `adr-verify` |
| contract §119 | read-only destination directory: the replacing-rename plan loses nothing; a mixed plan is `PARTIALLY APPLIED` | T2, T3 | CI, `adr-verify` |
| break campaign probe `[rename-dest-toolong]` | exit 2, nothing written | T1 | release campaign diff |

Exit codes are unchanged: 1 for a failed hunk at validation, 2 for a filesystem failure at staging or commit.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `abortStage` shared by the staging loops | T1 | T3 | No |
| `commitRenameFn` seam and path-op undo | T2 | T3 | No |
| commit-failure verdicts | T3 | — | Partly: a caller that read `ok` on a failed commit now reads `failed`/`skipped`, which is true |

## Implementation

See `docs/adr/ADR-066-a-plan-that-cannot-commit-whole-says-what-it-wrote/tasks/README.md`.

## Consequences

- **Positive:** the data loss and the half-apply the chaos pass found are gone. A failed path-op commit leaves the unlinks and renames as they were. Every receipt says what reached disk.
- **Negative:** one `MkdirAll` per rename moves into staging and runs even when a later content rename fails; the abort takes it back.
- **Neutral:** the `hunks.status` enum and the JSON fields are unchanged.

## Out of Scope

- Recording the ledger after a partial commit (deferred: docs/adr/BACKLOG.md)
- Staging the unlink placeholder, or probing directory writability before commit (deferred: docs/adr/BACKLOG.md)
- A `.mrw-aside-*` left when the final removal at `pathop.go:213` fails after a plan that applied (deferred: docs/adr/BACKLOG.md)
- Concurrent writers to one file (permanent: boundary: ADR-002 keeps locking out of scope; BACKLOG:502 records the measured loss)
- Undoing content renames already committed (permanent: boundary: ADR-001 accepts a partial tree on a failing rename and names what was written)
- The handshake's shared sentence "if any hunk fails, nothing is written" (`internal/guide/guide.go:17`), which a commit failure now contradicts in the rare partial case (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a dirs-only `staged` entry shifts the index `discard(i)` uses | Med | High | appended after the content entries; a mixed fixture aborts a content rename and checks the rename directory is gone |
| an undo step fails and the receipt claims the tree is whole | Low | High | each undo is checked; a failed undo keeps its record written and names it; a seam test drives it |
| §119 needs a non-root runner to see `chmod 0555` bite | Med | Med | the row probes the directory first and SKIPs visibly as root, as §21 does; the Go tests cover it through the seam |

## Rollback

Revert the validation check, the staging loop, the undo, the verdict function, the seam, `report()`'s case, §118, §119 and the campaign probe. Nothing persistent moves.

## Follow-ups

- [ ] Release with ADR-065 as v1.23.0, with the campaign diffed against v1.22.3. The data loss is named in the tag message.
