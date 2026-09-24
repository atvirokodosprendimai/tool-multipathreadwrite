# Task ADR-065-T1: one splitter numbers lines for read, write, `--grep` and MCP paging; ast-grep refuses CR-only hits; contract §120

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `lines.Split`; served lines match the write engine's; CR-only ast-grep hits reported; §120; campaign expectation
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a CR-only file is served as its lines`, `a CRLF line is served without its terminator`, `read still hashes the raw bytes`, `MCP paging covers the same lines`, `an ast-grep row is not served on a CR-only file`, `the binary writes the line it served`

## Goal

`internal/lines.Split` holds ADR-005 §3's rule, moved unchanged from `internal/apply/apply.go:1533-1556`. Its callers:
- `apply.readLines` calls it;
- `read.split` (`internal/read/read.go:699`) takes its lines from it and still hashes the raw bytes; `--grep` follows through `walk.go:207`;
- MCP `countFileLines` (`internal/mcp/tools.go:1043`) counts `len(lines.Split(…))`;
- `read.AstGrep` (`internal/read/astgrep.go:103`) reports a hit in a file whose terminator is `\r` as a `Problem` naming the file, instead of serving ast-grep's `\n`-counted row.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/lines/lines.go` | new | `Split`, package comment |
| `internal/lines/lines_test.go` | new | `TestSplitOfAnEmptyFileHasNoLines` |
| `internal/apply/apply.go` | edit | `readLines` calls `lines.Split`; `eolOf` removed |
| `internal/read/read.go` | edit | `split` takes lines from `lines.Split`, hash unchanged |
| `internal/read/astgrep.go` | edit | CR-only hit becomes a `Problem` |
| `internal/read/lines_agree_test.go` | new | read and ast-grep tests below |
| `internal/mcp/tools.go` | edit | `countFileLines`; its comment |
| `internal/mcp/lines_agree_test.go` | new | the paging test below |
| `cmd/mrw/lines_agree_test.go` | new | read-then-write tests through the commands |
| `scripts/contract.sh` | edit | §120 |
| `docs/adr/BACKLOG.md` | edit | ADR-065 row; the ast-grep offset-translation deferral |

## Ordered Steps

1. [S1] Write the RED tests and confirm each fails on `main` on an assertion. The package stub is created first, so no test fails to build. [proof: mutation]
   - `TestACROnlyFileIsServedAsTheLinesAWriteAddresses`: `read.Run` on `one\rtwo\rthree\r` prints `3L`, and `cr.txt:2` serves `two`.
   - `TestACRLFLineIsServedWithoutItsTerminator`: `crlf.txt:/one$/` serves line 1 with content `one`.
   - `TestGrepMatchesADollarAnchoredLineInACRLFFile`: `--grep 'two$'` serves line 2.
   - `TestAWholeReadOfACROnlyFileLicensesOnlyTheLinesItServed`: after a whole read of a CR-only file, `@@ cr.txt 1 replace` replaces exactly the served line 1 (`one`), and the result is `X\rtwo\rthree\r`.
   - `TestPagingACROnlyFileToExhaustionCoversEveryLine`: a CR-only file of N lines, N fixed by the fixture and chosen to need more than one page under the default ceiling, read through MCP `mrw_read`. `next_read` is used only to advance, with at most N pages. The oracle is the fixture: the numbered lines served are exactly 1..N with the fixture's contents, and no line is served twice. An unacknowledged page's lines stay unwritable; after `ack`, a line on it is writable.
   - `TestAstGrepReportsAHitInACROnlyFile`: with a fake ast-grep reporting row 2 in a CR-only file, `AstGrep` returns a Problem naming the file and serves no range for it.
   - `TestSplitOfAnEmptyFileHasNoLines`.
   Guards that already hold, run beside them and required to stay green, each proved by a mutation in S2 rather than a red run:
   - `TestACRLFFileReadWholeThenWrittenKeepsItsSha`: read a pure CRLF file whole, then write line 2. The write applies, because the raw-bytes sha agrees with the ledger, and every terminator is still `\r\n`.
   - `TestAMixedEndingFileKeepsTheStrayCarriageReturnOnBothSides`;
   - `TestAReadOfLineOneDoesNotLicenseLineTwoOfACROnlyFile`;
   - `TestACRTerminatedFileHasAddressableLines` and `TestACRLFFileKeepsItsLineEndings` (`internal/adversarial/filesystem_test.go`).
