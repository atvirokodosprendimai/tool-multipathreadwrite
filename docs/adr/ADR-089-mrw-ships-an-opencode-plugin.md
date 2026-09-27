# ADR-089: mrw ships an opencode plugin

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — "i have also added a remote branch - feat/opencode-adapter, we need to merge it here, and mention in the docs that we now support opencode", then "Fix, then merge"
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-010, ADR-037, ADR-088
**Governs:** `cmd/opencode/mrw-plugin/**`, `opencode.json`
**Enforced-by:** None — the plugin is TypeScript; `cmd/opencode/mrw-plugin/test/smoke.test.mjs`, run by the task's fence and by CI, drives it against the built binary
**Served-path change:** a new surface — opencode sessions can call eight `mrw_*` tools that spawn the mrw binary; the binary itself is unchanged.

## Context

`feat/opencode-adapter` (Zy, 2026-09-27, commits `5414b7d`, `108c753`, `cf1fbe0`) adds an opencode
plugin that wraps the CLI as opencode tools. Read before merging, it could not work outside Windows
and never delivered a write: `resolveBinary` returned `bin/mrw.exe` on every platform; `mrw_write`
saved the plan to `.mrw-plan-input` in the checkout and ran `mrw write` without naming it or
feeding stdin; `--root` went after `read`, where the parser does not take it; `mrw_iter` passed specs
without the verb `iter` needs; `stats` and `seen` dropped the exit code and stderr.

## Existing Primitives Audit

- `mrw mcp` (ADR-010) already serves `mrw_read` and `mrw_write` to any MCP host, opencode included.
  The plugin is the shell-backed surface: every CLI subcommand, the CLI's default check after a write
  (ADR-054), and no acknowledgement step, because a CLI read licenses what it served.
- `mrw write -` reads a plan from stdin, which is how the plugin delivers one.

## Decision

1. The plugin resolves `bin/mrw` (`bin/mrw.exe` on Windows) in the worktree, else `mrw` on `PATH`.
2. `mrw_write` sends the plan to `mrw write -` on stdin; nothing is written into the checkout.
3. Every tool returns the exit code, stderr and stdout; an aborted call kills its child.
4. `--root` goes before the subcommand; `mrw_iter` takes its verb and specs as given.
5. Arguments are declared with `tool.schema`, the zod the plugin API ships, and `tsc` refuses unused
   locals and parameters.
6. `test/smoke.test.mjs` drives the built plugin against the built binary; CI runs it on Linux.

## Alternatives Considered

- **Document `mrw mcp` for opencode and drop the plugin.** Kept as the second route in the README, not
  instead: the plugin reaches `check`, `stats`, `seen` and `iter`, which the MCP surface does not.
- **Merge the branch as it was.** Rejected: `mrw_write` could not write and nothing spawned outside
  Windows, so "we support opencode" would have been false.

## Component / Boundary Impact

`cmd/opencode/mrw-plugin` (new), `opencode.json`, CI, README, CONTRIBUTING. No Go package changes.

## Wiring & Contract Changes

A new opt-in surface. The CLI, its exit codes and the MCP server are unchanged.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- An opencode session in any checkout can call mrw's full CLI as tools, with the ledger and every
  refusal the CLI has.
- The plugin and `mrw mcp` both name tools `mrw_read` and `mrw_write`; register one or the other.

## Out of Scope

- Running the plugin inside opencode in CI (permanent: boundary: opencode is not installed on the runners; the smoke test drives the plugin's exported tools against the real binary instead)
- Publishing the plugin to npm (permanent: boundary: it is loaded from the checkout's `dist/`; revisit if a caller asks to install it without cloning)
- Windows in the smoke test (permanent: boundary: the Linux CI job runs it; binary resolution for Windows is one expression, read in review)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The plugin API changes shape | Medium | Medium | `@opencode-ai/plugin` is pinned by `package-lock.json`; CI builds it |
| A PATH `mrw` older than the checkout | Medium | Low | `bin/mrw` in the worktree wins |

## Rollback

Revert the task; remove `opencode.json`'s plugin entry.

## Follow-ups

None.
