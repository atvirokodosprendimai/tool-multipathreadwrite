# Task ADR-094-T1: A placeholder in a step is refused before anything is written

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `check.Placeholder`, the placeholder refusal on `write` and `check`; contract §178
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a declared step holding a token is refused`, `the refusal and the expansion name the same tokens`, `an ad-hoc step holding a token is refused`, `write refuses before anything is written`, `write refuses an ad-hoc token before the plan is read`, `check refuses before its own check runs`, `the tally counts the refusals as ADR-083 does`, `a step without a token still runs`, `a write asking for no step never reads the block`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

A step whose command holds `{files}` or `{packages}` — declared under `"steps"` or given with
`--then-sh` — is refused, exit 2, before anything is written or run, naming the step and the token.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `Placeholder(cmdline string) string` beside `command()`, naming the two tokens `command()` substitutes; `checkSteps` refuses a declared step whose command holds one. `command()` is not touched |
| `internal/check/steps094_test.go` | add | the two unit tests below |
| `cmd/mrw/main.go` | edit | `askedStepsError` refuses a `--then-sh` holding a token, beside the empty-command rule. What SELECTS both refusals is already wired: `askedStepsError` at `:1114` (write) and `:1929` (check), `resolveSteps` → `StepCommands` → `checkSteps` at `:1256` (write) and `:1932` (check) |
| `cmd/mrw/then094_test.go` | add | the two CLI tests below, one per subcommand, each covering the declared and the ad-hoc case |
| `scripts/contract.sh` | edit | §178 drives the built binary |

## Ordered Steps

1. [S1] Write the four failing tests; they fail to build until `check.Placeholder` exists, and the CLI
   tests fail on exit 0 once it does and nothing calls it. [proof: mutation]
2. [S2] `check.Placeholder`: the first of `{packages}`, `{files}` the command holds, or "". Its doc
   comment says `command()` is the one place the tokens mean something; `command()`'s comment points
   back. [proof: mutation]
3. [S3] `checkSteps`: a declared step whose command holds a token is refused, naming the step and the
   token — `.quality-harness.json: step "fmt": its command holds {files}, which mrw expands only in
   scoped_check; a step runs as written` — in the same sorted order as its other cases. Reached through
   `StepCommands` and `resolveSteps` on both subcommands, so a write refuses before `writer.Apply` and
   a check before `check.Run`. [proof: mutation]
4. [S4] `askedStepsError`: a `--then-sh` whose command holds a token is refused the same way, before
   the plan is read (write: given a plan path that does not exist, the refusal still names the token,
   not the path) and before the check runs (check). [proof: mutation]
5. [S5] Contract §178, on the built binary: a declared `fmt` step holding `{files}` is refused by
   `write` (exit 2, the file byte-identical, no step marker) and by `check` (exit 2, and neither the
   declared check's marker nor the step's — the check touches one, so a refusal moved below `check.Run`
   fails the row); a `--then-sh` holding `{packages}` is refused by both the same way; and, the pair
   that must pass, the same declared step and the same `--then-sh` with the token written out as a
   path run and leave their marker.
   [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §178's rows printed — the whole contract takes minutes and a fence runs at least three times per task, so the fence greps the section and lifecycle §6 runs it]

## Acceptance

```bash
set -o pipefail
go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestADeclaredStepHoldingAPlaceholderIsRefused|TestPlaceholderNamesEveryTokenTheScopedCheckExpands|TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten|TestCheckRunsNoStepThatHoldsAPlaceholder' -v 2>&1 | tee /tmp/adr094-T1.out \
  && missing=$(for t in TestADeclaredStepHoldingAPlaceholderIsRefused TestPlaceholderNamesEveryTokenTheScopedCheckExpands TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten TestCheckRunsNoStepThatHoldsAPlaceholder; do grep -qE "^--- PASS: $t \(" /tmp/adr094-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./internal/check/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestStepsRunInOrderAndStopAtTheFirstThatDoesNotPass|TestAMalformedStepsBlockStillLoads|TestAnUnknownStepIsRefusedBeforeAnythingIsWritten|TestAPlainWriteIgnoresAMalformedStepsBlock' \
  && grep -q '^func Placeholder(' internal/check/check.go \
  && grep -q '^# 178\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestADeclaredStepHoldingAPlaceholderIsRefused` | `internal/check/steps094_test.go` | `StepCommands` refuses a block whose step holds `{files}`, and one holding `{packages}`, each error naming the step and the token; the same block with `{file}`, `{ files }` or `{FILES}` in place of the token loads | — | S1, S3 |
