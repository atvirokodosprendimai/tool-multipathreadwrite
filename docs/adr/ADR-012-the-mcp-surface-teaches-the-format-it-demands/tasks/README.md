# ADR-012 Tasks

Implementation tasks for ADR-012: the MCP surface teaches the format it demands. See the parent ADR
for the decision, and in particular for the criterion it deliberately does **not** adopt.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers. This
README is a derived index — when it disagrees with a task file, the task file wins and the README
must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

Chosen, not forced: neither task produces anything the other consumes, and they touch different
functions — T1 lives in `initializeResult()` and `tools()`, T2 in `schema.go` and the two schema
builders. T1 goes first because it is the gap a caller meets *before* making a call at all: without
it there is nothing to author a plan from. If they are worked in parallel, expect one conflict in
`scripts/contract.sh` where §43 and §44 land side by side.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Teach the format on the wire, and show a plan that really applies | pending | — | 3 named `--- PASS:` lines, `# 43.` in `contract.sh`, both engine clauses, `./scripts/contract.sh` |
| T2 | Say what every field of the answer means, and keep saying it | pending | — | 2 named `--- PASS:` lines, `# 44.` in `contract.sh`, both engine clauses, `./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `instructionsText`, `examplePlan`, `exampleReadSpecs` | — | nothing in this ADR consumes them; they are read off the wire |
| T2 | `describeResult` and the description tables | — | same |

## Notes

- **Run every clause of every fence separately before writing a line of its task**, and record the
  counts. Both task files do. The tokens that were *rejected* matter as much as the ones chosen:
  `instructions` appears throughout the ADR corpus, and `description` already appears in both input
  schemas and in `readSchema()` — either would have been green the day it was written, which is the
  `outputSchema` near-miss ADR-011-T2 documented and the fifth of its kind in this repository.
- **§43 and §44 were confirmed free.** The highest section in `contract.sh` is 42, from ADR-011-T3.
- **The go/no-go conditions are in every fence, in both forms.** `git status --porcelain
  --untracked-files=all` sees an untracked new engine file; a merge-base `git diff` sees a committed
  change; neither sees what the other does. Neither task should come near them — the whole ADR is
  prose in `internal/mcp` — and a fence that goes red on an engine clause here means something went
  badly sideways, so it is withdrawn rather than argued with.
- **ADR-011 is extended, not superseded.** Its conformance tests must stay green *unmodified*. A
  change to `TestEveryDeclaredOutputSchemaValidatesARealResponse` to accommodate this work is a
  signal that the shape moved, which this ADR promised it would not.
