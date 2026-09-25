# Task ADR-071-T4: The Windows suite reaches its branches

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `needSh`, `plainTree`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a test that needs sh says so`, `a refusal needs no file`, `the tail test stops before it indexes`, `the engine packages are unchanged`

## Goal

Under PowerShell eleven `internal/check` tests failed rather than skipped for want of `sh`,
`TestTailAnnouncesWhatItLeftOut` panicked, and all thirteen padded-path tests skipped on NTFS
because their fixture needs a file named `x `. The refusals those tests pin fire before any I/O.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check_test.go` | edit | `needSh` at the top of every test that runs a command; `Fatalf` before `Tail[4]` |
| `internal/check/timeout_alias_test.go` | edit | `needSh` on the Zeus-shaped timeout test, which runs `sleep` |
| `cmd/mrw/paddedflag_test.go` | edit | `plainTree`; the file `x ` only where an assertion reads it |

## Ordered Steps

1. [S1] The fence is RED: `needSh` and `plainTree` do not exist, and under PowerShell the tail test panics (Windows report). Add `needSh` and the `Fatalf`; split the fixture; GREEN, and the suite stays green on macOS and Linux. [proof: acceptance]
2. [S2] The Windows shards run the eight padded tests that need no such file. CI runs `go test` without `-v`, so a pass prints no per-test line: the evidence is the `cmd/mrw` package passing on the shard, where these tests had skipped. The first Windows run of this change failed two tests that create their own ` -1= ` (NTFS stores it as ` -1=`); they went back to `paddedTree` and its skip. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ ./internal/check/ -count=1 -timeout 120s -run 'TestAFlagValue|TestAPadded|TestTailAnnounces' -v 2>&1 | tee /tmp/adr071-T4.out \
  && missing=$(for t in TestTailAnnouncesWhatItLeftOut; do grep -qE "^--- PASS: $t \(" /tmp/adr071-T4.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^func needSh(' internal/check/check_test.go \
  && grep -q '^func plainTree(' cmd/mrw/paddedflag_test.go \
&& git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/walk.go' internal/plan internal/seen internal/state internal/lines internal/iter internal/check ':(exclude)internal/check/*_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTailAnnouncesWhatItLeftOut` | `internal/check/check_test.go` | stops rather than panics when the tail is short | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the helpers |
| 2 — something selects it | every test that runs a command, every padded refusal test |
| 3 — the caller can discover it | a skip says it needs `sh` |
| 4 — it is used | the Windows shards |

## Verification Log
(empty until execute)
- 2026-09-25 · 77a408a* · exit 1 · `set -o pipefail …` · acceptance-sha256:495d8354db891c231dc04c73c9fd8388745d6ccbed21fa504d3dcd33b875e988 · ms:959 · test-lock-sha256:e9a7f303b06a35e2bc930cfd4f7a568977b7c33b6ca84abd9766a74b4a753c44 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFDaGVja1RoYXRDYW5ub3RTdGFydERpZE5vdFJ1bglkMTNhNzBiYTcyNzYxNTI0M2Y4ZjcwOTRjMTAyYjA5YTNhMTY0ZDIzYTQzMGMzMzNlNTFlOGMyNmZiZmNmNjIyCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QURpcmVjdG9yeVRoYXRDYW5ub3RCZVJlYWRJc1JlZnVzZWROb3RUcmVhdGVkQXNFbXB0eQk2NzIzN2E5M2E1MTI2Yjc4YmNlYWIwMjJhZmQzOTE0MDM0MDlhZDE4ZjY1YjRkNjVlZTIyODY4ZmQxZjZhMjAyCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QUZhaWxpbmdDaGVja0tlZXBzSXRzTG9nCTk5MWY0MjFmMGU3N2FiODQwZjMyOTQyZTI2YmI2NGFhYmM3MDllNGM0Y2UwYzA5YmZiOWM4ZGZiNWYxYzgwZWUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFkZGVkQ2hlY2tJc1N0aWxsVGhlRGVjbGFyZWRDaGVjawk5OWUwMjYyNjkyZjMxMWVkYTVjNGNmZjVhMzBkMWMwOGYyODFjOWI5NTUzMjgxYTc2ZmExZGZiYWE3MWJjODE1CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVBhZGRlZFNjb3BlZENoZWNrSXNTdGlsbERlY2xhcmVkCTZjNmVlODBkODNiMmQ2YmU2NzZjM2Y5YzAyN2FmM2JhMGI5ZDA2MjI4MDkyZWE1YmJjMGEzYzI1ZDVlOWM4MDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFzc2luZ0NoZWNrTGVhdmVzTm9Mb2dCZWhpbmQJNzU5ZTdjM2Q0Y2M0MTExYzExYTQzYzM4YmMwNTFlYTIzNzM1MjY1ODRmMzdiNTM2OTAxOTk4ZTY0ZmFhYjA3MApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFQcmVzZW50VW5wbGFjZWFibGVJblJvb3RTY29wZVN0aWxsRmFsbHNCYWNrCTU2YTAyZGY4NjEwMmY3ZTliY2JlZTM4OGYwNDJiNTY3Y2IwODg3MzExNWI0OTI2MmFhN2Y1YWUxZGU3ZjhkYjkKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUmVhZGFibGVEaXJlY3RvcnlTdGlsbFNjb3BlcwljNmVhY2E2N2JkMTI2Yjg0NzQ5YzlkNmEzZWVmMDJlMzI2NDI0MzgzYzVlYWEyOGZmOTA5NGY5MGViMDc2Y2Q3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVJlZnVzZWRTY29wZURvZXNOb3RJbmhlcml0VGhlUm9vdHNWZXJkaWN0CTk5MjY2NmU2M2Y4ZTU1ZGJhYTI0ODQ5NGRlZTVlZTRjMDk3YWRhNGIyYTQyZTA1M2RhYTIzYjlmNWY1MTUxZDgKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBU2NvcGVPdXRzaWRlVGhlUm9vdElzUmVmdXNlZE5vdEZhbGxlbkJhY2tUbwlhYjc3ZjEwODdmMDg1YWU0YjdmNzIyY2Y3Y2Q2NjMwMmY2ZGU3NzJlNzZhNzgyNWVjZmY1NzJlMzY3ZDZjM2M1CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVNoZWxsSW5qZWN0ZWRTY29wZVN0aWxsRmFpbHMJMWFlYTYwNzMxOGYzNGExYjA4YmY1NDI1MzRiY2E2NTU2ZjFjN2YwN2ZhZmVjZWY0ZjFmNjQ3YTkxYjU4ZWMwZApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFXaGl0ZXNwYWNlT25seUNoZWNrSXNOb3REZWNsYXJlZAlhOTA3ZWU0MzlmMTRkYzJlNDhkMmU0MTM4MDEwODc5MTUzM2E0YTRmMDhlMjRlOGQwZWYyZDRiMzU3OTFlYTQ0CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVdoaXRlc3BhY2VPbmx5U2NvcGVkQ2hlY2tJc05vdERlY2xhcmVkCWY2YmYwYzU0OWQ2NDZhNzUwNzUwZmZhNTEwYzk4ZjY5MzUxNThmODViMzcyMjYwZTg4NjI1MzViOWI0YWE1MWIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBbkluUm9vdE1pc3NJc1JlZnVzZWROb3RBU2lsZW50UGFzcwkwMTc0YTk2NjE1OGYyYThkNmE5OTIxMWQ3MGVlMWIxMGUwYzBmODA4NTc4YzExZmE4NzQ2MzA3MDA1YjQ5Y2QxCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QW5PdmVybGFyZ2VUaW1lb3V0SXNDbGFtcGVkTm90T3ZlcmZsb3dlZAk2MjI5MzU0ZDk1MzJmZDBhYTU2NWIzMGI0NTRkZDc3ZWRiZmQ1OWQ1YmNkNjZiNzRjY2QxOTlkZDIyMzg1YmI0CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNPbmx5V2l0aE5vUGF0aHNTdGlsbEZhbGxzQmFjawkwMThiMTY4NTRlMTU3ZmQzN2E4ZTgwY2E4YmIzZmU1OWQ2NTM2OWQ1ZGNhOTdhMjJhZGQ0NGQzNjU5NmUyNGViCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNQbGFjZWhvbGRlcglmYTRiYTBhNmQ1NDM3MzYwMGYwMDI0NzliMDk4YTc1MTEyMTlhMjU0MDhjNmE2OTg0MTU0ZGZlMmUwNTkzN2MxCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0RmlsZXNQbGFjZWhvbGRlckRvZXNOb3ROZWVkUGFja2FnZXMJODFjZmQ0MzU3NTM0ODBlOTZkZGUxNTIxN2JlZTJhYTg1OTA2MDllZDYzZTBlZjQ3ODA4ZjQ1NTkwNGY1NTY3ZApib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRJbmZlcnNHb0NoZWNrQnV0U2F5c1NvCTA5OWU2YmZmM2ZlNzdhMzM5MDc1M2M1ZWZiYzdlMjE1NWQzYjY0MDY5YTk5YjJlMmRlMWVlZmRkMzllYjMzMWEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RMb2FkT25BQmFyZURpcmVjdG9yeUhhc05vQ2hlY2sJZjU1ZTJiM2ZlZWVmMzc4ZmJjOTI1OTRlZmM0ZmI1YmZkNjMzNGIzNzM1OTg1M2IzNTNjM2Y4Mjg1Y2MxNGU2Mwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRQcmVmZXJzVGhlRGVjbGFyZWRDaGVjawkyNWQxZjdjZTU5ZGU1M2Y4ZGJjYzg2Zjg4ZWJiNTVlNWNlMzRlNGU2ZWUyOTcxZmJhZDgzZTBmZTZjZTI5MDllCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0TWl4ZWRQbGFjZWhvbGRlcnNGYWxsQmFja1doZW5Vbm1hcHBlZAkxYWM3NzM4Yzk3OWM3MDgxYjcxYTIyODkwNjI2MzA0Njk4NmVjZjUyZDNiOWM2MTU3MmVkZDhjOTdhZDNkMzM3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0Tm9DaGVja0lzTm90QVBhc3MJYTNiMzNkYWU0MjdlYTZjNWE2YmIwODk5NDA4MjJkZTVhYzhiZTk0MmFlYjAyMTkwYmE1YTg0NjAzYjA1OGFiNgpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFBhY2thZ2VzT25seU5vbkdvU3RpbGxGYWxsc0JhY2sJMWJhMGIyNTMwYmZjOTNmNDg2NzEwZTkwMmU2Mjk2NGEzNzhkZTYxZTAzYzdkZGQ2YjRlZjhmN2E5ZjQ1ODM5ZQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFJ1blBhc3Nlcwk4ZWI3MGZmOGY4OTFjOTAzY2JlNmQ4YjkyMDI1Y2M5OTc0NzU1NmE5MmJkY2EzMGM4YzRhNjEwOGY1ZTU4ZmE3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0UnVuUmVwb3J0c1RoZVJlYWxFeGl0Q29kZQlhMGE0YzBmNjEwZDk1Y2Y0NDFjMDBkODViZDgyNGE4NWQ1Mzk2NjBhYjI3ZDYzYjkwZDI0Njg0MWMxNTdhZGQ3CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0U2NvcGVEZXJpdmF0aW9uCTY0ZThhYmY3OGM1ZTM4MzU2NmFmZDFlNTI5YjNlMGFjYzVhZWI2ZmVlYzhmMWMyOGU0MDdhN2EzNTkyNDNlMzQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RTdWJzdGl0dXRlZFBhdGhzQXJlT25lU2hlbGxBcmd1bWVudEVhY2gJZWMzZWRlZWU5NTViZGJjMDdjYThkZGJlYzI1NGEyNTgzMDMxYzMyZDRiMDIwZjY2NjM3ODUyMWQ0OWNkNjE1Ngpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFRhaWxBbm5vdW5jZXNXaGF0SXRMZWZ0T3V0CTQ3NDk0M2RjNjE1MzlhMmViNDVhOGM3OTMzOGI5ZTk2OGY4YmNkMmViMDkwZTE3ZTcwYjg3ZGM4Y2JjODZkODUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RUaW1lb3V0SXNSZXBvcnRlZEFzQUZhaWx1cmVOb3RBUGFzcwlmZjdkMDYwYzg3NWVkN2U1ZDlmNmNiYjEzMjIyY2FiY2U0MjI5ZGY3MDZmMmMwY2MyZDBhNTk3ZTk0NGFlMmFl
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.181s
  === RUN   TestTailAnnouncesWhatItLeftOut
  --- PASS: TestTailAnnouncesWhatItLeftOut (0.02s)
  === RUN   TestAPaddedCheckIsStillTheDeclaredCheck
  --- PASS: TestAPaddedCheckIsStillTheDeclaredCheck (0.00s)
  === RUN   TestAPaddedScopedCheckIsStillDeclared
  --- PASS: TestAPaddedScopedCheckIsStillDeclared (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	0.125s
  ```
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6350300fa3fe07d431d62e1b19d6d0ee92dfb298377da8a1eb63b5a5a18eccbf · ms:396
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6350300fa3fe07d431d62e1b19d6d0ee92dfb298377da8a1eb63b5a5a18eccbf · ms:1733
- 2026-09-25 · 77a408a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6350300fa3fe07d431d62e1b19d6d0ee92dfb298377da8a1eb63b5a5a18eccbf · ms:627
- 2026-09-25 · 1357828* · exit 0 · `set -o pipefail …` · acceptance-sha256:6350300fa3fe07d431d62e1b19d6d0ee92dfb298377da8a1eb63b5a5a18eccbf · ms:1127
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:88e650968c914aa273c7f9343ce4609bb083545909e0a4720fd56743ee7cc5a9 · ms:557
- 2026-09-25 · fb7d18d* · exit 0 · `set -o pipefail …` · acceptance-sha256:88e650968c914aa273c7f9343ce4609bb083545909e0a4720fd56743ee7cc5a9 · ms:309

## Mutation Log
(empty until execute)
- 2026-09-25 · 77a408a* · mutant killed · exit 1 · `internal/check/check_test.go` · needSh skips where sh IS present, so the tail test never runs and its PASS line is missing · acceptance-sha256:6350300fa3fe07d431d62e1b19d6d0ee92dfb298377da8a1eb63b5a5a18eccbf · covers:a test that needs sh says so
- 2026-09-25 · fb7d18d* · mutant killed · exit 1 · `internal/check/check_test.go` · needSh skips where sh IS present, so the tail test never runs and its PASS line is missing · acceptance-sha256:88e650968c914aa273c7f9343ce4609bb083545909e0a4720fd56743ee7cc5a9 · covers:a test that needs sh says so

## Invariants

- No assertion is weakened; an assertion that reads `x ` keeps the file and its skip.

## Risks

- None beyond the shard's own flakiness.

## Out of Scope

- `-race` without cgo and `contract.sh` without `jq` on Windows (permanent: fact: CI's Windows job runs `go test -race` with its own toolchain and contract.sh runs on Linux only; citation: file `.github/workflows/ci.yml:95`)

## Stop Condition

Stop if a production file would change.
