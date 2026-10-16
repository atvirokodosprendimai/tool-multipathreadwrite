# Task ADR-161-T6: read --quiet licenses what it serves and prints no line

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one flag, its test, one row)
**Owner:** Zy
**Produces:** `mrw read --quiet`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a quiet read licenses what a plain read would`, `a quiet read prints no line`

## Goal

An agent licenses lines it already knows without paying for them in its context and without a scratch file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.Quiet`: headers and the closing count printed, no line, the same observations recorded |
| `cmd/mrw/main.go` | edit | `--quiet` on `read`, its refusals, `--help` |
| `cmd/mrw/quiet161_test.go` | add | `TestAQuietReadLicensesAndPrintsNoLine` |
| `scripts/contract.sh` | edit | §266: the built binary |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] `--quiet` prints each file's header and `-- licensed N line(s) in M file(s)`, no served line, no `@@` range line; records exactly the observations the same read without it records (whole file, spans, a `--max-cols` cut left out, `--max-lines` withheld left out); a minimal sentence naming `--quiet` in `guide.CLI()`, as every read flag must be. [proof: mutation]
3. [S3] Usage error (exit 2) beside `--count`, `--stat` or `--sub`. [proof: mutation]
4. [S4] Contract §266: `$MRW read --quiet f:2-3` then a plan replacing line 2 applies, exit 0, and stdout held no line of f; the pair, `$MRW read f:2-3 > /dev/null` then the same plan, is refused as unread. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./cmd/mrw/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestAQuietReadLicensesAndPrintsNoLine ' <<<"$out" \
  && test -z "$(gofmt -l .)" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a quiet read licenses its lines and prints none of them' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the test the fence names joins this table in the change that writes it, since `TestEveryTestNamedByARecordExists` holds every named test to exist | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Options.Quiet` |
| 2 — something selects it | `--quiet` on `read` |
| 3 — the caller can discover it | `--help`, recipe 7 (T3) |
| 4 — it is used | agents writing a plan for lines they know |

## Mutation Log
(empty until execute)

## Invariants

- A read without `--quiet` is unchanged; a read into `/dev/null` still licenses nothing (ADR-133).

## Risks

- An agent licenses lines it never saw; that is already true of a read into a file (ADR-133), and `--quiet` only makes it explicit.

## Stop Condition

The observations a quiet read records cannot be made identical to a plain read's.

## Out of Scope

- `quiet` on `mrw_read` (permanent: boundary: T9 carries it, with the other MCP arguments)

## Verification Log
(empty until execute)
