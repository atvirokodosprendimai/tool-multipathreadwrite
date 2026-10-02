# Backlog — what is not yet an ADR

Deferred work and open questions, each naming the record that punted it. An
entry here is a receipt: a deferral whose pointer names this file but never
wrote anything into it passes every check and exists nowhere, so the entry is
written in the same commit as the deferral.

`adr-debt docs/adr` sweeps the deferrals in the records and expects to find them
here.

## Inventory — M, 2026-09-12: *"I NAME THEM ALL"*

M said *"I NAME THEM ALL"* after the coordinator listed what ADR-040 left out,
what was dangling, and what every turn treated as noise. That quote **ranked this
table**. On 2026-09-12 they said *"good, accepted all"*: ADR-040 and every
backlog-named leftover became its own Accepted record. Engine dreams stay
unimplemented until a later execute names that number.

The same list is named in ADR-040 Out of Scope. Disposition is the column that
stops a later turn treating a row as noise. **Next quote** is the exact sentence
that arms work; silence leaves the row where it is.

| Item | Disposition | Next quote that arms work |
|---|---|---|
| Wrap-tail / read past range | **shipped** — `AGENTS.md`; not 040 | — |
| `raw=true` | **shipped** — ADR-015; not 040 | — |
| `create` at 0 (`body=0`) | **shipped** — ADR-027; not 040 | — |
| Insert vs neighbour-replace | **shipped** — plan ops; not 040 | — |
| ADR-035 `anchor=` on multi-line replace | **shipped** — not 040 | — |
| Original-file addresses | **shipped** — ADR-001; not 040 | — |
| Never `write \| head` | **shipped** — `AGENTS.md`; not 040 | — |
| `body=` is lines | **shipped** — taught; ADR-027 | — |
| Teach-only quoting on `write --help` | **shipped** — ADR-040 T1 | — |
| Parse unquoted `anchor=` until next `key=` | **shipped** — ADR-040 T2 | — |
| `mrw version` subcommand | **shipped** — ADR-040 T3 | — |
| Single quotes parse | **shipped** — ADR-040 T4 | — |
| `-C` vs `--root` (019 A stands; help names both global flags) | **decided** — ADR-040 Decision 5, a permanent boundary: help names both global flags, 019 pick A stands | — |
| PATH binary vs skill **version skew** | **ADR-041 Accepted** — record only | — (T1 receipts 2026-09-12; no protocol) |
| `mrw check` silent in-root fallback | **ADR-042 Accepted** — miss refused (T2) | — (2026-09-12: *"accepted, close"*; exit 2, no result) |
| Torn `Load` / atomic save | **ADR-043 Accepted** — measure, not a lock | — (2026-09-12: not observed; Load unlocked). Since ADR-075 the ledger a writer validates against is loaded under its lock (`seen.Snapshot`), and since ADR-079 `mrw seen`, the working set and the tally are too |
| MCP cargo: `check`/`iter`/`seen`/`stats` | **ADR-044 Accepted** — still two tools | — (T1 receipts 2026-09-12; no cargo tools) |
| Generate AGENTS.md from `Shared()` | **ADR-045 Accepted** — still refuse the tax | — (T1 receipts 2026-09-12; no generator) |
| Host-cut under ceiling | **ADR-046 Accepted** — measure, not a lock | — (2026-09-12: live cut not observed; no ack change). The measurement itself is still open: see "From ADR-032" below and the spec's UC-2, observed wire-only 2026-09-16, model check not run |
| Python `str` body character-split | **ADR-047 Accepted** — taught in 040 help | — (T1 receipts 2026-09-12; already taught) |
| Syntax awareness | **ADR-048 Accepted** — record only | — (T1 receipts 2026-09-12; no parser). Neighbour license is ADR-052, not a parser. |
| Padded write echo / neighbour license | **ADR-052 Accepted** — opt-in `--echo-pad`; End+1 license | — (M 2026-09-13: *"echo, license"*) |
| `--check` by default / `--no-check` | **ADR-054 Accepted** — CLI default when a check exists and a written path is not prose; §89 | — (M 2026-09-13: *"accepted"*; T1) |
| Delimiter-balance delta in the receipt | **ADR-054 Accepted** — visibility, not refuse; omitted on prose; a balanced insert is invisible to it; §90 | — (T2) |
| `stats` row: applied then a failing check | **ADR-054 Accepted** — every name at zero + landed line (`check_not_run` is in N); §91 | — (T3) |
| Neighbour license on a single-line address | **declined** — M 2026-09-26: none of the three Zeus cases would have fired it, and ADR-060 Out of Scope keeps the hint multi-line. Re-opens only on a field case where it would have fired | — |
| Advisory count on the write summary + `advisories` in JSON | **ADR-055 Accepted** (T1, §92) — the row fired, the summary M read did not carry it (twice in one hour, v1.16.0 field run) | *"advisory count"* |
| Repeat-pattern line on the receipt / `stats` | **ADR-055 Accepted** (T2, §93) — recent-window ring beside the ledger (op + advisory bit + time; no paths, ADR-009); "3rd balance advisory in your last 5 replaces" | *"repeat pattern"* |
| `--strict-balance` opt-in refusal on the wrap-tail signature | **ADR-055 Accepted** (T3, §94; default pre-registered below) — replace, single-line address, consumed net ≠ 0, body net ≠ consumed; exit 1, nothing written; campaign prices false positives before any default | *"strict balance"* |
| Streaming apply | **ADR-049 Accepted** — record only | — (T1 receipts 2026-09-12; still waits for a size that hurts) |
| Windows `%LOCALAPPDATA%` | **ADR-050 Accepted** — record only | — (T1 receipts 2026-09-12; XDG stays) |
| Foreign plan grammars / `apply_patch` | **ADR-051 Accepted** — compile to `@@`; first slice is `--format=apply_patch` | — (this steal; not Morph, not syntax-write) |
| Aider SEARCH/REPLACE as a second `--format` | **shipped** — ADR-051 F-26 | — (M 2026-09-12: *"commit, accepted, do work"*; `--format=search_replace`) |
| MCP `format` on `mrw_write` | **shipped** — ADR-051 F-27 | — (M 2026-09-12: *"YES, we have to be competitive"*) |
| `apply_patch` `*** Delete File:` / `*** Move to:` | **shipped** — ADR-057 | *"unlink op"* |
| `apply_patch` Move to with hunks | **shipped** — ADR-114 | Zy 2026-10-02: *"Engine: edit then rename"* |
| Honour quality-harness `fenceTimeout` | **ADR-059 Accepted** — alias of `timeout_seconds`; disagreeing keys refuse | *"both"* |
| Honour `{files}` when `packages()` cannot map | **ADR-061 Accepted** — `{files}`-only `scoped_check` runs on `.rs`; `{packages}`-only and mixed still fall back | *"so work on 054"* |
| `mrw instructions` as effective-use; always + plan (not 3+) | **ADR-062 Accepted** — Shared() first sentence is always + plan; CLI() cookbook includes `@@ path 0 create`; handshake stays Shared, 4096 | *"accept"* then *"use it always and plan activity"* |
| Centralised `mrw` skill always + plan (v20) | **shipped** — the palace skill `mrw` (v27, 2026-09-26) opens "Use mrw always: plan one read of every site, then one plan, then one write"; ADR-062's follow-up is ticked | — |
| `mrw instructions` teaches the read side (addresses, `--grep`, `--ast-grep`, `--exclude`, `--files-from`) | **ADR-063 Accepted** — read section on CLI(); every read flag named; root, `read` and `--ast-grep` Usage name finding; handshake unchanged | *"approve"* |
| Centralised `mrw` skill description names the read side | **shipped** — palace skill `mrw` v21 (2026-09-24), pinned at v1.22.2: description and body name the read side | *"release"* |
| `read` exits 1 when a `--max-lines` cap withholds lines | **decided** — M 2026-09-24: keep; ADR-033 stands; the two `TestKnownGap_*` read tests became decided tests | *"Keep exit 1"* |
| MCP registration scope (open since 2026-09-09) | **decided** — M 2026-09-24: user scope stays; README describes both scopes instead of prescribing one | *"Keep user scope"* |
| `--ast-grep` breaks ADR-007's exclusion rule in both halves (drops a named file; ignores a walked excluded directory); taught pipelines never run under an agent stdin | **ADR-064 Accepted** — named/ancestor-aware check in `astgrep.go`; contract §116/§117 | *"accepted"* |
| ast-grep symlink spellings: a named path through a symlinked directory and then `..`, and a hit reached through a symlink | **closed 2026-09-27 for directories** — `TestAstGrepAgreesWithWalkThroughSymlinks` pins the exact files each finder serves for a hit through a linked directory (link excluded, real directory excluded) and a named `link/../f.go`. **A symlink to a FILE stays deferred, as an accepted difference**: with `--exclude real.go`, the walk judges `alias.go` by the name it discovers and serves it, while AstGrep resolves a reported `alias.go` to `real.go` and drops it — pinned by `TestAstGrepJudgesAFileSymlinkByItsTarget`. Serving the alias would key AstGrep's hits by spelling. Arm on a caller who needs the alias served | — |
| A rename whose destination cannot be made half-applies the plan with every hunk `ok`; a failed path-op commit restores an unlink over a file renamed onto it (data loss since v1.19.0) | **ADR-066 Accepted** — destinations checked at validation and staged; path-op commit undone as a unit; truthful commit-failure receipts; contract §118/§119 | *"accepted"* |
| A CR-only file is one line to read and several to write (a write to an unserved line applied); CRLF lines served with their `\r`; apply_patch/search_replace refuse CRLF targets | **ADR-065 Accepted** — one `lines.Split` for read, write, `--grep`, MCP paging and both compilers; ast-grep CR-only hits reported; ADR-051 F-10 superseded; contract §120/§121 | *"accepted"*, *"Supersede F-10 in ADR-065"* |
| `mrw read f.go:/a/,$` was taught as "from here to the end" (ADR-036 Consequences, AGENTS.md) but serves the match line and the last line as two ranges, exit 0; as a write address it is refused | **docs fixed** — 2026-09-24 housekeeping: AGENTS.md now teaches `f.go:/a/,+99999` (a read clamps a relative end) and says `/a/,$` is two ranges; ADR-036's Consequences sentence is left as the historical record. A real pattern-to-end form is not built | — |
| A failing or truncated check keeps its `mrw-check-*.log` in the system temp directory for good, and nothing bounds how many accumulate | **fixed by ADR-080** — each check run removes its own logs older than 7 days from the temp directory and says how many; a timed-out or interrupted check now names the log it keeps, and one that never started keeps none. Found 2026-09-24 (3,103 on one macOS machine); contract §163 | — |
| An MCP `ast_grep` answer too large to serve comes back as an INDEX that carries only the COUNT of problems, so a CR-only file ADR-065 refuses is not named there | **fixed by ADR-067 T1** — found by the Codex review of #208 (P2, source-traced). `matchIndex` now prints one `-- <path>: <reason>` line per walk problem at all three index returns (`internal/mcp/tools.go`), never trimmed; contract §122 | — |
| `contract.sh` run as `./contract.sh` from inside `scripts/` resolves `SRC` to the repository's parent | **fixed** — 2026-09-26 housekeeping: the script captures the repository and its own path absolutely before its first `cd`, so §30, §43, §60's prologue probe, §117 and the conflict-marker check read this checkout from any directory. Found by Codex reviewing #204 (2026-09-24) | — |
| Desktop reach measure, under-ceiling host-cut, concurrent silent apply, strict-balance campaign, JSX nest probe | **spec** — `docs/specs/2026-09-16-dangling-high-impact.md`. Concurrent silent apply **closed by ADR-075** (contract §150; UC3-S2 now asserts the lock). Of the other four: UC-5 (JSX nest) was **probed 2026-09-16** (`docs/break/jsx-nest/`; `tsc` 0, DOM parent `#accidental-wrapper`) and is not a finding; UC-2 was observed wire-only 2026-09-16 with the model check not run, so it stays open; UC-1 (Desktop reach) and UC-4 (strict-balance campaign) are unrun. None counted as coverage | *"write a spec for these findings"* |
| leftover `body=` extra count, `--dry-run` parsed hunks, read neighbour hint, unquoted `anchor=` `"`, `body=@path`, check last-error line | **shipped** — ADR-060 | — |
| Per-extension check skip (`.jsonl` vs Cargo.toml) | **deferred** — ADR-054 / ADR-059; widening prose takes `.toml` | *"per-extension check"* |
| ast-grep-shaped `--grep` | **shipped** — ADR-058; Shipped 2026-09-15 as ADR-058 | *"structural find only"* |
| Playtrix T4 / that paste | **not-this-repo** | — (wing_playtrix) |
| Other wings' inboxes (quality-harness 28, etc.) | **not-this-repo** | — |
| Reopen ADR-019 pick B/C or `roots/list` | **not-this-repo** — Accepted A | — |
| Canvas file | **not-this-repo** — lives outside this repo | — |
| Installing mrw as a side quest | **not-this-repo** — consumer | — |
| `keep/` gitignore convention | **not-this-repo** — consumer hygiene | — |
| "Use CLI not MCP" as this binary's contract | **not-this-repo** — consumer harness | — |
| ADR-029 alias ledger BACKLOG row | **shipped** — closed on this page, 2026-09-12 | — |
| `shellArgs` quoting BACKLOG row | **shipped** — closed on this page, 2026-09-12 | — |

## From ADR-001 (a plan addresses the original file)

- **Streaming or memory-bounded application for very large files.**
  `internal/apply` reads whole files into memory and splits them into lines.
  Measured 2026-08-31 on a 14 MB / 200,000-line file: five hunks applied in
  0.15 s, a narrow ranged read in 0.03 s. Fine at that size, unbounded in
  principle. No decision needed until a real file makes it hurt — record the
  size that does.

- ~~**A `cmd N` registry of saved commands addressable by number.**~~ **DECLINED 2026-09-27.** Proposed
  2026-08-31 alongside the `@N` file pointers, deliberately not built: a shell
  command invoked by number is unreadable at the call site, so a wrong number
  runs the wrong thing with nothing to inspect. The safe variant is saved
  *plans* (`mrw write @p1`), which stay inspectable artifacts. Declined for that reason: `@N`
  pointers and `body=@path` already give the inspectable variant. Reopens only on a caller who needs
  replay and cannot use a saved plan file.

## From ADR-002 (mrw will not edit a file it has not seen)

- ~~**`mrw forget <path>`.**~~ **CLOSED 2026-09-03 — `seen.Forget` deleted.**
  It had no CLI caller and its doc comment described one ("Used by `mrw forget`
  when a caller knows their picture is stale") that never existed. Wiring it
  would have added a public subcommand, which needs an ADR; deleting removed
  dead code and a comment that read as evidence of a caller. `--force` remains
  the escape hatch for a stale picture. This was the second instance that day
  of "finished and unreachable" — `rooted.Descendable` was the first.

- ~~**Pruning ledger entries for deleted files.**~~ **MEASURED 2026-09-03 and
  DECLINED.** The entry said "harmless at current sizes; no measurement taken",
  so the measurement was taken:

  | | |
  |---|---|
  | this repo's ledger | 45 entries, 4,496 B, **1 stale** |
  | largest ledger on the authoring machine | 505 entries, **41 KB** |
  | every ledger on that machine, together | 13,819 entries, 935 KB |

  41 KB is not a problem, and the fix is not free. Sweeping inside `Record`
  couples a persistence function to the filesystem, and an implementation that
  stat'ed every entry broke two legitimate unit tests that use synthetic paths
  — `Record(obs)` followed by `Load()` no longer returned what was recorded.
  A narrower version (sweep only entries the caller did NOT just record) works
  and is written down here rather than shipped, because it buys 41 KB.

  **Reopen with a number.** If a ledger reaches a size where load or save is
  measurable, that is the trigger. Growth alone is not.

## From ADR-003 (a check's verdict comes from the process)

- ~~**Cleaning up the temp output files.**~~ **CLOSED 2026-09-03 — delete on
  success, keep on failure.** Measured before the fix on the authoring machine:
  **11,129 `mrw-check-*.log` files totalling 43 MB**, one per `--check` run ever
  made, none ever removed. The "harmless" reading was wrong and only a count
  showed it.

  Two conditions guard the delete, not one. A FAILING check keeps its log,
  because the tail is a summary and the file is the evidence. A truncated report
  keeps it even on success, because the report says "N earlier line(s) in
  <file>" and deleting a file the report points at is worse than leaving it.
  `Result.OutputFile` is cleared when the file is removed, so nothing ever names
  a path that is not there. Asserted by `scripts/contract.sh` §29, and by
  `TestAPassingCheckLeavesNoLogBehind` / `TestAFailingCheckKeepsItsLog`.

- ~~**Scope derivation for languages other than Go.**~~ **CLOSED 2026-09-27 — narrowed by ADR-061,
  the rest declined.** `{packages}` is derived by
  mapping `.go` files to their directories; any non-Go path forces the full
  check. A Python or Rust project gets the full command every time, which is
  correct but slow.
  Since ADR-061 a `scoped_check` whose template names `{files}` and not `{packages}` scopes any
  language, so a Python or Rust project that wants scoped runs declares one. What is left, mapping a
  path to a Cargo crate or a Python package so `{packages}` works there, means modelling each build
  tool, which ADR-061 rejects in its Alternatives ("Invent a Rust `packages()`") on ADR-054's
  permanent boundary. Reopens on a project whose `{files}` template cannot express its
  scope.

- **`--check` under `--dry-run`: settled as exit 2, recorded here because the
  question is ADR-003's and the answer was reached in a PR about ADR-008.**
  Nothing is written under `--dry-run`, so no check can run. Two readings, both
  defensible:

  *Exit 0 with a warning* — nothing was written, so there is nothing to verify,
  and that is materially different from a configured check going missing. Rule 2
  is about "no evidence" where evidence was possible. The caller also still gets
  the plan validation they asked for. **This is what shipped first, and it is
  wrong.**

  *Exit 2* — the caller asked for verification and received none. Rule 2 says a
  check that did not run is not a pass, and ADR-003's own exit table files a
  missing check under `2 | usage, parse, missing check, bad pointer`. The flag
  combination is also a plain contradiction, which is what exit 2 is for. **This
  is what ships now.**

  Exit 2 wins because rule 2 is already decided and this is an instance of it,
  not a new question — applying an accepted rule is conformance. The reason it
  is written down anyway: the first version returned exit 0, and a reader who
  found that in the tree could reasonably conclude rule 2 had been abandoned.
  If the opposite reading is ever preferred, it is a change to ADR-003 and wants
  a record, because a caller scripting `write --dry-run --check` now gets a hard
  failure where they got success.

  A refusal also has to be POSITIONED, and the exit-0 version hid that: the
  first refusal sat after the plan was parsed and applied, so an unparseable
  plan preempted it while a plan whose HUNK failed lost to it. Both are "your
  plan is wrong" and they ranked differently only because of where the test sat
  — and exit 1, which promises an untouched tree, became exit 2. It is settled
  before the plan is read now, so a usage error preempts everything, which is
  what exit 2 means. Pinned by the precedence rows in `contract.sh`.

## From ADR-004 (mrw leaves nothing in the working tree)

- **Pruning orphaned state directories.** ✅ **RESOLVED by ADR-034 (2026-09-07).** Moving or
  deleting a checkout leaves its `$XDG_STATE_HOME/mrw/<key>/` behind. `mrw seen --prune` now removes
  the entries whose `root` marker names a path that is gone, and says what it removed;
  `--prune --dry-run` shows the list first.

  **The size estimate here was wrong and is corrected.** This entry and ADR-004's Consequence at
  `:164` both said each orphan is *"a few hundred bytes"* and that a human can clean up with
  `grep -r . "${XDG_STATE_HOME:-$HOME/.local/state}/mrw"/*/root`. Measured 2026-09-07 on the
  maintainer's machine: 22,836 directories using 242 MB of disk, of which 22,591 were dead — and a
  second reading hours later, 24,067 directories and 256 MB. ⚠ Those are `du` figures. The FILES
  came to 10.7 MB across 65,235 of them; the rest is one block per tiny file and one per directory,
  so the real cost is inodes. Either way, selecting 22,591 `rm -rf` targets by hand out of that grep
  is not a cleanup a human does.

- **Windows conventions.** The state path is XDG-shaped; Windows would want
  `%LOCALAPPDATA%`. Nothing currently builds or tests mrw on Windows beyond
  cross-compiling the binary in CI, so this is unexercised rather than broken.

## Not tied to a record

- **The `human-decisions` amendment on shell file-writers.** Drafted 2026-08-31,
  awaiting M's approval. The workspace rule bans ad-hoc shell file-writers
  because gates keyed to `Edit`/`Write` cannot see a heredoc; the proposed
  narrowing exempts a single named tool that emits a machine-readable receipt,
  on the grounds that a `Bash` hook can match its name and read what it wrote.
  Evidence: `wing_craft/tooling` drawer
  `16b01a4c4c8a4448839ca08100cd63ca8e0ef0281193d0c8b6ac1c5688bcf0f7`. Not an ADR
  because it is a workspace-wide convention, not a decision about this
  repository.

- ~~**Go-level coverage of `cmd/mrw`'s CLI wiring.**~~ **CLOSED 2026-09-27.** `cmd/mrw` has only
  `version_test.go`; pointer resolution in a hunk path and the exit-status
  mapping are covered end-to-end by `scripts/contract.sh` instead. Noted in
  ADR-003-T2 as a stated limitation rather than an oversight, but a Go test that
  execs the built binary would close it.
  Since then `cmd/mrw` gained 45 test files, the exit-status mapping is driven in-process by ten of
  them, and `TestTheReceiptIsOnStdoutBeforeTheCheckStarts` execs a built binary. The one wiring left
  uncovered, a pointer as a hunk path, is `TestAPointerHunkPathNamesExactlyOneFile`
  (`cmd/mrw/pointer_write_test.go`): the built binary refuses `@1-2` with exit 2 and nothing
  written, and lands `@2` on its one file.

## From ADR-007 (mrw finds the files it serves)

- **A cross-file `--max-lines` budget.** Deferred again from ADR-033, which
  settles what a cap of ZERO means and leaves the scope alone. The cap is per
  SPEC today — `read.Run`
  resets `budget := opt.MaxLines` (now `capped := opt.MaxLines != nil`, ADR-033) for each one — and ADR-007's walk deduplicates
  so that it is per file for everything the walk produces. What nobody has
  decided is whether `mrw read --grep PAT .` over a large tree should have a
  budget for the WHOLE answer rather than per file, which is the number an agent
  paying for context actually cares about. No measurement taken; the shape of
  the answer probably depends on what T2's cost measurement says.
  **Still deferred, 2026-09-27 — waits for a trigger:** a real `--grep` answer that outgrows what a
  caller can use, or a cost measurement that names the whole-answer number. The MCP surface already
  pages the whole answer (PARTIAL and `next_read`), and redefining `--max-lines` would change what
  every existing cap means.

- **Parallel walking or searching.** ADR-007's walk reads every candidate to
  match it and then `read.Run` reads the matching ones again, serially. That is
  the price of `Run` staying the only reader that observes (ADR-005). If T2's
  go/no-go cost measurement comes back close to the 2× threshold, parallelism is
  the first thing to reach for — and it needs its own answer to whether output
  order stays deterministic, which the receipt format assumes.

## From ADR-010 (mrw speaks MCP over the same engine)

- **A stateless hash-in-request mode, as `mcp-text-editor` uses.** The caller passes back the SHA it
  read, so the tool holds no ledger and has no concurrency limit on any transport. It is the
  cheapest fix for the parallel-read limitation and ADR-010 rejected it for one reason: it moves the
  read-before-write guarantee INTO the caller, and a caller that echoes a SHA it never read has
  licensed itself. ADR-002 exists so the tool holds that fact. **This is the fallback if the server
  path fails its go/no-go** — recorded here so it is not re-derived from scratch.

- **Exposing `check`, `iter`, `seen` and `stats` as MCP tools.** ADR-010 ships two tools because
  read and write are the product. Each of the others is a separate decision with its own answer to
  "what did the caller see": `check` runs a subprocess, `seen` exposes the ledger, `stats` exposes
  the tally. None is obviously wrong; none is free.

- ~~**The CLI path's parallel-read limitation.**~~ **CLOSED 2026-09-09 by ADR-038.**
  Exclusive lock around `seen.Record`. §76: 40 concurrent reads keep 40. The
  hash-in-request fallback above stays; it is a different decision.

- **Publishing to an MCP registry or directory.** Deferred from ADR-010-T3. The config block in the
  README is the install path; a registry listing is distribution work with its own review surface.

## From ADR-009 (mrw counts what happens to the plans it is given)

- **A live-model benchmark: ask a model for a plan, grade whether it parses.** The direct answer to
  "can a model author this format", and deferred as the second move rather than the first. It needs
  an API key, costs money per run, cannot run in CI, and measures the model available on the day
  rather than the format. ADR-009's tally answers the same question from production for nothing;
  build this when the tally says WHICH parse failures dominate, so the benchmark knows what to probe.

- **A blind-agent bench: a fresh agent given only `mrw instructions` and a tree it has never seen.**
  PRE-REGISTERED 2026-09-24, before the harness exists (`.claude/rules/adr.md`). Criterion: on the
  nine read/plan tasks of the 2026-09-24 ADR-063 chaos pass, a fresh agent answers at least 8 of 9
  correctly, using at most 20 mrw calls in the whole run, on Haiku and on Sonnet, 3 runs each. A run
  that used a banned tool (grep, rg, find, ls, cat, sed, awk, head, tail, the Read/Grep/Glob/Edit/Write
  tools, or any `--help`) is VOID, not a miss, and is replaced by a fresh trial, at most 3 replacements
  per model. The verdict per model, first rule that applies: FAIL if any non-void run misses the
  criterion; otherwise INCONCLUSIVE if it has fewer than 3 non-void runs after its replacements;
  otherwise PASS. The bench passes only if both models PASS. The 2026-09-24 one-off (Haiku, 9 of 9) predates this
  criterion and is not a reading. Harness `scripts/blind-agent.sh` (scoring
  `scripts/blind-score.py`); plan and results under `docs/blind/`, first as blind reading 01.
  **Reading 03 (2026-09-24): PASS**, both models (`docs/blind/blind-03-result.md`). Readings 01 and
  02 are void on scorer defects. Haiku needed all three replacements, each VOID for `cat`.
  **Reading 04 (2026-09-24): FAIL**, both models, on the call budget: h3 and s2 took 21 calls with
  9 of 9 correct (`docs/blind/blind-04-result.md`). It removes reading 03's confound: headless
  `claude -p --safe-mode` under `/tmp`, so no repository documentation in context. Sonnet's calls
  roughly doubled (non-void runs, 23 → 49); Haiku's rose 45 → 55; correctness did not move.
  **Criterion amendment, pre-registered 2026-09-25 before reading 05's first trial** (ADR-070 T3):
  a non-void run must also make at least one mrw call to meet the criterion; 8 of 9 correct with
  zero calls did not use the tool the bench measures. `--help` stays banned as an argument word of
  any command; text that only contains it (`echo "see --help"`) is no longer a use. Re-scoring
  readings 03 and 04 under both changes moved no verdict.

- **`mrw instructions` does not show where `body=` goes.** It says what `body=` means but never shows
  it on an `@@` header. In blind reading 04, 4 of 7 trials put `body=` on a plan line of its own (3 as
  `body=N`), where it is body text: mrw wrote `body=1` into s2's file, and undoing it cost s2 six Bash
  calls, seven of its 21 mrw invocations.
  **Closed by ADR-070** (2026-09-25): `mrw instructions` and the MCP handshake show `body=` on a
  worked header, and a hunk with no count whose first body line begins `body=` is refused. Reading 05
  re-runs `blind-04-plan.md` on the build that ships it: deferred, on M's go (six or more headless
  `claude -p` runs).

