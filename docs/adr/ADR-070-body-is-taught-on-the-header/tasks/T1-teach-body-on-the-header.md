# Task ADR-070-T1: `mrw instructions` and the handshake show `body=` on a header; contract §132

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** a worked `@@ … body=N` line on both served instructions
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `instructions show body on a header`, `the handshake shows body on a header`, `the binary prints it`, `no engine file changes`

## Goal

Reading 04's agents saw `body=` described but never on a header. Add one worked header line to `internal/guide/guide.go` (the `mrw instructions` text) beside the line-count sentence, give the MCP handshake's `examplePlan` (`internal/mcp/instructions.go:39`) `body=4` on its first hunk, and give AGENTS.md §2's example `body=`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | worked header line |
| `internal/mcp/instructions.go` | edit | `examplePlan` first hunk counts its body |
| `internal/guide/bodyheader_test.go` | new | `TestInstructionsShowBodyOnAHeader` |
| `internal/mcp/bodyheader_test.go` | new | `TestTheHandshakeShowsBodyOnAHeader` |
| `AGENTS.md` | edit | §2: where `body=` goes |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerate | the pinned `initialize` answer carries the worked plan; the diff is ` body=4` only |
| `internal/mcp/era_test.go` | edit | the golden's comment records this regeneration |
| `scripts/contract.sh` | edit | §132 |

## Ordered Steps

1. [S1] Write both tests; confirm RED. Each requires a line matching `^@@ \S+ \S+ \S+ .*\bbody=\d+` in its text. [proof: mutation]
2. [S2] Add the lines; GREEN. The existing `examplePlan` tests (`mcp_test.go:300`, `conformance_test.go:306`) stay green, and the example still parses and applies. [proof: mutation]
   Mutant: the worked line removed from `guide.go`.
3. [S3] §132: `mrw instructions` output carries such a header. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 132\. ' scripts/contract.sh \
  && go test ./internal/guide/ ./internal/mcp/ -count=1 -v -run 'TestInstructionsShowBodyOnAHeader|TestTheHandshakeShowsBodyOnAHeader' 2>&1 | tee /tmp/adr070-T1.out \
  && grep -q '^--- PASS: TestInstructionsShowBodyOnAHeader ' /tmp/adr070-T1.out \
  && grep -q '^--- PASS: TestTheHandshakeShowsBodyOnAHeader ' /tmp/adr070-T1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr070-T1.out \
  && ./scripts/contract.sh > /tmp/adr070-T1-contract.out 2>&1 \
  && grep -q '^  PASS  mrw instructions shows body= on a header' /tmp/adr070-T1-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/seen internal/check internal/state internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l internal/guide internal/mcp)" ] \
  && go vet ./internal/guide/ ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestInstructionsShowBodyOnAHeader` | `internal/guide/bodyheader_test.go` | `mrw instructions` carries a worked header with `body=` | — | S1, S2 |
| `TestTheHandshakeShowsBodyOnAHeader` | `internal/mcp/bodyheader_test.go` | the MCP handshake's worked plan counts its body | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests and §132 |
| 2 — something selects it | `mrw instructions` prints `guide.CLI()`; `initialize` serves the handshake |
| 3 — the caller can discover it | it is in the text the caller is told to read first |
| 4 — it is used | blind reading 05 measures it |

## Verification Log
(empty until execute)
- 2026-09-25 · c43095d* · exit 1 · `set -o pipefail …` · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · ms:739 · test-lock-sha256:9c1794d3b7842259065e591d974b0f48985a5c95eca055782de7c4f44fb8fcf1 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2d1aWRlL2JvZHloZWFkZXJfdGVzdC5nbwlUZXN0SW5zdHJ1Y3Rpb25zU2hvd0JvZHlPbkFIZWFkZXIJNmZkNDE1NDk2ZTQyODYxNzEyYzk2ZWZiZDdlOTg0YzE5OGMxMjI4YTZlNTY1ZTI4ZDJmYjM2ZTBiN2M5N2JlNwpib2R5CWludGVybmFsL21jcC9ib2R5aGVhZGVyX3Rlc3QuZ28JVGVzdFRoZUhhbmRzaGFrZVNob3dzQm9keU9uQUhlYWRlcgk4NGUyNDgxMGFjZWJhOTYwZDA0Yjg3MGIzYWM0ZjUzMWUyNDVmYTZhMzZjMmEzMDdiODMwOTVkYzVmNDI3MmEz
  ```
  --- last 10 line(s) of stdout (of 45 after folding 45 raw)
          func (s *Store) Get(id string) (Row, bool) {
          	r, ok := s.rows[id]
          	return r, ok
          }
          @@ cmd/app/main.go 12 insert-after
          	"example.com/app/internal/store"
  --- FAIL: TestTheHandshakeShowsBodyOnAHeader (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.008s
  FAIL
  ```
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · ms:29880
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · ms:29345
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · ms:29227
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · ms:29172

## Mutation Log
(empty until execute)
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/guide/guide.go` · the worked header is removed: the guide test and §132 go red · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · covers:instructions show body on a header
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/guide/guide.go` · the worked header is removed: the guide test and §132 go red · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · covers:the binary prints it
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/mcp/instructions.go` · the handshake plan stops counting its body: its test goes red · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · covers:the handshake shows body on a header
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-070 does not own changes: the go/no-go guard must go red · acceptance-sha256:0c0d4f9bab62b6e4fce8abb06367d46431ac5498aec046dd52702399b1f66c39 · covers:no engine file changes

## Invariants

- The worked `examplePlan` still parses and applies.
- Every other line of both instructions is unchanged.

## Risks

- The worked line is copied with its count unchanged; ADR-015 and ADR-060 refuse the resulting over- or undercount.

## Out of Scope

- `mrw write --help` already shows `body=` inline (permanent: boundary: `main.go:825`; the blind prompt bans it, so it is not the gap)

## Stop Condition

Stop if the change needs a plan-grammar change or an exit-code change.
