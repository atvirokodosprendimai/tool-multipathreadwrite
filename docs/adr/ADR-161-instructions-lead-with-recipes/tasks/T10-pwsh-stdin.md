# Task ADR-161-T10: a plan or a read piped from PowerShell loses PowerShell's trailing CRLF

**Depends-on:** T8
**Covers:** none — no spec
**Estimated scope:** S (one shared rule, two callers, one row)
**Owner:** Zy
**Produces:** `ingest.TrimPipedCRLF`, applied to a plan on standard input and to `mrw read -`
**Consumes:** `mrw read -` (T8)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a piped plan gains no line`, `a piped read gains no line`

## Goal

A plan or a read piped from PowerShell means what the caller typed: PowerShell's trailing CRLF does not become a blank line the write adds.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/create.go` | edit | ADR-142's rule as `TrimPipedCRLF`, shared by `CreateContent` |
| `internal/ingest/piped.go` | add | `PipedCRLFReader`: the same rule applied as a plan streams past, holding back only the last two bytes |
| `cmd/mrw/main.go` | edit | a plan read from the real standard input streams through the reader, and stderr says when it dropped the CRLF |
| `cmd/mrw/stdin.go` | edit | `mrw read -` is trimmed, and stderr says so |
| `cmd/mrw/pwsh161_test.go` | add | `TestAPipedCRLFIsDroppedFromStandardInput`, `TestAPipedPlanStreamsAndAFileNamedStdinIsAFile` |
| `internal/ingest/piped161_test.go` | add | `TestThePipedCRLFReaderAgreesWithTrimPipedCRLF` |
| `scripts/contract.sh` | edit | §270 |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] `TrimPipedCRLF` drops a final CRLF after text that holds an LF and no other CR — ADR-142's rule, which `CreateContent` now calls; every other shape is returned as it came. [proof: mutation]
3. [S3] A plan read from the real standard input (`src == os.Stdin`: `mrw write -`, any `--format`; not `--create`, which keeps ADR-142's notes, and not a plan file whatever its name) streams through `PipedCRLFReader`, which holds back only the last two bytes and drops the CRLF at EOF by the same rule; stderr says so once. Streaming keeps what reading it whole lost (the Codex review of T10): an unknown `--format` is refused without waiting for EOF, a line over the parser's limit is refused at once, and no cap on the whole plan is added. [proof: mutation]
4. [S4] `mrw read -` trims the same way and says so. [proof: mutation]
5. [S5] Contract §270: a plan piped with a trailing CRLF after LF lines replaces one line with one line, exit 0, the note on stderr; the pair, an all-CRLF plan, is applied as it always was and gets no note. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/ingest/ ./cmd/mrw/ -count=1 -v -run '^(TestAPipedCRLFIsDroppedFromStandardInput|TestAPipedPlanStreamsAndAFileNamedStdinIsAFile|TestThePipedCRLFReaderAgreesWithTrimPipedCRLF)$' 2>&1) \
  && for n in TestAPipedCRLFIsDroppedFromStandardInput TestAPipedPlanStreamsAndAFileNamedStdinIsAFile TestThePipedCRLFReaderAgreesWithTrimPipedCRLF; do grep -qE "^--- PASS: $n " <<<"$out" || exit 1; done \
  && test -z "$(gofmt -l .)" \
  && go build -o bin/mrw ./cmd/mrw && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a plan piped with PowerShell'"'"'s trailing CRLF writes no extra line' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPipedCRLFIsDroppedFromStandardInput` | `cmd/mrw/pwsh161_test.go` | a plan piped with a trailing CRLF after LF lines replaces one line with one line and names the drop; an all-CRLF plan is applied as before with no note; `read -` of the same shape serves two lines, not three, and names the drop | none | S2, S3, S4 |
| `TestAPipedPlanStreamsAndAFileNamedStdinIsAFile` | `cmd/mrw/pwsh161_test.go` | `--format=bogus --json -` exits 2 at once with standard input left open; a plan FILE named `<stdin>` is applied as that file, not standard input | none | S3 |
| `TestThePipedCRLFReaderAgreesWithTrimPipedCRLF` | `internal/ingest/piped161_test.go` | over fifteen shapes and three read sizes (whole, one byte, half), the reader yields exactly what `TrimPipedCRLF` returns and calls `OnTrim` once exactly when it cut | none | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `ingest.TrimPipedCRLF` |
| 2 — something selects it | the stdin plan source in `write`, and `stdinRead` |
| 3 — the caller can discover it | the stderr note; AGENTS.md's ADR-142 paragraph (T4) |
| 4 — it is used | Windows agents piping plans from PowerShell |

## Mutation Log
(empty until execute)
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `internal/ingest/create.go` · S2: PowerShell's trailing CRLF is never recognised · acceptance-sha256:1b63fec0c9b277e2fb85aa6fb3a1ca98de5029097fd20611ae38abebb226e14b
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: a plan on standard input is parsed with PowerShell's CRLF as a body line · acceptance-sha256:1b63fec0c9b277e2fb85aa6fb3a1ca98de5029097fd20611ae38abebb226e14b
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S4: read - serves PowerShell's CRLF as a line · acceptance-sha256:1b63fec0c9b277e2fb85aa6fb3a1ca98de5029097fd20611ae38abebb226e14b
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `internal/ingest/create.go` · S2: PowerShell's trailing CRLF is never recognised · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: a plan on standard input is parsed with PowerShell's CRLF as a body line · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `internal/ingest/piped.go` · S3: the streaming reader never drops the CRLF · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `cmd/mrw/main.go` · S3: a plan file named <stdin> is taken for standard input (the Codex review of T10) · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e
- 2026-10-16 · 0472f8bb* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S4: read - serves PowerShell's CRLF as a line · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e

## Invariants

- A plan from a file, `--sub`, `--rewrite` or `--create` is unchanged by this task; ADR-142's notes for `--create` are unchanged.

## Risks

- A plan whose last body line really is blank and ends in CRLF after LF lines loses that line: the shape cannot be told from PowerShell's, as ADR-142 says of `--create`, and the note names the drop.

## Stop Condition

The trim changes what any existing plan, contract row or test applies.

## Out of Scope

- An all-CRLF plan that ends in an empty line (permanent: boundary: ADR-142 keeps that shape, since it cannot be told from a blank line the caller meant)

## Verification Log
(empty until execute)
- 2026-10-16 · 0472f8bb* · exit 1 · `set -o pipefail …` · acceptance-sha256:1b63fec0c9b277e2fb85aa6fb3a1ca98de5029097fd20611ae38abebb226e14b · ms:888 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b63fec0c9b277e2fb85aa6fb3a1ca98de5029097fd20611ae38abebb226e14b · ms:80788
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b63fec0c9b277e2fb85aa6fb3a1ca98de5029097fd20611ae38abebb226e14b · ms:79538
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:1b63fec0c9b277e2fb85aa6fb3a1ca98de5029097fd20611ae38abebb226e14b · ms:79466
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e · ms:82548
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e · ms:81580
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e · ms:82639
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e · ms:141617
- 2026-10-16 · 0472f8bb* · exit 0 · `set -o pipefail …` · acceptance-sha256:4423a00c9490ff1b06bf4110847c45de76cc7c43f0a9207c25ab72e399292f9e · ms:80527
