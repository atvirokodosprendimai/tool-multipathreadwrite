# Task ADR-079-T2: a dry run is not a landing

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the dry-run case in both tally switches
**Consumes:** `authoring.Record`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a clean dry run records nothing`, `a refused dry run is one refusal`, `both surfaces agree`, `a contract row drives the binary`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A clean `--dry-run` was tallied as `applied` on the CLI and as `refused_apply` over MCP.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the CLI tally skips a clean dry run |
| `internal/mcp/tools.go` | edit | the MCP tally does too |
| `cmd/mrw/bookkeeping079_test.go`, `internal/mcp/dryrun079_test.go` | new | the tests below |
| `scripts/contract.sh`, `docs/adr/BACKLOG.md` | edit | §161; the round's item closed |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: the CLI dry-run case removed; the MCP dry-run case removed.
3. [S3] Contract §161. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 240s -run 'TestACleanDryRunRecordsNothing|TestACleanMCPDryRunRecordsNothing' -v 2>&1 | tee /tmp/adr079-T2.out \
  && missing=$(for t in TestACleanDryRunRecordsNothing TestACleanMCPDryRunRecordsNothing; do grep -qE "^--- PASS: $t \(" /tmp/adr079-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 161\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/check internal/lines internal/subproc internal/rooted internal/read \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/check internal/lines internal/subproc internal/rooted internal/read)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACleanDryRunRecordsNothing` | `cmd/mrw/bookkeeping079_test.go` | a clean dry run records nothing; a refused one is one refusal | — | S1, S2 |
| `TestACleanMCPDryRunRecordsNothing` | `internal/mcp/dryrun079_test.go` | the same over MCP | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the functions under Produces |
| 2 — something selects it | every `--dry-run` on `mrw write` and every `dry_run` on `mrw_write` |
| 3 — the caller can discover it | `mrw stats` no longer counts a clean dry run, and counts a refused one once |
| 4 — it is used | the review of #229 met it on both surfaces |

## Verification Log
(empty until execute)
- 2026-09-26 · 686cb60* · exit 1 · `set -o pipefail …` · acceptance-sha256:f90c428f23c944eda93d428a24a53dd9a5321a44e169c768f973f143bb76e07d · ms:1269 · test-lock-sha256:bb5c956030335ade5fb46984b035b1c8b5ca996cda3947c24374b45138396766 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYm9va2tlZXBpbmcwNzlfdGVzdC5nbwlUZXN0QUNsZWFuRHJ5UnVuUmVjb3Jkc05vdGhpbmcJNzNlMTdhMDg2MDIwOTQ3YWQ0YTk2YmY1YmE2ZjE4MjJhM2NkYTBmODY1YTg0MTA5ZjkxMDEwZTQyNjI0NzEyMwpib2R5CWNtZC9tcncvYm9va2tlZXBpbmcwNzlfdGVzdC5nbwlUZXN0U2Vlbk5ldmVyUHJpbnRzQUhhbGZTYXZlZExlZGdlcglkNGFlNDI3ZmZjMjliZTg2ODFlYjQ3MDljOTQxM2VjM2ZkM2RmN2RjNTNjOTllM2I3MTJiODU3NzdkMWZjMzRiCmJvZHkJaW50ZXJuYWwvbWNwL2RyeXJ1bjA3OV90ZXN0LmdvCVRlc3RBQ2xlYW5NQ1BEcnlSdW5SZWNvcmRzTm90aGluZwk1ZGFjMTQ1NGZjZDI5YjUzZjVhNzYzZDkzZDIwMjRjYjQ5MGQ3ZmU5ODZiZTYwYzJhOGY1MTk0ZTdjN2YwOTVk
  ```
  --- last 10 line(s) of stdout (of 13 after folding 13 raw)
  --- FAIL: TestACleanDryRunRecordsNothing (0.04s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.179s
  === RUN   TestACleanMCPDryRunRecordsNothing
      dryrun079_test.go:18: a clean MCP dry run was tallied: map[refused_apply:1]
      dryrun079_test.go:22: a refused MCP dry run is not one refusal: map[refused_apply:2]
  --- FAIL: TestACleanMCPDryRunRecordsNothing (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.198s
  FAIL
  ```
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:f90c428f23c944eda93d428a24a53dd9a5321a44e169c768f973f143bb76e07d · ms:632
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:f90c428f23c944eda93d428a24a53dd9a5321a44e169c768f973f143bb76e07d · ms:686
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:f90c428f23c944eda93d428a24a53dd9a5321a44e169c768f973f143bb76e07d · ms:323
- 2026-09-26 · 686cb60* · exit 0 · `set -o pipefail …` · acceptance-sha256:f90c428f23c944eda93d428a24a53dd9a5321a44e169c768f973f143bb76e07d · ms:338

## Mutation Log
(empty until execute)
- 2026-09-26 · 686cb60* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI tallies a clean dry run as applied · acceptance-sha256:f90c428f23c944eda93d428a24a53dd9a5321a44e169c768f973f143bb76e07d · covers:a clean dry run records nothing
- 2026-09-26 · 686cb60* · mutant killed · exit 1 · `internal/mcp/tools.go` · MCP tallies a clean dry run as a refusal · acceptance-sha256:f90c428f23c944eda93d428a24a53dd9a5321a44e169c768f973f143bb76e07d · covers:both surfaces agree

## Invariants

- `mrw stats` counts a plan only for what happened to the tree or to the plan.

## Risks

- None beyond the counts' meaning, which the record states.

## Out of Scope

- Counting dry runs in a bucket of their own (permanent: boundary: the tally measures what became of plans given to write, ADR-009; a dry run changes nothing)

## Stop Condition

The fence exits 0 and every Mutation Log row reads `killed`.
