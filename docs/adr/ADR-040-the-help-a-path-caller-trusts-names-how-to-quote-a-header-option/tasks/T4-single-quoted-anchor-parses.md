# Task ADR-040-T4: Single-quoted `anchor=` parses

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** single-quoted `anchor=` parse (T4)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `single-quoted anchor parsing`, `double-quoted anchor still working`

## Goal

`anchor='func openTestStore'` is a successful parse whose Anchor is `func openTestStore`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `splitHeader` — single quotes. The selector. |
| `internal/plan/plan_test.go` | edit | `TestASingleQuotedAnchorParses` |
| `scripts/contract.sh` | edit | §80 drives the built binary |

## Ordered Steps

1. [S1] Write `TestASingleQuotedAnchorParses` and confirm it is RED. [proof: mutation]
2. [S2] Toggle `'` in `splitHeader` the way `"` already toggles. [proof: mutation]
3. [S3] Drive the built binary via §80. [proof: mutation]
4. [S4] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ -count=1 -v \
  -run 'TestASingleQuotedAnchorParses' 2>&1 | tee /tmp/adr040-t4.out \
  && grep -q '^--- PASS: TestASingleQuotedAnchorParses' /tmp/adr040-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr040-t4.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/plan/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASingleQuotedAnchorParses` | `internal/plan/plan_test.go` | `anchor='func openTestStore'` parses; Anchor is `func openTestStore` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestASingleQuotedAnchorParses` |
| 2 — something selects it | `splitHeader` `'` toggle; reverting it fails S1 |
| 3 — the caller can discover it | `write --help` names single-quoted |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 4b70dd0* · mutant killed · exit 1 · `internal/plan/plan.go` · without toggling single quotes the T4 fixture must fail · acceptance-sha256:5754d8f66a4bb9977a11bc13619d95d3aad3375c3a40d124600cfc64706d0fb6
- 2026-09-12 · 4b70dd0* · mutant killed · exit 1 · `internal/plan/plan.go` · without toggling single quotes the T4 fixture must fail · acceptance-sha256:5754d8f66a4bb9977a11bc13619d95d3aad3375c3a40d124600cfc64706d0fb6 · covers:single-quoted anchor parsing

## Invariants

- Double-quoted `anchor=` still works.
- A multi-line replace still requires `anchor=` (ADR-035).

## Risks

- `'` inside a pattern toggles. Mitigation: the toggle is `!inPat && !inQ`.

## Stop Condition

Stop if single quotes need an escape other than the existing double-quote `\"` rule.

## Out of Scope

- Consume-to-next-key (that's T2)
- `mrw version` (that's T3)

## Verification Log
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:5754d8f66a4bb9977a11bc13619d95d3aad3375c3a40d124600cfc64706d0fb6 · ms:367
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:5754d8f66a4bb9977a11bc13619d95d3aad3375c3a40d124600cfc64706d0fb6 · ms:363
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:5754d8f66a4bb9977a11bc13619d95d3aad3375c3a40d124600cfc64706d0fb6 · ms:455
- 2026-09-12 · 4b70dd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:5754d8f66a4bb9977a11bc13619d95d3aad3375c3a40d124600cfc64706d0fb6 · ms:407