2. [S2] Create `internal/lines`, rewire the four sites, and confirm GREEN. [proof: mutation]
   Mutants:
   - the CR-only branch returns `"\n"`: kills the CR-only read and whole-read tests;
   - the CRLF test `==` becomes `>0`: kills the mixed-ending guard;
   - `split` reverts to breaking at `"\n"`: kills the CR-only and CRLF read tests;
   - `countFileLines` reverts to `bufio.Scanner`: kills the paging test;
   - `split` hashes lines rejoined with `"\n"`: kills the CRLF read-then-write test;
   - the ast-grep check removed: kills the ast-grep test.
3. [S3] §120, the next free number after ADR-066's §119. [proof: mutation]
   - Good: CR-only, `read cr.txt:2` then `@@ cr.txt 2 replace` exits 0 with bytes `one\rTWO\rthree\r`. CRLF, `read crlf.txt` then a line-1 write keeps every terminator `\r\n`.
   - Must-fail: `read crlf.txt:'/one\r$/'` exits 1. A whole read of `cr.txt` prints `3L`: grep that the header does NOT say `1L`.
   - Confirm RED against v1.22.3 in a mini-harness, then GREEN in the full `./scripts/contract.sh`.
4. [S4] BACKLOG rows; `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 120\. ' scripts/contract.sh \
  && go test ./internal/lines/ ./internal/read/ ./internal/mcp/ ./cmd/mrw/ ./internal/adversarial/ -count=1 -v \
    -run 'TestSplitOfAnEmptyFileHasNoLines|TestACROnlyFileIsServedAsTheLinesAWriteAddresses|TestACRLFLineIsServedWithoutItsTerminator|TestGrepMatchesADollarAnchoredLineInACRLFFile|TestAWholeReadOfACROnlyFileLicensesOnlyTheLinesItServed|TestACRLFFileReadWholeThenWrittenKeepsItsSha|TestPagingACROnlyFileToExhaustionCoversEveryLine|TestAstGrepReportsAHitInACROnlyFile|TestAMixedEndingFileKeepsTheStrayCarriageReturnOnBothSides|TestAReadOfLineOneDoesNotLicenseLineTwoOfACROnlyFile|TestACRTerminatedFileHasAddressableLines|TestACRLFFileKeepsItsLineEndings' 2>&1 | tee /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestSplitOfAnEmptyFileHasNoLines ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestACROnlyFileIsServedAsTheLinesAWriteAddresses ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestACRLFLineIsServedWithoutItsTerminator ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestGrepMatchesADollarAnchoredLineInACRLFFile ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestAWholeReadOfACROnlyFileLicensesOnlyTheLinesItServed ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestACRLFFileReadWholeThenWrittenKeepsItsSha ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestPagingACROnlyFileToExhaustionCoversEveryLine ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestAstGrepReportsAHitInACROnlyFile ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestAMixedEndingFileKeepsTheStrayCarriageReturnOnBothSides ' /tmp/adr065-t1.out \
  && grep -q '^--- PASS: TestACRTerminatedFileHasAddressableLines ' /tmp/adr065-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr065-t1.out \
  && ./scripts/contract.sh > /tmp/adr065-t1-contract.out 2>&1 \
  && grep -q '^  PASS  a CR-only line read is the line a write addresses' /tmp/adr065-t1-contract.out \
  && grep -q '^  PASS  a CRLF file read whole is written without changing its endings' /tmp/adr065-t1-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/seen internal/check internal/state internal/guide \
  && [ "$(grep -c require go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/lines internal/apply internal/read internal/mcp cmd/mrw)" ] \
  && go vet ./internal/lines/ ./internal/apply/ ./internal/read/ ./internal/mcp/ ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestSplitOfAnEmptyFileHasNoLines` | `internal/lines/lines_test.go` | empty input has no lines | — | S1, S2 |
