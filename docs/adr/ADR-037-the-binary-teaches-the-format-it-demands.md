# ADR-037: The binary teaches the format it demands

**Status:** Accepted
**Accepted:** 2026-09-09 by M — *"Accept"*
**Date:** 2026-09-09
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-012 (the MCP twin of this record), ADR-016 (which surface to be on), ADR-010 (two tools, same engine), ADR-001 (the plan grammar), ADR-002 (per-line ledger), ADR-003 (check from process exit), ADR-035 (no syntax; `anchor=`), docs/adr/BACKLOG.md (the 2026-09-09 review findings this record classifies)
**Governs:** `cmd/mrw/main.go`, `internal/mcp/instructions.go`
**Enforced-by:** `internal/guide/guide_test.go::TestEverySurfaceContainsTheSharedSentences`
**Invalidates:** none — checked. ADR-012 still owns MCP `instructions` and its examples; this record extracts the sentences both surfaces must share and does not drop a field ADR-012 or ADR-016 declared. ADR-010's two-tool boundary is reaffirmed. ADR-019 remains reserved for reach and is not this number.
**Served-path change:** `mrw instructions` prints the contract from the binary, so a caller who installed mrw and has neither this checkout nor the central skill can learn when to reach for it, the two rules that produce most refusals, and the traps that make a red run look green.

## Context

A 2026-09-09 review of this repository (wing_tool-multipathreadwrite, room `reviews`) named six
weaknesses. Five of them are already decided elsewhere, or are the design. One is not: a caller who
has only the binary cannot learn the contract.

**The class this record governs.** Every surface that can start an mrw session for a caller who does
not have this checkout. Enumerated 2026-09-09 with

```
git ls-files cmd/mrw/main.go internal/mcp/instructions.go AGENTS.md README.md .claude/skills/mrw/SKILL.md
```

Five files. This record is authoritative over the first two — the binary. The last three are
mirrors for callers who already have the repo or the palace; they are left out of `Governs:`
because a change to the README is not a change to this decision. T2/T3 assert they *name* the new
command, so a caller who has both is not taught two stories. That is issue #73's gate
(`cmd/mrw/agentsdoc_test.go::TestEverySubcommandReachesTheAgentFacingGuide`), which already fails
when a subcommand is added and AGENTS.md does not mention `` `mrw <name>` ``.

**What each finding becomes, so the next session does not re-derive the split.**

| Finding | Disposition in this record |
|---|---|
| W1 — line-oriented, no target syntax | `(permanent: boundary:)` below. Taught, not fixed. ADR-001 and ADR-035 own the engine. |
| W2 — the operator is the failure mode | **This record.** The CLI teaching text names the pipe trap and exit 3. |
| W3 — two surfaces, one almost-shared contract | **This record** for the shared sentences. Widening MCP is still ADR-010's deferred follow-up. |
| W4 — ledger outside the tree | `(deferred: docs/adr/BACKLOG.md)` — ADR-034 shipped prune; the CLI race is ADR-010's. |
| W5 — process tax (35 ADRs, 4,771-line contract) | `(permanent: boundary:)` — the cost of the promises, not a defect to remove. |
| W6 — BACKLOG gaps (memory, non-Go check, unguarded delete, …) | `(deferred: docs/adr/BACKLOG.md)` — already receipted there, not reopened here. |

**Why this is a record and not a README edit.** On 2026-09-03 the same design — `mrw instructions`,
one `go:embed`, printable into any repo's AGENTS.md — was considered and **not taken**, because it
makes a new public contract. The palace still says so (`wing_tool-multipathreadwrite/decisions`,
PR #41). The review is the reason it now matters: the highest-leverage remaining work was named as
"teach the format from the binary", and doing that without a record is the thing the 2026-09-03
decision refused.

**What already exists, and what it does not cover.** ADR-012 put `instructions` on MCP `initialize`
and bounded it at 4,096 characters (`maxInstructionsChars`, `internal/mcp/instructions.go:30`).
ADR-016 told a caller with both surfaces that the CLI is richer. Neither reaches a person who
installed the binary, has a shell, and has never cloned this repository. `mrw --help` lists
subcommands; it does not teach the plan format, the per-line ledger, or that `mrw write plan | head`
returns head's status.

## Existing Primitives Audit

- **`internal/mcp/instructionsText` (ADR-012):** the handshake document. **Reuse as the source of
  the shared sentences**, not as the CLI's printer. MCP-only rules (ack, paging, the encoded-result
  ceiling) stay there. The CLI must not grow a second authored copy of the same sentences.
