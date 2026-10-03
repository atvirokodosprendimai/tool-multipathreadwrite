# ADR-123: a served path is resolved once

**Status:** Accepted
**Accepted:** 2026-10-03 by Zy — "Fix the Windows per-path cost (Recommended)", redirecting the plan's ADR-123 ("grep matches while reading") after a macOS measurement showed `--grep` at parity with `grep -rl` and a Windows peer's stack showed the cost in the resolve. The record's text was drafted after that answer.
**Date:** 2026-10-03
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-071, ADR-077, ADR-104, ADR-116
**Invalidates:** None — the boundary ADR-077 draws is unchanged
**Governs:** `internal/rooted/rooted.go`
**Enforced-by:** `internal/rooted/resolve123_test.go::TestAStateBaseBehindARepointedLinkIsRefused`
**Served-path change:** None in what is served or refused. `rooted.Resolve` makes fewer filesystem calls per path: the state base is resolved once while it exists, and the real path Resolve already computed is reused for the state check.

## Context

Windows peers measured `--grep` at 10–14 ms a file (3,000 files in 31–63 s, against `grep -rl`'s 0.41 s), and a timeout stack placed it in `read.Run` → `rooted.Resolve` → `InState` → `RealAsFarAsItExists` → `filepath.EvalSymlinks`, per served path (BACKLOG "From the Windows peers"). On Windows every path component is a syscall, and `InState` resolved the state base afresh — two passes over its components — and then the path itself a second time, after `Resolve` had just resolved it. The plan's original ADR-123, matching while reading, measured not needed: on macOS `--grep` over 20,000 files / 234 MB takes 1.5 s against `grep -rl`'s 1.8 s.

**Audit of the class** — *a per-path call that repeats work another call made*: `mrw read --grep 'RealAsFarAsItExists\(|EvalSymlinks\(' --exclude '*_test.go' internal/rooted/` — `Resolve` (its own `EvalSymlinks`, then `InState`'s `RealAsFarAsItExists` of the same path and of the base), `RealAsFarAsItExists` itself, and `InState` for its other callers.

## Existing Primitives Audit

- **`InState`** (ADR-077) — kept for its other callers; `inState` takes a path already resolved.
- **The identity walk** (the Codex review of #238) — kept exact: it is what refuses `.st/MRW` on a filesystem that folds case.

## Decision

1. **The state base is resolved once while it exists.** `resolvedBase` caches its real path and `FileInfo`. An entry stands only while the base, followed now, and its cached real path are both that same directory (`os.Stat` and `SameFile`, two calls, no walk over components); otherwise it is resolved again. Checking the old real path alone served a new base behind a re-pointed symlink — the in-process review of #330 found it, and `TestAStateBaseBehindARepointedLinkIsRefused` pins it. A base that does not exist yet is never reused: `SameFile` against a nil `FileInfo` is false.
2. **Resolve's real path is reused.** The path `EvalSymlinks(target)` returned is the state check's path; a missing leaf is resolved as `InState` resolved it.
3. **The boundary is unchanged.** The identity walk over the path's ancestors stays, one `Stat` each; caching it is deferred with a trigger (BACKLOG "From ADR-123").

## Alternatives Considered

- **Matching while reading** — the plan's original record; measured not needed (Context).
- **Caching the identity walk per directory** — deferred: it is ADR-077's guard against a case or mount spelling, and a long MCP session would need a staleness bound the record cannot yet state.

## Component / Boundary Impact

Owns `internal/rooted` (engine). `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `rooted.Resolve` | fewer syscalls; same answers | T1 | every reader and writer |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** `BenchmarkResolveADeepFile` (a file six directories down, the state base present) went from about 90 µs to about 38 µs per Resolve on macOS (2026-10-03, 2000 iterations, three runs); Windows pays more per syscall, so more per file.
- **Negative:** one package-level cache, guarded by a mutex and re-validated per use.
- **Neutral:** nothing served or refused changes.

## Out of Scope

- A contract row (permanent: boundary: no served or refused answer changes, and contract.sh cannot time Windows)
- Matching while reading, and caching the identity walk (deferred: docs/adr/BACKLOG.md "From ADR-123")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a cached base stands for a directory the base no longer names | Low | High | an entry stands only while the base followed now is the same directory by identity; `TestAStateBaseBehindARepointedLinkIsRefused` (a re-pointed link) and `TestACaseSpellingOfARecreatedStateBaseIsRefused` (a recreated base) |

## Rollback

Revert the task: Resolve resolves the base and the path again for each call.

## Follow-ups

- None — the record carries no open follow-up.
