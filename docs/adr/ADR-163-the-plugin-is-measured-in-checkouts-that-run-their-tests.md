# ADR-163: Bench runs SWE-bench attempts in checkouts that can run their tests, for quality-harness ADR-112

**Status:** Accepted
**Accepted:** 2026-10-20 by Zy — picked "Accept (Recommended)" after two challenger reviews (adr-challenger, 22 findings; Codex xhigh in two scoped parts, 7 findings, then a re-check that closed them and raised 1 more), every finding that held taken into the draft; quality-harness agreed its two pre-registration points (D1 read at 1200 s; the surviving pool frozen before its pre-registration)
**Amended:** 2026-10-21 by Zy — "amend, accepted", then "Fix #477 in place" after a Codex xhigh challenge (2 P1, 4 P2), then C redefined as `turnEnd: notify` (Zy, 2026-10-21) after the re-check showed `continue` leaves a runnable declared check's continuation unchanged: Decision 9 and T6 add the side arms G (Go hooks) and C (the quiet Stop, `turnEnd: notify`), asked for by quality-harness (its ADR-110 T2 S3; ADR-120, from this repository's 2026-10-21 field report §1)
**Date:** 2026-10-19
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-156, ADR-162; quality-harness ADR-106, ADR-111 and ADR-112 (`/Users/zy/CursorProjects/quality-harness/docs/adr/ADR-112-the-lifecycle-is-measured-where-it-can-work.md`, Accepted 2026-10-18)
**Depends-on:** ADR-162 T5's merge to `main`, which brings `docs/bench/stage-5/` (block 1's receipts and the amended scan) that this record cites; until then they are on the pushed branch `adr-162-t5-stage-5` at 4eec8caa, and this record re-cites the squash commit when it lands
**Governs:** `internal/bench/**`, `cmd/bench/**`, `bench/**`, `docs/bench/**`
**Enforced-by:** None — until T1 lands, which adds `internal/bench/container_test.go::TestStage6ShimsRunInTheInstanceContainer`
**Invalidates:** none — checked: without `--profile stage6` every configuration's arguments, PATH and tools stay as ADR-162 left them, so Stages 1 to 5 stay reproducible; `MaxTurns` stays 60 unless a stage sets another; Stage 5's receipts, looks and stop rule are untouched
**Served-path change:** None — this ADR changes only measurement: `bench`, its registration and its stage.

## Context

Stage 5 block 1 (ADR-162) measured the quality-harness plugin (P) in checkouts that cannot run their tests: the agent's Bash runs on macOS in a plain checkout, and the swebench environment exists only inside each instance's image, which bench uses for grading. Measured 2026-10-18 from block 1's 54 P transcripts:
- no `qh-check` run passed; it reported unstarted 13 times, failed 11 times (several of those missing modules, not test results) and unproven 5 times;
- P made 0 `Skill` calls, so the plugin's lifecycle (specs, records, tasks) never ran;
- P's false completions on Opus equalled A's (6 of 39 and 6 of 40), and its broke-tests count did not fall (Opus P 3, A 2; Sonnet 6 and 6).

Quality-harness ADR-112 (Accepted) pre-registers a stage that measures the plugin where its mechanism can work, and names this runner for it: arms A, P and W (P plus one prompt line asking the agent to run `/quality-harness:work` first), every arm "in the instance's container through the bench's shims", the same `.quality-harness.json` written before the base commit, its `check` from the repository alone. On its side it fixes the choice between the repository's whole runner and a scoped check (D1), the instance rule and the surviving list's digest (D2), the attempt caps (D3), the endpoints (D4), blocks of 40 with PAYS and FUTILE at each look at 0.0125/L_max, L_max being the surviving pool's block count (D5), and what the verdicts decide (D6). Its scorer (`scripts/eval-paired.mjs:349-375`) refuses a pool whose sha256 or count differs from the pre-registration, and a scored list that is not a block prefix of it. Its Out of Scope names this ADR for the shims and configuration W.

Zy, 2026-10-19: "Draft now, run after": draft this record and build its code while Stage 5's A/B blocks run; the pilots and the stage start only after Stage 5 closes, so they never compete with it for the weekly usage limit or the machine.

