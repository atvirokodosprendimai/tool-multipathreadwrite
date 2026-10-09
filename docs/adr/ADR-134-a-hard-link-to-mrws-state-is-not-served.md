# ADR-134: a hard link to mrw's own state is not served

**Status:** Accepted
**Accepted:** 2026-10-09 by Zy — "do adr first, fix, test, then release", the answer to the three open decisions after the Windows chaos round (hard-link ledger record, skipped-names key, cut a release)
**Date:** 2026-10-09
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-077, ADR-123, ADR-131, ADR-007, ADR-002, docs/adr/BACKLOG.md
**Invalidates:** None — it closes a gap in ADR-077's promise; no accepted clause says a hard link to the ledger is served
**Governs:** `internal/rooted/rooted.go`, `internal/rooted/hardlink.go`, `internal/rooted/identity_unix.go`, `internal/rooted/identity_windows.go`, `internal/state/state.go`, `scripts/contract.sh`, `AGENTS.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/rooted/hardlink134_test.go::TestAHardLinkToMrwsStateIsRefused`
**Served-path change:** a regular file inside the root that is the same file as a file in THIS checkout's state directory (a hard link to its ledger, ack store or lock) is refused on every surface that ADR-077 refuses the state itself — a read, a plan, a `check` path, a working-set entry — naming mrw's own state; a `--grep` drops it as it drops any path the boundary refuses. A file with a second name that cannot be compared to the end is refused saying so. A hard link to an ordinary file is served as before, and so is a link to another checkout's ledger.

## Context

ADR-077 promises that mrw's own state is never served as the caller's file: a read of the ledger would let a caller see what licenses a write, and a plan could edit it. The boundary compares a path, and each of its ancestors, with the state BASE directory by identity (`inStateAt`, `internal/rooted/rooted.go`). It never compares the served FILE with the files inside the base.

**What was observed** (2026-10-09, the Windows chaos round on v1.52.0): three Windows sessions (quality-harness, quality-blueprints, wcag-web) made a hard link `hl.txt` in the root to `<state>/mrw/<key>/seen`. `mrw read hl.txt` served the whole ledger (`#mrw-seen v4` and the sha lines) and `--grep` matched it, while the direct state path was refused "inside mrw's own state directory". Reproduced on macOS the same day with v1.52.0 in a scratch root: after `mrw read f.txt`, `ln <state>/mrw/<key>/seen hl.txt`, then `mrw read hl.txt` exits 0 and prints the ledger, and `mrw read --grep mrw-seen .` prints its first line.

**Bound.** Making the link needs write access to the state directory and to the root. The ledger is saved by rename (`state.Write`), so after the next save the tree copy is the OLD inode: a stale snapshot, no longer the live file, and a write through the link lands on the snapshot only. Until that save the link IS the live ledger.

