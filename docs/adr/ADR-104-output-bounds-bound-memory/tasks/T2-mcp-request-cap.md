# Task ADR-104-T2: an MCP request line is capped

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the bounded request read in `Serve`; contract §199
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an oversized request is answered, not read whole`, `the session continues`, `a contract row drives the binary`, `only the owned engine packages change`

## Goal

`Serve` reads a request line up to `maxRequestBytes` (64 MiB); a longer one is answered with -32600, id null,
naming the limit, the rest of the line is discarded, and the next line is served.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | `Serve` (`:203-222`) reads through a bounded line reader; `maxRequestBytes` |
| `internal/mcp/cap104_test.go` | add | the test below |
| `scripts/contract.sh` | edit | §199 |

## Ordered Steps

1. [S1] Write the failing test `TestAnOversizedRequestIsRefusedAndTheServerKeepsServing`; confirm RED. [proof: mutation]
2. [S2] Bound the read. [proof: mutation] Mutants: the cap check removed; the discarded tail is served as the next message.
3. [S3] Contract §199, driving `$MRW mcp`: a 65 MiB line then a `tools/list` gets a -32600 answer naming the limit and then the tool list. [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §199's rows printed]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAnOversizedRequestIsRefusedAndTheServerKeepsServing' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnOversizedRequestIsRefusedAndTheServerKeepsServing \(' "$out" \
  && grep -q '^# 199\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/state internal/lines internal/iter internal/rooted)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnOversizedRequestIsRefusedAndTheServerKeepsServing` | `internal/mcp/cap104_test.go` | with the limit set to 1 KiB, a 4 KiB line then a `tools/list` gets two answers: -32600 with a null id naming the limit, then the tool list; the oversized line's tail is not answered as a message | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every check, every MCP request, every read and ast-grep run goes through it |
| 3 — the caller can discover it | the refusal or the marker names the limit |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/mcp/mcp.go` · the cap check removed · acceptance-sha256:345b1e0f8d99c2263265f483ba7c72c8eb8dac812871c93ea02761ea64df1537 · covers:an oversized request is answered, not read whole
- 2026-09-30 · f911dd7* · mutant survived · exit 0 · `internal/mcp/mcp.go` · the discarded tail is kept as a message · acceptance-sha256:345b1e0f8d99c2263265f483ba7c72c8eb8dac812871c93ea02761ea64df1537 · covers:the session continues
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-30 · f911dd7* · mutant killed · exit 1 · `internal/mcp/mcp.go` · Serve ends the session after answering an oversized line · acceptance-sha256:345b1e0f8d99c2263265f483ba7c72c8eb8dac812871c93ea02761ea64df1537 · covers:the session continues

## Invariants

- Under the limit, every answer is byte-identical to before.
- Exit codes keep their meanings.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-104 tasks, each in its own file.

## Verification Log
- 2026-09-30 · f911dd7* · exit 1 · `set -o pipefail …` · acceptance-sha256:345b1e0f8d99c2263265f483ba7c72c8eb8dac812871c93ea02761ea64df1537 · ms:252 · test-lock-sha256:2068b9d453c2db7cdbee83b7292b5b7cf60087659f993896e606523d09646690 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9jYXAxMDRfdGVzdC5nbwlUZXN0QW5PdmVyc2l6ZWRSZXF1ZXN0SXNSZWZ1c2VkQW5kVGhlU2VydmVyS2VlcHNTZXJ2aW5nCWVjMTJiYTg1YzM4ZWNlMjgwNjlhYjFkZWYzYTJhMTU4N2JhN2VhMjkwNmEyYTE4ZDVhNWFjMmVlZDRkNDQzMTk
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/cap104_test.go:15:9: undefined: maxRequestBytes
  internal/mcp/cap104_test.go:16:2: undefined: maxRequestBytes
  internal/mcp/cap104_test.go:17:21: undefined: maxRequestBytes
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:345b1e0f8d99c2263265f483ba7c72c8eb8dac812871c93ea02761ea64df1537 · ms:316
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:345b1e0f8d99c2263265f483ba7c72c8eb8dac812871c93ea02761ea64df1537 · ms:308
- 2026-09-30 · human-observed · note 2026-09-30 on the survived mutant 'the discarded tail is kept as a message': it is equivalent — once over is set, Serve answers the refusal and never reads the bytes readLine kept, so keeping them changes nothing a caller sees. The session-continues clause is bound instead by the mutant that returns after the refusal; approved
- 2026-09-30 · f911dd7* · exit 0 · `set -o pipefail …` · acceptance-sha256:345b1e0f8d99c2263265f483ba7c72c8eb8dac812871c93ea02761ea64df1537 · ms:288
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped in the ADR-104 worktree, exit 0, with §199 printed: through the built binary's mcp a 65 MiB line then tools/list gets -32600 (id null) naming 67108864 and then the tool list; the server ends at EOF with exit 0
