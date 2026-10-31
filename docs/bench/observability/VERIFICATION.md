# Final-check evidence — local evaluation, 2026-10-30

Status: experimental benchmark companion; no MRW engine, installed binary or public receipt change.
Owner requested implementation and evaluation: "go on, try these here, evalaute".

## What this adds

The existing collector identifies its binary and source. Its new `verify` command also observes one
final full check against a named baseline. It compares the actual contents and executable modes of
declared gate files with that baseline before running the checker. Committing a gate change or hiding
it with Git's assume-unchanged flag does not make the contents equal.

After the check it compares file fingerprints again. A passing process that changed the measured tree,
declared instructions or binary produces rejected evidence. Failed checks and incomplete receipts also
remain rejected. It never infers success from a log's PASS text.

The idea comes from ESF's gate-integrity checks and retained qualification evidence
([ESF](https://github.com/mitkox/esf)); this implementation was written independently for the existing
MRW collector. No ESF runtime or dependencies are required.

## Use on an assessment checkout

Run after the task's final write, with the exact baseline recorded before that task. Declare every
acceptance script, test file, lockfile or configuration that must stay fixed for this comparison.
`.quality-harness.json` is always a declared gate. Its baseline must already configure the intended
full checker; the observer does not invent or repair one.

```sh
python3 /path/to/mrw/scripts/mrw_observe.py verify \
  --root /path/to/assessment-checkout \
  --baseline BASELINE_COMMIT \
  --gate scripts/acceptance.sh \
  --instruction AGENTS.md \
  --mrw /path/to/pinned/mrw \
  --timeout 60 > /path/outside-checkout/trial.json
```

Repeat `--gate` and `--instruction` as needed; both take root-relative file paths. Instructions
should include the exact guidance given to the agent, including any materialized profile snapshot.
The observer additionally hashes the pinned binary's `instructions --core` output.
The checker is exactly `mrw --root CHECKOUT check --json --full`. It executes the configured command,
so review that command and its cost first. This is a POSIX collector, requiring Python 3.11+ and Git.
The checker timeout defaults to 30 seconds and is capped at 120 seconds in this pilot; fingerprinting
and Git reads take additional time. Longer real suites need a separately bounded extension before use.

Exit 0 means that this observation verified its scoped tree. Exit 1 retains a rejected observation
as JSON, including a skipped checker when a declared gate changed. Exit 2 reports setup/collection
errors; it is never verified evidence. Keep every result and exit code, including failures.
Write output outside the checkout so that the output file cannot become part of its own fingerprint.

The record contains baseline commit/blob identities, before/after SHA256 file manifests and tree
digests, gate baseline hashes, declared instruction hashes, binary version/hash, collector hash,
environment metadata, process metrics and the actual check receipt. The final file manifest plus
baseline identifies the measured result; it is not a stored patch. Retain the actual patch separately
if reviewers need to reconstruct the task.

## Acceptance and observed controls

The eight new test methods were authored before or alongside implementation, including a red run
with `verify_trial` absent. The pre-existing thirteen checks remain in the same discovered suite.
The initial clean control also caught incorrect handling of the direct `mrw check --json` receipt;
the implementation was corrected to handle both direct and nested receipts.

`python3 -m unittest discover -s scripts -p test_mrw_observe.py -v` passes 21 tests locally against
installed MRW v1.79.0. These are correctness controls, not speed measurements:

| Control | Observed outcome |
|---|---|
| Unmodified gates, valid final source | Verified; full checker ran and exited 0 |
| Gate edited, committed, assume-unchanged, deleted, or config replaced | Rejected before checker execution |
| Owner-execute removed while group-execute remains | Rejected before checker execution, matching Git's executable flag |
| Original checker rejects bad final source | Rejected; process failure retained |
| Baseline checker exits 0 after adding a source file | Rejected for tree drift |
| Baseline checker exits 0 after changing instructions | Rejected for instruction/tree drift |
| Interrupted check, or exit-zero process with a truncated, nonobject or invalid receipt | Rejected; process record retained |
| New untracked source file | Included; tree identity changes |
| Symlink in evidence scope | Collection refused |
| Real CLI on clean / altered-gate fixtures | Exit 0 / 1 with verified / rejected JSON |

A separate counter-control put bad source beside a weakened `exit 0` checker: ordinary
`mrw check --json --full` returned 0, while the observer rejected the changed gate without running it.
A fresh-context review identified a masking problem in the initial invalid-receipt test (all cases
also had process exit 130). Testing exit-zero cases independently then exposed Python's `false == 0`
comparison trap. Receipt exits now require an integer, and that control passes after the fix.
The first review timed out before returning a verdict; its partial finding was verified locally.
A completed second review found an executable-mode mismatch: Git uses the owner-execute bit, while
the initial observer used any execute bit. A new 0755-to-0654 gate control reproduced a false verified
result, then passed after normalization was changed to Git's owner-execute rule.
A fresh focused review approved that fix with no remaining findings. The final 21-test suite also
passed against a fresh source-built MRW in the CI-equivalent temporary-PATH setup. Ruff 0.15.6 F/E9,
workflow structure validation (actionlint 1.7.12, ShellCheck disabled), and the existing 513-clause
prose check passed locally; no remote CI run or full Go regression campaign was started for this pilot.


Synthetic fixtures, Git commits and read ledgers live in temporary directories and are removed.
CI builds MRW before this discovered suite so the controls exercise a binary rather than silently skip.

## What a performance qualification would need

This evaluation qualifies the observer's controls only. `performance_verdict` stays `not_attempted`.
It does not establish agent-task savings, improve the warning heuristic, or replace the pending paired
assessment. Preserve the existing registrations and their thresholds.

Before a new instruction/profile comparison, freeze a dated registration with held-out tasks grouped
by repository and task family; task and instruction digests; baseline/candidate binary and model
identities; repeat count and order; independent acceptance criteria; the primary quality or cost
endpoint; explicit minimum benefit and allowed cost/p95 regressions; and the uncertainty method.
Write those numerical bars before seeing results. Retain every raw run, checker outcome, acceptance
verdict, token/cost total, elapsed time and retry count. Missing usage is unknown, never zero.
Compare equivalent final verification in both arms. Synthetic fixtures cannot authorize a claim
about real task quality, and several repeats of one task are not independent repositories.
No profile promotion or default-instruction change is justified by this smoke evaluation.

## Scope limits

These are unsigned observations from a trusted operator, not an attestation service or sandbox.
The operator supplies the baseline and gate list; omitted gate dependencies are not protected.
The tree covers baseline paths, current tracked paths and nonignored untracked paths. Ignored
untracked files, external dependencies, the full process environment and ignored build products are
not fingerprinted. Symlinks and submodules are refused. Before/after snapshots cannot detect changes
that were made and restored between them; run in an exclusive checkout with no concurrent writer.
Do not treat this as protection against a hostile checker or a fully reproducible build.

Legitimate changes to declared tests need a separate reviewed comparison or a new baseline, rather
than an exception that silently blesses the change. Check receipts may contain private paths or log
text: review and sanitize assessment artifacts before publishing them. The retained controls use
synthetic names only.
