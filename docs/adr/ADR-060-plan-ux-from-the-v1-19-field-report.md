# ADR-060: Five field-report frictions on the served path

**Status:** Accepted
**Accepted:** 2026-09-14 by M — *"all. we have to address these"*
**Date:** 2026-09-14
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-015, ADR-027, ADR-040, ADR-052, ADR-054, docs/adr/BACKLOG.md
**Governs:** `internal/plan/plan.go`, `internal/read/read.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/guide/guide.go`, `scripts/contract.sh` (§100–§104)
**Enforced-by:** `internal/plan/plan_test.go::TestASatisfiedBodyCountNamesTheExtraLines`
**Invalidates:** none — checked. ADR-040's unquoted-until-next-key still holds for `anchor=func openTestStore body=1`. ADR-052's neighbour licence is unchanged; this record only hints. ADR-003 still reads the process exit; this record only prints the check's last line. ADR-027 `body=0` is unchanged; `body=@path` is another way to declare a counted body.
**Served-path change:** a leftover `body=` names declared vs extra lines and `--dry-run` prints parsed hunk body counts; a ranged `read` whose span is not through last line hints that a multi-line replace needs a served neighbour; an unquoted `anchor=` containing `"` is refused with a quote instruction; `body=@path` loads the body from a root-relative file; a failing check prints its last error line above the log path.
**Notes:** quality-blueprints inbox `23c3061…`, filed 2026-09-14, mrw v1.19.0 (`93c8a1a`), ~12 subagents, ~30 plans. Ledger held. M: *"all. we have to address these"*.

## Context

**The class this record governs.** Every served-path friction quality-blueprints named after a day on v1.19.0, except the ed-style end-marker they offered as one of two fixes for leftover `body=` (Decision 1 + Decision 4 cover that ask). Enumerated 2026-09-14 from inbox drawer `23c3061bea674f66209e5cebb2b5162dcf5959e110fb0fbbb16c13c5f6652536` (five numbered items) against

```
git ls-files internal/plan/plan.go internal/read/read.go cmd/mrw/main.go internal/mcp/tools.go internal/guide/guide.go
```

Five tracked files. Members left out, and why:

- `internal/apply/apply.go` — bodies are loaded into `plan.Hunk` before `Apply`. Apply still splices lines.
- `internal/check/check.go` — `Result.Tail` already exists. The last-error line is a `reportCheck` print.
- `internal/seen`, `internal/state` — ledger and state dir unchanged.

**Why this is a record.** Five independent refusals or retries on a PATH that already held the ledger. None is a silent no-op. Together they are how a caller who counted wrong, read exactly A–B, pasted an unquoted import line, or needed a 600-line create, spent extra turns.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| leftover `body=` | `internal/plan/plan.go` ~247 | Reports the first extra line once; does not name declared N vs extra M. |
| `--dry-run` | `cmd/mrw` write | Applies nothing; does not print parsed hunk body counts. |
| ADR-052 neighbour | `internal/apply` | Refuses multi-line replace without a served line after End. `read` does not hint. |
| unquoted `anchor=` | `joinUnquotedAnchorOptions` | Consumes until next `key=` (ADR-040 T2). `splitHeader` treats `"` as a quote toggle, so `from "vitest"` strips the quotes and the guard matches the wrong line. |
| `body=` integer | `parseHeader` `case "body"` | Line count. Empty create is `body=0` (ADR-027). No file-backed body. |
| `check.Result.Tail` | `internal/check` | Last 30 lines. `reportCheck` dumps all of them, then `full output: <path>`. |
| `rooted.IsRooted` / `Resolve` | `internal/rooted` | The only root check. Use it for `body=@path`. |

## Decision

**1. Leftover `body=` names the count.** One error per hunk (unchanged). It names declared `body=N` and how many extra non-empty lines followed, and keeps the first extra line in the message. `--dry-run` (human, not `--json`) prints one `parsed:` line per hunk: path, address, op, `body=N` (after `body=@path` load). No third grammar.

**2. `read` hints the neighbour.** After printing `@@ start-end` for a served span of two or more lines whose End is not the file's last line, print one note that a multi-line replace of that span needs a served line after End. Quiet on whole-file reads, single-line spans, and when End is last (ADR-052 already skips the licence there). The licence itself does not change.

**3. Unquoted `anchor=` containing `"` is refused.** Scan the raw header: if `anchor=` is not immediately `"…"` or `'…'` and the unquoted span contains `"`, refuse and tell them to write `anchor="…"`. Quoted form, including escaped quotes inside, is unchanged. ADR-040 `anchor=func openTestStore body=1` stays valid.

