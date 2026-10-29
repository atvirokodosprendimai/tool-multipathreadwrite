# Task ADR-173-T1: a read whose only unserved ranges are the caller's own caps exits 0

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `read.Options.Capped`; the exit decision and the `mrw: capped:` line in `cmd/mrw` (named specs, walks, `mrw read -`); contract §318; the migrated expectations; the words in README, AGENTS.md, `docs/agents.md` and `mrw instructions`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the cap tests run`, `the new test passed`, `the kind-line test passed`, `the stdin test passed with its migrated expectation`, `the max-matches licence test passed with its migrated expectation`, `the if-served budget test passed with its migrated expectation`, `the read and guide packages pass`, `the contract run`, `section 318 exists`, `the max-lines row passes`, `the max-matches row passes`, `the missed-range row passes`, `the max-chars row passes`, `the binary builds`, `the core rule says it`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1–5: `mrw read … --max-lines N` and `mrw read --grep P --max-matches N …` exit 0, last stderr line `mrw: capped: …`, when every problem the read counted was one of those caps; a read that also missed something, was refused a path, or reached `--max-chars`, exits 1 `incomplete:` as before; a search that matched no file exits 1 `no match:` as before; withheld lines stay unlicensed; the docs stop saying a cap exits 1.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/caps173_test.go` | new | the red test |
| `internal/read/read.go` | edit | `Options.Capped *int` beside `Unshown` (`:205-215`); fill it at the three cap sites (`:1037-1043`, `:1158-1160`, `:1205-1214`) |
| `internal/mcp/readreceipt.go` | edit | the read receipt gains `capped` from `Options.Capped` (Decision 3) |
| `docs/receipts.txt` | edit | `mcp_read capped` appended (ADR-111; `TestNoShippedMCPReceiptFieldDisappears` holds the list) |
| `cmd/mrw/main.go` | edit | the read action's exit (`:1666-1672`) and its `capped:` line |
| `cmd/mrw/stdin.go` | edit | `mrw read -`'s exit (`:128-138`), the cap-only test before the no-match test |
| `cmd/mrw/maxmatches179.go` | edit | `incompleteLine` (`:8-13`): the `--max-matches` sentence is reused for the `capped:` line; the exit-1 branch stays for a cap-only read that `--max-chars` cut (ADR Existing Primitives Audit) |
| `internal/guide/guide.go` | edit | recipe 15 (`:72`, "the read exits 1"), the flag line (`:131`), core rule 5 (`:152`, embedded in `Skill` through `Core()` at `:177`) |
| `README.md` | edit | `--max-lines` row (`:237`), `--max-matches` row (`:242`), the exit-1 paragraph (`:596-600`) |
| `AGENTS.md` | edit | `:242-243`, `:286-287`, and the exit-code paragraph on a `mrw read` that exits `1` |
| `docs/agents.md` | edit | `:145`, `:153` ("exit 1 only when some remain") |
| `scripts/contract.sh` | edit | §318 (new), and the migrated rows below |
| `cmd/mrw/stdin161_test.go` | edit | migrated expectations below |
| `cmd/mrw/maxmatches179_test.go` | edit | migrated expectation below |
| `cmd/mrw/ifserved179b_test.go` | edit | migrated expectation below |
| `.claude/skills/mrw/SKILL.md` | regenerate | `mrw skill --dir .claude/skills` after core rule 5 changes |
| `docs/adr/locks/*.lock` and the Verification Logs that hold them | relock | every record `adr-lint` names after the test edits — at least ADR-179 T7 (`TestMaxMatchesLicensesOnlyWhatWasShown`) and ADR-161 T8 (`TestReadingStandardInputLicensesNothing`) |

### The expectations that change (re-taken 2026-10-29 against `e8a6a3da`)

Enumerated with `grep -n -- '--max-lines\|--max-matches' scripts/contract.sh` (32 lines) and `grep -ln -- 'max-lines\|max-matches' cmd/mrw/*_test.go internal/read/*_test.go` (22 files), each hit read for an exit expectation:

