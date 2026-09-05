# Task ADR-023-T1: The read result drops its structured envelope

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (three source files, three test files, one contract section, two wording sites)
**Owner:** unassigned
**Produces:** `mrw_read` results with no `structuredContent` and no declared `outputSchema`; the receipt at `content[1]`
**Consumes:** `result()`, `pagedResult()`, `indexResult()` and `tools()` in `internal/mcp` (ADR-011 T2, ADR-014, ADR-017)
**Data dependency:** hermetic for the fence; the sign-off run needs Claude Code on PATH (see S6)
**Proof map:** v1
**Rests-on:** `a served read carries no structuredContent`, `a page and an index carry none either`, `mrw_read declares no outputSchema while mrw_write still does`, `content[1] is still the receipt`

## Goal

Every receipt-bearing `mrw_read` result — served, paged, index, grep-no-match — reaches a host as
`content[0]` served text or report plus `content[1]` the serialized receipt and nothing else; a bare
refusal stays one text block with no receipt, as before; `tools/list` declares no `outputSchema`
for it; and a fresh Claude Code session quotes a served line back.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `readTool`, `pagedResult`, `indexResult` and the refusal paths emit no `structuredContent`; `result()` gains the bare form; comments say why |
| `internal/mcp/mcp.go` | edit | the `mrw_read` descriptor declares no `OutputSchema`; its description says the receipt is `content[1]` — this is what SELECTS the change on the wire |
| `internal/mcp/instructions.go` | edit | the instructions say where a read's receipt is |
| `internal/mcp/conformance_test.go` | edit | ADR-011's Enforced-by validates the tools that DECLARE a schema and asserts `mrw_write` still does; the new Enforced-by asserts absence on a served, a paged and an index read; `TestTheFirstContentBlockIsTheSerializedStructuredContent` reads the receipt from `content[1]` for `mrw_read` |
| `internal/mcp/tools_test.go` | edit | `structured()` callers on `mrw_read` results read `content[1]`; a `receipt()` helper does it |
| `internal/mcp/mcp_test.go` | edit | `TestMrwReadDeclaresNoOutputSchema` |
| `scripts/contract.sh` | edit | §41 keeps its `mrw_write` assertions and drops the `mrw_read` outputSchema clause; every row reading `structuredContent` off an `mrw_read` result reads `content[1]`; §61 is the new row |
| `README.md` | edit | line ~462's `structuredContent` sentence names `mrw_write` only |

## Ordered Steps

1. [S1] Write the failing tests first: `TestAReadResultCarriesNoStructuredContent` (served, paged and index results of `mrw_read` have no `structuredContent` key; `content[1]` parses to the receipt with `observed`; a page's `content[1]` carries `next_read`; an index's carries `index`) and `TestMrwReadDeclaresNoOutputSchema` (`tools()` — `mrw_read` has nil `OutputSchema`, `mrw_write` non-nil). Run: red. [proof: acceptance]
2. [S2] Drop `structuredContent` from every `mrw_read` result path: `readTool`'s served result, `pagedResult`, `indexResult`, `servedOrIndex`, and the refusals that carry a receipt. One marshal still feeds `content[1]`. [proof: mutation]
3. [S3] Drop `OutputSchema` from the `mrw_read` descriptor; keep `readSchema()` and its description table for the receipt's documentation and the description test. Say in the description and the instructions that the receipt is `content[1]`. [proof: mutation]
4. [S4] Move every test and contract consumer of `mrw_read`'s `structuredContent` to `content[1]`: the conformance tests, the `tools_test.go` callers, contract §41 (mrw_write clause kept; the `for name in (…)` loop asserts `outputSchema` PRESENT on `mrw_write` and ABSENT on `mrw_read`), §45–§51's grep/index rows, §58's paging rows. [proof: acceptance]
5. [S5] Contract §61: drives the built binary through a served read, a paged read and an index, asserts `structuredContent` absent on each and the receipt present at `content[1]`, and pairs it with the case that must fail — `mrw_write`'s result still carries `structuredContent` equal to its `content[1]`. [proof: acceptance]
6. [S6] Sign-off against the host: `claude -p --model haiku --mcp-config <json naming bin/mrw> --strict-mcp-config --allowedTools mcp__<name>` with a two-line spec, `--output-format stream-json --verbose`, and the assistant's reply quotes the served line. Recorded in the Verification Log with `claude --version`. [proof: human: the transcript's assistant text quotes `# service registry — every service has the same shape`, and the tool_result block in the same transcript begins with `==>`]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/mcp/ -run 'TestAReadResultCarriesNoStructuredContent|TestMrwReadDeclaresNoOutputSchema' -v 2>&1 | tee /tmp/adr023-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr023-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestAReadResultCarriesNoStructuredContent|TestMrwReadDeclaresNoOutputSchema)\b' /tmp/adr023-t1.out)" = "2" ] \
  && grep -q '^# 61\. ' scripts/contract.sh \
  && grep -q "write result's \`structuredContent\`" README.md \
  && grep -q 'with no \`structuredContent\`' README.md \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state cmd/mrw/main.go \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: both test names and
