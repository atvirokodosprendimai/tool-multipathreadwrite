# ADR-214: A write keeps a file's leading byte order mark at byte zero

**Status:** Accepted
**Accepted by:** Zy, 2026-11-21: *"DO IT"*, to seven decisions put to them that day, of which this was the second, worded: "byte order marks. A write keeps the mark at byte zero; the cost is that a line edit can no longer remove one." Accepted as option B, after its two Codex xhigh challenges.
**Date:** 2026-11-21
**Amended:** 2026-11-27, on the owner's word (Zy, choosing "amend the wording" over skipping byte-identical writes, after T1's red run found the engine rewrites the same bytes for every file, marked or not): Decision 6's `SEARCH Mone` → `REPLACE one` writes the same bytes again, and a prior showing survives under the same-sha merge; the record's `Written` flag is the write's, as for any file. Skipping a byte-identical write would need a receipt state no receipt has and is not part of this record.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-005, ADR-055, ADR-065, ADR-073, ADR-111, ADR-117, ADR-119, ADR-142, ADR-165, ADR-167, ADR-168, ADR-203, ADR-208, issue #686, #685
**Supersedes:** in part: ADR-167 Decision 3's "a copy is the lines as the caller was served them" for one case, a `from=` source range that starts at line 1 of a marked file (Decision 9); nothing else of ADR-167.
**Governs:** `internal/apply/apply.go`, `internal/apply/fromcopy.go`, `internal/apply/pathop.go`, `internal/read/read.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/apply/mark214_test.go::TestAWriteKeepsTheFilesLeadingMarkAtByteZero`
**Served-path change:** a line edit that would leave a file without the UTF-8 byte order mark it began with writes the file with the mark at byte zero; a mark an edit would leave inside the file, on an original line that moved, is taken off that line; the receipt, the preview and the echo pad describe the file that lands. A read that prints line 1 of a marked file adds one `-- note:` line after the header. A `from=` copy of line 1 of a marked file carries the line without its mark. An exact-match refusal on a marked line names the mark. Line 1 is still served with its mark, `^one` still misses it, and bytes, shas, spans, ledger keys and ast-grep offsets are what they were. No receipt key is added (ADR-111).

## Context

**What #686 found.** A file that begins with a UTF-8 byte order mark is served with the mark as part of line 1, an invisible character. A plan `@@ marked.txt 1 replace` with body `ONE` writes the file without the mark: exit 0, no advisory. The issue names three options and chooses none; it is a public contract, so it needs a record and the owner's choice.

**Measured on `main` at `5587200f`** (a `mrw` built from this tree, scratch directories, fixture `M` + `one\ntwo\nthree\n`, every file read before it was written, the fixtures removed; `M` is `EF BB BF`).

| Operation on the marked file | Result | The receipt says |
|---|---|---|
| `mrw read` | line 1 is `M one` (the mark is invisible in a terminal); header `3L 17B` counts the three bytes | nothing about a mark |
| `read --grep '^one'`; address `/^one/`; `--grep '^one' --sub` | no match; `--grep 'one'`, `anchor="one"` and `^.one` match | n/a |
| replace line 1, body `ONE` (anchor `one`, `M one`, or `/one/`) | the file begins `4f 4e 45`: the mark is lost | `ok`, 0 advisories |
| replace line 1, body `M ONE` | the mark is kept (the body brought it) | `ok` |
| replace lines 1-2, body `A`,`B` | lost | `ok` |
| delete line 1 | lost with the line | `ok ... from "﻿one"` |
| delete line 1, expecting the visible `one` | refused at exit 1: `plan says "one", file has "﻿one"` (the six ASCII characters of `﻿` fail again) | FAIL |
| insert-before 1; insert-after 0 | the new line is first; the mark is now inside the old line, now line 2 | `ok` |
| insert-after 1; replace line 2 | kept | `ok` |
| `--grep 'one' --sub ONE` | kept: `ReplaceAllString` runs on the whole line, so the result is `M ONE` | `ok` |
| `--grep '.one' --sub ONE` | lost: the dot consumes the mark | `ok` |
| `--ast-grep P --rewrite R` on line 1 | kept: the rewrite splices the raw bytes after the match | `ok` |
| `search_replace` SEARCH `one` | `SEARCH matched no lines`, exit 2, no mention of a mark; on `Mone\none\n` the same old side matches line 2; SEARCH `M one`, REPLACE `ONE` loses the mark | refused; `ok` |
| `apply_patch` `-one` | `old side matched no lines`, exit 2; `-M one`, `+ONE` loses the mark | refused; `ok` |
| `from=src.txt:1` into another file | the mark travels into the middle of the destination | `ok` |
| plan `create` with body line 1 `M hello`; `--create` from stdin, one or two marks; `body=@file` with a leading mark | the mark is the body's and is kept (ADR-142 names a double one) | `ok` |
| a UTF-16 or UTF-32 file | edits refused (ADR-073); a named read serves it with a note; a walk skips it | FAIL |

