# Grouped-edit Sonnet pilot — results, 2026-10-30

**All ten attempts passed**, but this run did not show a batching advantage over native Sonnet.
Both arms completed the same 50 registered edit sites across 35 file instances. Native already grouped
its edits into Bash scripts. MRW's task cost was 20.3% higher and its CLI wall time 24.0% longer
in these observations. This is a descriptive result from five synthetic pairs, not a general verdict.

[Registration](GROUPED-PREREGISTRATION.md) froze the workload and budgets before model calls.
[Raw metrics, manifests and grading receipts](grouped-pilot.json) retain every cell and both setup calls.
[Fixtures and reference patches](../../../bench/grouped-pilot/tasks) are separate from historical stages.
The subsequent [trace diagnosis](TRACE-DIAGNOSIS.md) separates context costs, turn counters and setup overhead.

## Same work, measured calls

| Measure | Native A | MRW B |
|---|---:|---:|
| Registered edit sites completed | 50 / 50 | 50 / 50 |
| Changed file instances | 35 | 35 |
| Exact-reference solutions / verified final checks | 5 / 5 | 5 / 5 |
| Claude task turns | 16 | 17 |
| Tool calls | 11 | 12 |
| Tool kind used | Bash | Bash |
| MRW write commands | — | 5 |
| Output tokens | 2,835 | 2,906 |
| Cache-read input tokens | 112,883 | 218,496 |
| Cache-creation input tokens | 14,744 | 16,933 |
| Other input tokens | 32 | 34 |
| CLI-reported task cost | $0.0986783 | $0.1187096 |
| Claude CLI wall | 52.540 s | 65.136 s |
| Supervised cell subprocess wall | 53.782 s | 66.226 s |

