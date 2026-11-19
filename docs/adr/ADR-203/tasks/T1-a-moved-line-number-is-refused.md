# Task ADR-203-T1: a number alone is refused on an unshown line of a written file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the refusal and `refusal.StaleNumber`; the `mrw stats` sentence; contract §349 and the changed §343; the instructions, `AGENTS.md`, `README.md`, `BESTPRACTICES.md`
**Consumes:** `staleMarks` and `unshown` (`internal/apply/apply.go`, ADR-200), `refusal.Kind`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a number alone on an unshown line of a written file is refused`, `every way to say which line still lands`, `the refusal names the first unshown line and the ways out`, `the stats block says what its counters are now`, `the other engine packages are untouched`

## Goal

ADR-203 Decisions 1 to 5. `f.txt` holds `L1` to `L10`, is read whole, and is then written by mrw with `@@ f.txt 3 insert-after` one line `NEW`. Then, with no read between unless the row says so:

| Second plan | Outcome |
|-------------|---------|
| `@@ f.txt 8 replace` | refused: exit 1, the file unchanged, the reason naming line 8 and the three ways out, kind `stale-number` |
| `@@ f.txt 2 replace` | refused: above the insert too, since the rule asks only whether the line was shown |
| `@@ f.txt 4-6 delete` | refused, naming line 4 |
| `@@ f.txt 8 replace anchor="L7"` | lands, on the line that holds L7 |
| `@@ f.txt /^L7$/ replace` | lands |
| `mrw read f.txt:8`, then `@@ f.txt 8 replace` | lands |
| `mrw read f.txt:5-6`, then `@@ f.txt 4-6 delete` | refused, naming line 4; after `mrw read f.txt:4-6` it lands |
| `mrw read --quiet f.txt:8`, then `@@ f.txt 8 replace` | refused: a quiet read shows nothing |
| `@@ f.txt 8 replace` under `--force` | lands |
| `mrw write --grep '^L7$' --sub X` | lands: computed in the call |
| `@@ f.txt $ replace`, `@@ f.txt 0 insert-after`, `@@ f.txt 1 insert-before`, `@@ f.txt 12 insert-before` | land: positions |
| `@@ f.txt 5 insert-after` | refused: it names line 5 |
| a second hunk `@@ g.txt 1 replace anchor=…` on a read file beside the refused one | `skip`, nothing written to either |

And: after `mrw write --create c.txt` of five lines, `@@ c.txt 4 replace` is refused and `@@ c.txt 4 replace anchor="…"` lands. A file read whole and never written lands by number as before. Over MCP, `mrw_write` with the first row's plan answers `isError` with the reason in the hunk.

`mrw stats` after a refused write: `landed writes` unchanged, and the block's last line no longer names 2026-12-18.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the refusal at `staleMarks`' call sites |
| `internal/refusal/refusal.go` | edit | `StaleNumber` |
| `internal/apply/stale203_test.go` | new | the Goal's table against the engine |
| `cmd/mrw/stale203_test.go` | new | the first, fourth, sixth, eighth, ninth and create rows through the built binary, and the stats sentence |
| `internal/mcp/stale203_test.go` | new | the MCP row |
| `internal/guide/stale203_test.go` | new | the instructions |
| `cmd/mrw/main.go` | edit | the `mrw stats` sentence |
| `cmd/mrw/stale200_test.go` | edit | its chain lands by `anchor=`; its stats expectation follows the sentence (ADR-200 T1's lock is relocked after review) |
| `internal/guide/guide.go`, `internal/guide/renumber200_test.go`, `.claude/skills/mrw/SKILL.md` | edit | rule 2, recipes 7 and 21 |
| `scripts/contract.sh` | edit | §343's seed, §349 |
| `AGENTS.md`, `README.md`, `BESTPRACTICES.md` | edit | the same sentences |
| `docs/adr/BACKLOG.md` | edit | the reading set aside, the friction reading, the deferrals |
| other tests and contract rows that chain numbered writes | edit | each by an anchor, a pattern or a read, listed in the pull request |

## Ordered Steps

1. [S1] The four tests, RED. [proof: acceptance]
2. [S2] The refusal and its kind. Mutants, each killed: the refusal is not made; it is made on a shown line; it is made with `anchor=`; it is made under `--force`; it is made on a computed plan; it is made on a file read whole and never written; a range is judged on its first line only; the reason names the hunk's first line when that line is shown and a later one is not. [proof: mutation]
3. [S3] The `mrw stats` sentence. Mutant, killed: the sentence still names 2026-12-18. [proof: mutation]
4. [S4] Contract §349 drives the built binary: the first row refused with the file unchanged, paired with the anchored form and the re-read form landing, and a create followed by a numbered edit refused then landing with `anchor=`. §343 keeps its counters with a seed that lands. The instructions, `AGENTS.md`, `README.md`, `BESTPRACTICES.md`, BACKLOG. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/apply ./cmd/mrw ./internal/mcp ./internal/guide -run '^(TestANumberAloneIsRefusedOnALineNotShownSinceTheWrite|TestAStaleNumberIsRefusedByTheBinary|TestMrwWriteRefusesAStaleNumber|TestTheInstructionsSayASecondEditNamesItsLine|TestStatsPrintsTheStaleNumberPricing)$' > "${TMPDIR:-/tmp}/adr203-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr203-fence.out")" -eq 5 ] \
  && grep -q '^# 349\. ' scripts/contract.sh \
  && grep -q 'StaleNumber' internal/refusal/refusal.go \
  && grep -q 'has not been shown since mrw wrote the file' AGENTS.md \
  && ! grep -q '2026-12-18' cmd/mrw/main.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state internal/writer \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state internal/writer)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANumberAloneIsRefusedOnALineNotShownSinceTheWrite` | `internal/apply/stale203_test.go` | every row of the Goal's table, each refusal by its reason, its kind and the file left unchanged | none | S1, S2 |
| `TestAStaleNumberIsRefusedByTheBinary` | `cmd/mrw/stale203_test.go` | the named rows through `mrw write` with exit codes; `mrw stats` afterwards | none | S1, S2, S3 |
| `TestMrwWriteRefusesAStaleNumber` | `internal/mcp/stale203_test.go` | `mrw_write`: `isError`, the reason in the hunk, nothing written | none | S1, S2 |
| `TestTheInstructionsSayASecondEditNamesItsLine` | `internal/guide/stale203_test.go` | the core and the full instructions say a second edit to a file mrw wrote names its line by pattern, `anchor=` or after a read that shows it; the core stays under its limit | none | S4 |
| `TestStatsPrintsTheStaleNumberPricing` | `cmd/mrw/stale200_test.go` | exists today (ADR-200 T1); its chain is changed to land by `anchor=` and its last-line expectation to the new sentence, which is what goes red | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests and §349 |
| 2 — something selects it | every `mrw write` and `mrw_write` goes through apply's validation |
| 3 — the caller can discover it | the refusal's own sentence; `mrw instructions --core` rule 2; `AGENTS.md` |
| 4 — it is used | telemetry is refused by ADR-009; the reading of BACKLOG.md "Pre-registered for ADR-203" |

## Mutation Log

## Verification Log

## Invariants

- A plan that addresses by pattern, carries `anchor=`, is quoted, forced, or computed by mrw lands as before.
- A file read and never written by mrw is judged as before.
- The ledger's format and every receipt key are unchanged.

## Risks

- Every existing test, contract row and document example that edits by number alone a file mrw wrote, with no read between, goes red: each is changed to say which line it means, and the list is in the pull request.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- A selective refusal with ledger state (deferred: docs/adr/BACKLOG.md)