So the mark behaves as line-1 text: two shapes lose it, one leaves it mid-file, one carries it into another file, every start-anchored pattern misses line 1, and an exact-match caller who types the visible text of line 1 meets a refusal that does not teach the retry.

**How often it matters.** On 2026-11-21 a count over every git-tracked text file (no NUL in the first 8 KiB) found 0 of 5,763 that begin with a mark in this repository and 0 of 1,226 in three other local checkouts. That says nothing of .NET or Windows trees, where editors add marks (`.cs`, `.csproj`, `.ps1`); no rate is claimed. Telemetry is refused by ADR-009; the evidence for building is #686, one probe.

**Two challenges, and what they showed.** The first draft decided the mark is the file's metadata, stripped at the split and given back at the join (design A). The Codex xhigh challenge returned AMEND with thirteen points that fall on the design: every consumer of line 1 needs a rule. The recommendation became B, a write-time rule, and a second challenge kept B and found that its contract had to be stated on the assembled output (with where the first output line came from), that the receipt, the preview and the pad must describe the file that lands, that the read note needs an eligibility and budget rule, and that `from=` needs an explicit amendment. This record is the result.

**Precedents.** ADR-065 took the `\r` of a CRLF file out of the line text and gave it back at the join; that works for `\r` because every consumer already agrees a line has no terminator, and a mark has no such agreement (ast-grep, `--json-path`, `--outline`, the hook and the ledger see raw bytes). `mrw skill --dir` keeps a SKILL.md's mark on a replace (`cmd/mrw/skill168.go:164`). ADR-073 prints a `-- note:` for a file mrw cannot edit; ADR-055's `balance` and ADR-119's `closer` rows are advisories under an `ok` hunk, counted in `advisories` and `hints`, so a new kind would not reuse either count.

## Existing Primitives Audit

- **`text{lines, eol, final}`** and `join()` (`internal/apply/apply.go:2648-2663`): the lines stay raw, so the join is where an output is judged; `final` and the receipt's line count come from the splitter (`lines.Split`), which reads `M` alone as one unterminated line.
- **The assembly loop** (`apply.go:2401-2515`) sorts hunks in original coordinates, puts inserts ahead of a consuming hunk at the same position, and appends untouched lines both between hunks and after the last one; the first output line can come from a body, from an original line promoted by a deletion, or from original line 1 in place.
- **The receipt's `lines_after` and `sha_after`** (`apply.go:911`, `pathop.go:540`) are computed from the staged bytes; previews (`Before`/`After`) and the echo pad are captured from the submitted body (`apply.go:2462`).
- **The ledger** (`internal/writer/writer.go:127-175`, `internal/seen/seen.go:843`): a native write records a whole-file licence with nothing shown; a quote-only write records the lines written; a same-sha merge keeps a prior showing.
- **`fromcopy.go:95`** builds a `from=` body from the source's served lines and reports `copied_first`/`copied_last` from the raw slice.
- **ADR-073's read note** is emitted after the file loop (`read.go:1287`); the new note is not.

## Decision

This is the recommendation: **option B**, a write-time rule. Alternatives Considered costs the other three, so the owner can accept this as it stands or choose another.