- **The blind-bench scorer misreads several shell and answer shapes.** Two gaps in
  `scripts/blind-score.py` were found in blind reading 03:
  - `command_words` splits on an unquoted newline even after `\`;
  - it parses each heredoc body line as a command.

  The Codex review of PR #206 reproduced more:
  - `command cat` and a `cat` in `"$(…)"` escape the ban;
  - `env mrw` is not counted;
  - a banned word printed in a quoted argument after `;` voids a run;
  - a final non-object JSON fence is skipped;
  - a wrongly typed answer crashes the scorer.

  None changed a reading 03 verdict (`docs/blind/blind-03-result.md`, "Known limitations"). The
  criterion also sets no minimum number of mrw calls: 8 of 9 matching answers with zero calls would
  score MEETS. Every reading 03 run made at least 7 calls. **Reading 04 ran with the scorer
  unchanged, on purpose** (`blind-04-plan.md`), and none of these shapes appears in its transcripts.
  **Closed by ADR-070 T3** (2026-09-25): each shape is fixed and pinned by a synthetic transcript in
  `scripts/test_blind_score.py`. The quoted-word void was the raw `"--help" in cmd` test, which voided
  `echo "see --help"`. Readings 03 and 04 re-scored with no verdict moved.

- ~~**Teaching leads from blind reading 03, not yet acted on.**~~ **CLOSED 2026-09-27 — by ADR-084**,
  contract §166: `-M` in a write is refused naming `1-M`; `mrw instructions` teaches a write's exit 1
  and 2; the instructions, `read --help` and AGENTS.md say a bare directory prunes. A write's `-M` refusal is a bare parse
  error; a write's exit 1 and exit 2 are not taught; `--exclude` does not say it prunes a bare
  directory name. Each needs its own decision on whether `mrw instructions` should carry it, since
  the handshake and the CLI text are budgeted (`docs/blind/blind-03-result.md`, "Teaching leads").

- **A fixture corpus of recorded model-authored plans, graded hermetically.** Better than a live
  benchmark — repeatable, no key, runs in CI — and blocked on the same thing: somebody has to
  collect the corpus, and the honest source is the production signal ADR-009 adds. The tally tells
  us what the corpus should contain.

- **Shortening the per-hunk receipt for large plans.** SWE-agent's ACI work (arXiv:2405.15793) states
  "feedback should be informative yet concise to respect context limitations", and mrw prints one
  `ok`/`FAIL` line per hunk plus a summary — about 30 lines for a 27-hunk plan, on a tool whose whole
  pitch is context economy. Not obviously wrong: the per-hunk verdict IS the product, and collapsing
  it would be the silent-success shape this project refuses. What is missing is a measurement of what
  the receipt costs at scale before anyone trims it. Undecided, and ADR-009 explicitly does not
  decide it.

- **A cross-model comparison of authoring success.** Which models can emit the plan format and which
  cannot. Deferred from ADR-009-T3: one repository's tally is one population, and the parent record
  says so — a comparison needs several, which needs the fixture corpus above.

- **Changing the plan format in response to the reading.** ADR-009 pre-registers the criterion
  (parse refusals over 5% means the format is the problem, not the caller) and deliberately stops
  there. What to CHANGE is a separate decision, and making it before the number exists would be the
  formality the criterion was written to avoid.

## From ADR-008 (a delete says what it removed)

- **Requiring a guard on every `delete`.** ADR-008 makes a delete REPORT its
  bounds and lets it DECLARE an expected body, but leaves an unguarded
  `@@ f.go 5-8 delete` legal. Requiring `lines=` or `anchor=` was the runner-up
  and is genuinely close: it taxes every correct two-line delete to catch the
  rare wrong one. Revisit once the receipt bounds have been in use — if a wrong
  range still reaches a build after ADR-008, the tax is worth paying.

- **Requiring the expected body rather than accepting it.** The body is opt-in
  because it costs the caller the tokens of the lines being removed, which is
  worth it for a delete worth pinning and not for a two-line one. If plans in
  practice omit it exactly where it would have helped, that judgement is wrong
  and the default should move.

- ~~**`anchor=` reports its failure above the ledger check, so a failed anchor
  reads one line of a range the caller was never served.**~~ **CLOSED 2026-09-07 by ADR-028, FOR THE
  SPELLING THE LEDGER RECORDED** — the check moved below `covered()`, contract §66 drives all four
  anchored ops through the built binary, and the row was proved red against the released v1.4.0 tree.
  ⚠ It is NOT closed for an alias spelling: that misses `covered()` altogether, so the anchor still
  prints the line — see the alias entry below, which is the same defect one level down. The entry's
  own warning about the fixture was right and is why the test serves line 1 and anchors line 2. Kept
  because the reasoning below is what made it a defect rather than a judgement call. ADR-008 moved its own
  expected-removal comparison BELOW `covered()` for exactly this reason and
  pinned it with a test; its sibling one line up was noticed at the same time
  and deliberately left, so this entry exists to stop the asymmetry reading as
  an oversight. Reproduced 2026-09-01, `internal/apply/apply.go:453-456` — with
  `f.txt` served at line 3 only:

      FAIL f.txt 1 replace (plan line 1): anchor "nope" not in line 1: SECRET-LINE-ALPHA

  It fires on `replace` and `delete` alike, and is repeatable per address, so it
  is bounded by `clip`'s 60 characters per failed hunk rather than by one line
  in total. Two corrections to the reasoning that first justified leaving it,
  both from the PR #11 re-review. The gloss "the anchor is the caller's own
  text" does not hold for the half that matters: the anchor is theirs, the
  PRINTED LINE is the file's, and that is the part never served. In the other
  direction the blast radius is smaller than it looks — `--root` still refuses
  first, so nothing outside the tree mrw was pointed at is reachable. So this is
  a contract violation against ADR-002 and ADR-005's "mrw does not tell you what
  it has not shown you", not a privilege boundary: the caller could read the
  file directly. Fix by moving the anchor check below `covered()`, which is a
  one-line move plus the fixture that proves it — a fixture whose FIRST version
  must be checked against a mutant, because the obvious one trips the
  whole-file gate instead and passes with the ordering reversed.

- ~~**An ALIAS spelling of a partially-read file bypasses the per-line ledger
  ENTIRELY, so a write to lines the caller was never served APPLIES.**~~
  **CLOSED 2026-09-09 by ADR-029 T1+T2** —
  `internal/adversarial/ledger_test.go::TestAnAliasSpellingIsTheSameFileToThePerLineLedger`,
  contract §67. Ranked as open until 2026-09-12 because this receipt was never
  struck (M: *"I NAME THEM ALL"*). The write-back two-key SHA refusal further
  down this file is a different defect and stays open. Found by
  the Codex review of PR #128, reproduced 2026-09-07 against that branch at
  `1efd1a6`. `internal/apply/apply.go:510` recovers an aliased observation with
  `sameFileEntry` — that is issue #47's fix and it must keep working — but `:553`
  looks the ledger up again by exact key and `covered()` treats the miss as
  COVERED (`:555`), so every per-line check passes:

      $ mrw --root . read 'real.txt:1'                             # line 1 only
      $ printf '@@ link.txt 4 replace\nPWNED\n' | mrw --root . write -
      ok   link.txt 4 replace  -1 +1

  with `link.txt -> real.txt`; a case-only `REAL.txt` does the same on APFS and
  on NTFS. The exact spelling is refused, as it should be. **This is larger than
  the anchor entry above, which is one visible symptom of it**: for an alias,
  ADR-002's per-line guard is not weakened, it is absent.

  **Pre-registered before the harness exists**, because the obvious fixture
  passes for the wrong reason in two different ways.

  1. A test asserting only that the alias write is REFUSED is green against a fix
     that refuses every alias — which undoes #47. It must also assert that a
     WHOLE read of `real.txt` still licenses a write spelled `link.txt`.
  2. Neither alias is portable, and each is covered on exactly one of this
     repository's two CI operating systems. The symlink half runs on Linux and
     may fail to create on Windows; the case-only half needs a case-INsensitive
     filesystem, so it runs on Windows and cannot run on Linux. Probe at runtime
     and skip — never assert the platform — and say in the record that
     `scripts/contract.sh` can carry the symlink half only, being Linux-only.

  `--root` still refuses first: checked the same day on an in-root symlink
  pointing outside the root, where read and write were both refused by name. So
  this is a contract violation against ADR-002 and ADR-005, not a privilege
  boundary — exactly as for the entry above.

## Found by probing the built binary (2026-09-01)

Five were recorded here rather than fixed in passing, because each makes
something currently legal illegal. **Three are now closed — 2026-09-03 — and
are struck through below**; the two that remain are genuine either-way
decisions rather than defects. The bugs found in the same pass (an unknown
subcommand exiting 3, an absolute path reported as "does not exist", `sha=`
accepting non-hex, `$-1` reported as out of range, `--check` silently dropped
under `--dry-run`) were fixed then and carry contract rows.
- ~~**A DUPLICATED guard key silently discards the earlier one.**~~ **CLOSED
  2026-09-03 — refused at parse time.** It was the codebase's own principle
  turned inside out: `internal/apply/apply.go` says "a guard that is parsed and
  then discarded would be worse than no guard at all — the caller believes the
  edit is pinned", and the parser did exactly that. Re-probed 2026-09-03 and
  still true, so it was fixed rather than re-recorded: `anchor="NOPE"
  anchor="a"` applied at exit 0 with the false guard gone.

  **Refused, not resolved.** Two guards on one hunk are two different claims
  about one edit; picking either silently is how the caller keeps believing the
  other. Holds for `sha=`, `lines=`, `anchor=`, `body=` and `raw=`. It is a
  behaviour change — a plan that applied yesterday now fails — and the plan in
  question is one carrying a guard that was never checked. Asserted by
  `scripts/contract.sh` §31 and `TestARepeatedGuardKeyIsRefused`;
  `TestDistinctGuardKeysStillParse` pins that one of each is still fine.
- ~~**`create` with an EMPTY body succeeds and reports `ok`.**~~ **CLOSED — ADR-027**: a body-less
  `create` is refused ("say body=0 if you mean an empty file"), and `body=0` is the deliberate
  form. The history, as first written: ADR-006 refuses an
  empty-bodied `replace` because a body lost in transit — a truncated emission,
  an editor eating the last line — deletes code while the receipt says it
  worked. The same truncation on a `create` at the end of a plan produces an
  empty file and exits 0. The counter-argument is real: an empty file is a
  legitimate thing to create, and `touch` is not a mistake. So this is a genuine
  decision rather than an oversight, and it wants ADR-006's reasoning applied to
  it explicitly — probably `create` with no body being legal only when written
  as such deliberately, which the format has no way to say today.

- ~~**An ABSOLUTE path is silently reinterpreted as root-relative, and every
  surface prints the original.**~~ **CLOSED 2026-09-03 — fixed on every
  surface, verified by probe.** The write path gained the diagnosis in #36/#37;
  the READ path was closed by `rooted.IsRooted` (#38), which also fixed the
  Windows case where `filepath.IsAbs` is false for `/etc/hosts`.

  Verified 2026-09-03 against the built binary: `mrw read /etc/hosts` from a
  temp root now exits 1 with `==> /etc/hosts  REFUSED  /etc/hosts is outside
  the root <root>: read it with --root pointed where you mean`. The receipt no
  longer names a path the tool did not touch, and nothing outside the root was
  ever reachable. Asserted by `scripts/contract.sh` §26 and §27.
- ~~**`raw=true` without `body=` is accepted and does nothing.**~~ **CLOSED
  2026-09-03 — a usage error.** `raw=` only switches off the valid-header check
  INSIDE a counted body, so without `body=` it is a guard that cannot fire —
  the same class as the duplicate key above and fixed in the same change. The
  legitimate pairing (`body=N raw=true`, the escape hatch for a plan whose body
  contains a real `@@` header) is pinned by its own contract row, because
  refusing the useless form must not break the useful one.
- ~~**`--max-lines 0` means UNLIMITED, and this repository decided the opposite
  question the other way once already.**~~ **CLOSED 2026-09-07 by ADR-033** — zero
  is a cap of zero now, `Options.MaxLines` is a `*int` so omission still means no
  cap, and contract §69 drives both spellings. The reasoning is kept because it is
  what made this a decision rather than a bug report. It read: a cap of zero was
  indistinguishable from no cap, so there was no way to ask for the header alone
  THROUGH THIS FLAG — `--stat` always covered that need by another route — — and nothing is reported as withheld, though the README
  promises "whatever is withheld is always reported". The precedent cuts
  against the current behaviour: `body=0` in a plan means an EMPTY body, not an
  unbounded one, and `TestBodyZeroMeansAnEmptyBody` exists because treating an
  exhausted count as "keep scanning" silently handed a hunk lines the caller
  wrote for something else. `lines=0` is likewise a real assertion, not a
  disabled one. Against changing it: 0-means-unlimited is a widespread CLI
  convention, and `--stat` already serves the "header only" need. Either way it
  is a behaviour change, which is why it is here — negative values, which were
  silently ignored, are now a usage error and shipped with this round.

### Probed and found correct (2026-09-01, round two)

Recorded so the next probing round starts somewhere new rather than
re-measuring these. Each was driven at the built binary, not read:

- **A WHOLLY unparseable ledger fails closed.** Garbage written over the entire
  `seen` file leaves every write refused with "has not been read", including
  for files that genuinely were read. **Corrected 2026-09-01 after a Codex
  review**: the first version of this entry said "a corrupt or truncated
  ledger", which the measurement does not support. `internal/seen/seen.go`
  parses line by line and silently SKIPS malformed lines while keeping the
  valid ones, so a ledger holding a good record for `a.go` plus corrupt data
  still authorizes `a.go`. The fixture happened to corrupt every line. Whether
  a partially unparseable ledger should be an error rather than a silent
  partial read is an open question, and this entry is the only place it is
  written down.
  **DECLINED 2026-09-27 (the partial case).** A skipped line can only withdraw a licence, so the
  failure it causes is a refusal, never a wrong write; and an error would stop `Record` from ever
  rewriting the file, the trap the stale-header comment in `internal/seen/seen.go` describes. The
  wholly unparseable case keeps failing closed. Reopens on a skipped line that licensed a write it
  should not have.

- **Two concurrent writes to one file were not observed to both land, and that
  is weaker than a guarantee.** Racing two writes, the second was refused with
  "changed since mrw last saw it (recorded …, now …)". **Corrected 2026-09-01
  after a Codex review**: the first version claimed they "cannot both land",
  which the code does not promise. `internal/apply/apply.go` reads and
  validates the prior sha, then writes later, and `writeFile` renames
  atomically without a lock or compare-and-swap — so two processes can validate
  against the same original and both rename, last writer winning. The sha guard
  makes the race UNLIKELY and loud when it loses, not impossible. Closing it
  needs a lock plus an under-lock sha recheck.

  **Observed 2026-09-02, and one clause above is now false.** The race is no
  longer theoretical: 20 concurrent `mrw write` invocations against one 100-line
  file, each replacing a different line, three trials. Surviving edits varied
  1, 17 and 20 of 20 — so the loss reproduces easily on this machine. What the
  clause above gets wrong is "loud when it loses". It is not loud. A writer that
  lost printed a full success receipt and exited 0:

      ok   f.txt 24 replace  -1 +1
      wrote f.txt  100L -> 100L  sha 01a5e658
      1 hunk(s), 1 file(s), 0 failed — applied
      exit=0
      (W-8 absent from f.txt)

  The mechanism is last-writer-wins, confirmed by sha rather than inferred: each
  writer prints the sha of the file it wrote, and only the final writer's sha
  matches the file afterwards (writer 20 printed `b5db03f3`, which is what `mrw
  read` then reports). A losing writer's rename DID land and was superseded by a
  later one. The sha guard is racy, not absent — in the same run 6 of 20 were
  refused loudly with an explicit mismatch. So the accurate statement is: the
  guard makes the race unlikely, and SOMETIMES loud; when it loses it can lose
  silently, with a receipt that says applied. Locking was then out of scope
  (ADR-002); this only sharpened the risk that scope accepted.
  **Invalidated by ADR-075** (2026-09-25): writers take a per-checkout lock through apply and the
  ledger update, so none exits 0 having lost its edit; a stale one is refused (contract §150).
  The sentence that put locking permanently out of scope was withdrawn on 2026-09-26: it
  stayed here beside the invalidation, and a test asserting it stayed green (now
  `TestTheConcurrentWriteRiskIsClosedByADR075`).
- Pattern addresses, `/start/,/end/` pairs, pointer resolution (`@0`, `@-1`,
  `@abc`, `@N` out of range), overlapping and descending range lists, filenames
  with spaces, unicode and a leading dash, and a missing `$HOME` with no
  `XDG_STATE_HOME` — all correct, all with the right exit status.

## From a Codex review of PR #13 (2026-09-01)

- ~~**`{packages}` and `{files}` are substituted into `sh -c` WITHOUT quoting.**~~
  **CLOSED 2026-09-09 — `shellArgs` in `internal/check/check.go`**, ADR-003
  amendment, contract rows around §15. Ranked as open until 2026-09-12 because
  this receipt was never struck (M: *"I NAME THEM ALL"*). The reviewer's case
  (`pkg; true #/x.go` → `go test ./pkg; true #`, silent PASS) is the defect that
  shipped the quoting. Kept because the reasoning is what made it HIGH rather
  than a drive-by. `internal/check/check.go`'s `command()` once built the scoped
  command with a plain `strings.NewReplacer`, so every character of a derived
  path reached the shell as syntax.

