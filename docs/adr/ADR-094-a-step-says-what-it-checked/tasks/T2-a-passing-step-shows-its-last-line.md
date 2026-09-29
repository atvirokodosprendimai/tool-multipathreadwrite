# Task ADR-094-T2: A passing step shows its last line

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the passing-step line in the human receipt; the `--quiet` usage; contract §179
**Consumes:** none — ordered after T1 only because both edit `cmd/mrw/main.go` and `scripts/contract.sh`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a passing step prints its last non-empty line`, `the line is the JSON tail's last non-empty entry`, `a kept-log pointer counts every line above the shown one`, `a blank tail keeps the kept-log pointer`, `a silent pass prints its head alone`, `then last stays failure-only`, `quiet keeps the step lines`, `the quiet usage says what it drops`, `the tree is gofmt-clean`, `no other engine package changes`, `internal/check changes only in T1's files`

## Goal

Under a passing step's `— PASS` line the human receipt prints the last non-empty line of the step's
output, so `go test ./... || true` shows its `FAIL`, and `write --quiet` says it keeps that line.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `reportSteps`' pass branch prints the last non-empty tail line, with the check's tail prefix, after the head and the kept-log pointer, whose count becomes `Truncated` plus the tail lines above the shown one; `--quiet`'s usage. What SELECTS the line is `reportSteps`, already called after the check's report on write (`:1412`) and check (`:1951`) |
| `cmd/mrw/then094b_test.go` | add | the three tests below |
| `scripts/contract.sh` | edit | §179 drives the built binary |

## Ordered Steps

1. [S1] Write the three failing tests; they fail on a receipt whose pass line has nothing under it.
   [proof: mutation]
2. [S2] `reportSteps`: for a passing step, the last non-empty tail line, printed raw as `  | <line>`
   under the head (after `... N earlier line(s) in FILE` when the log was kept, N being `Truncated`
   plus the shown line's index in the tail, so blank lines after it add nothing); for a tail with no
   non-empty line, no tail line and the kept-log pointer as today; `then last:` and the failure branch
   untouched. [proof: mutation]
3. [S3] `--quiet`'s usage names what it drops — failed hunks and the summary stay, ok and skip rows and
   file lines go — and that the check's report and the step lines still print. [proof: mutation]
4. [S4] Contract §179, on the built binary: `--then-sh 'echo LASTLINE; false || true'` exits 0 and the
   line after its `— PASS` head is `  | LASTLINE`; the pair — `--then-sh true` prints no `  | ` line
   under its head, and `--then-sh 'echo X; false'` still exits 3 with `then last: X`.
   [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §179's rows printed — the fence greps the section and lifecycle §6 runs it]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestAPassingStepShowsItsLastLine|TestASilentPassingStepPrintsOnlyItsHead|TestQuietKeepsEveryStepVerdictAndItsLastLine' -v 2>&1 | tee /tmp/adr094-T2.out \
  && missing=$(for t in TestAPassingStepShowsItsLastLine TestASilentPassingStepPrintsOnlyItsHead TestQuietKeepsEveryStepVerdictAndItsLastLine; do grep -qE "^--- PASS: $t \(" /tmp/adr094-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestAPassingStepThatKeptItsLogNamesIt|TestAFailedStepExitsThreeAndNamesTheStepsNotRun|TestWriteHelpNamesHowToQuoteAHeaderOption' \
  && grep -q '^# 179\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l .)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state)" ] \
  && [ -z "$( { git diff --name-only "$(git merge-base HEAD origin/main)" -- internal/check; git ls-files --others --exclude-standard -- internal/check; } | grep -vxE 'internal/check/(check\.go|steps094_test\.go)')" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPassingStepShowsItsLastLine` | `cmd/mrw/then094b_test.go` | on `write` and on `check`, `--then-sh 'echo first; echo LASTLINE; echo'` exits 0 and the line after its `— PASS` head is the tail line `LASTLINE` with the check's tail prefix, the trailing blank skipped; under `--json` the same step's `tail` has `LASTLINE` as its last non-empty entry; with `tail_lines` 2, `seq 1 50` prints `... 49 earlier line(s) in` then the tail line `50`, and `seq 1 50; echo` prints the same two lines — the second fixture kills a count of `Truncated` plus the tail length less one, which the first cannot | — | S1, S2 |
