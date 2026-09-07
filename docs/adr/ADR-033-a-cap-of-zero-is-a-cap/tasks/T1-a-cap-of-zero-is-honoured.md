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

1. [S1] Write `TestACapOfZeroServesNothing` and confirm it is RED: with a cap of zero, no content line is served and every span is reported `WITHHELD` with its own count. [proof: mutation]
2. [S2] Change the field to `*int` and update the two guards. ⚠ Not a `-1` sentinel: the three call sites that omit the field would then mean "serve nothing", and omission must keep meaning "no cap". [proof: mutation]
3. [S3] Read the flag through `IsSet` in `cmd/mrw`. [proof: mutation]
4. [S4] Assert the control in the same test — no cap set, whole file served — because "serve nothing always" would otherwise pass. [proof: mutation]
5. [S5] Confirm the three call sites that never set the field are untouched and still serve whole files: `internal/mcp` twice and `internal/curve` once. [proof: acceptance]
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
| `TestACapOfZeroServesNothing` | `internal/read/read_test.go` | A cap of zero serves no content line and reports every span withheld with its count; an absent cap serves the file whole | — | S1, S2, S3, S4 |

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
- ADR-025 decides what a zero cap RETURNS: a read that served nothing is an error, and that is inherited rather than re-decided here.
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
