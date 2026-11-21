# ADR-215: A plan path that is rooted is refused, whatever the op

**Status:** Proposed
**Date:** 2026-11-21
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-006, ADR-007, ADR-029, ADR-051, ADR-071, ADR-076, ADR-111, ADR-132, ADR-176, issue #648
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/plan/plan.go`, `internal/guide/guide.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/rooted215_test.go::TestEveryPlanOpRefusesARootedPath`
**Served-path change:** (if the Decision is accepted as written) a plan path that is rooted (`/dd/zz.txt`, `\dd\zz.txt`, `C:x`, `c:/x`, `//server/share/x`, or an absolute path inside or outside the root) is refused for every op, as a failed hunk before any file is looked up: exit 1, siblings skipped, nothing written. Today it joins to the root and applies for replace, insert, delete, unlink, rename source and create, and an absolute path to a file inside the root creates a copy of its directory chain under the root. A plan that applied yesterday is refused tomorrow; the release note and the tag message carry the cases in Decision 9. A header whose first token begins with `/` and has no second `/` keeps exit 2 and gains a sentence.

## Context

**What #648 found.** A `create` whose path begins with `/` applies, as a path relative to the root; a `rename` whose destination is spelled the same way is refused with "every path in a plan is relative to the root". The two disagree, and the refusal's own words describe the create as well. Which reading is right is a decision (ADR-076 is the nearest record and does not decide it).

**Measured** on a `mrw` built from `main` at `5587200f`, scratch roots, every file read before it was written. Rows marked ✓ were re-run for this record; the others are fix-740's matrix (`b648/matrix.out`) or traced from the code by the Codex challenge, and were not re-run. `ROOT` is the `--root`; `/abs/in` is the absolute path of a file inside it, `/abs/out` of one outside it.

| Plan path | Op | Result today |
|---|---|---|
| `/dd/zz.txt` (`ROOT/dd/zz.txt` exists) | replace ✓ | applied to `ROOT/dd/zz.txt`, exit 0; the receipt keeps `/dd/zz.txt` and says `target dd/zz.txt` |
| same | insert, delete, unlink, rename source | applied (matrix) |
| `/sub/y.txt` | create ✓ | `ROOT/sub/y.txt` and its parents created, exit 0 |
| `/abs/in` or `/abs/out`, any op but create | replace ✓ (and the rest) | the joined path is `ROOT/<abs>`: refused with "is absolute … mrw looked for ROOT/<abs>, which does not exist" ONLY IF that nested file is missing. With root `/repo` a replace of `/repo/x` applies to `/repo/repo/x` when that file exists and is licensed ✓ |
| `/abs/in` or `/abs/out` | create ✓ | applied: the whole directory chain of the absolute path is created under the root, exit 0 |
| `/dd/zz.txt` and `//dd/zz.txt` ✓; `"/dd/zz.txt"` ✓ | header forms that parse | the path is read as the joined file; a quoted path avoids the pattern reading |
| rename destination `/sub2/y.txt` ✓ | rename | refused at `pathop.go:239`, but only AFTER the source is loaded and covered: an unread source gives "has not been read" first ✓ |
| `@@ "/dd/x/" 0 create` ✓ | directory spelling | refused by the directory rule first (`apply.go:538`) |
| `body=@/dd/zz.txt` | CLI/MCP body loading | exit 2 in `LoadBodyFiles` (`plan.go:1070`), after `Parse`, before `Apply` |
| `from=/dd/zz.txt:1` | parse | exit 2 inside `Parse` (`plan.go:614`) |
| `--create /p.txt` ✓; `*** Add File: /x`; `<<<<<<< SEARCH /x` | ingest | refused before a native plan exists, exit 2 (`create.go:23`, `applypatch.go:261`, `searchreplace.go:41`) |
| `mrw read /dd/zz.txt` ✓ / `read /abs/in` ✓ | read | refused as outside the root / served as the relative path (`read.go:1349`) |
| `mrw write --grep P --sub R /abs/in` ✓ | walk | the plan names the relative path: never rooted |
| `mrw check /abs/in` | check | refuses the scope when the nested name `ROOT/<abs>` is missing: the inside-root argument passes `Contains`, then `Resolve` re-joins it (`check.go:587`); an outside argument fails the earlier containment check. Source-traced, not re-run |
| `mrw iter add /dd/zz.txt:1` ✓ then `@@ @1 1 replace` ✓ | working set | the entry keeps the rooted spelling, and the pointer expands to it: the hunk is `/dd/zz.txt` and applies as above; `--create @1` and an `apply_patch` `Add File: @1` expand the same way |
| `@@ out.txt 0 create body=1` with the body line `@@ /zz.txt 1 replace` ✓ | counted body | written literally, exit 0 |
| piped `@@ /zz.txt 1 replace` ✓ | header | exit 2, `unterminated pattern in header: expected a second /`, the plan kept and named (ADR-176) |

