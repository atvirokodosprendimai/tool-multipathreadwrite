# Task ADR-166-T1: the CLI read serves, records nothing, and says so

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the read action's handling of a state-directory failure from `seen.Record` and `iter.Load`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the read's own exit code`, `the nothing-was-recorded line`, `no licence exists after it`, `a read that needs the set still refuses`

## Goal

Decision 1: `mrw read` with an unwritable or uncreatable state directory serves its answer, exits as the read itself would, and ends stderr with `mrw: nothing was recorded, …`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the read action: `iter.Load`'s state failure skipped when no `@N` and at least one argument; `seen.Record`'s state failure becomes the stderr line and the read's own exit |
| `cmd/mrw/readonly166_test.go` | add | the test, through the built binary |
| `cmd/mrw/statehint_test.go` | edit | ADR-164 T3's `{"read", "a.txt"}` row now expects the content served and exit 0 with the line; its plan-read row keeps exit 2 (Decision 3) |
| `scripts/contract.sh` | edit | §284; §272's read row moves from exit 2 to exit 0 with the line (superseded by Decision 1), its message check kept |
| `README.md`, `AGENTS.md` | edit | one sentence each |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestAReadWithAnUnwritableLedgerServesAndSaysNothingWasRecorded (the built binary, `XDG_STATE_HOME` (a) a read-only existing state tree and (b) under a read-only parent: `mrw read f.txt:1` serves `1| alpha`, exits 0, the last stderr line begins `mrw: nothing was recorded`; `mrw read f.txt:9` exits 1 with the line; `--files-from` naming `f.txt:1` serves; `mrw read @1`, a bare `mrw read`, `--files-from` naming `@1`, `--quiet` and `--grep a --sub b` still exit 2; with the state writable again, a write to line 1 is refused as unread, since the setup read licensed another file). [proof: acceptance]
2. [S2] The two call sites. [proof: mutation]
3. [S3] The README and AGENTS.md sentences; contract §284 (the pair: the same read with a writable state directory records, and the write to line 1 applies). [proof: human: the executor runs ./scripts/contract.sh once and reads the §284 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAReadWithAnUnwritableLedgerServesAndSaysNothingWasRecorded$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAReadWithAnUnwritableLedgerServesAndSaysNothingWasRecorded \(' "$out" \
  && grep -q '^# 284\. ' scripts/contract.sh \
  && grep -q 'nothing was recorded' README.md \
  && grep -q 'nothing was recorded' AGENTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadWithAnUnwritableLedgerServesAndSaysNothingWasRecorded` | `cmd/mrw/readonly166_test.go` | both state shapes; the exit codes; the set-needing reads; no licence after | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two call sites |
| 2 — something selects it | every `mrw read` passes them; contract §284 drives the binary |
| 3 — the caller can discover it | the stderr line; README §Exit status |
| 4 — it is used | the Codex agent's read-only sandbox of 2026-10-21 |

## Invariants

- No licence is recorded; a later write to the served lines is refused.
- Only `statehint.About` errors take the new path; every other error exits as today.
- The answer is flushed before the line is printed.

## Risks

- A test that makes a directory read-only must restore it, or `t.TempDir` cleanup fails: the test chmods back in a cleanup.

## Out of Scope

- Windows ACLs: the test's read-only directory is POSIX mode bits (permanent: boundary: Windows CI runs the same code path through `statehint.About`; the permission setup is unix-only and the test skips on Windows)

## Stop Condition

Stop if a state failure can reach the read action from a call this record does not name (the audit's grep missed one): add it to the record first.

## Mutation Log
(empty until execute)
- 2026-10-21 · 13132ca1* · mutant killed · exit 1 · `cmd/mrw/main.go` · a content read whose ledger cannot be written exits 2 again · acceptance-sha256:f6b0a940471f3e9eba02bcf2f4948bbcda9c38aaca82b06faa8e519c8c86ab40
- 2026-10-21 · 13132ca1* · mutant killed · exit 1 · `cmd/mrw/main.go` · a read that names its specs still needs the working set · acceptance-sha256:f6b0a940471f3e9eba02bcf2f4948bbcda9c38aaca82b06faa8e519c8c86ab40

## Verification Log
(empty until execute)
- 2026-10-21 · 13132ca1* · exit 0 · `set -o pipefail …` · acceptance-sha256:f6b0a940471f3e9eba02bcf2f4948bbcda9c38aaca82b06faa8e519c8c86ab40 · ms:5156
- 2026-10-21 · 13132ca1* · exit 0 · `set -o pipefail …` · acceptance-sha256:f6b0a940471f3e9eba02bcf2f4948bbcda9c38aaca82b06faa8e519c8c86ab40 · ms:5317
