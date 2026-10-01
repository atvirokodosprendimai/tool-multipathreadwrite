# Task ADR-098-T2: The served text says what the code does

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** the corrected instructions, tool descriptions and write outputSchema; contract §190
**Consumes:** `after` with `ast_grep` (`afterCursor`) (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `each corrected claim is served`, `the instructions stay under the bound`, `the legacy golden is regenerated`, `a contract row drives the binary`, `only engine comments change`, `the tree is gofmt-clean`

## Goal

Every row M7–M15 and H7 of ADR-098's Context table is corrected in the text the server sends, the two
engine comments match the code beside them, and the instructions stay within 4,096 bytes.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go` | edit | M8 guards, M11 the CLI's check, M14 `elided`, M15 the duplicate sentence (`instructionsText`, `:62-117`); M12 and M13 are NOT added here but to the plan description in `mcp.go`, which the 4,096-byte bound does not cover |
| `internal/mcp/mcp.go` | edit | H7 `after` (`:463`), M8 plan guards (`:546`), M9 `exclude` (`:452`), M10 `specs` and `ast_grep` (`:432`, `:446`), M11 write description (`:491-492`), M12/M13 plan description |
| `internal/mcp/schema.go` | edit | M7 `hunks.status` (`:253`), M14 `elided` (`:227`) |
| `internal/apply/apply.go` | edit | comment only, `:38-41` (M7's engine twin) |
| `internal/read/read.go` | edit | comment only, `:589` (the `apply.go:748` pointer) |
| `internal/mcp/says098_test.go` | add | the test below |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerate | `MRW_UPDATE_LEGACY_GOLDEN=1`, logged in `era_test.go`'s comment block (`:214-224`) |
| `internal/mcp/era_test.go` | edit | the comment block names this regeneration |
| `scripts/contract.sh` | edit | §190 |

**What selects it:** `instructionsText()` is served by `initialize`; `tools()` by `tools/list`;
`writeSchema()` as mrw_write's outputSchema. Each is already wired; §190 reads them from the built binary.

## Ordered Steps

1. [S1] Write `TestTheServedTextSaysWhatTheCodeDoes` in `internal/mcp/says098_test.go` and confirm it RED
   on T1's tree (every row still carries the old claim). [proof: mutation]
2. [S2] Correct each row, measuring the instructions after every edit; M15 pays for M8, M11 and M14 in the
   instructions, and M12 and M13 go to the plan description. Keep the literal `--check` (`TestTheSurfaceSaysTheCLIIsRicher`, §50) and pass §43's
   `_adr035_desc_problems`. [proof: mutation] Mutants: each corrected phrase reverted one at a time (the test
   names the row that went back).
3. [S3] Correct the two engine comments, and nothing else in the engine.
   [proof: human: read each comment against the code beside it, apply.go:589-594 and apply.go:1169; a comment has no behaviour a test can pin]
4. [S4] Regenerate the golden and log it in `era_test.go`. [proof: acceptance]
5. [S5] Contract §190, driving `$MRW mcp`: `initialize` and `tools/list` carry "next_index", "empty", the
   refused-`exclude` sentence and the rename shape; paired, neither carries "next_index is absent" nor "Only
   meaningful with" — `next_read is absent` is correct and stays.
   [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §190's rows printed — the whole contract takes minutes, so the fence greps the section and this step runs it]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestTheServedTextSaysWhatTheCodeDoes|TestALegacyResultIsUnchangedByTheModernPath|TestTheInstructionsTellAHostHowToAuthorAPlan|TestMCPInstructionsContainShared|TestTheSurfaceSaysTheCLIIsRicher|TestTheStatusDescriptionNamesTheValuesTheEngineSends|TestAnAstGrepIndexPagesToTheEnd' -v 2>&1 | tee "$out" \
  && missing=$(for t in TestTheServedTextSaysWhatTheCodeDoes TestALegacyResultIsUnchangedByTheModernPath TestTheInstructionsTellAHostHowToAuthorAPlan TestMCPInstructionsContainShared TestTheSurfaceSaysTheCLIIsRicher TestTheStatusDescriptionNamesTheValuesTheEngineSends TestAnAstGrepIndexPagesToTheEnd; do grep -qE "^--- PASS: $t \(" "$out" || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 190\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && base="$(git merge-base HEAD origin/main)" \
  && eng="$(git diff -U0 "$base" -- internal/apply/apply.go internal/read/read.go)" \
  && [ -z "$(printf '%s\n' "$eng" | grep -E '^[-+][^-+]' | grep -vE '^[-+][[:space:]]*//')" ] \
  && names="$(git diff --name-only "$base" -- internal/apply internal/read)" \
  && [ -z "$(printf '%s\n' "$names" | grep -v '^$' | grep -vxE 'internal/apply/apply.go|internal/read/read.go')" ] \
  && git diff --quiet "$base" -- internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc | grep '^??')" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

The corrected-text test carries the verdict and is red on T1's tree. The engine clauses allow COMMENT
lines in exactly `internal/apply/apply.go` and `internal/read/read.go`, no other engine change and no
untracked engine file. Each `git diff` is its own `&&` step, so a git failure (a bad merge-base) fails the
fence instead of reading as an empty diff — the `|| true` form an earlier draft used masked exactly that.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheServedTextSaysWhatTheCodeDoes` | `internal/mcp/says098_test.go` | one subtest per row, read from `instructionsText()`, `tools()` and `writeSchema()`: H7 `after` says `next_index` is empty, not absent (and `next_read is absent` still stands); M7 `hunks.status` names the dry run; M8 says create, unlink and rename refuse `anchor=`/`lines=`; M9 `exclude` is refused without a finder; M10 `specs` names `ast_grep`; M11 says the CLI's write runs the project's check after a code change when a check is declared or inferred, and this surface runs none; M12 rename's `-` address and one-line body; M13 the served-line-after-End rule and its last-line exception; M14 `elided` names skipped verdicts and UNWRITTEN file records; M15 "With a shell prefer the CLI" appears once | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the corrected strings |
| 2 — something selects it | `initialize`, `tools/list` and the outputSchema serve them; §190 reads them from the built binary |
| 3 — the caller can discover it | it is the handshake every MCP host reads |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-29 audit |

## Mutation Log
- 2026-09-29 · 360d572 · mutant killed · exit 1 · `internal/mcp/mcp.go` · H7 reverted: after says absent · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/schema.go` · M7 reverted: status forgets the dry run · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/instructions.go` · M8 reverted in the instructions · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/mcp.go` · M9 reverted: exclude optional again · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/mcp.go` · M10 reverted: specs forget ast_grep · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/mcp.go` · M11 reverted in the write description · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/mcp.go` · M12 reverted: rename unshaped · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/mcp.go` · M13 reverted: the last-line exception dropped · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/schema.go` · M14 reverted: elided forgets written files · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/instructions.go` · M15 reverted: the routing sentence twice · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:each corrected claim is served
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/apply/apply.go` · an engine STATEMENT changed, not a comment · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:only engine comments change
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the instructions pushed past 4096 bytes · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:the instructions stay under the bound
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/testdata/legacy_golden.jsonl` · the golden not regenerated: it still holds an old claim · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:the legacy golden is regenerated
- 2026-09-29 · 360d572* · mutant killed · exit 1 · `internal/mcp/says098_test.go` · a changed file not gofmt-clean · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · covers:the tree is gofmt-clean

## Invariants

- The instructions stay ≤ 4,096 bytes; the bound is not raised.
- `--check` stays in the instructions; §43's wording rules hold.
- No engine statement changes.

## Risks

- The byte budget: measured after each edit (4,035 of 4,096 bytes when done); M12 and M13 live in the plan description, outside the bound.

## Stop Condition

Stop and ask if the corrections cannot fit in 4,096 bytes, or if an existing text test must be weakened.

## Out of Scope

- `internal/guide` and `--help` — the record's Out of Scope.
- The behaviour — T1.

## Verification Log
- 2026-09-29 · db39d42* · exit 1 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1933 · test-lock-sha256:a2a7f60716de09ba089f2dd778159c979a9e1d01e57d47a2fdd2131120f1bbf4 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9zYXlzMDk4X3Rlc3QuZ28JTTE1IG9uZSBjb3B5IG9mIHRoZSByb3V0aW5nIHNlbnRlbmNlCTU1NjQ0OTlkYThjYjQ4MzRlMzVmMGJjZWQ3MDhjZmIxNjllMWJlMjBiMDFmYThjYTFmMzNlMjY0NjkzZWJkZWYKYm9keQlpbnRlcm5hbC9tY3Avc2F5czA5OF90ZXN0LmdvCVRlc3RUaGVTZXJ2ZWRUZXh0U2F5c1doYXRUaGVDb2RlRG9lcwlhOTc4OWRjYWYzMzE2ZTRkY2MzNjU2NmU4NTgwZjA3MzIyNDcyNjJkNTUwODM4ZTQ3NmYzNzQ0ZWRiNTc5YzZi
  ```
  --- last 10 line(s) of stdout (of 500 after folding 500 raw)
      --- FAIL: TestTheServedTextSaysWhatTheCodeDoes/M11_write (0.00s)
      --- FAIL: TestTheServedTextSaysWhatTheCodeDoes/M11_instructions (0.00s)
      --- FAIL: TestTheServedTextSaysWhatTheCodeDoes/M12_rename (0.00s)
      --- FAIL: TestTheServedTextSaysWhatTheCodeDoes/M13_served_line_after_End (0.00s)
      --- FAIL: TestTheServedTextSaysWhatTheCodeDoes/M14_elided (0.00s)
      --- FAIL: TestTheServedTextSaysWhatTheCodeDoes/M14_instructions (0.00s)
      --- FAIL: TestTheServedTextSaysWhatTheCodeDoes/M15_one_copy_of_the_routing_sentence (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	1.536s
  FAIL
  ```
- 2026-09-29 · 360d572 · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1583
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1322
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1109
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1080
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1164
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1110
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1089
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1086
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1078
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1102
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1092
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1105
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1080
- 2026-09-29 · 360d572* · exit 0 · `set -o pipefail …` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:1124
- 2026-09-29 · human-observed · S3 observed 2026-09-29: apply.go:38-41 now says a hunk is ok when its file reached disk, or would have on a dry run — read against the dry-run return at apply.go:589-594, which hands back StatusOK hunks with nothing written; read.go:589 now names the THE END IS A DELIMITER block in Apply, read against apply.go:1169-1188. Only comment lines changed (the fence's engine clauses)
- 2026-09-29 · human-observed · S5 observed 2026-09-29: ./scripts/contract.sh run unpiped on this branch (rebased on 7ebdf51), exit 0 'contract holds', with §190 printed: PASS initialize and tools/list serve the corrected sentences and none of the old claims
- 2026-09-29 · human-observed · correction to the S5 row above: the contract run it cites was taken before the rebase (base db39d42), not on 7ebdf51. Re-run now on the rebased tree (360d572 + working changes): ./scripts/contract.sh exit 0 'contract holds', §189 and §190 both PASS
- 2026-10-01 · 95f5dab* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b1cb5dabb1fca86ec9d862877ee494fa858465bed130b68d4b0bb01e2f32c3e6 · ms:0 · test-lock-sha256:45e66ba60a54bd14e7f80f438151b1e738248fdc8d5351fd08b765f3f896fb18 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3NheXMwOThfdGVzdC5nbwlNMTUgb25lIGNvcHkgb2YgdGhlIHJvdXRpbmcgc2VudGVuY2UJNTU2NDQ5OWRhOGNiNDgzNGUzNWYwYmNlZDcwOGNmYjE2OWUxYmUyMGIwMWZhOGNhMWYzM2UyNjQ2OTNlYmRlZgpib2R5CWludGVybmFsL21jcC9zYXlzMDk4X3Rlc3QuZ28JVGVzdFRoZVNlcnZlZFRleHRTYXlzV2hhdFRoZUNvZGVEb2VzCWYzZWYzYTNjMGZiMWJhYjU0MWI1N2NkZGQ2MTg4ZDk1Yjc2NDI3YjI4M2JjNTIwMTRkMTM3Y2M1NTJhMjg1Yjg · test-lock-kind:replace
