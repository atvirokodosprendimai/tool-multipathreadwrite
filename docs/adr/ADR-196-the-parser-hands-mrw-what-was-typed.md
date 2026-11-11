# ADR-196: the argument parser hands mrw what was typed, so mrw stops modelling it

**Status:** Accepted
**Accepted:** 2026-11-10 by Zy — answered *"Retire guard (Recommended)"* to which way #627 goes, then *"Accept and execute (Recommended)"* to accepting this record and executing T1 and T2 in one PR, after the Codex challenge of the draft (REQUEST CHANGES, three findings, folded in)
**Date:** 2026-11-10
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-069, ADR-161, ADR-151, ADR-078, ADR-164, ADR-100, ADR-071, issue #627, PR #549
**Governs:** `go.mod`, `go.sum`, `cmd/mrw/main.go`, `cmd/mrw/stdin.go`, `cmd/mrw/shortroot.go`, `cmd/mrw/run177.go`, `cmd/mrw/keep196.go`, `internal/rooted/links.go`, `internal/rooted/hardlink.go`, `internal/guide/guide.go`, `scripts/contract.sh`, `AGENTS.md`, `README.md`
**Enforced-by:** `cmd/mrw/parser196_test.go::TestTheParserHandsMrwOnlyWhatWasTyped`
**Invalidates:** ADR-069 Decision items 1 and 5–12 (the CLI refuses a positional or an attached flag value the parser trimmed, the whole-argv walk, the random differential test against the guard, and `mrw instructions` teaching the `--` escape for a padded path); ADR-161 Decision 11's clause "a lone `-` moves last before the arguments are parsed" and its refusal of a path, a second `-` or a `--` after a lone `-`. Checked and unchanged: ADR-069 items 2–4 (mrw's own trims of `--files-from` lines, the working set, a rename destination and the foreign formats' paths, contract §129–§131), which no parser reaches; ADR-151's digit-`-C` note (`shortRoot`), which keeps its classifier; ADR-161's `mrw read -` itself; ADR-071's Windows refusal of a name Win32 reads as another (`b.txt `), which is now what stops a padded path on NTFS.
**Served-path change:** a positional with leading or trailing whitespace (`mrw read 'x '`) reaches the command as typed, so it reads, writes or names the file `x ` (exit 0 when it exists, exit 1 `UNREADABLE` naming `x ` when it does not) where it was refused with exit 2; an attached flag value that ends in whitespace (`--files-from='list '`) is that value as typed where it was refused with exit 2; the refusals that said a token after a lone `-` "would be dropped" are gone, and the commands' own refusals answer those calls (`write - x` says `write takes at most one plan file`, `read - x` says a read of `-` names nothing else), still exit 2.
**Calls refused before that now run** (found by the Codex challenge of this draft, 2026-11-10; the first executed on a v3.14.0 build): `printf PLAN | mrw write - --` applies the plan, where the lift refused the `--` with exit 2 — the `--` is an empty terminator and the plan is the one the caller piped; `mrw iter note - TEXT` records the note; `mrw check - --` and `mrw run - --` reach their own argument checks. And an attached `--create='x '` creates `x ` where it was refused.

## Context

