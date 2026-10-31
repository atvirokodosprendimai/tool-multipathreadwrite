# ADR-184: `mrw instructions --core` names the hunk header a plan is made of

**Status:** Proposed
**Date:** 2026-10-31
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-141, ADR-171, ADR-161, ADR-168, ADR-164, docs/bench/observability/CORE-RESULTS.md
**Governs:** `internal/guide/guide.go`, `internal/guide/core141_test.go`, `scripts/contract.sh`, `.claude/skills/mrw/SKILL.md`
**Enforced-by:** `internal/guide/core184_test.go::TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf`
**Invalidates:** None — ADR-141's eight rules, its 300-word ceiling and every clause its test holds stay; rule 4 gains the header, and rules 2, 3, 5 and 6 are reworded shorter to make room. Checked and unchanged: ADR-173 (rule 5 keeps `a range your own --max-lines or --max-matches withheld is not one: exits 0 (capped)` word for word, which its fence greps and which keeps `--max-chars` out of the exit-0 case), ADR-171 Decision 1 (rule 4 still says to omit `body=`), ADR-161 Decision 5 (rule 1 still names `mrw write --grep P --sub R`), ADR-164 T1 (the core still does not say `sha=` stands in for nothing), ADR-168 (`mrw skill` embeds the core byte for byte).
**Served-path change:** `mrw instructions --core`, and the core inside `mrw skill`, print rule 4 as `A hunk is @@ PATH N-M replace anchor="TEXT", @@ PATH N insert-after or @@ PATH N-M delete, then its lines; anchor=, required on a multi-line replace, is copied from line N as read; omit body= unless mrw instructions names your case.`; rules 2, 3, 5 and 6 say the same in fewer words; `mrw instructions` without the flag is unchanged.

## Context

The core is the text a host is told to hand an agent at session start (`docs/agents.md:51-53`). Rule 1 tells the agent to make every edit as one plan, `mrw write PLANFILE`; no rule says what a plan line looks like. The core shows `--grep … --sub` and `@@ path 0 create`, and rule 4's `anchor=` and `body=` are parameters of a header it never shows.

The core-first ablation (`docs/bench/observability/CORE-RESULTS.md`, run 2, 2026-10-31) measured what that costs. The evidence, its registration (`CORE-PREREGISTRATION.md`), records and calculation merged in #576 (`f3ee1cf9`). Five synthetic jobs, each solved by a hand-built plan, two counterbalanced rounds, 20 of 20 cells solved:

- The arm handed the core ran the full `mrw instructions` in 8 of its 10 cells, 9 calls, 48,905 and 71,803 characters of tool result per round.
- That text arrives mid-conversation, so each job paid the cache-creation price for it. The arm handed the full guide in its system prompt read it at the cache-read price after the canary.
- The core arm's five-job totals were $0.207571 and $0.263914 against $0.142737 and $0.177929. It cost more in both rounds, although each of its requests started about 7,608 tokens smaller.

The results file reads the missing header as the likely cause, from accounting over 20 synthetic cells, not from an ablation of the header itself; this decision supplies the header. Whether the core arm then costs less is a measurement this record does not claim (Out of Scope).

**Run 3, against this branch's binary (same file, same day, registered as Amendment 2 before its first call).** The core arm ran the full `mrw instructions` in none of its 10 cells; its five-job cost was $0.147101 and $0.123706 against the full-guide arm's $0.143115 and $0.134596, a difference inside the repeat spread, so no cost result. Read with run 2: this change removed the extra fetch and the excess cost; it did not show core-first cheaper.

**Audit of the class** — *a plan form an agent handed only the core must write and cannot learn from it*. Enumerated from the ops `plan.Parse` accepts, `mrw instructions | grep -o 'Ops: [^.]*'` → `replace, insert-after, insert-before, delete, create, unlink, rename`, against the core at `2c5a2abd`:

| Form | In the core now | After |
|---|---|---|
| `@@ PATH N-M replace anchor="TEXT"` then lines | parameters only (rule 4) | shown (rule 4) |
| `@@ PATH N insert-after` then lines | no | shown (rule 4) |
| `@@ PATH N-M delete` | no | shown (rule 4) |
| `@@ path 0 create` | yes (rule 7) | unchanged |
| `--grep P --sub R` (a plan mrw writes) | yes (rule 1) | unchanged |
| `insert-before`, `unlink`, `rename`, `from=`, pattern addresses | no | no: in the full text, which a refusal names (Out of Scope) |

