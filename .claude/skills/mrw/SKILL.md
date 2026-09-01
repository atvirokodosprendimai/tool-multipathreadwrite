---
name: mrw
description: >-
  Pointer to the authoritative mrw skill, which lives centrally in agentsmemory
  because mrw is installed globally and is used from repositories that cannot
  see this file. Load it with am_load_skill("mrw"). FIRST CHECK THE BINARY
  EXISTS — `command -v mrw || ls ./bin/mrw`; if neither resolves, mrw is not
  installed: use Read/Edit/Write, and do not install it or improvise a
  substitute with sed/awk/python. In short, mrw reads many file ranges and
  applies many edits in ONE call, with a per-hunk verdict and a
  read-before-modify guard; use it for 3+ edits, edits across 2+ files, or
  several ranges read. One or two targeted edits stay on Edit; a new file stays
  on Write. NOT a licence to use shell for ordinary file edits.
---

# mrw — pointer, not the record

**The authoritative copy is the centralised team skill.** Load it:

```
am_load_skill("mrw")
```

## Why it lives there and not here

`mrw` is installed globally (`/usr/local/bin/mrw`), so it is used from
repositories that never see this file. A copy discoverable only inside its own
repo is missing from every place the tool is actually reached for.

This file was a second full copy until 2026-08-31. Two live copies of the same
guidance drift, and the stale one is indistinguishable from the current one at
the moment you read it — so this one was reduced to a pointer rather than kept
in sync by hope.

**Do not re-expand it.** If the guidance needs changing, change the centralised
skill with `am_update_skill`. The only things that belong here are facts about
*this repository*, below.

## ⚠ Step 0 — does the binary exist?

Repeated here because it is the one thing you need before deciding whether to
load anything at all:

```sh
command -v mrw || ls ./bin/mrw
```

If neither resolves, mrw is not installed. Use Read / Edit / Write, say nothing
about mrw, and do **not** install it or hand-roll a substitute out of
`sed`/`awk`/`python -` — reaching for shell because the nice tool is missing is
how the file-edit ban gets broken by a good intention.

## Facts about this repository

- Build: `go build -o bin/mrw ./cmd/mrw`. Doing so here is fine — it is the
  project you were asked to work on. Not elsewhere.
- `./scripts/measure.sh` — what mrw saves, three shapes, including the one where
  it loses. Quote the script, never a remembered number.
- `./scripts/contract.sh` — 64 assertions against the real binary, each made by
  breaking a promise on purpose. Mutation-verified; runs in CI.
- `.quality-harness.json` declares the check: `go test ./...`.
- The read-before-modify guard arrived in **v0.0.2**. A `bin/mrw` built from a
  working tree reports `dev`, which is newer than any tag, not older.
