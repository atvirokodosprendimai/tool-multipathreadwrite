# ADR-007: mrw finds the files it serves

**Status:** Accepted
**Date:** 2026-09-01
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-006 (the root boundary this walk must obey), ADR-005 (the ledger rule this must not weaken — a span withheld is not observed), ADR-004 (the no-git-dependency premise the `.gitignore` non-goal rests on)
**Governs:** `internal/read/**`, `internal/rooted/**`, `cmd/mrw/main.go`, `scripts/contract.sh`, `README.md`
**Enforced-by:** None — nothing enforces this yet, because nothing is built. Naming ADR-006's read-confinement test here would have been worse than naming none: it is green on the merge-base and would stay green if no part of this decision were ever implemented. T2 introduces the walk's own boundary and observation tests; this header is updated to name them when they exist, which is the Follow-up below.
**Invalidates:** none — checked. Grepped every accepted record for `read`, `--max-lines`, `spec` and `Spec{`: ADR-005 defines what an observation records and ADR-006 defines the boundary; this record adds a way to NAME files and changes neither rule. `read.Spec` gains no field.
**Served-path change:** `mrw read --grep PATTERN <paths...>` walks the given paths (recursively, for a directory) and serves the ranges matching PATTERN, where today the caller must run a searcher first and compose one `path:/PATTERN/` spec per hit on the command line.

## Context

The pipeline this record is about already works, and was measured on this
repository on 2026-09-01 against three files containing `EvalSymlinks`
(`internal/apply/apply.go`, `internal/state/state.go`,
`internal/rooted/rooted.go`):

| | bytes |
|---|---|
| the three files read whole | 29,243 |
| `mrw read -C 3 f1:/EvalSymlinks/ f2:… f3:…` | 1,141 |

**25.6×**, in one `mrw` call, with overlapping context merged so no line is paid
for twice. The saving is not in question. Two things about getting there are.

**It is two calls, not one.** `mrw read` cannot find files; it can only be told
about them. So every use of this shape is `grep -rl` (or `rg -l`, or an agent's
own search) and then `mrw read`.

**The spec list is composed in a shell, and that is fragile.** Measured the same
day while writing the measurement above: newline-separated paths from `grep -rl`
collapsed into a single argument under ordinary word splitting, and mrw
faithfully reported a file named `"a.go\nb.go\nc.go"` as `UNREADABLE`. It failed
loudly, which is the tool working — but an agent emitting that command line hits
it exactly as a human does, and the fix is quoting knowledge that has nothing to
do with the task.

The question this record settles is not whether the saving is real. It is
whether closing the two-call gap is worth mrw knowing how to walk a directory —
because that is the first capability it would have that is about FINDING files
rather than about serving or changing them.

## Existing Primitives Audit

- **`read.Range` with a `/pattern/` form and `-C N` context** (`internal/read/read.go`):
  the regex engine this needs already exists and is tested. **Reused unchanged** —
  `--grep` applies the same engine to a set of files instead of one, and adds no
  pattern syntax. Case-insensitivity is already available as Go's `(?i)`.
- **`rooted.Resolve`** (`internal/rooted/rooted.go`): the boundary from ADR-006.
  It already canonicalises the root once with `EvalSymlinks`, so a symlinked
  root is accepted and everything is compared against its real path.
  **Reused, and extended by one function** — a walk needs to know whether to
  DESCEND into a directory, which is the same question about a different kind of
  path.
- **`read.Run`'s span merging and `--max-lines` budget:** **reused unchanged.**
  The budget is per SPEC and is reset for each one (`internal/read/read.go:258`),
  which is why deduplication below is a correctness rule and not a tidiness one.
- **`iter`, the working set:** `mrw read` with no arguments reads it today
  (`README.md:311`). **Untouched** — `--grep` with no paths walks `--root`, and
  without `--grep` the no-argument behaviour is exactly what it is now.
- **`grep`, `rg`, `git grep`:** the tools that do this today. **Reused as the
  runner-up** (`--files-from -`, below), not reimplemented: mrw gains a walk and
  a filter, not ranking, counts, replace, or `--files-without-match`.

## Decision

