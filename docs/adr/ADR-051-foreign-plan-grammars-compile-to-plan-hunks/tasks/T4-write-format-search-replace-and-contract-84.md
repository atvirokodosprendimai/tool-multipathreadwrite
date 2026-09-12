# Task ADR-051-T4: `--format=search_replace` on write and MCP; contract §84

**Depends-on:** T3
**Covers:** F-26, UC5-S1, UC5-S2, UC5-S3
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `write --format=search_replace` (T4), `mrw_write.format=search_replace` (T4), contract §84 (T4)
**Consumes:** `ingest.CompileApplyPatch` (T1), `write --format=apply_patch` (T2), `mrw_write.format` (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `SEARCH/REPLACE unread writing nothing`, `near-miss SEARCH refusing`, `§84 driving the built binary`

## Goal

Aider SEARCH/REPLACE is a second explicit `--format` / `format` value (`search_replace`). It compiles to the same `@@` plan. An unread compiled sibling writes nothing. No auto-detect. No fuzzy apply. CLI + MCP. Contract §84 drives the built binary. 4096 stays.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/searchreplace.go` | add | `CompileSearchReplace` — second compiler, not inside apply_patch. |
| `internal/ingest/searchreplace_test.go` | add | Unread two-hunk, Parse, near-miss, ambiguous, empty SEARCH. |
| `cmd/mrw/main.go` | edit | `--format=search_replace` compiles then Parses. |
| `cmd/mrw/writehelp_test.go` | edit | Help names `--format=search_replace`. |
| `cmd/mrw/writeformat_test.go` | edit | CLI unread SEARCH/REPLACE. |
| `internal/mcp/mcp.go` | edit | schema enum grows `search_replace`. |
| `internal/mcp/tools.go` | edit | `writeTool` compiles `search_replace`. |
| `internal/mcp/instructions.go` | edit | name `search_replace` without raising 4096. |
| `internal/mcp/writeformat_test.go` | edit | MCP unread SEARCH/REPLACE; no-format parse refuse. |
| `scripts/contract.sh` | edit | **§84** — pair unread / served / no-flag / near-miss / MCP unread / cargo. |

## Ordered Steps

1. [S1] Write `TestATwoHunkSearchReplaceWithOneUnreadLineWritesNothing` and confirm it is RED. [proof: mutation]
2. [S2] Write `TestWriteFormatSearchReplaceUnreadWritesNothing` (CLI + MCP) and confirm they are RED. [proof: mutation]
3. [S3] Implement `CompileSearchReplace` and wire `--format` / `format`. Exact unique SEARCH only. Confirm S1–S2 GREEN. [proof: mutation]
4. [S4] Write §84 against a binary that does not yet honour `search_replace` and confirm it is RED (unread must be exit 1 with FAIL+skip, not exit 2), then rebuild and confirm it is GREEN. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 84\. ' scripts/contract.sh \
  && go test ./internal/ingest/ ./cmd/mrw/ ./internal/mcp/ -count=1 -v \
    -run 'TestATwoHunkSearchReplaceWithOneUnreadLineWritesNothing|TestANearMissSearchIsACompileRefusal|TestWriteFormatSearchReplaceUnreadWritesNothing|TestWriteHelpNamesApplyPatchFormat' 2>&1 | tee /tmp/adr051-t4.out \
  && grep -q '^--- PASS: TestATwoHunkSearchReplaceWithOneUnreadLineWritesNothing' /tmp/adr051-t4.out \
  && grep -q '^--- PASS: TestANearMissSearchIsACompileRefusal' /tmp/adr051-t4.out \
  && grep -c '^--- PASS: TestWriteFormatSearchReplaceUnreadWritesNothing' /tmp/adr051-t4.out | grep -qx 2 \
  && grep -q '^--- PASS: TestWriteHelpNamesApplyPatchFormat' /tmp/adr051-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr051-t4.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/ingest/ ./cmd/mrw/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestATwoHunkSearchReplaceWithOneUnreadLineWritesNothing` | `internal/ingest/searchreplace_test.go` | Compile succeeds; Apply FAIL+skip; file unchanged | — | S1, S3 |
| `TestANearMissSearchIsACompileRefusal` | `internal/ingest/searchreplace_test.go` | Extra space in SEARCH refuses; no fuzzy | — | S3 |
| `TestWriteFormatSearchReplaceUnreadWritesNothing` | `cmd/mrw/writeformat_test.go` | CLI flag; exit 1; ledger | — | S2, S3 |
| `TestWriteFormatSearchReplaceUnreadWritesNothing` | `internal/mcp/writeformat_test.go` | MCP format; FAIL+skip | — | S2, S3 |
| `§84` | `scripts/contract.sh` | Built binary + server | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §84 |
| 2 — something selects it | `writeCmd` / `writeTool` `search_replace`; deleting `CompileSearchReplace` fails S2 and §84 |
| 3 — the caller can discover it | `write --help` and `mrw_write` schema |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 71fefa1* · mutant killed · exit 1 · `internal/ingest/searchreplace.go` · a near-miss SEARCH compiles and applies at line 1 instead of refusing — fuzzy unread · acceptance-sha256:5f31ead803f112cecb560300831f17c1b747b0010eb4df032466abe6a3d07ebe · covers:near-miss SEARCH refusing

## Invariants

- Default `--format` / `format` is still `plan`. Native write is unchanged.
- Compile refusal is exit 2. An unread compiled hunk is exit 1.
- Unique exact SEARCH is location, not a license. A near-miss refuses.
- `maxInstructionsChars` stays 4096. No third MCP tool.
- ADR-019 pick A stands.

## Risks

- §84's unread half passes on exit 2 when `--format` is ignored. Mitigation: assert exit 1, `FAIL`, `skip`, and `has not been read`.
- Fuzzy apply of a near-miss SEARCH. Mitigation: S3 near-miss test and §84.

## Stop Condition

Stop if the proposed fix is auto-detect, fuzzy SEARCH, raising 4096, a third MCP tool, or applying without Parse.

## Out of Scope

- `*** Delete File:` / `*** Move to:` (deferred: docs/adr/BACKLOG.md)
- ast-grep-shaped `--grep` (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-09-12 · 71fefa1* · exit 0 · `set -o pipefail …` · acceptance-sha256:5f31ead803f112cecb560300831f17c1b747b0010eb4df032466abe6a3d07ebe · ms:690
