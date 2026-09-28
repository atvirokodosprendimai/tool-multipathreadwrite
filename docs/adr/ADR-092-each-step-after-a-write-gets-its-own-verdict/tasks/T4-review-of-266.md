# Task ADR-092-T4: Every refusal and every kept log reaches the receipt

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `TestACheckStepRefusalIsOneJSONDocument`, `TestAWriteWhoseCheckCannotRunStillNamesItsSteps`, `TestAPassingStepThatKeptItsLogNamesIt`
**Consumes:** `--then`, `--then-sh`, the `then` receipt (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a check --json refusal is one document`, `a write whose check cannot run names its steps not run`, `a passing step that kept its log names it`, `the tree is gofmt-clean`, `no other engine package changes`

## Goal

Close the three findings of the Codex review of PR #266 (xhigh, 2026-09-28, head 4483cc9), each
source-traced there and confirmed against `cmd/mrw/main.go` here.

1. `mrw check --json` refused a step — an unknown `--then`, an empty `--then-sh`, a malformed `steps`
   — with no JSON on stdout (`main.go:1861-1869`), against Decision 5's single refusal document.
2. A write whose `check.Run` returned an error (a scope refused, or a temp log that could not be
   created) left through `refuseWith` without `then` (`main.go:1378`), so the steps asked for were
   named nowhere, against Decision 5's "every step `not_run`".
3. A passing step whose output ran past `tail_lines` keeps its log (ADR-080), and the human receipt
   named neither the log nor the truncation (`main.go:1676`).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `check`'s load and step refusals encode one `{error}` document under `--json`; `refuseWith` carries the steps as `not_run` after a check that could not run; `reportSteps` names a passing step's kept log |
| `cmd/mrw/then092b_test.go` | add | the three tests below |

## Ordered Steps

1. [S1] Write the three failing tests. [proof: mutation]
2. [S2] `check --json`: the harness, empty-step and unknown-step refusals encode `{"error": …}`. [proof: mutation]
3. [S3] `write`: at the `check.Run` error, the asked steps become `not_run`, carried by `refuseWith`
   into the JSON document and printed in the human form. [proof: mutation]
4. [S4] `reportSteps`: a passing step with truncated output names `... N earlier line(s) in <log>`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestACheckStepRefusalIsOneJSONDocument|TestAWriteWhoseCheckCannotRunStillNamesItsSteps|TestAPassingStepThatKeptItsLogNamesIt' -v 2>&1 | tee /tmp/adr092-T4.out \
  && missing=$(for t in TestACheckStepRefusalIsOneJSONDocument TestAWriteWhoseCheckCannotRunStillNamesItsSteps TestAPassingStepThatKeptItsLogNamesIt; do grep -qE "^--- PASS: $t \(" /tmp/adr092-T4.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && [ -z "$(gofmt -l .)" ] \
  && [ -z "$(git diff --name-only 8ecb059 -- internal/read internal/apply internal/plan internal/seen internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACheckStepRefusalIsOneJSONDocument` | `cmd/mrw/then092b_test.go` | `check --json` with an unknown `--then`, an empty `--then-sh`, and a harness with an empty step each exits 2 with one JSON document carrying `error` | — | S1, S2 |
| `TestAWriteWhoseCheckCannotRunStillNamesItsSteps` | `cmd/mrw/then092b_test.go` | with no temp directory the check cannot create its log: the write lands, exits 2, and its `--json` document and human receipt name every step `not_run`; no step ran | — | S3 |
| `TestAPassingStepThatKeptItsLogNamesIt` | `cmd/mrw/then092b_test.go` | a step printing past `tail_lines` passes, and the receipt names its kept log, which exists | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three paths in `main.go` |
| 2 — something selects it | `check`'s action, `write`'s check-error path, `reportSteps` |
| 3 — the caller can discover it | the receipt itself |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex review of #266 |

## Mutation Log
(empty until execute)
- 2026-09-28 · 4483cc9* · mutant killed · exit 1 · `cmd/mrw/main.go` · the check --json refusal carries no error · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · covers:a check --json refusal is one document
- 2026-09-28 · 4483cc9* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refusal drops the steps · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · covers:a write whose check cannot run names its steps not run
- 2026-09-28 · 4483cc9* · mutant killed · exit 1 · `cmd/mrw/main.go` · a passing step never names its kept log · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · covers:a passing step that kept its log names it
- 2026-09-28 · 4483cc9* · mutant killed · exit 1 · `cmd/mrw/main.go` · main.go is not gofmt-clean · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · covers:the tree is gofmt-clean
- 2026-09-28 · 4483cc9* · mutant killed · exit 1 · `internal/plan/plan.go` · an engine package changes · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · covers:no other engine package changes

## Invariants

- Without `--then`/`--then-sh`, `check --json`'s success output is unchanged.

## Risks

- None beyond the record's.

## Out of Scope

- `check --json` refusals unrelated to steps — an unreadable working set, a refused scope — which printed no JSON before ADR-092 (permanent: boundary: ADR-092 owns the step surface; the rest of `check --json` predates it and is ADR-072's to widen)

## Stop Condition

The fence exits 0.

## Verification Log
(empty until execute)
- 2026-09-28 · 4483cc9* · exit 1 · `set -o pipefail …` · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · ms:774 · test-lock-sha256:1f4a00af9e63b02c4c9507f27f84c5ec3f385829b4a7c267e61278dc0b2d3543 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGhlbjA5MmJfdGVzdC5nbwlUZXN0QUNoZWNrU3RlcFJlZnVzYWxJc09uZUpTT05Eb2N1bWVudAljMzBkYmVmODViZjQ3YjVjOTk4MWU2ZWIxYjA5MDVlMWM1M2I1YjM3NWUxZTYzNDNiOTNiOWIzNTZjODY3Y2FkCmJvZHkJY21kL21ydy90aGVuMDkyYl90ZXN0LmdvCVRlc3RBUGFzc2luZ1N0ZXBUaGF0S2VwdEl0c0xvZ05hbWVzSXQJYjc5ZDQzMTNiM2ExYTA1MDdmM2ViY2U5YzRhYjg2ZWVmYzU1YWE0Zjg1ZmRjZWU0ZTllZDUzMGE0ODgxOTk1ZQpib2R5CWNtZC9tcncvdGhlbjA5MmJfdGVzdC5nbwlUZXN0QVdyaXRlV2hvc2VDaGVja0Nhbm5vdFJ1blN0aWxsTmFtZXNJdHNTdGVwcwkzMTc5ZTNhYzIzMTA1MTdkNDE3NTFmMDJjMmFhM2UxN2QyZjJlNjZjZTI0YWQyZjE1YTNiOGM4NTM4NmExNDMw
  ```
  --- last 10 line(s) of stdout (of 59 after folding 59 raw)
  === RUN   TestAPassingStepThatKeptItsLogNamesIt
      then092b_test.go:88: the receipt does not name the kept log:
          ok   a.go 2 replace  -1 +1
          wrote a.go  2L -> 2L  sha 86ab99a1
          1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied
          then 1/1 loud: seq 1 50 — PASS
  --- FAIL: TestAPassingStepThatKeptItsLogNamesIt (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.303s
  FAIL
  ```
- 2026-09-28 · 4483cc9* · exit 0 · `set -o pipefail …` · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · ms:351
- 2026-09-28 · 4483cc9* · exit 0 · `set -o pipefail …` · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · ms:359
- 2026-09-28 · 4483cc9* · exit 0 · `set -o pipefail …` · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · ms:354
- 2026-09-28 · 4483cc9* · exit 0 · `set -o pipefail …` · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · ms:348
- 2026-09-28 · 4483cc9* · exit 0 · `set -o pipefail …` · acceptance-sha256:59d2cc2072150a90d90c4d84c75094952cdff3405285bfc5e6ab69385eba4fe4 · ms:356
