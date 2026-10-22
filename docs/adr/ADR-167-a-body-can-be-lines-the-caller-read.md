# ADR-167: A hunk can copy its body from lines the caller has read

**Status:** Accepted
**Accepted:** 2026-10-22 by Zy — "start implementing this copy adr", then "mrw copy adr continue"; the four open questions settled with the recommended answers: a separate `from=` key, `--force` does not reach the source, a source under `.git` may be copied, pattern, JSON-path and working-set sources stay refused.
**Date:** 2026-10-22
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-005, ADR-006, ADR-008, ADR-013, ADR-021, ADR-027, ADR-029, ADR-030, ADR-035, ADR-052, ADR-060, ADR-073, ADR-077, ADR-086, ADR-087, ADR-091, ADR-104, ADR-108, ADR-111, ADR-113, ADR-118, ADR-137, ADR-143, ADR-154, ADR-155, ADR-161
**Governs:** `internal/plan/plan.go`, `internal/apply/apply.go`, `internal/apply/wirepath.go`, `internal/refusal/refusal.go`, `internal/writer/flow.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/mcp.go`, `internal/mcp/schema.go`, `internal/curve/score.go`, `internal/guide/guide.go`, `docs/receipts.txt`
**Enforced-by:** None — until T2 lands, which adds `internal/apply/fromcopy167_test.go::TestAFromCopyCopiesOnlyLinesTheCallerWasServed`
**Invalidates:** none — checked. `body=@` keeps ADR-060's meaning exactly, colons included; ADR-060's Out of Scope "Requiring the body file to have been served" still holds for it, because `from=` is a different key with its own rule.
**Served-path change:** a plan hunk may say `from=PATH:ADDR` (`ADDR` one of `N`, `N-M`, `A,+N`, `$`, `N-$`) in place of a body: mrw copies those lines of the file as it was before the plan, refuses unless the caller was served them in the version on disk, and the receipt names the source range and its first and last copied line; on `mrw write` and `mrw_write` alike.

## Context

**The ask.** M, 2026-10-22, relayed by Zy: *"I read files 1 2 3, then `mrw --copy @1:fromline:toline to file @2:1:3:replace` — instead of output tokens."* An agent that moves or duplicates a block today reads it (input tokens) and then emits every line of it again as a hunk body (output tokens, the expensive half). Output tokens are what mrw saves most: README's Stage 5 table (SWE-bench, 191 tasks per model, Opus 5.5, `docs/bench/stage-5/looks/look-5.txt`, the stage ADR-162 registered) has output tokens per task 5,489 → 3,888 (−29%), read 2026-10-22. ADR-161 moved the same way for rewrites by letting mrw write the plan. A copy is the remaining case where the caller re-types lines mrw already holds.

**What exists.** `body=@path` (ADR-060 Decision 4, T4 and T7) loads a WHOLE root-relative file as a hunk body, on `create`, `replace` and `insert-*`, and is not a ledger read: the motivating case was a body the caller wrote to a scratch file. There is no range form: `grep -niE 'copy|body=@' docs/adr/BACKLOG.md` and `adr-debt docs/adr` (2026-10-22) name no record or backlog entry for one.

**Where the licence can be checked.** `plan.LoadBodyFiles` runs before the ledger exists on every surface: `cmd/mrw/main.go:1767`, `internal/mcp/tools.go:919`, `internal/curve/score.go:102`; the ledger is taken inside `writer.Prepare` (`internal/writer/flow.go:152`) and handed to `apply.Apply` as `Options.Seen`. The lookups a licence needs live in `internal/apply/apply.go`: the alias lookup (`sameFileEntry`, `apply.go:1185`, ADR-021/ADR-029), the recorded sha, `Observation.Covers`, and the per-hunk verdict that makes siblings `skip` (ADR-001). So the source is resolved in apply. But the TARGET's gate cannot simply be reused: `covered` (`apply.go:1262`) returns true when there is no observation, under `Force`, and for a nil ledger, and the target's refusals advise `--force`. The source gets its own gate over the same lookups (Decision 3).

**Audit of the class** — *every place a hunk body is produced, carried, consumed, printed or named between the plan text and the receipt*. Enumerated 2026-10-22 on `48be90a4` (origin/main `f862d8a3` changes none of these files but `internal/guide/guide.go`, by one line):

```
git grep -nE '\.Body([^A-Za-z0-9_]|$)' -- '*.go' ':!*_test.go'                         # 44 lines
git grep -nE '\.Body([^A-Za-z0-9_]|$)|BodyFile|LoadBodyFiles' -- '*.go' ':!*_test.go'   # 59 lines
git grep -n 'apply.Input{' -- '*.go' ':!*_test.go'                                      # 3 builders
git grep -ln 'plan.Parse' -- '*.go' ':!*_test.go'                                       # 7 files
```