A turn is the CLI result's `num_turns`, not an edit site or a tool call. The owner clarified a
50-turn-per-arm budget and the hypothesis that 50 native edits could take roughly five MRW writes.
Five jobs × `--max-turns 10` supplied that budget; the fixed work finished well before the cap.
The [CLI reference](https://code.claude.com/docs/en/cli-reference) documents the agentic-turn limit.
No extra jobs were added to equalize observed turns.

The five MRW writes are confirmed by inspecting the five corresponding Bash commands: each builds a
plan and invokes MRW once after its loop. The machine metric is deliberately named
`bash_mrw_write_mentions`; it is not a general shell parser or a receipt invocation counter.
Native also used batch scripts. It was offered Read/Edit/Write/Glob/Grep as well as Bash and chose
Bash throughout. A comparison that assumes one native Edit call per site does not describe this run.

## Per-job observations

Each job contains ten registered sites, legacy decoys, and a fixed hash-based acceptance gate.
The prompt requires preserving formatting and all unrelated text exactly.

| Job | First arm | Files changed per arm | Native / MRW turns | Native / MRW tool calls | Native / MRW reported cost |
|---|---|---:|---:|---:|---:|
| Multi-line structural changes | A | 5 | 4 / 3 | 3 / 2 | $0.0220245 / $0.0234739 |
| Active JSON configuration | B | 10 | 3 / 4 | 2 / 3 | $0.0199681 / $0.0251842 |
| Coupled source/test labels | A | 10 | 3 / 3 | 2 / 2 | $0.0203344 / $0.0210702 |
| Current Markdown sections | B | 5 | 3 / 4 | 2 / 3 | $0.0177802 / $0.0259651 |
| Active routing constants | A | 5 | 3 / 3 | 2 / 2 | $0.0185711 / $0.0230162 |

MRW had lower call and turn counts on the structural job, equal counts on two jobs, and higher
counts on two. Its reported cost was higher on all five. The full MRW instructions and receipts
were part of the treatment. The recorded token totals show more cache-read context, but the pilot
does not isolate how much each instruction, receipt or model choice contributed to cost or latency.

CLI wall includes client startup and model/tool execution. It excludes the observer stage.
Supervised subprocess wall includes adapter startup and final observation, but excludes initial
fixture preparation and the driver's direct reference grading. The observer's shared final-check
stage took 129–229 ms per cell on these tiny fixtures. These are observed ranges, not p95 estimates.

## Validity and retained evidence

- Claude Code 2.1.314, requested and observed model `claude-sonnet-5-5`, identical model/default
  effort in both arms. No plugins or MCP servers appeared in init metadata.
- Installed MRW v1.79.0, SHA256
  `5427ceaebc8d8c3514a62eb1bb102d304acacf203d3280914c9570b9eeee0abe`.
  Claude and MRW executable hashes were unchanged after collection.
- Corpus SHA256:
  `2ea19ac60e1ec218c22627db6c55edf3bb7c9330b5428b567e186a126572e506`.
  All five baseline checks failed, all reference checks passed, and all wrong-decoy controls failed.
  Forward/reverse patch checks and reference Python compilation passed before paid task calls.
- Fresh isolation canaries passed before the jobs. A login-shell probe could not reach MRW on
  the native PATH. B retained the existing PreToolUse write guard and pinned instructions.
- Seed 5030 froze job order; AB/BA alternated. With five pairs, three were A-first and two B-first.
  Both arms had identical initial manifests and task-prompt hashes within each pair.
- Each final exact-tree grade and full-check verdict was saved before deleting its task tree.
  All fifty sites and thirty-five changed file instances matched in each arm. No model failure,
  timeout, turn-cap error, missing cell or paid retry occurred.
- All usage/cost fields were known. Task calls totaled $0.2173879; setup included brings the
  CLI-reported token value to $0.2390969, below the $3 stop. These are subscription estimates,
  not an invoice or the cost of all development/review work.
- Before collection, the driver was hardened to forward cancellation and retain transcript hashes;
  preflight caught two bookkeeping mistakes, fixed before any counted task. The execution manifest
  and archived pre-task source identify the exact observed driver (`09a75b86…`), adapter and observer.
  After collection, a metrics guard was added for non-object/truncated stream rows.
  Replaying all ten retained streams through the final parser reproduces their stored metrics.
  Fresh review also found future failure/cancellation accounting gaps: attempted failures could be
  called unrun, and interruption outside the child wait could skip a summary. Both were corrected
  with focused regressions. These paths did not occur in the collected ten cells.
  No model calls were repeated after these changes; the archived observed driver remains the source of those observations.

Raw transcripts, stderr, configurations, exact source snapshots and immutable cell records are
retained outside the public checkout. The exported JSON contains fingerprints and sanitized
diagnostics. Scope is baseline/current tracked plus nonignored untracked paths; ignored files,
external dependencies and changes made then restored between snapshots are outside this unsigned
observer. This is not a sandbox or an independently reproduced build.

Local harness validation passed: 38 focused Python tests (6 driver, 11 adapter, 21 observer),
scoped ruff F/E9, workflow structure with `actionlint -shellcheck=`, the repository prose gate,
and `git diff --check`. No Go engine source or installed binary changed; the full Go/contract/static
release gates were not repeated for this local experiment.

## Interpretation and limits

The five MRW batch writes did complete fifty edit sites. Native batching removed the
assumed gap in transport calls: its ordinary Bash tool could also update a whole group.
This run therefore offers no support for a 50-native-turns-to-5-MRW-turns claim.

These were regular mechanical edits in five fresh contexts per arm, with one attempt per job.
They do not test a sustained fifty-round conversation, irregular bug fixes, concurrent writers,
or expensive repository test suites. Warm provider caches, a 3/2 order imbalance and service latency
limit timing/cost interpretation. Equal scores on these exact-byte fixtures do not prove general
quality equivalence. No significance, PAYS/LOSES, profile promotion or default-guidance change follows.
The earlier small pilot remains separate; the contrary result here is retained.
