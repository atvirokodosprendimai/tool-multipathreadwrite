# ADR-190: a SEARCH block that quotes every line it replaces licenses them

**Status:** Proposed
**Date:** 2026-11-02
**Amended:** 2026-11-02, before acceptance, from a Codex high challenge (5 findings, each checked against the code): the mark is a `quote=` option carrying a MAC under a key only the running process holds, so Parse stays the one door (ADR-030) and no caller can forge it; uniqueness is re-checked under the lock (second challenge); ADR-052's line after is met only by a quoted kept line; the assertions that pin today's refusal move in T1; native `delete` with an expected-removal body and apply_patch's compiled shape are named in the audit; the mutants cover range leakage, an unquoted sibling and a comparison outside the lock.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-051, ADR-175, ADR-191
**Supersedes:** ADR-051, in one clause: "A unique exact SEARCH match is location, not a license" (its Existing Primitives Audit and Decision 3), for `--format=search_replace` only
**Governs:** `internal/ingest/searchreplace.go`, `internal/apply/apply.go`, `internal/plan/plan.go`, `scripts/contract.sh`
**Enforced-by:** `internal/ingest/quote190_test.go::TestAnExactSearchBlockLicensesTheLinesItQuotes`
**Invalidates:** None for a native `@@` plan, an apply_patch document, or a SEARCH block that does not match exactly once: each is licensed by the read ledger as now.
**Served-path change:** a `--format=search_replace` block whose SEARCH side matches the file exactly once, in whole lines, is licensed for the lines it quotes, with no read before it; anything it does not quote (an insertion's neighbour, a line after the range) is judged by the ledger as now.

## Context

The read-before-write guard (ADR-002) licenses a line once mrw has served it. That record holds what a model saw. It has two measured costs.

An external, pre-registered A/B on a private Ruby repository (2026-10-31, mrw v1.79.0, 17 pairs; the bundle stays outside this repository) classified, per file, each re-read in the arm with mrw:
- 90 re-reads after a compaction, against 12 in the native arm. The compaction hook (ADR-175) empties the ledger, and every file the agent meant to edit had to be read again.
- 26 re-reads of a file the agent had already read with the host's own `Read` tool, which never reaches the ledger. ADR-191 takes this one.

Claude Code's own Edit tool asks no read: its `old_string` must match the file exactly once, and that quote is the proof that the model knows what it is replacing. mrw already compiles that shape: an Aider SEARCH/REPLACE block (ADR-051). ADR-051 ruled that the match is a location, not a licence, so a SEARCH block still needs a read.

An exact, unique, whole-line quote of every line a hunk replaces shows more than the ledger does. The ledger shows lines were served at some point. The quote shows the model holds their current text now. After a compaction, the quote is the only one of the two that is still true.

Owner direction, 2026-11-02: "the idea is to hide as much from agent and do all that we can with static tools, we have to think a few steps into the future for the agent"; then "draft ADRs for both after the release".

**Audit of the class**: *every write input that names the old text of the lines it changes*. Enumerated with `rg -n 'old side|SEARCH|old_string' internal/ingest internal/plan --glob '!*_test.go'` on 2026-11-02: two inputs.
- `--format=search_replace`: its SEARCH side is the whole old text. Decided here.
- `--format=apply_patch`: it compiles context and `-` lines into the replaced range, so its old side does quote what it replaces. It is left out of this record all the same: one input proves the rule first (Out of Scope).
- A native `delete` whose body lists the lines it expects to remove: apply checks each removed line against that body (`internal/apply/apply.go`, the delete check). It is a quote too, and it keeps the ledger requirement here: a native plan's licence stays exactly the ledger's.
- A native plan's `anchor=` quotes a substring of one line, not whole lines, so it is not a quote of what is replaced.

## Existing Primitives Audit

- `CompileSearchReplace` (`internal/ingest/searchreplace.go:22`) finds the unique exact match and returns plan TEXT that `plan.Parse` reads like any caller's. ADR-030 holds that every production `apply.Input` comes from Parse, and ADR-051 kept that door. So the mark goes in that text, as an option no caller can forge: `quote=` carries a MAC over the path, the range and the quoted lines, keyed by a random secret made once per mrw process. A caller who types `quote=` cannot produce a valid MAC, and Parse stays the only door.
- The ledger check in `apply` (`Options.Seen`, ADR-002) is where a quoted hunk skips the check for the lines it quotes. ADR-005's file-wide licence for a file mrw just wrote is the precedent for a licence that does not come from a read.
- `sha=` and the file-changed check stay as they are. A quote is compared with the file at apply time, under the write lock (`internal/writer`). The ingest test helper calls `apply.Apply` directly, without that lock, so T1 tests through the locked writer path.
- `covered()` passes a file that has no observation at all (`internal/apply/apply.go`), and ADR-052's line-after check runs only when there is one. A quoted hunk on a never-read file would therefore skip the line after silently. Decision 2 closes that.

## Decision

1. **A SEARCH block that matches exactly once, in whole lines, licenses the lines it quotes.** The compiler writes each hunk with `quote=MAC`, where MAC is computed with this process's secret over the path, the range and the quoted lines. Apply verifies the MAC, then, under the write lock, re-runs the match on the locked file: the quote must still match exactly once, at the compiled range, byte for byte. Only then does it skip the ledger check, for those lines only. A `quote=` with a bad MAC, or one that reaches another process, is refused as an unknown guard.
2. **ADR-052's line after a multi-line replace is met only by a quoted line.** A multi-line block is accepted when its SEARCH and REPLACE end in the same line, which is the closer the agent kept. The compiler then replaces the lines before that kept line, and the kept line is the quoted line after. Otherwise the line after needs the ledger, as now. A quoted hunk never takes the no-observation path in `covered()`.
3. **Nothing else is licensed by it.** An unquoted sibling hunk in the same file, a line outside the quote, and a range the compiler would widen past the quote are all judged by the ledger as now. A block that matches zero times or more than once is refused as now.
4. **Native `@@` plans, `anchor=`, a native `delete` with an expected-removal body, and apply_patch are unchanged.** MCP's `format: search_replace` gets the same rule through the same compiler.
5. **The guide says so.** `mrw instructions` names SEARCH/REPLACE as the form that needs no read: after a compaction, or for a file the agent saw with the host's own tools.

**What would make this decision wrong:** a model that quotes text it does not hold and happens to match exactly once. Its quote then licenses a write it would have made blind. The match is exact and in whole lines, so a wrong guess fails rather than lands. The paid run in the Follow-ups measures refused and misapplied writes against the native arm.

## Alternatives Considered

- **Keep `--forget`, and keep each file's sha.** On a file whose sha is unchanged, a write would then need only `anchor=`. ADR-175 rejected an epoch in the ledger as a format change (ADR-150). That would also be one, and `anchor=` quotes a substring, not the lines replaced.
- **Stop forgetting after a compaction.** Rejected: ADR-175 measured two writes to lines the model no longer held.
- **License any `anchor=` match.** Rejected: an anchor proves one substring of one line, and a multi-line replace would write over lines nobody quoted (ADR-035).
- **Compile to structured hunks and skip Parse.** Rejected: ADR-030 requires every production input to come through Parse, and ADR-051 rejected exactly this; the MAC keeps the door.
- **Native `@@` hunks with a `was=` body.** Possible, but it is a second grammar for what SEARCH/REPLACE already says, and models already emit SEARCH/REPLACE.

## Component / Boundary Impact

`internal/ingest` and `internal/apply` (engine, owned here), and `internal/plan` for the hunk's quoted mark. ADR-002's rule changes for one input.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `--format=search_replace`, MCP `format: search_replace` | an exact unique block needs no prior read for the lines it quotes | T1 | agents after a compaction, or after a host read |
| `mrw instructions` | says so | T1 | agents |
| contract §328 | new rows | T1 | CI |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** after a compaction, and for a file the agent saw with the host's tools, a quoted edit lands without a read: the shape of the 90 and 26 re-reads above.
- **Negative:** one input no longer goes through ADR-002's ledger. The guarantee there becomes "the model quoted it exactly", not "mrw served it".
- **Neutral:** a native plan behaves exactly as before.

## Out of Scope

- apply_patch's `-` lines (deferred: docs/adr/BACKLOG.md)
- The measurement against native (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A model quotes lines it does not hold and matches | Low | Med | exact, whole-line and unique; the paid run counts misapplied writes |
| The quoted mark leaks to a hunk the block did not quote | Low | High | T1's mutants: mark on, quote mismatched; the line after the range still needs the ledger |

## Rollback

Revert T1; a SEARCH block needs a read again.

## Follow-ups

- [ ] A pre-registered paid run against native on the release that carries T1 and ADR-191 T1 (BACKLOG.md)
