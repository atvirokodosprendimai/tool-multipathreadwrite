# ADR-020: The served-size curve is measured, not asserted

**Status:** Accepted
**Accepted:** 2026-09-04 by M — standing goal, stated as *"budget … this now limits us from full tool potential, need to rethink this. some claim to do 10k tokens and call it a day, but us — we need to know if the max throughput hurts or behaves."*
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Served-path change:** none. Nothing about what `mrw` reads, serves, refuses or applies changes. This record adds a SECOND binary, `curve`, which generates fixtures and scores plans somebody else authored.
**Cross-references:** ADR-009 (owns the outcome tally this borrows as secondary DVs), ADR-011-T3 (owns `MaxResultChars`, the number this exists to justify or retire), ADR-014 (owns the RSS-against-served-bytes measurement, which is the WRONG instrument for this question and is why the record says so), ADR-012 (owns the finding that a measurement pointed at the wrong population is worse than none)
**Governs:** `internal/curve/**`, `cmd/curve/**`
**Enforced-by:** `internal/curve/score_test.go::TestTheScorerCountsAWrongLineAsAMiss`
**Invalidates:** none. It does not settle `MaxResultChars` either — it builds the instrument that could.

## Context

`mcp.MaxResultChars` is 200,000. Nothing in this corpus says why, and ADR-011-T3 does not claim to:
it bounds a resource so a host is not handed something it cannot use. Every gate around it — the
index degradation, the served-page degradation, the contract rows — asserts the bound is RESPECTED.
None of them asks whether the bound is RIGHT.

**A pre-registration for the answer already exists**, written 2026-09-04 in
`docs/adr/BACKLOG.md` before any harness existed and before any datum was collected, deliberately
placed there rather than in a record because a criterion authored after the first look is not a
criterion. This record builds the instrument that pre-registration describes and changes none of it.

What it commits to, restated so this record is readable on its own:

- **Primary DV: did the plan address the line it was supposed to address?** The failure that parses,
  applies cleanly, reports `ok`, and changed the wrong line. Every other gate is blind to it, and it
  is the failure `mrw` exists to prevent.
- **Ground truth is PLANTED, not judged.** The generator places the target and therefore knows the
  correct line by construction. No rubric, no rater.
- `applied` / `refused_parse` / `refused_apply` are **secondary**. They measure whether the caller
  could author the FORMAT, which has nothing to do with how much was served; they will be flat, and
  a flat curve there would read as "the cap does not matter" when it means "the easy thing was
  measured".
- **IV: served bytes**, varied by padding on both sides. The instruction and the target CONTENT are
  identical across cells; the ADDRESS is not, and cannot be — padding adds lines.
- **Distractor count is held CONSTANT across size cells**, with the remaining padding inert. If the
  padding were itself near-identical distractors, "more context" could not be separated from "more
  candidates".
- **Target position is stratified**, not randomised away.
- **N trials per cell, fixed in advance**, reported as a proportion with an interval.
- **A flat curve is a publishable answer.** The null is accepted in advance; the cap then stays
  arbitrary *with evidence*.

**The obvious instrument is the wrong one and is already built.** ADR-014 records peak RSS against
served bytes. That is what the SERVER spends. The question is what the CALLER's accuracy does, and
using a memory curve to justify a context budget is the category error ADR-012 rejected one level up.

## Existing Primitives Audit

- **`plan.Parse` (ADR-001):** the real parser. **Reused unchanged.** A scorer with its own parser
  would measure agreement with the scorer, not with `mrw`.
- **`apply.Apply` (ADR-001):** the real applier. **Reused unchanged, and this is the design's centre
  of gravity** — see Decision 3. Scoring by comparing address STRINGS would have been simpler and
  would have measured a different thing.
- **`seen.SHA` and `apply.Options.Seen` (ADR-002):** the ledger's hash and its in-memory shape.
  **Used unchanged** to license the fixture WHOLE so the read-before-write guard cannot convert a
  wrong address into a refusal — see Decision 4. ⚠ Note what this is NOT: the harness builds the
  observation in memory and never calls `seen.Record`, because nothing about a scoring run should
  outlive it. An earlier revision of this row said "`seen.Record` … reused unchanged", which was a
  primitive the code does not call. Corrected after review of #85.
