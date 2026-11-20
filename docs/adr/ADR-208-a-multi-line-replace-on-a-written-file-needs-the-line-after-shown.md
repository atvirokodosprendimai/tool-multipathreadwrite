# ADR-208: A multi-line replace on a file mrw wrote needs the line after it shown

**Status:** Accepted
**Accepted by:** Zy, 2026-11-20: *"accepted the rest"*, to the question of accepting the open records ADR-207 to ADR-211 (#723, #720) as they stand.
**Drafted for:** Zy, 2026-11-20, from an independent critique of the same day (issue #704). Zy's instruction on the critique's findings, as relayed to the drafter by the session that commissioned these records, was *"address all, no choosing"*, so this record carries one decision. Challenged before acceptance (Codex at xhigh, 2026-11-20, on `2ded135b`: AMEND); every amendment is applied below or answered in History.
**Date:** 2026-11-20
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-052, ADR-118, ADR-161, ADR-190, ADR-193, ADR-200, ADR-203
**Supersedes:** in part, and only for a record that is whole and written-shaped:
- ADR-052 Decision 2's "unless that observation already covers End+1": such a record covers every line, so the line after must have been shown since the write, in the file's current version;
- ADR-005's whole-file licence of a written file, for the line after a multi-line replace only (every line stays licensed for every other purpose);
- ADR-190 Decision 2's "Otherwise the line after needs the ledger, as now": on such a record, it needs a showing;
- ADR-161 Decision 3's "The read records each span and the line after it as licensed", where the file's record was already whole and written-shaped: `merge` keeps that record (`internal/seen/seen.go:846`), so a `read --rewrite` plan written by a later call needs the line after shown; Decision 9, for a multi-line replace after a whole-file `--quiet` read; Decision 10's "`--quiet` licenses it without printing it", where the record was already whole and written-shaped: a ranged `--quiet` read there adds no showing (`seen.go:841-846`; probed 2026-11-20 after `mrw write --create` and `mrw read --quiet f.txt:2-3`: `mrw seen` still says `no lines, and licensed the whole file`), so it no longer meets the line-after rule; Decision 12, for the same cases through `mrw_read`;
- ADR-203's "ADR-052's licence of the line after a range … untouched".

ADR-052's rule for every other record, its EOF rule and its "`--force` does not waive", stand. A ranged `--quiet` read on a record that stays partial keeps its bounded licence, judged as today; on a record already whole and written-shaped it adds no showing (ADR-161 Decision 10, above).
**Governs:** `internal/apply/apply.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/apply/after208_test.go::TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown`
**Served-path change:** a multi-line `replace` that ends before the last line, on a file whose ledger record is whole and written-shaped (mrw wrote it, or a whole-file `--quiet` read licensed it), is refused unless a read has shown the line after the range in the file's current version: exit 1, nothing written, ADR-052's reason ending with the read to make (`mrw read --max-cols 0 PATH:N+1` when the path is one shell word, a description of it otherwise). It lands after that read, when the range ends on the last line, and when `mrw write --ast-grep … --rewrite` (or `mrw_write`'s `ast_grep` with `rewrite`) made the plan in the same call. A ranged `--quiet` read of such a file does not meet it. On `mrw_write` the refusal is the usual `isError` receipt with the failed hunk.

## Context

**ADR-052's rule.** A multi-line replace is refused unless the ledger covers the line after its range, because the closer a short range leaves behind sits on that line and the receipt cannot show it. The check is `needAfter` (`internal/apply/apply.go:1932-1936`): `haveObs && !obs.Covers(end+1, end+1)` for a numbered hunk, and for a quoted one (ADR-190) `tail == 0 && … && !(haveObs && recorded.SHA == shaBefore && obs.Covers(end+1, end+1))`.

**Why it does not fire on a file mrw wrote.** A file mrw wrote is wholly licensed (ADR-005): its record has `Spans == nil`, and `Covers` answers true for every line of a whole record (`internal/seen/seen.go:90-93`). The ledger also keeps what was shown since the write (`Shown`, ADR-118) and has the method that asks it, `ServedLine` (`seen.go:81-86`); `needAfter` does not call it. Neither does the numbered branch compare the record's sha with the file's: under `--force` the file-level "changed since" refusal is skipped (`apply.go:1430`, `:1444`) and the record kept is the old one (`:1464`), so its `Shown` can describe another version.

**Reproduced** on v1.93.0 (`18ffa49`; its `internal/apply` and `internal/seen` equal the reviewed commit `2ded135b`), 2026-11-20. `f.go` = `func A() {` / `\tif x {` / `\t}` / `}`:

| Record of `f.go` | Plan | Result |
|------------------|------|--------|
| created by `mrw write --create` | `@@ f.go 1-3 replace anchor="func A"`, body `func A() {` / `\ty()` / `}` | exit 0, a second `}` at line 4; the receipt's only sign is a `balance` row (`{ +1 → +0`) |
| read `g.go:1-2` (served 1-3, ADR-161) | the same on `g.go` | exit 1, `replace of 1-3 needs a served line after 3: read line 4` |
| created by `mrw write --create` | the same change as a `--format=search_replace` block | exit 0, a second `}` at line 4 |
| created by `mrw write --create` | the same change as a `--format=apply_patch` hunk (three `-` lines, three `+` lines, no context) | exit 0, a second `}` at line 4 |
| read 1-3, and not read | the same block | exit 1, the same refusal with ADR-190's hint |

The issue says the written case lands "with only a closer hint"; on this fixture there is no `closer` row, because the replaced range and the body both end in `}` (ADR-119's condition, `internal/apply/shape.go`), only the `balance` row.

**Which records are written-shaped.** `Written: true` is set by a landed write (`internal/writer/writer.go:108`, whole; a write whose every hunk was a quote records read-shaped spans instead, `:106`, ADR-190), a `read --sub` plan (`internal/read/read.go:996`, partial: the planned lines), a `read --rewrite` plan (`internal/read/rewrite.go:208`, partial: each span and the line after it, `:197`), and a whole-file `--quiet` read (probed: `mrw seen` says `no lines, and licensed the whole file`). A partial licence merged into a whole written record leaves it whole (`seen.go:846`).

**Which plans mrw makes.** `--sub` emits one single-line hunk per changed line (`internal/read/sub.go:59-64`), so ADR-052 never asks it for a line after. `--rewrite` emits one hunk per span of changed lines, which can be multi-line (`rewrite.go:189-196`). Both are marked `Computed` in `apply.Options`, and so are the foreign formats (`cmd/mrw/main.go:2331`, `internal/mcp/tools.go:1036`).

**Audit of the class**: *the records a multi-line replace can be judged against.* None (no ledger: every check skipped); read-shaped partial (`Covers` the spans: refused today when the line after was not served); read-shaped whole (shown everything); written-shaped partial (`Covers` the licensed spans, which for `--rewrite` include the line after); written-shaped whole (the hole). Only the last changes.

## Existing Primitives Audit

- `Observation.ServedLine` answers "shown since the write" for a written record; ADR-118's `occurrence=` check (`apply.go:1683`) and the `from=` check (`fromcopy.go:246`) already use it.
- The quoted branch already binds the record to the file's sha; this record adds the same binding to the numbered branch, for this case.
- `staleRemedy` (`apply.go:2759`, ADR-203) builds the read command a refusal names, with `--max-cols 0`, so a line cut by `MRW_MAX_COLS` is read whole and licensed.

## Decision

1. **On a record that is whole and written-shaped, the line after is met only by a showing in the current version.** In both branches of `needAfter`, when `obs.Written && obs.Whole()`, the condition is `recorded.SHA == shaBefore && obs.ServedLine(end+1)`. Every other record is judged exactly as today, the no-observation path included.
2. **The skips stay.** A range that ends on the last line needs no line after; a single-line replace is unchanged.
3. **One plan source is not judged by it: `--rewrite` in the same call** (`mrw write --ast-grep P --rewrite R`, `mrw_write`'s `ast_grep` with `rewrite`). Its span is the match mrw found, and the plan maker licenses the line after each span it computed (`rewrite.go:197`). `apply.Options` gains the fact that a plan came from that source; `Computed` keeps its meaning for ADR-200 and ADR-203. A plan printed by `mrw read --rewrite` or `mrw_read`'s `rewrite` and written by a later call is judged. `--sub` needs no exemption. Foreign formats (`--format=search_replace`, `--format=apply_patch`) are judged: the probe shows the hole there too.
4. **`--force` does not waive it**, as ADR-052 says, and cannot meet it with a showing of another version.
5. **The refusal names the read**: on such a record, ADR-052's reason ends with the read `staleRemedy` names for line N+1 (`internal/apply/apply.go:2752-2758`): a command to paste, `mrw read --max-cols 0 PATH:N+1`, when the path is one shell word, and a description of that read otherwise (a path such as `a b.go`, or a single letter).
6. **The words.** `AGENTS.md` (rule 4's sentence on the line after), `README.md`'s neighbour-licence bullet and the full instructions (`internal/guide/guide.go`, the sentence "A file mrw just wrote is wholly known …") each say: `on a file mrw wrote, a multi-line replace that ends before the last line also needs the line after it shown since the write`.

## Alternatives Considered

- **Refuse only when the body ends in a closer the next line repeats** (ADR-119's `closer` condition). Rejected: it reads target syntax mrw does not model, and the probe lands with no `closer` row.
- **Licence the line after automatically by echoing it in the receipt.** Rejected: an echo after the write cannot stop the write; `--echo-pad` already exists for that.
- **Apply `staleMarks`' predicate whole: `Force` and every `Computed` plan exempt.** Rejected: it leaves the `search_replace` hole the probe shows, and ADR-052 already refuses to let `--force` waive the line after.
- **`ServedLine` alone, without the sha.** Rejected: under `--force` the record kept can describe another version.
- **Leave it.** Rejected: the closer guard is off on files the caller has just written.

## Component / Boundary Impact

Engine package owned here: `internal/apply` (`needAfter`, the plan-source field, the refusal's remedy). `cmd/mrw/main.go` and `internal/mcp/tools.go` set the field for a same-call `--rewrite`. `internal/seen` is unchanged (`ServedLine` exists); `internal/plan`, `internal/read`, `internal/check` and `internal/state` are unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw write` / `mrw_write`: a multi-line replace on a whole written-shaped record | refused, exit 1 / `isError` with the failed hunk, unless the line after was shown in this version | T1 | every caller |
| the refusal's reason on such a record | ends with the read to run | T1 | every caller |
| `apply.Options` | marks a plan made by a same-call `--rewrite` | T1 | `cmd/mrw`, `internal/mcp` |
| `AGENTS.md`, `README.md`, `mrw instructions` | Decision 6's sentence | T1 | every caller |
| `scripts/contract.sh` §359 | the created-file refusal paired with the same plan landing after `mrw read f359.go:4`; a `search_replace` block without a kept tail refused and with one landing; an `apply_patch` hunk refused; a whole `--quiet` read refused; with §264's recorded ast-grep, a same-call `--rewrite` on a created file landing and the same plan from `read --rewrite`, written later, refused until the line after is read | T1 | the contract gate |

No receipt key is added or renamed (ADR-111).

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1.

## Consequences

- **Positive:** the closer guard holds on files mrw wrote, for native plans and foreign formats, as it already does on files the caller read.
- **Negative — the cost:** a second multi-line replace on a file mrw just wrote, or after a whole-file `--quiet` read, needs the line after the range shown first: one `mrw read PATH:N+1`, which a ranged read of the lines being replaced already serves (ADR-161); or, for a `search_replace` block, a REPLACE side that ends with the SEARCH side's last line (ADR-190's kept tail). A `read --rewrite` plan written later on a file mrw had written needs the same read. How often the refusal will fire is not measured; no existing count prices this shape.
- **Neutral:** single-line replaces, inserts, deletes and ranges ending on the last line are unchanged.

## Out of Scope

- A closer above the body (a short START), which the line after cannot show (permanent: boundary: ADR-052 chose the line after; a line before is a different miss, `internal/apply/apply.go:1922-1923`)
- A read-shaped record kept under `--force` after the file changed (permanent: boundary: its spans are judged by `Covers` as today; ADR-052's `--force` rule for a partial observation is unchanged, and this record changes only the whole written-shaped case)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A session that edits a file it wrote in many multi-line steps pays a read per step | Unmeasured | Low | the refusal names the read; a ranged read of the lines being replaced already serves the line after |
| A caller forgets to set the plan-source field | Med | Med | §359 drives both the same-call and the later-call `--rewrite` through the built binary, and T1's CLI and MCP tests do the same on Windows |

## Rollback

Revert T1; no state is involved.

## History

- 2026-11-20, Codex at xhigh on `2ded135b`, AMEND, three points, all applied after checking each against the code: the showing is bound to the file's sha (the numbered branch keeps the old record under `--force`, `apply.go:1430-1464`); the exemption is narrowed to a same-call `--rewrite` (`--sub` is single-line, `sub.go:59-64`), and the superseded promises of ADR-052, ADR-190 and ADR-161 are named; the friction estimate taken from ADR-203's E5 is withdrawn, since E5 priced edits by number alone and these replaces already carry `anchor=` or a quote; the wiring is proved through the CLI and MCP entry points.

## Follow-ups

- [ ] None beyond T1.
