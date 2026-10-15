# Stage 4's draw (ADR-160 T2)

What T2 drew its own tasks and its SWE-bench slice from, on 2026-10-15, with the commands that drew them. The rule is
ADR-160 Decision 1 as implemented by `bench mine` and `bench swebench-select` at the drawing commit `8c9bba92`.

## The pin

agentsmemory, `~/Downloads/agentsmemory-main`, after `git fetch --unshallow origin`: `main` and `origin/main` at
`d038def25a7fadd2586674b954ea07896a330b4a`, 2,092 commits, `git rev-parse --is-shallow-repository` false.

## `wide-rename`

```
bench mine --kind wide-rename --repo ~/Downloads/agentsmemory-main --commit d038def25a7fadd2586674b954ea07896a330b4a
```

93 candidates, [mine-wide-rename.txt](mine-wide-rename.txt), in registered order. Each reference was built with a
placeholder name first ([gate-wide-rename-placeholder-names.txt](gate-wide-rename-placeholder-names.txt)): three of the
first seventeen were passed over for `go build`, `Clean`, `Contains` and `Handler` (and `Node`, candidate 20, for `go vet`). The names
([names.tsv](names.tsv)) were then chosen, each of the old name's byte length and absent from the tree, and every
reference rebuilt with them ([gate-wide-rename.txt](gate-wide-rename.txt)). The first ten are the sample; the next
four (`hookPlans`, `homeDir`, `Harness`, `jsonResult`) are T3's calibration candidates, their references in
[calibration/wide-rename/](calibration/wide-rename/).

### The raised floor (Decision 2, after T3's pilot)

A solved 7 of its 8 calibration attempts in the first pilot (`docs/bench/stage-4-pilot/calibration/`; the one miss
was BSD `sed` given `\b`, which changed nothing), so the floor was raised once, as registered, and the sample redrawn
by the same code:

```
bench mine --kind wide-rename --repo ~/Downloads/agentsmemory-main --commit d038def25a7fadd2586674b954ea07896a330b4a \
  --min-occurrences 25 --min-files 12
```

36 candidates, [mine-wide-rename-25.txt](mine-wide-rename-25.txt), in the same registered order. The build gate passed
over `Clean`, `Contains` and `Handler` (`go build`) and `Node` (`go vet`) again, and `Harness` because it was a
calibration task the first pilot ran on A, and no measured task is a calibration task
([gate-wide-rename-25-placeholder-names.txt](gate-wide-rename-25-placeholder-names.txt)). The first ten that hold are
the sample (`WithTenant` and `configFromCmd` drawn again, with the instruction as amended, "in the Go files"); the next
four (`AddInput`, `Migrations`, `EntryRoom`, `WithReranker`) are calibration-2's candidates. Names:
[names-25.tsv](names-25.tsv); every reference rebuilt with them passes
([gate-wide-rename-25.txt](gate-wide-rename-25.txt)). The earlier draws, none of which ran in a counted stage, were
removed from `bench/tasks`; the first pilot's receipts keep their ids.

## `check-coupled`

```
bench mine --kind check-coupled --repo ~/Downloads/agentsmemory-main --commit d038def25a7fadd2586674b954ea07896a330b4a
```

310 candidates, [mine-check-coupled.txt](mine-check-coupled.txt). Each was gated in order on an export of the pinned
tree: the package's tests pass there, at least one fails with only the source literal changed, and they pass on the
reference (the literal changed in the source and in every test file of its directory that holds it). Of the first 32,
12 hold and 20 were passed over: 10 because no test asserts the source literal alone, 10 because a test the reference
does not reach still asserts it (such as the tool list in the README, the emphasis list, the hook channels, a recorded
baseline). The first pass ran at load 45 to 58 ([gate-check-coupled-first-pass.txt](gate-check-coupled-first-pass.txt)),
so it was run again at load 4 ([gate-check-coupled.txt](gate-check-coupled.txt)): every verdict is the same. The first
eight that hold (candidates 1, 3, 4, 8, 13, 15, 22, 23) are the sample; the next four (28, 30, 31, 32) are T3's
calibration candidates, in [calibration/check-coupled/](calibration/check-coupled/).

After T3's pilot the category is graded by its tests (Zy, "Grade by the tests"): each task names its package as
`test_package`, is not `exact`, and holds `require` and `forbid` on the source file only. The `forbid` is written only
where the reference source no longer quotes the old literal: in `doctor.go` and `querypolicy.go` a comment quotes the
value, so those two tasks rest on the `require` and the package's tests.

## `swe-multi-hunk`

```
bench swebench-select --salt adr-160 --min-hunks 5 --exclude bench/swe-bench/selection.json \
  --dataset ~/.local/share/mrw-bench/swe-probe/verified-78f471bf.json --n 20
```

The output is `bench/swe-bench/selection-adr-160.json` with the dataset, revision, split, instance count and harness
added as in Stage 3's file; rerunning the command reproduces its `selected` and `unnamed` lists. `pylint-dev/pylint`
and `pydata/xarray` were cloned and added to `bench/repos.json`; every `base_commit` of the twenty is present.

## The task list

`bench/stage-4-tasks.txt`: the ten `wide-rename` ids, the eight `check-coupled` ids and the six `single-line` controls.
