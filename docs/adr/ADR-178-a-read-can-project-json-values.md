# ADR-178: A read can project JSON values instead of serving lines

**Status:** Accepted
**Accepted:** 2026-10-27 by Zy — *"yes, go, address these, measure, plan, implement verify, udupate the instrucitons, no counter-arguing, all follow same direction"*, and, after the plan was presented, *"go on, implement all of it"*.
**Date:** 2026-10-27
**Amended:** 2026-10-27, before acceptance, from a Codex challenge and a Claude adr-challenger pass, each finding checked against the code: an expression is checked for its `$` before the parser sees it (`parseJSONPath` assumes it, `internal/read/jsonpath.go:45`); a projection records nothing, since `--if-changed` reads the file on disk and needs no record (`internal/read/read.go:191-196`); standard input keeps its own rules (`cmd/mrw/stdin.go:35`, `:38`); ADR-179's flags are refused beside it; the existing ADR-155 tests run in the fence; the licensing warning goes in the recipe the skill prints. ADR-178 ships on one branch with ADR-179, apart from ADR-177 (tasks/README.md).
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-033, ADR-104, ADR-111, ADR-141, ADR-146, ADR-153, ADR-155, ADR-157, ADR-161, ADR-165, ADR-169, ADR-170, ADR-177, ADR-179, docs/bench/output-study/README.md, docs/agents.md, README.md, AGENTS.md
**Governs:** `internal/read/jsonpath.go`, `internal/read/read.go`, `cmd/mrw/main.go`, `cmd/mrw/stdin.go`, `internal/guide/guide.go`, `scripts/contract.sh`
**Enforced-by:** `internal/read/project178_test.go::TestAJSONPathProjectionPrintsCompactValues`
**Invalidates:** none — checked: a JSON path address (ADR-155) keeps serving and licensing the lines of its value; the new flag is separate and licenses nothing.
**Served-path change:** `mrw read --json-path '$.a.b,$.c[0]' FILE` (or `-` for standard input) prints each selected value as one line, `EXPR = <compact JSON>`, instead of the lines that hold it — including from single-line and piped JSON — and licenses no line.

## Context

Zy, 2026-10-27, on the output study: *"yes, go, address these, measure, plan, implement verify, udupate the instrucitons, no counter-arguing, all follow same direction."*

Measured by `docs/bench/output-study/measure_output.py` over the registered window (`--since 2026-09-27T10:11:00Z --until 2026-10-27T10:11:00Z`, 129 main-session transcripts, 100.9M characters of tool output; `docs/bench/output-study/baseline-2026-10-27.json`): **JSON documents of 1,000 characters or more were 6.19M characters in 1,020 calls**, 6.1% of all output — and **only 72K of it arrived through `mrw read`**. The motivating case is a peer session's loop over quality-harness `task-state --json`, about 60-70 KB per call through Bash, to read a handful of fields.

A JSON path address (ADR-155, `internal/read/jsonpath.go`) serves the LINES of one value. Over single-line JSON that is the whole document: `mrw read 'one.json:$.a'` of `{"a":{"b":[1,2]}}` serves line 1, all 18 bytes, and so does `mrw read -- '-:$.a.b' < one.json` (both probed 2026-10-27). Minified JSON and API output are single-line; for them a path address saves nothing.

Confounds: the saving exists only if agents route JSON through mrw, which today they do for about 1% of it; a projection pays only where the caller knows the paths it wants.

## Existing Primitives Audit

