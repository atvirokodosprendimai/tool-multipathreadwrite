# ADR-131: a walk resolves each directory once

**Status:** Accepted
**Accepted:** 2026-10-08 by Zy — "ok, go next with ADR-131", then, shown the Windows profile, "Full per-walk cache (Recommended)" over the plan's narrower state-verdict map and over withdrawing the record. The record's text was drafted after that answer.
**Date:** 2026-10-08
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-071, ADR-077, ADR-081, ADR-106, ADR-122, ADR-123
**Invalidates:** None — the boundary ADR-007, ADR-071 and ADR-077 draw is unchanged; ADR-123's deferral of the identity-walk cache is taken here
**Governs:** `internal/rooted/rooted.go`, `internal/read/walk.go`, `internal/read/astgrep.go`
**Enforced-by:** `internal/rooted/resolver131_test.go::TestAResolverAnswersAsResolveDoes`
**Served-path change:** None in what is served or refused. A `--grep` walk and an `--ast-grep` call resolve the files they discover through a `rooted.Resolver` that resolves the root and the state base once and each directory once, and then judges a regular file by one `Lstat` of its name.

## Context

ADR-123 removed the repeated resolution of the state base and deferred the rest: *"The ancestor identity walk in `inState` … Arm when a Windows timing after ADR-123 still shows the per-file cost, with a per-walk cache whose staleness the record can bound"* (BACKLOG "From ADR-123"). The plan (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`, item 7) gated this record on such a timing and sketched a per-directory map of the state verdict alone.

**What was measured** (2026-10-06, a Windows peer, AMD Ryzen 7 5800X3D, Windows 11 Pro 10.0.26200, Defender real-time protection on, `main` at `d493df7`, the peer's Go 1.24.2): `BenchmarkWalkDeep131` — 3,000 files, 300 in each of ten directories ten levels deep — took **9.39 s a walk, 3.1 ms a file**. `rooted.Resolve` was **84.6%** of all CPU (51.3 s of 60.6 s), every sample under `walkDir`. Within it: `inState` 33.8%, `filepath.EvalSymlinks` 33.4%, `rooted.Abs` 19.9%, `throughLinks` 12.7%. Reading the files was about 7%.

So the gate is earned, but not for the map the plan sketched: the state verdict is a third of Resolve. The rest is the same work repeated for every file of a directory — the root resolved again (`walk.go` passed the unresolved root to `Resolve` per file), and every component from the volume root walked twice, once for links and once for the real path. Shown this, Zy chose the full per-walk cache.

**Audit of the class** — *a loop that resolves every path a finder DISCOVERED*, where the count is the tree's size rather than what the caller named: `mrw read --grep 'rooted\.Resolve\(' --exclude '*_test.go' internal/ cmd/` (2026-10-08) — 15 sites. Two are discovery loops and are converted: the walk's discovered files (`walk.go`, `walkDir`) and ast-grep's hits (`astgrep.go`, ADR-122's same shape). The other 13 resolve what a caller named — a spec, a hunk, a working-set entry, an MCP argument — and stay on `Resolve`: `read.go`'s serve (deliberately, below), `walk.go`'s named path, `apply.go`, `plan.go`, `check.go` (3), `ingest` (3), `mcp/tools.go` (2), `main.go`.

## Existing Primitives Audit

- **`rooted.Resolve`** — kept unchanged in behaviour; its body after `Abs` becomes `resolveIn`, which the Resolver calls for every path it does not judge itself, so there is one implementation of the hard cases.
- **`inState`** (ADR-077, ADR-123) — its comparison becomes `inStateAt(b, bi, q)`, called once per directory by the Resolver.
- **`resolvedBase`** (ADR-123) — the Resolver asks it once, at its first path.
- **`throughLinks`** (ADR-071) — the Resolver follows a directory's junctions with it, once per directory.

## Decision

1. **`rooted.Resolver`** answers every path exactly as `Resolve(root, path)` does. `NewResolver(root)` resolves the root once; its refusal is returned for every path.
2. **Each directory is resolved once**: its links (`throughLinks`), its real path (`EvalSymlinks`), and whether it or an ancestor is the state base (`inStateAt`). A file whose `Lstat` says it is regular is then judged from its directory: it is not a link, so its real path is the directory's real path joined with its name; it is not a directory, so it is the state base only if it is the base's own path. Device names are checked on the path as written and through the directory's links, as `Resolve` checks them.
3. **Everything else is resolved whole**, by `resolveIn`: a link, a directory, a name that is not there or cannot be examined, a name spelled as a directory, a directory that would not resolve, and every path when the state base exists and is not a directory.
4. **A Resolver lives as long as one walk.** `read.Walk` makes one and drops it when it returns; `AstGrep` makes one per call. What a directory resolved to is stale for at most that long: a directory swapped for a link, or made the state base, during a walk is seen by the next walk.
5. **The serve is unchanged.** `read.Run` resolves every path it serves with `Resolve`, afresh, before it opens it to serve (`read.go`), and `Walk` records nothing (ADR-005). A stale entry can therefore never serve a file or license a line; it can only let the walk read a file to MATCH it — which is the pattern oracle `walk.go` describes, and is the bound in Risks.

## Alternatives Considered

- **The plan's map: a per-directory state verdict only** — rejected after the profile: `inState` is a third of Resolve, so it would leave two thirds of the measured cost (Context).
- **Withdrawing the record** — rejected: the gate's own condition held, 85% of a walk in Resolve.
- **A process-wide cache with a time bound** — rejected: an MCP session lives for hours, and no time bound is a bound on what changed; the walk is the unit a caller sees.
- **Opening the walk through `os.Root`** — rejected for the read path by ADR-106 (Out of Scope there); the walk names files the root cannot hand it.

## Component / Boundary Impact

Owns `internal/rooted` and `internal/read` (both engine). `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines` stay byte-identical. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `rooted.NewResolver`, `(*Resolver).Resolve` | new; answers as `Resolve` | T1 | `read.Walk`, `read.AstGrep` |
| `mrw read --grep`, `--ast-grep`, `mrw_read` `grep`/`ast_grep` | fewer syscalls per discovered file; same answers | T1 | callers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** on macOS (Apple M5, Go 1.27.1, 2026-10-08, load 6–13 on 10 cores, three runs each) `BenchmarkWalkDeep131` went from about 190 ms to about 62 ms a walk. The Windows timing, before and after on one machine, is T1's S4.
- **Negative:** a second way into the boundary; the differential test is what keeps it the same boundary.
- **Neutral:** nothing served or refused changes; receipts and exit codes are unchanged.

## Out of Scope

- A contract row (permanent: boundary: no served or refused answer changes, and a cache that dies with its process cannot be observed across two `mrw` runs)
- The 13 named-path call sites (permanent: boundary: each resolves what a caller named, a count bounded by the request, not by the tree)
- A Resolver for `read.Run`'s serve (permanent: boundary: the serve's fresh Resolve is what bounds this record's staleness to the walk's matching reads)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a directory swapped for a link out of the root mid-walk lets the walk read a file outside to match it, and a REFUSED line then says it matched | Low | Med | the window grows from one syscall gap to the rest of the walk; the attacker must race writes inside the checkout; the serve still refuses it; the next walk sees it (`TestAWalkCacheDoesNotOutliveItsWalk`) |
| the Resolver answers differently from Resolve | Low | High | `TestAResolverAnswersAsResolveDoes` asks both over every hazard, twice; `TestAResolverFollowsAJunctionAsResolveDoes` on the Windows shards; every hard case is resolved by Resolve's own code |
| the leaf's case differs from disk on Windows, where EvalSymlinks would normalise it | Low | Low | only a refusal's message names the real path, and a discovered refusal is silent; the verdict is decided by the directory |

## Rollback

Revert the task: the walk and ast-grep call `rooted.Resolve` per file again.

## Follow-ups

- None — the record carries no open follow-up.
