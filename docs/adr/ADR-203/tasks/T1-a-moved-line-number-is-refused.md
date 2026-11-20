# Task ADR-203-T1: a number alone is refused on an unshown line of a written file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** the refusal and `refusal.StaleNumber`; the `mrw stats` sentence; contract §349 and the changed §343; the instructions, `AGENTS.md`, `README.md`, `BESTPRACTICES.md`
**Consumes:** `staleMarks` and `unshown` (`internal/apply/apply.go`, ADR-200), `refusal.Kind`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `a number alone on an unshown line of a written file is refused`, `every way to say which line still lands`, `the refusal names the first unshown line and the ways out`, `the stats block says what its counters are now`, `the other engine packages are untouched`

## Goal

ADR-203 Decisions 1 to 5. `f.txt` holds `L1` to `L10`, is read whole, and is then written by mrw with `@@ f.txt 3 insert-after` one line `NEW`. Then, with no read between unless the row says so:

| Second plan | Outcome |
|-------------|---------|
| `@@ f.txt 8 replace` | refused: exit 1, the file unchanged, the reason naming line 8 and the three ways out, kind `stale-number` |
| `@@ f.txt 2 replace` | refused: above the insert too, since the rule asks only whether the line was shown |
| `@@ f.txt 4-6 delete` | refused, naming line 4 |
| `@@ f.txt 8 replace anchor="L7"` | lands, on the line that holds L7 |
| `@@ f.txt /^L7$/ replace` | lands |
| `mrw read f.txt:8`, then `@@ f.txt 8 replace` | lands |
| `mrw read f.txt:5-6`, then `@@ f.txt 4-6 delete` | refused, naming line 4; after `mrw read f.txt:4-6` it lands |
| `mrw read --quiet f.txt:8`, then `@@ f.txt 8 replace` | refused: a quiet read shows nothing |
| `@@ f.txt 8 replace` under `--force` | lands |
| `mrw write --grep '^L7$' --sub X` | lands: computed in the call |
| `@@ f.txt $ replace`, `@@ f.txt 0 insert-after`, `@@ f.txt 1 insert-before`, `@@ f.txt 12 insert-before` | land: positions |
| `@@ f.txt 5 insert-after` | refused: it names line 5 |
| a second hunk `@@ g.txt 1 replace anchor=…` on a read file beside the refused one | `skip`, nothing written to either |

And: after `mrw write --create c.txt` of five lines, `@@ c.txt 4 replace` is refused and `@@ c.txt 4 replace anchor="…"` lands. A file read whole and never written lands by number as before. Over MCP, `mrw_write` with the first row's plan answers `isError` with the reason in the hunk.

