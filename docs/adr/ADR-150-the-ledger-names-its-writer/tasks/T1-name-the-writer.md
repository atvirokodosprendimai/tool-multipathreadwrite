# Task ADR-150-T1: the header names its writer, v4 stays current, a newer ledger is left alone and named

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `seen.SetWriter`, `seen.Newer`, `seen.NewerNotice` in `internal/seen/seen.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the header names its writer`, `a v4 ledger is still current`, `an older ledger is discarded`, `a newer ledger licenses nothing`, `a newer ledger is not overwritten`, `the CLI names the newer writer`, `a refused write names the newer ledger`, `the docs name the newer ledger`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | the header with its writer; the format read as a number in `Load`, `IsStale` and the damage scan; `Record` leaves a newer ledger alone; `NewerNotice` |
| `internal/seen/writer150_test.go` | add | the tests |
| `internal/seen/seen_test.go` | edit | `TestAV4LedgerIsAcceptedAndTheBumpIsDeliberate` keeps v4 accepted; the save half expects v5 |
| `cmd/mrw/main.go` | edit | `seen.SetWriter` in the root command's `Before` (atomic: in-process tests run commands concurrently, and `go test -race` caught a plain variable); `Before` prints the newer line instead of the stale one |
| `cmd/mrw/writer150_test.go` | add | the CLI line and the write refusal |
| `internal/writer/flow.go` | edit | a refusal while the ledger is newer carries the sentence |
| `scripts/contract.sh` | edit | §254 drives the built binary |
| `AGENTS.md`, `README.md` | edit | one sentence each |
| `docs/adr/BACKLOG.md` | edit | the survey entry is marked taken |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestALedgerNamesItsWriter` (a `Record` writes `#mrw-seen v5 <writer>` with the writer set, `unknown` without; a v4 ledger written byte for byte is parsed and not stale; a v3 and a CRLF one are discarded and stale); `TestANewerLedgerIsNeitherUsedNorOverwritten` (a `#mrw-seen v6 v9.9.9` ledger whose entry would license the line licenses nothing, is not stale, is reported newer with its writer, and its bytes are unchanged after `Record`); `TestTheCLINamesANewerLedgersWriter` (`mrw read` exits 0 and prints the newer line once and not the stale one; a write to a line the v6 entry covers is refused, exit 1, and the refusal names the newer ledger; the ledger's bytes are unchanged).
2. [S2] The header and the numeric format in `internal/seen`. [proof: mutation]
3. [S3] `Record` leaves a newer ledger alone. [proof: mutation]
4. [S4] `cmd/mrw` sets the writer and prints the newer line. [proof: mutation]
5. [S5] The write refusal carries the sentence. [proof: mutation]
6. [S6] AGENTS.md, README.md, BACKLOG. [proof: mutation]
7. [S7] Contract §254: a v6 ledger planted in the state directory survives a read and a write byte for byte, the line names its writer, and the pair — a v4 ledger — licenses its write. [proof: human: the executor runs ./scripts/contract.sh once and reads the §254 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/seen/ ./cmd/mrw/ -count=1 -timeout 300s -run 'TestALedgerNamesItsWriter|TestANewerLedgerIsNeitherUsedNorOverwritten|TestTheCLINamesANewerLedgersWriter|TestAV4LedgerIsAcceptedAndTheBumpIsDeliberate|TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestALedgerNamesItsWriter \(' "$out" \
  && grep -qE '^--- PASS: TestANewerLedgerIsNeitherUsedNorOverwritten \(' "$out" \
  && grep -qE '^--- PASS: TestTheCLINamesANewerLedgersWriter \(' "$out" \
  && grep -qE '^--- PASS: TestAV4LedgerIsAcceptedAndTheBumpIsDeliberate \(' "$out" \
  && grep -qE '^--- PASS: TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions \(' "$out" \
  && grep -q '^# 254\. ' scripts/contract.sh \
  && grep -q 'written by a newer mrw' AGENTS.md \
  && grep -q 'written by a newer mrw' README.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestALedgerNamesItsWriter` | `internal/seen/writer150_test.go` | the v5 header with its writer; v4 current; v3 and CRLF discarded | none | S1, S2 |
| `TestANewerLedgerIsNeitherUsedNorOverwritten` | `internal/seen/writer150_test.go` | a v6 ledger licenses nothing, is reported newer with its writer, and keeps its bytes | none | S1, S2, S3 |
| `TestTheCLINamesANewerLedgersWriter` | `cmd/mrw/writer150_test.go` | the CLI line, the refused write naming the newer ledger, the bytes kept | none | S1, S4, S5 |
| `TestAV4LedgerIsAcceptedAndTheBumpIsDeliberate` | `internal/seen/seen_test.go` | ADR-118's v4 still parsed; v3 still stale | none | — |
| `TestTheStaleNoticeIsOneSentenceAndSilentOnVersionStatsInstructions` | `cmd/mrw/quiet136_test.go` | ADR-136's notice unchanged for an older ledger | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `seen.SetWriter`, `seen.Newer`, `seen.NewerNotice`, the numeric header |
| 2 — something selects it | `Load`, `IsStale`, `Record` on every read and write; `Before` on every command that reads the ledger; contract §254 drives the built binary |
| 3 — the caller can discover it | the stderr line; the write refusal; AGENTS.md and README |
| 4 — it is used | nothing measures this; built because five survey sessions reported two binaries discarding each other's ledger (2026-10-09) |

## Mutation Log
(empty until execute)
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `internal/seen/seen.go` · a v4 ledger is no longer current · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `internal/seen/seen.go` · a newer ledger is parsed and licenses its entries · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `internal/seen/seen.go` · a v4 ledger is no longer current · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:a v4 ledger is still current
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `internal/seen/seen.go` · a newer ledger is parsed and licenses its entries · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:a newer ledger licenses nothing
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `internal/seen/seen.go` · an older format is parsed · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:an older ledger is discarded
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `internal/seen/seen.go` · a read saves over a newer ledger · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:a newer ledger is not overwritten
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never tells the ledger its writer · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:the header names its writer
- 2026-10-14 · 2f95832* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · the CLI calls a newer ledger older · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:the CLI names the newer writer
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `internal/writer/flow.go` · a refused write does not name the newer ledger · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:a refused write names the newer ledger
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `README.md` · README stops naming the newer ledger · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:the docs name the newer ledger
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI says nothing of a newer ledger · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:the CLI names the newer writer
- 2026-10-14 · 2f95832* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never tells the ledger its writer (SetWriter form) · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · covers:the header names its writer

## Verification Log
(empty until execute)
- 2026-10-14 · 2f95832* · exit 1 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:415 · test-lock-sha256:946b957f01683eb8a4d01900239413c98cf8612af42629099fba298bf3b0c406 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9xdWlldDEzNl90ZXN0LmdvCVRlc3RUaGVTdGFsZU5vdGljZUlzT25lU2VudGVuY2VBbmRTaWxlbnRPblZlcnNpb25TdGF0c0luc3RydWN0aW9ucwkwYTdjYWJhNzFiN2M0YTJhMzEwNWNlZTIwN2YxYWE5M2I3YjRhYjFiYjk1ZmZlZGY3ZDg0ZGNiNzQ2ZGRjMmQwCmJvZHkJY21kL21ydy93cml0ZXIxNTBfdGVzdC5nbwlUZXN0VGhlQ0xJTmFtZXNBTmV3ZXJMZWRnZXJzV3JpdGVyCTRkMGQ2ZDk2MTM1NTEzYmU3ODNmYzU0MjBiMTM0OGIzZjkyZTIzYmU1YzUzNjVkZjU1OWY3ZDBiZTlmMjg5ZjAKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZGdlclRoaXNCdWlsZFdyb3RlTG9hZHNCYWNrCWQxZDZiODk0ZTUxZWZmYWYwMDYyNDVkN2Y3Y2VhMTU2OGEyMzZjMmU0YTViYWI1Yzk3NjQzYjNjNjE2N2EwMDAKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZ2FjeVBhdGhDb250YWluaW5nQURvdWJsZVNwYWNlSXNPbmVQYXRoCWNiN2I1Y2NkNDRjMzE5NGFmODk1YzM4ZTA5ZDYyYjgwNDNkZmYxMWQ3MmUwYjUyMjc1NTk4NjJhYmJhMmFmMTUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVByZVYySW5UcmVlTGVkZ2VySXNEaXNjYXJkZWROb3RUcnVzdGVkCTk2MGNmMjllYmY2ZmEyZDJiOThlYmFmYWU2YjM1NDg3NjgzYzAzNzY3NWM1NGY0OGIxMThiZjc3NDkwZWFmZWUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVY0TGVkZ2VySXNBY2NlcHRlZEFuZFRoZUJ1bXBJc0RlbGliZXJhdGUJOGI4ZTMzZjkyODAxZDZiYmM2NDFhMDk4ZDJiMTFmODY5MjhlMTI3ZjEzNjUzYzEzNzE0MDA5MGFlZDA2YzQ2ZQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBV3JpdHRlbkZpbGVJc0xpY2Vuc2VkQnV0Tm90U2hvd24JZWY1Mjk4YzQxYjhmMDI2NzYyYTQxNDUzNzM5ZTI3MjgzOGRlNWJiNTI3OTgxNTMwOWE5NmUxMTNhNzcwNDc0Ywpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RDb25jdXJyZW50UmVjb3Jkc0tlZXBFdmVyeVBhdGgJZWI5YTZjOTUwMTJlYzYzNWE0ZTc1MTliNzVjNzYxMzRjYjk3YzAwNGNlZDBmMjEyZjdiODlhMjNiNDg2OTUzYQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RMZWRnZXJJc05vdFdyaXR0ZW5JbnRvVGhlUm9vdAk0NDFlOWY3OGE0YTkzMmJmNjJiOTZiMjIwNjIyNzdhMDI3ZTg3YjdkYzI3OGVlYThlNGQ4YzI4NmE4YTE4NGE2CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdExlZ2FjeUluVHJlZUxlZGdlcklzU3RpbGxSZWFkCWM0ZGE4Yzc0MzJkYTNiMzUzNDdmYzhlMzAyOGEzYjE2ODg0OWZjNTIzOWY5OTRhNTMwYjllODE4Mzg1OGM4YzUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TWFpbgkzODNhMzc0MGViNWIxMzcyNDA0M2U2ZjdkYTFiOGFiYmJlNzE1Y2E4Y2E0YzUyMWMzYzc2NWQ5MDFjOTgyZmQ0CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdE1pc3NpbmdMZWRnZXJJc0VtcHR5Tm90QW5FcnJvcgk1ODU3M2Q4NGZjYWU1ZTUwY2ExMDIzMzA2OGQzNjI2ZGEzYTZlMjFmMzljNWY5YzE3MjI2ZDI1ZDkzZWMwNWU4CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE1lcmdlc1JhdGhlclRoYW5SZXBsYWNlcwkyMzdiODIwNTgzZmY5YmJiODZkOGJmM2I0NWY3OTVjZjY1YzJlNGM0MzI3MTRiMGI5YmQ4YTNjYTg2MDEyMWFjCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE92ZXJ3cml0ZXNUaGVTYW1lUGF0aAk1YmM2MDA2NGNkZjMxNmU2MTJkNzI4ODA5NDE3OTc0ZTRlMDU5MzQ3NGM2MGJmYjViMDJlNjA2M2VjMTJmOWE3CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJvdW5kVHJpcAkxZTE4MjhmMDJlMjM4ZWNmYzA4ZjBmNTJhODZiNmRkMGQ4ZWI4MDg2NjY1OWJhNDAwYWJmZWUzZjY1NzgwMjJmCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFNIQUlzU3RhYmxlCTg5OGY5YzZjMDU2MDE3M2RjNDFiNmRjNTMyYTk2NzA1NjFjYTIyZDAxMTdmOGNiMDE1MzVkZjViNzk0MWVhMGQKYm9keQlpbnRlcm5hbC9zZWVuL3dyaXRlcjE1MF90ZXN0LmdvCVRlc3RBTGVkZ2VyTmFtZXNJdHNXcml0ZXIJZTUyMTUwZTIwZDQ0NmNkYjQ3MDEwNDkxOWFjZDAzMzYxYTUwODBiMzNhNTYxYTUxZjI0N2U5OWVjMGZmYjZkZQpib2R5CWludGVybmFsL3NlZW4vd3JpdGVyMTUwX3Rlc3QuZ28JVGVzdEFOZXdlckxlZGdlcklzTmVpdGhlclVzZWROb3JPdmVyd3JpdHRlbgk1YTAzOTkyN2ZlNzhkZjkxMjYzYjNjNTEyYWU0ZDRhMTg3NGQ1OTRlODRiZGExYWJmZTc4NTRiZjA5NWU2YWU5
  ```
  --- last 10 line(s) of stdout (of 27 after folding 27 raw)
      writer150_test.go:55: a write the newer ledger would license exited 0, receipt "ok   f.txt 1 replace  -1 +1\nwrote f.txt  2L -> 2L  sha c78a5ec2\n1 hunk(s), 1 file(s), 0 failed, 0 advisories — applied\n" stderr ""; want it refused naming the newer ledger
      writer150_test.go:58: the refused write changed the file: "ONE\ntwo\n"
      writer150_test.go:61: the newer ledger was written over: "#mrw-seen v4\nc78a5ec2c28be893afb6225ef05c556ef289bb4b6b76e7fc358c29e791179123  w  f.txt\n"
      writer150_test.go:68: the pair: a v4 ledger's entry did not license the write, exit 1: FAIL f.txt 1 replace (plan line 1): f.txt changed since mrw last saw it (recorded c3f9c8c2, now c78a5ec2): re-read it before editing, or pass --force to overwrite blind
          1 hunk(s), 1 file(s), 1 failed, 0 advisories — NOTHING WRITTEN
          1 hunk(s) failed — nothing was written
  --- FAIL: TestTheCLINamesANewerLedgersWriter (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.132s
  FAIL
  ```
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1814
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1825
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1213
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1371
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1338
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1301
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1403
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1631
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1488
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1478
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:1553
- 2026-10-14 · 2f95832* · exit 0 · `set -o pipefail …` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:2081
- 2026-10-14 · human-observed · observed by the executor 2026-10-14: S7 ./scripts/contract.sh exit 0 ('contract holds'); all six §254 rows PASS — the header names its format and writer, a read against a v6 ledger exits 0 naming the newer writer once and leaves it untouched, a write its entry would license is refused naming it and writes nothing, and the pair (v4 header) licenses the write
- 2026-10-14 · 2f95832* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6ad570277ad985ab3a6e856ea159333c92961c4ef2cb3088c3d4f87eda04d157 · ms:0 · test-lock-sha256:b5a59217040ee99fae10ce62f9a6ba4006044d44e7ca279f22a80b16f6127bb7 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9xdWlldDEzNl90ZXN0LmdvCVRlc3RUaGVTdGFsZU5vdGljZUlzT25lU2VudGVuY2VBbmRTaWxlbnRPblZlcnNpb25TdGF0c0luc3RydWN0aW9ucwkwYTdjYWJhNzFiN2M0YTJhMzEwNWNlZTIwN2YxYWE5M2I3YjRhYjFiYjk1ZmZlZGY3ZDg0ZGNiNzQ2ZGRjMmQwCmJvZHkJY21kL21ydy93cml0ZXIxNTBfdGVzdC5nbwlUZXN0VGhlQ0xJTmFtZXNBTmV3ZXJMZWRnZXJzV3JpdGVyCWNiNjM2NjFjZTJkYmFmN2MyNzk2ZThhN2I4ZGM5ZmUwNjQzNDVlNTEyMjkxNGQyM2I2ODc4YTVhZDY5ZjM5ZmMKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZGdlclRoaXNCdWlsZFdyb3RlTG9hZHNCYWNrCWQxZDZiODk0ZTUxZWZmYWYwMDYyNDVkN2Y3Y2VhMTU2OGEyMzZjMmU0YTViYWI1Yzk3NjQzYjNjNjE2N2EwMDAKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QUxlZ2FjeVBhdGhDb250YWluaW5nQURvdWJsZVNwYWNlSXNPbmVQYXRoCWNiN2I1Y2NkNDRjMzE5NGFmODk1YzM4ZTA5ZDYyYjgwNDNkZmYxMWQ3MmUwYjUyMjc1NTk4NjJhYmJhMmFmMTUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVByZVYySW5UcmVlTGVkZ2VySXNEaXNjYXJkZWROb3RUcnVzdGVkCTk2MGNmMjllYmY2ZmEyZDJiOThlYmFmYWU2YjM1NDg3NjgzYzAzNzY3NWM1NGY0OGIxMThiZjc3NDkwZWFmZWUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0QVY0TGVkZ2VySXNBY2NlcHRlZEFuZFRoZUJ1bXBJc0RlbGliZXJhdGUJOGI4ZTMzZjkyODAxZDZiYmM2NDFhMDk4ZDJiMTFmODY5MjhlMTI3ZjEzNjUzYzEzNzE0MDA5MGFlZDA2YzQ2ZQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RBV3JpdHRlbkZpbGVJc0xpY2Vuc2VkQnV0Tm90U2hvd24JZWY1Mjk4YzQxYjhmMDI2NzYyYTQxNDUzNzM5ZTI3MjgzOGRlNWJiNTI3OTgxNTMwOWE5NmUxMTNhNzcwNDc0Ywpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RDb25jdXJyZW50UmVjb3Jkc0tlZXBFdmVyeVBhdGgJZWI5YTZjOTUwMTJlYzYzNWE0ZTc1MTliNzVjNzYxMzRjYjk3YzAwNGNlZDBmMjEyZjdiODlhMjNiNDg2OTUzYQpib2R5CWludGVybmFsL3NlZW4vc2Vlbl90ZXN0LmdvCVRlc3RMZWRnZXJJc05vdFdyaXR0ZW5JbnRvVGhlUm9vdAk0NDFlOWY3OGE0YTkzMmJmNjJiOTZiMjIwNjIyNzdhMDI3ZTg3YjdkYzI3OGVlYThlNGQ4YzI4NmE4YTE4NGE2CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdExlZ2FjeUluVHJlZUxlZGdlcklzU3RpbGxSZWFkCWM0ZGE4Yzc0MzJkYTNiMzUzNDdmYzhlMzAyOGEzYjE2ODg0OWZjNTIzOWY5OTRhNTMwYjllODE4Mzg1OGM4YzUKYm9keQlpbnRlcm5hbC9zZWVuL3NlZW5fdGVzdC5nbwlUZXN0TWFpbgkzODNhMzc0MGViNWIxMzcyNDA0M2U2ZjdkYTFiOGFiYmJlNzE1Y2E4Y2E0YzUyMWMzYzc2NWQ5MDFjOTgyZmQ0CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdE1pc3NpbmdMZWRnZXJJc0VtcHR5Tm90QW5FcnJvcgk1ODU3M2Q4NGZjYWU1ZTUwY2ExMDIzMzA2OGQzNjI2ZGEzYTZlMjFmMzljNWY5YzE3MjI2ZDI1ZDkzZWMwNWU4CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE1lcmdlc1JhdGhlclRoYW5SZXBsYWNlcwkyMzdiODIwNTgzZmY5YmJiODZkOGJmM2I0NWY3OTVjZjY1YzJlNGM0MzI3MTRiMGI5YmQ4YTNjYTg2MDEyMWFjCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJlY29yZE92ZXJ3cml0ZXNUaGVTYW1lUGF0aAk1YmM2MDA2NGNkZjMxNmU2MTJkNzI4ODA5NDE3OTc0ZTRlMDU5MzQ3NGM2MGJmYjViMDJlNjA2M2VjMTJmOWE3CmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFJvdW5kVHJpcAkxZTE4MjhmMDJlMjM4ZWNmYzA4ZjBmNTJhODZiNmRkMGQ4ZWI4MDg2NjY1OWJhNDAwYWJmZWUzZjY1NzgwMjJmCmJvZHkJaW50ZXJuYWwvc2Vlbi9zZWVuX3Rlc3QuZ28JVGVzdFNIQUlzU3RhYmxlCTg5OGY5YzZjMDU2MDE3M2RjNDFiNmRjNTMyYTk2NzA1NjFjYTIyZDAxMTdmOGNiMDE1MzVkZjViNzk0MWVhMGQKYm9keQlpbnRlcm5hbC9zZWVuL3dyaXRlcjE1MF90ZXN0LmdvCVRlc3RBTGVkZ2VyTmFtZXNJdHNXcml0ZXIJNDA3ZWQ0MDY4M2RjOWE1NGYyODZiMDRjYTliN2Q1ZWM2OWYwNGQ2OGM4ZjZlYWQ0YWNiMDYzMzUyYTUwMWJiZApib2R5CWludGVybmFsL3NlZW4vd3JpdGVyMTUwX3Rlc3QuZ28JVGVzdEFOZXdlckxlZGdlcklzTmVpdGhlclVzZWROb3JPdmVyd3JpdHRlbglkNTQ1OWFkNDU0NTEwMjRlYmExOTlmOGU3MGQwYTFmZWFkMDM5NjUxMTg3NjUyNjllZTk3OTg0YWZhYjRiYzA3 · test-lock-kind:replace

## Invariants

- ADR-136's stale notice keeps its words for an older ledger; ADR-144's damage notice keeps its rule for a current one.
- A ledger never licenses a line from a format this mrw does not know.

## Risks

- None beyond the record's.

## Out of Scope

- A notice on the MCP read path (permanent: boundary: the MCP server has no stderr; the write refusal carries it)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
