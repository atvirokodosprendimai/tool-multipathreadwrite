# Task ADR-214-T1: a write keeps a file's mark at byte zero

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the assembly rule and the landed-file receipt in `internal/apply`; the `from=` rule; the two refusal sentences; the read's `-- note:` line; the tests below
**Consumes:** `text{lines, eol, final}`, the assembly loop, `fromcopy.go`, the receipt's counts, previews and pad, `internal/read`'s header printer
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a write keeps the mark at byte zero and never doubles it`, `the rule is judged on the assembled output by provenance`, `a mark does not stay on a line that moved`, `the receipt, preview and pad describe the landed file`, `the ledger follows the same rules`, `a from= copy of line 1 carries no mark`, `a refusal teaches the retry`, `a read names the mark by one rule`, `transport marks stay transport`, `an unmarked file gains no mark`, `the packages this record does not own are untouched`, `every named test passed and none skipped`

## Goal

ADR-214 Decisions 1 to 10 and 12's recipe. Every row asserts exact bytes and the raw sha before (the engine's `sha_before` equals the raw sha a read recorded). `M` is `EF BB BF`; rows are fixtures with the plan's valid licences; "ok" means exactly these bytes.

- **`TestAWriteKeepsTheFilesLeadingMarkAtByteZero`** (`internal/apply`), the table, on `Mone\ntwo\n` unless stated:
  - replace line 1 with `ONE`, and with `MONE`: `MONE\ntwo\n` both; delete line 1: `Mtwo\n`; `insert-before 1` and `insert-after 0` of `zero`: `Mzero\none\ntwo\n` both (the old line 1 carries no mark); replace 1-2 (anchor `one`) with `A`,`B`: `MA\nB\n`; delete 1-2: `M`; one plan replacing 1 with `ONE` and 2 with `TWO`: `MONE\nTWO\n`; replace 1 with one empty body line: `M\ntwo\n`; an insert of `zero` and a replacement of line 1 at the same position: `Mzero\n…` with the one mark; replace line 2 and `insert-after 1`: line 1's bytes unchanged.
  - `MMone\n`, replace 1 with `X`: `MX\n`; `MMone\ntwo\n`, replace line 2: `MMone\ntwo` unchanged line 1; `Mone\nMtwo\n`, delete 1: `MMtwo\n`.
  - the file `M` alone: `insert-after 0` and `insert-before 1` of `zero`: `Mzero\n` both; delete its line: `M`; replace its line with `x`: `Mx`.
  - rename to `g.txt`: `g.txt` holds `Mone\ntwo\n`, the old path is gone, the sha is the raw one; rename plus replace 1 with `ONE`: `g.txt` holds `MONE\ntwo\n`; `unlink`: the path is absent (not a line edit).
  - create a missing file with body `Mhello`: `Mhello\n`.
  - `--sub` `one`→`ONE` and `.one`→`ONE`: `MONE\ntwo\n` both; `^one` selects no site. `--rewrite`: a raw hit `[3,6)`→`ONE` and `[0,6)`→`ONE`: `MONE\ntwo\n` both; on `MMone\n` a hit `[0,3)`→`Mone` gives `Mone\n` (Decision 2).
  - `SEARCH one` and patch `-one`: refused, bytes unchanged; `SEARCH Mone`/`-Mone` with new side `ONE` or `MONE`: `MONE\ntwo\n`.
  - `Mone\r\ntwo\r\n` and `Mone\rtwo\r`, replace 1 with `ONE`: `MONE\r\ntwo\r\n` and `MONE\rtwo\r`.
  - **Both controls, each by name and each asserted on exact bytes:** an unmarked `one\ntwo\n` with line 1 replaced by `ONE` is exactly `ONE\ntwo\n` (`an_unmarked_file_gains_no_mark`); an unmarked `one\ntwo\n` with line 1 replaced by the body `MONE` begins with `M` as today (`an_unmarked_file_takes_a_marked_body`). The test counts its rows and fails when fewer ran.
- **`TestAWriteReceiptDescribesTheLandedFile`** (`internal/apply`): deleting 1-2 of `Mone\ntwo\n` reports `lines_after` 1 and a `sha_after` equal to `SHA256(M)`, in a dry run and a live write, and on the destination row of delete-all plus rename, while the hunk keeps `-2 +0`; `--grep '.one' --sub ONE --dry-run` previews `MONE`; `insert-before 1` with `--echo-pad 1` shows line 2 as `one`; replacing line 1 of `Mx\n}\n` with `}` raises the closer hint exactly as it does for `x\n}\n`.
- **`TestALedgerAfterAMarkedWriteFollowsTheSameRules`** (`internal/writer`): after a native replace of line 1 the record is whole and shows nothing, a bare `1 replace` is refused and `anchor="ONE"` lands; after a quote-only `SEARCH Mone` → `REPLACE ONE` the record licenses `[1,1]` and a bare `1 replace` lands; `SEARCH Mone` → `REPLACE one` writes the same bytes again and a prior showing of line 1 survives (its sha and spans unchanged); a dry run leaves the ledger as it was.
- **`TestAFromCopyOfALineOneCarriesNoMark`** (`internal/apply`): `from=src.txt:1` of `Mone\n` into the middle of an unmarked `dst`: `one`; to the top of a marked `dst`: `Mone` with one mark; to the top of an unmarked: `one`; `copied_first` and `copied_last` are `one`; a source `MMone\n` to the top of a marked destination lands `Mone`; a source `M` copies one empty line; an unmarked source is copied as before; the source sha gate stays raw.
- **`TestAMarkedFileRefusalTeachesTheMark`** (`internal/apply`, `internal/ingest`): the delete expecting `one`, the SEARCH miss and the patch miss on a file whose line 1 is `Mone` each name the mark and the retry; on `Mone\none\n` a SEARCH `one` still matches line 2 (the guide says so); no refusal on an unmarked file mentions a mark.
- **`TestAReadOfAMarkedFileSaysSoOnce`** (`internal/read`): the note is printed after the header and before the first range, under `-N`, and in `--json` `content`; once for `read f.txt:1 f.txt:1`; for a line 1 cut by `--max-cols` and for a folded display; not for `read f.txt:2`, `--quiet`, `--stat`, `--count`, `--max-lines 0`, a plan maker, a successful `--if-changed`/`--if-served` reference; a guard miss prints it; for the file `M` it prints when line 1 is printed and not for `read f.txt:2`; at `--max-chars 57` the header fits and neither note nor line 1 is delivered, at 193 the note and line 1 are both withheld (never the note alone), the withheld range is named on standard error and the read exits 1 `incomplete:`; the note earns no line count, showing or licence; the header's `NL NB sha` is the raw one.
- **`TestMrwReadAndWriteKeepAMarkLikeTheCLI`** (`internal/mcp`): `mrw_read` carries the note in `content[0]` outside the checkpoint spans and its checkpoints cover only the numbered rows; at the ceiling an open-ended read pages, a finder returns an index and a closed range that cannot fit is refused; `mrw_write` of a line-1 replace keeps the mark and its receipt's `lines_after` agrees with the bytes.
- **`TestTransportMarksStayTransport`** (`cmd/mrw`, the portable suite; the PowerShell pipe is the Windows shell harness's): in ONE fixture a plan document with two leading marks before its first `@@` and a counted raw body line `M@@ x`: the document marks are dropped, the body line is written as content; `read -` of `MMone\n` drops the marks and records no licence; `write --create` of the same bytes keeps both and names them; a CRLF target file keeps its terminators under a native edit.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `text` knows the original began with a mark; the assembly tracks provenance of the first output line and of original line 1; the join and the landed counts, previews and pad |
| `internal/apply/fromcopy.go` | edit | Decision 9 |
| `internal/apply/pathop.go` | edit | the rename row's `lines_after` from the landed bytes |
| `internal/ingest/applypatch.go`, `internal/ingest/searchreplace.go` | edit | the two misses name the mark |
| `internal/read/read.go` | edit | the note, with Decision 8's rule |
| `internal/apply/mark214_test.go`, `internal/writer/mark214_test.go`, `internal/ingest/mark214_test.go`, `internal/read/mark214_test.go`, `internal/mcp/mark214_test.go`, `cmd/mrw/mark214_test.go` | new | the tests |

## Ordered Steps

1. [S1] The tests. RED on `main`, row by row: the rows whose bytes differ on `main` (a line-1 replace without a mark, a delete of line 1, `insert-before 1`, replace 1-2, delete 1-2, the empty body line, `from=`, the refusal sentences, the note, the receipt count) fail; the rows that already behave as expected (a body that brings its mark, a line-2 edit, a rename, the two controls) pass and must keep passing. [proof: acceptance]
2. [S2] First list the assembly's cases (first line from a body, from a promoted original line, from original line 1 in place; untouched lines between hunks and the tail), then write the rule, the landed counts, `from=`, the refusals, the note. Mutants, each killed by a named row: the mark is not put back; it is put back when the body brought its own (`MONE`→`MMONE`); a moved line 1 keeps its mark (`insert-before 1`); a promoted line is taken to supply the mark (`MMtwo`); the delete-all case lands an empty file; `lines_after` counts the unmarked splice (0 for `M`); the preview shows the submitted body; the rule is applied to an unmarked file (the first control); a marked body on an unmarked file loses its mark (the second control); `from=` copies the mark; the note is dropped, is printed for line 2, is printed twice, or is written without its line 1; the note's `^` claim is wrong. The mark-only branch is observable through `lines_after` and the `insert-before 1` row, not through a trailing-slice entry. Then `adr-lint` on every record whose test lock names a file this task edits. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && env go test -count=1 -v ./internal/apply -run '^(TestAWriteKeepsTheFilesLeadingMarkAtByteZero|TestAWriteReceiptDescribesTheLandedFile|TestAFromCopyOfALineOneCarriesNoMark|TestAMarkedFileRefusalTeachesTheMark)$' > "$d/apply" \
  && env go test -count=1 -v ./internal/writer -run '^TestALedgerAfterAMarkedWriteFollowsTheSameRules$' > "$d/writer" \
  && env go test -count=1 -v ./internal/read -run '^TestAReadOfAMarkedFileSaysSoOnce$' > "$d/read" \
  && env go test -count=1 -v ./internal/mcp -run '^TestMrwReadAndWriteKeepAMarkLikeTheCLI$' > "$d/mcp" \
  && env go test -count=1 -v ./cmd/mrw -run '^TestTransportMarksStayTransport$' > "$d/cmd" \
  && for t in TestAWriteKeepsTheFilesLeadingMarkAtByteZero TestAWriteReceiptDescribesTheLandedFile TestAFromCopyOfALineOneCarriesNoMark TestAMarkedFileRefusalTeachesTheMark; do grep -q "^--- PASS: $t (" "$d/apply" || exit 1; done \
  && grep -q '^--- PASS: TestALedgerAfterAMarkedWriteFollowsTheSameRules (' "$d/writer" \
  && grep -q '^--- PASS: TestAReadOfAMarkedFileSaysSoOnce (' "$d/read" \
  && grep -q '^--- PASS: TestMrwReadAndWriteKeepAMarkLikeTheCLI (' "$d/mcp" \
  && grep -q '^--- PASS: TestTransportMarksStayTransport (' "$d/cmd" \
  && grep -q '^    --- PASS: TestAWriteKeepsTheFilesLeadingMarkAtByteZero/an_unmarked_file_gains_no_mark' "$d/apply" \
  && grep -q '^    --- PASS: TestAWriteKeepsTheFilesLeadingMarkAtByteZero/an_unmarked_file_takes_a_marked_body' "$d/apply" \
  && ! grep -q -- '--- SKIP' "$d/apply" "$d/writer" "$d/read" "$d/mcp" "$d/cmd" \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/lines internal/plan internal/seen internal/check internal/state internal/writer internal/mcp ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/lines internal/plan internal/seen internal/check internal/state internal/writer internal/mcp ':(exclude)*_test.go')" \
  && [ -z "$st" ]
```

