# Task ADR-186-T1: the guard resolves a plan file's path

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `resolveTargets` in `internal/bench/hook.go`; `TestAPlanFileAtAResolvablePathOutsideTheTreePassesTheHook`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a resolvable plan path outside the tree passes`, `a rebindable name is not resolved`

## Goal

Decisions 1-4 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/hook.go` | edit | `resolveTargets`, called by `classify` on the text outside heredoc bodies |
| `internal/bench/hook_test.go` | edit | the new test; `TestAWriteOutsideTheTaskTreePassesTheHook` drops its `$TMPDIR/plan.txt` refusal, which the new test passes |

## Ordered Steps

1. [S1] Write `TestAPlanFileAtAResolvablePathOutsideTheTreePassesTheHook`, with `TMPDIR` set to the attempt's scratch directory, which holds the tree: each refused shape from the ablation passes (`P=$TMPDIR/plan.txt; : > $P`, `cat > $TMPDIR/plan.txt`, `cat > "$TMPDIR/plan.txt"`, `printf x > ${TMPDIR}/p.txt`, `p=$(mktemp -d); printf x > $p/plan.txt`, `P=$(mktemp); cat >> $P` with a heredoc, `P=$(mktemp -d)/plan.txt; printf x > $P`, `p=<scratch>/plan.txt; : > $p`, `export P=$TMPDIR/p; printf x > "$P"`, a `for` loop over another name writing `${P}`); and each of these is refused: a resolved path in the tree (`P=$TMPDIR/tree/f.txt; echo x > $P`), a dot-dot (`P=$TMPDIR/../x; echo x > $P/tree/f`), an unassigned variable, a name assigned twice, an appended name, a name bound by `read` or `unset`, an `eval`, `mktemp` with arguments, a `TMPDIR` the command assigns, and `$TMPDIR` when the hook's `TMPDIR` is unset or relative; and the forms the Codex review found: `$TMPDIR_TREE` taken for `$TMPDIR`, a command-prefix assignment, an assignment inside `$(…)`, a subshell or quoted plan text, `IFS= read -r P` and `(read -r P …)`, an expansion before its assignment, a backslash or backquote, and `${TMPDIR}tree/…` when `TMPDIR` ends in a slash. See it RED.
2. [S2] `resolveTargets`. Mutant: resolve a name assigned twice (drop the single-assignment rule) — the twice-assigned case passes and the test goes red. [proof: mutation]
3. [S3] Mutant: drop the rebinding-word check — the `read` case passes and the test goes red. [proof: mutation]
4. [S4] Mutant: resolve `$TMPDIR` to the root instead of its value, so a resolved path is not where the shell writes — the in-tree case (`$TMPDIR/tree/f.txt`) lands outside the tree, passes, and the test goes red. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/bench/ -count=1 -v -run '^(TestAPlanFileAtAResolvablePathOutsideTheTreePassesTheHook|TestAWriteOutsideTheTaskTreePassesTheHook|TestARedirectionToAPathTheHookCannotResolveNamesTheTwoFormsThatPass|TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads|TestAnInterpreterIsJudgedOnItsOwnSegmentAndTheHeredocItReads)$' 2>&1) \
  && grep -qE '^--- PASS: TestAPlanFileAtAResolvablePathOutsideTheTreePassesTheHook \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAWriteOutsideTheTaskTreePassesTheHook \(' <<<"$out" \
  && grep -qE '^--- PASS: TestARedirectionToAPathTheHookCannotResolveNamesTheTwoFormsThatPass \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheArmBHookRefusesAWriteByAnyMeansButMrwAndCountsReads \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAnInterpreterIsJudgedOnItsOwnSegmentAndTheHeredocItReads \(' <<<"$out" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/read internal/apply internal/plan internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPlanFileAtAResolvablePathOutsideTheTreePassesTheHook` | `internal/bench/hook_test.go` | the ablation's refused plan paths pass; a resolved path in the tree, a rebindable name and every unresolvable form are refused | none | S1, S2, S3, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | `Hook` calls `ClassifyBash`, which calls `classify`, for every Bash command configuration B's agent runs |
| 3 — the caller can discover it | the refusal hint names the forms that pass; these now pass too |
| 4 — it is used | no telemetry (ADR-009); the evidence is the refused plan files in about one MRW cell in three across runs 2-5 |

## Mutation Log
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S2: a name assigned twice is resolved · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a rebindable name is not resolved
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S3: a name read or unset is resolved · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a rebindable name is not resolved
- 2026-11-01 · de68a89a* · mutant survived · exit 0 · `internal/bench/hook.go` · S4: a resolved value is not held to plain path characters · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a resolvable plan path outside the tree passes
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S4: $TMPDIR resolves to the root, not its value · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a resolvable plan path outside the tree passes
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S2: a name assigned twice is resolved · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a rebindable name is not resolved
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S3: a name read or unset is resolved · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a rebindable name is not resolved
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S4: $TMPDIR resolves to the root, not its value · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a resolvable plan path outside the tree passes
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S3: a command-prefix assignment is a statement (Codex review) · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a rebindable name is not resolved
- 2026-11-01 · de68a89a* · mutant killed · exit 1 · `internal/bench/hook.go` · S3: an expansion before its assignment reads it (Codex review) · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · covers:a rebindable name is not resolved

## Invariants

- Every refusal ADR-156 records stays a refusal except the forms Decision 1 names.
- Every resolved target goes through `outsideTree`.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0; or a refused form from the ablation cannot be resolved without running the command: stop and bring it to Zy.

## Out of Scope

- Quoted literal targets (permanent: boundary: ADR-186 Alternatives)

## Verification Log
- 2026-11-01 · de68a89a* · exit 1 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:614 · test-lock-sha256:daf363a17ebb433a309fcec66c5b29a338b4e75175c5d8e8d10fe9113c9a5c9b · test-lock-file:docs/adr/locks/daf363a17ebb433a309fcec66c5b29a338b4e75175c5d8e8d10fe9113c9a5c9b.lock
  ```
  ```
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:950
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:965
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:906
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:956
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:758
- 2026-11-01 · de68a89a* · exit 1 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:666
  ```
  ```
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:864
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:987
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:955
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:971
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:1019
- 2026-11-01 · de68a89a* · exit 0 · `set -o pipefail …` · acceptance-sha256:6385262e88277f892b6f7c7117a81f709328004357c6ada092ce903ead33935c · ms:336