**Measured on this Mac, for the design:**
- **Images** (`internal/bench/swebench.go:256-264`): each instance's image is linux/amd64 only, so on this arm64 Mac it runs emulated. ADR-162 measured one SWE image at about 10.6 GB (2026-10-17); each is pulled once per instance and removed after grading.
- **The tools an attempt finds** (`internal/bench/runner.go:373`, `:794-815`): `toolPath` links `git`, `go`, `gofmt`, `node`, `npm`, `npx` and the host `python3`; PATH is that directory then `/usr/bin:/bin:/usr/sbin:/sbin`.
- **Host interpreters and runners outside PATH** (listed 2026-10-19 with `ls /opt/homebrew/bin /usr/bin` filtered by `python|pip|py\.|pytest|uv|tox`): `/usr/bin/python3`, `/usr/bin/pip3`; `/opt/homebrew/bin/python3`, `python3.14`, `python3.15`, `pytest`, `py.test`, `pip3`, `pip3.14`, `pip3.15`, `uv`, `uvx` (13 entries, plus `xcrun python3`). An absolute path, `env -i` or `command -p` reaches the system ones whatever PATH holds.
- **Docker** is not on an attempt's PATH (ADR-158), but `/usr/local/bin/docker` and `~/.docker/bin/docker` both resolve to `/Applications/Docker.app/Contents/Resources/bin/docker`.
- **The pool** (`bench/swe-bench/selection-adr-162.json`): 191 instances, 78 django, 27 sympy, 22 sphinx, 16 scikit-learn, 14 matplotlib, 9 pytest, 9 astropy, 7 xarray, 4 requests, 3 pylint, 2 seaborn. The 40 instances of `selection.json` and `selection-adr-160.json` lie outside it and cover the same repositories.
- **Cost of a block today** (Stage 5 block 1, 245 receipts, `total_cost_usd` as the CLI reports it): $46.80 in all (Opus A $11.64, B $8.82, P $17.71; Sonnet A $2.63, B $1.99, P $4.01); about 10 to 13 points of the 7-day limit, other sessions' use included. Mean agent wall time 1.31 min per receipt, grading not included; ADR-162 measured about 2 minutes per attempt with grading on Stage 4b. W, and every arm with runnable tests, are unmeasured.
- **The turn cap** (`runner.go:51`, `:719`; `internal/bench/export.go:158`): `MaxTurns` is a constant, 60, read by the over-budget rule and written into ADR-106's rows. ADR-112 D3 needs a cap set per stage.
- **What a swebench image keeps of the repository's history.** Unverified on this machine. An image built from a clone may keep the upstream fix in its object store or reflog; T1 S1 inspects one image, after Stage 5, before any decision rests on it.

**The class this record governs:** every way an agent's command in a `stage6` attempt can run Python, a test runner or a container. Enumerated by the command above (13 host entries) plus the names the pool's documented runners use: `python`, `python3`, `pytest` and `py.test` by name, and shebang scripts that resolve them through PATH (sympy's `bin/test`, django's `tests/runtests.py`), and `make` targets that call `python` by name (scikit-learn), all reach the shims. Absolute host interpreter paths, `env -i`, `command -p`, `xcrun python3`, a script whose shebang names an absolute host interpreter, and an absolute `docker` do not, and are counted per transcript (Decision 6). `pip`, `uv` and `tox` would need the network, which no container has.

## Existing Primitives Audit

- **Reused unchanged:**
  - the runner's attempt loop, retries, resume, receipts and exclusions (ADR-156 Decision 6);
  - the Stage 5 profile's parts (ADR-162 Decision 1): skills on, `Skill` offered, one real `node`, the Codex stub, no synced skills or plugins, no `Agent`;
  - configuration P and its canary (ADR-162 T1), the `STATUS:` line and the harness's own result (ADR-162 T2);
  - `bench export-paired` (ADR-162 T2), which already takes `--pair`;
  - the SWE-bench grading path: one image per instance, swebench 5.0.2, removal after the instance;
  - `--block` and `--configs-for` (ADR-162 T3);
  - Decision 8's contamination scan as amended after Stage 5 block 1 (`docs/bench/stage-5/scan.py`).
- **Reshaped:**
  - `toolPath` gains the shims under the `stage6` profile, in place of the host `python3`;
  - `Materialize`'s SWE-bench path writes the declared check before the base commit, under the profile;
  - `MaxTurns` becomes the stage's cap, written in `run.json`, defaulting to 60;
  - `export-paired` takes its instance list from a selection-shaped file, so it can export against the surviving pool, and binds `max_turns`, `checks_sha256` and `work_line_sha256` per block.
