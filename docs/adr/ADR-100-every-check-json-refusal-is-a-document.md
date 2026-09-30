# ADR-100: Every `mrw check --json` refusal is a document

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — offered "make `mrw check --json` refuse in JSON" as a follow-up to ADR-099's kept advisories, Zy answered "fix them, do this json suggestion". "Them" is read as both advisories kept on ADR-099 (the plain-text refusal under `--json`, and the `--full` refusal answering before ADR-095's depth refusal). The record's text was drafted after that answer and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-069, ADR-072, ADR-092, ADR-095, ADR-099
**Governs:** `cmd/mrw/main.go`, `scripts/contract.sh`, `docs/adr/BACKLOG.md`
**Enforced-by:** `cmd/mrw/json100_test.go::TestEveryCheckRefusalIsAJSONDocumentUnderJSON`
**Invalidates:** ADR-099 Decision 1, narrowly — "nothing on stdout" for `mrw check --full PATH` still holds without `--json`; under `--json` stdout now carries the refusal document. Otherwise none — checked. ADR-092 T4's Out of Scope left the refusals "unrelated to steps" plain as a boundary of ADR-092 ("is ADR-072's to widen"); this record widens them, and contract §187's "no step asked prints no document" row changes with it. ADR-072 T3 made every `write --json` refusal after the plan is named a document, and this is the same rule for `check`. ADR-095's depth refusal keeps its words and its exit code; only its turn changes.
**Served-path change:** under `mrw check --json`, five refusals that printed only a message on stderr now also print one JSON document on stdout, `{"error": "<the same message>"}`, still exit 2: an argument with edge whitespace (ADR-069), `--full` with a PATH (ADR-099), a working set that cannot be read, a refused scope (a path outside the root or not there), and a check whose log cannot be created with no step asked. At `MRW_STEP_DEPTH` 8 every `mrw check` form that reaches the Action now answers with the depth refusal, where `check --full x.go` and `check 'x '` answered with their own usage refusal. Otherwise, without `--json`, every message, stream and exit code is unchanged.

## Context

**What was observed** (2026-09-30, `cmd/mrw/main.go` at `7537718`):

1. `checkCmd`'s Action already has a `refuse` helper that prints `{"error": …, "then": …}` under `--json`
   (`:2027-2039`), used by the depth refusal, a bad harness and a bad step (ADR-072, ADR-092 T4). Five
   earlier or later refusals bypass it and return `cli.Exit(…, exitUsage)` directly: `refusePaddedArgs`
   (`:2004`), `--full` with a PATH (`:2013`), `iter.Load` (`:2018`), and `check.Run`'s error when no command
   was chosen or no step was asked (`:2065-2066`). Under `--json` those print nothing on stdout, so a
   consumer that parses stdout reads an empty document for some refusals and a document for others.
   `cmd/mrw/thenrefusal_test.go:112` pins one of them as plain, "as before".
2. The depth refusal (ADR-095) sits after `refusePaddedArgs`, the `--full` refusal and the working-set load,
   so at depth 8 `check --full x.go` answers "--full runs the whole project" (the advisory kept on ADR-099,
   2026-09-29). ADR-095's comment at `:2040` says a check is "refused in every form" at the limit.

**Audit of the class.** The class is *a refusal under `--json` that prints no document*. Enumerated
2026-09-30 by `mrw read --grep 'Name: +"json"' cmd/mrw/main.go` (three commands carry the flag: `write`,
`check`, `stats`) and reading each Action's `cli.Exit(…, exitUsage)` returns:
- `check`: the five above — **5**, all in scope.
- `write`: five usage refusals before the plan is named (`refusePaddedArgs`, `--check` with `--dry-run`,
  `--check` with `--no-check`, a negative `--echo-pad`, a second plan file; `:1065-1090`) — left out: ADR-072
  drew its line "after the plan is named" and these contradict flags before any receipt exists. Deferred to
  BACKLOG.
- `stats`: two state-read failures (`:474`, `:483`) — left out: `stats --json` is a report of this
  checkout's tally, not a verdict a caller branches on. Deferred to BACKLOG.
- A flag urfave rejects, and an attached flag value with edge whitespace that `main` refuses in
  `refusePaddedFlagValues` (`:279`, ADR-069 T5) — left out: both run before any command's flags are parsed,
  so no `--json` has been read; `mrw check --json --then-sh='true '` exits 2 with nothing on stdout (probed
  2026-09-30), and §193 pins that boundary through the binary.
