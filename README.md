# mrw — multi-path read and write

Read many parts of many files, and make many edits across them, in one command
instead of a dozen — and be told, for every single edit, whether it landed.

It is an ordinary command-line tool. It was built for AI coding agents, which
are the ones doing hundreds of small edits a day, but nothing about it requires
one.

Install it with one command — see [Install](#install) for the details:

```sh
curl -fsSL -o mrw "https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest/download/mrw-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" && chmod +x mrw
```

## The problem it solves

Say you need to change four things, in four different files.

The usual way is eight steps: open each file, then edit each file. The tedium is
not the real problem. The real problem is this:

> **An edit that changes nothing usually still reports success.**

If your third replacement matched no text — the line moved, someone renamed the
function, you had a typo — most tools still say "done". You find out later, when
something breaks, that one of your four changes never happened.

It is worth seeing why that asymmetry exists. A *read* that finds nothing is
obvious: you get an empty result and you know immediately. A *write* that
changes nothing looks exactly like a write that worked. That is the bug mrw is
built to make impossible.

## How you use it

Two commands: look, then change.

You have to look before you change. That is not a style suggestion — mrw
enforces it, and will refuse to edit a file it has not shown you. The reason is
in the list further down: it is how it can tell "the file is as you last saw it"
from "someone else changed it while you were working".

**Look** — several places in several files, one call:

```sh
mrw read config.go:3 'server.go:/func Start/'
```

```
==> config.go  4L  51B  sha f5cad94e
@@ 3-3
    3| const timeout = 30
==> server.go  5L  49B  sha 075a39fa
@@ 3-3
    3| func Start() error {
```

You asked for one line by number and one line by a search pattern, in two files,
and got back exactly those lines — not the whole files.

**Change** — write a short plan listing every edit, then apply it in one call:

```
@@ config.go 3 replace
const timeout = 60
@@ server.go 99 replace
	panic("x")
```

```sh
mrw write plan.mrw
```

```
skip config.go 3 replace
FAIL server.go 99 replace (plan line 3): range 99 is out of range (file has 5 lines)
2 hunk(s), 2 file(s), 1 failed — NOTHING WRITTEN
```

The second edit was wrong — line 99 does not exist in a 5-line file. So:

- **Every edit gets its own line.** Nothing is summarised into one "success".
- **One bad edit means nothing is written at all.** `config.go` was not touched.
  You never end up with a half-changed set of files, which is worse than no
  change, because it looks finished.
- **The exit status says so too** (`1` here), so a script notices without
  reading the text.

Fix the plan, run it again, and you get a verdict per edit and a summary of what
changed on disk.

## What you get

- **One call instead of many.** Two round trips, whatever N is.
- **A verdict for every edit** — `ok`, `FAIL` with a reason, or `skip` because a
  sibling failed. Never a bare "success".
- **All-or-nothing.** Any failure and the files are left exactly as they were.
- **It refuses to edit a file it has not shown you.** If the file changed behind
  its back, it stops and says so instead of overwriting your colleague's work.
- **It can run your tests for you.** `mrw write --check plan.mrw` applies the
  edits and then runs your project's tests for the code you just changed, in the
  same call, and reports whether they passed. (`mrw check --full` runs
  everything.)

## What it is not

It is not a replacement for ordinary editing tools. If you need one change in
one file, use whatever you already use — see the honest measurement below, where
mrw *loses* on that shape. It pays off when there are many edits, many files, or
both.


## Where it fits — the QAM stack

mrw is one of three tools in the **[QAM stack](https://atvirokodosprendimai.github.io/qamstack/)**:

> Three tools that make Claude's work checkable: gates that exit non-zero, memory
> that outlives the session, edits that come back with a receipt.

| | | |
|---|---|---|
| **Quality Harness** | *Gates, not vibes* | a Claude Code plugin whose gates report through exit codes rather than assertions |
| **AI Agent Memory** | *The reasoning survives* | an MCP server letting agents read and write shared memory across sessions, decisions and rejected alternatives included |
| **mrw** | *Edits with a receipt* | this: batched edits applied atomically, with a verdict per hunk |

**mrw stands alone and needs neither of them.** It is an ordinary binary; nothing
here depends on a plugin or a server, and `AGENTS.md` carries everything an agent
needs to drive it from a plain checkout.

**With AI Agent Memory it gets a memory.** The same guidance is mirrored as a
centralised `mrw` skill, so a session working in a *different* repository — where
this README is not visible, but the globally installed binary is — loads it with
`am_load_skill("mrw")`. More usefully, corrections accumulate: two facts in that
guidance were learned the hard way in one session and now reach every project
rather than being rediscovered per repo.

The three overlap on one conviction, which is why they are a stack rather than a
bundle: **a report that cannot fail is not a report.** A gate that cannot exit
non-zero, a decision nobody wrote down, and an edit that matched nothing but said
"done" are the same bug wearing three hats.


## Does it actually save anything?

**The unit is agent turns, not seconds.** mrw does not make anything faster to
execute, and no figure here is a wall-clock figure. What it removes is *steps* —
the read-edit-read-edit round trips an agent spends to change N places, each one
a full model turn with its own latency, its own context, and its own chance to
lose track. Two calls, for any N. That is the whole product.

Measured on this repository, by a script you can re-run — a number nobody can
reproduce is a claim, not a measurement:

```sh
./scripts/measure.sh
```

> **Windows:** the two reproduction scripts are `bash`, so run them under **WSL**
> or **Git Bash** — they are POSIX shell, not PowerShell. The binary itself is
> native; only these scripts need a shell. See [Prerequisites](#from-source).

**This is the only benchmark in the repository, and it measures round trips and
input bytes — not time.** There are no `go test -bench` benchmarks, on purpose: a
CPU figure for a tool that spends its life blocked on a file read would measure
nothing anyone is paying, and it is not the axis the tool competes on.

**Round trips are the claim that survives every reading: 2 calls, for any N.**
Bytes depend entirely on what you compare against, so the table gives both
baselines rather than the flattering one — including the two shapes where mrw
sends MORE bytes than the thing it replaces.

| shape | | baseline | mrw | |
|---|---|---|---|---|
| **A.** 4 sites, 4 large files | bytes vs reading those files **whole** | 104,697 | 2,951 | **35.4× less** |
| | bytes vs a **windowed** `offset`/`limit` read | 2,289 | 2,951 | **1.2× MORE** |
| | calls, whole-file (reads + edits) | 8 | 2 | 4.0× fewer |
| | calls, windowed (search + reads + edits) | 9 | 2 | **4.5× fewer** |
| **B.** 2 sites, 2 mid-sized files | bytes vs whole | 20,630 | 1,329 | 15.5× less |
| | bytes vs windowed | 866 | 1,329 | 1.5× MORE |
| | calls | 4 / 5 | 2 | 2.0–2.5× fewer |
| **C.** 1 site, whole small file | bytes (window *is* the whole file) | 12,881 | 15,765 | **1.2× MORE** |
| | calls | 2 / 3 | 2 | 1.0–1.5× fewer |
| **D.** 1 site in **every** Go file — 27 sites, 27 files | calls (reads + edits) | 54 | 2 | **27.0× fewer** |
| | bytes vs whole | 324,256 | 2,421 | 133.9× less |
| | bytes vs windowed | 410 | 2,421 | **5.9× MORE** |

Measured at `87b43d4`; the script builds the binary it stamps. Shape A's
whole-file baseline moves whenever the four files it reads do — it went from
104,486 to 104,697 bytes between two commits a day apart, and the ratio did not
budge. That is the drift this note exists for, and why the stamp is here.

**Shape D is the one to read.** It is the change every codebase gets eventually —
a renamed symbol, an added build tag, a changed import — one site in each of 27
files, and the file list comes from `git ls-files` so it grows with the
repository instead of measuring a subset somebody typed once.

The arithmetic is the product, and it does not depend on this repository:

| sites across files | Read + Edit | mrw | |
|---|---|---|---|
| 4 in 4 | 8 calls | 2 | 4× fewer |
| 13 in 1 | 14 calls | 2 | 7× fewer |
| 27 in 27 | 54 calls | 2 | **27× fewer** |
| 36 in 36 | 72 calls | 2 | **36× fewer** |
| N in M | M + N | **2** | — |

Each of those calls is a **full model turn**: a request, a response, a result
block read back, and one more opportunity to lose the thread between site 19 and
site 20. That is the cost mrw removes, and it is why the floor is 2 rather than
"fewer".

**Shape D also sends 5.9× MORE bytes than a windowed reader, and that is fine.**
Each site is a single line, so mrw is paying a per-file header and a per-line
number on the smallest possible payload — the worst byte case there is. It is in
the table at its worst because the calls column is the claim, and a table that
hid the row where the other axis loses would not be worth reading.

**Read the two byte rows together or neither.** `Read` takes `offset`/`limit`, so
the windowed reader is the documented interface, not a strawman — and against it
mrw costs *more* bytes, because it adds a header and a line number per line. The
35.4× is real for the case an agent is usually in: it does not yet know where to
look, so it reads whole files. Once it knows, the byte advantage is gone and the
round trips are what is left.

That is also why the calls row splits. Reading whole files needs no search — the
file reveals the site. Reading windows presupposes knowing where the window is,
and finding it costs a call. mrw needs neither: these specs are regexes, so the
finding happens inside the read.

**Re-run it rather than quoting this table.** The figures it replaced claimed
22.0× for shape A and were understated by a third within a day — these ratios
track how large this repository's own files are. Shape C is the only byte figure
that is drift-proof, because it reads a whole file either way, and it is the
shape where mrw LOSES, which is why it is here.

**Shape C is in the table on purpose.** When you need a whole file and there is
one site, mrw prints *more* than the file holds — it adds a header and a line
number per line — and saves no round trips. Use Read + Edit there. The saving
scales with how much of each file you *don't* need, and with N.

The method, and its biases: the Read+Edit column counts each file's **raw**
bytes, which understates it, because the real Read tool numbers every line. The
mrw column counts its **actual** output, headers and line numbers included.
Output tokens are not measured — the plan you emit for mrw and the
old_string/new_string pairs Edit needs are the same order of magnitude — so
this is an input-side and round-trip result, not a total-cost one.

## Why it exists — the design gap

This section is the technical version of the problem described at the top.
Skip it if the top was enough.

mrw sits in a gap between two primitives an agent already has:

| | batches | fails loudly |
|---|---|---|
| `Edit` | no — one replacement per call | yes, on a bad anchor |
| `Write` | yes — whole file | no, it cannot say which change did not land |
| `mrw` | yes — N hunks across M files | yes, per hunk, and writes nothing on any failure |

The failure it is built around, stated exactly: **a read that returns nothing is
visible; a write that changes nothing is not.** Batching four replacements into
one script and getting "success" while one of them silently matched nothing is
the bug this refuses to reproduce.

## Does the contract hold under abuse?

Every row below is asserted by a script, against the real binary in a throwaway
repo, by making each promise go wrong on purpose:

```sh
./scripts/contract.sh      # exit 0 only if every assertion holds (216 today)
```

The count moves as rows are added, so the script prints its own total rather
than being trusted to match a number written here. Same shell requirement as
above: **WSL or Git Bash** on Windows.

| test | result |
|---|---|
| 3 hunks, 1 bad anchor | offender FAILs naming the line, siblings `skip` (never `ok`), nothing written, exit 1 |
| 3 hunks / 2 files valid, `--check` | applied, scoped `go test .` PASS, exit 0 |
| good write + deliberately red test | write **kept**, no revert, exit 3, failing tail in `--json` |
| `@1`, `@3`, `@1:1-2` pointers | resolve; `@9` errors with the entry count, exit 2 — never an empty result |
| check whose output says `PASS` but exits 1 | reported FAIL, exit 3 — the process is believed, never the text |

The last row is the one worth staring at. The check printed `PASS` and returned
1; mrw reported failure, because output goes to a file and the verdict comes
from the process's real status. A tail in the pipeline would have believed the
word.

## Install

Download a released binary — raw, so there is nothing to unpack:

```sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')   # linux | darwin
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -fsSL -o mrw \
  "https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest/download/mrw-${OS}-${ARCH}"
chmod +x mrw && ./mrw --version
```

Windows: `mrw-windows-amd64.exe`. Every release also carries conventional
archives (`mrw_<os>_<arch>.tar.gz` / `.zip`) and a `SHA256SUMS.txt` covering
every asset.

### From source

Prerequisites: **Go 1.26.6 or newer** — the version in `go.mod` — and nothing
else. mrw has one dependency and no cgo, so the build is one command anywhere
Go runs:

```sh
go build -o bin/mrw ./cmd/mrw          # Linux, macOS
go build -o bin/mrw.exe ./cmd/mrw      # Windows
```

Running the tests needs only Go (`go test ./...`). Running the two reproduction
scripts additionally needs **bash**, **git** and **bc** on `PATH`. `bc` is *not*
present on Alpine or most slim container images — `apk add bc` — and on Windows
they need WSL or Git Bash.


### Use it from an MCP host

`mrw mcp` speaks the Model Context Protocol on stdio, so an agent reaches the
same engine without shell access. Add one block to your host's config:

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

Use an absolute path for `command` if `mrw` is not on the host's `PATH` — a host
does not always inherit your shell's. Pass `--root` before `mcp` to bind the
server to one checkout: `"args": ["--root", "/path/to/repo", "mcp"]`.

Two tools are exposed. `mrw_read` takes `specs` — the same range syntax the CLI
takes — and `mrw_write` takes `plan`, the same plan text. Nothing listens on a
port; the server speaks over the pipe the host already opened, and it writes
only MCP messages to stdout.

**What the server does not change.** It is the same engine: the same read-before-write
ledger, shared with the CLI so a file read over MCP can be edited from a shell
and the reverse; the same plan format; the same per-hunk verdict, carried in the
tool result's `structuredContent` and byte-identical to what `mrw write --json`
prints; and the same meanings for every exit status when you go back to the
shell. There is nothing to choose between the two paths and no behaviour to
learn twice — the server is a second caller of one engine, not a second product.
The single difference is the concurrency note above.
### ⚠ Git Bash on Windows mangles a regex address

`mrw read 'f.go:/^func main/'` **fails in Git Bash**, and the error names a line
number you never typed:

```
mrw: "cmd\mrw\main.go;C:\...\Git\^func main\": bad line number "\Users\..."
```

MSYS2 rewrites the argument *before* mrw is started: it reads `a:b` as a POSIX
path list so the `:` becomes `;`, and `/^func main/` looks root-relative so it is
expanded against the Git installation prefix. **Quoting does not prevent this** —
it happens in the process-spawn layer, after the shell has finished.

| environment | regex addresses |
|---|---|
| Git Bash / MSYS2 | **fail** |
| Git Bash with `MSYS2_ARG_CONV_EXCL='*'` | work |
| PowerShell | work |
| WSL | work (a Linux environment; no MSYS layer) |

Line-number, range and `$` addresses are unaffected — they carry no leading `/`.
mrw recognises the wreckage and says so, but it cannot undo it: the bytes it
receives are already the mangled ones.

## Releasing

`.github/workflows/ci.yml` runs gofmt, `go vet`, `go test ./...` and
`go test -race ./...` on every push and PR — on **Linux and Windows**, because
this project ships a windows/amd64 binary and cross-compiling one proves only
that it links. `scripts/contract.sh` runs on Linux, being POSIX shell. Pushing a
**strict** `vX.Y.Z` tag additionally cross-compiles five targets and publishes
them:

```sh
git tag v0.1.0 && git push origin v0.1.0
```

The tag filter on `push` is a glob and cannot express "digits only", so a
`check` job re-matches the tag with a regex and everything downstream gates on
it — `v1.2.3-rc1` builds nothing. Binaries publish only after the tests and the
race detector are green.

The build stamps `-X main.version=<tag>`; `cmd/mrw/version_test.go` keeps that
symbol reachable, because the linker discards a `-X` for a symbol that no
longer exists and says nothing.

## Read

```sh
mrw read internal/apply/apply.go:1-40
mrw read a.go:1-8,100-130 b.go:/func Handle/,/^}/ c.go --stat
```

A range is `3-6`, `5`, `3-` (to EOF), `-20` (from the start), `/pattern/` (each
matching line, with `-C N` context) or `/start/,/end/`. Overlapping ranges are
merged, so no line is printed — or paid for — twice.

Output ranges print as `@@ 3-6`, which is exactly the address a write plan takes.

| flag | effect |
|---|---|
| `--stat` | length, bytes and sha only — no content |
| `-C N` | context lines around a single-pattern match |
| `--max-lines N` | cap per SPEC; whatever is withheld is always reported. Two hand-written specs naming one file therefore get two budgets — `--max-lines 2 f.txt f.txt` prints four lines. `--grep` deduplicates, so it is per file for everything the walk produces. |
| `-N` | drop line numbers |
| `--grep PATTERN` | serve every matching range in the files under the given paths; a directory is walked, and with no paths the walk starts at `--root` |
| `--exclude GLOB` | skip paths matching GLOB (repeatable); needs `--grep` |
| `--files-from FILE\|-` | read one spec per line from a file, or from stdin |

### mrw finds the files it serves

Without `--grep`, `mrw read` has to be **told** which files to serve, so every
use of it is a searcher followed by a read — two calls, and a spec list composed
in a shell. That composition is where it broke on first contact: a
newline-separated list from `grep -rl` collapses into a single argument under
ordinary word splitting, and mrw then faithfully reports a file named
`"a.go\nb.go\nc.go"` as unreadable.

```sh
mrw read --grep 'func Handle' -C 3 --exclude vendor --exclude '*_test.go' internal/
```

One call. The walk serves the same `@@` ranges a hand-written
`path:/func Handle/` spec would, for every file that matches.

**`--files-from` is the same idea for a searcher you already trust**, and it
ships whether or not you use `--grep`:

```sh
rg -l 'func Handle' | sed 's|$|:/func Handle/|' | mrw read -C 3 --files-from -
```

Blank lines are skipped and a leading `#` is a comment, so a generated list can
carry its own provenance.

**What it does not do:** it does not read `.gitignore` — mrw has no git
dependency, and `--exclude` is the control — and it does not sniff for binary
files. A regular file is a candidate, so a build artifact in the tree is served
like anything else; exclude it by name. `.git/` is always skipped.

#### Precedence

| combination | behaviour |
|---|---|
| `--grep P`, no paths | walk `--root` |
| `--grep P` with paths | walk those paths |
| `--grep P` + a positional spec carrying its own `:RANGE` | usage error — two answers to one question |
| `--grep P` + `--files-from` | usage error — two sources of specs |
| `--files-from` + positional paths | usage error — same reason |
| `--exclude` without `--grep` | usage error — nothing to exclude from |
| `--grep P`, no paths, non-empty working set | walks `--root`, **not** the working set — `iter` holds its own specs; write `mrw read --grep P @1 @2` for both |
| `--grep P` + `--stat` | allowed: the matching files' headers, no content, observing nothing |
| no `--grep`, no paths | unchanged — the working set |

A pattern that matches no file is **reported by name** and exits 1. A path the
walk cannot serve is printed with its reason and counts as a problem, while
every valid sibling is still served.

#### What `--exclude` matches

Each glob is matched with `path.Match` against **both** the cleaned
root-relative path **and** the basename, and a match on either excludes.
Matching a directory prunes everything under it. An explicitly named path is
never pruned.

The basename half is not a convenience — it is the difference between the flag
working and doing nothing. `path.Match`'s `*` does not cross `/`, and `**` is
not a token, it is two `*`, neither of which crosses either:

    glob "*.go"      vs "internal/read/walk.go"      -> false
    glob "*_test.go" vs "internal/read/read_test.go" -> false
    glob "**/*.go"   vs "internal/read/walk.go"      -> false

None of those is a bad pattern, so nothing would warn you: `--exclude
'*_test.go'` matched against the full path alone gives you every test file.
Matching the basename makes the first two work as written. The third has no
working spelling — write `*.go` for every Go file, or a path like
`internal/read/testdata`. A glob `path.Match` rejects is a usage error, not a
pattern that silently matches nothing. Matching is case-sensitive everywhere.

#### Is the walk worth it?

The concern was that `--grep` reads each candidate twice — once to match, once
to serve — where `grep -rl` reads it once. This was measured 2026-09-03 on this
repository, on an Apple M5 (darwin/arm64), pattern `EvalSymlinks`, with both
sides verified to select the **same 12 files** before any ratio was taken:

| | best of 5 |
|---|---|
| `grep -rl --exclude-dir=.git … \| mrw read -C 3 --files-from -` | 38 ms |
| `mrw read --grep EvalSymlinks -C 3 .` | **29 ms** |

**0.76×** — the walk is *faster*, because it spawns no second process and moves
nothing through a pipe. The decision that introduced it set 2× as the point at
which `--grep` would be withdrawn.

## Write

A plan is a sequence of hunks. It is deliberately not JSON: an output token
costs ~5× an input one, and JSON would escape every newline and quote in every
code body — the one part of the document that is already large.

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
mrw write plan.mrw            # apply
mrw write --dry-run plan.mrw  # validate only
mrw write --check plan.mrw    # apply, then run the tests for what changed
mrw write --json plan.mrw     # machine-readable receipt
mrw write -                   # read the plan from stdin
```

**Generate the plan when there are many sites.** A plan is line-oriented text,
so anything that prints lines can build one — and at real scale that is how it
is done. Observed in use: thirteen hunks in a single `mrw write`, from a plan
built by a one-liner.

```sh
# thirteen replacements, one read, one write — 2 round trips instead of 14
mrw read 'app.css:40,80,102-103,126,244,446,500,600,700,800,900,978'
for n in 40 80 102 126 244 446 500 600 700 800 900 978; do
  printf '@@ app.css %d replace\nCHANGED-%d\n' "$n" "$n"
done | mrw write --check -
```

The read comes first because of the guard, not as a courtesy: a hunk addressing
a line mrw has not served you is refused, and refused as a whole — so a plan
whose sites you only partly read applies none of itself, and says which hunks
were short. That is the intended loop, not an obstacle to route around.

**Every address resolves against the original file.** Read once, note several
ranges, edit them all — no offset arithmetic between hunks.

Ops are `replace`, `insert-after`, `insert-before`, `delete`, `create`.
Addresses are 1-based and inclusive; `$` is the last line, `0` is before the
first, `N-` runs to EOF. The same address means the same thing to `read` and to
`write` — `mrw read f.go:$` prints one line and `@@ f.go $ replace` changes one.
`read` used to disagree, because it shared one sentinel between `$` and an
omitted end and so served the whole file for `f.go:$`.

Three optional guards make a batch safe to trust, and all three are cheap to
write, which is the point:

| guard | asserts |
|---|---|
| `sha=<8+ hex>` | the whole file is what you read |
| `lines=N` | the addressed range covers exactly N lines — an insertion's address is a position, so it covers `1` at a real line and `0` at the two boundary positions below |
| `anchor=<substring>` | it appears in the addressed range's first line |

All three are checked on **every** op, insertions included. An insertion at a
drifted address puts the right text in the wrong place exactly as a replacement
does, so a guard that is parsed and then discarded would be worse than no guard
at all — the caller believes the edit is pinned.

The two boundary addresses have no line to check, and say so rather than
passing: `insert-after 0` (before the first line) and `insert-before` one past
the last line **refuse** an `anchor=`, because there is nothing there for it to
appear in. Prepend and append are the two edits an anchor cannot guard.

`anchor=` is matched as a substring, and a backslash escapes a quote or another
backslash — an anchor names a line of source, and source contains quotes:

    @@ page.templ 12 replace anchor="class=\"muted\""   ← searches for class="muted"
    @@ page.templ 12 replace anchor="class="            ← shorter, and usually better

Nothing else is escape-processed: `\t` is still the two characters backslash and
`t`, not a tab. Quote the whole value if it contains spaces, keep it short, and
prefer a distinctive fragment of the line over the whole line.

A wrong anchor fails loudly and prints your anchor beside the real line, so it
costs one attempt rather than a bad write. (This very edit needed `body=30`,
because those two example lines begin with `@@ ` and would otherwise be read as
headers — the escape hatch documented above.)

`body=N` takes exactly N following lines as the body, so a body may itself
contain lines starting with `@@ `. The count is checked in both directions: a
plan that ends before it is satisfied is refused, and so is text after it is
satisfied.

One case is checked more closely, because it is the only way left to lose a
hunk in silence. If a counted body line is a **complete, valid header**, an
overcount would swallow that hunk and the plan would still apply — so it is
refused, and the message names the hunk. Prose about the format is unaffected:
these two lines are not valid headers, because their trailing text is not
`key=value`.

    @@ page.templ 12 replace anchor="class="  ← what you meant
    @@ page.templ 12 replace lines=1          ← and this

When a body really does contain a real header — a plan editing this README, or
a test fixture — say `raw=true` and the check stands down for that hunk:

    @@ docs/example.md 4 replace body=1 raw=true
    @@ a.go 1 replace

If any hunk fails, **every** hunk is reported and nothing is written. Siblings
report `skip` in the human output and `"skipped"` in `--json`, never `ok`.

### A delete says what it removed

Every other op carries a body you wrote. For `replace` that body is itself
proof you looked at what you were addressing — you cannot write the new lines
without reading the old ones. An insertion's body proves less than it appears
to: its address is a position, so the body says what to add and nothing about
where, and `anchor=` is what pins that. `delete` has neither, so it is the one
op where a range that is a line too long removes something you never
saw. The receipt closes that gap: a delete names the first and last line it
took.

```
ok   a.go 201-204 delete  -4 +0 from "}" to "var _ = fmt.Sprintf"
```

Two strings, whatever the size of the range — a 500-line delete prints the same
two — each trimmed to 60 characters the way a failed `anchor=` trims the line it
prints. In `--json` they are `removed_first` and `removed_last`, present on
delete hunks only.

### A delete may say which lines it expects to remove

The receipt tells you afterwards. To be told *before* anything is written, give
the `delete` a body: the lines you expect it to remove.

```
@@ a.go 201-204 delete
	}
	return out, nil
}
var _ = fmt.Sprintf
```

If those lines are not exactly what the range holds, the hunk fails, the message
names the first line that differed — your text beside the file's — and, as
always, **nothing in the plan is written**. A `delete` with no body is unchanged
and still the right thing to write for a two-line removal.

This is the fourth guard, and the only one you cannot get wrong by accident:

| guard | asserts | who computes it |
|---|---|---|
| `sha=`, `lines=`, `anchor=` | the file, the range, the line | you, from what you read |
| a body on `delete` | every line the range holds | you, from what you *believe* it holds |

The distinction is the whole point. mrw can check a range against the file it
just served you, but any guard it derives for you is computed from the same
bytes it would check against, so it always passes. What the caller believed
lines 201-204 contained is the one fact not already in the system — which is
also why writing the body by copying it back out of `mrw read` asserts nothing.
Write it from your intent, or leave it off.

## Stats — can a caller actually author a plan?

```sh
mrw stats            # what became of the plans this checkout has been given
mrw stats --json     # the same numbers, machine-readable
mrw stats --reset    # empty the tally, saying how many records it discarded
```

Every number this project publishes about mrw — the byte savings, the round
trips — assumes the plan was authored correctly. Nothing measured that. `mrw
stats` does:

```
  applied           1 of 3 plan(s) (33.3%)
  refused_apply     1 of 3 plan(s) (33.3%)
  refused_parse     1 of 3 plan(s) (33.3%)
```

**`refused_parse` is the one that matters.** It is the only outcome that says
the FORMAT was the problem rather than your picture of a file. ADR-009
pre-registers the reading: above **5% of plans**, the format is what needs
changing, not the caller.

### The first reading — and why it is not a rate

Taken 2026-09-03 on this repository, on an Apple M5, from the tally this
repository's own development produced:

```
  applied           9 of 9 plans (100.0%)
```

**Do not read that as 100%.** Nine is below the floor ADR-009-T3 sets for
publishing a rate at all — thirty — and a percentage on nine samples is noise
wearing a decimal point. It is printed here as a COUNT, with its floor stated,
because an admission is worth more than a figure nobody should act on.

**The population is the narrowest one possible.** Nine plans, one repository,
one model, one session, authored by someone who had just written the format's
documentation. That is the best case for the format and the worst case for
generalising from it. ADR-009's criterion — parse refusals above 5% mean the
FORMAT is the problem — is not yet testable against this.

**What the tally cannot show, at any sample size.** It counts parse refusals,
and only those are about the format. It cannot distinguish a model that could
not author a plan from a model that authored a correct plan for a file that had
moved — that shows up as `refused_apply`, which is about the caller's picture of
the tree rather than about the document. And it says nothing about plans that
were never attempted because the author reached for a different tool.

Re-run `mrw stats` in your own checkout rather than quoting this. A second
reading, once the sample is real, is a follow-up on ADR-009.

**Counts only.** No plan text, no paths, no anchors, no SHAs, no command lines —
the tally is something you can read in full and find nothing of your work in,
and a test reads the written bytes to keep it that way. Nothing is ever
transmitted; it lives beside the ledger in the directory `mrw seen` names, and
this command is the only reader.

A rate always carries its denominator, because a percentage without its sample
size is the form that gets quoted out of the population it was measured on. An
empty tally says so in words rather than printing zeros — nothing measured is
not the same as nothing failed.

## Read before modify

`mrw` refuses to edit a file whose current contents it has not seen.

```
$ mrw write plan.mrw
FAIL f.txt 2 replace (plan line 1): f.txt has not been read: mrw does not know
what it currently holds, and a line address means nothing without that.
Run `mrw read f.txt` first, or pass --force
```

This is the guarantee the harness's own `Write` tool has — it will not
overwrite a file you have not `Read` — and a *range* edit needs it more, not
less: `replace 42-58` means nothing without the version of the file those line
numbers were counted in.

`mrw read` and `mrw write` both record what each file now holds, in
a per-checkout state directory **outside the working tree** — mrw creates
nothing in your repository. `mrw seen` prints where it is and what it holds.

Run mrw **one call at a time** against a checkout — this is the CLI path's
limitation, and after `mrw mcp` it is no longer the whole story. Each read
rewrites the whole ledger for that checkout, and parallel invocations overwrite
one another's entries — 40 racing reads kept 5. Nothing is corrupted and nothing
is wrongly written; the cost is that a file whose entry was lost has to be read
again. Naming every path in ONE `mrw read` is both faster and unaffected, which
is the call shape the tool is built around anyway.

Calls made **through the server** do not race: one server is one process and
serializes its own calls in-process, so an agent speaking MCP can fan out
freely. A `mrw` invocation running *beside* a server is still a second process
rewriting the same ledger file, so that pairing is still subject to the
paragraph above.

What is recorded is **what you were shown**, not what mrw hashed. A read that
printed no content observes nothing, and a read of lines 1-5 observes lines
1-5: you are the one counting line numbers, so an address in lines you never
saw is exactly the stale picture this refuses.

| you do | result |
|---|---|
| edit a file never read | refused — read it first |
| read it, then edit | applies |
| `mrw read f.go --stat`, then edit | **refused** — a stat prints no content |
| `mrw read f.go:1-5`, then edit line 40 | **refused** — you have not seen line 40 |
| `mrw read f.go:1-5`, then edit line 3 | applies |
| `mrw read f.go:/nomatch/`, then edit | **refused** — it printed nothing, so it observed nothing |
| edit again straight after | applies — mrw knows what it just wrote, all of it |
| something else changes the file, then you edit | **refused** — changed since mrw last saw it |
| `mrw write --force` | applies regardless |
| `create` a new file | applies — no existing content to be stale about |

The "changed since" row is the one that matters. Because the ledger is written on
**write** as well as on read, a chain of edits needs no re-read between steps,
while an edit made behind mrw's back leaves the recorded sha and the real one
disagreeing:

```
FAIL f.txt 1 replace: f.txt changed since mrw last saw it
(recorded 4b7a79c7, now 58ae9445): re-read it before editing,
or pass --force to overwrite blind
```

A per-hunk `sha=` guard is still available and is stronger where you want it
pinned in the plan itself; the ledger is the ambient default that costs no
tokens to use.

## What a write will not do

Three boundaries, each one a bug that was found by trying to break the tool
rather than by reading it:

- **It will not write outside `--root`.** A `../` in a hunk's path is refused,
  and so is a symlink whose target leaves the tree. `-C` names the scope of
  what a plan may change, and used to only name where paths start from.
- **It will not replace a symlink.** The write goes to the file the link points
  at. (Writing by rename is what makes a crash mid-write safe; renaming over
  the link would leave the edit in a new regular file and the real one
  untouched.)
- **It will not half-apply a plan because the filesystem said no.** Every file
  is staged beside its target first, and only then are they all renamed into
  place. A write phase that wrote each file and moved on left the earlier ones
  already applied when a later one could not be written — with no receipt, and
  with the ledger still holding the pre-write hash, so the next edit to a file
  mrw had itself changed was refused as "changed since mrw last saw it". An
  unwritable directory, a read-only mount and a full disk all fail during
  staging, when nothing has been renamed. If a *rename* fails after earlier
  renames succeeded, the tree really is partial and the error says which files
  are already written.
- **It will not change your line endings.** A CRLF file comes back CRLF, an LF
  file LF, and a file that mixes them is left mixed — the lines a hunk did not
  address survive byte for byte. A file terminated with lone `\r` has
  addressable lines like any other.

## The working set — write once, use many

```sh
mrw iter note "scoped check wiring"
mrw iter add internal/check/check.go internal/check/check_test.go
mrw iter add 'internal/read/read.go:/func Run/,/^}/'
mrw iter                       # list, numbered
```

```
@1   internal/check/check.go
@2   internal/check/check_test.go
@3   internal/read/read.go:/func Run/,/^}/
```

Those numbers are addresses — a shared symbol table between your context and the
tool, so a later call costs `@3` instead of a path:

```sh
mrw read                 # the whole working set, at its recorded ranges
mrw read @1:20-40        # entry 1's path, this range instead of its own
mrw read @1-2 @3
```

```
@@ @2 88 insert-after
        // a hunk can point into the set too
