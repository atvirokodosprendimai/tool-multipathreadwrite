# ADR-017: The MCP surface can find what it serves

**Status:** Proposed
**Date:** 2026-09-04
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-007 (owns `--grep` and `read.Walk`, reused unchanged), ADR-010 (owns the two-tool MCP shape this extends rather than widens), ADR-011 (owns `MaxResultChars`, the bound a tree walk must respect), ADR-014 (owns paging, and the dead end this record must not reintroduce), ADR-016 (its routing text becomes FALSE on this branch and moves in the same commit)
**Governs:** `internal/mcp/**`
**Enforced-by:** `internal/mcp/tools_test.go::TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd`
**Invalidates:** ADR-016's Decision 1 in one clause. Its shipped `instructions` and both tool descriptions say the CLI *"also has --grep"*, and that stops being true here. This is not a drafting slip to fix later: §50 asserts every flag the routing names against the help of the subcommand it binds to, so the sentence and this behaviour cannot both ship unchanged. ADR-016's refusal of parity was scoped to itself and said so; this record is the growth that line anticipated, not a reversal of it.
**Served-path change:** a caller with no shell can point mrw at a folder it cannot enumerate and get back the sites — and when there are too many to serve, it gets the addresses instead of a refusal.

## Context

**The analyst cannot enumerate the files.** M, 2026-09-04, naming the population this is for:
*"the potential here IS HUGE, for analysts reading and writing mega documents into csv files …
humans aren't structured in their daily work … this is the single biggest feature that we ship only
for coders but not desktop users."*

A coder with a shell finds sites with `rg -l` and pipes them into `mrw read --files-from -`. A Claude
Desktop user has no shell. Over MCP today, `mrw_read` takes specs and nothing else, so the caller
must already know which files to name. `path:/regexp/` finds the site *within* a file it was given;
nothing finds the *files*. For a folder of documents, that is the whole task.

**The gap is one flag, not the CLI's flag surface.** BACKLOG's Desktop entry already refuses the
parity framing: `--check` is a Go-test runner and means nothing for a folder of CSVs, and
`iter`/`seen`/`stats` are introspection this population has no use for. That leaves `--grep` — and
`--files-from`, which turns out not to be a gap at all (below).

**`--files-from` is meaningless over MCP, by its own documented rationale.** `specList`'s comment
(`cmd/mrw/main.go:1146`) says what it is for:

> It exists so any searcher composes with mrw in one call … The shell quoting that a composed argv
> needs is the failure this avoids — a newline-separated list from `grep -rl` collapses into ONE
> argument under ordinary word splitting.

Over MCP there is no argv, no word splitting and no quoting: `specs` is already a JSON array of
strings, which is precisely what `--files-from` exists to reconstruct. Shipping it would be cargo,
and this is a stronger reason than the one BACKLOG gives for `--check` — it is the tool's own
documented purpose rather than a judgement about a population.

**The walk is not free, and nothing had measured it.** `read.Walk` reads every file under the root to
match, and accumulates one `Spec` per matching file with a range per matching line. `WalkOptions`
carries `Pattern` and `Exclude` and no bound of any kind. ADR-014 measured MCP peak RSS at ~12x the
*served* bytes, but a walk's cost is not served bytes. Measured 2026-09-04 at `2e6b7e7`, 20,000 files
/ 78 MB, darwin:

| Request | Matches | Wall | Peak RSS |
|---|---|---|---|
| `--grep ZZZNOMATCH` (walk only) | 0 files | 0.67 s | 13.4 MB |
| `--grep NEEDLE` | 5,000 files, 1 line each | 1.75 s | 16.3 MB |
| `--grep row` (worst case) | 20,000 files, 600,000 lines | 4.24 s | 41.9 MB |

So the walk itself is cheap and roughly flat — it streams, and 78 MB of input costs 13 MB — while the
*spec slice* is what grows: 600,000 ranges cost about 26 MB over the floor, ~45 bytes each. That is
linear in matches and bounded by nothing. It is acceptable at this scale and it is not a number to
extrapolate: a root pointed at a large document folder is the case this record exists to serve, and
the honest position is that the cap bounds what is SERVED and nothing bounds what is FOUND.

## Existing Primitives Audit

