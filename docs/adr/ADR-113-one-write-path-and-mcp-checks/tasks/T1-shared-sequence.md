# Task ADR-113-T1: the write sequence is shared, and the CLI's outcomes do not move

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `writer.Prepare`, `Prepared.Land`, `Landed.Verify`, `Landed.Settle`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the write sequence is shared, and the CLI's outcomes do not move`

## Goal

Move the CLI write's gates, landing count, check, drift, steps, reclassification and pricing out of `cmd/mrw/main.go` into `internal/writer/flow.go` as four phases — `Prepare`, `Land`, `Verify`, `Settle` — returning facts, with the CLI rendering them exactly as before: the human receipt before the check, the `--json` receipt before `Settle` counts and prices.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/flow.go` | add | the four phases, `CheckMode`, `Refusal`, and the helpers moved from `cmd/mrw` that `mrw check` shares (`ResolveSteps`, `RunSteps`, `StepStop`, `Shown`) or only the sequence needs (`CheckPaths`, `touchesCode`) |
| `internal/writer/flow113_test.go` | add | `TestLandCountsEachOutcomeOnce`, and `TestCheckPathsKeepsRenameSourcePackage` moved from `cmd/mrw/pathop_check_test.go` |
| `cmd/mrw/main.go` | edit | the write action calls the phases and renders; `gateRefusal`; `mrw check` calls the moved step helpers |
| `cmd/mrw/outcomes113_test.go`, `cmd/mrw/testdata/outcomes113.golden` | add | `TestTheCLIWriteOutcomesAreUnchanged` and the golden generated from `95f5dab`'s code |
| `cmd/mrw/pathop_check_test.go` | edit | its `writeCheckPaths` test moves with the function |
| `scripts/fence-prose.py` | edit | ADR-075 T1's `writer\.Apply(` clause exempted, naming ADR-113 |

## Ordered Steps

1. [S1] Write `TestTheCLIWriteOutcomesAreUnchanged` and generate its golden from the code BEFORE the move (`95f5dab`), deterministic across two runs: one row per outcome of the write action — refusals before the plan is read (contradictory flags, an empty `--then-sh`, an unreadable plan, an unknown format), refusals as a document (native, `apply_patch`, `search_replace`, a missing `body=@` file), refusals after parsing (an unresolved pointer, a non-UTF-8 path under `--json`, a malformed harness, an undeclared step, the depth limit, an unloadable ledger, an unread line, a refused dry run, a plan naming a directory), and landings (a partial commit; a ledger failure with no check, with a check due, with a step; a clean dry run; prose with no check due; `--no-check`; no harness; a check passed, failed, demanded on prose, demanded with none declared, unable to run, unable to start, timed out, inferred from `go.mod`, scoped over an unlink and a rename; drift; steps passed, named, failed, timed out, after a failed check, unable to start; `--strict-balance` advisory and refused; pricing would-refuse held and broke; the pattern line) — each in `--json` and human form, pinning the exit code, the `stats --json` counts and pricing, and the whole stdout with run-specific paths and durations normalized. Not driven, named in the test: a signal during a check or step, mrw killed during its check (its own test), and a failing `--json` encoder. Then write `TestLandCountsEachOutcomeOnce` against the new API; confirm RED. [proof: mutation]
2. [S2] Add `internal/writer/flow.go` and move the CLI onto it; `Settle` runs after the CLI renders. Mutants, each killed by the CLI's golden (which proves the CLI is wired through the sequence, since the writer unit test does not drive these rows): `Land` stops counting a partial commit; `Settle` stops moving a failed check to `failed_check`; `Prepare` skips the depth refusal. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/writer/ ./cmd/mrw/ -count=1 -timeout 600s -run 'TestLandCountsEachOutcomeOnce|TestTheCLIWriteOutcomesAreUnchanged' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestLandCountsEachOutcomeOnce \(' "$out" \
  && grep -qE '^--- PASS: TestTheCLIWriteOutcomesAreUnchanged \(' "$out" \
  && grep -q '^func Prepare(' internal/writer/flow.go \
  && grep -q 'writer\.Prepare(' cmd/mrw/main.go \
  && grep -q 'land\.Settle(v)' cmd/mrw/main.go \
  && ! grep -q 'checkWanted := func' cmd/mrw/main.go \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheCLIWriteOutcomesAreUnchanged` | `cmd/mrw/outcomes113_test.go` | every outcome of `mrw write` keeps its exit code, stdout, tally and pricing across the move, against a golden generated from the code before it | none | S1, S2 |
| `TestLandCountsEachOutcomeOnce` | `internal/writer/flow113_test.go` | `Prepare` counts a harness refusal, `Land` an applied, refused and dry-run landing exactly once, and `Verify`/`Settle` move an applied landing to `failed_check` or `check_not_run` | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/writer/flow.go` |
| 2 — something selects it | every CLI write calls the three phases |
| 3 — the caller can discover it | not caller-facing: the CLI's output is unchanged by design |
| 4 — it is used | T2 is the second caller |

