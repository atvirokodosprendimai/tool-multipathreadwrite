# Task ADR-169-T1: `mrw read --if-changed PATH=SHA` answers an unchanged file without its text

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `read.Options.IfChanged`, the `unchanged` receipt list, `mrw read --if-changed`, contract §294
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guard tests run`, `the guard tests passed`, `the receipt-key tests passed`, `the contract run`, `section 294 exists`, `the unchanged row passes`, `the changed row passes`, `the refusal row passes`, `receipts.txt lists mcp_read unchanged`, `gofmt`, `the engine packages it does not own`

## Goal

Decisions 1 to 4: the flag and its refusals, the unused-guard line for a walk, the per-spec unchanged line with what it withheld, exit codes, `--max-chars` as ADR-165 has it, nothing recorded in the ledger by an unchanged answer, and `unchanged` in the `--json` receipt.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Options.IfChanged`; in `Run`, the sha comparison at each spec's load, the unchanged line with N lines and B bytes, the per-spec receipt entry |
| `internal/read/ifchanged169_test.go` | add | the read-engine tests below |
| `cmd/mrw/main.go` | edit | the `--if-changed` flag on `read` — the line that selects the guard; parse at the last `=`, the refusals, the unused-guard line, `unchanged` in `--json`; the usage text |
| `cmd/mrw/ifchanged169_test.go` | add | the refusal, walk, budget and `--json` test |
| `internal/mcp/readreceipt.go` | edit | `ReadReceipt` carries `unchanged`; it builds the CLI `--json` receipt too |
| `internal/mcp/schema_test.go` | edit | `readSchema()` declares `unchanged` and its element fields |
| `internal/mcp/receipts111_test.go` | edit | the shipped-key walk reads a list's `items`, so `mcp_read unchanged[].path` is held |
| `docs/receipts.txt` | edit | `mcp_read unchanged` and its element fields appended (ADR-111) |
| `scripts/contract.sh` | edit | §294, driving the built binary |

## Ordered Steps

1. [S1] Write the tests and confirm RED. Cases:
   - `TestAGuardedReadOfAnUnchangedFileServesNothing`: a file read once, then read with its printed 8-hex sha: one `unchanged since sha … — N lines, B bytes not served` line with the spec's true N and B, no numbered line, exit 0; with the full 64-hex sha and in upper case the same; two specs of one file print two unchanged lines.
   - `TestAGuardedReadOfAChangedFileServesIt`: the file edited between the reads: served whole as unguarded, the header naming the new sha; one unchanged and one changed file in one read serves only the changed one.
   - `TestAnUnchangedAnswerLicensesNothing`: the ledger after an unchanged answer equals the ledger before it, byte for byte — with a prior entry for the file and with none; a write to a line the first read did not serve is still refused.
   - `TestAGuardTheReadCannotUseIsRefused` (cmd): exit 2 and nothing served for a guard on a named file no spec serves, two guards on one file (also through two spellings), a prefix shorter than 8, longer than 64 or not hex, and the flag beside `--count`, `--stat`, `--quiet`, `--sub` or `--rewrite`; PATH holding `=` splits at the last one; under `--grep` a guarded file the walk did not reach is named on the `guard not used` line, exit as without it, and a guarded walked file is judged; `--max-chars 0`, a budget too small for the unchanged line, one that fits it exactly, and a mixed read follow ADR-165 — a withheld unchanged line is listed, exit 1, and absent from `unchanged`; `--json` carries `unchanged: [{"path","sha","lines","bytes"}]` and omits the key when empty; an unchanged spec adds no problem of its own, and a range it would not have served is reported as without the guard; a hard link to the guarded file is the same file; `--max-lines` caps N and B; the right 8 digits with a wrong tail is not the sha.
