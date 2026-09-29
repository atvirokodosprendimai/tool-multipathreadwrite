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
| `cmd/mrw/teach094_test.go` | add | `TestEverySurfaceTeachesWhatAStepChecked` — `cmd/mrw` can run `mrw instructions` in-process, as `instructions_test.go` does, and read AGENTS.md and README.md |
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
go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestEverySurfaceTeachesWhatAStepChecked' -v 2>&1 | tee /tmp/adr094-T3.out \
  && grep -qE '^--- PASS: TestEverySurfaceTeachesWhatAStepChecked \(' /tmp/adr094-T3.out \
  && go test ./cmd/mrw/ ./internal/guide/ -count=1 -timeout 600s -run 'TestEverySurfaceTeachesThen|TestEverySurfaceContainsTheSharedSentences' \
  && [ -z "$(gofmt -l .)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state)" ] \
  && [ -z "$( { git diff --name-only "$(git merge-base HEAD origin/main)" -- internal/check; git ls-files --others --exclude-standard -- internal/check; } | grep -vxE 'internal/check/(check\.go|steps094_test\.go)')" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEverySurfaceTeachesWhatAStepChecked` | `cmd/mrw/teach094_test.go` | the output of `mrw instructions` (run, exit 0), AGENTS.md and README.md each carry both sentences verbatim | — | S1, S2 |

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
