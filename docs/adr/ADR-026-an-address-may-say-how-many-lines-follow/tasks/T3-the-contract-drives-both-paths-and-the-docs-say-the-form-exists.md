# Task ADR-026-T3: The contract drives both paths, and the docs say the form exists

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** M (contract script plus two documents)
**Owner:** Zy
**Produces:** contract §64, and the documented address grammar
**Consumes:** the `A,+N` read grammar and its refusals (T1), `plan.ParseAddr` accepting `A,+N` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built binary serving start..start+N on the read path`, `the built binary applying start..start+N on the plan path`, `the two refusals coming from the built binary`, `the clamp at EOF`

## Goal

Prove the form works in the BUILT binary on both paths in one contract section, and write the
grammar where a caller reads it, so the form is discoverable rather than folklore.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 64.` section. A unit test proves each parser; it cannot prove the binary calls them, which is why §53 exists. This is also the only place the two paths are checked to agree — the divergence ADR-026 was written to remove is invisible to a test that exercises one package. |
| `README.md` | edit | The address grammar a caller reads. `,+N` is added beside `N-M`, `$` and `/regexp/`, for reads and for plan addresses. |
| `AGENTS.md` | edit | Sections 1 and 2 carry the same grammar for agents; a form documented in one and not the other is the defect issues #51 and #73 recorded. |

## Ordered Steps

1. [S1] Confirm §64 does not exist and that the built binary FAILS the row as written: `grep -c '^# 64\. ' scripts/contract.sh` is 0 before the row is added, and the row is red against a binary built before T1 and T2 land. That is this task's red-first — the fence is a contract run, so the row must be able to fail. [proof: acceptance]
2. [S2] Write §64 pairing the good case with the case that must fail, in both directions: a read of `A,+2` serves three lines and exits 0; a plan hunk addressed `A,+2` replaces exactly those three lines; `+3` with no start is refused on BOTH paths and the refusal names the fix; `,+0` is refused on both. A row that scores only the good case passes with a tool that always says yes. [proof: acceptance]
3. [S3] Assert the clamp in the row: a relative end past the last line serves to EOF and exits 0, so the row fails if a later editor turns the clamp into a refusal. [proof: acceptance]
4. [S4] Add the form to `README.md`'s address grammar for both the read spec and the plan address, saying what `N` counts — the lines AFTER the start, so `A,+2` is three lines. An off-by-one in the documentation is the whole cost of this form. [proof: human: read the two grammar passages against §64's fixture and confirm the line counts in the prose match the lines the row asserts]
5. [S5] Add the same to `AGENTS.md` sections 1 and 2, and say plainly that mrw has no backwards relative address, so a caller does not have to discover that by being refused. [proof: human: grep both documents for `,+` and confirm every passage that lists address forms now carries it; a form in one document and not the other is the defect this step exists to prevent]
6. [S6] Run `./scripts/contract.sh` whole, not just the new row — a new fixture can break a later section. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 64\. ' scripts/contract.sh \
  && grep -q ',+N' README.md \
  && grep -q ',+N' AGENTS.md \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr026-t3.out \
  && grep -q '^contract holds$' /tmp/adr026-t3.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr026-t3.out \
  && go test ./... -count=1
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§64` | `scripts/contract.sh` | The built binary serves `A,+2` as three lines, applies it as a plan address, clamps past EOF, and refuses `+3` and `,+0` on both paths | — | S1, S2, S3 |
| `TestARelativeEndServesTheLinesAfterTheStart` | `internal/read/read_test.go` | Unchanged from T1: the row and the unit test must agree, and the row is what proves the binary reaches it | — | — |
| `TestAPlanAddressTakesARelativeEnd` | `internal/plan/plan_test.go` | Unchanged from T2 | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §64 in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section; the row drives `$MRW`, the binary built at the top of the script, so it fails if the wiring is absent even when both unit tests pass |
| 3 — the caller can discover it | `README.md` and `AGENTS.md` carry the form; S5's grep is the check that neither is missing it |
| 4 — it is used | The contract runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI stops carrying the relative end into apply.Input, so `@@ a.go 3,+1 replace` edits only line 3 while every unit test in internal/plan and internal/read stays green; §64 is the only thing that notices · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · covers:the built binary applying start..start+N on the plan path
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/mcp/tools.go` · the MCP server stops carrying the relative end into apply.Input, so an mrw_write plan addressed 3,+1 edits only line 3; the CLI probes in the same section stay green, which is why the section drives the MCP path as well · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · covers:the built binary applying start..start+N on the plan path
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · the built binary stops widening a read span, so §64 read probes serve one line where they assert three — the row is what notices, since a unit test proves the function and not that the shipped binary calls it · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · covers:the built binary serving start..start+N on the read path
- 2026-09-06 · 0cb2ad1* · mutant killed · exit 1 · `internal/read/read.go` · the built binary stops refusing a bare `+N` on the read path, so `mrw read a.go:+3` exits 0 serving line 3 and §64 must go red on the refusal half rather than only on the serving half · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · covers:the two refusals coming from the built binary

## Invariants

- Every existing contract section still passes; §64 adds a fixture of its own and touches no other section's `$R`.
- The row drives `$MRW`, never a Go test.
- `contract holds` is the green footer; the row's verdict comes from the process exit code, never from its output (ADR-003).
- `README.md` and `AGENTS.md` describe the SAME grammar. A form in one and not the other is a defect, not a documentation preference.

## Risks

- The row could assert only the good case, which passes with a parser that accepts everything. Mitigated by S2 requiring both refusals in the same section, in both directions.
- The prose could state the line count off by one — `A,+2` is three lines, not two. Mitigated by S4's human proof reading the prose against the row's own fixture.

## Stop Condition

Stop and ask if §64 cannot be made red before T1 and T2 land: a row that is green against a binary
without the feature is asserting nothing, and finding that out here is the point of S1.

## Out of Scope

- The parsers themselves — T1 and T2
- A backwards relative address in the docs (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · ms:32379
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · ms:24282
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · ms:23036
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · ms:23616
- 2026-09-06 · 0cb2ad1* · exit 0 · `set -o pipefail …` · acceptance-sha256:d3b46705176abe892bd60fa597830d08ada5fda73febb6f2575026f89cba96a0 · ms:21324