- **`state.Dir` (ADR-004):** **NOT called.** The harness creates the per-cell directory itself and
  hands it to the runner as `XDG_STATE_HOME`; `state.Dir` is what MRW then uses to resolve it. The
  reuse is of the PROTOCOL, one process away, not of the function.
- **`mcp.MaxResultChars`:** **referenced in prose only.** The size is a caller-supplied flag; this
  record measures the constant's value but no code here reads it.
- **The stdlib `flag` package:** taken for the new binary rather than the CLI framework, so `go.mod`
  still declares exactly one requirement.

## Decision

**1. The harness generates and scores. It does not call a model.** `curve generate` writes a fixture
tree and a manifest; something else — any agent, any harness, any model — authors plans; `curve
score` reads the results back and reports the cell. No network, no model client, no second
dependency.

This is not only about `go.mod`. A harness that can only measure whatever it was wired to produces a
curve about one client; a manifest-in/results-in harness produces a curve any client can be run
through, which is the difference between a number and a finding.

**2. The scorer refuses results it did not ask for.** The manifest carries a trial id and the
measured served-byte count; the results must echo both, and a mismatch is refused rather than scored.
A manifest emitted at 190 KB with results pasted back from a 10 KB run would otherwise score
perfectly and mean nothing — the same class as a fixture that never reaches the bug.
The manifest carries NO ground truth in the wider sense either: the target's stratum and the
distractor count are written to `answer.json`, not to the manifest, because together they name
the target's block by arithmetic — early is the first block, late the last — and a client holding
them could localise by counting instead of by reading. Found by review of #85; the first cut leaked
both.

**3. The primary DV is measured by APPLYING, not by comparing addresses.** The scorer copies the
fixture, runs `apply.Apply`, and diffs against the original: **the changed line number IS the
measurement**. This is what the pre-registration's wording literally says — "an edit that parses,
applies cleanly, reports `ok`, and changed the wrong line" — and it removes the whole class of
scorer bugs where a pattern address is resolved one way by the scorer and another way by `mrw`.

**4. The ledger is seeded WHOLE, so it cannot mask a miss as a refusal.** A plan addressing a line
outside the served window would otherwise be refused by ADR-002's guard and land in the secondary
bucket, quietly removing the worst misses from the primary denominator. The fixture is recorded as
wholly observed before applying. **The read-before-write guard is not under test here**; converting
its refusals into data would corrupt the variable that is.

**5. A refused plan is a THIRD outcome.** It is excluded from the correct-address denominator and
reported separately, with its count. Folding a format failure into a localisation rate would bend the
primary curve with the secondary variable — and silence on the interaction is what makes a published
proportion unreadable.
The two refusal kinds are counted APART, because the pre-registration names `refused_parse` and
`refused_apply` as two secondary variables, and an outcome the scorer never produces is refused by
the tally rather than bucketed. **Cells key on the REQUESTED size, not the measured one.** Padding
is fitted to reach a size and overshoots by a seed-dependent amount, so five repeats of one 6,000-byte
cell served 6,011 to 6,040 bytes: keyed on the measurement they would have been five cells of one and
every interval the interval of a single observation. Found by review of #85.

**6. Each cell owns a fresh `XDG_STATE_HOME`.** Verified during pre-registration: with a fresh state
home per trial, `mrw stats` reports that trial alone. If a cell inherited one, the previous trial's
ledger would license reads the current one never made and both the secondary DVs and the seeding in
Decision 4 would go soft.

**7. The harness is a SEPARATE BINARY.** `mrw`'s subcommand surface is unchanged, so the agent-facing
guide gate stays true without documenting a benchmark to callers who will never run one.

**Go/no-go, checked during execution:**

- **No engine change.** `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`,
  `internal/check` and `internal/state` stay byte-identical against a merge-base diff. If the harness
  needs any of them to change, it has grown into the engine and must stop — that is the Stop
  Condition, not a widening.
- **No new dependency**; `go.mod` still declares exactly one requirement.
- **`internal/mcp` is not touched.** `MaxResultChars` is read, not moved.
- **`gofmt -l .` empty and `go vet ./...` clean**, in the fence.

## Alternatives Considered

- **Put a model client in the harness.** The direct route to a curve. Rejected: a second dependency,
  a network call inside `go test`, a cost per run, nondeterminism inside the tool rather than in the
  data, and a result that describes one client.
- **Score by comparing the address string to the planted line.** Simplest possible scorer. Rejected
  under Decision 3 — it re-implements pattern resolution and measures agreement with itself.
