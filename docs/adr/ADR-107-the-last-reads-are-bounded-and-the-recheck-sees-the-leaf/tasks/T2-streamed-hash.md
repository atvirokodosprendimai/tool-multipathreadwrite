# Task ADR-107-T2: acknowledgement hashes in bounded memory

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** acknowledgement hashes in bounded memory
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `acknowledgement holds no file in memory`

## Goal

`currentSHA` streams the file into the hash, so promotion holds no file in memory; the digest is unchanged.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/ack.go` | edit | `currentSHA` streams |
| `internal/mcp/hash107_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test `TestAcknowledgementHashesInBoundedMemory`; confirm RED. [proof: mutation]
2. [S2] Stream the hash. Mutant: `currentSHA` reads the file whole again. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAcknowledgementHashesInBoundedMemory' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAcknowledgementHashesInBoundedMemory \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted internal/check internal/links \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAcknowledgementHashesInBoundedMemory` | `internal/mcp/hash107_test.go` | `currentSHA` of a 50 MB file equals `seen.SHA` of its bytes and allocates under 1 MB | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every write, or every MCP acknowledgement |
| 3 — the caller can discover it | the refusal names the file and the reason |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 final design review |

## Mutation Log
- 2026-09-30 · 433ed20* · mutant killed · exit 1 · `internal/mcp/ack.go` · currentSHA reads the file whole again · acceptance-sha256:3d87b4a93297338da4371b6c7b202cfe0aece4d92e6401e45e73a0ca11b5c2e0 · covers:acknowledgement holds no file in memory

## Invariants

- Exit codes keep their meanings; plans under the limits are unaffected.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-107 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 433ed20* · exit 1 · `set -o pipefail …` · acceptance-sha256:3d87b4a93297338da4371b6c7b202cfe0aece4d92e6401e45e73a0ca11b5c2e0 · ms:841 · test-lock-sha256:e39b17b2c276e3e5a0fb77c5cf94ee77a70c19da0f1ba61b1f6b9a455bf41879 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9oYXNoMTA3X3Rlc3QuZ28JVGVzdEFja25vd2xlZGdlbWVudEhhc2hlc0luQm91bmRlZE1lbW9yeQk1NGM5ZTEzODVkZGRiMjJiMzkzNWYwYjQ1ZTVmZDhhYTMyYzAyODQ0ZjRjZDdkZTQ4MTIwOWUxMzcwYTQ2NDUz
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAcknowledgementHashesInBoundedMemory
      hash107_test.go:32: hashing a 50 MB file allocated 52437656 bytes, want under 1 MB
  --- FAIL: TestAcknowledgementHashesInBoundedMemory (0.05s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.307s
  FAIL
  ```
- 2026-09-30 · 433ed20* · exit 0 · `set -o pipefail …` · acceptance-sha256:3d87b4a93297338da4371b6c7b202cfe0aece4d92e6401e45e73a0ca11b5c2e0 · ms:344
- 2026-09-30 · 433ed20* · exit 0 · `set -o pipefail …` · acceptance-sha256:3d87b4a93297338da4371b6c7b202cfe0aece4d92e6401e45e73a0ca11b5c2e0 · ms:362