- **`read.Walk` (ADR-007):** already returns `[]Spec` and `[]Problem` — it FINDS without serving.
  **Reused unchanged, and it is what makes this record small**: the CLI calls `Walk` then `read.Run`,
  and MCP can call the same two in the same order. The separation this record needs already exists.
- **`read.Run` and the capped writer (ADR-011-T3):** **reused unchanged.** The cap fires on the serve,
  which is exactly where the bytes are.
- **ADR-014's `firstPage`:** **NOT reusable here, and that is the design problem.** It requires
  `len(specs) == 1` and an open-ended spec (`tools.go:389`). A grep produces many specs across many
  files, so an oversized grep falls through to the flat refusal — which is precisely the dead end
  ADR-014 removed, reappearing through a new door.
- **The CLI's two refusals (`main.go:452`, `main.go:499`):** `--grep` with `--files-from`, and `--grep`
  with a spec that carries a range — *"two answers to one question"*. **Mirrored, not reinvented**: a
  grammar the two surfaces disagree on is the class ADR-016 exists to prevent.

## Decision

**1. `mrw_read` gains an optional `grep`, and an optional `exclude`.** Not a third tool: `--grep` is a
flag on `read` in the CLI, and ADR-010's two-tool shape is a boundary worth keeping. When `grep` is
present, the server calls `read.Walk` over the given paths (the root when none are given) and then
serves the resulting specs exactly as it serves caller-supplied ones — same cap, same ledger, same
recording.

**2. An oversized grep returns the MATCH INDEX, not a refusal and not a truncation.** When the served
content would exceed the cap, the result carries every match as a bare `path:N` spec with no content,
plus the count. This is a first-class answer rather than a consolation: the index IS the language of
the next call, it is roughly fifty times smaller than the content it describes, and *finding the
sites* is the thing this record exists to do. The caller then reads the specs it actually wants, each
pageable by ADR-014's existing mechanism.

**3. An index too large to serve PAGES BY FILE.** The index is a list, and a list has a natural
continuation the way a file's lines do: the result carries the first N entries and names the path to
resume after. This is stated as a decision rather than left to the implementation because the
tempting alternative — refuse with a count — is ADR-014's dead end a third time, and the step after
the satisfying part is the one this project keeps dropping. The Enforced-by drives the
overflowing-INDEX case, not only the overflowing-content case.

**4. `--files-from` is not added, permanently.** See Context: it exists to undo shell word-splitting,
and MCP has no shell. `specs` already is the list.

**5. The two CLI refusals are mirrored.** `grep` together with a spec that carries a range is refused
with the CLI's own reasoning. There is no `files_from` to conflict with, per Decision 4.

**6. ADR-016's routing text moves in the same commit.** It says the CLI *"also has --grep"*, in
`instructions` and in both tool descriptions, and §50 asserts each named flag against the help of the
subcommand the advice binds it to. After this, the CLI's remaining exclusives are `--files-from`,
`--check`, and the `check`/`iter`/`seen`/`stats` subcommands. The sentence must be true when it
ships, which is the whole of ADR-016.

**Go/no-go, checked during execution:**

- **No engine change.** `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`,
  `internal/check` and `internal/state` stay byte-identical against a merge-base diff. This record
  calls `read.Walk`; it does not change it.
- **No new dependency**; `go.mod` still declares exactly one requirement.
- **No root change.** Reach is a different question with a different safety profile and is NOT
  touched here — the walk starts at the same root the server already had.
- **`gofmt -l .` empty and `go vet ./...` clean**, in every fence.

## Alternatives Considered

- **A third tool, `mrw_grep`.** Rejected: it splits finding from reading at the tool boundary when the
  engine already composes them, and ADR-010's two-tool shape is a deliberate limit. `--grep` is a flag
  on `read` in the CLI for the same reason.
- **Serve grep matches and refuse when too large.** Rejected — it is ADR-014's dead end with a new
  trigger, and for this population it fires on the ordinary case rather than the exotic one.
- **Truncate the match list silently.** Rejected, restating ADR-011: a part that arrives looking like
  the whole is the silent wrong answer this tool exists to refuse.
- **Return only the index, always.** Tempting for its simplicity and rejected: for a grep that fits,
  making the caller take a second round trip to read what would have fitted spends the context this
  tool exists to save.
