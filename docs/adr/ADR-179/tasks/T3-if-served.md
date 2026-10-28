# Task ADR-179-T3: `--if-served PATH=SHA` answers a block the caller and the ledger both hold by reference, opt-in

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw read --if-served PATH=SHA`; the `same_as_served[]` receipt key; contract §307
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the tests run`, `TestIfServedAnswersASpanTheCallerAndTheLedgerHold passed`, `TestIfServedWithoutTheCallersShaServesTheText passed`, `TestIfServedServesAFileMrwWroteButDidNotShow passed`, `TestIfServedKeepsAMissedRangesReport passed`, `TestIfServedServesAgainAfterForget passed`, `TestIfServedIsListedInTheJSONReceipt passed`, `TestNoShippedReceiptFieldDisappears passed`, `the touched packages pass whole`, `the contract run`, `section 307 exists`, `the row: --if-served answers a block read at this sha by reference`, `the row: --if-served without the sha the caller holds serves the text`, `the row: after mrw seen --forget --if-served serves the text`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decision (d): a spec of PATH whose file still has the sha the caller quotes, and every line of which `Observation.ServedLine` reports shown, is answered `-- same as served: lines A-B[, C-D…] at sha X` and records nothing; anything else is served; a missed range is still reported and exits 1; the reference is listed in the `--json` receipt; the conflicts are refused; after `mrw seen --forget` everything is served.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/ifserved179_test.go` | new | the red unit tests |
| `cmd/mrw/ifserved179_test.go` | new | the red tests through the binary |
| `internal/read/read.go` | edit | the answer by reference beside `IfChanged` (`:191-200`, `:733-762`), given the ledger snapshot and the caller's shas |
| `cmd/mrw/main.go` | edit | the flag, its conflicts, the snapshot passed to `Run` |
| `internal/mcp/readreceipt.go` | edit | `same_as_served[]` in the read receipt `mrw read --json` shares with `mrw_read` |
| `docs/receipts.txt` | edit | `mcp_read same_as_served[].path`, `.sha`, `.spans` |
| `internal/guide/guide.go` | edit | the flag named in `CLI()` (`:77`), which `TestEveryReadFlagIsTaughtByInstructions` (`cmd/mrw/instructions_test.go:32`) requires the moment the flag exists |
| `scripts/contract.sh` | edit | §307 |

## Ordered Steps

1. [S1] Write `TestIfServedAnswersASpanTheCallerAndTheLedgerHold`, `TestIfServedWithoutTheCallersShaServesTheText`, `TestIfServedServesAFileMrwWroteButDidNotShow`, `TestIfServedKeepsAMissedRangesReport`, `TestIfServedServesAgainAfterForget`, `TestIfServedIsListedInTheJSONReceipt`, `TestNoShippedReceiptFieldDisappears` and see them RED.
2. [S2] The answer. Mutant: the coverage check reads only the spec's first line. [proof: mutation]
3. [S3] Written files. Mutant: a `Written` observation's whole licence counts as shown. [proof: mutation]
4. [S4] The caller's sha. Mutant: the quoted sha ignored. [proof: mutation]
5. [S5] Forget: a fresh process reads the ledger `mrw seen --forget` emptied. [proof: acceptance]
6. [S6] Missed ranges. Mutant: a missed range beside a reference is not counted a problem. [proof: mutation]
7. [S7] Contract §307: the built binary, the good case beside the case that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/read/ -count=1 -v -run '^(TestIfServedAnswersASpanTheCallerAndTheLedgerHold|TestIfServedWithoutTheCallersShaServesTheText|TestIfServedServesAFileMrwWroteButDidNotShow|TestIfServedKeepsAMissedRangesReport|TestIfServedServesAgainAfterForget|TestIfServedIsListedInTheJSONReceipt|TestNoShippedReceiptFieldDisappears)$' 2>&1) \
  && grep -qE '^--- PASS: TestIfServedAnswersASpanTheCallerAndTheLedgerHold \(' <<<"$out" \
  && grep -qE '^--- PASS: TestIfServedWithoutTheCallersShaServesTheText \(' <<<"$out" \
  && grep -qE '^--- PASS: TestIfServedServesAFileMrwWroteButDidNotShow \(' <<<"$out" \
  && grep -qE '^--- PASS: TestIfServedKeepsAMissedRangesReport \(' <<<"$out" \
  && grep -qE '^--- PASS: TestIfServedServesAgainAfterForget \(' <<<"$out" \
  && grep -qE '^--- PASS: TestIfServedIsListedInTheJSONReceipt \(' <<<"$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/guide/ ./internal/mcp/ ./internal/read/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 307\.' scripts/contract.sh \
  && grep -qF 'PASS  --if-served answers a block read at this sha by reference' <<<"$c" \
  && grep -qF 'PASS  --if-served without the sha the caller holds serves the text' <<<"$c" \
  && grep -qF 'PASS  after mrw seen --forget --if-served serves the text' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestIfServedAnswersASpanTheCallerAndTheLedgerHold` | `internal/read/ifserved179_test.go` | covered by the union of two reads, sha matching: by reference with both spans listed; one line unserved, or the file changed: served | none | S1, S2 |
