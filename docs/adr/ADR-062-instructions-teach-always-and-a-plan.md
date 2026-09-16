# ADR-062: `mrw instructions` teaches always and a plan

**Status:** Accepted
**Accepted:** 2026-09-16 by M — *"accept"*; then *"we must instruct to use it always and plan activity"* (the 3+ When clause *"signals for agents to never use MRW, since these always wil be one truns"*)
**Date:** 2026-09-16
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-012, ADR-016, ADR-037, ADR-045, docs/adr/BACKLOG.md
**Governs:** `internal/guide/guide.go`, `internal/mcp/instructions.go`, `internal/mcp/mcp.go`, `scripts/contract.sh`
**Enforced-by:** `internal/guide/guide_test.go::TestCLITeachesAlwaysAndAPlan`
**Invalidates:** ADR-037 — CLI() is no longer a pamphlet; Shared()'s first sentence is no longer a 3+ threshold
**Served-path change:** `mrw instructions` (and MCP `initialize` / tool descriptions, which print Shared() and `triggerRule`) tell a caller to use mrw always and to plan one read, one plan, one write — not to stay on their editor below three edits.

## Context

**The class this record governs.** Every agent-facing string the binary emits that tells a caller when to use mrw. Enumerated 2026-09-16 with

```
git grep -n '3 or more edits\|Below that' -- '*.go' '*.md' 'scripts/contract.sh'
```

Members: `guide.Shared()`, `triggerRule`, MCP tool descriptions, `mrw instructions` stdout, contract §43 / §75 / §85 Shared tuples, AGENTS.md "Using mrw" (the §43 mirror; not in `Governs:`). Left out: historical ADR prose (ADR-012, ADR-037) — those stay as the record of what was true; this record invalidates the live clause. Left out: apply / plan / seen / check / state — this is teaching, not the engine.

**Why this is a record.** A PATH agent has the binary, not AGENTS.md. `mrw instructions` was the on-ramp (ADR-037). Its first Shared() sentence was *"Reach for mrw when the task touches 3 or more edits, 2 or more files, or several ranges you need to read."* Agent turns are one edit. That sentence trains them never to call mrw. M accepted growing CLI() into the effective-use document (*"accept"*), then corrected the When: use it always, and plan the activity.

ADR-037 still owns the subcommand (`mrw instructions`, extra args exit 2, handshake ≤ 4096, Shared verbatim on both surfaces). This record changes what Shared says and what CLI() may carry after it.

## Existing Primitives Audit

- **`guide.Shared()` / `guide.CLI()` / `guide.WhyAllOrNothing()`.** Reused. Shared stays five sentences both surfaces contain verbatim. CLI() is still stdout of `mrw instructions`. The why stays extra, not a sixth Shared sentence (contract §85).
- **`triggerRule` and contract §43.** Reused as the wire copy of Shared's first sentence. The duplication stays asserted, not trusted.
- **`maxInstructionsChars` (4096).** Unchanged. The cookbook lives on CLI(), not on `initialize`.
- **Generating AGENTS.md from Shared().** Audited and rejected (ADR-045). AGENTS.md "Using mrw" is still authored; it must contain the same first sentence because §43 greps it.

## Decision

**Use mrw always. Plan the activity as one read of every site, then one plan, then one write.**

Shared()'s first sentence is that instruction, verbatim on CLI and on the MCP handshake:

`Use mrw always: plan the activity as one read of every site, then one plan, then one write.`

The other four Shared sentences stand. `triggerRule` is that first sentence. Tool descriptions pitch it; they do not say "below that, don't."

**`guide.CLI()` is the effective-use document.** After Shared and the why and the operator traps ADR-037 already named, it teaches the plan ops (including `@@ path 0 create` for a new file), so a caller with only the binary can drive a write. It does not dump AGENTS.md. It does not carry the contributor drill, first-red, or `scripts/contract.sh`.

