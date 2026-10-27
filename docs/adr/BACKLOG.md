# Backlog — what is not yet an ADR

Deferred work and open questions, each naming the record that punted it. An
entry here is a receipt: a deferral whose pointer names this file but never
wrote anything into it passes every check and exists nowhere, so the entry is
written in the same commit as the deferral.

`adr-debt docs/adr` sweeps the deferrals in the records and expects to find them
here.

On 2026-10-21 the closed, done and obsolete entries, and the dormant leads (open, but with no
trigger and no report), moved verbatim to [BACKLOG-archive.md](BACKLOG-archive.md); a dormant lead
reopens on the trigger it names, by moving back here. What stays is what is live, and every
pre-registration.

Archived: [BACKLOG-archive.md](BACKLOG-archive.md)

## From ADR-009 (mrw counts what happens to the plans it is given)

- **A blind-agent bench: a fresh agent given only `mrw instructions` and a tree it has never seen.**
  PRE-REGISTERED 2026-09-24, before the harness exists (`.claude/rules/adr.md`). Criterion: on the
  nine read/plan tasks of the 2026-09-24 ADR-063 chaos pass, a fresh agent answers at least 8 of 9
  correctly, using at most 20 mrw calls in the whole run, on Haiku and on Sonnet, 3 runs each. A run
  that used a banned tool (grep, rg, find, ls, cat, sed, awk, head, tail, the Read/Grep/Glob/Edit/Write
  tools, or any `--help`) is VOID, not a miss, and is replaced by a fresh trial, at most 3 replacements
  per model. The verdict per model, first rule that applies: FAIL if any non-void run misses the
  criterion; otherwise INCONCLUSIVE if it has fewer than 3 non-void runs after its replacements;
  otherwise PASS. The bench passes only if both models PASS. The 2026-09-24 one-off (Haiku, 9 of 9) predates this
  criterion and is not a reading. Harness `scripts/blind-agent.sh` (scoring
  `scripts/blind-score.py`); plan and results under `docs/blind/`, first as blind reading 01.
  **Reading 03 (2026-09-24): PASS**, both models (`docs/blind/blind-03-result.md`). Readings 01 and
  02 are void on scorer defects. Haiku needed all three replacements, each VOID for `cat`.
  **Reading 04 (2026-09-24): FAIL**, both models, on the call budget: h3 and s2 took 21 calls with
  9 of 9 correct (`docs/blind/blind-04-result.md`). It removes reading 03's confound: headless
  `claude -p --safe-mode` under `/tmp`, so no repository documentation in context. Sonnet's calls
  roughly doubled (non-void runs, 23 → 49); Haiku's rose 45 → 55; correctness did not move.
  **Criterion amendment, pre-registered 2026-09-25 before reading 05's first trial** (ADR-070 T3):
  a non-void run must also make at least one mrw call to meet the criterion; 8 of 9 correct with
  zero calls did not use the tool the bench measures. `--help` stays banned as an argument word of
  any command; text that only contains it (`echo "see --help"`) is no longer a use. Re-scoring
  readings 03 and 04 under both changes moved no verdict.
  Reading 05 re-runs `blind-04-plan.md` on the build that ships ADR-070's worked `body=` header: deferred, on M's go
  (six or more headless `claude -p` runs).

## From ADR-008 (a delete says what it removed)

- ~~**An ALIAS spelling of a partially-read file bypasses the per-line ledger
  ENTIRELY, so a write to lines the caller was never served APPLIES.**~~
  **CLOSED 2026-09-09 by ADR-029 T1+T2** —
  `internal/adversarial/ledger_test.go::TestAnAliasSpellingIsTheSameFileToThePerLineLedger`,
  contract §67. Ranked as open until 2026-09-12 because this receipt was never
  struck (M: *"I NAME THEM ALL"*). The write-back two-key SHA refusal further
  down this file is a different defect and stays open. Found by
  the Codex review of PR #128, reproduced 2026-09-07 against that branch at
  `1efd1a6`. `internal/apply/apply.go:510` recovers an aliased observation with
  `sameFileEntry` — that is issue #47's fix and it must keep working — but `:553`
  looks the ledger up again by exact key and `covered()` treats the miss as
  COVERED (`:555`), so every per-line check passes:

      $ mrw --root . read 'real.txt:1'                             # line 1 only
      $ printf '@@ link.txt 4 replace\nPWNED\n' | mrw --root . write -
      ok   link.txt 4 replace  -1 +1

  with `link.txt -> real.txt`; a case-only `REAL.txt` does the same on APFS and
  on NTFS. The exact spelling is refused, as it should be. **This is larger than
  the anchor entry above, which is one visible symptom of it**: for an alias,
  ADR-002's per-line guard is not weakened, it is absent.

  **Pre-registered before the harness exists**, because the obvious fixture
  passes for the wrong reason in two different ways.

  1. A test asserting only that the alias write is REFUSED is green against a fix
     that refuses every alias — which undoes #47. It must also assert that a
     WHOLE read of `real.txt` still licenses a write spelled `link.txt`.
  2. Neither alias is portable, and each is covered on exactly one of this
     repository's two CI operating systems. The symlink half runs on Linux and
     may fail to create on Windows; the case-only half needs a case-INsensitive
     filesystem, so it runs on Windows and cannot run on Linux. Probe at runtime
     and skip — never assert the platform — and say in the record that
     `scripts/contract.sh` can carry the symlink half only, being Linux-only.

  `--root` still refuses first: checked the same day on an in-root symlink
  pointing outside the root, where read and write were both refused by name. So
  this is a contract violation against ADR-002 and ADR-005, not a privilege
  boundary — exactly as for the entry above.

## From the `$` divergence (2026-09-02, no ADR)

