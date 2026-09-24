# Dangling high-impact leftovers — execution plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run the five leftovers in `docs/specs/2026-09-16-dangling-high-impact.md` in product-leverage order, file each result so unrun is never coverage, and start an engine ADR only if a probe finds a defect.

**Architecture:** The spec's Go tests pin recipes in BACKLOG. This plan runs the leftovers themselves. mrw's behaviour does not change unless Task 2's campaign meets the pre-registered bar *and* M quotes a default, or Tasks 3–4 find a defect that gets its own record. Pricing cannot be reconstructed from old plans (ADR-009); the only source is `mrw stats --json` `.pricing` in each live checkout (ADR-056).

**Tech Stack:** PATH `mrw` (must be ≥ v1.18.0 to have priced anything; prefer the tagged `v1.22.0`), `jq`, this repo's contract/break scripts, one Desktop MCP session for Task 5, a one-file TypeScript `tsc` probe for Task 4. No new Go packages. Do not edit Zeus or Playtrix.

## Global Constraints

- Spec Non-Goals are in force: no lock/CAS; no `--strict-balance` default without the campaign; no neighbour licence on single-line; no per-extension check skip; no Move-to with hunks; no `body=@` inside `Parse`; no widen prose; no Rust `packages()`; no ADR-048 parser; no reopen of ADR-019 B/C without a new quote; do not retag `v1.20.0` / `v1.20.1` / `v1.21.0` / `v1.21.1` / `v1.22.0`.
- Engine go/no-go: `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` stay byte-identical against the merge-base unless a *new* ADR owns the change.
- Do not synthesize wrap-tails to fill pricing counters (BACKLOG: "the criterion is real corpora").
- Do not write a test whose example is "would_refuse is 0" — that goes red when real data arrives (stress-testing v6).
- Commit only if M asks. Do not push `main`.
- Oracle is the spec Facts and the BACKLOG pre-registration, never the implementation.

## Files

| Path | Role |
|------|------|
| `docs/specs/2026-09-16-dangling-high-impact.md` | Spec (already Draft; `spec-verify --spec` 0) |
| `docs/adr/BACKLOG.md` | Dated campaign table + probe receipts (append, do not rewrite the 2026-09-13 zero row) |
| `docs/adr/ADR-023-a-reads-answer-is-the-served-text.md` | Envelope recipe already filed; do not reuse as the under-ceiling observation |
| `docs/adr/ADR-031-a-page-licenses-only-what-came-back.md` | Follow-up: re-run reading 18 against a truncating host |
| `docs/curve/reading-18-result.md` | Prior paged cut; under-ceiling result is a *new* file beside it |
| `docs/curve/reading-under-ceiling-result.md` | Create in Task 3 |
| `docs/break/jsx-nest/` | Create in Task 4 (fixture that stays wrong-DOM on purpose) |
| `internal/adversarial/dangling_probe_test.go` | Pin tests; do not assert campaign counts |

---

### Task 1: Land the spec pins

**Files:**
- Already on branch `spec/dangling-high-impact`: the spec, `internal/adversarial/dangling_probe_test.go`, BACKLOG inventory row + "From the 2026-09-16 dangling spec" receipt.

**Interfaces:**
- Consumes: spec Facts F-1…F-11
- Produces: merged pins so later tasks' BACKLOG edits cannot be mistaken for the spec itself

- [ ] **Step 1: Confirm the gate is still green**

```bash
spec-verify --spec docs/specs/2026-09-16-dangling-high-impact.md
go test ./internal/adversarial/ -count=1 -run 'TestADesktopReach|TestACoderPlan|TestAnUnderCeiling|TestAnUnrunUnderCeiling|TestConcurrentLastWriter|TestLockingStays|TestTheStrictBalance|TestAnUnrunStrictBalance|TestAJsxNest|TestAnUnrunJsxNest'
go test ./internal/apply/ -count=1 -run '^TestStrictBalanceIsOffByDefault$'
```

Expected: spec-verify `[PASS]`; both `go test` lines `ok`.

- [ ] **Step 2: PR and merge only if M asks**

Same drill as other docs+test PRs. Do not push `main`. Do not treat this merge as having run the campaign.

- [ ] **Step 3: Stop**

Pins landed is not product. Continue at Task 2.

---

### Task 2: Re-read the `--strict-balance` campaign (UC-4) — highest product leverage

**Files:**
- Modify: `docs/adr/BACKLOG.md` (append a dated table under "Pre-registration: a default `--strict-balance`", leave the 2026-09-13 zero row in place)
- Read-only: Zeus and Playtrix checkouts named in that 2026-09-13 table. Do not edit those trees.

