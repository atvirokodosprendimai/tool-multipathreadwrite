# Spec: Dangling high-impact leftovers

> **Date:** 2026-09-16 · **Status:** Draft
> **Owner:** M · **Becomes:** five destinations — all BACKLOG / Verification-Log pins; no engine ADR unless a probe finds a defect — ADR-019 Follow-ups (UC-1), ADR-031/039 + BACKLOG under-ceiling (UC-2), ADR-002 + BACKLOG concurrent writes (UC-3), ADR-055/056 campaign pre-registration (UC-4), BACKLOG JSX probe (UC-5)
> **Gate:** Status may become Ready-for-ADR only after `spec-verify --spec <this file>` exits 0.
> **Cross-references:** `docs/adr/BACKLOG.md`, `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md`, `docs/adr/ADR-031-a-page-licenses-only-what-came-back.md`, `docs/adr/ADR-039-a-fitting-read-licenses-only-what-came-back.md`, `docs/adr/ADR-002-mrw-will-not-edit-a-file-it-has-not-seen.md`, `docs/adr/ADR-055-the-receipt-counts-its-advisories-and-notices-a-pattern.md`, `docs/adr/ADR-056-the-receipt-carries-the-pattern-and-the-tally-prices-the-flag.md`, `docs/adr/ADR-048-mrw-models-no-target-syntax.md`, `docs/curve/reading-18-result.md`

## Problem

After v1.22.0 the remaining high-impact leftovers are recipes and accepted risks, not unarmed engine holes. Sessions rediscover them as "should we lock / flip `--strict-balance` / reopen multi-root / treat ADR-039 as the host-cut closed". An unrun probe scored as coverage is the same class as the four-leftovers Desktop miss-rate hole.

## Goal

One spec, five use cases: each leftover has a named test that goes red if the recipe, the unrun≠coverage rule, or the parked decision is deleted.

## Actors

| Actor | Kind | Goal |
|-------|------|------|
| Operator | human role | Know which measurements are still open and that unrun is not a pass |
| Concurrent writer | system | Not be promised exclusive apply; last rename can win silently |
| CLI/MCP caller | human role | `--strict-balance` stays opt-in until a campaign that reports false positives qualifies a default |

## Use Cases

### UC-1: Operator knows Desktop reach is pick A plus an unrun population measure

- **Trigger:** a reader asks whether Desktop analysts with scattered files are covered · **Preconditions:** ADR-019 pick A shipped (one named `--root` per `mrw mcp` run)
- **Main flow:**
  1. Operator follows the BACKLOG reach receipt: measure how many trees one Desktop session actually needs; whether Desktop sends `roots/list` is filed for pick C only.
  2. The coder-session plan count (~40 plans, all single-root) stays named as the wrong population.
- **Failure paths:** a. quoting that count as Desktop evidence → BACKLOG "WRONG POPULATION" / "not evidence about Desktop" must still be there. b. reopening B/C or growing `Apply` to a per-hunk ledger → new record, new quote; not this spec.
- **Postconditions:** pick A stands; capability (`grep`/`exclude`) stays shipped; no engine change.

### UC-2: Operator knows under-ceiling host truncation is measured, not closed

- **Trigger:** a reader asks whether host-cut of served text is done · **Preconditions:** ADR-024 dropped `isError` on pages; ADR-031/039 ack before they license
- **Main flow:**
  1. Operator follows the BACKLOG under-ceiling recipe: a host measured truncating a result under the advertised ceiling; file that observation beside reading 18 rather than treating ADR-039 as it.
  2. ADR-031 Follow-ups still name re-running reading 18's fixture against a truncating host.
- **Failure paths:** a. "039 shipped so the class is closed" → the under-ceiling entry must still say not to treat ADR-039 as that evidence. b. probe not run → stays deferred / open; must not be scored as coverage.
- **Postconditions:** no engine ADR unless the probe finds a defect; paged+fitting ack stays.

