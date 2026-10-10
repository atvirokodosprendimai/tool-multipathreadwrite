# ADR-141: `mrw instructions --core` prints the eight rules

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — answered "Guidance core (Recommended)" to "Which should the loop take next?", after "/loop continue delivering, end to end, no dead code"
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-062, ADR-063, ADR-097, docs/adr/BACKLOG.md
**Invalidates:** None — `mrw instructions` with no flag prints exactly what it printed; `--core` is new
**Governs:** `internal/guide/guide.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `AGENTS.md`
**Enforced-by:** `internal/guide/core141_test.go::TestTheCoreIsEightShortRulesTheFullContractStillCarries`
**Served-path change:** `mrw instructions --core` prints eight rules in under 300 words, the part of the contract a session acts on, and its first line says `mrw instructions` prints the whole of it. `mrw instructions` without the flag is unchanged, byte for byte, apart from two sentences at its END: a pattern may start with `(?i)` to ignore case, and `mrw instructions --core` prints the eight rules.

## Context

The 2026-10-09 survey (14 sessions, as the author's synthesis of the replies recorded) put one thing under "guidance": the text agents are handed about mrw is about 900 words for roughly eight rules, and what is long is history — "check `mrw --version`… rebuild", an MCP-versus-CLI account of why — which 8 and 5 sessions said they never acted on. `mrw instructions` prints 1,306 words (measured 2026-10-10 on this tree: `mrw instructions | wc -w`), because it is the whole contract a caller with only the binary is entitled to (ADR-062, ADR-063), and it should stay that: ADR-063 made it complete so a caller with nothing else can learn the format.

**What is wrong is the lack of a short thing to hand an agent**, not the long thing's length. A person writing the line that goes into an agent's standing instructions (a `CLAUDE.md`, a hook, a skill header) has two choices today: paste the 1,306 words or write their own eight rules and let them drift from the binary. A short form the binary itself prints, held to the full text by a test, is the third.

**The core is not a summary of AGENTS.md.** It is the rules whose absence the field data shows costing a turn: read before write per line and read on past a multi-line body; a plan applies whole or not at all and a refusal is read, not forced; `anchor=` and the `body=` count; the exit codes and the pipe trap; finding with `--grep` and the spellings sessions guessed wrong (`(?i)`, `--stat` as a file list, `--max-cols`, `-N`); a new file in one command; where `-C` goes. History, per-version change notes and platform traps stay in the full text.

**Audit of the class** — *a place that tells an agent how to use mrw and can drift from the binary*: `mrw read --grep 'guide\.|instructions' --exclude '*_test.go' cmd internal` names `guide.Shared`, `guide.CLI` and the MCP handshake document (`internal/mcp/instructions.go`); AGENTS.md, README and the centralised skill are prose copies maintained by hand. The core is a fourth, code-held and tested against the second; the prose copies are out of this record.

## Existing Primitives Audit

- **`guide.CLI()` and `guide.Shared()`** (ADR-062, ADR-063) — the full contract; the core is checked against it rather than generated from it.
- **`TestEveryReadFlagIsTaughtByInstructions`** (ADR-063) — the precedent: a test that reads `readCmd().Flags` so the text cannot name a flag the binary lacks.
- **The `instructions` command** (ADR-097) — takes no arguments; one boolean flag is added.

## Decision

1. **`mrw instructions --core` prints `guide.Core()`**: eight numbered rules, under 300 words, whose first line names the full form. It takes no arguments, as `instructions` does not.
2. **`guide.Core()` is held to the binary and to the full text by test.** Every `--flag` it names is a flag of `read`, `write` or the root command; its rule on the exit codes, the `anchor=` rule, the `--create` and `--max-cols` and `(?i)` spellings each appear in `guide.CLI()` as well, so the full text cannot lose a rule the core teaches.
3. **`mrw instructions` without the flag is the full contract**, byte-identical to before except two sentences appended at its end (not its start: the first lines are read by scripts, and the MCP handshake shares them): `A pattern is a Go regular expression: start it with (?i) to ignore case.` and `mrw instructions --core prints the eight rules that matter most.` The first makes the core's `(?i)` rule true of the full text too. A script that read the whole output is unaffected.

## Alternatives Considered

- **Make the short form the default and add `--full`** — rejected: it changes what a caller with only the binary is handed (ADR-063's promise) and touches a dozen contract rows and tests that read `instructions`; the survey's complaint is about what gets pasted into standing instructions, not about the binary's own full text.
- **Generate the core from the full text** — rejected: choosing the eight rules is the work; a generator would pick by length or position, and the point is that a person chose them.
- **Shorten AGENTS.md and the skill instead** — rejected here: they are prose copies maintained by hand and the skill is a verbatim mirror (its provenance says so); the short form belongs where a test can hold it.
- **Leave it** — rejected: agents were handed 900 words and acted on a fraction.

## Component / Boundary Impact

`internal/guide` (one function), `cmd/mrw` (one flag). No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw instructions --core` | new flag | T1 | CLI callers, whoever writes standing instructions |
| `guide.Core()` | new function | T1 | `cmd/mrw` |
| `mrw instructions` | one sentence at the start | T1 | CLI callers |
| `scripts/contract.sh` | §242 | T1 | CI Linux |
| `AGENTS.md` | the `mrw instructions` bullet names `--core` | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** the binary prints the thing to paste, and a test keeps it true; the long text keeps its job.
- **Negative:** a fourth copy of the rules to keep honest; the tests that hold it are the cost.
- **Neutral:** `mrw instructions` and the MCP handshake are not shortened; the MCP document is already bounded at 4,096 bytes.

## Out of Scope

- Shortening `mrw instructions`, AGENTS.md or the skill (permanent: boundary: the full text is the contract ADR-063 promised; the prose copies are hand-maintained mirrors)
- A staleness line on `mrw version` (deferred: docs/adr/BACKLOG.md — it needs a way to learn the latest release, which mrw does not have)
- The user's own `CLAUDE.md` block (permanent: boundary: it is not in this repository; the core is what to put there)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the core names a flag the binary lacks | Low | an agent is sent wrong | `TestTheCoreNamesOnlyFlagsTheBinaryHas` reads the flags from the commands |
| the full text loses a rule the core teaches | Low | the core is the only place left | `TestTheCoreIsEightShortRulesTheFullContractStillCarries` |
| the core grows past a page | Medium | it becomes the 900 words again | the same test bounds it at 300 words |

## Rollback

Revert T1: the flag is gone and `mrw instructions` loses one sentence. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the deferred item above.
