# Task ADR-051-T3: `mrw_write` grows `format`; contract §83

**Depends-on:** T2
**Covers:** F-24, F-25, F-27, UC4-S1, UC4-S2, UC4-S3
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw_write.format` (T3), contract §83 (T3)
**Consumes:** `ingest.CompileApplyPatch` (T1), `write --format=apply_patch` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `MCP format apply_patch unread writing nothing`, `format declared on existing write`, `git refuse on MCP`

## Goal

`mrw_write` optional `format` (`plan` default, `apply_patch` same compile path as CLI `--format=apply_patch`). `git` is refuse. No third tool. 4096 stays. Contract §83 drives the built server.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | `format` on `mrw_write` input schema. |
| `internal/mcp/tools.go` | edit | compile → Parse → Apply; `git` / unknown are usage. |
| `internal/mcp/instructions.go` | edit | name `format` without raising 4096. |
| `internal/mcp/writeformat_test.go` | add | unread two-hunk MCP write; no-format parse refuse; git refuse; schema + cargo. |
| `scripts/contract.sh` | edit | **§83** — pair unread / served / no-format / git / two-tool cargo. |

## Ordered Steps

1. [S1] Write `TestWriteFormatApplyPatchUnreadWritesNothing` and confirm it is RED: format is ignored and the blob is a native parse error. [proof: mutation]
2. [S2] Write `TestWriteDeclaresFormatOnTheExistingTool` and confirm it is RED. [proof: mutation]
3. [S3] Wire `format` on `writeTool` and the schema. `apply_patch` compiles then Parses. `git` is usage. Shorten handshake so 4096 holds. Confirm S1–S2 GREEN. [proof: mutation]
4. [S4] Write §83 against a binary that does not yet honour MCP `format` and confirm it is RED (unread must be failed+skipped, not a parse error), then rebuild and confirm it is GREEN. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 83\. ' scripts/contract.sh \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestWriteFormatApplyPatchUnreadWritesNothing|TestAnApplyPatchBlobOnWritePlanIsABadNativePlan|TestWriteFormatGitIsRefused|TestWriteDeclaresFormatOnTheExistingTool' 2>&1 | tee /tmp/adr051-t3.out \
  && grep -q '^--- PASS: TestWriteFormatApplyPatchUnreadWritesNothing' /tmp/adr051-t3.out \
  && grep -q '^--- PASS: TestAnApplyPatchBlobOnWritePlanIsABadNativePlan' /tmp/adr051-t3.out \
  && grep -q '^--- PASS: TestWriteFormatGitIsRefused' /tmp/adr051-t3.out \
  && grep -q '^--- PASS: TestWriteDeclaresFormatOnTheExistingTool' /tmp/adr051-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr051-t3.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWriteFormatApplyPatchUnreadWritesNothing` | `internal/mcp/writeformat_test.go` | MCP format apply_patch, one unread sibling: FAIL+skip, file unchanged | — | S1, S3 |
| `TestAnApplyPatchBlobOnWritePlanIsABadNativePlan` | `internal/mcp/writeformat_test.go` | No format: parse refuse, no third tool | — | S3 |
| `TestWriteFormatGitIsRefused` | `internal/mcp/writeformat_test.go` | format=git is usage | — | S3 |
| `TestWriteDeclaresFormatOnTheExistingTool` | `internal/mcp/writeformat_test.go` | schema + handshake name format; 4096 stays; two tools | — | S2, S3 |
| `§83` | `scripts/contract.sh` | Built server: unread / served / no-format / git / cargo | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §83 |
| 2 — something selects it | `writeTool` `format`; deleting the `CompileApplyPatch` call fails S1 and §83 |
| 3 — the caller can discover it | `mrw_write` schema `format` and initialize instructions |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 71fefa1* · mutant killed · exit 1 · `internal/mcp/tools.go` · format apply_patch is a pass-through: the blob is parsed as a native plan so unread is a parse error, not FAIL+skip · acceptance-sha256:05a95844a0d1681f220d9242833c9eceb13948e2ed98e030613ae8e0ac0bd225 · covers:MCP format apply_patch unread writing nothing

## Invariants

- Default `format` is `plan`. Native `mrw_write` is unchanged.
- Compile refusal is a tool error (CLI exit 2). An unread compiled hunk is FAIL+skip (CLI exit 1).
- `maxInstructionsChars` stays 4096. Do not put `format` in `guide.Shared()`.
- ADR-044 cargo stays two tools. ADR-019 pick A stands.

## Risks

- §83's unread half passes on a parse error when `format` is ignored. Mitigation: assert failed+skipped and `has not been read`.
- Teaching on Shared() overflows 4096. Mitigation: shorten MCP-only prose, not Shared().

## Stop Condition

Stop if the proposed fix is a third MCP tool, auto-detect, raising 4096, or applying without Parse.

## Out of Scope

- Aider SEARCH/REPLACE as `--format` (deferred: docs/adr/BACKLOG.md)
- `*** Delete File:` / `*** Move to:` (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-09-12 · 71fefa1* · exit 0 · `set -o pipefail …` · acceptance-sha256:05a95844a0d1681f220d9242833c9eceb13948e2ed98e030613ae8e0ac0bd225 · ms:804
