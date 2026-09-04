# Task ADR-017-T2: Teach finding, and repair ADR-016's sentence in the same commit

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (prose, one test, one contract row)
**Owner:** unassigned
**Produces:** the taught form, and ADR-016's corrected routing
**Consumes:** grep over MCP, its index degradation and its refusals (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the taught behaviour is the shipped behaviour`, `the routing names only flags the CLI still exclusively has`

## Goal

A caller with only the MCP surface learns that it can find files it cannot name, and the routing text
stops claiming `--grep` is something only the CLI has.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go` | edit | the READING paragraph gains finding; the routing sentence drops `--grep` |
| `internal/mcp/mcp.go` | edit | `mrw_read`'s description says the same, for a host that ignores `instructions` |
| `internal/mcp/mcp_test.go` | edit | the taught text is asserted against the shipped behaviour and against the CLI's help |
| `AGENTS.md` | edit | the MCP paragraph maps `--grep` onto the tool argument |
| `README.md` | edit | the MCP section documents the new arguments and the index |
| `scripts/contract.sh` | edit | §52 — what the wire teaches is what the binary does |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): the shipped `instructions` and `mrw_read`'s
   description say the surface can find across a tree, and neither claims `--grep` as a CLI
   exclusive. [proof: acceptance]
2. [S2] Repair ADR-016's sentence. It names `--grep` among the CLI's exclusives, in two places, and
   §50 asserts each named flag against the help of the subcommand it binds to. The remaining
   exclusives are `--files-from`, `--check`, and the `check`/`iter`/`seen`/`stats` subcommands.
   ⚠ The check §50 performs is that a named flag EXISTS in the CLI's help — it cannot notice that a
   flag is no longer exclusive, because MCP has no help output to diff against. So this step is not
   protected by the gate that motivated it, and S3 is what protects it. [proof: mutation]
3. [S3] Assert the exclusivity claim in the direction the gate cannot: for every flag the routing
   text names as CLI-only, the MCP input schema must NOT declare an argument of that name. That is
   what makes "only it has X" checkable rather than merely well-intentioned. [proof: mutation]
4. [S4] Say the consequence of the index, not only its existence: a caller handed addresses has not
   been handed content, and must read what it chose. [proof: mutation]
5. [S5] Fix `AGENTS.md` and `README.md`. AGENTS.md's MCP paragraph maps shell recipes onto the tools
   and would otherwise send an agent to a flag the tool now has. [proof: acceptance]
6. [S6] Add contract §52: assert the shipped text describes finding AND that a real grep against the
   built binary does what it describes, in one row. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr017-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr017-t2.out \
  && [ "$(grep -cE '^--- PASS: (TestTheSurfaceTeachesFinding|TestTheRoutingClaimsOnlyRealExclusives)\b' /tmp/adr017-t2.out)" = "2" ] \
  && grep -q '^# 52\.' scripts/contract.sh \
  && go test ./cmd/mrw/ -run TestEverySubcommandReachesTheAgentFacingGuide \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause is to be run BEFORE this fence is finalised and must return **zero hits**: the two test
names and `# 52.`. The `cmd/mrw` clause is deliberately a POSITIVE run of an existing gate rather
than a new one — this task edits `AGENTS.md`, and #73's gate is the thing that notices if that edit
removes a subcommand mention while adding a paragraph.

`TestTheRoutingClaimsOnlyRealExclusives` is the interesting one, and it exists because §50 cannot do
this job: §50 checks that a named flag exists in the CLI's help, which stays true of `--grep`
forever. Nothing today can notice that a flag named as CLI-ONLY has since appeared on the MCP
surface. That is the defect this task would otherwise ship, one record after ADR-016 was written
about exactly this class.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSurfaceTeachesFinding` | `internal/mcp/mcp_test.go` | S1, S4 — the wire teaches finding and the index's consequence | — | S1, S4 |
| `TestTheRoutingClaimsOnlyRealExclusives` | `internal/mcp/mcp_test.go` | S2, S3 — no flag named CLI-only is declared by the MCP schema | — | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests above |
| 2 — something selects it | `initializeResult()` and `tools()` are read by every host |
| 3 — the caller can discover it | this task IS the discovery surface |
| 4 — it is used | not measured and will not be — see T1's rung 4 |

## Mutation Log
<!-- filled during execution -->

## Invariants

- The behaviour described on the wire is the behaviour the binary has.
- No flag the routing calls CLI-only is an argument the MCP schema declares.
- `instructions` stays under `maxInstructionsChars`.
- Every subcommand still reaches `AGENTS.md` (#73's gate).

## Risks

- The routing text is edited for `--grep` and some other claim in it goes stale unnoticed.
  Mitigated by S3 asserting the whole exclusivity list rather than the one flag that changed.
- Teaching finding pushes `instructions` over its bound. It was at 3,964 of 4,096 when ADR-016
  merged, so this WILL bind. See the Stop Condition — it is expected, not a surprise.

## Stop Condition

Stop if teaching this needs `instructions` past `maxInstructionsChars`. Shorten what is there or cut
an example, and say which — do not raise the constant to fit. The routing paragraph is the obvious
donor, since dropping `--grep` from it already frees text.

## Out of Scope

- Any behaviour change (T1 owns it)
- Re-opening ADR-016's routing decision (its DIRECTION stands; only the flag list is wrong)
- Any root change (deferred: ADR-018)

## Verification Log
<!-- filled during execution -->
