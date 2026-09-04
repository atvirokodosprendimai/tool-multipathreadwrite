# ADR-013 Tasks

Implementation tasks for ADR-013: a plan addresses what it can find, and refuses what it finds twice.
See the parent ADR for the decision, the measurement behind it, and the scope limit it puts on that
measurement.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers. This
README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |

Forced, not chosen. T2 resolves what T1 parses, so a resolver written first would be tested against a
grammar no record had accepted. T3 is last because ADR-012's whole finding is that a surface teaching
a format the binary does not have is worse than a surface teaching nothing — and the near-miss it
shipped, an enum the engine never sent, is exactly what teaching-before-implementing produces.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The grammar accepts a pattern, and changes nothing else | done | — | 3 named `--- PASS:` lines, gofmt + vet, `./scripts/contract.sh` |
| T2 | Resolve it exactly once, or refuse it | done | — | 4 named `--- PASS:` lines, `# 45.` in `contract.sh`, gofmt + vet, `./scripts/contract.sh` |
| T3 | Teach the form, now that it exists | done | — | 2 named `--- PASS:` lines, `# 46.` in `contract.sh`, gofmt + vet, `./scripts/contract.sh` |

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `plan.Addr` carrying a compiled pattern | T2 | T1 before T2 — resolution needs something to resolve |
| T2 | resolution and the ambiguity refusal | T3 | T2 before T3 — teach only what the binary does |

## Notes

- **`gofmt -l .` and `go vet ./...` lead every fence here, and that is new.** ADR-012's fences ran
  neither; CI's Format step runs `gofmt`, and a formatting failure reached CI twice in one day
  because the fence a task called "done" was not the gate CI applied. ADR-012 carries a follow-up to
  retrofit its own fences; this record does not repeat the mistake.
- **§45 and §46 were confirmed free.** The highest section in `contract.sh` is 44, from ADR-012-T2.
- **The Enforced-by fixture must be authored independently of the plan it tests.** ADR-012 shipped a
  surviving mutant because `treeFor` plants a fixture from the plan, so an `anchor=` guard in an
  example could never fail. A two-match test whose file is generated from the pattern would be the
  identical defect. T2's Acceptance says so in its own words rather than relying on this note.
- **The parent ADR's measurement has a stated limit, and the tasks inherit it.** One round trip and
  53% of bytes saved, on the class where new content does not depend on current text. For a
  content-dependent edit the read happens anyway and a pattern saves only the arithmetic. Nothing
  here should be described as making plans cheaper — plans get 14% bigger.
