# Task ADR-157-T1: a newer ledger is named first, once, and before an ack

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the newer-ledger refusal reason in `internal/writer/flow.go`; the `mrw_read` newer-ledger line in `internal/mcp/ack.go` and `internal/mcp/tools.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a newer ledger's refusal leads with its cause`, `the refusal keeps its not-read kind`, `the CLI says it once`, `mrw_read says no ack licenses`, `ADR-150's test reads the new lead`, `contract §260 drives the built binary`

## Goal

Decisions 1–3 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/flow.go` | edit | `newerLedger` replaces a not-read reason with the newer-ledger sentence led by the path, instead of appending it |
| `internal/writer/newer157_test.go` | add | the replaced reason keeps `refusal.NotRead` |
| `cmd/mrw/main.go` | edit | the root `Before` skips the newer-ledger line when the verb is `write`; the write command prints it once on stderr after the receipt when the ledger is newer and no hunk's reason carried it |
| `cmd/mrw/newer157_test.go` | add | the CLI refusal: led by the cause, no `mrw read`, no `--force`, the sentence once in stdout and stderr together |
| `cmd/mrw/writer150_test.go` | edit | its refused-write half asserted "has not been read"; it asserts the new lead instead (ADR-150 T1 is re-locked) |
| `internal/mcp/ack.go`, `internal/mcp/tools.go` | edit | the fitting-read footer and the page footer carry one line when `seen.Newer` reports the ledger newer; `nameTheAck` (`tools.go:2332`) adds no ack advice then |
| `internal/mcp/newer157_test.go` | add | `mrw_read` and `mrw_write` under a newer ledger |
| `scripts/contract.sh` | edit | §254's refused-write row reads the new lead; §260 is added |

## Ordered Steps

1. [S1] Write the failing tests and see them red:
   - `TestANewerLedgerReplacesTheNotReadReason` (internal/writer): under a planted `#mrw-seen v6 v9.9.9` ledger, a write to an existing file is refused; the hunk's reason starts `f.txt cannot be edited by this mrw: the read ledger was written by a newer mrw (v9.9.9)`, holds neither `mrw read` nor `--force`, and its `Kind` is `refusal.NotRead`. The pair: the same write under a v5 ledger that does not cover the file keeps today's "has not been read" reason.
   - `TestANewerLedgersRefusalLeadsWithItsCause` (cmd/mrw), under the v6 ledger:
     - `mrw write` of one refused hunk exits 1, and the newer-ledger sentence appears exactly once across stdout and stderr, in the FAIL line;
     - a plan refused in two files prints it once per FAIL line and nowhere else;
     - a successful create of a new file, a `--dry-run`, and a create over an existing file each print it exactly once on stderr;
     - `mrw read` still prints it once on stderr.
   - `TestAnMCPReadUnderANewerLedgerSaysNoAckLicenses` (internal/mcp): `mrw_read` of a file under the v6 ledger carries `an ack here licenses nothing`, and so does a paged read; under a v5 ledger neither does. An `mrw_write` sent WITHOUT acking that read's checkpoints is refused with the reason led by the cause and no ack advice; the pair, the same under a v5 ledger, keeps the ack advice.
