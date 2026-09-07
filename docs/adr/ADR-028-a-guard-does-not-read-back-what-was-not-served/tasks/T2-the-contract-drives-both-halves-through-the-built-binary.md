# Task ADR-028-T2: The contract drives both halves through the built binary

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (one contract section)
**Owner:** Zy
**Produces:** contract §65
**Consumes:** the anchor check evaluated only for lines the ledger licensed (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built binary printing no unserved line on a failed anchor`, `the built binary still quoting a served line`

## Goal

Prove in the BUILT binary that a failed anchor on an unserved line prints none of it, and that a
failed anchor on a served line still quotes it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 65.` section. A unit test proves the ordering inside the package; it cannot prove the shipped binary has it, which is why §53 exists. This is also the only place the two halves are asserted against one binary in one run. |

## Ordered Steps

1. [S1] Confirm §65 does not exist and that the row is RED against a binary built before T1: the leaked line appears in the refusal on the pre-ADR-028 tree. A row that cannot fail asserts nothing. [proof: acceptance]
2. [S2] Write §65 with both halves and a distinctive sentinel: serve line 1, anchor line 2 with a wrong value, and assert the sentinel text of line 2 is ABSENT from the output while the ledger's own refusal is present. [proof: acceptance]
3. [S3] Assert the other half in the same section: with the line served, a failed anchor still quotes it. Without this the row would pass against a binary that had stopped checking anchors. [proof: acceptance]
4. [S4] Run `./scripts/contract.sh` whole, and every other gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 65\. ' scripts/contract.sh \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr028-t2.out \
  && grep -q '^contract holds$' /tmp/adr028-t2.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr028-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§65` | `scripts/contract.sh` | The built binary refuses an anchored hunk on an unserved line without printing any of that line, and still quotes a served line whose anchor failed | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `§65` in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section against `$MRW`, the binary built at the top of the script, so the row fails if the shipped ordering is wrong even when the unit test passes |
| 3 — the caller can discover it | The refusal itself; no documented surface changes |
| 4 — it is used | The contract runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · bd73ee0* · mutant killed · exit 1 · `internal/apply/apply.go` · the built binary reads back the unserved line again, so §65 sees the sentinel in the refusal — the row drives the shipped binary, which the unit test cannot · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · covers:the built binary printing no unserved line on a failed anchor
- 2026-09-07 · bd73ee0* · mutant killed · exit 1 · `internal/apply/apply.go` · the built binary stops checking anchors, so the served-line half of §65 no longer sees the line quoted and the row goes red rather than crediting a removal as a fix · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · covers:the built binary still quoting a served line

## Invariants

- Every existing contract section still passes; §65 uses its own fixture.
- The row drives `$MRW`, never a Go test.
- Both halves live in one section, because a row asserting only the absence would pass against a binary that printed nothing at all.

## Risks

- A sentinel that appears elsewhere in the output would make the absence assertion vacuous. Mitigated by using a string that appears nowhere else in the fixture or in mrw's own vocabulary.

## Stop Condition

Stop and ask if §65 cannot be made red against the pre-ADR-028 tree: a row green before the fix
exists is asserting nothing, and finding that out here is the point of S1.

## Out of Scope

- The ordering itself — that is T1's job

## Verification Log
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · ms:32344
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · ms:29195
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · ms:31276
