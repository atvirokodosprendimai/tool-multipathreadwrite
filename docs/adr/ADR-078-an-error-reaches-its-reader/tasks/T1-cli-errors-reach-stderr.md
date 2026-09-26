# Task ADR-078-T1: the CLI's errors reach stderr and say what to do

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `installUsageErrors`, `usageError`, `hasSinglePattern`, `cmdExeForm`; the plan parser's broken-header state
**Consumes:** `cli.Exit`, `exitUsage`, `plan.Parse`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a usage error writes nothing to stdout`, `mcp takes no arguments`, `-C needs a pattern`, `a bad header is one error`, `the tab is named`, `cmd.exe gets its form`, `the POSIX fix quotes an apostrophe`, `a contract row drives the binary`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A usage error printed help to stdout, `mrw mcp` took stray arguments, `-C` with no pattern was ignored, a bad header's body was reported line by line, a tab header read as prose, and cmd.exe could not use the padded-path fix.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `installUsageErrors`, `usageError`, the `mcp` argument check, `hasSinglePattern`, `cmdExeForm` |
| `internal/plan/plan.go` | edit | the broken-header state; the tab |
| `internal/guide/guide.go`, `AGENTS.md` | edit | the cmd.exe form |
| `cmd/mrw/usage078_test.go`, `cmd/mrw/cmdquote_windows_test.go`, `internal/plan/parse078_test.go` | new | the tests below |
| `scripts/contract.sh` | edit | §155, §159 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: OnUsageError installed on the root only; the `mcp` argument check removed; `hasSinglePattern` returns true; `broken` never set; the tab branch removed.
3. [S3] Contract §155 and §159; the guide and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ ./internal/plan/ ./internal/guide/ -count=1 -timeout 240s -run 'TestAUsageErrorWritesNothingToStdout|TestMcpTakesNoArguments|TestContextWithNoPatternIsRefused|TestABadHeaderIsOneErrorNotOnePerBodyLine|TestATabAfterTheAtSignsIsNamed|TestThePaddedPathFixNamesTheCmdExeForm|TestThePaddedPathFixQuotesAnApostrophe' -v 2>&1 | tee /tmp/adr078-T1.out \
  && missing=$(for t in TestAUsageErrorWritesNothingToStdout TestMcpTakesNoArguments TestContextWithNoPatternIsRefused TestABadHeaderIsOneErrorNotOnePerBodyLine TestATabAfterTheAtSignsIsNamed TestThePaddedPathFixQuotesAnApostrophe; do grep -qE "^--- PASS: $t \(" /tmp/adr078-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^func TestThePaddedPathFixNamesTheCmdExeForm(' cmd/mrw/cmdquote_windows_test.go \
  && grep -q '^# 155\. ' scripts/contract.sh \
  && grep -q '^# 159\. ' scripts/contract.sh \
  && grep -q 'cmd.exe' internal/guide/guide.go \
  && GOOS=windows go vet ./cmd/mrw/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/check internal/state internal/lines internal/iter internal/seen internal/subproc internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/check internal/state internal/lines internal/iter internal/seen internal/subproc internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAUsageErrorWritesNothingToStdout` | `cmd/mrw/usage078_test.go` | a bad flag on the root, `mcp`, `read`, `write`: exit 2, stdout empty, names `--help`; `--help` prints | — | S1, S2 |
| `TestMcpTakesNoArguments` | `cmd/mrw/usage078_test.go` | `mcp stray` is exit 2, nothing on stdout | — | S1, S2 |
| `TestContextWithNoPatternIsRefused` | `cmd/mrw/usage078_test.go` | `-C` on a range refused; with `--ast-grep` refused by name; a noted working set leaves stdout empty; on a pattern and a grep it serves context | — | S1, S2 |
| `TestABadHeaderIsOneErrorNotOnePerBodyLine` | `internal/plan/parse078_test.go` | one error for a bad op; stray text before any header still reported | — | S1, S2 |
| `TestATabAfterTheAtSignsIsNamed` | `internal/plan/parse078_test.go` | the tab named, as one error; a second tab header under a broken one named too | — | S1, S2 |
| `TestThePaddedPathFixNamesTheCmdExeForm` | `cmd/mrw/cmdquote_windows_test.go` | Windows CI: the refusal names the cmd.exe form, keeps a name's apostrophe, and cmd.exe runs it | — | S1, S2 |
| `TestThePaddedPathFixQuotesAnApostrophe` | `cmd/mrw/usage078_test.go` | the POSIX fix closes and reopens its quote around a name's apostrophe | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the functions under Produces |
| 2 — something selects it | every CLI call, or every MCP request |
| 3 — the caller can discover it | each refusal names the fix |
| 4 — it is used | the v1.25.1 adversarial round and the review of #232 hit each |

