# ADR-161: mrw writes the plan itself, and its instructions lead with recipes that keep the agent's context clean

**Status:** Accepted
**Accepted:** 2026-10-16 by Zy — "accept, go on", then, on how mrw should make the plan, "Both, plus ast-grep rewrite" and "Fold into ADR-161", before any code.
**Date:** 2026-10-16
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-037, ADR-052, ADR-054, ADR-060, ADR-062, ADR-063, ADR-069, ADR-070, ADR-071, ADR-084, ADR-092, ADR-095, ADR-122, ADR-133, ADR-136, ADR-141, ADR-150, ADR-153, ADR-155, ADR-156, ADR-160
**Governs:** `internal/read/**`, `internal/seen/**`, `internal/mcp/**`, `cmd/mrw/**`, `internal/guide/**`, `AGENTS.md`, `README.md`, `BESTPRACTICES.md`, `.claude/skills/mrw/SKILL.md`, `.claude/agents/*.md`, `scripts/contract.sh`, `docs/receipts.txt`
**Enforced-by:** None — until T1 lands, which adds `internal/read/sub161_test.go::TestASubReadPrintsAPlanAndLicensesItsLines`
**Invalidates:** ADR-136's neighbour note (`-- note: a multi-line replace of A-B needs a served line after B`) — Decision 10 serves that line instead of naming it. Amends ADR-153 (a plain `--grep` serves each match and the line after it; `--count` counts the matching lines) and ADR-155 (a JSON path serves and licenses its value's lines and the line after them), chosen by Zy on 2026-10-16 when T7 met them; ADR-150's ledger format moves to v6 (T1). ADR-069 T8's lone `-` that ended the parse now moves last, so the flags after it are read (T8, Zy: "Honor them"); its fence's §137 PASS line is replaced. Otherwise none — checked: ADR-037's five shared sentences stay first and verbatim; every phrase the text's existing tests and rows pin stays — ADR-062, ADR-063, ADR-069, ADR-071 (`internal/guide/guide_test.go`), ADR-070 (`bodyheader_test.go`, contract §132), ADR-084 (`teach084_test.go`, §166), ADR-092 (`teach092_test.go`, the `--then-sh` caveat verbatim), ADR-094 and ADR-095 (`teach094_test.go`, `teach095_test.go`, `then092t5_test.go`), §115 (every `read --help` flag named, no unknown flag), §117 (the `rg` pipeline verbatim), §§140 and 242; ADR-141's `--core` stays eight rules under 300 words naming only real flags, its rule 1 keeping the phrases `internal/guide/core141_test.go:18-33` pins, and the plain text stays over 900 words; ADR-052's multi-line licence rule holds for every generated hunk and is easier to meet (Decision 10); ADR-118 holds: a generated licence is not a shown line (Decisions 1 and 3); ADR-133's null-device rule is unchanged
**Served-path change:** new flags `mrw read --sub`, `mrw write --grep --sub`, `--rewrite` on `read --ast-grep` and `write --ast-grep`, `mrw read --quiet`; `mrw read -` reads standard input; a ranged read serves the line after each range; new text from `mrw instructions` and `--core`.

