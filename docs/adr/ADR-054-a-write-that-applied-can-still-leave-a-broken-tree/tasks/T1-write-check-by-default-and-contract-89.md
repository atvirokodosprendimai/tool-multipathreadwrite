# Task ADR-054-T1: CLI write runs the check by default on non-prose; `--no-check`; contract §89

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** default check / `--no-check` (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `check exists then default run`, `prose skip`, `no command then no exit 2`, `explicit --check still exit 2`, `--no-check opts out`, `explicit check still runs on prose`

## Goal

After a successful apply, CLI `write` runs `check.Load`+`Run` when a command exists, `--no-check` is off, and at least one written path is not prose (`.md`, `.markdown`, `.txt`, `.rst`, `.adoc`). A markdown-only plan in a harnessed tree is Applied, exit 0, does not spawn the check, and records `applied`. A tree with no harness and no `go.mod` still exits 0. Explicit `--check` with no command stays exit 2. Explicit `--check` on prose still runs. `--check` and `--no-check` together is usage. `--dry-run` implies no check; `--check --dry-run` stays usage. MCP unchanged. `command` / `packages` byte-identical.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `writeCmd`: `--no-check`; default run when Load has a command and a written path is not prose. The selector. |
| `cmd/mrw/writecheck_test.go` | create | Red tests for default-on (code), prose skip, opt-out, bare directory, both-flags usage, explicit `--check` on prose. |
| `internal/apply/prose.go` | create | `IsProse` — the one closed extension list both arm 1 and arm 2 read. New file; `apply.go` stays T2's. |
| `scripts/contract.sh` | edit | **§89** — next free after §88. Pair default-check-red on a `.go` file (exit 3, tree kept) with a `.md` apply that does not run the check (exit 0), `--no-check` (exit 0), and a no-go.mod apply (exit 0). |

## Ordered Steps

1. [S1] Write `TestWriteRunsTheCheckByDefault` and confirm it is RED (a Go fixture write without `--check` must run the inferred/declared check). [proof: mutation]
2. [S2] Write `TestWriteOfProseDoesNotRunTheDefaultCheck`, `TestExplicitCheckStillRunsOnProse`, `TestNoCheckOptsOut`, `TestWriteWithoutACheckCommandStillApplies`, and `TestCheckAndNoCheckTogetherIsUsage` — RED until the flags and the prose skip exist. [proof: mutation]
3. [S3] Implement default check / `--no-check` / prose skip in `writeCmd`. Both flags = usage. Dry-run does not run a check unless `--check` was also passed (then usage, as today). Confirm S1–S2 GREEN. Deleting the default-run call must fail S1. Deleting the prose skip must fail `TestWriteOfProseDoesNotRunTheDefaultCheck`. [proof: mutation]
4. [S4] Write §89 against a binary that still opts in `--check` and confirm it is RED, then rebuild and confirm GREEN. The `.md` half must fail if the binary runs the check on prose. [proof: mutation]
5. [S5] Run the scoped tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 89\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v \
    -run 'TestWriteRunsTheCheckByDefault|TestWriteOfProseDoesNotRunTheDefaultCheck|TestExplicitCheckStillRunsOnProse|TestNoCheckOptsOut|TestWriteWithoutACheckCommandStillApplies|TestCheckAndNoCheckTogetherIsUsage' 2>&1 | tee /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestWriteRunsTheCheckByDefault' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestWriteOfProseDoesNotRunTheDefaultCheck' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestExplicitCheckStillRunsOnProse' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestNoCheckOptsOut' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestWriteWithoutACheckCommandStillApplies' /tmp/adr054-t1.out \
  && grep -q '^--- PASS: TestCheckAndNoCheckTogetherIsUsage' /tmp/adr054-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr054-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWriteRunsTheCheckByDefault` | `cmd/mrw/writecheck_test.go` | Write without `--check` on a `.go` file runs the declared check; exit 3 | — | S1, S3 |
| `TestWriteOfProseDoesNotRunTheDefaultCheck` | `cmd/mrw/writecheck_test.go` | Markdown-only write in a harnessed tree does not spawn the check; exit 0 | — | S2, S3 |
| `TestExplicitCheckStillRunsOnProse` | `cmd/mrw/writecheck_test.go` | `--check` on markdown still runs; exit 3 | — | S2, S3 |
| `TestNoCheckOptsOut` | `cmd/mrw/writecheck_test.go` | `--no-check` does not run it; exit 0 even though the check would fail | — | S2, S3 |
| `TestWriteWithoutACheckCommandStillApplies` | `cmd/mrw/writecheck_test.go` | No harness, no `go.mod`: exit 0, not 2 | — | S2, S3 |
| `TestCheckAndNoCheckTogetherIsUsage` | `cmd/mrw/writecheck_test.go` | Both flags → exit 2; file untouched | — | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the six tests and §89 |
| 2 — something selects it | `writeCmd` after apply; deleting the default-run call fails S1 and §89; deleting the prose skip fails `TestWriteOfProseDoesNotRunTheDefaultCheck` |
| 3 — the caller can discover it | T4 teaches `--no-check` and the prose skip on `write --help` |
| 4 — it is used | T3's `failed_check` column; ADR-009 refused telemetry |

## Mutation Log
(empty until execute)
- 2026-09-13 · b68df7d* · mutant killed · exit 1 · `cmd/mrw/main.go` · the cover gate is inverted: the default check runs on a prose-only plan and not on a .go write, so TestWriteRunsTheCheckByDefault (exit 0, want 3) and TestWriteOfProseDoesNotRunTheDefaultCheck (exit 3, want 0) must both go red · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5
- 2026-09-13 · b68df7d* · mutant killed · exit 1 · `cmd/mrw/main.go` · the prose skip is deleted: every written path counts as code, so a markdown-only plan spawns the check and TestWriteOfProseDoesNotRunTheDefaultCheck (exit 3, want 0) must go red · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5

## Invariants

- ADR-003: exit 3 does not revert; missing demanded check is still exit 2.
- Explicit `--check --dry-run` stays usage.
- MCP write does not grow a check flag.
- `internal/check.command` and `packages` stay byte-identical.
- Prose skip records `applied`, not `check_not_run`.
- The prose list is the Decision's five extensions; do not add `.json` / empty-ext / `.rs` here.

## Risks

- A Zeus `.rs` write pays the whole-project `Check` because `packages()` cannot map it: full workspace clippy plus 3096 tests, measured 2026-09-13 at 58–107 seconds for the nextest half alone. Named in the parent Consequences; `--no-check` is the escape, not a Rust mapper.
- Inferred `go test ./...` on a dirty sibling package can fail a write that did not touch it. That is today's `--check` behaviour; do not special-case it here.

## Stop Condition

If the only way to go green is to run a check when `Load` has no command (Desktop exit 2), stop — that fork was rejected.
If the only way to go green is to spawn the check on a markdown-only plan, stop — that is the blocking review finding.

## Out of Scope

- Balance delta (T2)
- stats rendering (T3)
- Teaching (T4)
- MCP check
- A harness `covers` glob / a non-Go `packages()`

## Verification Log
(empty until execute)
- 2026-09-13 · b68df7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:839 · test-lock-sha256:edad48a112a65a8266624fda782a0008319c74d1da9d9a777f7091d3aca29f98 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwp1bnByb3ZlbgljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0Q2hlY2tBbmROb0NoZWNrVG9nZXRoZXJJc1VzYWdlCnVucHJvdmVuCWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RFeHBsaWNpdENoZWNrU3RpbGxSdW5zT25Qcm9zZQp1bnByb3ZlbgljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0Tm9DaGVja09wdHNPdXQKdW5wcm92ZW4JY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdFdyaXRlT2ZQcm9zZURvZXNOb3RSdW5UaGVEZWZhdWx0Q2hlY2sKdW5wcm92ZW4JY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdFdyaXRlUnVuc1RoZUNoZWNrQnlEZWZhdWx0CnVucHJvdmVuCWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RXcml0ZVdpdGhvdXRBQ2hlY2tDb21tYW5kU3RpbGxBcHBsaWVzCnVucHJvdmVuCXNjcmlwdHMvY29udHJhY3Quc2gJwqc4OQ
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
  === RUN   TestNoCheckOptsOut
      writecheck_test.go:125: --no-check exited 2, want 0:
  --- FAIL: TestNoCheckOptsOut (0.00s)
  === RUN   TestWriteWithoutACheckCommandStillApplies
  --- PASS: TestWriteWithoutACheckCommandStillApplies (0.00s)
  === RUN   TestCheckAndNoCheckTogetherIsUsage
  --- PASS: TestCheckAndNoCheckTogetherIsUsage (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.356s
  FAIL
  ```
- 2026-09-13 · b68df7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:1155
- 2026-09-13 · b68df7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:509
- 2026-09-13 · b68df7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:935
- 2026-09-24 · c940c54* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:0 · test-lock-sha256:665831fe49f2cac9428975eabf643e3166052e157430f533a8008263b4a25eb1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RDaGVja0FuZE5vQ2hlY2tUb2dldGhlcklzVXNhZ2UJMTEyOTVhMzRkNzY5NThiZWU5ZjAwYzdhYzk2MDlhMjczYmUxYTE5OGY4Y2U4ZmEyNDU5YzdjM2IxZjEzNTE4MQpib2R5CWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RFeHBsaWNpdENoZWNrU3RpbGxSdW5zT25Qcm9zZQk0OGM1MWUzYTY1YTIzZWNiZDE2ZjhjYTI1NjlmYjNlMTg1NDA4NGRhNDQzODVhNmUwNjg0ZWEzYzBkMmUzYzljCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdEZhaWxlZENoZWNrUHJpbnRzVGhlTGFzdEVycm9yTGluZQkxMDI4MmVmM2I5MjQ1M2U2YmQwYTQ1OTgzMzUwODFlODI2YmQwNDcwMTE3YTZlZGMwMzE5YjFlMDY2OWVjYTNmCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdE5vQ2hlY2tPcHRzT3V0CTljNmJmYzVlODNmNmUzZGJhODExODE1YjdlNDYzZjQ0ODQ5ODVlZTFiYjIyZmFmNGQ5NTZjN2JjZjRiZDA0MGMKYm9keQljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0V3JpdGVPZlByb3NlRG9lc05vdFJ1blRoZURlZmF1bHRDaGVjawllNzg5ZTk2ZGIwYTY2NDc2NGEwOWI0ZGUyMzNiZWQyYWUwN2ZjOWMzODFhMWY0MmM5ZTQwMzIwYmE5ZWJjZmQxCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdFdyaXRlUnVuc1RoZUNoZWNrQnlEZWZhdWx0CTg5MWM2M2I5ZmZlN2ZiMTUzZWU2NGYwODAzNmU2NjUzZDE2NWY3ODA0ODdhMTdlNzE0Njg5ZGY5MjBkYzNjMmUKYm9keQljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0V3JpdGVXaXRob3V0QUNoZWNrQ29tbWFuZFN0aWxsQXBwbGllcwlkYTJmZDk3MDY0ZTEyNDA3NTA3NzFmZDRhMjRkYzA4ZGJlNDA4ZWE5MTE3NDYyNWExZjM3NjhjZjdkMzFlODRlCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JZmFpbAljODBlYmM3MWQyOTQwMTZmOWRhNDg4NTUzMzg4MWU2OWZiZWJlNmYyMTE1MDE3ZDc1MTExOWVkNjRhNDNhNjYzCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JcGFzcwlhYjBlZmM5NzQ3YTIzNWRlZGNiOTczMjA5MGU0OGVjZDc2MzM5NzI1NDk3NTg4ZDI0MTljMDdjMmU3MTM5Njhi · test-lock-kind:replace
- 2026-09-24 · c940c54* · exit 0 · `set -o pipefail …` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:3677
- 2026-09-26 · 31fe531* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:0 · test-lock-sha256:944cfc0f12d2b799a1885862e72ea99802d33d498d16ddb743cf4e154344820a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RDaGVja0FuZE5vQ2hlY2tUb2dldGhlcklzVXNhZ2UJMTEyOTVhMzRkNzY5NThiZWU5ZjAwYzdhYzk2MDlhMjczYmUxYTE5OGY4Y2U4ZmEyNDU5YzdjM2IxZjEzNTE4MQpib2R5CWNtZC9tcncvd3JpdGVjaGVja190ZXN0LmdvCVRlc3RFeHBsaWNpdENoZWNrU3RpbGxSdW5zT25Qcm9zZQk1NjcyMGRjODU1NTE0Mzg5N2NjZTc5M2NiMTgxODIzMTAwMGQ1ZGRkMmMzYjllOWFiMzAxNWE3NGQ4OWU2MWJiCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdEZhaWxlZENoZWNrUHJpbnRzVGhlTGFzdEVycm9yTGluZQliOTE0OGM3ZWM3YmE2ZmRhNDg1Y2I2MzI3YTM2YzgzZWQ3ZjU3YmE2NDM4ODUwNGIyNTBjM2U2NzBlYTkwN2RmCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdE5vQ2hlY2tPcHRzT3V0CTljNmJmYzVlODNmNmUzZGJhODExODE1YjdlNDYzZjQ0ODQ5ODVlZTFiYjIyZmFmNGQ5NTZjN2JjZjRiZDA0MGMKYm9keQljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0V3JpdGVPZlByb3NlRG9lc05vdFJ1blRoZURlZmF1bHRDaGVjawllNzg5ZTk2ZGIwYTY2NDc2NGEwOWI0ZGUyMzNiZWQyYWUwN2ZjOWMzODFhMWY0MmM5ZTQwMzIwYmE5ZWJjZmQxCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JVGVzdFdyaXRlUnVuc1RoZUNoZWNrQnlEZWZhdWx0CTUxYTAwMjZlMTBjMmEzZThhODhmNTc1ZTBhMmRkMDA2ZmY4ZmNhYjU3M2QwNTE5N2UwOTE5ZmYzMDAxZjJmYTAKYm9keQljbWQvbXJ3L3dyaXRlY2hlY2tfdGVzdC5nbwlUZXN0V3JpdGVXaXRob3V0QUNoZWNrQ29tbWFuZFN0aWxsQXBwbGllcwlkYTJmZDk3MDY0ZTEyNDA3NTA3NzFmZDRhMjRkYzA4ZGJlNDA4ZWE5MTE3NDYyNWExZjM3NjhjZjdkMzFlODRlCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JZmFpbAljODBlYmM3MWQyOTQwMTZmOWRhNDg4NTUzMzg4MWU2OWZiZWJlNmYyMTE1MDE3ZDc1MTExOWVkNjRhNDNhNjYzCmJvZHkJY21kL21ydy93cml0ZWNoZWNrX3Rlc3QuZ28JcGFzcwlhYjBlZmM5NzQ3YTIzNWRlZGNiOTczMjA5MGU0OGVjZDc2MzM5NzI1NDk3NTg4ZDI0MTljMDdjMmU3MTM5Njhi · test-lock-kind:replace
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:1187
- 2026-09-26 · 31fe531* · exit 0 · `set -o pipefail …` · acceptance-sha256:44ecb5fe0b5ec4a0422dfa6765dbc0a4e7bbf7e6917f54021aa714b95fd667f5 · ms:768
