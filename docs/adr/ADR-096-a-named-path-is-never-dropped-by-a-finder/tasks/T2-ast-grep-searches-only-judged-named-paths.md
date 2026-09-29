# Task ADR-096-T2: ast-grep searches only the named paths the judge accepts

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `read.AstGrep`'s named-path judgement; §184's ast-grep rows (`argv184`); campaign probe `astgrep-named-outside`
**Consumes:** `judgeNamed` (T1), contract §184 (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a refused named path never reaches ast-grep`, `every refused named path is reported`, `a path mrw cannot open is refused`, `ast-grep is not started when nothing named is left`, `an accepted path is handed on as judged`, `an accepted sibling is still served`, `a refused path is no start`, `a path that is the root is handed on as .`, `both surfaces reach the judge`, `the existing ast-grep rules still hold`, `a contract row drives the binary`, `the campaign names the probe`, `the tree is gofmt-clean`, `the package vets for Windows`, `no other engine package changes`, `go.mod declares one requirement`

## Goal

`mrw read --ast-grep P <paths>` and MCP `ast_grep` judge every named path as `--grep` does (T1) before
the `ast-grep` binary starts: a refused path is a REFUSED line and is never handed to the binary, so a
path outside the root is not searched and no outside file's name comes back; a path mrw cannot open is
refused with the OS's error rather than left to read as no hits; an accepted path reaches the binary
as the absolute path mrw judged, so no `..` or leading `-` makes ast-grep read another file or take
it for an option; a path the judge resolves to the root reaches the binary as `.`; and when every
named path is refused the binary is not started at all.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/astgrep.go` | edit | judge each named path with `judgeNamed` and open each accepted one before the arguments are built (`:75-80`); a refused one becomes a `Problem`; hand on the judged absolute path (`.` for the root); feed `astGrepStarts` (`:113`) the accepted list; when paths were named and none is left, return without starting the binary |
| `internal/read/astgrep096_test.go` | add | the five `AstGrep` tests, with a fake `ast-grep` that appends its argv to a file before printing its fixed JSON |
| `cmd/mrw/astgrep096_test.go` | add | the CLI test, with the same recording fake |
| `internal/mcp/astgrep096_test.go` | add | the MCP `ast_grep` test, with the same recording fake |
| `scripts/contract.sh` | edit | §184's ast-grep rows, whose fake records into `argv184` |
| `scripts/break-campaign.sh` | edit | probe `astgrep-named-outside` |
| `AGENTS.md` | edit | the `--ast-grep` sentence (`:173-174`) says a named path is judged as `--grep` judges it, and reaches the binary as the absolute path mrw judged, before the binary runs |

**What selects it:** `read.AstGrep` calls `judgeNamed` itself, before `subproc.Command`
(`astgrep.go:89`). Its callers — `cmd/mrw/main.go:807` and `internal/mcp/tools.go:1446` — do not
change. The existing fakes print fixed JSON whatever they are asked (`installFakeAstGrep`,
`installFakeAstGrepJSON`, §116's `fake116`), so they cannot tell whether a path reached the binary;
the recording fake can, and every test here reads what it recorded.

## Ordered Steps

1. [S1] Write the seven tests below with the recording fake and confirm them RED at T1's head: the
   fake's argv holds the refused paths and the caller's spellings, and with only refused paths named
   it ran. [proof: mutation]
2. [S2] In `read.AstGrep`, after the `LookPath` check (`astgrep.go:65-67`, so a missing binary is
   still exit 2 first) and before the arguments are built, judge each named path with `judgeNamed`.
   A refused path is appended to the `Problem`s and left out. An accepted path is opened and closed
   (`os.Open`); one that cannot be opened is a `Problem` carrying the OS's error, and left out. What
   is handed on is the path the judge resolved — absolute and cleaned, as `rooted.Resolve` returned
   it — or `.` for one the judge resolved to the root (relative `self` → `.`, a linked `--root` named
   absolutely), as the walk walks the root for it (T1). `astGrepStarts` (`astgrep.go:113`) takes that
   same accepted list, not the caller's `paths`, so a refused `dlink` is no start. [proof: mutation]
   Mutants: the judge skipped; refused paths still appended to the arguments; an accepted path handed
   on as the caller spelled it; a path the judge resolved to the root handed on as spelled; the open
   check removed; `astGrepStarts` fed the caller's `paths`.
3. [S3] When the caller named paths and the judge accepted none, return the `Problem`s with no spec
   and do not start the binary. The `.` fallback (`astgrep.go:76-77`) stays for a caller who named
   nothing. [proof: mutation]
   Mutant: the empty list falls through to `.`.
4. [S4] Contract §184's ast-grep rows, driving `$MRW` through `bounded` with a fake on `PATH` that
   appends `"$@"` to `argv184`: `--ast-grep P $WORK/<outside dir>` exits 1 with a REFUSED line naming
   the outside root and `argv184` does not exist; `--ast-grep P dlink` exits 1 naming `d` and
   `argv184` still does not exist; a mode-000 file named beside `a.go` exits 1 with a REFUSED line
   naming "permission denied" and `argv184` lacks it (contract.sh's `skip` under uid 0, which ignores the mode);
   paired, `--ast-grep P d` exits 0 serving the fake's hit and `argv184` holds `d`'s absolute path,
   `--ast-grep P -- -p`, with a file `-p` in `$R`, exits 0 and `argv184`'s last operand is a path
   ending `/-p` (the v1.31.0 CLI accepts `read --grep x -- -p` and serves it, probed 2026-09-29), and
   with `self` → `.`, `--ast-grep P self/../f.go` gives an `argv184` holding `$R/f.go`'s absolute
   path and not the spelling. The start rule stays unit-only (`TestARefusedNamedDirectoryIsNoAstGrepStart`
   and its mutant): it needs `--exclude` beside a refused directory link, and that test drives
   `read.AstGrep`, the one function both surfaces call. Add the campaign probe
   `astgrep-named-outside`. [proof: acceptance]
5. [S5] AGENTS.md: a path named to `--ast-grep` is judged as `--grep` judges it before the binary
   runs, reaches it as the absolute path mrw judged, and the binary never sees one mrw refuses.
   [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestAstGrepNeverReceivesANamedPathTheBoundaryRefuses|TestAstGrepDoesNotRunWhenEveryNamedPathIsRefused|TestAstGrepHandsAPathThatIsTheRootOnAsDot|TestAstGrepHandsOnTheAbsolutePathItJudged|TestAstGrepRefusesANamedPathItCannotOpen|TestARefusedNamedDirectoryIsNoAstGrepStart|TestAstGrepRefusesANamedPathOutsideTheRootBeforeRunning|TestMcpAstGrepRefusesANamedPathOutsideTheRoot' -v 2>&1 | tee /tmp/adr096-T2.out \
  && missing=$(for t in TestAstGrepNeverReceivesANamedPathTheBoundaryRefuses TestAstGrepDoesNotRunWhenEveryNamedPathIsRefused TestAstGrepHandsAPathThatIsTheRootOnAsDot TestAstGrepHandsOnTheAbsolutePathItJudged TestAstGrepRefusesANamedPathItCannotOpen TestARefusedNamedDirectoryIsNoAstGrepStart TestAstGrepRefusesANamedPathOutsideTheRootBeforeRunning TestMcpAstGrepRefusesANamedPathOutsideTheRoot; do grep -qE "^--- PASS: $t \(" /tmp/adr096-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && go test ./internal/read/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestAstGrep|TestMissingAstGrepIsUsageAndNamesTheBinary|TestWalkSourceNamesNoAstGrep' \
  && grep -q '^# 184\. ' scripts/contract.sh \
  && grep -q 'argv184' scripts/contract.sh \
  && grep -q 'astgrep-named-outside' scripts/break-campaign.sh \
  && [ -z "$(gofmt -l internal/read cmd/mrw internal/mcp)" ] \
  && GOOS=windows go vet ./internal/read/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/check internal/state internal/iter internal/rooted internal/lines internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

`grep -q '^# 184\. '` alone would pass as soon as T1 landed; `argv184` is this task's own clause.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAstGrepNeverReceivesANamedPathTheBoundaryRefuses` | `internal/read/astgrep096_test.go` | beside an accepted `a.go`, a named `../out` (holding a file the pattern matches), `nosuch.go`, `dlink`, a link loop, `.st/mrw` (with `XDG_STATE_HOME` inside the root) and, where `mkfifo` exists, a FIFO each give one `Problem` with its reason; the fake's argv holds `a.go`'s absolute path and none of the others, and `a.go`'s hit is returned as a spec | — | S1, S2 |
| `TestAstGrepDoesNotRunWhenEveryNamedPathIsRefused` | `internal/read/astgrep096_test.go` | with only `../out` and `dlink` named: two `Problem`s, no spec, and the fake's argv file does not exist | — | S1, S3 |
| `TestAstGrepHandsAPathThatIsTheRootOnAsDot` | `internal/read/astgrep096_test.go` | with a relative `self` → `.` named, and again with the root reached through a link and named absolutely, the fake's argv holds `.` and not the link, and its hit is served | — | S1, S2 |
| `TestAstGrepHandsOnTheAbsolutePathItJudged` | `internal/read/astgrep096_test.go` | with `self` → `.`, a named `self/../f.go` (a different `f.go` sits in the root's parent), a named file `-p` and a named `$ROOT/dlink` (the absolute spelling decision 2 accepts): the fake's argv holds the root's `f.go`, `-p` and `d` as absolute paths and none of the spellings, and the hits are served | — | S1, S2 |
| `TestAstGrepRefusesANamedPathItCannotOpen` | `internal/read/astgrep096_test.go` | where the process is not uid 0 and not on Windows, a mode-000 file and a mode-000 directory named beside `a.go` each give one `Problem` naming "permission denied"; the fake's argv holds neither, and `a.go`'s hit is served | — | S1, S2 |
| `TestARefusedNamedDirectoryIsNoAstGrepStart` | `internal/read/astgrep096_test.go` | with `.` and `dlink` named and `--exclude d`, a fake hit on `d/f.go` is not served — `d` is no start, since `dlink` was refused — and `dlink` is one `Problem` | — | S1, S2 |
| `TestAstGrepRefusesANamedPathOutsideTheRootBeforeRunning` | `cmd/mrw/astgrep096_test.go` | `mrw read --ast-grep P <absolute path outside the root>` prints a REFUSED line saying outside the root and fails, and the fake never ran; `--ast-grep P a.go` serves the fake's hit | — | S1, S2, S3 |
| `TestMcpAstGrepRefusesANamedPathOutsideTheRoot` | `internal/mcp/astgrep096_test.go` | `mrw_read` with `ast_grep` over an outside path is `isError`, says outside the root, and the fake never ran | — | S1, S2 |

The link cases skip where the platform cannot create a link.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the judgement inside `read.AstGrep`; the five `internal/read` tests |
| 2 — something selects it | `read.AstGrep`, reached from `cmd/mrw/main.go:807` and `internal/mcp/tools.go:1446`; the CLI and MCP tests read the fake's argv, so deleting the judgement turns them red; §184 drives the built binary |
| 3 — the caller can discover it | the REFUSED line; AGENTS.md's `--ast-grep` sentence |
| 4 — it is used | nothing measures this yet; ast-grep is not installed on the machine the record was drafted on |

## Mutation Log
(empty until execute)
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: refused paths are still appended to the arguments, as spelled · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:a refused named path never reaches ast-grep
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: a refused path is left out in silence · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:every refused named path is reported
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: the open check removed · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:a path mrw cannot open is refused
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · S3: the empty list falls through and ast-grep starts with no path, searching . · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:ast-grep is not started when nothing named is left
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: an accepted path is handed on as the caller spelled it · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:an accepted path is handed on as judged
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: a path the judge resolved to the root is handed on as spelled · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:a path that is the root is handed on as .
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · S2: astGrepStarts fed the caller's paths, so a refused dlink is a start · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:a refused path is no start
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · one refused path costs the caller its accepted siblings · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:an accepted sibling is still served
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · the judge skipped: neither the CLI nor MCP refuses an outside path before ast-grep runs · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:both surfaces reach the judge
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · ADR-064: a named file stops being exempt from --exclude · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:the existing ast-grep rules still hold
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `scripts/break-campaign.sh` · the campaign probe is renamed away · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:the campaign names the probe
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · the judge skipped (re-run to read which surface tests kill it) · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:both surfaces reach the judge
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/read/astgrep.go` · astgrep.go is not gofmt-clean · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:the tree is gofmt-clean
- 2026-09-29 · 0b2f304* · mutant killed · exit 1 · `internal/rooted/rooted.go` · an engine package changes · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · covers:no other engine package changes

## Invariants

- A missing `ast-grep` is still exit 2 naming the binary, checked before any path is judged
  (`TestMissingAstGrepIsUsageAndNamesTheBinary`).
- With no path named, ast-grep still searches the root.
- ADR-064's rules hold: a named file is never excluded, a named directory is a start
  (`TestAstGrepServesANamedFileTheGlobWouldExclude`, §116).
- A HIT outside the root from an accepted path is still a `Problem` (ADR-058).
- `walk.go` still names no ast-grep: the judge is called from `astgrep.go`.
- An accepted named path reaches ast-grep as the absolute path the judge resolved, or `.` for the
  root, never as the caller spelled it; `astGrepStarts` sees only accepted paths.

## Risks

- A named path through a link that ends in `..` (`link/../f.go`) is joined and cleaned by
  `rooted.Resolve`, and it is that cleaned path ast-grep now receives, so the binary reads the file the
  judge judged; the hit mapping stays as ADR-064 left it, pinned by
  `TestAstGrepAgreesWithWalkThroughSymlinks`.
- The recording fake is a new helper in three test packages; a helper only tests use lives in a
  `_test.go` file (static-analysis rule), and three copies match the two existing fakes' pattern.

## Stop Condition

Stop and ask if `TestAstGrepAgreesWithWalkThroughSymlinks`, `TestAstGrepJudgesAFileSymlinkByItsTarget`
or §116 must change to pass, or if judging a named path needs anything from `walk.go` beyond
`judgeNamed`.

## Out of Scope

- The walk's own named paths — T1.
- What the real `ast-grep` does with a path it is given — the record's Out of Scope, deferred to BACKLOG.

## Verification Log
(empty until execute)
- 2026-09-29 · 0b2f304* · exit 1 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:2401 · test-lock-sha256:1968b655268bcc94d30cee12955e9656a9bbf75b625e97027dec52750a191a71 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYXN0Z3JlcDA5Nl90ZXN0LmdvCVRlc3RBc3RHcmVwUmVmdXNlc0FOYW1lZFBhdGhPdXRzaWRlVGhlUm9vdEJlZm9yZVJ1bm5pbmcJOGE5MWI2NmRlNzE1MmUzMDU5OGY3ZWY4ZGFmMWU1MmQ2NTRjZmVhMWI5NzY4NTBjOWMzODdjNjZiN2U0ZTEwMwpib2R5CWludGVybmFsL21jcC9hc3RncmVwMDk2X3Rlc3QuZ28JVGVzdE1jcEFzdEdyZXBSZWZ1c2VzQU5hbWVkUGF0aE91dHNpZGVUaGVSb290CTdlMjU1M2M3ZGU2ZmY0NzU0ZDFiY2M5NDQ2MzYwODhhYjhlNWQ4NTVkMGExOTY1YzNkMjBjNmY4ZTJiZDQ1ODIKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAwOTZfdGVzdC5nbwlUZXN0QVJlZnVzZWROYW1lZERpcmVjdG9yeUlzTm9Bc3RHcmVwU3RhcnQJNGE5Mjg2NmQ2MmM1YzhiMmI3YmFlNjA4Zjc4OWJjZGI4ZGFhMDQ3OGFkMTcwNTFlZjNlNDIzNzYxMGRmZmYzYQpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDA5Nl90ZXN0LmdvCVRlc3RBc3RHcmVwRG9lc05vdFJ1bldoZW5FdmVyeU5hbWVkUGF0aElzUmVmdXNlZAkzNDU5YTA5YTM1YjAzMTk4ODY4MTEzNDg4M2Q2YWVhZTNkYmUwYjdhMjUzNTJkYWE4YmMwMGQ0N2I3NzA3NmI1CmJvZHkJaW50ZXJuYWwvcmVhZC9hc3RncmVwMDk2X3Rlc3QuZ28JVGVzdEFzdEdyZXBIYW5kc0FQYXRoVGhhdElzVGhlUm9vdE9uQXNEb3QJZjc5YTA4M2U5YTI0NWI2MDZkMjk1ZDJiYWVjOWExZDJkY2E1MjljODg2OGFhMDkzMjE3ZDgxMzcwOTAwZmFlYgpib2R5CWludGVybmFsL3JlYWQvYXN0Z3JlcDA5Nl90ZXN0LmdvCVRlc3RBc3RHcmVwSGFuZHNPblRoZUFic29sdXRlUGF0aEl0SnVkZ2VkCTdhZmFlM2E3NmM2MjA4NWMxMWRhNzBkYmJiZTVjNzJkMjMxOTgyZmY4ZGYxYTBjNmFkNTdiMWViNjQ2NWRmYjUKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAwOTZfdGVzdC5nbwlUZXN0QXN0R3JlcE5ldmVyUmVjZWl2ZXNBTmFtZWRQYXRoVGhlQm91bmRhcnlSZWZ1c2VzCTJmMzE1ZWMwMGZkYTVmYzRlYThmYmU3NzU4N2UzNWE2Zjc5Y2IxNTY4ZmFjNTUxOGQzMWNiYzBjZDk0NGRjOWYKYm9keQlpbnRlcm5hbC9yZWFkL2FzdGdyZXAwOTZfdGVzdC5nbwlUZXN0QXN0R3JlcFJlZnVzZXNBTmFtZWRQYXRoSXRDYW5ub3RPcGVuCTFhMmRmYzMwMTQ0N2VjOWM5N2IwOTFlZDhiMmZmNTcyODBhNWYxNmI0NjBmNjNjODIxZjgzMTU3ODI1MGNhZjc
  ```
  --- last 10 line(s) of stdout (of 57 after folding 57 raw)
              2| func Target() {}
          -- ck 87881700a66cb498 close
          -- This serve licenses NOTHING until you acknowledge it.
          -- Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
           type:text] map[text:{"observed":{"a.go":{"SHA":"d84ffc8564ab0b079bf556e06573c0f779e6789f64e161ba4579218825f4ef8c","Spans":[[2,2]]}},"problems":0} type:text]]
      astgrep096_test.go:77: ast-grep ran on a path outside the root
  --- FAIL: TestMcpAstGrepRefusesANamedPathOutsideTheRoot (0.27s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.355s
  FAIL
  ```
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10390
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10819
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10589
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10903
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10930
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10425
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10631
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10388
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10336
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10864
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:10660
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:11573
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:13946
- 2026-09-29 · 0b2f304* · exit 0 · `set -o pipefail …` · acceptance-sha256:76e3b3217b838f680eecb7e1393b0d8216571f9b2cee64664ed8c1a80a9e787a · ms:12370
