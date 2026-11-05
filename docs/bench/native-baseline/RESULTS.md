# Native performance baseline — results, 2026-10-29

These are the runs of [PREREGISTRATION.md](PREREGISTRATION.md), agreed by Zy on 2026-10-29 and merged in #558 before any measurement. They run on macOS arm64 only.

**Run 2 is the result.** Run 1, kept below, is invalid for a verdict: the review of #565 found five ways its driver did not measure what the pre-registration registers (listed under "Run 1"). The driver was fixed in `c39f950e` and `7103c4ed`, and run 2 was taken with the fixed driver. No bar, workload or size was changed between the runs.

It says nothing about agent task time, tokens or cost; that is item 4.

## What ran (run 2)

| | |
|---|---|
| A | release **v1.78.0 (06075f7)** `mrw-darwin-arm64`, checked against the release's `SHA256SUMS.txt`: `cf4050beacb4390b539fbeb89087875b8468e2a3d97cb2aca20ee10116e0bf38` |
| B | a source build of `fc70c69c` with the release flags (`CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=v1.78.0-dev+fc70c69c"`): `d5c89c7ce69f93a5a5c8e2d732b17e2c0855472d008795a85f8c1fb68ae61733` |
| driver | `cmd/nativebench` at `7103c4ed`; `main.go` sha256 `54c08c50f7c80fe83cef6fde117269a67f6b47534bb59c75b52e771df9870263`. Fixtures come from seed 20261029 |
| R3 "repo" tree | `git archive v1.78.0`, 4,554 files, 32 MB |
| machine | Mac17,2, Apple M5, 10 cores, 32 GB, on AC power; macOS 27.0.1 (26A434); go1.27.3 |
| when | 2026-10-29 12:33:24Z – 12:35:14Z: the driver took 110 s of the 15-minute budget. No cell was cut by the budget (`cut_by_budget` is unset in every row of [raw/run2/cells.jsonl](raw/run2/cells.jsonl)) |
| load | every cell started below the 2.0 gate. The 1-minute load before and after each cell is in `cells.jsonl`: 1.66–1.91 before, 1.66–2.54 after. The one reading over 2.0 is after R3 "repo", the parallel walk measured itself |
| samples | 1,080: 18 cells × 2 binaries × 30 timed runs, after 5 discarded warm-ups, each written as it was taken to [raw/run2/samples.jsonl](raw/run2/samples.jsonl) |

The largest sizes (R1 1,000,000 lines, W1 500 hunks / 50 files) run last, as registered, so a budget cut would have dropped them first.

## Timings, run 2 (ms; RSS from rusage)