**The compatibility set, by platform and by what parses.** POSIX: `/d/x`, `//x` and `"/x"` can apply today when the joined target passes its normal guards; plain `@@ /x 0 create` and `@@ /x 1 replace` hit the splitter error. `./x`, `a//b`, `.hidden`, `-x` and a literal `~/x` are relative and stay so (the parser does no tilde expansion); `x/` and `x/.` are the directory rule. Windows (from code, not run): `/x`, `\x`, `C:/x`, `c:x` and `//server/share/x` are rooted by `IsRooted`; `\x` behaves as `/x`. What `C:x` does today is not claimed: `win32Alias` strips the volume name before it looks for colons and `resolveIn` joins afterwards, so the earlier draft's explanation of the NTFS-stream message was wrong. T1 measures it on Windows. "Rooted" is the word, not "absolute": `c:x` is rooted and not absolute.

**Where the readings come from.** `rooted.Resolve` does `filepath.Join(absRoot, path)` (`rooted.go:111`). `IsRooted` is consulted where a caller thought of it. `resolve` (`apply.go:3519`) is not a single point: it serves rename's later resolution pass (`apply.go:853`), `foldClashes` (`apply.go:3692`, which treats a resolution error as "present"), ledger hints (`:3552`) and ancestor diagnostics (`:3796`), so a refusal inside it cannot fix the verdict or its order; the order the checks run in decides which refusal a caller meets.

**The header splitter.** A `/` at the start of ANY token, the path token included, opens a pattern address that runs to the next unescaped `/` anywhere later in the line (`plan.go:728`). So `@@ /zz.txt 1 replace` and `/dd` exit 2; `/zz.txt … anchor="x/y"` closes at the quote's slash and ends in `unterminated quote`; `/dd/zz.txt` closes at its own second slash and reads as a path; `/a b/c.txt` is one unquoted path today; `/a"b"/c.txt` keeps its quotes today. The same scanner reads `@@ dir/x /a\/b/ replace anchor="x/y"`, `/a/,/b/`, `@@ @1 /re/ replace` and an unquoted `anchor=prefix /a  b/` (the double space survives), which must keep working.

**No record decides it.** ADR-007, ADR-071, ADR-076 and `AGENTS.md`'s Portability section say how to ask (`rooted.IsRooted`, never `filepath.IsAbs`) and not what the answer means for a plan path. ADR-051 F-15 is the one decided refusal.

**How much a refusal would break.** In tests and the contract: nothing asserts that a leading-slash plan path applies; what names one asserts the refusal, and two of those assert words this record changes: contract §32 requires the doubled path `$R$R/sub/real.go` and `reason198_test.go:305-308` requires `mrw looked for …` for a rename destination and an unlink of `/nothere.txt`; `apply_test.go:887` keeps its two phrases. In local use: of 1,025 Claude Code transcripts on this machine last written before 2026-11-19 none holds an applied `ok   /path` receipt row or the "is absolute" refusal (52 `@@ /` strings, prose and examples). One host, one machine: no Codex, Cursor or Windows; `mrw stats` does not count this; the Codex challenge did not re-verify the counts. The harm the transcripts do not show is in the table: an absolute path pasted from a tool's output into a `create` builds a copy of its directory chain under the root, silently, exit 0.

