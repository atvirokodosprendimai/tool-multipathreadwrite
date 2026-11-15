# ADR-197: An anchor miss names the lines that hold the anchor

**Status:** Accepted
**Accepted:** 2026-11-15 by Zy — asked *"Do you accept ADR-197 as written?"* on the narrowed draft (the anchor miss alone, the failed-check decision withdrawn to BACKLOG.md), answered *"accepted"*.
**Date:** 2026-11-15
**Amended:** 2026-11-15, before acceptance, from an adr-challenger pass and a Codex xhigh challenge (each finding checked against the code): the second decision of the first draft, a failed check saying where its output points, is withdrawn to BACKLOG.md because the lines a write produced are not available outside `internal/apply` and a bare file name in Go test output does not prove which file was meant; the class audit is keyed on the refusal sites and gained three members; one kill criterion instead of two that disagreed; the scan is committed.
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-028, ADR-035, ADR-052, ADR-118, ADR-194, ADR-009, docs/adr/BACKLOG.md
**Supersedes:** none
**Governs:** `internal/apply/apply.go`, `scripts/contract.sh`, `README.md`, `AGENTS.md`, `docs/bench/anchor-miss-study/scan.py`
**Enforced-by:** `internal/apply/anchor197_test.go::TestAnAnchorMissNamesTheLinesThatHoldIt`
**Served-path change:** a hunk refused because its `anchor=` is not in the addressed line also names up to five lines of the file that do hold the anchor, or says no line does. The exit code, the verdict and the words the refusal has today are unchanged; the new clause follows them.

## Context

**Where this came from.** `OnlyTerp/turnpack` (an MCP server, read at b14330d on 2026-11-15; memoryd `tool-multipathreadwrite/ebbbdf89`, `f2aa56b2`) answers a miss with where the text is. Zy, 2026-11-15: *"we need only what is beneficial to us here, not to exact copy"*. This record takes the one idea the transcripts support and the code can carry today.

**What mrw says today**, measured 2026-11-15 on `mrw` built from 663b047e, in a scratch root holding `f.go` (`func invoiceTotal(s int) int {` at line 2), every line read:

```
FAIL f.go 2-3 replace (plan line 1): anchor "func invoiceTotal(sub int) int {" not in line 2: "func invoiceTotal(s int) int {"
```

It quotes the addressed line (ADR-028) and does not say whether any line holds the anchor. ADR-035 made the anchor required on a multi-line replace because a wrong range is the common mistake; when the range is the wrong part, the caller needs the right line number and gets a quote of the wrong line.

**What the transcripts say.** `python3 -I docs/bench/anchor-miss-study/scan.py`, run 2026-11-15 over the 116 top-level Claude Code transcripts under `~/.claude/projects/` on this machine (subagent transcripts excluded): 140 tool results hold `anchor "…" not in line N`, in 25 transcripts; the next tool call was a read after 104 of them and something else after 36. A sample of 42 parsed by file named real project files in 41 (this repository, quality-harness, db-doctor and others). Limits, as the script's header says: a result is not an incident; nothing shows the refusal caused the read; and a read after a miss is sometimes one the guard requires whatever the refusal says, since a hunk moved to another line needs that line served (ADR-002) and a multi-line replace needs the line after its end (ADR-052). The other refusals were counted once with an uncommitted scan of the same shape: a `search_replace` or `apply_patch` block that matched nothing, 7; a pattern that matched no line, 6; a failed check, 49.

**Audit of the class**: *every refusal that names a place in a file and not where the thing the caller wrote is.* Read from the refusal sites, `git grep -n 'fail(h, "\|Errorf("' -- internal/apply internal/ingest ':!*_test.go'` on 663b047e, the hits that name a place:
- `anchor=` not in the addressed line, `internal/apply/apply.go:2655` (`anchorMiss`): decided here.
- `expected removal differs at line N: plan says …, file has …`, `apply.go:1942`: the anchor miss's twin for a delete's expected body. Not counted in any transcript scan; deferred (Out of Scope).
- A `search_replace` SEARCH or an `apply_patch` old side that matched nothing, or matched N times without naming the lines, `internal/ingest/searchreplace.go:118, 134`, `applypatch.go:251, 259`: deferred with their count (Out of Scope).
- A start or end pattern that matched no line, `apply.go:1652, 1690`: left out; a regular expression has no text to be near. The pattern that matched several lines already names them (`apply.go:1657`).
- A failed check, which names where it failed and not whether the write produced that line: deferred with what the challengers found (Out of Scope).

## Existing Primitives Audit

