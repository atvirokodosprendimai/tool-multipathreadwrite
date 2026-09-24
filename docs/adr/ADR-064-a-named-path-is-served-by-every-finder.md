# ADR-064: Both finders apply ADR-007's exclusion rule, and a taught pipeline runs as an agent runs it

**Status:** Accepted
**Accepted:** 2026-09-24 by M — *"accepted"*
**Date:** 2026-09-24
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-007, ADR-016, ADR-017, ADR-053, ADR-058, ADR-063, docs/adr/BACKLOG.md
**Governs:** `internal/read/astgrep.go`, `scripts/contract.sh`, `scripts/break-campaign.sh`
**Enforced-by:** `internal/read/leftovers_stress_test.go::TestAstGrepServesANamedFileTheGlobWouldExclude`
**Served-path change:** `mrw read --ast-grep P --exclude G …`, and MCP `ast_grep` with `exclude`, now follow ADR-007 as `--grep` does. A file the caller NAMED is served even when `G` matches it; it used to be dropped silently, exiting 1 "no file matched". A hit under a directory the search walked into and `G` matches is dropped; it used to be served.

## Context

**What was observed.** On 2026-09-24 two read-only reviews, looking for what makes mrw less robust, found that the two finders disagree about ADR-007's exclusion rule in both of its halves.

- ADR-007 decided the rule (`docs/adr/ADR-007-mrw-finds-the-files-it-serves.md:206-210`). `--exclude GLOB` matches a candidate's root-relative path or its basename. "Matching a DIRECTORY prunes it and everything under it. An EXPLICITLY named path is never pruned — the caller named it."
- The `--grep` walker honours both halves.
  - A named regular file goes to `offer` (`internal/read/walk.go:60-62`, `:133`) without `excluded()` being consulted.
  - A named directory is let through at `:151-153`. Directories the walk discovers are pruned at `:154`, and files at `:185`.
  - MCP `grep` uses the same `read.Walk` (`internal/mcp/tools.go:1218`).
- The ast-grep finder honours neither.
  - `internal/read/astgrep.go:98` runs `pathExcluded(rel, exclude)` on every hit, named or not. So `--ast-grep P --exclude b.go b.go` drops `b.go` and exits 1 "no file matched" (`cmd/mrw/main.go:775-777`).
  - `pathExcluded` (`:161-172`) tests only the hit's own path and basename, never its ancestor directories. So with nothing named, `--exclude vendor` serves `vendor/b.go`, which `--grep` prunes.
  - MCP `ast_grep` calls the same `read.AstGrep` (`internal/mcp/tools.go:1248`).
- ADR-058 Decision 1 gives the ast-grep finder "the same walk-then-serve shape" as `--grep`, and Decision 4 says the surfaces must not disagree. ADR-007's rule was meant to carry over, and the implementation missed both halves.
- **No test pins ADR-007:206-210's named or directory half on the ast-grep finder, nor the named half on `--grep`.** `cmd/mrw/planpath_test.go:275`, `internal/read/walk_test.go:81` and `internal/read/leftovers_stress_test.go:444` all pass no paths.

**A second member of the same class: a taught pipeline that only a human's shell exercises.**
- On 2026-09-24 the ADR-063 chaos pass ran the pipeline `mrw instructions` taught, `rg -l X | sed … | mrw read --files-from -`, with stdin an open pipe, as an agent host provides it. It hung until a 5 s timeout, because rg given no path searches stdin.
- ADR-063 added the `.`, and a Go test pins the text (`TestCLITeachesTheReadSide`). But no contract row executes either copy of this pipeline (`guide.go:58`, `AGENTS.md:160`) under an open stdin.
- §30 runs only AGENTS.md's ```` ```bash ```` fences, with inherited stdin. The one row that holds an open pipe (the MCP FIFO writer near `scripts/contract.sh:1744`) drives the server, not a taught pipeline.

**The class this record governs:** a caller-facing promise about which paths a finder serves, or about a pipeline the docs teach, that one path honours and nothing executes on the other. Enumerated 2026-09-24 with

```
git grep -n 'pathExcluded\|w.excluded(' -- 'internal/read/*.go'
grep -nE 'rg -l .*--files-from -' AGENTS.md internal/guide/guide.go README.md
```

- The first command finds three call sites. `walk.go:154` and `walk.go:185` are correct. `astgrep.go:98` is both defects.
- The second finds three copies of the pipeline. `AGENTS.md:160` and `guide.go:58` are governed here.
- Left out:
  - `README.md:68`, as this record's scope choice (see Out of Scope). ADR-053 retired README heading and tutorial-phrase assertions; it does not by itself forbid running a README example.
  - AGENTS.md's other ```` ```sh ```` fences. `:15` would run `contract.sh` from inside `contract.sh`, and `:133`/`:143` read this repository's `internal/`.

## Existing Primitives Audit

- **`astGrepRel(absRoot, p)`** (`internal/read/astgrep.go:138-159`). Reused to normalise a named path, so named paths and hits compare in one root-relative form: relative, absolute, and a direct in-root symlink to a file. It joins and cleans BEFORE resolving symlinks, so a named spelling that passes through a symlinked directory and then `..` can resolve to a different file than the filesystem would. That case is recorded as unresolved (Out of Scope), not promised.
- **`pathExcluded`** (`:161-172`). Reused unchanged for a file and for each ancestor directory.
- **`read.Walk` / `walk.go`**. Unchanged. Its byte-identity is held by the acceptance fence's merge-base diff. `TestWalkSourceNamesNoAstGrep` (`internal/read/leftovers_stress_test.go:459`) only checks that `walk.go` names no ast-grep strings.
- **Fake ast-grep helpers.** Reused: `installFakeAstGrepJSON` (`internal/read/leftovers_stress_test.go:474`), `installFakeAstGrep` (`cmd/mrw/astgrep_test.go:110`), and §111's shell-script fake with its `perl -e 'alarm shift; exec @ARGV'` bound.
- **§30's documented-example row.** Audited and not widened; see Alternatives.