(The first cut of this record used `\.Body\b`, which `git grep -E` does not read as a word boundary and which matched nothing of `.Body` alone; found by the challenger review.)

- **Parse**: `internal/plan/plan.go` — `parseHeader`'s option switch (`case "body"` at 518), `CountedBody` set from `fixed` at 208 and 340, the empty-body deferral at 816, the unknown-option message listing the keys. Changed by T1.
- **Carry**: the three `apply.Input` builders, `cmd/mrw/main.go:1820`, `internal/mcp/tools.go:948`, `internal/curve/score.go:176`. Each must carry the source, or that surface writes no body. Changed by T2, with a test per surface.
- **Consume**: `internal/apply/apply.go` — `Input` (280), the hunk copy (401), the empty-body and create refusals (1354–1410), the splice (1840–1879). Changed by T2.
- **Print and name**: the `--dry-run` `parsed:` line (`cmd/mrw/main.go:1790`); the ADR-086 guard that refuses a name a `--json` receipt cannot carry (`cmd/mrw/main.go:1775-1787`); the human receipt line (`cmd/mrw/main.go:2635`); ADR-091's slash spelling of receipt paths (`internal/apply/wirepath.go:24-40`); `nameTheAck`, which adds the ack remedy to a `NotRead` refusal (`internal/mcp/tools.go:2459-2514`); `newerLedger`, which rewrites a `NotRead` reason under a newer ledger (`internal/writer/flow.go:214-227`). Changed by T2.
- **Receipt**: `apply.HunkResult` and its `MarshalJSON`; `docs/receipts.txt` `write hunks[]` and `mcp_write hunks[]`; the output-schema descriptions `TestEveryOutputSchemaPropertyIsDescribed` holds (`internal/mcp/schema.go:287-293`). Changed by T2.
- **Teach**: `internal/guide/guide.go:84`, `internal/mcp/mcp.go:741`, `AGENTS.md:349`. Changed by T3.
- **Left out, and why**: `touchesCode` (`internal/writer/flow.go:417`) decides whether the check runs from what the plan WRITES; a source is only read, so it does not count. The path-op validation (`internal/apply/pathop.go:125-157`) never sees a `from=` hunk: apply refuses `from=` on `unlink` and `rename` before it (Decision 2). The plan makers (`--sub`, `--rewrite`, `internal/read/plansource.go`) and the foreign formats (`internal/ingest/applypatch.go`, `create.go`) produce inline bodies and never a `from=`. `internal/read/read.go` only names `plan.ParseAddr` in a comment.

## Existing Primitives Audit

| Primitive | Where | Use |
|-----------|-------|-----|
| `parseHeader`'s option keys | `internal/plan/plan.go:470-530` | one more key, `from=`, beside `sha`, `lines`, `anchor`, `body`, `raw`, `occurrence` |
| `plan.ParseAddr` | `internal/plan/plan.go:744` | parses the source address; the result is then held to the numeric subset (Decision 2) |
| `rooted.IsRooted`, `rooted.Resolve` | `internal/rooted` | the source path's boundary: a rooted or drive path refused by name, outside the root, mrw's own state (ADR-077, ADR-134) and device names (ADR-081) refused, as for any path a plan names |
| `readLines` | `internal/apply/apply.go:2048` | reads the source as apply reads a target, through the 1 GiB bound (ADR-104/107) and `lines.Split` (ADR-065); its `existed` and `foreign` results are checked by the pre-pass itself, since `readLines` refuses neither |
| `sameFileEntry`, `Observation.SHA`, `Observation.Covers` | `internal/apply/apply.go:1170-1190`, `internal/seen/seen.go:90` | the source gate's lookups; not `covered`, whose fallbacks a source must not have |
| `RemovedFirst`/`RemovedLast` and their `MarshalJSON` | `internal/apply/apply.go:85`, `:129-160`, ADR-008 | the shape of the new receipt keys: bounded strings through `trim`, presence keyed on the form |
| `Slash`, `SlashIn` | `internal/apply/wirepath.go` | ADR-091's spelling for `body_from`'s path |
| `refusal.NotRead` | `internal/refusal/refusal.go:26` | not reused: a source refusal gets its own kind, so the two consumers that act on `NotRead` cannot mistake the source for the target (Decision 3) |

## Decision

