# mrw — multi-path read and write

Read many ranges across many files, and apply many edits across them, in one
invocation — and get a verdict for every edit. A plan that fails validation
writes nothing, because a write that changed nothing is invisible.

The numbers — two calls for any N, shapes A–D — live in [docs/measure.md](docs/measure.md). The six-guarantee comparison lives in [docs/comparison.md](docs/comparison.md). Model × score readings live in [docs/model-benches.md](docs/model-benches.md).

Whether mrw pays for itself on agent runs is measured against a bar registered before any run ([ADR-156](docs/adr/ADR-156-whether-mrw-pays-for-itself-is-measured-on-agent-runs.md); every verdict and row in [docs/bench/RESULTS.md](docs/bench/RESULTS.md)). Every verdict so far is **INCONCLUSIVE** but one, Stage 4 on mrw v1.66.0, which **LOSES**:

- **Stage 1**, Sonnet 5.5, 30 own tasks: INCONCLUSIVE, cost (B−A)/A +2.6% [−3.9%, +9.0%]. Configuration B made more silent mis-edits than A, 4 against 2, all on three tasks later superseded as misgraded (ADR-158, ADR-159).
- **Stage 1b**, Sonnet 5.5, 40 own tasks: INCONCLUSIVE, cost −0.4% [−5.0%, +4.2%].
- **Stage 2**, Opus 5.5, 40 own tasks: INCONCLUSIVE, cost −2.7% [−7.8%, +2.1%]. B made more silent mis-edits than A, 2 against 0: both renamed a test function beside the asked rename, and said so.
- **Stage 3**, 20 SWE-bench Verified instances: Sonnet 5.5 INCONCLUSIVE, cost −6.1% [−17.0%, +4.6%]; Opus 5.5 INCONCLUSIVE, cost −5.3% [−22.9%, +14.8%].
- **Stage 4**, Sonnet 5.5, 24 own tasks of the kinds where mrw's mechanism should matter ([ADR-160](docs/adr/ADR-160-stage-4-measures-mrw-where-its-mechanism-should-matter.md)), on mrw v1.66.0: LOSES, cost +17.1% [+9.6%, +24.7%]. B made more silent mis-edits than A, 1 against 0. The stage stopped there, and was rerun as Stage 4b after [ADR-161](docs/adr/ADR-161-instructions-lead-with-recipes.md) changed what mrw teaches.
- **Stage 4b**, the same 24 tasks on mrw v1.68.0: Sonnet 5.5 INCONCLUSIVE, cost −2.1% [−6.2%, +1.8%]; Opus 5.5 INCONCLUSIVE, cost −1.6% [−4.6%, +1.4%]. On Sonnet, B made more silent mis-edits than A, 3 against 2; on Opus, 2 each.
- **Stage 4b**, 20 multi-hunk SWE-bench Verified instances on mrw v1.68.0: Sonnet 5.5 INCONCLUSIVE, cost −17.0% [−26.6%, −7.6%]; Opus 5.5 INCONCLUSIVE, cost −20.9% [−32.3%, −8.6%]. B cost less on both models, but PAYS also needs the solved-rate interval's lower bound at −5 points or above, and it reached −30 on Sonnet and −20 on Opus.
- **Stage 5**, 191 SWE-bench Verified instances with 2 or more hunks, on mrw v1.68.0, in five blocks with a look after each ([ADR-162](docs/adr/ADR-162-stage-5-runs-until-the-verdict-is-conclusive.md)): Sonnet 5.5 INCONCLUSIVE, cost −7.0% [−12.1%, −1.8%]; Opus 5.5 INCONCLUSIVE, cost −19.4% [−25.5%, −12.7%]. B cost less on both models, but the solved interval's lower bound reached −8.7 points on Sonnet and −6.7 on Opus, short of the −5 PAYS needs, and both pairs were still open when the stage ended at look 5.
- **Stage 5**, configuration P (Claude Code with the quality-harness plugin v3.8.40) against A: LOSES on both models, closed at look 1, cost +70.1% [+47.7%, +92.6%] on Sonnet 5.5 and +48.6% [+31.2%, +67.4%] on Opus 5.5.

Stage 2's attempts solved 156 of 160 and Stage 3's 70 of 80, so the solved difference had little room to move; [ADR-160](docs/adr/ADR-160-stage-4-measures-mrw-where-its-mechanism-should-matter.md) registered the harder Stage 4 above.

