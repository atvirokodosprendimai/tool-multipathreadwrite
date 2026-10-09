# Task ADR-138-T1: the walk counts a link to a directory and says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `WalkSkipped.LinkedDirs`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a link to a directory met by a walk inside the root is counted on the skipped line and not followed`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/walk.go` | edit | the set, the count, the `SkipNote` clause and tail |
| `internal/read/linkeddirs138_test.go` | add | the tests |
| `internal/mcp/schema_test.go`, `docs/receipts.txt` | edit | `skipped.linked_dirs` in the read schema the receipt test holds, and the registry |
| `scripts/contract.sh` | edit | §239; the §216 row that holds `skipped` to an exact object lists the key |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | the `-- skipped:` paragraph; the taken bullet |

## Ordered Steps

1. [S1] Write `TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow` (a tree with a directory holding a matching file and a link to it: the walk serves the real path only, `LinkedDirs` is 1, the note says `1 link(s) to a directory, not followed` and `name one to be told why`), `TestAnEscapingLinkToADirectoryIsNotCounted` and `TestALinkToAFileIsServedNotCounted`. Confirm RED.
2. [S2] The set, the count, the clause and tail. Mutants: the link is dropped uncounted again; a file link is counted. [proof: mutation]
3. [S3] `skipped.linked_dirs` in the schema and registry, contract §239, AGENTS.md, README, the BACKLOG bullet removed. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow|TestAnEscapingLinkToADirectoryIsNotCounted|TestALinkToAFileIsServedNotCounted' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow \(' "$out" \
  && grep -qE '^--- PASS: TestAnEscapingLinkToADirectoryIsNotCounted \(' "$out" \
  && grep -qE '^--- PASS: TestALinkToAFileIsServedNotCounted \(' "$out" \
  && go test ./internal/read/ ./internal/mcp/ -count=1 -timeout 900s \
  && grep -q '^# 239\. ' scripts/contract.sh \
  && grep -q '^mcp_read skipped.linked_dirs$' docs/receipts.txt \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow` | `internal/read/linkeddirs138_test.go` | the link is counted, the real path served once, the note says so | none | S1, S2 |
| `TestAnEscapingLinkToADirectoryIsNotCounted` | `internal/read/linkeddirs138_test.go` | a link out of the root stays silent | none | S1, S2 |
| `TestALinkToAFileIsServedNotCounted` | `internal/read/linkeddirs138_test.go` | a link to a file is a candidate, not a directory link | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `LinkedDirs` and its set |
| 2 — something selects it | the walker's non-file skip, on every discovered path |
| 3 — the caller can discover it | the `-- skipped:` line and `skipped.linked_dirs`; AGENTS.md; README |
| 4 — it is used | the 2026-10-09 Windows chaos round; no telemetry (ADR-009) |

## Invariants

- What a walk serves and every exit code are unchanged.
- A path the caller names is refused by name as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if counting a link to a directory needs a read of the directory's contents: that is the oracle ADR-007 rule 2 shuts.

## Out of Scope

- A FIFO, socket or device (deferred: docs/adr/BACKLOG.md)

## Mutation Log

## Verification Log