**Audit of the class** — *a path by which a served file can be mrw's state file*: `mrw read --grep 'inStateAt|inState\(|rooted\.InState' --exclude '*_test.go' .` names three sites in `internal/rooted/rooted.go` (`resolveIn`, `Resolver.dir`, `InState`) and two callers of `InState` in `cmd/mrw/main.go` (1190, 2580), which judge a command-line path and then pass it to `Resolve`. Of those, only `resolveIn` and `Resolver.Resolve` judge a file; `Resolver.dir` judges a directory, which a hard link cannot be (no filesystem links directories). Two sites.

**What the first draft got wrong** (the Codex review of #361, and a measurement of 2026-10-09). The first draft scanned every checkout's directory under the state base, compared with `os.SameFile`, and treated a directory it could not list as "no match". Four findings held. (1) An incomplete scan answered "not a state file" for the part it could not read, so an unlistable directory removed a ledger from the comparison and left its alias served. (2) On Windows `os.SameFile` opens with share mode 0 (Go 1.26.9, `os/types_windows.go`, `loadFileId`) and answers false when another process holds the file, so a ledger held open was served by its alias. (3) `read.Run` resolves every matching path afresh, so a grep matching K hard-linked files scanned the base K times: measured on macOS, 6,000 ordinary hard-linked files and a base of 500 other checkouts' directories, 0.54-0.75 s on v1.52.0, 62-77 s on the first draft. (4) A scan kept for one walk does not see a link made to a state file during that walk; that one is a bound, below, not a defect.

## Existing Primitives Audit

- **`rooted.Resolve` / `Resolver.Resolve`** (ADR-006, ADR-131) — the one boundary every surface calls; the refusal belongs there, not in each walker.
- **`inStateAt`** (ADR-077) — judges a path by identity against the base; this record adds the file-level counterpart beside it.
- **`GetFileInformationByHandle` and `Stat_t`** — the volume, the file index and the number of names, on Windows from a handle opened for its attributes only and shared with every opener, elsewhere from the `Lstat` already taken. `os.SameFile` is not used: it opens exclusively on Windows and answers false for a held file.
- **`state.Base`** — names the base without making it; **`state.DirPath`** (new, beside `Dir`) names a checkout's state directory without making it or writing its marker.
- **The `Lstat` the `Resolver` already takes** (ADR-131) — on unix it carries the identity and the count, so the pre-test costs no syscall.

## Decision

1. **A regular file that is the same file as a file in this checkout's state directory is refused**, after the root and state-directory checks, by `rooted.Resolve` and by `Resolver.Resolve`: "`<path>` is a hard link to a file in mrw's own state directory; mrw does not serve or edit its own ledger". It judges the REAL path's file, so a symlink to a hard link is refused too. "The same file" is the volume and the file on it, read from the file system, not from a path.
2. **A link count of one is the pre-test.** A file no other name leads to cannot be a state file's second name. On unix the identity and the count come from the `Lstat` already taken; on Windows from one handle opened for the file's attributes only (`FILE_READ_ATTRIBUTES`, shared read, write and delete), so a file another process holds open is still read. Only a file whose count is above one is compared.
3. **The comparison is with this checkout's state directory** (`state.DirPath`), the files that license a write here. On first need the directory is listed for files whose own link count is above one — only those can have a second name — and their identities are kept for one walk (`Resolver`) or one `Resolve` call. A name that vanishes while the directory is listed (a ledger saved by rename) is passed over. A checkout with no state directory, or a machine with no state base, holds nothing to link to and costs nothing.
4. **A comparison that cannot be completed refuses.** A file with a second name whose identity cannot be read, or whose state directory cannot be listed or examined to the end, is refused: "`<path>` has more than one name, and mrw cannot tell whether one is a file in its own state directory". A file with one name is never held up by it.
5. **A hard link to an ordinary file is served.** A pnpm store, a Go build cache and a package manager's deduplicated tree are full of link counts above one; only identity with a state file refuses them.

## Alternatives Considered

- **Compare with every checkout's directory under the base** — built first, rejected on measurement: `read.Run` resolves each served path afresh, so a grep matching K hard-linked files scanned the base K times (62-77 s against 0.54-0.75 s for 6,000 files over a base of 500 directories, macOS, 2026-10-09). Another checkout's ledger licenses no write here, so the files that matter are this checkout's own.
- **Refuse every file with a link count above one** — rejected: it would refuse a deduplicated `node_modules` wholesale, to close a gap one file wide.
- **Make the ledger unreadable to a link (mode 0600 and a different owner)** — rejected: the link is made by the same user who owns both ends, so a mode stops nothing.
- **Leave it (making the link needs write access to both places)** — rejected: the break is reproduced on two platforms by three sessions, the promise is mrw's own, and the cost is one count per served file.

## Component / Boundary Impact

`internal/rooted` and one function in `internal/state`: `rooted.go` gains the comparison, `hardlink.go` holds it, two platform files read the identity; `state.DirPath` names a directory `Dir` would make. `read`, `apply` and the other engine packages stay byte-identical; `go.mod` keeps one requirement.

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
- **Negative:** one identity read per served file (free on unix; an open on Windows, measured below), one listing of this checkout's state directory per walk or `Resolve` call that meets a file with a link count above one, and a refusal of any file with a second name while the state directory cannot be listed. Measured 2026-10-09 on macOS: a grep matching 6,000 ordinary hard-linked files over a base of 500 other directories took 0.89-1.34 s against 0.54-0.75 s on v1.52.0, output identical.
- **Neutral:** the refusal is a REFUSED line on a named read and a silent drop in a walk, as for any path the boundary refuses (ADR-007 rule 3).

## Out of Scope

- A COPY of a state file, and a hard link whose original was replaced by a later save (permanent: boundary: a copy holds a past ledger, not mrw's live state; the ledger is saved by rename, so a link made before a save is a snapshot afterwards and has the identity of nothing)
- A symlink to a state file (permanent: fact: already refused by ADR-077 — the real path is judged; citation: file `internal/rooted/rooted.go:171`)
- A state file reached by a name the caller made in ANOTHER filesystem (permanent: boundary: a hard link cannot cross volumes, so the identity never matches; nothing to refuse)
- A hard link to ANOTHER checkout's ledger (permanent: boundary: it licenses no write in this checkout, whose ledger is keyed by its own path; a plan that writes the alias replaces the checkout-local name, because a commit renames a staged file over it, and leaves the other ledger as it was; what a read shows is another checkout's paths and shas. Comparing with every checkout's directory cost 62-77 s on a 6,000-file grep, Alternatives; `TestALinkToAnotherCheckoutsLedgerIsNotJudged` pins the decision)
- A link made to a state file DURING a walk (permanent: boundary: the comparison is read once per walk, the staleness ADR-131 accepts for a directory swapped mid-walk; `read.Run` resolves afresh before it serves, so no content is served, and only a caller racing the walk could learn that a pattern matched)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| the Windows identity open slows a `--grep` walk | Realised, within the bar | the 25x walk speed-up of ADR-131 is partly given back | measured 2026-10-09 on windows-latest (AMD EPYC 7763, Windows Server 2025, `timing-123` against v1.52.0, 3,000 files, median of 5). Final head d6c84ba, run 37980188383: 233 ms before, 297 ms after on the plain tree (1.27x, 21 µs a file); 234 ms before, 292 ms after through a junction root (1.25x); `BenchmarkWalkDeep131` 339 ms before (mean of 3), 409 ms after (1.21x). First draft 3a4d5da, run 37977126786: 1.23x, 1.30x, 1.25x. The bar was 1.3x; the cost is the handle opened for each served file, and a reading above the bar would send the Windows pre-test to a volume test the record does not specify (T1 Stop Condition) |
| a state directory that cannot be listed refuses every file with a second name in the checkout | Low | a pnpm tree is refused while the directory is unlistable | mrw makes the directory itself, mode 0700; the refusal names the cause; the alternative, serving what could not be compared, is the breach |
| the identity of a held file cannot be read on Windows | Low | the file is refused as not comparable | the handle asks for attributes only and shares read, write and delete; `TestAHeldLedgerIsStillRefusedThroughItsAlias` holds the ledger open on windows-latest |

## Rollback

Revert T1: a hard link to a state file is served again. No persistent state changes shape.

## Follow-ups

- None — the record carries no open follow-up.
