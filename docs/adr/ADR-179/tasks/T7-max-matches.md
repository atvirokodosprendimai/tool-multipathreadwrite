# Task ADR-179-T7: `--max-matches N` caps the matches a whole read shows, and counts the rest

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `mrw read --max-matches`; contract §311
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest passed`, `TestMaxMatchesCountsAMatchShownAsContext passed`, `TestMaxMatchesOfExactlyNSaysNothing passed`, `TestMaxMatchesLicensesOnlyWhatWasShown passed`, `the touched packages pass whole`, `the contract run`, `section 311 exists`, `the row: --max-matches shows N matches across files and counts the rest`, `the row: a match --max-matches did not show is refused a write`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (g): with `--grep` (on files, and on `-`) or `--ast-grep` (on files; `-` refuses it), matches are shown in walk order until N matching lines have been shown — a match shown as context or as the line after a range counts and is served with its own span, so a budget of at least the matches serves what the read without the flag serves — then no new span starts and the rest are counted, only those that reached the answer counting as shown; with matches left unshown the answer ends `-- max-matches N reached: M more matching line(s) in K file(s) not shown` and exits 1 `incomplete:` (naming the budget when it alone withheld); exactly N says nothing and exits 0; only what was shown is licensed; 0 shows none; without a finder, or beside `--count`, `--stat`, `--sub`, `--rewrite` or `--json-path`, it is refused. A spec `--if-changed` answers unchanged is neither shown nor counted.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/maxmatches179_test.go` | new | the red unit tests |
| `cmd/mrw/maxmatches179_test.go` | new | the red licensing test through the binary |
| `internal/read/read.go` | edit | the budget across specs in `Run` (`:654`), beside `MaxLines` (`:136-151`), aware of context (`:869`) and the line after a range (`:908`) |
| `cmd/mrw/main.go` | edit | the flag and its conflicts; `max-matches` joins ADR-178's `--json-path` refusal list (`:1057`), so that text speaks first |
| `cmd/mrw/stdin.go` | edit | the budget at standard input's `Options` (`:103-117`) |
| `internal/guide/guide.go` | edit | the flag named in `CLI()` (`:77`), which `TestEveryReadFlagIsTaughtByInstructions` (`cmd/mrw/instructions_test.go:32`) requires the moment the flag exists |
| `scripts/contract.sh` | edit | §311 |

## Ordered Steps

