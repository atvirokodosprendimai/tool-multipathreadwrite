# Task ADR-109-T1: the helper, and read opens through it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `regular.Open`, `regular.ErrNotRegular`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the helper, and read opens through it`

## Goal

A leaf package `internal/regular` holds the one non-blocking, descriptor-checked open; `read.readCapped` and the ast-grep path probe open through it, so a file swapped for a FIFO after the path check is refused rather than blocking a read or a `--grep` walk.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/regular/regular.go` | add | `Open`, `ErrNotRegular` |
| `internal/read/read.go` | edit | `readCapped` |
| `internal/read/astgrep.go` | edit | the named-path probe |
| `internal/regular/regular_unix_test.go` | add | the helper's test |
| `internal/read/fifo109_unix_test.go` | add | the loader's test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestOpenRefusesANonRegularFileAtOnce`, `TestReadCappedRefusesAFIFOAtOnce`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: `Open` without `O_NONBLOCK`; `Open` without the descriptor check; `readCapped` back to `os.Open`. The ast-grep probe opens and closes without reading, and a swap between `judgeNamed` and it cannot be arranged without a seam, so it rests on `Open`'s own test. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/regular/ ./internal/read/ -count=1 -timeout 300s -run 'TestOpenRefusesANonRegularFileAtOnce|TestReadCappedRefusesAFIFOAtOnce' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestOpenRefusesANonRegularFileAtOnce \(' "$out" \
  && grep -qE '^--- PASS: TestReadCappedRefusesAFIFOAtOnce \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestOpenRefusesANonRegularFileAtOnce` | `internal/regular/regular_unix_test.go` | `Open` of a FIFO returns `ErrNotRegular` within 5 s with no writer; a regular file and a directory open; unix only, where FIFOs exist | — | S1, S2 |
| `TestReadCappedRefusesAFIFOAtOnce` | `internal/read/fifo109_unix_test.go` | `readCapped` of a FIFO returns the not-a-regular-file refusal within 5 s; unix only, where FIFOs exist | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write, compile or check that opens the file |
| 3 — the caller can discover it | the refusal names the file and why |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex re-review of #304 and the 2026-10-01 measurement |

## Mutation Log
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/regular/regular.go` · Open without O_NONBLOCK: the open of a FIFO waits for a writer · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/regular/regular.go` · Open without the descriptor check: a FIFO is handed back as a file · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6
- 2026-10-01 · 92450b2* · mutant inconclusive · exit 1 · `internal/read/read.go` · readCapped back to os.Open: a FIFO blocks the read · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/read/read.go` · readCapped back to os.Open: a FIFO blocks the read · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6
- 2026-10-01 · 1fb4932* · mutant killed · exit 1 · `internal/regular/regular.go` · Open without O_NONBLOCK: the open of a FIFO waits for a writer · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9
- 2026-10-01 · 1fb4932* · mutant killed · exit 1 · `internal/regular/regular.go` · Open without the descriptor check: a FIFO is handed back as a file · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9
- 2026-10-01 · 1fb4932* · mutant killed · exit 1 · `internal/read/read.go` · readCapped back to os.Open: a FIFO blocks the read · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9
- 2026-10-01 · 1fb4932* · mutant killed · exit 1 · `internal/regular/regular.go` · a failed open is not classified: a socket comes back as the open error, not ErrNotRegular · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9

## Invariants

- A regular file opens and reads as before; refusal texts and exit codes are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-109 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 92450b2* · exit 1 · `set -o pipefail …` · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6 · ms:5693 · test-lock-sha256:e50701b6baf6283a56ab1fc0506bbc6bcb9b83c17cebc4eb12a9743a8eae51de · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9maWZvMTA5X3VuaXhfdGVzdC5nbwlUZXN0UmVhZENhcHBlZFJlZnVzZXNBRklGT0F0T25jZQk0MTNkYTU2MWRjMWJkZTk5NzMzZGVmM2MzNWZlODc2NTc4NGRjNjNmMDc5NzNiM2M1NjQ5NDE3Nzc2OTU0YzVjCmJvZHkJaW50ZXJuYWwvcmVndWxhci9yZWd1bGFyX3VuaXhfdGVzdC5nbwlUZXN0T3BlblJlZnVzZXNBTm9uUmVndWxhckZpbGVBdE9uY2UJY2IwZjhjMGEzN2Y4ZDI3YzhhNjUzYTVkOTc2NzYyOTZiNzA4OTdlNTZhYWQxMDllNjQ5NjI1YjNmYmY2ODU3ZA
  ```
  --- last 10 line(s) of stdout (of 15 after folding 15 raw)
  internal/regular/regular_unix_test.go:51:19: undefined: Open
  internal/regular/regular_unix_test.go:56:19: undefined: Open
  internal/regular/regular_unix_test.go:61:18: undefined: Open
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular [build failed]
  === RUN   TestReadCappedRefusesAFIFOAtOnce
      fifo109_unix_test.go:38: readCapped blocked on a FIFO
  --- FAIL: TestReadCappedRefusesAFIFOAtOnce (5.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	5.235s
  FAIL
  ```
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6 · ms:831
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6 · ms:640
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6 · ms:727
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:da72124b15ee453a7652c621960d03579087dd2c0330fb17255624738348b1a6 · ms:653
- 2026-10-01 · 1fb4932* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9 · ms:0 · test-lock-sha256:82f81ac74b070d5bfaaab9324a0235902063aadca812a477fe2b2b0703f56d1b · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC9maWZvMTA5X3VuaXhfdGVzdC5nbwlUZXN0UmVhZENhcHBlZFJlZnVzZXNBRklGT0F0T25jZQk0MTNkYTU2MWRjMWJkZTk5NzMzZGVmM2MzNWZlODc2NTc4NGRjNjNmMDc5NzNiM2M1NjQ5NDE3Nzc2OTU0YzVjCmJvZHkJaW50ZXJuYWwvcmVndWxhci9yZWd1bGFyX3VuaXhfdGVzdC5nbwlUZXN0T3BlblJlZnVzZXNBTm9uUmVndWxhckZpbGVBdE9uY2UJOTE3MmRmNTFkNDhiODk3MjQ0MjVlYmEwYmM0NWM2NzdiZmIyMDk3NTgyZDllMmY3YjU2ZjFjMDc5ZTY3NTA2OQ · test-lock-kind:replace
- 2026-10-01 · human-observed · Claude's session observed the relock: after red the test gained a socket case from the Codex review of #306 (a socket cannot be opened, so its open fails before the descriptor is asked, and it must still be refused as not regular); the FIFO assertions are unchanged
- 2026-10-01 · 1fb4932* · exit 0 · `set -o pipefail …` · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9 · ms:827
- 2026-10-01 · 1fb4932* · exit 0 · `set -o pipefail …` · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9 · ms:819
- 2026-10-01 · 1fb4932* · exit 0 · `set -o pipefail …` · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9 · ms:854
- 2026-10-01 · 1fb4932* · exit 0 · `set -o pipefail …` · acceptance-sha256:5d13b9126bf8ff20d31d62b4171801b2d974be71c691c708a66ea24c887c2fc9 · ms:855
