# ADR-096: A path the caller names is never dropped by a finder

**Status:** Accepted
**Accepted:** 2026-09-29 by Zy — "ADR-096 named links", selected under "Which records do you accept as written, so I can execute them?", with the two choices the draft left open: the absolute spelling of a named directory link, "Keep walking it" (which amends Decision 2: the draft refused it), and the relative link to the root, "Walk it as the root (Recommended)"
**Date:** 2026-09-29
**Owner:** M
**Spec:** None — no spec stage
**Cross-references:** ADR-006, ADR-007, ADR-016, ADR-058, ADR-064, ADR-071, ADR-076, ADR-077, docs/adr/BACKLOG.md
**Governs:** `internal/read/walk.go`, `internal/read/astgrep.go`, `cmd/mrw/main.go`, `scripts/contract.sh`, `scripts/break-campaign.sh`, `AGENTS.md`, `README.md`, `docs/adr/ADR-007-mrw-finds-the-files-it-serves.md`, `docs/adr/BACKLOG.md`
**Enforced-by:** `internal/read/link096_test.go::TestWalkRefusesANamedLinkToADirectoryAndNamesItsTarget`
**Invalidates:** none (amends ADR-007 rule 2) — checked. ADR-007 rule 3 is kept word for word: the walk descends no link to a directory. What changes is rule 2's named clause (ADR-007:84-86), which this record makes true for a link it was silently false for; T1 adds an `Amended by ADR-096` paragraph to ADR-007 in the shape of ADR-077's (ADR-007:124-127). ADR-064's named-path exclusion rules, ADR-058's "a `../` hit is a Problem", ADR-076's directory spelling and ADR-077's state refusal are consulted in the same order and are unchanged.
**Served-path change:** `mrw read --grep P dlink`, where `dlink` is an in-root link to a directory named relative to the root (`dlink`, `dlink/`, `dlink/.`, a chain of links), answers `==> dlink  REFUSED  is a link to the directory d …: name d` on the CLI and over MCP `grep` — naming the path as the caller wrote it — where v1.31.0 answered "no file matched" with no line about `dlink`; the same link spelled absolutely (`$ROOT/dlink`, `$ROOT/dlink/`) still walks `d`, as v1.31.0 does; a call that names such a link relatively beside paths it serves — `--grep P d dlink`, or a shell glob `*` in a directory holding one — exits 1 where v1.31.0 exited 0, on both finders; a named path that resolves to the root walks the root however it is spelled; `--ast-grep` and MCP `ast_grep` judge every named path the same way before ast-grep starts, so a path outside the root, missing, a loop, a FIFO, inside mrw's state, a relative link to a directory, or one mrw cannot open is refused by name and never handed to the binary, and an accepted path reaches the binary as the absolute path mrw judged rather than as spelled; and `mrw iter add` names the error the OS gave for a path it cannot stat (a link loop, a denied directory) instead of "no such file".

## Context

**What was observed.** The 2026-09-29 gap survey of v1.31.0 (agentsmemory drawer `349dbd2b`, item 7,
"loops-L10") reported that a named in-root link to a directory is dropped by `--grep`. Re-run for this
record on 2026-09-29 against the installed `mrw v1.31.0 (2ea8bd5)`, macOS/APFS, on a scratch fixture
holding `d/f.txt` and `d/sub/g.txt` (both containing `needle`) and the links named below:

