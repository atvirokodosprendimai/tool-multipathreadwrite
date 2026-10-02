# Task ADR-117-T1: mrw_read takes max_lines, stat and files_from

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `readArgs.MaxLines`, `.Stat`, `.FilesFrom`; `internal/speclist.Parse`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `mrw_read takes max_lines, stat and files_from`

## Goal

`mrw_read` caps, stats, and reads a list of specs from a file inside the root, each as the CLI's flag does, and licenses exactly what it served.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | the three arguments, their refusals, `read.Run`'s options, the overflow rule for a cap, the routing |
| `internal/mcp/mcp.go`, `internal/mcp/instructions.go` | edit | the input properties; the description and instructions |
| `internal/speclist/speclist.go` | add | the list parser both surfaces use |
| `cmd/mrw/main.go` | edit | `specList` scans through `speclist.Parse` |
| `internal/mcp/readargs117_test.go` | add | `TestMrwReadTakesMaxLinesStatAndFilesFrom` |
| `internal/mcp/undeclared093_test.go`, `internal/mcp/mcp_test.go` | edit | the undeclared examples and the routing move to `context` / `--context` |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md`, `cmd/opencode/mrw-plugin/src/index.ts` | edit | say so; the plugin's args |
| `scripts/contract.sh` | edit | §217 |

## Ordered Steps

1. [S1] Write `TestMrwReadTakesMaxLinesStatAndFilesFrom`: `max_lines: 1` serves one line and WITHHELD, and its ack licenses that line and not the next; `max_lines: 0` serves headers; a negative cap is refused; `stat` serves no line and licenses nothing; `files_from` reads a list, skipping a comment and a blank line; `files_from` is refused beside `specs`, as `-`, through `..`, through a link out of the root, inside the state directory, as a FIFO (within a time bound), empty, and when it holds no spec; a capped read too large for one answer is refused, not paged. Confirm RED. [proof: mutation]
2. [S2] The change. Mutants: `MaxLines` not passed to `read.Run`; `Stat` not passed; the files_from boundary check dropped; the FIFO-safe open replaced by `os.Open`; the cap's overflow paging past the cap; the encoded overflow naming the raw size; the withheld line naming `--max-lines`. [proof: mutation]
3. [S3] The routing, docs, plugin, golden, contract §217 (each argument served; the pair, its refusal). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 600s -run 'TestMrwReadTakesMaxLinesStatAndFilesFrom' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom \(' "$out" \
  && go test ./internal/mcp/ ./internal/speclist/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 217\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMrwReadTakesMaxLinesStatAndFilesFrom` | `internal/mcp/readargs117_test.go` | the cap, the stat and the list, each served and each refused as the record says, and each licensing only what it served | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three `readArgs` fields |
| 2 — something selects it | `readTool` passes them to `read.Run` and `speclist.Parse` |
| 3 — the caller can discover it | the input schema and the description name them |
| 4 — it is used | the gap list of 2026-10-02, item 3b |

## Mutation Log
- 2026-10-02 · d5d9a0e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: MaxLines not passed to read.Run · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · d5d9a0e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: Stat not passed to read.Run · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · d5d9a0e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the files_from boundary check dropped · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · d5d9a0e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the FIFO-safe open replaced by os.Open · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · d5d9a0e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the cap's overflow pages past the cap · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · d5d9a0e* · mutant survived · exit 0 · `internal/mcp/tools.go` · S2: the encoded overflow names the raw size · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-02 · d5d9a0e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the withheld line names --max-lines · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · d5d9a0e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the encoded overflow names the raw size · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e

## Invariants

- A served read without these arguments is byte-identical; the CLI's `--files-from` behaves as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a capped read cannot be checkpointed exactly for what it served.

## Out of Scope

- `--context`, `--no-numbers` (deferred: `docs/adr/BACKLOG.md` "From ADR-117")