- **Score semantic correctness of the edit.** The question a reader will actually ask. Rejected by
  the pre-registration itself: unscoreable without a rubric and a rater, which is what makes a
  benchmark unrepeatable and unaffordable.
- **Reuse ADR-009's tally as the primary DV.** Free, already recorded. Rejected in the
  pre-registration and restated here: it will be flat for a reason that has nothing to do with the
  question.
- **Add `mrw bench` as a subcommand.** Rejected under Decision 7 — the benchmark is for whoever is
  measuring the tool, not for whoever is using it.
- **Wait for the reach record (ADR-019) first.** Rejected: they are independent, and the cap question
  is older.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/curve` | How a cell is generated and how a result is scored | Yes |
| `cmd/curve` | The three verbs, and nothing else | Wiring only |
| `internal/mcp` | Untouched — it still owns the cap this measures | Untouched |
| every engine package | Untouched — reused as libraries, byte-identical | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| a second binary, `curve`, with `generate`, `score` and `tally` | New, additive | whoever runs the benchmark |
| `mrw`'s surface | Unchanged | — |
| `MaxResultChars` | Unchanged | — |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the generator, the manifest shape, and the scorer | T1 | the first reading, which is a follow-up and not a task here | No |

## Implementation

One task. The harness, its self-test with known-wrong answers, and the contract row that drives the
built binary end to end.

**The first reading is deliberately NOT a task in this record.** It needs N trials across cells
against a real client, which is a budget decision and a scheduling decision rather than an
implementation one, and a record that shipped an unexecuted task would carry debt that measures
nothing. It is a Follow-up with its criterion already fixed by the pre-registration, which is the
part that had to be written first.

## Consequences

- **Positive:** the cap becomes answerable. Whichever way the curve goes, the answer is evidence.
- **Positive:** the null is publishable and was committed before the instrument existed, so a flat
  result cannot quietly become a search for a metric that bends.
- **Positive:** the harness is client-agnostic, so a second model is a second run rather than a
  second harness.
- **Negative:** it ships an instrument with no reading. The follow-up says so plainly rather than
  the record implying a measurement was taken.
- **Negative:** a second binary in a repository that has had exactly one. Contained by Decision 7 and
  by the harness importing the engine rather than touching it.
- **Neutral:** nothing a `mrw` user runs behaves differently.

## Out of Scope

- The first reading itself (deferred: Follow-ups below — its criterion is already pre-registered in docs/adr/BACKLOG.md)
- Changing `MaxResultChars` (permanent: boundary: ADR-011-T3 owns it; this record measures it and a change needs the reading)
- Distractor density as a second curve (permanent: boundary: named in the pre-registration — one curve answers one question)
- Multi-root or reach (permanent: boundary: ADR-019 owns it)
- Any model client, credential, or network call (permanent: boundary: Decision 1)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The scorer reports a perfect rate because it never sees a wrong answer | Med | **High** | The Enforced-by feeds it a plan addressing the WRONG line and requires a miss. A scorer tested only on correct plans is unfalsifiable |
| Results from one cell are scored against another cell's manifest | Med | **High** | Trial id and served bytes are echoed and checked; a mismatch is refused, not scored |
| The read-before-write guard converts the worst misses into refusals | High if unhandled | **High** | Decision 4 — the fixture is licensed whole before applying |
| A cell inherits a previous trial's ledger | Med | Med | Decision 6 — fresh `XDG_STATE_HOME` per cell, asserted |
| The planted target is findable by string matching, so the task measures matching | High if unhandled | **High** | Distractors are near-identical, differing only in the detail the instruction names — the threat the pre-registration names first |
| The harness starts needing engine changes | Low | Med | The go/no-go clauses stay in the fence and report it; the Stop Condition says to stop rather than widen |

## Rollback

Delete `internal/curve`, `cmd/curve` and contract §54. Nothing else imports them, and the engine is
byte-identical, so the revert is a deletion.

## Follow-ups

- [ ] **Take the first reading.** N trials per cell, cells fixed in advance, across at least two
      clients. The criterion is already pre-registered in `docs/adr/BACKLOG.md` and must not be
      re-derived: correct-address rate against served bytes, stratified by target position, with
      refusals reported separately, and **a flat curve accepted as the answer**.
- [ ] If the reading bends, the cap becomes a measured number and ADR-011-T3's value is revisited by
      the record that owns it.
