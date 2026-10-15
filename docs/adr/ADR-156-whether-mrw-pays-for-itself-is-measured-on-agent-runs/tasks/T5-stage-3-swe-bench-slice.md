# Task ADR-156-T5: Stage 3, the SWE-bench Verified slice

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** L (machine time: heavy — Docker, one container at a time)
**Owner:** Zy
**Produces:** `docs/bench/stage-3-swe-bench/` receipts, `bench/swe-bench/selection.json`
**Consumes:** `bench run` (T2)
**Data dependency:** the SWE-bench Verified dataset and the official harness, version recorded in S2; Docker; starts only on Zy's word
**Proof map:** v1
**Rests-on:** `the selection is the registered rule`, `the harness grades a gold patch here`, `the agent is told the problem statement alone`, `an attempt is graded by the harness's report`, `an instance's image is pulled once and removed after`, `every registered attempt has a receipt`

## Goal

20 instances, chosen by the registered rule, on both models × 2 configurations × 1, graded by the official harness.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/swebench.go` | add | selection by `sha256("adr-156:" + instance_id)`; `LoadSWEBench` builds a task from the problem statement and base commit alone; `SWEGrader` pulls an instance's image, grades through the official harness, removes the image |
| `internal/bench/swebench_test.go` | add | the selection rule; the three tests below |
| `internal/bench/runner.go` | edit | `RunOptions.SWE`; an instance's attempts run together, its image pulled before and removed after; a SWE-bench attempt is graded by `SWEGrader` and its receipt's source is `swe-bench` |
| `internal/bench/task.go` | edit | a task carries its SWE-bench instance |
| `cmd/bench/main.go` | edit | `bench run --source swe-bench --dataset FILE --swebench-python PATH`; `bench analyze` prints "not measured" for a SWE-bench row's silent mis-edits |
| `bench/repos.json` | edit | the slice's seven repositories, cloned under `~/.local/share/mrw-bench/swe-repos/` |
| `bench/swe-bench/selection.json` | add | the 20 instance ids, the dataset revision and the harness version |
| `docs/adr/BACKLOG.md` | edit | "Pre-registered for ADR-156 T5": how an attempt is built, graded and ordered, written before any Stage 3 run |
| `docs/bench/stage-3-swe-bench/` | add | the receipts |

## Ordered Steps

1. [S1] Write the failing test and see it red: `TestTheSWEBenchSliceIsTheLowestTwentyHashes` (given a fixed list of ids, the selection is the 20 lowest hashes, in that order).
2. [S2] Record the dataset revision and the harness version; grade ONE gold patch here as a feasibility probe, bounded at 20 minutes. If it fails or runs over, stop and report. [proof: human: the executor records the probe's time and verdict]
3. [S5] Write the failing tests and see them red — `TestTheSWEBenchTaskIsTheProblemStatementAlone`, `TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime`, `TestAnEmptySWEBenchDiffIsUnsolvedWithoutTheHarness` — then the runner's SWE-bench path: `LoadSWEBench`, `SWEGrader`, the grouping, `bench run --source swe-bench`; and register how an attempt is built, graded and ordered in BACKLOG before any Stage 3 run. [proof: mutation]
4. [S3] Zy's word to start, recorded, given with the price: about 84 attempts, twenty image pulls of several minutes each, hours of machine time. [proof: human: Zy says go for Stage 3]
5. [S6] Clone the seven repositories `bench/repos.json` names for the slice; the pilot (the first instance × A/B × 1 on Sonnet, stage `pilot`), every transcript read; a harness defect is fixed and the pilot rerun into a new directory. [proof: human: the executor reads every pilot receipt and transcript]
6. [S4] The runs: `bench run --source swe-bench --stage stage-3-swe-bench --model claude-sonnet-5-5 --model claude-opus-5-5 --repeats 1`, one container at a time. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ -count=1 -run 'TestTheSWEBenchSliceIsTheLowestTwentyHashes|TestTheSWEBenchTaskIsTheProblemStatementAlone|TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime|TestAnEmptySWEBenchDiffIsUnsolvedWithoutTheHarness|TestAnInterruptedSWEBenchAttemptIsNotGraded|TestATimedOutHarnessRemovesItsContainer' -v > "$out" 2>&1 \
  && grep -qE '^--- PASS: TestTheSWEBenchSliceIsTheLowestTwentyHashes \(' "$out" \
  && grep -qE '^--- PASS: TestTheSWEBenchTaskIsTheProblemStatementAlone \(' "$out" \
  && grep -qE '^--- PASS: TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime \(' "$out" \
  && grep -qE '^--- PASS: TestAnEmptySWEBenchDiffIsUnsolvedWithoutTheHarness \(' "$out" \
  && grep -qE '^--- PASS: TestAnInterruptedSWEBenchAttemptIsNotGraded \(' "$out" \
  && grep -qE '^--- PASS: TestATimedOutHarnessRemovesItsContainer \(' "$out" \
  && go run ./cmd/bench verify --stage stage-3-swe-bench --tasks 20 --repeats 1 docs/bench/stage-3-swe-bench \
  && jq -e '.models == ["claude-sonnet-5-5", "claude-opus-5-5"]' docs/bench/stage-3-swe-bench/run.json > /dev/null \
  && jq -e '.configs == ["A", "B"]' docs/bench/stage-3-swe-bench/run.json > /dev/null \
  && diff <(jq -r '.tasks[]' docs/bench/stage-3-swe-bench/run.json) <(jq -r '.selected[]' bench/swe-bench/selection.json) \
  && jq -s -e 'all(.[]; .source == "swe-bench")' docs/bench/stage-3-swe-bench/*.receipt.json > /dev/null
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSWEBenchSliceIsTheLowestTwentyHashes` | `internal/bench/swebench_test.go` | the registered selection rule | none | S1 |
| `TestTheSWEBenchTaskIsTheProblemStatementAlone` | `internal/bench/swebench_test.go` | the agent is told the problem statement alone; the selection's order; an id the dataset lacks is refused | none | S5 |
| `TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime` | `internal/bench/swebench_test.go` | `Run` grades through the harness's report, the receipt's source, an instance's attempts together with its image pulled once and removed after | none | S5 |
| `TestAnEmptySWEBenchDiffIsUnsolvedWithoutTheHarness` | `internal/bench/swebench_test.go` | no change is unsolved and starts no harness | none | S5 |
| `TestAnInterruptedSWEBenchAttemptIsNotGraded` | `internal/bench/swebench_test.go` | an interrupted attempt starts no harness, and the stage's error names the image it leaves | none | S5 |
| `TestATimedOutHarnessRemovesItsContainer` | `internal/bench/swebench_test.go` | a harness past its budget is stopped and its containers removed by name | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the selection and the receipts |
| 2 — something selects it | `bench run --source swe-bench`; T6 |
| 3 — the caller can discover it | `docs/bench/stage-3-swe-bench/` |
| 4 — it is used | T6 |