- **New:**
  - the per-attempt container and the shims (T1);
  - configuration W and the checks table (T2);
  - `bench baseline`, the surviving pool and the block plan drawn from it (T3).

## Decision

1. **Every configuration runs its tests inside the instance's own container, through shims, under a `stage6` profile.**
   - **The container.** Before the agent starts, bench starts one container of the instance's image for the attempt, with no network (`--network none`). Before any agent command reaches it, bench makes `/testbed`'s tracked files exactly the host tree's base commit, the one bench prepared (instructions, the declared check), keeping the image's ignored build output, and replaces `/testbed/.git` with a fresh repository holding that one commit, so the host and the container start from the same tree and no history, reflog or upstream object survives inside it. Any evaluation artifact T1 S1 finds in an image (a test patch, an evaluation script, a list of test names) is removed by a fixed rule before the agent's first command, in every image; one that no fixed rule can find stops the stage for Zy. The container is removed when the attempt ends, on every path, interrupted ones included.
   - **The shims.** `python`, `python3`, `pytest` and `py.test` in the attempt's tools directory are shims. Each one syncs the agent's change into the container and runs its command there under the image's `testbed` environment, with the agent's working directory and every argument that names a path in the attempt's tree mapped to `/testbed`. Stdout, stderr and the exit code come back unchanged. A working directory outside the tree is refused, exit 2, saying why. Each shim logs its own call, so every Python or test command the agent ran is accounted for.
   - **One command at a time.** A shim holds a per-container lock from its sync to its command's end, so a background call (ADR-111's session-start baseline in P and W, or a backgrounded Bash) and a foreground call never share `/testbed` at once. The time each call waited is logged and summed per attempt, and reported per arm.
   - **The sync, in.** The container's `/testbed` is reset to its one commit, keeping ignored build output (`git reset --hard` and `git clean -fd`, never `-x`), and then takes the tree's binary diff against bench's base commit plus its untracked files. The diff is computed with a temporary `GIT_INDEX_FILE`, so the agent's own index and staging are never touched. A diff that does not apply is the command's own failure: the shim exits non-zero and says so on stderr, the count goes in the receipt per arm, and the attempt is graded as any other, since the harness grades the host diff in a fresh container whatever the shim did.
   - **The sync, back.** A command run through a shim may write files, as an agent's `python - <<EOF` edit does. After the command, every tracked or untracked file it changed in `/testbed`, outside what `.gitignore`, `__pycache__/`, `*.pyc` and `.pytest_cache/` cover, is written back into the host tree, so what the agent changed through Python is in the diff the harness grades, exactly as on a host. A file the host changed too since the sync is a conflict: nothing is written for it, the shim exits non-zero naming it, and the count goes in the receipt per arm.
   - **What stays on the host.** The agent, its tools, `git`, `node` and the plugin's hooks run on the host as in Stage 5. A hook that runs the declared check reaches the container through the same shims.
   - **The environment.** The shim forwards every variable the agent's command set or changed, with tree paths mapped; never forwarded are PATH, HOME, TMPDIR, USER, PWD, OLDPWD, SHLVL, _, TERM, SHELL and the MRW_, BENCH_, CLAUDE, ANTHROPIC_, CODEX_, XDG_, GIT_ and BASH_ families (LANG and LC_* are forwarded), so PATH, HOME and TMPDIR stay the container's.
   - **Without the profile** every configuration's tools and PATH are exactly ADR-162's.