### UC-3: Operator knows concurrent writes can print applied and still lose

- **Trigger:** a reader asks whether two `mrw write`s on one file are exclusive · **Preconditions:** ADR-002 locking is permanently out of scope
- **Main flow:**
  1. BACKLOG states the observed class: last-writer-wins; a loser can print a full success receipt and exit 0; the sha guard is sometimes loud.
- **Failure paths:** a. "cannot both land" / "loud when it loses" as a guarantee → those clauses are already struck; the silent-applied observation must remain. b. adding a lock to "close" it → refused here; needs a new quote that invalidates ADR-002's scope.
- **Postconditions:** no lock, no CAS; the risk stays described accurately. — superseded by ADR-075 (2026-09-25): M's answer to the plan for the v1.25.1 round is the quote failure path b asks for; writers take a per-checkout lock and a stale one is refused.

### UC-4: Operator knows `--strict-balance` as default is a campaign, not a flip

- **Trigger:** a reader asks to make `--strict-balance` the default · **Preconditions:** ADR-055 T3 shipped the flag opt-in; ADR-056 prices it as writes land
- **Main flow:**
  1. Operator reads the pre-registered criterion: three corpora (Zeus, this repository, playtrix), false positives under 5% of refusals in every corpus, at least 50 refusals total.
  2. `mrw stats --json` `.pricing` is the source; a campaign that reports true positives without false positives does not qualify.
- **Failure paths:** a. unrun campaign or TP-only report treated as a pass → default stays off. b. flipping the default without the campaign → existing `TestStrictBalanceIsOffByDefault` stays red for that change.
- **Postconditions:** flag remains opt-in; neighbour licence on single-line was declined on 2026-09-26 (BACKLOG inventory), not this campaign.

### UC-5: Operator knows the JSX nest probe is unmeasured, not a finding

- **Trigger:** a reader asks whether mrw's wrap-tail / indent class includes JSX · **Preconditions:** ADR-048 forbids a write-time parser; YAML indent reparent is documented; JSX was predicted and not run
- **Main flow:**
  1. Operator follows the BACKLOG probe: a balanced-but-wrongly-nested JSX subtree, `tsc` green, DOM wrong.
  2. If it reproduces it is the worst silent-damage shape reported; until then it is a probe, not a finding.
- **Failure paths:** a. unrun probe treated as a finding or as coverage → BACKLOG must still say "not as a finding".
- **Postconditions:** no syntax-awareness ADR; no engine change unless the probe reproduces and M quotes a record.

## Scenarios

### UC1-S1 [happy] Desktop reach recipe names trees-per-session and roots/list [@spec] → `internal/adversarial/dangling_probe_test.go::TestADesktopReachRecipeNamesTreesPerSessionAndRootsList` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestADesktopReachRecipeNamesTreesPerSessionAndRootsList$'`

```gherkin
Given ADR-019 pick A shipped
When a reader looks for the remaining reach work
Then BACKLOG names a Desktop-population measurement of how many trees one session needs
And it files whether Desktop sends roots/list for pick C only
```

### UC1-S2 [failure] A coder plan count is not Desktop reach evidence [@spec] → `internal/adversarial/dangling_probe_test.go::TestACoderPlanCountIsNotDesktopReachEvidence` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestACoderPlanCountIsNotDesktopReachEvidence$'`

```gherkin
Given the ~40-plan single-root count from a coder session
When a reader quotes it as evidence that Desktop is covered
Then BACKLOG still says that measurement was taken on the wrong population
And that it is not evidence about Desktop
```

### UC2-S1 [happy] Under-ceiling host-cut recipe is filed and not closed by ADR-039 [@spec] → `internal/adversarial/dangling_probe_test.go::TestAnUnderCeilingHostCutRecipeIsFiledAndNotClosedByAdr039` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestAnUnderCeilingHostCutRecipeIsFiledAndNotClosedByAdr039$'`

