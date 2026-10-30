# Cost phases and warning evidence — registration, 2026-10-30

Status: Accepted for implementation and bounded verification by Zy: "okay, let us implement, accept and verify."
This registration predates the new observations. It extends the earlier
[native registration](../native-baseline/PREREGISTRATION.md), whose results and bars remain unchanged.

## Fixed scope and budget

Implement an experimental Python analysis/fixture tool, not a change to the shipped MRW interface.
No optimization, production timer, persistent telemetry, parser or heuristic-default change.
Use installed MRW v1.79.0, recording executable SHA256 and version, script SHA256, source revision,
dirty status, Python, OS, machine architecture, logical CPU count and load before/after.
Do not publish absolute private paths or identifiers. Fixtures are synthetic, deterministic and temporary.

Offline native reporting consumes all recorded samples, failures included. Use nearest-rank quantiles
(index ceil(p*n/100)-1); retain p25/p50/p75/p90/max and add p95, sample/failure counts, and output bytes.
Group by workload/size/binary. Ratio each clean cell to the smallest numeric size in its workload/binary;
report adjacent p50 growth divided by size growth only for a single positive numeric dimension. Report
undefined ratios as null for failed samples, zero-size baselines and compound or named sizes such as W1/R3.
Do not reinterpret the original verdict from these descriptive additions.

For command-cost collection use edited-file widths 1, 10, 100 in a fixed 100-file synthetic corpus.
Each file holds one line, "old" or "new". A POSIX checker rejects any other content and prints the
number of verified files. Scoped checks receive the edited paths; full checks visit all 100 files.
The checker is deliberately cheap; its time is not representative of a real project's suite.
Measure read, no-check write, write with scoped check, write with full check, explicit scoped check,
explicit full check and a queue of one-file no-check writes ending in one full check.
Every timed sample begins from identical fixture contents and licensed lines. Fixture restoration and
licensing are untimed. Verify final content and checker coverage outside timing, and retain child failures.
Record per-command and aggregate wall/CPU, stdout/stderr bytes, exit/timeout and checker coverage.
Comparisons use equivalent edits and end with full verification; report each workflow's total separately
from its components. A subtraction is an estimate, never an internal phase timer.

Verification run: 1 discarded warm-up and 3 timed repetitions, sequential, widths 1/10/100;
maximum 120 seconds total collection, 5 seconds per child call, 1-minute load below 4.0 on this
10-core Mac, memory free at least 20%, no peer build. Check load at each size and record before/after.
Exhaustion, load refusal or child failure stops new samples, preserving collected data and an incomplete
status; never retry to select better timings. This is a harness verification run, not a performance verdict.
A later performance campaign needs its own dated registration with at least 30 samples and a declared
optimization bar before data collection. No timing threshold can make this smoke verification pass.

Warning collection has eight fixed controls: valid sibling expectation; real duplicate Markdown fence;
ordinary repeated statement; nested valid closers; blank lines around a duplicate; no repeated closer;
replacement already including its closer; repeated closing token outside the four-nonblank window.
Run each through the real binary, no-check, with a private ledger. Verify applied content exactly.
Record input span/body, replacement's last token, whether the old span ended in it, actual closer receipt,
matched written-file line and nonblank distance, expected/actual hint and write exit. Use the receipt's
actual match, not a second heuristic. Missing or inconsistent evidence is an execution error.
Report every mismatch and confusion counts; this small synthetic set does not estimate a real-world
false-positive rate or meet ADR-119's history-replay shipping bar. The known sibling false positive stays
open; diagnostics do not justify changing the warning default.

## Acceptance and cleanup

Before implementation the focused tests must fail because the tool is absent. After implementation:
quantiles, ordering, scaling boundaries, failure retention, input rejection and receipt-evidence validation
must pass; a deliberately broken percentile or evidence interpretation must make its focused test fail.
Run the real cost and warning fixtures against the pinned binary, retaining raw JSON and metadata.
A complete collection with a warning mismatch is not a passing warning-quality verdict: report both
collection status and quality status explicitly. Temporary fixtures and state are removed; no installs,
services, containers, commits, pushes or changes in other repositories are part of this task.
