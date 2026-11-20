# Task ADR-208-T1: the line after is shown on a written file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `needAfter` asking for a showing in the current version on a whole written-shaped record; the `apply.Options` field for a same-call `--rewrite`; the refusal's remedy; contract §359; Decision 6's sentence in `AGENTS.md`, `README.md` and the full instructions
**Consumes:** `seen.Observation.ServedLine`, `staleRemedy`, `apply.Options`
**Data dependency:** none — hermetic (§359 uses §264's recorded ast-grep answer, not a real ast-grep)
**Proof map:** v1
**Rests-on:** `a written file's line after must be shown`, `a showing in this version meets it`, `a showing of another version does not, forced or not`, `the last line needs none`, `a read-shaped record is judged as before`, `a same-call rewrite is not judged`, `a later-call rewrite plan is judged`, `foreign formats are judged`, `a kept tail still lands`, `force does not waive it`, `every subtest ran and none skipped`, `the CLI refuses it and lands after the read`, `mrw_write refuses it and lands after the read and ack`, `the contract row exists with its plans`, `the new sentence is in the three documents`, `the engine packages are untouched, tracked and untracked`

## Goal

ADR-208 Decisions 1 to 6.

**`TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown`** (`internal/apply`), one subtest per row, each an `apply.Apply` call with a ledger, on `f.go` = `func A() {` / `\tif x {` / `\t}` / `}` and a plan replacing 1-3 with `func A() {` / `\ty()` / `}`, `anchor="func A"`, unless the row says otherwise:

| Subtest | Record of `f.go` | Plan | Outcome |
|---------|------------------|------|---------|
| `written_nothing_shown` | written, whole, current sha, nothing shown | numbered | failed, `needs a served line after 3`, the reason ending with the `--max-cols 0` read of line 4; bytes unchanged |
| `written_line_after_shown` | the same, line 4 shown | numbered | ok |
| `written_ends_on_last_line` | the same, nothing shown | replacing 1-4 | ok |
| `read_lines_1_to_4` | read-shaped, lines 1-4 | numbered | ok (as today) |
| `read_lines_1_to_3` | read-shaped, lines 1-3 | numbered | failed (as today) |
| `same_call_rewrite` | written, whole, nothing shown | marked as a same-call `--rewrite` | ok |
| `later_call_rewrite_plan` | written, whole, nothing shown; a `read --rewrite` licence of 1-4 merged in | the same hunk, not marked | failed |
| `search_replace_no_kept_tail` | written, whole, nothing shown | the compiled `search_replace` hunk, REPLACE not ending with SEARCH's last line | failed |
| `search_replace_kept_tail` | written, whole, nothing shown | a 4-line SEARCH whose REPLACE ends with its last line, compiled to a 3-line replace with the kept line quoted | ok |
| `apply_patch` | written, whole, nothing shown | the compiled `apply_patch` hunk | failed |
| `forced_stale_showing` | written, whole, an older sha, line 4 shown in it; the file changed since | numbered, `Force` | failed |
| `forced_fresh_showing` | written, whole, current sha, line 4 shown | numbered, `Force` | ok |
| `forced_nothing_shown` | written, whole, current sha, nothing shown | numbered, `Force` | failed |
| `partial_written_licence` | written-shaped, partial licence 1-4 (a `read --rewrite` plan's on a read file) | numbered | ok (as today) |

**`TestTheCLIRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown`** (`cmd/mrw`, the built binary, runs on Windows CI): `mrw write --create f.go`, the plan exits 1 with the failed hunk in `--json`; `mrw read f.go:4`, the same plan exits 0; a whole `--quiet` read of a fresh copy, then the plan, exits 1.

**`TestMrwWriteRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown`** (`internal/mcp`): the same over `mrw_write`: `isError` with the failed hunk; after `mrw_read` of line 4 and its `ack`, the same plan lands.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `needAfter`; the plan-source field in `Options`; the remedy in the reason |
| `cmd/mrw/main.go`, `internal/mcp/tools.go` | edit | set the field for a same-call `--rewrite` |
| `internal/apply/after208_test.go` | new | the first table |
| `cmd/mrw/after208_test.go` | new | the CLI test |
| `internal/mcp/after208_test.go` | new | the MCP test |
| `internal/guide/guide.go` | edit | Decision 6's sentence |
| `scripts/contract.sh` | edit | §359 |
| `AGENTS.md`, `README.md` | edit | Decision 6's sentence |

## Ordered Steps

1. [S1] The three tests; RED on `written_nothing_shown`, `later_call_rewrite_plan`, `search_replace_no_kept_tail`, `apply_patch`, `forced_stale_showing`, `forced_nothing_shown`, the CLI test and the MCP test; the other subtests hold today and stay. [proof: acceptance]
2. [S2] The rule. Mutants, each killed: `Covers` kept for written records (`written_nothing_shown`); the sha comparison dropped (`forced_stale_showing`); the plan-source field ignored (`same_call_rewrite`); every `Computed` plan exempt (`search_replace_no_kept_tail`, `apply_patch`); `Force` exempt (`forced_nothing_shown`); the last-line skip dropped (`written_ends_on_last_line`); the field never set by `cmd/mrw` (§359's same-call row, and the CLI test where `ast-grep` is faked). [proof: mutation]
3. [S3] Contract §359 drives `$MRW`: the rows in the record's Wiring table, each paired: refused then landing after the read; a kept tail landing beside a block without one refused; §264's fake ast-grep for the same-call `--rewrite` landing and the later-call plan refused, then landing after `mrw read f359.go:N+1`. The sentences. [proof: acceptance]

## Acceptance

```bash
out="$(mktemp)" \
  && go test -count=1 -v ./internal/apply ./cmd/mrw ./internal/mcp -run '^(TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown|TestTheCLIRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown|TestMrwWriteRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown)$' > "$out" \
  && [ "$(grep -c '^--- PASS: ' "$out")" -eq 3 ] \
  && [ "$(grep -c '^    --- PASS: TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown/' "$out")" -eq 14 ] \
  && ! grep -q -- '--- SKIP' "$out" \
  && grep -q '^# 359\. ' scripts/contract.sh \
  && grep -q 'f359.go' scripts/contract.sh \
  && grep -q 'stream264.jsonl' <(sed -n '/^# 359\. /,/^# [0-9][0-9]*\. /p' scripts/contract.sh) \
  && grep -q 'needs the line after it shown since the write' <(tr -s '\n ' '  ' < AGENTS.md) \
  && grep -q 'needs the line after it shown since the write' <(tr -s '\n ' '  ' < README.md) \
  && grep -q 'needs the line after it shown since the write' internal/guide/guide.go \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown` | `internal/apply/after208_test.go` | the first table, by the verdict, the reason and the bytes on disk | none | S1, S2 |
| `TestTheCLIRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown` | `cmd/mrw/after208_test.go` | exit 1 and the `--json` failed hunk; landing after the read; the whole `--quiet` case | none | S1, S2 |
| `TestMrwWriteRefusesAMultiLineReplaceOnAWrittenFileUntilTheLineAfterIsShown` | `internal/mcp/after208_test.go` | `isError` with the failed hunk; landing after `mrw_read` and `ack` | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §359 |
| 2 — something selects it | every multi-line replace through apply's validation, on both surfaces |
| 3 — the caller can discover it | the refusal names the read; `AGENTS.md`, `README.md`, `mrw instructions` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence for building it is the probe in the record |

## Mutation Log

## Verification Log

## Invariants

- A record that is not whole and written-shaped is judged exactly as before.
- No receipt key is added, removed or renamed; the ledger format is unchanged.

## Risks

- The fence proves §359's text exists and uses §264's fixture, not that it ran: §359 is proved by `scripts/contract.sh`, a gate run before the commit (`lifecycle.md` §6). Running the whole contract inside the fence would run it once per mutant. `contract.sh` does not run on Windows; the CLI and MCP tests carry the behaviour there.
- AGENTS.md and README.md are hard-wrapped, so the fence joins their lines before matching.

## Stop Condition

The fence exits 0, `./scripts/contract.sh` exits 0 with §359's rows printed, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes on Linux and Windows CI, and the record is Accepted by Zy.

## Out of Scope

- A closer above the body (permanent: boundary: the record's Out of Scope; ADR-052 chose the line after)
