# Task ADR-212-T1: the schema says a moved link has no content

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the two descriptions in `internal/mcp/schema.go`; the regenerated legacy golden and its line in `era_test.go`'s list of regenerations
**Consumes:** `mrw_write`'s output schema; `TestALegacyResultIsUnchangedByTheModernPath`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the schema states the moved-link case for both keys`, `the legacy golden changed for those strings only`, `the behaviour is still #725's`, `every named test passed and none skipped`, `the engine packages are untouched`

## Goal

ADR-212 Decision 3. `TestTheWriteSchemaSaysAMovedLinkCarriesNoContent` (`internal/mcp`) reads `mrw_write`'s output schema from `tools/list` and asserts that `files.sha_after`'s description holds "Absent for a removed file and for a moved symbolic link, whose target mrw does not read." and `files.lines_after`'s holds "0 for a removed file and for a moved symbolic link, whose target mrw does not read: there it says nothing about the lines the link leads to." `TestALegacyResultIsUnchangedByTheModernPath` passes against the regenerated golden, whose diff from `main` is those two strings and nothing else; its comment gains "and once by ADR-212 … those two descriptions, nothing else". `TestAMovedSymlinkCarriesNoContentAndLicensesNothing` passes unchanged.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/schema.go` | edit | the two descriptions |
| `internal/mcp/schema212_test.go` | new | the Goal's assertion |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerate (`MRW_UPDATE_LEGACY_GOLDEN=1`) | the two strings |
| `internal/mcp/era_test.go` | edit | its list of regenerations names ADR-212 |

## Ordered Steps

1. [S1] `TestTheWriteSchemaSaysAMovedLinkCarriesNoContent`, RED. [proof: acceptance]
2. [S2] The descriptions; regenerate the golden; the comment. Mutants, each killed: either sentence dropped; the golden left stale (`TestALegacyResultIsUnchangedByTheModernPath`). Then `adr-lint` on every record whose test lock names `era_test.go`, relocking what it reports. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && go test -count=1 -v ./internal/mcp -run '^(TestTheWriteSchemaSaysAMovedLinkCarriesNoContent|TestALegacyResultIsUnchangedByTheModernPath)$' > "$d/mcp" \
  && go test -count=1 -v ./internal/writer -run '^TestAMovedSymlinkCarriesNoContentAndLicensesNothing$' > "$d/writer" \
  && grep -q '^--- PASS: TestTheWriteSchemaSaysAMovedLinkCarriesNoContent (' "$d/mcp" \
  && grep -q '^--- PASS: TestALegacyResultIsUnchangedByTheModernPath (' "$d/mcp" \
  && grep -q '^--- PASS: TestAMovedSymlinkCarriesNoContentAndLicensesNothing (' "$d/writer" \
  && ! grep -q -- '--- SKIP' "$d/mcp" "$d/writer" \
  && grep -qF 'a moved symbolic link, whose target mrw does not read' internal/mcp/schema.go \
  && grep -qF 'once by ADR-212' internal/mcp/era_test.go \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/seen internal/check internal/state)" \
  && [ -z "$st" ]
```

The `go test` runs pass and name each test; the two text clauses are red today; the engine clauses hold every engine package to no committed non-test change since the merge base and no uncommitted or untracked file (ignored files excepted), and a failing `git status` fails the chain. `TestAMovedSymlinkCarriesNoContentAndLicensesNothing` skips where symlinks cannot be made (Windows without the privilege); the fence runs on unix.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWriteSchemaSaysAMovedLinkCarriesNoContent` | `internal/mcp/schema212_test.go` | the two descriptions, read from `tools/list` | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every `tools/list` an MCP host sends |
| 3 — the caller can discover it | the output schema's descriptions; README's rename paragraph |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #712 |

## Mutation Log

## Verification Log

## Invariants

- No receipt key is added, removed or renamed; the behaviour is #725's.

## Risks

- Editing `era_test.go` can move other records' test locks; S2 lints and relocks them.

## Stop Condition

The fence exits 0, every gate in `AGENTS.md` passes, `adr-debt` reports nothing broken or unreceipted, and the Codex challenge's findings are answered in the record.

## Out of Scope

- `docs/receipts.txt` (permanent: boundary: the record's Out of Scope; it lists keys, not meanings)