1. **`from=PATH:ADDR` copies a body from lines of a file (T1, T2).** On `replace`, `insert-after`, `insert-before` and `create`, the option `from=` names a source in place of a body: the value splits at its LAST colon into `PATH`, root-relative, and `ADDR`. The body is lines `A..B` of `PATH` as it is on disk before the plan applies — which, since nothing is written until every hunk validates (ADR-001), is the original even when the same plan edits, renames or unlinks `PATH`. Source and target may be one file. `from=` excludes every other body: a `from=` hunk with `body=`, `body=@` or any body line under its header is refused at parse. `body=@` is unchanged (ADR-060): `body=@a:3` still loads the file named `a:3`.
2. **What a source may not be (T1 at parse; T2 in apply too, ADR-030).**
   - The address is exactly one of `N`, `N-M`, `A,+N`, `$` or `N-$`, with `N >= 1`. Refused, each with a T1 table row: `0`, `-`, `N-` (an empty end), an address with `$` as its START (`$-N`, `$,+N`), a comma list of ranges, a pattern (`/re/`, `/a/,/b/`, and any value holding `:/`, so `x.go:/a:b/` is named as a pattern rather than split inside it), and a JSON path (`$.k`, ADR-155). A range that runs past the last line, or whose end precedes its start, is refused at apply, as a write refuses one (AGENTS.md §3).
   - The path: `rooted.IsRooted` is asked before the split's result is used, so `/x:3`, `\x:3`, `C:\x:3` and `C:x:3` are refused as rooted, by name; a value with no colon, or whose tail is not an address (`from=a.go`, `from=C:\x`), is refused naming the tail it failed to read as an address. A source file whose own name holds a colon is reached by the last-colon split as long as an address follows (`from=a:b.txt:3` is lines 3 of `a:b.txt`).
   - A working-set pointer (`from=@1:3-5`) is refused: pointers resolve in the three builders for the hunk's own path only, a source would need a fourth resolution site, and the caller who was served the lines has the path in front of them.
   - `from=` on `delete`, `unlink` or `rename` is refused: their bodies are an expected removal and a destination, not content. `raw=true` with `from=` is refused, since no plan body is parsed for it to switch off; the parser's existing `raw=true without body=` refusal already covers it, and T1 pins that.
   - In apply the same shapes an `apply.Input` can express are refused again: `From` on `delete`, `unlink` or `rename`, `From` beside a non-empty `Body`, and a `From` span that is not numeric. `raw=` cannot be mirrored: `apply.Input` carries no `Raw` (`apply.go:280-296`).
   - A `from=` hunk never sets `CountedBody` (`plan.go:208`, `:340`): a `create` whose source was dropped by a builder then meets ADR-027's empty-create refusal instead of writing an empty file at exit 0; validate's empty-body deferral (`plan.go:816`) waits on `BodyFrom` as it waits on `BodyFile`.