## Decision

**1. `read.AstGrep` applies ADR-007's rule as `read.Walk` does, in both halves.**
- **Named files.** A path the caller named that resolves (through `astGrepRel`) to a regular file is never dropped.
- **Directories.** For any other hit, the hit's path and every ancestor directory strictly BELOW a start that contains it meet the glob. The hit is kept if ANY containing start admits it, whatever order the paths were named in.
- **Starts.** Each named directory is a start, and `.` names the root. With nothing named, the root is the only start. A hit that no named path contains is tested from the root. A start itself, and anything above it, is never tested (`walk.go:151-154`).
- **Spelling.** Exclusion matches the resolved root-relative path `astGrepRel` returns. For a hit reached through a symlink this can differ from the spelling `Walk` discovers; the divergence is recorded, not closed (Out of Scope).

**2. Every pipeline mrw teaches for agents is executed under an agent-shaped stdin on every contract run where rg exists.**
- The row takes the line from the built binary's `mrw instructions`, and the literal line from AGENTS.md.
- It runs each with stdin an open pipe and a 5 s alarm.
- It pairs them with the binary's line minus its path, which the alarm must kill.
- Where rg is absent, the row SKIPs visibly.

## Alternatives Considered

- **Make the walker match ast-grep (drop named files, keep directories).** Rejected: ADR-007 decided the reverse, and `--grep` is the reference shape.
- **Report a named-but-excluded file as a Problem (exit 1).** Rejected: it contradicts ADR-007:208 and `--grep`'s behaviour.
- **Widen §30's awk to ```` ```sh ```` fences.** Rejected: `AGENTS.md:15` is recursive and `:133`/`:143` need this repository's tree.
- **Also execute README.md's copy.** Rejected for this record: the binary's and AGENTS.md's copies are the ones agents are taught from. A README row would reopen the argument ADR-053 settled against README-as-fixture rows.

## Component / Boundary Impact

- `internal/read` owns the fix, in `astgrep.go` only.
- `cmd/mrw` and `internal/mcp` change no code; both reach the fix through `read.AstGrep`.
- `scripts/contract.sh` gains §116 and §117, and `scripts/break-campaign.sh` gains one probe.
- Byte-identical: `internal/read/walk.go`, `internal/guide`, and the engine packages `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, plus all of `internal/mcp`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `read.AstGrep` exclusion | named files exempt; walked ancestor directories tested | T1 | `mrw read --ast-grep`, MCP `ast_grep` |
| contract §116 | both finders: named excluded file served, walked excluded directory pruned, named directory not pruned | T1 | CI, `adr-verify` |
| `scripts/break-campaign.sh` probe | fake ast-grep, `--exclude` + named file | T1 | release campaign diff |
| contract §117 | taught pipeline runs under an open-pipe stdin; the path-less twin is killed | T2 | CI, `adr-verify` |

Exit codes unchanged. What callers see: a case that exited 1 now serves and exits 0, and a hit under an excluded walked directory is no longer served.

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| both finders apply ADR-007's rule | T1 | — | Partly. A caller relying on `--ast-grep --exclude vendor` still serving `vendor/…` loses those hits; that was a defect against ADR-007 |
| taught pipeline executed under open stdin | T2 | — | No |

## Implementation

See `docs/adr/ADR-064-a-named-path-is-served-by-every-finder/tasks/README.md`.

## Consequences

- **Positive:** the two finders and both surfaces give one answer for a named path and for an excluded directory. A taught pipeline that would hang an agent fails a contract run instead of a session.
- **Negative:** `contract.sh` runs two pipelines under a 5 s bound where rg exists. The must-fail twin always costs its 5 s.
- **Neutral:** ADR-058's shape is now what it said it was. README's pipeline stays outside contract.sh by this record's choice.

## Out of Scope

- Changing `--grep`'s behaviour (permanent: boundary: ADR-007:206-210 decided it and `--grep` is the reference)
- README.md's copy of the pipeline in contract.sh (permanent: boundary: agents are taught from the binary and AGENTS.md; a README row reopens the README-as-fixture argument ADR-053 settled)
- Installing rg or ast-grep on CI (permanent: boundary: the fake ast-grep and a visible SKIP cover it without a runner-image dependency)
- Engine packages `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, and `internal/mcp` (permanent: boundary: the fix lives in `internal/read/astgrep.go`)
- Executing AGENTS.md's other sh fences (permanent: boundary: AGENTS.md:15 is recursive and :133/:143 need this repository's tree)
- Symlink spellings: a named path through a symlinked directory and then `..`, and a hit reached through a symlink, are matched on the resolved path `astGrepRel` returns rather than the spelling `Walk` discovers (deferred: docs/adr/BACKLOG.md)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| rg absent on CI, so §117 only SKIPs there | High | Med | the SKIP is printed; T2's fence requires rg and fails on a SKIP, so the proof is taken locally |
| SIGALRM exit code differs by platform | Low | Low | assert `rc -gt 128` and a ≥ 4 s duration, not only 142 |
| a fixed-JSON fake ignores `paths` | Med | Low | the row comment says so; the Go tests use a per-case fake |

## Rollback

Revert the exclusion change in `astgrep.go`, the Go tests, §116, §117 and the campaign probe. Nothing persistent moves.

## Follow-ups

- [ ] Cut the release after merge (served-path change). Diff the campaign against v1.22.2 and name the new probe as the one expected difference.