| Named to `--grep needle` | v1.31.0 answer |
|---|---|
| `d` (the directory) | walked: `d/f.txt`, `d/sub/g.txt`, exit 0 |
| `flink` → `d/f.txt` | served as `flink`, exit 0 |
| `dlink/sub` (a path THROUGH the link) | walked: `dlink/sub/g.txt`, exit 0 |
| `.git` | walked (ADR-007's exclusion algorithm) |
| `dlink` → `d` | **`no file matched /needle/`, exit 1, no line about `dlink`** |
| `dlink/`, `dlink/.` | **the same** |
| `dlink2` → `dlink` → `d` | **the same** |
| `self` → `.` | **the same** |
| `$ROOT/dlink` (absolute) | **walked: `d/f.txt`, `d/sub/g.txt`, exit 0** |
| `$ROOT/self` (absolute) | walked the root, exit 0 |
| the root, named absolutely through a `--root` that is itself a link | walked the root, exit 0 |
| `loop` → `loop` | `REFUSED  stat …: too many levels of symbolic links` |
| `dangling` → nothing | `REFUSED  stat …: no such file or directory` |
| `up` → `..`, `outlink` → `/tmp` | `REFUSED  … outside the root` |
| `noperm` (mode 000) | `REFUSED  open …: permission denied` |
| `xs/mrw` (mrw's state, `XDG_STATE_HOME=xs`) | `REFUSED  … inside mrw's own state directory` |

| Named to `mrw iter add` | v1.31.0 answer |
|---|---|
| `loop` | **`no such file: loop (quote a spec containing spaces)`**, exit 2 |
| `noperm/f` | **`no such file: noperm/f (quote a spec containing spaces)`**, exit 2 |
| `dangling` | `no such file: dangling …`, exit 2 — true: the target is not there |

ADR-007 rule 2 says a named path mrw will not read "must produce a line, not silence"
(ADR-007:84-86). Five spellings produce silence, and one — the absolute spelling — descends the link
rule 3 says is never descended (ADR-007:93-95).

**Why, traced in the code at `8cbb89e`.** `walker.consider` stats the named path with `os.Stat`, which
follows the link, sees a directory and calls `walkDir` (`internal/read/walk.go:121-128`).
`filepath.WalkDir` then `Lstat`s its own start (`walk.go:147`), so the start arrives in the callback as
a non-directory entry, and the callback cannot tell it from a link it met while walking: it passes
`rooted.Resolve`, is not a regular file, and is dropped in silence by the branch written for
DISCOVERED entries (`walk.go:188-190`). `dlink/` and `dlink/.` reach the same place because
`rooted.Resolve` returns a joined, cleaned path (`internal/rooted/rooted.go:83`). The absolute spelling
takes another road: `consider` turns an absolute path into a root-relative one through `rooted.Real`
(`walk.go:91-115`), which resolves the final link, so `$ROOT/dlink` becomes `d` before anything asks
what it was.

**The same question, asked of the other finder.** `read.AstGrep` hands every named path to the
`ast-grep` binary unjudged (`internal/read/astgrep.go:75-80`) and judges only the HITS that come back
(`:116-130`). Read from the code, not run — `command -v ast-grep sg` finds nothing on this machine
(2026-09-29), so what the binary does with a link, a loop or a missing path is unmeasured:

- a named path outside the root is searched by ast-grep, and each matching file comes back as
  `REFUSED  … is outside the root` — its name printed. That is the pattern oracle `walk.go:165-179`
  closed for the walk's discovered paths, open on the named path (ADR-006: the root confines reads);
- a named path that is missing, a loop, a FIFO, inside mrw's state or a link to a directory produces
  whatever ast-grep produces, and `astGrepStarts` drops one it cannot stat without a line
  (`astgrep.go:236-239`), so none of them is reported by mrw;
- and there is no refusal to fall back on: with no path given, ast-grep searches `.`
  (`astgrep.go:76-77`), so a fix that drops refused paths must not leave an empty list behind.

**`iter add`** stats each path and files ANY error as missing (`cmd/mrw/main.go:1821-1823`), then
prints "no such file … (quote a spec containing spaces)" (`:1829-1831`) — a hint about word splitting,
given for a link loop and for a denied directory.

## The class, enumerated

**The class:** a path the caller NAMES to a finder — or to the working set that feeds the reader —
that comes back as no candidate, or as the wrong reason, without a line saying why. Enumerated
2026-09-29 at `8cbb89e`:

1. `mrw read --grep 'read\.(Walk|AstGrep)\(' --exclude '*_test.go' cmd internal` — **4 call sites,
   2 finders**: `read.Walk` from `cmd/mrw/main.go:793` (`--grep`) and `internal/mcp/tools.go:1416`
   (MCP `grep`); `read.AstGrep` from `cmd/mrw/main.go:807` (`--ast-grep`) and
   `internal/mcp/tools.go:1446` (MCP `ast_grep`). Neither MCP site judges a path itself, so a fix in
   `internal/read` reaches all four.
2. `mrw read --grep 'os\.(Stat|Lstat)\(' --exclude '*_test.go' cmd/mrw internal/read internal/iter internal/mcp` —
   **10 sites**: `walk.go:121` (named, the defect above), `walk.go:188` (discovered, silent by rule 2 —
   kept), `astgrep.go:236` (named, exclusion bookkeeping only), `read.go:446` (a named spec: reported
   `UNREADABLE`, not silent), `cmd/mrw/main.go:1821` (`iter add`, the wrong reason),
   `internal/iter/iter.go:309`, `:313` (mrw's own working-set file, not a caller's path), and three in
   `internal/mcp` that are not finders: `root.go:99` (`isUsableDir`, choosing the server's own root)
   and `tools.go:1678`, `:1686` (`anySameFile`, recognising an alias in an acknowledgement remedy).
3. The runtime probe above — eighteen named shapes against `--grep`, three against `iter add`.
4. The cold Codex review of this record (2026-09-29) found three more `--ast-grep` members, read from
   the code: a named path ast-grep cannot open, whose failure can parse as no hits
   (`astgrep.go:95-109`, `:175-179`); a named path whose `..` follows a link, judged cleaned by
   `rooted.Resolve` (`rooted.go:83`) but read by the binary as spelled; and a named path beginning with
   `-`, appended after ast-grep's options with nothing to end them (`astgrep.go:75-80`).

**Members, and what this record does with each:**

| Member | v1.31.0 | This record |
|---|---|---|
| named link to a directory, relative (`dlink`, `dlink/`, `dlink/.`, a chain) | silent | refused, naming the directory (T1) |
| the same link spelled absolutely (`$ROOT/dlink`, `$ROOT/dlink/`) | walked into | walked, unchanged — M's choice at acceptance (decision 2) |
| named link that resolves to the root (`self`, a linked `--root` named absolutely) | silent relative, walked absolute | the root, walked, every spelling (T1) |
| named link loop, dangling link, denied directory, path outside, state path | reported by the walk | unchanged |
| a named `.git` | walked | unchanged — ADR-007's exclusion algorithm names it |
| every named path under `--ast-grep` / MCP `ast_grep` | unjudged, handed to the binary | judged as the walk judges it, before the binary runs (T2) |
| a named path ast-grep cannot open — a mode-000 file or directory | the walk reports it (`walk.go:197-199`, `:208-211`); ast-grep's failure can read as no hits | refused with the OS's error before the binary runs (T2) |
| an accepted named path whose `..` follows a link, or that begins with `-` | ast-grep reads a file other than the one judged, or takes the path for an option | handed on as the absolute, cleaned path mrw judged (T2) |
| `iter add` on a path it cannot stat for a reason other than absence | "no such file" | the OS's error (T3) |

**Left out, on purpose:** a link to a directory the walk MEETS stays skipped in silence (rule 2 —
reporting discovered paths re-creates the oracle `walk.go:182-184` names); a path THROUGH a link
(`dlink/sub`) stays walked, judged by the boundary as `mrw read dlink/f.txt` is; a plain `mrw read`
spec (`read.go:446`) already answers `UNREADABLE … is a directory`; `mrw check PATH` maps paths to
packages rather than finding files, and its dropped path is survey item C8, a separate record.

## Existing Primitives Audit

- **`rooted.Resolve`** (`internal/rooted/rooted.go:72-153`) — the one boundary (ADR-006), already
  refusing outside paths, device names (ADR-081), a file spelled as a directory (ADR-076) and mrw's
  own state (ADR-077). **Reused unchanged**, and consulted first, so every existing refusal keeps its
  reason and its order.
- **`rooted.Real`, `rooted.IsRooted`, `rooted.Contains`** — how the walk already turns an absolute
  spelling into a root-relative one (`walk.go:91-115`), junctions included on Windows (ADR-071).
  **Reused unchanged.**
- **`walker.consider`** (`walk.go:82-140`) — the named-path judgement, today inlined with the
  dispatch. **Reshaped:** the judgement moves into one function both finders call.
- **`read.Problem`** — how both finders already report a path, printed as `==> PATH  REFUSED  REASON`
  by the CLI (`cmd/mrw/main.go:865-867`) and listed after `no file under the root matches` by MCP
  (`internal/mcp/tools.go:302-311`, `:320-334`). **Reused unchanged** — no new output shape.
- **`read.AstGrep`'s hit judgement** (`astgrep.go:116-130`) — kept for hits; it cannot judge a named
  path, because it runs after the binary has already searched it.

## Decision

1. **One judge for a named path, in `internal/read`.** `judgeNamed` answers, for each path the caller
   names: refused (with the reason), a file, or a directory to walk. It runs, in order, the absolute →
   root-relative conversion `consider` does today, `rooted.Resolve`, `os.Stat` (whose error is the
   reason, as the OS words it), and then the two rules below. `read.Walk` calls it for every named
   path; `read.AstGrep` calls it before the binary starts (decision 4). It lives in `walk.go`, which
   names no ast-grep (`TestWalkSourceNamesNoAstGrep`).
2. **A named link to a directory, spelled relative to the root, is refused by name.** When `Stat`
   reports a directory but `Lstat` of the path `rooted.Resolve` returned — joined onto the root and
   cleaned — does not, the path is refused: `is a link to the directory T, and a walk does not follow
   a link: name T`, where `T` is the resolved directory, root-relative. Cleaning is what makes `dlink/`
   and `dlink/.` name the link, since `Lstat("dlink/")` follows it. It asks `Lstat`'s `IsDir`, not
   `ModeSymlink`, so a Windows junction — `ModeIrregular` since Go 1.23 (ADR-071) — is caught by the
   same test. The refusal names the path as the caller wrote it (`dlink/` stays `dlink/`). **An
   absolute spelling is not refused** — M's choice at acceptance, to keep a spelling v1.31.0 serves.
   `consider` already turns `$ROOT/dlink` into `d` through `rooted.Real` (`walk.go:91-115`) before the
   test runs, so what the test sees, and what is walked, is the directory itself, served under its
   own names (`d/f.txt`), and no link is descended below it. The price is an asymmetry, stated rather
   than hidden: one link, two treatments by spelling — `dlink` is refused naming `d`, and `$ROOT/dlink`
   walks `d`. The refusal stays the reversible choice for the relative spellings (Alternatives); the
   absolute walk is kept because withdrawing it would break a caller v1.31.0 serves.
3. **Except the root.** A named path that resolves to the root names the root, however it is spelled,
   and the root is walked from its resolved path — which is what every absolute spelling of it does
   today, and what `.` does; walked from the link's own path, `WalkDir`'s `Lstat` of its start
   (`walk.go:147`) would drop it again. Relative `self` → `.` joins them instead of being refused as a
   link, which is why this rule is asked before decision 2's test. This is the first Alternative's
   follow-once, applied to the root alone. Every absolute spelling of the root — a linked `--root`
   named absolutely among them — is already walked by `consider`'s rewrite, as v1.31.0 walks it; the
   rule is what keeps a RELATIVE spelling that resolves to the root from meeting decision 2's refusal.
   Relative `self` was SILENT on v1.31.0, so walking it is chosen, not preserved; refusing it
   (`name .`) was the stricter reading, and M chose the walk at acceptance ("Walk it as the root").
4. **`--ast-grep` searches only what the judge accepts, as the judge resolved it.** `read.AstGrep`
   judges every named path first. A refused path is a `Problem` and is not handed to the binary. An
   accepted one is then opened and closed; one that cannot be opened — a mode-000 file or directory —
   is refused with the OS's error, because ast-grep's own failure to read it can come back as no hits
   (`astgrep.go:95-109`) where the walk reports it (`walk.go:197-199`, `:208-211`). What reaches the
   binary is the path the judge resolved — absolute and cleaned, `rooted.Resolve`'s answer — never the
   caller's spelling: a cleaned path leaves no `..` for the filesystem to read differently from the
   judge (with `self` → `.`, `self/../f.go` is judged as the root's `f.go` and would be read from the
   root's parent), and an absolute one cannot begin with `-` and be taken for an ast-grep option. One
   the judge resolved to the root is handed on as `.`, so the two finders agree on `self` without
   depending on what ast-grep does with a link. `astGrepStarts` — ADR-064's named-file and start
   rules — is fed the same accepted list, so a refused `dlink` is no start; the hit mapping and
   ADR-058's outside-hit Problem are untouched. When paths were named and none was accepted, ast-grep
   is NOT started: the `.` fallback is for a caller who named nothing.
   An absolute spelling of a link to a directory, which decision 2 accepts, is handed on as the
   directory it resolves to (`$ROOT/dlink` reaches the binary as `$ROOT/d`), so the two finders agree
   on it as they do on `self`.
5. **`iter add` says what the OS said.** A path whose `Stat` fails with `fs.ErrNotExist` keeps the
   "no such file … (quote a spec containing spaces)" sentence, whose hint is about that case; any
   other error is reported as the OS worded it, which names the path, at the same exit 2 with nothing
   added. On Windows `rooted.Resolve` follows links itself and refuses a loop before any `Stat`
   (`internal/rooted/links.go:77`), so there the boundary's own reason is the answer and stays so.
6. **Rule 3 stays one sentence.** The walk descends no link to a directory. A link it meets is skipped
   in silence, as every discovered non-candidate is; a link the caller names relative to the root is
   refused with the directory to name, unless it leads to the root (decision 3); one the caller spells
   absolutely names the directory it resolves to, which is walked (decision 2); a path through a link
   is the caller's address and the boundary judges it.

**What would make this the wrong decision**, checked by the tests T1 and T2 name: any named shape that
v1.31.0 served — `d`, `flink`, `dlink/sub`, `.git`, `.`, the root spelled through its link, `$ROOT/dlink` — stops
being served; or an existing walk or ast-grep symlink test (`TestWalkDoesNotDescendASymlinkedDirectory`,
`TestAstGrepAgreesWithWalkThroughSymlinks`, `TestAstGrepJudgesAFileSymlinkByItsTarget`) has to change
to pass. The fixture above can produce either failure today, so the criterion can fail. No shape
v1.31.0 served is withdrawn: the absolute spelling of an in-root directory link, which the draft
refused, keeps its walk (decision 2), pinned by `TestWalkStillWalksTheAbsoluteSpellingOfANamedDirectoryLink`.

## Alternatives Considered

- **Follow a named link once, after `Resolve`, and walk its target under the target's names** — what
  the absolute spelling does today, and one line. Rejected: the link would be followed when named and
  skipped when met — one link, two treatments, although the files served under the target's names
  would be the same; it reopens exactly what ADR-007's Amendment says would reopen it ("a walk that
  follows symlinks"); and a refusal is the reversible choice — relaxing it later breaks no caller,
  where withdrawing a follow breaks every script that learned it. It costs one extra call, whose
  argument the refusal prints. Decision 3 keeps this alternative for the root alone, for the reason it
  gives. **The draft also refused the absolute spelling's walk; M decided at acceptance to keep it**, so
  this alternative survives for that spelling as the rewrite `consider` already does, and decision 2
  states the asymmetry it leaves.
- **Walk the link under link-relative names (`dlink/f.txt`).** Rejected: ADR-007's Amendment already
  declined to thread a logical link-relative name through the walker, and rule 4's dedup would then
  serve one file under two names in `--grep P d dlink`.
- **Report the start where it is dropped — the `p == full` non-directory case in `walkDir`'s callback
  (`walk.go:188-190`).** Rejected as the location: the absolute spelling has already become `d` there,
  so it would stay walked, and `AstGrep` would have no judge to share.
- **Fix the walk and leave ast-grep's named paths to ast-grep.** Rejected: ADR-058 Decision 1 gives
  `--ast-grep` the walk's shape and ADR-064 Decision 1 has `read.AstGrep` apply ADR-007's rule as
  `read.Walk` does, so the two finders must not answer `dlink` differently; and the outside-root
  oracle stays open. ast-grep's own link and missing-path behaviour is also unmeasured here, and a fix
  that depends on it cannot be checked.
- **Pass ast-grep a link-following flag instead.** Rejected: it changes what ast-grep walks below a
  named directory, which ADR-064's rules are written against, and it closes nothing for an outside or
  missing path.
- **`iter add`: keep "no such file" and append the OS error.** Rejected: the hint "quote a spec
  containing spaces" is wrong for a loop or a denial, and a caller follows the hint.

## Component / Boundary Impact

| Component | Ownership after change | One reason to change? |
|---|---|---|
| `internal/read` (`walk.go`, `astgrep.go`) | judging the paths a caller names to a finder, then finding | Yes — it already owns finding |
| `cmd/mrw` (`iter add`) | wording a working-set refusal | Yes |
| `internal/mcp` | unchanged in production — `grepSpecs` and `astGrepSpecs` reach the judge through `read.*` | — |
| `internal/rooted` | unchanged | — |

**Engine go/no-go:** this record owns `internal/read/walk.go` and `internal/read/astgrep.go`.
`internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`,
`internal/iter`, `internal/rooted`, `internal/lines` and `internal/subproc` stay byte-identical against
the branch's merge-base with `origin/main` in every task (so a sibling record merged first, such as
ADR-094 in `internal/check`, does not fail these fences), and `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read --grep` / MCP `grep`, a named link to a directory | refused by name, naming the directory, in every relative spelling (was silence); spelled absolutely it still walks the directory it resolves to, as on v1.31.0 | `read.Walk` via `judgeNamed` (T1) | CLI and MCP callers |
| `mrw read --grep` / MCP `grep`, a named path resolving to the root | the root is walked in every spelling (relative `self` was silence) | `judgeNamed` (T1) | CLI and MCP callers |
| `mrw read --ast-grep` / MCP `ast_grep`, named paths | judged, and opened, before the binary runs; a refused one is a REFUSED line and never reaches ast-grep; an accepted one reaches it as the absolute path mrw judged (`.` for the root); none left means ast-grep is not started | `read.AstGrep` (T2) | CLI and MCP callers |
| `mrw iter add` | an unstatable path other than a missing one is refused with the OS's error | `cmd/mrw` (T3) | CLI |
| `judgeNamed` | new unexported function | `internal/read` (T1) | `read.Walk` (T1), `read.AstGrep` (T2) |
| `scripts/contract.sh` | §184 (both finders), §185 (`iter add`) | T1, T2, T3 | CI Linux, `adr-verify` |
| `scripts/break-campaign.sh` | probes `grep-named-dir-link`, `astgrep-named-outside` | T1, T2 | release campaign diff |

No new exit code: a refused named path is one more REFUSED line, counted as a range not served
(`cmd/mrw/main.go:899-901`, exit 1); over MCP it is named in the answer's text and counted in
`problems`, and the answer is `isError` only when nothing was served (`tools.go:311`, `:334`; ADR-024
leaves an answer that served a sibling unflagged: `readTool` composes its marked serve of numbered
lines with `isError` false). `iter add` stays exit 2. A
given call can move, though: one that named a directory link beside paths it serves exited 0 on
v1.31.0 and exits 1 now (Served-path change, Consequences).

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| `judgeNamed` (T1) | T1 | T2 | No — new, unexported |
| contract §184 (T1) | T1 | T2 | No — T2 adds its ast-grep rows to the section |

T3 is independent of both.

## Implementation

See `docs/adr/ADR-096-a-named-path-is-never-dropped-by-a-finder/tasks/README.md`. Three tasks: the walk
(T1), ast-grep (T2, consuming T1's judge), `iter add` (T3).

## Consequences

- **Positive:** every path a caller names to either finder, on either surface, is served or answered
  by name with the reason — the promise ADR-007 rule 2 made, which ADR-064 Decision 1 extends to both
  finders and ADR-058 Decision 4 to both surfaces.
- **Positive:** a named path outside the root is no longer searched by ast-grep, so no outside file's
  name comes back as a REFUSED line.
- **Positive:** ast-grep reads the file mrw judged: no `..` after a link or leading `-` in a named
  path makes it read another file or take the path for an option.
- **Positive:** `iter add`'s hint appears only for the mistake it is about.
- **Neutral:** one link, two treatments by spelling: `--grep P dlink` is refused naming `d`, while
  `--grep P $ROOT/dlink` walks `d` as v1.31.0 does (decision 2, M's choice at acceptance).
- **Negative:** a call that names a directory link beside paths it serves — `--grep P d dlink`, or a
  shell glob `*` that expands to one — exits 1 where v1.31.0 exited 0, on both finders; the served
  files are unchanged and the REFUSED line names the directory to name instead.
- **Neutral:** relative `self` → `.` now walks the root rather than serving nothing.
- **Neutral:** every named path pays one `Lstat` when it is a directory; negligible beside the walk.

## Out of Scope

- Following a link to a directory the caller names, other than one that leads to the root (decision 3) (permanent: boundary: rule 3 stays one rule for named and met links, and a refusal can be relaxed later without breaking a caller while a follow cannot be withdrawn)
- Reporting a link to a directory that the walk meets (permanent: boundary: ADR-007 rule 2 skips discovered non-candidates in silence, and reporting them re-creates the oracle `walk.go:182-184` names)
- A path through a link, such as `dlink/sub` (permanent: boundary: it is the caller's address, judged by `rooted.Resolve` as a plain read of `dlink/f.txt` is, and nothing below it is followed)
- What the real `ast-grep` binary does with an accepted path through a directory link (`dlink/sub`) and with an absolute operand, which only the fakes exercise here (deferred: docs/adr/BACKLOG.md)
- A named Windows junction as a walk start, exercised on a Windows runner (deferred: docs/adr/BACKLOG.md)
- `mrw check PATH` dropping a path it cannot place, survey item C8 (deferred: docs/adr/BACKLOG.md)
- `mrw iter add` accepting a directory (permanent: boundary: the read that consumes the entry answers `UNREADABLE … is a directory`, measured 2026-09-29 with `mrw read dlink`, so it is not silent)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| A caller meets the asymmetry — `dlink` refused, `$ROOT/dlink` walked — and reads one as a defect | Low | Low | decision 2 states it; the refusal names the directory; AGENTS.md and README say it in one sentence; `TestWalkStillWalksTheAbsoluteSpellingOfANamedDirectoryLink` pins the walk |
| A caller's list — most often a shell glob `*` — holds a directory link, and a call that exited 0 exits 1 | Med | Med | the served files do not change and the REFUSED line names the directory; the Served-path change and the release note say so; §184 pairs `d dlink` (exit 1) with `d dlink/sub` (exit 0) |
| The predicate refuses the root reached through a linked `--root` | Med | High | decision 3; `TestWalkStillWalksTheRootNamedThroughTheLinkItWasGivenBy` and its mutant |
| `AstGrep` falls back to searching `.` when every named path is refused | Med | High | decision 4; `TestAstGrepDoesNotRunWhenEveryNamedPathIsRefused` asserts the fake never started, and its own mutant |
| A trailing `/` makes `Lstat` follow the link and the refusal miss | Med | Med | the predicate cleans first; `dlink/` and `dlink/.` are refusal cases and `$ROOT/dlink/` a walk case |
| A Windows junction is not exercised | Med | Low | the predicate asks `IsDir`, which a junction's `Lstat` answers false; deferred to BACKLOG for a Windows run |
| ast-grep's own behaviour differs from the fakes | Med | Low | refused paths never reach it; an accepted one reaches it absolute and cleaned, and `astGrepRel` already maps an absolute or a relative hit (`astgrep.go:192-194`); what ast-grep prints for an absolute operand is unmeasured here (Out of Scope) |

## Rollback

Revert the three tasks' commits. No state, ledger or receipt format changes and no exit code moves, so
a tree used with the new binary is served identically by the old one.

## Follow-ups

- [ ] Release with the served-path change named, and the two campaign probes named as the expected differences against v1.31.0.
- [ ] Update the centralised `mrw` skill from AGENTS.md at the release that ships this record.
- [x] In T1's commit, set **Enforced-by:** to `internal/read/link096_test.go::TestWalkRefusesANamedLinkToADirectoryAndNamesItsTarget`.
- [x] In the commit that lands this record, add the three `docs/adr/BACKLOG.md` entries its Out of Scope defers to: real ast-grep on an accepted path through a directory link and on an absolute operand; a named Windows junction as a walk start, on a Windows runner; survey item C8, `mrw check PATH`.