| `TestASilentPassingStepPrintsOnlyItsHead` | `cmd/mrw/then094b_test.go` | `--then-sh true`, and a `--then-sh` whose `false` is masked to exit 0 and which prints nothing, each print their head with no tail line under it; with `tail_lines` 2, a step printing five blank lines prints its head and `... 3 earlier line(s) in` a log that exists, and no tail line; no passing step prints `then last:`, and a failing step still does | — | S2 |
| `TestQuietKeepsEveryStepVerdictAndItsLastLine` | `cmd/mrw/then094b_test.go` | `write --quiet --then-sh 'echo LASTLINE'` drops the `ok` hunk row and keeps the step's `— PASS` head and its tail line `LASTLINE`; `write --help` shows `--quiet`'s usage naming that the check and step verdicts still print | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the pass branch of `reportSteps` |
| 2 — something selects it | `reportSteps` at write `:1412` and check `:1951`; `TestAPassingStepShowsItsLastLine` drives both subcommands, and §179 the built binary |
| 3 — the caller can discover it | the receipt itself; `--quiet`'s usage; T3 teaches the line |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the survey's lint-L2 and X1 |

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · a passing step prints nothing under its head · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:a passing step prints its last non-empty line
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · the shown line is the last tail entry, blank or not · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:the line is the JSON tail's last non-empty entry
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · the pointer counts the tail length less one, not the lines above the shown one · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:a kept-log pointer counts every line above the shown one
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · a kept log whose tail is blank is not named · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:a blank tail keeps the kept-log pointer
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · every pass prints a tail line, even one with nothing to show · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:a silent pass prints its head alone
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · a pass prints its line as then last: · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:then last stays failure-only
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · --quiet hides the step lines · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:quiet keeps the step lines
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · --quiet usage does not say the verdicts still print · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:the quiet usage says what it drops
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `cmd/mrw/main.go` · main.go is not gofmt-clean · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:the tree is gofmt-clean
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:no other engine package changes
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/check/check_test.go` · a file of internal/check outside T1 changes · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · covers:internal/check changes only in T1's files

## Invariants

- `then last:` is printed for a step that did not pass and never for one that did.
- A failing step's lines, the check's report, `--json` and every exit code are unchanged.

## Risks

- The pointer's count changes meaning for a kept pass log (from lines before the tail to lines above
  the shown one); `TestAPassingStepThatKeptItsLogNamesIt` asserts only the named log, and stays in the
  fence to prove it.

## Stop Condition

Stop and ask if the line cannot be added without changing `--json` or a failing step's lines, or if
`TestAPassingStepThatKeptItsLogNamesIt` has to change. Otherwise: the fence exits 0.

## Out of Scope

- Printing the whole tail under a pass — rejected in the parent.
- Teaching the line — T3.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:496 · test-lock-sha256:d9fe2a0fef3109a011facd0814306938fd4fe5966572a9172a6b7ded645d93ed · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGhlbjA5NGJfdGVzdC5nbwlUZXN0QVBhc3NpbmdTdGVwU2hvd3NJdHNMYXN0TGluZQlkMDZkNWIyMzE3MGYyZjgxOGY5MTM4MWQ5NTk0Yjg1YjM0YjU1MTNmYjliZjQ3MTQ4MjMyNzI4ZDk1NDFlYmQ2CmJvZHkJY21kL21ydy90aGVuMDk0Yl90ZXN0LmdvCVRlc3RBU2lsZW50UGFzc2luZ1N0ZXBQcmludHNPbmx5SXRzSGVhZAkzNGVmYTY1MjMwZmUyYzRmNGExNmY4OTI3ZTc4YmY1MTk2NWVkZTc0MGFkMTQzMWRlZWUzMmJmMjZkYzY0NzNiCmJvZHkJY21kL21ydy90aGVuMDk0Yl90ZXN0LmdvCVRlc3RRdWlldEtlZXBzRXZlcnlTdGVwVmVyZGljdEFuZEl0c0xhc3RMaW5lCWUxMmJiYTFmODUxZWUyYmY4NGUwYTA0ZGJjYTllZmViYjZiZGUxN2YwY2NjMjgxMGNjZmE2NGMzZGE3MmRjYzQKYm9keQljbWQvbXJ3L3RoZW4wOTRiX3Rlc3QuZ28JYSBrZXB0IGxvZyBjb3VudHMgZXZlcnkgbGluZSBhYm92ZSB0aGUgc2hvd24gb25lCTg5MGJjYmU2NmY5YjQwYWMzZmU2OTQ4MGRhYzg3YzBkYjZkY2ViNjUyOThhOGFiZDgzMTQ1YmNjNzM5MDE4OTA
  ```
  --- last 10 line(s) of stdout (of 130 after folding 130 raw)
             --then NAME       after a landed write and a passing check, run the step NAME declared in .quality-harness.json "steps" (repeatable; runs in command-line order with --then-sh; the first that does not pass stops the rest)
             --then-sh CMD     like --then, but run CMD with sh -c as given (repeatable). --then-sh runs any shell command it is given: a harness rule that allows mrw without reading its arguments allows arbitrary shell through --then-sh.
             --help, -h        show help
          
          GLOBAL OPTIONS:
             --root DIR, -C DIR  resolve every path relative to DIR (default: ".")
  --- FAIL: TestQuietKeepsEveryStepVerdictAndItsLastLine (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.233s
  FAIL
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:885
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:809
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:841
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:883
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:847
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:844
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:852
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:853
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:896
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:851
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:890ce572d72f1baf261d09e1a4ef0f1f1deb27963ceec89c5a430d067bff0717 · ms:799
