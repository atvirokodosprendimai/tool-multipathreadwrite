# ADR-081: A Windows device name is refused by name

**Status:** Accepted
**Accepted:** 2026-09-26 by M — chose "Refuse by name" when asked how v1.27.1 should treat reserved device names after the Windows peer's run of v1.27.0
**Date:** 2026-09-26
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-006, ADR-071, ADR-076, docs/adr/BACKLOG.md
**Governs:** `internal/rooted/rooted.go`, `internal/rooted/links_windows.go`, `internal/rooted/links_other.go`, `internal/rooted/links.go`, `internal/read/read.go`, `internal/read/astgrep.go`, `internal/apply/apply.go`, `internal/apply/encoding_test.go`, `internal/rooted/paths076_test.go`, `cmd/mrw/device_windows_test.go`, `AGENTS.md`, `docs/adr/ADR-076-a-path-means-what-it-says.md`
**Enforced-by:** `cmd/mrw/device_windows_test.go::TestADeviceNameIsRefusedNotReadAsAnEmptyFile`
**Invalidates:** ADR-076 Decision 3's *"`opensDevice` asks the OS"*
**Served-path change:** on Windows a spec, a plan path or a rename destination any component of which, below the root — or the target of a link it passes through — is a reserved device name — `CON`, `PRN`, `AUX`, `NUL`, `COM1`–`9`, `LPT1`–`9`, `CONIN$`, `CONOUT$`, in any case, with or without an extension — is refused by name, on every Windows build; an ast-grep hit on one is dropped as the walk drops one, and the refusal carries no `--root` advice; before, only a last component `GetFullPathName` mapped to a device was refused.

## Context

**What was observed** (a Windows peer, `desktop-3laqmbq-declarative-pie`, against the v1.27.0 asset on
Windows 11 Pro 10.0.26200, 2026-09-26). `read NUL` and `@@ NUL - create` were refused, but
`@@ nul.txt 0 create`, `@@ con - create` and `@@ COM1.txt - create` applied at exit 0. The file `con`
then could not be removed by mrw — `@@ con - unlink` said "con does not exist" while `ls` listed it —
and PowerShell 5's `Remove-Item -LiteralPath` and `Get-Item` could not find `COM1.txt` or `nul.txt`,
while `cmd /c del` removed them. On that build `GetFullPathName`, which ADR-076 asked, no longer maps
these names inside a directory, while other Windows APIs still treat them as devices: mrw made files
that it, and other tools, cannot reach.

**And the reviews of #243** (Codex and in-process) found the first cut incomplete: a symlink inside the
root that led to `con.txt` passed, since the name was checked before the link was followed; an
ast-grep hit's CR-only probe read the file before the boundary saw it; a reserved name as a directory
component (`con/a.txt`) was still created; the refusal ended with advice about `--root` that no root
fixes; and the Windows CI shard refused this repository's own fixture `nul.bin`.

## Existing Primitives Audit

- `win32Device` already picks every candidate by Go's own rule (`internal/filepathlite`
  `isReservedName`), with a table test that runs on every platform.
- `opensDevice` was its only other half, and has no other caller.

## Decision

1. On Windows, a path any component of which, below the root, `win32Device` names is refused at the
   boundary (`rooted.Resolve`, ADR-006), whatever this build's `GetFullPathName` says; so is a path
   whose link target, below the root, holds one. `opensDevice` is removed. The root's own
   components are never judged.
2. The refusal wraps `rooted.ErrDeviceName`, so `read` and `apply` print it without the advice that
   fits an escape from the root.
3. Every ast-grep hit passes `rooted.Resolve` before anything opens it: a discovered hit it refuses
   is dropped, as the walk drops one (ADR-007 rule 2).
4. AGENTS.md says so, and ADR-076 carries a pointer to this record.

## Alternatives Considered

- **Keep asking the OS, and make unlink and read reach what create made.** Rejected by M: which API
  a later tool uses decides whether it can reach the file, so mrw would still make files others
  cannot, and it needs Windows debugging through a peer to find each path.
- **Refuse only on create and rename.** Rejected by M: a name mrw may read and edit but not create is
  a rule no caller can predict.

## Component / Boundary Impact

`internal/rooted` only. On other platforms nothing changes: a POSIX name opens no device by being
spelled like one.

## Wiring & Contract Changes

A new refusal on Windows. No contract row: `scripts/contract.sh` runs on Linux only (AGENTS.md); the
Windows CI shard runs the test named in Enforced-by, which drives the command in-process.

## Inter-task Contracts

None.

## Implementation

See `tasks/`.

## Consequences

- On Windows an existing file named `con.txt` or `aux.go` in a repository cannot be read or edited
  through mrw; it had to be addressed by another tool anyway on any build that reserves the name.

## Out of Scope

- A device name among the root's own components (permanent: boundary: the root is the caller's choice; only what lies below it is judged)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A repository that tracks `aux.go`, `con.txt` or `nul.bin` is edited on Windows | Medium — this repository's own `nul.bin` fixture met it on the first Windows run | Low | the refusal names the file and the reason; the fixture is `nulbyte.bin` now |

## Rollback

Revert the task.

## Follow-ups

- [x] Release as v1.27.1, and ask the Windows peer to re-run its item 2 — tagged at `6d35d58` (#243), Status #244; the peer confirmed every name refused (BACKLOG).