1. [S1] Write `TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest`, `TestMaxMatchesCountsAMatchShownAsContext`, `TestMaxMatchesOfExactlyNSaysNothing`, `TestMaxMatchesLicensesOnlyWhatWasShown` and see them RED.
2. [S2] The budget. Mutant: the budget resets for each file. [proof: mutation]
3. [S3] The licence and exit. Mutant: the read exits 0 when the budget withheld matches. [proof: mutation]
4. [S4] Contract §311: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/read/ -count=1 -v -run '^(TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest|TestMaxMatchesCountsAMatchShownAsContext|TestMaxMatchesOfExactlyNSaysNothing|TestMaxMatchesLicensesOnlyWhatWasShown)$' 2>&1) \
  && grep -qE '^--- PASS: TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest \(' <<<"$out" \
  && grep -qE '^--- PASS: TestMaxMatchesCountsAMatchShownAsContext \(' <<<"$out" \
  && grep -qE '^--- PASS: TestMaxMatchesOfExactlyNSaysNothing \(' <<<"$out" \
  && grep -qE '^--- PASS: TestMaxMatchesLicensesOnlyWhatWasShown \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ ./internal/read/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 311\.' scripts/contract.sh \
  && grep -qF 'PASS  --max-matches shows N matches across files and counts the rest' <<<"$c" \
  && grep -qF 'PASS  a match --max-matches did not show is refused a write' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest` | `internal/read/maxmatches179_test.go` | three files of four matches, N=5: five shown across two files, seven counted in two files; 0 shows none; `--max-lines` charges only the matches printed, across files | none | S1, S2 |
| `TestMaxMatchesCountsAMatchShownAsContext` | `internal/read/maxmatches179_test.go` | adjacent matches with `-C 1` and N=1: the second is shown as context and counted, its span served with it, no third span; N at least the matches serves and records what the read without the flag does (adjacent hits, `-C`, ast-grep multi-line hits); a match on the line after a span is shown | none | S1, S2 |
| `TestMaxMatchesOfExactlyNSaysNothing` | `cmd/mrw/maxmatches179_test.go` | exactly N matches: no note, exit 0; without `--grep` exit 2, as beside `--count`, `--sub` and a negative N; beside `--json-path` exit 2 with "--json-path prints values", in either flag order | none | S1, S3 |
| `TestMaxMatchesLicensesOnlyWhatWasShown` | `cmd/mrw/maxmatches179_test.go` | a write to a counted, unshown match is refused; exit 1 `incomplete:` naming the budget | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §311 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the budget resets for each file (S2) · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the read exits 0 when the budget withheld matches (S3) · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the budget resets for each file (S2) · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the read exits 0 when the budget withheld matches (S3) · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesLicensesOnlyWhatWasShown passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a match inside a span already taken opens no coverage, so it spends the budget again · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesCountsAMatchShownAsContext passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · a negative budget is accepted · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesOfExactlyNSaysNothing passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the line after a span is no longer counted shown · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesCountsAMatchShownAsContext passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a match --max-lines cut is still charged as shown · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/maxmatches179.go` · the kind line never names the budget · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesLicensesOnlyWhatWasShown passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a covered match is no longer shown once the budget is spent · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesCountsAMatchShownAsContext passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a hit is charged when only its first line was printed · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a file whose header --max-chars rejected stays counted · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · covers:TestMaxMatchesStopsServingAcrossFilesAndCountsTheRest passed

## Invariants

- A line not shown whole is not licensed.
- Without the flag, the read or write prints what it printed before this task, plus T1's one hint line.
- `--json` receipt keys are unchanged (ADR-111).

## Risks

- None beyond ADR-179's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The recipes and the documents (T9's job); the `CLI()` line is this task's.

## Verification Log
- 2026-10-27 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:321 · test-lock-sha256:1edaadfd0d671600b5fca3ecc34c9d169d0d5e8aa2930fa87211bb24991ba673 · test-lock-file:docs/adr/locks/1edaadfd0d671600b5fca3ecc34c9d169d0d5e8aa2930fa87211bb24991ba673.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:0 · test-lock-sha256:1edaadfd0d671600b5fca3ecc34c9d169d0d5e8aa2930fa87211bb24991ba673 · test-lock-file:docs/adr/locks/1edaadfd0d671600b5fca3ecc34c9d169d0d5e8aa2930fa87211bb24991ba673.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:134508
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135749
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:130095
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135013
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135252
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135092
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135578
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:0 · test-lock-sha256:f85d712e75baad766bd6b2d20f5db94fb9959850504a26a9a12d151d18201cf4 · test-lock-file:docs/adr/locks/f85d712e75baad766bd6b2d20f5db94fb9959850504a26a9a12d151d18201cf4.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135176
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135332
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135178
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:134986
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:129938
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:0 · test-lock-sha256:3c2a51686285f74f7d5df9140ea335116bc4d9ac9d993df09dddfee4c32895de · test-lock-file:docs/adr/locks/3c2a51686285f74f7d5df9140ea335116bc4d9ac9d993df09dddfee4c32895de.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:135086
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:134921
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:130360
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:133939
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:133480
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:133583
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:133985
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:206158
- 2026-10-29 · 821a88bb* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f3e137816509ce3a93841c63387343105cbef375487af2efd97554db1f917f61 · ms:0 · test-lock-sha256:0a4a88820a8f742eaa14a02d626891057500b6e40c9e0a191e9b47147b04b0b3 · test-lock-file:docs/adr/locks/0a4a88820a8f742eaa14a02d626891057500b6e40c9e0a191e9b47147b04b0b3.lock · test-lock-kind:replace
