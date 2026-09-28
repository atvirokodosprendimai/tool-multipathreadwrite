# ADR-089: mrw ships an opencode plugin

**Status:** Accepted
**Accepted:** 2026-09-27 by Zy — "i have also added a remote branch - feat/opencode-adapter, we need to merge it here, and mention in the docs that we now support opencode", then "Fix, then merge"
**Date:** 2026-09-27
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-010, ADR-037, ADR-088
**Governs:** `cmd/opencode/mrw-plugin/**`, `opencode.json`
**Enforced-by:** None — the plugin is TypeScript; `cmd/opencode/mrw-plugin/test/smoke.test.mjs`, run by the task's fence and by CI, drives it against the built binary
**Served-path change:** a new surface — opencode sessions can call eight `mrw_*` tools; read and write run `mrw mcp`, the rest the CLI. The binary itself is unchanged.

## Context

`feat/opencode-adapter` (Zy, 2026-09-27, commits `5414b7d`, `108c753`, `cf1fbe0`) adds an opencode
plugin that wraps the CLI as opencode tools. Read before merging, it could not work outside Windows
and never delivered a write: `resolveBinary` returned `bin/mrw.exe` on every platform; `mrw_write`
saved the plan to `.mrw-plan-input` in the checkout and ran `mrw write` without naming it or
feeding stdin; `--root` went after `read`, where the parser does not take it; `mrw_iter` passed specs
without the verb `iter` needs; `stats` and `seen` dropped the exit code and stderr.

The Codex review of #260 found the design defect under those: opencode truncates a plugin's result
at 2,000 lines or 50 KiB (`packages/opencode/src/tool/truncate.ts`, v1.18.32), and a CLI read has
recorded every line it served before the tool returns, so the part the model never saw was writable
— ADR-002 inverted, the failure ADR-031's acknowledgements exist to prevent on the MCP surface.

## Existing Primitives Audit

- `mrw mcp` (ADR-010) serves `mrw_read` and `mrw_write` with a caller-set result ceiling (ADR-032)
  and checkpoint acknowledgements (ADR-031, ADR-039): a served page licenses only the runs the caller
  acknowledges holding whole. The plugin's read and write call it, one `tools/call` per process, as
  contract §77 already drives it.
- The CLI's `check`, `stats`, `seen` and `iter` have no MCP tool; the plugin runs them directly.

## Decision

1. `mrw_read` and `mrw_write` run `mrw --root <dir> mcp --max-result-chars 40000`: a page stays under
   opencode's 50 KiB limit, and a page cut anyway loses its tail's close markers, which cannot then be
   acknowledged. `mrw_write` takes `ack` and runs no check; `mrw_check` does.
2. The plugin resolves `bin/mrw` (`bin/mrw.exe` on Windows) in the session's checkout, else `mrw` on
   `PATH`; with no git worktree (opencode reports `/`) the checkout is the session directory.
3. The CLI tools return the exit code, stderr and stdout; the MCP tools return the content and say
   `error:` first on a refusal. A child that stops reading its input is reported, not raised, and an
   aborted call kills its child.
4. `--root` names the checkout on both surfaces; `mrw_iter` takes its verb and specs as given.
5. Arguments are declared with `tool.schema`, the zod the plugin API ships, and `tsc` refuses unused
   locals and parameters.
6. `test/smoke.test.mjs` drives the built plugin against the built binary, a cut page included; CI runs
   it on Linux, and a release waits for it.

## Alternatives Considered

- **CLI read and write.** Rejected after review: the CLI records a read before opencode truncates
  it. A byte cap on `mrw read` was weighed and rejected too — a new flag, and still a bet that
  opencode's limits never move, where an acknowledgement survives any cut.
- **Document `mrw mcp` for opencode and drop the plugin.** Kept as the second route in the README: the
  plugin adds `check`, `stats`, `seen` and `iter`, which the MCP surface does not have.
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

- An opencode session can call mrw as tools, with the ledger, every refusal, and acknowledgements
  that hold however opencode cuts a result.
- opencode prefixes MCP tools with their server's name, so the plugin's `mrw_read` and a server's
  `mrw_mrw_read` can both be registered; they are one engine and one ledger.

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