2. [S2] The flag, the check in `Run` and the receipt. Mutants: the comparison always says changed; the comparison always says unchanged; the unchanged spec is recorded in the ledger whole (nil Spans); the unchanged spec counts as a problem (exit 1); a guard on a named, unserved file is accepted; a walked, unreached guard is refused; the N or B in the line is the file's, not the spec's. [proof: mutation]
3. [S3] Contract §294, each row paired with its refusal: `mrw read --if-changed answers an unchanged file in one line and exits 0`, `mrw read --if-changed serves a changed file`, `mrw read --if-changed refuses a guard the read cannot use`. Append `mcp_read unchanged` and its element fields to `docs/receipts.txt`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./cmd/mrw/ ./internal/mcp/ -count=1 -v -run '^(TestAGuardedReadOfAnUnchangedFileServesNothing|TestAGuardedReadOfAChangedFileServesIt|TestAnUnchangedAnswerLicensesNothing|TestAGuardTheReadCannotUseIsRefused|TestNoShippedMCPReceiptFieldDisappears|TestTheReadReceiptMatchesItsSchema)$' 2>&1) \
  && for t in TestAGuardedReadOfAnUnchangedFileServesNothing TestAGuardedReadOfAChangedFileServesIt TestAnUnchangedAnswerLicensesNothing TestAGuardTheReadCannotUseIsRefused TestNoShippedMCPReceiptFieldDisappears TestTheReadReceiptMatchesItsSchema; do grep -qE "^--- PASS: $t \(" <<<"$out" || exit 1; done \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 294\.' scripts/contract.sh \
  && grep -qF 'PASS  mrw read --if-changed answers an unchanged file in one line and exits 0' <<<"$c" \
  && grep -qF 'PASS  mrw read --if-changed serves a changed file' <<<"$c" \
  && grep -qF 'PASS  mrw read --if-changed refuses a guard the read cannot use' <<<"$c" \
  && grep -qxF 'mcp_read unchanged' docs/receipts.txt \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAGuardedReadOfAnUnchangedFileServesNothing` | `internal/read/ifchanged169_test.go` | the per-spec line with N and B, exit 0, prefix and case | none | S1, S2 |
| `TestAGuardedReadOfAChangedFileServesIt` | `internal/read/ifchanged169_test.go` | a changed file is served as before; mixed reads | none | S1, S2 |
| `TestAnUnchangedAnswerLicensesNothing` | `internal/read/ifchanged169_test.go` | the ledger is untouched, with and without a prior entry; no licence is gained | none | S1, S2 |
| `TestAGuardTheReadCannotUseIsRefused` | `cmd/mrw/ifchanged169_test.go` | the refusals, the walk's unused line, `--max-chars`, the `--json` key, the exit kinds | none | S1, S2 |
| `TestNoShippedMCPReceiptFieldDisappears` | `internal/mcp/receipts111_test.go` | `mcp_read unchanged` and its element fields are held | none | S3 |
| `TestTheReadReceiptMatchesItsSchema` | `internal/mcp/schema_test.go` | the receipt with `unchanged` matches its schema | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the unit tests |
| 2 — something selects it | `--if-changed` on `read` in `cmd/mrw/main.go`; contract §294 drives the built binary |
| 3 — the caller can discover it | the usage text; T3's instructions and skill |
| 4 — it is used | T3 S3's adoption measurement |

