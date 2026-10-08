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
