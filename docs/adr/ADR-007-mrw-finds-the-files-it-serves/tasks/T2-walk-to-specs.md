# Task ADR-007-T2: Turn a walk and a pattern into specs

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `read.Walk(root string, paths []string, opt WalkOptions) ([]Spec, []Problem, error)`, `read.WalkOptions`, `read.Problem`
**Consumes:** `rooted.Descendable(absRoot, path string) (bool, error)` (T1)
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
   otherwise walk it, consulting `rooted.Descendable` before descending and
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

       baseline: grep -rl EvalSymlinks . | <compose specs> | mrw read -C 3
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
| `TestWalkDoesNotDescendASymlinkedDirectory` | `internal/read/walk_test.go` | rule 3, and that `Descendable` is consulted | — | S1, S2 |
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
| 2 — something selects it | `TestWalkDoesNotDescendASymlinkedDirectory` fails when the `Descendable` call is removed — that is the mutation proving T1 is reached from here |
| 3 — the caller can discover it | n/a: no declared interface until T3 adds the flag |
| 4 — it is used | nothing measures this yet |

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
