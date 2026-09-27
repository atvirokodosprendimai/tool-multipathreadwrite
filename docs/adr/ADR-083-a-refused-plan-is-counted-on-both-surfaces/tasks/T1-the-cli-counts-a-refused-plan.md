# Task ADR-083-T1: the CLI counts a plan refused after it parsed

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the `refused_apply` count in the CLI write action
**Consumes:** `authoring.Record`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a filesystem refusal counts`, `a refused dry run counts`, `a post-parse refusal counts`, `a pre-parse refusal does not count`, `a contract row drives the binary`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

A plan naming a directory exited 2 on the CLI and `mrw stats` recorded nothing, while `mrw_write` counted it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/tally083_test.go` | new | the class, in-process: a directory, the same under `--dry-run`, a pointer naming two entries, a missing plan file, a clean write |
| `cmd/mrw/main.go` | edit | `parsed`; `refuseWith` counts a post-parse refusal; the filesystem-error branch counts one |
| `scripts/contract.sh` | edit | §164 drives the binary |
| `docs/adr/BACKLOG.md` | edit | the entry closed |

## Ordered Steps

1. [S1] Write `TestAPlanRefusedAfterItParsedIsOneRefusal`; it fails on v1.28.0, where the directory plan leaves the tally empty. [proof: mutation]
2. [S2] Count the refusals in `cmd/mrw/main.go`. [proof: mutation]
3. [S3] Contract §164 pairs the refused plan with a clean write beside it. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 180s -run 'TestAPlanRefusedAfterItParsedIsOneRefusal' -v 2>&1 | tee /tmp/adr083-T1.out \
  && missing=$(for t in TestAPlanRefusedAfterItParsedIsOneRefusal; do grep -qE "^--- PASS: $t \(" /tmp/adr083-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 164\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/mcp internal/authoring \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/mcp internal/authoring)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanRefusedAfterItParsedIsOneRefusal` | `cmd/mrw/tally083_test.go` | each post-parse refusal is one `refused_apply`, a pre-parse refusal is none, a clean write is `applied` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the count in `refuseWith` and the filesystem-error branch |
| 2 — something selects it | every CLI write that parses and is refused before landing |
| 3 — the caller can discover it | `mrw stats` and `mrw stats --json` |
| 4 — it is used | the review of #240 found the gap; ADR-009 refuses telemetry, so use is not observed |

## Verification Log
(empty until execute)
- 2026-09-27 · 5ababa0* · exit 1 · `set -o pipefail …` · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · ms:929 · test-lock-sha256:28e2f61120160385b3efa4a5ebd315ccd26dffc7e180c498514875cdafc1a6af · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvdGFsbHkwODNfdGVzdC5nbwlUZXN0QVBsYW5SZWZ1c2VkQWZ0ZXJJdFBhcnNlZElzT25lUmVmdXNhbAk3ZjU5MTkyZDNlZmEwZWM2M2MwMGVjNWIwM2E3NjJjNWIyYzIwNjgwOTk1YmI2YjUyYzkyMTBmYmVhMWNhMjIz
  ```
  --- last 10 line(s) of stdout
  === RUN   TestAPlanRefusedAfterItParsedIsOneRefusal
      tally083_test.go:48: a filesystem refusal: tally map[], want refused_apply 1, applied 0, plans 1
      tally083_test.go:53: a refused dry run: tally map[], want refused_apply 2, applied 0, plans 2
      tally083_test.go:58: a pointer refused after the plan parsed: tally map[], want refused_apply 3, applied 0, plans 3
      tally083_test.go:63: a plan file that could not be read: tally map[], want refused_apply 3, applied 0, plans 3
      tally083_test.go:68: a clean write beside them: tally map[applied:1], want refused_apply 3, applied 1, plans 4
  --- FAIL: TestAPlanRefusedAfterItParsedIsOneRefusal (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.191s
  FAIL
  ```
- 2026-09-27 · 5ababa0* · exit 1 · `set -o pipefail …` · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · ms:1516
  ```
  --- last 4 line(s) of stdout
  === RUN   TestAPlanRefusedAfterItParsedIsOneRefusal
  --- PASS: TestAPlanRefusedAfterItParsedIsOneRefusal (0.01s)
  PASS
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.188s
  ```
- 2026-09-27 · 5ababa0* · exit 0 · `set -o pipefail …` · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · ms:1255
- 2026-09-27 · 5ababa0* · exit 0 · `set -o pipefail …` · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · ms:538
- 2026-09-27 · 5ababa0* · exit 0 · `set -o pipefail …` · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · ms:347

## Mutation Log
(empty until execute)
- 2026-09-27 · 5ababa0* · mutant killed · exit 1 · `cmd/mrw/main.go` · refuseWith no longer counts a post-parse refusal · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · covers:a post-parse refusal counts
- 2026-09-27 · 5ababa0* · mutant killed · exit 1 · `cmd/mrw/main.go` · the filesystem-error branch no longer counts its refusal · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · covers:a filesystem refusal counts
- 2026-09-27 · 5ababa0* · mutant killed · exit 1 · `cmd/mrw/main.go` · a refusal before the plan parsed is counted too · acceptance-sha256:983f60e3878d71cf91d8d7aba297db444e6691944b1af823b93be0e9d1aabcf9 · covers:a pre-parse refusal does not count

## Invariants

- A plan is counted at most once, and a landed plan is never counted as a refusal.
- Exit codes and receipts are unchanged.

## Risks

- A future refusal added before `parsed` is set goes uncounted, correctly: it is pre-parse by construction.

## Out of Scope

- `mrw_write`'s tally (permanent: boundary: it already counts these; this task matches it)

## Stop Condition

The fence exits 0 and contract §164 passes.