| workload | size | binary | n | failed | p25 ms | p50 ms | p75 ms | p90 ms | max ms | cpu p50 ms | max rss MB | stdout B |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| R1 | 100 | A | 30 | 0 | 9.17 | 9.88 | 10.06 | 10.16 | 10.72 | 4.45 | 14.3 | 1266 |
| R1 | 100 | B | 30 | 0 | 9.05 | 9.87 | 9.96 | 10.19 | 10.68 | 4.45 | 13.8 | 1266 |
| R1 | 10000 | A | 30 | 0 | 9.87 | 9.99 | 10.05 | 10.10 | 11.05 | 4.67 | 15.4 | 1275 |
| R1 | 10000 | B | 30 | 0 | 9.77 | 9.90 | 10.04 | 10.07 | 10.94 | 4.70 | 14.8 | 1275 |
| R1 | 1000000 | A | 30 | 0 | 33.78 | 34.23 | 34.99 | 35.46 | 37.65 | 31.28 | 107.3 | 1330 |
| R1 | 1000000 | B | 30 | 0 | 33.79 | 34.04 | 34.94 | 35.66 | 39.99 | 31.31 | 92.3 | 1330 |
| R2 | 1 | A | 30 | 0 | 9.72 | 9.94 | 10.04 | 10.14 | 10.20 | 4.54 | 14.5 | 1278 |
| R2 | 1 | B | 30 | 0 | 9.87 | 9.90 | 10.05 | 10.09 | 10.76 | 4.47 | 14.0 | 1278 |
| R2 | 10 | A | 30 | 0 | 10.01 | 10.15 | 11.06 | 11.50 | 16.05 | 5.25 | 15.9 | 12782 |
| R2 | 10 | B | 30 | 0 | 9.89 | 10.02 | 10.94 | 11.16 | 12.01 | 5.21 | 15.0 | 12782 |
| R2 | 100 | A | 30 | 0 | 17.95 | 18.11 | 18.97 | 19.04 | 19.73 | 13.69 | 20.4 | 127888 |
| R2 | 100 | B | 30 | 0 | 17.93 | 18.00 | 18.89 | 19.00 | 20.14 | 13.63 | 20.3 | 127888 |
| R3 | repo | A | 30 | 0 | 101.74 | 102.20 | 103.79 | 107.05 | 119.81 | 381.19 | 26.1 | 1612 |
| R3 | repo | B | 30 | 0 | 101.46 | 102.36 | 103.71 | 109.09 | 148.92 | 382.59 | 25.4 | 1612 |
| R3 | deep3000 | A | 30 | 30 | 50.09 | 50.28 | 50.46 | 50.84 | 58.02 | 282.20 | 22.0 | 0 |
| R3 | deep3000 | B | 30 | 30 | 50.10 | 50.29 | 50.61 | 50.70 | 101.09 | 282.19 | 21.6 | 0 |
| W1 | 1h/1f | A | 30 | 0 | 11.69 | 11.76 | 11.98 | 12.48 | 12.88 | 6.10 | 15.0 | 133 |
| W1 | 1h/1f | B | 30 | 0 | 11.60 | 11.74 | 11.83 | 12.03 | 12.92 | 6.09 | 14.7 | 133 |
| W1 | 50h/10f | A | 30 | 0 | 14.63 | 14.86 | 15.76 | 15.93 | 21.16 | 9.76 | 19.0 | 2139 |
| W1 | 50h/10f | B | 30 | 0 | 14.79 | 14.94 | 15.72 | 16.05 | 16.78 | 9.94 | 18.6 | 2139 |
| W1 | 500h/50f | A | 30 | 0 | 30.05 | 30.96 | 31.89 | 32.06 | 40.93 | 30.37 | 24.1 | 18710 |
| W1 | 500h/50f | B | 30 | 0 | 30.23 | 30.92 | 31.19 | 31.95 | 33.44 | 30.52 | 23.4 | 18710 |
| W2 | 10 | A | 30 | 0 | 15.99 | 16.82 | 17.13 | 18.09 | 24.66 | 7.74 | 16.3 | 387 |
| W2 | 10 | B | 30 | 0 | 15.99 | 16.75 | 17.70 | 17.83 | 19.14 | 7.87 | 15.9 | 387 |
| W2 | 1000 | A | 30 | 0 | 22.70 | 23.05 | 23.58 | 24.22 | 26.15 | 15.96 | 22.7 | 30993 |
| W2 | 1000 | B | 30 | 0 | 22.68 | 22.77 | 23.09 | 23.76 | 25.76 | 15.79 | 22.5 | 30993 |
| L1 | 0 | A | 30 | 0 | 10.09 | 10.22 | 11.09 | 14.24 | 19.93 | 4.86 | 15.3 | 1276 |
| L1 | 0 | B | 30 | 0 | 10.09 | 10.28 | 11.18 | 13.43 | 18.76 | 4.92 | 14.7 | 1276 |
| L1 | 1000 | A | 30 | 0 | 10.08 | 10.22 | 10.91 | 11.21 | 16.32 | 5.74 | 18.1 | 1273 |
| L1 | 1000 | B | 30 | 0 | 10.05 | 10.26 | 11.05 | 11.56 | 12.20 | 5.67 | 17.7 | 1273 |
| L1 | 10000 | A | 30 | 0 | 14.65 | 14.94 | 15.79 | 16.55 | 17.47 | 11.44 | 24.6 | 1277 |
| L1 | 10000 | B | 30 | 0 | 14.71 | 14.96 | 15.60 | 16.01 | 17.63 | 11.43 | 24.4 | 1277 |
| C1 | first | A | 30 | 0 | 9.62 | 9.75 | 10.60 | 10.85 | 12.16 | 4.84 | 15.0 | 1278 |
| C1 | first | B | 30 | 0 | 9.57 | 9.79 | 10.60 | 10.76 | 15.21 | 4.93 | 14.7 | 1278 |
| C1 | warm | A | 30 | 0 | 9.82 | 9.95 | 10.12 | 10.78 | 10.99 | 4.71 | 15.2 | 1278 |
| C1 | warm | B | 30 | 0 | 9.81 | 10.01 | 10.10 | 10.33 | 14.02 | 4.68 | 14.8 | 1278 |

