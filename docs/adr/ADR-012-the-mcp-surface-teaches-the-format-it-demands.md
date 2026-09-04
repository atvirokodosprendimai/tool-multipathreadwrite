# ADR-012: The MCP surface teaches the format it demands

**Status:** Accepted
**Date:** 2026-09-04
**Accepted:** 2026-09-04 by M, in the instruction that opened the work — *"lets do this new adr and start implementing it"* — answering the five gaps put to M in the session that found them. Recorded here rather than inferred, because ADR-011 had to add that sentence retroactively when a reviewer asked whose decision the status was.
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-011 (the declared surface this extends), ADR-010 (the transport both sit on), ADR-001 (owns the plan grammar this teaches), ADR-002 (owns the read-before-write guard this warns about), ADR-009 (the tally whose criterion this record deliberately does NOT borrow — see Context)
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/conformance_test.go::TestEveryEmbeddedExamplePlanReallyApplies`
**Invalidates:** none — checked. ADR-011 governs the same directory and is **extended, not superseded**: every field it declares stays declared, and none of its three permanent Out of Scope boundaries (`requiresUserInteraction`, `listChanged`, resources/prompts/sampling) is touched. ADR-001 owns the grammar; this record documents it and changes nothing about it.
**Served-path change:** an MCP host learns from the wire — not from a file it cannot see — when to reach for mrw over its own editor, how to author a plan, and what every field of the answer means.

## Context

ADR-011 made the tool surface honest about **what the tools are**: which one writes, what shape comes
back, how large it may get. It said nothing about **how to drive them**, and over MCP that is the
whole of the caller's education. Five gaps, all readable from one `tools/list` call against the
shipped binary:

**1. `initialize` carries no `instructions`.** The 2025-06-18 lifecycle response has a top-level
optional `instructions` field for exactly this — the server telling a host how it is meant to be
driven — and ours returns `protocolVersion`, `serverInfo` and `capabilities` and stops. A host that
supports the field puts it where the model will see it before the first call.

**2. `plan` gets 60 characters for a bespoke format.** The whole documentation an MCP-only caller has
for the format this project is built on is: *"The plan text: `@@ <path> <addr> <op> [guards]`
followed by body lines."* That is what the placeholders are called, not what goes in them. In this
repository, AGENTS.md §"Using mrw" carries the grammar, the guard rules and a worked plan — and
AGENTS.md is a file in a checkout, invisible to a host driving `mrw mcp` in someone else's tree.
ADR-009 exists on the premise that no model has this format in training data; the description is the
only place that premise gets answered over MCP.

**3. No examples.** JSON Schema has an `examples` keyword and neither input schema uses it. For a
format that must be produced exactly, one plan that actually applies is worth more than a paragraph
about its shape.

**4. `mrw_write`'s output schema describes six properties and says nothing about any of them.** It is
generated from `apply.Result` (ADR-011's decision, and the right one), and the generator emits types
without descriptions. `mrw_read`'s two top-level properties have descriptions only because
`readSchema()` is hand-written around the generated part. So the machine-readable half of the
contract declares `applied`, `failed`, `dry_run` and does not say what any of them means.

**5. The descriptions say what a tool does, not when to reach for it.** Over MCP, mrw competes with
the host's own Edit and Write, and the description is the entire pitch. AGENTS.md:77 states the
threshold — *"3 or more edits, 2 or more files, or several ranges you need to read"* — along with the
counter-advice that a single edit costs more through mrw than through an ordinary editor. None of it
reaches a host.

### The measurement this record does NOT make, and why

The session that found these gaps proposed ADR-009's tally as a pre-registered instrument: change the
descriptions, watch `refused_parse`. **That criterion cannot produce data, and writing it down as if
it could would be worse than not measuring at all.** Three reasons, the third decisive. The first
two are weaknesses of sensitivity; only the third makes the instrument the wrong one, and this
paragraph was rewritten during review because its earlier form overstated the first two into
impossibility claims the arithmetic does not support.

- **The baseline is already inside the threshold.** `refused_parse` sits at 1.5% against a proposed
  5%. That is not literally unfalsifiable — a cumulative rate CAN rise past 5% if enough later plans
  are refused — but a criterion the tool satisfies before the intervention tests the wrong
  direction: it can only fail if the change makes authoring *worse*, and it registers nothing if the
  change helps. It is one step from the defect this corpus has produced four times, a clause that
  was green the day it was written.
- **One event, not one observation.** 68 outcomes are recorded and exactly ONE of them is a
  `refused_parse`; 5% of 68 is 3.4 events. The sample is 68 and the event count is 1, so the
  interval around 1.5% is wide and the criterion is insensitive to any change smaller than the noise
  in a handful of refusals. Insensitive, not powerless — the earlier wording said "no power to
  detect a change of any size", which is false and is corrected here rather than quietly dropped.
- **THE DECISIVE ONE — the tally cannot attribute, and cannot see the population.** `cmd/mrw` and
  `internal/mcp` write the same counters, so an outcome cannot be assigned to the MCP descriptions
  this ADR changes. And the caller worth measuring is one who has *only* the description: in another
  checkout on another machine, where ADR-009's Out of Scope permanently refuses transmission — the
  tally is per-checkout and `mrw stats` is its only reader. Splitting by source fixes attribution and
  still leaves every caller here one who has read AGENTS.md. No amount of sensitivity repairs an
  instrument pointed at the wrong population.

So this record is justified by the contract, not by an experiment: each gap is a thing a host is
entitled to be told and is not. Uptake goes in Reachability rung 4 as *not measured and will not be*,
for the same reason ADR-011-T2 gave — counting which fields a host read needs telemetry, and ADR-009
refused telemetry on the premise that this tool does not phone home. The tally split is deferred to
`BACKLOG.md` with the above as its stated reason, so the next session does not re-propose it.

## Existing Primitives Audit

- **`internal/mcp/mcp.go` `tools()` and `initializeResult()` (ADR-010, ADR-011):** the descriptor set
  and the handshake. **Extended, not replaced.** Every field ADR-011 declares stays; this adds
  `instructions`, rewrites two `description` strings and adds `examples` to two input properties.
- **`internal/mcp/schema.go` `SchemaOf` (ADR-011):** the reflection generator. **Reused unchanged as
  a generator**, and given a description table to consult. It is not replaced by hand-written
  schemas — that is exactly the failure ADR-011 measured in a peer's server.
- **Struct tags on `apply.Result` / `seen.Observation`:** audited and **NOT taken.** `internal/apply`
  and `internal/seen` are two of the six engine directories every ADR-010 and ADR-011 fence asserts
  byte-identical, and that boundary is marked permanent. A `desc:"…"` tag is inert at runtime and
  still changes those bytes. The descriptions live in `internal/mcp`, which ADR-011's own boundary
  table already gives ownership of "the tool descriptors". The obvious objection — that a table
  drifts from the struct — is answered harder than tags would answer it: the coverage test fails when
  a field is **added**, which a tag on the old fields would not.
- **`internal/plan.Parse` and `internal/apply` in `--dry-run`:** **reused as the example's judge.**
  The embedded example is not compared against a golden string; it is parsed and dry-run applied
  against a temporary tree, which is ADR-011's "validate a REAL response" applied to documentation.
- **AGENTS.md §"Using mrw":** **reused as the source** of the trigger thresholds and the grammar.
  Duplicating them into a Go string is deliberate — a host cannot read AGENTS.md — and the duplication
  is asserted rather than trusted: §43 greps the same threshold sentence out of both.
- **An MCP SDK, or a JSON Schema library:** still not taken, on ADR-010's and ADR-011-T2's arithmetic.
  `examples` is a slice literal and `instructions` is a string.

## Decision

**1. `initialize` returns `instructions`.** A short document telling a host what mrw is for, when to
reach for it instead of an ordinary editor, the plan grammar with one worked plan, and the two rules
that produce most refusals — read-before-write is per LINE, and a plan is all-or-nothing. It is
written for a caller with no access to this repository.

**2. Both tool descriptions lead with the trigger.** What the tool is *for* and when it beats the
host's own editor come first; what it does comes second. The threshold is AGENTS.md's, quoted.

**3. Both input schemas carry `examples`.** `plan` carries a complete two-hunk plan across two files;
`specs` carries a spec list showing a line range, a regexp address and `$`. The plan example is
parsed and dry-run applied by a test, so it cannot rot into documentation that describes a format the
binary no longer accepts.

**4. Every property of every declared output schema carries a non-empty `description`**, sourced from
a table in `internal/mcp` and applied to the generated schema. Coverage is total and enforced as
total: a property with no description fails, and a description for a property that no longer exists
fails too — a stale entry is how a table starts describing a shape nobody sends.

**Go/no-go, checked during execution and recorded in the task verification logs. If any fails, the
task is withdrawn rather than shipped:**

- **No engine change.** `git status --porcelain --untracked-files=all` and a merge-base `git diff`
  over `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and
  `internal/state` are both empty. Both forms, for ADR-010's reason: each sees what the other misses.
