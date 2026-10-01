# ADR-113: One write path, and an MCP write is checked

**Status:** Accepted
**Accepted:** 2026-10-01 by Zy — "all", on the gap list whose items 1 (an MCP write is unverified) and 2 (the two surfaces decide the same outcome in two places) are this record's scope, with the answer "Default on, opt-out (Recommended)" to "should mrw_write run the project check when one is due?". The record's text was drafted after that instruction and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-009, ADR-024, ADR-054, ADR-056, ADR-072, ADR-075, ADR-083, ADR-092, ADR-093, ADR-095, ADR-102, ADR-111, ADR-112
**Invalidates:** ADR-054 — the clause that the check runs on a CLI write only (an MCP write now runs it too); ADR-083 — its line that the MCP path never runs a check, so an applied MCP plan is always counted `applied`; ADR-102 — the clause of its Decision 4 counting an MCP landing whose ledger failed as `applied`: with a check due it is now `check_not_run`, as the CLI counts it; ADR-112 — its Out of Scope entry "MCP runs no check"; ADR-075 — not a clause of its Decision, but its T1 fence's grep for `writer.Apply(` in `cmd/mrw/main.go`, which now reaches the call through `writer.Prepare` and `Land` (exempted in `scripts/fence-prose.py`, as ADR-108 did for ADR-038)
**Governs:** `internal/writer/flow.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/schema.go`, `internal/mcp/instructions.go`, `docs/receipts.txt`, `AGENTS.md`, `README.md`, `scripts/contract.sh`, `scripts/fence-prose.py`
**Enforced-by:** `internal/mcp/check113_test.go::TestAnMCPWriteRunsTheCheck`
**Served-path change:** `mrw_write` runs the project's check after a write that lands, by the same rule `mrw write` uses (ADR-054), and returns its verdict in the receipt — `check` and `drift`, as `mrw write --json` carries them. `check: false` opts out. A check that fails leaves the write applied and the call not an error; the receipt says the tree is changed and unverified.

## Context

`mrw write` and `mrw_write` each prepare a plan, apply it, count the outcome and render it, in two copies: `cmd/mrw/main.go:1271-1536` and `internal/mcp/tools.go:635-747` (at `95f5dab`). ADR-102 shared only the classification of what landed (`writer.MutationOf`) and filed the rest in BACKLOG "From ADR-102", to arm "on the next finding whose cause is that the two surfaces decided the same outcome differently". The gap list Zy answered on 2026-10-01 names that finding: an MCP write runs no check, so a caller on that surface — Claude Desktop, any host with no shell — gets a receipt that says what was written and nothing about whether it still builds, while the same plan through the CLI is checked by default (ADR-054). The CLI's rule lives in a closure (`checkWanted`, `main.go:1290`) that MCP cannot call.

**Audit of the class** — *a decision about one write that both surfaces make*: every `authoring.Record`, `authoring.Reclassify`, `authoring.RecordRecent` and `authoring.RecordPricing` call, and every `check.Load`, `check.Run`, `check.DepthRefusal`, `writer.Before` and `writer.Drift` call, outside tests. Enumerated with `mrw read --grep 'authoring\.(Record|Reclassify|RecordRecent|RecordPricing)\(|check\.(Load|Run|DepthRefusal)\(|writer\.(Before|Drift)\(' --exclude '*_test.go' cmd internal`. Moved into the shared sequence: every one from the harness read onward. Left per surface, deliberately: the decisions made before a plan reaches the sequence — a document that did not parse or compile, a `body=@` file that did not load, a working-set pointer that did not resolve, a `--json` path that is not UTF-8 — each counted where its surface decides it, because each surface's input differs (a file and flags, a JSON argument); and `mrw check` (`main.go`), which shares the step helpers (`ResolveSteps`, `RunSteps`, `StepStop`) but has no preceding plan landing to count.

**The host's clock** (`https://code.claude.com/docs/en/mcp` and `/env-vars`, read 2026-10-01): Claude Code bounds a tool call by a hard wall-clock limit, `MCP_TOOL_TIMEOUT`, default about 28 hours, which progress notifications do not extend; and by an idle window — 30 minutes for a stdio server, `CLAUDE_CODE_MCP_TOOL_IDLE_TIMEOUT` — which a response or a progress notification resets. Claude Desktop's limits are not documented. The project check's own default bound is 5 minutes (`internal/check/check.go:80`), inside both. A check that outlives the host's limit loses the receipt, not the write: the write landed, and the ledger and the tally record it (ADR-075, ADR-072). The ledger is not an idempotency guard — after a landing it records the files as wholly known, so a re-sent plan whose hunks still validate, an `insert-after` for one, applies a second time.

