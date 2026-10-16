# Task ADR-155-T1: a JSON path serves its value's lines, fails as a missed range, and a plan refuses it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `read.Range.JSON`, `jsonSpan` in `internal/read/jsonpath.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a path serves its value's lines`, `a missing path or bad JSON is a missed range`, `a duplicated name on the path is refused`, `bare $ keeps meaning the last line`, `the grammar refuses what it does not have`, `the read licenses exactly the span`, `a plan refuses the form`, `the docs teach it`

## Goal

Decisions 1–7 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/jsonpath.go` | add | the path parser and `jsonSpan` |
| `internal/read/read.go` | edit | `parseRange` takes the form; `splitRanges` keeps a `/` inside it; `resolve` resolves it against the file's bytes |
| `internal/read/jsonpath155_test.go` | add | the resolution and grammar tests |
| `internal/plan/plan.go` | edit | `ParseAddr` refuses the form by name |
| `internal/plan/jsonpath155_test.go` | add | the refusal |
| `cmd/mrw/jsonpath155_test.go` | add | the read licenses exactly the span |
| `internal/mcp/jsonpath155_test.go` | add | `mrw_read` takes the form |
| `internal/guide/guide.go` | edit | `mrw instructions` teaches the form |
| `scripts/contract.sh` | edit | §259 drives the built binary |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | the form taught; the survey entry marked; the write half deferred |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestAJSONPathAddressesTheLinesOfItsValue` (nested object, array index, quoted name, a one-line value, the same value from LF, CRLF and CR-only copies, a BOM; a missing name, an index past the end, a step into a string, a duplicated name on the path naming both lines, JSON Lines, trailing text and a syntax error each a missed range naming why; a duplicate off the path ignored); `TestAJSONPathIsParsedOnlyInTheFormsItHas` (bare `$`, `$-2`, `2-$` unchanged; `$..a`, `$.*`, `$['a']`, `$.`, `$[x]`, `$.a]`, `$.a,+3` refused at parse); `TestAJSONPathReadLicensesExactlyItsLines` (the CLI read serves the value's lines only, a write to its first line applies and to the line after it is refused "has not been read"; a missing path exits 1 with `incomplete:`); `TestAPlanAddressRefusesAJSONPath`; `TestAnMCPReadTakesAJSONPath`.
2. [S2] The grammar in `parseRange` and `splitRanges`. [proof: mutation]
3. [S3] `jsonSpan` and its case in `resolve`. [proof: mutation]
4. [S4] The duplicate refusal. [proof: mutation]
5. [S5] `ParseAddr` refuses the form. [proof: mutation]
6. [S6] AGENTS.md, README.md, `mrw instructions`, BACKLOG. [proof: mutation]
7. [S7] Contract §259: through the built binary, the path read serves exactly the value's lines, a write inside applies and outside is refused, a bare `$` still serves the last line, and a plan in the form is refused. [proof: human: the executor runs ./scripts/contract.sh once and reads the §259 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ ./internal/plan/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 600s -run 'TestAJSONPathAddressesTheLinesOfItsValue|TestAJSONPathIsParsedOnlyInTheFormsItHas|TestAJSONPathReadLicensesExactlyItsLines|TestAPlanAddressRefusesAJSONPath|TestAnMCPReadTakesAJSONPath' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAJSONPathAddressesTheLinesOfItsValue \(' "$out" \
  && grep -qE '^--- PASS: TestAJSONPathIsParsedOnlyInTheFormsItHas \(' "$out" \
  && grep -qE '^--- PASS: TestAJSONPathReadLicensesExactlyItsLines \(' "$out" \
  && grep -qE '^--- PASS: TestAPlanAddressRefusesAJSONPath \(' "$out" \
  && grep -qE '^--- PASS: TestAnMCPReadTakesAJSONPath \(' "$out" \
  && grep -q '^# 259\. ' scripts/contract.sh \
  && grep -q 'file.json:\$\.a\.b' AGENTS.md \
  && grep -q 'file.json:\$\.a\.b' README.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAJSONPathAddressesTheLinesOfItsValue` | `internal/read/jsonpath155_test.go` | the span of a value, across line endings and a BOM; every failure a missed range naming why; a duplicated name on the path refused, one off it ignored | none | S1, S3, S4 |
| `TestAJSONPathIsParsedOnlyInTheFormsItHas` | `internal/read/jsonpath155_test.go` | the grammar; bare `$` unchanged | none | S1, S2 |
| `TestAJSONPathReadLicensesExactlyItsLines` | `cmd/mrw/jsonpath155_test.go` | the CLI read's licence, the write in and out of it, exit 1 on a miss | none | S1, S3 |
| `TestAPlanAddressRefusesAJSONPath` | `internal/plan/jsonpath155_test.go` | the plan refusal by name | none | S1, S5 |
| `TestAnMCPReadTakesAJSONPath` | `internal/mcp/jsonpath155_test.go` | `mrw_read` serves the same span | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `jsonSpan`, `Range.JSON` |
| 2 — something selects it | `parseRange` on every read spec, CLI and MCP; `resolve` on every served file; contract §259 drives the built binary |
| 3 — the caller can discover it | AGENTS.md, README, `mrw instructions`; the parse refusals name the forms |
| 4 — it is used | nothing measures this; built because six survey sessions read JSON by hand (2026-10-09) |

## Mutation Log
(empty until execute)
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/read/jsonpath.go` · an index with a leading zero is accepted · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:the grammar refuses what it does not have
- 2026-10-14 · 100f502b* · mutant survived · exit 0 · `internal/read/read.go` · a slash inside a quoted JSON name splits the range as a pattern · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:the grammar refuses what it does not have
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/read/jsonpath.go` · a bare $ and $-N are taken as JSON paths · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:bare $ keeps meaning the last line
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/read/jsonpath.go` · a CR-only file numbers every value on line 1 · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:a path serves its value's lines
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/read/jsonpath.go` · JSON Lines and trailing text are walked as one document · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:a missing path or bad JSON is a missed range
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/read/read.go` · a path licenses the line after its value · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:the read licenses exactly the span
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/read/jsonpath.go` · a name twice in an object on the path resolves to the first copy · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:a duplicated name on the path is refused
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/plan/plan.go` · a plan address takes a JSON path as a line number · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:a plan refuses the form
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md stops teaching the form · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:the docs teach it
- 2026-10-14 · 100f502b* · mutant killed · exit 1 · `internal/read/read.go` · a slash inside a quoted JSON name swallows the pattern after it · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:the grammar refuses what it does not have
- 2026-10-14 · 3a67e3b2* · mutant killed · exit 1 · `internal/read/jsonpath.go` · a duplicate is reported by its values' lines, not its names' · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · covers:a duplicated name on the path is refused

## Verification Log
(empty until execute)
- 2026-10-14 · 100f502b* · exit 1 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:1556 · test-lock-sha256:9068892f52b2248bae378015dd37789cef5656d47e0aed88d0c778ffc3a3d162 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhSZWFkTGljZW5zZXNFeGFjdGx5SXRzTGluZXMJZmNhNWQ0YmJjODU3ZGM4ODc4MzlkZTgwMTIyZmI5YjNmNWIyNDVjZTY1ZWY0NDE5ZGFkOTFiOTFiZGQzMWJmNgpib2R5CWludGVybmFsL21jcC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBbk1DUFJlYWRUYWtlc0FKU09OUGF0aAk1MWM5NTBhMWQ1YWZkNDczNTQ1NzhhZTQxNDk4ZDQ3MmFkMDE3YzZlN2Y3ZjQ4ZjNjM2M3MDhjOWExYjQ0MDgzCmJvZHkJaW50ZXJuYWwvcGxhbi9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBUGxhbkFkZHJlc3NSZWZ1c2VzQUpTT05QYXRoCTBmOWQ3YWM3NDA5Y2I4ZDFiMjQ3NDkzZjAzZWFmZjgzNzEwYzMzZWE1Nzg4MWZjMDM2MzdiNjNkODNlZjgyMTIKYm9keQlpbnRlcm5hbC9yZWFkL2pzb25wYXRoMTU1X3Rlc3QuZ28JVGVzdEFKU09OUGF0aEFkZHJlc3Nlc1RoZUxpbmVzT2ZJdHNWYWx1ZQkzNWFlMDUyMmI4MjM4ZTNiMGMzMThkNzcwMmEwMzJmYjliMmQxZTNmZTU2YTg5NWY3OGE0ZTU4ZGVjYzIzYTZhCmJvZHkJaW50ZXJuYWwvcmVhZC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhJc1BhcnNlZE9ubHlJblRoZUZvcm1zSXRIYXMJNjgxNmZmYjgxNGVlOWI2NGEzZmUwZjk3NzlhM2NmN2U3NzdjYjY5OTZkMWY4YTJhZDJmOThmNzVkZWNkYjViNA
  ```
  --- last 10 line(s) of stdout (of 33 after folding 33 raw)
  --- FAIL: TestAJSONPathReadLicensesExactlyItsLines (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.229s
  === RUN   TestAnMCPReadTakesAJSONPath
      jsonpath155_test.go:21: mrw_read of f.json:$.scripts.build did not serve line 4 alone:
          f.json:$.scripts.build: "f.json:$.scripts.build": bad line number "$.scripts.build"
  --- FAIL: TestAnMCPReadTakesAJSONPath (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.192s
  FAIL
  ```
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2374
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2358
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2190
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2193
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2163
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2161
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2114
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2723
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:3025
- 2026-10-14 · 100f502b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:0 · test-lock-sha256:13ad48ed17848f8a7ae4ac7e594bde7aa9d320d561239dd9dada41bd976e1b47 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhSZWFkTGljZW5zZXNFeGFjdGx5SXRzTGluZXMJZmNhNWQ0YmJjODU3ZGM4ODc4MzlkZTgwMTIyZmI5YjNmNWIyNDVjZTY1ZWY0NDE5ZGFkOTFiOTFiZGQzMWJmNgpib2R5CWludGVybmFsL21jcC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBbk1DUFJlYWRUYWtlc0FKU09OUGF0aAk1MWM5NTBhMWQ1YWZkNDczNTQ1NzhhZTQxNDk4ZDQ3MmFkMDE3YzZlN2Y3ZjQ4ZjNjM2M3MDhjOWExYjQ0MDgzCmJvZHkJaW50ZXJuYWwvcGxhbi9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBUGxhbkFkZHJlc3NSZWZ1c2VzQUpTT05QYXRoCTBmOWQ3YWM3NDA5Y2I4ZDFiMjQ3NDkzZjAzZWFmZjgzNzEwYzMzZWE1Nzg4MWZjMDM2MzdiNjNkODNlZjgyMTIKYm9keQlpbnRlcm5hbC9yZWFkL2pzb25wYXRoMTU1X3Rlc3QuZ28JVGVzdEFKU09OUGF0aEFkZHJlc3Nlc1RoZUxpbmVzT2ZJdHNWYWx1ZQkzNWFlMDUyMmI4MjM4ZTNiMGMzMThkNzcwMmEwMzJmYjliMmQxZTNmZTU2YTg5NWY3OGE0ZTU4ZGVjYzIzYTZhCmJvZHkJaW50ZXJuYWwvcmVhZC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhJc1BhcnNlZE9ubHlJblRoZUZvcm1zSXRIYXMJNDUzYWQyNzJjZTQxYWU3NGRjNGU2Y2ZjMzQzZjVhNTMwMGRjYjAwM2M1MDQ5ZWRiMmVkNDU1Mzg0NjQ2ZGRmNw · test-lock-kind:replace
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2196
- 2026-10-14 · human-observed · observed by the executor 2026-10-14: S7 ./scripts/contract.sh exit 0 ('contract holds'); every §259 row PASS — the path read serves only the value's lines, the write after the span is refused as unread, the write inside applies, a missing path exits 1 naming it, a bare $ still serves the last line, and a plan in the form is refused by name
- 2026-10-14 · human-observed · observed and approved by the executor 2026-10-14: S7 ./scripts/contract.sh exit 0 ('contract holds'); all §259 rows PASS (the path read, the line after the span, the write inside, a missing path, a bare $, the plan form)
- 2026-10-14 · 100f502b* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:983
- 2026-10-14 · b5bbe01d · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:1151
- 2026-10-14 · 3a67e3b2* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:0 · test-lock-sha256:2d3f7110ca35db6346017be5c4901d8e602f55f375a1d29ce7c28247adfa04a2 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhSZWFkTGljZW5zZXNFeGFjdGx5SXRzTGluZXMJZmNhNWQ0YmJjODU3ZGM4ODc4MzlkZTgwMTIyZmI5YjNmNWIyNDVjZTY1ZWY0NDE5ZGFkOTFiOTFiZGQzMWJmNgpib2R5CWludGVybmFsL21jcC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBbk1DUFJlYWRUYWtlc0FKU09OUGF0aAk1MWM5NTBhMWQ1YWZkNDczNTQ1NzhhZTQxNDk4ZDQ3MmFkMDE3YzZlN2Y3ZjQ4ZjNjM2M3MDhjOWExYjQ0MDgzCmJvZHkJaW50ZXJuYWwvcGxhbi9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBUGxhbkFkZHJlc3NSZWZ1c2VzQUpTT05QYXRoCTBmOWQ3YWM3NDA5Y2I4ZDFiMjQ3NDkzZjAzZWFmZjgzNzEwYzMzZWE1Nzg4MWZjMDM2MzdiNjNkODNlZjgyMTIKYm9keQlpbnRlcm5hbC9yZWFkL2pzb25wYXRoMTU1X3Rlc3QuZ28JVGVzdEFKU09OUGF0aEFkZHJlc3Nlc1RoZUxpbmVzT2ZJdHNWYWx1ZQljNmI4MWUzZWVkZDliNTk3YzNkODQxOTQ5ZjI2MzdhNTk3MjE5YjE2MWQ3YjFkNDNlZGQ0NmM5ZWFkMjQyOGM2CmJvZHkJaW50ZXJuYWwvcmVhZC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhJc1BhcnNlZE9ubHlJblRoZUZvcm1zSXRIYXMJNDUzYWQyNzJjZTQxYWU3NGRjNGU2Y2ZjMzQzZjVhNTMwMGRjYjAwM2M1MDQ5ZWRiMmVkNDU1Mzg0NjQ2ZGRmNw · test-lock-kind:replace
- 2026-10-14 · 3a67e3b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:2464
- 2026-10-14 · 3a67e3b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:753
- 2026-10-16 · c8106825* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:1e17957bdc033d3d5ef03e86c53329d544fd9f6843e27a9e3f398b557b627dbc · ms:0 · test-lock-sha256:64fa930da515b36aabe9af6c3f67e157fedfb52c7d46d7e8cd12c4066622af90 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhSZWFkTGljZW5zZXNFeGFjdGx5SXRzTGluZXMJYTVlMzUxZWZjMjYwZTZhZDM3MGVhMGNlMDQzNzgyMzk2NjhkYzBmMDUwZTA4NTMzMzM1Y2U5ODc3NTJhZDM4Mwpib2R5CWludGVybmFsL21jcC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBbk1DUFJlYWRUYWtlc0FKU09OUGF0aAk1MWM5NTBhMWQ1YWZkNDczNTQ1NzhhZTQxNDk4ZDQ3MmFkMDE3YzZlN2Y3ZjQ4ZjNjM2M3MDhjOWExYjQ0MDgzCmJvZHkJaW50ZXJuYWwvcGxhbi9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBUGxhbkFkZHJlc3NSZWZ1c2VzQUpTT05QYXRoCTBmOWQ3YWM3NDA5Y2I4ZDFiMjQ3NDkzZjAzZWFmZjgzNzEwYzMzZWE1Nzg4MWZjMDM2MzdiNjNkODNlZjgyMTIKYm9keQlpbnRlcm5hbC9yZWFkL2pzb25wYXRoMTU1X3Rlc3QuZ28JVGVzdEFKU09OUGF0aEFkZHJlc3Nlc1RoZUxpbmVzT2ZJdHNWYWx1ZQljNmI4MWUzZWVkZDliNTk3YzNkODQxOTQ5ZjI2MzdhNTk3MjE5YjE2MWQ3YjFkNDNlZGQ0NmM5ZWFkMjQyOGM2CmJvZHkJaW50ZXJuYWwvcmVhZC9qc29ucGF0aDE1NV90ZXN0LmdvCVRlc3RBSlNPTlBhdGhJc1BhcnNlZE9ubHlJblRoZUZvcm1zSXRIYXMJNDUzYWQyNzJjZTQxYWU3NGRjNGU2Y2ZjMzQzZjVhNTMwMGRjYjAwM2M1MDQ5ZWRiMmVkNDU1Mzg0NjQ2ZGRmNw · test-lock-kind:replace

## Invariants

- A bare `$`, `$-N` and `N-$` mean the last line on both paths.
- `internal/apply`, `internal/seen`, `internal/check`, `internal/state` do not change.

## Risks

- None beyond the record's.

## Out of Scope

- Counting with a JSON path (permanent: boundary: `--count` takes a `--grep`, and a grep spec carries no range)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0. If the span mapping needs more than `encoding/json`'s Decoder gives, stop and report.
