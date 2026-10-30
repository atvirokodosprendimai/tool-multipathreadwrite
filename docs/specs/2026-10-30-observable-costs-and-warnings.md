# Observable costs and warnings — follow-up plan

**Status:** Accepted, 2026-10-30 by Zy: "okay, let us implement, accept and verify."
The documentation work and experimental measurement/evidence tool are implemented. See the
[registration and verification results](../bench/observability/RESULTS.md). This changes no shipped
MRW command, receipt, heuristic default or original measurement threshold. A public diagnostic interface
remains conditional on evidence and a separate ADR, as specified below.

## Execution checklist

- [x] Accept and register the bounded verification before observations.
- [x] Implement offline native reporting, command-cost collection and receipt-backed warning evidence.
- [x] Run focused tests red first, then green; kill percentile and evidence-distance mutants.
- [x] Verify the installed binary with the registered synthetic cost and warning fixtures; retain raw results.
- [x] Wire the focused tool checks into CI.
- [x] Fresh implementation review approved; the independent legacy-stderr correction also reviewed and approved.


## Inspiration and existing work

Borrowed with credit from [Herdr](https://github.com/herdrdev/herdr), an Apache-2.0 agent terminal runtime.
The inspected downloaded snapshot declares package version 0.9.5; it was not built or runtime-tested here.
These requirements are independently written; no implementation code was copied.

| Herdr source | Pattern | MRW application |
|--------------|---------|-----------------|
| `src/server/render_scale_benchmark.rs` | Fixed geometry, increasing cardinality, separate stage distributions and ratios | Extend the existing native baseline's reporting and isolate expensive phases |
| `src/detect/manifest.rs`, `src/app/api/agents.rs` | Explain the decision using the same evaluator, with rule and evidence | Make an advisory's basis inspectable without treating it as a correctness verdict |
| `README.md` | Short introduction and installation, links to detailed docs | Keep the MRW quick start in the README and keep detailed reference and evidence out of the opening |

Reuse [the native baseline](../bench/native-baseline/PREREGISTRATION.md) and
[its recorded results](../bench/native-baseline/RESULTS.md). It already varies file sizes, batch widths
and ledger sizes, records distributions and profiles candidates. This is an extension, not a replacement
or a claim that MRW had no scaling measurements. Reuse the
[sibling-closer regression and real duplicate control](../break/shape-hints/sibling-closer.md) for warning work.

## 1. Documentation — implemented and locally verified

- [x] README keeps the first successful two-file edit, limits and links to task-specific guidance.
- [x] Full CLI, safety and integration detail stays in one [expandable README section](../../README.md#command-reference).
- [x] Numeric benchmark tables and their limitations live in [the benchmark overview](../bench/overview.md).
- [x] Verified local links and the previous README section anchors against HEAD. Preserved moved numeric
  tables and reference text; qualified the benchmark scope after review. Focused documentation tests
  and all 513 prose clauses pass.
- [x] Fresh review completed; benchmark-scope and legacy-anchor findings corrected and rechecked.

The prose gate found historical task clauses that require interface details in README. Keep those details
in the expandable section; moving the reference to another file would require a separate contract migration.
Keep the numbered-line MCP acknowledgement rule and shell/process safety statements verbatim where the
existing surface tests require them. Keep AGENTS and the binary's instruction recipes in sync.

## 2. Measurement — implemented as an experimental tool

Write a separate dated pre-registration before collecting new measurements. Link the earlier registration
and results unchanged; do not retroactively add metrics, sizes or acceptance bars to those runs.

The new registration must specify:

- Exact release/source binaries, checksums, source revision and dirty state, driver revision, Go toolchain,
  OS, machine, load gate, concurrency, fixed fixtures and seed, output destination and check commands.
- Existing workload sizes from R1/R2/W1/L1, plus a fixed-repository comparison where useful. Vary one size
  dimension at a time. Keep output limits and matched/served content controlled, or report their changes.
- Separate modes for `--no-check`, a declared scoped check and a declared full check. Use identical edits
  and equivalent final verification coverage. Include a queue ending in one check covering every edit.
- A bounded lock-contention diagnostic if waiting is implicated: writer count, hold duration, timeout and
  whether the holder is applying or checking. Report it separately from uncontended write overhead.
- Warm-up count, timed repetitions, interleaving, cold/warm state definition, total time/CPU budget,
  per-command deadline, cleanup and the stop condition. Respect the costly-runs skill when executing.
- Raw per-sample results with failures, timeouts, refused operations, exit codes, stdout/stderr bytes,
  total wall time and CPU. Do not silently remove slow or failing samples.

Report p50, p90 and max to remain comparable with the existing baseline; add p95 and the sample count,
with the quantile convention stated. Show absolute values and ratios to the smallest registered size.
Report time-growth divided by workload-growth for adjacent sizes, keeping size units explicit.

First separate observable command costs: read, no-check write, checker and final queue verification.
A full-write minus no-check-write difference is an estimate, not a direct internal phase measurement.
Do not label it lock, plan validation or commit time. If profiles identify a need for finer attribution,
use benchmark-only instrumentation with explicit non-overlapping phase boundaries, a measured timer cost
and an untimed control. Do not add production receipt fields or persistent telemetry for this study.

A report must answer: which command or phase dominates, at which size and check mode, and what is directly
measured versus inferred? Show output volume and failure counts beside timing. Declare the optimization
candidate and keep/revert bar before the experiment; use the original bar for any original-baseline claim.
No optimization is justified solely by fewer tool calls. Native timing cannot establish task cost,
model turns, token savings or solved-task quality; the paired agent assessment remains separate.

## 3. Warning evidence — fixture report implemented

The present closer hint is advisory. Its repeated-token match is evidence of a shape, not a parser's
proof of a syntax defect. First characterize true and false positives under ADR-119's existing bar.

For each synthetic fixture, record the operation, replaced span, replacement's last nonblank token,
matched following line and distance, whether the old span already ended in that token, expected advisory,
actual advisory and write/check exit codes. Start with the valid sibling expectation and actual duplicate
Markdown closer already captured. Add ordinary repeated statements, nested valid closers, blank-line
spacing and no repeated token as controls. A missing warning on a true duplicate is a regression too.

If explanation output is warranted, design it as an opt-in view of the same decision inputs used by the
warning. Do not build a second evaluator that can disagree. Expose a stable rule identifier and the
matched evidence; identify skipped evaluation and unavailable evidence honestly. Keep normal successful
receipts concise. Before adding any public field or command, obtain a new ADR, additive receipt-contract
coverage, CLI/MCP parity and a test proving that explained and ordinary decisions agree.

Do not quietly promote heuristics into refusals, add language parsers or change warning defaults here.
The sibling-closer obligation stays in [BACKLOG](../adr/BACKLOG.md#from-adr-119-sibling-closer-false-positive)
until the real and false-positive controls meet the established bar.

## Boundaries and reporting

- Existing schema generation and receipt-compatibility checks already cover much of Herdr's single-source
  contract pattern. No schema framework rewrite is proposed.
- A terminal runtime, background MRW service, event bus, plugin marketplace and shared-session cache are
  outside this change. Event-loss recovery belongs to a future design only if such a consumer exists.
- Keep private repository names, paths and customer identifiers out of tracked reports and fixtures.
  Use synthetic names; retain external provenance only in the authorized private record.
- The implementation report names completed work, exact commands and exit codes, measured findings,
  unmeasured assumptions, remaining obligations and cleanup. A plan or passing write is not implementation
  or verification of the proposed measurement/diagnostic work.
