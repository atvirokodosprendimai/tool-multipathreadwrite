# Task ADR-178-T1: `--json-path` prints the selected values and licenses no line

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw read --json-path`; its `CLI()` line; contract §304
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the projection tests run`, `the compact-value test passed`, `the splitter test passed`, `the miss test passed`, `the licence test passed`, `the expression test passed`, `the budget test passed`, `the conflict test passed`, `the JSON receipt test passed`, `the ADR-155 address test passed`, `the ADR-155 grammar test passed`, `the ADR-155 licence test passed`, `the read, cmd and guide packages pass`, `the contract run`, `section 303 exists`, `the single-line row passes`, `the stdin row passes`, `the refused-write row passes`, `the miss row passes`, `gofmt`, `the engine packages it does not own`, `no untracked file in them`

## Goal

Decisions 1-5: `--json-path` checks each expression's `$` and splits on commas outside `["…"]` (escapes honoured), resolves each with ADR-155's resolver, prints `EXPR = <compact JSON>` under the file's header, records nothing, reports misses and non-JSON as incomplete, refuses the flags it cannot act beside (ADR-179's included), keeps `-`'s rules, and carries `--json` for named files.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/project178_test.go` | new | the red unit tests: values, the splitter, misses |
| `cmd/mrw/project178_test.go` | new | the red tests through the binary: licensing, expressions, budget, conflicts, `--json` |
| `internal/read/jsonpath.go` | edit | the byte span split out of `jsonSpan` (`:115`); the splitter; the `$` check before `parseJSONPath` (`:40`) |
| `internal/read/read.go` | edit | the projection in `Run` (`:614`) when `Options` asks for it, recording nothing |
| `cmd/mrw/main.go` | edit | the flag on `read` (`:896-951`) and its conflict rules beside the others (`:1056-1127`) |
| `cmd/mrw/stdin.go` | edit | the projection on `-` (`:34`), its `Options` (`:92-99`), its existing refusals kept (`:35-42`) |
| `internal/guide/guide.go` | edit | `--json-path` named in `CLI()` (`:75`), which `TestEveryReadFlagIsTaughtByInstructions` requires |
| `scripts/contract.sh` | edit | §304 |

## Ordered Steps

