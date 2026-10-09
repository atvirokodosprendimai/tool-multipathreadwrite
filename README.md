# mrw — multi-path read and write

Read many ranges across many files, and apply many edits across them, in one
invocation — and get a verdict for every edit. A plan that fails validation
writes nothing, because a write that changed nothing is invisible.

The numbers — two calls for any N, shapes A–D — live in [docs/measure.md](docs/measure.md). The six-guarantee comparison lives in [docs/comparison.md](docs/comparison.md). Model × score readings live in [docs/model-benches.md](docs/model-benches.md).

**Status: stable at v1.54.0 (2026-10-10), the tag cut from `bba9e21`.** Break campaign for it: [docs/break/campaign-v1.54.0.txt](docs/break/campaign-v1.54.0.txt), 59 probes, exit codes identical to v1.53.0.

Decisions: [docs/adr/](docs/adr/). How a change reaches `main`: [CONTRIBUTING.md](CONTRIBUTING.md). Driving it from a checkout: [AGENTS.md](AGENTS.md).
Caller practices: [BESTPRACTICES.md](BESTPRACTICES.md). Updating the binary: [UPDATE.md](UPDATE.md).
Hosts: any shell (the CLI), any MCP host (`mrw mcp`, [below](#mcp)), and [opencode](#opencode) through its plugin.

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
arguments are usage (exit 2).

`mrw instructions` prints the contract from the binary: use mrw always and plan
the activity, the two rules that produce most refusals, and the traps that make
a red run look green. Exit 0. No flags. `mrw --instructions` is refused, exit 2,
with a message naming `mrw instructions`; so is any other undefined flag that is
exactly the name of a subcommand of the command it was given to.

### From source

Go 1.26.9 or newer (`go.mod`). One dependency, no cgo:

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
`/start/,/end/`. `$` is the last line. Output ranges print as `@@ 3-6`, which is
the address a write plan takes.

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
| `-C N` | context around a single-pattern match |
| `--max-lines N` | cap per spec, where zero means zero: `0` serves nothing. Omit the flag for no cap |
| `--grep PATTERN` | serve every regexp match under the given paths (measured 2026-09-03: the walk took 0.76× the time of a `grep -rl … \| mrw read --files-from -` pipeline over this repository, ADR-007) |
| `--ast-grep PATTERN` | serve every `ast-grep` hit (binary on PATH; missing is exit 2; a hang is, on unix, sent SIGTERM at 2 s and killed by 3 s if it ignores it; on Windows it is killed at 2 s). A hit in a file whose lines end in `\r` alone is reported, not served — read that file directly |
| `--exclude GLOB` | skip matching paths (needs `--grep` or `--ast-grep`) |
| `--files-from FILE\|-` | one spec per line |

A pattern that matches no file is reported by name and exits 1.

A read whose standard output is the null device (`>/dev/null`, cmd.exe's `>NUL`) is still
served and exits as usual, but records nothing in the ledger and says so on
stderr: a write to those lines is then refused as unread, as after `--stat`.
mrw cannot see past a pipe or a file, so those record as before, and so does PowerShell's
`> $null` or `| Out-Null`, which hand mrw a pipe, as does PowerShell 7's `> NUL`. In Windows PowerShell 5.1 `> NUL` fails
before mrw starts (Out-File cannot open a device), so nothing is read (ADR-133).

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
empty file is `@@ new.txt 0 create body=0`. A bare `create` with nothing under
it is refused and leaves no file behind.

`--format=apply_patch` compiles a Codex `*** Begin Patch` document to that same
native plan. `--format=search_replace` compiles Aider SEARCH/REPLACE. Both still
refuse an unread sibling and write nothing. A git patch is not an apply_patch;
`--format=git` is usage. The flag is required — there is no auto-detect.
Sequential apply_patch — one hunk, then another — is the leak, not the feature.

`anchor=` is required on a `replace` that addresses more than one line. Take it
from the `NNN| content` a read printed; one typed from memory can be wrong in
the same way the address is. `sha=` and `lines=` are optional and are checked
on every op, insertions included.

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
  `packages()` cannot map; `{packages}`-only still falls back. A non-prose hunk
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
| 1 | a hunk failed, or the answer is incomplete; **nothing written** |
| 2 | usage, parse or I/O error — including a check miss |
| 3 | the write applied and the check failed, timed out or was interrupted (before it started, too); the tree is changed and unverified. Not a rollback |

Exit 1 on `read` means incomplete: `UNREADABLE`, `REFUSED`, `no match`, or
`WITHHELD`. The output always names which.

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
