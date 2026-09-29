# Task ADR-094-T3: Every surface teaches what a step checked

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `TestEverySurfaceTeachesWhatAStepChecked`
**Consumes:** the placeholder refusal (T1), the passing-step line (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `mrw instructions teaches both rules`, `AGENTS.md teaches both rules`, `README teaches both rules`, `the instructions name no new flag`, `the tree is gofmt-clean`, `no other engine package changes`, `internal/check changes only in T1's files`

## Goal

A caller who reads `mrw instructions`, AGENTS.md or the README learns that a step runs as written — a
step command holding `{files}` or `{packages}` is refused — and that a passing step prints its last
line.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | the `CLI()` steps paragraph gains the two sentences, naming no flag beyond `--then` and `--then-sh` (contract §115 allows only those beyond `read`'s) |
| `cmd/mrw/teach094_test.go` | add | `TestEverySurfaceTeachesWhatAStepChecked` — `cmd/mrw` can run `mrw instructions` in-process, as `instructions_test.go` does, and read AGENTS.md and README.md; and `TestTheInstructionsNameNoFlagTheCLILacks` |
| `AGENTS.md` | edit | "Using mrw" §4 `--then` bullet |
| `README.md` | edit | the "Steps after the check" bullet |

## Ordered Steps

1. [S1] Write the failing test `TestEverySurfaceTeachesWhatAStepChecked`: the output of `mrw
   instructions`, run in-process and exiting 0, AGENTS.md and README.md each contain both sentences
   verbatim — "A step runs as written: a step command holding {files} or {packages} is refused, since
   mrw expands them only in scoped_check." and "A passing step prints the last line of its output
   under its verdict." [proof: mutation]
2. [S2] The two sentences on each surface. [proof: mutation]
3. [S3] Contract §115 still passes: the instructions name no flag `read` lacks beyond `--then` and
   `--then-sh`. [proof: human: `./scripts/contract.sh` run unpiped before the commit, exit 0 with §115's rows printed]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestEverySurfaceTeachesWhatAStepChecked|TestTheInstructionsNameNoFlagTheCLILacks' -v 2>&1 | tee /tmp/adr094-T3.out \
  && [ -z "$(for t in TestEverySurfaceTeachesWhatAStepChecked TestTheInstructionsNameNoFlagTheCLILacks; do grep -qE "^--- PASS: $t \(" /tmp/adr094-T3.out || echo "$t"; done)" ] \
  && go test ./cmd/mrw/ ./internal/guide/ -count=1 -timeout 600s -run 'TestEverySurfaceTeachesThen|TestEverySurfaceContainsTheSharedSentences' \
  && [ -z "$(gofmt -l .)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state)" ] \
  && [ -z "$(base=$(git merge-base HEAD origin/main); own=$(git log -1 --format=%H "$base..HEAD" -- docs/adr/ADR-094-a-step-says-what-it-checked); head=$(git rev-parse HEAD); if [ "${own:-$head}" != "$head" ]; then git diff --name-only "$base" "$own" -- internal/check; else git diff --name-only "$base" -- internal/check; git ls-files --others --exclude-standard -- internal/check; fi | grep -vxE 'internal/check/(check\.go|steps094_test\.go)')" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEverySurfaceTeachesWhatAStepChecked` | `cmd/mrw/teach094_test.go` | the output of `mrw instructions` (run, exit 0), AGENTS.md and README.md each carry both sentences verbatim | — | S1, S2 |
| `TestTheInstructionsNameNoFlagTheCLILacks` | `cmd/mrw/teach094_test.go` | every `--name` in the output of `mrw instructions` (run, exit 0) is a flag some mrw command declares | — | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the text |
| 2 — something selects it | `mrw instructions` prints `guide.CLI()` (`cmd/mrw/main.go:414`); the test runs the command and reads what it printed, so a sentence missing from that output, or from either file, turns it red |
| 3 — the caller can discover it | the three surfaces themselves |
| 4 — it is used | nothing measures this yet |

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/guide/guide.go` · mrw instructions drops the passing-step sentence · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · covers:mrw instructions teaches both rules
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md loses the placeholder sentence · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · covers:AGENTS.md teaches both rules
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `README.md` · README loses the placeholder sentence · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · covers:README teaches both rules
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/guide/guide.go` · mrw instructions names a flag read lacks · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · covers:the instructions name no new flag
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/guide/guide.go` · guide.go is not gofmt-clean · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · covers:the tree is gofmt-clean
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · covers:no other engine package changes
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/check/check_test.go` · a file of internal/check outside T1 changes · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · covers:internal/check changes only in T1's files
- 2026-09-29 · cc8fa3b* · mutant killed · exit 1 · `internal/guide/guide.go` · mrw instructions appends a separate sentence naming --expand, a flag no command declares; both taught sentences stay · acceptance-sha256:3009d3e22aca3d9ac863cf86c7f0645cf65dec7cb85c8a2f090a6d0793a07041 · covers:the instructions name no new flag
- 2026-09-29 · cc8fa3b* · mutant killed · exit 1 · `internal/check/check_test.go` · a file of internal/check outside T1 changes, under the change-range clause · acceptance-sha256:3009d3e22aca3d9ac863cf86c7f0645cf65dec7cb85c8a2f090a6d0793a07041 · covers:internal/check changes only in T1's files

## Corrections

**Correction (2026-09-29):** the Mutation Log row labelled `covers:the instructions name no new flag` ("mrw instructions names a flag read lacks") was killed by the sentence assertion of `TestEverySurfaceTeachesWhatAStepChecked` (`cmd/mrw/teach094_test.go:32`), because that mutant edited the taught sentence itself; nothing in the fence checked a flag, so the row proves only that the sentence is verbatim. The claim is bound now by `TestTheInstructionsNameNoFlagTheCLILacks`, added to the fence: every `--name` in the output of `mrw instructions` must be a flag some mrw command declares. Its kill is the row that appends a separate `--expand` sentence and leaves both taught sentences intact. Contract §115 still holds the stricter rule on the built binary — no flag `read` lacks beyond its exemption set (S3).

**Correction (2026-09-29):** the fence's last clause compared the whole working tree's `internal/check` with the merge-base, so it failed wherever a later record stacked on this branch touched `internal/check` (ADR-095's test files) though ADR-094 had not. It now compares ADR-094's own change range: while the last commit touching this record's directory is HEAD, or no such commit exists yet, the working tree and its untracked files against the merge-base, as before; once commits sit above that one, the merge-base against it. The row `covers:internal/check changes only in T1's files` above was killed on the first branch, which the clause keeps; the row of the same claim under the new digest re-runs that mutant.

## Invariants

- ADR-037's shared sentences stay on every surface (`TestEverySurfaceContainsTheSharedSentences`).
- ADR-092's teaching stays (`TestEverySurfaceTeachesThen`).

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if teaching the rule needs `mrw instructions` to name a flag §115 does not allow.
Otherwise: the fence exits 0.

## Out of Scope

- The centralised `mrw` skill — updated from AGENTS.md at the release, the parent's Follow-up.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:666 · test-lock-sha256:e48aafb28a01957dd616de97be38f39551ba0030d78058e65fd6188313fd0823 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGVhY2gwOTRfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlVGVhY2hlc1doYXRBU3RlcENoZWNrZWQJZWQ5NTRlNTliMzk2YjU3ZWM0NDFjNWZkYjg5YjAzNzViOGZmOTllMzhkMzRkZjM1Yzg4YmNjODcxZmRkYzg0Zg
  ```
  --- last 10 line(s) of stdout (of 11 after folding 11 raw)
      teach094_test.go:36: mrw instructions does not teach "A step runs as written: a step command holding {files} or {packages} is refused, since mrw expands them only in scoped_check."
      teach094_test.go:36: mrw instructions does not teach "A passing step prints the last line of its output under its verdict."
      teach094_test.go:36: AGENTS.md does not teach "A step runs as written: a step command holding {files} or {packages} is refused, since mrw expands them only in scoped_check."
      teach094_test.go:36: AGENTS.md does not teach "A passing step prints the last line of its output under its verdict."
      teach094_test.go:36: README.md does not teach "A step runs as written: a step command holding {files} or {packages} is refused, since mrw expands them only in scoped_check."
      teach094_test.go:36: README.md does not teach "A passing step prints the last line of its output under its verdict."
  --- FAIL: TestEverySurfaceTeachesWhatAStepChecked (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.196s
  FAIL
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:744
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:645
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:675
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:634
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:773
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:696
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:bfdde0d79be7b61a02bd953ac98c8c00d24aba89ff747c336d0f29ac882268aa · ms:670
- 2026-09-29 · cc8fa3b* · exit 0 · `set -o pipefail …` · acceptance-sha256:3009d3e22aca3d9ac863cf86c7f0645cf65dec7cb85c8a2f090a6d0793a07041 · ms:884
- 2026-09-29 · cc8fa3b* · exit 0 · `adr-verify --relock` · acceptance-sha256:3009d3e22aca3d9ac863cf86c7f0645cf65dec7cb85c8a2f090a6d0793a07041 · ms:0 · test-lock-sha256:8d4bc560290da95858e74d8c39c07a02e4d1a3d3e36e0f47ebb69b6f048e23bb · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGVhY2gwOTRfdGVzdC5nbwlUZXN0RXZlcnlTdXJmYWNlVGVhY2hlc1doYXRBU3RlcENoZWNrZWQJZWQ5NTRlNTliMzk2YjU3ZWM0NDFjNWZkYjg5YjAzNzViOGZmOTllMzhkMzRkZjM1Yzg4YmNjODcxZmRkYzg0Zgpib2R5CWNtZC9tcncvdGVhY2gwOTRfdGVzdC5nbwlUZXN0VGhlSW5zdHJ1Y3Rpb25zTmFtZU5vRmxhZ1RoZUNMSUxhY2tzCTc0NGY3ZjI2Zjk3ZTc3NGMwZjFlMmJkOGU1NTgwNDM4ZmFlMzFmYjJlOGMxYzg2YzlkY2MxZDkxODEzODc0ODc · test-lock-kind:relock
- 2026-09-29 · cc8fa3b* · exit 0 · `set -o pipefail …` · acceptance-sha256:3009d3e22aca3d9ac863cf86c7f0645cf65dec7cb85c8a2f090a6d0793a07041 · ms:999
- 2026-09-29 · cc8fa3b* · exit 0 · `set -o pipefail …` · acceptance-sha256:3009d3e22aca3d9ac863cf86c7f0645cf65dec7cb85c8a2f090a6d0793a07041 · ms:739
