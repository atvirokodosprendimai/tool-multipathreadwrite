# Task ADR-099-T1: `mrw check --full PATH` is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the `--full` PATH refusal in `checkCmd`; contract §191
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the refusal runs nothing`, `full alone still runs`, `a contract row drives the binary`, `no engine package changes`

## Goal

`mrw check --full x.go` exits 2 with `--full runs the whole project; it takes no PATH` on stderr, runs no
check and prints nothing on stdout; `mrw check --full` alone runs the whole-project check as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `checkCmd` refuses `--full` with positionals right after reading them (`:2000`); the `--full` usage (`:1991`) says it takes no PATH |
| `cmd/mrw/path099_test.go` | add | the test below |
| `scripts/contract.sh` | edit | §191 |

**What selects it:** `checkCmd`'s Action is the only reader of `--full`; the refusal sits before the
working-set load and before `check.Load`, so nothing runs.

## Ordered Steps

1. [S1] Write `TestACheckWithFullAndAPathIsRefused` and confirm it RED on `7ebdf51`. [proof: mutation]
2. [S2] Add the refusal and the usage words. [proof: mutation] Mutants: the refusal removed; the
   refusal moved after the check runs (a marker the check writes then exists).
3. [S3] Contract §191, driving `$MRW`: `check --full x.go` exits 2, names the fix, and the declared check
   (which touches a marker) did not run; paired, `check --full` runs it.
   [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §191's rows printed — the fence greps the section, since the whole contract takes minutes]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestACheckWithFullAndAPathIsRefused' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestACheckWithFullAndAPathIsRefused \(' "$out" \
  && grep -q '^# 191\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal | grep '^??')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACheckWithFullAndAPathIsRefused` | `cmd/mrw/path099_test.go` | in a tree whose declared check touches a marker: through `runSplit`, `check --full x.go` returns exactly `--full runs the whole project; it takes no PATH` at exit 2 with empty stdout and no marker; `check --full` alone exits 0 and the marker exists | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the refusal in `checkCmd` |
| 2 — something selects it | `checkCmd`'s Action; the test runs `rootCommand`, §191 the built binary |
| 3 — the caller can discover it | the refusal names the fix; `--full`'s usage says it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-29 audit |

## Mutation Log
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refusal removed: --full drops the PATH and runs · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · covers:the refusal runs nothing
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `cmd/mrw/main.go` · full alone: a changed whole-project path still runs (equivalent guard for the pair) · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · covers:full alone still runs
- 2026-09-29 · 7ebdf51* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package changed against the merge-base · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · covers:no engine package changes
- 2026-09-29 · 29a29ef* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refusal also printed to stdout · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · covers:the refusal runs nothing

## Invariants

- `check --full` alone is unchanged; every other `check` form is unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if an existing test must change to pass.

## Out of Scope

- The help subcommand — T2.

