# ADR-014 Tasks

Implementation tasks for ADR-014: a read too large is a first page, not a dead end. See the parent
ADR for the decision, the RSS measurement behind it, and — importantly — the question it refuses to
answer.

**Source of truth:** the task files' headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

Forced. T2 teaches what T1 builds, and this corpus has now shipped two records that taught something
the binary did not do: ADR-012's `hunks.status` enum, caught by two independent reviewers, and
ADR-013's two examples that could not match anything. Both were prose written beside behaviour
rather than against it.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Serve the first page, and say how to continue | done | — | 3 named `--- PASS:` lines, `# 47.` in `contract.sh`, gofmt + vet, both engine clauses, `./scripts/contract.sh` |
| T2 | Teach the continuation, once it exists | done | — | 1 named `--- PASS:` line, `# 48.` in `contract.sh`, gofmt + vet, `./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done` | `withdrawn`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the first-page result and its continuation field | T2 | T1 before T2 — teach only what exists |

## Notes

- **§47 and §48, not §45 and §46.** The highest section on `main` is 44, but ADR-013 takes 45 and 46
  on an open branch (#74). Taking them here would produce a merge conflict rather than a gate
  failure, which is the worse of the two because nothing would report it until both landed.
- **This ADR touches only `internal/mcp`**, so both engine go/no-go clauses stay in T1's fence and
  must pass. That is the opposite of ADR-013, whose branch turns them red by design.
- **T2's `grep -q 'continue' README.md` clause is weak and is labelled as weak in its own fence.** It
  returns a hit today, from an unrelated sentence. It is kept rather than replaced because a
  fence that hides its weakest clause is worse than one that names it; §48 is what actually binds.
- **The Enforced-by must page to exhaustion and compare a reassembly**, not check that a field
  exists. ADR-012 shipped a surviving mutant from a fixture that agreed with the plan by
  construction; a continuation checked only for presence is the same shape of non-test.
