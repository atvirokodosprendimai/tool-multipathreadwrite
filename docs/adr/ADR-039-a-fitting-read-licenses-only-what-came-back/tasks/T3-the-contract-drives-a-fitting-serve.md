# Task ADR-039-T3: The contract drives a fitting serve, and the wire says served not only paged

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** none
**Consumes:** fitting serves held pending per file (T1), CLI fitting reads still license without ack (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `§77 driving a fitting serve`, `the wire teaching served reads not only pages`, `maxInstructionsChars staying 4096`, `AckRule verbatim`

## Goal

Drive a fitting MCP read through the built server so an unacked write is refused and an acked one
applies, and retarget every "paged read" ack sentence to "read that served lines" without raising
`maxInstructionsChars`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 77.` after the current highest (`76.`). Fitting `mrw mcp` read of a small file, parse `-- ck` ids, write without ack (must fail), write with ack (must apply). A unit test cannot prove the server calls `hold`. Reserve 77 with `grep -oE '^# [0-9]+\. ' scripts/contract.sh \| sort -k2,2n \| tail -1`. Confirm `# 77. ` returns zero hits before writing. |
| `internal/mcp/mcp.go` | edit | Both tools' `ack` schema descriptions currently say "paged read". Retarget. Schema is what a host lists; a handler that requires a field the schema still describes as pages-only is undiscoverable for fitting reads. |
| `internal/mcp/instructions.go` | edit | "A PAGE LICENSES NOTHING" → the same rule for any serve that carried numbered lines. Shorten MCP-only prose if the bound test goes red. Do not change `maxInstructionsChars`. |
| `internal/mcp/tools.go` | edit | `nameTheAck` / fitting footer, if T1 added one, must embed `AckRule` not a paraphrase. |
| `README.md` | edit | Static duplicate of the ack rule currently talks about a paged read (v1.6.0 paragraph). Update the sentence. Do **not** generate `AGENTS.md` from `guide.Shared()`. |
| `AGENTS.md` | edit | Same static duplicate. Edit the ack paragraph in place. |

## Ordered Steps

1. [S1] Write the failing `# 77.` row and confirm it is RED: an unacked fitting write still
   applies. Confirm `# 77.` matched nothing before writing. Pair the pass (ack, apply) with the
   fail (no ack, refused, tree unchanged). Assert the refusal names `ack` / `AckRule`, not merely
   a non-zero exit. [proof: mutation]
2. [S2] Write the failing assertion that the `ack` schema, handshake, README and `AGENTS.md` still
   describe the requirement as pages-only, confirm it is RED, then retarget those surfaces so they
   no longer say that. Keep `AckRule` verbatim. [proof: mutation]
3. [S3] Confirm `TestTheInstructionsTellAHostHowToAuthorAPlan` (or the bound assertions in
   `internal/mcp/mcp_test.go`) still see `len(instructions) <= 4096` and
   `maxInstructionsChars == 4096`. [proof: acceptance]
4. [S4] Run `gofmt` and `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 77\. ' scripts/contract.sh \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr039-t3.out \
  && grep -q '^# 77\. ' scripts/contract.sh \
  && go test ./internal/mcp/ -count=1 -run 'TestTheInstructionsTellAHostHowToAuthorAPlan|TestEverySurfaceCarriesTheOneRule' \
  && grep -q 'maxInstructionsChars = 4096' internal/mcp/instructions.go \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§77` | `scripts/contract.sh` | Built server: fitting read, unacked write refused, acked write applies | — | S1 |
| `TestEverySurfaceCarriesTheOneRule` | `internal/mcp/ack_test.go` | `AckRule` still appears on handshake, schemas, footer; retarget must not drop it | — | S2 |
| `TestTheInstructionsTellAHostHowToAuthorAPlan` | `internal/mcp/mcp_test.go` | instructions stay ≤ 4096 bytes and the bound is still the literal 4096 | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `# 77.` |
| 2 — something selects it | `readTool` fitting return (T1). A binary that still Records on serve makes S1's fail-case apply |
| 3 — the caller can discover it | `ack` schema + handshake. Mutating the schema back to "paged read" only must fail a test that the fitting requirement is stated — add that assertion in S2 rather than trusting a word-search for `ack` |
| 4 — it is used | nothing measures this yet — ADR-009 |

## Mutation Log

- 2026-09-09 · ebbc4be* · mutant killed · exit 1 · `internal/mcp/tools.go` · Record-on-serve makes §77 unacked fitting write apply · acceptance-sha256:43d57736666363323113a080ed8e7bfa9cf31167da2aa373c1dbaa4fca29eb8b
- 2026-09-09 · ebbc4be* · mutant killed · exit 1 · `internal/mcp/mcp.go` · ack schema framed as pages-only fails TestEverySurfaceCarriesTheOneRule · acceptance-sha256:43d57736666363323113a080ed8e7bfa9cf31167da2aa373c1dbaa4fca29eb8b

## Invariants

- `AckRule` byte-identical to ADR-031.
- `maxInstructionsChars` is the literal 4096.
- `AGENTS.md` is edited, not generated from `Shared()`.
- CLI rows of `contract.sh` stay green (T2).

## Risks

- `./scripts/contract.sh` as the fence is slow (paid three times). Mitigation: this is the
  established T3 shape for a new section; the grep for `# 77.` is what makes the fence RED at
  authoring, before the script's body can run.
- Checking the heading "A PAGE LICENSES NOTHING" stays green if the heading remains and the rule
  is gutted (ADR-031 T3 survivor). Mitigation: assert `AckRule` verbatim via
  `TestEverySurfaceCarriesTheOneRule`; do not grep the heading alone.
- README/AGENTS.md drift: changing the constant does not update them. Mitigation: keep the
  existing pin that those files contain `AckRule`; extend it so they no longer say the rule is
  pages-only.

## Stop Condition

Stop if the only way to keep the bound test green is to raise `maxInstructionsChars`. Shorten
MCP-only prose; if that weakens a claim another test already guards, stop and ask. Stop if teaching
this requires generating `AGENTS.md` from `Shared()`. Stop if §77 can only be made red by paging
the fixture — the row must be a fitting serve, under the ceiling, with checkpoints.

## Out of Scope

- Engine packages (parent go/no-go)
- A host-truncation measurement of an under-ceiling result (parent deferred BACKLOG)

## Verification Log
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:43d57736666363323113a080ed8e7bfa9cf31167da2aa373c1dbaa4fca29eb8b · ms:25736
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:43d57736666363323113a080ed8e7bfa9cf31167da2aa373c1dbaa4fca29eb8b · ms:26125
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:43d57736666363323113a080ed8e7bfa9cf31167da2aa373c1dbaa4fca29eb8b · ms:28825
