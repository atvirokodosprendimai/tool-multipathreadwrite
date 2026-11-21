# Task ADR-214-T2: the words and the executed probe

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** one sentence in the full instructions, the README and the MCP descriptions, and the guide's lines on writing U+FEFF; contract §380; the fence's own probe of the built binary
**Consumes:** T1's behaviour and the note's exact words; `internal/guide/guide.go`; the legacy golden and `era_test.go`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the instructions and the descriptions say a file's mark is kept and how to match it`, `the legacy golden changed by those descriptions only`, `the built binary keeps the mark of a marked file and adds none to an unmarked one`, `contract 380 runs and its control runs`, `the packages this record does not own are untouched`, `every named test passed and none skipped`

## Goal

ADR-214's words, and the row `lifecycle.md` §5 asks for, run and not only read.

- **The guide**: the full instructions say, beside the line-ending paragraph, that a file's leading byte order mark is part of line 1 for reads and patterns, that a line edit keeps it at byte zero, that `^one` misses it while `^.one` and `^\x{FEFF}one` match, how to write U+FEFF in an old side, and the one-plan recipe to remove a mark (`unlink` plus `rename`); `mrw instructions --core` stays under 300 words (ADR-141).
- **MCP**: `mrw_read`'s and `mrw_write`'s descriptions carry the same sentence; the legacy golden is regenerated once and `era_test.go`'s list names ADR-214.
- **Contract §380** drives the built binary on a marked file: the read prints the note once after the header; a replace of line 1 without the mark leaves the file beginning `ef bb bf` with the mark nowhere else; `insert-before 1` leaves it at byte zero and not on the moved line; delete-all leaves exactly `M` and a receipt `lines_after` of 1. Its rows have fixed names, among them `380 control: an unmarked file gains no mark`, and the unmarked control compares bytes with `cmp`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go`, `README.md` | edit | the sentences |
| `internal/mcp/schema.go`, `internal/mcp/testdata/legacy_golden.jsonl`, `internal/mcp/era_test.go` | edit | descriptions and the golden |
| `scripts/contract.sh` | edit | §380 |
| `internal/guide/mark214_test.go` | new | the guide test |
| `docs/adr/BACKLOG.md` | edit | the removal entry names the one-plan recipe |

## Ordered Steps

1. [S1] §380 and the guide test first. RED: on `main` the row's replace loses the mark and the guide test lacks the sentences. [proof: acceptance]
2. [S2] The words, the golden, the era note. Mutants, each killed by an EXECUTED result: the guide sentence dropped; the golden left stale; §380's control run on a marked input (its `cmp` fails, read from a FULL `./scripts/contract.sh` run's output for the row named `380 control`, which must appear as `PASS`; a deleted control shows as a missing row and the log line is checked); the product mutant "an unmarked file gains the mark" (T1's mutant, run through the fence's probe below). `bash -n scripts/contract.sh`. Then `adr-lint` on every record whose test lock names a file this task edits. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && env -u MRW_UPDATE_LEGACY_GOLDEN go test -count=1 -v ./internal/guide ./internal/mcp -run '^(TestTheInstructionsSayAFilesMarkIsKept|TestALegacyResultIsUnchangedByTheModernPath)$' > "$d/t" \
  && grep -q '^--- PASS: TestTheInstructionsSayAFilesMarkIsKept (' "$d/t" \
  && grep -q '^--- PASS: TestALegacyResultIsUnchangedByTheModernPath (' "$d/t" \
  && ! grep -q -- '--- SKIP' "$d/t" \
  && grep -qF 'once by ADR-214' internal/mcp/era_test.go \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && g=internal/mcp/testdata/legacy_golden.jsonl \
  && git show "$base:$g" > "$d/golden.base" \
  && ! cmp -s "$d/golden.base" "$g" \
  && git diff --quiet "$base" -- go.mod go.sum internal/lines internal/plan internal/seen internal/check internal/state internal/writer internal/ingest internal/apply internal/read ':(exclude)*_test.go' \
  && go build -o "$d/mrw" ./cmd/mrw \
  && mkdir "$d/r" \
  && printf '\357\273\277one\ntwo\n' > "$d/r/m.txt" \
  && printf 'one\ntwo\n' > "$d/r/u.txt" \
  && XDG_STATE_HOME="$d/st" "$d/mrw" --root "$d/r" read m.txt u.txt > "$d/read.out" 2>&1 \
  && [ "$(grep -c '^-- note: a UTF-8 byte order mark begins this file' "$d/read.out")" = 1 ] \
  && printf '@@ m.txt 1 replace anchor="one"\nONE\n@@ u.txt 1 replace anchor="one"\nONE\n' | XDG_STATE_HOME="$d/st" "$d/mrw" --root "$d/r" write --no-check - > "$d/write.out" 2>&1 \
  && printf '\357\273\277ONE\ntwo\n' | cmp - "$d/r/m.txt" \
  && printf 'ONE\ntwo\n' | cmp - "$d/r/u.txt" \
  && bash -n scripts/contract.sh
```

What each clause holds. The guide test and the legacy test pass; the golden differs from the merge base's, and the record's claim that only descriptions changed is checked in S2 by stripping the new sentences and comparing (the fence states the golden differs, the mutation step proves the delta). The T1 packages are frozen here: T2 changes no engine code, and the clause says so. The probe runs the binary this tree builds: the marked file's read prints the note once, a replace of its line 1 leaves exactly `M ONE` and two lines, and the same replace on an unmarked file leaves exactly `ONE` (a `cmp` that must succeed, so the control cannot be dropped without the fence failing and cannot pass when the product adds a mark). §380 itself is run by the contract gate, whose output S2 reads for the named control row.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheInstructionsSayAFilesMarkIsKept` | `internal/guide/mark214_test.go` | the sentences in the full text, the core under 300 words | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the guide test, the probe and §380 |
| 2 — something selects it | every `mrw instructions` and `tools/list` |
| 3 — the caller can discover it | the instructions, the descriptions, the README, the read's note |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #686 |

## Mutation Log

## Verification Log

## Invariants

- `mrw instructions --core` stays under 300 words.

## Risks

- Editing `guide.go` and `era_test.go` can move other records' test locks; S2 lints and relocks them.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI with §380 passing and its control row present; every gate in `AGENTS.md` passes; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- A receipt key for the mark (deferred: docs/adr/BACKLOG.md "From ADR-214")
