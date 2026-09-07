# Task ADR-030-T1: The boundary refuses the enumerated list

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (ten  branches at one boundary, a table test, and the cross-site test that makes the drift claim true)
**Owner:** Zy
**Produces:** the enumerated engine-boundary refusals
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the engine refusing every enumerated shape the parser refuses`, `the engine still applying every shape the parser accepts`, `the two sites refusing in the same words`

## Goal

Make `Apply`'s "validates every hunk" true, for the list that came from enumeration rather than from
memory, without importing `internal/plan`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | The boundary block that already refuses a relative end (ADR-026) and a body-less `create` (ADR-027) gains the remaining rules, with the messages copied verbatim from `plan.validate` so the two sites cannot say different things about one mistake. |
| `internal/apply/apply_test.go` | edit | `TestTheEngineRefusesEveryShapeTheParserRefuses` is a TABLE, one row per rule, so a rule added to `validate` with no engine counterpart is a missing row rather than silence. |

## Ordered Steps

1. [S1] Write the table test with one row per enumerated rule and confirm it is RED for seven of the eight rows first probed — the eighth, `replace` with `Start: 0`, is already refused and is in the table as a control that must stay green. [proof: mutation]
2. [S2] Add the rules to the existing boundary block, copying each message verbatim from `plan.validate`. [proof: mutation]
3. [S3] Assert the ACCEPTING half in the same test: an ordinary `replace`, `create`, `delete` and both insertions still apply. Without it, "refuse everything" passes. [proof: mutation]
4. [S4] Assert `Failed` AND the file's bytes for every refused row. A refusal that still wrote is worse than the behaviour being removed, and `replace` with an empty body is the row where that matters — it deleted lines while reporting `ok`. [proof: acceptance]
5. [S5] ⚠ **Walk `plan.validate` BRANCH BY BRANCH, not shape by shape**, and give every branch a row. The first pass probed the shapes that came to mind, found seven, and wrote that the list was complete; the review of PR #130 walked the branches and found two more — an insertion with a pattern RANGE, never named, and a `create` with a pattern address, named absent on a rationale that was false, since `Input` carries `StartPat` and the boundary runs before resolution. Both were then confirmed accepted by driving `Apply`. [proof: mutation]
6. [S6] Make the drift claim TRUE rather than asserting it. `TestTheEngineAndTheParserRefuseInTheSameWords` parses each malformed plan, takes the expected text from the PARSER's own error at run time, and compares it to what `Apply` says for the equivalent `Input` — so rewording either site alone goes red. The table test cannot do this: its strings are hardcoded and it never invokes the parser, which is exactly what the review found. [proof: mutation]
7. [S7] Run every gate, including `go test -race ./...` and `./scripts/contract.sh` — no contract row is added, so the contract must be unchanged and still green. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -count=1 -v \
  -run 'TestTheEngineRefusesEveryShapeTheParserRefuses' 2>&1 | tee /tmp/adr030-t1.out \
  && grep -q '^--- PASS: TestTheEngineRefusesEveryShapeTheParserRefuses' /tmp/adr030-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr030-t1.out \
  && go test ./internal/adversarial/ -count=1 -v \
       -run 'TestTheEngineAndTheParserRefuseInTheSameWords' 2>&1 | tee /tmp/adr030-t1x.out \
  && grep -q '^--- PASS: TestTheEngineAndTheParserRefuseInTheSameWords' /tmp/adr030-t1x.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr030-t1x.out \
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
| `TestTheEngineRefusesEveryShapeTheParserRefuses` | `internal/apply/apply_test.go` | Every `plan.validate` branch is refused by a direct `Apply` with `Failed: 1` and the file byte-identical; every well-formed shape still applies AND leaves the exact bytes it should, so a no-op reporting success cannot pass | — | S1, S2, S3, S4, S5 |
| `TestTheEngineAndTheParserRefuseInTheSameWords` | `internal/adversarial/planformat_test.go` | For nine of the ten verbatim-mirrored branches — the tenth being unreachable from a plan document, and said so in the test — the parser's message and the engine's `Reason` are EQUAL, the expected text taken from the parser at run time, so rewording either site alone goes red | — | S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestTheEngineRefusesEveryShapeTheParserRefuses` |
| 2 — something selects it | The boundary block runs for every hunk `Apply` receives, whatever built it; the S2 mutation removes a clause and the fence goes red on the row that names it |
| 3 — the caller can discover it | The refusal wording is `plan.validate`'s, already documented; no new surface |
| 4 — it is used | ⚠ No contract row, and that is deliberate: `scripts/contract.sh` drives the built binary, which cannot reach `Apply` without the parser, so a row would prove the PARSER's refusal and credit it here. The Go test is the whole of rung 4, as it was for ADR-027-T3. No telemetry, per ADR-009 |

## Mutation Log

