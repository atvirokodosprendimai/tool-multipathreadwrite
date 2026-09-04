# AGENTS.md — working in this repository

For any coding agent (Codex, Cursor, Claude Code, or a human who wants the short
version). Nothing here requires a plugin, an MCP server, or a private skill.

## What this is

`mrw` is one Go binary at `./cmd/mrw`. It reads many ranges across many files,
and applies many edits across many files, in **one** invocation — and it reports
a verdict for **every** edit. Read `README.md` for the full interface; this file
is only about changing the code.

## Build and check

```sh
go build -o bin/mrw ./cmd/mrw          # bin/mrw.exe on Windows
go test ./...                          # the project's declared check
go test -race ./...
gofmt -l .                             # must print nothing
go vet ./...
./scripts/contract.sh                  # bash; WSL or Git Bash on Windows
```

`.quality-harness.json` names `go test ./...` as the check. A run that is piped
or ends in `|| true` is not evidence: the exit code is the whole verdict, and a
pipeline reports the *last* command's status, not the script's.

## The rules this codebase is built around

These are decided, recorded in `docs/adr/`, and asserted by
`scripts/contract.sh`. Do not relax one without retiring the ADR.

1. **A plan applies whole or not at all.** Any failing hunk writes nothing.
   Siblings report `skip`, never `ok`. — ADR-001
2. **mrw will not edit a file it has not read**, and the guard is per *line*,
   not per file. — ADR-002
3. **A check's verdict comes from the process, never its output.** A check that
   prints `PASS` and exits 1 is a failure. — ADR-003
4. **Nothing is left in the working tree** by a failed run. — ADR-004
5. **mrw finds the files it serves**, and says which path it looked for when it
   cannot. — ADR-007
6. **A delete says what it removed.** — ADR-008

The failure the whole design exists to prevent: *a read that returns nothing is
visible; a write that changes nothing is not.*

## Portability — one trap, learned the hard way

**Never ask `filepath.IsAbs` whether a caller's path is relative to the root.
Ask `rooted.IsRooted`.** On Windows `/etc/hosts` and `\etc\hosts` are *rooted* —
they name the root of the current drive — but carry no volume, so `IsAbs` is
false for both, and `C:etc` is drive-relative and equally not root-relative.
Four guards were skipped on Windows because of this. In `check` it produced a
silent PASS: the joined path placed no package, the run fell back, and the
verdict was green. The `windows` CI job found it on its first run.

Go tests run on Linux **and** Windows in CI. `scripts/contract.sh` runs on Linux
only — it drives a POSIX shell.

## Exit codes are the contract

`0` fine · `1` a hunk failed and nothing was written · `2` usage or filesystem
failure · `3` the write applied but `--check` failed, so the tree is changed and
unverified. Tests assert these; changing one is a breaking change.

## Adding behaviour

- A new promise needs a row in `scripts/contract.sh` that makes it go wrong on
  purpose. A test that cannot fail is worse than no test.
- A durable decision — a public contract, an exit code, a trust boundary — needs
  an ADR in `docs/adr/`, not just a commit message.
- Exported identifiers carry doc comments starting with the name. Packages carry
  a package comment.

## Using mrw — read this before you edit anything

**Reach for mrw when the task touches 3 or more edits, 2 or more files, or
several ranges you need to read.** Below that, use your normal editor: one edit
in one file costs mrw the same two calls and prints *more* bytes than the file
holds. It is for scattered sites, not for every edit.

`./bin/mrw` after `go build`. Building and using it here is fine — it is the
project.


If you are reaching mrw over MCP (`mrw mcp`), use the `mrw_read` and `mrw_write`
tools instead of the shell recipes below. The arguments are the same strings —
`specs` is what you would pass to `mrw read`, `plan` is the file you would pass
to `mrw write` — and every rule in this section applies unchanged, because it is
the same engine and the same ledger.

### 1. Read many ranges in one call, and let the read do the finding

Addresses are line numbers, `N-M` ranges, `$` for the last line, or a **regex**
— so you do not need a separate search call to locate the site:

```sh
mrw read 'internal/apply/apply.go:/^func Apply/,/^}/' \
         'internal/read/read.go:120-160' \
         'cmd/mrw/main.go:$'
```

**And when you do not know which files to name, do not enumerate them —
`--grep` walks and serves in one call:**

```sh
mrw read --grep 'func Handle' -C 3 --exclude vendor --exclude '*_test.go' internal/
```

A named directory is walked; with no paths at all the walk starts at `--root`.
`--exclude GLOB` is repeatable and matches against **both** the root-relative
path and the basename — the basename half is what makes `'*_test.go'` work at
any depth, because `path.Match`'s `*` does not cross `/`. A pattern that matches
no file is reported by name and exits 1.

It does **not** read `.gitignore` and does not sniff for binary files: a regular
file is a candidate, so exclude build artifacts by name (`--exclude bin`).
`.git/` is always skipped.

**`--files-from` is the same idea for a searcher you already trust:**

```sh
rg -l 'func Handle' | sed 's|$|:/func Handle/|' | mrw read -C 3 --files-from -
```

Blank lines are skipped and a leading `#` is a comment. Use it when your own
index is better than a walk, or when `--grep` is not what you want.

