# Task ADR-070-T2: An uncounted hunk whose first body line is `body=` is refused; contract §133

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** a parse error for `body=` written under the header
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a misplaced body count is refused`, `a counted body writes the line`, `the binary refuses it`, `no engine file changes but plan`

## Goal

`plan.Parse` (`internal/plan/plan.go:265-268`) takes every non-header line of an uncounted hunk as body, so `body=1` under a header was written into s2's file in reading 04. When a hunk has no `body=` count and its FIRST body line matches `^\s*body=` (reading 04 wrote both `body=1` and `body=status: x`), record an error naming the line and showing the header it belongs on. A counted hunk is untouched, and that is the escape for a file that really starts with such a line (`raw=true` is refused without `body=`, `plan.go:421`). The compilers' `emit` (`internal/ingest/applypatch.go:325-333`) counts only a body with an `@@` line or an empty create; it must also count a body whose first line begins `body=`, or an apply_patch `+body=3` would be refused with a message about a header the caller never wrote.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | the guard in the uncounted-body branch |
| `internal/plan/bodyline_test.go` | new | `TestABodyLineThatIsABodyCountIsRefused`, `TestACountedBodyWritesALiteralBodyCountLine` |
| `internal/ingest/applypatch.go` | edit | `emit` counts a body whose first line begins `body=` |
| `internal/ingest/bodyline_test.go` | new | `TestACompiledBodyStartingWithBodyIsCounted` |
| `scripts/contract.sh` | edit | §133 |

## Ordered Steps

1. [S1] Write the three tests; confirm the refusal test RED (today the plan parses). [proof: mutation]
   - `@@ f 1 replace` / `body=1` / `X` does not parse; the error names the plan line and says `body= belongs ON the header`. The same for `body=@b.txt`, a bare `body=`, and h4's `body=status: x`.
   - `@@ f 1 replace body=1` / `body=1` parses to a one-line body `body=1`.
   - `@@ f 1 replace` / `X` / `body=1` (not the first line) and `@@ f 1 replace` / `x body=1` (not at the start) parse unchanged.
   - An apply_patch Add File whose first line is `+body=3`, and a search_replace REPLACE whose first line is `body=3`, compile to a counted hunk that parses and writes `body=3`.
2. [S2] Add the guard; GREEN; every `internal/plan` and `cmd/mrw` test stays green. [proof: mutation]
   Mutants: the guard deleted; the regex's `^` anchor dropped (killed by `x body=1`); the first-line condition dropped; `emit` stops counting a `body=` first line.
3. [S3] §133 through the binary: the misplaced plan exits 2 (the plan does not parse) with the file unchanged; the counted plan exits 0 and writes `body=1`. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 133\. ' scripts/contract.sh \
  && go test ./internal/plan/ ./internal/ingest/ -count=1 -v -run 'TestABodyLineThatIsABodyCountIsRefused|TestACountedBodyWritesALiteralBodyCountLine|TestACompiledBodyStartingWithBodyIsCounted' 2>&1 | tee /tmp/adr070-T2.out \
  && grep -q '^--- PASS: TestABodyLineThatIsABodyCountIsRefused ' /tmp/adr070-T2.out \
  && grep -q '^--- PASS: TestACountedBodyWritesALiteralBodyCountLine ' /tmp/adr070-T2.out \
  && grep -q '^--- PASS: TestACompiledBodyStartingWithBodyIsCounted ' /tmp/adr070-T2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr070-T2.out \
  && ./scripts/contract.sh > /tmp/adr070-T2-contract.out 2>&1 \
  && grep -q '^  PASS  a body= line under the header is refused' /tmp/adr070-T2-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/plan internal/ingest)" ] \
  && go vet ./internal/plan/ ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestABodyLineThatIsABodyCountIsRefused` | `internal/plan/bodyline_test.go` | `body=N`, `body=@p`, `body=` and `body=status: x` as the first line of an uncounted body are refused; deeper or mid-line `body=` is not | — | S1, S2 |
