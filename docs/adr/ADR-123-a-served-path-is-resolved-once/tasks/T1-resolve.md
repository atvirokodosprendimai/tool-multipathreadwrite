# Task ADR-123-T1: the state base cached, Resolve's real path reused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `rooted.resolvedBase`, `rooted.inState`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the state base cached, Resolve's real path reused`

## Goal

`rooted.Resolve` resolves the state base once while it exists and reuses its own real path for the state check, answering exactly as before.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | edit | `resolvedBase`, `inState`, Resolve's reuse |
| `internal/rooted/resolve123_test.go`, `internal/rooted/resolve123_bench_test.go` | add | the test and the benchmark |
| `docs/adr/BACKLOG.md` | edit | the Windows entry taken; the deferrals |

## Ordered Steps

1. [S1] Write `TestACaseSpellingOfARecreatedStateBaseIsRefused` and `BenchmarkResolveADeepFile`; record the benchmark before the change. [proof: mutation]
2. [S2] The cache and the reuse. Mutant: the cache used without its Stat and SameFile check (killed). An entry made while the base was absent needs no mutant of its own: `os.SameFile` against its nil FileInfo is false, so the same check rejects it; the two mutants that tried it are logged as equivalent. [proof: mutation]
3. [S3] The Windows timing from a peer, before and after, in the record. [proof: human: a Windows peer's timing of --grep over the 3,000-file tree with v1.46.0 and the branch build]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/rooted/ -count=1 -timeout 300s -run 'TestACaseSpellingOfARecreatedStateBaseIsRefused|TestAPathInsideMrwsStateIsRefused|TestACaseSpellingOfTheStateBaseIsRefused' -v 2>&1 | tee "$out" \
  && grep -qE '^--- (PASS|SKIP): TestACaseSpellingOfARecreatedStateBaseIsRefused \(' "$out" \
  && grep -qE '^--- PASS: TestAPathInsideMrwsStateIsRefused \(' "$out" \
  && go test ./internal/rooted/ ./internal/read/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 900s \
  && go test ./internal/rooted/ -run '^$' -bench BenchmarkResolveADeepFile -benchtime 100x \
  && grep -q '^func resolvedBase(base string) (string, os.FileInfo) {' internal/rooted/rooted.go \
  && GOOS=windows go vet ./internal/rooted/ \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestACaseSpellingOfARecreatedStateBaseIsRefused` | `internal/rooted/resolve123_test.go` | a base removed and made again is still recognised by identity | none | S1, S2 |
| `BenchmarkResolveADeepFile` | `internal/rooted/resolve123_bench_test.go` | the per-Resolve cost, before and after | none | S1 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `resolvedBase`, `inState` |
| 2 — something selects it | `Resolve` and `InState` call them |
| 3 — the caller can discover it | the timing in the record |
| 4 — it is used | every read, walk and write resolves through it |

## Mutation Log
- 2026-10-03 · 1c84cfd* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: the cached base is re-validated by identity on every use · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298
- 2026-10-03 · 1c84cfd* · mutant survived · exit 0 · `internal/rooted/rooted.go` · S2: an absent base is not cached · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-03 · 1c84cfd* · mutant survived · exit 0 · `internal/rooted/rooted.go` · S2: an entry made while the base was absent is never reused (corrects the equivalent mutant above: a nil FileInfo was never reused anyway) · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```

## Invariants

- Every path served or refused before is served or refused after.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the Windows timing shows no gain.

## Out of Scope

- Caching the identity walk (deferred: docs/adr/BACKLOG.md "From ADR-123")

## Verification Log
- 2026-10-03 · 1c84cfd* · exit 1 · `set -o pipefail …` · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298 · ms:39475 · test-lock-sha256:5d144f64375b901b23726cbacdfb71620d3b075e1c0f8eaa40f1e5761957dd25 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcm9vdGVkL3Jlc29sdmUxMjNfdGVzdC5nbwlUZXN0QUNhc2VTcGVsbGluZ09mQVJlY3JlYXRlZFN0YXRlQmFzZUlzUmVmdXNlZAk5YWY0YWZhZWQ0NzA0ZTEzMmI5MjY1ODc4MDYwNTEzMWJiMjU3MTJiNTZjNTdkNTA3ZDI0YjMxYjdiNWZhODkyCnVucHJvdmVuCWludGVybmFsL3Jvb3RlZC9yZXNvbHZlMTIzX2JlbmNoX3Rlc3QuZ28JQmVuY2htYXJrUmVzb2x2ZUFEZWVwRmlsZQ
  ```
  --- last 10 line(s) of stdout (of 19 after folding 19 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	14.426s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	27.668s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	37.868s
  goos: darwin
  goarch: arm64
  pkg: github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted
  cpu: Apple M5
  BenchmarkResolveADeepFile-10    	     100	     52304 ns/op
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.070s
  ```
- 2026-10-03 · 1c84cfd* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298 · ms:40227
- 2026-10-03 · 1c84cfd* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298 · ms:40489
- 2026-10-03 · 1c84cfd* · exit 0 · `set -o pipefail …` · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298 · ms:40454
- 2026-10-03 · 1c84cfd* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:f7032d25411dc31abd74d80bd93371c452d4f44ebbfebd705d3086bae8723298 · ms:0 · test-lock-sha256:67d56289b68ef18187218c29ca389a22b5625ea41b4efc13479a9620ed52a8f3 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcm9vdGVkL3Jlc29sdmUxMjNfdGVzdC5nbwlUZXN0QUNhc2VTcGVsbGluZ09mQVJlY3JlYXRlZFN0YXRlQmFzZUlzUmVmdXNlZAliMGYyOWNjYzAyMmFmOTY1YmU0MTRlMzg1NmVjMjk5YWIzMTAxOGY5MTQzOWMxMmQ5MTYyYjkyMTc1MzgwYjQ0CnVucHJvdmVuCWludGVybmFsL3Jvb3RlZC9yZXNvbHZlMTIzX2JlbmNoX3Rlc3QuZ28JQmVuY2htYXJrUmVzb2x2ZUFEZWVwRmlsZQ · test-lock-kind:replace
