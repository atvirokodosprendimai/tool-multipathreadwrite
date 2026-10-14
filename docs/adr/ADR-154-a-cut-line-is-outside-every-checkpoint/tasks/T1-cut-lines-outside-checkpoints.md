# Task ADR-154-T1: max_cols on mrw_read, and checkpoints that bracket only recorded lines

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `readArgs.MaxCols` in `internal/mcp/tools.go`; `interleave`'s licensed-line predicate in `internal/mcp/ack.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `max_cols reaches the read`, `a cut line is in no checkpoint`, `the marker text is not taken for a cut`, `a page is cut and leaves its cut lines out`, `a negative max_cols is refused`, `the answer names the cut lines`, `the docs drop CLI only`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `max_cols` decoded, validated and passed to `read.Run` in `readTool` and `firstPage` |
| `internal/mcp/mcp.go` | edit | `max_cols` in `mrw_read`'s input schema |
| `internal/mcp/ack.go` | edit | `interleave` takes which lines the read recorded; `markServed` passes the observation |
| `internal/mcp/maxcols154_test.go` | add | the tests |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerate | the schema gains one property |
| `scripts/contract.sh` | edit | §258 drives the built binary |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | `--max-cols` is no longer CLI only |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestACutLineIsOutsideEveryCheckpoint` (a fitting read with `max_cols` 20 cuts line 2; no checkpoint's span contains 2; the answer names line 2; after acknowledging every id, a write to line 2 is refused as not read and a write to line 3 applies); `TestALineThatSaysColsIsNotTakenForACutOne` (a short line whose content is `[cols 1-5 of 9]` is bracketed and its write applies); `TestANegativeMaxColsIsRefused`; `TestAPageCutByMaxColsLeavesTheCutLinesOut` (an oversized read paged under a small ceiling is cut, and its cut lines are in no checkpoint).
2. [S2] `max_cols` decoded, validated and passed to `read.Run` in `readTool`; the schema entry. [proof: mutation]
3. [S3] `interleave` ends a checkpoint at a line the observation does not cover, and `markServed` passes the observation of the slice's version. [proof: mutation]
4. [S4] The `-- not licensed:` line naming the cut lines. [proof: mutation]
5. [S5] `firstPage` cuts with the same width and the same predicate. [proof: mutation]
6. [S6] AGENTS.md, README.md, BACKLOG. [proof: mutation]
7. [S7] Contract §258: through `mrw mcp`, a cut line is in no checkpoint and its write is refused after every ack, while a line served whole is written. [proof: human: the executor runs ./scripts/contract.sh once and reads the §258 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestACutLineIsOutsideEveryCheckpoint|TestALineThatSaysColsIsNotTakenForACutOne|TestANegativeMaxColsIsRefused|TestAPageCutByMaxColsLeavesTheCutLinesOut|TestMrwReadTakesMaxLinesStatAndFilesFrom' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestACutLineIsOutsideEveryCheckpoint \(' "$out" \
  && grep -qE '^--- PASS: TestALineThatSaysColsIsNotTakenForACutOne \(' "$out" \
  && grep -qE '^--- PASS: TestANegativeMaxColsIsRefused \(' "$out" \
  && grep -qE '^--- PASS: TestAPageCutByMaxColsLeavesTheCutLinesOut \(' "$out" \
  && grep -qE '^--- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom \(' "$out" \
  && grep -q '^# 258\. ' scripts/contract.sh \
  && ! grep -q 'max-cols N` (CLI only' AGENTS.md \
  && grep -q 'mrw_read` takes it as `max_cols`' AGENTS.md \
  && grep -q '`max_cols` on `mrw_read`' README.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACutLineIsOutsideEveryCheckpoint` | `internal/mcp/maxcols154_test.go` | the cut line is in no checkpoint, named, and unwritable after every ack; a whole line is writable | none | S1, S2, S3, S4 |
| `TestALineThatSaysColsIsNotTakenForACutOne` | `internal/mcp/maxcols154_test.go` | the marker text in a file's content does not cost the line its licence | none | S1, S3 |
| `TestANegativeMaxColsIsRefused` | `internal/mcp/maxcols154_test.go` | a negative width is refused | none | S1, S2 |
| `TestAPageCutByMaxColsLeavesTheCutLinesOut` | `internal/mcp/maxcols154_test.go` | a first page is cut and its cut lines are in no checkpoint | none | S1, S5 |
| `TestMrwReadTakesMaxLinesStatAndFilesFrom` | `internal/mcp/readargs117_test.go` | ADR-117's arguments still hold | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `readArgs.MaxCols`, `interleave`'s predicate |
| 2 — something selects it | `readTool` and `firstPage` on every `mrw_read`; contract §258 drives `mrw mcp` |
| 3 — the caller can discover it | the input schema; the `-- not licensed:` line; AGENTS.md and README |
| 4 — it is used | nothing measures this; built because the 14-session survey asked for it (2026-10-09) |

## Mutation Log
(empty until execute)
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `internal/mcp/tools.go` · the fitting read ignores max_cols · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:max_cols reaches the read
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `internal/mcp/tools.go` · a negative width is served · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:a negative max_cols is refused
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `internal/mcp/ack.go` · interleave brackets every numbered line · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:a cut line is in no checkpoint
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `internal/mcp/ack.go` · the recorded predicate ignores the observation · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:the marker text is not taken for a cut
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `internal/mcp/ack.go` · the not-licensed line is dropped · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:the answer names the cut lines
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `internal/mcp/tools.go` · firstPage reads uncut · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:a page is cut and leaves its cut lines out
- 2026-10-14 · e5e2e911* · mutant inconclusive · exit 1 · `internal/mcp/tools.go` · firstPage brackets the cut line · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:a page is cut and leaves its cut lines out
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `AGENTS.md` · AGENTS says mrw_read lacks it · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:the docs drop CLI only
- 2026-10-14 · e5e2e911* · mutant killed · exit 1 · `internal/mcp/tools.go` · firstPage brackets the cut line · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:a page is cut and leaves its cut lines out
- 2026-10-14 · 0259ebed · mutant killed · exit 1 · `internal/mcp/tools.go` · a width beside count is served · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · covers:a negative max_cols is refused

## Verification Log
(empty until execute)
- 2026-10-14 · e5e2e911* · exit 1 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:2393 · test-lock-sha256:0f5167a25db04271aa3417fcd89a05b13b17465fc7e8300873559eff2bcead34 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL21heGNvbHMxNTRfdGVzdC5nbwlUZXN0QUN1dExpbmVJc091dHNpZGVFdmVyeUNoZWNrcG9pbnQJZmM4YWI3Yjg2ODRjZDU4OWI3YzI1ZjE0ZDQxMDIzZDRmOTg2YjkyOTA3YmJmOWY0MjdkOGFhMTMwZWZhN2NmNApib2R5CWludGVybmFsL21jcC9tYXhjb2xzMTU0X3Rlc3QuZ28JVGVzdEFMaW5lVGhhdFNheXNDb2xzSXNOb3RUYWtlbkZvckFDdXRPbmUJYTE4NjA1MjQ0ODI5NmJmODc3NmI1MTJlYjRjYTM5M2RhN2E0OTMzYTdjNTQ5ZmM0YjZhMDFmNmRhODFiMjIxZQpib2R5CWludGVybmFsL21jcC9tYXhjb2xzMTU0X3Rlc3QuZ28JVGVzdEFOZWdhdGl2ZU1heENvbHNJc1JlZnVzZWQJZmY4OTlhODA4ZjEwMTMxMDA4ZGZlNGM2Y2UzYzMzMWYzNTI2MTRkN2U4OTQ5NDMxZWZmZWQ3ZDkxNDY2YTAxZQpib2R5CWludGVybmFsL21jcC9tYXhjb2xzMTU0X3Rlc3QuZ28JVGVzdEFQYWdlQ3V0QnlNYXhDb2xzTGVhdmVzVGhlQ3V0TGluZXNPdXQJYWI4NTk2OTQ3NmUxY2NiNjdlNjYzNzQ5MTFjNDFhZmZjNjkwNzRmMGM4YjFjNjg3NzgwM2Y2MGZmYzBiNTVlOQpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCVRlc3RNcndSZWFkVGFrZXNNYXhMaW5lc1N0YXRBbmRGaWxlc0Zyb20JN2Q5N2I0YzA5NDE2ZDU3YjFmMTRiNjFhZGJkYzgzYzA0MmIwYWI2YmY2ODhlMmZkM2QyNmRmNTE1M2M4ZWExYwpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgY2FwcGVkIGdyZXAgdG9vIGxhcmdlIHRvIHNlcnZlIGFuc3dlcnMgd2l0aCBpdHMgaW5kZXgJODVhN2ZmNTM4Y2NjNmEzMGRlNDZjMTMzYmIwNDVmZGYzNmEzNjZhMWIzMTk1OTAxMjhlYzgzZmE1YTYwMTlhNwpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgY2FwcGVkIHJlYWQgdG9vIGxhcmdlIGZvciBvbmUgYW5zd2VyIGlzIHJlZnVzZWQsIG5vdCBwYWdlZCBwYXN0IHRoZSBjYXAJOTBmYzMzOGZmYTljNTEyNTUyZTAyNjM0OGMyNzRjZWM5ZGNiNzdjM2FhN2NiY2Y2YTAzZjI1YmM2YWU4M2M0Zgpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgbGlzdCBuYW1pbmcgb25lIGZpbGUgdHdpY2UgbGljZW5zZXMgYm90aCByYW5nZXMJM2VkMGMxZDQyZTQyYmRkOTk1MmI2NDU5MDUzY2JmMjJjODJjNDJjNDkxNzI5ZWE4MTgxODM1YzJkM2Y3OGY2Ngpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgbmVnYXRpdmUgbWF4X2xpbmVzIGlzIHJlZnVzZWQJNDgyNjM3NjdiYzRhMGQ5MjBlYmI2NjY4ZGQ3NTIzNzI5NDYyOWY1ZGIxMmQ3ZmU2MzdhNTc2NDQxYzc2NDY0ZApib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgc3RhdCB0b28gbGFyZ2UgbmFtZXMgZmV3ZXIgc3BlY3MsIGFuZCBhIGxvd2VyIGJvdW5kCWExNTRiMTZlYmNlNzk3MmVmMTI4MTVmMzZiMmNjZDIxNTA4YmVkNTM0Y2MxOTkwMDRlNWU3OGU4ZTU3OGZlM2UKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlmaWxlc19mcm9tIGlzIHJlZnVzZWQgd2hlcmUgaXQgY2Fubm90IGJlIGhvbm91cmVkCWIwZDhkOGVlODVkN2ZiNmRkMjg0MGM5ZDAxOWU1NTU1MDhjMmQ2MDliYTk3M2RkZmUzNjdjYjM5NzdmMjgyMmEKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlmaWxlc19mcm9tIHJlYWRzIGEgbGlzdCBvZiBzcGVjcwlhNTEyZDUyYjRmODdiMGQ5ODY4YTkzNGQ0MTk5MmQ0ZDI3MTQwZmI0ZmVkYmM2MDBkN2M4YTQzZmE3MWQzZTU1CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JbWF4X2xpbmVzIDAgc2VydmVzIGhlYWRlcnMgb25seQkwOTg2Nzk3YjM0MWEwMzljODRhOTc5N2M4MTBmMmVjMGI0NjY3MjM3NDZjMDNiOWE5ZTM2ZDhjOWY3YWE2MDA0CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JbWF4X2xpbmVzIHNlcnZlcyBOIGxpbmVzIGFuZCBsaWNlbnNlcyBvbmx5IHRob3NlCWE5YWUyMzQ1NjE3MTA2N2ZlMWRlZWViYTczMTA1ZmRlYTYzMWZhOWE5NWRkMmZmYWMxNWI2NWI2NTU4NmJjYzMKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlzdGF0IHNlcnZlcyBubyBsaW5lIGFuZCBsaWNlbnNlcyBub3RoaW5nCTlmNmQxZDUxNDM2N2ZkOWY4ODQ1N2Q3MmM3MzhkYzMzY2M4YzlkOGJjZjhlMzAwNDE0M2QzYzdhMjkyMjczNDA
  ```
  --- last 10 line(s) of stdout (of 41 after folding 41 raw)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/files_from_is_refused_where_it_cannot_be_honoured (0.00s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_list_naming_one_file_twice_licenses_both_ranges (0.02s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_capped_grep_too_large_to_serve_answers_with_its_index (0.05s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_stat_too_large_names_fewer_specs,_and_a_lower_bound (0.18s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_capped_read_too_large_for_one_answer_is_refused,_not_paged_past_the_cap (0.01s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_capped_read_whose_answer,_encoded,_is_too_large_by_its_markers_names_that_size (0.01s)
      --- PASS: TestMrwReadTakesMaxLinesStatAndFilesFrom/a_capped_read_whose_answer,_encoded,_is_too_large_by_its_escaping_names_that_size (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.503s
  FAIL
  ```
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:868
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1155
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1207
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1258
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1179
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1185
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1265
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1199
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1136
- 2026-10-14 · e5e2e911* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1175
- 2026-10-14 · 0259ebed · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:1308
- 2026-10-14 · 0259ebed* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:0 · test-lock-sha256:f460ae63c091f7a0e66e8c75426f01a3e731d7fbf12793ce78026a9ce6acaaa1 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL21heGNvbHMxNTRfdGVzdC5nbwlUZXN0QUN1dExpbmVJc091dHNpZGVFdmVyeUNoZWNrcG9pbnQJZmM4YWI3Yjg2ODRjZDU4OWI3YzI1ZjE0ZDQxMDIzZDRmOTg2YjkyOTA3YmJmOWY0MjdkOGFhMTMwZWZhN2NmNApib2R5CWludGVybmFsL21jcC9tYXhjb2xzMTU0X3Rlc3QuZ28JVGVzdEFMaW5lVGhhdFNheXNDb2xzSXNOb3RUYWtlbkZvckFDdXRPbmUJYTE4NjA1MjQ0ODI5NmJmODc3NmI1MTJlYjRjYTM5M2RhN2E0OTMzYTdjNTQ5ZmM0YjZhMDFmNmRhODFiMjIxZQpib2R5CWludGVybmFsL21jcC9tYXhjb2xzMTU0X3Rlc3QuZ28JVGVzdEFOZWdhdGl2ZU1heENvbHNJc1JlZnVzZWQJMTBjOWFmMjk3OGE0OWRiODgxM2U1MjAwMzRhNzIzZjAzNzViOGU3MDI2M2M4MWFlOWRhMTUwZDk1MTdmMDNhMwpib2R5CWludGVybmFsL21jcC9tYXhjb2xzMTU0X3Rlc3QuZ28JVGVzdEFQYWdlQ3V0QnlNYXhDb2xzTGVhdmVzVGhlQ3V0TGluZXNPdXQJYWI4NTk2OTQ3NmUxY2NiNjdlNjYzNzQ5MTFjNDFhZmZjNjkwNzRmMGM4YjFjNjg3NzgwM2Y2MGZmYzBiNTVlOQpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCVRlc3RNcndSZWFkVGFrZXNNYXhMaW5lc1N0YXRBbmRGaWxlc0Zyb20JN2Q5N2I0YzA5NDE2ZDU3YjFmMTRiNjFhZGJkYzgzYzA0MmIwYWI2YmY2ODhlMmZkM2QyNmRmNTE1M2M4ZWExYwpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgY2FwcGVkIGdyZXAgdG9vIGxhcmdlIHRvIHNlcnZlIGFuc3dlcnMgd2l0aCBpdHMgaW5kZXgJODVhN2ZmNTM4Y2NjNmEzMGRlNDZjMTMzYmIwNDVmZGYzNmEzNjZhMWIzMTk1OTAxMjhlYzgzZmE1YTYwMTlhNwpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgY2FwcGVkIHJlYWQgdG9vIGxhcmdlIGZvciBvbmUgYW5zd2VyIGlzIHJlZnVzZWQsIG5vdCBwYWdlZCBwYXN0IHRoZSBjYXAJOTBmYzMzOGZmYTljNTEyNTUyZTAyNjM0OGMyNzRjZWM5ZGNiNzdjM2FhN2NiY2Y2YTAzZjI1YmM2YWU4M2M0Zgpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgbGlzdCBuYW1pbmcgb25lIGZpbGUgdHdpY2UgbGljZW5zZXMgYm90aCByYW5nZXMJM2VkMGMxZDQyZTQyYmRkOTk1MmI2NDU5MDUzY2JmMjJjODJjNDJjNDkxNzI5ZWE4MTgxODM1YzJkM2Y3OGY2Ngpib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgbmVnYXRpdmUgbWF4X2xpbmVzIGlzIHJlZnVzZWQJNDgyNjM3NjdiYzRhMGQ5MjBlYmI2NjY4ZGQ3NTIzNzI5NDYyOWY1ZGIxMmQ3ZmU2MzdhNTc2NDQxYzc2NDY0ZApib2R5CWludGVybmFsL21jcC9yZWFkYXJnczExN190ZXN0LmdvCWEgc3RhdCB0b28gbGFyZ2UgbmFtZXMgZmV3ZXIgc3BlY3MsIGFuZCBhIGxvd2VyIGJvdW5kCWExNTRiMTZlYmNlNzk3MmVmMTI4MTVmMzZiMmNjZDIxNTA4YmVkNTM0Y2MxOTkwMDRlNWU3OGU4ZTU3OGZlM2UKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlmaWxlc19mcm9tIGlzIHJlZnVzZWQgd2hlcmUgaXQgY2Fubm90IGJlIGhvbm91cmVkCWIwZDhkOGVlODVkN2ZiNmRkMjg0MGM5ZDAxOWU1NTU1MDhjMmQ2MDliYTk3M2RkZmUzNjdjYjM5NzdmMjgyMmEKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlmaWxlc19mcm9tIHJlYWRzIGEgbGlzdCBvZiBzcGVjcwlhNTEyZDUyYjRmODdiMGQ5ODY4YTkzNGQ0MTk5MmQ0ZDI3MTQwZmI0ZmVkYmM2MDBkN2M4YTQzZmE3MWQzZTU1CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JbWF4X2xpbmVzIDAgc2VydmVzIGhlYWRlcnMgb25seQkwOTg2Nzk3YjM0MWEwMzljODRhOTc5N2M4MTBmMmVjMGI0NjY3MjM3NDZjMDNiOWE5ZTM2ZDhjOWY3YWE2MDA0CmJvZHkJaW50ZXJuYWwvbWNwL3JlYWRhcmdzMTE3X3Rlc3QuZ28JbWF4X2xpbmVzIHNlcnZlcyBOIGxpbmVzIGFuZCBsaWNlbnNlcyBvbmx5IHRob3NlCWE5YWUyMzQ1NjE3MTA2N2ZlMWRlZWViYTczMTA1ZmRlYTYzMWZhOWE5NWRkMmZmYWMxNWI2NWI2NTU4NmJjYzMKYm9keQlpbnRlcm5hbC9tY3AvcmVhZGFyZ3MxMTdfdGVzdC5nbwlzdGF0IHNlcnZlcyBubyBsaW5lIGFuZCBsaWNlbnNlcyBub3RoaW5nCTlmNmQxZDUxNDM2N2ZkOWY4ODQ1N2Q3MmM3MzhkYzMzY2M4YzlkOGJjZjhlMzAwNDE0M2QzYzdhMjkyMjczNDA · test-lock-kind:replace
- 2026-10-14 · 0259ebed* · exit 0 · `set -o pipefail …` · acceptance-sha256:01b6c0c6020964bc5b0490667dc5eee1d2be5ef555ade9be526dad15d0d4f9b6 · ms:714
- 2026-10-14 · human-observed · observed by the executor 2026-10-14: S7 ./scripts/contract.sh exit 0 ('contract holds') on the tree rebased onto 100f502; all three §258 rows PASS — the cut line 2 is in no checkpoint and its write is refused after every ack, line 3 beside it is bracketed and written, and without max_cols line 2 is bracketed and written

## Invariants

- Without `max_cols`, the served text and its checkpoints are what they were.
- `internal/read`, `internal/apply` and `internal/seen` do not change.

## Risks

- None beyond the record's.

## Out of Scope

- Paging a read whose cut lines alone overflow the ceiling differently (permanent: boundary: a cut read pages as any read does, ADR-014)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
