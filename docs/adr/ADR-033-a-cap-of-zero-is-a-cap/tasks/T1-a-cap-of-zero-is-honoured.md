# Task ADR-033-T1: A cap of zero is honoured

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one field's type, two guards, and the flag's absence)
**Owner:** Zy
**Produces:** `MaxLines *int` and the zero-is-a-cap rule
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a cap of zero serving nothing and reporting it withheld`, `an absent flag still serving the whole file`

## Goal

Make `--max-lines 0` mean zero, without making the struct's zero value mean "serve nothing".

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.MaxLines` becomes `*int`. Nil is "no cap"; a pointer to zero is a cap of zero. The two `opt.MaxLines > 0` guards become "a cap is set", so a zero budget reaches the `WITHHELD` line that already exists. |
| `cmd/mrw/main.go` | edit | The flag is read through `IsSet`, so an absent `--max-lines` and `--max-lines 0` stop being the same input. |
| `internal/read/read_test.go` | edit | `TestACapOfZeroServesNothing`, and the control that an absent cap still serves the file whole. |

## Ordered Steps

1. [S1] Write `TestACapOfZeroServesNothing` and confirm it is RED: with a cap of zero, no content line is served and the span is reported `WITHHELD` with its count. The fixture uses ONE span. §69 is also single-span — it reads a bare path — so the multi-span case is §14's, which already drives `big.txt:1-2,10-12,30-32` through the built binary. [proof: mutation]
2. [S2] Change the field to `*int` and update the two guards. ⚠ Not a `-1` sentinel: the three call sites that omit the field would then mean "serve nothing", and omission must keep meaning "no cap". [proof: mutation]
3. [S3] Read the flag through `IsSet` in `cmd/mrw`. ⚠ The Go test cannot reach this: it drives `read.Run` directly, so CLI wiring that stopped distinguishing an absent flag from an explicit zero would leave it green. §69 (T2) is what covers it, and the mutant for it is logged there. [proof: acceptance]
4. [S4] Assert the control in the same test — no cap set, whole file served — because "serve nothing always" would otherwise pass. ⚠ Assert the SEQUENCE, compared element for element over DISTINCT fixture content. Two weaker forms were tried and both were reproduced as passing on reordered or duplicated output: counting numbered lines, then checking each line's PRESENCE and ordering only the first against the last (reviews two and three of PR #133). [proof: mutation]
5. [S5] Confirm the three call sites that never set the field are untouched and still serve WHAT THEY ASK FOR, uncapped: `internal/mcp` twice — one of which deliberately serves a bounded first page — and `internal/curve` once, which can start from a chosen line. "Whole files" was the wrong words for it. [proof: acceptance]
6. [S6] Run every gate, including `go test -race ./...` and `./scripts/contract.sh`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ -count=1 -v \
  -run 'TestACapOfZeroServesNothing' 2>&1 | tee /tmp/adr033-t1.out \
  && grep -q '^--- PASS: TestACapOfZeroServesNothing' /tmp/adr033-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr033-t1.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr033-t1c.out \
  && grep -q '^contract holds$' /tmp/adr033-t1c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACapOfZeroServesNothing` | `internal/read/read_test.go` | A cap of zero serves no content line and reports its one span withheld with the count; an absent cap serves the file whole — the served sequence extracted and compared element for element, so reordering or a repeat fails. It drives `read.Run` directly and says nothing about CLI wiring, which is §69's | — | S1, S2, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestACapOfZeroServesNothing` |
| 2 — something selects it | The guards run for every read; the S2 mutation restores `> 0` and the fence goes red on a cap of zero serving the file |
| 3 — the caller can discover it | The flag's usage string and the README table (T2) |
| 4 — it is used | Contract §69 (T2) drives both spellings through the built binary; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · 73649a9* · mutant killed · exit 1 · `internal/read/read.go` · a cap of zero stops firing, so it serves the file whole and reports nothing withheld — the behaviour this record removes, which is invisible because a read that served everything looks like success · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · covers:a cap of zero serving nothing and reporting it withheld
- 2026-09-07 · 73649a9* · mutant killed · exit 1 · `internal/read/read.go` · an ABSENT cap becomes a cap of zero, so every read that omits the flag serves nothing — the failure a numeric sentinel would have made the default, and why the field is a pointer · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · covers:an absent flag still serving the whole file

## Invariants
- Omitting the field means NO CAP. The pointer's nil is the safe default, which a `-1` sentinel would have inverted.
- A negative value stays the usage error it already is; this record does not touch `-C`.
- The exit code follows the chain the CLI already had — `read.Run` counts a withheld span as a problem, `cmd/mrw` turns a positive count into exit 1, and contract §14 pins it against the built binary. Not ADR-025, which governs `internal/mcp/**`, and not `readspec_test.go`, which calls `read.Run` directly and cannot see the CLI mapping.
- `internal/apply`, `internal/plan`, `internal/seen` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- A call site that omits the field would break if the sentinel were numeric. It is a pointer for that reason, and S5 checks the three that omit it.
- The test could pass because the read failed for an unrelated reason. S1 asserts the `WITHHELD` line names the right count, which an empty file or a missing path cannot satisfy.

## Stop Condition

Stop and ask if any caller outside this repository is found passing `--max-lines 0` to mean "no cap":
that turns a behaviour change into a break, and the answer is a refusal naming "drop the flag",
not a return to zero-means-infinity.

## Out of Scope

- The contract row and the documentation — T2
- A cross-file budget (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-07 · 73649a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · ms:30557
- 2026-09-07 · 73649a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · ms:30464
- 2026-09-07 · 73649a9* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · ms:31783
- 2026-09-07 · 2fda99d* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · ms:39801
- 2026-09-07 · fd4ce17* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · ms:37411
- 2026-09-07 · 2a67824* · exit 0 · `set -o pipefail …` · acceptance-sha256:ac6b274ce4e8fc9f114b1c63bd412cad025bf73c379d8c824990f13cd4ff5581 · ms:33659
