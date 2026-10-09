# Task ADR-137-T1: `--max-cols` cuts a long line to a window and a cut line licenses nothing

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `read.Options.MaxCols`, `window` in `internal/read/maxcols.go`, `mrw read --max-cols`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a line longer than the width is served as a window around the match and is not recorded as read`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.MaxCols`; the print loop cuts and leaves the line out of the recorded spans |
| `internal/read/maxcols.go` | add | `window`: the cut text for a line, its pattern and a width |
| `cmd/mrw/main.go` | edit | the flag, its domain check, the option |
| `internal/read/maxcols137_test.go`, `cmd/mrw/maxcols137_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §238 |
| `AGENTS.md`, `README.md` | edit | the flag and its licence |

## Ordered Steps

1. [S1] Write `TestAnOverlongLineIsCutToAWindowAndLicensesNothing` (a 400-character line holding a match at column 300 read with a pattern and width 40: the match is in the output, the marker names the columns, the line is not in the recorded spans, its neighbours are), `TestTheWindowCentresOnTheMatch` (centred and clamped at both ends, on runes), and in `cmd/mrw` `TestAPlanToACutLineIsRefusedAsUnread` (read with `--max-cols`, then replace the cut line: refused "has not been read"; replace a whole line: applies) and a negative width is exit 2. Confirm RED.
2. [S2] `Options.MaxCols`, `window`, the loop, the flag. Mutants: the cut line recorded as read; the window not centred on the match (taken from the start). [proof: mutation]
3. [S3] Contract §238, AGENTS.md, README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAnOverlongLineIsCutToAWindowAndLicensesNothing|TestTheWindowCentresOnTheMatch|TestAWholeFileReadThatCutALineStillLeavesItOutOfTheLedger' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnOverlongLineIsCutToAWindowAndLicensesNothing \(' "$out" \
  && grep -qE '^--- PASS: TestTheWindowCentresOnTheMatch \(' "$out" && grep -qE '^--- PASS: TestAWholeFileReadThatCutALineStillLeavesItOutOfTheLedger \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAPlanToACutLineIsRefusedAsUnread' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPlanToACutLineIsRefusedAsUnread \(' "$out" \
  && go test ./internal/read/ ./internal/mcp/ -count=1 -timeout 900s \
  && grep -q '^# 238\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnOverlongLineIsCutToAWindowAndLicensesNothing` | `internal/read/maxcols137_test.go` | the cut line shows the match and is not in the recorded spans | none | S1, S2 |
| `TestTheWindowCentresOnTheMatch` | `internal/read/maxcols137_test.go` | the window is centred, clamped, and cut on runes | none | S1, S2 |
| `TestAWholeFileReadThatCutALineStillLeavesItOutOfTheLedger` | `internal/read/maxcols137_whole_test.go` | a whole-file read that cut a line records the lines either side, not the file | none | S1, S2 |
| `TestAPlanToACutLineIsRefusedAsUnread` | `cmd/mrw/maxcols137_test.go` | a plan to the cut line is refused, one to a whole line applies, a negative width is exit 2 | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `window` and `Options.MaxCols` |
| 2 — something selects it | the flag in `readCmd`, passed to `read.Run` |
| 3 — the caller can discover it | `mrw read --help`; AGENTS.md; README |
| 4 — it is used | the 2026-10-09 survey (8 sessions); no telemetry (ADR-009) |

## Invariants

- Without the flag every line is served whole and recorded as before.
- A line within the width is served and recorded whole.
- A line served cut is never in a recorded span.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the ledger cannot represent a span with a hole (a split span) — then the cut line must make the whole range unrecorded, a weaker but safe rule the record would amend.

## Out of Scope

- `mrw_read`, JSON paths, `--limit` (deferred: docs/adr/BACKLOG.md)

## Mutation Log
- 2026-10-10 · 2cb2cc0 · mutant killed · exit 1 · `internal/read/read.go` · S2: the lines before a cut line are not recorded · acceptance-sha256:64e0738ae554ba800cd39707eb701198388a9315e81f0361148f96ce67cc4675 · covers:a line longer than the width is served as a window around the match and is not recorded as read
- 2026-10-10 · 2cb2cc0* · mutant survived · exit 0 · `internal/read/read.go` · S2: a read that cut a line is still recorded as the whole file · acceptance-sha256:64e0738ae554ba800cd39707eb701198388a9315e81f0361148f96ce67cc4675 · covers:a line longer than the width is served as a window around the match and is not recorded as read
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-10 · 2cb2cc0* · mutant killed · exit 1 · `internal/read/maxcols.go` · S2: the window is taken from the start, not centred on the match · acceptance-sha256:64e0738ae554ba800cd39707eb701198388a9315e81f0361148f96ce67cc4675 · covers:a line longer than the width is served as a window around the match and is not recorded as read
- 2026-10-10 · 0b77c71 · mutant killed · exit 1 · `internal/read/read.go` · S2: the lines before a cut line are not recorded · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · covers:a line longer than the width is served as a window around the match and is not recorded as read
- 2026-10-10 · 0b77c71* · mutant killed · exit 1 · `internal/read/read.go` · S2: a whole-file read that cut a line is still recorded as the whole file · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · covers:a line longer than the width is served as a window around the match and is not recorded as read
- 2026-10-10 · 0b77c71* · mutant killed · exit 1 · `internal/read/maxcols.go` · S2: the window is taken from the start, not centred on the match · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · covers:a line longer than the width is served as a window around the match and is not recorded as read
- 2026-10-10 · 0b77c71* · mutant inconclusive · exit 1 · `internal/read/read.go` · S2: the width is counted in bytes, not characters · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · covers:a line longer than the width is served as a window around the match and is not recorded as read
  ```
  the fence failed on a build/parse error, not an assertion
  ```

## Verification Log
- 2026-10-10 · 2cb2cc0 · exit 0 · `set -o pipefail …` · acceptance-sha256:64e0738ae554ba800cd39707eb701198388a9315e81f0361148f96ce67cc4675 · ms:39426
- 2026-10-10 · 2cb2cc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:64e0738ae554ba800cd39707eb701198388a9315e81f0361148f96ce67cc4675 · ms:34678
- 2026-10-10 · 2cb2cc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:64e0738ae554ba800cd39707eb701198388a9315e81f0361148f96ce67cc4675 · ms:36490
- 2026-10-10 · 0b77c71 · exit 0 · `set -o pipefail …` · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · ms:34886
- 2026-10-10 · 0b77c71* · exit 0 · `set -o pipefail …` · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · ms:33900
- 2026-10-10 · 0b77c71* · exit 0 · `set -o pipefail …` · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · ms:36206
- 2026-10-10 · 0b77c71* · exit 0 · `set -o pipefail …` · acceptance-sha256:c65357cb4d683f51207791ae7b7b146ddfa7d25c8287f15e05dd30be073c94ef · ms:36796
