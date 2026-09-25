# Task ADR-073-T1: An encoded or non-regular file is not line-edited

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `lines.Unsplittable`; the refusal in `apply`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a BOM names its encoding`, `UTF-32 is checked before UTF-16`, `the NUL scan is bounded in bytes`, `path ops are not line edits`, `a non-regular file is refused before it is read`, `a contract row drives the binary`, `the engine packages are unchanged`

## Goal

A UTF-16LE file was served as byte-split lines and a replace rewrote it at exit 0, BOM gone and
encodings mixed. A FIFO named in a plan blocked the write in `os.ReadFile`. Refuse both by name,
before anything is written.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/lines/lines.go` | edit | `Unsplittable` |
| `internal/lines/lines_test.go` | edit | the table of prefixes |
| `internal/apply/apply.go` | edit | `text.foreign` set by `readLines`; the per-file refusal; the regular-file check beside the ADR-021 stat |
| `internal/apply/encoding_test.go` | new | refusals, the byte bound, the path ops |
| `internal/apply/encoding_unix_test.go` | new | the FIFO (Mkfifo is unix-only) |
| `scripts/contract.sh` | edit | §146 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN; the apply suite stays green. [proof: mutation]
   Mutants: the `FF FE` branch dropped; UTF-16 checked before UTF-32; the 8192 bound halved; the guard applied to unlink; the regular-file check dropped.
