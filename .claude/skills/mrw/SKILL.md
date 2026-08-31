---
name: mrw
description: >-
  Read many file ranges and apply many edits in ONE call, with a per-hunk
  verdict, a read-before-modify guard, and the project's own tests chained to
  the write. FIRST CHECK IT EXISTS — `command -v mrw || ls ./bin/mrw`; if
  neither resolves the binary is not installed, so abandon this skill silently
  and use Read/Edit/Write rather than installing it or improvising with shell.
  When it does exist, use it for 3+ edits, edits across 2+ files, or several
  ranges read — where Edit (one replacement per call) and Write (whole file,
  cannot say which change missed) both cost you. One or two targeted edits stay
  on Edit; a new file stays on Write; one site in a file you need whole is
  measurably worse in mrw. NOT a licence to use shell for ordinary file edits.
---

# mrw — multi-path read and write

## ⚠ STEP 0 — DOES IT EXIST? IF NOT, THIS SKILL DOES NOT APPLY

```sh
command -v mrw || ls ./bin/mrw
```

**If neither resolves, stop and use Read / Edit / Write.** Say nothing about
mrw — a tool the machine does not have is not a fallback the user needs
narrated. Do NOT install it, and do NOT hand-roll a substitute out of
`sed`/`awk`/`python -`: reaching for shell because the nice tool is missing is
how the file-edit ban gets broken by a good intention.

Three ways it may be present, in the order to prefer:

1. **On PATH** (`command -v mrw`) — a released binary. Use `mrw`.
2. **`./bin/mrw`** in the repo you are in. Use `./bin/mrw`.
3. **You are in the mrw repo itself** and `bin/mrw` is missing:
   `go build -o bin/mrw ./cmd/mrw` is fine — you are building the project you
   were asked to work on. Nowhere else.

⚠ **Version matters for one thing**: the read-before-modify guard arrived in
**v0.0.2**. `mrw --version` reporting older means that guard is absent, so
`sha=` on each hunk is your only staleness protection.

Releases carry `mrw-<os>-<arch>`, archives and `SHA256SUMS.txt`:
https://github.com/atvirokodosprendimai/tool-multipathreadwrite

## ⚠ READ THIS FIRST — it does NOT replace Edit and Write

Ad-hoc shell file-writers are banned in this workspace because gates keyed to
`Edit`/`Write` cannot see a heredoc. **That ban still stands.** The narrowing
that would exempt a single named tool emitting a machine-readable receipt is
drafted and awaiting M's approval — this file is not that approval.

So the choice is about CAPABILITY, exactly as before:

| situation | tool |
|---|---|
| one or two targeted edits in one file | **Edit** |
| a new file | **Write** |
| whole-file rewrite you have just Read | **Write** |
| **3+ hunks, or hunks across 2+ files** | **mrw write** |
| **several ranges, from one or many files** | **mrw read** |
| you need to know *which* of N changes missed | **mrw write** |

The gap it fills: Edit batches nothing but fails loudly on a bad anchor; Write
batches a whole file but cannot say which intended change did not land. The
originating bug was four replacements in one script where one silently matched
nothing and the script printed success.

★ **A read that returns nothing is visible; a write that changes nothing is
not.** That asymmetry is the whole reason this exists.

## read

```sh
mrw read a.go:1-8,100-130 b.go:/func Handle/,/^}/ c.go:3- --stat
```

A range is `3-6`, `5`, `3-` (to EOF), `-20` (from the start), `/pattern/` (each
matching line, `-C N` for context) or `/start/,/end/`. Overlapping ranges merge,
so no line is printed — or paid for — twice.

- `--stat` — length, bytes, sha only. Ask for the fact, not the artifact.
- `--max-lines N` — cap per file; whatever is withheld is always announced.
- `-C N` context, `-N` no line numbers, `-C DIR` on the root command sets the base.

Output ranges print as `@@ 3-6` — **exactly the address a write plan takes**, so
a read and the edit it informs share one vocabulary.

⚠ **Quote a spec containing a space**: `'a.go:/func Foo/,/^}/'`. Unquoted, the
shell splits it and the fragments look like plausible relative paths.

## write

```sh
mrw write plan.mrw              # apply
mrw write --dry-run plan.mrw    # validate only
mrw write --check plan.mrw      # apply, then test what changed
mrw write --json plan.mrw       # receipt for a hook or gate
mrw write -                     # plan on stdin
```

The plan is line-oriented, deliberately **not JSON** — JSON escapes every
newline and quote in every code body, which is the one part of the document
already large, and an output token costs ~5x an input one.

```
@@ internal/apply/apply.go 42-58 replace anchor="func Apply" lines=17
...new lines...
@@ internal/apply/apply.go 12 insert-after
	"sort"
@@ README.md 3-4 delete
@@ docs/new.md - create
# a new file
```

Ops: `replace`, `insert-after`, `insert-before`, `delete`, `create`. Addresses
are 1-based inclusive; `$` is the last line, `0` before the first, `N-` to EOF.

★ **EVERY ADDRESS RESOLVES AGAINST THE ORIGINAL FILE.** Read once, note several
ranges, edit them all — no offset arithmetic between hunks. This is the property
that makes batching safe, and the reason not to hand-roll it in a script.

Guards, all cheap to emit, which is the point:

| guard | asserts |
|---|---|
| `sha=<8+ hex>` | the whole file is still what you read |
| `lines=N` | the range covers exactly N lines |
| `anchor=<substring>` | it appears in the range's FIRST line |

`body=N` takes exactly N following lines as the body, so a body may itself
contain lines starting with `@@ `.

