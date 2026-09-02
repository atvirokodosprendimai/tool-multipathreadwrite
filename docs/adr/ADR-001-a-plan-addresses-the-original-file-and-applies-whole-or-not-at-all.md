# ADR-001: A plan addresses the original file, and applies whole or not at all

**Status:** Accepted
**Date:** 2026-08-31
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-002 (adds the precondition that the file being addressed is one mrw has seen; this record assumes the addresses are honest, ADR-002 makes that checkable), ADR-003 (owns what happens after a write lands)
**Governs:** `internal/plan/**`, `internal/apply/**`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/apply_test.go::TestEarlierHunksNeverShiftLaterAddresses`
**Invalidates:** none — checked (this is the first record in this corpus; `node scripts/adr-state.mjs` reported "No decision records found under this repository" on 2026-08-31)
**Served-path change:** An agent can change N sites across M files in one `mrw write` call and is told, per hunk, which ones would not apply — where previously `Edit` did one replacement per call and `Write` could not report a change that silently matched nothing.

## Context

**This is a retrofit.** The decision was made and shipped before it was written
down: the engines landed in `676296e` (2026-08-31) and the behaviour has been
released as v0.0.1 and v0.0.2. The record exists because the *why* is the one
thing the repository cannot reconstruct, and because the plan format is a public
contract — anything that authors a `.mrw` file depends on these semantics.

Its consequence for execution is stated plainly rather than papered over: the
TDD-red step of each task happened historically and cannot be re-performed. The
proof this corpus will accept instead is a **mutation** — break the mechanism,
watch the fence go red — which is a stronger claim than a red test at authoring
time, and is what `adr-verify --mutant` records.

The originating failure, from the session that motivated the tool: four
replacements batched into one `python -` script, one of them silently matched
nothing, and the script printed success. Generalised: **a read that returns
nothing is visible; a write that changes nothing is not.**

Measured 2026-08-31 against this repository, by `./scripts/measure.sh`: for four
sites across four files, the Read+Edit path costs 50,163 bytes over 8 tool calls
against 2,916 bytes over 2. For one site in a file you need whole, mrw costs
1.2× *more* and saves no round trips — the saving is a property of the task
shape, not of the tool.

**Amended 2026-09-02:** that byte figure has an unstated baseline, and it was
unstated here as well as in the README. It compares mrw against reading each
file WHOLE. `Read` takes `offset`/`limit`, so an agent that already knows the
line ranges reads only those, and against THAT baseline mrw costs about 1.2×
more bytes, not less — it adds a header and a line number per line. Re-measured
2026-09-02: 96,871 whole / 2,289 windowed / 2,951 mrw for the same four sites.

This does not disturb the decision, and it is worth saying why rather than
leaving it to be re-derived. The decision rests on ROUND TRIPS and on
all-or-nothing application, and round trips survive both baselines — they get
better under the windowed one, because a windowed read must first FIND the
range and mrw's regex specs do that inside the same call (9 calls versus 2,
where the whole-file path is 8 versus 2). What is conditional is the byte
saving, and the condition is "the agent does not yet know where to look" —
which is the common case, and the case regex specs exist for.

## Existing Primitives Audit

- **The harness `Edit` tool** — one replacement per call, fails loudly on a bad
  anchor. **Reused conceptually, not replaced**: `anchor=` is the same idea, and
  the tool's own guidance still routes one or two edits to `Edit`.
- **The harness `Write` tool** — whole-file replacement, refuses to overwrite an
  unread file. **Reshaped**: the read-before-write property is kept (ADR-002),
  the whole-file granularity is not.
- **Unified diff / `patch(1)`** — the obvious existing format. **Rejected**, see
  Alternatives: context matching makes the failure mode fuzzy, and hunks are
  offset-relative.
- **`sed -n 'A,Bp'`** — already used for cheap ranged reading; `internal/read`
  generalises it to many files and many ranges per call rather than replacing it.

## Decision

A plan is a **line-oriented document**, not JSON:

    @@ <path> <addr> <op> [sha=… lines=… anchor=… body=…]
    <body lines, verbatim>

Three properties are the decision, and each is load-bearing:

1. **Every address resolves against the ORIGINAL file.** Hunks are written
   top-to-bottom in any order; an insert at line 10 does not move the address of
   a replace at line 100. This removes the bottom-to-top ordering discipline that
   every `sed`-style batch edit requires, and it is the property that makes
   batching safe rather than merely convenient.
2. **Validate every hunk before writing anything; one failure aborts the run.**
   A partially applied plan is worse than no change, because the caller believes
   it succeeded. This holds against a FILESYSTEM failure as well as a validation
   one: every file is staged beside its target before any is renamed into place,
   so an unwritable directory, a read-only mount or a full disk aborts while the
   tree is still untouched — temp files unlinked, and any directories staging
   had to create removed with them. A write phase that wrote each file and moved
   on satisfied this rule against bad hunks and broke it against bad permissions.
   Two things followed from that, and **only one of them is closed.** The ledger
   is recorded from the receipt, so it used to disagree with files mrw had just
   written; it no longer can, because nothing is written. But the receipt itself
   is **still not rendered on this path** — the error returns before it, on the
   text and `--json` surfaces alike — so rule 3 below remains OPEN for a
   filesystem failure and this record does not close it. Only a failing *rename*
   can still leave the tree partial, and it names the files already written
   rather than reporting the bare error.
   **Both are closed as of 2026-09-02** — see the amendment under rule 3. The
   paragraph above is left as written because it records what was true while it
   stood, but "only one of them is closed" and "remains OPEN" are stated in the
   present tense, and a reader who arrives here and never reaches rule 3 would
   otherwise be told something false about the code in front of them.
3. **Every hunk carries its own verdict.** ⚠ Was open for a filesystem failure —
   that path returned an error and rendered no receipt at all — until 2026-09-02;
   see the amendment below, which closed it.
   Siblings of a failed hunk report
   `skipped`, never `ok` — "ok but not written" is precisely the lie being
   avoided. Every file the plan *addressed* appears in the receipt, written or
   not.

   **Amended 2026-09-02 — rule 3 is now CLOSED on the filesystem path, and the
   ⚠ above is retired.** The original text is kept because it records what was
   true for as long as it stood. What changed is the code, not the reading: the
   staging abort now assigns every hunk a verdict before it returns — the hunks
   of the file that could not be staged report `failed` with the filesystem
   error as their reason, every sibling reports `skipped` — fills the receipt
   with every file the plan addressed, none of them written, and renders it
   before exiting. `--json` renders a receipt too; on this path it previously
   emitted a bare `mrw: …: permission denied` and nothing parseable at all,
   which is the half a programmatic caller felt.

   The exit code is deliberately unchanged. A filesystem failure stays 2,
   distinct from a failing hunk's 1: they are different conditions and a caller
   that distinguishes them is doing something correct.

   Rule 2 is untouched by this. The abort was always correct — nothing was
   written, no temp survived, no directory staging created was left behind —
   and only the *reporting* was missing. The rename path is also untouched and
   is deliberately different in kind: a failing rename can leave the tree
   genuinely partial, so it names the files already written rather than
   pretending nothing happened.

   Pinned by `scripts/contract.sh`: the unstageable hunk reports FAIL, its
   sibling reports skip, the summary names *both* addressed files, and `--json`
   parses under `jq -e`. Four of those six rows go red if the receipt render is
   removed from the error path; the other two are nested behind the parse row
   and are guarded by it rather than independently proven.

Optional per-hunk guards make a batch safe to trust and are cheap to emit:
`sha=` (whole file), `lines=` (range size), `anchor=` (substring in the range's
first line).

**The format is not JSON because output tokens are the scarce currency.** JSON
escapes every newline and quote in every code body — the one part of the
document that is already large. This is a reasoned choice, not a measured one:
the input-side saving is measured (`scripts/measure.sh`), the escaping overhead
is not, and the record should not imply otherwise.

**What would falsify the batching claim:** if measurement showed multi-hunk
edits to be rare in practice, the round-trip saving would not pay for a
homegrown dependency. That objection was raised when the tool was proposed and
answered on 2026-08-31 by `scripts/measure.sh`, which is valid for *this*
repository's file sizes and says so; it is not a general claim about all
codebases.

## Alternatives Considered

- **Read the file, then Write it whole (2 calls, no new tool):** the position
  argued against building anything. Rejected because it does not batch across
  files, gives no per-hunk verdict, and re-emits the entire file — the expensive
  direction. It remains the right answer for one file, and the tool's own
  guidance says so.
- **A JSON plan:** rejected because escaping inflates code bodies, which are the
  bulk of the document, and because a hand-written JSON plan is materially
  harder to author correctly.
- **Unified diff / `patch(1)`:** rejected because context-line matching makes
  failure fuzzy ("hunk #2 succeeded at offset -3" is a silent relocation), and
  because offsets are relative to preceding hunks — the exact property decision
  point 1 exists to remove.
- **An MCP server instead of a binary:** rejected because a server binds at
  session start and is unrecoverable mid-session; a binary is re-invoked per
  call and cannot enter that state.
- **Applying hunks bottom-to-top and keeping relative offsets:** rejected
  because it pushes the ordering burden onto every caller, and a caller that
  gets it wrong produces a plausible-looking wrong file.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/plan` | Parsing the plan document into hunks. Knows the format, not the filesystem. | Yes — changes only when the format changes |