3. [S3] Contract §146: a UTF-16 file refused, bytes unchanged; the UTF-8 pair applies; unlink of the UTF-16 file applies. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/lines/ ./internal/apply/ -count=1 -timeout 120s -run 'TestUnsplittable|TestAUTF16File|TestANULInTheFirst8KiB|TestUnlinkAndRenameOfAForeign|TestAFIFONamedInAPlan' -v 2>&1 | tee /tmp/adr073-T1.out \
  && missing=$(for t in TestUnsplittableNamesEveryEncodingItKnows TestAUTF16FileIsRefusedNotRewrittenAsMixedEncodings TestANULInTheFirst8KiBIsRefusedAndOneAfterItIsNot TestUnlinkAndRenameOfAForeignFileStillApply TestAFIFONamedInAPlanIsRefusedNotWaitedOn; do grep -qE "^--- PASS: $t \(" /tmp/adr073-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 146\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/encoding_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/read ':(exclude)internal/read/read.go' ':(exclude)internal/read/encoding_test.go')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestUnsplittableNamesEveryEncodingItKnows` | `internal/lines/lines_test.go` | each BOM, UTF-32 before UTF-16, a NUL and its offset, plain UTF-8 and a UTF-8 BOM pass | — | S1, S2 |
| `TestAUTF16FileIsRefusedNotRewrittenAsMixedEncodings` | `internal/apply/encoding_test.go` | five encodings refused, with and without `--force`, bytes unchanged | — | S1, S2 |
| `TestANULInTheFirst8KiBIsRefusedAndOneAfterItIsNot` | `internal/apply/encoding_test.go` | offset 8191 refused, 8192 edited | — | S1, S2 |
| `TestUnlinkAndRenameOfAForeignFileStillApply` | `internal/apply/encoding_test.go` | unlink, rename and create still apply | — | S1, S2 |
| `TestAFIFONamedInAPlanIsRefusedNotWaitedOn` | `internal/apply/encoding_unix_test.go` | refused within 3 s, not blocked | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the guard |
| 2 — something selects it | every write, CLI and MCP |
| 3 — the caller can discover it | the refusal names the encoding |
| 4 — it is used | the v1.25.1 round corrupted a UTF-16 file this way |

## Verification Log
(empty until execute)
- 2026-09-25 · 59a91a8* · exit 1 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:4000 · test-lock-sha256:325cb58dad904a9f63f7cf8624475dec345bdee6ca904f70223aef8ae7eec2e5 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2VuY29kaW5nX3Rlc3QuZ28JVGVzdEFOVUxJblRoZUZpcnN0OEtpQklzUmVmdXNlZEFuZE9uZUFmdGVySXRJc05vdAkyMzZhNDU5NjYwYzM0M2JlNTE1MDZlOTM4OWYyZjdlODkzYjc3MDVhNjhmYjEwNzVkNGRkMWYyMjg2NmI1MjNmCmJvZHkJaW50ZXJuYWwvYXBwbHkvZW5jb2RpbmdfdGVzdC5nbwlUZXN0QVVURjE2RmlsZUlzUmVmdXNlZE5vdFJld3JpdHRlbkFzTWl4ZWRFbmNvZGluZ3MJMmNmNWI5MmZlYTI2MjE0MTFkMzU1NTkwNWQ3OGJiOGQ1ZTFlMzIxNjRkYzc3NmUzMTc5MzdjNDcxZmZlNzUwMApib2R5CWludGVybmFsL2FwcGx5L2VuY29kaW5nX3Rlc3QuZ28JVGVzdFVubGlua0FuZFJlbmFtZU9mQUZvcmVpZ25GaWxlU3RpbGxBcHBseQkxNGU2NjZlYTc2YzNlYmJiMTExMzY5ODk5MjljMzIxMzA1ZDA4ZTBjYWFjOWY4MTNhY2ZjYjc1MWMyNjVjYWRjCmJvZHkJaW50ZXJuYWwvYXBwbHkvZW5jb2RpbmdfdW5peF90ZXN0LmdvCVRlc3RBRklGT05hbWVkSW5BUGxhbklzUmVmdXNlZE5vdFdhaXRlZE9uCWM2NjNlNzgyNWFjYmYzOTk3ZDgwOWM4YTc3YTQ2NDlmMmNiMDljMTY5NDhmYjUwZTAxOTJkNTcwMzZjNjE4NzIKYm9keQlpbnRlcm5hbC9saW5lcy9saW5lc190ZXN0LmdvCVRlc3RTcGxpdE9mQW5FbXB0eUZpbGVIYXNOb0xpbmVzCTY4OTI5MTM4OWViYTgwOTFhNjFmMTZmMGE0ZTg1MzFhMDVlNjBkZTE0MDI1NDY3MzQyZjM1ZGZkYjAzZDRkZTUKYm9keQlpbnRlcm5hbC9saW5lcy9saW5lc190ZXN0LmdvCVRlc3RVbnNwbGl0dGFibGVOYW1lc0V2ZXJ5RW5jb2RpbmdJdEtub3dzCTI5MDg4Y2YzYzhkYjU3YzdiYjMwZjMxYjI3ZTEwMWM0MGUzNGNhYWE2NmY2MjdkMjJkMDQ2OWRiYzBjZWEzNmE
  ```
  --- last 10 line(s) of stdout (of 36 after folding 36 raw)
      encoding_test.go:64: a NUL at offset 8191: applied true, want refused true ([{singleLineCode:false wrapTail:false Path:f.txt Addr:2 Op:replace Status:ok Reason: Removed:1 Added:1 SrcLine:0 RemovedFirst: RemovedLast: Echo:[] Balance:}])
  --- FAIL: TestANULInTheFirst8KiBIsRefusedAndOneAfterItIsNot (0.00s)
  === RUN   TestUnlinkAndRenameOfAForeignFileStillApply
  --- PASS: TestUnlinkAndRenameOfAForeignFileStillApply (0.00s)
  === RUN   TestAFIFONamedInAPlanIsRefusedNotWaitedOn
      encoding_unix_test.go:33: a write naming a FIFO blocked
  --- FAIL: TestAFIFONamedInAPlanIsRefusedNotWaitedOn (3.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	3.181s
  FAIL
  ```
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:1329
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:739
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:357
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:510
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:519
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:298
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:442
- 2026-09-25 · 59a91a8* · exit 0 · `set -o pipefail …` · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · ms:377

## Mutation Log
(empty until execute)
- 2026-09-25 · 59a91a8* · mutant killed · exit 1 · `internal/lines/lines.go` · a UTF-16LE file is splittable again and a line edit rewrites it · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · covers:a BOM names its encoding
- 2026-09-25 · 59a91a8* · mutant killed · exit 1 · `internal/lines/lines.go` · UTF-32LE is no longer recognised first, so it is named UTF-16 · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · covers:UTF-32 is checked before UTF-16
- 2026-09-25 · 59a91a8* · mutant killed · exit 1 · `internal/lines/lines.go` · the NUL scan stops at 4 KiB, so a NUL at offset 8191 is missed · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · covers:the NUL scan is bounded in bytes
- 2026-09-25 · 59a91a8* · mutant inconclusive · exit 1 · `internal/apply/apply.go` · an unlink or rename of an encoded file is refused as a line edit · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · covers:path ops are not line edits
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-25 · 59a91a8* · mutant killed · exit 1 · `internal/apply/apply.go` · a FIFO named in a plan is opened and the write blocks · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · covers:a non-regular file is refused before it is read
- 2026-09-25 · 59a91a8* · mutant inconclusive · exit 1 · `internal/apply/apply.go` · x · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · covers:path ops are not line edits
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-25 · 59a91a8* · mutant killed · exit 1 · `internal/apply/apply.go` · an unlink of an encoded file is refused as if it were a line edit · acceptance-sha256:a52c1ea755f0ff244be25d88eaf5490736bd1dafe6ce0edba8d1ea3c72892814 · covers:path ops are not line edits

## Invariants

- A UTF-8 file with a UTF-8 BOM is edited as before.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-073 Out of Scope)

## Stop Condition

Stop if the guard needs the walk.
