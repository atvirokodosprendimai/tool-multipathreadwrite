# ADR-160: Stage 4 measures mrw where its mechanism should matter, on tasks chosen by rule

**Status:** Accepted
**Accepted:** 2026-10-15 by Zy — picked "Accept (Recommended)", after the adr-challenger's 18 findings were taken into the draft.
**Date:** 2026-10-15
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-158, ADR-159, ADR-054
**Governs:** `internal/bench/**`, `cmd/bench/**`, `bench/**`, `docs/bench/**`
**Enforced-by:** None — until T1 lands, which adds `internal/bench/mine_test.go::TestAWideRenameCandidateIsDeclaredOnceAndNeverInAString`
**Invalidates:** none — checked: ADR-156's bar, admission and budget are reused unchanged; of ADR-156 Decision 3's corpus only the six `single-line` tasks are rerun, unchanged, as the control; `SelectSWEBench`'s `adr-156:` salt stays the default, so Stage 3's selection is unchanged; `redbase_test.go`'s ten red-at-base tasks stay ten, since a `check-coupled` candidate red at base is skipped
**Served-path change:** None — this ADR changes only measurement: `bench`, its corpus and its registration.

## Context

ADR-156 asks whether mrw pays for itself on agent runs, against a bar registered before any run (`docs/adr/BACKLOG.md`, "Pre-registered for ADR-156"). Every stage so far is INCONCLUSIVE, and the receipts show why: both configurations solve almost everything, so the solved difference cannot move and the verdict rests on cost alone.

- Stage 1b, Sonnet 5.5, 40 tasks × 2 × 2 (`docs/bench/stage-1b-sonnet/`): INCONCLUSIVE.
- Stage 2, Opus 5.5, the same 40 tasks (`docs/bench/stage-2-opus/`, `go run ./cmd/bench analyze docs/bench`): INCONCLUSIVE; 156 of 160 attempts solved; cost (B−A)/A −0.027 [−0.078, +0.021], solved B−A −0.025 [−0.075, +0.000]. Its largest `num_turns` is 22 (`qh-rename-git-lines.A.claude-opus-5-5.1`); the agentsmemory renames took 3 to 6.
- Stage 3, the SWE-bench Verified slice, running on this Mac on 2026-10-15 (`~/.local/share/mrw-bench/runs/2026-10-15-s3/stage3`, not yet published): 22 of the first 23 receipts solved.

The corpus's multi-site tasks change 3 to 6 places. mrw's claims are about wider edits (one plan for many sites, one read for many ranges), about a write that silently changes nothing, and about the check that runs after a write (ADR-054). A task both configurations finish in a few calls stresses none of them.

Zy, 2026-10-15: "draft the stage 4 ADR … design advanced-swe cases, where we will show the power of MRW vs standard tools". This record takes the first half as its scope — tasks in the categories where mrw's mechanism should matter, harder than the corpus so far — and declines the second half as a design goal: tasks are chosen by rules written before any run, difficulty is calibrated on configuration A alone and outside the sample, a control category where mrw is expected to lose is kept, and the bar is ADR-156's. A result that could only come out one way is what ADR-156 was written to prevent.

Measured 2026-10-15, to size the categories:

- SWE-bench Verified at revision `78f471bf655a3137b2e8a75af1501690ec009ec3` (500 instances; the file T5 of ADR-156 recorded, `~/.local/share/mrw-bench/swe-probe/verified-78f471bf.json`), counting `diff --git` lines and `^@@ -` hunk headers in each `patch`: 429 gold patches touch one file and one touches eight or more, so wide edits cannot come from it. 53 patches have five or more hunks; 51 once Stage 3's twenty are left out, and in 38 of those no patched file's path or basename appears in the problem statement. Their repositories: django 18, sympy 7, xarray 5, pylint 5, scikit-learn 5, sphinx 4, astropy 3, matplotlib 2, pytest 2.
- `git ls-files --eol` over the four own repositories' indexes at their checked-out commits finds no `i/crlf` file (agentsmemory 1,235 files, quality-harness 1,592, crossagentschat 32, the site 14); quality-harness's working tree has CRLF files that git converts. Of the seven SWE-bench clones, astropy's index has 1 CRLF file and matplotlib's 5.
- The agentsmemory clone `bench/repos.json` names (`~/Downloads/agentsmemory-main`) is shallow (its oldest commit is from 2026-08-16) and is checked out on a feature branch, `spec/retrieval-ablation-ladder`; its `origin/main` holds 1,785 commits, and the 17 existing agentsmemory tasks pin `d038def`. Mining needs the whole history, so T2 fetches it first.

