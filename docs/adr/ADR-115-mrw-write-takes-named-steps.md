# ADR-115: mrw_write takes named steps

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — "Named steps only (Recommended)", the answer to "Item 2 — steps on mrw_write. --then-sh runs arbitrary shell, which is why ADR-113 kept steps CLI-only. What should mrw_write take?", on the refreshed gap list of 2026-10-02 and the plan he approved the same day. The record's text was drafted after that answer and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-02
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-092, ADR-093, ADR-095, ADR-102, ADR-111, ADR-113
**Invalidates:** ADR-113 — its Decision 6 ("Steps stay CLI-only") and its Out of Scope entry deferring steps over MCP
**Governs:** `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/schema.go`, `docs/receipts.txt`, `AGENTS.md`, `README.md`, `cmd/opencode/mrw-plugin/src/index.ts`, `scripts/contract.sh`
**Enforced-by:** `internal/mcp/steps115_test.go::TestAnMCPWriteRunsItsSteps`
**Served-path change:** `mrw_write` takes `then`: a list of step names declared in `.quality-harness.json` "steps", run in order after a write that landed and whose check, when one ran, passed — the receipt carries `then`, as `mrw write --json` does. No ad-hoc shell: `--then-sh` stays CLI-only.

## Context

ADR-113 gave `mrw_write` the project check and left steps on the CLI (Decision 6), because `--then-sh` runs any shell command and a named step needed its own floor and elision. The refreshed gap list of 2026-10-02 names that as the sharp MCP gap: "the check moved and the steps did not". The shared sequence already runs steps (`writer.Prepare` resolves them, `Landed.Verify` runs them, `Settle` counts a stopped one); only the MCP surface passes none.

**Audit of the class** — *a step decision the CLI makes that MCP must make the same way*: `mrw read --grep 'askedStepsError|ResolveSteps|RunSteps|StepStop|stepsExit|DepthRefusal\("--then' cmd/mrw/main.go internal/writer/flow.go`. The resolution, run, and count are shared (flow.go); the depth refusal for asked steps (`askedStepsError`, cmd/mrw) and the exit mapping (`stepsExit`) are CLI-side and get their MCP counterparts here.

## Existing Primitives Audit

- **`writer.Request.Steps`, `Prepare`, `Verify`, `Settle`** (ADR-113) — resolve, run and count steps.
- **`check.DepthRefusal`** (ADR-095) — the depth limit for a step that re-runs mrw.
- **`keptTail`, `checkPhrase`, `floorAt`** (ADR-113, `internal/mcp/tools.go`) — the receipt's tail elision and the bounded terminal sentence.

## Decision

1. **`then` is a list of declared step names** (strings). Each resolves against `.quality-harness.json` "steps"; a name not declared refuses the write before anything lands, naming the declared ones. No ad-hoc command: an MCP caller cannot run arbitrary shell through mrw.
2. **The depth limit refuses first.** With `then` given, `MRW_STEP_DEPTH` at its limit refuses the call before the plan is parsed, as `--then` is refused on the CLI, and the refusal names `then`.
3. **The receipt carries `then`** (the CLI's `check.StepsResult`): every step's verdict, `not_run` for each after a stop, and every one `not_run` when the check did not pass or the ledger could not record the landing.
4. **A step that ran and did not pass is data, not `isError`** — the write landed, as for a failed check (ADR-113 Decision 4). A step that could not start is `isError`, with the receipt — `stepsExit`'s exit 2.
5. **Elision and the floor.** A step's tail goes with the check's tail, first; a passing step whose log was removed keeps its tail in a log of its own. The terminal sentence names a stopped step in a fixed phrase, never its name, and `writeFloor` carries the longest.

## Alternatives Considered

- **Named and ad-hoc steps** — rejected by Zy's answer: `then_sh` would let any MCP caller run any command in the checkout.
- **Keep steps CLI-only** — rejected by Zy's answer.

## Component / Boundary Impact

`internal/mcp` and the opencode plugin. No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_write` input schema | `then` (array of step names) | T1 | MCP callers |
| `mrw_write` structured value | `then` and its keys | T1 | MCP callers |
| `docs/receipts.txt` | `mcp_write then…` | T1 | ADR-111's tests |
| descriptions, AGENTS.md, README, plugin | say so | T1 | readers |
| `scripts/contract.sh` | §215 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** an MCP caller verifies a write with the project's own steps in the same call.
- **Negative:** a step-bearing MCP write holds the server for the steps' duration too.
- **Neutral:** the CLI is unchanged.

## Out of Scope

- Ad-hoc shell steps over MCP (permanent: boundary: the project, not the caller, decides what runs)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a long step outlives the host's tool timeout | Low | Medium | ADR-113's note on host clocks applies; the description names it |

## Rollback

Revert T1. The `then` receipt keys disappear with it, which ADR-111 would call a removal: revert before a release ships them.

## Follow-ups

- None — the record carries no open follow-up.