```gherkin
Given ADR-039 fitting ack shipped
When a reader looks for whether host truncation is done
Then BACKLOG files an under-ceiling measurement beside reading 18
And it says not to treat ADR-039 as that evidence
```

### UC2-S2 [failure] An unrun under-ceiling probe is not the class closed [@spec] → `internal/adversarial/dangling_probe_test.go::TestAnUnrunUnderCeilingProbeIsNotTheClassClosed` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunUnderCeilingProbeIsNotTheClassClosed$'`

```gherkin
Given the under-ceiling probe has not been run
When a reader scores the host-cut class
Then that BACKLOG entry stays deferred or open
And nothing treats ADR-039 as the missing observation
```

### UC3-S1 [happy] Concurrent last-writer-wins is filed as silent applied [@spec] → `internal/adversarial/dangling_probe_test.go::TestConcurrentLastWriterWinsIsFiledAsAnAcceptedSilentAppliedRisk` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestConcurrentLastWriterWinsIsFiledAsAnAcceptedSilentAppliedRisk$'`

```gherkin
Given two mrw write processes against one file
When a reader asks whether both can report applied
Then BACKLOG records a loser printing a success receipt that says applied
And last-writer-wins is the mechanism
```

### UC3-S2 [failure] The race is closed by ADR-075, and the old scope is withdrawn [@spec] → `internal/adversarial/dangling_probe_test.go::TestTheConcurrentWriteRiskIsClosedByADR075` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestTheConcurrentWriteRiskIsClosedByADR075$'`

```gherkin
Given the silent-applied race was real
When a reader asks whether it is still open
Then BACKLOG records that ADR-075 closed it and names contract §150
And no longer says locking stays permanently out of scope
```

### UC4-S1 [happy] Strict-balance default campaign criterion is filed [@spec] → `internal/adversarial/dangling_probe_test.go::TestTheStrictBalanceDefaultCampaignCriterionIsFiled` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestTheStrictBalanceDefaultCampaignCriterionIsFiled$'`

```gherkin
Given --strict-balance shipped opt-in
When a reader looks for how a default could be argued
Then BACKLOG names Zeus, this repository, and playtrix
And false positives under 5% of refusals with at least 50 refusals total
```

### UC4-S2 [failure] An unrun campaign does not qualify a default [@spec] → `internal/adversarial/dangling_probe_test.go::TestAnUnrunStrictBalanceCampaignDoesNotQualifyADefault` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunStrictBalanceCampaignDoesNotQualifyADefault$'`

```gherkin
Given no campaign has met the criterion
When a reader treats a TP-only report or an unrun campaign as a pass
Then BACKLOG still says a TP-without-FP campaign does not qualify
And the default stays off
```

### UC5-S1 [happy] JSX nest probe recipe is filed as unmeasured [@spec] → `internal/adversarial/dangling_probe_test.go::TestAJsxNestProbeRecipeIsFiledAsUnmeasured` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestAJsxNestProbeRecipeIsFiledAsUnmeasured$'`

```gherkin
Given the YAML indent reparent is documented
When a reader looks for the JSX analogue
Then BACKLOG names a balanced-but-wrongly-nested JSX subtree
And it says tsc green, DOM wrong
```

### UC5-S2 [failure] An unrun JSX nest probe is not a finding [@spec] → `internal/adversarial/dangling_probe_test.go::TestAnUnrunJsxNestProbeIsNotAFinding` cmd:`go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunJsxNestProbeIsNotAFinding$'`

```gherkin
Given the JSX probe has not been run
When a reader scores silent-damage shapes
Then BACKLOG still says it is a probe, not a finding
```

## Facts

