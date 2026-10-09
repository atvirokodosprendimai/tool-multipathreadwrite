# ADR-134: a hard link to mrw's own state is not served

**Status:** Accepted
**Accepted:** 2026-10-09 by Zy — "do adr first, fix, test, then release", the answer to the three open decisions after the Windows chaos round (hard-link ledger record, skipped-names key, cut a release)
**Date:** 2026-10-09
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-077, ADR-123, ADR-131, ADR-007, ADR-002, docs/adr/BACKLOG.md
**Invalidates:** None — it closes a gap in ADR-077's promise; no accepted clause says a hard link to the ledger is served
**Governs:** `internal/rooted/rooted.go`, `internal/rooted/hardlink.go`, `internal/rooted/linkcount_unix.go`, `internal/rooted/linkcount_windows.go`, `scripts/contract.sh`, `AGENTS.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/rooted/hardlink134_test.go::TestAHardLinkToMrwsStateIsRefused`
**Served-path change:** a regular file inside the root that is the same file as a file under mrw's state base (a hard link to a ledger, an ack store, a lock) is refused on every surface that ADR-077 refuses the state itself — a read, a plan, a `check` path, a working-set entry — naming mrw's own state; a `--grep` drops it as it drops any path the boundary refuses. A hard link to an ordinary file is served as before.

## Context

ADR-077 promises that mrw's own state is never served as the caller's file: a read of the ledger would let a caller see what licenses a write, and a plan could edit it. The boundary compares a path, and each of its ancestors, with the state BASE directory by identity (`inStateAt`, `internal/rooted/rooted.go`). It never compares the served FILE with the files inside the base.

**What was observed** (2026-10-09, the Windows chaos round on v1.52.0): three Windows sessions (quality-harness, quality-blueprints, wcag-web) made a hard link `hl.txt` in the root to `<state>/mrw/<key>/seen`. `mrw read hl.txt` served the whole ledger (`#mrw-seen v4` and the sha lines) and `--grep` matched it, while the direct state path was refused "inside mrw's own state directory". Reproduced on macOS the same day with v1.52.0 in a scratch root: after `mrw read f.txt`, `ln <state>/mrw/<key>/seen hl.txt`, then `mrw read hl.txt` exits 0 and prints the ledger, and `mrw read --grep mrw-seen .` prints its first line.

**Bound.** Making the link needs write access to the state directory and to the root. The ledger is saved by rename (`state.Write`), so after the next save the tree copy is the OLD inode: a stale snapshot, no longer the live file, and a write through the link lands on the snapshot only. Until that save the link IS the live ledger.

**Audit of the class** — *a path by which a served file can be mrw's state file*: `mrw read --grep 'inStateAt|inState\(|rooted\.InState' --exclude '*_test.go' .` names three sites in `internal/rooted/rooted.go` (`resolveIn`, `Resolver.dir`, `InState`) and two callers of `InState` in `cmd/mrw/main.go` (1190, 2580), which judge a command-line path and then pass it to `Resolve`. Of those, only `resolveIn` and `Resolver.Resolve` judge a file; `Resolver.dir` judges a directory, which a hard link cannot be (no filesystem links directories). Two sites.

## Existing Primitives Audit

- **`rooted.Resolve` / `Resolver.Resolve`** (ADR-006, ADR-131) — the one boundary every surface calls; the refusal belongs there, not in each walker.
- **`inStateAt`** (ADR-077) — judges a path by identity against the base; this record adds the file-level counterpart beside it.
- **`os.SameFile`** — compares two file infos by volume and file index; no new dependency.
- **The `Lstat` the `Resolver` already takes** (ADR-131) — on unix it carries the link count, so the pre-test costs no syscall.
- **`state.Base`** — names the base without making it.

## Decision

1. **A regular file that is the same file as a regular file under the state base is refused**, after the root and state-directory checks, by `rooted.Resolve` and by `Resolver.Resolve`: "`<path>` is a hard link to a file in mrw's own state directory; mrw does not serve or edit its own ledger". It judges the REAL path's file, so a symlink to a hard link is refused too.
2. **A link count of one is the pre-test.** A file no other name leads to cannot be a state file's second name. On unix the count is `Stat_t.Nlink` of the `Lstat` already taken; on Windows it is `NumberOfLinks` of an open handle (`linkcount_windows.go`). A count that cannot be read counts as "may be linked": the comparison runs. Only a file whose count is above one is compared.
3. **The comparison is against the base, not one root's directory.** On first need the state base is scanned for regular files whose own link count is above one — only those can have a second name — and a candidate is compared with each by `os.SameFile`. The scan runs once per `Resolver` (one walk) and once per `Resolve` call; a base that does not exist holds nothing to link to and costs nothing. A path or an ancestor that is inside the base is refused earlier, as before.
4. **A hard link to an ordinary file is served.** A pnpm store, a Go build cache and a package manager's deduplicated tree are full of link counts above one; only identity with a state file refuses.

## Alternatives Considered

- **Compare only with this root's state directory** — rejected: another root's ledger is mrw's own state too, and a link to it is the same breach; the scan is lazy, so the wider reach costs a pnpm tree nothing.
- **Refuse every file with a link count above one** — rejected: it would refuse a deduplicated `node_modules` wholesale, to close a gap one file wide.
- **Make the ledger unreadable to a link (mode 0600 and a different owner)** — rejected: the link is made by the same user who owns both ends, so a mode stops nothing.
- **Leave it (making the link needs write access to both places)** — rejected: the break is reproduced on two platforms by three sessions, the promise is mrw's own, and the cost is one count per served file.

## Component / Boundary Impact

`internal/rooted` only: `rooted.go` gains the comparison; two platform files read the link count. `internal/state`, `read`, `apply` and the other engine packages stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `rooted.Resolve`, `Resolver.Resolve` | refuse a hard link to a state file | T1 | read, apply, check, mcp, `--grep`, `--ast-grep` (through the boundary) |
| `scripts/contract.sh` | §236 | T1 | CI Linux |
| `AGENTS.md` | ADR-077 paragraph names the hard link | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** "mrw's own state is never served" holds for a second name of a state file, on every platform.
- **Negative:** one link count per served file (free on unix; an open on Windows, measured below), and one scan of the state base per walk that meets a file with a link count above one.
- **Neutral:** the refusal is a REFUSED line on a named read and a silent drop in a walk, as for any path the boundary refuses (ADR-007 rule 3).

## Out of Scope

- A COPY of a state file, and a hard link whose original was replaced by a later save (permanent: boundary: a copy holds a past ledger, not mrw's live state; the ledger is saved by rename, so a link made before a save is a snapshot afterwards and has the identity of nothing)
- A symlink to a state file (permanent: fact: already refused by ADR-077 — the real path is judged; citation: file `internal/rooted/rooted.go:171`)
- A state file reached by a name the caller made in ANOTHER filesystem (permanent: boundary: a hard link cannot cross volumes, so the link count and identity never match; nothing to refuse)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the Windows link-count open slows a `--grep` walk | Medium | the 25x walk speed-up of ADR-131 is partly given back | measured on windows-latest before merge with the ADR-123/131 timing workflow; the record is amended with the numbers, and the Windows pre-test is narrowed by volume if the walk is more than 1.3 times slower |
| the scan of a large state base (22,836 directories on one machine, 2026-09-07) is slow | Low | one slow walk | lazy: it runs only for a file with a link count above one, once per walk, and only files whose own link count is above one are kept |
| a link count that cannot be read hides a link | Low | the breach stays | an unreadable count counts as "may be linked", so the comparison runs |

## Rollback

Revert T1: a hard link to a state file is served again. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up.
