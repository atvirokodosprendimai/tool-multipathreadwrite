# Task ADR-156-T1: the task corpus and an oracle per task, proved without an agent

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** Zy
**Produces:** `bench/tasks/<id>/`, `internal/bench.Oracle`, `internal/bench.LoadTask`
**Consumes:** none
**Data dependency:** pinned commits of four public repositories on this machine (`bench/repos.json`); a task whose repository or commit is not here is skipped by name, never silently. The three fixture tasks carry their own base trees and run anywhere: they are what CI runs.
**Proof map:** v1
**Rests-on:** `an oracle fails on the base`, `an oracle passes on the reference fix`, `an oracle refuses a stray edit`, `the corpus has the registered shapes`, `no task carries host configuration`, `an instruction file keeps every line but those naming mrw`

## Goal

About 30 tasks in the shapes Decision 3 registers, each with an oracle whose three properties a test proves without an agent.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/task.go` | add | `LoadTask`, `LoadTasks`: instruction, repository, base commit, require and forbid, and the allowed regions read from the reference fix; `PrepareInstructions` (S5 renamed it from `StripInstructions`) |
| `internal/bench/materialize.go` | add | `Materialize`: the base tree from `git archive` at the pinned commit, or a fixture's `base/`, stripped and committed |
| `internal/bench/oracle.go` | add | `Oracle`: required change present, nothing outside the allowed regions changed, structural check of touched files |
| `internal/bench/oracle_test.go`, `helpers_test.go` | add | the three properties for every task; the registered shape counts; stripping |
| `bench/tasks/<id>/task.json`, `reference.patch`, `base/` (fixtures) | add | the corpus: 30 measured tasks, 3 fixtures |
| `bench/repos.json` | add | the four repositories, by URL and local path |
| `docs/bench/README.md` | add | the task, repository and receipt formats |

## Ordered Steps

1. [S1] Write the failing tests and see them red: `TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit` (for each task: the oracle exits non-zero on the base, 0 with the reference fix applied, and non-zero with the reference fix plus a one-line change outside every allowed region) and `TestTheCorpusHasTheRegisteredShapeCounts` (8, 6, 6, 4, 6 by shape; no task's base tree holds `CLAUDE.md`, `AGENTS.md`, `.claude/`, `.mcp.json`, `.cursor/` or `.codex/` once stripped).
2. [S2] `LoadTask` and `Oracle`, standard library only. [proof: mutation]
3. [S3] The corpus, with the instruction text naming no tool. [proof: acceptance]
4. [S4] `docs/bench/README.md` with the formats. [proof: acceptance]
5. [S5] After the uncounted pilot, Zy, 2026-10-14 ("Redact, don't strip"): the prepared tree keeps `CLAUDE.md`, `CLAUDE.local.md`, `AGENTS.md` and `.cursorrules`, at any depth, without the lines that name mrw, and the files in the tree they import with `@path`, the same way; an instruction file that is a link into the tree stays a link with its target redacted, and one leading out of the tree is removed (the review of #413). It still removes `.claude/`, `.cursor/` and `.codex/` at any depth and `.mcp.json` at the root; a `.mcp.json` below the root is kept (agentsmemory's tests read its plugin's). agentsmemory's tests read its `AGENTS.md`, so a stripped tree failed configuration B's automatic check for a reason the harness made. `TestThePreparedTreeRedactsInstructionsAndStripsConfiguration` replaces `TestStrippingRemovesEveryInstructionFile`. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/bench/ -count=1 -timeout 600s -run 'TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit|TestTheCorpusHasTheRegisteredShapeCounts|TestThePreparedTreeRedactsInstructionsAndStripsConfiguration|TestTheOracleRefusesWhatTheReviewFound' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit \(' "$out" \
  && grep -qE '^--- PASS: TestTheCorpusHasTheRegisteredShapeCounts \(' "$out" \
  && grep -qE '^--- PASS: TestThePreparedTreeRedactsInstructionsAndStripsConfiguration \(' "$out" \
  && grep -qE '^--- PASS: TestTheOracleRefusesWhatTheReviewFound \(' "$out" \
  && [ -f docs/bench/README.md ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit` | `internal/bench/oracle_test.go` | each oracle's three properties, every task | none | S1, S2 |
