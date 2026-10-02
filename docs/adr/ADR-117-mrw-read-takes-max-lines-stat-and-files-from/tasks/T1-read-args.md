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
| `internal/mcp/ack.go` | edit | `markServed` merges a file's checkpoints across specs |
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
4. [S4] What the reviews of #318 found: two specs naming one file — a list split into ranges, each with its own budget — kept only the second's checkpoints, so an honest ack of the first licensed nothing (it failed closed, and predates this record). The first fix merged every slice's spans per path, and its confirmation review proved that a licence hole: read.Run reads a file once per spec, so a writer between specs leaves slices of two versions, and the merge held the older version's spans under the newer sha — a probe swapping the file by rename got a write of a line never served accepted. `markServed` now keys each slice by its header's sha and `heldSpans` holds only the observed version's, merged. Also: a stat too large was told to send a smaller `max_lines`; a list that grew past the bound after it was measured was cut without a word; the state-directory check after `rooted.Resolve` could never fire, and is gone; a capped grep too large answers with its index, which the record now says and a subtest pins. Mutants: the spans overwritten instead of merged; a replaced version's spans held; the stat refusal advising `max_lines`. [proof: mutation]

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
| `TestASpanFromAReplacedVersionIsNotHeld` | `internal/mcp/heldspans117_test.go` | slices of one file from two versions are kept apart, and only the observed version's spans are held, merged | none | S4 |

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
- 2026-10-02 · fe43b6e* · mutant killed · exit 1 · `internal/mcp/ack.go` · S4: a file's checkpoints overwritten across specs · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · fe43b6e* · mutant killed · exit 1 · `internal/mcp/tools.go` · S4: a stat too large told to send a smaller max_lines · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e
- 2026-10-02 · 2e258eb* · mutant killed · exit 1 · `internal/mcp/ack.go` · S4: a replaced version's spans held under the observed sha · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e

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
- 2026-10-02 · 57e6085* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:0 · test-lock-sha256:cc172b8cef5afe8580aee48f143599cd2cfa460b078a749490a6debbdcb248db · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JVGVzdE1yd1JlYWRUYWtlc01heExpbmVzU3RhdEFuZEZpbGVzRnJvbQk5OTZmMGU5YzVjYmNiNzM1YzI5MjY1YjdlMjQyNTcyYjIxNjdmNzAxMDAzNTE0N2QwZTUyZjdhMzk2MDVjNTg2CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBjYXBwZWQgcmVhZCB0b28gbGFyZ2UgZm9yIG9uZSBhbnN3ZXIgaXMgcmVmdXNlZCwgbm90IHBhZ2VkIHBhc3QgdGhlIGNhcAk5MGZjMzM4ZmZhOWM1MTI1NTJlMDI2MzQ4YzI3NGNlYzlkY2I3N2MzYWE3Y2JjZjZhMDNmMjViYzZhZTgzYzRmCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBuZWdhdGl2ZSBtYXhfbGluZXMgaXMgcmVmdXNlZAk0ODI2Mzc2N2JjNGEwZDkyMGViYjY2NjhkZDc1MjM3Mjk0NjI5ZjVkYjEyZDdmZTYzN2E1NzY0NDFjNzY0NjRkCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JZmlsZXNfZnJvbSBpcyByZWZ1c2VkIHdoZXJlIGl0IGNhbm5vdCBiZSBob25vdXJlZAlkNzU0YzdhMDJmYTllMThjNTA5ODU4Y2NkMTgwOTRhMzY4Yzk2MjdmNTQ3MzMxZWNhNDFmZTFiNjA3ZDI1MGU5CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JZmlsZXNfZnJvbSByZWFkcyBhIGxpc3Qgb2Ygc3BlY3MJYTUxMmQ1MmI0Zjg3YjBkOTg2OGE5MzRkNDE5OTJkNGQyNzE0MGZiNGZlZGJjNjAwZDdjOGE0M2ZhNzFkM2U1NQpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCW1heF9saW5lcyAwIHNlcnZlcyBoZWFkZXJzIG9ubHkJMDk4Njc5N2IzNDFhMDM5Yzg0YTk3OTdjODEwZjJlYzBiNDY2NzIzNzQ2YzAzYjlhOWUzNmQ4YzlmN2FhNjAwNApib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCW1heF9saW5lcyBzZXJ2ZXMgTiBsaW5lcyBhbmQgbGljZW5zZXMgb25seSB0aG9zZQlhOWFlMjM0NTYxNzEwNjdmZTFkZWVlYmE3MzEwNWZkZWE2MzFmYTlhOTVkZDJmZmFjMTViNjViNjU1ODZiY2MzCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28Jc3RhdCBzZXJ2ZXMgbm8gbGluZSBhbmQgbGljZW5zZXMgbm90aGluZwk5ZjZkMWQ1MTQzNjdmZDlmODg0NTdkNzJjNzM4ZGMzM2NjOGM5ZDhiY2Y4ZTMwMDQxNDNkM2M3YTI5MjI3MzQw · test-lock-kind:replace
- 2026-10-02 · fe43b6e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:46291
- 2026-10-02 · fe43b6e* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:54584
- 2026-10-02 · fe43b6e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:0 · test-lock-sha256:ee3a72f35c13ab96ce0ae17b270a781209b4c719124c50651dfef6258d9f0512 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JVGVzdE1yd1JlYWRUYWtlc01heExpbmVzU3RhdEFuZEZpbGVzRnJvbQkxNGU2ODY4YmFhYjRiMTM2YjkzOTNmNmVhYWM1YmNjZjM3NTZkOGExZTg1YzFmZjZiNzlmMjA3OTM2MTc0MjIxCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBjYXBwZWQgZ3JlcCB0b28gbGFyZ2UgdG8gc2VydmUgYW5zd2VycyB3aXRoIGl0cyBpbmRleAk4NWE3ZmY1MzhjY2M2YTMwZGU0NmMxMzNiYjA0NWZkZjM2YTM2NmExYjMxOTU5MDEyOGVjODNmYTVhNjAxOWE3CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBjYXBwZWQgcmVhZCB0b28gbGFyZ2UgZm9yIG9uZSBhbnN3ZXIgaXMgcmVmdXNlZCwgbm90IHBhZ2VkIHBhc3QgdGhlIGNhcAk5MGZjMzM4ZmZhOWM1MTI1NTJlMDI2MzQ4YzI3NGNlYzlkY2I3N2MzYWE3Y2JjZjZhMDNmMjViYzZhZTgzYzRmCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBsaXN0IG5hbWluZyBvbmUgZmlsZSB0d2ljZSBsaWNlbnNlcyBib3RoIHJhbmdlcwkzZWQwYzFkNDJlNDJiZGQ5OTUyYjY0NTkwNTNjYmYyMmM4MmM0MmM0OTE3MjllYTgxODE4MzVjMmQzZjc4ZjY2CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBuZWdhdGl2ZSBtYXhfbGluZXMgaXMgcmVmdXNlZAk0ODI2Mzc2N2JjNGEwZDkyMGViYjY2NjhkZDc1MjM3Mjk0NjI5ZjVkYjEyZDdmZTYzN2E1NzY0NDFjNzY0NjRkCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JYSBzdGF0IHRvbyBsYXJnZSBuYW1lcyBmZXdlciBzcGVjcywgYW5kIGEgbG93ZXIgYm91bmQJYTE1NGIxNmViY2U3OTcyZWYxMjgxNWYzNmIyY2NkMjE1MDhiZWQ1MzRjYzE5OTAwNGU1ZTc4ZThlNTc4ZmUzZQpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWZpbGVzX2Zyb20gaXMgcmVmdXNlZCB3aGVyZSBpdCBjYW5ub3QgYmUgaG9ub3VyZWQJYzIxMWVhNGY0YWJkYjlmYWJhNGJmYzA0ZmQ4MjBkMTA5MWU5NGEzYTQ3YzBiNjUxYWJhZGMzYTlkMjA3YjcxYwpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWZpbGVzX2Zyb20gcmVhZHMgYSBsaXN0IG9mIHNwZWNzCWE1MTJkNTJiNGY4N2IwZDk4NjhhOTM0ZDQxOTkyZDRkMjcxNDBmYjRmZWRiYzYwMGQ3YzhhNDNmYTcxZDNlNTUKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwltYXhfbGluZXMgMCBzZXJ2ZXMgaGVhZGVycyBvbmx5CTA5ODY3OTdiMzQxYTAzOWM4NGE5Nzk3YzgxMGYyZWMwYjQ2NjcyMzc0NmMwM2I5YTllMzZkOGM5ZjdhYTYwMDQKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwltYXhfbGluZXMgc2VydmVzIE4gbGluZXMgYW5kIGxpY2Vuc2VzIG9ubHkgdGhvc2UJYTlhZTIzNDU2MTcxMDY3ZmUxZGVlZWJhNzMxMDVmZGVhNjMxZmE5YTk1ZGQyZmZhYzE1YjY1YjY1NTg2YmNjMwpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCXN0YXQgc2VydmVzIG5vIGxpbmUgYW5kIGxpY2Vuc2VzIG5vdGhpbmcJOWY2ZDFkNTE0MzY3ZmQ5Zjg4NDU3ZDcyYzczOGRjMzNjYzhjOWQ4YmNmOGUzMDA0MTQzZDNjN2EyOTIyNzM0MA · test-lock-kind:replace
- 2026-10-02 · 2e258eb* · exit 0 · `set -o pipefail …` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:40524
- 2026-10-02 · 2e258eb* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:06924b038ef577e7143d32aba5a5f60093b977bda1c9d29fe7bee9446238d97e · ms:0 · test-lock-sha256:e4ac76dcecc38436911c0481dc8d41c8443874f6fdbd773a59051048e68a3e3a · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL2hlbGRzcGFuczExN190ZXN0LmdvCVRlc3RBU3BhbkZyb21BUmVwbGFjZWRWZXJzaW9uSXNOb3RIZWxkCWM0OTQ5MjRhMzYwYTc4MTgxZTI0Nzc2NGQxNjZiOTUyZWZiMDk4YTE0YTY4YmE1NDU0YjA2OTMwYWE4MjhjNDYKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlUZXN0TXJ3UmVhZFRha2VzTWF4TGluZXNTdGF0QW5kRmlsZXNGcm9tCTE0ZTY4NjhiYWFiNGIxMzZiOTM5M2Y2ZWFhYzViY2NmMzc1NmQ4YTFlODVjMWZmNmI3OWYyMDc5MzYxNzQyMjEKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlhIGNhcHBlZCBncmVwIHRvbyBsYXJnZSB0byBzZXJ2ZSBhbnN3ZXJzIHdpdGggaXRzIGluZGV4CTg1YTdmZjUzOGNjYzZhMzBkZTQ2YzEzM2JiMDQ1ZmRmMzZhMzY2YTFiMzE5NTkwMTI4ZWM4M2ZhNWE2MDE5YTcKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlhIGNhcHBlZCByZWFkIHRvbyBsYXJnZSBmb3Igb25lIGFuc3dlciBpcyByZWZ1c2VkLCBub3QgcGFnZWQgcGFzdCB0aGUgY2FwCTkwZmMzMzhmZmE5YzUxMjU1MmUwMjYzNDhjMjc0Y2VjOWRjYjc3YzNhYTdjYmNmNmEwM2YyNWJjNmFlODNjNGYKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlhIGxpc3QgbmFtaW5nIG9uZSBmaWxlIHR3aWNlIGxpY2Vuc2VzIGJvdGggcmFuZ2VzCTNlZDBjMWQ0MmU0MmJkZDk5NTJiNjQ1OTA1M2NiZjIyYzgyYzQyYzQ5MTcyOWVhODE4MTgzNWMyZDNmNzhmNjYKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlhIG5lZ2F0aXZlIG1heF9saW5lcyBpcyByZWZ1c2VkCTQ4MjYzNzY3YmM0YTBkOTIwZWJiNjY2OGRkNzUyMzcyOTQ2MjlmNWRiMTJkN2ZlNjM3YTU3NjQ0MWM3NjQ2NGQKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlhIHN0YXQgdG9vIGxhcmdlIG5hbWVzIGZld2VyIHNwZWNzLCBhbmQgYSBsb3dlciBib3VuZAlhMTU0YjE2ZWJjZTc5NzJlZjEyODE1ZjM2YjJjY2QyMTUwOGJlZDUzNGNjMTk5MDA0ZTVlNzhlOGU1NzhmZTNlCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JZmlsZXNfZnJvbSBpcyByZWZ1c2VkIHdoZXJlIGl0IGNhbm5vdCBiZSBob25vdXJlZAljMjExZWE0ZjRhYmRiOWZhYmE0YmZjMDRmZDgyMGQxMDkxZTk0YTNhNDdjMGI2NTFhYmFkYzNhOWQyMDdiNzFjCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JZmlsZXNfZnJvbSByZWFkcyBhIGxpc3Qgb2Ygc3BlY3MJYTUxMmQ1MmI0Zjg3YjBkOTg2OGE5MzRkNDE5OTJkNGQyNzE0MGZiNGZlZGJjNjAwZDdjOGE0M2ZhNzFkM2U1NQpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCW1heF9saW5lcyAwIHNlcnZlcyBoZWFkZXJzIG9ubHkJMDk4Njc5N2IzNDFhMDM5Yzg0YTk3OTdjODEwZjJlYzBiNDY2NzIzNzQ2YzAzYjlhOWUzNmQ4YzlmN2FhNjAwNApib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCW1heF9saW5lcyBzZXJ2ZXMgTiBsaW5lcyBhbmQgbGljZW5zZXMgb25seSB0aG9zZQlhOWFlMjM0NTYxNzEwNjdmZTFkZWVlYmE3MzEwNWZkZWE2MzFmYTlhOTVkZDJmZmFjMTViNjViNjU1ODZiY2MzCmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28Jc3RhdCBzZXJ2ZXMgbm8gbGluZSBhbmQgbGljZW5zZXMgbm90aGluZwk5ZjZkMWQ1MTQzNjdmZDlmODg0NTdkNzJjNzM4ZGMzM2NjOGM5ZDhiY2Y4ZTMwMDQxNDNkM2M3YTI5MjI3MzQw · test-lock-kind:replace
