# ADR-034: State that names a checkout nobody has is removable, and only when asked

**Status:** Accepted
**Date:** 2026-09-07
**Owner:** M
**Accepted:** M, 2026-09-07, *"prne, fix properly, no jokes."* — choosing a tool-level prune over the script-level workaround already applied to `scripts/measure.sh`, after the accumulation was measured on M's machine.
**Spec:** None — no spec stage
**Cross-references:** `docs/adr/ADR-004-mrw-leaves-nothing-in-the-working-tree.md`, `docs/adr/ADR-008-a-delete-says-what-it-removed.md`, `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`
**Governs:** `internal/state/*.go`, `cmd/mrw/main.go`, `scripts/contract.sh`
**Enforced-by:** `internal/state/prune_test.go::TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned`
**Invalidates:** none — checked
**Served-path change:** `mrw seen` prints one line naming how many state directories the base holds, and `mrw seen --prune` is a new surface that DELETES directories under `$XDG_STATE_HOME/mrw/` — the first thing mrw removes that it was not asked to write.

## Context

ADR-004 moved mrw's per-checkout state out of the working tree and into
`$XDG_STATE_HOME/mrw/<key>/`, keyed by a truncated sha256 of the resolved absolute root. That
record fixed the bug it was written for — a ledger committed by accident under `git add -A` — and
created a second one it did not consider: **nothing ever removes an entry.** A root that is
deleted, moved or renamed leaves its directory behind for ever, and mrw can never consult it again,
because the key is a hash of a path that no longer resolves.

Measured on M's machine at `/Users/zy/.local/state/mrw`. Two readings, hours apart on 2026-09-07,
reported as taken rather than reconciled — the growth between them is itself the finding:

| | first reading | second reading |
|---|---|---|
| state directories | **22,836** | **24,067** |
| whose `root` marker names a path that is gone | **22,591 (98.9%)** | 23,809 (98.9%) |
| whose root still exists | 245 | — |
| with no readable `root` marker | 0 | — |
| disk used (`du -sh`) | **242 MB** | **256 MB** |
| apparent size (`du -shA`) | — | 47 MB |
| the files themselves (sum of `stat` sizes) | — | **10.7 MB** across 65,235 files |

⚠ **THE THREE SIZES ARE DIFFERENT QUESTIONS AND ONLY ONE OF THEM IS "DATA".** Eleven megabytes of
ledgers occupy 256 MB of disk because each of 65,235 tiny files takes a whole block and each of
24,067 directories costs its own. The reclaimable figure is the `du` one; the content figure is the
`stat` one; and a record that prints "242 MB" without saying which reads as 242 MB of data, which it
is not. The first reading has no `-A` or `stat` column because those were not taken at the time, and
scaling the second reading's ratio back would be an inference dressed as a measurement.

`Entry.Bytes` and the `--prune` report use the CONTENT figure, because block size is a filesystem
property mrw cannot portably ask about — so the space a prune actually returns is LARGER than the
number it prints, never smaller.

Between the two readings the base grew by 1,231 directories in a few hours of ordinary work. This is
not a historical accumulation that has settled, and inodes rather than bytes are what it is really
spending.

**The producer is this repository's own gate.** Classified from the 22,591 dead roots: 22,590 are
`mktemp` fixtures and exactly 1 is a Go `t.TempDir()` — 11 of the 32 test files already pin
`XDG_STATE_HOME` into their own temp directory, and the untidy ones are rare. `scripts/contract.sh`
does not pin it, deliberately, and says so at `scripts/contract.sh:1588`: *"contract.sh does not pin
`XDG_STATE_HOME`, it isolates by giving every fixture a fresh root."* Isolation by fresh root is
correct and this record does not disturb it; what it misses is that a fresh root is also a fresh
key, so every `fixture` call in a run that ends with `rm -rf "$WORK"` converts one temporary
directory into one permanent one.

**Two halves, and only one of them is a script change.** A gate that stops leaking does nothing
about the 242 MB already there, or about the same accumulation on any other machine that has ever
run mrw against a temporary directory. mrw is the only thing that can clean it: the key is a hash,
so a human staring at `a3f81c0e77b2d491/` cannot tell which checkout it belonged to without reading
the marker mrw itself writes.