- **The path grammar** `parseJSONPath` (`internal/read/jsonpath.go:40`): `$`, `.name`, `[N]`, `["name"]`; `..`, `*`, filters and single quotes refused by name. It assumes its caller checked the prefix — it starts at `s[1:]` (`:45`), so an empty string panics and `x.a` parses as `$.a`; today's only caller checks with `isJSONPath` first (`internal/read/read.go:500`). Reuse it behind the same check.
- **The resolver** `jsonSpan(b, steps)` (`internal/read/jsonpath.go:115`): one JSON document, a leading BOM skipped, CRLF and CR-only line ends, a duplicated name on the path refused with its lines (ADR-155). It returns lines; T1 splits out the byte span it already computes, so the projection and the line address resolve through one function. Its tests (`internal/read/jsonpath155_test.go`, `cmd/mrw/jsonpath155_test.go`) run in T1's fence.
- **Standard input** `stdinRead` (`cmd/mrw/stdin.go:34`): `-` stands alone (`:35`), refuses `--json`, `--quiet`, `--count` and the rest by name (`:38-42`), builds its own `read.Options` (`:92-99`) and records nothing (ADR-161). The projection on `-` goes through it and keeps those rules.
- **`--if-changed PATH=SHA`** (ADR-169, `internal/read/read.go:191-196`, `:694`): compares the caller's sha with the file on disk, so it needs no ledger record to work after a projection.
- **The flag conflicts** (`cmd/mrw/main.go:1056-1127`): `--sub`, `--rewrite`, `--count`, `--quiet` and `--if-changed` each refuse the flags they cannot act beside, by name. Extend the same table.
- **The character budget** `read.Budget` and `Withheld` (`internal/read/read.go:184-190`, ADR-165): what does not fit is listed on stderr, never dropped silently. Reuse.

## Decision

**1. The flag and its grammar.** `mrw read --json-path EXPR[,EXPR…] PATH…`, PATH a file or `-`. EXPR is ADR-155's grammar, unchanged, and must begin `$.` or `$[` — checked before the parser runs, so an empty expression, `x.a` or `a` is a usage error, exit 2, naming it and the forms a step takes. A comma separates expressions only outside a `["…"]` step; inside one the step is a JSON string, so `\"` and `\\` are escapes and a comma is part of the name: `--json-path '$["a,b"],$["x\"y"],$.c'` is three expressions. The flag also repeats; expressions are taken in command-line order, then in order within each value.

**2. The output.** Per file, the header a read prints today (`==> PATH  NL  NB  sha X`), then one line per expression: `EXPR = VALUE`, VALUE the selected value with its insignificant whitespace removed (`json.Compact`, so every literal keeps its bytes). Nothing else of the file is printed.

**3. Licensing: none, and nothing recorded.** A projection licenses no line and records nothing in the ledger, for a named file as for `-`. A value is not a line, the ledger licenses lines (ADR-002), and a compacted value does not show the caller the bytes a plan would replace. `--if-changed` needs no record: it compares the sha the header printed with the file on disk (ADR-169).

**4. Misses and refusals.** An expression the document does not hold prints `EXPR  NOT FOUND: <the resolver's reason>`; a file that is not exactly one JSON document prints the header and `NOT JSON: <reason>` (a JSON Lines file of several values included). Either makes the read exit 1 with `mrw: incomplete: …` (ADR-146), as a range that could not be served does; the expressions that resolved are still printed.

