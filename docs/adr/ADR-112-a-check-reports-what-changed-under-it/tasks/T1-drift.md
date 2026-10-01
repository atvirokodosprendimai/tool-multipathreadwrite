# Task ADR-112-T1: the files a write touched that changed since are named

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `writer.Before`, `writer.Drift`, `writer.Snapshot`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the files a write touched that changed since are named`

## Goal

`writer.Before(root, res)` hashes each file the write touched — written, not removed — as it stands just before the check, streaming each hash; `writer.Drift(root, before)` returns, sorted, each whose bytes no longer hash as they did, or that is gone or no longer a regular file. The baseline is on disk, not the write's `sha_after`, which for a renamed relative symlink names the old referent (the Codex review of #309).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/drift.go` | add | `Drift` |
| `internal/writer/drift112_test.go` | add | the test |

## Ordered Steps

1. [S1] Write the failing test(s) `TestDriftNamesAFileChangedAfterTheWrite`; confirm RED. [proof: mutation]
2. [S2] The change named in Goal. Mutants: the hash comparison removed; the baseline taken from `sha_after` instead of the disk. [proof: mutation]

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
| `TestDriftNamesAFileChangedAfterTheWrite` | `internal/writer/drift112_test.go` | after a three-file write nothing has drifted, and after one file is changed and one removed `Drift` names those two, sorted; after a renamed relative symlink an idle check names nothing, and a change to its new referent names it | — | S1, S2 |

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
- 2026-10-01 · 7fe2f57* · mutant killed · exit 1 · `internal/writer/drift.go` · the hash comparison removed: nothing is ever drift · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2
- 2026-10-01 · 7fe2f57* · mutant killed · exit 1 · `internal/writer/drift.go` · the baseline taken from sha_after: a renamed relative symlink reports drift under an idle check · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2

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
- 2026-10-01 · 7fe2f57* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2 · ms:0 · test-lock-sha256:ac68825fe7a66019be76eaa1e8da87056ec2cb26441b085626f626f955afc98f · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvd3JpdGVyL2RyaWZ0MTEyX3Rlc3QuZ28JVGVzdERyaWZ0TmFtZXNBRmlsZUNoYW5nZWRBZnRlclRoZVdyaXRlCTZlODQ0ZjUwMmU3ODcwNjVlMmQ3NTgyNjUzYjcwYTc5N2Q1ZTYxY2VkOTZmZDQ4NjgwZTAyYmJhMjYwYmNmMmU · test-lock-kind:replace
- 2026-10-01 · human-observed · Claude's session observed the relock: after red the test moved to the Before/Drift snapshot API and gained the Codex review of #309's renamed-relative-symlink case (idle check names nothing; a change to the new referent is named); the three-file assertions are unchanged
- 2026-10-01 · 7fe2f57* · exit 0 · `set -o pipefail …` · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2 · ms:930
- 2026-10-01 · 7fe2f57* · exit 0 · `set -o pipefail …` · acceptance-sha256:0921d0e6ee54d3ee086f1e5e48587a85973fe7d6664c6f0402db564eaff3b6f2 · ms:694