1. **The rule is judged on the assembled output, from where its first line came.** The raw original still governs matching and validation. After every hunk is spliced: if the original began with `M`, the file written begins with exactly one file mark. If the first output line is (a) original line 1 in place (nothing consumed or inserted before it), nothing changes; (b) a body line that begins with `M`, that mark is the file's and is not doubled; (c) anything else (a body line without `M`, an original line promoted by a deletion, a line that came from further down), the mark is put in front of it, and the promoted line keeps every one of its own bytes as content. A `HasPrefix` test on the final bytes is not the rule: `Mone\nMtwo\n` with line 1 deleted must give `MMtwo\n`. The rule covers line edits (replace, insert, delete, and `from=`, `--sub`, `--rewrite`, `search_replace` and `apply_patch` plans, which compile to them); `unlink` is not one, and a rename moves the bytes as they are.
2. **A body's leading mark is the file's, never content.** A body line that begins with `M`, whether typed, computed by `--sub` or `--rewrite`, or copied, is read as the file's mark when it is the first output line. So `MMone\n` with a raw rewrite that consumes only the signature `[0,3)` produces the body `Mone`, and the file is `Mone\n`: the content mark the hit left in the body is read as the signature. This is the one place where "a second mark is content" does not hold, and it is stated so a caller can predict it; a hit that spans `[0,6)` gives the same bytes.
3. **A mark does not stay on a line that moved.** If original line 1 survives unchanged but not first (`insert-before 1`, `insert-after 0`, an insert and a replacement at the same position), its leading `M` is taken off it. A promoted line that begins with `M` keeps it as content (Decision 1c).
4. **A file with no lines.** A marked file that an edit leaves with no lines is exactly `M`, 3 bytes. `insert-after 0` and `insert-before 1` into the file `M` (which the unchanged splitter reads as one unterminated line that is the mark) both give `Mzero\n`, with a final LF as the same edit on an empty file would write.
5. **The receipt, the preview and the pad describe the file that lands.** `lines_after` is the count the unchanged splitter reads in the landed bytes: deleting every line of `Mone\ntwo\n` lands `M`, and `lines_after` is 1 (`sha_after` from the staged bytes), in a dry run and a live write, and on the destination row of a delete-all plus rename; the hunk's `-2 +0` stays the splice count. A computed dry-run preview (`--sub`, `--rewrite`) and `--echo-pad` show landed text: `MONE`, and a pad that includes the adjusted tail (`insert-before 1` with `--echo-pad 1` shows line 2 as `one`, without the mark). The shape hints (`balance`, `closer`) judge the body as submitted, since the mark is not something the caller typed and cannot change delimiter balance: replacing line 1 of `Mx\n}\n` with `}` raises the closer hint as it does without a mark.
6. **The ledger follows the same rules, not the same records.** After a native write to line 1 the record is whole-file with nothing shown (ADR-203): a bare `1 replace` is refused until line 1 is shown, while `anchor="ONE"` lands. After a quote-only `SEARCH Mone` → `REPLACE ONE` the record licenses `[1,1]`, so a later bare `1 replace` can land (ADR-208's rule for quote-only lines); B inherits this for a line whose mark mrw put back. `SEARCH Mone` → `REPLACE one` is byte-identical under B, so the same bytes are written again, as apply writes any byte-identical file, and a prior showing survives under the same-sha merge. A dry run leaves the ledger alone. Each is tested.
7. **A refusal teaches the retry.** The exact-match delete refusal, and the `SEARCH`/`old side` misses, say when the file's line 1 begins with a mark and the caller's text equals it without the mark: `…: line 1 begins with a UTF-8 byte order mark (U+FEFF); an exact match must include it, and a pattern can say `^\x{FEFF}one$``. The guide says the same, how to write the character in an old side, that `^one` misses line 1 while `^.one` and `^\x{FEFF}one` match, and that an old side `one` can match line 2 of `Mone\none\n` instead, which the refusal cannot tell. Quote verification stays over the raw bytes.
8. **The read note has one rule.** It is `-- note: a UTF-8 byte order mark begins this file and is part of line 1: ^one does not match it (^.one and ^\x{FEFF}one do), and every edit keeps it at byte zero`, printed after the `==>` header and before the first range of a file, in the CLI's stdout, inside `--json` `content`, and under `-N`. It is report text: it counts toward `--max-chars` and the MCP ceiling and earns no line count, no showing, no licence and no checkpoint. It is printed when the answer prints line 1 of a marked file, once per file per invocation (`read f.txt:1 f.txt:1` prints one); a line 1 that `--max-cols` cuts or `--fold` displays counts as printed, and the line stays unlicensed. `--quiet`, `--stat`, `--count`, the plan makers, `--max-lines 0` and a successful `--if-changed`/`--if-served` reference print none. The file `M` is an ordinary marked file: it prints the note when its line 1 is printed and not for `read f.txt:2`. The note and line 1 are delivered together or not at all: a `--max-chars` that cannot hold both withholds both, names line 1's range on standard error, and exits 1 `incomplete:` as any cap does, recording only delivered lines. Over MCP the note is in `content[0]` outside the checkpoint spans, and at the ceiling an open-ended read may page, a finder may return an index and a closed range that cannot fit is refused, as before.
9. **`from=` of line 1 of a marked file** copies the line without its mark, so the mark does not travel into the middle of another file: the body is `one`, and `copied_first` and `copied_last` report the body's lines (`one`), not the raw slice's. At the top of a marked destination it lands as `Mone` (Decision 1c); at the top of an unmarked one, as `one`. A source `MMone\n` copies the body `Mone`, which at byte zero of a marked destination is read as the destination's mark (Decision 2) and lands `Mone`. A source that is only `M` copies one empty line. The source's sha and showing gate stay raw. **This amends ADR-167 Decision 3 for this case only.**
10. **Transport marks stay transport.** A leading mark on a plan or a foreign document is dropped by the existing parser and unchanged here; a mark before a later `@@` takes part in header detection; a counted raw body line can carry `M@@` as content; `read -` drops leading marks and records no licence; `write --create` keeps them as content and names them (ADR-142). B changes none of these.
11. **No receipt key and no new row** (ADR-111 untouched). The receipt does not say a mark was kept; the next read says the mark is there.
12. **A line edit cannot remove a mark.** To remove one, write the content without it to another path and, in ONE plan, `unlink` the marked path and `rename` that file onto it (the path-op code accepts and orders the pair, and the replacement brings its own mode); a direct `unlink` then `create` of the same path needs two plans and gives the new file a default mode. An op for it is deferred (BACKLOG).

**What would make this wrong.** A caller who edits a marked file to remove the mark by replacing line 1: it is kept, and the next read's note says so. A caller who relies on a pattern anchored at `^`: it still misses line 1 and the note says why. A write that used to remove the mark now keeps it: a change of the bytes written, in the direction the issue asks for.

## Alternatives Considered

- **A. The mark is the file's metadata: strip it at split, give it back at the join (the first draft).** A new splitter for seven target sites and a rule for every consumer of line 1. Its costs, each from the first challenge (C1 to C13): the join must be lossless (`M M one`, a promoted line); a saved `@@ f /^one/ replace occurrence=1 sha=<raw>` retargets from `one-B` to `one-A` with its guard passing, and `--grep '^one' --sub` gains a site; quotes that included the mark (anchor, `/^\x{FEFF}one/`, a delete's expectation, SEARCH, `-M one`, generated anchors) turn from matching to refusing; old ledger showings and pending MCP checkpoints survive by raw sha for text that is now different (`--if-served`, `unchanged[].bytes`); the mark-only file; caps and the notice; the hook read, the null-read recount, the ignore parser, bench and measure tooling; `--json-path`, outlines, binary detection and invalid UTF-8 after the mark; ast-grep's raw offsets; Windows target files against transport marks; fences that execute. It gives the cleanest read (`^one` matches line 1). Not recommended.
- **C. Advisory only.** A new row kind under an `ok` hunk and a NEW receipt key (not `advisories`, which counts balance rows, and not `hints`, which counts closer rows), appended to `docs/receipts.txt`, when a write removes, moves or copies a file's leading mark, plus the same read note. It reaches more receipt surfaces than B (both receipt types, the schema, the golden, `docs/receipts.txt`, the contract) and needs a detector for three shapes; its engine can be SMALLER than B's, since it neither normalises output nor reconciles line counts, previews and `from=` bodies. Nothing is prevented: a mark an agent did not mean to lose is reported after the file changed, and its remedy is to type U+FEFF at the start of a body, which an agent cannot see. It is the right answer if a line edit must stay able to remove a mark. Not recommended.
- **D. Document only.** One sentence in the instructions: "a marked file's line 1 includes the mark; type it into a replace's body to keep it". The silent loss stays. The right answer if one probe is too little to spend an engine change on. Rejected.
- **The issue's option "keep the mark unless the body brings its own"** is B with its corner cases decided (Decisions 1 to 4).

