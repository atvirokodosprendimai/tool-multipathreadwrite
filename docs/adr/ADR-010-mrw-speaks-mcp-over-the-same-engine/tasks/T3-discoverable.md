# Task ADR-010-T3: Make it installable, and say what it does not change

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** unassigned
**Produces:** the MCP section of `README.md` and the `AGENTS.md` note
**Consumes:** MCP tools `mrw_read` / `mrw_write` (T2), `mcp.Serve` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the host config block`, `the statement that the CLI path is unchanged`

## Goal

Someone can add mrw to an MCP host by copying one config block, and a reader learns which limitation
the server lifts and which it does not.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | the config block, what the two tools are, and the correction to the parallel-read paragraph |
| `AGENTS.md` | edit | one line: an agent that speaks MCP does not need the shell recipe |
| `scripts/contract.sh` | edit | §39 — the documented config block names the subcommand the binary actually has |

## Ordered Steps

1. [S1] Write the failing check first (TDD red): the fence asserts `README.md` carries an MCP host
   config block naming `mrw` and `mcp`. It fails today, because the section does not exist.
   [proof: acceptance]
2. [S2] Write the config block a host actually takes — command, args, and nothing else. A block a
   reader must adapt is a block that gets adapted wrongly. [proof: acceptance]
3. [S3] **Correct the parallel-read paragraph rather than leaving it.** It currently says "Run mrw
   one call at a time against a checkout" with no qualifier; after T2 that is true of the CLI path
   and false of the MCP path, and a limitation stated more broadly than it holds is the kind of
   thing readers route around unnecessarily. [proof: acceptance]
4. [S4] Say what the server does NOT change: same engine, same ledger, same exit-status meanings,
   same plan format. A second transport reads like a second product, and a caller deciding between
   them should know there is nothing to decide. [proof: human: a reader must be told what is unchanged, and no test asserts the absence of a misunderstanding]
5. [S5] Add the `AGENTS.md` line. That file is the authored source for agent guidance, and an agent
   reading it over MCP should not be told to shell out. [proof: acceptance]
6. [S6] Add contract §39 asserting the documented block names a subcommand `mrw --help` lists — a
   config example that names a command the binary does not have is the documentation equivalent of
   a dangling pointer, and this repository shipped one of those on 2026-09-03. The fence's
   `grep -q '^# 39\.'` proves only that the section exists; that it ASSERTS anything is proved by
   this task's `adr-verify --mutant` entry — rename the subcommand and §39 must go red.
   [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '### Use it from an MCP host' README.md \
  && grep -q '"command": "mrw"' README.md \
  && grep -q 'through the server' README.md \
  && grep -q 'CLI path' README.md \
  && grep -q 'mrw mcp' AGENTS.md \
  && grep -q '^# 39\.' scripts/contract.sh \
  && ./scripts/contract.sh \
  && go test ./...
```

Every clause names something only this task writes, and each was grepped for BEFORE this fence was
written and returned **zero hits** — `### Use it from an MCP host`, `"command": "mrw"`,
`through the server` and `CLI path` in `README.md`, `mrw mcp` in `AGENTS.md`, `# 39.` in
`contract.sh`. That sentence was here before the greps were, and two of the clauses it covered were
already matching: `grep -q '"mrw"'` hits `README.md:133` and `grep -q 'one call at a time'` hits
`README.md:727`, both written by unrelated work. The fence stayed red overall, so nothing failed
loudly — which is the point. A vacuous clause inside a red fence is invisible until the day the
other clauses go green, and then it is a clause that was never going to fail. Both are replaced
with strings this task must write: the config block's `"command": "mrw"`, and the qualifier
`through the server` that S3 has to add to the parallel-read paragraph.

`grep -q 'CLI path'` and `grep -q 'through the server'` are the two that matter: together they fail
unless S3 actually qualified the paragraph, rather than leaving a limitation stated more broadly
than it holds.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | no unit test: the deliverable is documentation, proved by the fence's greps and by §39 driving the binary | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the README and AGENTS.md sections, asserted by the fence |
| 2 — something selects it | n/a: prose has no call site. §39 is what fails when the block names a subcommand the binary lacks |
| 3 — the caller can discover it | it IS the discovery surface — a host config block is how anyone installs an MCP server |
| 4 — it is used | nothing measures installs, and nothing will: counting who added a config block would need telemetry, which ADR-009 refused on the same premise |

## Mutation Log

- 2026-09-03 · fae3f88 · mutant killed · exit 1 · `cmd/mrw/main.go` · rename the subcommand so the README config block names a command the binary no longer has — the dangling-pointer defect §39 exists to catch · acceptance-sha256:b9fbe8fde0edaccdd01f873c93d3f98264097c19fdb10dccaf1bf578a5e7bd17

## Invariants

- The README states that the MCP path serializes calls made through the server, and that a CLI
  invocation running beside it is still subject to the CLI limitation.
- No claim is made about the server that the CLI does not also satisfy.
- `measure.sh` and the existing measurement sections are untouched.

## Risks

- A config block goes stale when a host changes its schema. Mitigated by §39 checking the part this
  repository owns — the subcommand name — and by keeping the block minimal.
- "Use the MCP server instead" is an easy line to write and would be wrong. S4 exists to say the
  opposite: they are the same engine and there is nothing to choose between.

## Stop Condition

Stop if documenting the server requires explaining a behaviour difference from the CLI. There should
be nothing to explain; if there is, T2's Stop Condition was crossed and the fix belongs there rather
than in a paragraph warning readers about it.

## Out of Scope

- Publishing to any MCP registry or directory (deferred: docs/adr/BACKLOG.md)
- Measuring adoption (permanent: boundary: it needs telemetry, which ADR-009 refused on the premise that this tool acquires no dependencies it can avoid and does not phone home)

## Verification Log
- 2026-09-03 · f46617a · exit 0 · `set -o pipefail …` · acceptance-sha256:b9fbe8fde0edaccdd01f873c93d3f98264097c19fdb10dccaf1bf578a5e7bd17 · ms:10066
- 2026-09-03 · fae3f88 · exit 0 · `set -o pipefail …` · acceptance-sha256:b9fbe8fde0edaccdd01f873c93d3f98264097c19fdb10dccaf1bf578a5e7bd17 · ms:9078
