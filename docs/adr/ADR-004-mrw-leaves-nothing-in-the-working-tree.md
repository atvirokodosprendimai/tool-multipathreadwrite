# ADR-004: mrw leaves nothing in the working tree

**Status:** Accepted
**Date:** 2026-08-31
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002 (owns the read-before-modify guarantee, which is unchanged — this record moves only where its ledger lives), ADR-001 (unaffected: a plan and its addresses touch no tool state)
**Governs:** `internal/state/**`, `internal/seen/**`, `internal/iter/**`
**Enforced-by:** `internal/state/state_test.go::TestNoStateIsWrittenUnderTheRepoRoot`
**Invalidates:** ADR-002 — four clauses naming `.mrw/seen` as the storage location: the sentence in its Decision, the `.mrw/seen file format` row of its Wiring table, the Risk row whose mitigation reads "`/.mrw/` is gitignored", and step 2 of its Rollback. Its DECISION is untouched: mrw still refuses to edit a file it has not seen, the ledger is still path → SHA-256 recorded on read and on write, and `--force` and the `create` exemption are unchanged.
**Served-path change:** Running mrw in a repository no longer creates `.mrw/` in it; the ledger and working set move to a per-checkout directory under `$XDG_STATE_HOME`, and `mrw seen` prints where.

## Context

Reported 2026-08-31 by a session using mrw on another project: `.mrw/seen` is
created silently on the first `mrw read` — including `--stat` — and nothing in
the repository ignores it. It was committed by accident under `git add -A` and
only noticed because someone asked about the tool.

The report's sharpest line is the one to act on: *"the skill body documents the
guard's semantics carefully and never mentions that it leaves a file behind."*

⚠ **ADR-002's mitigation for exactly this risk was wrong in scope, and that is
the more interesting failure.** It reads "`/.mrw/` is gitignored", which was
true of THIS repository because the same commit that added the ledger added the
ignore line. It was never true of any other repository, and the risk row reads
as though the class were handled. A mitigation that protects the repo you are
standing in, stated as though it protects the mechanism, is worse than an
unmitigated risk — it stops anyone looking again.

This record also covers `.mrw/iteration`, which has the identical defect and was
not reported only because nobody had committed one yet.

## Existing Primitives Audit