| `TestPlaceholderNamesEveryTokenTheScopedCheckExpands` | `internal/check/steps094_test.go` | for every token `Placeholder` reports, a `scoped_check` holding it has it substituted by `command()` on a mapped Go path, so the refusal and the expansion name the same tokens; `Placeholder` returns "" for the near-misses | — | S2 |
| `TestAStepWithAPlaceholderIsRefusedBeforeAnythingIsWritten` | `cmd/mrw/then094_test.go` | on `write`: `--then fmt` declared with `{files}`, and `--then-sh` holding `{packages}`, each exit 2 naming the step and the token, the file byte-identical and no step marker; under `--json` each is one refusal document; the `--then-sh` case given a plan path that does not exist still names the token, not the path; the tally (`authoring.Load`) gains one `refused_apply` for the declared case and nothing for the ad-hoc one; a subtest runs the same declared step and `--then-sh` with a path in place of the token, which pass and leave their marker | — | S1, S3, S4 |
| `TestCheckRunsNoStepThatHoldsAPlaceholder` | `cmd/mrw/then094_test.go` | on `mrw check a.go`, with a declared check that touches its own marker: the declared and the ad-hoc case each exit 2 with neither the check's marker nor a step marker, under `--json` each is one refusal document, and the tally is unchanged; a subtest runs the token-free siblings after a passing check | — | S3, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `check.Placeholder` and the two refusal cases |
| 2 — something selects it | `askedStepsError` (write `:1114`, check `:1929`) and `resolveSteps` → `StepCommands` → `checkSteps` (write `:1256`, check `:1932`); each CLI test covers a declared and an ad-hoc step on its subcommand, so deleting any one of those four call sites, or either refusal case, turns a test red, and moving either `check` call below `check.Run` (`:1935`) leaves the check's marker, which the check test forbids; §178 drives the built binary |
| 3 — the caller can discover it | the refusal names the step, the token and why; T3 teaches it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-29 survey and the probe in the parent's Context |

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/check/check.go` · checkSteps accepts a declared step holding a token · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:a declared step holding a token is refused
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/check/check.go` · Placeholder names a token command() does not substitute · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:the refusal and the expansion name the same tokens
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · askedStepsError lets a --then-sh holding a token through · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:an ad-hoc step holding a token is refused
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · write goes on to apply the plan after a declared step is refused · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:write refuses before anything is written
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · write never judges its --then-sh commands before reading the plan · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:write refuses an ad-hoc token before the plan is read
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · check refuses a declared step only after its own check ran · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:check refuses before its own check runs
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · a refusal after the plan parsed goes uncounted · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:the tally counts the refusals as ADR-083 does
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/check/check.go` · every step command is taken to hold a token, so a token-free step is refused too · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:a step without a token still runs
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · the steps block is judged on every write, asked or not · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:a write asking for no step never reads the block
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/check/check.go` · check.go is not gofmt-clean · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:the tree is gofmt-clean
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · covers:no other engine package changes

## Invariants

- A step without a token runs exactly as before; `scoped_check` and `check` are chosen and run exactly
  as before (`command()` is not edited).
- A write or check asking for no step never reads the `steps` block (ADR-092 T5).
- Every refusal lands before a byte is written (ADR-072) and exits 2.

## Risks

- A near-miss token (`{file}`) must still run; the unit test pins three near-misses.
- The whole-block judgement means a token in an unasked step refuses an asked one; that is ADR-092
  T5's existing rule for every other defect in the block, and the message names the offending step.

## Stop Condition