`mrw read --grep PATTERN <paths...>` serves every range of every file under
`<paths...>` that matches PATTERN, as if the caller had written
`path:/PATTERN/` for each matching file.

**1. What is walked.** A named file is served; a named directory is walked
recursively; with no paths and `--grep` given, the walk starts at `--root`.
Without `--grep`, no-argument `mrw read` keeps reading the working set.

**2. What is a candidate.** A candidate must resolve to a REGULAR FILE, and the
question is asked AFTER `rooted.Resolve`, which already follows symlinks. So a
symlink to a file inside the root is a candidate — `mrw read link.txt` serves it
today and `--grep` must not start refusing it — while a FIFO, a device, a socket
or a symlink whose target is one of those is not. A non-candidate found by
WALKING is skipped silently, because the caller did not name it and mrw would
otherwise block on a pipe or stream a device without end. One the caller names
EXPLICITLY is refused by name: naming something mrw will not read must produce a
line, not silence.

This is stated as "resolve, then ask" rather than "`os.Lstat` reports regular"
because the two differ on exactly the case rule 4 depends on. An `Lstat` check
would refuse every symlink, contradicting rule 4's promise that an in-root
symlink is a candidate of its own, and would regress a served path.

**3. The boundary, and what is authoritative.** Every candidate passes
`rooted.Resolve`; a symlinked DIRECTORY is never descended, because following
one can leave the tree and can loop. The walk's own reads are for MATCHING only
and record nothing: `read.Run`'s read is the authoritative one and the only one
that observes. A file that changes between the walk and the serve is served as
`Run` finds it — if it no longer matches, nothing is printed and nothing is
observed, which is the honest answer and is tested.

**4. Identity and duplicates.** Two specs for one file would double its
`--max-lines` budget, because `Run` resets the budget per spec — verified
2026-09-01: `mrw read --max-lines 2 f.txt f.txt` prints four lines. So the walk
deduplicates on the CLEANED ROOT-RELATIVE PATH before serving, which makes
"the cap is per file" true again for everything the walk produces. Two paths
that reach one inode by different names — a hardlink, an in-root symlink — are
DIFFERENT candidates and are each served, because mrw addresses files by path
and pretending otherwise would make an address ambiguous.

**5. Failures are per path.** A refused path, an unreadable file or a directory
that cannot be listed is reported and counted as a problem; the walk continues
and every valid sibling is still served. Only a `--root` that cannot be resolved
aborts. This is the existing batch behaviour of `read.Run` — "a missing file is
reported in the output rather than aborting the batch" — applied to discovery.

**6. `.git/` is skipped; `.gitignore` is not read.** mrw has no git dependency
and ADR-004 rejected `.git/mrw/` partly to keep it that way. `--exclude GLOB`
(repeatable) is the caller's control.

### Go/no-go: what makes this the wrong decision

Rule counting is not falsifiable, so this is stated as conditions checked during
T2 and recorded in its verification log. **If any of these fails, `--grep` is
withdrawn and `--files-from` — which T3 ships regardless — is the whole answer:**

- **Candidate types:** the walk serves regular files and needs no further
  content policy (no binary sniffing, no encoding detection).
- **Dependencies:** no new module in `go.mod`.
- **Flag surface:** no flag beyond the three this record names.
- **Cost:** `--grep` over this repository, for a pattern matching about three
  files, completes within **2×** the wall-clock of the `grep -rl` + `mrw read`
  baseline measured 2026-09-01 on this machine, measured the same way on the
  same machine.

## Alternatives Considered

- **`mrw read --files-from -` (specs on stdin):** the runner-up, and cheap —
  around twenty lines, no walking, no ignore-file question, no boundary question
  beyond the one that already holds. `grep -rl PAT . | sed 's|$|:/PAT/|' | mrw
  read -C 3 --files-from -` is one mrw call and no argv quoting. Rejected as the
  ONLY answer because it leaves the round trip in place and still asks the
  caller to compose specs, just in a pipe instead of a command line — but it is
  strictly smaller, and it is what to build if any go/no-go condition fails. It
  is not mutually exclusive with this decision; T3 ships it.