## Component / Boundary Impact

`internal/apply` (`text`, the assembly loop and join, the receipt's counts and previews, `fromcopy.go`, the delete refusal), `internal/read` (the note), `internal/ingest` (the two foreign misses name a mark: one sentence each), `internal/guide` and `scripts/contract.sh`. Not touched: the splitter (`internal/lines`), `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/writer`, `internal/mcp`'s non-test code, the ledger, the receipts' keys. An engine diff: it names this record as owner and gets a `go-reviewer` pass (`.claude/rules/lifecycle.md` §6).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a line edit of a marked file | the output begins with one mark; a moved line 1 loses its mark | T1 | every write |
| `lines_after`, `sha_after`, previews, the echo pad | describe the landed bytes | T1 | receipt readers |
| `from=` of line 1 of a marked file | the copy has no mark; `copied_first`/`last` report the body | T1 | `from=` callers (ADR-167 amended) |
| `mrw read` / `mrw_read`: one `-- note:` line | new, with the rule of Decision 8 | T1 | callers, adapters |
| an exact-match refusal on a marked line 1 | one sentence | T1 | callers |
| `mrw instructions`, README, MCP descriptions | one sentence each | T2 | agents, hosts |
| contract §380 | new | T2 | the contract gate |

No receipt key is added (ADR-111).

## Inter-task Contracts

T2 consumes T1's behaviour and the note's exact words.

## Implementation

`tasks/README.md`: T1, T2.

## Consequences

- **Positive:** no write loses, moves or copies a file's mark by accident; every read, pattern, anchor, sha, ledger key, offset is what it was; the receipt describes the file that lands.
- **Negative:** a mark cannot be removed by a line edit (the unlink-plus-rename plan does it); `^one` still misses line 1, and the note and the refusals are what say why; a body that begins with a mark loses a content mark in the rewrite shape of Decision 2; the note adds a line to every read that prints line 1 of a marked file.
- **Neutral:** the bytes, shas and line numbers of every file read are unchanged.

## Out of Scope

- An op that removes a file's leading mark (deferred: docs/adr/BACKLOG.md "From ADR-214")
- A `bom` key in the read or write receipt (deferred: docs/adr/BACKLOG.md "From ADR-214")
- A start-anchored pattern matching line 1 of a marked file (permanent: boundary: reads, patterns, anchors and the ledger see the raw bytes, which is what ast-grep, `--json-path` and the hook see; making them agree is option A)
- A mark-looking sequence inside a file's content, anywhere but its first bytes (permanent: boundary: only the leading bytes are an encoding signature; the pre-existing content marks of a file are untouched, and B moves a file's own signature only)
- UTF-16 and UTF-32 files (permanent: fact: ADR-073 refuses edits to them and a walk skips them as binary; citation: file `internal/lines/lines.go:64`)
- A mark on standard input and in a created file's content (permanent: fact: #685 and ADR-142 decide those, and a created file's content keeps its mark; citation: file `internal/ingest/create.go:38`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The assembly misjudges where the first output line came from | Med | High | T1's table pins every shape's bytes; a mutant per rule |
| A receipt count, preview or pad disagrees with the bytes | Med | Med | T1 asserts `lines_after`, `sha_after`, the preview and the pad against the landed bytes, dry run and live |
| The note breaks a cap, a checkpoint or an adapter | Low | Med | T1 tests the budget boundary, the MCP checkpoints and the ceiling |
| A body's leading mark is read as the file's when it was content | Low | Low | Decision 2 says so; a test pins it |
| CRLF files get the mark after the first line | Low | High | T1's CRLF and CR-only rows |