| `TestTheCorpusHasTheRegisteredShapeCounts` | `internal/bench/oracle_test.go` | the registered shapes and stripped instruction files | none | S1, S3 |
| `TestThePreparedTreeRedactsInstructionsAndStripsConfiguration` | `internal/bench/oracle_test.go` | at any depth, an instruction file keeps every line but those naming mrw, byte for byte; configuration a host loads is gone; other files are untouched | none | S5 |
| `TestTheOracleRefusesWhatTheReviewFound` | `internal/bench/oracle_test.go` | a binary, a mode-only and a deleting change are outside; slack does not license deleting a neighbour; an exact task fails with its fix line commented out | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Oracle`, `LoadTask`, the corpus |
| 2 — something selects it | T2's runner grades every attempt with `Oracle` |
| 3 — the caller can discover it | `docs/bench/README.md` |
| 4 — it is used | nothing yet; T3 is its first use |

## Mutation Log
(empty until execute)
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/oracle.go` · a change outside the regions is never named · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · covers:an oracle refuses a stray edit
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/oracle.go` · an unmet require is never named · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · covers:an oracle fails on the base
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/oracle.go` · a clean tree is never solved · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · covers:an oracle passes on the reference fix
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/task.go` · a replaced line is placed three lines off · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · covers:an oracle passes on the reference fix
- 2026-10-14 · 05ec3f84* · mutant survived · exit 0 · `internal/bench/task.go` · a task may claim the fixture shape without a fixture tree · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · covers:the corpus has the registered shapes
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/task.go` · the registered multi-site shape no longer names the corpus shape · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · covers:the corpus has the registered shapes
- 2026-10-14 · 05ec3f84* · mutant survived · exit 0 · `internal/bench/task.go` · a .claude directory is no longer an instruction path · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · covers:no task carries instruction files
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/oracle.go` · a change outside the regions is never named · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:an oracle refuses a stray edit
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/oracle.go` · an unmet require is never named · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:an oracle fails on the base
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/oracle.go` · a clean tree is never solved · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:an oracle passes on the reference fix
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/task.go` · a replaced line is placed three lines off · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:an oracle passes on the reference fix
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/task.go` · the registered multi-site shape no longer names the corpus shape · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:the corpus has the registered shapes
- 2026-10-14 · 05ec3f84* · mutant killed · exit 1 · `internal/bench/task.go` · a .claude directory is no longer stripped · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:no task carries instruction files
- 2026-10-14 · c430d730* · mutant survived · exit 0 · `internal/bench/task.go` · slack licenses changing the lines beside the fix again · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:an oracle refuses a stray edit
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/task.go` · slack licenses changing the lines beside the fix again · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · covers:an oracle refuses a stray edit
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/oracle.go` · a change outside the regions is never named · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle refuses a stray edit
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/oracle.go` · an unmet require is never named · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle fails on the base
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/oracle.go` · a clean tree is never solved · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle passes on the reference fix
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/task.go` · a replaced line is placed three lines off · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle passes on the reference fix
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/task.go` · the registered multi-site shape no longer names the corpus shape · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:the corpus has the registered shapes
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/task.go` · a .claude directory is no longer stripped · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:no task carries instruction files
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/task.go` · slack licenses changing the lines beside the fix again · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle refuses a stray edit
- 2026-10-14 · c430d730* · mutant survived · exit 0 · `internal/bench/oracle.go` · a binary, mode-only or deleting change passes · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle refuses a stray edit
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/oracle.go` · an exact task accepts a fix line commented out · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle refuses a stray edit
- 2026-10-14 · c430d730* · mutant killed · exit 1 · `internal/bench/oracle.go` · a binary, mode-only or deleting change passes · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle refuses a stray edit
- 2026-10-14 · b16cc705* · mutant inconclusive · exit 1 · `internal/bench/oracle.go` · an exact task accepts a file that is not the reference result · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle refuses a stray edit
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-14 · b16cc705* · mutant killed · exit 1 · `internal/bench/oracle.go` · an exact task accepts a file that is not the reference result · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · covers:an oracle refuses a stray edit
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/bench/task.go` · an instruction file keeps the lines that name mrw · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/bench/task.go` · a host configuration directory is kept in the tree · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/bench/task.go` · a .mcp.json below the root, a plugin manifest its tests read, is removed · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · covers:no task carries instruction files
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/bench/task.go` · an instruction file keeps the lines that name mrw · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · covers:an instruction file keeps every line but those naming mrw
- 2026-10-14 · ef6fbe5f* · mutant killed · exit 1 · `internal/bench/task.go` · a file an instruction file imports keeps its lines naming mrw · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · covers:an instruction file keeps every line but those naming mrw
- 2026-10-14 · ef6fbe5f* · mutant killed · exit 1 · `internal/bench/task.go` · a letter of another script beside mrw no longer makes another word · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · covers:an instruction file keeps every line but those naming mrw
- 2026-10-14 · ef6fbe5f* · mutant killed · exit 1 · `internal/bench/task.go` · an instruction file that is a link into the tree is removed · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · covers:no task carries host configuration

