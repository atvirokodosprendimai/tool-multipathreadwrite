@AGENTS.md

## Claude Code

The contributor drill — how a change gets from a decision to `main` in this repository — lives in
`.claude/rules/`, one file per topic. Claude Code loads those on its own; do not `@`-import them
here, because that loads every one of them twice. `AGENTS.md` above is the tool-usage guide and the
build-and-check list, and it is what every other agent reads; the rules are project policy and are
pointed at from there.
