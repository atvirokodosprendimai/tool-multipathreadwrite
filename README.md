# mrw — multi-path read and write

One binary that reads many file ranges and applies many edits in one call, with
a verdict per hunk and the project's own tests chained to the write.

It exists because of a gap between two primitives an agent already has:

| | batches | fails loudly |
|---|---|---|
| `Edit` | no — one replacement per call | yes, on a bad anchor |
| `Write` | yes — whole file | no, it cannot say which change did not land |
| `mrw` | yes — N hunks across M files | yes, per hunk, and writes nothing on any failure |

The failure it is built around: **a read that returns nothing is visible; a
write that changes nothing is not.** Batching four replacements into one script
and getting "success" while one of them silently matched nothing is the bug this
refuses to reproduce.

## Does it actually save anything?

Measured on this repository, by a script you can re-run — a number nobody can
reproduce is a claim, not a measurement:

```sh
./scripts/measure.sh
```

**Round trips are the claim that survives every reading: 2 calls, for any N.**
Bytes depend entirely on what you compare against, so the table gives both
baselines rather than the flattering one.

| shape | | baseline | mrw | |
|---|---|---|---|---|
| **A.** 4 sites, 4 large files | bytes vs reading those files **whole** | 98,921 | 2,951 | **33.5× less** |
| | bytes vs a **windowed** `offset`/`limit` read | 2,289 | 2,951 | **1.2× MORE** |
| | calls, whole-file (reads + edits) | 8 | 2 | 4.0× fewer |
| | calls, windowed (search + reads + edits) | 9 | 2 | **4.5× fewer** |
| **B.** 2 sites, 2 mid-sized files | bytes vs whole | 20,630 | 1,329 | 15.5× less |
| | bytes vs windowed | 866 | 1,329 | 1.5× MORE |
| | calls | 4 / 5 | 2 | 2.0–2.5× fewer |
| **C.** 1 site, whole small file | bytes (window *is* the whole file) | 12,881 | 15,765 | **1.2× MORE** |
| | calls | 2 / 3 | 2 | 1.0–1.5× fewer |

Measured at `7df758b`; the script builds the binary it stamps.

**Read the two byte rows together or neither.** `Read` takes `offset`/`limit`, so
the windowed reader is the documented interface, not a strawman — and against it
mrw costs *more* bytes, because it adds a header and a line number per line. The
32.8× is real for the case an agent is usually in: it does not yet know where to
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

## Does the contract hold under abuse?

Every row below is asserted by a script, against the real binary in a throwaway
repo, by making each promise go wrong on purpose:

```sh
./scripts/contract.sh      # 198 assertions; exit 0 only if all hold
```

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

From source:

```sh
go build -o bin/mrw ./cmd/mrw
```

## Releasing

`.github/workflows/ci.yml` runs gofmt, `go vet`, `go test ./...` and
`go test -race ./...` on every push and PR. Pushing a **strict** `vX.Y.Z` tag
additionally cross-compiles five targets and publishes them:

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
| `--max-lines N` | cap per file; whatever is withheld is always reported |
| `-N` | drop line numbers |

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

Run mrw **one call at a time** against a checkout. Each read rewrites the whole
ledger for that checkout, and parallel invocations overwrite one another's
entries — 40 racing reads kept 5. Nothing is corrupted and nothing is wrongly
written; the cost is that a file whose entry was lost has to be read again.
Naming every path in ONE `mrw read` is both faster and unaffected, which is the
call shape the tool is built around anyway.

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
