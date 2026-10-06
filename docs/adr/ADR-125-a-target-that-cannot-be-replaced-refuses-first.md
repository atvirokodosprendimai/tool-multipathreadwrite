# ADR-125: a target that cannot be replaced refuses before any rename

**Status:** Accepted
**Accepted:** 2026-10-06 by Zy — "Probe + POSIX rename (Recommended)", on the Windows peers' finding that a held target turns a plan PARTIALLY APPLIED by plan order (BACKLOG "From the Windows peers"), and "plan these fixes then". The record's text was drafted after that answer.
**Date:** 2026-10-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-066, ADR-086, ADR-106, ADR-107, ADR-109
**Invalidates:** None — ADR-066's PARTIALLY APPLIED stays the honest answer for a failure inside the commit; this record moves the common Windows cause ahead of it
**Governs:** `internal/apply/apply.go`, `internal/apply/tree.go`, `internal/apply/replace*.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/apply/replace125_test.go::TestAnUnreadableTargetGetsAReceipt`
**Served-path change:** a plan naming a file mrw cannot read — permission denied, held exclusively by another process, a name the system refuses — exits 1 with a receipt whose failed hunk names the cause, instead of a bare `mrw: …` line at exit 2; on Windows a target another process holds without delete sharing fails before any rename — exit 2, NOTHING WRITTEN, the hunk naming that it is held — rather than after the files before it have landed.

## Context

Five Windows peers probed v1.42.0 on 2026-10-02 (BACKLOG "From the Windows peers"):

- A file another process holds open without `FILE_SHARE_DELETE` makes the commit's rename fail ("Access is denied"). Held last or in the middle, the earlier files have landed: PARTIALLY APPLIED, exit 2; held first, NOTHING WRITTEN. A peer probe found that an open asking for `DELETE` access fails early (err 32) for every such holder, and that with `ReadWrite,Delete` sharing the probe passes while `os.Rename` (MoveFileEx with replace) still fails (err 5).
- An exclusively held file, or an invalid name (`q?.txt`), printed one bare `mrw: …` line, exit 2, no verdicts. Source: `apply.go:573-575` returns the load error before `res.Hunks` is filled (675), so `main.go:1340` prints no receipt. The same path takes a unix file whose mode denies reading.
- A file-level ACL deny gave "its identity could not be read … send the plan again" (`apply.go:507-508`): on Windows `os.SameFile` loads the file ID with an open whose error it swallows.

**Audit of the class** — *a step of `Apply` that opens or renames an existing target and can fail for a reason the caller cannot see*: `mrw read --grep 'loadFn\(|os.SameFile\(info, info\)|commitRenameFn\(' internal/apply/apply.go internal/apply/pathop.go` — the load (556), the identity check (507), and the commit renames (`apply.go:1003`, `pathop.go` unlink, rename and undo, all through `commitRenameFn`). Staging writes new files only and is not in the class.

## Existing Primitives Audit

- **`refuseFile`** — the per-hunk refusal every validation failure uses; the load error and the identity refusal join it.
- **`abortStage`** — the before-any-rename abort (NOTHING WRITTEN, a full receipt); the probe fails through it.
- **`probeNameFn`** (ADR-086) — the precedent for asking the filesystem before anything is written.
- **`commitRenameFn` / `(*tree).rename`** — the one seam every commit rename passes; the Windows rename is made there.
- **`os.Root.Rename`** (ADR-106) — the commit's rename; on Windows (Go 1.27, `internal/syscall/windows/at_windows.go` `Renameat`) it already renames by handle with `FILE_RENAME_REPLACE_IF_EXISTS | FILE_RENAME_POSIX_SEMANTICS` and falls back where a volume refuses the class, so a holder that shares delete does not block it. Found while drafting; the record keeps it rather than rebuilding it.
- **`syscall.CreateFile`** — the probe's open, in the standard library, so `go.mod` keeps one requirement.

## Decision

