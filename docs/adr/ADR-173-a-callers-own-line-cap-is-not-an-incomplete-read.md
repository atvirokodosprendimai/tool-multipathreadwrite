# ADR-173: A caller's own `--max-lines` cap is not an incomplete read

**Status:** Proposed
**Date:** 2026-10-24
**Amended:** 2026-10-24, before acceptance, from a Codex xhigh challenge (5 findings, each checked against the code): ADR-033's cap-to-exit-1 clause is named in Invalidates; the five existing expectations of exit 1 (contract `:388`, `:394`, `:4363`, `cmd/mrw/stdin161_test.go:25`, `:58`) migrate in T1; the receipt's `problems` count is asserted unchanged; core rule 5 changes unconditionally; a search that matched nothing keeps `no match:`.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-033, ADR-117, ADR-146, ADR-165, ADR-111, README.md, AGENTS.md, internal/guide/guide.go
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/stdin.go`, `internal/read/read.go`, `internal/guide/guide.go`, `scripts/contract.sh`
**Enforced-by:** None — until T1 lands, which adds `cmd/mrw/maxlines173_test.go::TestACallersOwnLineCapIsNotAnIncompleteRead`
**Invalidates:** ADR-033's clause that a capped read, a cap of zero included, exits non-zero (its contract §14/§69 rows) — the cap's meaning and its licensing stand; and ADR-146 Decision 4 ("the exit codes do not change") for this one case: a read whose every unserved range was withheld by the caller's own `--max-lines` exits 0, not 1. ADR-146's prefixes, and every other exit 1 of `read`, stand.
**Served-path change:** `mrw read … --max-lines N` that served everything it was asked for except what the cap withheld exits 0 instead of 1; the WITHHELD lines are still printed and still unlicensed. A read that also missed a range, or reached `--max-chars`, exits 1 `incomplete:` as today.

## Context

An outside A/B on a large Rails monolith (Rubie tech summary v1, 2026-10-24, mrw v1.74.0, 17 paired Claude Code runs) filed this as MRW-05: 3 of the 11 failed `mrw read` calls were exit 1 from the caller's own `--max-lines`, and the commands chained after them with `&&` did not run. Reproduced on v1.75.0, 2026-10-24: `mrw read big.rb --max-lines 3` prints `!! 197 more line(s) withheld: --max-lines reached` and exits 1, `mrw: incomplete: 1 range(s) could not be served`.

The cap is what mrw teaches in place of `| head` (README `:212`, `mrw instructions` `internal/guide/guide.go:111`, AGENTS.md `:228`), and recipe 4 ends `mrw read 'f.go:/P/' -C 3 --max-lines 40` (`guide.go:51`). `head` exits 0 when it stops early; mrw's replacement exits 1 for the same request, so the recipe that is meant to keep a chain honest breaks it whenever the cap does its job.

Zy chose this record on 2026-10-24 — *"A. Guide cost (Recommended), B. Bounded fixes, C. Contract ADRs"* — bundle C being this record, ADR-174 and ADR-175.

How the count arises (read 2026-10-24): `read.Run` counts a span the cap withholds whole (`internal/read/read.go:832-840`) and a span the cap cuts (`:896-899`) as `problems`, the same counter a missed pattern increments (`:780-783`); `cmd/mrw/main.go:1388` and `cmd/mrw/stdin.go:104-114` exit 1 on any problem. ADR-146 kept exit 1 for every incomplete read and rejected a new exit code (its Alternatives: "breaking for scripts that compare with 1, a major version for two reports").

## Existing Primitives Audit

- `problems` and the WITHHELD lines (`read.go:832-899`): kept as they are; this record only separates, inside the same loop, how many problems were the cap's.
- `ADR-146`'s `incomplete:` kind line: kept for every read that still exits 1.
- `--max-chars` (ADR-165): a second budget in the same loop. Not changed (Decision 2).
- `mrw_read`'s `problems` count and `max_lines` (ADR-117): MCP has no exit code; unchanged (Decision 3).

## Decision

**1. The caller's own line cap is not a failure.** `read.Run` reports, beside `problems`, how many of them were spans the `--max-lines` cap withheld or cut. `mrw read` (named specs, `--grep`, `--ast-grep`, `--files-from`, and `mrw read -`) exits 0 when every problem it counted was the cap's and nothing else is missing. Everything else keeps today's exit and kind line: an addressed pattern that matched nothing, a start past the end, an UNREADABLE or REFUSED path, or `--max-chars` reached exits 1 `incomplete:`, its count still including the cap's ranges; a `--grep` or `--ast-grep` search that matched no file exits 1 `no match:` (ADR-146 Decision 1, `cmd/mrw/main.go:1309`, `:1320`; `mrw read --grep z --max-lines 0 -` likewise). The WITHHELD and `withheld: --max-lines reached` lines are printed exactly as now; withheld lines license nothing (ADR-117 Decision 1), so a write to one is refused as before. `--max-lines 0` (headers only) is the same case and exits 0.

**2. `--max-chars` keeps exit 1.** It exists for a host whose output limit would cut the answer where it likes (ADR-165 Context): the caller cannot know in advance which files or ranges it drops, a walk can lose whole files to it, and ADR-165 Decision 1 made "a cut answer must never read as complete" its rule. `--max-lines` is a per-spec count the caller chose, whose cut point the caller can predict exactly, which is why `head` exits 0.

**3. Receipts and MCP are unchanged.** `mrw read --json`'s `receipt.problems` and `mrw_read`'s `problems` keep their meaning — ranges not served, the cap's included (ADR-111: a key keeps its meaning) — and no key is added: a CLI caller reads the exit code, and `mrw_read` has none.

**4. Told where callers look, and released with a loud note.** README's `--max-lines` row and exit-code sentence, AGENTS.md's cap sentence (`:228-229`), `mrw instructions` (`guide.go:111`, the "A read exits 1 when …" clause loses "lines --max-lines withheld"), and core rule 5 (`guide.go:130`, which `mrw skill` embeds at `:155`): "A read exits 1 when a range cannot be served" gains "— a range your own --max-lines withheld is not one". A script that relied on exit 1 to learn that the cap was reached reads the WITHHELD line instead; the release notes say so first.

**What would make this decision wrong:** a caller that treats exit 0 as "I saw the whole range" and edits a withheld line. The ledger refuses that edit, so the cost is a refusal, not a wrong write.

## Alternatives Considered

- **Keep exit 1, document `|| true`.** Rejected: it hides every other incomplete read in the same chain.
- **A new exit code for "capped".** Rejected for the reason ADR-146 rejected a new code: breaking for every script that compares with 1, and a major version.
- **An opt-in flag (`--cap-ok`) that keeps today's default.** Offered to Zy: not breaking, but the recipe and every guide line must then teach a second flag, and the agent that hit this followed the recipe as written.
- **Also exit 0 on `--max-chars`.** Rejected (Decision 2).
- **Version: v1.76.0 with the change leading the notes (as ADR-170 shipped in v1.75.0), or v2.0.0.** Zy's call; the record recommends the minor with the loud note, released together with ADR-174 if both are accepted.

## Component / Boundary Impact

- `internal/read` (engine package, owned here): the cap's problems are counted apart from the others. No ownership change.
- `cmd/mrw`: the exit decision in the read action and in `mrw read -`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --max-lines N` exit code | 0 when only the cap withheld; 1 otherwise | `cmd/mrw` | shell callers, `&&` chains, recipes 4 and 9 |
| `mrw instructions`, README, AGENTS.md | the exit sentence | `internal/guide`, docs | agents |
| `receipt.problems`, `mrw_read` `problems` | unchanged | — | adapters |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `read.Run`'s count of cap-withheld problems | T1 | T1 | No (internal) |

