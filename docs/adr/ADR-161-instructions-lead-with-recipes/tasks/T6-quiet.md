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
| `TestAQuietReadLicensesAndPrintsNoLine` | `cmd/mrw/quiet161_test.go` | the output (headers and the count, no numbered or range line), the exit and the ledger equal to a plain read's for a whole file, a range, patterns, `--max-lines`, `--max-cols` and `--grep`; a plan for a quietly read line applies; the three refusals | none | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Options.Quiet` |
| 2 — something selects it | `--quiet` on `read` |
| 3 — the caller can discover it | `--help`, recipe 7 (T3) |
| 4 — it is used | agents writing a plan for lines they know |

## Mutation Log
(empty until execute)
- 2026-10-16 · 3bef2660* · mutant killed · exit 1 · `internal/read/read.go` · S2: a quiet read prints its lines · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b
- 2026-10-16 · 3bef2660* · mutant killed · exit 1 · `internal/read/read.go` · S2: the licensed count counts whole files for a ranged read · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b
- 2026-10-16 · 3bef2660* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: --quiet beside --stat is not refused · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b
- 2026-10-16 · 3bef2660* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: the --quiet flag is parsed and never reaches the read · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b
- 2026-10-16 · 3bef2660* · mutant killed · exit 1 · `internal/read/read.go` · S2: a withheld span prints its @@ range line under --quiet (Codex review of T6) · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b

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
- 2026-10-16 · 3bef2660* · exit 1 · `set -o pipefail …` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:51870 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-16 · 3bef2660* · exit 0 · `set -o pipefail …` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:175821
- 2026-10-16 · 3bef2660* · exit 0 · `set -o pipefail …` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:163591
- 2026-10-16 · 3bef2660* · exit 0 · `set -o pipefail …` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:162513
- 2026-10-16 · 3bef2660* · exit 0 · `set -o pipefail …` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:122786
- 2026-10-16 · 3bef2660* · exit 0 · `set -o pipefail …` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:127568
- 2026-10-16 · 3bef2660* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:0 · test-lock-sha256:585406e7f0e4cd56ddcc2e157ab41e0ff9cfb2ff3668d17865ad2133fe7a543a · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9xdWlldDE2MV90ZXN0LmdvCVRlc3RBUXVpZXRSZWFkTGljZW5zZXNBbmRQcmludHNOb0xpbmUJZjYwNzM2MTNmYTVmZTRmOGQyYTdiNGY4NDdkYmVmM2Y3NGRiOTBmNjIyMzI3ZjFmMTU2Y2M5MjYwMTFjNjhlMw · test-lock-kind:replace
- 2026-10-16 · c8106825* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:0 · test-lock-sha256:aefd8a59ec743912f70b268efbea810a8af8d256d3b6de6c822cec8cb9e45763 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9xdWlldDE2MV90ZXN0LmdvCVRlc3RBUXVpZXRSZWFkTGljZW5zZXNBbmRQcmludHNOb0xpbmUJZjU4ZDRiMGFkNmUwYzhhOGU0MzJlM2Y4ZWQ5N2U5NDljYmQyZDc2ZGJjMTdjNDg5ZDA1MTc5Nzc2YmE5Y2I4MA · test-lock-kind:replace
- 2026-10-22 · 874387ba* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:27fdd6ca66dae0e3aec4bb8228690902ff41712e23699d979f9461a1450aac1b · ms:0 · test-lock-sha256:77752962a3546b5d2eb1f96458aefc225f41b1f1820d1eed280b3740cbf6722d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9xdWlldDE2MV90ZXN0LmdvCVRlc3RBUXVpZXRSZWFkTGljZW5zZXNBbmRQcmludHNOb0xpbmUJMmI3NTI1YjhhNTRkNzhlNDMwOTQ3ZTQ0NzdlMTRkMDgzZmIxMTA0NjhmZmM2MWFlZjQ1ZWMwMzdlMzQ2NzM2OQ · test-lock-kind:replace
