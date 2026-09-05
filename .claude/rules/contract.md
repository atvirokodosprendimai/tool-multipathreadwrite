---
paths:
  - "scripts/contract.sh"
---

# contract.sh: the row that drives the built binary

- Sections are `# NN.` and are NOT in file order. Reserve the next with
  `grep -oE '^# [0-9]+\. ' scripts/contract.sh | sort -k2,2n | tail -1` — the trailing space is
  load-bearing: without it a dated comment such as `# 1.6 s` matches (#91), and the `sort -n` form
  answers `# 9.`.
- `$MRW` is the binary built at the top of the script. Drive IT. A Go test on the function cannot
  prove the binary calls it; §53 exists because `CheckRoot` could be correct and wired to nothing.
- Helpers: `want <exit> "$?" "<claim>"`, `ok "<claim>"`, `bad "<claim>"`, `fixture` (no arguments;
  creates `$R`), and `skip "<claim>"` for a trigger root ignores.
- **A row pairs the case that must pass with the case that must fail.** A row that only scores the
  good case passes with a tool that always says yes.
- `CONTRIBUTING.md` owns the platform note (bash, Linux CI) and why an exit code is never read
  through a pipe.
- Every new promise gets a row; the row's number is cited in the record's Acceptance fence as
  `grep -q '^# NN\.' scripts/contract.sh`.
- **Every run of hook code in §55 goes under an alarm** — `hook55`, `closed55` and the `plan_paths`
  import all use `perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" …` — and the last row of the
  run asserts no child is still attached at exit (direct children only; an orphan already under
  parent 1 is invisible to it, so the alarm is the fix and that row is the check). A mutant that
  hangs is a mutant the harness must bound: the 2026-09-04 regex mutant was killed by its row and
  then ran for fifteen hours under parent 1 (#101). Extend the same idiom to any new child a row
  could leave running.
