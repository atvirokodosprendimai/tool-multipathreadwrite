# Task ADR-215-T1: a rooted plan path is refused

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the preflight and the message in `internal/apply`; one sentence in a splitter error; the tests below; the two old assertions amended
**Consumes:** `rooted.IsRooted`, `rooted.Contains`, apply's validation entry, the rename-destination site, working-set pointer expansion
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `every op and a rename destination refuse a rooted path before any lookup`, `the preflight orders before the directory rule, the rename source and the collision maps`, `a message names a spelling only when it is clear`, `an expanded working-set pointer is refused`, `the plan makers still emit relative paths`, `the Windows spellings are refused by a real apply`, `a refused plan writes nothing and its siblings skip`, `the packages this record does not own are untouched`, `every named test passed and none skipped`

## Goal

ADR-215 Decisions 1 to 6 and 9.

- **`TestEveryPlanOpRefusesARootedPath`** (`internal/apply`), fixtures that reach the branch:
  - for replace, insert-before, insert-after, delete, unlink, rename (source), a file `dd/zz.txt` that exists, licensed, and the path `/dd/zz.txt`; the absolute path of a file inside the root; the absolute path of one outside it;
  - create at `/new/x` where `ROOT/new` does not exist and the nested target is missing (no directory is made); create at an absolute path inside the root;
  - a nested target that EXISTS and is licensed: `ROOT/<abs of root>/x` with the plan path `<abs of root>/x` (it applied before; it is refused now);
  - rename of a licensed `src.txt` to `/new/x` (the refusal is the rooted one, not "has not been read" and not a loaded source) and the same with an unread source;
  - `"/dd/x/"` (the rooted refusal, not the directory rule); a case pair `/new/x` and `/NEW/x`;
  - each with an independently valid sibling hunk on another file: the rooted hunk is `failed`, the sibling `skipped`, the tree byte-identical and no directory made; a rooted hunk beside an unrelated directory hunk reports exit-1 outcomes (no `loadErr`).
  - the message holds `is absolute, and every path in a plan is relative to the root`, holds a concrete spelling only for the inside-root absolute path, and names none for `/etc/hosts`, `/../x`, `/`; a spelling starting with `@` is `./@…`, one with a space is double-quoted.
- **`TestAWindowsRootedPlanPathIsRefused`** (`internal/apply`, `rooted215_windows_test.go`, build-tagged `windows`): a real `Apply` of `c:x`, `\x`, `C:/x` and a forward-slash UNC path on a Windows build refuses each with the sentence, and relative `nul.txt`, `b.txt::$DATA` and `j/x` under an out-of-root junction still meet their existing refusals. It runs on the Windows CI shard; on Linux the fence only compiles it (`GOOS=windows go test -c`).
- **`TestAnExpandedWorkingSetPointerThatIsRootedIsRefused`** (`cmd/mrw`): after `mrw iter add /dd/zz.txt:1`, `@@ @1 1 replace`, `--create @1` and an `apply_patch` `Add File: @1` are each refused with the pointer sentence; `--sub` and `--rewrite` over an inside-root absolute argument emit relative hunks and apply.
- **`TestAHeaderWhoseFirstTokenBeginsWithASlashSaysSo`** (`internal/plan`): `@@ /zz.txt 1 replace` and `/dd` keep exit-2 parse errors and now hold the sentence; `@@ /a b/c.txt 1 replace`, `@@ "/a"b/c.txt` forms, `@@ dir/x /a\/b/ replace anchor="x/y"`, `@@ dir/x /a/,/b/ replace`, `@@ @1 /re/ replace`, an unquoted `anchor=prefix /a  b/` keep their parse and the double space; a counted body line literally `@@ /zz.txt 1 replace` under `body=1` is written.
- **`TestMrwWriteRefusesARootedPathLikeTheCLI`** (`internal/mcp`): `mrw_write` of the matrix gives `isError` and the structured receipt: the rooted row `failed` with the sentence, the valid sibling `skipped`.
- **A CLI case** in the apply test's sibling or `cmd/mrw`: a piped `@@ /dd/zz.txt 1 replace` exits 1 and the plan is kept and named by `refused_plan`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the preflight at the start of validation, after pointer expansion, before the grouping and collision maps |
| `internal/apply/pathop.go` | edit | the rename destination uses the same sentence; the not-exist wording loses the rooted branch |
| `internal/plan/plan.go` | edit | one sentence on the unterminated-pattern error for a first token |
| `internal/apply/reason198_test.go` | edit | lines 305-308: the rename destination and the unlink of `/nothere.txt` no longer say `mrw looked for` |
| `scripts/contract.sh` | edit | §32 stops requiring the doubled path `$R$R/sub/real.go`; it asserts the sentence and the tree |
| `internal/apply/rooted215_test.go`, `internal/apply/rooted215_windows_test.go`, `internal/plan/rooted215_test.go`, `internal/mcp/rooted215_test.go`, `cmd/mrw/rooted215_test.go` | new | the tests |

