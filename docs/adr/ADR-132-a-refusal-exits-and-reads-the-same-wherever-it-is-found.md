# ADR-132: a refusal exits and reads the same wherever it is found

**Status:** Accepted
**Accepted:** 2026-10-06 by Zy — "Unify on exit 1" and "Whole class → 1" for the exit, the Windows peers' four findings among the round's items, then "go next, properly, build a spec/adr and execute. i want complete stability and completeness". The record's text was drafted after those answers and revised after two Codex reviews of it.
**Date:** 2026-10-06
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-001, ADR-066, ADR-080, ADR-086, ADR-091, ADR-105, ADR-106, ADR-111, ADR-125, ADR-126, ADR-128
**Invalidates:** None — it amends the exit code ADR-125, ADR-106 and ADR-086 gave their staging refusals (2 → 1) when the cause is one ADR-125 names; their rules, and when they refuse, are unchanged
**Governs:** `internal/apply/apply.go`, `internal/apply/replace.go`, `internal/apply/replace_other.go`, `internal/apply/wirepath.go`, `internal/mcp/wirepath.go`, `internal/mcp/tools.go`, `cmd/mrw/main.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/apply/refusal132_test.go::TestATargetsStateRefusesItsHunkAndReturnsNoError`
**Served-path change:** a plan refused before its first rename because of a target's state exits 1, "N hunk(s) failed — nothing was written", where it exited 2 — a target held by another process, a permission, a name the system refuses, a target changed or removed since mrw read it, a rename destination under a link to nothing or outside the root. Anything else found there — disk full, an I/O error, too many open files, an error mrw cannot name — stays exit 2. The CLI's receipt spells paths with `/` on every platform, as ADR-091 decided and `mrw_write` already does; its exit-2 line no longer repeats the FAIL line; `mrw_write`'s read advice names `mrw_read`; a check stopped by an interrupt or its deadline is headed so, not FAIL.

## Context

The Windows field test of v1.48.0 (2026-10-06, Windows 11 10.0.26200) found one condition reported two ways. A file held with `FileShare.None` fails the load, which ADR-125 made a hunk refusal: exit 1. A file held with `FileShare.Read` loads, then fails ADR-125's probe before the first rename, which goes through `abortStage` (`internal/apply/apply.go:883`): exit 2. Both receipts read "held open by another process", both write nothing. Zy, 2026-10-06: "Unify on exit 1", then, asked about the siblings, "Whole class → 1".

**Audit of the class** — *a refusal returned through `abortStage`, before the first rename, with nothing written*: `mrw read --grep 'return abortStage\(' internal/apply/apply.go` names nine sites (2026-10-06, `135eb2e`):

| Line | Cause | Kind |
|------|-------|------|
| 907 | the root cannot be opened | environment — the root is not a plan target |
| 921 | staging a file's new copy failed | by its error |
| 932 | the filesystem will not create a new name (ADR-086) | by its error; a probe left behind (ADR-105) is the environment |
| 957 | a rename destination under a link to nothing | target |
| 962 | a rename destination outside the root | target |
| 983 | a rename destination's directory or name refused | by its error |
| 994 | a target changed after mrw read it (ADR-106) | a detected change, or a target gone, is the target; any other error reading it is by its error |
| 1007, 1013 | a target cannot be replaced (ADR-125) | by its error |

"By its error" means: a cause `causeOf` names — a permission, another process holding the file, a name the system refuses — is the target's; everything else is the environment's, including errors no list here anticipates. That is the conservative side: an unknown error keeps the exit it has today. The Codex review of this record found two sites that format their error into text before anything can classify it — `changedSince` (`apply.go:2314-2317`) and `errors.New(openRefusal(...))` at 1007 and 1013 — so the cause is carried separately from the words. On unix `causeOf` names no invalid name today; APFS refuses a byte that is not UTF-8 with `EILSEQ` (contract §168), so `EILSEQ` and `ENAMETOOLONG` are named there too, which makes ADR-125's load refusal of such a name exit 1 as well.

Contract §119 (a rename into a read-only directory: `EACCES` from the probe) and §168 (a name APFS refuses) assert exit 2 and change with T1. Commit-stage failures after the first rename — `cmd/mrw/landed102_test.go:44`'s read-only-directory unlink among them — keep exit 2 (ADR-066).

