# Task ADR-012-T1: Teach the format on the wire, and show a plan that really applies

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `mcp.instructionsText`, `mcp.examplePlan`, `mcp.exampleReadSpecs`, the trigger-first tool descriptions
**Consumes:** the ADR-011 descriptor set (unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the initialize instructions`, `the worked example plan`, `the trigger sentence in each description`

## Goal

A host driving `mrw mcp` in a checkout that has no AGENTS.md learns, from the wire alone, when to
reach for mrw instead of its own editor and how to author a plan — and the example it copies is one
the binary has just proved it accepts.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go` | add | `instructionsText`, `examplePlan`, `exampleReadSpecs` — the prose and the two examples, in one place so the handshake and the descriptors quote one source |
| `internal/mcp/mcp.go` | edit | `initializeResult()` gains `instructions`; both `Description` strings lead with the trigger; `plan` and `specs` gain `examples` |
| `internal/mcp/conformance_test.go` | edit | **the ADR's Enforced-by** — parse the embedded plan and dry-run apply it against a temp tree |
| `internal/mcp/mcp_test.go` | edit | the handshake carries `instructions`; the descriptions carry the trigger |
| `scripts/contract.sh` | edit | §43 — drive the real binary and assert the wire teaches what AGENTS.md teaches |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): `initialize` returns a non-empty top-level
   `instructions`; the embedded example plan **parses and dry-run applies** against a temp tree; both
   tool descriptions carry the trigger thresholds. [proof: acceptance]
2. [S2] Write `instructionsText` for a caller who cannot see this repository: what mrw is, when it
   beats an ordinary editor, the plan grammar, one worked plan, and the two rules that produce most
   refusals — read-before-write is per LINE, and a plan is all-or-nothing so a failed hunk writes
   nothing. Do not reference AGENTS.md, README.md or any path in this checkout: the reader has none
   of them, and a pointer to a file they cannot open reads as help and is not. [proof: acceptance]
3. [S3] Return it as top-level `instructions` from `initializeResult()`. It is a field of the
   2025-06-18 lifecycle result this server already declares at `mcp.go:26`, not an extension.
   [proof: mutation]
4. [S4] Rewrite both `Description` strings trigger-first: what the tool is FOR and when it beats the
   host's built-in editor, then what it does. Over MCP the description is the whole pitch, and today
   it competes with Edit and Write while saying nothing about when it wins — or when it loses, which
   matters as much: one edit in one file costs two calls and prints more bytes than the file holds.
   [proof: mutation]
5. [S5] Add `examples` to the `plan` and `specs` input properties, quoting `examplePlan` and
   `exampleReadSpecs`. The plan example is a two-hunk plan across two files, because the single-file
   case is the one a caller would not have needed mrw for. [proof: acceptance]
6. [S6] Add contract §43: drive the built binary, assert `initialize` carries `instructions`, assert
   the AGENTS.md threshold sentence appears in the `tools/list` output of the running binary (the
   duplication the ADR accepts, asserted rather than trusted), and assert both input schemas carry a
   non-empty `examples` array. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -v 2>&1 | tee /tmp/adr012-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr012-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestEveryEmbeddedExamplePlanReallyApplies|TestTheInstructionsTellAHostHowToAuthorAPlan|TestTheDescriptionsSayWhenToReachForTheTool)\b' /tmp/adr012-t1.out)" = "3" ] \
  && grep -q '^# 43\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the three test names,
`# 43.` in `contract.sh`, and the Go identifiers `instructionsText`, `examplePlan`,
`exampleReadSpecs`. The lowercase word `instructions` was **NOT** usable as a clause — it appears
throughout the ADR corpus and in the plugin prose — and neither was `description`, which appears in
both existing input schemas. That is the same near-miss `outputSchema` produced in ADR-011-T2, and
the reason each token is counted rather than chosen by eye. `# 43.` was confirmed free: the highest
section in `contract.sh` is 42, from ADR-011-T3.

