# Task ADR-206-T2: M and N are two builds of one tag that differ in one refusal

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `bench/stage-9/no-refusal.patch`, `bench/stage-9/build.sh`, the kept log's path and format, `bench/stage-9/builds.md` (the tag, the toolchain, the flags, both hashes)
**Consumes:** ADR-203's refusal (`internal/apply`), `writer.Land`
**Data dependency:** needs the pinned tag in the local clone and the module cache; builds two binaries in a temporary directory (about a minute cold, an estimate). CI's shallow checkout has no tag, so the test is skipped there with its reason and the operator fence forbids the skip
**Proof map:** v1
**Rests-on:** `N lands the hunk M refuses`, `N logs one object per exposed landed hunk`, `a dry run and a refused plan log nothing`, `N still refuses an unread line`, `N says what M says`, `the log's failure changes nothing an agent sees but one stderr line`, `no tracked Go file holds the patch`

## Goal

ADR-206 Decision 4, as ADR-211 Decision 3 amends it (T1 of ADR-211). `build.sh TAG OUTDIR` builds `mrw-m` and `mrw-n`. The test builds both and drives them from identical fresh trees and state homes; N's log is `stage9-exposed.jsonl` in the parent of `$XDG_STATE_HOME`, beside the state home and none inside it:

| Case | M | N |
|------|---|---|
| write a file, then replace a line of it by number alone | exit 1, ADR-203's refusal | exit 0, the line replaced; one object in the log with `path`, `op`, `start`, `end`, `first_unshown`, `sha_before`; none in the state home |
| the same as a numbered `delete` of two lines, and as an `insert-after` | exit 1 | exit 0; one object each, `op` and the span as resolved (a multi-line `replace` needs `anchor=`, which takes it out of this rule) |
| a plan of two files that commits one and fails on the other, the landed one holding such a hunk | — | one object, for the hunk that landed |
| the same under `--dry-run`, and in a plan whose other hunk fails | exit as M | nothing landed; no object, and no log file |
| the same second edit with `anchor=` | exit 0 | exit 0; no object; the anchored write leaves no log file at all |
| replace by number a line no read has served, in a file mrw did not write | exit 1 | exit 1, the same reason |
| `version`, `instructions`, `instructions --core`, `skill` | — | byte for byte M's |
| the first case with the log's path made a directory, or a FIFO with no reader | — | exit 0, the same stdout, the file as written, and one more line on stderr, `mrw: stage9 log: <what failed>` |
| an event logged by one write, then a later write whose append fails | — | the later write lands, prints the one line, and the log still holds the first event |
| `$XDG_STATE_HOME` unset (mrw itself refuses a relative one) | — | the write lands as it did, with the one `mrw: stage9 log:` line |

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
- 2026-11-19 · c3604110 · mutant killed · exit 1 · `bench/stage-9/builds.md, bench/stage-9/no-refusal.patch` · N logs nothing (the platform pin moved aside so the behaviour assertions run) · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · covers:N logs one object per exposed landed hunk
- 2026-11-19 · c3604110* · mutant killed · exit 1 · `bench/stage-9/builds.md` · builds.md records a hash the build does not produce · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · covers:N says what M says

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
- 2026-11-19 · c3604110 · exit 0 · `set -o pipefail …` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:4594
- 2026-11-19 · c3604110* · exit 0 · `set -o pipefail …` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:3881
- 2026-11-19 · 0da8265e · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:0 · test-lock-sha256:60aa3a98558132f91610e3e1b325ed8e3797ea62dec7a7037b1c88f951602a32 · test-lock-file:docs/adr/locks/60aa3a98558132f91610e3e1b325ed8e3797ea62dec7a7037b1c88f951602a32.lock · test-lock-kind:replace
- 2026-11-19 · 0da8265e* · exit 0 · `set -o pipefail …` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:3796
- 2026-11-25 · 1e55658d* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:14 · test-lock-sha256:d3ddea24e3d73733eeccdf64fefc9d28ceb68baeace5cf63660d72d254331512 · test-lock-file:docs/adr/locks/d3ddea24e3d73733eeccdf64fefc9d28ceb68baeace5cf63660d72d254331512.lock · test-lock-kind:replace
- 2026-11-25 · 222928de* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:18 · test-lock-sha256:8a9c19d35bc25a4b938896e90deba0cc6cd36b2ace484692d442a9f99df9e5fd · test-lock-file:docs/adr/locks/8a9c19d35bc25a4b938896e90deba0cc6cd36b2ace484692d442a9f99df9e5fd.lock · test-lock-kind:replace
- 2026-11-26 · 3657eb95* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:21 · test-lock-sha256:314198c1f88224cf56c4bfb5a5a051079392d5efa89e08e7ed996f8c8300f886 · test-lock-file:docs/adr/locks/314198c1f88224cf56c4bfb5a5a051079392d5efa89e08e7ed996f8c8300f886.lock · test-lock-kind:replace
- 2026-11-26 · 9eb2f968* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:15 · test-lock-sha256:06ee083d631faaa628e969fd66fa1e23f3dd493cbf9b6c5aa31f69774f58f366 · test-lock-file:docs/adr/locks/06ee083d631faaa628e969fd66fa1e23f3dd493cbf9b6c5aa31f69774f58f366.lock · test-lock-kind:replace
- 2026-11-26 · 837587b0 · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:20 · test-lock-sha256:06ee083d631faaa628e969fd66fa1e23f3dd493cbf9b6c5aa31f69774f58f366 · test-lock-file:docs/adr/locks/06ee083d631faaa628e969fd66fa1e23f3dd493cbf9b6c5aa31f69774f58f366.lock · test-lock-kind:replace
- 2026-11-26 · ea298826 · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:17 · test-lock-sha256:06ee083d631faaa628e969fd66fa1e23f3dd493cbf9b6c5aa31f69774f58f366 · test-lock-file:docs/adr/locks/06ee083d631faaa628e969fd66fa1e23f3dd493cbf9b6c5aa31f69774f58f366.lock · test-lock-kind:replace
- 2026-11-26 · e60d3102* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:17 · test-lock-sha256:6ed0035becf9f8861e32197d3f06486e2d6cfaa142d7e5b748869655faf80bc8 · test-lock-file:docs/adr/locks/6ed0035becf9f8861e32197d3f06486e2d6cfaa142d7e5b748869655faf80bc8.lock · test-lock-kind:replace
- 2026-11-26 · e2718628* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:31 · test-lock-sha256:ebe087b0f2e2d8efd51ec4cfba4744985ed8a6acb0146ea9a89705940690873d · test-lock-file:docs/adr/locks/ebe087b0f2e2d8efd51ec4cfba4744985ed8a6acb0146ea9a89705940690873d.lock · test-lock-kind:replace
- 2026-11-26 · 0de0a0bf* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:17 · test-lock-sha256:ba03ff26f9d7af10fdb6f49094d6c91f098736f2a5c8a64f62566db471f120a4 · test-lock-file:docs/adr/locks/ba03ff26f9d7af10fdb6f49094d6c91f098736f2a5c8a64f62566db471f120a4.lock · test-lock-kind:replace
- 2026-11-26 · 62c9fe8a* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:18 · test-lock-sha256:e8735053a833072ab6758be40bcd8c0f44a818356ceace6d2f169d573d4e3228 · test-lock-file:docs/adr/locks/e8735053a833072ab6758be40bcd8c0f44a818356ceace6d2f169d573d4e3228.lock · test-lock-kind:replace
- 2026-11-26 · ce74e857* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:32 · test-lock-sha256:1c80e4a52eb4f0a4605fe72aed457e4dd04a2e77c7fd6b101a65ae069e10fe8f · test-lock-file:docs/adr/locks/1c80e4a52eb4f0a4605fe72aed457e4dd04a2e77c7fd6b101a65ae069e10fe8f.lock · test-lock-kind:replace
- 2026-11-26 · 163ef3cd* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:90b8e67864f72d49cf57e2a9b9973c7f4d01f37159b0418ba60c1ad1d74ba094 · ms:22 · test-lock-sha256:6214f25dd5cf937daa8bdfd6b76aa6c0f59e78740b5f7e3fccad89b3a4d95a85 · test-lock-file:docs/adr/locks/6214f25dd5cf937daa8bdfd6b76aa6c0f59e78740b5f7e3fccad89b3a4d95a85.lock · test-lock-kind:replace

## Invariants

- No file under `internal/` or `cmd/mrw` changes in this repository but the new test.
- Neither binary is installed or released.

## Risks

- The patch is against one tag and is re-made when the tag changes.

## Stop Condition

The fence exits 0 without a skip, and every gate in `AGENTS.md` passes.

## Out of Scope

- Carrying the switch in shipped source (permanent: boundary: ADR-206 Alternatives)
