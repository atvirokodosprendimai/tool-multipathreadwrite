# Task ADR-035-T1: A multi-line replace is refused without an anchor

**Depends-on:** none
**Covers:** none — no spec
**Owner:** Zy
**Produces:** the refusal in `apply.Apply`, and its message naming `anchor=`
**Consumes:** none
**Proof map:** v1

**Rests-on:** `a multi-line replace without an anchor being refused`, `a single-line replace still needing no anchor`, `a pattern range spanning many lines being refused on the same terms`, `the refusal naming the remedy`

## Goal

Make a `replace` whose address resolves to more than one line fail unless it
carries `anchor=`, on every address form, for every caller of `apply.Apply`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | The guard, at the resolved-range site — after the out-of-range and ends-before-starts checks, which name their own reasons, and after `covered()`, for the reason ADR-028 gives about the anchor check three lines below: mrw does not quote a line it has not served. |
| `internal/apply/apply_test.go` | edit | The four tests below. |

## Ordered Steps

1. [S1] Write `TestAMultiLineReplaceWithoutAnAnchorIsRefused` and confirm it is RED: a `replace` over `3-6` with no anchor fails, nothing is written, and the message names `anchor=`. [proof: mutation]
2. [S2] Write `TestASingleLineReplaceNeedsNoAnchor` and confirm it is RED against a guard that fires on every replace. This is the control that bounds the change: without it, "refuse every replace" passes S1. [proof: mutation]
3. [S3] Write `TestAPatternRangeSpanningManyLinesNeedsAnAnchor` and confirm it is RED. ⚠ This is the step that decides WHERE the guard lives. A pattern has no resolved span at parse time — `plan.go:599`'s `patterned` escape says so — so a guard in `plan.validate` leaves this case green while S1 passes, which is the address-form-conditional hole ADR-026 and ADR-027 each found once. [proof: mutation]
4. [S4] Write `TestAMultiLineReplaceWithAnAnchorStillApplies` and confirm it is RED before the guard exists only in the sense that the guard is absent; assert the file's resulting CONTENT, not just `Failed == 0`, so a guard that refuses everything and a guard that writes the wrong lines both fail it. [proof: mutation]
5. [S5] Add the guard. Placed on the resolved `start`/`end`, so `3-6`, `/a/,/b/`, `A,+N` and `$` are all covered by one comparison. [proof: mutation]
6. [S6] Confirm `delete`, `insert-after`, `insert-before` and `create` are untouched, and that `lines=` and `sha=` still work as they did — this record narrows nothing else. [proof: acceptance]
7. [S7] Run every gate, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -count=1 -v \
  -run 'TestAMultiLineReplaceWithoutAnAnchorIsRefused|TestASingleLineReplaceNeedsNoAnchor|TestAPatternRangeSpanningManyLinesNeedsAnAnchor|TestAMultiLineReplaceWithAnAnchorStillApplies' 2>&1 | tee /tmp/adr035-t1.out \
  && grep -q '^--- PASS: TestAMultiLineReplaceWithoutAnAnchorIsRefused' /tmp/adr035-t1.out \
  && grep -q '^--- PASS: TestASingleLineReplaceNeedsNoAnchor' /tmp/adr035-t1.out \
  && grep -q '^--- PASS: TestAPatternRangeSpanningManyLinesNeedsAnAnchor' /tmp/adr035-t1.out \
  && grep -q '^--- PASS: TestAMultiLineReplaceWithAnAnchorStillApplies' /tmp/adr035-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr035-t1.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMultiLineReplaceWithoutAnAnchorIsRefused` | `internal/apply/apply_test.go` | A `3-6` replace with no anchor fails, the file is unchanged, and the message names `anchor=` — the remedy, not just the complaint | — | S1, S5 |
| `TestASingleLineReplaceNeedsNoAnchor` | `internal/apply/apply_test.go` | A one-line replace still applies with no anchor. The control that bounds the guard | — | S2, S5 |
| `TestAPatternRangeSpanningManyLinesNeedsAnAnchor` | `internal/apply/apply_test.go` | `/from/,/to/` resolving to several lines is refused on the same terms as `3-6`, which is what places the guard after resolution rather than in `plan.validate` | — | S3, S5 |
| `TestAMultiLineReplaceWithAnAnchorStillApplies` | `internal/apply/apply_test.go` | With an anchor the hunk applies and the file holds the expected content — compared in full, so refusing everything and writing the wrong lines both fail | — | S4, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | the guard runs on every resolved multi-line replace; the S5 mutation removes it and `…WithoutAnAnchorIsRefused` goes red |
| 3 — the caller can discover it | the refusal message names `anchor=`; the plan grammar in `README.md` and `AGENTS.md` (T2) |
| 4 — it is used | contract §73 drives both spellings through the built binary (T2); no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-08 · 3245e1a* · mutant killed · exit 1 · `internal/apply/apply.go` · the guard stops firing, so an anchorless multi-line replace applies again — the behaviour this record removes, and the one that is invisible because a replace that wrote over the wrong span reports ok · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · covers:a multi-line replace without an anchor being refused
- 2026-09-08 · 3245e1a* · mutant killed · exit 1 · `internal/apply/apply.go` · the guard keys on the op instead of the SPAN, so it refuses a single-line replace too — a ban rather than a narrowing, which the table of refusals alone cannot distinguish · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · covers:a single-line replace still needing no anchor
- 2026-09-08 · 3245e1a* · mutant killed · exit 1 · `internal/apply/apply.go` · the guard skips a pattern-resolved range, which is exactly what putting it in plan.validate would produce: a requirement conditional on address form, green for 3-6 and silent for /a/,/b/ — the hole ADR-026 and ADR-027 each found one field at a time · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · covers:a pattern range spanning many lines being refused on the same terms
- 2026-09-08 · 3245e1a* · mutant killed · exit 1 · `internal/apply/apply.go` · the refusal stops naming anchor=, so the caller is told what is wrong but not what to write — and the fence stops being able to say WHICH guard fired, since a dozen other refusals also exit 1 · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · covers:the refusal naming the remedy

## Invariants

- A single-line `replace` is unchanged. `delete`, `insert-after`, `insert-before` and `create` are unchanged.
- `lines=` and `sha=` keep their existing meanings and stay optional; neither satisfies this requirement, for the reason ADR-035's audit table gives.
- The guard sits after `covered()`, so a refusal never quotes a line the caller was not served (ADR-028).
- `internal/read`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- The guard could fire on a single-line replace whose address happens to be written `4-4`. That IS one line and must apply; `TestASingleLineReplaceNeedsNoAnchor` uses that spelling for exactly this reason.
- A test could pass because the hunk failed for an unrelated reason — an unread line, a bad path. Every fixture asserts the MESSAGE names `anchor=`, which no other refusal produces.

## Stop Condition

Stop and ask if a caller is found generating multi-line replaces mechanically with
no access to the served text: the anchor is only a real guard when it is extracted
from the read, and such a caller would be forced to invent one, which is worse than
no guard at all.

## Out of Scope

- The contract row and the documentation — T2
- Any requirement on `delete` (deferred: `docs/adr/BACKLOG.md`)
- Deriving the anchor from the ledger automatically (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · ms:19038
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · ms:15608
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · ms:15672
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · ms:15480
- 2026-09-08 · 3245e1a* · exit 0 · `set -o pipefail …` · acceptance-sha256:c588f2f8b2b54ed44fdb71d9f4c13e9e4f10d859e10de5e346471487bb449d55 · ms:15398