## Implementation

`tasks/README.md`: T1 (the count, the exit, the words). One pull request.

## Consequences

- **Positive:** the cap does what `head` did in a chain; recipe 4 no longer breaks the `&&` it sits in.
- **Negative:** a script that read exit 1 as "the cap was reached" must read the WITHHELD line; exit codes are a contract, so this is a breaking change in a minor version if Zy chooses so.
- **Neutral:** nothing more is licensed; MCP is untouched.

## Out of Scope

- A partial batch read (one missed address among served ones) exiting 0 — MRW-06 in the same report (permanent: boundary: a range the caller asked for and did not get is what exit 1 `incomplete:` means, ADR-146; the caller did not choose that cut)
- `--max-chars` (permanent: boundary: Decision 2)
- A new receipt key for capped ranges (permanent: boundary: Decision 3; the exit code carries it on the CLI)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A script relied on exit 1 for a reached cap | Low | Low | the WITHHELD line is unchanged; the release notes lead with it |
| The cap's count drifts from `problems` (a new withholding site forgets to count) | Low | Med | T1's test pairs a capped-only read (0) with capped + missed (1) and capped + `--max-chars` (1) |

## Rollback

Count the cap's problems as problems again (one condition); nothing persistent changes.

## Follow-ups

- [ ] none at authoring