That marker is the whole reason this is an exact rule rather than a heuristic. `Dir()` writes it
already, and `internal/state/state.go:40` says why in advance — *"so an orphan left behind by a
moved or deleted repository is identifiable rather than an anonymous hash."* The mechanism for
identifying garbage has been in the tree since ADR-004; nothing read it.

**ADR-004 pre-registered every part of this, including the objection.** It deferred *"Pruning
orphaned state directories"* to `docs/adr/BACKLOG.md:112`, and the entry there states the reason a
reaper would be wrong: *"deciding a directory is dead means deciding a path will never come back,
and a tool should not decide that."* That is correct and this record does not overturn it — it is
precisely why the prune is an explicit command and never runs on its own. A tool still does not
decide; it reports what the marker says and removes it when an operator asks.

What the same passage got wrong is the SIZE. ADR-004's Consequence at `:164` and the backlog entry
both say each orphan is *"a few hundred bytes"* and that *"a human can clean up"*. Measured today:
242 MB across 22,836 directories is ~10.6 KB each, thirty times the estimate, and the cleanup the
backlog offers is `grep -r . …/mrw/*/root` over 22,836 files followed by 22,591 `rm -rf`s the human
must select by hand. Both claims were reasonable when written and neither survived contact with a
test suite that makes a fresh root per fixture; the estimate is corrected here rather than left to
be re-read as current.

## Existing Primitives Audit

- **The `root` marker (`internal/state/state.go:60`)** — written on every `Dir()` call, holds the
  symlink-resolved absolute root, and its doc comment names orphan identification as its purpose.
  **Reused unchanged, and it is the entire test**: an entry is garbage when the path this file names
  is no longer a directory. No new file, no new format, no timestamps to interpret.
- **`absReal` (`:139`)** — resolves symlinks, so the marker holds the same spelling `Dir()` would
  compute for a live checkout today. **Reused**: this is what makes `os.Stat` on the marker's
  contents an existence test rather than a spelling test. A repository reached through a symlink
  that still exists is alive under both spellings.
- **`stateHome` (`:119`)** — resolves the base and refuses a relative `XDG_STATE_HOME`. **Reused
  unchanged**; the prune inherits that refusal, so it can never be pointed at a directory relative
  to the working tree.
- **`LegacyDir` (`:35`)** — the pre-ADR-004 in-tree `.mrw/`, documented as *"never written or
  deleted"*. **Untouched.** It is inside the checkout and not under the base at all, so it is
  outside the prune's reach by construction rather than by a check that could be removed.
- **`seenCmd` (`cmd/mrw/main.go:318`)** — already the command that answers "where is my state".
  **Reused as the host**: the prune is a flag on it rather than a new subcommand, because a caller
  who wants to know what mrw is keeping and a caller who wants to drop what it no longer needs are
  asking about the same thing.
- **`mrw stats`, `mrw iter`, `mrw check`** — considered and **not reused**. None of them is about
  the state base; `stats` reads one file inside one entry, `iter` edits the working set, `check`
  runs the project's tests.

## Decision

**An entry under `$XDG_STATE_HOME/mrw/` whose `root` marker names a path that is no longer a
directory is garbage, and `mrw seen --prune` removes it and says what it removed. Nothing else ever
deletes it.**

Three parts.

**1. `state.Entries()` reports the base.** It lists every direct child of `<base>/mrw/`, and for
each returns the directory, the root its marker names, whether that root still resolves to a
directory, and the entry's size on disk. It judges nothing and deletes nothing.

**2. `state.Prune()` removes only what is provably unreachable**, and returns the entries it
removed so the caller can report them. It refuses to touch, in order:

- anything that is not a direct child of `<base>/mrw/` — the walk is a single `ReadDir`, never
  recursive, and a child that is not a directory is skipped;
- an entry whose `root` marker is missing, unreadable, empty, or not an absolute path. **These are
  KEPT and reported as unidentifiable.** Deleting something of unknown provenance is the one
  mistake here that cannot be undone by re-reading a file, so unknown always means keep;
- the entry belonging to the root in hand, whatever its marker says — a checkout being pruned from
  inside itself is not garbage even if it has just been deleted underneath the process;
- a symlink. `ReadDir` gives the entry type without following it, and an entry that is a symlink is
  reported and skipped rather than followed out of the base.

