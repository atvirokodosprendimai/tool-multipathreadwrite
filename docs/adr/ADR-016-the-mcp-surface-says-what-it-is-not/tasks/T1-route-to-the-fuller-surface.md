# Task ADR-016-T1: Route to the fuller surface, and keep the claim true

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (prose, one test, one contract row)
**Owner:** unassigned
**Produces:** the routing text and its test
**Consumes:** —
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the routing text reaches the wire`, `the flags it names really exist`

## Goal

A caller with both a shell and this server is told, before it picks, that the CLI is the fuller
tool and what it is giving up — and the flags that advice names cannot silently disappear.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go` | edit | the handshake opens with which surface to be on |
| `internal/mcp/mcp.go` | edit | both descriptions carry the same routing in one clause |
| `internal/mcp/mcp_test.go` | edit | **the ADR's Enforced-by** — the advice is on the wire AND its named flags exist |
| `scripts/contract.sh` | edit | §50 — the shipped binary says it, and its own help agrees |

## Ordered Steps

1. [S1] Write the failing test first (TDD red): the shipped `instructions` name the CLI, `--grep`
   and `--check`, and say this server serves one fixed checkout; both tool descriptions carry the
   routing; and every flag the text names appears in the CLI's own help output. [proof: acceptance]
2. [S2] Put the routing FIRST in `instructions`. A caller that has already picked a tool has stopped
   reading, and the point of this paragraph is to be read before the choice. [proof: mutation]
3. [S3] Name the capabilities concretely — `--grep`, `--files-from`, `--check`, `--json`, and `-C`
   for reach. "The CLI is richer" routes nobody; naming what is lost is what makes the choice real.
   [proof: mutation]
4. [S4] Add the same routing to both descriptions, because a host that ignores `instructions` reads
   only those — ADR-012's own finding, applied to this record. [proof: mutation]
5. [S5] Add contract §50: the built binary says it on the wire, and `mrw read --help` /
   `mrw write --help` really list the flags the wire names. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr016-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr016-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestTheSurfaceSaysTheCLIIsRicher)\b' /tmp/adr016-t1.out)" = "1" ] \
  && grep -q '^# 50\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name, `# 50.`
and the Go identifier `cliIsRicher`. **§50 rather than §49**: 48 is the highest on `main`, and
ADR-015 holds 49 on #77, so this takes the next free one rather than colliding on merge — the same
choice that made #74 and #76 merge textually instead of by renumbering.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSurfaceSaysTheCLIIsRicher` | `internal/mcp/mcp_test.go` | **the ADR's Enforced-by** — the routing reaches both surfaces, and every flag it names exists in the CLI's help | — | S1, S2, S3, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `initializeResult()` and `tools()` are what every host reads |
| 3 — the caller can discover it | this task IS the discovery surface, and it is first in the document on purpose |
| 4 — it is used | nothing measures whether an agent switched surfaces, and nothing will — telemetry is refused by ADR-009, and ADR-012's Context records why the local tally cannot attribute it. The evidence that prompted this record is M's direct observation, which is the honest provenance |

## Mutation Log

- 2026-09-04 · 1bbcdeb · mutant killed · exit 1 · `internal/mcp/instructions.go` · review fix: recommend `-C` for choosing a checkout again — after `read` that is the integer context flag, so `mrw read -C DIR` errors with "invalid value … for flag -C". Advice that FAILS WHEN FOLLOWED is worse than no advice, and this shipped in the first cut · acceptance-sha256:c06636f0674ed460d73244d34e767a85efa3586c4c8231e9175e47e50687826a
- 2026-09-04 · 1bbcdeb · mutant killed · exit 1 · `internal/mcp/instructions.go` · review fix: drop the serialization counterweight — "the CLI is strictly fuller" is FALSE, because one server is one writer to the ledger while parallel CLI processes race (ADR-010:42,185), and a caller told only half of that has been misrouted · acceptance-sha256:c06636f0674ed460d73244d34e767a85efa3586c4c8231e9175e47e50687826a
- 2026-09-04 · d90be24 · mutant killed · exit 1 · `internal/mcp/instructions.go` · S3: rename a flag the advice recommends to one the CLI does not have — advice that recommends a flag which has since been renamed is worse than no advice, and this is the rot the help-output check exists to catch · acceptance-sha256:c06636f0674ed460d73244d34e767a85efa3586c4c8231e9175e47e50687826a
- 2026-09-04 · d90be24 · mutant killed · exit 1 · `internal/mcp/instructions.go` · S2: move the routing out of first position — a caller that has already picked a tool has stopped reading, so advice further down is advice nobody acts on · acceptance-sha256:c06636f0674ed460d73244d34e767a85efa3586c4c8231e9175e47e50687826a

## Invariants

- Every flag the routing text names appears in the CLI's own help output.
- The routing appears in `instructions` AND in both tool descriptions.
- `instructions` stays under `maxInstructionsChars`.
- No tool, schema or behaviour changes.

## Risks

- The advice names a flag that is later renamed, leaving the wire recommending something gone.
  Mitigated by asserting each named flag against `--help`, which is generated from the flag set.

## Stop Condition

Stop if the paragraph will not fit under `maxInstructionsChars`. Shorten something already there and
say which; do not raise the bound, which every session pays.

## Out of Scope

- Adding capability to the MCP surface (permanent: boundary: ADR-010 owns the two-tool decision)
- Any CLI change (permanent: boundary: the CLI does not learn that MCP exists)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · d90be24* · exit 0 · `set -o pipefail …` · acceptance-sha256:c06636f0674ed460d73244d34e767a85efa3586c4c8231e9175e47e50687826a · ms:12051
- 2026-09-04 · 1bbcdeb* · exit 0 · `set -o pipefail …` · acceptance-sha256:c06636f0674ed460d73244d34e767a85efa3586c4c8231e9175e47e50687826a · ms:41932
