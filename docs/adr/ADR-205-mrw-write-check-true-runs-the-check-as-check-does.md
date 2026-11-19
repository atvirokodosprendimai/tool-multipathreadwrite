# ADR-205: mrw_write's check: true runs the check, as --check does

**Status:** Proposed
**Drafted for:** Zy, 2026-11-19. A design review of v1.92.0 (Codex gpt-6.2-astra at xhigh, read-only on `ca15bec8`) found that an explicit check means two things on the two surfaces. Zy said of the review *"fair review, i think we need to address astra findings"* and, to the proposal to start with this record and the measurement, *"yes, start with 1 and 2"*. That is an instruction to draft this; the record is not accepted until Zy reads it.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-054, ADR-111, ADR-113, ADR-115, ADR-187
**Supersedes:** in part: ADR-187's rule for `mrw_write` ("runs none unless `check: true`, which runs it by ADR-054's rule") and the sentence of ADR-113 it rests on, for the one case below. ADR-187's default, no check unless asked, stands on both surfaces.
**Governs:** `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/schema.go`, `internal/writer/flow.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/mcp/check205_test.go::TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan`
**Served-path change:** `mrw_write` with `check: true` runs the project's check after any write that lands, a prose-only one included, and reports a tree with no check command as a check that could not run. Today it runs the check only when the write touches a file that is not prose and a check is declared or inferred, and otherwise returns no `check` at all.

## Context

**Two meanings of one word.** `mrw write --check` selects `CheckDemand` (`cmd/mrw/main.go`): the check runs on any plan, and a tree with no command is reported as a check that could not run, exit 2. `mrw_write` with `check: true` selects `CheckAuto` (`internal/mcp/tools.go`): ADR-054's older rule, under which a write that touches only prose runs nothing, and a tree with no command runs nothing, and in both cases the receipt simply has no `check`. A caller that asked for verification and reads a receipt with no failure in it has no way to tell "passed" from "never ran" except by noticing an absent key.

**How it came about.** ADR-187 made the CLI's check opt-in and, to avoid changing what MCP callers saw in the same release, left `check: true` on ADR-054's rule. It is written down (`AGENTS.md`, `README.md`, the tool's description), so it is a decision, not a slip. The design review's point is about the caller: one ported from the CLI can report verification that did not happen.

**What the code offers.** `writer.CheckDemand` is already what a surface asks for when the caller demanded the check. `mrw_write` is the only user of `CheckAuto` (`git grep -n CheckAuto -- cmd internal ':!*_test.go'` at `ca15bec8`: its definition and that one call).

**Audit of the class**: *the ways a caller asks mrw for the project's check.* `mrw write --check` (demand), `mrw check` (demand, no write), `mrw_write` `check: true` (auto: decided here), and `--then`/`then` steps, which run after a passing check and are unchanged.

## Existing Primitives Audit

- `writer.Prepare` and `Land` take the mode from the surface; `wanted` returns true for `CheckDemand` whatever the files are.
- The receipt's `check` object already has a shape for a check that could not run (`ran: false`, `exit_code: -1`, ADR-100, ADR-101), which `mrw check --json` and `mrw write --check --json` use.
- A check that ran and failed is not `isError` over MCP: the write applied, and an error would invite the caller to send the plan again (ADR-113).

## Decision

1. **`check: true` means the check runs.** `mrw_write` selects `CheckDemand`. After a write that lands, the project's check runs whatever the files are, by the same scoping rules the CLI uses.
2. **No command is said, not hidden.** In a tree that declares and infers no check, `check: true` returns a `check` object with `ran: false` and the reason, as `mrw write --check --json` does. The call is not `isError`: the write applied.
3. **Everything else about `mrw_write` is unchanged.** No check without `check: true` (ADR-187). A failed check is not `isError`. `then` runs after a passing check. A dry run runs none.
4. **`CheckAuto` goes.** It has no other caller; the mode and the branch of `wanted` that only it reached are removed.
5. **What would make this decision wrong.** A host that sends `check: true` on every write, prose included, and now pays for a check it did not want. The tool's description says what `check: true` does, and such a host sets it per call; BACKLOG.md records the case to look for in the first field reports.

## Alternatives Considered

- **Leave it, documented.** What ADR-187 chose. Rejected now: the cost falls on the caller who trusts a receipt, which is the failure this tool exists to prevent.
- **A third value, `check: "always"`.** Rejected: two spellings of "run the check" is the same problem with a longer name.
- **Make an absent `check` an explicit `ran: false` and keep the auto rule.** Rejected: it tells the caller the check was skipped, and still skips a check the caller asked for.

## Component / Boundary Impact

`internal/mcp` (the mode it selects, the tool's description, the receipt key's description) and `internal/writer/flow.go` (the removed mode). `internal/writer` is not one of the engine packages; no engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_write` `check: true` | runs on any landed write; reports a tree with no command | T1 | every MCP caller |
| the tool description and `check`'s schema description | say so | T1 | hosts that read them |
| `AGENTS.md`, `README.md` | the two sentences that describe the old rule | T1 | every caller |
| `scripts/contract.sh` §351 | a prose-only `mrw_write` with `check: true` runs the check; the same without it runs none | T1 | the contract gate |

No receipt key is added or renamed (ADR-111): `check` is present in more cases than before.

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** a requested check ran, or the receipt says it could not.
- **Negative:** an MCP caller that set `check: true` on prose-only writes now waits for a check; a breaking change for a caller that relied on the skip.
- **Neutral:** the CLI is unchanged.

## Out of Scope

- The acknowledgement protocol and paging of `mrw_read` (deferred: docs/adr/BACKLOG.md)
- Making a failed check `isError` (permanent: boundary: the write applied, and an error invites a second application, ADR-113)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A host relies on prose writes skipping the check | Low | Med | named in the release note as a behaviour change for MCP callers; the description is explicit |
| Removing `CheckAuto` breaks a test that names it | Med | Low | those tests are about the MCP surface and move with it; listed in the pull request |

## Rollback

Revert T1; no state is involved.

## Follow-ups

- [ ] First field reports after the release: any host sending `check: true` on prose-only writes.