⚠ **The `051880f` entry reading "the pattern-range refusal for insertions is reworded … the
cross-site test is what now sees the divergence" CLAIMED THE WRONG KILLER.** At that commit the
cross-site test had no pattern-range row, so the fence went red on the apply-side table, whose
expected string is hardcoded — which is engine-side drift, not the parser-side drift the entry
describes. The review of PR #130 found it. The entry is kept as it ran and superseded by the
`internal/plan/plan.go` mutant below, which rewords the PARSER and leaves the engine alone: that is
the direction only the cross-site test can see, and it now has the row to see it with.

- 2026-09-07 · d26e39a* · mutant killed · exit 1 · `internal/apply/apply.go` · the engine stops refusing a replace carrying no body, so a direct Apply caller DELETES the addressed lines and gets a receipt saying ok — the shape plan.validate calls the failure this whole format exists to refuse · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · covers:the engine refusing every enumerated shape the parser refuses
- 2026-09-07 · d26e39a* · mutant killed · exit 1 · `internal/apply/apply.go` · an insertion stops refusing a RANGE address, so it silently uses the start and ignores the end the caller wrote — an address half-ignored, reported ok · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · covers:the engine refusing every enumerated shape the parser refuses
- 2026-09-07 · d26e39a* · mutant killed · exit 1 · `internal/apply/apply.go` · the insertion rule widens to refuse every insertion, which a table of refusals alone would call success — the accepting half is what refuses it, and it is why the fix is a narrowing rather than a ban · acceptance-sha256:42aebec5aa67d55597a759fd7cd72f0a7a41b4660ba4bde949061f0901b4fc7b · covers:the engine still applying every shape the parser accepts
- 2026-09-07 · ee97f6b* · mutant killed · exit 1 · `internal/apply/apply.go` · the engine stops refusing a create with a PATTERN address — the branch the first cut named deliberately absent on a rationale that was false, and which a direct caller had accepted · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · covers:the engine refusing every enumerated shape the parser refuses
- 2026-09-07 · ee97f6b* · mutant killed · exit 1 · `internal/apply/apply.go` · one engine message is reworded and the parser is left alone — the drift the first cut claimed was mitigated by copying strings verbatim, which the hardcoded table test could not see · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · covers:the two sites refusing in the same words
- 2026-09-07 · ee97f6b* · mutant killed · exit 1 · `internal/apply/apply.go` · the pattern-range refusal for insertions is reworded away from plan.validate wording — it is the branch the first cut never named, and the cross-site test is what now sees the divergence · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · covers:the two sites refusing in the same words
- 2026-09-07 · ee97f6b* · mutant killed · exit 1 · `internal/apply/apply.go` · the guard itself is removed for an insertion with a PATTERN range, so the hunk resolves and then silently uses the start and ignores the end the caller wrote — the branch the first pass never named, reported ok · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · covers:the engine refusing every enumerated shape the parser refuses
- 2026-09-07 · 051880f* · mutant killed · exit 1 · `internal/plan/plan.go` · the PARSER is reworded and the engine left alone, on the pattern-range branch the first pass never named — this is the direction the hardcoded apply-side table cannot see, and the direction T1 previously claimed evidence for without having it · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · covers:the two sites refusing in the same words

## Invariants
- `internal/apply` does not import `internal/plan`. That inversion is ADR-027-T3's Stop Condition and is the reason the duplication is accepted.
- The messages are `plan.validate`'s, verbatim. Two sites saying different things about one mistake is worse than two sites.
- Every shape the parser ACCEPTS still applies at the engine — asserted in the same test, because "refuse everything" would otherwise pass.
- `internal/read`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- ⚠ The enumeration WAS incomplete, which is the exact mistake this record was written to stop repeating, made inside it. Probing shapes is not walking branches; two branches were missed and the review of PR #130 found them. The list now comes from a branch-by-branch walk, and the table carries every one so an eleventh is a missing row.
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
- 2026-09-07 · ee97f6b* · exit 0 · `set -o pipefail …` · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · ms:34176
- 2026-09-07 · ee97f6b* · exit 0 · `set -o pipefail …` · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · ms:29881
- 2026-09-07 · ee97f6b* · exit 0 · `set -o pipefail …` · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · ms:31282
- 2026-09-07 · ee97f6b* · exit 0 · `set -o pipefail …` · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · ms:30193
- 2026-09-07 · ee97f6b* · exit 0 · `set -o pipefail …` · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · ms:30133
- 2026-09-07 · 051880f* · exit 0 · `set -o pipefail …` · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · ms:34952
- 2026-09-07 · 051880f* · exit 0 · `set -o pipefail …` · acceptance-sha256:4419231c12d93d17907c57dc8737b405e52dc0b08221047bb7a384af293045c6 · ms:38468
