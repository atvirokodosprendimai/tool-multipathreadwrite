# Task ADR-164-T2: a plan maker's dry run shows each change

**Depends-on:** T1, T4
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `apply.HunkResult.Before`/`After` and the `apply.Options` switch; `internal/preview` (`preview.Lines`); the `~`/`-`/`+` receipt lines; JSON `hunks[].before`/`hunks[].after`
**Consumes:** `internal/apply/apply.go` as T1 leaves it (T1); `internal/mcp/tools.go`, `internal/mcp/schema.go` and `internal/mcp/testdata/legacy_golden.jsonl` as T4 leaves them (T4)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a one-line change is marked in place`, `both ends of a long span stay visible`, `a change past column 200 is shown`, `a multi-line change shows old then new`, `a rewrite shows it too`, `a plain dry run shows nothing`, `a real write shows nothing`, `the MCP text shows it and elides it first`, `the receipt keys are listed`

## Goal

Decision 2 on both surfaces, for both plan makers.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Options.Preview`; `HunkResult.Before`/`After` (`json:"before,omitempty"`, `json:"after,omitempty"`) from the replaced lines and body `planFile` already holds |
| `internal/preview/preview.go` | add | `Lines(before, after []string, firstLine int) []string`: the `~` form and the `-`/`+` form with Decision 2's windows |
| `internal/preview/preview_test.go` | add | the unit tests |
| `cmd/mrw/main.go` | edit | sets `Options.Preview` for a plan-maker dry run; `report()` prints the lines under the ok row |
| `cmd/mrw/preview164_test.go` | add | the CLI tests |
| `internal/mcp/tools.go` | edit | sets `Options.Preview` for `grep`/`ast_grep` with `dry_run`; `writeReport` prints the lines; the elision ladder drops them first |
| `internal/mcp/schema.go` | edit | descriptions for `hunks[].before`/`after`; the `elided` description's order |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated for the descriptions, as ADR-152 did |
| `internal/mcp/preview164_test.go` | add | the MCP tests |
| `docs/receipts.txt` | edit | `write hunks[].before`, `write hunks[].after`, `mcp_write hunks[].before`, `mcp_write hunks[].after` (a `[]string` leaf carries no `[]`, as `write hunks[].echo`) |
| `internal/guide/guide.go`, `internal/mcp/instructions.go`, `AGENTS.md` | edit | recipe 2's words say the dry run shows each change; the core is untouched, and the MCP handshake stays within its 4096 bytes (ADR-161 Decision 12c) |
| `scripts/contract.sh` | edit | §274 |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestALineChangeIsMarkedInPlace (the `~` form; a double match on one line and a span over 160 characters keep both ends), TestAMultiLineChangeShowsOldThenNew (the `-`/`+` form; two adjacent 300-character lines each changed at column 10 and at column 290 show both changes on each line), TestASubDryRunShowsEachChange (CLI: a `--sub` dry run over a CSS file prints `~` lines with `[-red-]{+blue+}`; `--json` carries `hunks[].before`/`after`; the same on a CRLF file and a file with no final newline), TestARewriteDryRunShowsEachChange (CLI: an `--ast-grep --rewrite` dry run over a two-line span, with `rewrite161_test.go`'s fake ast-grep), TestAPlainDryRunShowsNoPreview (a hand-written plan's dry run and a real `--sub` write print no `~`, `-` or `+` line and carry no `before`), TestAnMCPSubDryRunShowsEachChange (`grep`+`sub` and `ast_grep`+`rewrite` with `dry_run`; under a ceiling that elides, `before`/`after` go first and `elided` names them). [proof: acceptance]
2. [S2] `internal/preview`. [proof: mutation]
3. [S3] `apply` fills `Before`/`After` under `Options.Preview`. [proof: mutation]
4. [S4] The CLI sets the switch for a plan-maker dry run and prints. [proof: mutation]
5. [S5] The MCP sets it, prints, and elides it first; schema descriptions and the golden. [proof: mutation]
6. [S6] `docs/receipts.txt`; recipe 2's words on the three surfaces. [proof: mutation]
7. [S7] Contract §274: through the built binary, a `--sub --dry-run` prints the `~` line and writes nothing; the pair: a plain plan's dry run prints none. [proof: human: the executor runs ./scripts/contract.sh once and reads the §274 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/preview/ ./cmd/mrw/ -count=1 -timeout 300s -run 'TestALineChangeIsMarkedInPlace$|TestASubDryRunShowsEachChange$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestALineChangeIsMarkedInPlace \(' "$out" \
  && grep -qE '^--- PASS: TestASubDryRunShowsEachChange \(' "$out" \
  && go test ./internal/preview/ ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestAMultiLineChangeShowsOldThenNew|TestARewriteDryRunShowsEachChange|TestAPlainDryRunShowsNoPreview|TestAnMCPSubDryRunShowsEachChange|TestDryRunPrintsParsedHunkBodies|TestASubWriteAppliesThePlanInOneCommand|TestAnMCPSubWriteAppliesThePlanInOneCall|TestNoShippedReceiptFieldDisappears|TestNoShippedMCPReceiptFieldDisappears|TestEveryOutputSchemaPropertyIsDescribed|TestALegacyResultIsUnchangedByTheModernPath' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAMultiLineChangeShowsOldThenNew \(' "$out" \
  && grep -qE '^--- PASS: TestARewriteDryRunShowsEachChange \(' "$out" \
  && grep -qE '^--- PASS: TestAPlainDryRunShowsNoPreview \(' "$out" \
  && grep -qE '^--- PASS: TestAnMCPSubDryRunShowsEachChange \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep -x 'write hunks\[\].before' docs/receipts.txt > /tmp/adr164-t2-key.txt \
  && grep '^# 274\. ' scripts/contract.sh > /tmp/adr164-t2-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestDryRunPrintsParsedHunkBodies` | `cmd/mrw/dryrun_parsed_test.go` | the `parsed:` lines unchanged | none | — |
| `TestASubWriteAppliesThePlanInOneCommand` | `cmd/mrw/sub161_test.go` | ADR-161's summary line unchanged | none | — |
| `TestALineChangeIsMarkedInPlace` | `internal/preview/preview_test.go` | the `~` form: a double match on one line, a span over 160 characters and a change past column 200 stay visible | none | S1, S2 |
| `TestAMultiLineChangeShowsOldThenNew` | `internal/preview/preview_test.go` | the `-`/`+` form: two adjacent 300-character lines changed at columns 10 and 290 show both changes on each | none | S1, S2 |
| `TestASubDryRunShowsEachChange` | `cmd/mrw/preview164_test.go` | a `--sub` dry run prints the `~` rows and `--json` carries before/after, on a CRLF file and a file with no final newline | none | S1, S3, S4 |
| `TestARewriteDryRunShowsEachChange` | `cmd/mrw/preview164_test.go` | an `--ast-grep --rewrite` dry run over a three-line span prints the `-`/`+` rows | none | S1, S3, S4 |
| `TestAPlainDryRunShowsNoPreview` | `cmd/mrw/preview164_test.go` | a hand-written plan's dry run and a real `--sub` write print no row and carry no before | none | S1, S4 |
| `TestAnMCPSubDryRunShowsEachChange` | `internal/mcp/preview164_test.go` | `grep`+`sub` and `ast_grep`+`rewrite` with `dry_run` show each change; under a ceiling the preview goes first and `elided` names it | none | S1, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `preview.Lines`, `HunkResult.Before`/`After` |
| 2 — something selects it | `Options.Preview`, set by the CLI and `mrw_write` only for a plan maker's dry run; contract §274 drives the binary |
| 3 — the caller can discover it | recipe 2 in `mrw instructions`, the MCP handshake and AGENTS.md; the schema descriptions |
| 4 — it is used | nothing measures this; asked for by one field report (2026-10-20) |

## Mutation Log
(empty until execute)
- 2026-10-20 · df4d511a* · mutant killed · exit 1 · `internal/preview/preview.go` · a one-line change loses its in-place ~ row · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16
- 2026-10-20 · df4d511a* · mutant killed · exit 1 · `internal/apply/apply.go` · apply never fills Before/After · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16
- 2026-10-20 · df4d511a* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never asks for the preview · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16
- 2026-10-20 · df4d511a* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write never asks for the preview · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16
- 2026-10-20 · df4d511a* · mutant killed · exit 1 · `docs/receipts.txt` · the write receipt key hunks[].before is not listed · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16
- 2026-10-20 · d9224ca1* · mutant killed · exit 1 · `internal/apply/shape.go` · a hunk demoted to skipped keeps its preview (Codex #468 P2) · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16

## Verification Log
(empty until execute)
- 2026-10-20 · df4d511a* · exit 1 · `set -o pipefail …` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:1783 · test-lock-sha256:dc19d3ed4bfa8419088f78e3ee7f6bdacc335bea13b3090086d847313efa2d00 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9kcnlydW5fcGFyc2VkX3Rlc3QuZ28JVGVzdERyeVJ1blByaW50c1BhcnNlZEh1bmtCb2RpZXMJNGY2Mjg3ZTk2MzQwZTRjNmMxMTBhZDExNWNhYmMzYjliMmQ5YTg2Y2ExM2FiODc3MzM5OWUyNzQwY2IzYjlkNgpib2R5CWNtZC9tcncvcHJldmlldzE2NF90ZXN0LmdvCVRlc3RBUGxhaW5EcnlSdW5TaG93c05vUHJldmlldwkwYzIwN2I5MGIyYWUyZDI5N2NhM2MzNWMxYzlhNjZjNDgzYTc0MzM5OTlhYTM4YWVlNDg3YWVkNjFlNDQ0NTU3CmJvZHkJY21kL21ydy9wcmV2aWV3MTY0X3Rlc3QuZ28JVGVzdEFSZXdyaXRlRHJ5UnVuU2hvd3NFYWNoQ2hhbmdlCWZiZDRmNTlkMzhmZTg3YjI3M2YxYTlhZTkyNWVhNWU3NGYyYjY2ODhhMDU4ZTFiMWUwNjIyZjI0MjE0ODc1ZjAKYm9keQljbWQvbXJ3L3ByZXZpZXcxNjRfdGVzdC5nbwlUZXN0QVN1YkRyeVJ1blNob3dzRWFjaENoYW5nZQk2YjgzMWFmYjI0ODE4ZGU4MjUyYWU2NTk0MTkwNWM3MDFkNjBhNzZhZjZhNmJiZTY3ZGVlNmY3Mjc0MThlOWJmCmJvZHkJY21kL21ydy9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YlJlYWRSZWZ1c2Fsc0FuZEV4aXRzCWVhYTg2YmJiNWYwMzFlOTI5ZmFhNTNmOTM2YjhjOWQ4ZTA2NzdhZWZkZDhlYjg3NDI5NzFkYTUwMTk2ZTZjMjEKYm9keQljbWQvbXJ3L3N1YjE2MV90ZXN0LmdvCVRlc3RBU3ViV3JpdGVBcHBsaWVzVGhlUGxhbkluT25lQ29tbWFuZAk3ZDYwYTQ1MzEyMTE3YmY2ZDU4NmU5YmVjNTY1ZWUyNjE2NzhhNmFiYThlZDM0NDI2NzE5MjEzM2JjODQzYzBkCmJvZHkJY21kL21ydy9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YldyaXRlTmV2ZXJSZWFkc0FGaWxlTmFtZUFzQVBvaW50ZXIJZTMyMDQ1ZGEzZjNjMmJiYjRmZThlYWQ0ODJhZWFlNTEwOTk3YzdmMDM4NmQ5YjM4Mjc4MGJmNzE2ZjNlNjkyMwpib2R5CWludGVybmFsL21jcC9wcmV2aWV3MTY0X3Rlc3QuZ28JVGVzdEFuTUNQU3ViRHJ5UnVuU2hvd3NFYWNoQ2hhbmdlCWQ0NzlmYzkzY2Q5YmNjMjM4ODViNTczZmRmNWUyNDdiYTk4YzI1MDUwZjRhZDg0M2U3ZDZjM2Q1NWY0ZDc5MWIKYm9keQlpbnRlcm5hbC9tY3AvcHJldmlldzE2NF90ZXN0LmdvCWFzdF9ncmVwIGFuZCByZXdyaXRlCTMwNDJhODg4NTRmNjllMzgxOWM4ZDJlMDU5YWQ0ZmU0N2U4N2Y1NjE0ZjVkYjkzMDQ4ZmQxNmM1NTMwOWVmYWUKYm9keQlpbnRlcm5hbC9tY3AvcHJldmlldzE2NF90ZXN0LmdvCWdyZXAgYW5kIHN1YglmNjc1YmMwYzgyNjVmN2ZlNmNlMTczM2M1MDE4NTgxNzE3MjVmMGJhODIwMDc1YTdjNGQzMWI1ZWQxNDM0YjU1CmJvZHkJaW50ZXJuYWwvbWNwL3ByZXZpZXcxNjRfdGVzdC5nbwl1bmRlciBhIGNlaWxpbmcgdGhlIHByZXZpZXcgZ29lcyBmaXJzdCBhbmQgZWxpZGVkIG5hbWVzIGl0CTc3NDM2NWYxZTNkYzRkMjBkNmVkYzM4Yzg1ZTc3M2FhMDhmMzhiZjBiNDQ5ZmEwZTNmMzVjNTU2N2EzYzBhMDUKYm9keQlpbnRlcm5hbC9wcmV2aWV3L3ByZXZpZXdfdGVzdC5nbwlUZXN0QUxpbmVDaGFuZ2VJc01hcmtlZEluUGxhY2UJZGFmODkyMThiYTdjYWI1NmY2ZjFkMmZjMzAzNzQxODQ5YThlMGNhODFmMmE2NDM1NGJlYjFmZGM4Zjc1MmUxMQpib2R5CWludGVybmFsL3ByZXZpZXcvcHJldmlld190ZXN0LmdvCVRlc3RBTXVsdGlMaW5lQ2hhbmdlU2hvd3NPbGRUaGVuTmV3CWRmODJlNTgwZDM4MTY5NTNjNDhlODkyNDgwZGQwYjUwNWVhMmEzZTgzMWM5YjMyZWFjMTEzN2I1M2U4ZTRiMzE
  ```
  --- last 10 line(s) of stdout (of 90 after folding 91 raw)
          5 hunk(s), 4 file(s), 0 failed, 0 advisories — dry run, nothing written
      preview164_test.go:79: a.css: before [] after [], want one line each  (x2)
      preview164_test.go:79: crlf.css: before [] after [], want one line each
      preview164_test.go:79: last.css: before [] after [], want one line each
      preview164_test.go:79: twice.go: before [] after [], want one line each
      preview164_test.go:88: hunks with before/after for map[], want all four files
  --- FAIL: TestASubDryRunShowsEachChange (0.57s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.817s
  FAIL
  ```
- 2026-10-20 · df4d511a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:5351
- 2026-10-20 · df4d511a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:4978
- 2026-10-20 · df4d511a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:4954
- 2026-10-20 · df4d511a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:5268
- 2026-10-20 · df4d511a* · exit 0 · `set -o pipefail …` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:5114
- 2026-10-20 · human-observed · Claude observed: ./scripts/contract.sh exit 0 at the T2 tree on df4d511a (2026-10-20), §274 all 4 rows PASS: a --sub dry run exits 0 and prints the changed line marked in place with nothing written; the pair: a plain plan's dry run prints no ~, - or + row · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16
- 2026-10-20 · d9224ca1* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:0 · test-lock-sha256:0b3e130a463df934bbda99a4b65b64c89d73f94232a005b7e540c73c189779d0 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9kcnlydW5fcGFyc2VkX3Rlc3QuZ28JVGVzdERyeVJ1blByaW50c1BhcnNlZEh1bmtCb2RpZXMJNGY2Mjg3ZTk2MzQwZTRjNmMxMTBhZDExNWNhYmMzYjliMmQ5YTg2Y2ExM2FiODc3MzM5OWUyNzQwY2IzYjlkNgpib2R5CWNtZC9tcncvcHJldmlldzE2NF90ZXN0LmdvCVRlc3RBUGxhaW5EcnlSdW5TaG93c05vUHJldmlldwkwYmU2ZGVhZjkwMTAyOTRiOTc0MjhhYTQzZWEzN2I3NGQ5MGQ4YmM3MWQxYTQ4YTQwOWNjODEwZDkwZjAwNzAzCmJvZHkJY21kL21ydy9wcmV2aWV3MTY0X3Rlc3QuZ28JVGVzdEFSZXdyaXRlRHJ5UnVuU2hvd3NFYWNoQ2hhbmdlCWZiZDRmNTlkMzhmZTg3YjI3M2YxYTlhZTkyNWVhNWU3NGYyYjY2ODhhMDU4ZTFiMWUwNjIyZjI0MjE0ODc1ZjAKYm9keQljbWQvbXJ3L3ByZXZpZXcxNjRfdGVzdC5nbwlUZXN0QVN1YkRyeVJ1blNob3dzRWFjaENoYW5nZQk2YjgzMWFmYjI0ODE4ZGU4MjUyYWU2NTk0MTkwNWM3MDFkNjBhNzZhZjZhNmJiZTY3ZGVlNmY3Mjc0MThlOWJmCmJvZHkJY21kL21ydy9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YlJlYWRSZWZ1c2Fsc0FuZEV4aXRzCWVhYTg2YmJiNWYwMzFlOTI5ZmFhNTNmOTM2YjhjOWQ4ZTA2NzdhZWZkZDhlYjg3NDI5NzFkYTUwMTk2ZTZjMjEKYm9keQljbWQvbXJ3L3N1YjE2MV90ZXN0LmdvCVRlc3RBU3ViV3JpdGVBcHBsaWVzVGhlUGxhbkluT25lQ29tbWFuZAk3ZDYwYTQ1MzEyMTE3YmY2ZDU4NmU5YmVjNTY1ZWUyNjE2NzhhNmFiYThlZDM0NDI2NzE5MjEzM2JjODQzYzBkCmJvZHkJY21kL21ydy9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YldyaXRlTmV2ZXJSZWFkc0FGaWxlTmFtZUFzQVBvaW50ZXIJZTMyMDQ1ZGEzZjNjMmJiYjRmZThlYWQ0ODJhZWFlNTEwOTk3YzdmMDM4NmQ5YjM4Mjc4MGJmNzE2ZjNlNjkyMwpib2R5CWludGVybmFsL21jcC9wcmV2aWV3MTY0X3Rlc3QuZ28JVGVzdEFuTUNQU3ViRHJ5UnVuU2hvd3NFYWNoQ2hhbmdlCWQ0NzlmYzkzY2Q5YmNjMjM4ODViNTczZmRmNWUyNDdiYTk4YzI1MDUwZjRhZDg0M2U3ZDZjM2Q1NWY0ZDc5MWIKYm9keQlpbnRlcm5hbC9tY3AvcHJldmlldzE2NF90ZXN0LmdvCWFzdF9ncmVwIGFuZCByZXdyaXRlCTMwNDJhODg4NTRmNjllMzgxOWM4ZDJlMDU5YWQ0ZmU0N2U4N2Y1NjE0ZjVkYjkzMDQ4ZmQxNmM1NTMwOWVmYWUKYm9keQlpbnRlcm5hbC9tY3AvcHJldmlldzE2NF90ZXN0LmdvCWdyZXAgYW5kIHN1YglmNjc1YmMwYzgyNjVmN2ZlNmNlMTczM2M1MDE4NTgxNzE3MjVmMGJhODIwMDc1YTdjNGQzMWI1ZWQxNDM0YjU1CmJvZHkJaW50ZXJuYWwvbWNwL3ByZXZpZXcxNjRfdGVzdC5nbwl1bmRlciBhIGNlaWxpbmcgdGhlIHByZXZpZXcgZ29lcyBmaXJzdCBhbmQgZWxpZGVkIG5hbWVzIGl0CTc3NDM2NWYxZTNkYzRkMjBkNmVkYzM4Yzg1ZTc3M2FhMDhmMzhiZjBiNDQ5ZmEwZTNmMzVjNTU2N2EzYzBhMDUKYm9keQlpbnRlcm5hbC9wcmV2aWV3L3ByZXZpZXdfdGVzdC5nbwlUZXN0QUxpbmVDaGFuZ2VJc01hcmtlZEluUGxhY2UJZGFmODkyMThiYTdjYWI1NmY2ZjFkMmZjMzAzNzQxODQ5YThlMGNhODFmMmE2NDM1NGJlYjFmZGM4Zjc1MmUxMQpib2R5CWludGVybmFsL3ByZXZpZXcvcHJldmlld190ZXN0LmdvCVRlc3RBTXVsdGlMaW5lQ2hhbmdlU2hvd3NPbGRUaGVuTmV3CWRmODJlNTgwZDM4MTY5NTNjNDhlODkyNDgwZGQwYjUwNWVhMmEzZTgzMWM5YjMyZWFjMTEzN2I1M2U4ZTRiMzE · test-lock-kind:replace
- 2026-10-20 · d9224ca1* · exit 0 · `set -o pipefail …` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:6305
- 2026-10-29 · 41a53637* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:0304d71e34802e08a51cb59eaa258123f7dd110369b874fbc19e5f75f7220f16 · ms:0 · test-lock-sha256:cc77f4a2f7be0a5ce6ae77b3b7c67887c3d0ea36375356980022da65dee5f7cc · test-lock-file:docs/adr/locks/cc77f4a2f7be0a5ce6ae77b3b7c67887c3d0ea36375356980022da65dee5f7cc.lock · test-lock-kind:replace

## Invariants

- A real write's receipt, a plain plan's dry run and every existing receipt key are unchanged (ADR-111).
- The `parsed:` lines and the summary line are unchanged.

## Risks

- None beyond the record's.

## Out of Scope

- A preview for a plain plan's dry run (permanent: boundary: the record's Alternatives; its author wrote every body line)

## Stop Condition

Stop and bring it to Zy if recipe 2's new words do not fit the MCP handshake's 4096 bytes, or if `planFile` does not hold a replace hunk's replaced lines for every address form a plan maker emits.