## Verification Log
(empty until execute)
- 2026-10-14 · 05ec3f84* · exit 1 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:200 · test-lock-sha256:45c71ef44fc8e8d8e21dfcf654fbbf4817caf71508247e38a5881702add01480 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CWM0ZDQ3MjEwNWFlNzM0Mjk0MjgzNjFhOTg3MGVkOGEyOTQxMmE4MGJiNjE4NzU3OWZjOWMwNGYyZGNmYmM5MGQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0U3RyaXBwaW5nUmVtb3Zlc0V2ZXJ5SW5zdHJ1Y3Rpb25GaWxlCTNmYTA0ZjE3ZDAyOWZkYWFkZjA3ODUzZWViZDlhNzJmMjA3Mzc5YmYzMTNkNjA0OTU0N2YzOTEwOTYwMGNmOGQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjM
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/bench/analyze_test.go:16:106: undefined: Verdict
  internal/bench/analyze_test.go:17:4: undefined: Receipt
  internal/bench/analyze_test.go:17:106: undefined: Verdict
  internal/bench/analyze_test.go:39:18: undefined: Receipt
  internal/bench/oracle_test.go:99:35: undefined: Task
  internal/bench/runner_test.go:47:45: undefined: Task
  internal/bench/runner_test.go:70:32: undefined: RunOptions
  internal/bench/analyze_test.go:39:18: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench [build failed]
  FAIL
  ```
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:7472
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:7006
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:6467
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:15402
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:11411
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:12423
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:8819f5890c9bd141c4715555570723c209397fc2882b0714970bb3e5099deb25 · ms:14607
- 2026-10-14 · 05ec3f84* · exit 1 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:314
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/bench/analyze_test.go:16:106: undefined: Verdict
  internal/bench/helpers_test.go:12:9: undefined: Task
  internal/bench/helpers_test.go:23:9: undefined: Task
  internal/bench/helpers_test.go:42:9: undefined: Task
  internal/bench/oracle_test.go:100:35: undefined: Task
  internal/bench/runner_test.go:52:45: undefined: Task
  internal/bench/runner_test.go:75:32: undefined: RunOptions
  internal/bench/analyze_test.go:16:106: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench [build failed]
  FAIL
  ```
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:9300
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:7552
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:7408
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:7502
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:6778
- 2026-10-14 · 05ec3f84* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:6558
- 2026-10-14 · 05ec3f84* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:0 · test-lock-sha256:d20fdfd7b2d55711a4a77c1825717892a41e9bff7a87fd5c0ca8554c13b46416 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTI0MGM4MThhYWQ4ZTAwZDMxNDY3ZDk0MDQ5YzZkYWUxOGUxZDBmMmVhYWNmZDViNzA4ODQzYjFhMmMxODE1ZWQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0U3RyaXBwaW5nUmVtb3Zlc0V2ZXJ5SW5zdHJ1Y3Rpb25GaWxlCTNmYTA0ZjE3ZDAyOWZkYWFkZjA3ODUzZWViZDlhNzJmMjA3Mzc5YmYzMTNkNjA0OTU0N2YzOTEwOTYwMGNmOGQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjM · test-lock-kind:replace
- 2026-10-14 · c430d730* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:0 · test-lock-sha256:5e2cfc4f866f02563e25e55a9ebc2f004c9363ee65bb7a2e17da6a092bbf9770 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTA5MDQ4MmM0MDM3MDdjYjJlYzQyMWNiOTBmZjQ5MDRlZjFjNDNiOTk1N2ViZWY5MjdkZjRmN2E3ODcxYjAzYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0U3RyaXBwaW5nUmVtb3Zlc0V2ZXJ5SW5zdHJ1Y3Rpb25GaWxlCTNmYTA0ZjE3ZDAyOWZkYWFkZjA3ODUzZWViZDlhNzJmMjA3Mzc5YmYzMTNkNjA0OTU0N2YzOTEwOTYwMGNmOGQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjMKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAkxYWRjZmYzNTJkYTY4ZGZjMWIwNWJiMDk5MjY4Mzg4M2RkOThiNjk5NWZmNDVhNTkzNGM3MGFkMzMwMzlkZDU1 · test-lock-kind:replace
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:11863
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:b26ca682576563c93de14c1e691c67615d531bc81018296298e0ee4b9e7fcdd4 · ms:27933
- 2026-10-14 · c430d730* · exit 1 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:120
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/bench/helpers_test.go:23:9: undefined: Task
  internal/bench/helpers_test.go:42:9: undefined: Task
  internal/bench/helpers_test.go:76:9: undefined: Task
  internal/bench/oracle_test.go:122:35: undefined: Task
  internal/bench/oracle_test.go:211:81: undefined: Task
  internal/bench/runner_test.go:54:45: undefined: Task
  internal/bench/runner_test.go:77:32: undefined: RunOptions
  internal/bench/analyze_test.go:12:11: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench [build failed]
  FAIL
  ```
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:8003
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:8004
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:8003
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:7972
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:8069
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:7991
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:7956
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:7983
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:7987
- 2026-10-14 · c430d730* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:0 · test-lock-sha256:2ed71a201fd26aa12d825f27e83386184ed9bb32fd0a1cf5d49050f41ca77180 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTA5MDQ4MmM0MDM3MDdjYjJlYzQyMWNiOTBmZjQ5MDRlZjFjNDNiOTk1N2ViZWY5MjdkZjRmN2E3ODcxYjAzYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0U3RyaXBwaW5nUmVtb3Zlc0V2ZXJ5SW5zdHJ1Y3Rpb25GaWxlCTNmYTA0ZjE3ZDAyOWZkYWFkZjA3ODUzZWViZDlhNzJmMjA3Mzc5YmYzMTNkNjA0OTU0N2YzOTEwOTYwMGNmOGQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjMKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAkxZjUxYWRmMjYzOTAzYThlYmJiN2JjZmRmYzE2M2VlMzlkMGIzNzkwYTQ4NGRkNTQzMzViODI4NDcyNTdmNWY2 · test-lock-kind:replace
- 2026-10-14 · c430d730* · exit 1 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:8663
  ```
  --- last 10 line(s) of stdout (of 78 after folding 78 raw)
  === RUN   TestTheCorpusHasTheRegisteredShapeCounts
  --- PASS: TestTheCorpusHasTheRegisteredShapeCounts (0.00s)
  === RUN   TestStrippingRemovesEveryInstructionFile
  --- PASS: TestStrippingRemovesEveryInstructionFile (0.00s)
  === RUN   TestTheOracleRefusesWhatTheReviewFound
      oracle_test.go:294: a mode-only change: {Solved:true Missing:[] Outside:[] Structure:[]}, want it outside
  --- FAIL: TestTheOracleRefusesWhatTheReviewFound (0.73s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench	7.917s
  FAIL
  ```
- 2026-10-14 · c430d730* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:0 · test-lock-sha256:2ed71a201fd26aa12d825f27e83386184ed9bb32fd0a1cf5d49050f41ca77180 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTA5MDQ4MmM0MDM3MDdjYjJlYzQyMWNiOTBmZjQ5MDRlZjFjNDNiOTk1N2ViZWY5MjdkZjRmN2E3ODcxYjAzYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0U3RyaXBwaW5nUmVtb3Zlc0V2ZXJ5SW5zdHJ1Y3Rpb25GaWxlCTNmYTA0ZjE3ZDAyOWZkYWFkZjA3ODUzZWViZDlhNzJmMjA3Mzc5YmYzMTNkNjA0OTU0N2YzOTEwOTYwMGNmOGQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjMKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAkxZjUxYWRmMjYzOTAzYThlYmJiN2JjZmRmYzE2M2VlMzlkMGIzNzkwYTQ4NGRkNTQzMzViODI4NDcyNTdmNWY2 · test-lock-kind:replace
- 2026-10-14 · c430d730* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:7997
- 2026-10-14 · b16cc705* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:0 · test-lock-sha256:d8b4e577db98779e7b89829ddfbb2e6693c50d34571da25ac54ed02099904644 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTA5MDQ4MmM0MDM3MDdjYjJlYzQyMWNiOTBmZjQ5MDRlZjFjNDNiOTk1N2ViZWY5MjdkZjRmN2E3ODcxYjAzYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0U3RyaXBwaW5nUmVtb3Zlc0V2ZXJ5SW5zdHJ1Y3Rpb25GaWxlCTNmYTA0ZjE3ZDAyOWZkYWFkZjA3ODUzZWViZDlhNzJmMjA3Mzc5YmYzMTNkNjA0OTU0N2YzOTEwOTYwMGNmOGQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjMKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAk0OTlkMWRiNmY0NzlkMTY2Mzk1OGVjYWY3MTIxYzIwMDk3NmU5OGU2ZTYxYjQ5MGEyZjZiZjhmNTBkMzYwY2Rh · test-lock-kind:replace
- 2026-10-14 · b16cc705* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:16713
- 2026-10-14 · b16cc705* · exit 0 · `set -o pipefail …` · acceptance-sha256:bb008abbb4d498b8eb0f35f0c592bb22501454608c3adb7ad1d1a116977d290d · ms:13934
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:12356
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:9434
- 2026-10-14 · be76e27f* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:0 · test-lock-sha256:2285fb29832a64ed12d8ecf730d67d5a4bca44540a53dfd635c9d5f047dfc19f · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTA5MDQ4MmM0MDM3MDdjYjJlYzQyMWNiOTBmZjQ5MDRlZjFjNDNiOTk1N2ViZWY5MjdkZjRmN2E3ODcxYjAzYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjMKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAk0OTlkMWRiNmY0NzlkMTY2Mzk1OGVjYWY3MTIxYzIwMDk3NmU5OGU2ZTYxYjQ5MGEyZjZiZjhmNTBkMzYwY2RhCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZVByZXBhcmVkVHJlZVJlZGFjdHNJbnN0cnVjdGlvbnNBbmRTdHJpcHNDb25maWd1cmF0aW9uCWQwYTFhMGI1NWQ0ZDFlMjdmZjNkMTkwOWU3ZjU2ZThlMzYzZGI0M2VjNzlmOWU2YjQ4N2NkZDlkMzEwYmYwYjI · test-lock-kind:replace
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:9110
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:9119
- 2026-10-14 · be76e27f* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:0 · test-lock-sha256:ca147e95efa49eb8f635659c94b1b293c4f87526ca74b66e2fbf56506cd2b98c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTA5MDQ4MmM0MDM3MDdjYjJlYzQyMWNiOTBmZjQ5MDRlZjFjNDNiOTk1N2ViZWY5MjdkZjRmN2E3ODcxYjAzYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjMKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAk0OTlkMWRiNmY0NzlkMTY2Mzk1OGVjYWY3MTIxYzIwMDk3NmU5OGU2ZTYxYjQ5MGEyZjZiZjhmNTBkMzYwY2RhCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZVByZXBhcmVkVHJlZVJlZGFjdHNJbnN0cnVjdGlvbnNBbmRTdHJpcHNDb25maWd1cmF0aW9uCWIzNjRmOGM4MzIyMjQxMDQxNGE3MzRiZTVlMDEzYjJkNDliM2EzOWZkOGZmMDI2NjJjNWM1MThlOTZhODVmZmU · test-lock-kind:replace
- 2026-10-14 · ef6fbe5f* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:13370
- 2026-10-14 · ef6fbe5f* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:32719
- 2026-10-14 · ef6fbe5f* · exit 0 · `set -o pipefail …` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:9952
- 2026-10-14 · ef6fbe5f* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:cd14e821fd3ff6291b010edb18924ae75b0dcc0ef961e209d9a16b0784246d0f · ms:0 · test-lock-sha256:d7729c617e3557eea75441919f47472b666e5394a199f622de02d5df6f75d5cd · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdEV2ZXJ5T3JhY2xlRmFpbHNPbkJhc2VQYXNzZXNPbkZpeEFuZFJlZnVzZXNBU3RyYXlFZGl0CTA5MDQ4MmM0MDM3MDdjYjJlYzQyMWNiOTBmZjQ5MDRlZjFjNDNiOTk1N2ViZWY5MjdkZjRmN2E3ODcxYjAzYzQKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlQ29ycHVzSGFzVGhlUmVnaXN0ZXJlZFNoYXBlQ291bnRzCWU1ZDdjNDY0YzM4YzY1YTIxMzBjYmUzYjE2YjAxZDkwZTRhYTE3OTJmNWQ4ZDUzMTY2N2Y5MmM3Y2Y5MTkyYjMKYm9keQlpbnRlcm5hbC9iZW5jaC9vcmFjbGVfdGVzdC5nbwlUZXN0VGhlT3JhY2xlUmVmdXNlc1doYXRUaGVSZXZpZXdGb3VuZAk0OTlkMWRiNmY0NzlkMTY2Mzk1OGVjYWY3MTIxYzIwMDk3NmU5OGU2ZTYxYjQ5MGEyZjZiZjhmNTBkMzYwY2RhCmJvZHkJaW50ZXJuYWwvYmVuY2gvb3JhY2xlX3Rlc3QuZ28JVGVzdFRoZVByZXBhcmVkVHJlZVJlZGFjdHNJbnN0cnVjdGlvbnNBbmRTdHJpcHNDb25maWd1cmF0aW9uCTIxNzUyMzgwMDgzNWIxNjYzMjM5YjFkZDJhNzJiZmQ4YWM3M2M5YTc1MWQwZWY1MzUxNjI4Y2JhMzkwMmEyN2U · test-lock-kind:replace

## Invariants

- An oracle never consults a model or a rater.
- `go.mod` keeps one requirement.

## Risks

- A task whose public repository moves or disappears: pinned by full commit and URL; the test names a task it cannot reach rather than skipping it silently, and CI reaches only the fixtures.

## Out of Scope

- Private repositories (permanent: boundary: Decision 3)

## Stop Condition

The fence exits 0.