- **Measuring whether served size degrades edit accuracy — DEFERRED, and it is **Closed 2026-10-21: note rewritten from docs/curve/: the curve was run and it bent for one client only. Readings 2 and 3 (Sonnet, read arm) were flat, 42/45 and 45/45, the three misses at target+2. Reading 4 (Haiku, read arm) fell 15/15, 12/15, 8/15 across 2, 20 and 200 KB. Reading 5 traced every miss to the served text's row index (target-117 with the window from line 120), not to size. With the bare tool-result delivery Haiku scored 15/15 at all three sizes (readings 8, 9); an outer row number equal to the index brought the miss back (readings 11, 13); gpt-5.6-sol scored 45/45 (reading 16) and Sonnet 45/45 on a thirteen-service fixture (reading 17); readings 6 and 7 are void and reading 14 is evidence-limited. So no size effect is shown, and the bend is a row-index misreading, not a context-size effect. Not answered: the MCP delivery arm at 200 KB (reading 18 void because the host truncates the page, readings 12 and 19 void, reading 20 covers 2 and 20 KB only, 30/30); that is the open measurement under 'A host-truncation measurement of an under-ceiling result'.**
  the most valuable unanswered question this project has.** Named in ADR-014.

  `MaxResultChars` is 200,000 because that fits under Claude Code's per-tool
  ceiling. It bounds a RESOURCE and says nothing about quality. What nobody has
  measured is whether a model's next edit gets WORSE as more context is served:
  the reported degradation with long context is about retrieval and QA, and edit
  authoring is a narrower, more mechanical task it has not been measured on.

  ⚠ The obvious instrument is the wrong one. ADR-014 records peak RSS against
  served bytes (about 12x the served size over MCP, flat on the CLI). That is
  what the SERVER spends. The question is what the CALLER's accuracy does, and
  using a memory curve to justify a context budget is the same category error
  ADR-012 rejected one level up — a measurement pointed at the wrong population.

  Answering it needs the benchmark harness: served bytes as the independent
  variable, edit outcome as the dependent one, across models. If the curve turns
  over, the cap becomes a measured number instead of an inherited one; if it is
  flat, "arbitrary" is a defensible answer and we will be able to say so.

  **PRE-REGISTRATION, written 2026-09-04 BEFORE any harness exists or any datum
  was collected.** It is here rather than in a record because a criterion
  authored after the first look is not a criterion.

  ⚠ THE DEPENDENT VARIABLE IS THE WHOLE PROBLEM, and the obvious ones are
  traps. ADR-009 already counts `applied`, `refused_parse` and `refused_apply`,
  so they are free — and they will be FLAT, because they measure whether the
  caller could author the FORMAT, which has nothing to do with how much was
  served. A flat curve on those would read as "the cap does not matter" when it
  actually means "the easy thing was measured". They are recorded as secondary
  and are not the claim.


  ⚠ AND "FREE FROM THE TALLY" NEEDED CHECKING, because ADR-012's Context
  records that tally as per-checkout and unable to ATTRIBUTE. Verified
  2026-09-04 before the design was allowed to depend on it: with a fresh
  `XDG_STATE_HOME` per trial, `mrw stats` reports that trial alone —
  `applied 1 of 1 plan(s)`, one plan recorded. So the secondary DVs are
  per-cell readable, and the isolation is the state dir rather than the
  checkout. If that ever stops holding, the fallback is parsing each
  `mrw_write` result directly, which is a different code path and should be
  named as such rather than quietly substituted.
  **PRIMARY DV: did the plan address the line it was supposed to address?**
  This is the failure mode that matters and the one every other gate is blind
  to — an edit that parses, applies cleanly, reports `ok`, and changed the
  wrong line. `refused_parse` and `refused_apply` both stay green through it.
  It is also the failure mrw exists to prevent, so it is the honest thing to
  put on the y-axis.

  GROUND TRUTH IS PLANTED, NOT JUDGED. The generator places the target line and
  therefore knows the correct address by construction; scoring is
  `plan addressed line N` against a known N, mechanically. Nothing here needs a
  human to rate an edit, which is what keeps the measurement affordable and
  repeatable — and it is the reason this DV was chosen over "is the edit
  semantically right", which cannot be scored without authoring a rubric and a
  rater.

  ⚠ THE THREAT TO VALIDITY, named up front: a planted target is easy to find if
  it is a unique string, and then the task measures string matching rather than
  reading. The distractors must be NEAR-IDENTICAL to the target — same shape,
  differing in the detail the instruction names — or the curve will be flat for
  a reason that has nothing to do with context size.

  IV: served bytes, varied by padding around the target on both sides.

  ⚠ "THE TASK IS HELD FIXED" WAS FALSE AS FIRST WRITTEN, and the correction
  matters because it is the difference between a clean manipulation and two
  variables moving at once. Padding adds LINES, so the line-number space grows
  and the address the model must produce changes by construction. It cannot be
  held fixed while size varies — the honest statement is that the INSTRUCTION
  and the target CONTENT are identical across cells, the address is not, and
  target position is a stratum rather than a nuisance.

  ⚠ DISTRACTOR COUNT IS A SECOND IV HIDING INSIDE THE FIRST, and this is the
  one that would make a real curve uninterpretable rather than merely absent.
  If the padding IS near-identical distractors, then larger cells have more
  near-misses and "more context" cannot be separated from "more candidates".
  So: the number of near-identical distractors is HELD CONSTANT across size
  cells, and the remaining padding is inert content that is obviously not a
  candidate. Distractor density is a legitimate second experiment; it is not
  this one, and one curve answers one question.

  STRATIFY BY TARGET POSITION rather than randomising it away. Position within
  the served window is the best-documented effect in the long-context
  literature, and if it is folded into the noise the headline curve will
  average a real effect into nothing. Early / middle / late are separate cells.

  REPEATS: the model is nondeterministic, so N trials per cell with the count
  fixed in advance. A cell is a proportion, and proportions need their interval
  reported, not a point.

  **DIRECTION AND THE FLAT ANSWER, committed now.** The prediction is that
  correct-address rate DEGRADES as served bytes grow, most sharply for
  mid-window targets. **If the curve is flat, that is a publishable answer and
  the cap stays arbitrary with evidence** — it does not become a reason to
  search for a different metric until something bends. ADR-009 and ADR-012 both

  **RECEIPT, 2026-09-04 — the instrument exists, the reading does not.** ADR-020 (PR #85) built
  the harness this pre-registration describes: `curve generate` / `score` / `tally`, no model
  client, scored by applying. **The FIRST READING is deferred to ADR-020's Follow-ups**, with the
  criterion above unchanged and not to be re-derived. This paragraph is the entry `adr-debt` looks
  for when the record's Out of Scope names this file.
  refused criteria that could not go red; this one commits to accepting a null.
- **MCP coverage for the Desktop population — the largest open product
  question in this repository.** M, 2026-09-04: *"we need wider MCP coverage,
  which basically calls local mrw under the hood … the potential here IS HUGE,
  for analysts reading and writing mega documents into csv files … humans aren't
  structured in their daily work … this is the single biggest feature that we
  ship only for coders but not desktop users."*

  WHY IT IS NOT JUST "ADD THE FLAGS". The capability gap and the reach gap are
  different problems with different answers, and only some of the CLI surface
  is even meaningful to this population:
  - `--grep` and `--files-from` are the ones that matter — finding the sites is
    exactly what an analyst cannot do by hand across many documents.
  - `--check` is a Go-test runner scoped to changed files; it means nothing for
    a folder of CSVs, and shipping it would be cargo.
  - `iter`, `seen`, `stats` are introspection a Desktop user has no use for.
  - The ROOT is the hard part, not the flags. See the correction above.

  WHAT TO DECIDE FIRST, before any of it: whether an analyst's answer is a wider
  MCP tool set, or one tool whose spec language already covers finding (mrw's
  read specs take a regexp) plus a root model that fits scattered documents.
  Those are different records. Needs its own ADR, with M's scope decision, and
  it should NOT be inferred from ADR-016 — that record deliberately covers only
  what the surface SAYS, and its refusal of parity is scoped to itself.

  **M ANSWERED THE SCOPE QUESTION on 2026-09-04: both, as two records.** The
  CAPABILITY half is done — ADR-017 shipped `grep` and `exclude` on `mrw_read`,
  with a match INDEX when the results will not fit and `after` to page it. It
  also settled `--files-from` permanently on the flag's own documented
  rationale: it exists to undo shell word-splitting, and MCP has no shell.

  **THE REACH HALF IS STILL OPEN AND IS FILED HERE.** ADR-017, its T1 and its
  T2 each defer "any root or multi-root change" to this entry, and this
  paragraph is their receipt — they pointed here first at a filename
  (`ADR-018`) that did not exist, which `adr-debt` correctly refused as a
  pointer to nowhere.

  What the reach record has to decide, carried forward so it is not re-argued:
  repeatable `--root`, ALLOW-ONLY (a deny list needs symlink resolution and
  reintroduces every path-handling bug already fixed), resolved once at
  startup, the boundary set by the launcher rather than by a file mrw can read
  and therefore widen, and one state namespace per root. `state.Dir(root)`
  already hashes the resolved absolute root so N namespaces come free, but
  `apply.Apply(root, …)` takes ONE root and `seen.Load(root)` loads ONE ledger
  per run — a cross-repo plan needs the ledger resolved PER HUNK, which changes
  what a run is.

  ⚠ **#81 IS DONE AND REACH IS NOT — THEY SEPARATED.** When #81 was filed this
  entry said it "belongs to that record too". It does not any more. ADR-018 took
  it, because the axis turned out to be EXPLICIT VERSUS ACCIDENTAL rather than
  which directory is unpleasant: `ResolveRoot` already returns a `Source`, so
  "did anyone say this?" is a value that exists. A fallback onto a filesystem
  root or the home directory is refused; anything explicit is honoured, `/`
  included. That rule says nothing about how MANY roots there may be, which is
  why it could ship without waiting for this entry.

  ⚠ **AND THE NUMBER MOVED.** ADR-017's Out of Scope, its T1 and its T2 all say
  reach is `ADR-018`. That was true when they were written and is now wrong:
  **018 is the root guard, and REACH IS ADR-019**. The deferrals point at this
  entry rather than at a number precisely so a renumbering cannot break them —
  which is the reason `adr-debt` refused the original `ADR-018` pointer as a
  pointer to nowhere.

  **RECEIPT, 2026-09-10 — ADR-019 drafted as Proposed.**
  `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md` exists.

  **RECEIPT, 2026-09-10 — Naming pick: A — launch --root only.**
  M: *"A"*. Status Accepted. Fork 3: `mrw mcp` stays single-root. Do not
  measure Desktop `roots/list`. Do not implement B or C. T1 encoded the pick;
  T2 pins two-root isolation; T3 teaches launch `--root`. The historical
  analysis above is what the record carried forward; it is not deleted.

  **From ADR-019 Follow-ups, kept here so the record's deferrals have a receipt:**
  - A Desktop-population measurement of how many trees one session actually
    needs. The coder count is not that measurement.
  - Whether Claude Desktop sends `roots/list` — filed for pick C only. Pick A
    (ADR-019:108-110) wires no client request, so this is measured only if a
    reading ever argues for C. ADR-011 deferred the same client request here;
    this is that receipt too — the follow-up trigger (second host;
    `CLAUDE_PROJECT_DIR` host-specific) is met.
    Not run 2026-09-16 — this session is Cursor, not a Claude Desktop MCP session. Still not the coder count. Pick A stands.

  **Desktop reading 0 — 2026-09-24, this Mac. NOT the analyst measure.** Claude
  Desktop has `mrw` registered as `--root /Users/zy/GolandProjects/tool-multipathreadwrite
  mcp`. `~/Library/Logs/Claude/mcp-server-mrw.log`, 2026-09-04 → 2026-09-24:
  84 launches (`mrw mcp: serving` lines, all this one root), 34 `initialize`,
  1 `tools/call`, 0 `roots/list`, 0 `outside the root`. Population: one coder's
  machine, where the CLI is the tool of choice. It says Desktop reach is idle
  here; it says nothing about what an analyst session needs. Trees-per-session
  stays unmeasured. Reading 1 is one analyst task M runs in Desktop.

  **Desktop reading 1 — 2026-09-24, a Windows 11 Pro 10.0.26200 machine, Claude
  Desktop Store build, mrw v1.22.3 (`63729bd`, checksum matched SHA256SUMS). A
  CODER task, NOT the analyst measure** — the folder M picked was the
  quality-harness repository. Task: compare two ADR task folders in different
  branches of the tree and write a summary into a third place. From the
  Desktop log (`%LOCALAPPDATA%\Claude\Logs\mcp-server-mrw-docs.log`; the Store
  build's config lives under `%LOCALAPPDATA%\Packages\Claude_<id>\LocalCache\Roaming\Claude\`,
  not `%APPDATA%\Claude`): 3 `serving` lines, all one root; Desktop launched
  the server twice within 0.4 s and shut the first down before `initialize`;
  5 `tools/call` over ~66 s; 0 `roots/list`; 0 `outside the root`. Per-tool
  counts are NOT observable — this log format omits the tool name; the 4 reads
  + 1 write split is inferred from the output file's mtime. Trees this session
  needed: 1. Taken by a peer session on M's machine; reading 2 (an analyst task
  over documents) is still the open measure.

  **Reading 2 protocol — refreshed 2026-10-03 for v1.47.0 (plan item 5). M runs it; nothing ships for it.**
  - Setup: Claude Desktop with `mrw --root <the analyst folder> mcp` (launch `--root`, ADR-019 pick A), mrw
    v1.47.0 or later checked against `SHA256SUMS.txt`. The folder holds documents, not a repository: no
    `.git`, no `go.mod`, no `.quality-harness.json`.
  - Task: one analyst task — read several large text, markdown or CSV files across subfolders, pull named
    fields out of them, write one CSV and a short summary. Phrase it as a person would, without naming mrw.
  - Record from `mcp-server-mrw*.log` and the transcript: the version and build; the `serving` lines and
    their roots; counts of `initialize`, `tools/call` (per tool where the log names it), `roots/list` and
    `outside the root`; how many trees the session needed — the open measure; which of `grep` (and its
    INDEX and `after` paging), `files_from`, `max_lines` and `stat` were used; whether any write ran a check
    (none is due in such a folder, ADR-054/113 — one that ran is a finding) or asked for `then` (ADR-115);
    every refusal and whether the session recovered; any `-- PARTIAL:` page and any checkpoint not acked;
    wall time.
  - Write here: trees needed, whether one launch root sufficed, and each argument the analyst never touched.

