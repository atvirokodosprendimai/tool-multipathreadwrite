# Task ADR-164-T5: -v and -o on read get a pointer

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `foreignFlags` entries `v` and `o` in `cmd/mrw/foreignflag.go`; the pointer on the padded-value refusal
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `-v names it`, `-o names it`, `the padded shape names it`, `-z unchanged`, `the binary prints it`

## Goal

Decision 5: `mrw read --grep -v P …` (both the no-match and the padded-value shape), `mrw read -v …` and `mrw read -o …` end with a pointer.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/foreignflag.go` | edit | two entries; the comment that says `-v, -o` have no pointer |
| `cmd/mrw/main.go` | edit | the padded-argument refusal of the value after `--grep` appends the table's pointer when the flag taken as the pattern is in the table |
| `cmd/mrw/foreignflag140_test.go` | edit | line 45's negative case (`--grep -v` gets no hint) moves to `-z`; `TestEveryForeignFlagHintIsATrueSpelling` gains `-v`/`-o` cases (`--max-cols`) |
| `cmd/mrw/foreignflag164_test.go` | add | the new test |
| `scripts/contract.sh` | edit | §277 |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestAnInvertedGrepSaysThereIsNone (`mrw read --grep -v a x` ends with `there is no inverted match` through `patternFlagHint`; `mrw read --grep -v ' a' x` and `--grep -i ' a' x` keep their edge-whitespace refusal and end with their pointers; `mrw read -v x` reaches `usageError` with the pointer; `mrw read -o x` names `--max-cols`; `--grep -z a x` gets no hint). [proof: acceptance]
2. [S2] The two entries, each run against the built binary first as ADR-140's were. [proof: mutation]
3. [S3] The pointer on the padded-value refusal. [proof: mutation]
4. [S4] Rewrite `foreignflag140_test.go`'s `-v` negative case to `-z`; add the `-v`/`-o` spelling cases. [proof: acceptance]
5. [S5] Contract §277: `--grep -v` exits as today with the pointer; the pair: `--grep -i` still prints ADR-140's pointer. [proof: human: the executor runs ./scripts/contract.sh once and reads the §277 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAnInvertedGrepSaysThereIsNone$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnInvertedGrepSaysThereIsNone \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAForeignReadFlagIsAnsweredWithMrwsSpelling|TestEveryForeignFlagHintIsATrueSpelling' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAForeignReadFlagIsAnsweredWithMrwsSpelling \(' "$out" \
  && grep -qE '^--- PASS: TestEveryForeignFlagHintIsATrueSpelling \(' "$out" \
  && grep '^# 277\. ' scripts/contract.sh > /tmp/adr164-t5-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAForeignReadFlagIsAnsweredWithMrwsSpelling` | `cmd/mrw/foreignflag140_test.go` | ADR-140's entries; the negative case now on `-z` | none | S4 |
| `TestEveryForeignFlagHintIsATrueSpelling` | `cmd/mrw/foreignflag140_test.go` | the spellings, `-v`/`-o` cases added | none | S4 |

Planned, named without backticks until it exists: TestAnInvertedGrepSaysThereIsNone in `cmd/mrw/foreignflag164_test.go` (S1, S2, S3).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two entries; the padded-value pointer |
| 2 — something selects it | `patternFlagHint` and `usageError` read `flagTables` (ADR-140); the padded-argument check now does too; contract §277 drives the binary |
| 3 — the caller can discover it | the message itself |
| 4 — it is used | nothing measures this; met once by this session (2026-10-20) |

## Mutation Log
(empty until execute)

## Verification Log
(empty until execute)

## Invariants

- Exit codes unchanged; every existing `foreignFlags` entry unchanged; the padded refusal's existing words unchanged.

## Risks

- None beyond the record's.

## Out of Scope

- `-z` (permanent: boundary: no honest pointer; ADR-140)

## Stop Condition

Stop and bring it to Zy if the padded-argument check (it runs before the read, `cmd/mrw/main.go`) cannot tell which flag became the pattern without re-parsing argv.