## Existing Primitives Audit

- **`rooted.IsRooted`** is the predicate, used at the places above; **`rooted.Contains`** is a lexical prefix check (including equality with the root), so it says where a path points, not where a link under it resolves.
- **The directory-spelling rule and the grouping maps** (`apply.go:520-660`) are built from each hunk's path, so a refusal after them is a refusal of something already grouped.
- **ADR-132's rule on exit codes**: a refusal caused by the plan's own path is a failed hunk (exit 1, siblings `skip`, nothing written); parse errors and input-loading errors are exit 2.
- **ADR-176** keeps a piped plan that ended with nothing in the tree and names it in `refused_plan`.

## Decision

This is the recommendation. Alternatives Considered says what the other choices cost.

1. **A rooted plan path is refused for every op, by a preflight at the start of apply's validation.** After working-set pointers have been expanded and before the grouping, directory-spelling and collision maps are built, every hunk's RAW path and every rename's RAW destination is tested with `rooted.IsRooted`; a hunk that fails is a failed hunk and every other hunk is `skipped`; nothing is looked up, loaded or created. The ordinary resolver stays for filesystem checks and is not changed. Windows: `/x`, `\x`, `C:x`, `c:/x` and `//server/share/x` are rooted and refused, with the one message below.
2. **Precedence.** Errors before `Apply` keep their place and their exit 2: a parse error, `from=` (it fails in `Parse`), `body=@` (loaded after `Parse`, before `Apply`: `body=@missing.txt` on a rooted hunk is the body-file error), an ingest compilation, a foreign format's reads of its targets. From `Apply` on the preflight is first: before the directory-spelling rule (`"/dd/x/"` is the rooted refusal), before a rename's source loading and coverage, before `foldClashes`. A plan with a rooted hunk and an unrelated hunk that would have set `loadErr` exits 1, not 2, because nothing is loaded. This guarantees the order inside `Apply`; it does not change the caller's sequence, so a read that happens before `Apply` (a copy source, a foreign target) still happens.
3. **The message** keeps the first sentence the existing assertions hold: `P is absolute, and every path in a plan is relative to the root`. It drops "mrw looked for …, which does not exist", which a rooted path no longer reaches. It gives a concrete spelling only when the meaning is clear: for an absolute path that `rooted.Contains` puts inside the root, `: for the file inside the root write R` with R the lexical remainder, never a followed link's target (a rename or unlink of a symlink entry names the entry); otherwise it describes: `: name the file relative to the root, without the leading separator or drive`. Never strip a leading separator into a suggestion: `/etc/hosts` would point the next plan at an unrelated `etc/hosts`, `/../x` strips to `../x`, `/` to nothing, and `/repo/x` is the real file or the old nested `ROOT/repo/x` (the sentence says which it names). A header spelling that begins with `@` is written `./@…` and one with a space is double-quoted; a rename destination is the literal body line. Git Bash advice stays about editing the plan document: plan-file and standard-input bytes are not converted by MSYS, a pasted argument can be.
4. **The header splitter keeps its reading and says so.** A first token that begins with `/` and has no second `/` stays exit 2 (a document error, ADR-132) and its sentence gains `: a plan path is relative to the root and never begins with /`. No token boundary, pattern, quote or counted-body rule changes, so `/a b/c.txt`, `/a"b"/c.txt`, `@@ dir/x /a\/b/ replace`, `/a/,/b/`, `@@ @1 /re/ replace`, an unquoted `anchor=prefix /a  b/` and a literal body line `@@ /zz.txt 1 replace` behave as today. The cost: the same mistake exits 1 for `/dd/zz.txt` and 2 for `/zz.txt`, a difference the words explain.
5. **Working-set pointers.** `mrw iter add` keeps accepting a rooted spelling: it records a spec for reads, where an absolute path inside the root is valid. A pointer that expands to a rooted spelling in a plan, a `--create @N` or an `apply_patch` `Add File: @N` is refused by Decision 1 with the sentence of Decision 3 plus, for a pointer, `(the working-set entry @N holds it: mrw iter lists it; add it again without the leading separator)`. `--sub` and `--rewrite` normalise their walk inputs and keep emitting relative hunks.
6. **`mrw_write` and the CLI are one engine**: MCP returns `isError` with the structured receipt, failed row first and skipped siblings; a piped plan with a rooted hunk that ends with nothing in the tree is kept and named in `refused_plan`, as before, now at exit 1.
7. **The read side is not changed**: a read takes a real path and refuses one outside the root.
8. **Ledger keys.** No migration: a legacy successful write recorded `FileResult.Path`, so a rooted key can exist at the checkout top; it is left alone and the ordinary alias lookup stands. A legacy plan is repaired by reading the file it meant, under its relative spelling, and writing the plan with that spelling.
9. **The breaking-change note**, carried into the implementation PR and the tag message: `@@ /sub/y.txt 0 create` goes from exit 0 (creating `sub/y.txt`) to exit 1; replace, insert, delete, unlink and rename of a `/dd/zz.txt` that applied go from exit 0 to exit 1; an absolute path whose nested join existed goes from applied to refused; a rename to a rooted destination keeps exit 1 and loses "mrw looked for"; a `@@ @1` whose entry holds a rooted spelling goes from applied to refused. No exit code of the contract table changes.

