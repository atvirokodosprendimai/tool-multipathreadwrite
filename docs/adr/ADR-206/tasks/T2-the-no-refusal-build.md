# Task ADR-206-T2: arm N's binary differs from M's in one refusal

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `bench/stage-9/no-refusal.patch`, `bench/stage-9/build-n.sh`, the exposed log's format, M's and N's hashes in `bench/stage-9/builds.md`
**Consumes:** the tag M is built from; ADR-203's refusal in `internal/apply`
**Data dependency:** needs the M tag in the local clone and a Go toolchain; builds two binaries in a temporary directory
**Proof map:** v1
**Rests-on:** `N lands the hunk M refuses`, `N logs the path, the line and its text`, `N still refuses an unread line`, `N says what M says`, `no build of this repository applies the patch`

## Goal

ADR-206 Decision 4. `build-n.sh TAG OUT` builds N; the test builds M from the same tag with the same flags and drives both in fresh trees:

| Case | M | N |
|------|---|---|
| write a file, then replace a line of it by number alone | exit 1, ADR-203's refusal | exit 0, the line replaced; one log line `PATH<TAB>LINE<TAB>TEXT` |
| replace by number a line no read has served, in a file mrw did not write | exit 1 | exit 1, the same reason; no log line |
| the same second edit with `anchor=` | exit 0 | exit 0; no log line |
| `mrw version`, `mrw instructions`, `mrw instructions --core`, `mrw skill` | — | byte for byte M's |
| the first case with the state directory unwritable | — | what M does there; no log |

And `git grep` finds the patch's added lines in no tracked `.go` file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/stage-9/no-refusal.patch`, `bench/stage-9/build-n.sh` | new | the build |
| `bench/stage-9/builds.md` | new | the tag, the flags, both hashes |
| `internal/bench/stage9_build_test.go` | new | the Goal's table, skipped with a reason when the tag is absent |
| `scripts/static.sh` | edit | shellcheck covers the new script (it lists scripts by directory) |

## Ordered Steps

1. [S1] `TestArmNDiffersFromMInOneRefusal`, RED: there is no patch. [proof: acceptance]
2. [S2] The patch and the script. Mutants on the patch, each killed: N refuses as M does; N logs nothing; N stops checking unread lines; N's version string differs. [proof: mutation]
3. [S3] `builds.md` with the tag and both hashes. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/bench -run '^TestArmNDiffersFromMInOneRefusal$' > "${TMPDIR:-/tmp}/adr206-t2.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr206-t2.out")" -eq 1 ] \
  && ! grep -q '^--- SKIP' "${TMPDIR:-/tmp}/adr206-t2.out" \
  && test -s bench/stage-9/no-refusal.patch && grep -qE '^n sha256: [0-9a-f]{64}$' bench/stage-9/builds.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestArmNDiffersFromMInOneRefusal` | `internal/bench/stage9_build_test.go` | every row of the Goal's table by driving both built binaries | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test builds and drives it |
| 2 — something selects it | `--mrw` of arm N's run |
| 3 — the caller can discover it | `bench/stage-9/builds.md`, `docs/bench/README.md` |
| 4 — it is used | T5's smoke and T6 |

## Mutation Log

## Verification Log

## Invariants

- No file under `internal/` or `cmd/mrw` changes.
- N's output to an agent equals M's except where the refusal would have printed.

## Risks

- The test builds two binaries (about a minute cold): it is one test, run once per fence.

## Stop Condition

The fence exits 0 without a skip, and every gate in `AGENTS.md` passes.

## Out of Scope

- Carrying the switch in shipped source (permanent: boundary: ADR-206 Alternatives)
