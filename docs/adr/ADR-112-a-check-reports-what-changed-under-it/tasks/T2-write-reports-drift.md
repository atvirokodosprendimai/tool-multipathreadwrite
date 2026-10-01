# Task ADR-112-T2: a write reports what changed while its check ran

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `drift` in the write receipt and the `drift:` line
**Consumes:** `writer.Drift` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a write reports what changed while its check ran`

## Goal

The CLI write calls `writer.Drift` once its check has run and reports each path — a `drift: <path> changed while the check ran` line after the verdict, and `drift` in the `--json` receipt, absent when empty — leaving the exit code to the check. `write drift` joins `docs/receipts.txt`; AGENTS.md says so.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `receipt.Drift`, the call after the check, `reportDrift` |
| `cmd/mrw/drift112_test.go` | add | the test |
| `docs/receipts.txt` | edit | `write drift` |
| `AGENTS.md` | edit | the exit-3 bullet |
| `scripts/contract.sh` | edit | §211 |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAWriteNamesAFileChangedWhileItsCheckRan`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal; contract §211 through `$MRW`. Mutants: the `Drift` call removed; the receipt key dropped. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAWriteNamesAFileChangedWhileItsCheckRan' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteNamesAFileChangedWhileItsCheckRan \(' "$out" \
  && grep -q '^# 211\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteNamesAFileChangedWhileItsCheckRan` | `cmd/mrw/drift112_test.go` | a `write --check --json` whose check appends to the written file exits 0 with `drift` naming it; one whose check changes nothing carries no `drift` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every CLI write whose check runs |
| 3 — the caller can discover it | the `drift:` line and the `drift` key name each file |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review (B5) |

## Mutation Log
- 2026-10-01 · e396f7e* · mutant killed · exit 1 · `cmd/mrw/main.go` · the Drift call removed: the receipt never names a changed file · acceptance-sha256:8a31e86510bbb661e2b049df24ffe924b6adb8b195780a6621a6a2beae1fa599
- 2026-10-01 · e396f7e* · mutant killed · exit 1 · `cmd/mrw/main.go` · the receipt key dropped · acceptance-sha256:8a31e86510bbb661e2b049df24ffe924b6adb8b195780a6621a6a2beae1fa599

## Invariants

- Exit codes are the check's; a write with no check, or no drift, prints and carries what it did before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-112 task, in its own file.

## Verification Log
- 2026-10-01 · e396f7e* · exit 1 · `set -o pipefail …` · acceptance-sha256:8a31e86510bbb661e2b049df24ffe924b6adb8b195780a6621a6a2beae1fa599 · ms:954 · test-lock-sha256:97dcf5f608b854899adf4281f13ba67892f9b4da21b1ba429d5597ee5c5f8cf0 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9kcmlmdDExMl90ZXN0LmdvCVRlc3RBV3JpdGVOYW1lc0FGaWxlQ2hhbmdlZFdoaWxlSXRzQ2hlY2tSYW4JZGZlNTk4ZmE2OTRhZmQ0YTYyYTEyZDU0NGQzZWJjMDM5YTVlODcyMWMzMjc1ZGU4YTc4OWEwNzMzYWJjYjU5OQ
  ```
  --- last 10 line(s) of stdout (of 46 after folding 46 raw)
            "pattern": {
              "advisory_writes": 0,
              "window": 1,
              "fires": false
            }
          }
  --- FAIL: TestAWriteNamesAFileChangedWhileItsCheckRan (0.08s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.267s
  FAIL
  ```
- 2026-10-01 · e396f7e* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a31e86510bbb661e2b049df24ffe924b6adb8b195780a6621a6a2beae1fa599 · ms:1190
- 2026-10-01 · e396f7e* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a31e86510bbb661e2b049df24ffe924b6adb8b195780a6621a6a2beae1fa599 · ms:1187
