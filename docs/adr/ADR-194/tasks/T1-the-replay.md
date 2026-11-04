# Task ADR-194-T1: replay the Stage 8 transcripts and price the narrow licence against the pre-registered bar

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `scripts/licence194_replay.py`; `scripts/test_licence194_replay.py`; `docs/bench/licence194/replay.json`, `manifest.txt` and `REPLAY.md` with the verdict
**Consumes:** none
**Data dependency:** needs the Stage 8 pilot transcripts, local to the machine that ran them (`docs/bench/stage-8/RESULTS.md`, `transcripts:` line); the sign-off line names the directory, the file count and the manifest's sha256
**Proof map:** v1
**Rests-on:** `the replay carries a licence as the rule says`, `each licence source is modelled`, `the verdict follows from W, R and U by the pre-registered rule`, `the committed result is what the corpus gives`

## Goal

Count, on real agent sessions, how many landed hunks ADR-194's rule would refuse, and apply the criterion pre-registered in `docs/adr/BACKLOG.md` (From ADR-194) before this script existed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/test_licence194_replay.py` | new | the red test: synthetic transcripts with known W, R and U |
| `scripts/licence194_replay.py` | new | the replay; `--verify` re-runs it on the corpus and compares with a committed result |
| `docs/bench/licence194/replay.json`, `manifest.txt`, `REPLAY.md` | new | W, R, U, the verdict, each R and U hunk, the corpus manifest |
| `docs/bench/README.md` | edit | a row for the replay |

## Ordered Steps

1. [S1] Write `scripts/test_licence194_replay.py` and see it RED (the script does not exist). Synthetic transcripts in Claude Code's stream-json shape, each with known W, R and U (a ranged read `A-B` prints `@@ A-(B+1)`, ADR-161's line after):
   - a ranged read of 3-4, a write at 3, a write at 6: R = 1; a write at 5 instead: R = 0;
   - the same with a whole read first: R = 0;
   - a ranged read of 10-20, an insert of 3 lines after 12, then a write at 24 (old 21): R = 0; at 25 (old 22): R = 1;
   - a ranged read of 3-8, a delete of 4-5, then a write at 4 (old 6): R = 0; at 8 (old 10): R = 1;
   - a create, then any write: R = 0;
   - a SEARCH/REPLACE hunk on an unread file: R = 0; a multi-line one whose line after was not served: R = 1;
   - a `--sub` write, and a `--force` write at 3: R = 0; then a `--force` replace of 3-4 whose line 5 was not served: R = 1 (ADR-052 holds under `--force`);
   - an unlink of a file read 3-4 and then written: R = 1;
   - a pattern-addressed hunk the transcript cannot place: U = 1, and a later hunk on the same file: U = 2, until a whole read;
   - a `--quiet` read, then a write: U = 1; a read piped into `sed -n 1,80p` whose output ends inside its last `@@` span, then a write: U = 1;
   - an unplaceable hunk, then a ranged read, then a hunk on that file: U = 2; then a whole read and a hunk: U stays 2;
   - a refused write: not in W.
   The verdict function on (W, R, U) = (100, 6, 0) is STOP, (100, 3, 1) GO, (100, 3, 3) INCONCLUSIVE, (100, 0, 21) INCONCLUSIVE, (40, 0, 0) INCONCLUSIVE.
2. [S2] The replay walks each attempt's `mrw` Bash calls and their results in order and keeps, per file, the licence and its numbering as BACKLOG.md's criterion says. [proof: mutation] Mutants, each killed:
   - the carry ignores inserted lines, so the insert case goes red;
   - a SEARCH/REPLACE hunk's quoted lines counted against the ledger, so the quoted case goes red;
   - an uncertain file made certain again by any read, so the second U case goes red.
3. [S3] The verdict follows the criterion exactly. [proof: mutation] Mutant: the STOP bound read as `R > W/10`, killed by (100, 6, 0).
4. [S4] Run it on the Stage 8 transcripts. Commit `manifest.txt` (each file's name and sha256), `replay.json` (W, R, U, verdict, by model and cell, each R and U hunk with its reason) and `REPLAY.md` (the verdict line first, `verdict: GO|STOP|INCONCLUSIVE W=… R=… U=…`). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
python3 -m unittest discover -s scripts -p 'test_licence194_replay.py' 2>&1 | tee /tmp/acc194t1.out \
  && grep -qE '^OK' /tmp/acc194t1.out \
  && ! grep -qE 'Ran 0 tests' /tmp/acc194t1.out \
  && python3 scripts/licence194_replay.py --verify docs/bench/licence194/replay.json \
  && jq -e '(.W>=50 and .U<=.W/5) as $enough | (if ($enough|not) then "INCONCLUSIVE" elif .R>.W/20 then "STOP" elif (.R+.U)<=.W/20 then "GO" else "INCONCLUSIVE" end) == .verdict' docs/bench/licence194/replay.json >/dev/null \
  && v=$(jq -r '"verdict: \(.verdict) W=\(.W) R=\(.R) U=\(.U)"' docs/bench/licence194/replay.json) \
  && [ "$(head -1 docs/bench/licence194/REPLAY.md)" = "$v" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal cmd
```

`--verify` recomputes the manifest from the corpus directory the result names, exits 1 when any file's sha256 differs or a file is missing or added, re-runs the replay, and exits 1 when its W, R, U or verdict differ from the committed result.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `test_licence194_replay` | `scripts/test_licence194_replay.py` | known W, R, U on synthetic transcripts for each licence source; the verdict's bounds | none | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the script and its test |
| 2 — something selects it | run by hand on the named corpus; T2's fence reads its verdict |
| 3 — the caller can discover it | `docs/bench/README.md` |
| 4 — it is used | T2 starts only on its GO, or on Zy's recorded override |

## Mutation Log

## Verification Log

## Invariants

- No Go code changes in this task.
- The criterion in BACKLOG.md is not edited after the first run.

## Risks

- The replay places a hunk wrongly and miscounts R: mitigated by the synthetic cases, and by U, which counts what it cannot place, and every later hunk on that file, instead of guessing.

## Stop Condition

The fence exits 0 with any verdict. A STOP or INCONCLUSIVE verdict is reported to Zy and T2 stays pending. If Zy decides T2 goes ahead anyway, `replay.json` gains `"override": {"by": "Zy", "date": …, "said": "<the words>"}` in a commit of its own; the verdict is not changed.

## Out of Scope

- Transcripts from other stages (permanent: boundary: the criterion names the Stage 8 pilot; a wider corpus is a new pre-registration)
