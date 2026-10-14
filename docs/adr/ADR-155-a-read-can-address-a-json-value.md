# ADR-155: A read can address a JSON value by its path

**Status:** Accepted
**Accepted:** 2026-10-14 by Zy — approved the plan `glimmering-frolicking-harp` ("ADR-155 — JSON path address, read only") and set the goal "implement this plan end to end".
**Date:** 2026-10-14
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-005, ADR-026, ADR-036, ADR-063, ADR-104, ADR-137, ADR-146, ADR-153
**Governs:** `internal/read/read.go`, `internal/read/jsonpath.go`, `internal/plan/plan.go`, `internal/guide/guide.go`, `AGENTS.md`, `README.md`
**Enforced-by:** `internal/read/jsonpath155_test.go::TestAJSONPathAddressesTheLinesOfItsValue`
**Invalidates:** none — checked: a bare `$`, `$-N` and `N-$` keep meaning the last line on both paths (ADR-063); every other address form parses as before
**Served-path change:** `mrw read file.json:$.a.b` (and `$.a[0]`, `$["key"]`) serves and licenses the lines the value at that path spans; a path that is not there, or a file that is not one JSON document, is a range that could not be served (exit 1); a plan address in that form is refused by name

## Context

The 14-session survey (2026-10-09, `docs/adr/BACKLOG.md`, "Look mode, what is left") counted six sessions reading JSON: a `package.json` script, a `tsconfig` option, one key of a large fixture. Each read the file whole or guessed a `/pattern/` and then counted braces. A pattern finds a line, not a value: `/"name"/` matches every object's `name`, and the value's end is wherever its closing bracket happens to be. The caller wants the value's lines, by the path they already know.

ADR-153 and ADR-154 took the other look-mode items. This record takes the JSON path, for reading only: a plan addresses lines, and a write follow-up is deferred.

**Audit of the class** — *a parser that turns an address string into lines*: `mrw read --grep 'func (ParseSpec|parseRange|ParseAddr|splitRanges)\(' internal/` names four: `read.ParseSpec`, `read.splitRanges` and `read.parseRange` on the read path, `plan.ParseAddr` on the write path. The read path takes the form; `ParseAddr` refuses it by name. `read.Walk` and `read.AstGrep` build their own specs and never meet one.

## Existing Primitives Audit

- `read.resolve` turns ranges into spans against the split lines; a JSON path becomes one more case there, resolved against the file's bytes, so serving, `--max-lines`, the ledger and the `!! no match` report are the ones every range already uses.
- `encoding/json`'s `Decoder` gives each token's end offset (`InputOffset`); a value's lines are counted from the file's own bytes with the terminator `internal/lines` chose (ADR-065).
- `readCapped` already bounds what a read loads (ADR-104, 1 GiB).

## Decision

1. **The form.** A read range that begins `$.` or `$[` is a JSON path: `$` followed by one or more steps — `.name` (up to the next `.` or `[`), `[N]` (an array index, from 0) or `["name"]` (a JSON string, for a name holding `.`, `[`, `]` or a space). A bare `$`, `$-N` and `N-$` keep meaning the last line. Refused at parse, exit 2, naming what is missing: an empty name, `..`, `*`, a single-quoted name, an index that is not a decimal, text after the last step, and a relative end (`$.a,+3`; the path names its own span).
2. **What it serves.** The lines the value at that path spans, from its first byte to its last, whole lines — the same span and the same licence a line range naming those lines would give (ADR-005). A value on one line serves that one line, with whatever else is on it. `--max-lines` caps it like any span; `-C` does not widen it (it widens patterns only).
3. **What fails, as a missed range** (`!! no match for <path> (<why>)`, exit 1, ADR-146's `incomplete:`): a file that is not exactly one JSON document — JSON Lines, trailing text and a syntax error included, the line of the error named; a name not in its object; an index past its array's end; a step into a value of the wrong kind; and **a name that appears twice in one object along the path**, with both lines, since a path cannot say which one it meant and either answer would be a guess. Other duplicates in the file are not looked at.
4. **A leading UTF-8 byte order mark** is skipped before decoding; it is on line 1 either way.
5. **Names compare exactly**, after JSON unescaping (`"ab"` is `ab`), and case-sensitively.
6. **A plan address in this form is refused by name**: "a JSON path is a read address: read it, then address the lines it served" (`plan.ParseAddr`). The write half is deferred.
7. **The docs teach it**: AGENTS.md's address list, README's read section and `mrw instructions`.

It fails if a path serves a line outside its value, licenses one, misses a line of it, or resolves a duplicated name to either copy.

## Alternatives Considered

- **Full JSONPath (filters, wildcards, recursive descent).** Rejected: several spans for one address and a grammar larger than the problem; the six sessions each named one value.
- **Serve the value's bytes, re-indented.** Rejected: mrw serves numbered lines so a read is the licence for the write; text that is not the file's lines licenses nothing.
- **Take the last duplicate, as `encoding/json` unmarshals.** Rejected: the caller would be served one copy while a reader of the file sees two; refusing names both lines.
- **Accept it in plans now.** Rejected for this record: a write address must resolve to one site on the ORIGINAL file and a value's span shifts with every hunk above it; read, then address the lines.

## Component / Boundary Impact

`internal/read` (an engine package; this record owns the change) gains `jsonpath.go`; `internal/plan` gains one refusal. No other engine package changes.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `mrw read` / `mrw_read` spec address | `$.a.b`, `$.a[0]`, `$["k"]` | T1 | the caller |
| plan address | the form refused by name | T1 | the caller |
| AGENTS.md, README.md, `mrw instructions` | the form taught | T1 | agents, people |

## Inter-task Contracts

None — one task.

## Implementation

See `tasks/`.

## Consequences

- **Positive:** one call serves one JSON value and licenses exactly its lines.
- **Negative:** a JSON path read decodes the file once per step; on a file near the read cap that is seconds.
- **Neutral:** `mrw_read` takes the form through the same spec parser; nothing in the MCP surface changes.

## Out of Scope

- A JSON path as a plan address (deferred: docs/adr/BACKLOG.md "From ADR-155")
- Names containing `,`, `:` or `/` in any form (permanent: boundary: a spec splits ranges at `,` and its path at the last `:`; citation-free, it is the spec grammar of ADR-026)
- Wildcards, filters and recursive descent (permanent: boundary: one address, one span)
- YAML, TOML and JSON with comments (permanent: boundary: one format with a decoder in the standard library; go.mod keeps one requirement)

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| An offset maps to the wrong line on a CRLF or CR-only file | Low | High | the test serves the same value from LF, CRLF and CR-only copies and requires the same lines |
| A path licenses a line outside its value | Low | High | the CLI test writes to the line after the span and requires the refusal |

## Rollback

Revert T1. Specs in the new form become `bad line number` again (exit 2).

## Follow-ups

None.
