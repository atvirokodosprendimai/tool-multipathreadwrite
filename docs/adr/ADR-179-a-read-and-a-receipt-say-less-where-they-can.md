# ADR-179: A read and a write receipt say less where they can

**Status:** Accepted
**Accepted:** 2026-10-27 by Zy — *"yes, go, address these, measure, plan, implement verify, udupate the instrucitons, no counter-arguing, all follow same direction"*, and, after the plan was presented, *"go on, implement all of it"*.
**Date:** 2026-10-27
**Amended:** 2026-10-27, before acceptance, from two Codex challenges and a Claude adr-challenger pass, each finding checked against the code: `--if-served` takes the sha the caller holds and judges each line with `Observation.ServedLine` (`internal/seen/seen.go:81`), keeps a missed range's report, and names when it is safe; a per-file notice from `--outline` or `--no-comments` is one line per invocation; `MRW_MAX_COLS` is read with `os.Getenv` and reaches standard input (`cmd/mrw/stdin.go:103-117`); `--max-matches` defines its boundary with context and the line after a range; T5 enumerates the notices it holds; each flag task names its flag in `CLI()` (`cmd/mrw/instructions_test.go:32`); the baseline is re-measured over a fixed window. ADR-178 (#541) and ADR-177 (#536) are merged, so ADR-179 ships on its own branch from main.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-033, ADR-038, ADR-052, ADR-054, ADR-055, ADR-100, ADR-104, ADR-111, ADR-118, ADR-119, ADR-133, ADR-136, ADR-137, ADR-141, ADR-144, ADR-146, ADR-148, ADR-153, ADR-154, ADR-157, ADR-161, ADR-164, ADR-165, ADR-169, ADR-170, ADR-171, ADR-175, ADR-177, ADR-178, docs/bench/output-study/README.md, docs/agents.md, README.md, AGENTS.md
**Governs:** `internal/read/*.go`, `cmd/mrw/main.go`, `cmd/mrw/stdin.go`, `internal/guide/guide.go`, `docs/agents.md`, `docs/bench/output-study/**`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/longline179_test.go::TestAReadOfAnOverlongLineHintsOnceOnStderr`
**Invalidates:** none — checked: every part is opt-in or a stderr line; `--max-cols` (ADR-137, ADR-154, ADR-164), `--if-changed` (ADR-169), `mrw write --quiet` and the `--json` receipts (ADR-111) keep their meaning; one receipt key is added (`same_as_served`).
**Served-path change:** a read that serves a line over 400 columns whole says so once on stderr, and `MRW_MAX_COLS=N` makes `--max-cols N` the default; `mrw read` gains `--outline`, `--if-served PATH=SHA`, `--fold`, `--max-matches N` and `--no-comments`, each of which shows less than the lines and licenses only what it shows; `mrw write` gains `--terse`; no notice is printed twice in one invocation.

## Context

Zy, 2026-10-27, on the output study: *"yes, go, address these, measure, plan, implement verify, udupate the instrucitons, no counter-arguing, all follow same direction."*

**The registered baseline** is `docs/bench/output-study/baseline-2026-10-27.json` and `repeat-baseline-2026-10-27.json`: `measure_output.py` and `measure_repeat.py` with `--since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z`, every tool result stamped in that window, 129 main-session transcripts (107 with an mrw call), 100,931,328 characters of tool output. `mrw read` alone is **57.3M characters in 19,075 calls**; a further 6,943 calls (5.8M) ran `mrw read` and `mrw write` in one command. Each part of this record addresses one bucket:

| part | bucket | upper bound | calls |
|---|---|---|---|
| (a) long lines | lines over 400 columns in `mrw read`, beyond 200 | 4.55M | 1,939 (2,080 reads already passed `--max-cols`) |
| (b) outline | whole-file reads over 150 lines | 2.73M (Read 1.57M, `mrw read` 1.16M) | 125 |
| (d) served repeats | identical mrw content blocks served again in one session | 0.22M | 240 |
| (c) fold | runs identical up to digits / identical | 1.14M / 0.17M | 463 over 500 chars |
| (e) notices | the two counted mrw notices, all of them / notice lines repeated inside one result | 0.10M / 0.03M | 243 / 149 |
| (g) match budget | `mrw read --grep` output | 16.4M | 7,525 |

The first study run (2026-10-27, files chosen by mtime; the figures this record was first written with) reported 0.52M of repeated mrw blocks; it counted blocks that held no served line, which the fixed script does not.

A second peer session (mac-quality-harness-main-dev) measured one session on 2026-10-27, mrw v1.76.0: mrw output was 83% of 709K characters; ranged reads 340K in 69 calls, `--grep` reads 246K in 114, write receipts 52K for 44 plans; the largest reads were 15-25K characters of 120-180-line ranges taken to edit about 5 lines, partly to get each line's licence; and quality-harness Verification Log `test-lock` rows run to a **median of 3,399 columns, a maximum of 39,685**. That session gives parts (f), (g) and (h); (f) and (h) are not measured on the 30-day corpus, and none of the three exists in v1.76.0 (checked on that binary).

Confounds, written before any code: subagent transcripts are not scanned; one machine, one owner; each bound assumes the part is used wherever it could be and that the agent needed nothing it removed. A notice line repeated inside one tool result may be two invocations in one command, each saying it once.

## Existing Primitives Audit

- **`--max-cols N`** (`internal/read/read.go:152-156`, `:1022-1031`; `window` at `internal/read/maxcols.go:15`; ADR-137, ADR-154, ADR-164): already cuts a line of any read, ranged or grep, to a window round the first match and leaves the cut line out of the observation, so a write to it is refused until it is read whole. Reuse unchanged for (a): the default and the hint only decide when it applies.
- **Two `read.Options` sites**: the file read (`cmd/mrw/main.go:1506`) and standard input (`cmd/mrw/stdin.go:103-117`). Every part that sets an option sets it at both, or says why `-` refuses it.
- **The flag conflict table** (`cmd/mrw/main.go:1076-1217`) is keyed on `cmd.IsSet`; a flag given a value from the environment through the parser would be `IsSet` and trip `--sub`'s and `--rewrite`'s refusals of `--max-cols` (`:1107`, `:1122`). So `MRW_MAX_COLS` is read with `os.Getenv`, never as a flag source.
- **A read that serves no line** `--count` (ADR-153, `internal/read/read.go:157-160`): hashed, no spans. Reuse for `--outline`.
- **`--if-changed PATH=SHA`** (ADR-169, `internal/read/read.go:191-200`, `:733-762`): a caller-held sha, compared with the file on disk; a spec answered without text still reports its missed ranges and exits 1. (d) is its span-level sibling and keeps that rule.
- **The ledger's observation** `seen.Observation` (`internal/seen/seen.go:67-74`) and `ServedLine` (`:81-86`): a line counts as shown if a read served it in this version — for a file mrw wrote, only the lines read since (`Shown`, ADR-118, merged at `:839-846`). (d) asks `ServedLine` of every line it would serve and changes nothing in `internal/seen`. `mrw seen --forget` (ADR-175) empties the ledger.
- **Notices said once**: the neighbour note once per read (ADR-136, `internal/read/read.go:987-989`), a damaged ledger told once (ADR-144), a newer ledger said once (ADR-157); the null-device and still-licensed notices (`cmd/mrw/main.go:1554-1569`, `cmd/mrw/nullread148.go:25`). `mrw read --grep '"mrw: |"-- [a-z]' --count cmd internal --exclude '*_test.go'` finds 37 notice sites in 17 files (2026-10-27), a lower bound. (e) holds the class with one test.
- **`mrw write --quiet`** (`cmd/mrw/main.go:1706-1707`, `report` at `:2981`): prints failed hunks and the summary only — and drops the `balance` and `closer` advisory rows (`:3053-3060`) with the ok rows they sit under, which is why it is not (f). The default receipt prints an `ok` row per hunk and a file line per file; a preview (`preview.Lines`, `:3048`) only for a plan maker's dry run (`Options.Preview`, `:2188`).
- **`--max-lines N`** caps each spec (ADR-033, `internal/read/read.go:136-151`); `--count` gives counts only (ADR-153); `--max-chars N` bounds characters and lists what it withheld (ADR-165). A ranged read also serves the line after each range (`:908`), and context spans merge (`:869`). (g) is the cap across the whole read, in matches.
- **`CLI()` names every read flag**: `TestEveryReadFlagIsTaughtByInstructions` (`cmd/mrw/instructions_test.go:32`) goes red for a read flag `guide.CLI()` does not name, so each flag task names its own. `Skill()` prints `Core()` and `Recipes()`, not `CLI()` (`internal/guide/guide.go:159-168`); the repository's `.claude/skills/mrw/SKILL.md` must equal it (`cmd/mrw/skill168_test.go:301`).
- **The standard library's `go/parser` and `go/scanner`**: (b) and (h) parse Go with them; no new module requirement (`CONTRIBUTING.md`, Prerequisites).

## Decision

**(a) Long lines say so; a default can cut them (T1).** A content read that serves at least one line longer than 400 characters whole — no `--max-cols` given and none from the environment — adds one line to stderr after its answer: `mrw: N line(s) over 400 columns served whole, the longest L; --max-cols 200 cuts each to a window round the match, and a cut line licenses nothing until it is read whole`. Once per invocation (ADR-136), and only advice that helps (ADR-157). `MRW_MAX_COLS=N`, read with `os.Getenv` when `--max-cols` is absent, makes `--max-cols N` the default for a read that serves lines, files and `-` alike; it does not apply to `--quiet` (a read that exists to license would license less without saying so), `--sub`, `--rewrite`, `--count`, `--stat`, `--outline` or `--json-path` (ADR-178), and it never trips their refusals. `--max-cols 0` given on the command line turns it off for that call. A value that is not a non-negative integer is named on stderr with the form it takes, and ignored. CLI only: `mrw_read` has `max_cols` (ADR-154). This covers the "default column budget for task-file reads" the peer session asked for: a quality-harness task file's Verification Log rows (median 3,399 columns, max 39,685) are cut by the default, and the rows a caller edits are short.

**(b) An outline (T2).** `mrw read --outline FILE…` prints, under each file's header, one line per top-level item with its line range, and licenses nothing (the file is hashed, no spans, as `--count` records it). For `.go`: every top-level `func` (with its receiver), `type`, `var` and `const` declaration, from `go/parser` with `parser.SkipObjectResolution|parser.ParseComments`, `A-B func (s *Store) Get`; a file that does not parse prints what parsed and `!! parse error: …` under its header, and the read exits 1 `incomplete:`. For `.md` and `.markdown`: each ATX heading outside a fenced block, `A-B ## Heading`, the range ending before the next heading of the same or a higher level; a fence closes only on the same character, at least as long as its opener, with nothing after it, and a closing sequence needs whitespace before it (`# C#` is `# C#`). A declaration a parse error cut off has no end and prints `A-? …`. Any other file gets its header and no outline, and one stderr line for the whole invocation names them — `mrw: --outline reads .go and .md files; no outline for a.py, b.sh` (ADR-148's five names and a count) — and the read exits 1. Refused beside it, exit 2, by name: `--grep`, `--ast-grep`, `--sub`, `--rewrite`, `--count`, `--stat`, `--quiet`, `-C`, `--if-changed`, `--if-served`, `--fold`, `--no-comments`, `--max-matches`, `--json-path`, `--max-lines`, `--max-cols` (a cap on a structure means nothing), `-`, and a PATH with an address. Under `--max-chars` a file whose rows do not fit is named on standard error (`PATH: outline WITHHELD: --max-chars reached`) and counted a problem, so `--json` agrees with the exit.