- **Leave it as it is and document the `grep -rl` recipe in the README:** free,
  and it is what the tool does today. Rejected because the recipe is exactly the
  shell quoting that broke on first contact, and a README recipe that breaks
  under word splitting teaches the failure rather than preventing it.
- **A `mrw search` subcommand distinct from `read`:** rejected — it would need
  its own output format, its own ledger rule and its own answer to "what did the
  caller see", and the whole value here is that the answer is already `read`'s.
- **Honour `.gitignore`:** rejected on ADR-004's premise rather than on taste —
  mrw does not require git (`mrw -C /any/dir` works today), so honouring it
  needs either a git subprocess or a reimplementation of its matching rules, and
  the second is a search tool's job. `--exclude GLOB` covers the cases that
  motivated it.
- **Deduplicate by inode rather than by path:** rejected. mrw addresses files by
  path everywhere else — the ledger, the plan format, the receipt — and an
  inode-keyed walk would serve one of two names and silently drop the other.
- **Shell out to `rg` when present:** rejected. It makes mrw's output depend on
  what is installed, and the receipt would describe a search mrw did not do.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/rooted` | What "inside the root" means, and now what may be DESCENDED into | Yes — both are the same question about a path |
| `internal/read` | Turning a request into files and ranges, and serving them | Yes — it already owns the second half |
| `cmd/mrw` | Flag surface, precedence, and help text | Yes |

`internal/read` gains the walk rather than a new package, because the walk's
only purpose is to produce `Spec` values that `read.Run` already knows how to
serve. A separate package would have one caller and one consumer.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --grep PATTERN` | new public flag | `cmd/mrw` | callers, agents, CI |
| `mrw read --exclude GLOB` (repeatable) | new public flag | `cmd/mrw` | callers |
| `mrw read --files-from FILE\|-` | new public flag | `cmd/mrw` | callers, pipelines |
| `read.Walk(root string, paths []string, opt WalkOptions) ([]Spec, []Problem, error)` | new internal API | `internal/read` | `cmd/mrw` |
| `read.Problem{Path, Reason string}` | new internal type | `internal/read` | `cmd/mrw` |
| `rooted.Descendable(absRoot, path string) (bool, error)` | new internal API | `internal/read` | `internal/read` |
| `mrw read` with no arguments | unchanged (reads the working set) unless `--grep` is given, in which case it walks `--root` | `cmd/mrw` | callers |

### Flag precedence

| combination | behaviour |
|---|---|
| `--grep P` + no paths | walk `--root` |
| `--grep P` + paths | walk those paths |
| `--grep P` + a positional spec carrying its own `:RANGE` | usage error — the range and the pattern are two answers to one question |
| `--grep P` + `--files-from` | usage error — two sources of specs |
| `--files-from` + positional paths | usage error — same reason |
| `--exclude` without `--grep` | usage error — nothing to exclude from |
| `--grep P` + no paths + a non-empty working set | walk `--root`, NOT the working set — `iter` is a set of read specs and `--grep` supplies its own; a caller who wants both writes `mrw read --grep P @1 @2` |
| `--grep P` + `--stat` | allowed: the matching files' headers, no content, observing nothing (ADR-005) |
| no `--grep`, no paths | unchanged: the working set |

### The exclusion algorithm

`--exclude GLOB` is matched with `path.Match` against BOTH a candidate's cleaned
root-relative path AND its basename, and a match on either excludes it. Matching
a DIRECTORY prunes it and everything under it. An EXPLICITLY named path is never
pruned — the caller named it — and neither is `.git/` when named explicitly,
though it is always skipped during a walk.

The basename half is not a convenience, it is the difference between the flag
working and doing nothing. `path.Match`'s `*` does not cross `/`, and `**` is
not a token — it is two `*`, neither of which crosses either. Verified
2026-09-01 against `path.Match`:

    glob "*.go"      vs "internal/read/walk.go"      -> false
    glob "*_test.go" vs "internal/read/read_test.go" -> false
    glob "testdata"  vs "internal/read/testdata"     -> false
    glob "**/*.go"   vs "internal/read/walk.go"      -> false

