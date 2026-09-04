# Task ADR-012-T2: Say what every field of the answer means, and keep saying it

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one package)
**Owner:** unassigned
**Produces:** `mcp.describeResult`, the two description tables
**Consumes:** `SchemaOf` (ADR-011-T2, unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the description table`, `total coverage in both directions`

## Goal

Every property of every declared `outputSchema` says what it means, and a field added to
`apply.Result` tomorrow cannot ship undescribed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/schema.go` | edit | `describeResult` walks a generated schema and attaches descriptions from a table; the two tables live beside the schema builders |
| `internal/mcp/schema_test.go` | edit | the table is applied; an unknown key is refused |
| `internal/mcp/conformance_test.go` | edit | total coverage over the REAL declared schemas, both directions |
| `scripts/contract.sh` | edit | §44 — the shipped binary's `tools/list` describes every property it declares |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): every property of every declared `outputSchema`,
   at every depth, has a non-empty `description`; a table entry naming a property the schema does
   not declare is a failure, not a no-op. [proof: acceptance]
2. [S2] Add the descriptions as a table in `internal/mcp`, **not as struct tags on `apply.Result`
   or `seen.Observation`.** Those are two of the six engine directories every ADR-010 and ADR-011
   fence asserts byte-identical, and that boundary is permanent; a `desc:` tag is inert at runtime
   and still changes those bytes. ADR-011's boundary table already gives `internal/mcp` ownership of
   the tool descriptors, and prose about a field is a descriptor. [proof: mutation]
3. [S3] Apply the table to the generated schema rather than hand-writing schemas that carry prose.
   Generation is what keeps the SHAPE derived from the type — ADR-011 measured a peer's hand-written
   schema attached to the wrong tool. Only the prose is authored. [proof: mutation]
4. [S4] Enforce coverage in BOTH directions. An undescribed property is the drift a new field
   causes; a described property that no longer exists is the drift a removed field causes, and it is
   the quieter of the two because the schema still validates. [proof: mutation]
5. [S5] Add contract §44: drive the built binary and assert that every property of every declared
   `outputSchema` in the real `tools/list` response carries a non-empty description — read off the
   wire, not off the source. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr012-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr012-t2.out \
  && [ "$(grep -cE '^--- PASS: (TestEveryOutputSchemaPropertyIsDescribed|TestADescribedPropertyThatNoLongerExistsIsRefused)\b' /tmp/adr012-t2.out)" = "2" ] \
  && grep -q '^# 44\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the two test names,
`# 44.` in `contract.sh`, and the Go identifier `describeResult`. The word `description` was **NOT**
usable as a clause — it already appears in both input schemas and in `readSchema()`, so a fence
greping for it would have been green the day it was written. `# 44.` was confirmed free: the highest
section in `contract.sh` is 42, and T1 takes 43.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryOutputSchemaPropertyIsDescribed` | `internal/mcp/conformance_test.go` | S1, S4 — total coverage over the real declared schemas, at every depth | — | S1, S3, S4 |
| `TestADescribedPropertyThatNoLongerExistsIsRefused` | `internal/mcp/schema_test.go` | S4 — the quieter direction: a stale entry describes a shape nobody sends | — | S1, S4 |
| `TestTheStatusDescriptionNamesTheValuesTheEngineSends` | `internal/mcp/conformance_test.go` | added in review — the prose names the values `apply.Status` really sends, not values a host would filter on and never match | — | S2, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests above |
| 2 — something selects it | `readSchema()` and `writeSchema()` are what `tools/list` publishes; dropping a description turns the conformance test and §44 red |
| 3 — the caller can discover it | the descriptions ride in `outputSchema`, which is the only machine-readable account of the answer a host gets |
| 4 — it is used | nothing measures whether a host reads a property description, and nothing will — see the parent ADR's Context for why the available instrument cannot see that population |

## Mutation Log

- 2026-09-04 · 99ee7a0 · mutant killed · exit 1 · `internal/mcp/schema.go` · S2: drop `hunks.status` from the table — the single most load-bearing field of the receipt, and the one a caller must read to tell an ok from a skip · acceptance-sha256:e346ac09339dfc93a0c595edcfffedf121843cd7ecb9fffe9d56ddff54ed2852
- 2026-09-04 · 96c5098 · mutant killed · exit 1 · `internal/mcp/schema.go` · review fix: restore the shipped `fail`/`skip` wording on hunks.status — the defect two independent reviewers found, where a host filtering on the documented value reads every failing run as clean · acceptance-sha256:e346ac09339dfc93a0c595edcfffedf121843cd7ecb9fffe9d56ddff54ed2852
- 2026-09-04 · 99ee7a0 · mutant killed · exit 1 · `internal/mcp/schema.go` · S4: make a stale table entry a no-op instead of a refusal — the quiet direction of the drift, where the schema still validates every response while the prose describes a field nobody sends · acceptance-sha256:e346ac09339dfc93a0c595edcfffedf121843cd7ecb9fffe9d56ddff54ed2852

## Invariants

- Every property of every declared `outputSchema`, at every depth, has a non-empty `description`.
- Every entry in a description table names a property the schema actually declares.
- Shapes stay generated: no output schema is hand-written, only its prose is.
- `internal/apply` and `internal/seen` carry no annotation added for this task.
- `go.mod` declares exactly one requirement, and the six engine directories are untouched.

## Risks

- The coverage test passes vacuously if it walks nothing. Mitigated by asserting the walk found the
  properties the real schemas declare, not merely that it found no undescribed one.
- Descriptions become restatements of the field name. Not mechanically preventable; the table sits
  beside the schema builders so a reviewer sees prose and shape together.

## Stop Condition

Stop if attaching prose requires abandoning generation and hand-writing the schemas. That is the
exact failure ADR-011-T2 was built to make unrepresentable, and it is not worth a description string.

## Out of Scope

- Input-schema examples and the handshake instructions (T1)
- Struct tags on the engine types (permanent: boundary: the six directories stay byte-identical)
- A JSON Schema library (permanent: boundary: restated from ADR-010 and ADR-011-T2)

## Verification Log
- 2026-09-04 · 99ee7a0* · exit 1 · `set -o pipefail …` · acceptance-sha256:e346ac09339dfc93a0c595edcfffedf121843cd7ecb9fffe9d56ddff54ed2852
  ```
  --- last 4 line(s) of stdout
  internal/mcp/schema_test.go:139:15: undefined: describeResult
  internal/mcp/schema_test.go:142:11: undefined: describeResult
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-09-04 · 99ee7a0 · exit 0 · `set -o pipefail …` · acceptance-sha256:e346ac09339dfc93a0c595edcfffedf121843cd7ecb9fffe9d56ddff54ed2852
- 2026-09-04 · 99ee7a0* · exit 0 · `set -o pipefail …` · acceptance-sha256:e346ac09339dfc93a0c595edcfffedf121843cd7ecb9fffe9d56ddff54ed2852 · ms:32235
- 2026-09-04 · 96c5098* · exit 0 · `set -o pipefail …` · acceptance-sha256:e346ac09339dfc93a0c595edcfffedf121843cd7ecb9fffe9d56ddff54ed2852 · ms:12326
