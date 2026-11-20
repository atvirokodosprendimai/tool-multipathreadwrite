# Task ADR-207-T1: an empty body file on a delete is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the refusal in `LoadBodyFiles`; contract §371; the sentence in `AGENTS.md` and `mrw write --help`
**Consumes:** `plan.Hunk.Op`, `plan.LoadBodyFiles`, `lines.Split`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a delete with a zero-byte body file is refused`, `a body file of one blank line is an expectation`, `a delete with a non-empty body file still loads`, `an empty create from a file is still an empty file`, `an empty body file on a replace or an insert still reaches apply`, `every subtest ran and none skipped`, `the CLI refuses it with nothing written`, `mrw_write refuses it with isError`, `the contract row exists with its refused and failing plans`, `the refusal sentence is in the code`, `the new sentence is in AGENTS.md and the help`, `the engine packages are untouched, tracked and untracked`

## Goal

ADR-207 Decisions 1 to 4.

**`TestAnEmptyBodyFileOnADeleteIsRefused`** (`internal/plan`), one subtest per row, `LoadBodyFiles` over a root holding `empty.txt` (zero bytes), `crlf.txt` (`\r\n` only) and `b.txt` (`b`):

| Subtest | Hunk | Outcome |
|---------|------|---------|
| `delete_zero_bytes` | `@@ f.txt 2 delete body=@empty.txt` | an error naming the plan line, `body=@empty.txt` and `0 bytes` |
| `delete_one_blank_line` | `@@ f.txt 2 delete body=@crlf.txt` | no error; the body is `[""]` |
| `delete_non_empty` | `@@ f.txt 2 delete body=@b.txt` | no error; the body is `["b"]` |
| `create_zero_bytes` | `@@ new.txt 0 create body=@empty.txt` | no error; the body is nil and `CountedBody` is true |
| `replace_zero_bytes` | `@@ f.txt 1 replace anchor="a" body=@empty.txt` | no error; the body is nil (apply refuses it, as today) |
| `insert_before_zero_bytes` | `@@ f.txt 1 insert-before body=@empty.txt` | no error; the body is nil (apply refuses it, as today) |
| `delete_body_zero` | `@@ f.txt 2 delete body=0` | no error |

**`TestTheCLIRefusesAnEmptyBodyFileOnADelete`** (`cmd/mrw`): the built binary, `f.txt` = `a b c` and `g.txt` = `x` read whole; the plan `@@ g.txt 1 replace` / `y` then `@@ f.txt 2 delete body=@empty.txt` exits 2, both files byte-identical, and `--json` gives one document whose top-level `error` names plan line 3 and `body=@empty.txt`.

**`TestMrwWriteRefusesAnEmptyBodyFileOnADelete`** (`internal/mcp`): the same plan through `mrw_write` answers `isError`, names the file, and leaves both files byte-identical.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | the refusal in `LoadBodyFiles` |
| `internal/plan/emptybody207_test.go` | new | the first table |
| `cmd/mrw/emptybody207_test.go` | new | the CLI test |
| `internal/mcp/emptybody207_test.go` | new | the MCP test |
| `cmd/mrw/main.go` | edit | the help sentence at 1758-1759 |
| `scripts/contract.sh` | edit | §371 |
| `AGENTS.md` | edit | the sentence at 427-428 |

## Ordered Steps

1. [S1] The three tests; RED on `delete_zero_bytes`, the CLI test and the MCP test; the other subtests hold today and stay. [proof: acceptance]
2. [S2] The refusal. Mutants, each killed: the zero-byte check removed (`delete_zero_bytes`); the check applied to every op (`create_zero_bytes`); the refusal returned as a hunk failure after loading instead of a load error (the CLI test sees exit 1, not 2). A check on `len(Body) == 0` after splitting is not a mutant: `lines.Split` gives zero lines only for zero bytes. [proof: mutation]
3. [S3] Contract §371 drives `$MRW` on `f.txt` = `a b c` and `g.txt` = `x`, both read whole, with `empty371.txt` (zero bytes), `wrong371.txt` (`zzz`) and `b371.txt` (`b`): the sibling plan with `delete body=@empty371.txt` exits 2, the refusal names the plan line and the file, both files byte-identical; `delete body=@wrong371.txt` exits 1 with `f.txt` unchanged; `delete body=@b371.txt` exits 0 and removes line 2; on fresh copies `delete body=0` and a bodyless delete exit 0; `insert-after body=@empty371.txt` and `insert-before body=@empty371.txt` exit 1. The sentences. [proof: acceptance]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./internal/plan ./cmd/mrw ./internal/mcp -run '^(TestAnEmptyBodyFileOnADeleteIsRefused|TestTheCLIRefusesAnEmptyBodyFileOnADelete|TestMrwWriteRefusesAnEmptyBodyFileOnADelete)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 3 ] \
  && [ "$(grep -c '^    --- PASS: TestAnEmptyBodyFileOnADeleteIsRefused/' "$out")" -eq 7 ] \
  && ! grep -q -- '--- SKIP' "$out" \
  && grep -q '^# 371\. ' scripts/contract.sh \
  && grep -q 'delete body=@empty371.txt' scripts/contract.sh \
  && grep -q 'delete body=@wrong371.txt' scripts/contract.sh \
  && grep -q "a delete's body=@ is the lines it expects to remove" internal/plan/plan.go \
  && grep -q 'an empty file is body=0 for a create, is refused on a delete, and is an empty body, refused, on a replace or an insert' <(tr -s '\n ' '  ' < AGENTS.md) \
  && grep -q 'an empty file is body=0 for a create, is refused on a delete, and is an empty body, refused, on a replace or an insert' <(tr -s '\n ' '  ' < cmd/mrw/main.go) \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/read internal/apply internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnEmptyBodyFileOnADeleteIsRefused` | `internal/plan/emptybody207_test.go` | the first table, by the error and the loaded body | none | S1, S2 |
