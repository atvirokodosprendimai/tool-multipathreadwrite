# Batch renames in OpenCode

How many calls and bytes does a rename across several files cost through the OpenCode plugin, when `mrw_write` builds the plan itself instead of taking one written from a read? On 2026-11-26 we called the built plugin's tools (`execute`) against a real `mrw mcp` child, once per arm, on every task in the frozen Stage 9 `batch` class (`bench/stage-9/classes.json`).

- **Baseline:** the adapter as of `43aea360` (#806). One `mrw_read({grep, specs})` over the named files, then one `mrw_write({plan, ack})` with a hunk for each served line that held the old name, acknowledging every checkpointed run that arrived complete.
- **Candidate:** the adapter of #807. One `mrw_write({grep, sub, paths})`, with `paths` set to the named files.

Both arms used the same `mrw` binary, built from #807 at `0bb359da`, and a fresh ledger per task. The adapter's later commits up to its final head, `12409fe1`, changed only the tool descriptions, which no call output includes.

## Outcomes

Both arms produced each task's checked-in `reference.patch` on all 12 tasks, across pylint, pytest, Sphinx and xarray: 37 task/file pairs. Each fixture also held a file the task does not name, containing the old name. That file kept its original bytes in both arms.

| Route | Tool calls | Request bytes | Response bytes | Total bytes |
| --- | ---: | ---: | ---: | ---: |
| Baseline: grep read, then an acknowledged plan | 24 | 11,082 | 50,345 | 61,427 |
| Candidate: one scoped generated write | 12 | 2,026 | 24,570 | 26,596 |

The candidate moved 34,831 fewer bytes (56.7%). Per task the reduction ran from 53.5% to 59.7%:

| Task | Files | Baseline bytes | Candidate bytes | Reduction |
| --- | ---: | ---: | ---: | ---: |
| `s9-pylint-loop-exits-early` | 3 | 5,706 | 2,457 | 56.9% |
| `s9-pylint-augment-sys-path` | 3 | 4,581 | 2,025 | 55.8% |
| `s9-pylint-handle-config-commands` | 3 | 4,614 | 2,049 | 55.6% |
| `s9-pylint-open-graph` | 4 | 5,726 | 2,665 | 53.5% |
| `s9-pylint-parse-rich-type-value` | 3 | 5,664 | 2,366 | 58.2% |
| `s9-pytest-get-multicapture` | 3 | 5,495 | 2,351 | 57.2% |
| `s9-pytest-compare-eq-any` | 3 | 4,490 | 2,075 | 53.8% |
| `s9-xarray-iter-nc-groups` | 3 | 5,106 | 2,120 | 58.5% |
| `s9-xarray-broadcast-helper` | 3 | 4,838 | 2,107 | 56.4% |
| `s9-sphinx-normalize-options` | 3 | 5,465 | 2,201 | 59.7% |
| `s9-sphinx-input-mode-colour-func` | 3 | 4,722 | 2,086 | 55.8% |
| `s9-xarray-duplicate-dimensions` | 3 | 5,020 | 2,094 | 58.3% |

Request bytes are the UTF-8 length of the compact JSON arguments; response bytes are the UTF-8 length of the plugin's output.

## What the definition costs

The new arguments make the tool definition larger, and a host sends that definition whether or not a session uses them. The compact JSON Schema draft-7 projection of `mrw_write`'s id, description and parameters measured 2,780 bytes for the baseline adapter and 4,022 bytes at `12409fe1`: 1,242 bytes more per advertised definition. OpenCode 1.19.5's own registry, run against `12409fe1`, listed all 11 of `mrw_write`'s arguments.

## Limits

These figures are call counts and payload sizes for one class of task, renames whose files the task names. They are not an agent's cost. Model tokens, prompt caching, billing, wall time, and how often an agent picks this route over a read and a plan were not measured. A session that never makes a batch rename pays the larger definition and saves nothing.

The task instructions count occurrences inside longer identifiers, so both arms searched for the old name as a literal, not as a whole word. A generated write changes every match in the walk. Without `paths`, or with an empty list, the walk covers the whole checkout. The unnamed file kept its bytes here only because `paths` left it out.

## Method

For each task, each arm got a fresh git repository holding the files the task names, extracted from the upstream commit in its `task.json`. Each file's blob id was checked against that commit. Each repository also held an unnamed `unlisted.py` containing `# keep <old name>`. The task's `reference.patch` was applied to a separate git index only. After the arm's calls, `git diff --exit-code` against that index had to pass for the named files and the unnamed one, and the unnamed file had to hold its original line. Every call had to return without an error. The figures were first published in [the measurement comment on #807](https://github.com/atvirokodosprendimai/tool-multipathreadwrite/pull/807#issuecomment-7205207611).
