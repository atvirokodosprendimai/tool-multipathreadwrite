# ADR-182: `mrw seen` names no provenance the ledger does not hold

**Status:** Proposed
**Date:** 2026-10-29
**Amended:** 2026-10-29, before acceptance, from a Codex xhigh challenge (2 P2, 2 P3). Each finding was checked against the code. (1) Both tests whose text changes are in active inline locks: six Verification Log receipts carry them as `test-lock-b64`, not under `docs/adr/locks/`, which was the only place the draft searched. T1 now refreshes those six receipts (Invalidates, T1 S5). (2) `parseLine` decodes `Shown` and `Spans` independently, so `w2-3|` is a shape too: it is added to the table and to the unit test, and the claim that `Shown` is a subset of `Spans` is limited to the records producers and `merge` write (Decision 1). (3) The fence no longer greps PASS lines that the existing regression tests print today; they still run inside it. (4) S1 carries its proof form.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-005, ADR-118, ADR-150, ADR-161, ADR-167
**Governs:** `internal/seen/seen.go`, `internal/seen/served182_test.go`, `cmd/mrw/seen182_test.go`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/seen182_test.go::TestSeenNamesNoProvenanceTheLedgerDoesNotHold`
**Invalidates:** ADR-167 — the cost its Consequences name at `docs/adr/ADR-167-a-body-can-be-lines-the-caller-read.md:116`, "(`mrw seen` prints them like a `--sub` plan's lines)": a `--quiet` licence and a `--sub` plan's lines still print alike, because the ledger holds them alike, but neither is credited to a `--sub` plan any more. The exact strings asserted by ADR-118 T1 (`internal/seen/seen_test.go:308,311`, `TestAWrittenFileIsLicensedButNotShown`) and ADR-161 T1 (`cmd/mrw/sub161_test.go:212`, `TestASubReadRefusalsAndExits`) change with this record. Those two test bodies are locked inline (`test-lock-b64`) in six receipts: ADR-075 T1, ADR-079 T1, ADR-118 T1 and ADR-150 T1 for the first, ADR-161 T1 and ADR-164 T2 for the second. T1 refreshes the six receipts, and none of those records' decisions changes. Checked and unchanged: ADR-118's rule (only shown lines count for `occurrence=N`), ADR-161's `w<shown>|<licensed>` line, ADR-150's ledger format v6, ADR-167's `from=` gate and its refusal text (`internal/apply/fromcopy.go:234`, which already names no provenance).
**Served-path change:** `mrw seen` and three write refusals (`… has not been read: mrw served <X>`) describe an observation mrw wrote, planned or quietly licensed by what was shown and what is licensed — `no lines, and licensed lines 1-3`, `no lines, and licensed the whole file`, `lines 2-3, and licensed the whole file` — instead of `… since mrw wrote it` or `…, and a --sub plan licensed …`; an empty licence reads `and licensed none` instead of a dangling `lines `. The ledger, every licence and every exit code are unchanged.

## Context

Issue #555, reproduced 2026-10-29 on this branch's base (e8a6a3da, `go build ./cmd/mrw`, macOS, a scratch git checkout): after `mrw read --quiet h.txt:1-2`, `mrw read --quiet g.txt`, `mrw read --quiet 'e.txt:/nomatch/'` and `mrw read h.txt:2`, `mrw seen` printed

```
7fdf2c70  no lines, and a --sub plan licensed lines   e.txt
09834d48  no lines since mrw wrote it  g.txt
cf2c7f63  lines 2-3, and a --sub plan licensed lines 1-3  h.txt
```

over a ledger holding `w|`, `w` and `w2-3|1-3`. No `--sub` plan ran and mrw wrote nothing. Since ADR-167 (Zy, 2026-10-22) a `--quiet` read records what it licenses as licensed-not-shown — `mcp.LicensedNotShown` (`internal/mcp/licensed167.go:10-17`) sets `Written` and an empty `Shown`, on the CLI through `recordedAs` (`cmd/mrw/quiet167.go:10`) and on `mrw_read` (`internal/mcp/tools.go:704`) — which is the shape ADR-118 gave a file mrw wrote (`internal/writer/writer.go:102`) and ADR-161 gave a `--sub`/`--rewrite` plan's lines (`internal/read/read.go:973`, `internal/read/rewrite.go:208`). `seen.Observation.Served` (`internal/seen/seen.go:115-136`) was written before that and reads provenance off the shape: `Written` with nil `Spans` is "since mrw wrote it", `Written` with `Spans` is "a --sub plan licensed". The ledger line (`formatSpans`, `internal/seen/seen.go:603-620`; `parseLine`, `:530-556`) carries no source, so the label is a guess, and for a quiet read it is wrong. The licence itself is right. ADR-167 saw the cost and accepted it in prose (`:116`); its own `from=` refusal was rewritten to name no provenance for this reason (`internal/apply/fromcopy.go:230-234`, the Codex re-check of 8c00f465), but `Served` was not.

The third line shows a second defect the issue did not: a quiet read whose pattern matched nothing records an empty, non-nil licence (`w|`), and `quoteSpans` of no spans renders `lines ` with nothing after it.

**The class** — *every place a ledger observation is rendered into text*:
`grep -rn 'Served()\|quoteSpans(' --include='*.go' cmd internal | grep -v _test` — run 2026-10-29, 9 hits: one producer, `Observation.Served` with its helper `quoteSpans` (`internal/seen/seen.go:115-149`), and four consumers: `mrw seen` (`cmd/mrw/main.go:869`) and three refusals in `internal/apply/apply.go` (`:1357` a path op, `:1362` a line address, `:1573` `occurrence=N`). The ninth hit, `internal/apply/fromcopy.go:233`, is a comment: the `from=` refusal deliberately does not call `Served` and is out of this class. The MCP read receipt serializes observations with `Written` and `Shown` excluded (`internal/seen/seen.go:70-73`), so it renders no provenance. All four consumers change through the one producer; none of them changes its own bytes.

## Existing Primitives Audit

- `Observation.Served` (`internal/seen/seen.go:115`): the one renderer. Changed in place; its signature and its non-`Written` branches (`the whole file`, `no lines`, `lines …` — a read that showed what it licenses) are unchanged.
- `quoteSpans` (`internal/seen/seen.go:139`): reused for non-empty spans; the empty licence is rendered before it is called.
- `formatSpans` / `parseLine`: unchanged; no field is added (Alternatives).
- `fromcopy.go:234`'s wording ("license an edit and show nothing"): the model for naming the shape and not the source; not reused as text, since `Served` completes the sentence `mrw served <X>`.

## Decision

**1. A written-shaped observation is described by what was shown and what is licensed, and by nothing else.** For `Written`, `Served` returns `<shown>, and licensed <licence>`, where `<shown>` is `no lines` when `Shown` is empty and `quoteSpans(Shown)` otherwise, and `<licence>` is `the whole file` when `Spans` is nil, `none` when `Spans` is empty and non-nil, and `quoteSpans(Spans)` otherwise. Every shape the ledger holds, as `mrw seen` will print it:

| Ledger | Sources the ledger cannot tell apart | Today | Decided |
|--------|--------------------------------------|-------|---------|
| `w` | a write; a whole-file `--quiet` read | `no lines since mrw wrote it` | `no lines, and licensed the whole file` |
| `w2-3` | either, then a read of 2-3 | `lines 2-3 since mrw wrote it` | `lines 2-3, and licensed the whole file` |
| `w\|1-3` | a `--sub`/`--rewrite` plan; a ranged `--quiet` read | `no lines, and a --sub plan licensed lines 1-3` | `no lines, and licensed lines 1-3` |
| `w2-3\|1-3` | either, merged with a read | `lines 2-3, and a --sub plan licensed lines 1-3` | `lines 2-3, and licensed lines 1-3` |
| `w\|` | a `--quiet` read whose range matched nothing | `no lines, and a --sub plan licensed lines ` | `no lines, and licensed none` |
| `w2-3\|` | an empty licence merged with a read, or any ledger so decoded (`parseLine` reads the two fields independently) | `lines 2-3, and a --sub plan licensed lines ` | `lines 2-3, and licensed none` |

The licence is printed whole, not minus what was shown. In the records producers and `merge` write, `Spans` includes the lines also in `Shown` (`merge`, `internal/seen/seen.go:822-824,839-844`). `parseLine` does not enforce that (`internal/seen/seen.go:544-552` accepts `w2-3|7-8`). "shown X, and licensed Y" states the two fields as stored, so it is exact either way, and no subtraction is needed. Every form reads in both frames it is printed in: the `mrw seen` column, and `… has not been read: mrw served no lines, and licensed lines 1-3. A line address means nothing …`.

**2. Nothing else changes.** No ledger field, no format version (ADR-150 stays at v6), no licence, no exit code, and no non-`Written` text. The issue's own suggestion (`licensed, not shown: lines 1-3` / `the whole file licensed, none of it shown`) is not taken verbatim: the first is false after a merge, when some licensed lines were shown, and neither completes `mrw served …`.

**What would make this decision wrong:** a source the ledger does record, so that a label could name it truthfully — none does (`parseLine`), and adding one is the rejected alternative; or a consumer that parses the old text — none is known: the only matches are the two tests this record changes, and `scripts/contract.sh:4148` greps the prefix `has not been read: mrw served`, which stays.

## Alternatives Considered

- **A provenance tag in the ledger** (`wq…`, `ws…`, or a source field). Would let `mrw seen` say which command licensed the lines. Rejected: it is a ledger format change (ADR-150: a v7 that v1.66–v1.78 refuse to trust) and a merge rule for mixed sources, bought for a diagnostic; the licence, the only thing the ledger exists for, does not depend on the source.
- **The issue's wording verbatim.** Rejected in Decision 2: inexact after a merge and unreadable after `mrw served`.
- **Subtract `Shown` from the licence** (`lines 2-3, and licensed, not shown: lines 1`). Exact, but more code in an engine package to say what the two lists already say together.
- **Leave the label and document it.** Rejected: `mrw seen` is the file that settles a refusal (AGENTS.md §5), and it names a cause that did not happen.

## Component / Boundary Impact

- `internal/seen` (engine): `Observation.Served` only. `cmd/mrw` and `internal/apply` print what it returns and do not change. Ownership unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw seen` stdout | the description column of a written-shaped entry | `internal/seen` | people, agents settling a refusal |
| write refusals at `internal/apply/apply.go:1357,1362,1573` | the text after `mrw served` for a written-shaped entry | `internal/seen` | CLI and `mrw_write` callers |
| contract §316 | new rows | `scripts/contract.sh` | CI |

