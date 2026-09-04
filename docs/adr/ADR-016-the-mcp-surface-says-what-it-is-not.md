# ADR-016: The MCP surface says what it is not

**Status:** Accepted
**Accepted:** 2026-09-04 by M — *"our MCP definition of MRW adds ambiguity, agents prefer to stay with mcp limited tool, rather than using the binary local one. we need to fix that / add instructions that if you see this tool locally - use it at its full potential, not the mcp. mcp is for desktop app"*, and *"MCP tool is limited by the projects dir as well, local mrw binary is not limited"*.
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-012 (the surface teaches the format; this teaches which surface to be on), ADR-011 (owns the root binding that makes MCP's reach narrower on purpose), ADR-010 (owns the two-tool surface this declines to widen)
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/mcp_test.go::TestTheSurfaceSaysTheCLIIsRicher`
**Invalidates:** none — checked. Nothing about the tools' behaviour changes; this adds one paragraph to `instructions` and one clause to each description. ADR-010's two-tool boundary and ADR-011's root binding are both reaffirmed rather than touched.
**Served-path change:** an agent that has BOTH a shell and this MCP server is told, on the wire, that the CLI is the fuller tool — instead of settling for the smaller surface because it is the one in its tool list.

## Context

**A registered MCP tool outcompetes a CLI the agent has to remember exists.** An MCP tool appears in
the tool list with a schema and a description; the CLI is a string an agent must recall from
AGENTS.md, a skill, or habit. M observed the consequence directly: agents *"prefer to stay with mcp
limited tool, rather than using the binary local one."* Nothing on the wire tells them the surface
they can see is the smaller one.

**And it is much smaller. Measured 2026-09-04 at `d90be24`,** by enumerating both surfaces:

| Only on the CLI | What an MCP-only caller cannot do |
|---|---|
| `--grep PATTERN`, `--exclude GLOB` | walk a tree and serve every match in ONE call — the lever AGENTS.md calls the part that gets missed |
| `--files-from FILE` | pipe a searcher's output straight in |
| `--check` | run the project's check, scoped to the files the write touched |
| `--json` | a parseable receipt for a hook or a gate |
| `--stat`, `-C`, `--max-lines`, `-N` | cheap shapes and context windows |
| `check`, `iter`, `seen`, `stats` | four subcommands, including the ledger and ADR-009's tally |

The MCP surface is `specs` and `plan` plus `dry_run`. Everything above is absent.

**The reach differs too, and that one is deliberate.** Verified the same day: one CLI process serves
two unrelated checkouts with `-C`, while an MCP server refuses any path outside the root it resolved
at startup —

    ==> /tmp/other/b.txt  REFUSED  … outside the root

That is ADR-011 working exactly as designed: the root binding is the safety model, and issue #75
records that a host which does not set `CLAUDE_PROJECT_DIR` must pass `--root` or serve `/`. So the
narrower reach is a feature of the transport, not a defect to widen — but a caller choosing between
surfaces should know it is choosing one checkout over any checkout.

**Why this is a record and not a doc edit.** Deciding that a tool's own description should advertise
a better alternative is a decision, and the obvious alternative — reach feature parity — is the one
ADR-010 already refused. Writing it down is what stops the next session "fixing" the gap by widening
MCP.

## Existing Primitives Audit

- **`instructionsText` (ADR-012-T1):** the handshake document. **Extended** with one paragraph, at
  the top, because a caller that has already chosen a tool has stopped reading.
- **The two tool descriptions (ADR-012-T1):** already lead with when to reach for the tool.
  **Extended** with the same routing in one clause each, since a host that ignores `instructions`
  reads only these — ADR-012's own lesson.
- **`maxInstructionsChars` (ADR-012):** the 4,096-byte bound every session pays. **Reused
  unchanged**, and the Stop Condition below refuses to raise it.
- **`ResolveRoot` and the root binding (ADR-011):** **reaffirmed, not touched.** The narrower reach
  is described, never widened.
- **Adding `--grep` and `--check` as MCP tools:** audited and **NOT taken.** See Alternatives.

## Decision

**1. The handshake says which surface to be on, first.** `instructions` opens with: if you can run
shell commands and `mrw` is on PATH, prefer the CLI — it has `--grep`, `--files-from`, `--check` and
`--json`, and `-C` points it at any checkout. This server is two tools over ONE fixed checkout, and
exists for hosts without shell access.

**2. Both tool descriptions carry the same routing in one clause**, because a host that ignores
`instructions` reads only the descriptions. That is ADR-012's finding applied to itself.

**3. It says what the CLI has, concretely.** "The CLI is richer" routes nobody. Naming `--grep` and
`--check` tells a caller what it is giving up, and those two are the ones that change what a task
costs: find-and-serve in one call, and verify after write.

**4. Nothing about behaviour changes, and the MCP surface is not widened.** Two tools, one root.

**Go/no-go, checked during execution:**

- **`instructions` stays under `maxInstructionsChars`.** If it will not fit, something already there
  is shortened and the record says which — the bound is not raised.
- **The claim is true when made.** The test asserts the named flags exist in the CLI's own help
  output, so a flag that is renamed or removed turns the advice red instead of leaving the wire
  recommending something that is gone.
- **No behaviour change**: `go test ./...` and the full contract suite pass unmodified apart from the
  new row.

## Alternatives Considered

- **Say nothing and rely on AGENTS.md.** Today's state, and the one M's observation refutes. AGENTS.md
  is a file in a checkout; the tool list is in front of the model. The surface with a schema wins.
- **Give MCP feature parity — `--grep`, `--check` as tools.** The obvious "fix", and refused. ADR-010
  decided the tool surface is two tools over the same engine, and every capability added is a second
  place for the contract to drift (ADR-012 shipped a wrong enum on a surface with two tools). It also
  would not close the reach gap, which is deliberate. If a specific capability is genuinely needed
  over MCP, that is its own record with its own evidence.
- **Remove the MCP server.** It exists for Claude Desktop, which has no shell — the population this
  routing paragraph explicitly excludes. Removing it would strand exactly the caller it serves.
- **Make the MCP tools refuse when a shell is available.** Not knowable: a server sees a JSON-RPC
  message, not the host's capabilities, and guessing would break Desktop.
- **Put the routing only in the descriptions, not `instructions`.** Cheaper and worse: the
  description is read when a tool is already being considered, and the point is to be read before the
  choice.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | The wire protocol, the descriptors, and now the routing advice between surfaces | Yes |
| `cmd/mrw` | Unchanged. The CLI does not learn that MCP exists | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| `instructions` opens with surface routing | Public contract, additive text | Any MCP host |
| Both descriptions gain a routing clause | Public contract, additive text | Any MCP host |
| No tool, schema or behaviour changes | — | — |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the routing text and its test | T1 | — | No — additive |

## Implementation

One task. It is prose plus one test plus one contract row, and splitting prose from the test that
holds it true is how ADR-012 shipped a wrong enum.

## Consequences

- **Positive:** an agent with both surfaces is told which one is fuller, before it picks.
- **Positive:** the reach difference is stated, so "why can it not see that file" has an answer on
  the wire rather than in an issue.
- **Negative:** `instructions` grows, and every session pays it. Bounded, and the Stop Condition
  refuses to raise the bound.
- **Negative, and worth naming:** a tool that advertises a better alternative is unusual, and a host
  could plausibly surface it as a warning. Accepted — the alternative is a caller quietly using the
  weaker surface, which is the failure M actually observed.
- **Neutral:** Desktop callers read a paragraph whose first clause does not apply to them. It is two
  sentences, and it tells them why their surface is shaped as it is.

## Out of Scope

- Adding any capability to the MCP surface (permanent: boundary: ADR-010 owns the two-tool decision; a specific capability needs its own record)
- Widening the MCP root, or multi-root (deferred: docs/adr/BACKLOG.md — measured as unneeded for this user's plans, and it is ADR-011's boundary)
- Changing any CLI behaviour (permanent: boundary: the CLI does not learn that MCP exists)
- Detecting host capabilities (permanent: boundary: a server sees a message, not a shell)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The advice names a flag that later disappears | Med | Med | The test asserts each named flag appears in the CLI's own help output, so a rename turns it red |
| The paragraph pushes `instructions` over its bound | Med | Low | Asserted; the Stop Condition shortens rather than raises |
| A Desktop caller is told to use a CLI it does not have | High by design | Low | The clause is conditional — "if you can run shell commands" — and the next sentence says what this server is for |

## Rollback

Revert the commit. The text is additive; no schema, tool or behaviour changes.

## Follow-ups

- [ ] If a specific CLI capability turns out to be needed over MCP often enough to matter, record it
      with the evidence rather than widening the surface by reflex
