# Task ADR-034-T2: `mrw seen` reports the base, and prunes it when asked

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (two flags and one line on an existing command)
**Owner:** Zy
**Produces:** `mrw seen --prune`, `mrw seen --dry-run`, and the count line on plain `mrw seen` (T3)
**Consumes:** `state.Entries()`, `state.Prune()` (T1)
**Data dependency:** hermetic — the tests build a base under `t.TempDir()` and pin `XDG_STATE_HOME`
**Proof map:** v1
**Rests-on:** `the flag is the only way to delete`, `a dry run deletes nothing`, `the state directory is still the first line`

## Goal

Put `state.Prune` behind an explicit flag, and make the growth it cleans up visible on the command
that already answers "where is my state" — without moving the line every existing caller reads.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `seenCmd` gains `--prune` and `--dry-run`, the report, and the count line. It is the command whose subject is already the state directory; a second top-level verb for the same subject is a surface that has to be discovered separately. |
| `cmd/mrw/prune_test.go` | create | The CLI-level fixtures. Separate from `seen`'s existing tests because these drive a code path that deletes, and a shared `t.TempDir()` helper between the two would bind cleanup to the first caller's `t` — the trap `.claude/rules/testing.md` names. |
| `README.md` | edit | The `mrw seen` section documents both flags and what the count line means. T3 owns the claim; this task owns the text so the flag is never in a release nobody can find. |

## Ordered Steps

1. [S1] Write `TestSeenPruneRemovesOnlyTheDeadEntries` and confirm it is RED — a base with a live,
   a dead and an unidentifiable entry, driven through `seenCmd`'s action rather than through
   `state.Prune`, so this asserts the WIRING and not the function T1 already proved. ⚠ Assert the
   filesystem, not the printed report: a command that prints the right sentence and removes nothing
   passes an output check.
2. [S2] Add the `--prune` and `--dry-run` flags. `--dry-run` without `--prune` is a usage error
   (exit 2), because a dry run of the thing that only reads is not a request anyone can mean.
   [proof: mutation]
3. [S3] Print the report: one line per removed entry naming the state directory and the checkout its
   marker named, then a total. ADR-008's rule — a delete says what it removed — applies whole, so a
   run that removes nothing says so rather than printing nothing. [proof: mutation]
4. [S4] Write `TestADryRunPruneRemovesNothing`: the same base, `--prune --dry-run`, identical report
   text, and every entry still on disk afterwards. [proof: mutation]
5. [S5] Add the count line to plain `mrw seen`, AFTER the directory. Write
   `TestTheStateDirectoryIsStillTheFirstLine`, which asserts `head -1` of the output is the
   directory and nothing else — the one thing `scripts/contract.sh:1590` and the documented recipe
   depend on. [proof: mutation]
6. [S6] Document both flags and the count line in `README.md`'s `mrw seen` section, and in the
   command's own `Description`. [proof: acceptance]
7. [S7] Run the package, the whole suite, `gofmt`, `go vet`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -v \
  -run 'TestSeenPruneRemovesOnlyTheDeadEntries|TestADryRunPruneRemovesNothing|TestTheStateDirectoryIsStillTheFirstLine' 2>&1 | tee /tmp/adr034-t2.out \
  && grep -q '^--- PASS: TestSeenPruneRemovesOnlyTheDeadEntries' /tmp/adr034-t2.out \
  && grep -q '^--- PASS: TestADryRunPruneRemovesNothing' /tmp/adr034-t2.out \
  && grep -q '^--- PASS: TestTheStateDirectoryIsStillTheFirstLine' /tmp/adr034-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr034-t2.out \
  && grep -q -- '--prune' README.md \
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
| `TestSeenPruneRemovesOnlyTheDeadEntries` | `cmd/mrw/prune_test.go` | Driving `seen --prune` over a three-entry base removes the dead entry and leaves the live and unidentifiable ones ON DISK — asserted by stat, not by reading the report | — | S1, S2, S3 |
| `TestADryRunPruneRemovesNothing` | `cmd/mrw/prune_test.go` | `--prune --dry-run` reports the same entry and every directory survives; `--dry-run` alone exits 2 | — | S2, S4 |
| `TestTheStateDirectoryIsStillTheFirstLine` | `cmd/mrw/prune_test.go` | The first line of plain `mrw seen` is the state directory and the count line follows it, so the recipe at `scripts/contract.sh:1590` keeps working | — | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | The `--prune` flag on `seenCmd` |
| 2 — something selects it | The flag is parsed by the command the binary already registers at `cmd/mrw/main.go:165`; the test drives the command, not the function |
| 3 — the caller can discover it | `mrw seen --help`, the README section, and the count line on plain `mrw seen`, which is the only place the growth becomes visible without knowing the flag exists |
| 4 — it is used | Contract §71 (T3) drives it through the built binary. No adoption is claimed: this ships today, so the evidence for building it is the 22,836-directory measurement in the parent record, not usage |

