# ADR-146: A read that exits 1 says which kind

**Status:** Accepted
**Accepted:** 2026-10-11 on Zy's answer "Additive prefix (Recommended)" to the question how to answer the adopter survey's "exit 1 means both no match and 'N range(s) could not be served'" (docs/adr/BACKLOG.md, 14-session survey of 2026-10-09)
**Date:** 2026-10-11
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-116, ADR-122, ADR-140, docs/adr/BACKLOG.md
**Invalidates:** None — the exit codes stay as they are; this adds a stable word to the line that explains exit 1
**Governs:** `cmd/mrw/main.go`, `README.md`, `AGENTS.md`, `scripts/contract.sh`
**Enforced-by:** `cmd/mrw/readexit146_test.go::TestAReadThatExitsOneSaysWhichKind`
**Served-path change:** the last stderr line of a `mrw read` that exits 1 begins `mrw: no match: ` when it found nothing and nothing it was asked for was refused, and `mrw: incomplete: ` when a requested range or path was not served. The words after the prefix are the ones printed before. Exit codes are unchanged.

## Context

`mrw read` exits 1 for two different things: a `--grep` or `--ast-grep` that matched no file (benign, as grep's own exit 1), and a read where a requested range or path was UNREADABLE, REFUSED or WITHHELD (part of what was asked for is missing). The survey of 2026-10-09 had two Windows sessions ask to tell them apart from the exit alone, or from a stable stderr prefix. The lines already differ in words (`no file matched /P/` and `N range(s) could not be served`), but nothing documents them as a contract, and one case is wrong: a `--grep` that matches no file while another path it was given is refused ends `no file matched`, hiding that part of the search was never made.

A real split of the exit code would be a breaking change to a documented contract (AGENTS.md: changing one is a breaking change) on the word of two sessions, who also accepted the prefix. Zy chose the prefix.

**Audit of the class** — *a read exit 1*: `mrw read --grep 'cli.Exit\(.*, 1\)' cmd/mrw/main.go` names three returns in `readCmd` (the `--grep` no-match, the `--ast-grep` no-match, and the problems count); `write`'s exit 1 is a failed hunk and carries a receipt, and is out of scope. Counted 2026-10-11.

## Existing Primitives Audit

- **The three `cli.Exit(…, 1)` returns in `readCmd`** — the whole change is their message text.
- **`refusals`** — already in scope at the no-match returns (the REFUSED lines printed above them); its length decides which prefix.
- **ADR-140's hint suffix** — kept after the words, unchanged.

## Decision

1. **`no match: ` begins the line when the search found no file and `refusals` is empty.** `mrw: no match: no file matched /P/`, with ADR-140's hint where it applies.
2. **`incomplete: ` begins every other exit-1 line of `read`.** `mrw: incomplete: N range(s) could not be served`, and for a search that matched no file while paths were refused, `mrw: incomplete: no file matched /P/, and N path(s) were refused (see the REFUSED lines)`.
3. **The two prefixes and the rule are documented** in README (the sentence under the exit-code table) and AGENTS.md (the exit-code paragraph), and a contract row asserts them. A caller that must tell the cases apart reads the first word after `mrw: `; a caller that does not is unaffected, since every earlier phrase survives inside the line.
4. **The exit codes do not change**, and neither does stdout.

## Alternatives Considered

- **A new exit code for incomplete reads** — rejected by Zy: breaking for scripts that compare with 1, a major version for two reports.
- **A `reason` in a read `--json` receipt** — rejected: `read` has no receipt; adding one is a larger surface than a word.
- **Document the existing phrases and change nothing** — rejected: the refused-while-no-match line would stay wrong.

## Component / Boundary Impact

`cmd/mrw` (three message strings). No engine package changes; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| last stderr line of a `mrw read` exiting 1 | a leading `no match:` or `incomplete:` | T1 | scripts and agents |
| `README.md`, `AGENTS.md` | the two prefixes stated | T1 | every agent |
| `scripts/contract.sh` | §250 | T1 | CI Linux |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a caller tells a benign empty search from a read that lost part of its answer by one word, and a search that skipped a refused path is no longer called a plain no-match.
- **Negative:** a script that compared the whole line to `no file matched /P/` now sees a prefix; every substring search still matches.
- **Neutral:** `write` and the MCP surface are unchanged.

## Out of Scope

- A distinct exit code for incomplete reads (permanent: boundary: Zy chose the additive prefix; a real split is a major version and needs a new record)
- `write`'s exit 1 (permanent: boundary: a failed hunk, with a receipt)
- The MCP `mrw_read` (permanent: boundary: it has no exit code; its `problems` count already says how many requested ranges were not served)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a script anchors on `^mrw: no file matched` | Low | the script stops matching | every phrase survives; the prefix is documented before it ships |

## Rollback

Revert T1: the lines read as before. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up beyond the boundaries above.