## Verification Log
(empty until execute)
- 2026-09-26 · 6b17481* · exit 1 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:919 · test-lock-sha256:40a58f7a4dce1c40117bb2e5ba6f8cb357e2177a3f0e9b08aa3f58c8862ff3d1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvY21kcXVvdGVfd2luZG93c190ZXN0LmdvCVRlc3RUaGVQYWRkZWRQYXRoRml4TmFtZXNUaGVDbWRFeGVGb3JtCTkzNDdiNzE5NzMwNmM1OTEyZTE4MzdhNjNhNGIyMGU1MDg3Yjk4NDRmNjViYzhmMzEyYzZjOWEwMTY3MTE2NmMKYm9keQljbWQvbXJ3L3VzYWdlMDc4X3Rlc3QuZ28JVGVzdEFVc2FnZUVycm9yV3JpdGVzTm90aGluZ1RvU3Rkb3V0CTRhZDRmYWU3MTk2YTExMTFmMmJmNDM3NTA4N2M5MTkxNmI4MGExM2Y3NmNhOGRlMzJlNjM4NjI1MWZiN2JmOGIKYm9keQljbWQvbXJ3L3VzYWdlMDc4X3Rlc3QuZ28JVGVzdENvbnRleHRXaXRoTm9QYXR0ZXJuSXNSZWZ1c2VkCTBiNjM3NjlhNWYzMzEwMzE3MGIxZjk0NDA1MTFiYzFkZDE4ZTI3MzFkNjM4ZDJkMDA5MGI0YzVhY2NmZWIyN2EKYm9keQljbWQvbXJ3L3VzYWdlMDc4X3Rlc3QuZ28JVGVzdE1jcFRha2VzTm9Bcmd1bWVudHMJMThmMTdkMWExMTIxMzgyZGE5NzU2ZTc3NTIwOWFhYWNjYjRlZTI5NGJjZDhiZjFmYWZhYTRkMzNmMDhjMmRlMQpib2R5CWludGVybmFsL3BsYW4vcGFyc2UwNzhfdGVzdC5nbwlUZXN0QUJhZEhlYWRlcklzT25lRXJyb3JOb3RPbmVQZXJCb2R5TGluZQkxZmQ0YTQzNjE1ZDBhNzdiY2ZkMmE0MDE3NGFlMWE4YzJjOTgwNGRhZWMwOTdkZTZhMGMzMjM0MDA2MTUzYmMzCmJvZHkJaW50ZXJuYWwvcGxhbi9wYXJzZTA3OF90ZXN0LmdvCVRlc3RBVGFiQWZ0ZXJUaGVBdFNpZ25zSXNOYW1lZAk3YTU1YzBlNTlhMzFiZTBkYTk2NDFiNmI2NmNiMzkxYzA5MzE2ZGYxMjc4MDIwODZhZTNjMGZhNTUwNWRiNjZi
  ```
  --- last 10 line(s) of stdout (of 235 after folding 235 raw)
      parse078_test.go:27: want the tab named, as one error: plan has 2 error(s):
            line 1: text before the first @@ header: "@@\ta.go\t1\treplace"
            line 2: text before the first @@ header: "X"
  --- FAIL: TestATabAfterTheAtSignsIsNamed (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.160s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide	0.157s [no tests to run]
  FAIL
  ```
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:1298
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:517
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:461
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:456
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:480
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:466
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:1533
- 2026-09-26 · 6b17481* · exit 0 · `set -o pipefail …` · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · ms:634
- 2026-09-26 · 97c7d92* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:0 · test-lock-sha256:77a4b8f5fef829b9607407635dbc9b602aa0e0f43cd407364f35d1858e390cd5 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvY21kcXVvdGVfd2luZG93c190ZXN0LmdvCVRlc3RUaGVQYWRkZWRQYXRoRml4TmFtZXNUaGVDbWRFeGVGb3JtCTJhZThjMWU5OTE0NGZjZDkwNTU1MzdkNjYxMTYzMjE4NzQ2MmM1MGRhMDMzNDQxOWZkOGNkNDY3M2RlMzljYmQKYm9keQljbWQvbXJ3L3VzYWdlMDc4X3Rlc3QuZ28JVGVzdEFVc2FnZUVycm9yV3JpdGVzTm90aGluZ1RvU3Rkb3V0CTRhZDRmYWU3MTk2YTExMTFmMmJmNDM3NTA4N2M5MTkxNmI4MGExM2Y3NmNhOGRlMzJlNjM4NjI1MWZiN2JmOGIKYm9keQljbWQvbXJ3L3VzYWdlMDc4X3Rlc3QuZ28JVGVzdENvbnRleHRXaXRoTm9QYXR0ZXJuSXNSZWZ1c2VkCTJhMDYyNDc5ZjYyMjY2MmE0NWNmMzQ3Y2U3NTQ4ZmRkN2EzM2U1MjNiYTk5MTc2NjljNDg2NDBmOWRhODhlZmIKYm9keQljbWQvbXJ3L3VzYWdlMDc4X3Rlc3QuZ28JVGVzdE1jcFRha2VzTm9Bcmd1bWVudHMJMThmMTdkMWExMTIxMzgyZGE5NzU2ZTc3NTIwOWFhYWNjYjRlZTI5NGJjZDhiZjFmYWZhYTRkMzNmMDhjMmRlMQpib2R5CWNtZC9tcncvdXNhZ2UwNzhfdGVzdC5nbwlUZXN0VGhlUGFkZGVkUGF0aEZpeFF1b3Rlc0FuQXBvc3Ryb3BoZQkwMGMzZDczOWEyZGY3NzI4MTlhMjQyNzlmYmZjZGM3MmYyNGUwMjUxMWY2OGEzZTZlMGViNDU4ZDAyODQxOWE4CmJvZHkJaW50ZXJuYWwvcGxhbi9wYXJzZTA3OF90ZXN0LmdvCVRlc3RBQmFkSGVhZGVySXNPbmVFcnJvck5vdE9uZVBlckJvZHlMaW5lCTFmZDRhNDM2MTVkMGE3N2JjZmQyYTQwMTc0YWUxYThjMmM5ODA0ZGFlYzA5N2RlNmEwYzMyMzQwMDYxNTNiYzMKYm9keQlpbnRlcm5hbC9wbGFuL3BhcnNlMDc4X3Rlc3QuZ28JVGVzdEFUYWJBZnRlclRoZUF0U2lnbnNJc05hbWVkCTljMjgyODQxOGRhODhmYWQ2ZmYxNmI1MDFjNWZmY2UxMjZhOTY0MTYwNjQ5ZjAwMzk4MjMyYjYzYTEzOTIwNTA · test-lock-kind:replace
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:790
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:787
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:563
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:413
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:563
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:417
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:435
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:447
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:568
- 2026-09-26 · 97c7d92* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · ms:666