| ID | Assertion (invariant / behavior) | Test (`path::name`) | Tag | Cmd (optional) |
|----|----------------------------------|---------------------|-----|----------------|
| F-1 | The class is any claim that Desktop reach is settled; members are trees-per-session, `roots/list`, pick A stands, B/C need a new quote. BACKLOG names a Desktop-population measurement of how many trees one session needs and whether Desktop sends `roots/list` | `internal/adversarial/dangling_probe_test.go::TestADesktopReachRecipeNamesTreesPerSessionAndRootsList` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestADesktopReachRecipeNamesTreesPerSessionAndRootsList$' |
| F-2 | The ~40-plan coder-session count is the wrong population and is not evidence about Desktop | `internal/adversarial/dangling_probe_test.go::TestACoderPlanCountIsNotDesktopReachEvidence` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestACoderPlanCountIsNotDesktopReachEvidence$' |
| F-3 | The class is any claim that host-cut of served text is closed; members are paged ack (031), fitting ack (039), under-ceiling cut (open), reading 18. An under-ceiling measurement is filed beside reading 18 and must not treat ADR-039 as that evidence | `internal/adversarial/dangling_probe_test.go::TestAnUnderCeilingHostCutRecipeIsFiledAndNotClosedByAdr039` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestAnUnderCeilingHostCutRecipeIsFiledAndNotClosedByAdr039$' |
| F-4 | An unrun under-ceiling probe is not the class closed; the entry stays deferred or open | `internal/adversarial/dangling_probe_test.go::TestAnUnrunUnderCeilingProbeIsNotTheClassClosed` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunUnderCeilingProbeIsNotTheClassClosed$' |
| F-5 | The class is concurrent writers on one path; members are silent applied, sometimes-loud sha mismatch, last rename wins. BACKLOG records a loser printing a success receipt that says applied | `internal/adversarial/dangling_probe_test.go::TestConcurrentLastWriterWinsIsFiledAsAnAcceptedSilentAppliedRisk` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestConcurrentLastWriterWinsIsFiledAsAnAcceptedSilentAppliedRisk$' |
| F-6 | Superseded by ADR-075 (2026-09-25): writers take a per-checkout lock and a stale one is refused; BACKLOG records it and no longer says locking stays out of scope | `internal/adversarial/dangling_probe_test.go::TestTheConcurrentWriteRiskIsClosedByADR075` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestTheConcurrentWriteRiskIsClosedByADR075$' |
| F-7 | The class is flipping `--strict-balance` to default; members are opt-in flag, three corpora, false positives under 5% of refusals, at least 50 refusals, unrun is not a default. BACKLOG names Zeus, this repository, playtrix, 5%, and 50 | `internal/adversarial/dangling_probe_test.go::TestTheStrictBalanceDefaultCampaignCriterionIsFiled` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestTheStrictBalanceDefaultCampaignCriterionIsFiled$' |
| F-8 | `--strict-balance` stays off by default until a campaign that reports false positives qualifies | `internal/apply/apply_test.go::TestStrictBalanceIsOffByDefault` | @spec | go test ./internal/apply/ -count=1 -run '^TestStrictBalanceIsOffByDefault$' |
| F-9 | A campaign that reports true positives without false positives does not qualify; the default stays off | `internal/adversarial/dangling_probe_test.go::TestAnUnrunStrictBalanceCampaignDoesNotQualifyADefault` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunStrictBalanceCampaignDoesNotQualifyADefault$' |
| F-10 | The class is silent structural damage with a green typecheck; members are YAML indent (documented) and JSX nest (unmeasured). BACKLOG names a balanced-but-wrongly-nested JSX subtree, tsc green, DOM wrong | `internal/adversarial/dangling_probe_test.go::TestAJsxNestProbeRecipeIsFiledAsUnmeasured` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestAJsxNestProbeRecipeIsFiledAsUnmeasured$' |
| F-11 | An unrun JSX nest probe is a probe, not a finding | `internal/adversarial/dangling_probe_test.go::TestAnUnrunJsxNestProbeIsNotAFinding` | @spec | go test ./internal/adversarial/ -count=1 -run '^TestAnUnrunJsxNestProbeIsNotAFinding$' |