- **A declared check can be inert, and mrw cannot tell.** `"check": "true"`,
  `":"`, `"# comment"`, `"$UNSET"`, `"VERIFY="`, or `"exit 7 | true"` all make
  `sh -c` return 0 without verifying anything, so `Ran` is true, `OK()` is
  true, and the verdict is PASS. The reviewer offered this as a gap in ADR-003
  rule 2; it is recorded here rather than fixed, because **it is a different
  question from the one that rule answers.** Rule 2 forbids mrw from inventing
  a pass for a check that did not run. It cannot make mrw judge whether a
  command the project declared does useful work — `make verify` may be empty
  too, and `internal/check/check_test.go::TestRunPasses` deliberately requires
  `true` to be OK. The whitespace case was a defect because nobody DECLARES
  `"   "`; it is an accident being read as a declaration. `"true"` is a
  declaration. What would change this is an enforceable check protocol — a
  command required to emit a signed or structured result — which is a much
  larger decision than a trim, and nothing here needs it yet.

## From the `$` divergence (2026-09-02, no ADR)

- **One address parser, not two.** `internal/plan.ParseAddr` and the range
  parser inside `internal/read` both read the same address grammar and drifted:
  `$` meant the last line to one and "unbounded" to the other, so `mrw read
  f.go:$` served the whole file while `@@ f.go $ replace` changed one line.
  Fixed inside `read` rather than by unifying them, because read's grammar is
  strictly richer — `/re/`, `/re/,/re2/`, comma-separated spans — and folding
  that into `ParseAddr` grows a second grammar inside it.

  This is ADR-006 rule 2's argument ("the boundary lives in one place;
  duplicating it would have re-created the divergence in a slower form") applied
  to addresses instead of the root, and the divergence it predicts has now
  happened once. Worth a record if it happens twice, or if read's grammar stops
  being the larger one. No decision needed yet.

- **A read over MCP buffered what the CLI streams — TAKEN, and the reason this
  entry gave for deferring it was wrong.** Superseded by ADR-011-T3.

  The measurement stands: an 18 MB file cost 44 MB via the CLI and 169 MB over
  MCP; ten of them cost 5.9 MB and 1268 MB. What was wrong was the conclusion.
  This entry said a cap "is a behaviour divergence from the CLI, which
  ADR-010's whole thesis is careful about", and left the decision open on that
  ground. Claude Code caps MCP tool output at 25,000 tokens by default and
  offers a per-tool override up to 500,000 characters; an oversized result is
  "persisted to disk and replaced with a file reference in the conversation",
  so the model never receives it as the file either way. The choice was never
  cap-versus-fidelity; it was refuse legibly, or pay the memory to build a
  result the host then takes out of the conversation.

  Worth keeping as a lesson about deferrals: the reason recorded here was
  plausible, internally consistent and unchecked, and it deferred the work for
  as long as nobody read the host's documentation. A deferral's REASON deserves
  the same scepticism as a claim in a record, because it is one.

- **The copy amplification itself.** ADR-011-T3 bounds the RESULT, which fixed
  the memory (2617 MB to 87 MB on the 40-file case, measured 2026-09-03), but
  a read still renders into a buffer, is marshalled into the result and
  marshalled again into the response. For anything under the limit that is a
  few hundred kilobytes and not worth the complexity of a streaming encoder.
  Revisit only if the limit rises far enough for the constant factor to matter.

- **Splitting the authoring tally by call source, CLI versus MCP — DEFERRED,
  with the reason written down so it is not re-proposed every month.** Raised
  and rejected in ADR-012.

  `cmd/mrw` and `internal/mcp` both call `authoring.Record` into one set of
  counters, so `mrw stats` cannot tell a plan authored by a shell user from one
  authored by a host. Splitting them is easy and it does not buy the thing it
  looks like it buys. The population worth measuring is a caller who has ONLY
  the tool description — no AGENTS.md, no README — and every caller in this
  checkout has read AGENTS.md. Splitting partitions the population we have; it
  does not create the one we want, and ADR-009's Out of Scope permanently
  refuses the transmission that would bring the other one's outcomes here.

  Revisit only if a source split answers a question someone actually has about
  THIS checkout — for instance whether MCP callers hit `refused_apply` more
  than shell callers, which is about the ledger and not about the format.

- **Measuring whether served size degrades edit accuracy — DEFERRED, and it is
  the most valuable unanswered question this project has.** Named in ADR-014.

  `MaxResultChars` is 200,000 because that fits under Claude Code's per-tool
  ceiling. It bounds a RESOURCE and says nothing about quality. What nobody has
  measured is whether a model's next edit gets WORSE as more context is served:
  the reported degradation with long context is about retrieval and QA, and edit
  authoring is a narrower, more mechanical task it has not been measured on.

  ⚠ The obvious instrument is the wrong one. ADR-014 records peak RSS against
  served bytes (about 12x the served size over MCP, flat on the CLI). That is
  what the SERVER spends. The question is what the CALLER's accuracy does, and
  using a memory curve to justify a context budget is the same category error
  ADR-012 rejected one level up — a measurement pointed at the wrong population.

  Answering it needs the benchmark harness: served bytes as the independent
  variable, edit outcome as the dependent one, across models. If the curve turns
  over, the cap becomes a measured number instead of an inherited one; if it is
  flat, "arbitrary" is a defensible answer and we will be able to say so.

  **PRE-REGISTRATION, written 2026-09-04 BEFORE any harness exists or any datum
  was collected.** It is here rather than in a record because a criterion
  authored after the first look is not a criterion.

  ⚠ THE DEPENDENT VARIABLE IS THE WHOLE PROBLEM, and the obvious ones are
  traps. ADR-009 already counts `applied`, `refused_parse` and `refused_apply`,
  so they are free — and they will be FLAT, because they measure whether the
  caller could author the FORMAT, which has nothing to do with how much was
  served. A flat curve on those would read as "the cap does not matter" when it
  actually means "the easy thing was measured". They are recorded as secondary
  and are not the claim.


  ⚠ AND "FREE FROM THE TALLY" NEEDED CHECKING, because ADR-012's Context
  records that tally as per-checkout and unable to ATTRIBUTE. Verified
  2026-09-04 before the design was allowed to depend on it: with a fresh
  `XDG_STATE_HOME` per trial, `mrw stats` reports that trial alone —
  `applied 1 of 1 plan(s)`, one plan recorded. So the secondary DVs are
  per-cell readable, and the isolation is the state dir rather than the
  checkout. If that ever stops holding, the fallback is parsing each
  `mrw_write` result directly, which is a different code path and should be
  named as such rather than quietly substituted.
  **PRIMARY DV: did the plan address the line it was supposed to address?**
  This is the failure mode that matters and the one every other gate is blind
  to — an edit that parses, applies cleanly, reports `ok`, and changed the
  wrong line. `refused_parse` and `refused_apply` both stay green through it.
  It is also the failure mrw exists to prevent, so it is the honest thing to
  put on the y-axis.

  GROUND TRUTH IS PLANTED, NOT JUDGED. The generator places the target line and
  therefore knows the correct address by construction; scoring is
  `plan addressed line N` against a known N, mechanically. Nothing here needs a
  human to rate an edit, which is what keeps the measurement affordable and
  repeatable — and it is the reason this DV was chosen over "is the edit
  semantically right", which cannot be scored without authoring a rubric and a
  rater.

  ⚠ THE THREAT TO VALIDITY, named up front: a planted target is easy to find if
  it is a unique string, and then the task measures string matching rather than
  reading. The distractors must be NEAR-IDENTICAL to the target — same shape,
  differing in the detail the instruction names — or the curve will be flat for
  a reason that has nothing to do with context size.

  IV: served bytes, varied by padding around the target on both sides.

  ⚠ "THE TASK IS HELD FIXED" WAS FALSE AS FIRST WRITTEN, and the correction
  matters because it is the difference between a clean manipulation and two
  variables moving at once. Padding adds LINES, so the line-number space grows
  and the address the model must produce changes by construction. It cannot be
  held fixed while size varies — the honest statement is that the INSTRUCTION
  and the target CONTENT are identical across cells, the address is not, and
  target position is a stratum rather than a nuisance.

  ⚠ DISTRACTOR COUNT IS A SECOND IV HIDING INSIDE THE FIRST, and this is the
  one that would make a real curve uninterpretable rather than merely absent.
  If the padding IS near-identical distractors, then larger cells have more
  near-misses and "more context" cannot be separated from "more candidates".
  So: the number of near-identical distractors is HELD CONSTANT across size
  cells, and the remaining padding is inert content that is obviously not a
  candidate. Distractor density is a legitimate second experiment; it is not
  this one, and one curve answers one question.

  STRATIFY BY TARGET POSITION rather than randomising it away. Position within
  the served window is the best-documented effect in the long-context
  literature, and if it is folded into the noise the headline curve will
  average a real effect into nothing. Early / middle / late are separate cells.

  REPEATS: the model is nondeterministic, so N trials per cell with the count
  fixed in advance. A cell is a proportion, and proportions need their interval
  reported, not a point.

  **DIRECTION AND THE FLAT ANSWER, committed now.** The prediction is that
  correct-address rate DEGRADES as served bytes grow, most sharply for
  mid-window targets. **If the curve is flat, that is a publishable answer and
  the cap stays arbitrary with evidence** — it does not become a reason to
  search for a different metric until something bends. ADR-009 and ADR-012 both

  **RECEIPT, 2026-09-04 — the instrument exists, the reading does not.** ADR-020 (PR #85) built
  the harness this pre-registration describes: `curve generate` / `score` / `tally`, no model
  client, scored by applying. **The FIRST READING is deferred to ADR-020's Follow-ups**, with the
  criterion above unchanged and not to be re-derived. This paragraph is the entry `adr-debt` looks
  for when the record's Out of Scope names this file.
  refused criteria that could not go red; this one commits to accepting a null.
- ~~**`occurrence=N`, or any positional disambiguator for a pattern address —
  DEFERRED.**~~ **CLOSED 2026-10-02 — ADR-118: `occurrence=N`, refused unless every match before the Nth has
  been served, which answers the objection below: the ledger is per sha, so the count is taken in the file as it
  is.** Raised and refused in ADR-013.

  When a start pattern matches several lines the hunk is refused and the
  refusal names the matched line numbers. The obvious next step is to let the
  caller say *which* one — `occurrence=2`. It is deferred rather than taken
  because it makes a plan depend on the ORDER of matches in a file, which
  changes when unrelated code moves above them: the address would then resolve
  somewhere else while still looking correct, which is the silent-wrong-edit
  class the exactly-once rule exists to refuse. The caller already has the line
  numbers from the refusal and can address by number.

  Revisit only if the ambiguity refusal proves common enough in practice to be
  friction rather than a guard — and note that nothing currently counts it, for
  the attribution reason ADR-012's Context sets out.

- **Widening the MCP root, or multi-root — DEFERRED, with a measurement behind
  it.** Named in ADR-016's Out of Scope; the analysis was done 2026-09-04 and is
  recorded here so the decision is not re-argued from scratch.

  The proposal that prompted it was sound in shape: repeatable `--root`,
  ALLOW-ONLY (a deny list needs symlink resolution and reintroduces every
  path-handling bug already fixed), resolved once at startup, boundary set by
  the launcher rather than by a file mrw can read and therefore widen, and one
  state namespace per root.

  ⚠ TWO THINGS CHECKED AGAINST THE CODE. `state.Dir(root)` already hashes the
  resolved absolute root, so N namespaces come free. But `apply.Apply(root, …)`
  takes ONE root and `seen.Load(root)` loads ONE ledger per run, so a cross-repo
  plan needs the ledger resolved PER HUNK. That is not a small generalisation of
  the existing invariant; it changes what a run is.

  THE MEASUREMENT, which is what actually decided it: the test proposed by the
  argument itself is "count how many recent plans would have wanted a second
  root". Over one full session of ~40 plans, the answer was ZERO. The single
  file touched outside the root was a one-file config edit — below mrw's own
  trigger threshold, so multi-root would not have earned it even then.

  So N registrations wins on this evidence, and issue #75 records
  one-registration-per-repo as the Claude Desktop shape. Revisit only for the
  case that would justify it: routinely changing a contract in one repo and its
  consumer in another, where the value is an ATOMIC plan across both. Note the
  cost if it is ever taken — an all-or-nothing plan spanning two git trees rolls
  both back on one failed hunk, which is the right semantics, but neither tree's
  history reflects the other.

- **⚠ THE MULTI-ROOT MEASUREMENT ABOVE WAS TAKEN ON THE WRONG POPULATION.**
  Correction filed 2026-09-04, hours after the entry it corrects.

  That entry concluded "N registrations wins" from counting plans in one
  session: ~40, all single-root, so a second root would have earned nothing. The
  count is accurate and the population is a CODER working inside one git
  checkout — which is the population that already has the CLI and can point it
  anywhere with `--root`.

  M has since named the population that matters for this question: a Claude
  Desktop user, an analyst reading and writing large documents into CSV. Their
  files are not in one repository. For them, "one fixed checkout chosen at
  startup" is not a safety property they trade away — it is the thing that stops
  the tool working at all, and they have no CLI to fall back to.

  So the earlier conclusion stands ONLY for coders with a shell. It is not
  evidence about Desktop, and it should not be quoted as if it were. The measure
  worth taking is on the Desktop population, and nothing has taken it.

- **MCP coverage for the Desktop population — the largest open product
  question in this repository.** M, 2026-09-04: *"we need wider MCP coverage,
  which basically calls local mrw under the hood … the potential here IS HUGE,
  for analysts reading and writing mega documents into csv files … humans aren't
  structured in their daily work … this is the single biggest feature that we
  ship only for coders but not desktop users."*

  WHY IT IS NOT JUST "ADD THE FLAGS". The capability gap and the reach gap are
  different problems with different answers, and only some of the CLI surface
  is even meaningful to this population:
  - `--grep` and `--files-from` are the ones that matter — finding the sites is
    exactly what an analyst cannot do by hand across many documents.
  - `--check` is a Go-test runner scoped to changed files; it means nothing for
    a folder of CSVs, and shipping it would be cargo.
  - `iter`, `seen`, `stats` are introspection a Desktop user has no use for.
  - The ROOT is the hard part, not the flags. See the correction above.

  WHAT TO DECIDE FIRST, before any of it: whether an analyst's answer is a wider
  MCP tool set, or one tool whose spec language already covers finding (mrw's
  read specs take a regexp) plus a root model that fits scattered documents.
  Those are different records. Needs its own ADR, with M's scope decision, and
  it should NOT be inferred from ADR-016 — that record deliberately covers only
  what the surface SAYS, and its refusal of parity is scoped to itself.

  **M ANSWERED THE SCOPE QUESTION on 2026-09-04: both, as two records.** The
  CAPABILITY half is done — ADR-017 shipped `grep` and `exclude` on `mrw_read`,
  with a match INDEX when the results will not fit and `after` to page it. It
  also settled `--files-from` permanently on the flag's own documented
  rationale: it exists to undo shell word-splitting, and MCP has no shell.

  **THE REACH HALF IS STILL OPEN AND IS FILED HERE.** ADR-017, its T1 and its
  T2 each defer "any root or multi-root change" to this entry, and this
  paragraph is their receipt — they pointed here first at a filename
  (`ADR-018`) that did not exist, which `adr-debt` correctly refused as a
  pointer to nowhere.

  What the reach record has to decide, carried forward so it is not re-argued:
  repeatable `--root`, ALLOW-ONLY (a deny list needs symlink resolution and
  reintroduces every path-handling bug already fixed), resolved once at
  startup, the boundary set by the launcher rather than by a file mrw can read
  and therefore widen, and one state namespace per root. `state.Dir(root)`
  already hashes the resolved absolute root so N namespaces come free, but
  `apply.Apply(root, …)` takes ONE root and `seen.Load(root)` loads ONE ledger
  per run — a cross-repo plan needs the ledger resolved PER HUNK, which changes
  what a run is.

  ⚠ **#81 IS DONE AND REACH IS NOT — THEY SEPARATED.** When #81 was filed this
  entry said it "belongs to that record too". It does not any more. ADR-018 took
  it, because the axis turned out to be EXPLICIT VERSUS ACCIDENTAL rather than
  which directory is unpleasant: `ResolveRoot` already returns a `Source`, so
  "did anyone say this?" is a value that exists. A fallback onto a filesystem
  root or the home directory is refused; anything explicit is honoured, `/`
  included. That rule says nothing about how MANY roots there may be, which is
  why it could ship without waiting for this entry.

  ⚠ **AND THE NUMBER MOVED.** ADR-017's Out of Scope, its T1 and its T2 all say
  reach is `ADR-018`. That was true when they were written and is now wrong:
  **018 is the root guard, and REACH IS ADR-019**. The deferrals point at this
  entry rather than at a number precisely so a renumbering cannot break them —
  which is the reason `adr-debt` refused the original `ADR-018` pointer as a
  pointer to nowhere.

  **RECEIPT, 2026-09-10 — ADR-019 drafted as Proposed.**
  `docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md` exists.

  **RECEIPT, 2026-09-10 — Naming pick: A — launch --root only.**
  M: *"A"*. Status Accepted. Fork 3: `mrw mcp` stays single-root. Do not
  measure Desktop `roots/list`. Do not implement B or C. T1 encoded the pick;
  T2 pins two-root isolation; T3 teaches launch `--root`. The historical
  analysis above is what the record carried forward; it is not deleted.

  **From ADR-019 Follow-ups, kept here so the record's deferrals have a receipt:**
  - A Desktop-population measurement of how many trees one session actually
    needs. The coder count is not that measurement.
  - Whether Claude Desktop sends `roots/list` — filed for pick C only. Pick A
    (ADR-019:108-110) wires no client request, so this is measured only if a
    reading ever argues for C. ADR-011 deferred the same client request here;
    this is that receipt too — the follow-up trigger (second host;
    `CLAUDE_PROJECT_DIR` host-specific) is met.
    Not run 2026-09-16 — this session is Cursor, not a Claude Desktop MCP session. Still not the coder count. Pick A stands.

  **Desktop reading 0 — 2026-09-24, this Mac. NOT the analyst measure.** Claude
  Desktop has `mrw` registered as `--root /Users/zy/GolandProjects/tool-multipathreadwrite
  mcp`. `~/Library/Logs/Claude/mcp-server-mrw.log`, 2026-09-04 → 2026-09-24:
  84 launches (`mrw mcp: serving` lines, all this one root), 34 `initialize`,
  1 `tools/call`, 0 `roots/list`, 0 `outside the root`. Population: one coder's
  machine, where the CLI is the tool of choice. It says Desktop reach is idle
  here; it says nothing about what an analyst session needs. Trees-per-session
  stays unmeasured. Reading 1 is one analyst task M runs in Desktop.

  **Desktop reading 1 — 2026-09-24, a Windows 11 Pro 10.0.26200 machine, Claude
  Desktop Store build, mrw v1.22.3 (`63729bd`, checksum matched SHA256SUMS). A
  CODER task, NOT the analyst measure** — the folder M picked was the
  quality-harness repository. Task: compare two ADR task folders in different
  branches of the tree and write a summary into a third place. From the
  Desktop log (`%LOCALAPPDATA%\Claude\Logs\mcp-server-mrw-docs.log`; the Store
  build's config lives under `%LOCALAPPDATA%\Packages\Claude_<id>\LocalCache\Roaming\Claude\`,
  not `%APPDATA%\Claude`): 3 `serving` lines, all one root; Desktop launched
  the server twice within 0.4 s and shut the first down before `initialize`;
  5 `tools/call` over ~66 s; 0 `roots/list`; 0 `outside the root`. Per-tool
  counts are NOT observable — this log format omits the tool name; the 4 reads
  + 1 write split is inferred from the output file's mtime. Trees this session
  needed: 1. Taken by a peer session on M's machine; reading 2 (an analyst task
  over documents) is still the open measure.

- **A heredoc-style body terminator for the plan format — DEFERRED.** Raised and
  refused in ADR-015.

  A body line beginning with `@@` needs `body=<n> raw=true`, and the friction is
  COUNTING the lines. `body=<<END … END` would remove the count. It is refused
  for now rather than taken because it is a genuine grammar addition that gives
  the format a second way to say one thing, and it does not remove `body=`
  anyway — a terminator can itself appear in a body.

  The judgement behind the deferral: what hurt was not counting lines, it was
  not being TOLD to. ADR-015 makes the refusal name the escape, so the evidence
  for a terminator is now "the hint landed and counting is still the friction".
  Revisit with that evidence, not before.

