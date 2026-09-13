# Task ADR-057-T1: parse and apply `unlink` / `rename`

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `OpUnlink`, `OpRename`; apply removes/moves; `seen.Drop`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `unlink removes the path`, `partial read does not license unlink`, `sibling fail restores`, `rename moves`, `dest exists refuses`

## Goal

A plan can remove a path and can move a path. Line-range `delete` is unchanged. A sibling failure writes nothing. A one-line read does not license removing the file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | Two ops; validate `-` address; rename body is one dest. |
| `internal/apply/apply.go` | edit | Path-level commit; mix refuse; dest-exists refuse. |
| `internal/seen/seen.go` | edit | `Drop`. |
| `internal/plan/unlink_test.go` | create | Red: parse unlink and rename. |
| `internal/apply/unlink_test.go` | create | Red: remove, licence, sibling, rename, dest exists. |
| `internal/seen/drop_test.go` | create | Red: Drop removes the path from the ledger. |

## Ordered Steps

1. [S1] Write the tests — RED. [proof: mutation]
2. [S2] Parser + validate. S1 parse tests GREEN. [proof: mutation]
3. [S3] Apply + Drop. Remaining GREEN. [proof: mutation]
4. [S4] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ ./internal/apply/ ./internal/seen/ -count=1 -v \
  -run 'TestUnlinkParses|TestRenameParsesWithDestBody|TestUnlinkRemovesThePath|TestUnlinkWithoutWholeFileReadIsRefused|TestUnlinkRestoresWhenASiblingFails|TestRenameMovesThePath|TestRenameOntoExistingDestIsRefused|TestDropRemovesThePath' 2>&1 | tee /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestUnlinkParses' /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestRenameParsesWithDestBody' /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestUnlinkRemovesThePath' /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestUnlinkWithoutWholeFileReadIsRefused' /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestUnlinkRestoresWhenASiblingFails' /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestRenameMovesThePath' /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestRenameOntoExistingDestIsRefused' /tmp/adr057-t1.out \
  && grep -q '^--- PASS: TestDropRemovesThePath' /tmp/adr057-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr057-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/plan/ ./internal/apply/ ./internal/seen/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestUnlinkParses` | `internal/plan/unlink_test.go` | `@@ gone.txt - unlink` parses; empty body | — | S1, S2 |