1. **A target mrw cannot open, for a cause it can name, is refused on its hunk.** A load error at `apply.go:573` whose cause is a permission, another process holding the file, or a name the system refuses, and a false `SameFile(info, info)`, become `refuseFile` refusals: the file's first hunk fails, its others and every sibling skip, nothing is written, exit 1. The reason names the cause — `permission denied`, `held open by another process`, `not a valid name on this system` — beside the system's error; "send the plan again" stays only when nothing explains the failure. Any other load error — a directory named in a plan, an I/O failure — stays the exit-2 failure it was, which `TestAPlanRefusedAfterItParsedIsOneRefusal` (ADR-083) pins.
2. **On Windows every existing target is asked whether it can be replaced before the first rename**: its entry is opened for `DELETE` with full sharing, relative to its parent directory's handle taken through the root (`NtCreateFile` with a root directory), and closed at once. A sharing violation or an access denial fails that file through `abortStage` — exit 2, NOTHING WRITTEN, a full receipt naming the cause. Content targets, unlink sources and rename sources are asked. On unix the probe is nothing: a rename does not care who holds the file. Opening relative to a confined handle keeps the probe inside the root when a parent is swapped for a junction, and leaves no `MAX_PATH` limit (the Codex review of #338).
3. **The commit rename stays `os.Root.Rename`**, which on Windows already replaces with POSIX semantics (Go's `Renameat`); T2 pins that with a holder that shares delete, so a future toolchain that drops it fails a test rather than a user.
4. **The window stays and is said.** A file taken between the probe and its rename still fails inside the commit, and ADR-066's PARTIALLY APPLIED reports what landed. What changes is that the common Windows cause is caught before anything is written.

## Alternatives Considered

- **Better reporting only** — rejected by Zy's answer: the order-dependent partial write would stay the ordinary Windows outcome.
- **Probe only, without POSIX semantics** — moot: Go's `Root.Rename` already uses them; what Zy's "Probe + POSIX rename" asked for is the probe plus a test that keeps the POSIX rename true.
- **Retry the rename** — rejected: an editor or a language server holds a file for as long as it is open; a retry turns a refusal into a hang.

## Component / Boundary Impact

Owns `internal/apply` (engine). `os.Root` confinement stays: the probe opens its leaf relative to a parent handle taken through the root, and the rename goes through the root. `ntdll` is reached through `syscall.NewLazyDLL`, so `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| a write naming a file mrw cannot open | exit 1 and a receipt, not a bare exit 2 | T1 | CLI, MCP |
| the identity refusal | names its cause | T1 | CLI, MCP |
| a write over a held file on Windows | NOTHING WRITTEN before any rename; a delete-sharing holder does not block (Go's rename, pinned by a test) | T2 | CLI, MCP |
| `scripts/contract.sh` | §223 | T1 | CI Linux |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `causeOf` names a cause from an open error | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2.

## Consequences

- **Positive:** the commonest Windows partial write becomes a refusal with nothing written; every unopenable target gets a receipt that says why.
- **Negative:** one extra open per existing target on Windows; a held file now refuses the whole plan where before a plan holding it last landed its other files.
- **Neutral:** unix renames are unchanged.

## Out of Scope

- A case-only rename, the write lock during a check, and the MCP refusal texts (deferred: docs/adr/BACKLOG.md "From the Windows peers" — taken by later records of the 2026-10-06 plan)
- Waiting for a holder to let go (permanent: boundary: a holder can keep a file for as long as it is open; mrw refuses and names it rather than hang)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a Go release changes `Root.Rename` on Windows | Low | Medium | `TestAHolderThatSharesDeleteDoesNotBlockTheCommit` on the windows shards |
| a probe's open itself blocks another process for its instant | Low | Low | full sharing on the probe, and it is closed at once |
| a file taken after the probe | Low | Medium | ADR-066's PARTIALLY APPLIED, unchanged |

## Rollback

Revert the tasks: the load error is bare again and Windows renames go through `os.Root.Rename`.

## Follow-ups

- None — the record carries no open follow-up.
