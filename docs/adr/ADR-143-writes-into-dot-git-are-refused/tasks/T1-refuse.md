# Task ADR-143-T1: A write whose path lands in `.git` is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `rooted.GitDir`, the refusal in `apply.resolve`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a write whose path has a .git component, spelled or real, is refused before anything is written, and a read of it is not`

## Goal

Decisions 1–5 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/gitdir.go` | add | `GitDir` |
| `internal/apply/apply.go` | edit | `resolve` calls it |
| `internal/rooted/gitdir143_test.go`, `internal/apply/gitdir143_test.go`, `cmd/mrw/gitdir143_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §246 |
| `AGENTS.md` | edit | one sentence in section 4 |

## Ordered Steps

1. [S1] Write `TestAPathInsideADotGitIsRefused` (rooted), `TestAPlanThatTouchesDotGitWritesNothing` (apply) and `TestWriteIntoDotGitIsRefusedAndReadIsNot` (cmd/mrw). Confirm RED.
2. [S2] `rooted.GitDir` and its call in `apply.resolve`. Mutants: the call removed; the real-location check removed; the case fold removed; the root's own components ignored. [proof: mutation]
3. [S3] Contract §246 and AGENTS.md. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/rooted/ -count=1 -timeout 300s -run 'TestAPathInsideADotGitIsRefused' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPathInsideADotGitIsRefused \(' "$out" \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAPlanThatTouchesDotGitWritesNothing' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPlanThatTouchesDotGitWritesNothing \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestWriteIntoDotGitIsRefusedAndReadIsNot' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestWriteIntoDotGitIsRefusedAndReadIsNot \(' "$out" \
  && go test ./internal/rooted/ ./internal/apply/ -count=1 -timeout 900s \
  && grep -q '^# 246\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPathInsideADotGitIsRefused` | `internal/rooted/gitdir143_test.go` | `.git` components refused (spelled, folded, through a link, in the root, `git~1` on a Windows build); `.github`, `.gitignore`, `x.git` and `git` are not | none | S1, S2 |
| `TestAPlanThatTouchesDotGitWritesNothing` | `internal/apply/gitdir143_test.go` | create, replace, delete, unlink and a rename into or out of `.git` each fail their hunk and write nothing, siblings skip | none | S1, S2 |
| `TestWriteIntoDotGitIsRefusedAndReadIsNot` | `cmd/mrw/gitdir143_test.go` | the CLI exits 1 with the named reason for `--create` and a plan, and `mrw read` of the same path exits 0 | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.GitDir` |
| 2 — something selects it | the call in `apply.resolve`, which every hunk path and rename destination passes |
| 3 — the caller can discover it | the refusal names the cause and git as the tool; AGENTS.md |
| 4 — it is used | the Windows retest of v1.60.0 (2026-10-10) found the gap; no telemetry (ADR-009) |

## Invariants

- Reads of `.git` are unchanged.
- A refused plan writes nothing and its siblings skip (ADR-001); exit codes are the contract.
- A name that is not `.git` (`.github`, `.gitignore`, `x.git`) is never refused by this rule.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if refusing by the real location turns away a root or a path the existing suite or the contract legitimately writes.

## Out of Scope

- A hard link to a file under `.git`, and HFS+ ignorable code points (deferred: docs/adr/BACKLOG.md)

## Mutation Log

## Verification Log
