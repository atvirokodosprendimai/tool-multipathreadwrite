# Backlog — what is not yet an ADR

Deferred work and open questions, each naming the record that punted it. An
entry here is a receipt: a deferral whose pointer names this file but never
wrote anything into it passes every check and exists nowhere, so the entry is
written in the same commit as the deferral.

`adr-debt docs/adr` sweeps the deferrals in the records and expects to find them
here.

## From ADR-001 (a plan addresses the original file)

- **Streaming or memory-bounded application for very large files.**
  `internal/apply` reads whole files into memory and splits them into lines.
  Measured 2026-08-31 on a 14 MB / 200,000-line file: five hunks applied in
  0.15 s, a narrow ranged read in 0.03 s. Fine at that size, unbounded in
  principle. No decision needed until a real file makes it hurt — record the
  size that does.

- **A `cmd N` registry of saved commands addressable by number.** Proposed
  2026-08-31 alongside the `@N` file pointers, deliberately not built: a shell
  command invoked by number is unreadable at the call site, so a wrong number
  runs the wrong thing with nothing to inspect. The safe variant is saved
  *plans* (`mrw write @p1`), which stay inspectable artifacts. Undecided.

## From ADR-002 (mrw will not edit a file it has not seen)

- ~~**`mrw forget <path>`.**~~ **CLOSED 2026-09-03 — `seen.Forget` deleted.**
  It had no CLI caller and its doc comment described one ("Used by `mrw forget`
  when a caller knows their picture is stale") that never existed. Wiring it
  would have added a public subcommand, which needs an ADR; deleting removed
  dead code and a comment that read as evidence of a caller. `--force` remains
  the escape hatch for a stale picture. This was the second instance that day
  of "finished and unreachable" — `rooted.Descendable` was the first.

- ~~**Pruning ledger entries for deleted files.**~~ **MEASURED 2026-09-03 and
  DECLINED.** The entry said "harmless at current sizes; no measurement taken",
  so the measurement was taken:

  | | |
  |---|---|
  | this repo's ledger | 45 entries, 4,496 B, **1 stale** |
  | largest ledger on the authoring machine | 505 entries, **41 KB** |
  | every ledger on that machine, together | 13,819 entries, 935 KB |

  41 KB is not a problem, and the fix is not free. Sweeping inside `Record`
  couples a persistence function to the filesystem, and an implementation that
  stat'ed every entry broke two legitimate unit tests that use synthetic paths
  — `Record(obs)` followed by `Load()` no longer returned what was recorded.
  A narrower version (sweep only entries the caller did NOT just record) works
  and is written down here rather than shipped, because it buys 41 KB.

  **Reopen with a number.** If a ledger reaches a size where load or save is
  measurable, that is the trigger. Growth alone is not.

## From ADR-003 (a check's verdict comes from the process)

- ~~**Cleaning up the temp output files.**~~ **CLOSED 2026-09-03 — delete on
  success, keep on failure.** Measured before the fix on the authoring machine:
  **11,129 `mrw-check-*.log` files totalling 43 MB**, one per `--check` run ever
  made, none ever removed. The "harmless" reading was wrong and only a count
  showed it.

  Two conditions guard the delete, not one. A FAILING check keeps its log,
  because the tail is a summary and the file is the evidence. A truncated report
  keeps it even on success, because the report says "N earlier line(s) in
  <file>" and deleting a file the report points at is worse than leaving it.
  `Result.OutputFile` is cleared when the file is removed, so nothing ever names
  a path that is not there. Asserted by `scripts/contract.sh` §29, and by
  `TestAPassingCheckLeavesNoLogBehind` / `TestAFailingCheckKeepsItsLog`.

- **Scope derivation for languages other than Go.** `{packages}` is derived by
  mapping `.go` files to their directories; any non-Go path forces the full
  check. A Python or Rust project gets the full command every time, which is
  correct but slow.

- **`--check` under `--dry-run`: settled as exit 2, recorded here because the
  question is ADR-003's and the answer was reached in a PR about ADR-008.**
  Nothing is written under `--dry-run`, so no check can run. Two readings, both
  defensible:

  *Exit 0 with a warning* — nothing was written, so there is nothing to verify,
  and that is materially different from a configured check going missing. Rule 2
  is about "no evidence" where evidence was possible. The caller also still gets
  the plan validation they asked for. **This is what shipped first, and it is
  wrong.**

  *Exit 2* — the caller asked for verification and received none. Rule 2 says a
  check that did not run is not a pass, and ADR-003's own exit table files a
  missing check under `2 | usage, parse, missing check, bad pointer`. The flag
  combination is also a plain contradiction, which is what exit 2 is for. **This
  is what ships now.**

  Exit 2 wins because rule 2 is already decided and this is an instance of it,
  not a new question — applying an accepted rule is conformance. The reason it
  is written down anyway: the first version returned exit 0, and a reader who
  found that in the tree could reasonably conclude rule 2 had been abandoned.
  If the opposite reading is ever preferred, it is a change to ADR-003 and wants
  a record, because a caller scripting `write --dry-run --check` now gets a hard
  failure where they got success.

  A refusal also has to be POSITIONED, and the exit-0 version hid that: the
  first refusal sat after the plan was parsed and applied, so an unparseable
  plan preempted it while a plan whose HUNK failed lost to it. Both are "your
  plan is wrong" and they ranked differently only because of where the test sat
  — and exit 1, which promises an untouched tree, became exit 2. It is settled
  before the plan is read now, so a usage error preempts everything, which is
  what exit 2 means. Pinned by the precedence rows in `contract.sh`.

## From ADR-004 (mrw leaves nothing in the working tree)

- **Pruning orphaned state directories.** Moving or deleting a checkout leaves
  its `$XDG_STATE_HOME/mrw/<key>/` behind. Each is a few hundred bytes and
  carries a `root` file naming the checkout it belonged to, so a human can see
  what is dead:
  `grep -r . "${XDG_STATE_HOME:-$HOME/.local/state}/mrw"/*/root`. No automatic
  reaper — deciding a directory is dead means deciding a path will never come
  back, and a tool should not decide that.

- **Windows conventions.** The state path is XDG-shaped; Windows would want
  `%LOCALAPPDATA%`. Nothing currently builds or tests mrw on Windows beyond
  cross-compiling the binary in CI, so this is unexercised rather than broken.

## Not tied to a record

- **The `human-decisions` amendment on shell file-writers.** Drafted 2026-08-31,
  awaiting M's approval. The workspace rule bans ad-hoc shell file-writers
  because gates keyed to `Edit`/`Write` cannot see a heredoc; the proposed
  narrowing exempts a single named tool that emits a machine-readable receipt,
  on the grounds that a `Bash` hook can match its name and read what it wrote.
  Evidence: `wing_craft/tooling` drawer
  `16b01a4c4c8a4448839ca08100cd63ca8e0ef0281193d0c8b6ac1c5688bcf0f7`. Not an ADR
  because it is a workspace-wide convention, not a decision about this
  repository.

- **Go-level coverage of `cmd/mrw`'s CLI wiring.** `cmd/mrw` has only
  `version_test.go`; pointer resolution in a hunk path and the exit-status
  mapping are covered end-to-end by `scripts/contract.sh` instead. Noted in
  ADR-003-T2 as a stated limitation rather than an oversight, but a Go test that
  execs the built binary would close it.

## From ADR-007 (mrw finds the files it serves)

- **A cross-file `--max-lines` budget.** The cap is per SPEC today — `read.Run`
  resets `budget := opt.MaxLines` for each one — and ADR-007's walk deduplicates
  so that it is per file for everything the walk produces. What nobody has
  decided is whether `mrw read --grep PAT .` over a large tree should have a
  budget for the WHOLE answer rather than per file, which is the number an agent
  paying for context actually cares about. No measurement taken; the shape of
  the answer probably depends on what T2's cost measurement says.

- **Parallel walking or searching.** ADR-007's walk reads every candidate to
  match it and then `read.Run` reads the matching ones again, serially. That is
  the price of `Run` staying the only reader that observes (ADR-005). If T2's
  go/no-go cost measurement comes back close to the 2× threshold, parallelism is
  the first thing to reach for — and it needs its own answer to whether output
  order stays deterministic, which the receipt format assumes.

## From ADR-010 (mrw speaks MCP over the same engine)

- **A stateless hash-in-request mode, as `mcp-text-editor` uses.** The caller passes back the SHA it
  read, so the tool holds no ledger and has no concurrency limit on any transport. It is the
  cheapest fix for the parallel-read limitation and ADR-010 rejected it for one reason: it moves the
  read-before-write guarantee INTO the caller, and a caller that echoes a SHA it never read has
  licensed itself. ADR-002 exists so the tool holds that fact. **This is the fallback if the server
  path fails its go/no-go** — recorded here so it is not re-derived from scratch.

- **Exposing `check`, `iter`, `seen` and `stats` as MCP tools.** ADR-010 ships two tools because
  read and write are the product. Each of the others is a separate decision with its own answer to
  "what did the caller see": `check` runs a subprocess, `seen` exposes the ledger, `stats` exposes
  the tally. None is obviously wrong; none is free.

- **The CLI path's parallel-read limitation.** ADR-010 lifts it for MCP callers for free — one
  server is one writer — and deliberately does not touch the CLI path, where parallel PROCESSES
  race on a whole-file ledger rewrite. Measured: 40 racing reads kept 5. Fixing it there means
  per-entry locking or an append-only ledger, which is a format change ADR-002 governs.

- **Publishing to an MCP registry or directory.** Deferred from ADR-010-T3. The config block in the
  README is the install path; a registry listing is distribution work with its own review surface.

## From ADR-009 (mrw counts what happens to the plans it is given)

- **A live-model benchmark: ask a model for a plan, grade whether it parses.** The direct answer to
  "can a model author this format", and deferred as the second move rather than the first. It needs
  an API key, costs money per run, cannot run in CI, and measures the model available on the day
  rather than the format. ADR-009's tally answers the same question from production for nothing;
  build this when the tally says WHICH parse failures dominate, so the benchmark knows what to probe.

- **A fixture corpus of recorded model-authored plans, graded hermetically.** Better than a live
  benchmark — repeatable, no key, runs in CI — and blocked on the same thing: somebody has to
  collect the corpus, and the honest source is the production signal ADR-009 adds. The tally tells
  us what the corpus should contain.

- **Shortening the per-hunk receipt for large plans.** SWE-agent's ACI work (arXiv:2405.15793) states
  "feedback should be informative yet concise to respect context limitations", and mrw prints one
  `ok`/`FAIL` line per hunk plus a summary — about 30 lines for a 27-hunk plan, on a tool whose whole
  pitch is context economy. Not obviously wrong: the per-hunk verdict IS the product, and collapsing
  it would be the silent-success shape this project refuses. What is missing is a measurement of what
  the receipt costs at scale before anyone trims it. Undecided, and ADR-009 explicitly does not
  decide it.

- **A cross-model comparison of authoring success.** Which models can emit the plan format and which
  cannot. Deferred from ADR-009-T3: one repository's tally is one population, and the parent record
  says so — a comparison needs several, which needs the fixture corpus above.

- **Changing the plan format in response to the reading.** ADR-009 pre-registers the criterion
  (parse refusals over 5% means the format is the problem, not the caller) and deliberately stops
  there. What to CHANGE is a separate decision, and making it before the number exists would be the
  formality the criterion was written to avoid.

## From ADR-008 (a delete says what it removed)

- **Requiring a guard on every `delete`.** ADR-008 makes a delete REPORT its
  bounds and lets it DECLARE an expected body, but leaves an unguarded
  `@@ f.go 5-8 delete` legal. Requiring `lines=` or `anchor=` was the runner-up
  and is genuinely close: it taxes every correct two-line delete to catch the
  rare wrong one. Revisit once the receipt bounds have been in use — if a wrong
  range still reaches a build after ADR-008, the tax is worth paying.

- **Requiring the expected body rather than accepting it.** The body is opt-in
  because it costs the caller the tokens of the lines being removed, which is
  worth it for a delete worth pinning and not for a two-line one. If plans in
  practice omit it exactly where it would have helped, that judgement is wrong
  and the default should move.

- **`anchor=` reports its failure above the ledger check, so a failed anchor
  reads one line of a range the caller was never served.** ADR-008 moved its own
  expected-removal comparison BELOW `covered()` for exactly this reason and
  pinned it with a test; its sibling one line up was noticed at the same time
  and deliberately left, so this entry exists to stop the asymmetry reading as
  an oversight. Reproduced 2026-09-01, `internal/apply/apply.go:453-456` — with
  `f.txt` served at line 3 only:

      FAIL f.txt 1 replace (plan line 1): anchor "nope" not in line 1: SECRET-LINE-ALPHA

  It fires on `replace` and `delete` alike, and is repeatable per address, so it
  is bounded by `clip`'s 60 characters per failed hunk rather than by one line
  in total. Two corrections to the reasoning that first justified leaving it,
  both from the PR #11 re-review. The gloss "the anchor is the caller's own
  text" does not hold for the half that matters: the anchor is theirs, the
  PRINTED LINE is the file's, and that is the part never served. In the other
  direction the blast radius is smaller than it looks — `--root` still refuses
  first, so nothing outside the tree mrw was pointed at is reachable. So this is
  a contract violation against ADR-002 and ADR-005's "mrw does not tell you what
  it has not shown you", not a privilege boundary: the caller could read the
  file directly. Fix by moving the anchor check below `covered()`, which is a
  one-line move plus the fixture that proves it — a fixture whose FIRST version
  must be checked against a mutant, because the obvious one trips the
  whole-file gate instead and passes with the ordering reversed.

## Found by probing the built binary (2026-09-01)

Five were recorded here rather than fixed in passing, because each makes
something currently legal illegal. **Three are now closed — 2026-09-03 — and
are struck through below**; the two that remain are genuine either-way
decisions rather than defects. The bugs found in the same pass (an unknown
subcommand exiting 3, an absolute path reported as "does not exist", `sha=`
accepting non-hex, `$-1` reported as out of range, `--check` silently dropped
under `--dry-run`) were fixed then and carry contract rows.
- ~~**A DUPLICATED guard key silently discards the earlier one.**~~ **CLOSED
  2026-09-03 — refused at parse time.** It was the codebase's own principle
  turned inside out: `internal/apply/apply.go` says "a guard that is parsed and
  then discarded would be worse than no guard at all — the caller believes the
  edit is pinned", and the parser did exactly that. Re-probed 2026-09-03 and
  still true, so it was fixed rather than re-recorded: `anchor="NOPE"
  anchor="a"` applied at exit 0 with the false guard gone.

  **Refused, not resolved.** Two guards on one hunk are two different claims
  about one edit; picking either silently is how the caller keeps believing the
  other. Holds for `sha=`, `lines=`, `anchor=`, `body=` and `raw=`. It is a
  behaviour change — a plan that applied yesterday now fails — and the plan in
  question is one carrying a guard that was never checked. Asserted by
  `scripts/contract.sh` §31 and `TestARepeatedGuardKeyIsRefused`;
  `TestDistinctGuardKeysStillParse` pins that one of each is still fine.
- **`create` with an EMPTY body succeeds and reports `ok`.** ADR-006 refuses an
  empty-bodied `replace` because a body lost in transit — a truncated emission,
  an editor eating the last line — deletes code while the receipt says it
  worked. The same truncation on a `create` at the end of a plan produces an
  empty file and exits 0. The counter-argument is real: an empty file is a
  legitimate thing to create, and `touch` is not a mistake. So this is a genuine
  decision rather than an oversight, and it wants ADR-006's reasoning applied to
  it explicitly — probably `create` with no body being legal only when written
  as such deliberately, which the format has no way to say today.

- ~~**An ABSOLUTE path is silently reinterpreted as root-relative, and every
  surface prints the original.**~~ **CLOSED 2026-09-03 — fixed on every
  surface, verified by probe.** The write path gained the diagnosis in #36/#37;
  the READ path was closed by `rooted.IsRooted` (#38), which also fixed the
  Windows case where `filepath.IsAbs` is false for `/etc/hosts`.

  Verified 2026-09-03 against the built binary: `mrw read /etc/hosts` from a
  temp root now exits 1 with `==> /etc/hosts  REFUSED  /etc/hosts is outside
  the root <root>: read it with --root pointed where you mean`. The receipt no
  longer names a path the tool did not touch, and nothing outside the root was
  ever reachable. Asserted by `scripts/contract.sh` §26 and §27.
- ~~**`raw=true` without `body=` is accepted and does nothing.**~~ **CLOSED
  2026-09-03 — a usage error.** `raw=` only switches off the valid-header check
  INSIDE a counted body, so without `body=` it is a guard that cannot fire —
  the same class as the duplicate key above and fixed in the same change. The
  legitimate pairing (`body=N raw=true`, the escape hatch for a plan whose body
  contains a real `@@` header) is pinned by its own contract row, because
  refusing the useless form must not break the useful one.
- **`--max-lines 0` means UNLIMITED, and this repository decided the opposite
  question the other way once already.** A cap of zero is currently
  indistinguishable from no cap, so there is no way to say "serve me the header
  and nothing else" — and nothing is reported as withheld, though the README
  promises "whatever is withheld is always reported". The precedent cuts
  against the current behaviour: `body=0` in a plan means an EMPTY body, not an
  unbounded one, and `TestBodyZeroMeansAnEmptyBody` exists because treating an
  exhausted count as "keep scanning" silently handed a hunk lines the caller
  wrote for something else. `lines=0` is likewise a real assertion, not a
  disabled one. Against changing it: 0-means-unlimited is a widespread CLI
  convention, and `--stat` already serves the "header only" need. Either way it
  is a behaviour change, which is why it is here — negative values, which were
  silently ignored, are now a usage error and shipped with this round.

### Probed and found correct (2026-09-01, round two)

Recorded so the next probing round starts somewhere new rather than
re-measuring these. Each was driven at the built binary, not read:

- **A WHOLLY unparseable ledger fails closed.** Garbage written over the entire
  `seen` file leaves every write refused with "has not been read", including
  for files that genuinely were read. **Corrected 2026-09-01 after a Codex
  review**: the first version of this entry said "a corrupt or truncated
  ledger", which the measurement does not support. `internal/seen/seen.go`
  parses line by line and silently SKIPS malformed lines while keeping the
  valid ones, so a ledger holding a good record for `a.go` plus corrupt data
  still authorizes `a.go`. The fixture happened to corrupt every line. Whether
  a partially unparseable ledger should be an error rather than a silent
  partial read is an open question, and this entry is the only place it is
  written down.

- **Two concurrent writes to one file were not observed to both land, and that
  is weaker than a guarantee.** Racing two writes, the second was refused with
  "changed since mrw last saw it (recorded …, now …)". **Corrected 2026-09-01
  after a Codex review**: the first version claimed they "cannot both land",
  which the code does not promise. `internal/apply/apply.go` reads and
  validates the prior sha, then writes later, and `writeFile` renames
  atomically without a lock or compare-and-swap — so two processes can validate
  against the same original and both rename, last writer winning. The sha guard
  makes the race UNLIKELY and loud when it loses, not impossible. Closing it
  needs a lock plus an under-lock sha recheck.

  **Observed 2026-09-02, and one clause above is now false.** The race is no
  longer theoretical: 20 concurrent `mrw write` invocations against one 100-line
  file, each replacing a different line, three trials. Surviving edits varied
  1, 17 and 20 of 20 — so the loss reproduces easily on this machine. What the
  clause above gets wrong is "loud when it loses". It is not loud. A writer that
  lost printed a full success receipt and exited 0:

      ok   f.txt 24 replace  -1 +1
      wrote f.txt  100L -> 100L  sha 01a5e658
      1 hunk(s), 1 file(s), 0 failed — applied
      exit=0
      (W-8 absent from f.txt)

  The mechanism is last-writer-wins, confirmed by sha rather than inferred: each
  writer prints the sha of the file it wrote, and only the final writer's sha
  matches the file afterwards (writer 20 printed `b5db03f3`, which is what `mrw
  read` then reports). A losing writer's rename DID land and was superseded by a
  later one. The sha guard is racy, not absent — in the same run 6 of 20 were
  refused loudly with an explicit mismatch. So the accurate statement is: the
  guard makes the race unlikely, and SOMETIMES loud; when it loses it can lose
  silently, with a receipt that says applied. Locking stays permanently out of
  scope (ADR-002); this only sharpens the risk that scope accepts.
  line that only exists after the earlier writes resolves correctly.
- Pattern addresses, `/start/,/end/` pairs, pointer resolution (`@0`, `@-1`,
  `@abc`, `@N` out of range), overlapping and descending range lists, filenames
  with spaces, unicode and a leading dash, and a missing `$HOME` with no
  `XDG_STATE_HOME` — all correct, all with the right exit status.

## From a Codex review of PR #13 (2026-09-01)

- **`{packages}` and `{files}` are substituted into `sh -c` WITHOUT quoting.**
  `internal/check/check.go`'s `command()` builds the scoped command with a
  plain `strings.NewReplacer`, so every character of a derived path reaches the
  shell as syntax. A path containing a space splits into two arguments; one
  containing `;`, `$(…)` or a glob is executed. The reviewer's case: a file
  named `pkg; true #/x.go` with `scoped_check: "go test {packages}"` produces
  `go test ./pkg; true #` — the package is never tested, the shell exits 0, and
  both `mrw check` and `write --check` report PASS at exit 0. **This is the
  silent-PASS class again, one layer down**, and it is not hypothetical for
  spaces alone. It is HIGH and it is pre-existing, so it gets its own record
  and its own change rather than riding along in the PR that found it. The fix
  is to shell-quote each substituted token before joining, plus contract rows
  driving the real binary with spaces, semicolons, `$()` and glob characters.

- **A declared check can be inert, and mrw cannot tell.** `"check": "true"`,
  `":"`, `"# comment"`, `"$UNSET"`, `"VERIFY="`, or `"exit 7 | true"` all make
  `sh -c` return 0 without verifying anything, so `Ran` is true, `OK()` is
  true, and the verdict is PASS. The reviewer offered this as a gap in ADR-003
  rule 2; it is recorded here rather than fixed, because **it is a different
  question from the one that rule answers.** Rule 2 forbids mrw from inventing
  a pass for a check that did not run. It cannot make mrw judge whether a
  command the project declared does useful work — `make verify` may be empty
  too, and `internal/check/check_test.go::TestRunPasses` deliberately requires
  `true` to be OK. The whitespace case was a defect because nobody DECLARES
  `"   "`; it is an accident being read as a declaration. `"true"` is a
  declaration. What would change this is an enforceable check protocol — a
  command required to emit a signed or structured result — which is a much
  larger decision than a trim, and nothing here needs it yet.

## From the `$` divergence (2026-09-02, no ADR)

- **One address parser, not two.** `internal/plan.ParseAddr` and the range
  parser inside `internal/read` both read the same address grammar and drifted:
  `$` meant the last line to one and "unbounded" to the other, so `mrw read
  f.go:$` served the whole file while `@@ f.go $ replace` changed one line.
  Fixed inside `read` rather than by unifying them, because read's grammar is
  strictly richer — `/re/`, `/re/,/re2/`, comma-separated spans — and folding
  that into `ParseAddr` grows a second grammar inside it.

  This is ADR-006 rule 2's argument ("the boundary lives in one place;
  duplicating it would have re-created the divergence in a slower form") applied
  to addresses instead of the root, and the divergence it predicts has now
  happened once. Worth a record if it happens twice, or if read's grammar stops
  being the larger one. No decision needed yet.

- **A read over MCP buffered what the CLI streams — TAKEN, and the reason this
  entry gave for deferring it was wrong.** Superseded by ADR-011-T3.

  The measurement stands: an 18 MB file cost 44 MB via the CLI and 169 MB over
  MCP; ten of them cost 5.9 MB and 1268 MB. What was wrong was the conclusion.
  This entry said a cap "is a behaviour divergence from the CLI, which
  ADR-010's whole thesis is careful about", and left the decision open on that
  ground. Claude Code caps MCP tool output at 25,000 tokens by default and
  offers a per-tool override up to 500,000 characters; an oversized result is
  "persisted to disk and replaced with a file reference in the conversation",
  so the model never receives it as the file either way. The choice was never
  cap-versus-fidelity; it was refuse legibly, or pay the memory to build a
  result the host then takes out of the conversation.

  Worth keeping as a lesson about deferrals: the reason recorded here was
  plausible, internally consistent and unchecked, and it deferred the work for
  as long as nobody read the host's documentation. A deferral's REASON deserves
  the same scepticism as a claim in a record, because it is one.

- **The copy amplification itself.** ADR-011-T3 bounds the RESULT, which fixed
  the memory (2617 MB to 87 MB on the 40-file case, measured 2026-09-03), but
  a read still renders into a buffer, is marshalled into the result and
  marshalled again into the response. For anything under the limit that is a
  few hundred kilobytes and not worth the complexity of a streaming encoder.
  Revisit only if the limit rises far enough for the constant factor to matter.

- **Splitting the authoring tally by call source, CLI versus MCP — DEFERRED,
  with the reason written down so it is not re-proposed every month.** Raised
  and rejected in ADR-012.

  `cmd/mrw` and `internal/mcp` both call `authoring.Record` into one set of
  counters, so `mrw stats` cannot tell a plan authored by a shell user from one
  authored by a host. Splitting them is easy and it does not buy the thing it
  looks like it buys. The population worth measuring is a caller who has ONLY
  the tool description — no AGENTS.md, no README — and every caller in this
  checkout has read AGENTS.md. Splitting partitions the population we have; it
  does not create the one we want, and ADR-009's Out of Scope permanently
  refuses the transmission that would bring the other one's outcomes here.

  Revisit only if a source split answers a question someone actually has about
  THIS checkout — for instance whether MCP callers hit `refused_apply` more
  than shell callers, which is about the ledger and not about the format.

- **Measuring whether served size degrades edit accuracy — DEFERRED, and it is
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
- **`occurrence=N`, or any positional disambiguator for a pattern address —
  DEFERRED.** Raised and refused in ADR-013.

  When a start pattern matches several lines the hunk is refused and the
  refusal names the matched line numbers. The obvious next step is to let the
  caller say *which* one — `occurrence=2`. It is deferred rather than taken
  because it makes a plan depend on the ORDER of matches in a file, which
  changes when unrelated code moves above them: the address would then resolve
  somewhere else while still looking correct, which is the silent-wrong-edit
  class the exactly-once rule exists to refuse. The caller already has the line
  numbers from the refusal and can address by number.

  Revisit only if the ambiguity refusal proves common enough in practice to be
  friction rather than a guard — and note that nothing currently counts it, for
  the attribution reason ADR-012's Context sets out.

- **Widening the MCP root, or multi-root — DEFERRED, with a measurement behind
  it.** Named in ADR-016's Out of Scope; the analysis was done 2026-09-04 and is
  recorded here so the decision is not re-argued from scratch.

  The proposal that prompted it was sound in shape: repeatable `--root`,
  ALLOW-ONLY (a deny list needs symlink resolution and reintroduces every
  path-handling bug already fixed), resolved once at startup, boundary set by
  the launcher rather than by a file mrw can read and therefore widen, and one
  state namespace per root.

  ⚠ TWO THINGS CHECKED AGAINST THE CODE. `state.Dir(root)` already hashes the
  resolved absolute root, so N namespaces come free. But `apply.Apply(root, …)`
  takes ONE root and `seen.Load(root)` loads ONE ledger per run, so a cross-repo
  plan needs the ledger resolved PER HUNK. That is not a small generalisation of
  the existing invariant; it changes what a run is.

  THE MEASUREMENT, which is what actually decided it: the test proposed by the
  argument itself is "count how many recent plans would have wanted a second
  root". Over one full session of ~40 plans, the answer was ZERO. The single
  file touched outside the root was a one-file config edit — below mrw's own
  trigger threshold, so multi-root would not have earned it even then.

  So N registrations wins on this evidence, and issue #75 records
  one-registration-per-repo as the Claude Desktop shape. Revisit only for the
  case that would justify it: routinely changing a contract in one repo and its
  consumer in another, where the value is an ATOMIC plan across both. Note the
  cost if it is ever taken — an all-or-nothing plan spanning two git trees rolls
  both back on one failed hunk, which is the right semantics, but neither tree's
  history reflects the other.

- **⚠ THE MULTI-ROOT MEASUREMENT ABOVE WAS TAKEN ON THE WRONG POPULATION.**
  Correction filed 2026-09-04, hours after the entry it corrects.

  That entry concluded "N registrations wins" from counting plans in one
  session: ~40, all single-root, so a second root would have earned nothing. The
  count is accurate and the population is a CODER working inside one git
  checkout — which is the population that already has the CLI and can point it
  anywhere with `--root`.

  M has since named the population that matters for this question: a Claude
  Desktop user, an analyst reading and writing large documents into CSV. Their
  files are not in one repository. For them, "one fixed checkout chosen at
  startup" is not a safety property they trade away — it is the thing that stops
  the tool working at all, and they have no CLI to fall back to.

  So the earlier conclusion stands ONLY for coders with a shell. It is not
  evidence about Desktop, and it should not be quoted as if it were. The measure
  worth taking is on the Desktop population, and nothing has taken it.

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
  **018 is the root guard, and REACH IS ADR-019**, still unwritten. The
  deferrals point at this entry rather than at a number precisely so a
  renumbering cannot break them — which is the reason `adr-debt` refused the
  original `ADR-018` pointer as a pointer to nowhere.

- **A heredoc-style body terminator for the plan format — DEFERRED.** Raised and
  refused in ADR-015.

  A body line beginning with `@@` needs `body=<n> raw=true`, and the friction is
  COUNTING the lines. `body=<<END … END` would remove the count. It is refused
  for now rather than taken because it is a genuine grammar addition that gives
  the format a second way to say one thing, and it does not remove `body=`
  anyway — a terminator can itself appear in a body.

  The judgement behind the deferral: what hurt was not counting lines, it was
  not being TOLD to. ADR-015 makes the refusal name the escape, so the evidence
  for a terminator is now "the hint landed and counting is still the friction".
  Revisit with that evidence, not before.

- **Two `create` ops that collide on a case-insensitive filesystem — DEFERRED by
  ADR-021.** `New.txt` and `new.txt` created in one plan have no inode to compare
  until one is written, so the identity check that refuses two spellings of an
  EXISTING file (`os.SameFile` at grouping time, ADR-021) cannot see them, and
  the second rename would win as before. Promote to a record when one such plan
  is seen in the wild; the fix needs either a case-fold belief about the
  filesystem, which issue #47 and ADR-021 both refused, or write-then-stat.
  ⚠ Beside it: two `create` ops for the SAME path in one plan are accepted today and produce a
  two-line file with both hunks `ok` (review of #89, 2026-09-04). Same spelling, so outside
  ADR-021's identity check — but it is a plan that "says two things" in exactly Decision 2's
  sense, and it belongs to whichever record takes the create collision.

- **The rules hook on Windows — DEFERRED by ADR-022.** `.claude/hooks/rules-on-read.py`
  is exercised by contract §55 on Linux and by its Go Enforced-by wherever `python3` is
  on PATH; the Windows runner may lack `python3` (the test skips there), and the hook's
  path handling (`os.sep`, `O_NOFOLLOW` absent, `realpath` on drive letters) has not been
  run on NTFS. Promote when a Windows contributor reports a delivery that did not happen,
  or when CI gains a Windows python3.

## From ADR-020-T2 (a target the instruction does not name)

- **Run a reading against the relational fixture.** T2 builds the selector; it does not spend the
  trials. The criterion is the one already pre-registered above and must not be re-derived — correct
  address rate against served bytes, stratified by position, refusals reported separately, a flat
  curve accepted as an answer. What the first reading adds is that the NAMED fixture is at ceiling
  (42/45, `docs/curve/reading-02-result.md`), so a relational reading is the first one whose curve
  has room to bend. It is a budget decision, which is why ADR-020 keeps readings out of its tasks.
- **A served window that does not begin at line 1.** All three misses of the first reading named the
  line exactly two below the target, and the reading cannot say why: every cell serves `@@ 1-N`, so a
  row count in the served rendering and the target's line number plus two are the same integer in all
  45 trials. A cell served from, say, line 500 separates them by 498 and settles it in one trial.
  Deliberately NOT in T2: it answers a different question from "can the task be failed", and folding
  it in would put two claims under one fence. Promote it when a reading needs to explain a miss
  rather than count one.

  **RECEIPT, 2026-09-05 — promoted and spent.** ADR-020 T4 built the cell (`-from`); reading 5
  (`docs/curve/reading-05-result.md`) served 15 trials from line 120 and every miss sat at
  `target − 117`: the row count. Whose count — the read arm's file reader's, which the transcript
  suggests, or mrw's own rows — was settled by reading 8 (`docs/curve/reading-08-result.md`): the read arm's. See the ADR-020-T4 entry below.

## From ADR-020-T4 (a served window that does not begin at line one)

- **The read format, if the row-count account holds — CLOSED 2026-09-05, no engine change.** Every
  miss in 135 read-arm trials sat at `target+2`, and mrw's served rendering opens with two rows that
  carry no line number, so the candidate was that the tool's own output induces the miss. Reading 5
  (`docs/curve/reading-05-result.md`) confirmed the row-count account — every miss is the row index
  of the served text, −117 with the window from line 120. Reading 8
  (`docs/curve/reading-08-result.md`) delivered the same client the same cells as a Bash tool result,
  mrw's gutter the only gutter, and it scored 15 of 15 with no plan at the row index: the count was
  the read arm's file reader's, not mrw's. No served-path record, no change to the header rows.
  What remains documented, not fixed: a client that saves mrw's output to a file and reads it back
  through a numbering viewer recreates the collision.

## From ADR-023 (a read's answer is the served text)

- **ADR-023: other hosts.** The host measurement behind ADR-023 is Claude Code 2.1.261 only: a
  successful result with `structuredContent` reached the model as the structured half alone. Claude
  Desktop and other hosts were not measured; the two host issues (#55677, #15412) say Claude.ai and
  ChatGPT render both halves. Worth one probe each when a Desktop or third-party session is at hand,
  recorded beside ADR-023's Verification Log. Not blocking: the bare envelope is right for a host that
  shows either half.
