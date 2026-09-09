@AGENTS.md

## Claude Code

The contributor drill — how a change gets from a decision to `main` in this repository — lives in
`.claude/rules/`, one file per topic. Claude Code loads those on its own; do not `@`-import them
here, because that would include their text a second time. `AGENTS.md` above is the tool-usage guide
and the build-and-check list, and it is what every other agent reads; the rules are project policy
and are pointed at from there. The three path-scoped rules (`adr.md`, `testing.md`, `contract.md`)
arrive natively only on a Read-tool read of a matching file; `.claude/settings.json` installs a
PostToolUse hook that delivers them on `mrw read`, `cat`, Write and the MCP tools too (issue #86,
contract §55). Installed on M's go-ahead of 2026-09-04 — *"fix of course"*, to the three items
flagged as M's call, of which the hook was one. It reads a command heuristically, so a subshell that
changes directory or a read whose output is redirected delivers nothing. With hooks disabled, or for
those shapes, Read one such file, or the rule itself, first.

**In Claude Code, drive mrw through Bash.** M, 2026-09-04: *"we must use mrw as bash tool when we
can, mcp is only for desktop."* The `mrw_read`/`mrw_write` MCP tools are for hosts without a shell.

The server was registered for this checkout anyway, on purpose. On 2026-09-09,
on the ONE account the measurement below was taken from, it was at USER scope in
that account's `~/.claude.json` (`command: mrw`, `args: ["mcp"]`), with no
`.mcp.json` in the repository. Scoped that tightly on purpose: `~` is not a
shared address, and a reviewer running under a different `HOME` read the same
path and found no `mrw` there, which is not a contradiction of this and cannot
be settled by either side re-reading. So treat it as dated evidence about one
file on one account — check your own before relying on it.

It is deliberate, because exercising the MCP arm is this project's job —
ADR-023, ADR-024 and ADR-031 were all measured through it — so a
`permissions.deny` on `mcp__mrw__mrw_*` here would break the one place those
tools must stay reachable. The lever that scopes them is the registration, not a
rule in this file. Measured 2026-09-09 across every Claude Code transcript on
that account: this repository made 9 MCP calls, all reads, all fixture probes
for those records, against roughly 2,800 shell invocations; a repository with no
such rule, reached through the same user-scope registration, made 57 MCP writes
in a day.