- **`.mrw/` in the working tree (ADR-002, ADR's working-set decision):** the
  thing being replaced. Its one real advantage is that a human can see and diff
  it; that is bought back by `mrw seen`.
- **The XDG Base Directory specification's `XDG_STATE_HOME`:** exists for
  precisely this class — "state data that should persist between restarts but
  is not important enough for the data directory". **Adopted** rather than
  inventing a layout.
- **`os.UserCacheDir` / `os.UserConfigDir`:** Go's stdlib helpers. **Rejected**:
  a cache may be deleted at any time and this is not config. There is no
  `os.UserStateDir`, so the path is derived here.
- **`.git/` as a host for tool state:** **Rejected**, see Alternatives.

## Decision

Per-checkout tool state lives outside the working tree, at:

    $XDG_STATE_HOME/mrw/<sha256(abs root)[:16]>/{seen,iteration,root}

falling back to `~/.local/state` when `XDG_STATE_HOME` is unset. The `root` file
records the absolute path the directory belongs to, so an orphaned entry is
identifiable rather than an anonymous hash.

`mrw seen` prints the resolved directory and the ledger's contents, which is how
the inspectability of an in-tree file is preserved.

**Migration is one-time, additive, and never deletes.** When a legacy
`.mrw/seen` or `.mrw/iteration` exists and the new location has none, its
contents are copied across and a notice tells the caller they may now remove
`.mrw/`. Deleting a user's files to tidy up is not this tool's business, and a
tool that removes something committed to a repository is a worse bug than the
one being fixed.

**What would falsify this:** if state outside the tree turned out to be lost
often enough that callers were re-reading files constantly, the in-tree version
would be better despite the gitignore cost. The failure direction is safe — a
missing ledger means "read it again", never a wrong write — so the cost is
friction, not correctness, and it is measurable by how often a refusal names an
unseen file that was read earlier in the same session. Nothing measures that
today, and this record does not claim it has been measured.

## Alternatives Considered

- **`.git/mrw/`, as first proposed:** rejected on two checks rather than on
  taste. In a linked worktree `.git` is a FILE, not a directory (verified
  2026-08-31: `git worktree add` produces `.git` containing `gitdir: …`), so
  resolving it correctly needs `git rev-parse --git-dir` — a subprocess on
  every invocation, and a git dependency in a tool that has none. And mrw does
  not require git at all: `mrw -C /any/dir` works today, so a no-git fallback
  would be needed anyway, leaving two locations and a rule about which applies.
  It is also git's namespace, not ours.
- **Keep `.mrw/` but write a self-ignoring `.mrw/.gitignore` containing `*`:**
  a genuinely good answer — it fixes the reported bug completely, needs no
  per-repo setup, keeps the state visible, and adds no git dependency. Rejected
  because the objection was to putting per-checkout tool state next to source at
  all, not only to the ignore gap; ranked second, and the one to revert to if
  the state directory proves annoying.
- **A single global ledger keyed by absolute path:** rejected — one file that
  every checkout writes is a concurrency problem the current design does not
  have.
- **Do nothing; document the gitignore line:** rejected. It puts the burden on
  every user of every repository, and the failure is silent until something is
  committed.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/state` | Resolving the per-checkout state directory, and the one-time migration. Knows nothing about ledgers or working sets. | Yes — changes only when the location policy changes |
| `internal/seen` | The ledger's content and merge semantics; asks `state` where to put it | Yes |
| `internal/iter` | The working set's content; asks `state` where to put it | Yes |

Both consumers keep their existing shape — `Load(root)` / `Record(root, …)` —
so the location is a detail neither of them holds.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `$XDG_STATE_HOME/mrw/<key>/seen` | replaces `.mrw/seen` | `internal/state` | `internal/seen` |
| `$XDG_STATE_HOME/mrw/<key>/iteration` | replaces `.mrw/iteration` | `internal/state` | `internal/iter` |
| `$XDG_STATE_HOME/mrw/<key>/root` | new — names the checkout a state dir belongs to | `internal/state` | `mrw seen`, a human |
| `mrw seen` | new public contract | `cmd/mrw` | callers inspecting the ledger |
| One-time migration notice on stderr | new observable behaviour | `cmd/mrw` | callers with a legacy `.mrw/` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `state.Dir` (T1), `state.Migrate` (T1) | T1 | T2 | No — T2 consumes the resolved path |

## Implementation

See `ADR-004-mrw-leaves-nothing-in-the-working-tree/tasks/README.md`. Two tasks:
the state directory, then moving both consumers onto it with migration and
`mrw seen`.

## Consequences

- **Positive:** mrw can be run in any repository without leaving a trace, and
  without a per-repo `.gitignore` line nothing told you to add.
- **Positive:** the guarantee is testable as a property — "no file appears under
  the repo root that the plan did not write" — rather than as a list of paths to
  remember.
- **Negative:** state is now invisible unless you ask. `mrw seen` buys that back
  but you have to know it exists.
- **Negative:** moving or renaming a checkout orphans its state directory, and
  nothing prunes them. Each is a few hundred bytes and carries a `root` file
  naming what it belonged to, so a human can clean up; no automatic reaper.
- **Neutral:** repositories that already gitignored `/.mrw/` keep a harmless
  line, and one containing a legacy ledger keeps it until removed by hand.

## Out of Scope

- Pruning orphaned state directories (deferred: docs/adr/BACKLOG.md)
- Deleting a legacy `.mrw/` on the caller's behalf (permanent: a tool that
  removes files it did not create, one of which may be committed, is a worse
  bug than the one being fixed)
- Sharing a working set between machines or checkouts (permanent: per-checkout
  is the point; a shared set would need a merge policy)
- Windows `%LOCALAPPDATA%` conventions (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller cannot find their state and thinks it was lost | Med | Low | `mrw seen` prints the resolved directory first, before the contents |
| Migration copies a stale legacy ledger over a fresher one | Low | Med | It copies only when the destination has no such file, so a live state dir is never overwritten |
| Two checkouts of one repo at the same absolute path in sequence (a delete and re-clone) inherit stale entries | Low | Low | Every entry is a hash compared against the file on disk; a stale entry refuses and asks for a re-read, which is the safe direction |
| Orphaned directories accumulate | High | Low | Small, each names its `root`; pruning deferred |

## Rollback

Two halves, and the second is why this record exists.

1. **Behaviour:** revert this ADR's commits. `internal/seen` and `internal/iter`
   return to `.mrw/` paths relative to root.
2. **State:** the state directories under `$XDG_STATE_HOME/mrw/` become
   orphaned; `rm -rf "${XDG_STATE_HOME:-$HOME/.local/state}/mrw"` removes them.
   Nothing is lost that is not recomputable by reading the files again — the
   ledger is a cache of observations, and a working set is a convenience.

Migration is additive and non-destructive, so a legacy in-tree `.mrw/` is still
present after the move unless a human deleted it, and reverting picks it back up
where it was left.

## Follow-ups

- [ ] Decide whether orphaned state directories are ever worth pruning (see BACKLOG.md)
