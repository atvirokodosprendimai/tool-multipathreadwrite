# Task ADR-164-T4: a check that places some files runs those and names the rest

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `check.Result.OutsideScope`; `check.Run`'s whole-check flag, set by `writer.Verify` for a removed or renamed-away `.go` or module file; per-path judgement in `packages()` and `command()` in `internal/check/check.go`; the `outside the scoped check:` line on both surfaces; `internal/mcp/tools.go`, `internal/mcp/schema.go` and `internal/mcp/testdata/legacy_golden.jsonl` as T2 consumes them
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a file outside is named and its package runs`, `an unplaceable .go runs everything`, `a removed package runs everything`, `a module file runs everything`, `a directory runs everything`, `files and dirs templates report nothing outside`, `the CLI prints it`, `the MCP text keeps the count`, `the receipt keys are listed`

## Goal

Decision 4, with the tests that pin today's all-or-nothing fallback for non-Go FILES rewritten and every other fallback test unchanged.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/check/check.go` | edit | `packages()` judges per path (Decision 4's list); `command()` scopes over placed packages for a `{packages}`-only template; `Result.OutsideScope` |
| `internal/check/partial164_test.go` | add | the engine test |
| `internal/check/check_test.go` | edit | `TestScopeDerivation`'s "non-Go falls back" and "a directory and a non-Go" cases, which hold a non-Go FILE beside a mapped path, now expect the scoped command and `OutsideScope`; "prose beside a package" (a directory) is unchanged |
| `internal/check/adr061_stress_test.go` | edit | the oracle: `{a.go, a.rs}` leaves `unmappedPool`, and a {packages}-only template over it scopes with `a.rs` outside; mixed templates over it scope with nothing outside |
| `internal/writer/flow.go` | edit | `Verify` passes `check.Run` a whole-check flag when the write removed or renamed away a `.go` or module file; `CheckPaths` is unchanged, so no missing name reaches `confine` (ADR-042) (Codex, re-checks of the record) |
| `cmd/mrw/main.go` | edit | `reportCheck` prints the line |
| `cmd/mrw/partial164_test.go` | add | the CLI line and JSON key |
| `internal/mcp/tools.go` | edit | `checkReport` prints the line; the terminal phrase under the ceiling keeps the count (ADR-032) |
| `internal/mcp/schema.go` | edit | a description for `check.outside_scope` (`TestEveryOutputSchemaPropertyIsDescribed`); the `elided` description's order |
| `internal/mcp/testdata/legacy_golden.jsonl` | edit | regenerated for the new description, as ADR-152 did for `check.hint` |
| `internal/mcp/partial164_test.go` | add | the MCP line, and the count under a small ceiling |
| `docs/receipts.txt` | edit | `check outside_scope`, `mcp_write check.outside_scope`, `write check.outside_scope` |
| `scripts/contract.sh` | edit | §276 |
| `README.md`, `AGENTS.md` | edit | the sentence on `{packages}` falling back |

## Ordered Steps

1. [S1] Write the failing tests and see them red: TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest (a module with packages `a` and `b`, `b` imported by `a`; `scoped_check` `go test {packages}`: `a/a.go` + `run.sh` runs `go test ./a` with `OutsideScope` `[run.sh]`; `run.sh` alone runs the whole check; `a/a.go` + `go.mod` runs the whole check; `a/a.go` + a directory named `z.go` (a `.go` path `packages()` cannot place; a present `testdata/x.go` maps today and is not one) runs the whole check; `a/a.go` + the unlink of `go.sum` runs the whole check through the writer (`writer.Verify`), not by handing `go.sum` to `Run`; `a/a.go` + the rename of `b/b.go` to `b/c.go` likewise; `mrw check go.sum` for a missing `go.sum` is still refused (ADR-042); `a/a.go` + `b` as a directory holding no package (what `CheckPaths` hands for an unlinked `b/b.go`) runs the whole check; `go test {packages} && lint {files}` over `a/a.go` + `run.sh` scopes with nothing outside; `{files}` alone is unchanged; called through `command()` and through `Run`, the missing-path refusal of ADR-042 kept as its own case), TestTheCLIPrintsTheFilesOutsideTheScopedCheck (the line and `check.outside_scope`; also `mrw check a/a.go run.sh`), TestTheMCPTextKeepsTheCountOutsideTheScopedCheck (the line; under a ceiling small enough to elide the list, the terminal phrase still says `1 file(s) were outside it`). [proof: acceptance]
2. [S2] `packages()` per path, Decision 4's list. [proof: mutation]
3. [S3] `command()` scopes over placed packages for a `{packages}`-only template; `Run` sets `OutsideScope`. [proof: mutation]
4. [S4] Rewrite the two `TestScopeDerivation` cases and the stress oracle; leave every other fallback test as it is and run them. [proof: acceptance]
5. [S5] `reportCheck` and `checkReport` print the line; the ceiling's terminal phrase; schema descriptions and the golden; `docs/receipts.txt`; README and AGENTS. [proof: mutation]
6. [S6] Contract §276: through the built binary in a scratch module, a `.go` + `.sh` write's receipt shows the scoped command and `outside the scoped check: run.sh`; the pair: a `.sh`-only write shows the whole command; §15, §86 and §113 still pass. [proof: human: the executor runs ./scripts/contract.sh once and reads the §276, §15, §86 and §113 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/check/ -count=1 -timeout 300s -run 'TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest \(' "$out" \
  && go test ./internal/check/ ./cmd/mrw/ ./internal/mcp/ ./internal/writer/ -count=1 -timeout 300s -run 'TestAWriteThatRemovesCodeRunsTheWholeCheckThroughVerify|TestTheCLIPrintsTheFilesOutsideTheScopedCheck|TestTheMCPTextKeepsTheCountOutsideTheScopedCheck|TestScopeDerivation|TestRandomisedCommandMatchesTheFilesScopeOracle|TestPackagesOnlyNonGoStillFallsBack|TestMixedPlaceholdersFallBackWhenUnmapped|TestFilesOnlyWithNoPathsStillFallsBack|TestAPresentUnplaceableInRootScopeStillFallsBack|TestAScopeOutsideTheRootIsRefusedNotFallenBackTo|TestASlowUnscopedCheckSaysHowToScopeIt|TestNoShippedReceiptFieldDisappears|TestNoShippedMCPReceiptFieldDisappears|TestEveryOutputSchemaPropertyIsDescribed|TestALegacyResultIsUnchangedByTheModernPath' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteThatRemovesCodeRunsTheWholeCheckThroughVerify \(' "$out" \
  && grep -qE '^--- PASS: TestTheCLIPrintsTheFilesOutsideTheScopedCheck \(' "$out" \
  && grep -qE '^--- PASS: TestTheMCPTextKeepsTheCountOutsideTheScopedCheck \(' "$out" \
  && grep -qE '^--- PASS: TestScopeDerivation \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' "$out" \
  && grep -qE '^--- PASS: TestNoShippedMCPReceiptFieldDisappears \(' "$out" \
  && ! grep -qE '^(--- )?FAIL' "$out" \
  && grep -x 'write check.outside_scope' docs/receipts.txt > /tmp/adr164-t4-key.txt \
  && grep '^# 276\. ' scripts/contract.sh > /tmp/adr164-t4-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestScopeDerivation` | `internal/check/check_test.go` | two cases rewritten, the rest unchanged | none | S4 |
| `TestRandomisedCommandMatchesTheFilesScopeOracle` | `internal/check/adr061_stress_test.go` | the oracle under the new rule | none | S4 |
| `TestMixedPlaceholdersFallBackWhenUnmapped` | `internal/check/check_test.go` | an empty map still falls back (ADR-061) | none | — |
| `TestAPresentUnplaceableInRootScopeStillFallsBack` | `internal/check/check_test.go` | a directory alone still falls back (ADR-042) | none | — |
| `TestASlowUnscopedCheckSaysHowToScopeIt` | `internal/check/slow152_test.go` | ADR-152's hint unchanged | none | — |
| `TestAWriteThatPlacesSomeFilesRunsThoseAndNamesTheRest` | `internal/check/partial164_test.go` | a .go + .sh write runs the .go's package and names the .sh; module files, unplaceable .go and a package-less directory keep the whole check | none | S1, S2, S3 |
| `TestAWriteThatRemovesCodeRunsTheWholeCheckThroughVerify` | `internal/writer/partial164_test.go` | an unlinked go.sum or a renamed-away .go runs the whole check through writer.Verify | none | S1 |
| `TestTheCLIPrintsTheFilesOutsideTheScopedCheck` | `cmd/mrw/partial164_test.go` | the human line and check.outside_scope; mrw check too | none | S1, S5 |
| `TestTheMCPTextKeepsTheCountOutsideTheScopedCheck` | `internal/mcp/partial164_test.go` | the line, and the count survives the ceiling | none | S1, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Result.OutsideScope`, the per-path judgement |
| 2 — something selects it | `check.Run`, which every write's check and `mrw check PATH` call; contract §276 drives the built binary |
| 3 — the caller can discover it | the line on every receipt; the schema description; README and AGENTS |
| 4 — it is used | nothing measures this; met by this session (2026-10-20: 300 s, exit 3) |

## Mutation Log
(empty until execute)
- 2026-10-20 · 859e3af1* · mutant killed · exit 1 · `internal/check/check.go` · a regular non-Go file empties the map again, so a .go + .sh write falls back to the whole check · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77
- 2026-10-20 · 859e3af1* · mutant killed · exit 1 · `internal/check/check.go` · Run no longer names the files the scoped check left out · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77
- 2026-10-20 · 859e3af1* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write text drops the outside-the-scoped-check line · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77
- 2026-10-20 · 859e3af1* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI receipt drops the outside-the-scoped-check line · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77
- 2026-10-20 · 859e3af1* · mutant killed · exit 1 · `internal/writer/flow.go` · a write that removes a .go or module file runs the scoped check instead of the whole · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77

## Verification Log
(empty until execute)
- 2026-10-20 · 859e3af1* · exit 1 · `set -o pipefail …` · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77 · ms:315 · test-lock-sha256:5b4e6f132b921a55a555a0aaf7dc37624252ee0a62ce5fe80cb02f41e2c86f6c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9wYXJ0aWFsMTY0X3Rlc3QuZ28JVGVzdFRoZUNMSVByaW50c1RoZUZpbGVzT3V0c2lkZVRoZVNjb3BlZENoZWNrCTQ4MzhmMjgyMWQ1YzRhNmQwZjZkNDcyODQ3NDQ3ODdlMTIzNTM0OTNiMmI2OGZjNDQ1ZmUyOGU2ODFhMTIwM2QKYm9keQlpbnRlcm5hbC9jaGVjay9hZHIwNjFfc3RyZXNzX3Rlc3QuZ28JRnV6elNoZWxsQXJnCWE4OTFkYjg4MmI3NzU4ZmM2ZjM0ZDNhZjE5NjY2MzAyMDM2Yjc5NDU3OTA4ZjEyOGZmYzc0Mzg4Nzk3YTg5NGYKYm9keQlpbnRlcm5hbC9jaGVjay9hZHIwNjFfc3RyZXNzX3Rlc3QuZ28JVGVzdFJhbmRvbWlzZWRDb21tYW5kTWF0Y2hlc1RoZUZpbGVzU2NvcGVPcmFjbGUJMzUwZjFkY2RlYTEwNDU0YjU0MDM4MjBmMjI3Y2ZjOWE1NzdhNzMwZmE2NTRjYTc3ZmU1NTVmODZmNzI3ZDBiYQpib2R5CWludGVybmFsL2NoZWNrL2FkcjA2MV9zdHJlc3NfdGVzdC5nbwlUZXN0UmVmU2hlbGxBcmdBZ3JlZXNPblRoZVNwZWNFeGFtcGxlcwkyNDBhNTRhODJhZGEwNjMwZjU3MWIxYTEyYTRiYTc1NzM2N2NhYzNkYjg2Y2U4MmYyYWQ2OWYxZDliNmUzZjkyCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QUNoZWNrVGhhdENhbm5vdFN0YXJ0RGlkTm90UnVuCWQxM2E3MGJhNzI3NjE1MjQzZjhmNzA5NGMxMDJiMDlhM2ExNjRkMjNhNDMwYzMzM2U1MWU4YzI2ZmJmY2Y2MjIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBRGlyZWN0b3J5VGhhdENhbm5vdEJlUmVhZElzUmVmdXNlZE5vdFRyZWF0ZWRBc0VtcHR5CTY3MjM3YTkzYTUxMjZiNzhiY2VhYjAyMmFmZDM5MTQwMzQwOWFkMThmNjViNGQ2NWVlMjI4NjhmZDFmNmEyMDIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBRmFpbGluZ0NoZWNrS2VlcHNJdHNMb2cJOTkxZjQyMWYwZTc3YWI4NDBmMzI5NDJlMjZiYjY0YWFiYzcwOWU0YzRjZTBjMDliZmI5YzhkZmI1ZjFjODBlZQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFQYWRkZWRDaGVja0lzU3RpbGxUaGVEZWNsYXJlZENoZWNrCTk5ZTAyNjI2OTJmMzExZWRhNWM0Y2ZmNWEzMGQxYzA4ZjI4MWM5Yjk1NTMyODFhNzZmYTFkZmJhYTcxYmM4MTUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUGFkZGVkU2NvcGVkQ2hlY2tJc1N0aWxsRGVjbGFyZWQJNmM2ZWU4MGQ4M2IyZDZiZTY3NmMzZjljMDI3YWYzYmEwYjlkMDYyMjgwOTJlYTViYmMwYTNjMjVkNWU5YzgwNwpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFQYXNzaW5nQ2hlY2tMZWF2ZXNOb0xvZ0JlaGluZAk3NTllN2MzZDRjYzQxMTFjMTFhNDNjMzhiYzA1MWVhMjM3MzUyNjU4NGYzN2I1MzY5MDE5OThlNjRmYWFiMDcwCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVByZXNlbnRVbnBsYWNlYWJsZUluUm9vdFNjb3BlU3RpbGxGYWxsc0JhY2sJNTZhMDJkZjg2MTAyZjdlOWJjYmVlMzg4ZjA0MmI1NjdjYjA4ODczMTE1YjQ5MjYyYWE3ZjVhZTFkZTdmOGRiOQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFSZWFkYWJsZURpcmVjdG9yeVN0aWxsU2NvcGVzCWM2ZWFjYTY3YmQxMjZiODQ3NDljOWQ2YTNlZWYwMmUzMjY0MjQzODNjNWVhYTI4ZmY5MDk0ZjkwZWIwNzZjZDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBUmVmdXNlZFNjb3BlRG9lc05vdEluaGVyaXRUaGVSb290c1ZlcmRpY3QJYmEwYjEzMzI3ZTEzNmNlZDVjNTI3OGMxNmVmMDIyMTc1MmIwNGU2YzBjYzc0YzRlZWM2OGUwMjQxZTNhM2Q1Ngpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFTY29wZU91dHNpZGVUaGVSb290SXNSZWZ1c2VkTm90RmFsbGVuQmFja1RvCWFiNzdmMTA4N2YwODVhZTRiN2Y3MjJjZjdjZDY2MzAyZjZkZTc3MmU3NmE3ODI1ZWNmZjU3MmUzNjdkNmMzYzUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBU2hlbGxJbmplY3RlZFNjb3BlU3RpbGxGYWlscwkxYWVhNjA3MzE4ZjM0YTFiMDhiZjU0MjUzNGJjYTY1NTZmMWM3ZjA3ZmFmZWNlZjRmMWY2NDdhOTFiNThlYzBkCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVN0ZXBIb2xkaW5nRGlyc0lzUmVmdXNlZAkwYmEwMjllYzMyMWViYzZjYzhkNzhiMTM4MmYxZjFmNTdlYTc1YjVlYjgyMzFmYTk1N2ZkZjUxMjA3M2VjMjQ2CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0QVdoaXRlc3BhY2VPbmx5Q2hlY2tJc05vdERlY2xhcmVkCWE5MDdlZTQzOWYxNGRjMmU0OGQyZTQxMzgwMTA4NzkxNTMzYTRhNGYwOGUyNGU4ZDBlZjJkNGIzNTc5MWVhNDQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBV2hpdGVzcGFjZU9ubHlTY29wZWRDaGVja0lzTm90RGVjbGFyZWQJZjZiZjBjNTQ5ZDY0NmE3NTA3NTBmZmE1MTBjOThmNjkzNTE1OGY4NWIzNzIyNjBlODg2MjUzNWI5YjRhYTUxYgpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdEFuSW5Sb290TWlzc0lzUmVmdXNlZE5vdEFTaWxlbnRQYXNzCTAxNzRhOTY2MTU4ZjJhOGQ2YTk5MjExZDcwZWUxYjEwZTBjMGY4MDg1NzhjMTFmYTg3NDYzMDcwMDViNDljZDEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RBbk92ZXJsYXJnZVRpbWVvdXRJc0NsYW1wZWROb3RPdmVyZmxvd2VkCTYyMjkzNTRkOTUzMmZkMGFhNTY1YjMwYjQ1NGRkNzdlZGJmZDU5ZDViY2Q2NmI3NGNjZDE5OWRkMjIzODViYjQKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3REaXJzUGxhY2Vob2xkZXJOYW1lc0VhY2hFZGl0ZWREaXJlY3RvcnlPbmNlCTlkMGMwMDNlOWMyNGJmY2FkYWQ4MThmODQ5ZDk2NTFhMzJmYWMzY2YzMzU0NmE4MjQ4ZWRmOTk4OTVjNjMwOWUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RGaWxlc09ubHlXaXRoTm9QYXRoc1N0aWxsRmFsbHNCYWNrCTAxOGIxNjg1NGUxNTdmZDM3YThlODBjYThiYjNmZTU5ZDY1MzY5ZDVkY2E5N2EyMmFkZDQ0ZDM2NTk2ZTI0ZWIKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RGaWxlc1BsYWNlaG9sZGVyCWZhNGJhMGE2ZDU0MzczNjAwZjAwMjQ3OWIwOThhNzUxMTIxOWEyNTQwOGM2YTY5ODQxNTRkZmUyZTA1OTM3YzEKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RGaWxlc1BsYWNlaG9sZGVyRG9lc05vdE5lZWRQYWNrYWdlcwk4MWNmZDQzNTc1MzQ4MGU5NmRkZTE1MjE3YmVlMmFhODU5MDYwOWVkNjNlMGVmNDc4MDhmNDU1OTA0ZjU1NjdkCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0TG9hZEluZmVyc0dvQ2hlY2tCdXRTYXlzU28JMDk5ZTZiZmYzZmU3N2EzMzkwNzUzYzVlZmJjN2UyMTU1ZDNiNjQwNjlhOTliMmUyZGUxZWVmZGQzOWViMzMxYQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdExvYWRPbkFCYXJlRGlyZWN0b3J5SGFzTm9DaGVjawlmNTVlMmIzZmVlZWYzNzhmYmM5MjU5NGVmYzRmYjViZmQ2MzM0YjM3MzU5ODUzYjM1M2MzZjgyODVjYzE0ZTYzCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0TG9hZFByZWZlcnNUaGVEZWNsYXJlZENoZWNrCTI1ZDFmN2NlNTlkZTUzZjhkYmNjODZmODhlYmI1NWU1Y2UzNGU0ZTZlZTI5NzFmYmFkODNlMGZlNmNlMjkwOWUKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RNaXhlZFBsYWNlaG9sZGVyc0ZhbGxCYWNrV2hlblVubWFwcGVkCTFhYzc3MzhjOTc5YzcwODFiNzFhMjI4OTA2MjYzMDQ2OTg2ZWNmNTJkM2I5YzYxNTcyZWRkOGM5N2FkM2QzMzcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3ROb0NoZWNrSXNOb3RBUGFzcwlhM2IzM2RhZTQyN2VhNmM1YTZiYjA4OTk0MDgyMmRlNWFjOGJlOTQyYWViMDIxOTBiYTVhODQ2MDNiMDU4YWI2CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0UGFja2FnZXNPbmx5Tm9uR29TdGlsbEZhbGxzQmFjawkxYmEwYjI1MzBiZmM5M2Y0ODY3MTBlOTAyZTYyOTY0YTM3OGRlNjFlMDNjN2RkZDZiNGVmOGY3YTlmNDU4MzllCmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0UnVuUGFzc2VzCThlYjcwZmY4Zjg5MWM5MDNjYmU2ZDhiOTIwMjVjYzk5NzQ3NTU2YTkyYmRjYTMwYzhjNGE2MTA4ZjVlNThmYTcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RSdW5SZXBvcnRzVGhlUmVhbEV4aXRDb2RlCWEwYTRjMGY2MTBkOTVjZjQ0MWMwMGQ4NWJkODI0YTg1ZDUzOTY2MGFiMjdkNjNiOTBkMjQ2ODQxYzE1N2FkZDcKYm9keQlpbnRlcm5hbC9jaGVjay9jaGVja190ZXN0LmdvCVRlc3RTY29wZURlcml2YXRpb24JYTgyZjZjMDgwNTM2MGNkMDNhY2Q2NGZjZGEzOTVkODc0MWQwNDRjZDZmMTkxMDVhZjcyODNhYjE2YzJiYzI5ZQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFN1YnN0aXR1dGVkUGF0aHNBcmVPbmVTaGVsbEFyZ3VtZW50RWFjaAllYzNlZGVlZTk1NWJkYmMwN2NhOGRkYmVjMjU0YTI1ODMwMzFjMzJkNGIwMjBmNjY2Mzc4NTIxZDQ5Y2Q2MTU2CmJvZHkJaW50ZXJuYWwvY2hlY2svY2hlY2tfdGVzdC5nbwlUZXN0VGFpbEFubm91bmNlc1doYXRJdExlZnRPdXQJNDc0OTQzZGM2MTUzOWEyZWI0NWE4Yzc5MzM4YjllOTY4ZjhiY2QyZWIwOTBlMTdlNzBiODdkYzhjYmM4NmQ4NQpib2R5CWludGVybmFsL2NoZWNrL2NoZWNrX3Rlc3QuZ28JVGVzdFRpbWVvdXRJc1JlcG9ydGVkQXNBRmFpbHVyZU5vdEFQYXNzCWZmN2QwNjBjODc1ZWQ3ZTVkOWY2Y2JiMTMyMjJjYWJjZTQyMjlkZjcwNmYyYzBjYzJkMGE1OTdlOTQ0YWUyYWUKYm9keQlpbnRlcm5hbC9jaGVjay9wYXJ0aWFsMTY0X3Rlc3QuZ28JVGVzdEFXcml0ZVRoYXRQbGFjZXNTb21lRmlsZXNSdW5zVGhvc2VBbmROYW1lc1RoZVJlc3QJYWI4YWUxZWVmZmM1NTliNjM3NTNmZDk3ZDA2MGM2ZDhhZDA5YTZhZDAzMzUzYjYzZGY0NDE4Mjk1YzM3NTllMQpib2R5CWludGVybmFsL2NoZWNrL3Nsb3cxNTJfdGVzdC5nbwlUZXN0QVNsb3dVbnNjb3BlZENoZWNrU2F5c0hvd1RvU2NvcGVJdAllMGM4ZDY1NjlhNzZmMDk1NWExYjUwMTRlYWI5MjY1YTZjMjU0OWY3NWY5MGNlNTY5ZDUyZWYyMWEyZjdmZGU4CmJvZHkJaW50ZXJuYWwvbWNwL3BhcnRpYWwxNjRfdGVzdC5nbwlUZXN0VGhlTUNQVGV4dEtlZXBzVGhlQ291bnRPdXRzaWRlVGhlU2NvcGVkQ2hlY2sJZjc0ZjQyNjBmOTE5YzRhZWNjYzJkZDBhNmFiYjQzZTVjNTFlMDI3YjlmN2FhOGQwMjI0Mzg5ZGViMjk0Zjg3ZQpib2R5CWludGVybmFsL21jcC9wYXJ0aWFsMTY0X3Rlc3QuZ28JYSB3cml0ZSB0aGF0IHBsYWNlcyBldmVyeXRoaW5nIHNheXMgbm90aGluZwk0ZWRhZTZmOWFjYzNiMTI0MmE5NmUyZWU4MTBmYjRkOWNjNzczZDU4ZTBiMzEzY2FlYmViYzI2ODkxZWY0MTk2CmJvZHkJaW50ZXJuYWwvbWNwL3BhcnRpYWwxNjRfdGVzdC5nbwl0aGUgdGVybWluYWwgcGhyYXNlIGtlZXBzIHRoZSBjb3VudAliYmZhMWNhM2QxY2ZmYjk2NzRjZjE1YTUzZDUyMTEwYWJmYTRiMWM3OGFjM2RiYzE2YzkyNGNlMTRlYzYwYzJiCmJvZHkJaW50ZXJuYWwvbWNwL3BhcnRpYWwxNjRfdGVzdC5nbwl0aGUgdGV4dCBhbmQgdGhlIHJlY2VpcHQgbmFtZSB0aGUgZmlsZQljYzA2OTA5NzkwYWZkN2MzYmQ3ODhjN2E4NWZjZTUzYjk3OTJmZTcxYzcwNGI5YjYxOWIxMGE1ZWQxOWI1ZGJiCmJvZHkJaW50ZXJuYWwvd3JpdGVyL3BhcnRpYWwxNjRfdGVzdC5nbwlUZXN0QVdyaXRlVGhhdFJlbW92ZXNDb2RlUnVuc1RoZVdob2xlQ2hlY2tUaHJvdWdoVmVyaWZ5CTY3NWMzMGY4N2MzZWEwZjE5YmE1ZDdlZjMyYjY2ZTIyOTU2ODk0ZmY5NzY0ODFmNGFjN2RiNWIyNWFkOTc5YzQ
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/check/partial164_test.go:56:96: undefined: Whole
  internal/check/partial164_test.go:61:15: cannot use ... in call to non-variadic Run
  internal/check/partial164_test.go:69:29: res.OutsideScope undefined (type Result has no field or method OutsideScope)
  internal/check/partial164_test.go:70:50: res.OutsideScope undefined (type Result has no field or method OutsideScope)
  internal/check/partial164_test.go:85:16: res.OutsideLine undefined (type Result has no field or method OutsideLine)
  internal/check/partial164_test.go:88:23: (Result{}).OutsideLine undefined (type Result has no field or method OutsideLine)
  internal/check/partial164_test.go:94:27: undefined: Option
  internal/check/partial164_test.go:94:27: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check [build failed]
  FAIL
  ```
- 2026-10-20 · 859e3af1* · exit 0 · `set -o pipefail …` · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77 · ms:2767
- 2026-10-20 · 859e3af1* · exit 0 · `set -o pipefail …` · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77 · ms:2423
- 2026-10-20 · 859e3af1* · exit 0 · `set -o pipefail …` · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77 · ms:2349
- 2026-10-20 · 859e3af1* · exit 0 · `set -o pipefail …` · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77 · ms:2623
- 2026-10-20 · 859e3af1* · exit 0 · `set -o pipefail …` · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77 · ms:2519
- 2026-10-20 · human-observed · Claude observed: ./scripts/contract.sh exit 0 at the T4 tree on aa6f709a (2026-10-20): §276 all 6 rows PASS (a .go + .sh write runs the scoped command and names run.sh; mrw check a.go run.sh the same; the pair: a .sh-only write runs the whole check with no outside line), and §15, §86, §113 pass within the same exit-0 run · acceptance-sha256:dce929c59b28402ea0dce57309af11e88b3a78eafe6eb1c714e9de840881ec77

## Invariants

- An unplaceable `.go`, a directory that holds no package, a module file, a write whose paths place nothing, `--full`, and a template with `{files}` or `{dirs}` run exactly what they run today, apart from a mixed template over a partial map, which now scopes with every path in `{files}`.
- ADR-042's refusal of a missing path is unchanged.
- No receipt key is renamed or removed (ADR-111).

## Risks

- A Go test that reads an edited non-Go file is skipped; the file is named (the record's Risks).

## Out of Scope

- Mapping `.sh` or testdata to the Go packages that read them (permanent: boundary: the record's Out of Scope)

## Stop Condition

Stop and bring it to Zy if a contract row other than the ones this task names goes red, or if keeping `TestAPresentUnplaceableInRootScopeStillFallsBack` green needs a rule Decision 4 does not state.
