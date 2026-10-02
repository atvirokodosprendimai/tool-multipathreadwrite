# Task ADR-119-T2: the measurement against the pre-registered bar

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `docs/break/shape-hints/stress.py`, `docs/break/shape-hints/README.md`
**Consumes:** `apply.HunkResult.Closer` (T1)
**Data dependency:** the non-merge history of the repositories under `~/GolandProjects` and `~/CursorProjects` on this machine, measured once; the result is recorded in the README with its date and corpus
**Proof map:** v1
**Rests-on:** `the measurement against the pre-registered bar`

## Goal

Each hint's false-positive rate per language bucket, from real history, against the bar BACKLOG registered before measuring; a hint over the bar is withdrawn before the PR merges.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/break/shape-hints/stress.py` | add | the replay |
| `docs/break/shape-hints/README.md` | add | the method and the result |
| `docs/adr/ADR-119-a-write-says-when-its-shape-looks-wrong.md` | edit | the numbers, and which hint ships |

## Ordered Steps

1. [S1] `stress.py`: for each repository, each non-merge commit, each modification hunk of `git diff -U0 k^ k` (vendored, generated, lock and `node_modules` files skipped; a cap of hunks a repository): the parent's lines as the original, the hunk as a native replace, every variant counted by extension bucket; with `--mrw`, the built binary's `write --dry-run --json --no-check` on each, its `closer` compared with the computed one. [proof: human: the run's numbers go in the README with the corpus and date]
2. [S2] The result against the bar, in the README and the record: run 1 on the registered heuristics, the amendment Zy chose registered before run 2 on a fresh sample, and the binary cross-check. A hint over the bar is taken out of T1's code before merge. [proof: human: the bar comparison is a judgement on the recorded numbers]

## Acceptance

```bash
set -o pipefail
python3 -c 'import ast,sys; ast.parse(open(sys.argv[1]).read())' docs/break/shape-hints/stress.py \
  && grep -q 'false-positive rate' docs/break/shape-hints/README.md \
  && grep -q 'Measured 2026-' docs/break/shape-hints/README.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | the measurement is a recorded run, not a test | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `stress.py` |
| 2 — something selects it | the bar in BACKLOG names it |
| 3 — the caller can discover it | the README and the record |
| 4 — it is used | it decides which hint ships |

## Mutation Log
- 2026-10-02 · ac4fb85* · mutant killed · exit 1 · `docs/break/shape-hints/README.md` · S2: the README carries the measured rate the fence asks for · acceptance-sha256:b4a0757f7df9d65bcd285d8e9e301c5df659f3bedbbfc8d476bccaa6cc7fa051

## Invariants

- The bar is the one registered in BACKLOG before the run; the run does not move it.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask Zy if a hint misses the bar: whether to withdraw it or to change it and measure again.

## Out of Scope

- A continuous measurement in CI (permanent: boundary: the corpus is this machine's repositories, not the project's)

## Verification Log
- 2026-10-02 · ac4fb85* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4a0757f7df9d65bcd285d8e9e301c5df659f3bedbbfc8d476bccaa6cc7fa051 · ms:49
- 2026-10-02 · ac4fb85* · exit 2 · `set -o pipefail …` · acceptance-sha256:b4a0757f7df9d65bcd285d8e9e301c5df659f3bedbbfc8d476bccaa6cc7fa051 · ms:47 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  --- last 1 line(s) of stderr
  grep: docs/break/shape-hints/README.md: No such file or directory
  ```
- 2026-10-02 · ac4fb85* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4a0757f7df9d65bcd285d8e9e301c5df659f3bedbbfc8d476bccaa6cc7fa051 · ms:103
