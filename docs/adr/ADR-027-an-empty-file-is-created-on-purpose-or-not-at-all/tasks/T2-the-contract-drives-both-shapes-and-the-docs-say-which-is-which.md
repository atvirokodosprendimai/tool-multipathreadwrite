# Task ADR-027-T2: The contract drives both shapes, and the docs say which is which

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (a contract section and two documents)
**Owner:** Zy
**Produces:** contract §65, and the documented spelling for a deliberate empty file
**Consumes:** the body-less `create` refusal and its wording (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built binary refusing a body-less create`, `the built binary creating an empty file for body=0`

## Goal

Prove both shapes in the BUILT binary — the refusal and the deliberate empty file — and write the
spelling where a caller reads it, so `body=0` is discoverable rather than folklore.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 65.` section. A unit test proves `validate`; it cannot prove the binary calls it, which is why §53 exists. The row pairs the refusal with the case that must still succeed — a row asserting only the refusal would pass against a tool that had stopped creating files at all. |
| `README.md` | edit | The `create` description says what an empty file now costs: `body=0`. A caller who meets the refusal without this reads it as a bug. |
| `AGENTS.md` | edit | Section 2's op list carries the same, for the agents that read it instead of the README. |

## Ordered Steps

1. [S1] Confirm §65 does not exist and that the row is RED against a binary built before T1: `grep -c '^# 65\. ' scripts/contract.sh` is 0 first, and the section fails against the pre-ADR-027 tree. A contract row that cannot fail asserts nothing. [proof: acceptance]
2. [S2] Write §65 pairing the two: `@@ new.txt 0 create` with no body is refused naming `body=0` and creates NO file, while `@@ new.txt 0 create body=0` exits 0 and leaves a zero-byte file that exists. Asserting the file's absence matters — a refusal that still created the file would be worse than the behaviour being removed. [proof: acceptance]
3. [S3] Assert the neighbours are untouched in the same section: an ordinary `create` with a body still applies, and an empty-bodied `replace` is still refused with its own wording. [proof: acceptance]
4. [S4] Document the spelling in `README.md` and `AGENTS.md`. [proof: human: read both passages against §65's fixture and confirm each names `body=0` as the way to create an empty file, and that neither claims mrw refuses empty files]
5. [S5] Run every gate, including `go test -race ./...`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 65\. ' scripts/contract.sh \
  && grep -q 'body=0' README.md \
  && grep -q 'body=0' AGENTS.md \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr027-t2.out \
  && grep -q '^contract holds$' /tmp/adr027-t2.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr027-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§65` | `scripts/contract.sh` | The built binary refuses a body-less `create` and creates no file, applies `create body=0` leaving a zero-byte file, still applies an ordinary `create`, and still refuses an empty-bodied `replace` | — | S1, S2, S3 |
| `TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero` | `internal/plan/plan_test.go` | Unchanged from T1: the row and the unit test must agree, and the row is what proves the binary reaches it | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §65 in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section against `$MRW`, the binary built at the top of the script, so the row fails if the wiring is absent even when the unit test passes |
| 3 — the caller can discover it | `README.md` and `AGENTS.md`, checked in S4 |
| 4 — it is used | The contract runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

## Invariants

- Every existing contract section still passes; §65 uses its own fixture and touches no other section's `$R`.
- The row drives `$MRW`, never a Go test.
- The refusal must leave NO file behind — ADR-004's rule, and the one way this change could be worse than what it replaces.

## Risks

- A row asserting only the refusal would pass against a tool that stopped creating files entirely. Mitigated by S2 requiring the `body=0` success and the file's existence in the same section.
- The documentation could describe the refusal without naming the remedy, which is the failure ADR-015 exists to prevent. Mitigated by S4's human proof reading both passages against the fixture.

## Stop Condition

Stop and ask if §65 cannot be made red against the pre-ADR-027 tree: a row that is green before the
feature exists is asserting nothing, and finding that out here is the point of S1.

## Out of Scope

- The validation itself — that is T1's job
- Applying the same rule to any other op (permanent: fact: `replace`, `insert-after` and `insert-before` already refuse an empty body, so `create` was the last one; citation: file `internal/plan/plan.go:617`)

## Verification Log
