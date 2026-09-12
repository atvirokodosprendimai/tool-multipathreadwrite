# Task ADR-040-T1: `write --help` and `CLI()` name how to quote a header option

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `write --help` quoting sentences (T1), `guide.CLI()` quoting sentences (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `write --help naming double quotes`, `CLI() naming double quotes`, `the refusal naming double quotes`, `contract §79 driving the built binary`

## Goal

A PATH caller who reads `mrw write --help` or `mrw instructions` learns that an `anchor=` value
with spaces can be double-quoted, single-quoted, or — for `anchor=` only — left unquoted until
the next `key=`, and that `body=` is a line count while `lines=` is a range guard; a leftover
after a finished quoted `anchor=` names the same fix.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `writeCmd` `Description` — the text `write --help` prints. The selector. |
| `cmd/mrw/writehelp_test.go` | add | `TestWriteHelpNamesHowToQuoteAHeaderOption` — fails if Description drops any of the four sentences. |
| `internal/guide/guide.go` | edit | `CLI()` gains the four sentences so `instructions` agrees. |
| `internal/guide/guide_test.go` | edit | `TestCLIContainsSharedAndTheOperatorTraps` also requires the quoting sentences. |
| `internal/plan/plan.go` | edit | when a token is not `key=value` and the previous token started with `anchor=`, name double quotes. |
| `internal/plan/plan_test.go` | edit | `TestATrailingTokenAfterAQuotedAnchorNamesDoubleQuotes` — leftover after a finished quoted `anchor=`. |
| `scripts/contract.sh` | edit | **§79** — drives `$MRW write --help`. Pair the good case with a help text that must fail if quoting is dropped. Reserve 79 with `grep -oE '^# [0-9]+\. ' scripts/contract.sh \| sort -k2,2n \| tail -1`. |

## Ordered Steps

1. [S1] Write `TestWriteHelpNamesHowToQuoteAHeaderOption` and confirm it is RED: `write --help` / `writeCmd.Description` does not contain double-quote, single-quote, `body=`, and `lines=` as the Decision names them. [proof: mutation]
2. [S2] Extend `TestCLIContainsSharedAndTheOperatorTraps` with the four sentences and confirm it is RED. [proof: mutation]
3. [S3] Write `TestATrailingTokenAfterAQuotedAnchorNamesDoubleQuotes` with a leftover after a finished quoted `anchor=` and confirm it is RED: today's message is only `option "leftover" is not key=value`. A usage error whose previous token is not `anchor=` must keep the generic message. [proof: mutation]
4. [S4] Edit `Description`, `CLI()`, and the `parseHeader` refusal. Confirm the three tests are GREEN. Deleting the `Description` sentences must fail S1 — that is rung 2. [proof: mutation]
5. [S5] Write §79 against a binary whose `write --help` does **not** yet name quoting and confirm it is RED, then rebuild and confirm it is GREEN. Pair the good case (`write --help` names double-quote and `body=`) with a failing case (a help invocation that is usage, or a binary with the sentences stripped). [proof: mutation]
6. [S6] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 79\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ ./internal/guide/ ./internal/plan/ -count=1 -v \
    -run 'TestWriteHelpNamesHowToQuoteAHeaderOption|TestCLIContainsSharedAndTheOperatorTraps|TestATrailingTokenAfterAQuotedAnchorNamesDoubleQuotes' 2>&1 | tee /tmp/adr040-t1.out \
  && grep -q '^--- PASS: TestWriteHelpNamesHowToQuoteAHeaderOption' /tmp/adr040-t1.out \
  && grep -q '^--- PASS: TestCLIContainsSharedAndTheOperatorTraps' /tmp/adr040-t1.out \
  && grep -q '^--- PASS: TestATrailingTokenAfterAQuotedAnchorNamesDoubleQuotes' /tmp/adr040-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr040-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/ ./internal/guide/ ./internal/plan/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWriteHelpNamesHowToQuoteAHeaderOption` | `cmd/mrw/writehelp_test.go` | `write --help` names double-quote, single-quote, `body=` as line count, `lines=` as range guard | — | S1, S4 |
| `TestCLIContainsSharedAndTheOperatorTraps` | `internal/guide/guide_test.go` | `CLI()` still contains `Shared()` and now the quoting sentences | — | S2, S4 |
| `TestATrailingTokenAfterAQuotedAnchorNamesDoubleQuotes` | `internal/plan/plan_test.go` | leftover after a finished quoted `anchor=` names double quotes; a non-anchor trailing token keeps the generic message | — | S3, S4 |
| `§79` | `scripts/contract.sh` | The built binary's `write --help` names quoting; a stripped help fails the paired case | — | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests |
| 2 — something selects it | `writeCmd.Description`; deleting the quoting sentences fails S1 and §79 |
| 3 — the caller can discover it | `mrw write --help`; `mrw instructions` via `CLI()` |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 4b70dd0* · mutant killed · exit 1 · `cmd/mrw/main.go` · dropping the Python sentence from write --help must fail T1 · acceptance-sha256:00a504a990d445e9f1fbc382b13a297a7b7f3c9c26d7f320d207f2317359bb1b

## Invariants

- `guide.Shared()` is byte-identical unless Accept quoted putting the sentences there.
- `maxInstructionsChars` remains 4096.
- A multi-line replace still requires `anchor=` (ADR-035).
- Teach-only used to refuse an unquoted spaced `anchor=`. Accept armed T2; that string now parses. This task's refusal fixture is a leftover after a finished quoted `anchor=`.

## Risks

- Editing `CLI()` and not `Description`, or the reverse. Mitigation: S1 and S2 are both inside the fence.
- A generic `option %q is not key=value` rewrite that fires on every trailing token. Mitigation: S3's negative case.

## Stop Condition

Stop if the proposed fix is raising `maxInstructionsChars` or putting the sentences in `Shared()`.

Stop if `write --help` cannot be driven from the built binary (no `Description` / urfave path).
Then the help text is not a served path and the record is withdrawn.

## Out of Scope

- The parse fork (that's T2)
- The `version` subcommand (that's T3)
- AGENTS.md / the central skill (already show double quotes)
- Engine packages other than the refusal string in `internal/plan`

## Verification Log
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:00a504a990d445e9f1fbc382b13a297a7b7f3c9c26d7f320d207f2317359bb1b · ms:893
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:00a504a990d445e9f1fbc382b13a297a7b7f3c9c26d7f320d207f2317359bb1b · ms:1246
