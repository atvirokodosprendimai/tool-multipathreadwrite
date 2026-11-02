# Host Read with the read hook versus native: results, 2026-11-02

Registered in [HOOK-PREREGISTRATION.md](HOOK-PREREGISTRATION.md), pushed at `f54e7b81` before the
first call. This is run 9 (A vs R), two rounds of 12 launches each. Both canary pairs passed in both
rounds, and all 20 counted cells were solved with exact trees and verified final checks.

- **Spend:** $0.279263 and $0.264093 all-in, $0.543356 in total.
- **Binary:** mrw `v1.83.0-dev+94d125a8`. The core guide's sha256 is `4ba462e0…`, the same as run 8's.
- **Load at launch:** 8.72 and 9.29, with no peer build. Round 2's file records 10.16, taken when it
  was relaunched; the run then waited until load was at most 10.
- **Two launches stopped before any call, $0 spent, each kept beside its round:**
  - round 1, `setup-failure-1`: the round directories did not exist;
  - round 2, `gate-refusal-1`: its adapter carried no `max_usd` under $3 minus round 1's spend.
- **Records:** [hook-pilot.json](hook-pilot.json).

| Five-job total | Round 1 | Round 2 |
|---|---:|---:|
| Native (A) cost | $0.103759 | $0.109355 |
| MRW, host Read and hook (R) cost | $0.148403 | $0.134898 |
| A minus R | $-0.044644 | $-0.025543 |
| A / R cache creation | 15,427 / 21,853 | 16,267 / 20,925 |
| A / R CLI turns | 16 / 22 | 18 / 20 |
| A / R output tokens | 3,155 / 4,415 | 3,121 / 3,592 |
| R host `Read` calls | 0 | 0 |
| R hook stderr | empty | empty |

## The registered verdicts

- **Cost: native lower.** It is lower by $0.044644 and $0.025543, a mean of $0.035094. That mean is
  larger than both spreads (A $0.005596, R $0.013505).
- **Cache creation, turns and output tokens: native lower**, by the same rule.
- **Prediction 1, every cell solved: held.**
- **Prediction 2, the hook is exercised: missed.** R called the host `Read` in 0 of its 10 cells, so
  the hook never ran. The mechanism counts are all zero:
  - 0 overlapping reads;
  - 0 hook-licensed writes;
  - 0 refused unread writes;
  - 10 unknown calls per round, which are `--grep` reads and plan-file writes.

  By the registration, predictions 3–5 are therefore **untested, not refuted**.
- **Prediction 3, mechanism: untested.**
- **Prediction 4, native's lead $0.000–$0.015 a round: missed.** The lead was $0.044644 and
  $0.025543. Because R never used `Read`, this measures R's unused `Read` and `Grep` definitions and
  run-to-run variance, not the hook.
- **Run 10 (C vs R): held for the owner's decision.** The registration runs it after run 9. Its
  question is what the host-Read bundle buys, and run 9 shows the agent does not take the bundle up
  under the core guide, so as registered it could only price unused tool definitions.

## Why R did not read with the host

The appended core guide's first rule is "Use mrw for every file read, edit and create"
(`mrw instructions --core`). Every R cell read with `mrw read` or `mrw read --grep`, as C does. ADR-191's hook only
saves the re-read the recheck counted when the agent uses the host `Read` first. That happened there
because that setup's guide said nothing against it. This one does.

## Against run 8, descriptively

These runs are not pooled. Native's five-job cost stayed in its band: $0.101–$0.103 in run 8,
$0.104–$0.109 here. Run 8's C cost $0.114–$0.121; R here cost $0.135–$0.148. R's cache creation runs
about 3,000–4,000 tokens a round above run 8's C. That is consistent with R's extra tool definitions
(`Read`, `Grep`) on every request, but runs 8 and 9 are different runs, so this does not separate
the definitions from other variance.

## The classifier, as run

`scripts/bench_mechanism.py` keys cells by file basename, so one call over both rounds' transcripts
keeps only the second round's cells; each round was therefore classified on its own. Every host
`Read` count above also comes from the driver's own `by_kind` metrics, which agree: 0 in each round.