## Mutation Log
(empty until execute)
- 2026-10-15 · human-observed · mutant killed · test exit 1 · `internal/bench/runner.go` · line 438 · from `	if len(o.Tasks) > 0 && o.Tasks[0].swe != nil {` · to `	if false {` · test `TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime` · S5: an instance's attempts no longer run together, so its image is pulled more than once; the fence cannot complete until Stage 3 has receipts
- 2026-10-15 · human-observed · mutant killed · test exit 1 · `internal/bench/swebench.go` · line 146 · from `	if strings.TrimSpace(patch) == "" {` · to `	if false {` · test `TestAnEmptySWEBenchDiffIsUnsolvedWithoutTheHarness` · S5: an empty diff starts the harness; the fence cannot complete until Stage 3 has receipts
- 2026-10-15 · human-observed · mutant killed · test exit 1 · `internal/bench/swebench.go` · line 53 · from `` 	ProblemStatement string `json:"problem_statement"` `` · to `` 	ProblemStatement string `json:"hints_text"` `` · test `TestTheSWEBenchTaskIsTheProblemStatementAlone` · S5: the agent is told the hints instead of the problem statement; the fence cannot complete until Stage 3 has receipts
- 2026-10-15 · human-observed · mutant killed · test exit 1 · `internal/bench/runner.go` · line 543 · from `		v, err = o.SWE.grade(a.Task, tree, scratch, a.ID)` · to `		v, err = Oracle(a.Task, tree) // mutant: the corpus oracle grades a SWE-bench attempt` · test `TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime` · S5: a SWE-bench attempt is graded by the corpus oracle, not the harness; the fence cannot complete until Stage 3 has receipts
- 2026-10-15 · human-observed · mutant killed · test exit 1 · `internal/bench/swebench.go` · line 176 · from `		if cerr := g.removeContainers(t.ID, run); cerr != nil {` · to `		if cerr := error(nil); cerr != nil {` · test `TestATimedOutHarnessRemovesItsContainer` · S5, review of #422: a harness past its budget leaves its containers; the fence cannot complete until Stage 3 has receipts
- 2026-10-15 · human-observed · mutant killed · test exit 1 · `internal/bench/runner.go` · line 206 · from `			if image != "" {` · to `			if false {` · test `TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime` · S5, review of #422: an instance boundary drops no image, so a resume leaves one behind; the fence cannot complete until Stage 3 has receipts
- 2026-10-15 · human-observed · mutant killed · test exit 1 · `internal/bench/runner.go` · line 540 · from `	if r.interrupted { // nothing grades it` · to `	if false { // nothing grades it` · test `TestAnInterruptedSWEBenchAttemptIsNotGraded` · S5, review of #422: an interrupted attempt is graded; the fence cannot complete until Stage 3 has receipts