## Existing Primitives Audit

- **`writer.Apply`** (`internal/writer/writer.go`) — the one apply both surfaces call, under the write lock; records what landed in the ledger.
- **`writer.MutationOf`** (ADR-102) — none, partial or complete, from a receipt.
- **`writer.Before` / `writer.Drift`** (ADR-112) — what changed while the check ran.
- **`check.Load`, `check.Run`, `check.DepthRefusal`, `check.RunSteps`** — the harness, the check, the depth guard, the steps.
- **`authoring`** — the ADR-009 tally, the ADR-055 ring and the ADR-056 pricing.

The new code is the sequence that calls these, moved out of `cmd/mrw` so both surfaces run it; no new primitive.

## Decision

1. **One sequence, in four phases, in `internal/writer/flow.go`.** `writer.Prepare` runs the gates that refuse before anything is written — the harness read when a check may be due or a step is asked for, the steps resolved, the depth refusal when a check would be due — and counts a refusal `refused_apply`. `Prepared.Land` takes the ledger snapshot (a failure there is a refusal too), applies, and counts the landing: `partially_applied`, `refused_apply`, `applied`, or, for a landing whose ledger failed, `check_not_run` when a check was due and `applied` otherwise; it records the recent ring, and prices a landing nothing will check (a partial commit, a ledger failure). `Landed.Verify` runs the check when it is due, the drift, and the steps; a check that could not run at all (`check.Run`'s error) moves the count to `check_not_run` there and is priced nowhere, as today. `Landed.Settle` moves the count to the verdict and prices it. Each phase returns facts; each surface renders them, maps them to its exit code or `isError`, and keeps its own words. The phases are where they are because the CLI prints its human receipt and counts the landing BEFORE the check (ADR-072), so a write killed during its check is still printed and counted, and renders its `--json` receipt BEFORE it reclassifies and prices, so an output failure leaves the count where it was.
2. **The CLI moves onto it unchanged.** Every exit code, receipt value, tally, pricing count and line of output stays as it was, held by a characterization test over the write path's outcomes — each run in `--json` and human form, its full normalized stdout and its `stats` compared with a golden generated from the binary before the move. The outcomes it does not drive are named in T1: a check or step interrupted by a signal, and a `--json` encoder that fails on stdout; their order is preserved by construction (Settle runs after rendering) and the signal paths keep their own tests.
3. **`mrw_write` runs the check by default.** The rule is ADR-054's: a write that lands, touches a file that is not prose, in a tree that declares or infers a check, runs it; `check: false` turns it off, as `--no-check` does. There is no MCP twin of `--check` (demand a check on prose): a caller that wants one on prose can use the CLI. The harness is read before anything is written, so a malformed `.quality-harness.json` now refuses an MCP write that would have checked, as it refuses the CLI's — a refusal MCP did not make before; `check: false` skips the read.
4. **The verdict travels in the receipt.** `mrw_write`'s structured value gains `check` (the CLI's `check.Result`) and `drift`; the text leads with the verdict. A check that ran and did not pass — or timed out — is NOT `isError`: the call did what it was asked — write, then check — and the verdict is data. ADR-024 measured that `isError` makes a host drop the middle of a result, and here the middle is the tail of the check; and a host that reads `isError` as "the call failed" invites a re-send, which applies the write twice. A check that could not run — a shell that could not start, a log that could not be created, an interrupt before it started — IS `isError`, with the receipt, as ADR-102 Decision 4 decided for a landing whose ledger failed: the server could not do what it was asked. (A check demanded with no command cannot happen here: MCP has no `--check`.)
5. **The receipt is never cut past the verdict.** When the receipt does not fit the ceiling, the check's tail lines go first, then the successful hunk verdicts and unwritten files as now; the verdict (`ran`, `exit_code`, `skipped`) is kept. The terminal sentence — the answer when not even that fits — gains a phrase naming the verdict from a fixed set (`its check passed`, `its check did not pass (exit N)`, `its check could not run`, `no check ran`), never the check's own text; `writeFloor` is measured over every phrase at the widest exit code, so a write whose verdict the server could not report is refused before it applies. That terminal answer stays `isError`, whatever the verdict: it carries no receipt, and a caller holding only a sentence must read the files before acting — the one place the flag follows the size of the answer rather than the verdict.
6. **Steps stay CLI-only.** `--then-sh` runs any shell command, and a named `--then` over MCP would need its own floor and elision; neither was asked for. The shared `Verify` takes a step list; MCP passes none.