Stop and ask if refusing in `checkSteps` would refuse a write that asks for no step, if any engine
package other than `internal/check` needs to change, or if an existing test outside this task starts
to fail. Otherwise: the fence exits 0.

## Out of Scope

- The `check` field holding a token — the parent's Out of Scope, deferred to BACKLOG.
- Teaching the rule — T3.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:585 · test-lock-sha256:2ff869a9badd18404570fa09513e72273436e5552cdba06eb1baa3268c53958e · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGhlbjA5NF90ZXN0LmdvCVRlc3RBU3RlcFdpdGhBUGxhY2Vob2xkZXJJc1JlZnVzZWRCZWZvcmVBbnl0aGluZ0lzV3JpdHRlbgkzYTA0N2VhMWM3OGFlN2E1OTI3YWZjOWE5MWE4NmE1YTZjOWZmZDZiODM0YjdhOTY0ZDU5MmIwYTY3YTAxODE3CmJvZHkJY21kL21ydy90aGVuMDk0X3Rlc3QuZ28JVGVzdENoZWNrUnVuc05vU3RlcFRoYXRIb2xkc0FQbGFjZWhvbGRlcgkwNjI4NjAxZDcwNWU3MTliNWI3ZjI3OTlkMDVjOGE1NTI5MWJlNDc0N2NkMGQ3M2QyMDkwNjU5OTZjYzE3Zjk3CmJvZHkJY21kL21ydy90aGVuMDk0X3Rlc3QuZ28JdGhlIHNhbWUgc3RlcHMgd2l0aCBhIHBhdGggaW4gcGxhY2Ugb2YgdGhlIHRva2VuIHJ1bgkxN2I1N2FhNzlkOGJkZTUwNDFhMzUyNDRjNjJhYWE5NDRiNmRmNTQ5NDdkZWM0NDY3NjYxMjZjNWNjM2U1OTgxCmJvZHkJY21kL21ydy90aGVuMDk0X3Rlc3QuZ28JdGhlIHRva2VuLWZyZWUgc2libGluZ3MgcnVuIGFmdGVyIGEgcGFzc2luZyBjaGVjawkxMDQwNTRkOTA4ZTQzN2FjZDliMTVlZDgzYjExMTNhNjMxODZjODQwMTA4YWE0YzdlYzVkNDA3YTM3YzdiNDdlCmJvZHkJaW50ZXJuYWwvY2hlY2svc3RlcHMwOTRfdGVzdC5nbwlUZXN0QURlY2xhcmVkU3RlcEhvbGRpbmdBUGxhY2Vob2xkZXJJc1JlZnVzZWQJZGQxMzI0MTQzMzZkOGM1NmVhNjJmNTdlZWUzZTliN2VlYjgwMmQwNWI5YjYwNzVlZjEzYjgxYjZlY2ZmMWExZgpib2R5CWludGVybmFsL2NoZWNrL3N0ZXBzMDk0X3Rlc3QuZ28JVGVzdFBsYWNlaG9sZGVyTmFtZXNFdmVyeVRva2VuVGhlU2NvcGVkQ2hlY2tFeHBhbmRzCTY0ZmU3NDVkMzcxOWY4ODBjZTUyY2QwM2YyMDI1YmFmYzRmZWRiMWRhNTM3OTRkMjBiOTRhMTJlMGRlNzlhOTM
  ```
  --- last 10 line(s) of stdout (of 374 after folding 374 raw)
              ],
              "pruned_logs": 0
            }
          }
  === RUN   TestCheckRunsNoStepThatHoldsAPlaceholder/the_token-free_siblings_run_after_a_passing_check
  --- FAIL: TestCheckRunsNoStepThatHoldsAPlaceholder (0.12s)
      --- PASS: TestCheckRunsNoStepThatHoldsAPlaceholder/the_token-free_siblings_run_after_a_passing_check (0.03s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.350s
  FAIL
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:723
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:751
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:726
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:793
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:931
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:712
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:699
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:743
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:722
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:704
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:30b5822cf3d3ed496530ad83e76e273d0737e40a6a1efb986526b959bb0d8d5a · ms:705
