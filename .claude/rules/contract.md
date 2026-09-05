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
- **Every child a row spawns runs under an alarm** — `perl -e 'alarm shift; exec @ARGV' N cmd`, the
  idiom §55 uses on every hook call — and the last row asserts nothing outlived the run. A mutant
  that hangs is a mutant the harness must bound: the 2026-09-04 regex mutant was killed by its row
  and then ran for fifteen hours under parent 1 (#101).
