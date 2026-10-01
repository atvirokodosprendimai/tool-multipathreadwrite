# ADR-111: A receipt field, once shipped, stays

**Status:** Accepted
**Accepted:** 2026-10-01 by Zy — "work on the deffered ones", on the ADR-108 deferrals in `docs/adr/BACKLOG.md` "From ADR-108", of which B4 is this record's scope. The record's text, including the promise it makes, was drafted after that instruction and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-10-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-023, ADR-054, ADR-072, ADR-092, ADR-100, ADR-102
**Governs:** `docs/receipts.txt`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `AGENTS.md`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/receipts111_test.go::TestNoShippedReceiptFieldDisappears`
**Served-path change:** None to any output: two anonymous receipt structs and one map become named types with the same JSON. What changes is a promise, written down and held by a test: every key a JSON receipt carries keeps its name, type and meaning; keys are only added.

## Context

Receipt fields have only ever been added (ADR-054's stats keys, ADR-072's `error`, ADR-092's `then`, ADR-102's
`partially_applied`, ADR-105's `left_behind`), but nothing says a caller may rely on that, and nothing fails when a
field is removed or renamed (B4 in BACKLOG "From ADR-108"; the 2026-10-01 Codex design review). A script or an
agent parsing `--json` breaks silently on such a change: the key it reads is absent, which reads like zero.

**Audit of the class** — *a JSON document mrw prints for a caller to parse*: `mrw read --grep 'json\.NewEncoder|
readResult\(|result\(writeReceipt' --exclude '*_test.go' cmd internal` — the CLI's `write --json` (`receipt`),
`check --json` (`checkReceipt`) and its refusal, `stats --json`, and MCP's `mrw_write` (`writeReceipt`) and
`mrw_read` (the receipt in `content[1]`, ADR-023). The MCP protocol envelope (`tools/list`, `initialize`) is the
protocol's, held by the golden in `internal/mcp/testdata`, and not a receipt.

## Existing Primitives Audit

- **`internal/mcp/testdata/legacy_golden.jsonl`** — pins the whole MCP byte stream of one era, and is regenerated on
  every additive change; it cannot tell an addition from a removal, so it does not hold this promise.
- **`fieldsOf`** (`internal/mcp/schema.go:60`) — the schema generator's reflection over the same structs; its
  traversal is the model for the test's.

## Decision

1. Every key a receipt above carries keeps its name, its JSON type and its meaning across releases. Keys are only
   added. A removal, a rename or a change of meaning is a breaking change: it waits for a major version, and the
   release before it names the key as going in its tag message and the README's Status.
2. `docs/receipts.txt` lists every key path of every receipt, one per line (`write hunks[].status`). Two tests —
   the CLI's in `cmd/mrw`, MCP's in `internal/mcp` — reflect the receipt types and require the file and the types
   to agree: a path in the file that a type lost fails (a removal), and a path in a type that the file lacks fails
   (an addition must be written into the file, where review sees it).
3. The stats receipt, the check refusal and `mrw_read`'s receipt become named types, so the tests can reflect them.
4. Contract §210 checks the built binary's `write --json` keys against the file.

## Alternatives Considered

- **A `schema_version` field** — rejected: it tells a caller something changed after it already broke; the promise
  prevents the break.
- **Freeze the golden file** — rejected: it holds bytes, so every addition breaks it, and a regenerated golden
  hides a removal as easily as an addition.

## Component / Boundary Impact

`cmd/mrw` (T1) and `internal/mcp` (T2), neither an engine package. Every engine package stays byte-identical;
`go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `docs/receipts.txt` | the shipped key paths | T1 | T1, T2, §210 |
| CLI receipts | `statsReceipt`, `checkRefusal` named; JSON unchanged | T1 | callers |
| `mrw_read` receipt | `readReceipt` named; JSON unchanged | T2 | MCP hosts |
| `scripts/contract.sh` | §210 (T1) | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `docs/receipts.txt` | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2.

## Consequences

- **Positive:** a caller may rely on a key it reads today; a removal cannot land unnoticed.
- **Negative:** every new receipt field touches `docs/receipts.txt` too.
- **Neutral:** no output changes.

## Out of Scope

- The other ADR-108 robustness items, B1, B2 and B5 (deferred: `docs/adr/BACKLOG.md` "From ADR-108")
- The human-readable output (permanent: boundary: text is for reading; the JSON receipts are the parsed contract, and ADR-072 makes every `--json` refusal one)
- The MCP protocol envelope (permanent: boundary: the protocol's own fields, pinned byte for byte by the golden)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a key's meaning changes while its name stays | Low | Medium | the promise names meaning; review, not a test, holds it |

## Rollback

Revert T1–T2. No output change.

## Follow-ups

- None — the record carries no open follow-up.
