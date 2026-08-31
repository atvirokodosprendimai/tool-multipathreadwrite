---
name: mrw
description: >-
  Read many file ranges and apply many edits in ONE call, with a per-hunk
  verdict and the project's own tests chained to the write. Use when you are
  about to make 3+ edits, edits across 2+ files, or read several ranges — that
  is where Edit (one replacement per call) and Write (whole file, cannot say
  which change missed) both cost you. One or two targeted edits stay on Edit; a
  new file stays on Write. Also covers reading line ranges cheaply instead of
  whole files, a saved working set addressed by @N pointers, and running the
  project's check scoped to what just changed. NOT a licence to use shell for
  ordinary file edits.
---

# mrw — multi-path read and write

`go build -o bin/mrw ./cmd/mrw`. Intended to become part of the quality harness.

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

- `cmd/mrw` has no tests; the five engine packages do.
- Nobody has measured how often multi-hunk edits actually occur across sessions.
  The claim that this pays for itself is reasoned, not measured — say so.
- This file is the authoritative copy. A centralised `mrw` skill of the same
  content also exists in agentsmemory (v1, filed 2026-08-31). Two live copies
  can drift: when you change one, change the other or retire it.
