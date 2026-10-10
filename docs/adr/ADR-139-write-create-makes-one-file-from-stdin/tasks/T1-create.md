# Task ADR-139-T1: `--create PATH` compiles standard input to a create plan

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `ingest.CompileCreate`, `mrw write --create`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `write --create PATH makes the file a create plan of stdin's lines would, and refuses what a plan would refuse`

## Goal

Decisions 1–4 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/create.go` | add | `CompileCreate`: lines of the content to one create hunk |
| `cmd/mrw/main.go` | edit | the flag, its domain checks, the `create` case of the format switch |
| `internal/guide/guide.go` | edit | `mrw instructions` names the flag (a contract row requires every write flag taught) |
| `internal/ingest/create139_test.go`, `cmd/mrw/create139_test.go` | add | the tests |
| `scripts/contract.sh` | edit | §240 |
| `AGENTS.md`, `README.md` | edit | the write section |

## Ordered Steps

1. [S1] Write `TestCompileCreateMakesACreatePlanOfStdin` (the compiled plan parses to one create hunk with the content's lines; a line beginning `@@`, empty content and a path with a space are carried; UTF-16 and a NUL are refused; an absolute path is refused) and, in `cmd/mrw`, `TestWriteCreateMakesTheFileAndRefusesAnExistingOne` (the file holds the lines; a second `--create` of the same path exits 1 and leaves it; `--create` with a PLAN argument or with `--format` exits 2). Confirm RED.
2. [S2] `CompileCreate`, the flag and the case. Mutants: the `raw=true` dropped (a content `@@` line truncates the file); the existing-file refusal bypassed is not reachable here (the apply owns it), so the mutant is the empty-content case emitting no `body=0`. [proof: mutation]
3. [S3] Contract §240, the guide, AGENTS.md, README. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/ingest/ -count=1 -timeout 300s -run 'TestCompileCreateMakesACreatePlanOfStdin' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestCompileCreateMakesACreatePlanOfStdin \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestWriteCreateMakesTheFileAndRefusesAnExistingOne' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestWriteCreateMakesTheFileAndRefusesAnExistingOne \(' "$out" \
  && go test ./internal/ingest/ ./internal/guide/ -count=1 -timeout 900s \
  && grep -q '^# 240\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestCompileCreateMakesACreatePlanOfStdin` | `internal/ingest/create139_test.go` | the compiled plan carries the lines, `@@` lines, empty content and a spaced path; unsplittable content and a rooted path are refused | none | S1, S2 |
| `TestWriteCreateMakesTheFileAndRefusesAnExistingOne` | `cmd/mrw/create139_test.go` | the file is made, an existing one refused, the usage errors are exit 2 | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `CompileCreate` |
| 2 — something selects it | the flag in `writeCmd`, the `create` case of the format switch |
| 3 — the caller can discover it | `mrw write --help`; `mrw instructions`; AGENTS.md; README |
| 4 — it is used | the 2026-10-09 survey (6 sessions); no telemetry (ADR-009) |

## Invariants

- Without the flag the write action is byte-for-byte what it was.
- A create made this way passes every guard a create plan does.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a create plan cannot carry a content the survey sessions need (a file with no final newline): the record would then need the engine change it rules out.

## Out of Scope

- Byte-exact content, `mrw_write` (permanent: boundary: the record's Out of Scope)

## Mutation Log

## Verification Log
