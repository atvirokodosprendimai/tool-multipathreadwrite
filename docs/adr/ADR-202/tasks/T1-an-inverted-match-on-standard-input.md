# Task ADR-202-T1: an inverted match on standard input

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** E7 in `docs/bench/effect-study/` and its reading in BACKLOG.md; `mrw read --invert`; `read.Options.NoReplaceNote`; the `-v` sentence; contract §348; the sentences in `mrw instructions`, `mrw read --help`, `AGENTS.md` and `README.md`
**Consumes:** `stdinRead` (`cmd/mrw/stdin.go`), `read.Run`'s numbered ranges, `foreignFlags` and `readFlagPatternRefusal` (`cmd/mrw/foreignflag.go`), `lines.Split`, the effect-study scanner
**Data dependency:** local transcripts — S1 reads `~/.claude/projects` on the machine it runs on and prints aggregates only; S2 to S5 are hermetic
**Proof map:** v1
**Rests-on:** `the lines that do not match are served, numbered as in the input`, `nothing left serves no line and exits 1`, `the flag is refused everywhere else, before anything is read`, `every -v refusal points at the form`, `the engine leaves the replace note out when told to`, `the other engine packages are untouched`

## Goal

ADR-202 Decisions 1 to 6, after the count Decision 7 asks for.

**S1, the count (E7).** A replication of a tally already seen, by a committed scanner with a stated rule (BACKLOG.md, "Pre-registered for ADR-202", with its correction of 2026-11-19). Measured UNCERTAIN: `v` 108, `v_multi` 93. By the rule that went to Zy, who accepted the record on the 108 (2026-11-19). S2 to S5 follow.

**S2 to S5, the flag.** Fixture A is `alpha`, `beta`, `gamma`, `beta2`, `delta`, one a line. Fixture B is `one`, `two`, `skip`, `four`.

| Command, standard input | Outcome |
|---------|---------|
| A: `mrw read --grep beta --invert -` | exit 0; exactly `@@ 1-1` alpha, `@@ 3-3` gamma, `@@ 5-5` delta; the served numbers are every line of A that does not match and no other; the ledger of the root holds nothing afterwards |
| B: `--grep skip --invert -` | `@@ 1-2` one, two; `@@ 4-4` four; no `-- note:` line |
| A: with `--max-lines 2` | lines 1 and 3 served, line 5 named withheld, exit 0 ending `mrw: capped:` |
| A: with `--max-chars 1` | no numbered line, exit 1 ending `mrw: incomplete:` and naming `--max-chars 1 reached` |
| A: with `-N` | the three lines with no number, the `@@` lines kept |
| three `x 1`, `x 2`, `x 3` lines and a `skip`: `--grep skip --invert --fold -` | the run folded as `--fold` folds numbered lines |
| a 40-character line and a `skip`: `--grep skip --invert --max-cols 10 -` | the line cut from column 1 |
| A: `--grep a --invert -` | stdout: the header and `!! every line matches /a/`, no numbered line; exit 1 ending `mrw: no match: every line of standard input matched /a/` |
| empty: `--grep a --invert -` | stdout: the header and `!! standard input is empty`; exit 1 ending `mrw: no match: standard input is empty` |
| `a\r\nskip\r\nb\r\n`; `a\rskip\rb`; `a\nskip\nb` with no final newline; `\n` with `--grep x` | lines 1 and 3 in the first three; line 1, empty, in the last |
| `mrw read --invert -` | exit 2, the sentence saying `--invert` needs `--grep` |
| `--grep beta --invert` with `f.txt`, a directory, no argument, `./-`, `-:1-2`, `- f.txt` | exit 2, standard input not read; for `f.txt` the sentence says a read of files serves the lines that match; each names `--grep PATTERN --invert -` |
| `--grep beta --invert --sub X f.txt`, `--invert --count f.txt` | exit 2 by `--invert`'s refusal: no plan, no count |
| `--grep beta --invert -` with `-C 1`, `--max-matches 1`, `--no-comments`, `--quiet` | exit 2, the sentence naming the flag |
| `--grep --invert beta -`, `--grep --invert beta f.txt` | exit 2: `--grep took "--invert" as its pattern` |
| `--grep='(?:--invert)' -` on a line holding `--invert` | that line served |
| `--grep -v beta -`, `--grep beta -v -`, `-v --grep beta -`, `-v f.txt`, `--grep -v beta f.txt` | `--grep -v beta f.txt` exits 1 as today, the other four exit 2; each carries `; an inverted match of a stream is --grep PATTERN --invert -` after ADR-164's sentence |
| `mrw read --help` | names `--invert` and that it reads standard input |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `docs/adr/BACKLOG.md` | edit | E7's reading; the deferrals |
| `docs/bench/effect-study/scan.py`, `test_scan.py`, `README.md`, result files | edit | E7 |
| `cmd/mrw/invert202_test.go` | new | the red test |
| `internal/read/note202_test.go` | new | the engine option |
| `internal/guide/invert202_test.go` | new | the instructions |
| `internal/read/read.go` | edit | `NoReplaceNote` |
| `cmd/mrw/stdin.go`, `cmd/mrw/main.go` | edit | the flag, the inverted read, the refusals before dispatch |
| `cmd/mrw/foreignflag.go` | edit | the `-v` sentence |
| `cmd/mrw/stdingrep140_test.go` | edit | `-v` now ends in the pointer |
| `internal/guide/guide.go`, `.claude/skills/mrw/SKILL.md` | edit | one sentence in the full instructions; the skill only if its text moves |
| `scripts/contract.sh` | edit | §348 |
| `AGENTS.md`, `README.md` | edit | the form, where it is refused, outside the recipe block |

