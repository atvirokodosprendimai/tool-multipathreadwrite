# Core-first MRW guide ablation — results, 2026-10-31

Registered in [CORE-PREREGISTRATION.md](CORE-PREREGISTRATION.md). Each run is reported as it ended.
No cell was retried, and no run is pooled with another or with the 2026-10-30 pilots.

## Run 1: no eligible comparison

Round 1 stopped after four counted cells, and round 2 was not run, as registered. The comparison
needs all 20 cells solved, so run 1 makes no F-versus-C claim.

- **What stopped it.** The configuration job's F cell reached the 75-second CLI deadline. Its
  record therefore has no cost, and the adapter refused the next launch for unknown prior cost,
  exactly as the budget rule says. Five cells were not run.
- **What it cost.** $0.262079 for the five launches that reported a cost: two canaries and three
  counted cells. The sixth, the timed-out cell, has no cost record, so the run's total is unknown.
- **The CLI changed.** Claude Code reported 2.1.316; the 2026-10-30 pilots ran 2.1.314.

| Cell | Arm | First-request context | CLI turns | Cache read | Cache creation | Output | Cost | Full-guide calls |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| canary | F | 12,346 | 3 | 24,836 | 12,685 | 191 | $0.05515 | 0 |
| canary | C | 4,743 | 4 | 20,390 | 5,791 | 282 | $0.02804 | 0 |
| structural | F | 12,699 | 4 | 41,027 | 15,402 | 1,110 | $0.07683 | 0 |
| structural | C | 5,092 | 5 | 39,185 | 10,790 | 937 | $0.05647 | 1 |
| configuration | C | 4,955 | 6 | 47,752 | 7,590 | 1,044 | $0.04560 | 1 |
| configuration | F | — | timed out | — | — | — | unknown | 0 |

Costs reproduce from the token counts with the diagnosis's coefficients, within 7e-18 per cell.

## What run 1 shows, without a comparison

1. **Prediction 1 missed low.** C's first request held 4,743–5,092 tokens, below the registered
   7,000–7,900; F's held 12,346–12,699. The full guide costs about 7,600 tokens per request, not
   the 5,215 the diagnosis's 5,604-token difference implied. The registration allowed for this
   case: arm B had fewer tools than native, which hid part of the guide's cost.
2. **Prediction 3 held for round 1.** C ran the full `mrw instructions` in both of its counted jobs
   (structural and configuration), each time after reading the core.
3. **The warm-prefix assumption broke.** Cache creation ran 5,791–15,402 tokens per cell, against
   about 2,300–3,900 per cell on 2026-10-30. The canary did not leave the next cell's prefix cached.
   Prediction 2's range was derived from warm-prefix costs; a cost difference measured under these
   conditions is dominated by cache creation.
4. **The benchmark's write guard cost the MRW arms rounds.** In three of four counted cells, both
   arms, the agent built its plan in `$(mktemp -d)/plan.txt`. `bench hook` refused the redirection,
   since it cannot resolve a path holding an expansion, and its refusal named neither form that
   passes: a literal path outside the tree, or a plan piped to `mrw write -`. Structural F and C each
   recovered with a pipe on the next call. Configuration F made no further request before its
   deadline. The native arm has no such guard, so this is a cost the benchmark adds to MRW alone.
5. **A stale recipe cost one round.** Configuration C tried `--then-sh`, which mrw refuses unless
   `MRW_SHELL=on` (ADR-174), then ran the write and the checker as two commands.

## Changed before run 2

`internal/bench/hook.go`: a redirection to a path the hook cannot resolve is still refused, and the
refusal now names the two forms that pass. Test
`TestARedirectionToAPathTheHookCannotResolveNamesTheTwoFormsThatPass` failed before the change and
passes after it; `go test ./internal/bench` and `go vet` pass. Nothing else changed. The amendment in
the registration records this before run 2's first call.

Raw transcripts, stderr, configurations and cell records are kept outside the public checkout,
under the benchmark's private run root.

## Run 2: eligible, and the core-first arm cost more

