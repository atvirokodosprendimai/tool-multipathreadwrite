# Task ADR-034-T1: The state base can describe itself, and drop only its dead

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (one new file in `internal/state`, plus the fixture that pins it)
**Owner:** Zy
**Produces:** `state.Entry`, `state.Entries()`, `state.Prune()` (T2)
**Consumes:** none
**Data dependency:** hermetic — every fixture builds its own base under `t.TempDir()` and pins `XDG_STATE_HOME` to it
**Proof map:** v1
**Rests-on:** `only an entry whose marker names a missing path is removed`, `an unidentifiable entry is kept`, `the walk never leaves the base`, `a dry run removes nothing and reports the same list`, `the running checkout's own entry is kept`

## Goal

Give `internal/state` an exact, reporting answer to "which of these directories can never be used
again", without giving it the power to decide that on its own.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/prune.go` | create | `Entry`, `Entries()` and `Prune()`. A new file rather than an addition to `state.go`, because `state.go` answers "where does this root's state live" and every function in it takes a root; these take a base and answer a different question about all of them. |
| `internal/state/prune_test.go` | create | The fixture and its three-entry base. In `internal/state` and not `internal/adversarial`, because what it asserts is this package's own contract rather than a ledger property. |

## Ordered Steps

1. [S1] Write `TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned` and confirm it is RED. ⚠ **The base must
   hold THREE entries, not one.** A base holding only a dead entry is green against a `Prune` that
   removes everything — which is the whole failure this record's Risks table names as High. Build a
   LIVE entry (its root a directory that exists), a DEAD entry (its root a directory created and
   then removed), and an UNIDENTIFIABLE entry (a directory with no `root` file at all). Assert the
   dead one is gone AND both others are still on disk.
2. [S2] Write `Entry` and `Entries()`: one `os.ReadDir` of `<base>/mrw`, and per child the
   directory, the root its marker names, whether that root is a directory now, and the size. No
   recursion, no deletion, no judgement. [proof: mutation]
3. [S3] Write `Prune(root string, dryRun bool)`. It calls `Entries()`, selects entries whose marker
   resolved AND is absolute AND names a path that is not a directory, excludes the entry for `root`
   itself, and — unless `dryRun` — removes each with `os.RemoveAll`. It returns the selected entries
   either way, so a dry run and a real run report the same list. [proof: mutation]
4. [S4] Assert the kept classes in the same test: an entry with no marker survives, one whose marker
   holds a RELATIVE path survives, one whose marker is empty survives, and the entry belonging to
   the root passed in survives even when its own root has been removed. Without these the fix could
   be "delete every entry", which S1 alone would not catch for the relative and empty cases.
   [proof: mutation]
5. [S5] Write `TestThePruneStaysInsideTheStateBase`: plant a symlink under the base pointing at a
   directory outside it whose target is live, and a plain file, and assert `Prune` removes neither
   and that the symlink's target still exists afterwards. [proof: mutation]
6. [S6] Write `TestAnUnidentifiableEntryIsKeptAndReported`: `Entries()` marks the no-marker entry
   with an empty `Root` and `Identified` false, so the command layer can print it. A kept entry that
   is not reported is indistinguishable from one that was never looked at. [proof: mutation]