## Alternatives Considered

- **Run the check on MCP without sharing the sequence** — rejected: it would be a third copy of ADR-054's rule beside the CLI's closure and `writeTool`'s tally switch, which is the split the gap list named.
- **One `writer.Write` call that returns the final outcome** — rejected: the CLI prints and counts the landing before the check, and renders its JSON before it reclassifies (Decision 1); one call would move one of those.
- **Default the MCP check off, opt-in** — rejected by Zy's answer: the surface that has no shell is the one whose caller cannot run the check another way.
- **`isError` on a failed check, mirroring the CLI's exit 3** — rejected (Decision 4): a host drops the middle of an `isError` result (ADR-024) and may read it as "retry".
- **Progress notifications during the check** — not taken now: they would keep Claude Code's 30-minute stdio idle window open, but the check's default bound is 5 minutes, and they cannot extend the hard limit. Deferred with a trigger.

## Component / Boundary Impact

`internal/writer` (T1), `cmd/mrw` (T1), `internal/mcp` (T2). None is an engine package: `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` and `internal/rooted` stay byte-identical; `go.mod` keeps one requirement. `internal/writer` gains an import of `internal/check` and `internal/authoring`; neither imports `writer` (`go list -deps`, 2026-10-01), so there is no cycle. The step helpers `mrw check` shares (`ResolveSteps`, `RunSteps`, `StepStop`, `Shown`) move with the sequence.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `writer.Prepare` / `Land` / `Verify` / `Settle` | the shared sequence | T1 | `cmd/mrw` (T1), `internal/mcp` (T2) |
| `mrw_write` input schema | `check` (boolean, default true) | T2 | MCP callers |
| `mrw_write` structured value and output schema | `check`, `drift`, each described | T2 | MCP callers |
| `docs/receipts.txt` | `mcp_write check` and every key under it, `mcp_write drift` | T2 | ADR-111's tests |
| `mrw_write` description, handshake, AGENTS.md, README | say an MCP write is checked; inspect before re-sending after a lost receipt | T2 | readers |
| `scripts/contract.sh` | §213 (T2) | T2 | CI Linux |
| `scripts/fence-prose.py` | ADR-075 T1's clause exempted | T1 | the static gate |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `writer.Prepare` / `Land` / `Verify` / `Settle` | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2, in one pull request as separate commits — `scripts/fence-prose.py` checks every task's fence, so a merged T1 beside a pending T2 would leave the static gate red.

## Consequences

- **Positive:** an MCP write says whether the tree still passes its check; the two surfaces count and verify a write by one sequence.
- **Negative:** an MCP write that touches code now waits for the check — up to its bound, 5 minutes by default — and holds the server for that long; a malformed harness now refuses an MCP write.
- **Neutral:** the CLI's behaviour is unchanged.

## Out of Scope

- Steps (`then`) over MCP (deferred: `docs/adr/BACKLOG.md` "From ADR-113")
- Progress notifications while an MCP check runs (deferred: `docs/adr/BACKLOG.md` "From ADR-113")
- A host whose tool timeout is shorter than the check (permanent: boundary: mrw cannot see the host's clock; the description names `check: false`, and a lost receipt leaves the write applied, recorded in the ledger and the tally)
- `mrw check` on its own (permanent: boundary: it has no preceding plan landing to count; it shares only the step helpers)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the extraction changes a CLI outcome | Medium | High | the characterization golden over the outcomes, written first and generated from the binary before the move |
| a host times out during a long check and the caller re-sends | Low | High | the description says the write lands first, that a lost receipt means read the files before re-sending, and names `check: false`; nothing in mrw makes a re-send idempotent |
| an existing MCP caller now sees writes refused over a malformed harness | Low | Low | the refusal names the file and `check: false` |

## Rollback

Revert T2, then T1. T2's receipt keys disappear with it, which ADR-111 would call a removal: revert before a release ships them.

## Follow-ups

- None — the record carries no open follow-up.