| `TestACROnlyFileIsServedAsTheLinesAWriteAddresses` | `internal/read/lines_agree_test.go` | a CR-only file reads as 3 lines | — | S1, S2 |
| `TestACRLFLineIsServedWithoutItsTerminator` | `internal/read/lines_agree_test.go` | `/one$/` resolves on a CRLF file | — | S1, S2 |
| `TestGrepMatchesADollarAnchoredLineInACRLFFile` | `internal/read/lines_agree_test.go` | `--grep 'two$'` serves line 2 | — | S1, S2 |
| `TestAMixedEndingFileKeepsTheStrayCarriageReturnOnBothSides` | `internal/read/lines_agree_test.go` | guard: mixed file keeps `\r` in content | — | S1 |
| `TestAstGrepReportsAHitInACROnlyFile` | `internal/read/lines_agree_test.go` | a CR-only hit is a Problem, no range served | — | S1, S2 |
| `TestPagingACROnlyFileToExhaustionCoversEveryLine` | `internal/mcp/lines_agree_test.go` | MCP paging covers exactly 1..N; only acknowledged lines are writable | — | S1, S2 |
| `TestAWholeReadOfACROnlyFileLicensesOnlyTheLinesItServed` | `cmd/mrw/lines_agree_test.go` | a line-1 write replaces the served line 1 only | — | S1, S2 |
| `TestACRLFFileReadWholeThenWrittenKeepsItsSha` | `cmd/mrw/lines_agree_test.go` | a CRLF read-then-write applies and keeps `\r\n` | — | S1, S2 |
| `TestAReadOfLineOneDoesNotLicenseLineTwoOfACROnlyFile` | `cmd/mrw/lines_agree_test.go` | guard: a span read still licenses only its span | — | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §120 |
| 2 — something selects it | `read.Run`, `walker.offer`, `countFileLines`, `AstGrep` and `apply.readLines` all reach `lines.Split`; each call-site mutant fails a test |
| 3 — the caller can discover it | the header's `NL`, the served lines, the ast-grep problem line |
| 4 — it is used | found by the 2026-09-24 chaos pass; ADR-009 refuses telemetry |

