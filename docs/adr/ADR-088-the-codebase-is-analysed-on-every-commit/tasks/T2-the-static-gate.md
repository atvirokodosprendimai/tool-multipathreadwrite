# Task ADR-088-T2: the static gate, clean

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `scripts/static.sh`; `.golangci.yml`; the CI step
**Consumes:** a codebase deadcode and U1000 pass on (T1)
**Data dependency:** hermetic, except `govulncheck` reads the Go vulnerability database
**Proof map:** v1
**Rests-on:** `static.sh runs every analyser`, `a lint finding fails the gate`, `CI runs the gate`

## Goal

No linter, dead-code check or vulnerability scan ran anywhere; `golangci-lint` found 19 defects with this record's configuration at `d352ba7`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/static.sh` | new | the gate |
| `.golangci.yml` | new | the linters and every exclusion's reason |
| `internal/check/check.go`, `internal/mcp/tools.go`, `internal/read/astgrep.go`, `internal/read/walk.go`, `internal/state/state.go` | edit | errors wrapped with `%w`; `errors.Is`; wastedassign |
| `internal/authoring/authoring.go` | edit | `recordRecent`/`recordPricing` return nothing; the swallow in `save` annotated |
| `internal/mcp/ack.go`, `internal/mcp/root.go`, `internal/read/walk.go`, `internal/rooted/rooted.go` | edit | deliberate swallows annotated |
| `internal/seen/seen_test.go`, `cmd/mrw/iterroot_test.go`, `cmd/mrw/prune_test.go`, `cmd/mrw/paddedstress_test.go` | edit | SA4000, ST1008, copyloopvar |
| `.github/workflows/ci.yml`, `CONTRIBUTING.md`, `AGENTS.md` | edit | CI step; the gate lists |

## Ordered Steps

1. [S1] The fence fails on `d352ba7`: `scripts/static.sh` does not exist. [proof: acceptance]
2. [S2] `.golangci.yml` and `scripts/static.sh`. [proof: mutation]
3. [S3] Fix every finding. [proof: mutation]
4. [S4] CI runs the script; CONTRIBUTING.md and AGENTS.md list it. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
./scripts/static.sh > /tmp/adr088-T2.out 2>&1 \
  && grep -q '^static analysis clean$' /tmp/adr088-T2.out \
  && for s in gofmt vet golangci-lint deadcode staticcheck govulncheck; do grep -q "^== $s" /tmp/adr088-T2.out || exit 1; done \
  && grep -q 'scripts/static.sh' .github/workflows/ci.yml \
  && grep -q 'scripts/static.sh' CONTRIBUTING.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/lines internal/iter internal/subproc \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the fence runs the gate itself; the packages' own tests keep passing | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `scripts/static.sh` |
| 2 — something selects it | the CI `test` job and T3's hook |
| 3 — the caller can discover it | CONTRIBUTING.md and AGENTS.md list it |
| 4 — it is used | every CI run |

## Verification Log
(empty until execute)
- 2026-09-27 · d352ba7* · exit 127 · `set -o pipefail …` · acceptance-sha256:d7a8a93afe02ea4800de9dcdfb9391b8156af80195c6f0b7e9a9eb5d52397ba2 · ms:79 · test-lock-sha256:d0a88974807379b1a850b3ee812a98f67f902bffe32b9e36b9a17cac84f8be11 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMw
  ```
  ```
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b206ca3ad105d9c93bbefb2ea1e080eab419be380b4d924f8ada0d467b555a9 · ms:14430
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b206ca3ad105d9c93bbefb2ea1e080eab419be380b4d924f8ada0d467b555a9 · ms:7096
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b206ca3ad105d9c93bbefb2ea1e080eab419be380b4d924f8ada0d467b555a9 · ms:7018
- 2026-09-27 · d352ba7* · exit 0 · `set -o pipefail …` · acceptance-sha256:0b206ca3ad105d9c93bbefb2ea1e080eab419be380b4d924f8ada0d467b555a9 · ms:7110

## Mutation Log
(empty until execute)
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `scripts/static.sh` · the script stops running one analyser · acceptance-sha256:0b206ca3ad105d9c93bbefb2ea1e080eab419be380b4d924f8ada0d467b555a9 · covers:static.sh runs every analyser
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `internal/mcp/tools.go` · a wasted assignment goes back into production · acceptance-sha256:0b206ca3ad105d9c93bbefb2ea1e080eab419be380b4d924f8ada0d467b555a9 · covers:a lint finding fails the gate
- 2026-09-27 · d352ba7* · mutant killed · exit 1 · `.github/workflows/ci.yml` · CI stops running the gate · acceptance-sha256:0b206ca3ad105d9c93bbefb2ea1e080eab419be380b4d924f8ada0d467b555a9 · covers:CI runs the gate

## Invariants

- Every refusal's text and every exit code is unchanged: `%w` formats as `%v` does.

## Risks

- None beyond the record's.

## Out of Scope

- The linters the record declines (permanent: boundary: see the record's Alternatives)

## Stop Condition

The fence exits 0.
