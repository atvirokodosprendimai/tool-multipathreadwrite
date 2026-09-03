---
name: mrw
description: >-
  The mrw guidance for THIS repository lives in AGENTS.md, section "Using mrw" —
  that is the authored source. A centralised agentsmemory skill mirrors it for
  sessions in other repositories: am_load_skill("mrw"). FIRST CHECK THE BINARY
  EXISTS — `command -v mrw || ls ./bin/mrw`; if neither resolves, mrw is not
  installed: use Read/Edit/Write, and do not install it or improvise a
  substitute with sed/awk/python. In short, mrw reads many file ranges and
  applies many edits in ONE call, with a per-hunk verdict and a
  read-before-modify guard; use it for 3+ edits, edits across 2+ files, or
  several ranges read. One or two targeted edits stay on Edit; a new file stays
  on Write. NOT a licence to use shell for ordinary file edits.
---

# mrw — AGENTS.md is the source, the AAM skill is the copy

**The guidance lives in [`AGENTS.md`](../../../AGENTS.md), section "Using mrw".**
That is the authored source: trigger-first, carrying the plan-generation loop
that turns 54 calls into 2, and it is the same text every other agent in this
repository sees — with or without a memory server.

A **centralised agentsmemory skill mirrors it** for sessions working in *other*
repositories, where this file is not visible:

```
am_load_skill("mrw")
```

## Which one wins, and how they stay together

`AGENTS.md` is authored; the skill is a mirror of it. **Correct AGENTS.md first,
then push the same change with `am_update_skill`** — and if you find yourself
correcting the skill first, back-port it here in the same session. Two live
copies drift, and the stale one is indistinguishable from the current one at the
moment you read it.

The reason the mirror exists at all: mrw is installed globally and is reached for
from repositories that never see this file, so a copy discoverable only inside
its own repo is missing from most of the places the tool is used. Shipping the
text inside the binary (`mrw instructions`) is the option that needs no memory
server and reaches every repository; it was considered on 2026-09-03 and not
taken, because it makes a new public contract.

## What this file used to claim, and why the correction is here

Until 2026-09-03 this file said the authoritative copy was that centralised
skill — and **the skill did not exist**. `am_load_skill("mrw")` returned
`skill: not found`, and `am_list_skills` returned nine skills, none of them mrw.
The pointer had been dangling for an unknown period, because nobody followed it.

It was then created, on the same day, from the AGENTS.md text. Both halves of
that story are kept because the lesson is not "the skill was missing" but **a
pointer reads exactly like a working one until someone follows it** — so follow
one when you write it, and again when you cite it.

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
