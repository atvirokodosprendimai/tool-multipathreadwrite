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

| shape | | Read+Edit | mrw | |
|---|---|---|---|---|
| **A.** 4 sites, 4 large files | bytes in | 50,163 | 2,916 | **17.2× less** |
| | tool calls | 8 | 2 | **4× fewer** |
| **B.** 2 sites, most of 2 small files | bytes in | 10,827 | 1,790 | 6.0× less |
| | tool calls | 4 | 2 | 2× fewer |
| **C.** 1 site, whole small file | bytes in | 3,895 | 4,888 | **1.2× MORE** |
| | tool calls | 2 | 2 | no change |

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
./scripts/contract.sh      # 19 assertions; exit 0 only if all hold
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

**Every address resolves against the original file.** Read once, note several
ranges, edit them all — no offset arithmetic between hunks.

Ops are `replace`, `insert-after`, `insert-before`, `delete`, `create`.
Addresses are 1-based and inclusive; `$` is the last line, `0` is before the
first, `N-` runs to EOF.

Three optional guards make a batch safe to trust, and all three are cheap to
write, which is the point:

| guard | asserts |
|---|---|
| `sha=<8+ hex>` | the whole file is what you read |
| `lines=N` | the addressed range covers exactly N lines |
| `anchor=<substring>` | it appears in the range's first line |

`body=N` takes exactly N following lines as the body, so a body may itself
contain lines starting with `@@ `.

If any hunk fails, **every** hunk is reported and nothing is written. Siblings
report `skipped`, never `ok`.

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
`.mrw/seen`. So:

| you do | result |
|---|---|
| edit a file never read | refused — read it first |
| read it, then edit | applies |
| edit again straight after | applies — mrw knows what it just wrote |
| something else changes the file, then you edit | **refused** — changed since mrw last saw it |
| `mrw write --force` | applies regardless |
| `create` a new file | applies — no existing content to be stale about |

The fourth row is the one that matters. Because the ledger is written on
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
mrw check internal/apply # scoped to these paths
mrw check --full         # the whole project
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

`{packages}` expands to the Go packages containing the changed files, `{files}`
to the paths. If any changed path is not a Go file the scoped form is abandoned
for the full one — a scoped run that quietly omits a changed file is worse than
a slow complete one.

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
| 1 | a hunk failed, or a range could not be served; **nothing written** | fix the plan |
| 2 | usage, parse or I/O error | fix the call |
| 3 | a check ran and did not pass | read the test output |

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
