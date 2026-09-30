# Task ADR-105-T1: a failed cleanup is named in `left_behind`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `Result.LeftBehind` (`left_behind`); `removeFn`; the CLI `left behind:` line; the `mrw_write` field and schema
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every failed cleanup is named`, `a directory still holding something is not named`, `every receipt shows the field`, `only the owned engine packages change`

## Goal

Every removal `internal/apply` attempts on something it made in the tree goes through one helper over `removeFn`;
a path still there afterwards is named, root-relative, in `Result.LeftBehind`, and every receipt shows it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `LeftBehind`; `removeFn` and the helper; `discard` (`:659-669`); `stageFile`'s error paths (`:1761-1780`) hand the temp back; `probeName` (`:1703-1713`) |
| `internal/apply/pathop.go` | edit | the placeholder (`:207`, `:210`), the final aside removal (`:283-285`), the undo's kept asides (`:180-186`) |
| `internal/apply/left105_test.go` | add | the engine test |
| `cmd/mrw/main.go` | edit | `report` prints `left behind: <path>` (`:2240-2245`) |
| `cmd/mrw/left105_test.go` | add | the CLI receipt test |
| `internal/mcp/wirepath.go` | edit | `slashResult` slashes `LeftBehind` |
| `internal/mcp/schema.go` | edit | the `left_behind` description |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated for the new schema property |
| `internal/mcp/tools.go` | edit | `boundedReceipt`'s terminal sentences name the leftover count; `floorAt` measures the longest (the review of the record) |
| `internal/mcp/left105_test.go` | add | the MCP test |

## Ordered Steps

1. [S1] Write the failing test `TestEveryFailedCleanupIsNamedInLeftBehind`; confirm RED. [proof: mutation]
2. [S2] Add `removeFn`, the helper and `LeftBehind`; route the 12 sites through it; `stageFile` returns its temp on error so `discard` removes and reports it; a probe is named only when mrw created it and could not remove it (`probeLeftError`), and a path is cleared only when it is known to be gone (the Codex review of #297). [proof: mutation] Mutants: the helper never records; the non-empty-directory exclusion removed; the final aside removal bypasses the helper; a failed stage's temp dropped before the abort; the undo's kept aside not named; a probe that met another process's file named; an uninspectable path cleared.
3. [S3] The receipts: `report` prints `left behind: <path>` on success and failure; `--json` carries the field; `slashResult` slashes it; the schema describes it; a terminal `mrw_write` sentence names the count; the legacy golden is regenerated and diffed structurally (one new property). [proof: mutation] Mutants: `report` skips the line; `slashResult` leaves `LeftBehind` unslashed; the terminal sentence drops the count.

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 600s -run 'TestEveryFailedCleanupIsNamedInLeftBehind|TestTheReceiptsNameWhatWasLeftBehind|TestMrwWriteSpellsLeftBehindWithSlashes' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestEveryFailedCleanupIsNamedInLeftBehind \(' "$out" \
  && grep -qE '^--- PASS: TestTheReceiptsNameWhatWasLeftBehind \(' "$out" \
  && grep -qE '^--- PASS: TestMrwWriteSpellsLeftBehindWithSlashes \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/plan internal/lines internal/rooted internal/read internal/check internal/links \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/lines internal/rooted internal/read internal/check internal/links)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryFailedCleanupIsNamedInLeftBehind` | `internal/apply/left105_test.go` | with `removeFn` refusing chosen paths: a staging abort names the temp it could not remove; a failed stage's own temp is named; a directory mrw made is named only when empty, not when it holds another file; an applied plan whose aside removal fails is applied and names the aside, which holds the removed file; a placeholder that could not be removed is named; a create's probe and a rename destination's probe that could not be removed are named; an aside the undo keeps (through `commitRenameFn`) is named and holds the unlinked file; a path gone despite the error is not named; a probe that met another process's file names nothing and leaves that file alone; a path whose inspection fails (through `lstatFn`) is named; with `removeFn` real, nothing is named | — | S1, S2 |