## Mutation Log
(empty until execute)
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `cmd/mrw/main.go` · a usage error prints help to stdout again · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · covers:a usage error writes nothing to stdout
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `cmd/mrw/main.go` · mcp starts past a stray argument · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · covers:mcp takes no arguments
- 2026-09-26 · 6b17481* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · every range counts as a pattern for -C · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · covers:-C needs a pattern
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/plan/plan.go` · a bad header's body is reported line by line · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · covers:a bad header is one error
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `internal/plan/plan.go` · a tab header reads as prose · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · covers:the tab is named
- 2026-09-26 · 6b17481* · mutant killed · exit 1 · `cmd/mrw/main.go` · every range counts as a pattern for -C · acceptance-sha256:03f8577b1eb35514d1ed0c538d071630d9c1b3b8c098e8056d4c4e3c3610d25e · covers:-C needs a pattern
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `cmd/mrw/main.go` · a usage error prints help to stdout again · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:a usage error writes nothing to stdout
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `cmd/mrw/main.go` · mcp starts past a stray argument · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:mcp takes no arguments
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `cmd/mrw/main.go` · every range counts as a pattern for -C · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:-C needs a pattern
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `cmd/mrw/main.go` · -C with --ast-grep gets the pattern refusal after the finder runs · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:-C needs a pattern
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `cmd/mrw/main.go` · the note prints before the refusals · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:a usage error writes nothing to stdout
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/plan/plan.go` · a bad header's body is reported line by line · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:a bad header is one error
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/plan/plan.go` · a tab header reads as prose · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:the tab is named
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `internal/plan/plan.go` · a tab header under a broken one is swallowed · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:the tab is named
- 2026-09-26 · 97c7d92* · mutant killed · exit 1 · `cmd/mrw/main.go` · the POSIX fix keeps a bare apostrophe · acceptance-sha256:cf6d1a7763466d12739dfa2b76b565a82f88319ad69d51dfa892a62c707b6bb5 · covers:the POSIX fix quotes an apostrophe

## Invariants

- Nothing but MCP messages reaches `mrw mcp`'s stdout; a usage error is exit 2 on every command.

## Risks

- A script that parsed the help text a usage error printed; `--help` still prints it.

## Out of Scope

- A `@@` and a tab inside an uncounted body (permanent: fact: a body line is content; a counted body is the escape)

## Stop Condition

The fence exits 0 and the latest Mutation Log row for each mutant reads `killed`.