## Mutation Log
(empty until execute)
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/lines/lines.go` · a CR-only file is split at \n again: TestACROnlyFileIsServedAsTheLinesAWriteAddresses and §120 must go red · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · covers:a CR-only file is served as its lines
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/lines/lines.go` · a mixed-ending file is taken for CRLF: TestAMixedEndingFileKeepsTheStrayCarriageReturnOnBothSides must go red · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · covers:a CRLF line is served without its terminator
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/read/read.go` · read splits at \n alone again: TestACROnlyFileIsServedAsTheLinesAWriteAddresses, TestACRLFLineIsServedWithoutItsTerminator and §120 must go red · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · covers:a CRLF line is served without its terminator
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/mcp/tools.go` · MCP paging counts \n again: TestPagingACROnlyFileToExhaustionCoversEveryLine must go red · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · covers:MCP paging covers the same lines
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/read/read.go` · read hashes the CRLF-normalised text instead of the raw bytes: TestACRLFFileReadWholeThenWrittenKeepsItsSha and §120 must go red · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · covers:read still hashes the raw bytes
- 2026-09-24 · cd91659* · mutant killed · exit 1 · `internal/read/astgrep.go` · a CR-only ast-grep hit is served again: TestAstGrepReportsAHitInACROnlyFile must go red · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · covers:an ast-grep row is not served on a CR-only file

## Invariants

- Read hashes the raw bytes; `seen.SHA` agreement holds.
- LF files are served byte-identically to before.
- Exit codes unchanged.
- No `§NN` row in the Tests table.

## Risks

- A CRLF file's served text loses its `\r`; stated in the Served-path change.
- The paging test's N comes from the fixture; `next_read` only advances the loop.

## Stop Condition

- If going green needs a change in `internal/plan`, `internal/seen`, `internal/check`, `internal/state` or `internal/guide`, stop.
- If the ledger format has to change, stop.

## Out of Scope

- The plan compilers (deferred: ADR-065 T2)
- Translating ast-grep byte offsets on a CR-only file (deferred: docs/adr/BACKLOG.md)

## Verification Log
(empty until execute)
- 2026-09-24 · cd91659* · exit 1 · `set -o pipefail …` · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · ms:637 · test-lock-sha256:a858efdd6b0a44fd0bb21edb5ba5ee3c41aed0d915b70a394fb295d421f59dbe · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGluZXNfYWdyZWVfdGVzdC5nbwlUZXN0QUNSTEZGaWxlUmVhZFdob2xlVGhlbldyaXR0ZW5LZWVwc0l0c1NoYQlhYTYxYmQ3YTJhMTE1NDhmYzFjMWZhMTNmODZmNTk1NzE4YmYzYjZkYWRiNjA5NzUxOWZjNGU2N2JmMjFjOGY4CmJvZHkJY21kL21ydy9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RBUmVhZE9mTGluZU9uZURvZXNOb3RMaWNlbnNlTGluZVR3b09mQUNST25seUZpbGUJZDY3M2JjNjg0NGNiZjNlYzYyNWVjZDYwNWEyOWU5YTdkZWRhNDMwNWI3NWZkY2FhYWFjOWI4OWM0NTYxOWJlMgpib2R5CWNtZC9tcncvbGluZXNfYWdyZWVfdGVzdC5nbwlUZXN0QVdob2xlUmVhZE9mQUNST25seUZpbGVMaWNlbnNlc09ubHlUaGVMaW5lc0l0U2VydmVkCWE5MWExNjM5MmY2YTRkMGE5ZjkwMjg3MTUxYTIzODBkMDlhNTMzOTIwNmQzNDJhNTI4Y2U5NTY0MjYyYmQ2ZTEKYm9keQlpbnRlcm5hbC9saW5lcy9saW5lc190ZXN0LmdvCVRlc3RTcGxpdE9mQW5FbXB0eUZpbGVIYXNOb0xpbmVzCTY4OTI5MTM4OWViYTgwOTFhNjFmMTZmMGE0ZTg1MzFhMDVlNjBkZTE0MDI1NDY3MzQyZjM1ZGZkYjAzZDRkZTUKYm9keQlpbnRlcm5hbC9tY3AvbGluZXNfYWdyZWVfdGVzdC5nbwlUZXN0UGFnaW5nQUNST25seUZpbGVUb0V4aGF1c3Rpb25Db3ZlcnNFdmVyeUxpbmUJYmUzNjc0Zjg5ZGVkOGNkMWQ1MGFlOGM5N2E0ZmFlZjBjNTk5OTAzYWU2MTNmMDdlNmFhNDlhOWM3YmU2ZjkyNQpib2R5CWludGVybmFsL3JlYWQvbGluZXNfYWdyZWVfdGVzdC5nbwlUZXN0QUNSTEZMaW5lSXNTZXJ2ZWRXaXRob3V0SXRzVGVybWluYXRvcgk5OWFiYjhjMDZmOWRjNDAyNjUzNDE0OTI0ZGRlMTE0OTc0ZTIzNGI2ODExN2QwMGE2NjcyZjRiYTM2MjdhNDBlCmJvZHkJaW50ZXJuYWwvcmVhZC9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RBQ1JPbmx5RmlsZUlzU2VydmVkQXNUaGVMaW5lc0FXcml0ZUFkZHJlc3NlcwliMTgxNGZhNTBjOTdlMWVhNzgzODczMzEyZTI1Zjc5ZjM2MGY3ZjhmZjU5OGE0NDhhZWU2YTVmMTA5OGRjNmY4CmJvZHkJaW50ZXJuYWwvcmVhZC9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RBTWl4ZWRFbmRpbmdGaWxlS2VlcHNUaGVTdHJheUNhcnJpYWdlUmV0dXJuT25Cb3RoU2lkZXMJYjM2ZmFlZWJiNmRmNWQ2OTM1ZTY0NDA3ZTY5YjJjYzk2NmE4MzA0MDA2OGYxZGJkNDRmZmZkMmIzNzFiODQ4Zgpib2R5CWludGVybmFsL3JlYWQvbGluZXNfYWdyZWVfdGVzdC5nbwlUZXN0QXN0R3JlcFJlcG9ydHNBSGl0SW5BQ1JPbmx5RmlsZQlkN2FjZTBiNDkwMTkzNmFlN2Q0NGIzNTBlZDFmMzJiOWM4OTEwMTU2ZmFkOWYyNzFlN2E2MWMzYzVjNzZmOWYwCmJvZHkJaW50ZXJuYWwvcmVhZC9saW5lc19hZ3JlZV90ZXN0LmdvCVRlc3RHcmVwTWF0Y2hlc0FEb2xsYXJBbmNob3JlZExpbmVJbkFDUkxGRmlsZQlkYTlmYzU4MGU4OTRlODJhN2U4ZGI1NWJlYzQ5OTYwODM3ZDNmMDM0YmM5ZmI1YTgwNzhlOThhOTYzYjU3ZDk4
  ```
  --- last 10 line(s) of stdout (of 47 after folding 47 raw)
  --- PASS: TestAReadOfLineOneDoesNotLicenseLineTwoOfACROnlyFile (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.020s
  === RUN   TestACRLFFileKeepsItsLineEndings
  --- PASS: TestACRLFFileKeepsItsLineEndings (0.00s)
  === RUN   TestACRTerminatedFileHasAddressableLines
  --- PASS: TestACRTerminatedFileHasAddressableLines (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.012s
  FAIL
  ```
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · ms:28713
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · ms:28506
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · ms:28299
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · ms:28490
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · ms:28469
- 2026-09-24 · cd91659* · exit 0 · `set -o pipefail …` · acceptance-sha256:992dd292b2cd56a9548e46f4ca6c6354e18c3b8f65b3e534b00f8061c28f7153 · ms:28425
