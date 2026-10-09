# ADR-129: a rename may change only the case of a name

**Status:** Accepted
**Accepted:** 2026-10-06 by Zy — "case-only rename" among the items chosen for this round, then "go next, properly, build a spec/adr and execute. i want complete stability and completeness". The record's text was drafted after that answer and revised after the Codex review of the record.
**Date:** 2026-10-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-021, ADR-057, ADR-066, ADR-106, ADR-125
**Invalidates:** None — ADR-057's "dest exists: refuse" stays for every destination that is another entry; this record says when the destination the filesystem finds is the source itself
**Governs:** `internal/apply/pathop.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/apply/respell129_test.go::TestACaseOnlyRenameApplies`
**Served-path change:** on a filesystem that folds case (APFS, NTFS by default), `@@ a.txt - rename` with the body `A.txt` renames the file to `A.txt`, where it was refused "rename dest A.txt already exists".

## Context

`planPathOp` (`internal/apply/pathop.go:122`) refuses a rename whose destination `os.Lstat` finds. On a filesystem that folds case, `A.txt` finds `a.txt` — the source — so the only way to change a name's case was two plans through a third name. BACKLOG "From the Windows peers" carries it; the 2026-10-06 plan chose it for this round.

Measured 2026-10-06 on this Mac (APFS, Go 1.27.1): `os.Root.Rename("a.txt", "A.txt")` returns nil and `os.ReadDir` then lists `A.txt`. Measured the same day by a Windows peer (Windows 11 10.0.26200, NTFS, a directory whose case-sensitive attribute is disabled, Go 1.25.0): `os.Root.Rename` and `os.Rename` both return nil and the listing shows the new spelling. So the rename needs no intermediate name on either; the `windows-latest` CI shard runs every test in this record (T1 S1) and is the evidence of record.

Two shapes must stay refused, because each is a rename that would do nothing and report ok — the failure this tool exists to prevent:

- **A hard link under another name.** `b.txt` hard-linked to `a.txt` is the same file (`os.SameFile`) under a name that really exists. POSIX `rename(2)` of one onto the other "does nothing, and returns a success status" (rename(2), Linux man-pages 6.9).
- **A respelling of only the directory.** `d/a.txt` → `D/a.txt` names the same entry with the same leaf: nothing to rename.

**Audit of the class** — *a path op whose destination the filesystem resolves to the source*: `rename` is the only op with a destination (`mrw read --grep '"rename"' internal/apply/` names `planPathOp`, the staging block at `apply.go:944-984` and `renameOne` at `pathop.go:249`); `unlink` has none. ADR-021 already refuses a plan that names one file under two spellings as two hunk paths; a rename's destination is not a hunk path, so ADR-021 does not reach it.

## Existing Primitives Audit

- **`os.SameFile`** — the filesystem's own identity answer, as ADR-021 and ADR-106 use it; nothing folds case in mrw.
- **`commitRenameFn` / `(*tree).rename`** — the commit rename through `os.Root`; the respelling is one more call of it, undone by the same `undo` (ADR-066).
- **`os.ReadDir` of the parent** — the only way to see the spelling a directory holds; `Lstat` answers for any spelling that folds to it.

## Decision

1. **A destination is a respelling of the source when** its parent and the source's parent are the same directory (`os.SameFile`), `Lstat` of the destination is the same file as the source, the leaf names differ, no entry in the directory is spelled exactly as the destination leaf, the directory lists the source exactly as the plan spells it, and no other entry there is the source's file. The last two are from the Codex review of the record: a source the plan spells otherwise (`foo.txt` for `Foo.txt`) would be undone to a name it never had, so it is refused "not spelled that way in its directory"; and a hard link under another spelling (`B.txt`, with the plan asking `b.txt`) folds to the destination while being another entry, so it is refused "already exists", as every other existing destination is.
2. **A destination whose leaf is spelled exactly as the source's, under a parent that is the same directory spelled another way, is the source**, and is refused "rename dest … is the source", as `dest == path` already is. **And a respelling spells the directory as the source does**: `a/x.txt` → `A/X.txt` passes every identity test, renames the leaf only, and left `a/X.txt` on disk under a receipt naming `A/X.txt` (the in-process review of #346); it is refused "respells the directory".
3. **The commit renames directly** — `renameOne` skips its "appeared before commit" refusal for a respelling only while Decision 1 still holds, asked again at commit — **and then, once the rename is recorded for the undo, checks that the source's spelling has left the listing.** A filesystem that reported success and kept the old spelling (or a rename that did nothing) fails that hunk, and the path operations are undone (ADR-066). Content writes in the same plan commit before path operations, so they stay written and the receipt says PARTIALLY APPLIED, as for any commit failure. The check is on the source's spelling leaving, not on the destination's bytes arriving, because a filesystem that stores names decomposed (HFS+) lists a new case in another normalisation.
4. **Nothing else changes**: the receipt reports the source removed and the destination created, as every rename does; the ledger records the destination as written.

