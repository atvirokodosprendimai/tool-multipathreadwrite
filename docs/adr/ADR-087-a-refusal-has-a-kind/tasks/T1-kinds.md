# Task ADR-087-T1: kinds on the parser, the engine and the ack remedy

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `internal/refusal`; `plan.ParseError`; `HunkResult.Kind`; `nameTheAck` by kind
**Consumes:** nothing
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the parser names each mirrored kind`, `the engine names each mirrored kind`, `the engine names not-read`, `the ack remedy keys on the kind`, `a contract row drives the binary`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

The parser and the engine were paired by message text, and the MCP acknowledgement remedy found its refusal by matching "has not been read" in the reason.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/refusal/refusal.go`, `internal/refusal/refusal_test.go` | new | the kinds, `Error`, `KindOf` |
| `internal/adversarial/kinds087_test.go` | new | each mirrored rule: one kind on both sides; not-read on the engine |
| `internal/plan/plan.go` | edit | `validate` returns kinded refusals; `ParseError` |
| `internal/apply/apply.go` | edit | `HunkResult.Kind`; kinded `fail` at the mirrored and not-read sites |
| `internal/mcp/tools.go` | edit | `nameTheAck` keys on `refusal.NotRead` |
| `scripts/contract.sh` | edit | §170 |
| `docs/adr/BACKLOG.md` | edit | the entry closed |

## Ordered Steps

