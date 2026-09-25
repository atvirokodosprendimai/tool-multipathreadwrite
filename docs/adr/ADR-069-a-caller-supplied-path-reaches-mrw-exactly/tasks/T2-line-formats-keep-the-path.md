# Task ADR-069-T2: `--files-from` and the working set keep a line's trailing space; contract §129

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** untrimmed `--files-from` specs and working-set entries
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `files-from keeps the path`, `the working set keeps the path`, `the binary serves x-space from a list`, `no engine file changes`

## Goal

`--files-from` (`cmd/mrw/main.go:1706`) and the working set (`internal/iter/iter.go:60` Load, `:109`
Add, `:125` Remove) trim each line. The trim is needed only to recognise a blank line or a `#`
comment. Test with a trimmed copy, keep the line as written. Load and Add change together: Add
storing `x ` is useless if Load trims it back.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `--files-from` keeps the line |
| `internal/iter/iter.go` | edit | Load, Add, Remove keep the entry |
| `cmd/mrw/filesfrom_space_test.go` | new | `TestFilesFromKeepsAPathsTrailingSpace` |
| `internal/iter/space_test.go` | new | `TestTheWorkingSetKeepsAPathsTrailingSpace` |
| `scripts/contract.sh` | edit | §129 |

## Ordered Steps

1. [S1] Write both tests; confirm each RED on an assertion. [proof: mutation]
   - `TestFilesFromKeepsAPathsTrailingSpace`: files `x`, `x ` with different bytes; `--files-from`
     fed `x ` serves `x `'s bytes, not `x`'s.
   - `TestTheWorkingSetKeepsAPathsTrailingSpace`: `Add("x ")`, `Save`, `Load` gives `x `; `Remove("x ")`
     removes it and leaves `x`; a blank line and a `# note` are still skipped/kept as today.
2. [S2] Change the two sites; GREEN; `cmd/mrw` and `internal/iter` stay green. [proof: mutation]
   Mutants: `--files-from` re-trims; iter Load re-trims.
3. [S3] §129: `printf 'x \n' | mrw read --files-from -` serves `x `. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 129\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ ./internal/iter/ -count=1 -v -run 'TestFilesFromKeepsAPathsTrailingSpace|TestTheWorkingSetKeepsAPathsTrailingSpace' 2>&1 | tee /tmp/adr069-T2.out \
  && grep -q '^--- PASS: TestFilesFromKeepsAPathsTrailingSpace ' /tmp/adr069-T2.out \
  && grep -q '^--- PASS: TestTheWorkingSetKeepsAPathsTrailingSpace ' /tmp/adr069-T2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T2.out \
  && ./scripts/contract.sh > /tmp/adr069-T2-contract.out 2>&1 \
  && grep -q '^  PASS  a --files-from line keeps its trailing space' /tmp/adr069-T2-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw internal/iter internal/apply internal/ingest)" ] \
  && go vet ./cmd/mrw/ ./internal/iter/ ./internal/apply/ ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestFilesFromKeepsAPathsTrailingSpace` | `cmd/mrw/filesfrom_space_test.go` | a `--files-from` line keeps its trailing space | — | S1, S2 |
| `TestTheWorkingSetKeepsAPathsTrailingSpace` | `internal/iter/space_test.go` | Load/Add/Remove keep an entry's trailing space | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §129 |
| 2 — something selects it | the CLI / compiler path every caller of that surface takes |
| 3 — the caller can discover it | the header, receipt or refusal names the exact path |
| 4 — it is used | §129 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · ms:29636
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · ms:29629
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · ms:30184
- 2026-09-25 · c43095d* · exit 1 · `set -o pipefail …` · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · ms:246 · test-lock-sha256:3e2a8f7bbba90e36295ab6ce6db2865cffb463e5f8a99cb5ccf483345fa9bf72 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvZmlsZXNmcm9tX3NwYWNlX3Rlc3QuZ28JVGVzdEZpbGVzRnJvbUtlZXBzQVBhdGhzVHJhaWxpbmdTcGFjZQkzM2NmZjZjMTg3OWIxZDc4NTQ0OGJlNTA3OWU3Y2ZiMjQ3YzM3MTQwMDc2ZWYyZjEwMmYyOTgwZTM5YTkzMzY0CmJvZHkJaW50ZXJuYWwvaXRlci9zcGFjZV90ZXN0LmdvCVRlc3RUaGVXb3JraW5nU2V0S2VlcHNBUGF0aHNUcmFpbGluZ1NwYWNlCWNjYWJiZThlZDAzMjA0YWQ3ZWM4NTM4YzllMzNhODJhOTlmYmE4M2JhZThhNGZjODhmNThmZTdjZjc2MDEzY2I
  ```
  --- last 10 line(s) of stdout (of 15 after folding 15 raw)
  --- FAIL: TestFilesFromKeepsAPathsTrailingSpace (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.008s
  === RUN   TestTheWorkingSetKeepsAPathsTrailingSpace
      space_test.go:15: Add("x ", "x", "  ") added 1, want 2 (a blank argument is skipped)
      space_test.go:25: Load gave ["x"], want ["x " "x"]
  --- FAIL: TestTheWorkingSetKeepsAPathsTrailingSpace (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	0.006s
  FAIL
  ```
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · ms:32771
- 2026-09-25 · c43095d* · exit 0 · `set -o pipefail …` · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · ms:30632

## Mutation Log
(empty until execute)
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/iter/iter.go` · iter Load re-trims: the working-set test must go red · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · covers:the working set keeps the path
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `cmd/mrw/main.go` · --files-from re-trims: its test and §129 must go red · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · covers:files-from keeps the path
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `cmd/mrw/main.go` · --files-from re-trims: §129 goes red · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · covers:the binary serves x-space from a list
- 2026-09-25 · c43095d* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:cf6a059a2bc0e8094c2a395eb8795b4b1bb0d0a1b0cfe9bad27d8fffa3cdc4f8 · covers:no engine file changes

## Invariants

- Blank lines and `#` comments are skipped exactly as before.
- The working-set file format is unchanged.

## Risks

- A generated list with stray trailing spaces now names files that do not exist; they are reported UNREADABLE, exit 1.
- Windows cannot hold `x` and `x ` apart; the space fixtures skip there, visibly.

## Out of Scope

- A path whose first non-blank character is `#`, or that is only whitespace (permanent: boundary: one spec per line with no quoting)

## Stop Condition

Stop if keeping the path exactly needs a format change or an exit-code change.
