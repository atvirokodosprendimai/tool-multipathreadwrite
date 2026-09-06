# Task ADR-027-T3: The engine refuses it too, and one fence segment was vacuous

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** M (the engine boundary, four adapters, and a fence that was asserting nothing)
**Owner:** Zy
**Produces:** `apply.Input.CountedBody` and the engine-boundary refusal
**Consumes:** `plan.Hunk.CountedBody` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the engine refusing a create carrying no body`

## Goal

Refuse the body-less `create` where every caller passes, not only where the parser does, and repair
a fence segment that ran no tests.

## Context this task exists for

The Codex review of PR #126 found the parser-only rule had a hole:

    Apply(root, []Input{
        {Path: "empty.txt",   Op: "create", Lines: -1},
        {Path: "sibling.txt", Op: "create", Body: []string{"x"}, Lines: -1},
    })
    → failed=0, empty.txt exists, sibling.txt exists

`plan.validate` protects the CLI, the MCP server and the curve scorer because each calls
`plan.Parse`. `Apply` is a public entry point whose own doc comment says it validates every hunk, and
this package's tests build `Input`s directly throughout. So a direct caller created the empty file —
**and its valid sibling was written too, so ADR-001's all-or-nothing was bypassed as well.**

This is the same shape as ADR-026's engine-boundary finding, one record later. That one was about
`RelEnd`; the lesson generalised and I did not carry it, which is why the fix here is deliberately
the same shape: the field travels to `apply.Input`, and the boundary refuses what the parser refuses.

⚠ **And a segment of T1's own Acceptance fence was asserting nothing.** It read
`go test ./internal/plan/ -run 'TestAReplaceWithNoBodyIsRejected|TestBodyZeroMeansAnEmptyBody'`.
`TestBodyZeroMeansAnEmptyBody` does not exist anywhere, and `TestAReplaceWithNoBodyIsRejected` lives
in `internal/adversarial`. The segment matched zero tests in that package and exited 0 — the failure
`testing.md` names first, in the gate rather than in the code. Found while acting on the review's
LOW about fence scope, which is what pointed at it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | `Input` and the internal `hunk` carry `CountedBody`, and the boundary check that already refuses a relative end on an op that cannot honour one now also refuses a `create` carrying no body. |
| `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/curve/score.go`, `internal/mcp/tools_test.go` | edit | The four `plan.Hunk → apply.Input` conversions. ADR-026 established there are four; this record does not get to rediscover that. |
| `internal/apply/apply_test.go` | edit | `TestTheEngineRefusesABodyLessCreate` drives `Apply` directly with a refused create AND a valid sibling, asserting NEITHER file exists. |
| `internal/plan/plan.go`, `internal/plan/plan_test.go` | edit | Two comments: the create block had been inserted into the middle of an existing sentence, and the test-row comment still said "create takes an empty body" after its own row stopped saying that. |
| `docs/adr/ADR-027-…md` | edit | Two claims corrected: the record said the refusal message was parameterised rather than duplicated, which is false — they are two strings, each naming the remedy for its own op — and three line citations no longer pointed at what they named. |
| `T1` | edit | The vacuous fence segment replaced with one that runs three real tests and greps for two of them by name. |

## Ordered Steps

1. [S1] Write `TestTheEngineRefusesABodyLessCreate` — refused create plus valid sibling, neither file present, and `body=0` still applying at the engine — and confirm it is RED. [proof: mutation]
2. [S2] Carry `CountedBody` onto `apply.Input` and the internal hunk, and through all four conversions. [proof: mutation]
3. [S3] Refuse the body-less `create` in the same boundary block that refuses a relative end on an op that cannot honour one. [proof: mutation]
4. [S4] Replace T1's vacuous fence segment, and re-run T1's mutants against the new digest — a mutant bound to a fence that has changed is evidence about a fence nobody runs. [proof: acceptance]
5. [S5] Repair the two comments and the record's two claims. [proof: human: read the create branch top to bottom and confirm the PR #74 sentence about pattern addresses is whole again, and that the record no longer claims a shared message]
6. [S6] Run every gate including `go test -race ./...`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/apply/ -count=1 -v -run 'TestTheEngineRefusesABodyLessCreate' 2>&1 | tee /tmp/adr027-t3.out \
  && grep -q '^--- PASS: TestTheEngineRefusesABodyLessCreate' /tmp/adr027-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr027-t3.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr027-t3c.out \
  && grep -q '^contract holds$' /tmp/adr027-t3c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheEngineRefusesABodyLessCreate` | `internal/apply/apply_test.go` | A direct `Apply` refuses the body-less create, writes NEITHER it nor its valid sibling, names `body=0`, and still applies `CountedBody: true` | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestTheEngineRefusesABodyLessCreate` |
| 2 — something selects it | The boundary block runs for every hunk `Apply` receives, whatever built it; the S3 mutation removes the create clause and the fence goes red on a file that should not exist |
| 3 — the caller can discover it | Unchanged: the refusal wording is T1's and the documentation is T2's |
| 4 — it is used | §65 covers the parser path in CI; the engine path is covered by this test, because no CLI or MCP call can reach `Apply` without the parser |

## Mutation Log

- 2026-09-06 · a5e4347* · mutant killed · exit 1 · `internal/apply/apply.go` · the engine stops refusing a body-less create, so a direct Apply caller writes the empty file — and its valid sibling with it, because nothing failed · acceptance-sha256:5ac88fac56f17e342ae1cf5503079eb166471f2b0474ac99499736e3743ea9ad · covers:the engine refusing a create carrying no body
- 2026-09-06 · a5e4347* · mutant killed · exit 1 · `internal/apply/apply.go` · the declaration is dropped between Input and the internal hunk, so a caller who said body=0 is refused at the engine and the change becomes a ban rather than a narrowing · acceptance-sha256:5ac88fac56f17e342ae1cf5503079eb166471f2b0474ac99499736e3743ea9ad · covers:the engine refusing a create carrying no body
- 2026-09-06 · a5e4347* · mutant survived · exit 0 · `internal/apply/apply.go` · the engine reports the refusal and then carries on with the hunk, so the empty file is written anyway and the sibling with it — a verdict that says failed while the tree changed is worse than no verdict · acceptance-sha256:5ac88fac56f17e342ae1cf5503079eb166471f2b0474ac99499736e3743ea9ad · covers:the refusal writing nothing at all
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```

