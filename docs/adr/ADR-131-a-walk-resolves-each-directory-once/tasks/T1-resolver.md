# Task ADR-131-T1: a per-walk Resolver

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `rooted.NewResolver`, `rooted.Resolver`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the Resolver answers as Resolve`, `the Resolver dies with its walk`, `the walk and ast-grep select the Resolver`

## Goal

A walk and an ast-grep call resolve the files they discover through a `rooted.Resolver` that does each directory's work once and answers every path exactly as `rooted.Resolve` does.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | edit | `Resolver`, `NewResolver`, `resolveIn`, `inStateAt`, `aliasRefusal` |
| `internal/read/walk.go` | edit | the walker makes one Resolver and resolves discovered files through it — what selects it |
| `internal/read/astgrep.go` | edit | one Resolver per call for the hits — what selects it |
| `internal/rooted/resolver131_test.go`, `internal/rooted/resolver131_windows_test.go`, `internal/read/walk131_test.go`, `internal/read/walk131_bench_test.go` | add | the tests and the benchmark |
| `.github/workflows/timing-123.yml` | edit | the Windows timing of S4, before and after |
| `docs/adr/BACKLOG.md` | edit | ADR-123's deferral taken |

## Ordered Steps

1. [S1] Write the failing tests first — `TestAResolverAnswersAsResolveDoes` and `TestAResolverFollowsAJunctionAsResolveDoes` are red, since `NewResolver` does not exist; `TestAWalkCacheDoesNotOutliveItsWalk` passes before and is the guard S3's mutant breaks — and `BenchmarkWalkDeep131`; record the benchmark before the change. [proof: mutation]
2. [S2] The Resolver. Mutants: a link leaf judged from its directory; the directory's state verdict dropped; the containment check dropped. [proof: mutation]
3. [S3] The wiring: `read.Walk` and `read.AstGrep` each make their own Resolver. Mutant: one Resolver per root kept across walks. Deleting the wiring changes no answer — that is the design — so the fence pins the two lines by `grep` and the timing shows them used. [proof: mutation]
4. [S4] The Windows timing of `BenchmarkWalkDeep131` and of `--grep` over the 3,000-file tree, v1.51.0 against this branch, on one machine — a Windows peer, or a GitHub `windows-latest` runner through `.github/workflows/timing-123.yml` as ADR-123 did. Pre-registered from the profile: Resolve's share of a walk falls from about 85% to under a third, and a walk is at least 3× faster. [proof: human: a Windows timing of BenchmarkWalkDeep131 and --grep, v1.51.0 and this branch, on one machine]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/rooted/ ./internal/read/ -count=1 -timeout 300s -run 'TestAResolverAnswersAsResolveDoes|TestAWalkCacheDoesNotOutliveItsWalk|TestAResolverFollowsAJunctionAsResolveDoes' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAResolverAnswersAsResolveDoes \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestAWalkCacheDoesNotOutliveItsWalk \(' "$out" \
  && go test ./internal/rooted/ ./internal/read/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 900s \
  && go test ./internal/read/ -run '^$' -bench BenchmarkWalkDeep131 -benchtime 3x \
  && grep -q '^func NewResolver(root string) \*Resolver {' internal/rooted/rooted.go \
  && grep -q 'res:.*rooted\.NewResolver(root)}' internal/read/walk.go \
  && grep -q 'full, err := w\.res\.Resolve(rel)' internal/read/walk.go \
  && grep -q 'res := rooted\.NewResolver(absRoot)' internal/read/astgrep.go \
  && GOOS=windows go vet ./internal/rooted/ ./internal/read/ \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/lines \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAResolverAnswersAsResolveDoes` | `internal/rooted/resolver131_test.go` | every hazard is answered as Resolve answers it, twice | none | S1, S2 |
| `TestAResolverFollowsAJunctionAsResolveDoes` | `internal/rooted/resolver131_windows_test.go` | a junction out, a junction in, a junction root, on Windows | none | S1, S2 |
| `TestAWalkCacheDoesNotOutliveItsWalk` | `internal/read/walk131_test.go` | a state base made, or a directory swapped for a link, between two walks is seen by the second | none | S1, S3 |

