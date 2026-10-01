# Task ADR-111-T1: the CLI's receipt keys are listed and held

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `docs/receipts.txt`; `statsReceipt`, `checkRefusal`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the CLI's receipt keys are listed and held`

## Goal

`docs/receipts.txt` lists every key path of the CLI's JSON receipts — `write --json`, `check --json` and its refusal, `stats --json` — and a test reflects the receipt types and requires the file and the types to agree. The stats receipt and the check refusal become named types with the same JSON. Contract §210 checks the built binary's `write --json` keys against the file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/receipts.txt` | add | the shipped key paths |
| `cmd/mrw/main.go` | edit | `statsReceipt`, `checkRefusal` |
| `cmd/mrw/receipts111_test.go` | add | the test |
| `scripts/contract.sh` | edit | §210 |

## Ordered Steps

1. [S1] Write the failing test(s) `TestNoShippedReceiptFieldDisappears`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal; contract §210 through `$MRW`. Mutants: a write receipt key renamed (`pattern`); a stats key renamed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestNoShippedReceiptFieldDisappears' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestNoShippedReceiptFieldDisappears \(' "$out" \
  && grep -q '^# 210\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestNoShippedReceiptFieldDisappears` | `cmd/mrw/receipts111_test.go` | every path of `receipt`, `checkReceipt`, `checkRefusal` and `statsReceipt` is in `docs/receipts.txt`, every listed path of those receipts is still in a type, and no receipt lists nothing | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every receipt the CLI or MCP prints |
| 3 — the caller can discover it | `docs/receipts.txt` names every key a caller may rely on |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review (B4) |

## Mutation Log
- 2026-10-01 · 5ad968f* · mutant killed · exit 1 · `cmd/mrw/main.go` · a write receipt key renamed: pattern disappears and patterns is unlisted · acceptance-sha256:96d95ad05fab07be186b5c29db8f71226428b678cb4e31cdfb096d197950e701
- 2026-10-01 · 5ad968f* · mutant killed · exit 1 · `cmd/mrw/main.go` · a stats key renamed: landed disappears · acceptance-sha256:96d95ad05fab07be186b5c29db8f71226428b678cb4e31cdfb096d197950e701

## Invariants

- Every receipt's JSON is byte-for-byte what it was.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-111 task, in its own file.

## Verification Log
- 2026-10-01 · 5ad968f* · exit 1 · `set -o pipefail …` · acceptance-sha256:96d95ad05fab07be186b5c29db8f71226428b678cb4e31cdfb096d197950e701 · ms:466 · test-lock-sha256:f3f7a29fd673b36eab118f2d07fd896565e9bf6d892695658b30895a30e4390c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9yZWNlaXB0czExMV90ZXN0LmdvCVRlc3ROb1NoaXBwZWRSZWNlaXB0RmllbGREaXNhcHBlYXJzCWM2ZDY3MWVhZDk1NTA3OTRiMDllZDlhNTI3MjIwYWRmYjhjYjM3OGU5NTcyZmM5ZTYyOGRmODFmOGE0M2I4ZGU
  ```
  --- last 5 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw.test]
  cmd/mrw/receipts111_test.go:20:35: undefined: checkRefusal
  cmd/mrw/receipts111_test.go:21:35: undefined: statsReceipt
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw [build failed]
  FAIL
  ```
- 2026-10-01 · 5ad968f* · exit 0 · `set -o pipefail …` · acceptance-sha256:96d95ad05fab07be186b5c29db8f71226428b678cb4e31cdfb096d197950e701 · ms:1600
- 2026-10-01 · 5ad968f* · exit 0 · `set -o pipefail …` · acceptance-sha256:96d95ad05fab07be186b5c29db8f71226428b678cb4e31cdfb096d197950e701 · ms:1736