- **Ship `--files-from` for parity.** Rejected on the tool's own documented rationale — see Decision 4.
  Parity is not the goal; a caller that can do the job is.
- **Bound the walk (a max-files option).** Not taken HERE, and named rather than ignored: the
  measurement above says the walk is cheap and the spec slice is what grows, so a bound belongs with
  the root model — a walk is only alarming when the root is large, and the root is ADR-018's subject.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/mcp` | The transport, its bound, and now the discovery it can offer | Yes |
| `internal/read` | Unchanged — `Walk` already finds and `Run` already serves | Untouched |
| `internal/seen` | Unchanged — a grep records exactly what it served | Untouched |

## Wiring & Contract Changes

| Change | Kind | Consumers |
|---|---|---|
| `mrw_read` accepts optional `grep` and `exclude` | Public contract, additive | MCP callers |
| An oversized grep returns a match index and a count | Public contract, additive | Any host reading `structuredContent` |
| An oversized index pages by file | Public contract, additive | Same |
| `grep` with a ranged spec is refused | Public contract, additive refusal | Callers who would otherwise get an unspecified answer |
| ADR-016's routing text no longer claims `--grep` is CLI-only | Correction to shipped prose | Every host reading `instructions` |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| grep over MCP, its index degradation and its refusals | T1 | T2 | No — new |
| the taught form, and ADR-016's corrected routing | T2 | — | No — corrective |

## Implementation

Two tasks. T1 builds it and proves the index path by driving an overflow; T2 teaches it and repairs
ADR-016's sentence in the same commit, because a surface that teaches what the binary does not do is
the defect ADR-012 was written about and ADR-013 and ADR-016 each repeated.

## Consequences

- **Positive:** a caller with no shell can find its sites, which is the difference between mrw being
  usable and unusable for the Desktop population.
- **Positive:** an oversized grep answers with addresses instead of failing, and addresses are what
  the next call takes anyway.
- **Negative:** the walk's cost is linear in matches and bounded by nothing. Measured acceptable at
  20,000 files; NOT measured on a large document tree, and the record says so rather than implying
  the number generalises.
- **Negative:** the MCP surface grows, which is a thing ADR-010 deliberately kept small. This is one
  optional argument on an existing tool, and the reasoning for refusing the rest is written down.
- **Neutral:** the CLI is unchanged. It already had this.

## Out of Scope

- `--files-from` over MCP (permanent: boundary: it exists to undo shell word-splitting, which MCP does not have — `cmd/mrw/main.go:1146`)
- `--check` and the `check`/`iter`/`seen`/`stats` subcommands over MCP (permanent: boundary: BACKLOG's Desktop entry — a Go-test runner and introspection are not this population's work)
- Any change to the root, or to how many roots there are (deferred: ADR-018 owns reach; this record is capability only, and the two were split deliberately)
- Bounding the walk (deferred: belongs with the root model — see Alternatives)
- Changing `MaxResultChars` (deferred: still needs the quality curve — ADR-014 Decision 4)
- Any CLI behaviour change (permanent: boundary: ADR-010's go/no-go)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The index path is written but never driven, so an oversized grep silently dead-ends | Med | **High** | The Enforced-by drives a real overflow and asserts an index comes back with a resumable continuation — not that a field exists |
| A grep licenses lines it never served | Low | **Critical** | The serve is `read.Run` through the same recorder; the index path serves NO content and must therefore record nothing, asserted separately |
| ADR-016's sentence ships false | Med | **High** | §50 already asserts every named flag against the CLI's help; the sentence moves in the same commit or §50 goes red |
| The walk is slow or large on a real document tree | Med | Med | Measured at 20,000 files and stated as not generalising; a bound is named in Alternatives and deferred to the root record |
| The two surfaces disagree on the grammar | Low | Med | The CLI's own refusals are mirrored and asserted |

## Rollback

Revert the commits. `grep` is an optional argument nothing depends on, and ADR-016's sentence reverts
with it — the two are in one commit precisely so neither can be reverted alone.

## Follow-ups

- [ ] Measure the walk on a real document tree once the root model exists, and decide whether
      `WalkOptions` needs a bound or whether the root is the only bound that matters.