R3 "deep3000" exits 1 in every run, by design. The tree is BenchmarkWalkDeep131's shape and matches nothing, so `mrw read --grep` reports `no match`. As registered, each non-zero exit is counted in `failed` and not dropped.

A and B agree within noise in every cell: the commits after v1.78.0 changed no served code path.

Every process pays a floor of about 10 ms (R1 at 100 lines, C1, L1 at 0): process start and state-directory work, not the workload.

## Against the registered bar (run 2)

**Candidate bottlenecks.** A candidate is a cell whose p50 exceeds 100 ms, or whose time grows faster than linearly between adjacent sizes (time ratio over size ratio above 1.5).

- **R3 "repo" is the one candidate.** It runs the registered command, `mrw read --grep "func Run" .`, over the v1.78.0 tree. Its p50 is 102.20 ms (A) and 102.36 ms (B).
- **Nothing grows super-linearly.** Time ratio over size ratio for each adjacent pair, A (B within 0.002):
  - W1 (50,10)→(500,50): 0.21, the largest;
  - R2 10→100 specs: 0.18;
  - L1 1,000→10,000 entries: 0.15;
  - R2 1→10: 0.10;
  - R1 10,000→1,000,000 lines: 0.034;
  - W1 (1,1)→(50,10): 0.025;
  - W2 10→1,000 sites: 0.014;
  - R1 100→10,000: 0.010.
- **Every other cell** has a p50 at or below 51 ms; the highest are R3 "deep3000" (50.3 ms) and R1 at 1,000,000 lines (34.2 ms).

**The candidate, profiled before any code is changed** ([raw/run2/r3.cpu.pprof](raw/run2/r3.cpu.pprof), [raw/run2/r3.mem.pprof](raw/run2/r3.mem.pprof), top nodes in [raw/run2/r3.top.txt](raw/run2/r3.top.txt)). The profile is of `BenchmarkR3Repo` (`cmd/nativebench/bench_test.go`), 30 iterations of the same walk and the same served read in process, writing to `io.Discard`. It was taken at a 1-minute load of 7.0–7.2, above the gate, so its timing (151.8 ms/op) is not a result. Its shape is the finding:

- **76% of CPU is in syscalls** (`syscall.rawsyscalln`).
- `internal/read.readBudget` takes 58% cumulatively, and within it `regular.Open` (`os.OpenFile`) 48%: one open per walked file, 4,554 of them.
- `filepath.WalkDir` takes 19%, `os.Lstat` 9% and `rooted.(*Resolver).Resolve` 9% (overlapping).
- `io.ReadAll` takes 7.4%.
- The regexp is under 1%: it is not among the top 80 cumulative nodes.
- The walk is parallel: CPU p50 is 381 ms against a wall p50 of 102 ms.