2. **The declared check comes from the repository alone, is frozen before any baseline, and every arm gets the same file.**
   - **The table.** `bench/swe-bench/checks.json` names one command per repository in the pool: the whole runner the repository's own files document, with no path or subset arguments, and the file, line and commit it came from (for example django's `tests/runtests.py`, sympy's `bin/test`, `python -m pytest` where `setup.cfg`, `tox.ini` or `pytest.ini` configures pytest). Nothing in it is read from the dataset: never FAIL_TO_PASS, PASS_TO_PASS, the test patch or the gold patch.
   - **Frozen.** Its SHA-256 is written into the BACKLOG's "Pre-registered for ADR-163" section before T4's baseline pilot. An amendment after that reruns the baseline pilot on instances it has not used.
   - **The file.** Under the profile, `Materialize` writes `.quality-harness.json` with that `check` into every arm's tree before bench's base commit, so it is in A, P and W alike and never in the diff the harness grades. An agent that edits it changes its own diff, as with any other file. Side arm C's file carries one more key, `turnEnd` (Decision 9); its `check` is the same.
   - **The form.** This is the whole-runner form of ADR-112 D1. Whether the stage may use it is ADR-112's rule, read on T4's baseline pilot against a 1200 s attempt budget, the floor ADR-112 D3 sets, because D3's own value is known only after the agent pilot. Quality-harness agrees that reading in its pre-registration before T4. If the rule fails, the stage waits for quality-harness's scoped form, and this record is amended before anything is counted.
3. **Configuration W is P plus one prompt line, and the caps are the stage's.**
   - **W.** W has P's tools, profile and plugin, and its instruction carries one more line: the exact text quality-harness's ADR-112 pre-registration names, whose SHA-256 is in `run.json`. The canary proves W engaged as P, and that the line is in W's prompt and in no other configuration's.
   - **The caps.** `bench run --budget` and a new `--max-turns` set the wall-time and turn caps for every arm of the stage. They are written in `run.json`, in ADR-106's rows, and inside the rows' `scaffold`, so quality-harness's scorer compares them with its pre-registration. ADR-112 D3 fixes their derivation, registered before the agent pilot: three times W's median wall time and turns over both models' pilot attempts together, never below 1200 s and 60. Their numbers are registered after the agent pilot and before block 1. The agent pilot itself runs at 1200 s and 60 turns, so its medians may be censored, and that is said beside them.
   - **A run that hits a cap.** Under `stage6`, `export-paired` writes a capped run as unresolved with no done claim (`status` `not done`, `resolved` false, `limit` true), as ADR-112 D3 requires, whatever its claim line and the harness said; the receipt keeps both.
