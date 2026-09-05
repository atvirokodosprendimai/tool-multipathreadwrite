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
  import all use `perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" …`. A mutant that hangs is a
  mutant the harness must bound: the 2026-09-04 regex mutant was killed by its row and then ran for
  fifteen hours under parent 1 (#101). Extend the idiom to any new child a row could leave running.
- **The runner is a fresh process group, and the last rows prove nothing survived it.** The top of
  the file is a thin wrapper that forks the runner into a new group (handlers before the fork, the
  group set from both sides), forwards INT/TERM/HUP to it, and repeats its exit status; the runner
  consumes the `CONTRACT_GROUP` marker so nothing inherits it. The tail plants a deliberate orphan,
  asserts `pgrep -g $$` sees it, asserts the group is then empty, and runs a nested probe built from
  the file's own prologue to prove the exit status comes back, the EXIT trap's group kill reaps a
  re-parented orphan, and an INT to the wrapper ends the run 143. Always a new group, never "if not
  already a leader": as the first command of `… | tee` the script leads a group `tee` shares, and
  the kill took `tee` (found by both reviews). `pgrep -P $$` was the first form and misses an orphan
  already re-parented — a peer session named the group form on 2026-09-05. Write `pgrep` to a
  file, not a `$( )`: the substitution's own subshell is a group member and lists itself.