`BenchmarkWalkDeep131` (`internal/read/walk131_bench_test.go`) is run by the fence and timed in the record, and it is not listed in the table: the test lock hashes `Test` functions only (ADR-123 T1).

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.Resolver` |
| 2 — something selects it | `read.Walk` and `read.AstGrep` make one; the fence greps both lines |
| 3 — the caller can discover it | the timing in the record |
| 4 — it is used | every `--grep` and `--ast-grep`, CLI and MCP |

## Mutation Log
- 2026-10-08 · f74dfd0* · mutant inconclusive · exit 1 · `internal/rooted/rooted.go` · S2: a link leaf judged from its directory · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the Resolver answers as Resolve
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-08 · f74dfd0* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: a link leaf judged from its directory (replaces the inconclusive row: that mutant left fi unused) · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the Resolver answers as Resolve
- 2026-10-08 · f74dfd0* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: the directory state verdict dropped · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the Resolver answers as Resolve
- 2026-10-08 · f74dfd0* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: the containment check dropped · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the Resolver answers as Resolve
- 2026-10-08 · f74dfd0* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S3: one Resolver per root kept across walks · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the Resolver dies with its walk
- 2026-10-08 · f74dfd0* · mutant killed · exit 1 · `internal/read/walk.go` · S3: the walk resolves discovered files without its Resolver (caught by the wiring grep only: every answer is the same, by design) · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the walk and ast-grep select the Resolver
- 2026-10-08 · f74dfd0* · mutant killed · exit 1 · `internal/read/astgrep.go` · S3: ast-grep hits resolved without the Resolver (caught by the wiring grep only) · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the walk and ast-grep select the Resolver
- 2026-10-08 · fd9f889* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: a state base that is a file judged by its directory alone (the in-process review of #353) · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · covers:the Resolver answers as Resolve

## Invariants

- Every path served or refused before is served or refused after.
- A Resolver outlives no walk.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the Windows timing shows no gain, or the differential test cannot be made to agree without resolving the file whole.

## Out of Scope

- The named-path call sites (permanent: boundary: the record's Out of Scope)

## Verification Log
- 2026-10-08 · f74dfd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:45056
- 2026-10-08 · f74dfd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:43584
- 2026-10-08 · f74dfd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:41601
- 2026-10-08 · f74dfd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:41307
- 2026-10-08 · f74dfd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:40856
- 2026-10-08 · f74dfd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:42477
- 2026-10-08 · f74dfd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:54806
- 2026-10-08 · d9ec641 · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:46063
- 2026-10-08 · fd9f889* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:40753
- 2026-10-08 · fd9f889* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:40478
- 2026-10-08 · human-observed · S4 Windows timing on GitHub windows-latest (Windows Server 2025, EPYC 9V74, Go 1.26.6), run 37760151095 at fd9f889: BenchmarkWalkDeep131 v1.51.0 5.63-6.00 s vs branch 0.22-0.24 s a walk; --grep over 3,000 files median 1269 ms vs 190 ms, junction root 1408 vs 186 ms; same 2 files served. Runner substitute as ADR-123 (no Windows peer online).
- 2026-10-08 · d174dc4* · exit 1 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:850 · test-lock-sha256:59f743f0c0cf3db256103b0dad923d80258978cb6bd98bf212f36c21f073ffa1 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvcmVhZC93YWxrMTMxX3Rlc3QuZ28JVGVzdEFXYWxrQ2FjaGVEb2VzTm90T3V0bGl2ZUl0c1dhbGsJNDdjM2RkODMxN2RhZGY2NjM3MzA5Njc5YzdlNWU4MDZkNWQ1OWQzMmIzNDMwNDAzN2MyOTM1ZjE4MDMyZmE3Nwpib2R5CWludGVybmFsL3Jvb3RlZC9yZXNvbHZlcjEzMV90ZXN0LmdvCVRlc3RBUmVzb2x2ZXJBbnN3ZXJzQXNSZXNvbHZlRG9lcwliMmU3MWQxZmYxYTAwNWIxYjVkNjQ1MTU5MzhlYWQyMzZiYTI3M2QzMzgyNWNiYjA2ZDVmYzk4MDcyNzBiNWUzCmJvZHkJaW50ZXJuYWwvcm9vdGVkL3Jlc29sdmVyMTMxX3dpbmRvd3NfdGVzdC5nbwlUZXN0QVJlc29sdmVyRm9sbG93c0FKdW5jdGlvbkFzUmVzb2x2ZURvZXMJYjQ0ODQyYzEzNWRiZmNhOTZkNTcyYzIxMWI2Y2E1MTlmNDIwYmFlY2E4YTFmNjZkNzcwYWFjMWNmMWQyY2Q3NQ
  ```
  --- last 10 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted.test]
  internal/rooted/resolver131_test.go:47:7: undefined: NewResolver
  internal/rooted/resolver131_test.go:65:15: undefined: NewResolver
  internal/rooted/resolver131_test.go:80:8: undefined: NewResolver
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted [build failed]
  === RUN   TestAWalkCacheDoesNotOutliveItsWalk
  --- PASS: TestAWalkCacheDoesNotOutliveItsWalk (0.00s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	0.306s
  FAIL
  ```
- 2026-10-08 · d174dc4* · exit 0 · `set -o pipefail …` · acceptance-sha256:8bf6271768a76c3f1e9c5688968896eb8f98f49a345bba1ab5201f9418cdeb9e · ms:39711