## Verification Log
(empty until execute)
- 2026-10-15 · human-observed · S2 observed 2026-10-15 on this Mac (Docker Desktop 29.8.2, arm64, 8 GB VM): dataset SWE-bench/SWE-bench_Verified test split at revision 78f471bf655a3137b2e8a75af1501690ec009ec3 (500 instances, the same ids as princeton-nlp/SWE-bench_Verified at c104f840, which lacks the image field swebench 5.0.2 reads), harness swebench 5.0.2; the registered selection is in bench/swe-bench/selection.json and equals bench swebench-select's output; the gold patch of the first selected instance, matplotlib__matplotlib-22719, graded resolved: True, all required and regression tests passing, in 25 s, after a 349 s pull of its 10.6 GB amd64 image (pulled with --platform linux/amd64, since no arm64 manifest exists) run under emulation; within the 20-minute bound, so the probe passed. Disk: twenty such images would need about 200 GB against 107 GB free, so each instance's image is removed after its attempts are graded.
- 2026-10-15 · 5b77a44c* · exit 1 · `set -o pipefail …` · acceptance-sha256:defb1b81b2a80435c58265e20264b3721a440c610f2b9ceef936c70d5c5aa9b7 · ms:5663 · test-lock-sha256:f10185d235d9175fcf538624db4cdb098cd6cccc2de81c983224004cbb141c34 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvc3dlYmVuY2hfdGVzdC5nbwlUZXN0QVNXRUJlbmNoU3RhZ2VJc0dyYWRlZEJ5VGhlSGFybmVzc09uZUltYWdlQXRBVGltZQllMjdhZmIyOWJiN2NhMGNkMjhkOTVkMGM3NDNiNWQ2N2Y1ZWE3MDI5NjllY2RkZmNlZWNlYTA4ZDE4ODk1ZjQxCmJvZHkJaW50ZXJuYWwvYmVuY2gvc3dlYmVuY2hfdGVzdC5nbwlUZXN0QW5FbXB0eVNXRUJlbmNoRGlmZklzVW5zb2x2ZWRXaXRob3V0VGhlSGFybmVzcwkzNzAyZGRkNjcxYmQ3ODM5MTdlNjQ0MmNiMDlkNjllZWQ3ZTI0OTJlOGZhNjU5OGQ5YTg2MGU1NmU1NjZjYTZhCmJvZHkJaW50ZXJuYWwvYmVuY2gvc3dlYmVuY2hfdGVzdC5nbwlUZXN0VGhlU1dFQmVuY2hTbGljZUlzVGhlTG93ZXN0VHdlbnR5SGFzaGVzCTY1YTlmMjJlNzIyN2YzZDczMmE5Mjg5MGQ3OWFjODA4ZDBhMTdjYTcyNzBhZmNlYTg2NjNjNDI1YzBlMjIwMDgKYm9keQlpbnRlcm5hbC9iZW5jaC9zd2ViZW5jaF90ZXN0LmdvCVRlc3RUaGVTV0VCZW5jaFRhc2tJc1RoZVByb2JsZW1TdGF0ZW1lbnRBbG9uZQlhOTI1ODJhYWVjYTc4OTY2NzgwMGI3ZGIyYjVjOTA2MTJhOWZkMzE2ODU3NTYyZmNhMDM0NTFmODE3NGViMzli
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-3-swe-bench/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-15 · 815c158d* · exit 1 · `set -o pipefail …` · acceptance-sha256:4b2b77187bb0bca707ff197b2dac575e33b44c46abd6ca6f4eb143dd6b7f8c9a · ms:4201
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-3-swe-bench/run.json: no such file or directory
  exit status 1
  ```
- 2026-10-15 · 815c158d* · exit 1 · `set -o pipefail …` · acceptance-sha256:3b8c6d24fe969f6dbe74474f38a5809f730de55c97e13a63589473ff3a93a224 · ms:6635
  ```
  --- last 2 line(s) of stderr
  bench: open docs/bench/stage-3-swe-bench/run.json: no such file or directory
  exit status 1
  ```

## Invariants

- No gold patch or hidden test reaches the agent.

## Risks

- x86 images under emulation on Apple Silicon: S2's probe decides.

## Out of Scope

- Silent mis-edits on this slice (permanent: boundary: a SWE-bench task has no allowed regions)

## Stop Condition

The fence exits 0; or S2's probe fails, which goes back to Zy.
