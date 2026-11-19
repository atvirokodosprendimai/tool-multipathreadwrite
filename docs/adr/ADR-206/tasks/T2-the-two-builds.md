# Task ADR-206-T2: M and N are two builds of one tag that differ in one refusal

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `bench/stage-9/no-refusal.patch`, `bench/stage-9/build.sh`, the kept log's path and format, `bench/stage-9/builds.md` (the tag, the toolchain, the flags, both hashes)
**Consumes:** ADR-203's refusal (`internal/apply`), `writer.Land`
**Data dependency:** needs the pinned tag in the local clone and the module cache; builds two binaries in a temporary directory (about a minute cold, an estimate). CI's shallow checkout has no tag, so the test is skipped there with its reason and the operator fence forbids the skip
**Proof map:** v1
**Rests-on:** `N lands the hunk M refuses`, `N logs one object per exposed landed hunk`, `a dry run and a refused plan log nothing`, `N still refuses an unread line`, `N says what M says`, `the log's failure changes nothing an agent sees`, `no tracked Go file holds the patch`

## Goal

ADR-206 Decision 4. `build.sh TAG OUTDIR` builds `mrw-m` and `mrw-n`. The test builds both and drives them from identical fresh trees and state homes:

| Case | M | N |
|------|---|---|
| write a file, then replace a line of it by number alone | exit 1, ADR-203's refusal | exit 0, the line replaced; one object in `mrw/stage9-exposed.jsonl` with `path`, `op`, `start`, `end`, `first_unshown`, `sha_before` |
| the same as a numbered `delete` of two lines, and as an `insert-after` | exit 1 | exit 0; one object each, `op` and the span as resolved (a multi-line `replace` needs `anchor=`, which takes it out of this rule) |
| a plan of two files that commits one and fails on the other, the landed one holding such a hunk | — | one object, for the hunk that landed |
| the same under `--dry-run`, and in a plan whose other hunk fails | exit as M | nothing landed; no object |
| the same second edit with `anchor=` | exit 0 | exit 0; no object; the log exists and is empty |
| replace by number a line no read has served, in a file mrw did not write | exit 1 | exit 1, the same reason |
| `version`, `instructions`, `instructions --core`, `skill` | — | byte for byte M's |
| the first case with the log's path made a directory | — | exit 0, stdout and stderr what they were with the log writable |

And no tracked `.go` file contains the patch's added lines.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/stage-9/no-refusal.patch`, `bench/stage-9/build.sh`, `bench/stage-9/builds.md` | new | the builds |
| `internal/bench/stage9_build_test.go` | new | the Goal's table |

## Ordered Steps

1. [S1] `TestArmNDiffersFromMInOneRefusal`, RED: `build.sh` does not exist. [proof: acceptance]
2. [S2] The patch and the script. Mutants on the patch, each killed: N refuses as M does; N logs nothing; N logs a dry run; N stops checking unread lines; N's version string differs; a failed append prints a line. [proof: mutation]
3. [S3] `builds.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/bench -run '^TestArmNDiffersFromMInOneRefusal$' > "${TMPDIR:-/tmp}/adr206-t2.out" \
  && [ "$(grep -c '^--- PASS: TestArmNDiffersFromMInOneRefusal ' "${TMPDIR:-/tmp}/adr206-t2.out")" -eq 1 ] \
  && ! grep -q -- '--- SKIP' "${TMPDIR:-/tmp}/adr206-t2.out" \
  && test -s bench/stage-9/no-refusal.patch \
  && m=$(sed -n 's/^m sha256: //p' bench/stage-9/builds.md) && n=$(sed -n 's/^n sha256: //p' bench/stage-9/builds.md) \
  && [ ${#m} -eq 64 ] && [ ${#n} -eq 64 ] && [ "$m" != "$n" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestArmNDiffersFromMInOneRefusal` | `internal/bench/stage9_build_test.go` | every row of the Goal's table by driving the two built binaries | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test builds and drives both |
| 2 — something selects it | `--mrw` of arm M's and arm N's runs |
| 3 — the caller can discover it | `bench/stage-9/builds.md`, `docs/bench/README.md` |
| 4 — it is used | T5's smoke and T6 |

## Mutation Log
- 2026-11-19 · e83e4009* · mutant killed · exit 1 · `bench/stage-9/no-refusal.patch` · N logs nothing · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · covers:N logs one object per exposed landed hunk
- 2026-11-19 · e83e4009* · mutant killed · exit 1 · `bench/stage-9/no-refusal.patch` · N never makes an event · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · covers:N lands the hunk M refuses
- 2026-11-19 · e83e4009* · mutant killed · exit 1 · `bench/stage-9/build.sh` · N reports another version · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · covers:N says what M says
- 2026-11-19 · e83e4009* · mutant survived · exit 0 · `bench/stage-9/no-refusal.patch` · N logs hunks of files that did not land · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · covers:a dry run and a refused plan log nothing
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-19 · e83e4009* · mutant killed · exit 1 · `bench/stage-9/no-refusal.patch` · a hunk of a file that did not land is reported as exposed · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · covers:a dry run and a refused plan log nothing

## Verification Log
- 2026-11-19 · e83e4009* · exit 1 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:993 · test-lock-sha256:2d3c35b74823626e45b8c1983c21a73559091c190e2412494f5db530ee7e2ba5 · test-lock-file:docs/adr/locks/2d3c35b74823626e45b8c1983c21a73559091c190e2412494f5db530ee7e2ba5.lock
  ```
  ```
- 2026-11-19 · e83e4009* · exit 0 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:3111
- 2026-11-19 · e83e4009* · exit 0 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:5915
- 2026-11-19 · e83e4009* · exit 0 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:5135
- 2026-11-19 · e83e4009* · exit 0 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:5342
- 2026-11-19 · e83e4009* · exit 0 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:4333
- 2026-11-19 · e83e4009* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:1 · test-lock-sha256:f2e00966f780447e6e0783b0669a9d885c014085f4b311d1c69edca203f2be7b · test-lock-file:docs/adr/locks/f2e00966f780447e6e0783b0669a9d885c014085f4b311d1c69edca203f2be7b.lock · test-lock-kind:replace
- 2026-11-19 · e83e4009* · exit 0 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:4421
- 2026-11-19 · e83e4009* · exit 0 · `set -o pipefail …` · acceptance-sha256:94538eb3982f5b9d8d0351bcef625182684bb38901f8c97b6a02782aadb1cde3 · ms:3575
- 2026-11-19 · 2ec60d27* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:0 · test-lock-sha256:60aa3a98558132f91610e3e1b325ed8e3797ea62dec7a7037b1c88f951602a32 · test-lock-file:docs/adr/locks/60aa3a98558132f91610e3e1b325ed8e3797ea62dec7a7037b1c88f951602a32.lock · test-lock-kind:replace
- 2026-11-19 · 2ec60d27* · exit 0 · `set -o pipefail …` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:7284

## Invariants

- No file under `internal/` or `cmd/mrw` changes in this repository but the new test.
- Neither binary is installed or released.

## Risks

- The patch is against one tag and is re-made when the tag changes.

## Stop Condition

The fence exits 0 without a skip, and every gate in `AGENTS.md` passes.

## Out of Scope

- Carrying the switch in shipped source (permanent: boundary: ADR-206 Alternatives)