| `internal/apply` | Turning hunks into writes. Knows the filesystem, not the format. | Yes — changes only when write semantics change |

The split is deliberate: `apply` does not import `plan`, so the engine is
testable with hand-built hunks and the format can change without touching the
write path. `apply.Input` is the seam.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `.mrw` plan format (`@@` header grammar, ops, addresses, guards) | new public contract | `internal/plan` | any author of a plan file, human or agent |
| `apply.Input` / `apply.Options` / `apply.Result` | new internal contract | `internal/apply` | `cmd/mrw` |
| `mrw write --json` receipt (`files[]`, `hunks[]`, statuses) | new public contract | `cmd/mrw` | hooks, quality gates |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `plan.Hunk` / `plan.Parse` (T1) | T1 | T2, T3 | No — additive; T2/T3 read hunks, they do not define them |
| `apply.Apply` original-file resolution (T2) | T2 | T3 | No — T3 constrains the same function's failure path |

## Implementation

See `ADR-001-a-plan-addresses-the-original-file-and-applies-whole-or-not-at-all/tasks/README.md`.
Three tasks: the format, the addressing, the abort-and-report contract.

## Consequences

- **Positive:** N sites across M files cost 2 tool calls and no offset
  arithmetic. A wrong address fails loudly instead of writing plausible garbage.
  The receipt names every file the run was about to touch.