**Interfaces:**
- Consumes: `authoring.Pricing` JSON tags `strict_candidates`, `strict_would_refuse`, `strict_would_refuse_broke`, `strict_would_refuse_held`, `strict_would_refuse_unchecked` (`internal/authoring/authoring.go`); `Pricing.FalsePositiveRate() (rate float64, ok bool)` — `ok` is false when `broke+held == 0`
- Produces: a dated table + one of three verdicts: **does not qualify (too few refusals)** · **does not qualify (FP ≥ 5% in a corpus)** · **qualifies — wait for M quote before any default ADR**

Prior read (BACKLOG): 2026-09-13 PATH `mrw` v1.18.0, all three corpora `would_refuse = 0`. "Same quote re-arms a later read."

- [ ] **Step 1: Confirm which `mrw` will answer**

```bash
command -v mrw
mrw version
```

Expected: a binary ≥ v1.18.0. Prefer `v1.22.0 (6896442)`. If `mrw version` is older than v1.18.0, stop — that binary never priced.

- [ ] **Step 2: Dump this repository's pricing**

From `/Users/zy/GolandProjects/tool-multipathreadwrite`:

```bash
mrw stats --json | jq '.pricing'
```

Record `strict_candidates`, `strict_would_refuse`, `strict_would_refuse_broke`, `strict_would_refuse_held`, `strict_would_refuse_unchecked`. If `.pricing` is missing, the binary is too old — stop.

- [ ] **Step 3: Dump Zeus and Playtrix the same way**

In each checkout (the trees that produced the 2026-09-13 rows; do not clone a fresh empty one):

```bash
mrw stats --json | jq '.pricing'
```

State is per-root under XDG. Running `mrw stats` in a different clone of the same project is a **different corpus** and does not count.

- [ ] **Step 4: Score against the pre-registered bar**

Let `R` = sum of `strict_would_refuse` across the three corpora.

Per corpus, if `broke + held == 0` then that corpus has `ok=false` (no checked refusals) — it cannot pass "under 5% in every corpus" and it cannot flatter. Report `unchecked` anyway. A named corpus with no defined FP rate means the campaign does not qualify.

If `broke + held > 0`, FP rate = `held / (broke + held)`. Must be `< 0.05` in **every named corpus** (this repository, Zeus, Playtrix). A corpus with only `unchecked` would-refuse does not count those toward the 5% (BACKLOG: `_unchecked` counts neither way) but those refusals **do** count toward `R`.

Verdict:

| Condition | Verdict |
|-----------|---------|
| `R < 50` | Does not qualify. Default stays off. |
| any named corpus has `broke+held == 0` | Does not qualify. Insufficient evidence. Default stays off. |
| `R ≥ 50` and any named corpus has FP ≥ 5% | Does not qualify. Default stays off. |
| `R ≥ 50` and every named corpus has `broke+held > 0` and FP < 5% | Qualifies as *evidence*. Do **not** flip the default in this task. |

Do not fill counters with constructed wrap-tails.

- [ ] **Step 5: File the table in BACKLOG** (same columns as the 2026-09-13 row)

```
Ran YYYY-MM-DD against PATH mrw <version> (<sha>).

| corpus | landed | candidates | would_refuse | broke | held | unchecked |
|---|---|---|---|---|---|---|
| this repository | … | … | … | … | … | … |
| Zeus | … | … | … | … | … | … |
| Playtrix | … | … | … | … | … | … |

**Does not qualify** — … / **Qualifies as evidence** — wait for *"price strict balance"* plus an explicit default quote before any ADR that changes the default.
```

`landed` comes from `mrw stats --json` (existing tally), not from `.pricing`. If a checkout has no `landed`, write `n/a` and say so; do not invent it from `candidates`.

- [ ] **Step 6: Mutant the pin tests, not the counts**

```bash
# Temporary: delete the "false positives under 5%" sentence from BACKLOG, run:
go test ./internal/adversarial/ -count=1 -run '^TestTheStrictBalanceDefaultCampaignCriterionIsFiled$'
# Expected: FAIL. Restore the sentence. Re-run: PASS.
```

Do not add `TestPricingWouldRefuseIsZero`.

- [ ] **Step 7: Stop**

If the verdict is "qualifies as evidence", tell M and wait. A default is a new ADR, not an amendment sneaking into 055. If it does not qualify, Task 2 is done; continue.

---

### Task 3: Under-ceiling host-cut probe (UC-2)

**Files:**
- Create: `docs/curve/reading-under-ceiling-result.md`
- Modify: `docs/adr/BACKLOG.md` (the under-ceiling entry: still Deferred, or "observed YYYY-MM-DD, class closed / class still open")
- Read-only: `docs/curve/reading-18-result.md`, ADR-031 Follow-ups, ADR-039 Out of Scope

