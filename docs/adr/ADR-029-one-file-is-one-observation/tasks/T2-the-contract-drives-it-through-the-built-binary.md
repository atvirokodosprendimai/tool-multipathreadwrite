# Task ADR-029-T2: The contract drives it through the built binary

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (one contract section)
**Owner:** Zy
**Produces:** contract §67
**Consumes:** the per-line ledger check consuming the alias-resolved observation (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built binary refusing an alias-spelled write to unserved lines`, `the built binary still applying an alias-spelled write after a whole read`

## Goal

Prove in the BUILT binary that an alias spelling is the same file to the per-line ledger — both that
it is refused for lines never served, and that it is still accepted for lines that were.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 67.` section. A unit test proves the resolution; it cannot prove the binary reaches it, which is why §53 exists. The row pairs the refusal with the case that must still succeed — a row asserting only the refusal passes against a binary that refuses every alias, which is issue #47 undone. |

## Ordered Steps

1. [S1] Confirm §67 does not exist and that the row is RED against a binary built before T1: `grep -c '^# 67\. ' scripts/contract.sh` is 0 first, and the section fails against the pre-ADR-029 tree — where the alias write APPLIES at exit 0. [proof: acceptance]
2. [S2] Write §67: with `link.txt -> real.txt` and `real.txt:1` served, `@@ link.txt 4 replace` exits 1, writes nothing, and the file is byte-identical; then with `real.txt` served whole, the same hunk exits 0 and the file changes. [proof: acceptance]
3. [S3] Assert the anchored case in the same section: with a narrow read, an alias-spelled anchored hunk prints no line the caller was not served — ADR-028's §66 property, which only holds for the alias once T1 lands. [proof: acceptance]
4. [S4] ⚠ Say in the section's comment that it carries the SYMLINK half only. `scripts/contract.sh` runs on Linux, where a case-only variant names two different files; the case-only half is a Go test guarded by a runtime probe and runs on Windows CI. A row that silently covers half a defect is worse than one that says which half. [proof: human: read the section comment and confirm it names the half it cannot carry and where that half is covered]
5. [S5] Run every gate, including `go test -race ./...`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 67\. ADR-029: one file is one observation, whatever the plan calls it\.' scripts/contract.sh \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr029-t2.out \
  && grep -q '^contract holds$' /tmp/adr029-t2.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr029-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§67` | `scripts/contract.sh` | The built binary refuses an alias-spelled write to lines never served with the PER-LINE refusal and leaves the file byte-identical by digest, still applies one after a whole read, and prints no unserved line on an alias-spelled anchor failure | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §67 in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section against `$MRW`, the binary built at the top of the script |
| 3 — the caller can discover it | The refusal is the ledger's existing message; no new wording to document |
| 4 — it is used | The contract runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · cd259f2* · mutant killed · exit 1 · `internal/apply/apply.go` · the per-line gate goes back to its own exact-key lookup, so §67 sees the built binary APPLY an alias-spelled write to a line never served — the row drives $MRW, which a unit test cannot · acceptance-sha256:168c71c1acfb96fd88a82f177f928c51fe99a354c9ab52b7fac6a3d6eac4f389 · covers:the built binary refusing an alias-spelled write to unserved lines
- 2026-09-07 · cd259f2* · mutant killed · exit 1 · `internal/apply/apply.go` · alias recovery is dropped in the built binary, so §67 sees the whole-read half refused too and the change reads as a ban rather than a resolution · acceptance-sha256:168c71c1acfb96fd88a82f177f928c51fe99a354c9ab52b7fac6a3d6eac4f389 · covers:the built binary still applying an alias-spelled write after a whole read

## Invariants

- Every existing contract section still passes; §67 uses its own fixture and touches no other section's `$R`.
- The row drives `$MRW`, never a Go test.
- The section names the half it cannot carry, rather than reading as full coverage.

## Risks

- A row asserting only the refusal passes against a binary that refuses every alias. Mitigated by S2 requiring the whole-read success in the same section.
- A reader takes the Linux-only row as covering the case-only alias. Mitigated by S4's human proof on the section comment.

## Stop Condition

Stop and ask if §67 cannot be made red against the pre-ADR-029 tree: a row that is green before the
fix exists is asserting nothing, and finding that out here is the point of S1.

## Out of Scope

- The resolution itself — that is T1's job
- Any change to how the ledger is keyed on disk (permanent: boundary: ADR-029's Alternatives rejects re-keying; `mrw seen` prints the keys the caller typed)

## Verification Log
- 2026-09-07 · cd259f2* · exit 0 · `set -o pipefail …` · acceptance-sha256:168c71c1acfb96fd88a82f177f928c51fe99a354c9ab52b7fac6a3d6eac4f389 · ms:32259
- 2026-09-07 · cd259f2* · exit 0 · `set -o pipefail …` · acceptance-sha256:168c71c1acfb96fd88a82f177f928c51fe99a354c9ab52b7fac6a3d6eac4f389 · ms:30438
- 2026-09-07 · cd259f2* · exit 0 · `set -o pipefail …` · acceptance-sha256:168c71c1acfb96fd88a82f177f928c51fe99a354c9ab52b7fac6a3d6eac4f389 · ms:30221
- 2026-09-07 · 2f8adce* · exit 0 · `set -o pipefail …` · acceptance-sha256:168c71c1acfb96fd88a82f177f928c51fe99a354c9ab52b7fac6a3d6eac4f389 · ms:31183
