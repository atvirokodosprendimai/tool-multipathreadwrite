# ADR-205: mrw_write's check: true runs the check, as --check does

**Status:** Accepted
**Accepted by:** Zy, 2026-11-19: *"accept"*, to the question "accept ADR-205 as amended?" put with the four things an MCP caller that sends `check: true` will see (a wait on prose writes, a refusal before the write at step depth 8, `then` steps that wait for the check, reply order and counts). Drafted the same day from a design review of v1.92.0 (Codex gpt-6.2-astra at xhigh, read-only on `ca15bec8`), of which Zy said *"fair review, i think we need to address astra findings"* and then *"yes, start with 1 and 2"*.
**Date:** 2026-11-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-054, ADR-056, ADR-089, ADR-095, ADR-098, ADR-111, ADR-113, ADR-115, ADR-121, ADR-128, ADR-187
**Supersedes:** in part: ADR-187's rule for `mrw_write` ("runs none unless `check: true`, which runs it by ADR-054's rule") ADR-113 Decision 3's "there is no MCP twin of `--check`", and the parenthesis that ends its Decision 4 ("a check demanded with no command cannot happen here"). ADR-187's default (no check unless asked) and ADR-113 Decision 4 (a failed check is not `isError`; one that could not run is) stand.
**Governs:** `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/schema.go`, `internal/mcp/instructions.go`, `internal/writer/flow.go`, `cmd/opencode/mrw-plugin/src/index.ts`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/mcp/check205_test.go::TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan`
**Served-path change:** `mrw_write` with `check: true` runs the project's check after any write that lands, a prose-only one included, and a tree with no check command answers as any check that could not run does today: the applied receipt, `check.ran: false`, `isError`. Today it runs the check only when the write touches a file that is not prose and a check is declared or inferred, and otherwise returns no `check` at all. Three things a caller could rely on change with it (Decision 3): a refusal before the write at step depth 8, `then` steps that no longer run when the check does not pass, and the order of replies.

## Context

**Two meanings of one word.** `mrw write --check` selects `CheckDemand` (`cmd/mrw/main.go`): the check runs on any plan, and a tree with no command is reported as a check that could not run, exit 2. `mrw_write` with `check: true` selects `CheckAuto` (`internal/mcp/tools.go`): ADR-054's older rule, under which a write that touches only prose runs nothing, and a tree with no command runs nothing, and in both cases the receipt simply has no `check`. A caller that asked for verification and reads a receipt with no failure in it has no way to tell "passed" from "never ran" except by noticing an absent key.

**How it came about.** ADR-187 made the CLI's check opt-in and, to avoid changing what MCP callers saw in the same release, left `check: true` on ADR-054's rule. It is written down (`AGENTS.md`, `README.md`, the tool's description), so it is a decision, not a slip. The design review's point is about the caller: one ported from the CLI can report verification that did not happen.

**Challenged before acceptance** (Codex gpt-6.2-sol at xhigh, 2026-11-19, on `d437abe6`: AMEND, eleven findings, each checked against the code). The first draft said a commandless tree would not be `isError`, which reverses ADR-113 Decision 4 and what `internal/mcp/tools.go` renders today, and said "everything else is unchanged", which is false in the three ways Decision 3 now names. The in-repo challenger did not finish (its session lost its login); it is run on this text before Zy is asked.

**What the current rule already does.** Auto reads the harness before it decides, so a malformed `.quality-harness.json` already refuses a `check: true` write, prose or not. And a declared `scoped_check` does not guarantee a process: a `{packages}`-only template that places no edited path falls back to the whole check, and with none declared the answer is `ran: false` even for a code write. Demand changes when a check is *due*; the scoping and start-up rules still decide whether a process runs.

**What the code offers.** `writer.CheckDemand` is already what a surface asks for when the caller demanded the check. `mrw_write` is the only caller that names `CheckAuto`; but it is also the zero value, so the audit is of every `Request` that sets no mode: `git grep -nE 'Request\{' -- internal cmd | grep -v 'Check:'` at `295083ed` finds eight test sites in four files of `internal/writer` (`flow113_test.go`, `writes127_test.go`, `partial164_test.go`, `stale200_test.go`) and `cmd/mrw/main.go`, whose literal sets it on a later line. Nothing switches on the mode or stores it.

**Audit of the class**: *the ways a caller asks mrw for the project's check.* `mrw write --check` (demand), `mrw check` (demand, no write), `mrw_write` `check: true` (auto: decided here), and `--then`/`then` steps, which run after a passing check and are unchanged.

## Existing Primitives Audit

- `writer.Prepare` and `Land` take the mode from the surface; `wanted` returns true for `CheckDemand` whatever the files are.
- The receipt's `check` object already has a shape for a check that could not run (`ran: false`, `exit_code: -1`, ADR-100, ADR-101), which `mrw check --json` and `mrw write --check --json` use.
- A check that ran and failed is not `isError` over MCP: the write applied, and an error would invite the caller to send the plan again (ADR-113).

## Decision

1. **`check: true` means a check is due.** `mrw_write` selects `CheckDemand`. After a write that lands, the project's check is due whatever the files are, and the existing scoping rules pick the command. This holds for every way `mrw_write` gets a plan: `plan`, a compiled `format`, `grep` with `sub`, `ast_grep` with `rewrite`.
2. **A check that could not run answers as it does today.** No command in the tree, a shell that could not start, an interruption before the start: the applied receipt with `check.ran: false`, `exit_code: -1` and `skipped`, `isError`, and the text `check DID NOT RUN: … — the write applied; the tree is changed and unverified`. When the check's log could not be made the answer is `isError` with `error` set and no `check`. All of it is ADR-113 Decision 4's and none of it is new; what changes is that a prose-only write and a commandless tree now reach these answers, where they returned no `check`.
3. **What else changes for a caller that sends `check: true`; the release note names each.**
   - *A refusal before the write.* At `MRW_STEP_DEPTH` 8 a due check refuses the call before anything is written (ADR-095). A prose-only write, or a write in a commandless tree, lands there today and is refused after this.
   - *Steps wait for the check.* `then` steps run after a passing check. With `check: true`, a check that fails or cannot run leaves them not run, including in a tree that declares steps and no check; a prose-only write that ran its steps alone today runs the check first. A caller that wants the steps alone omits `check`.
   - *Reply order.* A call whose check is due releases the server's loop (ADR-121), so answers to later requests can arrive before it, now also for a prose-only write.
   - *What a prose write is counted as.* In `mrw stats` it moves from `applied` to `failed_check` or `check_not_run` when its check does not pass, and into the strict-balance pricing (ADR-056); its receipt can carry `drift` and `drift_writers`; and a cancel stops its check (ADR-128).
4. **Unchanged.** No check without `check: true` (ADR-187). A check that ran and failed is not `isError`. A dry run validates, resolves named steps, and runs neither check nor step.
5. **`CheckAuto` goes, and the zero value is off.** It has no caller outside `mrw_write` (`cmd/mrw/main.go` and `cmd/mrw/skill168.go` set the mode). `CheckOff` becomes the zero value of `CheckMode`, pinned by a test, so a `Request` that sets nothing runs nothing; the writer tests that relied on the old zero value ask for `CheckDemand`.
6. **What would make this decision wrong.** A host that sends `check: true` on every write, prose included, and now pays for a check, loses a step, or is refused at depth. The tool's description says what `check: true` does; BACKLOG.md records the case to look for in the first field reports.

## Alternatives Considered

- **Leave it, documented.** What ADR-187 chose. Rejected now: the cost falls on the caller who trusts a receipt, which is the failure this tool exists to prevent.
- **A third value, `check: "always"`.** Rejected: two spellings of "run the check" is the same problem with a longer name.
- **Make an absent `check` an explicit `ran: false` and keep the auto rule.** Rejected: it tells the caller the check was skipped, and still skips a check the caller asked for.

## Component / Boundary Impact

`internal/mcp` (the mode it selects, the tool's description, the receipt key's description) and `internal/writer/flow.go` (the removed mode). `internal/writer` is not one of the engine packages; no engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw_write` `check: true` | due on any landed write; the three changes of Decision 3 | T1 | every MCP caller |
| `writer.CheckMode` | `CheckAuto` removed; zero value `CheckOff` | T1 | `internal/mcp`, `cmd/mrw`, the writer's tests |
| tests asserting the old rule | `internal/mcp/check113_test.go`, `depth187_test.go`, `thread121_test.go`, `says098_test.go`, the `era_test.go` golden | T1 | the suite |
| tests relying on the zero mode | `internal/writer/flow113_test.go`, `writes127_test.go`, `partial164_test.go`, `stale200_test.go`; a comment in `cmd/mrw/then092t5_test.go` | T1 | the suite |
| every sentence that states the old rule | the tool description (`mcp.go`), `check`'s description (`schema.go`), the initialize instructions (`instructions.go`, 4096-byte limit, held by `says098_test.go`), `cliRoute["mrw_write"]` (`tools.go`), the opencode plugin's tool and argument descriptions (`index.ts`, ADR-089) | T1 | hosts and callers |
| the tool description and `check`'s schema description | say so | T1 | hosts that read them |
| `AGENTS.md`, `README.md` | the two sentences that describe the old rule | T1 | every caller |
| `scripts/contract.sh` §352 | a prose-only `mrw_write` with `check: true` runs the check; the same without it runs none | T1 | the contract gate |

No receipt key is added or renamed (ADR-111): `check` is present in more cases than before.

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** a requested check ran, or the receipt says it could not.
- **Negative:** a breaking change for an MCP caller that set `check: true` and relied on the skip: it waits for a check, can lose a `then` step, and at depth 8 is refused.
- **Neutral:** the CLI is unchanged.

## Out of Scope

- Making a failed check `isError`, or a check that could not run not one (permanent: boundary: ADR-113 Decision 4 stands as it is)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A host relies on prose writes skipping the check | Low | Med | named in the release note as a behaviour change for MCP callers; the description is explicit |
| The zero mode becoming off turns a writer test red | High | Low | the four files are named above and each asks for `CheckDemand`; the audit command is in Context |

## Rollback

Revert T1; no state is involved.

## Follow-ups

- [ ] First field reports after the release: any host sending `check: true` on prose-only writes.
