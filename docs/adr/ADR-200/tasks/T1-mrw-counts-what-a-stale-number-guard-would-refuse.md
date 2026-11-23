# Task ADR-200-T1: mrw counts the numbered hunks a stale-number guard would refuse

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** unassigned
**Produces:** `apply.Result.StaleNumbered` and `StaleUnshown` (not receipt keys); `apply.Options.Computed`; `authoring.Numbering`, `RecordNumbering(root, candidates, unshown int)`, `LoadNumbering`; the `stale-number pricing:` block and `numbering` in `mrw stats`; contract §343; the sentence in `mrw instructions` and `AGENTS.md`
**Consumes:** ADR-056's marks and stats block as a pattern (`internal/apply/apply.go`, `internal/authoring`, `cmd/mrw`), `seen.Observation.ServedLine`, `writer.MutationOf`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a candidate is marked and nothing is refused`, `a landing is counted once in Land`, `a computed plan is not counted on either surface`, `stats prints three distinct counts`, `no shipped receipt key disappears`, `the instructions and AGENTS.md say numbers move`, `the engine packages this record does not own are untouched`

## Goal

ADR-200 Decisions 1 to 4. A landing that sends a numbered hunk with no `anchor=` to a file whose ledger record is whole-file written-shaped is counted, and counted again when a line it addresses is not shown in that record. Nothing a caller sends is refused that was not refused before, and no write's receipt, exit code or file bytes change.

**The predicate** (Decision 1), on a hunk that passed validation, in a write with a ledger, not forced, with `Options.Computed` false:
- op `replace`, `delete`, `insert-after` or `insert-before`;
- address typed in numbers (`N`, `N-M`, `N-$`, `N,+K`): no start pattern, not `$` alone;
- it names a line: not `insert-after 0`, not `insert-before 1`, not `insert-before` one past the last line;
- no `anchor=`, and not a quoted hunk (ADR-190);
- the observation apply selected for the file is `Written` and `Whole()`.
It is *unshown* when any line it addresses is not `ServedLine` in that observation: every line of a `replace` or `delete` range; for `insert-after N` and `insert-before N`, line N.

**`Options.Computed`** is set true by exactly the branches that build a plan from the file as it is, and by nothing else: CLI `--sub`, `--rewrite`, `--format=apply_patch`, `--format=search_replace` (`cmd/mrw/main.go`); MCP `grep`+`sub`, `ast_grep`+`rewrite`, and `format` (`internal/mcp/tools.go`). A native plan, typed or filed, leaves it false.

