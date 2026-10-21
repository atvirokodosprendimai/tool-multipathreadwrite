# Task ADR-163-T1: the instance's container and the shims

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** the per-attempt container started from the host's base, the `python`/`python3`/`pytest`/`py.test` shims with their lock, log and write-back, `bench run --profile stage6`, the receipt's `host_interpreter_calls`, `shim_calls`, `shim_wait_ms`, `sync_failures`, `write_backs` and `write_back_conflicts`, the canary's container checks; the receipt keys `write_back_failures`, `bare_python_calls`, `unlogged_python_calls` and `docker_calls` beside those; the `bench shim` subcommand; `--docker` and `--canary-image` on `bench run` and `bench canary`; `internal/bench/container_unix.go`, `container_windows.go`; the forwarding of the variables a command set; the orphan kill; the label, lifetime and sweep of the containers; the `settled` constant
**Consumes:** none
**Data dependency:** hermetic tests against a fake `docker`; S1's inspection of one real SWE-bench image (no model, no usage) waits until Stage 5 closes
**Proof map:** v1
**Rests-on:** `tests run inside the instance's container`, `the container has no network, no history and no evaluation artifact`, `the container starts from the host's base`, `what Python writes through a shim reaches the host tree`, `one shim command at a time`, `a sync never touches the agent's index and never excludes`, `a shim maps the tree and nothing outside it`, `the container is removed on every path`, `the canary proves the container`, `without the profile nothing changes`, `a shim leaves no command running in the container`, `a shim forwards the variables the command set`, `the container start is checked, labelled and swept`, `a failed write-back is counted apart`, `bench run refuses the unsettled profile`, `unlogged python calls are matched to the shim log`, `the shim log notes what it could not write`

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
| `cmd/bench/main.go` | edit | `stage6` in `--profile`; the usage text; the `bench shim` subcommand the shims run; `--docker` and `--canary-image` in the flags `run` and `canary` share |
| `docs/bench/README.md` | edit | the profile, the shims, what is counted and what stops the stage |
| `internal/bench/container_unix.go`, `internal/bench/container_windows.go` | add | the flock behind the per-container lock; Windows has none and `resolveContainer` refuses stage6 there |
| `internal/bench/transcript.go` | edit | the Bash commands of a transcript, which the receipt counts |

## Ordered Steps

1. [S1] Write the six failing tests and see them red. After Stage 5 closes (ADR-163 Decision 7), inspect one real SWE-bench image from `selection.json`, outside the pool, in a fresh container: `git log --all`, the reflog and the object count in `/testbed/.git`; every file outside `/testbed`'s tracked tree that could be an evaluation artifact (a test patch, an eval script, a list of test names), and a fixed rule that finds and removes each kind; how the `testbed` environment is activated; how long a container takes to start under emulation, whether the image has ps/procps and git, and whether runc setsids the processes docker exec starts. [proof: human: the executor records the image, its history, each artifact found and the rule that removes it, the activation command and the start time]
2. [S2] `container.go`: start one container per attempt with no network; make `/testbed`'s tracked files exactly the host's prepared base commit, keeping ignored build output, remove S1's evaluation artifacts by their rule, and replace `/testbed/.git` with a fresh repository of that one commit, all before the agent starts; write the shims into the attempt's tools directory. Each shim takes the per-container lock, resets `/testbed` keeping ignored output (`git reset --hard`, `git clean -fd`), computes the tree's binary diff against bench's base commit plus its untracked files through a temporary `GIT_INDEX_FILE`, applies it, maps the working directory and path arguments from the tree to `/testbed`, runs the command in the `testbed` environment, writes back every file the command changed outside `.gitignore`, `__pycache__/`, `*.pyc` and `.pytest_cache/` into the host tree (a file the host changed since the sync is a conflict: not written, named, counted), logs the call and its wait, and returns its output and exit code. A working directory outside the tree exits 2, saying why. A diff that does not apply makes the shim exit non-zero with the reason on stderr, and counts in `sync_failures`; it never excludes the attempt. [proof: mutation]
3. [S3] `runAttempt` removes the container on a clean end, an error, the budget and an interrupt. The receipt counts commands naming an absolute host interpreter or runner, `env -i` or `command -p` before one, or `xcrun python3`; Python or test commands with no shim log entry; and commands naming `docker`, a pool image or the container, which T5 treats as a stop. [proof: mutation]
4. [S4] The canary's three container checks under `stage6`; `stage6` in `cmd/bench/main.go` and its usage text; `docs/bench/README.md`. [proof: mutation]

