# Task ADR-051-T1: Compile apply_patch to plan hunks; an unread sibling writes nothing

**Depends-on:** none
**Covers:** F-1, F-2, F-3, F-4, F-5, F-6, F-7, F-8, F-9, F-10, F-11, F-12, F-13, F-14, F-15, F-16, F-17, F-18, UC1-S2, UC2-S1, UC2-S2
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `ingest.CompileApplyPatch` (T1)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the unread two-hunk apply_patch writing nothing`, `compile going through plan.Parse`, `ambiguous old side refusing`

## Goal

`ingest.CompileApplyPatch` turns a Codex `apply_patch` document into native plan text. A two-hunk update whose second old side was never served compiles, then Apply writes nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/applypatch.go` | add | `CompileApplyPatch(root, doc) ([]byte, error)` — the compiler. |
| `internal/ingest/applypatch_test.go` | add | The unread two-hunk test, plus compile→Parse and ambiguous-old-side. |

## Ordered Steps

1. [S1] Write `TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing` and confirm it is RED: `CompileApplyPatch` does not exist. [proof: mutation]
2. [S2] Write `TestCompileApplyPatchGoesThroughParse` and confirm it is RED. [proof: mutation]
3. [S3] Write `TestAnAmbiguousOldSideIsACompileRefusal` and confirm it is RED. [proof: mutation]
4. [S4] Implement `CompileApplyPatch`. Emit plan text only. Locate the old side by exact unique match. Multi-line replace carries `anchor=` from the first old-side line. Confirm S1–S3 GREEN. [proof: mutation]
5. [S5] Run the scoped package tests and `gofmt` / `go vet` unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/ingest/ -count=1 -v \
  -run 'TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing|TestCompileApplyPatchGoesThroughParse|TestAnAmbiguousOldSideIsACompileRefusal' 2>&1 | tee /tmp/adr051-t1.out \
  && grep -q '^--- PASS: TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing' /tmp/adr051-t1.out \
  && grep -q '^--- PASS: TestCompileApplyPatchGoesThroughParse' /tmp/adr051-t1.out \
  && grep -q '^--- PASS: TestAnAmbiguousOldSideIsACompileRefusal' /tmp/adr051-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr051-t1.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./internal/ingest/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestATwoHunkApplyPatchWithOneUnreadLineWritesNothing` | `internal/ingest/applypatch_test.go` | Two update hunks, one unread line: exit path is Apply fail+skip, file unchanged | — | S1, S4 |
| `TestCompileApplyPatchGoesThroughParse` | `internal/ingest/applypatch_test.go` | Compiled bytes are accepted by `plan.Parse` | — | S2, S4 |
| `TestAnAmbiguousOldSideIsACompileRefusal` | `internal/ingest/applypatch_test.go` | An old side matching twice refuses at compile; nothing is written | — | S3, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests |
| 2 — something selects it | T2's `writeCmd` calls `CompileApplyPatch`; deleting the function fails T2 |
| 3 — the caller can discover it | T2's `--format` flag and `write --help` |
| 4 — it is used | nothing measures this yet — ADR-009 refused telemetry |

## Mutation Log
- 2026-09-12 · 9f5c2bf* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · first-match locate: an old side that hits three times compiles instead of refusing · acceptance-sha256:bf2a6fbcb787bbfb4a1056036503bee5edd926afb0a4855be0434136ec55c1b3

## Invariants

- Compile emits plan text. It does not call `apply.Apply` and it does not construct `apply.Input`.
- An unread compiled line is Apply's refusal, not a compile success that writes.
- `maxInstructionsChars` stays 4096.
- ADR-019 pick A stands.

## Risks

- S1 passes on a compile error (no function) that never reaches Apply. Mitigation: S1 applies through `plan.Parse` + `apply.Apply` after a successful compile; a missing function fails the compile call first, which is the TDD red, and the green path must show `failed` + `skip`.
- Locating at the first match. Mitigation: S3.

## Stop Condition

Stop if the proposed fix is applying hunks sequentially, constructing `apply.Input` without Parse, fuzzy-matching a near-miss old side, or parsing target syntax.

## Out of Scope

- The CLI flag (that's T2)
- Contract §82 (that's T2)
- Aider SEARCH/REPLACE
- `*** Delete File:` / `*** Move to:`
- MCP `format`

## Verification Log
- 2026-09-12 · 9f5c2bf* · exit 0 · `set -o pipefail …` · acceptance-sha256:bf2a6fbcb787bbfb4a1056036503bee5edd926afb0a4855be0434136ec55c1b3 · ms:911
