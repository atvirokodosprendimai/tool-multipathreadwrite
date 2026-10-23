# ADR-170: `mrw mcp` serves no tools until it is enabled

**Status:** Accepted
**Accepted:** 2026-10-23 by Zy — "Accept", on the record as amended after its Codex xhigh challenge.
**Amended:** 2026-10-23, before acceptance, from a Codex xhigh challenge (1 P1, 9 P2), every finding checked against the code: the opencode plugin enables by environment, not by flag, so it works against binaries before and after this record (Decision 3); the release is v1.75.0 with the migration leading its notes (Zy: *"v1.75.0 with a loud note"*); the off answer covers both protocol eras and discovery and refuses both tools with no effect (Decision 1); a precedence table (Decision 2); the host experience is stated as the protocol allows, not promised (Decision 1); the break campaign, `scripts/chaos.py`, `docs/agents.md`, README's registration examples and this machine's fixture registration are named (Decision 4, T1, T2).
**Date:** 2026-10-23
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-010, ADR-011, ADR-016, ADR-019, ADR-067, ADR-089, ADR-098, ADR-168, README.md, AGENTS.md, CLAUDE.md, docs/agents.md
**Governs:** `cmd/mrw/main.go`, `internal/mcp/mcp.go`, `internal/mcp/instructions.go`, `cmd/opencode/mrw-plugin/src/index.ts`, `scripts/contract.sh`, `scripts/break-campaign.sh`, `scripts/chaos.py`
**Enforced-by:** None — until T1 lands, which adds `cmd/mrw/mcpoff170_test.go::TestMrwMcpServesNoToolsUntilEnabled`
**Invalidates:** ADR-010 — the clause that `mrw mcp` serves `mrw_read` and `mrw_write` to any host that starts it: from this record they are served only when enabled. ADR-016's guidance to prefer the CLI with a shell is kept.
**Served-path change:** `mrw mcp` started without `--enable` or `MRW_MCP=on` lists no tools and refuses both tools, its instructions naming the CLI and the switch; with either, it serves as before.

## Context

Zy, 2026-10-23: adopters report that mrw "does not work" for them, and that their agents use the MCP tools instead of the CLI — asked what was reported, *"Agent picks MCP, gets worse results"*; asked the direction, *"jsut make a config option, and make it off by default"*, then *"or at least - turn if off by default"*; asked the version, *"v1.75.0 with a loud note"*.

An MCP server a host has registered puts its tools in front of the model in every session the registration reaches; at user scope that is every project on the machine (README, "Use it from an MCP host"). The model then picks `mrw_read`/`mrw_write` over shelling out, even where a shell exists, and the MCP surface is the narrower one: pages and `-- ck` checkpoints a caller must acknowledge before a write is licensed (ADR-014, ADR-031), no `--context`. The server's instructions already say to prefer the CLI with a shell (ADR-016); that advice reaches the model after it has been offered the tools, and some hosts never show it. Measured once on 2026-09-09 on one account (CLAUDE.md): a repository reached only through a user-scope registration made 57 MCP writes in a day.

The MCP surface is still the only way for a host with no shell, and the opencode plugin drives it as a library: `cmd/opencode/mrw-plugin/src/index.ts:80` spawns `mrw --root DIR mcp` and sends one `tools/call` (ADR-089). The plugin resolves `bin/mrw` in the checkout, else the `mrw` on PATH (`index.ts:37`), so it meets binaries older and newer than itself.

## Existing Primitives Audit

- `mrw mcp`'s `--max-result-chars` with `MRW_MAX_RESULT_CHARS` (`cmd/mrw/main.go:431`, `resultBudget` at `:504`): the flag-and-variable shape the switch takes. Reuse.
- `tools()`, `instructionsText()` and discovery (`internal/mcp/mcp.go:118`, `:470`): the off answer is the same server with an empty tool list and its own instructions, in both eras. Reuse.
- Removing `mrw mcp`: rejected (Alternatives).

## Decision

**1. Off by default.** `mrw mcp` started without the switch runs the same startup — root resolution and the result budget are validated as today, so a bad registration still fails at start — and then speaks the protocol in both eras (ADR-067): `initialize`, `server/discover`, `ping` answer. It keeps `capabilities.tools` and answers `tools/list` with an empty array. A `tools/call` of `mrw_read`, `mrw_write` or any name is a JSON-RPC error naming the switch, and has no effect: nothing is read, written or recorded. The instructions — in `initialize` and in discovery — are short: mrw's MCP tools are off; with a shell, use the CLI (`mrw read`, `mrw write`, `mrw --skill`); a host with no shell enables them with `mrw mcp --enable` or `MRW_MCP=on`. What a host shows is the host's: instructions and protocol errors are not guaranteed to reach a model or a person (ADR-016), so README and the release notes, not only the server, carry the switch. A process already running keeps what it was started with; a host picks up the change when it restarts the server, and a host that cached the tool list may keep offering the tools until then, each call then refused.