No `--json` or MCP receipt key changes (`docs/receipts.txt` untouched): the refusal reason is a free-form string.

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request; an engine diff, so it names this record and gets a `go-reviewer` pass (`.claude/rules/lifecycle.md` §6).

## Consequences

- **Positive:** `mrw seen` and the refusals stop naming a command that did not run; the empty licence stops printing a dangling `lines `.
- **Negative:** `mrw seen` can no longer hint which command licensed a file, a hint that was right only when no `--quiet` read had touched it.
- **Neutral:** two existing tests take the new text; no contract row changes.

## Out of Scope

- A source tag in the ledger (permanent: boundary: rejected in Alternatives; a format change for a diagnostic)
- Rewording the `from=` refusal (permanent: boundary: `internal/apply/fromcopy.go:234` already names no provenance, and ADR-167 owns it)
- `mrw seen --json` (permanent: boundary: `mrw seen` has no JSON form and no consumer has asked; ADR-180 drew the same line)
- `cmd/mrw/fromcopy167_shown_test.go:39`, which asserts the old phrases are absent from a `from=` refusal (permanent: boundary: after this record they exist nowhere in production, so the assertion keeps guarding only a revert; it is in a lock (`docs/adr/locks/61a81dab….lock`) and editing it to buy nothing would move ADR-167's lock)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A consumer matched the old text | Low | Low | none known (Decision, "What would make this decision wrong"); release notes name the change |
| A shape the table misses | Low | Low | the shapes are the ones `parseLine` can produce (`w`, `w<shown>`, `w\|<l>`, `w<shown>\|<l>`, with `<l>` possibly empty and not necessarily covering `<shown>`); T1's unit test takes each, a disjoint pair included |

## Rollback

Before a release: revert T1. After a release: restore the old strings in a later release; nothing persistent changes, since the ledger is untouched.

## Follow-ups

- [ ] none at authoring