**Interfaces:**
- Consumes: ADR-024 (`isError` absent on a successful page); ADR-031/039 ack-before-license; reading 18's paged cut
- Produces: one observation of a result **under** `MaxResultChars` (default 200000) on a host that has truncated before. Not a miss rate. Not "039 closed it".

- [ ] **Step 1: Pick the cell**

Do **not** reuse the 3619-line / first-page-of-2727 cell as the only trial — that was a *paged* `isError: true` cut, since fixed by ADR-024. The leftover is: a host truncating a result that is **under the advertised ceiling**.

Two members, both required in the write-up even if only one can be run this session:

1. **Fitting** MCP `mrw_read` whose encoded result is under 200000 and is not a page (no `-- PARTIAL:`).
2. **Paged with `isError` absent** (post-024) of similar size to reading 18's 152k page.

If the host session is Claude Code, say so. If it is Desktop, say so. Do not quote reading 12 (VOID).

- [ ] **Step 2: Drive the read from a JSON-RPC client *and* from the model**

Wire check (no model): `mrw mcp` over stdio, one `mrw_read`, save the raw JSON-RPC result. Confirm the payload the server sent is continuous (first line 1, last line = served end, no host `characters truncated` marker — that phrase is not in this repo).

Model check: the same spec, same server, through the host. Save what the model quotes.

- [ ] **Step 3: Score**

| Server sent | Model received | Verdict |
|-------------|----------------|---------|
| Continuous, under ceiling | Continuous, same span | Class closed for this host at this size. File it. Still not a miss rate for other hosts. |
| Continuous, under ceiling | Head+tail / `characters truncated` | Defect. Stop this plan's engine go/no-go. New ADR (ledger vs host-cut of a fitting/under-ceiling result). Do not "fix" it in this task. |
| Page with `-- PARTIAL:` | Gapped | Compare to reading 18; if `isError` is absent and still gapped, that is a new member of the class. |

- [ ] **Step 4: File `docs/curve/reading-under-ceiling-result.md`**

Must name: date, host + version, `mrw version`, fixture line count and encoded chars, whether mrw paged, whether the model saw a gap, and "this is not ADR-039".

- [ ] **Step 5: BACKLOG**

Keep "Do not treat ADR-039 as that evidence". Either stay Deferred (could not run) or add "Observed YYYY-MM-DD: …". Unrun stays not coverage — `TestAnUnrunUnderCeilingProbeIsNotTheClassClosed` must still pass.

- [ ] **Step 6: Stop**

A defect → Task 7 (new ADR), not a silent patch.

---

### Task 4: JSX nest probe (UC-5)

**Files:**
- Create: `docs/break/jsx-nest/App.tsx` (the attack stays wrong-DOM on purpose)
- Create: `docs/break/jsx-nest/README.md` (how to run `tsc`, what "DOM wrong" means)
- Modify: `docs/adr/BACKLOG.md` JSX paragraph: still "not as a finding", or "reproduced YYYY-MM-DD"

**Interfaces:**
- Consumes: ADR-048 (no write-time parser); BACKLOG YAML indent reparent as the analogue
- Produces: reproduced / did not reproduce. If reproduced: a later record *maybe*; not a parser.

