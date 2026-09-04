# ADR-017 Tasks

Implementation tasks for ADR-017: the MCP surface can find what it serves. See the parent ADR for
the decision, the walk measurement behind it, and the parity it refuses on the tool's own documented
grounds.

**Source of truth:** the task files' headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

Forced. T2 teaches what T1 builds, and this corpus has now shipped three records that taught
something the binary did not do: ADR-012's `hunks.status` enum, ADR-013's two unmatchable examples,
and ADR-016's `-C` advice that ERRORED when followed. All three were prose written beside behaviour
rather than against it.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Find over MCP, and degrade to the index rather than to a dead end | done | — | 4 named `--- PASS:` lines, `# 51.` in `contract.sh`, gofmt + vet, both engine clauses, `./scripts/contract.sh` |
| T2 | Teach finding, and repair ADR-016's sentence in the same commit | done | — | 2 named `--- PASS:` lines, `# 52.` in `contract.sh`, #73's gate, gofmt + vet, `./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | grep over MCP, its index degradation and its refusals | T2 | T1 before T2 — teach only what exists |

## Notes

- **§51 and §52, and the section numbers are NOT in file order.** 50 is the highest on `main` after
  #78. Reading the last `# N.` in `contract.sh` returns 49, because the sections are not sorted in
  the file; the free number was found by extracting all of them and sorting numerically. Anyone
  reserving the next one should do the same rather than reading the tail.

- **⚠ §50 CANNOT protect T2's claim, and T2 says so in its own Acceptance.** §50 asserts that every
  flag the routing names exists in the CLI's help — which stays true of `--grep` forever, including
  after MCP gains it. Nothing existing can notice that a flag advertised as CLI-ONLY has appeared on
  the other surface. `TestTheRoutingClaimsOnlyRealExclusives` is what closes that, by asserting the
  MCP input schema declares no argument named by the exclusivity list. This is the same shape as the
  finding already recorded here: a gate that checks a thing EXISTS cannot catch a thing that is
  present and false.

- **T1 keeps both engine go/no-go clauses and they must PASS.** This record calls `read.Walk`; it
  does not change it. A red engine clause on this branch means the design slipped from calling the
  primitive into changing it, which is T1's Stop Condition — not a wrong-branch artifact, as it was
  for ADR-013.

- **The Enforced-by must drive a real overflow.** ADR-012 shipped a surviving mutant from a fixture
  that agreed with the plan by construction; an index checked only for presence is the same shape of
  non-test, and it would pass over an index naming the wrong files.

- **The index-overflow case (T1 S4) is the step most likely to be dropped**, because the
  content-overflow case is the satisfying one. That pattern — the step after the satisfying part —
  is documented across ADR-013, -014 and -015, each of which deferred something to BACKLOG.md
  without writing the entry.
