# Task ADR-013-T3: Teach the form, now that it exists

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S (prose and one contract row)
**Owner:** unassigned
**Produces:** the taught form on the MCP surface and in AGENTS.md
**Consumes:** pattern resolution and the ambiguity refusal (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the taught example really applies`, `the taught ambiguity rule is the shipped one`

## Goal

A caller who has only the MCP surface learns that a plan can address by pattern, that ambiguity is a
refusal, and sees a worked plan using it — and that plan is one the binary has just proved it
accepts.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go` | edit | the grammar paragraph gains the pattern forms and the exactly-once rule |
| `internal/mcp/mcp.go` | edit | the `plan` property description says addresses may be patterns |
| `internal/mcp/mcp_test.go` | edit | the test that pins the taught rule to the shipped one |
| `AGENTS.md` | edit | §"One plan, not N writes" — the pattern form, and when a number is still better |
| `scripts/contract.sh` | edit | §46 — the taught rule is the shipped rule |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): the shipped `instructions` mention the pattern form
   and the exactly-once rule, and that the same rule reaches the `plan` property description a host
   reads when it ignores `instructions`. [proof: acceptance]
2. [S2] Teach it where ADR-012 put the format: `instructionsText` and the `plan` property
   description. **Say the refusal, not only the syntax.** A caller told that `/re/` works and not
   told that two matches fail will read the first refusal as a bug. [proof: mutation]
3. [S3] Update AGENTS.md and README, keeping the existing guidance that a line number is the better
   address for a site already read — the measurement in the parent ADR says a pattern buys nothing
   there and costs bytes. [proof: acceptance]
4. [S4] Add contract §46: the ambiguity rule the wire TEACHES is the rule the binary ENFORCES. Drive
   a two-match plan through the built binary and assert it is refused, then assert the shipped
   `instructions` say so. Two independent reviewers caught ADR-012 teaching an enum the engine never
   sent; this row is that lesson applied to a rule rather than a value. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr013-t3.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr013-t3.out \
  && [ "$(grep -cE '^--- PASS: (TestTheInstructionsTeachThePatternAddress|TestEveryEmbeddedExamplePlanReallyApplies)\b' /tmp/adr013-t3.out)" = "2" ] \
  && grep -q '^# 46\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits** except
`TestEveryEmbeddedExamplePlanReallyApplies`, which exists already and is named here so ADR-012's
Enforced-by is asserted to stay green over the changed surface rather than quietly replaced. It does
NOT walk a pattern-addressed example: no shipped plan carries one. Adding one needs `treeFor` to
build a fixture for a pattern, and it derives file length from `Addr.Start`/`End`, which are zero
until `apply` resolves them. That is a follow-up, recorded as one, and this fence does not pretend
otherwise: naming the test here asserts it stays green, not that it grew new coverage.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheInstructionsTeachThePatternAddress` | `internal/mcp/mcp_test.go` | S2 — the form AND the exactly-once refusal reach the wire, and the same rule reaches the `plan` description | — | S1, S2 |
| `TestEveryEmbeddedExamplePlanReallyApplies` | `internal/mcp/conformance_test.go` | ADR-012's Enforced-by, asserted to stay green over the rewritten surface — it walks the existing line-addressed example, not a patterned one | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests above |
| 2 — something selects it | `initializeResult()` and `tools()` are what every host reads |
| 3 — the caller can discover it | this task IS the discovery surface — it is the whole point of it |
| 4 — it is used | not measured and will not be, for the reason ADR-012's Context gives: counting which fields a host read needs telemetry, and ADR-009 refused it |

## Mutation Log

- 2026-09-04 · 2e8d959 · mutant killed · exit 1 · `internal/mcp/instructions.go` · S2: teach the pattern form without the exactly-once rule — a caller told `/re/` works and not told two matches fail meets the refusal as a surprise and reads it as a bug · acceptance-sha256:f7e0f1fb30aaa5ac51c3bbd55beb4fd6b8a2a4f94ae71a87e56450be602baa25

## Invariants

- Every plan the server publishes still parses and dry-run applies — ADR-012's Enforced-by, unchanged.
- The ambiguity rule stated on the wire is the rule `apply` enforces.
- AGENTS.md still recommends a line number for a site already read.

## Risks

- The taught rule drifts from the enforced one. Mitigated by §46, which drives both in one row —
  the shape ADR-012 needed and did not have when it taught `fail` instead of `failed`.

## Stop Condition

Stop if teaching the pattern form makes `instructions` exceed `maxInstructionsChars`. That bound is
paid by every session, and the right answer would be to shorten what is already there rather than to
raise it silently.

## Out of Scope

- Any change to resolution behaviour (T2 owns it)
- Symbol / AST addressing (permanent: boundary: stated in the parent ADR)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 2e8d959* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7e0f1fb30aaa5ac51c3bbd55beb4fd6b8a2a4f94ae71a87e56450be602baa25 · ms:39880
- 2026-09-04 · eeeb615* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7e0f1fb30aaa5ac51c3bbd55beb4fd6b8a2a4f94ae71a87e56450be602baa25 · ms:13038
