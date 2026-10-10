# Task ADR-142-T1: `--create` drops a pipe terminator, names a BOM, and says a path exists

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `ingest.CreateContent`, the create-exists message
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/create.go` | edit | `CreateContent` |
| `internal/apply/apply.go` | edit | the create-on-existing message when the path was not read |
| `cmd/mrw/main.go` | edit | `--create` calls `CreateContent` and prints each note on stderr |
| `internal/ingest/create142_test.go`, `internal/apply/create142_test.go`, `cmd/mrw/create142_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §243 |
| `AGENTS.md` | edit | the `--create` paragraph |

## Ordered Steps

1. [S1] Write `TestAPowerShellPipeTerminatorAfterLFContentIsDropped` and `TestALeadingByteOrderMarkIsKeptAndNamed` (ingest), `TestACreateOfAnUnreadExistingPathSaysItExists` (apply) and `TestWriteCreateDropsAPipeTerminatorAndSaysSo` (cmd/mrw). Confirm RED.
2. [S2] `CreateContent`, its call and the notes in `cmd/mrw`, the create-exists case in `apply`. Mutants: the terminator dropped for every content that ends in CRLF; a drop that needs no LF in the text before it; the BOM note removed; the call from `cmd/mrw` removed; the create-exists case removed. [proof: mutation]
3. [S3] Contract §243 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/ingest/ -count=1 -timeout 300s -run 'TestAPowerShellPipeTerminatorAfterLFContentIsDropped|TestALeadingByteOrderMarkIsKeptAndNamed' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPowerShellPipeTerminatorAfterLFContentIsDropped \(' "$out" \
  && grep -qE '^--- PASS: TestALeadingByteOrderMarkIsKeptAndNamed \(' "$out" \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestACreateOfAnUnreadExistingPathSaysItExists' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestACreateOfAnUnreadExistingPathSaysItExists \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestWriteCreateDropsAPipeTerminatorAndSaysSo' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestWriteCreateDropsAPipeTerminatorAndSaysSo \(' "$out" \
  && go test ./internal/ingest/ ./internal/apply/ -count=1 -timeout 900s \
  && grep -q '^# 243\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPowerShellPipeTerminatorAfterLFContentIsDropped` | `internal/ingest/create142_test.go` | a final CRLF after LF text with no other CR is dropped with a note; every other shape is returned untouched | none | S1, S2 |
| `TestALeadingByteOrderMarkIsKeptAndNamed` | `internal/ingest/create142_test.go` | a BOM is kept and named | none | S1, S2 |
| `TestACreateOfAnUnreadExistingPathSaysItExists` | `internal/apply/create142_test.go` | a lone create of an unread existing path says it exists; an edit keeps its refusal | none | S1, S2 |
| `TestWriteCreateDropsAPipeTerminatorAndSaysSo` | `cmd/mrw/create142_test.go` | the CLI creates the file and prints each note on stderr; a mixed shape is still refused | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `ingest.CreateContent` |
| 2 — something selects it | the `--create` branch of `write` in `cmd/mrw/main.go` |
| 3 — the caller can discover it | the stderr note names what happened; AGENTS.md |
| 4 — it is used | the 2026-10-10 Windows round (three of five sessions); no telemetry (ADR-009) |

## Invariants

- Every refusal of ADR-139 other than the pipe terminator stands; `CompileCreate` is unchanged.
- The receipt and the exit codes do not change; the notes go to stderr only.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the pipe terminator cannot be told from content without refusing a shape the 2026-10-10 reports show a real caller sending.

## Out of Scope

- Name refusals and NTFS stream names (deferred: docs/adr/BACKLOG.md)

## Mutation Log
- 2026-10-10 · fdbde90* · mutant killed · exit 1 · `internal/ingest/create.go` · S2: the final CRLF is dropped from every content that ends in one · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · covers:--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists
- 2026-10-10 · fdbde90* · mutant killed · exit 1 · `internal/ingest/create.go` · S2: the drop needs no LF in the text before the CRLF · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · covers:--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists
- 2026-10-10 · fdbde90* · mutant killed · exit 1 · `internal/ingest/create.go` · S2: the byte order mark is not named · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · covers:--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists
- 2026-10-10 · fdbde90* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: --create does not call CreateContent · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · covers:--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists
- 2026-10-10 · fdbde90* · mutant killed · exit 1 · `internal/apply/apply.go` · S2: the create-exists case is removed · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · covers:--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists
- 2026-10-10 · ea0fe3d · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the notes are printed for a content that is then refused · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · covers:--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists

## Verification Log
- 2026-10-10 · fdbde90* · exit 1 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:151 · test-lock-sha256:4ca2b7ea745f7497d0b80e237a503921c7a52352db25daf84eb9b8ceedb0bd71 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jcmVhdGUxNDJfdGVzdC5nbwlUZXN0V3JpdGVDcmVhdGVEcm9wc0FQaXBlVGVybWluYXRvckFuZFNheXNTbwkwZmQ0NmJlMWNjMGFjMjZhNzkyMmFlOWZiOTAyODhhNWUwNjZmNTBjYjhkMjIwZWY2YjYwZTZhNWJhODQ2NzE1CmJvZHkJaW50ZXJuYWwvYXBwbHkvY3JlYXRlMTQyX3Rlc3QuZ28JVGVzdEFDcmVhdGVPZkFuVW5yZWFkRXhpc3RpbmdQYXRoU2F5c0l0RXhpc3RzCTYzZWMzYTc4NDE2YTQwZGRkMzM2OWZkZGI3NmZjOTU5YTc1NmU0ODEzZmI1NzUzNzhkZmYzNDkzZjhmZmNjYmUKYm9keQlpbnRlcm5hbC9pbmdlc3QvY3JlYXRlMTQyX3Rlc3QuZ28JVGVzdEFMZWFkaW5nQnl0ZU9yZGVyTWFya0lzS2VwdEFuZE5hbWVkCWExZDc4YTgyMmMzNWU0OTYyOWI1ODc0ODY2YWM4NDMzNWRlYjJhNjMwMTY1NTc0Y2YxOWUzNjY1OTdjZDhjZTEKYm9keQlpbnRlcm5hbC9pbmdlc3QvY3JlYXRlMTQyX3Rlc3QuZ28JVGVzdEFQb3dlclNoZWxsUGlwZVRlcm1pbmF0b3JBZnRlckxGQ29udGVudElzRHJvcHBlZAkzMDg5NDJlMmFiYWYxNDc4YTAyNDMwMTNjZjM5NTMwYjRiNDkyODkwNWI5ZWQxZDA5ZjliNTkxZWU2NjkyNmQx
  ```
  --- last 7 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest.test]
  internal/ingest/create142_test.go:18:17: undefined: CreateContent
  internal/ingest/create142_test.go:27:17: undefined: CreateContent
  internal/ingest/create142_test.go:41:16: undefined: CreateContent
  internal/ingest/create142_test.go:48:17: undefined: CreateContent
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest [build failed]
  FAIL
  ```
- 2026-10-10 · fdbde90* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:2995
- 2026-10-10 · fdbde90* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:4231
- 2026-10-10 · fdbde90* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:3925
- 2026-10-10 · fdbde90* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:3899
- 2026-10-10 · fdbde90* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:3667
- 2026-10-10 · fdbde90* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:3678
- 2026-10-10 · 0401713* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:0 · test-lock-sha256:11552bb3d98d9f5061aee7fd6d01b42faf09f78e80addda53f8adf79811bc9dc · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jcmVhdGUxNDJfdGVzdC5nbwlUZXN0V3JpdGVDcmVhdGVEcm9wc0FQaXBlVGVybWluYXRvckFuZFNheXNTbwljM2U4MDExMzk5NGE2ODAzYzQxZjIxOGM3NmZlMjExNDY3OGRkYWQxOTZlNmZlZjU4MGM2MjZiNzZjMmI2Mjc0CmJvZHkJaW50ZXJuYWwvYXBwbHkvY3JlYXRlMTQyX3Rlc3QuZ28JVGVzdEFDcmVhdGVPZkFuVW5yZWFkRXhpc3RpbmdQYXRoU2F5c0l0RXhpc3RzCTYzZWMzYTc4NDE2YTQwZGRkMzM2OWZkZGI3NmZjOTU5YTc1NmU0ODEzZmI1NzUzNzhkZmYzNDkzZjhmZmNjYmUKYm9keQlpbnRlcm5hbC9pbmdlc3QvY3JlYXRlMTQyX3Rlc3QuZ28JVGVzdEFMZWFkaW5nQnl0ZU9yZGVyTWFya0lzS2VwdEFuZE5hbWVkCWExZDc4YTgyMmMzNWU0OTYyOWI1ODc0ODY2YWM4NDMzNWRlYjJhNjMwMTY1NTc0Y2YxOWUzNjY1OTdjZDhjZTEKYm9keQlpbnRlcm5hbC9pbmdlc3QvY3JlYXRlMTQyX3Rlc3QuZ28JVGVzdEFQb3dlclNoZWxsUGlwZVRlcm1pbmF0b3JBZnRlckxGQ29udGVudElzRHJvcHBlZAkzMDg5NDJlMmFiYWYxNDc4YTAyNDMwMTNjZjM5NTMwYjRiNDkyODkwNWI5ZWQxZDA5ZjliNTkxZWU2NjkyNmQx · test-lock-kind:replace
- 2026-10-10 · ea0fe3d · exit 0 · `set -o pipefail …` · acceptance-sha256:8c7846286b4f3658d5fd863402dc65e7419da9debfccdd648ffebf3aa40a8aa1 · ms:3779
