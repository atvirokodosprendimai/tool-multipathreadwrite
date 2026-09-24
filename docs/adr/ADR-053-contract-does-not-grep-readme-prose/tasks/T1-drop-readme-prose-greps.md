# Task ADR-053-T1: Pin and drop README prose greps; keep AckRule and §75

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** none
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the pin fails on a live README prose grep`, `§39 no longer parses README`, `AckRule and §75 still exist`

## Goal

A README without `### Use it from an MCP host` does not fail `contract.sh` or `go test`, except `AckRule` if that sentence remains. §39 drives the binary. §64's README greps are gone. §75 and `TestEverySurfaceCarriesTheOneRule` stay.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/adversarial/contract_prose_test.go` | add | The pin: live `scripts/contract.sh` lines must not assert the fossils. AckRule must not require the heading. §75 heading still present. The selector. |
| `scripts/contract.sh` | edit | Retarget §39 onto `--help` + `mrw mcp`. Retire §64's four README greps with a comment pointing at ADR-053. Do not touch §75. |

## Ordered Steps

1. [S1] Write `TestContractDoesNotGrepReadmeForTutorialPhrases` and `TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure` and confirm they are RED while the fossils are live. [proof: mutation]
2. [S2] Retarget §39: drop the README parse; keep `mrw --help` listing `mcp` and `mrw mcp` starting with `mrw_write`. Confirm S1 still red if §64 greps remain, or green once they are gone. [proof: mutation]
3. [S3] Retire §64's four README greps with a comment pointing at ADR-053. Keep AGENTS.md / `--help` / MCP-wire / binary pairing. Confirm S1 GREEN. Re-adding one fossil grep must fail S1. [proof: mutation]
4. [S4] Confirm `TestEverySurfaceCarriesTheOneRule` still passes and `# 75.` is still in `scripts/contract.sh`. `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/adversarial/ -count=1 -v \
  -run 'TestContractDoesNotGrepReadmeForTutorialPhrases|TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure' 2>&1 | tee /tmp/adr053-t1.out \
  && grep -q '^--- PASS: TestContractDoesNotGrepReadmeForTutorialPhrases' /tmp/adr053-t1.out \
  && grep -q '^--- PASS: TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure' /tmp/adr053-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr053-t1.out \
  && go test ./internal/mcp/ -count=1 -v -run 'TestEverySurfaceCarriesTheOneRule$' 2>&1 | tee /tmp/adr053-ack.out \
  && grep -q '^--- PASS: TestEverySurfaceCarriesTheOneRule' /tmp/adr053-ack.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr053-ack.out \
  && grep -q '^# 75\. ADR-037' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/adversarial/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestContractDoesNotGrepReadmeForTutorialPhrases` | `internal/adversarial/contract_prose_test.go` | Live contract.sh lines do not assert README heading / `A,+N` AFTER / CLAMPS / mcpServers-from-README | — | S1, S2, S3 |
| `TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure` | `internal/adversarial/contract_prose_test.go` | AckRule still exists and does not require the MCP host heading | — | S1, S4 |
| `TestEverySurfaceCarriesTheOneRule` | `internal/mcp/ack_test.go` | Unchanged: teaching still matches the server | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two pin tests |
| 2 — something selects it | `./scripts/contract.sh` and `go test ./internal/adversarial/`; re-adding a fossil grep fails S1 |
| 3 — the caller can discover it | ADR-053 names the retired rows |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-24 · c940c54* · mutant killed · exit 1 · `scripts/contract.sh` · a live line greps README for the retired MCP-host heading: TestContractDoesNotGrepReadmeForTutorialPhrases must go red · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · covers:the pin fails on a live README prose grep

## Invariants

- `TestEverySurfaceCarriesTheOneRule` still exists and still requires `AckRule` in README / AGENTS.md / the wire.
- `# 75. ADR-037` still heads the Shared() binary row.
- `maxInstructionsChars` stays 4096.
- `BESTPRACTICES.md` and `docs/measure.md` are not gutted.
- Engine packages stay byte-identical.

## Risks

- A comment that quotes a fossil is not a live grep. Mitigation: the pin skips `#` lines.
- Re-adding the greps "to be safe". Mitigation: S1 fails.

## Stop Condition

Stop if the proposed fix is deleting AckRule, deleting §75, raising 4096, or gutting `BESTPRACTICES.md` / `docs/measure.md`.

## Out of Scope

- AGENTS.md greps in §64 (that's not README)
- Documented `@@` plan-header parse
- Engine behaviour

## Verification Log
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:1009
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:612
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:427
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:407
