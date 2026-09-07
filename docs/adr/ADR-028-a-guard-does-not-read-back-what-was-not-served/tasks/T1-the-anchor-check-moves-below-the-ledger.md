# Task ADR-028-T1: The anchor check moves below the ledger

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one check moved, plus the fixture that pins it)
**Owner:** Zy
**Produces:** the anchor check evaluated only for lines the ledger licensed (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a failed anchor printing no line the caller was not served`, `the anchor still checked for lines that were served`

## Goal

Stop a failed `anchor=` printing a line the caller was never shown, without weakening the anchor for
the lines they were.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | The `anchor=` check moves from above `covered()` to below it, matching the ordering ADR-008 already gave the delete-body guard three lines further down, and inheriting its comment's reasoning by reference rather than repeating it. |
| `internal/adversarial/ledger_test.go` | edit | `TestAFailedAnchorDoesNotReadBackAnUnservedLine` lives with the other ledger-boundary tests, not in `internal/apply`, because what it asserts is an ADR-002/005 property rather than an apply mechanic. |

## Ordered Steps

1. [S1] Write `TestAFailedAnchorDoesNotReadBackAnUnservedLine` and confirm it is RED. ⚠ **The fixture must serve a NARROW range and address a line just outside it.** `BACKLOG.md:226` pre-registers the trap: the obvious fixture — serve nothing, address anything — trips the WHOLE-FILE gate instead, so it passes with the ordering reversed and proves nothing. Serve line 1, anchor line 2.
2. [S2] Move the `anchor=` check below `covered()`. [proof: mutation]
3. [S3] Assert the other half in the same test: a hunk whose lines WERE served still gets the anchor message, with the line quoted, because the caller is entitled to it. Without this the fix could be "never check anchors", which would pass a one-sided test. [proof: mutation]
4. [S4] Confirm ADR-008's delete-body ordering test is untouched, and that `lines=` still reports above the ledger — it prints only arithmetic over caller-supplied values. [proof: acceptance]
5. [S5] Run the package, the adversarial package, `gofmt` and `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/adversarial/ -count=1 -v \
  -run 'TestAFailedAnchorDoesNotReadBackAnUnservedLine' 2>&1 | tee /tmp/adr028-t1.out \
  && grep -q '^--- PASS: TestAFailedAnchorDoesNotReadBackAnUnservedLine' /tmp/adr028-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr028-t1.out \
  && go test ./internal/adversarial/ -count=1 -v \
       -run 'TestADeleteWhoseExpectedRemovalDiffersWritesNothing|TestARangedReadLicensesOnlyTheLinesItServed' 2>&1 | tee /tmp/adr028-t1n.out \
  && grep -q '^--- PASS: TestARangedReadLicensesOnlyTheLinesItServed' /tmp/adr028-t1n.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr028-t1n.out \
  && go test ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFailedAnchorDoesNotReadBackAnUnservedLine` | `internal/adversarial/ledger_test.go` | With line 1 served, a failed anchor on line 2 is refused by the ledger and the refusal contains no text from line 2; and with the line served, the anchor message still quotes it | — | S1, S2, S3 |
| `TestARangedReadLicensesOnlyTheLinesItServed` | `internal/adversarial/ledger_test.go` | Unchanged: the per-line licence itself is not disturbed | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAFailedAnchorDoesNotReadBackAnUnservedLine` |
| 2 — something selects it | The two checks sit in the same per-hunk loop every write passes through; the S2 mutation swaps them back and the fence goes red on the leaked text |
| 3 — the caller can discover it | The refusal a caller sees changes from the anchor message to the ledger message, which already names the file, the address and the served spans |
| 4 — it is used | Contract §65 (T2) drives both halves through the built binary; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · bd73ee0* · mutant killed · exit 1 · `internal/apply/apply.go` · the ledger check stops gating the anchor, so a failed anchor guess reads back a line the caller was never served — the ordering this record exists to fix · acceptance-sha256:95fd930c5d89aef070fb5f36c8010495b8e89a709cb4c0bdd31d35db3244dac2 · covers:a failed anchor printing no line the caller was not served
- 2026-09-07 · bd73ee0* · mutant killed · exit 1 · `internal/apply/apply.go` · the anchor stops being checked at all, which a one-sided test would call success — the served-line half of the fixture is what refuses it · acceptance-sha256:95fd930c5d89aef070fb5f36c8010495b8e89a709cb4c0bdd31d35db3244dac2 · covers:the anchor still checked for lines that were served

## Invariants

- A hunk whose lines were served still gets the anchor check, and its message still quotes the line — the fix is an ORDER, not a removal, and a one-sided test would not notice the difference.
- ADR-008's delete-body guard keeps its own position below `covered()`; this record moves its sibling to match rather than moving either of them anywhere new.
- `lines=` stays above the ledger: it prints arithmetic over values the caller supplied and no file content.
- `internal/read`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- ⚠ The fixture can pass for the wrong reason. `BACKLOG.md:226` says so in advance: a fixture that serves NOTHING is refused by the whole-file gate before either guard runs, so it is green with the ordering reversed. S1 serves line 1 and anchors line 2 for that reason, and the S2 mutant is what proves the distinction held.
- The refusal text changes for one input shape, which a caller's script could match on. Judged acceptable: both messages already exist, and matching on a refusal's prose is not a supported interface.

## Stop Condition

Stop and ask if moving the check below `covered()` makes any currently-passing anchor test fail —
that would mean a test depends on the anchor being evaluated for unserved lines, which is the
behaviour this record removes, and the test would need reading before either is changed.

## Out of Scope

- The contract row — that is T2's job
- Making `anchor=` mandatory or promoting it into the licence (permanent: boundary: it matches the first line of a range and cannot carry a per-line claim)

## Verification Log
- 2026-09-07 · bd73ee0* · exit 1 · `set -o pipefail …` · acceptance-sha256:c6faf620acd542923379df4de02c5d72695cea7b07ed365a58937656fe955ff6 · ms:2140
  ```
  --- last 10 line(s) of stdout (of 13 after folding 13 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.438s
  testing: warning: no tests to run
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	1.121s [no tests to run]
  === RUN   TestARangedReadLicensesOnlyTheLinesItServed
  --- PASS: TestARangedReadLicensesOnlyTheLinesItServed (0.00s)
  === RUN   TestADeleteWhoseExpectedRemovalDiffersWritesNothing
  --- PASS: TestADeleteWhoseExpectedRemovalDiffersWritesNothing (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial	0.472s
  ```
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:95fd930c5d89aef070fb5f36c8010495b8e89a709cb4c0bdd31d35db3244dac2 · ms:5593
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:95fd930c5d89aef070fb5f36c8010495b8e89a709cb4c0bdd31d35db3244dac2 · ms:5425
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:95fd930c5d89aef070fb5f36c8010495b8e89a709cb4c0bdd31d35db3244dac2 · ms:5380
