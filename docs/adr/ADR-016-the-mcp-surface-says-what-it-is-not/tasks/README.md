# ADR-016 Tasks

Implementation tasks for ADR-016: the MCP surface says what it is not. See the parent ADR for the
decision, the direction it routes callers in, and — importantly — the parity it deliberately does
not claim.

**Source of truth:** the task file's headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

One task. Nothing here is sequenced, which is why this index exists only for the reason the corpus
asks for it: `adr-lint` reports a tasks directory without an index as advice, and this record shipped
without one.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Route to the fuller surface, and keep the claim true | done | — | 1 named `--- PASS:` line, `# 50.` in `contract.sh`, gofmt + vet, `./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the routing text and its test | — | none — single task |

## Notes

- **§50, not §49.** 48 was the highest on `main` when this branch opened and ADR-015 held 49 on #77,
  so T1 took the next free number rather than colliding on merge. That is the same choice that made
  #74 and #76 merge textually instead of by renumbering, and it is worth repeating: reserving the
  number up front turns a merge conflict into no event at all.

- **The routing advice must be true of the binary, not merely present.** T1's test checks each flag
  against the help of the SUBCOMMAND the advice binds it to. The first cut recommended `-C` for
  choosing a checkout, which after `read` is the integer context flag — so the advice ERRORED when
  followed. A concatenated help blob would have absorbed that; per-subcommand checking is what
  catches it.

- **"The CLI is strictly fuller" was false and the record says so.** MCP always returns
  `structuredContent`, and one server is one writer to the read-before-write ledger while parallel
  CLI processes race for it (ADR-010:42,185). The routing states a direction with a counterweight
  rather than a ranking.

- **The parity refusal is scoped to this record.** ADR-016 covers only what the surface SAYS. Whether
  the MCP surface should GROW is a different question, it is open, and `docs/adr/BACKLOG.md` carries
  it as the largest open product question in the repository. Do not read this record's Out of Scope
  as having settled it — that line is deferred, not permanent, and says so.