4. **The surviving pool is frozen before the first counted attempt.**
   - **`bench baseline`.** For each instance it runs the declared check once on the unmodified tree inside its container, and records green or red, the exit code and the wall time in `baseline.json`. A baseline uses no model and no usage. One run decides: a red is never rerun to make it green.
   - **The whole pool, before the pre-registration.** After D1 passes and before quality-harness pre-registers, `bench baseline` runs over all 191 pool instances. The green ones, in ADR-162's pool order, are the surviving pool, written as a selection-shaped file (`bench/swe-bench/selection-adr-163.json`). The scorer's `--pool` is its `.selected` ids one per line, and its digest is the sha256 of those ids joined by a newline with no trailing newline, the convention ADR-162's export and ADR-106's scorer already share. That digest and the count are what ADR-112 D2's pre-registration carries; a look's `--instances` is the first 40·L of the same list. The count of red instances is published by rule, never the ids (quality-harness's §6).
   - **ADR-106's pilot is already outside it.** ADR-112 D2 excludes ADR-106's pilot from the pool. That pilot (ADR-162 T4, `docs/bench/stage-5-pilot/`) ran 2 instances of `selection-adr-160.json`, and ADR-162's 191 exclude every instance of that selection and of `selection.json` (ADR-162 Decision 4), so no exclusion is left to take; checked 2026-10-19, neither pilot instance is among the 191. T4's baseline pilot draws from the same two selections, so it is outside the pool too.
   - **The blocks.** Block K is instances 40(K−1)+1 to 40K of the surviving pool; the last block may be partial. L_max is the surviving pool's block count.
5. **The stop rule is ADR-112's, and bench follows it.** After each block, bench exports the rows for each model and pair (`--pair A,P`, `--pair A,W`) against the surviving pool. Quality-harness's scorer, at the blob its pre-registration names, reads them with `--verdict-out`, one file per model and pair, committed here as `looks/look-K/<model>-<pair>.json` beside its printed output. A look is valid only when all four files exist and each carries one of the scorer's words for a look; a missing measurement is not a look, and the next block waits. A pair once PAYS or FUTILE stays stopped. The next block runs each model with A and the arms whose pair is still open, and its `run.json` must say exactly that. A model whose pairs have both stopped runs no more blocks. The stage ends when every pair has stopped or the surviving pool is spent. bench's own `bench look` is not used for this stage, because its verdicts are mrw's. The pair-symmetric recall row (ADR-112 D2) is computed by quality-harness from each block's `recall.txt`, as ADR-106 T3 did.
6. **Contamination: the container is clean, and the transcripts are scanned.**
   - **Inside the container:** no network, no history, no dataset. The dataset, the probe venv and the Hugging Face cache stay on the host as in Stage 5.
   - **After each block:** the amended Decision 8 scan (`docs/bench/stage-5/scan.py`) runs, a hit stops the stage for Zy, and its RECALL lines are committed beside the block.
   - **Counted per transcript and printed beside the verdicts:** commands naming an absolute host interpreter or runner (the 13 entries above, `xcrun python3`), `env -i` or `command -p` before one, and any Python or test command in the transcript with no matching shim log entry.
   - **A stop, as a scan hit is:** any Bash command naming `docker`, an image of the pool, or the container, since a fresh container of the image would have the network and the original `/testbed/.git`.
   - **Decided at the agent pilot:** if the pilot's host-interpreter count is not zero, the host interpreters are hidden from the attempt before block 1, and this record says how.
7. **Two pilots, after Stage 5 closes.** Stage 5 closes when ADR-162 T5's `bench look` exits 0, or when Zy ends it.
   - **A baseline pilot, first and without any model.** `bench baseline` over 10 instances outside the pool: in file order from `selection.json` then `selection-adr-160.json`, the first 4 django, 2 sympy, and 1 each of sphinx, scikit-learn, matplotlib and pytest. The list is derived by that rule and committed before the pilot runs. It measures what ADR-112 D1's rule reads: how many are green, and the median wall time against 1200 s. If the rule fails, the stage stops here and goes to Zy and quality-harness, having spent no usage.
   - **The whole-pool baseline,** only if D1 passes (Decision 4). No usage; machine time and image pulls only.
   - **An agent pilot,** only if D1 passes and quality-harness's pre-registration is committed. The first 4 green pilot instances × A/P/W × both models, 24 attempts, at 1200 s and 60 turns. It records every arm's median wall time, turns and cost (ADR-112 T2 asks for all three), sets D3's caps from W's, reads one W and one P transcript whole, and gives Decision 6's counts.
   - **The price goes to Zy** from the pilots' numbers, for the whole surviving pool (ADR-112's follow-up).
8. **Price, said first and unmeasured.**
   - **The whole-pool baseline:** 191 image pulls of about 10.6 GB each, one at a time, and one whole-suite run each, at most a quarter of the budget if D1 passes. A guess of 25 to 35 machine hours. No usage.
   - **A block:** 40 instances × the open arms × 2 models, up to 240 attempts. Stage 5 measured about 2 minutes per attempt with grading on Stage 4b, with no tests run; attempts that run real tests under emulation take longer, and W longer again. A guess of 25 to 40 machine hours, and 15 to 25 points of the 7-day limit, per full block.
   - **The stage** may need every surviving block, since one block cannot show PAYS (ADR-112 D5).
   - **The likely outcome of the baseline pilot.** The pool is 41% django, and Django's whole suite is long, so D1's quarter-budget rule may fail on the first pilot, before any usage is spent. That outcome is a result.