⚠ **In Git Bash on Windows, MSYS rewrites a regex address before mrw is
started** — `f.go:/^func main/` arrives as `f.go;C:\…\Git\^func main\` and the
error names a line number you never typed. Quoting does not help; it happens in
the process-spawn layer, after the shell. Export `MSYS2_ARG_CONV_EXCL='*'`, or
use PowerShell or WSL. Line-number and `$` addresses are unaffected.

⚠ **A shell glob and an address suffix do not mix.** `mrw read 'dir/*.go:1-3'`
is served with the star taken literally, so it reports the path UNREADABLE; and
unquoted, zsh refuses it before mrw starts — *"no matches found"* — because
`dir/*.go:1-3` matches no file. Quoting is what most callers try next and it is
the wrong half of the fix. Use `--grep` to walk and serve in one call, or
`--files-from` to pipe a list in and add the suffix per line. mrw says this in
the UNREADABLE line too, but by then you have spent a call.

### 2. One plan, not N writes

Every hunk gets a verdict. If any hunk fails, **nothing is written at all** and
the siblings report `skip`, never `ok`. Ops are `replace`, `insert-after`,
`insert-before`, `delete`, `create`.

```
@@ internal/apply/apply.go 42-58 replace anchor="func Apply" lines=17
        ... new lines ...
@@ cmd/mrw/main.go 12 insert-after
        "sort"
```

Every address resolves against the **original** file, so there is no offset
arithmetic between hunks. Guards `sha=`, `lines=` and `anchor=` are checked on
every op, insertions included.

Or address by pattern, when you would otherwise read the file only to learn a
line number:

```
@@ internal/store/store.go /^func \(s \*Store\) Get/,/^\}/ replace
        ... new lines ...
```

A pattern is **not** a way to edit a file you have not read: it resolves to a
line, and that line still has to have been served to you.

### 3. Generate the plan when there are many sites — this is the lever

A plan is line-oriented text, so anything that prints lines can build one. This
is the part that turns 54 calls into 2, and it is the part that gets missed.
**A plan address may be a line number, an `N-M` range, `$`, or a pattern —
`/regexp/` or `/from/,/to/`.** A pattern must match **exactly one** line; none
or several fails that hunk and the refusal names the lines it matched. Reach
for a pattern when you have not read the file for any other reason; take the
numbers from the read you just did when you have:

```bash
specs=()
for f in $(git ls-files '*.go'); do specs+=("$f:/^package/"); done

out=$(mrw read "${specs[@]}")                       # call 1: every site, one read

awk '/^==> /{f=$2} /^ *[0-9]+\|/{n=$1; sub(/\|/,"",n);
     print "@@ " f " " n " replace"; print "package fresh"}' <<<"$out" \
  | mrw write --check -                             # call 2: every edit, one write
```

Use `bash`, not `zsh`: zsh does not word-split an unquoted `$specs`, so the
whole list arrives as one argument and the regex swallows the rest of the line.

### 4. The rules that will bite you

- **Read before write is ENFORCED, and it is per LINE.** Being served lines
  10-12 does not license an edit at line 50.
- **`--check`** applies the edits and then runs the project's tests scoped to
  what changed. **`--json`** gives a parseable receipt on success *and* failure.
- **Exit `3` means the write APPLIED and the check failed** — the tree is
  changed and unverified. It is not a rollback.
- **Never read an exit code through a pipe.** `mrw write plan | head` returns
  head's status. This is the single most common way a red run reads as green.
- A refusal is the tool working. It names the file, the plan line and the
  reason; read it rather than reaching for a bigger hammer.

### 5. The other subcommands

Four commands exist that the recipes above never reach for. They are listed
here because the centralised `mrw` skill mirrors this file, so a command
absent from it is a command no agent knows to ask for — the defect in issues #51
and #73, one release apart.

- **`mrw check`** runs the project's check on its own, scoped to the working
  set or to paths you name. `mrw write --check` is the same runner bolted onto
  a write; this is it without the write, for when you want the verdict again
  without applying another plan. ⚠ It is NOT read-only: the declared command is
  whatever the project declared, run with `sh -c` in the checkout, so a check
  that generates code or writes fixtures does exactly that.
- **`mrw stats`** prints what became of the plans this checkout has been given
  — how many applied, how many were refused because the document did not PARSE,
  and how many parsed but failed to APPLY. That last one is deliberately one
  bucket and not three: a failed guard, an unread line and a path outside the
  root all land in it, because splitting them would mean matching on free-form
  reason text that changes. Reach for it to see whether the format is costing
  you attempts, rather than guessing.
- **`mrw iter`** shows or edits the working set: the specs mrw is currently
  carrying. `mrw seen` prints the state directory and the read-before-modify
  ledger — the record of which lines you have actually been served, which is
  what licenses a write. When a write is refused for a line you believe you
  read, `mrw seen` is the file that settles it.

mrw's OWN state lives outside the tree (ADR-004), so none of these commands
writes to your checkout of its own accord — with the one exception above, where
`mrw check` runs a command the project chose.

See `CONTRIBUTING.md` for the gate list and the release process, and `README.md`
for the full interface.
