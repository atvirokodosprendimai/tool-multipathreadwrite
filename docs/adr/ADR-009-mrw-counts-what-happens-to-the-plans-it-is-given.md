# ADR-009: mrw counts what happens to the plans it is given

**Status:** Accepted
**Date:** 2026-09-03
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001 (owns the plan format this measures), ADR-004 (owns the state directory this writes into), ADR-003 (owns `--check`, which this deliberately does not change)
**Governs:** `internal/authoring/**`
**Enforced-by:** `internal/authoring/authoring_test.go::TestTheTallyNeverRecordsPlanContentOrPaths`
**Invalidates:** none — checked. Grepped every accepted record for `state`, `Path(`, `rung 4` and `measure`: ADR-004 owns where per-checkout state lives and this adds one more file under the directory it already defines, changing none of its rules. No record claims the authoring question.
**Served-path change:** `mrw write` records the OUTCOME of every plan it is handed, and `mrw stats` prints the tallies — the first number this project has about whether an agent can author the format the project is built on.

## Context

**Every task in this corpus leaves the top rung of its own ladder empty.** The task template's
Reachability table ends at `4 — it is used`, and across the 13 tasks of ADR-001..008 that cell reads
`nothing measures this yet` five times and `nothing counts them` six times. Enumerated 2026-09-03:

    grep -h '4 — it is used' docs/adr/*/tasks/T*.md | sort | uniq -c

Rung 4 is not a gate and the template says so — but eleven consecutive "nobody counts" is not eleven
independent judgements, it is a habit. This record closes it for the one question that matters most.

**`scripts/measure.sh` publishes numbers that are all conditional on an unmeasured variable.** It
reports 35.4× fewer input bytes and 2 calls for any N (measured 2026-09-03 at `87b43d4`, this
repository) — every one of them assuming the plan was authored correctly. `scripts/contract.sh`'s
271 assertions test what mrw does with a plan. **Nothing tests whether the thing meant to write one
can.**

That gap is exactly where the outside evidence says formats differ:

- **Diff-XYZ** (arXiv:2510.12487, JetBrains Research) benchmarks apply / anti-apply / diff-generation
  and finds format choice materially changes success rates, with smaller models benefiting little
  from structured formats.
- **aider's edit-format benchmarks** report the same shape: weaker models score better with
  whole-file than with diffs, which is why aider selects the format per model.
- mrw's plan format is bespoke. No model has it in training data, and this project has never
  measured whether one can emit it.

**And the cost of being wrong is not a lower score — it is a false report.** SilentProbe
(arXiv:2609.00035) measured agents against tools that fail silently: detected in 12% of cases,
**repaired in 0%**, and a false negative asserted to the user in 41%. Its conclusion is this
project's founding premise arriving from outside: *"the solution is schema improvement rather than
better models"*. mrw already refuses rather than lying. What it cannot yet say is how often it has
to.

> The numbers above are cited from preprints located 2026-09-03 and read at abstract level. They
> motivate the measurement; nothing in this record depends on their exact values.

## Existing Primitives Audit

- **`internal/state`** (`state.Path`, `state.Dir`): resolves a per-checkout directory outside the
  working tree and creates it. **Reused unchanged** — the tally is one more small file beside
  `seen` and `iteration`, and ADR-004's rules about where state lives are not touched.
- **`internal/seen`**: the pattern to copy. A small line-oriented file, loaded whole, rewritten
  whole, tolerant of a corrupt read by failing closed. **Reused as a shape, not as code** — the two
  hold different things and sharing a serialiser would couple them for no gain.
- **`internal/plan.Parse`**: already produces the classified errors this counts (`option %q is not
  key=value`, `unknown op %q`, `body= asked for…`, `given twice`, `text before the first @@ header`).
  **Reused unchanged** — the categories are derived at the call site from what Parse already
  returns; Parse itself learns nothing about counting.
- **`internal/iter`**: a second existing state file, and the precedent that a state file may back a
  subcommand (`mrw iter`). **Reused as precedent** for `mrw stats`.
- **A hosted analytics or telemetry SDK:** not audited as a candidate. Sending anything off the
  machine is refused in Out of Scope, so no dependency is weighed.

## Decision

`mrw write` records the outcome of every plan it is handed, into `<state-dir>/authoring`, and
`mrw stats` prints the tallies.

**What is recorded, exactly:** a count per outcome — `applied`, `refused_parse`, `refused_guard`,
`refused_unseen`, `refused_boundary`, `failed_check` — plus, for parse refusals only, a count per
error CATEGORY drawn from a closed vocabulary that `internal/authoring` owns. Counts and category
names. Nothing else.

**What is never recorded, and this is the boundary rather than a default:** no plan text, no file
paths, no anchors, no SHAs, no timestamps finer than a date, no command line. The tally must be
something a caller can read in full and find nothing of their work in. It lives in the same
per-checkout state directory as the ledger, is never transmitted, and `mrw stats --reset` empties it.

**The pre-registered reading, and what would make it fail.** The claim this measurement exists to
test is *an agent can author this format*. It fails if, over a corpus of real use, **parse refusals
exceed 5% of plans** — that is, if more than one plan in twenty is malformed at the document level
rather than wrong about the file. At that point the format is the problem and ADR-001's plan
document is the thing to revisit, not the caller.

Two honesties about that threshold. It is **valid for the population that produces the tally** — a
number from one repository driven by one model says nothing about another, and `mrw stats` prints
the sample size beside the rate for that reason. And **data that could produce the failure exists
today**: this repository's own development authors plans continuously, so a reading is obtainable
without waiting for adoption. A criterion nothing can falsify would be a formality; this one can be
falsified by next week's use.

**What this does not change:** every existing exit status, the plan format, the ledger, and the
decision to leave `--check` opt-in.

## Alternatives Considered

- **A live-model benchmark — call a model, ask for a plan, grade it.** The direct answer to the
  question, and rejected as the FIRST move: it needs an API key, costs money per run, cannot run in
  CI, and measures the model available on the day rather than the format. It also answers a
  laboratory question when a production one is available for less. Worth building later against a
  fixture corpus this tally would tell us how to assemble; recorded in Out of Scope as deferred.
- **A fixture corpus of recorded model output, graded hermetically.** Better than a live benchmark
  and still second best here: somebody has to collect the corpus, and the honest source of it is
  exactly the production signal this record adds. Build the tally first, then the corpus knows what
  to contain.
- **Instrument nothing and reason from `contract.sh`.** Free, and it is the status quo. Rejected
  because contract.sh asserts mrw's behaviour given a plan; the whole gap is the step before that.
- **Telemetry to a server.** Rejected on the same premise as ADR-004's refusal of `.git/mrw/`: this
  tool acquires no dependencies it can avoid, and a tool that phones home is a different product.
  The tally is local, and `mrw stats` is the only way to read it.
- **Record failing plans verbatim for later study.** The most useful data and the easiest to justify
  in the moment. Rejected because a plan body contains the caller's source code, and a file that
  accumulates their code under a name they did not choose is a liability the counts do not carry.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/authoring` (new) | The outcome vocabulary, the tally file's format, and the rule that it holds counts only | Yes — changes when the vocabulary changes |
| `internal/state` | Where per-checkout state lives | Unchanged — it gains a caller, not a responsibility |
| `cmd/mrw` | Recording an outcome after a write, and the `stats` subcommand | Yes — it already owns outcome-to-exit-status mapping |

`internal/authoring` is its own package rather than a function in `cmd/mrw` because the closed
vocabulary is the thing under test, and a vocabulary defined inside a command cannot be unit-tested
without the command.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw stats` | new subcommand | `cmd/mrw` | callers, this repository's own reporting |
| `mrw stats --json` | new flag | `cmd/mrw` | scripts, hooks |
| `mrw stats --reset` | new flag | `cmd/mrw` | callers |
| `<state-dir>/authoring` | new state file | `internal/authoring` | `internal/authoring` |
| `authoring.Outcome` (closed vocabulary), `authoring.Record`, `authoring.Load`, `authoring.Tally` | new internal API | `internal/authoring` | `cmd/mrw` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `authoring.Outcome`, `authoring.Record`, `authoring.Load`, `authoring.Tally` (T1) | T1 | T2 | No — new package, no existing caller |
| `mrw stats` output shape (T2) | T2 | T3 | No — new surface |

## Implementation

See `ADR-009-mrw-counts-what-happens-to-the-plans-it-is-given/tasks/README.md`. Three tasks: the
tally and its vocabulary, the subcommand that surfaces it, and the first published reading.

## Consequences

- **Positive:** rung 4 stops being empty for the question the project rests on, and it is answered
  from real use rather than a benchmark.
- **Positive:** `measure.sh`'s numbers gain the denominator they have always been missing — a saving
  per plan means less when some fraction of plans never applied.
- **Positive:** a parse-refusal category that dominates is a direct instruction about which part of
  the format to change, which is the actionable form of the Diff-XYZ finding.
- **Negative:** mrw writes one more file per checkout, and a file that exists is a file that can be
  corrupted, migrated or leaked. The mitigation is that it holds nothing worth leaking, which is a
  boundary the tests enforce rather than a promise.
- **Negative:** a tally is a temptation. Every future request to record "just the path" or "just the
  first line" is a request to cross the boundary above, and the record is here to be pointed at.
- **Neutral:** `mrw stats` on a fresh checkout prints zeros, which is the honest answer and reads
  like a broken feature. The output says so in words.

## Out of Scope

- Sending any tally, count or category off the machine (permanent: boundary: this tool acquires no dependencies it can avoid, and a tool that phones home is a different product — the same premise ADR-004 used to refuse a git dependency)
- Recording plan bodies, file paths, anchors or SHAs (permanent: boundary: a plan body contains the caller's source, and a file accumulating their code under a name they did not choose is a liability the counts do not carry)
- Making `--check` default-on (permanent: fact: mrw is paired with quality-harness from the QAM stack, which supplies the gate, so mrw must not force one; citation: file `README.md:120`)
- A live-model benchmark that asks a model to author a plan and grades it (deferred: docs/adr/BACKLOG.md)
- A fixture corpus of recorded model-authored plans (deferred: docs/adr/BACKLOG.md)
- Shortening the per-hunk receipt for large plans (deferred: docs/adr/BACKLOG.md)
- Counting anything about `mrw read` (permanent: boundary: the question is whether a plan can be AUTHORED; a read has no authoring step to fail at)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The tally becomes a place to put "just one path" | High over time | High — it is the whole boundary | The closed vocabulary is a type, not a string; `TestTheTallyNeverRecordsPlanContentOrPaths` reads the written file and fails on anything outside it |
| Concurrent writes lose counts, as the ledger does | Med | Low | Counts are advisory, not a guarantee; `mrw stats` says the number is a floor. The ledger's own concurrency note (README) already documents the shape |
| A corrupt tally breaks `mrw write` | Low | High | Fails OPEN: an unreadable tally is discarded and the write proceeds. The tally may never be able to fail a write — it is measurement, and measurement that can break the tool is worse than no measurement |
| The 5% threshold is measured on one repository and read as universal | Med | Med | `mrw stats` prints the sample size beside the rate, and the Decision states the criterion is valid for the population that produced it |
| Nobody ever runs `mrw stats` and the data rots unread | Med | Low | T3 publishes the first reading in the README, which is the one place this project's numbers have been read before |

## Rollback

Delete `internal/authoring`, the `stats` subcommand, and the single call site in `mrw write`. The
tally file is additive state in a directory ADR-004 already treats as disposable — an older binary
ignores an unknown file there, and a tree written by this version is served identically by the
previous one. No format, exit status or ledger entry changes, so nothing migrates.

## Follow-ups

- [ ] Publish a second reading once the tally has a larger sample, and say whether the 5% criterion held