## Domain

Five leftovers, one spec, no new engine record. UC-1–UC-2–UC-5 are human-observed probes whose executable half is "the recipe is filed and unrun is not coverage". UC-3 was an accepted risk (ADR-002) until ADR-075 closed it. UC-4 is a pre-registered campaign whose default-off is already an apply test. Highest shipped ADR file is 062.

## Contracts Touched

None — documentation and adversarial recipe tests. No `scripts/contract.sh` row; no CLI/MCP change.

## Non-Goals

- Reopen ADR-019 pick B/C, `roots/list` wiring, or a per-hunk ledger — permanent: boundary: pick A shipped; new quote
- A file lock / CAS on `writeFile` — permanent: boundary: ADR-002 locking out of scope
- `--strict-balance` as default — permanent until the campaign meets F-7/F-9
- Neighbour licence on a single-line address — permanent: open question, not a proposed fix
- Per-extension check skip; `apply_patch` Move-to with hunks; `body=@` inside `Parse`; widen prose — permanent: boundary: already refused without a new quote
- Inventing Rust `packages()`; editing Zeus or quality-harness — permanent: boundary
- Write-time / apply parser (ADR-048) — permanent: mrw models no target syntax
- Retag `v1.20.0` / `v1.20.1` / `v1.21.0` / `v1.21.1` / `v1.22.0` — permanent: releases already shipped
- Engine go/no-go: apply / plan / seen / check / state stay byte-identical against the merge-base

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Unrun probe scored as coverage | High | High — a hole reads as measured | F-2, F-4, F-9, F-11; same class as leftover F-16 |
| Campaign criterion shaped after the result | Med | High — a default argued from TPs only | F-7 filed before any campaign; F-9 refuses TP-without-FP |
| A session "fixes" the race with a lock | Med | High — invalidates ADR-002 silently | F-6; Non-Goal; new quote required |
| JSX probe never run, YAML treated as the whole class | Med | Med | F-10/F-11 keep the probe named |

## Open Questions

<!-- empty: remaining grill items accepted from BACKLOG; quote-gated engine leftovers are Non-Goals -->

## Verify

```bash
spec-verify --spec docs/specs/2026-09-16-dangling-high-impact.md
```

## Grill Log (appendix)

| # | Question | Fact | Decision |
|---|----------|------|----------|
| 1 | One spec or five? | non-behavioral | One spec at `docs/specs/2026-09-16-dangling-high-impact.md` fans to five BACKLOG/ADR pins; same shape as 2026-09-15 leftovers |
| 2 | Which leftovers are in? | non-behavioral | The five high-impact dangling items from the 2026-09-16 briefing; quote-gated engine leftovers stay Non-Goals |
| 3 | Desktop reach: measure or reopen B/C? | F-1 | Measure only; pick A stands; B/C need a new quote |
| 4 | Coder plan count as Desktop evidence? | F-2 | Reject — wrong population; must stay named as such |
| 5 | Host-cut closed by 031/039? | F-3 | Reject — under-ceiling measurement remains; do not treat ADR-039 as it |
| 6 | Unrun under-ceiling as coverage? | F-4 | Must not |
| 7 | Concurrent writes: characterize or lock? | F-5 | Characterize silent applied; last-writer-wins |
| 8 | Close the race with a lock? | F-6 | Reject — ADR-002 scope |
| 9 | Strict-balance default now? | F-7 | Reject — campaign first; 5% / 50 / three corpora |
| 10 | Default stays off until then? | F-8 | Accept — existing apply test |
| 11 | TP-only / unrun campaign as a pass? | F-9 | Must not; default stays off |
| 12 | JSX in this spec as a parser? | F-10 | Probe recipe only; ADR-048 stands |
| 13 | Unrun JSX as a finding? | F-11 | Must not |
| 14 | Next contract section / new ADR number? | non-behavioral | No contract row; no new ADR unless a probe finds a defect |
