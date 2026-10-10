# Task ADR-142-T1: `--create` drops a pipe terminator, names a BOM, and says a path exists

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `ingest.CreateContent`, the create-exists message
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `--create drops a pipe's final CRLF after LF lines, names it and a byte order mark, and says an unread existing path exists`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/create.go` | edit | `CreateContent` |
| `internal/apply/apply.go` | edit | the create-on-existing message when the path was not read |
| `cmd/mrw/main.go` | edit | `--create` calls `CreateContent` and prints each note on stderr |
| `internal/ingest/create142_test.go`, `internal/apply/create142_test.go`, `cmd/mrw/create142_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §243 |
| `AGENTS.md` | edit | the `--create` paragraph |

## Ordered Steps

1. [S1] Write `TestAPowerShellPipeTerminatorAfterLFContentIsDropped` and `TestALeadingByteOrderMarkIsKeptAndNamed` (ingest), `TestACreateOfAnUnreadExistingPathSaysItExists` (apply) and `TestWriteCreateDropsAPipeTerminatorAndSaysSo` (cmd/mrw). Confirm RED.
2. [S2] `CreateContent`, its call and the notes in `cmd/mrw`, the create-exists case in `apply`. Mutants: the terminator dropped for every content that ends in CRLF; a drop that needs no LF in the text before it; the BOM note removed; the call from `cmd/mrw` removed; the create-exists case removed. [proof: mutation]
3. [S3] Contract §243 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/ingest/ -count=1 -timeout 300s -run 'TestAPowerShellPipeTerminatorAfterLFContentIsDropped|TestALeadingByteOrderMarkIsKeptAndNamed' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPowerShellPipeTerminatorAfterLFContentIsDropped \(' "$out" \
  && grep -qE '^--- PASS: TestALeadingByteOrderMarkIsKeptAndNamed \(' "$out" \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestACreateOfAnUnreadExistingPathSaysItExists' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestACreateOfAnUnreadExistingPathSaysItExists \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestWriteCreateDropsAPipeTerminatorAndSaysSo' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestWriteCreateDropsAPipeTerminatorAndSaysSo \(' "$out" \
  && go test ./internal/ingest/ ./internal/apply/ -count=1 -timeout 900s \
  && grep -q '^# 243\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPowerShellPipeTerminatorAfterLFContentIsDropped` | `internal/ingest/create142_test.go` | a final CRLF after LF text with no other CR is dropped with a note; every other shape is returned untouched | none | S1, S2 |
| `TestALeadingByteOrderMarkIsKeptAndNamed` | `internal/ingest/create142_test.go` | a BOM is kept and named | none | S1, S2 |
| `TestACreateOfAnUnreadExistingPathSaysItExists` | `internal/apply/create142_test.go` | a lone create of an unread existing path says it exists; an edit keeps its refusal | none | S1, S2 |
| `TestWriteCreateDropsAPipeTerminatorAndSaysSo` | `cmd/mrw/create142_test.go` | the CLI creates the file and prints each note on stderr; a mixed shape is still refused | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `ingest.CreateContent` |
| 2 — something selects it | the `--create` branch of `write` in `cmd/mrw/main.go` |
| 3 — the caller can discover it | the stderr note names what happened; AGENTS.md |
| 4 — it is used | the 2026-10-10 Windows round (three of five sessions); no telemetry (ADR-009) |

## Invariants

- Every refusal of ADR-139 other than the pipe terminator stands; `CompileCreate` is unchanged.
- The receipt and the exit codes do not change; the notes go to stderr only.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the pipe terminator cannot be told from content without refusing a shape the 2026-10-10 reports show a real caller sending.

## Out of Scope

- Name refusals and NTFS stream names (deferred: docs/adr/BACKLOG.md)

## Mutation Log

## Verification Log