## Ordered Steps

1. [S1] Add E7 to the scanner with synthetic tests, push, run it, record the reading. Stop here unless it says PROCEED. [proof: acceptance]
2. [S2] Write the three new tests with every row of the Goal's table, change the `-v` expectation in `TestAGrepFlagBeforeThePatternOnStandardInputIsNamed`, and see all four RED. [proof: acceptance]
3. [S3] `NoReplaceNote`, the flag and the inverted read. Mutants, each killed: the lines that match are served; the line after a run is served; the note is printed; every line matched serves the whole input; empty input exits 0; a file is accepted; a file is accepted beside `--sub`; `-C` is accepted; `--max-matches` is accepted; `--invert` works with no `--grep`. [proof: mutation]
4. [S4] The `-v` sentence. Mutant, killed: the sentence ends where ADR-164 left it. [proof: mutation]
5. [S5] Contract §348 drives the built binary: fixture B's inverted read, its exact numbered lines and no note, paired with `--invert` on a file refused, exit 2, nothing served. The sentences in `mrw instructions` (full only), `mrw read --help`, `AGENTS.md`, `README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./cmd/mrw ./internal/guide ./internal/read -run '^(TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch|TestAGrepFlagBeforeThePatternOnStandardInputIsNamed|TestTheInstructionsNameTheInvertedMatch|TestARunCanLeaveOutTheReplaceNote)$' > "${TMPDIR:-/tmp}/adr202-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr202-fence.out")" -eq 4 ] \
  && python3 -I docs/bench/effect-study/test_scan.py > /dev/null \
  && grep -q '^# 348\. ' scripts/contract.sh \
  && grep -q '^\*\*Accepted on the 108 by Zy, 2026-11-19' docs/adr/BACKLOG.md \
  && grep -q -- '--grep PATTERN --invert -' AGENTS.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/apply internal/seen internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/apply internal/seen internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnInvertedMatchServesTheLinesOfStandardInputThatDoNotMatch` | `cmd/mrw/invert202_test.go` | every row of the Goal's table, each refusal by its sentence and not only its exit code; the served numbers against the lines the test itself finds not matching, with `regexp` over `lines.Split` | none | S2, S3, S4 |
| `TestAGrepFlagBeforeThePatternOnStandardInputIsNamed` | `cmd/mrw/stdingrep140_test.go` | exists today; its `-v` expectation changes to the sentence with the pointer, on a stream and on a file, which is what goes red | none | S2, S4 |
| `TestTheInstructionsNameTheInvertedMatch` | `internal/guide/invert202_test.go` | the full instructions name `--grep PATTERN --invert -` and say it is standard input only; the core does not hold it | none | S5 |
| `TestARunCanLeaveOutTheReplaceNote` | `internal/read/note202_test.go` | with `NextLine` off, a two-line range before the end prints the note; with `NoReplaceNote` it does not, and nothing else in the answer differs | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests and §348 |
| 2 — something selects it | `mrw read` parses `--invert`; `stdinRead` serves it |
| 3 — the caller can discover it | every refusal a caller gets for typing `-v` names it; `mrw read --help`, `mrw instructions`, `AGENTS.md`, `README.md` |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is E7's count, and the tally of BACKLOG.md "Pre-registered for ADR-202's use" |

## Mutation Log

## Verification Log

## Invariants

- A read of files or directories serves what it served before. Its refusals are the same but for two: `-v` gains the pointer, and `--grep --invert P` is now named by ADR-140's refusal where it was a missed pattern.
- A read of standard input without `--invert` serves what it served before, the line after each match included.
- `internal/plan`, `internal/apply`, `internal/seen`, `internal/check`, `internal/state` are untouched; `internal/read` gains one option whose zero value is today's behaviour.

## Risks

- The fence's `# 348.` and `PROCEED` clauses are markers: §348 is proved by running `scripts/contract.sh`, a gate, and the count by the committed result file the BACKLOG line cites.
- `--max-chars` and `--fold` have one row each and no mutant.
- A stream with very many runs under a small `--max-lines` prints one `WITHHELD` line per run that did not fit, as the engine does for any spans it withholds.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes, and the record is Accepted by Zy.

## Out of Scope

- An inverted match on files or directories (deferred: docs/adr/BACKLOG.md)
- `-o`, match-only output (deferred: docs/adr/BACKLOG.md)
