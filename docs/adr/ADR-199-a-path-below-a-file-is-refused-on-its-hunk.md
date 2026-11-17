# ADR-199: A path below something that is not a directory is refused on its hunk

**Status:** Accepted
**Accepted:** 2026-11-15 by Zy — asked *"Do you accept ADR-199, and the ADR-198 amendment …?"* on the draft rewritten after the two challenges, answered *"accep"*.
**Date:** 2026-11-15
**Amended:** 2026-11-15, before acceptance, from an adr-challenger pass and a Codex xhigh challenge (each finding checked against the code): the check runs only when the path cannot be found or opened, and asks about a prefix only after the root boundary has accepted that prefix, where the first draft examined every prefix of every file with a bare `Stat`; earlier refusals keep their place and the record says which; a link to nothing and a link loop are members; a prefix the same plan writes is deferred; ADR-066 is named beside ADR-125 and each is quoted from the record; the table gained the ops it lacked and says staging where it said commit.
**Amended:** 2026-11-15, after acceptance: the reason says what the ancestor is, where the accepted text said `is not a directory` for every kind. Building it showed an overlap the audit had missed: a rename to a destination under a link to nothing already had a refusal of its own (ADR-106, at staging), whose words name the link. Zy was given three options (take the uniform words, keep ADR-106's for a rename destination, or make the new reason say what the ancestor is) and answered *"make it polished, validated"*. Reading that as the third option is this session's interpretation, and it is the one built.
**Amended:** 2026-11-15, from the go-reviewer pass on the first build and a Windows 11 probe of it (decisions narrowed, nothing widened): a link to nothing is no longer a member, because a plan that makes the directory such a link leads to applied before and the first build refused it; a path whose parent is a directory is not examined, because every create and every rename to a new name reached the check and a plan of 2,000 creates took four times as long; a link whose target is itself below a file is a member; the claims about cost and about the boundary test are corrected.
**Amended:** 2026-11-15, from a second go-reviewer pass, on the build with the cost fix (the rule narrowed again): a link is never the obstruction. The build judged a link by what `Stat` said of it, and Go reads Windows' "the path was not found" as "not a directory" too (`syscall/zerrors_windows.go`: `ENOTDIR = ERROR_PATH_NOT_FOUND`), so on Windows a link to a directory two levels deep that another hunk makes would have been refused, a plan that applies today; and a link out of the root whose target lies below an outside file was judged by looking outside. A link that leads to a directory is walked through; one that leads anywhere else is left to what happens today. The two reasons that named a link are withdrawn with it: the wording Zy chose keeps `is a file, not a directory` and `is not a directory`.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-066, ADR-106, ADR-125, ADR-132, ADR-198, ADR-111, issue #644, docs/adr/BACKLOG.md
**Supersedes:** ADR-125, in one clause of Decision 1, "Any other load error — a directory named in a plan, an I/O failure — stays the exit-2 failure it was": a load that fails because a prefix of the path is itself a file, a FIFO or a device (Decision 1; not a link, whatever it leads to) is a failed hunk. ADR-066, in one clause of its Decision, "A destination whose `Lstat` fails for any reason other than 'does not exist' fails validation, with the error as the reason": for that same cause the reason is this record's words; a destination below a link that does not lead to a directory keeps ADR-066's.
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/below199_test.go::TestAPathBelowAFileIsRefusedOnItsHunk`
**Served-path change:** a hunk whose path, or whose rename destination, lies below something that is itself not a directory (`d/f.txt/x` where `d/f.txt` is a file, a FIFO or a device, reached directly or through links to directories) is refused at validation on every platform: exit 1, a FAIL row, nothing written, the reason `PATH cannot be written: ANCESTOR is a file, not a directory`, or `… is not a directory` for a FIFO or a device. Before: on macOS every op on such a path exited 2 with one line and no receipt, and a rename to one exited 1 with the system's `lstat` text; on Windows a rename or a one-level create exited 2 at staging with the system's `mkdirat` text, a two-level create exited 1 with the system's `openat` text, and the other ops said `does not exist`. A path below a link that does not lead to a directory (a link to a file, to nothing, or a loop) is unchanged, and so is one below a Windows reparse point that is not a link (a cloud placeholder file).

## Context

Issue #644. One fact about a plan is reported several ways. Measured 2026-11-15, each plan after its files were read, nothing written in any case; macOS with `mrw` `dev (0d49d6e)`, Windows 11 build 26200 with a binary built from 0d49d6e8:

| Plan | macOS | Windows 11 |
|------|-------|------------|
| rename `d/g.txt` → `d/f.txt/x` | exit 1, `rename dest d/f.txt/x cannot be used: lstat /…/d/f.txt/x: not a directory` | exit 2, `mkdirat d\f.txt: file exists`, at staging |
| `create` `d/f.txt/x` (and `mrw write --create d/f.txt/x`, which compiles to that hunk; macOS measured, Windows by the code) | exit 2, `mrw: d/f.txt/x: open /…: not a directory`, no receipt | exit 2, `mkdirat d\f.txt: file exists`, at staging |
| `create` `d/f.txt/sub/y.txt` | exit 2, the same one line, no receipt | exit 1, `openat d\f.txt: The system cannot find the path specified.` |
| `replace`, `insert-after`, `unlink`, a rename source at `d/f.txt/x` | exit 2, one line, no receipt | not measured; by the code, exit 1 `does not exist` |
| `create` below a link to nothing (`d/dl/x`) | exit 2, `FAIL … mkdirat d/dl: file exists` | not measured |
| `create` below a link loop | exit 2, `open …: too many levels of symbolic links`, no receipt | not measured |
| an edit of `d/g.txt` beside the rename | exit 1, the edit skipped | exit 2, the edit skipped |

Linux was not measured; it takes the macOS route in the code (`ENOTDIR`). Through `mrw_write` on Windows the rename and the create answer `isError`, `error: "d\\g.txt: mkdirat d\\f.txt: file exists"`.

**Why.** A line op loads its file first (`apply.go:669`). On macOS and Linux the open fails with "not a directory", which `causeOf` does not name, so ADR-125's rule returns it bare (`apply.go:696-698`): exit 2, no receipt. On Windows the same open answers "path not found", so the file counts as missing: a `create` is planned and fails at staging, where `mkdirAll` fails (`apply.go:1105`, `2449`) with an error `targetCause` (`apply.go:2312`) does not name; two levels down the error is "not found", which it does name, hence exit 1. A rename destination is cleared only when `Lstat` says it does not exist (`pathop.go:203-227`, ADR-066), and Windows says exactly that.

ADR-132 Decision 2 makes a staging refusal a hunk refusal only for the causes ADR-125 names. This cause is a fact about the plan and the tree, knowable before anything is staged.

**Audit of the class**: *every way a plan's hunk names a path below something that is not a directory.* Read from where `Apply` first touches each path, and probed on macOS:
- A hunk's own path, every op (`create`, `replace`, `insert-after`, `insert-before`, `delete`, `unlink`, a rename's source), below a file, a FIFO or a device, directly or below a link to a directory that holds one: decided here.
- The same below a link that does not lead to a directory (a link to a file, to nothing, a loop, or one whose target is itself below a file): left as it is (Out of Scope, deferred). What such a link leads to is reported differently on each platform, and a link to nothing may lead to a directory another hunk of the plan makes: `create newdir/y.txt` beside `create d/ln/x.txt`, `d/ln` a link to `../newdir`, exits 0 on `dev (0d49d6e)` when the hunks come in that order.
- A rename's destination, alone or beside edits of its source (`apply.go:1436, 1455` call `planPathOp`): decided here, by the same rule.
- A rename's destination under a link to nothing: refused at staging, exit 1, in ADR-106's words (`apply.go:1108`). Unchanged.
- A prefix the same plan unlinks or renames away (`unlink d/f.txt` beside `create d/f.txt/x`): refused by this rule, since the prefix is a file when the plan is judged. Exit 2 today.
- A prefix the same plan writes as a file (`create d/n.txt` beside `create d/n.txt/x`): exit 2 today, `d/n.txt appeared before commit`. The prefix does not exist when the plan is judged, so this rule does not see it; `nestedPath` (`pathop.go:181-192`) covers only rename destinations. Deferred (Out of Scope).
- A `from=` source (`fromcopy.go:173-180`): already a failed hunk, exit 1. Left out (Out of Scope).
- `body=@path` (`internal/plan/plan.go:1032`): refused when the plan is read, exit 2, counted as a parse refusal (`cmd/mrw/main.go:2190`). Left out (Out of Scope).
- The plan makers (`--grep --sub`, `--ast-grep --rewrite`): they walk files that exist. Not members.

## Existing Primitives Audit

- `resolve` (in `apply.go`) is the boundary: it refuses an escape, `.git`, mrw's state and a device name. The whole path has passed it before the check runs, and when the prefixes are judged one by one each passes it again before its entry is read. Before that, the nearest existing ancestor is found by `Lstat` on the names the whole path's resolution walked.
- `refuseFile` (in `apply.go`) fails a file's first hunk and skips its siblings; `fail` in `planPathOp` fails a path op.
- `asPlan` (ADR-198) spells both paths in the reason.

## Decision

1. **A path below something that is itself not a directory is refused on its hunk.** When a hunk's file cannot be opened, or does not exist, and when a rename destination's `Lstat` fails, `Apply` looks for the obstruction: the shortest proper prefix of the path that the boundary accepts and whose own entry (`Lstat`) is neither a directory nor a link. A prefix that is a link is walked through: when it leads to a directory the prefixes below it are judged, and when it leads anywhere else the next one cannot be examined and the search ends with no refusal. On Windows every reparse point Go reports as irregular counts as a link here: a junction, and also a cloud placeholder file, which is therefore left as it is. If there is an obstruction, the hunk fails with `PATH cannot be written: ANCESTOR is a file, not a directory` (or `… is not a directory` for a FIFO or a device), both paths spelled as a plan spells a path (ADR-198); the file's other hunks skip, nothing is written, and `Apply` returns no error.
2. **Refusals that come earlier keep their place.** A path the boundary refuses, two spellings of one file, a `from=` source that fails, and for a rename the source's own load, its read guard and the destination's duplicate and nesting refusals are judged before the obstruction is looked for, and win.
3. **A path whose nearest existing ancestor is a directory is not examined.** A file that opens never reaches the check. A file that does not exist yet (every `create`, every rename to a new name) reaches it and is answered by finding the nearest ancestor that exists: an `Lstat` for the parent and for each missing directory above it. When that ancestor is itself a directory there is no such prefix and nothing more is done. Otherwise, a link included, the prefixes are judged one by one, each through the boundary before its entry is read; no entry is followed before the boundary has been asked about it.
4. **Nothing else changes.** A path below a directory that does not exist yet is made as now. A prefix whose `Lstat` or `Stat` fails for another reason (a permission) is left to the refusals that exist. A prefix that turns into a file after validation is refused at staging as now (ADR-106).

**What would make this decision wrong:** a caller that relies on exit 2 for this plan. None is known; the exit already differs by platform and by depth.

## Alternatives Considered

- **Examine every prefix of every file a plan names** (the first draft). Rejected: it costs a `Stat` per component on plans that apply, and it looked at prefixes with a bare `Stat` on the strength of the whole path having passed the boundary, which is not a judgement of each prefix.
- **Teach `targetCause` that `ENOTDIR` and Windows' `mkdirat … file exists` are the target's.** Rejected: the refusal would still come from staging in the system's words, and the macOS load error would still have no receipt.
- **Fix only Windows' answer in `planPathOp`.** Rejected: it leaves the rows with no receipt.

## Component / Boundary Impact

`internal/apply` (engine; `apply.go` and `pathop.go`, owned here and by issue #644, go-reviewer pass per `lifecycle.md` §6). `read`, `plan`, `seen`, `check`, `state`, `writer`, `rooted`, `mcp` and `cmd` stay byte-identical. No receipt key is added (ADR-111).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a hunk whose path or rename destination is below a non-directory, CLI and `mrw_write` | exit 1, a FAIL row, mrw's words, on every platform | T1 | every caller |
| `scripts/contract.sh` | a new section | T1 | CI Linux |
| `README.md`, `AGENTS.md` | none: neither states what this plan does (`grep -n -i "not a directory" README.md AGENTS.md`, 2026-11-15: the hits are the trailing-separator rule) | — | — |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request, after ADR-198 T2, which edits the same test file.

## Consequences

- **Positive:** one verdict and one exit code for one fact; every op below a file gets a receipt.
- **Negative:** for each file a plan creates and each rename to a new name, an `Lstat` for its parent and for each missing directory above that; on the failure path, and for a file whose nearest existing ancestor is a link, one boundary judgement and one `Lstat` per component. Measured 2026-11-15 on macOS at load 9-14, `main` 58365dc0 against the branch at 77ca0152, median of 11 runs each, alternating: a dry run of 2,000 creates in a six-deep directory 302 against 319 ms; the same as a real write 2,131 against 2,106 ms; 2,000 creates each in its own new two-level directory 1,734 against 1,763 ms; 500 renames in one directory 332 against 343 ms. Two earlier builds cost more and were changed for it: one walked every prefix for every create (about four times as long in the reviewer's in-process benchmark), and one judged the first missing prefix through the boundary (593f12ac: 1,943 against 1,734 ms on the new-directories plan on macOS, and 62.8 against 49.7 s on Windows 11). Windows 11 on 593f12ac, the other three plans: 23.1 against 22.5 s, 55.8 against 55.9 s, 16.1 against 16.1 s; Windows 11 on 77ca0152, mean of 5: the new-directories plan 50.8 against 50.3 s, the dry run 19.4 against 19.1 s, 500 renames 16.3 against 16.1 s, and 2,000 creates into one new six-deep directory 63.3 against 62.6 s. The last head, which reads the nearest ancestor with `Lstat` alone, was timed on macOS twice more at load 4-7 with another job on the machine: the best of 11 for the four plans was 264/266, 1,854/1,860, 1,645/1,638 and 302/303 ms, while the medians moved by up to a third in both directions between the two runs.
- **Neutral:** `scripts/contract.sh` runs on Linux and shows the rule there; Windows is shown by the Go test in the Windows CI job and a probe of the built binary before the merge. The link cases skip where links need a privilege.

## Out of Scope

- A prefix the same plan writes as a file (deferred: docs/adr/BACKLOG.md)
- A path below a link that does not lead to a directory: a link to a file, to nothing, a loop (deferred: docs/adr/BACKLOG.md)
- A path spelled with a leading `/`, which `Apply` joins under the root (permanent: boundary: the check keeps a rooted path as the caller typed it, as ADR-198 does, and whether such a spelling should be accepted at all is another question; exit 2 as before)
- A `from=` source below a non-directory (permanent: boundary: it is already a failed hunk at exit 1, and ADR-167 owns that refusal's words)
- `body=@path` below a non-directory (permanent: boundary: the body file is read while the plan is parsed, and ADR-009's tally counts that refusal as a parse refusal, exit 2)
- A prefix that becomes a file after validation (permanent: boundary: staging reopens every path by name and refuses there, ADR-106)
- The read side (permanent: boundary: a read reports the path UNREADABLE with the system's reason and exits 1 already)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The check refuses a path that works today | Low | High | it runs only for a path that could not be found or opened, and refuses only for a prefix that exists and leads to no directory; T1 holds a path below a real directory, a missing one and a link to a directory to "applied" |
| The check looks at something outside the root | Low | High | it reads entries only, with `Lstat`, and follows none. The whole path has passed `resolve` (`apply.go:585`, `pathop.go:193`), and each prefix passes it again before its entry is read; a link that leads out of the root to a directory is refused there. A link that leads anywhere but a directory ends the search, so nothing is concluded from what lies beyond it. T1's two link-out-of-the-root cases are refused by the whole-path judgement; a third, a path that leaves the root through one link and comes back through another, passes it and is stopped by the per-prefix one, and a mutant that removes that judgement is killed by it |
| An existing test or contract row pins the old exit or words | Med | Low | found 2026-11-15: only `reason198_test.go`'s `rename dest … cannot be used` case, which T1 moves to ADR-066's long-name fixture; `TestAPlanRefusedAfterItParsedIsOneRefusal` uses a directory and is unaffected |

## Rollback

Revert T1: the behaviours of the table return.

## Follow-ups

- [ ] The Windows probe of the built binary before the merge (T1, Stop Condition)
- [ ] A prefix the same plan writes as a file (BACKLOG.md, From ADR-199)
- [ ] A path below a link that does not lead to a directory (BACKLOG.md, From ADR-199)