- **`triggerRule` and contract §43 (ADR-012):** the threshold sentence, grepped out of the binary
  and AGENTS.md. **Extended:** the same shape covers the rest of `guide.Shared()`, not only the
  trigger.
- **`maxInstructionsChars` (ADR-012):** **reused unchanged.** If Shared() plus the MCP-only
  paragraphs will not fit, existing MCP prose is shortened and the record says which. The bound is
  not raised — every session pays it, whether or not a tool is called.
- **`cmd/mrw.rootCommand` `Commands`:** the registry. **Extended** with one subcommand. The
  selector is this table; `TestEverySubcommandReachesTheAgentFacingGuide` is the reachability gate
  that already exists.
- **`internal/authoring`:** the tally (ADR-009). **Not reused.** Teaching is not counting.
- **A syntax-aware engine, or an LSP, or a tree-sitter pass on the body:** audited and **NOT
  taken.** See Alternatives.
- **MCP tools for `check` / `iter` / `seen` / `stats`:** audited and **NOT taken.** ADR-010 deferred
  that; ADR-016's amendment says widening is a separate record. This one does not become it.

## Decision

**1. One function owns the sentences both surfaces must teach.** `guide.Shared()` is those
sentences, and only those. It is not the MCP handshake and not the CLI pamphlet. A surface may
print more after it; it may not print a different version of it.

The shared sentences, named so a test can fail on any one of them, are:

- the trigger (`3 or more edits, 2 or more files, or several ranges you need to read`)
- a plan applies whole or not at all (if any hunk fails, nothing is written)
- read-before-write is per line, not per file
- mrw models no target syntax: after a multi-line body, read on past the range until the enclosing
  structure closes
- a refusal names the file, the plan line, and the reason

**2. `mrw instructions` prints `guide.CLI()`.** That is `Shared()` plus the CLI-only traps the
review named: never read an exit code through a pipe; exit 3 means the write applied and the check
failed; MSYS rewrites a regex address; a shell glob and an address suffix do not mix. Exit 0.
Stdout is the document. No flags. A caller can `mrw instructions >> AGENTS.md` in any repo.

**3. MCP `instructionsText()` contains `guide.Shared()` verbatim.** `strings.Contains`, not "the
same ideas". The interpolated examples and the MCP-only paragraphs stay in `internal/mcp`. If
adding the syntax sentence pushes the handshake over `maxInstructionsChars`, shorten MCP-only
prose — do not raise the bound, and do not drop a sentence ADR-012 or ADR-016 already required.

**4. Nothing about apply, read, the ledger, or the MCP tool set changes.**

**Go/no-go, checked during execution. If any fails, the task is withdrawn rather than shipped:**

- **`maxInstructionsChars` is still 4096** and the existing bound assertions inside
  `TestTheInstructionsTellAHostHowToAuthorAPlan` stay green unmodified in their bound.
- **No engine change.** `git diff` over `internal/read`, `internal/apply`, `internal/plan`,
  `internal/seen`, `internal/check`, `internal/state` is empty.
- **`guide.Shared()` is not interpolated.** Examples stay in the MCP file, where
  `TestEveryEmbeddedExamplePlanReallyApplies` already executes them. A shared string that embeds
  an example is how the two copies disagree the next time the example changes.
- **The new command is in `rootCommand().Commands`.** Deleting that one line fails both the
  AGENTS.md gate and contract §75.

## Alternatives Considered

- **Leave teaching in AGENTS.md and the skill.** Today's state, and the one the 2026-09-03 decision
  recorded as not reaching a binary-only installer. Rejected because that is the population this
  record is for.
- **Point `mrw --help` at AGENTS.md.** Free, and the same defect ADR-012 rejected: a reference to a
  file the reader cannot open reads as help and is not.
- **Dump AGENTS.md on `mrw instructions` via `go:embed`.** Rejected: AGENTS.md is 376 lines and
  growing, and it is written for an agent already in this checkout. The binary's pamphlet is the
  sentences a stranger needs, not the contributor drill.
- **Raise `maxInstructionsChars` and put the CLI pamphlet on the MCP wire too.** Rejected: every
  MCP session pays the handshake, and the pipe / MSYS / exit-3 traps are not true of that
  transport. ADR-012 set the bound so this file would not become a second AGENTS.md.
- **Make mrw syntax-aware** (parse the target, refuse a body that leaves a surviving closer).
  Rejected: that is a different product. ADR-001 and ADR-035 are the line-oriented contract; this
  record teaches the consequence rather than reversing it.
