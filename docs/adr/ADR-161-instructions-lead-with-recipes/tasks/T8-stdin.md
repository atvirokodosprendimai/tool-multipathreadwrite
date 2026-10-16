# Task ADR-161-T8: mrw read - reads standard input

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one source of lines, its refusals, one row)
**Owner:** Zy
**Produces:** `mrw read -`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `standard input is read like a file`, `it licenses nothing`

## Goal

A command's output is narrowed with mrw's own addresses and caps, not with `grep`, `sed -n` or `head`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | a spec `-` reads standard input once, bounded as a file is (ADR-104), served as `==> -` |
| `cmd/mrw/main.go` | edit | `-` as a read path, its refusals |
| `cmd/mrw/stdin161_test.go` | add | `TestReadingStandardInputLicensesNothing` |
| `internal/guide/guide.go` | edit | one sentence naming `-` |
| `scripts/contract.sh` | edit | §268 |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] `mrw read [ranges and flags] -` serves standard input with every address, `--grep`, `-C`, `--max-lines` and `--max-cols`; header `==> -`; the 1 GiB bound applies; nothing recorded. [proof: mutation]
3. [S3] Usage error beside `--sub`, `--quiet`, `--stat`, `--count`, `--files-from` or another path; `./-` names a file called `-`. [proof: mutation]
4. [S4] Contract §268: `printf 'a\nb\nc\n' | $MRW read --grep b -` serves line 2, exit 0; then a plan for a file the same command never read is refused; the pair, `... | $MRW read --grep z -`, exits 1 `no match:`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./cmd/mrw/ ./internal/guide/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestReadingStandardInputLicensesNothing ' <<<"$out" \
  && test -z "$(gofmt -l .)" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a read of standard input serves it and licenses nothing' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the test the fence names joins this table in the change that writes it, since `TestEveryTestNamedByARecordExists` holds every named test to exist | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the stdin source in `read.Run` |
| 2 — something selects it | the path `-` |
| 3 — the caller can discover it | `--help`, recipe 9 (T3) |
| 4 — it is used | agents narrowing output |

## Mutation Log
(empty until execute)

## Invariants

- A read of a path is unchanged; nothing read from standard input licenses a line.

## Risks

- A caller pipes a file in to dodge the ledger; it licenses nothing, so a write still needs a read of the file.

## Stop Condition

Reading standard input would block a read that has no pipe; then `-` is refused on a terminal.

## Out of Scope

- `mrw_read` reading a request body as a file (permanent: boundary: the MCP surface has no standard input to give)

## Verification Log
(empty until execute)
