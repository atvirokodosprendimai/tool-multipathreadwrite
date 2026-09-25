# Task ADR-069-T3: A rename destination keeps its trailing space; contract §130

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** an untrimmed rename destination
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a rename lands at d-space`, `the binary renames to d-space`, `no engine file changes`

## Goal

A `rename` hunk's one-line body is its destination. `internal/apply/apply.go:387`, `:453` and
`pathop.go:76` pass it through `strings.TrimSpace`, so `@@ a.txt - rename` with body `d ` renames
to `d`. The plan scanner has already removed the line terminator. Use `Body[0]` as written. The
emptiness guards (`pathop.go:48`, `plan.go:749`) stay.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `:387`, `:453` use the body line as written |
| `internal/apply/pathop.go` | edit | `:76` likewise |
| `internal/apply/rename_space_test.go` | new | `TestARenameDestinationKeepsItsTrailingSpace` |
| `scripts/contract.sh` | edit | §130 |

## Ordered Steps

1. [S1] Write the test; confirm RED on an assertion (today the file lands at `d`). [proof: mutation]
   - A plan `@@ a.txt - rename` / `d ` renames `a.txt` to `d `; no `d` exists afterwards; the
     receipt's `RenamedTo` is `d `.
2. [S2] Drop the three trims; GREEN; every `internal/apply` test stays green. [proof: mutation]
   Mutant: `:453` trims again.
3. [S3] §130 through the binary: the same plan leaves `d ` and no `d`. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 130\. ' scripts/contract.sh \
  && go test ./internal/apply/ -count=1 -v -run 'TestARenameDestinationKeepsItsTrailingSpace' 2>&1 | tee /tmp/adr069-T3.out \
  && grep -q '^--- PASS: TestARenameDestinationKeepsItsTrailingSpace ' /tmp/adr069-T3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T3.out \
  && ./scripts/contract.sh > /tmp/adr069-T3-contract.out 2>&1 \
  && grep -q '^  PASS  a rename destination keeps its trailing space' /tmp/adr069-T3-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw internal/iter internal/apply internal/ingest)" ] \
  && go vet ./cmd/mrw/ ./internal/iter/ ./internal/apply/ ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARenameDestinationKeepsItsTrailingSpace` | `internal/apply/rename_space_test.go` | a rename lands at `d `, not `d` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §130 |
| 2 — something selects it | the CLI / compiler path every caller of that surface takes |
| 3 — the caller can discover it | the header, receipt or refusal names the exact path |
| 4 — it is used | §130 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · ms:29933
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · ms:29977
- 2026-09-25 · c43095d* · exit 1 · `set -o pipefail …` · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · ms:220 · test-lock-sha256:81023a12fdef99fed54a442d7e57e1e08d0ebf5108fb51a590b714024b76099a · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL2FwcGx5L3JlbmFtZV9zcGFjZV90ZXN0LmdvCVRlc3RBUmVuYW1lRGVzdGluYXRpb25LZWVwc0l0c1RyYWlsaW5nU3BhY2UJNWUwZWY1NWZjOTdiM2Q2YTYwNjMzMTk2NGVkZmE4ODNlN2I0MjRmYzUxMDVmZjJiMGQxYTY0MDNjM2ZhYTA5ZQ
  ```
  --- last 7 line(s) of stdout
  === RUN   TestARenameDestinationKeepsItsTrailingSpace
      rename_space_test.go:31: the rename landed at d, not at the destination the plan named
      rename_space_test.go:34: the rename did not land at "d "
  --- FAIL: TestARenameDestinationKeepsItsTrailingSpace (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply	0.005s
  FAIL
  ```
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · ms:30572
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · ms:30296

## Mutation Log
(empty until execute)
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/apply/apply.go` · the rename destination is trimmed again: the test and §130 must go red · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · covers:a rename lands at d-space
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/apply/apply.go` · the rename destination is trimmed again: §130 goes red · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · covers:the binary renames to d-space
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:9e2d4fa38c3f0aa7942eb1b89a020a90b1e38f67218f7721d216262f308ae830 · covers:no engine file changes

## Invariants

- A destination with no edge whitespace renames exactly as before.
- An empty destination is still refused.

## Risks

- A hand-written plan with a stray trailing space after a destination now renames to the padded name; the receipt names it.
- Windows cannot hold `x` and `x ` apart; the space fixtures skip there, visibly.

## Out of Scope

- A leading space in the destination is kept too; nothing strips it (permanent: boundary: the body line is the destination, exactly)

## Stop Condition

Stop if keeping the path exactly needs a format change or an exit-code change.
