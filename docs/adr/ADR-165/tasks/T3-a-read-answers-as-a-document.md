# Task ADR-165-T3: a read answers as one JSON document, the receipt being mrw_read's

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `--json` on `mrw read`; the `read` receipt in `docs/receipts.txt`; the receipt composer shared by `readTool` and the CLI
**Consumes:** `read.Options.MaxChars` and the `WITHHELD … --max-chars reached` lines (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `content is the plain read's text`, `receipt equals mrw_read's content[1]`, `a refusal is an error document`, `the exit code is the plain read's`, `the keys are in receipts.txt`, `the refused combinations`

## Goal

Decision 2: `mrw read --json …` prints `{"content": <the plain read's standard output>, "receipt": <built by the function readTool uses for content[1]>}`, exits as the plain read does, writes the plain read's standard error, and licenses what the plain read would have.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | the receipt composition in `readTool` becomes a call to the shared function |
| `internal/mcp/readreceipt.go` | add | `ReadReceipt(observed, problems, skipped, counts…) map[string]any`, the one composer; `next_read` stays `readTool`'s own, added after |
| `cmd/mrw/main.go` | edit | the flag; the plain read written to a buffer, then the document; the `{"error": …}` form for a refusal before any file; the refused combinations (`--sub`, `--rewrite`) |
| `docs/receipts.txt` | edit | the `read` receipt's keys appended (ADR-111) |
| `cmd/mrw/readjson165_test.go` | add | the CLI test |
| `internal/mcp/readreceipt165_test.go` | add | the composer test |
| `cmd/mrw/receipts111_test.go` | edit | the `read` receipt joins `TestNoShippedReceiptFieldDisappears` |
| `scripts/contract.sh` | edit | §283 |
| `README.md`, `AGENTS.md` | edit | the flag; the document's shape in one sentence |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends (`cmd/mrw`, the binary: for two-file ranged specs, a `--grep` walk and a `--grep` that matches nothing, each small enough that `mrw_read` answers it whole, `content` equals the plain `mrw read` standard output byte for byte and `receipt` equals `mrw mcp`'s `content[1]` parsed, with `next_read`, `index` and `next_index` absent from `receipt`; a budgeted read (`--max-chars`, T2) has its served part in `content`, its WITHHELD list on standard error and its count in `problems`; a refusal before any file is `{"error": …}` on standard output with the plain read's exit; `--json --sub` refused exit 2; `--json --quiet` has the quiet read's headers and `-- licensed` line in `content` and licenses as `--quiet`; a later write to a served line applies) and TestTheReadReceiptIsComposedOnceForBothSurfaces (`internal/mcp`: `readTool`'s `content[1]` for an unpaged read is exactly the exported composer's output). [proof: acceptance]
2. [S2] The composer, lifted from `readTool`, and `readTool` calling it; `TestTheReadReceiptMatchesItsSchema` still green. [proof: mutation]
3. [S3] The flag, the buffer, the document, the error form, the refusals; `docs/receipts.txt`'s `read` section and the receipts test. [proof: mutation]
4. [S4] Contract §283: `mrw read --json f.txt:1` is one JSON document whose `content` holds `1| …` and whose `receipt.observed` names `f.txt` with its sha; a refusal (`--json --sub`) exits 2 with an error document; the pair: the plain read's text is the document's `content`. [proof: human: the executor runs ./scripts/contract.sh once and reads the §283 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends \(' "$out" \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestTheReadReceiptIsComposedOnceForBothSurfaces$|TestTheReadReceiptMatchesItsSchema$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheReadReceiptIsComposedOnceForBothSurfaces \(' "$out" \
  && grep -qE '^--- PASS: TestTheReadReceiptMatchesItsSchema \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestNoShippedReceiptFieldDisappears$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' "$out" \
  && grep -q '^# 283\. ' scripts/contract.sh \
  && grep -q '^read ' docs/receipts.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestReadJSONCarriesTheServedTextAndTheReceiptMrwReadSends` | `cmd/mrw/readjson165_test.go` | the document against the plain read and against `mrw_read` for whole answers; the error form; the refusals; quiet; the licence | none | S1, S3 |
| `TestTheReadReceiptIsComposedOnceForBothSurfaces` | `internal/mcp/readreceipt165_test.go` | `readTool` uses the composer | none | S1, S2 |
| `TestTheReadReceiptMatchesItsSchema` | `internal/mcp/schema_test.go` | existing: `mrw_read`'s receipt unchanged by the lift | none | — |
| `TestNoShippedReceiptFieldDisappears` | `cmd/mrw/receipts111_test.go` | existing, extended: the `read` receipt's keys are listed | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the flag, the composer |
| 2 — something selects it | `cmd/mrw` wires `--json`; `readTool` calls the composer (its test fails if it stops); contract §283 drives the binary |
| 3 — the caller can discover it | `mrw read --help`, README, AGENTS.md, `docs/receipts.txt` |
| 4 — it is used | the Codex report's adapter parses the report today; nothing measures it after |

## Invariants

- `mrw_read`'s answer is unchanged (ADR-023): no `structuredContent`, the same `content[1]`, paged and indexed reads included.
- `content` is the CLI's report, never MCP's `content[0]` (checkpoint markers, acknowledgement footer, different walk wording).
- A plain `mrw read` is unchanged.
- Exit codes are the plain read's (ADR-146); `--json` changes the shape, never the verdict.
- The ledger records what `content` holds and nothing more (ADR-117, ADR-133).

## Risks

- The buffer holds the whole answer before printing: a read with no budget can be large. Mitigation: it is the same text the plain read streams; the document is the caller's choice, and `--max-chars` bounds it.
- A refusal after some files were served: the plain read prints their text and then a `REFUSED` header; the document carries exactly that in `content` and the problem in `receipt.problems`, never an `error` document — the test includes one such read.

## Out of Scope

- Paging (`next_read`) on the CLI (permanent: boundary: the CLI has no session to hold a cursor; ADR-014's page is the MCP answer)
- A `--json` form for `--sub`/`--rewrite` plans: `mrw read --sub` prints the plan, which is already a document (permanent: boundary: a plan is its own format, ADR-161)

## Stop Condition

Stop if `readTool`'s receipt cannot be composed by one function without `internal/mcp` importing `cmd/mrw` or the reverse creating a cycle: then the composer needs a home package (`internal/receipt`), which is a module move this record did not decide — report before making it.

## Mutation Log
(empty until execute)
- 2026-10-21 · f31b6ef4* · mutant killed · exit 1 · `internal/mcp/readreceipt.go` · the shared composer drops skipped, on both surfaces · acceptance-sha256:491e83ddab15ab33a97c059d0929940abfc027966bb8fb3172dd422ee5102561
- 2026-10-21 · f31b6ef4* · mutant killed · exit 1 · `cmd/mrw/main.go` · the answer goes to standard output, not into content · acceptance-sha256:491e83ddab15ab33a97c059d0929940abfc027966bb8fb3172dd422ee5102561
- 2026-10-21 · f31b6ef4* · mutant killed · exit 1 · `cmd/mrw/main.go` · a refusal under --json prints no error document · acceptance-sha256:491e83ddab15ab33a97c059d0929940abfc027966bb8fb3172dd422ee5102561

## Verification Log
(empty until execute)
- 2026-10-21 · f31b6ef4* · exit 0 · `set -o pipefail …` · acceptance-sha256:491e83ddab15ab33a97c059d0929940abfc027966bb8fb3172dd422ee5102561 · ms:7123
- 2026-10-21 · f31b6ef4* · exit 0 · `set -o pipefail …` · acceptance-sha256:491e83ddab15ab33a97c059d0929940abfc027966bb8fb3172dd422ee5102561 · ms:6867
- 2026-10-21 · f31b6ef4* · exit 0 · `set -o pipefail …` · acceptance-sha256:491e83ddab15ab33a97c059d0929940abfc027966bb8fb3172dd422ee5102561 · ms:6649
