# Task ADR-202-T1: an inverted match on standard input

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** E7 in `docs/bench/effect-study/` and its reading in BACKLOG.md; `mrw read --invert`; the pointer in the `--grep -v P -` refusal; contract §346; the sentences in `mrw instructions`, `mrw read --help` and `AGENTS.md`
**Consumes:** `stdinRead` (`cmd/mrw/stdin.go`), `read.Run`'s numbered ranges, `stdinFlags` (`cmd/mrw/foreignflag.go`), the effect-study scanner
**Data dependency:** local transcripts — S1 reads `~/.claude/projects` on the machine it runs on and prints aggregates only; S2 to S5 are hermetic
**Proof map:** v1
**Rests-on:** `the lines that do not match are served, numbered as in the input`, `the flag is refused everywhere else`, `nothing left exits 1 with its kind`, `the refusal of --grep -v points at the form`, `the engine packages are untouched`

## Goal

ADR-202 Decisions 1 to 5, after the count Decision 6 asks for.

**S1, the count (E7).** The scanner counts, per shell call whose mrw answer holds the refusal "a read of standard input names nothing else", the flag `--grep` took as its pattern when the command has the shape `--grep -X WORD … -` (`-X` a dash and one letter). The bar is in BACKLOG.md ("Pre-registered for ADR-202"), written before the scanner is changed: **PROCEED** when calls where `--grep` took `-v` are at least 50 in the run without this repository's sessions; otherwise the record is withdrawn and S2 to S5 are not done.

**S2 to S5, the flag.** With standard input `alpha`, `beta`, `gamma`, `beta2`, `delta`, one a line:

| Command | Outcome |
|---------|---------|
| `mrw read --grep beta --invert -` | exit 0; `@@ 1-1` alpha, `@@ 3-3` gamma, `@@ 5-5` delta; no line after a run; nothing recorded |
| the same with `--max-lines 2` | the first two lines served, the rest named as withheld, exit 0 ending `capped:` |
| `mrw read --grep 'a' --invert -` | exit 1, ending `mrw: no match: every line of standard input matched /a/` |
| empty standard input | exit 1, ending `mrw: no match: standard input is empty` |
| `mrw read --invert -` | exit 2: `--invert` needs `--grep` |
| `mrw read --grep beta --invert f.txt`, or a directory, or no path | exit 2: a read of files serves the lines that match; the flag works on `-` |
| `--invert` with `-C 1`, with `--max-matches 1`, with `--json-path` | exit 2, naming the flag it cannot act with |
| `mrw read --grep -v beta -` | exit 2, the refusal ending `an inverted match of a stream is --grep PATTERN --invert -` |
| `mrw read -v f.txt`, `mrw read --grep -v beta f.txt` | as before: ADR-164's sentence |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/adr/BACKLOG.md` | edit | E7's bar, before the scanner; then its reading; the deferred `-o` |
| `docs/bench/effect-study/scan.py`, `test_scan.py`, `README.md`, result files | edit | E7 |
| `cmd/mrw/invert202_test.go` | new | the red test |
| `cmd/mrw/stdin.go` | edit | the inverted read and its refusals |
| `cmd/mrw/main.go` | edit | the flag, its help, the refusal for files |
| `cmd/mrw/foreignflag.go` | edit | the stream pointer for `-v` |
| `cmd/mrw/stdingrep140_test.go` | edit | `-v` on a stream now ends in the pointer |
| `internal/guide/guide.go` | edit | one sentence in the full instructions |
| `scripts/contract.sh` | edit | §346; §345's `-v` assertion |
| `AGENTS.md` | edit | the form, where it is refused |

## Ordered Steps

1. [S1] Pre-register E7's bar in BACKLOG.md and push; add E7 to the scanner with synthetic tests and push; run it; record the reading. Stop here if it says withdraw. [proof: acceptance]
2. [S2] Write `TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch` with every row of the Goal's table and see it RED. [proof: acceptance]
3. [S3] The flag and the inverted read. Mutants, each killed: the lines that match are served; the line after a run is served; the numbers are the runs' own and not the input's; a file is accepted; `-C` is accepted; nothing left exits 0. [proof: mutation]
4. [S4] The pointer in the `--grep -v P -` refusal; `-v` on files unchanged. Mutant, killed: the stream refusal still says there is no inverted match. [proof: mutation]
5. [S5] Contract §346 drives the built binary: the inverted read of a five-line stream, paired with the plain `--grep` of the same stream, the two together serving every line once; and `--invert` on a file refused, exit 2. §345's `-v` row asserts the new pointer. The sentences in `mrw instructions`, `mrw read --help` and `AGENTS.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./cmd/mrw ./internal/guide -run '^(TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch|TestAGrepFlagBeforeThePatternOnStandardInputIsNamed|TestTheInstructionsNameTheInvertedMatch)$' > "${TMPDIR:-/tmp}/adr202-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr202-fence.out")" -eq 3 ] \
  && grep -q '^# 346\. ' scripts/contract.sh \
  && grep -q '^\*\*Measured .* for ADR-202 (E7): PROCEED' docs/adr/BACKLOG.md \
  && grep -q 'an inverted match of a stream is --grep PATTERN --invert -' cmd/mrw/foreignflag.go \
  && grep -q -- '--grep PATTERN --invert -' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch` | `cmd/mrw/invert202_test.go` | every row of the Goal's table but the last two; and that the inverted read and the plain `--grep` of one input serve every line exactly once between them | none | S2, S3 |
| `TestAGrepFlagBeforeThePatternOnStandardInputIsNamed` | `cmd/mrw/stdingrep140_test.go` | `--grep -v P -` ends in the pointer; `-v` on a read of files keeps ADR-164's sentence; the other sixteen flags as before | none | S4 |
| `TestTheInstructionsNameTheInvertedMatch` | `internal/guide/invert202_test.go` | the full instructions name `--grep PATTERN --invert -` and say it is standard input only | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §346 |
| 2 — something selects it | `mrw read` parses `--invert`; `stdinRead` serves it |
| 3 — the caller can discover it | the refusal of `--grep -v P -`, which is what 109 callers typed, names it; `mrw read --help`, `mrw instructions`, `AGENTS.md` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is E7's count, and Decision 6's tally thirty days after the release |

## Mutation Log

## Verification Log

## Invariants

- A read of files or directories behaves exactly as before, `-v` and `--grep -v` included.
- A read of standard input without `--invert` behaves exactly as before.
- No engine package changes; nothing is recorded or licensed by a read of standard input.

## Risks

- The lines are split in `cmd/mrw` with the function the standard-input read already uses for its "no match" check; the test's "every line exactly once" case is what ties the numbering to the engine's.
- `--fold` and `--max-cols` are allowed as on any numbered lines and are covered by one case each, not by the mutants.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- An inverted match on files or directories (permanent: boundary: ADR-164 Decision 5 stands there)
- `-o`, match-only output (deferred: docs/adr/BACKLOG.md)