- The exit 2 after a receipt, when no check could run (`:2084-2092`) — not a refusal before a receipt: the
  Action has already printed its one document, `{"ran": false, "skipped": …, "exit_code": 0}` (probed
  2026-09-30 in a tree with no harness and no `go.mod`). Routing it through `refuse` would print a second
  document. It stays, T1's test asserts it is exactly one document, and a consumer reads `ran` before
  `exit_code`; an `exit_code` of 0 beside `"ran": false` is deferred to BACKLOG.

## Existing Primitives Audit

- **`refuse`** in `checkCmd` — reused as the one exit for every refusal; it already writes the document
  under `--json` and the steps report without it.
- **`check.DepthRefusal`** — reused, moved first.
- **`runSplit`, `runIn`, `stepsTree`** (cmd/mrw tests) — reused.

## Decision

1. Every refusal `mrw check`'s Action makes before it prints a receipt goes through `refuse`: under `--json`
   it prints one document on stdout, `{"error": "<message>"}` (with `then` when steps were named not_run, as
   ADR-092 already does), and the message on stderr, exit 2. Without `--json` nothing changes.
2. The depth refusal is the Action's first check: at `MRW_STEP_DEPTH` 8 every form of `mrw check` that
   reaches the Action answers with it, before an argument is judged or the working set is read.

## Alternatives Considered

- **Leave the pre-run refusals plain and document the split** — rejected: a `--json` consumer cannot tell
  "refused" from "printed nothing" without reading stderr, the failure ADR-072 fixed for `write`.
- **Also cover `write`'s pre-plan refusals and `stats`** — deferred, not rejected: Zy asked for `check`,
  and `write`'s line was drawn by ADR-072 on purpose. Both are receipted in BACKLOG.
- **Keep usage refusals ahead of the depth refusal** — rejected: at the limit a caller who fixes the usage
  meets the depth refusal next, and ADR-095 says every form is refused.

## Component / Boundary Impact

None — `cmd/mrw` only. No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw check --json` refusals | one `{"error"}` document on stdout, exit 2 | T1 | CLI callers parsing stdout |
| `mrw check` at depth 8 | the depth refusal answers first | T2 | nested callers |
| `scripts/contract.sh` | §193 (T1), §194 (T2); §187's no-step row now expects the document (T1) | T1, T2 | CI Linux |
| `docs/adr/BACKLOG.md` | `write` pre-plan and `stats` refusals under `--json`, and `exit_code` beside `"ran": false`, deferred | T1 | readers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `refuse` defined before every refusal | T1 | T2 | no — T2 moves one call inside it |

## Implementation

See `tasks/README.md`: T1 (every refusal a document), T2 (the depth refusal first).

## Consequences

- **Positive:** a `--json` consumer of `mrw check` reads exactly one document on every exit from the Action.
- **Negative:** a script that treated an empty stdout from `check --json` as "refused" now sees a document;
  its exit code is unchanged.
- **Neutral:** without `--json` nothing changes except which message answers at depth 8. A refusal before
  any command parses its flags still prints nothing on stdout.

## Out of Scope

- `write --json` refusals before the plan is named (deferred: `docs/adr/BACKLOG.md` "From ADR-100")
- `stats --json` state-read failures (deferred: `docs/adr/BACKLOG.md` "From ADR-100")
- A flag urfave rejects, and `refusePaddedFlagValues` in `main` (permanent: boundary: both run before any command's flags are parsed, so `--json` has not been read; ADR-078 keeps stdout empty there)
- The exit 2 after a `"ran": false` receipt (permanent: boundary: it follows the one document the Action already printed)
- `exit_code` 0 beside `"ran": false` in that receipt (deferred: `docs/adr/BACKLOG.md` "From ADR-100")
- MCP (permanent: boundary: `mrw_write` already answers every refusal in its JSON result)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a consumer reads `exit_code` from any document on stdout | Low | Med | the refusal document has no `exit_code`; contract §193 and the existing rows at §15 and §86 assert it; the `"ran": false` receipt's `exit_code` is deferred to BACKLOG |
| a refusal prints the document twice or on stderr | Low | Low | T1's test decodes stdout as exactly one document and compares its `error` with the returned error |

## Rollback

Revert T1 and T2. No state, format or exit code for any other input changes.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `cmd/mrw/json100_test.go::TestEveryCheckRefusalIsAJSONDocumentUnderJSON` in T1's commit.