This is the same shape run 1's profile had. The cost is the filesystem's per-file open, not mrw's own code. No code change is made here. Any optimisation goes back to Zy first, and is kept only if the registered bar holds: a p50 gain of 20% or more with non-overlapping p25–p75 intervals, every gate passing, and no guard, licence or bound changed.

## Allocations, run 2 (`go test -bench -benchmem -count 5`)

These ran after the timed cells at a 1-minute load of 4.3–6.6, above the gate; the load at the start of each benchmark is the `#  load` line before it. The pre-registration gates workloads, not these benchmarks, and B/op and allocs/op do not depend on load. Their ns/op are reported, not judged. Raw output: [raw/run2/benchmem.txt](raw/run2/benchmem.txt).

```
#  load { 4.27 3.84 5.74 }
BenchmarkWalkDeep131-10    	      26	  68505809 ns/op	 9741512 B/op	   78714 allocs/op
BenchmarkWalkDeep131-10    	      27	  74178756 ns/op	 9844117 B/op	   78716 allocs/op
BenchmarkWalkDeep131-10    	      21	  57425744 ns/op	 9744438 B/op	   78706 allocs/op
BenchmarkWalkDeep131-10    	      21	  83101982 ns/op	 9800867 B/op	   78713 allocs/op
BenchmarkWalkDeep131-10    	      21	  59707258 ns/op	 9696963 B/op	   78703 allocs/op
#  load { 5.72 4.17 5.83 }
BenchmarkResolveADeepFile-10    	   33490	     37504 ns/op	   16784 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   28639	     38889 ns/op	   16912 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   31834	     37675 ns/op	   16912 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   31933	     38039 ns/op	   16784 B/op	     146 allocs/op
BenchmarkResolveADeepFile-10    	   31410	     37398 ns/op	   16912 B/op	     146 allocs/op
#  load { 6.55 4.41 5.90 }
BenchmarkR1-10    	    4416	    268487 ns/op	  889575 B/op	     239 allocs/op
BenchmarkR1-10    	    4423	    273165 ns/op	  889570 B/op	     239 allocs/op
BenchmarkR1-10    	    4650	    267820 ns/op	  889584 B/op	     239 allocs/op
BenchmarkR1-10    	    4484	    267743 ns/op	  889566 B/op	     239 allocs/op
BenchmarkR1-10    	    4480	    267811 ns/op	  889585 B/op	     239 allocs/op
#  load { 6.30 4.43 5.89 }
BenchmarkW1-10    	     343	   3569280 ns/op	 4057482 B/op	    6366 allocs/op
BenchmarkW1-10    	     325	   3636004 ns/op	 4057120 B/op	    6366 allocs/op
BenchmarkW1-10    	     320	   3656244 ns/op	 4053958 B/op	    6365 allocs/op
BenchmarkW1-10    	     319	   3692018 ns/op	 4060993 B/op	    6366 allocs/op
BenchmarkW1-10    	     333	   3675357 ns/op	 4060247 B/op	    6366 allocs/op
```

BenchmarkW1 now resets its timer after building the fixture, so its figures no longer include setup: 6,366 allocs/op against run 1's 6,462–6,470.

## Where run 2 departs from the text

