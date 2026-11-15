# ADR-199: A path below something that is not a directory is refused on its hunk

**Status:** Accepted
**Accepted:** 2026-11-15 by Zy — asked *"Do you accept ADR-199, and the ADR-198 amendment …?"* on the draft rewritten after the two challenges, answered *"accep"*.
**Date:** 2026-11-15
**Amended:** 2026-11-15, before acceptance, from an adr-challenger pass and a Codex xhigh challenge (each finding checked against the code): the check runs only when the path cannot be found or opened, and asks about a prefix only after the root boundary has accepted that prefix, where the first draft examined every prefix of every file with a bare `Stat`; earlier refusals keep their place and the record says which; a link to nothing and a link loop are members; a prefix the same plan writes is deferred; ADR-066 is named beside ADR-125 and each is quoted from the record; the table gained the ops it lacked and says staging where it said commit.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-066, ADR-106, ADR-125, ADR-132, ADR-198, ADR-111, issue #644, docs/adr/BACKLOG.md
**Supersedes:** ADR-125, in one clause of Decision 1, "Any other load error — a directory named in a plan, an I/O failure — stays the exit-2 failure it was": a load that fails because a prefix of the path is not a directory is a failed hunk. ADR-066, in one clause of its Decision, "A destination whose `Lstat` fails for any reason other than 'does not exist' fails validation, with the error as the reason": for this cause the reason is this record's words.
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/below199_test.go::TestAPathBelowAFileIsRefusedOnItsHunk`
**Served-path change:** a hunk whose path, or whose rename destination, lies below something that exists and is not a directory (`d/f.txt/x` where `d/f.txt` is a file, a FIFO, a link to a file, a link to nothing or a link loop) is refused at validation on every platform: exit 1, a FAIL row, nothing written, the reason `PATH cannot be written: ANCESTOR is not a directory`. Before: on macOS every op on such a path exited 2 with one line and no receipt, and a rename to one exited 1 with the system's `lstat` text; on Windows a rename or a one-level create exited 2 at staging with the system's `mkdirat` text, and a two-level create exited 1 with the system's `openat` text.

## Context

Issue #644. One fact about a plan is reported several ways. Measured 2026-11-15, each plan after its files were read, nothing written in any case; macOS with `mrw` `dev (0d49d6e)`, Windows 11 build 26200 with a binary built from 0d49d6e8:

| Plan | macOS | Windows 11 |
|------|-------|------------|
| rename `d/g.txt` → `d/f.txt/x` | exit 1, `rename dest d/f.txt/x cannot be used: lstat /…/d/f.txt/x: not a directory` | exit 2, `mkdirat d\f.txt: file exists`, at staging |
| `create` `d/f.txt/x` | exit 2, `mrw: d/f.txt/x: open /…: not a directory`, no receipt | exit 2, `mkdirat d\f.txt: file exists`, at staging |
| `create` `d/f.txt/sub/y.txt` | exit 2, the same one line, no receipt | exit 1, `openat d\f.txt: The system cannot find the path specified.` |
| `replace`, `insert-after`, `unlink`, a rename source at `d/f.txt/x`; `mrw write --create d/f.txt/x` | exit 2, one line, no receipt | not measured; by the code, exit 1 `does not exist` |
| `create` below a link to nothing (`d/dl/x`) | exit 2, `FAIL … mkdirat d/dl: file exists` | not measured |
| `create` below a link loop | exit 2, `open …: too many levels of symbolic links`, no receipt | not measured |
| an edit of `d/g.txt` beside the rename | exit 1, the edit skipped | exit 2, the edit skipped |

Linux was not measured; it takes the macOS route in the code (`ENOTDIR`). Through `mrw_write` on Windows the rename and the create answer `isError`, `error: "d\\g.txt: mkdirat d\\f.txt: file exists"`.

**Why.** A line op loads its file first (`apply.go:669`). On macOS and Linux the open fails with "not a directory", which `causeOf` does not name, so ADR-125's rule returns it bare (`apply.go:696-698`): exit 2, no receipt. On Windows the same open answers "path not found", so the file counts as missing: a `create` is planned and fails at staging, where `mkdirAll` fails (`apply.go:1105`, `2449`) with an error `targetCause` (`apply.go:2312`) does not name; two levels down the error is "not found", which it does name, hence exit 1. A rename destination is cleared only when `Lstat` says it does not exist (`pathop.go:203-227`, ADR-066), and Windows says exactly that.

ADR-132 Decision 2 makes a staging refusal a hunk refusal only for the causes ADR-125 names. This cause is a fact about the plan and the tree, knowable before anything is staged.

**Audit of the class**: *every way a plan's hunk names a path below something that is not a directory.* Read from where `Apply` first touches each path, and probed on macOS:
- A hunk's own path, every op (`create`, `replace`, `insert-after`, `insert-before`, `delete`, `unlink`, a rename's source), below a file, a FIFO, a link to a file: decided here.
- The same below a link to nothing or a link loop: decided here. The prefix exists as an entry and leads to no directory.
- A rename's destination, alone or beside edits of its source (`apply.go:1436, 1455` call `planPathOp`): decided here.
- A prefix the same plan unlinks or renames away (`unlink d/f.txt` beside `create d/f.txt/x`): refused by this rule, since the prefix is a file when the plan is judged. Exit 2 today.
- A prefix the same plan writes as a file (`create d/n.txt` beside `create d/n.txt/x`): exit 2 today, `d/n.txt appeared before commit`. The prefix does not exist when the plan is judged, so this rule does not see it; `nestedPath` (`pathop.go:181-192`) covers only rename destinations. Deferred (Out of Scope).
- A `from=` source (`fromcopy.go:173-180`): already a failed hunk, exit 1. Left out (Out of Scope).
- `body=@path` (`internal/plan/plan.go:1032`): refused when the plan is read, exit 2, counted as a parse refusal (`cmd/mrw/main.go:2190`). Left out (Out of Scope).
- The plan makers (`--grep --sub`, `--ast-grep --rewrite`): they walk files that exist. Not members.

## Existing Primitives Audit

- `resolve` (`apply.go:2742`) is the boundary: it refuses an escape, `.git`, mrw's state and a device name. The check asks it about each prefix before looking at that prefix, so nothing is examined that a write to it would be refused for.
- `refuseFile` (`apply.go:2951`) fails a file's first hunk and skips its siblings; `fail` in `planPathOp` fails a path op.
- `asPlan` (ADR-198) spells both paths in the reason.

## Decision

1. **A path below something that is not a directory is refused on its hunk.** When a hunk's file cannot be opened, or does not exist, and when a rename destination's `Lstat` fails, `Apply` looks for the obstruction: the shortest proper prefix of the path that the boundary accepts, that exists as an entry (`Lstat`), and that does not lead to a directory (`Stat` reports a non-directory, nothing there, or a link loop). If there is one, the hunk fails with `PATH cannot be written: ANCESTOR is not a directory`, both spelled as a plan spells a path (ADR-198); the file's other hunks skip, nothing is written, and `Apply` returns no error.
2. **Refusals that come earlier keep their place.** A path the boundary refuses, two spellings of one file, a `from=` source that fails, and for a rename the source's own load, its read guard and the destination's duplicate and nesting refusals are judged before the obstruction is looked for, and win.
3. **A path that can be found is not examined.** A file that opens has no such prefix, so a plan that applies today pays nothing.
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
- **Negative:** on the failure path only, one boundary judgement, one `Lstat` and one `Stat` per component.
- **Neutral:** `scripts/contract.sh` runs on Linux and shows the rule there; Windows is shown by the Go test in the Windows CI job and a probe of the built binary before the merge. The link cases skip where links need a privilege.

## Out of Scope

- A prefix the same plan writes as a file (deferred: docs/adr/BACKLOG.md)
- A `from=` source below a non-directory (permanent: boundary: it is already a failed hunk at exit 1, and ADR-167 owns that refusal's words)
- `body=@path` below a non-directory (permanent: boundary: the body file is read while the plan is parsed, and ADR-009's tally counts that refusal as a parse refusal, exit 2)
- A prefix that becomes a file after validation (permanent: boundary: staging reopens every path by name and refuses there, ADR-106)
- The read side (permanent: boundary: a read reports the path UNREADABLE with the system's reason and exits 1 already)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The check refuses a path that works today | Low | High | it runs only for a path that could not be found or opened, and refuses only for a prefix that exists and leads to no directory; T1 holds a path below a real directory, a missing one and a link to a directory to "applied" |
| The check looks at something outside the root | Low | High | each prefix passes `resolve` before it is looked at; T1 has a prefix that is a link out of the root, refused by the boundary's own words |
| An existing test or contract row pins the old exit or words | Med | Low | found 2026-11-15: only `reason198_test.go`'s `rename dest … cannot be used` case, which T1 moves to ADR-066's long-name fixture; `TestAPlanRefusedAfterItParsedIsOneRefusal` uses a directory and is unaffected |

## Rollback

Revert T1: the behaviours of the table return.

## Follow-ups

- [ ] The Windows probe of the built binary before the merge (T1, Stop Condition)
- [ ] A prefix the same plan writes as a file (BACKLOG.md, From ADR-199)
