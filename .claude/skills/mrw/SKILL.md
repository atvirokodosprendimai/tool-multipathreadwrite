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

# mrw — pointer to AGENTS.md

**The guidance lives in [`AGENTS.md`](../../../AGENTS.md), section "Using mrw".**
Read it there. It is trigger-first, carries the plan-generation loop that turns
54 calls into 2, and is the same text every other agent in this repository sees.

## Why it moved here from the memory server

This file used to say the authoritative copy was a centralised team skill,
loaded with `am_load_skill("mrw")`. **That skill does not exist.** Checked on
2026-09-03: `am_load_skill("mrw")` returns `skill: not found`, and
`am_list_skills` returns nine skills, none of them mrw. The pointer had been
dangling for an unknown period, and a dangling pointer reads exactly like a
working one until someone follows it.

The reasoning that put it there was sound and still is: mrw is installed
globally, so it is reached for from repositories that never see this file, and a
copy discoverable only inside its own repo is missing from most of the places
the tool is used. That problem is **not solved here** — the repo-local answer
serves anyone who clones this repository or runs an agent inside it, and nobody
else. Shipping the text inside the binary (`mrw instructions`) is the option
that would reach every repository; it was considered on 2026-09-03 and not
taken, because it makes a new public contract and this file was the cheaper fix.

**One authored copy, in AGENTS.md.** Do not expand this file back into a second
one, and do not point it at a skill without loading that skill first to see
whether it answers.

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
- `./scripts/contract.sh` — every promise the README makes, asserted against the
  real binary by breaking each one on purpose. Mutation-verified; runs in CI on
  Linux only, because it drives a POSIX shell. It prints its own assertion count;
  do not repeat one here, because a number written beside the script drifts from
  it silently — this line claimed 76 for weeks while the script asserted 216.
- `go test ./...` runs in CI on **Linux and Windows**. The windows job found a
  real defect on its first run: `filepath.IsAbs` is false there for `/etc/hosts`,
  so four root-relative guards were skipped — use `rooted.IsRooted`, never
  `filepath.IsAbs`, to ask whether a caller's path is relative to the root.
- `.quality-harness.json` declares the check: `go test ./...`.
- The read-before-modify guard arrived in **v0.0.2**. A `bin/mrw` built from a
  working tree reports `dev`, which is newer than any tag, not older.
