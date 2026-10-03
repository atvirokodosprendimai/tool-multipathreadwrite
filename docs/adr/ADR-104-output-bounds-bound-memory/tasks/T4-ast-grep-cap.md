# Task ADR-104-T4: an ast-grep answer over the cap is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `subproc.Output`'s limit; `maxAstGrepBytes` in `read`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an oversized answer is refused, not read whole`, `under the cap nothing changes`, `only the owned engine packages change`

## Goal

`subproc.Output` takes a limit and refuses an answer larger than it without reading it; `read.AstGrep` passes
`maxAstGrepBytes` (256 MiB) and refuses with the size and the limit, advising a narrower pattern or fewer paths.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/subproc/subproc.go` | edit | `Output(c, limit)`; `ErrOutputTooLarge` |
| `internal/read/astgrep.go` | edit | passes `maxAstGrepBytes`; names the refusal |
| `internal/read/cap104_test.go` | edit | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestAnAstGrepAnswerOverTheCapIsRefused`; confirm RED. [proof: mutation]
2. [S2] Cap the answer. [proof: mutation] Mutant: the size check in `Output` removed.

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ ./internal/subproc/ -count=1 -timeout 300s -run 'TestAnAstGrepAnswerOverTheCapIsRefused' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnAstGrepAnswerOverTheCapIsRefused \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnAstGrepAnswerOverTheCapIsRefused` | `internal/read/cap104_test.go` | with a fake `ast-grep` answering 200 bytes and the cap set to 64, `AstGrep` returns an error naming 200 bytes and the limit; with the cap at 1 KiB the same answer is parsed | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every check, every MCP request, every read and ast-grep run goes through it |
| 3 — the caller can discover it | the refusal or the marker names the limit |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/subproc/subproc.go` · the size check in Output removed · acceptance-sha256:7cc6a0cb5f3abcddb1a9ccf5b68934d709ed28bac45686e82a490efb36cd008e · covers:an oversized answer is refused, not read whole

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
- 2026-09-30 · f911dd7* · exit 1 · `set -o pipefail …` · acceptance-sha256:7cc6a0cb5f3abcddb1a9ccf5b68934d709ed28bac45686e82a490efb36cd008e · ms:503 · test-lock-sha256:520c51f433c3fb91ee333f256532a7fc9a6dbb66a8c328df2031a6c752d64576 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL3JlYWQvY2FwMTA0X3Rlc3QuZ28JVGVzdEFGaWxlT3ZlclRoZVJlYWRDYXBJc1JlZnVzZWRCeU5hbWUJY2ZiMjMzMWEzMjc5YTgyYTEyYThmODQyMmM0ZjE2MTViNWIzOGY0MWYwM2M1MGYzYzczZTk4MjFkNWZhZmFkNQpib2R5CWludGVybmFsL3JlYWQvY2FwMTA0X3Rlc3QuZ28JVGVzdEFuQXN0R3JlcEFuc3dlck92ZXJUaGVDYXBJc1JlZnVzZWQJNWQ3OGZjZWU0OWFmZWFkNTFlMjA0ZmQzZTUwYjg4NjYyMjZhNzQ4Zjk3YzU4NTM5OTE3MTU3ODAxN2RlZmI4Yw
  ```
  --- last 10 line(s) of stdout (of 13 after folding 13 raw)
  internal/read/cap104_test.go:17:21: undefined: maxFileBytes
  internal/read/cap104_test.go:51:9: undefined: maxAstGrepBytes
  internal/read/cap104_test.go:52:21: undefined: maxAstGrepBytes
  internal/read/cap104_test.go:53:2: undefined: maxAstGrepBytes
  internal/read/cap104_test.go:57:2: undefined: maxAstGrepBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc	0.308s [no tests to run]
  FAIL
  ```
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc6a0cb5f3abcddb1a9ccf5b68934d709ed28bac45686e82a490efb36cd008e · ms:613
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc6a0cb5f3abcddb1a9ccf5b68934d709ed28bac45686e82a490efb36cd008e · ms:736
- 2026-10-03 · 4b07a1b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:7cc6a0cb5f3abcddb1a9ccf5b68934d709ed28bac45686e82a490efb36cd008e · ms:0 · test-lock-sha256:d4373f8faa922dde5dd2479fc77f943e11d3212dea1ca20a18b226e4d0050329 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9jYXAxMDRfdGVzdC5nbwlUZXN0QUZpbGVPdmVyVGhlUmVhZENhcElzUmVmdXNlZEJ5TmFtZQljZmIyMzMxYTMyNzlhODJhMTJhOGY4NDIyYzRmMTYxNWI1YjM4ZjQxZjAzYzUwZjNjNzNlOTgyMWQ1ZmFmYWQ1CmJvZHkJaW50ZXJuYWwvcmVhZC9jYXAxMDRfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwQW5zd2VyT3ZlclRoZUNhcElzUmVmdXNlZAlmM2JiMWViZjhhODQ3NDM3Y2ViNjA0ZWM4ODQ3ZjZmOTFlMjc4YjkwOTc0ZmVhMzRjMDY0NWUwZmM2NjI5MDYwCmJvZHkJaW50ZXJuYWwvcmVhZC9jYXAxMDRfdGVzdC5nbwlUZXN0QW5Bc3RHcmVwSGl0T25BRmlsZU92ZXJUaGVDYXBJc1JlcG9ydGVkT25jZQk4YzhiZTkzOThjNzgzOTc1NTk4ZmFiODkzZjhjZjg5N2NiMjQ0ZjVlZjEwOTIzZDhiNTI2Mzg1M2U3MDdjMmUzCmJvZHkJaW50ZXJuYWwvcmVhZC9jYXAxMDRfdGVzdC5nbwlUZXN0UmVhZENhcHBlZFJlZnVzZXNBU3RyZWFtT3ZlclRoZUNhcAk5MjM5ZjRmNzJlNjIyNTU2MDkxZWYyZmNjOTBlNGI3MzZmNzg0OTBlYTQ2ZDI0OWIxNTU1Zjk2NmExZDNlOTU1 · test-lock-kind:replace