- **No new dependency.** `go.mod` still declares exactly one requirement.
- **Every embedded example is executed, not read.** The plan example is parsed and dry-run applied;
  a test that only asserts the string is present would pass on a plan that has not been valid for
  months.
- **Description coverage is total in both directions.** Every declared property described; every
  described property declared.
- **Nothing ADR-011 declared is dropped.** Its conformance tests stay green unmodified.

## Alternatives Considered

- **Point the description at AGENTS.md.** Free, and wrong for the only caller that matters: a host
  driving `mrw mcp` in a tree it did not clone from here has no AGENTS.md, and a reference to a file
  the reader cannot open is worse than silence because it reads as help.
- **Put the format documentation in `instructions` only, and leave the descriptions short.** Rejected:
  `instructions` is optional and host support is uneven, and a host that ignores it would be left with
  today's 60 characters. The tool description is the one surface every MCP client reads.
- **Put it in the descriptions only, and skip `instructions`.** Rejected on the other side of the same
  argument: the trigger guidance — *when* to reach for this over the built-in editor — is a statement
  about the server as a whole, and repeating it in both tools is where the two copies start to
  disagree. `instructions` is the field the spec provides for exactly that.
- **Struct tags on `apply.Result` for the field descriptions.** Rejected on the engine-boundary
  argument in the audit above, and on the weaker coverage it would give.