The same field test, and the peer that drove ADR-128's probes, found four more places where an answer says something other than what happened:

- **The CLI's receipt spells paths with `\` on Windows.** ADR-091 decided a receipt names a root-relative path with `/`; `mrw_write` converts in `slashResult` (`internal/mcp/wirepath.go:36`), and the CLI's `write --json`, its refusals and its text receipt never did — nor its `drift` (`main.go:1403`) or the `created d\` line (`main.go:2088`, `filepath.Separator`).
- **The exit-2 line repeats the FAIL line.** `main.go:1352` prints the whole error — the reason the receipt's FAIL row already carries, with the OS error and absolute path — about 1.5 KB again at a 400-character path. The error can carry more than the row: `writer.ledgerFailed` (`internal/writer/writer.go:125`) appends a ledger failure to a partial commit's.
- **`mrw_write`'s read advice says "Run `mrw read X` first"** (`apply.go:1204`) — the CLI's command, on a surface whose caller may have no shell. ADR-128 cut the `--force` clause there and left this one.
- **A check stopped by an interrupt or its deadline is headed FAIL** — "check FAIL (exit -1, …) — interrupted" (`main.go:2010`) and "check FAILED (exit -1): … — interrupted" (`tools.go:1282`) — while the receipt says `skipped: "interrupted"` and no process exited with a verdict.

## Existing Primitives Audit

- **`abortStage`** — assigns every hunk's verdict for a staging failure; the target refusal is the same verdicts with no error returned, which is what a validation refusal already returns. The Codex review traced the callers: `MutationOf` stays `None`, the ledger and the write generation are untouched, `Land` records `refused_apply`, no check runs, `mrw_write` sets `isError` from `res.Failed`, cleanup and `left_behind` run before the return. One thing changes: steps a caller asked for reach `Verify` and are reported `not_run`, where the error path bypassed it.
- **`causeOf` / `platformCause`** (ADR-125) — the target causes; the classification is "does `causeOf` name it".
- **`slashResult`** (ADR-091, `internal/mcp/wirepath.go`) — moves to `internal/apply` as `Slashed`, so the CLI and `mrw_write` convert with one function.
- **`apply.Options.NoForce`** (ADR-128) — set by `mrw_write`; the read advice is cut by the same option.
- **`check.Interrupted`, `check.TimedOutBeforeStart`, `check.StoppedBeforeStart`** (ADR-080, ADR-126) — what the headline reads.

## Decision

1. **A refusal by the staging checks — the nine `abortStage` sites, all before the commit loop — whose cause is the target's is a hunk refusal**: its hunk fails with the reason it has today, its siblings skip, nothing is written, and `Apply` returns no error — the CLI exits 1 with "N hunk(s) failed — nothing was written", `mrw_write` answers as for any refused plan, the tally counts it refused, and a requested step is reported `not_run`. A probe mrw could not remove is mrw's own leftover whatever error its removal hit, so it stays exit 2 even when that error is a permission (the second Codex review of this record).
2. **The target's causes are named, not inferred**: a cause `causeOf` names (permission; held by another process; a name the system refuses — on unix now `EILSEQ` and `ENAMETOOLONG` too); a target that changed, was replaced, or no longer exists since mrw read it; a rename destination under a link to nothing or outside the root. Anything else — disk full, quota, I/O, a read-only filesystem, too many open files, a probe that could not be removed, the root, an error nothing names — keeps exit 2. Each site carries its cause beside its words, so classifying never parses text.
3. **The CLI's receipt spells every root-relative path with `/`** — `write --json`, its refusals, the text receipt, `drift`, and the `created d/` line — through the one converter `mrw_write` uses. `root` and log paths stay the platform's: they are absolute (ADR-091).
4. **An exit-2 line drops only what the receipt already printed**: when a failed hunk's reason is part of the error, the line names that hunk's FAIL line in its place, and keeps every other cause the error carries — a ledger failure after a partial commit included. Under `--json`, `error` is unchanged.
5. **`mrw_write`'s read advice names `mrw_read`**: "Read X with mrw_read first", cut by `NoForce` as ADR-128 cut the force clause.
6. **A check stopped by an interrupt or its deadline is headed by what stopped it** — `check INTERRUPTED` / `check TIMED OUT`, before it started or while it ran — on the CLI and on `mrw_write`, in place of FAIL / FAILED. The exit code (3) and the receipt are unchanged.

## Alternatives Considered

- **Held file only** — rejected by Zy (2026-10-06): one rule a caller can branch on is the point, and changed-since-read is the same kind of fact as a `sha=` guard, which is already exit 1.
- **Exit 1 for everything but a named list of environment errors** (this record's first draft) — rejected after the Codex review of it: `EMFILE`, `ENFILE` and errors nothing anticipated would have become "fix your plan". An error mrw cannot name keeps exit 2.
- **Convert paths to `/` inside the engine** — rejected: the ledger and the drift baseline key on the platform spelling, and changing those is a state migration nobody asked for.

## Component / Boundary Impact

`internal/apply` (the refusal kind, the named causes, the read advice, the shared converter), `cmd/mrw` (the trailer, the headline, the receipt paths), `internal/mcp` (the headline; the converter's new home), `internal/guide` (the exit sentence). Engine change owned by this record: `apply.go`'s staging returns and one refusal's words. `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| exit code | target-caused staging refusals 2 → 1 — **breaking** for a caller that branched on 2 there | T1 | CLI callers |
| `mrw_write` | the same refusals answer as a refused plan | T1 | MCP callers |
| CLI receipt paths | `/` on Windows, as ADR-091 and the schema say — a value change on Windows | T2 | `write --json` callers on Windows |
| CLI exit-2 line | names the FAIL line, keeps other causes | T2 | CLI callers |
| `mrw_write` read advice | names `mrw_read` | T2 | MCP callers |
| check headline | INTERRUPTED / TIMED OUT | T2 | CLI and MCP readers |
| `scripts/contract.sh` | §119 and §168 to exit 1; §231, §232 | T1, T2 | CI Linux |
| `AGENTS.md`, `README.md`, `internal/guide/guide.go` | the exit sentences | T1, T2 | every agent |
| `internal/mcp/testdata/legacy_golden.jsonl`, `cmd/mrw/testdata/outcomes113.golden` | the read advice; the timeout headline | T2 | their tests |