`TestEveryEmbeddedExamplePlanReallyApplies` is the ADR's `Enforced-by`. It is deliberately not a
string comparison against a golden plan: it feeds the example through `plan.Parse` and a real dry-run
apply, because an example asserted to be *present* stays green long after it stops being *valid*.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryEmbeddedExamplePlanReallyApplies` | `internal/mcp/conformance_test.go` | **the ADR's Enforced-by** — the documented plan is one the engine accepts today | — | S1, S5 |
| `TestTheInstructionsTellAHostHowToAuthorAPlan` | `internal/mcp/mcp_test.go` | S2, S3 — the handshake carries instructions, and they contain a plan that parses | — | S1, S2, S3 |
| `TestTheDescriptionsSayWhenToReachForTheTool` | `internal/mcp/mcp_test.go` | S4 — the trigger, not just the behaviour | — | S1, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests above |
| 2 — something selects it | `initializeResult()` answers every `initialize`; `tools()` is what `tools/list` returns — dropping either string turns the tests and §43 red |
| 3 — the caller can discover it | this task IS the discovery surface: `initialize` and `tools/list` are the only things an MCP-only caller can read |
| 4 — it is used | nothing here measures whether a host surfaces `instructions` or whether a caller's plans improve, and nothing will — counting that needs telemetry, which ADR-009 refused on the premise this tool does not phone home. The parent ADR records why the `refused_parse` tally cannot stand in for it |

## Mutation Log

- 2026-09-04 · 99ee7a0 · mutant killed · exit 1 · `internal/mcp/mcp.go` · S3: drop `instructions` from the initialize result — the one field the lifecycle spec provides for telling a host how to drive this server · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7
- 2026-09-04 · 99ee7a0 · mutant killed · exit 1 · `internal/mcp/mcp.go` · S4: take the trigger threshold out of `mrw_read`'s description, leaving it saying only what the tool does — which is what it said before this ADR · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7
- 2026-09-04 · 99ee7a0 · mutant killed · exit 1 · `internal/mcp/instructions.go` · S5: give the published example a regexp address — the exact authoring mistake its own description warns about, and the one a stale example would teach a caller to make · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7
- 2026-09-04 · 99ee7a0 · mutant survived · exit 0 · `internal/mcp/instructions.go` · S5: change the example's `anchor=` text. It survives by construction — `treeFor` plants the anchor FROM the plan, so a guard in an example naming files no repository has can never fail. Recorded rather than repaired, because the fixture cannot know what a fictional file holds, and written into the test's own comment so the next reader does not mistake a green run for a checked guard · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7

## Invariants

- `initialize` returns a non-empty top-level `instructions` string.
- Every plan embedded in a description, in `examples`, or in `instructions` parses and dry-run
  applies against a real tree.
- Both tool descriptions state when to reach for the tool, using AGENTS.md's thresholds.
- Nothing ADR-011 declared is dropped: `title`, `annotations`, `outputSchema` and `_meta` stay, and
  its conformance tests stay green unmodified.
- `go.mod` declares exactly one requirement, and the six engine directories are untouched.

## Risks

- The instructions grow into a second AGENTS.md and cost more than they teach. Mitigated by writing
  them for a reader who has nothing else, and by §43 bounding their length.
- An example that is valid today rots silently. Mitigated by executing it rather than asserting it.

## Stop Condition

Stop if teaching the format on the wire requires changing the format. ADR-001 owns the grammar and
this task documents it; if the worked example cannot be written without an amendment to the plan
syntax, that is a separate record, not a paragraph in a description string.

## Out of Scope

- Output-schema property descriptions (T2)
- MCP `prompts` or `resources` as a place to publish the guide (permanent: boundary: restated from ADR-010 and ADR-011)
- Any change to the plan grammar (permanent: boundary: ADR-001 owns it)

## Verification Log
- 2026-09-04 · 99ee7a0* · exit 1 · `set -o pipefail …` · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7
  ```
  --- last 8 line(s) of stdout
  internal/mcp/conformance_test.go:275:42: undefined: examplePlan
  internal/mcp/mcp_test.go:278:28: undefined: examplePlan
  internal/mcp/mcp_test.go:293:16: undefined: maxInstructionsChars
  internal/mcp/mcp_test.go:294:75: undefined: maxInstructionsChars
  internal/mcp/mcp_test.go:304:40: undefined: triggerRule
  internal/mcp/mcp_test.go:305:98: undefined: triggerRule
  internal/mcp/mcp_test.go:311:23: undefined: instructionsText
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  ```
- 2026-09-04 · 99ee7a0* · exit 1 · `set -o pipefail …` · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7

  §43 was red here for two reasons worth keeping. AGENTS.md wraps the threshold sentence across two
  lines, so `grep -qF` could not see it; the row now folds whitespace before comparing. And
  `grep -q 'fail'` matched the string "0 failed" in a PASSING receipt — the near-miss shape this
  corpus keeps producing, green on every run; the row now anchors the verdict to the first column.
- 2026-09-04 · 99ee7a0 · exit 0 · `set -o pipefail …` · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7
- 2026-09-04 · 99ee7a0* · exit 0 · `set -o pipefail …` · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7 · ms:14182
- 2026-09-04 · 96c5098* · exit 0 · `set -o pipefail …` · acceptance-sha256:e8e34fb105c0b7ad65ef22c4329a99f67278f545ddf63348b983b1d7209101e7 · ms:17671