**(d) A repeat can be answered by reference, opt-in (T3).** `mrw read --if-served PATH=SHA SPEC…` (repeatable, one per file, the sha the caller holds from a header it was shown — 8 to 64 hex digits, as `--if-changed` takes) answers a spec of PATH by `-- same as served: lines A-B[, C-D…] at sha X` under the file's header, instead of its text, when the file on disk still has that sha AND `Observation.ServedLine` is true for every line the spec resolves to (the line after each range included). A line covered by the union of several earlier reads counts; so does one an earlier `-N` read showed without its number. A file mrw wrote counts only for the lines read since (`Shown`, ADR-118). Anything else — a line never served, a changed file, a sha that does not match — is served as without the flag. A missed range is reported as without the flag (`!! no match for …`), and the read exits 1 `incomplete:` (ADR-169's rule). A spec answered by reference records nothing new. Under `--json` each reference is listed in the receipt as `same_as_served[]` (`path`, `sha`, `spans`), appended to `docs/receipts.txt`. Refused beside it, exit 2: `--sub`, `--rewrite`, `--count`, `--stat`, `--quiet`, `--outline`, `--json-path`, `--fold`, `--no-comments`, `-`, and `--if-changed` naming the same PATH under any spelling, whatever the finder matches. A reference beside `--max-matches` is no clash. **Opt-in only, and the guide says exactly when it is safe:** the ledger is the checkout's, not the session's (ADR-038), and a line recorded as shown may never have reached the model — a host that truncates a long output, a read piped to `head`, a read redirected to a file. So `--if-served` is for a session alone in its checkout, whose host empties the ledger on start, resume, clear and compaction (ADR-175's hook, matcher `startup|resume|compact|clear`), and whose reads were made under `--max-chars` below the host's output limit (ADR-165). Asking for the sha is what keeps a fresh session from matching: it holds no header to quote. No footer is added to any read: the header's `sha` and the `@@ A-B` line already name the block (Alternatives gives the arithmetic).

**(c) Fold (T4).** `mrw read --fold` — on files and on `-` — prints a run of two or more consecutive non-blank lines that are identical, or identical once each run of digits is read as one placeholder, as its first line followed by `-- ×N: lines A-B like A` (with `, differing only in digits` when they are not identical). The first line is licensed; the folded lines are not, as a line `--max-cols` cut is not (ADR-137), so a write to one is refused until it is read without `--fold`. Lines are compared by their text between digit runs, so a literal `#` is no digit. Only the lines asked for fold: the line after a range (ADR-161) is never part of a run, and a line a `--grep` matched is never folded away, so `--fold --grep P` shows every match and folds only context. Refused beside `--sub`, `--rewrite`, `--count`, `--stat`, `--quiet`, `--outline`, `--if-served`, `--no-comments`, `--max-matches` (a folded line the budget counted would not be shown) and `--json-path`.

**(e) Each notice once per invocation (T5).** Every stderr line mrw prints that begins `mrw:` and every `-- ` note inside a read's answer is printed at most once per invocation; a notice about many files, or many specs of one file, is one line naming every path it covers once (ADR-148's form: five names and a count where a list could be long), so `a.py, b.py` and never one line per file. Per-range reports are not notices and stay one per range: `!! …`, `WITHHELD`, `NOT FOUND` and a `--count` total. T5's test drives one fixture that triggers each notice on a list the task enumerates (from the command in the Audit, the new ones from (a), (b), (g) and (h) included) across several specs and files, and fails on any notice printed twice — and on any listed notice the fixture did not trigger.