| `TestTheCLIRefusesAnEmptyBodyFileOnADelete` | `cmd/mrw/emptybody207_test.go` | exit 2, the JSON `error`, both files byte-identical | none | S1, S2 |
| `TestMrwWriteRefusesAnEmptyBodyFileOnADelete` | `internal/mcp/emptybody207_test.go` | `isError`, the sentence, both files byte-identical | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §371 |
| 2 — something selects it | every plan with `body=@`, through the three callers of `LoadBodyFiles` |
| 3 — the caller can discover it | the refusal's own sentence; `AGENTS.md`; `mrw write --help` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is the probe in the record |

## Mutation Log
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/plan.go` · the zero-byte check removed: delete_zero_bytes, the CLI test and the MCP test go red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:a delete with a zero-byte body file is refused
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/plan.go` · the check applied to every op: create_zero_bytes goes red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:an empty create from a file is still an empty file
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/plan.go` · the refusal made a hunk failure after loading, exit 1 not 2: the CLI test goes red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:the CLI refuses it with nothing written
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/plan.go` · a body file of one blank line treated as empty: delete_one_blank_line goes red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:a body file of one blank line is an expectation
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/plan.go` · every delete body file refused, empty or not: delete_non_empty goes red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:a delete with a non-empty body file still loads
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/plan.go` · the refusal reaching a replace and an insert: replace_zero_bytes and insert_before_zero_bytes go red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:an empty body file on a replace or an insert still reaches apply
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/emptybody207_test.go` · a subtest skipped: the fence refuses a --- SKIP line · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:every subtest ran and none skipped
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write ignoring the load error: the MCP test goes red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:mrw_write refuses it with isError
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `scripts/contract.sh` · the failing plan renamed in the row: the fence's grep for it fails · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:the contract row exists with its refused and failing plans
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/plan/plan.go` · the refusal sentence reworded: the test and the fence grep go red · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:the refusal sentence is in the code
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `AGENTS.md` · the AGENTS.md sentence reworded: the fence grep fails · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:the new sentence is in AGENTS.md and the help
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `cmd/mrw/main.go` · the help sentence reworded: the fence grep fails · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:the new sentence is in AGENTS.md and the help
- 2026-11-20 · 25093571* · mutant killed · exit 1 · `internal/seen/seen.go` · an engine package changed: the fence git diff against the merge base fails · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · covers:the engine packages are untouched, tracked and untracked
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/plan/plan.go` · the zero-byte check removed: delete_zero_bytes, the CLI test and the MCP test go red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:a delete with a zero-byte body file is refused
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/plan/plan.go` · the check applied to every op: create_zero_bytes goes red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:an empty create from a file is still an empty file
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refusal made a hunk failure after loading, exit 1 not 2: the CLI test goes red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:the CLI refuses it with nothing written
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/plan/plan.go` · a body file of one blank line treated as empty: delete_one_blank_line goes red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:a body file of one blank line is an expectation
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/plan/plan.go` · every delete body file refused, empty or not: delete_non_empty goes red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:a delete with a non-empty body file still loads
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/plan/plan.go` · the refusal reaching a replace and an insert: replace_zero_bytes and insert_before_zero_bytes go red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:an empty body file on a replace or an insert still reaches apply
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/plan/emptybody207_test.go` · a subtest skipped: the fence refuses a --- SKIP line · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:every subtest ran and none skipped
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write ignoring the load error: the MCP test goes red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:mrw_write refuses it with isError
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `scripts/contract.sh` · the failing plan renamed in the row: the fence grep for it fails · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:the contract row exists with its refused and failing plans
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/plan/plan.go` · the refusal sentence reworded: the test and the fence grep go red · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:the refusal sentence is in the code
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `AGENTS.md` · the AGENTS.md sentence reworded: the fence grep fails · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:the new sentence is in AGENTS.md and the help
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `cmd/mrw/main.go` · the help sentence reworded: the fence grep fails · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:the new sentence is in AGENTS.md and the help
- 2026-11-20 · e3ade699* · mutant killed · exit 1 · `internal/seen/seen.go` · an engine package changed: the fence git diff against the merge base fails · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · covers:the engine packages are untouched, tracked and untracked

## Verification Log
- 2026-11-20 · 25093571* · exit 1 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:612 · test-lock-sha256:6eede568539b212f3a16478afe62158d7911131dbd38718782302aceed7ed04d · test-lock-file:docs/adr/locks/6eede568539b212f3a16478afe62158d7911131dbd38718782302aceed7ed04d.lock
  ```
  ```
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:8133
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:4367
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:2300
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:3132
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:3819
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:3071
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:3575
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:3402
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:2562
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:2268
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:2235
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:2946
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:2495
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:2508
- 2026-11-20 · 25093571* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:666
- 2026-11-20 · 3c263198* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:34ae9555de29999236ce112ec768665d5f6f20392a2b25afcd190052401c455f · ms:0 · test-lock-sha256:241c07dc05a51afd0e2af4739e98fbc8e139279f2c9cc49303432299091ab5e3 · test-lock-file:docs/adr/locks/241c07dc05a51afd0e2af4739e98fbc8e139279f2c9cc49303432299091ab5e3.lock · test-lock-kind:replace
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:1729
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2328
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2560
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2664
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:3648
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:4797
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2722
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2548
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2652
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2308
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2871
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2209
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2388
- 2026-11-20 · e3ade699* · exit 0 · `out="$(mktemp)" \ …` · acceptance-sha256:e64addc4d36dafa0fe347f15311e1cbee22a7fc2294a621c6727994403a118ab · ms:2791

## Invariants

- `body=0` and a delete with no body delete without an expectation, as before.
- No receipt key is added, removed or renamed.

## Risks

- The fence proves §371's text exists, not that it ran: §371 is proved by `scripts/contract.sh`, a gate in `AGENTS.md` §Build and check, run before the commit (`lifecycle.md` §6). Running the whole contract inside the fence would run it once per mutant.

## Stop Condition

The fence exits 0, `./scripts/contract.sh` exits 0 with §371's rows printed, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes, and the record is Accepted by Zy.

## Out of Scope

- Moving the empty-body refusal of `replace` and the inserts to load time (permanent: boundary: the record's Decision 3; both already refuse with nothing written)