**5. Beside other flags.** Refused, exit 2, each by name: `--grep`, `--ast-grep`, `--sub`, `--rewrite`, `--count`, `--stat`, `--quiet`, `-C`, `--max-lines`, `--if-changed`, a PATH carrying an address (`f.json:3-9`), and ADR-179's `--outline`, `--if-served`, `--fold`, `--max-matches` and `--no-comments` — each serves, folds or licenses lines a projection does not have. Whichever of the two records' tasks lands second adds the refusal for the other's flags, in both directions. Accepted: `--max-chars` (the budget bounds what is printed; a value that does not fit is withheld and listed on stderr with its expression, ADR-165, and the read exits 1), `--max-cols` (cuts a long projected line to its window, as any line; ADR-179's `MRW_MAX_COLS` default does not apply to a projection), `--files-from` and several named files, and `--json` for named files (the projection is `content`; the receipt's keys are unchanged and its `observed` is empty). `-` stands alone, as it does today, and refuses `--json` as standard input reads do (`cmd/mrw/stdin.go:38`).

**6. CLI only.** `mrw_read` gains no argument: the MCP tools are off by default (ADR-170) and their callers have no shell piping JSON to them. Deferred with a trigger.

**What would make this decision wrong:** agents keep piping JSON to `jq` or reading it whole. The pre-registered re-measure (`docs/adr/BACKLOG.md`, "Pre-registered for ADR-177, ADR-178 and ADR-179") measures the share of JSON documents of 1 KB or more in tool output, and counts `--json-path` calls; a share that does not move says the projection is not used.

## Alternatives Considered

- **Make the JSON path address itself project when the value is on one line.** Rejected: an address that licenses lines in one file and nothing in the next, by a property the caller cannot see, breaks ADR-155's promise that an address names a span.
- **Record the file hashed with no spans, as `--count` does.** The first draft, for `--if-changed`'s sake. Rejected after the challenge: `--if-changed` reads the file on disk and needs no record, and a record that licenses nothing is a ledger write for nothing.
- **`*` and `..` in the grammar.** Rejected for this record: one expression names one value, and ADR-155 refused both for that reason; the `task-state` loop reads named fields. Deferred with a trigger.
- **Shell out to `jq`.** Rejected: a dependency for twenty lines of the standard library, and `jq` already exists for callers who want it.
- **Pretty-printed values.** Rejected: a value of many lines is what the line address already serves.
- **A projection that licenses the value's lines.** Rejected: Decision 3.

## Component / Boundary Impact

- `internal/read` (engine package, owned here): the byte span split out of `jsonSpan`, the expression splitter, and the projection writer.
- `cmd/mrw`: the flag, its conflict rules in `main.go`, and the standard-input path in `stdin.go`.
- `internal/guide`: the flag's line in `CLI()` (T1, which `TestEveryReadFlagIsTaughtByInstructions` at `cmd/mrw/instructions_test.go:32` needs the moment the flag exists) and the recipe (T2).
No package changes ownership. This record holds `internal/check` byte-identical, which ADR-177 changes, so it ships with ADR-179 on a branch apart from ADR-177.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --json-path EXPR[,EXPR…]` | new flag; prints `EXPR = VALUE` lines; licenses and records nothing | `cmd/mrw`, `internal/read` | agents, scripts |
| `mrw read --json` | `content` may hold a projection; `observed` empty; no receipt key changes | `cmd/mrw` | adapters |
| `mrw instructions`, `mrw skill`, `.claude/skills/mrw/SKILL.md`, README, AGENTS.md, `docs/agents.md` | a recipe that says a projection licenses nothing | T2 (the `CLI()` line in T1) | agents |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `--json-path` and its output lines | T1 | T2 | No |

## Implementation

`tasks/README.md`: T1 (the projection, the `CLI()` line, contract §304), T2 (the instructions). One pull request, on the branch it shares with ADR-179.

## Consequences

- **Positive:** a 60 KB JSON answer read for five fields costs five lines.
- **Negative:** one more read mode whose output is not lines of the file; a caller who wants to edit the value reads it with the path address.
- **Neutral:** without the flag nothing changes.

## Out of Scope

- `*` and `..` in a projection (deferred: docs/adr/BACKLOG.md "From ADR-178")
- JSON Lines, one projection per line (deferred: docs/adr/BACKLOG.md "From ADR-178")
- `json_path` on `mrw_read` (deferred: docs/adr/BACKLOG.md "From ADR-178")
- Editing a value by path (permanent: boundary: a plan addresses lines, ADR-155)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller edits from a projected value and is refused | Med | Low | the refusal names the unread lines; the recipe says to read the path address before editing |
| A large value on one line floods the context | Med | Med | `--max-cols` and `--max-chars` act on it; the recipe shows `--max-cols` |
| An expression's quoting differs by shell | Med | Low | the recipe carries AGENTS.md's measured quoting for a JSON path (ADR-157's table: pwsh 7, Windows PowerShell 5.1, cmd, POSIX) |
| Numbers re-encoded differently | Low | Low | `json.Compact` keeps the bytes of every literal; no decode-and-re-encode |

## Rollback

Before a release: remove the flag. After: deprecate it for one minor release, then remove; no state changes.

## Open Questions

- **`*` over arrays.** The `task-state` loop may want `$.tasks[*].status`. Not decided here; the BACKLOG entry's trigger is a caller asking for it.

## Follow-ups

- [ ] none at authoring