No receipt key is added or removed (ADR-111).

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `refuseStage` (T1) — the exit-2 line only meets errors T1 left as errors | T1 | T2 | no |

## Implementation

See `tasks/README.md`: T1, then T2.

## Consequences

- **Positive:** one exit code per kind of refusal, whichever check found it; a CLI receipt that keeps its documented spelling on Windows; refusal words a caller on either surface can follow.
- **Negative:** breaking for a script that branched on exit 2 for these refusals, and for a Windows `--json` consumer that matched `\` in paths. Both are named in the release notes.
- **Neutral:** what is refused, and when, does not change.

## Out of Scope

- Commit-loop failures, including the commit loop's own recheck of a target just before its rename, even when it refuses the plan's first target before anything was renamed (permanent: boundary: inside the commit loop a failure can follow renames already made, and ADR-066's PARTIALLY APPLIED / NOTHING WRITTEN with exit 2 is how a caller learns which; a change landing in the instant between the staging check and the commit loop is the window ADR-106 already names)
- The CLI `read --json` receipt's paths (permanent: boundary: the read side is its own surface; ADR-091 converted `mrw_read`'s, and a CLI read's spelling has had no report)
- Paths quoted inside a reason's prose, beyond the hunk's own path where the reason leads with it and the `mrw read X` advice — a rename destination, the ALREADY WRITTEN list, a path the system printed (permanent: boundary: a reason is prose that also quotes the operating system's errors, and a filename may hold any character, so no rule can tell mrw's quoted path from part of a system-printed one; three Codex rounds on #347 each found a new way a boundary rule half-rewrote a path. The receipt's path fields — what a caller parses — are all `/`; the reason keeps the platform's spelling beyond those two template positions)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a target cause classified as the environment | Medium | exit 2 where re-reading would help — today's behaviour | the named list is one function; a new cause is added by name when one is reported |
| a Windows consumer matching `\` | Low | its lookup misses | the release notes name it; ADR-091 and the schema always said `/` |

## Rollback

Revert the tasks. No persistent state changes; the ledger is untouched.

## Follow-ups

- None — the record carries no open follow-up.