**The MCP handshake stays Shared() plus the why plus the MCP-only rules (ack, paging, two tools).** It does not grow the cookbook. `maxInstructionsChars` stays 4096. A CLI-only needle (`@@ path 0 create`) must not appear in `instructionsText()`.

A 3+ threshold, or "below that use your editor", is a defect on any live teaching surface this class covers.

## Alternatives Considered

- **Keep 3+ / "below that, don't" as the When.** Rejected: agent work is one turn; the threshold trains them never to call mrw. M named this.
- **Put the cookbook on MCP `initialize`.** Rejected: paid by every session; 4096 is the bound that keeps it from becoming a second AGENTS.md (ADR-037).
- **Raise `maxInstructionsChars`.** Rejected: same.
- **Generate AGENTS.md from Shared().** Rejected: ADR-045.
- **Leave AGENTS.md on 3+ and only change the binary.** Rejected: §43 holds the wire against AGENTS.md; two stories is the defect that row exists to catch.

## Component / Boundary Impact

`internal/guide` owns Shared() and CLI(). `internal/mcp` copies Shared's first sentence as `triggerRule` and must not teach the old threshold in descriptions or `instructionsText()`. `cmd/mrw` still prints `guide.CLI()`; no new flag. Engine packages unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `guide.Shared()` first sentence | always + plan | T1 | CLI(); MCP handshake; §43 / §75 / §85 |
| `guide.CLI()` | effective-use after the traps (`@@ path 0 create`) | T1 | `mrw instructions` stdout |
| `triggerRule` / tool descriptions | same first sentence; no "below that" | T1 | `tools/list`; §43 |
| MCP `instructionsText()` | Shared changes with it; no cookbook needle | T1 | `initialize` |
| contract §114 | next free after §113 | T1 | `adr-verify`, CI |

Exit codes unchanged. Extra args to `mrw instructions` stay exit 2.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| always + plan on Shared(); cookbook on CLI() only (T1) | T1 | — | Yes — public teaching text |

## Implementation

See `docs/adr/ADR-062-instructions-teach-always-and-a-plan/tasks/README.md`.

## Consequences

- **Positive:** a PATH agent that reads `mrw instructions` is told to use mrw and to batch the turn. MCP handshake and descriptions say the same When.
- **Negative:** one-file one-edit still costs two calls and more bytes than the file; that cost is accepted so the agent does not skip the tool. `StrReplace` remains shorter on bytes (docs/comparison.md names the cost; it must not teach "stay out").
- **Neutral:** handshake size stays ≤ 4096. ADR-037's subcommand and extra-args usage stand. ADR-045 stands.

## Out of Scope

- Raising `maxInstructionsChars` (permanent: boundary: the handshake is paid once per session)
- Generating AGENTS.md from Shared() (permanent: boundary: ADR-045)
- Dumping AGENTS.md (contributor drill, first-red, contract.sh) into CLI() (permanent: boundary: CLI() is effective-use, not this repository's process)
- A cookbook on MCP `initialize` (permanent: boundary: ADR-037's 4096 bound)
- Engine packages `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` (permanent: boundary: teaching, not apply)
- Zeus or quality-harness edits (external: those repositories: not this checkout)
- Updating the centralised `mrw` skill in another palace (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Handshake overflows 4096 because Shared grew | Low | High | Shared stays one replaced sentence; cookbook is CLI-only; §85 already bounds 4096 |
| AGENTS.md and the binary disagree | Med | High | §43 greps the same first sentence out of both |
| Historical ADRs still quote 3+ and a later session re-teaches it | Med | Med | this record Invalidates the live clause; historical prose stays |

## Rollback

Revert Shared()'s first sentence, `triggerRule`, CLI() cookbook lines, §43 / §75 / §85 / §114 tuples, and the AGENTS.md / BESTPRACTICES / comparison / skill mirrors. Extra-args exit 2 and 4096 do not move.

## Follow-ups

- [ ] Centralised `mrw` skill (agentsmemory) — receipt in BACKLOG; not this binary