## From ADR-020-T2 (a target the instruction does not name)

- ~~**Run a reading against the relational fixture.**~~ **CLOSED 2026-09-05 by reading 3** (#96, `docs/curve/reading-03-result.md`): 45 of 45 under the pre-registered criterion, flat, and the prediction that the relational fixture is harder refuted (ADR-020 Follow-ups). T2 builds the selector; it does not spend the
  trials. The criterion is the one already pre-registered above and must not be re-derived — correct
  address rate against served bytes, stratified by position, refusals reported separately, a flat
  curve accepted as an answer. What the first reading adds is that the NAMED fixture is at ceiling
  (42/45, `docs/curve/reading-02-result.md`), so a relational reading is the first one whose curve
  has room to bend. It is a budget decision, which is why ADR-020 keeps readings out of its tasks.
## From ADR-027 (an empty file is created on purpose, or not at all)

- **A successful alias-spelled write leaves ONE file recorded under TWO ledger
  keys, and the recorded spelling then gets a refusal that is not true.**
  Measured 2026-09-07 while writing ADR-029, against the v1.4.0 binary. With
  `link.txt -> real.txt`, reading `real.txt:1-3` and writing `@@ link.txt 2`
  succeeds and `mrw seen` shows both:

      4c650896  the whole file            link.txt
      880553fc  lines 1-3                 real.txt

  A later plan naming `real.txt` is then refused with

      real.txt changed since mrw last saw it (recorded 880553fc, now 4c650896):
      re-read it before editing, or pass --force to overwrite blind

  which says something false — mrw made that change itself, one call earlier.
  `sameFileEntry` does not help here: the exact key IS present, so no alias
  recovery runs, and the SHA comparison is against the stale entry.

  **Not fixed by ADR-029, and deliberately.** That record resolves identity for
  the per-line gate on the READ side; this is the WRITE-back side, where two
  keys are created rather than one consulted. It fails SAFE — it refuses, it
  never applies — so it costs a caller a re-read and a confusing sentence rather
  than a wrong edit. Fixing it means either reconciling SHAs across aliases in
  the file-level check, or keying the ledger on resolved identity, which
  ADR-029's Alternatives rejects because `mrw seen` prints the keys the caller
  typed. Promote it if a caller reports the refusal in the wild; the entry
  exists so the next reader does not mistake it for part of ADR-029.

  Reproduced on v1.69.0, 2026-10-21: after a write through link.txt, a write to real.txt is refused 'changed since mrw last saw it'; fails safe, the message is false.

## From ADR-034 (state that names a checkout nobody has)

- **A prune that refuses when the marker's volume is absent.** PRE-REGISTERED, so the criterion
  predates the first report rather than being written to fit it. ADR-034's exact test — the `root`
  marker names a path that is not a directory — cannot distinguish a deleted checkout from one on
  an unmounted volume. Today the operator is the gate: the command is explicit, `--dry-run` shows
  the list, and a lost ledger costs a re-read and cannot cause a wrong edit (ADR-002) — though the
  iteration working set and the authoring tally in the same directory are lost outright.

  **What would promote this:** one report of a real `--prune` that removed state for a checkout on
  a volume that was merely unmounted. The answer is then a refusal for entries whose marker names a
  path under an absent mount point, NOT an age gate — an age gate deletes the same entries a
  fortnight later and adds a number nobody can defend. ADR-034's Alternatives rejects the age gate
  in advance for that reason.

## From ADR-054 (a write that applied can still leave a broken tree)

### From ADR-055 (v1.17.0)

- **The pricing campaign had no data source** — **closed by ADR-056 T2**
  (`mrw stats` pricing block; contract §96). The criterion above is unchanged;
  only its source sentence was corrected.

### Pre-registration: a default `--strict-balance`

Written 2026-09-13, BEFORE any campaign, so the criterion is not shaped by the
result (this file's rule). ADR-055 T3 ships the flag opt-in. It may be argued
as a default only if a campaign over three real corpora — Zeus, this
repository, playtrix — reads `mrw stats --json` `.pricing` in each checkout
(ADR-056: mrw prices the flag as writes land, because ADR-009 holds no plan
text to replay after the fact) and reports, per corpus: how many landed writes
the flag would have refused (`strict_would_refuse`); of those, how many went on
to break the tree (`_broke`: the check ran and failed) versus how many did not
(`_held`: the false positives). `_unchecked` counts neither way and is reported
so an MCP-heavy corpus cannot flatter the flag. Criterion:
**false positives under 5% of refusals in every corpus, with at least 50
refusals total**, or the default stays off and the flag stays what it is. A
campaign that reports the true-positive count without the false-positive count
does not qualify — the number that argues a default is the one that costs the
caller, not the one that flatters the tool. Arm the campaign with *"price
strict balance"*.
A named corpus with no checked refusals (`broke+held == 0`) has no defined rate and does not pass.

Ran 2026-09-13 against PATH `mrw` v1.18.0 (`a5c4f24`). `mrw stats --json`
`.pricing` in each checkout (landed writes are the older tally; pricing only
counts writes that landed on a binary that prices):

| corpus | landed | candidates | would_refuse | broke | held | unchecked |
|---|---|---|---|---|---|---|
| this repository | 307 | 0 | 0 | 0 | 0 | 0 |
| Zeus | 296 | 0 | 0 | 0 | 0 | 0 |
| Playtrix | 127 | 0 | 0 | 0 | 0 | 0 |

**Does not qualify** — 0 refusals total; the bar is ≥50. Default stays off.
v1.18.0 shipped the same day; earlier landed writes were not priced. Same
quote re-arms a later read. Do not synthesize wrap-tails to fill the
counters: the criterion is real corpora.
Ran 2026-09-16 against PATH `mrw` v1.22.0 (`6896442`). Same three checkouts as 2026-09-13. `mrw stats --json` `.pricing` (landed is the tally, not `.pricing`):

| corpus | landed | candidates | would_refuse | broke | held | unchecked |
|---|---|---|---|---|---|---|
| this repository | 320 | 0 | 0 | 0 | 0 | 0 |
| Zeus | 360 | 26 | 1 | 0 | 0 | 1 |
| Playtrix | 135 | 4 | 0 | 0 | 0 | 0 |

**Does not qualify** — 1 refusal total (Zeus, `_unchecked`); the bar is ≥50. Zeus `broke+held == 0` so the FP rate is undefined and that corpus cannot pass "under 5% in every corpus with checked refusals". Default stays off. Same quote re-arms a later read. Do not synthesize wrap-tails.

Stress of that same binary, 2026-09-13: `FuzzBalanceDelta` ~2.06e6 execs
clean; `TestRandomisedApplyBalanceFollowsTheDecision` seeds 54/1/7/13/99
(400 ops each, all five parser ops) clean; break campaign 47 = 47, no exit
diff vs `docs/break/campaign-v1.18.0.txt`; random CLI matrix 440 writes
across seeds 118/7/13/99, exits only in {0,1,2,3}, zero crashes, ADR-001
held on every exit 1, `--json` receipts carried `pattern`, constructed
wrap-tails with `--strict-balance` refused at exit 1. `*** Delete File:`
and a hunk-less `*** Move to:` compile and apply (ADR-057); Move to with
extra `@@` hunks still compile-refuse at exit 2 and leave the tree.

## Pre-registered for ADR-119 (indent and closer advisories) — the bar, written before any measurement

Zy, 2026-10-02 ("<5% and 5/5 fixtures"). Registered here, in its own commit, before the heuristics are measured,
so the result cannot move it:

- **closer** — fires on a `replace` whose last non-blank body line equals the line right after the replaced range
  (when that line exists and is not blank). **indent** — fires on a non-prose `replace` whose first or last
  non-blank body line's leading whitespace differs from the first or last replaced line's.
- **False positives:** replayed from the non-merge history of every repository under `~/GolandProjects` and
  `~/CursorProjects` (each modification hunk of `git diff -U0 k^ k` as a replace, vendored, generated, lock and
  `node_modules` files excluded, hunks capped per repository), each heuristic must fire on **under 5% of those
  replaces in every language bucket** that has at least 50 of them.
- **True positives:** each must catch **5 of 5** field fixtures (the offsets recorded under "From the field
  reports" above: Blade `@endif`, HTML `</div>`, markdown fence, YAML block scalar, Ansible `when:`) — closer the
  closer cases, indent the indentation cases, and together all five.
- A heuristic that misses its bar does not ship; the record says so with the numbers.
- **Amended after the first measurement (Zy, 2026-10-02, "K4 closer, defer indent").** The first run (seed 119,
  400 replaces a repository, 3,475 replaces from 16 repositories) missed the bar on both registered heuristics:
  closer caught 2 of its 3 fixtures — the markdown fence's survivor is the fourth line after the range, which a
  line-right-after comparison cannot reach — at 0.00% everywhere; indent caught both of its fixtures but fired on
  19.35% of `.py`, 14.29% of extensionless, 11.89% of `.js` and 8.99% of `.php` replaces. Registered now, before
  the second run, and chosen AFTER seeing the first run's numbers, which is why it is measured again on a fresh
  sample rather than shipped on those: **closer** fires on a `replace` whose last non-blank body line, trimmed,
  is a closer-shaped token (` ``` `/`~~~` fences, a run of `}` `]` `)` with trailing `;` `,` `)`, `</tag>`,
  `@end…`, `end`, `fi`, `done`, `esac`) and equals one of the next 4 non-blank lines after the range, trimmed. The
  bar is unchanged: under 5% in every bucket of 50 or more, and 3 of 3 closer fixtures. The second run uses seed
  2119, 1,200 replaces a repository and 6,000 commits sampled a repository (`stress.py --seed 2119 --cap 1200
  --max-commits 6000`), so it overlaps the first run's hunks only partly. **indent** is withdrawn (see "From
  ADR-119").
- **Amended after review (Zy, 2026-10-02, "Refine and re-measure").** The in-process review of PR #322 found
  that run 2's corpus could not contain the commonest correct shape: `git diff -U0` hunks never hold an unchanged
  closer, so a replace THROUGH its own closer (`if x {…}` replaced through its `}`, the outer `}` below) was never
  replayed, and the shipped closer fires on it. Registered now, before the third run: **closer** additionally
  requires that the replaced range's last non-blank line, trimmed, is NOT equal to the body's last non-blank line,
  trimmed — the body brought a closer the range did not end in. The corpus gains a second replay: each `-U0`
  replace whose next 4 non-blank original lines hold a closer-shaped line is ALSO replayed extended through the
  first such line, its range ending there and its body carrying the same unchanged lines — the shape callers are
  taught to write. The bar is unchanged and applies to EACH replay separately: under 5% in every bucket of 50 or
  more, and 3 of 3 closer fixtures. The third run uses seed 3119, 1,200 replaces a repository and 6,000 commits
  (`stress.py --seed 3119 --cap 1200 --max-commits 6000`).

## Pre-registered for ADR-156 (does mrw pay for itself on agent runs) — the bar, written before any run

Zy, 2026-10-14: both Sonnet 5.5 and Opus 5.5; our own oracle-graded tasks and a SWE-bench Verified slice; configuration B with mrw forced; the subscription only. Registered here, in the record's first commit, before `bench`, any task or any run exists, so a result cannot move it:

- **Unit and pairing.** An attempt is one `claude -p` run of one task, in one configuration, on one model, in a fresh worktree. Attempts are paired by task within one model and one task source, and never pooled across models or sources.
- **Primary outcomes.** *Solved*: the oracle exits 0 (SWE-bench: the official harness says resolved). *Silent mis-edit* (own tasks only): the run's stream-json result is a success, and the tree differs from the attempt's base outside the task's allowed regions, or a touched file fails its structural check.
- **Cost.** The `total_cost_usd` of the CLI's stream-json result; the raw input, output, cache-read and cache-write tokens are recorded beside it.
- **Verdict**, per model and task source, on the admitted attempts, with a paired bootstrap over tasks (10,000 resamples, seed 156) and 95% intervals:
  - **PAYS** — the mean per-task relative cost (B − A) / A is −10% or lower, its interval lies wholly below 0, and the interval of the solved-rate difference B − A has its lower bound at −5 percentage points or above.
  - **LOSES** — the cost interval lies wholly above 0, or the solved-rate interval lies wholly below 0.
  - **INCONCLUSIVE** — neither.
  - **Where PAYS and LOSES both hold** — the cost interval wholly below 0 and the mean at −10% or lower, while the solved-rate interval has its lower bound at −5 points or above yet lies wholly below 0 — **LOSES wins.** A clarification decided by Zy on 2026-10-14, before any stage ran; the bar's numbers are unchanged.
- **Configuration B's shell reads** (`cat`, `grep`, `head` and their kin given a path) are counted, not refused, and the count is reported beside every verdict. Decided by Zy on 2026-10-14, before any stage ran.
- **Silent mis-edits** are counted per configuration and printed beside every verdict; if B has more than A, the headline says so whatever the verdict.
- **Admission.** The only exclusions:
  - an attempt that ended before its first tool call with an authentication, rate-limit or CLI start error, re-queued at most twice;
  - an attempt `bench` could not set up or grade (worktree, oracle start), re-queued once.
  Every excluded attempt stays in the receipts. An agent that gives up, errs or runs past the budget (20 minutes, 60 turns) is unsolved, with its cost counted.
- **Sample.** Stage 1: a pilot of 5 tasks × 2 × 1, not counted, then 30 tasks × 2 configurations × 2 repeats on Sonnet 5.5. Stage 2: the same on Opus 5.5. Stage 3: 20 SWE-bench Verified instances (the lowest `sha256("adr-156:" + instance_id)`) × 2 × 1 on each model, labelled small.
- **Power, said first.** Thirty paired tasks cannot show a small difference in solved rate. The bar asks for non-inferiority within 5 points, not equality, and INCONCLUSIVE is a result.
- **Publication.** PAYS, LOSES and INCONCLUSIVE are published alike, in `docs/bench/RESULTS.md` and one README sentence each.
- **Not measured now, recorded so it is not forgotten:** other hosts (Codex, Cursor, Claude Desktop); mrw over MCP rather than the CLI; an API-key run with `--bare`, which returns only on Zy's word if the subscription canary fails.

## Pre-registered for ADR-158 (Stage 1b) — written before any Stage 1b run

Zy, 2026-10-15, after Stage 1 came out INCONCLUSIVE: fix what Stage 1 showed was the harness's or the corpus's, add tasks a global substitution gets wrong, and run Stage 1b. Registered in ADR-158's first commit, before any of its tasks or runs exist:

- **The bar is ADR-156's**, unchanged: unit and pairing, outcomes, cost, the PAYS / LOSES / INCONCLUSIVE rule with LOSES winning an overlap, admission and exclusions, the budget, B's counted shell reads and slipped writes, the silent mis-edit headline.
- **Sample.** A pilot of 5 tasks × 2 × 1 — three `varied-multi-site`, at least one agentsmemory — stage `pilot`, not counted; then every measured task not superseded — Stage 1's 28 kept tasks, `site-html-cost-faq-2`, `am-go-sha-message-2` and ten `varied-multi-site` tasks, 40 in all — × 2 configurations × 2 repeats on Sonnet 5.5, stage `stage1b`, every attempt on this Mac, Stage 1's host.
- **After the pilot**, a task is superseded only for an oracle defect shown on its base or its reference fix, never for what an attempt did; a superseded task is replaced one for one, so the stage stays at 40.
- **Never pooled.** Stage 1b's verdict is its own; it is not combined with Stage 1's, whose verdict stands as computed.
- **Descriptive rows, registered now:** the `check_red_at_base` row, as in Stage 1; and one more over the ten `varied-multi-site` tasks alone, those red at base included, printed beside the verdict in parentheses and never in its place.
- **What changed from Stage 1, all in both configurations:** the attempt's shell is bash, so no zsh startup file is read; the CLI's task directory, git's global configuration and Python's user site are not the user's; and agentsmemory's "## Gate" and "## When the tools are absent" sections are dropped from its `AGENTS.md`.

## Pre-registered for ADR-156 T4 (Stage 2 on Opus) — written before any Stage 2 run

Zy, 2026-10-15, after Stage 1b and ADR-159: Stage 2 runs the corpus as it stands, not Stage 1's thirty. ADR-156's registration said "Stage 2: the same on Opus 5.5", written before three of Stage 1's tasks were found misgraded and superseded and before ADR-158 closed Stage 1's isolation gaps; Zy chose "Current 40" over "Stage 1's 30 as is". Registered here, before any Stage 2 attempt exists:

- **The bar is ADR-156's**, unchanged: unit and pairing, outcomes, cost, the PAYS / LOSES / INCONCLUSIVE rule with LOSES winning an overlap, admission and exclusions, the budget, B's counted shell reads and slipped writes, the silent mis-edit headline.
- **Sample.** A pilot of Stage 1b's five pilot tasks (`am-vms-stamp-builds`, `qh-vms-listing-timeout`, `site-vms-social-metadata`, `am-go-sha-message-2`, `site-html-cost-faq-2`) × 2 × 1 on Opus 5.5, stage `pilot`, not counted; then every measured task not superseded — Stage 1b's forty with `cac-rename-post-send-2` in place of `cac-rename-post-send` — × 2 configurations × 2 repeats on Opus 5.5, stage `stage2`, every attempt on this Mac.
- **The attempt environment is Stage 1b's** (ADR-158 Decision 1 and 2), in both configurations.
- **After the pilot**, a task is superseded only for an oracle defect shown on its base or its reference fix, never for what an attempt did, and replaced one for one, so the stage stays at 40 (ADR-158's rule).
- **Never pooled.** Stage 2's verdict is its own, one per model as ADR-156 registered; it is not combined with Stage 1's or Stage 1b's.
- **Descriptive rows, registered now:** the `check_red_at_base` row; the ten `varied-multi-site` tasks alone; each printed beside the verdict in parentheses and never in its place.

## Pre-registered for ADR-156 T5 (Stage 3, the SWE-bench slice) — written before any Stage 3 run

Written on 2026-10-15, after T5's feasibility probe and before any Stage 3 attempt, and before Zy's word to start it (T5 S3). ADR-156's registration fixed the selection rule, the harness grading and "20 SWE-bench Verified instances × 2 × 1 on each model, labelled small"; this fixes how an attempt is built, graded and ordered:

- **The bar is ADR-156's**, unchanged, one verdict per model; silent mis-edits are not measured on this slice (ADR-156 Decision 4), and `bench analyze` prints "not measured" there, not a count.
- **The slice.** `bench/swe-bench/selection.json`: the dataset `SWE-bench/SWE-bench_Verified`, test split, revision `78f471bf655a3137b2e8a75af1501690ec009ec3` (500 instances), and its 20 lowest `sha256("adr-156:" + instance_id)`; graded by `swebench` 5.0.2.
- **An attempt** runs on this Mac as every other stage's does (ADR-156 Decision 2, Stage 1b's environment): its tree is the instance's repository at `base_commit`, built from a local clone named in `bench/repos.json`; its prompt is the instance's `problem_statement` and nothing else of the row — no hints, no test patch, no names of the tests that grade it. The attempt has no per-instance Python environment, so it cannot run the repository's tests unless it builds one itself; that holds for both configurations alike.
- **Grading.** The tree's whole change against its base, new files included, is the prediction; the official harness runs it in the instance's image (`linux/amd64`, under emulation on this Mac); `resolved` is solved. A tree with no change is unsolved and the harness is not started. A harness run that wrote no report is the registered "could not grade" exclusion, re-queued once.
- **Order.** Instances run in the selection's order, and an instance's attempts — both models, both configurations — run one after another in the seeded order among themselves, so its image is pulled once and removed after they are graded: twenty images (10.6 GB for the first) do not fit on this disk together (T5 S2). This replaces Stages 1 and 2's seeded order across tasks, for this stage only.
- **Pilot.** The first selected instance, `matplotlib__matplotlib-22719`, × A/B × 1 on Sonnet 5.5, stage `pilot`, not counted; every transcript is read before the stage runs, and a harness defect is fixed and the pilot rerun into a new directory, the old one kept.
- **Sample.** Then 20 instances × 2 configurations × 1 repeat × Sonnet 5.5 and Opus 5.5, stage `stage-3-swe-bench`, one directory, 80 attempts.

## Pre-registered for ADR-160 (Stage 4) — written before any candidate is drawn

Zy, 2026-10-15, after Stages 1b and 2 came out INCONCLUSIVE with both configurations near ceiling: a harder stage in the categories where mrw's mechanism should matter, chosen by rule. Registered in ADR-160's first commit, before `bench mine` exists or any Stage 4 task, pilot or run:

- **The bar is ADR-156's**, unchanged: unit and pairing, outcomes, cost, PAYS / LOSES / INCONCLUSIVE with LOSES winning an overlap, admission and exclusions, the budget of 20 minutes or 60 turns, B's counted shell reads and slipped writes, the silent mis-edit headline. One verdict per model and per task source (Stage 4's own tasks; the `swe-multi-hunk` slice), never pooled with Stages 1 to 3.
- **Selection, in the order of `sha256("adr-160:" + key)`:**
  - `wide-rename`, 10 tasks: a Go identifier in agentsmemory at a pinned commit of `origin/main`, declared once in the module, in no string literal and no generated file, every whole-word occurrence outside comments an identifier token, 15 or more occurrences across 8 or more files counting comments (key: the identifier). The new name has the same byte length, is absent from the repository and is fixed before any attempt; the reference renames comments too and must pass `go build ./...`, `go vet ./...` and `gofmt -l`, or the candidate is skipped for the next, the skip recorded — the build gate is part of selection.
  - `check-coupled`, 8 tasks — amended by Zy on 2026-10-15 ("Construct from the tree") before anything was drawn for it, since agentsmemory's history up to `d038def` holds no commit the first rule admits (248 commits of the right shape, none changing only literals): a string literal in a non-test `.go` file of agentsmemory at `d038def`, not an import path or a struct tag, starting with a letter, appearing exactly once in that file and in no other non-test file, whose value a `_test.go` file in the same directory also holds as a literal (key: the path and the value); the new value flips its first letter's case; the instruction comes from a template, "In <file>, change the string <old> to <new>, and keep the tests passing."; the reference changes the source and every test file of the directory holding it; the package's tests must pass at the pinned commit, fail when only the source literal is changed, and pass on the reference, or it is skipped, the skip recorded (the middle condition added before any check-coupled task was built: without it a literal no test asserts gives the task two answers).
  - `swe-multi-hunk`, 20 instances: SWE-bench Verified at revision `78f471bf655a3137b2e8a75af1501690ec009ec3` whose gold patch has 5 or more `^@@ -` hunk headers, Stage 3's twenty left out (key: the instance id).
  - The control: the six existing `single-line` tasks.
  - The 24 own-task ids are written to `bench/stage-4-tasks.txt` and run by `--task`.
  - The code at the drawing commit is the rule: the predicates are what `bench mine` and `bench swebench-select` implement at the commit T2 draws with, named in T2's sign-off; a gap between this prose and that code found after the draw is recorded beside the stage, never fixed by redrawing. A `check-coupled` commit may also touch a non-Go file holding the same literal, judged by the same token rule.
- **Calibration on configuration A only, outside the sample:** `wide-rename` candidates 11–14 and `check-coupled` 9–12 × A × 2 on Sonnet 5.5, stage `pilot`. At 7 or more solved of 8, the floor is raised once (`wide-rename` 25 occurrences across 12 files; `check-coupled` a literal held by 2 or more test files), the sample redrawn and the earlier draws deleted before any counted run; if A still solves 7 or more on the new floor's candidates outside the sample, the category runs as drawn and is reported at ceiling. No task is kept or dropped for an outcome, and no measured task is a calibration task. Harness pilots — the first 2 tasks of each own category × B × 1, and the first instance of each SWE-bench repository Stage 3 did not use × A/B × 1 — choose nothing.
- **Sample.** 24 own tasks × 2 configurations × 2 repeats, and the 20-instance slice × 2 × 1, on Sonnet 5.5 and on Opus 5.5.
- **Descriptive rows**, beside the verdict and never in its place: one per own category; for the slice, the instances whose problem statement contains none of the patched files' repository paths and none of their basenames other than `__init__.py`.
- **Power, said first.** Twenty-four paired tasks cannot show a small solved difference; INCONCLUSIVE is a result, and LOSES is published like PAYS.

## Pre-registered for ADR-162 (Stage 5) — written with the record, before any code, pilot or run

Zy, 2026-10-17: "we need conclusive resutls, go for them"; then "One run, A/B/P", "Sonnet and Opus", "Looks every 100"; after the two challenges, "Verified 2+ hunks (191)", "Prune other projects' unused images", "Amend it before Stage 5". Registered in ADR-162's first commit:

- **The bar is ADR-156's** with two changes for this stage alone. A look reads 99% intervals (1 − 0.05/5, Bonferroni over five looks) in place of 95%. The solved interval is Tango's score interval for a paired difference, which does not collapse when no instance is discordant (the percentile bootstrap read [0, 0] there and could close PAYS on a true 5-point loss with probability 12.9% at 40 instances); cost keeps the percentile bootstrap. The control is nominal and per comparison, and the four verdicts are four claims, not one family at 5%. One verdict per model and comparison: B against A (mrw), P against A (the quality-harness plugin, the same rule with P in B's place).
- **The pool:** SWE-bench Verified at revision `78f471bf655a3137b2e8a75af1501690ec009ec3`, every instance whose gold patch has 2 or more hunk headers, except the 40 that `bench/swe-bench/selection.json` and `selection-adr-160.json` selected: 191, ordered by `sha256("adr-162:" + id)` (`bench swebench-select --salt adr-162`, which appends the colon). Descriptive row: the instances with 5 or more hunks (31).
- **Blocks and looks:** blocks 1 to 4 are instances 40(K−1)+1 to 40K, block 5 is 161 to 191; a look after each. A look is eligible for a model when every scheduled attempt has a receipt, exclusions are at most 5% per configuration, and at least 95% of the instances run are paired. A comparison closes at the first eligible look where it is PAYS or LOSES, and later attempts of its second configuration are not run or read; a model whose two comparisons are closed runs no more blocks; the stage ends after the fifth look or when no model runs on. An open comparison then is INCONCLUSIVE. Three consecutive start-error exclusions stop the run, resumable.
- **Configurations and the profile:** A as in Stages 3 to 4b; B, A's sibling with the pinned mrw; P, A with the quality-harness plugin loaded with `--plugin-dir` at its latest tag. The Stage 5 profile, in every configuration and every block: skills on, `Skill` offered, one real `node` on PATH, `CODEX_CLI_PATH` at a stub that refuses (no external model), no account-synced skill or plugin; `Agent`, `Task`, `WebFetch` and `WebSearch` offered to none. The canary checks each configuration's skills, plugins, hook sources and markers.
- **Every instruction ends with** the fixed `STATUS:` sentence ADR-162 T2 records. The receipt keeps `status`, `oracle.broke_tests` and `oracle.harness_resolved` (the harness's verdict, kept over the budget); ADR-156's `solved` is unchanged. Reported beside each verdict, never in its place: false completions, under-claims, broke-tests and unreadable claims per configuration.
- **Sample:** both models, three configurations, one repeat, each instance's attempts together with its image pulled once, in bench's seeded order; stage `stage5`, one directory per block; one mrw release, one plugin tag and one pinned CLI for the whole stage.
- **Contamination:** after each block every transcript is scanned for the dataset file, `swe-probe`, `huggingface`, the selection files, and the instance's FAIL_TO_PASS test names except those its own problem statement gives; a hit stops the stage for Zy, and no attempt is replaced, dropped or kept at the executor's discretion.
- **Machine:** `docker image prune -a` before block 1 (images no container uses); a 25 GiB free-disk floor before each pull; costly-runs before each block.
- **Quality-harness ADR-106** is amended in its repository to take these rows, and its pre-registration committed, before the first counted attempt.
- **Power, said first:** PAYS for B against A needs a mean cost saving of 10% or more, which Stage 3's no-floor population (about 6%) cannot reach and the 5+ hunk slice (17% to 21%) did; this pool lies between and is unmeasured. At 191 the 99% solved half-width is about 7 points. P's effect is unmeasured. INCONCLUSIVE is a result.

## From ADR-162

- **A plugin arm with `Agent` or external models allowed** (2026-10-17): ADR-162 keeps `Agent` out of every configuration (ADR-156) and stubs Codex, so the plugin's subagent and review skills run without them. Whether to measure P with them offered to A and P alike, their cost captured, is Zy's call after Stage 5's P against A is read.
- **Other datasets** (2026-10-17): if a comparison stays INCONCLUSIVE at 191 instances, SWE-rebench or SWE-bench Pro would give a larger multi-hunk pool; Zy's call then.

## Pre-registered for ADR-163 (Stage 6, for quality-harness ADR-112) — written with the record, before any code, pilot or run

Zy, 2026-10-18, approved measuring the plugin where its mechanism can work; 2026-10-19: "Draft now, run after". Registered in ADR-163's first commit:

- **What bench provides:** every configuration runs `python`, `python3`, `pytest` and `py.test` through shims inside a per-attempt container of the instance's own image, with no network and `/testbed/.git` replaced by a one-commit repository before the agent starts; one shim command at a time; the agent's change synced in before each command without touching its index; a sync that does not apply fails the command, never the attempt. Host interpreters reached by absolute path, `env -i`, `command -p` or `xcrun` are counted; a command naming `docker`, a pool image or the container stops the stage.
- **The declared check:** one whole runner per pool repository in `bench/swe-bench/checks.json`, with no path or subset arguments, from the repository's own files with a file:line@commit source, never from the dataset; written as `.quality-harness.json` into every arm's tree before the base commit. Its SHA-256, registered 2026-10-23 by ADR-163 T2 S2 before T4's baseline pilot: `7f169124fbdcdc8e5fa118614aeb275122fc7947fdd7034570141c3c93fb71a7` (each repository read at its newest pool base commit and checked at its oldest; a runner that named its suite root is recorded bare, and a difference at the oldest base is noted in the entry — Zy, 2026-10-23); an amendment after that reruns the pilot on unused instances.
- **The arms:** A, P, and W (P plus quality-harness's pre-registered `/quality-harness:work` line). The caps come from ADR-112 D3, set from W's pilot medians, never below 1200 s and 60 turns.
- **The instances:** after ADR-112 D1 passes (read against 1200 s) and before quality-harness pre-registers, one baseline run of the declared check per pool instance on its unmodified tree; the green ones in ADR-162's pool order are the surviving pool (`bench/swe-bench/selection-adr-163.json`), whose digest and count the pre-registration carries; blocks are its 40-instance slices, the last partial; red ones are counted, never named.
- **The stop rule is ADR-112's:** after each block, quality-harness's scorer at its pre-registered blob reads the rows per model and pair; its output is the look; bench runs the open arms next.
- **Pilots first, all after Stage 5 closes:** a baseline pilot with no model on 10 instances outside the pool, listed by rule before it runs, settles ADR-112 D1; then the whole-pool baseline; then, after quality-harness's pre-registration, an agent pilot of 24 attempts at 1200 s and 60 turns, which sets D3's caps from W's medians over both models.
- **Side arms G and C (amendment of 2026-10-21, Decision 9; registered with the amendment, before any side-arm attempt):** each is read against P alone, per model, by `bench side --control P`, paired by task. Cost uses `bench analyze`'s percentile bootstrap and solved uses Tango's score interval, both at 99%, cumulated over every block in which the arm ran. Neither is ever read by ADR-112's scorer, and neither stops or prolongs the stage. Both run at the stage's one frozen plugin tag, named here before T4's agent pilot with C's setting spelling and G's asset sha.
  - **C (`turnEnd: notify`, the continuation off) against P (as shipped, the continuation on):** "the quiet Stop costs less, with solving inside a 10-point non-inferiority margin" holds when the upper bound of cost (C-P)/P is below 0 and the lower bound of solved C-P is above -0.10. Turns and wall time are reported beside it.
  - **G (hooks run by the Go binary) against P (Node):** "Go hooks stay inside a +5% cost margin and a 10-point solving margin of Node" holds when the upper bound of cost (G-P)/P is below +0.05 and the lower bound of solved G-P is above -0.10. Turns and wall time are reported beside it.
  - Neither verdict is read before the arm has at least 60 paired tasks per model, at which the 99% Tango lower bound for all-concordant outcomes is -0.0996. The agent pilot never counts. Below 60 the rows are published as descriptive only.

## From ADR-163

- **Arms P+mrw and P with a batching cue** (2026-10-19): quality-harness ADR-107 T4 asks for them with turns per attempt as the target. ADR-112's stage measures A, P and W only; these need their own pre-registration and a stage after ADR-163's.
- **Hiding the host interpreters from the attempt** (2026-10-19): ADR-163 counts calls to host interpreters by absolute path, `env -i`, `command -p` or `xcrun`. If T4's agent pilot counts any, how to hide them before block 1 is decided with Zy then, and this entry closes.
- **The side reader applies rulings to P only, untested** (2026-10-23): `bench side` applies `rule` to the control's attempts (`internal/bench/side.go`, `if src.config == control {`), since a side arm's attempts live in `side/`, which no ruling names. No T6 test sets a ruling, so a mutant that drops the call or rules the side arm survives. Add a ruled-P fixture to `TestTheSideReaderUsesPAsControlAndTheRegisteredBar` before the first side verdict is read (T5).

## From ADR-175

- **Compaction hooks for Codex and Cursor** (2026-10-24): the `mrw seen --forget` hook is documented for Claude Code only (`docs/agents.md` §8); equivalents for other hosts are deferred (ADR-175: compaction hooks for Codex and Cursor). Arm when a Codex or Cursor user reports a post-compaction unlicensed write, or when either host documents a compaction event a command can hang on.


## Pre-registered for a whole-file re-read guard (`--if-changed sha=X`) — written before the measurement

The idea (memoryd tool-multipathreadwrite/40053273): a read that names the sha it last saw is answered
"unchanged, nothing served" when the whole file still has that sha. It is the only delta-read design the
ledger can carry safely: the ledger is shared by every caller of a checkout, has no session and stores no
text (`internal/seen/seen.go`), so mrw cannot elide a line on its own judgement (rejected: eliding from
the ledger) nor serve a line delta (rejected: needs the old text stored).

**Measure.** `docs/bench/context-study/scan.py` M2, extended: the share of served chars (`ext`) that fall
in one tool call's serve of one file whose every line is a within-epoch re-serve with no write to the file
between — the most such a guard could answer, since it ignores changes made outside the session and
assumes the caller held the sha.

**Bar.** An ADR for the guard is written only if that share is **at least 1.0% of served chars on at
least one host**. Below it on every host, this entry closes with the numbers and the idea is dropped.

**Measured 2026-10-23** (`scan.py` with the per-call grouping Codex asked for on #512 — the first count,
1.35% / 0.95% / 6.14%, grouped by a run of one path and overcounted a call that interleaved files —
two runs, identical counts), share of all served chars, with this run's within-epoch share beside it:

| | Claude Code | Codex | Cursor IDE |
|---|---|---|---|
| whole-file unchanged, within an epoch (the guard's ceiling) | **1.32%** (1,687,016 chars) | 0.86% (198,239) | **6.10%** (8,558,790) |
| within-epoch re-serves, any | 4.03% | 2.41% | 9.87% |

**The bar is met** on Claude Code and Cursor. Next: ADR-169, the guard, for Zy's acceptance. The ceiling
counts every host read, not only reads through mrw, and assumes the caller held the sha; both are the
ADR's to price. ADR-169 was accepted 2026-10-23; its adoption bar is the next entry.

## Pre-registered for ADR-169 (adoption) — written before the release that ships `--if-changed`

Measured once, 14 days after that release's tag, by `docs/bench/context-study/scan.py` at the commit
that wrote this entry (its `M2.ext.guard_*` counts), over the transcripts on this Mac whose sessions
start on or after the tag's date and within 14 days of it — Claude Code, Codex and Cursor IDE.

- **Numerator:** the sum of `B` over `unchanged since sha … — N lines, B bytes not served` lines in that
  cohort: what the guard withheld, read from mrw's own line.
- **Denominator:** the chars mrw served in the cohort — mrw output and `mrw_read` answers only, not the
  hosts' own reads — plus the numerator.
- **Cost, reported beside it and not netted:** the chars of the `--if-changed` / `if_changed` arguments
  typed.
- **Keep** the guard if the share is at least 1.0% on Claude Code or Cursor IDE.
- **Deprecate** it (ADR-169 Rollback: the taught lines go, the flag keeps working, removal waits for a
  major version) if the share is below 1.0% on both, each with at least 1,000,000 mrw-served chars.
- **Inconclusive** on a host with fewer: the window is extended once by 14 days; still inconclusive,
  the numbers go to Zy, who decides.

## Findings filed from memoryd (2026-10-21)

- **quality-harness cannot see writes made through mrw (filed 2026-10-02 by quality-harness, open).**
  Its PostToolUse hook appends a `file.written` event, with path and blob, only for its own Write/Edit
  tools. A `mrw write` run from Bash leaves no line, so quality-harness cannot reuse a passing check
  after docs-only edits made with mrw. Measured there 2026-10-02: 14 full checks, 54.5 min; 7 of 13
  re-runs followed mrw docs edits. Their BACKLOG §331 defers that reuse on this. Options they named:
  1. mrw appends one line per applied file to a caller-named JSONL (an env var such as `MRW_EVENT_LOG`);
  2. their hook reads mrw's stable `--json` receipt (ADR-111, docs/receipts.txt), which already exists.

  Open question: whether mrw should know any caller's log. Trigger: quality-harness picks up §331, or
  Zy asks.

## Closed, kept here because a done task's Acceptance fence reads this file

These entries are closed, but the Acceptance fences of done tasks (ADR-037, ADR-052, ADR-058, ADR-075, ADR-099, ADR-100, ADR-101, ADR-102, ADR-105, ADR-106) grep this file for them, and changing a fence would invalidate its recorded verification.

## From ADR-037 (the binary teaches the format it demands)

Classified 2026-09-09 from the review; this heading is the receipt so `adr-debt`
does not report UNRECEIPTED. Nothing here is newly invented — each line names
the older entry or record that still owns it.

## From ADR-051 (foreign plan grammars compile to plan hunks)

- **ast-grep-shaped `--grep`.** Shipped 2026-09-15 as ADR-058. Structural
  find only, and only if it does not become a write-time parser (ADR-048).
  READ path only. Flag `--ast-grep` (MCP `ast_grep`) beside `--grep` shells
  out to the `ast-grep` CLI if present, maps hits to line ranges, and serves
  through existing `read`. Missing binary: exit 2, names `ast-grep`. Apply
  does not change. License is still served lines, not AST nodes. `--grep`
  stays regex; the two may disagree on the same token — that is the feature.
  Rejected: tree-sitter inside apply; replacing regex `--grep`; bundling a
  per-language parser.

## From ADR-052 (echo pad is opt-in; a multi-line replace needs a served line after End)

Filed 2026-09-13 from a Zeus field report that read `internal/apply/apply.go:960`
before proposing, and that dropped two ideas after that read. ADR-052 already
refuses the wrap-tail miss (`end > start` and ledger does not cover `End+1`).
`--check`, `--echo-pad`, `sha=` and the ledger already exist. Nothing below is
new capability.

## From ADR-066 (a plan that cannot commit whole says what it wrote)

- **The ledger after a partial commit** — **Closed** by ADR-102 (`writer.Apply` records a partial commit's written files, 2026-09-30). Only an applied plan records what it
  wrote, so a file a partial commit DID write is refused as "changed since mrw
  last saw it" on the next edit. Loud, not silent; arm if a caller hits it.
- **A probe left behind** — **Closed** by ADR-105 (`left_behind` names it, 2026-09-30) (ADR-086, not observed): when a staging probe's removal fails, the empty
  file stays in the tree while the receipt says NOTHING WRITTEN; the refusal names it, but the
  receipt's written list does not. Say so on the receipt if it is ever observed.
- **A `.mrw-aside-*` left behind.** — **Closed** by ADR-105 (`left_behind` names it, 2026-09-30). When the final aside removal fails after a
  plan that applied, the placeholder stays in the tree (ADR-004 hygiene, not a
  false receipt). Say so on the receipt if it is ever observed. The same holds
  for a staged `.mrw-*` temp file whose removal fails after a staging abort
  (`discard` in `apply.go`). Since ADR-088 both discards are written `_ =` at
  their sites; the Codex review of #259 traced them, and reporting the survivor
  is this entry's work.
## From the v1.25.1 adversarial round (2026-09-25, peer sessions told to break it on purpose)

- **Two writers off one read can both exit 0 with one edit lost.** `scripts/chaos.py` already
  counts this as a known, accepted risk (race suite, "concurrent writes"); the Windows chaos runs
  measured it at 45–53% of racing writers over five full-scale corpora, and it reproduces on
  macOS. The acceptance was recorded before the rate was known: a decision for M.
  **Fixed by ADR-075**, contract §150: a per-checkout write lock is held from apply through the
  ledger update, with the ledger loaded before it under its own lock (`seen.Snapshot`, so never half-saved), so a writer whose file changed while it waited is
  refused, exit 1, "changed since mrw last saw it"; none exits 0 without its edit. `chaos.py`'s race
  suite fails on a lost edit by default. A writer that starts after another finished still writes on
  that writer's whole-file licence (ADR-002, kept by M the same day).
## From ADR-097 (a flag named for a subcommand names the subcommand)

- **`mrw read help` and `mrw read h` print `read`'s help** — **Closed** by ADR-099 T2 (`HideHelpCommand` on read, write and check). (ADR-097 Follow-up, observed 2026-09-29 against
  v1.31.0). urfave appends its `help` command (alias `h`) to every command, and it wins over a
  positional of that name, even after `--`: a file named `help` or `h` is reached only as `./help`. Arm
  on the first caller who names such a file, or with the next record that touches argument dispatch.
## From ADR-100 (every `mrw check --json` refusal is a document)

- **`exit_code` 0 beside `"ran": false` in `check --json`'s receipt** — **Closed** by ADR-101 (a check with no exit status reports -1 in both receipts, 2026-09-30). (ADR-100 Out of Scope). When no
  check could run (no harness, no `go.mod`) the receipt is `{"ran": false, "skipped": …, "exit_code": 0}`
  and mrw exits 2 (probed 2026-09-30). A consumer that reads `exit_code` alone reads a pass. Omitting the
  field when nothing ran changes the receipt shape ADR-054 and ADR-092 pin; arm on the first consumer that
  branches on `exit_code`.

## From ADR-102 (a write that landed is reported and counted as landed)

- **The cleanup errors mrw ignores, and what a failed run leaves behind** — **Closed** by ADR-105 (`left_behind`, atomic state writes, the README failure matrix, 2026-09-30) (ADR-102 Out of Scope; the 2026-09-30
  Codex design review, finding 5). `apply.go` `discard`, `stageFile`'s error paths and `pathop.go`'s aside removal
  drop a failed `os.Remove`, so a `.mrw-*` temp file or aside can remain unreported; state files are written in
  place. Planned as ADR-105 (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`): a `left_behind` receipt field,
  atomic state writes, a failure matrix.

## From ADR-103 (one checkout, one identity; and the filesystem root is a root)

- **Anchoring file operations to a handle** — **Closed** by ADR-106 (the write path through `os.Root`, and an identity recheck before each commit rename, 2026-09-30) (ADR-103 Out of Scope; the 2026-09-30 Codex design review, finding
  3). `rooted.Resolve` validates a path and returns it; read, stage, commit and path ops reopen it by name, so a
  link swapped in between can redirect them. Planned as ADR-106 (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`):
  the write path through an `os.Root` handed the resolved path, and a pre-commit identity recheck.

## From ADR-171

- **A one-line answer to a repeated `mrw instructions`** (2026-10-24): when the same state directory sees `mrw instructions` again, print one line instead of 12.9K characters. Deferred: it makes a read-only command depend on state, and ADR-171 Decision 2 removes the reason for the repeat (the header and the trigger say to read the rules once a session). Reopen if a re-measure after ADR-171 still shows repeated full reads.
