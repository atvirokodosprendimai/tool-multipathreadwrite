# Task ADR-153-T1: `--count` on the CLI and `count` on `mrw_read`, and `--max-lines` taught in place of a pipe into head

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `read.Options.Count`, `read.Options.Counts` in `internal/read/read.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a line is counted once`, `a counted file licenses nothing`, `the CLI selects the count`, `a count needs a grep`, `flags on served lines are refused`, `mrw_read counts`, `the docs teach --count and --max-lines`, `a file that no longer matches is a miss`

## Goal

Decisions 1–6 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.Count`, `Options.Counts`; the header with its count; the total line |
| `cmd/mrw/main.go` | edit | the `--count` flag and its refusals |
| `cmd/mrw/count153_test.go` | add | the CLI tests |
| `internal/mcp/tools.go`, `internal/mcp/mcp.go` | edit | the `count` argument, its refusals, the `counts` receipt key, the overflow refusal |
| `internal/mcp/count153_test.go` | add | the MCP test |
| `internal/mcp/schema_test.go` | edit | `readSchema` declares `counts`; a counted answer joins the receipts compared |
| `internal/guide/guide.go` | edit | `mrw instructions` teaches `--count` and `--max-lines` in place of `\| head` |
| `docs/receipts.txt` | edit | `mcp_read counts` |
| `scripts/contract.sh` | edit | §257 drives the built binary |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | the docs, and the survey entry marked |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestAGrepCountSaysHowManyLinesMatchAndLicensesNothing` (two files, one line matching twice: the counts and the total are printed, no line is served, exit 0, and a write to a counted line is refused as unread); `TestACountWithoutAGrepOrBesideAServedLineFlagIsRefused` (`--count` alone, with a range, with `--ast-grep`, and beside `--stat`, `-C`, `--max-lines`, `--max-cols`, `-N`: exit 2, each naming the flag); `TestACountThatMatchesNothingExitsAsAGrep` (exit 1, `mrw: no match:`); `TestMCPReadCountsWhatGrepMatches` (`count: true` with `grep` gives `counts` per file, serves no line, and is refused without `grep`, beside `stat`, `max_lines` or `ast_grep`).
2. [S2] `Options.Count` and `Options.Counts` in `read.Run`. [proof: mutation]
3. [S3] The CLI flag and its refusals. [proof: mutation]
4. [S4] The MCP argument, its refusals and the `counts` key. [proof: mutation]
5. [S5] `mrw instructions`, AGENTS.md, README, BACKLOG, `docs/receipts.txt`. [proof: mutation]
6. [S6] Contract §257: counts and the total through the built binary, the counted line's write refused, and the pair — no match exits 1, `--count` without `--grep` exits 2. [proof: human: the executor runs ./scripts/contract.sh once and reads the §257 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ ./internal/mcp/ ./internal/read/ -count=1 -timeout 300s -run 'TestAGrepCountSaysHowManyLinesMatchAndLicensesNothing|TestACountWithoutAGrepOrBesideAServedLineFlagIsRefused|TestACountThatMatchesNothingExitsAsAGrep|TestMCPReadCountsWhatGrepMatches|TestTheReadReceiptMatchesItsSchema|TestNoShippedMCPReceiptFieldDisappears|TestEveryReadFlagIsTaughtByInstructions|TestACountOfAFileThatNoLongerMatchesIsAProblem' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAGrepCountSaysHowManyLinesMatchAndLicensesNothing \(' "$out" \
  && grep -qE '^--- PASS: TestACountWithoutAGrepOrBesideAServedLineFlagIsRefused \(' "$out" \
  && grep -qE '^--- PASS: TestACountThatMatchesNothingExitsAsAGrep \(' "$out" \
  && grep -qE '^--- PASS: TestMCPReadCountsWhatGrepMatches \(' "$out" \
  && grep -qE '^--- PASS: TestTheReadReceiptMatchesItsSchema \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && grep -qE '^--- PASS: TestEveryReadFlagIsTaughtByInstructions \(' "$out" \
  && grep -qE '^--- PASS: TestACountOfAFileThatNoLongerMatchesIsAProblem \(' "$out" \
  && grep -q '^# 257\. ' scripts/contract.sh \
  && grep -q 'instead of `| head`' AGENTS.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAGrepCountSaysHowManyLinesMatchAndLicensesNothing` | `cmd/mrw/count153_test.go` | counts per file and in total, a line counted once, nothing served, the counted line's write refused | none | S1, S2, S3 |
| `TestACountWithoutAGrepOrBesideAServedLineFlagIsRefused` | `cmd/mrw/count153_test.go` | each refusal, exit 2, by name | none | S1, S3 |
| `TestACountThatMatchesNothingExitsAsAGrep` | `cmd/mrw/count153_test.go` | exit 1 with ADR-146's `no match:` | none | S1, S3 |
| `TestMCPReadCountsWhatGrepMatches` | `internal/mcp/count153_test.go` | `counts` in the receipt, nothing served, the refusals | none | S1, S4 |
| `TestTheReadReceiptMatchesItsSchema` | `internal/mcp/schema_test.go` | a counted answer's receipt matches the declared schema | none | S4 |
| `TestNoShippedMCPReceiptFieldDisappears` | `internal/mcp/receipts111_test.go` | `mcp_read counts` is listed in `docs/receipts.txt` | none | S5 |
| `TestEveryReadFlagIsTaughtByInstructions` | `cmd/mrw/instructions_test.go` | `mrw instructions` teaches `--count` | none | S5 |
| `TestACountOfAFileThatNoLongerMatchesIsAProblem` | `internal/read/count153_test.go` | a file the walk found that no longer matches is a missed range and a problem, not a count of 0 at exit 0 (the Codex review of #406) | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `read.Options.Count` |
| 2 — something selects it | the `--count` flag; `mrw_read`'s `count`; contract §257 drives the built binary |
| 3 — the caller can discover it | `mrw read --help`, `mrw instructions`, the `mrw_read` schema, AGENTS.md and README |
| 4 — it is used | nothing measures this; built because five survey sessions asked for counts (2026-10-09) |

## Mutation Log
(empty until execute)
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `internal/read/read.go` · a counted file is served and licensed like a grep · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f3964* · mutant inconclusive · exit 1 · `internal/read/read.go` · every line of a counted file is counted, matching or not · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never selects the count · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `cmd/mrw/main.go` · --count without --grep is not refused · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_read never carries counts · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_read takes count without grep · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `internal/guide/guide.go` · mrw instructions never teaches --count · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md stops teaching --max-lines in place of head · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f3964* · mutant killed · exit 1 · `internal/read/read.go` · every line of a counted file is counted, matching or not · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225
- 2026-10-14 · 29f39646* · mutant killed · exit 1 · `internal/read/read.go` · every line of a counted file is counted, matching or not · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · covers:a line is counted once
- 2026-10-14 · 29f39646* · mutant killed · exit 1 · `internal/read/read.go` · a counted file is served and licensed like a grep · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · covers:a counted file licenses nothing
- 2026-10-14 · 29f39646* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never selects the count · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · covers:the CLI selects the count
- 2026-10-14 · 29f39646* · mutant killed · exit 1 · `cmd/mrw/main.go` · --count without --grep is not refused · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · covers:a count needs a grep
- 2026-10-14 · 29f39646* · mutant killed · exit 1 · `cmd/mrw/main.go` · a flag that acts on served lines is taken beside --count · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · covers:flags on served lines are refused
- 2026-10-14 · 29f39646* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_read never carries counts · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · covers:mrw_read counts
- 2026-10-14 · 29f39646* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md stops teaching --max-lines in place of head · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · covers:the docs teach --count and --max-lines
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `internal/read/read.go` · every line of a counted file is counted, matching or not · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:a line is counted once
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `internal/read/read.go` · a counted file is served and licensed like a grep · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:a counted file licenses nothing
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never selects the count · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:the CLI selects the count
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `cmd/mrw/main.go` · --count without --grep is not refused · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:a count needs a grep
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `cmd/mrw/main.go` · a flag that acts on served lines is taken beside --count · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:flags on served lines are refused
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_read never carries counts · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:mrw_read counts
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md stops teaching --max-lines in place of head · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:the docs teach --count and --max-lines
- 2026-10-14 · fc7bb31c* · mutant killed · exit 1 · `internal/read/read.go` · a counted file that no longer matches is a count of 0 at exit 0 · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · covers:a file that no longer matches is a miss

## Verification Log
(empty until execute)
- 2026-10-14 · 29f3964* · exit 1 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:888 · test-lock-sha256:18abdf0bc0904cc3a1a5dd35b0e286abd002a0834414b3e9dc33f10912d41202 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3VudDE1M190ZXN0LmdvCVRlc3RBQ291bnRUaGF0TWF0Y2hlc05vdGhpbmdFeGl0c0FzQUdyZXAJYTFjYjZhOTUzM2Y4YTE1ZTFmOGZiZTIyOTQ1YmU2NjcyY2JlOWRjZWY0NjJlOTk0ODZiN2VhYmY0MTI0M2QyMwpib2R5CWNtZC9tcncvY291bnQxNTNfdGVzdC5nbwlUZXN0QUNvdW50V2l0aG91dEFHcmVwT3JCZXNpZGVBU2VydmVkTGluZUZsYWdJc1JlZnVzZWQJNmIyZGM4ZjhhMjQyNjA3NjEwYjYwNDg5NTcyMDVmNjdlZDFhM2Y5YTBlMDY5NzU2ZmRmM2QwMjU3NDVmODg1OApib2R5CWNtZC9tcncvY291bnQxNTNfdGVzdC5nbwlUZXN0QUdyZXBDb3VudFNheXNIb3dNYW55TGluZXNNYXRjaEFuZExpY2Vuc2VzTm90aGluZwlkNGM1ZWJmYTBlMjlhMDk1MGM0ZWU5OGVjMDhiZmU3YzRmZTc5YzFjNzVlYTVjOWM3ZjAzMmQ5MTJjOTViNzM2CmJvZHkJY21kL21ydy9pbnN0cnVjdGlvbnNfdGVzdC5nbwlUZXN0RXZlcnlSZWFkRmxhZ0lzVGF1Z2h0QnlJbnN0cnVjdGlvbnMJODdlZGE1ZDFlMjNlMGE5N2NkNjg0NmJhMzlhNTg2MGE1YTZlZGNmYmUzYmE0YThmOTk1YzIwODY0YmZiYTFhZgpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEhlbHBTdW1tYXJpZXNOYW1lVGhlUmVhZFNpZGUJMWViYmI5NzU2MDExN2YxYzBjNzRkZWViOGM5NzU5OGVjMTFhYjcxYWYwMjcwYTViYmU0NzhjYWZkMjk3ZDUyYQpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvbW1hbmRQcmludHNDTEkJZTMwNjAwMzhkNGZhNDE0NmEzODU1NjhjNmQ1MmY0MzVmY2VmYmQyNzhiMDFkM2Q2NTlmZmFlYTFlOGRlYmMwMApib2R5CWludGVybmFsL21jcC9jb3VudDE1M190ZXN0LmdvCVRlc3RNQ1BSZWFkQ291bnRzV2hhdEdyZXBNYXRjaGVzCWUxYzYzNTQ0MzYwZTIzZDczMzc5ODBlNTAxNDk1ZmIwMDcwNjY1ZDk5ZTM3MDBlZGM2ZjY2MjMyMmJlMDQwYTkKYm9keQlpbnRlcm5hbC9tY3AvcmVjZWlwdHMxMTFfdGVzdC5nbwlUZXN0Tm9TaGlwcGVkTUNQUmVjZWlwdEZpZWxkRGlzYXBwZWFycwk1YjZmYjY5ZDA2ZGYzZjM4MDYwNTMwOTljY2NlOGQzNDk3ODUyNzc4ZDgyNjAxNjU5MWIyYmEwODU0ZmU1OGQ4CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBQ3VzdG9tTWFyc2hhbGxlck1lYW5zTm9SZXF1aXJlZFByb21pc2UJMjM5NDhkOTRmOWYxMDllOWFmYThjMWIyOTAzMGM2NTM2Mzc1YTY4YTY5YWFjYjQ5NGJjZmMzODM3NWI4ZjlkOApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QURlc2NyaWJlZFByb3BlcnR5VGhhdE5vTG9uZ2VyRXhpc3RzSXNSZWZ1c2VkCTlkYjg3ZWVjMzYzNGQ3YjRhY2FiYjQzYTg3YzE1NDRmNzYxNzBmYTc3N2U2YzE3MzViNTExMTJlNDJmZTIzZDEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFOaWxTbGljZUlzQWRtaXR0ZWRCeVRoZVNjaGVtYQljODA5ZGJiNDQ2MjU1YmQwMzg4Njk5YzU0YzI3MTNjYjllODAwM2U2YzNmOTNlYzhiYmUxNjAxOGQxNWNiZjg4CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBUHJvcGVydHlsZXNzT2JqZWN0U2NoZW1hSXNSZWZ1c2VkCWJmZGY5OWUzOGUzMzhiNTA3OGYzZDJkMGI5ZDNhNzNiMGRiZjE1ZmE0ZWQ4ZWNlZWQ0YjY3Zjg3NDkyMmM0ZjEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdFRoZVJlYWRSZWNlaXB0TWF0Y2hlc0l0c1NjaGVtYQk1MjljMDllM2Y2YmUxMzE0NmFkMmQ0OTM0OTg2Y2FlZWYyMDRhOGRkZGFiMmYxOWZiYzUyNDI1Mzk3ZGE4ODk0CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVTY2hlbWFOYW1lc1RoZUZpZWxkc09mVGhlUmVzdWx0CTViM2E4Y2QxODkyMWFhNGZjZjhhYWYyZjc1ZDAxMjAwOTVhOGY5YjgxMmIxNzc0YWVlMmM2ODRkMjI5YzY0YjY
  ```
  --- last 10 line(s) of stdout (of 32 after folding 32 raw)
  --- FAIL: TestMCPReadCountsWhatGrepMatches (0.00s)
  === RUN   TestNoShippedMCPReceiptFieldDisappears
      receipts111_test.go:35: a receipt field is not in docs/receipts.txt: mcp_read counts
  --- FAIL: TestNoShippedMCPReceiptFieldDisappears (0.00s)
  === RUN   TestTheReadReceiptMatchesItsSchema
      schema_test.go:195: want two content blocks (the answer, then the receipt), got [map[text:mrw_read does not take "count", so nothing was done: nothing was served or written, and an ack sent with this call was not recorded. mrw_read takes "ack", "after", "ast_grep", "exclude", "files_from", "grep", "max_lines", "no_ignore", "specs", "stat" (this server is mrw dev, and a key a later mrw declares is refused by this one). `mrw read` in a shell has what this tool does not, such as --context and --no-numbers. type:text]]
  --- FAIL: TestTheReadReceiptMatchesItsSchema (0.14s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.353s
  FAIL
  ```
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1614
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1521
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1539
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1543
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1776
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1548
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1524
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1776
- 2026-10-14 · 29f3964* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1633
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:716
- 2026-10-14 · human-observed · observed by the executor 2026-10-14: S6 ./scripts/contract.sh exit 0 ('contract holds'); all eight §257 rows PASS — the counts and the total through the built binary with no line served, the counted line's write refused and the file unchanged, and the pair: no match exits 1 with ADR-146's no match:, --count without --grep exits 2 naming --grep
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1911
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1533
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1575
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1590
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1637
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:2145
- 2026-10-14 · 29f39646* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c075a5ef9d7f431034cf168251cfa501d6a68cff0af139d7b71f59c850a5225 · ms:1569
- 2026-10-14 · fc7bb31c* · exit 1 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:699
  ```
  --- last 10 line(s) of stdout (of 31 after folding 31 raw)
          ==> b.txt  1L  23B  sha a919af56  0 matching line(s)
          -- 1 matching line(s) in 2 file(s)
      count153_test.go:36: the miss is not reported as a plain grep reports it:
          ==> a.txt  1L  7B  sha 821f4025  1 matching line(s)
          ==> b.txt  1L  23B  sha a919af56  0 matching line(s)
          -- 1 matching line(s) in 2 file(s)
  --- FAIL: TestACountOfAFileThatNoLongerMatchesIsAProblem (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.132s
  FAIL
  ```
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1666
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1655
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1512
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1547
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1575
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1572
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1764
- 2026-10-14 · fc7bb31c* · exit 0 · `set -o pipefail …` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:1545
- 2026-10-14 · human-observed · observed by the executor 2026-10-14 after the Codex review fix: S6 ./scripts/contract.sh exit 0 ('contract holds'); all eight §257 rows PASS — the counts and the total through the built binary with no line served, the counted line's write refused and the file unchanged, and the pair: no match exits 1 with ADR-146's no match:, --count without --grep exits 2 naming --grep
- 2026-10-16 · c8106825* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:0 · test-lock-sha256:e3b72382163422114db0cc6d607bdf94bebb15153c208cfa9935aa26247c9c76 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3VudDE1M190ZXN0LmdvCVRlc3RBQ291bnRUaGF0TWF0Y2hlc05vdGhpbmdFeGl0c0FzQUdyZXAJYTFjYjZhOTUzM2Y4YTE1ZTFmOGZiZTIyOTQ1YmU2NjcyY2JlOWRjZWY0NjJlOTk0ODZiN2VhYmY0MTI0M2QyMwpib2R5CWNtZC9tcncvY291bnQxNTNfdGVzdC5nbwlUZXN0QUNvdW50V2l0aG91dEFHcmVwT3JCZXNpZGVBU2VydmVkTGluZUZsYWdJc1JlZnVzZWQJNmIyZGM4ZjhhMjQyNjA3NjEwYjYwNDg5NTcyMDVmNjdlZDFhM2Y5YTBlMDY5NzU2ZmRmM2QwMjU3NDVmODg1OApib2R5CWNtZC9tcncvY291bnQxNTNfdGVzdC5nbwlUZXN0QUdyZXBDb3VudFNheXNIb3dNYW55TGluZXNNYXRjaEFuZExpY2Vuc2VzTm90aGluZwk0MzkxNmY1MGE0YzI0NTI0ZjU5MTAzMTA4MTZkMWFiN2I0ZjA0MWZlNGYwNmU4YzE0OGIwZDhhODA2MzMxNGZmCmJvZHkJY21kL21ydy9pbnN0cnVjdGlvbnNfdGVzdC5nbwlUZXN0RXZlcnlSZWFkRmxhZ0lzVGF1Z2h0QnlJbnN0cnVjdGlvbnMJODdlZGE1ZDFlMjNlMGE5N2NkNjg0NmJhMzlhNTg2MGE1YTZlZGNmYmUzYmE0YThmOTk1YzIwODY0YmZiYTFhZgpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEhlbHBTdW1tYXJpZXNOYW1lVGhlUmVhZFNpZGUJMWViYmI5NzU2MDExN2YxYzBjNzRkZWViOGM5NzU5OGVjMTFhYjcxYWYwMjcwYTViYmU0NzhjYWZkMjk3ZDUyYQpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvbW1hbmRQcmludHNDTEkJZTMwNjAwMzhkNGZhNDE0NmEzODU1NjhjNmQ1MmY0MzVmY2VmYmQyNzhiMDFkM2Q2NTlmZmFlYTFlOGRlYmMwMApib2R5CWludGVybmFsL21jcC9jb3VudDE1M190ZXN0LmdvCVRlc3RNQ1BSZWFkQ291bnRzV2hhdEdyZXBNYXRjaGVzCWUxYzYzNTQ0MzYwZTIzZDczMzc5ODBlNTAxNDk1ZmIwMDcwNjY1ZDk5ZTM3MDBlZGM2ZjY2MjMyMmJlMDQwYTkKYm9keQlpbnRlcm5hbC9tY3AvcmVjZWlwdHMxMTFfdGVzdC5nbwlUZXN0Tm9TaGlwcGVkTUNQUmVjZWlwdEZpZWxkRGlzYXBwZWFycwk1YjZmYjY5ZDA2ZGYzZjM4MDYwNTMwOTljY2NlOGQzNDk3ODUyNzc4ZDgyNjAxNjU5MWIyYmEwODU0ZmU1OGQ4CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBQ3VzdG9tTWFyc2hhbGxlck1lYW5zTm9SZXF1aXJlZFByb21pc2UJMjM5NDhkOTRmOWYxMDllOWFmYThjMWIyOTAzMGM2NTM2Mzc1YTY4YTY5YWFjYjQ5NGJjZmMzODM3NWI4ZjlkOApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QURlc2NyaWJlZFByb3BlcnR5VGhhdE5vTG9uZ2VyRXhpc3RzSXNSZWZ1c2VkCTlkYjg3ZWVjMzYzNGQ3YjRhY2FiYjQzYTg3YzE1NDRmNzYxNzBmYTc3N2U2YzE3MzViNTExMTJlNDJmZTIzZDEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFOaWxTbGljZUlzQWRtaXR0ZWRCeVRoZVNjaGVtYQljODA5ZGJiNDQ2MjU1YmQwMzg4Njk5YzU0YzI3MTNjYjllODAwM2U2YzNmOTNlYzhiYmUxNjAxOGQxNWNiZjg4CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBUHJvcGVydHlsZXNzT2JqZWN0U2NoZW1hSXNSZWZ1c2VkCWJmZGY5OWUzOGUzMzhiNTA3OGYzZDJkMGI5ZDNhNzNiMGRiZjE1ZmE0ZWQ4ZWNlZWQ0YjY3Zjg3NDkyMmM0ZjEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdFRoZVJlYWRSZWNlaXB0TWF0Y2hlc0l0c1NjaGVtYQk1MjljMDllM2Y2YmUxMzE0NmFkMmQ0OTM0OTg2Y2FlZWYyMDRhOGRkZGFiMmYxOWZiYzUyNDI1Mzk3ZGE4ODk0CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVTY2hlbWFOYW1lc1RoZUZpZWxkc09mVGhlUmVzdWx0CTViM2E4Y2QxODkyMWFhNGZjZjhhYWYyZjc1ZDAxMjAwOTVhOGY5YjgxMmIxNzc0YWVlMmM2ODRkMjI5YzY0YjYKYm9keQlpbnRlcm5hbC9yZWFkL2NvdW50MTUzX3Rlc3QuZ28JVGVzdEFDb3VudE9mQUZpbGVUaGF0Tm9Mb25nZXJNYXRjaGVzSXNBUHJvYmxlbQlmZjdmOTc3MWJhYjM5MjJmNzQ4N2ExNDlmMjk2ZjY4NDkxNDY2ZDYzNDQzZDIwOGRhNmEyY2Y4MzUzMjE2NDdl · test-lock-kind:replace
- 2026-10-23 · 4bfd6600* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5 · ms:0 · test-lock-sha256:5c8285393c5f417af9de37ea2f36ae9b71e53f1345307b683059000eae6e1eee · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9jb3VudDE1M190ZXN0LmdvCVRlc3RBQ291bnRUaGF0TWF0Y2hlc05vdGhpbmdFeGl0c0FzQUdyZXAJYTFjYjZhOTUzM2Y4YTE1ZTFmOGZiZTIyOTQ1YmU2NjcyY2JlOWRjZWY0NjJlOTk0ODZiN2VhYmY0MTI0M2QyMwpib2R5CWNtZC9tcncvY291bnQxNTNfdGVzdC5nbwlUZXN0QUNvdW50V2l0aG91dEFHcmVwT3JCZXNpZGVBU2VydmVkTGluZUZsYWdJc1JlZnVzZWQJNmIyZGM4ZjhhMjQyNjA3NjEwYjYwNDg5NTcyMDVmNjdlZDFhM2Y5YTBlMDY5NzU2ZmRmM2QwMjU3NDVmODg1OApib2R5CWNtZC9tcncvY291bnQxNTNfdGVzdC5nbwlUZXN0QUdyZXBDb3VudFNheXNIb3dNYW55TGluZXNNYXRjaEFuZExpY2Vuc2VzTm90aGluZwk0MzkxNmY1MGE0YzI0NTI0ZjU5MTAzMTA4MTZkMWFiN2I0ZjA0MWZlNGYwNmU4YzE0OGIwZDhhODA2MzMxNGZmCmJvZHkJY21kL21ydy9pbnN0cnVjdGlvbnNfdGVzdC5nbwlUZXN0RXZlcnlSZWFkRmxhZ0lzVGF1Z2h0QnlJbnN0cnVjdGlvbnMJODdlZGE1ZDFlMjNlMGE5N2NkNjg0NmJhMzlhNTg2MGE1YTZlZGNmYmUzYmE0YThmOTk1YzIwODY0YmZiYTFhZgpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEhlbHBTdW1tYXJpZXNOYW1lVGhlUmVhZFNpZGUJMWViYmI5NzU2MDExN2YxYzBjNzRkZWViOGM5NzU5OGVjMTFhYjcxYWYwMjcwYTViYmU0NzhjYWZkMjk3ZDUyYQpib2R5CWNtZC9tcncvaW5zdHJ1Y3Rpb25zX3Rlc3QuZ28JVGVzdEluc3RydWN0aW9uc0NvbW1hbmRQcmludHNDTEkJZTMwNjAwMzhkNGZhNDE0NmEzODU1NjhjNmQ1MmY0MzVmY2VmYmQyNzhiMDFkM2Q2NTlmZmFlYTFlOGRlYmMwMApib2R5CWludGVybmFsL21jcC9jb3VudDE1M190ZXN0LmdvCVRlc3RNQ1BSZWFkQ291bnRzV2hhdEdyZXBNYXRjaGVzCWUxYzYzNTQ0MzYwZTIzZDczMzc5ODBlNTAxNDk1ZmIwMDcwNjY1ZDk5ZTM3MDBlZGM2ZjY2MjMyMmJlMDQwYTkKYm9keQlpbnRlcm5hbC9tY3AvcmVjZWlwdHMxMTFfdGVzdC5nbwlUZXN0Tm9TaGlwcGVkTUNQUmVjZWlwdEZpZWxkRGlzYXBwZWFycwk0ZGQ3MGRiOTJlNzBiNmY3ZDVkZWM2Yzc2OTdiMmQ4NGQ4ZWQzYjVhYjE2ZDhiZmUyM2FiZWFhNDE2OGI0NzNjCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBQ3VzdG9tTWFyc2hhbGxlck1lYW5zTm9SZXF1aXJlZFByb21pc2UJMjM5NDhkOTRmOWYxMDllOWFmYThjMWIyOTAzMGM2NTM2Mzc1YTY4YTY5YWFjYjQ5NGJjZmMzODM3NWI4ZjlkOApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QURlc2NyaWJlZFByb3BlcnR5VGhhdE5vTG9uZ2VyRXhpc3RzSXNSZWZ1c2VkCTlkYjg3ZWVjMzYzNGQ3YjRhY2FiYjQzYTg3YzE1NDRmNzYxNzBmYTc3N2U2YzE3MzViNTExMTJlNDJmZTIzZDEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFOaWxTbGljZUlzQWRtaXR0ZWRCeVRoZVNjaGVtYQljODA5ZGJiNDQ2MjU1YmQwMzg4Njk5YzU0YzI3MTNjYjllODAwM2U2YzNmOTNlYzhiYmUxNjAxOGQxNWNiZjg4CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBUHJvcGVydHlsZXNzT2JqZWN0U2NoZW1hSXNSZWZ1c2VkCWJmZGY5OWUzOGUzMzhiNTA3OGYzZDJkMGI5ZDNhNzNiMGRiZjE1ZmE0ZWQ4ZWNlZWQ0YjY3Zjg3NDkyMmM0ZjEKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdFRoZVJlYWRSZWNlaXB0TWF0Y2hlc0l0c1NjaGVtYQk4NGNiYzkxMThjMDExNTMxMWI2MjdkOTc2MzllMTVjZTQ5ZTg2NjNkMzUxMTk5ZmY4MDExYWQ3YjFkYWQ1Yjg5CmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVTY2hlbWFOYW1lc1RoZUZpZWxkc09mVGhlUmVzdWx0CTViM2E4Y2QxODkyMWFhNGZjZjhhYWYyZjc1ZDAxMjAwOTVhOGY5YjgxMmIxNzc0YWVlMmM2ODRkMjI5YzY0YjYKYm9keQlpbnRlcm5hbC9yZWFkL2NvdW50MTUzX3Rlc3QuZ28JVGVzdEFDb3VudE9mQUZpbGVUaGF0Tm9Mb25nZXJNYXRjaGVzSXNBUHJvYmxlbQlmZjdmOTc3MWJhYjM5MjJmNzQ4N2ExNDlmMjk2ZjY4NDkxNDY2ZDYzNDQzZDIwOGRhNmEyY2Y4MzUzMjE2NDdl · test-lock-kind:replace
- 2026-10-23 · human-observed · relock 2026-10-23 reviewed for ADR-169: the moved locks are additive edits to shared tests — TestInstructionsLeadWithTheRecipes counts 11 recipes (ADR-169 adds the re-read guard), TestTheReadReceiptMatchesItsSchema declares the unchanged list, TestNoShippedMCPReceiptFieldDisappears reads a list's items; no assertion removed or weakened; approved · acceptance-sha256:3cf588c88ae7775188b4e0915d900125af459d6971470b3d95758d6c51a82ab5

## Invariants

- A read that serves no line licenses no line.
- `--stat`, and a `--grep` read without `--count`, answer as before.

## Risks

- None beyond the record's.

## Out of Scope

- A count on a named range or `--files-from` (permanent: boundary: a count answers a grep; a named range is already a served read)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
