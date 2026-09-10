# Task ADR-019-T3: Teaching matches the pick

**Depends-on:** T1, T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** Teaching matches the pick
**Consumes:** Naming pick encoded in this Decision as `Naming pick:` (T1), Binary behaves as the pick (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the handshake sentence is true of the pick`, `maxInstructionsChars is still 4096`, `the surface names how this pick names the tree`

## Goal

The wire, the README MCP section, and `TestTheSurfaceSaysTheCLIIsRicher` say the same thing the
binary does after T2 — including whether this surface is still one fixed checkout.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/instructions.go` | edit | only if B or C makes *"ONE fixed checkout"* false; pick A leaves the sentence |
| `internal/mcp/mcp_test.go` | edit | `TestTheSurfaceSaysTheCLIIsRicher` must stay true; add `TestTheSurfaceNamesTheRootThePickChose` if B or C |
| `internal/mcp/schema.go` | edit | only if B or C adds a call-level root *choice among the allow-list*; never an unconstrained `root` |
| `README.md` | edit | "Use it from an MCP host" already documents `--root` for Desktop; retarget if the pick is B or C |
| `AGENTS.md` | edit | only if a Shared() sentence this file copies becomes false; do not generate the file from `guide.Shared()` |

## Ordered Steps

1. [S1] Write `TestTheSurfaceNamesTheRootThePickChose` so it is RED until handshake / README
   name how this pick names the tree. Under pick A, that is still ONE fixed checkout plus the
   existing `--root` Desktop block; no sentence claims a second root. [proof: acceptance]
2. [S2] If pick B or C: retarget *"ONE fixed checkout"* to the allow-list rule. Keep
   `maxInstructionsChars == 4096`; shorten MCP-only prose rather than raise the bound. Rewrite
   `TestTheSurfaceSaysTheCLIIsRicher` in the same commit so it cannot protect the old sentence.
   [proof: mutation]
3. [S3] Confirm a caller reading only `tools/list` / handshake can see how to name the tree.
   Pick A satisfies this with the existing README block plus the existing handshake. [proof: acceptance]
4. [S4] `gofmt` / `go vet` on `./internal/mcp`. Confirm `maxInstructionsChars` is still the literal
   4096. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -qE '^Naming pick: (A — launch --root only|B — launch allow-list|C — MCP roots/list)$' \
  docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestTheSurfaceSaysTheCLIIsRicher|TestTheSurfaceNamesTheRootThePickChose' 2>&1 \
  | tee /tmp/adr019-t3.out \
  && grep -q '^--- PASS: TestTheSurfaceSaysTheCLIIsRicher' /tmp/adr019-t3.out \
  && grep -q '^--- PASS: TestTheSurfaceNamesTheRootThePickChose' /tmp/adr019-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr019-t3.out \
  && grep -n 'maxInstructionsChars = 4096' internal/mcp/instructions.go \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

`TestTheSurfaceNamesTheRootThePickChose` was grepped BEFORE this fence was written and returned
**zero hits**. `TestTheSurfaceSaysTheCLIIsRicher` already exists; the fence requires the new test
too, so a pick-A no-op that leaves the old test passing cannot carry T3.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSurfaceSaysTheCLIIsRicher` | `internal/mcp/mcp_test.go` | Routing remains true of the pick; *"ONE fixed checkout"* holds under A and is gone under B/C | — | S2 |
| `TestTheSurfaceNamesTheRootThePickChose` | `internal/mcp/mcp_test.go` | Handshake or tool description names how to name the tree this pick uses | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two tests |
| 2 — something selects it | `instructionsText()` is what `initialize` returns; deleting a retarget and leaving the old sentence fails S2 under B/C |
| 3 — the caller can discover it | this task *is* rung 3 |
| 4 — it is used | nothing measures this yet |

## Mutation Log

- 2026-09-10 · c444fc5* · mutant killed · exit 1 · `internal/mcp/instructions.go` · without launch --root DIR mcp the new test is the only one that fails; the old richer test stays green · acceptance-sha256:2652a08b9d13bd8db0292ca23067946dbe2d46f126fd90a58f414d0a498f918c · covers:the surface names how this pick names the tree

## Invariants

- `maxInstructionsChars` is the literal 4096.
- Do not generate `AGENTS.md` from `guide.Shared()`.
- Do not add cargo tools to the handshake as if they were reach.
- `AckRule` is unchanged.

## Risks

- Handshake overflow. Mitigation: go/no-go; Stop Condition if the fix is raising 4096.
- Rewording *"ONE fixed checkout"* so the old test still greps a token. The test must assert the
  pick's rule, not a substring that survives.

## Stop Condition

Stop and ask if the only way to teach the pick is to raise `maxInstructionsChars`. Stop if T1 has
no `Naming pick:` line or T2's §78 is missing.

## Out of Scope

- The naming implementation — that is T2's job
- Encoding the pick — that is T1's job
- The handshake body-less `create` clause (deferred: docs/adr/BACKLOG.md — From ADR-027; 71 bytes, one spare)

## Verification Log
- 2026-09-10 · c444fc5* · exit 0 · `set -o pipefail …` · acceptance-sha256:2652a08b9d13bd8db0292ca23067946dbe2d46f126fd90a58f414d0a498f918c · ms:853