7. [S7] Run the package, `gofmt`, `go vet`, and the whole suite. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/state/ -count=1 -v \
  -run 'TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned|TestThePruneStaysInsideTheStateBase|TestAnUnidentifiableEntryIsKeptAndReported|TestADryRunPruneReportsTheSameAndRemovesNothing' 2>&1 | tee /tmp/adr034-t1.out \
  && grep -q '^--- PASS: TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned' /tmp/adr034-t1.out \
  && grep -q '^--- PASS: TestThePruneStaysInsideTheStateBase' /tmp/adr034-t1.out \
  && grep -q '^--- PASS: TestAnUnidentifiableEntryIsKeptAndReported' /tmp/adr034-t1.out \
  && grep -q '^--- PASS: TestADryRunPruneReportsTheSameAndRemovesNothing' /tmp/adr034-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr034-t1.out \
  && go test ./internal/state/ -count=1 -v -run 'TestNoStateIsWrittenUnderTheRepoRoot' 2>&1 | tee /tmp/adr034-t1n.out \
  && grep -q '^--- PASS: TestNoStateIsWrittenUnderTheRepoRoot' /tmp/adr034-t1n.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./... \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned` | `internal/state/prune_test.go` | In a base holding a live, a dead and an unidentifiable entry, only the dead one is removed; and the same base after `dryRun` still holds all three while reporting the same one | — | S1, S2, S3, S4 |
| `TestThePruneStaysInsideTheStateBase` | `internal/state/prune_test.go` | A symlink under the base is not followed and not removed, its target survives, and a plain file under the base is left alone | — | S5 |
| `TestAnUnidentifiableEntryIsKeptAndReported` | `internal/state/prune_test.go` | An entry with no marker, one with an empty marker and one with a relative marker are each reported with `Identified` false and each still on disk after a real prune | — | S6 |
| `TestADryRunPruneReportsTheSameAndRemovesNothing` | `internal/state/prune_test.go` | A dry run over the same base names the dead entry and leaves every directory on disk, and the real run that follows removes exactly what the dry run promised — which is what makes `--dry-run` a preview rather than a second question | — | S3 |
| `TestNoStateIsWrittenUnderTheRepoRoot` | `internal/state/state_test.go` | Unchanged: ADR-004's guarantee is not disturbed by a package that now also deletes | — | S7 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `state.Prune` in `internal/state/prune.go` |
| 2 — something selects it | T2 wires it to `mrw seen --prune`; until then it is reachable only from its test, which this task states plainly rather than claiming a caller it does not have |
| 3 — the caller can discover it | T2's `--prune` flag and its usage string; the count line on plain `mrw seen` is what points at it |
| 4 — it is used | Contract §71 (T3) drives it through the built binary against a base built by the row itself; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · 882fdea · mutant killed · exit 1 · `internal/state/prune.go` · the unidentified guard goes, so an entry with no readable root marker is deleted — the one mistake here that re-reading a file cannot undo · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · covers:an unidentifiable entry is kept
- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `internal/state/prune.go` · the self guard never matches, so the entry for the root the caller is running in is deleted when that checkout has been removed underneath the process. ⚠ The obvious spelling — dropping the clause — leaves `self` unused and does NOT COMPILE, and a mutant that does not build proves nothing; this one keeps self live and never matches · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · covers:the running checkout's own entry is kept
- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `internal/state/prune.go` · the live guard goes, so a checkout that still exists loses its ledger and its next write is refused · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · covers:only an entry whose marker names a missing path is removed
- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `internal/state/prune.go` · dryRun is ignored, so the preview deletes — the flag a caller reaches for precisely because they are not sure · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · covers:a dry run removes nothing and reports the same list

## Verification Log

- 2026-09-07 · fa14205* · exit 1 · `set -o pipefail …` · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · ms:10441
  ```
  --- last 10 line(s) of stdout (of 36 after folding 36 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	2.386s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/curve	2.067s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	1.996s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	8.722s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	2.283s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	2.363s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	2.362s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	2.439s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state	2.241s
  FAIL
  ```
- 2026-09-07 · fa14205* · exit 1 · `set -o pipefail …` · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · ms:9693
  ```
  --- last 10 line(s) of stdout (of 36 after folding 36 raw)
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check	2.396s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/curve	1.116s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter	0.263s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	8.131s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan	1.034s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read	1.281s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	1.251s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen	1.233s
  ok  	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state	1.216s
  FAIL
  ```
- 2026-09-07 · 882fdea · exit 0 · `set -o pipefail …` · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · ms:17659
- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · ms:15795
- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · ms:15348
- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:b9360ac7fdbee50aa8c145c8d3d246d7bd00d360be0113b75a1f8e4d4b90cc74 · ms:20270

## Invariants

- `Dir()`, `Path()`, `LegacyPath()`, `Migrate()`, `stateHome()`, `absReal()` and `key()` are
  unchanged. This task adds a file; it does not edit `state.go`.
- `LegacyDir` is never passed to `os.RemoveAll` on any path. It is inside the checkout and the walk
  only ever reads `<base>/mrw`, so this holds by construction rather than by a check.
- `Prune` with `dryRun` true performs no filesystem write of any kind, and returns the identical
  slice a real run would.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and `internal/check` stay
  byte-identical against the merge base, and `go.mod` declares exactly one requirement. This record
  owns `internal/state`, which is why that package is absent from the fence's go/no-go clause.

## Risks

- ⚠ **The fixture passes for the wrong reason.** A one-entry base is green against a `Prune` that
  deletes the base wholesale. S1 builds three entries for that reason and the S3 mutant is what
  proves the distinction held. This is pre-registered in the parent record's Risks table as the
  High-likelihood failure.
- A dead root can be a live root on an unmounted volume. `Prune` cannot tell, and does not try; the
  parent record's Decision says why the operator is the gate. Nothing in this task changes when it
  runs.
- `os.RemoveAll` on a directory the process cannot fully remove leaves a partial entry. Reported as
  an error per entry rather than aborting the run, so one unremovable directory does not strand the
  other 22,590.

## Stop Condition

Stop and ask if `Entries()` cannot distinguish a symlink from a directory without following it on
some platform — that would mean the containment guarantee rests on a `Stat` that leaves the base,
and the design needs revisiting before either is written.

## Out of Scope

- The CLI flag — T2
- The contract row — T3
- Any age, size or count criterion for selecting an entry (permanent: boundary: the marker test is exact; the parent record's Alternatives rejects a heuristic that is strictly weaker)