1. [S1] Write the tests in the table and see them RED: compact values from a single-line and a pretty-printed document, a string with escapes kept byte for byte; `$["a,b"],$["x\"y"],$.c` is three expressions; `NOT FOUND` with the others still printed and `NOT JSON` for a JSON Lines file; after a projection of `f.json` the ledger holds no entry for it and a plan replacing the value's line is refused naming it; `--json-path ''`, `x.a`, `$.` exit 2, and two `--json-path` flags keep command-line order (through the binary); `--max-chars` that fits one of two values names the second on stderr, exit 1; each of Decision 5's refused flags exits 2 naming it, and `--json-path P - --json` exits 2; `--json` on a named file gives `content` exactly the plain output and an empty `observed`.
2. [S2] The projection. Mutant: the value printed with `json.Indent` instead of `json.Compact`. [proof: mutation]
3. [S3] Nothing recorded. Mutant: the projection records the value's span. [proof: mutation]
4. [S4] The splitter. Mutant: split on every comma. [proof: mutation]
5. [S5] The `$` check. Mutant: the check removed, so `x.a` parses as `$.a`. [proof: mutation]
6. [S6] Misses and the budget. Mutant: a `NOT FOUND` expression does not count as a problem. [proof: mutation]
7. [S7] Contract §304: single-line JSON projected through the built binary, piped JSON projected, the write after a projection refused and after a path-address read allowed, a miss exiting 1. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./cmd/mrw/ -count=1 -v -run '^(TestAJSONPathProjectionPrintsCompactValues|TestACommaInsideABracketedNameDoesNotSplit|TestAMissingJSONPathIsIncomplete|TestAProjectionLicensesNoLine|TestAProjectionExpressionMustBeAPath|TestAProjectionTheBudgetDropsIsListed|TestAProjectionRefusesTheFlagsThatServeLines|TestAProjectionUnderJSONIsTheContent|TestAJSONPathAddressesTheLinesOfItsValue|TestAJSONPathIsParsedOnlyInTheFormsItHas|TestAJSONPathReadLicensesExactlyItsLines)$' 2>&1) \
  && grep -qE '^--- PASS: TestAJSONPathProjectionPrintsCompactValues \(' <<<"$out" \
  && grep -qE '^--- PASS: TestACommaInsideABracketedNameDoesNotSplit \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAMissingJSONPathIsIncomplete \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAProjectionLicensesNoLine \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAProjectionExpressionMustBeAPath \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAProjectionTheBudgetDropsIsListed \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAProjectionRefusesTheFlagsThatServeLines \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAProjectionUnderJSONIsTheContent \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAJSONPathAddressesTheLinesOfItsValue \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAJSONPathIsParsedOnlyInTheFormsItHas \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAJSONPathReadLicensesExactlyItsLines \(' <<<"$out" \
  && go test ./internal/read/ ./cmd/mrw/ ./internal/guide/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 304\.' scripts/contract.sh \
  && grep -qF 'PASS  --json-path projects a value out of single-line JSON' <<<"$c" \
  && grep -qF 'PASS  --json-path projects a value out of piped JSON' <<<"$c" \
  && grep -qF 'PASS  a projection licenses no line, and the path address still does' <<<"$c" \
  && grep -qF 'PASS  a --json-path miss exits 1 incomplete' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAJSONPathProjectionPrintsCompactValues` | `internal/read/project178_test.go` | compact values, bytes of strings kept, one-line and pretty input | none | S1, S2 |
| `TestACommaInsideABracketedNameDoesNotSplit` | `internal/read/project178_test.go` | the splitter, escapes inside `["…"]` | none | S1, S4 |
| `TestAMissingJSONPathIsIncomplete` | `internal/read/project178_test.go` | `NOT FOUND`, `NOT JSON`, the rest still printed | none | S1, S6 |
| `TestAProjectionLicensesNoLine` | `cmd/mrw/project178_test.go` | no ledger entry; the write is refused | none | S1, S3 |
| `TestAProjectionExpressionMustBeAPath` | `cmd/mrw/project178_test.go` | empty, `x.a`, `$.` exit 2; repeated flags keep order; the quoted comma through the CLI | none | S1, S4, S5 |
| `TestAProjectionTheBudgetDropsIsListed` | `cmd/mrw/project178_test.go` | a withheld value is named, exit 1 | none | S1, S6 |
| `TestAProjectionRefusesTheFlagsThatServeLines` | `cmd/mrw/project178_test.go` | Decision 5's refusals, ADR-179's flags and `-` with `--json` | none | S1 |
| `TestAProjectionUnderJSONIsTheContent` | `cmd/mrw/project178_test.go` | `content` equals the plain output; `observed` empty | none | S1, S3 |
| `TestAJSONPathAddressesTheLinesOfItsValue` | `internal/read/jsonpath155_test.go` | ADR-155's address unchanged by the split (CRLF, CR, BOM, duplicates) | none | S2 |
| `TestAJSONPathIsParsedOnlyInTheFormsItHas` | `internal/read/jsonpath155_test.go` | ADR-155's grammar unchanged | none | S5 |
| `TestAJSONPathReadLicensesExactlyItsLines` | `cmd/mrw/jsonpath155_test.go` | ADR-155's licence unchanged | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | the `--json-path` flag on `read`; §304 drives the built binary |
| 3 — the caller can discover it | `mrw read --help`; `CLI()` (held by `TestEveryReadFlagIsTaughtByInstructions`); the recipe in T2 |
| 4 — it is used | nothing measures this yet; the BACKLOG re-measure counts `--json-path` calls |