## Alternatives Considered

- **Always through an intermediate `.mrw-case-*` name** (the plan's first sketch) — rejected: APFS renames directly (measured above), and an intermediate adds a second rename, a second undo step and a window where a crash leaves the file under a name nobody chose. Kept as the fallback if the Windows shard shows `os.Root.Rename` refusing a case-only rename; T1's stop condition.
- **Fold case in mrw (`strings.EqualFold`)** — rejected: Unicode folding is not the filesystem's (APFS normalises, NTFS uses its upcase table), and on Linux two such names are two files. ADR-021 chose `os.SameFile` for the same reason.

## Component / Boundary Impact

`internal/apply` only. Engine change owned by this record: `pathop.go`. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `rename` op | a case-only respelling applies on a folding filesystem | T1 | CLI and MCP callers |
| `scripts/contract.sh` | §230 | T1 | CI Linux (the hard-link pair); local macOS (the respelling) |
| `AGENTS.md` | §2's rename sentence | T1 | every agent |

No exit code and no receipt key changes.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a name's case is changed in one plan, on the filesystems where it was impossible.
- **Negative:** one directory listing per respelling, at validation and after the commit.
- **Neutral:** where two spellings name two files, nothing changes. Where the filesystem folds normalisation but not case (case-sensitive APFS), Decision 1 can allow a normalisation-only respelling by the same rule.

## Out of Scope

- A respelling of a directory component (permanent: boundary: mrw renames files; a directory is never a plan path, ADR-057)
- A test of a Unicode-normalisation-only respelling (é as one code point or two) (permanent: boundary: Decision 1 decides it by identity and listing like a case respelling, and Decision 3's check is on the source's spelling leaving, so it holds on APFS and on HFS+; on HFS+ a source named in a normalisation the directory does not list is refused as "not spelled that way". No caller has asked for it, and the CI runners hold no such filesystem)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| `os.Root.Rename` refuses a case-only rename on Windows | Medium | the respelling fails at commit, undone | the Windows CI shard runs T1's tests before merge; the intermediate-name fallback is the stop condition |
| a hard link mistaken for a respelling | Low | a rename that does nothing reports ok | the listing check; `TestAHardLinkUnderAnotherNameIsStillRefused` on every platform |

## Rollback

Revert T1: a case-only rename is refused again. No persistent state changes.

## Follow-ups

- None — the record carries no open follow-up.

## Amendment 2026-10-09 — the Windows chaos round

Two refusals of a case-only rename named the wrong cause (a quality-blueprints Windows session, Windows 11 10.0.26200, v1.52.0). (1) `respelling` called `os.Lstat` on every entry of the directory, and Win32 cannot open a name that ends in a dot or a space, so one such sibling made the call fail and the failure was read as "the destination is another entry": every case-only rename in that directory was refused "already exists". (2) A plan that spelled the source otherwise than its directory does, and named as the destination the spelling the directory holds (`PLAIN.TXT` → `plain.txt` on a disk holding `plain.txt`), met the early "the destination is a listed name" return and was refused "already exists", where the cause is the source's spelling. Now each listed entry is asked through `lstatEntry`. On Windows a listed name that ends in a dot or a space is asked through its extended-length path FIRST (`extendedPath`: a drive path gains `\\?\`, an ordinary UNC path becomes `\\?\UNC\…`, a device-form drive path `\\.\C:\…` becomes `\\?\C:\…`, an extended path is kept), because Win32 strips the dot or space from an ordinary spelling: a listed `dot.` is "not found" and a listed `plain.txt.` opens `plain.txt` itself and would be counted as a second link to the source. So a sibling named `dot.` is compared with the source like any other, and a second hard link to the source that is itself named `dot.` (CreateHardLink accepts an extended path) still counts toward Decision 1 and refuses the rename (the Codex reviews of #360, P2, three times). An entry not found either way has gone and is passed over. Elsewhere `lstatEntry` is `os.Lstat`. The destination's own leaf, listed as the source's entry, falls through to the `otherSpelling` refusal ("name it as the directory lists it"). A hard link, and a listed destination that is another file, are refused as before. Pinned by `internal/apply/respell1052_test.go::TestARenameWhoseDestIsTheDirectorysOwnSpellingNamesTheSourcesSpelling`, on Windows `internal/apply/respell1052_windows_test.go::TestAnUnopenableSiblingDoesNotRefuseACaseOnlyRename`, `TestAnUnopenableHardLinkOfTheSourceStillRefusesACaseOnlyRename`, `TestAnExtendedPathSpellsEveryKindOfAbsolutePath` and `TestAnOrdinaryUNCRootGetsTheSameAnswers` (skipped where the administrative share is not reachable), and contract §235. Red first on windows-latest: runs 37970280110 and 37971494595. Not covered: the Windows paths were run on windows-latest only, not on a desktop; the UNC and device-form paths are source-traced and exercised only where `\\localhost\C$` answers; a listed name that no spelling can open is passed over, and no such name is known.
