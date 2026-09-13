# ADR-057: Native `unlink` and `rename`

**Status:** Accepted
**Accepted:** 2026-09-13 by M — *"both"*
**Date:** 2026-09-13
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-002, ADR-004, ADR-008, ADR-021, ADR-044, ADR-051, docs/adr/BACKLOG.md
**Governs:** `internal/plan/plan.go`, `internal/apply/apply.go`, `internal/ingest/applypatch.go`, `internal/seen/seen.go`, `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/schema.go`, `scripts/contract.sh` (§98–§99)
**Enforced-by:** `internal/apply/unlink_test.go::TestUnlinkRemovesThePath`
**Invalidates:** None — closes ADR-051's deferred `*** Delete File:` / `*** Move to:`
**Served-path change:** `@@ path - unlink` removes the path; `@@ old - rename` with a one-line dest body moves it; `apply_patch` `*** Delete File:` / `*** Move to:` compile to those hunks. A sibling failure still writes nothing.
**Notes:** BACKLOG From ADR-051, planned 2026-09-13, arm *"unlink op"*. Zeus field report 2026-09-13: after splitting a suite into 22 files they `rm`'d the original outside the ledger. M: *"both"*. `delete` stays a line-range (ADR-008). Emptying a file is not unlink.

## Context

mrw can create a path and can delete lines. It cannot remove a path. Zeus's 104-hunk split landed; the leftover original had to be removed by `rm`. `apply_patch` already parses `*** Delete File:` and `*** Move to:` and compile-refuses them: *"mrw has no unlink"*.

## Existing Primitives Audit

| Primitive | Where | Finding |
|-----------|-------|---------|
| `OpCreate` / address `-` | `internal/plan` | Path-level op with no line address. Unlink/rename are the same shape. |
| Two-phase stage then rename | `internal/apply` `apply()` | Writes land together. Unlink copies aside and removes on commit; a later fail restores. |
| `seen.Record` | `internal/seen` | Merges; cannot drop a path. `Forget` was deleted with no caller. Unlink needs `Drop`. |
| `*** Delete File:` refuse | `internal/ingest/applypatch.go` | Compiles nothing. Maps to `@@ p - unlink`. |
| `delete` | ADR-008 | Line-range. `1-$ delete` leaves the empty path. |

## Decision

**1. Native ops.** `@@ path - unlink` — no line address, empty body OK. `@@ old - rename` — no line address, body is exactly one dest path, root-relative. Dest exists: refuse unless dest is also unlinked in this plan.

**2. Whole-file licence.** The ledger must cover lines 1–N (or `Whole()`, or `--force`). A one-line read does not license removing the path.

**3. One path, one kind.** Unlink or rename may not mix with line-edits on the same path in one plan.

**4. Stage then commit.** Copy-aside, then remove/rename. Any sibling failure restores the aside (ADR-001, ADR-004). Regular file writes keep today's two-phase loop; unlinks commit after those writes succeed, so a failed write never leaves a hole.

**5. `apply_patch` compiles.** `*** Delete File: p` → `@@ p - unlink`. `*** Move to: q` after `*** Update File: p` → one rename hunk. Same `--format=apply_patch`. No third MCP tool (ADR-044). A Move to that also carries `@@` hunks is refused this slice.

**6. Receipt names the path.** FileResult `removed` / `renamed_to`. Human line `removed path`. ADR-008's `removed_first`/`removed_last` stay on line-range `delete`.

**7. Ledger.** `seen.Drop` the unlinked path. Record the dest after rename as a whole-file write.

## Alternatives Considered

- **Map Delete File to `@@ p 1-$ delete`** — leaves the empty path. Zeus still `rm`s.
- **`os.Remove` before siblings succeed** — a later fail leaves a hole. ADR-001.
- **Unlink-then-create for Move to** — two hunks, dest-exists races, Windows `os.Rename` onto existing dest fails. One rename hunk.
- **A third MCP tool** — ADR-044.

## Component / Boundary Impact

Parser gains two ops. Apply gains path-level commit. seen gains Drop. ingest compiles the two apply_patch headers. cmd and MCP drop ledger entries. MCP schema describes `files.removed` and `files.renamed_to`.

## Wiring & Contract Changes

- `plan.OpUnlink`, `plan.OpRename`.
- `apply.FileResult.Removed`, `RenamedTo`.
- `seen.Drop(root, paths)`.
- Contract **§98** (native unlink/rename; sibling fail restores) and **§99** (Delete File / Move to compile and apply).