**Status: stable at v1.69.0 (2026-10-21), the tag cut from `9cae6d8`.** Break campaign for it: [docs/break/campaign-v1.69.0.txt](docs/break/campaign-v1.69.0.txt), 59 probes, exit codes identical to v1.68.0 but one: `pat-resolves-to-unserved-line` is refused (exit 1), as on v1.67.0. Nothing changed in mrw for it: the v1.68.0 file records the probe's old form, and the v1.68.0 binary refuses the new one too (re-run 2026-10-21).

Decisions: [docs/adr/](docs/adr/). How a change reaches `main`: [CONTRIBUTING.md](CONTRIBUTING.md). Driving it from a checkout: [AGENTS.md](AGENTS.md).
Caller practices: [BESTPRACTICES.md](BESTPRACTICES.md). Updating the binary: [UPDATE.md](UPDATE.md).
Hosts: any shell (the CLI), any MCP host (`mrw mcp`, [below](#mcp)), and [opencode](#opencode) through its plugin.

**Start with the recipes `mrw instructions` prints first** (ADR-161): the same change at many sites is one
command with nothing in your context — `mrw write --grep '\bOld\b' --sub 'New' .` — and the others count
before reading, rewrite by syntax, license without printing (`read --quiet`) and narrow any command's output
(`cmd | mrw read --grep P -`). [AGENTS.md](AGENTS.md#using-mrw--read-this-before-you-edit-anything) lists all nine.

## Install

```sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')   # linux | darwin
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -fsSL -o mrw \
  "https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest/download/mrw-${OS}-${ARCH}"
chmod +x mrw && ./mrw version
```

Windows: `mrw-windows-amd64.exe`. Every release also carries archives
(`mrw_<os>_<arch>.tar.gz` / `.zip`) and a `SHA256SUMS.txt`.

`mrw version` prints the same string `-v` / `--version` already print. Extra
arguments are usage (exit 2). `mrw version --check-latest` also asks GitHub for
the latest release and says whether this mrw is current, newer or older: exit 1
when a newer release exists, 2 when it could not ask, and 0 for a local build,
which it names without a verdict. It is the only network call mrw makes, and
only that flag makes it (ADR-149).

`mrw instructions` prints the contract from the binary: use mrw always and plan
the activity, the two rules that produce most refusals, and the traps that make
a red run look green. Exit 0. No flags. `mrw --instructions` is refused, exit 2,
with a message naming `mrw instructions`; so is any other undefined flag that is
exactly the name of a subcommand of the command it was given to.

### From source

Go 1.26.10 or newer (`go.mod`). One dependency, no cgo:

```sh
go build -o bin/mrw ./cmd/mrw          # Linux, macOS
go build -o bin/mrw.exe ./cmd/mrw      # Windows
```

`go test ./...` needs only Go. `scripts/contract.sh` and `scripts/measure.sh`
need bash (WSL or Git Bash on Windows) — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Read

```sh
mrw read internal/apply/apply.go:1-40
mrw read a.go:1-8,100-130 b.go:/func Handle/,/^}/ c.go --stat
mrw read --grep 'func Handle' -C 3 --exclude vendor --exclude '*_test.go' internal/
```

A range is `3-6`, `5`, `3-` (to EOF), `-20` (from the start), `A,+N` (line `A`
plus the `N` lines after it, so `12,+2` is 12–14), `/pattern/`, or
`/start/,/end/`. `$` is the last line. In a JSON file a path — `file.json:$.a.b`,
`$.a[0]`, `$["k"]` — serves and licenses the lines of that one value; a path that is not
there, or a file that is not one JSON document, is a range not served (exit 1), and a plan
refuses the form, so read it and address the lines (ADR-155). Output ranges print as `@@ 3-6`, which is
the address a write plan takes.
Quote a JSON path for the shell: pwsh 7 takes `'t.json:$["k"]'`, Windows PowerShell 5.1 needs `'t.json:$[\"k\"]'`,
cmd needs `"t.json:$[\"k\"]"`, and Git Bash and other POSIX shells take single quotes; an unquoted `[0]` is a glob (ADR-157).

`--grep` walks and serves in one call. A named directory is walked; with no
paths the walk starts at `--root`. `--exclude GLOB` is repeatable and matches
both the root-relative path and the basename — that is what makes `'*_test.go'`
work at any depth. Inside a git checkout it skips what `.gitignore` and
`.git/info/exclude` ignore, and any walk skips a binary file; a `-- skipped:`
line counts them, and `--no-ignore` walks every file. It does not enter a
nested repository — a directory below the checkout holding its own `.git` — as
git does not (ADR-130); a path you name inside one is walked. A `.git/` the walk meets
is skipped; one you name is walked.
On Windows, a file whose name Windows will not keep (`aux.txt`, `nul`, `trail.`) is not served by a walk and is
counted on the `-- skipped:` line; name it to be told why (ADR-135).
A link to a directory met inside the root is not followed and is counted on the `-- skipped:` line; name the directory
it leads to (ADR-138).
A link to a directory is not followed: one the walk meets is skipped, one you name is refused with
the directory to name instead — unless you spell it as an absolute path, which names the directory
it leads to — and one that leads to the root is walked as the root.

`--files-from FILE|-` is the same idea for a searcher you already trust:

```sh
rg -l 'func Handle' . | sed 's|$|:/func Handle/|' | mrw read -C 3 --files-from -
```

| flag | effect |
|---|---|
| `--stat` | length, bytes and sha only — no content, so it licenses nothing |
| `--count` | with `--grep`: each matching file's header and how many of its lines match, then a total — no content, so it licenses nothing (ADR-153) |
| `-C N` | context around a single-pattern match |
| `--max-lines N` | cap per spec, where zero means zero: `0` serves nothing. Omit the flag for no cap. Use it instead of `\| head`, which loses mrw's exit code: what it withholds is named, and the read exits 1 |
| `--max-cols N` | cut a line longer than N characters to a window of N centred on the first match of a pattern (the start otherwise), holding the whole match when it fits, marked `[cols A-B of L]`; a line served cut is not recorded as read, so a plan that replaces it is refused until it is read whole (ADR-137, ADR-164). `max_cols` on `mrw_read` does the same, and a line it served cut is in no checkpoint, so no ack licenses it (ADR-154) |
| `--grep PATTERN` | serve every regexp match under the given paths (measured 2026-09-03: the walk took 0.76× the time of a `grep -rl … \| mrw read --files-from -` pipeline over this repository, ADR-007) |
| `--ast-grep PATTERN` | serve every `ast-grep` hit (binary on PATH; missing is exit 2; a hang is, on unix, sent SIGTERM at 2 s and killed by 3 s if it ignores it; on Windows it is killed at 2 s). A hit in a file whose lines end in `\r` alone is reported, not served — read that file directly |
| `--exclude GLOB` | skip matching paths (needs `--grep` or `--ast-grep`) |
| `--files-from FILE\|-` | one spec per line |

A pattern that matches no file is reported by name and exits 1.

A read whose standard output is the null device (`>/dev/null`, cmd.exe's `>NUL`) is still
served and exits as usual, but records nothing in the ledger and says so on stderr (ADR-133).
That read adds no licence; it does not remove one. When every line it served in full of a file is
still licensed by an earlier read or write of the same version, under any spelling of that file,
a second stderr line names the file, and a write to those lines is judged on that licence with the
usual checks. A file it leaves unnamed may still be licensed in part; a line no read or write has
licensed is refused as unread, as after `--stat` (ADR-148). mrw cannot see past a pipe or a file,
so those record as before, and so does PowerShell's `> $null` or `| Out-Null`, which hand mrw a
pipe, as does PowerShell 7's `> NUL`. In Windows PowerShell 5.1 `> NUL` fails before mrw starts
(Out-File cannot open a device), so nothing is read. A licence means you saw the lines: an answer
sent to a file you never open licenses all the same, and mrw cannot tell.

The ledger's first line names its format and the mrw that wrote it (`#mrw-seen v5 v1.66.0 (abc1234)`). A
ledger in a format older than v4 is discarded, once, with a notice; v4 and v5 are read. A ledger in a newer
format, which only a newer mrw writes, is left untouched: nothing this mrw reads licenses a write from it, its
reads record nothing, every command that consults the ledger says it was written by a newer mrw and names
that one, and so does a write refused for an unread line (ADR-150). The writer name is information: a v5
ledger is read whoever wrote it. An mrw up to v1.65.0 compares that line byte for byte, so it still calls a
v5 ledger older and replaces it.
Inside a git checkout (a `.git` directory or file at or above the root) the read ledger is the checkout's, not the directory's: relative keys are stored from the checkout's top, so a read in `sub/` licenses a write from the top and the reverse, and the write lock is the checkout's too. Paths still resolve against the root (ADR-161 Decision 14).

**A shell glob and an address suffix do not mix.** `mrw read 'dir/*.go:1-3'`
takes the star literally and reports the path UNREADABLE; unquoted, zsh refuses
it first. Use `--grep` or `--files-from`.

## Write

A plan is a sequence of hunks. Every address resolves against the original file
— no offset arithmetic between hunks.

```
@@ internal/apply/apply.go 42-58 replace anchor="func Apply" lines=17
        ... new lines ...
@@ internal/apply/apply.go 12 insert-after
        "sort"
@@ README.md 3-4 delete
@@ docs/new.md - create
# a new file
```

```sh
mrw write plan.mrw
mrw write --dry-run plan.mrw
mrw write --no-check plan.mrw
mrw write --json plan.mrw
mrw write -
```

Ops are `replace`, `insert-after`, `insert-before`, `delete`, `create`,
`unlink`, `rename`. `@@ path - unlink` removes the path. `@@ old - rename`
with a one-line dest body moves it, edited first when the plan also edits it; on a filesystem that folds case it may change only the case of the name (`a.txt` → `A.txt`, ADR-129). Only `delete` may carry no body among
the line-range ops: a lost body reads like one never written, so an
empty file is `@@ new.txt 0 create body=0`; a create's address is `-` or `0`. A bare `create` with nothing under
it is refused and leaves no file behind.

`--format=apply_patch` compiles a Codex `*** Begin Patch` document to that same
native plan. `--format=search_replace` compiles Aider SEARCH/REPLACE. Both still
refuse an unread sibling and write nothing. A git patch is not an apply_patch;
`--format=git` is usage. The flag is required — there is no auto-detect.
Sequential apply_patch — one hunk, then another — is the leak, not the feature.

`mrw write --create PATH` makes one new file from standard input without a plan or a hand-counted `body=`:
the lines of the content become the file exactly as the `create` plan of the same lines would (each line ends
in a newline, so a missing final newline is added and CRLF becomes LF, which stderr names; empty input makes an empty file), through
the same lock, check and receipt. It is refused if PATH exists and takes no PLAN argument and no `--format`
(ADR-139). CLI only.

`anchor=` is required on a `replace` that addresses more than one line. Take it
from the `NNN| content` a read printed; one typed from memory can be wrong in
the same way the address is. `sha=` and `lines=` do not stand in for it: `sha=`
pins the whole file and `lines=` the count, neither which lines (ADR-164). Both
are optional and are checked on every op, insertions included.

Addresses are 1-based and inclusive; `$` is the last line, and `A,+N` is the line `A` plus the `N` lines AFTER it. A read CLAMPS a relative end at the last line; a write REFUSES one that runs past it. `/from/,/to/` means the same on both paths: the end is the first match
at or after the start. A write refuses unless the start matches exactly once, or
names the Nth match with `occurrence=N` — refused unless every match before it has
been served (ADR-118); a read serves a span for every start match that is not
already inside a span it served.

## Safety

These are gates, not a tour of the records behind them.

- **All-or-nothing.** Any hunk that fails validation writes nothing. Siblings
  report `skip`, never `ok`. A filesystem failure while committing reports what
  reached disk: `PARTIALLY APPLIED` naming what landed, or `NOTHING WRITTEN`
  when the undo put everything back.
- **Per-line licence.** Being served lines 1–5 does not license line 40.
  `--stat` and a match that printed nothing observe nothing.
  A file mrw just wrote is wholly known: a chain of edits needs no re-read.
- **One writer at a time.** Writers on one checkout take turns (ADR-075). A writer waits up to 120 s for the
  one before it, then is refused, exit 2, nothing applied, naming the holder's pid; `MRW_WRITE_LOCK_TIMEOUT`
  sets the wait in seconds, `0` for one try (ADR-110).
- **MCP ack.** A served `mrw_read` licenses nothing until you send `ack` ids.
  Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
  A checkpoint brackets consecutive lines only, and served lines holding bytes that are not valid UTF-8 get none:
  the JSON answer replaces those bytes, so edit such a file with the CLI (ADR-108).
- **Neighbour licence.** A multi-line `replace` is refused unless a prior read
  already covered a line after End. Last line of the file is exempt.
  `--echo-pad N` (MCP `echo_pad`, default 0) prints N lines after an applied
  body so a surviving closer is visible; the hunk stays `ok`. The pad is not a
  checker.
- **Check by default.** A CLI write to a non-prose path runs the project's
  check when one exists; `--no-check` opts out; a markdown-only plan does not
  spawn it. A `{files}`- or `{dirs}`-only `scoped_check` still runs on a `.rs` write when
  `packages()` cannot map. A `{packages}`-only one that places some of the written files runs over
  those packages and names the rest on an `outside the scoped check:` line (`check.outside_scope`
  in `--json`); it runs the whole check when nothing places, or a `.go` path cannot be placed, a
  directory holds no package, a module file (`go.mod`, `go.sum`, `go.work`, `go.work.sum`) was
  written, or the write removed a Go or module file. Its pass covers the packages it could place only.
  A non-prose hunk
  whose `{}` `()` `[]` nets moved prints a balance row and stays `ok` — a
  balanced insert in the wrong place is invisible to it.
  A replace whose body ends in a closer (a fence, `}`, `</div>`, `@endif`) that
  its range did not end in, and that one of the next four non-blank lines
  repeats, prints a `closer` row and stays `ok`, counted in `hints` (ADR-119);
  it is advice, never a refusal.
  `mrw stats` prints every outcome at zero — `failed_check` and, since ADR-102, `partially_applied`
  among them — and a landed-writes line.
- **Steps after the check.** `--then NAME` runs a step declared under `"steps"` in
  `.quality-harness.json`, and `--then-sh 'CMD'` an ad-hoc shell step; both repeat and run in
  command-line order after a landed write whose check passed, on `write` and `check`. The first that
  does not pass stops the rest, named `NOT RUN`; exit 3 (2 if it could not start); `--json` carries
  `then: {steps, pruned_logs}`. --then-sh runs any shell command it is given: a harness rule that allows mrw without reading its arguments allows arbitrary shell through --then-sh.
  `could_not_start` means the shell could not start (a missing command inside it is a `fail`, exit
  127); a step runs with `MRW_STEP_DEPTH` one deeper, and `--then` is refused at depth 8, so a step
  that re-runs mrw with steps cannot recurse without end; `"steps"` is read only when a step is asked for.
  A step runs as written: a step command holding {files}, {dirs} or {packages} is refused, since mrw expands them only in scoped_check. A passing step prints the last line of its output under its verdict.
  A check, like a step, runs with MRW_STEP_DEPTH one higher than mrw's own; at depth 8 mrw starts neither: --then and --then-sh, a write whose check is due, and mrw check are refused, exit 2, before anything is written or run, while a write that starts no check still lands (--no-check writes without it). A command that clears the environment, such as env -i or sudo, restarts the count below it, as setsid leaves the process group. On unix a stopped check, step or ast-grep process group hears SIGTERM first, and whatever ignores it is killed a second later; an mrw killed that way can leave its own check running (ADR-095).
  A check that ran the whole project's command for the files a write touched (or `mrw check PATH`), and took 60 s or more, adds a `hint:` line saying to declare `"scoped_check"` in `.quality-harness.json`, or why the declared one could not name those paths; the JSON carries it as `check.hint` (ADR-152).
  On Windows a check, a step or ast-grep runs in a job object that is closed with it, so its grandchildren stop too and none can leave the job (ADR-120).
- **Advisories are counted where you read.** The summary line says
  `N failed, A advisories`, zero included, and the JSON receipt carries
  `advisories`. Three advisories in your last ten writes print a `pattern:`
  line on the receipt. `--strict-balance` (MCP `strict_balance`) is opt-in and
  refuses the wrap-tail shape — a single-line replace whose line's delimiters
  do not balance and whose body does not match — as a failed hunk, exit 1,
  nothing written. Both JSON receipts carry `pattern` on every write, and
  `mrw stats` prices the flag as writes land: how many it would have refused,
  and whether those writes then broke, held or went unchecked — the numbers the
  pre-registered default question (BACKLOG) is decided on.
- **Check miss refuses.** An in-root `mrw check` miss is exit 2 and names the
  path — not a silent whole-project PASS. Under `--json` every refusal `mrw check` makes once its flags
  are parsed is one document, `{"error": …}`, with no `exit_code`; a flag the parser rejects, or an
  attached flag value with edge whitespace, is refused before that and prints nothing on stdout. When no
  check could run the receipt says `"ran": false` and `"exit_code": -1`, the value every check or step
  with no exit status carries, never 0 (ADR-100, ADR-101).
- **The process is the verdict.** A check that prints `PASS` and exits 1 is a
  failure. Never read an exit code through a pipe: `mrw write plan | head` is
  `head`'s status.
- **A path means what it says.** A trailing `/` names a directory, so a file
  spelled `a.txt/` is refused, and so is a rename to `d/`. A read-only file is
  refused for every op that would change it. On Windows a name the OS opens as
  a device (`NUL` always, `CON` and the rest where that Windows reserves them)
  is refused. The receipt names a symlink's `target`, the directories
  a plan made (`dirs_created`), what it made and could not remove
  (`left_behind`), and a removed file's former sha.
- **A delete says what it removed.** Its verdict carries `removed_first` and
  `removed_last`, and a `delete` may carry a body: the lines it expects to remove,
  refused before anything is written when the count differs (ADR-008).

After any multi-line body, read on past the range until the enclosing structure
closes. mrw models no target syntax; the damage is never inside the lines you
named, which is why the receipt cannot show it.

A plan names a file once, however it is spelled. Two spellings that reach the
same file — case-folded names, or a file and a symlink to it — are refused with
both named.

It will not write outside `--root`, even through a symlink or a Windows
junction, and not even through a directory another process swaps for a link
while the write runs: every change after validation goes through the root, held
open (ADR-106). It will not replace a symlink, and will not change your line
endings. A file a plan edits that another process replaces or rewrites after
mrw read it is refused by name rather than overwritten, when either of two
checks sees the change: the one before the first rename, which leaves nothing
written, or the one just before that file's own rename, where the commit stops
at that file (ADR-106). A change in the instant after the second check, or an
in-place rewrite that keeps both size and modification time, is not seen; and
the source of an unlink or a rename is not checked this way. Staging
failures write nothing; a later rename failure can leave a partial tree and
names the files already written. When two callers share a checkout, pin each
file with `sha=` from the read header (`==> a.go … sha 1a2b3c4d`), so a plan
written against a file the other caller has since changed is refused. The
records are in [docs/adr/](docs/adr/).

## What a failure leaves on disk

Each file lands by a rename, so no file is ever half-written. What a failed run
leaves depends on where it stopped (ADR-001, ADR-066, ADR-102, ADR-105). Anything
mrw made in the tree and could not take away is named in `left_behind` (`left
behind: <path>` on the human receipt), on success and on failure.

| Stage | On disk | Receipt | Exit | Recovery |
|-------|---------|---------|------|----------|
| validation | nothing changed | the failing hunks with reasons, the rest `skip`, NOTHING WRITTEN; a file mrw cannot open names why — permission denied, held exclusively by another process, a name the system refuses (ADR-125) | 1 | fix the plan, or the file, and send it again |
| staging | nothing changed; the temp files and directories staging made are removed, and any that could not be are in `left_behind` (a temp another process moved away with its directory is at a path mrw never made, and is not named) | the unstageable file's hunks fail with the filesystem error, the rest `skip`; on Windows a target held without delete sharing fails here, before any rename, naming that it is held (ADR-125) | 1 when the cause is the target's — held, a permission, a name refused, changed or removed since mrw read it; 2 for the environment — disk full, an I/O error, an error mrw cannot name (ADR-132) | fix the plan or the file, or the cause (space, permissions), and send again; delete what `left_behind` names |
| commit | the files renamed before the failure stay written (PARTIALLY APPLIED), or the undo put every one back (NOTHING WRITTEN) | `files[].written` says what landed; `error` names the step that stopped | 2 | read the written files; send only what did not land, never the whole plan again |
| undo | an unlinked file whose path a rename still holds stays in a `.mrw-aside-*` recovery file | `UNDO INCOMPLETE` names the aside and the path it belongs at, and `left_behind` names the aside | 2 | settle what now holds that path first, then move the aside there; never move it over a file |
| ledger | the write landed; mrw could not record it | `applied: true` with `error` naming the ledger (CLI and `mrw_write`) | 2 | read the files again before the next edit |
| check | the write landed and is unverified | the check's verdict and `full output:`; a failing check that printed shows `check last:` above it | 3; 2 when no check could start | fix the code or the check; the tree is already changed |
| process death | a run killed mid-commit can leave some files renamed and others not, `.mrw-*` temps beside them, and a `.mrw-aside-*` that is either an empty placeholder (its file is still in place) or an unlinked file's only copy; mrw's state files are whole, since each is replaced by rename | none: a killed run writes neither its undo nor a receipt | none | `git status` shows what landed; read each `.mrw-aside-*` before touching it — move a copy back only to a path that is missing, never over a file; delete the other `.mrw-*` temps; read before the next edit |
| power loss | staged tree files are not synced (+1335 % measured, ADR-105), so a write that returned can be lost; the ledger and the acknowledgement store are synced, but a name's directory is not, so either can come back as its previous version | the one you already had | none | check the tree; a ledger that came back older only refuses edits until you read the files again |

## MCP

Two tools: `mrw_read` (`specs`) and `mrw_write` (`plan`). Same engine, same
ledger.

### Use it from an MCP host

```json
{
  "mcpServers": {
    "mrw": {
      "command": "mrw",
      "args": ["mcp"]
    }
  }
}
```

A write's check does not hold the server: requests sent while it runs are answered, and a call that sends
`_meta.progressToken` hears `notifications/progress` until its answer (ADR-121).

Use an absolute path for `command` if `mrw` is not on the host's `PATH`. Launch
with an explicit root when the host does not set `CLAUDE_PROJECT_DIR`:

```sh
mrw --root DIR mcp
```

```json
{
  "mcpServers": {
    "mrw": {
      "command": "/absolute/path/to/mrw",
      "args": ["--root", "/absolute/path/to/repo", "mcp"]
    }
  }
}
```

On a host with a shell, prefer the CLI. Where you register the server decides
where its tools appear. At user scope (`claude mcp add -s user mrw -- mrw mcp`)
both tools load in every project on the machine; at local scope, run from one
checkout (`claude mcp add -s local mrw -- mrw mcp`), they load only there.
Choose user scope when every project should reach mrw over MCP, and local scope
when only one should.

`format`, `echo_pad`, and `ack` sit on the existing write/read — not a third
tool. `format` is `plan` (default), `apply_patch`, or `search_replace`.
`echo_pad` is the same opt-in pad as `--echo-pad`. `ack` is how a served read
becomes a licence. `mrw_write` runs the project's check after a write that touches
code, by the rule `mrw write` uses (ADR-113), and returns the verdict as `check`
(and `drift`); `check: false` turns it off. A check that ran and failed leaves the
write applied and the call not an error. The write lands before the check runs, so
a lost answer means read the files before re-sending the plan. `then` names steps declared in
`.quality-harness.json` to run after a passing check (ADR-115), their verdicts in the receipt's
`then`; ad-hoc shell (`--then-sh`) stays on the CLI. Its receipt's hunks carry
the same `balance` field the CLI prints.
An argument a tool does not declare is refused, naming it and the arguments the tool takes; nothing is done.

Without `--root`, the server uses `CLAUDE_PROJECT_DIR` when the host sets it,
else its working directory. A silent fallback to `/` or `$HOME` is refused
(exit 2). An explicit `--root` is always honoured, including `--root /`.

An answer is bounded at 200,000 characters of encoded result. Set it with
`mrw mcp --max-result-chars N` or `MRW_MAX_RESULT_CHARS`. The flag beats the
variable; `0` means zero. A ceiling too small to report a write refuses the
write before anything is applied.
What mrw reads is bounded as well (ADR-104): a request line up to 64 MiB, a file up to 1 GiB,
an ast-grep answer up to 256 MiB; past each the answer says so by size.

`grep` / `exclude` map onto `--grep` / `--exclude`. `ast_grep` maps onto
`--ast-grep`. A grep too large to serve returns an index — one spec per
matching file, no content — which licenses nothing — and names every path the
walk could not use. Like a grep's, an ast_grep index pages with `after`: send the
same finder again with `after` set to `next_index`, until it is empty.

A read of ONE spec — a whole file, or `path:N-` — too large for the ceiling comes
back as a first page with `next_read`: send that back as `specs` to continue, and
repeat until it is absent. Any other read too large — several specs, or a closed
range, which already says what it wants — is refused, and the refusal says what
to do instead: ask for a narrower range, or name fewer files.
`mrw_read`'s result is the served text, with no `structuredContent` (ADR-023);
the write result's `structuredContent` carries the receipt.

The server speaks MCP `2025-11-25` and `2025-06-18` through `initialize`, and
answers the version the host asked for. It also speaks `2026-07-28`: a request
carrying `_meta["io.modelcontextprotocol/protocolVersion"]` is served per
request, with `server/discover`, `resultType`, and caching hints on
`tools/list`. A version it does not speak gets `-32022` naming the ones it does.
A mistake inside a tool's arguments comes back as a tool result with `isError`,
which the model reads. A call that is not a valid request stays a JSON-RPC error.

### Git Bash on Windows mangles a regex address

`mrw read 'internal/f.go:/^func main/'` **fails in Git Bash**, and the error
names a line number you never typed. MSYS2 rewrites the argument *before* mrw
is started when its file part holds a `/`, and a one-letter pattern such as
`/x/` becomes a drive. Quoting does not prevent this.

| environment | regex addresses |
|---|---|
| Git Bash / MSYS2 | **fail** |
| Git Bash with `MSYS_NO_PATHCONV=1` | work |
| Git Bash with `MSYS2_ARG_CONV_EXCL='*'` | work |
| PowerShell | work |
| WSL | work |

Line-number, range and `$` addresses are unaffected. A `--grep` pattern that
starts with `/` is rewritten too, into a false "no match".

## Exit status

| code | meaning |
|---|---|
| 0 | everything asked for succeeded |
| 1 | a hunk failed, or the answer is incomplete; **nothing written**. For `mrw version --check-latest`: a newer release exists |
| 2 | usage, parse or I/O error — including a check miss |
| 3 | the write applied and the check failed, timed out or was interrupted (before it started, too); the tree is changed and unverified. Not a rollback |

Exit 1 on `read` means incomplete: `UNREADABLE`, `REFUSED`, `no match`, or
`WITHHELD`. The output always names which, and the last stderr line begins with the kind
(ADR-146): `mrw: no match: …` when a search found no file and nothing it was asked for was
refused, `mrw: incomplete: …` when a requested range or path was not served, a search that
matched no file beside a refused path included.

## opencode

mrw supports [opencode](https://opencode.ai) through a plugin in this repository,
`cmd/opencode/mrw-plugin/` (ADR-089). It gives an opencode session eight tools:

| tool | runs | arguments |
|---|---|---|
| `mrw_read` | `mrw mcp`'s `mrw_read` | `specs`, `grep`, `astGrep`, `exclude`, `after`, `ack`, `maxLines`, `stat`, `filesFrom`, `root` |
| `mrw_write` | `mrw mcp`'s `mrw_write` | `plan`, `ack`, `dryRun`, `check`, `then`, `format`, `echoPad`, `strictBalance` |
| `mrw_check` | `mrw check` | `paths` |
| `mrw_stats` | `mrw stats` | `json` |
| `mrw_seen` | `mrw seen` | `prune`, `dryRun` |
| `mrw_iter` | `mrw iter` | `args`: a verb and its specs, e.g. `["add", "a.go:10-20"]` |
| `mrw_version` | `mrw version` | none |
| `mrw_instructions` | `mrw instructions` | none |

**Read and write go through the MCP engine, because opencode cuts long output.** It
truncates a tool's result at 2,000 lines or 50 KiB, and a CLI read has licensed
every line it served by then, so the hidden tail would be writable unseen. Over
`mrw mcp`, a read is a page under that limit, bracketed in `-- ck` checkpoints,
and licenses nothing until acknowledged. Send an id in `ack` only when both of its
markers and all its lines arrived; a cut page loses its tail's close markers, and
those runs stay unwritable (see [MCP](#mcp)). `mrw_write` runs the check after a code
write, as `mrw write` does; `check: false` turns it off, and `mrw_check` runs it on
demand. The other six tools run the CLI and answer `exit: N` first, the CLI's
exit code (see [Exit status](#exit-status)). An answer the server marks as an
error starts `error:` — a write whose receipt says it applied still landed.

### Setup

Build the plugin once:

```sh
cd cmd/opencode/mrw-plugin
npm ci
npm run build
```

Then list it in the workspace's `opencode.json` (this repository ships one), or in
your global opencode config with an absolute path, and restart opencode:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "plugin": ["./cmd/opencode/mrw-plugin/dist/index.js"]
}
```

The plugin runs `bin/mrw` (`bin\mrw.exe` on Windows) from the worktree when it is
there, and otherwise `mrw` on `PATH`. In a session with no git worktree it works in
the session's directory.

### Or over MCP alone

opencode is also an MCP host, so `mrw mcp` works there without the plugin: list it
under `mcp` in `opencode.json` as a local server running `mrw mcp` (see [MCP](#mcp)).
opencode prefixes an MCP tool with its server's name, so a server named `mrw`
answers as `mrw_mrw_read` and `mrw_mrw_write` beside the plugin's tools. Both work
together, over the same engine and ledger; the plugin adds `check`, `stats`, `seen`
and `iter`.

`cmd/opencode/mrw-plugin/test/smoke.test.mjs` drives the built plugin's tools against
the built binary, a page cut the way opencode cuts one included, and CI runs it on
Linux. opencode itself is not run in CI.
## Other commands

`mrw check`, `mrw iter`, `mrw seen`, `mrw seen --prune`, and `mrw stats` exist
only on the CLI, and so through the opencode plugin, which runs it. `mrw check` is not read-only: it runs whatever the project
declared. Nothing calls `--prune` for you. Details: [AGENTS.md](AGENTS.md).

Round trips are 2 for any N. Re-run `./scripts/measure.sh` rather than quoting
a table — the ratios track this repository's own files. The contract is
`./scripts/contract.sh`; it prints its own total.
