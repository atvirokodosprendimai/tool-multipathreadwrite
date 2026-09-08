# Task ADR-034-T3: The contract drives the prune, and the docs say it exists

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** M (one contract section, four documents)
**Owner:** Zy
**Produces:** contract §71, and the documented `--prune` surface
**Consumes:** `mrw seen --prune`, `mrw seen --dry-run`, the count line (T2)
**Data dependency:** hermetic — §71 builds its own base under `$WORK` and pins `XDG_STATE_HOME` to it
**Proof map:** v1
**Rests-on:** `the built binary removes the dead entry`, `the built binary keeps the live and the unidentifiable one`, `a dry run through the binary removes nothing`

## Goal

Prove the BUILT binary prunes, and pair every passing case with the case that must fail — then say
in the documents a caller reads that the flag exists, because a command nobody can find is a
command nobody runs.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | §71. A Go test proves `state.Prune`; it cannot prove the binary calls it — the lesson §53 was written for. |
| `README.md` | edit | The full interface lives here. The `mrw seen` entry gains both flags and the count line. |
| `AGENTS.md` | edit | Its "other subcommands" section exists because a command absent from it is a command no agent asks for — the defect in issues #51 and #73. A new flag on `seen` belongs there for the same reason. |
| `CONTRIBUTING.md` | edit | Only if the gate list changes. Expected `none`; recorded here so the check is made rather than assumed. |

## Ordered Steps

1. [S1] Write §71's assertions FIRST and confirm they are RED against a binary that has no prune —
   `git stash` the T2 commit, or build from the merge base into `$MRW`, and watch the row fail. A
   contract row's red is "the binary does not do this yet", and a row that is green before the
   feature exists is scoring something else. Confirm at the same time that
   `grep -c '^# 71\. ' scripts/contract.sh` was 0 before the fence was written, using the
   trailing-SPACE form and `sort -k2,2n`: `.claude/rules/contract.md` records both traps, and §70 is
   taken by ADR-032 on an unmerged branch, so the corpus is grepped rather than the file's tail
   read. [proof: acceptance]
2. [S2] Write §71. It exports `XDG_STATE_HOME` to a directory of its own under `$WORK`, creates a
   live root, a dead root and a marker-less entry through the BINARY (`$MRW -C <root> read …` for
   the two roots, a bare `mkdir` for the third), removes the dead root, and then drives
   `$MRW seen --prune`. [proof: mutation]
3. [S3] Pair every case. The row asserts the dead entry is GONE **and** the live entry is PRESENT
   **and** the unidentifiable entry is PRESENT. A row scoring only the removal passes against a
   binary that removes the base. [proof: mutation]
4. [S4] Add the dry-run half: `$MRW seen --prune --dry-run` over a rebuilt base leaves all three on
   disk and still names the dead one on stdout. And `$MRW seen --dry-run` alone exits 2.
   [proof: mutation]
5. [S5] Add the first-line row: `$MRW seen | head -1` is a directory that exists, so the recipe
   §~54 already uses is pinned against the added count line. ⚠ Read the exit code from `$MRW`, never
   through the pipe — `.claude/rules/contract.md` and `CONTRIBUTING.md` both say why, and this row
   is a pipe by construction. [proof: mutation]
6. [S6] Document the flags in `README.md` and `AGENTS.md`, including the sentence that says a prune
   removes nothing whose `root` marker is missing or unreadable — the kept class is the part a
   reader will otherwise assume the other way. [proof: acceptance]
7. [S7] Run `./scripts/contract.sh` stand-alone and confirm `contract holds`, then the full gate
   list. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 71\. ' scripts/contract.sh \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr034-t3c.out \
  && grep -q '^contract holds$' /tmp/adr034-t3c.out \
  && grep -q -- '--prune' README.md \
  && grep -q -- '--prune' AGENTS.md \
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
| `none — the row IS the test` | `scripts/contract.sh` | §71: the BUILT binary removes the dead entry, keeps the live and the unidentifiable ones, reports what it removed, removes nothing under `--dry-run`, refuses `--dry-run` alone with exit 2, and still prints the state directory as line 1 | — | S1, S2, S3, S4, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §71 in `scripts/contract.sh` |
| 2 — something selects it | The section runs on every `./scripts/contract.sh`, which is half of this project's declared check |
| 3 — the caller can discover it | `README.md`'s interface section and `AGENTS.md`'s subcommand list — the two places issues #51 and #73 established are where a missing command goes unnoticed |
| 4 — it is used | The row drives `$MRW`, the binary built at the top of the script, so it fails if `seenCmd` stops calling `state.Prune`; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `cmd/mrw/main.go` · the BUILT binary no longer prunes: section 71 must go red, which is what proves the row drives the binary rather than the Go tests. A unit test cannot prove the binary calls the function — the lesson section 53 was written for · acceptance-sha256:e44e06338ac4db500a1a3c0ff81c3265bc28ecb1a12ebc77fbf0e672a7fdc0fc · covers:the built binary removes the dead entry

## Verification Log

- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:e44e06338ac4db500a1a3c0ff81c3265bc28ecb1a12ebc77fbf0e672a7fdc0fc · ms:32848

## Invariants

- §71 leaves the base it created inside `$WORK`, which the script's existing `EXIT` trap removes. It
  is the first row that would otherwise write outside `$WORK` by design, which is why it pins
  `XDG_STATE_HOME` itself even before T4 pins it for the whole file.
- No exit code is read through a pipe. Every `want` in §71 reads `$?` from the binary directly.
- Section numbers stay unique against the corpus, not against this file's tail.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and `internal/check` stay
  byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- §71 collides with ADR-032's §70 if that record's number moves. Mitigated by grepping the corpus
  for the number rather than reading this file's tail, and by taking 71 while 70 is claimed on an
  unmerged branch.
- A row that builds its base by hand rather than through the binary would prove nothing about the
  marker `Dir()` writes. S2 creates the two live-then-dead roots by running `$MRW read` against
  them, so the markers under test are the ones the tool actually writes.
- The documentation could describe a kept class the code does not have. S6 names the unidentifiable
  case explicitly and §71 asserts it, so the claim and the row move together.

## Stop Condition

Stop and ask if §71 cannot be made to fail against a binary without the prune — a row that is green
either way is the defect `.claude/rules/testing.md` names first, and it would mean the row is
scoring something other than the deletion.

## Out of Scope

- `contract.sh`'s own leak — T4, which pins `XDG_STATE_HOME` for the whole file
- The instructions string served over MCP (permanent: boundary: `mrw seen` has no MCP surface; the server exposes `mrw_read` and `mrw_write` only, so nothing there could name the flag)
- A `mrw stats` row for pruned entries (permanent: boundary: `stats` counts what happened to plans, and a prune is not a plan)
