# Task ADR-030-T1: The boundary refuses the enumerated list

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (seven rules at one boundary, and the table test that keeps the two sites honest)
**Owner:** Zy
**Produces:** the enumerated engine-boundary refusals
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the engine refusing every enumerated shape the parser refuses`, `the engine still applying every shape the parser accepts`

## Goal

Make `Apply`'s "validates every hunk" true, for the list that came from enumeration rather than from
memory, without importing `internal/plan`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | The boundary block that already refuses a relative end (ADR-026) and a body-less `create` (ADR-027) gains the remaining rules, with the messages copied verbatim from `plan.validate` so the two sites cannot say different things about one mistake. |
| `internal/apply/apply_test.go` | edit | `TestTheEngineRefusesEveryShapeTheParserRefuses` is a TABLE, one row per rule, so a rule added to `validate` with no engine counterpart is a missing row rather than silence. |

## Ordered Steps

1. [S1] Write the table test with one row per enumerated rule and confirm it is RED for seven of the eight rows — the eighth, `replace` with `Start: 0`, is already refused and is in the table as a control that must stay green. [proof: mutation]
2. [S2] Add the rules to the existing boundary block, copying each message verbatim from `plan.validate`. [proof: mutation]
3. [S3] Assert the ACCEPTING half in the same test: an ordinary `replace`, `create`, `delete` and both insertions still apply. Without it, "refuse everything" passes. [proof: mutation]
4. [S4] Assert `Failed` AND the file's bytes for every refused row. A refusal that still wrote is worse than the behaviour being removed, and `replace` with an empty body is the row where that matters — it deleted lines while reporting `ok`. [proof: acceptance]
5. [S5] Name the rules that are deliberately NOT mirrored, in the test, with the reason: `patterned` is a parse fact and the engine has resolved the address by the time it could ask. A silent omission is how the next enumeration goes wrong. [proof: human: read the test's absent-rules comment against `plan.validate` and confirm every unmirrored branch is named there]
6. [S6] Run every gate, including `go test -race ./...` and `./scripts/contract.sh` — no contract row is added, so the contract must be unchanged and still green. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -count=1 -v \
  -run 'TestTheEngineRefusesEveryShapeTheParserRefuses' 2>&1 | tee /tmp/adr030-t1.out \
  && grep -q '^--- PASS: TestTheEngineRefusesEveryShapeTheParserRefuses' /tmp/adr030-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr030-t1.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr030-t1c.out \
  && grep -q '^contract holds$' /tmp/adr030-t1c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheEngineRefusesEveryShapeTheParserRefuses` | `internal/apply/apply_test.go` | Every enumerated shape is refused by a direct `Apply` with `Failed: 1` and the file byte-identical; every well-formed shape still applies; the unmirrored parser-only rules are named | — | S1, S2, S3, S4, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestTheEngineRefusesEveryShapeTheParserRefuses` |
| 2 — something selects it | The boundary block runs for every hunk `Apply` receives, whatever built it; the S2 mutation removes a clause and the fence goes red on the row that names it |
| 3 — the caller can discover it | The refusal wording is `plan.validate`'s, already documented; no new surface |
| 4 — it is used | ⚠ No contract row, and that is deliberate: `scripts/contract.sh` drives the built binary, which cannot reach `Apply` without the parser, so a row would prove the PARSER's refusal and credit it here. The Go test is the whole of rung 4, as it was for ADR-027-T3. No telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · d26e39a* · mutant killed · exit 1 · `internal/apply/apply.go` · the engine stops refusing a replace carrying no body, so a direct Apply caller DELETES the addressed lines and gets a receipt saying ok — the shape plan.validate calls the failure this whole format exists to refuse · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · covers:the engine refusing every enumerated shape the parser refuses
- 2026-09-07 · d26e39a* · mutant killed · exit 1 · `internal/apply/apply.go` · an insertion stops refusing a RANGE address, so it silently uses the start and ignores the end the caller wrote — an address half-ignored, reported ok · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · covers:the engine refusing every enumerated shape the parser refuses
- 2026-09-07 · d26e39a* · mutant killed · exit 1 · `internal/apply/apply.go` · the insertion rule widens to refuse every insertion, which a table of refusals alone would call success — the accepting half is what refuses it, and it is why the fix is a narrowing rather than a ban · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · covers:the engine still applying every shape the parser accepts

## Invariants
- `internal/apply` does not import `internal/plan`. That inversion is ADR-027-T3's Stop Condition and is the reason the duplication is accepted.
- The messages are `plan.validate`'s, verbatim. Two sites saying different things about one mistake is worse than two sites.
- Every shape the parser ACCEPTS still applies at the engine — asserted in the same test, because "refuse everything" would otherwise pass.
- `internal/read`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- ⚠ The enumeration could be incomplete, which is the exact mistake this record was written to stop repeating. It came from driving `Apply` with one `Input` per `validate` rule and recording the verdict; the table carries the same list so an eighth rule is a missing row.
- A refused row that still wrote would be worse than the behaviour removed. S4 asserts the file's bytes, not only the verdict.

## Stop Condition

Stop and ask if any rule cannot be mirrored without importing `internal/plan` or without the parse
tree — that is the boundary this record accepts, and a rule that crosses it needs naming in the
record rather than a workaround in the code.

## Out of Scope

- Typed error kinds shared by the two sites (deferred: `docs/adr/BACKLOG.md`)
- Any change to `internal/plan` (permanent: boundary: the parser is not what is wrong here)

## Verification Log
- 2026-09-07 · d26e39a* · exit 0 · `set -o pipefail …` · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · ms:30135
- 2026-09-07 · d26e39a* · exit 0 · `set -o pipefail …` · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · ms:30034
- 2026-09-07 · d26e39a* · exit 0 · `set -o pipefail …` · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · ms:29881
- 2026-09-07 · d26e39a* · exit 0 · `set -o pipefail …` · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · ms:30332