| Site | Today | After |
|------|-------|-------|
| contract §14, `:387-388` — `want 1 … "a --max-lines cap -> exit 1"` | 1 | 0, the `withheld` assertion kept |
| contract §14, `:393-394` — `want 1 … "a span withheld whole -> exit 1"` | 1 | 0, the `WITHHELD` assertion kept |
| contract §69, `:4371-4372` — `want 1 … "a cap of zero serves nothing, and a read that served nothing is an error"` | 1 | 0, the claim reworded; the `WITHHELD 3 line(s)` and no-content assertions kept |
| contract §311, `:10567-10570` — `rc311 = 1` for `--max-matches 3` over 4 matches | 1 | 0, the row's `ok` text unchanged (ADR-179 T7's fence greps it); the note and the N=total pair kept |
| `cmd/mrw/stdin161_test.go:25` — `read --max-lines 1 -` | 1 | 0, the licensing assertions kept |
| `cmd/mrw/stdin161_test.go:53-60` — a CR-only match withheld by `--max-lines 0` | 1 `incomplete:` | 0 `capped:`; the comment at `:53-54` rewritten; the `no match:` sibling at `:55` stays 1 |
| `cmd/mrw/maxmatches179_test.go:43-47` — `TestMaxMatchesLicensesOnlyWhatWasShown` | 1 `incomplete: --max-matches 1 reached, …` | 0 `capped: --max-matches 1 reached, …`; the write assertions at `:49-53` kept |
| `cmd/mrw/ifserved179b_test.go:16-17` — `TestIfServedBesideAMatchBudget`, stale sha beside `--max-matches 2` | 1 | 0, the budgeted-text assertions kept |

Read and left as they are: `cmd/mrw/quiet161_test.go:53-55` compares `--quiet`'s exit with the plain read's, and both move together; `cmd/mrw/notices179_test.go:29` and `cmd/mrw/stat133_test.go:24` discard the exit; contract `:318`, `:10573` ignore the read's exit; every other hit is a refusal (exit 2) of `--max-lines` or `--max-matches` beside another flag, or a cap on `mrw_read` (contract §217, MCP, Decision 3). `internal/read/*_test.go` assert `problems`, which keeps its meaning (Decision 3).

## Ordered Steps