What each clause holds. The named tests exist, pass and none is skipped; both controls pass by name, so neither can be deleted without the fence noticing. Every assertion is on bytes, receipt fields and exit codes, not on a grep of the source. The packages the record does not own, `internal/mcp`'s non-test code included and now with its untracked files (the status clause no longer omits the directory), have no committed non-test change since the merge base and no uncommitted or untracked one. The row-by-row red of S1 is read from the captured output before S2.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteKeepsTheFilesLeadingMarkAtByteZero` | `internal/apply/mark214_test.go` | every shape of Decisions 1 to 4 and 12 by exact bytes, raw shas, CRLF/CR-only, both controls | none | S1, S2 |
| `TestAWriteReceiptDescribesTheLandedFile` | `internal/apply/mark214_test.go` | `lines_after`, `sha_after`, preview, pad, the closer hint, dry run and live | none | S1, S2 |
| `TestAFromCopyOfALineOneCarriesNoMark` | `internal/apply/mark214_test.go` | Decision 9, `copied_first`/`last`, the double mark | none | S1, S2 |
| `TestAMarkedFileRefusalTeachesTheMark` | `internal/apply/mark214_test.go` | Decision 7 | none | S1, S2 |
| `TestALedgerAfterAMarkedWriteFollowsTheSameRules` | `internal/writer/mark214_test.go` | Decision 6 | none | S1, S2 |
| `TestAReadOfAMarkedFileSaysSoOnce` | `internal/read/mark214_test.go` | Decision 8: position, eligibility, once, caps, the withheld pair, no licence | none | S1, S2 |
| `TestMrwReadAndWriteKeepAMarkLikeTheCLI` | `internal/mcp/mark214_test.go` | MCP parity, checkpoints, the ceiling cases | none | S1, S2 |
| `TestTransportMarksStayTransport` | `cmd/mrw/mark214_test.go` | Decision 10 | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the eight tests |
| 2 — something selects it | every write to a file that begins with a mark |
| 3 — the caller can discover it | the read's note, the refusals' sentences; T2's words |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #686, one Windows probe |