**(f) A terse write receipt (T6).** `mrw write --terse` prints one line per file — `ok   PATH  N hunk(s)  -A +B  sha X`; for an unlink `ok   PATH  unlink  was sha X`; for a rename `ok   OLD -> NEW  rename`; under `--dry-run`, `dry  PATH  N hunk(s)` — and the summary line, with no hunk preview. Every failed hunk with its skip rows, and every `balance`, `closer` and `hint:` row (ADR-054, ADR-055, ADR-119), still prints in full under its file, and so do the check's report and every step verdict, with their exit codes (3 when the check fails). It is a flag, not the default: the default receipt is asserted by contract rows, and the measurement behind it is one session. `--json` and the MCP receipt are unchanged (ADR-111). Refused beside `--quiet`, exit 2.

**(g) A match budget across the read (T7).** `mrw read --grep P --max-matches N` (and `--ast-grep`) serves matches in walk order until N matching lines have been shown, then starts no new span and keeps walking only to count. A matching line shown as context (`-C`) or as the line after a range (ADR-161) counts as shown and toward N; such a match is served with its own span, context and next line included, so a budget no smaller than the matches serves and records what the read without the flag does; an ast-grep hit is one match however many lines it spans, and overlapping hits count once each. When matches were left unshown, the answer ends `-- max-matches N reached: M more matching line(s) in K file(s) not shown` and the read exits 1 `incomplete:` (ADR-146), as `--max-lines` and `--max-chars` do when they withhold, and when only the budget withheld the kind line names it (`incomplete: --max-matches N reached, M matching line(s) not shown`); when exactly N matched, nothing is said and it exits 0. Only what was shown is licensed, and a match counts only if it reached the answer: one `--max-lines` or `--max-chars` cut is counted as unshown, matches in files `--max-chars` kept back are not counted (the note says so), and a spec `--if-changed` answers unchanged is neither shown nor counted. `--max-matches 0` shows no match and counts them all (ADR-033). On `-` with `--grep` it applies to the one input. Refused without `--grep` or `--ast-grep`, and beside `--count`, `--stat`, `--sub`, `--rewrite`, `--outline`, `--fold`, `--no-comments`, `--json-path`.