- `anchorMiss` already scans the original for the anchor (`apply.go:2661-2665`) when the #589 quote advice might apply. This record runs that scan on every miss and reports what it finds.
- Naming line numbers the caller has not been served has precedent: the ambiguous-pattern refusal lists every matching line, and `occurrence=` names the unread ones (`apply.go:1640, 1657`; ADR-118). A line number is not content, and the read guard still judges the re-addressed hunk.

## Decision

1. **An anchor miss names the lines that hold the anchor.** When `anchor=` is not in the addressed line and other lines of the original contain it as written, the refusal ends `; the anchor is in line N`, or `lines N, M, …` with the first five in file order and `and K more` after them. When no line contains it, the refusal ends `; no line of PATH holds it`, except in the #589 case (the addressed line holds the anchor once its double quotes are gone), where the quote advice stands alone as now.
2. **It licenses nothing.** The clause quotes no line content. A hunk re-addressed to a named line is refused by the read guard if that line was not served.
3. **Nothing else changes**: the exit code, the hunk's `reason` key, an anchor that matches, and every other refusal.
4. **What would make this decision wrong.** Thirty days after the release that ships it, `scan.py --since <release date>` is run again. If half or more of the anchor misses are still followed by a read, the clause is words nobody uses and goes back to Zy. The share is an upper bound on the reads a refusal can save (Context), so a share under half is the bar, not a drop to zero.

## Alternatives Considered

- **A failed check says where its output points** (the first draft's second decision). Withdrawn: the lines a write produced are `newSpan`, unexported, and `FileResult.Spans` is filled only for a quote-licensed file (`apply.go:69, 724-730`); ADR-194 T2, blocked, owns that carry. Go prints bare file names in test failures, so a name that is unique in the root is not proof of which file failed; and a check can change the files it reports on (ADR-112). Recorded in BACKLOG.md with those findings.
- **Turnpack's loose matching** (indentation, line endings, typographic quotes). Rejected: mrw edits lines it served, by number or pattern; and its `reindentLike` leaves a replacement line shallower than the matched block as written, which in YAML reparents silently (AGENTS.md §4).
- **Undo a write whose check fails.** Not taken: ADR-080 and ADR-187 keep the write on exit 3, and no count says how often a caller reverts by hand (BACKLOG.md).
- **Quote the lines that hold the anchor.** Rejected: quoted content would have to be licensed or marked unlicensed, and the number is what the caller needs to read or re-address.

## Component / Boundary Impact

`internal/apply` (engine; `anchorMiss` only, owned here, go-reviewer pass per `lifecycle.md` §6). `read`, `plan`, `seen`, `check`, `state` and `writer` stay byte-identical. No receipt key is added: the words are in `hunks[].reason`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a refused `anchor=` hunk, CLI and `mrw_write` | its reason ends with the lines holding the anchor, or that none does | T1 | every caller |
| `scripts/contract.sh` | §337 | T1 | CI Linux |
| `README.md`, `AGENTS.md` | one sentence where `anchor=` is described | T1 | every agent |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** a wrong range is corrected from the refusal's own line number.
- **Negative:** a longer refusal; and each anchor miss scans the file's lines once, under the write lock, where before only the #589 case did. Measured by the go-reviewer pass on 0d5defe4: 2,000 missed anchors in one 200,000-line file took 1.44 s against 9 ms, about 3.6 ns per line per missed hunk.
- **Neutral:** the caller still reads a named line it was not served.

## Out of Scope

- A failed check says whether its output names a line the write changed (deferred: docs/adr/BACKLOG.md)
- `expected removal differs` names where the expected text is (deferred: docs/adr/BACKLOG.md)
- A SEARCH or old side that matched nothing, or several times without naming the lines (deferred: docs/adr/BACKLOG.md)
- Undoing a write whose check failed (deferred: docs/adr/BACKLOG.md)
- A start or end pattern that matched no line (permanent: boundary: a regular expression names no text to be close to, and naming lines a looser pattern would match is a guess at what the caller meant)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| An existing assertion needs the reason to end where it ends now | Low | Low | every assertion found is a substring (`anchor_quote_test.go:35, 38, 65`, `adversarial/ledger_test.go:302`, `contract.sh:4107, 10176, 10934`); T1 searches again before editing |
| A short anchor (`}`) is in hundreds of lines | Med | Low | five are named and the rest counted |
| A plan with thousands of missed anchors in a very large file holds the write lock while each is scanned | Low | Med | measured above; 10,000 misses in a million-line file is about 36 s of the 120 s another writer waits (ADR-110). Not capped: a plan that large is generated, and a generated plan takes its anchors from the file. A cap per file is the fix if one is ever seen |

## Rollback

Revert T1: the refusal ends where it ends today.

## Follow-ups

- [ ] The 30-day reading of Decision 4 (BACKLOG.md, From ADR-197)