`mrw stats` after a refused write: `landed writes` unchanged, and the block's last line no longer names 2026-12-18.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/apply.go` | edit | the refusal at `staleMarks`' call sites |
| `internal/refusal/refusal.go` | edit | `StaleNumber` |
| `internal/apply/stale203_test.go` | new | the Goal's table against the engine |
| `cmd/mrw/stale203_test.go` | new | the first, fourth, sixth, eighth, ninth and create rows through the built binary, and the stats sentence |
| `internal/mcp/stale203_test.go` | new | the MCP row |
| `internal/guide/stale203_test.go` | new | the instructions |
| `cmd/mrw/main.go` | edit | the `mrw stats` sentence |
| `cmd/mrw/stale200_test.go` | edit | its chain lands by `anchor=`; its stats expectation follows the sentence (ADR-200 T1's lock is relocked after review) |
| `internal/guide/guide.go`, `internal/guide/renumber200_test.go`, `.claude/skills/mrw/SKILL.md` | edit | rule 2, recipes 7 and 21 |
| `scripts/contract.sh` | edit | §343's seed, §349 |
| `AGENTS.md`, `README.md`, `BESTPRACTICES.md` | edit | the same sentences |
| `docs/adr/BACKLOG.md` | edit | the reading set aside, the friction reading, the deferrals |
| other tests and contract rows that chain numbered writes | edit | each by an anchor, a pattern or a read, listed in the pull request |

## Ordered Steps

1. [S1] The four tests, RED. [proof: acceptance]
2. [S2] The refusal and its kind. Mutants, each killed: the refusal is not made; it is made on a shown line; it is made with `anchor=`; it is made under `--force`; it is made on a computed plan; it is made on a file read whole and never written; a range is judged on its first line only; the reason names the hunk's first line when that line is shown and a later one is not. [proof: mutation]
3. [S3] The `mrw stats` sentence. Mutant, killed: the sentence still names 2026-12-18. [proof: mutation]
4. [S4] Contract §349 drives the built binary: the first row refused with the file unchanged, paired with the anchored form and the re-read form landing, and a create followed by a numbered edit refused then landing with `anchor=`. §343 keeps its counters with a seed that lands. The instructions, `AGENTS.md`, `README.md`, `BESTPRACTICES.md`, BACKLOG. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/apply ./cmd/mrw ./internal/mcp ./internal/guide -run '^(TestANumberAloneIsRefusedOnALineNotShownSinceTheWrite|TestAStaleNumberIsRefusedByTheBinary|TestMrwWriteRefusesAStaleNumber|TestTheInstructionsSayASecondEditNamesItsLine|TestStatsPrintsTheStaleNumberPricing)$' > "${TMPDIR:-/tmp}/adr203-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr203-fence.out")" -eq 5 ] \
  && grep -q '^# 349\. ' scripts/contract.sh \
  && grep -q 'StaleNumber' internal/refusal/refusal.go \
  && grep -q 'has no recorded showing in this version' AGENTS.md \
  && ! grep -q '2026-12-18' cmd/mrw/main.go \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/seen internal/check internal/state ':(exclude)*_test.go' \
  && [ -z "$(git diff --name-only "$(git merge-base HEAD origin/main)" -- internal/writer | grep -v -e '_test\.go$' -e '^internal/writer/flow\.go$')" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestANumberAloneIsRefusedOnALineNotShownSinceTheWrite` | `internal/apply/stale203_test.go` | every row of the Goal's table, each refusal by its reason, its kind and the file left unchanged | none | S1, S2 |
| `TestAStaleNumberIsRefusedByTheBinary` | `cmd/mrw/stale203_test.go` | the named rows through `mrw write` with exit codes; `mrw stats` afterwards | none | S1, S2, S3 |
| `TestMrwWriteRefusesAStaleNumber` | `internal/mcp/stale203_test.go` | `mrw_write`: `isError`, the reason in the hunk, nothing written | none | S1, S2 |
| `TestTheInstructionsSayASecondEditNamesItsLine` | `internal/guide/stale203_test.go` | the core and the full instructions say a second edit to a file mrw wrote names its line by pattern, `anchor=` or after a read that shows it; the core stays under its limit | none | S4 |
| `TestStatsPrintsTheStaleNumberPricing` | `cmd/mrw/stale200_test.go` | exists today (ADR-200 T1); its chain is changed to land by `anchor=` and its last-line expectation to the new sentence, which is what goes red | none | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests and §349 |
| 2 — something selects it | every `mrw write` and `mrw_write` goes through apply's validation |
| 3 — the caller can discover it | the refusal's own sentence; `mrw instructions --core` rule 2; `AGENTS.md` |
| 4 — it is used | telemetry is refused by ADR-009; the reading of BACKLOG.md "Pre-registered for ADR-203" |

## Mutation Log
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `internal/apply/apply.go` · the refusal is not made at the replace and delete site · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:a number alone on an unshown line of a written file is refused
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made on a shown line · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made with anchor= · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made under --force · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made on a computed plan · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made on a file read whole and never written · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `internal/apply/apply.go` · a range is judged on its first line only · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:a number alone on an unshown line of a written file is refused
- 2026-11-19 · 6cf84b40* · mutant killed · exit 1 · `cmd/mrw/main.go` · the stats sentence still names 2026-12-18 · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:the stats block says what its counters are now
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `internal/apply/apply.go` · the refusal is not made at the splice loop · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:a number alone on an unshown line of a written file is refused
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made on a shown line · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made with anchor= · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made under --force · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made on a computed plan · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `internal/apply/apply.go` · it is made on a file read whole and never written · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:every way to say which line still lands
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `internal/apply/apply.go` · a range is judged on its first line only · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:a number alone on an unshown line of a written file is refused
- 2026-11-19 · 72f22a66* · mutant killed · exit 1 · `cmd/mrw/main.go` · the stats sentence still names 2026-12-18 · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · covers:the stats block says what its counters are now