1. [S1] Write `TestTheEngineAndTheParserRefuseWithOneKind` and `TestKindOfFindsAWrappedRefusal`; the first fails on `81b8012`, where neither side names a kind. [proof: mutation]
2. [S2] Kinds in `validate`, `Parse`, the engine and `nameTheAck`. [proof: mutation]
3. [S3] Contract §170: over `mrw mcp`, a write to a served, unacknowledged line names the acknowledgement remedy; a write to a file never read does not. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/adversarial/ ./internal/refusal/ -count=1 -timeout 180s -run 'TestTheEngineAndTheParserRefuseWithOneKind|TestKindOfFindsAWrappedRefusal|TestTheEngineAndTheParserRefuseInTheSameWords' -v 2>&1 | tee /tmp/adr087-T1.out \
  && missing=$(for t in TestTheEngineAndTheParserRefuseWithOneKind TestKindOfFindsAWrappedRefusal TestTheEngineAndTheParserRefuseInTheSameWords; do grep -qE "^--- PASS: $t \(" /tmp/adr087-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 170\. ' scripts/contract.sh \
  && ! grep -q 'strings.Contains(h.Reason, "has not been read")' internal/mcp/tools.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheEngineAndTheParserRefuseWithOneKind` | `internal/adversarial/kinds087_test.go` | each mirrored rule has one kind on both sides; not-read on the engine | — | S1, S2 |
| `TestKindOfFindsAWrappedRefusal` | `internal/refusal/refusal_test.go` | a kind survives wrapping; text unchanged | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `internal/refusal` |
| 2 — something selects it | every mirrored parser rule, its engine copy, and every MCP write refused as not read |
| 3 — the caller can discover it | the acknowledgement remedy still appears on the refusal it belongs to |
| 4 — it is used | `nameTheAck` in production; ADR-009 refuses telemetry |

## Verification Log
(empty until execute)
- 2026-09-27 · 81b8012* · exit 1 · `set -o pipefail …` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:509 · test-lock-sha256:3bf0fc73448aa7c00c6f29d2c54b35db733eaad69249fbe60f1513bad82892de · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FkdmVyc2FyaWFsL2tpbmRzMDg3X3Rlc3QuZ28JVGVzdFRoZUVuZ2luZUFuZFRoZVBhcnNlclJlZnVzZVdpdGhPbmVLaW5kCTZjNjFiNGE3NGU5MGVlMThhYzJjOWNlOGQzZTRjMjg3MzAxYzhjMmNlMWYzMWIzNmQzMjU4ODBlOTI2OTc0MTcKYm9keQlpbnRlcm5hbC9yZWZ1c2FsL3JlZnVzYWxfdGVzdC5nbwlUZXN0S2luZE9mRmluZHNBV3JhcHBlZFJlZnVzYWwJNTJmZTNhMmRlOTdjYjIxNzBjYzQwYmU4OTk2ZWRhZTFjMGE5NDMwNWIwNDllMjkwMmU4YzQ0NjVjNTZkM2JlNQ
  ```
  --- last 10 line(s) of stdout (of 12 after folding 12 raw)
  internal/adversarial/kinds087_test.go:46:39: res.Hunks[0].Kind undefined (type apply.HunkResult has no field or method Kind)
  internal/adversarial/kinds087_test.go:47:87: res.Hunks[0].Kind undefined (type apply.HunkResult has no field or method Kind)
  internal/adversarial/kinds087_test.go:57:38: res.Hunks[0].Kind undefined (type apply.HunkResult has no field or method Kind)
  internal/adversarial/kinds087_test.go:58:84: res.Hunks[0].Kind undefined (type apply.HunkResult has no field or method Kind)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/adversarial [build failed]
  === RUN   TestKindOfFindsAWrappedRefusal
  --- PASS: TestKindOfFindsAWrappedRefusal (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal	0.103s
  FAIL
  ```
- 2026-09-27 · 81b8012* · exit 0 · `set -o pipefail …` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:794
- 2026-09-27 · 81b8012* · exit 0 · `set -o pipefail …` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:646
- 2026-09-27 · 81b8012* · exit 0 · `set -o pipefail …` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:404
- 2026-09-27 · 81b8012* · exit 0 · `set -o pipefail …` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:1004
- 2026-09-27 · 81b8012* · exit 0 · `set -o pipefail …` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:812
- 2026-09-27 · 2c0f396* · exit 0 · `set -o pipefail …` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:682
- 2026-10-02 · 3ede136* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · ms:0 · test-lock-sha256:0802da81eba0a390deff0bf1c1262ae1c0deb9ac5f339fe2c0dc8fc892ae4c14 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYWR2ZXJzYXJpYWwva2luZHMwODdfdGVzdC5nbwlUZXN0UGFyc2VFcnJvcktpbmRzTGluZVVwV2l0aFRoZUVycm9ycwllOWVmOTM5ZWQ0NjJjNWY0MmRiN2JjZDNlZGM2NWRiM2EwZGU5MGYyMGU3OWJhMDk2Nzg3NDhjOGYwNmUzOTI3CmJvZHkJaW50ZXJuYWwvYWR2ZXJzYXJpYWwva2luZHMwODdfdGVzdC5nbwlUZXN0VGhlRW5naW5lQW5kVGhlUGFyc2VyUmVmdXNlV2l0aE9uZUtpbmQJNDA3MGIwN2MzYmY5YTVkODUwOGU5MzNlMDY2OTA2NjRiNjYwMjViMzRkMDlhOTcyNzZlZTVhOGNlMGM5NmRiZQpib2R5CWludGVybmFsL3JlZnVzYWwvcmVmdXNhbF90ZXN0LmdvCVRlc3RLaW5kT2ZGaW5kc0FXcmFwcGVkUmVmdXNhbAk1MmZlM2EyZGU5N2NiMjE3MGNjNDBiZTg5OTZlZGFlMWMwYTk0MzA1YjA0OWUyOTAyZThjNDQ2NWM1NmQzYmU1 · test-lock-kind:replace

## Mutation Log
(empty until execute)
- 2026-09-27 · 81b8012* · mutant killed · exit 1 · `internal/apply/apply.go` · the engine drops the insert-range kind on a numeric range · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · covers:the engine names each mirrored kind
- 2026-09-27 · 81b8012* · mutant killed · exit 1 · `internal/apply/apply.go` · the engine drops the not-read kind on an unread file · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · covers:the engine names not-read
- 2026-09-27 · 81b8012* · mutant killed · exit 1 · `internal/plan/plan.go` · the parser stops carrying kinds on ParseError · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · covers:the parser names each mirrored kind
- 2026-09-27 · 81b8012* · mutant killed · exit 1 · `internal/mcp/tools.go` · the ack remedy goes back to matching the refusal words · acceptance-sha256:263a4b3f241285ab54152c004eada0d0d60114dfa98517f99741f0331d9ff63a · covers:the ack remedy keys on the kind

## Invariants

- Every refusal's text is byte-identical; `TestTheEngineAndTheParserRefuseInTheSameWords` still passes.
- No receipt field changes.

## Risks

- None beyond the record's.

## Out of Scope

- Path-op and other refusal kinds (permanent: boundary: nothing classifies them)

## Stop Condition

The fence exits 0 and contract §170 passes.