| `TestRenameParsesWithDestBody` | `internal/plan/unlink_test.go` | `@@ old.txt - rename` body dest | — | S1, S2 |
| `TestUnlinkRemovesThePath` | `internal/apply/unlink_test.go` | path gone; hunk ok; FileResult.Removed | — | S1, S3 |
| `TestUnlinkWithoutWholeFileReadIsRefused` | `internal/apply/unlink_test.go` | spans 1-1 on a 5-line file fails; file stays | — | S1, S3 |
| `TestUnlinkRestoresWhenASiblingFails` | `internal/apply/unlink_test.go` | failed sibling → file still there | — | S1, S3 |
| `TestRenameMovesThePath` | `internal/apply/unlink_test.go` | source gone, dest has the bytes | — | S1, S3 |
| `TestRenameOntoExistingDestIsRefused` | `internal/apply/unlink_test.go` | dest exists, not unlinked → fail | — | S1, S3 |
| `TestDropRemovesThePath` | `internal/seen/drop_test.go` | Drop then Load has no key | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | CLI/MCP T3; deleting apply unlink leaves TestUnlinkRemovesThePath red |
| 3 — the caller can discover it | T3 teach |
| 4 — it is used | Zeus field report; no telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/plan/plan.go` · unlink and rename parse as unknown ops again, so TestUnlinkParses and TestRenameParsesWithDestBody must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f
- 2026-09-13 · 9e48f4f* · mutant inconclusive · exit 1 · `internal/apply/pathop.go` · rename onto an existing dest is accepted, so TestRenameOntoExistingDestIsRefused must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · covers:dest exists refuses
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-13 · 9e48f4f* · mutant inconclusive · exit 1 · `internal/apply/pathop.go` · a one-line read licenses unlink of a five-line file, so TestUnlinkWithoutWholeFileReadIsRefused must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · covers:partial read does not license unlink
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/seen/seen.go` · Drop is a no-op, so TestDropRemovesThePath must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/apply/pathop.go` · unlink never moves the path aside, so TestUnlinkRemovesThePath must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · covers:unlink removes the path
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/apply/pathop.go` · rename onto an existing dest is accepted, so TestRenameOntoExistingDestIsRefused must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · covers:dest exists refuses
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/apply/pathop.go` · a one-line read licenses unlink of a five-line file, so TestUnlinkWithoutWholeFileReadIsRefused must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · covers:partial read does not license unlink
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/apply/pathop.go` · rename never moves the path, so TestRenameMovesThePath must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · covers:rename moves
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/apply/apply.go` · a failed sibling leaves the unlink hunk ok instead of skip, so TestUnlinkRestoresWhenASiblingFails must go red · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · covers:sibling fail restores

## Invariants

- `delete` is still a line-range.
- A nil ledger still disables the guard (engine tests).
- ADR-021: one path, one spelling.

## Risks

| Risk | Mitigation |
|------|------------|
| Unlink implemented as splice-to-empty | TestUnlinkRemovesThePath asserts `os.IsNotExist` |

## Stop Condition

Stop if unlink needs a target-syntax parser or a third MCP tool.

## Out of Scope

- apply_patch compile (T2).
- Contract rows (T3).

## Verification Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · exit 1 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:985 · test-lock-sha256:c1364f2a0021069e0803a184c5ba5e6222d85c883e848cab083041722ce3c42b · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L3VubGlua190ZXN0LmdvCVRlc3RSZW5hbWVNb3Zlc1RoZVBhdGgJZGE0YWJmZmJjODkzMzIxNzAzMzljMGIyODE1NjAzOTM4N2FmNjA5NWUxNzM0YTIzMmVlYzExMGJhODRiODcyNwpib2R5CWludGVybmFsL2FwcGx5L3VubGlua190ZXN0LmdvCVRlc3RSZW5hbWVPbnRvRXhpc3RpbmdEZXN0SXNSZWZ1c2VkCTJjOThjODQwOGY4NGJhODI1NWY5NzliNGFjNTIyZTdiNmIwNDUyNmU4N2FkMDU1OTU4YmU5OTczNjA4ZDRmM2UKYm9keQlpbnRlcm5hbC9hcHBseS91bmxpbmtfdGVzdC5nbwlUZXN0VW5saW5rUmVtb3Zlc1RoZVBhdGgJYmRhMjA2OGFlMDlmZGEwNGVmOTM0N2Q2NDBjM2ExMDU1ZWI3YjI0OTgyZWRmYWUzMWU1NzJhNDhhMmUyNDhkNQpib2R5CWludGVybmFsL2FwcGx5L3VubGlua190ZXN0LmdvCVRlc3RVbmxpbmtSZXN0b3Jlc1doZW5BU2libGluZ0ZhaWxzCTFkNWQ4YzMxMjE2NDFjNTMyMjliODgyNDJjNzc5ZDgzYzA0NDk2NzM4MzRjMjczYWVhMTE0ZTY0YjBiODg2ZTgKYm9keQlpbnRlcm5hbC9hcHBseS91bmxpbmtfdGVzdC5nbwlUZXN0VW5saW5rV2l0aG91dFdob2xlRmlsZVJlYWRJc1JlZnVzZWQJODYzNGE3ODQ4ZGUwYmQ4MDA2NGQ3ZDlmMmQ5N2JkMDY4MDUzZGU4OWIxYjc3NDU1N2Y3MjhjNWE3YjRiMWI0Ygpib2R5CWludGVybmFsL3BsYW4vdW5saW5rX3Rlc3QuZ28JVGVzdFJlbmFtZVBhcnNlc1dpdGhEZXN0Qm9keQk1NTI1YzE1MTg4YTVhMzEwYjViMDMyZTI1MzRlYzMyMGQzMDUyZjYwZTY0YWY5YWZmY2M3YjcwMGQxYTNjNzFiCmJvZHkJaW50ZXJuYWwvcGxhbi91bmxpbmtfdGVzdC5nbwlUZXN0VW5saW5rUGFyc2VzCTM2N2NmZjU5YTJmN2JjODA0NzNjOTBlZGY2NDRiODIyZjhmMGQ3YzM4MjQ5NTBmMTE0NzFmNjI5NGNiOGEwODIKYm9keQlpbnRlcm5hbC9zZWVuL2Ryb3BfdGVzdC5nbwlUZXN0RHJvcFJlbW92ZXNUaGVQYXRoCWYzODE4ZmQ3Zjc3NzFiODBiZjUwZjRmNTg3YWQ1MzE4ODcwNmM3ZDliNjQxYzY2NzY3ZTczNzU1NTUwYTY0ODQ
  ```
  --- last 10 line(s) of stdout (of 34 after folding 34 raw)
      unlink_test.go:139: reason = "unsupported op \"rename\"", want already exists
  --- FAIL: TestRenameOntoExistingDestIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.279s
  === RUN   TestDropRemovesThePath
      drop_test.go:21: Drop left gone.go in the ledger: map[gone.go:{aaa []} keep.go:{bbb []}]
  --- FAIL: TestDropRemovesThePath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	0.412s
  FAIL
  ```
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:959
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:726
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:742
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:798
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:1408
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:1140
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:670
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:1140
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9705512829b8ff5f3a149b26fc6b26885e2bcf03d4b292ed79fab0df1155d82f · ms:1617