S1's commit flips `settled` in `internal/bench/container.go` (const, false). While it is false `gates` refuses `bench run --profile stage6` and still lets `bench canary` run, which S1's inspection uses. The commit also fills `evalArtifacts` and corrects `activateTestbed` from what the image shows.

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
| `TestStage6ShimsRunInTheInstanceContainer` | `internal/bench/container_test.go` | each shim calls `docker exec` on the attempt container with the mapped working directory and arguments, after the reset and the diff; output and exit code pass through; an unapplied diff exits non-zero, counts in `sync_failures`, and the attempt is still graded; without `stage6` (and under `stage5`) `toolPath` links the host `python3` and no container starts | none | S2 |
| `TestTheContainerHasNoNetworkAndNoHistory` | `internal/bench/container_test.go` | the container starts with `--network none`; `/testbed/.git` is replaced and a planted evaluation artifact is removed before the agent starts; the container is removed on a clean end, an error, the budget and an interrupt | none | S2, S3 |
| `TestAShimStartsFromTheHostBaseAndKeepsWhatPythonWrote` | `internal/bench/container_test.go` | after start, the container tracked tree equals the host prepared base (the check file included); a shim command that writes a source file and a new test leaves both in the host tree and in its diff, `__pycache__` and `.pytest_cache` not; a file the host also changed is a named conflict, not written; an edit to the check file syncs | none | S2 |
| `TestShimCallsTakeTurnsAndNeverTouchTheIndex` | `internal/bench/container_test.go` | two concurrent shim calls run one after the other, the second wait logged; after a sync the agent `git diff`, `git diff --cached` and index are unchanged | none | S2 |
| `TestAShimRefusesAPathOutsideTheTreeAndCountsHostCalls` | `internal/bench/container_test.go` | a working directory outside the tree exits 2 naming it; commands naming `/usr/bin/python3`, `/opt/homebrew/bin/pytest`, `env -i python3`, `command -p python3`, `xcrun python3` count in `host_interpreter_calls`; a `pytest` with no shim log entry counts; a command naming `docker` is counted for T5 stop | none | S3 |
| `TestTheStage6CanaryProvesTheContainer` | `internal/bench/canary_test.go` | a stage6 canary passes only when `python` prints `linux`, the network call fails and `/testbed` shows one commit; each failing alone fails the canary | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--profile stage6` in `cmd/bench/main.go`; the fence runs `go run ./cmd/bench` |
| 3 — the caller can discover it | the usage text and `docs/bench/README.md` |
| 4 — it is used | T4's pilots and T5's stage |

## Mutation Log
(empty until execute)
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the container starts with a network · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the container has no network, no history and no evaluation artifact
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · two shim calls share the container at once · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:one shim command at a time
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a write-back carries .pytest_cache into the host tree · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:what Python writes through a shim reaches the host tree
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a file the tree changed meanwhile is overwritten by the write-back · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:what Python writes through a shim reaches the host tree
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the sync builds its tree in the index of the agent · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a sync never touches the agent's index and never excludes
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/runner.go` · the container is left behind on every way out of an attempt · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the container is removed on every path
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · xcrun python3 escapes the host interpreter count · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/canary.go` · a container whose network answers passes the canary · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the canary proves the container
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/canary.go` · the probe never asks the container about its network · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the canary proves the container
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · an evaluation artifact the table names stays in the container · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the container has no network, no history and no evaluation artifact
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the command runs in the root of the testbed, not the mapped directory · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:tests run inside the instance's container
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the base of the host is never extracted into the container · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the container starts from the host's base
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/runner.go` · the host python3 is not linked when the profile is off · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:without the profile nothing changes
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/runner.go` · a stage without the profile starts a container · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:without the profile nothing changes
- 2026-10-21 · 6715a44c* · mutant survived · exit 0 · `internal/bench/container.go` · a command left in the container is never killed · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim leaves no command running in the container
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a shim told to stop leaves its command running in the container · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim leaves no command running in the container
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the next sync does not kill the command an earlier shim left · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim leaves no command running in the container
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the short spelling of the tree is replaced first · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 6715a44c* · mutant survived · exit 0 · `internal/bench/container.go` · a bare call matches any log entry of its program · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:unlogged python calls are matched to the shim log
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a bare call with other arguments consumes the log entry of its program · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:unlogged python calls are matched to the shim log
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/canary.go` · the canary asks from a directory outside the tree, so the shim refuses · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the canary proves the container
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the sweep finds containers and removes none · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the container start is checked, labelled and swept
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a container whose tree is not the host base tree is accepted · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the container start is checked, labelled and swept
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the container does not remove itself · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the container start is checked, labelled and swept
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a failed write-back is counted as a failed sync · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a failed write-back is counted apart
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a variable reaches the container with the host path of the tree · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim forwards the variables the command set
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · PATH, HOME and TMPDIR are forwarded to the container · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim forwards the variables the command set
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/runner.go` · bench run accepts the unsettled profile · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:bench run refuses the unsettled profile
- 2026-10-21 · 6715a44c* · mutant survived · exit 0 · `internal/bench/container.go` · a log line that cannot be written leaves no note · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the shim log notes what it could not write
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a log line that cannot be written leaves no note beside the log · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:the shim log notes what it could not write
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · the shims in the tools directory of the attempt count as host interpreters · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · a variable the command did not change is forwarded · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim forwards the variables the command set
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · group kill removed: a grandchild of the exec leader survives (a shim leaves no command running in the container) · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · fd-redirect digit kept as an arg: 2>&1 call counted unlogged (unlogged python calls are matched to the shim log) · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 6715a44c* · mutant killed · exit 1 · `internal/bench/container.go` · sweep loses its stage value: another stage container is removed (the container start is checked, labelled and swept) · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · sync shell records no pid: a shim killed mid-sync leaves its shell for the next shim to race · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · env PATH=… python3 not counted: host python runs unseen · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · tree path mapped without a boundary: tree-backup is rewritten · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `cmd/bench/main.go` · the built bench loses its shim dispatcher · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · post snapshot drops the sync paths: a file that becomes ignored is deleted on the host · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · a container shell reaps the pid file naming itself · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim leaves no command running in the container
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · a path that merely ends in the tree path is mapped · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · sync shell registers after its cleanup: a shim killed during cleanup leaves an unregistered shell · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · one fixed pid.sync name: a new shell overwrites an orphan record before reaping it · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d
- 2026-10-21 · 73230f4a* · mutant survived · exit 0 · `internal/bench/container.go` · the write-back shell unregisters before its diff · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim leaves no command running in the container
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · the write-back shell unregisters before its diff · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim leaves no command running in the container
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · env is taken as a command wherever it stands in the line · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 73230f4a* · mutant killed · exit 1 · `internal/bench/container.go` · text in quotes is read as commands · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 52c2f9f3* · mutant killed · exit 1 · `internal/bench/container.go` · a command substitution inside double quotes is not a command position · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 52c2f9f3* · mutant killed · exit 1 · `internal/bench/container.go` · a substitution or backtick does not open a command position · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it
- 2026-10-21 · 52c2f9f3* · mutant killed · exit 1 · `internal/bench/container.go` · a quoted assignment value hides the command that follows · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · covers:a shim maps the tree and nothing outside it

