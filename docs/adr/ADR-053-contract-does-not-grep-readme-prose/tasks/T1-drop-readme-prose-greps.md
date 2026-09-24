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
- 2026-09-24 · 89572e8* · exit 0 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:425
- 2026-09-24 · 89572e8* · exit 0 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:425
- 2026-09-24 · 89572e8* · exit 1 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:167 · test-lock-sha256:baadc953c87d867a8bd0e3db430b2a82a93105385fda6e0e5dc8b8ac52c71893 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FkdmVyc2FyaWFsL2NvbnRyYWN0X3Byb3NlX3Rlc3QuZ28JVGVzdEFSZWFkbWVXaXRob3V0VGhlTWNwSG9zdEhlYWRpbmdJc05vdEFHb1Rlc3RGYWlsdXJlCWU3ZGJmMDdlMWRjNjc1NGQ5YmM3YmM3OTc4MmY3ODVmYTBlYzlkMzkxNmE1ZmMyMGU4OTEwZjc1NDA0OGQ0NGEKYm9keQlpbnRlcm5hbC9hZHZlcnNhcmlhbC9jb250cmFjdF9wcm9zZV90ZXN0LmdvCVRlc3RDb250cmFjdERvZXNOb3RHcmVwUmVhZG1lRm9yVHV0b3JpYWxQaHJhc2VzCWUwZjY5ZjMzZjY2ZTM5MDFlMDE5OGNmN2I2NDFiZGM3NGUxNDA5YjkxNDFlODE2YjY4ZDdhZTE5YWY0NWM3MGUKYm9keQlpbnRlcm5hbC9tY3AvYWNrX3Rlc3QuZ28JVGVzdEFDTElSZWFkU3RpbGxMaWNlbnNlc1dpdGhvdXRBY2sJYjcyOWQxOTY3NmU5OGI2MzVkODhmMTcyMDZiNmVlNTY1ODhiZWRlZjgwMjUyNjc5ZWEyODYxYmVmYmE5OWQ2Ygpib2R5CWludGVybmFsL21jcC9hY2tfdGVzdC5nbwlUZXN0QUNoZWNrcG9pbnRDb3ZlcnNUaGVTcGFuSXRCcmFja2V0cwk4NmVhZjllYjBhYjJhMGFhZDg0ZDIwMDA2ZmFiY2I5YmRjZGJjYTdlNDU4NzYyYTMxMTQyM2Q0MjhmNWIzN2JhCmJvZHkJaW50ZXJuYWwvbWNwL2Fja190ZXN0LmdvCVRlc3RBRml0dGluZ1JlYWRBY2tzV2hlblRoZUhlYWRlclNwZWxsaW5nSXNOb3RUaGVMZWRnZXJLZXkJOGM0Nzc5M2E5YTQ4YTlhYzM2NDgxYjQ1ZjFhMWMzYTgwZTdkNjUzN2U5NjlhNWQzMDUxYzNmMTViOTM5MDE5OQpib2R5CWludGVybmFsL21jcC9hY2tfdGVzdC5nbwlUZXN0QUZpdHRpbmdSZWFkSG9sZHNQZW5kaW5nUGVyRmlsZQllYjBmNDA0OTgzZWY5MWM1YTliMzdjNzcyZWQ2ZGVhNjNkNzEwZjI0ZmMwYWE3NDMxMWU2MDJjMDVkNDlmMzI4CmJvZHkJaW50ZXJuYWwvbWNwL2Fja190ZXN0LmdvCVRlc3RBRml0dGluZ1JlYWRMaWNlbnNlc09ubHlXaGF0Q2FtZUJhY2sJOTg3ZDk1NzY5YjQ2YmQ4NGM0N2Q3ZjI5ODdjODE0OWFlYTcxMWQ1MzZlNGM5MjJiZWVkODMyZjk5OWIwYjlkMApib2R5CWludGVybmFsL21jcC9hY2tfdGVzdC5nbwlUZXN0QVBlbmRpbmdSZWNvcmRSZWFjaGVzTm9MZWRnZXIJZDY5MDA0NjYwZjg0MjYyNzhhZjY3MGMxYmI4MmRhMWUyN2ZmNGQ0Mjk2ZmYzM2E5NDFjNDFkZmM0ODRkNzEwOQpib2R5CWludGVybmFsL21jcC9hY2tfdGVzdC5nbwlUZXN0QVN0YWxlQWNrbm93bGVkZ2VtZW50RG9lc05vdExpY2Vuc2VUaGVDdXJyZW50RmlsZQljZTk0OTYwOWFhODg2NTJhOTE2ZjgxZGUxYzI5YTkwMWE3Yzc0MmZlMDdmYmE3NGI1MzJjZDQ0NWQ2OTYyMjQ0CmJvZHkJaW50ZXJuYWwvbWNwL2Fja190ZXN0LmdvCVRlc3RBU3RhbGVBY2tub3dsZWRnZW1lbnREb2VzTm90UmV2b2tlVGhlQ3VycmVudE9uZQk2MjI0NGRkZGVjZTYwYzc2NGVkMjhhZDc0MTI5OWY5MmY4NTA3OGE2YjNhNzZjNzMwMGE0YmQ0ZGZmMDFhMzA5CmJvZHkJaW50ZXJuYWwvbWNwL2Fja190ZXN0LmdvCVRlc3RDaGVja3BvaW50c0FyZU5vdEFEZW5zZVNlcXVlbmNlCWJmM2ZiOThhNzQ3ZmRkNjBjNTMzYjg5ODUwYzNiOWVlZjFiN2I1ODZjMDllZjc1YmVkM2Y1Zjc3YmEyMmJkNTYKYm9keQlpbnRlcm5hbC9tY3AvYWNrX3Rlc3QuZ28JVGVzdEV2ZXJ5U3VyZmFjZUNhcnJpZXNUaGVPbmVSdWxlCWViNWIzOWZkODcxYjYzMjc5MTczM2QyMDZjYmRmZDA4ZjEzM2MyNmZhODlkMGMyZmI4ZWRmZDkyZjAzYjVjYTkKYm9keQlpbnRlcm5hbC9tY3AvYWNrX3Rlc3QuZ28JVGVzdE9ubHlBY2tlZFNlZ21lbnRzQXJlUmVjb3JkZWQJODdjOGRjNjZkZTVhODEyMTNhYjE5NGQ3NmU4MDZjZDNhMTkxYmM1ZWVlZmZjMDY0Y2Y2YWQ3ZGE0MGY4MjYwMApib2R5CWludGVybmFsL21jcC9hY2tfdGVzdC5nbwlUZXN0VGhlUGVuZGluZ1N0b3JlSXNCb3VuZGVkCWI2YzM2ZGRmNzhkMDgwODJhNDMyNWM1YjliZDkyYTNiODA0ZTQ5YWUxNjgwY2I3Zjk3YzVlMjc3OTQxMGU5MzcKYm9keQlpbnRlcm5hbC9tY3AvYWNrX3Rlc3QuZ28JVGVzdFRoZVJlbWVkeU1hdGNoZXNUaGVSZWZ1c2VkQWRkcmVzcwlmNWNmMGVhMThkMzdiOWNlN2IyY2Y5ZTk0MDc2ZDAyMmFkYjAxMTI4MGE4YmEwOTc1MGFiYTU3MzVjYjk4N2Ri
  ```
  --- last 8 line(s) of stdout
  === RUN   TestContractDoesNotGrepReadmeForTutorialPhrases
      contract_prose_test.go:29: scripts/contract.sh:4772 still greps README for "### Use it from an MCP host" (ADR-053: drop prose greps)
  --- FAIL: TestContractDoesNotGrepReadmeForTutorialPhrases (0.00s)
  === RUN   TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure
  --- PASS: TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.006s
  FAIL
  ```
- 2026-09-24 · 89572e8* · exit 0 · `set -o pipefail …` · acceptance-sha256:5fea9524cda3b4a1ae88df49196b951f4d1cad99e46d077b0748195464333e33 · ms:426
