# Task ADR-001-T1: The plan document parses into hunks, and reports every syntax error at once

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `plan.Parse` (T1), `plan.Hunk` (T1), `plan.ParseAddr` (T1)
**Consumes:** none
**Data dependency:** hermetic

## Goal

`internal/plan` turns a `@@`-headed document into `[]plan.Hunk`, rejecting a
malformed plan with every error it found rather than the first.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/plan/plan.go` | add | The parser, the `Hunk`/`Addr` types, and address grammar |
| `internal/plan/plan_test.go` | add | The failing tests, then the passing ones |
| `cmd/mrw/main.go` | edit | `writeCmd` is what SELECTS the parser — without this call site the package is unreachable |

## Ordered Steps

1. Confirm the failing tests for this task's behaviour exist and can go red.
   **Retrofit note:** the TDD-red run happened historically in `676296e` and
   cannot be re-performed. The substitute proof this corpus accepts is a
   mutation: `adr-verify <this file> --mutant internal/plan/plan.go --from ... --to ...`
   must record `killed`. That is a stronger claim than a red test at authoring
   time, because it proves the test binds to the mechanism.
2. Parse the `@@ <path> <addr> <op> [k=v...]` header, honouring double quotes so
   a path or an `anchor=` may contain spaces.
3. Parse addresses: `N`, `N-M`, `N-` (to EOF), `$`, `0`, `-`.
4. Collect body lines to the next header, or exactly `body=N` lines when given.
5. Validate op/address/body agreement (`delete` takes no body, `create` takes no
   address, an insert needs a single line and a non-empty body).
6. Accumulate errors and return them together.

## Acceptance

```bash
set -o pipefail
go test ./internal/plan/ -run 'TestParse|TestQuoted|TestExplicitBody' -v 2>&1 | tee /tmp/adr001t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr001t1.out \
  && go test ./internal/plan/
```

The new unit runs alone first, then the whole package, chained with `&&` so
neither can carry the verdict by itself.

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestParseAddr` | `internal/plan/plan_test.go` | Every address form, and that bad ones error | — |
| `TestParseMultipleFilesAndOps` | `internal/plan/plan_test.go` | Four hunks, four ops, two files, from one document | — |
| `TestExplicitBodyLengthProtectsHeaderLikeLines` | `internal/plan/plan_test.go` | `body=N` lets a body contain a line starting with `@@ ` | — |
| `TestParseRejectsBadPlans` | `internal/plan/plan_test.go` | Eleven malformed plans each error | — |
| `TestParseReportsEveryError` | `internal/plan/plan_test.go` | Two independent errors are both named in one return | — |
| `TestQuotedFieldsSurvive` | `internal/plan/plan_test.go` | A quoted path and a quoted anchor keep their spaces | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestParseMultipleFilesAndOps` |
| 2 — something selects it | `cmd/mrw/main.go` `writeCmd` calls `plan.Parse`; deleting that call makes `scripts/contract.sh` row 1 fail |
| 3 — the caller can discover it | `mrw write --help` documents the header grammar; `scripts/contract.sh` exercises the documented form |
| 4 — it is used | Every `mrw write` in this repository's own development; nothing counts them |

## Mutation Log

## Invariants

- An unknown op is an error, never a silently ignored hunk.
- Parsing never touches the filesystem — `internal/plan` imports no `os`.
- A plan with zero `@@` headers is an error, not an empty success.

## Risks

- A body line beginning with `@@ ` is misread as a header when `body=` is
  omitted. Mitigated by `body=N`, and by the parser treating an unparseable
  header as an error rather than as body.

## Stop Condition

Stop if the format needs to express something the `@@` line cannot carry
(nested structure, binary content, per-hunk encoding) — that is a format change
and belongs in a new ADR, not in this task.

## Out of Scope

- Applying the hunks — that is T2 and T3's job.
- Resolving `@N` pointers into the working set — that is the CLI's job, and
  belongs to the working-set decision, not to the format.

## Verification Log