**What was observed.** Renovate's #549 bumps `github.com/urfave/cli/v3` from v3.11.0 to v3.14.0, and 11 tests in `cmd/mrw` fail on every CI platform, all of them ADR-069's guard family (CI run 41915039112, 2026-11-04; issue #627). Issue #627 said `mrw read 'x '` then "serves `x`". Measured 2026-11-10 with a v3.14.0 build in a scratch directory holding both `x` and `x `: it serves the file `x `, whose content is `padded`. The issue's description was wrong. The guard stopped firing because it detects a padded path by finding the token's TRIMMED spelling among the parsed arguments (`cmd/mrw/main.go:3519`), and v3.14.0 no longer delivers the trimmed spelling.

**What changed upstream, measured 2026-11-10 by diffing `command_parse.go` across the module cache:**

| Release | Positionals | Attached flag value (`--f=v `) | Lone `-` |
|---|---|---|---|
| v3.11.0 | trimmed (`:117`) | trimmed with its token (`:148`) | trimmed, kept, parse ends, every later token dropped (`:123-125`) |
| v3.12.0, v3.13.0 | kept byte for byte (`:118`, "the argument itself is what the action receives, byte for byte") | kept (`:152`) | trimmed, kept, parse ends |
| v3.14.0 | kept byte for byte | kept | kept as typed; parsing continues (`:124-127`) |

v3.14.0 still trims each token to CLASSIFY it (flag lookup, the `--` test, the dash checks: `command_parse.go:82`, `:107`, `:133`, `:145`), but the positional and the string value it hands the Action are the bytes typed. The table's positional column is the ordinary positional branch; a `--` and a dash before a non-letter are separate branches that keep what follows as given in every release.

**The premise, measured against the parser rather than the diff.** A scratch test drove random argv (every subcommand except the six the existing differential test excludes, every own and inherited flag, padded names, attached and separate values, `--`, `-`, padded `-`, a dash before a digit) through `parserDelivers`, the recorder in `cmd/mrw/paddedstress_test.go` that returns every positional and every string, slice and `--then` flag value the parser sets, and asserted each delivered string was a typed token or the text after `=` in one. Measured 2026-11-10 on macOS: v3.14.0, 0 untyped deliveries in 2,444, 2,409 and 2,413 dispatched cases (seeds 1, 2, 3); v3.11.0, red on seed 1 at once (`iter add 'x '` delivers `x`; `-C='y '` delivers `y`; `write … '-\n'` delivers `-`).

**The class this record governs:** *code in mrw that models a v3.11.0 parser behaviour v3.14.0 changed.* Enumerated with `grep -n 'command_parse\|command_run.go\|urfave/cli v3.11' cmd/mrw/*.go | grep -v _test` (2026-11-10, 13 hits) and the callers of what they name:

| Member | Where | What it models | Here |
|---|---|---|---|
| `refusePaddedArgs` | `main.go:3466`, called from read, write, iter, check, run | positional trim | removed |
| `refusePaddedFlagValues` | `main.go:3673`, called from `main` | attached-value trim, root `--` | removed |
| `padAttached` | `main.go:3653` | attached-value trim | removed |
| `liftLoneDash` | `stdin.go:170`, called from `main` | lone `-` drops what follows | removed |
| `flagRole`, `flagKinds`, `takesValue` | `main.go:3551-3650` | how the parser classifies a token | kept: `shortRoot` (ADR-151, `shortroot.go:24`, which trims each token to classify it as the parser does) and `skillSpelling` (ADR-168, `skill168.go:63`) walk argv with it to find where root options end; neither refuses a path. Its lone-`-` comment is made true |
| `TestTheGuardsAgreeWithTheParserOnRandomArgv` | `paddedstress_test.go` | the guard against the parser | replaced by this record's Enforced-by |
| refusal-message reader | `main.go:367` | v3.11.0's wording of a refused flag | kept: its tests pass on v3.14.0 (measured, `go test ./cmd/mrw` on the bump, 2026-11-10) |

**The lone-dash lift, measured.** With `liftLoneDash` bypassed on a v3.14.0 build, eight calls (`read - x`, `read x -`, `write - x`, `write - --dry-run`, `write - --then-sh true`, `write - -- x`, `read - --grep a`, `read - --max-lines 1`) exit with the same code as with it (2026-11-10, scratch directory). Only the message differs, and the lift's own was the false one: v3.14.0 does not drop the token it says would be dropped.

**Windows.** Removing the guard leaves `rooted.Resolve` (ADR-071) between `mrw read 'x '` and `x` on NTFS, where Win32 opens `x` for `x `. ADR-071 refuses a component ending in a space by its spelling. ADR-069's differential test skipped on NTFS, so Windows agreement was never measured there; this record adds a test that needs no file named `x ` and runs on every CI platform.

**Other records whose promises met a removed refusal**, found when T1 ran `go test ./cmd/mrw` on v3.14.0 (2026-11-10), each changed only where it rested on the refusal: ADR-078 T1, the cmd.exe and apostrophe forms of the padded-path fix (the fix is gone; the name is served as typed); ADR-164 T5, `read --grep -v ' a' f` was exit 2 before the read and is now the read's exit 1, its pointer kept; ADR-097, `--instructions= ` now reaches the parser's own refusal; ADR-092 T2, an attached `--then-sh` value ending in whitespace now runs as typed; ADR-100 and AGENTS.md/README's sentence that `check --json` refuses an attached padded value before its document; ADR-168 T2, whose row named the random differential test. Zy was told in the PR.

## Existing Primitives Audit

- **urfave/cli v3.14.0** itself keeps every positional and attached value as typed: the guard's reason is now the parser's behaviour.
- **`parserDelivers`** (`paddedstress_test.go`) already records what the parser hands each Action; the new property test reuses it.
- **`rooted.Resolve`** (ADR-071) already refuses on Windows a name Win32 would read as another.
- **The commands' own argument checks** (`stdinRead`'s one-spec rule, write's `at most one plan file`) already refuse what the lift refused.

## Decision

1. **mrw requires `github.com/urfave/cli/v3` v3.14.0** and adds no check of its own on how a token was parsed. A path is whatever the parser delivers, which is what was typed.
2. **The guards that modelled v3.11.0 are removed**: `refusePaddedArgs`, `refusePaddedFlagValues`, `padAttached`, `liftLoneDash`, and `subcommand` and `cmdExeForm`, which only they called. `flagRole` stays for `shortRoot` and `skillSpelling`, with its comments made true. Each test that pinned a removed refusal and that a done task locks keeps its name and asserts what v3.14.0 does with the same argv, as ADR-187 kept ADR-054's; the tests of ADR-078, ADR-092, ADR-097, ADR-100, ADR-164, ADR-165 and ADR-168 that met a removed refusal are retargeted the same way and relocked.
3. **The promise ADR-069 made is kept by a property, not a model.** `TestTheParserHandsMrwOnlyWhatWasTyped` drives random argv through the parser's recorder and fails when any delivered string was not typed, so a later urfave release that trims again goes red in CI. `TestAPaddedPathIsNeverServedAsItsTrimmedSibling` reads `x ` with only `x` on disk, through the binary, on every platform: it must not serve `x`.
4. **The contract sections ADR-069 T1 and T5–T12 own (§128, §134–§140) keep their numbers** and assert the new behaviour, each pairing it with the case that must fail; done fences in ADR-069, ADR-100, ADR-108, ADR-151 and ADR-164 grep those headers.
5. **The text stops teaching the escape.** `mrw instructions`, `--core`, AGENTS.md and README no longer say a padded path must go after `--`; they say a path reaches mrw as typed.
6. **A path the caller hands the command line that no root resolves is judged for Windows here.** A plan file and a `--files-from` list are opened with `os.Open`, never through `rooted.Resolve`, so on Windows `write 'p.mrw '` applied `p.mrw` once the guard was gone, and a separate `--files-from 'list '` had opened `list` on every release (the Codex review of #628). `windowsKeeps` refuses, on Windows, such a path whose component ends in a dot or a space, as `skill --dir` already did (`unkeptComponent`, ADR-168). A drive prefix is not a component (`C:.\p.mrw` is kept), and a `\\?\` path, which Win32 does not normalize, is not judged.
7. **The same paths are judged for a device name and an NTFS stream** (T4, the Windows field test of v1.86.0, 2026-11-10: four Windows 11 peers, one shell each). `write CON` waited on the console until it was killed, `write NUL` and `--files-from NUL` read the null device, and `write p.mrw:alt` applied a plan held in a stream that `type p.mrw`, `dir` and git do not show; inside the root each is refused (ADR-071, ADR-081). `windowsKeeps` now refuses a component holding `:`, under `\\?\` too since the filesystem parses streams, and outside `\\?\` a component whose name before its first dot is a device name, by the list `rooted` uses (`rooted.DeviceBase`). This reverses T3's Out of Scope line on a stream spelling, which held that a stream is "empty unless written": the field test wrote one. Only the volume is skipped (a drive, a UNC server and share, a device path's prefix with either separator), and a named pipe (`\\.\pipe\…`) local or remote, is not judged unless a `..` could climb out of it (the Codex reviews of #630).
8. **Past a device prefix, a path is judged by the kind of object it names** (T5, the Windows re-probe of v1.86.1, 2026-11-11). `\\.\GLOBALROOT\Device\ConDrv\Console` waited on the console, and `\\.\pipe\..\GLOBALROOT\Device\Null` and `\\?\GLOBALROOT\Device\Null` read the null device, because Decision 7 matched device names and no list of names covers every object. Past `\\.\`, `//./` or `\\?\`, once `.` and `..` are resolved as Windows resolves them (not under `\\?\`), the first component must be a drive, `UNC`, `Volume{…}` or `pipe`; anything else is refused, a file reached through GLOBALROOT included, which the caller names by its drive. A pipe also skips the state check (ADR-077, ADR-134), which opened the path before the read and spent a one-shot pipe's only instance, and both open sites judge a path before anything touches it.
9. **A volume, share or pipe namespace names a file only with a component after it** (T6, the Windows field test of v1.86.3, 2026-11-11). `\\.\C:`, `\\?\C:` and `\\.\Volume{…}` passed Decision 8 and reached os.Open on the raw volume, stopped only by "Access is denied" in shells that were not elevated, and `\\.\pipe` opened the pipe namespace. Past a device prefix, a drive, a `Volume{…}`, a UNC server and share, or `pipe` must have a component after it, and so must a plain `\\server\share`; otherwise the path is refused before it is opened.
10. **A plan or list refusal names the reason that held** (T7, the Windows field test of v1.86.4, 2026-11-11). `write C:\$MFT` was refused as "inside mrw's own state directory, or a hard link to a file in it". The refusal was right, because Windows would not open `$MFT` to read its identity and ADR-134 Decision 4 refuses a comparison that cannot finish, but neither reason was true. `\\.\C:\..` was refused as `"" names a Windows object`. An identity that cannot be read is now named as such (`errNoIdentity`). `rooted.OwnStateRefusal` replaces `InStateOrLinked` and returns the reason it found, which the CLI prints. A path that names nothing past its prefix says so. What is refused is unchanged.

**What would make this decision wrong:** a caller who relied on the refusal to catch a typo, `mrw read 'x '` meant as `x`. That call now reads `x ` and, when `x ` does not exist, exits 1 naming `x ` with its space, which is loud but no longer exit 2. And an urfave release that trims again: the property test is what catches it.

## Alternatives Considered

- **Keep refusing every edge-whitespace positional before `--`, as policy.** Rejected by Zy (2026-11-10, "Retire guard"): the refusal existed because the parser acted on a file the caller did not name, and it no longer does; a refusal of an exact name contradicts ADR-076's "a path means what it says".
- **Stay on v3.11.0 and pin Renovate.** Rejected: every later urfave bump fails the same way, and the guard family, eleven amendments deep, keeps modelling a parser that has fixed itself.
- **Patch the guard with one line (`got[t] || got[tok]`).** Tried 2026-11-10: 8 of the 11 tests pass, and the differential test cannot, because it asserts the guard refuses exactly when the parser trims, which v3.14.0 never does.

## Component / Boundary Impact

`cmd/mrw` only, plus `internal/guide/guide.go` text. No engine package changes. Byte-identical: `internal/apply`, `internal/read`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`, `internal/iter`, `internal/ingest`.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| CLI positionals | a padded path reaches the command as typed; no exit 2 | T1 | every CLI caller |
| CLI attached flag values | a value ending in whitespace is that value; no exit 2 | T1 | every CLI caller |
| a lone `-` | no lift; a path after it meets the command's own refusal, still exit 2; `write - --` applies the piped plan | T1 | CLI callers piping a plan |
| contract §128, §134–§139 | retargeted to the new promise, numbers kept | T1 | CI, done fences |
| `mrw instructions`, `--core`, AGENTS.md, README, contract §140 | the `--` escape for padded paths is no longer taught | T2 | every agent |
| `go.mod` | urfave/cli v3.11.0 → v3.14.0 | T1 | build |

## Inter-task Contracts

T2 describes the behaviour T1 ships; it depends on T1.

## Implementation

See `docs/adr/ADR-196/tasks/README.md`.

## Consequences

- **Positive:** a path reaches mrw as typed on every surface; the parser modelling (four functions in `main.go` and `stdin.go`) and the tests that pinned it go; Renovate's urfave bumps stop failing.
- **Negative:** a typo with a trailing space is no longer refused with exit 2; it reads a file of that exact name or names it UNREADABLE.
- **Neutral:** `mrw read -- 'x '` is unchanged; ADR-069 items 2–4 are unchanged.

## Out of Scope

- ADR-069 items 2–4, mrw's own trims of line formats and rename destinations (permanent: boundary: no parser reaches them; contract §129–§131 keep them)
- Shell-completion mode, where v3.14.0 still keeps a trimmed `--` (permanent: fact: completion hands no path to an Action; citation: version `github.com/urfave/cli/v3@v3.14.0`)
- The refusal-message reader at `main.go:367`, worded for v3.11.0 (permanent: boundary: its tests pass on v3.14.0, measured 2026-11-10; a wording change upstream turns them red)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A later urfave release trims again | Low | Med | the property test goes red in Renovate's PR |
| On Windows a padded path opens its trimmed sibling | Low | High | inside the root ADR-071 refuses it, and `TestAPaddedPathIsNeverServedAsItsTrimmedSibling` runs on the Windows job; a plan file and a `--files-from` list, which no root resolves, are refused by `windowsKeeps` (Decision 6, `TestAWindowsAliasOfAPlanOrListIsRefused`) |
| A done fence greps a removed contract header | Med | Low | headers kept (Decision 4); `scripts/fence-prose.py` in the static gate says which |

## Rollback

Revert the T1 and T2 commits: go.mod returns to v3.11.0 and the guards and their text come back together. They cannot be split, because the guards do not work on v3.14.0.

## Follow-ups

- Close #627 and #549 when T1 merges; the release that ships it names the served-path change.