**2. On by choice; precedence.** The switch is resolved once, at start:

| `--enable` | `MRW_MCP` | Tools |
|---|---|---|
| given (`--enable` or `--enable=true`) | anything | on |
| `--enable=false` | anything | off |
| absent | `on` | on |
| absent | `off` or empty or unset | off |
| absent | any other value | off, and stderr names the value and the two it accepts |

The variable is read only when the flag is absent, as `MRW_MAX_RESULT_CHARS` is.

**3. The opencode plugin enables it by environment.** The plugin's `mcp` call sets `MRW_MCP=on` in the child's environment and passes no new flag. A binary from before this record ignores the variable and serves as it did; a binary from this record on reads it and serves. The plugin works against either, with no minimum binary.

**4. Told where callers look, and the release is v1.75.0.** `mrw mcp --help`, README's MCP section (its registration examples for a shell-less host carry `--enable`), the opencode "Or over MCP alone" recipe, AGENTS.md and `docs/agents.md` say the tools are off until enabled and that a host with a shell should not enable them. The v1.75.0 release notes lead with the migration: a registration that wants the tools adds `--enable` to its args or `MRW_MCP=on` to its environment, then restarts. This machine's deliberate fixture registration (CLAUDE.md, user scope, `args: ["mcp"]`) is the owner's to migrate, and CLAUDE.md's note says so.

**What would make this decision wrong:** shell-less adopters who never learn the switch and conclude mrw is broken. Nothing measures it yet; the release notes and README are the mitigation.

## Alternatives Considered

- **Remove `mrw mcp`.** Rejected: shell-less hosts lose mrw, the opencode plugin loses its engine, and it is a breaking removal for a problem a default solves.
- **Keep it on and say "prefer the CLI" more loudly.** Rejected: ADR-016 already says it, inside the server, after the tools are offered.
- **A config file.** Rejected for now: mrw has no user config file, and an argument or variable lives in the registration that starts the server.
- **Off means the server refuses to start.** Rejected: a host shows a failing server and learns nothing about why.
- **The plugin passes `--enable`.** The first draft. Rejected after the challenge: an older binary on PATH refuses an unknown flag, and the plugin and the binary are installed separately.
- **v2.0.0.** Offered; Zy chose v1.75.0 with the migration leading the notes.

## Component / Boundary Impact

- `cmd/mrw`: the switch and its precedence.
- `internal/mcp`: the off answer in both eras.
- `cmd/opencode/mrw-plugin`: sets `MRW_MCP=on` for its child.
No package changes ownership.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw mcp --enable`, `MRW_MCP` | new switch, off by default | `cmd/mrw` | MCP host registrations |
| `tools/list` without the switch | empty array, `capabilities.tools` kept | `internal/mcp` | MCP hosts |
| `tools/call` without the switch | JSON-RPC error naming the switch, no effect | `internal/mcp` | MCP hosts, plugins that do not enable |
| `initialize` and `server/discover` instructions without the switch | the off text | `internal/mcp` | MCP hosts |
| opencode plugin child environment | `MRW_MCP=on` | `cmd/opencode/mrw-plugin` | `mrw mcp` |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `mrw mcp --enable` / `MRW_MCP=on` | T1 | T2 | Yes for an existing registration that relied on the tools: it adds the switch (Decision 4) |

## Implementation

`tasks/README.md`: T1 (the switch, the off answer, every harness that starts `mrw mcp`), T2 (the plugin and the words). One pull request; released as v1.75.0.

## Consequences

- **Positive:** a registered server no longer puts mrw's MCP tools in front of a model that has a shell.
- **Negative:** every existing MCP registration that wants the tools must add the switch and restart; a minor version carries a change some callers feel (Zy's choice).
- **Neutral:** a host with no shell keeps the full MCP surface, one argument away; the plugin works against any binary.

## Out of Scope

- Removing `mrw mcp` (permanent: boundary: shell-less hosts and the opencode plugin need it)
- A user config file for mrw (permanent: boundary: the switch lives in the registration that starts the server)
- Measuring whether shell-less adopters find the switch (permanent: boundary: no data source reaches their hosts; README and the release notes are the mitigation)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A shell-less user does not find the switch | Med | Med | the off instructions and the refusal name it where a host shows them; README and the release notes say it regardless |
| A host keeps a cached tool list | Med | Low | each call is refused naming the switch; a restart of the server clears it; the modern answer's `ttlMs` is a hint, not a guarantee |
| An existing registration silently loses the tools | High | Low | the release notes lead with it; the change is the intent |

## Rollback

Make the default on again (one line), keeping the switch; nothing persistent changes.

## Follow-ups

- [ ] none at authoring
