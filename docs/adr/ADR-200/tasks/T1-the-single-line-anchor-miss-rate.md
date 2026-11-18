# Task ADR-200-T1: The single-line anchor-miss rate, measured against its bar

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `scan.py` counts `anchored_single_line_replaces` and `anchor_miss_lines_single_line`; `test_single_line_anchor_misses`; the reading in BACKLOG.md
**Consumes:** `docs/bench/effect-study/scan.py` (effect-study)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a single-line anchored replace is counted apart`, `a miss on a single-line address is counted apart`

## Goal

ADR-200 builds nothing until it knows how often a single-line `replace` that carries an anchor is refused for
naming the wrong line. The effect-study scanner counts anchored hunks and anchor-miss lines of every length
together. This task splits out the single-line ones: a hunk header whose address is `N`, `$` or `N-N` with an
`anchor=`, and a refusal row `FAIL PATH ADDR replace …: anchor "…" not in line N` whose `ADDR` is such an
address. The ratio of the second to the first is read against the bar pre-registered in
`docs/adr/BACKLOG.md` ("Pre-registered for ADR-200") and recorded there.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/bench/effect-study/test_scan.py` | edit | the red test |
| `docs/bench/effect-study/scan.py` | edit | the two counts and their ratio |
| `docs/bench/effect-study/README.md` | edit | the method and the reading |
| `docs/adr/BACKLOG.md` | edit | the reading against the bar |

## Ordered Steps

1. [S1] Write `test_single_line_anchor_misses` and see it RED: a plan with one anchored single-line replace, one anchored multi-line replace and one bare single-line replace, answered by a miss on the single-line hunk and a miss on the multi-line one, must report one anchored single-line replace and one single-line miss. The keys do not exist. [proof: acceptance]
2. [S2] `scan.py` reports `anchored_single_line_replaces`, `anchor_miss_lines_single_line` and their ratio. Mutants, each killed: every anchored hunk counted as single-line; every miss counted as single-line. [proof: mutation]
3. [S3] Run the scanner twice over the transcripts with `--until` in the past, confirm identical output, and record the reading and the verdict in BACKLOG.md and the README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
python3 -I docs/bench/effect-study/test_scan.py >/dev/null \
  && grep -q '^def test_single_line_anchor_misses' docs/bench/effect-study/test_scan.py \
  && grep -q '"anchor_miss_lines_single_line"' docs/bench/effect-study/scan.py \
  && grep -q '^\*\*Measured .*ADR-200' docs/adr/BACKLOG.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_single_line_anchor_misses` | `docs/bench/effect-study/test_scan.py` | a single-line anchored replace and a miss on a single-line address are each counted apart from the multi-line ones | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `scan.py` prints the keys on every run |
| 3 — the caller can discover it | the README's method and the BACKLOG reading name them |
| 4 — it is used | the reading decides whether ADR-200's remaining tasks are written |

## Mutation Log

## Verification Log

## Invariants

- Every count effect-study already reports keeps its value on the same corpus and window.
- The scanner prints aggregates only.

## Risks

- The ratio divides refusal lines by hunks typed, as effect-study's does: a plan sent twice counts its miss twice. The distinct-per-session count is reported beside it.
- A refusal row whose path holds a space is split wrongly by a simple pattern; the address is taken from the end of the row's head, before the op.

## Stop Condition

The fence exits 0 and the reading is in BACKLOG.md.

## Out of Scope

- Any change to `internal/read`, `internal/plan` or `internal/apply` (deferred: docs/adr/BACKLOG.md)
