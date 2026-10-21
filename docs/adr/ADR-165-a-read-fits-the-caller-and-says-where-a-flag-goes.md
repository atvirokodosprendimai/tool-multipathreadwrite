# ADR-165: A read fits the caller's budget, answers as a document, and a misplaced global flag is pointed home

**Status:** Accepted
**Accepted:** 2026-10-21 by Zy — "good, fix, test, push, release, update the readmes, send info if necessary to the webpage agent", after the three-task draft was summarised to them; the reporter (a Codex agent, pasted by Zy) had answered "one small polish record with those three tasks makes sense … each should have a failing test and a documented contract … nonblocking". A Codex xhigh challenger (two parts, 2026-10-21) then changed three things before execution, each reported to Zy: the `--max-chars` WITHHELD list goes to standard error so standard output stays within N with no exception, and the budget covers every standard-output producer of the read action; the `--root` lookup is limited to flags the application defines, by long name, behind ADR-097's pointer and ahead of ADR-140's; `--json`'s `receipt` equals `mrw_read`'s `content[1]` only for reads `mrw_read` answers whole, and `content` is the CLI's report, quiet lines included.
**Date:** 2026-10-21
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-023, ADR-032, ADR-033, ADR-097, ADR-100, ADR-111, ADR-117, ADR-133, ADR-140, ADR-146, ADR-151, ADR-161, ADR-164
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/foreignflag.go`, `internal/read/read.go`, `internal/mcp/tools.go`, `docs/receipts.txt`
**Enforced-by:** None — three unrelated surfaces; each task's tests and contract row (§281–§283) hold its own decision
**Invalidates:** none — checked: ADR-032's ceiling stays the MCP server's and is not moved; ADR-023 decides `mrw_read`'s answer and says nothing about the CLI; ADR-117's rule that a cap licenses only what it served is kept by Decision 1
**Served-path change:** `mrw read --max-chars N` writes at most N characters to standard output and lists on standard error every range and file it did not serve; `mrw read --json` prints the plain read's report and the receipt `mrw_read` would send for it, as one JSON document; `mrw read --root DIR …` (`mrw`'s own flag after the subcommand) is refused with `--root goes before the subcommand: mrw --root DIR read …`.

## Context

**A field report**, 2026-10-21, from a Codex agent that had used mrw v1.69.1 in another project for a week, pasted by Zy. It found mrw useful ("grouped reads and guarded batch writes reduce tool calls"), claimed no speed or token number, and named three frictions; after reading `mrw read --help` it withdrew two as covered (`--max-lines`; `--grep --sub` and `--ast-grep --rewrite`) and kept two:

1. "a global byte/token budget across a read batch": its read batches exceeded its host's output limit, which cuts the answer where it likes, so lines mrw had licensed never reached the agent. `--max-lines` caps per spec (ADR-033, ADR-117); nothing on the CLI caps the whole answer. `mrw mcp` has that cap, `--max-result-chars` (ADR-032), because a host declares one; a shell host has one too and declares nothing.
2. "structured read output with content, ranges and SHA": its adapter parses mrw's report to find what was served and at which sha. `mrw_read` already answers that over MCP: the served text as `content[0]` and a receipt object as `content[1]` (ADR-023), with `observed` per file (ranges and sha), `problems`, `skipped` and `next_read`. The CLI has `--json` on `write`, `check` and `stats` (`grep -n 'Name: "json"' cmd/mrw/main.go`: three sites) and not on `read`.

And one it dropped but which, as it said, "did cost me a tool round trip": `mrw read --root DIR f` answers `flag provided but not defined: -root (see: mrw read --help)`, exit 2, and nothing says that `--root` is `mrw`'s own flag and goes before the subcommand. Measured on v1.69.1, 2026-10-21. ADR-097 points a flag that names a subcommand, ADR-140 a flag another tool has, ADR-151 the all-digit `-C`; a global flag typed after the subcommand has no pointer.

**Audit of the class** — *a place where mrw's answer costs the caller a turn it did not need: an answer the host cuts, an answer a program must re-parse, a refusal that does not say where the flag goes*. Enumerated 2026-10-21:
- refusals built from the parser's "not defined" text: `grep -n 'not defined' cmd/mrw/*.go` — two sites, `cmd/mrw/foreignflag.go` (`refusedFlag`) and `cmd/mrw/main.go` (`subcommandForFlag`), both reached from `usageError`; the global flags a caller can misplace are those of `mrw` itself, `mrw --help` (T1 enumerates them in its test, so a flag added later is covered);
- answer bounds: `grep -rn 'MaxResultChars\|MaxLines\|MaxCols' internal cmd --include='*.go' -l` — the MCP ceiling (`internal/mcp/schema.go`), the per-spec line cap and the per-line window (`internal/read/read.go`, `maxcols.go`); no call-wide bound on the CLI;
- JSON answers: `write`, `check`, `stats` on the CLI, `mrw_write`'s structured value and `mrw_read`'s `content[1]` over MCP; `read` on the CLI has none.

Three members, one each, and each is a task. Left out on purpose: `mrw mcp`'s bound, already ADR-032's; a token budget, which is a model's unit and not mrw's (Out of Scope).

Stage 5 (ADR-162) is unaffected: its harness pins `mrw-v1.68.0`.

## Existing Primitives Audit

- `usageError` (`cmd/mrw/main.go`) already composes every parser refusal and already carries two pointer tables (`subcommandForFlag`, `foreignFlagHint`); T1 adds a third lookup there, against `mrw`'s own flags, and no new parser.
- `read.Options.MaxLines` and the `WITHHELD` range line (`internal/read/read.go`, the budget loop) already express "stopped, and here is what you did not get"; T2 adds a second budget beside it in the same loop and reuses the same line form, so a caller reads one vocabulary.
- `readTool` (`internal/mcp/tools.go`) already composes the receipt as a map (`observed`, `problems`, `counts`, `skipped`, `next_read`) from `read.Run`'s `observed` and `problems`; T3 lifts that composition into a function both surfaces call, so the CLI document and the MCP receipt cannot disagree.
- `docs/receipts.txt` and `TestNoShippedReceiptFieldDisappears` (ADR-111) already hold every shipped JSON receipt; T3 appends the `read` receipt there.

## Decision

1. **A read takes a call-wide character budget (T2).** `mrw read --max-chars N` bounds everything the read writes to standard output — the served lines, their headers and ranges, and the notes and walk reports `cmd/mrw` prints around `read.Run` (`cmd/mrw/main.go`, the read action) — in characters, as `mrw mcp --max-result-chars` measures its result (ADR-032) and as `--max-cols` measures a line. One counting writer wraps standard output in `cmd/mrw` and every producer writes through it. Absent is no cap; `0` is a cap of zero (ADR-033); negative is refused as `--max-lines` is. When the next line would pass N, serving stops at the last whole line that fit; no line is cut, since a cut line is a line not read (ADR-137). Standard output then holds at most N characters, with no exception. **Every range and file not served is listed on standard error**, one `WITHHELD` line each in the form `--max-lines` uses (`f.go: lines A-B WITHHELD (n line(s)): --max-chars reached`, `g.go: WITHHELD: --max-chars reached`), before the last line, which begins `mrw: incomplete: `, keeps ADR-146's `N range(s) could not be served` and adds `; --max-chars N reached`; exit 1 (ADR-146). Standard error is not charged, because a budget that silences its own report is ADR-032's elided-receipt failure on the CLI, and the list is bounded by the specs and matches, which the `--grep` walk has already collected before serving begins (`internal/read/walk.go`, `Walk` returns its specs first), so naming a file the budget excluded opens nothing. Only the lines printed are licensed (ADR-117, ADR-133). Not with `--sub`, `--rewrite` (a plan applies whole or not at all, and a plan too large is refused whole, ADR-161), `--count` or `--stat` (they serve no line); with `--quiet` the budget bounds the headers and the `-- licensed` line, which is what a quiet read prints. `mrw_read` is unchanged: its ceiling is ADR-032's.
   Measured while building it, 2026-10-21: a budget one character short of a whole `--grep` walk dropped only the walk's own last line (the skip line, printed after `read.Run`) and served every range, so the read counted no problem and exited 0 with a cut answer. A budget that dropped anything therefore ends the read `incomplete:` with exit 1, whether or not a range was withheld.
   After the go-reviewer and Codex reviews of a043d346 (2026-10-21), three more rules, each a way part of the answer was cut and unsaid: every report line the budget drops — a REFUSED or UNREADABLE header, a `--max-lines` notice, a missed pattern, the quiet `-- licensed` line — is also written to the withheld list (`read.Reporting`), prefixed with its path where it names none; the list is written to standard error after the answer is flushed, so streams merged by a host keep the answer whole and the list after it; and `mrw read -` takes the budget as a file does. A budget with nothing left counts as reached, so a file met then is named without being opened. The engine takes the budget as a `read.Budget` passed as its writer; there is no `Options` field for it.
2. **A read answers as a document (T3).** `mrw read --json` prints one JSON object, `{"content": …, "receipt": …}`. `content` is the plain read's standard output exactly as the same read without `--json` prints it (numbers, `==>` headers, `@@` ranges, `--` notes; under `--quiet`, the headers and the `-- licensed` line). It is not `mrw_read`'s `content[0]`, which adds checkpoint markers and an acknowledgement footer (`internal/mcp/ack.go`) and words a walk's report differently; a caller who wants MCP's text uses MCP. `receipt` is built by the same exported function `readTool` uses for `content[1]` (`internal/mcp`, which `cmd/mrw` already imports, so no package moves): `observed` (per file, its ranges and sha, paths with `/`), `problems`, and `counts`, `skipped` and `matches` where a read of that kind has them. For a read `mrw_read` answers whole — not paged (ADR-014) and not as an index (ADR-017) — the CLI's `receipt` equals `content[1]` parsed; `next_read`, `index` and `next_index` never appear on the CLI, which neither pages nor indexes. Exit codes are the plain read's (ADR-146), and stderr is the plain read's (the `--max-chars` WITHHELD list included). A refusal before any file is served is `{"error": "<the plain read's stderr line>"}` on standard output, as `mrw check --json` refuses (ADR-100), exit unchanged. Licensing is the plain read's. Not with `--sub` or `--rewrite`, which print a plan. The `read` receipt's keys are appended to `docs/receipts.txt` (ADR-111) in the task's commit.
   After the Codex review of 0f92edf0 (2026-10-21): the document is checked before anything is recorded — content that is not valid UTF-8 is refused, exit 2, since JSON would change those bytes while their lines were licensed (as `mrw_read` leaves them without a checkpoint, ADR-108), and a document that could not be written records nothing; `--json` is refused on standard input, which has no receipt; under `--max-chars` the budget bounds `content`, and the JSON around it and the receipt are not charged.
3. **`mrw`'s own flag typed after the subcommand is pointed home (T1).** When the parser refuses a flag of a subcommand as not defined, and its name is the long name of a flag the application defines on the root command (`--root` today, `cmd/mrw/main.go`; urfave's added `--help` and `--version` are not the application's and are left out), `usageError` ends the refusal with `; --root goes before the subcommand: mrw --root DIR read …` — the flag's long name, a value placeholder from its own definition, and the subcommand the caller used. Exit 2, as today (ADR-097). A root flag's short alias that the subcommand defines for itself (`-C`, which `read` uses for context) is the subcommand's and never reaches `usageError`; an alias the subcommand lacks is pointed home like the long name. The existing pointers keep precedence where they fire today: ADR-097's subcommand-named flag first, then this lookup, then `foreignFlags` (ADR-140), whose `-v` and `-o` entries (ADR-164) are not root flags and are unchanged. The lookup reads `cmd.Root().Flags` at run time, so a root flag the application adds later is pointed home without a table entry.
   Measured on the built binary, 2026-10-21: urfave makes a root flag persistent, so every subcommand but `read` already takes `--root` after the verb; `read` refuses it because its own `-C` (context) clashes with `--root`'s alias and urfave then skips the root flag whole (`flagKinds`, `cmd/mrw/main.go`). The pointer therefore fires on `read` today, and on any subcommand a later clash makes refuse a root flag.

**What would falsify each decision as shipped:** D1 — a read with `--max-chars N` whose standard output exceeds N characters, or that licenses a line it did not print; D2 — a `--json` document whose `receipt` differs from `mrw_read`'s `content[1]` for the same specs on the same tree; D3 — `mrw read --root DIR f` exiting without the pointer, or `mrw --root DIR read f` changing in any way. Each is a contract row's bad case.

## Alternatives Considered

- **Count bytes, not characters (`--max-bytes`):** a host's limit is sometimes bytes. Rejected: mrw's two existing bounds (`--max-result-chars`, `--max-cols`) count characters, and a third unit for the same question is the kind of drift ADR-164 T7 cleaned up; a byte count is derivable from the characters served, the reverse is not.
- **A budget that cuts the last line to fit:** serves more. Rejected: a cut line is a line not read (ADR-137), and a budget whose last line licenses nothing is a trap a caller cannot see.
- **`--json` as lines in an array (`[{n, text}]`) instead of the plain text:** easier to index. Rejected: `mrw_read`'s answer is the served text (ADR-023), the receipt already carries the ranges and sha, and a second line encoding is a second thing to keep equal.
- **A table of global flags for the pointer, as `foreignFlags` is:** matches ADR-140. Rejected: the root command already knows its flags; a table is a copy that goes stale when a global flag is added.
- **Leave all three to the adapter:** the reporter did adapt. Rejected by Zy, 2026-10-21: "one small polish record with those three tasks makes sense … each should have a failing test and a documented contract."

## Component / Boundary Impact

No module is added or moved. `internal/read` (an engine package, lifecycle §6) gains one option and one report line; `internal/mcp` lends its receipt composer to `cmd/mrw`; `cmd/mrw` gains two read flags and one refusal pointer. No boundary crosses: the receipt composer moves from `internal/mcp/tools.go` into a function that takes `read.Run`'s outputs and returns a map, and `internal/mcp` keeps calling it. No architecture document exists; no Module Map to update.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --max-chars N` | new flag; `WITHHELD … --max-chars reached` lines; exit 1 `incomplete:` when reached | `cmd/mrw/main.go`, `internal/read/read.go` | shell callers; `mrw instructions` recipe 7's neighbour |
| `mrw read --json` | new flag; a `read` receipt (`content`, `receipt{observed, problems, counts, skipped, matches}`) | `cmd/mrw/main.go`, `internal/mcp` (the shared composer) | adapters; `docs/receipts.txt` (ADR-111) |
| `usageError` | a misplaced global flag ends with its home spelling | `cmd/mrw/main.go` | every subcommand's refusal |
| `README.md`, `AGENTS.md` §Using mrw, `mrw instructions` | the two flags and the pointer, one sentence each | each task | callers |
| `scripts/contract.sh` §281–§283 | one row per decision, each with its pair | each task | CI |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `read.Budget` and the `--max-chars` WITHHELD lines | T2 | T3 (the `--json` document carries them in `content` and in `problems`) | No — T3's test includes one budgeted read |
| the receipt composer shared by `readTool` and `read --json` | T3 | none | No |

## Implementation

`tasks/README.md`; `tasks/T1-a-misplaced-global-flag-is-pointed-home.md`, `tasks/T2-a-read-fits-a-character-budget.md`, `tasks/T3-a-read-answers-as-a-document.md`. T1 and T2 are independent; T3 consumes T2's lines. Each task: red test, code, a killed mutant per proof step, a contract row with its pair, the gates in AGENTS.md §Build and check, a PR with a Codex review; T2 touches `internal/read`, so its PR names this record as owner and gets a `go-reviewer` pass first (lifecycle §6).

## Consequences

- **Positive:** a shell host's output limit stops cutting what mrw licensed; an adapter reads one document instead of parsing a report; a misplaced `--root` costs no turn.
- **Negative:** two more read flags to teach; `internal/read`'s budget loop carries two budgets; a receipt shape becomes a CLI contract under ADR-111.
- **Neutral:** nothing changes for a caller who passes no new flag; `mrw_read` is unchanged.

## Out of Scope

- A token budget (`--max-tokens`): tokens are a model's unit, differ per model, and mrw has no tokenizer; characters are what every host can count (permanent: boundary: mrw counts what it writes, not what a model reads)
- Changing `mrw mcp`'s ceiling or its elision (permanent: fact: the MCP ceiling is a caller-set budget on the encoded result; citation: file `docs/adr/ADR-032-the-ceiling-is-the-callers.md:1`)
- A `--root` read from an environment variable or a config file (permanent: boundary: a root nobody named is not a root, ADR-018's rule for the CLI as for MCP)
- Paging on the CLI (`next_read` in the document): a shell caller re-issues the specs it did not get, which the WITHHELD lines name (permanent: boundary: the CLI has no session to hold a cursor, and a page is ADR-014's answer to a host that has one)
- The reporter's third friction, verbose whole-file plans: withdrawn by the reporter as its own adapter's usage after it found `--grep --sub` (permanent: fact: `--sub` and `--rewrite` make narrow plans; citation: file `docs/adr/ADR-161-instructions-lead-with-recipes.md:1`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The character count and the host's count disagree (a host counts bytes, or counts its own wrapper) | Med | Low | the budget is the caller's to set below the host's; the README says it bounds mrw's own standard output and nothing a host adds |
| The receipt composer, once shared, is changed for the CLI and silently changes `mrw_read` | Low | Med | `TestTheReadReceiptMatchesItsSchema` (`internal/mcp/schema_test.go`) already holds `mrw_read`'s receipt; T3's test holds the CLI's `receipt` equal to `content[1]` for unpaged reads on the same tree |
| A global flag added later is not pointed home | Low | Low | T1's test enumerates the root command's flags at run time and tries each after a subcommand |
| A budget reached inside a `--grep` walk hides matches in files never opened | Med | Med | files the walk did not reach are reported `WITHHELD`, by name, and the read exits 1 `incomplete:` |

## Rollback

Each flag is removable before the release that ships it; after that, `--max-chars` and `--json` on `read` are contracts (ADR-111 for the receipt's keys) and a removal waits for a major version. The pointer is a message and carries no contract beyond its contract row. No persistent state changes.

## Follow-ups

- [ ] `mrw instructions` recipe 7 (`--quiet`) gets a sibling for `--max-chars` if callers reach for it; measured after one month of `mrw stats` on this checkout.
