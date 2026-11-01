# ADR-185: the core says `--stat` lists the matching files

**Status:** Proposed
**Date:** 2026-11-01
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-184, ADR-141, ADR-161, docs/bench/observability/CORE-RESULTS.md
**Governs:** `internal/guide/guide.go`, `scripts/contract.sh`, `.claude/skills/mrw/SKILL.md`
**Enforced-by:** `internal/guide/core185_test.go::TestTheCoreSaysStatListsTheMatchingFiles`
**Invalidates:** ADR-184 Decision 2's rule-6 clause `--stat lists files`, which goes back to `--stat lists the matching files`. Checked and unchanged: ADR-184 Decision 1 (rule 4's hunk header), its word ceiling, and every clause ADR-141's test holds; ADR-161 Decision 5 (rule 1 still names `mrw write --grep P --sub R`).
**Served-path change:** `mrw instructions --core`, and the core inside `mrw skill`, print rule 6 with `--stat lists the matching files` again, rule 1 as `one change at many sites:` and rule 6's last clause as `--no-numbers drops numbers`; the core stays 298 words; `mrw instructions` without the flag is unchanged.

## Context

ADR-184 shortened rule 6's `--stat lists the matching files` to `--stat lists files` to fit the hunk header under ADR-141's ceiling. `--stat` lists the files a read names or a `--grep` matches; with neither and no working set (`mrw iter`), `mrw read --stat` is refused, exit 2: `mrw: read needs a PATH[:RANGE] or @N, or a working set`.

The core-first ablation's transcripts (`docs/bench/observability/CORE-RESULTS.md`, runs 2 to 5) are consistent with the shorter wording being read as a command of its own. A bare `mrw read --stat`, refused each time:

| Run | Core handed to the C arm | Bare `mrw read --stat` in C cells | In F cells |
|---|---|---:|---:|
| 2 | v1.79.0's, `--stat lists the matching files` | 0 of 10 | 0 of 10 |
| 3 | ADR-184's, `--stat lists files` | 1 of 10 | 0 of 10 |
| 4 | the same (v1.80.0) | 1 of 10 | 0 of 10 |
| 5 | the same (v1.80.0) | 3 of 10 | 0 of 10 |

Counted with a regular expression over each Bash command's segments (`\bmrw read --stat\s*$`), from the private transcripts the exports fingerprint. Five in 30 cells is a small count and an association, not a proof; it appears only where the short wording was the agent's whole guide, and the refusal cost the call it was in.

**Audit of the class** — *a clause the ADR-184 shortening changed in meaning, not only in length*. The four rules it shortened, against the v1.79.0 text:

| Rule | v1.79.0 | ADR-184 | Meaning kept? |
|---|---|---|---|
| 2 | `a write to a line mrw has not served you is refused` | `writing a line mrw never served you is refused` | yes |
| 3 | `read the refusal, which names the file, the plan line and the reason, and do not reach for --force` | `read the refusal, which names file, plan line and reason; never --force` | yes |
| 5 | `3 applied but the check did not pass (failed, timed out or interrupted)` | the parenthesis dropped | yes: the full text names the three |
| 6 | `--grep PATTERN [paths]` | `--grep PATTERN` | yes: a walk with no paths starts at the root |
| 6 | `--stat lists the matching files` | `--stat lists files` | **no**: it no longer says which files |
| 6 | `--max-cols N cuts a long line, and a cut line is unread` | `--max-cols N cuts long lines, leaving them unread` | yes |

## Existing Primitives Audit

- `guide.Core()` (`internal/guide/guide.go`), held by `TestTheCoreIsEightShortRulesTheFullContractStillCarries` (rule 6's clause `--stat lists` stays) and by ADR-184's `TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf` (rule 4 unchanged, under 300 words).
- The checked-in skill, held to `mrw skill` by `TestTheRepositorySkillIsTheBinarysSkill`, regenerated in the same change.

## Decision

1. **Rule 6 says `--stat lists the matching files`** again, as v1.79.0 did.
2. **Two words come back elsewhere**, keeping every held clause: rule 1's `the same change at many sites:` becomes `one change at many sites:`, and rule 6's `--no-numbers drops the numbers` becomes `--no-numbers drops numbers`. The core stays 298 words.
3. **The skill is regenerated** with `mrw skill --dir .claude/skills`.

**What would make this decision wrong:** a bare `--stat` still appearing in core-arm cells, or the two shortened clauses being misread. Either is visible in a later run's transcripts; this record makes no new measurement.

## Alternatives Considered

- **Make a bare `mrw read --stat` list the root.** Rejected here: it changes a read's meaning and its exit code, a contract change of its own, for a call not observed in the reported old-wording cells.
- **Leave it.** Rejected: the short wording is a suspected regression of ADR-184 (five calls, none before it), and the old wording costs two words that fit.

## Component / Boundary Impact

`internal/guide` (one function's text), its test, contract §321, the checked-in skill. No engine package; no flag; no exit code.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw instructions --core`, `mrw skill` | rules 1 and 6 reworded | T1 | agents |
| contract §321 | new rows | T1 | CI |
| `.claude/skills/mrw/SKILL.md` | regenerated | T1 | Claude Code here |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** rule 6 says again which files `--stat` lists.
- **Negative:** rules 1 and 6 lose an article each.
- **Neutral:** `mrw instructions` without the flag is unchanged.

## Out of Scope

- Measuring whether the bare `--stat` calls stop (permanent: boundary: this record restores wording and claims no measured effect; a later run, registered for it, would be its own decision)
- A bare `mrw read --stat` that lists the root (permanent: boundary: Alternatives; a read's meaning and exit code are a separate decision)
- The centralised `mrw` skill copy in memoryd `craft` (external: memoryd craft: refreshed by whoever merges, from `mrw skill`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A shortened clause loses a meaning | Low | Low | the audit above; every held clause asserted per rule |

## Rollback

Revert T1.

## Follow-ups

- [ ] none at authoring
