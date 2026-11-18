# ADR-200: A served line carries a tag a plan can cite

**Status:** Proposed
**Date:** 2026-11-18
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-035, ADR-052, ADR-137, ADR-169, ADR-179, ADR-197, ADR-009, docs/adr/BACKLOG.md, docs/bench/effect-study/README.md
**Supersedes:** none
**Governs:** `docs/bench/effect-study/scan.py`, `docs/bench/effect-study/test_scan.py`
**Enforced-by:** `docs/bench/effect-study/test_scan.py::test_single_line_anchor_misses`
**Served-path change:** none until task T1's reading meets its bar; then, with `MRW_TAGS=on` or `--tags`, a read prints two hex digits after each line number (` 42:a3| …`), and a plan address may cite them (`@@ f.go 42:a3 replace`), refused when the line no longer has that tag.

## Context

**Where this came from.** A reading of 2026 work on edit formats (memoryd craft/34fd8f89): a paper measuring line-range edits misapplied silently in 99.1% of files after a one-line shift (arXiv 2609.11957), and an edit format that tags each served line with a short content hash and has edits cite the tag (the "hashline" post of 2026-02-12, its author's own benchmark, no independent replication found). Zy, 2026-11-18: *"Ok go, build experiments"*, then *"Yes"* to drafting this record once the bar was met.

**What mrw already refuses.** The drift the paper measures does not land here: a file that changed since mrw served it is refused whole by its sha (ADR-002), a write to a line never served is refused (ADR-005), and a `replace` of more than one line must carry `anchor=`, checked against the addressed line (ADR-035).

**What is left.** A single-line `replace` needs no anchor (ADR-035 Decision: "A single-line `replace` is unchanged"). If the caller types line 41 where it meant 42, in a file that has not changed, and both lines were served, the body lands on the wrong line and the hunk reports `ok`. ADR-035 accepted that for a caller who "has read the lines". Nothing measures how often it happens.

**What the transcripts say** (`docs/bench/effect-study/`, 942 Claude Code transcripts on one Mac, entries before 2026-11-18T10:00:00Z, bars pre-registered in BACKLOG.md before the run):
- 22.3% of the hunks typed are a single-line `replace` with no anchor (6,356 of 28,563). This is exposure, not a rate of wrong edits.
- Anchor-miss lines are 2.23% of the anchored hunks typed (254 against 11,414). This is a ratio of result lines to hunks typed, over hunks of every length, and a miss may be a stale memory of a moved line as well as a wrong number.
- Anchor text is 0.16% of recorded output tokens: a tag saves almost no typing.
- On this repository's tracked text files, 0.65% of lines share a 2-hex-digit tag with a different line directly above or below, and 0.41% have an identical neighbour, which no tag or anchor tells apart.

**What is not known.** The rate at which a single-line `replace` names the wrong line. The agent benchmark's "silent mis-edits" column (`docs/bench/RESULTS.md`) counts any change outside a task's allowed regions, on own tasks only, and does not separate this cause. Task T1 measures the nearest thing the transcripts hold, and no code is written before it.

**Audit of the class**: *every hunk that changes or removes a line it does not name by content.*
- A multi-line `replace`: guarded by `anchor=` (ADR-035).
- A `delete`: its receipt says what it removed (ADR-008), after the fact; a body, when given, is checked.
- A single-line `replace`: unguarded. Decided here.
- `insert-after` and `insert-before`: change no existing line; a wrong address puts new lines in the wrong place. Left out (Out of Scope).
- A pattern address: names the line by content already.

## Existing Primitives Audit

- `anchor=` is accepted on a single-line `replace` today and checked. A caller that wants this guard has it, at the cost of typing a piece of the line.
- `sha=` and `lines=` pin the file, not the line (AGENTS.md §2).
- The read's gutter ` NNN| ` is parsed by the PostToolUse hook (`.claude/hooks/rules-on-read.py`), `mrw hook read`, the MCP checkpoints and the scanners under `docs/bench/`. A tag changes that gutter for every one of them when it is on.
- `MRW_MAX_COLS` (ADR-179) is the precedent for a read default taken from the environment, so a host turns it on once.

## Decision

1. **Measure first (T1).** The scanner counts, among anchored hunks, the single-line `replace` hunks and the anchor misses whose refused address is a single line. That ratio estimates how often a caller that bothered to anchor a single line had the wrong one. It is biased: callers who anchor a single line are not a sample of those who do not. The bar is in BACKLOG.md ("Pre-registered for ADR-200"), written with this record and before the scanner is changed.
2. **Only if the bar is met, a read can print a tag.** With `MRW_TAGS=on` in mrw's environment, or `--tags` on the call, each served line is printed ` NNN:hh| ` where `hh` is the first two hex digits of the SHA-1 of the line's bytes. Off by default: the tag costs three characters on every served line, and every reader of the gutter has to accept both forms.
3. **A plan address may cite the tag.** `N:hh` wherever a single line number is accepted (`@@ f.go 42:a3 replace`, and either end of `N-M`). A cited tag that does not match the line's current tag fails the hunk as an anchor miss does, naming the lines within five of `N` whose tag is `hh`. An address without a tag behaves as today. Nothing is required: a required tag on every single-line replace is a breaking change to every caller (Alternatives).
4. **What would make this decision wrong.** Thirty days after the release that ships Decisions 2 and 3, the share of single-line `replace` hunks that cite a tag is read from the transcripts. ADR-169's optional guard was used 27 times in 19.9 million served characters. If under 10% of single-line replaces in sessions whose host set `MRW_TAGS=on` cite a tag, the flag is words nobody uses and goes back to Zy with the count.

## Alternatives Considered

- **Require `anchor=` on a single-line `replace`.** The guard already exists and needs no read change. Not taken as the first step: it refuses every plan written today that has one, 22.3% of hunks on this corpus, for a mistake whose rate is unknown; and ADR-035 scoped the requirement to more than one line on purpose. It is the stronger answer if T1 finds the rate high.
- **The receipt quotes the line a single-line `replace` removed** (`was: "…"`), as a `delete` does. No input cost, no adoption problem, and it is on for every caller. It shows the mistake after it has landed and relies on the caller reading the row; under `--terse` or `MRW_RECEIPT=quiet` it is not printed. Deferred, with its own count to take first (Out of Scope).
- **A tag always on.** Rejected: about three characters on every served line for every caller, and a second gutter form for every parser, to guard one op.
- **A longer tag.** Three hex digits cut the neighbour collision from 0.65% to 0.056% of lines at one more character per line. Left at two: the tag is checked beside a line number, so it has to tell a line from its near neighbours, not from the file.
- **Fuzzy re-addressing** (apply the hunk to the nearby line that has the tag). Rejected: mrw applies what the plan names or refuses (ADR-001); a guess that lands is the failure this exists to prevent.

## Component / Boundary Impact

T1 touches `docs/bench/effect-study/` only. Decisions 2 and 3, if they are built, change `internal/read` (the gutter), `internal/plan` (the address) and `internal/apply` (the check): three engine packages, each needing a go-reviewer pass, and the tasks for them are written only after T1's reading and Zy's acceptance.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `docs/bench/effect-study/scan.py` | counts single-line anchored hunks and single-line anchor misses | T1 | this record's bar |
| `docs/adr/BACKLOG.md` | the bar, then the reading | T1 | Zy |
| a read under `MRW_TAGS=on` or `--tags` | ` NNN:hh| ` | tasks not yet written | every caller, every gutter parser |
| a plan address `N:hh` | refused on a tag mismatch | tasks not yet written | every caller |

## Inter-task Contracts

None — one task until T1's reading.

## Implementation

`tasks/README.md`: T1, the measurement. The tasks for Decisions 2 and 3 are written after T1's reading meets the bar and Zy accepts this record; if the bar is missed, this record is withdrawn with the numbers.

## Consequences

- **Positive:** a wrong line number on a single-line `replace` can be refused before it lands, for a caller whose host turns tags on.
- **Negative:** a second gutter form; three characters a served line when on; one more thing for a host to configure, with ADR-169's adoption as the warning.
- **Neutral:** identical neighbouring lines stay indistinguishable, as they are to an anchor.

## Out of Scope

- A receipt row quoting the line a single-line `replace` removed (deferred: docs/adr/BACKLOG.md)
- A tag on `insert-after` and `insert-before` addresses beyond accepting the same `N:hh` form (deferred: docs/adr/BACKLOG.md)
- Tags over `mrw_read` and its checkpoints (deferred: docs/adr/BACKLOG.md)
- Telling identical neighbouring lines apart (permanent: boundary: two lines with the same bytes have the same tag and the same anchor; only the line number separates them)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| T1's ratio is read as the rate of wrong unanchored replaces | Med | Med | Decision 1 states the bias; the bar is a floor for building, not a claim about the rate |
| The flag ships and is not used | High | Low | Decision 4's 30-day reading and its 10% floor |
| A gutter parser breaks on ` NNN:hh| ` | Med | Med | off by default; the tasks, when written, list every parser found by `git grep` and change each in the same change |
| A 2-hex tag matches the wrong neighbour | Low | Med | 0.65% of lines on this repository; the hunk still has its line number, so both must be wrong together |

## Rollback

T1 is a scanner change and a BACKLOG entry: revert it. Decisions 2 and 3 are off by default; removing the flag returns every read and plan to today's form.

## Follow-ups

- [ ] T1's reading against the bar (BACKLOG.md, Pre-registered for ADR-200)
- [ ] Decision 4's 30-day reading, if Decisions 2 and 3 ship
