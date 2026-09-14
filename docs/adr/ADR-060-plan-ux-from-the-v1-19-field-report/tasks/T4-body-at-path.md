# Task ADR-060-T4: `body=@path` loads the body from a rooted file

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `Hunk.BodyFile`; `plan.LoadBodyFiles`; §103
**Consumes:** T1 leftover (message may name `body=@path`)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `body=@ loads lines`, `rooted path refused`, `outside-root refused`, `CLI and MCP both call LoadBodyFiles`

## Goal

`body=@root-relative-path` fills the hunk body from that file. Rooted and outside-root paths refuse by name. CLI and MCP load before apply.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | edit | `BodyFile`; parse `@`; `LoadBodyFiles`. |
| `internal/plan/bodyfile_test.go` | create | Red: load, rooted refuse, outside refuse. |
| `cmd/mrw/main.go` | edit | Call `LoadBodyFiles` after parse. |
| `internal/mcp/tools.go` | edit | Same call — the selector on this transport. |
| `scripts/contract.sh` | edit | **§103**. |

## Ordered Steps

1. [S1] Write the failing tests. [proof: mutation]
2. [S2] Parse `body=@` and `LoadBodyFiles`. S1 GREEN. [proof: mutation]
3. [S3] Wire CLI and MCP. [proof: mutation]
4. [S4] §103 RED then GREEN. [proof: mutation]
5. [S5] Scoped tests, `gofmt`, `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 103\. ' scripts/contract.sh \
  && go test ./internal/plan/ ./cmd/mrw/ -count=1 -v \
    -run 'TestBodyAtPathLoadsTheFile|TestWriteBodyAtPathCreatesFromFile' 2>&1 | tee /tmp/adr060-t4.out \
  && grep -q '^--- PASS: TestBodyAtPathLoadsTheFile' /tmp/adr060-t4.out \
  && grep -q '^--- PASS: TestWriteBodyAtPathCreatesFromFile' /tmp/adr060-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr060-t4.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/plan/ ./cmd/mrw/ ./internal/mcp/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/seen internal/check internal/state \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestBodyAtPathLoadsTheFile` | `internal/plan/bodyfile_test.go` | `body=@src.txt` loads those lines; `body=@/etc/hosts` (IsRooted) errors naming the path; `body=@../out` errors as outside | — | S1, S2 |
| `TestWriteBodyAtPathCreatesFromFile` | `cmd/mrw/bodyfile_write_test.go` | CLI `create body=@src.txt` writes those bytes | — | S1, S3 |

Subtests of that one name so a mutant of any arm is inside the fence.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `parseHeader` `case "body"`; CLI/MCP `LoadBodyFiles`. Deleting either leaves create-from-file empty or the unit red |
| 3 — the caller can discover it | §103; T5 `write --help` / AGENTS |
| 4 — it is used | quality-blueprints 600-line creates via Edit; no telemetry (ADR-009) |

## Mutation Log
_(tool-written)_
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · LoadBodyFiles is a no-op so TestBodyAtPathLoadsTheFile must go red · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · covers:body=@ loads lines
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · rooted refusal drops the path so the rooted subtest must go red · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · covers:rooted path refused
- 2026-09-14 · 708adf2* · mutant survived · exit 0 · `internal/plan/plan.go` · outside-root Resolve error drops the path so the outside subtest must go red · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · covers:outside-root refused
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `internal/plan/plan.go` · outside-root Resolve error is swallowed so the outside subtest must go red · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · covers:outside-root refused
- 2026-09-14 · 708adf2* · mutant killed · exit 1 · `cmd/mrw/main.go` · CLI skips LoadBodyFiles so TestWriteBodyAtPathCreatesFromFile must go red · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · covers:CLI and MCP both call LoadBodyFiles

## Invariants

- Integer `body=N` unchanged.
- Empty body file ≡ `body=0` (ADR-027).
- ADR-002 does not require the body file to have been served.
- `rooted.IsRooted`, never `filepath.IsAbs`.

## Risks

- Loading in Parse would have no root. Load after parse with the checkout root.

## Stop Condition

Stop if `body=@` needs to be a second plan grammar rather than a `body=` value.

## Out of Scope

- Teach (T5).
- Reading the body file through the ledger.

## Verification Log
_(tool-written)_
- 2026-09-14 · 708adf2* · exit 1 · `set -o pipefail …` · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · ms:492 · test-lock-sha256:4825e48ac2d44a67d8f7fc0fd41967b38570ea1a3ae93312992fe7a2cd4e3eba · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvYm9keWZpbGVfd3JpdGVfdGVzdC5nbwlUZXN0V3JpdGVCb2R5QXRQYXRoQ3JlYXRlc0Zyb21GaWxlCTM1YWZmNWQ1YTY4NjQwZGVjOTI3ZjhmZjY5YmJiMmQxN2Y5YWQyOTRhMTYwZDEyNTA1N2YzMjZiMGVlZGQ1NzUKYm9keQlpbnRlcm5hbC9wbGFuL2JvZHlmaWxlX3Rlc3QuZ28JVGVzdEJvZHlBdFBhdGhMb2Fkc1RoZUZpbGUJYTIzYWI5ZWI4YjNiZmEzODRlYmEzZTJlYTYxMjdjMTA2MTUyNGZjNzZjNDg1NGFmZTI0N2E1NGYwZTE4MDMxOQpib2R5CWludGVybmFsL3BsYW4vYm9keWZpbGVfdGVzdC5nbwlsb2Fkcwk3MGM5NmViMzQxY2NjNDQyNjIzZjk3OTA5Yzc3MzhiNjQzOWQzYjIxM2MxMGM2YzhiZDBmOTI2ZjA4YTY0YjIyCmJvZHkJaW50ZXJuYWwvcGxhbi9ib2R5ZmlsZV90ZXN0LmdvCW91dHNpZGUJMDJlNmFiYWZjYjU1OTg2ODExOWZhN2YzZTk3YTA5YzNhOGI4NzBjODU3MGY4MTY3ZWRkMjc2ZDAwYWExOTNjNwpib2R5CWludGVybmFsL3BsYW4vYm9keWZpbGVfdGVzdC5nbwlyb290ZWQJMWQzZGMxYTE2YzM5OGZjY2FlOWIyMTY5NDM3MTE1MjkyZTUwYzJiZWY4OTI5ZTNmNzVmZGQ5NmJhMGVkNDA4Mw
  ```
  --- last 10 line(s) of stdout (of 22 after folding 22 raw)
      --- FAIL: TestBodyAtPathLoadsTheFile/rooted (0.00s)
      --- FAIL: TestBodyAtPathLoadsTheFile/outside (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	0.172s
  === RUN   TestWriteBodyAtPathCreatesFromFile
      bodyfile_write_test.go:16: create body=@src.txt exited 2:
  --- FAIL: TestWriteBodyAtPathCreatesFromFile (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.287s
  FAIL
  ```
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · ms:593
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · ms:612
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · ms:599
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · ms:912
- 2026-09-14 · 708adf2* · exit 0 · `set -o pipefail …` · acceptance-sha256:5a60703cee05a2a4411877ace1068727b2d062bc8448e1a3eba61970eca76b5d · ms:668