`# 61. `. The two README greps replaced a clause the Codex review of #110 showed to be vacuous
(`! grep -q … | grep -q …` succeeds whatever the file says); they now require the two sentences the
task adds — the write's receipt is its `structuredContent`, a read's answer comes with none. The fence is red until S1's tests exist and pass, and `./scripts/contract.sh` is red until
§61 exists and every moved consumer reads `content[1]`.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadResultCarriesNoStructuredContent` | `internal/mcp/conformance_test.go` | served, paged and index `mrw_read` results carry no `structuredContent`; `content[1]` is the receipt | — | S1, S2 |
| `TestMrwReadDeclaresNoOutputSchema` | `internal/mcp/mcp_test.go` | `mrw_read` declares no `outputSchema`; `mrw_write` still does | — | S1, S3 |

S4 is proved by the regression half of the fence — `go test ./...` and `./scripts/contract.sh` — through
`TestEveryDeclaredOutputSchemaValidatesARealResponse` (ADR-011's Enforced-by, narrowed to the tools
that declare a schema) and `TestTheFirstContentBlockIsTheSerializedStructuredContent` (`content[1]`
is the receipt for both tools, equal to `structuredContent` where one exists), both moved in this task.

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two new tests |
| 2 — something selects it | `tools()` is what `tools/list` returns and `readTool` is what `tools/call` dispatches to; restoring `StructuredContent` on the served result makes the Enforced-by and §61 red, inside the fence |
| 3 — the caller can discover it | the description and the instructions say the receipt is `content[1]`; `TestTheSurfaceSaysTheCLIIsRicher`'s sibling checks on the instructions run in the fence |
| 4 — it is used | the S6 sign-off is one observed use on one host; nothing measures uptake, and nothing will (ADR-009 refuses telemetry) |

## Mutation Log

- 2026-09-05 · 679d457* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: put the structuredContent back on every mrw_read result — the envelope the measured host renders in place of the served text · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · covers:a served read carries no structuredContent
- 2026-09-05 · 679d457* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: put the structuredContent back on a PAGE only — a served read stays bare, so a test that checked one shape would pass this · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · covers:a page and an index carry none either
- 2026-09-05 · 679d457* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S3: declare an outputSchema for mrw_read again — a schema declared is a structuredContent promised, and none is sent · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · covers:mrw_read declares no outputSchema while mrw_write still does
- 2026-09-05 · 6948737* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: put the structuredContent back on every mrw_read result — the envelope the measured host renders in place of the served text · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · covers:a served read carries no structuredContent
- 2026-09-05 · 6948737* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: put the structuredContent back on a PAGE only — a served read stays bare, so a test that checked one shape would pass this · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · covers:a page and an index carry none either
- 2026-09-05 · 6948737* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S3: declare an outputSchema for mrw_read again — a schema declared is a structuredContent promised, and none is sent · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · covers:mrw_read declares no outputSchema while mrw_write still does
- 2026-09-05 · 201869c* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: send an empty object where the receipt belongs in content[1] — the only machine-readable copy a read now has · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · covers:content[1] is still the receipt
- 2026-09-05 · 201869c* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: put the structuredContent back on every mrw_read result — the envelope the measured host renders in place of the served text · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · covers:a served read carries no structuredContent
- 2026-09-05 · 201869c* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: put the structuredContent back on a PAGE only — a served read stays bare, so a test that checked one shape would pass this · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · covers:a page and an index carry none either
- 2026-09-05 · 201869c* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S3: declare an outputSchema for mrw_read again — a schema declared is a structuredContent promised, and none is sent · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · covers:mrw_read declares no outputSchema while mrw_write still does

## Invariants

- `mrw_write`'s result carries `structuredContent` equal to its `content[1]`, and declares a schema it validates against.
- `mrw_read`'s `content[1]` is the receipt it used to put in `structuredContent`, byte for byte, from one marshal.
- The ledger records exactly what `content[0]` served — unchanged.
- The engine directories and `cmd/mrw/main.go` are byte-identical to the merge-base; `go.mod` declares one requirement.

## Risks

- A contract consumer moved to `content[1]` by pattern misses one that spelled it differently. Mitigated: §61 asserts the key ABSENT, so a row that still reads it would fail its own KeyError first, and the grep in S4 is over `structuredContent` across the whole script.
- The `-p` sign-off measures one host version. Mitigated: recorded with the version; Decision 4 says why the bare envelope is right regardless.

## Stop Condition

Stop if removing `structuredContent` from `mrw_read` requires changing the receipt's shape, or touching any engine directory. The receipt is the same object in a different block; if that stops being true the design has slipped, and ADR-011's schema tests are the ones to ask why.

## Out of Scope

- `mrw_write`'s envelope (permanent: boundary: the parent's Decision 3)
- Other hosts (deferred: docs/adr/BACKLOG.md — the "ADR-023: other hosts" entry)

## Verification Log
- 2026-09-05 · 679d457* · exit 1 · `set -o pipefail …` · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · ms:1008
  ```
  --- last 8 line(s) of stdout
  === RUN   TestAReadResultCarriesNoStructuredContent
  --- PASS: TestAReadResultCarriesNoStructuredContent (0.04s)
  === RUN   TestMrwReadDeclaresNoOutputSchema
      mcp_test.go:685: mrw_read declares an outputSchema; it returns no structuredContent, so the declaration promises a field that never arrives
  --- FAIL: TestMrwReadDeclaresNoOutputSchema (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.366s
  FAIL
  ```
- 2026-09-05 · 679d457* · exit 1 · `set -o pipefail …` · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · ms:492
  ```
  --- last 8 line(s) of stdout
  === RUN   TestAReadResultCarriesNoStructuredContent
  --- PASS: TestAReadResultCarriesNoStructuredContent (0.02s)
  === RUN   TestMrwReadDeclaresNoOutputSchema
      mcp_test.go:685: mrw_read declares an outputSchema; it returns no structuredContent, so the declaration promises a field that never arrives
  --- FAIL: TestMrwReadDeclaresNoOutputSchema (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.163s
  FAIL
  ```
- 2026-09-05 · 679d457* · exit 1 · `set -o pipefail …` · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · ms:503
  ```
  --- last 8 line(s) of stdout
  === RUN   TestAReadResultCarriesNoStructuredContent
  --- PASS: TestAReadResultCarriesNoStructuredContent (0.03s)
  === RUN   TestMrwReadDeclaresNoOutputSchema
      mcp_test.go:685: mrw_read declares an outputSchema; it returns no structuredContent, so the declaration promises a field that never arrives
  --- FAIL: TestMrwReadDeclaresNoOutputSchema (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.180s
  FAIL
  ```
- 2026-09-05 · human-observed · S6 sign-off 2026-09-05: claude -p --model haiku (Claude Code 2.1.261) with --mcp-config naming bin/mrw and --strict-mcp-config, spec tmp/curve/r12/200000-early-1/tree/services.conf:1-2 — the tool_result block began with "==> tmp/curve/r12/200000-early-1/tree/services.conf  3619L  174648B  sha 2c37d4ac" and the assistant replied "TEXT: # service registry — every service has the same shape"; the same prompt against the pre-ADR-023 server (e917310) replied "NO TEXT: {\"observed\":…,\"problems\":0}"
- 2026-09-05 · 679d457* · exit 0 · `set -o pipefail …` · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · ms:59441
- 2026-09-05 · 679d457* · exit 0 · `set -o pipefail …` · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · ms:30627
- 2026-09-05 · 679d457* · exit 0 · `set -o pipefail …` · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · ms:29415
- 2026-09-05 · 679d457* · exit 0 · `set -o pipefail …` · acceptance-sha256:14ab23ac7e8e6b7ed62bfad07512fa7228b138e6ef85a9f4b6539023e0544a9b · ms:29891
- 2026-09-05 · 6948737* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:68908
- 2026-09-05 · 6948737* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:31033
- 2026-09-05 · 6948737* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:32268
- 2026-09-05 · 6948737* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:32040
- 2026-09-05 · 201869c* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:32352
- 2026-09-05 · 201869c* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:33246
- 2026-09-05 · 201869c* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:37020
- 2026-09-05 · 201869c* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:34263
- 2026-09-05 · 201869c* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:32074
- 2026-09-05 · 201869c* · exit 1 · `set -o pipefail …` · acceptance-sha256:6a58ea72ccf195e7ed08d1d34d3fc2f0ad678ffc53ca3d193c349164a112da72 · ms:37839
  ```
  --- last 10 line(s) of stdout (of 537 after folding 537 raw)
    PASS  a window that exists but starts after an early target is refused too
    PASS  and says so, rather than serving a window with no answer in it
    PASS  the runner is its own process-group leader
    PASS  an orphan no longer under this shell is visible to the group check: pid 36380 under 1
    PASS  no process of this run survives it
    PASS  the probe is this file's prologue: one trap, the wrapper, the group kill, and no rows
    PASS  a nested run of this file's own prologue repeats the runner's exit status through the wrapper
    PASS  and its EXIT trap reaped an orphan already re-parented away (pid 36406)
    PASS  INT to the wrapper is forwarded: the nested runner ended 143 in 0s and its orphan with it
  1 assertion(s) FAILED
  --- last 1 line(s) of stderr
  Terminated: 15
  ```