## Alternatives Considered

- **(1b) Refuse a rooted path, except an absolute path to a file inside the root, which names that file as the read side does.** `read /repo/x` already serves and licenses `x`, so a caller can reuse the served name; accepting it in a plan adds a second spelling the ledger, receipt rows and alias lookup (ADR-029) must treat as one file, and rename destination, `body=@`, `from=` and the ingest formats would follow or stay inconsistent. Not recommended; deferred (BACKLOG). The right choice if an agent's pasted absolute path must work.
- **(2) Accept `/x` as root-relative everywhere** and make the rename destination, `body=@`, `from=` and ingest agree. Rejected: the read side would refuse `/dd/zz.txt` as outside or flip meaning, and an absolute path pasted from a tool's output (`/Users/x/proj/f.go`) becomes a nested copy of its directory chain under the root, silently.
- **(3) Leave it and document the difference.** Costs nothing now; the nested-copy create stays. Rejected.
- **Make the splitter read field one as a path** (the first draft's Decision 4), so `/zz.txt` exits 1 with the uniform sentence. Rejected after the challenge: it breaks `/a b/c.txt` (needs quotes), `/a"b"/c.txt` (loses its quotes) and a literal counted body line `@@ /zz.txt 1 replace`, which would parse as a header and exit 2 unless the outer hunk carries `raw=true`.
- **Put the refusal inside `apply.resolve`** (the first draft). Rejected: `resolve` serves later passes and `foldClashes` swallows its errors, so the verdict and the order would depend on which caller reached it first (Decision 2).

## Component / Boundary Impact

`internal/apply` (a preflight at the start of validation, the message, the rename-destination site), `internal/plan` (one sentence of one error), `internal/guide` and `scripts/contract.sh`; T1 amends the two old assertions it breaks. Not touched: `internal/rooted`, the read side, the ingest formats' own refusals, the ledger, `internal/writer`. This is an engine diff: it names this record as owner and gets a `go-reviewer` pass (`.claude/rules/lifecycle.md` §6).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a hunk path or rename destination that is rooted | refused by the preflight, exit 1, the sentence of Decision 3 | T1 | every plan caller, CLI and `mrw_write` |
| a header whose first token begins with `/` and has no second `/` | exit 2 as before, one sentence added | T1 | the plan parser |
| contract §32 and `reason198_test.go` | amended: the joined-path and `mrw looked for` assertions become the new sentence | T1 | the contract gate |
| `mrw instructions`, README | one sentence: a plan path is relative to the root; a read may take a real path | T2 | agents |
| contract §381 | new | T2 | the contract gate |
| the release note and the tag message | Decision 9 | the release | callers |

No receipt key is added (ADR-111): `reason`, `status`, `path` and `target` carry different values.

## Inter-task Contracts

T2 consumes T1's message and behaviour.

## Implementation

`tasks/README.md`: T1, T2.

## Consequences

- **Positive:** one meaning of a leading slash in a plan, the one the refusal already states; no silent nested copy of a directory chain; a refusal that comes first and says what to write only when it is sure.
- **Negative:** a plan that applied is refused; a legacy `@N` entry must be re-added; the same mistake exits 1 or 2 by how many slashes the header holds; the read and the plan still differ on an absolute path inside the root.
- **Neutral:** no exit code of the contract table changes; nothing outside the root was ever touched.

## Out of Scope

- Accepting an absolute path inside the root as that file (deferred: docs/adr/BACKLOG.md "From ADR-215")
- `mrw iter add` refusing a rooted spelling (permanent: boundary: the working set holds specs for reads, where an absolute path inside the root is valid; a write judges the expanded spelling as a plan path)
- The words of the `--create`, `apply_patch` and `search_replace` refusals (permanent: boundary: they refuse before a plan exists and exit 2 as usage, ADR-051 F-15; unifying their words would change a contract those records pin)
- `body=@` and `from=` staying exit 2 (permanent: boundary: they are errors of the plan document, found before `Apply`, not verdicts of a hunk)
- The read side's treatment of a leading slash (permanent: fact: a read takes the real filesystem path and refuses one outside the root; citation: file `internal/read/read.go:1349`)
- Changing the header splitter's token rules (permanent: boundary: the first draft's change broke three forms; Alternatives Considered)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A rooted spelling reaches a lookup before the preflight (a path through `foldClashes`, rename's later pass, a pointer) | Med | High | T1's fixtures reach each branch: `/new/x` with a missing nested target, a licensed source with a rooted destination, an expanded pointer, a case pair |
| Windows spellings differ from the POSIX ones measured | Med | Med | a real Windows apply test for `c:x`, `\x` and UNC on the Windows CI shard; relative `nul.txt`, `b.txt::$DATA` and a junction keep their existing checks |
| The suggested spelling points at the wrong file | Low | High | a concrete spelling only when the path is lexically inside the root; otherwise a description (Decision 3) |
| A saved plan or a working-set entry that applied is refused | Med | Med | the sentence names the fix; the release note names the case |

## Rollback

Revert T1 and T2: a rooted path is joined to the root again, as on `main` at `5587200f`.

## History

- 2026-11-21: drafted for the owner's choice, from fix-740's matrix and a re-run of its key rows.
- 2026-11-21, Codex xhigh challenge at `bc3d465e`: AMEND, nine points, each checked against the code (the re-run rows above) and applied; the direction stands. (1) The `check` row and the blanket claim about absolute paths were wrong: a non-create op at an absolute path depends on whether the nested file exists, and `mrw check` refuses an inside-root argument too. (2) `resolve` cannot carry the verdict or the order: the check moved to a preflight on the raw path and raw rename destination, with the precedence stated. (3) Working-set pointers widen the break: Decision 5, a disposition for `iter add`. (4) The suggested spelling could point at an unrelated file: Decision 3. (5) The compatibility set is stated by platform and by what parses, and the `C:x` explanation withdrawn. (6) The first draft's splitter change broke three forms: Decision 4 now only adds a sentence, and the old design is a costed alternative. (7) Mixed failures, MCP, the kept plan and legacy ledger keys are decided (Decisions 2, 6, 8). (8) The fences execute the positive case and reach the branches (T1, T2). (9) T1 owns the assertions it breaks, and the breaking-change note is Decision 9.

## Follow-ups

- [ ] The owner chooses before T1 starts: the Decision as recorded, (1b), (2) or (3).
