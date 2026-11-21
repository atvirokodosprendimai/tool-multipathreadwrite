# Task ADR-213-T1: one call where the OS has one

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `internal/apply/noreplace_linux.go`, `noreplace_darwin.go`, `noreplace_other.go` behind the seam `noReplaceFn`; `renameNew` trying it first; `golang.org/x/sys` in `go.mod` and `go.sum`; the licence notice in the release archives; `govulncheck` for darwin; the `CONTRIBUTING.md` row; two `EXEMPT` entries in `scripts/fence-prose.py`; contract §375 and §370's amendment
**Consumes:** `tree.renameNew`, `finishLink`, `checkThenRename`, `respelling`, the seams `linkFn`, `fallbackLstatFn`; `rooted`'s directory-only open
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a free destination is moved by the real call`, `a taken destination is refused by the real call with identities intact and no fallback`, `a cross-directory move is one call`, `a directory moves and is refused the same way`, `the fallback list covers only the call's own errors`, `a parent that is a FIFO fails at once`, `respelling stays ADR-129's`, `the #744 tests hold on the forced fallback`, `Darwin's refused rename makes no second name`, `the module requirements are exactly two`, `every named subtest passed and none skipped`, `the contract rows exist`, `the other engine packages are untouched`

## Goal

ADR-213 Decisions 1 to 7.

**`TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne`** (`internal/apply`), run on Linux and macOS natively (CI's macOS job included). One subtest per row, through `tree.noReplace().rename`. The seam `noReplaceFn` delegates to the real system call and records its arguments and its errno; `linkFn` and `fallbackLstatFn` record whether they were called:

| Subtest | Platforms | Setup | Outcome |
|---------|-----------|-------|---------|
| `free_destination` | linux, darwin | `a` exists, `b` does not | the real call returned nil; no `linkFn`; `b` is `a`'s file by identity, `a` gone |
| `taken_destination` | linux, darwin | `b` exists | the real call returned `EEXIST`; an error satisfying `fs.ErrExist`; `a` and `b` keep their identities and bytes; no `linkFn`, no `fallbackLstatFn` |
| `cross_directory` | linux, darwin | `d1/a` to a free `d2/b`, then `d1/c` to a taken `d2/b` | one call each, the first done, the second `EEXIST` with both files intact |
| `directory` | linux, darwin | a directory `d` to a free `e`; then to a taken `f` | one call; then `ENOTEMPTY` or `EEXIST`, `d` intact |
| `fallback_errnos` | linux, darwin | the seam answers `EINVAL`, `ENOSYS`, `ENOTSUP`, `EOPNOTSUPP` in turn without calling the OS | `linkFn` used, the move lands as #744's does |
| `exdev_is_the_moves` | linux, darwin | the seam answers `EXDEV` | the move fails with it; no `linkFn` |
| `parent_open_error_is_the_moves` | linux, darwin | the destination's parent is replaced by a FIFO | the open fails at once (no hang, a deadline of 5 s), the move fails, no call, no `linkFn` |
| `respelling_exact_and_identity` | darwin | `a.txt` to `A.txt`, the same entry | renamed; one entry `A.txt`; ADR-129's checks ran first |
| `respelling_linux_eexist` | linux | the seam answers `EEXIST` for a respelling that `respelling` judged one entry | the plain-rename branch renames it |
| `link_path_elsewhere` | windows, other | `a` to a free `b` | `linkFn` used, as #744 |

**#744's tests on the forced fallback**: `review744_test.go`, `review744b_test.go`, `move744_test.go` and `extraname744_darwin_test.go` set `noReplaceFn` to answer `ENOTSUP`, so they keep driving the link path they were written for. **§370's Go twin** (`TestAMoveThatCannotRemoveEitherNameNamesTheExtraOneAsLeftBehind`) keeps its expectation on the forced fallback and gains a sibling on the real call: the rename is refused, there is no second name, `left_behind` is empty.

**The module requirements**: `go mod edit -json` lists exactly `github.com/urfave/cli/v3` and `golang.org/x/sys`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/tree.go` | edit | `renameNew` tries `noReplaceFn` first |
| `internal/apply/noreplace_linux.go`, `noreplace_darwin.go`, `noreplace_other.go` | new | the call per OS, parents opened through the root with `O_RDONLY\|O_DIRECTORY\|O_NONBLOCK` |
| `internal/apply/noreplace213_test.go` | new | the Goal's table |
| `internal/apply/review744_test.go`, `review744b_test.go`, `move744_test.go`, `extraname744_darwin_test.go` | edit | forced onto the fallback; the Darwin sibling |
| `go.mod`, `go.sum` | edit | `golang.org/x/sys` |
| `.github/workflows/ci.yml`, `THIRD_PARTY_NOTICES` | edit | `x/sys`'s BSD-3-Clause notice goes in `THIRD_PARTY_NOTICES`, which the release archives carry and CI's notice check compares with the modules compiled in (#762); the Linux contract job installs `strace` for §375 |
| `scripts/static.sh` | edit | `govulncheck` also with `GOOS=darwin` |
| `CONTRIBUTING.md` | edit | Prerequisites: two dependencies, why the second |
| `scripts/fence-prose.py` | edit | `EXEMPT`: ADR-010 T1's and T2's `grep -q '^require github.com/urfave/cli/v3 ' go.mod`, freed by ADR-213 |
| `scripts/contract.sh` | edit | §375; §370 |

## Ordered Steps

1. [S1] The test; RED on every linux/darwin row but `fallback_errnos`, `parent_open_error_is_the_moves` and `respelling_exact_and_identity`, which hold today through #744's path; #744's tests forced onto the fallback stay green. [proof: acceptance]
2. [S2] The calls and the fallback. Mutants, each killed by a named failure: the call skipped (`free_destination`: `linkFn` called); `EEXIST` sent to the fallback (`taken_destination`: `linkFn` called); `EXDEV` sent to the fallback (`exdev_is_the_moves`); the parents opened without `O_NONBLOCK` (`parent_open_error_is_the_moves`: its deadline); the fallback list applied to the open's errors (`parent_open_error_is_the_moves`: `linkFn` called); the respelling branch dropped (`respelling_linux_eexist`). [proof: mutation]
3. [S3] The requirement, the licence notice, `govulncheck` for darwin, `CONTRIBUTING.md`, the two exemptions; `./scripts/static.sh` clean. Contract §375 on the Linux runner that installs `strace`, required there (a skip only where the job says it has no `strace`, which fails the Stop Condition): `strace -f -y -e trace=link,linkat,renameat,renameat2` over a create and a rename of fixture files, matching their paths, shows `renameat2(…f375…, RENAME_NOREPLACE) = 0` and no `link`/`linkat`; paired with a taken destination made beforehand, whose trace shows `renameat2(…) = -1 EEXIST`, the source unchanged. §370 amended as Decision 7. [proof: acceptance]

## Acceptance

```bash
need() { f=$1; t=$2; shift 2; grep -q "^--- PASS: $t (" "$f" || return 1; for s in "$@"; do grep -q "^    --- PASS: $t/$s (" "$f" || return 1; done; }
d="$(mktemp -d)" \
  && go test -count=1 -v ./internal/apply -run '^TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne$' > "$d/apply" \
  && case "$(go env GOOS)" in darwin) need "$d/apply" TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne free_destination taken_destination cross_directory directory fallback_errnos exdev_is_the_moves parent_open_error_is_the_moves respelling_exact_and_identity ;; linux) need "$d/apply" TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne free_destination taken_destination cross_directory directory fallback_errnos exdev_is_the_moves parent_open_error_is_the_moves respelling_linux_eexist ;; *) need "$d/apply" TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne link_path_elsewhere ;; esac \
  && ! grep -q -- '--- SKIP' "$d/apply" \
  && go test -count=1 ./internal/apply -run '744' > "$d/f744" \
  && go mod edit -json > "$d/mod.json" \
  && python3 -I -c 'import json,sys; r=sorted(x["Path"] for x in json.load(open(sys.argv[1]))["Require"]); sys.exit(r!=["github.com/urfave/cli/v3","golang.org/x/sys"])' "$d/mod.json" \
  && grep -qF 'ADR-213' scripts/fence-prose.py \
  && grep -qF 'golang.org/x/sys' CONTRIBUTING.md \
  && grep -qF 'GOOS=darwin' scripts/static.sh \
  && s375="$(awk '/^# [0-9]+\. /{on = /^# 375\. /} on' scripts/contract.sh)" \
  && grep -q '^# 375\. ' <<<"$s375" \
  && grep -qF 'RENAME_NOREPLACE) = 0' <<<"$s375" \
  && grep -qF 'EEXIST' <<<"$s375" \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && git diff --quiet "$base" -- internal/plan internal/read internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/seen internal/check internal/state)" \
  && [ -z "$st" ]