**The record call** is one, in `Land` (`internal/writer/flow.go`), for any result with `MutationOf(res) != None`, before a check runs; `Settle` and `Verify` make none. `RecordNumbering` takes the two counts and adds one landing, one candidate landing when candidates > 0, one would-refuse landing when unshown > 0.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/stale200_test.go` | new | the red test for the marks |
| `internal/apply/apply.go` | edit | the marks, `Options.Computed`, the two result counts |
| `internal/authoring/numbering200_test.go` | new | the red test for the counters |
| `internal/authoring/authoring.go` | edit | `Numbering` and its file |
| `internal/writer/stale200_test.go` | new | the red test for the one call site |
| `internal/writer/flow.go` | edit | record each landing in `Land` |
| `cmd/mrw/stale200_test.go` | new | the red test for the stats block and the CLI's `Computed` |
| `cmd/mrw/main.go` | edit | the block, `numbering`, `Computed` |
| `internal/mcp/stale200_test.go` | new | the red test for `mrw_write` |
| `internal/mcp/tools.go` | edit | `Computed` |
| `internal/guide/stale200_test.go` | new | the red test for the sentence |
| `internal/guide/guide.go` | edit | the sentence |
| `cmd/opencode/mrw-plugin/src/index.ts` | edit | the stats description names the block |
| `docs/receipts.txt` | edit | the four keys |
| `scripts/contract.sh` | edit | §343 |
| `AGENTS.md` | edit | the sentence, and the stats bullet |

## Ordered Steps

1. [S1] Write the six tests and see them RED. [proof: acceptance]
2. [S2] Apply marks the hunks and sums them on the result. Fixture for the shown clauses: after a write, a read of line 4 only, which leaves the record written with line 4 shown and 3 and 5 not. Mutants, each killed by `TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused`: a hunk with `anchor=` is a candidate; a shown line is unshown (the line-4 fixture); `insert-after N` is judged on line N+1 (line 4 shown, `insert-after 4` against `insert-after 3`); a computed plan is counted; a range is unshown only when its first line is; the mark fails the hunk. [proof: mutation]
3. [S3] `authoring.Numbering` and the one record call in `Land`. Mutants, each killed: `RecordNumbering` adds the candidate count instead of one (`TestNumberingCountsALandedWriteOnce`, counts 2 and 2); the call is made in `Settle` as well (`TestALandingIsCountedOnceInLand`, a write with a passing check); a dry run is counted (`TestStatsPrintsTheStaleNumberPricing`). [proof: mutation]
4. [S4] `mrw stats` prints the block and `--json` carries `numbering`; the keys are in `docs/receipts.txt`; each computing branch sets `Computed` on the CLI and over MCP. The stats fixture has three distinct values: three landings, two with a candidate, one unshown. Mutants, each killed: the block prints the candidates where it says would-refuse; the CLI's `apply_patch` branch does not set `Computed`; MCP's `sub` branch does not. [proof: mutation]
5. [S5] Contract §343 drives the built binary: the reproduction, then `mrw stats --json` shows `numbered_would_refuse` 1; then `@@ f.txt 5 replace anchor="d"`, which lands, moves `landed_writes` by one and leaves `numbered_candidates` and `numbered_would_refuse` where they were. [proof: acceptance]
6. [S6] `mrw instructions` and `AGENTS.md` say that after a write that adds or removes lines, a number from an earlier read names another line. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
[ "$(go test -count=1 -v ./internal/apply ./internal/authoring ./internal/writer ./internal/guide ./internal/mcp ./cmd/mrw -run '^(TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused|TestNumberingCountsALandedWriteOnce|TestALandingIsCountedOnceInLand|TestStatsPrintsTheStaleNumberPricing|TestAnMcpWriteIsCountedAndAComputedPlanIsNot|TestInstructionsSayNumbersMoveAfterAWrite|TestNoShippedReceiptFieldDisappears)$' | grep -c '^--- PASS: ')" -eq 7 ] \
  && grep -q '^# 343\. ' scripts/contract.sh \
  && grep -q 'numbered_would_refuse' docs/receipts.txt \
  && grep -q 'StaleUnshown' internal/apply/apply.go \
  && grep -q 'a number from an earlier read names another line' AGENTS.md \
  && grep -q 'a number from an earlier read names another line' internal/guide/guide.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused` | `internal/apply/stale200_test.go` | the reproduction lands with the same status and file bytes and counts one candidate, one unshown; then one case per clause: each op, `N-$` and `N,+K`, `$` alone, the three position-only inserts, `anchor=`, a pattern, a quoted hunk, `Computed`, `Force`, no ledger, a whole read since the write, a partial licence, a whole-file `--quiet`-shaped record, the line-4 fixture for shown, range and insert, an alias observation | none | S1, S2 |
| `TestNumberingCountsALandedWriteOnce` | `internal/authoring/numbering200_test.go` | three counters; counts of 2 and 2 add one each; 2 and 0 add a candidate landing only; 0 and 0 a landing only; a garbage file reads as zero | none | S1, S3 |
| `TestALandingIsCountedOnceInLand` | `internal/writer/stale200_test.go` | a landing is counted before its check and not again after it; a landing whose check cannot start is counted; a dry run and a refusal are not; `landed_writes` moves with the tally's landed count | none | S1, S3 |
| `TestStatsPrintsTheStaleNumberPricing` | `cmd/mrw/stale200_test.go` | the block and the JSON keys with three distinct values after real CLI writes; `--sub`, `--rewrite` where ast-grep is present, `--format=apply_patch` and `--format=search_replace` writes add a landing and no candidate; a native plan given with no format is counted | none | S1, S3, S4 |
| `TestAnMcpWriteIsCountedAndAComputedPlanIsNot` | `internal/mcp/stale200_test.go` | `mrw_write` with `plan` is counted; with `grep`+`sub` and with `format` it adds a landing and no candidate | none | S1, S4 |
| `TestInstructionsSayNumbersMoveAfterAWrite` | `internal/guide/stale200_test.go` | the sentence is in the full instructions | none | S1, S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the six tests and §343 |
| 2 — something selects it | every landing goes through `Land`, which records it |
| 3 — the caller can discover it | `mrw stats`, `docs/receipts.txt`, `AGENTS.md`, the opencode plugin's description |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is the reproduced failure and E5's reading, which stopped on calls only the ledger can class |

