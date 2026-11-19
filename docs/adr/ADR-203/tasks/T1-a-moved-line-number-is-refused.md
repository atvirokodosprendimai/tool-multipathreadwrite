# Task ADR-203-T1: a moved line number is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** unassigned
**Produces:** `seen.Observation.Moved` and ledger v7; apply's per-file move and the refusal; contract §349; the sentences in `mrw instructions`, `AGENTS.md` and `README.md`; the BACKLOG entries
**Consumes:** `staleMarks` (ADR-200), `seen.merge`, `parseLine`, `formatSpans`, `writer.go`'s observation of a written file
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the ledger records where a write moved numbering`, `the move accumulates and a whole read clears it`, `a numbered hunk at or below the move on an unshown line is refused`, `everything Decision 4 lists still lands`, `a v6 ledger is read as no move, and v7 is what is written`, `the other engine packages are untouched`

## Goal

ADR-203 Decisions 1 to 6. With `f.txt` of ten lines `L1` to `L10`, read whole, then written by mrw:

| First write | Then, with no read between | Outcome |
|-------------|----------------------------|---------|
| `@@ f.txt 3 insert-after` one line | `@@ f.txt 8 replace` | refused, exit 1, nothing written; the reason names line 8, `from 5 on` and the three ways out |
| the same | `@@ f.txt 2 replace` | lands: above the move |
| the same | `@@ f.txt 4 replace` | lands: the inserted line, whose number the caller wrote |
| the same | `@@ f.txt 5 replace` | refused: the first moved line |
| the same | `@@ f.txt 8 replace anchor="L7"` | lands |
| the same | `@@ f.txt /^L7$/ replace` | lands |
| the same | `mrw read f.txt:8` then `@@ f.txt 8 replace` | lands |
| the same | `--force`, and a `--grep L7 --sub X` plan | land |
| `@@ f.txt 3 replace` one line by one | `@@ f.txt 8 replace` | lands: nothing moved |
| `@@ f.txt 3-4 replace` two lines by one | `@@ f.txt 8 replace` | refused, `from 4 on` |
| `@@ f.txt 6 delete` | `@@ f.txt 8 replace` | refused, `from 6 on` |
| two hunks, `2 replace` one by one and `6 insert-after` | `@@ f.txt 5 replace`; `@@ f.txt 9 replace` | the first lands; the second is refused, `from 8 on` |
| `mrw write --create g.txt` of five lines | `@@ g.txt 4 replace` | lands: a create moves nothing |
| insert after 3, then a second write `@@ f.txt 9 insert-after anchor=…` | `@@ f.txt 6 replace` | refused, `from 5 on`: the older move stands |
| insert after 3, then `mrw read f.txt` whole | `@@ f.txt 8 replace` | lands: a whole read clears it |
| a `--format search_replace` write that adds a line (ADR-190's partial record) | a numbered hunk below it on a line of that licence | refused or landed by the same rule, the record carrying the move |

And for the ledger: after the first row's write `mrw seen` shows the record with `~5`; a ledger whose header says v6 with a `w` record is read with no move and the next write rewrites it as v7; a `w1-3~5|7-9` line round-trips; a v7 ledger read by the v6 parser's rule (a header number above its own) is left alone, which the existing ADR-150 tests assert for any newer number.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | `Moved`, the line's shape, v7, `merge` |
| `internal/seen/moved203_test.go` | new | parse, format, merge, format number |
| `internal/apply/apply.go` | edit | the per-file move; the refusal |
| `internal/apply/moved203_test.go` | new | the Goal's table against the engine |
| `internal/writer/writer.go` | edit | the observation carries the move |
| `cmd/mrw/moved203_test.go` | new | the first, seventh and create rows through the built binary |
| `internal/guide/guide.go`, `.claude/skills/mrw/SKILL.md` | edit | the refusal in the instructions |
| `scripts/contract.sh` | edit | §349 |
| `AGENTS.md`, `README.md` | edit | rule 2's sentence and the stats bullet |
| `docs/adr/BACKLOG.md` | edit | the reading set aside; the tally; the deferrals |

## Ordered Steps

1. [S1] The four tests, RED. [proof: acceptance]
2. [S2] `Moved` in the ledger: field, line shape, v7, `merge`. Mutants, each killed: the suffix is not written; it is not parsed; `merge` drops it on a same-sha write; `merge` keeps it across a whole read; a v6 header is refused as older. [proof: mutation]
3. [S3] The per-file move in apply and writer. Mutants, each killed: the first hunk is taken whether or not it changes the count; the last moving hunk is taken; the move is the hunk's first line and not the line after its body; a create records a move; a later write takes the larger value. [proof: mutation]
4. [S4] The refusal. Mutants, each killed: it fires above the move; it fires on a shown line; it fires with `anchor=`; it fires on a computed plan; it fires under `--force`; it does not fire at the move itself. [proof: mutation]
5. [S5] Contract §349 drives the built binary: the first row refused with nothing written, paired with the anchored and the re-read forms landing, and a create followed by a numbered edit landing. The instructions, `AGENTS.md`, `README.md`, BACKLOG. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/seen ./internal/apply ./cmd/mrw ./internal/guide -run '^(TestTheLedgerRecordsWhereAWriteMovedNumbering|TestANumberAWriteMovedIsRefusedUntilTheLineIsShown|TestAMovedNumberIsRefusedByTheBinary|TestTheInstructionsNameTheMovedNumberRefusal)$' > "${TMPDIR:-/tmp}/adr203-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr203-fence.out")" -eq 4 ] \
  && grep -q '^# 349\. ' scripts/contract.sh \
  && grep -q 'formatVersion = 7' internal/seen/seen.go \
  && grep -q 'an earlier write moved the lines from' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheLedgerRecordsWhereAWriteMovedNumbering` | `internal/seen/moved203_test.go` | the ledger rows of the Goal: shape, round trip, `merge` across a write, a partial read and a whole read, v6 read and v7 written | none | S1, S2 |
| `TestANumberAWriteMovedIsRefusedUntilTheLineIsShown` | `internal/apply/moved203_test.go` | every row of the Goal's table, each refusal by its reason and by the file left unchanged | none | S1, S3, S4 |
| `TestAMovedNumberIsRefusedByTheBinary` | `cmd/mrw/moved203_test.go` | the first, fifth, seventh and create rows through `mrw write`, with exit codes and `mrw seen` | none | S1, S5 |
| `TestTheInstructionsNameTheMovedNumberRefusal` | `internal/guide/moved203_test.go` | the full instructions name the refusal and its ways out; the core stays under its limit | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §349 |
| 2 — something selects it | every `mrw write` and `mrw_write` goes through apply's validation |
| 3 — the caller can discover it | the refusal's own sentence; `mrw instructions`; `AGENTS.md` |
| 4 — it is used | telemetry is refused by ADR-009; the tally of BACKLOG.md "Pre-registered for ADR-203" |

## Mutation Log

## Verification Log

## Invariants

- A plan that addresses by pattern, carries `anchor=`, is forced, or is computed by mrw lands as before.
- A file never written by mrw since its last whole read is judged as before.
- A v4 to v6 ledger is read; nothing in it is refused for a move.

## Risks

- Tests of other records that write twice by number without a read will now be refused where the first write moved lines: each is a finding to fix in the test by an anchor or a read, and every record whose locked test moves is relinted.
- The contract script has rows that chain numbered writes; the same applies.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- Width A and a receipt row naming the shift (deferred: docs/adr/BACKLOG.md)