- **Positive:** the format is cheap to emit, which is the axis that actually
  costs (output tokens are ~5× input).
- **Negative:** a homegrown format is one more thing to learn, and one more
  thing that can drift from its documentation. Mitigated by the format being
  small enough to state in six lines, and by `scripts/contract.sh` asserting the
  semantics against the built binary.
- **Negative:** `apply` reads whole files into memory. Fine at the sizes
  measured (14 MB / 200k lines: 5 hunks in 0.15 s), unbounded in principle.
- **Neutral:** the tool deliberately does not replace `Edit` or `Write`, so the
  repository now has three ways to change a file and a rule for choosing.

## Out of Scope

- Fuzzy or context-based hunk matching, à la `patch` (permanent: the whole point
  is that a bad address FAILS rather than relocating)
- Automatic conflict resolution or three-way merge (permanent: mrw refuses and
  reports; deciding is the caller's job)
- Streaming or memory-bounded application for very large files (deferred: docs/adr/BACKLOG.md)
- A `cmd N` registry of saved commands addressable by number (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller replaces a multi-line range without reading it, and `anchor=`+`lines=` both pass while the END is wrong | Med | High — a closing brace or loop header is eaten silently | Documented in the skill as the guard's known blind spot; `--dry-run` shows the line delta; the compiler or the chained check catches it. Hit once during development (`internal/read/read.go`, a `for` header) |
| The format drifts from its documentation | Med | Med | `scripts/contract.sh` asserts the semantics against the built binary and runs in CI |
| Whole-file read makes a very large file expensive | Low | Med | Measured at 14 MB; deferred to BACKLOG if it becomes real |
| An agent uses mrw for one-site edits where it is worse | Med | Low | `scripts/measure.sh` publishes the losing shape; the skill's decision table routes those to `Edit` |

## Rollback

The plan format is a read contract with no persistent state of its own, so
rollback is a revert plus a version bump: `git revert` the engine commits and
re-tag. Anything holding a `.mrw` file keeps a plain-text document that a human
can apply by hand — the format's own legibility is the fallback. Consumers of
`mrw write --json` would need to handle the receipt disappearing; there are none
outside this repository as of 2026-08-31.

## Follow-ups

- [ ] Decide whether the deferred streaming/memory-bounded application is ever worth building (see BACKLOG.md)
