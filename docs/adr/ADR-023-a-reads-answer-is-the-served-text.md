# ADR-023: A read's answer is the served text, and no envelope may stand in for it

**Status:** Accepted
**Accepted:** 2026-09-05 by M — *"Ok close these"*, given to the four open items of the served-size curve, under the standing instruction to fix the real defects first; the first open item, the MCP delivery arm, is what found this one.
**Date:** 2026-09-05
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-011 (the record whose T2 declared `outputSchema` and `structuredContent` on both tools; this retires that half for `mrw_read`), ADR-002 (the ledger promise this transport inverted), ADR-010 (the transport), ADR-014 (paged reads — their result shape changes here too), ADR-017 (the index — likewise), ADR-020 (the MCP delivery arm it left open cannot run until this ships), issue #109 (the measurement)
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/conformance_test.go::TestAReadResultCarriesNoStructuredContent`
**Invalidates:** ADR-011 — the clause of its Decision and T2 reading "every declared `outputSchema` validates a REAL response" and "`content[1]` is the serialized `structuredContent`" as they apply to `mrw_read`; `mrw_write` keeps both
**Served-path change:** over MCP, every `mrw_read` result that carries a receipt — a served read, a page, an index, a grep that matched nothing — carries the served text or report in `content[0]` and the JSON receipt in `content[1]`, and NO `structuredContent`; a bare refusal (`exclude` without `grep`, an unparseable spec) stays one text block with no receipt, as it was; `tools/list` declares no `outputSchema` for `mrw_read`. A Claude Code session that calls `mrw_read` now receives the lines it asked for. The CLI is unchanged.

## Context

**Over Claude Code, a successful `mrw_read` handed the model the receipt and none of the lines.**
Measured 2026-09-05 on Claude Code 2.1.261 (macOS), server at `e917310` whose `internal/mcp` differs
from `main` only in `instructions.go`, from three clients: this session's own `mcp__mrw__mrw_read`
call, a Haiku subagent's, and a fresh `claude -p --model haiku` session. The server sends two text
blocks — the served text, then the serialized receipt — and a `structuredContent` object holding the
same receipt (ADR-011 T2). The model received exactly `{"observed":{…},"problems":0}`. On the error
path — an UNREADABLE spec, a refused oversize read, `isError: true` — the model received both text
blocks. Two issues in the host's tracker describe the behaviour, one closed as acknowledged:
anthropics/claude-code#55677 ("tool result content[].text dropped from model when structuredContent
is also present") and #15412.

**Which field the host keys on was measured, not read off the issue.** Three builds of this server
were driven from a fresh `claude -p` session each, 2026-09-05, against the same two-line spec:
`structuredContent` dropped and `outputSchema` kept — the model quoted the served line;
`outputSchema` dropped and `structuredContent` kept — the model reported metadata and no lines;
both dropped — the model quoted the served line. The host renders `structuredContent` in place of
`content` whenever it is present on a non-error result, and `outputSchema` plays no part.

**Why this is a defect in mrw and not only in the host.** `readTool` records the served span in the
read-before-modify ledger before the result leaves the server (`internal/mcp/tools.go`,
`seen.Record`). Over this host the lines were recorded as seen by a model that never saw them, and a
following `mrw_write` to those lines was licensed. That is ADR-002's promise — mrw will not edit a
line it has not shown you — inverted by the envelope, and it went unmeasured through eleven
readings of the served-size curve because every reading delivered the served text through a file
reader or a Bash result and none through the MCP tool it was written for. ADR-020's open item, the
MCP delivery arm, is what found it: the cells were staged and the first probe call returned no text.

**What ADR-011 T2 rested on.** Its Context said "`structuredContent` is consumed in practice: a peer
session measured reading a field out of `result.structuredContent` from a live client call, so both
halves are load-bearing." That was true of a raw client reading the receipt. It said nothing about
a host that shows the model one half and not the other, and for `mrw_read` the half it hides is the
whole answer.

## Existing Primitives Audit

- **`result()` in `internal/mcp/tools.go`:** assembles both blocks and the `structuredContent` from
  one marshal, for both tools. **Reshaped:** it gains a form that omits `structuredContent`; the
  one-marshal discipline stays, because `content[1]` still has to equal what the receipt would have
  been. `pagedResult` and `indexResult` build their own envelopes and are reshaped the same way.
- **`tools()` in `internal/mcp/mcp.go`:** the descriptor set. **Reshaped:** `mrw_read` declares no
  `OutputSchema`; `mrw_write` keeps `writeSchema()`. `readSchema()` and its description table stay,
  because the conformance test that every property is described still reads them — they describe
  the receipt a caller parses out of `content[1]`, and that receipt has not changed shape.
- **`internal/mcp/conformance_test.go`:** ADR-011's Enforced-by validates every declared schema
  against a real response. **Kept**, narrowed to the tools that declare one; the new Enforced-by sits
  beside it.
- **`scripts/contract.sh` §41 and the rows that read `structuredContent` off an `mrw_read` result
  (§45–§51 region, §58):** **reshaped** to read the receipt from `content[1]`, which is where a host
  that shows `content` finds it.

## Decision

**1. `mrw_read` returns no `structuredContent`, on any result.** A served read, a page (ADR-014), an
index (ADR-017) and a grep that matched nothing all carry the served text or the report in
`content[0]` and the serialized receipt in `content[1]`, and nothing else; a bare refusal — one
that never had a receipt, such as `exclude` without `grep` — stays one text block, as before. The receipt's shape is unchanged — `observed`,
`problems`, and `next_read` or `matches`/`index`/`next_index` where they applied before — so a caller
that parsed `structuredContent` parses `content[1]` and gets the same object. A host that renders
`structuredContent` in place of `content` has nothing to render in its place, and shows the answer.

**2. `mrw_read` declares no `outputSchema`.** The 2025-06-18 tools specification says a tool that
declares one MUST return `structuredContent` that validates against it; a tool that returns none
must declare none. The property descriptions written for the receipt stay in the source, because
`content[1]` still carries that object and a reader of the code should find its fields explained.

**3. `mrw_write` is unchanged.** Its answer IS the receipt — `apply.Result`, one verdict per hunk —
and a host that shows the receipt in place of the report shows the answer. The measured host does
exactly that, and ADR-011's schema, description table and conformance test stay for it.

**4. The measured host behaviour is recorded here, not depended on.** This record is not a
workaround for a host bug that a future host version removes. Without `structuredContent`, every
host that shows `content` shows the served text — the ones that rendered both, the one that rendered
only the structured half, and any that renders only `content` — and the receipt is still there as
JSON. A host fix makes the server's behaviour no worse; the host bug made the old behaviour
unusable. The decision would be wrong only if a host existed that shows `structuredContent` and NOT
`content[0]` when `structuredContent` is absent, which is a host that shows nothing for any tool
returning plain text; none is known.

**What would falsify the decision as shipped:** a fresh `claude -p` session against the built
binary reporting no served lines for a two-line spec. That run is T1's sign-off and is recorded in
its Verification Log with the Claude Code version it was taken against.

**Go/no-go, checked during execution:**

- **Engine untouched:** `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`,
  `internal/check`, `internal/state` and `cmd/mrw/main.go` stay byte-identical against the
  merge-base; the change is in `internal/mcp` only.
- **`go.mod` declares exactly one requirement.**
- **`gofmt -l .` empty and `go vet ./...` clean**, in the fence.

## Alternatives Considered

- **Put the served text INTO `structuredContent` as well** (`"text": …`), keep the schema. Every
  host shows the lines; the spec's SHOULD is still met. Rejected: the served text then travels
  twice — a 200,000-character read becomes a 400,000-character response against a declared
  200,000 limit — and the copy the measured host shows is JSON-escaped, so the model reads
  `\n` and `\"` where the CLI shows lines. The delivery this record exists to fix would arrive
  worse than the Bash one.
