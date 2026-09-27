# ADR-086: A name is written as given, or refused before anything is written

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — approved the backlog plan that names this record, having said "we can take our own decisison besides M, since we own this now"
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-004, ADR-066, ADR-071, ADR-076, docs/adr/BACKLOG.md
**Governs:** `internal/plan/plan.go`, `internal/apply/apply.go`, `internal/rooted/links.go`, `internal/rooted/rooted.go`, `cmd/mrw/main.go`
**Enforced-by:** `internal/plan/bytes086_test.go::TestAHeaderKeepsEveryByteOfItsPath`
**Invalidates:** ADR-066 T3's contract §119 trigger: a rename into a read-only directory no longer reaches commit, because the staging probe is refused there; §119 now asserts nothing is written, and the partial-commit report is driven through the seam
**Served-path change:** a plan header keeps every byte its author wrote, so a path, anchor or pattern holding bytes that are not valid UTF-8 is no longer rewritten to U+FFFD; and a create target or rename destination the filesystem cannot hold (on APFS, a name that is not valid UTF-8; any name in a directory mrw cannot write) is refused at staging — the hunk fails, exit 2 as a filesystem failure, nothing written — instead of landing under another name at exit 0 or failing at commit as PARTIALLY APPLIED.

## Context

**What was observed** (BACKLOG, "Seen, 2026-09-25", chaos seed 25 on v1.25.0; reproduced
2026-09-27 on APFS against `main` at `537b896`):

- `@@ b.txt - rename` with destination `moved/\xffdash.txt` passes validation and fails at commit
  with `illegal byte sequence`; the plan's content edit stays written, PARTIALLY APPLIED, exit 2.
  APFS refuses the name on create and rename (EILSEQ) while `lstat` of it answers ENOENT, so
  ADR-066's staging `Lstat` passes it.
- Found while reproducing it: `@@ bad\xffname.txt 0 create` exits 0 and creates
  `bad\xef\xbf\xbdname.txt`. `splitHeader` (`internal/plan/plan.go`) walked the header as
  `[]rune(s)`, which replaces each invalid byte with U+FFFD, so the path, an anchor or a pattern is
  silently another string. A rename destination is a body line, never through that walk, which is
  why the two ops failed differently.

**The class.** Names a plan leaves on disk: every `create` target and every `rename` destination
(`internal/apply/apply.go`, the staging loop and ADR-066's rename-directory loop). Strings a header
carries: the path, `anchor=`, `sha=`, `lines=`, `body=` and a pattern address — all through
`splitHeader`. `mrw read --grep '\[\]rune\(|strings\.Map\(' internal cmd` on 2026-09-27 found that one
conversion and no other.

## Existing Primitives Audit

- ADR-066's staging loop already makes a rename's missing directories and re-asks the destination;
  the probe goes in the same place, before anything is renamed.
- `stageFileFn` and `commitRenameFn` are the seams the staging and commit tests drive; `probeNameFn`
  is one more.

## Decision

1. `splitHeader` walks the header byte by byte. Every delimiter it looks for is ASCII, and a UTF-8
   continuation byte is never one, so valid text splits as before and an invalid byte survives.
2. At staging, a name that does not exist yet — each create's target, each rename's destination —
   is created exclusively and removed again. A name the filesystem refuses fails its hunk, exit 2
   (a filesystem failure), and nothing is written; ADR-066's commit-time handling stays for failures a probe cannot see.
3. A probe that cannot be removed is reported by name, as a `.mrw-aside-*` is (ADR-004).
4. On Windows a path component that is not valid UTF-8 is refused by name, as a Win32 alias is
   (ADR-071 Decision 4): Windows converts a path to UTF-16 and maps such a byte to U+FFFD without an
   error, so the probe and the commit would both reach the replacement name (Codex review of #254).
5. A `--json` plan whose path, rename destination or expanded working-set pointer is not valid UTF-8
   is refused before anything lands: `encoding/json` would name the file with U+FFFD in the receipt
   (Codex review of #254). A name the filesystem supplies is not covered (Out of Scope).

## Alternatives Considered

- **Refuse names that are not valid UTF-8 on darwin, by string, in `rooted.Resolve`.** Rejected: an
  HFS+, FAT or SMB volume on macOS may hold such a name, and reads of it would be refused too.
- **Normalise a header to valid UTF-8 on purpose.** Rejected: a path that means something else than
  its author wrote is the defect ADR-076 refuses.

## Component / Boundary Impact

`internal/plan` (T1) and `internal/apply` (T2).

## Wiring & Contract Changes

A header's bytes reach the engine unchanged; a new refusal at staging. Contract §168 drives the built
binary on whichever filesystem runs it.

## Inter-task Contracts

T2's refusal is reachable for a create only once T1 delivers the name the author wrote.

## Implementation

See `tasks/`.

## Consequences

- A plan that created a U+FFFD-named file by accident on a filesystem that refuses the raw byte is
  now refused instead; on one that accepts it, the file gets the bytes the author wrote.

## Out of Scope

- A contract row that shows the APFS refusal on CI (permanent: fact: the contract script runs on Linux CI only, where ext4 accepts the byte, and §168 checks whichever the running filesystem does; citation: file `AGENTS.md:63`)
- Two distinct invalid bytes treated as one name by the case-fold key (deferred: docs/adr/BACKLOG.md, "foldKey collapses distinct invalid bytes")
- A probe that cannot be removed leaves a file while the receipt says nothing was written (deferred: docs/adr/BACKLOG.md, "A probe left behind")
- A `--json` receipt naming a filesystem-derived name that is not valid UTF-8 — a symlink's target, a created directory, the root (deferred: docs/adr/BACKLOG.md, "A --json receipt with a filesystem-derived invalid name")
- A Windows absolute read spec that `rooted.Real` normalises before the alias check (deferred: docs/adr/BACKLOG.md, "A Windows absolute spec normalised before the alias check")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A probe name collides with a file another process creates between staging and commit | Low | Low | the exclusive create refuses it, which is the "appeared before commit" case ADR-071 already stops |

## Rollback

Revert the tasks.

## Follow-ups

None.