Every MRW write in the grouped pilot was a `replace` plan (`GROUPED-RESULTS.md`, "The five MRW writes are confirmed", merged in #576), so only the replace form is what the measured jobs needed. `insert-after` and `delete` are shown because they share the header's shape and cost nine words; that they help is not measured.

## Existing Primitives Audit

- `guide.Core()` (`internal/guide/guide.go:146-157`) and its test `TestTheCoreIsEightShortRulesTheFullContractStillCarries` (`core141_test.go:39`), which holds each rule's clauses in the core and in `guide.CLI()`. Reused; rule 4's row gains the header clauses.
- The word ceiling: `core141_test.go:41` (at most 300), `cmd/mrw/core141_test.go:64` and `ifchanged169_test.go:20` (under 300), contract §242 and §297 (`wc -w`, under 300). The core is 299 words at `2c5a2abd`; the new text is 298.
- `TestTheCoreNamesOnlyFlagsTheBinaryHas` (`cmd/mrw/core141_test.go:37-54`): the new rule names no flag.
- `mrw skill --dir .claude/skills` and `cmd/mrw/skill168_test.go:302-311`: the checked-in skill must equal `mrw skill`, so it is regenerated in the same change.

## Decision

1. **Rule 4 shows the hunk header.** `A hunk is @@ PATH N-M replace anchor="TEXT", @@ PATH N insert-after or @@ PATH N-M delete, then its lines; anchor=, required on a multi-line replace, is copied from line N as read; omit body= unless mrw instructions names your case.` The forms are the ones `plan.Parse` accepts unchanged; "copied from line N" is what the anchor check compares (`anchor "c" not in line 2`, measured 2026-10-31).
2. **Four rules get shorter, keeping every held clause.** Rule 2: `writing a line mrw never served you is refused`. Rule 3: `read the refusal, which names file, plan line and reason; never --force.` Rule 5: the parenthesis `(failed, timed out or interrupted)` goes, and its read clause is unchanged. Rule 6: `--grep PATTERN` without `[paths]`, `--stat lists files`, `--max-cols N cuts long lines, leaving them unread`. The full text keeps each of these at full length.
3. **The core stays under 300 words, eight rules, flags the binary has.** No ceiling moves.
4. **The checked-in skill is regenerated** with `mrw skill --dir .claude/skills`.

**What would make this decision wrong:** the core arm still fetching the full guide for these forms, or an agent misreading `@@ PATH N insert-after` as taking a range. The first is a measurement (Out of Scope); the second, a refused plan, which names the hunk.

## Alternatives Considered

- **Raise the ceiling above 300 words.** Rejected: ADR-141's ceiling is the point of the core, and four tests and two contract rows hold it; 298 words fit.
- **Point rule 1 at `mrw instructions` for the plan format.** Rejected: that is the fetch the ablation measured costing more than the full guide.
- **A ninth rule.** Rejected: ADR-141 fixes eight, and the header is rule 4's own subject.
- **Teach the plan in `docs/agents.md` only.** Rejected: the core is what the binary prints and a test holds; prose copies drift.

## Component / Boundary Impact

`internal/guide` (one function's text), its test, contract §320, the checked-in skill. No engine package; no flag; no exit code.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw instructions --core` | rule 4 shows the header; rules 3, 5, 6 shorter | T1 | agents, standing instructions |
| `mrw skill` | the same core inside it | T1 | skill hosts |
| contract §320 | new rows | T1 | CI |
| `.claude/skills/mrw/SKILL.md` | regenerated | T1 | Claude Code in this repository |

## Inter-task Contracts

None — one task.

## Implementation

`tasks/README.md`: T1. One pull request.

## Consequences

- **Positive:** an agent handed only the core can write the plans the measured jobs needed.
- **Negative:** four rules lose words the full text keeps: the three ways a check does not pass, that `--grep` takes paths, what `--stat` lists, and "a cut line".
- **Neutral:** `mrw instructions` without the flag is byte-identical.

## Out of Scope

- Measuring whether the core arm now costs less than the full-guide arm (deferred: docs/adr/BACKLOG.md — a re-run of the CORE-PREREGISTRATION design against a binary carrying this core, registered before it runs)
- `insert-before`, `unlink`, `rename`, `from=` and pattern addresses in the core (permanent: boundary: the measured jobs needed none of them, and the ceiling leaves no room; the full text names each and a refusal points at it)
- Changing `docs/agents.md`'s core-first advice (deferred: docs/adr/BACKLOG.md — it waits for the measurement above)
- The centralised `mrw` skill copy in memoryd `craft` (external: memoryd craft: refreshed by whoever merges, from `mrw skill`)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A shorter rule loses a clause an agent relied on | Low | Low | every clause ADR-141's test holds stays, asserted per rule; the full text keeps the longer wording |
| The taught forms drift from what `plan.Parse` accepts | Low | Med | §320 applies the three forms exactly as the core prints them, through the built binary, and pairs them with a wrong anchor that is refused |

## Rollback

Revert T1; the core returns to its 2c5a2abd text.

## Follow-ups

- [ ] none at authoring
