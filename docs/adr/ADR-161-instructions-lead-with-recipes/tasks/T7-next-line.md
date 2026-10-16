# Task ADR-161-T7: a ranged read serves the line after each range

**Depends-on:** T6
**Covers:** none — no spec
**Estimated scope:** M (read output pinned by many tests and rows)
**Owner:** Zy
**Produces:** the line after each range served and licensed; ADR-136's note retired
**Consumes:** `Options.Quiet` (T6), which licenses that line without printing it
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a range is served with the line after it`, `a replace of the lines read is not refused`

## Goal

A multi-line replace of exactly the lines an agent read is never refused for the line below them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | each span ending before the last line served one line further; ADR-136's note removed |
| `internal/read/next161_test.go` | add | `TestARangedReadServesTheLineAfterIt` |
| `internal/read/read_test.go`, `quiet136_test.go`, `maxcols137_test.go`, `cmd/mrw` tests pinning read output | edit | the extra line; the note's tests retired with ADR-136's note |
| `scripts/contract.sh` | edit | §267, and rows pinning the note or a ranged read's exact output |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] A span `A-B` with `B` before the last line is served `A-(B+1)`, the extra line numbered like the rest and licensed; `$`, a whole-file read and a span ending on the last line unchanged; `--max-lines` counts it; `--quiet` licenses it without printing; a `--grep` match gets its next line; the `@@ A-B` range line names what was served. [proof: mutation]
3. [S3] ADR-136's note retired; every test and contract row pinning it or a ranged read's exact output updated in the same change. [proof: acceptance]
4. [S4] Contract §267: `$MRW read f:3-5`, then `@@ f 3-5 replace anchor=… body=3` applies, exit 0; the pair, a plan of `3-6` after the same read, is refused for line 7. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./cmd/mrw/ ./internal/guide/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestARangedReadServesTheLineAfterIt ' <<<"$out" \
  && test -z "$(gofmt -l .)" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a replace of exactly the lines read is not refused for the line below' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the test the fence names joins this table in the change that writes it, since `TestEveryTestNamedByARecordExists` holds every named test to exist | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the span widening in `read.Run` |
| 2 — something selects it | every ranged `mrw read` |
| 3 — the caller can discover it | `mrw instructions` (T3) |
| 4 — it is used | every plan of the lines just read |

## Mutation Log
(empty until execute)

## Invariants

- ADR-052's rule is unchanged: a multi-line replace still needs the line after its end served.
- A whole-file read is unchanged.

## Risks

- A caller that parses a ranged read's output by line count sees one more line; the `@@ A-B` line says so.

## Stop Condition

The extra line breaks an existing contract row or test whose promise is not about the extra line itself; then stop and bring it to Zy.

## Out of Scope

- The MCP surface's served pages (permanent: boundary: T9 serves the extra line there, inside its checkpoints)

## Verification Log
(empty until execute)
