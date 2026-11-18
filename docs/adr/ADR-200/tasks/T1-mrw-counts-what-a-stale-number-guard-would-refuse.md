# Task ADR-200-T1: mrw counts the numbered hunks a stale-number guard would refuse

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `apply.Result.StaleNumbered` and `StaleUnshown`; `apply.Options.Computed`; `authoring.Numbering`, `RecordNumbering`, `LoadNumbering`; the `stale-number pricing:` block and `numbering` in `mrw stats`; contract §343; the sentence in `mrw instructions` and `AGENTS.md`
**Consumes:** ADR-056's pricing path (`internal/writer/flow.go`, `internal/authoring`), `seen.Observation.ServedLine`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a candidate is marked and nothing is refused`, `a landed write is counted once`, `stats prints the three counts`, `the contract row drives the binary`, `the instructions say numbers move`

## Goal

ADR-200 Decisions 1 to 4. A write that sends a numbered hunk with no `anchor=` to a file mrw wrote is counted, and counted again when a line it addresses was not shown since that write. Nothing a caller sends is refused that was not refused before, and no receipt of a write changes.

**The predicate** (Decision 1), on a hunk that is `ok`, in a write with a ledger, not forced, whose plan mrw did not compute in this call:
- op `replace`, `delete`, `insert-after` or `insert-before`;
- address typed in line numbers: not a pattern, not `$` alone, not line 0;
- no `anchor=`, and not a quoted hunk (ADR-190);
- the file's observation is `Written` with no spans.
It is *unshown* when any addressed line, or for an insert the line it names, is not `ServedLine`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/stale200_test.go` | new | the red test for the marks |
| `internal/apply/apply.go` | edit | the marks, `Options.Computed`, the two result counts |
| `internal/authoring/numbering200_test.go` | new | the red test for the counters |
| `internal/authoring/authoring.go` | edit | `Numbering` and its file |
| `internal/writer/flow.go` | edit | record each landed write |
| `cmd/mrw/stale200_test.go` | new | the red test for the stats block |
| `cmd/mrw/main.go` | edit | the block, `numbering`, `Computed` for a plan maker |
| `internal/mcp/tools.go` | edit | `Computed` for a plan maker |
| `internal/guide/stale200_test.go` | new | the red test for the sentence |
| `internal/guide/guide.go` | edit | the sentence |
| `docs/receipts.txt` | edit | the four keys |
| `scripts/contract.sh` | edit | §343 |
| `AGENTS.md` | edit | the sentence, and the stats bullet |
| `docs/adr/BACKLOG.md` | edit | the bar, before the code |

## Ordered Steps

1. [S1] Write the four tests and see them RED: `TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused` (the record's reproduction lands exactly as before, with the same file bytes and hunk status, and the result counts one candidate, one unshown; then one case per clause of the predicate), `TestNumberingCountsALandedWriteOnce`, `TestStatsPrintsTheStaleNumberPricing`, `TestInstructionsSayNumbersMoveAfterAWrite`. [proof: acceptance]
2. [S2] Apply marks the hunks and sums them on the result. Mutants, each killed: a hunk with `anchor=` is a candidate; a shown line is unshown; a computed plan is counted; an insert names the line after the one it gives. [proof: mutation]
3. [S3] `authoring.Numbering` and the record call in `internal/writer/flow.go`, on every path that lands. Mutants, each killed: a write with two candidates counts two; a dry run is counted. [proof: mutation]
4. [S4] `mrw stats` prints the block and `--json` carries `numbering`; the keys are in `docs/receipts.txt`; a plan maker's write sets `Computed` on the CLI and over MCP. Mutant, killed: the block prints the candidates where it says would-refuse. [proof: mutation]
5. [S5] Contract §343 drives the built binary: the reproduction, then `mrw stats --json` shows one would-refuse write; paired with the same second hunk carrying `anchor=`, which leaves the would-refuse count where it was. [proof: acceptance]
6. [S6] `mrw instructions` and `AGENTS.md` say that numbers move after a write that adds or removes lines. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 ./internal/apply ./internal/authoring ./internal/guide ./cmd/mrw -run 'TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused|TestNumberingCountsALandedWriteOnce|TestStatsPrintsTheStaleNumberPricing|TestInstructionsSayNumbersMoveAfterAWrite' >/dev/null \
  && grep -q 'func TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused' internal/apply/stale200_test.go \
  && grep -q 'func TestNumberingCountsALandedWriteOnce' internal/authoring/numbering200_test.go \
  && grep -q 'func TestStatsPrintsTheStaleNumberPricing' cmd/mrw/stale200_test.go \
  && grep -q 'func TestInstructionsSayNumbersMoveAfterAWrite' internal/guide/stale200_test.go \
  && grep -q '^# 343\. ' scripts/contract.sh \
  && grep -q 'numbered_would_refuse' docs/receipts.txt \
  && grep -q 'StaleUnshown' internal/apply/apply.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused` | `internal/apply/stale200_test.go` | the reproduction lands unchanged and is counted; each clause of the predicate | none | S1, S2 |
| `TestNumberingCountsALandedWriteOnce` | `internal/authoring/numbering200_test.go` | three counters, one landed write counted once whatever its hunks | none | S1, S3 |
| `TestStatsPrintsTheStaleNumberPricing` | `cmd/mrw/stale200_test.go` | the block and the JSON keys after a real write through the CLI; a `--sub` write is not a candidate; a dry run counts nothing | none | S1, S3, S4 |
| `TestInstructionsSayNumbersMoveAfterAWrite` | `internal/guide/stale200_test.go` | the sentence is in the full instructions | none | S1, S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §343 |
| 2 — something selects it | every landed write goes through `internal/writer/flow.go`, which records it |
| 3 — the caller can discover it | `mrw stats`, `docs/receipts.txt`, `AGENTS.md` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is the reproduced failure and E5's reading, whose uncertain share only the ledger can settle |

## Mutation Log

## Verification Log

## Invariants

- No write is refused, and no write receipt changes, because of this task.
- The counters hold counts only: no path, no content.
- A counter that cannot be read or written never fails a write.

## Risks

- The marks are computed where apply already holds the observation; a hunk judged without one (no ledger, `--force`) is not a candidate.
- The landed-writes counter starts at this release, so it is smaller than the tally's.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- The refusal (deferred: docs/adr/BACKLOG.md)
- A receipt row naming where the numbers moved (deferred: docs/adr/BACKLOG.md)