## Ordered Steps

1. [S1] The tests. RED on `main`: `/dd/zz.txt` applies, the nested target applies, the pointer applies, the sentence is absent from the header error. The Windows test is red on the Windows shard. [proof: acceptance]
2. [S2] The preflight, the message, the sentence, then the two amended assertions (§32, `reason198_test.go`). First list every caller of `resolve` and every pass that reaches a path, and say in the commit which the preflight precedes. Mutants, each killed by a named fixture: the preflight skips create; it skips the rename destination; it tests the resolved path instead of the raw one (the nested-target-exists fixture); it runs after the directory rule (`"/dd/x/"`); the message suggests a stripped spelling for `/etc/hosts`; a sibling is `ok` instead of `skipped`; the pointer is not refused. Then `adr-lint` on every record whose test lock names a file this task edits. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && env go test -count=1 -v ./internal/apply -run '^TestEveryPlanOpRefusesARootedPath$' > "$d/apply" \
  && env go test -count=1 -v ./internal/plan -run '^TestAHeaderWhoseFirstTokenBeginsWithASlashSaysSo$' > "$d/plan" \
  && env go test -count=1 -v ./internal/mcp -run '^TestMrwWriteRefusesARootedPathLikeTheCLI$' > "$d/mcp" \
  && env go test -count=1 -v ./cmd/mrw -run '^TestAnExpandedWorkingSetPointerThatIsRootedIsRefused$' > "$d/cmd" \
  && grep -q '^--- PASS: TestEveryPlanOpRefusesARootedPath (' "$d/apply" \
  && grep -q '^--- PASS: TestAHeaderWhoseFirstTokenBeginsWithASlashSaysSo (' "$d/plan" \
  && grep -q '^--- PASS: TestMrwWriteRefusesARootedPathLikeTheCLI (' "$d/mcp" \
  && grep -q '^--- PASS: TestAnExpandedWorkingSetPointerThatIsRootedIsRefused (' "$d/cmd" \
  && ! grep -q -- '--- SKIP' "$d/apply" "$d/plan" "$d/mcp" "$d/cmd" \
  && GOOS=windows go test -c -o "$d/apply-windows.test" ./internal/apply \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- go.mod go.sum internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/read internal/seen internal/check internal/state internal/writer internal/rooted internal/ingest)" \
  && [ -z "$st" ]
```

What each clause holds. The four named tests exist, pass and none is skipped; the Windows test file compiles for Windows here and runs on the Windows CI shard, which the Stop Condition names. The packages the record does not own have no committed non-test change since the merge base and no uncommitted one (this is the unowned-engine guard `adr.md` asks for: `internal/apply`, `internal/plan`, `internal/guide` are owned).

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryPlanOpRefusesARootedPath` | `internal/apply/rooted215_test.go` | every op, the nested target, the rename order, the directory rule, the case pair, siblings, the tree, the message's spelling rules | none | S1, S2 |
| `TestAWindowsRootedPlanPathIsRefused` | `internal/apply/rooted215_windows_test.go` | a real Windows apply of `c:x`, `\x`, `C:/x`, UNC; the relative controls | none | S1, S2 |
| `TestAnExpandedWorkingSetPointerThatIsRootedIsRefused` | `cmd/mrw/rooted215_test.go` | the three pointer makers, `--sub` and `--rewrite` positives, the kept plan at exit 1 | none | S1, S2 |
| `TestAHeaderWhoseFirstTokenBeginsWithASlashSaysSo` | `internal/plan/rooted215_test.go` | the sentence, and every header form that must not change | none | S1, S2 |
| `TestMrwWriteRefusesARootedPathLikeTheCLI` | `internal/mcp/rooted215_test.go` | `isError`, the structured failed and skipped rows | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests |
| 2 — something selects it | every plan hunk and every expanded pointer |
| 3 — the caller can discover it | the refusal; T2's words; the release note |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #648 and the counts in the record |

## Mutation Log

## Verification Log

## Invariants

- No receipt key is added; a refused plan writes nothing; no exit code of the table changes.

## Risks

- Editing a test file that another record locks moves its lock; S2 lints and relocks what it names.

## Stop Condition

The fence exits 0; the Windows CI shard passes `TestAWindowsRootedPlanPathIsRefused`; `./scripts/contract.sh` exits 0 on Linux CI with §32 amended; every gate in `AGENTS.md` passes; the engine diff names ADR-215 as owner and has a `go-reviewer` pass; the implementation PR and the tag message carry Decision 9; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- The words and the contract row (deferred: T2)