- **Peak RSS** comes from `ProcessState.SysUsage()`'s `Maxrss`, the kernel rusage figure `/usr/bin/time -l` prints, not from running `/usr/bin/time` itself.
- **The wait for the load gate was outside the driver.** A bounded loop (at most 180 checks, one a minute) waited 139 minutes, from about 10:14Z, for a 1-minute load below 2.0; the driver then ran in 110 s. The pre-registration's 15-minute budget is counted from the driver's start, not from the start of that wait. The gate itself was not changed.
- **The R3 profile and the allocation benchmarks** ran after the timed cells, above the load gate (stated above). No timing from them is used for a verdict.
- **The first allocation-benchmark attempt failed to start.** The script that ran it passed each package and benchmark name as one argument (`stat …/internal/read BenchmarkWalkDeep131: directory not found`, exit 1 for each of the four). It was re-run straight after the timed cells with the names split; only the re-run is kept.
- **The first run-2 profile still used `--count`.** `BenchmarkR3Repo` was not fixed with the driver and passed `Count: true`. It was fixed to the served read (`read.Options{Numbers: true, NextLine: true}`, as the CLI's `mrw read --grep P .` passes) and the profile re-taken; only the re-take is kept. Its shares match the first attempt's within 3 points.

## Run 1 — invalid for a verdict, kept as it was taken

Run 1 ran 2026-10-29 09:58:42Z – 10:02:45Z with the driver at `e36be25f` (`main.go` sha256 `2d61ade7bec70e8e0ec178022ff18719885d56cea481c61673f7dfbfbfe149e1`), the same A and B binaries, every cell below the gate (1.80–1.97). The review of #565 found that it did not measure what the pre-registration registers:

1. R3 "repo" ran `mrw read --grep P --count .`, not the registered served read.
2. The 15-minute budget was checked only between cells, and cells did not run largest-last.
3. A process failing to start would have lost every sample collected before it: samples were written once, at the end.
4. Only the load before each cell was read, not after.
5. L1's timed ledgers held E+1 entries, since the first warm-up added `f.txt`, so the zero-entry case was never timed.

BenchmarkW1 also counted its fixture setup. Its raw data is in [raw/run1/](raw/run1/). Its R3 "repo" p50 was 100.62 ms (A) and 100.63 ms (B) with `--count`; its largest growth ratio was W1 (50,10)→(500,50) at 0.21.

| workload | size | binary | n | failed | p25 ms | p50 ms | p75 ms | p90 ms | max ms | cpu p50 ms | max rss MB | stdout B |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| R1 | 100 | A | 30 | 0 | 9.75 | 9.96 | 10.03 | 10.06 | 10.97 | 4.35 | 14.3 | 1266 |
| R1 | 100 | B | 30 | 0 | 9.04 | 9.95 | 10.04 | 10.13 | 10.21 | 4.33 | 13.8 | 1266 |
| R1 | 10000 | A | 30 | 0 | 9.92 | 10.01 | 10.09 | 10.14 | 13.75 | 4.60 | 15.2 | 1275 |
| R1 | 10000 | B | 30 | 0 | 9.88 | 9.93 | 10.03 | 10.07 | 13.44 | 4.57 | 14.7 | 1275 |
| R1 | 1000000 | A | 30 | 0 | 32.03 | 32.90 | 33.07 | 33.23 | 34.07 | 29.37 | 106.8 | 1330 |
| R1 | 1000000 | B | 30 | 0 | 32.77 | 32.87 | 32.93 | 33.05 | 33.23 | 29.66 | 107.1 | 1330 |
| R2 | 1 | A | 30 | 0 | 9.06 | 9.71 | 10.01 | 10.05 | 11.88 | 4.33 | 14.6 | 1278 |
| R2 | 1 | B | 30 | 0 | 9.02 | 9.84 | 9.98 | 10.37 | 15.04 | 4.36 | 14.1 | 1278 |
| R2 | 10 | A | 30 | 0 | 9.95 | 10.09 | 10.97 | 11.03 | 11.29 | 5.22 | 15.5 | 12782 |
| R2 | 10 | B | 30 | 0 | 9.94 | 10.08 | 10.90 | 11.06 | 11.83 | 5.23 | 15.1 | 12782 |
| R2 | 100 | A | 30 | 0 | 17.91 | 18.01 | 18.14 | 18.97 | 20.26 | 13.31 | 20.5 | 127888 |
| R2 | 100 | B | 30 | 0 | 17.85 | 17.96 | 18.05 | 18.18 | 20.97 | 13.32 | 20.0 | 127888 |
| R3 | repo | A | 30 | 0 | 99.89 | 100.62 | 101.97 | 104.21 | 153.18 | 371.14 | 26.2 | 638 |
| R3 | repo | B | 30 | 0 | 99.88 | 100.63 | 102.14 | 103.65 | 150.25 | 372.69 | 25.8 | 638 |
| R3 | deep3000 | A | 30 | 30 | 49.71 | 50.00 | 50.23 | 51.15 | 52.01 | 280.28 | 22.0 | 0 |
| R3 | deep3000 | B | 30 | 30 | 49.98 | 50.18 | 50.38 | 50.50 | 55.66 | 283.07 | 21.5 | 0 |
| W1 | 1h/1f | A | 30 | 0 | 11.70 | 11.86 | 11.99 | 12.13 | 14.79 | 6.02 | 15.0 | 133 |
| W1 | 1h/1f | B | 30 | 0 | 11.64 | 11.82 | 11.86 | 11.90 | 13.77 | 6.04 | 14.6 | 133 |
| W1 | 50h/10f | A | 30 | 0 | 14.93 | 15.93 | 16.45 | 16.89 | 19.05 | 9.80 | 18.9 | 2139 |
| W1 | 50h/10f | B | 30 | 0 | 15.02 | 15.86 | 16.03 | 16.69 | 17.70 | 9.64 | 19.1 | 2139 |
| W1 | 500h/50f | A | 30 | 0 | 31.47 | 32.23 | 33.21 | 34.24 | 36.34 | 30.70 | 23.3 | 18710 |
| W1 | 500h/50f | B | 30 | 0 | 31.35 | 33.09 | 33.94 | 35.13 | 41.14 | 30.71 | 23.1 | 18710 |
| W2 | 10 | A | 30 | 0 | 15.91 | 16.03 | 16.90 | 17.08 | 21.26 | 7.54 | 16.6 | 387 |
| W2 | 10 | B | 30 | 0 | 15.88 | 16.70 | 16.98 | 17.92 | 18.37 | 7.48 | 16.0 | 387 |
| W2 | 1000 | A | 30 | 0 | 22.88 | 23.11 | 23.72 | 24.19 | 26.09 | 15.88 | 23.0 | 30993 |
| W2 | 1000 | B | 30 | 0 | 22.72 | 22.85 | 23.33 | 24.53 | 28.62 | 15.79 | 22.5 | 30993 |
| L1 | 0 | A | 30 | 0 | 9.91 | 10.00 | 10.07 | 10.13 | 10.34 | 4.61 | 15.2 | 1276 |
| L1 | 0 | B | 30 | 0 | 9.91 | 9.96 | 10.04 | 10.16 | 11.82 | 4.63 | 14.7 | 1276 |
| L1 | 1000 | A | 30 | 0 | 9.91 | 10.00 | 10.09 | 10.93 | 11.22 | 5.55 | 18.1 | 1273 |
| L1 | 1000 | B | 30 | 0 | 9.97 | 10.04 | 10.11 | 10.74 | 11.76 | 5.51 | 17.6 | 1273 |
| L1 | 10000 | A | 30 | 0 | 13.92 | 14.11 | 14.89 | 15.04 | 18.41 | 11.03 | 24.6 | 1277 |
| L1 | 10000 | B | 30 | 0 | 13.92 | 14.02 | 14.24 | 14.92 | 16.01 | 10.80 | 24.4 | 1277 |
| C1 | first | A | 30 | 0 | 9.66 | 10.43 | 10.56 | 10.63 | 11.63 | 4.82 | 15.1 | 1278 |
| C1 | first | B | 30 | 0 | 9.68 | 9.83 | 10.63 | 10.77 | 13.30 | 4.76 | 14.6 | 1278 |
| C1 | warm | A | 30 | 0 | 9.95 | 10.04 | 10.12 | 10.15 | 11.16 | 4.62 | 15.4 | 1278 |
| C1 | warm | B | 30 | 0 | 9.88 | 9.94 | 10.01 | 10.05 | 10.17 | 4.60 | 14.7 | 1278 |