**4. `body=@path` loads the body from a file.** Same `body=` key. `@` plus a root-relative path. `rooted.IsRooted` / `Resolve`; name the path on failure. An empty file is `body=0`. Inline `body=N` is unchanged. A `create` of hundreds of lines is the motivating path. The body file is not an edit: ADR-002 does not require it to have been served. Load after parse, before apply, so CLI and MCP share one function.

**5. A failing check prints its last error above the log path.** On FAIL, one `check last:` line — the last non-empty Tail line — immediately above the `full output:` verdict line. Tail dump stays. PASS is unchanged. Verdict still from the process (ADR-003).

## Alternatives Considered

- **Ed-style `.` terminator** — a third body grammar next to unbounded and `body=N`. Leftover count plus `body=@path` cover both asks. Rejected.
- **Warning then still match the stripped unquoted anchor** — the field-report plan would keep matching the wrong line. Refuse.
- **Parse `"` as literal inside unquoted tokens** — would make the original plan work, and would also change how `splitHeader` treats a new token that starts with `"`. Refuse-and-quote is the smaller lexer change and names the fix ADR-015 requires.
- **Print the neighbour hint on every line** — noise. One note per multi-line span that is not through EOF.
- **Drop the 30-line Tail dump** — the last line is the ask; the dump is still the nearby context. Keep both.

## Component / Boundary Impact

Parser leftover message and two header options (`"` refuse, `body=@path`). `read` prints one extra note. CLI `--dry-run` and `reportCheck` print one extra line each. MCP calls the same `LoadBodyFiles` the CLI does. Apply, seen, check engine, state: unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| leftover error text | names N vs extra M | `plan.Parse` | CLI/MCP parse refusal |
| `write --dry-run` | `parsed:` lines | `cmd/mrw` | human dry-run |
| `read` span | neighbour note | `read.Run` | CLI/MCP read |
| `anchor=` | unquoted `"` refuse | `plan.parseHeader` | CLI/MCP parse refusal |
| `body=@path` | `Hunk.BodyFile`; `plan.LoadBodyFiles` | parse + load | CLI/MCP write |
| `reportCheck` | `check last:` | `cmd/mrw` | exit 3 receipt |
| contract | §100–§104 | `scripts/contract.sh` | CI |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| leftover names extra count | T1 | T4 message may mention `body=@path` | No — T1 wording allows the later clause |
| `Hunk.BodyFile` + `LoadBodyFiles` | T4 | CLI/MCP in T4 | No |
| none otherwise | T2, T3, T5 | — | No |

## Implementation

Tasks in `docs/adr/ADR-060-plan-ux-from-the-v1-19-field-report/tasks/`. Red tests that fail on assertion → implement → mutants → §100–§104 → teach.

## Consequences

- A miscounted `body=` says how far off the count was, and `--dry-run` shows what parsed.
- A first ranged read of A–B that will be a multi-line replace names the neighbour before the write refuses.
- An unquoted import-line `anchor=` fails at parse with a quote instruction instead of matching the wrong line.
- A 600-line create can be `body=@path` instead of a counted paste.
- Exit 3 still means applied-and-unverified; the last check line is visible without opening the log.

## Out of Scope

- An ed-style `.` body terminator. (permanent: boundary: leftover count plus `body=@path` cover the two asks)
- Widening unquoted-until-next-key to `sha=` / `lines=` / `body=`. (permanent: boundary: ADR-040 named only `anchor=`)
- Neighbour licence on a single-line address. (deferred: docs/adr/BACKLOG.md "Neighbour license on a single-line address")
- Changing default Tail length. (permanent: fact: Tail is still lastLines 30; citation: file `internal/check/check.go:241`)
- Loading `body=@` inside `Parse` (no root). (permanent: fact: Parse takes only `io.Reader`; citation: file `internal/plan/plan.go:156`)
- Arming `--strict-balance` as default. (permanent: boundary: ADR-056 prices it first)
- Requiring the body file to have been served. (permanent: boundary: ADR-002 is the file being edited)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Hint noise on every ranged read | Med | Low | Only 2+ line spans not through EOF; contract greps the note and a whole-file read without it. |
| `body=@` escape outside the root | Low | High | `rooted.IsRooted` then `Resolve`; §103 names the miss. |
| Leftover message breaks `TestAnUnaccountedBlockIsReportedOncePerHunk` | Low | Med | Keep one error and the phrase `is not part of any hunk`. |

## Rollback

Revert leftover wording, the `parsed:` lines, the read note, the quote refuse, `BodyFile` / `LoadBodyFiles`, `check last:`, §100–§104, and the teach. Native `body=N` and ADR-052 stay.

## Follow-ups

- [ ] Close inbox drawer `23c3061…` after ship (T5).
