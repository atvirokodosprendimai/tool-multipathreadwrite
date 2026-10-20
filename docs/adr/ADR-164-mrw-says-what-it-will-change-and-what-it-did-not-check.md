# ADR-164: mrw says what it will change, what a guard does not cover, and what its check left out

**Status:** Accepted
**Accepted:** 2026-10-20 by Zy — "Accept (Recommended)" for the record as drafted, six tasks, after Zy chose all four plan items ("Scoped + name the rest" for T4) and asked for the peer report's three refusals as a sixth task ("sixt task, no baclog")
**Date:** 2026-10-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-032, ADR-035, ADR-042, ADR-054, ADR-055, ADR-056, ADR-061, ADR-087, ADR-111, ADR-124, ADR-140, ADR-141, ADR-152, ADR-161
**Governs:** `internal/apply/apply.go`, `internal/guide/guide.go`, `internal/check/check.go`, `internal/read/plansource.go`, `internal/read/maxcols.go`, `internal/read/sub.go`, `internal/plan/plan.go`, `internal/writer/flow.go`, `internal/preview/**`, `internal/statehint/**`, `internal/mcp/tools.go`, `internal/mcp/ack.go`, `internal/mcp/sub161.go`, `internal/mcp/schema.go`, `internal/mcp/instructions.go`, `cmd/mrw/main.go`, `cmd/mrw/sub.go`, `cmd/mrw/foreignflag.go`, `cmd/mrw/statehint.go`, `docs/receipts.txt`
**Enforced-by:** None — eight unrelated surfaces; each task's tests and contract row (§273–§280) hold its own decision
**Invalidates:** ADR-003 — Decision: "If any changed path is not a Go file the scoped form is abandoned for the full one", and its Risks row "It cannot: any non-Go path forces the full command" (line 182), whose cited `TestScopeDerivation` case T4 rewrites: a non-Go regular FILE beside mapped Go paths is now named, not a reason to run everything; ADR-054 — "When `packages()` cannot map (non-Go paths), the whole-project `Check` runs", already narrowed by ADR-061 (its line 11), now narrowed again to the cases Decision 4 keeps; ADR-061 — "A `{packages}`-only template still falls back to `Check`" (line 37) and "Mixed templates keep falling back" (line 74) now hold only when nothing maps; ADR-124 Decision 2 — "otherwise the full check runs, as before" likewise; ADR-032 — the elision order of an oversized `mrw_write` receipt gains two steps (Decisions 2 and 4); ADR-137 — Decision 2's "centred on the first match" now means centred on the match itself (Decision 7); ADR-161 — Decision 1's `--sub` replacement still expands `$1` and `${name}`, and a reference to a group the pattern lacks is refused (Decision 8; found first-hand while writing this amendment, when a `--sub` replacement holding those two examples wrote them as nothing); ADR-140 — Out of Scope line 71, "`-v` `-o` `-z` (permanent: boundary: a hint would be a guess)": `-v` and `-o` get a pointer (Decision 5). ADR-042 is NOT invalidated: a directory of prose or `testdata` still falls back (its line 33), because Decision 4 applies to files only.
**Served-path change:** a `--sub`/`--rewrite` dry run shows each changed line; a multi-line replace refused for no `anchor=` says that a `sha=` it carries does not stand in; a check whose `{packages}` template can place some edited files runs those packages and names the files it left out; `mrw mcp` and the CLI plan makers name the state directory when they cannot use it; `mrw read --grep -v` says there is no inverted match; a pattern address ended by an unescaped `/`, a line served cut by `--max-cols`, and an addressed `create` are each refused naming their real cause; a `--max-cols` window holds a match that fits; a `--sub` replacement naming a group its pattern lacks is refused

## Context

Two sources, both 2026-10-20.