## Mutation Log

- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `cmd/mrw/main.go` · the --prune flag is parsed and never acted on, so the command prints the ledger and removes nothing — a flag that exists and does nothing · acceptance-sha256:f98379ddd2ebf26ac7323976eb5baeabdcbc3ec0b732ebba67e04e5038639609 · covers:the flag is the only way to delete
- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `cmd/mrw/main.go` · --dry-run is dropped on the way in, so the preview deletes — the flag a caller reaches for precisely because they are not sure · acceptance-sha256:f98379ddd2ebf26ac7323976eb5baeabdcbc3ec0b732ebba67e04e5038639609 · covers:a dry run deletes nothing
- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `cmd/mrw/main.go` · the count line is printed BEFORE the directory, so `mrw seen | head -1` stops naming the state directory and contract section 54 reads a comment as a path · acceptance-sha256:f98379ddd2ebf26ac7323976eb5baeabdcbc3ec0b732ebba67e04e5038639609 · covers:the state directory is still the first line

## Verification Log

- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:f98379ddd2ebf26ac7323976eb5baeabdcbc3ec0b732ebba67e04e5038639609 · ms:15691
- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:f98379ddd2ebf26ac7323976eb5baeabdcbc3ec0b732ebba67e04e5038639609 · ms:15606
- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:f98379ddd2ebf26ac7323976eb5baeabdcbc3ec0b732ebba67e04e5038639609 · ms:14934

## Invariants

- The state directory stays the FIRST line of `mrw seen`, with or without the count line. The
  existing `scripts/contract.sh:1590` recipe reads it that way and is not changed by this task.
- Nothing but `--prune` reaches `state.Prune`. No other subcommand, no startup path, no migration.
- `--prune` exits 0 whether it removed anything or not; nothing here invents a new exit code.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and `internal/check` stay
  byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- A test that asserts the report rather than the filesystem passes against a command that prints
  correctly and deletes nothing. S1 states the assertion explicitly for that reason, and the S3
  mutant removes the `RemoveAll` to prove it.
- The added line could break a caller parsing `mrw seen` positionally. Mitigated by keeping the
  directory on line 1 and pinning it with its own test and with §71; named in the parent record's
  Inter-task Contracts as the one breaking edge.
- `--dry-run` is a flag name the write path already uses with a different subject. Judged
  acceptable: it means the same thing in both — apply nothing — and a different word for the same
  promise is worse than a shared one.

## Stop Condition

Stop and ask if any existing test or contract row reads `mrw seen` beyond its first line — that
would mean the count line moves something a gate depends on, and the line's position needs deciding
before it is written.

## Out of Scope

- The contract row and the rest of the documentation sweep — T3
- Making `contract.sh` stop producing orphans — T4
- A confirmation prompt before deleting (permanent: boundary: the command is already explicit and `--dry-run` is the preview; a prompt would make the flag unusable from a script, which is where the 22,590 came from)
