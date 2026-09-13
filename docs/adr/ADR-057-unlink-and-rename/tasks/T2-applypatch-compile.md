# Task ADR-057-T2: `apply_patch` Delete File / Move to compile

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** compile of `*** Delete File:` and `*** Move to:`
**Consumes:** T1 ops
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `Delete File compiles to unlink`, `Move to compiles to rename`

## Goal

The two apply_patch headers that today compile-refuse become native hunks. No third MCP tool. Move to with extra `@@` hunks is refused this slice.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/applypatch.go` | edit | Compile instead of refuse. |
| `internal/ingest/applypatch_test.go` | edit | Invert the delete/move subtest. |
| `internal/ingest/applypatch_unlink_test.go` | create | Compile-shape tests (own file so the first-red lock does not pin the invert). |

## Ordered Steps

1. [S1] Write `TestCompileDeleteFileIsUnlink` and `TestCompileMoveToIsRename` — RED. [proof: mutation]
2. [S2] Compile. S1 GREEN. [proof: mutation]
3. [S3] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/ingest/ -count=1 -v \
  -run 'TestCompileDeleteFileIsUnlink|TestCompileMoveToIsRename' 2>&1 | tee /tmp/adr057-t2.out \
  && grep -q '^--- PASS: TestCompileDeleteFileIsUnlink' /tmp/adr057-t2.out \
  && grep -q '^--- PASS: TestCompileMoveToIsRename' /tmp/adr057-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr057-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestCompileDeleteFileIsUnlink` | `internal/ingest/applypatch_unlink_test.go` | compiles to `@@ a.go - unlink`; writes nothing | — | S1, S2 |
| `TestCompileMoveToIsRename` | `internal/ingest/applypatch_unlink_test.go` | compiles to `@@ a.go - rename` with body `b.go` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | `--format=apply_patch` / MCP `format` already call CompileApplyPatch |
| 3 — the caller can discover it | T3 |
| 4 — it is used | Codex Delete File is the motivating caller |

## Mutation Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · Delete File compiles as an unknown op instead of unlink, so TestCompileDeleteFileIsUnlink must go red · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · covers:Delete File compiles to unlink
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · Move to compiles as an unknown op instead of rename, so TestCompileMoveToIsRename must go red · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · covers:Move to compiles to rename
- 2026-09-13 · 9e48f4f* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · Move to with extra @@ hunks is compiled as an update, so TestCompileMoveToIsRename must go red · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · covers:Move to compiles to rename

## Invariants

- A git patch is still refused.
- Add File / Update File are unchanged.

## Risks

| Risk | Mitigation |
|------|------------|
| Move to with hunks silently dropped | refuse that shape; BACKLOG |

## Stop Condition

Stop if compile needs to apply hunks onto the dest in the same document.

## Out of Scope

- Contract (T3).

## Verification Log
_(tool-written)_
- 2026-09-13 · 9e48f4f* · exit 1 · `set -o pipefail …` · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · ms:5486 · test-lock-sha256:202ba6ec94dc00445e853972b645c471e8e4b37912e1ea60f8c9a1b4568c4bc3 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2luZ2VzdC9hcHBseXBhdGNoX3VubGlua190ZXN0LmdvCVRlc3RDb21waWxlRGVsZXRlRmlsZUlzVW5saW5rCTg5OTg3N2RkNDNmN2YzM2I4NzhjMjAwZWJlMzFkYmJkNTc3MjY0YThmMzMwYjUzODdjZGQzZGI5ZjA0MzMzNDkKYm9keQlpbnRlcm5hbC9pbmdlc3QvYXBwbHlwYXRjaF91bmxpbmtfdGVzdC5nbwlUZXN0Q29tcGlsZU1vdmVUb0lzUmVuYW1lCWI2MWU3YTVkZDNhMzc3NzJmNTJjMGMyYWI3NWQyMjkwYmY3MWVkY2Y4MzE0ZmI3MzI4YWI1YmU5ZDdjMmQzYWQ
  ```
  --- last 9 line(s) of stdout
  === RUN   TestCompileDeleteFileIsUnlink
      applypatch_unlink_test.go:18: Delete File compile refused: apply_patch: *** Delete File is not compiled this slice (mrw has no unlink)
  --- FAIL: TestCompileDeleteFileIsUnlink (0.00s)
  === RUN   TestCompileMoveToIsRename
      applypatch_unlink_test.go:54: Move to compile refused: apply_patch: *** Move to is not compiled this slice (mrw has no unlink)
  --- FAIL: TestCompileMoveToIsRename (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest	4.805s
  FAIL
  ```
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · ms:1335
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · ms:1393
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · ms:979
- 2026-09-13 · 9e48f4f* · exit 0 · `set -o pipefail …` · acceptance-sha256:b18122dc1b0acd052bada78774febf13edcf31010e349e83de8b20f06e1f8067 · ms:807
