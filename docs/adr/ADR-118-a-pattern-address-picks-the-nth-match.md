# ADR-118: a pattern address picks the Nth match

**Status:** Accepted
**Accepted:** 2026-10-02 by Zy — "All matches up to N served (Recommended)", the answer to "occurrence=N — what makes picking the Nth match safe?", on the refreshed gap list of 2026-10-02 and the plan he approved the same day; and again on the assessment of 2026-10-02, whose first cost names it. The record's text was drafted after that answer and not shown to Zy before execution: this line is the answer, not a review of these words.
**Date:** 2026-10-02
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-013, ADR-030, ADR-036, ADR-087
**Invalidates:** ADR-013 — only its rejection of "An `occurrence=N` guard to disambiguate" and its Out of Scope entry deferring it; the exactly-once rule stands for a start pattern without `occurrence=`
**Governs:** `internal/plan/plan.go`, `internal/apply/apply.go`, `internal/seen/seen.go`, `internal/writer/writer.go`, `internal/refusal/refusal.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/curve/score.go`, `internal/guide/guide.go`, `AGENTS.md`, `README.md`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/occurrence118_test.go::TestAnOccurrencePicksTheNthStartMatch`
**Served-path change:** A plan hunk whose start is a pattern may carry `occurrence=N`: the Nth line the start pattern matches, counted from 1 over the original file, provided every match before it has been served. Without it a start pattern must match exactly once, as before.

## Context

A start pattern that matches twice is refused, naming the lines (ADR-013). Two functions of one shape, or a repeated block, cost the caller another read to learn line numbers, then a plan by number. The assessment of 2026-10-02 names this the first cost a model pays beyond the edit.

ADR-013 rejected `occurrence=N` because it "makes a plan depend on the ORDER of matches in a file, which changes when unrelated code moves": the address would resolve somewhere else while still looking right. That objection assumed the caller might count matches in a version of the file they had not seen. Zy's rule removes the assumption: every match up to and including the Nth must have been served to the caller in the file's current version. The ledger is keyed by the file's sha, so a caller who has seen the current first N matches has counted them in the file as it is, and a file that changed since is refused before any address resolves.

**Audit of the class** — *a place that builds an `apply.Input` from a `plan.Hunk`*: `mrw read --grep 'RelEnd: *h\.Addr\.RelEnd' cmd internal` — `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/curve/score.go`, and two test helpers (`internal/ingest/applypatch_test.go`, `internal/mcp/tools_test.go`); and `internal/apply/apply.go`'s own Input-to-hunk copy. Each carries the new field; `TestEveryBuilderCarriesOccurrence` fails when one does not. The ingest compilers emit no pattern, so no `occurrence=`.

## Existing Primitives Audit

- **`matchLines`** (`internal/apply/apply.go`) — every line a pattern matches, the list the several-matches refusal already names.
- **`Observation.Covers`** — the per-line ledger check `covered()` uses.
- **`refusal.Kind`** (ADR-087) — kinds the parser and engine share for mirrored rules.

## Decision

1. **`occurrence=N`** is a hunk option, N a positive integer. It is legal only when the start is a pattern; on a line address, a `create`, an `unlink` or a `rename` it is refused by the parser and, mirrored, by the engine (ADR-030), with the refusal kind `occurrence-address`.
2. **N counts start-pattern matches over the whole original file**, from 1 — the list the several-matches refusal prints. N past the count is refused, naming the matches.
3. **Every match before the Nth must have been served** — shown to the caller in this version of the file. When the ledger applies (no `--force`), each earlier matching line must have been served by a read; otherwise the hunk is refused, naming the unread matches. A file mrw has just written is wholly licensed for edits (ADR-005) but counts none of its lines as served for this rule — Zy, on the review of #320: "strict: always served" — so the ledger records which wholes came from a write (`seen.Observation.Written`, with the lines read since in `Shown`), and moves to `#mrw-seen v4`, discarding v3 records once since they cannot say which kind they were. The Nth line itself meets the ordinary per-line check.
4. **The end pattern is unchanged**: the first match at or after the chosen start (ADR-036).
5. **The several-matches refusal names the remedy**: "narrow it, address by line number, or pick one with occurrence=N".

## Alternatives Considered

- **`occurrence=N` with no ledger rule** — rejected: it is ADR-013's objection unanswered, a count taken in a version the caller may not have seen.
- **First match on ambiguity** — rejected as ADR-013 rejected it: a silent choice with a receipt saying `ok`.

## Component / Boundary Impact

Owns `internal/plan`, `internal/apply` and `internal/seen` (engine packages; `internal/seen` for the written-versus-shown record and the v4 header). `internal/refusal`, `internal/writer`, `cmd/mrw`, `internal/mcp`, `internal/curve`, `internal/guide` are not engine. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| plan grammar | `occurrence=N` | T1 | every plan surface |
| `apply.Input.Occurrence` | the field, copied by every builder | T1 | CLI, MCP, curve |
| `refusal.OccurrenceAddress` | a shared kind | T1 | parser, engine |
| guide, MCP description, AGENTS.md, README | say so | T1 | readers |
| `scripts/contract.sh` | §218 | T1 | CI Linux |

No receipt key is added.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a repeated site is edited from the read that found it, without a second read for numbers.
- **Negative:** a plan with `occurrence=` is refused when any earlier match was not served, so a caller who read only the site must read the earlier matches too.
- **Neutral:** without `occurrence=` nothing changes.

## Out of Scope

- `occurrence=` on a read address (permanent: boundary: a read already serves every match; picking one is a write's question)
- A negative or "last" occurrence (permanent: boundary: counting from the end changes when lines are appended, which the ledger rule does not cover)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| an earlier match the caller saw in an older version | Low | High | the ledger is per sha: a changed file is refused before resolution |
| a builder that drops the field | Medium | High | `TestEveryBuilderCarriesOccurrence` |

## Rollback

Revert T1. No receipt key and no state format change.

## Follow-ups

- None — the record carries no open follow-up.
