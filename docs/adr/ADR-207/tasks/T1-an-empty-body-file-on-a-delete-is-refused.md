# Task ADR-207-T1: an empty body file on a delete is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the refusal in `LoadBodyFiles`; contract §358; the sentence in `AGENTS.md` and `mrw write --help`
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
| `scripts/contract.sh` | edit | §358 |
| `AGENTS.md` | edit | the sentence at 427-428 |

## Ordered Steps

1. [S1] The three tests; RED on `delete_zero_bytes`, the CLI test and the MCP test; the other subtests hold today and stay. [proof: acceptance]
2. [S2] The refusal. Mutants, each killed: the zero-byte check removed (`delete_zero_bytes`); the check applied to every op (`create_zero_bytes`); the refusal returned as a hunk failure after loading instead of a load error (the CLI test sees exit 1, not 2). A check on `len(Body) == 0` after splitting is not a mutant: `lines.Split` gives zero lines only for zero bytes. [proof: mutation]
3. [S3] Contract §358 drives `$MRW` on `f.txt` = `a b c` and `g.txt` = `x`, both read whole, with `empty358.txt` (zero bytes), `wrong358.txt` (`zzz`) and `b358.txt` (`b`): the sibling plan with `delete body=@empty358.txt` exits 2, the refusal names the plan line and the file, both files byte-identical; `delete body=@wrong358.txt` exits 1 with `f.txt` unchanged; `delete body=@b358.txt` exits 0 and removes line 2; on fresh copies `delete body=0` and a bodyless delete exit 0; `insert-after body=@empty358.txt` and `insert-before body=@empty358.txt` exit 1. The sentences. [proof: acceptance]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./internal/plan ./cmd/mrw ./internal/mcp -run '^(TestAnEmptyBodyFileOnADeleteIsRefused|TestTheCLIRefusesAnEmptyBodyFileOnADelete|TestMrwWriteRefusesAnEmptyBodyFileOnADelete)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 3 ] \
  && [ "$(grep -c '^    --- PASS: TestAnEmptyBodyFileOnADeleteIsRefused/' "$out")" -eq 7 ] \
  && ! grep -q -- '--- SKIP' "$out" \
  && grep -q '^# 358\. ' scripts/contract.sh \
  && grep -q 'delete body=@empty358.txt' scripts/contract.sh \
  && grep -q 'delete body=@wrong358.txt' scripts/contract.sh \
  && grep -q "a delete's body=@ is the lines it expects to remove" internal/plan/plan.go \
  && grep -q 'an empty file is body=0 for a create, is refused on a delete' <(tr -s '\n' ' ' < AGENTS.md) \
  && grep -q 'an empty file is body=0 for a create, is refused on a delete' <(tr -s '\n' ' ' < cmd/mrw/main.go) \
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
| 1 — exists | the tests and §358 |
| 2 — something selects it | every plan with `body=@`, through the three callers of `LoadBodyFiles` |
| 3 — the caller can discover it | the refusal's own sentence; `AGENTS.md`; `mrw write --help` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is the probe in the record |

## Mutation Log

## Verification Log

## Invariants

- `body=0` and a delete with no body delete without an expectation, as before.
- No receipt key is added, removed or renamed.

## Risks

- The fence proves §358's text exists, not that it ran: §358 is proved by `scripts/contract.sh`, a gate in `AGENTS.md` §Build and check, run before the commit (`lifecycle.md` §6). Running the whole contract inside the fence would run it once per mutant.

## Stop Condition

The fence exits 0, `./scripts/contract.sh` exits 0 with §358's rows printed, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes, and the record is Accepted by Zy.

## Out of Scope

- Moving the empty-body refusal of `replace` and the inserts to load time (permanent: boundary: the record's Decision 3; both already refuse with nothing written)