**(h) Comment elision (T8).** `mrw read --no-comments` replaces each run of two or more whole-line comment lines in a `.go` file with `-- … N comment line(s), A-B`. The whole file is scanned with `go/scanner` even for a ranged read, so a range that starts inside a `/* */` block is known to, and a `//` inside a string is not a comment. A comment after code on the same line stays; so do `//go:` and `//line` directives and the cgo preamble (the comment immediately above `import "C"`). Every C import spec keeps its preamble, grouped imports included, and `//export` is a cgo directive and stays. Only the lines asked for are elided: the line after a range (ADR-161) is never part of a run, and a line a `--grep` matched is never elided. The elided lines are not licensed. A file that is not `.go`, a `.go` file that does not scan, and `-`, is served whole, and one stderr line for the invocation names them — `mrw: --no-comments reads .go files; served whole: a.py` — as (e) requires. Refused beside `--sub`, `--rewrite`, `--count`, `--stat`, `--quiet`, `--outline`, `--if-served`, `--fold`, `--max-matches` (an elided line the budget counted would not be shown) and `--json-path`.

**Order.** By measured payoff: (a), (b), (d), (c), (e); then (f), (g), (h). (b) and (h) each measure the share of their bucket they can reach (T2 S2, T8 S2), after their red tests and before any implementation, over the registered window and against a bar written in `docs/adr/BACKLOG.md` now; below the bar the task stops and goes to Zy.

