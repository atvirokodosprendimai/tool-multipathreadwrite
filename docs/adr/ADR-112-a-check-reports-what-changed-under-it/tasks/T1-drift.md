# Task ADR-112-T1: the files a write touched that changed since are named

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `writer.Drift`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the files a write touched that changed since are named`

## Goal

`writer.Drift(root, res)` returns, sorted, each file the write touched — written, not removed, with a `sha_after` — whose bytes no longer hash to it, or that is gone or no longer a regular file, looking at a rename's destination and a symlink's target.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/drift.go` | add | `Drift` |
| `internal/writer/drift112_test.go` | add | the test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestDriftNamesAFileChangedAfterTheWrite`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: the hash comparison removed; a missing file not counted. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/writer/ -count=1 -timeout 300s -run 'TestDriftNamesAFileChangedAfterTheWrite' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestDriftNamesAFileChangedAfterTheWrite \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestDriftNamesAFileChangedAfterTheWrite` | `internal/writer/drift112_test.go` | after a three-file write nothing has drifted; after one file is changed and one removed, `Drift` names those two, sorted | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every CLI write whose check runs |
| 3 — the caller can discover it | the `drift:` line and the `drift` key name each file |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-10-01 Codex design review (B5) |

## Mutation Log
- 2026-10-01 · e396f7e* · mutant killed · exit 1 · `internal/writer/drift.go` · the hash comparison removed: a changed file holds · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2
- 2026-10-01 · e396f7e* · mutant killed · exit 1 · `internal/writer/drift.go` · a missing file not counted: a removed file holds · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2

## Invariants

- Exit codes are the check's; a write with no check, or no drift, prints and carries what it did before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-112 task, in its own file.

## Verification Log
- 2026-10-01 · e396f7e* · exit 1 · `set -o pipefail …` · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2 · ms:352 · test-lock-sha256:d0040fdd338913e986cfae62ef508b92262e55345a8756ba1a2cf04835c87d82 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvd3JpdGVyL2RyaWZ0MTEyX3Rlc3QuZ28JVGVzdERyaWZ0TmFtZXNBRmlsZUNoYW5nZWRBZnRlclRoZVdyaXRlCTdiYzE4YzFjNzE5MjRmY2Y1NGNlNWJiMWI3MTQ0NTkxMjE3MjYyYWM2MGY4ZTk4MzZhMjY1ODBlZmFjZWRkNmQ
  ```
  --- last 5 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer.test]
  internal/writer/drift112_test.go:32:10: undefined: Drift
  internal/writer/drift112_test.go:41:10: undefined: Drift
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer [build failed]
  FAIL
  ```
- 2026-10-01 · e396f7e* · exit 0 · `set -o pipefail …` · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2 · ms:772
- 2026-10-01 · e396f7e* · exit 0 · `set -o pipefail …` · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2 · ms:656
