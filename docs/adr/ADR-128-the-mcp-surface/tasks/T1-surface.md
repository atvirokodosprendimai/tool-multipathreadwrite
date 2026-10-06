# Task ADR-128-T1: a BOM read, force advice dropped, unknown acks named, a cancel that stops a check

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `apply.Options.NoForce`, the call context, the unknown-ack note
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a BOM read, force advice dropped, unknown acks named, a cancel that stops a check`

## Goal

The four MCP-surface fixes of the record, each with a test that fails before it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | BOM; the cancel registry and the notification |
| `internal/mcp/tools.go` | edit | the call context through Verify; NoForce; the ack note |
| `internal/mcp/ack.go` | edit | `promote` returns the ids it did not match |
| `internal/apply/apply.go` | edit | `Options.NoForce`, the force clause dropped |
| `internal/mcp/surface128_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §226 |
| `AGENTS.md` | edit | the MCP paragraph |

## Ordered Steps

1. [S1] Write `TestABOMOnTheFirstLineStillInitializes`, `TestAnMCPRefusalDoesNotAdviseForce`, `TestAnUnknownAckIsNamed`, `TestACancelStopsAWritesRunningCheck`. Confirm RED. [proof: mutation]
2. [S2] The four fixes. Mutants: the BOM kept; NoForce not set by mrw_write; the ack note dropped; the cancel not wired to the call's context. [proof: mutation]
3. [S3] AGENTS.md and contract §226. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestABOMOnTheFirstLineStillInitializes|TestAnMCPRefusalDoesNotAdviseForce|TestAnUnknownAckIsNamed|TestACancelStopsAWritesRunningCheck' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestABOMOnTheFirstLineStillInitializes \(' "$out" \
  && grep -qE '^--- PASS: TestAnMCPRefusalDoesNotAdviseForce \(' "$out" \
  && grep -qE '^--- PASS: TestAnUnknownAckIsNamed \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestACancelStopsAWritesRunningCheck \(' "$out" \
  && go test ./internal/mcp/ ./internal/apply/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 226\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestABOMOnTheFirstLineStillInitializes` | `internal/mcp/surface128_test.go` | an initialize line with a BOM is answered, not -32700 | none | S1, S2 |
| `TestAnMCPRefusalDoesNotAdviseForce` | `internal/mcp/surface128_test.go` | an mrw_write on an unread file is refused without "--force"; the CLI's words keep it | none | S1, S2 |
| `TestAnUnknownAckIsNamed` | `internal/mcp/surface128_test.go` | an ack id matching nothing is named in the answer's text | none | S1, S2 |
| `TestACancelStopsAWritesRunningCheck` | `internal/mcp/surface128_test.go` | a cancel during a write's `sleep` check answers inside seconds, the check interrupted | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four behaviours |
| 2 — something selects it | `handle` for every line; `writeTool` for every write; `promote` for every ack |
| 3 — the caller can discover it | the answers; AGENTS.md |
| 4 — it is used | the Windows peers and ADR-121's review met them; no telemetry (ADR-009) |

## Invariants

- The CLI's words and behaviour are unchanged.
- No receipt key changes.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a cancel would need to stop a write before it lands.

## Out of Scope

- `written: true` for a byte-identical body (permanent: boundary: the record's)

## Mutation Log
- 2026-10-06 · 5f21dcb* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: NoForce not set by mrw_write · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450
- 2026-10-06 · 5f21dcb* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the ack note dropped · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450
- 2026-10-06 · 5f21dcb* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S2: the cancel not wired to the call context · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450
- 2026-10-06 · 5f21dcb* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S2: the BOM kept (trimmed from the wrong end) · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450

## Verification Log
- 2026-10-06 · 5f21dcb* · exit 1 · `set -o pipefail …` · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450 · ms:100 · test-lock-sha256:d093d3b309533746d7ae8fdcf1063c6b6ab9449d0af3267000843495bfbce162 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3N1cmZhY2UxMjhfdGVzdC5nbwlUZXN0QUJPTU9uVGhlRmlyc3RMaW5lU3RpbGxJbml0aWFsaXplcwk2YjJmMTk2NzljMzM4MDBjMmQ5ZGJmZTBkMDE4NGM2OTJjMTEyNDU3YmFhZDYyZjA4NGE5NDYzYmU3MDMwNTgxCmJvZHkJaW50ZXJuYWwvbWNwL3N1cmZhY2UxMjhfdGVzdC5nbwlUZXN0QUNhbmNlbFN0b3BzQVdyaXRlc1J1bm5pbmdDaGVjawkzZmNiNDAwYjQ5MDEwYTgwNWQ0ODQwMzdkNzMxMjg4ZmQzNWIwNTE4ZDY0MjkxYTQ3ZTQxZDhkNjM3ZDBiNWMxCmJvZHkJaW50ZXJuYWwvbWNwL3N1cmZhY2UxMjhfdGVzdC5nbwlUZXN0QW5NQ1BSZWZ1c2FsRG9lc05vdEFkdmlzZUZvcmNlCTE5NDZlMjkwYjY3ZjNhN2JkZjAwOGViYWM0NzU0OWJiM2U2N2VhYzkxZjU2OTY5ZjJhMjA0ZTNhOTEyZjNjYWUKYm9keQlpbnRlcm5hbC9tY3Avc3VyZmFjZTEyOF90ZXN0LmdvCVRlc3RBblVua25vd25BY2tJc05hbWVkCTc5Yjk3YmRkYzhlMTAxZWI0OWJmMGE4YThlNzJjNjVhODExNTBlYTc5ZTQ4YWM4YjNhYzAyZTczYjk1ZWY0NjQ
  ```
  --- last 4 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp
  internal/mcp/surface128_test.go:16:11: illegal byte order mark
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [setup failed]
  FAIL
  ```
- 2026-10-06 · 5f21dcb* · exit 1 · `set -o pipefail …` · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450 · ms:30769
  ```
  --- last 10 line(s) of stdout (of 25 after folding 25 raw)
          -- ck af785d0328e4135e close
          -- This serve licenses NOTHING until you acknowledge it.
          -- Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end.
  --- FAIL: TestAnUnknownAckIsNamed (0.01s)
  === RUN   TestACancelStopsAWritesRunningCheck
      surface128_test.go:61: the cancelled write answered after 30.037815875s: the check ran to its end
  --- FAIL: TestACancelStopsAWritesRunningCheck (30.05s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	30.306s
  FAIL
  ```
- 2026-10-06 · 5f21dcb* · exit 0 · `set -o pipefail …` · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450 · ms:39853
- 2026-10-06 · 5f21dcb* · exit 0 · `set -o pipefail …` · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450 · ms:40326
- 2026-10-06 · 5f21dcb* · exit 0 · `set -o pipefail …` · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450 · ms:41062
- 2026-10-06 · 5f21dcb* · exit 0 · `set -o pipefail …` · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450 · ms:38405
- 2026-10-06 · 5f21dcb* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:d31abce880c7f9b321812d8160af0eda51f12d6d01247864d4476dfd3ec03450 · ms:0 · test-lock-sha256:8540ca293da440a8ada5be31ddbec880822809c6189d9c5e9ebf6e3c0b75365b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3N1cmZhY2UxMjhfdGVzdC5nbwlUZXN0QUJPTU9uVGhlRmlyc3RMaW5lU3RpbGxJbml0aWFsaXplcwkxMzRjMTM5OWU3YjkxNWI1OWI5ZTRlOTQxNzAyNzZjZjA0MzVhM2M0MjE5YWVmMmNkOTkyMmNkOWFiYzY0NTE0CmJvZHkJaW50ZXJuYWwvbWNwL3N1cmZhY2UxMjhfdGVzdC5nbwlUZXN0QUNhbmNlbFN0b3BzQVdyaXRlc1J1bm5pbmdDaGVjawkzZmNiNDAwYjQ5MDEwYTgwNWQ0ODQwMzdkNzMxMjg4ZmQzNWIwNTE4ZDY0MjkxYTQ3ZTQxZDhkNjM3ZDBiNWMxCmJvZHkJaW50ZXJuYWwvbWNwL3N1cmZhY2UxMjhfdGVzdC5nbwlUZXN0QW5NQ1BSZWZ1c2FsRG9lc05vdEFkdmlzZUZvcmNlCTE5NDZlMjkwYjY3ZjNhN2JkZjAwOGViYWM0NzU0OWJiM2U2N2VhYzkxZjU2OTY5ZjJhMjA0ZTNhOTEyZjNjYWUKYm9keQlpbnRlcm5hbC9tY3Avc3VyZmFjZTEyOF90ZXN0LmdvCVRlc3RBblVua25vd25BY2tJc05hbWVkCTc5Yjk3YmRkYzhlMTAxZWI0OWJmMGE4YThlNzJjNjVhODExNTBlYTc5ZTQ4YWM4YjNhYzAyZTczYjk1ZWY0NjQ · test-lock-kind:replace
