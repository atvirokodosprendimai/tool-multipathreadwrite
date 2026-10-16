# Task ADR-161-T7: a ranged read serves the line after each range

**Depends-on:** T6
**Covers:** none — no spec
**Estimated scope:** M (read output pinned by many tests and rows)
**Owner:** Zy
**Produces:** the line after each range served and licensed; ADR-136's note retired
**Consumes:** `Options.Quiet` (T6), which licenses that line without printing it
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a range is served with the line after it`, `a replace of the lines read is not refused`

## Goal

A multi-line replace of exactly the lines an agent read is never refused for the line below them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.NextLine`: each span ending before the last line served one line further; ADR-136's note printed only where it is off |
| `cmd/mrw/main.go` | edit | the CLI read sets `NextLine` (what selects it); `mrw_read` does not until T9 |
| `internal/read/next161_test.go` | add | `TestARangedReadServesTheLineAfterIt` |
| `cmd/mrw/{count153,jsonpath155,lines_agree,nullread148,quiet161}_test.go`, `internal/adversarial/adr060_test.go` | edit | the extra line on the CLI (ADR-153 and ADR-155 amended, Zy 2026-10-16) |
| `AGENTS.md` | edit | the ADR-136 sentence, now the CLI's next line and `mrw_read`'s note |
| `scripts/contract.sh` | edit | §267, and rows pinning the note or a ranged read's exact output |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] A span `A-B` with `B` before the last line is served `A-(B+1)`, the extra line numbered like the rest and licensed; `$`, a whole-file read and a span ending on the last line unchanged; `--max-lines` counts it; `--quiet` licenses it without printing; a `--grep` match gets its next line; the `@@ A-B` range line names what was served. [proof: mutation]
3. [S3] ADR-136's note retired on the CLI (`mrw_read` keeps it until T9, whose pages are ranges too); every test and contract row pinning it or a ranged CLI read's exact output updated in the same change. [proof: acceptance]
4. [S4] Contract §267: `$MRW read f:3-5`, then `@@ f 3-5 replace anchor=… body=3` applies, exit 0; the pair, a plan of `3-6` after the same read, is refused for line 7. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./cmd/mrw/ ./internal/guide/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestARangedReadServesTheLineAfterIt ' <<<"$out" \
  && test -z "$(gofmt -l .)" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a replace of exactly the lines read is not refused for the line below' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARangedReadServesTheLineAfterIt` | `internal/read/next161_test.go` | with `NextLine`: the extra line for N, N-M, A,+N, /re/ and /a/,/b/ (a `--grep` match is a /re/ range); none for `$` or a span on the last line; whole files unchanged; the licence; `--max-lines` counts it; `--quiet` licenses it; no ADR-136 note | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the span widening in `read.Run` |
| 2 — something selects it | every ranged `mrw read` |
| 3 — the caller can discover it | `mrw instructions` (T3) |
| 4 — it is used | every plan of the lines just read |

## Mutation Log
(empty until execute)
- 2026-10-16 · c8106825* · mutant killed · exit 1 · `internal/read/read.go` · S2: a range is served without the line after it · acceptance-sha256:2f600d698311e205d94052a06187816635ff24e5e903a4376b31a4e11ab68136
- 2026-10-16 · c8106825* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: the CLI never asks for the line after · acceptance-sha256:2f600d698311e205d94052a06187816635ff24e5e903a4376b31a4e11ab68136
- 2026-10-16 · c8106825* · mutant killed · exit 1 · `internal/read/read.go` · S3: the CLI still prints the note it replaced · acceptance-sha256:2f600d698311e205d94052a06187816635ff24e5e903a4376b31a4e11ab68136

## Invariants

- ADR-052's rule is unchanged: a multi-line replace still needs the line after its end served.
- A whole-file read is unchanged.

## Risks

- A caller that parses a ranged read's output by line count sees one more line; the `@@ A-B` line says so.

## Stop Condition

The extra line breaks an existing contract row or test whose promise is not about the extra line itself; then stop and bring it to Zy.

## Out of Scope

- The MCP surface's served pages (permanent: boundary: T9 serves the extra line there, inside its checkpoints)

## Verification Log
(empty until execute)
- 2026-10-16 · c8106825* · exit 1 · `set -o pipefail …` · acceptance-sha256:2f600d698311e205d94052a06187816635ff24e5e903a4376b31a4e11ab68136 · ms:46811 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-16 · c8106825* · exit 0 · `set -o pipefail …` · acceptance-sha256:2f600d698311e205d94052a06187816635ff24e5e903a4376b31a4e11ab68136 · ms:142179
- 2026-10-16 · c8106825* · exit 0 · `set -o pipefail …` · acceptance-sha256:2f600d698311e205d94052a06187816635ff24e5e903a4376b31a4e11ab68136 · ms:137980
- 2026-10-16 · c8106825* · exit 0 · `set -o pipefail …` · acceptance-sha256:2f600d698311e205d94052a06187816635ff24e5e903a4376b31a4e11ab68136 · ms:170176
