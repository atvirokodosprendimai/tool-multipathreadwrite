# Task ADR-017-T1: Find over MCP, and degrade to the index rather than to a dead end

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (one package, one new result shape)
**Owner:** unassigned
**Produces:** grep over MCP, its index degradation and its refusals
**Consumes:** `read.Walk` (ADR-007, unchanged), `mcp.MaxResultChars` (ADR-011-T3, unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the walk's specs are served exactly as caller specs are`, `an oversized grep still answers`, `the index is resumable`, `a served grep records what it served`

## Goal

A caller with only the MCP surface can find its sites across a tree, and a grep too large to serve
comes back as a usable list of addresses rather than as a failure.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `readTool` accepts `grep`/`exclude`, calls `read.Walk`, and owns the index degradation |
| `internal/mcp/mcp.go` | edit | `mrw_read`'s input schema declares the two new arguments, described — ADR-012's rule |
| `internal/mcp/schema.go` | edit | the read schema declares `matches`, `index` and `next_index`, described. ⚠ An earlier revision of this row said the file was NOT touched, on the reasoning that `matchIndex` builds its own map. That reasoning is exactly how a response comes to violate the schema its own tool advertises: a schema-validating host would have rejected the index. Corrected after review of #80 |
| `internal/mcp/tools_test.go` | edit | **the ADR's Enforced-by**, plus the refusal and ledger properties |
| `scripts/contract.sh` | edit | §51 — drive a real oversized grep through the built binary |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a grep serves matching ranges and records them; an
   oversized grep returns an index of `path:/pattern/` specs with a count and NO content; an index
   to serve names where to resume; `grep` with a ranged spec is refused. [proof: acceptance]
2. [S2] Call `read.Walk` and then the existing serve path. The CLI does exactly this
   (`main.go:510`), so the two surfaces run the same code in the same order rather than two
   implementations that agree today. [proof: mutation]
3. [S3] Degrade to the index on overflow, not to a refusal. The cap already reports that it fired;
   the new part is that the answer is the SPEC LIST, which is both smaller and directly usable as the
   next call's `specs`. [proof: mutation]
4. [S4] Page the index by file when the index itself will not fit. This is the step most likely to
   be skipped, because the content case is the satisfying one — and skipping it puts ADR-014's dead
   end back for the exact caller this record is for. [proof: mutation]
5. [S5] Record ONLY what was served. The index path serves no content, so it must record nothing: an
   index that licensed writes to lines the caller never saw would be the read-before-write bypass
   wearing a third costume. [proof: mutation]
6. [S6] Mirror the CLI's refusal of `grep` with a ranged spec, with its reasoning — "two answers to
   one question" (`main.go:499`). A grammar the two surfaces disagree on is what ADR-016 exists to
   prevent. [proof: acceptance]
7. [S7] Add contract §51: drive the built binary over a tree larger than the cap, assert an index
   comes back, that following it reads a real file, and that a ranged spec plus `grep` is refused.
   [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr017-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr017-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd|TestGrepServesWhatItFindsAndRecordsIt|TestGrepRefusesARangedSpec|TestAnIndexTooLargeToServePagesByFile)\b' /tmp/adr017-t1.out)" = "4" ] \
  && grep -q '^# 51\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the four test names,
`# 51.`, and the Go identifiers `matchIndex` and `grepSpecs`. **§51 rather than §49 or §50**: 50 is
the highest on `main` after #78 merged, and 51/52 were confirmed free by sorting the section numbers
rather than reading the last one in file order — they are not in order in `contract.sh`, and taking
the tail would have said 49.

`TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd` is the ADR's `Enforced-by`, and it is deliberately
not a presence check on an index field: it drives a REAL overflow and asserts the result is usable —
an index whose entries parse as specs, with a count, and no content. A test that asked only whether
an `index` key exists would pass over an index of the wrong files.

**The engine go/no-go clauses stay in this fence and must pass.** This record calls `read.Walk`; it
does not change it. If `internal/read` moves on this branch, that is the signal that the design
slipped from "call the primitive" into "change the primitive", which is the Stop Condition below.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd` | `internal/mcp/tools_test.go` | **the ADR's Enforced-by** — an overflowing grep answers with usable addresses | — | S1, S3 |
| `TestAnIndexTooLargeToServePagesByFile` | `internal/mcp/tools_test.go` | S4 — the index itself is resumable, not a third dead end | — | S1, S4 |
| `TestGrepServesWhatItFindsAndRecordsIt` | `internal/mcp/tools_test.go` | S2, S5 — a fitting grep serves content and records exactly it; the index path records nothing | — | S1, S2, S5 |
| `TestGrepRefusesARangedSpec` | `internal/mcp/tools_test.go` | S6 — the grammar matches the CLI's | — | S1, S6 |
| `TestTheIndexSurvivesAPatternThatLooksLikeARange` | `internal/mcp/tools_test.go` | the entry form is unambiguous for a pattern containing `/`, which `path:/pattern/` was not | — | S3 |
| `TestAWalkProblemIsReportedAndNotSwallowed` | `internal/mcp/tools_test.go` | a path the walk could not use is named, not silently dropped | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | `readTool` is the only path an MCP read takes; §51 drives it through the real binary |
| 3 — the caller can discover it | partly — the arguments are in the declared `inputSchema` with descriptions and examples; T2 teaches them in prose |
| 4 — it is used | not measured and will not be: telemetry is refused by ADR-009, and the per-checkout tally cannot attribute a surface (ADR-012's Context). The evidence for building it is M's stated population, which is the honest provenance |

## Mutation Log

<!-- filled during execution -->
- 2026-09-04 · ba05252* · mutant killed · exit 1 · `internal/mcp/tools.go` · S4: give the index an unbounded budget so it never pages — the step after the satisfying part, the one this record predicted would be dropped. It also found a DEAD ASSERTION: the mutant sailed past len(idx) >= 9000 because the fixture had been resized to 8000, so that check could never fire; the threshold is bound to the fixture count now · acceptance-sha256:6240852dc249e1c084a95d1ecd3379962f63338ad67d77618f849b06930b08b7
- 2026-09-04 · 4ae94cc* · mutant killed · exit 1 · `internal/mcp/tools.go` · S3: put the walk address back into each index entry. TestTheIndexSurvivesAPatternThatLooksLikeARange fails: for the pattern alpha/,/beta the entry f.txt:/alpha/,/beta/ parses back as a pattern RANGE, so it reads different lines than it claims to index — a wrong answer that only appears for patterns containing a slash, which is why it survived the suite, a contract row and a first review. Found by review of #80 · acceptance-sha256:6240852dc249e1c084a95d1ecd3379962f63338ad67d77618f849b06930b08b7

## Invariants

- A grep serves the same ranges the CLI's `--grep` serves for the same pattern and root.
- An oversized grep returns addresses, never a bare refusal and never truncated content.
- An oversized INDEX names where to resume.
- The index path records nothing in the ledger, because it served nothing.
- `grep` with a ranged spec is refused, with the CLI's reasoning.
- No engine directory changes; `go.mod` declares exactly one requirement.

## Risks

- The index is built but never exercised, so the overflow path dead-ends in practice. Mitigated by
  the Enforced-by driving a real overflow rather than constructing a result.
- The index licenses writes. Mitigated by asserting the ledger is untouched on that path.
- `read.Walk` starts to need changes, which would mean this record has grown into the engine. That is
  the Stop Condition, and the fence's engine clauses report it.

## Stop Condition

Stop if serving a grep requires `internal/read` to learn anything new. `Walk` already finds and `Run`
already serves; if either must change, the design has slipped and the record should say so rather
than the branch quietly widening.

## Out of Scope

- Teaching any of this in prose (T2 — ADR-012's finding, applied)
- `--files-from` (permanent: boundary: parent ADR, Decision 4)
- Any root or multi-root change (deferred: docs/adr/BACKLOG.md — the Desktop-coverage entry, until the record that owns reach exists)
- Bounding the walk (deferred: parent ADR, Alternatives)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · ba05252* · exit 0 · `set -o pipefail …` · acceptance-sha256:6240852dc249e1c084a95d1ecd3379962f63338ad67d77618f849b06930b08b7 · ms:37140
- 2026-09-04 · ba05252* · exit 0 · `set -o pipefail …` · acceptance-sha256:6240852dc249e1c084a95d1ecd3379962f63338ad67d77618f849b06930b08b7 · ms:19228
- 2026-09-04 · 4ae94cc* · exit 0 · `set -o pipefail …` · acceptance-sha256:6240852dc249e1c084a95d1ecd3379962f63338ad67d77618f849b06930b08b7 · ms:149481
