# Task ADR-072-T3: `--json` is JSON on every refusal after the plan is named

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the `receipt.Error` field; the refusal helper in `write`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a refusal under --json is a document`, `the arrays are empty, not null`, `the error names the cause`, `a contract row drives the binary`, `the engine packages are unchanged`

## Goal

`--json` printed text on a plan that did not parse and on every other refusal between opening the plan and applying it; only a failed hunk and a filesystem failure got JSON. Route every such refusal through one helper that writes a receipt with an `error` field.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `receipt.Error`; the helper; every post-plan `cli.Exit(…, 2)` routed through it |
| `cmd/mrw/json_refusal_test.go` | new | parse error, missing body file, bad harness |
| `scripts/contract.sh` | edit | §144 |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN; the existing write and check tests stay green. [proof: mutation]
   Mutants: the parse site bypasses the helper; `files`/`hunks` left nil; the `error` field dropped.
3. [S3] The contract row drives the built binary with the good case and the one that must fail. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 180s -run 'IsAJSONDocumentUnderJSON' -v 2>&1 | tee /tmp/adr072-T3.out \
  && missing=$(for t in TestAPlanThatDoesNotParseIsAJSONDocumentUnderJSON TestAMissingBodyFileIsAJSONDocumentUnderJSON TestAMalformedHarnessIsAJSONDocumentUnderJSON TestAFilesystemFailureBeforeAnyHunkIsAJSONDocumentUnderJSON; do grep -qE "^--- PASS: $t \(" /tmp/adr072-T3.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 144\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanThatDoesNotParseIsAJSONDocumentUnderJSON` | `cmd/mrw/json_refusal_test.go` | exit 2; parses; `applied` false; `error` names the plan; `hunks` is `[]` | — | S1, S2 |
