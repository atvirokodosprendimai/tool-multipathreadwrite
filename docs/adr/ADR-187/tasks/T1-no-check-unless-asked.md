# Task ADR-187-T1: a write runs no check unless asked for

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the new defaults in `cmd/mrw/main.go` and `internal/mcp/tools.go`; `TestAWriteRunsNoCheckUnlessAskedFor`, `TestAnMCPWriteRunsNoCheckUnlessAskedFor`; contract §322
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a CLI write runs no check unless --check`, `an MCP write runs no check unless check: true`

## Goal

Decisions 1-4 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/check187_test.go` | new | the CLI red test |
| `internal/mcp/check187_test.go` | new | the MCP red test |
| `cmd/mrw/main.go` | edit | the write command's default mode, `--no-check`'s help, the write help paragraph on the check |
| `internal/mcp/tools.go` | edit | `mrw_write`'s default mode and its description |
| `internal/guide/guide.go` | edit | recipe 6's "a code write already runs the project's check" |
| the tests that asserted the automatic check | edit | each asks for the check (`--check` or `check: true`) |
| `scripts/contract.sh` | edit | §322; the rows that read the automatic check's exit 3 ask for it |
| `README.md`, `AGENTS.md`, `docs/agents.md` | edit | the check is opt-in |

## Ordered Steps

1. [S1] Write `TestAWriteRunsNoCheckUnlessAskedFor`: in a Go module whose test fails, a `.go` write with no flag exits 0, prints no check line and changes the file; with `--no-check` the same; with `--check` it exits 3. And `TestAnMCPWriteRunsNoCheckUnlessAskedFor`: the same through `mrw_write`, with no `check` and with `check: false` no check runs; with `check: true` it does, and the answer reports it failed. See both RED.
2. [S2] The CLI default becomes `CheckOff`. Mutant: start from `CheckAuto` again — the CLI test goes red. [proof: mutation]
3. [S3] The MCP default becomes `CheckOff`, and `check: true` selects `CheckAuto`. Mutant: start from `CheckAuto` — the MCP test goes red. [proof: mutation]
4. [S4] Each test and contract row that asserted the automatic check asks for it; help, the tool description, recipe 6, README, AGENTS.md and docs/agents.md say the check is opt-in. [proof: acceptance]
5. [S5] Contract §322: in a fixture module whose test fails, a `.go` write with no flag exits 0 and runs no check; the pair, the same write with `--check`, exits 3. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ ./internal/mcp/ -count=1 -v -run '^(TestAWriteRunsNoCheckUnlessAskedFor|TestAnMCPWriteRunsNoCheckUnlessAskedFor)$' 2>&1) \
  && grep -qE '^--- PASS: TestAWriteRunsNoCheckUnlessAskedFor \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAnMCPWriteRunsNoCheckUnlessAskedFor \(' <<<"$out" \
  && go test ./cmd/mrw/ ./internal/mcp/ ./internal/writer/ ./internal/guide/ -count=1 >/dev/null \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 322\. ' scripts/contract.sh \
  && grep -qF 'PASS  a .go write with no flag runs no check and exits 0' <<<"$c" \
  && grep -qF 'PASS  the pair: the same write with --check exits 3' <<<"$c" \
  && ! grep -qF 'runs the project'"'"'s check by default' AGENTS.md README.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteRunsNoCheckUnlessAskedFor` | `cmd/mrw/check187_test.go` | no flag and `--no-check` run no check; `--check` runs it, exit 3 on failure | none | S1, S2 |
| `TestAnMCPWriteRunsNoCheckUnlessAskedFor` | `internal/mcp/check187_test.go` | no `check` and `check: false` run none; `check: true` runs it | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | every `mrw write` and `mrw_write` call |
| 3 — the caller can discover it | `mrw write --help`, the tool description, `mrw instructions` |
| 4 — it is used | no telemetry (ADR-009); the evidence is qh-core's report (#102) and this repository's own fallbacks |

## Mutation Log
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the CLI starts from the automatic check again · acceptance-sha256:2886d13bc86e5b0218c7cf88c36cba3f13c23e3ab139558badfa4ba40d89bc41 · covers:a CLI write runs no check unless --check
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: mrw_write starts from the automatic check again · acceptance-sha256:2886d13bc86e5b0218c7cf88c36cba3f13c23e3ab139558badfa4ba40d89bc41 · covers:an MCP write runs no check unless check: true

## Invariants

- `--check`, `check: true`, exit 3, `scoped_check`, steps and `mrw check` keep their meaning.
- No engine package changes.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0; or a test cannot keep testing the check by asking for it: stop and bring it to Zy.

## Out of Scope

- Removing `--no-check` (permanent: boundary: ADR-187 Out of Scope)

## Verification Log
- 2026-11-01 · de68a89a* · exit 1 · `set -o pipefail …` · acceptance-sha256:2886d13bc86e5b0218c7cf88c36cba3f13c23e3ab139558badfa4ba40d89bc41 · ms:502 · test-lock-sha256:b9749ad5ee062db89a654c1a1439dd7dd75af4a90578773438d4d3b99f362b91 · test-lock-file:docs/adr/locks/b9749ad5ee062db89a654c1a1439dd7dd75af4a90578773438d4d3b99f362b91.lock
  ```
  ```
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:2886d13bc86e5b0218c7cf88c36cba3f13c23e3ab139558badfa4ba40d89bc41 · ms:141974
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:2886d13bc86e5b0218c7cf88c36cba3f13c23e3ab139558badfa4ba40d89bc41 · ms:143270
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:2886d13bc86e5b0218c7cf88c36cba3f13c23e3ab139558badfa4ba40d89bc41 · ms:141234
- 2026-11-01 · 5baa630e* · exit 0 · `set -o pipefail …` · acceptance-sha256:2886d13bc86e5b0218c7cf88c36cba3f13c23e3ab139558badfa4ba40d89bc41 · ms:150277
