# Task ADR-007-T2: Turn a walk and a pattern into specs

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `read.Walk(root string, paths []string, opt WalkOptions) ([]Spec, []Problem, error)`, `read.WalkOptions`, `read.Problem`
**Consumes:** nothing — this originally consumed `rooted.Descendable` (T1), which was withdrawn on 2026-09-03; see the parent ADR's Amendment
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Produce the `Spec` values a caller would otherwise have composed by hand — walk
the named paths, keep the files that match, deduplicate them — and report every
path that could not be served instead of dropping it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/walk.go` | add | `Walk`, `WalkOptions`, `Problem` — the walk, the `--exclude` filter, the match test, the dedup |
| `internal/read/walk_test.go` | add | its tests |
| `internal/adversarial/filesystem_test.go` | edit | the boundary and observation tests a walk must also satisfy |

`Walk` is selected by `cmd/mrw`'s `--grep` flag, which T3 adds; until then the
only caller is its own test, and T3's Affected Files table carries the line that
makes it reachable.

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): one spec per matching file and
   none for a non-matching one; a `--exclude` glob drops a match and prunes a
   directory; the walk refuses to leave the root and does not descend a
   symlinked directory.
2. [S2] Implement the walk: serve a named path directly if it is a regular file,
   otherwise walk it. A symlinked directory needs no guard: `filepath.WalkDir`
   does not follow one, so it arrives as a non-directory entry and step S3's
   test declines it. Consult `rooted.Resolve` before reading. Skip `.git/`
   `rooted.Resolve` before reading. Skip `.git/` during a walk.
3. [S3] Apply rule 2 — a candidate must RESOLVE to a regular file, asked after
   `rooted.Resolve` so a symlink to a file inside the root is a candidate and
   `mrw read link.txt` keeps working. An `os.Lstat` check would refuse every
   symlink and contradict rule 4; use `os.Stat` on the resolved path. A FIFO,
   device or socket found by walking is skipped silently, and one named
   EXPLICITLY becomes a `Problem` rather than silence.
4. [S4] Apply rule 5 — a refused path, an unreadable file or an unlistable
   directory becomes a `Problem` and the walk CONTINUES; only an unresolvable
   root returns the error.
5. [S5] Apply rule 4 — deduplicate candidates on the cleaned root-relative path
   before returning, so a path named twice, or named while also inside a named
   directory, yields one spec. Two names for one inode stay two specs.
6. [S6] Match by reading each candidate and testing the compiled pattern against
   its lines, returning `Spec{Path, Ranges: []Range{{Re: pattern}}}` for a file
   with at least one match and nothing for one without. `Walk` records NO
   observation: `read.Run`'s read is the authoritative one, and a file that
   stops matching between the walk and the serve prints nothing and observes
   nothing.
7. [S7] Measure the go/no-go cost condition from the ADR. Both halves must see
   the SAME FILES, which is harder than it looks and was got wrong once already:

       baseline: grep -rl --exclude-dir=.git EvalSymlinks . | <compose specs> | mrw read -C 3
       walk:     mrw read --grep EvalSymlinks -C 3 .

   Compare the two MATCH SETS, not a remembered number, and record the count
   observed rather than asserting one. The first draft of this step named a
   fixed 5 and filtered the baseline with `--include='*.go'` while the walk has
   no such filter — and by then the ADR and two task files had written
   `EvalSymlinks` into themselves, so the real counts were 5 against 8. A
   measurement whose corpus a document can change by describing it is not a
   measurement; the sets matching is the precondition, and if they differ that
   is the finding rather than the ratio.

   Record both wall-clock numbers, the match set, the machine and the date in
   the verification log. Over 2× and the ADR says `--grep` is withdrawn. [proof: human: a wall-clock ratio is a claim about one machine and one corpus; a test can pin behaviour but not a number, and the go/no-go needs the number]

   `--exclude-dir=.git` is load-bearing rather than tidy: rule 6 has the walk
   skip `.git/` unconditionally and a plain recursive grep does not, so a
   pattern appearing in a commit message or a reflog puts a file in the
   baseline's set that the walk can never return. Not hypothetical — measured
   2026-09-01 on the authoring machine, `/usr/bin/grep -rl EvalSymlinks .`
   returned 10 files and one of them was `.git/COMMIT_EDITMSG`, put there by the
   commit that wrote this step. Same shape as the corpus problem above, one
   directory further out.

## Acceptance

```bash
set -o pipefail
go test ./internal/read/ -run 'TestWalk' -v 2>&1 | tee /tmp/adr007-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr007-t2.out \
  && go test ./internal/adversarial/ -run 'TestAWalked' -v 2>&1 | tee /tmp/adr007-t2b.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr007-t2b.out \
  && go test ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestWalkReturnsOneSpecPerMatchingFile` | `internal/read/walk_test.go` | the core behaviour | — | S1, S2, S6 |
| `TestWalkSkipsAFileWithNoMatch` | `internal/read/walk_test.go` | an unmatched file is not served and not observed | — | S1, S6 |
| `TestWalkHonoursExcludeGlobsAndPrunesDirectories` | `internal/read/walk_test.go` | the exclusion algorithm | — | S1, S2 |
| `TestWalkSkipsTheGitDirectory` | `internal/read/walk_test.go` | rule 6 | — | S1, S2 |
| `TestWalkCannotLeaveTheRoot` | `internal/read/walk_test.go` | ADR-006 holds on the walk | — | S1, S2 |
| `TestWalkDoesNotDescendASymlinkedDirectory` | `internal/read/walk_test.go` | rule 3 as BEHAVIOUR. ⚠ It cannot fail for any change to mrw's own code — it asserts a property of `filepath.WalkDir`. Kept as a regression test against a future walk that follows symlinks, not as proof of a guard. | — | S1, S2 |
| `TestWalkSkipsADiscoveredFifoAndReportsANamedOne` | `internal/read/walk_test.go` | rule 2, both halves | — | S1, S3 |
| `TestWalkReportsEveryBadPathAndServesTheGoodOnes` | `internal/read/walk_test.go` | rule 5 — a mixed valid/refused/unreadable set | — | S1, S4 |
| `TestWalkDeduplicatesAPathNamedTwiceOrCoveredTwice` | `internal/read/walk_test.go` | rule 4, including a `--max-lines` budget that stays per file | — | S1, S5 |
| `TestWalkKeepsTwoNamesForOneInode` | `internal/read/walk_test.go` | rule 4's second half — a hardlink is two addresses | — | S1, S5 |
| `TestWalkServesASymlinkToAFileInsideTheRoot` | `internal/read/walk_test.go` | rule 2 — resolve then ask, so a served path does not regress | — | S1, S3 |
| `TestAWalkedSpecObservesOnlyWhatItPrinted` | `internal/adversarial/filesystem_test.go` | ADR-005's ledger rule is unchanged by this path | — | S1, S6 |
| `TestAWalkedFileThatStopsMatchingServesAndObservesNothing` | `internal/adversarial/filesystem_test.go` | rule 3 — the scan-to-serve window is visible, not silent | — | S1, S6 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the thirteen tests above |
| 2 — something selects it | Three mutations against `internal/read/walk.go`, each killed by a named test: the `.git` skip (`TestWalkSkipsTheGitDirectory`), the dedup (`TestWalkDeduplicatesAPathNamedTwiceOrCoveredTwice`) and the exclusion filter (`TestWalkHonoursExcludeGlobsAndPrunesDirectories`). The rung originally read "`TestWalkDoesNotDescendASymlinkedDirectory` fails when the `Descendable` call is removed"; that mutation was run on 2026-09-03 and SURVIVED, which is what withdrew T1 |
| 3 — the caller can discover it | n/a: no declared interface until T3 adds the flag |
| 4 — it is used | nothing measures this yet |

## Mutation Findings, 2026-09-03

Three mutants SURVIVED here, and they are the reason T1 was withdrawn. They are recorded as prose
rather than as Mutation Log rows because they were performed and written by hand, carry no
acceptance digest, and `adr-verify` cannot parse them — a row the tool cannot read is not tool-written
provenance, whatever it looks like. The Mutation Log below holds only rows a real run produced.

- **Calling `rooted.Descendable` and ignoring its answer changed nothing.** Every walk test stayed
  green. `filepath.WalkDir` already prevents the descent, so the function had no reachable effect.
- **The same mutation against a REWRITE that consulted `Descendable` for symlinked directories and
  descended when permitted was also green.** Rule 4 deduplicates on the root-relative path, so
  descending a link's real target yields a path already served. The property is over-determined
  three ways.
- **Dropping rule 2's regular-file test entirely was green**, for the same reason: no mutation of
  mrw's own code can kill that test, because the standard library prevents the case first.

Together these say the property was real and the CODE for it was not: `rooted.Descendable` was
built, tested, mutation-logged and deleted unused on 2026-09-03 (`8a73748`). That is what a survived
mutant is for.

## Mutation Log
## Invariants

- `read.Run` is unchanged: `Walk` produces the same `Spec` values a caller can
  type, and nothing about serving them is special-cased.
- `Walk` observes nothing. Every ledger entry on this path comes from `Run`.
- A file with no match contributes nothing to the ledger, because it never
  becomes a spec.
- After dedup, one file yields at most one spec, so `--max-lines` is per file
  for everything the walk produces.

## Risks

- Reading every candidate to match it costs more than `grep -rl` does. This is
  the go/no-go cost condition, measured in S7 — not mitigated, decided.
- A pattern that matches nothing anywhere returns no specs, which `read.Run`
  would then report as nothing at all. The CLI must say so; T3 owns that line.

## Stop Condition

Stop if any go/no-go condition in the ADR fails: a candidate policy beyond
"regular files", a new module dependency, a fourth flag, or a cost over 2× the
baseline. Any of them means `--grep` is withdrawn and `--files-from` is the
whole answer.

## Out of Scope

- The CLI flags — T3's job.
- A cross-file `--max-lines` budget (deferred: docs/adr/BACKLOG.md)
- Holding candidates open between the walk and the serve — rule 3 makes `Run`
  authoritative instead.

## Verification Log

**2026-09-03 · acceptance · exit 0.** `go test ./internal/read/ -run TestWalk -v`
→ 11 tests, all PASS, no "no tests to run";
`go test ./internal/adversarial/ -run TestAWalked -v` → 2 tests, both PASS;
`go test ./...` → all packages ok. Run on darwin/arm64 (Apple M5).

**2026-09-03 · S7, the go/no-go cost condition · PASS.** Both sides were first
verified to select the SAME 12 files, as the step demands — that check is the
precondition, not a formality, and the sets agreed exactly:

    bin/mrw
    docs/adr/ADR-007-mrw-finds-the-files-it-serves.md
    docs/adr/…/tasks/T1-descend-rule.md
    docs/adr/…/tasks/T2-walk-to-specs.md
    internal/apply/apply.go      internal/check/check.go
    internal/read/read.go        internal/read/walk.go
    internal/rooted/rooted.go    internal/rooted/rooted_test.go
    internal/state/state.go      internal/state/state_test.go

Pattern `EvalSymlinks`, best of 5, Apple M5 (darwin/arm64), at `1b88cb9`+:

| | best of 5 |
|---|---|
| `grep -rl --exclude-dir=.git … \| mrw read -C 3 --files-from -` | 38 ms |
| `mrw read --grep EvalSymlinks -C 3 .` | 29 ms |

**0.76× — the walk is FASTER**, against a threshold of 2× over. It spawns no
second process and moves nothing through a pipe, which more than pays for
reading each candidate twice. The count recorded is the count observed: 12, not
the 5 or 8 earlier drafts of this step guessed at.

Other go/no-go conditions: candidate policy is regular-files-only with no
content sniffing ✓; no new module in `go.mod` ✓; flag surface is exactly the
three the ADR names ✓. **`--grep` ships.**
- 2026-09-01 · e56fb3d* · exit 1 · `set -o pipefail …` · acceptance-sha256:cddf8e1e51a197f0e6d42156a8cc6429bb08a2a4f829771249bdd1f0eac0f0bd · ms:139
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read.test]
  internal/read/walk_test.go:38:97: undefined: Problem
  internal/read/walk_test.go:44:23: undefined: Walk
  internal/read/walk_test.go:44:38: undefined: WalkOptions
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read [build failed]
  FAIL
  ```
- 2026-09-03 · 8aaafc3 · exit 0 · `set -o pipefail …` · acceptance-sha256:cddf8e1e51a197f0e6d42156a8cc6429bb08a2a4f829771249bdd1f0eac0f0bd · ms:487