- **Hand-write the output schemas now that they need prose.** Rejected: this is precisely how a peer's
  schema ended up attached to the wrong tool. Generation plus a description table keeps the shape
  derived and only the prose authored.
- **MCP `prompts` or `resources` carrying the format guide.** Rejected: restated from ADR-010 and
  ADR-011 — the tools ARE the product, and a caller who must fetch a resource before writing a plan
  has been handed a second round trip in place of a sentence.
- **Pre-register `refused_parse` against the ADR-009 tally.** Rejected on the three grounds in
  Context. Recorded as an alternative rather than omitted, because it is the obvious idea and the
  reasons it fails are not obvious.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | The wire protocol, the tool descriptors, the result envelope, **and now the prose a host is taught by** | Yes — changes when the declared surface or what it says about itself changes |
| `internal/mcp/schema.go` | Shape generation, plus applying a description table to it. Still knows reflection, not the tools | Yes |
| `internal/apply`, `internal/seen` | Unchanged, and deliberately: the prose about their fields lives with the descriptors, not with the types | Untouched |
| `AGENTS.md` | Still the source of the trigger thresholds; §43 asserts the copy in the binary matches it | Yes |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| `initialize` result gains top-level `instructions` | Public contract, additive | Any MCP host |
| Both tool `description` strings are rewritten trigger-first | Public contract, content | Any MCP host |
| `plan` and `specs` input properties gain `examples` | Public contract, additive | Any MCP host |
| Every output-schema property gains `description` | Public contract, additive | Any MCP host reading `outputSchema` |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| `mcp.instructionsText`, `mcp.examplePlan`, `mcp.exampleReadSpecs` | T1 | — | No — new |
| `mcp.describeResult(schema, table)` and the two description tables | T2 | — | No — new |

## Implementation

Two tasks. T1 is the prose a caller reads before authoring anything; T2 is the prose attached to what
comes back. They touch different functions and could run in either order; T1 is first because it is
the gap a caller meets before making a call at all.

## Consequences

- **Positive:** a host driving mrw in a tree with no AGENTS.md is told the format, the two rules that
  produce most refusals, and when the tool is not worth reaching for.
- **Positive:** the worked plan cannot rot — a change to the grammar turns the example red.
- **Positive:** the six output properties stop being an untyped promise.
- **Negative:** `tools/list` grows. Roughly 2 KB of prose sent once per session, against a per-tool
  result ceiling of 200,000 characters; the trade is paid once and read on every call.
- **Negative:** the trigger thresholds now exist in two places. Deliberate, and asserted by §43
  rather than trusted.
- **Neutral:** a host that ignores `instructions` loses nothing it had — the tool descriptions carry
  the format on their own.

## Out of Scope

- Splitting the authoring tally by call source, CLI versus MCP (deferred: docs/adr/BACKLOG.md — it partitions the population we have and does not create the description-only one we would want to measure)
- Any pre-registered criterion over `refused_parse` for this change (permanent: boundary: the three reasons in Context; the instrument cannot see the population)
- MCP `prompts` or `resources` as a place to publish the format guide (permanent: boundary: restated from ADR-010 and ADR-011 — the tools ARE the product)
- Telemetry of any kind about which fields a host read (permanent: boundary: ADR-009 refused transmission; nothing here reopens it)
- Changing the plan grammar itself (permanent: boundary: ADR-001 owns it; this record documents it and must not quietly amend it)
- Changing any CLI behaviour, or any byte in the six engine directories (permanent: boundary: ADR-010's go/no-go, still binding)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The embedded example plan stops being valid and nobody notices | Med | High — it is the one thing a caller copies | The Enforced-by test parses it and dry-run applies it against a temp tree; a grammar change turns it red |
| The description table drifts from `apply.Result` | Med | Med | Coverage asserted in BOTH directions: an undescribed property fails, and a described property that no longer exists fails |
| The trigger thresholds in the description drift from AGENTS.md | Med | Low | §43 greps the same sentence out of AGENTS.md and out of the running binary's `tools/list` |
| `instructions` is long enough to cost more than it teaches | Med | Low | It points rather than reproduces where it can, and the length is bounded by an assertion in §43 |
| A host rejects an unknown top-level field in the initialize result | Low | Med | It is a spec field of the revision this server declares (2025-06-18 lifecycle), not an extension |

## Rollback

Revert the commits. Every change is additive prose: `instructions` disappears from the handshake,
the descriptions shorten, `examples` and property descriptions vanish. No wire shape changes, no
state format moves, no ledger entry changes meaning, and a tree touched under this ADR is served
identically by the previous binary.

## Follow-ups

- [ ] If a second MCP host is ever driven against this server, check whether it surfaces
      `instructions` at all — the field is optional and support is uneven, and that is the one fact
      here that a single host cannot establish
