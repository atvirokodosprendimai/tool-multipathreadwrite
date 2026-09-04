---
paths:
  - "scripts/contract.sh"
---

# contract.sh: the row that drives the built binary

- Sections are `# NN.` and are NOT in file order. Reserve the next with
  `grep -oE '^# [0-9]+\.' scripts/contract.sh | sort -k2,2n | tail -1` (answers `# 53.` today; the
  `sort -n` form answers `# 9.`, and `^# [0-9]+` without the dot matches a dated comment).
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