## Context
Stage 4 of the bench (ADR-160), Sonnet 5.5, own tasks, mrw v1.66.0 (`docs/bench/stage-4-sonnet/` at `921ef645` on `adr-160-t4-stage-4`, which merges before this record's T5; `go run ./cmd/bench analyze docs/bench`, 2026-10-16): LOSES, cost (B−A)/A +0.171 [+0.096, +0.247]. On the ten `wide-rename` tasks B cost +0.328 with the same 20 of 20 solved: 3.8 tool calls against A's 2.6, 1.9k output tokens against 0.6k, 131k cached tokens re-read against 87k. `mrw instructions` at `8c4b3f31` (v1.67.0's text) is 40 lines and 1,422 words.
Stage 4 of the bench (ADR-160), Sonnet 5.5, own tasks, mrw v1.66.0 (`docs/bench/stage-4-sonnet/`, `go run ./cmd/bench analyze docs/bench`, 2026-10-16): LOSES, cost (B−A)/A +0.171 [+0.096, +0.247]. On the ten `wide-rename` tasks B cost +0.328 with the same 20 of 20 solved: 3.8 tool calls against A's 2.6, 1.9k output tokens against 0.6k, 131k cached tokens re-read against 87k.

The transcripts show why. A renamed with one command (`grep -rlw X | xargs perl -pi -e 's/\bX\b/Y/g'`). B read every site into its context with `mrw read --grep`, then wrote a script over that output across several turns, then wrote the plan. Configuration B is told only what `mrw instructions` prints (`internal/bench/runner.go:696`), and mrw has no way to compute a new line from an old one: a plan is text the caller supplies, so a wide edit either passes every line through the agent's context or through a script the agent writes.

Measured 2026-10-16 in a scratch tree (40 Go files, 81 sites): `mrw read --grep '\bOldName\b' . | perl -ne '…s/\bOldName\b/NewName/g…' | mrw write -` changed all 81 sites in one command, leaving `OldNameX` and `OldName1`; the same pipe through BSD awk, which has no word boundary, replaced every line with itself and mrw reported 81 hunks applied. The generator is the fragile part, and it is what mrw can do itself.

Zy, 2026-10-16: "need to re-teach the agents to use mrw efficiently, no prose"; "Recipes first, prose cut"; "we have to teach agents how to avoid context polluting while using mrw … why you used perl to write an mrw plan? can't it be written with mrw itself? maybe mrw even can help with static tools to write the plan properly?"; "Both, plus ast-grep rewrite"; "Fold into ADR-161".

## Decision

1. **`mrw read --grep PATTERN --sub REPLACEMENT [paths]` prints a plan instead of lines.** For each line the walk serves, the line's text (without its ending) is rewritten with Go's `regexp.ReplaceAllString` (`$1` and `${name}` expand, `$$` is a dollar); a line the rewrite leaves unchanged is not planned. Each changed line is one hunk: `@@ PATH N replace sha=<the file's sha256 the plan was computed from> body=1 raw=true` and the new text, so a body starting `@@`, `body=` or a byte order mark is never read as a header, and a plan computed from one version of a file is refused against any other. A PATH with whitespace or quotes is double-quoted, a quote or backslash inside escaped, as the plan grammar reads it (`splitHeader`, `internal/plan/plan.go:575`); one starting with `@` is spelled `./@…`, so it is never read as a working-set pointer. A new text ending in a CR — a line of a file whose endings are mixed — is refused and nothing is planned: a plan line cannot carry it. A REPLACEMENT holding a CR or LF is a usage error: the rewrite is one line for one line. Stdout holds only the plan; stderr holds `-- planned: N site(s) in M file(s)` and the `-- skipped:` footer. The read records each planned line as licensed and not shown (ADR-118's `Written` with an empty `Shown`, the licence narrowed to the planned lines: the ledger line is `w<shown>|<licensed>`, in ledger format v6, so an mrw that reads v5 (v1.66.0, v1.67.0) leaves the ledger untouched and licenses nothing from it (ADR-150) rather than take that line for a path; v4 and v5 ledgers are still read, holding no such line), so a plan may replace it and `occurrence=N` counts none of it. Exit 0 with one hunk or more; exit 1 `no match:` when no line matched and `incomplete:` when lines matched and none changed. Usage error (exit 2): `--sub` without `--grep`, or beside `--ast-grep`, `-C`, `--count`, `--stat`, `--quiet`, `--max-cols`, `--max-lines`, `--no-numbers` or `--files-from`; a named path with a range.
2. **`mrw write --grep PATTERN --sub REPLACEMENT [paths]` builds that plan and applies it in one command**, as one plan: all or nothing, a verdict per hunk, the project's check and `--then`/`--then-sh` as for any write, `--dry-run` and `--json` as for any write, a refusal under `--json` one document that keeps its exit; `--exclude` and `--no-ignore` shape the walk as `read --grep`'s do; no PLAN argument. A CRLF file keeps its line endings. A file the walk refused or could not read is never left out quietly: the write applies nothing and exits 1 naming it, so a partial edit never exits 0 (the challenge of this record: a pipe through a script applied the readable files and exited 0). A binary file the walk skips is outside the edit set by design — mrw does not edit bytes as lines — and stderr carries the skipped count.
3. **`--rewrite REPLACEMENT` with `--ast-grep PATTERN`** does the same from ast-grep's own rewrite: on `read` it prints a plan, on `write` it applies it. From ast-grep's `--json` it takes each match's `range.byteOffset` (UTF-8 bytes, end exclusive) and its `replacement` (`""` is a replacement; a missing field, or a range outside the file, is refused, never guessed). The file is read once and its sha checked against the bytes ast-grep was given; matches whose byte ranges overlap — nested matches, such as `foo(foo(a, b), c)` for `foo($A, $B)` (ast-grep 0.45.3 reports [0,17) and [4,13), measured by the Codex challenge of this record) — are refused by name, and nothing is printed or written. The rest are applied to the bytes; the changed lines are grouped into whole-line spans per file, spans that touch merged, and each span is one `replace` hunk with `sha=`, `body=N raw=true` and `anchor=` its first line, quoted and escaped as for a path. The read records each span and the line after it as licensed and not shown (ADR-052, ADR-118). ADR-122's ignore and refusal rules apply as to `--ast-grep` today; with no ast-grep on PATH it exits 2 naming it.
4. **`mrw instructions` prints, right after ADR-037's five shared sentences, a block headed `Recipes:`** — each recipe a numbered label line ending in `:` and, on the next line, the command indented by two spaces; no prose:
   1. the same change at many sites, nothing in your context: `mrw write --grep '\bOld\b' --sub 'New' .`
   2. see what it will do first, nothing written: `mrw write --grep '\bOld\b' --sub 'New' --dry-run .`
   3. a change by syntax, not text: `mrw write --ast-grep 'foo($A, $B)' --rewrite 'bar($B, $A)' .`
   4. look before you change, only at what you will use: `mrw read --grep P --count . && mrw read 'f.go:/P/' -C 3 --max-lines 40`
   5. different changes at several sites, one read of all of them, then one plan on standard input: `mrw read 'a.go:/func A/,+12' b.go:40-60 && printf '%s\n' "$plan" | mrw write -`
   6. a step after the write, in the same call: `mrw write - --then NAME` (a step from `.quality-harness.json`) — the project's check already runs after a code write; the label carries ADR-092's `--then-sh` caveat. Every command line is commands only, joined by `&&` or `|`; the Codex review of T3 found two that held prose and exited 2 as printed.
   7. license lines you already know, without them in your context: `mrw read --quiet f.go:40-60` (a read into `/dev/null` licenses nothing); a file mrw just wrote needs no re-read
   8. a file of long lines (logs, JSONL): `mrw read --grep P --max-cols 200 f.jsonl` — each line cut to a window round the match
   9. narrow any command's output without leaving mrw: `cmd | mrw read --grep P --max-lines 20 -`
   The reference below keeps every sentence earlier records pin and stays over 900 words; this record moves it below the recipes and does not promise to shorten it.
5. **`--core` rule 1 names `mrw write --grep P --sub R`**, keeping the phrases `internal/guide/core141_test.go` pins ("every file read, edit and create", "one read of every site", "mrw write -"); other rules shortened so the eight stay under 300 words.
6. **Recipes are tested by running them**: contract rows take recipes 1 and 3 from `$MRW instructions` and run them in a scratch tree, recipe 3 against a fake ast-grep that answers with recorded `--json` output, as the existing `--ast-grep` rows do.
7. **Every surface that teaches mrw carries the recipes or inherits them.** The class, enumerated 2026-10-16 with `mrw read --grep 'mrw write -|mrw instructions|files-from -' --count --exclude docs --exclude '*_test.go' --exclude scripts .` (18 files): `AGENTS.md` "Using mrw" opens with them; `README.md`, `BESTPRACTICES.md` and the tracked `.claude/skills/mrw/SKILL.md` point at them; `.claude/agents/{adr-challenger,go-implementer,go-reviewer,scout}.md` carry recipe 1 in their mrw block; the opencode plugin inherits them through `mrw instructions` (`cmd/opencode/mrw-plugin/src/index.ts:290-299`); the MCP handshake is deferred with the MCP arguments. At release the centralised skill is updated in memoryd's `craft` (the store of record on this Mac) and mirrored to agentsmemory as earlier releases did, both compared byte for byte.
8. **A paired B pilot before Stage 4 runs again**: the four `wide-rename` calibration tasks of calibration-2 (`am-wide-add-input`, `am-wide-migrations`, `am-wide-entry-room`, `am-wide-with-reranker`) and the four `check-coupled` calibration tasks of calibration-3 (`am-coupled-installer-bash`, `am-coupled-doctor-discarded`, `am-coupled-doctor-duplicated`, `am-coupled-agentkit-commands`) × B × 2 on Sonnet, twice: once with v1.66.0, once with a build of this record, stage `pilot`, and each beside the calibration A receipts of the same tasks. It measures the generators and the recipes together and cannot tell them apart; it chooses nothing. Rerunning Stage 4 on the release is ADR-160's: its T4 is amended by the executor of this record's T5, before the rerun (Zy: "Rerun Stage 4 on the new mrw").
9. **`mrw read --quiet SPECS` licenses what the read serves and prints no line of it**: each file's header, then `-- licensed N line(s) in M file(s)`; the ledger records exactly what the same read without `--quiet` would. It exists because the read before a plan is often of lines the agent already knows, and the two ways to keep them out of its context today are a redirect to a file, which ADR-133 already counts as a read, and a redirect to `/dev/null`, which licenses nothing — the slip this record's executor made again and again while writing it (Zy: "this mrw tool had to alleviate you and increase your productivity, not to stop you on every step"). `--quiet` makes the first explicit and leaves no scratch file. Usage error beside `--count`, `--stat` or `--sub`, which serve or license differently; exits as a read.
10. **A ranged read serves the line after each range that ends before the file does**, numbered like the rest and licensed, so a multi-line replace of exactly the lines read is not refused for the line below it (ADR-052's rule, unchanged). ADR-136 printed a note naming that line instead; this record's executor met the refusal about five times while writing it, each time on lines just read. A whole-file read, `$`, and a range already ending on the last line are unchanged; under `--max-lines` the extra line counts against the cap; `--quiet` licenses it without printing it; a `--grep` match is a range of one line and gets its next line too. It is the CLI's: `mrw_read` keeps ADR-136's note until T9, since its pages are ranges and a page must not repeat a line. Zy chose it whole on 2026-10-16 ("As written, amend others"), so it amends ADR-153 — a plain `--grep` serves each matching line and the line after it, and `--count` still counts the matching lines — and ADR-155 — a JSON path read serves the value's lines and the line after them, and licenses both; tests and contract rows that pinned the old exact output (ADR-026, ADR-036, ADR-052, ADR-060 T2, ADR-065, ADR-136, ADR-148) were updated in the same change.
11. **`mrw read -` reads standard input** as one unnamed file — `cmd | mrw read --grep P --max-lines 20 -` — with every address, `--grep`, `-C`, `--max-lines` and `--max-cols`, so a command's output is narrowed by mrw and not by `grep`, `sed -n` or `head` (the other slip of this record's executor). It licenses nothing: it is not a file. A file named `-` is `./-`. Usage error beside `--sub`, `--quiet`, `--stat`, `--count`, `--files-from` or a second path. And a lone `-` moves last before the arguments are parsed, on every subcommand: urfave/cli stops at a lone `-` and drops what follows, flags included, so `printf plan | mrw write - --dry-run` wrote the plan and `mrw write - --then vet` ran no step (found by T8's executor, 2026-10-16); Zy chose to honour those flags rather than refuse them ("Honor them"), which amends ADR-069 T8 — its §137 row pinned the drop as "not refused", and a padded token after a lone `-` is now judged as one before it is. The `-` moves only when flags alone follow it; a path, a second `-` or a `--` after it is refused by name, since moving it would reorder positionals (the Codex review of T8). A terminal on standard input is refused rather than waited on.
12. **The MCP surface takes the same tools**: `mrw_read` gains `sub`, `rewrite` and `quiet`; `mrw_write` gains `grep`+`sub` and `ast_grep`+`rewrite` as a plan source in place of `plan`, with the CLI's refusals and licences; a served page carries the line after each range inside its checkpoint, counted (ADR-108); the handshake text carries the recipes in their MCP form. Zy, 2026-10-16: "do let us not leave anything dangling out of this adr" — nothing of this record is deferred to the BACKLOG.
13. **A plan or a read piped from PowerShell loses PowerShell's trailing CRLF.** PowerShell appends CRLF to what it pipes, so a plan whose lines end in LF arrived with a last line holding only CR, and the write added a blank line to the file (`printf '@@ f 1 replace\nA\n\r\n' | mrw write -` wrote `A\n\n`; found on Windows by a peer session probing T8, 2026-10-16, and reproduced on macOS). ADR-142 already drops that CRLF for `write --create`; `mrw write -` (any `--format`) and `mrw read -` now drop it by the same rule — a final CRLF after text that holds an LF and no other CR — and say so once on stderr. An all-CRLF input ending in an empty line is kept, as ADR-142 keeps it.

## Alternatives Considered

- **Teach a perl pipe and keep mrw as it is.** Rejected by Zy: the plan's generator is the fragile part (BSD awk's missing `\b` wrote nothing and reported success), and it is the part mrw can own.
- **Only `write --sub`.** Rejected by Zy ("Both"): `read --sub` lets the plan be seen, kept or piped before anything is written.
- **gopls rename for Go identifiers.** Not taken: one language, a second requirement in `go.mod` (one is the rule), and `--rewrite` covers syntax-aware edits in every language ast-grep knows.

## Existing Primitives Audit

- **Reused unchanged:** the `--grep` walk and its ignore and binary rules (ADR-116, ADR-130), the `--ast-grep` runner and its bounds (ADR-104, ADR-122), the plan parser and `Apply` (all or nothing, verdicts, ADR-001), the ledger (ADR-002, ADR-052), the write lock and check (ADR-054, ADR-075), `--then`.
- **Reshaped:** `read` gains a plan-printing mode; `write` gains a plan source besides PLAN and stdin.
- **New:** the substitution and the span grouping; the recipes text.

## Component / Boundary Impact

`internal/read` builds the plan text for both commands; `cmd/mrw` parses the flags and, for `write`, hands the plan to the existing apply path. No new package.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --grep P --sub R` | new: prints a plan, licenses its lines | T1 | agents, `mrw write -` |
| `mrw write --grep P --sub R` | new: applies that plan in one command | T1 | agents |
| the read ledger (`internal/seen`) | a line `w<shown>\|<licensed>` for lines a `--sub` plan licensed and no read showed; the ledger format becomes v6, so an mrw reading v5 leaves the ledger untouched (ADR-150) instead of taking that line for a path it could keep or write back as a whole-file licence (Codex review of T1) | T1 | every write's licence check |
| `--rewrite R` with `--ast-grep P`, on read and write | new | T2 | agents |
| `mrw read --quiet` | new: licenses, prints headers and a count | T6 | agents |
| a ranged read | serves the line after each range; ADR-136's note replaced | T7 | agents, any plan of exactly the read lines |
| `mrw read -` | new: standard input as one unnamed file, licensing nothing | T8 | agents narrowing a command's output |
| `mrw write -`, `mrw read -` | PowerShell's trailing CRLF dropped by ADR-142's rule, and named on stderr | T10 | Windows agents piping from PowerShell |
| `mrw_read` `sub`, `rewrite`, `quiet`; `mrw_write` `grep`+`sub`, `ast_grep`+`rewrite`; the handshake's recipes | new | T9 | MCP hosts |
| `mrw instructions`, `--core` | the `Recipes:` block; core rule 1 | T3 | agents, configuration B (ADR-156) |
| `docs/receipts.txt` | any key the write receipt gains for its plan source, appended (ADR-111) | T1 | callers of `--json` |
| `AGENTS.md`, the `mrw` skill | open with the recipes | T4 | agents |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `read --sub`, `write --sub` | T1 | T2, T3, T5 | No — new flags |
| `--rewrite` | T2 | T3, T5 | No — new flag |
| the `Recipes:` text | T3 | T4, T5 | No — text |

## Implementation

See `tasks/README.md`: T1 `--sub`, T2 `--rewrite`, T6 `--quiet`, T7 the line after a range, T8 `read -`, T10 PowerShell's CRLF on standard input, T3 the recipes, T9 the MCP surface, T4 the mirrors, T5 the B pilot.

## Consequences

- **Positive:** a wide edit is one command with nothing in the agent's context and no script to get wrong; a structural one too, where ast-grep is installed.
- **Negative:** two flags and a plan-printing mode to keep correct; `--rewrite` depends on ast-grep's JSON.
- **Neutral:** whether B's cost falls is measured, not assumed.

## Out of Scope

- A multi-line regular-expression substitution (permanent: boundary: `--sub` rewrites line by line, as `sed` does; a change across lines is `--rewrite` or a plan)
- Reading standard input over MCP (permanent: boundary: `mrw_read` has no standard input to give; its caller already holds the text)
- The Stage 4 rerun itself (permanent: boundary: it is ADR-160's stage; this record's T5 amends ADR-160's T4 to register it)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A replacement that rewrites more than meant (a pattern without `\b`) | Med | Med | `read --sub` shows the plan first; the receipt names every site; recipe 1 shows `\b` |
| ast-grep's JSON changes shape | Low | Med | the fake used by the tests is recorded output; a missing field is refused, not guessed |
| A recipe stops working against the binary | Low | High | contract rows run them |

## Rollback

Remove the flags and the text; no state or receipt key depends on them beyond what `docs/receipts.txt` records. The ledger line `w<shown>|<licensed>` a `--sub` read leaves is in a v6 ledger, which v1.66.0 and v1.67.0 leave untouched and license nothing from, naming the newer mrw (ADR-150); rolling back costs one re-read of each file.

## Follow-ups

None yet.
