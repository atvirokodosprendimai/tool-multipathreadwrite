# Task ADR-191-T1: `mrw hook read` records what a host Read showed

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `mrw hook read`; `TestAHostReadLicensesOnlyTheLinesItShowedAsTheyAre`; contract §329; the hook block in `docs/agents.md`
**Consumes:** none
**Data dependency:** a real Claude Code PostToolUse payload for Read, captured once (S1)
**Proof map:** v1
**Rests-on:** `a host read licenses only lines shown whole`, `the hook never fails the host`

## Goal

Decisions 1-4 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/hookread191_test.go` | new | the red test |
| `cmd/mrw/hookread191.go` | new | the subcommand |
| `cmd/mrw/main.go` | edit | register `hook` and its `read` command |
| `internal/seen/seen.go` | edit only if `Record` needs a caller it lacks | reuse the read's record path |
| `docs/agents.md` | edit | the PostToolUse block |
| `scripts/contract.sh` | edit | §329 |

## Ordered Steps

1. [S1] For each case (a whole file, an offset and limit, a line over 2,000 characters, a file past Read's size limit, a non-text file), capture the PostToolUse payload together with the same call's tool result in the session transcript. Save both as test fixtures, with synthetic file content only, and record the Claude Code version. Choose the payload representation that equals the transcript's in every case. If none does, stop: the record's Decision 4. [proof: human: the payload and the transcript are the host's, not ours]
2. [S2] Write the failing test `TestAHostReadLicensesOnlyTheLinesItShowedAsTheyAre` and see it RED. Cases, from the captured fixtures:
   - the returned lines are licensed, and a line outside them is not;
   - a returned line that differs from disk (cut, or changed after the read) is not licensed;
   - an unrecognised response shape, or an error response, records nothing;
   - a held ledger lock: the command returns within its deadline, records nothing, exits 0;
   - a legacy `./.mrw` present with its destination lock held: the command still returns within its deadline;
   - a path outside the root records nothing;
   - a non-Read tool records nothing;
   - a malformed payload records nothing and exits 0.
3. [S3] The subcommand. Mutants, each killed:
   - license the requested range rather than the returned content, so the outside-range case goes red;
   - skip the comparison with disk, so the changed-line case goes red;
   - wait on the lock without a deadline, so the held-lock case goes red;
   - run the startup migration, so the legacy-state case goes red.
   [proof: mutation]
4. [S4] `docs/agents.md` gains the hook block, and contract §329 drives the built binary. A Read payload on stdin licenses its lines, and a plan to one of them applies, exit 0. The pair: a plan to a line outside its range is refused, exit 1. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./cmd/mrw/ -count=1 -v -run '^TestAHostReadLicensesOnlyTheLinesItShowedAsTheyAre$' 2>&1) \
  && grep -qE '^--- PASS: TestAHostReadLicensesOnlyTheLinesItShowedAsTheyAre \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 329\. ' scripts/contract.sh \
  && grep -qF 'PASS  a host Read payload licenses the lines it showed' <<<"$c" \
  && grep -qF 'PASS  the pair: a line outside that read stays unlicensed' <<<"$c" \
  && grep -qF 'mrw hook read' docs/agents.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/read internal/apply internal/plan internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAHostReadLicensesOnlyTheLinesItShowedAsTheyAre` | `cmd/mrw/hookread191_test.go` | only lines shown whole are licensed; a bad or foreign payload records nothing and exits 0 | none | S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | a host's PostToolUse hook with matcher `Read` |
| 3 — the caller can discover it | `docs/agents.md`, `mrw hook --help` |
| 4 — it is used | no telemetry (ADR-009); the paid run measures it |

## Mutation Log

## Verification Log

## Invariants

- Nothing outside the root, in `.git` or in mrw's state is recorded.
- The hook exits 0 whatever it is given.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0, or S1 finds the payload over-claims, which stops the task.

## Out of Scope

- Grep and Bash output (permanent: boundary: ADR-191 Out of Scope)
