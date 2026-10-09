# ADR-135: a walk counts the discovered names Windows will not keep

**Status:** Accepted
**Accepted:** 2026-10-09 by Zy — "do adr first, fix, test, then release" (the answer to the open decisions after the Windows chaos round, which listed this key) and "/loop continue delivering, end to end, no dead code"
**Date:** 2026-10-09
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-071, ADR-081, ADR-111, ADR-116, ADR-130, docs/adr/BACKLOG.md
**Invalidates:** None — it adds a count where ADR-007 rule 2 skipped in silence; the rule's reason (a pattern oracle) does not apply to a name
**Governs:** `internal/rooted/rooted.go`, `internal/read/walk.go`, `docs/receipts.txt`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/read/unkeepable135_test.go::TestAWalkCountsTheNamesWindowsWillNotKeep`
**Served-path change:** on a Windows build, a `--grep` walk (CLI and `mrw_read`) that discovers a file whose name Windows will not keep — a reserved device name such as `aux.txt`, `con`, `nul`, `com1`, `lpt1.md`, or a name ending in a dot or a space — serves the rest as before and says so on the `-- skipped:` line (`N file(s) with a name Windows will not keep`); `mrw_read` carries the count as `skipped.unkeepable`. Exit codes and what is served do not change, and nothing changes off Windows, where `Resolve` refuses no name.

## Context

A `--grep` walk sends every discovered file through `rooted.Resolve` and drops a refusal in silence (ADR-007 rule 2): reporting it would turn the problem list into a pattern oracle, because a refusal that names a path only when its content matched says what the content held. ADR-116 then said nothing is skipped silently, and counted what the ignore rules and the binary test drop.

**What was observed** (2026-10-09, the Windows chaos round on v1.52.0, four of five sessions): `aux.txt`, `nul`, `com1`, `con.txt`, `trail.`, `trail ` and `lpt1.md`, made through `\\?\` paths, each held a matching line. The walk served the rest and exited 0, and `-- skipped:` said nothing, while it counts binaries, ignored files and nested repositories. A caller whose grep returned fewer files than exist has no line telling it why. Named directly, each is refused honestly.

**Why a count reveals nothing the walk did not already list.** The oracle ADR-007 shut is content: a path printed only when it matched. A count of refused NAMES is taken from the directory listing before any byte is read, whether or not the file would have matched, so it says that a name exists, not what the file holds. Resolve refuses these names by their spelling alone (`ErrDeviceName`, the Win32 alias test), never by reading.

**Audit of the class** — *a discovered path `Resolve` refuses by its name*: `mrw read --grep 'ErrDeviceName|win32Alias|aliasRefusal' --exclude '*_test.go' internal/` names the device-name refusal (`deviceName`, `internal/rooted/rooted.go`) and the Win32 alias refusal (`aliasRefusal`); both are reached by the walk's `Resolver.Resolve`, and both sit behind `followLinks`, so only a Windows build makes them. Two refusals. The other refusals Resolve gives a discovered path (outside the root, mrw's own state, a hard link to it) are not about the name and stay silent.

## Existing Primitives Audit

- **`rooted.ErrDeviceName`** (ADR-081) — already an error of its own, so the walk can tell a device name from an escape.
- **`WalkSkipped` and `SkipNote`** (ADR-116, ADR-130) — one struct, one sentence, shared by the CLI and `mrw_read`; a new key is one field and one clause.
- **ADR-111** — a receipt key is only added, and listed in `docs/receipts.txt` in the change that adds it.

## Decision

1. **`rooted` tells a name refusal from the others.** `rooted.UnkeepableName(err)` reports whether `err` is Resolve's refusal of a name Windows will not keep: a reserved device name (`ErrDeviceName`), or a name Windows would not keep as written (a new `ErrWin32Alias`, carried by `aliasRefusal`'s unchanged text). The messages do not change.
2. **A discovered path so refused is counted, not dropped.** The walk records it in a set; `WalkSkipped.Unkeepable` counts the set (`json:"unkeepable"`), a file another path served after all is not counted, and the file is still not served. A path the caller NAMES is refused by name as before.
3. **`SkipNote` says it.** `N file(s) with a name Windows will not keep`, in the existing `-- skipped:` sentence. The sentence's tail says the flag walks them; `--no-ignore` does not walk a name Windows will not keep, so when this is the only thing skipped the tail reads `name one to be told why`, and beside other counts it adds that the flag walks the others.
4. **`mrw_read` carries `skipped.unkeepable`**, appended to `docs/receipts.txt` (ADR-111).

## Alternatives Considered

- **Report each name as a REFUSED line** — rejected: it prints a path only for files the walk reached, but ADR-007's concern is a refusal printed for a file that matched; the count gives the signal that matters (the grep returned fewer files than exist) with no path to chase and no per-file oracle.
- **Count every silent drop in one key** — rejected: the other refusals (outside the root, a symlink out) are exactly the paths whose existence outside the root a caller should not learn; this key is limited to refusals made by spelling alone.
- **Leave it silent** — rejected: four sessions measured a grep that returned fewer files than exist with nothing saying why, which ADR-116 exists to prevent.

## Component / Boundary Impact

`internal/rooted` (one exported function and one sentinel) and `internal/read` (one field, one set, one clause). `apply`, `plan`, `check`, `seen` and `state` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `read.WalkSkipped.Unkeepable`, `SkipNote` | the count and its clause | T1 | CLI `--grep`, `mrw_read` |
| `docs/receipts.txt` | `mcp_read skipped.unkeepable` | T1 | receipt readers (ADR-111) |
| `AGENTS.md`, `README.md` | the `-- skipped:` paragraph names it | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a grep that returns fewer files than exist says why; the BACKLOG item "a grep walk drops, without a line, a discovered file whose name Windows will not keep" closes.
- **Negative:** none off Windows. On Windows a tree holding `aux.txt` now prints a `-- skipped:` line where it printed none.
- **Neutral:** nothing served and no exit code changes.

## Out of Scope

- `--ast-grep` hits with such a name (deferred: docs/adr/BACKLOG.md — the hit judge drops by the walk's rules and ast-grep names the files; a Windows caller reports it)
- A link to a directory met inside a walk, also skipped uncounted (deferred: docs/adr/BACKLOG.md — the same chaos round named it; it is not about a name)
- A refusal that is not about a name: outside the root, mrw's own state (permanent: boundary: these are the paths whose existence a caller should not learn from a count, ADR-007 rule 2)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a refusal that is not a name is counted as one | Low | a count that misleads | `UnkeepableName` accepts only the two sentinels; `TestAnEscapeIsNotCountedAsAName` pairs an escaping symlink named like a normal file with the device name |
| the count becomes an oracle | Low | a name's existence is learned | it counts names the listing already showed, before any read, whether or not they matched |

## Rollback

Revert T1: the names are dropped in silence again. No persistent state changes shape; a receipt reader that ignores an unknown key is unaffected (ADR-111).

## Follow-ups

- None — the record carries no open follow-up beyond the deferred items above.
