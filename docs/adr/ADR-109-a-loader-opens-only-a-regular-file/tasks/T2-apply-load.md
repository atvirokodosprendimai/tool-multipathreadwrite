# Task ADR-109-T2: apply's load asks the descriptor

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** apply refuses a non-regular load on its hunk
**Consumes:** `regular.Open` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `apply's load asks the descriptor`

## Goal

`apply.readLines` opens through `regular.Open`, and a file the load finds is no longer regular is refused on its hunk, as ADR-107 refuses one that grew; `tree.readDir` opens its directory through the root with `O_NONBLOCK`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `readLines`, the `loadFn` caller |
| `internal/apply/tree.go` | edit | `readDir` |
| `internal/apply/fifo109_unix_test.go` | add | the test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestApplyLoadRefusesAFIFOAtOnce`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: `readLines` back to `os.Open`; the hunk refusal for `ErrNotRegular` removed. `tree.readDir` meets a FIFO only by a swap no test can arrange; it is declared uncovered. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestApplyLoadRefusesAFIFOAtOnce' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestApplyLoadRefusesAFIFOAtOnce \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/state internal/lines internal/rooted internal/seen \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestApplyLoadRefusesAFIFOAtOnce` | `internal/apply/fifo109_unix_test.go` | `readLines` of a FIFO returns `ErrNotRegular` within 5 s; a plan whose load meets one (through `loadFn`) is refused on its hunk with every verdict kept and no error; unix only, where FIFOs exist | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write, compile or check that opens the file |
| 3 — the caller can discover it | the refusal names the file and why |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex re-review of #304 and the 2026-10-01 measurement |

## Mutation Log
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/apply/apply.go` · readLines back to os.Open: a FIFO swapped in blocks the write · acceptance-sha256:7ebc08e13b70296d01ef0b43e719a81be7ef3b554a2683378764558541fa668b
- 2026-10-01 · 92450b2* · mutant killed · exit 1 · `internal/apply/apply.go` · the hunk refusal removed: the write returns an error and drops every verdict · acceptance-sha256:7ebc08e13b70296d01ef0b43e719a81be7ef3b554a2683378764558541fa668b

## Invariants

- A regular file opens and reads as before; refusal texts and exit codes are unchanged.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-109 tasks, each in its own file.

## Verification Log
- 2026-10-01 · 92450b2* · exit 1 · `set -o pipefail …` · acceptance-sha256:7ebc08e13b70296d01ef0b43e719a81be7ef3b554a2683378764558541fa668b · ms:163 · test-lock-sha256:722fef273b2450e19555f00938df527b6f0ad32820e9d436151ea11ae5fa1c60 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYXBwbHkvZmlmbzEwOV91bml4X3Rlc3QuZ28JVGVzdEFwcGx5TG9hZFJlZnVzZXNBRklGT0F0T25jZQkyMzJiNDk5MThhYzVhOGQ1NGZhNTk2OWJmYmQ2Mjg0NzY2ZDY4OTk3MzZiM2ZkMjExMTQ4ZDMzMzRiZTQ4OTE3
  ```
  --- last 3 line(s) of stdout
  github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular: no non-test Go files in ~/GolandProjects/wt-109/internal/regular
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply [build failed]
  FAIL
  ```
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:7ebc08e13b70296d01ef0b43e719a81be7ef3b554a2683378764558541fa668b · ms:727
- 2026-10-01 · 92450b2* · exit 0 · `set -o pipefail …` · acceptance-sha256:7ebc08e13b70296d01ef0b43e719a81be7ef3b554a2683378764558541fa668b · ms:766