## Mutation Log
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/apply/apply.go` · a hunk with anchor= is a candidate, so the anchor= case must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a candidate is marked and nothing is refused
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/apply/apply.go` · a shown line is unshown, so the line-4 cases must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a candidate is marked and nothing is refused
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/apply/apply.go` · a computed plan is counted, so the computed case and the plan-maker writes must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a computed plan is not counted on either surface
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/apply/apply.go` · a range is judged on its first line only, so the range whose last line is unshown must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a candidate is marked and nothing is refused
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/authoring/authoring.go` · a landing with two candidates counts two, so the 2-and-2 case must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a landing is counted once in Land
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/writer/flow.go` · a dry run is counted, so the dry-run cases must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a landing is counted once in Land
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `cmd/mrw/main.go` · the block prints the candidates where it says would-refuse, so the three-distinct-values block must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:stats prints three distinct counts
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI apply_patch branch does not set Computed, so its write must be counted and go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a computed plan is not counted on either surface
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP sub branch does not set Computed, so its write must be counted and go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a computed plan is not counted on either surface
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/apply/apply.go` · insert-after N is judged on line N+1, so insert-after 4 with line 4 shown must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a candidate is marked and nothing is refused
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/apply/apply.go` · only ok hunks are summed, so the partial commit whose marked hunk did not reach disk must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a candidate is marked and nothing is refused
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/writer/flow.go` · Settle records the landing as well, so a write with a passing check must count twice and go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a landing is counted once in Land
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `docs/receipts.txt` · a shipped key is missing from the list, so the receipts test and the fence grep must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:no shipped receipt key disappears
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/guide/guide.go` · the instructions lose the sentence, so the guide test must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:the instructions and AGENTS.md say numbers move
- 2026-11-18 · 5c00e088* · mutant killed · exit 1 · `internal/seen/seen.go` · an engine package this record does not own is changed, so the byte-identical clause must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:the engine packages this record does not own are untouched
- 2026-11-18 · c5956f54* · mutant killed · exit 1 · `internal/apply/apply.go` · a written record licensed in part is a candidate, so that case must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a candidate is marked and nothing is refused
- 2026-11-18 · 1bfcaace* · mutant killed · exit 1 · `internal/apply/apply.go` · a start of $ with a numbered end is a candidate, so the $-N case must go red · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · covers:a candidate is marked and nothing is refused

