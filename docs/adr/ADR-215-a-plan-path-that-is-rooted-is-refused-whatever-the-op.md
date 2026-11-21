# ADR-215: A plan path that is rooted is refused, whatever the op

**Status:** Proposed
**Date:** 2026-11-21
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-006, ADR-007, ADR-029, ADR-051, ADR-071, ADR-076, ADR-111, ADR-132, issue #648
**Governs:** `internal/apply/apply.go`, `internal/apply/pathop.go`, `internal/plan/plan.go`, `internal/guide/guide.go`, `scripts/contract.sh`
**Enforced-by:** `internal/apply/rooted215_test.go::TestEveryPlanOpRefusesARootedPath`
**Served-path change:** (if Decisions 1 to 5 are accepted as written) a plan path that begins with a separator or names a drive (`/dd/zz.txt`, `\dd\zz.txt`, `C:x`, `C:\x`) is refused for every op with one sentence that names the root-relative spelling, exit 1 and nothing written. Today it applies as `ROOT/dd/zz.txt` for replace, insert, delete, unlink, rename source and create, and an absolute path to a file inside the root creates a copy of its whole directory chain under the root. A plan that applied yesterday is refused tomorrow, so the release note carries it.

## Context

**What #648 found.** A `create` whose path begins with `/` applies, as a path relative to the root; a `rename` whose destination is spelled the same way is refused with "every path in a plan is relative to the root". The two disagree about what a leading slash means, and the refusal's own words describe the create as well. The issue records that they differ; which reading is right is a decision (ADR-076 is the nearest record and does not decide it).

**Measured** on a `mrw` built from `main` at `5587200f`, scratch roots, every file read before it was written. Rows marked ✓ were re-run for this record; the others are fix-740's matrix (`b648/matrix.out`, same binary) and were not re-run. `ROOT` is the `--root`; `/abs/in` is the absolute path of a file inside it, `/abs/out` of one outside it.

