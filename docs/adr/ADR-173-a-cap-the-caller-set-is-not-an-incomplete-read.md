# ADR-173: A cap the caller set is not an incomplete read

**Status:** Accepted
**Date:** 2026-10-24
**Accepted:** 2026-10-29 by Zy — "accept, add, merge all, push" (the three records #566, #567, #568 together; "add" is the `capped` receipt key ADR-173 left to Zy)
**Amended at acceptance:** 2026-10-29, on Zy's "add": the receipt key the first draft left to Zy is taken — Decision 3 adds `capped` beside `problems` on the read receipt; the alternative, the Wiring row, T1 and its fence follow.
**Amended:** 2026-10-24, before acceptance, from a Codex xhigh challenge (5 findings, each checked against the code): ADR-033's cap-to-exit-1 clause is named in Invalidates; the existing expectations of exit 1 migrate in T1; the receipt's `problems` count is asserted unchanged; core rule 5 changes unconditionally; a search that matched nothing keeps `no match:`. 2026-10-29, before acceptance, on Zy's direction of that day: `--max-matches` (ADR-179, shipped since) is covered as `--max-lines` is; ADR-179 (g)'s two exit clauses are named in Invalidates; the line numbers and the migration list are re-taken against main `e8a6a3da`; a cap-only read names what its cap withheld on standard error as `mrw: capped: …`; the cap's count travels in `read.Options`, so `read.Run`'s signature and its MCP callers do not change; `internal/read` is named as an engine package this record owns. Then from a Codex xhigh challenge of the amended text (1 finding, checked against the code and reproduced): `incompleteLine`'s `--max-matches` branch stays, because `--max-chars` can drop only the budget's own note, leaving the cap as the one problem while the read still exits 1; that case joins T1's paired cases. A first run of that challenge was killed at host-review's 600 s budget and returned nothing.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-033, ADR-111, ADR-117, ADR-146, ADR-165, ADR-166, ADR-179, README.md, AGENTS.md, docs/agents.md, internal/guide/guide.go
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/stdin.go`, `cmd/mrw/maxmatches179.go`, `internal/read/read.go`, `internal/guide/guide.go`, `internal/mcp/readreceipt.go`, `docs/receipts.txt`, `scripts/contract.sh`
**Enforced-by:** None — until T1 lands, which adds `cmd/mrw/caps173_test.go::TestACapTheCallerSetIsNotAnIncompleteRead`
**Invalidates:** ADR-033's clause that a capped read, a cap of zero included, "exits non-zero by the chain the CLI already had" (its contract §14 and §69 rows) — the cap's meaning and its licensing stand. ADR-179 (g)'s clause that a read whose matches were left unshown "exits 1 `incomplete:` (ADR-146), as `--max-lines` and `--max-chars` do when they withhold", and its clause that "when only the budget withheld the kind line names it (`incomplete: --max-matches N reached, M matching line(s) not shown`)": when only the caller's caps withheld, the read exits 0 and that sentence moves to a `mrw: capped:` line. ADR-146 Decision 4 ("the exit codes do not change") for this one case. ADR-146's two kinds, and every other exit 1 of `read`, stand; ADR-165's `--max-chars` exit 1 stands.
**Served-path change:** `mrw read … --max-lines N` or `mrw read --grep P --max-matches N …` that served everything it was asked for except what those caps withheld exits 0 instead of 1, and its last stderr line is `mrw: capped: …` instead of `mrw: incomplete: …`; the withheld lines are still named in the answer and still unlicensed. A read that also missed a range, or reached `--max-chars`, exits 1 `incomplete:` as today.

## Context

An outside A/B on a large Rails monolith (external tech summary v1, 2026-10-24, mrw v1.74.0, 17 paired Claude Code runs) filed this as MRW-05: 3 of the 11 failed `mrw read` calls were exit 1 from the caller's own `--max-lines`, and the commands chained after them with `&&` did not run.

Reproduced on the installed v1.78.0 (`06075f7`), 2026-10-29, in a scratch directory:

- `mrw read big.txt --max-lines 3` over 50 lines prints `!! 47 more line(s) withheld: --max-lines reached` on stdout and exits 1, stderr `mrw: incomplete: 1 range(s) could not be served`;
- `mrw read --grep x --max-matches 1 .` over three spaced matches prints `-- max-matches 1 reached: 2 more matching line(s) in 1 file(s) not shown` on stdout and exits 1, stderr `mrw: incomplete: --max-matches 1 reached, 2 matching line(s) not shown`.

The cap is what mrw teaches in place of `| head` (README `:237`, AGENTS.md `:242-243`, `mrw instructions` `internal/guide/guide.go:131`), and two recipes end in a cap: recipe 4, `mrw read 'f.go:/P/' -C 3 --max-lines 40` (`guide.go:51`), and recipe 15, `mrw read --grep P --max-matches 40 .` (`guide.go:73`). `head` exits 0 when it stops early; mrw's replacement exits 1 for the same request, so the recipe that is meant to keep a chain honest breaks it whenever the cap does its job.

Zy chose this record on 2026-10-24 — *"A. Guide cost (Recommended), B. Bounded fixes, C. Contract ADRs"* — bundle C being this record, ADR-174 and ADR-175, and the acceptance line of ADR-171, ADR-175 and ADR-176 quotes *"ADR-173 + ADR-174"* among the records chosen that day. The draft was never committed: it stayed in the stash `4f2e05af`, and `docs/adr/BACKLOG.md` (the "ADR-173 (decision before implementation)" entry) asked for it to be rebased against ADR-179's `--max-matches` and `--max-chars` before deciding. On 2026-10-29 Zy decided the direction: **caps the caller set exit 0.** `--max-lines` and `--max-matches` both exit 0 when the only thing withheld is what the caller's own cap withheld; the withheld lines stay unlicensed and are still named on stderr; `--max-chars` keeps exit 1, because the host's output limit sets its cut point and the caller does not choose it (ADR-165); a read that also missed a range, or hit `--max-chars`, keeps exit 1 `incomplete:`; this is a breaking exit-code change and its release note leads. This record stays Proposed until Zy accepts the amended text.

How the count arises (read 2026-10-29 at `e8a6a3da`): `read.Run` counts a span `--max-lines` withholds whole (`internal/read/read.go:1037-1043`), a span the cap cuts (`:1158-1160`) and the matches `--max-matches` left unshown (`:1205-1214`, one problem for all of them) as `problems`, the same counter a missed pattern, an UNREADABLE path and the `--max-chars` budget increment (for instance `:742-743`, `:1028-1029`). `cmd/mrw/main.go:1666-1672` and `cmd/mrw/stdin.go:128-138` exit 1 on any problem, and `incompleteLine` (`cmd/mrw/maxmatches179.go:8-13`) words the kind line, naming `--max-matches` only when it is the one problem. ADR-146 kept exit 1 for every incomplete read and rejected a new exit code (its Alternatives: "breaking for scripts that compare with 1, a major version for two reports").

Where the notices go today: the `@@ A-B  WITHHELD …`, `!! … withheld: --max-lines reached` and `-- max-matches N reached: …` lines are part of the answer on standard output (`reportOn` and `report`, `read.go:730-736`, `:1210`); only the kind line is on standard error. So "still named on stderr" needs a stderr line of its own once the kind line goes: Decision 1's `mrw: capped:`.

## Existing Primitives Audit

- `problems` and the withheld notices (`read.go:1037-1043`, `:1158-1160`, `:1205-1214`): kept as they are; this record only counts, at the same three sites, how many problems were the caps'.
- `read.Options.Unshown` (`read.go:213`, a `*int` the caller passes and `Run` fills): the shape the new count copies, so `Run`'s two return values and its callers in `internal/mcp` (`tools.go:556`, `:1862`) and `internal/curve` do not change. Reuse.
- `incompleteLine` (`maxmatches179.go:8-13`): its `--max-matches` sentence is reused for the `capped:` line, and its first branch stays: a read whose one problem is the `--max-matches` cap still exits 1 when `--max-chars` dropped only the budget's own note (`printf 'hit\nhit\nhit\n' | mrw read --grep hit --max-matches 0 --max-chars 1 -` today ends `incomplete: --max-matches 0 reached, …; --max-chars 1 reached`, Codex challenge 2026-10-29), and that wording is kept.
- ADR-146's kind line, `mrw: incomplete:` / `mrw: no match:`: kept for every read that still exits 1.
- `--max-chars` (ADR-165): a second budget in the same loop. Not changed (Decision 2).
- `mrw_read`'s `problems` and `max_lines` (ADR-117): MCP has no exit code, and its `isError` does not follow `problems`; unchanged (Decision 3).

## Decision

**1. A cap the caller set is not a failure.** `read.Run` counts, beside `problems`, how many of them were the caller's caps: a span `--max-lines` withheld whole, a span it cut, and the one problem `--max-matches` adds when matches remain unshown. The count is returned through a new `read.Options.Capped *int`, as `Unshown` is. `mrw read` (named specs, `--grep`, `--ast-grep`, `--files-from`, `--quiet`, and `mrw read -`) exits 0 when every problem it counted was a cap's, no path was refused, and `--max-chars` was not reached. Its last stderr line is then `mrw: capped: …`: `mrw: capped: --max-matches N reached, M matching line(s) not shown` when the only cap problem is `--max-matches`, otherwise `mrw: capped: N range(s) withheld by --max-lines` (with `; --max-matches N reached, M matching line(s) not shown` added when both withheld). ADR-166's `mrw: nothing was recorded` line, when there is one, stays above it. Everything else keeps today's exit and kind line, its count still including the caps': an addressed pattern that matched nothing, a start past the end, an UNREADABLE or REFUSED path, an outline that did not parse, or `--max-chars` reached exits 1 `incomplete:`; a `--grep` or `--ast-grep` search that matched no file exits 1 `no match:` (ADR-146 Decision 1, `main.go:1541-1565`). On `mrw read -` the cap-only test comes before the stdin no-match test (`stdin.go:135`), and that test is reached only when a problem was not a cap's, so `mrw read --grep z --max-lines 0 -` over input with no `z` still says `no match:` while a match the cap withheld exits 0. The withheld notices on standard output are printed exactly as now; withheld and unshown lines license nothing (ADR-117 Decision 1, ADR-179 (g)), so a write to one is refused as before. `--max-lines 0` (headers only) and `--max-matches 0` (every match counted, none shown) are the same case and exit 0.

**2. `--max-chars` keeps exit 1.** It exists for a host whose output limit would cut the answer where it likes (ADR-165 Context): the caller does not choose the cut, cannot know in advance which files or ranges it drops, a walk can lose whole files to it, and ADR-165 Decision 1 made "a cut answer must never read as complete" its rule. `--max-lines` and `--max-matches` are counts the caller chose, whose cut point the caller can predict exactly, which is why `head` exits 0. A read with a cap and `--max-chars` reached is incomplete.

**3. The receipt says how many problems were the caller's caps.** `mrw read --json`'s receipt and `mrw_read`'s `content[1]` (the same receipt, ADR-165) gain one key, `capped`: an integer, the number of `problems` that were the caller's own caps — a span `--max-lines` withheld whole, a span it cut, the one problem `--max-matches` adds — so `problems - capped` is what the caller did not choose. `problems` keeps its meaning — ranges not served, the caps' included (ADR-111: a key keeps its meaning) — and `capped` is appended to `docs/receipts.txt` as `mcp_read capped` in the change that adds it. So after this record a `--json` read can carry `problems > 0` and exit 0, and the receipt says why: `capped == problems`. `mrw_read` has no exit code and its `isError` does not follow `problems`; unchanged. ADR-179 (b)'s "so `--json` agrees with the exit" is about an outline under `--max-chars`, which still exits 1, and stands. Zy, 2026-10-29: "add".

**4. Told where callers look, and released with a loud note.** README's `--max-lines` and `--max-matches` rows (`:237`, `:242`) and its exit-1 paragraph (`:596-600`, which lists `WITHHELD` among the exit-1 words); AGENTS.md's cap sentences (`:242-243`, `:286-287`) and its exit-code section (the `mrw read` that exits `1` paragraph); `docs/agents.md` (`:145`, `:153`); `mrw instructions` (recipe 15 at `guide.go:72`, the flag line at `:131`) and core rule 5 (`guide.go:152`, which `mrw skill` embeds through `Core()` at `:177`): "A read exits 1 when a range cannot be served" gains "— a range your own --max-lines or --max-matches withheld is not one; it exits 0 and says capped". A script that relied on exit 1 to learn that a cap was reached reads the `mrw: capped:` line or the withheld notice instead; the release notes lead with it.

**5. `internal/read` is an engine package this record owns** (`.claude/rules/lifecycle.md` §6): T1's diff there is the three counting sites and the `Capped` option, named in the PR body with this record, and gets a `go-reviewer` pass before the Codex review. The fence drops `internal/read` from its go/no-go clause and keeps the clause for `internal/apply`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state`.

**What would make this decision wrong:** a caller that treats exit 0 as "I saw the whole range" and edits a withheld line. The ledger refuses that edit, so the cost is a refusal, not a wrong write.

## Alternatives Considered

- **Keep exit 1, document `|| true`.** Rejected: it hides every other incomplete read in the same chain.
- **A new exit code for "capped".** Rejected for the reason ADR-146 rejected a new code: breaking for every script that compares with 1, and a major version.
- **An opt-in flag (`--cap-ok`) that keeps today's default.** Offered to Zy on 2026-10-24: not breaking, but the recipes and every guide line must then teach a second flag, and the agent that hit this followed the recipe as written.
- **`--max-lines` only, `--max-matches` keeping exit 1.** Rejected by Zy on 2026-10-29: both are counts the caller set, and recipe 15 has the same `&&` problem as recipe 4.
- **Also exit 0 on `--max-chars`.** Rejected (Decision 2; Zy, 2026-10-29).
- **Exit 0 with nothing on stderr**, the withheld notice on stdout being the only mention. Rejected: Zy's direction says the withheld lines are "still named on stderr", and a caller that reads only stderr and the exit would lose the one line that said a cap was reached.
- **No receipt key, the exit code and the `capped:` line carrying it.** The first draft's choice, left to Zy; taken the other way on 2026-10-29 ("add"): an adapter reading the receipt has no exit code and no stderr to read the kind from, so `capped` (Decision 3) is the one place it can learn that `problems > 0` was the caller's own cap.
- **Version: v1.79.0 with the change leading the notes (as ADR-170 shipped in v1.75.0), or v2.0.0.** Zy's call; the record recommends the minor with the loud note, released together with ADR-174 if both are accepted.

## Component / Boundary Impact

- `internal/read` (engine package, owned here — Decision 5): the caps' problems are counted apart from the others through `Options.Capped`. No ownership change; `Run`'s signature is unchanged.
- `cmd/mrw`: the exit decision and the stderr line in the read action, in `mrw read -`, and in `incompleteLine`.
No module moves; `docs/architecture.md`'s map is unchanged.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --max-lines N`, `--max-matches N` exit code | 0 when only those caps withheld; 1 otherwise | `cmd/mrw` | shell callers, `&&` chains, recipes 4, 9 and 15 |
| last stderr line of a cap-only read | `mrw: capped: …` instead of `mrw: incomplete: …` | `cmd/mrw` | callers that read the kind (ADR-146) |
| `read.Options.Capped` | new `*int` filled by `Run` | `internal/read` | `cmd/mrw` |
| `mrw instructions`, `mrw skill`, README, AGENTS.md, `docs/agents.md` | the exit sentences | `internal/guide`, docs | agents |
| `receipt.problems`, `mrw_read` `problems` | unchanged; `capped` added beside them (`docs/receipts.txt`, `mcp_read capped`) | `internal/mcp` (the read receipt) | adapters |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `read.Options.Capped` | T1 | T1 | No (internal) |

## Implementation

`tasks/README.md`: T1 (the count, the exit, the words, the migrated expectations). One pull request.

## Consequences

- **Positive:** a cap does what `head` did in a chain; recipes 4 and 15 no longer break the `&&` they sit in.
- **Negative:** a script that read exit 1 as "a cap was reached" must read the `capped:` line or the withheld notice; exit codes are a contract, so this is a breaking change in a minor version if Zy chooses so. A `--json` read can now carry `problems > 0` with exit 0 (Decision 3).
- **Neutral:** nothing more is licensed; MCP is untouched.

## Out of Scope

- A partial batch read (one missed address among served ones) exiting 0 — MRW-06 in the same report (permanent: boundary: a range the caller asked for and did not get is what exit 1 `incomplete:` means, ADR-146; the caller did not choose that cut)
- `--max-chars` (permanent: boundary: Decision 2)
- `mrw_read` (permanent: boundary: no exit code; `problems` and `isError` unchanged)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A script relied on exit 1 for a reached cap | Low | Low | the withheld notice is unchanged and `capped:` names it; the release notes lead with it |
| The caps' count drifts from `problems` (a new withholding site forgets to count) | Low | Med | that case exits 1, the safe side; T1's test pairs each capped-only read (0) with capped + missed (1) and capped + `--max-chars` (1) |
| Migrating the exit-1 expectations moves other records' test locks (ADR-179 T7, ADR-161 T8, any whose Tests table names an edited file) | High | Low | T1 runs `adr-lint` over every record before committing and relocks what it names |

## Rollback

Count the caps' problems as problems again (one condition in each exit site) and restore the `incomplete:` wording; nothing persistent changes.

## Follow-ups

- [ ] none at authoring
