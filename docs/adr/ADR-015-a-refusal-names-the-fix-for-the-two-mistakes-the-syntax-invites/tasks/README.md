# ADR-015 Tasks

Implementation tasks for ADR-015: a refusal names the fix, for the two mistakes the syntax invites.
See the parent ADR for the decision and for the heredoc terminator it refuses.

**Source of truth:** the task file's headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

One task. The two hints are independent code paths but one idea, one review and one contract row;
splitting them would triple the ceremony for four lines of Go each.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Say the escape, on both failure paths | done | — | 4 named `--- PASS:` lines, `# 49.` in `contract.sh`, gofmt + vet, `./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

## Notes

- ⚠ **A MERGE CAN COMMIT CONFLICT MARKERS INTO A FILE NO GATE READS, AND THEY
  TRAVEL.** `main` shipped `<<<<<<<`, `=======` and `>>>>>>>` into
  `docs/adr/BACKLOG.md` at `d90be24` — #76's squash merge — and #77 and #78 both
  inherited them by branching from it. The first review of this branch attributed
  them to THIS merge and corrected itself; the origin matters, because a marker
  on a branch is one PR's mistake and a marker on `main` is every later branch's
  inheritance. Nothing noticed: CI does not read that file, `adr-debt` found every
  deferral it was looking for on BOTH sides of the markers, and `adr-lint`
  treats BACKLOG.md as prose. A reviewer found it by eye. §49 now carries a
  tree-wide `git grep` for markers, because the class is "a merge artifact in a
  file no gate parses" and a gate over the files that ARE parsed cannot see it.
  Run `git grep -n -E '^(<<<<<<< |=======$|>>>>>>> )'` before committing any
  merge.

- ⚠ **A deferral pointing at `BACKLOG.md` needs an ENTRY in `BACKLOG.md`, in the same commit.**
  ADR-013, ADR-014 and ADR-015 each shipped a first draft whose Out of Scope deferred something to
  `BACKLOG.md` with no such entry, and `adr-debt` named all three. Three in a row is not three
  mistakes, it is a missing step: writing the deferral is satisfying and writing the receipt is not,
  so the receipt is what gets dropped. Reviewers caught every one of them, which is the system
  working and also a poor use of a reviewer. **Grep `BACKLOG.md` for the deferral before opening the
  PR.** Noted here because it is a habit rather than a fact about ADR-015.
- **§49 rather than §47 or §48.** ADR-014 held those on an open branch. The reservation worked —
  when both merged, `scripts/contract.sh` needed a textual merge and no renumbering — but the file
  still needed the merge, and the sections had to be reordered so a reader meets 45, 46, 47, 48, 49
  in sequence rather than in the order the branches landed.
- **Both silence cases are tests, and one of them was missing from the table.**
  `TestAnOrdinaryMissingFileGetsNoGlobHint` existed and passed while the Tests table listed three
  tests and the fence counted three, so `Rests-on: both stay quiet when they should` had only half a
  test bound to it. Caught in review; the table and the fence now count four.
