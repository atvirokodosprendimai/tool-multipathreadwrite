# Task ADR-149-T1: `mrw version --check-latest`, the one network path

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `checkLatest` and `releasesAPI` in `cmd/mrw/latest149.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a newer release exits 1`, `current and newer exit 0`, `a dev build gives no verdict`, `could not ask exits 2`, `the flag is what asks`, `only one file imports the network`, `the docs teach the flag`

## Goal

Decisions 1–5 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/latest149.go` | add | `checkLatest`, `releasesAPI`, the semver comparison |
| `cmd/mrw/main.go` | edit | the `--check-latest` flag on `versionCmd` |
| `cmd/mrw/latest149_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §253 drives a built binary against a local server |
| `README.md`, `AGENTS.md` | edit | one sentence each |
| `docs/adr/ADR-009-mrw-counts-what-happens-to-the-plans-it-is-given.md` | edit | the pointer on the telemetry rejection |
| `docs/adr/BACKLOG.md` | edit | the survey's staleness line is taken |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestCheckLatestSaysWhetherANewerReleaseExists` (a local `httptest` server answers each case: equal → exit 0 "is current"; older running version → exit 1, stdout names the latest and the download URL, the error says a newer release exists; newer → 0; `dev` → 0 "cannot be compared"; 404, a body that is not JSON, a tag that is not strict, a closed server → exit 2 "could not ask"; the request path is the releases-latest path and the User-Agent is `mrw/<version>`), `TestVersionWithoutTheFlagAsksNothing` (plain `mrw version` against a counting server: zero requests), `TestOnlyTheCheckLatestFileImportsTheNetwork` (every non-test `.go` file under `cmd/` and `internal/` but `cmd/mrw/latest149.go` imports neither `net` nor `net/...`).
2. [S2] `checkLatest`, `releasesAPI` and the flag. [proof: mutation]
3. [S3] README, AGENTS.md, the ADR-009 pointer, the BACKLOG line. [proof: mutation]
4. [S4] Contract §253: a binary built with `-X main.version=v1.65.0 -X main.releasesAPI=http://127.0.0.1:<port>` against a local server: current → 0, behind → 1, server gone → 2, not JSON → 2; a `dev` build → 0 and "cannot be compared". The fence greps only its heading. [proof: human: the executor runs ./scripts/contract.sh once and reads the §253 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestCheckLatestSaysWhetherANewerReleaseExists|TestVersionWithoutTheFlagAsksNothing|TestOnlyTheCheckLatestFileImportsTheNetwork|TestVersionCommandPrintsVersionString' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestCheckLatestSaysWhetherANewerReleaseExists \(' "$out" \
  && grep -qE '^--- PASS: TestVersionWithoutTheFlagAsksNothing \(' "$out" \
  && grep -qE '^--- PASS: TestOnlyTheCheckLatestFileImportsTheNetwork \(' "$out" \
  && grep -qE '^--- PASS: TestVersionCommandPrintsVersionString \(' "$out" \
  && grep -q '^# 253\. ' scripts/contract.sh \
  && grep -q 'mrw version --check-latest' README.md \
  && grep -q 'mrw version --check-latest' AGENTS.md \
  && grep -q 'Amended by ADR-149' docs/adr/ADR-009-mrw-counts-what-happens-to-the-plans-it-is-given.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestCheckLatestSaysWhetherANewerReleaseExists` | `cmd/mrw/latest149_test.go` | each answer's line and exit code; the request's path and User-Agent | none | S1, S2 |
| `TestVersionWithoutTheFlagAsksNothing` | `cmd/mrw/latest149_test.go` | plain `mrw version` makes no request | none | S1, S2 |
| `TestOnlyTheCheckLatestFileImportsTheNetwork` | `cmd/mrw/latest149_test.go` | no other shipped file imports `net` | none | S1 |
| `TestVersionCommandPrintsVersionString` | `cmd/mrw/version_test.go` | ADR-040 holds: `mrw version` prints what it printed | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `checkLatest` |
| 2 — something selects it | the `--check-latest` flag on `versionCmd`; contract §253 drives a built binary |
| 3 — the caller can discover it | `mrw version --help`, README, AGENTS.md |
| 4 — it is used | nothing measures this (ADR-009); built because the survey and the 2026-10-13 Windows round found stale binaries on PATH |

## Mutation Log
(empty until execute)
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · an older mrw exits 0 · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/main.go` · plain mrw version asks the network · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · a dev build is compared as if released · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · could not ask exits 1 like a newer release · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `README.md` · the README stops teaching the flag · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · an older mrw exits 0 · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:a newer release exits 1
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · compareSemver calls the lower version higher, so current and newer misreport · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:current and newer exit 0
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · a dev build is compared as if released · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:a dev build gives no verdict
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · could not ask exits 1 like a newer release · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:could not ask exits 2
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/main.go` · plain mrw version asks the network · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:the flag is what asks
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `cmd/mrw/main.go` · a second shipped file imports the network · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:only one file imports the network
- 2026-10-14 · 6b565cb* · mutant killed · exit 1 · `README.md` · the README stops teaching the flag · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:the docs teach the flag
- 2026-10-14 · f4b7283* · mutant killed · exit 1 · `cmd/mrw/latest149.go` · an answer over 1 MiB is parsed as if whole · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · covers:could not ask exits 2

## Verification Log
(empty until execute)
- 2026-10-14 · 6b565cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:539 · test-lock-sha256:cefc00c34b33af8b7265244fd7fd6571e210751c010ffe03c8664cdfe941f2d7 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9sYXRlc3QxNDlfdGVzdC5nbwlUZXN0Q2hlY2tMYXRlc3RTYXlzV2hldGhlckFOZXdlclJlbGVhc2VFeGlzdHMJMTI3NzBhOTllZWM5NjdkZjgyNjM3ZDBkZjZkODk2ZjFlYzZmOWE3OTBmYmI5MzZkZWZhN2MyNDkxYWVjZTMzMwpib2R5CWNtZC9tcncvbGF0ZXN0MTQ5X3Rlc3QuZ28JVGVzdE9ubHlUaGVDaGVja0xhdGVzdEZpbGVJbXBvcnRzVGhlTmV0d29yawkxM2ZkNjFlZWQwNTA4MmUxNzY0YzMwODA4NmQ5OGFjMjQwNTA1ODI1YjNkZWU5MzhjOWQ5YjFiYmY1NzJjOGE4CmJvZHkJY21kL21ydy9sYXRlc3QxNDlfdGVzdC5nbwlUZXN0VmVyc2lvbldpdGhvdXRUaGVGbGFnQXNrc05vdGhpbmcJOWUyYzNjNjM0YmRhNjQ5ZjVjODZjMTVhODY5ZDA3NjM3YTBjMzAxMTNiYzI5MGU3NmI2MTM0YjkzMDE2NGJkOQpib2R5CWNtZC9tcncvbGF0ZXN0MTQ5X3Rlc3QuZ28JYSBzZXJ2ZXIgdGhhdCBpcyBnb25lIGNvdWxkIG5vdCBhc2sJNDVkMzdjZmY4NGE0Y2YzNTU1NjJmMTgyODlmOGYxOGE0NDc4NDA5ZGRiNTI0NGViYTgxMTUwZGMzMjkyMGFlMApib2R5CWNtZC9tcncvdmVyc2lvbl90ZXN0LmdvCVRlc3RWZXJzaW9uQ29tbWFuZFByaW50c1ZlcnNpb25TdHJpbmcJN2JlMzVkZDBmMzg1N2FlYjEzZjAyNDVmMTM0OTczMDYwODVhOWZmOWJjNmI3YWQwYmRkMTQyYTAwYWVmODBiYQpib2R5CWNtZC9tcncvdmVyc2lvbl90ZXN0LmdvCVRlc3RWZXJzaW9uRmxhZ1ByaW50c1RoZVN0YW1wZWRWZXJzaW9uCWU4MTlkZmQzYzc4NGZlMjVhNzA0NDk5MzNiNTE2NjUzMzhlN2FlODBmYzU5NGU0Y2YyZDJiMzVhYWZhMTQzMjgKYm9keQljbWQvbXJ3L3ZlcnNpb25fdGVzdC5nbwlUZXN0VmVyc2lvbkZsYWdQcmludHNXaGF0VmVyc2lvblN0cmluZ0NvbXBvc2VzCTFjNGE1MjNlNWRlNzE5ZjQ3NzE1ZjVkMWVhNDA5MjQ2ZjlhYTIyMGQzNWY1MGY5MzNkYWYzZDIxNWNhMTNhZTgKYm9keQljbWQvbXJ3L3ZlcnNpb25fdGVzdC5nbwlUZXN0VmVyc2lvbklzQVZhcmlhYmxlVGhlTGlua2VyQ2FuU3RhbXAJYmZhOTE3YmI2MjExNTQyMWRjMWIwZDY5ZjU0YWRiZGUzMTQwYmI1NmJhZmMxZWZjOTQyYThlNjdiZmNjZDBjNgpib2R5CWNtZC9tcncvdmVyc2lvbl90ZXN0LmdvCVRlc3RWZXJzaW9uUmVwb3J0c1RoZVJldmlzaW9uSXRXYXNCdWlsdEZyb20JN2RlMjRkYWFhZjEzOGQ3MzJiZGNiY2ZlZmYyYTcxMzA4Y2Y2ZDZlMDE3YWZmZDg3MzhmMWIwM2M0OWM2OTU0Mw
  ```
  --- last 10 line(s) of stdout (of 48 after folding 48 raw)
  === RUN   TestVersionWithoutTheFlagAsksNothing
  --- PASS: TestVersionWithoutTheFlagAsksNothing (0.00s)
  === RUN   TestOnlyTheCheckLatestFileImportsTheNetwork
      latest149_test.go:183: cmd/mrw/latest149.go imports no net package: the walk is not reaching the file it allows
  --- FAIL: TestOnlyTheCheckLatestFileImportsTheNetwork (0.00s)
  === RUN   TestVersionCommandPrintsVersionString
  --- PASS: TestVersionCommandPrintsVersionString (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.129s
  FAIL
  ```
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1510
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1305
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1304
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1304
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1531
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1744
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1851
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:2008
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1623
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1636
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1493
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:1477
- 2026-10-14 · human-observed · observed by the executor 2026-10-14: S4 ./scripts/contract.sh exit 0 ('contract holds'); all thirteen §253 rows PASS — current exits 0, the server saw the request and plain mrw version asks nothing, behind exits 1 naming the newer release, a dev build exits 0 with no verdict, not-JSON and a gone server exit 2 saying it could not ask
- 2026-10-14 · 6b565cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:679
- 2026-10-14 · f4b7283* · exit 0 · `set -o pipefail …` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:3092
- 2026-10-14 · 83c8042 · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b36d890843739af3d24a2d869ea9ef5e3d50c9bfb90aab507ab21be3d64a32e8 · ms:0 · test-lock-sha256:044671eef06a7297494f5b817fe4d33ae7224494052d364547fe03b9c2c1891d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9sYXRlc3QxNDlfdGVzdC5nbwlUZXN0Q2hlY2tMYXRlc3RTYXlzV2hldGhlckFOZXdlclJlbGVhc2VFeGlzdHMJMDIzMTBhMDA5YjQxODg4YjQ5MDUxOTZkZTI0Mzc2ZjE4MTRlMWFkOWU4Mjg5ZmRhOGZlODgxNDliMTIwYjY4Ywpib2R5CWNtZC9tcncvbGF0ZXN0MTQ5X3Rlc3QuZ28JVGVzdE9ubHlUaGVDaGVja0xhdGVzdEZpbGVJbXBvcnRzVGhlTmV0d29yawkyYmYzN2NhNGE3ODY0Njk1MjNlYjIzYmE3ZDgyMmY4N2M2NTVlNDgyYmU5MmJhY2RlNzcwNGUxNTk1MWU2NmM3CmJvZHkJY21kL21ydy9sYXRlc3QxNDlfdGVzdC5nbwlUZXN0VmVyc2lvbldpdGhvdXRUaGVGbGFnQXNrc05vdGhpbmcJOWUyYzNjNjM0YmRhNjQ5ZjVjODZjMTVhODY5ZDA3NjM3YTBjMzAxMTNiYzI5MGU3NmI2MTM0YjkzMDE2NGJkOQpib2R5CWNtZC9tcncvbGF0ZXN0MTQ5X3Rlc3QuZ28JYSByZWRpcmVjdCBpcyBmb2xsb3dlZCB0byB0aGUgYW5zd2VyIGl0IGVuZHMgYXQJMmEwMjVhYjg3MDEyYzgyNmE1NGI3MWQ4ODQwYTQ2Nzk3ZDE0YzMyMTU2NTI3NmZkYzVlNmQwMGQzNGY3MjQ1YQpib2R5CWNtZC9tcncvbGF0ZXN0MTQ5X3Rlc3QuZ28JYSBzZXJ2ZXIgdGhhdCBpcyBnb25lIGNvdWxkIG5vdCBhc2sJNDVkMzdjZmY4NGE0Y2YzNTU1NjJmMTgyODlmOGYxOGE0NDc4NDA5ZGRiNTI0NGViYTgxMTUwZGMzMjkyMGFlMApib2R5CWNtZC9tcncvdmVyc2lvbl90ZXN0LmdvCVRlc3RWZXJzaW9uQ29tbWFuZFByaW50c1ZlcnNpb25TdHJpbmcJN2JlMzVkZDBmMzg1N2FlYjEzZjAyNDVmMTM0OTczMDYwODVhOWZmOWJjNmI3YWQwYmRkMTQyYTAwYWVmODBiYQpib2R5CWNtZC9tcncvdmVyc2lvbl90ZXN0LmdvCVRlc3RWZXJzaW9uRmxhZ1ByaW50c1RoZVN0YW1wZWRWZXJzaW9uCWU4MTlkZmQzYzc4NGZlMjVhNzA0NDk5MzNiNTE2NjUzMzhlN2FlODBmYzU5NGU0Y2YyZDJiMzVhYWZhMTQzMjgKYm9keQljbWQvbXJ3L3ZlcnNpb25fdGVzdC5nbwlUZXN0VmVyc2lvbkZsYWdQcmludHNXaGF0VmVyc2lvblN0cmluZ0NvbXBvc2VzCTFjNGE1MjNlNWRlNzE5ZjQ3NzE1ZjVkMWVhNDA5MjQ2ZjlhYTIyMGQzNWY1MGY5MzNkYWYzZDIxNWNhMTNhZTgKYm9keQljbWQvbXJ3L3ZlcnNpb25fdGVzdC5nbwlUZXN0VmVyc2lvbklzQVZhcmlhYmxlVGhlTGlua2VyQ2FuU3RhbXAJYmZhOTE3YmI2MjExNTQyMWRjMWIwZDY5ZjU0YWRiZGUzMTQwYmI1NmJhZmMxZWZjOTQyYThlNjdiZmNjZDBjNgpib2R5CWNtZC9tcncvdmVyc2lvbl90ZXN0LmdvCVRlc3RWZXJzaW9uUmVwb3J0c1RoZVJldmlzaW9uSXRXYXNCdWlsdEZyb20JN2RlMjRkYWFhZjEzOGQ3MzJiZGNiY2ZlZmYyYTcxMzA4Y2Y2ZDZlMDE3YWZmZDg3MzhmMWIwM2M0OWM2OTU0Mw · test-lock-kind:replace

## Invariants

- `mrw version`, `-v` and `--version` print what they printed; nothing under `internal/` changes; `go.mod` keeps one requirement.

## Risks

- None beyond the record's.

## Out of Scope

- The MCP server (permanent: boundary: the record's Decision 4)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