- **Drop `outputSchema` only, keep `structuredContent`.** Measured 2026-09-05 (variant B above): the
  model still received no lines. Rejected by measurement.
- **Drop `structuredContent` only, keep `outputSchema`.** Measured to work (variant A). Rejected
  because a declared schema with no structured content to validate violates the specification's MUST,
  and the conformance test that validates every declared schema against a real response could not be
  kept honest.
- **Wait for the host fix.** #55677 is closed as acknowledged; this session's 2.1.261 still shows it,
  and a subagent inherits it. Rejected: mrw's promise is measured against the hosts it runs in, and
  ADR-002 is inverted in the one it was registered for today.
- **Emit `structuredContent` only when the host is not Claude Code**, read off `clientInfo`.
  Rejected: a server that answers differently by client name is a server whose contract cannot be
  asserted by a contract row, and the `-p` measurement shows the plain answer is right for every
  host.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | the envelope of both tools' results, and what `tools/list` declares | Yes — unchanged owner, narrower promise for one tool |
| `internal/*` engine, `cmd/mrw` | Untouched | Untouched |

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `tools/call` result of `mrw_read` (served, paged, index, grep-no-match) | `structuredContent` removed; `content[0]` served text or report, `content[1]` the serialized receipt, unchanged in shape; a bare refusal stays one text block | `internal/mcp` | any MCP host; `scripts/contract.sh` rows that read the receipt |
| `tools/list` entry for `mrw_read` | `outputSchema` removed; `title`, `annotations`, `_meta`, `inputSchema` unchanged | `internal/mcp` | any MCP host |
| `initialize.instructions` and the `mrw_read` description | say where the receipt is (`content[1]`) | `internal/mcp` | a model reading the surface |
| `mrw_write` | unchanged | — | — |