None of those is `ErrBadPattern`, so the parse-time guard sees nothing wrong: a
caller writing `--exclude '*_test.go'` against the full path alone would get no
error and every test file. Matching the basename too makes each of those work as
written. A caller who means a path rather than a name writes one —
`internal/read/testdata` matches the path form.

A glob `path.Match` rejects is a usage error at parse time, not a pattern that
silently matches nothing. Matching is case-sensitive on every platform, so a
case-insensitive filesystem may hold a file an exclusion does not catch; that is
stated rather than special-cased.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `rooted.Descendable` (T1) | T1 | T2 | No — new function, no existing caller |
| `read.Walk`, `read.WalkOptions`, `read.Problem` (T2) | T2 | T3 | No — new API |

## Implementation

See `ADR-007-mrw-finds-the-files-it-serves/tasks/README.md`. Three tasks: the
descend rule, the walk that produces specs, and the CLI surface plus the
runner-up `--files-from`.

## Consequences

- **Positive:** the pipeline collapses from two calls to one, and the caller
  stops composing specs in a shell. The 25.6× measured above is unchanged; what
  changes is the cost of getting it.
- **Positive:** `--files-from` lands regardless, so any searcher — `rg`, `git
  grep`, an agent's own index — composes with mrw in one call.
- **Positive:** deduplication makes `--max-lines` mean per file for anything the
  walk produces, where two hand-written specs for one file double it today.
- **Negative:** mrw gains a directory walk, which is a capability about finding
  rather than serving. Every future question about search pressure lands here.
- **Negative:** a `--grep` walk reads each candidate twice — once to match, once
  to serve. That is the price of `Run` staying the authoritative reader; the
  go/no-go cost condition is what decides whether it is affordable.
- **Neutral:** a file that changes between the walk and the serve may be served
  with no matching range. It is visible in the output and observes nothing.

## Out of Scope

- Ranking, counts-only output, `--files-without-match`, replace, or any other
  searcher feature beyond "which ranges match" (permanent: boundary: the value here is that the answer is already `read`'s output; every one of these needs its own output format and its own ledger rule)
- Honouring `.gitignore` or any ignore file (permanent: fact: mrw has no git dependency and ADR-004 rejected `.git/mrw/` partly to preserve that; citation: file `docs/adr/ADR-004-mrw-leaves-nothing-in-the-working-tree.md:81`)
- Deduplicating two paths that reach one inode (permanent: boundary: mrw addresses files by path everywhere else, and an inode-keyed walk would silently drop one of two valid addresses)
- Binary or encoding detection for discovered candidates (permanent: boundary: rule 2 admits regular files only, and a content heuristic that silently skips one is the silent-omission shape this project refuses)
- A cross-file `--max-lines` budget (deferred: docs/adr/BACKLOG.md)
- Parallel walking or searching (deferred: docs/adr/BACKLOG.md)
- Atomic scan-to-serve — mrw does not hold candidates open between the walk and the serve (permanent: boundary: rule 3 makes `Run`'s read authoritative, so the window is visible in the output rather than closed by locking)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The walk becomes a search tool by increments | Med | High | The go/no-go conditions in Decision, checked in T2 and recorded in its verification log; failing any withdraws `--grep` |
| Reading each candidate twice is too slow on a large tree | Med | Med | The cost condition names a threshold, a corpus and a machine; T2 measures before T3 ships the flag |
| A FIFO or device blocks the walk forever | Low | High | Rule 2 — discovered candidates are regular files; T2 tests a FIFO in the tree |
| Duplicate specs double a `--max-lines` budget | High without the rule | Med | Rule 4 — dedup on cleaned root-relative path, tested with a path named twice and with a directory overlapping a named file |
| A symlinked directory loop hangs the walk | Low | High | Rule 3 — symlinked directories are not descended at all; T1's test walks a self-referential link |
| `--exclude` globs behave differently from a caller's shell globs | Med | Low | The exclusion algorithm above, pinned by CLI tests and a contract row |

## Rollback

Revert the three commits. The flags are additive, `read.Spec` is unchanged, and
the ledger format is untouched, so a tree written by the new binary is served
identically by the old one. No persistent state migrates.

## Follow-ups

- [ ] Replace `Enforced-by` with the tests T2 introduces, once they exist
