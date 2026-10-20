# The drill: how a change reaches `main`

Substantive development starts with `/quality-harness:work` in the main session; the plugin owns
the lifecycle mechanics and the templates. This file is what THIS repository requires at each step.
It collects failures met here, and each section names the one it came from.

**Which of this applies.** §§2-6 are for a change a record owns. A bounded change — no new public
contract, exit code, persistent state or trust boundary (`CONTRIBUTING.md`, "What a change needs") —
needs a red test, the gates in `AGENTS.md` §Build and check, and a pull request.

## 1. A durable decision starts as a record

`CONTRIBUTING.md` says which changes need a record. Here one starts as `docs/adr/ADR-NNN-<slug>.md`
with `docs/adr/ADR-NNN/tasks/README.md` and one `docs/adr/ADR-NNN/tasks/T<n>-<slug>.md` per task: the
folder carries the number only, so a long slug does not spend the 120-character path budget
(`CONTRIBUTING.md`, Prerequisites). Records before 2026-10-20 keep their `ADR-NNN-<slug>/` folders.

**Reserve the number by reading, not by listing.** A number can be reserved in prose before its
directory exists — ADR-019 was reserved for reach while ADR-018 shipped — so `grep -rn 'ADR-0NN'
docs/` before taking one, not just `ls docs/adr`.

## 2. Every fence clause returns zero hits before the fence is written

The Acceptance fence greps for test names, a contract section `# NN.`, and new Go identifiers. Run
each of those greps BEFORE writing the fence and confirm every one is empty; a clause that already
matches is a clause that cannot go red.

**Contract sections are not in file order.** The next free number is
`grep -oE '^# [0-9]+\. ' scripts/contract.sh | sort -k2,2n | tail -1` — keyed on the second field,
because `sort -n` on a line that begins with `#` reads every key as zero, falls back to lexical order
and answers `# 9.`; and with the trailing SPACE, because without it the pattern also matches a comment
that happens to begin with a decimal — `# 1.6 s here` matched as a second `# 1.` and was filed as
issue #91 before anyone read the line. Run the recipe before committing it: the first version of
this rule did not, and shipped the `# 9.` form; the second shipped without the space. Taking the
file's tail once said 49 when the highest was 50.

`adr-lint <record.md>` must print `PASS` before any code. Its advice lines are findings too.

## 3. Red first, then the code

The record's `Enforced-by` test is written and FAILS before the implementation exists. Then the
implementation, then the rest.

## 4. Every `[proof: mutation]` step gets a killed mutant, logged

Break the step, watch the Acceptance fence go red, restore it, and log the mutant in the task's
Mutation Log. Let `adr-verify <task.md>` write the Verification Log line and its `acceptance-sha256`:
the digest is an ordinary sha256 of the normalised fence — anyone can reproduce it, and `adr-lint`
recomputes it to detect fence drift — so the point is a tool-written row, not a secret.

A mutant that SURVIVES means, first, that the fixture never reached the branch — not that the
mutation was weak. See `testing.md`.

## 5. A contract row drives the built binary

A unit test proves the function; it cannot prove the function is CALLED. Every promise gets a
`scripts/contract.sh` row that drives `$MRW` — the built binary — and pairs the good case with the
case that must fail. See `contract.md`.

## 6. Receipts, then gates, then commit

- `adr-verify <task.md>` exit 0 writes the Verification Log line. `adr-debt` must report
  0 broken and 0 unreceipted before a PR opens.
- The gates are the ones in `AGENTS.md` §Build and check, each run stand-alone, never through a pipe.
- **Run `./scripts/contract.sh` before committing, not after.** §50 went red on a commit whose author
  had run only the Go tests.
- **Engine changes are owned, not frozen.** A diff to `internal/read`, `internal/apply`,
  `internal/plan`, `internal/seen`, `internal/check` or `internal/state` names in its PR body the ADR
  or issue that owns it, and gets a `go-reviewer` pass before the Codex review. A record's fence keeps
  the byte-identical clause for each engine package it does not own (`adr.md`). A new module
  requirement follows `CONTRIBUTING.md`, Prerequisites.

## 7. Then the branch, the review, the merge, the memory

`git.md` → `reviews.md` → persist what the session learned in memoryd: `insert` or `fact` in the space
`tool-multipathreadwrite` (`craft` when it holds in any project), and an `open-thread` where you stopped.

## The habit to watch for

**The step after the satisfying part is the one that gets dropped.** A deferral without a receipt
in `docs/adr/BACKLOG.md`, a note instead of a fix, a follow-up that names nobody: this corpus has
recorded that pattern four times. Every Out of Scope item carries a disposition `adr-lint` accepts
(see `adr.md`), and a deferral that names BACKLOG.md has an entry there in the same commit.
