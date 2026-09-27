# Task ADR-083-T2: mrw_write counts the same refusals, and a ledger-failed landing

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the `refused_apply` and `applied` counts at `mrw_write`'s early returns
**Consumes:** `authoring.Record`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `an mrw_write pointer refusal counts`, `a refused MCP dry run counts`, `an mrw_write landing is counted once`, `a contract row drives the binary`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

`mrw_write` returned before its tally on a pointer, working-set or ledger-snapshot refusal, on its result-ceiling refusal and on a ledger failure after landing, so it disagreed with the CLI (Codex review of #251).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tally083_test.go` | new | a pointer naming two entries, the same as a dry run, a pointer past the set, a pointer that resolves |
| `internal/mcp/tools.go` | edit | each early return after the body files loaded records its outcome |
| `scripts/contract.sh` | edit | §165 drives `mrw mcp` |

## Ordered Steps

1. [S1] Write `TestAnMCPPlanRefusedAfterItParsedIsOneRefusal`; it fails on v1.28.0, where the pointer refusal leaves the tally empty. [proof: mutation]
2. [S2] Record the outcome at each early return in the write tool. [proof: mutation]
3. [S3] Contract §165 pairs the refused pointer plan with one that resolves and lands. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -timeout 180s -run 'TestAnMCPPlanRefusedAfterItParsedIsOneRefusal' -v 2>&1 | tee /tmp/adr083-T2.out \
  && missing=$(for t in TestAnMCPPlanRefusedAfterItParsedIsOneRefusal; do grep -qE "^--- PASS: $t \(" /tmp/adr083-T2.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 165\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/authoring \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc internal/authoring)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnMCPPlanRefusedAfterItParsedIsOneRefusal` | `internal/mcp/tally083_test.go` | each pointer refusal is one `refused_apply`, dry run or not; a pointer that resolves lands as `applied` | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the counts at the write tool's early returns |
| 2 — something selects it | every `mrw_write` call that parses and is refused before landing |
| 3 — the caller can discover it | `mrw stats` over the same checkout |
| 4 — it is used | the Codex review of #251 traced the gap; ADR-009 refuses telemetry, so use is not observed |

## Verification Log
(empty until execute)
- 2026-09-27 · a8ff606* · exit 1 · `set -o pipefail …` · acceptance-sha256:dac40dcc45ace7184364ea2ecc5e3404c16a185a9fcbe060cf86212ee070596b · ms:1806 · test-lock-sha256:cccf224d064597a142659a977d95f5fb115515038d0b07915bbea631be34792f · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC90YWxseTA4M190ZXN0LmdvCVRlc3RBbk1DUFBsYW5SZWZ1c2VkQWZ0ZXJJdFBhcnNlZElzT25lUmVmdXNhbAlhOWMyZjQ5ZTY1MmUyMWI3YjZmZmE3ODk1ZmE1ZDJlZDhiZWM0NmRhYzg4MmI4ZDU0NmY2MzliZDI4YWM2NjIz
  ```
  --- last 9 line(s) of stdout
  === RUN   TestAnMCPPlanRefusedAfterItParsedIsOneRefusal
      tally083_test.go:38: a pointer naming two entries: tally map[], want refused_apply 1, applied 0, plans 1
      tally083_test.go:40: the same plan as a dry run: tally map[], want refused_apply 2, applied 0, plans 2
      tally083_test.go:42: a pointer past the working set: tally map[], want refused_apply 3, applied 0, plans 3
      tally083_test.go:50: a pointer that resolves: tally map[applied:1], want refused_apply 3, applied 1, plans 4
  --- FAIL: TestAnMCPPlanRefusedAfterItParsedIsOneRefusal (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.266s
  FAIL
  ```
- 2026-09-27 · a8ff606* · exit 0 · `set -o pipefail …` · acceptance-sha256:dac40dcc45ace7184364ea2ecc5e3404c16a185a9fcbe060cf86212ee070596b · ms:2132
- 2026-09-27 · a8ff606* · exit 0 · `set -o pipefail …` · acceptance-sha256:dac40dcc45ace7184364ea2ecc5e3404c16a185a9fcbe060cf86212ee070596b · ms:1967
- 2026-09-27 · a8ff606* · exit 0 · `set -o pipefail …` · acceptance-sha256:dac40dcc45ace7184364ea2ecc5e3404c16a185a9fcbe060cf86212ee070596b · ms:469

## Mutation Log
(empty until execute)
- 2026-09-27 · a8ff606* · mutant killed · exit 1 · `internal/mcp/tools.go` · a pointer naming several entries is not counted · acceptance-sha256:dac40dcc45ace7184364ea2ecc5e3404c16a185a9fcbe060cf86212ee070596b · covers:an mrw_write pointer refusal counts
- 2026-09-27 · a8ff606* · mutant killed · exit 1 · `internal/mcp/tools.go` · a pointer past the working set is not counted · acceptance-sha256:dac40dcc45ace7184364ea2ecc5e3404c16a185a9fcbe060cf86212ee070596b · covers:an mrw_write pointer refusal counts

## Invariants

- A plan is counted at most once on this surface, as on the CLI.
- Receipts and JSON-RPC error codes are unchanged.

## Risks

- The unreadable-working-set, unreadable-ledger and ceiling refusals have no hermetic fixture here; they share one statement shape with the pointer refusal the test drives, and the mutants name them.

## Out of Scope

- A fixture for an unreadable ledger or working set over MCP (permanent: boundary: making the state directory unreadable mid-call is the platform-dependent fixture ADR-034 already declined)

## Stop Condition

The fence exits 0 and contract §165 passes.