★ **Use `anchor=` on every replace you did not just read.** It is the
loud-failure half Write lacks: without it a drifted line number overwrites the
wrong lines and reports success.

⚠ If any hunk fails, **every** hunk is reported and **nothing** is written.
Siblings report `skipped`, never `ok`.

## ⚠ Read before modify — mrw refuses to edit what it has not seen

`mrw` will not edit a file whose current contents it has not observed. Same
guarantee the harness's `Write` has, and a RANGE edit needs it more, not less:
`replace 42-58` means nothing without the version those numbers were counted in.

`read` (including `--stat`) and `write` both record each file's sha in
`.mrw/seen`.

| you do | result |
|---|---|
| edit a file never read | refused — `mrw read <path>` first |
| read it, then edit | applies |
| edit again straight after | applies — mrw knows what it just wrote |
| something else changed it, then you edit | **refused** — changed since mrw last saw it |
| `--force` | applies regardless |
| `create` | applies — nothing existing to be stale about |

★ The fourth row is the point. Because the ledger is written on WRITE too, a
chain of edits needs no re-read between steps, while anything that changed the
file behind mrw's back is caught:

```
FAIL f.txt 1 replace: f.txt changed since mrw last saw it
(recorded 4b7a79c7, now 58ae9445): re-read it before editing, or pass --force
```

⚠ This catches edits made by ANY other route — another agent, a `git checkout`,
a `cp`, your editor. If you changed a file outside mrw, re-read it.

★ `mrw read --stat <path>` re-authorises a file for the price of one line: it
hashes without printing content, so confirming "nothing moved" is nearly free.
Use it when you already know the content and only need the staleness check.

## The working set — write once, use many

```sh
mrw iter note "scoped check wiring"
mrw iter add internal/check/check.go internal/check/check_test.go
mrw iter add 'internal/read/read.go:/func Run/,/^}/'
mrw iter                      # list, numbered
```

```
@1   internal/check/check.go
@2   internal/check/check_test.go
@3   internal/read/read.go:/func Run/,/^}/
```

Those numbers are a shared symbol table between your context and the tool, so a
later call costs `@3` instead of a path — the emission happens once:

```sh
mrw read              # the whole set, at its recorded ranges
mrw read @1:20-40     # entry 1's PATH, this range instead of its own
mrw read @1-2 @3      # @* is all of them
```

```
@@ @2 88 insert-after
	// a hunk can point into the set too
```

⚠ The `@` sigil is required: a bare number is a legal filename and would resolve
silently. An out-of-range pointer is an ERROR, never an empty result. In a hunk
a pointer must name exactly one entry. Entries live in `.mrw/iteration` — plain
text, diffable, gitignored, per-developer state.

## check — chain the verification to the change

```sh
mrw check                 # scoped to the working set
mrw check internal/apply  # scoped to these paths
mrw check --full
mrw write --check plan.mrw
```

Edit, run the tests, read the output is three round trips; this is one. The
command comes from `.quality-harness.json`:

```json
{"check": "go test ./...", "scoped_check": "go test {packages}",
 "timeout_seconds": 300, "tail_lines": 30}
```

`{packages}` expands to the Go packages holding the changed files, `{files}` to
the paths. If any changed path is not a Go file the scoped form is abandoned for
the full one — a scoped run that quietly omits a changed file is worse than a
slow complete one.

Three properties to rely on, each from a check that lied:

- **The exit code is never inferred from output.** Output goes to a FILE, a
  bounded tail is shown, the process's real status is reported. A `tail` in the
  pipeline would make the pipeline's status the tail's.
- **A check that did not run is NOT a pass.** It reports `SKIPPED` and `OK()` is
  false. No evidence is reported as no evidence.
- **A red check never reverts.** You are told, with a distinct exit status.

⚠ An inferred command is labelled `inferred`. That matters: an inferred check can
be red on a tree you never touched, and then the finding is about the machine,
not about your change. Say which you had.

## Exit status — the four mean different next moves

| code | meaning | do |
|---|---|---|
| 0 | everything asked for succeeded | — |
| 1 | a hunk failed / a range could not be served; **nothing written** | fix the plan |
| 2 | usage, parse or I/O error | fix the call |
| 3 | a check ran and did not pass | read the test output |

⚠ Never collapse 1 and 3 into "it failed". 1 promises an UNTOUCHED tree — fix
the plan and retry. 3 says a check ran and failed, and what that implies about
the tree depends on which command: after `write --check` the edit IS applied and
now unverified; after a bare `check` nothing was written.

## Habits

- **Author the plan with your harness's file tool and pass the path**, rather
  than piping it in. A plan on disk is reviewable, is visible to whatever
  watches file writes, and can be `--dry-run` then applied from ONE emission
  instead of two.
- **`--dry-run` first** on anything you did not read this turn.
- **Put the working set in place before a multi-step task**, not after.
- Read ranges, never whole files: 400 lines is ~6k tokens, the 30 you need ~400.

## Known limits

- `cmd/mrw` has only `version_test.go`; the CLI wiring (pointer resolution in a
  hunk path, exit-status selection) is covered end-to-end by
  `./scripts/contract.sh` rather than by Go tests.
- The payoff IS measured now — `./scripts/measure.sh`, three shapes including
  the one where mrw loses. Quote that script, not a remembered number: the
  ratio is a property of the task shape.
- This file and the centralised `mrw` skill in agentsmemory (v2) carry the same
  content and were updated together on 2026-08-31. Two live copies drift: when
  you change one, change the other in the same breath, or retire one.