**3. `mrw seen --prune` is the only caller**, and `--prune --dry-run` prints the identical report
and removes nothing. `mrw seen` without the flag gains one line — how many entries the base holds,
and, when it holds more than one, that `--prune` exists. That line costs a single `ReadDir` plus one
`Lstat` per child and makes no claim about how many are dead, because counting the dead means
reading 22,836 markers and stating every checkout they name — and the point of the line is to cost
nothing. ⚠ **It called `Entries()` and did exactly that until 2026-09-08**, when a review found the
comment above the call describing the cheap behaviour while the call did the expensive one;
`state.Count` is what makes this paragraph true.

**No automatic deletion, on any code path, ever.** A root that does not resolve means *deleted* or
*on a volume that is not mounted right now*, and mrw cannot tell those apart. An explicit command
makes the operator the one who can. This is the same shape as `Migrate`, which copies and never
deletes for the same reason, and it is why `--dry-run` exists on the same flag rather than in a
follow-up.

**What the stakes actually are, corrected 2026-09-08 after a review found this understated:** the
LEDGER is a licence to edit and not a record of content (ADR-002), so losing it costs a re-read and
cannot produce a wrong edit. But the entry holds more than the ledger. `internal/iter` keeps the
working set there and `internal/authoring` keeps the plan-outcome tally there — `mrw iter` and
`mrw stats` are their readers — and **neither is recreated by re-reading source.** A wrong prune
therefore costs a re-read PLUS a working set the caller must re-specify and a tally that is simply
gone. Still no shape of this mistake produces a wrong edit, which is what makes an explicit,
reporting, exact-test prune sufficient and an age-out heuristic unnecessary; but "a re-read and
nothing else" was wrong and is withdrawn.

**What would falsify this:** an operator who runs `--prune` with an external disk unmounted and
loses ledgers they wanted. The report names every root it removed, and `--dry-run` shows the same
list first, so the failure is visible and recoverable by re-reading. If it is reported in practice,
the answer is a refusal when the marker's volume is absent — not an age gate, which would still
delete the same entries a fortnight later.

**And the gate stops producing them.** `scripts/contract.sh` exports `XDG_STATE_HOME` into the
`$WORK` directory its existing `EXIT` trap already removes. Fresh-root isolation is unchanged and
still the mechanism; the pin is a second layer that puts the state beside the fixtures that caused
it, so a run leaves nothing behind on any machine.

## Alternatives Considered

- **Prune automatically on every invocation.** Rejected: an absent root is also an unmounted
  volume, and "mrw silently deleted my ledger during an unrelated read" is a worse defect than the
  disk it saves. It also puts a full `ReadDir` plus one file read per entry — 22,836 of them today —
  on the hot path of every command.
- **Age out entries untouched for N days.** Rejected on two counts. N is a number this record would
  have to defend and nothing here can measure the right value; and the marker already gives an
  EXACT test, so replacing it with a heuristic is a downgrade that still deletes the unmounted
  volume's state, just later.
- **Document it as the user's to clean, and ship nothing.** Rejected: the key is a truncated hash,
  so the only thing that can map an entry back to a checkout is the marker mrw wrote. Telling a
  user to clean 22,836 opaque directories by hand is telling them to `rm -rf` the base and lose the
  245 live ledgers with them.
- **Prune unidentifiable entries too, since they are unusable anyway.** Rejected: an unreadable
  marker is a permissions error or a partial write as easily as it is garbage, and it is the one
  case where being wrong is not recoverable by re-reading. Reported, never removed.
- **A separate `mrw prune` subcommand.** Rejected: `mrw seen` is already the answer to "what is mrw
  keeping"; a second top-level verb for the same subject is a surface that has to be discovered
  separately.
- **Fix only `scripts/contract.sh` and file the rest.** Rejected — this is the workaround M
  declined. It leaves 242 MB in place on this machine and does nothing for any other.

## Component / Boundary Impact

