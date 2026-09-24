# Task ADR-052-T2: `--echo-pad` / `echo_pad`; contract §88

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `HunkResult.Echo` / `--echo-pad` (T2)
**Consumes:** neighbour license in `Apply` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the pad on an ok hunk`, `default 0 prints nothing`, `help names the flag`

## Goal

`--echo-pad N` and MCP `echo_pad` print N lines after an applied body. Default 0. A pad that shows a closer stays `ok`. Negative is usage.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Options.EchoPad`; attach numbered pad on ok replace/insert. |
| `internal/apply/apply_test.go` | edit | Pad content + `ok` when the pad is a closer. |
| `cmd/mrw/main.go` | edit | `--echo-pad` flag; pass through; `report` prints pad under `ok`. The selector. |
| `cmd/mrw/writehelp_test.go` | edit | Help names `--echo-pad` and that it is not a checker. |
| `cmd/mrw/echopad_test.go` | add | CLI flag through `rootCommand`. |
| `internal/mcp/tools.go` | edit | `echo_pad` on existing `writeTool`. |
| `internal/mcp/mcp.go` | edit | Declare `echo_pad` on `mrw_write` input schema. |
| `internal/mcp/schema.go` | edit | `hunks.echo` description. |
| `internal/mcp/echopad_test.go` | add | MCP pad on the existing tool; cargo still two tools. |
| `scripts/contract.sh` | edit | **§88**. Pair pad-shows-closer-and-ok with default-0-no-pad. |

## Ordered Steps

1. [S1] Write `TestAPaddedWriteEchoShowsTheLineAfterTheBody` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestWriteHelpNamesEchoPad` and confirm it is RED. [proof: mutation]
3. [S3] Wire `EchoPad`, `--echo-pad`, MCP `echo_pad`, schema prose, and `report`. Confirm S1–S2 GREEN. Deleting the flag wiring must fail the CLI test. [proof: mutation]
4. [S4] Write §88 against a binary that does not yet print a pad and confirm it is RED, then rebuild and confirm GREEN. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 88\. ' scripts/contract.sh \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 -v \
    -run 'TestAPaddedWriteEchoShowsTheLineAfterTheBody|TestEchoPadDefaultPrintsNoLines|TestEchoPadNPrintsNNumberedLines|TestEchoPadClampsAtEOF|TestWriteHelpNamesEchoPad|TestWriteEchoPadShowsTheLineAfterTheBody|TestWriteEchoPadDefaultPrintsNoLines|TestWriteEchoPadNPrintsNNumberedLines|TestWriteEchoPadClampsAtEOF|TestWriteToolDeclaresEchoPad|TestAnMCPWriteEchoPadPrintsNNumberedLines|TestAnMCPWriteOmitsEchoAtDefaultZero|TestAnMCPWriteEchoPadClampsAtEOF|TestAnMCPWriteEchoPadKeepsACloserOk' 2>&1 | tee /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestAPaddedWriteEchoShowsTheLineAfterTheBody' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestEchoPadDefaultPrintsNoLines' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestEchoPadNPrintsNNumberedLines' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestEchoPadClampsAtEOF' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestWriteHelpNamesEchoPad' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestWriteEchoPadShowsTheLineAfterTheBody' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestWriteEchoPadDefaultPrintsNoLines' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestWriteEchoPadNPrintsNNumberedLines' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestWriteEchoPadClampsAtEOF' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestAnMCPWriteEchoPadPrintsNNumberedLines' /tmp/adr052-t2.out \
  && grep -q '^--- PASS: TestAnMCPWriteOmitsEchoAtDefaultZero' /tmp/adr052-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr052-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/apply/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPaddedWriteEchoShowsTheLineAfterTheBody` | `internal/apply/apply_test.go` | EchoPad 1 attaches the post-body line; status stays `ok` | — | S1, S3 |
| `TestWriteHelpNamesEchoPad` | `cmd/mrw/writehelp_test.go` | `write --help` names `--echo-pad` and that it is not a checker | — | S2, S3 |
| `TestWriteEchoPadShowsTheLineAfterTheBody` | `cmd/mrw/echopad_test.go` | CLI flag prints the pad on `ok` | — | S3 |
| `TestWriteToolDeclaresEchoPad` | `internal/mcp/echopad_test.go` | Existing `mrw_write` declares `echo_pad`; still two tools | — | S3 |
| `TestEchoPadDefaultPrintsNoLines` | `internal/apply/apply_test.go` | EchoPad 0 attaches nothing | — | S1 |
| `TestEchoPadNPrintsNNumberedLines` | `internal/apply/apply_test.go` | EchoPad 3 attaches three numbered lines | — | S1 |
| `TestEchoPadClampsAtEOF` | `internal/apply/apply_test.go` | A pad past the last line is clamped | — | S1 |
| `TestWriteEchoPadDefaultPrintsNoLines` | `cmd/mrw/echopad_test.go` | CLI default prints no pad | — | S3 |
| `TestWriteEchoPadNPrintsNNumberedLines` | `cmd/mrw/echopad_test.go` | `--echo-pad 3` prints three numbered lines | — | S3 |
| `TestWriteEchoPadClampsAtEOF` | `cmd/mrw/echopad_test.go` | CLI pad clamps at EOF | — | S3 |
| `TestAnMCPWriteEchoPadPrintsNNumberedLines` | `internal/mcp/echopad_test.go` | `echo_pad` 3 prints three numbered lines | — | S3 |
| `TestAnMCPWriteOmitsEchoAtDefaultZero` | `internal/mcp/echopad_test.go` | omitted `echo_pad` prints no pad | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §88 |
| 2 — something selects it | `writeCmd` `--echo-pad`; MCP `echo_pad`; deleting either fails its test |
| 3 — the caller can discover it | `write --help`; `tools/list` schema |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-13 · 2f49847* · mutant killed · exit 1 · `internal/apply/apply.go` · deleting pad attachment must fail the echo tests · acceptance-sha256:8692225ddcdea969be60462122be0c235aa2a64cd4a5a198837bd1dde8151c5f
- 2026-09-13 · 4158e26* · mutant killed · exit 1 · `internal/apply/apply.go` · echoPad returns nothing for every n: --echo-pad 1 prints no pad line, and TestAPaddedWriteEchoShowsTheLineAfterTheBody plus the CLI/MCP pad tests must go red · acceptance-sha256:58a842a38cb8e82b822a70e81183b791a9673fc911683aa4cbbafacca13d897c

## Invariants

- Default 0 changes no receipt.
- A closer in the pad is not a FAIL.
- No third MCP tool.
- 4096 / Shared() untouched.

## Risks

- `hunks.echo` without a `writeDescriptions` entry fails the schema coverage test — add the prose in the same change.
- Quiet hides `ok` and must hide the pad with it.

## Stop Condition

If the pad refuses a hunk, stop — that is a checker.

## Out of Scope

- Teaching beyond help/schema (T3)
- Changing Shared()

## Verification Log
- 2026-09-13 · 2f49847* · exit 0 · `set -o pipefail …` · acceptance-sha256:8692225ddcdea969be60462122be0c235aa2a64cd4a5a198837bd1dde8151c5f · ms:700
- 2026-09-13 · 4158e26* · exit 0 · `set -o pipefail …` · acceptance-sha256:58a842a38cb8e82b822a70e81183b791a9673fc911683aa4cbbafacca13d897c · ms:2233
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:58a842a38cb8e82b822a70e81183b791a9673fc911683aa4cbbafacca13d897c · ms:403
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:58a842a38cb8e82b822a70e81183b791a9673fc911683aa4cbbafacca13d897c · ms:368