**What would make this decision wrong:** the parts being taught and not used, or used and then followed by a full re-read, which pays twice. The pre-registered re-measure (`docs/adr/BACKLOG.md`, "Pre-registered for ADR-177, ADR-178 and ADR-179") names the expected change per bucket and what happens to a part whose bucket does not move.

## Alternatives Considered

- **(d) A served-answer id footer on every content read**, which the first brief asked for. Rejected on the numbers: about 20 characters per read × 19,075 reads (26,018 with the mixed calls) ≈ 0.38-0.52M characters added over the registered window, against 0.22M measured in repeated mrw content blocks — the footer would cost about twice what the most `--if-served` could save. The block's identity (path, sha, range) is already printed in the header and the `@@` line and already stored in the ledger.
- **(d) Answer from the ledger alone, with no caller-held sha.** The first draft. Rejected after the challenge: a fresh session inherits the previous session's ledger and would be answered by reference for lines it never saw; the sha the caller quotes is the least it can show it holds.
- **(d) A served record per session.** Rejected: mrw has no session identity (ADR-038, ADR-175 Alternatives).
- **(a) `--max-cols` on by default.** Rejected: a cut line licenses nothing, and a caller who never asked for a cut would be refused a write to a line it believes it read. The hint and the opt-in default reach the same callers without that surprise.
- **(a) The variable as a flag source in the parser.** Rejected: the flag would then be `IsSet` and refuse `--sub` and `--rewrite` reads (`cmd/mrw/main.go:1107`, `:1122`).
- **(b) and (h) through ast-grep for other languages.** Rejected for this record: each language needs its own rule set, a configuration surface this record would own, and the share the other languages carry is unmeasured. Deferred with a trigger.
- **(b) A line scan for Go instead of `go/parser`.** Rejected: a scan cannot find where a declaration ends; the parser is in the standard library.
- **(b), (h) A notice per file.** The first draft. Rejected: it is the repeat (e) forbids.
- **(f) Teach `mrw write --quiet` instead.** Rejected: it drops the advisory rows (ADR-055, ADR-119) whose job is to be read after a write.
- **(f) Make the terse receipt the default.** Not now: contract rows assert the preview, and the evidence is one session. Pre-registered as a question for the re-measure.
- **(g) A cap per file.** Rejected: that is `--max-lines`, which caps each spec already; the peer session's 114 `--grep` reads (246K characters) were walks over many files, which a per-file cap does not bound.
- **(e) Dropping notices.** Rejected: each says something a caller acts on (ADR-157); the measured cost is repeats, not the notices.

## Component / Boundary Impact