Run 2 followed Amendment 1: the same design with the guard's new refusal text, fresh directories,
and a peer's two-core test loop running beside it (one-minute load 3.78 and 3.88 on 10 cores at the
round starts).
Both canaries passed in both rounds; all 20 counted cells were solved, with exact trees and
verified final checks. All-in usage: $0.375346 in round 1 and $0.466487 in round 2, canaries
included; round 2's ceiling was $2.624654. Every cell's cost reproduces from its tokens within 7e-18.
Exported records: [core-pilot.json](core-pilot.json); calculation: [core-ablation.py.txt](core-ablation.py.txt).
The registration as run is SHA256 `b6e9e431…`, which includes both amendment entries; run 2's
execution manifest hashed it one entry earlier (`f6ce2b4d…`), before the peer-process note.

| Five-job total | F round 1 | C round 1 | F round 2 | C round 2 |
|---|---:|---:|---:|---:|
| Cost | $0.142737 | $0.207571 | $0.177929 | $0.263914 |
| First-request context | 63,094 | 25,054 | 63,093 | 25,052 |
| CLI turns | 20 | 21 | 24 | 28 |
| Output tokens | 4,018 | 3,499 | 4,652 | 4,219 |
| Cache-read tokens | 262,452 | 155,365 | 320,848 | 230,915 |
| Cache-creation tokens | 19,058 | 39,240 | 24,808 | 49,630 |
| Jobs that ran the full `mrw instructions` | 0 | 3 | 0 | 5 |
| Guard refusals | 3 | 1 | 3 | 3 |
| Refused MRW writes | 0 | 0 | 0 | 0 |

### The registered verdicts

- **Repeat spread, cost: "within repeat spread: no cost result".** The rule counts C as cheaper only
  when it is lower in both rounds. It was higher in both, by $0.064833 and $0.085985. The registration
  defined no verdict for that direction, so the next point is reported as observed, not as a
  registered result: the mean excess, $0.075409, is larger than either arm's spread (F $0.035192,
  C $0.056343).
- **Repeat spread, first-request context: C lower.** 38,040 and 38,041 fewer tokens over five jobs;
  the spreads are 1 and 2 tokens. That is about 7,608 tokens per request: what appending the full guide
  costs at the start of every request.
- **Repeat spread, turns and output tokens: within repeat spread, no result.**
- **Prediction 1, C's first request at 7,000–7,900: missed** in both rounds (4,953–5,092), as in run 1.
- **Prediction 2, C cheaper by $0.004–$0.010 per round: missed** in both rounds; C cost more.
- **Prediction 3, C loads the full guide in at least one job per round: held** in both rounds (3 jobs
  in round 1, 5 in round 2, 9 calls in all).
- **Prediction 4, both arms solve all five jobs in both rounds: held.**

### Why the smaller start cost more

The first-request saving is real and about equal in both rounds; the money went the other way.
The F arm's cache creation stayed at 3,338–4,184 tokens in nine of its ten cells, as on 2026-10-30:
its guide sits in the system prompt, which the canary had already cached, so every later request
read it at the cache-read price. The C arm loaded the guide in 8 of its 10 cells by running
`mrw instructions` (48,905 and 71,803 characters of tool result per round). That text arrives
mid-conversation, so each job paid the cache-creation price for it, then re-read it on every later
turn. Its cache creation was 5,741–12,180 tokens per cell and twice F's in each round. The guide text
it fetched, about 12,000 and 18,000 tokens per round at the diagnosis's 4.14 characters per token,
is most of C's excess of about 20,000 and 25,000 cache-creation tokens; its extra turns and their
tool results are the rest.

The core text is the cause the record points to. It names `mrw write PLANFILE` but not the shape of
a hunk header, and every job needed a hand-built plan, so most C cells fetched the full guide to
learn it. These are accounting facts about these 20 cells, not a causal proof, and the jobs are
synthetic. What follows is a question for the owner, not a change: either the core carries the one
line every plan needs, or "core first, full on demand" is not the cheaper setup on a host that
caches the system prompt.

### Confounds and limits