## Mutation Log
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S2: the value is printed with json.Indent instead of json.Compact · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · covers:the compact-value test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/read.go` · S3: the projection records the whole file as read · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · covers:the licence test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S4: the splitter cuts at every comma, inside a quoted step too · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · covers:the splitter test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S6: a NOT FOUND expression does not count as a problem · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · covers:the miss test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S6: a value the budget dropped is not listed on the withheld stream · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · covers:the budget test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `cmd/mrw/main.go` · S1: --max-lines is not refused beside --json-path · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · covers:the conflict test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S5: the $ check is removed, so x.a parses as $.a · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · covers:the expression test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S2: the value is printed with json.Indent instead of json.Compact · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the compact-value test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/read.go` · S3: the projection records the whole file as read · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the licence test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S4: the splitter cuts at every comma, inside a quoted step too · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the splitter test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S6: a NOT FOUND expression does not count as a problem · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the miss test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `cmd/mrw/main.go` · S1: --max-lines is not refused beside --json-path · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the conflict test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S5: the $ check is removed, so x.a parses as $.a · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the expression test passed
- 2026-10-27 · 5952c1ce* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S6: a value the budget dropped is not listed on the withheld stream · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the budget test passed
- 2026-10-27 · 4a80c019* · mutant killed · exit 1 · `internal/read/jsonpath.go` · S4: the escape clause inside a quoted step is deleted, so an escaped quote closes it · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the splitter test passed
- 2026-10-27 · 4a80c019* · mutant killed · exit 1 · `cmd/mrw/main.go` · S1: -N is not refused beside --json-path · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · covers:the conflict test passed

## Invariants

- A JSON path ADDRESS serves and licenses exactly what it did (ADR-155).
- Nothing a projection prints is licensed or recorded.

## Risks

- None beyond ADR-178's: no doubles; the tests read real files and real standard input.

## Stop Condition

The fence exits 0. Stop and ask if splitting the byte span out of `jsonSpan` changes any ADR-155 test's answer.

## Out of Scope

- The recipe and the documents (T2's job).

## Verification Log
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:437 · test-lock-sha256:0a7c10871dece6443e03799c201578b4e7523bf9272dba046fab41ca37af7d31 · test-lock-file:docs/adr/locks/0a7c10871dece6443e03799c201578b4e7523bf9272dba046fab41ca37af7d31.lock
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:163080
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:230356
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:296111
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:247034
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:253788
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:207776
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:237770
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:241575
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:169211
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:200573
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:214354
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:b962327d82fc40d4cd1718663c5fe20af148f98dbc7f821f5e4f1e5e9edc801d · ms:171830
- 2026-10-27 · 5952c1ce* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:0 · test-lock-sha256:0a7c10871dece6443e03799c201578b4e7523bf9272dba046fab41ca37af7d31 · test-lock-file:docs/adr/locks/0a7c10871dece6443e03799c201578b4e7523bf9272dba046fab41ca37af7d31.lock · test-lock-kind:replace
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:138632
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:145751
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:160321
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:146674
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:174219
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:137243
- 2026-10-27 · 5952c1ce* · exit 1 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:211831
  ```
  ```
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:150540
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:149219
- 2026-10-27 · 5952c1ce* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:159546
- 2026-10-27 · 4a80c019* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:0 · test-lock-sha256:acdb4c429e857b5367a8a5777eea008e5a5f2318ab704618bae7403c9866c493 · test-lock-file:docs/adr/locks/acdb4c429e857b5367a8a5777eea008e5a5f2318ab704618bae7403c9866c493.lock · test-lock-kind:replace
- 2026-10-27 · 4a80c019* · exit 1 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:141428
  ```
  ```
- 2026-10-27 · 4a80c019* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:144614
- 2026-10-27 · 4a80c019* · exit 0 · `set -o pipefail …` · acceptance-sha256:01f788a82f335424465de84b7b5b76a39988590f5acb834d5699201f81b1ac18 · ms:134992