| Plan path | Op | Result |
|---|---|---|
| `/dd/zz.txt` (file `ROOT/dd/zz.txt` exists) | replace ✓ | applied to `ROOT/dd/zz.txt`, exit 0; the receipt row says `/dd/zz.txt` and `target dd/zz.txt` |
| same | insert-after, delete, unlink, rename source | applied (fix-740's matrix) |
| `/sub/y.txt` | create ✓ | `ROOT/sub/y.txt` created, with `created sub/`, exit 0 |
| `/abs/in` | replace ✓ | refused: `is absolute, and every path in a plan is relative to the root: mrw looked for ROOT/abs/in, which does not exist`, exit 1; insert, delete, unlink, rename source the same (matrix) |
| `/abs/in` | create ✓ | applied: `ROOT/<every component of /abs/in>` is created, a whole copy of the directory chain under the root, exit 0 |
| `/abs/out` | replace, insert, delete, unlink, rename source ✓ (replace) | refused as above; nothing outside the root was touched in any run |
| `/abs/out` | create | applied under the root, as `/abs/in` (matrix) |
| rename destination `/sub2/y.txt` ✓ | rename | refused (`pathop.go:239`) |
| `body=@/dd/zz.txt`, `from=/dd/zz.txt:1` | parse | refused at parse, exit 2 (`plan.go:1076`, `:614`) |
| `--create /p.txt` ✓ | create from stdin | refused: `the path is not relative to the root`, exit 2 (`ingest/create.go:23`) |
| `apply_patch` and `search_replace` documents naming `/x` | ingest | refused, exit 2 (ADR-051 F-15; `ingest/applypatch.go:265`, `searchreplace.go:49`) |
| `mrw read /dd/zz.txt` ✓ | read | `REFUSED … is outside the root`: the read side takes a leading slash as the real filesystem path |
| `mrw read /abs/in` ✓ | read | served as `in` (`read.go:1350`) |
| `mrw write --grep P --sub R /abs/in` ✓ | walk | the plan names `in`: a walk turns the argument into a root-relative path, so it is never rooted |
| `mrw check` with a rooted path outside the root | check | refused (`check.go:601`); inside the root it is accepted |

**Where the two readings come from.** `rooted.Resolve` does `filepath.Join(absRoot, path)` (`internal/rooted/rooted.go:111`), which nests `/x` under the root. `IsRooted` is consulted only where a caller thought of it: to word the "does not exist" message (`apply.go:1985`, `pathop.go:225`), for the rename destination (`pathop.go:239`), at parse for `body=@` and `from=`, and in the ingest formats. Every other op, and create, never tests the spelling. `resolve` (`apply.go:3519`) is the one function every hunk path goes through.

**A third reading, in the header splitter.** `@@ /zz.txt 1 replace` exits 2: `unterminated pattern in header: expected a second /`. The rule, from `internal/plan/plan.go:728`: a `/` at the start of ANY token, the path token included, opens a pattern address, which runs to the next unescaped `/` anywhere later in the line. So a rooted path with a second slash (`/dd/zz.txt`) happens to close its own "pattern" and reads as a path; a rooted path with one component (`/zz.txt`, `/dd`) is unterminated, exit 2; a later slash inside a quoted anchor (`@@ /zz.txt 1 replace anchor="x/y"`) closes it there and ends in `unterminated quote in header`; and a first segment with a space (`/a b/c.txt`) stays one token. A path is always the first token of a header and an address always the second, so the splitter reads the first token as a pattern for no reason.

**No record decides it.** ADR-007, ADR-071, ADR-076 and `AGENTS.md`'s Portability section say how to ask (`rooted.IsRooted`, never `filepath.IsAbs`) and not what the answer means for a plan path. ADR-051 F-15 is the one decided refusal (the two foreign formats). Windows, from code only: `IsRooted` is true for `/x`, `\x` and `C:x`; `\x` behaves as `/x`; `C:x` joins to `ROOT\C:x` and is refused by `win32Alias` with a message about NTFS streams.

**How much (1) would break.** Counted, not guessed. In the tests and the contract: no test or contract row asserts that a leading-slash plan path applies; the ones that name one assert the refusal (`apply_test.go:892`, `reason198_test.go:306`, contract lines 1275, 5866-5878 for `body=@/etc/hosts`). Of the `*_test.go` files and `scripts/` only a docs script mentions an `@@ /` header. In local use: of 1,025 Claude Code transcripts on this machine last written before 2026-11-19, none holds a receipt row `ok   /<path>` for an applied hunk, and none holds the "is absolute" refusal; 52 `@@ /` header strings occur, in prose and examples. That source is one agent host on one machine: it does not cover Codex, Cursor or Windows, `mrw stats` does not count this, and a receipt not captured in a transcript is not seen. The harm the transcripts do not show but the table does: an agent that pastes an absolute path from a tool's output into a `create` gets a nested copy of the directory chain under the root, silently, exit 0.

## Existing Primitives Audit

- **`rooted.IsRooted`** is the predicate, already used at the five places above; no new one is needed.
- **`resolve` (`apply.go:3519`)** is where every hunk path becomes a file; the refusal belongs there or just before it.
- **The refusal already worded** at `apply.go:1985` and `pathop.go:225`/`:239` says "is absolute, and every path in a plan is relative to the root: mrw looked for …, which does not exist": true for the cases it fires on, false as an explanation of why the create applied.
- **`rooted.Contains(absRoot, p)`** says whether an absolute path is inside the root; `read.go:1356` uses it.
- **ADR-132's rule on exit codes**: a refusal caused by the plan's own path is a failed hunk (exit 1, siblings `skip`, nothing written); parse errors are exit 2.

## Decision

This is the recommendation. Alternatives Considered says what the other choices cost.

1. **A rooted plan path is refused for every op, at one place.** `resolve` (or the loop that calls it) refuses a hunk path for which `rooted.IsRooted` is true, before the file is looked up: replace, insert-before, insert-after, delete, create, unlink, rename (source and destination), and the same for a path inside the root. On Windows `/x`, `\x`, `C:x` and `C:\x` are all rooted and all refused, with the one message below instead of `win32Alias`'s.
2. **The refusal names what to write.** `P is absolute, and every path in a plan is relative to the root: write D`, where D is the path relative to the root when `P` is inside it (`rooted.Contains`), the path without its leading separators otherwise (`/dd/zz.txt` → `dd/zz.txt`), and, for a drive path or a drive-relative one, the sentence without a D ("name it relative to the root"). It keeps the words the three tests that pin the refusal already hold (`apply_test.go:887`, `reason198_test.go:306`, contract line 1275: "is absolute, and every path in a plan is relative to the root") and drops "mrw looked for …, which does not exist", which a rooted path no longer reaches.
3. **It is a failed hunk, exit 1.** The plan's own path is the cause, as ADR-132 classes it: the hunk's verdict is `FAIL`, siblings report `skip`, nothing is written. `body=@` and `from=` stay parse errors, exit 2 (permanent: a parse error is the plan document's).
4. **The header splitter reads the first token as a path.** A path is never a pattern, so `@@ /zz.txt 1 replace` parses and gets Decision 1's refusal, exit 1, where it exits 2 with a message about patterns. Addresses (the second token) are read as before.
5. **`mrw_write` and the CLI are one engine**, so MCP is as above. The `--create`, `apply_patch` and `search_replace` refusals happen before a plan exists and keep their exit 2 and their words (Out of Scope). A walk's plan paths are root-relative and are not affected.
6. **The read side is not changed.** `mrw read` keeps taking an absolute path to a file inside the root as that file and refusing `/dd/zz.txt` as outside it. The remaining asymmetry, a read takes a real path and a plan takes a root-relative one, is documented in the instructions (T2) with the refusal's own sentence.

**This is a breaking change** in the sense that matters to a caller: a plan that applied is refused. It changes no exit code of the table in `AGENTS.md` (a failed hunk is already 1); the release note names it. The refusal says what to write, so the migration is one edit.

## Alternatives Considered

- **(1b) Refuse a rooted path, except an absolute path to a file inside the root, which names that file (as the read side does).** Removes the asymmetry for one spelling and leaves `/dd/zz.txt` refused, so the word "rooted" would mean two things. It adds a spelling the ledger, the receipt rows and the alias lookup (ADR-029) must treat as the same file as the relative one, and `rename` destination, `body=@`, `from=` and the ingest formats would have to follow or stay inconsistent. Not recommended; it is the right choice if the owner wants an agent's pasted absolute path to work.
- **(2) Accept `/x` as root-relative everywhere** (make the rename destination, `body=@`, `from=` and the ingest formats agree, and document it). Rejected: the read side would still refuse `/dd/zz.txt` as outside the root, or flip to serving ROOT/dd/zz.txt for what a caller means as the filesystem's; and an absolute path pasted from any tool's output, the usual way an agent meets one, becomes a copy of its directory chain under the root (the `/abs/in` create row), silently, exit 0. The one spelling `Join` makes harmless for `/dd/zz.txt` is the one that makes `/Users/x/proj/f.go` harmful.
- **(3) Leave it and document the difference.** Costs nothing now. The nested-copy create stays, and a document would have to say "an absolute path is relative to the root for create and not for replace". Rejected.
- **Make the splitter reject a leading `/` for a path with a clearer message and nothing else.** Smaller than Decision 4, and leaves the exit-2 versus exit-1 split for one family of spellings. Rejected as the recommendation; acceptable if Decision 4 is judged out of scope.

## Component / Boundary Impact

`internal/apply` (`resolve`, the two refusals in `pathop.go`, the not-exist wording), `internal/plan` (the header splitter's first token), `internal/guide` and `scripts/contract.sh`. Not touched: `internal/rooted` (`IsRooted`, `Resolve` and `Contains` are used as they are), the read side, the ingest formats, the ledger and `internal/writer`. This is an engine diff: it names this record as owner and gets a `go-reviewer` pass (`.claude/rules/lifecycle.md` §6).

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a hunk path that is rooted | refused, exit 1, one sentence naming the root-relative spelling | T1 | every plan caller, CLI and `mrw_write` |
| a header whose first token begins with `/` | read as a path | T1 | the plan parser |
| `mrw instructions`, README | one sentence: a plan path is relative to the root; a read may take a real path | T2 | agents |
| contract §381 | new | T2 | the contract gate |
| the release note | names the change and its migration | the release | callers |

No receipt key is added (ADR-111).

## Inter-task Contracts

T2 consumes T1's message and behaviour.

## Implementation

`tasks/README.md`: T1, T2.

## Consequences

- **Positive:** one meaning of a leading slash in a plan, the one the refusal already states; no silent nested copy of a directory chain; `@@ /zz.txt` gives the refusal that says what to write instead of a message about patterns.
- **Negative:** a plan that applied is refused; a caller who relied on `/x` meaning `x` edits the plan once. The read and the plan still differ on an absolute path inside the root.
- **Neutral:** no exit code of the contract table changes, and nothing outside the root was ever touched.

## Out of Scope

- Accepting an absolute path inside the root as that file (deferred: docs/adr/BACKLOG.md "From ADR-215")
- The words of the `--create`, `apply_patch` and `search_replace` refusals (permanent: boundary: they refuse before a plan exists and exit 2 as usage, ADR-051 F-15; unifying their words would change a contract those records pin)
- `body=@` and `from=` staying exit 2 (permanent: boundary: they are errors of the plan document, found at parse, not verdicts of a hunk)
- The read side's treatment of a leading slash (permanent: fact: a read takes the real filesystem path and refuses one outside the root; citation: file `internal/read/read.go:1350`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller's plan that applied is refused | Med | Med | the sentence names the spelling to write; the release note names the change |
| Windows spellings differ from the measured POSIX ones | Med | Med | T1's test drives `IsRooted` with the Windows spellings through its seam; the Windows CI shard runs it; a peer re-measures |
| The splitter change alters a header that parsed | Low | Med | T1's test pairs every header form that parsed before with its result now |
| A refusal at `resolve` runs for a path the engine also resolves for a link or an alias | Low | Med | T1 reads every caller of `resolve` and says which paths reach it (the audit) |

## Rollback

Revert T1 and T2: a rooted path is joined to the root again, as on `main` at `5587200f`.

## History

- 2026-11-21: drafted for the owner's choice, from fix-740's matrix and a re-run of its key rows. Not yet challenged.

## Follow-ups

- [ ] The owner chooses before T1 starts: the Decision as recorded, (1b), (2) or (3).
