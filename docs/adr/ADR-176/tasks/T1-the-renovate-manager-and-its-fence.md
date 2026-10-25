# Task ADR-176-T1: the Renovate manager and its fence

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the `renovate.json` regex custom manager over ADR-147's pin shapes, this task's acceptance fence, the ADR-147-T2 amendment note
**Consumes:** ADR-147's pin shapes (finding, install, change table) and T2's fenced grep
**Data dependency:** hermetic — the fence and the coverage test grep local files only; Renovate itself runs hosted
**Proof map:** v1
**Rests-on:** `renovate.json parses`, `every fence clause holds`, `the shipped matchStrings cover the pin shapes and skip the exclusions`, `fence-prose is clean`, `the engine is untouched`

## Goal

ADR-176 Decisions 1 and 2: the custom manager and its five matchStrings, the fence that pins the manager's shape and the five prose shapes it covers, the exclusions named and proven unmatched, T2 amended.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `renovate.json` | edit | the custom manager |
| `docs/adr/ADR-147-every-language-the-tree-runs-is-analysed/tasks/T2-the-gate-runs-them.md` | edit | amendment note: the fence's pin is manager-bumped |
| this record, `tasks/README.md`, this task | create | the record and its task |

## Ordered Steps

1. [S1] RED: every manager clause of the fence below fails on the pre-change tree; every prose-shape clause passes as a version-agnostic guard. Ran 2026-10-25, exit 1. [proof: acceptance]
2. [S2] GREEN: the manager written; `renovate.json` parses; every fence clause holds. [proof: acceptance]
3. [S3] Coverage: the shipped matchStrings, run by node over both ADR-147 files, match every bumpable pin — six matches over five lines, because T2's change-table row carries two shapes on one line — and none of the three exclusions (the published-date sentence, the split `declares` sentence, the `npx` rejection). [proof: test]
4. [S4] `python3 scripts/fence-prose.py .` clean. [proof: gate]

## Acceptance

```bash
set -o pipefail
grep -q '"customType": "regex"' renovate.json \
  && grep -q '"managerFilePatterns"' renovate.json \
  && grep -q '"datasourceTemplate": "npm"' renovate.json \
  && grep -q '"depNameTemplate": "knip"' renovate.json \
  && grep -qE 'managerFilePatterns.*ADR-147' renovate.json \
  && grep -q '(?<currentValue>' renovate.json \
  && grep -qE 'knip@[0-9]+\.[0-9]+\.[0-9]+' docs/adr/ADR-147-every-language-the-tree-runs-is-analysed.md \
  && grep -qE 'save-exact knip@[0-9]+\.[0-9]+\.[0-9]+' 'docs/adr/ADR-147-every-language-the-tree-runs-is-analysed/tasks/T2-the-gate-runs-them.md' \
  && grep -qE '`knip` \([0-9]+\.[0-9]+\.[0-9]+\)' docs/adr/ADR-147-every-language-the-tree-runs-is-analysed.md \
  && grep -qE '`knip` [0-9]+\.[0-9]+\.[0-9]+, exact' docs/adr/ADR-147-every-language-the-tree-runs-is-analysed.md \
  && grep -qE '`knip` [0-9]+\.[0-9]+\.[0-9]+, exact' 'docs/adr/ADR-147-every-language-the-tree-runs-is-analysed/tasks/T2-the-gate-runs-them.md' \
  && grep -qE '"knip": "[0-9]+\.[0-9]+\.[0-9]+"' 'docs/adr/ADR-147-every-language-the-tree-runs-is-analysed/tasks/T2-the-gate-runs-them.md'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
| — | — | the shipped matchStrings match the five pin spans and none of the three exclusions (S3, node) | — | — |

## Reachability

| Rung | How this task shows it |
| 1 — exists | the `customManagers` block in `renovate.json` |
| 2 — something selects it | the hosted Renovate app reads `renovate.json` on every schedule |
| 3 — the caller can discover it | this record's Context names run 40144353890 as the failure it prevents |
| 4 — it is used | the next hosted knip PR carries the ADR-147 prose bumps in the same diff |

## Mutation Log
(empty until execute)

## Verification Log
- 2026-10-25 · (pre-change tree) · exit 1 · `bash .atl/fence-adr176.sh` · six manager clauses RED, five shape guards PRESENT (guards version-agnostic by design) — the fence can go red
- 2026-10-25 · (schema note) · `fileMatch` was renamed `managerFilePatterns` upstream (auto-migrated); the shipped name is used, and its clause's RED is read-proven: `renovate.json` was 3 lines (sha 94693a99) when the fence was drafted
- 2026-10-25 · (clause) · exit 1 → fixed · the literal `docs/adr/ADR-147` clause went red when the JSON path was re-encoded with character classes (a legal regex spelling with no backslashes to grep); the clause now asserts `managerFilePatterns.*ADR-147`, the shape that survives every spelling
- 2026-10-25 · fence-prose · exit 0 · `python3 scripts/fence-prose.py .` · 488 clauses, 0 red, 6 exempt; the T1 fence verbatim exits 0; the coverage proof reports spans 6 / changed 6 / exclusions 0