```

The `@` sigil is required: a bare number is a legal filename, and would resolve
silently to the wrong thing. An out-of-range pointer is an error, never an empty
result. Entries live in `.mrw/iteration` — plain text, diffable, hand-editable.

## Check — the tests for what you are working on

```sh
mrw check                # scoped to the working set
mrw check internal/apply # that package and everything under it
mrw check .              # every package in the tree, scoped
mrw check --full         # the whole project, unscoped
mrw write --check plan.mrw
```

The command comes from `.quality-harness.json`:

```json
{
  "check": "go test ./...",
  "scoped_check": "go test {packages}",
  "timeout_seconds": 300,
  "tail_lines": 30
}
```

**A codegen step belongs inside the check.** `check` is arbitrary shell, so a
stack with a generate step between edit and test chains it there rather than
losing `--check` entirely:

```json
{"check": "templ generate && go test ./...",
 "scoped_check": "templ generate && go test {packages}"}
```

It composes: editing a `.templ` is not a `.go` path, so the scope falls back to
the full command — which is what you want after regenerating anyway.

**Where state lives.** `mrw seen` prints the per-checkout state directory
(`$XDG_STATE_HOME/mrw/<key>/`, or `~/.local/state/mrw/<key>/`) and the ledger.
mrw writes nothing into your repository; a pre-existing `.mrw/` from an older
version is copied across once, announced, and never deleted.

`{packages}` expands to the Go packages your paths cover, `{files}` to the paths
themselves. **Write the placeholder unquoted**: each value arrives already
quoted as one shell argument, so a path holding a space, a `;` or a `$(…)` is
one argument and not shell syntax — `go test {packages}` is right, `go test
"{packages}"` nests the quotes inside the argument. A **file** maps to its own package, `./dir`. A **directory** maps to
its subtree, `./dir/...` — because `mrw check .` is how you say "check
everything here", and go's `./dir` is the one package at the top: scoping a
directory that way reported PASS with a failing package one level down. The
trailing `/...` is stripped before a path is placed, so the scope mrw prints can
be handed straight back to it.

Anything mrw cannot place as a package abandons the scoped form for the full
one: a `.md` or a `.templ`, a path that is not there, a directory holding no
package go will build (prose, or one named `testdata`). A directory that exists
and **cannot be read** is refused instead — mrw cannot tell a package it cannot
look at from an absent one, and falling back would answer about the whole
project for a scope it never opened. A scoped run that
quietly omits a changed file is worse than a slow complete one, and a run scoped
to nothing that reports PASS is worse than both.

A path resolving **outside the root** is the one exception, and is refused (exit
2). The fallback is what makes every case above safe, because the whole-project
run still covers a typo — but it covers nothing you named when the name pointed
elsewhere, so the verdict it printed was about a different tree. `mrw check
../other` answered PASS at exit 0 while `../other` did not compile, and answered
3 when this repository's own tests went red: it tracked the root and never the
argument. `read` and `write` refuse such a path already; so does `check` now. An
absolute path is the same escape in the spelling that hides it — it is joined
onto the root rather than honoured, so it lands inside, places nothing and fell
back — and it is refused for the same reason.

Three rules it will not bend, each from a check that lied:

- **The exit code is never inferred from output.** Output goes to a file, a
  bounded tail is shown, the process's real status is reported. A `tail` in the
  pipeline would make the pipeline's status the tail's, so a failing suite would
  surface as a pass.
- **A check that did not run is not a pass.** No `.quality-harness.json` and no
  `go.mod` means no evidence, and it says so.
- **A red check never triggers a revert.** You are told, with a distinct exit
  status; undoing your edit could destroy work you wanted to inspect.

An inferred command is labelled `inferred`. That matters: an inferred check can
be red on a tree you never touched, and that finding is about the machine, not
about your change.

## Exit status

| code | meaning | what to do |
|---|---|---|
| 0 | everything asked for succeeded | — |
| 1 | a hunk failed, or the answer is incomplete; **nothing written** | fix the plan, or read the output |
| 2 | usage, parse or I/O error | fix the call |
| 3 | a check ran and did not pass | read the test output |

Exit 1 on `read` means **incomplete**, not necessarily wrong. Four things
produce it, because a partial answer that looks whole is the failure this tool
is built around:

| the output says | what happened |
|---|---|
| `UNREADABLE` | the file could not be opened |
| `REFUSED` | the path resolves outside `--root` (ADR-006) |
| `no match for …` | a pattern matched nothing |
| `WITHHELD` / `more line(s) withheld` | a `--max-lines` cap you asked for |

The exit status does not tell them apart; the output always does, and each one
names what is missing.

`--root`/`-C` moves the paths **inside** a plan. The plan file itself is a shell
argument like any other and resolves against your working directory, so
`mrw -C repo write plan.mrw` looks for `plan.mrw` beside you and not in `repo`.
A miss says which directory it looked in.

## A note on hooks

`mrw` writes files from a shell, which is normally the thing to avoid: harness
gates keyed to `Edit`/`Write` cannot see a `sed`, a heredoc or a `python -`, so
those bypass a guardrail silently.

`mrw` is different in the way that matters — it is **one named binary with a
machine-readable receipt**, so a `Bash` hook can recognise and inspect it in a
line or two. `sed`, `awk`, `cat >`, `echo >>`, `python -`, `perl -i`, `tee` and
`printf >` are eight spellings with the paths buried in arbitrary shell, and no
hook can reliably read them. Gate-ability is the whole argument; `mrw --json`
exists to serve it.

For anything material, author the plan with your harness's own file tool and
pass its path rather than piping it in. A plan on disk is reviewable, is visible
to whatever watches file writes, and — the token-economics point — can be
`--dry-run` and then applied from **one** emission instead of two.