- **Guard refusals still occurred.** There were 10 in the 20 counted cells, both arms. Run 2 had no
  timeout; one run is too few to say the hint is why.
- **Cache state is not controlled.** Run 1 ran right after a CLI update with the prefix not cached;
  run 2's F cells show it cached. Time of day and service load are uncontrolled.
- **The peer process was running** throughout run 2. Wall-clock time is reported as contended:
  52.4 s and 58.0 s for F, 53.7 s and 64.7 s for C, summed CLI wall per round.
- **Synthetic, ten pairs.** No significance, interval or general savings claim; no guidance, default
  or engine change follows from this.

## Run 3: with ADR-184's core, no full-guide loads and no cost difference

Run 3 followed Amendment 2: run 2's design with an mrw built from the ADR-184 branch (Proposed, not
merged), whose core shows the hunk header in rule 4. The full guide was byte-identical to run 2's.
Both canaries passed in both rounds and all 20 counted cells were solved. All-in usage: $0.314534
in round 1 and $0.277394 in round 2, canaries included; round 2's ceiling was $2.685465. The same
peer test loop ran beside it (one-minute load 5.48 and 3.43 at the round starts). Every cost
reproduces from its tokens within 7e-18. Exported records: [core-pilot-run3.json](core-pilot-run3.json).

| Five-job total | F round 1 | C round 1 | F round 2 | C round 2 |
|---|---:|---:|---:|---:|
| Cost | $0.143115 | $0.147101 | $0.134596 | $0.123706 |
| First-request context | 63,090 | 25,174 | 63,089 | 25,176 |
| CLI turns | 19 | 18 | 19 | 18 |
| Output tokens | 3,777 | 3,332 | 3,536 | 3,677 |
| Cache-read tokens | 249,291 | 96,527 | 247,596 | 97,322 |
| Cache-creation tokens | 20,085 | 26,014 | 18,600 | 19,283 |
| Jobs that ran the full `mrw instructions` | 0 | 0 | 0 | 0 |

### The registered verdicts

- **Prediction 1, C's first request at 4,960–5,110: missed, narrowly.** Four of five jobs landed in
  range in each round; the structural job's 5,118 and 5,116 are above it. F's repeated run 2's
  12,560–12,700.
- **Prediction 2, C loads the full guide in at most one job per round: held.** It loaded it in no
  job in either round, against 8 of 10 cells in run 2.
- **Prediction 3, C lower by the repeat-spread rule, $0.005–$0.030 per round: missed.** C cost
  $0.003986 more in round 1 and $0.010889 less in round 2. The mean saving, $0.003452, is inside both
  spreads (F $0.008519, C $0.023394): "within repeat spread: no cost result".
- **Prediction 4, both arms solve all five jobs in both rounds: held.**
- **Repeat spread, other measures:** first-request context, C lower by 37,916 and 37,913 tokens
  (spreads 1 and 2); turns, C lower by one in each round (spreads 0); output tokens, within spread.

### What changed from run 2, and what did not

Comparing runs is descriptive only; they are not pooled. With the header in the core, C stopped
fetching the full guide and its cache creation fell from 39,240 and 49,630 tokens to 26,014 and
19,283. Its five-job cost fell from $0.207571 and $0.263914 to $0.147101 and $0.123706, while F's
stayed in the same band ($0.142737 and $0.177929 in run 2, $0.143115 and $0.134596 here). So the
core-first setup no longer costs more on these jobs; it does not yet cost measurably less. C's
smaller prefix saves about 150,000 cache-read tokens per round, about $0.015, and the remaining cache
creation gives most of that back: in round 1 the structural job alone created 9,250 tokens.

What this does and does not support:

- ADR-184's change did what it was for in these 20 cells: no core cell needed the full guide.
- It does not show core-first is cheaper. A difference this size needs more rounds than two to
  leave the repeat spread, and this registration fixed two.
- Same limits as run 2: synthetic jobs, uncontrolled cache and service state, a peer process, and
  wall-clock times reported as contended (49.0 s and 47.5 s for F, 50.1 s and 49.1 s for C).
