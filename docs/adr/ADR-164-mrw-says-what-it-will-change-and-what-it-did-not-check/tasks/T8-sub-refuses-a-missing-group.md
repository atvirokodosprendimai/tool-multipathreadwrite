# Task ADR-164-T8: a --sub replacement may not name a missing group

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the undefined-group refusal in `internal/read/plansource.go` (`BuildSub`)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a number past the groups is refused`, `an unknown name is refused`, `a defined reference expands`, `a literal $$ passes`, `both surfaces refuse`

## Goal

Decision 8: a replacement naming a group the pattern lacks is a usage error, never a silent empty expansion.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/plansource.go` | edit | `BuildSub` checks the replacement's references against `re.NumSubexp()` and `re.SubexpNames()` before walking |
| `internal/read/sub164_test.go` | add | the test |
| `scripts/contract.sh` | edit | §280 |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestASubNamingAMissingGroupIsRefused (`BuildSub` with pattern `head -1` and replacement `^0?$1 x` is a usage `BuildError` naming `$1` and `$$`; with one group, `$1x` is refused, because Go reads the name `1x`, while `${1}x` plans; `${2}` with one group and `${name}` with no such name are refused; `$0`, `${word}` with `(?P<word>…)`, a literal `$$1`, a lone `$` at the end and `$-` plan as before, because Go keeps a `$` it cannot parse as a name). [proof: acceptance]
2. [S2] The check in `BuildSub`, by the same scan Go's `regexp.Expand` makes (`extract`: after `$`, either `{name}` or the longest run of letters, digits and `_`; an all-digit name is a group number; `$$` a literal dollar; a `$` that starts no name is kept), so it refuses exactly the references Go would expand to nothing. [proof: mutation]
3. [S3] Contract §280: `mrw write --grep 'head -1' --sub '$1 x' .` exits 2, nothing written; through `mrw mcp`, `mrw_write` with `grep`+`sub` answers an `isError` tool result and the server answers the next request; the pair: `--grep '(head) -1' --sub '$1 x'` applies. [proof: human: the executor runs ./scripts/contract.sh once and reads the §280 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestASubNamingAMissingGroupIsRefused$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestASubNamingAMissingGroupIsRefused \(' "$out" \
  && go test ./internal/read/ ./cmd/mrw/ ./internal/mcp/ -count=1 -timeout 300s -run 'TestASubReadPrintsAPlanAndLicensesItsLines|TestASubPlanParsesBackToTheSamePathsAndBodies|TestASubWriteAppliesThePlanInOneCommand|TestAnMCPSubWriteAppliesThePlanInOneCall' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestASubWriteAppliesThePlanInOneCommand \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep '^# 280\. ' scripts/contract.sh > /tmp/adr164-t8-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASubReadPrintsAPlanAndLicensesItsLines` | `internal/read/sub161_test.go` | a valid replacement plans as before (ADR-161) | none | — |
| `TestASubPlanParsesBackToTheSamePathsAndBodies` | `internal/read/sub161_test.go` | the plan text unchanged | none | — |
| `TestASubNamingAMissingGroupIsRefused` | `internal/read/sub164_test.go` | a reference Go would expand to nothing is a usage error naming it and $$; every other spelling plans | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the check in `BuildSub` |
| 2 — something selects it | `BuildSub`, which `write --grep --sub`, `read --grep --sub` and `mrw_write`/`mrw_read` with `sub` all call; contract §280 drives the binary |
| 3 — the caller can discover it | the refusal itself |
| 4 — it is used | nothing measures this; met by a peer session's game-day script (2026-10-20) |

## Mutation Log
(empty until execute)
- 2026-10-20 · 497e9ffc* · mutant killed · exit 1 · `internal/read/plansource.go` · without the check a missing group expands to nothing again · acceptance-sha256:2a373104180f587405b77ffbddd416b0d89068897d3c57e1a52e7f2e53f844da

## Verification Log
(empty until execute)
- 2026-10-20 · 497e9ffc* · exit 1 · `set -o pipefail …` · acceptance-sha256:2a373104180f587405b77ffbddd416b0d89068897d3c57e1a52e7f2e53f844da · ms:746 · test-lock-sha256:02e14d1c52bdb23391641c32d002802077b645ac2d3976597ac9aa8513762192 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9zdWIxNjFfdGVzdC5nbwlUZXN0QVN1YlBsYW5QYXJzZXNCYWNrVG9UaGVTYW1lUGF0aHNBbmRCb2RpZXMJOTQ4ODg0NmQxYzU1NzMwZDc5OWY2YjQ3MWNjZWZmMWM4ZjQ3MTcyYzgwOGQ5YWE1YTlkZTY5Y2IyYjA4YWViMgpib2R5CWludGVybmFsL3JlYWQvc3ViMTYxX3Rlc3QuZ28JVGVzdEFTdWJSZWFkUHJpbnRzQVBsYW5BbmRMaWNlbnNlc0l0c0xpbmVzCTNmYmEyNWQwN2UyYmRhYjA0ZDNhZmMyMDE4OTBkZDgxZGZjZDg4YzY4MTQ1NDgzYTk1YTBlN2FlNjk5M2YxOTkKYm9keQlpbnRlcm5hbC9yZWFkL3N1YjE2MV90ZXN0LmdvCVRlc3RBU3ViUmVmdXNlc0FOZXdUZXh0RW5kaW5nSW5BQ1IJMDZjZmZjZDQzNzQ4YWI3ODFhNmZiMTY1N2Q0YjcxMDc1OWZkZWJhYTM0MjUzZjljOTYwNjExZjU5YWZkZGE0Mg
  ```
  --- last 9 line(s) of stdout
  === RUN   TestASubNamingAMissingGroupIsRefused
      sub164_test.go:36: --grep "head -1" --sub "^0?$1 x": want a usage BuildError, got <nil>
      sub164_test.go:36: --grep "(head) -1" --sub "$1x": want a usage BuildError, got <nil>
      sub164_test.go:36: --grep "(head) -1" --sub "${2}": want a usage BuildError, got <nil>
      sub164_test.go:36: --grep "(head) -1" --sub "${name}": want a usage BuildError, got <nil>
  --- FAIL: TestASubNamingAMissingGroupIsRefused (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.129s
  FAIL
  ```
- 2026-10-20 · 497e9ffc* · exit 0 · `set -o pipefail …` · acceptance-sha256:2a373104180f587405b77ffbddd416b0d89068897d3c57e1a52e7f2e53f844da · ms:6645
- 2026-10-20 · human-observed · Claude observed: ./scripts/contract.sh exit 0 at the T7+T8 tree (2026-10-20), §280 all 5 rows PASS: --sub naming a missing group refused exit 2 naming it and $$, nothing written; mrw_write answers isError and the server answers the next request; the pair applies the group's text · acceptance-sha256:2a373104180f587405b77ffbddd416b0d89068897d3c57e1a52e7f2e53f844da

## Invariants

- A replacement whose references the pattern defines plans exactly as before; `--rewrite` (ast-grep's own metavariables) is untouched.

## Risks

- None beyond the record's.

## Out of Scope

- A `--literal-sub` flag (permanent: boundary: `$$` already writes a dollar, and the refusal names it)

## Stop Condition

Stop and bring it to Zy if Go's template rules (`regexp.Expand`) accept a reference form the check does not parse.