2. [S2] `newerLedger` replaces the reason. [proof: mutation]
3. [S3] `Before` skips the line for `write`, and the write command prints it once when no refusal carried it. [proof: mutation]
4. [S4] The two `mrw_read` footers carry the line, and `nameTheAck` adds no ack advice under a newer ledger. [proof: mutation]
5. [S5] `cmd/mrw/writer150_test.go` and contract §254 read the new lead, then ADR-150 T1 is re-locked: `adr-verify docs/adr/ADR-150-the-ledger-names-its-writer/tasks/T1-name-the-writer.md --relock --replace-hashes`, after confirming the diff to that test is this change alone. [proof: human: the executor reads the test diff before re-locking]
6. [S6] Contract §260: a v6 ledger planted in the state directory; `mrw write` is refused, exit 1, its output holds the sentence once and no `--force`; the pair, a v5 ledger covering the line, licenses the write. [proof: human: the executor runs ./scripts/contract.sh once and reads the §260 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/writer/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 600s -run 'TestANewerLedgerReplacesTheNotReadReason|TestANewerLedgersRefusalLeadsWithItsCause|TestAnMCPReadUnderANewerLedgerSaysNoAckLicenses|TestTheCLINamesANewerLedgersWriter' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestANewerLedgerReplacesTheNotReadReason \(' "$out" \
  && grep -qE '^--- PASS: TestANewerLedgersRefusalLeadsWithItsCause \(' "$out" \
  && grep -qE '^--- PASS: TestAnMCPReadUnderANewerLedgerSaysNoAckLicenses \(' "$out" \
  && grep -qE '^--- PASS: TestTheCLINamesANewerLedgersWriter \(' "$out" \
  && grep -q '^# 260\. ' scripts/contract.sh \
  && fmtout=$(gofmt -l internal cmd) && [ -z "$fmtout" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANewerLedgerReplacesTheNotReadReason` | `internal/writer/newer157_test.go` | the reason is led by the cause, names no read and no `--force`, keeps its kind; the v5 pair keeps today's reason | none | S1, S2 |
| `TestANewerLedgersRefusalLeadsWithItsCause` | `cmd/mrw/newer157_test.go` | the sentence once per refused hunk and nowhere else; once on stderr for a success, a dry run and a create over an existing file; once for a read | none | S1, S2, S3 |
| `TestAnMCPReadUnderANewerLedgerSaysNoAckLicenses` | `internal/mcp/newer157_test.go` | both `mrw_read` footers carry the line; an unacked write gets no ack advice; the v5 pair keeps both as today | none | S1, S4 |
| `TestTheCLINamesANewerLedgersWriter` | `cmd/mrw/writer150_test.go` | ADR-150's test, reading the new lead | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the replaced reason; the footer line |
| 2 — something selects it | `newerLedger` runs on every write on both surfaces; the read tool composes its footers on every served read; contract §260 drives the built binary |
| 3 — the caller can discover it | the FAIL line; the `mrw_write` refusal; the `mrw_read` answer |
| 4 — it is used | nothing measures this; built because five Windows sessions reported the doubled sentence and one MCP session the re-read loop (2026-10-14) |

## Mutation Log
(empty until execute)
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/writer/flow.go` · the newer-ledger sentence trails has-not-been-read again instead of replacing it · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `cmd/mrw/main.go` · the root Before prints the newer line for a write too, so a refusal says it twice · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `cmd/mrw/main.go` · a write no refusal speaks for says nothing of the newer ledger · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/mcp/tools.go` · a fitting mrw_read under a newer ledger says nothing of it · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/mcp/tools.go` · a paged mrw_read under a newer ledger says nothing of it · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/mcp/tools.go` · nameTheAck advises an ack under a newer ledger, where an ack licenses nothing · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `cmd/mrw/main.go` · a write that returns before its landing, a plan that does not parse, says nothing of the newer ledger · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `internal/mcp/tools.go` · a read of bytes that are not valid UTF-8 says nothing of the newer ledger, though its CLI advice cannot license either · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `internal/writer/flow.go` · only the first refused hunk is led by the newer ledger · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d

## Verification Log
(empty until execute)
- 2026-10-14 · be76e27f* · exit 1 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1070 · test-lock-sha256:cd60906ea0d1cc0117b80664584337fed64788b0e4775a10e98d36bc82867e9d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9uZXdlcjE1N190ZXN0LmdvCVRlc3RBTmV3ZXJMZWRnZXJzUmVmdXNhbExlYWRzV2l0aEl0c0NhdXNlCWU4ZWQwZDQzMDQ1YjZkM2IzZDhjNWNhYzdjZjk0MjVjMWFjNDRhMTk0MTNkMmJmZmY3Mzg3Yjg2ZmU0MmM5ODgKYm9keQljbWQvbXJ3L3dyaXRlcjE1MF90ZXN0LmdvCVRlc3RUaGVDTElOYW1lc0FOZXdlckxlZGdlcnNXcml0ZXIJY2I2MzY2MWNlMmRiYWY3YzI3OTZlOGE3YjhkYzlmZTA2NDM0NWU1MTIyOTE0ZDIzYjY4NzhhNWFkNjlmMzlmYwpib2R5CWludGVybmFsL21jcC9uZXdlcjE1N190ZXN0LmdvCVRlc3RBbk1DUFJlYWRVbmRlckFOZXdlckxlZGdlclNheXNOb0Fja0xpY2Vuc2VzCTk5MmVjYmMyOTEwMWVkMjFlMGUzMzU5OTdlYzYwMGQwMGU4ZjAzZWE1YTRkNDgwOTJkYTE2YzJmNjMyZTQ0YzQKYm9keQlpbnRlcm5hbC93cml0ZXIvbmV3ZXIxNTdfdGVzdC5nbwlUZXN0QU5ld2VyTGVkZ2VyUmVwbGFjZXNUaGVOb3RSZWFkUmVhc29uCTBhMGJjM2NhMWVlOTkxNGVjNjhkMjY5N2Q3ZjVjYzZlZTA1YTY5ZmI3NTQ0ZDQwYzFhZTkyMzE5YTU0MzIxZTY
  ```
  --- last 10 line(s) of stdout (of 322 after folding 322 raw)
          -- PARTIAL: lines 1-281 of 600. 319 line(s) remain.
          -- Send specs ["big.txt:282-"] to continue, or a narrower range of your own.
          -- Stopping here means you have part of this file, not the file.
          -- This page licenses NOTHING until you acknowledge it.
          -- Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
          -- An id you omit leaves its lines unwritable, which is the point.
  --- FAIL: TestAnMCPReadUnderANewerLedgerSaysNoAckLicenses (0.03s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.223s
  FAIL
  ```
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1694
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1426
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1459
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1428
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1409
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1443
- 2026-10-14 · be76e27f* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:0 · test-lock-sha256:d1d1bff8b33b21a2c4a41d1ee3611fb8ef85f8333cc3f21f41f612946c6fe55b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9uZXdlcjE1N190ZXN0LmdvCVRlc3RBTmV3ZXJMZWRnZXJzUmVmdXNhbExlYWRzV2l0aEl0c0NhdXNlCTMzMTBlNDc1YWI5ODllNzJjNDlkZGQyOTZjMjNlNjE5YTJjNzRkYmZiYmQ5YmUwMjk2Nzk5ZmU0NDgwYzVkZTYKYm9keQljbWQvbXJ3L3dyaXRlcjE1MF90ZXN0LmdvCVRlc3RUaGVDTElOYW1lc0FOZXdlckxlZGdlcnNXcml0ZXIJNDI1MWZhYWE2MWE3ZmJiZjQzN2I1YTc1YTIwOWE3MTA4MGVlZTg1MmY3NTRmYWE3NzMzZmUxYTNiMDVjODE4MApib2R5CWludGVybmFsL21jcC9uZXdlcjE1N190ZXN0LmdvCVRlc3RBbk1DUFJlYWRVbmRlckFOZXdlckxlZGdlclNheXNOb0Fja0xpY2Vuc2VzCTk5MmVjYmMyOTEwMWVkMjFlMGUzMzU5OTdlYzYwMGQwMGU4ZjAzZWE1YTRkNDgwOTJkYTE2YzJmNjMyZTQ0YzQKYm9keQlpbnRlcm5hbC93cml0ZXIvbmV3ZXIxNTdfdGVzdC5nbwlUZXN0QU5ld2VyTGVkZ2VyUmVwbGFjZXNUaGVOb3RSZWFkUmVhc29uCTBhMGJjM2NhMWVlOTkxNGVjNjhkMjY5N2Q3ZjVjYzZlZTA1YTY5ZmI3NTQ0ZDQwYzFhZTkyMzE5YTU0MzIxZTY · test-lock-kind:replace
- 2026-10-14 · human-observed · observed 2026-10-14: ./scripts/contract.sh exit 0, last line contract holds; the 260 rows and the 254 refused-write row PASS; cmd/mrw/writer150_test.go diff is line 54 alone, read before ADR-150 T1 was re-locked
- 2026-10-14 · 85bfc8b5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:0 · test-lock-sha256:0b950d263a249a1ef3a50abf216417168166fc40e128e467f8001a4ece182247 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9uZXdlcjE1N190ZXN0LmdvCVRlc3RBTmV3ZXJMZWRnZXJzUmVmdXNhbExlYWRzV2l0aEl0c0NhdXNlCTViMjViOTgwNGU5YWVjNjMzNWYxNzRiMWQyNTQxZDMxZTAxOTg0OWZjZDgyMzczMTRjNmI5NTMwN2QzNTE5YTIKYm9keQljbWQvbXJ3L3dyaXRlcjE1MF90ZXN0LmdvCVRlc3RUaGVDTElOYW1lc0FOZXdlckxlZGdlcnNXcml0ZXIJNDI1MWZhYWE2MWE3ZmJiZjQzN2I1YTc1YTIwOWE3MTA4MGVlZTg1MmY3NTRmYWE3NzMzZmUxYTNiMDVjODE4MApib2R5CWludGVybmFsL21jcC9uZXdlcjE1N190ZXN0LmdvCVRlc3RBbk1DUFJlYWRVbmRlckFOZXdlckxlZGdlclNheXNOb0Fja0xpY2Vuc2VzCTAzZDdmMTQ4Y2ExYjAxNTdkMGQxNTNmYTUzZTViMmEzMmYwZWQ0ODI5MjI0NGIxNjgxNWRlZmUzNzNiOGEzM2IKYm9keQlpbnRlcm5hbC93cml0ZXIvbmV3ZXIxNTdfdGVzdC5nbwlUZXN0QU5ld2VyTGVkZ2VyUmVwbGFjZXNUaGVOb3RSZWFkUmVhc29uCTBhMGJjM2NhMWVlOTkxNGVjNjhkMjY5N2Q3ZjVjYzZlZTA1YTY5ZmI3NTQ0ZDQwYzFhZTkyMzE5YTU0MzIxZTY · test-lock-kind:replace
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1403
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1411
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:1448
- 2026-10-16 · 8c4b3f31* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:ffe39bf05a696b6e52513f972b1ab206e3881dcba872260d00484788f494066d · ms:0 · test-lock-sha256:03d3e669776db8f4cb128771cff25a72cfa808e7c28eae7d00a7a572037c5bcf · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9uZXdlcjE1N190ZXN0LmdvCVRlc3RBTmV3ZXJMZWRnZXJzUmVmdXNhbExlYWRzV2l0aEl0c0NhdXNlCTViMjViOTgwNGU5YWVjNjMzNWYxNzRiMWQyNTQxZDMxZTAxOTg0OWZjZDgyMzczMTRjNmI5NTMwN2QzNTE5YTIKYm9keQljbWQvbXJ3L3dyaXRlcjE1MF90ZXN0LmdvCVRlc3RUaGVDTElOYW1lc0FOZXdlckxlZGdlcnNXcml0ZXIJODFkYmUxN2I3ZGJkZmU1MWI2Y2I5MGE2YThiZDdkMmQ4Yjk0MzRmNjg3ZDRhN2Y4ZmE4Zjc4NDNmMjQ3OWUwZQpib2R5CWludGVybmFsL21jcC9uZXdlcjE1N190ZXN0LmdvCVRlc3RBbk1DUFJlYWRVbmRlckFOZXdlckxlZGdlclNheXNOb0Fja0xpY2Vuc2VzCWNkMmQ3MGFiMjE3N2U2N2NmZDI3NmVhZTZmYzAxOTUxNzlhYzMzZDBjZmViNGZmOTAzYjBjMzg2ZDhhYzA0ZWEKYm9keQlpbnRlcm5hbC93cml0ZXIvbmV3ZXIxNTdfdGVzdC5nbwlUZXN0QU5ld2VyTGVkZ2VyUmVwbGFjZXNUaGVOb3RSZWFkUmVhc29uCTBkMTVmMDEyMGZkZDM5ODBkZGQ1ZDg4MjU1MTAyMGJkNjBlMjUxYTdhNzc2NmU3ZGJhMzNmMjY5YWIwMmY5Yzc · test-lock-kind:replace

## Invariants

- A newer ledger still licenses nothing and is never written over (ADR-150).
- Under a current ledger every not-read refusal keeps today's words.

## Risks

- None beyond the record's.

## Out of Scope

- The words of `ack.go:138` and `apply.go:1217`, whose advice is a read (permanent: boundary: the newer-ledger line printed beside them says why no read licenses; the record's class audit)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