## Inter-task Contracts

| Produces | Consumes |
|----------|----------|
| T1: parse + apply + Drop | — |
| T2: apply_patch compile | T1 ops |
| T3: §98–§99, teach, BACKLOG | T1, T2 |

## Implementation

Tasks in `docs/adr/ADR-057-unlink-and-rename/tasks/`. Red tests that fail on assertion → implement → mutants → contract → teach.

## Consequences

- A split that leaves the original behind can unlink it in the same plan.
- Codex Delete File / Move to stop compile-refusing.
- A path named twice (unlink + replace) is a refusal, both spellings ADR-021.

## Out of Scope

- Mapping Delete File to `1-$ delete`. (permanent: boundary: that leaves the empty path)
- Removing before siblings succeed. (permanent: boundary: ADR-001)
- A third MCP tool. (permanent: boundary: ADR-044)
- Move to with in-file hunks (content change plus rename). (deferred: docs/adr/BACKLOG.md "Move to with hunks")
- ast-grep-shaped `--grep`. (deferred: docs/adr/BACKLOG.md; ADR-058)

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Windows `os.Rename` onto existing dest | Med | Dest unlinked in-plan is removed first; otherwise refuse. `rooted.IsRooted`, never `filepath.IsAbs`. |
| Restore fails after a partial unlink | Low | Aside is beside the target; restore is rename-back. §98 asserts the sibling-fail case. |

## Rollback

Remove the two ops, Drop, the compile arms, §98–§99. `delete` and `create` are unchanged.

## Follow-ups

- Move to with hunks, if a caller hits it — BACKLOG.

## Stress suite

Measured 2026-09-13 on this tree, before the PR. Oracle is the Decision (path gone iff applied; dest-exists refuses unless dest unlinked in-plan; mix with a line-edit on the same path refuses; sibling fail writes nothing).

- Arm 1: `FuzzUnlinkRemovesOrRefuses` ~7k execs / 15s, applied iff path gone. `FuzzBalanceDelta` ~369k execs / 15s (regression, line-range nets).
- Arm 2: `TestRandomisedPathOpsFollowTheDecision` seeds 54/1/7/13/99 × 80. Parser has 7 ops (`Op =` in `plan.go`); the ADR-054 randomised test keeps the five line-range ops (path ops have no delimiter nets). Classes here: unlink, rename (incl. nested dest), dest-exists refuse, unread sibling restore, mix-on-same-path refuse, unlink-dest-then-rename-onto-it, two unlinks. Named extras: `TestUnlinkThenRenameOntoFreedDest`, `TestPathOpMixedWithLineEditOnSamePathIsRefused`, `TestUnlinkWithABodyIsRefused`.
- Arm 3: `TestRandomisedPathOpWriteMatrix` seeds 54/7/13 × 80 through the built binary (skips Windows). `scripts/break-campaign.sh` §10: eight probes, exits 0/1/2 as the Decision says. Dry-run of a refusing plan is exit 1, not 0.

Hand mutants against that suite (v3: baseline green, compile-check, FAIL vs runner error, restore by inverse edit — not `git checkout`):

| Mutant | Result |
|---|---|
| dest-exists: `&& false && !unlinked[dest]` | killed |
| mix: `if false && pathLevel && (lineLevel \|\| len(hs) != 1)` | killed |
| unlink never moves aside (`os.Rename(w.full, name)` → `os.Rename(name, name)`) | killed |
| rename never moves (`os.Rename(w.full, w.renameTo)` → `os.Rename(w.full, w.full)`) | killed |
| `seen.Drop` no-op | killed |
| receipt verb `"removed"` → `"wrote"` | killed |
| `compilePathOp("unlink"` → `"delete"` | killed |
| `compilePathOp("rename"` → `"unlink"` | killed |
| parser `OpUnlink && false && len(h.Body)` | killed |
| `false && err := os.Rename(...)` | inconclusive (does not compile) |
| apply-layer unlink-body gate | survived — parser already refuses; fixtures never reach apply |
| `commitPathOps` restore() no-op | survived — sibling fail never starts commit (content writes first); restore is FS-error-after-first-aside only |
| first `Move to with hunks` refuse inverted | survived — the test hits a later refuse site |

No product defect found. The generator is the first hypothesis for a failure; dumps carry seed, iter, kind.
