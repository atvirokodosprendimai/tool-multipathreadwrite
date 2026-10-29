# Task ADR-173-T1: a read whose only unserved ranges are the caller's own cap exits 0

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the cap's count beside `problems` in `read.Run`; the exit decision in `cmd/mrw` (named specs, walks, `mrw read -`); contract §298; the words in README, AGENTS.md and `mrw instructions`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the cap tests run`, `the cap test passed`, `the stdin test passed with its migrated expectation`, `the kind-line test passed`, `the contract run`, `section 298 exists`, `the capped row passes`, `the capped-and-missed row passes`, `the binary builds`, `the core rule says it`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1–4: `mrw read … --max-lines N` exits 0 when every range it did not serve was withheld or cut by that cap; a read that also missed something, or reached `--max-chars`, exits 1 `incomplete:` as before; withheld lines stay unlicensed; the docs stop saying the cap exits 1.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/maxlines173_test.go` | new | the red test, driving the built binary |
| `internal/read/read.go` | edit | count the cap's problems apart (`:832-840`, `:896-899`) and return or expose that count |
| `cmd/mrw/main.go` | edit | the read action's exit (`:1388-1390`) |
| `cmd/mrw/stdin.go` | edit | `mrw read -`'s exit (`:104-114`) |
| `internal/guide/guide.go` | edit | `:111`, "A read exits 1 when … lines --max-lines withheld"; core rule 5 (`:130`), embedded in the skill (`:155`) |
| `README.md` | edit | `--max-lines` row (`:212`) and the exit-code sentence |
| `AGENTS.md` | edit | `:228-229` |
| `scripts/contract.sh` | edit | §298: a capped-only read exits 0 and a write to a withheld line is refused; capped plus a missed pattern exits 1 `incomplete:`. And the three existing rows that want exit 1 for a cap (`:388`, `:394`, `:4363`, ADR-033's §14/§69) migrate to exit 0, keeping their WITHHELD/withheld assertions |
| `cmd/mrw/stdin161_test.go` | edit | `TestReadingStandardInputLicensesNothing` wants exit 1 for a capped stdin read (`:25`, `:58`): it migrates to 0, its licensing assertions kept |
| `.claude/skills/mrw/SKILL.md` | regenerate | `mrw skill --dir .claude/skills` after the core rule changes |

## Ordered Steps

1. [S1] Write `TestACallersOwnLineCapIsNotAnIncompleteRead` and see it RED on today's binary: a capped-only read (named spec, `--grep` walk, `mrw read -`, `--max-lines 0`) must exit 0 with the WITHHELD line on stdout; a capped read with a missed `/nomatch/` address must exit 1 with `mrw: incomplete:` last; a capped `--grep` that matched no file must exit 1 with `mrw: no match:` last; a capped read that reaches `--max-chars` must exit 1; a plan replacing a withheld line must be refused; `mrw read --json` of one capped span must carry `receipt.problems` 1 and of a capped span plus a missed address 2, equal to `mrw_read`'s `problems` for the same specs.
2. [S2] Count the cap's problems apart in `read.Run` and decide the exit from it in `cmd/mrw`. Mutant: the exit ignores the cap's count (today's behaviour). [proof: mutation]
3. [S3] Mutant: every problem counted as the cap's, so a capped read with a missed pattern exits 0 — the paired case must go red. [proof: mutation]
4. [S4] Contract §298 and the words in `guide.go`, README, AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestACallersOwnLineCapIsNotAnIncompleteRead|TestAReadThatExitsOneSaysWhichKind|TestReadingStandardInputLicensesNothing)$' 2>&1) \
  && for t in TestACallersOwnLineCapIsNotAnIncompleteRead TestAReadThatExitsOneSaysWhichKind TestReadingStandardInputLicensesNothing; do grep -qE "^--- PASS: $t \(" <<<"$out" || exit 1; done \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 298\.' scripts/contract.sh \
  && grep -qF 'PASS  a read capped only by its own --max-lines exits 0' <<<"$c" \
  && grep -qF 'PASS  a capped read that also missed a range exits 1 incomplete' <<<"$c" \
  && go build -o bin/mrw ./cmd/mrw && core=$(./bin/mrw instructions --core) && grep -qF 'own --max-lines withheld is not one' <<<"$core" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACallersOwnLineCapIsNotAnIncompleteRead` | `cmd/mrw/maxlines173_test.go` | capped-only exits 0 on every read form; capped + missed address and capped + `--max-chars` exit 1 `incomplete:`; a capped search with no match exits 1 `no match:`; withheld lines refuse a write; `receipt.problems` unchanged and equal to `mrw_read`'s | none | S1, S2, S3 |
| `TestAReadThatExitsOneSaysWhichKind` | `cmd/mrw/readexit146_test.go` | ADR-146's kind line still ends every exit-1 read | none | S2 |
| `TestReadingStandardInputLicensesNothing` | `cmd/mrw/stdin161_test.go` | a capped stdin read now exits 0 and still licenses nothing | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `--max-lines` on any read |
| 3 — the caller can discover it | README, AGENTS.md, `mrw instructions` |
| 4 — it is used | recipe 4 and 9 in `mrw instructions`; MRW-05 in the Rubie report is the evidence it is reached (3 of 11 failed reads) |

## Mutation Log
(empty until execute)

## Invariants

- Withheld lines license nothing (ADR-117 Decision 1).
- Every other exit-1 read is unchanged, kind line included (ADR-146).
- `--max-chars` reached exits 1 (ADR-165 Decision 1).
- `receipt.problems` and `mrw_read`'s `problems` keep their meaning (ADR-111).

## Risks

- A withholding site added later that forgets the cap's count makes that case exit 1, the safe side.

## Stop Condition

The fence exits 0. Stop and report if `read.Run`'s signature change reaches a caller outside `cmd/mrw` and `internal/mcp` that would need a behaviour change.

## Out of Scope

- `mrw_read` (permanent: boundary: no exit code; `problems` unchanged)

## Verification Log
(empty until execute)