## Mutation Log
- 2026-11-27 · 6cbf39bf · mutant killed · exit 1 · `internal/apply/apply.go` · the mark is not put back · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · the mark is put back when the body brought its own (MONE -> MMONE) · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · a moved line 1 keeps its mark (insert-before 1) · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · a promoted line is taken to supply the mark (MMtwo) · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · the delete-all case lands an empty file · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · lines_after counts the unmarked splice (0 for M) · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · the preview shows the submitted body · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · the rule is applied to an unmarked file (the first control) · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · a marked body on an unmarked file loses its mark (the second control) · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/fromcopy.go` · from= copies the mark · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/read/read.go` · the note is dropped · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/read/read.go` · the note is printed for line 2 · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/read/read.go` · the note is printed twice · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/read/read.go` · the note is written without its line 1 · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/read/read.go` · the note's ^ claim is wrong · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/apply/apply.go` · the delete refusal does not teach the mark · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b
- 2026-11-27 · 6cbf39bf* · mutant killed · exit 1 · `internal/ingest/searchreplace.go` · the SEARCH and old-side misses do not teach the mark · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b

## Verification Log
- 2026-11-26 · b787541e · exit 1 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:1699 · test-lock-sha256:065704c4b78c31bd1e0fd789c3334c188f7a3433053554d54e1761cf45a21887 · test-lock-file:docs/adr/locks/065704c4b78c31bd1e0fd789c3334c188f7a3433053554d54e1761cf45a21887.lock
  ```
  ```