## Verification Log
- 2026-09-29 · 7ebdf51* · exit 1 · `set -o pipefail …` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:791 · test-lock-sha256:f8421fcda4f4bc2041f1ac07b09524257341f74ca2856e270610ae51acab0475 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGF0aDA5OV90ZXN0LmdvCVRlc3RBQ2hlY2tXaXRoRnVsbEFuZEFQYXRoSXNSZWZ1c2VkCTc1OWQ1OTA0NWViMWU1M2UwNTNhZTdhMjJhNGM5ZjYzNjE0MDI5Yjc3ODAxMGYyMWZjZjc0MjYzZjdiYTRhMTQKYm9keQljbWQvbXJ3L3BhdGgwOTlfdGVzdC5nbwlUZXN0QUZpbGVOYW1lZEhlbHBJc0FQYXRoCWVjMzgzNDY1ZTE0MWU3NWIxOTQwNTE5YmNlZGZlNjk5ZmJlMzIwYmZkZjA1YzIxZDM1NDNmNTI2Zjg4YzUxNTI
  ```
  --- last 9 line(s) of stdout
  === RUN   TestACheckWithFullAndAPathIsRefused
      path099_test.go:34: check --full x.go: exit 0, want 2 naming the fix:
          check (declared): touch marker099
          check PASS (exit 0, 6ms)
      path099_test.go:37: check --full x.go ran the check: the refusal must come before anything runs
  --- FAIL: TestACheckWithFullAndAPathIsRefused (0.02s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.225s
  FAIL
  ```
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:489
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:354
- 2026-09-29 · 7ebdf51* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:345
- 2026-09-29 · human-observed · correction to the paths[:0:0] mutant row above: its why calls it an equivalent guard, but it was KILLED — an empty non-nil scope is not the whole project, so the test tells them apart; the mutant stands as evidence for 'full alone still runs'
- 2026-09-29 · human-observed · S3 observed 2026-09-29: ./scripts/contract.sh run unpiped on this branch (T1 and T2 applied, base 7ebdf51), exit 0 'contract holds', with §191 printed: check --full a.go exits 2 naming the fix with no check run; check --full alone runs it
- 2026-09-29 · human-observed · relock 2026-09-29: after the red run, TestAFileNamedHelpIsAPath's write case changed from 'the plan in the file help applied' to 'help reached write as the plan path (open help, no USAGE)', because a plan file resolves from the working directory, not --root, so the in-process run cannot open it; still red without the fix (write help printed USAGE), every other assertion kept
- 2026-09-29 · 7ebdf51* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:0 · test-lock-sha256:29e03e1a20aaf30934d42e04777b60dc8402e367e9c3fa1805c8b9be01ef802b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGF0aDA5OV90ZXN0LmdvCVRlc3RBQ2hlY2tXaXRoRnVsbEFuZEFQYXRoSXNSZWZ1c2VkCTc1OWQ1OTA0NWViMWU1M2UwNTNhZTdhMjJhNGM5ZjYzNjE0MDI5Yjc3ODAxMGYyMWZjZjc0MjYzZjdiYTRhMTQKYm9keQljbWQvbXJ3L3BhdGgwOTlfdGVzdC5nbwlUZXN0QUZpbGVOYW1lZEhlbHBJc0FQYXRoCTE0ZjMyZDI3NzBhNzU3ZjgwMDdkZGQ0OTdlMDM2MjAyNzRkNjkxZWQ5NzNkZGVkODE4M2FlMDA2OTRkNTljNmY · test-lock-kind:replace
- 2026-09-29 · human-observed · relock 2026-09-29 (Codex review of #285): TestACheckWithFullAndAPathIsRefused now asserts the exact refusal and empty stdout through runSplit; TestAFileNamedHelpIsAPath now runs write from the checkout (t.Chdir) and asserts the plan applied, and asserts check help exits 0 with its scoped check receiving help as {files}. Both stronger; nothing removed
- 2026-09-29 · 29a29ef* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:0 · test-lock-sha256:c0172a75b813c499bb635d5cb0ab06833ec5f5cce4910c702932b809de584979 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGF0aDA5OV90ZXN0LmdvCVRlc3RBQ2hlY2tXaXRoRnVsbEFuZEFQYXRoSXNSZWZ1c2VkCWRhMDZhYzcyODQ4ZTNmMmQ4YmNiZjM4YTEzYWFhOWU0NjExMTk2ZDU1ZmNhMDYxNmNmZGE4ODZhNTJkOTUwYzcKYm9keQljbWQvbXJ3L3BhdGgwOTlfdGVzdC5nbwlUZXN0QUZpbGVOYW1lZEhlbHBJc0FQYXRoCTIxMjVmZmUwYzBkZGYxYjRhZWE2MWJjMDI4MjllZDk5Y2EwZGNlODJiODk3Y2VjM2Q4NDRlNGFkY2ZiOWY3MWQ · test-lock-kind:replace
- 2026-09-29 · 29a29ef* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:330
- 2026-09-30 · human-observed · correction to the correction above (Codex review of 53e77b4, 2026-09-30): the paths[:0:0] mutant was killed by the fence's gofmt clause, because its '; _ = 0' suffix is not gofmt layout, and not by any test; check.command treats a nil and an empty scope alike (internal/check/check.go:540-545), so the mutant changed no behaviour and the row is no evidence for 'full alone still runs'. That clause is carried by TestACheckWithFullAndAPathIsRefused's full-alone case and contract §191. The paths = nil block it mutated was unreachable in effect after the refusal (paths is already empty under --full) and is deleted
- 2026-09-30 · ae58bf6* · exit 0 · `set -o pipefail …` · acceptance-sha256:9f536a515d3cad6b5244c70a9551837f39cd37f7a910d67cb9adf49e830c69b7 · ms:399