3. **The source has its own gate, and it never waives (T2).** In apply, before any target is validated: the source's ledger entry, found under another spelling through `sameFileEntry` (ADR-021/029), must exist; its sha must equal the file's sha now; and `Covers(A, B)` must hold. A file mrw just wrote counts whole (ADR-005): that is the licence that lets a write address the file by line number, and a source address is a line number; ADR-118's stricter `ServedLine` is for counting pattern matches, which a source address never does. A source that does not exist before the plan is refused: it has no original. A UTF-16 or otherwise unsplittable source (`readLines`' `foreign`, ADR-073), a FIFO, a directory and a file over 1 GiB are refused as they are for a target. The gate differs from the target's `covered` in three ways, on purpose: it refuses a source with no observation; it ignores `--force` (Force is for the hunk's own lines, ADR-052's rule, and a copy forced past its licence is a body nobody was shown); and with a nil ledger it refuses (an engine call with no ledger cannot license a copy). Its refusal advises a read, never `--force`. What counts as served is the checkout's one ledger (ADR-161 Decision 14): a CLI `mrw read` licenses an `mrw_write` copy and the reverse; an `mrw_read` serve counts once acked (ADR-108); lines `--max-cols` cut are not licensed (ADR-137, ADR-154); lines licensed by `--quiet` or a `--sub` plan count (ADR-161). A refusal carries kind `refusal.SourceNotRead` and the source's path and span (unexported `HunkResult` fields), so `nameTheAck` offers the ack remedy for an unacked SOURCE page, and `newerLedger` names the source, not the target. Siblings `skip`; nothing is written.
4. **No source anchor; the receipt says what was copied (T2).** ADR-035 demands `anchor=` because a wrong replace range damages lines OUTSIDE what the receipt shows. A wrong source range puts its error INSIDE the body, where a receipt can show it — ADR-008's shape, not ADR-035's. So each `from=` hunk carries three keys: `body_from` (`PATH:A-B`, the resolved span, its path spelled by ADR-091's `Slash`), `copied_first` and `copied_last` (the first and last copied line through the same `trim` as `removed_first`, so bytes that are not valid UTF-8 encode as U+FFFD exactly as `removed_first` does today), present on every such hunk and on no other, whatever their values. A `--json` write whose source path is not valid UTF-8 is refused before anything lands, as ADR-086 refuses such a target. The human receipt adds ` copied PATH:A-B from "<first>" to "<last>"`; `--dry-run`'s `parsed:` line names `from=PATH:ADDR` instead of a body count of 0; the schema describes the three keys; `docs/receipts.txt` lists them under `write` and `mcp_write`. A refused copy's receipt and reason disclose no copied line. The target's own guards are unchanged: a multi-line replace still needs `anchor=` (ADR-035) and a served line after End (ADR-052).
5. **A move is two hunks (T3 teaches it).** Copy into the target and `delete` the source range, in one plan: ADR-001 makes the pair atomic, both address the original, and the delete's receipt names what it removed (ADR-008) beside `body_from`, so a mismatch between the two ranges is visible on one receipt. No `move` op: it would need two paths in one header and duplicate what a delete already does.
6. **The other edges follow from where the source is read.** CRLF: copied lines carry no endings and take the target's, as every body does (ADR-005). A source under `.git` may be copied, as a read of it is served (ADR-143 refuses only writes); the pre-pass therefore resolves the source with `rooted.Resolve` and not with apply's `resolve`, which adds the `.git` refusal.
7. **No CLI sugar.** No `mrw --copy` or `mrw copy`: the plan already expresses a copy in one header line, a move needs a plan anyway for atomicity, and ADR-060 rejected a third body grammar. The form is taught as a recipe in `mrw instructions` (T3).

**What would falsify it:** a copy applying from lines the ledger does not cover, from a file whose sha moved since it was read, from a source with no observation, or under `--force` without a licence; a copy reading the plan's own edit of the source rather than the original; a surface (CLI, MCP, curve) writing an empty or different body for the same plan; a `from=` create landing an empty file; a `from=` hunk whose receipt lacks `body_from`; `body=@a:3` no longer loading the file `a:3`.

## Alternatives Considered

- **M's CLI form, `mrw --copy @1:A:B @2:C:D:replace`:** a second write grammar beside the plan, with working-set pointers as files. Rejected: one plan line says the same, and a move or several copies in one atomic write need the plan anyway (Decision 7).
- **Extend `body=@` to `body=@PATH:ADDR`:** one key for "where the body comes from". Rejected by Zy (2026-10-22): it reinterprets a `body=@` value holding a colon, which ADR-060 accepts today, and puts two licence rules behind one key.
- **Resolve the range in `LoadBodyFiles` with the ledger passed in:** keeps `apply.Input` unchanged. Rejected: the loader runs before the ledger is taken (`cmd/mrw/main.go:1767` against `internal/writer/flow.go:152`), and the gate's lookups would be copied outside the engine that owns them (ADR-030).
- **Reuse the target's `covered` for the source:** less code. Rejected: it passes a missing observation, `--force` and a nil ledger (`apply.go:1262`), each a way to copy lines nobody was shown.
- **Reuse `refusal.NotRead` for a source refusal:** rejected: `nameTheAck` and `newerLedger` act on `NotRead` by the hunk's own path, and would name the target for a source that was not read.
- **Require a source anchor on a multi-line source, mirroring ADR-035:** rejected (Decision 4): a wrong source range is visible in the receipt, and no measured failure asks for the key.
- **Allow pattern, JSON-path or working-set sources:** rejected (Decision 2).
- **A `move` op:** rejected (Decision 5).

## Component / Boundary Impact

`internal/plan` parses one more option into two `Hunk` fields; `internal/refusal` gains one kind; `internal/apply` gains a source pre-pass with its own gate and three receipt fields, and owns the source's licence. `cmd/mrw`, `internal/mcp` and `internal/curve` each carry the source into `apply.Input`; `internal/writer` and `internal/mcp` name a source refusal by the source. No module added or moved. Engine packages owned by this record: `internal/plan`, `internal/apply`; `internal/read`, `internal/seen`, `internal/check`, `internal/state` stay byte-identical.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| plan grammar | `from=PATH:ADDR` on replace, insert-*, create | `plan.Parse` (T1) | CLI, `mrw_write`, curve |
| `plan.Hunk` | `BodyFrom`, `BodyFromAddr` | T1 | the three `apply.Input` builders (T2) |
| `apply.Input` | `From`, `FromStart`, `FromEnd`, `FromRelEnd` | the builders (T2) | `apply.Apply` |
| `refusal.Kind` | `SourceNotRead` | `internal/refusal` (T2) | `apply`, `nameTheAck`, `newerLedger` |
| receipt | `hunks[].body_from`, `hunks[].copied_first`, `hunks[].copied_last` | `apply.HunkResult` (T2) | `--json`, `mrw_write`, `docs/receipts.txt`, `internal/mcp/schema.go` |
| human receipt, `--dry-run` | the `copied` clause; `parsed:` names `from=` | `cmd/mrw` (T2) | CLI callers |
| teaching | a recipe; one sentence each in the plan reference | `internal/guide`, `internal/mcp/mcp.go`, AGENTS.md, README (T3) | agents |
| `scripts/contract.sh` §286, §287 | copy and move through the binary, each with its refused pair; the recipe run as printed | T2, T3 | CI |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `plan.Hunk.BodyFrom` / `BodyFromAddr`, set only by `from=`, never with `CountedBody` | T1 | T2 | No — new fields |
| `body_from`, `copied_first`, `copied_last`; kind `SourceNotRead` | T2 | T3 (teaches them) | No — keys only added (ADR-111) |

## Implementation

`tasks/README.md`; `tasks/T1-the-plan-reads-a-body-range.md`, `tasks/T2-apply-copies-only-served-lines.md`, `tasks/T3-instructions-teach-copy-and-move.md`. T1, then T2, then T3; each a red test, the change, killed mutants, a contract row where it reaches the binary, the gates, and a Codex review. T1 and T2 are engine diffs and get a `go-reviewer` pass (lifecycle.md §6).

## Consequences

- **Positive:** a moved or duplicated block costs one header line of output instead of its whole text; the receipt shows what was copied without the caller re-sending it; a plan sent to `mrw_write` can place lines that are not valid UTF-8, once a CLI read has licensed them in the checkout's one ledger (ADR-161 Decision 14).
- **Negative:** `apply.Input` grows four fields every builder must set; a builder that forgets them is caught by ADR-006's and ADR-027's empty-body refusals rather than writing silently (T2 tests each surface). A second licence gate exists beside the target's, over the same lookups.
- **Neutral:** `body=@`, inline bodies and every target guard are unchanged. Nothing measures the token saving; the evidence for building it is M's request and the Stage 5 output-token figure, not a bench arm.

## Out of Scope

- A `move` op or a `--copy` subcommand (permanent: boundary: two hunks in one plan are atomic already, Decisions 5 and 7)
- Pattern or JSON-path source addresses (permanent: boundary: the read that licenses the copy printed the numbers, Decision 2)
- Working-set pointers as a source path (permanent: boundary: pointers resolve for the hunk's own path in three builders; a fourth site buys nothing the served path does not, Decision 2)
- A source guard key mirroring ADR-035 (permanent: boundary: a wrong source range is inside the receipt, Decision 4)
- Several source ranges concatenated into one body (permanent: boundary: one hunk per source range; a plan takes as many hunks as it needs)
- `--force` reaching a source (permanent: boundary: Force is for the hunk's own lines, ADR-052, Decision 3)
- Copying from outside the root, or from mrw's state directory (permanent: boundary: ADR-005 and ADR-077 confine reads as well as writes)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A builder (CLI, MCP, curve) drops the source | Med | High | `from=` never sets `CountedBody`, so ADR-006/027's empty-body refusals fire; T2 tests each surface with a `create`, `$` and `A,+N`, and a mutant per builder drops `FromRelEnd` |
| A wrong source range pastes plausible wrong code | Med | Med | the source was served to the caller; the receipt names `body_from`, `copied_first`, `copied_last`; the target's anchor and the project's check still run |
| The source gate lets through what it must not (no observation, Force, nil ledger, stale sha, alias) | Low | High | its own gate, not `covered`; T2's engine test drives each of those cases, and a malformed `apply.Input` with a nil ledger |
| A source refusal names the target | Low | Med | kind `SourceNotRead` with the source's path and span; T2's test has an unacked source and an acked target |

## Rollback

Revert the parse branch, the apply pre-pass, the refusal kind and the three builders. Keep the three `HunkResult` fields and never set them: `TestNoShippedReceiptFieldDisappears` and `TestNoShippedMCPReceiptFieldDisappears` fail on a key removed from the receipt type while `docs/receipts.txt` lists it (ADR-111). No state changes.

## Follow-ups

- [ ] Refresh the centralised `mrw` skill from AGENTS.md when T3 merges
