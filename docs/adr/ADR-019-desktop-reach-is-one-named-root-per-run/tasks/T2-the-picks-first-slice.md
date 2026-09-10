# Task ADR-019-T2: The pick's first slice, through the built binary

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** Binary behaves as the pick
**Consumes:** Naming pick encoded in this Decision as `Naming pick:` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a write cannot spend another root's ledger`, `the binary calls the naming the pick chose`, `§78 drives the built binary`, `Naming pick is encoded`, `go.mod still has one requirement`, `engine dirs stay byte-identical`

## Goal

Implement only the first slice the encoded pick can hold, and prove through `$MRW` that a write is
licensed only by the ledger of the root that served the read.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/root.go` | edit | only if B or C extends `Source` / CheckRoot; pick A leaves this file |
| `internal/mcp/mcp.go` | edit | only if B or C changes `Serve`'s one-root signature; pick A leaves this file |
| `internal/mcp/tools.go` | edit | the call site that must pass the *call's* root into `hold` / `promote` / `read.Run` / `apply.Apply` if the process can see more than one |
| `internal/mcp/root_test.go` | edit | `TestAWriteCannotSpendAnotherRootsLedger` — the two-root fixture |
| `cmd/mrw/main.go` | edit | **THE CALL SITE** if B adds repeatable `--root` on `mcp`, or C starts a client request. Pick A: README-only, no flag change |
| `scripts/contract.sh` | edit | **§78** — drives `$MRW`, pairs the good case with the case that must fail |

Pick A first slice: pin the isolation that already exists (two servers, two `--root`, A's ack
does not license B) through the built binary, and stop. That is not a no-op: §53 proved
`CheckRoot` can be correct and unwired; this row proves two ledgers do not mix.

Pick B first slice: launch-time allow-list; a call that names a listed root serves it; a call
that names an unlisted root is refused; ack on root A does not license a write on root B.

Pick C first slice: only after a measurement that Desktop sends `roots/list`. Without that
measurement, Stop Condition.

## Ordered Steps

1. [S1] Write `TestAWriteCannotSpendAnotherRootsLedger` so it is RED against a mutant that
   promotes A's checkpoints into B's ledger (shared `XDG_STATE_HOME`, two roots). If T1's
   `Naming pick:` line is missing, stop before implementing — do not guess. ⚠ A fixture that
   uses two `XDG_STATE_HOME` values is green against a shared pending store. [proof: mutation]
2. [S2] Implement only the pick's first slice. Do not touch `internal/apply` or `internal/plan`.
   If B or C, `hold`/`promote` take the call's root, not a process global. [proof: mutation]
3. [S3] Wire it. Pick B: the flag parser on `mcp` (or a dedicated allow flag) is what *selects*
   the list — a function nothing calls is ADR-018's unwired-guard class. Pick A: no new flag.
   Pick C: do not wire without the Desktop measurement. [proof: mutation]
4. [S4] Add `# 78.` to `scripts/contract.sh`. Pair the good case with the case that must fail. Drive
   `$MRW`. [proof: acceptance]
5. [S5] Run `gofmt` and `go vet` unpiped on the packages this task changed. Engine go/no-go against
   the merge-base. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -qE '^Naming pick: (A — launch --root only|B — launch allow-list|C — MCP roots/list)$' \
  docs/adr/ADR-019-desktop-reach-is-one-named-root-per-run.md \
  && go test ./internal/mcp/ -count=1 -v \
    -run 'TestAWriteCannotSpendAnotherRootsLedger' 2>&1 \
  | tee /tmp/adr019-t2.out \
  && grep -q '^--- PASS: TestAWriteCannotSpendAnotherRootsLedger' /tmp/adr019-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr019-t2.out \
  && grep -q '^# 78\. ' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state \
  && [ -z "$(gofmt -l internal/mcp cmd/mrw)" ] \
  && go vet ./internal/mcp/ ./cmd/mrw/
```

Every clause was grepped for BEFORE this fence was written and returned **zero hits**: the test
name `TestAWriteCannotSpendAnotherRootsLedger` and `# 78.`. Highest contract section on
`origin/main` (`c444fc5`) is `# 77.`.

⚠ **`go test -run` of a missing test exits 0.** The `grep PASS` and `no tests to run` clauses are
what keep this fence red until S1 lands.

⚠ **The engine go/no-go is two checks**, as ADR-010: porcelain for untracked new files, `git diff`
against the merge-base for committed ones.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteCannotSpendAnotherRootsLedger` | `internal/mcp/root_test.go` | Shared `XDG_STATE_HOME`; hold/ack on root A; write to the same relative path under root B is refused; a write under A with A's ack applies | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAWriteCannotSpendAnotherRootsLedger` |
| 2 — something selects it | `mcpCmd` → `Serve` (A) or the allow-list parser (B) or the `roots/list` request (C). Deleting the call site keeps the unit test green and fails §78 |
| 3 — the caller can discover it | T3. Pick A: README already has the `--root` block. Pick B/C: schema / handshake |
| 4 — it is used | nothing measures this yet — ADR-009 refuses transmission of caller outcomes; §78 is the evidence the binary does it |

## Mutation Log

- 2026-09-10 · c444fc5* · mutant killed · exit 1 · `internal/mcp/ack.go` · shared pending lets A license B · acceptance-sha256:2f3ed1243465786b2491af0ff52fe1bc6c007c78dc6f17531da2c83f06f69fab · covers:a write cannot spend another root's ledger

## Invariants

- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`,
  `internal/state` stay byte-identical against the merge-base.
- `go.mod` still declares exactly one requirement.
- ADR-018: explicit `--root /` still starts; fallback `/` still exits 2.
- A CLI `mrw --root DIR read` is unchanged.
- Cargo tools are not added.
- Under pick B or C, a run still has one root.

## Risks

- ⚠ Two `t.TempDir` roots with two `XDG_STATE_HOME` values never share a pending store, so the
  isolation test is green against a process-global pending map. The fixture shares one state home.
- Pick C wired without a Desktop measurement ships a client request that hears nothing and then
  falls back to cwd — ADR-018's defect. Stop Condition below.
- §78 that only scores the good case passes a tool that always says yes. Pair the refusal.

## Stop Condition

Stop and ask if the pick is C and no Desktop `roots/list` measurement exists. Stop if implementing
the pick requires changing `apply.Apply`'s one-root signature, adding a `go.mod` requirement, or
raising 4096. Stop if T1 has no `Naming pick:` line.

## Out of Scope

- Teaching — that is T3's job
- Encoding the pick — that is T1's job
- Per-hunk ledger (permanent: boundary: parent Decision 1)
- MCP `check` / `iter` / `seen` / `stats` (permanent: boundary: parent Decision 5)

## Verification Log
- 2026-09-10 · c444fc5* · exit 0 · `set -o pipefail …` · acceptance-sha256:2f3ed1243465786b2491af0ff52fe1bc6c007c78dc6f17531da2c83f06f69fab · ms:919