| `TestAMissingBodyFileIsAJSONDocumentUnderJSON` | `cmd/mrw/json_refusal_test.go` | `body=@nope` the same way | — | S1, S2 |
| `TestAMalformedHarnessIsAJSONDocumentUnderJSON` | `cmd/mrw/json_refusal_test.go` | T1's refusal the same way | — | S1, S2 |
| `TestAFilesystemFailureBeforeAnyHunkIsAJSONDocumentUnderJSON` | `cmd/mrw/json_refusal_test.go` | a failure before any hunk has a verdict (a plan naming a directory) is a document too (review of #229, B2) | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code and its tests |
| 2 — something selects it | every `mrw write`; `mrw check` for T4 |
| 3 — the caller can discover it | the receipt, the exit code and the refusal text |
| 4 — it is used | the v1.25.1 round reproduced each defect with the shipped binary |

## Verification Log
(empty until execute)
- 2026-09-25 · e3f7978* · exit 1 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:330 · test-lock-sha256:96cabd6fde4da827880a2b34baaf179a23bc72fedf6a6916daaa1ca324d07eee · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanNvbl9yZWZ1c2FsX3Rlc3QuZ28JVGVzdEFNYWxmb3JtZWRIYXJuZXNzSXNBSlNPTkRvY3VtZW50VW5kZXJKU09OCTM3N2ZlNjMxY2UzNmUwMzAxMTBiYzgxZTE3MTEwMmY3YjQ4NWIzMjBlYTIzNTVhOGE0NTZlYzIzMmRjZDA2YTAKYm9keQljbWQvbXJ3L2pzb25fcmVmdXNhbF90ZXN0LmdvCVRlc3RBTWlzc2luZ0JvZHlGaWxlSXNBSlNPTkRvY3VtZW50VW5kZXJKU09OCWJiODk5ZWNlNDkzMWMyNDllNWY0Njc0ZjFhNThhMTZmZGU2MGVhN2NmMDJjZmJmMTc3ZGM2MjUwZmU3ZGFiNWMKYm9keQljbWQvbXJ3L2pzb25fcmVmdXNhbF90ZXN0LmdvCVRlc3RBUGxhblRoYXREb2VzTm90UGFyc2VJc0FKU09ORG9jdW1lbnRVbmRlckpTT04JNmIxZjlhNzA0YjhhNjVlYmYzNWZiZmVkYWVkY2Y5YTMxYzlhNjE3ZjY0ZDcwOGNiNDViMGU4MTQ2MmY2MmRjOA
  ```
  --- last 10 line(s) of stdout (of 45 after folding 45 raw)
            "pattern": {
              "advisory_writes": 0,
              "window": 0,
              "fires": false
            }
          }
  --- FAIL: TestAMalformedHarnessIsAJSONDocumentUnderJSON (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.083s
  FAIL
  ```
- 2026-09-25 · e3f7978* · exit 1 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:489
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
  --- FAIL: TestAPlanThatDoesNotParseIsAJSONDocumentUnderJSON (0.00s)
  === RUN   TestAMissingBodyFileIsAJSONDocumentUnderJSON
      json_refusal_test.go:46: --json printed something that is not one JSON document: unexpected end of JSON input
  --- FAIL: TestAMissingBodyFileIsAJSONDocumentUnderJSON (0.00s)
  === RUN   TestAMalformedHarnessIsAJSONDocumentUnderJSON
      json_refusal_test.go:59: --json printed something that is not one JSON document: unexpected end of JSON input
  --- FAIL: TestAMalformedHarnessIsAJSONDocumentUnderJSON (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.095s
  FAIL
  ```
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:355
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:322
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:731
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:1496
- 2026-09-25 · e3f7978* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:0 · test-lock-sha256:9bdae1f58180ff37f296a0aa2be467a66c489639b36aa3299d718e4eb55f33df · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanNvbl9yZWZ1c2FsX3Rlc3QuZ28JVGVzdEFNYWxmb3JtZWRIYXJuZXNzSXNBSlNPTkRvY3VtZW50VW5kZXJKU09OCWRhNWIwOGM2MzQyODllZDcyODBlMTVhOTc2ZDY5YmQ3NTcwN2QxNzM2ZGNhODVmZDlhODk3NDFiZTRiZjhhZDYKYm9keQljbWQvbXJ3L2pzb25fcmVmdXNhbF90ZXN0LmdvCVRlc3RBTWlzc2luZ0JvZHlGaWxlSXNBSlNPTkRvY3VtZW50VW5kZXJKU09OCWJiODk5ZWNlNDkzMWMyNDllNWY0Njc0ZjFhNThhMTZmZGU2MGVhN2NmMDJjZmJmMTc3ZGM2MjUwZmU3ZGFiNWMKYm9keQljbWQvbXJ3L2pzb25fcmVmdXNhbF90ZXN0LmdvCVRlc3RBUGxhblRoYXREb2VzTm90UGFyc2VJc0FKU09ORG9jdW1lbnRVbmRlckpTT04JNmIxZjlhNzA0YjhhNjVlYmYzNWZiZmVkYWVkY2Y5YTMxYzlhNjE3ZjY0ZDcwOGNiNDViMGU4MTQ2MmY2MmRjOA · test-lock-kind:replace
- 2026-09-25 · e3f7978* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:934
- 2026-09-25 · 7836e8d* · exit 0 · `set -o pipefail …` · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · ms:349
- 2026-09-25 · d3f63be* · exit 0 · `set -o pipefail …` · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · ms:685
- 2026-09-25 · d3f63be* · exit 0 · `set -o pipefail …` · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · ms:326
- 2026-09-25 · d3f63be* · exit 0 · `set -o pipefail …` · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · ms:303
- 2026-09-25 · d3f63be* · exit 0 · `set -o pipefail …` · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · ms:304
- 2026-09-25 · d3f63be* · exit 0 · `set -o pipefail …` · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · ms:337

## Mutation Log
(empty until execute)
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · a plan that does not parse prints text under --json again · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · covers:a refusal under --json is a document
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · hunks is null in a refusal, which breaks .hunks[] in jq · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · covers:the arrays are empty, not null
- 2026-09-25 · e3f7978* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refusal document carries no error, so a consumer cannot say why · acceptance-sha256:7e2b5deb3501645914eb80e1a615bc42f8baa7cb4000d875df55e4798fec7c8c · covers:the error names the cause
- 2026-09-25 · d3f63be* · mutant killed · exit 1 · `cmd/mrw/main.go` · a plan that does not parse prints text under --json again · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · covers:a refusal under --json is a document
- 2026-09-25 · d3f63be* · mutant killed · exit 1 · `cmd/mrw/main.go` · hunks is null in a refusal, which breaks .hunks[] in jq · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · covers:the arrays are empty, not null
- 2026-09-25 · d3f63be* · mutant killed · exit 1 · `cmd/mrw/main.go` · the refusal document carries no error, so a consumer cannot say why · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · covers:the error names the cause
- 2026-09-25 · d3f63be* · mutant killed · exit 1 · `cmd/mrw/main.go` · a filesystem failure before any hunk prints nothing under --json again · acceptance-sha256:a8202ea885f6c3a88baf3dd07c581a17aade7842fcec58bbe25ed1abafd9f4ed · covers:a refusal under --json is a document

## Invariants

- Exit codes keep their meaning.

## Risks

- See the record.

## Out of Scope

- Everything the record lists (permanent: boundary: ADR-072 Out of Scope)

## Stop Condition

Stop if the change needs an engine package other than `internal/check`.
