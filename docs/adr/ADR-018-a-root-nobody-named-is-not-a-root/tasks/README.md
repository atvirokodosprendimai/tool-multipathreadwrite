# ADR-018 Tasks

Implementation tasks for ADR-018: a root nobody named is not a root. See the parent ADR for the
decision, the reframing of issue #81 that produced it, and the alternative M did not take.

**Source of truth:** the task file's headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

One task, so nothing is sequenced. The index exists because `adr-lint` reports a tasks directory
without one, and ADR-016 shipped without it.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Refuse the root nobody named | done | — | 1 named `--- PASS:` line, `# 53.` in `contract.sh`, gofmt + vet, both engine clauses, `./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the fallback-root guard and its refusal text | — | none — single task |

## Notes

- **⚠ THE NUMBER MOVED, AND EARLIER TEXT IN THIS CORPUS IS WRONG ABOUT IT.** ADR-017 and its tasks
  say "ADR-018 owns reach". They were written before this record existed. Reach — multi-root, the
  per-hunk ledger, repeatable `--root` — is **ADR-019** and is still unwritten; its design is carried
  in `docs/adr/BACKLOG.md`. This record took 018 because it is small and shippable, and making a
  safety guard wait on a larger design is backwards.

- **§53, not §51.** 50 is the highest on `main`; ADR-017 holds 51 and 52 on #80. Reserving the next
  free number turns a merge into a textual one rather than a renumbering.

- **⚠ A UNIT TEST CANNOT PROVE THIS ONE.** `CheckRoot` can be perfectly correct and called by
  nothing, and every Go test would stay green while the binary went on serving `/`. §53 runs the
  BUILT BINARY from `/` and reads its exit status. That is why wiring is a numbered step (T1 S6)
  rather than an implementation detail — this corpus has already shipped a gate that passed because
  the thing it checked was never reached.

- **The guard is on the SOURCE, not on the path**, and that is the whole design. `ResolveRoot`
  already returns `SourceFlag` / `SourceProjectDir` / `SourceWorkingDir`, so "did anyone say this?"
  is a value that exists rather than an inference. A path-based rule would be a list of directories
  somebody found distasteful, and it would refuse the home directory — where a Desktop analyst's
  documents actually live, which is the population ADR-017 was written for.

- **Explicitness is the licence, and it is asserted in both directions.** `--root /` must still
  start. If that ever regresses, the guard has quietly become a blacklist.