- **Give MCP feature parity** (`check`, `iter`, `seen`, `stats` as tools). Rejected for this
  record. ADR-010 deferred it; ADR-016's same-day amendment says widening is its own decision.
- **One record that also closes the ledger race, streaming apply, and unguarded delete.** Rejected:
  those are three other decisions, each with an existing BACKLOG entry. A kitchen-sink ADR is how
  a deferred item loses its owner.
- **Generate AGENTS.md from `guide.Shared()`.** Deferred: a two-way sync between the pamphlet and
  the contributor guide is a process tax this record is not paying. The existing `#73` gate plus
  `Contains(Shared())` is the cheaper drift check.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/guide` (new) | The shared sentences and the CLI pamphlet | Yes — changes when a sentence both surfaces must teach is added or reworded |
| `internal/mcp` | Still the handshake, the MCP-only paragraphs, the examples | Yes — `instructionsText` interpolates `guide.Shared()` |
| `cmd/mrw` | Still the CLI registry | Yes — one new `Commands` entry |
| `internal/apply`, `internal/read`, `internal/plan` | Unchanged | Untouched |

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| CLI | new `instructions` subcommand, exit 0, stdout is `guide.CLI()` | `cmd/mrw` | any caller with the binary |
| MCP `initialize.instructions` | must contain `guide.Shared()` verbatim | `internal/mcp` | any MCP host |
| contract.sh | new §75 driving `$MRW instructions` | `scripts/contract.sh` | CI, `adr-verify` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `guide.Shared()` (T1) | T1 | T2, T3 | No — additive |
| `mrw instructions` subcommand (T2) | T2 | T3 | No — additive |

## Implementation

See `docs/adr/ADR-037-the-binary-teaches-the-format-it-demands/tasks/README.md`.

## Consequences

- **Positive:** a binary-only installer can learn the contract. The two surfaces cannot silently
  disagree about the five shared sentences. The review's W1 is taught rather than rediscovered.
- **Negative:** a third copy of some sentences (guide / MCP extras / AGENTS.md). Drift is asserted,
  not eliminated. `mrw instructions` is another subcommand the `#73` gate will nag about forever.
- **Neutral:** MCP handshake length is unchanged as a bound; its text may shrink to make room for
  the syntax sentence.

## Out of Scope

- Making mrw parse or refuse target-language structure (permanent: boundary: line-oriented is ADR-001; this record teaches the consequence)
- The 35-ADR / contract.sh process tax (permanent: boundary: that is how these promises stay true)
- MCP tools for `check`, `iter`, `seen`, `stats` (deferred: docs/adr/BACKLOG.md)
- Fixing the CLI parallel-read ledger race (deferred: docs/adr/BACKLOG.md)
- Streaming / memory-bounded apply, non-Go check scoping, unguarded multi-line delete, Windows `%LOCALAPPDATA%` state path, a live-model plan-authoring benchmark (deferred: docs/adr/BACKLOG.md)
- ADR-019 reach / multi-root (permanent: boundary: that number is reserved and this is 037)
- Generating AGENTS.md from `guide.Shared()` (deferred: docs/adr/BACKLOG.md)
- Telemetry on who ran `mrw instructions` (permanent: fact: ADR-009 refused a hosted analytics or telemetry SDK; citation: file `docs/adr/ADR-009-mrw-counts-what-happens-to-the-plans-it-is-given.md:64`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Adding the syntax sentence overflows `maxInstructionsChars` | Med | High — the handshake test goes red and a rushed fix raises the bound | Go/no-go: shorten MCP-only prose; Stop Condition if the bound is the proposed fix |
| `Shared()` is copied by hand into MCP and then edited in one place | Med | High — the defect this record exists to close | `strings.Contains` test; Shared() is a function call, not a comment saying "keep these in sync" |
| `mrw instructions` is added and AGENTS.md is not | Low | Med — `#73` already fails the build | T2 includes that gate in its fence |
| Callers treat the pamphlet as AGENTS.md and miss the contributor drill | Low | Low | The pamphlet says it is the stranger's copy; AGENTS.md stays the in-repo guide |

## Rollback

Delete the `instructions` command and `internal/guide`. MCP `instructionsText` returns to an
inline string. Additive; no state, no plan-format change. A caller who scripted
`mrw instructions` gets "unknown command" at exit 2.

## Follow-ups

- [ ] After Shared() has been in production, decide whether AGENTS.md's "Using mrw" section should
      open with `mrw instructions` rather than restating the five sentences.
