# ADR-059: Honour `fenceTimeout` as the check bound

**Status:** Accepted
**Accepted:** 2026-09-13 by M — *"both"*
**Date:** 2026-09-13
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-054, docs/adr/BACKLOG.md
**Governs:** `internal/check/check.go`, `scripts/contract.sh` (§97)
**Enforced-by:** `internal/check/timeout_alias_test.go::TestLoadHonoursFenceTimeout`
**Invalidates:** None
**Served-path change:** a `.quality-harness.json` that declares `fenceTimeout` (seconds) bounds the check the same way `timeout_seconds` already does; both keys set to different values refuse at `Load` (CLI exit 2 on `mrw check`).
**Notes:** Zeus field report 2026-09-13. Their harness is `{ "check": "cargo clippy … && cargo nextest …", "fenceTimeout": 1800 }`. mrw `Load` unmarshalled only `timeout_seconds`; unknown JSON is dropped; `TimeoutSeconds` stayed 0; `Run` used `defaultTimeout` (5 minutes). That is the `.jsonl` hang they reported, not an unread file. M: *"both"* — this alias, and native unlink (ADR-057). Per-extension skip of the check is not this record.

## Context

quality-harness writes `fenceTimeout` (camelCase, seconds). mrw writes `timeout_seconds`. Both mean the same number. Zeus declared 1800 and still hit the five-minute default because encoding/json silently discards an unknown key. The check then ran the whole cargo line on every non-prose write, including `.jsonl`, and timed out at the default.

This repository's own `.quality-harness.json` uses `timeout_seconds`: 300. That path must keep working.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `Config.TimeoutSeconds` `json:"timeout_seconds"` | `internal/check/check.go` | The only bound `Load` keeps. Zero means `defaultTimeout`. |
| `defaultTimeout` | same | `5 * time.Minute`. Used when `TimeoutSeconds == 0`. |
| `maxTimeoutSeconds` | same | Clamp in `Run`, not in `Load`. Unchanged. |
| Unknown JSON keys | `encoding/json` | Dropped. `fenceTimeout` never reached `Run`. |
| `check.Load` error | `cmd/mrw` write and `mrw check` | Exit 2. Write currently Loads after apply (pre-existing for corrupt JSON). |

## Decision

**1. `fenceTimeout` is an alias of `timeout_seconds`.** After unmarshal, if `timeout_seconds` is 0 and `fenceTimeout` is > 0, `TimeoutSeconds` becomes `fenceTimeout`. `Run` keeps reading only `TimeoutSeconds`.

**2. Both set and equal is fine.** Both set and different is a `Load` error naming both values. Silently picking one is how a caller keeps believing the other.

**3. `timeout_seconds` alone is unchanged.** This checkout's harness stays valid.

## Alternatives Considered

- **Ignore `fenceTimeout` and teach Zeus to rename the key** — they already declared the bound; the hang is mrw dropping it. Teaching does not stop the next quality-harness checkout.
- **Prefer `fenceTimeout` when both are set** — two numbers that disagree are a config error, not a preference.
- **Accept any unknown camelCase alias** — one known alias is the field Zeus and quality-harness actually write.
- **Widen the prose list so `.jsonl` skips the check** — that takes `.toml` with it (Cargo.toml). Different record; Zeus asked for it as the other half of friction 2.

## Component / Boundary Impact

`internal/check.Config` gains one exported JSON field. `Load` gains the alias rule and the disagree refusal. No CLI flag. `Run` is unchanged.

## Wiring & Contract Changes

- `Config.FenceTimeout int` `json:"fenceTimeout"`.
- After a successful unmarshal, resolve into `TimeoutSeconds` or return an error.
- Contract **§97**: a Zeus-shaped `fenceTimeout` of 1 second times out a `sleep 5` check (exit 3 on a `.go` write); disagreeing keys refuse `mrw check` at exit 2.

## Inter-task Contracts

| Produces | Consumes |
|----------|----------|
| T1: alias + disagree-refuse + §97 | — |

## Implementation

Tasks in `docs/adr/ADR-059-honour-fencetimeout/tasks/`. Red tests that fail on assertion → implement `Load` → mutants through `adr-verify --mutant` → §97 → exit-0 under the current digest.

## Consequences

- A quality-harness / Zeus checkout that already declared `fenceTimeout` gets the bound it typed.
- A harness that sets both keys to different numbers starts refusing instead of silently using `timeout_seconds` (or the default). None is known; this checkout sets only `timeout_seconds`.

## Out of Scope

- Skipping the check for `.jsonl` or any extension beyond the closed prose list. (deferred: docs/adr/BACKLOG.md "Per-extension check skip")
- A Rust `{packages}` mapper or harness `covers` glob. (permanent: boundary: ADR-054 already refused both)
- Changing `defaultTimeout`. (permanent: fact: five minutes is the bound when neither key is set; citation: file `internal/check/check.go:64`)
- Loading the harness before apply so a disagreeing pair cannot land a write. (permanent: fact: corrupt JSON already Loads after apply; citation: file `cmd/mrw/main.go:1056`)
- Honouring other quality-harness keys (`strictFrom`, …). (permanent: boundary: one alias, the one Zeus declared)

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| A checkout sets both keys to different numbers and starts failing | Low | The error names both; they pick one. |
| `fenceTimeout` in milliseconds is misread as seconds | Low | quality-harness documents seconds; Zeus's 1800 is 30 minutes, not 1.8s. |

## Rollback

Remove `FenceTimeout` and the resolve step; delete §97. Existing `timeout_seconds` harnesses are untouched.

## Follow-ups

- Per-extension check skip, if a later record takes it — BACKLOG "Per-extension check skip".

## Amendment, 2026-09-25: the harness is read before apply (ADR-072)

Out of Scope called loading the harness before apply permanent, on the fact that corrupt JSON
already loaded after it (its citation had drifted from `cmd/mrw/main.go:1056` to `:1120`). The
v1.25.1 adversarial round showed what that order costs: a malformed `.quality-harness.json` applied
the write and then exited 2 with only the JSON error. ADR-072 reverses it: `mrw write` reads the
harness first and refuses, exit 2, nothing written. A disagreeing `timeout_seconds` and
`fenceTimeout` pair is refused the same way, before the write.
