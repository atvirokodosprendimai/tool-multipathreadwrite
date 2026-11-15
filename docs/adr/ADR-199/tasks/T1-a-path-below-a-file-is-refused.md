# Task ADR-199-T1: A path below something that is not a directory is refused on its hunk

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the obstruction check in `Apply` and `planPathOp`; `TestAPathBelowAFileIsRefusedOnItsHunk`; a contract section
**Consumes:** `asPlan` (ADR-198); lands after ADR-198's second task (pull request #645)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a path below a non-directory is refused on its hunk`, `a rename destination below one is refused in mrw's words`, `a path below a directory, a missing directory or a link to a directory applies`, `a prefix the boundary refuses is not looked at`, `the built binary exits 1 with a receipt`, `the packages the task does not own are unchanged`

## Goal

Decisions 1-4 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/below199_test.go` | new | the red test |
| `internal/apply/apply.go` | edit | the check where a load fails (`apply.go:696`) and where a file does not exist, after the boundary and the case-clash refusals |
| `internal/apply/pathop.go` | edit | the check where a rename destination's `Lstat` fails (`pathop.go:203-227`) |
| `internal/apply/reason198_test.go` | edit | its `rename dest … cannot be used` case used a destination below a file and skipped on Windows; it moves to ADR-066's fixture, a 300-byte leaf under an existing directory (`pathop_commit_test.go:71-81`), and runs everywhere. ADR-198 T1's lock moves: relock |
| `scripts/contract.sh` | edit | the next free section, through the built binary |
| `docs/adr/BACKLOG.md` | edit | the deferral of a prefix the same plan writes |

## Ordered Steps

1. [S1] Write the failing test `TestAPathBelowAFileIsRefusedOnItsHunk` and see it RED. Through `Apply` in a temporary root holding `d/f.txt` and `d/g.txt`, with the staging seam (`stageFileFn`) failing the test if it is called. Each refused case: no error returned, `Failed` 1, nothing changed on disk, the reason exactly `P cannot be written: A WHAT`, WHAT as the record lists it for that kind of ancestor:
   - below a file, one case per op: `create` one and two levels down, `replace`, `insert-after`, `insert-before`, `delete`, `unlink`, a rename whose source is `d/f.txt/x`;
   - a rename of `d/g.txt` to `d/f.txt/x`, alone and beside an edit of `d/g.txt`;
   - an edit of `d/g.txt` beside a `create` of `d/f.txt/x`: the edit reports skipped;
   - an `unlink` of `d/f.txt` beside a `create` of `d/f.txt/x`: refused;
   - below a FIFO (unix, in its own test `TestAPathBelowAFifoIsRefusedOnItsHunk`, which the fence does not name); below a link to a file, a link whose target is below a file and a link loop (each skipped where it cannot be made; the loop case skips on Windows, where the boundary refuses a loop first at its hop limit).
   Kept, each a case:
   - applied as now: `create` `d/new/x.txt`, `create` `d/x.txt`, `create` below a link to a directory;
   - applied as now: a `create` below a link to nothing whose directory another hunk of the plan makes;
   - a prefix that is a link out of the root: refused in the boundary's words, and the outside target's listing is unchanged (the whole-path judgement refuses these, so they do not exercise the per-prefix one);
   - two spellings of one file below a file: the case-clash refusal wins.
2. [S2] The check. Mutants, each killed:
   - the check removed where a load fails, so the `replace` case goes red;
   - the check removed where a file does not exist, so a Windows-shaped case goes red (the test drives it with `loadFn` answering "does not exist" for the path, as Windows does);
   - the check removed from the rename destination, so the rename case goes red on its words;
   - the check refusing a prefix that does not exist, so the missing-directory case goes red;
   - the check testing `IsRegular` where it tests "not a directory", so the FIFO case goes red;
   - the check using `Lstat` alone, so the create below a missing directory under a link to a directory goes red; a parent that exists ending the check whatever it is, so the create below a file goes red;
   - the boundary judgement of the prefix removed: SURVIVES, declared uncovered (the record's Risks row says why).
   [proof: mutation]
3. [S3] The contract section, through the built binary: `create` of `d/f.txt/x` exits 1, prints a FAIL row naming `d/f.txt is a file, not a directory`, and makes nothing; the pair: `create` of `d/new/x.txt` exits 0 and makes the file. [proof: acceptance]
4. [S4] `go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/` pass. [proof: acceptance]

Before the merge, outside the fence: the built binary is probed on Windows with issue #644's plans, and each exits 1 with the new reason.

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/apply/ -count=1 -v -run '^(TestAPathBelowAFileIsRefusedOnItsHunk|TestAReasonSpellsAPlansPathAsItsRowDoes)$' 2>&1) \
  && grep -qE '^--- PASS: TestAPathBelowAFileIsRefusedOnItsHunk \(' <<<"$out" \
  && grep -qE '^--- PASS: TestAReasonSpellsAPlansPathAsItsRowDoes \(' <<<"$out" \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ -count=1 >/dev/null 2>&1 \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a create below a file exits 1 with a receipt and makes nothing' <<<"$c" \
  && grep -qF 'PASS  the pair: a create below a missing directory makes it' <<<"$c" \
  && f=$(gofmt -l internal cmd) && [ -z "$f" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/check internal/state internal/seen internal/writer internal/rooted internal/mcp cmd \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/check internal/state internal/seen internal/writer internal/rooted internal/mcp cmd)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPathBelowAFileIsRefusedOnItsHunk` | `internal/apply/below199_test.go` | each op on a path below a non-directory, and a rename to one, is refused on its hunk before staging in mrw's words; paths below directories apply; a prefix outside the root is not looked at | none | S1, S2 |
