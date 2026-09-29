# Task ADR-096-T1: The walk refuses a named link to a directory, by name

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `judgeNamed`; the directory-link refusal and the root rule in `read.Walk`; contract §184; campaign probe `grep-named-dir-link`
**Consumes:** `rooted.Resolve`, `rooted.Real`, `rooted.IsRooted`, `rooted.Contains` (existing, unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a named directory link is refused naming its target`, `the absolute spelling still walks`, `the refusal names the path as the caller wrote it`, `a trailing separator still names the link`, `a path that resolves to the root is the root`, `what v1.31.0 served stays served`, `both surfaces reach the judge`, `a mixed MCP answer stays unflagged`, `walk.go names no ast-grep`, `a contract row drives the binary`, `the campaign names the probe`, `ADR-007 records the amendment`, `the tree is gofmt-clean`, `the package vets for Windows`, `no other engine package changes`, `go.mod declares one requirement`

## Goal

`mrw read --grep P dlink`, where `dlink` is an in-root link to a directory, answers with a REFUSED line
naming `dlink` and the directory to name instead — relative, with a trailing `/` or `/.`, and through
a chain of links — on the CLI and over MCP `grep`, while the same link spelled absolutely still walks
the directory it resolves to, as v1.31.0 does (M's choice at acceptance), and a named path that
resolves to the root walks the root in every spelling.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/walk.go` | edit | `judgeNamed`: the named-path judgement `consider` does today (`:91-137`) as one package-level function T2 can call (`read.AstGrep` has no `walker`), plus the directory-link refusal and the root rule; `consider` calls it and keeps the dispatch |
| `internal/read/link096_test.go` | add | the four `Walk` tests |
| `cmd/mrw/link096_test.go` | add | the CLI test: the REFUSED line and the exit |
| `internal/mcp/link096_test.go` | add | the MCP `grep` test |
| `scripts/contract.sh` | edit | §184 |
| `scripts/break-campaign.sh` | edit | probe `grep-named-dir-link`, the release diff's named expected difference |
| `docs/adr/ADR-007-mrw-finds-the-files-it-serves.md` | edit | an `Amended by ADR-096` paragraph after rule 6, in the shape of ADR-077's at `:124-127` |
| `AGENTS.md`, `README.md` | edit | one sentence beside the `.git` sentence (`AGENTS.md:178`, `README.md:64`) |

**What selects it:** `read.Walk` calls `consider` for every named path (`walk.go:63-65`), and
`consider` calls `judgeNamed`. `Walk`'s callers — `cmd/mrw/main.go:793` and
`internal/mcp/tools.go:1416` — do not change. `TestGrepRefusesANamedDirectoryLinkByName` and
`TestMcpGrepRefusesANamedDirectoryLinkByName` both go red if the `judgeNamed` call in `consider`, or
the refusal inside it, is deleted.

## Ordered Steps

1. [S1] Write the six tests below and confirm them RED at `8cbb89e`: the refusal tests find no
   `Problem` (today's silence), and the root test's relative `self` → `.` case finds nothing walked.
   `TestWalkStillServesAFileLinkAndAPathThroughALink`,
   `TestWalkStillWalksTheAbsoluteSpellingOfANamedDirectoryLink` and the root test's linked-root and
   `.` cases pass at `8cbb89e` and must stay green throughout. [proof: mutation]
2. [S2] Move the judgement out of `consider` (`walk.go:91-137`) into `judgeNamed`, a package-level
   function of the caller's root and one named path — not a `walker` method, because `read.AstGrep`
   (T2) has no `walker`. It answers one of: a refusal (a `Problem`), a file to offer, or a directory
   to walk, the last two with the root-relative spelling and the full path `consider` uses today;
   `consider` keeps the dispatch. Behaviour does not change in this step: every existing `TestWalk*`
   test stays green. Neither the function nor its doc comment names ast-grep —
   `TestWalkSourceNamesNoAstGrep` reads `walk.go`. [proof: acceptance]
3. [S3] In `judgeNamed`, after `os.Stat` reports a directory, take the path `rooted.Resolve`
   returned — joined onto the root and cleaned — AFTER `consider`'s absolute → root-relative rewrite
   (`walk.go:91-115`), so an absolute spelling has already become the directory it names and meets no
   refusal (the record's decision 2, M's choice at acceptance). Then:
   (a) if `rooted.Real` of it is the root, the root is the directory to walk, walked from the resolved
   root (the `absRoot` `Walk` computes, `walk.go:51-57`), not from the spelled path, whose `Lstat`
   inside `WalkDir` (`walk.go:147`) would drop it again;
   (b) else if `os.Lstat` of it does not report a directory, refuse:
   `is a link to the directory T, and a walk does not follow a link: name T`, `T` the resolved
   directory, root-relative, slash-separated. The `Problem`'s `Path` is the argument as the caller
   wrote it (`dlink/` stays `dlink/`). Every other refusal keeps the `Path` it reports today.
   Ask `Lstat`'s `IsDir`, not `ModeSymlink`, so a Windows junction (`ModeIrregular`, ADR-071) meets
   the same test. [proof: mutation]
   Mutants: the refusal removed; the refusal asked of the caller's absolute spelling before
   `rooted.Real` resolves it (so `$ROOT/dlink` is refused); `Lstat` on the uncleaned path (so `dlink/`
   follows the link); the root rule removed; the root walked from the spelled path; the refusal's
   `Path` taken cleaned (so `dlink/` is reported as `dlink`); the refusal asked of every named path, a
   link to a file included.
4. [S4] Contract §184, driving `$MRW`: `--grep` over `dlink`, `dlink/` and `dlink/.` each exits 1
   with a REFUSED line naming `d`; `--grep` over `d dlink` exits 1 serving `d/f.go` beside the
   REFUSED line; paired, `--grep` over `$R/dlink` exits 0 serving `d/f.go` with no REFUSED line,
   over `d dlink/sub` exits 0 serving `d/f.go` and `dlink/sub/g.go`, and over a relative link to `.`
   exits 0 serving the root's match. Add the campaign probe `grep-named-dir-link` (`--grep` over a
   named `dlink`). [proof: acceptance]
5. [S5] ADR-007's amendment paragraph: a link to a directory the caller names relative to the root is
   refused by name; spelled absolutely it names the directory it resolves to, which is walked as
   v1.31.0 walks it; one that resolves to the root names the root; rule 3 is unchanged. AGENTS.md and
   README gain one sentence beside the `.git` sentence: "A link to a directory is not followed: one
   the walk meets is skipped, one you name is refused with the directory to name instead — unless you
   spell it as an absolute path, which names the directory it leads to — and one that leads to the
   root is walked as the root." [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestWalkRefusesANamedLinkToADirectoryAndNamesItsTarget|TestWalkStillWalksTheAbsoluteSpellingOfANamedDirectoryLink|TestWalkStillWalksTheRootNamedThroughTheLinkItWasGivenBy|TestWalkStillServesAFileLinkAndAPathThroughALink|TestGrepRefusesANamedDirectoryLinkByName|TestMcpGrepRefusesANamedDirectoryLinkByName' -v 2>&1 | tee /tmp/adr096-T1.out \
  && missing=$(for t in TestWalkRefusesANamedLinkToADirectoryAndNamesItsTarget TestWalkStillWalksTheAbsoluteSpellingOfANamedDirectoryLink TestWalkStillWalksTheRootNamedThroughTheLinkItWasGivenBy TestWalkStillServesAFileLinkAndAPathThroughALink TestGrepRefusesANamedDirectoryLinkByName TestMcpGrepRefusesANamedDirectoryLinkByName; do grep -qE "^--- PASS: $t \(" /tmp/adr096-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestWalk|TestAstGrepAgreesWithWalkThroughSymlinks|TestAstGrepJudgesAFileSymlinkByItsTarget|TestTheWalkServesNothingFromMrwsState' \
  && grep -q '^# 184\. ' scripts/contract.sh \
  && grep -q 'grep-named-dir-link' scripts/break-campaign.sh \
  && grep -q 'Amended by ADR-096' docs/adr/ADR-007-mrw-finds-the-files-it-serves.md \
  && [ -z "$(gofmt -l internal/read cmd/mrw internal/mcp)" ] \
  && GOOS=windows go vet ./internal/read/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

The first command carries the verdict alone; the second is regression (every existing `TestWalk*`
test, `TestWalkSourceNamesNoAstGrep` among them, and the ast-grep symlink pins ADR-064 left).

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWalkRefusesANamedLinkToADirectoryAndNamesItsTarget` | `internal/read/link096_test.go` | `dlink`, `dlink/`, `dlink/.` and `dlink2` → `dlink` → `d` each give exactly one `Problem` naming the spelling and `d`, and no spec | — | S1, S3 |
| `TestWalkStillWalksTheAbsoluteSpellingOfANamedDirectoryLink` | `internal/read/link096_test.go` | `$ROOT/dlink` and `$ROOT/dlink/` each walk `d` — `d/f.go` and `d/sub/g.go` served under `d`'s names — with no `Problem`, as `8cbb89e` does (M's choice at acceptance, the record's decision 2) | — | S1, S3 |
| `TestWalkStillWalksTheRootNamedThroughTheLinkItWasGivenBy` | `internal/read/link096_test.go` | with the root reached through a link `L`, naming `L` absolutely walks the root; a relative `self` → `.` and `.` itself walk the root; no `Problem` | — | S1, S3 |
| `TestWalkStillServesAFileLinkAndAPathThroughALink` | `internal/read/link096_test.go` | `d`, `dlink/sub` (served as `dlink/sub/g.go`), `flink` and `.git` are served as at `8cbb89e`, with no `Problem` | — | S1, S2 |
| `TestGrepRefusesANamedDirectoryLinkByName` | `cmd/mrw/link096_test.go` | `mrw read --grep P dlink` prints `==> dlink  REFUSED` naming `d` and fails "no file matched"; with `d` also named, `d`'s files are served and it fails counting one range not served | — | S1, S3 |
| `TestMcpGrepRefusesANamedDirectoryLinkByName` | `internal/mcp/link096_test.go` | `mrw_read` with `grep` over `[dlink]` is `isError` and its text names `dlink` and `d`; over `[d, dlink]` it serves `d/f.go`, carries no flag (ADR-024: `readTool` composes its marked serve of numbered lines with `isError` false) and its text still names `dlink` | — | S1, S3 |

Each symlink test skips (`t.Skipf`) where the platform cannot create a link, as
`TestWalkDoesNotDescendASymlinkedDirectory` does.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `judgeNamed`; the four `internal/read` tests |
| 2 — something selects it | `consider` → `judgeNamed`, reached from `cmd/mrw/main.go:793` and `internal/mcp/tools.go:1416`; the CLI and MCP tests go red when the call is deleted; §184 drives the built binary |
| 3 — the caller can discover it | the REFUSED line names the directory to name; AGENTS.md and README say it |
| 4 — it is used | telemetry is refused (ADR-009); the evidence for building it is the 2026-09-29 survey and the probe in the record's Context |

**Correction (2026-09-29):** the Mutation Log row for `go.mod` dated 0b2f304 that reads "mutant
killed" and credits `covers:go.mod declares one requirement` does not bind that claim. Its own note is
the only evidence of what it did: the one-requirement clause counts only lines matching `^require` or
a leading blank, so the directive line that mutant added was invisible to it. The row names no failing
clause, so what went red is unrecorded; it shows the fence noticed that edit and nothing about the
clause. What binds the claim now is the row dated f949cb9: a second `require` line for a module
already in the graph (`github.com/stretchr/testify v1.11.1`). With that line applied by hand, `go test
./internal/read/ ./cmd/mrw/ ./internal/mcp/ -count=1 -run
'TestWalkRefusesANamedLinkToADirectoryAndNamesItsTarget|TestGrepRefusesANamedDirectoryLinkByName|TestMcpGrepRefusesANamedDirectoryLinkByName'`
exited 0, so the build holds and the count is the clause the mutant reaches. Both rows stay; the log is
tool-written.

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · S3: the refusal removed — a named relative link to a directory is walked into silence again · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · S3: the refusal asked of the caller's absolute spelling before rooted.Real resolves it, so $ROOT/dlink is refused · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · S3: Lstat on the uncleaned spelling, so dlink/ and dlink/. follow the link and are walked into silence · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · S3: the root rule removed, so a relative link to the root is refused as a directory link · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · S3: the root walked from the spelled path, whose Lstat inside WalkDir drops it again · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · S3: the refusal names the cleaned path, so dlink/ is reported as dlink rather than as the caller wrote it · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · S1/S3: the link refusal asked of every named path, so a named link to a file is refused · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · the refusal names the link, not the directory to name instead · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:a named directory link is refused naming its target
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · the absolute spelling is Lstat-ed before rooted.Real and refused · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:the absolute spelling still walks
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · the refusal names the cleaned path, not the spelling · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:the refusal names the path as the caller wrote it
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · Lstat on the uncleaned spelling follows dlink/ · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:a trailing separator still names the link
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · the root rule removed · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:a path that resolves to the root is the root
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · a named link to a file, which v1.31.0 served, is refused · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:what v1.31.0 served stays served
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · the judge refuses nothing, so neither the CLI nor MCP names dlink · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:both surfaces reach the judge
- 2026-09-29 · 0b2f304* · mutant survived · exit 0 · `internal/mcp/tools.go` · an MCP answer that served d/f.go beside the refused dlink is flagged isError · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:a mixed MCP answer stays unflagged
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · walk.go gains an ast-grep name · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:walk.go names no ast-grep
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `scripts/break-campaign.sh` · the campaign probe is renamed away · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:the campaign names the probe
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `docs/adr/ADR-007-mrw-finds-the-files-it-serves.md` · ADR-007 no longer records the amendment · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:ADR-007 records the amendment
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/mcp/tools.go` · a marked MCP serve of d/f.go beside the refused dlink is flagged isError (the flag site a numbered serve takes, tools.go:505-508) · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:a mixed MCP answer stays unflagged
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/walk.go` · walk.go is not gofmt-clean · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:the tree is gofmt-clean
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/rooted/rooted.go` · an engine package changes · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:no other engine package changes
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `go.mod` · go.mod gains a second directive line, which the one-requirement clause does not count, so this binds nothing · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:go.mod declares one requirement
- 2026-09-29 · 0b2f304* · mutant survived · exit 0 · `go.mod` · probe only · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:go.mod declares one requirement
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-29 · f949cb9* · mutant killed · exit 1 · `go.mod` · go.mod gains a second require line that still builds (testify is already in the module graph), so every go test in the fence passes and only the one-requirement count goes red · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · covers:go.mod declares one requirement

## Invariants

- Every named shape `8cbb89e` served — `d`, `flink`, `dlink/sub`, `.git`, `.`, the root spelled through
  its link, `$ROOT/dlink` — is served, under the same names.
- A link to a directory that the walk MEETS is still skipped in silence (ADR-007 rule 2).
- `rooted.Resolve` runs before the link test, so an outside, state, device or directory-spelled path
  keeps its existing reason.
- `walk.go` names no ast-grep (`TestWalkSourceNamesNoAstGrep`).
- A refused named path counts toward exit 1 as every other REFUSED line does (`cmd/mrw/main.go:899-901`).

## Risks

- The root rule compares resolved paths: on macOS `/var` is `/private/var`, so both sides go through
  `rooted.Real`, as `consider`'s absolute branch already does; the linked-root test is the check.
- A Windows junction is caught by construction, not by a test on this task's runners (record, Out of
  Scope).
- **Correction (2026-09-29):** "caught by construction" held for a junction the caller NAMES, not for
  a junction that IS the root. `Walk` and `read.AstGrep` built `absRoot` with
  `filepath.EvalSymlinks`, which since Go 1.23 leaves a junction as written, so decision 3's root
  rule returned the junction as the walk start and `WalkDir`, Lstat-ing it as no directory, served
  nothing for `--grep` with no path or `.` (traced in source by review; not run here). Both now build
  it with `rooted.Real`. `cmd/mrw/junction096_windows_test.go::TestGrepUnderAJunctionedRootWalksFromTheRoot`
  pins it on the Windows CI shards; it cannot run on this task's darwin runner, and no red was seen
  for it here. `internal/read/walkroot096_test.go::TestWalkUnderARootReachedThroughALinkServesEveryStart`
  keeps the symlinked-root case walked on every platform.

## Stop Condition

Stop and ask if an existing `TestWalk*` or ast-grep symlink test must change to pass, if the judge
needs a change in `internal/rooted`, or if the root rule cannot be told apart from the refusal without
comparing spellings of the root.

## Out of Scope

- `read.AstGrep`'s named paths — T2.
- `iter add`'s wording — T3.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:314 · test-lock-sha256:a6771d7a64c9775d44486f29df49a3db4b5d047b64affda9c1285aebf349cf50 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvbGluazA5Nl90ZXN0LmdvCVRlc3RHcmVwUmVmdXNlc0FOYW1lZERpcmVjdG9yeUxpbmtCeU5hbWUJZGU0MWM1ZDJhMGU5MjVlZmYwNzg2NDc0MDg3MjI3MGQ1MGFjNDFhYmE0ZWNjMjczZTk4ZTBlNzY5YjM3MmU2Zgpib2R5CWludGVybmFsL21jcC9saW5rMDk2X3Rlc3QuZ28JVGVzdE1jcEdyZXBSZWZ1c2VzQU5hbWVkRGlyZWN0b3J5TGlua0J5TmFtZQk0NzZlNTVjYmJmOGM3OTVkMjVhMzgwZmNmZDFlNzMzZmFmMzliODc4MTZmMWFiYjJmZDMzNDc0Mzc1NjMyZmU3CmJvZHkJaW50ZXJuYWwvcmVhZC9saW5rMDk2X3Rlc3QuZ28JVGVzdFdhbGtSZWZ1c2VzQU5hbWVkTGlua1RvQURpcmVjdG9yeUFuZE5hbWVzSXRzVGFyZ2V0CTcwNWJlZmVmMWE5NGY0ZTBkMGJkY2Y1MDYyOWEwMzNmZTA0MDM5ZTQ2ZTMzNDlmNGE5MTAxNDlkMDRlYmJkYzAKYm9keQlpbnRlcm5hbC9yZWFkL2xpbmswOTZfdGVzdC5nbwlUZXN0V2Fsa1N0aWxsU2VydmVzQUZpbGVMaW5rQW5kQVBhdGhUaHJvdWdoQUxpbmsJZGJhMmY2NDU2OGMwYzI1OTliN2E2ZDkzN2Y1MTVlY2UzOGQwZjdiYTEyNDZjN2ViNDNmMDQxOTFmYjE4ZjYyNwpib2R5CWludGVybmFsL3JlYWQvbGluazA5Nl90ZXN0LmdvCVRlc3RXYWxrU3RpbGxXYWxrc1RoZUFic29sdXRlU3BlbGxpbmdPZkFOYW1lZERpcmVjdG9yeUxpbmsJNTJmMTRhZTM4YjNhOGEzODVkM2I5NjFmZjFhZmEwOTBiMTVjNzJhYTk1MDM1ZGZlYThmZGI5MDY1ZDY5ZmI1OApib2R5CWludGVybmFsL3JlYWQvbGluazA5Nl90ZXN0LmdvCVRlc3RXYWxrU3RpbGxXYWxrc1RoZVJvb3ROYW1lZFRocm91Z2hUaGVMaW5rSXRXYXNHaXZlbkJ5CWE0MzA3YTdiZDMzNTY0OGVkZDQ4NjA5ZDRjZDcwMzM5MDI5OTRiYjg3MzZiMzYxNWZiNjkwYjgxMmFkYjg3ZDM
  ```
  --- last 10 line(s) of stdout (of 42 after folding 42 raw)
          -- ck 486c9536cba80dd4 open lines 2-2 (1 lines follow)
              2| WANTED
          -- ck 486c9536cba80dd4 close
          -- This serve licenses NOTHING until you acknowledge it.
          -- Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
           type:text] map[text:{"observed":{"d/f.go":{"SHA":"f11b91c9677049d075a16d5289f3de0653a698361d096494b15399cb3d82599b","Spans":[[2,2]]}},"problems":0} type:text]]
  --- FAIL: TestMcpGrepRefusesANamedDirectoryLinkByName (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.089s
  FAIL
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:2135
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1739
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1987
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1801
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1890
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:2103
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1802
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:2221
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1955
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:3096
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:2638
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:3153
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1898
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1845
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1890
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1986
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1871
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:2082
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:2011
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:2123
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1942
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1928
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:1806
- 2026-09-29 · f949cb9* · exit 0 · `set -o pipefail …` · acceptance-sha256:32e43c6cd0e344043442a16f6551e13cb884e6ffb61fa35fcdeca3abce66c7b7 · ms:4185
