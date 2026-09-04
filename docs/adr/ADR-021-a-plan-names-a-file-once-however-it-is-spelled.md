# ADR-021: A plan names a file once, however it is spelled

**Status:** Accepted
**Accepted:** 2026-09-04 by M — asked for a local triage: *"test this mrw tool, try to break it, and if we cant - we call it a day success."* It broke, once, in 45 probes; the standing instruction is that a real defect is a blocker, so it is fixed rather than filed.
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Served-path change:** a plan whose hunks name ONE existing file under TWO spellings — `Same.txt` and `same.txt` on a case-insensitive filesystem, or a file and a symlink to it anywhere — is refused, naming both spellings, instead of applying both and keeping only the last.
**Cross-references:** ADR-001 (owns "whole or not at all" and "every hunk carries a verdict", both of which this defect violated at once), ADR-002 (owns the ledger, whose lookup already answers identity by asking the filesystem — issue #47 — and is the precedent this reuses), ADR-006 (owns root confinement, which already resolves symlinks per path and is untouched)
**Governs:** `internal/apply/**`
**Enforced-by:** `internal/apply/apply_test.go::TestAPlanThatNamesOneFileTwiceIsRefusedWhicheverTheSpelling`
**Invalidates:** none. It narrows nothing an earlier record promised; it closes a gap between two of them.

## Context

**Measured 2026-09-04 against `main` at `838053e`, on APFS (case-insensitive by default), in a break
campaign of 45 probes.** A file `Same.txt` holding `one / two / three` was read under both spellings,
then given this plan:

```
@@ Same.txt 1 replace
X
@@ same.txt 3 replace
Z
```

The receipt said `2 hunk(s), 2 file(s), 0 failed — applied`, with `--json` listing `Same.txt`
written at one SHA and `same.txt` written at another, both hunks `ok`. The file afterwards held
`one / two / Z`: **the first hunk was gone, and nothing said so.** Both spellings staged a copy from
the same bytes, and the second rename replaced the first.

That is the failure this tool exists to prevent — *a write that changes nothing is not visible* —
reached through the promise that was supposed to make it impossible. ADR-001 says a plan applies whole
or not at all and every hunk carries its own verdict; here one hunk's verdict was `ok` and its effect
was nil.

**The ledger already knew the two spellings were one file.** Issue #47 taught `apply` to resolve a
ledger miss with `os.SameFile` — read `Same.txt`, and a write to `same.txt` is licensed, which the
same campaign confirmed. But the grouping of hunks into files, one step earlier, still keys on the
cleaned path STRING. Identity was answered on one axis and not the other, and a plan that fell into
the gap was applied with a receipt that could not be told from a correct one.

**The same shape reaches Linux through symlinks.** `link.txt -> real.txt` inside the root: a plan
naming both is two path strings, one inode, and the same last-rename-wins. So this is not a
case-folding bug with a macOS-only fix; it is a missing identity check with a portable one.

## Existing Primitives Audit

- **`sameFileEntry` (issue #47, in `internal/apply`):** already walks a set of paths asking the
  filesystem `os.SameFile` against a stat. **Reused in shape**: the grouping loop needs the same
  question asked against the files the plan has already resolved, and it is asked the same way —
  the filesystem's own answer, no belief about case encoded.
- **`resolve` (ADR-006):** every hunk path is already resolved to an absolute, symlink-followed,
  in-root path before anything is staged. **The identity check hangs off that resolution** and adds
  no new path handling.
- **`os.SameFile`:** taken over folding case or comparing resolved strings. `filepath.EvalSymlinks`
  does not canonicalise case, so two strings can differ and name one file; a stat comparison cannot.
- **The per-hunk refusal path (ADR-001 rule 3):** a file that fails validation reports the failure
  through its first hunk and aborts the run with nothing written. **Reused unchanged** — this is one
  more reason a file can fail, not a new way to fail.

## Decision

**1. Two hunks that name one existing file are the same file's hunks, however they spell it.** The
grouping loop resolves each new path and asks `os.SameFile` against every file already grouped. A
match is refused.

**2. A refusal, not a merge.** The second spelling could be folded into the first's hunk list and
applied in one pass. It is not, because the plan then says two things — two files, two receipts —
and the receipt would report one. A plan that names a file twice is a plan the author did not mean
to write; the refusal names both spellings so the fix is one edit to the plan.

**3. The refusal is reported through the second spelling's first hunk**, in the receipt, and aborts
the run with nothing written — ADR-001 rule 3, unchanged. The reason names the earlier spelling: *"names
the same file as `Same.txt` (plan line 1); one file, one spelling per plan"*.

**4. Only existing files are checked.** Two `create` ops for `New.txt` and `new.txt` have no inode
to compare until one is written. That case is named in Out of Scope with the evidence that would
promote it, not silently included in a claim the check cannot keep.

**5. Nothing folds case.** On ext4, `a.txt` and `A.txt` are two files and stay two files. The check
asks the filesystem and believes the answer.

**Go/no-go, checked during execution:**

- **`internal/apply` changes; every other engine directory stays byte-identical** against a
  merge-base diff — `internal/read`, `internal/plan`, `internal/seen`, `internal/check`,
  `internal/state`, and `internal/mcp`. If the fix needs any of them, it has grown past a grouping
  check and must stop.
- **No new dependency**; `go.mod` still declares exactly one requirement.
- **`gofmt -l .` empty and `go vet ./...` clean**, in the fence.

## Alternatives Considered

- **Fold case on filesystems believed to be case-insensitive.** Rejected for the reason #47 rejected
  it: a belief about the filesystem is wrong somewhere, and `os.SameFile` is the filesystem's own
  answer. It also misses symlink aliases entirely.
- **Key the grouping on the resolved absolute path.** Handles symlinks; misses case, because
  `EvalSymlinks` returns the spelling it was given. Half a fix that looks whole.
- **Merge the second spelling's hunks into the first.** Applies cleanly and reports one file. Rejected
  under Decision 2: it rewrites what the plan said, and the receipt would disagree with the plan.
- **Detect it after staging, before rename**, by comparing staged targets. Later, more code, and it
  still needs `SameFile`; nothing gained over asking at grouping time.
- **Leave it and document it.** The campaign's own conclusion, four times over, is that mrw breaks at
  input shape and never at volume; this is an input shape an LLM produces by accident
  (`README.md`/`Readme.md`). A documented silent loss is still a silent loss.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/apply` | Now also: which plan paths are one file | Yes |
| `internal/rooted` | Unchanged — what a path may reach | Untouched |
| `internal/seen` | Unchanged — already identity-aware on lookup | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| A plan naming one existing file under two spellings is refused, nothing written, both spellings named | Public contract, BREAKING for a plan that relied on last-wins | Any caller; the CLI and MCP share the engine |
| Two `create` ops that would collide on a case-insensitive filesystem | Unchanged, named in Out of Scope | — |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the identity check and its refusal text | T1 | — | Yes, deliberately — see Consequences |

## Implementation

One task. The check in the grouping loop, its test, and the contract row that drives the built binary
with the plan from the Context.

## Consequences

- **Positive:** a receipt that says `applied` for two hunks means two hunks applied. The promise
  ADR-001 makes is kept on the one axis it was not.
- **Positive:** symlink aliases inside the root get the same protection for free.
- **Negative, and it is a real break:** a plan that named a file twice and worked by accident — the
  second spelling's edits winning — now fails. That plan was already losing its first edit silently.
- **Negative:** one extra `os.Stat` per distinct path in a plan. A 500-file plan pays 500 stats it
  already paid to read the files.
- **Neutral:** the CLI and MCP surfaces change together, because the engine is shared.

## Out of Scope

- Two `create` ops that would collide on a case-insensitive filesystem (deferred: docs/adr/BACKLOG.md — the create-collision entry; the evidence to promote it is one such plan seen in the wild)
- Folding case anywhere (permanent: boundary: Decision 5 — the filesystem answers, the tool does not guess)
- Any change to how the ledger records or looks up a file (permanent: boundary: ADR-002 owns it and it already answers identity)
- A concurrent writer racing this run (permanent: boundary: the incident of 2026-09-02 and ADR-002 leave locking out of scope; this record is about one plan, one run)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The check is written and the CLI never exercises it | Low | **High** | §56 drives the BUILT binary with the Context's plan and requires exit 1, nothing written, both spellings in the refusal |
| The test passes on a case-sensitive filesystem without reaching the branch | High if unhandled | **High** | The test uses a symlink alias, which is one inode on every filesystem, and additionally the two-spelling case guarded by a runtime probe — the symlink half cannot be skipped |
| A legitimate plan naming two genuinely different files is refused | Low | Med | `os.SameFile` is false for two inodes; the test asserts two real files under near-identical names still apply |
| Performance on very large plans | Low | Low | One stat per distinct path; the scale campaign's 500-file plan would pay 500 stats |

## Rollback

Revert the commit. The check is one block in the grouping loop; removing it restores last-rename-wins
exactly, which is what the record exists to stop.

## Follow-ups

- [ ] If a plan with two colliding `create` ops is seen in the wild, promote the BACKLOG entry to a
      record: the check would need to fold case or write-then-stat, and either needs deciding.
