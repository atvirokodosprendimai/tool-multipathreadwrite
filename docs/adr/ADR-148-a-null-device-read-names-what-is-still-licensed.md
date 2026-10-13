# ADR-148: A read sent to the null device names what an earlier read still licenses

**Status:** Accepted
**Accepted:** 2026-10-13 by Zy — "okay, do what's possible and high impact now, including 1", item 1 being: make the null-device message say whether the lines it served are still licensed, and stop AGENTS.md recommending a file nobody reads.
**Date:** 2026-10-13
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-002, ADR-005, ADR-118, ADR-133
**Governs:** `cmd/mrw/main.go`, `cmd/mrw/nullread148.go`, `AGENTS.md`, `README.md`
**Enforced-by:** `cmd/mrw/nullread148_test.go::TestANullDeviceReadNamesWhatIsStillLicensed`
**Invalidates:** none — checked: ADR-133's line, its exit code and its empty ledger are kept; §234 greps `null device, so nothing was recorded`, which stays
**Served-path change:** `mrw read` whose standard output is the null device, when an earlier read or write still licenses every line it served of some file in that file's current version, prints one more stderr line naming those files; stdout, the exit code and the ledger are unchanged

## Context

ADR-133 made a read sent to the null device record nothing, and say so on stderr: "the answer went to the null device, so nothing was recorded: a write to these lines needs a read whose answer you see". That is true of the read. It is not a statement about the ledger, which may already license those lines from an earlier read, or wholly from a write mrw made (ADR-005, ADR-118).

Reported 2026-10-13 by a peer session: "twice this session I sent an mrw read to /dev/null … The second time mrw said nothing was recorded, but the write applied anyway." Reproduced the same day with v1.63.0 in a scratch root:
- a file never read, read to `/dev/null`, then `@@ f.txt 2 replace` — refused, exit 1;
- a file read to a file first, then read to `/dev/null`, then the same write — applied, exit 0;
- a file mrw had just written, read to `/dev/null`, then a write — applied, exit 0.

The guard held each time: the two writes that applied were licensed by the earlier read and the earlier write. What failed is what the caller was told. "nothing was recorded" read as "these lines are not licensed", and the write that then applied read as the guard broken.

The same report names the larger habit, and this session repeated it: a read sent to a file nobody opens (`> r.out`), only to license a write. mrw cannot tell such a read from one the caller looked at, and AGENTS.md recommended it: "Send it to a file, or read it, if you mean to edit (ADR-133)."

**Audit of the class** — *a CLI path that tells the caller a read licensed nothing*: `mrw read --grep 'nothing was recorded' cmd/ internal/` names two sites in `cmd/mrw/main.go`: the null-device line (this record) and "the answer could not be written, so nothing was recorded" (ADR-088 T4), which exits 2 on a failed flush, so no caller goes on to write believing the read worked. One site changes.

## Existing Primitives Audit

- `seen.Snapshot` reads the ledger under its lock, as a writer does (ADR-079); `seen.Observation.Covers` and `Whole` are the predicates the write's own check uses (`internal/apply/apply.go`, "has not been read").
- The read action already holds `observed`, the observations it would have recorded; the comparison needs nothing new from `internal/`.

## Decision

1. **A read sent to the null device that served lines** keeps ADR-133's line, word for word, and then, when the ledger still licenses every line it served in full of one or more files in their current version, prints one more line:
   `mrw: still licensed in this version by an earlier read or write, every line served here in full: a.txt, b.txt (2 of 3 files served); a write to those lines is judged on that licence, with the usual checks (ADR-148)`.
   - A file counts when its ledger entry has the sha this read observed and covers every span it served; for a whole-file read, an entry that is whole or whose spans reach the last line. An entry mrw wrote is whole (ADR-118).
   - The entry is found as a write finds it (ADR-029): by the spelling read, and on a miss by another spelling of the same file — an in-root link, a case variant where the filesystem folds case — through `os.SameFile`.
   - "In full": a line `--max-cols` cut is not in the observation (ADR-137) and is not vouched for. The line names a licence, not a verdict: a multi-line replace still needs the line after its range (ADR-052), and every other check still runs.
   - At most five names, then `and N more`; names sorted.
   - A file only partly covered is not named, though its covered lines stay licensed: a write to its uncovered lines is refused.
   - Nothing is printed for a file nothing licenses; ADR-133's line already says what a write meets there.
   - A ledger that cannot be read adds no line: the write reports what it finds.
2. **The ledger, stdout and the exit code are unchanged**: the read still records nothing.
3. **AGENTS.md and README stop recommending a file nobody reads.** They say a read licenses because the caller saw it, that an answer sent to a file never opened licenses all the same and mrw cannot tell, and that the null-device read names what an earlier licence still covers.

It fails if the line names a file the write would refuse, or misses one it would accept. The test drives both: a file read earlier is named and its write applies; a file never read, and one whose content changed since, are not named and their writes are refused.

## Alternatives Considered

- **A null-device read revokes the licence it reached.** Rejected: it would make discarding output a way to change what a write may do, and a whole-file write licence (ADR-005) would vanish on a read that served two lines.
- **Reword ADR-133's line only** ("this read recorded nothing; earlier reads still count"). Rejected: it cannot say which files, so the caller still learns it from the write.
- **Refuse a write to lines last served to the null device.** Rejected: the ledger does not keep which read served a line, and adding that is a ledger format change for a message problem.

## Component / Boundary Impact

None — internal to `cmd/mrw`'s read action. No engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` stderr, null-device stdout | one more line when an earlier licence covers the served lines | T1 | the caller |
| AGENTS.md, README.md | the read-before-write rule | T1 | agents, people |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** a caller hears from the read, not the write, that an earlier licence still covers its lines.
- **Negative:** a null-device read takes the ledger lock once more.
- **Neutral:** the MCP surface is untouched: it licenses through acks (ADR-031, ADR-039), not stdout.

## Out of Scope

- A read sent to a file nobody opens (permanent: boundary: mrw cannot see whether a file is read; the docs say so instead)
- Pipes into `head`, `grep` or PowerShell's `Out-Null` (permanent: boundary: ADR-133 Decision 3, mrw cannot see past a pipe)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The line drifts from the write's own licence check | Low | Medium | the test pairs each named or unnamed file with the write that must apply or be refused |

## Rollback

Revert T1: one function and one call in `cmd/mrw`, and the doc sentences.

## Follow-ups

None.