`internal/state` gains a second responsibility — describing and pruning the base — beside its
existing one of resolving a path within it. Both are "where mrw's state lives", one reason to
change, and the marker one reads is the marker the other writes; splitting them into two packages
would put the writer and the only reader of that file apart. `cmd/mrw` gains two flags on an
existing command. `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and
`internal/check` are untouched, and the engine go/no-go clause in every task's fence asserts it.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `state.Entry`, `state.Entries()`, `state.Prune()` | new, internal package | T1 | T2 |
| `mrw seen --prune`, `mrw seen --dry-run` | new flags on an existing command | T2 | callers, contract §71 |
| `mrw seen` stdout | one added line before the ledger | T2 | contract §71, and any script parsing `mrw seen` |
| `scripts/contract.sh` `XDG_STATE_HOME` | exported into `$WORK` | T4 | the whole script |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `state.Entries()` / `state.Prune()` | T1 | T2 | No — new API, no existing caller |
| `mrw seen --prune` and its report format | T2 | T3 | No — new flag |
| The added line on plain `mrw seen` | T2 | T3 | ⚠ Yes for a script reading `mrw seen` positionally past line 1. The directory stays line 1, which is what `scripts/contract.sh:1590` and the documented recipe use |

## Implementation

See `docs/adr/ADR-034-state-that-names-a-checkout-nobody-has/tasks/README.md`.

## Consequences

- **Positive:** the 242 MB is removable by one command, and every entry it removes is named with the
  checkout it belonged to, per ADR-008.
- **Positive:** `mrw seen` answers "how much is mrw keeping" without an argument, so the growth is
  visible before it is 22,836 directories.
- **Positive:** the repository stops producing the garbage: a `contract.sh` run leaves the base
  exactly as it found it.
- **Negative:** mrw deletes directories for the first time. The blast radius is bounded by the base,
  by the marker test, and by the flag, but the capability is new and that is why this is a record.
- **Negative:** a wrongly-pruned entry costs a re-read before the next write to those files, and
  also loses the iteration working set and the authoring tally kept beside the ledger — those do not
  come back from source. It cannot cost a wrong edit — see the Decision — but the loss is larger
  than this record first claimed.
- **Neutral:** entries with no readable marker accumulate for ever by design. There are none today
  and the report names them, so if that changes it will be visible rather than silently pruned.

## Out of Scope

- Removing the 1 Go test that leaks its state, and pinning `XDG_STATE_HOME` in the 21 test files that do not (deferred: `docs/adr/BACKLOG.md`)
- A prune that refuses when the marker's volume is absent rather than deleting (deferred: `docs/adr/BACKLOG.md` — pre-registered there so the criterion predates the first report)
- Pruning `LegacyDir`, the pre-ADR-004 in-tree `.mrw/` (permanent: boundary: it is inside the caller's checkout, may have been committed, and ADR-004 chose to leave deletion to the human who can see it)
- Any change to how a key is computed, or to what a state directory contains (permanent: boundary: this record reads the base, it does not reshape it)
- Reclaiming space inside a LIVE entry — a large `seen` ledger for a checkout that still exists (permanent: boundary: that entry is reachable and in use; dropping it would refuse writes the caller has earned)
- An `--older-than` or `--all` variant of the flag (permanent: fact: the marker test is exact, so a time-based selector adds a knob with no case it answers better; citation: file `internal/state/state.go:40`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A prune deletes state for a checkout on an unmounted volume | Medium | Low | `--dry-run` shows the list first and the report names every root. A lost ledger costs a re-read and cannot cause a wrong edit (ADR-002); the iteration working set and the tally beside it are lost outright, which is the real cost |
| The fixture passes against a prune that deletes everything | **High** — this is the default way to get it wrong | The fixture holds a live entry, a dead entry AND an unidentifiable entry in one base, and asserts the first and third survive. A prune that removes all three fails it, and the T1 mutant proves the distinction held |
| The walk follows a symlink out of the base | Low | High | `ReadDir` reports the entry type without following; a non-directory or a symlink is skipped and reported. Asserted by its own fixture |
| The added `mrw seen` line breaks a caller parsing the output | Low | Medium | The state directory stays the FIRST line, which is what `scripts/contract.sh:1590` and the documented recipe read. §71 pins that |
| Pinning `XDG_STATE_HOME` in `contract.sh` changes a row's verdict | Low | Medium | No row asserts the base's location; the one row that reads it takes it from `m seen \| head -1` rather than constructing it. T4's fence is the whole script |

## Rollback

Revert the commit. `state.Entries` and `state.Prune` have no caller but `seenCmd`, and no other code
path reads the marker, so removing them restores exactly the previous behaviour. **State already
pruned is not recoverable** — but a missing ledger is re-created by reading the files again, which
is the same position a fresh checkout starts from.

## Follow-ups

- [ ] If a `--prune` on an unmounted volume is reported in the wild, add a refusal when the marker's volume is absent, rather than an age gate.