## Invariants

- Without `--profile stage6`, every configuration's arguments, PATH and tools are ADR-162's.
- Nothing in the container can reach the network, the dataset or the repository's history.
- A container never outlives its attempt, and never runs two commands at once.
- No image is pulled or run while a Stage 5 block runs.

## Risks

- An image may lack `git`, or activate its environment differently; S1 settles both before the pilots.
- Emulated container starts may be slow enough to matter against the budget; S1 measures it.
- `host_interpreter_calls` is a heuristic over the text of the transcript and not a sandbox: the counter reads shell text without a shell, so a command it cannot parse can hide a bypass, and a heredoc body is read as commands.

## Stop Condition

The fence exits 0 and S1 is recorded; or S1 shows the history cannot be removed, an evaluation artifact that no fixed rule finds, or an environment `docker exec` cannot reach, any of which goes back to Zy.

## Out of Scope

- Hiding the host interpreters from the attempt (deferred: docs/adr/BACKLOG.md — "From ADR-163", decided at T4's agent pilot)
- The counter reads shell text heuristically, and a heredoc body is counted as commands (permanent: boundary: the counter reads shell text heuristically; a heredoc body is overcounted, which fails safe and shows in the receipt)

## Verification Log
(empty until execute)
- 2026-10-21 · 6715a44c* · exit 1 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:246 · steps:S1 · test-lock-sha256:31995dc41e205cb839b3442d4ed7e90c1fdbca62b98c427f609bef696fd3a172 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2FuYXJ5X3Rlc3QuZ28JVGVzdFRoZUNhbmFyeUZhaWxzV2hlblRoZVRyYW5zY3JpcHROYW1lc01yd09yVXNlckluc3RydWN0aW9ucwkwN2EzOTVkNDI3ODFjMmY0YjQ5YTY5ZTJiMGJkNTdlNjhkMzRlZDdmNzMxMTYxYzZjMTEwYWUxMzY2ZTA3NzM2CmJvZHkJaW50ZXJuYWwvYmVuY2gvY2FuYXJ5X3Rlc3QuZ28JVGVzdFRoZUNhbmFyeVJlZnVzZXNBUGx1Z2luVGhhdElzTm90RW5nYWdlZAk1Y2NkZWFjODc3YTZhODdkMWZmZDg5OWE4M2Y3NzM2ZWIxNGNjYzM2OTM5NTRiMjE3NmM0MjRlZThkZDU3NjViCmJvZHkJaW50ZXJuYWwvYmVuY2gvY2FuYXJ5X3Rlc3QuZ28JVGVzdFRoZUNvbmZpZ0RpcmVjdG9yeUlzTm90QUxlYWsJMWRiNDdkNGY1ZWEzYTI5ZTA3NDVlZDBmNzUyMzIzZGQ4NzhkZmE4NjMwNDA0YjRjMGIxMjU4MTI2YjZkNDYzZApib2R5CWludGVybmFsL2JlbmNoL2NhbmFyeV90ZXN0LmdvCVRlc3RUaGVTdGFnZTZDYW5hcnlQcm92ZXNUaGVDb250YWluZXIJZmQ1ZDMyNjU4YmRjZDExMTE2ZTdiNmQ1YzQ2NjhiYjAxYmNhYjZlMTgzMmU1NTRkOWE3MWYzNTUyZmIwYjQyNgpib2R5CWludGVybmFsL2JlbmNoL2NvbnRhaW5lcl90ZXN0LmdvCVRlc3RBU2hpbVJlZnVzZXNBUGF0aE91dHNpZGVUaGVUcmVlQW5kQ291bnRzSG9zdENhbGxzCWRkZjMwODIzYjlkZGI2N2NmMGQ0NWIxYWM4ZTAzYTA3ZmUwZjlkZDIxMjk5YWY2MzVjNjRmYWM5OWE1MDk1NWIKYm9keQlpbnRlcm5hbC9iZW5jaC9jb250YWluZXJfdGVzdC5nbwlUZXN0QVNoaW1TdGFydHNGcm9tVGhlSG9zdEJhc2VBbmRLZWVwc1doYXRQeXRob25Xcm90ZQk0YzBmMzhiNTQyODc2MDNmYjU2OTg5NmEzZTIyOTAwMzQzNWY0NTBhYWUwM2E1ZjgwODRhYWUzOTZhMjkwNTIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvY29udGFpbmVyX3Rlc3QuZ28JVGVzdE1haW4JMTk5ODQwOTJjYzc0MTg0MWUyNTE5ZWU1OGUyNmJlNjg1ZjM4ZWE1Nzg3ZDk3NDBjN2U1ODg5Y2YwYzU5ZGRhNgpib2R5CWludGVybmFsL2JlbmNoL2NvbnRhaW5lcl90ZXN0LmdvCVRlc3RTaGltQ2FsbHNUYWtlVHVybnNBbmROZXZlclRvdWNoVGhlSW5kZXgJZjA0OTVjNzE4OTAyZDg2MTNhYjQ0NjcyNzQ4MTk4ODliOGVmNTdkMTFlYmU1ZmRjMWI5OTMxZGRjNjE5MmNiMQpib2R5CWludGVybmFsL2JlbmNoL2NvbnRhaW5lcl90ZXN0LmdvCVRlc3RTdGFnZTZTaGltc1J1bkluVGhlSW5zdGFuY2VDb250YWluZXIJMDdiOTc3MWE3NGIzNjcyYTdkMzIxYjM4Y2QyN2U5YzhhZjNiYzcyZGZhZjVhNzMwYjRiMTgzYmI5YTAxMDY4Mgpib2R5CWludGVybmFsL2JlbmNoL2NvbnRhaW5lcl90ZXN0LmdvCVRlc3RUaGVDb250YWluZXJIYXNOb05ldHdvcmtBbmROb0hpc3RvcnkJMGMyMDk3ZjIwNmNjNTAwY2I4ODcyOWEwMTlmOGRjNTFlMjZkZDU3NzBlNzhiMGI0MGEyMTZjYTVlZjg5OWYwZg
  ```
  ```
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:17372 · steps:S2,S3,S4
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:17645
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:18033
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:19440
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:21174
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:21493
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:17794
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:18457
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:17693
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:30307
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:18332
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:21413
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:18549
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:34162
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:19385
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:28673
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:53261
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:25551
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:26089
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:45652
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:28488
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:24744
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:25488
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:26129
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:24392
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:38912
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:132808
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:61790
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:44046
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:29485
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:67506
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:46454
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:26487
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:33870
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:32798
- 2026-10-21 · 6715a44c* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:32858
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:37534
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:41751
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:39454
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:32164
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:45705
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:138552
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:50439
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:114982
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:60299
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:69748
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:37555
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:46813
- 2026-10-21 · 73230f4a* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:81755
- 2026-10-21 · 52c2f9f3* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:41157
- 2026-10-21 · 52c2f9f3* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:38110
- 2026-10-21 · 52c2f9f3* · exit 0 · `set -o pipefail …` · acceptance-sha256:b5e9d851509447352957f53af14611f5e2ed14ae8edf72c8201079c76341357d · ms:38105
