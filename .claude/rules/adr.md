---
paths:
  - "docs/adr/**"
---

# Records and tasks: what this corpus checks beyond the template

The template comes from `/quality-harness:adr-write`; `adr-lint` is the reader. This file is what
its advice lines have actually caught here. Where it quotes the checker's vocabulary it quotes
version 2.64.0; when the checker disagrees with this file, the checker is right and this file is stale.

## Headers that must point at something real

- **`Enforced-by`** names `path/to/file_test.go::TestName` and that test must EXIST when the record
  merges. A pointer to nothing is the rot the header exists to prevent; `adr-lint` says so as advice
  and advice is a finding.
- **`Governs`** globs must match tracked files, or `adr-context` answers "none governs" for the code
  the record was written about.
- **`Served-path change`** says in one sentence what a caller sees differently, or `none` and why.
- **`Accepted`** names who accepted it and quotes what they said. The quote is the provenance.

## Task files

- Status words the reader acts on are its own `KNOWN_TASK_STATUS` — `pending`, `partial`,
  `blocked`, `done` at 2.64.0, the legend the task READMEs here carry. Anything else — `in flight`
  — is not looked at, which is not the same as passing.
- Every step carries `[proof: acceptance]` or `[proof: mutation]`; every mutation step gets a
  Mutation Log line with a killed mutant, the file, the step, and an `acceptance-sha256` — the
  sha256 of the normalised fence, which `adr-verify` writes and `adr-lint` recomputes for drift.
- Reachability has four rungs. Rung 4 — "it is used" — is answered honestly: telemetry is refused
  by ADR-009, so say what the evidence for building it was rather than claiming adoption.
- The Acceptance fence's engine go/no-go clauses stay in and must pass; if a record needs them
  removed, the design has slipped into the engine and the Stop Condition applies.

## Out of Scope grammar

Each item ends with a disposition the checker accepts: `(deferred: <where it lives>)`,
`(permanent: boundary: <why>)`, `(permanent: fact: <what>; citation: <where>)` — ADR-007 and
ADR-011 use it — or `(external: <who owns it>)`. A deferral naming `docs/adr/BACKLOG.md` has its
entry there in the same commit — a deferral without a receipt is the habit `lifecycle.md` warns
about. BACKLOG.md also holds pre-registrations: a criterion authored after the first look is not a
criterion, so it is written there BEFORE the harness exists.

## Numbers

Reserve by grepping the corpus for the number, not by listing the directory: prose can reserve a
number before its directory exists (ADR-019, reach). Contract sections likewise — see `contract.md`.
