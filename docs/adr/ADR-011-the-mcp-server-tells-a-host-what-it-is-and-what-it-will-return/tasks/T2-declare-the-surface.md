# Task ADR-011-T2: Declare what each tool is, and what it returns

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** the tool descriptor set (`title`, `annotations`, `outputSchema`, `_meta`), `mcp.MaxResultChars`
**Consumes:** `mcp.ResolveRoot` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the generated output schemas`, `the truthfulness of the annotations`, `the serialized-JSON content block`

## Goal

A host reading `tools/list` learns which tool writes, what shape comes back and how big it may get;
and every declared schema is one a real response actually satisfies.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/schema.go` | add | derive a JSON Schema from a Go type by reflection, so a schema cannot drift from the struct it describes |
| `internal/mcp/schema_test.go` | add | its tests |
| `internal/mcp/mcp.go` | edit | the tool descriptors gain `title`, `annotations`, `outputSchema`, `_meta` |
| `internal/mcp/tools.go` | edit | `content[0]` becomes the serialized JSON; the report moves to `content[1]` |
| `internal/mcp/conformance_test.go` | add | **the ADR's Enforced-by** — call each tool for real and validate the response against its own declared schema |
| `scripts/contract.sh` | edit | §41 — drive the real binary and assert the declared surface and the two content blocks |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a schema generated from `apply.Result` names its
   fields and their types; a schema with no properties is REJECTED by the generator rather than
   returned; each tool's declared `outputSchema` validates the `structuredContent` of a real call;
   `content[0]` parses as JSON and equals `structuredContent`. [proof: acceptance]
2. [S2] Generate the schemas from the types the handlers return. **Not hand-written beside them.** A
   peer measured a hand-written schema attached to the WRONG tool and another that declared "object,
   no properties" and therefore validated anything; both read fine in review. The generator refuses
   to emit a property-less object schema for a struct, so the permissive form cannot be produced by
   accident. [proof: mutation]
3. [S3] Declare `title` and `annotations` on both tools, truthfully: `mrw_read` is `readOnlyHint:
   true`, `mrw_write` is `readOnlyHint: false` with `destructiveHint: true` and
   `idempotentHint: false`. A host shows these to a user before asking them to approve a call, so an
   annotation that flatters the tool is a lie the user acts on. [proof: mutation]
4. [S4] Declare `_meta["anthropic/maxResultSizeChars"]` from the same constant T3 enforces, so the
   advertised limit and the enforced limit cannot drift. One constant, two readers.
   [proof: acceptance]
5. [S5] Put the serialized JSON in `content[0]` and move the report to `content[1]`, satisfying
   *"a tool that returns structured content SHOULD also return the serialized JSON in a TextContent
   block"*. It is the SAME serialization as `structuredContent`, marshalled once and used twice —
   two marshals of one value is how the two halves start to disagree. [proof: mutation]
6. [S6] Add contract §41: drive the built binary, assert `tools/list` carries `title`, `annotations`,
   `outputSchema` and `_meta` for both tools, assert `mrw_read` really is read-only by checking the
   tree is unchanged after a call, and assert both content blocks are present with `content[0]`
   parsing as JSON equal to `structuredContent`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr011-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr011-t2.out \
  && [ "$(grep -cE '^--- PASS: (TestEveryDeclaredOutputSchemaValidatesARealResponse|TestAPropertylessObjectSchemaIsRefused|TestTheSchemaNamesTheFieldsOfTheResult|TestTheFirstContentBlockIsTheSerializedStructuredContent|TestTheAnnotationsMatchWhatTheToolDoes)\b' /tmp/adr011-t2.out)" = "5" ] \
  && grep -q '^# 41\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was grepped for BEFORE this fence was written and returned **zero hits**: the five test
names, `# 41.` in `contract.sh`, and `OutputSchema`/`Annotations` across `internal/` and `cmd/`. Note
that the lowercase `outputSchema` was NOT usable as a clause — it already appears once, in a
`tools.go` comment saying we do not declare one, so a fence greping for it would have been green from
the day it was written. That is the fourth near-miss of this kind in this repository and the reason
each token is counted rather than assumed.