| `TestAReasonSpellsAPlansPathAsItsRowDoes` | `internal/apply/reason198_test.go` | ADR-198's cases, the `cannot be used` one on ADR-066's fixture | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every write whose file cannot be found or opened: `Apply`'s file loop and `planPathOp` |
| 3 — the caller can discover it | it is the refusal itself |
| 4 — it is used | no telemetry (ADR-009); the evidence is issue #644's measurements |

## Mutation Log
- 2026-11-15 · 58365dc0* · mutant killed · exit 1 · `internal/apply/apply.go` · the check is not made where a load fails, so the replace below a file case must go red · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · covers:a path below a non-directory is refused on its hunk
- 2026-11-15 · 58365dc0* · mutant killed · exit 1 · `internal/apply/apply.go` · the check is not made where a file reports as missing, so the Windows-shaped cases must go red · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · covers:a path below a non-directory is refused on its hunk
- 2026-11-15 · 58365dc0* · mutant killed · exit 1 · `internal/apply/pathop.go` · the check is not made for a rename destination, so the rename case must go red on its words · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · covers:a rename destination below one is refused in mrw's words
- 2026-11-15 · 58365dc0* · mutant killed · exit 1 · `internal/apply/apply.go` · a prefix that does not exist is treated as an obstruction, so the create below a missing directory must go red · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · covers:a path below a directory, a missing directory or a link to a directory applies
- 2026-11-15 · 58365dc0* · mutant killed · exit 1 · `internal/apply/apply.go` · only a regular file or a link counts, so the FIFO case must go red · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · covers:a path below a non-directory is refused on its hunk
- 2026-11-15 · 58365dc0* · mutant killed · exit 1 · `internal/apply/apply.go` · the prefix is judged by the link itself, so the create below a missing directory under a link to a directory must go red · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · covers:a path below a directory, a missing directory or a link to a directory applies
- 2026-11-15 · 58365dc0* · mutant killed · exit 1 · `internal/apply/apply.go` · a parent that exists ends the check whatever it is, so the create below a file case must go red · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · covers:a path below a non-directory is refused on its hunk

## Verification Log
- 2026-11-15 · 58365dc0* · exit 1 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:1097 · test-lock-sha256:6af3851a85641b0ca90979ca22321234caab0b97e3ed2b98eb8a624b2a73840d · test-lock-file:docs/adr/locks/6af3851a85641b0ca90979ca22321234caab0b97e3ed2b98eb8a624b2a73840d.lock
  ```
  ```
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:210086
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:230696
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:243588
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:287867
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:213435
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:194771
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:208329
- 2026-11-15 · 58365dc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:8a35f0853190a1859d15c8fe411313aeb2ee40e53feb9558ddf28e7d15c15826 · ms:210749

## Invariants

- A plan that applies today applies. The check never runs for a file that opens; for a file the plan makes it costs one `Stat` when the parent is a directory, and a walk of the existing prefixes when the plan also makes the parent.
- A refusal by the check writes nothing, stages nothing and returns no error.

## Risks

- The Windows route (a file that reports as missing) is driven on macOS through `loadFn`; the real one is shown by the Windows CI job and the probe.

## Stop Condition

The fence exits 0 and the Windows probe exits 1 with the new reason for each of issue #644's plans. Stop and ask Zy if a test or contract row outside this record's pins the old exit code or words.

## Out of Scope

- A prefix the same plan writes as a file (deferred: docs/adr/BACKLOG.md)
- A `from=` source below a non-directory (permanent: boundary: the record leaves ADR-167's refusal as it is)