- ~~**Two `create` ops that collide on a case-insensitive filesystem — DEFERRED by
  ADR-021.**~~ **CLOSED — fixed by ADR-071 T2** (contract §141; see "Creates are not
  cross-checked" below, which also takes the same-path pair). `New.txt` and `new.txt` created in one plan have no inode to compare
  until one is written, so the identity check that refuses two spellings of an
  EXISTING file (`os.SameFile` at grouping time, ADR-021) cannot see them, and
  the second rename would win as before. Promote to a record when one such plan
  is seen in the wild; the fix needs either a case-fold belief about the
  filesystem, which issue #47 and ADR-021 both refused, or write-then-stat.
  ⚠ Beside it: two `create` ops for the SAME path in one plan are accepted today and produce a
  two-line file with both hunks `ok` (review of #89, 2026-09-04). Same spelling, so outside
  ADR-021's identity check — but it is a plan that "says two things" in exactly Decision 2's
  sense, and it belongs to whichever record takes the create collision.

- **The rules hook on Windows — DEFERRED by ADR-022.** `.claude/hooks/rules-on-read.py`
  is exercised by contract §55 on Linux and by its Go Enforced-by wherever `python3` is
  on PATH; the Windows runner may lack `python3` (the test skips there), and the hook's
  path handling (`os.sep`, `O_NOFOLLOW` absent, `realpath` on drive letters) has not been
  run on NTFS. Promote when a Windows contributor reports a delivery that did not happen,
  or when CI gains a Windows python3.

## From ADR-020-T2 (a target the instruction does not name)

- ~~**Run a reading against the relational fixture.**~~ **CLOSED 2026-09-05 by reading 3** (#96, `docs/curve/reading-03-result.md`): 45 of 45 under the pre-registered criterion, flat, and the prediction that the relational fixture is harder refuted (ADR-020 Follow-ups). T2 builds the selector; it does not spend the
  trials. The criterion is the one already pre-registered above and must not be re-derived — correct
  address rate against served bytes, stratified by position, refusals reported separately, a flat
  curve accepted as an answer. What the first reading adds is that the NAMED fixture is at ceiling
  (42/45, `docs/curve/reading-02-result.md`), so a relational reading is the first one whose curve
  has room to bend. It is a budget decision, which is why ADR-020 keeps readings out of its tasks.
- **A served window that does not begin at line 1.** All three misses of the first reading named the
  line exactly two below the target, and the reading cannot say why: every cell serves `@@ 1-N`, so a
  row count in the served rendering and the target's line number plus two are the same integer in all
  45 trials. A cell served from, say, line 500 separates them by 498 and settles it in one trial.
  Deliberately NOT in T2: it answers a different question from "can the task be failed", and folding
  it in would put two claims under one fence. Promote it when a reading needs to explain a miss
  rather than count one.

  **RECEIPT, 2026-09-05 — promoted and spent.** ADR-020 T4 built the cell (`-from`); reading 5
  (`docs/curve/reading-05-result.md`) served 15 trials from line 120 and every miss sat at
  `target − 117`: the row count. Whose count — the read arm's file reader's, which the transcript
  suggests, or mrw's own rows — was settled by reading 8 (`docs/curve/reading-08-result.md`): the read arm's. See the ADR-020-T4 entry below.

## From ADR-020-T4 (a served window that does not begin at line one)

- **The read format, if the row-count account holds — CLOSED 2026-09-05, no engine change.** Every
  miss in 135 read-arm trials sat at `target+2`, and mrw's served rendering opens with two rows that
  carry no line number, so the candidate was that the tool's own output induces the miss. Reading 5
  (`docs/curve/reading-05-result.md`) confirmed the row-count account — every miss is the row index
  of the served text, −117 with the window from line 120. Reading 8
  (`docs/curve/reading-08-result.md`) delivered the same client the same cells as a Bash tool result,
  mrw's gutter the only gutter, and it scored 15 of 15 with no plan at the row index: the count was
  the read arm's file reader's, not mrw's. No served-path record, no change to the header rows.
  What remains documented, not fixed: a client that saves mrw's output to a file and reads it back
  through a numbering viewer recreates the collision.

## From ADR-023 (a read's answer is the served text)

- **ADR-023: other hosts.** The host measurement behind ADR-023 is Claude Code 2.1.261 only: a
  successful result with `structuredContent` reached the model as the structured half alone. Claude
  Desktop and other hosts were not measured; the two host issues (#55677, #15412) say Claude.ai and
  ChatGPT render both halves. Worth one probe each when a Desktop or third-party session is at hand,
  recorded beside ADR-023's Verification Log. Not blocking: the bare envelope is right for a host that
  shows either half.

- **A page mrw serves can be truncated by the HOST before the model sees it, and the ledger is
  written anyway — ADR-002 inverted.** Measured 2026-09-05 on Claude Code 2.1.261 against the
  `d6c62e7` build, staging reading 18 (`docs/curve/reading-18-result.md`). A bare-path `mrw_read` of
  a 3,619-line fixture returned ADR-014's first page — lines 1-2727, `isError: true`, a correct
  `-- PARTIAL:` notice and a `next_read`. What reached the model was lines 1-90, a marker reading
  `[141140 characters truncated]`, and lines 2644-2727: the middle was discarded by the host, not by
  mrw, and the phrase appears nowhere in this source. `internal/mcp/tools.go:535` then records the
  page because "the page WAS shown" — the span it served, lines 1-2727, since `tools.go:522` builds
  a narrowed spec for the page rather than recording the file whole. (An earlier version of this
  entry said `mrw seen` claimed `lines 1-3619`; the code does not support that, and it does not
  matter to the defect: line 1500 is inside 1-2727 either way.) A plan replacing line
  1500 — inside the discarded middle, shown to nobody — applied with `"status": "ok"` and exit 0.
  So mrw edited a line its caller had not seen. It is issue #109's class by another route: ADR-023
  removed an envelope that replaced the served text; here the text survives mrw and is cut after it,
  with the ledger already written.

  What is NOT established: where the host's limit lies, or whether it is a property of the host at
  all. The figure `200,000 - 141,140 = 58,860` that this entry carried until 2026-09-06 was
  **arithmetic from the wrong base and is withdrawn**: `internal/mcp/tools.go:444` sizes a page at
  three quarters of the cap, so mrw sent ≈150,000 characters, not 200,000, and the remainder for
  that observation is ≈8,860 — which the observation itself corroborates, since 174 delivered lines
  at ~55 characters each is ≈9,600. It was one result either way, never a measured boundary.

  **What triggers it is the PAGED SHAPE, not the size. Measured 2026-09-06 on Claude Code 2.1.263,
  `claude-haiku-4-5` subagents, one bare-path `mrw_read` per fixture:**

  | fixture | served chars | mrw paged? | host truncated? |
  |---|---|---|---|
  | 2,048 lines | 131,136 | no | no |
  | 2,560 lines | 163,904 | no | no |
  | 3,072 lines | **196,672** | no | **no** |
  | 3,328 lines | 213,056 → page of **152,320** | yes | **YES** |
  | 4,437 lines | 284,032 → page of ~152,000 | yes | **YES** |

  A 196,672-character ordinary result arrived WHOLE. A 152,320-character PAGED result was gutted —
  44,000 characters smaller, and it is the one that was cut. So size does not explain it, and neither
  does the consuming model: the same Haiku subagents took the larger result intact. What the cut
  results have in common is that ADR-014 marks a page with **`isError: true`** and a `-- PARTIAL:`
  notice. Hosts truncate error-flagged tool results aggressively, keeping a head and a tail: the
  remnant here was ~150 lines ≈ 9,600 characters, and reading 18's was 174 lines ≈ 9,600, two CLI
  versions apart.

  **The trigger is in our own code, and it is now CONFIRMED by a controlled A/B.** `isError: true` on
  a SUCCESSFUL partial read is the collision: to a host the flag means "this call failed", while
  ADR-014 uses it to mean "there is more", and hosts truncate failed results head-and-tail. mrw was
  rebuilt with the paging path returning `isError: false`, the MCP server restarted, and the same
  fixture re-read by the same class of consumer:

  | `isError` on the page | served chars | what the consumer received |
  |---|---|---|
  | `true`  | 152,594 | **GAPPED** — line 78 then line 2309; ~150 lines of 2,380 survived |
  | `false` | 152,594 | **CONTINUOUS** — first line 1, last line 2380, no gap |

  Nothing else changed between the two runs. Read off the wire directly (a JSON-RPC client driving
  `mrw mcp` over stdio, not a model's account), the page still carries its notice in the served TEXT
  — `-- PARTIAL: lines 1-2380 of 3328. 948 line(s) remain.` — and simply omits the flag. So the
  visibility ADR-014 wanted survives, in the place a model actually reads, and the truncation stops.

  **The flag added to make partiality visible was making the page's middle invisible.** That is the
  whole defect, and `internal/mcp/tools.go:535` recording the page as seen is what turned it into a
  licensed write.

  **This is DONE, as `ADR-024: A page is known by its served text, not by an error flag`.** The
  record was accepted 2026-09-06 and executed the same day: `pagedResult`, `indexResult` and the
  served-read path that passed `problems > 0` all return the flag absent, `errorResult` keeps it,
  and contract §62 drives the shape through the built binary. ADR-024 formally invalidates the
  clause of ADR-014's Decision 2 that added the flag, and the matching assertion in ADR-017's
  Enforced-by test — ADR-017's own Decision never mentioned the flag.

  ⚠ **Executing it widened the class twice, and both widenings came from enumerating rather than
  recalling.** ADR-024's first enumeration used `awk '/IsError: *true/'` and found three sites; it
  missed `readResult`'s `IsError: isErr` parameter, which its callers at the time all passed
  `problems > 0` (only the deliberately retained `:202` branch still does). So an
  ORDINARY multi-file read that served content beside one unreadable path came back flagged — no
  oversized file needed, which makes it the most exposed member of the class and the one nobody was
  looking for. The corrected command is `grep -n 'IsError' internal/mcp/tools.go`, deliberately the
  broad one. The lesson generalises: a class-enumerating command has to be checked against what it
  CANNOT match.

  **ADR-024 deferred two obligations here by name; both are closed, each below.**

- ~~**The ledger records what was SENT, not what was SEEN** (`internal/mcp/tools.go`, the `seen.Record`
  calls).~~ **CLOSED 2026-09-07 by ADR-031 FOR PAGED READS** — a page is held pending against
  bracketed checkpoints and reaches the ledger only when the caller echoes them. The half for a read
  that FITS, then recorded on serve, closed 2026-09-09 by ADR-039 — its entry below. Kept
  because the reasoning is what made the class visible. Deferred from `docs/adr/ADR-024-a-page-is-known-by-its-served-text.md`, whose Decision 4
  says plainly that it narrows the exposure without removing the class: mrw cannot observe truncation
  from inside the server, a cut result and a whole one being identical to it. `anchor=` is the
  candidate echo-back — it already exists, and a caller that never saw a line cannot reproduce its
  text, which turns "did you see it?" from a server-side belief into a checkable claim.
  Deferred again, unchanged, from `docs/adr/ADR-025-a-read-that-served-nothing-is-an-error.md`, which
  narrows the flag on the served-read return and touches no ledger code.

- ~~**`MaxResultChars` is one host's ceiling hardcoded into a general-purpose tool.**~~ **CLOSED 2026-09-07 by ADR-032** (#135), contract §70: `mrw mcp --max-result-chars N` or `MRW_MAX_RESULT_CHARS` sets the ceiling, zero means zero, and it bounds the whole encoded result on both tools. `schema.go` says
  so itself — "The value is Claude Code's per-tool ceiling" — while mrw runs under any MCP host.
  Deferred from ADR-024, which explicitly does not move the number. The proposed shape is a
  caller-set knob (`MRW_MAX_RESULT_CHARS`, `mrw mcp --max-result-chars N`) with
  ⚠ **NOT `0` = no limit** — that shape was proposed here before ADR-033 settled
  the same question for `--max-lines`, and M chose zero-means-zero on 2026-09-07;
  omitting the knob is how a caller asks for the default — with `_meta`'s
  `anthropic/maxResultSizeChars` advertising the configured value rather than a
  constant.
  ⚠ The cap is NOT ceremony and must not simply be deleted: `internal/mcp/tools.go` records that an
  uncapped 40 × 18 MB read peaked at 2.6 GB and that the cap brought the same request to 87 MB,
  measured 2026-09-03. What a knob changes is WHO chooses, not whether the guard exists.
  Deferred again, unchanged, from `docs/adr/ADR-025-a-read-that-served-nothing-is-an-error.md`, which
  changes which answers carry `isError` and moves no ceiling.

  Reading 20 measured the same arm at 2 KB and 20 KB, where no paging and no
  truncation occur, and found 30 of 30 (`docs/curve/reading-20-result.md`); readings 12, 18 and 19
  voided on the way there. So the arm is measured BELOW the truncation point and unmeasurable AT it:
  the 200 KB case was the host-truncation defect, closed by ADR-024 and ADR-031, and what stays open
  is the under-ceiling host-cut measurement, filed under "From ADR-032" below.

## From five sessions field-testing the multi-line-body hazard (2026-09-06)

Reported by a peer session working in another repository, then field-tested on `f732fed` by four
sessions across markdown, Python/JS, PHP/Blade and YAML/Ansible; a fifth (React/TSX) confirmed its
binary and contributed an observation without running the test. The guidance those reports produced
is in `AGENTS.md` and `README.md`. What is recorded HERE is the one thing they reopen.

- **Should `mrw write` echo the written region, padded, after a write?** Rejected once already, on
  COST rather than principle: the caller wrote the plan and can re-read the file, so an echo restates
  what is reachable, and it would pay bytes on every write for a rare case. Four field reports change
  the inputs to that arithmetic and are recorded so the next reader can redo it rather than inherit
  the verdict.

  What is new. **The damage is never in the lines the plan named** — a surviving closer sits below
  the body by construction, and a short address orphans lines above it — so the ONE region a caller
  naturally re-reads is the one region that cannot show the problem. Measured offsets: Blade replaced
  7 wrote 7-9 orphan at 10; HTML `</div>` replaced 5 wrote 5-7 orphan at 8; YAML block scalar
  replaced 8 wrote 8-10 orphan at 11; markdown fence replaced 333 wrote 333-336 orphan at 340; and a
  34-line body at `3104-3108` where `3088-3108` was meant left sixteen dangling lines above.

  And **the automated gates were blind in three of the four stacks**: `yamllint`, `ansible-lint
  --profile production` and `ansible-playbook --syntax-check` all passed YAML whose meaning had
  changed; `php -l` passes a broken `.blade.php`; and a React repo's `vite build` does not type-check
  at all while its pre-push gate covers eight crash codes. `--check` cannot reach a file no test
  exercises, which is most templates. So "the caller can check it themselves" is weaker than it was
  when the idea was rejected.

  What has NOT changed is the design line, and it is the reason this is a question rather than a
  plan: mrw guards body content that breaks ITS OWN parse (`@@` needs `body=<N> raw=true`) and models
  no target syntax, because a checker for fences is a checker for braces is a checker for Blade
  directives, and that ends the property that one line-oriented editor takes Go, shell, markdown,
  JSON and YAML hunks in a single all-or-nothing plan. **A padded echo is not a checker** — it prints
  lines and understands none of them — so it does not cross that line. Whether it is worth its bytes
  is the open question. Anyone taking it up needs a record, an opt-in shape (a flag, not a default),
  and a contract row.

  **Receipted 2026-09-13 as ADR-052** — M said *"echo, license"*. Opt-in `--echo-pad` /
  `echo_pad` (default 0); a multi-line replace without a served line after End is refused.
  Still not a checker. Follow-ups that 052 does not catch (Zeus, 2026-09-13) are
  under **From ADR-052** at the end of this file.

- **PROBED 2026-09-16, still not a finding: a balanced-but-wrongly-nested JSX
  subtree.** Predicted by the React session to be valid TypeScript that renders differently — `tsc`
  green, DOM wrong — which would be the JSX analogue of the YAML case where a body at the wrong
  indent silently reparents keys and every linter stays green. Its author declined to speculate and
  did not run it; the session was briefing-only and correctly treated a field test as unrequested
  scope. It is here as a probe someone could run, not as a finding. If it reproduces it is the worst
  shape reported so far, because the file stays valid in a language whose type checker is the one
  gate that was expected to work.
  Probed 2026-09-16: `tsc --jsx react-jsx --strict` exit 0; `renderToStaticMarkup` parent of `#inner` was `#accidental-wrapper` not `#intended-parent`. Fixture `docs/break/jsx-nest/`. Still not a parser. Needs a quote for any record. Still not as a finding.

## From ADR-026 (an address may say how many lines follow)

- ~~**A backwards relative address, `A,-N` or `-N,A`.**~~ **DECLINED 2026-09-27** — no caller has
  asked. Deferred from ADR-026, which implements the
  forward form `A,+N` only. The backwards form is the one a caller wants after a match — *"show me
  the five lines that led up to this"* — and `--grep -C N` already answers that for a WALK but not
  for a named address. It is not free: `-` is the range separator, so `5,-2` has to be
  disambiguated from `5,-` and from a `-2` that never meant anything, and the refusal wording has
  to stay understandable when the caller wrote the ambiguous one. Anyone taking it up needs a
  record, both parsers changed together for the reason ADR-026 gives, and a contract row that pairs
  the good case with the ambiguous one.
  Reopens when a caller asks for it by name. It is cheaper than this entry assumes: the forward form
  is now parsed in one place, `addr.CutRelative` (`internal/addr/addr.go`), which both `read` and
  `plan` call.

## From ADR-027 (an empty file is created on purpose, or not at all)

- **The handshake `instructions` do not teach the body-less `create` refusal.** The `mrw_write`
  tool DESCRIPTION does, as of 2026-09-08, and the ADR-012 row asserts it through the built server.
  The handshake does not, and this entry is why rather than an oversight.

  **Measured, not assumed.** `maxInstructionsChars` is 4096 BYTES and the document sits at 4,095 —
  one byte of headroom. The shortest honest clause is 71 bytes, so it has to be funded by cutting
  something already there. Every candidate cut was tried: trimming the surface-choice paragraph to
  "prefer this with none, or to serialize writes" fits, and turns
  `TestTheSurfaceSaysTheCLIIsRicher` red on two assertions — that this surface *"serializes ledger
  writes"* and serves *"ONE fixed checkout"*. Those claims are guarded because they are the reason
  to choose this surface at all.

  So the choice is not "add a clause" but "which guarded claim is worth less than this one", and
  that is a decision about what a handshake paid for by EVERY session should carry — the question
  `maxInstructionsChars`'s own comment says the bound exists to force.

  **What would promote this:** an MCP caller reporting a bare `create` refusal they had no warning
  of, or any other change that frees 71 bytes there for an unrelated reason.

- ~~**Requiring `body=N` on every op, not just as an opt-in guard.**~~ **DECLINED 2026-09-27**, for
  the reason below. ADR-027 reuses `body=0` as the
  way to SAY "deliberately nothing" for `create`, which works because the count already exists and
  already means it. Making the count mandatory everywhere is the larger version of that idea: it
  would close the lost-body hazard for every op at once rather than one at a time, and it would cost
  a token on every hunk anyone ever writes. Nobody has asked for it, and the three ops that could
  lose a body silently are all closed without it. Anyone taking it up needs a record and a measured
  reason, not a symmetry argument.
  Reopens on one measured case of a body lost in transit on an op that is not already closed.

- ~~**A general "the engine re-validates everything the parser does" pass.**~~ **CLOSED 2026-09-07 by ADR-030** (#130): the enumeration walked `plan.validate` branch by branch, every branch has an engine counterpart, and `TestTheEngineAndTheParserRefuseInTheSameWords` compares their wording. Deferred from ADR-027-T3.
  Two records in a row have found the same hole one field at a time: `plan.validate` protects the
  CLI, the MCP server and the curve scorer because each calls `plan.Parse`, and a direct
  `apply.Apply` caller reaches none of it — ADR-026 for a relative end on an op that cannot honour
  one, ADR-027 for a `create` carrying no body. Each was fixed where it was found. The general
  question is whether `Apply` should re-assert every parser rule, which it cannot do by calling
  `internal/plan` without inverting the dependency the two packages are split to keep. Anyone taking
  it up needs a record, and the honest first step is enumerating what `validate` checks that `Apply`
  does not, rather than assuming the list is those two.

- **A successful alias-spelled write leaves ONE file recorded under TWO ledger
  keys, and the recorded spelling then gets a refusal that is not true.**
  Measured 2026-09-07 while writing ADR-029, against the v1.4.0 binary. With
  `link.txt -> real.txt`, reading `real.txt:1-3` and writing `@@ link.txt 2`
  succeeds and `mrw seen` shows both:

      4c650896  the whole file            link.txt
      880553fc  lines 1-3                 real.txt

  A later plan naming `real.txt` is then refused with

      real.txt changed since mrw last saw it (recorded 880553fc, now 4c650896):
      re-read it before editing, or pass --force to overwrite blind

  which says something false — mrw made that change itself, one call earlier.
  `sameFileEntry` does not help here: the exact key IS present, so no alias
  recovery runs, and the SHA comparison is against the stale entry.

  **Not fixed by ADR-029, and deliberately.** That record resolves identity for
  the per-line gate on the READ side; this is the WRITE-back side, where two
  keys are created rather than one consulted. It fails SAFE — it refuses, it
  never applies — so it costs a caller a re-read and a confusing sentence rather
  than a wrong edit. Fixing it means either reconciling SHAs across aliases in
  the file-level check, or keying the ledger on resolved identity, which
  ADR-029's Alternatives rejects because `mrw seen` prints the keys the caller
  typed. Promote it if a caller reports the refusal in the wild; the entry
  exists so the next reader does not mistake it for part of ADR-029.

- ~~**Typed error kinds shared by `internal/plan` and `internal/apply`, so the two
  refusal sites can be compared without matching message text.**~~ **CLOSED 2026-09-27 — ADR-087:
  `internal/refusal` kinds on every mirrored rule, on `plan.ParseError` and `HunkResult.Kind`, and
  the MCP acknowledgement remedy keyed on `refusal.NotRead`; the kind in a receipt is the entry
  below.** Deferred from
  ADR-030, which asserts every parser rule again at the engine boundary and keeps
  the two honest by copying `plan.validate`'s message strings VERBATIM and
  comparing them by equality. That works and it is what the test checks, but it
  is STRING equality, checked at run time for all ten verbatim-mirrored branches:
  `TestTheEngineAndTheParserRefuseInTheSameWords` parses each malformed plan,
  takes the expected text out of the parser's own error, and compares it to what
  `Apply` says for the equivalent Input, so rewording either site alone goes red.
  (An earlier draft of this entry said the pairing was only prose, which was true
  of ADR-030's FIRST cut — the table test hardcoded the strings and never invoked
  the parser. The review of PR #130 found that, and the cross-site test is the
  fix.) Typed kinds would still be better: they would make the pairing structural
  rather than textual, and they would survive a deliberate rewording of both
  sites, which string equality cannot tell from a drift. It is the same request ADR-009's open follow-up already
  makes for classifying refusals without matching text, so whoever takes one
  should take both.

- **A refusal kind in the JSON receipt.** Deferred from ADR-087, which gives the mirrored and
  not-read refusals a `refusal.Kind` but keeps `HunkResult.Kind` at `json:"-"`: a receipt field is a
  public contract, and no caller has asked to classify refusals. ADR-009's tally counts plans, not
  refusals, so it does not need one either. Adding it is one struct tag plus a contract row; promote
  it when a caller asks, or when the tally is asked WHICH refusals dominate.

- ~~**Checkpoints on small reads that fit whole.**~~ **CLOSED 2026-09-09 by ADR-039** (#157), contract §77: an MCP `mrw_read` that fits carries the same `-- ck` brackets and licenses nothing until acknowledged; a CLI read still licenses on serve (ADR-039 T2). ⚠ Promoted WITHOUT the evidence named below — an observed truncation of a non-paged answer — and that measurement stays open under **From ADR-032**. Deferred from ADR-031, which
  interleaves `-- ck` markers only into reads that PAGE. A read that fits in one
  answer is still recorded on serve, so the same host truncation would license
  lines nobody saw — the class is narrowed, not closed, and this entry is the
  only place that is written down. Against doing it now: every read would grow
  by TWO marker lines per 200 lines — spans are bracketed, open and close and every caller would have to acknowledge
  every read, which is a large tax for a case nothing has yet measured. The
  evidence to promote it is one observed truncation of a NON-paged answer; the
  measurement that produced ADR-031 was of a paged one (`docs/curve/reading-18-result.md`).

- ~~**`MaxResultChars` as a caller-set knob, bounding the whole encoded result.**~~ **CLOSED 2026-09-07 by ADR-032** (#135), contract §70 — M's shape as recorded here: caller-set, the whole encoded result bounded, the write cap enforced, `0` means zero.
  Chosen by M on 2026-09-07 and deferred out of ADR-031 so that record stays
  about the ledger. It is one host's ceiling (200,000) hardcoded into a
  general-purpose tool; `mrw_write` also advertises a cap it does not enforce.
  M's decision: make it caller-set, bound the ENTIRE encoded result rather than
  the served text alone, enforce the write cap, and explicitly do NOT give `0`
  the meaning "unlimited" — `0` means zero, per this repository's own precedent
  in `body=0` and `lines=0`. Its own record; nothing here blocks it.

- ~~**`--max-lines 0` means UNLIMITED, and M decided on 2026-09-07 that it should
  mean ZERO.**~~ **CLOSED 2026-09-07 by ADR-033** (#133), contract §69. The struck entry above states the
  question and the precedent — `body=0` is an empty body, `lines=0` is a real
  assertion — and M's answer is consistency with those, making "serve the header
  and nothing else" expressible. It is a breaking change for anyone passing `0`
  to mean no cap, so it needs an ADR, a contract row and a line in the README's
  flag table. Its own record.
- **Does the TURN COUNT dominate token cost, and if so does mrw win on tokens
  even against a windowed read?** Raised 2026-09-07 by the session working on
  atvirokodosprendimai.github.io while rewriting the landing page around
  `scripts/measure.sh`, and filed here rather than left in a chat log because it
  would change what this repository can honestly claim.

  The mechanism is not in doubt: every tool call is a fresh request carrying the
  whole conversation so far, so the Nth read pays for the N-1 before it.
  Cumulative input across a loop is closer to `N x base + R x N(N+1)/2` than to
  `N x R`. `measure.sh` reports R — the bytes one read puts in the context —
  and says nothing about the exponent. It says so in its own header, which is
  why the number is honest as far as it goes.

  ⚠ **The confound is large enough to reverse the answer, not merely shade it:
  prompt caching.** The re-sent prefix is charged at a fraction of fresh input,
  and the discount lands exactly on the quadratic term the argument rests on.
  "Turns dominate" and "turns are nearly free after the first" are both
  plausible readings of the same mechanism; which holds is an empirical question
  about cache hit rates in a real loop.

  **Why it is worth the work.** The windowed baseline reads fewer bytes per read
  but takes MORE turns — 105 against 2 in shape D. Those pull in opposite
  directions and the turn count is the one that compounds. If the quadratic term
  survives caching at all, mrw may win on cumulative tokens against the
  disciplined windowed reader, which is the one case where the per-read column
  says it LOSES 6.1x. That would invert the honest weakness this repository
  currently publishes.

  **Which is why nobody may state it until it is measured.** A claim that large,
  that convenient and that unmeasured has the same shape as quoting only the
  whole-file baseline: true under one accounting, stated as if it were the
  accounting. Better to carry a weakness we can prove.

  What a real measurement needs, and why it is not another shell script over
  file sizes: actual API token accounting across a scripted N-turn loop, or a
  defensible model of cache behaviour, for both the whole-file and the windowed
  baseline against mrw's two calls. `docs/curve/` is the closest existing shape.

## From ADR-032 (the ceiling is the caller's, and it bounds the whole answer)

### A read that FITS could record on serve rather than on acknowledgement

✅ **Accepted and executed as ADR-039 (2026-09-09) by M — *"accpted"***. The licensing half — a
fitting MCP serve recording on serve — is
`docs/adr/ADR-039-a-fitting-read-licenses-only-what-came-back.md`. M said *"ok, start"* on the
uneven-proof recommendation. The BACKLOG cost argument (a second round trip on every read) is
wrong for the 2-call recipe: `ack` already rides on `mrw_write`.

The measurement that would have settled it remains open below. Do not treat ADR-039 as that
evidence.

### A host-truncation measurement of an under-ceiling result

A host measured truncating a result that was UNDER the advertised ceiling. Reading-18 measured
a paged cut (Claude Code 2.1.261, 2026-09-05, `docs/curve/reading-18-result.md`). File an
under-ceiling measurement beside that reading rather than treating ADR-039 as it.

Deferred by `docs/adr/ADR-039-a-fitting-read-licenses-only-what-came-back.md`, Out of Scope.
Observed 2026-09-16: wire (`mrw mcp` stdio) fitting+paged continuous, `isError` absent; model check not run (Cursor, no mrw MCP tools). Class still open. File: `docs/curve/reading-under-ceiling-result.md`. This is not ADR-039.

## From the Codex review of PR #136 (2026-09-07)

### mrw's per-root state accumulates without bound, and the contract script is its heaviest producer

mrw keeps one state directory per ROOT it has ever been pointed at, outside the
tree by ADR-004. Nothing prunes it, and nothing needs to for ordinary use: a
developer has a handful of checkouts and the entries are tiny.

`scripts/contract.sh` is a different population. It isolates fixtures by giving
every one a FRESH root — a deliberate decision, stated at `contract.sh:1587`, and
the right one for isolation — so a single run mints dozens of roots that will
never be seen again.

**Measured 2026-09-07 on the maintainer's machine: 22,613 directories, 240 MB,
every one of them created that day.** Found while fixing the same leak in
`measure.sh`, which now pins `XDG_STATE_HOME` into its own disposable area.

Why it is not fixed here: pinning `XDG_STATE_HOME` in `contract.sh` is the same
one-line change, but that script is the repository's gate and several of its
sections assert things about where state lives — rows read `m seen` to find the
tally, and one deliberately asks mrw for the state path rather than guessing at
XDG layout. Changing the variable under them needs each of those rows read, not a
global edit, and it does not belong in a PR about rounding.

What a fix should decide, and it is a real question rather than an oversight: is
this the SCRIPT's problem or the TOOL's? A state directory per root with no
expiry is a design mrw chose; if an agent points mrw at temporary checkouts all
day — which is exactly what a test harness or a CI job does — it grows for ever
and nothing tells anyone. A prune, an age-out, or a documented "this is yours to
clean" are three different answers and only one of them is a script change.

**ANSWERED by ADR-034**, which chose the prune: `mrw seen --prune` removes the
entries whose `root` marker names a checkout that is gone, explicitly and never
automatically, and `contract.sh` pins `XDG_STATE_HOME` into the `$WORK` its trap
already removes. The question above is left as it was asked, because the reason
it was a real question — and not an oversight — is the argument ADR-034's
Alternatives had to answer.

## From ADR-034 (state that names a checkout nobody has)

- ~~**The Go tests that do not pin `XDG_STATE_HOME`.**~~ **CLOSED 2026-09-27.** 11 of the 32 test files pin it into their own
  `t.TempDir()`; the rest do not, and exactly ONE dead entry in the 22,591 measured on 2026-09-07
  came from a Go `t.TempDir()` root. So this is real but tiny — 1 entry against 22,590 from
  `scripts/contract.sh`, which ADR-034 T4 fixes. Bundling 21 files into that record would bury the
  one line that mattered. The fix is `t.Setenv("XDG_STATE_HOME", t.TempDir())` in each test that
  builds a root, and the check is the same before/after count T4's fence uses, run over
  `go test ./...` instead of `contract.sh`.
  By 2026-09-27 the suite had 169 test files, and the writers left unpinned were in `cmd/mrw`
  (every in-process run passes the root command's `Before` hook, whose `seen.IsStale` creates the
  root's state directory even for a read) and `internal/mcp` (`Serve` on a temp root). Each package
  now pins it once in `TestMain`, as `internal/seen` and `internal/iter` already did; `internal/read`
  never creates state.

- **A prune that refuses when the marker's volume is absent.** PRE-REGISTERED, so the criterion
  predates the first report rather than being written to fit it. ADR-034's exact test — the `root`
  marker names a path that is not a directory — cannot distinguish a deleted checkout from one on
  an unmounted volume. Today the operator is the gate: the command is explicit, `--dry-run` shows
  the list, and a lost ledger costs a re-read and cannot cause a wrong edit (ADR-002) — though the
  iteration working set and the authoring tally in the same directory are lost outright.

  **What would promote this:** one report of a real `--prune` that removed state for a checkout on
  a volume that was merely unmounted. The answer is then a refusal for entries whose marker names a
  path under an absent mount point, NOT an age gate — an age gate deletes the same entries a
  fortnight later and adds a number nobody can defend. ADR-034's Alternatives rejects the age gate
  in advance for that reason.

- **`absReal` resolving through ancestors that are gone.** ADR-034 T5 needed the spelling a state
  key was minted under while its checkout existed, and rebuilt it by resolving the deepest surviving
  ancestor. Doing that inside `absReal` instead would make `Dir` and every later lookup agree by
  construction rather than by consulting three candidates, which is the better shape.

  **Why it is not done:** it CHANGES THE KEY for any root under a symlinked parent that does not
  exist yet. `Dir` on such a path currently keys by the literal spelling; afterwards it would key by
  the resolved one, and every state directory already on disk for such a root becomes unreachable —
  a ledger silently starting empty, which reads exactly like a first run. That is a migration with a
  compatibility window, not a cleanup.

  **What would promote this:** a third site needing the same reconstruction, or a decision to write
  the migration. Measured cost today: three `key()` calls per prune, on a command an operator runs
  by hand.

- **The permission-error branch on Windows.** `TestARootThatCannotBeStattedIsKept` proves that only
  `fs.ErrNotExist` means "gone" by denying traversal of a parent with mode 0. That does not deny
  traversal on Windows, and the test SKIPS there — so on Windows nothing shows the branch holds, and
  a skip is not a pass. The Windows spelling is an ACL denying `FILE_TRAVERSE`, which needs either
  `golang.org/x/sys` or a `syscall` block, and `go.mod` declaring exactly one requirement is a
  standing invariant of this repository.

  **What would promote this:** a report of a prune removing live state on Windows, or a decision
  that the second requirement is worth it. The `windows` CI job runs the test today and reports the
  skip.

## From ADR-035 (a multi-line replace declares what it replaces)

- **Requiring a guard on a multi-line `delete` too.** ADR-008 pre-registered the
  condition — *"if a wrong range still reaches a build after ADR-008, the tax is
  worth paying"* — and ADR-035 pays it for `replace`, because both measured
  incidents are replaces. It does not extend to `delete`, because no wrong
  multi-line delete has been measured and extending on shape alone is the
  symmetry argument ADR-027's own deferral refuses. `delete` also already has the
  stronger opt-in guard: an expected body declares every line, not just the
  first. Reopen when a wrong multi-line delete range reaches a build.

- **Deriving the anchor from the ledger instead of asking the caller for it.**
  mrw already knows which lines it served (`internal/seen`), so in principle it
  could check the addressed line against what the caller was actually shown and
  need no `anchor=` at all. Not done, and the reason is not effort: the ledger
  records that a line was served, and re-deriving the anchor from the file at
  write time compares the file against itself, which is the tautology ADR-008
  names for its own guard. Making it real means the ledger storing the served
  CONTENT, not just the span — a size and staleness question this record did not
  need to answer, since a caller who read the lines can copy the anchor out of
  the `NNN| content` the read already printed.

- **The handshake `instructions` do not teach the anchor requirement.** Same wall
  and same reasoning as the ADR-027 entry above, which measured it: the document
  is at 4,095 of `maxInstructionsChars`' 4,096 BYTES, and every candidate cut
  turned a guarded claim red. The `mrw_write` tool DESCRIPTION carries it as of
  2026-09-08 and contract §43 asserts it through the built server, so an MCP
  caller does get it — from the tool, not the handshake. The refusal message also
  names the remedy, which is what bounds the cost of not saying it earlier.
  **What would promote this:** whatever promotes the ADR-027 entry, since one cut
  funds both.

  ⚠ **The handshake's ADR-035 edit REMOVED a falsehood; it did not add the rule.**
  `A pattern must match EXACTLY ONE line` became `The START must match EXACTLY ONE
  line` — nine characters for nine, so it cost none of the one spare byte — and the
  END-delimiter rule is carried by the `mrw_write` tool DESCRIPTION and `AGENTS.md`,
  not by the handshake. An eighth-round review caught this branch's own commit
  message claiming all three places state the split. They do not.

- **~~`/from/,/to/` MEANS DIFFERENT THINGS TO `read` AND TO `write`.~~ CLOSED by
  ADR-036, 2026-09-08, the same day it was filed.** Found by the eighth Codex
  review round of ADR-035, while checking whether a documentation correction
  about the end pattern held on the read path too. It did not.

  Two of the three differences are gone: the end is now the first match at or
  after the start on both paths, and a paired pattern whose end never matches is
  reported rather than served to EOF. The third — a read serving a span for every
  match of the start, where a write refuses unless it matches once — is KEPT on
  purpose and is recorded as a boundary in ADR-036, not as debt: the exactly-once
  rule answers which site a plan means, and an exploratory read need not answer
  it.

  Kept here rather than deleted because the entry is a receipt: the finding, the
  direction chosen, and the reason the remaining difference is not one.

## From the v1.8.0 release (contract §24, observed rather than decided)

- ~~**§24's concurrency row can FAIL spuriously, not only skip.**~~ **CLOSED 2026-09-09 by ADR-038** (#154), and not by the fix proposed below: the ledger lock keeps every entry, so `kept` has nothing to lose between the measure and the writes. §24 now fails outright when fewer than 40 survive, §76 asserts 40 of 40, and `TestConcurrentRecordsKeepEveryPath` pins the lock. `scripts/contract.sh:920`
  races 40 concurrent `mrw read` invocations, measures how many ledger entries
  survived as `kept`, then writes to all 40 and asserts `applied == kept`.
  Observed three times on 2026-09-08 on one host: `exit 22, want 23`,
  `exit 33, want 34`, and a clean pass. The window is between MEASURING `kept`
  and performing the writes — a ledger entry can be lost in it, so `applied`
  comes back one lower than the count the assertion was built from.

  This is a defect in the ROW, not in mrw: the safety property it exists to
  assert — that writability follows the ledger exactly, rather than failing open
  at 40 or closed at 0 — is the right property, and it holds. What is racy is
  the way the expected value is captured.

  ⚠ It also means the count is not the only thing this row perturbs. The README
  status paragraph explains a ±3 swing from its SKIP branch; a spurious FAIL is
  a different and louder outcome, and "no assertion fails on the tagged tree"
  is a statement about the runs that were measured rather than a guarantee.
  The v1.8.0 CI run (34254748001) had 0 FAIL.

  **What would fix it:** re-measure `kept` immediately before the writes, or
  drive the comparison from the same enumeration the writes use, so the expected
  value and the writes see one ledger state. Not attempted here — it is a change
  to a contract row that guards ADR-002, and doing it inside a documentation PR
  is how a gate quietly stops asserting what it was written for.

## From ADR-037 (the binary teaches the format it demands)

Classified 2026-09-09 from the review; this heading is the receipt so `adr-debt`
does not report UNRECEIPTED. Nothing here is newly invented — each line names
the older entry or record that still owns it.

- **MCP tools for `check`, `iter`, `seen`, `stats`.** Still ADR-010's deferred
  follow-up. ADR-037 teaches that they exist on the CLI; it does not put them
  on the wire.
- ~~**The CLI parallel-read ledger race.**~~ **CLOSED 2026-09-09 by ADR-038.**
  Exclusive lock in `seen.Record`. §76 green.
- **Streaming / memory-bounded apply, non-Go check scoping, unguarded
  multi-line delete, Windows `%LOCALAPPDATA%` state path, a live-model
  plan-authoring benchmark.** Already in this file under their parent records
  (ADR-001, ADR-003, ADR-008, ADR-004, ADR-009). ADR-037 does not reopen them.
- ~~**Generating AGENTS.md from `guide.Shared()`.**~~ **DECLINED 2026-09-27** with ADR-037's and
  ADR-040's follow-ups: AGENTS.md paraphrases for a reader in the checkout and names `mrw instructions`,
  and a two-way sync is the process tax ADR-037 refused. The old trigger ("if `Contains(Shared())`
  starts failing") had no gate to fail. Reopen when a caller follows AGENTS.md and the binary refuses it.

## From ADR-038 (a ledger write is one writer)

- **A shared lock on `Load`, or an atomic save via rename.** ADR-038 locks
  `Record` and leaves `Load` unlocked and `save` as `os.WriteFile`. A reader
  that opens the ledger in the middle of that write could see a torn file.
  Unmeasured. Promote when one is observed; prefer rename-over (with a Windows
  answer) rather than a shared lock first. The exclusive lock already serializes
  writers, so the only remaining reader is `Load` / `mrw seen` / `apply` at the
  start of a write.

## From ADR-040 (the help a PATH caller trusts names how to quote a header option)

Receipt for M, 2026-09-12: *"I NAME THEM ALL"* ranked the list; *"good, accepted
all"* recorded each leftover as ADR-041–050. These bullets stay as the 040
pointer. None of them is an engine change.

- **Putting the four quoting / `body=` / `lines=` sentences in `guide.Shared()`
  and the MCP handshake.** ADR-040 teaches them on `write --help` and
  `guide.CLI()` so a PATH caller sees them without paying 4096 on every MCP
  session. The trap is plan-text, so MCP authors can still hit it. Revisit if
  a handshake shortening makes room, or if Accept quotes this and the bound
  still holds. Do not raise `maxInstructionsChars`.
- **PATH binary vs skill version skew.** The field report that forced the
  consumer hard rule: PATH `mrw -v` → `dev (0b75313)` (mtime 8 Sep) while this
  tree and the skill were at v1.12.0. ADR-040 OOS previously said only "old
  binaries not upgraded". Named here as its own product: a PATH install and a
  central skill can disagree, and `--help` on the old binary is what the caller
  trusts. Not 040's Decision. Arm with *"spec version skew"*.
- ~~**`mrw check` silent in-root fallback.**~~ **CLOSED 2026-09-12** — ADR-042
  T2. An in-root miss is refused at exit 2; prose and testdata still fall
  back. M: *"accepted, close"*.
- **A torn `Load` during `WriteFile`.** Not this record. Still the ADR-038
  entry above. Do not promote unless M says *"measure torn Load"* — a measure
  task, not a lock.
- **MCP cargo: `check` / `iter` / `seen` / `stats`.** Still ADR-010 / ADR-019
  Decision 5. Not 040. Arm with *"spec MCP cargo"*.
- **Generate AGENTS.md from `guide.Shared()`.** Still the ADR-037 entry above.
  Arm with *"spec generate AGENTS.md"*.
- **Host-cut under the advertised ceiling.** Still the ADR-039 measurement
  entry above. Arm with *"measure host-cut under ceiling"*.
- **Python `str` body character-split.** Caller, not the engine. Decision 1
  already teaches that `body=` is a line count. Arm with *"document Python str
  in help"* or leave the caller.
- **Syntax awareness; streaming apply; Windows `%LOCALAPPDATA%`.** Engine
  dreams. Streaming and the Windows path already have parent entries
  (ADR-001, ADR-004). Syntax awareness has none and must not ride 040.
  Parked. Not 040.
- **Playtrix T4, other wings' inboxes, 019 B/C, a Canvas file, installing
  mrw, `keep/` gitignore, "use CLI not MCP" as this binary's contract.**
  Not this repository. Named so they stop arriving as implied 040 work.

## From ADR-051 (foreign plan grammars compile to plan hunks)

- **Aider SEARCH/REPLACE as a second `--format`.** Shipped 2026-09-12 as
  `--format=search_replace` / MCP `format=search_replace` (F-26). Exact unique
  SEARCH only; unread writes nothing.
- **MCP `format` on `mrw_write`.** Shipped 2026-09-12 (F-27). Optional
  `format` on the existing write tool (`plan` default, `apply_patch`).
- **`*** Delete File:` / `*** Move to:`.** Shipped 2026-09-13 as ADR-057
  (`@@ path - unlink` / `@@ old - rename`; `*** Delete File:` / `*** Move to:`
  compile to those hunks). `delete` stays a line-range (ADR-008). Move to
  with extra `@@` hunks is still refused — see From ADR-057.
- **ast-grep-shaped `--grep`.** Shipped 2026-09-15 as ADR-058. Structural
  find only, and only if it does not become a write-time parser (ADR-048).
  READ path only. Flag `--ast-grep` (MCP `ast_grep`) beside `--grep` shells
  out to the `ast-grep` CLI if present, maps hits to line ranges, and serves
  through existing `read`. Missing binary: exit 2, names `ast-grep`. Apply
  does not change. License is still served lines, not AST nodes. `--grep`
  stays regex; the two may disagree on the same token — that is the feature.
  Rejected: tree-sitter inside apply; replacing regex `--grep`; bundling a
  per-language parser.

## From ADR-052 (echo pad is opt-in; a multi-line replace needs a served line after End)

Filed 2026-09-13 from a Zeus field report that read `internal/apply/apply.go:960`
before proposing, and that dropped two ideas after that read. ADR-052 already
refuses the wrap-tail miss (`end > start` and ledger does not cover `End+1`).
`--check`, `--echo-pad`, `sha=` and the ledger already exist. Nothing below is
new capability.

The licence would not have caught the three Zeus breakages: they were
**single-line addresses with multi-line bodies**, so `end > start` is false.
Extending the trigger to "multi-line body" would not have caught them either —
the rule asks whether the ledger covers `End+1`, and those lines had been
served. No ledger rule catches this. The ledger knows what was seen, not what
the body did to the structure.

Evidence from that checkout's `mrw stats`: 259 plans, 251 applied (96.9%).
Three of those 251 left the tree uncompilable and mrw reported success on every
one. The tool measures its contract — did the hunk apply — and the outcome
diverged silently.

ADR-052 rejected a **default** echo on cost grounds. That reasoning does not
transfer to (1) on a *code* write: echo prints on every write; a cover-gated
`--check` runs the project's own command when the plan touched a non-prose
file. It does **not** transfer as "scoped `{packages}` on every write":
`packages()` is Go-only (`internal/check/check.go:385`). Zeus cannot write
that mitigation. ADR-054 takes option 2 after the 2026-09-13 review: default
only when a written path is not prose; a Zeus `.rs` write still pays the
whole-project cargo command.

Accepted 2026-09-13 by M (*"accepted"*) and executed the same day as
contract §89–§91: `--no-check`, `apply.IsProse`, `HunkResult.Balance`,
`authoring.Vocabulary` / `Tally.Landed`. This section is the receipt; the
record is `docs/adr/ADR-054-a-write-that-applied-can-still-leave-a-broken-tree.md`.
Twenty-two contract rows that write `.go` into the `go.mod` fixture now say
`--no-check`: they assert apply semantics, and the inferred `go test` would
otherwise fail them at 3.

Reconfirmed 2026-09-15 against v1.21.0 in a Zeus scratch repo: a `.rs` edit
with **both** `scoped_check` and `check` declared still ran the whole-project
command (`echo FULL`), because `command()` only scopes when `packages()` maps
every path and `packages()` is Go-only (`internal/check/check.go`). Declaring
`scoped_check` is not enough. The lever that is not a Rust `packages()` is
honoring `{files}` when `packages()` cannot map — that changes the ADR-054
Decision ("when packages() cannot map, whole-project Check runs") and needs a
new record. `--no-check` on intermediate writes remains the escape. FenceTimeout
1800 now binds (ADR-059).

Closed 2026-09-16 as ADR-061 (M: *"so work on 054"* after quality-harness
v2.99.5). `command()` returns the `{files}`-only template when the map is
empty and the path list is non-empty. `{packages}`-only and mixed still fall
back. No Rust `packages()`. Contract §113. `--no-check` remains the escape
when the project declared only `check`.

Closed 2026-09-16 as ADR-062 (M: *"accept"*, then *"use it always and plan
activity"*). The 3+ When trained one-turn agents never to call mrw.
`guide.Shared()` first sentence is always + plan; `guide.CLI()` is the
effective-use document (`@@ path 0 create`); MCP handshake stays Shared,
4096. Centralised skill POST is deferred (inventory row).

- **`--check` runs by default; `--no-check` opts out.** The only existing
  mechanism that would have caught all three, in the turn that caused them.
  Arm with *"check by default"*.

- **A delimiter-balance delta in the receipt — visibility, not a refuse.**
  Echo's sibling; stays inside ADR-048 (no AST). Arithmetic on the hunk: count
  `{}`, `()`, `[]` in the replaced range and in the body; if they differ, say
  so in the receipt. Two of the three Zeus breakages produce a delta (stub
  opening `{` replaced by a balanced body; stale-address replace in
  `message.rs` with an unmatched `}`). The third does not: a balanced
  `insert-before` of complete `#[test]` fns, net 0 vs 0. Arm 1 catches all
  three; this arm does not. Naive counting will miscount braces inside string
  literals, which is why this reports rather than refuses. Arm with
  *"balance delta"*.

- **`mrw stats` gains one row: of the applied plans, how many were followed by
  a failing check.** Today's tally is `applied` / `refused_apply` /
  `refused_parse` — facts about mrw's contract. The number that would have
  shown up after the first occurrence instead of the third is "your writes
  succeed and your files break". Arm with *"stats check-fail row"*.

- **Do not extend the neighbour licence to single-line addresses as a fix for
  this.** The wrap-tail risk is identical whether the address or the body is
  the multi-line half, but the three cases it was meant to fix catch none of
  them. Open question in its own right; not armed.

- **A body at the wrong indent reparents keys, and nothing is left to find.** Deferred from ADR-052,
  whose neighbour licence only requires a served line after a multi-line replacement range, so the
  caller can see a surviving closer; nothing like that exists for this: in indentation-structured files a
  body at the wrong depth moves keys under another parent, and the result is valid and means something
  else (Ansible `when:` became a module argument, three linters green). mrw models no target syntax, so
  the remedy is teaching (AGENTS.md §4) rather than a guard. Promote on a reported case a read-after
  could not have caught.

## From ADR-054 (a write that applied can still leave a broken tree)

Filed 2026-09-13 from M's field run of v1.16.0/v1.16.1 in Zeus (commit
`ed9dfdb9` there): nine mutants, one survivor, and the survivor was a real hole
in a test already recorded as mutation-proven that morning. Four breakage
shapes; ADR-054 arm 1 caught all four in the turn, arm 2 three of four. What
follows is what the run showed the receipt still does not do. Ordered by M's
value-per-work: 1, then 3, then 2. Designed 2026-09-13 as
`docs/adr/ADR-055-the-receipt-counts-its-advisories-and-notices-a-pattern.md`
(M: *"so plan then backlog"*, then *"accepted."*, 2026-09-13). Executed the same day as
contract §92–§94. Each line keeps its arming quote as its receipt.

- **Put the advisory count in the summary line, and `advisories: N` in the JSON
  receipt.** The `balance { +1 → +0` row fired correctly and was not acted on,
  twice in one hour, because the line actually read every time is
  `1 hunk(s), 1 file(s), 0 failed — applied`, which omits it; and the JSON
  consumer printed only `applied` and `failed` because those are the keys the
  summary taught it to care about. Make it
  `1 hunk(s), 1 file(s), 0 failed, 1 advisory — applied`. "Advisory" is the
  class (today: the balance row), not the row; `echo` is opt-in visibility the
  caller asked for and is not one. One line in `report`, one receipt field, one
  contract row. Arm with *"advisory count"*.

- **A repeat-pattern line.** The tally is cumulative and per-outcome (ADR-009:
  counts, no paths, no time axis — on purpose). What it cannot say is "this is
  the fourth replace in one session that carried a balance advisory", which is
  the signal that changes method instead of repeating it. Design: a small
  recent-window ring beside the ledger holding op, advisory bit and time —
  nothing ADR-009 refuses — and one line when a pattern repeats. It must print
  on the WRITE receipt, not only in `stats`: `stats` is run after the fact; the
  receipt is read in the turn. Zeus's reflex-counter idea, applied to the tool
  that watches the caller. Arm with *"repeat pattern"*.

- **`--strict-balance`, opt-in, refusing the wrap-tail signature only.** ADR-054
  rejected refusing on *any* delta, correctly (string literals, generated
  code). The case that has bitten four times is narrower: a `replace` whose
  addressed range has a non-zero net and whose body has a different one. That
  is the wrap-tail shape, and it is what ADR-052's neighbour licence already
  refuses for MULTI-line addresses; the single-line address is the hole. Three
  of the four Zeus breakages carry the signature; the fourth (balanced insert)
  has no delta and stays arm 1's. A refusal is a failed hunk — exit 1, nothing
  written, ADR-001 — not a new exit code. Opt-in first, the way `--echo-pad`
  went in; a campaign across real corpora (Zeus, this repository, playtrix)
  prices the false positives before anyone argues a default. Arm with
  *"strict balance"*.

- **Not proposed again:** a harness `covers` glob for the `.jsonl` / `.yml`
  cost. ADR-054 rejected it; the cost is real and bounded by `--no-check`.

### From ADR-055 (v1.17.0)

- **`pattern` on the MCP receipt** — ADR-055 T2 deferred it. **Closed by
  ADR-056 T1** (both JSON receipts, every write; contract §95). M, 2026-09-13:
  *"address these tow"*.
- **The pricing campaign had no data source** — **closed by ADR-056 T2**
  (`mrw stats` pricing block; contract §96). The criterion above is unchanged;
  only its source sentence was corrected.

### Pre-registration: a default `--strict-balance`

Written 2026-09-13, BEFORE any campaign, so the criterion is not shaped by the
result (this file's rule). ADR-055 T3 ships the flag opt-in. It may be argued
as a default only if a campaign over three real corpora — Zeus, this
repository, playtrix — reads `mrw stats --json` `.pricing` in each checkout
(ADR-056: mrw prices the flag as writes land, because ADR-009 holds no plan
text to replay after the fact) and reports, per corpus: how many landed writes
the flag would have refused (`strict_would_refuse`); of those, how many went on
to break the tree (`_broke`: the check ran and failed) versus how many did not
(`_held`: the false positives). `_unchecked` counts neither way and is reported
so an MCP-heavy corpus cannot flatter the flag. Criterion:
**false positives under 5% of refusals in every corpus, with at least 50
refusals total**, or the default stays off and the flag stays what it is. A
campaign that reports the true-positive count without the false-positive count
does not qualify — the number that argues a default is the one that costs the
caller, not the one that flatters the tool. Arm the campaign with *"price
strict balance"*.
A named corpus with no checked refusals (`broke+held == 0`) has no defined rate and does not pass.

Ran 2026-09-13 against PATH `mrw` v1.18.0 (`a5c4f24`). `mrw stats --json`
`.pricing` in each checkout (landed writes are the older tally; pricing only
counts writes that landed on a binary that prices):

| corpus | landed | candidates | would_refuse | broke | held | unchecked |
|---|---|---|---|---|---|---|
| this repository | 307 | 0 | 0 | 0 | 0 | 0 |
| Zeus | 296 | 0 | 0 | 0 | 0 | 0 |
| Playtrix | 127 | 0 | 0 | 0 | 0 | 0 |

**Does not qualify** — 0 refusals total; the bar is ≥50. Default stays off.
v1.18.0 shipped the same day; earlier landed writes were not priced. Same
quote re-arms a later read. Do not synthesize wrap-tails to fill the
counters: the criterion is real corpora.
Ran 2026-09-16 against PATH `mrw` v1.22.0 (`6896442`). Same three checkouts as 2026-09-13. `mrw stats --json` `.pricing` (landed is the tally, not `.pricing`):

| corpus | landed | candidates | would_refuse | broke | held | unchecked |
|---|---|---|---|---|---|---|
| this repository | 320 | 0 | 0 | 0 | 0 | 0 |
| Zeus | 360 | 26 | 1 | 0 | 0 | 1 |
| Playtrix | 135 | 4 | 0 | 0 | 0 | 0 |

**Does not qualify** — 1 refusal total (Zeus, `_unchecked`); the bar is ≥50. Zeus `broke+held == 0` so the FP rate is undefined and that corpus cannot pass "under 5% in every corpus with checked refusals". Default stays off. Same quote re-arms a later read. Do not synthesize wrap-tails.

Stress of that same binary, 2026-09-13: `FuzzBalanceDelta` ~2.06e6 execs
clean; `TestRandomisedApplyBalanceFollowsTheDecision` seeds 54/1/7/13/99
(400 ops each, all five parser ops) clean; break campaign 47 = 47, no exit
diff vs `docs/break/campaign-v1.18.0.txt`; random CLI matrix 440 writes
across seeds 118/7/13/99, exits only in {0,1,2,3}, zero crashes, ADR-001
held on every exit 1, `--json` receipts carried `pattern`, constructed
wrap-tails with `--strict-balance` refused at exit 1. `*** Delete File:`
and a hunk-less `*** Move to:` compile and apply (ADR-057); Move to with
extra `@@` hunks still compile-refuse at exit 2 and leave the tree.

## From ADR-057 (native unlink / rename)

- **Move to with hunks** — **Closed** by ADR-114 (the engine edits and renames a file in one plan, and apply_patch
  compiles the section, 2026-10-02). `*** Move to:` after `*** Update File:` with extra `@@` hunks was refused.

- **Teaching nits from the 2026-09-15 Zeus v1.21.0 field report.** `write --help`
  named address `-` for rename but not that the dest is the body (`to=` was
  guessed → exit 2). Unread unlink said "a line address means nothing" though
  unlink has no line address. Fixed in the same commit as contract §112; this
  line is the receipt.

- **ADR-066 found this record's commit path losing data.** A plan that unlinks
  `c`, renames `b → c`, then fails a later rename had `restore()` rename the
  unlinked `c` back over the file just renamed onto it (reproduced on v1.22.3,
  2026-09-24). The stress table's surviving "restore() no-op" mutant marked the
  path. Fixed by ADR-066; this line is the receipt.

## From ADR-065 (read and write split a file into the same lines)

- **ast-grep on a CR-only file.** ast-grep numbers rows by `\n`, so a hit in a
  CR-only file is reported as a problem rather than served on the wrong line.
  Translating ast-grep's byte offsets into mrw's lines would serve it; arm
  if a caller hits the problem line.

## From ADR-066 (a plan that cannot commit whole says what it wrote)

- **The ledger after a partial commit** — **Closed** by ADR-102 (`writer.Apply` records a partial commit's written files, 2026-09-30). Only an applied plan records what it
  wrote, so a file a partial commit DID write is refused as "changed since mrw
  last saw it" on the next edit. Loud, not silent; arm if a caller hits it.
- **Staging the unlink placeholder, or probing directory writability.** The
  aside placeholder (`pathop.go`, `CreateTemp`/`Remove`) and an unwritable
  destination or source directory still fail only at commit. ADR-066's unit
  undo makes that failure non-destructive and its receipt truthful; moving
  them earlier is the next step if one is ever seen.
  **Seen, 2026-09-25** (chaos seed 25 on v1.25.0, `9a586aa`): a rename to a destination whose name
  is not valid UTF-8 (`moved/\xffdash.txt`) passes validation and fails at commit on APFS with
  `illegal byte sequence`. The content edits of the same plan stay written. The receipt says
  PARTIALLY APPLIED, names the written files, and exits 2; that is the ADR-066 contract, and v1.24.1
  behaves identically, so this is not a regression. The trigger for moving the failure earlier is now
  met: validate the destination name at staging, so such a plan writes nothing. It needs a record.
  **CLOSED 2026-09-27 — by ADR-086**: each create target and rename destination is created and
  removed at staging, so such a plan fails before anything is written (exit 2, nothing written). The
  reproduction also found a create of `bad\xffname.txt` landing as `bad�name.txt` at exit 0:
  the plan header was walked as runes; it is walked as bytes now. Contract §168, and §119's
  read-only case is now a staging refusal.
- **foldKey collapses distinct invalid bytes** (the Codex review of #254, source-traced, not
  observed): `foldKey` (`internal/apply/apply.go`) decodes a name as runes, so `x\xfe` and `x\xff`
  both fold to U+FFFD and two creates of them are refused as one file. A false refusal, never a lost
  write. Arm on one observed plan that needs two such names.
- **A probe left behind** — **Closed** by ADR-105 (`left_behind` names it, 2026-09-30) (ADR-086, not observed): when a staging probe's removal fails, the empty
  file stays in the tree while the receipt says NOTHING WRITTEN; the refusal names it, but the
  receipt's written list does not. Say so on the receipt if it is ever observed.
- **A --json receipt with a filesystem-derived invalid name** (the Codex review of #254,
  source-traced, not observed): ADR-086 refuses a `--json` plan whose GIVEN names (path, rename
  destination, expanded pointer) are not valid UTF-8, but a name the filesystem supplies — the target
  of an in-root symlink, a directory created through a linked one, a Linux root holding such bytes —
  still reaches `Files[].Target`, `dirs_created` or `root` and serialises with U+FFFD. Needs a
  receipt-contract decision (refuse at staging from the resolved names, or encode names losslessly),
  not a patch. Arm on one observed case.
- **A Windows absolute spec normalised before the alias check** (the Codex review of #254,
  source-traced, Windows only): `read.go` and `walk.go` call `rooted.Real` (`EvalSymlinks`) before
  `rooted.Resolve`, and on Windows that returns the on-disk spelling, so an absolute spec holding an
  invalid byte whose U+FFFD twin exists is served as the twin. A write to the invalid name is still
  refused by `Resolve`. The fix is to check the original spelling first, in `internal/read`. Arm on a
  Windows report.
- **A `.mrw-aside-*` left behind.** — **Closed** by ADR-105 (`left_behind` names it, 2026-09-30). When the final aside removal fails after a
  plan that applied, the placeholder stays in the tree (ADR-004 hygiene, not a
  false receipt). Say so on the receipt if it is ever observed. The same holds
  for a staged `.mrw-*` temp file whose removal fails after a staging abort
  (`discard` in `apply.go`). Since ADR-088 both discards are written `_ =` at
  their sites; the Codex review of #259 traced them, and reporting the survivor
  is this entry's work.
- **The handshake said "if any hunk fails, nothing is written".** After ADR-066
  a commit failure can leave some files written with a hunk `failed`. Fixed in
  ADR-066 T3 after the Codex review of #207: the Shared sentence now reads "if
  any hunk fails validation, nothing is written; a failed commit says PARTIALLY
  APPLIED". This line is the receipt.

## From ADR-059 (honour `fenceTimeout`)

Zeus declared `fenceTimeout: 1800` and still hit the five-minute default
because mrw unmarshalled only `timeout_seconds`. ADR-059 takes the alias.
The other half of that hang — a `.jsonl` write paying the whole cargo
line — is not an alias.

- **Per-extension check skip.** A data file is not prose (closed list:
  `.md .markdown .txt .rst .adoc`). Growing the list to include `.jsonl`
  takes `.toml` (Cargo.toml) unless the rule is per-extension rather than
  "not code". Arm with *"per-extension check"*. Rejected as part of 059:
  treating `.jsonl` as prose.

## From the 2026-09-16 dangling spec

Receipt for `docs/specs/2026-09-16-dangling-high-impact.md`. Pins, not engine work. Unrun is not coverage.
Execution plan: `docs/specs/2026-09-16-dangling-high-impact-plan.md` (campaign first; no engine ADR unless a probe finds a defect).

- Desktop reach measure (UC-1) — trees-per-session on the Desktop population; coder plan count is the wrong population. Pick A stands. Not run 2026-09-16 (no Desktop session).
- Under-ceiling host-cut (UC-2) — beside reading 18; do not treat ADR-039 as that evidence. Observed 2026-09-16 wire-only; model not run; still Deferred.
- Concurrent silent apply (UC-3) — last-writer-wins; locking stays out of scope (ADR-002). — superseded by ADR-075: one writer per checkout.
- Strict-balance default campaign (UC-4) — 5% / 50 / three corpora; default stays off.
- JSX nest probe (UC-5) — probed 2026-09-16, `tsc` 0, DOM parent `#accidental-wrapper`; still not a finding.

## From ADR-068 (the read ledger keeps a path exactly)

- **`internal/iter` trims a working-set line.** **Closed by ADR-069 T2** (v1.25.0): Load, Add and
  Remove keep the entry as written.
- **urfave/cli trims a positional argument before `--`.** **Closed by ADR-069 T1 and T5**
  (v1.25.0, and the follow-up to the Codex review of v1.25.0): a padded positional is refused, exit
  2, naming `--`; a `--` consumed as a flag value no longer ends that guard; an attached flag value
  ending in whitespace (`--files-from='list '`, `--root='dir '`) is refused, naming the separate
  spelling, which the parser keeps as given.
- **Other places a caller-supplied path is trimmed.** **Closed by ADR-069 T2–T4** (v1.25.0):
  `--files-from`, a rename destination, and apply_patch / search_replace paths keep the path as
  written.

## From the v1.25.1 field tests (2026-09-25, twelve peer sessions, macOS and Windows)

- **`mrw instructions` did not teach the ADR-069 rule.** **Closed by ADR-069 T12.** Found by the
  session that tested the downloaded Windows asset: no line mentioned `--` or the attached-value
  refusal, though ADR-063 promised the read side's traps there.
- ~~**On Windows a spec with a trailing space reaches the OS as given and the OS folds it.**~~
  **CLOSED 2026-09-27 — by ADR-071 Decision 4** (v1.26.0): `rooted.Resolve` refuses a component
  Win32 would read as another name, so `x ` is refused before the ledger records anything and a
  folded pair can no longer be planned. `TestWin32AliasNamesTheComponentWindowsWouldRemap` runs on
  every platform, `TestResolveRefusesAWin32Alias` on the Windows CI job. Closed from the code, not
  from a Windows re-run of the probe. Over
  MCP, `mrw_read` with `specs: ["x "]` kept the spelling (the ADR-068 promise: the header printed
  `x ` with its space) and Win32 path normalisation then opened `x`, so the ledger holds `x ` and `x`
  as two keys over one file, each with its own ack id. Whether a plan naming both spellings is
  refused as one file (ADR-021 uses the filesystem's identity, and on NTFS the two ARE one) is
  untested there; `mrw_write` on the folded pair was deliberately not tried by the finder. Arm with a
  Windows fixture in `internal/apply` before trusting ADR-021 on that platform. Evidence: the
  desktop-3laqmbq MCP probe, v1.25.1 built from a partial checkout (see the next item).
- **Nine task paths exceed the Windows path limit without `core.longpaths`.** A plain `git clone`
  on Windows 11 aborts with "Filename too long"; two of the paths are ADR-027 T1 and ADR-041 T1.
  `git clone -c core.longpaths=true` works. A rename shortens them; a CONTRIBUTING note is the
  cheaper fix. Found by two independent Windows sessions.
  **Fixed** (2026-09-26, housekeeping): the 35 task files over 120 characters were renamed, and
  `TestNoTrackedPathIsLongerThan120` keeps every tracked path at 120 or under, so a plain clone
  works under a root of up to 138 characters (CONTRIBUTING, Prerequisites).
- **`scripts/contract.sh` is unmeasured on a Windows bash.** Git Bash there has Go but no `jq`, so
  the script exits 2 at its `jq` check before any assertion; WSL there has `jq`'s absence and no Go.
  Documented as Linux-only; this row records that the documentation was tested, not the script.
  **Permanent** (fact): the script is POSIX shell that parses receipts with `jq`; the Go suite is
  what runs on Windows, in CI and on the peer machines.
- **The random differential test (T11) agreed on every peer seed that ran**: 101 on macOS
  (20,000 cases, 13 s; the 5–8 minute estimate in the brief was wrong, measured under load 33).
  Three macOS sessions held their run for their user (load above the core count, or a permission
  classifier refusing another project's code) — correctly, per costly-runs.

## From the v1.25.1 adversarial round (2026-09-25, peer sessions told to break it on purpose)

M, 2026-09-25: *"ask session to go crazy, random, try to break, brute force cases, unexpected,
intentionally broken"*. Surfaces were split across sessions; every cross-platform claim below was
reproduced by the coordinating session on macOS with the shipped v1.25.1 (`5cf522c`) before it was
listed. The finders' full reports, repro commands and the long "what held" lists are in the team
memory (wing `wing_tool-multipathreadwrite`, room `llm_open_threads`, "WHAT DID THE v1.25.1
ADVERSARIAL ROUND FIND", and its read-side addendum; the junction repro script is in room
`incidents`). Nothing below is fixed. Pending when this was written: the write/check surface and,
held for M's approval in their sessions, concurrency and the state directory; a late report is an
append to this section.

Contract breaks, reproduced on macOS:

- **After any write the ledger licenses the whole file.** Read line 3, write line 3, then write
  line 1: applied, exit 0; `mrw seen` says "the whole file". Over MCP too. Contradicts ADR-002's
  per-line promise, or refines it; either way a record. Finders: the general and MCP sessions.
  **Disposition (ADR-071 T5):** as designed. ADR-002 and ADR-005 §4 record that a write observes
  the whole file; M kept it on 2026-09-25, and the prose that said "per line" without the
  exception now states it.
- **A malformed `.quality-harness.json` applies the write and exits 2 with only the JSON error.**
  No receipt; exit 2 is documented as usage or filesystem. The config is parsed after the commit.
  **Fixed by ADR-072 T1**, contract §142: the harness is read before apply, and a malformed one
  refuses the write, exit 2, nothing written.
- **A killed check leaves an applied write with zero bytes printed.** The receipt is rendered only
  after the check returns, so an outer kill of mrw during the check prints nothing. mrw's own bound
  (five minutes by default, `timeout_seconds`) kills only `sh`, and a grandchild survives it.
  **Fixed by ADR-072 T2 and T4**, contract §143 and §145: the human receipt is printed and the
  landing counted before the check runs; the check runs in its own process group, which its
  timeout kills, and an interrupt during the check reports `interrupted`, exit 3. On Windows only
  the bound on held pipes applies (deferred: a job object).
- **Creates are not cross-checked.** Two `create` hunks for one path both report ok and the bodies
  are concatenated; two spellings of a NEW file on a case-insensitive filesystem (`n.txt` +
  `N.TXT`, macOS and NTFS; `n.txt` + `n.txt.` on NTFS) both report "created", one file remains, and
  the first body is lost, exit 0. ADR-021's check works for files that exist; a create has nothing
  to stat. Mixed create-and-rename variants are caught only at commit (`PARTIALLY APPLIED`).
  **Fixed by ADR-071 T2**, contract §141: a second create of one path, and names that differ only
  by case where one does not exist yet, are refused on every platform before anything is written.
  Folds only the filesystem knows (ß and ss, NFC and NFD, found by the review of #228) stop the
  commit at the second create, PARTIALLY APPLIED, exit 2, instead of losing the first body.
- **Two writers off one read can both exit 0 with one edit lost.** `scripts/chaos.py` already
  counts this as a known, accepted risk (race suite, "concurrent writes"); the Windows chaos runs
  measured it at 45–53% of racing writers over five full-scale corpora, and it reproduces on
  macOS. The acceptance was recorded before the rate was known: a decision for M.
  **Fixed by ADR-075**, contract §150: a per-checkout write lock is held from apply through the
  ledger update, with the ledger loaded before it under its own lock (`seen.Snapshot`, so never half-saved), so a writer whose file changed while it waited is
  refused, exit 1, "changed since mrw last saw it"; none exits 0 without its edit. `chaos.py`'s race
  suite fails on a lost edit by default. A writer that starts after another finished still writes on
  that writer's whole-file licence (ADR-002, kept by M the same day).
- **Unlocked state beside the ledger** (the klientams peer's leads, not measured): `internal/iter`
  and `internal/authoring` rewrite their files without a lock, and `mrw seen` loads the ledger
  unlocked. Racing processes could lose a working-set entry or a tally count, never an edit. Deferred
  from ADR-075.
  **Fixed by ADR-079**, contract §160: the working set and the tally are read, changed and written
  under one lock each (`state.Hold`), their readers take it too, and `mrw seen` reads the ledger under
  its lock. Worse than recorded: a racing process could read a file emptied mid-rewrite and wipe the
  whole tally or working set, not one count.
  The reviews of #240 closed two gaps in that fix: a writer that could not take the tally's lock
  wrote unlocked, and `state.Migrate` copied legacy state with no lock; a writer now skips a lock it
  cannot take, and `Migrate` holds each destination's. Left open (deferred: found in the review of
  #240, not measured): the MCP pending-ack store is rewritten under the in-process gate only, so two
  servers on one checkout could race it, and `seen.IsStale` reads the ledger unlocked — a wrong
  stderr notice at worst. **CLOSED 2026-09-27 — the pending store by ADR-085** (contract §167: eight
  servers at once, each ack licenses its write; 0–1 of 64 concurrent holds survived before it);
  **`seen.IsStale` still deferred, as an accepted diagnostic risk**: `save` writes the header and every
  line through one `os.WriteFile`, which normally puts the short header in its first write, but a
  short first write is allowed (Codex review of #253), and a header prefix at EOF would read as stale
  and print one false notice. The ledger is unaffected. Arm on one observed false stale notice.
- **A UTF-16LE file is rewritten with exit 0**: served as byte-split lines, and a replace drops the
  BOM and mixes encodings. Nothing refuses a write to such a file.
  **Fixed by ADR-073**, contract §146: a line edit to a file that begins with a UTF-16 or UTF-32
  byte-order mark, or holds a NUL in its first 8 KiB, is refused, bytes unchanged; `read` serves it
  with a note naming the encoding. `unlink`, `rename` and `create` are unaffected.
- **`--json` prints text on a plan parse error.** The documentation promises a receipt on failure;
  a failed hunk does get JSON, an unparseable plan does not.
  **Fixed by ADR-072 T3**, contract §144: under `--json` every refusal after the plan is named is
  a JSON document with an `error` field.
- **The MCP page budget ignores JSON escaping.** A 153,600 B TSX file (many `<`, `>`, `&`, each six
  bytes on the wire) is refused whole with no `next_read`, while a 275,200 B plain file pages. The
  refusal blames the per-file receipt. Any markup file above roughly 110–150 KB is unreadable at
  the default ceiling without a hand-picked range.
  **Fixed by ADR-074 T3**, contract §149: the first page is sized by the JSON-encoded length the
  ceiling measures, and a closed range that still overflows is refused naming the encoding.
  Waived in the review of #232, open: a file whose line 1 alone encodes past the ceiling is refused
  first by `overflowMessage` with range advice, and only a second call names the CLI; that sentence
  says "no narrower range", though a range after the long line serves. Fix: the same
  `longestEncodedLine` check in `overflowMessage`, naming the line.
  **Fixed by ADR-078 T3**, contract §158: the refusal names the line — escaped past the ceiling or
  plain and longer than it — and the ranges around it, and the open range after it is served.
- **The 2 s `--ast-grep` kill fails when a grandchild holds stdout**: 30 s with a `sh` wrapper that
  runs `sleep 30`; the grandchild is orphaned when its stdio is redirected.
  `internal/read/astgrep.go` sets no `WaitDelay` and no process group (the finder's reading).
  **Fixed by ADR-074 T1**, contract §147: ast-grep runs through `internal/subproc`, so its process
  group is killed at the 2 s bound and the wait for held pipes is bounded; an interrupt, terminate
  or hangup sent to mrw stops it. On Windows only the wait bound applies (deferred with the check's
  job object, above).
  Waived in the review of #232, unverified (read from Go's exec code): a wrapper that exits 0 and
  leaves a background grandchild returns through `WaitDelay` without its context being cancelled, so
  its group is never killed and the grandchild outlives mrw.
  **Fixed by ADR-080 T1**, contract §162: `subproc.Run` and `subproc.Output` kill the child's group
  after every exit, not only on a cancel, for ast-grep and for the check (M: reap always).
- **A FIFO hangs `read`**, and `--stat`, a symlink to it, and `--files-from` on it; nothing is
  printed. A socket and a directory are reported by name.
  **Partly fixed by ADR-073**: a FIFO, socket or device named in a WRITE plan is refused before it
  is opened. The read side (a spec, `--stat`, a symlink to a FIFO) is ADR-074's.
  **The read side fixed by ADR-074 T2**, contract §148: a FIFO, socket or device named as a spec,
  or a symlink to one, is reported `UNREADABLE` by name, with or without `--stat`, and the rest of
  the call is served. `--files-from` naming a FIFO is unchanged: a list may be a pipe.

Windows only, each hand-confirmed by at least two sessions:

- **A junction inside `--root` escapes it.** Read, replace, create, rename into it and unlink
  through it all land outside the root with exit 0; `..` and POSIX symlinks are refused. A junction
  needs no privilege. Likely cause: Go reports a junction as irregular rather than as a symlink.
  `chaos.py`'s junction suite (PR #226) measures five escapes. Owed: a Windows CI test built from
  the repro (`mklink /J`, expect REFUSED and NOTHING WRITTEN).
  **Fixed by ADR-071 T1**: `rooted.Resolve` follows a junction on Windows. The owed CI test is
  `cmd/mrw/junction_windows_test.go`. The peer re-run against the v1.26.0 release asset was done on
  2026-09-26 (Windows 11, Git Bash): read, replace, create, rename into and out of, and unlink through a
  junction are each refused, NOTHING WRITTEN, over the CLI and MCP; the outside tree stayed
  byte-identical.
- **Win32 name aliasing.** A trailing dot, a trailing space, case, `::$DATA` and 8.3 names reach one
  file, so writes and unlinks land through a name that does not exist and the receipt names the
  alias; `-C 'dir '` and `-C 'dir.'` are accepted. The ledger resolves the aliases to one entry
  (that held). The guard needs the OS's canonical name: an ADR, with the junction item.
  **Fixed by ADR-071 T3**: a trailing dot or space and a `:` are refused by name on Windows, in a
  path and in the root. Case and 8.3 aliases stay accepted: they reach one file, and the ledger
  and ADR-021 match them.
- **MSYS (Git Bash) rewrites more than the docs say.** The documented example `f.go:/^func main/`
  is NOT rewritten; the trap fires when the file part contains a `/`, and a one-letter pattern
  becomes a drive (`/x/` → `X:\`). `--grep` patterns such as `/usr` or `NAME=/path` are rewritten
  into a false "no match" with no hint. `MSYS_NO_PATHCONV=1` works as well as
  `MSYS2_ARG_CONV_EXCL='*'`, and the hint names only the second. A leading-slash `--exclude` can
  never match. An attached `-C/path` arrives as `-CC:/…`.
  **Partly fixed by ADR-074 T4**: the hint names `MSYS_NO_PATHCONV=1` beside
  `MSYS2_ARG_CONV_EXCL='*'`, and AGENTS.md, README and the served guide say the rewrite fires when
  the file part holds a `/`. A leading-slash `--exclude` and an attached `-C/path` are unchanged.
  **The leading-slash `--exclude` is refused by ADR-078 T2** on both surfaces: a glob matches
  root-relative paths and base names, and none is rooted — so on Windows the drive MSYS makes of
  `/vendor` is refused the same way, as are `./vendor` and `vendor/`. The attached `-C/path` is
  permanent (fact: MSYS rewrites argv before mrw starts; both switches that stop it are named).
- **The Go suite under PowerShell.** Eleven check tests FAIL instead of skipping when `sh` is not
  on PATH (green from Git Bash on the same tree), and `TestTailAnnouncesWhatItLeftOut` panics.
  All thirteen padded-path tests SKIP on NTFS because their fixture needs a file named `x `. Owed:
  a fixture that does not need the file, since every refusal fires before any I/O. `go test ./...`
  hung once after `internal/lines` (n=1; every package passes alone; not retried as a whole under
  either shell). `-race` needs cgo there, and `contract.sh` needs `jq`.
  **Fixed by ADR-071 T4** for the check tests, the tail panic and the padded fixture (eight of the
  thirteen no longer need a file named with trailing whitespace; five serve one and keep their skip). The
  one hang of `go test ./...` under PowerShell is not reproduced (n=1) and stays open.
  **Re-measured against v1.27.0 (a Windows peer, 2026-09-26, PowerShell 7.6.6, go1.24.2, Windows 11
  26200):** three `go test ./... -count=1` runs finished in 187–214 s with no hang — the n=1 hang is
  closed. All three exit 1 with the same 13 FAILs, every one "could not start: exec: \"sh\":
  executable file not found in %PATH%": eleven in `cmd/mrw` (among them ADR-080's two cancel-before-
  start tests), one in `internal/adversarial`, one in `internal/check`. With Git's `usr\bin` on PATH
  the three packages pass. So "Fixed by ADR-071 T4 for the check tests" does not hold on a plain
  PowerShell PATH: those tests need `sh` and fail instead of skipping. **Fixed by ADR-082**: on
  Windows with no `sh` on PATH the check runs under the `sh.exe` Git for Windows installs beside
  `git.exe`, so those tests run under plain PowerShell; with no shell at all they skip, naming why.
  Still skipped on every Windows, found in the review of #247: tests whose Windows skip says "the
  check runs through sh" — `TestATimedOutCheckKeepsItsLog`, `TestATimedOutCheckNamesItsLog`,
  `TestTheReceiptIsOnStdoutBeforeTheCheckStarts`, and the ADR-054/060/061 matrices. ADR-082 makes
  that reason stale; each needs its own Windows run before its skip becomes `needShell` (deferred: M
  to schedule, since each may meet Windows timing or path behaviour the unix run never shows).
  The same peer round (v1.27.0 asset): Hidden and read-only files and MCP `specs:["x "]` hold; bare
  `NUL` was refused but `nul.txt`, `con` and `COM1.txt` were created — **fixed by ADR-081**
  (#243): every reserved name is refused by name, in any component and through a link. The OneDrive
  placeholder probe was not run (the peer's user has not consented to touching a synced folder):
  deferred until they do.
  mrw itself behaves honestly without `sh` — exit 2, "check SKIPPED: could not start … declare one" —
  but a PowerShell user without Git's `usr\bin` on PATH gets no default check.
  **ADR-081 verified on Windows** (the same peer, v1.27.1 asset, 2026-09-26): `NUL`, `con`,
  `nul.txt`, `COM1.txt`, `COM¹.txt`, `LPT²`, `CONIN$`, `conout$.txt`, `AUX.tar.gz`, `sub/NUL`, a
  directory component `con/b.txt` and a rename into `nul.txt/b.txt` are each refused, NOTHING
  WRITTEN, no stray entry; `console.txt` is created. The link path (a symlink to `con.txt`) is
  unverified on Windows: the peer had no symlink privilege, and CI does not show whether
  `TestALinkToADeviceNameIsRefused` ran or skipped. Open (deferred: needs a Windows run with symlinks).

Smaller, recorded as found: an in-root symlink is followed on write and the target is absent from
the receipt; plan-header paths are cleaned rather than refused (`a.txt/`, `"a.txt"`, `./a.txt`, and
a rename to `d/` makes a FILE `d`); inserts into an empty file produce no trailing newline; a rename
receipt carries an empty sha and creates missing directories silently; a killed write is missing
from `stats`; `NUL` reads as an empty file; unlink deletes a read-only file; a Hidden attribute is
stripped by a write; every header parse error is followed by a bogus "text before the first @@
header"; a tab-separated header is reported as no header; a nonexistent `-C` root reports "resolves
to /"; a file literally named `c:1-2` is unreachable even after `--`; `--files-from` dies on a line
over 8 MiB with no line number; `a.go:$-1` and `a.go:5-3` get different exit classes; `-C N` is
ignored on a numeric range; over MCP, ids of any JSON type are accepted, invalid UTF-8 becomes
U+FFFD so the engine looks for a path never sent, a bad flag prints usage on stdout at startup,
100,000 specs block the server past 120 s, and `exclude: ["["]` is not refused.
ADR-074 T4 names the `--files-from` line that exceeds 8 MiB.
**Fixed by ADR-076** (2026-09-26), contract §151–§153: the symlink target is named in the receipt
(`files[].target`); `a.txt/` and a rename to `d/` are refused, while `"a.txt"` and `./a.txt` are
cleaned as designed (ADR-005 §1, `TestTwoSpellingsOfOnePathAreOneFile`, `TestQuotedFieldsSurvive`);
an insert into an empty file ends with a newline; a rename or a create names the directories it
made (`dirs_created`), and a removed file's line gives its former sha; `NUL` is refused as a device;
a read-only file is refused for every op that would change it, and a write keeps Hidden and System;
a missing `-C` root is named as missing. The killed-write window is closed for the check by ADR-072
T2 (§143); the milliseconds between the commit rename and the tally are permanent: SIGKILL cannot be
handled.
**Fixed by ADR-078** (2026-09-26), contract §155–§159: a header that does not parse is one error, not
one per body line, and a tab after `@@` is named; `-C` with no `/pattern/` is refused rather than
ignored; over MCP an id that is neither a string nor a number with an integer value is refused `-32600`, invalid UTF-8 in
an argument is refused by name, a usage error writes nothing to stdout, a named read of many specs
stops once it has overflowed, and `exclude: ["["]` is refused as the CLI refuses it. `c:1-2`, `$-1`
against `5-3` and a FIFO list stay as ADR-074 decided (its Out of Scope).

Found by the review of #228 (Windows, from the documentation): Win32 also maps device names —
`CON`, `NUL`, `AUX`, `PRN`, `COM1`–`COM9`, `LPT1`–`LPT9`, and before Windows 11 the same names
with an extension (`nul.txt`) — to devices, so a plan naming one writes to a device, not a file.
ADR-071 refuses a trailing dot, a trailing space and a `:`; device names are the same class and
are not yet refused.
**Fixed by ADR-076 T1**: a name the OS opens as a device (`GetFullPathName` answers `\\.\NAME`) is
refused in a spec, a plan path and a rename destination; `nul.bin`, a file on Windows 11, is not.

Found after the list: every padded-path refusal suggests its fix in POSIX single quotes
(`mrw read -- 'x '`), which cmd.exe keeps as literal characters, so a cmd.exe user who pastes it
names a path with quotes in it (a Windows cmd.exe session, 2026-09-25).
**Fixed by ADR-078 T1**: on Windows each padded-path refusal also names the double-quoted form
(`in cmd.exe: mrw read -- " x"`), and the served guide and AGENTS.md say so; the POSIX form stays
first for PowerShell and Git Bash.

Not a finding: `a.txt:-1` serves line 1; `-M` is documented as "from the start to M".

Found while fixing it (ADR-072, contract §143): the contract's hang guard, `perl -e 'alarm shift;
exec @ARGV'`, cannot stop mrw. A Go program ignores SIGALRM unless it asks for the signal (the
runtime's signal table marks it notify-only), so a row that wraps `$MRW` in the alarm is unbounded
if mrw hangs; it bounds only non-Go children such as §55's Python hook. §143 kills mrw with SIGKILL
instead. The rows that still wrap `$MRW` in the alarm need a guard that can actually fire.
**Fixed** (2026-09-26, housekeeping): a `bounded` helper in `contract.sh` runs the command in the
background, polls it, and kills it with SIGKILL at the bound; §111 and §116 use it, and the
check that nothing of the run survives moved to the end of the file so every row is covered.

From the review of #229 (ADR-072), outside that record: a `--dry-run` is tallied as `applied`
and so counted among `landed writes` in `mrw stats`, though nothing landed; `main` did the same
before ADR-072. And a signal that lands between the check's signal handler being installed and
its process starting reports "could not start: context canceled" (exit 2, with advice to declare a
check) rather than "interrupted"; the window is microseconds, and nothing reaches it in a test.
**The signal window is fixed by ADR-080 T2**: a check cancelled before its process starts reports
`interrupted`, exit 3, like one stopped while it ran — and a context cancelled beforehand reaches it,
so a test does (`TestACheckCancelledBeforeItStartsSaysInterrupted`).
Found by the race suite on #241: `TestAnInterruptedCheckSaysSo` (ADR-072 T4) cancels at a fixed
300 ms and expects the check to have started by then; under the whole `-race` suite `sh` started
later, the cancel reached a check that never ran, and it failed (once in the full run, once in
three isolated runs; six of six passed on main and on the branch when idle). ADR-080 first pruned
old logs before the start, which widened the window; pruning now runs after the check exits.
**CLOSED 2026-09-27:** the test now cancels once the check has touched a start marker in its root.
The check has 5 s to start, and a start that never comes fails by name ("never wrote its start
marker"), not as a false `Ran=false`; the separate 5 s return bound runs from the cancel. The body
change moved the locks of ADR-072 T4, ADR-074 T1, ADR-080 T1 and ADR-080 T2, re-taken with
`--relock --replace-hashes` in the same change.
**The dry run is fixed by ADR-079**, contract §161: a clean `--dry-run` records nothing on either
surface (MCP counted every dry run as `refused_apply`), and a refused one is one refusal.
- ~~**Filesystem-error refusals are tallied on one surface**~~ **CLOSED 2026-09-27 — by ADR-083**,
  contract §164: a plan the CLI refuses after it parsed and before anything landed is one
  `refused_apply`, as `mrw_write` counts it, dry run or not. (The review of #240, older than ADR-079):
  a plan refused by a filesystem error before any hunk is judged — a path that names a directory —
  exits 2 on the CLI before the tally is reached, while `mrw_write` counts it as `refused_apply`, dry
  run or not. Deferred: which bucket such a refusal belongs in is ADR-009's call, not ADR-079's.

Found while ranking this backlog (2026-09-26): with `XDG_STATE_HOME` inside the root, or under
`--root "$HOME"`, a `--grep` walked mrw's state directory, a read served the ledger and the ack
store — whose checkpoint ids could then be acked without the lines ever being read — and a plan
could edit the ledger. **Fixed by ADR-077**, contract §154: a path inside the state base is refused
at the boundary, and a discovered one is dropped.

## From ADR-092 (each step after a write gets its own verdict)

- **Steps on the MCP surface** (ADR-092 Out of Scope). `mrw_write` runs no check today, and running a
  caller-named shell step from a host that may have withheld shell is a trust decision ADR-092 did not
  take; ADR-016 records that a permanent MCP boundary contradicted M's stated direction, so this stays
  deferred. Arm when an MCP-only host asks for verified writes.
- **A per-step timeout or tail setting** (ADR-092 Out of Scope). Every step takes the harness's
  `timeout_seconds` and `tail_lines`. Arm when one declared step needs a different bound from the check's.
- **From the ADR-092 stress round (2026-09-28, six local peer sessions on d8576c3).** Fixed by
  T5: a malformed `steps` block refusing every write; unbounded recursion; raw control bytes in step
  names; "could not start" said twice. Recorded here, lower, each with what arms it:
  - A step killed by a signal reports `exit_code: -1`, the not_run sentinel, and names no signal
    (Go's ProcessState). Arm when a caller needs the signal.
  - A duplicate key in `steps` is last-wins, and top-level keys fold case (`"Steps"`, `"ſteps"`):
    encoding/json. Arm on one real config that hit it.
  - A passing step's kept log (output past `tail_lines`) has no size cap (a 60 MB one was kept);
    ADR-080's 7-day prune is the only bound. Arm when a temp directory fills.
  - A dangling-symlink `.quality-harness.json` reads as no config, so `--then a` says "none are
    declared" of a config that is there. Arm on the first report.
  - A plan that edits `.quality-harness.json` is judged, and its steps resolved, against the
    pre-write config (ADR-072's order); the receipt is truthful about what ran. Arm when a re-read
    after the write is wanted.
  - A blank `--then-sh` is refused before the plan parses and so is not tallied; an unknown `--then`
    is, after it (ADR-083's line). Arm if `stats` should see both.
  - `stats` and `seen` on a root that does not exist answer empty, exit 0, where `read` refuses —
    older than ADR-092. Arm with the next could-not-look pass over `stats`.
  - A step whose cwd was removed reports "fork/exec /bin/sh: no such file or directory" (Go's ENOENT
    attribution). Arm on a real report.
  - A check's or step's tail reaches the human receipt with raw control bytes (T5 quotes names and
    commands, not output). Arm with a terminal-safety pass over tails.

## From ADR-093 (an MCP tool refuses an argument it does not declare)

- ~~**A line cap or a stat-only answer on `mrw_read`** (ADR-093 Out of Scope).~~ **CLOSED 2026-10-02 —
  ADR-117: `mrw_read` takes `max_lines` and `stat`, and `files_from` with them, on Zy's answer to the
  refreshed gap list.**
- **What opencode does with an argument the plugin's zod shape does not declare** (ADR-093 Out of
  Scope). The plugin forwards only a fixed map of keys, so a model's undeclared key (`context`, since
  ADR-117 declared `max_lines`) never reaches mrw, and whether opencode strips or refuses it first was
  not read. Arm by measuring whether opencode strips or refuses it before the plugin's `execute` runs.
- **Whether a host forwards an undeclared key, or enforces `additionalProperties: false`** (ADR-093
  Out of Scope). Nobody has captured a host's wire for a call carrying an undeclared key, Claude Code
  included. Arm by capturing one; a host that adds keys of its own to `arguments` is ADR-093 T1's
  Stop Condition.

## From ADR-096 (a path the caller names is never dropped by a finder)

- **Real ast-grep on an accepted named path** (ADR-096 Out of Scope). What the real binary does with an
  accepted path through a directory link (`dlink/sub`) and with an absolute operand is exercised only
  by the fakes here: ast-grep was not installed where the record was drafted. Arm on the first run
  against a real ast-grep, or a report that a hit's name did not map.
- **A named Windows junction as a walk start** (ADR-096 Out of Scope). The refusal asks `Lstat`'s
  `IsDir`, which a junction answers false (ADR-071), so it is caught by construction; no Windows runner
  exercises it. Arm with the next Windows-only record that touches the walk.
- **`mrw check PATH` dropping a path it cannot place** (the 2026-09-29 gap survey, item C8; ADR-096
  Out of Scope). It maps paths to packages rather than finding files, so it is a separate record. Arm
  when that record is drafted.

## From ADR-094 (a step says what it checked)

- **A placeholder in the `check` field** (ADR-094 Out of Scope). `check` runs as written, as a step does,
  so a `check` holding `{files}` or `{packages}` runs the literal text; a passing check already prints its
  tail, so the probe of 2026-09-29 showed `| lstat {files}: no such file or directory` above `check PASS`.
  Refusing it means refusing at `Load`, which every write reads (ADR-072), or after the write landed — a
  different weighing on a surface ADR-054 and ADR-061 own. Arm on the first real config that holds one.
- **The `then` head line quotes a command only for control bytes** (ADR-094 Out of Scope). `shown()`
  Go-quotes a command whose `strconv.Quote` differs from it, so one holding `"` or `\` is printed quoted
  too, not only one holding control bytes. Arm with a terminal-safety pass over receipts.

## From ADR-095 (a nested mrw stays bounded)

- **Stopping a grandchild on Windows, which needs a job object** (ADR-095 Out of Scope). Windows has no
  process groups, so a cancel there kills only the direct child, nothing hears TERM first, and a nested
  mrw's check outlives it as any grandchild does. Carried with ADR-080's and ADR-072's deferral of the
  same job object; arm when a Windows caller nests mrw or reports an orphaned check.

## From ADR-097 (a flag named for a subcommand names the subcommand)

- **`mrw read help` and `mrw read h` print `read`'s help** — **Closed** by ADR-099 T2 (`HideHelpCommand` on read, write and check). (ADR-097 Follow-up, observed 2026-09-29 against
  v1.31.0). urfave appends its `help` command (alias `h`) to every command, and it wins over a
  positional of that name, even after `--`: a file named `help` or `h` is reached only as `./help`. Arm
  on the first caller who names such a file, or with the next record that touches argument dispatch.
- **The doc comment `// instructionsCmd prints…` sits above `func versionCmd`** in `cmd/mrw/main.go`,
  and `func instructionsCmd` carries none (ADR-097 Follow-up, found while drafting). **Closed**
  (2026-09-29, the hygiene PR after v1.32.0): each function carries its own comment.

## From ADR-100 (every `mrw check --json` refusal is a document)

- **`write --json` refusals before the plan is named** (ADR-100 Out of Scope). Edge whitespace
  (`refusePaddedArgs`), `--check` with `--dry-run`, `--check` with `--no-check`, a negative `--echo-pad` and a
  second plan file (`cmd/mrw/main.go` `writeCmd`, audited 2026-09-30) print only a message under `--json`.
  ADR-072 T3 drew its line "after the plan is named" on purpose: these are flag contradictions before any
  receipt exists. Arm on the first `--json` caller that parses stdout from one of them.
- **`stats --json` state-read failures** (ADR-100 Out of Scope). `authoring.Reset` and `authoring.Load`
  errors (`statsCmd`) print only a message. `stats` reports the tally rather than a verdict a caller branches
  on. Arm with the first consumer that reads `stats --json` programmatically.
- **`exit_code` 0 beside `"ran": false` in `check --json`'s receipt** — **Closed** by ADR-101 (a check with no exit status reports -1 in both receipts, 2026-09-30). (ADR-100 Out of Scope). When no
  check could run (no harness, no `go.mod`) the receipt is `{"ran": false, "skipped": …, "exit_code": 0}`
  and mrw exits 2 (probed 2026-09-30). A consumer that reads `exit_code` alone reads a pass. Omitting the
  field when nothing ran changes the receipt shape ADR-054 and ADR-092 pin; arm on the first consumer that
  branches on `exit_code`.

## From ADR-102 (a write that landed is reported and counted as landed)

- **Sharing the rest of the CLI/MCP write orchestration** — **Closed** by ADR-113 (`internal/writer/flow.go`: one
  sequence, `Prepare` → `Land` → `Verify` → `Settle`, that both surfaces run, 2026-10-01) (ADR-102 Out of Scope; the
  2026-09-30 Codex design review's top-5 #5). `cmd/mrw/main.go` and `internal/mcp/tools.go` each prepared a plan,
  applied it, counted it and rendered it; ADR-102 shared only the classification (`writer.MutationOf`). Armed by the
  2026-10-01 gap list: an MCP write ran no check.
- **The cleanup errors mrw ignores, and what a failed run leaves behind** — **Closed** by ADR-105 (`left_behind`, atomic state writes, the README failure matrix, 2026-09-30) (ADR-102 Out of Scope; the 2026-09-30
  Codex design review, finding 5). `apply.go` `discard`, `stageFile`'s error paths and `pathop.go`'s aside removal
  drop a failed `os.Remove`, so a `.mrw-*` temp file or aside can remain unreported; state files are written in
  place. Planned as ADR-105 (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`): a `left_behind` receipt field,
  atomic state writes, a failure matrix.

## From ADR-103 (one checkout, one identity; and the filesystem root is a root)

- **Anchoring file operations to a handle** — **Closed** by ADR-106 (the write path through `os.Root`, and an identity recheck before each commit rename, 2026-09-30) (ADR-103 Out of Scope; the 2026-09-30 Codex design review, finding
  3). `rooted.Resolve` validates a path and returns it; read, stage, commit and path ops reopen it by name, so a
  link swapped in between can redirect them. Planned as ADR-106 (`~/.claude/plans/ok-create-a-plan-whimsical-newell.md`):
  the write path through an `os.Root` handed the resolved path, and a pre-commit identity recheck.

## From ADR-108 (what an agent sends arrives as sent)

The 2026-10-01 Codex design review (gpt-6-astra xhigh, v1.37.1) listed five robustness improvements beside the
seven defects ADR-108 fixed. None is a defect today; each has the trigger that would make it one. ADR-109's Out of
Scope defers B1–B5 here too, ADR-110's defers B1, B2, B4 and B5, ADR-111's defers B1, B2 and B5, and ADR-112's defers B1 and B2.

- **A foreign-compiled plan carries `sha=`** (B1) — **Closed**: the premise did not hold. Raised by the 2026-10-01 Codex
  design review. The read ledger records each file's whole sha when it is read, and apply refuses a file whose sha
  differs from it before any hunk (`internal/apply/apply.go:1039`, "changed since mrw last saw it"); a compiled
  `apply_patch` or `search_replace` plan goes through the same check, so a file changed after the caller's read is
  refused, exit 1, unless `--force` is passed, which bypasses the guard on purpose for every format (MCP has no force).
  Measured 2026-10-01 on v1.37.2 for both formats, and pinned by contract §212.
- **Combination tests and a host matrix** (B2) — the combination half **Closed**: `docs/break/write-flags/stress.py`
  drives seeded products of the write flags (`--json`, `--dry-run`, `--check`/`--no-check`, `--then-sh`,
  `--strict-balance`, `--echo-pad`) across native, `apply_patch` and `search_replace` plans and three check
  configurations, against seven invariants; 800 runs on two seeds, 0 violations, 2026-10-01. The host-matrix half
  stays deferred: the MCP arm is measured on Claude Code alone, and no second MCP host is installed here. Arm on a
  second host, or when a defect is found in a combination the harness does not drive.
- **A timeout on the writer lock** (B3) — **Closed** by ADR-110 (`state.HoldWithin`: a writer waits `MRW_WRITE_LOCK_TIMEOUT` seconds, 120 by default, then is refused naming the holder's pid; contract §209, 2026-10-01). A writer waits for the one before it (ADR-075) without a bound; a stuck
  writer blocks the next indefinitely. Arm when a wait is reported that ended only by killing a process; the
  refusal then says nothing was applied.
- **A receipt compatibility policy** (B4) — **Closed** by ADR-111 (`docs/receipts.txt` lists every receipt key; two tests hold the receipt types to it both ways; contract §210, 2026-10-01). Receipt fields are only ever added (ADR-054, ADR-102), but no record
  says what a caller may rely on across versions or how a removal would be announced. Arm on the first receipt
  change that is not purely additive.
- **A drift advisory during a check** (B5) — **Closed** by ADR-112 (`writer.Drift`: a CLI write names each file it touched that changed while its check ran, `drift:` and `drift` in `--json`; contract §211, 2026-10-01). A file edited by someone else while mrw's check runs is not reported;
  the check's verdict is about a tree that may no longer exist. Arm when a green check is reported over a tree
  another writer changed during it.
- **A FIFO swapped in between a loader's check and its open** — **Closed** by ADR-109 (`internal/regular.Open`: every loader opens without blocking and asks the descriptor; it also found `check.Load` hung outright on a FIFO `.quality-harness.json`, contract §207, 2026-10-01) (the Codex re-review of #304, residual). `ingest`'s
  `targetBytes`, `plan`'s `LoadBodyFiles`, `read`'s `readCapped`, `apply`'s `readLines` and ast-grep's probes judge
  a path by `Stat` and then open it blocking, so a file replaced by a FIFO in that window hangs the call until
  something writes to the pipe. ADR-108's A6 and A8 open non-blocking and check the descriptor; these were not
  changed, since two are engine packages ADR-108 does not own and the window is a swap inside the checkout. Arm
  when a hang is reported that ends at such a swap, or when a record next owns `read` or `apply`.

## From ADR-113 (one write path, and an MCP write is checked)

- **Steps over MCP** — **Closed** by ADR-115 (`mrw_write` takes `then`, declared step names only, 2026-10-02). `mrw_write`
  ran the check and no step: `--then-sh` is arbitrary shell, and a named `then` needed its own floor and elision on
  that surface (ADR-113 Decision 6). Armed by the gap list of 2026-10-02, item 2.
- **Progress notifications while an MCP check runs** — Claude Code's 30-minute idle window for a stdio server
  resets on a progress notification; the hard limit (`MCP_TOOL_TIMEOUT`, about 28 hours by default) does not. The
  check's default bound is 5 minutes, inside both (ADR-113 Context). Arm when a project's check bound past the idle
  window is reported, or a host is found whose idle window is shorter than the check.

## From ADR-116 (a walk honours .gitignore and skips binaries)

- **`--ast-grep` / `ast_grep` and the ignore rules** — ast-grep walks with its own ignore handling, and mrw drops
  excluded hits after it answers; `--no-ignore` does not reach it, so the two finders can disagree on an ignored
  file. Arm when a caller reports `--ast-grep` serving a file `--grep` skips, or the reverse, or asks for
  `--no-ignore` with `--ast-grep`.

## From ADR-117 (mrw_read takes max_lines, stat and files_from)

- **`--context` and `--no-numbers` over MCP** — the CLI's remaining read extras; the routing names them. Arm when
  an MCP caller shows it needs context around a match beyond what a regex address and a range give, or an
  unnumbered serve (a numbered line is the address a plan writes back, so unnumbered serves license nothing).
- **Checkpoints from a version a writer replaced mid-read are still printed** (ADR-117, the review of #318). A
  read of several specs of one file that a writer swapped between them prints every slice's markers, but only the
  observed version's are held; acking the others licenses nothing, and the caller learns it when a write is
  refused. Arm when a caller reports a refused write after acking such a page, or carry the full sha out of band
  (which also closes the 8-hex prefix collision the record's Risks names).
- **A "has not been read" refusal lists a served line once per spec** (seen in the review of #318: "mrw served
  lines 3,3,3,…" for 400 specs naming line 3). Not introduced by ADR-117. Arm when a caller reports the
  refusal as unreadable, or a receipt grows past its ceiling on it.

## Pre-registered for ADR-119 (indent and closer advisories) — the bar, written before any measurement

Zy, 2026-10-02 ("<5% and 5/5 fixtures"). Registered here, in its own commit, before the heuristics are measured,
so the result cannot move it:

- **closer** — fires on a `replace` whose last non-blank body line equals the line right after the replaced range
  (when that line exists and is not blank). **indent** — fires on a non-prose `replace` whose first or last
  non-blank body line's leading whitespace differs from the first or last replaced line's.
- **False positives:** replayed from the non-merge history of every repository under `~/GolandProjects` and
  `~/CursorProjects` (each modification hunk of `git diff -U0 k^ k` as a replace, vendored, generated, lock and
  `node_modules` files excluded, hunks capped per repository), each heuristic must fire on **under 5% of those
  replaces in every language bucket** that has at least 50 of them.
- **True positives:** each must catch **5 of 5** field fixtures (the offsets recorded under "From the field
  reports" above: Blade `@endif`, HTML `</div>`, markdown fence, YAML block scalar, Ansible `when:`) — closer the
  closer cases, indent the indentation cases, and together all five.
- A heuristic that misses its bar does not ship; the record says so with the numbers.
- **Amended after the first measurement (Zy, 2026-10-02, "K4 closer, defer indent").** The first run (seed 119,
  400 replaces a repository, 3,475 replaces from 16 repositories) missed the bar on both registered heuristics:
  closer caught 2 of its 3 fixtures — the markdown fence's survivor is the fourth line after the range, which a
  line-right-after comparison cannot reach — at 0.00% everywhere; indent caught both of its fixtures but fired on
  19.35% of `.py`, 14.29% of extensionless, 11.89% of `.js` and 8.99% of `.php` replaces. Registered now, before
  the second run, and chosen AFTER seeing the first run's numbers, which is why it is measured again on a fresh
  sample rather than shipped on those: **closer** fires on a `replace` whose last non-blank body line, trimmed,
  is a closer-shaped token (` ``` `/`~~~` fences, a run of `}` `]` `)` with trailing `;` `,` `)`, `</tag>`,
  `@end…`, `end`, `fi`, `done`, `esac`) and equals one of the next 4 non-blank lines after the range, trimmed. The
  bar is unchanged: under 5% in every bucket of 50 or more, and 3 of 3 closer fixtures. The second run uses seed
  2119, 1,200 replaces a repository and 6,000 commits sampled a repository (`stress.py --seed 2119 --cap 1200
  --max-commits 6000`), so it overlaps the first run's hunks only partly. **indent** is withdrawn (see "From
  ADR-119").
- **Amended after review (Zy, 2026-10-02, "Refine and re-measure").** The in-process review of PR #322 found
  that run 2's corpus could not contain the commonest correct shape: `git diff -U0` hunks never hold an unchanged
  closer, so a replace THROUGH its own closer (`if x {…}` replaced through its `}`, the outer `}` below) was never
  replayed, and the shipped closer fires on it. Registered now, before the third run: **closer** additionally
  requires that the replaced range's last non-blank line, trimmed, is NOT equal to the body's last non-blank line,
  trimmed — the body brought a closer the range did not end in. The corpus gains a second replay: each `-U0`
  replace whose next 4 non-blank original lines hold a closer-shaped line is ALSO replayed extended through the
  first such line, its range ending there and its body carrying the same unchanged lines — the shape callers are
  taught to write. The bar is unchanged and applies to EACH replay separately: under 5% in every bucket of 50 or
  more, and 3 of 3 closer fixtures. The third run uses seed 3119, 1,200 replaces a repository and 6,000 commits
  (`stress.py --seed 3119 --cap 1200 --max-commits 6000`).

## From ADR-119 (a write says when its shape looks wrong)

- **A short address that orphans lines ABOVE the body** — the field reports measured `3104-3108` written where
  `3088-3108` was meant, leaving sixteen lines above the body; neither hint looks above the range. Arm when a
  second such report arrives, or a cheap check for it is proposed with a measured false-positive rate.
- **The indent hint** — withdrawn after the first measurement (above): 9–19% false positives in `.py`, `.js`,
  `.php` and extensionless files, and the language it exists for, YAML, had 18 replaces in the corpus, under the
  50 the bar judges. Arm when a corpus of 50 or more YAML replaces is available to measure a YAML-only indent hint
  against the same bar, or a second field report of an indentation failure arrives.

## From the Windows peers (2026-10-02, v1.42.0)

Five Claude sessions on one Windows 11 Pro 10.0.26200.9457 desktop (NTFS, Git for Windows 2.49.0 with bash 5.2.37,
Developer Mode off) probed v1.42.0 on request, report-only. Zy, 2026-10-02: "BACKLOG now, records later". What held
is listed last; each item names what arms it.

- **A held target turns a plan PARTIALLY APPLIED, depending on plan order.** A file another process holds open
  without `FILE_SHARE_DELETE` (an editor, AV, a language server) makes the commit's rename fail ("Access is
  denied"). Held last or in the middle, the earlier files have landed and are not undone: PARTIALLY APPLIED, exit
  2. Held first: NOTHING WRITTEN. A rename or unlink of a held file behaves the same. ADR-066 calls a mid-commit
  failure rare; on Windows it is ordinary. A probe follow-up found that an open with DELETE access, or a
  rename-to-self, fails early (err 32) for every holder without delete sharing. But with `ReadWrite,Delete`
  sharing both probes pass while the real `MoveFileEx(tmp, path, REPLACE_EXISTING)` still fails (err 5).
  Candidates: a pre-commit probe, plus `FileRenameInfoEx` with `FILE_RENAME_FLAG_POSIX_SEMANTICS` for the
  delete-sharing case (untested); any probe is check-then-act. **Arm now** as the next Windows record: it breaks
  ADR-001's all-or-nothing promise in a common case.
- **An exclusively held file, or an invalid name, gets no receipt.** A holder with share mode None, or
  `@@ q?.txt 0 create`, prints one bare `mrw: …` line, exit 2, with no per-hunk verdicts. Nothing is written.
  Arm with the record above, which owns that path.
- **A file-level ACL deny says "send the plan again".** `icacls /deny (W,D)` on a target gives exit 1 and "its
  identity could not be read … send the plan again", which can never help. The cause, permission, is not named.
  Arm with the record above.
- **A case-only rename is impossible.** `a.txt` → `A.txt` is refused as "dest already exists". Following the
  advice (unlink `A.txt` first) is then refused as "names the same file". Nothing is lost. Arm on a second request.
- **The write lock is not held while the check runs.** A second writer's edit to another file landed 1.4 s into
  the first writer's 8 s check, so the first check verified a tree holding an unverified edit. ADR-112's drift
  advisory watches only the files the first write touched. Arm when the lock's scope is next revisited, or with
  ADR-121 (the check off the only MCP thread), which changes when a check runs.
- **`--grep` costs 10–14 ms a file on Windows.** 3,000 files of 40 lines took 31–63 s, from the CLI and over MCP,
  with output to a file and an MCP index serving no lines. `grep -rl` took 0.41 s; the same shape on macOS 0.29 s.
  The `.gitignore` matcher is ruled out (`.git` or not makes no difference). A peer's timeout stack points at
  `read.Run` → `rooted.Resolve` → `InState` → `RealAsFarAsItExists` → `filepath.EvalSymlinks` per served path
  (ADR-077's state-directory check). The same stack timed out `internal/mcp`'s
  `TestAnIndexTooLargeToServePagesByFile` at 6m31s under a full Git Bash run (127–169 s alone). **Arm now**,
  with ADR-123 (grep matches while reading), which owns the walk's per-file work.
- **A UTF-8 BOM on the first MCP line drops `initialize`** (-32700). CRLF alone is fine. Arm on the next MCP
  handshake change.
- **MCP refusals say "pass --force"**, which `mrw_write` does not take. An unknown ack id is ignored silently. A
  replace whose body equals the line reports `written: true` with equal shas. Arm with the next MCP text change.
- **A walk enters a nested untracked repository.** git lists it as `nested/` and does not descend; mrw walks in
  (applying that repository's own `.gitignore`). And mrw does not read `core.ignorecase`: with it set to false,
  `[Ab]*.txt`, `MIXED.txt` and `Upper/` still fold, as the filesystem does. Both are ADR-116's documented rule,
  which differs from git here. Arm on a second report, or with ADR-122 (ast-grep walks mrw's walk).
- **Process trees on Windows** are ADR-120's: a timed-out check kills only its direct `sh.exe`, and a killed mrw
  kills nothing below it. The orphans' parents are dead MSYS fork stubs, so a parent-PID walk would miss them, and
  they hold the check log open. And Git for Windows' sh asks for `CREATE_BREAKAWAY_FROM_JOB` whenever a job
  allows it: with `JOB_OBJECT_LIMIT_BREAKAWAY_OK` nothing below sh stays in the job. Zy, 2026-10-02: "Drop
  BREAKAWAY_OK". Carried into ADR-120.
- **Fixed with this entry:** `TestMrwReadTakesMaxLinesStatAndFilesFrom` failed on a desktop without symlink
  privilege. Its link row is now dropped, with a log line, when a symlink cannot be created.
- **Held:** read-only and hidden files, junctions out of the root, a 392-character path, trailing dots and
  spaces, `::$DATA`, device names, non-ASCII names (Я, Ŝ), one file under two spellings, CRLF kept per line,
  rename onto an existing file, `files_from` with backslashes and CRLF, `max_lines` and `stat` licensing, checks
  with and without `sh`, steps, huge-read pages, every `.gitignore` rule tried under git's defaults, and
  `occurrence=N` on LF, CRLF and a backslash pattern. Not covered: symlinks (no privilege), a console Ctrl+C.
