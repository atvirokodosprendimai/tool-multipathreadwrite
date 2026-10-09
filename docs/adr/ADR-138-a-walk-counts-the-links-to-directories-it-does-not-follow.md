# ADR-138: a walk counts the links to directories it does not follow

**Status:** Accepted
**Accepted:** 2026-10-10 by Zy — "/loop continue delivering, end to end, no dead code"; the item was filed in BACKLOG from the 2026-10-09 Windows chaos round and named in ADR-135's Out of Scope
**Date:** 2026-10-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-096, ADR-116, ADR-135, ADR-111, docs/adr/BACKLOG.md
**Invalidates:** None — it counts what ADR-096 decision 2 already skips; nothing is walked differently
**Governs:** `internal/read/walk.go`, `internal/mcp/schema_test.go`, `docs/receipts.txt`, `scripts/contract.sh`, `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/read/linkeddirs138_test.go::TestAWalkCountsTheLinksToDirectoriesItDoesNotFollow`
**Served-path change:** a `--grep` walk (CLI and `mrw_read`) that meets a link to a directory inside the root says so on the `-- skipped:` line (`N link(s) to a directory, not followed`), and `mrw_read` carries the count as `skipped.linked_dirs`. Nothing is walked or served differently, and exit codes do not change.

## Context

ADR-096 decision 2: a walk follows no link to a directory, because a link can lead out of the tree, loop, or serve one file under two names. ADR-116: nothing a walk skips is skipped silently. The two met in 2026-10-09's Windows chaos round: a link (a symlink, or a junction on Windows) to a directory met inside a walk is dropped with no mention, while ignored files, binaries, nested repositories (ADR-130) and unkeepable names (ADR-135) are counted. A caller whose grep returned fewer files than exist, because a linked directory held the rest, is told nothing. Naming the link is refused with the directory to name instead (ADR-096); a walk gives no such pointer.

**Why a count reveals nothing new.** The walk already resolves the link's target to refuse one that leaves the root (`rooted.Resolve` on the discovered path, ADR-007 rule 3); only a link that resolves inside the root reaches the count. It says that a link to a directory exists at a name the listing showed, never what the directory holds.

**Audit of the class** — *a discovered path the walk drops because of what it IS, not what it holds*: `mrw read --grep 'discovered non-file|skipped in silence' --exclude '*_test.go' internal/read` names two sites in `walk.go`: the refusal by `Resolve` (counted for names by ADR-135, silent for an escape, which is the oracle ADR-007 rule 2 keeps shut) and the non-regular-file skip (`os.Stat` not a regular file). The second covers a link to a directory, a FIFO, a socket and a device. This record counts the first of those, the one a caller can act on; a FIFO or socket is not something a grep is expected to find.

## Existing Primitives Audit

- **`WalkSkipped` / `SkipNote`** (ADR-116, ADR-130, ADR-135) — one struct, one sentence, shared by the CLI and `mrw_read`; a new key is one field and one clause.
- **The `Lstat`/`Stat` pair** already in ADR-096's named-path check — a link is a path whose `Lstat` is not a directory and whose `Stat` is (Windows junctions included, as that comment says).
- **ADR-111** — a receipt key is only added, and listed in `docs/receipts.txt` in the change that adds it.

## Decision

1. **The walk counts a discovered path that resolves inside the root, is a link, and leads to a directory** (its `Lstat` is not a directory, its `Stat` is). The set is keyed by root-relative path; `WalkSkipped.LinkedDirs` counts it (`json:"linked_dirs"`); the link is still not followed and nothing under it is served.
2. **`SkipNote` says it**: `N link(s) to a directory, not followed`. The tail says the flag walks them for the ignore-class counts; no flag follows a link, so when only unfollowed or unkeepable paths were skipped the tail is `name one to be told why`, and beside the other counts it adds that naming one tells why the rest are refused. Naming a link to a directory is refused with the directory to name (ADR-096).
3. **`mrw_read` carries `skipped.linked_dirs`**, appended to `docs/receipts.txt` (ADR-111).
4. **A link that leaves the root, loops or dangles is not counted**: `Resolve` refuses it first and the refusal stays silent (ADR-007 rule 2), so the count never says where a link leads.

## Alternatives Considered

- **Follow the link** — rejected by ADR-096: escape, loop and double-serve.
- **Report each link as a REFUSED line** — rejected: it would print a path for a link whose target is not what the caller searched; the count with a pointer to name one is enough, and ADR-135 chose the same.
- **Count every non-regular discovered path** — rejected: a FIFO or a socket is not something a caller expects a grep to find, and the count would be noise.
- **Leave it** — rejected: ADR-116 promises nothing is skipped silently, and a linked directory is exactly where a caller's missing files go.

## Component / Boundary Impact

`internal/read` only (one field, one set, one clause) plus the read schema in `internal/mcp/schema_test.go` that the receipt test holds to a real answer. `apply`, `plan`, `rooted`, `seen`, `state` and `check` stay byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `read.WalkSkipped.LinkedDirs`, `SkipNote` | the count and its clause | T1 | CLI `--grep`, `mrw_read` |
| `docs/receipts.txt` | `mcp_read skipped.linked_dirs` | T1 | receipt readers (ADR-111) |
| `scripts/contract.sh` | §239; the §216 exact-object row lists the key | T1 | CI Linux |
| `AGENTS.md`, `README.md` | the `-- skipped:` paragraph names it | T1 | every agent |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| None — one task | T1 | — | no |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** a grep that missed a linked directory says so, and says what to do; the BACKLOG item closes.
- **Negative:** a tree with symlinks to directories (a `node_modules/.bin`, a monorepo link farm) now prints a `-- skipped:` line where it printed none.
- **Neutral:** nothing served and no exit code changes.

## Out of Scope

- A link inside a nested or ignored directory (permanent: boundary: the walk does not enter those, so it never meets the link; they are counted as the directory)
- `--ast-grep` hits under a linked directory (deferred: docs/adr/BACKLOG.md — whether ast-grep reports files under a link was not verified here, and the hit judge applies the walk's rules to whatever it reports)
- A FIFO, socket or device met by a walk, still skipped uncounted (deferred: docs/adr/BACKLOG.md — not something a grep is expected to find)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a link that leaves the root is counted | Low | the count says a link to a directory exists whose target is outside | `Resolve` refuses it before the count; `TestAnEscapingLinkToADirectoryIsNotCounted` pairs it |
| a file link is counted as a directory link | Low | a misleading count | the count requires `Stat` to be a directory; `TestALinkToAFileIsServedNotCounted` |

## Rollback

Revert T1: the links are skipped in silence again. No persistent state changes shape; a receipt reader that ignores an unknown key is unaffected (ADR-111).

## Follow-ups

- None — the record carries no open follow-up beyond the deferred item above.
