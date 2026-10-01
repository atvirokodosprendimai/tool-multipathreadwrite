# Task ADR-108-T8: a page counts only what read would serve

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** a page counts only what read would serve
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a page counts only what read would serve`

## Goal

`countFileLines`, which `firstPage` runs before it pages a read too large for the result ceiling, opens the file as read would serve it: resolved inside the root, without blocking, a regular file only, refused over 1 GiB and read through a bound. A read that read itself refused — a FIFO, an oversized file — can overflow a small ceiling with its refusal and reach the count, which used `os.ReadFile` on the joined path and so blocked the server on a FIFO and read any size (Codex review of #304).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `countFileLines`, and its call in `firstPage` |
| `internal/mcp/page108_test.go` | add | the bound and the boundary |
| `internal/mcp/page108_unix_test.go` | add | the FIFO, through the `mrw_read` handler |

## Ordered Steps

1. [S1] Write the failing tests `TestAPageCountIsBoundedAndConfined` and `TestPagingARefusedFIFOReturnsAtOnce`; confirm RED. [proof: mutation]
2. [S2] `countFileLines(root, path)`: `rooted.Resolve`, a non-blocking open, a regular-file check on the descriptor, a size refusal and a bounded read. Mutants: the non-blocking open removed (with no writer, a non-blocking FIFO reads as EOF, so the open is what keeps the server answering); the size refusal removed; the root resolution removed. The regular-file check and the bound guard a device or a file that grows while it is read, which no hermetic test reaches. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAPageCountIsBoundedAndConfined|TestPagingARefusedFIFOReturnsAtOnce' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPageCountIsBoundedAndConfined \(' "$out" \
  && grep -qE '^--- PASS: TestPagingARefusedFIFOReturnsAtOnce \(' "$out" \
  && grep -q '^# 204\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/state internal/lines internal/rooted internal/check \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPageCountIsBoundedAndConfined` | `internal/mcp/page108_test.go` | a file over the bound is refused naming its size; a link out of the root is refused; a regular file is counted as `lines.Split` counts | — | S1, S2 |
| `TestPagingARefusedFIFOReturnsAtOnce` | `internal/mcp/page108_unix_test.go` | an `mrw_read` of a FIFO under a ceiling its refusal overflows answers within 5 s; unix only, where FIFOs exist | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | `firstPage`, on every named read whose answer overflows the ceiling |
| 3 — the caller can discover it | the ordinary refusal it falls back to names the reason |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex review of #304 |

## Mutation Log
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/mcp/tools.go` · the non-blocking open removed: the page count waits on a FIFO and the server hangs · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/mcp/tools.go` · the size refusal removed: an oversized file is read before it is refused, and the refusal stops naming its size · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166
- 2026-10-01 · 918bc7d* · mutant killed · exit 1 · `internal/mcp/tools.go` · the root resolution removed: a link out of the root is counted · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166

## Invariants

- A page of a regular file within the bound is unchanged; `firstPage` still declines when its own read serves nothing (`tools.go`, ADR-025).

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-108 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:0395887d37f0cb3a214a3377aa6a14eaf2180545eb3aee1b919279eed5d85c1a · ms:472 · test-lock-sha256:2a779d4ccbd701cdc24e40f2dcb5e7b9fec478099d2f6d83ead4fbef3ee7a818 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9wYWdlMTA4X3Rlc3QuZ28JVGVzdEFQYWdlQ291bnRJc0JvdW5kZWRBbmRDb25maW5lZAk5Y2UzZDkxMjAyNDdiN2JmYmEzNDE1ZTI0NjBhN2FkNWViMTExYzdlNzJiMGU2NDE5NTNmMmRmZWUzNzAyYTgxCmJvZHkJaW50ZXJuYWwvbWNwL3BhZ2UxMDhfdW5peF90ZXN0LmdvCVRlc3RQYWdpbmdBUmVmdXNlZEZJRk9SZXR1cm5zQXRPbmNlCWVmMGQxMjdmYjI4YzUzMTU3MGQ1MDJiMzUwNjNhMTA5Y2E1MmE3YTA2NzFhMmM5N2YxNmRlMzhlMzczYzdmM2Q
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
  	want (string)
  internal/mcp/page108_test.go:22:2: undefined: maxCountBytes
  internal/mcp/page108_test.go:29:37: too many arguments in call to countFileLines
  	have (string, string)
  	want (string)
  internal/mcp/page108_test.go:37:36: too many arguments in call to countFileLines
  	have (string, string)
  	want (string)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166 · ms:395
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAPageCountIsBoundedAndConfined
  --- PASS: TestAPageCountIsBoundedAndConfined (0.00s)
  === RUN   TestPagingARefusedFIFOReturnsAtOnce
  --- PASS: TestPagingARefusedFIFOReturnsAtOnce (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.078s
  ```
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166 · ms:376
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAPageCountIsBoundedAndConfined
  --- PASS: TestAPageCountIsBoundedAndConfined (0.00s)
  === RUN   TestPagingARefusedFIFOReturnsAtOnce
  --- PASS: TestPagingARefusedFIFOReturnsAtOnce (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.089s
  ```
- 2026-10-01 · 918bc7d* · exit 1 · `set -o pipefail …` · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166 · ms:394
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAPageCountIsBoundedAndConfined
  --- PASS: TestAPageCountIsBoundedAndConfined (0.00s)
  === RUN   TestPagingARefusedFIFOReturnsAtOnce
  --- PASS: TestPagingARefusedFIFOReturnsAtOnce (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.080s
  ```
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166 · ms:397
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166 · ms:410
- 2026-10-01 · 918bc7d* · exit 0 · `set -o pipefail …` · acceptance-sha256:2cd1f19f06a11dd0bc32b5a6e6e4b218bd1a0c14e307b8e804aaeed64f4f8166 · ms:403