1. [S1] Write `TestACapTheCallerSetIsNotAnIncompleteRead` in `cmd/mrw/caps173_test.go` and see it RED on today's code. Cap-only reads must exit 0 with the withheld notice on stdout and `mrw: capped:` as the last stderr line: a named spec under `--max-lines 3`, a spec of three ranges under `--max-lines 2` (a span withheld whole), a `--grep` walk under `--max-lines 1`, `mrw read -` under `--max-lines 1`, `--max-lines 0`, a `--grep` walk under `--max-matches 1` (stderr `mrw: capped: --max-matches 1 reached, 2 matching line(s) not shown`), and `--max-matches 0`. Paired cases must exit 1: a capped read with a missed `/nomatch/` address (`mrw: incomplete:` last), a capped read beside a REFUSED path (`mrw: incomplete:`), a capped read that reaches `--max-chars` (`mrw: incomplete:`, `--max-chars N reached`), `printf 'hit\nhit\nhit\n' | mrw read --grep hit --max-matches 0 --max-chars 1 -`, where the budget drops only the max-matches note (`mrw: incomplete: --max-matches 0 reached, 3 matching line(s) not shown; --max-chars 1 reached`), a capped `--grep` that matched no file (`mrw: no match:` last), and `mrw read --grep z --max-lines 0 -` over input with no `z` (`mrw: no match:`). A plan replacing a withheld line and one replacing an unshown match are refused. `mrw read --json` of one capped span carries `receipt.problems` 1, and of a capped span plus a missed address 2.
2. [S2] `Options.Capped` filled at the three sites, the exit decided from it in `main.go` and `stdin.go`, and the read receipt's `capped` set from it (`mrw read --json` and `mrw_read` alike; `docs/receipts.txt` gains `mcp_read capped`). Mutant: the exit ignores the caps' count (today's behaviour) — the cap-only cases go red. Mutant: `capped` left at zero — the receipt assertion goes red. [proof: mutation]
3. [S3] Mutant: every problem counted as a cap's, so a capped read with a missed pattern exits 0 — the paired case must go red. Mutant: the `--max-chars` test dropped from the cap-only condition — the `--max-chars` pairs must go red, the one where `--max-chars` drops only the max-matches note included, since there `--max-chars` adds no problem of its own. Mutant: `mrw read -` keeps today's exit while the named-spec path changes — the `mrw read -` cases must go red. [proof: mutation]
4. [S4] Migrate the expectations in the table above, keeping each one's other assertions, and relock what `adr-lint` names. [proof: acceptance]
5. [S5] Contract §318 and the words in `guide.go`, README, AGENTS.md, `docs/agents.md`; regenerate the skill. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^(TestACapTheCallerSetIsNotAnIncompleteRead|TestAReadThatExitsOneSaysWhichKind|TestReadingStandardInputLicensesNothing|TestMaxMatchesLicensesOnlyWhatWasShown|TestIfServedBesideAMatchBudget)$' 2>&1) \
  && for t in TestACapTheCallerSetIsNotAnIncompleteRead TestAReadThatExitsOneSaysWhichKind TestReadingStandardInputLicensesNothing TestMaxMatchesLicensesOnlyWhatWasShown TestIfServedBesideAMatchBudget; do grep -qE "^--- PASS: $t \(" <<<"$out" || exit 1; done \
  && go test ./cmd/mrw/ ./internal/read/ ./internal/guide/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 318\.' scripts/contract.sh \
  && grep -qF 'PASS  a read capped only by its own --max-lines exits 0 and says capped' <<<"$c" \
  && grep -qF 'PASS  a read capped only by its own --max-matches exits 0 and says capped' <<<"$c" \
  && grep -qF 'PASS  the pair: a capped read that also missed a range exits 1 incomplete' <<<"$c" \
  && grep -qF 'PASS  the pair: a capped read that reached --max-chars exits 1 incomplete' <<<"$c" \
  && grep -qx 'mcp_read capped' docs/receipts.txt \
  && go build -o bin/mrw ./cmd/mrw && core=$(./bin/mrw instructions --core) && grep -qF 'own --max-lines or --max-matches withheld is not one' <<<"$core" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

Red at authoring, checked 2026-10-29 at `e8a6a3da`: `git grep -n -e TestACapTheCallerSetIsNotAnIncompleteRead -e 'mrw: capped' -e caps173` and `grep -c '^# 318\. ' scripts/contract.sh` return nothing and 0, and the four `PASS` strings and the core-rule sentence appear nowhere in the tree. The two `internal/read` clauses are dropped from the go/no-go lines because this record owns that package (Decision 5).

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACapTheCallerSetIsNotAnIncompleteRead` | `cmd/mrw/caps173_test.go` | cap-only reads exit 0 with `capped:` on every read form, `--max-lines` and `--max-matches`; capped + missed, capped + refused and capped + `--max-chars` exit 1 `incomplete:`; capped searches with no match exit 1 `no match:`; withheld lines refuse a write; `receipt.problems` unchanged and `receipt.capped` counts the caps' problems (`capped == problems` on a cap-only read, `capped < problems` on capped + missed) on `--json` and on `mrw_read` | none | S1, S2, S3 |
| `TestAReadThatExitsOneSaysWhichKind` | `cmd/mrw/readexit146_test.go` | ADR-146's kind line still ends every exit-1 read | none | S2 |
| `TestReadingStandardInputLicensesNothing` | `cmd/mrw/stdin161_test.go` | a capped stdin read exits 0 and still licenses nothing; a stdin search with no match still says `no match:` | none | S4 |
| `TestMaxMatchesLicensesOnlyWhatWasShown` | `cmd/mrw/maxmatches179_test.go` | a `--max-matches` read exits 0 `capped:` and still licenses only what it showed | none | S4 |
| `TestIfServedBesideAMatchBudget` | `cmd/mrw/ifserved179b_test.go` | a stale sha beside `--max-matches` serves the budgeted text and exits 0 | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `--max-lines` on any read, `--max-matches` on a search |
| 3 — the caller can discover it | the `capped:` line; README, AGENTS.md, `docs/agents.md`, `mrw instructions` |
| 4 — it is used | recipes 4, 9 and 15 in `mrw instructions`; MRW-05 in the Rubie report is the evidence the cap is reached (3 of 11 failed reads); no adoption is measured (ADR-009) |

## Mutation Log
(empty until execute)
- 2026-10-29 · 821a88bb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the named-spec exit ignores the caps count (today behaviour) · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c
- 2026-10-29 · 821a88bb* · mutant killed · exit 1 · `internal/mcp/readreceipt.go` · S2: the receipt capped left at zero · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c
- 2026-10-29 · 821a88bb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: every problem counted as a cap, so a capped read with a missed range exits 0 · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c
- 2026-10-29 · 821a88bb* · mutant survived · exit 0 · `cmd/mrw/main.go` · S3: the --max-chars test dropped from the named-spec cap-only condition · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-29 · 821a88bb* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S3: the --max-chars test dropped from the mrw read - cap-only condition (budget drops only the max-matches note) · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c
- 2026-10-29 · 821a88bb* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S3: mrw read - keeps the old exit while the named-spec path changes · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c
- 2026-10-29 · 821a88bb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: the --max-chars test dropped from the named-spec cap-only condition (cut drops only the withheld note) · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c

## Invariants

- Withheld and unshown lines license nothing (ADR-117 Decision 1, ADR-179 (g)).
- Every other exit-1 read is unchanged, kind line included (ADR-146).
- `--max-chars` reached exits 1 (ADR-165 Decision 1).
- `receipt.problems` and `mrw_read`'s `problems` keep their meaning (ADR-111).
- `read.Run`'s signature does not change.

## Risks

- A withholding site added later that forgets the caps' count makes that case exit 1, the safe side.
- A test edit moves another record's lock; `adr-lint` over every record is run before the commit.

## Stop Condition

The fence exits 0 and `adr-lint` passes for every record. Stop and report if a caller of `read.Run` outside `cmd/mrw` would need a behaviour change, or if a contract row or test not in the table above turns out to expect exit 1 for a cap.

## Out of Scope

- `mrw_read` (permanent: boundary: no exit code; `problems` unchanged)

## Verification Log
(empty until execute)
- 2026-10-29 · 821a88bb* · exit 1 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:2365 · test-lock-sha256:bde19cb6ccb768ae91ca2c1b7bd48e20860f26f5ace688f9bcbb0636fcd94d97 · test-lock-file:docs/adr/locks/bde19cb6ccb768ae91ca2c1b7bd48e20860f26f5ace688f9bcbb0636fcd94d97.lock
  ```
  ```
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:168478
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:156254
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:177884
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:147967
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:182207
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:152832
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:144011
- 2026-10-29 · 821a88bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:143838
- 2026-10-29 · 821a88bb* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:0 · test-lock-sha256:ff76b39e56250bb67ab6aa3972ac0dfd3390ce5f573d78ec3251771df6bd028c · test-lock-file:docs/adr/locks/ff76b39e56250bb67ab6aa3972ac0dfd3390ce5f573d78ec3251771df6bd028c.lock · test-lock-kind:replace
- 2026-10-29 · 821a88bb* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:e0a92cb2b84062f2d73a1d9a22d261a686cccae3f7988104d469ddb142832d5c · ms:0 · test-lock-sha256:af39c4c576fc5c2fe95774544073ab3d5671e738888d6af9d66049e9f5ef0af1 · test-lock-file:docs/adr/locks/af39c4c576fc5c2fe95774544073ab3d5671e738888d6af9d66049e9f5ef0af1.lock · test-lock-kind:replace