| `TestACountedBodyWritesALiteralBodyCountLine` | `internal/plan/bodyline_test.go` | a counted body writes `body=1` as content | — | S1, S2 |
| `TestACompiledBodyStartingWithBodyIsCounted` | `internal/ingest/bodyline_test.go` | apply_patch and search_replace count a body whose first line begins `body=` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §133 |
| 2 — something selects it | every plan goes through `plan.Parse` (CLI `write`, MCP `mrw_write`, the foreign-format compilers' output) |
| 3 — the caller can discover it | the refusal shows the header the count belongs on |
| 4 — it is used | §133 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · c43095d* · exit 1 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:221 · test-lock-sha256:4c49d88375f5ecd511ee4afd6fb287d29d9122ea84de16bb95ca849b01611597 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2luZ2VzdC9ib2R5bGluZV90ZXN0LmdvCVRlc3RBQ29tcGlsZWRCb2R5U3RhcnRpbmdXaXRoQm9keUlzQ291bnRlZAlhYTE4NmY2NzM2MmMzZTgzMGU2NGVjMWI1NDdmZDc3YTM4ODBiY2QwMmI0MTc3N2U4MDQ3MjY4YTYxNTJjOGM1CmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5bGluZV90ZXN0LmdvCVRlc3RBQm9keUxpbmVUaGF0SXNBQm9keUNvdW50SXNSZWZ1c2VkCWE4YWE5NDA1YTM2OTI4OGY0OWQyMzBmMWFlYTEzOWE3Mzk0NzI5NmM5NTg2Y2Y3YTNiYzU0MTg5NzBhNGJlZDkKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlsaW5lX3Rlc3QuZ28JVGVzdEFDb3VudGVkQm9keVdyaXRlc0FMaXRlcmFsQm9keUNvdW50TGluZQkxODA5ZDFlMjQ2ZmM5NWZkMDZkOGEwY2RjNjJiN2I2NGUxN2VmNjlkMTFlYTQ1YTNjYTc0MTUzOWZiZjVhZGU3
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  --- FAIL: TestABodyLineThatIsABodyCountIsRefused (0.00s)
  === RUN   TestACountedBodyWritesALiteralBodyCountLine
  --- PASS: TestACountedBodyWritesALiteralBodyCountLine (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.005s
  === RUN   TestACompiledBodyStartingWithBodyIsCounted
  --- PASS: TestACompiledBodyStartingWithBodyIsCounted (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest	0.006s
  FAIL
  ```
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:29759
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:29249
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:29460
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:29254
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:29208
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:29315
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · ms:31078

## Mutation Log
(empty until execute)
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/plan/plan.go` · the guard is deleted: the refusal test and §133 go red · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · covers:a misplaced body count is refused
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/plan/plan.go` · the guard is deleted: the refusal test and §133 go red · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · covers:the binary refuses it
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/plan/plan.go` · the anchor is dropped: x body=1 is refused and the test goes red · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · covers:a misplaced body count is refused
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/plan/plan.go` · the first-line condition is dropped: a deeper body= line is refused and the test goes red · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · covers:a misplaced body count is refused
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/plan/plan.go` · a counted body falls through to the uncounted branch: the counted test goes red · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · covers:a counted body writes the line
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · emit stops counting a body= first line: the compiled test goes red · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · covers:a counted body writes the line
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-070 does not own changes: the go/no-go guard must go red · acceptance-sha256:2fd4c6783535872fb7cd0d9cd72d76e904703b4767fedc976d9904f0d04eb10e · covers:no engine file changes but plan

## Invariants

- A plan with no such line parses exactly as before.
- The overcount guard (`plan.go:239`) and the stray-text guard (ADR-060) are unchanged.

## Risks

- A file whose first line is `body=<n>`, edited by an uncounted hunk, is refused; the message names the escape.

## Out of Scope

- Other header options on a line of their own (permanent: boundary: not observed; ADR-070 Out of Scope)

## Stop Condition

Stop if the change needs a plan-grammar change or an exit-code change.