## Rollback

Revert T1 and T2: a write treats a mark as line-1 text again, as on `main` at `5587200f`.

## History

- 2026-11-21: drafted for the owner's choice, first as design A (the mark is the file's metadata), with a measured table.
- 2026-11-21, Codex xhigh challenge at `7128fd6c`: AMEND, thirteen points, each checked against the code and each holding; the first draft's factual error is C3 (`--grep 'one' --sub ONE` keeps the mark). The recommendation moved from A to B; A, C and D are costed alternatives; the fences stopped resting on a grep of call sites and on lexical checks.
- 2026-11-21, second Codex xhigh challenge at `6a3fec13`: AMEND, nine points, B kept. (1) The rule moved to the assembled output with the first line's provenance, `Mone\nMtwo\n` delete-1 and the table of exact bytes adopted as T1's fixtures; the raw-rewrite `MMone` case is stated (Decision 2). (2) The receipt, preview and pad describe the landed file (Decision 5). (3) The ledger follows the same rules (Decision 6). (4) Refusals and the guide teach the retry, and the note's `^` claim is corrected (Decision 7). (5) The note has one eligibility, position and budget rule (Decision 8). (6) ADR-167 is amended for `from=` and double marks (Decision 9). (7) C is repriced and the removal recipe corrected (Decision 12). (8) The fences prove the promises: both controls, receipt assertions, the frozen-package clause, §380 executed. (9) Transport marks stay transport (Decision 10).
- 2026-11-21: accepted by Zy as option B. Before the status changed, the replace-line-1 row was re-run on a `mrw` built from `main` at `5a853823` (the file begins `4f 4e 45`, exit 0, no advisory); the fences' test names and §380 are free.

## Follow-ups

- [x] The owner chooses before T1 starts: B as recorded, A, C (advisory with its own receipt key) or D (document only). Zy, 2026-11-21: B as recorded.