- 2026-11-27 · f80d6ec5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:26 · test-lock-sha256:564857b11168af50e85e5aa2f67885ea9b4198bb03831c6a4dc93a34244e30f6 · test-lock-file:docs/adr/locks/564857b11168af50e85e5aa2f67885ea9b4198bb03831c6a4dc93a34244e30f6.lock · test-lock-kind:replace
- 2026-11-27 · f80d6ec5* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:4911
- 2026-11-27 · 6cbf39bf · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7528
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7113
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7050
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7567
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:6810
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:6975
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7092
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:6917
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7185
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7174
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:6810
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7088
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7381
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7189
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7882
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7127
- 2026-11-27 · 6cbf39bf* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:bb0a56ac802cfc828ab7250f27545f5026221103fe742c68557365797227e93b · ms:7178

## Invariants

- No receipt key is added; a file's bytes read, shas, spans, ledger keys and line numbers are unchanged; an unmarked file is written exactly as before.

## Risks

- A shape the table lacks: S2 lists the assembly's cases before the rule is written.
- Editing a test file that another record locks moves its lock; S2 lints and relocks what it names.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI; the Windows shard passes the portable tests; every gate in `AGENTS.md` passes; the engine diff names ADR-214 as owner, amends ADR-167 in its text, and has a `go-reviewer` pass; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- The words and the executed probe (deferred: T2)