The fixture must remain a member of the class after the probe. Do not "fix" `App.tsx` to render correctly (v6: a test of today's deficiency).

- [ ] **Step 1: Write the fixture**

`docs/break/jsx-nest/App.tsx` — one outer element, a balanced inner tree whose closer binds to the wrong parent (extra wrapper or swapped close), still valid TSX.

- [ ] **Step 2: Typecheck**

```bash
cd docs/break/jsx-nest
npx --yes --package typescript tsc --noEmit --jsx react-jsx --strict App.tsx
```

Expected if the class is real: exit 0. If `tsc` is red, the fixture is not the class — fix the fixture, do not report "does not reproduce".

- [ ] **Step 3: DOM**

Render once (Vite playground, or `react-dom/server` `renderToStaticMarkup`). The README states the expected parent of the inner node vs the intended parent. Screenshot or markup dump goes in the README or beside it.

- [ ] **Step 4: File the BACKLOG sentence**

- Did not reproduce: "Probed YYYY-MM-DD, `tsc` red or DOM matched intent; still not a finding."
- Reproduced: "Probed YYYY-MM-DD, `tsc` 0, DOM parent was …; still not a parser. Needs a quote for any record."

`TestAnUnrunJsxNestProbeIsNotAFinding` still requires `not as a finding` until M accepts a record.

- [ ] **Step 5: Stop**

Reproduced → tell M, wait. Do not start ADR-048 invalidation.

---

### Task 5: Desktop trees-per-session (UC-1) — blocked on a Desktop session

**Files:**
- Modify: `docs/adr/BACKLOG.md` Desktop reach receipt (append an observation; do not delete WRONG POPULATION)
- Optional: ADR-019 Follow-ups checkbox

**Interfaces:**
- Consumes: ADR-019 pick A; BACKLOG "coder count is not evidence about Desktop"
- Produces: an integer "trees this session needed", plus — for pick C only — whether `roots/list` arrived. Not a B/C implementation.

- [ ] **Step 1: If no Desktop session is at hand, file that and skip**

Write "not run YYYY-MM-DD; still not the coder count." `TestACoderPlanCountIsNotDesktopReachEvidence` must still pass.

- [ ] **Step 2: When a session is at hand**

One analyst-shaped task (read/write more than one folder of documents). Count distinct trees touched. Note whether the host sent `roots/list` (needed only if someone later quotes pick C).

- [ ] **Step 3: Score**

| Trees needed | Action |
|--------------|--------|
| 1 | Pick A is enough for that session. File it. Do not call Desktop "covered". |
| ≥2 | Evidence to *ask* M about B/C. Do not implement. Do not grow `Apply` to a per-hunk ledger. |

- [ ] **Step 4: Stop**

---

### Task 6: Concurrent last-writer-wins (UC-3) — optional characterization, no lock

**Files:**
- Optional create: `docs/break/concurrent-last-writer.md` (dated rerun)
- Do not modify `internal/apply/apply.go`

**Interfaces:**
- Consumes: BACKLOG 20-writer observation; ADR-002 locking out of scope
- Produces: confirmation or a louder counterexample. Not a mutex.

- [ ] **Step 1: Decide whether to rerun**

Already observed (1 / 17 / 20 of 20 kept; silent `applied`). Skip if time is better spent on Tasks 2–4. A CI test of this race is flaky by construction — do not add one.

- [ ] **Step 2: If rerunning, outside CI**

20 concurrent `mrw write` against one 100-line file, each replacing a different line, three trials. Record surviving edits and whether any loser printed `applied` / exit 0.

- [ ] **Step 3: File or skip**

Do not propose a lock in the write-up. `TestLockingStaysPermanentlyOutOfScopeForConcurrentWrites` must still pass.

---

### Task 7: Only if a probe found a defect — new ADR, then three-arm stress

**Files:** unknown until the defect exists. Do not start this task speculatively.

**Interfaces:**
- Consumes: Task 3 (host still cuts under-ceiling) or Task 4 (JSX reproduced *and* M quoted a record) or Task 2 (M quoted a default)
- Produces: ADR-063+ (next free number: `grep -rn 'ADR-0[0-9][0-9]' docs/adr | …` then `ls` is not enough — grep prose reservations)

- [ ] **Step 1: `adr-write` for that one class.** One record. Out of Scope names this plan's Non-Goals.

- [ ] **Step 2: First-red lock** (`am_load_skill("first-red-lock")`). Tests table hashes Test* bodies only; no `§NN` in that table.

- [ ] **Step 3: Execute, contract row, then stress**

House recipe (`am_load_skill("stress-testing")`):

1. Fuzz vs an independent oracle (not the code).
2. Randomised package-boundary vs a reference model; pools enumerated from source (`grep` the parser / flags), name what was left out.
3. Built-binary matrix vs exit-code oracle (`$MRW`). If the defect is host-only, arm 3 does not apply — say so.
4. Baseline green, then 8–10 hand mutants; restore by inverse edit, not `git checkout --` on dirty work.

- [ ] **Step 4: Do not flip `--strict-balance` default inside a host-cut ADR** (or the reverse). One class per record.

---

## Stress of *this* plan (the pins)

After each BACKLOG edit in Tasks 2–6:

```bash
go test ./internal/adversarial/ -count=1 -run 'TestADesktopReach|TestACoderPlan|TestAnUnderCeiling|TestAnUnrunUnderCeiling|TestConcurrentLastWriter|TestLockingStays|TestTheStrictBalance|TestAnUnrunStrictBalance|TestAJsxNest|TestAnUnrunJsxNest'
```

Expected: `ok`. Then one delete-sentence mutant on the sentence just added or relied on; expected FAIL; restore; PASS.

Arm 1/2/3 of stress-testing **do not apply to the grep pins**. They apply in Task 7 only.

## Self-review

| Spec UC | Plan task |
|---------|-----------|
| UC-1 Desktop reach + wrong population | Task 5 |
| UC-2 under-ceiling, not ADR-039 | Task 3 |
| UC-3 silent applied, no lock | Task 6 |
| UC-4 campaign 5%/50/three corpora, default stays off until quote | Task 2 |
| UC-5 JSX unmeasured, not a finding | Task 4 |
| Pins / unrun ≠ coverage | Task 1 + stress-of-pins |
| New engine only on a found defect | Task 7 |

No placeholders. No synthesized pricing. No CI race. No default flip in this plan.