`TestEveryDeclaredOutputSchemaValidatesARealResponse` is the ADR's `Enforced-by`. It is deliberately
not a schema-shape test: it CALLS each tool and validates what came back, because a schema checked
against the code that wrote it is a mirror.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryDeclaredOutputSchemaValidatesARealResponse` | `internal/mcp/conformance_test.go` | **the ADR's Enforced-by** — the declared shape is the shipped shape | — | S1, S2 |
| `TestAPropertylessObjectSchemaIsRefused` | `internal/mcp/schema_test.go` | S2 — the permissive schema that validates anything cannot be emitted | — | S1, S2 |
| `TestTheSchemaNamesTheFieldsOfTheResult` | `internal/mcp/schema_test.go` | S2 — generation follows the type, including its json tags | — | S1, S2 |
| `TestTheFirstContentBlockIsTheSerializedStructuredContent` | `internal/mcp/conformance_test.go` | S5 — the spec's SHOULD, and that both halves are one marshal | — | S1, S5 |
| `TestTheAnnotationsMatchWhatTheToolDoes` | `internal/mcp/conformance_test.go` | S3 — a readOnly tool that writes is a lie a user acts on | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests above |
| 2 — something selects it | `tools()` is what `tools/list` returns; dropping a descriptor field makes the conformance test and §41 red, both inside the fence |
| 3 — the caller can discover it | this task IS the discovery surface — `tools/list` is how a host learns anything about these tools |
| 4 — it is used | nothing here measures host uptake, and nothing will: counting which fields a host read needs telemetry, which ADR-009 refused on the premise this tool does not phone home |

## Mutation Log

- 2026-09-03 · a4897b6 · mutant killed · exit 1 · `internal/mcp/schema.go` · S2: allow the permissive schema back — a property-less object validates anything and tells a caller nothing, which is the peer-measured failure this generator exists to make unrepresentable · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055
- 2026-09-03 · 711cf00 · mutant killed · exit 1 · `internal/mcp/mcp.go` · S3: understate mrw_read as not read-only — an annotation a host shows a user before they approve a call must match what the tool does · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055
- 2026-09-03 · c6f31e8 · mutant killed · exit 1 · `internal/mcp/tools.go` · S5: put the report where the serialized JSON belongs, so a host reading content[0] gets prose — the spec SHOULD this task exists to satisfy · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055

## Invariants

- Every tool that declares an `outputSchema` returns `structuredContent` that validates against it.
- `content[0]` is byte-identical to the marshalled `structuredContent` of the same response.
- No annotation claims a tool is read-only unless a real call leaves the tree unchanged.
- `capabilities.tools` stays `{}` — nothing sends `notifications/tools/list_changed`, so nothing
  advertises `listChanged`.
- `go.mod` declares exactly one requirement, and the six engine directories are untouched.

## Risks

- A reflection-based generator produces a schema that is technically valid and useless. Mitigated by
  refusing property-less object schemas and by validating a REAL response rather than the schema.
- Annotations drift from behaviour as tools change. Mitigated by §41 asserting read-only-ness by
  observation — running the tool and checking the tree — rather than by reading the descriptor.

## Stop Condition

Stop if generating a schema requires a JSON Schema library. ADR-010 and ADR-004 both refuse a
dependency that can be avoided, and the subset needed here is object/string/boolean/integer/array
over structs this repository owns. If the types outgrow that, say so and hand-write the schemas with
the conformance test as the guard — but do not add the dependency quietly.

## Out of Scope

- `_meta["anthropic/requiresUserInteraction"]` (permanent: boundary: stated in the parent ADR — a prompt per write makes batching unusable)
- Declaring `listChanged` (permanent: boundary: the tool set is fixed at compile time; the capability would be a promise with no sender)
- Schemas for anything other than the two tool results (permanent: boundary: the tools ARE the product)

## Verification Log
- 2026-09-03 · 7f38b05* · exit 1 · `set -o pipefail …` · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055 · ms:161
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  internal/mcp/conformance_test.go:120:9: tl.Annotations undefined (type tool has no field or method Annotations)
  internal/mcp/conformance_test.go:123:9: tl.Title undefined (type tool has no field or method Title)
  internal/mcp/conformance_test.go:143:33: byName["mrw_write"].Annotations undefined (type tool has no field or method Annotations)
  internal/mcp/conformance_test.go:150:14: tl.Annotations undefined (type tool has no field or method Annotations)
  internal/mcp/schema_test.go:12:12: undefined: SchemaOf
  internal/mcp/schema_test.go:46:15: undefined: SchemaOf
  internal/mcp/schema_test.go:56:15: undefined: SchemaOf
  internal/mcp/schema_test.go:56:15: too many errors
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-09-03 · 4483490 · exit 0 · `set -o pipefail …` · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055 · ms:9868
- 2026-09-03 · a4897b6 · exit 0 · `set -o pipefail …` · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055 · ms:9153
- 2026-09-03 · 711cf00 · exit 0 · `set -o pipefail …` · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055 · ms:9050
- 2026-09-03 · c6f31e8 · exit 0 · `set -o pipefail …` · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055 · ms:9225
- 2026-09-03 · a34a5d1 · exit 0 · `set -o pipefail …` · acceptance-sha256:d8a387f152fd5f68916a77e1436298653048500fc611b3eb941f5a4bae78c055 · ms:11629
