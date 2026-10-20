# Task ADR-163-T1: the instance's container and the shims

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** the per-attempt container started from the host's base, the `python`/`python3`/`pytest`/`py.test` shims with their lock, log and write-back, `bench run --profile stage6`, the receipt's `host_interpreter_calls`, `shim_calls`, `shim_wait_ms`, `sync_failures`, `write_backs` and `write_back_conflicts`, the canary's container checks
**Consumes:** none
**Data dependency:** hermetic tests against a fake `docker`; S1's inspection of one real SWE-bench image (no model, no usage) waits until Stage 5 closes
**Proof map:** v1
**Rests-on:** `tests run inside the instance's container`, `the container has no network, no history and no evaluation artifact`, `the container starts from the host's base`, `what Python writes through a shim reaches the host tree`, `one shim command at a time`, `a sync never touches the agent's index and never excludes`, `a shim maps the tree and nothing outside it`, `the container is removed on every path`, `the canary proves the container`, `without the profile nothing changes`

## Goal

Under `--profile stage6`, every configuration's `python`, `python3`, `pytest` and `py.test` run inside a container of the instance's own image, which has no network and no git history, one command at a time, with the agent's change synced in before each command (ADR-163 Decisions 1 and 6).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/container.go` | add | start the attempt's container (`--network none`, `--platform linux/amd64`), replace `/testbed/.git` with a one-commit repository, write the shims, the per-container lock and the shim log, sync the tree's diff through a temporary `GIT_INDEX_FILE`, remove the container on every path |
| `internal/bench/runner.go` | edit | `RunOptions.Profile` accepts `stage6`; `runAttempt` starts and removes the container; `toolPath` writes the shims in place of the host `python3` under the profile; the receipt's shim fields; the transcript counts of host interpreters, `docker` and unlogged Python commands |
| `internal/bench/profile.go` | edit | `ProfileStage6`, which keeps every part of `stage5` and adds the container |
| `internal/bench/canary.go` | edit | under `stage6`: `python -c 'import sys; print(sys.platform)'` prints `linux`, a network call fails, `git -C /testbed log --all --oneline` shows one commit |
| `internal/bench/container_test.go` | add | five of the tests below, against a fake `docker` that records its calls |
| `internal/bench/canary_test.go` | edit | `TestTheStage6CanaryProvesTheContainer` |
| `cmd/bench/main.go` | edit | `stage6` in `--profile`; the usage text |
| `docs/bench/README.md` | edit | the profile, the shims, what is counted and what stops the stage |

## Ordered Steps