## Verification Log
- 2026-11-18 · 5c00e088* · exit 1 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:3800 · test-lock-sha256:287db49aee92dcba9c6271b985f17b77ddf8495316190acdb81ae7fe358b6606 · test-lock-file:docs/adr/locks/287db49aee92dcba9c6271b985f17b77ddf8495316190acdb81ae7fe358b6606.lock
  ```
  --- last 10 line(s) of stderr (of 42 after folding 42 raw)
  internal/mcp/stale200_test.go:36:61: undefined: authoring.Numbering
  internal/mcp/stale200_test.go:43:23: undefined: authoring.LoadNumbering
  internal/mcp/stale200_test.go:43:62: undefined: authoring.Numbering
  internal/mcp/stale200_test.go:53:23: undefined: authoring.LoadNumbering
  internal/mcp/stale200_test.go:53:62: undefined: authoring.Numbering
  internal/mcp/stale200_test.go:60:23: undefined: authoring.LoadNumbering
  internal/mcp/stale200_test.go:60:62: undefined: authoring.Numbering
  internal/mcp/stale200_test.go:64:23: undefined: authoring.LoadNumbering
  internal/mcp/stale200_test.go:64:62: undefined: authoring.Numbering
  internal/mcp/stale200_test.go:64:62: too many errors
  ```
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:3745
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:4019
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2439
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2562
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:3132
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:6202
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:4047
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:3741
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2897
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2343
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:6344
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2325
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:4170
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:4294
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2408
- 2026-11-18 · 5c00e088* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2326
- 2026-11-18 · c5956f54* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2352
- 2026-11-18 · 1bfcaace* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2924
- 2026-11-18 · 1bfcaace* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:3746
- 2026-11-18 · b1d1687e* · exit 0 · `set -o pipefail …` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:2238
- 2026-11-18 · de2bea18* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:0 · test-lock-sha256:24da95d6cfd2f16ba2f530580f364eef21d7b1d4d96422ed2a9e657f4c3c9a93 · test-lock-file:docs/adr/locks/24da95d6cfd2f16ba2f530580f364eef21d7b1d4d96422ed2a9e657f4c3c9a93.lock · test-lock-kind:replace
- 2026-11-19 · 6cf84b40* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:0 · test-lock-sha256:9270d7dcf70427b3d27b04ddcecfde2e624478269ff96f4070b0b8146d2a1031 · test-lock-file:docs/adr/locks/9270d7dcf70427b3d27b04ddcecfde2e624478269ff96f4070b0b8146d2a1031.lock · test-lock-kind:replace
- 2026-11-19 · 69de5b37* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:0 · test-lock-sha256:7de8c35b4d5b98ae291aa151de1b6584bb1773eabb4a222cdb51f3661830cf29 · test-lock-file:docs/adr/locks/7de8c35b4d5b98ae291aa151de1b6584bb1773eabb4a222cdb51f3661830cf29.lock · test-lock-kind:replace
- 2026-11-20 · a4040443* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:0 · test-lock-sha256:b02ec972b8a4bff00ca7823685e13fd7159c85f4bb09816843d78a373bc837d9 · test-lock-file:docs/adr/locks/b02ec972b8a4bff00ca7823685e13fd7159c85f4bb09816843d78a373bc837d9.lock · test-lock-kind:replace
- 2026-11-23 · 31d2cca1* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f040c2c6b510562907c72f179f188e336cb53e2981f1d33d004a4c9b918a3374 · ms:0 · test-lock-sha256:6ed547245d84ab40af79cce2d4332c0458970cc5ecd8200fe31c28ddae65c7fe · test-lock-file:docs/adr/locks/6ed547245d84ab40af79cce2d4332c0458970cc5ecd8200fe31c28ddae65c7fe.lock · test-lock-kind:replace

## Invariants

- No write is refused, and no write's receipt, exit code or file bytes change, because of this task.
- The counters hold counts only: no path, no content.
- A counter that cannot be read or written never changes a write's verdict.
- `internal/seen`, `internal/plan`, `internal/read`, `internal/check` and `internal/state` are byte-identical to main.

## Risks

- A partial commit and a commit whose ledger update failed are landings `Land` records on its own branches. If no fault hook reaches them from `internal/writer`'s tests, the task says so here and in the pull request as uncovered, and does not claim them.
- `--rewrite` needs `ast-grep` on PATH; where it is absent the test skips that one case and says so.
- The landed-writes counter starts at this release, so it is smaller than the tally's.
- Named behaviours with no mutant. "The mark fails the hunk" has no line to mutate: every case of the apply test compares status, reason and bytes with the same plan run with no ledger. Setting `Computed` for `search_replace` and for `--rewrite` (CLI and MCP) cannot be observed by the count: a `search_replace` hunk is quoted and a rewrite hunk carries `anchor=`, so neither is ever a candidate. The flag is set there so the computing branches read alike, and the `--rewrite` case of the stats test shows only that such a write is a landing.
- The ledger-error branch of `Land` was driven by no test when T1 merged; `internal/writer/ledgererr200_test.go::TestALandingWhoseLedgerUpdateFailsIsCountedOnce` drives it since the follow-up. The partial commit is driven in `internal/apply`'s test (the marks survive) and not through `Land`.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- The refusal (deferred: docs/adr/BACKLOG.md)
- A receipt row naming where the numbers moved (deferred: docs/adr/BACKLOG.md)
- The narrower rule, width B (deferred: docs/adr/BACKLOG.md)