## Verification Log
- 2026-10-02 · d5d9a0e* · exit 1 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:1293 · test-lock-sha256:bf54b15c685e0fec3f8edb4a7b8ff115263818be51053c57d82aa13271ed2401 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JVGVzdE1yd1JlYWRUYWtlc01heExpbmVzU3RhdEFuZEZpbGVzRnJvbQllMDZlYjVjYzgzZmEzMTFlZjVkYTU3ZGQzYmI5YjUwNWYwOTVjMDMyZWMxOWMxYjIzYjYyMDQyODlkNzk3ZTA0CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBjYXBwZWQgcmVhZCB0b28gbGFyZ2UgZm9yIG9uZSBhbnN3ZXIgaXMgcmVmdXNlZCwgbm90IHBhZ2VkIHBhc3QgdGhlIGNhcAk4MTQ2ZTk3MWQxNzI2YTY4MjQ0ZThkZDEyOGQ1OWIyZWNlZDQ3MzRkMTY4MDA5MjllODg2ZjQxZWRlNTA2Zjg2CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBuZWdhdGl2ZSBtYXhfbGluZXMgaXMgcmVmdXNlZAk0ODI2Mzc2N2JjNGEwZDkyMGViYjY2NjhkZDc1MjM3Mjk0NjI5ZjVkYjEyZDdmZTYzN2E1NzY0NDFjNzY0NjRkCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JZmlsZXNfZnJvbSBpcyByZWZ1c2VkIHdoZXJlIGl0IGNhbm5vdCBiZSBob25vdXJlZAlkNzU0YzdhMDJmYTllMThjNTA5ODU4Y2NkMTgwOTRhMzY4Yzk2MjdmNTQ3MzMxZWNhNDFmZTFiNjA3ZDI1MGU5CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JZmlsZXNfZnJvbSByZWFkcyBhIGxpc3Qgb2Ygc3BlY3MJYTUxMmQ1MmI0Zjg3YjBkOTg2OGE5MzRkNDE5OTJkNGQyNzE0MGZiNGZlZGJjNjAwZDdjOGE0M2ZhNzFkM2U1NQpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCW1heF9saW5lcyAwIHNlcnZlcyBoZWFkZXJzIG9ubHkJMDYyYzc4ZDBlMzc3MGViYzY3MzkyNTU3MTNmZGNkYWZlZjlmODNlZjg0MWRlYTI4OWRmY2Q0ZWRlYzM2NmRmMQpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCW1heF9saW5lcyBzZXJ2ZXMgTiBsaW5lcyBhbmQgbGljZW5zZXMgb25seSB0aG9zZQkyZmRmZTg3MTdlYjc5NjhkMTFkOTUxZDNlNjAyYzNjZTllNzJmZDE3ODMxM2E5YWZiY2ExYzhlN2U1NzMyMDFmCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28Jc3RhdCBzZXJ2ZXMgbm8gbGluZSBhbmQgbGljZW5zZXMgbm90aGluZwk5ZjZkMWQ1MTQzNjdmZDlmODg0NTdkNzJjNzM4ZGMzM2NjOGM5ZDhiY2Y4ZTMwMDQxNDNkM2M3YTI5MjI3MzQw
  ```
  --- last 10 line(s) of stdout (of 24 after folding 24 raw)
      --- FAIL: TestMrwReadTakesMaxLinesStatAndFilesFrom/max_lines_serves_N_lines_and_licenses_only_those (0.00s)
      --- FAIL: TestMrwReadTakesMaxLinesStatAndFilesFrom/max_lines_0_serves_headers_only (0.00s)
      --- FAIL: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_negative_max_lines_is_refused (0.00s)
      --- FAIL: TestMrwReadTakesMaxLinesStatAndFilesFrom/stat_serves_no_line_and_licenses_nothing (0.00s)
      --- FAIL: TestMrwReadTakesMaxLinesStatAndFilesFrom/files_from_reads_a_list_of_specs (0.00s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/files_from_is_refused_where_it_cannot_be_honoured (0.00s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_capped_read_too_large_for_one_answer_is_refused,_not_paged_past_the_cap (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.202s
  FAIL
  ```
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:39556
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:39163
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:40109
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:40555
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:40090
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:40379
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:40548
- 2026-10-02 · d5d9a0e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:39350