1. [S1] Write the six failing tests and see them red. After Stage 5 closes (ADR-163 Decision 7), inspect one real SWE-bench image from `selection.json`, outside the pool, in a fresh container: `git log --all`, the reflog and the object count in `/testbed/.git`; every file outside `/testbed`'s tracked tree that could be an evaluation artifact (a test patch, an eval script, a list of test names), and a fixed rule that finds and removes each kind; how the `testbed` environment is activated; how long a container takes to start under emulation. [proof: human: the executor records the image, its history, each artifact found and the rule that removes it, the activation command and the start time]
2. [S2] `container.go`: start one container per attempt with no network; make `/testbed`'s tracked files exactly the host's prepared base commit, keeping ignored build output, remove S1's evaluation artifacts by their rule, and replace `/testbed/.git` with a fresh repository of that one commit, all before the agent starts; write the shims into the attempt's tools directory. Each shim takes the per-container lock, resets `/testbed` keeping ignored output (`git reset --hard`, `git clean -fd`), computes the tree's binary diff against bench's base commit plus its untracked files through a temporary `GIT_INDEX_FILE`, applies it, maps the working directory and path arguments from the tree to `/testbed`, runs the command in the `testbed` environment, writes back every file the command changed outside `.gitignore`, `__pycache__/`, `*.pyc` and `.pytest_cache/` into the host tree (a file the host changed since the sync is a conflict: not written, named, counted), logs the call and its wait, and returns its output and exit code. A working directory outside the tree exits 2, saying why. A diff that does not apply makes the shim exit non-zero with the reason on stderr, and counts in `sync_failures`; it never excludes the attempt. [proof: mutation]
3. [S3] `runAttempt` removes the container on a clean end, an error, the budget and an interrupt. The receipt counts commands naming an absolute host interpreter or runner, `env -i` or `command -p` before one, or `xcrun python3`; Python or test commands with no shim log entry; and commands naming `docker`, a pool image or the container, which T5 treats as a stop. [proof: mutation]
4. [S4] The canary's three container checks under `stage6`; `stage6` in `cmd/bench/main.go` and its usage text; `docs/bench/README.md`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestStage6ShimsRunInTheInstanceContainer|TestTheContainerHasNoNetworkAndNoHistory|TestAShimStartsFromTheHostBaseAndKeepsWhatPythonWrote|TestShimCallsTakeTurnsAndNeverTouchTheIndex|TestAShimRefusesAPathOutsideTheTreeAndCountsHostCalls|TestTheStage6CanaryProvesTheContainer)$' 2>&1) \
  && for t in TestStage6ShimsRunInTheInstanceContainer TestTheContainerHasNoNetworkAndNoHistory TestAShimStartsFromTheHostBaseAndKeepsWhatPythonWrote TestShimCallsTakeTurnsAndNeverTouchTheIndex TestAShimRefusesAPathOutsideTheTreeAndCountsHostCalls TestTheStage6CanaryProvesTheContainer; do grep -q "^--- PASS: $t " <<<"$out" || exit 1; done \
  && { go run ./cmd/bench 2>&1 || true; } | grep -q 'stage6'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheRunnerRefusesToOverwriteAnEarlierRunsReceipts` | `internal/bench/runner_test.go` | placeholder until this task's tests land: earlier stages' receipts survive (ADR-156 T2); the rows below replace it, with their Steps, in the commit that adds the tests | none | — |

Planned tests, which this table names in the commit that adds them:

- TestStage6ShimsRunInTheInstanceContainer in internal/bench/container_test.go: each shim calls `docker exec` on the attempt's container with the mapped working directory and arguments, after the reset and the diff; output and exit code pass through; an unapplied diff exits non-zero, counts in `sync_failures`, and the attempt is still graded; without `stage6` (and under `stage5`) `toolPath` links the host `python3` and no container starts (steps S2)
- TestTheContainerHasNoNetworkAndNoHistory in internal/bench/container_test.go: the container starts with `--network none`; `/testbed/.git` is replaced and a planted evaluation artifact is removed before the agent starts; the container is removed on a clean end, an error, the budget and an interrupt (steps S2, S3)
- TestAShimStartsFromTheHostBaseAndKeepsWhatPythonWrote in internal/bench/container_test.go: after start, the container's tracked tree equals the host's prepared base (the check file included); a shim command that writes a source file and a new test leaves both in the host tree and in its diff, `__pycache__` and `.pytest_cache` not; a file the host also changed is a named conflict, not written; an edit to the check file syncs (steps S2)
- TestShimCallsTakeTurnsAndNeverTouchTheIndex in internal/bench/container_test.go: two concurrent shim calls run one after the other, the second's wait logged; after a sync the agent's `git diff`, `git diff --cached` and index are unchanged (steps S2)
- TestAShimRefusesAPathOutsideTheTreeAndCountsHostCalls in internal/bench/container_test.go: a working directory outside the tree exits 2 naming it; commands naming `/usr/bin/python3`, `/opt/homebrew/bin/pytest`, `env -i python3`, `command -p python3`, `xcrun python3` count in `host_interpreter_calls`; a `pytest` with no shim log entry counts; a command naming `docker` is counted for T5's stop (steps S3)
- TestTheStage6CanaryProvesTheContainer in internal/bench/canary_test.go: a stage6 canary passes only when `python` prints `linux`, the network call fails and `/testbed` shows one commit; each failing alone fails the canary (steps S4)

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--profile stage6` in `cmd/bench/main.go`; the fence runs `go run ./cmd/bench` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's pilots and T5's stage |

## Mutation Log
(empty until execute)

## Invariants

- Without `--profile stage6`, every configuration's arguments, PATH and tools are ADR-162's.
- Nothing in the container can reach the network, the dataset or the repository's history.
- A container never outlives its attempt, and never runs two commands at once.
- No image is pulled or run while a Stage 5 block runs.

## Risks

- An image may lack `git`, or activate its environment differently; S1 settles both before the pilots.
- Emulated container starts may be slow enough to matter against the budget; S1 measures it.

## Stop Condition

The fence exits 0 and S1 is recorded; or S1 shows the history cannot be removed, an evaluation artifact that no fixed rule finds, or an environment `docker exec` cannot reach, any of which goes back to Zy.

## Out of Scope

- Hiding the host interpreters from the attempt (deferred: docs/adr/BACKLOG.md — "From ADR-163", decided at T4's agent pilot)

## Verification Log
(empty until execute)