## Mutation Log
(empty until execute)
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · the guard always says changed: an unchanged file is served · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · the guard always says unchanged: a changed file is withheld · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · an unchanged spec counts as a problem: exit 1 · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/guard.go` · a guard on a named, unserved file is accepted · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/guard.go` · a walked guard the walk did not reach is refused · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · N and B are the file's, not the spec's · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · a range a guarded spec would not have served is not reported · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · --max-lines does not cap N and B · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/guard.go` · a hard link to the guarded file is not the same file · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · the unchanged spec is recorded in the ledger whole · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
- 2026-10-23 · 4bfd6600* · mutant killed · exit 1 · `internal/read/read.go` · only the first 8 digits of a longer guard are compared · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b

## Invariants

- A read with no `--if-changed` is byte-identical to today's, on stdout, stderr, the exit code and the ledger.
- An unchanged answer adds and removes no ledger entry; a guard that does not match changes nothing a read would not.

## Risks

- `Run` loads per spec; the check sits at each load (`read.go:639-689`), so a walk and two specs of one file are judged the same way.

## Stop Condition

The fence exits 0; or the check cannot sit at a spec's load without reading the file twice, which goes back to Zy.

## Out of Scope

- The MCP argument (deferred: ADR-169 T2)
- Teaching it (deferred: ADR-169 T3)

## Verification Log
(empty until execute)
- 2026-10-23 · 4bfd6600* · exit 1 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:1247 · test-lock-sha256:4b8f4d093d4b16bdff82128ba030e28baa038cb31bd1bb3e09e5f7970147d559 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QUd1YXJkVGhlUmVhZENhbm5vdFVzZUlzUmVmdXNlZAk4YzU5ZGQyOTE4ODNiMmVlYTE5MDI1MGI4MzFiMjdmMDg3ZjNjN2JlNTcwMGEwM2JkNjkxMjI1YmE3ZTljZmNjCmJvZHkJY21kL21ydy9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QW5VbmNoYW5nZWRBbnN3ZXJBdFRoZUNMSUxpY2Vuc2VzTm9Xcml0ZQliOTI5OGMxMjlkMTlhMzBmNGM1YmYwYTdlYzE0NDQ0ZGQzYzU2OTM2MTRlNjg5MjhlZmZlYmUzOTgxZmU5Mjc3CmJvZHkJaW50ZXJuYWwvbWNwL3JlY2VpcHRzMTExX3Rlc3QuZ28JVGVzdE5vU2hpcHBlZE1DUFJlY2VpcHRGaWVsZERpc2FwcGVhcnMJNGRkNzBkYjkyZTcwYjZmN2Q1ZGVjNmM3Njk3YjJkODRkOGVkM2I1YWIxNmQ4YmZlMjNhYmVhYTQxNjhiNDczYwpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QUN1c3RvbU1hcnNoYWxsZXJNZWFuc05vUmVxdWlyZWRQcm9taXNlCTIzOTQ4ZDk0ZjlmMTA5ZTlhZmE4YzFiMjkwMzBjNjUzNjM3NWE2OGE2OWFhY2I0OTRiY2ZjMzgzNzViOGY5ZDgKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFEZXNjcmliZWRQcm9wZXJ0eVRoYXROb0xvbmdlckV4aXN0c0lzUmVmdXNlZAk5ZGI4N2VlYzM2MzRkN2I0YWNhYmI0M2E4N2MxNTQ0Zjc2MTcwZmE3NzdlNmMxNzM1YjUxMTEyZTQyZmUyM2QxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBTmlsU2xpY2VJc0FkbWl0dGVkQnlUaGVTY2hlbWEJYzgwOWRiYjQ0NjI1NWJkMDM4ODY5OWM1NGMyNzEzY2I5ZTgwMDNlNmMzZjkzZWM4YmJlMTYwMThkMTVjYmY4OApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QVByb3BlcnR5bGVzc09iamVjdFNjaGVtYUlzUmVmdXNlZAliZmRmOTllMzhlMzM4YjUwNzhmM2QyZDBiOWQzYTczYjBkYmYxNWZhNGVkOGVjZWVkNGI2N2Y4NzQ5MjJjNGYxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVSZWFkUmVjZWlwdE1hdGNoZXNJdHNTY2hlbWEJODRjYmM5MTE4YzAxMTUzMTFiNjI3ZDk3NjM5ZTE1Y2U0OWU4NjYzZDM1MTE5OWZmODAxMWFkN2IxZGFkNWI4OQpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0VGhlU2NoZW1hTmFtZXNUaGVGaWVsZHNPZlRoZVJlc3VsdAk1YjNhOGNkMTg5MjFhYTRmY2Y4YWFmMmY3NWQwMTIwMDk1YThmOWI4MTJiMTc3NGFlZTJjNjg0ZDIyOWM2NGI2CmJvZHkJaW50ZXJuYWwvcmVhZC9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QUd1YXJkZWRSZWFkT2ZBQ2hhbmdlZEZpbGVTZXJ2ZXNJdAk5NDNmMjhmZDNiNzNmZDQwMjdjOTAwMjlmMGY0NTVlYjdiODNmMWEyN2VlOTBkYmZjZWQzOTgyYTM4MTFiZmMxCmJvZHkJaW50ZXJuYWwvcmVhZC9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QUd1YXJkZWRSZWFkT2ZBblVuY2hhbmdlZEZpbGVTZXJ2ZXNOb3RoaW5nCTQ1NGNkNjRiYjVhMGM0ZmQ2MzI1NTdhODk1M2M3MmExNjZmNWQyOGU2Y2I5MWEwZWQzZjA1Yzg5NGVlODVlODAKYm9keQlpbnRlcm5hbC9yZWFkL2lmY2hhbmdlZDE2OV90ZXN0LmdvCVRlc3RBblVuY2hhbmdlZEFuc3dlckxpY2Vuc2VzTm90aGluZwk4MTMyYmYxYjk5MGQwOWZkNjdkMTc5MTUyZTVmYTE3YjUyM2I2MWQ4ZTlmNjljYzExOTEzMDVlYzlhOGUxOThk
  ```
  ```
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:83522
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:102350
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:88045
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:85989
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:86530
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:97105
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:80828
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:79251
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:77130
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:77428
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:77419
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:77448
- 2026-10-23 · 4bfd6600* · exit 0 · `set -o pipefail …` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:77987
- 2026-10-23 · 4bfd6600* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b · ms:0 · test-lock-sha256:57571e80d0c97452140c3802080ed2c80d129a77066f01237631aeb63ce589f8 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QUd1YXJkVGhlUmVhZENhbm5vdFVzZUlzUmVmdXNlZAk4YzU5ZGQyOTE4ODNiMmVlYTE5MDI1MGI4MzFiMjdmMDg3ZjNjN2JlNTcwMGEwM2JkNjkxMjI1YmE3ZTljZmNjCmJvZHkJY21kL21ydy9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QW5VbmNoYW5nZWRBbnN3ZXJBdFRoZUNMSUxpY2Vuc2VzTm9Xcml0ZQliOTI5OGMxMjlkMTlhMzBmNGM1YmYwYTdlYzE0NDQ0ZGQzYzU2OTM2MTRlNjg5MjhlZmZlYmUzOTgxZmU5Mjc3CmJvZHkJaW50ZXJuYWwvbWNwL3JlY2VpcHRzMTExX3Rlc3QuZ28JVGVzdE5vU2hpcHBlZE1DUFJlY2VpcHRGaWVsZERpc2FwcGVhcnMJNGRkNzBkYjkyZTcwYjZmN2Q1ZGVjNmM3Njk3YjJkODRkOGVkM2I1YWIxNmQ4YmZlMjNhYmVhYTQxNjhiNDczYwpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QUN1c3RvbU1hcnNoYWxsZXJNZWFuc05vUmVxdWlyZWRQcm9taXNlCTIzOTQ4ZDk0ZjlmMTA5ZTlhZmE4YzFiMjkwMzBjNjUzNjM3NWE2OGE2OWFhY2I0OTRiY2ZjMzgzNzViOGY5ZDgKYm9keQlpbnRlcm5hbC9tY3Avc2NoZW1hX3Rlc3QuZ28JVGVzdEFEZXNjcmliZWRQcm9wZXJ0eVRoYXROb0xvbmdlckV4aXN0c0lzUmVmdXNlZAk5ZGI4N2VlYzM2MzRkN2I0YWNhYmI0M2E4N2MxNTQ0Zjc2MTcwZmE3NzdlNmMxNzM1YjUxMTEyZTQyZmUyM2QxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RBTmlsU2xpY2VJc0FkbWl0dGVkQnlUaGVTY2hlbWEJYzgwOWRiYjQ0NjI1NWJkMDM4ODY5OWM1NGMyNzEzY2I5ZTgwMDNlNmMzZjkzZWM4YmJlMTYwMThkMTVjYmY4OApib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0QVByb3BlcnR5bGVzc09iamVjdFNjaGVtYUlzUmVmdXNlZAliZmRmOTllMzhlMzM4YjUwNzhmM2QyZDBiOWQzYTczYjBkYmYxNWZhNGVkOGVjZWVkNGI2N2Y4NzQ5MjJjNGYxCmJvZHkJaW50ZXJuYWwvbWNwL3NjaGVtYV90ZXN0LmdvCVRlc3RUaGVSZWFkUmVjZWlwdE1hdGNoZXNJdHNTY2hlbWEJODRjYmM5MTE4YzAxMTUzMTFiNjI3ZDk3NjM5ZTE1Y2U0OWU4NjYzZDM1MTE5OWZmODAxMWFkN2IxZGFkNWI4OQpib2R5CWludGVybmFsL21jcC9zY2hlbWFfdGVzdC5nbwlUZXN0VGhlU2NoZW1hTmFtZXNUaGVGaWVsZHNPZlRoZVJlc3VsdAk1YjNhOGNkMTg5MjFhYTRmY2Y4YWFmMmY3NWQwMTIwMDk1YThmOWI4MTJiMTc3NGFlZTJjNjg0ZDIyOWM2NGI2CmJvZHkJaW50ZXJuYWwvcmVhZC9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QUd1YXJkZWRSZWFkT2ZBQ2hhbmdlZEZpbGVTZXJ2ZXNJdAk5NDNmMjhmZDNiNzNmZDQwMjdjOTAwMjlmMGY0NTVlYjdiODNmMWEyN2VlOTBkYmZjZWQzOTgyYTM4MTFiZmMxCmJvZHkJaW50ZXJuYWwvcmVhZC9pZmNoYW5nZWQxNjlfdGVzdC5nbwlUZXN0QUd1YXJkZWRSZWFkT2ZBblVuY2hhbmdlZEZpbGVTZXJ2ZXNOb3RoaW5nCWU0NzQyNjViYzIwOGVkNDRkMjcwMmI4MGRhZDllMWUwZTdjODUxMWY4ODU2OTlkNGIwMjY0YzQ4ZjA1NjU4NjYKYm9keQlpbnRlcm5hbC9yZWFkL2lmY2hhbmdlZDE2OV90ZXN0LmdvCVRlc3RBblVuY2hhbmdlZEFuc3dlckxpY2Vuc2VzTm90aGluZwk4MTMyYmYxYjk5MGQwOWZkNjdkMTc5MTUyZTVmYTE3YjUyM2I2MWQ4ZTlmNjljYzExOTEzMDVlYzlhOGUxOThk · test-lock-kind:replace
- 2026-10-23 · human-observed · relock 2026-10-23 reviewed, after the go-reviewer pass: TestAGuardedReadOfAnUnchangedFileServesNothing gains four cases (the right 8 digits with a wrong tail is served; a missed range under a guard is one problem beside the unchanged line, per Zy's 'Still a problem'; --max-lines 1 caps N and B; a hard link is the same file through ResolveGuards); TestIfChangedAnswersUnchangedWithoutAnError gains a grep that matches nothing refusing two guards on one file; contract §294's licence row writes its reads to a file, not /dev/null, so it can fail. No assertion removed or weakened; approved · acceptance-sha256:fdc09e1095a1c7539b9ddd39d62d4119de187cbecbfe450cf3594e9b71353f2b
