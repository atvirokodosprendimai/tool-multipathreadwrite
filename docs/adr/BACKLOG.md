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

- **`mrw forget <path>`.** `seen.Forget` is implemented and tested but has no
  CLI caller — dead code until wired. Either wire it or delete it; leaving an
  unreachable exported function is the "finished and unreachable" shape this
  workspace keeps finding.

- **Pruning ledger entries for deleted files.** `.mrw/seen` grows by one short
  line per path read and nothing removes entries for files that no longer
  exist. Harmless at current sizes; no measurement taken.

## From ADR-003 (a check's verdict comes from the process)

- **Cleaning up the temp output files.** Each `--check` run leaves a
  `mrw-check-*.log` in the system temp directory. It is named in the report so
  the caller can read it, and nothing deletes it. Options: delete on success and
  keep on failure, or move them under `.mrw/`.

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

Four of these are behaviour changes rather than bugs — each makes something
currently legal illegal — so they are recorded rather than fixed in passing. The
bugs found in the same pass (an unknown subcommand exiting 3, an absolute path
reported as "does not exist", `sha=` accepting non-hex, `$-1` reported as out of
range, `--check` silently dropped under `--dry-run`) were fixed and carry
contract rows.

- **A DUPLICATED guard key silently discards the earlier one.** This is the
  most serious of the four, because it is the codebase's own stated principle
  turned inside out — `internal/apply/apply.go` says "a guard that is parsed and
  then discarded is worse than no guard, because the caller believes the edit is
  pinned", and the parser does exactly that. Reproduced: a hunk written
  `anchor="NOPE" anchor="a"` applies with exit 0, the false guard gone. It holds
  for `sha=`, `lines=`, `body=` and `raw=` too — last occurrence wins, silently.
  Fix is a parse error on a repeated key. It is a behaviour change because a
  plan that applies today would stop applying, which is why it is here and not
  in that commit.

- **`create` with an EMPTY body succeeds and reports `ok`.** ADR-006 refuses an
  empty-bodied `replace` because a body lost in transit — a truncated emission,
  an editor eating the last line — deletes code while the receipt says it
  worked. The same truncation on a `create` at the end of a plan produces an
  empty file and exits 0. The counter-argument is real: an empty file is a
  legitimate thing to create, and `touch` is not a mistake. So this is a genuine
  decision rather than an oversight, and it wants ADR-006's reasoning applied to
  it explicitly — probably `create` with no body being legal only when written
  as such deliberately, which the format has no way to say today.

- **An ABSOLUTE path is silently reinterpreted as root-relative, and every
  surface prints the original.** `mrw -C /root read /etc/hosts` serves
  `/root/etc/hosts` under the header `==> /etc/hosts`. The containment is
  correct and was verified — nothing outside the root is reachable — but the
  receipt names a path the tool did not touch, so a reader (or a hook parsing
  the output) concludes the system file was read. The write path now says the
  path was resolved against the root when the target is missing; the READ path,
  and a write that happens to find a file there, still display the original.
  Refusing an absolute path outright is the clean answer — joining one is never
  useful, since `-C /repo read /repo/f.go` already resolves to `/repo/repo/f.go`
  — but it is a behaviour change.

- **`raw=true` without `body=` is accepted and does nothing.** `raw=` exists to
  switch off the header check inside a COUNTED body, so without `body=` it is a
  guard the caller wrote that cannot fire. Harmless, and the same class as the
  duplicate key: a caller believing something is pinned when nothing is.

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