- `internal/read` (engine package, owned here): the long-line count for the hint, the outline writers, the `--if-served` answer, the fold, the match budget, the comment elision.
- `cmd/mrw`: the flags, their conflicts, `MRW_MAX_COLS`, the hint line, both `Options` sites, the write receipt's terse form.
- `internal/guide`: each flag's `CLI()` line (in its task) and the recipes (T9).
- `internal/seen` is read, not changed; if T5 finds a repeated notice printed by `internal/seen`, it stops and asks (T5 Stop Condition).
No package changes ownership. This record holds `internal/check` byte-identical, which ADR-177 changes, so it ships after ADR-177 (#536), which is merged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| stderr after a read with a line over 400 columns | one hint line | `cmd/mrw` | agents |
| `MRW_MAX_COLS` | new variable; the `--max-cols` default, files and `-` | `cmd/mrw` | agent setups |
| `mrw read --outline`, `--if-served PATH=SHA`, `--fold`, `--max-matches N`, `--no-comments` | new flags; each licenses only what it shows | `cmd/mrw`, `internal/read` | agents |
| `mrw read --json` receipt | `same_as_served[]` (`path`, `sha`, `spans`) added | `cmd/mrw` | adapters |
| `mrw write --terse` | new flag; `--json` unchanged | `cmd/mrw` | agents |
| notices | at most once per invocation | `cmd/mrw`, `internal/read` | agents |
| `mrw instructions`, `mrw skill`, `.claude/skills/mrw/SKILL.md`, README, AGENTS.md, `docs/agents.md` | the shorter reads taught | T9 (each flag's `CLI()` line in its own task) | agents |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| the hint line and `MRW_MAX_COLS` | T1 | T5, T9 | No |
| `--outline` and its one notice | T2 | T5, T9 | No |
| `--if-served` | T3 | T9 | No |
| `--fold` | T4 | T5, T9 | No |
| the once-per-invocation rule | T5 | T9 | No |
| `--terse` | T6 | T9 | No |
| `--max-matches` and its notice | T7 | T5, T9 | No |
| `--no-comments` and its one notice | T8 | T5, T9 | No |

## Implementation

`tasks/README.md`: T1-T8, one per part in the order above, and T9, the instructions. Contract §305-§312, one per part. One pull request per task or one for all, on one branch from main, after ADR-177 (#536) and ADR-178 (#541), which are merged; T5 lands after T1, T2, T4, T7 and T8; T9 lands last.

## Consequences

- **Positive:** each measured bucket gets a way to cost less, and every one of them keeps the rule that a line not shown is not licensed.
- **Negative:** five new read flags and one write flag to teach; the guide grows (ADR-171's bound moves, with its figure, in T9).
- **Neutral:** without a flag or the variable, a read and a write print what they print today, plus at most one hint line.

## Out of Scope

- `--outline` and `--no-comments` for languages other than Go (and Markdown for outline) (deferred: docs/adr/BACKLOG.md "From ADR-179")
- The terse receipt as the default (deferred: docs/adr/BACKLOG.md "Pre-registered for ADR-177, ADR-178 and ADR-179")
- `--if-served` that knows sessions (permanent: boundary: mrw has no session identity, ADR-038)
- These flags on `mrw_read` and `mrw_write` (deferred: docs/adr/BACKLOG.md "From ADR-179")
- A one-line answer to a repeated `mrw instructions` (deferred: docs/adr/BACKLOG.md "From ADR-171")
- Agents reading with `cat` and `grep` instead of mrw (permanent: boundary: adoption guidance, `docs/agents.md` §6, not a feature of this record)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| `--if-served` answers by reference for lines the model lost or never received | Med | Med | opt-in; the caller must quote the sha; the guide names the hook, the matcher and `--max-chars`; `mrw seen --forget` |
| `MRW_MAX_COLS` cuts a line the caller then tries to edit | Med | Low | the write refusal names the cut line and says to read it whole (ADR-154) |
| A folded or elided run hides the one line that differed | Low | Med | only identical or digit-only runs fold; directives and the cgo preamble are never elided; each line names its range |
| `--outline` ranges wrong for a file that does not parse | Low | Low | a parse error makes the read incomplete, exit 1 |
| The guide's cost grows past ADR-171's bound | Med | Low | T9 moves the bound with its figure, or drops text elsewhere |

## Rollback

Before a release: remove any part; each is independent. After a release: each flag and the variable is deprecated for one minor release before removal; the hint line can be removed at once; `same_as_served` stays listed (ADR-111). No state changes.

## Open Questions

- **(d) footer or no footer.** This record takes no footer, on the arithmetic above; the brief named one. Zy may overrule.
- **(f) default.** Flag now; the default is a question for the re-measure.
- **(b) and (h) beyond Go.** Unmeasured; T2 S2 and T8 S2 measure the share first.
- **(a) the 400-column threshold and the 200-column suggestion** are the study's numbers, not tuned.

## Follow-ups

- [ ] none at authoring