```

What each clause holds: the test passes in its package with every row its platform names and none skipped (rows for another platform are not registered); #744's tests pass on the forced fallback; the parsed requirements are exactly the two modules; the exemption, the prerequisite and the darwin `govulncheck` are in their files; §375's text, read up to its next heading, matches the call's success and its `EEXIST` (that it runs is the contract gate's, required on the Linux runner). The engine clauses hold the five engine packages this record does not own to no committed non-test change since the merge base and no uncommitted or untracked file there (test files included in the second; ignored files in neither), and a failing `git status` fails the chain. `go.mod` is not in them: this record changes it. The fence runs natively on Linux and macOS; a host it does not run on proves only `link_path_elsewhere`.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARenameThatMustNotReplaceIsOneSystemCallWhereTheOSHasOne` | `internal/apply/noreplace213_test.go` | the Goal's table, by the real call's result, the recorded calls, the identities and the tree | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §375 |
| 2 — something selects it | every create, path-op rename and undo that must not replace, on Linux and macOS |
| 3 — the caller can discover it | `left_behind` on Darwin after a refused rename; `CONTRIBUTING.md` names the dependency |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is the window #744's own comment names |

## Mutation Log

## Verification Log

## Invariants

- Refusals and exit codes are unchanged; the one receipt change is Decision 7's.
- Windows and the other systems take #744's path.

## Risks

- A filesystem may answer the flag with an error not in the fallback list; T1 adds one only with a reproduction.
- The done fences in the record's Context table read differently on a fresh run; the record's Decision 6 says how.

## Stop Condition

The fence exits 0 on Linux and on macOS; `./scripts/static.sh` (with both `govulncheck` runs) and `./scripts/contract.sh` exit 0 on Linux CI with §375 run, not skipped; the Go tests pass on Linux, macOS and Windows; the release archives carry the notice; `adr-debt` reports nothing broken or unreceipted; Zy has accepted the record.

## Out of Scope

- Windows' no-replace rename (permanent: boundary: the record's Out of Scope)
