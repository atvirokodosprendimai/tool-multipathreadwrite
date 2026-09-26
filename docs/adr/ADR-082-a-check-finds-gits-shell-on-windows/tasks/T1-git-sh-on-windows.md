# Task ADR-082-T1: a check finds Git's shell on Windows

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `check.Shell`, `gitShell`, the no-shell advice
**Consumes:** `check.Run`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `Git's shell is found beside git.exe`, `no shell says what to install`, `the check runs under the shell found`, `a test skips only without a shell`, `the packages vet for Windows`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

Under plain PowerShell, 13 tests failed and no check could start: `sh` is not on that PATH, though Git for Windows' own `sh.exe` is on disk.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/shell.go` | new | `Shell`, `gitShell`, `noShellAdvice` |
| `internal/check/check.go` | edit | `Run` starts the check with `Shell()`; the no-shell advice |
| `internal/check/shell082_test.go`, `internal/check/shell082_windows_test.go` | new | the tests below |
| `cmd/mrw/shell082_test.go`, `internal/adversarial/shell082_test.go` | new | `needShell` for each test package |
| the 13 tests that start a check | edit | `needShell(t)` first |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `gitShell` looks one directory only; `Shell` never falls back; the advice dropped.
3. [S3] The Windows CI shard runs `TestACheckRunsUnderGitsShellWhenShIsNotOnPath`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/check/ -count=1 -timeout 120s -run 'TestGitsShellIsFoundBesideGit|TestACheckWithNoShellSaysSo' -v 2>&1 | tee /tmp/adr082-T1.out \
  && missing=$(for t in TestGitsShellIsFoundBesideGit TestACheckWithNoShellSaysSo; do grep -qE "^--- PASS: $t \(" /tmp/adr082-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^func TestACheckRunsUnderGitsShellWhenShIsNotOnPath(' internal/check/shell082_windows_test.go \
  && [ "$(grep -rl 'needShell(t)' cmd/mrw internal/adversarial internal/check | wc -l | tr -d ' ')" -ge 7 ] \
  && GOOS=windows go vet ./internal/check/ ./cmd/mrw/ ./internal/adversarial/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/read internal/rooted internal/lines internal/iter internal/seen internal/state internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/read internal/rooted internal/lines internal/iter internal/seen internal/state internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestGitsShellIsFoundBesideGit` | `internal/check/shell082_test.go` | from `cmd\git.exe`, `mingw64\bin\git.exe` and `bin\git.exe` the shell is found; none, or a directory named `sh.exe`, yields nothing | — | S1, S2 |
| `TestACheckWithNoShellSaysSo` | `internal/check/shell082_test.go` | with no shell on PATH the check could not start and says "no sh on PATH" | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `check.Shell` |
| 2 — something selects it | every check `mrw write` and `mrw check` run |
| 3 — the caller can discover it | the check runs; without a shell the report names what to install |
| 4 — it is used | a Windows peer's three PowerShell runs of v1.27.0 hit the gap |

## Verification Log
(empty until execute)
- 2026-09-26 · 31fe531* · exit 1 · `set -o pipefail …` · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · ms:323 · test-lock-sha256:826daa4a488e7af77bac77acc5f0150f0ca0de1d42649d6efc4b3cf693012852 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL3NoZWxsMDgyX3Rlc3QuZ28JVGVzdEFDaGVja1dpdGhOb1NoZWxsU2F5c1NvCWU5Y2Q0YzQ4YTY3NjViYTIxMjExNGVhM2NlM2ZhZDk0NTM5ZDQ3NWU3YjgzZjg0NTI5NDAxMmQzY2I3OTE3MDAKYm9keQlpbnRlcm5hbC9jaGVjay9zaGVsbDA4Ml90ZXN0LmdvCVRlc3RHaXRzU2hlbGxJc0ZvdW5kQmVzaWRlR2l0CTM0MTM4NWZkNzFkZmQ2YjY5NjBhOTExODU5YWYyY2YzNjNmYTYyNGU0ZTM1ZDY5Y2MzNjJmZDRmNzViYzA5NTQ
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check.test]
  internal/check/shell082_test.go:31:19: undefined: gitShell
  internal/check/shell082_test.go:43:12: undefined: gitShell
  internal/check/shell082_test.go:49:12: undefined: gitShell
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check [build failed]
  FAIL
  ```
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · ms:1226
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · ms:677
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · ms:629
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · ms:352

## Mutation Log
(empty until execute)
- 2026-09-26 · 31fe531* · mutant killed · exit 1 · `internal/check/shell.go` · gitShell looks one directory only · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · covers:Git's shell is found beside git.exe
- 2026-09-26 · 31fe531* · mutant killed · exit 1 · `internal/check/shell.go` · a directory named sh.exe is taken for a shell · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · covers:Git's shell is found beside git.exe
- 2026-09-26 · 31fe531* · mutant killed · exit 1 · `internal/check/check.go` · the no-shell advice dropped · acceptance-sha256:12fb01e62e1ab20c59550644905727e20520660295e620341cb45623655c212e · covers:no shell says what to install

## Invariants

- A check that can run under some POSIX shell on the machine does run; one that cannot says why and what to install.

## Risks

- The end-to-end proof (`TestACheckRunsUnderGitsShellWhenShIsNotOnPath`) runs only on the Windows CI shard; the local fence proves the walk, the no-shell report, the Windows compile and that the tests skip through `needShell`.

## Out of Scope

- A contract row (permanent: fact: `scripts/contract.sh` runs on Linux only; the Windows CI shard runs the test)

## Stop Condition

The fence exits 0, the latest Mutation Log row for each mutant reads `killed`, and the Windows CI shard passes.
