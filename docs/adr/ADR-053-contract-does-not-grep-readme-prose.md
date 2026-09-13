# ADR-053: Contract does not grep README for tutorial phrases

**Status:** Accepted
**Accepted:** 2026-09-13 by M — *"drop prose greps"*
**Date:** 2026-09-13
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-010, ADR-012, ADR-026, ADR-031, ADR-037, docs/adr/BACKLOG.md
**Governs:** `scripts/contract.sh`, `internal/adversarial/contract_prose_test.go`
**Enforced-by:** `internal/adversarial/contract_prose_test.go::TestContractDoesNotGrepReadmeForTutorialPhrases`
**Invalidates:** ADR-010 — T3's clause that the documented host config is extracted from a README heading and `mcpServers` JSON fixture (§39). ADR-026's README tutorial-phrase greps in §64 are withdrawn the same way; its binary pairing and AGENTS.md / `--help` / MCP-wire greps stand.
**Served-path change:** None — callers see the same binary. A README tidy of a heading or tutorial sentence is no longer a red `contract.sh` row.
**Notes:** Keep **AckRule** (`TestEverySurfaceCarriesTheOneRule` — teaching must match the server). Keep §75 `Shared()` through the binary. 4096 stays. Do not gut `BESTPRACTICES.md` or `docs/measure.md`.

## Context

**The class this record governs.** Every live `scripts/contract.sh` assertion whose verdict is a grep (or parse) of `README.md` for a heading, a casing, or a tutorial phrase. Enumerated 2026-09-13 on `origin/main` `f5ac1ca` with

```
rg -n 'grep .*README\.md|cat README\.md' scripts/contract.sh
```

Five live members: §39's `python3 - "$(cat README.md)"` extract of `### Use it from an MCP host` plus the first ` ```json ` / `mcpServers` block (four assertions cascade from that heading); §64's four README greps (`^A range is .*A,+N`, `and \`A,+N\` is the line \`A\` plus the \`N\` lines AFTER it`, `A read CLAMPS a relative end at the last`, `clamps.*exactly as \`12-9999\``).

Members left out, and why: §55 hook fixtures that happen to be named `README.md` (they drive the hook, not README prose); `TestEverySurfaceCarriesTheOneRule` / `AckRule` (teaching must match the server, byte for byte); §75 `mrw instructions` / `Shared()` through `$MRW`; §43's assertion that the handshake does **not** name `README.md`; `TestEveryDocumentedReplaceCarriesItsAnchor` (parser on `@@` plan headers, not tutorial phrases); AGENTS.md greps in §64 (not README).

PR #169 went red on the `test` / Contract step (run 34754261323 on `a80a4e4`) for those seven README assertions — not `go test`. A tidy of a heading looked like a product break. The restore (`1fc3828`, squash `f5ac1ca`) put the strings back. M then said *"drop prose greps"*.

No BACKLOG deferral is pulled in. Existing debt named by `adr-debt` is unrelated follow-ups.

## Existing Primitives Audit

- **`$MRW` / `m` in `scripts/contract.sh`.** Reused. §39's useful half already drives `--help` and `mrw mcp`. After this record it does only that, with `mcp` named from the binary rather than parsed out of README.
- **§64's binary pairing.** Reused unchanged: read clamps, write refuses, AGENTS.md / `--help` / MCP wire still gated.
- **§75 `mrw instructions`.** Unchanged. Shared sentences are asserted against the shipped binary.
- **`mcp.AckRule` + `TestEverySurfaceCarriesTheOneRule`.** Unchanged. README still carries the acknowledgement sentence because the server enforces that sentence.
- **`TestEveryDocumentedReplaceCarriesItsAnchor`.** Unchanged. That is a parser check on documented plans, not a tutorial-phrase freeze.

## Decision

`scripts/contract.sh` does not grep `README.md` for headings, casing, or tutorial phrases. A README without `### Use it from an MCP host` must not fail `contract.sh` or `go test`, except `AckRule` if that sentence is still in the file.

§39 is retargeted onto the binary: `mrw --help` lists `mcp`, and `mrw mcp` starts and advertises `mrw_write`. The heading / JSON-as-README-fixture rows are retired with a comment pointing here.

§64's four README greps are retired with a comment pointing here. The rest of §64 stays.

§75 stays. `AckRule` stays. `maxInstructionsChars` stays 4096.

The pin is a Go test that fails if a live (non-comment) line of `scripts/contract.sh` still asserts those fossils, or still parses `mcpServers` out of `README.md`.

## Alternatives Considered

- **Keep the greps and freeze the README heading forever.** Rejected: that is what made a tidy look like a product break (PR #169).
- **Drop AckRule's README check too.** Rejected: M said keep AckRule — teaching must match the server.
- **Drop §75 and assert Shared() only in a package test.** Rejected: a unit test cannot prove the shipped binary prints it. §75 exists for that.
- **Retire all of §39.** Rejected: the binary half (help lists `mcp`, `mcp` starts) is a real product promise, not prose.

## Component / Boundary Impact

None — internal to the contract harness and one pin test. No new bounded context. No architecture doc exists; none is required.

C4: same CLI / MCP containers. The trust boundary is unchanged: the binary is still the source of truth (ADR-003, ADR-037).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `scripts/contract.sh` §39 | README parse retired; `--help` + `mrw mcp` stay | T1 | CI |
| `scripts/contract.sh` §64 | four README greps retired; binary / AGENTS / help / wire stay | T1 | CI |
| `TestContractDoesNotGrepReadmeForTutorialPhrases` | live contract.sh must not assert the fossils | T1 | `adr-verify`, CI |

No exit-code change. No new contract section number.

## Inter-task Contracts

None.

## Implementation

See `docs/adr/ADR-053-contract-does-not-grep-readme-prose/tasks/README.md`.

## Consequences

- **Positive:** a README tidy of a heading or tutorial sentence is documentation, not a red contract row.
- **Negative:** README can drift from the install block or the `A,+N` wording without `contract.sh` noticing. The binary, `mrw instructions`, AckRule, AGENTS.md, and the MCP wire remain gated.
- **Neutral:** callers see no behaviour change.

## Out of Scope

- Deleting `TestEverySurfaceCarriesTheOneRule` or changing `AckRule` (permanent: boundary: M said keep it)
- Deleting §75 or asserting Shared() only in-process (permanent: boundary: the shipped binary is the check)
- Raising `maxInstructionsChars` / 4096 (permanent: boundary: ADR-037's go/no-go)
- Rewriting or gutting `BESTPRACTICES.md` / `docs/measure.md` (permanent: boundary: M said they stay)
- Dropping AGENTS.md greps in §64 (permanent: boundary: this record is README prose)
- Dropping `TestEveryDocumentedReplaceCarriesItsAnchor` (permanent: boundary: parser on `@@` headers, not tutorial phrases)
- Generating AGENTS.md from `Shared()` (permanent: fact: ADR-045 still refuses that; citation: file `docs/adr/ADR-045-agents-md-is-not-generated-from-shared.md:7`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A comment naming a fossil is mistaken for a live grep | Med | Low | The pin skips `#` lines; a live `grep` of the phrase still fails it |
| §39 loses the "args the README prints actually start" check | Med | Med | Help lists `mcp` and `mrw mcp` starts; a broken documented argv is a docs bug, not a product break |
| Someone re-adds the greps in a later tidy | Med | Med | The pin test fails on the live line |

## Rollback

Restore the five README assertions in §39 and §64. Delete the pin test. The binary is unchanged.

## Follow-ups
