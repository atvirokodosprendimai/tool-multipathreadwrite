# Task ADR-109-T5: legacy state in the checkout is not waited on

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the legacy `.mrw/` reads open through `regular.Open`
**Consumes:** `regular.Open` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `legacy state in the checkout is not waited on`

## Goal

Before ADR-004 the ledger and the working set lived in the checkout, under `.mrw/`, and mrw still reads them there: `state.Migrate` at every CLI start, and `seen.Load`, `seen.IsStale` and `iter.load` when the state directory holds none. Each opened them blocking, so a FIFO at `.mrw/seen` hung every command (measured on v1.37.2: `mrw read` killed at 5 s; the Codex review of #306). They open through `regular.Open`; a legacy file that is not regular is not migrated and loads as nothing, which licenses nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/state.go` | edit | `Migrate` |
| `internal/seen/seen.go` | edit | `Load`, `IsStale` |
| `internal/iter/iter.go` | edit | `load` |
| `internal/iter/legacy109_unix_test.go` | add | the test below |
| `scripts/contract.sh` | edit | §208 |

## Ordered Steps

1. [S1] Write the failing test(s) `TestLegacyStateFIFOsAreNotWaitedOn`; confirm RED. [proof: mutation]
2. [S2] The four reads open through `regular.Open`; contract §208 through `$MRW`. Mutants: `Migrate` back to `os.ReadFile`; `iter.load` back to `os.Open`. `seen`'s two reads are reached only after `seen.ReadPath`, whose `fileExists` already admits a FIFO; they share the same mutant shape. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/iter/ -count=1 -timeout 300s -run 'TestLegacyStateFIFOsAreNotWaitedOn' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestLegacyStateFIFOsAreNotWaitedOn \(' "$out" \
  && grep -q '^# 208\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestLegacyStateFIFOsAreNotWaitedOn` | `internal/iter/legacy109_unix_test.go` | with FIFOs at `.mrw/seen` and `.mrw/iteration`, `state.Migrate`, `seen.Load`, `seen.IsStale` and `iter.Load` each return within 5 s, and the loads hold nothing; unix only, where FIFOs exist | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every CLI start (`state.Migrate`), every write (the ledger and working-set loads) |
| 3 — the caller can discover it | the command runs instead of hanging |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex review of #306 and the v1.37.2 measurement |

## Mutation Log
- 2026-10-01 · 1fb4932* · mutant killed · exit 1 · `internal/state/state.go` · Migrate back to os.ReadFile: a FIFO at .mrw/seen hangs every start · acceptance-sha256:91c710be684657d3f664a90f0ac615313be961afd7a22a38c1ee6715ff17b09a
- 2026-10-01 · 1fb4932* · mutant killed · exit 1 · `internal/iter/iter.go` · iter.load back to os.Open: a FIFO legacy working set hangs every write · acceptance-sha256:91c710be684657d3f664a90f0ac615313be961afd7a22a38c1ee6715ff17b09a

## Invariants

- A regular legacy file migrates and loads as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-109 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 1fb4932* · exit 1 · `set -o pipefail …` · acceptance-sha256:91c710be684657d3f664a90f0ac615313be961afd7a22a38c1ee6715ff17b09a · ms:5351 · test-lock-sha256:025186b8d50cd36a35558b87e340d2fc97ff47bc14fa0452ed0631207054f48b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvaXRlci9sZWdhY3kxMDlfdW5peF90ZXN0LmdvCVRlc3RMZWdhY3lTdGF0ZUZJRk9zQXJlTm90V2FpdGVkT24JZjIzMTA2YjE0NTAwYjIzOWE1ZTdiOTRlZTExZDFjNmQzNWFhMjEzOTAwOTg2OTEwMDU5ZmI5ZDg2Zjg5NzgwOQ
  ```
  --- last 6 line(s) of stdout
  === RUN   TestLegacyStateFIFOsAreNotWaitedOn
      legacy109_unix_test.go:48: state.Migrate blocked on a FIFO in .mrw
  --- FAIL: TestLegacyStateFIFOsAreNotWaitedOn (5.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	5.148s
  FAIL
  ```
- 2026-10-01 · 1fb4932* · exit 0 · `set -o pipefail …` · acceptance-sha256:91c710be684657d3f664a90f0ac615313be961afd7a22a38c1ee6715ff17b09a · ms:705
- 2026-10-01 · 1fb4932* · exit 0 · `set -o pipefail …` · acceptance-sha256:91c710be684657d3f664a90f0ac615313be961afd7a22a38c1ee6715ff17b09a · ms:697
