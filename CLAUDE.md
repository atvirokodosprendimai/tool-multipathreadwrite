@AGENTS.md

## Claude Code

The contributor drill — how a change gets from a decision to `main` in this repository — lives in
`.claude/rules/`, one file per topic. Claude Code loads those on its own; do not `@`-import them
here, because that would include their text a second time. `AGENTS.md` above is the tool-usage guide
and the build-and-check list, and it is what every other agent reads; the rules are project policy
and are pointed at from there. The three path-scoped rules (`adr.md`, `testing.md`, `contract.md`)
arrive natively only on a Read-tool read of a matching file; `.claude/settings.json` installs a
PostToolUse hook that delivers them on `mrw read`, `cat`, Write and the MCP tools too (issue #86,
contract §55). With hooks disabled, Read one such file, or the rule itself, first.

**In Claude Code, drive mrw through Bash.** M, 2026-09-04: *"we must use mrw as bash tool when we
can, mcp is only for desktop."* The `mrw_read`/`mrw_write` MCP tools are for hosts without a shell.
