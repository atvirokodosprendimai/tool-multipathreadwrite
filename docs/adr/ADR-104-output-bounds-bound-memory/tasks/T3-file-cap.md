# Task ADR-104-T3: a file over the read cap is refused by name

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `maxFileBytes` in `read`; the refusal in `Run` and the walk
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a file over the cap is refused before it is read`, `under the cap nothing changes`, `only the owned engine packages change`

## Goal

`read` refuses a regular file larger than `maxFileBytes` (1 GiB) before reading it: UNREADABLE with its size and
the limit; the `--grep` walk reports it as a problem naming the same; a smaller file is served as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | the size check beside the non-regular check (`:446`); `maxFileBytes` |
| `internal/read/walk.go` | edit | `offer` (`:252`) checks the size first |
| `internal/read/cap104_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestAFileOverTheReadCapIsRefusedByName`; confirm RED. [proof: mutation]
2. [S2] Check the size first. [proof: mutation] Mutants: the check removed from `Run`; the check removed from the walk.

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAFileOverTheReadCapIsRefusedByName|TestReadCappedRefusesAStreamOverTheCap|TestAnAstGrepHitOnAFileOverTheCapIsReportedOnce' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAFileOverTheReadCapIsRefusedByName \(' "$out" \
  && grep -qE '^--- PASS: TestReadCappedRefusesAStreamOverTheCap \(' "$out" \
  && grep -qE '^--- PASS: TestAnAstGrepHitOnAFileOverTheCapIsReportedOnce \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFileOverTheReadCapIsRefusedByName` | `internal/read/cap104_test.go` | with the cap set to 16 bytes, reading a 40-byte file answers UNREADABLE naming 40 bytes and the limit and exits with a problem; a 10-byte file is served; `--grep` over both reports the large one as a problem naming the limit and matches the small one | — | S1, S2 |
| `TestReadCappedRefusesAStreamOverTheCap` | `internal/read/cap104_test.go` | (the review of #295) `/dev/zero`, which a stat calls empty, is refused by the bounded read once the limit is passed | — | S2 |
| `TestAnAstGrepHitOnAFileOverTheCapIsReportedOnce` | `internal/read/cap104_test.go` | (the review of #295) two ast-grep hits on a file over the cap give no spec and one problem naming the limit | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every check, every MCP request, every read and ast-grep run goes through it |
| 3 — the caller can discover it | the refusal or the marker names the limit |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/read/read.go` · the read cap check removed · acceptance-sha256:21870331b1e9b297505b7e11ecb95061b38fd20e436457ed448e410f763c9589 · covers:a file over the cap is refused before it is read
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/read/walk.go` · the walk cap check removed · acceptance-sha256:21870331b1e9b297505b7e11ecb95061b38fd20e436457ed448e410f763c9589 · covers:a file over the cap is refused before it is read
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/read/read.go` · the bounded read no longer refuses what outran the stat · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · covers:a file over the cap is refused before it is read
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/read/astgrep.go` · the ast-grep probe ignores the cap · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · covers:a file over the cap is refused before it is read
- 2026-09-30 · f2b0ee2* · mutant survived · exit 0 · `internal/read/read.go` · the read cap check removed · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · covers:under the cap nothing changes
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package this record does not own changed · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · covers:only the owned engine packages change
- 2026-09-30 · f2b0ee2* · mutant killed · exit 1 · `internal/read/read.go` · readCapped no longer refuses by size first · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · covers:under the cap nothing changes

## Invariants

- Under the limit, every answer is byte-identical to before.
- Exit codes keep their meanings.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-104 tasks, each in its own file.

## Verification Log
- 2026-09-30 · f911dd7* · exit 1 · `set -o pipefail …` · acceptance-sha256:21870331b1e9b297505b7e11ecb95061b38fd20e436457ed448e410f763c9589 · ms:115 · test-lock-sha256:520c51f433c3fb91ee333f256532a7fc9a6dbb66a8c328df2031a6c752d64576 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3JlYWQvY2FwMTA0X3Rlc3QuZ28JVGVzdEFGaWxlT3ZlclRoZVJlYWRDYXBJc1JlZnVzZWRCeU5hbWUJY2ZiMjMzMWEzMjc5YTgyYTEyYThmODQyMmM0ZjE2MTViNWIzOGY0MWYwM2M1MGYzYzczZTk4MjFkNWZhZmFkNQpib2R5CWludGVybmFsL3JlYWQvY2FwMTA0X3Rlc3QuZ28JVGVzdEFuQXN0R3JlcEFuc3dlck92ZXJUaGVDYXBJc1JlZnVzZWQJNWQ3OGZjZWU0OWFmZWFkNTFlMjA0ZmQzZTUwYjg4NjYyMjZhNzQ4Zjk3YzU4NTM5OTE3MTU3ODAxN2RlZmI4Yw
  ```
  --- last 10 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read.test]
  internal/read/cap104_test.go:15:9: undefined: maxFileBytes
  internal/read/cap104_test.go:16:2: undefined: maxFileBytes
  internal/read/cap104_test.go:17:21: undefined: maxFileBytes
  internal/read/cap104_test.go:51:9: undefined: maxAstGrepBytes
  internal/read/cap104_test.go:52:21: undefined: maxAstGrepBytes
  internal/read/cap104_test.go:53:2: undefined: maxAstGrepBytes
  internal/read/cap104_test.go:57:2: undefined: maxAstGrepBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  FAIL
  ```
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:21870331b1e9b297505b7e11ecb95061b38fd20e436457ed448e410f763c9589 · ms:304
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:21870331b1e9b297505b7e11ecb95061b38fd20e436457ed448e410f763c9589 · ms:283
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · ms:575
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · ms:603
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · ms:573
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · ms:624
- 2026-09-30 · human-observed · note 2026-09-30: the survived mutant 'the read cap check removed' showed read.go's size check (and the walk's) had become redundant once readCapped refuses by size first with the same message; both are deleted, so readCapped is the one place the cap is enforced; approved
- 2026-09-30 · f2b0ee2* · exit 0 · `set -o pipefail …` · acceptance-sha256:a66f74287043d88bcad736492dbb3dccb1dcb2d51f4c2d0d0054536e962bccbf · ms:668