## Verification Log
- 2026-11-19 · 6cf84b40* · exit 1 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:285 · test-lock-sha256:24c171f0c4decb5838dc7227f731d23aaf4b6030c352866399acb589138a980f · test-lock-file:docs/adr/locks/24c171f0c4decb5838dc7227f731d23aaf4b6030c352866399acb589138a980f.lock
  ```
  --- last 2 line(s) of stderr
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read
  internal/read/plansource.go:117:175: sub.Unanchored undefined (type *Sub has no field or method Unanchored)
  ```
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:2191
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:3537
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:3999
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:4572
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:5075
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:11781
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:14902
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:20416
- 2026-11-19 · 6cf84b40* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:7105
- 2026-11-19 · 72f22a66* · exit 1 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:955
  ```
  --- last 2 line(s) of stderr
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read
  internal/read/plansource.go:117:175: sub.Unanchored undefined (type *Sub has no field or method Unanchored)
  ```
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:3201
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:4455
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:4065
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:4432
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:4485
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:3989
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:4263
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:4039
- 2026-11-19 · 72f22a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:3925
- 2026-11-19 · 4e893a30* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:0 · test-lock-sha256:407db267a54522da1e37143b276a80cf6d25483a0f81a702ac0a635bc40cf2c0 · test-lock-file:docs/adr/locks/407db267a54522da1e37143b276a80cf6d25483a0f81a702ac0a635bc40cf2c0.lock · test-lock-kind:replace
- 2026-11-19 · 4e893a30* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:2650
- 2026-11-20 · b1ad6d06* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:5 · test-lock-sha256:399745c8fda374cbd367ed9a8e6613ce5d6cac0537953b045d3f13715c984a81 · test-lock-file:docs/adr/locks/399745c8fda374cbd367ed9a8e6613ce5d6cac0537953b045d3f13715c984a81.lock · test-lock-kind:replace
- 2026-11-20 · b1ad6d06* · exit 0 · `set -o pipefail …` · acceptance-sha256:8641905cbfac27e479d8c97eca8bf7cb7122f059627159fac284df5aa4c22f82 · ms:3013

## Invariants

- A plan that addresses by pattern, carries `anchor=`, is quoted, forced, or computed by mrw lands as before.
- A file read whole by a read that shows it, and never written by mrw, is judged as before. A whole-file `--quiet` read is not such a read: a numbered edit after it is refused (Decision 7).
- The ledger's format and every receipt key are unchanged.

## Risks

- Every existing test, contract row and document example that edits by number alone a file mrw wrote or read whole with `--quiet`, with no showing read between, goes red. The second challenge's source audit found at least 13 Go test functions (among them `cmd/mrw/pwsh161_test.go`, `refused176_test.go`, `planpath_test.go`, `marks537_test.go`, `longline179_test.go`, `internal/writer/spelling597_test.go`, `stale200_test.go`, `ledgererr200_test.go`, `internal/apply/stale200_test.go`) and 15 contract sections (§§38, 172, 173, 196, 213, 215, 256, 270, 301, 303, 305, 320, 323, 332, 343); that is a lower bound. Each is changed by an anchor where the test proves the write's licence, by a showing read where it is about a check, step or receipt, and rewritten where it required an unshown candidate to land; every record whose locked test moves is relocked after review, and the full list is in the pull request.
- The Goal's rows each start from a fresh seed and give every `replace` and insert a body. Added by the second challenge and binding on the tests: a range with only a middle line missing (`read :3` and `:6`, then `4-6 delete`, names line 5); two stale hunks in one plan both reported; a whole CLI read after the seed write lands by number; a whole-file `--quiet` read then a numbered edit is refused and a ranged one lands; a rename's destination is refused until read under its name; `read --sub` then `write` of its output lands; MCP serve, refuse, retry with `ack` lands; the three `numbering` counters after a refusal, an anchored landing, a shown landing and a dry run.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- A selective refusal with ledger state (deferred: docs/adr/BACKLOG.md)
