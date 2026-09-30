# ADR-103: One checkout, one identity; and the filesystem root is a root

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — approved the plan "resolve the seven findings of the 2026-09-30 Codex design review" (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`), whose ADR-103 is this record's scope, and set the goal "deliver open tasks end to end". The record's text was drafted after that approval and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-004, ADR-006, ADR-071, ADR-075, ADR-076, ADR-077
**Governs:** `internal/rooted/rooted.go`, `internal/rooted/links.go`, `internal/links/`, `internal/state/state.go`, `internal/check/check.go`, `internal/read/astgrep.go`, `scripts/contract.sh`
**Enforced-by:** `internal/rooted/contains103_test.go::TestContainsHoldsUnderAFilesystemRoot`
**Invalidates:** none — checked. ADR-071's junction rule is kept and made to hold for the state key too; ADR-075's one-writer lock now means one per physical checkout on Windows as elsewhere; ADR-004's key is unchanged off Windows.
**Served-path change:** (1) `--root /` (or a Windows volume root such as `C:\`) serves, writes and checks the paths beneath it, where every one was refused as outside the root. (2) On Windows, a checkout reached through a junction shares the state directory — ledger, working set, tally and writer lock — of the checkout it leads to, where each spelling had its own; its old state becomes an orphan `mrw seen --prune` reports. (3) On Windows, `mrw check` compares a path given absolutely, and scopes one, against a junction-reached root as the target it is. Off Windows nothing else changes.

## Context

**What was observed** (2026-09-30, the Codex design review of `f1d5996`, confirmed in source; #1 reproduced):

1. `rooted.Contains` (`internal/rooted/rooted.go:176-177`) appends a separator to the root, so for `/` it asks
   for the prefix `//` and for `C:\` the prefix `C:\\`: every child is refused. `mrw --root / read etc/hosts`
   answered "/private/etc/hosts, which is outside the root /". MCP honours an explicit `--root /`
   (`internal/mcp/root.go:113-118`), so the refusal is reachable on both surfaces.
2. The state key (`internal/state/state.go:164-174` `absReal`) resolves a root with `filepath.EvalSymlinks`
   alone. Since Go 1.23 that does not follow a Windows junction, while `rooted.Abs` does (`throughLinks`,
   ADR-071). So a checkout and a junction to it get two state directories, two ledgers and two
   `seen.write.lock` files: two mrw processes can validate and commit against one tree at once.
3. `check.confine` (`check.go:461-467`) and `check.placed` (`:636-642`) and `read.astGrepRel`
   (`astgrep.go:229`) canonicalise with `EvalSymlinks` alone, then compare with a root `rooted.Abs` made
   through junctions — the same mismatch on Windows.
4. `rooted` imports `state` (`rooted.go:22`, for `InState`), so `state` cannot call `rooted`.

**Audit of the class.** The class is *a root or path canonicalised without the junction walk, or compared with
a separator-appending prefix*. Enumerated 2026-09-30 by `mrw read --grep 'EvalSymlinks' --exclude '*_test.go'
internal/ cmd/` and `mrw read --grep 'rooted\.Contains' internal/`: `state.absReal`, `check.confine`,
`check.placed`, `read.astGrepRel` — **4**, in scope; `rooted.Real` and `rooted.Abs` already walk junctions;
`mcp/root.go:154` compares the home directory for a refusal message only — **left out**, it canonicalises no
root mrw serves. `Contains` has five callers (`rooted.go:131`, `:297`, `check.go:475`, `read.go:404`,
`walk.go:127`), all fixed by fixing `Contains`.

## Existing Primitives Audit

- **`throughLinks`/`linkFS`/`Real`/`IsRooted`** — moved, not rewritten, into the leaf package `internal/links`,
  which `rooted` and `state` both import; `rooted.Real` and `rooted.IsRooted` forward to it, and `rooted` keeps
  `throughLinks`/`osLinks`/`followLinks` as one-line forwarders, so every caller and the walk's tests stay put.
- **`win32Alias`/`win32Device`** — stay in `rooted`: they are refusals, and `state` must key an alias root rather
  than refuse it (the 2026-09-30 design critique).

## Decision

1. `Contains(absRoot, p)` asks for the prefix `absRoot` when it already ends in a separator (a filesystem or
   volume root), and `absRoot` plus one separator otherwise.
2. The link walk (`Through`), its filesystem seam (`FS`), `IsRooted`, `Real` and `Follow` live in
   `internal/links`. `state.absReal` canonicalises through `links.Real`, so every spelling of a checkout —
   symlink or junction — keys one state directory. `check.confine`, `check.placed` and `read.astGrepRel`
   canonicalise through `rooted.Real`.

## Alternatives Considered

- **Set a canonicaliser into `state` from `rooted`'s `init`** — rejected: a dependency no import shows, and
  `state` used without `rooted` would key by the old rule.
- **Move `win32Alias` too** — rejected: `state` must give an alias root a stable key, not refuse it.
- **Refuse `--root /`** — rejected: ADR-062's MCP rule honours an explicit root, and the refusal was a bug in
  the comparison, not a policy.

## Component / Boundary Impact

New leaf package `internal/links` (imports only the standard library). Engine packages owned: `internal/rooted`,
`internal/state`, `internal/check`, `internal/read` (one line). `internal/apply`, `internal/plan`,
`internal/seen`, `internal/lines`, `internal/iter` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| every path check | a filesystem or volume root contains its children | T1 | read, write, check, MCP |
| the state directory (Windows) | one per physical checkout | T2 | ledger, working set, tally, writer lock |
| `scripts/contract.sh` | §198 (T1) | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| none | — | — | T1 and T2 are independent |

## Implementation

See `tasks/README.md`: T1 (`Contains`), T2 (one canonical identity).

## Consequences

- **Positive:** a root is a root whatever it is; one checkout has one lock on every platform.
- **Negative:** on Windows a junction-reached checkout starts from a fresh state directory once; the old one is
  an orphan until `mrw seen --prune`. During an upgrade, an old and a new binary on such a root take different
  locks until the old process exits.
- **Neutral:** off Windows the state key and every answer except the filesystem-root one are unchanged.

## Out of Scope

- Anchoring file operations to a handle (deferred: `docs/adr/BACKLOG.md` "From ADR-103")
- The home-directory comparison in `internal/mcp/root.go:154` (permanent: boundary: it chooses a refusal message and canonicalises no served root)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the link walk's tests stop exercising it | Low | Med | they stay in `internal/rooted` and drive `links.Through` through `throughLinks`; only their fake-filesystem helper changed (it builds a `links.FS`), so no test body — and no ADR-071 lock — moved |
| `Contains` now admits a sibling of a root that ends in a separator | Low | High | only a filesystem or volume root ends in one after `filepath.Clean`, and every path is beneath it; the `/repo` vs `/repo-backup` case stays in the test |

## Rollback

Revert T1 and T2. On Windows the junction-keyed state directory is abandoned again; nothing is deleted.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `internal/rooted/contains103_test.go::TestContainsHoldsUnderAFilesystemRoot` in T1's commit.
