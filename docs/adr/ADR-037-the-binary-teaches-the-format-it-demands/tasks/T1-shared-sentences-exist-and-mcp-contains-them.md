# Task ADR-037-T1: Shared sentences exist and MCP contains them

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `guide.Shared()` (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the five shared sentences being present`, `MCP instructions containing Shared verbatim`, `maxInstructionsChars staying 4096`

## Goal

Put the five shared sentences in one function and make the MCP handshake contain that function's
return value verbatim, without raising the handshake bound.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | add | `Shared()` — the five sentences. New package so `cmd/mrw` does not import `internal/mcp`. |
| `internal/guide/guide_test.go` | add | `TestEverySurfaceContainsTheSharedSentences`. |
| `internal/mcp/instructions.go` | edit | `instructionsText` interpolates `guide.Shared()`. The syntax sentence lands here only via that call. |
| `internal/mcp/mcp_test.go` | edit | `TestMCPInstructionsContainShared`; existing bound test stays. |

## Ordered Steps

1. [S1] Write `TestEverySurfaceContainsTheSharedSentences` and confirm it is RED: `guide.Shared()` does not exist or is missing one of the five sentences the Decision names. [proof: mutation]
2. [S2] Write `TestMCPInstructionsContainShared` and confirm it is RED: `instructionsText()` does not contain `guide.Shared()` as a substring. [proof: mutation]
3. [S3] Implement `guide.Shared()` with exactly those sentences, and call it from `instructionsText()`. If the bound test goes red, shorten MCP-only prose in the same file and say in the commit which paragraph shrank — do not change `maxInstructionsChars`. [proof: mutation]
4. [S4] Confirm `TestTheInstructionsTellAHostHowToAuthorAPlan` still passes with its bound assertion at 4096, and that `TestEveryEmbeddedExamplePlanReallyApplies` is unmodified. [proof: acceptance]
5. [S5] Run `go test ./internal/guide/ ./internal/mcp/` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/guide/ ./internal/mcp/ -count=1 -v \
  -run 'TestEverySurfaceContainsTheSharedSentences|TestMCPInstructionsContainShared|TestTheInstructionsTellAHostHowToAuthorAPlan|TestEveryEmbeddedExamplePlanReallyApplies' 2>&1 | tee /tmp/adr037-t1.out \
  && grep -q '^--- PASS: TestEverySurfaceContainsTheSharedSentences' /tmp/adr037-t1.out \
  && grep -q '^--- PASS: TestMCPInstructionsContainShared' /tmp/adr037-t1.out \
  && grep -q '^--- PASS: TestTheInstructionsTellAHostHowToAuthorAPlan' /tmp/adr037-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr037-t1.out \
  && go test ./internal/guide/ ./internal/mcp/ -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/guide/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEverySurfaceContainsTheSharedSentences` | `internal/guide/guide_test.go` | `Shared()` contains each of the five Decision sentences, including the trigger and "models no target syntax" | — | S1, S3 |
| `TestMCPInstructionsContainShared` | `internal/mcp/mcp_test.go` | `instructionsText()` contains `guide.Shared()` as a substring, so a rewritten handshake that merely paraphrases fails | — | S2, S3 |
| `TestTheInstructionsTellAHostHowToAuthorAPlan` | `internal/mcp/mcp_test.go` | The handshake is still under 4096 characters — the bound is not the fix | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two new tests |
| 2 — something selects it | `instructionsText()` calls `guide.Shared()`; deleting the call fails `TestMCPInstructionsContainShared` |
| 3 — the caller can discover it | the MCP `initialize.instructions` field, already advertised by ADR-012 |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log

- 2026-09-09 · 85b7455* · mutant killed · exit 1 · `internal/guide/guide.go` · S1: Shared drops the syntax sentence, so a surface that paraphrases the other four still fails TestEverySurfaceContainsTheSharedSentences · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b
- 2026-09-09 · 85b7455* · mutant inconclusive · exit 1 · `internal/mcp/instructions.go` · S2: instructionsText stops containing Shared verbatim, so a rewritten handshake that merely paraphrases the five sentences fails TestMCPInstructionsContainShared · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-09 · 85b7455* · mutant killed · exit 1 · `internal/mcp/instructions.go` · S2: instructionsText keeps the guide import but stops splicing Shared into the handshake, so a paraphrase of the five sentences fails TestMCPInstructionsContainShared · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b
- 2026-09-09 · 85b7455* · mutant killed · exit 1 · `internal/mcp/instructions.go` · S3: raising the handshake bound, which ADR-037 forbids — every session pays the field whether or not a tool is called · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b

## Invariants

- `maxInstructionsChars` remains 4096.
- Engine packages are untouched.
- `Shared()` contains no interpolated example plan.

## Risks

- The syntax sentence does not fit. Mitigation: shorten MCP-only prose; Stop Condition if the proposed fix is raising the bound.

## Stop Condition

Stop if the only way to keep the bound test green is to raise `maxInstructionsChars`, or if making
`instructionsText` call `Shared()` requires changing `internal/apply` or `internal/read`.

## Out of Scope

- The CLI subcommand (that's T2)
- README / skill edits (that's T3)

## Verification Log
- 2026-09-09 · 85b7455* · exit 1 · `set -o pipefail …` · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b · ms:675
  ```
  --- last 10 line(s) of stdout (of 28 after folding 28 raw)
      --- PASS: TestEveryEmbeddedExamplePlanReallyApplies/examplePlan (0.00s)
      --- PASS: TestEveryEmbeddedExamplePlanReallyApplies/mrw_write.plan.examples[0] (0.00s)
  === RUN   TestTheInstructionsTellAHostHowToAuthorAPlan
  --- PASS: TestTheInstructionsTellAHostHowToAuthorAPlan (0.00s)
  === RUN   TestMCPInstructionsContainShared
      mcp_test.go:327: instructionsText does not contain guide.Shared() verbatim
  --- FAIL: TestMCPInstructionsContainShared (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.268s
  FAIL
  ```
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b · ms:4361
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b · ms:4423
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b · ms:4499
- 2026-09-09 · 85b7455* · exit 0 · `set -o pipefail …` · acceptance-sha256:ea7a135ed2e4657d3188b0fb697419bc18cefda808d7716ca6968b877da4761b · ms:4199
