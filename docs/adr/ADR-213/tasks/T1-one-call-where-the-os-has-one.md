# Task ADR-213-T1: one call where the OS has one

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `internal/apply/noreplace_linux.go`, `noreplace_darwin.go`, `noreplace_other.go`; `renameNew` trying them first; `golang.org/x/sys` in `go.mod` and `go.sum`; the `CONTRIBUTING.md` row; two `EXEMPT` entries in `scripts/fence-prose.py`; contract §375
**Consumes:** `tree.renameNew`, `finishLink`, `checkThenRename`, `respelling`, the seams `beforeRemoveSource`, `linkFn`, `fallbackLstatFn`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a free destination is moved by one call`, `a taken destination refuses and keeps the source`, `a destination swapped in at the old window is refused, not lost`, `a respelling still renames`, `an unsupported flag falls back to #744`, `the parents are opened through the root`, `the link path is not used when the call succeeds`, `Windows and the other systems are unchanged`, `every named subtest passed and none skipped`, `the requirement and the exemptions are in place`, `the contract row exists`, `the other engine packages are untouched`

## Goal

ADR-213 Decisions 1 to 6. **`TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne`** (`internal/apply`), one subtest per row, through `tree.noReplace().rename` with a seam recording every system call the move makes (`noReplaceFn`, `linkFn`, `fallbackLstatFn`):

| Subtest | Platforms | Setup | Outcome |
|---------|-----------|-------|---------|
| `free_destination` | linux, darwin | `a` exists, `b` does not | one no-replace call; no `linkFn`; `b` holds `a`'s file (same identity), `a` gone |
| `taken_destination` | linux, darwin | `b` exists | an error satisfying `fs.ErrExist`; `a` and `b` unchanged; no `linkFn` |
| `swapped_in_at_the_old_window` | linux, darwin | the seam makes `b` just before the call | refused, `a` unchanged — the case #744's `beforeRemoveSource` seam loses on the link path |
| `respelling` | darwin (APFS folds case) | `a.txt` to `A.txt` | renamed, one entry `A.txt` |
| `unsupported_flag` | linux, darwin | the call seam answers `EINVAL` (then `ENOSYS`, `ENOTSUP`, `EOPNOTSUPP` in turn) | `linkFn` used, the move lands as #744's does |
| `parent_outside_the_root` | linux, darwin | the destination's parent is a link leading outside the root | refused by the root, nothing moved |
| `directory` | linux, darwin | a directory `d` to a free `e`; then to a taken `f` | moved by one call; then refused, `d` unchanged |
| `link_path_elsewhere` | windows, other | `a` to a free `b` | `linkFn` used, as #744 |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/tree.go` | edit | `renameNew` tries `noReplaceFn` first |
| `internal/apply/noreplace_linux.go`, `noreplace_darwin.go`, `noreplace_other.go` | new | the call per OS behind `noReplaceFn` |
| `internal/apply/noreplace213_test.go` | new | the Goal's table |
| `go.mod`, `go.sum` | edit | `golang.org/x/sys` |
| `CONTRIBUTING.md` | edit | Prerequisites: two dependencies, why the second |
| `scripts/fence-prose.py` | edit | `EXEMPT`: ADR-010 T1's and T2's `grep -q '^require github.com/urfave/cli/v3 ' go.mod`, freed by ADR-213 |
| `scripts/contract.sh` | edit | §375 |

## Ordered Steps

1. [S1] The test; RED on every linux/darwin row but `unsupported_flag` and `respelling`, which hold today through #744's path. [proof: acceptance]
2. [S2] The calls and the fallback. Mutants, each killed: the call skipped (`free_destination` sees `linkFn`); `EEXIST` read as unsupported (`taken_destination`: `linkFn` is called); the fallback list emptied (`unsupported_flag`); the parents opened by path, not through the root (`parent_outside_the_root`); the respelling branch dropped (`respelling`). [proof: mutation]
3. [S3] The requirement, `CONTRIBUTING.md`, the two exemptions; `./scripts/static.sh` clean, `govulncheck` included. Contract §375: under `strace -f -e trace=link,linkat,rename,renameat,renameat2` where present (a visible skip otherwise), a `create` and a `rename` show `renameat2` with `RENAME_NOREPLACE` and no `link`/`linkat`; paired with a destination made beforehand, which refuses the move with the source unchanged. [proof: acceptance]

## Acceptance

```bash
need() { f=$1; t=$2; shift 2; grep -q "^--- PASS: $t (" "$f" || return 1; for s in "$@"; do grep -q "^    --- PASS: $t/$s (" "$f" || return 1; done; }
d="$(mktemp -d)" \
  && go test -count=1 -v ./internal/apply -run '^TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne$' > "$d/apply" \
  && case "$(go env GOOS)" in darwin) need "$d/apply" TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne free_destination taken_destination swapped_in_at_the_old_window respelling unsupported_flag parent_outside_the_root directory ;; linux) need "$d/apply" TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne free_destination taken_destination swapped_in_at_the_old_window unsupported_flag parent_outside_the_root directory ;; *) need "$d/apply" TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne link_path_elsewhere ;; esac \
  && ! grep -q -- '--- SKIP' "$d/apply" \
  && grep -qE '^[[:space:]]*golang\.org/x/sys v' go.mod \
  && grep -qF 'ADR-213' scripts/fence-prose.py \
  && grep -qF 'golang.org/x/sys' CONTRIBUTING.md \
  && s375="$(awk '/^# [0-9]+\. /{on = /^# 375\. /} on' scripts/contract.sh)" \
  && grep -q '^# 375\. ' <<<"$s375" \
  && grep -qF 'RENAME_NOREPLACE' <<<"$s375" \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" \
  && [ -z "$st" ]
```

What each clause holds: the test passes in its package with every row its platform names and none skipped (rows for another platform are not registered, never skipped); the requirement, the exemption and the prerequisite are in their files; §375's text, read up to its next heading, names the flag (that it runs is the contract gate's, on Linux CI). The engine clauses hold the five engine packages this record does not own to no committed non-test change since the merge base and no uncommitted or untracked file there (test files included in the second; ignored files in neither), and a failing `git status` fails the chain. `go.mod` is not in them: this record changes it.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne` | `internal/apply/noreplace213_test.go` | the Goal's table, by the recorded calls, the identities and the tree | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §375 |
| 2 — something selects it | every create, path-op rename and undo that must not replace, on Linux and macOS |
| 3 — the caller can discover it | nothing changes for a caller; `CONTRIBUTING.md` names the dependency |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is the window #744's own comment names |

## Mutation Log

## Verification Log

## Invariants

- Receipts, refusals and exit codes are unchanged.
- Windows and the other systems take #744's path.

## Risks

- A filesystem may answer the flag with an error not in the fallback list; T1 adds one only with a reproduction.
- The 205 done fences that count one requirement fail if re-run; the record's Decision 6 says why they stay as they are.

## Stop Condition

The fence exits 0; `./scripts/static.sh` (with `govulncheck`) and `./scripts/contract.sh` exit 0 on Linux CI; the Go tests pass on Linux, macOS and Windows; `adr-debt` reports nothing broken or unreceipted; Zy has accepted the record.

## Out of Scope

- Windows' no-replace rename (permanent: boundary: the record's Out of Scope)
