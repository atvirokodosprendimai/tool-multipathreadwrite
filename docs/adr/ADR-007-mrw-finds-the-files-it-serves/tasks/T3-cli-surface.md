# Task ADR-007-T3: Make the walk reachable, and ship the runner-up

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `mrw read --grep`, `--exclude`, `--files-from`
**Consumes:** `read.Walk`, `read.WalkOptions`, `read.Problem` (T2)
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Wire the walk to the command line with the precedence the ADR fixes, and ship
`--files-from` in the same breath so any searcher composes with mrw in one call
whether or not `--grep` survives its go/no-go.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the three flags, the precedence table, the "no file matched" line, and the call to `read.Walk` — this is what SELECTS the walk |
| `cmd/mrw/planpath_test.go` | edit | CLI-level tests, possible here because #4 unwired the framework's exit handler |
| `README.md` | edit | the flags, the precedence table, the exclusion algorithm, one worked example, and the go/no-go cost measurement from T2 |
| `scripts/contract.sh` | edit | section 15 — every flag and every usage error driven through the real binary |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): `--grep` with no paths walks
   `--root` while no-`--grep` with no paths still reads the working set;
   `--files-from -` reads specs from stdin; a pattern matching nothing reports
   that rather than printing nothing.
2. [S2] Add `--grep PATTERN`, `--exclude GLOB` (repeatable) and `--files-from
   FILE|-`, and call `read.Walk` when `--grep` is given.
3. [S3] Implement the precedence table exactly as the ADR states it, including
   every usage error: a positional spec carrying its own range alongside
   `--grep`, `--grep` with `--files-from`, `--files-from` with positional paths,
   `--exclude` without `--grep`, and a glob `path.Match` rejects.
4. [S4] Render `read.Problem` values as output lines and count them as problems,
   so a refused or unreadable path reaches the caller with its reason and the
   exit status the README's table already documents.
5. [S5] Report a pattern that matched no file, naming the pattern — silence is
   the one output this project refuses.
6. [S6] Document in the README: the three flags, the precedence table, the
   exclusion algorithm including the basename rule and the separator example,
   a worked example of the pipeline, and T2's go/no-go measurement with its
   date and machine. Correct the README's flag table row in the same step — the
   row reading `` `--max-lines N` | cap per file `` says it caps "per file",
   which rule 4 shows is false for two
   hand-written specs naming one file; it caps per SPEC, and the walk
   deduplicates so it is per file for everything the walk produces. [proof: acceptance]
7. [S7] Add contract section 15 driving each flag, each usage error, and the
   empty-result case. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -run 'TestGrep|TestFilesFrom|TestExclude|TestNoArguments|TestTheDocumented' -v 2>&1 | tee /tmp/adr007-t3.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr007-t3.out \
  && grep -q '^# 28\.' scripts/contract.sh \
  && grep -q '\-\-files-from' README.md && grep -q '\-\-exclude' README.md \
  && grep -qE 'measured 2026-[0-9]{2}-[0-9]{2}' README.md \
  && ! grep -q 'cap per file' README.md \
  && go test ./... && ./scripts/contract.sh
```

The three `grep` clauses are not decoration: S6 and S7 are proved by acceptance,
and without them the fence passes with the README untouched and no new contract
rows — an unchanged `contract.sh` exits 0 on its own.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestGrepWithNoPathsWalksTheRoot` | `cmd/mrw/planpath_test.go` | the default path set | — | S1, S2 |
| `TestNoArgumentsWithoutGrepStillReadsTheWorkingSet` | `cmd/mrw/planpath_test.go` | the existing behaviour is untouched | — | S1, S2 |
| `TestGrepReportsAPatternThatMatchedNoFile` | `cmd/mrw/planpath_test.go` | S5 — the empty result is not silence | — | S1, S5 |
| `TestGrepReportsARefusedPathAndServesTheRest` | `cmd/mrw/planpath_test.go` | S4 — problems reach the caller | — | S1, S4 |
| `TestFilesFromReadsSpecsFromStdin` | `cmd/mrw/planpath_test.go` | the runner-up ships | — | S1, S2 |
| `TestExcludeDropsAMatchingFileAndPrunesADirectory` | `cmd/mrw/planpath_test.go` | the exclusion algorithm at the CLI | — | S1, S2 |
| `TestTheDocumentedUsageErrorsAreErrors` | `cmd/mrw/planpath_test.go` | S3 — every row of the precedence table that says "usage error" | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the seven tests above |
| 2 — something selects it | `cmd/mrw/main.go`'s flag registration; deleting the `--grep` case makes `TestGrepWithNoPathsWalksTheRoot` fail |
| 3 — the caller can discover it | `mrw read --help` lists all three, the README carries the precedence table, and `scripts/contract.sh` §15 drives them through the real binary — the acceptance fence asserts both documents changed |
| 4 — it is used | nothing measures this yet |

