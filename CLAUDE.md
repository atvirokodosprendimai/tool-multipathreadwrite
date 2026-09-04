@AGENTS.md

## Claude Code

The contributor drill — how a change gets from a decision to `main` in this repository — lives in
`.claude/rules/`, one file per topic. Claude Code loads those on its own; do not `@`-import them
here, because that would include their text a second time. `AGENTS.md` above is the tool-usage guide
and the build-and-check list, and it is what every other agent reads; the rules are project policy
and are pointed at from there. ⚠ The three path-scoped rules (`adr.md`, `testing.md`,
`contract.md`) arrive only on a Read-tool read of a matching file — never on an `mrw read` — so
Read one such file, or the rule itself, before working in that area (issue #86).
