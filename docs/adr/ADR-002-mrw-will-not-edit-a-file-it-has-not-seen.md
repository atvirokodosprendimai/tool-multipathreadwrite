# ADR-002: mrw will not edit a file it has not seen

**Status:** Accepted
**Date:** 2026-08-31
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (defines the addresses this record makes trustworthy — ADR-001 assumes the caller's line numbers describe the file on disk, and this is the check that they do)
**Governs:** `internal/seen/**`, `internal/apply/apply.go`
**Enforced-by:** `internal/apply/apply_test.go::TestAFileChangedBehindMrwsBackCannotBeEdited`
**Invalidates:** none — checked (grepped ADR-001 for `sha`, `ledger`, `stale`: ADR-001 names `sha=` as an OPTIONAL per-hunk guard and this record adds an ambient default beside it; the optional guard is unchanged and still stronger where a caller wants it pinned in the plan)
**Served-path change:** `mrw write` now refuses, with a named reason and both SHAs, when the target file is one mrw has never read or one that changed since it last saw it — where previously any plan applied to whatever was on disk.

## Context

**This is a retrofit**, shipped in `829aae7` and released as v0.0.2. As with
ADR-001, the TDD-red step is historical and the substitute proof is a mutation.

ADR-001 gives a caller the power to change N sites across M files from one
document. It does not, on its own, establish that the caller's line numbers
describe the file that is actually there. A range address is only meaningful in
the version of the file those numbers were counted in — so an edit written
against a stale picture puts the right text in the wrong place, and nothing
about that failure is visible afterwards. That is the same asymmetry ADR-001 is
built around, one level up: it is not the hunk that fails silently, it is the
whole plan's premise.

The harness's own `Write` tool already has this property — it refuses to
overwrite a file you have not `Read`. A *range* edit needs it more, not less.

The guard caught three real mistakes during its own construction, which is the
evidence that it is not theoretical: a README edit written after only grepping
the file, an `apply.go` restored with `cp` during mutation testing, and a plan
whose line numbers predated an earlier hunk.

## Existing Primitives Audit

- **`sha=` per-hunk guard (ADR-001):** already solves this *when the author
  remembers to write it*. **Reused, not replaced** — it stays, it is stronger
  where a caller wants the pin visible in the plan, and it satisfies this
  record's check on its own. What it cannot do is protect a caller who did not
  think to use it, which is every caller who did not know the file had moved.
- **The harness `Write` tool's read-before-write rule:** the model being copied.
  Reshaped: mrw records a hash rather than a tool-call history, so the check
  survives across processes.
- **git's index / `git status`:** would detect a changed file, but only for
  tracked files in a clean repo, and it cannot see a change mrw itself made.
  Rejected as the mechanism.

## Decision

mrw keeps a ledger at `.mrw/seen` mapping path → SHA-256, and refuses to edit an
existing file unless it holds a matching entry.

The ledger is written on **read and on write**. The write half is what makes it
usable rather than merely safe: a chain of edits needs no re-read between steps,
because mrw already knows what it just produced, while a change made by any
other route leaves the recorded hash and the real one disagreeing.

| Situation | Result |
|---|---|
| edit a file never read | refused — `mrw read <path>` first |
| read, then edit | applies |
| edit again straight after | applies — mrw knows what it just wrote |
| anything else changed it, then edit | **refused**, naming both SHAs |
| `--force` | applies |
| `create` | applies — no existing content to be stale about |

`mrw read --stat` records too, so re-authorising a file costs one line of output
rather than the whole file: the hash is the staleness proof, and content is not
needed to establish it.

**Retired 2026-09-02 by ADR-005, which narrowed the ledger from files to LINES.**
A `--stat` prints no content, so it now observes an empty span set and licenses
no edit at all: *"`mrw read --stat` is now purely informational"* (ADR-005
Consequences). Verified against the binary — after a stat, a write is refused
with *"2 of f.txt has not been read: mrw served no lines"*.

The sentence above is preserved because it is the reasoning this record was
decided on, and because a hash-is-enough argument is exactly the one a future
session would re-derive. What replaced it is not "re-read the whole file": a
RANGED re-read of only the lines the edit addresses licenses those lines and
nothing else, so the cheap remedy still exists — it is one call, and it is now
`mrw read <path>:<range>` rather than `--stat`.

**What would falsify this:** if the refusal fired mostly on false positives —
files that had changed in ways that did not invalidate the caller's addresses —
the guard would cost more than it saves. It cannot distinguish those cases and
does not try; a hash is deliberately all-or-nothing. As of 2026-08-31 every
refusal observed in development was a true positive, on a sample of three, which
is too small to be evidence of a rate and is recorded as an anecdote rather than
a measurement.

## Alternatives Considered

- **Require `sha=` on every hunk (no ledger, no state):** stateless and
  explicit. Rejected because it costs output tokens on every hunk — the
  expensive direction — and because it protects only the caller who already
  suspected a problem.
- **Compare modification time instead of content:** cheaper. Rejected because
  mtime changes when content does not (a `touch`, a checkout) and can fail to
  change when content does; a false refusal trains people to use `--force`.
- **Track reads in memory for the process lifetime:** no file, no cleanup.
  Rejected because every `mrw` invocation is a new process — the read and the
  write are different processes by construction.
- **Refuse only when the file is git-dirty:** rejected because it cannot see an
  untracked file, needs a repository, and says nothing about a change mrw made.
- **Do nothing; rely on `anchor=`:** rejected because an anchor pins the first
  line of one range, so a file that changed elsewhere still applies.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/seen` | The ledger: load, merge-record, forget, hash. Knows nothing about hunks. | Yes — changes only when the ledger format or policy changes |
| `internal/apply` | Gains an `Options.Seen` input and one check. Does not know where the ledger lives. | Yes — the check is part of write semantics |
| `cmd/mrw` | Loads the ledger, passes it in, records after read and after write | Yes |

The ledger is not `internal/apply`'s concern: `Apply` takes a `map[string]string`
and a nil map disables the check entirely, which is what the engine's own tests
use. That keeps the policy at the CLI and the mechanism in the engine.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `.mrw/seen` file format (`<sha>  <path>` per line, sorted) | new persistent state | `internal/seen` | `mrw read`, `mrw write` |
| `apply.Options.Seen` / `apply.Options.Force` | new internal contract | `internal/apply` | `cmd/mrw` |
| `mrw write --force` flag | new public contract | `cmd/mrw` | callers with a deliberate blind write |
| `read.Run` return signature (now returns observed SHAs) | changed internal contract | `internal/read` | `cmd/mrw` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `seen.Ledger`, `seen.Load`, `seen.Record` (T1) | T1 | T2 | No — T2 consumes the map, it does not define it |

## Implementation

See `ADR-002-mrw-will-not-edit-a-file-it-has-not-seen/tasks/README.md`. Two
tasks: the ledger, and the refusal that uses it.

## Consequences

- **Positive:** an edit written against a stale picture is refused rather than
  applied, and the refusal names both hashes so the caller can see what moved.
- **Positive:** it catches changes from any route — another agent, `git
  checkout`, `cp`, an editor — not only from mrw's own history.
- **Negative:** mrw now has per-developer persistent state, which is one more
  thing to reason about and to gitignore. A deleted `.mrw/seen` means every file
  must be re-read; that is a safe direction to fail in, but it is friction.
- **Negative:** `--force` exists, so the guard is advisory to anyone who reaches
  for it. That is deliberate; a guard with no escape hatch gets worked around in
  worse ways.
- **Neutral:** `mrw read --stat` is now load-bearing for authorisation, not only
  for cheap inspection. **Retired 2026-09-02 by ADR-005:** a stat prints no
  content, observes an empty span set, and is purely informational again. The
  consequence reversed rather than lapsed — the thing this bullet called
  load-bearing is now the one read that carries no authority at all.

## Out of Scope

- Locking, or any protection against two processes writing concurrently
  (permanent: mrw is a one-shot command, not a daemon; the ledger detects a
  change after the fact, it does not prevent one)
- Content-aware staleness ("this change did not affect your lines") (permanent:
  a hash is all-or-nothing by design, and the alternative is a merge algorithm)
- A `mrw forget <path>` subcommand to drop ledger entries — `seen.Forget` exists
  and is unwired (deferred: docs/adr/BACKLOG.md)
- Pruning ledger entries for deleted files (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Callers reflexively add `--force` after the first refusal, and the guard stops meaning anything | Med | High | The refusal message names the cheap remedy before it names `--force`; the CLI's own `--force` help calls it *"the escape hatch, not the habit"* (`cmd/mrw/main.go`). **Corrected 2026-09-02, twice:** the control holds, but both of its citations moved. The remedy the message names is a re-read, not `--stat`, which ADR-005 retired. And the phrase is the flag's own help text — visible in `mrw write --help` to the person who has just hit the refusal, with no agentsmemory registration required — so it is now cited there rather than to the skill. The score is unchanged because what mitigates the risk is the ORDER (remedy before escape hatch), which is intact, and it is now cited to something a reader can check from this repository |
| `.mrw/seen` is committed by accident and one developer's state becomes everyone's | Low | Med | `/.mrw/` is gitignored; the entry is anchored so it cannot be swallowed by a path component of the same name |
| The ledger grows without bound as files are read | Low | Low | One short line per path; pruning deferred to BACKLOG |
| A stale ledger entry after an external revert makes an ordinary edit refuse | Med | Low | Correct behaviour, and the remedy is one call. **Corrected 2026-09-02:** no longer `--stat`, which licenses nothing since ADR-005 — it is one RANGED read, `mrw read <path>:<range>`, which re-observes exactly the lines the edit addresses. Still one call, so the score is unchanged |

## Rollback

This ADR introduces persistent state, so rollback has two halves:

1. **Behaviour:** pass a nil ledger from `cmd/mrw` (or revert `829aae7`). The
   check disappears; nothing else in the write path depends on it.
2. **State:** `rm -rf .mrw/seen`. The file is per-developer, gitignored, and
   holds no information that is not recomputable by reading the files again.
   Nothing else reads it.

There is no migration to undo and no data loss — the ledger is a cache of
observations, not a source of truth.

## Follow-ups

- [ ] Decide whether `mrw forget <path>` is worth wiring, or whether `seen.Forget` should be deleted as dead code (see BACKLOG.md)