## Mutation Log

Two corrections to the fence above, both found by running it on 2026-09-03:

- It said `grep -q '^# 15\.' scripts/contract.sh`. **Section 15 has existed
  since the `check` work**, so that clause was satisfied by an UNTOUCHED
  `contract.sh` from the moment this task was written — the exact failure the
  note under the fence warns about, in the fence itself. The new rows are §28
  and the clause now names it.
- The `-run` regex covered five of this task's seven named tests;
  `TestNoArgumentsWithoutGrepStillReadsTheWorkingSet` and
  `TestTheDocumentedUsageErrorsAreErrors` matched nothing and would have been
  reported as passing by never running. Both are now in the pattern.

- 2026-09-03 · `1b88cb9`+ · mutant killed · exit 1 · `cmd/mrw/main.go` · `case pattern != "":` → `case false:`, so `--grep` never selects the walk · killed by `TestGrepWithNoPathsWalksTheRoot`
- 2026-09-03 · `1b88cb9`+ · mutant killed · exit 1 · `cmd/mrw/main.go` · remove the empty-result report, restoring silence for a pattern that matched nothing · killed by `TestGrepReportsAPatternThatMatchedNoFile`
- 2026-09-03 · `1b88cb9`+ · mutant killed · exit 1 · `cmd/mrw/main.go` · never render `read.Problem` values · killed by `TestGrepReportsARefusedPathAndServesTheRest`
- 2026-09-03 · `1b88cb9`+ · mutant killed · exit 1 · `cmd/mrw/main.go` · pass `Exclude: nil` to `read.Walk`, dropping every `--exclude` on the floor · killed by `TestExcludeDropsAMatchingFileAndPrunesADirectory`
- 2026-09-03 · `1b88cb9`+ · mutant killed · exit 1 · `cmd/mrw/main.go` · ignore `--files-from` · killed by `TestFilesFromReadsSpecsFromStdin`

## Invariants

- `mrw read <specs...>` without `--grep` behaves exactly as it does today,
  including the no-argument working-set read.
- The ledger records what was printed, on this path as on every other.
- Exit 1 keeps its meaning: a pattern matching no file, or a path refused during
  a walk, is incomplete — the same class as the four causes the README lists.

## Risks

- Three flags at once is the widest surface this ADR adds. Mitigated by shipping
  `--files-from` deliberately as the runner-up: if `--grep` is withdrawn by a
  go/no-go condition, `--files-from` stands alone and the pipeline still works.
- The precedence table is where a caller's mental model breaks. Mitigated by
  `TestTheDocumentedUsageErrorsAreErrors`, which fails if any documented usage
  error stops being one.

## Stop Condition

Stop if `--grep` and `--files-from` need different answers to "what did the
caller see" — the ledger rule must be one rule, and two would mean this ADR has
the wrong shape. Stop also if T2 recorded a failing go/no-go condition: then
this task ships `--files-from` alone.

## Out of Scope

- Anything a searcher does beyond naming matching ranges — see the parent ADR's
  Out of Scope.

## Verification Log

**2026-09-03 · acceptance · exit 0**, with the fence corrected as recorded
above. `go test ./cmd/mrw/ -run 'TestGrep|TestFilesFrom|TestExclude|TestNoArguments|TestTheDocumented' -v`
→ 7 tests, all PASS, no "no tests to run"; `go test ./...` → ok;
`./scripts/contract.sh` → **235 PASS, 0 FAIL, exit 0** (216 before this task, so
19 new rows in §28); `gofmt -l .` silent; `go vet ./...` clean;
`go test -race ./...` exit 0. Darwin/arm64, Apple M5.

The five mutations above were each run against the built binary's own package
and each killed by exactly one named test, so rung 2 is met by every flag this
task adds rather than by the flag registration alone.

**Driven through the real binary** as well as through Go tests, because a flag
surface is met through a shell: `--grep` with and without paths, `--exclude`
pruning a directory and dropping a basename match, `--files-from -` fed from a
`grep -rl` pipe and from a list carrying `#` comments, a pattern matching
nothing (exit 1, names the pattern), a refused path beside a served one, and
all six documented usage errors (exit 2 each).

One behaviour worth stating because it surprised the author: `mrw read --grep X .`
in this repository serves `bin/mrw`, a 7 MB build artifact, because a regular
file is a candidate and the walk does not read `.gitignore`. Both are permanent
ADR decisions and `--exclude bin` is the answer, but the README example now
carries an `--exclude` for that reason rather than for decoration.
- 2026-09-03 · 9e132f9 · exit 0 · `set -o pipefail …` · acceptance-sha256:2ba24b7fa5c028b6f9500cc9921e59a839eeb4c088d49b1edcc31e00a8d842e9 · ms:7843