## Existing Primitives Audit

- **Reused unchanged:** the runner, both configurations and their isolation (ADR-156 Decisions 1–2, ADR-158), the canary, receipts and resume (ADR-156 Decision 6), `bench run --task` and `--configs` (`cmd/bench/main.go:6-7`), the oracle with `exact` (byte equality of each touched file to the reference result), `bench verify` (it reads the configurations and models from `run.json`, `runner.go:352-356`), `bench analyze`, the SWE-bench path (ADR-156 T5, `internal/bench/swebench.go`) and its grading-interrupt handling (#425).
- **Reshaped:** `SelectSWEBench` (`internal/bench/swebench.go:23`) hard-codes the `adr-156:` salt and selects over every id it is given; it gains a salt and a hunk floor, so Stage 4's slice is drawn by the same function and `bench swebench-select` keeps its output for Stage 3's arguments. `bench analyze` prints one descriptive row per registered subset (`analyze.go:106`, ADR-158); it gains Stage 4's, and reads the slice's unnamed-files subset from the selection file, since it reads no gold patch (`swebench.go:49-58`).
- **New:** `bench mine`, which lists candidate tasks by the registered predicates. It uses only Go's standard library (`go/parser`, `go/scanner`, `go/types` with imports unresolved), since this module's `go.mod` declares exactly one requirement and a rename tool such as gopls would add one.

## Decision

1. **Four categories, each drawn by a rule registered before any run, in the order of `sha256("adr-160:" + key)`:**
   - **`wide-rename` — 10 tasks, agentsmemory at a pinned commit of `origin/main`.** A Go identifier declared exactly once in the module; occurring in no string literal and in no generated file (one whose first line matches `^// Code generated .* DO NOT EDIT\.$`); whose every whole-word occurrence outside comments is an identifier token; with 15 or more occurrences across 8 or more files, comments included. Key: the identifier. The new name Y has the same byte length as X, so no `gofmt` alignment changes (B's hook refuses `gofmt`, `internal/bench/hook.go:70`, and A may run it), is absent from the repository, and is written into the task before any attempt. The reference replaces every whole-word occurrence, comments included, and must pass `go build ./...`, `go vet ./...` and `gofmt -l` on the touched files; a candidate whose reference fails any of these is skipped for the next one, and the skip recorded — the build gate is part of selection (it filters a name that `go/scanner` cannot tell from a standard-library selector such as `resp.Header`). The instruction: "Rename the Go identifier X to Y, its mentions in comments included." Exact.
   - **`check-coupled` — 8 tasks, agentsmemory, commits of `origin/main` up to the pinned commit, merges excluded.** A commit of at most 4 files in which every touched file differs from its parent only by replacing one string literal's text with another, the same pair in every file, and which touches at least one non-test `.go` file and at least one `_test.go` file of the same package; the package's tests must pass at the parent in the tree bench builds. Key: the commit sha. The task pins the parent; the instruction is generated from a template: "Change the string <old> to <new> in <the non-test file>, and keep the tests passing."; the reference is the commit's diff. B's automatic check (ADR-054) shows the test that asserts the old text; an attempt that leaves it is unsolved. Exact.
   - **`swe-multi-hunk` — 20 instances, SWE-bench Verified** at the revision above, graded by swebench 5.0.2 as in Stage 3: the instances whose gold patch has 5 or more `^@@ -` hunk headers, Stage 3's twenty left out. Key: the instance id. Silent mis-edits are not measured, as on Stage 3's slice. A repository Stage 3 did not use is cloned and added to `bench/repos.json`.
   - **`single-line` control — the six existing `single-line` tasks** (ADR-156 Decision 3), rerun unchanged: the category where `measure.sh` says mrw loses, kept so the verdict is not over a population chosen for mrw.

   The stage's own-task list — the 10, the 8 and the 6 — is written to `bench/stage-4-tasks.txt` and passed to `bench run` as `--task` ids, since `bench run` without `--task` runs the whole corpus (`task.go:158-171`).
2. **Difficulty is calibrated on configuration A only, per category, on candidates outside the sample.** The pilot runs `wide-rename` candidates 11–14 and `check-coupled` candidates 9–12 × A × 2 on Sonnet 5.5, stage `pilot`, not counted. If A solves 7 or more of a category's 8 attempts, that category's floor is raised once — `wide-rename` to 25 occurrences across 12 files; `check-coupled` to a literal changed in 2 or more test files — its sample redrawn by the same rule, and the earlier draws deleted from `bench/tasks` before any counted run (none of them ran in one). If A still solves 7 or more on the new floor's candidates outside the sample, the category runs as drawn and is reported at ceiling. No task is kept or dropped for an outcome, and no measured task is a calibration task: a task A failed once by chance is likely solved on a rerun, which would read as B winning. A harness pilot runs the first 2 tasks of each own category in registered order × B × 1, and the first drawn instance of each SWE-bench repository Stage 3 did not use × A/B × 1, on Sonnet; their outcomes choose nothing. Stages 1 to 3 piloted A and B together on measured tasks; this one does not, because here the pilot chooses.
3. **The bar is ADR-156's, unchanged:** unit and pairing, outcomes, cost, PAYS / LOSES / INCONCLUSIVE with LOSES winning an overlap, admission and exclusions, B's counted shell reads and slipped writes, the silent mis-edit headline. One verdict per model and per task source — Stage 4's own tasks, and its `swe-multi-hunk` slice — never pooled with Stages 1 to 3. It fails, as registered, if B's cost interval lies wholly above 0 or its solved interval wholly below 0; nothing here can stop that from happening.
4. **The budget stays 20 minutes or 60 turns** (ADR-156's registration). Whether a 15-site rename made one `Edit` at a time comes near 60 turns is not measured; the most any Stage 2 attempt used is 22. An attempt past the budget is unsolved with its cost counted, in either configuration, as registered.
5. **Descriptive rows, registered now, printed beside the verdict and never in its place:** one per own category; for `swe-multi-hunk`, the instances whose problem statement contains none of the patched files' repository paths and none of their basenames other than `__init__.py`, written into the selection file by `bench swebench-select`.
6. **Sample:** the own tasks (10 + 8 + 6 = 24) × 2 configurations × 2 repeats, and the SWE slice × 2 × 1, on Sonnet 5.5 then Opus 5.5 — 136 attempts per model. Each stage starts only on Zy's word. Power, said first: 24 paired tasks cannot show a small solved difference; INCONCLUSIVE is a result.

What would make it FAIL is the same as ADR-156's: an oracle grading a reference fix unsolved or a stray edit solved, configuration A seeing mrw, a verdict computed from attempts the rules did not admit, a task chosen or dropped for an outcome, or a README sentence stating more than the verdict.

## Alternatives Considered

- **Hand-pick tasks where mrw is known to win.** Rejected: the verdict would describe the author's choices, not mrw. Zy asked to show mrw's power; this record shows it only if it is there.
- **Calibrate on the sampled tasks, or keep the tasks A failed.** Rejected: both select on outcomes, and a chance failure regresses on the rerun in B's favour. Calibration runs outside the sample.
- **Pick each new name freely.** Rejected: a longer or shorter name makes `gofmt` re-align fields, which only A can do, so `exact` would grade formatting.
- **Write `check-coupled` instructions by hand from commit messages.** Rejected: a sentence cannot pin byte equality, and its wording is an author choice; the template can.
- **A CRLF / byte-order-mark category from synthetic fixtures.** Rejected: no own repository's index has a CRLF file (measured above), and ADR-156 Decision 3 never measures fixtures, since a task written to exhibit the failure measures the writer. ADR-142's contract rows cover the behaviour.
- **A wide-edit slice from SWE-bench.** Rejected: one Verified instance touches eight or more files. Multi-hunk within files is what SWE-bench has.
- **Raise the budget for wide edits.** Rejected for now: the budget is part of ADR-156's bar, and changing it for one stage makes the stages harder to read side by side.

## Component / Boundary Impact

None — internal to `bench` (`internal/bench`, `cmd/bench`): one subcommand (`bench mine`), two shape names, a salt and hunk floor on the SWE-bench selection, and descriptive rows in `bench analyze`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench mine --kind wide-rename\|check-coupled --repo DIR --commit SHA [--n N]` | new subcommand, prints candidates in registered order | T1 | T2 |
| `bench swebench-select --salt S --min-hunks N --exclude FILE --dataset FILE` | new flags; without them, today's output; with `--dataset`, the unnamed-files subset in its output | T1 | T2 |
| `bench analyze --selection FILE` | new flag: reads a slice's subsets for its descriptive rows | T1 | T5 |
| `task.json` `shape` | adds `wide-rename`, `check-coupled` | T1 | T2, `bench analyze` |
| `bench/stage-4-tasks.txt` | new: the 24 own-task ids | T2 | T3, T4 |
| `bench/repos.json` | the repositories the slice draws that Stage 3 did not use | T2 | the runner |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `bench mine`, `swebench-select --salt/--min-hunks`, `analyze --selection`, the two shapes | T1 | T2, T5 | No — additive |
| `bench/tasks/<stage-4 ids>/`, `bench/stage-4-tasks.txt`, `bench/swe-bench/selection-adr-160.json` | T2 | T3, T4 | No — new files |
| the calibrated corpus and the pilots' sign-off | T3 | T4 | No |
| `docs/bench/stage-4-*/` receipts | T4 | T5 | No |

## Implementation

See `tasks/README.md`: T1 selection code, T2 the corpus, T3 the A-only calibration and the harness pilots, T4 the stages on Sonnet and Opus, T5 analysis and the README sentences.

## Consequences

- **Positive:** a stage whose tasks can separate the configurations, chosen so the result says something about mrw rather than about the author.
- **Negative:** about 272 counted attempts and some 30 pilot attempts; twenty more SWE-bench image pulls (around 10 GB each, one at a time); two or more repositories to clone for the slice.
- **Neutral:** Stage 4 may well come out INCONCLUSIVE or LOSES; either is published like PAYS.

## Out of Scope

- CRLF and byte-order-mark edits (permanent: fact: the four own repositories' git indexes hold no CRLF file, measured 2026-10-15; citation: file `docs/adr/ADR-160-stage-4-measures-mrw-where-its-mechanism-should-matter.md:28`)
- Indentation-sensitive edits as a new category (permanent: boundary: ADR-156's six `structural` tasks already measure YAML and markdown structure, and `exact` catches a reparented key)
- Other hosts and mrw over MCP (deferred: docs/adr/BACKLOG.md — "Pre-registered for ADR-156", not measured now and recorded so it is not forgotten)
- Concurrent writers in one checkout (permanent: boundary: the single-agent bench runs one attempt at a time and cannot measure them)
- A budget other than ADR-156's (deferred: docs/adr/BACKLOG.md — "From ADR-160", Zy's call after the pilot)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Fewer than 14 `wide-rename` or 12 `check-coupled` candidates meet the predicate | Med | High | T2 counts them before building anything; a short category stops and goes back to Zy |
| The comment-and-string rules leave few `wide-rename` candidates (agentsmemory documents its exported functions) | Med | Med | comments are allowed and renamed; only strings and generated files exclude |
| Both configurations still solve almost everything | Med | Med | the A-only calibration raises the floor once; a category at ceiling is said so, not hidden |
| A new SWE-bench repository's image or clone behaves unlike Stage 3's | Med | Med | the harness pilot runs its first instance before the stage |

## Rollback

None — measurement only: the new task directories, the list and selection files and the stage receipts can be removed; nothing on mrw's served path changes.

## Follow-ups

None yet.