**A field report** another agent wrote after a batch edit with mrw v1.68.0, pasted by Zy: 25 changes in 22 files went through one plan and a wrong plan wrote nothing, and four asks. Each was checked against the binary that day and filed in `docs/adr/BACKLOG.md`, "From a field report on batch editing (2026-10-20)" (added by PR #460, which this record merges after):
1. an unwritable state directory said only `mkdir /nonexistent: read-only file system` — fixed for the CLI's read and write paths by PR #460 (contract §272);
2. `--sub` shows nothing of what it changes: `mrw write --grep red --sub blue --dry-run .` printed `ok   s.css 1 replace  -1 +1` and no text;
3. a multi-line replace carrying `sha=` and no `anchor=` was refused, and the caller had taken `sha=` for the guard;
4. balance advisories are noisy on CSS, regex and string bodies.

**This session's own use** while fixing item 1 (PR #460, five Codex rounds):
- a write touching `cmd/mrw/*.go` and `scripts/contract.sh` ran the whole project's check, because `scoped_check` `go test {packages}` cannot place a `.sh` path and `packages()` is all-or-nothing (`internal/check/check.go`, `packages`); it hit the 300 s timeout and exited 3, tree unverified;
- the hint does not reach `mrw mcp`, nor the CLI's plan makers: `read.RangeFreePaths` and `read.BuildSub` turn a state error into `BuildError` text (`internal/read/plansource.go`), which no caller can unwrap;
- `mrw read --grep -v ' a' x` refuses `' a'` for edge whitespace before any flag hint runs, and `--grep -v a x` gets no pointer (ADR-140 left `-v` out); `--grep -i ' a' x` has the same whitespace shape;
- three balance advisories (`( +0 → +2`) fired on test files whose only unbalanced parentheses were inside string literals — item 4 seen first-hand.

**A peer's field report** (the quality-harness session, 2026-10-19/20, mrw v1.69.0-dev+18c171d1), each item re-run against that binary on 2026-10-20: a pattern address holding `` `scripts/release.mjs` `` was refused `unknown op "\\|"` — the unescaped `/` ends the pattern and the rest of the header is read as the op (`\/` applies); a line served cut by `--max-cols` was refused `43 … has not been read: mrw served lines 44`, true by ADR-137 and silent about why; `@@ n.txt 1 create` is refused `use "-"` while `0 create`, which AGENTS.md teaches, also applies. Two more asks were checked and need nothing: a JSON parse after a write is `--then-sh 'jq -e -s "length == 1" f.json'` today, and a reported en-dash decoding did not reproduce (mrw decodes nothing in a body). Zy, 2026-10-20: these three are a sixth task, not BACKLOG.

**Two more, 2026-10-20, each reproduced against v1.69.0-dev+39c347c3:** this session's own T5 work found that `--max-cols` centres its window on a match's START (`internal/read/maxcols.go`, `start = first - width/2`), so `--max-cols 8` over `NEEDLE` served `…pha NEED…`, a window that cuts a match it could hold; and a third peer session (infrastructure docs) reported that `mrw write --grep P --sub R` with a `$1` in R and no capture group in P wrote the line with `$1` deleted, exit 0, "0 failed, 0 advisories" — Go's expansion of a reference to an undefined group is the empty string. Zy chose the window as a seventh task; the `$1` case is the eighth, under Zy's rule for peer reports ("sixt task, no baclog").

**Audit of the class** — *a place where mrw's output leaves the caller to guess what happened*: the four field-report asks, the four session findings, and what two challengers of this record found inside them (2026-10-20: a deleted package reaching the check as a directory, module files, the whitespace shape of `--grep -v`, the MCP sites a regex missed). Enumerated by hand from two transcripts and two reviews, not by a command; each task's first step enumerates its own sites with a command and records the count.

Stage 5 (ADR-162) is unaffected: its harness pins `mrw-v1.68.0` (`run-block.sh`, line 11).

## Existing Primitives Audit

- `apply`'s `closerAfter` already holds each replace hunk's replaced lines and body (`internal/apply/apply.go`); `HunkResult.Echo` already carries lines under an ok hunk (ADR-052). T2 adds `Before`/`After` there, so no caller re-reads a file.
- `read.BuildSub` / `read.BuildRewrite` (ADR-161) already route both surfaces; both emit numeric addresses with `sha=`.
- `check.command()` already returns whether the scoped form applied; `scopeHint` already explains a fallback (ADR-152).
- `cmd/mrw/statehint.go` (PR #460) holds `stateHint`, `aboutState`, `exitWith`, `exitSaying`; T3 moves the first two.
- `foreignFlags` and `patternFlagHint` (`cmd/mrw/foreignflag.go`, `cmd/mrw/main.go`; ADR-140) are what T5 extends.
- No line-diff helper exists in served code; `internal/preview` is new.

## Decision

1. **`sha=` is not `anchor=` (T1).** The refusal of a multi-line `replace` with no `anchor=` (`internal/apply/apply.go`, the `carries no anchor=` site) adds, when the hunk carries `sha=`: `; sha= checks that the whole file is unchanged, not which lines this range covers`, and when it carries `lines=`: `; lines= checks how many lines this range covers, not which` (Zy, 2026-10-20, after T1's class sweep of the hunk guards). Its existing words stay, so §73 and `TestAMultiLineReplaceWithoutAnAnchorIsRefused` still hold. The full guide's sentence ("A multi-line replace requires anchor=…") gains `sha= and lines= do not stand in for it.` The core (`mrw instructions --core`) is NOT changed: it is 299 words against a ceiling under 300 (`cmd/mrw/core141_test.go`), measured 2026-10-20.
2. **A plan maker's dry run shows each change (T2).** When a plan came from `--grep … --sub` or `--ast-grep … --rewrite` (CLI) or `grep`+`sub` / `ast_grep`+`rewrite` (`mrw_write`) and the write is a dry run, `apply` fills `HunkResult.Before` (the replaced lines) and `HunkResult.After` (the body) for each ok `replace` hunk (JSON `hunks[].before`, `hunks[].after`, appended to `docs/receipts.txt` for `write` and `mcp_write`). The text receipts print, under the ok row, through `internal/preview`:
   - one line replaced by one line: `   ~ N| <line with the change marked [-old-]{+new+}>`. The marked span runs from the first to the last byte that differs; 40 characters of context each side; a span longer than 160 characters keeps its first and last 70 with `…[K chars]…` between, so both ends of every change — two matches on one line included — stay visible;
   - otherwise: `   - N| <old>` for each replaced line, then `   + N| <new>` for each body line, a line paired with the line at the same index on the other side marked and cut by the same span rule as the `~` form (first to last difference, 40 characters each side, a span over 160 keeping its first and last 70), and a line with no counterpart cut to its first 200 characters, so every change on a paired line is shown, however far apart.
   Line endings are not shown: the lines are the split `internal/lines` makes (ADR-065). A real write and a plain plan's dry run fill and print nothing new. Under the `mrw_write` ceiling (ADR-032), `before`/`after` and the preview lines are the first things elided, named in `elided`.
3. **The state-directory hint reaches every surface (T3).** `stateHint`'s judgement moves from `cmd/mrw/statehint.go` to a new `internal/statehint` (`Hint(err) error`), unchanged in meaning: by type, never text. `read.BuildError` gains the error it was built from (`Err`, `Unwrap`), kept by `RangeFreePaths` and `BuildSub`/`BuildRewrite`; `exitOf` keeps it and its exit code. Every `internal/mcp` route that returns a state error, enumerated by T3's first step, wraps with `%w` and passes `statehint.Hint` before it becomes an `isError` text or a JSON-RPC message, keeping its shape and code.
4. **A check that can place some edited files runs those and names the rest (T4; Zy, 2026-10-20: "Scoped + name the rest").** When `scoped_check` uses `{packages}` and neither `{files}` nor `{dirs}`, `packages()` is judged per path:
   - a `.go` path that cannot be placed, any directory path that holds no package (a removed file reaches the check as its parent directory, `writer.CheckPaths`), a write that removed or renamed away a `.go` or module file (the writer, which knows what it removed, tells `check.Run` to run the whole check; the name never reaches `confine`, which keeps refusing a missing path, ADR-042), and `go.mod`, `go.sum`, `go.work` or `go.work.sum` anywhere empty the map, so the whole `check` runs, as today;
   - any other regular file is set aside when it cannot be placed;
   - when at least one path maps, the scoped command runs over the mapped packages and each set-aside file is listed in `check.Result.OutsideScope` (JSON `outside_scope`; CLI and `mrw_write` line `outside the scoped check: <paths> — scoped_check "<template>" runs {packages}, which names Go packages only`).
   A template with `{files}` or `{dirs}` already receives every path and reports nothing outside. Nothing mapped: the whole check, ADR-061's empty-map rule and ADR-152's hint, unchanged. The verdict and exit code are the scoped command's: a pass with paths outside exits 0 — the trade ADR-003 refused for a *quiet* omission, named here on every receipt. Under the `mrw_write` ceiling the path list may be elided but its count never is: the terminal phrase says `Its check passed for the packages it could place; N file(s) were outside it`. `mrw check PATH…` follows the same rule. The name is not `unchecked`, which `mrw stats` already uses for "no check ran" (ADR-056).
5. **`-v` and `-o` get a pointer (T5).** `foreignFlags` gains `v`: `there is no inverted match: --grep serves the lines that match` and `o`: `there is no match-only output: --max-cols N cuts each line to a window round its match`. The edge-whitespace refusal of the value after `--grep` (`cmd/mrw/main.go`, the padded-argument check) appends the table's pointer when the flag before that value is in the table, so `--grep -v ' a'` and `--grep -i ' a'` name it too. `-z` stays out.

6. **A refusal names its real cause (T6).** In `internal/plan`: when a header's op is unknown, its address field is a pattern (`/…/`), and a later field that is not a `key=value` option still holds a `/` (so `anchor="a/b"` or `body=@dir/f` beside a misspelled op do not trigger it), the refusal adds `— the address %q ended at the first unescaped "/"; write a "/" inside a pattern as "\/"`. In `internal/apply`, the not-read refusal adds `(a line --max-cols served cut is not read: read it whole, or --quiet)`; the ledger does not record cut lines, and recording them would change its format (ADR-150), so the clause is on every not-read refusal. The `create`-address refusals (`internal/plan` and `internal/apply`) say `use "-" or "0"`. Every refusal keeps its kind, exit code and existing words.

7. **A `--max-cols` window holds the match when it can (T7).** `window` (`internal/read/maxcols.go`) centres on the match itself, not its first character: a match no longer than N lies whole inside the window; a longer one starts the window at the match. With no match the window still starts the line. The cut line licenses nothing, as before (ADR-137).
8. **A `--sub` replacement may not name a group the pattern lacks (T8).** `read.BuildSub` refuses — on the CLI's `write` and `read` a usage error, exit 2, and on `mrw_write`/`mrw_read`'s `sub` an `isError` tool result, nothing planned or written on either — a replacement whose `$N`, `${N}`, `$name` or `${name}` names a group number above the pattern's count or a name the pattern does not define, saying which and that a literal dollar is written `$$`. A reference the pattern defines expands as before (ADR-161).

Each decision fails if: (1) a `sha=`-only multi-line replace's refusal lacks the clause, one without `sha=` gains it, or the core changes; (2) a plan-maker dry run shows no change, hides one end of a change, or a plain dry run or real write shows one; (3) a CLI plan maker or `mrw mcp` with an unusable state directory answers without naming it, or an answer changes shape; (4) a `.go` + `.sh` write runs the whole check or omits the `.sh` silently, or a removed package, a module file or an unplaceable `.go` runs anything but the whole check; (5) `--grep -v` in either shape ends without the pointer; (6) one of the three refusals lacks its cause, or any refusal's kind, exit code or existing words change; (7) a match no longer than N is cut by its window; (8) a `--sub` replacement naming an undefined group writes anything.

## Alternatives Considered

- **Put the `sha=` sentence in the core.** Rejected: the core is at its word ceiling, and the refusal reaches exactly the caller who made the mistake.
- **A `--diff` flag for every dry run.** Rejected: a plain plan's author wrote the body; a plan maker's dry run is the one place the caller has not seen the change.
- **Re-read the old lines in the CLI and MCP at each hunk's address.** Rejected: `apply` already holds them; a second read is a second chance to disagree.
- **A unified diff instead of the `~` line.** Rejected for one-for-one lines: `--sub` changes spans inside a line, and a `-`/`+` pair of a 300-character line hides where.
- **Keep the whole-project fallback and only hint (ADR-152).** Rejected: on this session's write it cost 300 s and an exit 3 for a change whose Go half was fully testable in its own package, and the hint only tells a caller to rewrite a template that is already right for Go; Zy chose against it, 2026-10-20.
- **Per-path for directories too.** Rejected: a removed package reaches the check as a directory, and skipping it would leave its importers untested.
- **Match the state error by message text.** Rejected in #460's third Codex round: a refusal can echo caller text.

## Component / Boundary Impact

- `internal/apply` (engine): one clause in one refusal (T1); `HunkResult.Before`/`After` and an `Options` switch (T2). Owned by this record.
- `internal/check` (engine): per-path `packages()`; `Result.OutsideScope` (T4). Owned by this record.
- `internal/read` (engine): `BuildError.Err` (T3). Owned by this record.
- `internal/writer` (not engine): `Verify` tells `check.Run` to run the whole check when the write removed or renamed away a `.go` or module file (T4).
- New `internal/preview`, `internal/statehint`: leaf packages imported by `cmd/mrw` and `internal/mcp`.
- `cmd/mrw`, `internal/mcp`: print; route MCP errors; schema descriptions; recipe 2's wording.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| refusal text (CLI, `mrw_write`) | one clause when `sha=` is present | T1 | the caller |
| `mrw instructions` (full) | one sentence; recipe 2's words | T1, T2 | the caller |
| JSON `write hunks[].before`, `write hunks[].after`, `mcp_write hunks[].before`, `mcp_write hunks[].after` | new, appended | T2 | `--json` callers; ADR-111's tests |
| dry-run receipt text | `~`/`-`/`+` lines under plan-maker hunks | T2 | the caller |
| `read.BuildError.Err` | new field, `Unwrap` | T3 | `exitOf`, `internal/mcp` |
| `mrw mcp` and plan-maker error text | names the state directory | T3 | the caller |
| `check.Result.OutsideScope`; JSON `check outside_scope`, `write check.outside_scope`, `mcp_write check.outside_scope` | new, appended | T4 | the caller; ADR-111's tests |
| the check command for a mixed write | scoped over placed packages | T4 | every write in a Go module with a `{packages}`-only template |
| `mrw_write` output schema and `tools/list` (`internal/mcp/schema.go`, `testdata/legacy_golden.jsonl`) | descriptions for the new keys; the golden regenerated, as ADR-152 did for `check.hint` | T2, T4 | MCP hosts |
| `mrw read` usage errors for `-v`, `-o` | a pointer | T5 | the caller |
| refusal text: unknown op after a pattern address, not-read, `create` address | names the cause | T6 | the caller |
| `--max-cols` window placement | centred on the match | T7 | `read` callers |
| `--sub` with an undefined group reference | refused, exit 2 | T8 | the caller, CLI and MCP |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `internal/apply/apply.go` edits | T1 | T2 | No — different functions; T2 rebases on T1 |
| `internal/mcp/tools.go`, `schema.go`, legacy golden | T4 | T2 | No — each adds keys; T2 rebases on T4 |
| `internal/mcp/tools.go` | T2 | T3 | No — T3 rebases on T2 |
| `internal/apply/apply.go` | T1 | T6 | No — different refusals; T6 rebases on T1 |
| `internal/read` (`maxcols.go`, `sub.go`/`plansource.go`) | T7, T8 | T3 | No — different functions; whichever lands later rebases |

## Implementation

See `tasks/README.md`. This record merges after PR #460, whose BACKLOG section and `cmd/mrw/statehint.go` it builds on.

## Consequences

- **Positive:** recipe 2's dry run shows what will change; a mixed write's check takes the Go packages' time; every surface names its state directory.
- **Negative:** a pass can leave named files untested; a large `--sub` dry run is longer.
- **Neutral:** nothing changes for a plain plan, a `{files}`/`{dirs}` template, or a write whose paths all map or none do.

## Out of Scope

- Changing `mrw instructions --core` (permanent: boundary: Decision 1; the core is at its word ceiling, measured 2026-10-20)
- Balance-advisory noise on CSS, regex and string bodies (deferred: docs/adr/BACKLOG.md "From a field report on batch editing (2026-10-20)", the balance entry; arm: `mrw stats` advisory rates from a CSS- or regex-heavy repository against ADR-055's bar)
- A `--state-dir` flag (permanent: boundary: `XDG_STATE_HOME` moves it and every surface now says so)
- `-z` in the flag table (permanent: boundary: no honest pointer; ADR-140)
- A ledger record of lines served cut, so the not-read refusal could say it only when true (permanent: boundary: a ledger format change, ADR-150; the clause costs one parenthesis on a refusal)
- A refusal kind for the missing anchor (permanent: boundary: ADR-087 types only the parser-mirrored and not-read refusals)
- The hint on `seen --prune`, which prints each removal's own error and exits on a count (permanent: boundary: its lines already name each path under the state base, found by Codex on #460, 2026-10-20)
- Mapping a `.sh` or testdata file to the Go tests that read it (permanent: boundary: mrw does not model which test reads which file; ADR-054 "Do not add a `covers` glob")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A Go test that reads an edited non-Go file is skipped | Med | Med | the file is named on every receipt; `mrw check --full` runs everything |
| A preview hides a change | Low | Med | Decision 2's windows keep both ends of every span; fixtures for a double match and a change past column 200 |
| An MCP state-error route is missed | Med | Low | T3's first step enumerates with a command and records the sites |

## Rollback

Revert per task. Before a release, a task's revert takes its `docs/receipts.txt` lines with it: ADR-111's tests hold the receipt types to that file in both directions. After a release, a shipped key stays in the type and the file (`omitempty`) and is never filled. T4 is the only change to what runs; reverting it restores the whole-project fallback.

## Follow-ups

- [ ] Rebase on `main` after PR #460 merges; mark the BACKLOG field-report items `--sub` and `sha=` as taken by this record in the record's commit.