## Invariants
- ⚠ **"The refusal writes nothing at all" was declared in Rests-on and withdrawn.** The test asserts the sibling's absence and that assertion stays, but the property belongs to ADR-001's all-or-nothing rather than to anything this task added: a mutant removing the `continue` after `fail` SURVIVED, because the hunk is already marked failed and the run stages nothing regardless. Recorded rather than deleted — "I could not make the fence fail for that reason" is the finding, and a mutant that broke ADR-001 would have proved ADR-001, not this.

- `body=0` still creates an empty file at the engine as well as through the parser — the control is in the same test.
- ADR-001 holds: a refused hunk means nothing is written, siblings included, and the test asserts the sibling's absence rather than only the refusal.
- The relative-end boundary check from ADR-026 is untouched; this adds a clause beside it rather than reshaping it.
- `internal/read`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- Two places now refuse the same thing, and they could drift. Judged the lesser risk: the parser's message is the one a caller sees, the engine's exists for callers that never reach the parser, and ADR-026 already carries the same pair for `RelEnd`. A single check is not possible without making `Apply` depend on `internal/plan`.
- Re-running T1's mutants against a changed fence costs a cycle. Necessary: a mutant bound to a digest nobody runs is evidence about nothing.

## Stop Condition

Stop and ask if refusing at the engine requires `internal/apply` to import `internal/plan` — that
inverts the dependency the two packages are split to avoid, and would be a different decision.

## Out of Scope

- A general "the engine re-validates everything the parser does" pass (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:5ac88fac56f17e342ae1cf5503079eb166471f2b0474ac99499736e3743ea9ad · ms:31159
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:5ac88fac56f17e342ae1cf5503079eb166471f2b0474ac99499736e3743ea9ad · ms:30202
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:5ac88fac56f17e342ae1cf5503079eb166471f2b0474ac99499736e3743ea9ad · ms:31020
- 2026-09-06 · a5e4347* · exit 0 · `set -o pipefail …` · acceptance-sha256:5ac88fac56f17e342ae1cf5503079eb166471f2b0474ac99499736e3743ea9ad · ms:31300