## Inter-task Contracts

None — one task.

## Implementation

One task, `tasks/T1-the-read-result-drops-its-structured-envelope.md`: the failing tests, the
server change, the conformance and contract rows moved to `content[1]`, the wording, and the `-p`
sign-off against the real host.

## Consequences

- **Positive:** over Claude Code, the tool the MCP surface exists for serves what it was asked for,
  and the ledger licenses only lines a model was shown.
- **Positive:** the MCP delivery arm of ADR-020's curve can be run.
- **Negative:** a host that validated `mrw_read`'s result against its declared schema loses that
  check for one tool. The receipt is the same object at `content[1]`, parseable, and `mrw_write` —
  the tool whose receipt carries verdicts — keeps its schema.
- **Negative:** a caller reading `result.structuredContent` off `mrw_read` (the peer's raw client in
  ADR-011's Context) must read `content[1]` instead. The two were byte-identical by construction.
- **Neutral:** `mrw_read`'s response shrinks by one copy of the receipt.

## Out of Scope

- The host's own handling of `structuredContent` (external: anthropics/claude-code: https://github.com/anthropics/claude-code/issues/55677)
- `mrw_write`'s envelope (permanent: boundary: Decision 3 — its answer is the receipt, and the measured host shows the receipt)
- Measuring the same behaviour in Claude Desktop and other hosts (deferred: docs/adr/BACKLOG.md — the "ADR-023: other hosts" entry)
- The MCP delivery arm of the served-size curve (deferred: docs/adr/ADR-020-the-served-size-curve-is-measured-not-asserted.md — its open follow-up; runnable once this ships)
- Changing any CLI behaviour (permanent: boundary: ADR-010's go/no-go, still binding — the engine directories stay byte-identical)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A contract row still reads the receipt from `structuredContent` and goes green on a KeyError-free path | Low | High — a red row reads as coverage | every consumer is moved in T1 and §61 asserts the key is ABSENT on a served, a paged and an index result |
| The host fix lands and a future reader wonders why the envelope is bare | Med | Low | Decision 4 records why the bare envelope is right for every host, not only the broken one |
| A reader of the code changes `mrw_write` to match | Low | Med | Decision 3 and the conformance test that still validates `mrw_write`'s schema |

## Rollback

Revert the commit. `mrw_read` regains `structuredContent` and `outputSchema`; no ledger entry
changes meaning, no state format moves, and a tree read under this record is served identically by
the previous binary — through every host except the one this record was measured on.

## Follow-ups

- [ ] Run ADR-020's MCP delivery arm (reading 12) against the shipped binary, from a fresh session
      so the server is this build.