## Mutation Log
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/writer/flow.go` · Land stops counting a partial commit · acceptance-sha256:d6253be43a9fc9d8658c623e2c20b938cd518729088f262b7c7fbf14d011ea06
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/writer/flow.go` · Verify stops moving a failed check to failed_check · acceptance-sha256:d6253be43a9fc9d8658c623e2c20b938cd518729088f262b7c7fbf14d011ea06
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/writer/flow.go` · Prepare skips the depth refusal · acceptance-sha256:d6253be43a9fc9d8658c623e2c20b938cd518729088f262b7c7fbf14d011ea06
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/writer/flow.go` · Land stops counting a partial commit · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/writer/flow.go` · Settle stops moving a failed check to failed_check · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `internal/writer/flow.go` · Prepare skips the depth refusal · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9
- 2026-10-01 · 95f5dab* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI never settles: a failed check stays applied · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9

## Invariants

- Every CLI exit code, receipt value, tally, pricing count and output line is what it was; the commit that lands T1 edits no row of `scripts/contract.sh`, and `contract.sh` passes unedited.

## Risks

- An outcome the characterization table misses moves unseen — mitigated by enumerating the rows from the write action's branches and the 2026-10-01 Codex review of this record, which listed the ones a first table missed.

## Stop Condition

Stop and ask if a CLI outcome must change to make the sequence shareable.

## Out of Scope

- The MCP surface (permanent: boundary: T2)

## Verification Log
- 2026-10-01 · 95f5dab* · exit 1 · `set -o pipefail …` · acceptance-sha256:d6253be43a9fc9d8658c623e2c20b938cd518729088f262b7c7fbf14d011ea06 · ms:3170 · test-lock-sha256:31b6d8a1d770d570248fda4f64e62638ad84a038b54be57eb34a44078925da12 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9vdXRjb21lczExM190ZXN0LmdvCVRlc3RUaGVDTElXcml0ZU91dGNvbWVzQXJlVW5jaGFuZ2VkCWRjNDcwMTY5ZDRhOGYyMzdlZWJjODRjNGJiZTgwYWVjNmU5ZjlkMjRkMmUxNmZjN2I2MTQ4ZWNlNjQyMTllMTgKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCVRlc3RMYW5kQ291bnRzRWFjaE91dGNvbWVPbmNlCWQ5MGNmMWEzYzM3OGQ0NmJlODIyOGNhYjU0NWUyNTNlZWRiYTRlYzQxYjY3ZmUzYTc3OTQ4MGJkNGM5YjkyNWYKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgY2hlY2sgdGhhdCBjYW5ub3QgcnVuIG1vdmVzIHRoZSBsYW5kaW5nIHRvIGNoZWNrX25vdF9ydW4JZWE3ZGJkYWUxZDA1Mzc0MzRiMjc4YWIwNTk5MGY0Y2ZhYThiNmE0N2VkZWY4OTQ5NGQ4ZmJkYmQ2MmI5ZjcwNwpib2R5CWludGVybmFsL3dyaXRlci9mbG93MTEzX3Rlc3QuZ28JYSBkcnkgcnVuIGNvdW50cyBub3RoaW5nCTE3YTIwNjk1NmUxZDFmNDZjZDk0ODdhYWMwNzY0OGIwMmZiY2MyNTBhOWQ3MTYzM2ZjOGM5NDI1MmU0M2UxNzAKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgZmFpbGVkIGNoZWNrIG1vdmVzIHRoZSBsYW5kaW5nIHRvIGZhaWxlZF9jaGVjawkwNWZjMjE1MzNiMDZhNWY0NzU0YzBmZGQyNmQwNWU5N2NiMGQzODYwYjI0MTYxMzcyMWEzZjRlMjhhZWJkYzc5CmJvZHkJaW50ZXJuYWwvd3JpdGVyL2Zsb3cxMTNfdGVzdC5nbwlhIGxhbmRpbmcgd2l0aCBubyBjaGVjayBpcyBhcHBsaWVkCTRmZmEwMzk4ODc3Y2Y1N2YzMmVmZjFlNmViY2I5ZGJjMmRkMzA3MDg5YmQ4YzdlNGNhMjkwMTBlYjdlMGNjZTkKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgbWFsZm9ybWVkIGhhcm5lc3MgaXMgcmVmdXNlZCBiZWZvcmUgYW55dGhpbmcgaXMgd3JpdHRlbgkwZGNlMmUzMzA5MDE2ZDNjOTdlZjIyNTJlMmFmMDBjZmVlOTU0ZTE4NGFlOGJiMjM5N2VlZjIzMGViNGNlNTU4CmJvZHkJaW50ZXJuYWwvd3JpdGVyL2Zsb3cxMTNfdGVzdC5nbwlhbiB1bnJlYWQgbGluZSBpcyByZWZ1c2VkX2FwcGx5CTE2NDI4N2JlYTdjNGMwNWMzYzA0OWFhZjE4ZjcxNjI0OTAyNWVmYmQzODNmYzA5YWUyYTAwYTY0YjBmMGU4NjA
  ```
  --- last 10 line(s) of stdout (of 18 after folding 18 raw)
  internal/writer/flow113_test.go:71:40: undefined: StageHarness
  internal/writer/flow113_test.go:83:20: undefined: Request
  internal/writer/flow113_test.go:83:59: undefined: CheckOff
  internal/writer/flow113_test.go:83:59: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer [build failed]
  === RUN   TestTheCLIWriteOutcomesAreUnchanged
  --- PASS: TestTheCLIWriteOutcomesAreUnchanged (2.71s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	2.812s
  FAIL
  ```
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:d6253be43a9fc9d8658c623e2c20b938cd518729088f262b7c7fbf14d011ea06 · ms:4122
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:d6253be43a9fc9d8658c623e2c20b938cd518729088f262b7c7fbf14d011ea06 · ms:3855
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:d6253be43a9fc9d8658c623e2c20b938cd518729088f262b7c7fbf14d011ea06 · ms:3828
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9 · ms:7342
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9 · ms:7281
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9 · ms:6898
- 2026-10-01 · 95f5dab* · exit 0 · `set -o pipefail …` · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9 · ms:6636
- 2026-10-01 · 95f5dab* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9 · ms:0 · test-lock-sha256:1422156a9328bc89122b794f7de3135f5435bdc522f9183b2a0591168d30a6d6 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9vdXRjb21lczExM190ZXN0LmdvCVRlc3RUaGVDTElXcml0ZU91dGNvbWVzQXJlVW5jaGFuZ2VkCWRjNDcwMTY5ZDRhOGYyMzdlZWJjODRjNGJiZTgwYWVjNmU5ZjlkMjRkMmUxNmZjN2I2MTQ4ZWNlNjQyMTllMTgKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCVRlc3RDaGVja1BhdGhzS2VlcHNSZW5hbWVTb3VyY2VQYWNrYWdlCWZlNjMzMTlhYjk0OWYxYTNhOTYyNGFjM2MzYmJhNWIyZDMyYTNjM2RmNzkyODU4MmRkZTFiNjNjMTkyNGNkZjgKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCVRlc3RMYW5kQ291bnRzRWFjaE91dGNvbWVPbmNlCWU1ZTYxNjEwMjA5YTQzMjhkNTcxMGYyZjBlOGRlMTI4Y2QxNTgwNTU2ZGJmMGRlZDg4NjVmZWFkNDljMzI2NjMKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgY2hlY2sgdGhhdCBjYW5ub3QgcnVuIG1vdmVzIHRoZSBsYW5kaW5nIHRvIGNoZWNrX25vdF9ydW4JZWE3ZGJkYWUxZDA1Mzc0MzRiMjc4YWIwNTk5MGY0Y2ZhYThiNmE0N2VkZWY4OTQ5NGQ4ZmJkYmQ2MmI5ZjcwNwpib2R5CWludGVybmFsL3dyaXRlci9mbG93MTEzX3Rlc3QuZ28JYSBkcnkgcnVuIGNvdW50cyBub3RoaW5nCTE3YTIwNjk1NmUxZDFmNDZjZDk0ODdhYWMwNzY0OGIwMmZiY2MyNTBhOWQ3MTYzM2ZjOGM5NDI1MmU0M2UxNzAKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgZmFpbGVkIGNoZWNrIG1vdmVzIHRoZSBsYW5kaW5nIHRvIGZhaWxlZF9jaGVjawkwNWZjMjE1MzNiMDZhNWY0NzU0YzBmZGQyNmQwNWU5N2NiMGQzODYwYjI0MTYxMzcyMWEzZjRlMjhhZWJkYzc5CmJvZHkJaW50ZXJuYWwvd3JpdGVyL2Zsb3cxMTNfdGVzdC5nbwlhIGxhbmRpbmcgd2l0aCBubyBjaGVjayBpcyBhcHBsaWVkCTRmZmEwMzk4ODc3Y2Y1N2YzMmVmZjFlNmViY2I5ZGJjMmRkMzA3MDg5YmQ4YzdlNGNhMjkwMTBlYjdlMGNjZTkKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgbWFsZm9ybWVkIGhhcm5lc3MgaXMgcmVmdXNlZCBiZWZvcmUgYW55dGhpbmcgaXMgd3JpdHRlbgkwZGNlMmUzMzA5MDE2ZDNjOTdlZjIyNTJlMmFmMDBjZmVlOTU0ZTE4NGFlOGJiMjM5N2VlZjIzMGViNGNlNTU4CmJvZHkJaW50ZXJuYWwvd3JpdGVyL2Zsb3cxMTNfdGVzdC5nbwlhbiB1bnJlYWQgbGluZSBpcyByZWZ1c2VkX2FwcGx5CTE2NDI4N2JlYTdjNGMwNWMzYzA0OWFhZjE4ZjcxNjI0OTAyNWVmYmQzODNmYzA5YWUyYTAwYTY0YjBmMGU4NjA · test-lock-kind:replace
- 2026-10-01 · 0ee1f95* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9 · ms:0 · test-lock-sha256:beb7dacfc44a7321ba29a5c4abcbe84bf17c77db345b3a53902eb89b61a686ed · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9vdXRjb21lczExM190ZXN0LmdvCVRlc3RUaGVDTElXcml0ZU91dGNvbWVzQXJlVW5jaGFuZ2VkCTBhMWNlZGRiMGVlYzk2YmIzMzdlZTZlNTVlYTkzOGM1MDJmZDI3NDgzMDhiOGE4M2QxM2M4NTRhYTQyZDhkZGEKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCVRlc3RDaGVja1BhdGhzS2VlcHNSZW5hbWVTb3VyY2VQYWNrYWdlCWZlNjMzMTlhYjk0OWYxYTNhOTYyNGFjM2MzYmJhNWIyZDMyYTNjM2RmNzkyODU4MmRkZTFiNjNjMTkyNGNkZjgKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCVRlc3RMYW5kQ291bnRzRWFjaE91dGNvbWVPbmNlCWU1ZTYxNjEwMjA5YTQzMjhkNTcxMGYyZjBlOGRlMTI4Y2QxNTgwNTU2ZGJmMGRlZDg4NjVmZWFkNDljMzI2NjMKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgY2hlY2sgdGhhdCBjYW5ub3QgcnVuIG1vdmVzIHRoZSBsYW5kaW5nIHRvIGNoZWNrX25vdF9ydW4JZWE3ZGJkYWUxZDA1Mzc0MzRiMjc4YWIwNTk5MGY0Y2ZhYThiNmE0N2VkZWY4OTQ5NGQ4ZmJkYmQ2MmI5ZjcwNwpib2R5CWludGVybmFsL3dyaXRlci9mbG93MTEzX3Rlc3QuZ28JYSBkcnkgcnVuIGNvdW50cyBub3RoaW5nCTE3YTIwNjk1NmUxZDFmNDZjZDk0ODdhYWMwNzY0OGIwMmZiY2MyNTBhOWQ3MTYzM2ZjOGM5NDI1MmU0M2UxNzAKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgZmFpbGVkIGNoZWNrIG1vdmVzIHRoZSBsYW5kaW5nIHRvIGZhaWxlZF9jaGVjawkwNWZjMjE1MzNiMDZhNWY0NzU0YzBmZGQyNmQwNWU5N2NiMGQzODYwYjI0MTYxMzcyMWEzZjRlMjhhZWJkYzc5CmJvZHkJaW50ZXJuYWwvd3JpdGVyL2Zsb3cxMTNfdGVzdC5nbwlhIGxhbmRpbmcgd2l0aCBubyBjaGVjayBpcyBhcHBsaWVkCTRmZmEwMzk4ODc3Y2Y1N2YzMmVmZjFlNmViY2I5ZGJjMmRkMzA3MDg5YmQ4YzdlNGNhMjkwMTBlYjdlMGNjZTkKYm9keQlpbnRlcm5hbC93cml0ZXIvZmxvdzExM190ZXN0LmdvCWEgbWFsZm9ybWVkIGhhcm5lc3MgaXMgcmVmdXNlZCBiZWZvcmUgYW55dGhpbmcgaXMgd3JpdHRlbgkwZGNlMmUzMzA5MDE2ZDNjOTdlZjIyNTJlMmFmMDBjZmVlOTU0ZTE4NGFlOGJiMjM5N2VlZjIzMGViNGNlNTU4CmJvZHkJaW50ZXJuYWwvd3JpdGVyL2Zsb3cxMTNfdGVzdC5nbwlhbiB1bnJlYWQgbGluZSBpcyByZWZ1c2VkX2FwcGx5CTE2NDI4N2JlYTdjNGMwNWMzYzA0OWFhZjE4ZjcxNjI0OTAyNWVmYmQzODNmYzA5YWUyYTAwYTY0YjBmMGU4NjA · test-lock-kind:replace
- 2026-10-01 · 0ee1f95* · exit 0 · `set -o pipefail …` · acceptance-sha256:9561e4be0c82ab8abe43a47c12fd9e7a47e580007d3bde814d4ec0720be111e9 · ms:5770