| `TestTheReceiptsNameWhatWasLeftBehind` | `cmd/mrw/left105_test.go` | `report` prints `left behind: <path>` for each entry on an applied and a failed result, and nothing when the list is empty; the `--json` receipt carries `left_behind` | — | S3 |
| `TestMrwWriteSpellsLeftBehindWithSlashes` | `internal/mcp/left105_test.go` | `slashResult` with `\` spells every `LeftBehind` entry with `/` and leaves the engine's slice unchanged; the output schema describes `left_behind`; at a 64-byte ceiling both terminal sentences (nothing written, and written but unreportable) name the leftover count, and `floorAt` bounds the longer | — | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every cleanup removal in `internal/apply` goes through the helper |
| 3 — the caller can discover it | the receipt field and the `left behind:` line |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/apply/apply.go` · the helper never records what stayed · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every failed cleanup is named
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/apply/apply.go` · a directory still holding a file is named · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:a directory still holding something is not named
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/apply/pathop.go` · the final aside removal bypasses the helper · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every failed cleanup is named
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/apply/apply.go` · a failed stage its temp is dropped before the abort · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every failed cleanup is named
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `cmd/mrw/main.go` · the human receipt skips left behind · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every receipt shows the field
- 2026-09-30 · 1bd8690* · mutant killed · exit 1 · `internal/mcp/wirepath.go` · mrw_write leaves left_behind backslashed · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every receipt shows the field
- 2026-09-30 · 1e77123* · mutant killed · exit 1 · `internal/apply/pathop.go` · the undo's kept aside is not named · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every failed cleanup is named
- 2026-09-30 · 1e77123* · mutant killed · exit 1 · `internal/mcp/tools.go` · the nothing-written sentence drops the leftover count · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every receipt shows the field
- 2026-09-30 · cf45fb0* · mutant killed · exit 1 · `internal/apply/apply.go` · a probe that met another process's file is named · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every failed cleanup is named
- 2026-09-30 · cf45fb0* · mutant inconclusive · exit 1 · `internal/apply/apply.go` · an uninspectable path is cleared · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every failed cleanup is named
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-30 · cf45fb0* · mutant killed · exit 1 · `internal/mcp/tools.go` · the terminal hint drops the probe targets · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every receipt shows the field
- 2026-09-30 · cf45fb0* · mutant killed · exit 1 · `internal/apply/apply.go` · an uninspectable path is cleared (re-expressed: the first form dropped the fs import and did not build) · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · covers:every failed cleanup is named

## Invariants

- A receipt with nothing left behind is byte-identical to before.
- Exit codes keep their meanings; an applied write with a leftover exits 0.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass, or if a site cannot tell a gone path from a kept one.

## Out of Scope

- The other ADR-105 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 1bd8690* · exit 1 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:589 · test-lock-sha256:f67f685cacceeac7b9d8360ae795734a6c75ed947e2059bba0f5cdad1335bfcb · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlUZXN0RXZlcnlGYWlsZWRDbGVhbnVwSXNOYW1lZEluTGVmdEJlaGluZAk4NWYwODBjMzM3MWYwYTU5ZTc4NTczNDQ2ZTM5YjU0MjZmMGYzMGU4MzAwYmQxMGM2ZTljYjUwYmE4YmE3MmNmCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCWEgZGlyZWN0b3J5IGl0IG1hZGUgaXMgbmFtZWQgb25seSB3aGVuIGVtcHR5CWJlMDE4OWU3OWIyZTg2NjUyODA4ZDQwNGRlMzNlZTA0OGU3NmZkMjVlYWRkMWExZGZhYjRmYzAzZTc0YmJlZjQKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBmYWlsZWQgc3RhZ2UgaGFuZHMgaXRzIG93biB0ZW1wIHRvIHRoZSBhYm9ydAk5MzczZjM4MWUxZmZmN2Q0NWE5YjAxZWE0ZGU1OTIxNmI3NGQ4OTU3MWVhMmY0NzM4MTI1NDhiNjUxYTNhYzk3CmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCWEgcGF0aCBnb25lIGRlc3BpdGUgdGhlIGVycm9yIGlzIG5vdCBuYW1lZAkzMmZiOThhNzhmMDJhNDlkZTg3ZGM3MzM3MWE1ZDY0MTA4MGUxMWI1MDQ3NGIzYWE0OGEwZWU3NGViMzFlYTJlCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCWEgcGxhY2Vob2xkZXIgdGhhdCBjb3VsZCBub3QgYmUgcmVtb3ZlZCBpcyBuYW1lZAkyOWVmMmUwMzdkMjM2N2Q5ZWUxODBmMzJmNThiYWU5NDA1OTg5NjUwZmVjZGI0YjAyN2Y2ZmY3NDcyYmFhZTRmCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCWEgcHJvYmUgdGhhdCBjb3VsZCBub3QgYmUgcmVtb3ZlZCBpcyBuYW1lZAliMjE3ZGRlMWE5MGI1ZDQ2MzJlODUwNGFjNjE4MDk4YWVmZmI5YzY3ZGMzMDdhNWEzYWZmMWZjMjlmMDRmZmEwCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCWEgc3RhZ2luZyBhYm9ydCBuYW1lcyB0aGUgdGVtcCBpdCBjb3VsZCBub3QgcmVtb3ZlCTQ0NjgyNDM4NTJkOGM3ZGQyNTNiNGU1MmNhNjY1NWJiMDRkZGZhMDdkZGVmNTYxNGM2ODI3ODk4NDdlZWY1MWUKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYW4gYXBwbGllZCBwbGFuIG5hbWVzIHRoZSBhc2lkZSBpdCBjb3VsZCBub3QgcmVtb3ZlCWMzMWNhODE0Njk5YWVjZDJlYWNlMzVlNDM4Y2FmYmVkMzgzMTA1YjI3Mjc1YzExYzYxMDI3MGMyZjgwMzllOTMKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28Jd2l0aCByZW1vdmFsIHdvcmtpbmcgbm90aGluZyBpcyBuYW1lZAk4NjU2OTVjNTY3NmYzYWQwNzUwNzZiZjMzMTM2OWVhZDJiYmYzOWI4NDgyY2YyZDllMzk1YmExYTIxNWI1MDRkCnVucHJvdmVuCWNtZC9tcncvbGVmdDEwNV90ZXN0LmdvCVRlc3RUaGVSZWNlaXB0c05hbWVXaGF0V2FzTGVmdEJlaGluZAp1bnByb3ZlbglpbnRlcm5hbC9tY3AvbGVmdDEwNV90ZXN0LmdvCVRlc3RNcndXcml0ZVNwZWxsc0xlZnRCZWhpbmRXaXRoU2xhc2hlcw
  ```
  --- last 10 line(s) of stdout (of 23 after folding 23 raw)
  internal/apply/left105_test.go:94:79: res.LeftBehind undefined (type Result has no field or method LeftBehind)
  internal/apply/left105_test.go:94:79: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.071s [no tests to run]
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.163s [no tests to run]
  FAIL
  ```
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:468
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:360
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:372
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:352
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:368
- 2026-09-30 · 1bd8690* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:345
- 2026-09-30 · 1bd8690* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:0 · test-lock-sha256:6b0a9b100368c1ec9897ce3e79d11d5bf34df57af7f1999e2b2e764b781f5aad · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGVmdDEwNV90ZXN0LmdvCVRlc3RUaGVSZWNlaXB0c05hbWVXaGF0V2FzTGVmdEJlaGluZAk2YTJmZDUyMzBmMzc5MDQyN2ExODliYjE2ZGE5OWMxZTRjMTUzN2IwNmExYzhlMWJiNDEyNThkMTJlZjM4NjBhCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCVRlc3RFdmVyeUZhaWxlZENsZWFudXBJc05hbWVkSW5MZWZ0QmVoaW5kCWQ2YTRkMzFlODAyYzNiMzRlNWY0ZjY4ZjQ3MTY4ZDY2Y2JkOTk1NTMwNWY3YmRlMmJlZTc0ODFmMDYxOWI2YTMKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBkaXJlY3RvcnkgaXQgbWFkZSBpcyBuYW1lZCBvbmx5IHdoZW4gZW1wdHkJMWRjYzcyNTNjZjk4YTQxZWNlMTgxZmQ2OWJhMDdjZTliYTZiZTVhODc0N2E0MGJhOTVhNGE2ZTlhODA4YjhiNwpib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhIGZhaWxlZCBzdGFnZSBoYW5kcyBpdHMgb3duIHRlbXAgdG8gdGhlIGFib3J0CTM3ZDNlMmFkYTgxY2I0YzQ3YmRhMTliNWJlNDk1NDQwNmE5ZWU0YzgyNDVhYjhmZmQ5NzZkMWUzMDYzM2IxOWEKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwYXRoIGdvbmUgZGVzcGl0ZSB0aGUgZXJyb3IgaXMgbm90IG5hbWVkCTMyZmI5OGE3OGYwMmE0OWRlODdkYzczMzcxYTVkNjQxMDgwZTExYjUwNDc0YjNhYTQ4YTBlZTc0ZWIzMWVhMmUKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwbGFjZWhvbGRlciB0aGF0IGNvdWxkIG5vdCBiZSByZW1vdmVkIGlzIG5hbWVkCTE1MDczZjQ4NmYxZDZiYzUxNjBiYmNkM2FiNTMxNTlmNDBiMTUyODAxZjA0ODQ1NzFmNGU2ZDk3NjgxNmQ1MTYKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwcm9iZSB0aGF0IGNvdWxkIG5vdCBiZSByZW1vdmVkIGlzIG5hbWVkCTg5MGUzYTU3NjU2MGI2MDIwMGYxMjM0OWY3ZDViZTRlOGM5ZDk0NDZiYjIxZWQ4NDFlZjYyMzdjMWJhMmE2YzgKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBzdGFnaW5nIGFib3J0IG5hbWVzIHRoZSB0ZW1wIGl0IGNvdWxkIG5vdCByZW1vdmUJYTdmM2M4MjFjYTI3NGU4ZWJkYTJjODIyODk0NTAwYzQ0YjY3MDEwNjY4Njk4NWIyZWYwZDQ2YzcyYTM2ZTg5MApib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhbiBhcHBsaWVkIHBsYW4gbmFtZXMgdGhlIGFzaWRlIGl0IGNvdWxkIG5vdCByZW1vdmUJOTNkOWZlOGVmMDJmMGIzZDcwYzQ5MWU4OTgxMDU4OTMyN2I5YzA2OGFjNjNjY2MwN2JlNzMwYmY0MmEwYmJmNApib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwl3aXRoIHJlbW92YWwgd29ya2luZyBub3RoaW5nIGlzIG5hbWVkCTg2NTY5NWM1Njc2ZjNhZDA3NTA3NmJmMzMxMzY5ZWFkMmJiZjM5Yjg0ODJjZjJkOWUzOTViYTFhMjE1YjUwNGQKYm9keQlpbnRlcm5hbC9tY3AvbGVmdDEwNV90ZXN0LmdvCVRlc3RNcndXcml0ZVNwZWxsc0xlZnRCZWhpbmRXaXRoU2xhc2hlcwlkYjk2YzEwZTc2NTA1ZGQ0MTk0ZWFlNjA0ZjRlYmU2NmYyYTU0M2NhZjMzNTEyNjViYmIyMDI3OTFjNGI1NDcw · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: after the red run the engine test's helper onDisk was renamed leftOnDisk (it collided with an existing helper), every assertion kept; the CLI and MCP receipt tests were written after the red run, for S3, and were first locked by this relock; approved
- 2026-09-30 · 1e77123* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:0 · test-lock-sha256:6804a1578a2800988fd22d1f0fbbf8de18f74dd943cc652cb83285d1ff0987aa · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGVmdDEwNV90ZXN0LmdvCVRlc3RUaGVSZWNlaXB0c05hbWVXaGF0V2FzTGVmdEJlaGluZAk2YTJmZDUyMzBmMzc5MDQyN2ExODliYjE2ZGE5OWMxZTRjMTUzN2IwNmExYzhlMWJiNDEyNThkMTJlZjM4NjBhCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCVRlc3RFdmVyeUZhaWxlZENsZWFudXBJc05hbWVkSW5MZWZ0QmVoaW5kCWZiOWQxMjNmZGUwNTAyMTBkNmNkNWQ5MTkxMWI3MjAyY2ExNGU4YmUxYmZlMjNmOTg2NjM3OTU1Zjk5YmFkOTYKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBkaXJlY3RvcnkgaXQgbWFkZSBpcyBuYW1lZCBvbmx5IHdoZW4gZW1wdHkJMWRjYzcyNTNjZjk4YTQxZWNlMTgxZmQ2OWJhMDdjZTliYTZiZTVhODc0N2E0MGJhOTVhNGE2ZTlhODA4YjhiNwpib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhIGZhaWxlZCBzdGFnZSBoYW5kcyBpdHMgb3duIHRlbXAgdG8gdGhlIGFib3J0CTM3ZDNlMmFkYTgxY2I0YzQ3YmRhMTliNWJlNDk1NDQwNmE5ZWU0YzgyNDVhYjhmZmQ5NzZkMWUzMDYzM2IxOWEKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwYXRoIGdvbmUgZGVzcGl0ZSB0aGUgZXJyb3IgaXMgbm90IG5hbWVkCTMyZmI5OGE3OGYwMmE0OWRlODdkYzczMzcxYTVkNjQxMDgwZTExYjUwNDc0YjNhYTQ4YTBlZTc0ZWIzMWVhMmUKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwbGFjZWhvbGRlciB0aGF0IGNvdWxkIG5vdCBiZSByZW1vdmVkIGlzIG5hbWVkCTE1MDczZjQ4NmYxZDZiYzUxNjBiYmNkM2FiNTMxNTlmNDBiMTUyODAxZjA0ODQ1NzFmNGU2ZDk3NjgxNmQ1MTYKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwcm9iZSB0aGF0IGNvdWxkIG5vdCBiZSByZW1vdmVkIGlzIG5hbWVkCTg5MGUzYTU3NjU2MGI2MDIwMGYxMjM0OWY3ZDViZTRlOGM5ZDk0NDZiYjIxZWQ4NDFlZjYyMzdjMWJhMmE2YzgKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBzdGFnaW5nIGFib3J0IG5hbWVzIHRoZSB0ZW1wIGl0IGNvdWxkIG5vdCByZW1vdmUJYTdmM2M4MjFjYTI3NGU4ZWJkYTJjODIyODk0NTAwYzQ0YjY3MDEwNjY4Njk4NWIyZWYwZDQ2YzcyYTM2ZTg5MApib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhbiBhcHBsaWVkIHBsYW4gbmFtZXMgdGhlIGFzaWRlIGl0IGNvdWxkIG5vdCByZW1vdmUJOTNkOWZlOGVmMDJmMGIzZDcwYzQ5MWU4OTgxMDU4OTMyN2I5YzA2OGFjNjNjY2MwN2JlNzMwYmY0MmEwYmJmNApib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhbiBhc2lkZSB0aGUgdW5kbyBrZWVwcyBpcyBuYW1lZCBhbmQgaG9sZHMgdGhlIGZpbGUJNDE1YTcwODUyNWY2ZDg1ZjVlZmZmZGJjMWIxOWI5NTY5OGY5MDQzMDM3MGRlODhmNWY3NmM1ZTNjYzlmNGQxZApib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwl3aXRoIHJlbW92YWwgd29ya2luZyBub3RoaW5nIGlzIG5hbWVkCTg2NTY5NWM1Njc2ZjNhZDA3NTA3NmJmMzMxMzY5ZWFkMmJiZjM5Yjg0ODJjZjJkOWUzOTViYTFhMjE1YjUwNGQKYm9keQlpbnRlcm5hbC9tY3AvbGVmdDEwNV90ZXN0LmdvCVRlc3RNcndXcml0ZVNwZWxsc0xlZnRCZWhpbmRXaXRoU2xhc2hlcwk0N2RkYzhjYzdjZjQzMjA1YWQxMmNlZGQyZWUyNjFmYjQ0ZTM2MzllZWYzOGNhNjBhMGM5MmI0YTFjODM2YTlm · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: the Codex review of the record asked for the missing sites to be driven, so TestEveryFailedCleanupIsNamedInLeftBehind gained a rename destination's probe and an aside the undo keeps, and TestMrwWriteSpellsLeftBehindWithSlashes gained the terminal sentences at a 64-byte ceiling and the floor bound; every earlier assertion kept; approved
- 2026-09-30 · 1e77123* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:795
- 2026-09-30 · 1e77123* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:342
- 2026-09-30 · cf45fb0* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:0 · test-lock-sha256:e68b6458c887ea7c109a9b0d9eadc519cfc84da2dd8a1d7beadcff6fde31bbc0 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGVmdDEwNV90ZXN0LmdvCVRlc3RUaGVSZWNlaXB0c05hbWVXaGF0V2FzTGVmdEJlaGluZAk2YTJmZDUyMzBmMzc5MDQyN2ExODliYjE2ZGE5OWMxZTRjMTUzN2IwNmExYzhlMWJiNDEyNThkMTJlZjM4NjBhCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCVRlc3RFdmVyeUZhaWxlZENsZWFudXBJc05hbWVkSW5MZWZ0QmVoaW5kCTNhZTM3ZjdjMGQ4MzFlNWY4ODBiNTUwYzJlMTdkYWRlMDg3OTA0YWMwN2Q4NjY5YmUyNGMwNjhiMTYzMDk0NzgKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBkaXJlY3RvcnkgaXQgbWFkZSBpcyBuYW1lZCBvbmx5IHdoZW4gZW1wdHkJMWRjYzcyNTNjZjk4YTQxZWNlMTgxZmQ2OWJhMDdjZTliYTZiZTVhODc0N2E0MGJhOTVhNGE2ZTlhODA4YjhiNwpib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhIGZhaWxlZCBzdGFnZSBoYW5kcyBpdHMgb3duIHRlbXAgdG8gdGhlIGFib3J0CTM3ZDNlMmFkYTgxY2I0YzQ3YmRhMTliNWJlNDk1NDQwNmE5ZWU0YzgyNDVhYjhmZmQ5NzZkMWUzMDYzM2IxOWEKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwYXRoIGdvbmUgZGVzcGl0ZSB0aGUgZXJyb3IgaXMgbm90IG5hbWVkCTMyZmI5OGE3OGYwMmE0OWRlODdkYzczMzcxYTVkNjQxMDgwZTExYjUwNDc0YjNhYTQ4YTBlZTc0ZWIzMWVhMmUKYm9keQlpbnRlcm5hbC9hcHBseS9sZWZ0MTA1X3Rlc3QuZ28JYSBwYXRoIHdob3NlIGluc3BlY3Rpb24gZmFpbHMgaXMgbmFtZWQJZmNhOWNkNjg2ODE5MDA0ZmMwMmEyYTM1YmZiZjVhYzkzMDNlOWYxYzIzMDE2ODdjMjAyOGM1ODQ0NTc3ODZiYgpib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhIHBsYWNlaG9sZGVyIHRoYXQgY291bGQgbm90IGJlIHJlbW92ZWQgaXMgbmFtZWQJMTUwNzNmNDg2ZjFkNmJjNTE2MGJiY2QzYWI1MzE1OWY0MGIxNTI4MDFmMDQ4NDU3MWY0ZTZkOTc2ODE2ZDUxNgpib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhIHByb2JlIHRoYXQgY291bGQgbm90IGJlIHJlbW92ZWQgaXMgbmFtZWQJODkwZTNhNTc2NTYwYjYwMjAwZjEyMzQ5ZjdkNWJlNGU4YzlkOTQ0NmJiMjFlZDg0MWVmNjIzN2MxYmEyYTZjOApib2R5CWludGVybmFsL2FwcGx5L2xlZnQxMDVfdGVzdC5nbwlhIHN0YWdpbmcgYWJvcnQgbmFtZXMgdGhlIHRlbXAgaXQgY291bGQgbm90IHJlbW92ZQlhN2YzYzgyMWNhMjc0ZThlYmRhMmM4MjI4OTQ1MDBjNDRiNjcwMTA2Njg2OTg1YjJlZjBkNDZjNzJhMzZlODkwCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCWFuIGFwcGxpZWQgcGxhbiBuYW1lcyB0aGUgYXNpZGUgaXQgY291bGQgbm90IHJlbW92ZQk5M2Q5ZmU4ZWYwMmYwYjNkNzBjNDkxZTg5ODEwNTg5MzI3YjljMDY4YWM2M2NjYzA3YmU3MzBiZjQyYTBiYmY0CmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCWFuIGFzaWRlIHRoZSB1bmRvIGtlZXBzIGlzIG5hbWVkIGFuZCBob2xkcyB0aGUgZmlsZQk0MTVhNzA4NTI1ZjZkODVmNWVmZmZkYmMxYjE5Yjk1Njk4ZjkwNDMwMzcwZGU4OGY1Zjc2YzVlM2NjOWY0ZDFkCmJvZHkJaW50ZXJuYWwvYXBwbHkvbGVmdDEwNV90ZXN0LmdvCXdpdGggcmVtb3ZhbCB3b3JraW5nIG5vdGhpbmcgaXMgbmFtZWQJODY1Njk1YzU2NzZmM2FkMDc1MDc2YmYzMzEzNjllYWQyYmJmMzliODQ4MmNmMmQ5ZTM5NWJhMWEyMTViNTA0ZApib2R5CWludGVybmFsL21jcC9sZWZ0MTA1X3Rlc3QuZ28JVGVzdE1yd1dyaXRlU3BlbGxzTGVmdEJlaGluZFdpdGhTbGFzaGVzCWNhODgwNWZkZWI1YzU5MTc2MmM2NDM2N2VjMGYyNTBmOGRmMjBiYjMwMjU3ZDQ3NzkzNDgzMzE0OTczNGQ1Yzc · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: the Codex review of #297 found a probe could name another process's file and a failed inspection hid a leftover, so TestEveryFailedCleanupIsNamedInLeftBehind gained both cases and TestMrwWriteSpellsLeftBehindWithSlashes asserts the probe-target and aside wording of the terminal sentence; every earlier assertion kept; approved
- 2026-09-30 · cf45fb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:733
- 2026-09-30 · cf45fb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:346
- 2026-09-30 · cf45fb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:359
- 2026-09-30 · cf45fb0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3de2cbb3ed8e514e2fca348af506d69cbb8f6c8b35d73207aad98d13cf7fee27 · ms:478