9. **Two side arms, G and C, run at the stage's one plugin tag, are read against P alone, and never touch ADR-112's rows or stop rule** (amendment of 2026-10-21, revised after a Codex xhigh challenge the same day).
   - **G** is P with the plugin's hooks run by its Go binary, `qh`, instead of Node. The question is quality-harness's (ADR-110): does the Go runtime change turns, cost or wall time?
     - The hooks run on the host, so G needs `qh-darwin-arm64` from the stage's frozen plugin release, checked against that release's `SHA256SUMS`.
     - G runs only if that tag selects the hook runtime by a documented setting, with the same hook set, matchers and timeouts as Node. Possessing the binary is not enough: quality-harness's hook manifest calls Node directly until its ADR-116 wires a launcher.
     - bench pre-stages the verified asset, turns the plugin's own fetching off, and records the sha in the side metadata.
     - Every plugin arm (P, W, G, C) gets an equivalent fresh plugin data directory per attempt, so state isolation is not a second difference.
     - The canary proves that P's hooks ran under Node and G's under the staged binary.
   - **C** is P with one setting: `"turnEnd": "notify"` in the attempt's `.quality-harness.json`. With it, a turn-end finding goes to the user alone and the turn does not continue (`channelOf` in quality-harness's `plugin/scripts/check-ledger.mjs:537-540`, `turnEndSetting` in `project-config.mjs:63-67`, read at `c749638e`). Without it, a declared check that can start continues the turn when work is unchecked, at every tag through `c749638e` (quality-harness ADR-120 T6 and T11: "a declared `node --test` continues as today"). So at one tag, P is the plugin as shipped and C is the same plugin with the continuation off.
     - Stage 5's P, at 3.8.40, continued 10 of 40 Sonnet and 25 of 40 Opus attempts in checkouts that could run no test. It cost +49% (Opus) and +70% (Sonnet) over A with no change in solved (ADR-162, look 1).
     - Under `stage6` the declared check can start, so the pair measures what the continuation costs or saves where a check can run. It does not measure Stage 5's waste again, which is already measured. `"turnEnd": "continue"` (ADR-120 T11, unreleased at `c749638e`) changes only inferred and unstartable checks, so it would leave C equal to P here and is not the setting. If the frozen tag spells the setting differently, its release notes are the authority and the registration names the spelling.
   - **One tag, no late joiners.** The stage's plugin tag is frozen before T4's agent pilot (Decision 7). A side arm is in the stage only if it works at that tag, and its asset or setting is named in BACKLOG's "Pre-registered for ADR-163" before the agent pilot; otherwise it is not run in this stage. A side arm then runs, for each model, in every block in which that model's P runs, the pilot included, and never joins later.
   - **ADR-112's rows stay byte-identical.**
     - `run.json` keeps `configs` and `configs_for` for A, P and W alone, and carries the side arms in their own fields: `side_admitted` (the arms admitted at the tag, the same in every block), `side_configs_for` (for each model, the admitted arms where that model's block runs P, and none where it does not) and `side`.
     - `export-paired --pair A,P` and `--pair A,W` build `plugin` and `scaffold` from the core arms only, so a block with side arms exports exactly the rows it would without them.
     - T6 tests this with side arms absent and present.
   - **How they are read.** A new reader, `bench side`, reads receipts with P as the control and each side arm as the other side, paired by task. It uses the intervals `bench analyze` computes: the percentile bootstrap for cost and Tango's score interval for solved, at 99%. It reports turns and wall time beside them, against the bar registered in BACKLOG's "Pre-registered for ADR-163" before the agent pilot. ADR-112's scorer never reads a G or C row, and a side arm never stops or prolongs the stage.
   - **Sample.** A side verdict is read only once an arm has at least 60 paired tasks per model. At 40 pairs with all outcomes concordant, the 99% Tango lower bound is -0.142, wider than the 10-point margin; at 60 it is -0.0996. The pilot never counts. If P's pairs stop before 60, the side rows are published as descriptive only.
   - **Price.** Each side arm adds up to 80 attempts per full block (40 instances × 2 models), so a full block with both is up to 400 attempts. The agent pilot adds 8 attempts per side arm (4 instances × 2 models). The price asked after T4 says it per arm.

What would make it FAIL:
- an agent command that ran tests on the host while the transcript shows the shim's path;
- a container with network, history, an evaluation artifact or a dataset path, or two commands sharing it at once;
- a container that starts from another tree than the host's base, or a file the agent changed through a shim that the graded diff lacks;
- a capped run exported with a done claim or as resolved;
- a look taken from a missing measurement, or a stopped pair run again;
- a declared check derived from anything the dataset chose, or changed after the baseline pilot without rerunning it;
- the check file present in some arms and not others, or in the graded diff as written;
- a W whose line differs from the pre-registered text, or appears in P or A;
- an instance in a block other than by Decision 4's frozen list;
- a block run past a pair the scorer stopped, or folded into its verdict;
- any pilot, baseline or counted attempt while Stage 5 runs.

## Alternatives Considered

- **Native environments on macOS for each instance.** Rejected: old Python versions and compiled extensions (scikit-learn, astropy) per instance, and no guarantee the result matches the image the harness grades in.
- **Run the agent itself inside the container.** Rejected: Claude Code, `node`, the plugin and the account's credentials would all run under amd64 emulation, the plugin's hooks would no longer run where Stage 5's did, and every configuration's invocation would change.
- **Bind-mount the attempt's tree over `/testbed`.** Rejected: it hides the image's built extensions and editable-install metadata, which live in `/testbed`, so compiled repositories would fail on the unmodified tree.
- **Let the container keep its git history.** Rejected: if an image's object store holds the upstream fix, any `python -c` call through a shim could read it.
- **Scope the declared check from the dataset's PASS_TO_PASS tests.** Rejected: they name the files where the hidden tests live, which tells the agent where to look.
- **Baseline each block's candidates just before the block.** Rejected: ADR-112 D2 and D5 and the scorer need the surviving pool's digest and size before the first look.
- **Run the agent pilot before the baseline pilot.** Rejected: if ADR-112 D1's rule fails, the agent pilot's usage buys nothing.
- **Make an unapplied sync a setup exclusion.** Rejected: the attempt is still gradable, and its cause can differ by arm, so excluding it would bias the comparison.

## Component / Boundary Impact

None: the changes are internal to `bench` (`internal/bench`, `cmd/bench`). They add:
- the per-attempt container, the shims and the `stage6` profile;
- configuration W and the checks table;
- `--max-turns`, `bench baseline`, the surviving pool file and blocks drawn from it;
- receipt fields: `host_interpreter_calls`, `shim_calls`, `shim_wait_ms`, `sync_failures`, `write_backs` and `write_back_conflicts`.

The plugin, ADR-112 and quality-harness's scorer stay in that repository and are read, never changed. Docker is already a bench dependency (ADR-156 T5).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `bench run --profile stage6` | new profile: the per-attempt container, the shims, the declared check, W allowed | T1, T2 | T4, T5 |
| `bench canary` under `stage6` | proves the shims reach the container (`python -c 'import sys; print(sys.platform)'` prints `linux`), the container has no network and one commit, W's line is in W alone | T1, T2 | T4, T5 |
| receipt `host_interpreter_calls`, `shim_calls`, `shim_wait_ms`, `sync_failures`, `write_backs`, `write_back_conflicts` | new | T1 | T4, T5 |
| `bench shim` subcommand, `--docker` on `bench canary`, `bench canary --canary-image IMAGE` | new, delta of T1: the shims run `bench shim`; the canary needs an image for its fixture task | T1 | T4, T5 |
| receipt `write_back_failures`, `bare_python_calls`, `unlogged_python_calls`, `docker_calls` | new, delta of T1: beside the six keys above | T1 | T4, T5 |
| `bench/swe-bench/checks.json` | new: one whole runner per pool repository, with its source | T2 | T3, T4, T5 |
| configuration W; `run.json` `work_line_sha256`, `checks_sha256` | new | T2 | T4, T5 |
| `bench run --max-turns N`; `max_turns` in `run.json`, the rows and their `scaffold` | the stage's turn cap, default 60 | T2 | T4, T5, quality-harness's scorer |
| `bench baseline --stage S --out DIR --selection FILE --checks FILE` | new: per instance, the declared check on the unmodified tree, green or red, exit, wall time | T3 | T4, T5 |
| `bench/swe-bench/selection-adr-163.json` | new: the surviving pool, written by `bench baseline --surviving` | T3 | T5, quality-harness's pre-registration |
| `bench export-paired --pair A,W --selection selection-adr-163.json` | W tested; the surviving pool's digest and count; `checks_sha256`, `work_line_sha256` and `max_turns` bound per block and in `scaffold` | T3 | quality-harness's scorer |
| configurations G and C; `--qh-binary FILE --qh-sums FILE`; C's `turnEnd`; `side_admitted`, `side_configs_for` and `side` (asset sha, setting, tag) in `run.json` and the side receipts | new (Decision 9) | T6 | T4, T5 |
| `export-paired --pair A,P` / `--pair A,W` | `plugin` and `scaffold` built from the core arms only, byte-identical with or without side arms | T6 | quality-harness's scorer |
| `bench side --control P --arm G\|C` | new: the side reader, P as control, the registered bar | T6 | T5 |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the container, the shims, the lock, `--profile stage6`, the receipt's shim fields | T1 | T2, T3, T4, T5 | No — additive; without the profile nothing changes |
| configuration W, `checks.json`, the declared check, `--max-turns` | T2 | T3, T4, T5 | No — additive; `MaxTurns` defaults to 60 |
| `bench baseline`, the surviving pool file, its blocks, `--pair A,W` | T3 | T4, T5 | No — additive |
| configurations G and C, the staged binary, the setting, the core export projection, `bench side` | T6 | T4, T5 | No — additive; A, P and W rows unchanged |
| D1's answer, the surviving pool, D3's caps, the price | T4 | T5 | No |
| `docs/bench/stage-6/block-K/` and `looks/look-K.txt` | T5 | quality-harness ADR-112 T3 | No |

## Implementation

See `tasks/README.md`:
- T1: the instance's container and the shims;
- T2: configuration W, the declared check and the stage's caps;
- T3: the baseline, the surviving pool and the block plan;
- T4: the baseline pilot, the whole-pool baseline, then the agent pilot;
- T5: the stage, block by block, and its results.

## Consequences

- **Positive:**
  - the plugin is measured where a check can pass on the unmodified tree, so a red means the agent's change;
  - every arm, A included, can run tests, which is closer to how an agent works on a real project;
  - the baseline pilot can settle ADR-112 D1 before any usage is spent.
- **Negative:**
  - per-command sync, the lock and emulation make every test run slower;
  - a whole-pool baseline of 191 image pulls before the first counted attempt;
  - up to the surviving pool's block count of up to 240 attempts each, on the subscription;
  - one more moving part (a container per attempt) that can fail and exclude an attempt.
- **Neutral:** these rows are not comparable with Stage 5's, since A gains runnable tests too; each stage keeps its own verdicts.

## Out of Scope

- Configuration B in this stage (permanent: boundary: ADR-112 measures the plugin; mrw's own verdict is Stage 5's)
- Arms P+mrw and P with a batching cue, quality-harness ADR-107 T4 (deferred: docs/adr/BACKLOG.md — "From ADR-163")
- The scoped check form of ADR-112 D1 (external: quality-harness: the scoped check ADR-111 defers; this record is amended to use it)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Whole suites are too slow or too red under emulation | High | Med | the baseline pilot settles ADR-112 D1 before any usage; the stage then waits for the scoped form |
| An image's object store holds the upstream fix | Med | High | Decision 1 replaces `/testbed/.git` before any agent command; T1 S1 inspects one image; the canary checks it; a `docker` command stops the stage |
| A sync does not apply (generated files, line endings) | Med | Med | the command fails for the agent, counted per arm, never an exclusion; the pilot reports the count |
| Concurrent shim calls collide | Med | High | the per-container lock; waits counted per arm |
| A write-back conflicts with a host edit made while a shim ran | Low | Med | nothing is written for that file; the command fails naming it; counted per arm |
| A compiled extension goes stale when the agent edits its source | Med | Med | as in the harness's own grading image; the pilot reads which repositories rebuild |
| An agent reaches a host interpreter | Low | Med | counted per transcript; a non-zero pilot count hides them before block 1 |
| A container is left behind | Med | Low | removal on every path, interrupts included; T1's test drives the interrupt |
| Building alongside Stage 5 disturbs it | Med | Med | T1 to T3 are hermetic; anything that pulls or runs an image waits until Stage 5 closes |
| The price grows past what Zy approved | Med | High | the price is asked after the pilots, for the whole surviving pool |

## Rollback

None: this is measurement only. The profile, the shims, W, the flags and the subcommands can be removed. The stage's receipts, baselines, surviving pool and looks are data. Nothing on mrw's served path changes.

## Follow-ups

- [ ] quality-harness agrees D1 is read against 1200 s, and pre-registers the work line, the plugin tag, the scorer blob, D3's derivation rule and the surviving pool's digest before T4's agent pilot; then the caps' numbers after it, before block 1.
- [ ] Zy's word on the price, after T4.
- [ ] Re-cite ADR-162 T5's squash commit in place of 4eec8caa once it merges.
- [ ] G: quality-harness names the plugin tag whose documented setting selects the Go hook runtime, with `qh-darwin-arm64` in its `SHA256SUMS`, before T4's agent pilot. Without it, G is not in this stage.
- [ ] C: the tag and the exact spelling of the setting that keeps turn-end findings from the agent (`turnEnd: notify` through `c749638e`) from its release notes, written into the registration before T4's agent pilot.
