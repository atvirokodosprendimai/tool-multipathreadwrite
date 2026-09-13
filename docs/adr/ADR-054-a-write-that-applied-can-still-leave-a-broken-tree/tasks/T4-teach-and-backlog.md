# Task ADR-054-T4: Teach `--no-check` / balance / stats; BACKLOG 054

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** none
**Consumes:** default check / `--no-check` (T1), `HunkResult.Balance` (T2), stats five names + landed line (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `help names --no-check`, `4096 untouched`, `BACKLOG points at 054`

## Goal

`write --help` names that a check runs after a successful apply when a written path is not prose, unless `--no-check`; that a balance delta does not fail the hunk and is omitted on prose; and that it is not a checker. README Safety gets one line, not an essay. BACKLOG inventory rows for the three proposals cite ADR-054. Shared() is not rewritten.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `writeCmd` Description. The selector a PATH caller trusts. |
| `cmd/mrw/writehelp_test.go` | edit | Assert `--no-check` and that balance is not a refuse. |
| `README.md` | edit | One Safety line. AckRule stays verbatim. |
| `docs/adr/BACKLOG.md` | edit | Inventory disposition: the three proposals are ADR-054. |

## Ordered Steps

1. [S1] Write `TestWriteHelpNamesNoCheck` (or extend the existing help test) and confirm it is RED. [proof: mutation]
2. [S2] Teach on `write --help` and README. Un-strike `TestWriteHelpNamesNoCheck` when it exists. Confirm S1 GREEN. Deleting `--no-check` from Description must fail S1. [proof: mutation]
3. [S3] Point BACKLOG inventory at ADR-054. `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v -run 'TestWriteHelpNamesNoCheck|TestWriteHelpNamesHowToQuoteAHeaderOption' 2>&1 | tee /tmp/adr054-t4.out \
  && grep -q '^--- PASS: TestWriteHelpNamesHowToQuoteAHeaderOption' /tmp/adr054-t4.out \
  && grep -q '^--- PASS: TestWriteHelpNamesNoCheck' /tmp/adr054-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr054-t4.out \
  && grep -q 'ADR-054' docs/adr/BACKLOG.md \
  && grep -q -- '--no-check' README.md \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| ~~`TestWriteHelpNamesNoCheck`~~ | `cmd/mrw/writehelp_test.go` | not yet written — Proposed; T4 S1 writes it | — | S1, S2 |
| `TestWriteHelpNamesHowToQuoteAHeaderOption` | `cmd/mrw/writehelp_test.go` | ADR-040 still holds | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | help test + BACKLOG grep |
| 2 — something selects it | `writeCmd` Description is `write --help` |
| 3 — the caller can discover it | this task |
| 4 — it is used | PATH callers; nothing measures this yet |

## Mutation Log

(empty until execute)

## Invariants

- AckRule verbatim in README.
- Shared() / 4096 untouched.
- ADR-040 quoting sentences stay on `write --help`.

## Risks

- README AckRule test goes red if T4 rewrites the MCP ack bullet. Do not touch that sentence.

## Stop Condition

If teaching requires raising 4096, stop.

## Out of Scope

- Engine (T1–T3)
- MCP handshake

## Verification Log

(empty until execute)