| `TestIfServedWithoutTheCallersShaServesTheText` | `cmd/mrw/ifserved179_test.go` | a fresh caller quoting no sha, or a wrong one, over a ledger that covers the lines: served | none | S1, S4 |
| `TestIfServedServesAFileMrwWroteButDidNotShow` | `cmd/mrw/ifserved179_test.go` | after `mrw write --create` the file is served; after a read of lines 1-3 (which serves the line after, 4, ADR-161) lines 1-4 are referenced and a range reaching line 5 is served (ADR-118) | none | S1, S3 |
| `TestIfServedKeepsAMissedRangesReport` | `internal/read/ifserved179_test.go` | a pattern that matches nothing beside a referenced range: `!! no match`, exit 1 | none | S1, S6 |
| `TestIfServedServesAgainAfterForget` | `cmd/mrw/ifserved179_test.go` | read, by reference, `mrw seen --forget`, served again | none | S1, S5 |
| `TestIfServedIsListedInTheJSONReceipt` | `cmd/mrw/ifserved179_test.go` | `same_as_served[]` with path, sha and spans; the conflicts exit 2 | none | S1 |
| `TestNoShippedReceiptFieldDisappears` | `cmd/mrw/receipts111_test.go` | the receipt and `docs/receipts.txt` agree | none | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the flag (or the variable) in `cmd/mrw/main.go`; §307 drives the built binary |
| 3 — the caller can discover it | `--help`; the instructions in T9 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure |

## Mutation Log
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the coverage check reads only the spec first line (S2) · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedAnswersASpanTheCallerAndTheLedgerHold passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a Written observation whole licence counts as shown (S3) · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedServesAFileMrwWroteButDidNotShow passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the quoted sha is ignored (S4) · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedWithoutTheCallersShaServesTheText passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a missed range beside a reference is not a problem (S6) · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedKeepsAMissedRangesReport passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · delivery judged after a missed range diagnostic · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedKeepsAMissedRangesReport passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · the reference ignores -C: a widened match is judged unwidened · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedAnswersASpanTheCallerAndTheLedgerHold passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `internal/read/read.go` · a reference the budget cut is not withheld or named · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedKeepsAMissedRangesReport passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · one file under both flags is not refused up front · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedIsListedInTheJSONReceipt passed
- 2026-10-28 · 14bce654* · mutant killed · exit 1 · `cmd/mrw/main.go` · --if-served is refused beside --max-matches again · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · covers:TestIfServedWithoutTheCallersShaServesTheText passed

## Invariants

- A line not shown whole is not licensed. (`--max-lines` is a cap on a text; a reference prints no text, so `--max-lines 1` with a full reference exits 0, as (d) says.)
- Without the flag, the read or write prints what it printed before this task, plus T1's one hint line.
- `--json` receipt keys are unchanged (ADR-111).

## Risks

- None beyond ADR-179's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The recipes and the documents (T9's job); the `CLI()` line is this task's.

## Verification Log
- 2026-10-28 · 14bce654* · exit 1 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:877 · test-lock-sha256:5975b2f5599e2ebb4e4e1073cb2ccae614463bd6733f57207903cddba8b54964 · test-lock-file:docs/adr/locks/5975b2f5599e2ebb4e4e1073cb2ccae614463bd6733f57207903cddba8b54964.lock
  ```
  ```
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:0 · test-lock-sha256:4f976fd47fd226325095af3f0d86cae75f1f3a3a5528423b9d0607947e28824b · test-lock-file:docs/adr/locks/4f976fd47fd226325095af3f0d86cae75f1f3a3a5528423b9d0607947e28824b.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:142491
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:141750
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:142612
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:143565
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:136760
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:137350
- 2026-10-28 · 14bce654* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:0 · test-lock-sha256:9b30716c90a990023a91ae4312097a77e02a2c6b23ef3dc95b28c75aa00ec996 · test-lock-file:docs/adr/locks/9b30716c90a990023a91ae4312097a77e02a2c6b23ef3dc95b28c75aa00ec996.lock · test-lock-kind:replace
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:145281
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:142647
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:161812
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:165780
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:167170
- 2026-10-28 · 14bce654* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3fd85cddbfd512fa95b07c0b80633f39ccf6011038671791f5f9616d5c647b0 · ms:138300
